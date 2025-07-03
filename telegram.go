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
			case command == "/analyze":
				go t.handleAnalyzeCommand(chatID, args)
			case command == "/search":
				go t.handleSearchCommand(chatID, args)
			case command == "/import":
				go t.handleImportCommand(chatID, args)
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
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=1", t.apiKey, t.lastOffset)

	resp, err := t.client.Get(url)
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
	parts := strings.Split(command, "_")
	if len(parts) != 2 {
		t.SendMessage(chatID, "❌ Invalid command format. Use /detail_<id>")
		return
	}

	notificationID := parts[1]

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
	parts := strings.Split(command, "_")
	if len(parts) != 2 {
		t.SendMessage(chatID, "❌ Invalid command format. Use /history_<username>")
		return
	}

	username := parts[1]

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

func (t *TelegramService) handleExportCommand(chatID int64, command string) {
	// Extract username from command "/export_username"
	parts := strings.Split(command, "_")
	if len(parts) != 2 {
		t.SendMessage(chatID, "❌ Invalid command format. Use /export_<username>")
		return
	}

	username := parts[1]

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
		searchResults.WriteString(fmt.Sprintf("    Commands: /history_%s | /analyze %s\n\n", user.Username, user.Username))
	}

	// Add note about commands
	searchResults.WriteString("💡 <b>Quick Actions:</b>\n• Tap /history_username to view recent messages\n• Tap /analyze username to run second step analysis")

	t.SendMessage(chatID, searchResults.String())
}

func (t *TelegramService) handleAnalyzeCommand(chatID int64, args []string) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		t.SendMessage(chatID, "❌ Invalid command format. Use /analyze <username>\nExample: /analyze suspicious_user")
		return
	}

	username := strings.TrimSpace(args[0])

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
	go t.processAnalysisTask(taskID)

	// Start progress monitor
	go t.monitorAnalysisProgress(taskID)
}

func (t *TelegramService) handleImportCommand(chatID int64, args []string) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		t.SendMessage(chatID, "❌ Invalid command format. Use /import <csv_file>\nExample: /import community_tweets.csv")
		return
	}

	csvFile := strings.TrimSpace(args[0])

	// Send processing message
	t.SendMessage(chatID, fmt.Sprintf("🔄 Starting CSV import from '%s'...\nThis may take several minutes for large files.", csvFile))

	// Run import in goroutine to avoid blocking
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.SendMessage(chatID, fmt.Sprintf("❌ Import failed with panic: %v", r))
			}
		}()

		// Create CSV importer
		importer := NewCSVImporter(t.dbService)

		// Run import
		result, err := importer.ImportCSV(csvFile)
		if err != nil {
			t.SendMessage(chatID, fmt.Sprintf("❌ Import failed: %v", err))
			return
		}

		// Send success message with results
		successMessage := fmt.Sprintf("✅ <b>CSV Import Complete!</b>\n\n📊 <b>Import Statistics:</b>\n• Original tweets: %d\n• Reply tweets: %d\n• Remaining tweets: %d\n• Skipped tweets: %d\n• <b>Total processed: %d</b>\n\n📁 File: %s",
			result.OriginalTweets,
			result.ReplyTweets,
			result.RemainingTweets,
			result.SkippedTweets,
			result.TotalProcessed,
			csvFile)

		if result.SkippedTweets > 0 {
			successMessage += fmt.Sprintf("\n\n⚠️ %d tweets were skipped (missing parent tweets)", result.SkippedTweets)
		}

		t.SendMessage(chatID, successMessage)
	}()
}

func (t *TelegramService) handleHelpCommand(chatID int64) {
	helpMessage := `🤖 <b>FUD Detection Bot - Available Commands</b>

🔍 <b>Search & Analysis Commands:</b>
• <code>/search [query]</code> - Search users by username/name
  Example: /search john or /search (shows top 10 active users)

• <code>/analyze &lt;username&gt;</code> - Run manual FUD analysis
  Example: /analyze suspicious_user

📊 <b>User Investigation Commands:</b>
• <code>/history_&lt;username&gt;</code> - View recent messages (20 latest)
  Example: /history_john_doe

• <code>/export_&lt;username&gt;</code> - Export full message history as file
  Example: /export_john_doe

• <code>/detail_&lt;id&gt;</code> - View detailed FUD analysis
  (ID provided in alert notifications)

📁 <b>Data Management Commands:</b>
• <code>/import &lt;csv_file&gt;</code> - Import tweets from CSV file
  Example: /import community_tweets.csv

❓ <b>Help Commands:</b>
• <code>/help</code> or <code>/start</code> - Show this help message

💡 <b>Usage Tips:</b>
• Commands with underscore (_) need exact format
• Commands with space accept parameters
• All commands are case-sensitive
• Bot responds to FUD alerts automatically

🔔 <b>Alert Types:</b>
• 🚨🔥 Critical - Immediate action required
• 🚨 High - Monitor closely  
• ⚠️ Medium - Standard monitoring
• ℹ️ Low - Log and watch

👤 <b>Your Chat ID:</b> <code>%d</code>`

	t.SendMessage(chatID, fmt.Sprintf(helpMessage, chatID))
}

// processAnalysisTask processes the actual analysis work
func (t *TelegramService) processAnalysisTask(taskID string) {
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
