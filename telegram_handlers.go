package main

import (
	"fmt"
	"github.com/grutapig/hackaton/twitterapi_reverse"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func (t *TelegramService) handleSearchCommand(chatID int64, args []string) {
	var users []UserModel
	var err error
	var searchTitle string

	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {

		users, err = t.dbService.GetTopActiveUsers(10)
		searchTitle = "🔥 <b>Top 10 Most Active Users</b>"
	} else {

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

		searchResults.WriteString(fmt.Sprintf("    Commands: /history_%s | /analyze_%s\n\n", user.Username, user.Username))
	}

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

	taskID := t.generateNotificationID()

	initialText := fmt.Sprintf("🔄 <b>Starting Analysis for @%s</b>\n\n📋 <b>Status:</b> Initializing...\n🆔 <b>Task ID:</b> <code>%s</code>\n\n⏳ Please wait, this may take a few minutes.", username, taskID)
	messageID, err := t.SendMessageWithID(chatID, initialText)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Failed to start analysis: %v", err))
		return
	}

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

	go t.processAnalysisTask(taskID, chatID)

	go t.monitorAnalysisProgress(taskID)
}

func (t *TelegramService) handleHelpCommand(chatID int64) {
	helpMessage := `🤖 <b>FUD Detection Bot - Available Commands</b>

🔍 <b>Search & Analysis Commands:</b>
• /search - Search users by username/name

📊 <b>Analysis Management:</b>
• /fudlist - Show all detected FUD users
• /topfud - Show cached FUD users sorted by last message
• /exportfudlist - Export FUD usernames as comma-separated list

❓ <b>Help Commands:</b>
• /help - Show this help message
• /start - Show this help message

👤 <b>Your Chat ID:</b> %d`

	t.SendMessage(chatID, fmt.Sprintf(helpMessage, chatID))
}

func (t *TelegramService) handleFudListCommand(chatID int64, args []string, command string) {

	page := 1

	if strings.HasPrefix(command, "/fudlist_") {
		pageStr := strings.TrimPrefix(command, "/fudlist_")
		if pageNum, err := strconv.Atoi(pageStr); err == nil && pageNum > 0 {
			page = pageNum
		}
	} else if len(args) > 0 {

		if pageNum, err := strconv.Atoi(args[0]); err == nil && pageNum > 0 {
			page = pageNum
		}
	}

	const pageSize = 10

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

		message.WriteString("    🔍 <b>Commands:</b>\n")
		message.WriteString(fmt.Sprintf("      /export_%s - Message history\n", username))
		message.WriteString(fmt.Sprintf("      /ticker_history_%s - Ticker posts\n", username))
		message.WriteString(fmt.Sprintf("      /cache_%s - detailed analysis\n", username))
		message.WriteString(fmt.Sprintf("      https://x.com/%s\n", username))
		message.WriteString("\n")
	}

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

	var usernames []string
	for _, user := range fudUsers {
		username := user["username"].(string)
		usernames = append(usernames, username)
	}

	exportText := strings.Join(usernames, ", ")

	message := fmt.Sprintf("📋 <b>FUD Users Export (%d total)</b>\n\n<code>%s</code>", len(fudUsers), exportText)

	t.SendMessage(chatID, message)
}

