package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/grutapig/hackaton/twitterapi"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const CHAT_IDS_STORAGE_PATH = "users.txt"

type TelegramService struct {
	apiKey        string
	client        *http.Client
	chatIDs       map[int64]bool
	chatMutex     sync.RWMutex
	lastOffset    int64
	isRunning     bool
	notifications map[string]FUDAlertNotification
	notifMutex    sync.RWMutex
	formatter     *NotificationFormatter
	dbService     *DatabaseService
	// Services for manual analysis
	twitterApi             interface{}                // Will be set later
	claudeApi              interface{}                // Will be set later
	userStatusManager      interface{}                // Will be set later
	systemPromptSecondStep []byte                     // Will be set later
	ticker                 string                     // Will be set later
	analysisChannel        chan twitterapi.NewMessage // Channel for manual analysis requests
}

type TelegramUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  struct {
		MessageID int64 `json:"message_id"`
		From      struct {
			ID        int64  `json:"id"`
			IsBot     bool   `json:"is_bot"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name,omitempty"`
			Username  string `json:"username,omitempty"`
		} `json:"from"`
		Chat struct {
			ID    int64  `json:"id"`
			Type  string `json:"type"`
			Title string `json:"title,omitempty"`
		} `json:"chat"`
		Date int64  `json:"date"`
		Text string `json:"text"`
	} `json:"message"`
}

type TelegramResponse struct {
	OK     bool             `json:"ok"`
	Result []TelegramUpdate `json:"result"`
	Error  *TelegramError   `json:"error,omitempty"`
}

