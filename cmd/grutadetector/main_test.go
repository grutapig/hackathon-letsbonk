package main

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/grutapig/hackaton/claude"
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/grutapig/hackaton/twitterapi_reverse"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

type Tweet struct {
	Id   string
	Text string
	Date string
}

func TestGetHistory(t *testing.T) {
	godotenv.Load()
	ts := twitterapi.NewTwitterAPIService(os.Getenv("twitter_api_key"), "https://api.twitterapi.io", os.Getenv("proxy_dsn"))
	username := "a1lon9"
	var tweets []Tweet
	var cursor string
	for i := 0; i < 100; i++ {
		resp, err := ts.AdvancedSearch(twitterapi.AdvancedSearchRequest{
			Query:     "from:" + username,
			QueryType: twitterapi.LATEST,
			Cursor:    cursor,
		})
		if err != nil || len(resp.Tweets) == 0 || resp.NextCursor == "" {
			fmt.Println("no more, break", err)
			break
		}
		cursor = resp.NextCursor
		for n, tweet := range resp.Tweets {
			element := Tweet{Id: tweet.Id, Text: tweet.Text, Date: tweet.CreatedAt}
			tweets = append(tweets, element)
			fmt.Println(i, n, element)
		}
		fmt.Println("found tweet: ", len(resp.Tweets))
	}
	data, err := json.Marshal(tweets)
	assert.NoError(t, err)
	os.WriteFile("a1lon9.json", data, 0655)
}
func TestAnalyzeHistory(t *testing.T) {
	godotenv.Load()
	data, err := os.ReadFile("a1lon9.json")
	assert.NoError(t, err)
	var tweets []Tweet
	err = json.Unmarshal(data, &tweets)
	assert.NoError(t, err)
	claudeApi, err := claude.NewClaudeClient(os.Getenv("claude_api_key"), os.Getenv("proxy_dsn"), claude.CLAUDE_MODEL)
	assert.NoError(t, err)
	//prepare short messages list
	messages := []string{}
	for _, tweet := range tweets {
		messages = append(messages, tweet.Text)
	}
	data, _ = json.Marshal(messages)
	claudeMessages := claude.ClaudeMessages{
		{
			Role:    claude.ROLE_USER,
			Content: "history in json:" + string(data),
		},
		{
			Role:    claude.ROLE_USER,
			Content: "last message: I bought many DARk coins",
		},
	}
	response, err := claudeApi.SendMessage(claudeMessages, "You are lier detector, you have to check all user messages history, and detect lie in the last message if you can.")
	assert.NoError(t, err)
	fmt.Println(response)
	os.WriteFile("claude.txt", []byte(response.Content[0].Text), 0655)
}
func TestSendTelegramNotify(t *testing.T) {
	godotenv.Load()
	data, _ := os.ReadFile("claude.txt")
	bot, err := tgbotapi.NewBotAPI(os.Getenv("telegram_api_key"))
	assert.NoError(t, err)
	notifyChatIds := []int64{8188194753, 47109854, 6616342769}
	for _, chatId := range notifyChatIds {
		msg := tgbotapi.NewMessage(chatId, string(data))
		msg.ParseMode = tgbotapi.ModeMarkdown
		resp, err := bot.Send(msg)
		fmt.Println(resp, err)
	}
}

func TestSendTweet(t *testing.T) {
	godotenv.Load()
	api := twitterapi.NewTwitterAPIService(os.Getenv("twitter_api_key"), "https://api.twitterapi.io", os.Getenv(twitterapi.ENV_PROXY_DSN))

	postTweetResponse, err := api.PostTweet(twitterapi.PostTweetRequest{
		AuthSession: os.Getenv(twitterapi.ENV_TWITTER_AUTH),
		TweetText: `hi all!
`,
		QuoteTweetId:     "",
		InReplyToTweetId: "",
		MediaId:          "",
		Proxy:            os.Getenv(twitterapi.ENV_PROXY_DSN),
	})
	fmt.Println(postTweetResponse, err)
}
func TestMonitoring(t *testing.T) {
	godotenv.Load()
	auth := twitterapi_reverse.NewTwitterAuth(os.Getenv(twitterapi_reverse.ENV_TWITTER_REVERSE_AUTHORIZATION), os.Getenv(twitterapi_reverse.ENV_TWITTER_REVERSE_CSRF_TOKEN), os.Getenv(twitterapi_reverse.ENV_TWITTER_REVERSE_COOKIE))
	service := twitterapi_reverse.NewTwitterReverseApi(auth, os.Getenv(twitterapi.ENV_PROXY_DSN), false)
	tweets, err := service.GetNotificationsSimple()
	assert.NoError(t, err)
	currentBotname := os.Getenv("twitter_bot_tag")
	for i, tweet := range tweets {
		if tweet.Author.Username != currentBotname {
			fmt.Println(i, "|", tweet.Text, "|", tweet.CreatedAt, "|", tweet.Author.Username, "| reply:", tweet.ReplyToID)
		}
	}
}