func (t *TelegramService) handleTopFudCommand(chatID int64, args []string, command string) {
	log.Printf("🔍 TopFud command started - chatID: %d, command: %s", chatID, command)
	t.SendMessage(chatID, "🔄 Starting TopFud analysis...")

	page := 1

	if strings.HasPrefix(command, "/topfud_") {
		pageStr := strings.TrimPrefix(command, "/topfud_")
		if pageNum, err := strconv.Atoi(pageStr); err == nil && pageNum > 0 {
			page = pageNum
		}
		log.Printf("📄 Page number from command: %d", page)
	} else if len(args) > 0 {

		if pageNum, err := strconv.Atoi(args[0]); err == nil && pageNum > 0 {
			page = pageNum
		}
		log.Printf("📄 Page number from args: %d", page)
	}

	const pageSize = 10

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

		message.WriteString(fmt.Sprintf("      https://x.com/%s\n", username))
		message.WriteString("\n")
	}

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

	maxTasks := 20
	if len(tasks) > maxTasks {
		message.WriteString(fmt.Sprintf("📄 <i>Showing first %d tasks</i>\n\n", maxTasks))
	}

	for i, task := range tasks {

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

	users, err := t.dbService.GetTopActiveUsers(20)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error retrieving top users: %v", err))
		return
	}

	if len(users) == 0 {
		t.SendMessage(chatID, "📭 No users found in database")
		return
	}

	t.SendMessage(chatID, fmt.Sprintf("🔄 <b>Starting Top 20 Analysis</b>\n\n📊 Found %d users to analyze\n⏳ This will take several minutes...\n\n💡 Use /tasks to monitor progress", len(users)))

	analysisCount := 0
	skippedCount := 0

	for _, user := range users {

		if t.dbService.HasValidCachedAnalysis(user.ID) {
			log.Printf("Skipping user %s - has valid cached analysis", user.Username)
			skippedCount++
			continue
		}

		taskID := t.generateNotificationID()

		task := &AnalysisTaskModel{
			ID:             taskID,
			Username:       user.Username,
			UserID:         user.ID,
			Status:         ANALYSIS_STATUS_PENDING,
			CurrentStep:    ANALYSIS_STEP_INIT,
			ProgressText:   "Queued for analysis...",
			TelegramChatID: chatID,
			MessageID:      0,
			StartedAt:      time.Now(),
		}

		err = t.dbService.CreateAnalysisTask(task)
		if err != nil {
			log.Printf("Failed to create analysis task for user %s: %v", user.Username, err)
			continue
		}

		go t.processAnalysisTask(taskID, chatID)
		analysisCount++

		time.Sleep(100 * time.Millisecond)
	}

	summaryMessage := fmt.Sprintf("🚀 <b>Top 20 Analysis Started</b>\n\n📊 <b>Statistics:</b>\n• ✅ Started: %d analyses\n• ⏭️ Skipped: %d (cached)\n• 📋 Total: %d users\n\n🔍 Use /tasks to monitor progress\n💡 Use /fudlist to see detected FUD users", analysisCount, skippedCount, len(users))
	t.SendMessage(chatID, summaryMessage)

	log.Printf("Started top 20 analysis: %d analyses queued, %d skipped", analysisCount, skippedCount)
}

func (t *TelegramService) handleTop100AnalyzeCommand(chatID int64) {

	users, err := t.dbService.GetTopActiveUsers(100)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error retrieving top users: %v", err))
		return
	}

	if len(users) == 0 {
		t.SendMessage(chatID, "📭 No users found in database")
		return
	}

	t.SendMessage(chatID, fmt.Sprintf("🔄 <b>Starting Top 100 Analysis</b>\n\n📊 Found %d users to analyze\n⏳ This will take several minutes...\n\n💡 Use /tasks to monitor progress", len(users)))

	analysisCount := 0
	skippedCount := 0

	for _, user := range users {

		if t.dbService.HasValidCachedAnalysis(user.ID) {
			log.Printf("Skipping user %s - has valid cached analysis", user.Username)
			skippedCount++
			continue
		}

		taskID := t.generateNotificationID()

		task := &AnalysisTaskModel{
			ID:             taskID,
			Username:       user.Username,
			UserID:         user.ID,
			Status:         ANALYSIS_STATUS_PENDING,
			CurrentStep:    ANALYSIS_STEP_INIT,
			ProgressText:   "Queued for analysis...",
			TelegramChatID: chatID,
			MessageID:      0,
			StartedAt:      time.Now(),
		}

		err = t.dbService.CreateAnalysisTask(task)
		if err != nil {
			log.Printf("Failed to create analysis task for user %s: %v", user.Username, err)
			continue
		}

		go t.processAnalysisTask(taskID, chatID)
		analysisCount++

		time.Sleep(100 * time.Millisecond)
	}

	summaryMessage := fmt.Sprintf("🚀 <b>Top 100 Analysis Started</b>\n\n📊 <b>Statistics:</b>\n• ✅ Started: %d analyses\n• ⏭️ Skipped: %d (cached)\n• 📋 Total: %d users\n\n🔍 Use /tasks to monitor progress\n💡 Use /fudlist to see detected FUD users", analysisCount, skippedCount, len(users))
	t.SendMessage(chatID, summaryMessage)

	log.Printf("Started top 20 analysis: %d analyses queued, %d skipped", analysisCount, skippedCount)
}