type TelegramError struct {
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

type TelegramSendMessageRequest struct {
	ChatID         int64  `json:"chat_id"`
	Text           string `json:"text"`
	ParseMode      string `json:"parse_mode,omitempty"`
	DisablePreview bool   `json:"disable_web_page_preview,omitempty"`
}

type TelegramSendDocumentRequest struct {
	ChatID    int64  `json:"chat_id"`
	Caption   string `json:"caption,omitempty"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type TelegramEditMessageRequest struct {
	ChatID         int64  `json:"chat_id"`
	MessageID      int64  `json:"message_id"`
	Text           string `json:"text"`
	ParseMode      string `json:"parse_mode,omitempty"`
	DisablePreview bool   `json:"disable_web_page_preview,omitempty"`
}

type TelegramSendMessageResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		MessageID int64 `json:"message_id"`
		Chat      struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"result"`
}

func NewTelegramService(apiKey string, proxyDSN string, initialChatIDs string, formatter *NotificationFormatter, dbService *DatabaseService, analysisChannel chan twitterapi.NewMessage) (*TelegramService, error) {
	transport := &http.Transport{}
	if proxyDSN != "" {
		proxyURL, err := url.Parse(proxyDSN)
		if err != nil {
			return nil, fmt.Errorf("telegram service proxy dsn error: %s", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	service := &TelegramService{
		apiKey:          apiKey,
		client:          client,
		chatIDs:         make(map[int64]bool),
		lastOffset:      0,
		isRunning:       false,
		notifications:   make(map[string]FUDAlertNotification),
		formatter:       formatter,
		dbService:       dbService,
		analysisChannel: analysisChannel,
	}
	//Init chatIds from file if exists
	data, err := os.ReadFile(CHAT_IDS_STORAGE_PATH)
	if err == nil {
		chatIDStrings := strings.Split(string(data), "\n")
		for _, chatIDStr := range chatIDStrings {
			chatIDStr = strings.TrimSpace(chatIDStr) // Remove spaces
			if chatIDStr != "" {
				if chatID, err := strconv.ParseInt(chatIDStr, 10, 64); err == nil {
					service.chatIDs[chatID] = true
					log.Printf("Added initial Telegram chat ID: %d", chatID)
				} else {
					log.Printf("Warning: Invalid chat ID format: %s", chatIDStr)
				}
			}
		}
	}
	// Add initial chat IDs if provided (comma-separated)
	if initialChatIDs != "" {
		chatIDStrings := strings.Split(initialChatIDs, ",")
		for _, chatIDStr := range chatIDStrings {
			chatIDStr = strings.TrimSpace(chatIDStr) // Remove spaces
			if chatIDStr != "" {
				if chatID, err := strconv.ParseInt(chatIDStr, 10, 64); err == nil {
					service.chatIDs[chatID] = true
					log.Printf("Added initial Telegram chat ID: %d", chatID)
				} else {
					log.Printf("Warning: Invalid chat ID format: %s", chatIDStr)
				}
			}
		}
	}
	//back users list every 5 seconds into file
	go func() {
		for {
			time.Sleep(5 * time.Second)
			chatList := []string{}
			for chatId, _ := range service.chatIDs {
				chatList = append(chatList, strconv.Itoa(int(chatId)))
			}
			err = os.WriteFile(CHAT_IDS_STORAGE_PATH, []byte(strings.Join(chatList, "\n")), 0655)
			if err != nil {
				log.Println("cannot write file with notification users list.", err)
			}
		}
	}()

	return service, nil
}

// SetAnalysisServices sets the services needed for manual analysis
func (t *TelegramService) SetAnalysisServices(twitterApi interface{}, claudeApi interface{}, userStatusManager interface{}, systemPromptSecondStep []byte, ticker string) {
	t.twitterApi = twitterApi
	t.claudeApi = claudeApi
	t.userStatusManager = userStatusManager
	t.systemPromptSecondStep = systemPromptSecondStep
	t.ticker = ticker
}

func (t *TelegramService) StartListening() {
	if t.isRunning {
		return
	}
	t.isRunning = true

	go func() {
		for t.isRunning {
			err := t.processUpdates()
			if err != nil {
				log.Printf("Error processing Telegram updates: %v", err)
			}
			time.Sleep(2 * time.Second)
		}
	}()

	log.Println("Telegram service started listening for updates")
}

func (t *TelegramService) StopListening() {
	t.isRunning = false
	log.Println("Telegram service stopped listening")
}

func (t *TelegramService) isAdminChat(chatID int64) bool {
	adminChatsEnv := os.Getenv(ENV_TELEGRAM_ADMIN_CHAT_ID)
	if adminChatsEnv == "" {
		return false
	}

	adminChatIDs := strings.Split(adminChatsEnv, ",")
	chatIDStr := strconv.FormatInt(chatID, 10)

	for _, adminChatID := range adminChatIDs {
		if strings.TrimSpace(adminChatID) == chatIDStr {
			return true
		}
	}
	return false
}

func (t *TelegramService) processUpdates() error {
	updates, err := t.getUpdates()
	if err != nil {
		return err
	}

	for _, update := range updates {
		t.lastOffset = update.UpdateID + 1

		// Add new chat ID if not exists
		chatID := update.Message.Chat.ID
		t.chatMutex.Lock()
		if !t.chatIDs[chatID] {
			t.chatIDs[chatID] = true
			log.Printf("New Telegram chat registered: %d (from: %s)", chatID, update.Message.From.FirstName)

			// Send chat info as response
			info := fmt.Sprintf("✅ Chat registered!\nChat ID: %d\nUser: %s %s\nUsername: @%s",
				chatID,
				update.Message.From.FirstName,
				update.Message.From.LastName,
				update.Message.From.Username)

			go t.SendMessage(chatID, info)
		}
		t.chatMutex.Unlock()

		// Handle commands and messages
		if update.Message.Text != "" {
			text := strings.TrimSpace(update.Message.Text)

			// Parse command and arguments
			parts := strings.Fields(text)
			if len(parts) == 0 {
				return nil
			}

			command := parts[0]
			args := parts[1:]

			switch {
			case strings.HasPrefix(command, "/detail_"):
				go t.handleDetailCommand(chatID, text)
			case strings.HasPrefix(command, "/history_"):
				go t.handleHistoryCommand(chatID, text)
			case strings.HasPrefix(command, "/export_"):
				go t.handleExportCommand(chatID, text)
			case strings.HasPrefix(command, "/ticker_history_"):
				go t.handleTickerHistoryCommand(chatID, text)
			case strings.HasPrefix(command, "/cache_"):
				go t.handleCacheCommand(chatID, text)
			case command == "/analyze_all":
				if !t.isAdminChat(chatID) {
					go t.SendMessage(chatID, "❌ Access denied. This command is restricted to administrators only.")
					continue
				}
				go t.handleAnalyzeAllCommand(chatID)
				continue
			case strings.HasPrefix(command, "/analyze_"):
				go t.handleAnalyzeCommand(chatID, text)
			case command == "/search":
				go t.handleSearchCommand(chatID, args)
			case command == "/fudlist" || strings.HasPrefix(command, "/fudlist_"):
				go t.handleFudListCommand(chatID, args, command)
			case command == "/exportfudlist":
				go t.handleExportFudListCommand(chatID)
			case command == "/topfud" || strings.HasPrefix(command, "/topfud_"):
				go t.handleTopFudCommand(chatID, args, command)
			case command == "/tasks":
				go t.handleTasksCommand(chatID)
			case command == "/u":
				t.SendMessage(chatID, fmt.Sprintf("users: %d", len(t.chatIDs)))
			case command == "/top20_analyze":
				if !t.isAdminChat(chatID) {
					go t.SendMessage(chatID, "❌ Access denied. This command is restricted to administrators only.")
					continue
				}
				go t.handleTop20AnalyzeCommand(chatID)
			case command == "/top100_analyze":
				if !t.isAdminChat(chatID) {
					go t.SendMessage(chatID, "❌ Access denied. This command is restricted to administrators only.")
					continue
				}
				go t.handleTop100AnalyzeCommand(chatID)
			case command == "/batch_analyze":
				go t.handleBatchAnalyzeCommand(chatID, args)
			case command == "/help" || command == "/start":
				go t.handleHelpCommand(chatID)
			default:
				go t.handleHelpCommand(chatID)
			}
		}
	}

	return nil
}

func (t *TelegramService) getUpdates() ([]TelegramUpdate, error) {
	uri := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=1", t.apiKey, t.lastOffset)

	resp, err := t.client.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var telegramResp TelegramResponse
	err = json.Unmarshal(body, &telegramResp)
	if err != nil {
		return nil, err
	}

	if !telegramResp.OK {
		return nil, fmt.Errorf("telegram API error: %v", telegramResp.Error)
	}

	return telegramResp.Result, nil
}

func (t *TelegramService) SendMessage(chatID int64, text string) error {
	reqBody := TelegramSendMessageRequest{
		ChatID:         chatID,
		Text:           text,
		ParseMode:      "HTML",
		DisablePreview: true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.apiKey)
	resp, err := t.client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram send message failed: %s", string(body))
	}

	return nil
}

func (t *TelegramService) SendMessageWithID(chatID int64, text string) (int64, error) {
	reqBody := TelegramSendMessageRequest{
		ChatID:         chatID,
		Text:           text,
		ParseMode:      "HTML",
		DisablePreview: true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return 0, err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.apiKey)
	resp, err := t.client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("telegram send message failed: %s", string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var response TelegramSendMessageResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return 0, err
	}

	return response.Result.MessageID, nil
}

func (t *TelegramService) EditMessage(chatID int64, messageID int64, text string) error {
	reqBody := TelegramEditMessageRequest{
		ChatID:         chatID,
		MessageID:      messageID,
		Text:           text,
		ParseMode:      "HTML",
		DisablePreview: true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", t.apiKey)
	resp, err := t.client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram edit message failed: %s", string(body))
	}

	return nil
}

func (t *TelegramService) SendMessageWithResponse(chatID int64, text string) (*TelegramSendMessageResponse, error) {
	reqBody := TelegramSendMessageRequest{
		ChatID:         chatID,
		Text:           text,
		ParseMode:      "HTML",
		DisablePreview: true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.apiKey)
	resp, err := t.client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("telegram send message failed: %s", string(body))
	}

	var response TelegramSendMessageResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (t *TelegramService) generateTaskID() (string, error) {
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (t *TelegramService) SendDocument(chatID int64, filePath string, caption string) error {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add chat_id field
	err = writer.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	if err != nil {
		return err
	}

	// Add caption field if provided
	if caption != "" {
		err = writer.WriteField("caption", caption)
		if err != nil {
			return err
		}
		err = writer.WriteField("parse_mode", "HTML")
		if err != nil {
			return err
		}
	}

	// Add file field
	part, err := writer.CreateFormFile("document", filepath.Base(filePath))
	if err != nil {
		return err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	// Send request
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", t.apiKey)
	resp, err := t.client.Post(url, writer.FormDataContentType(), &requestBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram send document failed: %s", string(body))
	}

	return nil
}

func (t *TelegramService) BroadcastMessage(text string) error {
	t.chatMutex.RLock()
	defer t.chatMutex.RUnlock()

	if len(t.chatIDs) == 0 {
		log.Println("No registered Telegram chats to broadcast to")
		return nil
	}

	var errors []error
	for chatID := range t.chatIDs {
		err := t.SendMessage(chatID, text)
		if err != nil {
			log.Printf("Failed to send message to chat %d: %v", chatID, err)
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send to %d chats", len(errors))
	}

	log.Printf("Successfully broadcasted message to %d chats", len(t.chatIDs))
	return nil
}

func (t *TelegramService) GetRegisteredChats() []int64 {
	t.chatMutex.RLock()
	defer t.chatMutex.RUnlock()

	var chats []int64
	for chatID := range t.chatIDs {
		chats = append(chats, chatID)
	}
	return chats
}

func (t *TelegramService) generateNotificationID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (t *TelegramService) StoreAndBroadcastNotification(alert FUDAlertNotification) error {
	// Generate unique ID and store notification
	notificationID := t.generateNotificationID()

	t.notifMutex.Lock()
	t.notifications[notificationID] = alert
	t.notifMutex.Unlock()

	// Format message with detail command
	telegramMessage := t.formatter.FormatForTelegramWithDetail(alert, notificationID)

	// Broadcast to all chats
	return t.BroadcastMessage(telegramMessage)
}

func (t *TelegramService) handleDetailCommand(chatID int64, command string) {
	// Extract notification ID from command "/detail_12345abc"
	prefix := "/detail_"
	if !strings.HasPrefix(command, prefix) {
		t.SendMessage(chatID, "❌ Invalid command format. Use /detail_<id>")
		return
	}

	notificationID := strings.TrimPrefix(command, prefix)

	t.notifMutex.RLock()
	alert, exists := t.notifications[notificationID]
	t.notifMutex.RUnlock()

	if !exists {
		t.SendMessage(chatID, "❌ Notification not found or expired.")
		return
	}

	// Send detailed information
	detailMessage := t.formatter.FormatDetailedView(alert)
	t.SendMessage(chatID, detailMessage)
}

func (t *TelegramService) handleHistoryCommand(chatID int64, command string) {
	// Extract username from command "/history_username"
	prefix := "/history_"
	if !strings.HasPrefix(command, prefix) {
		t.SendMessage(chatID, "❌ Invalid command format. Use /history_username")
		return
	}

	username := strings.TrimPrefix(command, prefix)

	// Get 20 latest messages for the user
	tweets, err := t.dbService.GetUserMessagesByUsername(username, 20)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error retrieving messages for @%s: %v", username, err))
		return
	}

	if len(tweets) == 0 {
		t.SendMessage(chatID, fmt.Sprintf("📭 No messages found for @%s", username))
		return
	}

	// Format the message history
	var historyMessage strings.Builder
	historyMessage.WriteString(fmt.Sprintf("📝 <b>Message History for @%s</b> (Last 20)\n\n", username))

	for i, tweet := range tweets {
		historyMessage.WriteString(fmt.Sprintf("<b>%d.</b> %s\n", i+1, tweet.CreatedAt.Format("2006-01-02 15:04")))
		historyMessage.WriteString(fmt.Sprintf("📝 <i>%s</i>\n", t.truncateText(tweet.Text, 200)))
		if tweet.InReplyToID != "" {
			historyMessage.WriteString("↳ <i>Reply to tweet</i>\n")
		}
		historyMessage.WriteString(fmt.Sprintf("🆔 <code>%s</code>\n\n", tweet.ID))
	}

	// Add command for full export
	historyMessage.WriteString(fmt.Sprintf("📄 For full message history: /export_%s", username))

	t.SendMessage(chatID, historyMessage.String())
}

func (t *TelegramService) handleTickerHistoryCommand(chatID int64, command string) {
	// Extract username from command "/ticker_history_username"
	prefix := "/ticker_history_"
	if !strings.HasPrefix(command, prefix) {
		t.SendMessage(chatID, "❌ Invalid command format. Use /ticker_history_username")
		return
	}

	username := strings.TrimPrefix(command, prefix)
	ticker := t.ticker // Use the ticker from the environment

	// Get ALL ticker-related messages for the user (no limit for checking count)
	allOpinions, err := t.dbService.GetUserTickerOpinionsByUsername(username, ticker, 0)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error retrieving ticker history for @%s: %v", username, err))
		return
	}

	if len(allOpinions) == 0 {
		t.SendMessage(chatID, fmt.Sprintf("📭 No ticker-related messages found for @%s and %s", username, ticker))
		return
	}

	// If more than 15 items, export as file
	if len(allOpinions) > 15 {
		t.SendMessage(chatID, fmt.Sprintf("📊 Found %d ticker mentions for @%s (%s). Generating file...", len(allOpinions), username, ticker))
		t.exportTickerHistoryAsFile(chatID, username, ticker, allOpinions)
		return
	}

	// Format the ticker history message for small datasets
	var historyMessage strings.Builder
	historyMessage.WriteString(fmt.Sprintf("💰 <b>Ticker History for @%s (%s)</b> (%d messages)\n\n", username, ticker, len(allOpinions)))

	for i, opinion := range allOpinions {
		historyMessage.WriteString(fmt.Sprintf("<b>%d.</b> %s\n", i+1, opinion.TweetCreatedAt.Format("2006-01-02 15:04")))
		historyMessage.WriteString(fmt.Sprintf("💬 <i>%s</i>\n", t.truncateText(opinion.Text, 200)))

		// Show reply context if available
		if opinion.InReplyToID != "" && opinion.RepliedToAuthor != "" {
			historyMessage.WriteString(fmt.Sprintf("↳ <i>Reply to @%s: %s</i>\n",
				opinion.RepliedToAuthor,
				t.truncateText(opinion.RepliedToText, 100)))
		}

		historyMessage.WriteString(fmt.Sprintf("🆔 <code>%s</code>\n", opinion.TweetID))
		historyMessage.WriteString(fmt.Sprintf("🔍 <i>Search: %s</i>\n\n", opinion.SearchQuery))
	}

	// Add summary
	historyMessage.WriteString(fmt.Sprintf("📊 Total ticker mentions: %d\n", len(allOpinions)))
	historyMessage.WriteString(fmt.Sprintf("📄 For full message history: /export_%s", username))

	t.SendMessage(chatID, historyMessage.String())
}

func (t *TelegramService) handleCacheCommand(chatID int64, command string) {
	// Extract user identifier from command "/cache_username_or_id"
	prefix := "/cache_"
	if !strings.HasPrefix(command, prefix) {
		t.SendMessage(chatID, "❌ Invalid command format. Use /cache_<username_or_id>")
		return
	}

	userIdentifier := strings.TrimPrefix(command, prefix)
	if userIdentifier == "" {
		t.SendMessage(chatID, "❌ Please provide username or user ID. Use /cache_<username_or_id>")
		return
	}

	// Try to find user by username first (case insensitive), then by ID
	var user *UserModel
	var err error

	if user, err = t.dbService.GetUserByUsername(userIdentifier); err != nil {
		// If not found by username, try by ID
		if user, err = t.dbService.GetUser(userIdentifier); err != nil {
			t.SendMessage(chatID, fmt.Sprintf("❌ User not found: %s\nTried both username and ID lookup.", userIdentifier))
			return
		}
	}

	// Get cached analysis for the user
	cachedAnalysis, err := t.dbService.GetCachedAnalysis(user.ID)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("💾 <b>No Cached Analysis Found</b>\n\n👤 User: @%s (ID: %s)\n❌ No cached analysis available or cache has expired.", user.Username, user.ID))
		return
	}

	// Format cached analysis information
	var message strings.Builder
	message.WriteString(fmt.Sprintf("💾 <b>Cached Analysis for @%s</b>\n\n", user.Username))

	// User information
	message.WriteString(fmt.Sprintf("👤 <b>User Details:</b>\n"))
	message.WriteString(fmt.Sprintf("• Username: @%s\n", user.Username))
	message.WriteString(fmt.Sprintf("• Name: %s\n", user.Name))
	message.WriteString(fmt.Sprintf("• https://x.com/%s\n", user.Username))
	message.WriteString(fmt.Sprintf("• User ID: <code>%s</code>\n\n", user.ID))

	// Analysis results
	message.WriteString(fmt.Sprintf("🔍 <b>Analysis Results:</b>\n"))

	statusEmoji := "✅"
	statusText := "Clean User"
	if cachedAnalysis.IsFUDUser {
		statusEmoji = "🚨"
		statusText = "FUD User Detected"
	}

	message.WriteString(fmt.Sprintf("• %s Status: <b>%s</b>\n", statusEmoji, statusText))
	message.WriteString(fmt.Sprintf("• 🎯 FUD Type: %s\n", cachedAnalysis.FUDType))
	message.WriteString(fmt.Sprintf("• 📊 Confidence: %.1f%%\n", cachedAnalysis.FUDProbability*100))
	message.WriteString(fmt.Sprintf("• ⚡ Risk Level: %s\n", strings.ToUpper(cachedAnalysis.UserRiskLevel)))

	if cachedAnalysis.UserSummary != "" {
		message.WriteString(fmt.Sprintf("• 👤 Profile: %s\n", cachedAnalysis.UserSummary))
	}

	message.WriteString("\n")

	// Key evidence
	if len(cachedAnalysis.KeyEvidence) > 0 {
		message.WriteString("🔍 <b>Key Evidence:</b>\n")
		for i, evidence := range cachedAnalysis.KeyEvidence {
			message.WriteString(fmt.Sprintf("%d. %s\n", i+1, evidence))
		}
		message.WriteString("\n")
	}

	// Decision reasoning
	if cachedAnalysis.DecisionReason != "" {
		message.WriteString(fmt.Sprintf("🧠 <b>Decision Reasoning:</b>\n<i>%s</i>\n\n", cachedAnalysis.DecisionReason))
	}

	// Cache metadata - get cache record for metadata
	var cacheRecord CachedAnalysisModel
	err = t.dbService.db.Where("user_id = ?", user.ID).First(&cacheRecord).Error
	if err == nil {
		message.WriteString("📅 <b>Cache Information:</b>\n")
		message.WriteString(fmt.Sprintf("• 🕐 Analyzed At: %s\n", cacheRecord.AnalyzedAt.Format("2006-01-02 15:04:05 UTC")))
		message.WriteString(fmt.Sprintf("• ⏰ Expires At: %s\n", cacheRecord.ExpiresAt.Format("2006-01-02 15:04:05 UTC")))

		// Calculate time remaining
		timeRemaining := time.Until(cacheRecord.ExpiresAt)
		if timeRemaining > 0 {
			hours := int(timeRemaining.Hours())
			minutes := int(timeRemaining.Minutes()) % 60
			message.WriteString(fmt.Sprintf("• ⏳ Valid for: %dh %dm\n", hours, minutes))
		} else {
			message.WriteString("• ⏳ Status: <b>Expired</b>\n")
		}
		message.WriteString("\n")
	}

	// Related commands
	message.WriteString("🔍 <b>Related Commands:</b>\n")
	message.WriteString(fmt.Sprintf("• /history_%s - Message history\n", user.Username))
	message.WriteString(fmt.Sprintf("• /ticker_history_%s - Ticker posts\n", user.Username))
	message.WriteString(fmt.Sprintf("• /export_%s - Full export\n", user.Username))
	message.WriteString(fmt.Sprintf("• /analyze_%s - Force new analysis\n", user.Username))

	t.SendMessage(chatID, message.String())
}

func (t *TelegramService) exportTickerHistoryAsFile(chatID int64, username, ticker string, opinions []UserTickerOpinionModel) {
	// Build file content
	var fileContent strings.Builder
	fileContent.WriteString(fmt.Sprintf("TICKER HISTORY EXPORT\n"))
	fileContent.WriteString(fmt.Sprintf("User: @%s\n", username))
	fileContent.WriteString(fmt.Sprintf("Ticker: %s\n", ticker))
	fileContent.WriteString(fmt.Sprintf("Total Messages: %d\n", len(opinions)))
	fileContent.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05 UTC")))
	fileContent.WriteString(strings.Repeat("=", 60) + "\n\n")

	for i, opinion := range opinions {
		fileContent.WriteString(fmt.Sprintf("[%d] %s\n", i+1, opinion.TweetCreatedAt.Format("2006-01-02 15:04:05 UTC")))
		fileContent.WriteString(fmt.Sprintf("Tweet ID: %s\n", opinion.TweetID))

		// Add reply context if available
		if opinion.InReplyToID != "" {
			fileContent.WriteString(fmt.Sprintf("Reply to: %s\n", opinion.InReplyToID))
			if opinion.RepliedToAuthor != "" {
				fileContent.WriteString(fmt.Sprintf("Reply to @%s: %s\n", opinion.RepliedToAuthor, opinion.RepliedToText))
			}
		}

		fileContent.WriteString(fmt.Sprintf("Search Query: %s\n", opinion.SearchQuery))
		fileContent.WriteString(fmt.Sprintf("Found At: %s\n", opinion.FoundAt.Format("2006-01-02 15:04:05 UTC")))
		fileContent.WriteString("Message:\n")
		fileContent.WriteString(opinion.Text)
		fileContent.WriteString("\n" + strings.Repeat("-", 40) + "\n\n")
	}

	// Write to file
	filename := fmt.Sprintf("%s_ticker_%s_%s.txt", username, ticker, time.Now().Format("20060102_150405"))
	err := t.writeToFile(filename, fileContent.String())
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error creating file: %v", err))
		return
	}

	// Send file to Telegram
	caption := fmt.Sprintf("💰 <b>Ticker History Export</b>\n\n👤 User: @%s\n🏷️ Ticker: %s\n📊 Total Messages: %d\n📅 Generated: %s",
		username,
		ticker,
		len(opinions),
		time.Now().Format("2006-01-02 15:04:05"))

	err = t.SendDocument(chatID, filename, caption)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error sending file: %v\nFile created locally: %s", err, filename))
		return
	}

	// Clean up local file after successful send
	go func() {
		time.Sleep(10 * time.Second) // Wait a bit before cleanup
		os.Remove(filename)
	}()

	// Send confirmation message
	t.SendMessage(chatID, "✅ Ticker history file sent successfully!")
}

