package main

import (
	"encoding/csv"
	"fmt"
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/joho/godotenv"
	"os"
	"strings"
	"time"
)

func main() {
	err := godotenv.Load()
	panicErr(err)
	api := twitterapi.NewTwitterAPIService(os.Getenv(twitterapi.ENV_TWITTER_API_KEY), os.Getenv(twitterapi.ENV_TWITTER_API_BASE_URL), os.Getenv(twitterapi.ENV_PROXY_DSN))

	os.RemoveAll("community_tweets.csv")
	file, err := os.Create("community_tweets.csv")
	panicErr(err)
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{
		"message_author",
		"message_number",
		"message_date",
		"reply_count",
		"reply_to_tweet",
		"message_text",
		"author_id",
		"tweet_id",
	}
	err = writer.Write(headers)
	panicErr(err)
	writer.Flush()

	communityID := os.Getenv(twitterapi.ENV_DEMO_COMMUNITY_ID)
	cursor := ""
	totalTweets := 0
	totalReplies := 0
	pageCount := 0

	fmt.Printf("🚀 Starting community scraping ID: %s\n", communityID)
	startTime := time.Now()

	for {
		pageCount++
		pageStartTime := time.Now()

		var communityResponse *twitterapi.CommunityTweetsResponse

		err := retryRequest(func() error {
			var requestErr error
			communityResponse, requestErr = api.GetCommunityTweets(twitterapi.CommunityTweetsRequest{
				CommunityID: communityID,
				Cursor:      cursor,
			})
			return requestErr
		}, fmt.Sprintf("getting page %d community tweets (cursor: %s)", pageCount, cursor))

		panicErr(err)

		pageDuration := time.Since(pageStartTime)
		tweetsInPage := len(communityResponse.Tweets)
		totalTweets += tweetsInPage

		fmt.Printf("📄 Page %d: got %d tweets in %v\n", pageCount, tweetsInPage, pageDuration)

		for i, tweet := range communityResponse.Tweets {
			tweetStartTime := time.Now()

			err = writeTweetToCSV(writer, tweet, "")
			panicErr(err)
			writer.Flush()

			repliesCount, err := scrapeRepliesRecursively(api, writer, tweet.Id, tweet.Id, 0)
			panicErr(err)

			totalReplies += repliesCount
			tweetDuration := time.Since(tweetStartTime)

			fmt.Printf("  💬 Tweet %d/%d (ID: %s): processed %d replies in %v\n",
				i+1, tweetsInPage, tweet.Id, repliesCount, tweetDuration)
		}

		elapsed := time.Since(startTime)
		fmt.Printf("📊 Intermediate statistics: %d tweets, %d replies, runtime: %v\n",
			totalTweets, totalReplies, elapsed)

		if !communityResponse.HasNext || communityResponse.NextCursor == "" {
			fmt.Println("📋 Reached end of community tweets list")
			break
		}
		cursor = communityResponse.NextCursor
	}

	totalDuration := time.Since(startTime)
	fmt.Printf("🎉 Scraping completed!\n")
	fmt.Printf("📈 Final statistics:\n")
	fmt.Printf("   - Processed pages: %d\n", pageCount)
	fmt.Printf("   - Total tweets: %d\n", totalTweets)
	fmt.Printf("   - Total replies: %d\n", totalReplies)
	fmt.Printf("   - Total runtime: %v\n", totalDuration)
	fmt.Printf("   - Average speed: %.2f tweets/min\n", float64(totalTweets+totalReplies)/totalDuration.Minutes())
	fmt.Println("💾 Data saved to community_tweets.csv")
}

func retryRequest(requestFunc func() error, description string) error {
	maxRetries := 5
	for attempt := 1; attempt <= maxRetries; attempt++ {
		startTime := time.Now()
		err := requestFunc()
		duration := time.Since(startTime)

		if err == nil {
			if attempt > 1 {
				fmt.Printf("✅ Request successful on attempt %d in %v: %s\n", attempt, duration, description)
			} else {
				fmt.Printf("✅ Request successful in %v: %s\n", duration, description)
			}
			return nil
		}

		fmt.Printf("❌ Attempt %d/%d failed in %v for %s: %v\n", attempt, maxRetries, duration, description, err)

		if attempt < maxRetries {
			waitTime := time.Duration(attempt*2) * time.Second
			fmt.Printf("⏳ Waiting %v before next attempt...\n", waitTime)
			time.Sleep(waitTime)
		}
	}

	return fmt.Errorf("all %d attempts failed for %s", maxRetries, description)
}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}

func scrapeRepliesRecursively(api *twitterapi.TwitterAPIService, writer *csv.Writer, tweetID string, rootTweetID string, depth int) (int, error) {
	cursor := ""
	totalReplies := 0
	pageCount := 0
	indent := strings.Repeat("  ", depth+1)

	for {
		pageCount++
		pageStartTime := time.Now()

		var repliesResponse *twitterapi.TweetRepliesResponse

		err := retryRequest(func() error {
			var requestErr error
			repliesResponse, requestErr = api.GetTweetReplies(twitterapi.TweetRepliesRequest{
				TweetID: tweetID,
				Cursor:  cursor,
			})
			return requestErr
		}, fmt.Sprintf("getting replies for tweet %s (depth %d, page %d)", tweetID, depth, pageCount))

		if err != nil {
			fmt.Printf("%s❌ Error getting replies for tweet %s: %v\n", indent, tweetID, err)
			return totalReplies, err
		}

		pageDuration := time.Since(pageStartTime)
		repliesInPage := len(repliesResponse.Tweets)

		if repliesInPage > 0 {
			fmt.Printf("%s🔍 Depth %d, page %d: found %d replies in %v\n",
				indent, depth, pageCount, repliesInPage, pageDuration)
		}

		for _, reply := range repliesResponse.Tweets {
			err = writeTweetToCSV(writer, reply, tweetID)
			if err != nil {
				return totalReplies, err
			}
			writer.Flush()
			totalReplies++

			if reply.ReplyCount > 0 {
				nestedReplies, err := scrapeRepliesRecursively(api, writer, reply.Id, rootTweetID, depth+1)
				if err != nil {
					return totalReplies, err
				}
				totalReplies += nestedReplies
			}
		}

		if !repliesResponse.HasNextPage || repliesResponse.NextCursor == "" {
			if pageCount > 1 && repliesInPage > 0 {
				fmt.Printf("%s✅ Processed %d reply pages for tweet %s\n", indent, pageCount, tweetID)
			}
			break
		}
		cursor = repliesResponse.NextCursor
	}

	return totalReplies, nil
}

func writeTweetToCSV(writer *csv.Writer, tweet twitterapi.Tweet, replyToTweetID string) error {
	cleanText := strings.TrimPrefix(tweet.Text, "crypto_crores ")

	logText := cleanText
	if len(logText) > 50 {
		logText = logText[:47] + "..."
	}

	record := []string{
		tweet.Author.UserName,
		tweet.Id,
		tweet.CreatedAt,
		fmt.Sprintf("%d", tweet.ReplyCount),
		replyToTweetID,
		cleanText,
		tweet.Author.Id,
		tweet.Id,
	}

	err := writer.Write(record)
	if err == nil {
		fmt.Printf("    ✏️  Wrote tweet @%s: \"%s\"\n", tweet.Author.UserName, logText)
	}

	return err
}
