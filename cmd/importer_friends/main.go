package main

import (
	"encoding/csv"
	"fmt"
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/joho/godotenv"
	"os"
	"sort"
	"sync"
	"time"
)

type FriendResult struct {
	Username   string
	UserID     string
	FriendType string
	Friends    []twitterapi.User
	Error      error
}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	err := godotenv.Load()
	panicErr(err)

	fmt.Println("📖 Reading community_tweets.csv...")
	r, err := os.OpenFile("community_tweets.csv", os.O_RDONLY, 0655)
	panicErr(err)
	defer r.Close()

	cursor := csv.NewReader(r)
	rows, err := cursor.ReadAll()
	panicErr(err)

	authorsMap := map[string]string{}
	for _, row := range rows[1:] {
		if len(row) >= 7 {
			username := row[0]
			userID := row[6]
			authorsMap[username] = userID
		}
	}

	fmt.Printf("📊 Found %d unique authors in community\n", len(authorsMap))

	api := twitterapi.NewTwitterAPIService(
		os.Getenv(twitterapi.ENV_TWITTER_API_KEY),
		os.Getenv(twitterapi.ENV_TWITTER_API_BASE_URL),
		os.Getenv(twitterapi.ENV_PROXY_DSN),
	)

	w, err := os.OpenFile("friends_data.csv", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0655)
	panicErr(err)
	defer w.Close()

	writer := csv.NewWriter(w)
	defer writer.Flush()

	headers := []string{
		"user_username",
		"user_id",
		"friend_username",
		"friend_id",
		"friend_name",
		"friend_type",
		"scraped_at",
	}
	err = writer.Write(headers)
	panicErr(err)
	writer.Flush()

	authors := make([]string, 0, len(authorsMap))
	for username := range authorsMap {
		authors = append(authors, username)
	}
	sort.Strings(authors)

	userCh := make(chan struct {
		Username string
		UserID   string
	}, 100)

	resultCh := make(chan FriendResult, 100)

	wgWorkers := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wgWorkers.Add(1)
		go func(workerID int) {
			defer wgWorkers.Done()
			for user := range userCh {
				fmt.Printf("🔄 Worker %d: Processing @%s\n", workerID, user.Username)

				followersResult := FriendResult{
					Username:   user.Username,
					UserID:     user.UserID,
					FriendType: "follower",
				}

				err := retryRequest(func() error {
					followers, requestErr := api.GetUserFollowers(twitterapi.UserFollowersRequest{
						UserName: user.Username,
						PageSize: 200,
					})
					if requestErr == nil && followers != nil {
						followersResult.Friends = followers.Followers
					}
					return requestErr
				}, fmt.Sprintf("getting followers for @%s", user.Username))

				if err != nil {
					followersResult.Error = err
					fmt.Printf("❌ Worker %d: Failed to get followers for @%s: %v\n", workerID, user.Username, err)
				} else {
					fmt.Printf("✅ Worker %d: Got %d followers for @%s\n", workerID, len(followersResult.Friends), user.Username)
				}

				resultCh <- followersResult

				followingsResult := FriendResult{
					Username:   user.Username,
					UserID:     user.UserID,
					FriendType: "following",
				}

				err = retryRequest(func() error {
					followings, requestErr := api.GetUserFollowings(twitterapi.UserFollowingsRequest{
						UserName: user.Username,
						PageSize: 200,
					})
					if requestErr == nil && followings != nil {
						followingsResult.Friends = followings.Followings
					}
					return requestErr
				}, fmt.Sprintf("getting followings for @%s", user.Username))

				if err != nil {
					followingsResult.Error = err
					fmt.Printf("❌ Worker %d: Failed to get followings for @%s: %v\n", workerID, user.Username, err)
				} else {
					fmt.Printf("✅ Worker %d: Got %d followings for @%s\n", workerID, len(followingsResult.Friends), user.Username)
				}

				resultCh <- followingsResult
			}
		}(i)
	}

	wgProcessor := sync.WaitGroup{}
	wgProcessor.Add(1)

	totalFriends := 0
	processedUsers := 0
	startTime := time.Now()

	go func() {
		defer wgProcessor.Done()
		for result := range resultCh {
			if result.Error != nil {
				fmt.Printf("⚠️  Skipping %s %s for @%s due to error: %v\n",
					result.FriendType, result.Username, result.Username, result.Error)
				continue
			}

			scrapedAt := time.Now().Format(time.RFC3339)

			for _, friend := range result.Friends {
				record := []string{
					result.Username,
					result.UserID,
					friend.UserName,
					friend.Id,
					friend.Name,
					result.FriendType,
					scrapedAt,
				}

				err := writer.Write(record)
				if err != nil {
					fmt.Printf("❌ Error writing friend data: %v\n", err)
					continue
				}
				totalFriends++
			}

			writer.Flush()

			if result.FriendType == "following" {
				processedUsers++
				elapsed := time.Since(startTime)
				avgTime := elapsed / time.Duration(processedUsers)
				remaining := len(authors) - processedUsers
				eta := time.Duration(remaining) * avgTime

				fmt.Printf("📊 Progress: %d/%d users (%d friends total) | Avg: %v/user | ETA: %v\n",
					processedUsers, len(authors), totalFriends, avgTime, eta)
			}
		}
	}()

	go func() {
		wgWorkers.Wait()
		close(resultCh)
	}()

	fmt.Println("🚀 Starting friends import with 10 parallel workers...")
	for i, username := range authors {
		userID := authorsMap[username]
		fmt.Printf("📤 Queuing user %d/%d: @%s (ID: %s)\n", i+1, len(authors), username, userID)

		userCh <- struct {
			Username string
			UserID   string
		}{username, userID}
	}

	close(userCh)
	wgProcessor.Wait()

	totalDuration := time.Since(startTime)
	fmt.Printf("🎉 Friends import completed!\n")
	fmt.Printf("📈 Final statistics:\n")
	fmt.Printf("   - Processed users: %d\n", processedUsers)
	fmt.Printf("   - Total friends scraped: %d\n", totalFriends)
	fmt.Printf("   - Total runtime: %v\n", totalDuration)
	fmt.Printf("   - Average friends per user: %.1f\n", float64(totalFriends)/float64(processedUsers))
	fmt.Printf("   - Average speed: %.2f users/min\n", float64(processedUsers)/totalDuration.Minutes())
	fmt.Println("💾 Data saved to friends_data.csv")
}

func retryRequest(requestFunc func() error, description string) error {
	maxRetries := 5
	for attempt := 1; attempt <= maxRetries; attempt++ {
		startTime := time.Now()
		err := requestFunc()
		duration := time.Since(startTime)

		if err == nil {
			if attempt > 1 {
				fmt.Printf("✅ Request successful on attempt %d in %v: %s\n", attempt, duration, description)
			}
			return nil
		}

		fmt.Printf("❌ Attempt %d/%d failed in %v for %s: %v\n", attempt, maxRetries, duration, description, err)

		if attempt < maxRetries {
			waitTime := time.Duration(attempt*2) * time.Second
			fmt.Printf("⏳ Waiting %v before next attempt...\n", waitTime)
			time.Sleep(waitTime)
		}
	}

	return fmt.Errorf("all %d attempts failed for %s", maxRetries, description)
}
