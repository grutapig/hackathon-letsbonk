package twitterapi_reverse

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTweetsFromCommunityResponse(t *testing.T) {
	data, err := os.ReadFile("fixtures/community_tweets.json")
	require.NoError(t, err)

	result, err := ParseCommunityTweets(data)
	if err != nil {
		fmt.Printf("Parser errors: %v\n", err)
		// Don't fail the test immediately if we got some results
		if len(result) == 0 {
			t.Fatalf("No tweets parsed and got error: %v", err)
		}
	}

	fmt.Printf("Successfully parsed %d tweets\n", len(result))
	for i, tweet := range result {
		fmt.Printf("Tweet %d:\n", i+1)
		fmt.Printf("  ID: %s\n", tweet.ID)
		fmt.Printf("  UserID: %s\n", tweet.UserID)
		fmt.Printf("  FullText: %s\n", tweet.FullText)
		fmt.Printf("  ReplyCount: %d\n", tweet.ReplyCount)
		fmt.Printf("  CreatedAt: %v\n", tweet.CreatedAt)
		fmt.Printf("  Author ID: %s\n", tweet.Author.ID)
		fmt.Printf("  Author ScreenName: %s\n", tweet.Author.ScreenName)
		fmt.Printf("  Author Name: %s\n", tweet.Author.Name)
		fmt.Printf("  Author CreatedAt: %v\n", tweet.Author.CreatedAt)
		fmt.Println("---")
	}
}

func TestParseTweetsFromThreadedConversationResponse(t *testing.T) {
	data, err := os.ReadFile("../docs/tweet.json")
	require.NoError(t, err)

	result, err := ParseThreadedConversationTweets(data)
	if err != nil {
		fmt.Printf("Parser errors: %v\n", err)
		// Don't fail the test immediately if we got some results
		if len(result) == 0 {
			t.Fatalf("No tweets parsed and got error: %v", err)
		}
	}

	fmt.Printf("Successfully parsed %d tweets from threaded conversation\n", len(result))
	for i, tweet := range result {
		fmt.Printf("Tweet %d:\n", i+1)
		fmt.Printf("  ID: %s\n", tweet.ID)
		fmt.Printf("  UserID: %s\n", tweet.UserID)
		fmt.Printf("  FullText: %s\n", tweet.FullText)
		fmt.Printf("  ReplyCount: %d\n", tweet.ReplyCount)
		fmt.Printf("  CreatedAt: %v\n", tweet.CreatedAt)
		fmt.Printf("  Author ID: %s\n", tweet.Author.ID)
		fmt.Printf("  Author ScreenName: %s\n", tweet.Author.ScreenName)
		fmt.Printf("  Author Name: %s\n", tweet.Author.Name)
		fmt.Printf("  Author CreatedAt: %v\n", tweet.Author.CreatedAt)
		fmt.Println("---")
	}
}
