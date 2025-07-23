package main

import (
	"encoding/json"
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
)

var (
	bot                 *tgbotapi.BotAPI
	twitterApi          *twitterapi.TwitterAPIService
	twitterReverseAPI   *twitterapi_reverse.TwitterReverseService
	claudeApi           *claude.ClaudeApi
	lastNotificationIDs = make(map[string]bool)
	userSearchPages     = 100
	currentBotName      string

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

func initServices() error {
	var err error

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

func sendTelegramMessage(message string) {
	for _, chatId := range telegramChatIds {
		msg := tgbotapi.NewMessage(chatId, message)
		msg.ParseMode = tgbotapi.ModeMarkdown

		_, err := bot.Send(msg)
		if err != nil {
			errorMsg := fmt.Sprintf("Error: %s... - %v", message[:min(5, len(message))], err)
			retryMsg := tgbotapi.NewMessage(chatId, errorMsg)
			bot.Send(retryMsg)
		}
	}
}

func getUserTweets(username string) ([]twitterapi_reverse.SimpleTweet, error) {
	var tweets []twitterapi_reverse.SimpleTweet
	var cursor string

	for i := 0; i < userSearchPages; i++ {
		resp, err := twitterApi.AdvancedSearch(twitterapi.AdvancedSearchRequest{
			Query:     "from:" + username,
			QueryType: twitterapi.LATEST,
			Cursor:    cursor,
		})

		if err != nil || len(resp.Tweets) == 0 || resp.NextCursor == "" {
			break
		}

		cursor = resp.NextCursor
		for _, tweet := range resp.Tweets {
			twitterTime, _ := twitterapi_reverse.ParseTwitterTime(tweet.CreatedAt)
			tweets = append(tweets, twitterapi_reverse.SimpleTweet{
				TweetID:   tweet.Id,
				Text:      tweet.Text,
				CreatedAt: twitterTime,
			})
		}
	}
	cursor = ""
	if len(tweets) == 0 {
		for i := 0; i < userSearchPages; i++ {
			resp, err := twitterApi.AdvancedSearch(twitterapi.AdvancedSearchRequest{
				Query:     "from:" + username,
				QueryType: twitterapi.TOP,
				Cursor:    cursor,
			})

			if err != nil || len(resp.Tweets) == 0 || resp.NextCursor == "" {
				break
			}

			cursor = resp.NextCursor
			for _, tweet := range resp.Tweets {
				twitterTime, _ := twitterapi_reverse.ParseTwitterTime(tweet.CreatedAt)
				tweets = append(tweets, twitterapi_reverse.SimpleTweet{
					TweetID:   tweet.Id,
					Text:      tweet.Text,
					CreatedAt: twitterTime,
				})
			}
		}
	}

	return tweets, nil
}

func analyzeUser(username string, lastMessage string, question string) (string, error) {
	tweets, err := getUserTweets(username)
	if err != nil {
		return "", err
	}

	messages := [][2]string{}
	for _, tweet := range tweets {
		messages = append(messages, [2]string{tweet.Text, tweet.CreatedAt.Format(time.RFC3339)})
	}

	if len(messages) == 0 {
		sendTelegramMessage(fmt.Sprintln("no history found, skip this", len(messages), username, question))
		return "", fmt.Errorf("no messages history found for this user")
	}
	sendTelegramMessage(fmt.Sprintf("found history count %d, from %s to %s", len(tweets), tweets[0].CreatedAt.Format(time.RFC3339), tweets[len(tweets)-1].CreatedAt.Format(time.RFC3339)))

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
		return "", fmt.Errorf("cannot read prompt.txt, err: %s", err)
	}
	response, err := claudeApi.SendMessage(claudeMessages, string(prompt))
	if err != nil {
		return "", err
	}

	return response.Content[0].Text, nil
}

func postTweet(text string, replyId string) error {
	if len(text) > 280 {
		text = text[:277] + "..."
	}

	_, err := twitterApi.PostTweet(twitterapi.PostTweetRequest{
		AuthSession:      twitterAuth,
		TweetText:        text,
		QuoteTweetId:     "",
		InReplyToTweetId: replyId,
		MediaId:          "",
		Proxy:            proxyDSN,
	})

	return err
}
func processNotification(tweet twitterapi_reverse.SimpleTweet) {
	sendTelegramMessage(fmt.Sprintf("New notification from @%s: %s (%s)", tweet.Author.Username, tweet.Text, tweet.TweetID))
	if tweet.ReplyToID == "" {
		sendTelegramMessage(fmt.Sprintf("No replies found, ignored: %s", tweet.TweetID))
		return
	}
	if strings.ToLower(tweet.ReplyToUsername) == currentBotName {
		sendTelegramMessage(fmt.Sprintf("Reply on myself ignored: %s", tweet.TweetID))
		return
	}
	lastTweets, err := twitterApi.GetTweetsByIds([]string{tweet.ReplyToID})
	if err != nil || len(lastTweets.Tweets) == 0 {
		fmt.Println("cannot get reply tweet", err)
		sendTelegramMessage(fmt.Sprintf("error on get GetTweetsByIds reply, or empty list returned: %s(%s), err: %s", tweet.TweetID, tweet.ReplyToID, err))
	}
	replyTweet := lastTweets.Tweets[0]

	analysis, err := analyzeUser(replyTweet.Author.UserName, replyTweet.Text, tweet.Text)
	if err != nil {
		log.Printf("cannot analyze user, err: %s, tweetId: %s, replyId: %s\n", err, tweet.TweetID, tweet.ReplyToID)
		sendTelegramMessage(fmt.Sprintf("cannot analyze user, err: %s, tweetId: %s, replyId: %s", err, tweet.TweetID, tweet.ReplyToID))
		return
	}

	//err = postTweet(analysis, tweet.TweetID)
	if err != nil {
		log.Printf("Error posting tweet: %v", err)
		sendTelegramMessage(fmt.Sprintf("Error posting tweet: %v, tweet: %s", err, tweet.TweetID))
	}

	sendTelegramMessage(fmt.Sprintf("Analysis for @%s:\n", analysis))
	return
}

func handleTelegramUpdates() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if strings.HasPrefix(update.Message.Text, "/set ") {
			parts := strings.Split(update.Message.Text, " ")
			if len(parts) == 2 {
				if pages, err := strconv.Atoi(parts[1]); err == nil && pages > 0 {
					userSearchPages = pages
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Set user search pages to: %d", pages))
					bot.Send(msg)
				} else {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Invalid number. Usage: /set 20")
					bot.Send(msg)
				}
			}
		}
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Hi! %d", update.Message.Chat.ID))
		bot.Send(msg)
	}
}

func monitorNotifications() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	tweets, _ := twitterReverseAPI.GetNotificationsSimple()
	for _, tweet := range tweets {
		lastNotificationIDs[tweet.TweetID] = true
	}
	fmt.Printf("started, %d\n", len(lastNotificationIDs))
	for range ticker.C {
		tweets, err := twitterReverseAPI.GetNotificationsSimple()
		fmt.Println("checking...")
		if err != nil {
			log.Printf("Error getting notifications: %v", err)
			continue
		}

		for _, tweet := range tweets {
			if strings.ToLower(tweet.Author.Username) == twitterBotTag {
				continue
			}

			if !lastNotificationIDs[tweet.TweetID] {
				lastNotificationIDs[tweet.TweetID] = true

				go processNotification(tweet)
			}
		}
	}
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
	monitorNotifications()
}
