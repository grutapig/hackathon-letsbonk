package main

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/grutapig/hackaton/claude"
	"github.com/grutapig/hackaton/twitterapi_reverse"
	"log"
	"os"
	"time"
)

var userParsingStatus = make(map[string]bool)

func checkUserFullyParsed(username string) bool {
	return userParsingStatus[username]
}

func setUserFullyParsed(username string, status bool) {
	userParsingStatus[username] = status
}

func analyzeUserWithProgress(chatID int64, username string) {
	initialMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("🔍 Starting analysis for @%s...", username))
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
	var tweets []twitterapi_reverse.SimpleTweet
	var userFullyParsed bool

	if initialCount >= 1500 {
		updateProgress(fmt.Sprintf("🔍 *Analyzing @%s*\n\n💾 Using cached tweets from database (%d tweets)\n⏳ Loading tweets...", username, initialCount))
		tweets, err = getTweetsFromDB(username, userSearchPages*20)
		userFullyParsed = checkUserFullyParsed(username)
	} else {
		updateProgress(fmt.Sprintf("🔍 *Analyzing @%s*\n\n⏳ Sending to infinite parsing queue...\n📊 Current tweets: %d", username, initialCount))

		select {
		case parseQueue <- username:
			log.Printf("Queued infinite parsing for @%s", username)
		default:
			log.Printf("Parse queue full, cannot queue parsing for @%s", username)
			updateProgress(fmt.Sprintf("🔍 *Analysis Failed*\n\n❌ Parse queue is full, try again later"))
			return
		}

		progressTicker := time.NewTicker(10 * time.Second)
		done := make(chan bool)

		go func() {
			for {
				select {
				case <-progressTicker.C:
					currentCount := getTweetCountByUsername(username)
					fullyParsed := checkUserFullyParsed(username)
					if currentCount >= 1500 || fullyParsed {
						done <- true
						return
					}
					updateProgress(fmt.Sprintf("🔍 *Analyzing @%s*\n\n⏳ Waiting for parsing to complete...\n📊 Current tweets: %d\n🔄 Parsing in progress...", username, currentCount))
				case <-done:
					progressTicker.Stop()
					return
				}
			}
		}()

		<-done

		finalCount := getTweetCountByUsername(username)
		userFullyParsed = checkUserFullyParsed(username)
		if finalCount >= 1500 || userFullyParsed {
			updateProgress(fmt.Sprintf("🔍 *Analyzing @%s*\n\n✅ Sufficient tweets collected (%d tweets)\n⏳ Loading tweets for analysis...", username, finalCount))
			tweets, err = getTweetsFromDB(username, userSearchPages*20)
		} else {
			updateProgress(fmt.Sprintf("🔍 *Analysis Failed*\n\n❌ Could not collect enough tweets for @%s\n📊 Current tweets: %d (need 1500 minimum)", username, finalCount))
			return
		}
	}

	if err != nil {
		updateProgress(fmt.Sprintf("🔍 *Analysis Failed*\n\n❌ Error fetching tweets for @%s: %v", username, err))
		return
	}

	finalCount := getTweetCountByUsername(username)
	if len(tweets) == 0 {
		updateProgress(fmt.Sprintf("🔍 *Analysis Complete*\n\n⚠️ No tweets found for @%s\n📊 Total parsed tweets: %d", username, finalCount))
		return
	}

	sourceText := fmt.Sprintf("✅ Found %d tweets\n📊 Total parsed tweets: %d", len(tweets), finalCount)

	updateProgress(fmt.Sprintf("🔍 *Analyzing @%s*\n\n%s\n⏳ Preparing data for analysis...", username, sourceText))

	messages := [][2]string{}
	for _, tweet := range tweets {
		messages = append(messages, [2]string{tweet.Text, tweet.CreatedAt.Format(time.RFC3339)})
	}

	updateProgress(fmt.Sprintf("🔍 *Analyzing @%s*\n\n%s\n✅ Data prepared\n⏳ Sending to Claude AI...", username, sourceText))

	data, _ := json.Marshal(messages)
	claudeMessages := claude.ClaudeMessages{
		{
			Role:    claude.ROLE_USER,
			Content: "history in json:" + string(data),
		},
		{
			Role:    claude.ROLE_USER,
			Content: "analyze this user behavior and provide insights",
		},
	}

	prompt, err := os.ReadFile("prompt.txt")
	if err != nil {
		updateProgress(fmt.Sprintf("🔍 *Analysis Failed*\n\n❌ Cannot read prompt.txt: %v", err))
		return
	}

	updateProgress(fmt.Sprintf("🔍 *Analyzing @%s*\n\n%s\n✅ Data prepared\n✅ Prompt loaded\n⏳ Processing with Claude AI...", username, sourceText))

	response, err := claudeApi.SendMessage(claudeMessages, string(prompt))
	if err != nil {
		updateProgress(fmt.Sprintf("🔍 *Analysis Failed*\n\n❌ Claude AI error for @%s: %v", username, err))
		return
	}

	var backgroundText string
	if userFullyParsed {
		backgroundText = "\n\n✅ *User fully parsed - all tweets collected*"
	} else {
		backgroundText = "\n\n🔄 *Background parsing continues...*"
	}

	analysisResult := fmt.Sprintf("🎯 *Analysis Complete for @%s*\n\n📊 *Tweets analyzed:* %d\n📊 *Total parsed tweets:* %d\n📅 *From:* %s\n📅 *To:* %s\n\n🤖 *Claude Analysis:*\n%s%s",
		username,
		len(tweets),
		finalCount,
		tweets[len(tweets)-1].CreatedAt.Format("2006-01-02 15:04"),
		tweets[0].CreatedAt.Format("2006-01-02 15:04"),
		response.Content[0].Text,
		backgroundText,
	)

	updateProgress(analysisResult)
}

