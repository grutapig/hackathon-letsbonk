package main

import (
	"fmt"
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/grutapig/hackaton/twitterapi_reverse"
	"log"
	"os"
	"strconv"
	"time"
)

func MonitoringHandler(twitterApi *twitterapi.TwitterAPIService, newMessageCh chan twitterapi.NewMessage, dbService *DatabaseService, loggingService *LoggingService) {
	defer close(newMessageCh)

	var reverseService *twitterapi_reverse.TwitterReverseService
	if enabled, _ := strconv.ParseBool(os.Getenv(ENV_TWITTER_REVERSE_ENABLED)); enabled {
		auth := &twitterapi_reverse.TwitterAuth{
			Authorization: os.Getenv(ENV_TWITTER_REVERSE_AUTHORIZATION),
			XCSRFToken:    os.Getenv(ENV_TWITTER_REVERSE_CSRF_TOKEN),
			Cookie:        os.Getenv(ENV_TWITTER_REVERSE_COOKIE),
		}

		if auth.Authorization != "" && auth.XCSRFToken != "" && auth.Cookie != "" {
			reverseService = twitterapi_reverse.NewTwitterReverseService(auth, os.Getenv(ENV_PROXY_DSN), false)
			log.Println("Twitter Reverse API service initialized")
		} else {
			log.Println("Twitter Reverse API enabled but missing authentication data")
		}
	}

	MonitoringIncremental(twitterApi, newMessageCh, dbService, loggingService, reverseService)
}

func MonitoringIncremental(twitterApi *twitterapi.TwitterAPIService, newMessageCh chan twitterapi.NewMessage, dbService *DatabaseService, loggingService *LoggingService, reverseService *twitterapi_reverse.TwitterReverseService) {

	tweetsExistsStorage := map[string]int{}

	for {
		time.Sleep(30 * time.Second)

		tweets, err := getCommunityTweetsWithFallback(reverseService, twitterApi, os.Getenv(ENV_DEMO_COMMUNITY_ID))
		if err != nil {
			log.Println("Error getting community tweets:", err)
			continue
		}

		tweetsResponse := &twitterapi.CommunityTweetsResponse{
			Tweets: tweets,
		}

		if len(tweetsExistsStorage) == 0 {
			log.Println("First time monitoring initialization...")

			log.Println("Initializing monitoring mapping from 3 pages...")
			InitializeMonitoringMapping(twitterApi, tweetsExistsStorage, reverseService)

			log.Printf("Monitoring initialization completed with %d tweets in storage", len(tweetsExistsStorage))
			continue
		}

		for _, tweet := range tweetsResponse.Tweets {

			storeTweetAndUser(dbService, tweet)

			SendIfNotExistsTweetToChannel(tweet, newMessageCh, tweetsExistsStorage, twitterapi.Tweet{}, twitterapi.Tweet{}, loggingService)
			if tweet.ReplyCount > tweetsExistsStorage[tweet.Id] {
				tweetsExistsStorage[tweet.Id] = tweet.ReplyCount

				tweetRepliesResponse, err := twitterApi.GetTweetReplies(twitterapi.TweetRepliesRequest{
					TweetID: tweet.Id,
				})
				if err != nil {

					log.Printf("error on gettings replies for tweet, ERR: %s, TWEET ID: %s, TEXT: %s, AUTHOR: %s", err, tweet.Id, tweet.Text, tweet.Author.Name)
					continue
				}

				for _, tweetReply := range tweetRepliesResponse.Tweets {

					storeTweetAndUser(dbService, tweetReply)

					var parentTweet, grandParentTweet twitterapi.Tweet
					if tweetReply.InReplyToId != tweet.Id {

						log.Printf("Reply %s is responding to another reply %s, not main post %s", tweetReply.Id, tweetReply.InReplyToId, tweet.Id)

						if dbTweet, err := dbService.GetTweet(tweetReply.InReplyToId); err == nil {
							if dbUser, err := dbService.GetUser(dbTweet.UserID); err == nil {
								parentTweet = twitterapi.Tweet{
									Id:   dbTweet.ID,
									Text: dbTweet.Text,
									Author: twitterapi.Author{
										Id:       dbUser.ID,
										UserName: dbUser.Username,
										Name:     dbUser.Name,
									},
								}
								log.Printf("'%s', Found parent reply in database: %s by %s", tweetReply.Text, parentTweet.Text, parentTweet.Author.UserName)

								grandParentTweet = tweet
							}
						} else {
							log.Printf("Parent reply %s not found in database", tweetReply.InReplyToId)

							parentTweet = tweet
						}
					} else {
						log.Printf("Reply %s is responding to main post %s", tweetReply.Id, tweet.Id)

						parentTweet = tweet
					}

					SendIfNotExistsTweetToChannel(tweetReply, newMessageCh, tweetsExistsStorage, parentTweet, grandParentTweet, loggingService)
					tweetsExistsStorage[tweetReply.Id] = tweetReply.ReplyCount
				}
			}
			tweetsExistsStorage[tweet.Id] = tweet.ReplyCount
		}
	}
}

