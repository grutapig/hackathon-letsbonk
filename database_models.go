package main

import (
	"gorm.io/gorm"
	"time"
)

// Tweet model for database storage
type TweetModel struct {
	gorm.Model
	ID            string    `gorm:"primaryKey;column:id" json:"id"` // Twitter ID as unique index
	Text          string    `gorm:"column:text" json:"text"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	ReplyCount    int       `gorm:"column:reply_count" json:"reply_count"`
	UserID        string    `gorm:"column:user_id;index" json:"user_id"`
	InReplyToID   string    `gorm:"column:in_reply_to_id;index" json:"in_reply_to_id,omitempty"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`
	SourceType    string    `gorm:"column:source_type;index" json:"source_type"`       // "community", "ticker_search", "context", "monitoring"
	TickerMention string    `gorm:"column:ticker_mention;index" json:"ticker_mention"` // Тикер, если твит получен через поиск
	SearchQuery   string    `gorm:"column:search_query" json:"search_query,omitempty"` // Оригинальный запрос поиска
}

func (TweetModel) TableName() string {
	return "tweets"
}

// User model for database storage
type UserModel struct {
	gorm.Model
	ID               string    `gorm:"primaryKey;column:id" json:"id"`
	Username         string    `gorm:"column:username;uniqueIndex" json:"username"`
	Name             string    `gorm:"column:name" json:"name"`
	IsFUD            bool      `gorm:"column:is_fud;default:false" json:"is_fud"`
	FUDType          string    `gorm:"column:fud_type" json:"fud_type,omitempty"`
	IsDetailAnalyzed bool      `gorm:"column:is_detail_analyzed;default:false" json:"is_detail_analyzed"` // Has user been through detailed analysis
	CreatedAt        time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (UserModel) TableName() string {
	return "users"
}

// FUDUser model for storing FUD users information
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

// UserRelation model for storing user connections (followers/following)
type UserRelationModel struct {
	gorm.Model
	UserID        string    `gorm:"column:user_id;index" json:"user_id"`
	RelatedUserID string    `gorm:"column:related_user_id;index" json:"related_user_id"`
	RelationType  string    `gorm:"column:relation_type;index" json:"relation_type"` // "follower" or "following"
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (UserRelationModel) TableName() string {
	return "user_relations"
}

// Structures for user community activity analysis
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