func analyzeUserWithProgressForNotification(chatID int64, username string, lastMessage string, question string) {
	initialMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("🔍 Starting analysis for @%s from notification...", username))
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
	var tweets []twitterapi_reverse.SimpleTweet
	var userFullyParsed bool

	if initialCount >= 1500 {
		updateProgress(fmt.Sprintf("🔍 *Analyzing @%s* (from notification)\n\n💾 Using cached tweets from database (%d tweets)\n⏳ Loading tweets...", username, initialCount))
		tweets, err = getTweetsFromDB(username, userSearchPages*20)
		userFullyParsed = checkUserFullyParsed(username)
	} else {
		updateProgress(fmt.Sprintf("🔍 *Analyzing @%s* (from notification)\n\n⏳ Sending to infinite parsing queue...\n📊 Current tweets: %d", username, initialCount))

		select {
		case parseQueue <- username:
			log.Printf("Queued infinite parsing for @%s", username)
		default:
			log.Printf("Parse queue full, cannot queue parsing for @%s", username)
			updateProgress(fmt.Sprintf("🔍 *Analysis Failed*\n\n❌ Parse queue is full, try again later"))
			return
		}

		progressTicker := time.NewTicker(10 * time.Second)
		done := make(chan bool)

		go func() {
			for {
				select {
				case <-progressTicker.C:
					currentCount := getTweetCountByUsername(username)
					fullyParsed := checkUserFullyParsed(username)
					if currentCount >= 1500 || fullyParsed {
						done <- true
						return
					}
					updateProgress(fmt.Sprintf("🔍 *Analyzing @%s* (from notification)\n\n⏳ Waiting for parsing to complete...\n📊 Current tweets: %d\n🔄 Parsing in progress...", username, currentCount))
				case <-done:
					progressTicker.Stop()
					return
				}
			}
		}()

		<-done

		finalCount := getTweetCountByUsername(username)
		userFullyParsed = checkUserFullyParsed(username)
		if finalCount >= 1500 || userFullyParsed {
			updateProgress(fmt.Sprintf("🔍 *Analyzing @%s* (from notification)\n\n✅ Sufficient tweets collected (%d tweets)\n⏳ Loading tweets for analysis...", username, finalCount))
			tweets, err = getTweetsFromDB(username, userSearchPages*20)
		} else {
			updateProgress(fmt.Sprintf("🔍 *Analysis Failed*\n\n❌ Could not collect enough tweets for @%s\n📊 Current tweets: %d (need 1500 minimum)", username, finalCount))
			return
		}
	}

	if err != nil {
		updateProgress(fmt.Sprintf("🔍 *Analysis Failed*\n\n❌ Error fetching tweets for @%s: %v", username, err))
		return
	}

	finalCount := getTweetCountByUsername(username)
	if len(tweets) == 0 {
		updateProgress(fmt.Sprintf("🔍 *Analysis Complete*\n\n⚠️ No tweets found for @%s\n📊 Total parsed tweets: %d", username, finalCount))
		return
	}

	sourceText := fmt.Sprintf("✅ Found %d tweets\n📊 Total parsed tweets: %d", len(tweets), finalCount)

	updateProgress(fmt.Sprintf("🔍 *Analyzing @%s* (from notification)\n\n%s\n⏳ Preparing data for analysis...", username, sourceText))

	messages := [][2]string{}
	for _, tweet := range tweets {
		messages = append(messages, [2]string{tweet.Text, tweet.CreatedAt.Format(time.RFC3339)})
	}

	updateProgress(fmt.Sprintf("🔍 *Analyzing @%s* (from notification)\n\n%s\n✅ Data prepared\n⏳ Sending to Claude AI...", username, sourceText))

	data, _ := json.Marshal(messages)
	claudeMessages := claude.ClaudeMessages{
		{
			Role:    claude.ROLE_USER,
			Content: "history in json:" + string(data),
		},
		{
			Role:    claude.ROLE_USER,
			Content: "last message: " + lastMessage,
		},
		{
			Role:    claude.ROLE_USER,
			Content: "moderator question on last message: " + question,
		},
	}

	prompt, err := os.ReadFile("prompt.txt")
	if err != nil {
		updateProgress(fmt.Sprintf("🔍 *Analysis Failed*\n\n❌ Cannot read prompt.txt: %v", err))
		return
	}

	updateProgress(fmt.Sprintf("🔍 *Analyzing @%s* (from notification)\n\n%s\n✅ Data prepared\n✅ Prompt loaded\n⏳ Processing with Claude AI...", username, sourceText))

	response, err := claudeApi.SendMessage(claudeMessages, string(prompt))
	if err != nil {
		updateProgress(fmt.Sprintf("🔍 *Analysis Failed*\n\n❌ Claude AI error for @%s: %v", username, err))
		return
	}

	analysisResult := fmt.Sprintf("🎯 *Analysis Complete for @%s* (from notification)\n\n📊 *Tweets analyzed:* %d\n📊 *Total parsed tweets: %d\n📅 *From:* %s\n📅 *To:* %s\n\n🤖 *Claude Analysis:*\n%s",
		username,
		len(tweets),
		finalCount,
		tweets[len(tweets)-1].CreatedAt.Format("2006-01-02 15:04"),
		tweets[0].CreatedAt.Format("2006-01-02 15:04"),
		response.Content[0].Text,
	)

	updateProgress(analysisResult)
}
