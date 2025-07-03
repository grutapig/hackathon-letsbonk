package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DatabaseService struct {
	db *gorm.DB
	// In-memory storage for user data (temporary for debugging)
	users         map[string]*UserModel
	userAnalyzed  map[string]bool
	fudUsers      map[string]*FUDUserModel
	userRelations map[string][]UserRelationModel
	userDataMutex sync.RWMutex
}

// NewDatabaseService creates a new database service instance
func NewDatabaseService(dbPath string) (*DatabaseService, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Silent to reduce log noise
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	service := &DatabaseService{
		db:            db,
		users:         make(map[string]*UserModel),
		userAnalyzed:  make(map[string]bool),
		fudUsers:      make(map[string]*FUDUserModel),
		userRelations: make(map[string][]UserRelationModel),
	}

	// Run migrations
	if err := service.runMigrations(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return service, nil
}

// runMigrations runs database migrations
func (s *DatabaseService) runMigrations() error {
	return s.db.AutoMigrate(&TweetModel{}, &UserModel{}, &FUDUserModel{}, &UserRelationModel{})
}

// Tweet related methods

// SaveTweet saves or updates a tweet in the database
func (s *DatabaseService) SaveTweet(tweet TweetModel) error {
	tweet.UpdatedAt = time.Now()
	return s.db.Save(&tweet).Error
}

// GetTweet retrieves a tweet by Twitter ID (not auto_id)
func (s *DatabaseService) GetTweet(id string) (*TweetModel, error) {
	var tweet TweetModel
	err := s.db.Where("id = ?", id).First(&tweet).Error
	if err != nil {
		return nil, err
	}
	return &tweet, nil
}

// GetTweetByAutoID retrieves a tweet by auto-incrementing ID
func (s *DatabaseService) GetTweetByAutoID(autoID uint) (*TweetModel, error) {
	var tweet TweetModel
	err := s.db.Where("auto_id = ?", autoID).First(&tweet).Error
	if err != nil {
		return nil, err
	}
	return &tweet, nil
}

// TweetExists checks if a tweet exists by Twitter ID (not auto_id)
func (s *DatabaseService) TweetExists(id string) bool {
	var count int64
	s.db.Model(&TweetModel{}).Where("id = ?", id).Count(&count)
	return count > 0
}

// GetTweetReplyCount gets the reply count for a tweet by Twitter ID
func (s *DatabaseService) GetTweetReplyCount(id string) (int, error) {
	var tweet TweetModel
	err := s.db.Select("reply_count").Where("id = ?", id).First(&tweet).Error
	if err != nil {
		return 0, err
	}
	return tweet.ReplyCount, nil
}

// UpdateTweetReplyCount updates the reply count for a tweet by Twitter ID
func (s *DatabaseService) UpdateTweetReplyCount(id string, replyCount int) error {
	return s.db.Model(&TweetModel{}).Where("id = ?", id).Update("reply_count", replyCount).Error
}

// GetTweetsByUser retrieves all tweets by a specific user
func (s *DatabaseService) GetTweetsByUser(userID string) ([]TweetModel, error) {
	var tweets []TweetModel
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&tweets).Error
	return tweets, err
}

// GetRepliesForTweet retrieves all replies for a specific tweet
func (s *DatabaseService) GetRepliesForTweet(tweetID string) ([]TweetModel, error) {
	var replies []TweetModel
	err := s.db.Where("in_reply_to_id = ?", tweetID).Order("created_at ASC").Find(&replies).Error
	return replies, err
}

// DeleteTweet deletes a tweet by Twitter ID
func (s *DatabaseService) DeleteTweet(id string) error {
	return s.db.Delete(&TweetModel{}, "id = ?", id).Error
}

// User related methods

// SaveUser saves or updates a user in memory (temporary for debugging)
func (s *DatabaseService) SaveUser(user UserModel) error {
	s.userDataMutex.Lock()
	defer s.userDataMutex.Unlock()
	user.UpdatedAt = time.Now()
	s.users[user.ID] = &user
	return nil
}

