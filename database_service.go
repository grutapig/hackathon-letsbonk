package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DatabaseService struct {
	db *gorm.DB
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
		db: db,
	}

	// Run migrations
	if err := service.runMigrations(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return service, nil
}

// runMigrations runs database migrations
func (s *DatabaseService) runMigrations() error {
	return s.db.AutoMigrate(&TweetModel{}, &UserModel{}, &FUDUserModel{}, &UserRelationModel{}, &AnalysisTaskModel{}, &CachedAnalysisModel{}, &UserTickerOpinionModel{})
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

// SaveUser saves or updates a user in the database
func (s *DatabaseService) SaveUser(user UserModel) error {
	user.UpdatedAt = time.Now()
	return s.db.Save(&user).Error
}

// GetUser retrieves a user by ID from the database
func (s *DatabaseService) GetUser(id string) (*UserModel, error) {
	var user UserModel
	err := s.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername retrieves a user by username from the database (case insensitive)
func (s *DatabaseService) GetUserByUsername(username string) (*UserModel, error) {
	var user UserModel
	err := s.db.Where("LOWER(username) = ?", strings.ToLower(username)).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UserExists checks if a user exists in the database
func (s *DatabaseService) UserExists(id string) bool {
	var count int64
	s.db.Model(&UserModel{}).Where("id = ?", id).Count(&count)
	return count > 0
}

// UserExistsByUsername checks if a user exists by username in the database (case insensitive)
func (s *DatabaseService) UserExistsByUsername(username string) bool {
	var count int64
	s.db.Model(&UserModel{}).Where("LOWER(username) = ?", strings.ToLower(username)).Count(&count)
	return count > 0
}

// DeleteUser deletes a user from the database
func (s *DatabaseService) DeleteUser(id string) error {
	return s.db.Delete(&UserModel{}, "id = ?", id).Error
}

// FUD User related methods

// SaveFUDUser saves or updates a FUD user in the database
func (s *DatabaseService) SaveFUDUser(fudUser FUDUserModel) error {
	fudUser.UpdatedAt = time.Now()
	return s.db.Save(&fudUser).Error
}

// GetFUDUser retrieves a FUD user by user ID from the database
func (s *DatabaseService) GetFUDUser(userID string) (*FUDUserModel, error) {
	var fudUser FUDUserModel
	err := s.db.Where("user_id = ?", userID).First(&fudUser).Error
	if err != nil {
		return nil, err
	}
	return &fudUser, nil
}

// IsFUDUser checks if a user is marked as FUD in the database
func (s *DatabaseService) IsFUDUser(userID string) bool {
	var count int64
	s.db.Model(&FUDUserModel{}).Where("user_id = ?", userID).Count(&count)
	return count > 0
}

// GetAllFUDUsers retrieves all FUD users
func (s *DatabaseService) GetAllFUDUsers() ([]FUDUserModel, error) {
	var fudUsers []FUDUserModel
	err := s.db.Order("detected_at DESC").Find(&fudUsers).Error
	return fudUsers, err
}

// IncrementFUDUserMessageCount increments the message count for a FUD user in the database
func (s *DatabaseService) IncrementFUDUserMessageCount(userID string, messageID string) error {
	return s.db.Model(&FUDUserModel{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"message_count":   gorm.Expr("message_count + 1"),
		"last_message_id": messageID,
		"updated_at":      time.Now(),
	}).Error
}

// DeleteFUDUser deletes a FUD user from the database
func (s *DatabaseService) DeleteFUDUser(userID string) error {
	return s.db.Delete(&FUDUserModel{}, "user_id = ?", userID).Error
}

// UpdateUserFUDStatus updates user's FUD status in the users table
func (s *DatabaseService) UpdateUserFUDStatus(userID string, isFUD bool, fudType string) error {
	return s.db.Model(&UserModel{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"is_fud":     isFUD,
		"fud_type":   fudType,
		"updated_at": time.Now(),
	}).Error
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

// IsUserDetailAnalyzed checks if user has been through detailed analysis in the database
func (s *DatabaseService) IsUserDetailAnalyzed(userID string) bool {
	var user UserModel
	err := s.db.Where("id = ?", userID).First(&user).Error
	return err == nil && user.IsDetailAnalyzed
}

// MarkUserAsDetailAnalyzed marks user as having been through detailed analysis in the database
func (s *DatabaseService) MarkUserAsDetailAnalyzed(userID string) error {
	return s.db.Model(&UserModel{}).Where("id = ?", userID).Update("is_detail_analyzed", true).Error
}

// GetUserMessagesWithContext retrieves user messages with thread context for Telegram history
func (s *DatabaseService) GetUserMessagesWithContext(userID string, limit int) ([]TweetModel, error) {
	var tweets []TweetModel
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&tweets).Error
	return tweets, err
}

// GetUserMessagesByUsername retrieves user messages by username with thread context (case insensitive)
func (s *DatabaseService) GetUserMessagesByUsername(username string, limit int) ([]TweetModel, error) {
	// First find user by username to get ID
	user, err := s.GetUserByUsername(username)
	if err != nil {
		// Try to find user from tweets table
		var tweets []TweetModel
		err := s.db.Raw(`
			SELECT DISTINCT t.* FROM tweets t 
			JOIN users u ON t.user_id = u.id 
			WHERE LOWER(u.username) = ? 
			ORDER BY t.created_at DESC 
			LIMIT ?`, strings.ToLower(username), limit).Find(&tweets).Error
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

// GetAllUserMessagesByUsername retrieves all messages for a user by username (for full export, case insensitive)
func (s *DatabaseService) GetAllUserMessagesByUsername(username string) ([]TweetModel, error) {
	user, err := s.GetUserByUsername(username)
	if err != nil {
		// Try to find user from tweets table
		var tweets []TweetModel
		err := s.db.Raw(`
			SELECT DISTINCT t.* FROM tweets t 
			JOIN users u ON t.user_id = u.id 
			WHERE LOWER(u.username) = ? 
			ORDER BY t.created_at DESC`, strings.ToLower(username)).Find(&tweets).Error
		return tweets, err
	}

	return s.GetAllUserMessages(user.ID)
}

// GetTopActiveUsers gets the most active users based on tweet count
func (s *DatabaseService) GetTopActiveUsers(limit int) ([]UserModel, error) {
	var users []UserModel

	// Get users ordered by tweet count (most active first)
	query := `
		SELECT u.*, COUNT(t.id) as tweet_count 
		FROM users u 
		LEFT JOIN tweets t ON u.id = t.user_id 
		GROUP BY u.id 
		HAVING tweet_count > 0
		ORDER BY tweet_count DESC, u.username ASC`

	if limit > 0 {
		query += " LIMIT ?"
		err := s.db.Raw(query, limit).Scan(&users).Error
		if err != nil {
			return nil, err
		}
	} else {
		err := s.db.Raw(query).Scan(&users).Error
		if err != nil {
			return nil, err
		}
	}

	return users, nil
}

// SearchUsers searches for users by username substring (case-insensitive)
func (s *DatabaseService) SearchUsers(query string, limit int) ([]UserModel, error) {
	var users []UserModel
	queryLower := strings.ToLower(query)
	err := s.db.Where("LOWER(username) LIKE ? OR LOWER(name) LIKE ?", "%"+queryLower+"%", "%"+queryLower+"%").
		Order("username ASC").Limit(limit).Find(&users).Error
	return users, err
}

// GetUserTweetForAnalysis gets a recent tweet from user for second step analysis (case insensitive)
func (s *DatabaseService) GetUserTweetForAnalysis(username string) (*TweetModel, error) {
	var tweet TweetModel

	// Try to find user and get their most recent tweet
	user, err := s.GetUserByUsername(username)
	if err != nil {
		// Try to find from tweets table directly
		err := s.db.Raw(`
			SELECT t.* FROM tweets t 
			JOIN users u ON t.user_id = u.id 
			WHERE LOWER(u.username) = ? 
			ORDER BY t.created_at DESC 
			LIMIT 1`, strings.ToLower(username)).First(&tweet).Error
		return &tweet, err
	}

	err = s.db.Where("user_id = ?", user.ID).Order("created_at DESC").First(&tweet).Error
	return &tweet, err
}

// Analysis Task Management Methods

// CreateAnalysisTask creates a new analysis task
func (s *DatabaseService) CreateAnalysisTask(task *AnalysisTaskModel) error {
	return s.db.Create(task).Error
}

// GetAnalysisTask gets analysis task by ID
func (s *DatabaseService) GetAnalysisTask(taskID string) (*AnalysisTaskModel, error) {
	var task AnalysisTaskModel
	err := s.db.Where("id = ?", taskID).First(&task).Error
	return &task, err
}

// UpdateAnalysisTask updates existing analysis task
func (s *DatabaseService) UpdateAnalysisTask(task *AnalysisTaskModel) error {
	return s.db.Save(task).Error
}

// UpdateAnalysisTaskProgress updates task progress and step
func (s *DatabaseService) UpdateAnalysisTaskProgress(taskID string, step string, progressText string) error {
	return s.db.Model(&AnalysisTaskModel{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"current_step":  step,
			"progress_text": progressText,
			"status":        ANALYSIS_STATUS_RUNNING,
			"updated_at":    time.Now(),
		}).Error
}

// SetAnalysisTaskError sets task as failed with error message
func (s *DatabaseService) SetAnalysisTaskError(taskID string, errorMessage string) error {
	now := time.Now()
	return s.db.Model(&AnalysisTaskModel{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":        ANALYSIS_STATUS_FAILED,
			"error_message": errorMessage,
			"completed_at":  &now,
			"updated_at":    now,
		}).Error
}

// CompleteAnalysisTask marks task as completed with results
func (s *DatabaseService) CompleteAnalysisTask(taskID string, resultData string) error {
	now := time.Now()
	return s.db.Model(&AnalysisTaskModel{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":       ANALYSIS_STATUS_COMPLETED,
			"current_step": ANALYSIS_STEP_COMPLETED,
			"result_data":  resultData,
			"completed_at": &now,
			"updated_at":   now,
		}).Error
}

// GetRunningAnalysisTasks gets all running analysis tasks for status monitoring
func (s *DatabaseService) GetRunningAnalysisTasks() ([]AnalysisTaskModel, error) {
	var tasks []AnalysisTaskModel
	err := s.db.Where("status = ?", ANALYSIS_STATUS_RUNNING).Find(&tasks).Error
	return tasks, err
}

// GetAllRunningAnalysisTasks gets all pending and running analysis tasks
func (s *DatabaseService) GetAllRunningAnalysisTasks() ([]AnalysisTaskModel, error) {
	var tasks []AnalysisTaskModel
	err := s.db.Where("status IN ?", []string{ANALYSIS_STATUS_PENDING, ANALYSIS_STATUS_RUNNING}).
		Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

// ClearAllAnalysisFlags clears all FUD flags and analysis status for fresh start
func (s *DatabaseService) ClearAllAnalysisFlags() error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Clear all FUD users
	if err := tx.Exec("DELETE FROM fud_users").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to clear FUD users: %w", err)
	}

	// Reset all user analysis flags
	if err := tx.Model(&UserModel{}).Updates(map[string]interface{}{
		"is_fud":             false,
		"fud_type":           "",
		"is_detail_analyzed": false,
		"updated_at":         time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to reset user flags: %w", err)
	}

	// Clear all analysis tasks
	if err := tx.Exec("DELETE FROM analysis_tasks").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to clear analysis tasks: %w", err)
	}

	return tx.Commit().Error
}

// GetAnalysisStats returns statistics about analysis data
func (s *DatabaseService) GetAnalysisStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count total users
	var totalUsers int64
	s.db.Model(&UserModel{}).Count(&totalUsers)
	stats["total_users"] = totalUsers

	// Count analyzed users
	var analyzedUsers int64
	s.db.Model(&UserModel{}).Where("is_detail_analyzed = ?", true).Count(&analyzedUsers)
	stats["analyzed_users"] = analyzedUsers

	// Count FUD users
	var fudUsers int64
	s.db.Model(&FUDUserModel{}).Count(&fudUsers)
	stats["fud_users"] = fudUsers

	// Count running tasks
	var runningTasks int64
	s.db.Model(&AnalysisTaskModel{}).Where("status = ?", ANALYSIS_STATUS_RUNNING).Count(&runningTasks)
	stats["running_tasks"] = runningTasks

	return stats, nil
}

// Cached Analysis Methods

// SaveCachedAnalysis saves analysis result to cache with 24-hour expiration
func (s *DatabaseService) SaveCachedAnalysis(userID, username string, analysis SecondStepClaudeResponse) error {
	// Convert key evidence to JSON string
	keyEvidenceJSON := ""
	if len(analysis.KeyEvidence) > 0 {
		if jsonData, err := json.Marshal(analysis.KeyEvidence); err == nil {
			keyEvidenceJSON = string(jsonData)
		}
	}

	// First try to find existing cached analysis for this user
	var existing CachedAnalysisModel
	err := s.db.Where("user_id = ?", userID).First(&existing).Error

	if err == nil {
		// Update existing record
		existing.Username = username
		existing.IsFUDUser = analysis.IsFUDUser
		existing.FUDType = analysis.FUDType
		existing.FUDProbability = analysis.FUDProbability
		existing.UserRiskLevel = analysis.UserRiskLevel
		existing.UserSummary = analysis.UserSummary
		existing.KeyEvidence = keyEvidenceJSON
		existing.DecisionReason = analysis.DecisionReason
		existing.AnalyzedAt = time.Now()
		existing.ExpiresAt = time.Now().Add(24 * time.Hour)
		existing.UpdatedAt = time.Now()

		log.Printf("🔄 DB: Updating existing cached analysis for user %s (ID: %d)", username, existing.ID)
		return s.db.Save(&existing).Error
	} else {
		// Create new record
		cached := CachedAnalysisModel{
			UserID:         userID,
			Username:       username,
			IsFUDUser:      analysis.IsFUDUser,
			FUDType:        analysis.FUDType,
			FUDProbability: analysis.FUDProbability,
			UserRiskLevel:  analysis.UserRiskLevel,
			UserSummary:    analysis.UserSummary,
			KeyEvidence:    keyEvidenceJSON,
			DecisionReason: analysis.DecisionReason,
			AnalyzedAt:     time.Now(),
			ExpiresAt:      time.Now().Add(24 * time.Hour),
		}

		log.Printf("✅ DB: Creating new cached analysis for user %s", username)
		return s.db.Create(&cached).Error
	}
}

// GetCachedAnalysis retrieves cached analysis if not expired
func (s *DatabaseService) GetCachedAnalysis(userID string) (*SecondStepClaudeResponse, error) {
	var cached CachedAnalysisModel
	err := s.db.Where("user_id = ? AND expires_at > ?", userID, time.Now()).First(&cached).Error
	if err != nil {
		return nil, err
	}

	// Parse key evidence from JSON
	var keyEvidence []string
	if cached.KeyEvidence != "" {
		json.Unmarshal([]byte(cached.KeyEvidence), &keyEvidence)
	}

	result := &SecondStepClaudeResponse{
		IsFUDUser:      cached.IsFUDUser,
		FUDType:        cached.FUDType,
		FUDProbability: cached.FUDProbability,
		UserRiskLevel:  cached.UserRiskLevel,
		UserSummary:    cached.UserSummary,
		KeyEvidence:    keyEvidence,
		DecisionReason: cached.DecisionReason,
	}

	return result, nil
}

// HasValidCachedAnalysis checks if user has valid cached analysis
func (s *DatabaseService) HasValidCachedAnalysis(userID string) bool {
	var count int64
	s.db.Model(&CachedAnalysisModel{}).Where("user_id = ?", userID).Count(&count)
	return count > 0
}

// CleanExpiredCache removes expired cached analysis entries
func (s *DatabaseService) CleanExpiredCache() error {
	return s.db.Where("expires_at < ?", time.Now()).Delete(&CachedAnalysisModel{}).Error
}

// GetAllFUDUsersFromCache gets all FUD users from both active FUD table and cache with last message info
func (s *DatabaseService) GetAllFUDUsersFromCache() ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// Get from active FUD users table
	var fudUsers []FUDUserModel
	err := s.db.Order("detected_at DESC").Find(&fudUsers).Error
	if err != nil {
		return nil, err
	}

	for _, user := range fudUsers {
		// Get last message for this user
		var lastTweet TweetModel
		lastMessageDate := time.Time{}
		isAlive := false

		err = s.db.Where("user_id = ?", user.UserID).Order("created_at DESC").First(&lastTweet).Error
		if err == nil {
			lastMessageDate = lastTweet.CreatedAt
			// User is alive if last message was within 30 days
			isAlive = time.Since(lastMessageDate) <= 30*24*time.Hour
		}

		results = append(results, map[string]interface{}{
			"user_id":           user.UserID,
			"username":          user.Username,
			"fud_type":          user.FUDType,
			"fud_probability":   user.FUDProbability,
			"detected_at":       user.DetectedAt,
			"message_count":     user.MessageCount,
			"last_message_id":   user.LastMessageID,
			"last_message_date": lastMessageDate,
			"is_alive":          isAlive,
			"status":            map[bool]string{true: "alive", false: "dead"}[isAlive],
			"source":            "active",
		})
	}

	// Get from cached analysis (FUD users only)
	var cachedFUD []CachedAnalysisModel
	err = s.db.Where("is_fud_user = ?", true).
		Order("analyzed_at DESC").Find(&cachedFUD).Error
	if err != nil {
		return nil, err
	}

	// Create map to avoid duplicates
	seenUsers := make(map[string]bool)
	for _, user := range fudUsers {
		seenUsers[user.UserID] = true
	}

	for _, cached := range cachedFUD {
		if !seenUsers[cached.UserID] {
			// Get last message for this cached user
			var lastTweet TweetModel
			lastMessageDate := time.Time{}
			isAlive := false

			err = s.db.Where("user_id = ?", cached.UserID).Order("created_at DESC").First(&lastTweet).Error
			if err == nil {
				lastMessageDate = lastTweet.CreatedAt
				// User is alive if last message was within 30 days
				isAlive = time.Since(lastMessageDate) <= 30*24*time.Hour
			}

			results = append(results, map[string]interface{}{
				"user_id":           cached.UserID,
				"username":          cached.Username,
				"fud_type":          cached.FUDType,
				"fud_probability":   cached.FUDProbability,
				"detected_at":       cached.AnalyzedAt,
				"message_count":     0,
				"last_message_id":   "",
				"last_message_date": lastMessageDate,
				"is_alive":          isAlive,
				"status":            map[bool]string{true: "alive", false: "dead"}[isAlive],
				"source":            "cached",
				"user_summary":      cached.UserSummary,
				"expires_at":        cached.ExpiresAt,
			})
			seenUsers[cached.UserID] = true
		}
	}

	return results, nil
}

// GetActiveFUDUsersSortedByLastMessage gets FUD users from cache sorted by last message date (most recent first)
func (s *DatabaseService) GetActiveFUDUsersSortedByLastMessage() ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	log.Printf("🔍 DB: Starting GetActiveFUDUsersSortedByLastMessage")

	// Get from cached analysis (FUD users only)
	var cachedFUD []CachedAnalysisModel
	log.Printf("🔍 DB: Querying cached_analysis table for FUD users...")
	err := s.db.Where("is_fud_user = ?", true).
		Order("analyzed_at DESC").Find(&cachedFUD).Error
	if err != nil {
		log.Printf("❌ DB: Error querying cached_analysis: %v", err)
		return nil, err
	}

	log.Printf("📊 DB: Found %d cached FUD users", len(cachedFUD))

	// Process cached FUD users and add last message info
	log.Printf("🔍 DB: Processing %d cached FUD users...", len(cachedFUD))
	for i, cached := range cachedFUD {
		log.Printf("🔍 DB: Processing user %d/%d: %s (ID: %s)", i+1, len(cachedFUD), cached.Username, cached.UserID)

		// Get last message for this cached user
		var lastTweet TweetModel
		lastMessageDate := time.Time{}
		isAlive := false

		err = s.db.Where("user_id = ?", cached.UserID).Order("created_at DESC").First(&lastTweet).Error
		if err == nil {
			lastMessageDate = lastTweet.CreatedAt
			// User is alive if last message was within 30 days
			isAlive = time.Since(lastMessageDate) <= 30*24*time.Hour
			log.Printf("✅ DB: Found last message for %s: %s (alive: %t)", cached.Username, lastMessageDate.Format("2006-01-02"), isAlive)
		} else {
			log.Printf("⚠️ DB: No messages found for user %s: %v", cached.Username, err)
		}

		results = append(results, map[string]interface{}{
			"user_id":           cached.UserID,
			"username":          cached.Username,
			"fud_type":          cached.FUDType,
			"fud_probability":   cached.FUDProbability,
			"detected_at":       cached.AnalyzedAt,
			"message_count":     0,
			"last_message_id":   "",
			"last_message_date": lastMessageDate,
			"is_alive":          isAlive,
			"status":            map[bool]string{true: "alive", false: "dead"}[isAlive],
			"source":            "cached",
			"user_summary":      cached.UserSummary,
			"expires_at":        cached.ExpiresAt,
		})
	}

	log.Printf("🔍 DB: Sorting %d results by last message date...", len(results))

	// Sort by last message date (most recent first)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			date1 := results[i]["last_message_date"].(time.Time)
			date2 := results[j]["last_message_date"].(time.Time)
			if date1.Before(date2) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	log.Printf("✅ DB: Completed GetActiveFUDUsersSortedByLastMessage with %d results", len(results))

	// Log first few results for debugging
	for i, result := range results {
		if i >= 3 { // Only log first 3 results
			break
		}
		username := result["username"].(string)
		lastMsg := result["last_message_date"].(time.Time)
		log.Printf("📊 DB: Result %d: %s - last msg: %s", i+1, username, lastMsg.Format("2006-01-02 15:04"))
	}

	return results, nil
}

// User Ticker Opinion Methods

// SaveUserTickerOpinion saves user's ticker-related message from advanced search
func (s *DatabaseService) SaveUserTickerOpinion(opinion UserTickerOpinionModel) error {
	opinion.FoundAt = time.Now()
	return s.db.Save(&opinion).Error
}

// GetUserTickerOpinions retrieves all ticker-related messages for a user
func (s *DatabaseService) GetUserTickerOpinions(userID, ticker string, limit int) ([]UserTickerOpinionModel, error) {
	var opinions []UserTickerOpinionModel
	query := s.db.Where("user_id = ?", userID)
	if ticker != "" {
		query = query.Where("ticker = ?", ticker)
	}
	err := query.Order("tweet_created_at DESC").Limit(limit).Find(&opinions).Error
	return opinions, err
}

// GetUserTickerOpinionsByUsername retrieves ticker opinions by username (case insensitive)
func (s *DatabaseService) GetUserTickerOpinionsByUsername(username, ticker string, limit int) ([]UserTickerOpinionModel, error) {
	var opinions []UserTickerOpinionModel
	query := s.db.Where("LOWER(username) = ?", strings.ToLower(username))
	if ticker != "" {
		query = query.Where("ticker = ?", ticker)
	}
	query = query.Order("tweet_created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&opinions).Error
	return opinions, err
}

// TickerOpinionExists checks if a ticker opinion already exists
func (s *DatabaseService) TickerOpinionExists(tweetID string) bool {
	var count int64
	s.db.Model(&UserTickerOpinionModel{}).Where("tweet_id = ?", tweetID).Count(&count)
	return count > 0
}

// GetUserTickerOpinionCount returns the count of ticker opinions for a user
func (s *DatabaseService) GetUserTickerOpinionCount(userID, ticker string) (int64, error) {
	var count int64
	query := s.db.Model(&UserTickerOpinionModel{}).Where("user_id = ?", userID)
	if ticker != "" {
		query = query.Where("ticker = ?", ticker)
	}
	err := query.Count(&count).Error
	return count, err
}

// Close closes the database connection
func (s *DatabaseService) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
