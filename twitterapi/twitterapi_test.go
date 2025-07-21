package twitterapi

import (
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
	fmt.Println(communityTweetsResponse.NextCursor, len(communityTweetsResponse.Tweets))
	for i, tweet := range communityTweetsResponse.Tweets {
		fmt.Println(i, tweet.Author.UserName, tweet.Text, tweet.Id, tweet.QuoteCount, "reply_count", tweet.ReplyCount, "|||", tweet.CreatedAt, err)
	}
}

func TestTwitterAPIService_GetTweetReplies(t *testing.T) {
	godotenv.Load("../.env")
	api := NewTwitterAPIService(os.Getenv(ENV_TWITTER_API_KEY), os.Getenv(ENV_TWITTER_API_BASE_URL), os.Getenv(ENV_PROXY_DSN))
	tweetRepliesResponse, err := api.GetTweetReplies(TweetRepliesRequest{TweetID: os.Getenv(ENV_DEMO_TWEET_ID)})
	fmt.Println(tweetRepliesResponse.HasNextPage, tweetRepliesResponse.NextCursor)
	assert.NoError(t, err)
	for i, tweet := range tweetRepliesResponse.Tweets {
		fmt.Println(i, tweet.Author.Name, " || ", tweet.Author.UserName, " || ", tweet.Text, tweet.ReplyCount, "in reply to id||", tweet.InReplyToId, "date:", tweet.CreatedAt, err)
	}
}
func TestTwitterAPIService_GetTweetThreadContext(t *testing.T) {
	godotenv.Load("../.env")
	api := NewTwitterAPIService(os.Getenv(ENV_TWITTER_API_KEY), os.Getenv(ENV_TWITTER_API_BASE_URL), os.Getenv(ENV_PROXY_DSN))
	tweetRepliesResponse, err := api.GetTweetThreadContext(TweetRepliesRequest{TweetID: os.Getenv(ENV_DEMO_TWEET_ID)})
	fmt.Println("next page", tweetRepliesResponse.HasNextPage, tweetRepliesResponse.NextCursor)
	assert.NoError(t, err)
	for i, tweet := range tweetRepliesResponse.Tweets {
		fmt.Println(i, tweet.Author.Name, " || ", tweet.Author.UserName, " || ", tweet.Text, tweet.ReplyCount, "in reply to id||", tweet.InReplyToId, err)
	}
}
func TestTwitterAPIService_GetTweetsByIds(t *testing.T) {
	godotenv.Load("../.env")
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
	godotenv.Load("../.env")

	api := NewTwitterAPIService(os.Getenv(ENV_TWITTER_API_KEY), os.Getenv(ENV_TWITTER_API_BASE_URL), os.Getenv(ENV_PROXY_DSN))
	advancedSearchResponse, err := api.AdvancedSearch(AdvancedSearchRequest{
		Query:     fmt.Sprintf("@GrutaPig"),
		QueryType: LATEST,
		Cursor:    "",
	})
	for i, tweet := range advancedSearchResponse.Tweets {
		fmt.Println(i, tweet.Author.Id, tweet.Author.Name, tweet.Author.UserName, "tweet_id:", tweet.Id, tweet.CreatedAt, tweet.Text, tweet.ReplyCount, tweet.InReplyToId, err)
	}
	fmt.Println("tweets:", len(advancedSearchResponse.Tweets))
}
func TestTwitterAPIService_PostTweet(t *testing.T) {
	godotenv.Load("../.env")
	api := NewTwitterAPIService(os.Getenv(ENV_TWITTER_API_KEY), os.Getenv(ENV_TWITTER_API_BASE_URL), os.Getenv(ENV_PROXY_DSN))
	postTweetResponse, err := api.PostTweet(PostTweetRequest{
		AuthSession: os.Getenv(ENV_TWITTER_AUTH),
		TweetText: `Looking at the data, CDoughnath appears to be a solid long-term $DARK holder who's been consistently supportive of the community for months! 📊 They do technical analysis and seem genuinely invested in the project.
t.me/GrutaDarkBot?start=cache_CDoughnath
`,
		QuoteTweetId:     "",
		InReplyToTweetId: "",
		MediaId:          "",
		Proxy:            os.Getenv(ENV_PROXY_DSN),
	})
	fmt.Println(postTweetResponse, err)
}
