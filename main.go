package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/grutapig/hackaton/claude"
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/grutapig/hackaton/twitterapi_reverse"
	"github.com/joho/godotenv"
	"log"
	"os"
	"time"
)

const ENV_PROD_CONFIG = ".env"
const ENV_DEV_CONFIG = ".dev.env"
const PROMPT_FILE_STEP1 = "data/txt/prompt_simple.txt"
const PROMPT_FILE_STEP2 = "data/txt/prompt2.txt"

func main() {

	configFile := flag.String("config", ".env", "Configuration file to load (e.g., .env, .dev.env, .prod.env)")
	showHelp := flag.Bool("help", false, "Show help information")
	flag.BoolVar(showHelp, "h", false, "Show help information (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "FUD Detection System - Twitter/X Community Monitoring\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -config string\n")
		fmt.Fprintf(os.Stderr, "        Configuration file to load (default: none)\n")
		fmt.Fprintf(os.Stderr, "        Examples: .env, .dev.env, .prod.env\n")
		fmt.Fprintf(os.Stderr, "  -help, -h\n")
		fmt.Fprintf(os.Stderr, "        Show this help information\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s                    # Run with environment variables only\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -config .env       # Run with .env file\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -config .dev.env   # Run with development config\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -config .prod.env  # Run with production config\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Note: Environment variables will override config file values\n")
	}

	flag.Parse()

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if *configFile != "" {
		log.Printf("Loading configuration from: %s", *configFile)
		err := godotenv.Load(*configFile)
		if err != nil {
			log.Printf("Warning: Failed to load config file %s: %v", *configFile, err)
			log.Println("Continuing with environment variables...")
		} else {
			log.Printf("Successfully loaded configuration from %s", *configFile)
		}
	} else {
		log.Println("No config file specified, using environment variables only")
	}

	container, err := BuildContainer()
	if err != nil {
		panic(fmt.Sprintf("Failed to build container: %v", err))
	}

	err = container.Invoke(func(app *Application) {

		if err := app.Initialize(); err != nil {
			panic(fmt.Sprintf("Failed to initialize application: %v", err))
		}

		defer app.Shutdown()

		if err := app.Run(); err != nil {
			panic(fmt.Sprintf("Failed to run application: %v", err))
		}
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to invoke application: %v", err))
	}
}
func initializeData(dbService *DatabaseService, twitterApi *twitterapi.TwitterAPIService) {

	csvPath := os.Getenv(ENV_IMPORT_CSV_PATH)
	if csvPath != "" {
		log.Printf("CSV import path specified: %s", csvPath)

		importer := NewCSVImporter(dbService)
		result, err := importer.ImportCSV(csvPath)
		if err != nil {
			log.Printf("CSV import failed: %v", err)
			log.Println("Falling back to community data loading...")
		} else {
			log.Printf("CSV import successful: %s", result.String())
			return
		}
	}

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

func PrepareClaudeSecondStepRequest(userTickerData *UserTickerMentionsData, followers *twitterapi.UserFollowersResponse, followings *twitterapi.UserFollowingsResponse, dbService *DatabaseService, communityActivity *UserCommunityActivity) claude.ClaudeMessages {
	claudeMessages := claude.ClaudeMessages{}

	if userTickerData != nil {
		userDataJSON, _ := json.Marshal(userTickerData)
		claudeMessages = append(claudeMessages, claude.ClaudeMessage{
			Role:    claude.ROLE_USER,
			Content: fmt.Sprintf("USER'S TICKER MENTIONS AND REPLIES:\n%s", string(userDataJSON)),
		})
	} else {
		claudeMessages = append(claudeMessages, claude.ClaudeMessage{
			Role:    claude.ROLE_USER,
			Content: "USER'S TICKER MENTIONS AND REPLIES: No ticker mentions found",
		})
	}

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
		totalFriends, fudFriends, fudFriendsList := dbService.GetFUDFriendsAnalysis(allFriends)

		friendsAnalysis := map[string]interface{}{
			"total_friends":       totalFriends,
			"fud_friends":         fudFriends,
			"fud_percentage":      float64(fudFriends) / float64(totalFriends) * 100,
			"fud_friends_details": fudFriendsList,
		}

		friendsJSON, _ := json.Marshal(friendsAnalysis)
		claudeMessages = append(claudeMessages, claude.ClaudeMessage{
			Role:    claude.ROLE_USER,
			Content: fmt.Sprintf("USER'S FRIENDS FUD ANALYSIS:\n%s", string(friendsJSON)),
		})
	} else {
		claudeMessages = append(claudeMessages, claude.ClaudeMessage{
			Role:    claude.ROLE_USER,
			Content: "USER'S FRIENDS FUD ANALYSIS: No friends found",
		})
	}

	if communityActivity != nil && len(communityActivity.ThreadGroups) > 0 {
		communityActivityJSON, _ := json.Marshal(communityActivity)
		claudeMessages = append(claudeMessages, claude.ClaudeMessage{
			Role:    claude.ROLE_USER,
			Content: fmt.Sprintf("USER'S COMMUNITY ACTIVITY (ALL POSTS AND REPLIES GROUPED BY THREADS):\n%s", string(communityActivityJSON)),
		})
	} else {
		claudeMessages = append(claudeMessages, claude.ClaudeMessage{
			Role:    claude.ROLE_USER,
			Content: "USER'S COMMUNITY ACTIVITY: No activity found in community",
		})
	}

	return claudeMessages
}

type FirstStepClaudeResponse struct {
	IsFud bool `json:"is_fud"`
	//FudProbability float64 `json:"fud_probability"`
	//Reason         string  `json:"reason"`
}
type SecondStepClaudeResponse struct {
	IsFUDAttack    bool     `json:"is_fud_attack"`
	IsFUDUser      bool     `json:"is_fud_user"`
	FUDProbability float64  `json:"fud_probability"`
	FUDType        string   `json:"fud_type"`
	UserRiskLevel  string   `json:"user_risk_level"`
	KeyEvidence    []string `json:"key_evidence"`
	DecisionReason string   `json:"decision_reason"`
	UserSummary    string   `json:"user_summary"`
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
	const TOKEN_LIMIT = 50000

	userMessages := []UserMessageWithReplies{}
	cursor := ""
	totalPages := 0
	replyTweetIDs := []string{}

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

		for _, tweet := range searchResponse.Tweets {

			searchQuery := fmt.Sprintf("%s from:%s", ticker, username)
			storeTweetAndUserWithSource(dbService, tweet, TWEET_SOURCE_TICKER_SEARCH, ticker, searchQuery)

			if !dbService.TickerOpinionExists(tweet.Id) {
				tweetCreatedAt, _ := time.Parse(time.RFC3339, tweet.CreatedAt)
				opinion := UserTickerOpinionModel{
					UserID:         tweet.Author.Id,
					Username:       tweet.Author.UserName,
					Ticker:         ticker,
					TweetID:        tweet.Id,
					Text:           tweet.Text,
					TweetCreatedAt: tweetCreatedAt,
					InReplyToID:    tweet.InReplyToId,
					SearchQuery:    searchQuery,
				}

				err := dbService.SaveUserTickerOpinion(opinion)
				if err != nil {
					log.Printf("Failed to save ticker opinion for tweet %s: %v", tweet.Id, err)
				}
			}

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

	if len(replyTweetIDs) > 0 {
		repliedTweets, err := twitterApi.GetTweetsByIds(replyTweetIDs)
		if err == nil {

			replyMap := make(map[string]ReplyTweet)
			for _, tweet := range repliedTweets.Tweets {

				storeTweetAndUserWithSource(dbService, tweet, TWEET_SOURCE_CONTEXT, "", "context for "+username)

				replyMap[tweet.Id] = ReplyTweet{
					TweetID:   tweet.Id,
					CreatedAt: tweet.CreatedAt,
					Text:      tweet.Text,
					Author:    tweet.Author.UserName,
				}
			}

			for i := range userMessages {
				if userMessages[i].InReplyToID != "" {
					if reply, exists := replyMap[userMessages[i].InReplyToID]; exists {
						userMessages[i].RepliedTo = &reply

						opinion := UserTickerOpinionModel{}
						result := dbService.db.Where("tweet_id = ?", userMessages[i].TweetID).First(&opinion)
						if result.Error == nil {
							opinion.RepliedToText = reply.Text
							opinion.RepliedToAuthor = reply.Author
							dbService.SaveUserTickerOpinion(opinion)
						}
					}
				}
			}
		}
	}

	result := &UserTickerMentionsData{
		UserMessages:    userMessages,
		TotalMessages:   len(userMessages),
		RepliedMessages: len(replyTweetIDs),
	}

	jsonData, _ := json.Marshal(result)
	tokenCount := len(string(jsonData)) / 4
	result.TokenCount = tokenCount

	if tokenCount > TOKEN_LIMIT {
		halfLength := len(userMessages) / 2
		result.UserMessages = userMessages[:halfLength]
		result.TotalMessages = halfLength

		jsonData, _ = json.Marshal(result)
		result.TokenCount = len(string(jsonData)) / 4
	}

	return result
}

func SendIfNotExistsTweetToChannel(tweet twitterapi.Tweet, newMessageCh chan twitterapi.NewMessage, tweetsExistsStorage map[string]int, parentTweet twitterapi.Tweet, grandParentTweet twitterapi.Tweet, loggingService *LoggingService) {
	if _, ok := tweetsExistsStorage[tweet.Id]; !ok {
		newMessage := twitterapi.NewMessage{
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
		tweet.CreatedAtParsed, _ = twitterapi_reverse.ParseTwitterTime(tweet.CreatedAt)
		newMessage.CreatedAtParsed = tweet.CreatedAtParsed

		if loggingService != nil {
			err := loggingService.LogMessage(tweet.Id, tweet.Author.Id, tweet.Author.UserName, tweet.Text, TWEET_SOURCE_COMMUNITY, tweet.CreatedAtParsed)
			if err != nil {
				log.Printf("Error logging message: %v", err)
			}
		}

		newMessageCh <- newMessage
	}
}
