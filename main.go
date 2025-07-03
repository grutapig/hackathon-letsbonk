package main

import (
	"encoding/json"
	"fmt"
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/joho/godotenv"
	"log"
	"os"
	"sync"
)

const ENV_PROD_CONFIG = ".env"
const ENV_DEV_CONFIG = ".dev.env"
const PROMPT_FILE_STEP1 = "prompt1.txt"
const PROMPT_FILE_STEP2 = "prompt2.txt"

// initializeData handles initial data loading based on environment configuration
func initializeData(dbService *DatabaseService, twitterApi *twitterapi.TwitterAPIService) {
	// Check if CSV import is requested
	csvPath := os.Getenv(ENV_IMPORT_CSV_PATH)
	if csvPath != "" {
		log.Printf("CSV import path specified: %s", csvPath)

		// Import from CSV instead of full community load
		importer := NewCSVImporter(dbService)
		result, err := importer.ImportCSV(csvPath)
		if err != nil {
			log.Printf("CSV import failed: %v", err)
			log.Println("Falling back to community data loading...")
		} else {
			log.Printf("CSV import successful: %s", result.String())
			return // Skip community loading if CSV import was successful
		}
	}

	// Check if full community data loading is needed
	tweetCount, err := dbService.GetTweetCount()
	if err != nil {
		log.Printf("Error getting tweet count: %v", err)
		tweetCount = 0
	}

	if tweetCount < 10 {
		log.Printf("Tweet count (%d) is less than 10, performing full community load...", tweetCount)
		FullCommunityLoad(twitterApi, dbService)
	} else {
		log.Printf("Tweet count (%d) is >= 10, skipping full database initialization", tweetCount)
	}
}

func main() {
	err := godotenv.Load(ENV_DEV_CONFIG)
	if err != nil {
		panic(err)
	}
	claudeApi, err := NewClaudeClient(os.Getenv(ENV_CLAUDE_API_KEY), os.Getenv(ENV_PROXY_CLAUDE_DSN), CLAUDE_MODEL)
	if err != nil {
		panic(err)
	}
	ticker := os.Getenv(ENV_TWITTER_COMMUNITY_TICKER)
	if ticker == "" {
		panic("ticker should be set .env: " + ENV_TWITTER_COMMUNITY_TICKER)
	}
	twitterApi := twitterapi.NewTwitterAPIService(os.Getenv(ENV_TWITTER_API_KEY), os.Getenv(ENV_TWITTER_API_BASE_URL), os.Getenv(ENV_PROXY_DSN))
	notificationFormatter := NewNotificationFormatter()

	// Initialize database service
	dbName := os.Getenv(ENV_DATABASE_NAME)
	if dbName == "" {
		dbName = "hackathon.db"
	}
	dbService, err := NewDatabaseService(dbName)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize database: %v", err))
	}
	defer dbService.Close()
	log.Println("Database service initialized successfully")

	fudChannel := make(chan twitterapi.NewMessage, 10)

	telegramService, err := NewTelegramService(os.Getenv(ENV_TELEGRAM_API_KEY), os.Getenv(ENV_PROXY_DSN), os.Getenv(ENV_TELEGRAM_ADMIN_CHAT_ID), notificationFormatter, dbService, fudChannel)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize telegram service: %v", err))
	}

	// Initialize user status manager
	userStatusManager := NewUserStatusManager()
	userStatusManager.StartPeriodicSave()

	// Initialize data (CSV import or community loading)
	log.Println("Initializing data...")
	initializeData(dbService, twitterApi)

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
	//notification channel
	notificationCh := make(chan FUDAlertNotification, 10)

	//start monitoring for new messages in community
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		MonitoringHandler(twitterApi, newMessageCh, dbService)
	}()
	//handle new message first step
	wg.Add(1)
	go func() {
		defer wg.Done()
		FirstStepHandler(newMessageCh, fudChannel, claudeApi, systemPromptFirstStep, userStatusManager, dbService, notificationCh)
	}()
	//handle fud messages with dynamic routing
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(notificationCh)

		for newMessage := range fudChannel {
			log.Printf("Second step processing for user %s", newMessage.Author.UserName)
			SecondStepHandler(newMessage, notificationCh, twitterApi, claudeApi, systemPromptSecondStep, userStatusManager, ticker, dbService)
		}
	}()
	//notification handler
	wg.Add(1)
	go func() {
		defer wg.Done()
		NotificationHandler(notificationCh, telegramService)
	}()
	// Cleanup
	defer userStatusManager.StopPeriodicSave()
	wg.Wait()
}

