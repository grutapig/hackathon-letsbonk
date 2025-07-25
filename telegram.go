package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/grutapig/hackaton/twitterapi"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
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

	twitterApi             interface{}
	claudeApi              interface{}
	systemPromptSecondStep []byte
	ticker                 string
	analysisChannel        chan twitterapi.NewMessage
	loggingService         *LoggingService
	bot                    *tgbotapi.BotAPI
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
		Timeout:   30 * time.Second,
	}
	bot, err := tgbotapi.NewBotAPI(apiKey)
	if err != nil {
		return nil, fmt.Errorf("cannot initiate telegram bot, err: %s", err)
	}
	service := &TelegramService{
		bot:             bot,
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
	data, err := os.ReadFile(os.Getenv(ENV_CHAT_IDS_FILEPATH))
	if err == nil {
		chatIDStrings := strings.Split(string(data), "\n")
		for _, chatIDStr := range chatIDStrings {
			chatIDStr = strings.TrimSpace(chatIDStr)
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

	if initialChatIDs != "" {
		chatIDStrings := strings.Split(initialChatIDs, ",")
		for _, chatIDStr := range chatIDStrings {
			chatIDStr = strings.TrimSpace(chatIDStr)
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
			for chatId := range service.chatIDs {
				chatList = append(chatList, strconv.Itoa(int(chatId)))
			}
			err = os.WriteFile(os.Getenv(ENV_CHAT_IDS_FILEPATH), []byte(strings.Join(chatList, "\n")), 0655)
			if err != nil {
				log.Println("cannot write file with notification users list.", err)
			}
		}
	}()

	return service, nil
}

func (t *TelegramService) SetAnalysisServices(twitterApi interface{}, claudeApi interface{}, systemPromptSecondStep []byte, ticker string) {
	t.twitterApi = twitterApi
	t.claudeApi = claudeApi
	t.systemPromptSecondStep = systemPromptSecondStep
	t.ticker = ticker
}

func (t *TelegramService) StartListening() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := t.bot.GetUpdatesChan(u)

	go func() {
		for update := range updates {
			go t.HandleUpdate(update)
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

func (t *TelegramService) HandleUpdate(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	t.chatMutex.Lock()
	isNewChat := !t.chatIDs[chatID]
	if isNewChat {
		t.chatIDs[chatID] = true
		log.Printf("New Telegram chat registered: %d (from: %s)", chatID, update.Message.From.FirstName)
	}
	t.chatMutex.Unlock()

	if isNewChat {
		info := fmt.Sprintf("✅ Chat registered!\nChat ID: %d\nUser: %s %s\nUsername: @%s", chatID, update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName)
		go t.SendMessage(chatID, info)
	}

	if update.Message.Text != "" {
		text := strings.TrimSpace(update.Message.Text)

		parts := strings.Fields(text)
		if len(parts) == 0 {
			return
		}

		command := parts[0]
		args := parts[1:]

		switch {
		case strings.HasPrefix(command, "/reset"):
			if strings.Contains(os.Getenv(ENV_TELEGRAM_ADMIN_CHAT_ID), strconv.Itoa(int(chatID))) {
				t.SendMessage(chatID, "restarting bot")
				panic("bot restart command from telegram")
			} else {
				t.SendMessage(chatID, "only admins can restart bot")
			}
			return
		case strings.HasPrefix(command, "/detail_"):
			t.handleDetailCommand(chatID, text)
		case strings.HasPrefix(command, "/history_"):
			t.handleHistoryCommand(chatID, text)
		case strings.HasPrefix(command, "/export_"):
			t.handleExportCommand(chatID, text)
		case strings.HasPrefix(command, "/ticker_history_"):
			t.handleTickerHistoryCommand(chatID, text)
		case strings.HasPrefix(command, "/cache_"):
			t.handleCacheCommand(chatID, text)
		case command == "/analyze_all":
			if !t.isAdminChat(chatID) {
				t.SendMessage(chatID, "❌ Access denied. This command is restricted to administrators only.")
				return
			}
			t.handleAnalyzeAllCommand(chatID)
			return
		case strings.HasPrefix(command, "/analyze_"):
			t.handleAnalyzeCommand(chatID, text)
		case command == "/search":
			t.handleSearchCommand(chatID, args)
		case command == "/fudlist" || strings.HasPrefix(command, "/fudlist_"):
			t.handleFudListCommand(chatID, args, command)
		case command == "/goodlist" || strings.HasPrefix(command, "/goodlist_"):
			t.handleGoodListCommand(chatID, args, command)
		case command == "/exportfudlist":
			t.handleExportFudListCommand(chatID)
		case command == "/topfud" || strings.HasPrefix(command, "/topfud_"):
			t.handleTopFudCommand(chatID, args, command)
		case command == "/tasks":
			t.handleTasksCommand(chatID)
		case command == "/last5":
			t.handleLast5MessagesCommand(chatID)
		case command == "/u":
			t.SendMessage(chatID, fmt.Sprintf("users: %d", len(t.chatIDs)))
		case command == "/top20_analyze":
			if !t.isAdminChat(chatID) {
				t.SendMessage(chatID, "❌ Access denied. This command is restricted to administrators only.")
				return
			}
			t.handleTop20AnalyzeCommand(chatID)
		case command == "/top100_analyze":
			if !t.isAdminChat(chatID) {
				t.SendMessage(chatID, "❌ Access denied. This command is restricted to administrators only.")
				return
			}
			t.handleTop100AnalyzeCommand(chatID)
		case command == "/batch_analyze":
			t.handleBatchAnalyzeCommand(chatID, args)
		case command == "/update_reverse_auth":
			if !t.isAdminChat(chatID) {
				t.SendMessage(chatID, "❌ Access denied. This command is restricted to administrators only.")
				return
			}
			t.handleUpdateReverseAuthCommand(chatID, text)
		case command == "/start":
			t.handleStartCommand(chatID, strings.Join(args, ""))
		case command == "/help":
			t.handleHelpCommand(chatID)
		default:
			t.handleHelpCommand(chatID)
		}
	}
}
func (t *TelegramService) generateTaskID() (string, error) {
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
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
			if strings.Contains(err.Error(), `"error_code":403`) {
				t.removeChatId(chatID)
			}
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

	notificationID := t.generateNotificationID()

	t.notifMutex.Lock()
	t.notifications[notificationID] = alert
	t.notifMutex.Unlock()

	telegramMessage := t.formatter.FormatForTelegramWithDetail(alert, notificationID)

	return t.BroadcastMessage(telegramMessage)
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

func (t *TelegramService) processAnalysisTask(taskID string, chatID int64) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Analysis task %s panicked: %v", taskID, r)
			t.dbService.SetAnalysisTaskError(taskID, fmt.Sprintf("Internal error: %v", r))
		}
	}()

	task, err := t.dbService.GetAnalysisTask(taskID)
	if err != nil {
		log.Printf("Failed to get analysis task %s: %v", taskID, err)
		return
	}

	username := task.Username

	t.dbService.UpdateAnalysisTaskProgress(taskID, ANALYSIS_STEP_USER_LOOKUP, "Looking up user information...")
	user, err := t.dbService.GetUserByUsername(username)
	var userID string
	if err != nil {
		userID = "unknown_" + username
		log.Printf("User %s not found in database, using placeholder ID", username)
	} else {
		userID = user.ID

		task.UserID = userID
		t.dbService.UpdateAnalysisTask(task)
	}

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

	t.dbService.UpdateAnalysisTaskProgress(taskID, ANALYSIS_STEP_CLAUDE_ANALYSIS, "Sending for FUD analysis...")

	select {
	case t.analysisChannel <- newMessage:

		t.dbService.UpdateAnalysisTaskProgress(taskID, ANALYSIS_STEP_CLAUDE_ANALYSIS, "Processing with neural network...")

		log.Printf("Manual analysis task %s sent to Claude processing pipeline", taskID)

	default:

		t.dbService.SetAnalysisTaskError(taskID, "Analysis channel is full, please try again later")
	}
}
func (t *TelegramService) exportTickerHistoryAsFile(chatID int64, username, ticker string, opinions []UserTickerOpinionModel) {

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

	filename := fmt.Sprintf("%s_ticker_%s_%s.txt", username, ticker, time.Now().Format("20060102_150405"))
	err := t.writeToFile(filename, fileContent.String())
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error creating file: %v", err))
		return
	}

	caption := fmt.Sprintf("💰 <b>Ticker History Export</b>\n\n👤 User: @%s\n🏷️ Ticker: %s\n📊 Total Messages: %d\n📅 Generated: %s", username, ticker, len(opinions), time.Now().Format("2006-01-02 15:04:05"))

	err = t.SendDocument(chatID, filename, caption)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error sending file: %v\nFile created locally: %s", err, filename))
		return
	}

	go func() {
		time.Sleep(10 * time.Second)
		os.Remove(filename)
	}()

	t.SendMessage(chatID, "✅ Ticker history file sent successfully!")
}

