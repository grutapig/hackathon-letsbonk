package main

import (
	"context"
	"fmt"
	"github.com/grutapig/hackaton/twitterapi"
	"log"
	"os"
	"sync"
	"time"
)

type TwitterBotService struct {
	twitterAPI      *twitterapi.TwitterAPIService
	claudeAPI       *ClaudeApi
	databaseService *DatabaseService
	botTag          string
	authSession     string
	proxyDsn        string
	knownTweets     map[string]bool
	tweetsMutex     sync.RWMutex
	isMonitoring    bool
	monitoringMutex sync.Mutex
}

func NewTwitterBotService(twitterAPI *twitterapi.TwitterAPIService) *TwitterBotService {
	botTag := os.Getenv(ENV_TWITTER_BOT_TAG)
	if botTag == "" {
		panic("ENV_TWITTER_BOT_TAG environment variable is not set")
	}

	authSession := os.Getenv(ENV_TWITTER_AUTH)
	if authSession == "" {
		panic("ENV_TWITTER_AUTH environment variable is not set")
	}
	proxyDSN := os.Getenv(ENV_PROXY_DSN)
	if authSession == "" {
		panic("ENV_PROXY_DSN environment variable is not set")
	}

	return &TwitterBotService{
		twitterAPI:  twitterAPI,
		botTag:      botTag,
		authSession: authSession,
		proxyDsn:    proxyDSN,
		knownTweets: make(map[string]bool),
	}
}

func (t *TwitterBotService) StartMonitoring(ctx context.Context) error {
	t.monitoringMutex.Lock()
	if t.isMonitoring {
		t.monitoringMutex.Unlock()
		return fmt.Errorf("monitoring is already running")
	}
	t.isMonitoring = true
	t.monitoringMutex.Unlock()

	log.Printf("Starting monitoring for mentions to %s", t.botTag)

	if err := t.initializeKnownTweets(); err != nil {
		log.Printf("Error initializing known tweets: %v", err)
		return err
	}

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.monitoringMutex.Lock()
			t.isMonitoring = false
			t.monitoringMutex.Unlock()
			log.Println("Monitoring stopped")
			return ctx.Err()
		case <-ticker.C:
			log.Println("checking...")
			if err := t.checkForNewTweets(); err != nil {
				log.Printf("Error checking for new tweets: %v", err)
			}
		}
	}
}

func (t *TwitterBotService) initializeKnownTweets() error {
	query := fmt.Sprintf("%s", t.botTag)
	searchRequest := twitterapi.AdvancedSearchRequest{
		Query:     query,
		QueryType: twitterapi.LATEST,
	}

	response, err := t.twitterAPI.AdvancedSearch(searchRequest)
	if err != nil {
		return fmt.Errorf("error in initial search: %w", err)
	}

	t.tweetsMutex.Lock()
	defer t.tweetsMutex.Unlock()

	for _, tweet := range response.Tweets {
		t.knownTweets[tweet.Id] = true
		log.Println(tweet.CreatedAt, tweet.Id, tweet.Text, tweet.Author.UserName)
	}

	log.Printf("Initialized with %d known tweets", len(t.knownTweets))
	return nil
}

func (t *TwitterBotService) checkForNewTweets() error {
	query := fmt.Sprintf("%s", t.botTag)
	searchRequest := twitterapi.AdvancedSearchRequest{
		Query:     query,
		QueryType: twitterapi.LATEST,
	}

	response, err := t.twitterAPI.AdvancedSearch(searchRequest)
	if err != nil {
		return fmt.Errorf("error in search: %w", err)
	}
	log.Printf("%s: found tweets: %d\n", query, len(response.Tweets))

	newTweets := t.findNewTweets(response.Tweets)

	for _, tweet := range newTweets {
		log.Printf("Found new tweet from @%s: %s", tweet.Author.UserName, tweet.Text)
		if err := t.respondToTweet(tweet); err != nil {
			log.Printf("Error responding to tweet %s: %v", tweet.Id, err)
		}
	}

	return nil
}

func (t *TwitterBotService) findNewTweets(tweets []twitterapi.Tweet) []twitterapi.Tweet {
	t.tweetsMutex.Lock()
	defer t.tweetsMutex.Unlock()

	var newTweets []twitterapi.Tweet
	for _, tweet := range tweets {
		if !t.knownTweets[tweet.Id] {
			t.knownTweets[tweet.Id] = true
			newTweets = append(newTweets, tweet)
		}
	}

	return newTweets
}

func (t *TwitterBotService) respondToTweet(tweet twitterapi.Tweet) error {
	responseText := fmt.Sprintf("Hello @%s! Thank you for mentioning me.", tweet.Author.UserName)

	postRequest := twitterapi.PostTweetRequest{
		AuthSession:      t.authSession,
		TweetText:        responseText,
		InReplyToTweetId: tweet.Id,
		Proxy:            t.proxyDsn,
	}

	response, err := t.twitterAPI.PostTweet(postRequest)
	if err != nil {
		return fmt.Errorf("error posting tweet: %w", err)
	}

	log.Printf("Successfully responded to tweet %s with tweet %s", tweet.Id, response.Data.CreateTweet.TweetResult.Result.RestId)
	return nil
}
