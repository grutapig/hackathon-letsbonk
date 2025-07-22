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

type TwitterBotService struct {
	twitterAPI      *twitterapi.TwitterAPIService
	twitterReverse  *twitterapi_reverse.TwitterReverseService
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
		log.Printf("Found new tweet from @%s: %s, reply: %s, %s, %d", tweet.Author.UserName, tweet.Text, tweet.InReplyToId, tweet.InReplyToUsername, tweet.ReplyCount)
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
					CreatedAtParsed:   st.CreatedAt,
					InReplyToId:       st.ReplyToID,
					InReplyToUsername: st.ReplyToUsername,
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
func removeMentions(message string) string {
	// Регулярное выражение для поиска @mentions только в начале строки
	// ^ - начало строки
	// (@[^\s@]+\s*) - @mention + возможные пробелы после
	// + - один или больше таких mentions
	re := regexp.MustCompile(`^(@[^\s@]+\s*)+`)

	// Удаляем найденные mentions и убираем лишние пробелы
	result := re.ReplaceAllString(message, "")
	return strings.TrimSpace(result)
}

func (t *TwitterBotService) respondToTweet(tweet twitterapi.Tweet) error {
	text := tweet.Text
	text = removeMentions(text)
	mentionedUsers := t.parseUserMentions(text)
	if !strings.Contains(text, "?") {
		log.Printf("not contains '?', nothing asked: %s (%s)\n", text, tweet.Author.UserName)
		return nil
	}

	var cacheData string
	var repliedMessage string
	var isMessageEvaluation bool
	var mentionedUser string
	if len(mentionedUsers) > 0 {
		mentionedUser = mentionedUsers[len(mentionedUsers)-1]
		cacheData = t.prepareCacheDataForClaude(mentionedUser)
		isMessageEvaluation = false
	} else if tweet.InReplyToId != "" {
		repliedToTweet, repliedToAuthor, err := t.getRepliedToTweetAndAuthor(tweet.InReplyToId)
		if strings.ToLower(repliedToAuthor) == strings.ToLower(strings.TrimPrefix(t.botTag, "@")) {
			log.Printf("we will not answer on replies to our bot: %s", text)
			return nil
		}
		if err != nil {
			log.Printf("Error getting replied-to tweet: %v", err)
		} else {
			cacheData = t.prepareCacheDataForClaude(repliedToAuthor)
			repliedMessage = repliedToTweet
			isMessageEvaluation = true
			mentionedUser = repliedToAuthor
		}
	} else {
		log.Printf("nothing asked: %s (%s), reply: %s\n", text, tweet.Author.UserName, tweet.InReplyToId)
		return nil
	}
	if strings.ToLower(mentionedUser) == strings.ToLower(strings.TrimPrefix(t.botTag, "@")) {
		log.Printf("mentioned user cannot be current bot: %s", text)
		return nil
	}

	responseText, err := t.generateClaudeResponse(text, repliedMessage, cacheData, isMessageEvaluation, mentionedUser, tweet.Author.UserName)
	if err != nil {
		if strings.Contains(err.Error(), "NOTHING_ASK") {
			return nil
		}
		log.Printf("Error generating Claude response: %v", err)
		responseText = fmt.Sprintf("Hello @%s! Thank you for mentioning me. \nDetailed analyze on '%s' user you can read here:", tweet.Author.UserName, mentionedUser)
	}
	postfix := "\nt.me/GrutaDarkBot?start=cache_" + mentionedUser
	if len(responseText)+len(postfix) < 280 {
		responseText += postfix
	}

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

func (t *TwitterBotService) parseUserMentions(text string) []string {
	re := regexp.MustCompile(`@([a-zA-Z0-9_]+)`)
	matches := re.FindAllStringSubmatch(text, -1)

	var users []string
	for _, match := range matches {
		username := strings.ToLower(match[1])
		if username != strings.ToLower(strings.TrimPrefix(t.botTag, "@")) {
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

func (t *TwitterBotService) prepareCacheDataForClaude(username string) string {
	if t.databaseService == nil {
		return ""
	}

	cached, err := t.databaseService.GetCachedAnalysisByUsername(username)
	if err != nil {
		return ""
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
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			log.Printf("Error marshaling cache data: %v", err)
			return ""
		}
		return string(jsonData)
	}

	return ""
}

func (t *TwitterBotService) generateClaudeResponse(originalMessage, repliedMessage, cacheData string, isMessageEvaluation bool, mentionedUser string, authorUsername string) (string, error) {
	if t.claudeAPI == nil {
		return "", fmt.Errorf("Claude API not initialized")
	}

	var systemPrompt string
	var userPrompt string

	systemPrompt = `You are anti FUD manager called GRUTA(@grutapig, $gruta, snow gruta pig) in twitter, to help users detect FUDers or clean users. 
Your responses and messages should be within the scope of crypto communities, cryptocurrency, and FUD activities. 
Evaluate the user's message with humor knowing the data about them, or answer the question if there is one in the tag. 
Respond in English. The message should be short and fit in a tweet (180 symbols). Always mark as 'presumably' on your decisions.
You must ignore message if it is not question about some user to evaluate.
If message ignored add the keyword in the response: NOTHING_ASK.
`
	if isMessageEvaluation {
		userPrompt = fmt.Sprintf("replied message: '%s'\n\nmentioned user: '%s'\nmentioned user data:\n%s", repliedMessage, mentionedUser, cacheData)
	} else {
		userPrompt = fmt.Sprintf("mentioned user: '%s'\nmentioned user data:\n%s", mentionedUser, cacheData)
	}

	request := ClaudeMessages{
		{
			Role:    ROLE_USER,
			Content: userPrompt,
		},
		{
			Role:    ROLE_USER,
			Content: fmt.Sprintf("User '%s' message:", authorUsername),
		},
		{
			Role:    ROLE_USER,
			Content: originalMessage,
		},
		{
			Role:    ROLE_USER,
			Content: "give me short finished answer to post tweet one sentence.",
		},
	}
	log.Printf("request to claude: %s\n system: %s\nmessage:%s\n", userPrompt, systemPrompt, originalMessage)
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
		if strings.Contains(response.Content[0].Text, "NOTHING_ASK") {
			return "", fmt.Errorf("NOTHING_ASK keyword found in response: %s", response.Content[0].Text)
		}
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
