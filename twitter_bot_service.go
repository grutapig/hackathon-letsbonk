package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/grutapig/hackaton/twitterapi_reverse"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const TWEET_POST_ADMIN_USERS = "Jman369420,NyraanAlpha,kordaciusluk,Dogtor_a1"

type TwitterBotService struct {
	twitterAPI      *twitterapi.TwitterAPIService
	twitterReverse  *twitterapi_reverse.TwitterReverseService
	claudeAPI       *ClaudeApi
	databaseService *DatabaseService
	botTag          string
	authSession     string
	proxyDsn        string
	knownTweets     map[string]bool
	adminUsers      []string
	tweetsMutex     sync.RWMutex
	isMonitoring    bool
	monitoringMutex sync.Mutex
}

func NewTwitterBotService(twitterAPI *twitterapi.TwitterAPIService, twitterReverse *twitterapi_reverse.TwitterReverseService, databaseService *DatabaseService, claudeApi *ClaudeApi) *TwitterBotService {
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
		twitterAPI:      twitterAPI,
		twitterReverse:  twitterReverse,
		databaseService: databaseService,
		botTag:          botTag,
		authSession:     authSession,
		claudeAPI:       claudeApi,
		proxyDsn:        proxyDSN,
		adminUsers:      strings.Split(TWEET_POST_ADMIN_USERS, ","),
		knownTweets:     make(map[string]bool),
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

	ticker := time.NewTicker(30 * time.Second)
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
			log.Println("twitter bot checking...")
			if err := t.checkForNewTweets(); err != nil {
				log.Printf("Error checking for new tweets: %v", err)
			}
		}
	}
}

func (t *TwitterBotService) initializeKnownTweets() error {
	tweets, err := t.getNewMentions()
	if err != nil {
		return fmt.Errorf("error in initial search: %w", err)
	}

	t.tweetsMutex.Lock()
	defer t.tweetsMutex.Unlock()

	for _, tweet := range tweets {
		t.knownTweets[tweet.Id] = true
		log.Println(tweet.CreatedAt, tweet.Id, tweet.Text, tweet.Author.UserName, tweet.InReplyToId)
	}

	log.Printf("twitter bot Initialized with %d known tweets", len(t.knownTweets))
	return nil
}

func (t *TwitterBotService) checkForNewTweets() error {
	tweets, err := t.getNewMentions()
	if err != nil {
		return fmt.Errorf("error getting mentions: %w", err)
	}
	log.Printf("found tweets: %d\n", len(tweets))

	newTweets := t.findNewTweets(tweets)

	for _, tweet := range newTweets {
		log.Printf("Found new tweet from @%s: %s, reply: %s, %d", tweet.Author.UserName, tweet.Text, tweet.InReplyToId, tweet.ReplyCount)
		if err := t.respondToTweet(tweet); err != nil {
			log.Printf("Error responding to tweet %s: %v", tweet.Id, err)
		}
	}

	return nil
}