func (t *TelegramService) handleExportCommand(chatID int64, command string) {
	// Extract username from command "/export_username"
	prefix := "/export_"
	if !strings.HasPrefix(command, prefix) {
		t.SendMessage(chatID, "❌ Invalid command format. Use /export_username")
		return
	}

	username := strings.TrimPrefix(command, prefix)

	// Get all messages for the user
	tweets, err := t.dbService.GetAllUserMessagesByUsername(username)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error retrieving messages for @%s: %v", username, err))
		return
	}

	if len(tweets) == 0 {
		t.SendMessage(chatID, fmt.Sprintf("📭 No messages found for @%s", username))
		return
	}

	// Create text file content
	var fileContent strings.Builder
	fileContent.WriteString(fmt.Sprintf("FULL MESSAGE HISTORY FOR @%s\n", strings.ToUpper(username)))
	fileContent.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05 UTC")))
	fileContent.WriteString(fmt.Sprintf("Total Messages: %d\n", len(tweets)))
	fileContent.WriteString(strings.Repeat("=", 80) + "\n\n")

	for i, tweet := range tweets {
		fileContent.WriteString(fmt.Sprintf("[%d] %s\n", i+1, tweet.CreatedAt.Format("2006-01-02 15:04:05 UTC")))
		fileContent.WriteString(fmt.Sprintf("ID: %s\n", tweet.ID))
		if tweet.InReplyToID != "" {
			fileContent.WriteString(fmt.Sprintf("Reply to: %s\n", tweet.InReplyToID))
		}
		fileContent.WriteString(fmt.Sprintf("Source: %s\n", tweet.SourceType))
		if tweet.TickerMention != "" {
			fileContent.WriteString(fmt.Sprintf("Ticker: %s\n", tweet.TickerMention))
		}
		fileContent.WriteString("Message:\n")
		fileContent.WriteString(tweet.Text)
		fileContent.WriteString("\n" + strings.Repeat("-", 40) + "\n\n")
	}

	// Write to file
	filename := fmt.Sprintf("%s_messages_%s.txt", username, time.Now().Format("20060102_150405"))
	err = t.writeToFile(filename, fileContent.String())
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error creating file: %v", err))
		return
	}

	// Send file to Telegram
	caption := fmt.Sprintf("📄 <b>Full Message Export</b>\n\n👤 User: @%s\n📊 Total Messages: %d\n📅 Generated: %s",
		username,
		len(tweets),
		time.Now().Format("2006-01-02 15:04:05"))

	err = t.SendDocument(chatID, filename, caption)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error sending file: %v\nFile created locally: %s", err, filename))
		return
	}

	// Clean up local file after successful send
	go func() {
		time.Sleep(10 * time.Second) // Wait a bit before cleanup
		os.Remove(filename)
	}()

	// Send confirmation message
	t.SendMessage(chatID, "✅ Export file sent successfully!")
}

