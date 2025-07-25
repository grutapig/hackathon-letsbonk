package main

import "time"

type Tweet struct {
	TweetID   string `gorm:"primaryKey;column:tweet_id"`
	Text      string `gorm:"type:text"`
	Username  string `gorm:"index"`
	UserID    string `gorm:"index"`
	CreatedAt time.Time
}
