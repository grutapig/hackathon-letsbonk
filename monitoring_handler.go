package main

import (
	"github.com/grutapig/hackaton/twitterapi"
	"log"
	"os"
	"time"
)

// MonitoringHandler handles monitoring for new messages in community
func MonitoringHandler(twitterApi *twitterapi.TwitterAPIService, newMessageCh chan twitterapi.NewMessage) {
	defer close(newMessageCh)

	monitoringMethod := os.Getenv(ENV_MONITORING_METHOD)
	if monitoringMethod == "" {
		monitoringMethod = "incremental" // default
	}

	log.Printf("Starting monitoring with method: %s", monitoringMethod)

	if monitoringMethod == "full_scan" {
		MonitoringFullScan(twitterApi, newMessageCh)
	} else {
		MonitoringIncremental(twitterApi, newMessageCh)
	}
}

func MonitoringIncremental(twitterApi *twitterapi.TwitterAPIService, newMessageCh chan twitterapi.NewMessage) {
	// Local storage exists messages, with reply counts
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

		// Fill exists storage; first time we just fill storage
		if len(tweetsExistsStorage) == 0 {
			for _, tweet := range tweetsResponse.Tweets {
				tweetsExistsStorage[tweet.Id] = tweet.ReplyCount
				// Last page is enough for monitoring
				tweetRepliesResponse, err := twitterApi.GetTweetReplies(twitterapi.TweetRepliesRequest{
					TweetID: tweet.Id,
				})
				if err != nil {
					// First step we don't handle any errors, debug is enough
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

		// Start monitoring
		for _, tweet := range tweetsResponse.Tweets {
			SendIfNotExistsTweetToChannel(tweet, []string{}, newMessageCh, tweetsExistsStorage, twitterapi.Tweet{}, twitterapi.Tweet{})
			if tweet.ReplyCount > tweetsExistsStorage[tweet.Id] {
				tweetsExistsStorage[tweet.Id] = tweet.ReplyCount
				// Last page is enough for monitoring
				tweetRepliesResponse, err := twitterApi.GetTweetReplies(twitterapi.TweetRepliesRequest{
					TweetID: tweet.Id,
				})
				if err != nil {
					// First step we don't handle any errors, debug is enough
					log.Printf("error on gettings replies for tweet, ERR: %s, TWEET ID: %s, TEXT: %s, AUTHOR: %s", err, tweet.Id, tweet.Text, tweet.Author.Name)
					continue
				}
				tweets := []string{}
				for _, tweet := range tweetRepliesResponse.Tweets {
					tweets = append(tweets, tweet.Author.UserName+":"+tweet.Text)
				}
				for i, tweetReply := range tweetRepliesResponse.Tweets {
					SendIfNotExistsTweetToChannel(tweetReply, tweets[i:], newMessageCh, tweetsExistsStorage, tweet, twitterapi.Tweet{})
					tweetsExistsStorage[tweetReply.Id] = tweetReply.ReplyCount
				}
			}
			tweetsExistsStorage[tweet.Id] = tweet.ReplyCount
		}
	}
}

func MonitoringFullScan(twitterApi *twitterapi.TwitterAPIService, newMessageCh chan twitterapi.NewMessage) {
	processedReplies := map[string]bool{}
	replyCountStorage := map[string]int{} // Track reply counts like in incremental

	// Initial population - mark all existing messages as processed and save reply counts
	log.Println("Full scan: Initial population of existing messages...")
	InitialPopulationFullScan(twitterApi, processedReplies, replyCountStorage)
	log.Printf("Full scan: Initial population completed, marked %d messages as processed", len(processedReplies))

	for {
		time.Sleep(5 * time.Second) // Longer interval for full scan

		// Get all community posts
		tweetsResponse, err := twitterApi.GetCommunityTweets(twitterapi.CommunityTweetsRequest{
			CommunityID: os.Getenv(ENV_DEMO_COMMUNITY_ID),
		})
		if err != nil {
			log.Printf("Error getting community tweets: %v", err)
			continue
		}

		log.Printf("Full scan: found %d posts", len(tweetsResponse.Tweets))

		// Create mapping of all posts
		postsMapping := make(map[string]twitterapi.Tweet)
		for _, tweet := range tweetsResponse.Tweets {
			postsMapping[tweet.Id] = tweet
		}

		// For each post, get replies
		for _, mainTweet := range tweetsResponse.Tweets {
			tweetRepliesResponse, err := twitterApi.GetTweetReplies(twitterapi.TweetRepliesRequest{
				TweetID: mainTweet.Id,
			})
			if err != nil {
				log.Printf("Error getting replies for tweet %s: %v", mainTweet.Id, err)
				continue
			}

			// Process each reply
			for _, reply := range tweetRepliesResponse.Tweets {
				// Check if this is a new reply
				if !processedReplies[reply.Id] {
					// Mark as processed
					processedReplies[reply.Id] = true

					// Send new reply to channel
					newMessageCh <- twitterapi.NewMessage{
						TweetID:      reply.Id,
						ReplyTweetID: reply.InReplyToId,
						Author: struct {
							UserName string
							Name     string
							ID       string
						}{reply.Author.UserName, reply.Author.Name, reply.Author.Id},
						ParentTweet: struct {
							ID     string
							Author string
							Text   string
						}{ID: mainTweet.Id, Author: mainTweet.Author.UserName, Text: mainTweet.Text},
						GrandParentTweet: struct {
							ID     string
							Author string
							Text   string
						}{}, // Empty for direct replies to main posts
						Text:         reply.Text,
						CreatedAt:    reply.CreatedAt,
						ReplyCount:   reply.ReplyCount,
						LikeCount:    reply.LikeCount,
						RetweetCount: reply.RetweetCount,
						TweetsBefore: []string{},
					}

					log.Printf("Full scan: found new reply from %s to post %s", reply.Author.UserName, mainTweet.Id)
				}

				// Process replies to this reply (deeper level) - only if reply count changed
				if reply.ReplyCount > replyCountStorage[reply.Id] {
					replyCountStorage[reply.Id] = reply.ReplyCount
					ProcessReplyRepliesFullScan(twitterApi, reply, mainTweet, newMessageCh, processedReplies, replyCountStorage)
				}
			}
		}

		// Cleanup old processed replies to prevent memory growth
		if len(processedReplies) > 1000000 {
			log.Println("Cleaning up processed replies cache")
			newProcessedReplies := make(map[string]bool)
			// Keep only last 5000 entries (rough cleanup)
			count := 0
			for id, val := range processedReplies {
				if count >= 5000 {
					break
				}
				newProcessedReplies[id] = val
				count++
			}
			processedReplies = newProcessedReplies
		}
	}
}

// ProcessReplyRepliesFullScan handles replies to replies for full scan monitoring
func ProcessReplyRepliesFullScan(twitterApi *twitterapi.TwitterAPIService, parentReply twitterapi.Tweet, originalTweet twitterapi.Tweet, newMessageCh chan twitterapi.NewMessage, processedReplies map[string]bool, replyCountStorage map[string]int) {
	// Always get replies for full scan (no reply count checking)
	replyRepliesResponse, err := twitterApi.GetTweetReplies(twitterapi.TweetRepliesRequest{
		TweetID: parentReply.Id,
	})
	if err != nil {
		log.Printf("Error getting replies to reply %s: %v", parentReply.Id, err)
		return
	}

	// Process each reply to reply
	for _, replyReply := range replyRepliesResponse.Tweets {
		// Check if this is a new reply
		if !processedReplies[replyReply.Id] {
			// Mark as processed
			processedReplies[replyReply.Id] = true

			// Send reply to reply to channel
			newMessageCh <- twitterapi.NewMessage{
				TweetID:      replyReply.Id,
				ReplyTweetID: replyReply.InReplyToId,
				Author: struct {
					UserName string
					Name     string
					ID       string
				}{replyReply.Author.UserName, replyReply.Author.Name, replyReply.Author.Id},
				ParentTweet: struct {
					ID     string
					Author string
					Text   string
				}{ID: parentReply.Id, Author: parentReply.Author.UserName, Text: parentReply.Text},
				GrandParentTweet: struct {
					ID     string
					Author string
					Text   string
				}{ID: originalTweet.Id, Author: originalTweet.Author.UserName, Text: originalTweet.Text},
				Text:         replyReply.Text,
				CreatedAt:    replyReply.CreatedAt,
				ReplyCount:   replyReply.ReplyCount,
				LikeCount:    replyReply.LikeCount,
				RetweetCount: replyReply.RetweetCount,
				TweetsBefore: []string{},
			}

			log.Printf("Full scan: found reply to reply from %s (depth 2+)", replyReply.Author.UserName)
		}

		// Go deeper if there are more replies (recursive) - only if reply count changed
		if replyReply.ReplyCount > replyCountStorage[replyReply.Id] {
			replyCountStorage[replyReply.Id] = replyReply.ReplyCount
			ProcessReplyRepliesFullScan(twitterApi, replyReply, originalTweet, newMessageCh, processedReplies, replyCountStorage)
		}
	}
}

// InitialPopulationFullScan marks all existing messages as processed during startup
func InitialPopulationFullScan(twitterApi *twitterapi.TwitterAPIService, processedReplies map[string]bool, replyCountStorage map[string]int) {
	// Get all community posts
	tweetsResponse, err := twitterApi.GetCommunityTweets(twitterapi.CommunityTweetsRequest{
		CommunityID: os.Getenv(ENV_DEMO_COMMUNITY_ID),
	})
	if err != nil {
		log.Printf("Error getting community tweets for initial population: %v", err)
		return
	}

	log.Printf("Full scan: Initial population found %d main posts", len(tweetsResponse.Tweets))

	// For each main post, recursively mark all replies as processed
	for _, mainTweet := range tweetsResponse.Tweets {
		// Mark main tweet as processed and save reply count
		processedReplies[mainTweet.Id] = true
		replyCountStorage[mainTweet.Id] = mainTweet.ReplyCount

		// Recursively mark all replies as processed
		MarkRepliesAsProcessedRecursive(twitterApi, mainTweet.Id, processedReplies, replyCountStorage, 0)
	}
}

// MarkRepliesAsProcessedRecursive recursively marks all replies as processed
func MarkRepliesAsProcessedRecursive(twitterApi *twitterapi.TwitterAPIService, tweetId string, processedReplies map[string]bool, replyCountStorage map[string]int, depth int) {
	if depth > 10 { // Prevent infinite recursion
		log.Printf("Max depth reached for tweet %s", tweetId)
		return
	}

	// Get replies to this tweet
	tweetRepliesResponse, err := twitterApi.GetTweetReplies(twitterapi.TweetRepliesRequest{
		TweetID: tweetId,
	})
	if err != nil {
		log.Printf("Error getting replies for tweet %s during initial population: %v", tweetId, err)
		return
	}

	log.Printf("Full scan: Initial population depth %d - found %d replies for tweet %s", depth, len(tweetRepliesResponse.Tweets), tweetId)

	// Mark each reply as processed and save reply count, then recurse deeper
	for _, reply := range tweetRepliesResponse.Tweets {
		processedReplies[reply.Id] = true
		replyCountStorage[reply.Id] = reply.ReplyCount

		// Recurse deeper if this reply has replies
		if reply.ReplyCount > 0 {
			MarkRepliesAsProcessedRecursive(twitterApi, reply.Id, processedReplies, replyCountStorage, depth+1)
		}
	}
}