func storeTweetAndUser(dbService *DatabaseService, tweet twitterapi.Tweet) {

	createdAt, err := time.Parse(time.RFC1123, tweet.CreatedAt)
	if err != nil {

		createdAt, err = time.Parse("Mon Jan 02 15:04:05 -0700 2006", tweet.CreatedAt)
		if err != nil {
			log.Printf("Failed to parse tweet created_at: %v", err)
			createdAt = time.Now()
		}
	}

	user := UserModel{
		ID:       tweet.Author.Id,
		Username: tweet.Author.UserName,
		Name:     tweet.Author.Name,
	}

	if !dbService.UserExists(tweet.Author.Id) {
		err = dbService.SaveUser(user)
		if err != nil {
			log.Printf("Failed to save user %s: %v", tweet.Author.UserName, err)
		}
	}

	tweetModel := TweetModel{
		ID:            tweet.Id,
		Text:          tweet.Text,
		CreatedAt:     createdAt,
		ReplyCount:    tweet.ReplyCount,
		UserID:        tweet.Author.Id,
		Username:      tweet.Author.UserName,
		InReplyToID:   tweet.InReplyToId,
		SourceType:    TWEET_SOURCE_COMMUNITY,
		TickerMention: "",
		SearchQuery:   "",
	}

	err = dbService.SaveTweet(tweetModel)
	if err != nil {
		log.Printf("Failed to save tweet %s: %v", tweet.Id, err)
	}
}

func storeTweetAndUserWithSource(dbService *DatabaseService, tweet twitterapi.Tweet, sourceType, tickerMention, searchQuery string) {

	createdAt, err := time.Parse(time.RFC1123, tweet.CreatedAt)
	if err != nil {

		createdAt, err = time.Parse("Mon Jan 02 15:04:05 -0700 2006", tweet.CreatedAt)
		if err != nil {
			log.Printf("Failed to parse tweet created_at: %v", err)
			createdAt = time.Now()
		}
	}

	user := UserModel{
		ID:       tweet.Author.Id,
		Username: tweet.Author.UserName,
		Name:     tweet.Author.Name,
	}

	if !dbService.UserExists(tweet.Author.Id) {
		err = dbService.SaveUser(user)
		if err != nil {
			log.Printf("Failed to save user %s: %v", tweet.Author.UserName, err)
		}
	}

	tweetModel := TweetModel{
		ID:            tweet.Id,
		Text:          tweet.Text,
		CreatedAt:     createdAt,
		ReplyCount:    tweet.ReplyCount,
		UserID:        tweet.Author.Id,
		InReplyToID:   tweet.InReplyToId,
		SourceType:    sourceType,
		TickerMention: tickerMention,
		SearchQuery:   searchQuery,
	}

	err = dbService.SaveTweet(tweetModel)
	if err != nil {
		log.Printf("Failed to save tweet %s: %v", tweet.Id, err)
	}
}