func (t *TelegramService) handleAnalyzeAllCommand(chatID int64) {

	t.SendMessage(chatID, "🔄 <b>Starting Full Database Analysis</b>\n\n📊 Getting list of all users with messages...\nThis may take a moment.")

	go t.processAnalyzeAllUsers(chatID)
}

func (t *TelegramService) handleBatchAnalyzeCommand(chatID int64, args []string) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		t.SendMessage(chatID, "❌ Invalid command format. Use /batch_analyze <user1,user2,user3>\n\n📝 <b>Examples:</b>\n• <code>/batch_analyze john,mary,bob</code>\n• <code>/batch_analyze user1, user2, user3</code>\n\n💡 Separate usernames with commas")
		return
	}

	userListStr := strings.Join(args, " ")
	usernames := strings.Split(userListStr, ",")

	var validUsernames []string
	var invalidUsernames []string

	for _, username := range usernames {
		username = strings.TrimSpace(username)
		username = strings.TrimPrefix(username, "@")

		if username == "" {
			continue
		}

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

	analysisCount := 0
	skippedCount := 0

	for _, username := range validUsernames {

		user, err := t.dbService.GetUserByUsername(username)

		taskID := t.generateNotificationID()

		task := &AnalysisTaskModel{
			ID:             taskID,
			Username:       username,
			Status:         ANALYSIS_STATUS_PENDING,
			CurrentStep:    ANALYSIS_STEP_INIT,
			ProgressText:   "Queued for batch analysis...",
			TelegramChatID: chatID,
			MessageID:      0,
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

		go t.processBatchAnalysisTask(taskID, chatID)
		analysisCount++

		time.Sleep(150 * time.Millisecond)
	}

	summaryMessage := fmt.Sprintf("🚀 <b>Batch Analysis Started</b>\n\n📊 <b>Statistics:</b>\n• ✅ Started: %d analyses\n• ⏭️ Skipped: %d (cached)\n• 📋 Total: %d users\n\n🔔 Results will be sent to this chat as they complete\n🔍 Use /tasks to monitor progress", analysisCount, skippedCount, len(validUsernames))
	t.SendMessage(chatID, summaryMessage)

	log.Printf("Started batch analysis for chat %d: %d analyses queued, %d skipped", chatID, analysisCount, skippedCount)
}
func (t *TelegramService) handleUpdateReverseAuthCommand(chatID int64, curlCommand string) {
	if curlCommand == "" {
		t.SendMessage(chatID, "❌ Usage: /update_reverse_auth <curl_command>\n\nExample:\n/update_reverse_auth curl -H 'Authorization: Bearer xyz' -H 'x-csrf-token: abc' -H 'Cookie: session=123' ...")
		return
	}

	t.SendMessage(chatID, "🔄 Parsing curl command and updating reverse API authentication...")

	authorization, csrfToken, cookie, err := t.parseCurlCommand(curlCommand)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Failed to parse curl command: %v", err))
		return
	}

	oldAuth := os.Getenv(ENV_TWITTER_REVERSE_AUTHORIZATION)
	oldCsrf := os.Getenv(ENV_TWITTER_REVERSE_CSRF_TOKEN)
	oldCookie := os.Getenv(ENV_TWITTER_REVERSE_COOKIE)

	os.Setenv(ENV_TWITTER_REVERSE_AUTHORIZATION, authorization)
	os.Setenv(ENV_TWITTER_REVERSE_CSRF_TOKEN, csrfToken)
	os.Setenv(ENV_TWITTER_REVERSE_COOKIE, cookie)

	auth := &twitterapi_reverse.TwitterAuth{
		Authorization: authorization,
		XCSRFToken:    csrfToken,
		Cookie:        cookie,
	}

	t.SendMessage(chatID, "🧪 Testing reverse API with new credentials...")

	reverseService := twitterapi_reverse.NewTwitterReverseApi(auth, os.Getenv(ENV_PROXY_DSN), false)

	communityID := os.Getenv(ENV_DEMO_COMMUNITY_ID)
	tweets, err := reverseService.GetCommunityTweets(communityID, 10)
	if err != nil {

		os.Setenv(ENV_TWITTER_REVERSE_AUTHORIZATION, oldAuth)
		os.Setenv(ENV_TWITTER_REVERSE_CSRF_TOKEN, oldCsrf)
		os.Setenv(ENV_TWITTER_REVERSE_COOKIE, oldCookie)

		t.SendMessage(chatID, fmt.Sprintf("❌ Test failed, credentials rolled back: %v", err))
		return
	}

	err = t.updateEnvFile(authorization, csrfToken, cookie)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("⚠️ Authentication works but failed to update .env file: %v", err))
		return
	}

	var lastTweetText string
	if len(tweets) > 0 {
		lastTweet := tweets[len(tweets)-1]
		if len(lastTweet.Text) > 40 {
			lastTweetText = lastTweet.Text[:40] + "..."
		} else {
			lastTweetText = lastTweet.Text
		}
	}

	successMessage := fmt.Sprintf(`✅ <b>Reverse API authentication updated successfully!</b>

📊 <b>Test Results:</b>
• Found: %d tweets
• Last tweet: "%s"

🔧 <b>Updated credentials:</b>
• Authorization: %s...
• CSRF Token: %s...
• Cookie: %s...

💾 .env file updated and ready for use!`, len(tweets), lastTweetText, authorization[:min(20, len(authorization))], csrfToken[:min(20, len(csrfToken))], cookie[:min(50, len(cookie))])

	t.SendMessage(chatID, successMessage)
}