func (t *TelegramService) truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength-3] + "..."
}

func (t *TelegramService) writeToFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

func (t *TelegramService) handleSearchCommand(chatID int64, args []string) {
	var users []UserModel
	var err error
	var searchTitle string

	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		// No query provided - show top 10 most active users
		users, err = t.dbService.GetTopActiveUsers(10)
		searchTitle = "🔥 <b>Top 10 Most Active Users</b>"
	} else {
		// Search by query
		query := strings.Join(args, " ")
		users, err = t.dbService.SearchUsers(query, 20)
		searchTitle = fmt.Sprintf("🔍 <b>Search Results for '%s'</b> (Found %d)", query, len(users))
	}
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error searching users: %v", err))
		return
	}

	if len(users) == 0 {
		if len(args) == 0 {
			t.SendMessage(chatID, "📭 No active users found in database")
		} else {
			t.SendMessage(chatID, fmt.Sprintf("🔍 No users found matching '%s'", strings.Join(args, " ")))
		}
		return
	}

	// Format search results
	var searchResults strings.Builder
	searchResults.WriteString(searchTitle + "\n\n")

	for i, user := range users {
		fudStatus := ""
		if t.dbService.IsFUDUser(user.ID) {
			fudStatus = " 🚨 <b>FUD USER</b>"
		}

		analyzedStatus := ""
		if t.dbService.IsUserDetailAnalyzed(user.ID) {
			analyzedStatus = " ✅ Analyzed"
		}

		searchResults.WriteString(fmt.Sprintf("<b>%d.</b> @%s%s%s\n", i+1, user.Username, fudStatus, analyzedStatus))
		if user.Name != "" && user.Name != user.Username {
			searchResults.WriteString(fmt.Sprintf("    Name: %s\n", user.Name))
		}
		searchResults.WriteString(fmt.Sprintf("    ID: <code>%s</code>\n", user.ID))

		// Add quick action commands
		searchResults.WriteString(fmt.Sprintf("    Commands: /history_%s | /analyze_%s\n\n", user.Username, user.Username))
	}

	// Add note about commands
	searchResults.WriteString("💡 <b>Quick Actions:</b>\n• Tap /history_username to view recent messages\n• Tap /analyze_username to run second step analysis")

	t.SendMessage(chatID, searchResults.String())
}

func (t *TelegramService) handleAnalyzeCommand(chatID int64, command string) {
	prefix := "/analyze_"
	if !strings.HasPrefix(command, prefix) {
		t.SendMessage(chatID, "❌ Invalid command format. Use /cache_<username_or_id>")
		return
	}

	username := strings.TrimPrefix(command, prefix)
	if username == "" {
		t.SendMessage(chatID, "❌ Please provide username or user ID. Use /cache_<username_or_id>")
		return
	}

	// Generate unique task ID
	taskID := t.generateNotificationID()

	// Send initial progress message
	initialText := fmt.Sprintf("🔄 <b>Starting Analysis for @%s</b>\n\n📋 <b>Status:</b> Initializing...\n🆔 <b>Task ID:</b> <code>%s</code>\n\n⏳ Please wait, this may take a few minutes.", username, taskID)
	messageID, err := t.SendMessageWithID(chatID, initialText)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Failed to start analysis: %v", err))
		return
	}

	// Create analysis task in database
	task := &AnalysisTaskModel{
		ID:             taskID,
		Username:       username,
		Status:         ANALYSIS_STATUS_PENDING,
		CurrentStep:    ANALYSIS_STEP_INIT,
		ProgressText:   "Initializing analysis...",
		TelegramChatID: chatID,
		MessageID:      messageID,
		StartedAt:      time.Now(),
	}

	err = t.dbService.CreateAnalysisTask(task)
	if err != nil {
		t.EditMessage(chatID, messageID, fmt.Sprintf("❌ <b>Analysis Failed</b>\n\nFailed to create analysis task: %v", err))
		return
	}

	// Start analysis in goroutine
	go t.processAnalysisTask(taskID, chatID)

	// Start progress monitor
	go t.monitorAnalysisProgress(taskID)
}

func (t *TelegramService) handleHelpCommand(chatID int64) {
	helpMessage := `🤖 <b>FUD Detection Bot - Available Commands</b>

🔍 <b>Search & Analysis Commands:</b>
• /search - Search users by username/name
• /analyze_username - Run manual FUD analysis

📊 <b>User Investigation Commands:</b>
• /history_username - View recent messages (20 latest)
• /ticker_history_username - View ticker-related messages
• /cache_username - View cached analysis results
• /export_username - Export full message history as file
• /detail_id - View detailed FUD analysis

📊 <b>Analysis Management:</b>
• /fudlist - Show all detected FUD users
• /topfud - Show cached FUD users sorted by last message
• /exportfudlist - Export FUD usernames as comma-separated list
• /tasks - Show running analysis tasks
• /batch_analyze user1,user2,user3 - Analyze multiple users
• /top20_analyze - Analyze top 20 most active users (admin only)
• /analyze_all - Analyze ALL users with messages (admin only)

❓ <b>Help Commands:</b>
• /help - Show this help message
• /start - Show this help message

💡 <b>Usage Tips:</b>
• Commands with underscore (_) need exact format: /analyze_john
• Commands with space accept parameters: /search john
• All commands are case-sensitive
• Bot responds to FUD alerts automatically

🔔 <b>Alert Types:</b>
• 🚨🔥 Critical - Immediate action required
• 🚨 High - Monitor closely  
• ⚠️ Medium - Standard monitoring
• ℹ️ Low - Log and watch

👤 <b>Your Chat ID:</b> %d`

	t.SendMessage(chatID, fmt.Sprintf(helpMessage, chatID))
}

// processAnalysisTask processes the actual analysis work
func (t *TelegramService) processAnalysisTask(taskID string, chatID int64) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Analysis task %s panicked: %v", taskID, r)
			t.dbService.SetAnalysisTaskError(taskID, fmt.Sprintf("Internal error: %v", r))
		}
	}()

	// Get task details
	task, err := t.dbService.GetAnalysisTask(taskID)
	if err != nil {
		log.Printf("Failed to get analysis task %s: %v", taskID, err)
		return
	}

	username := task.Username

	// Step 1: User lookup
	t.dbService.UpdateAnalysisTaskProgress(taskID, ANALYSIS_STEP_USER_LOOKUP, "Looking up user information...")
	user, err := t.dbService.GetUserByUsername(username)
	var userID string
	if err != nil {
		userID = "unknown_" + username
		log.Printf("User %s not found in database, using placeholder ID", username)
	} else {
		userID = user.ID
		// Update task with found user ID
		task.UserID = userID
		t.dbService.UpdateAnalysisTask(task)
	}

	// Step 2: Get user tweet for analysis context
	t.dbService.UpdateAnalysisTaskProgress(taskID, ANALYSIS_STEP_TICKER_SEARCH, "Searching for user's ticker mentions...")
	tweet, err := t.dbService.GetUserTweetForAnalysis(username)

	var newMessage twitterapi.NewMessage

	if err != nil {
		log.Printf("No tweet found for %s, creating placeholder data", username)

		newMessage = twitterapi.NewMessage{
			TweetID:      "manual_analysis_" + username,
			ReplyTweetID: "",
			Author: struct {
				UserName string
				Name     string
				ID       string
			}{
				UserName: username,
				Name:     username,
				ID:       userID,
			},
			Text:      "Manual analysis request - no recent tweets found",
			CreatedAt: time.Now().Format(time.RFC3339),
			ParentTweet: struct {
				ID     string
				Author string
				Text   string
			}{
				ID:     "placeholder_parent",
				Author: "system",
				Text:   "No parent tweet available - manual analysis",
			},
			GrandParentTweet: struct {
				ID     string
				Author string
				Text   string
			}{
				ID:     "",
				Author: "",
				Text:   "",
			},
			IsManualAnalysis:  true,
			ForceNotification: true,
			TaskID:            taskID,
			TelegramChatID:    chatID,
		}
	} else {
		newMessage = twitterapi.NewMessage{
			TweetID:      tweet.ID,
			ReplyTweetID: tweet.InReplyToID,
			Author: struct {
				UserName string
				Name     string
				ID       string
			}{
				UserName: username,
				Name:     username,
				ID:       tweet.UserID,
			},
			Text:      tweet.Text,
			CreatedAt: tweet.CreatedAt.Format(time.RFC3339),
			ParentTweet: struct {
				ID     string
				Author string
				Text   string
			}{
				ID:     "manual_parent",
				Author: "system",
				Text:   "Manual analysis - limited context available",
			},
			GrandParentTweet: struct {
				ID     string
				Author string
				Text   string
			}{
				ID:     "",
				Author: "",
				Text:   "",
			},
			IsManualAnalysis:  true,
			ForceNotification: true,
			TaskID:            taskID,
			TelegramChatID:    chatID,
		}
	}

	// Step 3: Send to analysis channel
	t.dbService.UpdateAnalysisTaskProgress(taskID, ANALYSIS_STEP_CLAUDE_ANALYSIS, "Sending for FUD analysis...")

	select {
	case t.analysisChannel <- newMessage:
		// Successfully sent to analysis - now wait for neural network processing
		t.dbService.UpdateAnalysisTaskProgress(taskID, ANALYSIS_STEP_CLAUDE_ANALYSIS, "Processing with neural network...")

		// Task completion will be handled by SecondStepHandler after Claude analysis
		log.Printf("Manual analysis task %s sent to Claude processing pipeline", taskID)

	default:
		// Analysis channel is full
		t.dbService.SetAnalysisTaskError(taskID, "Analysis channel is full, please try again later")
	}
}

