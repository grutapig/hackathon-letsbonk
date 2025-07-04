package main

import (
	"github.com/grutapig/hackaton/twitterapi"
	"log"
	"os"
	"time"
)

// MonitoringHandler handles monitoring for new messages in community
func MonitoringHandler(twitterApi *twitterapi.TwitterAPIService, newMessageCh chan twitterapi.NewMessage, dbService *DatabaseService) {
	defer close(newMessageCh)

	MonitoringIncremental(twitterApi, newMessageCh, dbService)
}

func MonitoringIncremental(twitterApi *twitterapi.TwitterAPIService, newMessageCh chan twitterapi.NewMessage, dbService *DatabaseService) {
	// Local storage exists messages, with reply counts
	tweetsExistsStorage := map[string]int{}

	for {
		time.Sleep(30 * time.Second)
		tweetsResponse, err := twitterApi.GetCommunityTweets(twitterapi.CommunityTweetsRequest{
			CommunityID: os.Getenv(ENV_DEMO_COMMUNITY_ID),
		})
		if err != nil {
			log.Println(err)
			continue
		}

		// First time initialization
		if len(tweetsExistsStorage) == 0 {
			log.Println("First time monitoring initialization...")

			// Initialize mapping from 3 pages for monitoring
			log.Println("Initializing monitoring mapping from 3 pages...")
			InitializeMonitoringMapping(twitterApi, tweetsExistsStorage)

			log.Printf("Monitoring initialization completed with %d tweets in storage", len(tweetsExistsStorage))
			continue
		}

		// Start monitoring
		for _, tweet := range tweetsResponse.Tweets {
			// Store tweet and user data
			storeTweetAndUser(dbService, tweet)

			SendIfNotExistsTweetToChannel(tweet, newMessageCh, tweetsExistsStorage, twitterapi.Tweet{}, twitterapi.Tweet{})
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

				for _, tweetReply := range tweetRepliesResponse.Tweets {
					// Store reply tweet and user data
					storeTweetAndUser(dbService, tweetReply)

					// Check if this reply is responding to another reply (not the main post)
					var parentTweet, grandParentTweet twitterapi.Tweet
					if tweetReply.InReplyToId != tweet.Id {
						// This is a reply to another reply, not to the main post
						log.Printf("Reply %s is responding to another reply %s, not main post %s", tweetReply.Id, tweetReply.InReplyToId, tweet.Id)

						// Try to find the immediate parent in database
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

								// Set the main post as grandparent
								grandParentTweet = tweet
							}
						} else {
							log.Printf("Parent reply %s not found in database", tweetReply.InReplyToId)
							// Fallback: use main post as parent
							parentTweet = tweet
						}
					} else {
						log.Printf("Reply %s is responding to main post %s", tweetReply.Id, tweet.Id)
						// This is a direct reply to the main post
						parentTweet = tweet
					}

					SendIfNotExistsTweetToChannel(tweetReply, newMessageCh, tweetsExistsStorage, parentTweet, grandParentTweet)
					tweetsExistsStorage[tweetReply.Id] = tweetReply.ReplyCount
				}
			}
			tweetsExistsStorage[tweet.Id] = tweet.ReplyCount
		}
	}
}

