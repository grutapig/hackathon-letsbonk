package main

import (
	"encoding/json"
	"fmt"
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

type Tweet struct {
	Id   string
	Text string
	Date string
}

func TestGetHistory(t *testing.T) {
	godotenv.Load()
	ts := twitterapi.NewTwitterAPIService(os.Getenv("twitter_api_key"), "https://api.twitterapi.io", os.Getenv("proxy_dsn"))
	username := "CDoughnath"
	var tweets []Tweet
	var cursor string
	for i := 0; i < 20; i++ {
		resp, err := ts.AdvancedSearch(twitterapi.AdvancedSearchRequest{
			Query:     "from:" + username,
			QueryType: twitterapi.LATEST,
			Cursor:    cursor,
		})
		if err != nil || len(resp.Tweets) == 0 || resp.NextCursor == "" {
			fmt.Println("no more, break", err)
			break
		}
		cursor = resp.NextCursor
		for n, tweet := range resp.Tweets {
			element := Tweet{Id: tweet.Id, Text: tweet.Text, Date: tweet.CreatedAt}
			tweets = append(tweets, element)
			fmt.Println(i, n, element)
		}
		fmt.Println("found tweet: ", len(resp.Tweets))
	}
	data, err := json.Marshal(tweets)
	assert.NoError(t, err)
	os.WriteFile("CDoughnath.json", data, 0655)
}
func TestAnalyzeHistory(t *testing.T) {
	data, err := os.ReadFile("CDoughnath.json")
	assert.NoError(t, err)
	var tweets []Tweet
	err = json.Unmarshal(data, &tweets)
	assert.NoError(t, err)
	//claudeApi :=
}
