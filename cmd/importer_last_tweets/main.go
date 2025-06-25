package main

import (
	"encoding/csv"
	"fmt"
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/joho/godotenv"
	"os"
	"sort"
	"strconv"
	"sync"
)

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	godotenv.Load("../.env")
	r, err := os.OpenFile("community_tweets.csv", os.O_RDONLY, 0655)
	panicErr(err)
	cursor := csv.NewReader(r)
	rows, err := cursor.ReadAll()
	panicErr(err)
	authorsMap := map[string]bool{}
	for _, row := range rows[1:] {
		authorsMap[row[0]] = true
	}
	fmt.Println(len(authorsMap))
	api := twitterapi.NewTwitterAPIService(os.Getenv(twitterapi.ENV_TWITTER_API_KEY), os.Getenv(twitterapi.ENV_TWITTER_API_BASE_URL), os.Getenv(twitterapi.ENV_PROXY_DSN))
	//os.RemoveAll("last_tweets.csv")
	w, err := os.OpenFile("last_tweets.csv", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0655)
	panicErr(err)
	writer := csv.NewWriter(w)
	//writer.Write([]string{"author_name", "author_id", "tweet_id", "text", "created_at", "reply_count"})
	//writer.Flush()
	authors := []string{}
	for authorName, _ := range authorsMap {
		authors = append(authors, authorName)
	}
	start := 16
	fmt.Println("started from ", start)
	sort.Strings(authors)
	authorsCh := make(chan string)
	resultCh := make(chan struct {
		string
		twitterapi.UserLastTweetsResponse
		error
	})
	wgParallel := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wgParallel.Add(1)
		go func() {
			defer wgParallel.Done()
			for authorName := range authorsCh {
				result, err := api.GetUserLastTweets(twitterapi.UserLastTweetsRequest{UserName: authorName, IncludeReplies: true})
				resultCh <- struct {
					string
					twitterapi.UserLastTweetsResponse
					error
				}{authorName, *result, err}
			}
		}()
	}
	go func() {
		wgParallel.Wait()
		close(resultCh)
	}()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for resultStruct := range resultCh {
			result := resultStruct.UserLastTweetsResponse
			err := resultStruct.error
			authorName := resultStruct.string
			panicErr(err)
			fmt.Println(authorName, "found tweets", len(result.Data.Tweets))
			for _, tweet := range result.Data.Tweets {
				writer.Write([]string{tweet.Author.UserName, tweet.Author.Id, tweet.Id, tweet.Text, tweet.CreatedAt, strconv.Itoa(tweet.ReplyCount)})
			}
			writer.Flush()
			fmt.Println(authorName, "written to csv")
		}
	}()

	for i, authorName := range authors {
		if i < start {
			continue
		}
		fmt.Println(i, "request for author...", authorName)
		authorsCh <- authorName
	}
	close(authorsCh)
	wg.Wait()
}