func storeTweetAndUser(dbService *DatabaseService, tweet twitterapi.Tweet) {
	// Parse created_at time
	createdAt, err := time.Parse(time.RFC1123, tweet.CreatedAt)
	if err != nil {
		// Try alternative time format if RFC1123 fails
		createdAt, err = time.Parse("Mon Jan 02 15:04:05 -0700 2006", tweet.CreatedAt)
		if err != nil {
			log.Printf("Failed to parse tweet created_at: %v", err)
			createdAt = time.Now() // Fallback to current time
		}
	}

	// Store user
	user := UserModel{
		ID:       tweet.Author.Id,
		Username: tweet.Author.UserName,
		Name:     tweet.Author.Name,
	}

	// Only store if user doesn't exist (to avoid overwriting existing data)
	if !dbService.UserExists(tweet.Author.Id) {
		err = dbService.SaveUser(user)
		if err != nil {
			log.Printf("Failed to save user %s: %v", tweet.Author.UserName, err)
		}
	}

	// Store tweet with default community source
	tweetModel := TweetModel{
		ID:            tweet.Id,
		Text:          tweet.Text,
		CreatedAt:     createdAt,
		ReplyCount:    tweet.ReplyCount,
		UserID:        tweet.Author.Id,
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
	// Parse created_at time
	createdAt, err := time.Parse(time.RFC1123, tweet.CreatedAt)
	if err != nil {
		// Try alternative time format if RFC1123 fails
		createdAt, err = time.Parse("Mon Jan 02 15:04:05 -0700 2006", tweet.CreatedAt)
		if err != nil {
			log.Printf("Failed to parse tweet created_at: %v", err)
			createdAt = time.Now() // Fallback to current time
		}
	}

	// Store user
	user := UserModel{
		ID:       tweet.Author.Id,
		Username: tweet.Author.UserName,
		Name:     tweet.Author.Name,
	}

	// Only store if user doesn't exist (to avoid overwriting existing data)
	if !dbService.UserExists(tweet.Author.Id) {
		err = dbService.SaveUser(user)
		if err != nil {
			log.Printf("Failed to save user %s: %v", tweet.Author.UserName, err)
		}
	}

	// Store tweet with source information
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

		// Process each main post
		for _, mainTweet := range tweetsResponse.Tweets {
			// Save main post
			storeTweetAndUserWithSource(dbService, mainTweet, TWEET_SOURCE_COMMUNITY, "", "")
			totalPosts++

			// Get all replies for this post recursively
			repliesCount := LoadAllRepliesRecursive(twitterApi, dbService, mainTweet.Id, 0)
			totalReplies += repliesCount

			log.Printf("Loaded post %s with %d replies", mainTweet.Id, repliesCount)
		}

		// Check for next page
		if tweetsResponse.NextCursor == "" {
			log.Printf("No more pages available after page %d", page+1)
			break
		}
		cursor = tweetsResponse.NextCursor
	}

	log.Printf("Initial community load completed: %d posts, %d replies loaded", totalPosts, totalReplies)
}

func LoadAllRepliesRecursive(twitterApi *twitterapi.TwitterAPIService, dbService *DatabaseService, tweetID string, depth int) int {
	if depth > 10 { // Prevent infinite recursion
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
		// Save reply
		storeTweetAndUserWithSource(dbService, reply, TWEET_SOURCE_COMMUNITY, "", "")

		// Recursively load replies to this reply
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

		// Process each main post
		for _, mainTweet := range tweetsResponse.Tweets {
			// Save main post
			storeTweetAndUserWithSource(dbService, mainTweet, TWEET_SOURCE_COMMUNITY, "", "")
			totalPosts++

			// Get all replies for this post recursively
			repliesCount := LoadAllRepliesRecursive(twitterApi, dbService, mainTweet.Id, 0)
			totalReplies += repliesCount

			log.Printf("FULL load: saved post %s with %d replies", mainTweet.Id, repliesCount)
		}

		// Check for next page
		if tweetsResponse.NextCursor == "" {
			log.Printf("Reached end of community pages after page %d", pageCount)
			break
		}
		cursor = tweetsResponse.NextCursor
	}

	log.Printf("FULL community load completed: %d posts, %d replies loaded across %d pages", totalPosts, totalReplies, pageCount)
}

// InitializeMonitoringMapping initializes the monitoring storage with tweets from 3 pages (for tracking new messages)
func InitializeMonitoringMapping(twitterApi *twitterapi.TwitterAPIService, tweetsExistsStorage map[string]int) {
	cursor := ""
	pageCount := 0
	maxPages := 3

	for pageCount < maxPages {
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
			log.Printf("Error fetching monitoring mapping page %d: %v", pageCount+1, err)
			break
		}

		if len(tweetsResponse.Tweets) == 0 {
			log.Printf("No tweets found on monitoring mapping page %d", pageCount+1)
			break
		}

		log.Printf("Processing monitoring mapping page %d with %d posts...", pageCount+1, len(tweetsResponse.Tweets))

		for _, tweet := range tweetsResponse.Tweets {
			tweetsExistsStorage[tweet.Id] = tweet.ReplyCount

			// Get replies for monitoring mapping
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

		// Check for next page
		if tweetsResponse.NextCursor == "" {
			log.Printf("No more pages available for monitoring mapping after page %d", pageCount)
			break
		}
		cursor = tweetsResponse.NextCursor
	}

	log.Printf("Monitoring mapping initialized with %d tweets from %d pages", len(tweetsExistsStorage), pageCount)
}
