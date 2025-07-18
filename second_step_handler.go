package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/grutapig/hackaton/twitterapi"
	"log"
	"os"
	"time"
)

func SecondStepHandler(newMessage twitterapi.NewMessage, notificationCh chan FUDAlertNotification, twitterApi *twitterapi.TwitterAPIService, claudeApi *ClaudeApi, systemPromptSecondStep []byte, userStatusManager *UserStatusManager, ticker string, dbService *DatabaseService, loggingService *LoggingService) {
	// Generate UUID for this request processing
	requestUUID := uuid.New().String()

	// Start request processing logging
	if loggingService != nil {
		loggingService.StartRequestProcessing(requestUUID, newMessage.Author.ID, newMessage.Author.UserName, newMessage.TweetID, PROCESSING_TYPE_DETAILED, 5)
	}

	// Check if we have cached analysis first (for non-manual analysis)
	if !newMessage.IsManualAnalysis {
		if cachedResult, err := dbService.GetCachedAnalysis(newMessage.Author.ID); err == nil {
			log.Printf("Using cached analysis for user %s", newMessage.Author.UserName)

			// Use cached result instead of running full analysis
			aiDecision2 := *cachedResult

			// Update user status with cached result
			userStatusManager.UpdateUserAfterAnalysis(newMessage.Author.ID, newMessage.Author.UserName, aiDecision2, newMessage.TweetID)

			// If user is not FUD but was previously marked as FUD, remove from FUD list
			if !aiDecision2.IsFUDUser {
				if dbService.IsFUDUser(newMessage.Author.ID) {
					err := dbService.DeleteFUDUser(newMessage.Author.ID)
					if err != nil {
						log.Printf("Failed to remove user %s from FUD list: %v", newMessage.Author.UserName, err)
					} else {
						log.Printf("Removed user %s from FUD list - cached analysis shows user is clean", newMessage.Author.UserName)
					}

					// Also update user model to mark as not FUD
					err = dbService.UpdateUserFUDStatus(newMessage.Author.ID, false, "")
					if err != nil {
						log.Printf("Failed to update FUD status for user %s: %v", newMessage.Author.UserName, err)
					}
				}
			}

			// Send notification if needed
			if aiDecision2.IsFUDUser || newMessage.ForceNotification {
				sendCachedNotification(newMessage, aiDecision2, notificationCh, dbService)
			}

			// Mark user as analyzed
			dbService.MarkUserAsDetailAnalyzed(newMessage.Author.ID)

			// Complete manual analysis task if this was a manual analysis
			if newMessage.IsManualAnalysis && newMessage.TaskID != "" {
				completeManualAnalysisTask(newMessage, aiDecision2, dbService)
			}

			return
		}
	}

	// Get user's ticker mentions using advanced search (max 3 pages)
	startTime := time.Now()
	userTickerMentions := getUserTickerMentions(twitterApi, newMessage.Author.UserName, ticker, dbService)
	collectionTime := int(time.Since(startTime).Milliseconds())

	// Log ticker mentions data collection
	if loggingService != nil {
		dataCount := 0
		if userTickerMentions != nil {
			dataCount = len(userTickerMentions.UserMessages)
		}
		loggingService.LogDataCollection(requestUUID, newMessage.Author.ID, newMessage.Author.UserName, DATA_TYPE_TICKER_MENTIONS, dataCount, 0, collectionTime, userTickerMentions != nil, "", fmt.Sprintf("{\"ticker\":\"%s\"}", ticker))
	}

	// Get user's community activity from database
	startTime = time.Now()
	userCommunityActivity, err := dbService.GetUserCommunityActivity(newMessage.Author.ID)
	collectionTime = int(time.Since(startTime).Milliseconds())

	if err != nil {
		log.Printf("Error getting user community activity for %s: %v", newMessage.Author.UserName, err)
		userCommunityActivity = &UserCommunityActivity{
			UserID:       newMessage.Author.ID,
			ThreadGroups: []ThreadGroup{},
		}
	}

	// Log community activity data collection
	if loggingService != nil {
		dataCount := 0
		if userCommunityActivity != nil {
			dataCount = len(userCommunityActivity.ThreadGroups)
		}
		loggingService.LogDataCollection(requestUUID, newMessage.Author.ID, newMessage.Author.UserName, DATA_TYPE_COMMUNITY_ACTIVITY, dataCount, 0, collectionTime, err == nil, func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(), "")
	}

	// Get user's followers
	startTime = time.Now()
	followers, err := twitterApi.GetUserFollowers(twitterapi.UserFollowersRequest{UserName: newMessage.Author.UserName})
	collectionTime = int(time.Since(startTime).Milliseconds())

	// Log followers data collection
	if loggingService != nil {
		dataCount := 0
		if followers != nil {
			dataCount = len(followers.Followers)
		}
		loggingService.LogDataCollection(requestUUID, newMessage.Author.ID, newMessage.Author.UserName, DATA_TYPE_FOLLOWERS, dataCount, 0, collectionTime, err == nil, func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(), "")
	}

	// Get user's followings
	startTime = time.Now()
	followings, err := twitterApi.GetUserFollowings(twitterapi.UserFollowingsRequest{UserName: newMessage.Author.UserName})
	collectionTime = int(time.Since(startTime).Milliseconds())

	// Log followings data collection
	if loggingService != nil {
		dataCount := 0
		if followings != nil {
			dataCount = len(followings.Followings)
		}
		loggingService.LogDataCollection(requestUUID, newMessage.Author.ID, newMessage.Author.UserName, DATA_TYPE_FOLLOWING, dataCount, 0, collectionTime, err == nil, func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(), "")
	}

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
	systemPromptModified := string(systemPromptSecondStep)
	if newMessage.IsManualAnalysis {
		systemPromptModified += "\n\nIMPORTANT: This is a MANUAL ANALYSIS REQUEST initiated by an administrator. Please provide a thorough analysis regardless of normal filtering criteria."
	}
	systemPromptModified += " analyzed user is " + newMessage.Author.UserName
	systemTicker := os.Getenv(ENV_TWITTER_COMMUNITY_TICKER)

	// Log AI request
	startTime = time.Now()
	resp, err := claudeApi.SendMessage(claudeMessages, systemPromptModified+"\nthe system ticker is:"+systemTicker+", it cannot be used for any criteria or flag about decision FUD or not")
	processingTime := int(time.Since(startTime).Milliseconds())

	if loggingService != nil {
		loggingService.LogAIRequest(requestUUID, newMessage.Author.ID, newMessage.Author.UserName, newMessage.TweetID, REQUEST_TYPE_SECOND_STEP, 2, claudeMessages, resp, 0, processingTime, err == nil, func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}())
	}

	aiDecision2 := SecondStepClaudeResponse{}
	fmt.Println("claude make a decision for this user:", resp, err)

	if err != nil {
		if loggingService != nil {
			loggingService.UpdateRequestProcessingStatus(requestUUID, PROCESSING_STATUS_FAILED, 4)
		}
		failManualAnalysisTask(newMessage, err, dbService)
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

	// If user is not FUD but was previously marked as FUD, remove from FUD list
	if !aiDecision2.IsFUDUser {
		if dbService.IsFUDUser(newMessage.Author.ID) {
			err := dbService.DeleteFUDUser(newMessage.Author.ID)
			if err != nil {
				log.Printf("Failed to remove user %s from FUD list: %v", newMessage.Author.UserName, err)
			} else {
				log.Printf("Removed user %s from FUD list - analysis shows user is clean", newMessage.Author.UserName)
			}

			// Also update user model to mark as not FUD
			err = dbService.UpdateUserFUDStatus(newMessage.Author.ID, false, "")
			if err != nil {
				log.Printf("Failed to update FUD status for user %s: %v", newMessage.Author.UserName, err)
			}
		}
	}

	if aiDecision2.IsFUDUser || newMessage.ForceNotification {
		// Store FUD user in database only if actually detected as FUD
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

		// Create alert notification
		alertType := aiDecision2.FUDType
		alertSeverity := mapRiskLevelToSeverity(aiDecision2.UserRiskLevel)

		if newMessage.IsManualAnalysis {
			if !aiDecision2.IsFUDUser {
				alertType = "manual_analysis_clean"
				alertSeverity = "low"
			} else {
				alertType = "manual_analysis_fud"
			}
		}

		alert := FUDAlertNotification{
			FUDMessageID:          newMessage.TweetID,
			FUDUserID:             newMessage.Author.ID,
			FUDUsername:           newMessage.Author.UserName,
			ThreadID:              newMessage.ReplyTweetID,
			DetectedAt:            time.Now().Format(time.RFC3339),
			AlertSeverity:         alertSeverity,
			FUDType:               alertType,
			FUDProbability:        aiDecision2.FUDProbability,
			MessagePreview:        newMessage.Text,
			RecommendedAction:     getRecommendedAction(aiDecision2),
			KeyEvidence:           aiDecision2.KeyEvidence,
			DecisionReason:        aiDecision2.DecisionReason,
			UserSummary:           aiDecision2.UserSummary,
			OriginalPostText:      originalPostText,
			OriginalPostAuthor:    originalPostAuthor,
			ParentPostText:        parentPostText,
			ParentPostAuthor:      parentPostAuthor,
			GrandParentPostText:   grandParentPostText,
			GrandParentPostAuthor: grandParentPostAuthor,
			HasThreadContext:      hasThreadContext,
			TargetChatID:          newMessage.TelegramChatID, // Set target chat if specified
		}
		notificationCh <- alert
	}

	// Save analysis result to cache (24-hour expiration)
	err = dbService.SaveCachedAnalysis(newMessage.Author.ID, newMessage.Author.UserName, aiDecision2)
	if err != nil {
		log.Printf("Failed to save cached analysis for user %s: %v", newMessage.Author.UserName, err)
	} else {
		log.Printf("Saved cached analysis for user %s", newMessage.Author.UserName)
	}

	// Mark user as having been through detailed analysis
	err = dbService.MarkUserAsDetailAnalyzed(newMessage.Author.ID)
	if err != nil {
		log.Printf("Failed to mark user %s as detail analyzed: %v", newMessage.Author.UserName, err)
	} else {
		log.Printf("Marked user %s as detail analyzed", newMessage.Author.UserName)
	}

	// Complete manual analysis task if this was a manual analysis
	if newMessage.IsManualAnalysis && newMessage.TaskID != "" {
		completeManualAnalysisTask(newMessage, aiDecision2, dbService)
	}

	// Update request processing status as completed
	if loggingService != nil {
		loggingService.UpdateRequestProcessingStatus(requestUUID, PROCESSING_STATUS_COMPLETED, 5)
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

// sendCachedNotification sends notification using cached analysis result
func sendCachedNotification(newMessage twitterapi.NewMessage, aiDecision2 SecondStepClaudeResponse, notificationCh chan FUDAlertNotification, dbService *DatabaseService) {
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

	// Store FUD user in database only if actually detected as FUD
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
			err := dbService.IncrementFUDUserMessageCount(newMessage.Author.ID, newMessage.TweetID)
			if err != nil {
				log.Printf("Failed to increment FUD user message count: %v", err)
			}
		} else {
			// Save new FUD user
			err := dbService.SaveFUDUser(fudUser)
			if err != nil {
				log.Printf("Failed to save FUD user: %v", err)
			} else {
				log.Printf("Stored new FUD user: %s", newMessage.Author.UserName)
			}
		}
	}

	// Create alert notification
	alertType := aiDecision2.FUDType
	alertSeverity := mapRiskLevelToSeverity(aiDecision2.UserRiskLevel)

	if newMessage.IsManualAnalysis {
		if !aiDecision2.IsFUDUser {
			alertType = "manual_analysis_clean"
			alertSeverity = "low"
		} else {
			alertType = "manual_analysis_fud"
		}
	}

	alert := FUDAlertNotification{
		FUDMessageID:          newMessage.TweetID,
		FUDUserID:             newMessage.Author.ID,
		FUDUsername:           newMessage.Author.UserName,
		ThreadID:              newMessage.ReplyTweetID,
		DetectedAt:            time.Now().Format(time.RFC3339),
		AlertSeverity:         alertSeverity,
		FUDType:               alertType,
		FUDProbability:        aiDecision2.FUDProbability,
		MessagePreview:        newMessage.Text,
		RecommendedAction:     getRecommendedAction(aiDecision2),
		KeyEvidence:           aiDecision2.KeyEvidence,
		DecisionReason:        aiDecision2.DecisionReason,
		UserSummary:           aiDecision2.UserSummary,
		OriginalPostText:      originalPostText,
		OriginalPostAuthor:    originalPostAuthor,
		ParentPostText:        parentPostText,
		ParentPostAuthor:      parentPostAuthor,
		GrandParentPostText:   grandParentPostText,
		GrandParentPostAuthor: grandParentPostAuthor,
		HasThreadContext:      hasThreadContext,
		TargetChatID:          newMessage.TelegramChatID, // Set target chat if specified
	}
	notificationCh <- alert
}