func (t *TwitterBotService) getNewMentions() ([]twitterapi.Tweet, error) {
	if t.twitterReverse != nil {
		simpleTweets, err := t.twitterReverse.GetNotificationsSimple()
		if err == nil {
			var tweets []twitterapi.Tweet
			for _, st := range simpleTweets {
				tweet := twitterapi.Tweet{
					Id:   st.TweetID,
					Text: st.Text,
					Author: twitterapi.Author{
						UserName: st.Author.Username,
					},
					CreatedAtParsed: st.CreatedAt,
					InReplyToId:     st.ReplyToID,
				}
				tweets = append(tweets, tweet)
			}
			log.Printf("Success get from reverse: %d", len(tweets))
			return tweets, nil
		}
		log.Printf("Reverse service failed, falling back to advanced search: %v", err)
	}

	query := fmt.Sprintf("%s", t.botTag)
	searchRequest := twitterapi.AdvancedSearchRequest{
		Query:     query,
		QueryType: twitterapi.LATEST,
	}

	response, err := t.twitterAPI.AdvancedSearch(searchRequest)
	if err != nil {
		return nil, err
	}

	return response.Tweets, nil
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
	isAdmin := strings.Contains(TWEET_POST_ADMIN_USERS, tweet.Author.UserName)
	mentionedUsers := t.parseUserMentions(tweet.Text, tweet.Author.UserName)
	if !strings.Contains(tweet.Text, "?") && !isAdmin {
		log.Printf("not contains '?', nothing asked: %s (%s)\n", tweet.Text, tweet.Author.UserName)
		return nil
	}

	if !strings.Contains(strings.ToLower(tweet.Text), strings.ToLower(t.botTag)) {
		log.Printf("not contains '%s', just skip; text: %s; author: %s\n", t.botTag, tweet.Text, tweet.Author.UserName)
		return nil
	}

	var cacheData string
	var repliedMessage string
	var isMessageEvaluation bool
	var mentionedUser string
	if len(mentionedUsers) > 0 {
		cacheData = t.prepareCacheDataForClaude(mentionedUsers)
		isMessageEvaluation = false
		mentionedUser = mentionedUsers[0]
	} else if tweet.InReplyToId != "" {
		repliedToTweet, repliedToAuthor, err := t.getRepliedToTweetAndAuthor(tweet.InReplyToId)
		if err != nil {
			log.Printf("Error getting replied-to tweet: %v", err)
		} else {
			cacheData = t.prepareCacheDataForClaude([]string{repliedToAuthor})
			repliedMessage = repliedToTweet
			isMessageEvaluation = true
			mentionedUser = repliedToAuthor
		}
	} else {
		log.Printf("nothing asked: %s (%s), reply: %s\n", tweet.Text, tweet.Author.UserName, tweet.InReplyToId)
		return nil
	}

	responseText, err := t.generateClaudeResponse(tweet.Text, repliedMessage, cacheData, isMessageEvaluation, mentionedUser, tweet.Author.UserName)
	if err != nil {
		log.Printf("Error generating Claude response: %v", err)
		responseText = fmt.Sprintf("Hello @%s! Thank you for mentioning me. \nDetailed analyze on '%s' user you can read here:", tweet.Author.UserName, mentionedUser)
	}
	responseText += "\nt.me/GrutaDarkBot?start=cache_" + mentionedUser

	log.Println("Final response:", responseText)
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

func (t *TwitterBotService) parseUserMentions(text, currentUser string) []string {
	re := regexp.MustCompile(`@([a-zA-Z0-9_]+)`)
	matches := re.FindAllStringSubmatch(text, -1)

	var users []string
	for _, match := range matches {
		username := strings.ToLower(match[1])
		if username != strings.ToLower(currentUser) && username != strings.ToLower(strings.TrimPrefix(t.botTag, "@")) {
			users = append(users, username)
		}
	}
	return users
}

func (t *TwitterBotService) getCacheAnalysisInfo(usernames []string) string {
	if t.databaseService == nil {
		return ""
	}

	var results []string
	for _, username := range usernames {
		cached, err := t.databaseService.GetCachedAnalysisByUsername(username)
		if err != nil {
			log.Printf("Error getting cached analysis for %s: %v", username, err)
			continue
		}

		if cached != nil {
			info := fmt.Sprintf("@%s: %s (%.0f%% confidence)", username, cached.UserRiskLevel, cached.FUDProbability*100)
			if cached.IsFUDUser {
				info += fmt.Sprintf(" - FUD Type: %s", cached.FUDType)
			}
			results = append(results, info)
		}
	}

	if len(results) > 0 {
		return "Cache Analysis:\n" + strings.Join(results, "\n")
	}
	return ""
}

func (t *TwitterBotService) prepareCacheDataForClaude(usernames []string) string {
	if t.databaseService == nil || len(usernames) == 0 {
		return ""
	}

	var cacheDataList []map[string]interface{}
	for _, username := range usernames {
		cached, err := t.databaseService.GetCachedAnalysisByUsername(username)
		if err != nil {
			continue
		}

		if cached != nil {
			data := map[string]interface{}{
				"username":        cached.Username,
				"is_fud_user":     cached.IsFUDUser,
				"fud_type":        cached.FUDType,
				"fud_probability": cached.FUDProbability,
				"user_risk_level": cached.UserRiskLevel,
				"user_summary":    cached.UserSummary,
				"decision_reason": cached.DecisionReason,
			}
			cacheDataList = append(cacheDataList, data)
		}
	}

	if len(cacheDataList) == 0 {
		return ""
	}

	jsonData, err := json.MarshalIndent(cacheDataList, "", "  ")
	if err != nil {
		log.Printf("Error marshaling cache data: %v", err)
		return ""
	}

	return string(jsonData)
}

func (t *TwitterBotService) generateClaudeResponse(originalMessage, repliedMessage, cacheData string, isMessageEvaluation bool, mentionedUser string, authorUsername string) (string, error) {
	if t.claudeAPI == nil {
		return "", fmt.Errorf("Claude API not initialized")
	}

	var systemPrompt string
	var userPrompt string

	if isMessageEvaluation {
		systemPrompt = "You are anti FUD manager, to help users detect FUDers or clean users. Your responses and messages should be within the scope of crypto communities, cryptocurrency, and FUD activities, but admin users list can asks about anything. Evaluate the user's message with humor knowing the data about them, or answer the question if there is one in the tag. Respond in English. The message should be short and fit in a tweet (180 symbols). If user is FUDer, add remark presumably."
		userPrompt = fmt.Sprintf("Tagger's message: '%s'\n\nMessage to evaluate: '%s'\n\nAuthor of message and user to analyze: '%s'\nUser data:\n%s", originalMessage, repliedMessage, mentionedUser, cacheData)
	} else {
		systemPrompt = "You are anti FUD manager, to help users detect FUDers or clean users. Your responses and messages should be within the scope of crypto communities, cryptocurrency, and FUD activities, but admin users list can asks about anything. Answer the user's question with humor in English knowing the given information. The message should be short and fit in a tweet (180 symbols). If user is FUDer, add remark presumably."
		userPrompt = fmt.Sprintf("Original message: '%s'\nuser to analyze: '%s'\nCache information:\n%s", originalMessage, mentionedUser, cacheData)
	}

	request := ClaudeMessages{
		{
			Role:    ROLE_USER,
			Content: fmt.Sprintf("admins user list: %s", TWEET_POST_ADMIN_USERS),
		},
		{
			Role:    ROLE_USER,
			Content: fmt.Sprintf("User '%s' asks:", authorUsername),
		},
		{
			Role:    ROLE_USER,
			Content: userPrompt,
		},
		{
			Role:    ROLE_USER,
			Content: "analyze and give me short finished answer to post tweet one sentence.",
		},
	}
	log.Printf("request to claude: %s, system: %s\n", userPrompt, systemPrompt)
	response, err := t.claudeAPI.SendMessage(request, systemPrompt)
	if err != nil {
		log.Println("claude sendMessage:", request, ". error:", err, ". Try one more time...")
		response, err = t.claudeAPI.SendMessage(request, systemPrompt)
		if err != nil {
			return "", err
		}
	}

	if len(response.Content) > 0 {
		log.Printf("response for claude: %s", response.Content[0].Text)
		return response.Content[0].Text, nil
	}

	return "", fmt.Errorf("empty response from Claude")
}

func (t *TwitterBotService) limitResponseLength(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	if maxLength <= 3 {
		return text[:maxLength]
	}

	return text[:maxLength-3] + "..."
}

func (t *TwitterBotService) getRepliedToTweetAndAuthor(tweetID string) (text string, username string, err error) {
	tweetsByIdsResponse, err := t.twitterAPI.GetTweetsByIds([]string{tweetID})
	if err != nil {
		return "", "", fmt.Errorf("error fetching tweet by ID: %w", err)
	}

	if len(tweetsByIdsResponse.Tweets) == 0 {
		return "", "", fmt.Errorf("tweet not found")
	}

	tweet := tweetsByIdsResponse.Tweets[0]
	return tweet.Text, tweet.Author.UserName, nil
}
