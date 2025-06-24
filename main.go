package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"sync"
	"time"
)

const ENV_PROD_CONFIG = ".env"
const ENV_DEV_CONFIG = ".dev.env"
const PROMPT_FILE_STEP1 = "prompt1.txt"

func main() {
	err := godotenv.Load(ENV_DEV_CONFIG)
	if err != nil {
		panic(err)
	}
	claudeApi, err := NewClaudeClient(os.Getenv(ENV_CLAUDE_API_KEY), os.Getenv(ENV_PROXY_CLAUDE_DSN), CLAUDE_MODEL)
	if err != nil {
		panic(err)
	}
	twitterApi := NewTwitterAPIService(os.Getenv(ENV_TWITTER_API_KEY), os.Getenv(ENV_TWITTER_API_BASE_URL), os.Getenv(ENV_PROXY_DSN))
	systemPromptFirstStep, err := os.ReadFile(PROMPT_FILE_STEP1)
	if err != nil {
		panic(err)
	}
	//init channels
	newMessageCh := make(chan NewMessage, 10)

	//start monitoring for new messages in community
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(newMessageCh)
		//local storage exists messages, with reply counts
		tweetsExistsStorage := map[string]int{}
		for {
			time.Sleep(10 * time.Second)
			tweetsResponse, err := twitterApi.GetCommunityTweets(CommunityTweetsRequest{
				CommunityID: os.Getenv(ENV_DEMO_COMMUNITY_ID),
			})
			if err != nil {
				log.Println(err)
				continue
			}
			//region fill exists storage; first time we just fill storage
			if len(tweetsExistsStorage) == 0 {
				for _, tweet := range tweetsResponse.Tweets {
					tweetsExistsStorage[tweet.Id] = tweet.ReplyCount
					//last page is enough for monitoring
					tweetRepliesResponse, err := twitterApi.GetTweetReplies(TweetRepliesRequest{
						TweetID: tweet.Id,
					})
					if err != nil {
						//first step we don't handle any errors, debug is enough
						log.Printf("error on gettings replies for tweet, ERR: %s, TWEET ID: %s, TEXT: %s, AUTHOR: %s", err, tweet.Id, tweet.Text, tweet.Author.Name)
						continue
					}
					for _, tweetReply := range tweetRepliesResponse.Tweets {
						tweetsExistsStorage[tweetReply.Id] = tweetReply.ReplyCount
					}
				}
				log.Println("filled storage", len(tweetsExistsStorage))
				continue
			}
			//endregion
			//region start monitoring
			for _, tweet := range tweetsResponse.Tweets {
				SendNewTweetToChannel(tweet, newMessageCh, tweetsExistsStorage)
				if tweet.ReplyCount > tweetsExistsStorage[tweet.Id] {
					tweetsExistsStorage[tweet.Id] = tweet.ReplyCount
					//last page is enough for monitoring
					tweetRepliesResponse, err := twitterApi.GetTweetReplies(TweetRepliesRequest{
						TweetID: tweet.Id,
					})
					if err != nil {
						//first step we don't handle any errors, debug is enough
						log.Printf("error on gettings replies for tweet, ERR: %s, TWEET ID: %s, TEXT: %s, AUTHOR: %s", err, tweet.Id, tweet.Text, tweet.Author.Name)
						continue
					}
					for _, tweetReply := range tweetRepliesResponse.Tweets {
						SendNewTweetToChannel(tweetReply, newMessageCh, tweetsExistsStorage)
						tweetsExistsStorage[tweetReply.Id] = tweetReply.ReplyCount
					}
				}
				tweetsExistsStorage[tweet.Id] = tweet.ReplyCount
			}
		}
	}()
	//handle new message first step
	wg.Add(1)
	go func() {
		defer wg.Done()
		for newMessage := range newMessageCh {
			resp, err := claudeApi.SendMessage(ClaudeMessages{{ROLE_USER, newMessage.Text}}, string(systemPromptFirstStep))
			fmt.Println(resp, err)
		}
	}()
	wg.Wait()
}
func SendNewTweetToChannel(tweet Tweet, newMessageCh chan NewMessage, tweetsExistsStorage map[string]int) {
	if _, ok := tweetsExistsStorage[tweet.Id]; !ok {
		newMessageCh <- NewMessage{
			MessageType: TWITTER_MESSAGE_TYPE_POST,
			TweetID:     tweet.Id,
			Author: struct {
				UserName string
				Name     string
				ID       string
			}{tweet.Author.UserName, tweet.Author.Name, tweet.Author.Id},
			Text:         tweet.Text,
			CreatedAt:    tweet.CreatedAt,
			ReplyCount:   tweet.ReplyCount,
			LikeCount:    tweet.LikeCount,
			RetweetCount: tweet.RetweetCount,
		}
	}
}
