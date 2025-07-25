package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type LoggingService struct {
	db *gorm.DB
}

func NewLoggingService(dbPath string) (*LoggingService, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to logging database: %w", err)
	}

	service := &LoggingService{
		db: db,
	}

	if err := service.runMigrations(); err != nil {
		return nil, fmt.Errorf("failed to run logging migrations: %w", err)
	}

	return service, nil
}

func (s *LoggingService) runMigrations() error {
	return s.db.AutoMigrate(
		&MessageLogModel{},
		&UserActivityLogModel{},
		&AIRequestLogModel{},
		&DataCollectionLogModel{},
		&RequestProcessingLogModel{},
	)
}

func (s *LoggingService) LogMessage(tweetID, userID, username, text, sourceType string, tweetCreatedAt time.Time) error {
	messageLog := MessageLogModel{
		TweetID:        tweetID,
		UserID:         userID,
		Username:       username,
		Text:           text,
		SourceType:     sourceType,
		TweetCreatedAt: tweetCreatedAt,
		LoggedAt:       time.Now(),
	}
	return s.db.Create(&messageLog).Error
}

func (s *LoggingService) GetMessageCountByHour(date time.Time) (int64, error) {
	var count int64
	startOfHour := time.Date(date.Year(), date.Month(), date.Day(), date.Hour(), 0, 0, 0, date.Location())
	endOfHour := startOfHour.Add(time.Hour)

	err := s.db.Model(&MessageLogModel{}).
		Where("logged_at >= ? AND logged_at < ?", startOfHour, endOfHour).
		Count(&count).Error

	return count, err
}

func (s *LoggingService) GetMessageCountByDay(date time.Time) (int64, error) {
	var count int64
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	err := s.db.Model(&MessageLogModel{}).
		Where("logged_at >= ? AND logged_at < ?", startOfDay, endOfDay).
		Count(&count).Error

	return count, err
}

func (s *LoggingService) GetMessageCountByDateRange(startDate, endDate time.Time) (int64, error) {
	var count int64
	err := s.db.Model(&MessageLogModel{}).
		Where("logged_at >= ? AND logged_at < ?", startDate, endDate).
		Count(&count).Error

	return count, err
}

func (s *LoggingService) GetHourlyMessageStats() ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	now := time.Now()

	for i := 23; i >= 0; i-- {
		hourStart := now.Add(-time.Duration(i) * time.Hour).Truncate(time.Hour)
		count, err := s.GetMessageCountByHour(hourStart)
		if err != nil {
			return nil, err
		}

		results = append(results, map[string]interface{}{
			"hour":  hourStart.Format("2006-01-02 15:04"),
			"count": count,
		})
	}

	return results, nil
}

func (s *LoggingService) GetDailyMessageStats() ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	now := time.Now()

	for i := 29; i >= 0; i-- {
		dayStart := now.AddDate(0, 0, -i).Truncate(24 * time.Hour)
		count, err := s.GetMessageCountByDay(dayStart)
		if err != nil {
			return nil, err
		}

		results = append(results, map[string]interface{}{
			"date":  dayStart.Format("2006-01-02"),
			"count": count,
		})
	}

	return results, nil
}

func (s *LoggingService) LogUserActivity(userID, username, activityType, messageID, sourceType string) error {
	userActivity := UserActivityLogModel{
		UserID:       userID,
		Username:     username,
		ActivityType: activityType,
		MessageID:    messageID,
		SourceType:   sourceType,
		ActivityDate: time.Now().Truncate(24 * time.Hour),
		FirstSeenAt:  time.Now(),
		LastSeenAt:   time.Now(),
	}

	var existing UserActivityLogModel
	err := s.db.Where("user_id = ? AND activity_date = ?", userID, userActivity.ActivityDate).First(&existing).Error
	if err == nil {

		existing.LastSeenAt = time.Now()
		existing.ActivityType = activityType
		return s.db.Save(&existing).Error
	} else {

		return s.db.Create(&userActivity).Error
	}
}

func (s *LoggingService) GetUserActivityByDay(date time.Time) (map[string]interface{}, error) {
	targetDate := date.Truncate(24 * time.Hour)

	var newUsers int64
	var existingUsers int64

	err := s.db.Model(&UserActivityLogModel{}).
		Where("activity_date = ? AND activity_type = ?", targetDate, ACTIVITY_TYPE_NEW_USER).
		Count(&newUsers).Error
	if err != nil {
		return nil, err
	}

	err = s.db.Model(&UserActivityLogModel{}).
		Where("activity_date = ? AND activity_type = ?", targetDate, ACTIVITY_TYPE_EXISTING_USER).
		Count(&existingUsers).Error
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"date":           targetDate.Format("2006-01-02"),
		"new_users":      newUsers,
		"existing_users": existingUsers,
		"total_users":    newUsers + existingUsers,
	}, nil
}

