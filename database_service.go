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

func NewDatabaseService(dbPath string) (*DatabaseService, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	service := &DatabaseService{
		db: db,
	}

	if err := service.runMigrations(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return service, nil
}

func (s *DatabaseService) runMigrations() error {
	return s.db.AutoMigrate(&TweetModel{}, &UserModel{}, &FUDUserModel{}, &UserRelationModel{}, &AnalysisTaskModel{}, &CachedAnalysisModel{}, &UserTickerOpinionModel{})
}

func (s *DatabaseService) SaveTweet(tweet TweetModel) error {
	tweet.UpdatedAt = time.Now()
	return s.db.Save(&tweet).Error
}

func (s *DatabaseService) GetTweet(id string) (*TweetModel, error) {
	var tweet TweetModel
	err := s.db.Where("id = ?", id).First(&tweet).Error
	if err != nil {
		return nil, err
	}
	return &tweet, nil
}

func (s *DatabaseService) GetTweetByAutoID(autoID uint) (*TweetModel, error) {
	var tweet TweetModel
	err := s.db.Where("auto_id = ?", autoID).First(&tweet).Error
	if err != nil {
		return nil, err
	}
	return &tweet, nil
}

func (s *DatabaseService) TweetExists(id string) bool {
	var count int64
	s.db.Model(&TweetModel{}).Where("id = ?", id).Count(&count)
	return count > 0
}

func (s *DatabaseService) GetTweetReplyCount(id string) (int, error) {
	var tweet TweetModel
	err := s.db.Select("reply_count").Where("id = ?", id).First(&tweet).Error
	if err != nil {
		return 0, err
	}
	return tweet.ReplyCount, nil
}

func (s *DatabaseService) UpdateTweetReplyCount(id string, replyCount int) error {
	return s.db.Model(&TweetModel{}).Where("id = ?", id).Update("reply_count", replyCount).Error
}

func (s *DatabaseService) GetTweetsByUser(userID string) ([]TweetModel, error) {
	var tweets []TweetModel
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&tweets).Error
	return tweets, err
}

func (s *DatabaseService) GetRepliesForTweet(tweetID string) ([]TweetModel, error) {
	var replies []TweetModel
	err := s.db.Where("in_reply_to_id = ?", tweetID).Order("created_at ASC").Find(&replies).Error
	return replies, err
}

func (s *DatabaseService) DeleteTweet(id string) error {
	return s.db.Delete(&TweetModel{}, "id = ?", id).Error
}

func (s *DatabaseService) SaveUser(user UserModel) error {
	user.UpdatedAt = time.Now()
	return s.db.Save(&user).Error
}

func (s *DatabaseService) GetUser(id string) (*UserModel, error) {
	var user UserModel
	err := s.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *DatabaseService) GetUserByUsername(username string) (*UserModel, error) {
	var user UserModel
	err := s.db.Where("LOWER(username) = ?", strings.ToLower(username)).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *DatabaseService) UserExists(id string) bool {
	var count int64
	s.db.Model(&UserModel{}).Where("id = ?", id).Count(&count)
	return count > 0
}

func (s *DatabaseService) UserExistsByUsername(username string) bool {
	var count int64
	s.db.Model(&UserModel{}).Where("LOWER(username) = ?", strings.ToLower(username)).Count(&count)
	return count > 0
}

func (s *DatabaseService) DeleteUser(id string) error {
	return s.db.Delete(&UserModel{}, "id = ?", id).Error
}

func (s *DatabaseService) SaveFUDUser(fudUser FUDUserModel) error {
	fudUser.UpdatedAt = time.Now()
	return s.db.Save(&fudUser).Error
}

func (s *DatabaseService) GetFUDUser(userID string) (*FUDUserModel, error) {
	var fudUser FUDUserModel
	err := s.db.Where("user_id = ?", userID).First(&fudUser).Error
	if err != nil {
		return nil, err
	}
	return &fudUser, nil
}

func (s *DatabaseService) IsFUDUser(userID string) bool {
	var count int64
	s.db.Model(&FUDUserModel{}).Where("user_id = ?", userID).Count(&count)
	return count > 0
}