// monitorAnalysisProgress monitors task progress and updates Telegram message
func (t *TelegramService) monitorAnalysisProgress(taskID string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			task, err := t.dbService.GetAnalysisTask(taskID)
			if err != nil {
				log.Printf("Failed to get analysis task %s for monitoring: %v", taskID, err)
				return
			}

			// Update progress message
			progressText := t.formatAnalysisProgress(task)
			err = t.EditMessage(task.TelegramChatID, task.MessageID, progressText)
			if err != nil {
				log.Printf("Failed to update progress message for task %s: %v", taskID, err)
			}

			// Stop monitoring if task is completed or failed
			if task.Status == ANALYSIS_STATUS_COMPLETED || task.Status == ANALYSIS_STATUS_FAILED {
				return
			}
		}
	}
}

// formatAnalysisProgress formats the progress message for Telegram
func (t *TelegramService) formatAnalysisProgress(task *AnalysisTaskModel) string {
	if task.Status == ANALYSIS_STATUS_FAILED {
		return fmt.Sprintf(`❌ <b>Analysis Failed for @%s</b>

⚠️ <b>Error:</b> %s
🆔 <b>Task ID:</b> <code>%s</code>

🔄 You can try running the analysis again.`,
			task.Username,
			task.ErrorMessage,
			task.ID)
	}

	if task.Status == ANALYSIS_STATUS_COMPLETED {
		return fmt.Sprintf(`✅ <b>Analysis Completed for @%s</b>

📋 <b>Status:</b> Finished successfully
🔍 <b>Results:</b> Check FUD alerts for analysis results
🆔 <b>Task ID:</b> <code>%s</code>

✅ Analysis has been completed and results sent to notification system.`,
			task.Username,
			task.ID)
	}

	// Running status with progress steps
	stepEmoji := "🔄"
	stepText := task.ProgressText

	switch task.CurrentStep {
	case ANALYSIS_STEP_INIT:
		stepEmoji = "⚙️"
	case ANALYSIS_STEP_USER_LOOKUP:
		stepEmoji = "🔍"
	case ANALYSIS_STEP_TICKER_SEARCH:
		stepEmoji = "📊"
	case ANALYSIS_STEP_FOLLOWERS:
		stepEmoji = "👥"
	case ANALYSIS_STEP_FOLLOWINGS:
		stepEmoji = "👤"
	case ANALYSIS_STEP_COMMUNITY_ACTIVITY:
		stepEmoji = "🏠"
	case ANALYSIS_STEP_CLAUDE_ANALYSIS:
		stepEmoji = "🤖"
	case ANALYSIS_STEP_SAVING_RESULTS:
		stepEmoji = "💾"
	}

	// Calculate elapsed time
	elapsed := time.Since(task.StartedAt)
	elapsedStr := fmt.Sprintf("%.0fs", elapsed.Seconds())
	if elapsed.Minutes() >= 1 {
		elapsedStr = fmt.Sprintf("%.1fm", elapsed.Minutes())
	}

	return fmt.Sprintf(`🔄 <b>Analyzing @%s</b>

%s <b>Current Step:</b> %s
⏱️ <b>Running Time:</b> %s
🆔 <b>Task ID:</b> <code>%s</code>

⏳ Please wait, analysis in progress...`,
		task.Username,
		stepEmoji, stepText,
		elapsedStr,
		task.ID)
}

func (t *TelegramService) handleFudListCommand(chatID int64, args []string, command string) {
	// Parse page number from command or arguments
	page := 1

	// Check if page number is in command format /fudlist_X
	if strings.HasPrefix(command, "/fudlist_") {
		pageStr := strings.TrimPrefix(command, "/fudlist_")
		if pageNum, err := strconv.Atoi(pageStr); err == nil && pageNum > 0 {
			page = pageNum
		}
	} else if len(args) > 0 {
		// Fallback to old format with arguments
		if pageNum, err := strconv.Atoi(args[0]); err == nil && pageNum > 0 {
			page = pageNum
		}
	}

	const pageSize = 10 // Users per page

	fudUsers, err := t.dbService.GetAllFUDUsersFromCache()
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error retrieving FUD users: %v", err))
		return
	}

	if len(fudUsers) == 0 {
		t.SendMessage(chatID, "✅ <b>No FUD Users Detected</b>\n\n🎉 Great news! No FUD users have been detected in the system.")
		return
	}

	totalPages := (len(fudUsers) + pageSize - 1) / pageSize
	if page > totalPages {
		page = totalPages
	}

	startIdx := (page - 1) * pageSize
	endIdx := startIdx + pageSize
	if endIdx > len(fudUsers) {
		endIdx = len(fudUsers)
	}

	var message strings.Builder
	message.WriteString(fmt.Sprintf("🚨 <b>FUD Users (%d total) - Page %d/%d</b>\n\n", len(fudUsers), page, totalPages))

	activeFUD := 0
	cachedFUD := 0

	// Show users for current page
	for i := startIdx; i < endIdx; i++ {
		user := fudUsers[i]
		source := user["source"].(string)
		if source == "active" {
			activeFUD++
		} else {
			cachedFUD++
		}

		username := user["username"].(string)
		userID := user["user_id"].(string)
		fudType := user["fud_type"].(string)
		probability := user["fud_probability"].(float64)
		detectedAt := user["detected_at"].(time.Time)

		sourceEmoji := "🔥"
		if source == "cached" {
			sourceEmoji = "💾"
		}

		// Get last message info
		lastMessageDate := user["last_message_date"].(time.Time)
		isAlive := user["is_alive"].(bool)
		status := user["status"].(string)

		statusEmoji := "💀"
		if isAlive {
			statusEmoji = "🟢"
		}

		message.WriteString(fmt.Sprintf("<b>%d.</b> %s @%s (%s) %s %s\n", i+1, sourceEmoji, username, userID, statusEmoji, status))
		message.WriteString(fmt.Sprintf("    🎯 Type: %s (%.0f%%)\n", fudType, probability*100))
		message.WriteString(fmt.Sprintf("    📅 Detected: %s\n", detectedAt.Format("2006-01-02 15:04")))

		if !lastMessageDate.IsZero() {
			message.WriteString(fmt.Sprintf("    💬 Last msg: %s\n", lastMessageDate.Format("2006-01-02 15:04")))
		} else {
			message.WriteString("    💬 Last msg: unknown\n")
		}

		if userSummary, ok := user["user_summary"].(string); ok && userSummary != "" {
			message.WriteString(fmt.Sprintf("    👤 Profile: %s\n", userSummary))
		}

		// Add enhanced command links
		message.WriteString("    🔍 <b>Commands:</b>\n")
		message.WriteString(fmt.Sprintf("      /export_%s - Message history\n", username))
		message.WriteString(fmt.Sprintf("      /ticker_history_%s - Ticker posts\n", username))
		message.WriteString(fmt.Sprintf("      /cache_%s - detailed analysis\n", username))
		message.WriteString(fmt.Sprintf("      https://x.com/%s\n", username))
		message.WriteString("\n")
	}

	// Add pagination controls
	if totalPages > 1 {
		message.WriteString("📄 <b>Navigation:</b>\n")
		if page > 1 {
			message.WriteString(fmt.Sprintf("  ⬅️ /fudlist_%d (Previous)\n", page-1))
		}
		if page < totalPages {
			message.WriteString(fmt.Sprintf("  ➡️ /fudlist_%d (Next)\n", page+1))
		}
		message.WriteString("\n")
	}

	// Count all users for summary
	totalActiveFUD := 0
	totalCachedFUD := 0
	for _, user := range fudUsers {
		source := user["source"].(string)
		if source == "active" {
			totalActiveFUD++
		} else {
			totalCachedFUD++
		}
	}

	// Count alive and dead users
	aliveCount := 0
	deadCount := 0
	for _, user := range fudUsers {
		if user["is_alive"].(bool) {
			aliveCount++
		} else {
			deadCount++
		}
	}

	message.WriteString(fmt.Sprintf("📊 <b>Summary:</b>\n• 🔥 Active FUD users: %d\n• 💾 Cached detections: %d\n• 🟢 Alive users: %d\n• 💀 Dead users: %d\n\n", totalActiveFUD, totalCachedFUD, aliveCount, deadCount))
	message.WriteString("💡 <b>Legend:</b>\n• 🔥 Active (persistent in database)\n• 💾 Cached (expires in 24h)\n• 🟢 Alive (active within 30 days)\n• 💀 Dead (no activity >30 days)")

	if totalPages > 1 {
		message.WriteString(fmt.Sprintf("\n\n📖 Use <code>/fudlist_[page]</code> to navigate\nExample: <code>/fudlist_2</code>"))
	}

	t.SendMessage(chatID, message.String())
}

func (t *TelegramService) handleExportFudListCommand(chatID int64) {
	fudUsers, err := t.dbService.GetAllFUDUsersFromCache()
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error retrieving FUD users: %v", err))
		return
	}

	if len(fudUsers) == 0 {
		t.SendMessage(chatID, "✅ No FUD users detected")
		return
	}

	// Collect usernames without @ symbol
	var usernames []string
	for _, user := range fudUsers {
		username := user["username"].(string)
		usernames = append(usernames, username)
	}

	// Join with commas
	exportText := strings.Join(usernames, ", ")

	message := fmt.Sprintf("📋 <b>FUD Users Export (%d total)</b>\n\n<code>%s</code>", len(fudUsers), exportText)

	t.SendMessage(chatID, message)
}