func (s *LoggingService) GetDailyUserActivityStats() ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	now := time.Now()

	for i := 29; i >= 0; i-- {
		dayStart := now.AddDate(0, 0, -i).Truncate(24 * time.Hour)
		stats, err := s.GetUserActivityByDay(dayStart)
		if err != nil {
			return nil, err
		}
		results = append(results, stats)
	}

	return results, nil
}

func (s *LoggingService) LogAIRequest(requestUUID, userID, username, tweetID, requestType string, stepNumber int, requestData, responseData interface{}, tokensUsed, processingTime int, isSuccess bool, errorMessage string) error {
	requestJSON, _ := json.Marshal(requestData)
	responseJSON, _ := json.Marshal(responseData)

	aiLog := AIRequestLogModel{
		RequestUUID:    requestUUID,
		StepNumber:     stepNumber,
		RequestType:    requestType,
		UserID:         userID,
		Username:       username,
		TweetID:        tweetID,
		RequestData:    string(requestJSON),
		ResponseData:   string(responseJSON),
		TokensUsed:     tokensUsed,
		ProcessingTime: processingTime,
		IsSuccess:      isSuccess,
		ErrorMessage:   errorMessage,
		RequestedAt:    time.Now(),
	}

	return s.db.Create(&aiLog).Error
}

func (s *LoggingService) GetAIRequestsByUUID(requestUUID string) ([]AIRequestLogModel, error) {
	var requests []AIRequestLogModel
	err := s.db.Where("request_uuid = ?", requestUUID).Order("step_number ASC").Find(&requests).Error
	return requests, err
}

func (s *LoggingService) GetAIRequestStats(days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	var totalRequests int64
	var successfulRequests int64
	var failedRequests int64
	var totalTokens int64
	var avgProcessingTime float64

	err := s.db.Model(&AIRequestLogModel{}).
		Where("created_at >= ?", startDate).
		Count(&totalRequests).Error
	if err != nil {
		return nil, err
	}

	err = s.db.Model(&AIRequestLogModel{}).
		Where("created_at >= ? AND is_success = ?", startDate, true).
		Count(&successfulRequests).Error
	if err != nil {
		return nil, err
	}

	err = s.db.Model(&AIRequestLogModel{}).
		Where("created_at >= ? AND is_success = ?", startDate, false).
		Count(&failedRequests).Error
	if err != nil {
		return nil, err
	}

	err = s.db.Model(&AIRequestLogModel{}).
		Where("created_at >= ?", startDate).
		Select("SUM(tokens_used)").
		Scan(&totalTokens).Error
	if err != nil {
		return nil, err
	}

	err = s.db.Model(&AIRequestLogModel{}).
		Where("created_at >= ?", startDate).
		Select("AVG(processing_time)").
		Scan(&avgProcessingTime).Error
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_requests":      totalRequests,
		"successful_requests": successfulRequests,
		"failed_requests":     failedRequests,
		"success_rate":        float64(successfulRequests) / float64(totalRequests) * 100,
		"total_tokens":        totalTokens,
		"avg_processing_time": avgProcessingTime,
	}, nil
}

func (s *LoggingService) LogDataCollection(requestUUID, userID, username, dataType string, dataCount, dataSize, collectionTime int, isSuccess bool, errorMessage, additionalInfo string) error {
	dataLog := DataCollectionLogModel{
		RequestUUID:    requestUUID,
		UserID:         userID,
		Username:       username,
		DataType:       dataType,
		DataCount:      dataCount,
		DataSize:       dataSize,
		CollectionTime: collectionTime,
		IsSuccess:      isSuccess,
		ErrorMessage:   errorMessage,
		AdditionalInfo: additionalInfo,
		CollectedAt:    time.Now(),
	}

	return s.db.Create(&dataLog).Error
}

func (s *LoggingService) GetDataCollectionsByUUID(requestUUID string) ([]DataCollectionLogModel, error) {
	var logs []DataCollectionLogModel
	err := s.db.Where("request_uuid = ?", requestUUID).Order("collected_at ASC").Find(&logs).Error
	return logs, err
}

