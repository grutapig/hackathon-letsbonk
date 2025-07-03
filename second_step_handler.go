package main

import (
	"encoding/json"
	"fmt"
	"github.com/grutapig/hackaton/twitterapi"
	"log"
	"time"
)

func SecondStepHandler(newMessage twitterapi.NewMessage, notificationCh chan FUDAlertNotification, twitterApi *twitterapi.TwitterAPIService, claudeApi *ClaudeApi, systemPromptSecondStep []byte, userStatusManager *UserStatusManager, ticker string, dbService *DatabaseService) {
	// Get user's ticker mentions using advanced search (max 3 pages)
	userTickerMentions := getUserTickerMentions(twitterApi, newMessage.Author.UserName, ticker, dbService)

	// Get user's community activity from database
	userCommunityActivity, err := dbService.GetUserCommunityActivity(newMessage.Author.ID)
	if err != nil {
		log.Printf("Error getting user community activity for %s: %v", newMessage.Author.UserName, err)
		userCommunityActivity = &UserCommunityActivity{
			UserID:       newMessage.Author.ID,
			ThreadGroups: []ThreadGroup{},
		}
	}

	followers, err := twitterApi.GetUserFollowers(twitterapi.UserFollowersRequest{UserName: newMessage.Author.UserName})
	followings, err := twitterApi.GetUserFollowings(twitterapi.UserFollowingsRequest{UserName: newMessage.Author.UserName})

	// Save followers and followings to database
	if followers != nil && len(followers.Followers) > 0 {
		followerIDs := make([]string, len(followers.Followers))
		for i, follower := range followers.Followers {
			followerIDs[i] = follower.Id
			// Also save follower as user if not exists
			if !dbService.UserExists(follower.Id) {
				user := UserModel{
					ID:       follower.Id,
					Username: follower.UserName,
					Name:     follower.Name,
				}
				dbService.SaveUser(user)
			}
		}
		err = dbService.SaveUserRelations(newMessage.Author.ID, followerIDs, RELATION_TYPE_FOLLOWER)
		if err != nil {
			log.Printf("Failed to save followers for user %s: %v", newMessage.Author.UserName, err)
		} else {
			log.Printf("Saved %d followers for user %s", len(followerIDs), newMessage.Author.UserName)
		}
	}

	if followings != nil && len(followings.Followings) > 0 {
		followingIDs := make([]string, len(followings.Followings))
		for i, following := range followings.Followings {
			followingIDs[i] = following.Id
			// Also save following as user if not exists
			if !dbService.UserExists(following.Id) {
				user := UserModel{
					ID:       following.Id,
					Username: following.UserName,
					Name:     following.Name,
				}
				dbService.SaveUser(user)
			}
		}
		err = dbService.SaveUserRelations(newMessage.Author.ID, followingIDs, RELATION_TYPE_FOLLOWING)
		if err != nil {
			log.Printf("Failed to save followings for user %s: %v", newMessage.Author.UserName, err)
		} else {
			log.Printf("Saved %d followings for user %s", len(followingIDs), newMessage.Author.UserName)
		}
	}

	// Prepare claude request with community activity
	claudeMessages := PrepareClaudeSecondStepRequest(userTickerMentions, followers, followings, userStatusManager, userCommunityActivity)

	// Add thread context in order: grandparent -> parent -> current
	if newMessage.GrandParentTweet.ID != "" {
		claudeMessages = append(claudeMessages, ClaudeMessage{ROLE_USER, "the main post is: " + newMessage.GrandParentTweet.Author + ":" + newMessage.GrandParentTweet.Text})
		claudeMessages = append(claudeMessages, ClaudeMessage{ROLE_USER, "reply in thread: " + newMessage.ParentTweet.Author + ":" + newMessage.ParentTweet.Text})
	} else {
		claudeMessages = append(claudeMessages, ClaudeMessage{ROLE_USER, "the main post is: " + newMessage.ParentTweet.Author + ":" + newMessage.ParentTweet.Text})
	}

	claudeMessages = append(claudeMessages, ClaudeMessage{ROLE_USER, "user reply being analyzed: " + newMessage.Author.UserName + ":" + newMessage.Text})
	claudeMessages = append(claudeMessages, ClaudeMessage{Role: ROLE_ASSISTANT, Content: "{"})
	pretty, _ := json.MarshalIndent(claudeMessages, "", "\t")
	fmt.Println("send to analyze:", string(pretty))
	//fmt.Println("send to analyze:")
	resp, err := claudeApi.SendMessage(claudeMessages, string(systemPromptSecondStep)+"analyzed user is "+newMessage.Author.UserName)
	aiDecision2 := SecondStepClaudeResponse{}
	fmt.Println("claude make a decision for this user:", resp, err)

	if err != nil {
		log.Printf("error claude second step: %s", err)
		return
	}

	err = json.Unmarshal([]byte("{"+resp.Content[0].Text), &aiDecision2)
	if err != nil {
		log.Printf("error unmarshaling claude response: %s", err)
		return
	}
	pretty, _ = json.MarshalIndent(aiDecision2, "", "\t")
	fmt.Println(string(pretty))

	// Update user status after analysis
	userStatusManager.UpdateUserAfterAnalysis(newMessage.Author.ID, newMessage.Author.UserName, aiDecision2, newMessage.TweetID)

	if aiDecision2.IsFUDUser {
		// Store FUD user in database
		if aiDecision2.IsFUDUser {
			fudUser := FUDUserModel{
				UserID:         newMessage.Author.ID,
				Username:       newMessage.Author.UserName,
				FUDType:        aiDecision2.FUDType,
				FUDProbability: aiDecision2.FUDProbability,
				DetectedAt:     time.Now(),
				MessageCount:   1,
				LastMessageID:  newMessage.TweetID,
			}

			// Check if FUD user already exists
			if dbService.IsFUDUser(newMessage.Author.ID) {
				// Increment message count for existing FUD user
				err = dbService.IncrementFUDUserMessageCount(newMessage.Author.ID, newMessage.TweetID)
				if err != nil {
					log.Printf("Failed to increment FUD user message count: %v", err)
				}
			} else {
				// Save new FUD user
				err = dbService.SaveFUDUser(fudUser)
				if err != nil {
					log.Printf("Failed to save FUD user: %v", err)
				} else {
					log.Printf("Stored new FUD user: %s", newMessage.Author.UserName)
				}
			}
		}

		// Determine thread context from newMessage
		originalPostText := ""
		originalPostAuthor := ""
		parentPostText := ""
		parentPostAuthor := ""
		grandParentPostText := ""
		grandParentPostAuthor := ""
		hasThreadContext := false

		// Set thread context based on available data
		if newMessage.GrandParentTweet.ID != "" {
			grandParentPostText = newMessage.GrandParentTweet.Text
			grandParentPostAuthor = newMessage.GrandParentTweet.Author
			parentPostText = newMessage.ParentTweet.Text
			parentPostAuthor = newMessage.ParentTweet.Author
			originalPostText = newMessage.GrandParentTweet.Text
			originalPostAuthor = newMessage.GrandParentTweet.Author
			hasThreadContext = true
		} else if newMessage.ParentTweet.ID != "" {
			parentPostText = newMessage.ParentTweet.Text
			parentPostAuthor = newMessage.ParentTweet.Author
			originalPostText = newMessage.ParentTweet.Text
			originalPostAuthor = newMessage.ParentTweet.Author
			hasThreadContext = true
		}

		// Create FUD alert notification
		alert := FUDAlertNotification{
			FUDMessageID:          newMessage.TweetID,
			FUDUserID:             newMessage.Author.ID,
			FUDUsername:           newMessage.Author.UserName,
			ThreadID:              newMessage.ReplyTweetID,
			DetectedAt:            time.Now().Format(time.RFC3339),
			AlertSeverity:         mapRiskLevelToSeverity(aiDecision2.UserRiskLevel),
			FUDType:               aiDecision2.FUDType,
			FUDProbability:        aiDecision2.FUDProbability,
			MessagePreview:        newMessage.Text,
			RecommendedAction:     getRecommendedAction(aiDecision2),
			KeyEvidence:           aiDecision2.KeyEvidence,
			DecisionReason:        aiDecision2.DecisionReason,
			OriginalPostText:      originalPostText,
			OriginalPostAuthor:    originalPostAuthor,
			ParentPostText:        parentPostText,
			ParentPostAuthor:      parentPostAuthor,
			GrandParentPostText:   grandParentPostText,
			GrandParentPostAuthor: grandParentPostAuthor,
			HasThreadContext:      hasThreadContext,
		}
		notificationCh <- alert
	}

	// Mark user as having been through detailed analysis
	err = dbService.MarkUserAsDetailAnalyzed(newMessage.Author.ID)
	if err != nil {
		log.Printf("Failed to mark user %s as detail analyzed: %v", newMessage.Author.UserName, err)
	} else {
		log.Printf("Marked user %s as detail analyzed", newMessage.Author.UserName)
	}
}

func mapRiskLevelToSeverity(riskLevel string) string {
	switch riskLevel {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

func getRecommendedAction(decision SecondStepClaudeResponse) string {
	if decision.UserRiskLevel == "critical" {
		return "IMMEDIATE_ACTION_REQUIRED"
	} else if decision.UserRiskLevel == "high" {
		return "MONITOR_CLOSELY"
	} else {
		return "STANDARD_MONITORING"
	}
}