func PrepareClaudeSecondStepRequest(userTickerData *UserTickerMentionsData, followers *twitterapi.UserFollowersResponse, followings *twitterapi.UserFollowingsResponse, userStatusManager *UserStatusManager, communityActivity *UserCommunityActivity) ClaudeMessages {
	claudeMessages := ClaudeMessages{}

	// 1. User's ticker mentions with replied messages
	if userTickerData != nil {
		userDataJSON, _ := json.Marshal(userTickerData)
		claudeMessages = append(claudeMessages, ClaudeMessage{
			Role:    ROLE_USER,
			Content: fmt.Sprintf("USER'S TICKER MENTIONS AND REPLIES:\n%s", string(userDataJSON)),
		})
	} else {
		claudeMessages = append(claudeMessages, ClaudeMessage{
			Role:    ROLE_USER,
			Content: "USER'S TICKER MENTIONS AND REPLIES: No ticker mentions found",
		})
	}

	// 2. All user's friends with FUD analysis
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
		totalFriends, fudFriends, fudFriendsList := userStatusManager.GetFUDFriendsAnalysis(allFriends)

		friendsAnalysis := map[string]interface{}{
			"total_friends":       totalFriends,
			"fud_friends":         fudFriends,
			"fud_percentage":      float64(fudFriends) / float64(totalFriends) * 100,
			"fud_friends_details": fudFriendsList,
		}

		friendsJSON, _ := json.Marshal(friendsAnalysis)
		claudeMessages = append(claudeMessages, ClaudeMessage{
			Role:    ROLE_USER,
			Content: fmt.Sprintf("USER'S FRIENDS FUD ANALYSIS:\n%s", string(friendsJSON)),
		})
	} else {
		claudeMessages = append(claudeMessages, ClaudeMessage{
			Role:    ROLE_USER,
			Content: "USER'S FRIENDS FUD ANALYSIS: No friends found",
		})
	}

	// 3. User's community activity grouped by threads
	if communityActivity != nil && len(communityActivity.ThreadGroups) > 0 {
		communityActivityJSON, _ := json.Marshal(communityActivity)
		claudeMessages = append(claudeMessages, ClaudeMessage{
			Role:    ROLE_USER,
			Content: fmt.Sprintf("USER'S COMMUNITY ACTIVITY (ALL POSTS AND REPLIES GROUPED BY THREADS):\n%s", string(communityActivityJSON)),
		})
	} else {
		claudeMessages = append(claudeMessages, ClaudeMessage{
			Role:    ROLE_USER,
			Content: "USER'S COMMUNITY ACTIVITY: No activity found in community",
		})
	}

	return claudeMessages
}

type FirstStepClaudeResponse struct {
	IsFud          bool    `json:"is_fud"`
	FudProbability float64 `json:"fud_probability"`
	Reason         string  `json:"reason"`
}
type SecondStepClaudeResponse struct {
	IsFUDAttack    bool     `json:"is_fud_attack"`
	IsFUDUser      bool     `json:"is_fud_user"`
	FUDProbability float64  `json:"fud_probability"` // 0.0 - 1.0
	FUDType        string   `json:"fud_type"`        // "professional_trojan_horse", "professional_direct_attack", "professional_statistical", "emotional_escalation", "emotional_dramatic_exit", "casual_criticism", "none"
	UserRiskLevel  string   `json:"user_risk_level"` // "critical", "high", "medium", "low"
	KeyEvidence    []string `json:"key_evidence"`    // 2-4 most important evidence points
	DecisionReason string   `json:"decision_reason"` // 1-2 sentence summary of why this decision was made
	UserSummary    string   `json:"user_summary"`    // Short conclusion about user type for notifications
}

type UserTickerMentionsData struct {
	UserMessages    []UserMessageWithReplies `json:"user_messages"`
	TotalMessages   int                      `json:"total_messages"`
	RepliedMessages int                      `json:"replied_messages"`
	TokenCount      int                      `json:"token_count"`
}

type UserMessageWithReplies struct {
	TweetID     string      `json:"tweet_id"`
	CreatedAt   string      `json:"created_at"`
	Text        string      `json:"text"`
	InReplyToID string      `json:"in_reply_to_id,omitempty"`
	RepliedTo   *ReplyTweet `json:"replied_to,omitempty"`
}

type ReplyTweet struct {
	TweetID   string `json:"tweet_id"`
	CreatedAt string `json:"created_at"`
	Text      string `json:"text"`
	Author    string `json:"author"`
}

