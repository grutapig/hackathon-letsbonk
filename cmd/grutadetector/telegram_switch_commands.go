package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strconv"
	"strings"
)

func handleTelegramUpdates() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "set":
				args := update.Message.CommandArguments()
				if args != "" {
					if pages, err := strconv.Atoi(args); err == nil && pages > 0 {
						userSearchPages = pages
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Set user search pages to: %d", pages))
						bot.Send(msg)
					} else {
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Invalid number. Usage: /set 20")
						bot.Send(msg)
					}
				} else {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Usage: /set <number>")
					bot.Send(msg)
				}

			case "tweets":
				args := update.Message.CommandArguments()
				limit := 10
				if args != "" {
					if l, err := strconv.Atoi(args); err == nil && l > 0 && l <= 100 {
						limit = l
					}
				}

				tweets, err := getAllTweets(limit)
				if err != nil {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Error retrieving tweets: %v", err))
					bot.Send(msg)
					continue
				}

				if len(tweets) == 0 {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "No tweets found in database")
					bot.Send(msg)
					continue
				}

				response := fmt.Sprintf("📊 *Latest %d tweets:*\n\n", len(tweets))
				for i, tweet := range tweets {
					tweetText := tweet.Text
					if len(tweetText) > 100 {
						tweetText = tweetText[:100] + "..."
					}
					response += fmt.Sprintf("*%d.* @%s (%s)\n`%s`\n🕒 %s\n\n",
						i+1,
						tweet.Username,
						tweet.TweetID,
						tweetText,
						tweet.CreatedAt.Format("2006-01-02 15:04:05"),
					)
				}

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
				msg.ParseMode = tgbotapi.ModeMarkdown
				bot.Send(msg)

			case "help":
				helpText := `🤖 *GrutaDetector Bot Commands:*

*Main Commands:*
/help - Show this help message
/set <pages> - Set user search pages (default: 100)
/tweets [limit] - Show parsed tweets (default: 10, max: 100)
/analyze <username> - Analyze user (limited pages + background parsing)
/parse <username> - Parse all user tweets (unlimited pages)

*Background Task Management:*
/status - Show current background task status
/stop - Stop current background parsing job
/clear - Clear the background parsing queue

*Examples:*
• /set 50
• /tweets 20
• /analyze elonmusk
• /parse elonmusk

*Current Settings:*
• Search pages: %d`

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf(helpText, userSearchPages))
				msg.ParseMode = tgbotapi.ModeMarkdown
				bot.Send(msg)

			case "analyze":
				args := update.Message.CommandArguments()
				if args == "" {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Usage: /analyze <username>\nExample: /analyze elonmusk")
					bot.Send(msg)
					continue
				}

				username := strings.TrimSpace(args)
				if strings.HasPrefix(username, "@") {
					username = username[1:]
				}

				go analyzeUserWithProgress(update.Message.Chat.ID, username)

			case "parse":
				args := update.Message.CommandArguments()
				if args == "" {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Usage: /parse <username>\nExample: /parse elonmusk")
					bot.Send(msg)
					continue
				}

				username := strings.TrimSpace(args)
				if strings.HasPrefix(username, "@") {
					username = username[1:]
				}

				go parseUserWithProgress(update.Message.Chat.ID, username)

			case "status":
				queueLen := len(parseQueue)
				var statusText string

				if isWorkerBusy && currentlyParsing != "" {
					tweetCount := getTweetCountByUsername(currentlyParsing)
					statusText = fmt.Sprintf("🔄 *Background Task Status*\n\n*Currently parsing:* @%s\n📊 *Parsed tweets:* %d\n📋 *Queue length:* %d", currentlyParsing, tweetCount, queueLen)
				} else if queueLen > 0 {
					statusText = fmt.Sprintf("⏸️ *Background Task Status*\n\n*Currently parsing:* None\n📋 *Queue length:* %d\n⏳ *Status:* Waiting for next job", queueLen)
				} else {
					statusText = "✅ *Background Task Status*\n\n*Currently parsing:* None\n📋 *Queue length:* 0\n💤 *Status:* Idle"
				}

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, statusText)
				msg.ParseMode = tgbotapi.ModeMarkdown
				bot.Send(msg)

			case "stop":
				if isWorkerBusy && currentlyParsing != "" {
					select {
					case stopCurrentJob <- true:
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("🛑 *Stop Signal Sent*\n\nStopping current parsing job for @%s", currentlyParsing))
						msg.ParseMode = tgbotapi.ModeMarkdown
						bot.Send(msg)
					default:
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "⚠️ *Already Stopping*\n\nStop signal already sent to current job")
						msg.ParseMode = tgbotapi.ModeMarkdown
						bot.Send(msg)
					}
				} else {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "ℹ️ *No Active Job*\n\nNo background parsing job is currently running")
					msg.ParseMode = tgbotapi.ModeMarkdown
					bot.Send(msg)
				}

			case "clear":
				queueLen := len(parseQueue)

				for len(parseQueue) > 0 {
					<-parseQueue
				}

				clearedCount := queueLen
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("🗑️ *Queue Cleared*\n\nRemoved %d items from the background parsing queue", clearedCount))
				msg.ParseMode = tgbotapi.ModeMarkdown
				bot.Send(msg)

			default:
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Hi! %d\n\nUse /help to see all available commands.", update.Message.Chat.ID))
				bot.Send(msg)
			}
		} else {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Hi! %d\n\nUse /help to see all available commands.", update.Message.Chat.ID))
			bot.Send(msg)
		}
	}
}