func (t *TelegramService) handleDetailCommand(chatID int64, command string) {

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

	detailMessage := t.formatter.FormatDetailedView(alert)
	t.SendMessage(chatID, detailMessage)
}

func (t *TelegramService) handleHistoryCommand(chatID int64, command string) {

	prefix := "/history_"
	if !strings.HasPrefix(command, prefix) {
		t.SendMessage(chatID, "❌ Invalid command format. Use /history_username")
		return
	}

	username := strings.TrimPrefix(command, prefix)

	tweets, err := t.dbService.GetUserMessagesByUsername(username, 20)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error retrieving messages for @%s: %v", username, err))
		return
	}

	if len(tweets) == 0 {
		t.SendMessage(chatID, fmt.Sprintf("📭 No messages found for @%s", username))
		return
	}

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

	historyMessage.WriteString(fmt.Sprintf("📄 For full message history: /export_%s", username))

	t.SendMessage(chatID, historyMessage.String())
}

func (t *TelegramService) handleTickerHistoryCommand(chatID int64, command string) {

	prefix := "/ticker_history_"
	if !strings.HasPrefix(command, prefix) {
		t.SendMessage(chatID, "❌ Invalid command format. Use /ticker_history_username")
		return
	}

	username := strings.TrimPrefix(command, prefix)
	ticker := t.ticker

	allOpinions, err := t.dbService.GetUserTickerOpinionsByUsername(username, ticker, 0)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error retrieving ticker history for @%s: %v", username, err))
		return
	}

	if len(allOpinions) == 0 {
		t.SendMessage(chatID, fmt.Sprintf("📭 No ticker-related messages found for @%s and %s", username, ticker))
		return
	}

	if len(allOpinions) > 15 {
		t.SendMessage(chatID, fmt.Sprintf("📊 Found %d ticker mentions for @%s (%s). Generating file...", len(allOpinions), username, ticker))
		t.exportTickerHistoryAsFile(chatID, username, ticker, allOpinions)
		return
	}

	var historyMessage strings.Builder
	historyMessage.WriteString(fmt.Sprintf("💰 <b>Ticker History for @%s (%s)</b> (%d messages)\n\n", username, ticker, len(allOpinions)))

	for i, opinion := range allOpinions {
		historyMessage.WriteString(fmt.Sprintf("<b>%d.</b> %s\n", i+1, opinion.TweetCreatedAt.Format("2006-01-02 15:04")))
		historyMessage.WriteString(fmt.Sprintf("💬 <i>%s</i>\n", t.truncateText(opinion.Text, 200)))

		if opinion.InReplyToID != "" && opinion.RepliedToAuthor != "" {
			historyMessage.WriteString(fmt.Sprintf("↳ <i>Reply to @%s: %s</i>\n", opinion.RepliedToAuthor, t.truncateText(opinion.RepliedToText, 100)))
		}

		historyMessage.WriteString(fmt.Sprintf("🆔 <code>%s</code>\n", opinion.TweetID))
		historyMessage.WriteString(fmt.Sprintf("🔍 <i>Search: %s</i>\n\n", opinion.SearchQuery))
	}

	historyMessage.WriteString(fmt.Sprintf("📊 Total ticker mentions: %d\n", len(allOpinions)))
	historyMessage.WriteString(fmt.Sprintf("📄 For full message history: /export_%s", username))

	t.SendMessage(chatID, historyMessage.String())
}

