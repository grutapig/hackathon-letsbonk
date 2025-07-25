package main

import (
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/grutapig/hackaton/twitterapi_reverse"
)

func postTweet(text string, replyId string) error {
	if len(text) > 280 {
		text = text[:277] + "..."
	}

	_, err := twitterApi.PostTweet(twitterapi.PostTweetRequest{
		AuthSession:      twitterAuth,
		TweetText:        text,
		QuoteTweetId:     "",
		InReplyToTweetId: replyId,
		MediaId:          "",
		Proxy:            proxyDSN,
	})

	return err
}
func getUserTweets(username string) ([]twitterapi_reverse.SimpleTweet, error) {
	var tweets []twitterapi_reverse.SimpleTweet
	var cursor string

	for i := 0; i < userSearchPages; i++ {
		resp, err := twitterApi.AdvancedSearch(twitterapi.AdvancedSearchRequest{
			Query:     "from:" + username,
			QueryType: twitterapi.LATEST,
			Cursor:    cursor,
		})

		if err != nil || len(resp.Tweets) == 0 || resp.NextCursor == "" {
			break
		}

		cursor = resp.NextCursor
		for _, tweet := range resp.Tweets {
			saveTweetToDB(tweet)
			twitterTime, _ := twitterapi_reverse.ParseTwitterTime(tweet.CreatedAt)
			tweets = append(tweets, twitterapi_reverse.SimpleTweet{
				TweetID:   tweet.Id,
				Text:      tweet.Text,
				CreatedAt: twitterTime,
			})
		}
	}
	cursor = ""
	if len(tweets) == 0 {
		for i := 0; i < userSearchPages; i++ {
			resp, err := twitterApi.AdvancedSearch(twitterapi.AdvancedSearchRequest{
				Query:     "from:" + username,
				QueryType: twitterapi.TOP,
				Cursor:    cursor,
			})

			if err != nil || len(resp.Tweets) == 0 || resp.NextCursor == "" {
				break
			}

			cursor = resp.NextCursor
			for _, tweet := range resp.Tweets {
				saveTweetToDB(tweet)
				twitterTime, _ := twitterapi_reverse.ParseTwitterTime(tweet.CreatedAt)
				tweets = append(tweets, twitterapi_reverse.SimpleTweet{
					TweetID:   tweet.Id,
					Text:      tweet.Text,
					CreatedAt: twitterTime,
				})
			}
		}
	}

	return tweets, nil
}