func (t *TelegramService) handleTopFudCommand(chatID int64, args []string, command string) {
	log.Printf("🔍 TopFud command started - chatID: %d, command: %s", chatID, command)
	t.SendMessage(chatID, "🔄 Starting TopFud analysis...")

	// Parse page number from command or arguments
	page := 1

	// Check if page number is in command format /topfud_X
	if strings.HasPrefix(command, "/topfud_") {
		pageStr := strings.TrimPrefix(command, "/topfud_")
		if pageNum, err := strconv.Atoi(pageStr); err == nil && pageNum > 0 {
			page = pageNum
		}
		log.Printf("📄 Page number from command: %d", page)
	} else if len(args) > 0 {
		// Fallback to old format with arguments
		if pageNum, err := strconv.Atoi(args[0]); err == nil && pageNum > 0 {
			page = pageNum
		}
		log.Printf("📄 Page number from args: %d", page)
	}

	const pageSize = 10 // Users per page

	log.Printf("🔍 Calling GetActiveFUDUsersSortedByLastMessage...")
	t.SendMessage(chatID, "🔍 Querying database for FUD users...")

	fudUsers, err := t.dbService.GetActiveFUDUsersSortedByLastMessage()
	if err != nil {
		log.Printf("❌ Error retrieving active FUD users: %v", err)
		t.SendMessage(chatID, fmt.Sprintf("❌ Error retrieving active FUD users: %v", err))
		return
	}

	log.Printf("📊 Found %d FUD users from cache", len(fudUsers))
	t.SendMessage(chatID, fmt.Sprintf("📊 Found %d FUD users in cache", len(fudUsers)))

	if len(fudUsers) == 0 {
		t.SendMessage(chatID, "✅ <b>No Active FUD Users Found</b>\n\n🎉 Great news! No active FUD users have been detected in the cache.")
		return
	}

	log.Printf("📊 Preparing to display results...")
	t.SendMessage(chatID, "📊 Preparing results display...")

	totalPages := (len(fudUsers) + pageSize - 1) / pageSize
	if page > totalPages {
		page = totalPages
	}

	startIdx := (page - 1) * pageSize
	endIdx := startIdx + pageSize
	if endIdx > len(fudUsers) {
		endIdx = len(fudUsers)
	}

	log.Printf("📄 Page info: %d/%d, showing users %d-%d", page, totalPages, startIdx+1, endIdx)

	var message strings.Builder

	aliveCount := 0
	deadCount := 0

	// Show users for current page
	for i := startIdx; i < endIdx; i++ {
		user := fudUsers[i]

		username := user["username"].(string)
		userID := user["user_id"].(string)
		lastMessageDate := user["last_message_date"].(time.Time)
		isAlive := user["is_alive"].(bool)
		status := user["status"].(string)

		if isAlive {
			aliveCount++
		} else {
			deadCount++
		}

		statusEmoji := "💀"
		if isAlive {
			statusEmoji = "🟢"
		}

		message.WriteString(fmt.Sprintf("<b>%d.</b> 💾 @%s (%s) %s %s\n", i+1, username, userID, statusEmoji, status))

		if !lastMessageDate.IsZero() {
			message.WriteString(fmt.Sprintf("    💬 Last msg: %s\n", lastMessageDate.Format("2006-01-02 15:04")))
		} else {
			message.WriteString("    💬 Last msg: unknown\n")
		}

		// Add enhanced command links
		message.WriteString(fmt.Sprintf("      https://x.com/%s\n", username))
		message.WriteString("\n")
	}

	// Add pagination controls
	if totalPages > 1 {
		message.WriteString("📄 <b>Navigation:</b>\n")
		if page > 1 {
			message.WriteString(fmt.Sprintf("  ⬅️ /topfud_%d (Previous)\n", page-1))
		}
		if page < totalPages {
			message.WriteString(fmt.Sprintf("  ➡️ /topfud_%d (Next)\n", page+1))
		}
		message.WriteString("\n")
	}

	if totalPages > 1 {
		message.WriteString(fmt.Sprintf("\n\n📖 Use <code>/topfud_[page]</code> to navigate\nExample: <code>/topfud_2</code>"))
	}

	t.SendMessage(chatID, message.String())
}

func (t *TelegramService) handleTasksCommand(chatID int64) {
	log.Printf("📋 Tasks command started for chatID: %d", chatID)

	tasks, err := t.dbService.GetAllRunningAnalysisTasks()
	if err != nil {
		log.Printf("❌ Error retrieving analysis tasks: %v", err)
		t.SendMessage(chatID, fmt.Sprintf("❌ Error retrieving analysis tasks: %v", err))
		return
	}

	log.Printf("📊 Found %d running analysis tasks", len(tasks))

	if len(tasks) == 0 {
		log.Printf("✅ No running tasks, sending empty message")
		t.SendMessage(chatID, "✅ <b>No Running Analysis Tasks</b>\n\n🎯 All analysis tasks have been completed.")
		return
	}

	var message strings.Builder
	message.WriteString(fmt.Sprintf("🔄 <b>Running Analysis Tasks (%d total)</b>\n\n", len(tasks)))

	// Limit to first 20 tasks to avoid message being too long
	maxTasks := 20
	if len(tasks) > maxTasks {
		message.WriteString(fmt.Sprintf("📄 <i>Showing first %d tasks</i>\n\n", maxTasks))
	}

	for i, task := range tasks {
		// Limit number of tasks to prevent message being too long
		if i >= maxTasks {
			message.WriteString(fmt.Sprintf("... and %d more tasks\n\n", len(tasks)-maxTasks))
			break
		}

		statusEmoji := "⏳"
		if task.Status == ANALYSIS_STATUS_RUNNING {
			statusEmoji = "🔄"
		}

		stepEmoji := "🔄"
		switch task.CurrentStep {
		case ANALYSIS_STEP_INIT:
			stepEmoji = "⚙️"
		case ANALYSIS_STEP_USER_LOOKUP:
			stepEmoji = "🔍"
		case ANALYSIS_STEP_TICKER_SEARCH:
			stepEmoji = "📊"
		case ANALYSIS_STEP_FOLLOWERS:
			stepEmoji = "👥"
		case ANALYSIS_STEP_FOLLOWINGS:
			stepEmoji = "👤"
		case ANALYSIS_STEP_COMMUNITY_ACTIVITY:
			stepEmoji = "🏠"
		case ANALYSIS_STEP_CLAUDE_ANALYSIS:
			stepEmoji = "🤖"
		case ANALYSIS_STEP_SAVING_RESULTS:
			stepEmoji = "💾"
		}

		elapsed := time.Since(task.StartedAt)
		elapsedStr := fmt.Sprintf("%.0fs", elapsed.Seconds())
		if elapsed.Minutes() >= 1 {
			elapsedStr = fmt.Sprintf("%.1fm", elapsed.Minutes())
		}

		message.WriteString(fmt.Sprintf("<b>%d.</b> %s @%s\n", i+1, statusEmoji, task.Username))
		message.WriteString(fmt.Sprintf("    %s Step: %s\n", stepEmoji, task.ProgressText))
		message.WriteString(fmt.Sprintf("    ⏱️ Running: %s\n", elapsedStr))
		message.WriteString(fmt.Sprintf("    🆔 Task ID: <code>%s</code>\n\n", task.ID))

		log.Printf("📋 Added task %d: %s (%s)", i+1, task.Username, task.CurrentStep)
	}

	message.WriteString("💡 Use <code>/analyze_&lt;username&gt;</code> to start new analysis")

	finalMessage := message.String()
	log.Printf("📤 Sending tasks message with length: %d characters", len(finalMessage))

	err = t.SendMessage(chatID, finalMessage)
	if err != nil {
		log.Printf("❌ Failed to send tasks message: %v", err)
		t.SendMessage(chatID, "❌ Failed to send tasks list - message might be too long")
	} else {
		log.Printf("✅ Successfully sent tasks message")
	}
}

