package main

import (
	"time"

	"gorm.io/gorm"
)

// MessageLogModel tracks all new messages coming through monitoring
type MessageLogModel struct {
	gorm.Model
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TweetID        string    `gorm:"column:tweet_id;index" json:"tweet_id"`
	UserID         string    `gorm:"column:user_id;index" json:"user_id"`
	Username       string    `gorm:"column:username;index" json:"username"`
	Text           string    `gorm:"column:text" json:"text"`
	SourceType     string    `gorm:"column:source_type;index" json:"source_type"`
	TweetCreatedAt time.Time `gorm:"column:tweet_created_at;index" json:"tweet_created_at"`
	LoggedAt       time.Time `gorm:"column:logged_at;index" json:"logged_at"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
}

func (MessageLogModel) TableName() string {
	return "message_logs"
}

// UserActivityLogModel tracks user activity (new/existing users)
type UserActivityLogModel struct {
	gorm.Model
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       string    `gorm:"column:user_id;index" json:"user_id"`
	Username     string    `gorm:"column:username;index" json:"username"`
	ActivityType string    `gorm:"column:activity_type;index" json:"activity_type"` // "new_user", "existing_user"
	MessageID    string    `gorm:"column:message_id" json:"message_id"`
	SourceType   string    `gorm:"column:source_type;index" json:"source_type"`
	FirstSeenAt  time.Time `gorm:"column:first_seen_at" json:"first_seen_at"`
	LastSeenAt   time.Time `gorm:"column:last_seen_at" json:"last_seen_at"`
	ActivityDate time.Time `gorm:"column:activity_date;index" json:"activity_date"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
}

func (UserActivityLogModel) TableName() string {
	return "user_activity_logs"
}

// AIRequestLogModel tracks all AI requests and responses
type AIRequestLogModel struct {
	gorm.Model
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	RequestUUID    string    `gorm:"column:request_uuid;uniqueIndex" json:"request_uuid"`
	StepNumber     int       `gorm:"column:step_number;index" json:"step_number"`   // 1 for first step, 2 for second step
	RequestType    string    `gorm:"column:request_type;index" json:"request_type"` // "first_step", "second_step"
	UserID         string    `gorm:"column:user_id;index" json:"user_id"`
	Username       string    `gorm:"column:username;index" json:"username"`
	TweetID        string    `gorm:"column:tweet_id;index" json:"tweet_id"`
	RequestData    string    `gorm:"column:request_data" json:"request_data"`   // JSON of request sent to AI
	ResponseData   string    `gorm:"column:response_data" json:"response_data"` // JSON of response from AI
	TokensUsed     int       `gorm:"column:tokens_used" json:"tokens_used"`
	ProcessingTime int       `gorm:"column:processing_time" json:"processing_time"` // milliseconds
	IsSuccess      bool      `gorm:"column:is_success" json:"is_success"`
	ErrorMessage   string    `gorm:"column:error_message" json:"error_message,omitempty"`
	RequestedAt    time.Time `gorm:"column:requested_at;index" json:"requested_at"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
}

func (AIRequestLogModel) TableName() string {
	return "ai_request_logs"
}

// DataCollectionLogModel tracks data collection for AI requests
type DataCollectionLogModel struct {
	gorm.Model
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	RequestUUID    string    `gorm:"column:request_uuid;index" json:"request_uuid"`
	UserID         string    `gorm:"column:user_id;index" json:"user_id"`
	Username       string    `gorm:"column:username;index" json:"username"`
	DataType       string    `gorm:"column:data_type;index" json:"data_type"` // "user_tweets", "ticker_mentions", "followers", "following", "community_activity"
	DataCount      int       `gorm:"column:data_count" json:"data_count"`
	DataSize       int       `gorm:"column:data_size" json:"data_size"`             // bytes
	CollectionTime int       `gorm:"column:collection_time" json:"collection_time"` // milliseconds
	IsSuccess      bool      `gorm:"column:is_success" json:"is_success"`
	ErrorMessage   string    `gorm:"column:error_message" json:"error_message,omitempty"`
	AdditionalInfo string    `gorm:"column:additional_info" json:"additional_info,omitempty"` // JSON for extra data
	CollectedAt    time.Time `gorm:"column:collected_at;index" json:"collected_at"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
}

func (DataCollectionLogModel) TableName() string {
	return "data_collection_logs"
}

// RequestProcessingLogModel tracks overall request processing lifecycle
type RequestProcessingLogModel struct {
	gorm.Model
	ID             uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	RequestUUID    string     `gorm:"column:request_uuid;uniqueIndex" json:"request_uuid"`
	UserID         string     `gorm:"column:user_id;index" json:"user_id"`
	Username       string     `gorm:"column:username;index" json:"username"`
	TweetID        string     `gorm:"column:tweet_id;index" json:"tweet_id"`
	ProcessingType string     `gorm:"column:processing_type;index" json:"processing_type"` // "detailed", "fast"
	Status         string     `gorm:"column:status;index" json:"status"`                   // "started", "first_step_completed", "second_step_completed", "completed", "failed"
	TotalSteps     int        `gorm:"column:total_steps" json:"total_steps"`
	CompletedSteps int        `gorm:"column:completed_steps" json:"completed_steps"`
	StartedAt      time.Time  `gorm:"column:started_at;index" json:"started_at"`
	CompletedAt    *time.Time `gorm:"column:completed_at;index" json:"completed_at,omitempty"`
	TotalTime      int        `gorm:"column:total_time" json:"total_time"` // milliseconds
	CreatedAt      time.Time  `gorm:"column:created_at" json:"created_at"`
}

func (RequestProcessingLogModel) TableName() string {
	return "request_processing_logs"
}

// Activity type constants for user activity logging
const (
	ACTIVITY_TYPE_NEW_USER      = "new_user"
	ACTIVITY_TYPE_EXISTING_USER = "existing_user"
)

// Request type constants for AI request logging
const (
	REQUEST_TYPE_FIRST_STEP  = "first_step"
	REQUEST_TYPE_SECOND_STEP = "second_step"
)

// Data type constants for data collection logging
const (
	DATA_TYPE_USER_TWEETS        = "user_tweets"
	DATA_TYPE_TICKER_MENTIONS    = "ticker_mentions"
	DATA_TYPE_FOLLOWERS          = "followers"
	DATA_TYPE_FOLLOWING          = "following"
	DATA_TYPE_COMMUNITY_ACTIVITY = "community_activity"
	DATA_TYPE_REPLIED_TO_TWEETS  = "replied_to_tweets"
)

// Processing status constants
const (
	PROCESSING_STATUS_STARTED               = "started"
	PROCESSING_STATUS_FIRST_STEP_COMPLETED  = "first_step_completed"
	PROCESSING_STATUS_SECOND_STEP_COMPLETED = "second_step_completed"
	PROCESSING_STATUS_COMPLETED             = "completed"
	PROCESSING_STATUS_FAILED                = "failed"
)
