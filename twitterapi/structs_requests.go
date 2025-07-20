package twitterapi

import "time"

type CommunityTweetsRequest struct {
	CommunityID string `json:"community_id"`
	Cursor      string `json:"cursor,omitempty"`
}

type TweetRepliesRequest struct {
	TweetID   string `json:"tweet_id"`
	Cursor    string `json:"cursor,omitempty"`
	SinceTime int64
}

type UserLastTweetsRequest struct {
	UserId         string
	UserName       string
	Cursor         string
	IncludeReplies bool
}

type UserFollowersRequest struct {
	UserName string
	Cursor   string
	PageSize int
}
type UserFollowingsRequest struct {
	UserName string
	Cursor   string
	PageSize int
}

type NewMessage struct {
	TweetID      string
	ReplyTweetID string
	Author       struct {
		UserName string
		Name     string
		ID       string
	}
	Text            string
	CreatedAt       string
	CreatedAtParsed time.Time
	ParentTweet     struct {
		ID     string
		Author string
		Text   string
	}
	GrandParentTweet struct {
		ID     string
		Author string
		Text   string
	}
	ReplyCount        int
	LikeCount         int
	RetweetCount      int
	IsManualAnalysis  bool
	ForceNotification bool
	TaskID            string
	TelegramChatID    int64
}
type PostTweetRequest struct {
	AuthSession      string `json:"auth_session"`
	TweetText        string `json:"tweet_text"`
	QuoteTweetId     string `json:"quote_tweet_id"`
	InReplyToTweetId string `json:"in_reply_to_tweet_id"`
	MediaId          string `json:"media_id"`
	Proxy            string `json:"proxy"`
}

const (
	MessageTypeNewPost  = "new_post"
	MessageTypeNewReply = "new_reply"
)

type TweetState struct {
	ID         string
	ReplyCount int
	LastCheck  time.Time
	SinceTime  time.Time
}

type TweetsByIdsResponse struct {
	Tweets  []Tweet `json:"tweets"`
	Status  string  `json:"status"`
	Message string  `json:"message"`
}

const LATEST = "Latest"
const TOP = "Top"

type AdvancedSearchRequest struct {
	Query     string `json:"query"`
	QueryType string `json:"query_type,omitempty"`
	Cursor    string `json:"cursor,omitempty"`
}
