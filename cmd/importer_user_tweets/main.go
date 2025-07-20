package main

import (
	"encoding/csv"
	"fmt"
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/joho/godotenv"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {
	err := godotenv.Load()
	panicErr(err)

	targetUsersEnv := os.Getenv("target_users")
	if targetUsersEnv == "" {
		panic("target_users environment variable is required")
	}

	ticker := os.Getenv("twitter_community_ticker")
	if ticker == "" {
		panic("twitter_community_ticker environment variable is required")
	}

	usernames := strings.Split(targetUsersEnv, ",")
	for i, username := range usernames {
		usernames[i] = strings.TrimSpace(username)
	}

	fmt.Printf("🚀 Starting user tweets import for %d users with ticker '%s'\n", len(usernames), ticker)
	fmt.Printf("📋 Target users: %s\n", strings.Join(usernames, ", "))

	api := twitterapi.NewTwitterAPIService(os.Getenv("twitter_api_key"), os.Getenv("twitter_api_base_url"), os.Getenv("proxy_dsn"))

	filename := fmt.Sprintf("user_tweets_%s_%s.csv", ticker, time.Now().Format("20060102_150405"))
	file, err := os.Create(filename)
	panicErr(err)
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{
		"username",
		"user_id",
		"tweet_id",
		"created_at",
		"text",
		"in_reply_to_id",
		"reply_count",
		"like_count",
		"retweet_count",
		"ticker",
		"search_query",
	}
	err = writer.Write(headers)
	panicErr(err)
	writer.Flush()

	userCh := make(chan string, len(usernames))
	resultCh := make(chan UserSearchResult, len(usernames)*10)

	var wg sync.WaitGroup
	numWorkers := 5
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			searchWorker(api, ticker, userCh, resultCh)
		}()
	}

	var processorWg sync.WaitGroup
	processorWg.Add(1)
	go func() {
		defer processorWg.Done()
		processResults(writer, resultCh)
	}()

	startTime := time.Now()
	for _, username := range usernames {
		userCh <- username
	}
	close(userCh)

	wg.Wait()
	close(resultCh)

	processorWg.Wait()

	totalDuration := time.Since(startTime)
	fmt.Printf("🎉 Import completed in %v\n", totalDuration)
	fmt.Printf("💾 Data saved to %s\n", filename)
}

type UserSearchResult struct {
	Username string
	Tweets   []twitterapi.Tweet
	Error    error
}

func searchWorker(api *twitterapi.TwitterAPIService, ticker string, userCh <-chan string, resultCh chan<- UserSearchResult) {
	for username := range userCh {
		fmt.Printf("🔍 Searching tweets for user: %s\n", username)

		tweets, err := searchUserTweets(api, username, ticker)

		resultCh <- UserSearchResult{
			Username: username,
			Tweets:   tweets,
			Error:    err,
		}
	}
}

func searchUserTweets(api *twitterapi.TwitterAPIService, username, ticker string) ([]twitterapi.Tweet, error) {
	var allTweets []twitterapi.Tweet
	cursor := ""
	maxPages := 3
	pageCount := 0

	for pageCount < maxPages {
		searchQuery := fmt.Sprintf("%s from:%s", ticker, username)

		response, err := api.AdvancedSearch(twitterapi.AdvancedSearchRequest{
			Query:     searchQuery,
			QueryType: twitterapi.LATEST,
			Cursor:    cursor,
		})

		if err != nil {
			fmt.Printf("❌ Error searching tweets for %s: %v\n", username, err)
			return allTweets, err
		}

		tweetsInPage := len(response.Tweets)
		allTweets = append(allTweets, response.Tweets...)

		fmt.Printf("  📄 Page %d: found %d tweets for %s\n", pageCount+1, tweetsInPage, username)

		pageCount++

		if !response.HasNextPage || response.NextCursor == "" {
			break
		}
		cursor = response.NextCursor

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("  ✅ Total %d tweets found for %s\n", len(allTweets), username)
	return allTweets, nil
}

func processResults(writer *csv.Writer, resultCh <-chan UserSearchResult) {
	totalTweets := 0
	successfulUsers := 0
	failedUsers := 0

	for result := range resultCh {
		if result.Error != nil {
			fmt.Printf("❌ Failed to process %s: %v\n", result.Username, result.Error)
			failedUsers++
			continue
		}

		successfulUsers++
		userTweets := len(result.Tweets)
		totalTweets += userTweets

		fmt.Printf("💾 Writing %d tweets for %s to CSV\n", userTweets, result.Username)

		for _, tweet := range result.Tweets {
			record := []string{
				tweet.Author.UserName,
				tweet.Author.Id,
				tweet.Id,
				tweet.CreatedAt,
				tweet.Text,
				tweet.InReplyToId,
				fmt.Sprintf("%d", tweet.ReplyCount),
				fmt.Sprintf("%d", tweet.LikeCount),
				fmt.Sprintf("%d", tweet.RetweetCount),
				os.Getenv("twitter_community_ticker"),
				fmt.Sprintf("%s from:%s", os.Getenv("twitter_community_ticker"), result.Username),
			}

			err := writer.Write(record)
			if err != nil {
				fmt.Printf("❌ Error writing tweet to CSV: %v\n", err)
			}
		}
		writer.Flush()
	}

	fmt.Printf("\n📊 Final statistics:\n")
	fmt.Printf("   - Successful users: %d\n", successfulUsers)
	fmt.Printf("   - Failed users: %d\n", failedUsers)
	fmt.Printf("   - Total tweets: %d\n", totalTweets)
	if successfulUsers > 0 {
		fmt.Printf("   - Average tweets per user: %.1f\n", float64(totalTweets)/float64(successfulUsers))
	}
}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}
