package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/grutapig/hackaton/twitterapi"
)

type TwitterBotService struct {
	twitterAPI      *twitterapi.TwitterAPIService
	claudeAPI       *ClaudeApi
	databaseService *DatabaseService
	botTag          string
	authSession     string
}

func NewTwitterBotService(twitterAPI *twitterapi.TwitterAPIService, claudeAPI *ClaudeApi, databaseService *DatabaseService) *TwitterBotService {
	botTag := os.Getenv(ENV_TWITTER_BOT_TAG)
	if botTag == "" {
		panic("ENV_TWITTER_BOT_TAG environment variable is not set")
	}

	authSession := os.Getenv(ENV_TWITTER_AUTH)
	if authSession == "" {
		panic("ENV_TWITTER_AUTH environment variable is not set")
	}

	return &TwitterBotService{
		twitterAPI:      twitterAPI,
		claudeAPI:       claudeAPI,
		databaseService: databaseService,
		botTag:          botTag,
		authSession:     authSession,
	}
}

func (s *TwitterBotService) ProcessMentionTweet(message twitterapi.NewMessage) {
	log.Printf("Processing mention tweet from @%s: %s", message.Author.UserName, message.Text)

	// First Claude request to analyze the message
	analysisPrompt := fmt.Sprintf("Analyze this tweet for questions or requests: %s", message.Text)

	analysisResponse, err := s.claudeAPI.SendRequest(analysisPrompt, "")
	if err != nil {
		log.Printf("Failed to analyze tweet: %v", err)
		return
	}

	log.Printf("Analysis response: %s", analysisResponse)

	// Check if user is in FUD database
	var fudUser *FUDUser
	users, err := s.databaseService.SearchFUDUsers(message.Author.UserName, "", 1)
	if err == nil && len(users) > 0 {
		fudUser = &users[0]
	}

	// Prepare context for response generation
	contextInfo := ""
	if fudUser != nil {
		contextInfo = fmt.Sprintf("User @%s is marked as FUD user with type: %s", message.Author.UserName, fudUser.FUDType)
	} else {
		contextInfo = fmt.Sprintf("User @%s is not in FUD database", message.Author.UserName)
	}

	// Second Claude request for response generation
	responsePrompt := fmt.Sprintf(`Generate a response to this tweet. Context: %s
	
Tweet analysis: %s
Original tweet: %s

Respond in 180 characters or less. Do not use JSON format, just plain text.`, contextInfo, analysisResponse, message.Text)

	responseText, err := s.claudeAPI.SendRequest(responsePrompt, "")
	if err != nil {
		log.Printf("Failed to generate response: %v", err)
		return
	}

	// Limit response to 180 characters
	if len(responseText) > 180 {
		responseText = responseText[:180]
	}

	log.Printf("Generated response: %s", responseText)

	// Post reply tweet
	err = s.PostReplyTweet(responseText, message.TweetID)
	if err != nil {
		log.Printf("Failed to post reply tweet: %v", err)
	} else {
		log.Printf("Successfully posted reply to tweet %s", message.TweetID)
	}
}

func (s *TwitterBotService) PostReplyTweet(text, replyToID string) error {
	request := twitterapi.PostTweetRequest{
		AuthSession:      s.authSession,
		TweetText:        text,
		InReplyToTweetId: replyToID,
		Proxy:            os.Getenv(ENV_PROXY_DSN),
	}

	response, err := s.twitterAPI.PostTweet(request)
	if err != nil {
		return fmt.Errorf("failed to post tweet: %w", err)
	}

	log.Printf("Posted tweet with ID: %s", response.TweetID)
	return nil
}

func (s *TwitterBotService) ContainsBotTag(text string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(s.botTag))
}

func (s *TwitterBotService) StartMentionListener(newMessageCh chan twitterapi.NewMessage) {
	log.Printf("Starting mention listener for bot tag: %s", s.botTag)

	// Track processed tweets to avoid duplicates
	processedTweets := make(map[string]bool)

	for message := range newMessageCh {
		// Skip if already processed
		if processedTweets[message.TweetID] {
			continue
		}

		// Check if tweet contains bot tag
		if s.ContainsBotTag(message.Text) {
			log.Printf("Found mention in tweet %s from @%s", message.TweetID, message.Author.UserName)
			go s.ProcessMentionTweet(message)
			processedTweets[message.TweetID] = true
		}

		// Clean up old processed tweets to prevent memory growth
		if len(processedTweets) > 1000 {
			for tweetID := range processedTweets {
				delete(processedTweets, tweetID)
				if len(processedTweets) <= 500 {
					break
				}
			}
		}
	}
}
