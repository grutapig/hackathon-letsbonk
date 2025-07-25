package main

import (
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/grutapig/hackaton/twitterapi_reverse"
)

func saveTweetToDB(tweet twitterapi.Tweet) bool {
	twitterTime, _ := twitterapi_reverse.ParseTwitterTime(tweet.CreatedAt)
	dbTweet := Tweet{
		TweetID:   tweet.Id,
		Text:      tweet.Text,
		Username:  tweet.Author.UserName,
		UserID:    tweet.Author.Id,
		CreatedAt: twitterTime,
	}
	result := db.Create(&dbTweet)
	return result.Error == nil && result.RowsAffected > 0
}

func getAllTweets(limit int) ([]Tweet, error) {
	var tweets []Tweet
	result := db.Order("created_at DESC").Limit(limit).Find(&tweets)
	return tweets, result.Error
}

func getTweetCountByUsername(username string) int64 {
	var count int64
	db.Model(&Tweet{}).Where("username = ?", username).Count(&count)
	return count
}

func getTweetsFromDB(username string, limit int) ([]twitterapi_reverse.SimpleTweet, error) {
	var dbTweets []Tweet
	result := db.Where("username = ?", username).Order("created_at DESC").Limit(limit).Find(&dbTweets)
	if result.Error != nil {
		return nil, result.Error
	}

	var tweets []twitterapi_reverse.SimpleTweet
	for _, dbTweet := range dbTweets {
		tweets = append(tweets, twitterapi_reverse.SimpleTweet{
			TweetID:   dbTweet.TweetID,
			Text:      dbTweet.Text,
			CreatedAt: dbTweet.CreatedAt,
		})
	}
	return tweets, nil
}