func (t *TelegramService) handleTop20AnalyzeCommand(chatID int64) {
	// Get top 20 most active users
	users, err := t.dbService.GetTopActiveUsers(20)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error retrieving top users: %v", err))
		return
	}

	if len(users) == 0 {
		t.SendMessage(chatID, "📭 No users found in database")
		return
	}

	// Send initial confirmation
	t.SendMessage(chatID, fmt.Sprintf("🔄 <b>Starting Top 20 Analysis</b>\n\n📊 Found %d users to analyze\n⏳ This will take several minutes...\n\n💡 Use /tasks to monitor progress", len(users)))

	// Start analysis for each user in background
	analysisCount := 0
	skippedCount := 0

	for _, user := range users {
		// Check if user already has recent cached analysis
		if t.dbService.HasValidCachedAnalysis(user.ID) {
			log.Printf("Skipping user %s - has valid cached analysis", user.Username)
			skippedCount++
			continue
		}

		// Generate task ID for tracking
		taskID := t.generateNotificationID()

		// Create analysis task in database
		task := &AnalysisTaskModel{
			ID:             taskID,
			Username:       user.Username,
			UserID:         user.ID,
			Status:         ANALYSIS_STATUS_PENDING,
			CurrentStep:    ANALYSIS_STEP_INIT,
			ProgressText:   "Queued for analysis...",
			TelegramChatID: chatID,
			MessageID:      0, // No progress messages for batch analysis
			StartedAt:      time.Now(),
		}

		err = t.dbService.CreateAnalysisTask(task)
		if err != nil {
			log.Printf("Failed to create analysis task for user %s: %v", user.Username, err)
			continue
		}

		// Start analysis in background
		go t.processAnalysisTask(taskID, chatID)
		analysisCount++

		// Small delay between launches to avoid overwhelming the system
		time.Sleep(100 * time.Millisecond)
	}

	// Send summary
	summaryMessage := fmt.Sprintf("🚀 <b>Top 20 Analysis Started</b>\n\n📊 <b>Statistics:</b>\n• ✅ Started: %d analyses\n• ⏭️ Skipped: %d (cached)\n• 📋 Total: %d users\n\n🔍 Use /tasks to monitor progress\n💡 Use /fudlist to see detected FUD users", analysisCount, skippedCount, len(users))
	t.SendMessage(chatID, summaryMessage)

	log.Printf("Started top 20 analysis: %d analyses queued, %d skipped", analysisCount, skippedCount)
}
func (t *TelegramService) handleTop100AnalyzeCommand(chatID int64) {
	// Get top 20 most active users
	users, err := t.dbService.GetTopActiveUsers(100)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error retrieving top users: %v", err))
		return
	}

	if len(users) == 0 {
		t.SendMessage(chatID, "📭 No users found in database")
		return
	}

	// Send initial confirmation
	t.SendMessage(chatID, fmt.Sprintf("🔄 <b>Starting Top 100 Analysis</b>\n\n📊 Found %d users to analyze\n⏳ This will take several minutes...\n\n💡 Use /tasks to monitor progress", len(users)))

	// Start analysis for each user in background
	analysisCount := 0
	skippedCount := 0

	for _, user := range users {
		// Check if user already has recent cached analysis
		if t.dbService.HasValidCachedAnalysis(user.ID) {
			log.Printf("Skipping user %s - has valid cached analysis", user.Username)
			skippedCount++
			continue
		}

		// Generate task ID for tracking
		taskID := t.generateNotificationID()

		// Create analysis task in database
		task := &AnalysisTaskModel{
			ID:             taskID,
			Username:       user.Username,
			UserID:         user.ID,
			Status:         ANALYSIS_STATUS_PENDING,
			CurrentStep:    ANALYSIS_STEP_INIT,
			ProgressText:   "Queued for analysis...",
			TelegramChatID: chatID,
			MessageID:      0, // No progress messages for batch analysis
			StartedAt:      time.Now(),
		}

		err = t.dbService.CreateAnalysisTask(task)
		if err != nil {
			log.Printf("Failed to create analysis task for user %s: %v", user.Username, err)
			continue
		}

		// Start analysis in background
		go t.processAnalysisTask(taskID, chatID)
		analysisCount++

		// Small delay between launches to avoid overwhelming the system
		time.Sleep(100 * time.Millisecond)
	}

	// Send summary
	summaryMessage := fmt.Sprintf("🚀 <b>Top 100 Analysis Started</b>\n\n📊 <b>Statistics:</b>\n• ✅ Started: %d analyses\n• ⏭️ Skipped: %d (cached)\n• 📋 Total: %d users\n\n🔍 Use /tasks to monitor progress\n💡 Use /fudlist to see detected FUD users", analysisCount, skippedCount, len(users))
	t.SendMessage(chatID, summaryMessage)

	log.Printf("Started top 20 analysis: %d analyses queued, %d skipped", analysisCount, skippedCount)
}

func (t *TelegramService) handleBatchAnalyzeCommand(chatID int64, args []string) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		t.SendMessage(chatID, "❌ Invalid command format. Use /batch_analyze <user1,user2,user3>\n\n📝 <b>Examples:</b>\n• <code>/batch_analyze john,mary,bob</code>\n• <code>/batch_analyze user1, user2, user3</code>\n\n💡 Separate usernames with commas")
		return
	}

	// Join all arguments and split by comma
	userListStr := strings.Join(args, " ")
	usernames := strings.Split(userListStr, ",")

	// Clean and validate usernames
	var validUsernames []string
	var invalidUsernames []string

	for _, username := range usernames {
		username = strings.TrimSpace(username)
		username = strings.TrimPrefix(username, "@") // Remove @ if present

		if username == "" {
			continue
		}

		// Basic validation - check if it looks like a valid username
		if len(username) > 50 || strings.Contains(username, " ") {
			invalidUsernames = append(invalidUsernames, username)
			continue
		}

		validUsernames = append(validUsernames, username)
	}

	if len(validUsernames) == 0 {
		t.SendMessage(chatID, "❌ No valid usernames provided. Please check your input format.")
		return
	}

	if len(validUsernames) > 100 {
		t.SendMessage(chatID, fmt.Sprintf("❌ Too many users requested (%d). Maximum limit is 20 users per batch.", len(validUsernames)))
		return
	}

	// Send initial confirmation
	var confirmationMessage strings.Builder
	confirmationMessage.WriteString(fmt.Sprintf("🔄 <b>Starting Batch Analysis</b>\n\n📊 <b>Users to analyze (%d):</b>\n", len(validUsernames)))

	for i, username := range validUsernames {
		confirmationMessage.WriteString(fmt.Sprintf("%d. @%s\n", i+1, username))
	}

	if len(invalidUsernames) > 0 {
		confirmationMessage.WriteString(fmt.Sprintf("\n⚠️ <b>Skipped invalid usernames (%d):</b>\n", len(invalidUsernames)))
		for _, username := range invalidUsernames {
			confirmationMessage.WriteString(fmt.Sprintf("• %s\n", username))
		}
	}

	confirmationMessage.WriteString("\n⏳ Analysis will start shortly...\n💡 Results will be sent as notifications to this chat only")

	t.SendMessage(chatID, confirmationMessage.String())

	// Start analysis for each user
	analysisCount := 0
	skippedCount := 0

	for _, username := range validUsernames {
		// Check if user already has recent cached analysis
		user, err := t.dbService.GetUserByUsername(username)

		// Generate task ID for tracking
		taskID := t.generateNotificationID()

		// Create analysis task in database
		task := &AnalysisTaskModel{
			ID:             taskID,
			Username:       username,
			Status:         ANALYSIS_STATUS_PENDING,
			CurrentStep:    ANALYSIS_STEP_INIT,
			ProgressText:   "Queued for batch analysis...",
			TelegramChatID: chatID,
			MessageID:      0, // No progress messages for batch analysis
			StartedAt:      time.Now(),
		}

		if user != nil {
			task.UserID = user.ID
		}

		err = t.dbService.CreateAnalysisTask(task)
		if err != nil {
			log.Printf("Failed to create analysis task for user %s: %v", username, err)
			continue
		}

		// Start analysis in background with specific chat ID for notifications
		go t.processBatchAnalysisTask(taskID, chatID)
		analysisCount++

		// Small delay between launches to avoid overwhelming the system
		time.Sleep(150 * time.Millisecond)
	}

	// Send summary
	summaryMessage := fmt.Sprintf("🚀 <b>Batch Analysis Started</b>\n\n📊 <b>Statistics:</b>\n• ✅ Started: %d analyses\n• ⏭️ Skipped: %d (cached)\n• 📋 Total: %d users\n\n🔔 Results will be sent to this chat as they complete\n🔍 Use /tasks to monitor progress", analysisCount, skippedCount, len(validUsernames))
	t.SendMessage(chatID, summaryMessage)

	log.Printf("Started batch analysis for chat %d: %d analyses queued, %d skipped", chatID, analysisCount, skippedCount)
}

// processBatchAnalysisTask processes analysis task for batch analysis with specific chat notifications
func (t *TelegramService) processBatchAnalysisTask(taskID string, targetChatID int64) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Batch analysis task %s panicked: %v", taskID, r)
			t.dbService.SetAnalysisTaskError(taskID, fmt.Sprintf("Internal error: %v", r))
		}
	}()

	// Get task details
	task, err := t.dbService.GetAnalysisTask(taskID)
	if err != nil {
		log.Printf("Failed to get batch analysis task %s: %v", taskID, err)
		return
	}

	username := task.Username

	// Step 1: User lookup
	t.dbService.UpdateAnalysisTaskProgress(taskID, ANALYSIS_STEP_USER_LOOKUP, "Looking up user information...")
	user, err := t.dbService.GetUserByUsername(username)
	var userID string
	if err != nil {
		userID = "unknown_" + username
		log.Printf("User %s not found in database, using placeholder ID", username)
	} else {
		userID = user.ID
		// Update task with found user ID
		task.UserID = userID
		t.dbService.UpdateAnalysisTask(task)
	}

	// Step 2: Get user tweet for analysis context
	t.dbService.UpdateAnalysisTaskProgress(taskID, ANALYSIS_STEP_TICKER_SEARCH, "Searching for user's ticker mentions...")
	tweet, err := t.dbService.GetUserTweetForAnalysis(username)

	var newMessage twitterapi.NewMessage

	if err != nil {
		log.Printf("No tweet found for %s, creating placeholder data", username)

		newMessage = twitterapi.NewMessage{
			TweetID:      "batch_analysis_" + username,
			ReplyTweetID: "",
			Author: struct {
				UserName string
				Name     string
				ID       string
			}{
				UserName: username,
				Name:     username,
				ID:       userID,
			},
			Text:      "Batch analysis request - no recent tweets found",
			CreatedAt: time.Now().Format(time.RFC3339),
			ParentTweet: struct {
				ID     string
				Author string
				Text   string
			}{
				ID:     "placeholder_parent",
				Author: "system",
				Text:   "No parent tweet available - batch analysis",
			},
			GrandParentTweet: struct {
				ID     string
				Author string
				Text   string
			}{
				ID:     "",
				Author: "",
				Text:   "",
			},
			IsManualAnalysis:  true,
			ForceNotification: true,
			TaskID:            taskID,
			TelegramChatID:    targetChatID, // Set specific chat for notifications
		}
	} else {
		newMessage = twitterapi.NewMessage{
			TweetID:      tweet.ID,
			ReplyTweetID: tweet.InReplyToID,
			Author: struct {
				UserName string
				Name     string
				ID       string
			}{
				UserName: username,
				Name:     username,
				ID:       tweet.UserID,
			},
			Text:      tweet.Text,
			CreatedAt: tweet.CreatedAt.Format(time.RFC3339),
			ParentTweet: struct {
				ID     string
				Author string
				Text   string
			}{
				ID:     "batch_parent",
				Author: "system",
				Text:   "Batch analysis - limited context available",
			},
			GrandParentTweet: struct {
				ID     string
				Author string
				Text   string
			}{
				ID:     "",
				Author: "",
				Text:   "",
			},
			IsManualAnalysis:  true,
			ForceNotification: true,
			TaskID:            taskID,
			TelegramChatID:    targetChatID, // Set specific chat for notifications
		}
	}

	// Send to analysis channel for processing
	t.dbService.UpdateAnalysisTaskProgress(taskID, ANALYSIS_STEP_CLAUDE_ANALYSIS, "Starting AI analysis...")
	t.analysisChannel <- newMessage

	log.Printf("Sent batch analysis request for user %s (task %s) to analysis channel", username, taskID)
}

