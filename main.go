package main

import (
	"encoding/json"
	"fmt"
	"github.com/grutapig/hackaton/twitterapi"
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
	twitterApi := twitterapi.NewTwitterAPIService(os.Getenv(ENV_TWITTER_API_KEY), os.Getenv(ENV_TWITTER_API_BASE_URL), os.Getenv(ENV_PROXY_DSN))
	notificationFormatter := NewNotificationFormatter()
	telegramService, err := NewTelegramService(os.Getenv(ENV_TELEGRAM_API_KEY), os.Getenv(ENV_PROXY_DSN), os.Getenv(ENV_TELEGRAM_ADMIN_CHAT_ID), notificationFormatter)
	if err != nil {
		panic(err)
	}

	// Start Telegram service
	telegramService.StartListening()

	systemPromptFirstStep, err := os.ReadFile(PROMPT_FILE_STEP1)
	if err != nil {
		panic(err)
	}
	systemPromptSecondStep, err := os.ReadFile(PROMPT_FILE_STEP2)
	if err != nil {
		panic(err)
	}
	//init channels
	newMessageCh := make(chan twitterapi.NewMessage, 10)
	//flag channel is for second step
	flagCh := make(chan twitterapi.NewMessage, 10)
	//notification channel
	notificationCh := make(chan FUDAlertNotification, 10)

	//start monitoring for new messages in community
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(newMessageCh)
		//local storage exists messages, with reply counts
		tweetsExistsStorage := map[string]int{}
		for {
			time.Sleep(1 * time.Second)
			tweetsResponse, err := twitterApi.GetCommunityTweets(twitterapi.CommunityTweetsRequest{
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
					tweetRepliesResponse, err := twitterApi.GetTweetReplies(twitterapi.TweetRepliesRequest{
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
					tweetRepliesResponse, err := twitterApi.GetTweetReplies(twitterapi.TweetRepliesRequest{
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
							tweets = append(tweets, tweet.Author.UserName+":"+tweet.Text)
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
			log.Println("Got a new message:", newMessage.Author.UserName, " - ", newMessage.Text)
			messages := ClaudeMessages{}
			for _, text := range newMessage.TweetsBefore {
				messages = append(messages, ClaudeMessage{ROLE_USER, text})
			}
			messages = append(messages, ClaudeMessage{ROLE_USER, newMessage.Author.UserName + ":" + newMessage.Text})
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
			lastMessages, err := twitterApi.GetUserLastTweets(twitterapi.UserLastTweetsRequest{UserId: newMessage.Author.ID})
			followers, err := twitterApi.GetUserFollowers(twitterapi.UserFollowersRequest{UserName: newMessage.Author.UserName})
			followings, err := twitterApi.GetUserFollowings(twitterapi.UserFollowingsRequest{UserName: newMessage.Author.UserName})
			threadMessages, err := twitterApi.GetTweetReplies(twitterapi.TweetRepliesRequest{TweetID: newMessage.ReplyTweetID})
			postMessage, err := twitterApi.GetTweetsByIds([]string{newMessage.ReplyTweetID})
			//prepare claude request
			claudeMessages := PrepareClaudeSecondStepRequest(lastMessages, followers, followings, threadMessages, postMessage)
			resp, err := claudeApi.SendMessage(claudeMessages, string(systemPromptSecondStep)+"analyzed user is "+newMessage.Author.UserName)
			aiDecision2 := SecondStepClaudeResponse{}
			fmt.Println("claude make a decision for this user:", resp, err)
			err = json.Unmarshal([]byte("{"+resp.Content[0].Text), &aiDecision2)
			fmt.Println(aiDecision2)
			if aiDecision2.IsFUDMessage {
				// Create FUD alert notification
				alert := FUDAlertNotification{
					FUDMessageID:      newMessage.TweetID,
					FUDUserID:         newMessage.Author.ID,
					FUDUsername:       newMessage.Author.UserName,
					ThreadID:          newMessage.ReplyTweetID,
					DetectedAt:        time.Now().Format(time.RFC3339),
					AlertSeverity:     notificationFormatter.mapRiskLevelToSeverity(aiDecision2.UserRiskLevel),
					FUDType:           aiDecision2.FUDType,
					FUDProbability:    aiDecision2.FUDProbability,
					MessagePreview:    newMessage.Text,
					RecommendedAction: notificationFormatter.getRecommendedAction(aiDecision2),
					KeyEvidence:       aiDecision2.KeyEvidence,
					DecisionReason:    aiDecision2.DecisionReason,
				}
				notificationCh <- alert
			}

		}
	}()
	//notification handler
	wg.Add(1)
	go func() {
		defer wg.Done()
		for alert := range notificationCh {
			log.Printf("FUD Alert: %s (@%s) - %s", alert.FUDType, alert.FUDUsername, alert.AlertSeverity)

			// Store and broadcast notification with detail command
			err := telegramService.StoreAndBroadcastNotification(alert)
			if err != nil {
				log.Printf("Failed to send Telegram notification: %v", err)
			}
		}
	}()
	wg.Wait()
}

func PrepareClaudeSecondStepRequest(lastMessages *twitterapi.UserLastTweetsResponse, followers *twitterapi.UserFollowersResponse, followings *twitterapi.UserFollowingsResponse, threadMessages *twitterapi.TweetRepliesResponse, postMessage *twitterapi.TweetsByIdsResponse) ClaudeMessages {
	claudeMessages := ClaudeMessages{}

	// 1. User's messages from personal page
	if lastMessages != nil && len(lastMessages.Data.Tweets) > 0 {
		userMessages := make([]string, 0)
		for _, tweet := range lastMessages.Data.Tweets {
			userMessages = append(userMessages, tweet.CreatedAt+" - "+tweet.Text)
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
			message = append(message, fmt.Sprintf("@%s: %s (%s)", tweet.Author.UserName, tweet.Text, tweet.CreatedAt))
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
	claudeMessages = append(claudeMessages, ClaudeMessage{
		Role:    ROLE_ASSISTANT,
		Content: "{",
	})
	return claudeMessages
}

type FirstStepClaudeResponse struct {
	IsFlag          bool   `json:"is_flag"`
	FlagProbability int    `json:"flag_probability"`
	Reason          string `json:"reason"`
}
type SecondStepClaudeResponse struct {
	IsFUDMessage   bool     `json:"is_fud_message"`
	IsFUDUser      bool     `json:"is_fud_user"`
	FUDProbability float64  `json:"fud_probability"` // 0.0 - 1.0
	FUDType        string   `json:"fud_type"`        // "professional_trojan_horse", "professional_direct_attack", "professional_statistical", "emotional_escalation", "emotional_dramatic_exit", "casual_criticism", "none"
	UserRiskLevel  string   `json:"user_risk_level"` // "critical", "high", "medium", "low"
	KeyEvidence    []string `json:"key_evidence"`    // 2-4 most important evidence points
	DecisionReason string   `json:"decision_reason"` // 1-2 sentence summary of why this decision was made
}

func SendNewTweetToChannel(tweet twitterapi.Tweet, tweetsBefore []string, newMessageCh chan twitterapi.NewMessage, tweetsExistsStorage map[string]int) {
	if _, ok := tweetsExistsStorage[tweet.Id]; !ok {
		newMessageCh <- twitterapi.NewMessage{
			TweetID:      tweet.Id,
			ReplyTweetID: tweet.InReplyToId,
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
