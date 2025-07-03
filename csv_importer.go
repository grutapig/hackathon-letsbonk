package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type CSVImporter struct {
	dbService *DatabaseService
}

type CSVTweetData struct {
	AuthorUsername string
	TweetID        string
	AuthorID       string
	Date           string
	ReplyCount     int
	ReplyToID      string
	Text           string
}

func NewCSVImporter(dbService *DatabaseService) *CSVImporter {
	return &CSVImporter{
		dbService: dbService,
	}
}

func (c *CSVImporter) ImportCSV(csvFilePath string) (*ImportResult, error) {
	if _, err := os.Stat(csvFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("CSV file not found: %s", csvFilePath)
	}

	file, err := os.Open(csvFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	header := records[0]
	columnMap := c.mapColumns(header)

	if err := c.validateColumns(columnMap); err != nil {
		return nil, fmt.Errorf("CSV validation failed: %w", err)
	}

	tweetsData := []CSVTweetData{}
	for i, record := range records[1:] {
		if len(record) < len(header) {
			continue
		}

		replyCount, _ := strconv.Atoi(record[columnMap["reply_count"]])

		tweetData := CSVTweetData{
			AuthorUsername: record[columnMap["author_username"]],
			TweetID:        record[columnMap["tweet_id"]],
			AuthorID:       record[columnMap["author_id"]],
			Date:           record[columnMap["date"]],
			ReplyCount:     replyCount,
			ReplyToID:      record[columnMap["reply_to_id"]],
			Text:           record[columnMap["text"]],
		}

		tweetsData = append(tweetsData, tweetData)

		if (i+1)%1000 == 0 {
			fmt.Printf("Parsed %d tweets...\n", i+1)
		}
	}

	fmt.Printf("Found %d tweets to import\n", len(tweetsData))

	result := &ImportResult{}

	fmt.Println("Step 1: Importing original tweets...")
	for _, tweetData := range tweetsData {
		if tweetData.ReplyToID == "" {
			if c.importTweet(tweetData, "") {
				result.OriginalTweets++
			}
		}
	}

	fmt.Println("Step 2: Importing replies to existing tweets...")
	for _, tweetData := range tweetsData {
		if tweetData.ReplyToID != "" {
			if c.dbService.TweetExists(tweetData.ReplyToID) {
				if c.importTweet(tweetData, tweetData.ReplyToID) {
					result.ReplyTweets++
				}
			}
		}
	}

	fmt.Println("Step 3: Importing remaining tweets...")
	remainingTweets := []CSVTweetData{}

	for _, tweetData := range tweetsData {
		if tweetData.ReplyToID != "" && !c.dbService.TweetExists(tweetData.TweetID) {
			remainingTweets = append(remainingTweets, tweetData)
		}
	}

	maxIterations := 10
	iteration := 0

	for len(remainingTweets) > 0 && iteration < maxIterations {
		iteration++
		fmt.Printf("  Iteration %d: %d tweets remaining\n", iteration, len(remainingTweets))

		importedThisRound := 0
		newRemaining := []CSVTweetData{}

		for _, tweetData := range remainingTweets {
			if c.dbService.TweetExists(tweetData.ReplyToID) {
				if c.importTweet(tweetData, tweetData.ReplyToID) {
					importedThisRound++
					result.RemainingTweets++
				}
			} else {
				newRemaining = append(newRemaining, tweetData)
			}
		}

		remainingTweets = newRemaining

		if importedThisRound == 0 {
			break
		}
	}

	result.SkippedTweets = len(remainingTweets)
	result.TotalProcessed = result.OriginalTweets + result.ReplyTweets + result.RemainingTweets

	return result, nil
}

func (c *CSVImporter) mapColumns(header []string) map[string]int {
	columnMap := make(map[string]int)

	for i, col := range header {
		col = strings.TrimSpace(col)
		switch col {
		case "author_username", "username":
			columnMap["author_username"] = i
		case "tweet_id", "id":
			columnMap["tweet_id"] = i
		case "author_id", "user_id":
			columnMap["author_id"] = i
		case "date", "created_at":
			columnMap["date"] = i
		case "reply_count", "replies":
			columnMap["reply_count"] = i
		case "reply_to_id", "in_reply_to":
			columnMap["reply_to_id"] = i
		case "text", "content":
			columnMap["text"] = i
		}
	}

	return columnMap
}

func (c *CSVImporter) validateColumns(columnMap map[string]int) error {
	required := []string{"author_username", "tweet_id", "author_id", "date", "text"}

	for _, field := range required {
		if _, exists := columnMap[field]; !exists {
			return fmt.Errorf("required column not found: %s", field)
		}
	}

	return nil
}

func (c *CSVImporter) importTweet(tweetData CSVTweetData, replyToID string) bool {
	if c.dbService.TweetExists(tweetData.TweetID) {
		return false
	}

	if !c.dbService.UserExists(tweetData.AuthorID) {
		user := UserModel{
			ID:       tweetData.AuthorID,
			Username: tweetData.AuthorUsername,
			Name:     tweetData.AuthorUsername,
		}
		err := c.dbService.SaveUser(user)
		if err != nil {
			fmt.Printf("Error saving user %s: %v\n", tweetData.AuthorUsername, err)
			return false
		}
	}

	createdAt, err := c.parseDate(tweetData.Date)
	if err != nil {
		fmt.Printf("Error parsing date %s: %v\n", tweetData.Date, err)
		createdAt = time.Now()
	}

	tweet := TweetModel{
		ID:          tweetData.TweetID,
		Text:        tweetData.Text,
		CreatedAt:   createdAt,
		UpdatedAt:   time.Now(),
		ReplyCount:  tweetData.ReplyCount,
		UserID:      tweetData.AuthorID,
		InReplyToID: replyToID,
		SourceType:  TWEET_SOURCE_COMMUNITY,
	}

	err = c.dbService.SaveTweet(tweet)
	if err != nil {
		fmt.Printf("Error saving tweet %s: %v\n", tweetData.TweetID, err)
		return false
	}

	return true
}

func (c *CSVImporter) parseDate(dateStr string) (time.Time, error) {
	formats := []string{
		"Mon Jan 02 15:04:05 -0700 2006", // Twitter format
		"2006-01-02 15:04:05",            // SQL format
		"2006-01-02T15:04:05Z",           // ISO format
		"2006-01-02",                     // Date only
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

type ImportResult struct {
	OriginalTweets  int
	ReplyTweets     int
	RemainingTweets int
	SkippedTweets   int
	TotalProcessed  int
}

func (r *ImportResult) String() string {
	return fmt.Sprintf("Import Result:\n  Original tweets: %d\n  Reply tweets: %d\n  Remaining tweets: %d\n  Skipped tweets: %d\n  Total processed: %d",
		r.OriginalTweets, r.ReplyTweets, r.RemainingTweets, r.SkippedTweets, r.TotalProcessed)
}