func (t *TelegramService) formatAnalysisProgress(task *AnalysisTaskModel) string {
	if task.Status == ANALYSIS_STATUS_FAILED {
		return fmt.Sprintf(`❌ <b>Analysis Failed for @%s</b>

⚠️ <b>Error:</b> %s
🆔 <b>Task ID:</b> <code>%s</code>

🔄 You can try running the analysis again.`, task.Username, task.ErrorMessage, task.ID)
	}

	if task.Status == ANALYSIS_STATUS_COMPLETED {
		return fmt.Sprintf(`✅ <b>Analysis Completed for @%s</b>

📋 <b>Status:</b> Finished successfully
🔍 <b>Results:</b> Check FUD alerts for analysis results
🆔 <b>Task ID:</b> <code>%s</code>

✅ Analysis has been completed and results sent to notification system.`, task.Username, task.ID)
	}

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

	elapsed := time.Since(task.StartedAt)
	elapsedStr := fmt.Sprintf("%.0fs", elapsed.Seconds())
	if elapsed.Minutes() >= 1 {
		elapsedStr = fmt.Sprintf("%.1fm", elapsed.Minutes())
	}

	return fmt.Sprintf(`🔄 <b>Analyzing @%s</b>

%s <b>Current Step:</b> %s
⏱️ <b>Running Time:</b> %s
🆔 <b>Task ID:</b> <code>%s</code>

⏳ Please wait, analysis in progress...`, task.Username, stepEmoji, stepText, elapsedStr, task.ID)
}

