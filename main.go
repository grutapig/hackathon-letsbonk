package main

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const ENV_PROD_CONFIG = ".env"
const ENV_DEV_CONFIG = ".dev.env"
const PROMPT_FILE_STEP1 = "prompt1.txt"
const PROMPT_FILE_STEP2 = "prompt2.txt"

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
	systemPromptSecondStep, err := os.ReadFile(PROMPT_FILE_STEP2)
	if err != nil {
		panic(err)
	}
	//init channels
	newMessageCh := make(chan NewMessage, 10)
	//flag channel is for second step
	flagCh := make(chan NewMessage, 10)

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
				SendNewTweetToChannel(tweet, []string{}, newMessageCh, tweetsExistsStorage)
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
					tweets := []string{}
					for _, tweet := range tweetRepliesResponse.Tweets {
						{
							tweets = append(tweets, tweet.Text)
						}
					}
					for i, tweetReply := range tweetRepliesResponse.Tweets {
						SendNewTweetToChannel(tweetReply, tweets[i:], newMessageCh, tweetsExistsStorage)
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
			messages := ClaudeMessages{}
			for _, text := range newMessage.TweetsBefore {
				messages = append(messages, ClaudeMessage{ROLE_USER, text})
			}
			messages = append(messages, ClaudeMessage{ROLE_USER, newMessage.Text})
			messages = append(messages, ClaudeMessage{ROLE_ASSISTANT, "{"})
			resp, err := claudeApi.SendMessage(messages, string(systemPromptFirstStep))
			if err != nil {
				log.Printf("error claude: %s", err)
				continue
			}
			fmt.Println(resp.Content[0].Text)
			aiDecision := FirstStepClaudeResponse{}
			err = json.Unmarshal([]byte("{"+resp.Content[0].Text), &aiDecision)
			fmt.Println(aiDecision)
			if aiDecision.IsFlag {
				flagCh <- newMessage
			}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for newMessage := range flagCh {
			lastMessages, err := twitterApi.GetUserLastTweets(UserLastTweetsRequest{UserId: newMessage.Author.ID})
			followers, err := twitterApi.GetUserFollowers(UserFollowersRequest{UserName: newMessage.Author.UserName})
			followings, err := twitterApi.GetUserFollowings(UserFollowingsRequest{UserName: newMessage.Author.UserName})
			threadMessages, err := twitterApi.GetTweetReplies(TweetRepliesRequest{TweetID: newMessage.ReplyTweetID})
			postMessage, err := twitterApi.GetTweetsByIds([]string{newMessage.ReplyTweetID})
			//prepare claude request
			claudeMessages := PrepareClaudeSecondStepRequest(lastMessages, followers, followings, threadMessages, postMessage)
			resp, err := claudeApi.SendMessage(claudeMessages, string(systemPromptSecondStep)+"analyzed user is "+newMessage.Author.UserName)
			fmt.Println("claude make a decision for this user:", resp, err)
		}
	}()
	wg.Wait()
}

func PrepareClaudeSecondStepRequest(lastMessages *UserLastTweetsResponse, followers *UserFollowersResponse, followings *UserFollowingsResponse, threadMessages *TweetRepliesResponse, postMessage *TweetsByIdsResponse) ClaudeMessages {
	claudeMessages := ClaudeMessages{}

	// 1. User's messages from personal page
	if lastMessages != nil && len(lastMessages.Data.Tweets) > 0 {
		userMessages := make([]string, 0)
		for _, tweet := range lastMessages.Data.Tweets {
			userMessages = append(userMessages, tweet.Text)
		}
		userMessagesJSON, _ := json.Marshal(userMessages)
		claudeMessages = append(claudeMessages, ClaudeMessage{
			Role:    ROLE_USER,
			Content: fmt.Sprintf("USER'S MESSAGES FROM PERSONAL PAGE:\n%s", string(userMessagesJSON)),
		})
	} else {
		claudeMessages = append(claudeMessages, ClaudeMessage{
			Role:    ROLE_USER,
			Content: "USER'S MESSAGES FROM PERSONAL PAGE: No messages",
		})
	}

	// 2. All user's friends (usernames only)
	allFriends := make([]string, 0)
	if followers != nil {
		for _, follower := range followers.Followers {
			allFriends = append(allFriends, follower.UserName)
		}
	}
	if followings != nil {
		for _, following := range followings.Followings {
			allFriends = append(allFriends, following.UserName)
		}
	}

	if len(allFriends) > 0 {
		friendsJSON, _ := json.Marshal(allFriends)
		claudeMessages = append(claudeMessages, ClaudeMessage{
			Role:    ROLE_USER,
			Content: fmt.Sprintf("ALL USER'S FRIENDS (USERNAMES):\n%s", string(friendsJSON)),
		})
	} else {
		claudeMessages = append(claudeMessages, ClaudeMessage{
			Role:    ROLE_USER,
			Content: "ALL USER'S FRIENDS (USERNAMES): No friends",
		})
	}

	// 3. All messages in analyzed thread (with authors)
	if threadMessages != nil && len(threadMessages.Tweets) > 0 {
		message := []string{}
		for _, tweet := range threadMessages.Tweets {
			message = append(message, fmt.Sprintf("Author: %s\nText: %s", tweet.Author.UserName, tweet.Text))
		}
		claudeMessages = append(claudeMessages, ClaudeMessage{
			Role:    ROLE_USER,
			Content: fmt.Sprintf("ALL MESSAGES IN ANALYZED THREAD:\n%s", strings.Join(message, "\n")),
		})
	} else {
		claudeMessages = append(claudeMessages, ClaudeMessage{
			Role:    ROLE_USER,
			Content: "ALL MESSAGES IN ANALYZED THREAD: No messages in thread",
		})
	}

	// 4. Original thread post text with author
	if postMessage != nil && len(postMessage.Tweets) > 0 {
		originalTweet := postMessage.Tweets[0]
		originalPost := fmt.Sprintf("Author: %s\nText: %s", originalTweet.Author.UserName, originalTweet.Text)
		claudeMessages = append(claudeMessages, ClaudeMessage{
			Role:    ROLE_USER,
			Content: fmt.Sprintf("ORIGINAL THREAD POST:\n%s", originalPost),
		})
	} else {
		claudeMessages = append(claudeMessages, ClaudeMessage{
			Role:    ROLE_USER,
			Content: "ORIGINAL THREAD POST: Not found",
		})
	}

	return claudeMessages
}

type FirstStepClaudeResponse struct {
	IsFlag          bool   `json:"is_flag"`
	FlagProbability int    `json:"flag_probability"`
	Reason          string `json:"reason"`
}

func SendNewTweetToChannel(tweet Tweet, tweetsBefore []string, newMessageCh chan NewMessage, tweetsExistsStorage map[string]int) {
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
			TweetsBefore: tweetsBefore,
		}
	}
}
