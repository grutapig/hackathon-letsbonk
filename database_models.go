package main

import (
	"gorm.io/gorm"
	"time"
)

type TweetModel struct {
	gorm.Model
	ID            string    `gorm:"primaryKey;column:id" json:"id"`
	Text          string    `gorm:"column:text" json:"text"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	ReplyCount    int       `gorm:"column:reply_count" json:"reply_count"`
	UserID        string    `gorm:"column:user_id;index" json:"user_id"`
	Username      string    `gorm:"column:username;index" json:"username"`
	InReplyToID   string    `gorm:"column:in_reply_to_id;index" json:"in_reply_to_id,omitempty"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`
	SourceType    string    `gorm:"column:source_type;index" json:"source_type"`
	TickerMention string    `gorm:"column:ticker_mention;index" json:"ticker_mention"`
	SearchQuery   string    `gorm:"column:search_query" json:"search_query,omitempty"`
}

func (TweetModel) TableName() string {
	return "tweets"
}

type UserModel struct {
	gorm.Model
	ID               string     `gorm:"primaryKey;column:id" json:"id"`
	Username         string     `gorm:"column:username;uniqueIndex" json:"username"`
	Name             string     `gorm:"column:name" json:"name"`
	IsFUD            bool       `gorm:"column:is_fud;default:false" json:"is_fud"`
	FUDType          string     `gorm:"column:fud_type" json:"fud_type,omitempty"`
	FUDProbability   float64    `gorm:"column:fud_probability;default:0" json:"fud_probability"`
	IsDetailAnalyzed bool       `gorm:"column:is_detail_analyzed;default:false" json:"is_detail_analyzed"`
	Status           string     `gorm:"column:status;default:'unknown'" json:"status"`
	LastAnalyzedAt   *time.Time `gorm:"column:last_analyzed_at" json:"last_analyzed_at,omitempty"`
	LastMessageID    string     `gorm:"column:last_message_id" json:"last_message_id,omitempty"`
	AnalysisCount    int        `gorm:"column:analysis_count;default:0" json:"analysis_count"`
	FUDMessageCount  int        `gorm:"column:fud_message_count;default:0" json:"fud_message_count"`
	CreatedAt        time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (UserModel) TableName() string {
	return "users"
}

type FUDUserModel struct {
	gorm.Model
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         string    `gorm:"column:user_id;uniqueIndex" json:"user_id"`
	Username       string    `gorm:"column:username" json:"username"`
	FUDType        string    `gorm:"column:fud_type" json:"fud_type"`
	FUDProbability float64   `gorm:"column:fud_probability" json:"fud_probability"`
	DetectedAt     time.Time `gorm:"column:detected_at" json:"detected_at"`
	MessageCount   int       `gorm:"column:message_count;default:1" json:"message_count"`
	LastMessageID  string    `gorm:"column:last_message_id" json:"last_message_id"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (FUDUserModel) TableName() string {
	return "fud_users"
}

type UserRelationModel struct {
	gorm.Model
	UserID        string    `gorm:"column:user_id;index" json:"user_id"`
	RelatedUserID string    `gorm:"column:related_user_id;index" json:"related_user_id"`
	RelationType  string    `gorm:"column:relation_type;index" json:"relation_type"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (UserRelationModel) TableName() string {
	return "user_relations"
}

type UserCommunityActivity struct {
	UserID       string        `json:"user_id"`
	ThreadGroups []ThreadGroup `json:"thread_groups"`
}

type ThreadGroup struct {
	MainPost    ThreadPost  `json:"main_post"`
	UserReplies []UserReply `json:"user_replies"`
}

type ThreadPost struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

type UserReply struct {
	TweetID         string    `json:"tweet_id"`
	Text            string    `json:"text"`
	CreatedAt       time.Time `json:"created_at"`
	InReplyToID     string    `json:"in_reply_to_id"`
	RepliedToAuthor string    `json:"replied_to_author"`
	RepliedToText   string    `json:"replied_to_text"`
}

type AnalysisTaskModel struct {
	gorm.Model
	ID             string     `gorm:"primaryKey;column:id" json:"id"`
	Username       string     `gorm:"column:username;index" json:"username"`
	UserID         string     `gorm:"column:user_id;index" json:"user_id"`
	Status         string     `gorm:"column:status;index" json:"status"`
	CurrentStep    string     `gorm:"column:current_step" json:"current_step"`
	ProgressText   string     `gorm:"column:progress_text" json:"progress_text"`
	TelegramChatID int64      `gorm:"column:telegram_chat_id" json:"telegram_chat_id"`
	MessageID      int64      `gorm:"column:message_id" json:"message_id"`
	ErrorMessage   string     `gorm:"column:error_message" json:"error_message,omitempty"`
	ResultData     string     `gorm:"column:result_data" json:"result_data,omitempty"`
	StartedAt      time.Time  `gorm:"column:started_at" json:"started_at"`
	CompletedAt    *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`
	CreatedAt      time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (AnalysisTaskModel) TableName() string {
	return "analysis_tasks"
}

const (
	ANALYSIS_STATUS_PENDING   = "pending"
	ANALYSIS_STATUS_RUNNING   = "running"
	ANALYSIS_STATUS_COMPLETED = "completed"
	ANALYSIS_STATUS_FAILED    = "failed"
)

const (
	ANALYSIS_STEP_INIT               = "init"
	ANALYSIS_STEP_USER_LOOKUP        = "user_lookup"
	ANALYSIS_STEP_TICKER_SEARCH      = "ticker_search"
	ANALYSIS_STEP_FOLLOWERS          = "followers"
	ANALYSIS_STEP_FOLLOWINGS         = "followings"
	ANALYSIS_STEP_COMMUNITY_ACTIVITY = "community_activity"
	ANALYSIS_STEP_CLAUDE_ANALYSIS    = "claude_analysis"
	ANALYSIS_STEP_SAVING_RESULTS     = "saving_results"
	ANALYSIS_STEP_COMPLETED          = "completed"
)

type CachedAnalysisModel struct {
	gorm.Model
	UserID         string    `gorm:"column:user_id;uniqueIndex" json:"user_id"`
	Username       string    `gorm:"column:username;index" json:"username"`
	IsFUDUser      bool      `gorm:"column:is_fud_user" json:"is_fud_user"`
	FUDType        string    `gorm:"column:fud_type" json:"fud_type"`
	FUDProbability float64   `gorm:"column:fud_probability" json:"fud_probability"`
	UserRiskLevel  string    `gorm:"column:user_risk_level" json:"user_risk_level"`
	UserSummary    string    `gorm:"column:user_summary" json:"user_summary"`
	KeyEvidence    string    `gorm:"column:key_evidence" json:"key_evidence"`
	DecisionReason string    `gorm:"column:decision_reason" json:"decision_reason"`
	AnalyzedAt     time.Time `gorm:"column:analyzed_at;index" json:"analyzed_at"`
	ExpiresAt      time.Time `gorm:"column:expires_at;index" json:"expires_at"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (CachedAnalysisModel) TableName() string {
	return "cached_analysis"
}

type UserTickerOpinionModel struct {
	gorm.Model
	UserID          string    `gorm:"column:user_id;index" json:"user_id"`
	Username        string    `gorm:"column:username;index" json:"username"`
	Ticker          string    `gorm:"column:ticker;index" json:"ticker"`
	TweetID         string    `gorm:"column:tweet_id;uniqueIndex" json:"tweet_id"`
	Text            string    `gorm:"column:text" json:"text"`
	TweetCreatedAt  time.Time `gorm:"column:tweet_created_at;index" json:"tweet_created_at"`
	InReplyToID     string    `gorm:"column:in_reply_to_id" json:"in_reply_to_id,omitempty"`
	RepliedToText   string    `gorm:"column:replied_to_text" json:"replied_to_text,omitempty"`
	RepliedToAuthor string    `gorm:"column:replied_to_author" json:"replied_to_author,omitempty"`
	SearchQuery     string    `gorm:"column:search_query" json:"search_query"`
	FoundAt         time.Time `gorm:"column:found_at;index" json:"found_at"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (UserTickerOpinionModel) TableName() string {
	return "user_ticker_opinions"
}

const (
	USER_STATUS_UNKNOWN       = "unknown"
	USER_STATUS_CLEAN         = "clean"
	USER_STATUS_FUD_CONFIRMED = "fud_confirmed"
	USER_STATUS_ANALYZING     = "analyzing"
)