func (t *TelegramService) processBatchAnalysisTask(taskID string, targetChatID int64) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Batch analysis task %s panicked: %v", taskID, r)
			t.dbService.SetAnalysisTaskError(taskID, fmt.Sprintf("Internal error: %v", r))
		}
	}()

	task, err := t.dbService.GetAnalysisTask(taskID)
	if err != nil {
		log.Printf("Failed to get batch analysis task %s: %v", taskID, err)
		return
	}

	username := task.Username

	t.dbService.UpdateAnalysisTaskProgress(taskID, ANALYSIS_STEP_USER_LOOKUP, "Looking up user information...")
	user, err := t.dbService.GetUserByUsername(username)
	var userID string
	if err != nil {
		userID = "unknown_" + username
		log.Printf("User %s not found in database, using placeholder ID", username)
	} else {
		userID = user.ID

		task.UserID = userID
		t.dbService.UpdateAnalysisTask(task)
	}

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
			TelegramChatID:    targetChatID,
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
			TelegramChatID:    targetChatID,
		}
	}

	t.dbService.UpdateAnalysisTaskProgress(taskID, ANALYSIS_STEP_CLAUDE_ANALYSIS, "Starting AI analysis...")
	t.analysisChannel <- newMessage

	log.Printf("Sent batch analysis request for user %s (task %s) to analysis channel", username, taskID)
}