func getUserTickerMentions(twitterApi *twitterapi.TwitterAPIService, username string, ticker string, dbService *DatabaseService) *UserTickerMentionsData {
	const MAX_PAGES = 3
	const TOKEN_LIMIT = 50000 // Half of typical Claude token limit

	userMessages := []UserMessageWithReplies{}
	cursor := ""
	totalPages := 0
	replyTweetIDs := []string{}

	// Collect user messages with ticker mentions (max 3 pages)
	for totalPages < MAX_PAGES {
		searchResponse, err := twitterApi.AdvancedSearch(twitterapi.AdvancedSearchRequest{
			Query:     fmt.Sprintf("%s from:%s", ticker, username),
			QueryType: twitterapi.LATEST,
			Cursor:    cursor,
		})

		if err != nil {
			log.Printf("Error fetching user ticker mentions: %v", err)
			break
		}

		// Process messages and collect reply IDs
		for _, tweet := range searchResponse.Tweets {
			// Save tweet to database with ticker search source
			searchQuery := fmt.Sprintf("%s from:%s", ticker, username)
			storeTweetAndUserWithSource(dbService, tweet, TWEET_SOURCE_TICKER_SEARCH, ticker, searchQuery)

			userMessage := UserMessageWithReplies{
				TweetID:     tweet.Id,
				CreatedAt:   tweet.CreatedAt,
				Text:        tweet.Text,
				InReplyToID: tweet.InReplyToId,
			}

			if tweet.InReplyToId != "" {
				replyTweetIDs = append(replyTweetIDs, tweet.InReplyToId)
			}

			userMessages = append(userMessages, userMessage)
		}

		totalPages++

		if !searchResponse.HasNextPage || searchResponse.NextCursor == "" {
			break
		}
		cursor = searchResponse.NextCursor
	}

	// Get all replied-to messages in one batch request
	if len(replyTweetIDs) > 0 {
		repliedTweets, err := twitterApi.GetTweetsByIds(replyTweetIDs)
		if err == nil {
			// Create a map for quick lookup
			replyMap := make(map[string]ReplyTweet)
			for _, tweet := range repliedTweets.Tweets {
				// Save context tweets to database
				storeTweetAndUserWithSource(dbService, tweet, TWEET_SOURCE_CONTEXT, "", "context for "+username)

				replyMap[tweet.Id] = ReplyTweet{
					TweetID:   tweet.Id,
					CreatedAt: tweet.CreatedAt,
					Text:      tweet.Text,
					Author:    tweet.Author.UserName,
				}
			}

			// Associate replies with user messages
			for i := range userMessages {
				if userMessages[i].InReplyToID != "" {
					if reply, exists := replyMap[userMessages[i].InReplyToID]; exists {
						userMessages[i].RepliedTo = &reply
					}
				}
			}
		}
	}

	// Create result data
	result := &UserTickerMentionsData{
		UserMessages:    userMessages,
		TotalMessages:   len(userMessages),
		RepliedMessages: len(replyTweetIDs),
	}

	// Calculate token count and truncate if necessary
	jsonData, _ := json.Marshal(result)
	tokenCount := len(string(jsonData)) / 4 // Rough token estimation
	result.TokenCount = tokenCount

	// If exceeds token limit, cut data in half
	if tokenCount > TOKEN_LIMIT {
		halfLength := len(userMessages) / 2
		result.UserMessages = userMessages[:halfLength]
		result.TotalMessages = halfLength

		// Recalculate token count
		jsonData, _ = json.Marshal(result)
		result.TokenCount = len(string(jsonData)) / 4
	}

	return result
}

func SendIfNotExistsTweetToChannel(tweet twitterapi.Tweet, newMessageCh chan twitterapi.NewMessage, tweetsExistsStorage map[string]int, parentTweet twitterapi.Tweet, grandParentTweet twitterapi.Tweet) {
	if _, ok := tweetsExistsStorage[tweet.Id]; !ok {
		newMessageCh <- twitterapi.NewMessage{
			TweetID:      tweet.Id,
			ReplyTweetID: tweet.InReplyToId,
			Author: struct {
				UserName string
				Name     string
				ID       string
			}{tweet.Author.UserName, tweet.Author.Name, tweet.Author.Id},
			ParentTweet: struct {
				ID     string
				Author string
				Text   string
			}{ID: parentTweet.Id, Author: parentTweet.Author.UserName, Text: parentTweet.Text},
			GrandParentTweet: struct {
				ID     string
				Author string
				Text   string
			}{ID: grandParentTweet.Id, Author: grandParentTweet.Author.UserName, Text: grandParentTweet.Text},
			Text:         tweet.Text,
			CreatedAt:    tweet.CreatedAt,
			ReplyCount:   tweet.ReplyCount,
			LikeCount:    tweet.LikeCount,
			RetweetCount: tweet.RetweetCount,
		}
	}
}