func InitialCommunityLoad(twitterApi *twitterapi.TwitterAPIService, dbService *DatabaseService) {
	const MAX_PAGES = 3
	cursor := ""
	totalPosts := 0
	totalReplies := 0

	log.Printf("Starting initial community load - fetching %d pages...", MAX_PAGES)

	for page := 0; page < MAX_PAGES; page++ {
		var tweetsResponse *twitterapi.CommunityTweetsResponse
		var err error

		if cursor == "" {
			tweetsResponse, err = twitterApi.GetCommunityTweets(twitterapi.CommunityTweetsRequest{
				CommunityID: os.Getenv(ENV_DEMO_COMMUNITY_ID),
			})
		} else {
			tweetsResponse, err = twitterApi.GetCommunityTweets(twitterapi.CommunityTweetsRequest{
				CommunityID: os.Getenv(ENV_DEMO_COMMUNITY_ID),
				Cursor:      cursor,
			})
		}

		if err != nil {
			log.Printf("Error fetching community tweets page %d: %v", page+1, err)
			break
		}

		if len(tweetsResponse.Tweets) == 0 {
			log.Printf("No more tweets found on page %d", page+1)
			break
		}

		log.Printf("Processing page %d with %d posts...", page+1, len(tweetsResponse.Tweets))

		for _, mainTweet := range tweetsResponse.Tweets {

			storeTweetAndUserWithSource(dbService, mainTweet, TWEET_SOURCE_COMMUNITY, "", "")
			totalPosts++

			repliesCount := LoadAllRepliesRecursive(twitterApi, dbService, mainTweet.Id, 0)
			totalReplies += repliesCount

			log.Printf("Loaded post %s with %d replies", mainTweet.Id, repliesCount)
		}

		if tweetsResponse.NextCursor == "" {
			log.Printf("No more pages available after page %d", page+1)
			break
		}
		cursor = tweetsResponse.NextCursor
	}

	log.Printf("Initial community load completed: %d posts, %d replies loaded", totalPosts, totalReplies)
}

func LoadAllRepliesRecursive(twitterApi *twitterapi.TwitterAPIService, dbService *DatabaseService, tweetID string, depth int) int {
	if depth > 10 {
		log.Printf("Max depth reached for tweet %s", tweetID)
		return 0
	}

	repliesResponse, err := twitterApi.GetTweetReplies(twitterapi.TweetRepliesRequest{
		TweetID: tweetID,
	})
	if err != nil {
		log.Printf("Error getting replies for tweet %s: %v", tweetID, err)
		return 0
	}

	totalReplies := len(repliesResponse.Tweets)

	for _, reply := range repliesResponse.Tweets {

		storeTweetAndUserWithSource(dbService, reply, TWEET_SOURCE_COMMUNITY, "", "")

		nestedReplies := LoadAllRepliesRecursive(twitterApi, dbService, reply.Id, depth+1)
		totalReplies += nestedReplies
	}

	return totalReplies
}

func FullCommunityLoad(twitterApi *twitterapi.TwitterAPIService, dbService *DatabaseService) {
	cursor := ""
	totalPosts := 0
	totalReplies := 0
	pageCount := 0

	log.Printf("Starting FULL community load - fetching ALL pages...")

	for {
		pageCount++
		if pageCount > 5 {
			break
		}
		var tweetsResponse *twitterapi.CommunityTweetsResponse
		var err error

		if cursor == "" {
			tweetsResponse, err = twitterApi.GetCommunityTweets(twitterapi.CommunityTweetsRequest{
				CommunityID: os.Getenv(ENV_DEMO_COMMUNITY_ID),
			})
		} else {
			tweetsResponse, err = twitterApi.GetCommunityTweets(twitterapi.CommunityTweetsRequest{
				CommunityID: os.Getenv(ENV_DEMO_COMMUNITY_ID),
				Cursor:      cursor,
			})
		}

		if err != nil {
			log.Printf("Error fetching community tweets page %d: %v", pageCount, err)
			break
		}

		if len(tweetsResponse.Tweets) == 0 {
			log.Printf("No more tweets found on page %d", pageCount)
			break
		}

		log.Printf("Processing FULL load page %d with %d posts...", pageCount, len(tweetsResponse.Tweets))

		for _, mainTweet := range tweetsResponse.Tweets {

			storeTweetAndUserWithSource(dbService, mainTweet, TWEET_SOURCE_COMMUNITY, "", "")
			totalPosts++

			repliesCount := LoadAllRepliesRecursive(twitterApi, dbService, mainTweet.Id, 0)
			totalReplies += repliesCount

			log.Printf("FULL load: saved post %s with %d replies", mainTweet.Id, repliesCount)
		}

		if tweetsResponse.NextCursor == "" {
			log.Printf("Reached end of community pages after page %d", pageCount)
			break
		}
		cursor = tweetsResponse.NextCursor
	}

	log.Printf("FULL community load completed: %d posts, %d replies loaded across %d pages", totalPosts, totalReplies, pageCount)
}

