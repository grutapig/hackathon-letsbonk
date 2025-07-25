package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/grutapig/hackaton/claude"
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/grutapig/hackaton/twitterapi_reverse"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"strings"
)

func initServices() error {
	var err error

	db, err = gorm.Open(sqlite.Open("grutadetector.db"), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&Tweet{})
	if err != nil {
		return fmt.Errorf("failed to migrate database: %v", err)
	}

	bot, err = tgbotapi.NewBotAPI(os.Getenv("telegram_api_key"))
	if err != nil {
		return err
	}

	twitterApi = twitterapi.NewTwitterAPIService(os.Getenv("twitter_api_key"), "https://api.twitterapi.io", proxyDSN)

	auth := twitterapi_reverse.NewTwitterAuth(os.Getenv(twitterapi_reverse.ENV_TWITTER_REVERSE_AUTHORIZATION), os.Getenv(twitterapi_reverse.ENV_TWITTER_REVERSE_CSRF_TOKEN), os.Getenv(twitterapi_reverse.ENV_TWITTER_REVERSE_COOKIE))
	twitterReverseAPI = twitterapi_reverse.NewTwitterReverseApi(auth, os.Getenv(twitterapi.ENV_PROXY_DSN), false)

	claudeApi, err = claude.NewClaudeClient(os.Getenv("claude_api_key"), proxyDSN, claude.CLAUDE_MODEL)
	currentBotName = strings.ToLower(os.Getenv("twitter_bot_tag"))
	return err
}