func (s *DatabaseService) GetAllFUDUsers() ([]FUDUserModel, error) {
	var fudUsers []FUDUserModel
	err := s.db.Order("detected_at DESC").Find(&fudUsers).Error
	return fudUsers, err
}

func (s *DatabaseService) IncrementFUDUserMessageCount(userID string, messageID string) error {
	return s.db.Model(&FUDUserModel{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"message_count":   gorm.Expr("message_count + 1"),
		"last_message_id": messageID,
		"updated_at":      time.Now(),
	}).Error
}

func (s *DatabaseService) DeleteFUDUser(userID string) error {
	return s.db.Delete(&FUDUserModel{}, "user_id = ?", userID).Error
}

func (s *DatabaseService) UpdateUserFUDStatus(userID string, isFUD bool, fudType string) error {
	return s.db.Model(&UserModel{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"is_fud":     isFUD,
		"fud_type":   fudType,
		"updated_at": time.Now(),
	}).Error
}

func (s *DatabaseService) SearchTweets(query string, limit int) ([]TweetModel, error) {
	var tweets []TweetModel
	err := s.db.Where("text LIKE ?", "%"+query+"%").Limit(limit).Order("created_at DESC").Find(&tweets).Error
	return tweets, err
}

func (s *DatabaseService) GetRecentTweets(limit int) ([]TweetModel, error) {
	var tweets []TweetModel
	err := s.db.Order("created_at DESC").Limit(limit).Find(&tweets).Error
	return tweets, err
}

func (s *DatabaseService) GetTweetCount() (int64, error) {
	var count int64
	err := s.db.Model(&TweetModel{}).Count(&count).Error
	return count, err
}

func (s *DatabaseService) GetUserCount() (int64, error) {
	var count int64
	err := s.db.Model(&UserModel{}).Count(&count).Error
	return count, err
}

func (s *DatabaseService) GetFUDUserCount() (int64, error) {
	var count int64
	err := s.db.Model(&FUDUserModel{}).Count(&count).Error
	return count, err
}

func (s *DatabaseService) SaveUserRelations(userID string, relatedUsers []string, relationType string) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Where("user_id = ? AND relation_type = ?", userID, relationType).Delete(&UserRelationModel{}).Error; err != nil {
		tx.Rollback()
		return err
	}

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

func (s *DatabaseService) GetUserRelations(userID, relationType string) ([]UserRelationModel, error) {
	var relations []UserRelationModel
	err := s.db.Where("user_id = ? AND relation_type = ?", userID, relationType).Find(&relations).Error
	return relations, err
}

func (s *DatabaseService) GetUserFollowers(userID string) ([]UserRelationModel, error) {
	return s.GetUserRelations(userID, RELATION_TYPE_FOLLOWER)
}

func (s *DatabaseService) GetUserFollowings(userID string) ([]UserRelationModel, error) {
	return s.GetUserRelations(userID, RELATION_TYPE_FOLLOWING)
}

func (s *DatabaseService) GetTweetsBySourceType(sourceType string, limit int) ([]TweetModel, error) {
	var tweets []TweetModel
	err := s.db.Where("source_type = ?", sourceType).Order("created_at DESC").Limit(limit).Find(&tweets).Error
	return tweets, err
}

func (s *DatabaseService) GetTweetsByTickerMention(ticker string, limit int) ([]TweetModel, error) {
	var tweets []TweetModel
	err := s.db.Where("ticker_mention = ?", ticker).Order("created_at DESC").Limit(limit).Find(&tweets).Error
	return tweets, err
}

func (s *DatabaseService) GetUserTickerMentions(userID, ticker string, limit int) ([]TweetModel, error) {
	var tweets []TweetModel
	err := s.db.Where("user_id = ? AND ticker_mention = ?", userID, ticker).Order("created_at DESC").Limit(limit).Find(&tweets).Error
	return tweets, err
}

