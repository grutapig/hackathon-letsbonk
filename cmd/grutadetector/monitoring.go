package main

import (
	"fmt"
	"github.com/grutapig/hackaton/twitterapi_reverse"
	"log"
	"strings"
	"time"
)

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
		return
	}
	replyTweet := lastTweets.Tweets[0]

	for _, chatId := range telegramChatIds {
		go analyzeUserWithProgressForNotification(chatId, replyTweet.Author.UserName, replyTweet.Text, tweet.Text)
	}
}