func (t *TelegramService) sendCachedBatchNotification(username, userID string, cachedResult SecondStepClaudeResponse, targetChatID int64) {

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
🔍 <b>Commands:</b> /history_%s | /analyze_%s`, severityEmoji, username, map[bool]string{true: "FUD User Detected", false: "Clean User"}[cachedResult.IsFUDUser], alertType, cachedResult.FUDProbability*100, cachedResult.UserSummary, username, username)

	err := t.SendMessage(targetChatID, message)
	if err != nil {
		log.Printf("Failed to send cached batch notification for %s to chat %d: %v", username, targetChatID, err)
	} else {
		log.Printf("Sent cached batch analysis result for %s to chat %d", username, targetChatID)
	}
}

func (t *TelegramService) processAnalyzeAllUsers(chatID int64) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Analyze all users panicked: %v", r)
			t.SendMessage(chatID, fmt.Sprintf("❌ Analysis failed with error: %v", r))
		}
	}()

	users, err := t.dbService.GetTopActiveUsers(0)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error getting users list: %v", err))
		return
	}

	if len(users) == 0 {
		t.SendMessage(chatID, "📭 No users found with messages in database")
		return
	}

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

	progressCtx := make(chan bool, 1)
	go t.monitorAnalysisAllProgress(chatID, statusMsg.Result.MessageID, toAnalyzeCount, progressCtx)

	sentCount := 0
	for i, user := range usersToAnalyze {

		taskID, err := t.generateTaskID()
		if err != nil {
			log.Printf("Failed to generate task ID for user %s: %v", user.Username, err)
			continue
		}

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
			TweetID:          "",
			IsManualAnalysis: true,
			TaskID:           taskID,
			TelegramChatID:   chatID,
		}

		t.analysisChannel <- newMessage
		sentCount++

		log.Printf("Sent user %s (%d/%d) to main analysis channel", user.Username, i+1, toAnalyzeCount)

		time.Sleep(300 * time.Millisecond)
	}

	progressCtx <- true

	finalMessage := fmt.Sprintf("✅ <b>Analysis Complete</b>\n\n📊 <b>Final Statistics:</b>\n• 🚀 Sent for analysis: %d\n• 💾 Cached (skipped): %d\n• 📋 Total processed: %d\n\n🔔 All results have been sent to this chat", sentCount, skippedCount, totalUsers)
	t.SendMessage(chatID, finalMessage)

	log.Printf("Completed full database analysis: %d sent, %d skipped, %d total", sentCount, skippedCount, totalUsers)
}

func (t *TelegramService) monitorAnalysisAllProgress(chatID int64, messageID int64, totalUsers int, ctx chan bool) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx:

			return
		case <-ticker.C:

			stats, err := t.getAnalysisStatistics()
			if err != nil {
				log.Printf("Failed to get analysis statistics: %v", err)
				continue
			}

			statusMessage := fmt.Sprintf(`🔄 <b>Full Database Analysis Progress</b>

👥 <b>Total users to analyze:</b> %d

📊 <b>Current Status:</b>
• 📋 Pending: %d
• 🔄 Running: %d
• ✅ Completed: %d
• ❌ Failed: %d

⏱️ <b>Last updated:</b> %s`, totalUsers, stats["pending"], stats["running"], stats["completed"], stats["failed"], time.Now().Format("15:04:05"))

			err = t.EditMessage(chatID, messageID, statusMessage)
			if err != nil {
				log.Printf("Failed to update progress message: %v", err)
			}
		}
	}
}

func (t *TelegramService) getAnalysisStatistics() (map[string]int, error) {
	stats := make(map[string]int)

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

func (t *TelegramService) removeChatId(chatId int64) {
	t.chatMutex.Lock()
	defer t.chatMutex.Unlock()
	delete(t.chatIDs, chatId)
}

func (t *TelegramService) parseCurlCommand(curlCommand string) (string, string, string, error) {

	curlCommand = strings.TrimSpace(curlCommand)

	authRegex := regexp.MustCompile(`-H\s+['"]Authorization:\s*([^'"]+)['"]`)
	csrfRegex := regexp.MustCompile(`-H\s+['"]x-csrf-token:\s*([^'"]+)['"]`)
	cookieRegex := regexp.MustCompile(`-H\s+['"]Cookie:\s*([^'"]+)['"]`)

	var authorization, csrfToken, cookie string

	if matches := authRegex.FindStringSubmatch(curlCommand); len(matches) > 1 {
		authorization = matches[1]
	} else {
		return "", "", "", fmt.Errorf("Authorization header not found in curl command")
	}

	if matches := csrfRegex.FindStringSubmatch(curlCommand); len(matches) > 1 {
		csrfToken = matches[1]
	} else {
		return "", "", "", fmt.Errorf("x-csrf-token header not found in curl command")
	}

	if matches := cookieRegex.FindStringSubmatch(curlCommand); len(matches) > 1 {
		cookie = matches[1]
	} else {
		return "", "", "", fmt.Errorf("Cookie header not found in curl command")
	}

	return authorization, csrfToken, cookie, nil
}

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

			progressText := t.formatAnalysisProgress(task)
			err = t.EditMessage(task.TelegramChatID, task.MessageID, progressText)
			if err != nil {
				log.Printf("Failed to update progress message for task %s: %v", taskID, err)
			}

			if task.Status == ANALYSIS_STATUS_COMPLETED || task.Status == ANALYSIS_STATUS_FAILED {
				return
			}
		}
	}
}

func (t *TelegramService) updateEnvFile(authorization, csrfToken, cookie string) error {
	envPath := ".dev.env"

	content, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("failed to read .env file: %v", err)
	}

	envContent := string(content)

	envContent = t.updateEnvLine(envContent, "twitter_reverse_authorization", authorization)
	envContent = t.updateEnvLine(envContent, "twitter_reverse_csrf_token", csrfToken)
	envContent = t.updateEnvLine(envContent, "twitter_reverse_cookie", cookie)

	err = os.WriteFile(envPath, []byte(envContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write .env file: %v", err)
	}

	return nil
}

func (t *TelegramService) updateEnvLine(content, key, value string) string {
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		if strings.HasPrefix(line, key+"=") {
			lines[i] = fmt.Sprintf("%s=%s", key, value)
			break
		}
	}

	return strings.Join(lines, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
