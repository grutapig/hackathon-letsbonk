package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/grutapig/hackaton/claude"
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/grutapig/hackaton/twitterapi_reverse"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var (
	bot                 *tgbotapi.BotAPI
	twitterApi          *twitterapi.TwitterAPIService
	twitterReverseAPI   *twitterapi_reverse.TwitterReverseService
	claudeApi           *claude.ClaudeApi
	db                  *gorm.DB
	lastNotificationIDs = make(map[string]bool)
	userSearchPages     = 100
	currentBotName      string
	parseQueue          = make(chan string, 100)
	stopCurrentJob      = make(chan bool, 1)
	currentlyParsing    string
	isWorkerBusy        bool

	telegramChatIds []int64
	twitterBotTag   string
	twitterAuth     string
	proxyDSN        string
)

func loadConfig() {
	godotenv.Load()

	chatIdsStr := os.Getenv("tg_admin_chat_id")
	if chatIdsStr != "" {
		ids := strings.Split(chatIdsStr, ",")
		for _, id := range ids {
			if chatId, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64); err == nil {
				telegramChatIds = append(telegramChatIds, chatId)
			}
		}
	}

	twitterBotTag = os.Getenv("twitter_bot_tag")
	twitterAuth = os.Getenv("twitter_auth")
	proxyDSN = os.Getenv("proxy_dsn")
}

func parseUserWithProgress(chatID int64, username string) {
	initialMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("📥 Starting unlimited parsing for @%s...", username))
	sentMsg, err := bot.Send(initialMsg)
	if err != nil {
		log.Printf("Error sending initial message: %v", err)
		return
	}

	updateProgress := func(text string) {
		editMsg := tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, text)
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(editMsg)
	}

	initialCount := getTweetCountByUsername(username)
	updateProgress(fmt.Sprintf("📥 *Parsing @%s* (unlimited)\n\n⏳ Fetching all user tweets...\n📊 Parsed tweets: %d", username, initialCount))

	progressTicker := time.NewTicker(10 * time.Second)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-progressTicker.C:
				currentCount := getTweetCountByUsername(username)
				updateProgress(fmt.Sprintf("📥 *Parsing @%s* (unlimited)\n\n⏳ Fetching all user tweets...\n📊 Parsed tweets: %d", username, currentCount))
			case <-done:
				progressTicker.Stop()
				return
			}
		}
	}()

	getUserTweetsUnlimited(username)
	done <- true

	finalCount := getTweetCountByUsername(username)
	updateProgress(fmt.Sprintf("✅ *Parsing Complete for @%s*\n\n📊 Total parsed tweets: %d\n🎉 All available tweets have been saved to database!", username, finalCount))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	loadConfig()

	err := initServices()
	if err != nil {
		log.Fatal("Failed to initialize services:", err)
	}

	log.Println("Starting notification monitoring...")

	go handleTelegramUpdates()
	go backgroundParseWorker()
	monitorNotifications()
}