func (t *TelegramService) handleCacheCommand(chatID int64, command string) {

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

	var user *UserModel
	var err error

	if user, err = t.dbService.GetUserByUsername(userIdentifier); err != nil {

		if user, err = t.dbService.GetUser(userIdentifier); err != nil {
			t.SendMessage(chatID, fmt.Sprintf("❌ User not found: %s\nTried both username and ID lookup.", userIdentifier))
			return
		}
	}

	cachedAnalysis, err := t.dbService.GetCachedAnalysis(user.ID)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("💾 <b>No Cached Analysis Found</b>\n\n👤 User: @%s (ID: %s)\n❌ No cached analysis available or cache has expired.", user.Username, user.ID))
		return
	}

	var message strings.Builder
	message.WriteString(fmt.Sprintf("💾 <b>Cached Analysis for @%s</b>\n\n", user.Username))

	message.WriteString(fmt.Sprintf("👤 <b>User Details:</b>\n"))
	message.WriteString(fmt.Sprintf("• Username: @%s\n", user.Username))
	message.WriteString(fmt.Sprintf("• Name: %s\n", user.Name))
	message.WriteString(fmt.Sprintf("• https://x.com/%s\n", user.Username))
	message.WriteString(fmt.Sprintf("• User ID: <code>%s</code>\n\n", user.ID))

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

	if len(cachedAnalysis.KeyEvidence) > 0 {
		message.WriteString("🔍 <b>Key Evidence:</b>\n")
		for i, evidence := range cachedAnalysis.KeyEvidence {
			message.WriteString(fmt.Sprintf("%d. %s\n", i+1, evidence))
		}
		message.WriteString("\n")
	}

	if cachedAnalysis.DecisionReason != "" {
		message.WriteString(fmt.Sprintf("🧠 <b>Decision Reasoning:</b>\n<i>%s</i>\n\n", cachedAnalysis.DecisionReason))
	}

	var cacheRecord CachedAnalysisModel
	err = t.dbService.db.Where("user_id = ?", user.ID).First(&cacheRecord).Error
	if err == nil {
		message.WriteString("📅 <b>Cache Information:</b>\n")
		message.WriteString(fmt.Sprintf("• 🕐 Analyzed At: %s\n", cacheRecord.AnalyzedAt.Format("2006-01-02 15:04:05 UTC")))
		message.WriteString(fmt.Sprintf("• ⏰ Expires At: %s\n", cacheRecord.ExpiresAt.Format("2006-01-02 15:04:05 UTC")))

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

	message.WriteString("🔍 <b>Related Commands:</b>\n")
	message.WriteString(fmt.Sprintf("• /history_%s - Message history\n", user.Username))
	message.WriteString(fmt.Sprintf("• /ticker_history_%s - Ticker posts\n", user.Username))
	message.WriteString(fmt.Sprintf("• /export_%s - Full export\n", user.Username))
	message.WriteString(fmt.Sprintf("• /analyze_%s - Force new analysis\n", user.Username))

	t.SendMessage(chatID, message.String())
}

