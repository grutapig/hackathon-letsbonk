package twitterapi_reverse

import (
	"fmt"
	"log"
	"time"

	"github.com/buger/jsonparser"
)

type Tweet struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	FullText          string    `json:"full_text"`
	ReplyCount        int64     `json:"reply_count"`
	CreatedAt         time.Time `json:"created_at"`
	InReplyToStatusID string    `json:"in_reply_to_status_id_str"`
	Author            Author    `json:"author"`
}

type Author struct {
	ID         string    `json:"id"`
	ScreenName string    `json:"screen_name"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
}

func ParseCommunityTweets(data []byte) ([]Tweet, error) {
	var tweets []Tweet
	var parseErrors []string

	instructionsPath := []string{"data", "communityResults", "result", "ranked_community_timeline", "timeline", "instructions"}
	_, err := jsonparser.ArrayEach(data, func(instruction []byte, dataType jsonparser.ValueType, instructionOffset int, err error) {
		if err != nil {
			log.Println("some error", err)
			parseErrors = append(parseErrors, fmt.Sprintf("ArrayEach instruction callback error at offset %d: %v", instructionOffset, err))
			return
		}

		entriesPath := []string{"entries"}
		_, entriesErr := jsonparser.ArrayEach(instruction, func(entry []byte, dataType jsonparser.ValueType, entryOffset int, err error) {
			if err != nil {
				log.Println("some error 2", err)
				parseErrors = append(parseErrors, fmt.Sprintf("ArrayEach entry callback error at instruction %d, entry offset %d: %v", instructionOffset, entryOffset, err))
				return
			}

			tweetResultsPath := []string{"content", "itemContent", "tweet_results", "result"}
			tweetResultsData, _, _, err := jsonparser.Get(entry, tweetResultsPath...)
			if err != nil {
				log.Println("some error3", err)
				return
			}

			tweet := Tweet{}

			idPath := []string{"legacy", "id_str"}
			if id, err := jsonparser.GetString(tweetResultsData, idPath...); err == nil {
				tweet.ID = id
			} else {
				log.Println("some error4", err)
				parseErrors = append(parseErrors, fmt.Sprintf("Failed to get tweet ID at path %v: %v", idPath, err))
			}

			userIDPath := []string{"legacy", "user_id_str"}
			if userID, err := jsonparser.GetString(tweetResultsData, userIDPath...); err == nil {
				tweet.UserID = userID
			} else {
				log.Println("some error5", err)
				parseErrors = append(parseErrors, fmt.Sprintf("Failed to get user ID at path %v: %v", userIDPath, err))
			}

			fullTextPath := []string{"legacy", "full_text"}
			if fullText, err := jsonparser.GetString(tweetResultsData, fullTextPath...); err == nil {
				tweet.FullText = fullText
			} else {
				log.Println("some error6", err)
				parseErrors = append(parseErrors, fmt.Sprintf("Failed to get full text at path %v: %v", fullTextPath, err))
			}

			replyCountPath := []string{"legacy", "reply_count"}
			if replyCount, err := jsonparser.GetInt(tweetResultsData, replyCountPath...); err == nil {
				tweet.ReplyCount = replyCount
			} else {
				log.Println("some error7", err)
				parseErrors = append(parseErrors, fmt.Sprintf("Failed to get reply count at path %v: %v", replyCountPath, err))
			}

			inReplyToStatusIDPath := []string{"legacy", "in_reply_to_status_id_str"}
			if inReplyToStatusID, err := jsonparser.GetString(tweetResultsData, inReplyToStatusIDPath...); err == nil {
				tweet.InReplyToStatusID = inReplyToStatusID
			} else {
				log.Println("in_reply_to_status_id_str parse info (normal for non-replies)", err)
			}

			createdAtPath := []string{"legacy", "created_at"}
			if createdAtStr, err := jsonparser.GetString(tweetResultsData, createdAtPath...); err == nil {
				if parsedTime, err := ParseTwitterTime(createdAtStr); err == nil {
					tweet.CreatedAt = parsedTime
				} else {
					log.Println("some error8", err)
					parseErrors = append(parseErrors, fmt.Sprintf("Failed to parse created_at time '%s': %v", createdAtStr, err))
				}
			} else {
				log.Println("some error9", err)
				parseErrors = append(parseErrors, fmt.Sprintf("Failed to get created_at at path %v: %v", createdAtPath, err))
			}

			authorPath := []string{"author_community_relationship", "user_results", "result"}
			authorData, _, _, err := jsonparser.Get(tweetResultsData, authorPath...)
			if err != nil {
				log.Println("some error10", err)
				parseErrors = append(parseErrors, fmt.Sprintf("Failed to get author data at path %v: %v", authorPath, err))
				return
			}

			tweet.Author.ID = tweet.UserID

			screenNamePath := []string{"core", "screen_name"}
			if screenName, err := jsonparser.GetString(authorData, screenNamePath...); err == nil {
				tweet.Author.ScreenName = screenName
			} else {
				parseErrors = append(parseErrors, fmt.Sprintf("Failed to get screen name at path %v: %v", screenNamePath, err))
			}

			namePath := []string{"core", "name"}
			if name, err := jsonparser.GetString(authorData, namePath...); err == nil {
				tweet.Author.Name = name
			} else {
				parseErrors = append(parseErrors, fmt.Sprintf("Failed to get author name at path %v: %v", namePath, err))
			}

			authorCreatedAtPath := []string{"core", "created_at"}
			if authorCreatedAtStr, err := jsonparser.GetString(authorData, authorCreatedAtPath...); err == nil {
				if parsedTime, err := ParseTwitterTime(authorCreatedAtStr); err == nil {
					tweet.Author.CreatedAt = parsedTime
				} else {
					parseErrors = append(parseErrors, fmt.Sprintf("Failed to parse author created_at time '%s': %v", authorCreatedAtStr, err))
				}
			} else {
				parseErrors = append(parseErrors, fmt.Sprintf("Failed to get author created_at at path %v: %v", authorCreatedAtPath, err))
			}

			tweets = append(tweets, tweet)
		}, entriesPath...)

		if entriesErr != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("Failed to parse entries array in instruction %d: %v, path: %s, raw: %s", instructionOffset, entriesErr, entriesPath, instruction))
		}
	}, instructionsPath...)

	if err != nil {
		return nil, fmt.Errorf("failed to parse instructions array at path %v: %v", instructionsPath, err)
	}

	if len(parseErrors) > 0 {
		return tweets, fmt.Errorf("parsing completed with %d errors: %v", len(parseErrors), parseErrors)
	}

	return tweets, nil
}

func ParseThreadedConversationTweets(data []byte) ([]Tweet, error) {
	var tweets []Tweet
	var parseErrors []string

	instructionsPath := []string{"data", "threaded_conversation_with_injections_v2", "instructions"}
	_, err := jsonparser.ArrayEach(data, func(instruction []byte, dataType jsonparser.ValueType, instructionOffset int, err error) {
		if err != nil {
			log.Println("threaded conversation instruction error", err)
			parseErrors = append(parseErrors, fmt.Sprintf("ArrayEach instruction callback error at offset %d: %v", instructionOffset, err))
			return
		}

		entriesPath := []string{"entries"}
		_, entriesErr := jsonparser.ArrayEach(instruction, func(entry []byte, dataType jsonparser.ValueType, entryOffset int, err error) {
			if err != nil {
				log.Println("threaded conversation entry error", err)
				parseErrors = append(parseErrors, fmt.Sprintf("ArrayEach entry callback error at instruction %d, entry offset %d: %v", instructionOffset, entryOffset, err))
				return
			}

			tweetResultsPath := []string{"content", "itemContent", "tweet_results", "result"}
			tweetResultsData, _, _, err := jsonparser.Get(entry, tweetResultsPath...)
			if err == nil {
				tweet := parseTweetData(tweetResultsData, &parseErrors)
				if tweet.ID != "" {
					tweets = append(tweets, tweet)
				}
				return
			}

			itemsPath := []string{"content", "items"}
			_, itemsErr := jsonparser.ArrayEach(entry, func(item []byte, dataType jsonparser.ValueType, itemOffset int, err error) {
				if err != nil {
					log.Println("threaded conversation item error", err)
					parseErrors = append(parseErrors, fmt.Sprintf("ArrayEach item callback error at entry %d, item offset %d: %v", entryOffset, itemOffset, err))
					return
				}

				itemTweetResultsPath := []string{"item", "itemContent", "tweet_results", "result"}
				itemTweetResultsData, _, _, err := jsonparser.Get(item, itemTweetResultsPath...)
				if err == nil {
					tweet := parseTweetData(itemTweetResultsData, &parseErrors)
					if tweet.ID != "" {
						tweets = append(tweets, tweet)
					}
				}
			}, itemsPath...)

			if itemsErr != nil {
				log.Println("threaded conversation items parsing error (normal)", itemsErr)
			}
		}, entriesPath...)

		if entriesErr != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("Failed to parse entries array in instruction %d: %v", instructionOffset, entriesErr))
		}
	}, instructionsPath...)

	if err != nil {
		return nil, fmt.Errorf("failed to parse instructions array at path %v: %v", instructionsPath, err)
	}

	if len(parseErrors) > 0 {
		return tweets, fmt.Errorf("parsing completed with %d errors: %v", len(parseErrors), parseErrors)
	}

	return tweets, nil
}

func parseTweetData(tweetResultsData []byte, parseErrors *[]string) Tweet {
	tweet := Tweet{}

	idPath := []string{"legacy", "id_str"}
	if id, err := jsonparser.GetString(tweetResultsData, idPath...); err == nil {
		tweet.ID = id
	} else {
		log.Println("tweet ID parse error", err)
		*parseErrors = append(*parseErrors, fmt.Sprintf("Failed to get tweet ID at path %v: %v", idPath, err))
	}

	userIDPath := []string{"legacy", "user_id_str"}
	if userID, err := jsonparser.GetString(tweetResultsData, userIDPath...); err == nil {
		tweet.UserID = userID
	} else {
		log.Println("user ID parse error", err)
		*parseErrors = append(*parseErrors, fmt.Sprintf("Failed to get user ID at path %v: %v", userIDPath, err))
	}

	fullTextPath := []string{"legacy", "full_text"}
	if fullText, err := jsonparser.GetString(tweetResultsData, fullTextPath...); err == nil {
		tweet.FullText = fullText
	} else {
		log.Println("full text parse error", err)
		*parseErrors = append(*parseErrors, fmt.Sprintf("Failed to get full text at path %v: %v", fullTextPath, err))
	}

	replyCountPath := []string{"legacy", "reply_count"}
	if replyCount, err := jsonparser.GetInt(tweetResultsData, replyCountPath...); err == nil {
		tweet.ReplyCount = replyCount
	} else {
		log.Println("reply count parse error", err)
		*parseErrors = append(*parseErrors, fmt.Sprintf("Failed to get reply count at path %v: %v", replyCountPath, err))
	}

	inReplyToStatusIDPath := []string{"legacy", "in_reply_to_status_id_str"}
	if inReplyToStatusID, err := jsonparser.GetString(tweetResultsData, inReplyToStatusIDPath...); err == nil {
		tweet.InReplyToStatusID = inReplyToStatusID
	} else {
		log.Println("in_reply_to_status_id_str parse info (normal for non-replies)", err)
	}

	createdAtPath := []string{"legacy", "created_at"}
	if createdAtStr, err := jsonparser.GetString(tweetResultsData, createdAtPath...); err == nil {
		if parsedTime, err := ParseTwitterTime(createdAtStr); err == nil {
			tweet.CreatedAt = parsedTime
		} else {
			log.Println("created_at parse error", err)
			*parseErrors = append(*parseErrors, fmt.Sprintf("Failed to parse created_at time '%s': %v", createdAtStr, err))
		}
	} else {
		log.Println("created_at get error", err)
		*parseErrors = append(*parseErrors, fmt.Sprintf("Failed to get created_at at path %v: %v", createdAtPath, err))
	}

	authorPath := []string{"core", "user_results", "result"}
	authorData, _, _, err := jsonparser.Get(tweetResultsData, authorPath...)
	if err != nil {
		log.Println("author data parse error", err)
		*parseErrors = append(*parseErrors, fmt.Sprintf("Failed to get author data at path %v: %v", authorPath, err))
		return tweet
	}

	tweet.Author.ID = tweet.UserID

	screenNamePath := []string{"core", "screen_name"}
	if screenName, err := jsonparser.GetString(authorData, screenNamePath...); err == nil {
		tweet.Author.ScreenName = screenName
	} else {
		*parseErrors = append(*parseErrors, fmt.Sprintf("Failed to get screen name at path %v: %v", screenNamePath, err))
	}

	namePath := []string{"core", "name"}
	if name, err := jsonparser.GetString(authorData, namePath...); err == nil {
		tweet.Author.Name = name
	} else {
		*parseErrors = append(*parseErrors, fmt.Sprintf("Failed to get author name at path %v: %v", namePath, err))
	}

	authorCreatedAtPath := []string{"core", "created_at"}
	if authorCreatedAtStr, err := jsonparser.GetString(authorData, authorCreatedAtPath...); err == nil {
		if parsedTime, err := ParseTwitterTime(authorCreatedAtStr); err == nil {
			tweet.Author.CreatedAt = parsedTime
		} else {
			*parseErrors = append(*parseErrors, fmt.Sprintf("Failed to parse author created_at time '%s': %v", authorCreatedAtStr, err))
		}
	} else {
		*parseErrors = append(*parseErrors, fmt.Sprintf("Failed to get author created_at at path %v: %v", authorCreatedAtPath, err))
	}

	return tweet
}

func ParseTwitterTime(timeStr string) (time.Time, error) {
	layout := "Mon Jan 02 15:04:05 -0700 2006"
	return time.Parse(layout, timeStr)
}