// completeManualAnalysisTask completes manual analysis task with results
func completeManualAnalysisTask(newMessage twitterapi.NewMessage, aiDecision2 SecondStepClaudeResponse, dbService *DatabaseService) {
	// Update progress to saving results
	dbService.UpdateAnalysisTaskProgress(newMessage.TaskID, ANALYSIS_STEP_SAVING_RESULTS, "Analysis completed, saving results...")

	// Complete the task with analysis results
	resultData := fmt.Sprintf(`{"analysis_complete": true, "is_fud": %t, "fud_type": "%s", "user_summary": "%s", "timestamp": "%s"}`,
		aiDecision2.IsFUDUser, aiDecision2.FUDType, aiDecision2.UserSummary, time.Now().Format(time.RFC3339))

	err := dbService.CompleteAnalysisTask(newMessage.TaskID, resultData)
	if err != nil {
		log.Printf("Failed to complete analysis task %s: %v", newMessage.TaskID, err)
	} else {
		log.Printf("Completed manual analysis task %s for user %s", newMessage.TaskID, newMessage.Author.UserName)
	}
}
func failManualAnalysisTask(newMessage twitterapi.NewMessage, err error, dbService *DatabaseService) {
	dbService.SetAnalysisTaskError(newMessage.TaskID, err.Error())
}