func (t *TelegramService) handleExportCommand(chatID int64, command string) {

	prefix := "/export_"
	if !strings.HasPrefix(command, prefix) {
		t.SendMessage(chatID, "❌ Invalid command format. Use /export_username")
		return
	}

	username := strings.TrimPrefix(command, prefix)

	tweets, err := t.dbService.GetAllUserMessagesByUsername(username)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error retrieving messages for @%s: %v", username, err))
		return
	}

	if len(tweets) == 0 {
		t.SendMessage(chatID, fmt.Sprintf("📭 No messages found for @%s", username))
		return
	}

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

	filename := fmt.Sprintf("%s_messages_%s.txt", username, time.Now().Format("20060102_150405"))
	err = t.writeToFile(filename, fileContent.String())
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error creating file: %v", err))
		return
	}

	caption := fmt.Sprintf("📄 <b>Full Message Export</b>\n\n👤 User: @%s\n📊 Total Messages: %d\n📅 Generated: %s", username, len(tweets), time.Now().Format("2006-01-02 15:04:05"))

	err = t.SendDocument(chatID, filename, caption)
	if err != nil {
		t.SendMessage(chatID, fmt.Sprintf("❌ Error sending file: %v\nFile created locally: %s", err, filename))
		return
	}

	go func() {
		time.Sleep(10 * time.Second)
		os.Remove(filename)
	}()

	t.SendMessage(chatID, "✅ Export file sent successfully!")
}

func (t *TelegramService) handleLast5MessagesCommand(chatID int64) {
	tweets, err := t.dbService.GetRecentTweets(5)
	if err != nil {
		errorMsg := fmt.Sprintf("❌ Error getting recent messages: %v", err)
		t.SendMessage(chatID, errorMsg)
		return
	}

	if len(tweets) == 0 {
		t.SendMessage(chatID, "📭 No messages found in database")
		return
	}

	var response strings.Builder
	response.WriteString("📄 Last 5 Messages:\n\n")

	for i, tweet := range tweets {
		response.WriteString(fmt.Sprintf("<b>%d</b> %s - %s\n",
			i+1,
			tweet.Username,
			tweet.CreatedAt.Format("2006-01-02 15:04:05")))

		tweetText := tweet.Text
		if len(tweetText) > 200 {
			tweetText = tweetText[:200] + "..."
		}
		response.WriteString(fmt.Sprintf("💬 %s\n\n", tweetText))
	}

	err = t.SendMessage(chatID, response.String())
	if err != nil {
		errorMsg := fmt.Sprintf("❌ Error sending message: %v", err)
		t.SendMessage(chatID, errorMsg)
	}
}
