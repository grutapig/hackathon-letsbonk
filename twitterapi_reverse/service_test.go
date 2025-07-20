package twitterapi_reverse

import (
	"fmt"
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestTwitterReverseService_GetTweetDetail(t *testing.T) {
	t.Skip()
	godotenv.Load("../.dev.env")

	curlExample := `curl 'https://x.com/i/api/graphql/-0WTL1e9Pij-JWAF5ztCCA/TweetDetail' -H 'authorization: Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA' -H 'x-csrf-token: 51dba875c98111e3adb18449d1812e64bdb0ad803a9d55db7b7538e5bf4a9582d67cad7c709beacb40af100c47f3c2740525cd4116e89f6d90bb86a0858fde79c5bcc2d9c505455b7e992c9347f2132a' -H 'Cookie: auth_token=0323aebf3f717be2f83e09aa74af7959bb0cc93d; ct0=51dba875c98111e3adb18449d1812e64bdb0ad803a9d55db7b7538e5bf4a9582d67cad7c709beacb40af100c47f3c2740525cd4116e89f6d90bb86a0858fde79c5bcc2d9c505455b7e992c9347f2132a'`

	auth, err := ParseFromCurl(curlExample)
	if err != nil {
		fmt.Printf("Failed to parse curl: %v\n", err)
		return
	}

	fmt.Printf("Parsed auth - Authorization: %s\n", auth.Authorization[:50]+"...")
	fmt.Printf("Parsed auth - CSRF Token: %s\n", auth.XCSRFToken[:50]+"...")
	fmt.Printf("Parsed auth - Cookie: %s\n", auth.Cookie[:50]+"...")

	service := NewTwitterReverseApi(auth, os.Getenv("proxy_dsn"), true)

	tweetID := "1940098840440578176"
	tweet, err := service.GetTweetDetail(tweetID)
	if err != nil {
		fmt.Printf("Error getting tweet detail: %v\n", err)
		return
	}

	fmt.Printf("Tweet ID: %s\n", tweet.TweetID)
	fmt.Printf("Author: @%s (%s)\n", tweet.Author.Username, tweet.Author.Name)
	fmt.Printf("Text: %s\n", tweet.Text)
	fmt.Printf("Created At: %s\n", tweet.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Replies Count: %d\n", tweet.RepliesCount)
	if tweet.ReplyToID != "" {
		fmt.Printf("Reply To ID: %s\n", tweet.ReplyToID)
	} else {
		fmt.Printf("Reply To ID: none\n")
	}
}

func TestTwitterReverseService_GetCommunityTweets(t *testing.T) {
	t.Skip()
	godotenv.Load("../.dev.env")

	headers := map[string]string{
		"authorization": "Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA",
		"x-csrf-token":  "07169b33678da5112c0a1988e3b0423b1cbe32e39bbc9764b7234f190fb5838d5ec1aefa29df3285bebbbb9fd2f9177c4a6a0f4783e45e32d8bd9aeafed5b3a531eb57a52829ea1f174d3e5c313ffb35",
		"cookie":        "auth_token=47ec273f768b8b4cc97d694a02b632b2d30c95a2; ct0=07169b33678da5112c0a1988e3b0423b1cbe32e39bbc9764b7234f190fb5838d5ec1aefa29df3285bebbbb9fd2f9177c4a6a0f4783e45e32d8bd9aeafed5b3a531eb57a52829ea1f174d3e5c313ffb35",
	}

	auth, err := ParseFromHeaders(headers)
	if err != nil {
		fmt.Printf("Failed to parse headers: %v\n", err)
		return
	}

	fmt.Printf("Parsed from headers - Authorization: %s\n", auth.Authorization[:50]+"...")
	fmt.Printf("Parsed from headers - CSRF Token: %s\n", auth.XCSRFToken[:50]+"...")
	fmt.Printf("Parsed from headers - Cookie: %s\n", auth.Cookie[:50]+"...")

	service := NewTwitterReverseApi(auth, os.Getenv("proxy_dsn"), true)

	communityID := "1914102634241577036"
	tweets, err := service.GetCommunityTweets(communityID, 10)
	if err != nil {
		fmt.Printf("Error getting community tweets: %v\n", err)
		return
	}

	fmt.Printf("Found %d tweets in community\n", len(tweets))
	for i, tweet := range tweets {
		fmt.Printf("[%d] @%s (%s) - ID: %s\n", i+1, tweet.Author.Username, tweet.Author.Name, tweet.TweetID)
		fmt.Printf("    Text: %s\n", tweet.Text)
		fmt.Printf("    Created: %s, Replies: %d\n", tweet.CreatedAt.Format("2006-01-02 15:04:05"), tweet.RepliesCount)
		if tweet.ReplyToID != "" {
			fmt.Printf("    Reply to: %s\n", tweet.ReplyToID)
		}
		fmt.Printf("    ---\n")
	}
}
func TestTwitterReverseService_GetNotifications(t *testing.T) {
	godotenv.Load("../.env")
	auth := NewTwitterAuth(os.Getenv(ENV_TWITTER_REVERSE_AUTHORIZATION), os.Getenv(ENV_TWITTER_REVERSE_CSRF_TOKEN), os.Getenv(ENV_TWITTER_REVERSE_COOKIE))
	service := NewTwitterReverseApi(auth, os.Getenv(twitterapi.ENV_PROXY_DSN), false)
	tweets, err := service.GetNotificationsSimple()
	assert.NoError(t, err)
	for i, tweet := range tweets {
		fmt.Println(i, "|", tweet.Text, "|", tweet.CreatedAt, "|", tweet.Author.Username, "| reply:", tweet.ReplyToID)
	}
}