// sendCachedBatchNotification sends cached result as notification to specific chat
func (t *TelegramService) sendCachedBatchNotification(username, userID string, cachedResult SecondStepClaudeResponse, targetChatID int64) {
	// Create a formatted message for cached result
	alertType := cachedResult.FUDType
	if !cachedResult.IsFUDUser {
		alertType = "clean_user"
	}

	severityEmoji := "✅"
	if cachedResult.IsFUDUser {
		switch cachedResult.UserRiskLevel {
		case "critical":
			severityEmoji = "🚨🔥"
		case "high":
			severityEmoji = "🚨"
		case "medium":
			severityEmoji = "⚠️"
		default:
			severityEmoji = "ℹ️"
		}
	}

	message := fmt.Sprintf(`%s <b>Batch Analysis Result (Cached)</b>

👤 <b>User:</b> @%s
📊 <b>Status:</b> %s
🎯 <b>Type:</b> %s
📈 <b>Confidence:</b> %.0f%%
👥 <b>Profile:</b> %s

💾 <b>Source:</b> Cached analysis (< 24h)
🔍 <b>Commands:</b> /history_%s | /analyze_%s`,
		severityEmoji,
		username,
		map[bool]string{true: "FUD User Detected", false: "Clean User"}[cachedResult.IsFUDUser],
		alertType,
		cachedResult.FUDProbability*100,
		cachedResult.UserSummary,
		username, username)

	err := t.SendMessage(targetChatID, message)
	if err != nil {
		log.Printf("Failed to send cached batch notification for %s to chat %d: %v", username, targetChatID, err)
	} else {
		log.Printf("Sent cached batch analysis result for %s to chat %d", username, targetChatID)
	}
}

// handleAnalyzeAllCommand analyzes all users with messages, sorted by message count (descending)
func (t *TelegramService) handleAnalyzeAllCommand(chatID int64) {
	// Send initial confirmation
	t.SendMessage(chatID, "🔄 <b>Starting Full Database Analysis</b>\n\n📊 Getting list of all users with messages...\nThis may take a moment.")

	// Start analysis in background
	go t.processAnalyzeAllUsers(chatID)
}

// processAnalyzeAllUsers processes analysis for all users with progress tracking
func (t *TelegramService) processAnalyzeAllUsers(chatID int64) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Analyze all users panicked: %v", r)
			t.SendMessage(chatID, fmt.Sprintf("❌ Analysis failed with error: %v", r))
		}
	}()

	// Get all users sorted by message count (descending)
	users, err := t.dbService.GetTopActiveUsers(0) // 0 = no limit, get all users
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error getting users list: %v", err))
		return
	}

	if len(users) == 0 {
		t.SendMessage(chatID, "📭 No users found with messages in database")
		return
	}

	// Filter out users who already have cached analysis (< 24h)
	var usersToAnalyze []UserModel
	var skippedCount int

	for _, user := range users {
		if t.dbService.HasValidCachedAnalysis(user.ID) {
			skippedCount++
			continue
		}
		usersToAnalyze = append(usersToAnalyze, user)
	}

	totalUsers := len(users)
	toAnalyzeCount := len(usersToAnalyze)

	// Send status update
	statusMessage := fmt.Sprintf(`📊 <b>Analysis Preparation Complete</b>

👥 <b>Total users with messages:</b> %d
🔍 <b>Users to analyze:</b> %d
💾 <b>Cached (skipped):</b> %d

🚀 Starting analysis with buffer of 5 concurrent tasks...`, totalUsers, toAnalyzeCount, skippedCount)

	statusMsg, err := t.SendMessageWithResponse(chatID, statusMessage)
	if err != nil {
		log.Printf("Failed to send status message: %v", err)
		return
	}

	if toAnalyzeCount == 0 {
		t.EditMessage(chatID, statusMsg.Result.MessageID, "✅ All users already have recent analysis (cached). No new analysis needed.")
		return
	}

	// Start progress monitoring goroutine
	progressCtx := make(chan bool, 1)
	go t.monitorAnalysisAllProgress(chatID, statusMsg.Result.MessageID, toAnalyzeCount, progressCtx)

	// Process users in chunks, feeding to existing analysis channel
	sentCount := 0
	for i, user := range usersToAnalyze {
		// Create analysis task
		taskID, err := t.generateTaskID()
		if err != nil {
			log.Printf("Failed to generate task ID for user %s: %v", user.Username, err)
			continue
		}

		// Create task in database
		task := &AnalysisTaskModel{
			ID:             taskID,
			Username:       user.Username,
			UserID:         user.ID,
			Status:         ANALYSIS_STATUS_PENDING,
			CurrentStep:    ANALYSIS_STEP_INIT,
			ProgressText:   fmt.Sprintf("Queued for analysis (%d/%d)", i+1, toAnalyzeCount),
			TelegramChatID: chatID,
			MessageID:      0,
			StartedAt:      time.Now(),
		}

		err = t.dbService.CreateAnalysisTask(task)
		if err != nil {
			log.Printf("Failed to create analysis task for user %s: %v", user.Username, err)
			continue
		}

		// Send to existing analysis channel (will block if buffer is full - channel has buffer of 30)
		newMessage := twitterapi.NewMessage{
			Author: struct {
				UserName string
				Name     string
				ID       string
			}{
				UserName: user.Username,
				Name:     user.Name,
				ID:       user.ID,
			},
			TweetID:          "", // Not a specific tweet analysis
			IsManualAnalysis: true,
			TaskID:           taskID,
			TelegramChatID:   chatID,
		}

		t.analysisChannel <- newMessage
		sentCount++

		log.Printf("Sent user %s (%d/%d) to main analysis channel", user.Username, i+1, toAnalyzeCount)

		// Small delay to control flow and prevent overwhelming
		time.Sleep(300 * time.Millisecond)
	}

	// Stop progress monitoring
	progressCtx <- true

	// Send final status
	finalMessage := fmt.Sprintf("✅ <b>Analysis Complete</b>\n\n📊 <b>Final Statistics:</b>\n• 🚀 Sent for analysis: %d\n• 💾 Cached (skipped): %d\n• 📋 Total processed: %d\n\n🔔 All results have been sent to this chat", sentCount, skippedCount, totalUsers)
	t.SendMessage(chatID, finalMessage)

	log.Printf("Completed full database analysis: %d sent, %d skipped, %d total", sentCount, skippedCount, totalUsers)
}

// monitorAnalysisProgress monitors and reports analysis progress
func (t *TelegramService) monitorAnalysisAllProgress(chatID int64, messageID int64, totalUsers int, ctx chan bool) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx:
			// Analysis complete
			return
		case <-ticker.C:
			// Get current task statistics
			stats, err := t.getAnalysisStatistics()
			if err != nil {
				log.Printf("Failed to get analysis statistics: %v", err)
				continue
			}

			// Update status message
			statusMessage := fmt.Sprintf(`🔄 <b>Full Database Analysis Progress</b>

👥 <b>Total users to analyze:</b> %d

📊 <b>Current Status:</b>
• 📋 Pending: %d
• 🔄 Running: %d
• ✅ Completed: %d
• ❌ Failed: %d

⏱️ <b>Last updated:</b> %s`,
				totalUsers,
				stats["pending"],
				stats["running"],
				stats["completed"],
				stats["failed"],
				time.Now().Format("15:04:05"))

			err = t.EditMessage(chatID, messageID, statusMessage)
			if err != nil {
				log.Printf("Failed to update progress message: %v", err)
			}
		}
	}
}

// getAnalysisStatistics returns current analysis task statistics
func (t *TelegramService) getAnalysisStatistics() (map[string]int, error) {
	stats := make(map[string]int)

	// Get counts for each status
	var pending, running, completed, failed int64

	t.dbService.db.Model(&AnalysisTaskModel{}).Where("status = ?", ANALYSIS_STATUS_PENDING).Count(&pending)
	t.dbService.db.Model(&AnalysisTaskModel{}).Where("status = ?", ANALYSIS_STATUS_RUNNING).Count(&running)
	t.dbService.db.Model(&AnalysisTaskModel{}).Where("status = ?", ANALYSIS_STATUS_COMPLETED).Count(&completed)
	t.dbService.db.Model(&AnalysisTaskModel{}).Where("status = ?", ANALYSIS_STATUS_FAILED).Count(&failed)

	stats["pending"] = int(pending)
	stats["running"] = int(running)
	stats["completed"] = int(completed)
	stats["failed"] = int(failed)

	return stats, nil
}