// GetUser retrieves a user by ID from memory
func (s *DatabaseService) GetUser(id string) (*UserModel, error) {
	s.userDataMutex.RLock()
	defer s.userDataMutex.RUnlock()
	user, exists := s.users[id]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// GetUserByUsername retrieves a user by username from memory
func (s *DatabaseService) GetUserByUsername(username string) (*UserModel, error) {
	s.userDataMutex.RLock()
	defer s.userDataMutex.RUnlock()
	for _, user := range s.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

// UserExists checks if a user exists in memory
func (s *DatabaseService) UserExists(id string) bool {
	s.userDataMutex.RLock()
	defer s.userDataMutex.RUnlock()
	_, exists := s.users[id]
	return exists
}

// UserExistsByUsername checks if a user exists by username in memory
func (s *DatabaseService) UserExistsByUsername(username string) bool {
	s.userDataMutex.RLock()
	defer s.userDataMutex.RUnlock()
	for _, user := range s.users {
		if user.Username == username {
			return true
		}
	}
	return false
}

// DeleteUser deletes a user from the database
func (s *DatabaseService) DeleteUser(id string) error {
	return s.db.Delete(&UserModel{}, "id = ?", id).Error
}

// FUD User related methods

// SaveFUDUser saves or updates a FUD user in memory
func (s *DatabaseService) SaveFUDUser(fudUser FUDUserModel) error {
	s.userDataMutex.Lock()
	defer s.userDataMutex.Unlock()
	fudUser.UpdatedAt = time.Now()
	s.fudUsers[fudUser.UserID] = &fudUser
	return nil
}

// GetFUDUser retrieves a FUD user by user ID from memory
func (s *DatabaseService) GetFUDUser(userID string) (*FUDUserModel, error) {
	s.userDataMutex.RLock()
	defer s.userDataMutex.RUnlock()
	fudUser, exists := s.fudUsers[userID]
	if !exists {
		return nil, fmt.Errorf("FUD user not found")
	}
	return fudUser, nil
}

// IsFUDUser checks if a user is marked as FUD in memory
func (s *DatabaseService) IsFUDUser(userID string) bool {
	s.userDataMutex.RLock()
	defer s.userDataMutex.RUnlock()
	_, exists := s.fudUsers[userID]
	return exists
}

// GetAllFUDUsers retrieves all FUD users
func (s *DatabaseService) GetAllFUDUsers() ([]FUDUserModel, error) {
	var fudUsers []FUDUserModel
	err := s.db.Order("detected_at DESC").Find(&fudUsers).Error
	return fudUsers, err
}

// IncrementFUDUserMessageCount increments the message count for a FUD user in memory
func (s *DatabaseService) IncrementFUDUserMessageCount(userID string, messageID string) error {
	s.userDataMutex.Lock()
	defer s.userDataMutex.Unlock()
	fudUser, exists := s.fudUsers[userID]
	if !exists {
		return fmt.Errorf("FUD user not found")
	}
	fudUser.MessageCount++
	fudUser.LastMessageID = messageID
	fudUser.UpdatedAt = time.Now()
	return nil
}

// DeleteFUDUser deletes a FUD user from the database
func (s *DatabaseService) DeleteFUDUser(userID string) error {
	return s.db.Delete(&FUDUserModel{}, "user_id = ?", userID).Error
}

// Search and query methods

// SearchTweets searches tweets by text content
func (s *DatabaseService) SearchTweets(query string, limit int) ([]TweetModel, error) {
	var tweets []TweetModel
	err := s.db.Where("text LIKE ?", "%"+query+"%").Limit(limit).Order("created_at DESC").Find(&tweets).Error
	return tweets, err
}

// GetRecentTweets retrieves recent tweets with optional limit
func (s *DatabaseService) GetRecentTweets(limit int) ([]TweetModel, error) {
	var tweets []TweetModel
	err := s.db.Order("created_at DESC").Limit(limit).Find(&tweets).Error
	return tweets, err
}

// GetTweetCount returns the total number of tweets in the database
func (s *DatabaseService) GetTweetCount() (int64, error) {
	var count int64
	err := s.db.Model(&TweetModel{}).Count(&count).Error
	return count, err
}

// GetUserCount returns the total number of users in the database
func (s *DatabaseService) GetUserCount() (int64, error) {
	var count int64
	err := s.db.Model(&UserModel{}).Count(&count).Error
	return count, err
}

// GetFUDUserCount returns the total number of FUD users in the database
func (s *DatabaseService) GetFUDUserCount() (int64, error) {
	var count int64
	err := s.db.Model(&FUDUserModel{}).Count(&count).Error
	return count, err
}

// User relation related methods

// SaveUserRelations saves user relations (followers/following) to database
func (s *DatabaseService) SaveUserRelations(userID string, relatedUsers []string, relationType string) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete existing relations of this type for this user
	if err := tx.Where("user_id = ? AND relation_type = ?", userID, relationType).Delete(&UserRelationModel{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Insert new relations
	for _, relatedUserID := range relatedUsers {
		relation := UserRelationModel{
			UserID:        userID,
			RelatedUserID: relatedUserID,
			RelationType:  relationType,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := tx.Create(&relation).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// GetUserRelations retrieves user relations by type
func (s *DatabaseService) GetUserRelations(userID, relationType string) ([]UserRelationModel, error) {
	var relations []UserRelationModel
	err := s.db.Where("user_id = ? AND relation_type = ?", userID, relationType).Find(&relations).Error
	return relations, err
}

// GetUserFollowers retrieves all followers of a user
func (s *DatabaseService) GetUserFollowers(userID string) ([]UserRelationModel, error) {
	return s.GetUserRelations(userID, RELATION_TYPE_FOLLOWER)
}

// GetUserFollowings retrieves all users that a user is following
func (s *DatabaseService) GetUserFollowings(userID string) ([]UserRelationModel, error) {
	return s.GetUserRelations(userID, RELATION_TYPE_FOLLOWING)
}

// GetTweetsBySourceType retrieves tweets by source type
func (s *DatabaseService) GetTweetsBySourceType(sourceType string, limit int) ([]TweetModel, error) {
	var tweets []TweetModel
	err := s.db.Where("source_type = ?", sourceType).Order("created_at DESC").Limit(limit).Find(&tweets).Error
	return tweets, err
}

// GetTweetsByTickerMention retrieves tweets that mention a specific ticker
func (s *DatabaseService) GetTweetsByTickerMention(ticker string, limit int) ([]TweetModel, error) {
	var tweets []TweetModel
	err := s.db.Where("ticker_mention = ?", ticker).Order("created_at DESC").Limit(limit).Find(&tweets).Error
	return tweets, err
}

// GetUserTickerMentions retrieves tweets from a user mentioning a specific ticker
func (s *DatabaseService) GetUserTickerMentions(userID, ticker string, limit int) ([]TweetModel, error) {
	var tweets []TweetModel
	err := s.db.Where("user_id = ? AND ticker_mention = ?", userID, ticker).Order("created_at DESC").Limit(limit).Find(&tweets).Error
	return tweets, err
}

// GetUserCommunityActivity retrieves all user activity in community grouped by main posts
func (s *DatabaseService) GetUserCommunityActivity(userID string) (*UserCommunityActivity, error) {
	// Get all user tweets from community (both main posts and replies)
	var userTweets []TweetModel
	err := s.db.Where("user_id = ? AND source_type = ?", userID, TWEET_SOURCE_COMMUNITY).
		Order("created_at DESC").Find(&userTweets).Error
	if err != nil {
		return nil, err
	}

	// Group tweets by thread (main post)
	activity := &UserCommunityActivity{
		UserID:       userID,
		ThreadGroups: make([]ThreadGroup, 0),
	}

	// Map to track threads we've already processed
	processedThreads := make(map[string]*ThreadGroup)

	for _, tweet := range userTweets {
		var mainPostID string

		// Determine the main post ID for this tweet
		if tweet.InReplyToID == "" {
			// This is a main post
			mainPostID = tweet.ID
		} else {
			// This is a reply - find the root post
			mainPostID = s.findRootPostID(tweet.InReplyToID)
		}

		// Get or create thread group
		threadGroup, exists := processedThreads[mainPostID]
		if !exists {
			// Get main post details
			mainPost, err := s.GetTweet(mainPostID)
			if err != nil {
				// If we can't find main post, skip this tweet
				continue
			}

			threadGroup = &ThreadGroup{
				MainPost: ThreadPost{
					ID:        mainPost.ID,
					Text:      mainPost.Text,
					Author:    "", // Will be filled by user lookup
					CreatedAt: mainPost.CreatedAt,
				},
				UserReplies: make([]UserReply, 0),
			}

			// Get main post author
			if mainPost.UserID != "" {
				user, err := s.GetUser(mainPost.UserID)
				if err == nil {
					threadGroup.MainPost.Author = user.Username
				}
			}

			processedThreads[mainPostID] = threadGroup
			activity.ThreadGroups = append(activity.ThreadGroups, *threadGroup)
		}

		// Add user reply to thread group (skip if this is the main post by the same user)
		if tweet.ID != mainPostID {
			userReply := UserReply{
				TweetID:     tweet.ID,
				Text:        tweet.Text,
				CreatedAt:   tweet.CreatedAt,
				InReplyToID: tweet.InReplyToID,
			}

			// Find which tweet this is replying to
			if tweet.InReplyToID != "" {
				repliedToTweet, err := s.GetTweet(tweet.InReplyToID)
				if err == nil && repliedToTweet.UserID != "" {
					repliedToUser, err := s.GetUser(repliedToTweet.UserID)
					if err == nil {
						userReply.RepliedToAuthor = repliedToUser.Username
						userReply.RepliedToText = repliedToTweet.Text
					}
				}
			}

			threadGroup.UserReplies = append(threadGroup.UserReplies, userReply)
		}
	}

	// Sort thread groups by main post date (newest first)
	for i := 0; i < len(activity.ThreadGroups); i++ {
		for j := i + 1; j < len(activity.ThreadGroups); j++ {
			if activity.ThreadGroups[i].MainPost.CreatedAt.Before(activity.ThreadGroups[j].MainPost.CreatedAt) {
				activity.ThreadGroups[i], activity.ThreadGroups[j] = activity.ThreadGroups[j], activity.ThreadGroups[i]
			}
		}
	}

	return activity, nil
}

// findRootPostID recursively finds the root post ID for a reply
func (s *DatabaseService) findRootPostID(tweetID string) string {
	tweet, err := s.GetTweet(tweetID)
	if err != nil || tweet.InReplyToID == "" {
		return tweetID
	}
	return s.findRootPostID(tweet.InReplyToID)
}

// IsUserDetailAnalyzed checks if user has been through detailed analysis in memory
func (s *DatabaseService) IsUserDetailAnalyzed(userID string) bool {
	s.userDataMutex.RLock()
	defer s.userDataMutex.RUnlock()
	return s.userAnalyzed[userID]
}

// MarkUserAsDetailAnalyzed marks user as having been through detailed analysis in memory
func (s *DatabaseService) MarkUserAsDetailAnalyzed(userID string) error {
	s.userDataMutex.Lock()
	defer s.userDataMutex.Unlock()
	s.userAnalyzed[userID] = true
	return nil
}

// GetUserMessagesWithContext retrieves user messages with thread context for Telegram history
func (s *DatabaseService) GetUserMessagesWithContext(userID string, limit int) ([]TweetModel, error) {
	var tweets []TweetModel
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&tweets).Error
	return tweets, err
}

// GetUserMessagesByUsername retrieves user messages by username with thread context
func (s *DatabaseService) GetUserMessagesByUsername(username string, limit int) ([]TweetModel, error) {
	// First find user by username to get ID
	user, err := s.GetUserByUsername(username)
	if err != nil {
		// Try to find user from tweets table
		var tweets []TweetModel
		err := s.db.Raw(`
			SELECT DISTINCT t.* FROM tweets t 
			JOIN users u ON t.user_id = u.id 
			WHERE u.username = ? 
			ORDER BY t.created_at DESC 
			LIMIT ?`, username, limit).Find(&tweets).Error
		return tweets, err
	}

	return s.GetUserMessagesWithContext(user.ID, limit)
}

// GetAllUserMessages retrieves all messages for a user (for full export)
func (s *DatabaseService) GetAllUserMessages(userID string) ([]TweetModel, error) {
	var tweets []TweetModel
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&tweets).Error
	return tweets, err
}

// GetAllUserMessagesByUsername retrieves all messages for a user by username (for full export)
func (s *DatabaseService) GetAllUserMessagesByUsername(username string) ([]TweetModel, error) {
	user, err := s.GetUserByUsername(username)
	if err != nil {
		// Try to find user from tweets table
		var tweets []TweetModel
		err := s.db.Raw(`
			SELECT DISTINCT t.* FROM tweets t 
			JOIN users u ON t.user_id = u.id 
			WHERE u.username = ? 
			ORDER BY t.created_at DESC`, username).Find(&tweets).Error
		return tweets, err
	}

	return s.GetAllUserMessages(user.ID)
}

// SearchUsers searches for users by username substring (case-insensitive)
func (s *DatabaseService) SearchUsers(query string, limit int) ([]UserModel, error) {
	var users []UserModel

	// Search in database first
	err := s.db.Where("username LIKE ? OR name LIKE ?", "%"+query+"%", "%"+query+"%").
		Order("username ASC").Limit(limit).Find(&users).Error
	if err != nil {
		return nil, err
	}

	// Also search in memory cache
	s.userDataMutex.RLock()
	defer s.userDataMutex.RUnlock()

	seenUsers := make(map[string]bool)
	for _, user := range users {
		seenUsers[user.ID] = true
	}

	// Add users from memory that match and weren't found in DB
	for _, user := range s.users {
		if seenUsers[user.ID] {
			continue
		}
		if strings.Contains(strings.ToLower(user.Username), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(user.Name), strings.ToLower(query)) {
			users = append(users, *user)
			if len(users) >= limit {
				break
			}
		}
	}

	return users, nil
}

// GetUserTweetForAnalysis gets a recent tweet from user for second step analysis
func (s *DatabaseService) GetUserTweetForAnalysis(username string) (*TweetModel, error) {
	var tweet TweetModel

	// Try to find user and get their most recent tweet
	user, err := s.GetUserByUsername(username)
	if err != nil {
		// Try to find from tweets table directly
		err := s.db.Raw(`
			SELECT t.* FROM tweets t 
			JOIN users u ON t.user_id = u.id 
			WHERE u.username = ? 
			ORDER BY t.created_at DESC 
			LIMIT 1`, username).First(&tweet).Error
		return &tweet, err
	}

	err = s.db.Where("user_id = ?", user.ID).Order("created_at DESC").First(&tweet).Error
	return &tweet, err
}

// Close closes the database connection
func (s *DatabaseService) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