func (s *DatabaseService) GetUserCommunityActivity(userID string) (*UserCommunityActivity, error) {

	var userTweets []TweetModel
	err := s.db.Where("user_id = ? AND source_type = ?", userID, TWEET_SOURCE_COMMUNITY).
		Order("created_at DESC").Find(&userTweets).Error
	if err != nil {
		return nil, err
	}

	activity := &UserCommunityActivity{
		UserID:       userID,
		ThreadGroups: make([]ThreadGroup, 0),
	}

	processedThreads := make(map[string]*ThreadGroup)

	for _, tweet := range userTweets {
		var mainPostID string

		if tweet.InReplyToID == "" {

			mainPostID = tweet.ID
		} else {

			mainPostID = s.findRootPostID(tweet.InReplyToID)
		}

		threadGroup, exists := processedThreads[mainPostID]
		if !exists {

			mainPost, err := s.GetTweet(mainPostID)
			if err != nil {

				continue
			}

			threadGroup = &ThreadGroup{
				MainPost: ThreadPost{
					ID:        mainPost.ID,
					Text:      mainPost.Text,
					Author:    "",
					CreatedAt: mainPost.CreatedAt,
				},
				UserReplies: make([]UserReply, 0),
			}

			if mainPost.UserID != "" {
				user, err := s.GetUser(mainPost.UserID)
				if err == nil {
					threadGroup.MainPost.Author = user.Username
				}
			}

			processedThreads[mainPostID] = threadGroup
			activity.ThreadGroups = append(activity.ThreadGroups, *threadGroup)
		}

		if tweet.ID != mainPostID {
			userReply := UserReply{
				TweetID:     tweet.ID,
				Text:        tweet.Text,
				CreatedAt:   tweet.CreatedAt,
				InReplyToID: tweet.InReplyToID,
			}

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

	for i := 0; i < len(activity.ThreadGroups); i++ {
		for j := i + 1; j < len(activity.ThreadGroups); j++ {
			if activity.ThreadGroups[i].MainPost.CreatedAt.Before(activity.ThreadGroups[j].MainPost.CreatedAt) {
				activity.ThreadGroups[i], activity.ThreadGroups[j] = activity.ThreadGroups[j], activity.ThreadGroups[i]
			}
		}
	}

	return activity, nil
}

func (s *DatabaseService) findRootPostID(tweetID string) string {
	tweet, err := s.GetTweet(tweetID)
	if err != nil || tweet.InReplyToID == "" {
		return tweetID
	}
	return s.findRootPostID(tweet.InReplyToID)
}

func (s *DatabaseService) IsUserDetailAnalyzed(userID string) bool {
	var user UserModel
	err := s.db.Where("id = ?", userID).First(&user).Error
	return err == nil && user.IsDetailAnalyzed
}

func (s *DatabaseService) MarkUserAsDetailAnalyzed(userID string) error {
	return s.db.Model(&UserModel{}).Where("id = ?", userID).Update("is_detail_analyzed", true).Error
}

func (s *DatabaseService) GetUserMessagesWithContext(userID string, limit int) ([]TweetModel, error) {
	var tweets []TweetModel
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&tweets).Error
	return tweets, err
}

func (s *DatabaseService) GetUserMessagesByUsername(username string, limit int) ([]TweetModel, error) {

	user, err := s.GetUserByUsername(username)
	if err != nil {

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

func (s *DatabaseService) GetAllUserMessages(userID string) ([]TweetModel, error) {
	var tweets []TweetModel
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&tweets).Error
	return tweets, err
}

func (s *DatabaseService) GetAllUserMessagesByUsername(username string) ([]TweetModel, error) {
	user, err := s.GetUserByUsername(username)
	if err != nil {

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

func (s *DatabaseService) GetTopActiveUsers(limit int) ([]UserModel, error) {
	var users []UserModel

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

func (s *DatabaseService) SearchUsers(query string, limit int) ([]UserModel, error) {
	var users []UserModel
	queryLower := strings.ToLower(query)
	err := s.db.Where("LOWER(username) LIKE ? OR LOWER(name) LIKE ?", "%"+queryLower+"%", "%"+queryLower+"%").
		Order("username ASC").Limit(limit).Find(&users).Error
	return users, err
}

func (s *DatabaseService) GetUserTweetForAnalysis(username string) (*TweetModel, error) {
	var tweet TweetModel

	user, err := s.GetUserByUsername(username)
	if err != nil {

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

func (s *DatabaseService) CreateAnalysisTask(task *AnalysisTaskModel) error {
	return s.db.Create(task).Error
}

func (s *DatabaseService) GetAnalysisTask(taskID string) (*AnalysisTaskModel, error) {
	var task AnalysisTaskModel
	err := s.db.Where("id = ?", taskID).First(&task).Error
	return &task, err
}

func (s *DatabaseService) UpdateAnalysisTask(task *AnalysisTaskModel) error {
	return s.db.Save(task).Error
}

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

func (s *DatabaseService) GetRunningAnalysisTasks() ([]AnalysisTaskModel, error) {
	var tasks []AnalysisTaskModel
	err := s.db.Where("status = ?", ANALYSIS_STATUS_RUNNING).Find(&tasks).Error
	return tasks, err
}

func (s *DatabaseService) GetAllRunningAnalysisTasks() ([]AnalysisTaskModel, error) {
	var tasks []AnalysisTaskModel
	err := s.db.Where("status IN ?", []string{ANALYSIS_STATUS_PENDING, ANALYSIS_STATUS_RUNNING}).
		Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

func (s *DatabaseService) ClearAllAnalysisFlags() error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Exec("DELETE FROM fud_users").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to clear FUD users: %w", err)
	}

	if err := tx.Model(&UserModel{}).Updates(map[string]interface{}{
		"is_fud":             false,
		"fud_type":           "",
		"is_detail_analyzed": false,
		"updated_at":         time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to reset user flags: %w", err)
	}

	if err := tx.Exec("DELETE FROM analysis_tasks").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to clear analysis tasks: %w", err)
	}

	return tx.Commit().Error
}

func (s *DatabaseService) GetAnalysisStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalUsers int64
	s.db.Model(&UserModel{}).Count(&totalUsers)
	stats["total_users"] = totalUsers

	var analyzedUsers int64
	s.db.Model(&UserModel{}).Where("is_detail_analyzed = ?", true).Count(&analyzedUsers)
	stats["analyzed_users"] = analyzedUsers

	var fudUsers int64
	s.db.Model(&FUDUserModel{}).Count(&fudUsers)
	stats["fud_users"] = fudUsers

	var runningTasks int64
	s.db.Model(&AnalysisTaskModel{}).Where("status = ?", ANALYSIS_STATUS_RUNNING).Count(&runningTasks)
	stats["running_tasks"] = runningTasks

	return stats, nil
}

func (s *DatabaseService) SaveCachedAnalysis(userID, username string, analysis SecondStepClaudeResponse) error {

	keyEvidenceJSON := ""
	if len(analysis.KeyEvidence) > 0 {
		if jsonData, err := json.Marshal(analysis.KeyEvidence); err == nil {
			keyEvidenceJSON = string(jsonData)
		}
	}

	var existing CachedAnalysisModel
	err := s.db.Where("user_id = ?", userID).First(&existing).Error

	if err == nil {

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

func (s *DatabaseService) GetCachedAnalysis(userID string) (*SecondStepClaudeResponse, error) {
	var cached CachedAnalysisModel
	err := s.db.Where("user_id = ?", userID).First(&cached).Error
	if err != nil {
		return nil, err
	}

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

func (s *DatabaseService) HasValidCachedAnalysis(userID string) bool {
	var count int64
	s.db.Model(&CachedAnalysisModel{}).Where("user_id = ?", userID).Count(&count)
	return count > 0
}

func (s *DatabaseService) GetAllFUDUsersFromCache() ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	var fudUsers []FUDUserModel
	err := s.db.Order("detected_at DESC").Find(&fudUsers).Error
	if err != nil {
		return nil, err
	}

	for _, user := range fudUsers {

		var lastTweet TweetModel
		lastMessageDate := time.Time{}
		isAlive := false

		err = s.db.Where("user_id = ?", user.UserID).Order("created_at DESC").First(&lastTweet).Error
		if err == nil {
			lastMessageDate = lastTweet.CreatedAt

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

	var cachedFUD []CachedAnalysisModel
	err = s.db.Where("is_fud_user = ?", true).
		Order("analyzed_at DESC").Find(&cachedFUD).Error
	if err != nil {
		return nil, err
	}

	seenUsers := make(map[string]bool)
	for _, user := range fudUsers {
		seenUsers[user.UserID] = true
	}

	for _, cached := range cachedFUD {
		if !seenUsers[cached.UserID] {

			var lastTweet TweetModel
			lastMessageDate := time.Time{}
			isAlive := false

			err = s.db.Where("user_id = ?", cached.UserID).Order("created_at DESC").First(&lastTweet).Error
			if err == nil {
				lastMessageDate = lastTweet.CreatedAt

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

func (s *DatabaseService) GetActiveFUDUsersSortedByLastMessage() ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	log.Printf("🔍 DB: Starting GetActiveFUDUsersSortedByLastMessage")

	var cachedFUD []CachedAnalysisModel
	log.Printf("🔍 DB: Querying cached_analysis table for FUD users...")
	err := s.db.Where("is_fud_user = ?", true).
		Order("analyzed_at DESC").Find(&cachedFUD).Error
	if err != nil {
		log.Printf("❌ DB: Error querying cached_analysis: %v", err)
		return nil, err
	}

	log.Printf("📊 DB: Found %d cached FUD users", len(cachedFUD))

	log.Printf("🔍 DB: Processing %d cached FUD users...", len(cachedFUD))
	for i, cached := range cachedFUD {
		log.Printf("🔍 DB: Processing user %d/%d: %s (ID: %s)", i+1, len(cachedFUD), cached.Username, cached.UserID)

		var lastTweet TweetModel
		lastMessageDate := time.Time{}
		isAlive := false

		err = s.db.Where("user_id = ?", cached.UserID).Order("created_at DESC").First(&lastTweet).Error
		if err == nil {
			lastMessageDate = lastTweet.CreatedAt

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

	for i, result := range results {
		if i >= 3 {
			break
		}
		username := result["username"].(string)
		lastMsg := result["last_message_date"].(time.Time)
		log.Printf("📊 DB: Result %d: %s - last msg: %s", i+1, username, lastMsg.Format("2006-01-02 15:04"))
	}

	return results, nil
}

func (s *DatabaseService) SaveUserTickerOpinion(opinion UserTickerOpinionModel) error {
	opinion.FoundAt = time.Now()
	return s.db.Save(&opinion).Error
}

func (s *DatabaseService) GetUserTickerOpinions(userID, ticker string, limit int) ([]UserTickerOpinionModel, error) {
	var opinions []UserTickerOpinionModel
	query := s.db.Where("user_id = ?", userID)
	if ticker != "" {
		query = query.Where("ticker = ?", ticker)
	}
	err := query.Order("tweet_created_at DESC").Limit(limit).Find(&opinions).Error
	return opinions, err
}

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

func (s *DatabaseService) TickerOpinionExists(tweetID string) bool {
	var count int64
	s.db.Model(&UserTickerOpinionModel{}).Where("tweet_id = ?", tweetID).Count(&count)
	return count > 0
}

func (s *DatabaseService) GetUserTickerOpinionCount(userID, ticker string) (int64, error) {
	var count int64
	query := s.db.Model(&UserTickerOpinionModel{}).Where("user_id = ?", userID)
	if ticker != "" {
		query = query.Where("ticker = ?", ticker)
	}
	err := query.Count(&count).Error
	return count, err
}

func (s *DatabaseService) GetUserStatus(userID string) string {
	var user UserModel
	err := s.db.Select("status").Where("id = ?", userID).First(&user).Error
	if err != nil {
		return USER_STATUS_UNKNOWN
	}
	return user.Status
}

func (s *DatabaseService) SetUserAnalyzing(userID, username string) error {
	now := time.Now()
	return s.db.Model(&UserModel{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"status":           USER_STATUS_ANALYZING,
		"last_analyzed_at": &now,
		"analysis_count":   gorm.Expr("analysis_count + 1"),
		"updated_at":       now,
	}).Error
}

func (s *DatabaseService) UpdateUserAfterAnalysis(userID, username string, aiDecision SecondStepClaudeResponse, messageID string) error {
	now := time.Now()

	status := USER_STATUS_CLEAN
	if aiDecision.IsFUDUser || aiDecision.IsFUDAttack {
		status = USER_STATUS_FUD_CONFIRMED
	}

	updates := map[string]interface{}{
		"username":         username,
		"status":           status,
		"last_analyzed_at": &now,
		"last_message_id":  messageID,
		"analysis_count":   gorm.Expr("analysis_count + 1"),
		"updated_at":       now,
	}

	if status == USER_STATUS_FUD_CONFIRMED {
		updates["is_fud"] = true
		updates["fud_type"] = aiDecision.FUDType
		updates["fud_probability"] = aiDecision.FUDProbability
		updates["fud_message_count"] = gorm.Expr("fud_message_count + 1")
	} else {
		updates["is_fud"] = false
		updates["fud_type"] = ""
		updates["fud_probability"] = 0
	}

	return s.db.Model(&UserModel{}).Where("id = ?", userID).Updates(updates).Error
}

func (s *DatabaseService) MarkUserAsFUD(userID, username, messageID string, fudType string, probability float64) error {
	now := time.Now()
	return s.db.Model(&UserModel{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"username":          username,
		"status":            USER_STATUS_FUD_CONFIRMED,
		"is_fud":            true,
		"fud_type":          fudType,
		"fud_probability":   probability,
		"last_message_id":   messageID,
		"last_analyzed_at":  &now,
		"fud_message_count": gorm.Expr("fud_message_count + 1"),
		"updated_at":        now,
	}).Error
}

func (s *DatabaseService) IsFUDUserByStatus(userID string) bool {
	return s.GetUserStatus(userID) == USER_STATUS_FUD_CONFIRMED
}

func (s *DatabaseService) IsUserBeingAnalyzed(userID string) bool {
	return s.GetUserStatus(userID) == USER_STATUS_ANALYZING
}

func (s *DatabaseService) GetFUDFriendsAnalysis(usernames []string) (int, int, []string) {
	if len(usernames) == 0 {
		return 0, 0, []string{}
	}

	totalFriends := len(usernames)
	fudFriends := 0
	fudFriendsList := make([]string, 0)

	lowerUsernames := make([]string, len(usernames))
	for i, username := range usernames {
		lowerUsernames[i] = strings.ToLower(username)
	}

	var fudUsers []UserModel
	err := s.db.Where("LOWER(username) IN ? AND status = ?", lowerUsernames, USER_STATUS_FUD_CONFIRMED).Find(&fudUsers).Error
	if err != nil {
		log.Printf("Error getting FUD friends analysis: %v", err)
		return totalFriends, 0, []string{}
	}

	for _, user := range fudUsers {
		fudFriends++
		fudFriendsList = append(fudFriendsList, fmt.Sprintf("%s (%s, %.1f%%)",
			user.Username, user.FUDType, user.FUDProbability*100))
	}

	return totalFriends, fudFriends, fudFriendsList
}

func (s *DatabaseService) GetUserStats() (map[string]int, error) {
	stats := map[string]int{
		"total_users":   0,
		"fud_confirmed": 0,
		"clean_users":   0,
		"analyzing":     0,
		"unknown":       0,
	}

	var totalCount int64
	err := s.db.Model(&UserModel{}).Count(&totalCount).Error
	if err != nil {
		return stats, err
	}
	stats["total_users"] = int(totalCount)

	var statusCounts []struct {
		Status string
		Count  int64
	}

	err = s.db.Model(&UserModel{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&statusCounts).Error
	if err != nil {
		return stats, err
	}

	for _, statusCount := range statusCounts {
		switch statusCount.Status {
		case USER_STATUS_FUD_CONFIRMED:
			stats["fud_confirmed"] = int(statusCount.Count)
		case USER_STATUS_CLEAN:
			stats["clean_users"] = int(statusCount.Count)
		case USER_STATUS_ANALYZING:
			stats["analyzing"] = int(statusCount.Count)
		case USER_STATUS_UNKNOWN:
			stats["unknown"] = int(statusCount.Count)
		}
	}

	return stats, nil
}

func (s *DatabaseService) GetUserInfo(userID string) (*UserModel, error) {
	var user UserModel
	err := s.db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *DatabaseService) GetCachedAnalysisByUsername(username string) (*CachedAnalysisModel, error) {
	var cached CachedAnalysisModel
	err := s.db.Where("LOWER(username) = ?", strings.ToLower(username)).First(&cached).Error
	if err != nil {
		return nil, err
	}
	return &cached, nil
}

func (s *DatabaseService) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