func InitializeMonitoringMapping(twitterApi *twitterapi.TwitterAPIService, tweetsExistsStorage map[string]int, reverseService *twitterapi_reverse.TwitterReverseService) {
	pageCount := 0
	maxPages := 3

	for pageCount < maxPages {

		tweets, err := getCommunityTweetsWithFallback(reverseService, twitterApi, os.Getenv(ENV_DEMO_COMMUNITY_ID))
		if err != nil {
			log.Printf("Error fetching monitoring mapping page %d: %v", pageCount+1, err)
			break
		}

		if len(tweets) == 0 {
			log.Printf("No tweets found on monitoring mapping page %d", pageCount+1)
			break
		}

		log.Printf("Processing monitoring mapping page %d with %d posts...", pageCount+1, len(tweets))

		for _, tweet := range tweets {
			tweetsExistsStorage[tweet.Id] = tweet.ReplyCount

			tweetRepliesResponse, err := twitterApi.GetTweetReplies(twitterapi.TweetRepliesRequest{
				TweetID: tweet.Id,
			})
			if err != nil {
				log.Printf("Error getting replies for monitoring mapping, tweet %s: %v", tweet.Id, err)
				continue
			}

			for _, tweetReply := range tweetRepliesResponse.Tweets {
				tweetsExistsStorage[tweetReply.Id] = tweetReply.ReplyCount
			}
		}

		pageCount++

		if reverseService != nil {
			log.Println("Reverse API doesn't support pagination yet, stopping after first page")
			break
		}

		break
	}

	log.Printf("Monitoring mapping initialized with %d tweets from %d pages", len(tweetsExistsStorage), pageCount)
}

func getCommunityTweetsWithFallback(reverseService *twitterapi_reverse.TwitterReverseService, twitterApi *twitterapi.TwitterAPIService, communityID string) ([]twitterapi.Tweet, error) {

	if reverseService != nil {
		log.Println("Trying reverse API for community tweets...")
		simpleTweets, err := reverseService.GetCommunityTweets(communityID, 20)
		if err != nil {
			log.Printf("Reverse API failed: %v, falling back to original API", err)
		} else {
			log.Printf("Reverse API success: got %d tweets", len(simpleTweets))
			return convertSimpleTweetsToTweets(simpleTweets), nil
		}
	}

	log.Println("Using original API for community tweets...")
	tweetsResponse, err := twitterApi.GetCommunityTweets(twitterapi.CommunityTweetsRequest{
		CommunityID: communityID,
	})
	if err != nil {
		return nil, err
	}

	return tweetsResponse.Tweets, nil
}

func convertSimpleTweetsToTweets(simpleTweets []twitterapi_reverse.SimpleTweet) []twitterapi.Tweet {
	var tweets []twitterapi.Tweet

	for _, simpleTweet := range simpleTweets {
		var inReplyToId string
		if simpleTweet.ReplyToID != nil {
			inReplyToId = *simpleTweet.ReplyToID
		}

		tweet := twitterapi.Tweet{
			Id:              simpleTweet.TweetID,
			Text:            simpleTweet.Text,
			CreatedAt:       simpleTweet.CreatedAt.Format("Mon Jan 02 15:04:05 -0700 2006"),
			CreatedAtParsed: simpleTweet.CreatedAt,
			ReplyCount:      simpleTweet.RepliesCount,
			InReplyToId:     inReplyToId,
			Author: twitterapi.Author{
				Id:       simpleTweet.Author.ID,
				UserName: simpleTweet.Author.Username,
				Name:     simpleTweet.Author.Name,
			},
		}
		fmt.Println("tweet:", tweet.Id, tweet.Text, tweet.Author.Id, tweet.Author.UserName, tweet.Author.Name, "reply", tweet.ReplyCount)
		tweets = append(tweets, tweet)
	}

	return tweets
}
