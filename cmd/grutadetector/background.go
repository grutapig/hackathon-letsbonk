package main

import (
	"github.com/grutapig/hackaton/twitterapi"
	"log"
)

func getUserTweetsUnlimited(username string) bool {
	var cursor string
	pageCount := 0

	for {
		select {
		case <-stopCurrentJob:
			log.Printf("Stop signal received for @%s at page %d", username, pageCount)
			return false
		default:
		}

		resp, err := twitterApi.AdvancedSearch(twitterapi.AdvancedSearchRequest{
			Query:     "from:" + username,
			QueryType: twitterapi.LATEST,
			Cursor:    cursor,
		})

		if err != nil || len(resp.Tweets) == 0 || resp.NextCursor == "" {
			break
		}

		cursor = resp.NextCursor
		pageCount++

		newTweetsSaved := 0
		for _, tweet := range resp.Tweets {
			if saveTweetToDB(tweet) {
				newTweetsSaved++
			}
		}

		if newTweetsSaved == 0 {
			log.Printf("No new tweets saved for @%s at page %d, breaking", username, pageCount)
			break
		}

		if pageCount%10 == 0 {
			log.Printf("Unlimited parsing for @%s: processed %d pages", username, pageCount)
		}
	}

	cursor = ""
	if pageCount == 0 {
		for {
			select {
			case <-stopCurrentJob:
				log.Printf("Stop signal received for @%s (TOP search) at page %d", username, pageCount)
				return false
			default:
			}

			resp, err := twitterApi.AdvancedSearch(twitterapi.AdvancedSearchRequest{
				Query:     "from:" + username,
				QueryType: twitterapi.TOP,
				Cursor:    cursor,
			})

			if err != nil || len(resp.Tweets) == 0 || resp.NextCursor == "" {
				break
			}

			cursor = resp.NextCursor
			pageCount++

			newTweetsSaved := 0
			for _, tweet := range resp.Tweets {
				if saveTweetToDB(tweet) {
					newTweetsSaved++
				}
			}

			if newTweetsSaved == 0 {
				log.Printf("No new tweets saved for @%s (TOP search) at page %d, breaking", username, pageCount)
				break
			}

			if pageCount%10 == 0 {
				log.Printf("Unlimited parsing (TOP) for @%s: processed %d pages", username, pageCount)
			}
		}
	}

	log.Printf("Completed unlimited parsing for @%s: total %d pages", username, pageCount)
	return true
}

func backgroundParseWorker() {
	for username := range parseQueue {
		currentlyParsing = username
		isWorkerBusy = true
		log.Printf("Starting background unlimited parsing for @%s", username)

		completed := getUserTweetsUnlimited(username)

		if completed {
			log.Printf("Completed background unlimited parsing for @%s", username)
			setUserFullyParsed(username, true)
		} else {
			log.Printf("Stopped background unlimited parsing for @%s", username)
		}

		currentlyParsing = ""
		isWorkerBusy = false

		select {
		case <-stopCurrentJob:
		default:
		}
	}
}