func (s *LoggingService) StartRequestProcessing(requestUUID, userID, username, tweetID string, totalSteps int) error {
	processLog := RequestProcessingLogModel{
		RequestUUID:    requestUUID,
		UserID:         userID,
		Username:       username,
		TweetID:        tweetID,
		Status:         PROCESSING_STATUS_STARTED,
		TotalSteps:     totalSteps,
		CompletedSteps: 0,
		StartedAt:      time.Now(),
	}

	return s.db.Create(&processLog).Error
}

func (s *LoggingService) UpdateRequestProcessingStatus(requestUUID, status string, completedSteps int) error {
	updates := map[string]interface{}{
		"status":          status,
		"completed_steps": completedSteps,
	}

	if status == PROCESSING_STATUS_COMPLETED || status == PROCESSING_STATUS_FAILED {
		now := time.Now()
		updates["completed_at"] = &now

		var existing RequestProcessingLogModel
		if err := s.db.Where("request_uuid = ?", requestUUID).First(&existing).Error; err == nil {
			totalTime := int(now.Sub(existing.StartedAt).Milliseconds())
			updates["total_time"] = totalTime
		}
	}

	return s.db.Model(&RequestProcessingLogModel{}).
		Where("request_uuid = ?", requestUUID).
		Updates(updates).Error
}

func (s *LoggingService) GetRequestProcessingByUUID(requestUUID string) (*RequestProcessingLogModel, error) {
	var processLog RequestProcessingLogModel
	err := s.db.Where("request_uuid = ?", requestUUID).First(&processLog).Error
	return &processLog, err
}

func (s *LoggingService) CleanupOldLogs(days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	log.Printf("🧹 Cleaning up logging database records older than %d days (before %s)", days, cutoffDate.Format("2006-01-02"))

	result := s.db.Where("created_at < ?", cutoffDate).Delete(&MessageLogModel{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup message logs: %w", result.Error)
	}
	log.Printf("🧹 Cleaned up %d message log records", result.RowsAffected)

	result = s.db.Where("created_at < ?", cutoffDate).Delete(&UserActivityLogModel{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup user activity logs: %w", result.Error)
	}
	log.Printf("🧹 Cleaned up %d user activity log records", result.RowsAffected)

	result = s.db.Where("created_at < ?", cutoffDate).Delete(&AIRequestLogModel{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup AI request logs: %w", result.Error)
	}
	log.Printf("🧹 Cleaned up %d AI request log records", result.RowsAffected)

	result = s.db.Where("created_at < ?", cutoffDate).Delete(&DataCollectionLogModel{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup data collection logs: %w", result.Error)
	}
	log.Printf("🧹 Cleaned up %d data collection log records", result.RowsAffected)

	result = s.db.Where("created_at < ?", cutoffDate).Delete(&RequestProcessingLogModel{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup request processing logs: %w", result.Error)
	}
	log.Printf("🧹 Cleaned up %d request processing log records", result.RowsAffected)

	return nil
}

func (s *LoggingService) VacuumDatabase() error {
	log.Printf("🧹 Running VACUUM on logging database to reclaim space...")
	err := s.db.Exec("VACUUM").Error
	if err != nil {
		return fmt.Errorf("failed to vacuum logging database: %w", err)
	}
	log.Printf("✅ VACUUM completed successfully")
	return nil
}

func (s *LoggingService) GetDatabaseStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var messageCount int64
	s.db.Model(&MessageLogModel{}).Count(&messageCount)
	stats["message_logs"] = messageCount

	var userActivityCount int64
	s.db.Model(&UserActivityLogModel{}).Count(&userActivityCount)
	stats["user_activity_logs"] = userActivityCount

	var aiRequestCount int64
	s.db.Model(&AIRequestLogModel{}).Count(&aiRequestCount)
	stats["ai_request_logs"] = aiRequestCount

	var dataCollectionCount int64
	s.db.Model(&DataCollectionLogModel{}).Count(&dataCollectionCount)
	stats["data_collection_logs"] = dataCollectionCount

	var requestProcessingCount int64
	s.db.Model(&RequestProcessingLogModel{}).Count(&requestProcessingCount)
	stats["request_processing_logs"] = requestProcessingCount

	var oldestMessage MessageLogModel
	s.db.Order("created_at ASC").First(&oldestMessage)
	if oldestMessage.ID != 0 {
		stats["oldest_record"] = oldestMessage.CreatedAt.Format("2006-01-02 15:04:05")
	}

	var newestMessage MessageLogModel
	s.db.Order("created_at DESC").First(&newestMessage)
	if newestMessage.ID != 0 {
		stats["newest_record"] = newestMessage.CreatedAt.Format("2006-01-02 15:04:05")
	}

	return stats, nil
}

func (s *LoggingService) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
