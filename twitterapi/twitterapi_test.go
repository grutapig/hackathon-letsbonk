package twitterapi

import (
	"encoding/csv"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestTwitterAPIService_GetCommunityTweets(t *testing.T) {
	err := godotenv.Load("../.env")
	assert.NoError(t, err)
	api := NewTwitterAPIService(os.Getenv(ENV_TWITTER_API_KEY), os.Getenv(ENV_TWITTER_API_BASE_URL), os.Getenv(ENV_PROXY_DSN))
	communityTweetsResponse, err := api.GetCommunityTweets(CommunityTweetsRequest{CommunityID: os.Getenv(ENV_DEMO_COMMUNITY_ID), Cursor: ""})
	assert.NoError(t, err)
	fmt.Println(communityTweetsResponse.HasNext, communityTweetsResponse.NextCursor, len(communityTweetsResponse.Tweets))
	for i, tweet := range communityTweetsResponse.Tweets {
		fmt.Println(i, tweet.Author.UserName, tweet.Text, tweet.Id, tweet.QuoteCount, tweet.ReplyCount, "|||", tweet.CreatedAt, err)
	}
}

func TestTwitterAPIService_GetTweetReplies(t *testing.T) {
	godotenv.Load()
	api := NewTwitterAPIService(os.Getenv(ENV_TWITTER_API_KEY), os.Getenv(ENV_TWITTER_API_BASE_URL), os.Getenv(ENV_PROXY_DSN))
	tweetRepliesResponse, err := api.GetTweetReplies(TweetRepliesRequest{TweetID: os.Getenv(ENV_DEMO_TWEET_ID), Cursor: ""})
	fmt.Println(tweetRepliesResponse.HasNextPage, tweetRepliesResponse.NextCursor)
	assert.NoError(t, err)
	for i, tweet := range tweetRepliesResponse.Tweets {
		fmt.Println(i, tweet.Author.Name, " || ", tweet.Author.UserName, " || ", tweet.Text, tweet.ReplyCount, "in reply to id||", tweet.InReplyToId, err)
	}
}
func TestTwitterAPIService_GetTweetsByIds(t *testing.T) {
	godotenv.Load()
	api := NewTwitterAPIService(os.Getenv(ENV_TWITTER_API_KEY), os.Getenv(ENV_TWITTER_API_BASE_URL), os.Getenv(ENV_PROXY_DSN))
	tweetRepliesResponse, err := api.GetTweetsByIds([]string{os.Getenv(ENV_DEMO_TWEET_ID)})
	assert.NoError(t, err)
	for i, tweet := range tweetRepliesResponse.Tweets {
		fmt.Println(i, tweet.Author.Name, " || ", tweet.Author.UserName, " || ", tweet.Text, tweet.ReplyCount, err)
	}
}
func TestTwitterAPIService_GetUserLastTweets(t *testing.T) {
	godotenv.Load()
	api := NewTwitterAPIService(os.Getenv(ENV_TWITTER_API_KEY), os.Getenv(ENV_TWITTER_API_BASE_URL), os.Getenv(ENV_PROXY_DSN))
	lastTweetsResponse, err := api.GetUserLastTweets(UserLastTweetsRequest{
		UserId: os.Getenv(ENV_DEMO_USER_ID),
	})
	assert.NoError(t, err)
	fmt.Println(lastTweetsResponse.HasNextPage, lastTweetsResponse.NextCursor)
	for i, tweet := range lastTweetsResponse.Data.Tweets {
		fmt.Println(i, tweet.Author.UserName, tweet.Text, tweet.ReplyCount, tweet.CreatedAt, err)
	}
}
func TestTwitterAPIService_GetUserFollowers(t *testing.T) {
	godotenv.Load()
	api := NewTwitterAPIService(os.Getenv(ENV_TWITTER_API_KEY), os.Getenv(ENV_TWITTER_API_BASE_URL), os.Getenv(ENV_PROXY_DSN))
	followersResponse, err := api.GetUserFollowers(UserFollowersRequest{
		UserName: os.Getenv(ENV_DEMO_USER_NAME),
		Cursor:   "",
		PageSize: 100,
	})
	assert.NoError(t, err)
	fmt.Println("next page:", followersResponse.HasNextPage, followersResponse.NextCursor)
	for i, user := range followersResponse.Followers {
		fmt.Println(i, user.Name, user.ScreenName, user.Protected, user.CreatedAt, err)
	}
}
func TestTwitterAPIService_GetUserFollowings(t *testing.T) {
	godotenv.Load()
	api := NewTwitterAPIService(os.Getenv(ENV_TWITTER_API_KEY), os.Getenv(ENV_TWITTER_API_BASE_URL), os.Getenv(ENV_PROXY_DSN))
	followings, err := api.GetUserFollowings(UserFollowingsRequest{
		UserName: os.Getenv(ENV_DEMO_USER_NAME),
		Cursor:   "",
		PageSize: 100,
	})
	assert.NoError(t, err)
	fmt.Println(followings.HasNextPage, followings.NextCursor)
	for i, user := range followings.Followings {
		fmt.Println(i, user.Name, user.ScreenName, user.Protected, user.CreatedAt, err)
	}
}
func TestTwitterAPIService_AdvancedSearch(t *testing.T) {
	godotenv.Load()

	// Create CSV file
	os.RemoveAll("advanced_search_results.csv")
	csvFile, err := os.Create("advanced_search_results.csv")
	assert.NoError(t, err)
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	// Write CSV header
	header := []string{"Index", "Author_ID", "Author_Name", "Author_UserName", "Tweet_ID", "Created_At", "Text", "In_Reply_To_ID", "Reply_Author_UserName", "Reply_Text"}
	err = writer.Write(header)
	assert.NoError(t, err)
	cursor := ""
	api := NewTwitterAPIService(os.Getenv(ENV_TWITTER_API_KEY), os.Getenv(ENV_TWITTER_API_BASE_URL), os.Getenv(ENV_PROXY_DSN))
	for n := 0; ; n++ {
		fmt.Println("#", n, "next cursor:", cursor)
		advancedSearchResponse, err := api.AdvancedSearch(AdvancedSearchRequest{
			Query:     fmt.Sprintf("%s from:%s", os.Getenv(ENV_TWITTER_COMMUNITY_TICKER), os.Getenv(ENV_DEMO_USER_NAME)),
			QueryType: LATEST,
			Cursor:    cursor,
		})
		if err != nil {
			break
		}
		for i, tweet := range advancedSearchResponse.Tweets {
			fmt.Println(i, tweet.Author.Id, tweet.Author.Name, tweet.Author.UserName, "tweet_id:", tweet.Id, tweet.CreatedAt, tweet.Text, tweet.InReplyToId, err)

			replyAuthorUsername := ""
			replyText := ""

			if tweet.InReplyToId != "" {
				reply, err := api.GetTweetsByIds([]string{tweet.InReplyToId})
				if err == nil {
					for _, tweetR := range reply.Tweets {
						fmt.Println(i, "in reply of", tweetR.Author.UserName, tweetR.Text, err)
						replyAuthorUsername = tweetR.Author.UserName
						replyText = tweetR.Text
					}
				}
			}

			// Write tweet data to CSV
			record := []string{
				fmt.Sprintf("%d", i),
				tweet.Author.Id,
				tweet.Author.Name,
				tweet.Author.UserName,
				tweet.Id,
				tweet.CreatedAt,
				tweet.Text,
				tweet.InReplyToId,
				replyAuthorUsername,
				replyText,
			}
			err = writer.Write(record)
			writer.Flush()
			assert.NoError(t, err)
		}
		if advancedSearchResponse.NextCursor == "" {
			break
		}
		cursor = advancedSearchResponse.NextCursor
	}
}
