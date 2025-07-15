package main

import (
	"encoding/json"
	"fmt"
	"github.com/grutapig/hackaton/twitterapi"
	"log"
	"os"
	"time"
)

const FUD_TYPE = "known_fud_user_activity"

func FirstStepHandler(newMessageCh chan twitterapi.NewMessage, fudChannel chan twitterapi.NewMessage, claudeApi *ClaudeApi, systemPromptFirstStep []byte, userStatusManager *UserStatusManager, dbService *DatabaseService, notificationCh chan FUDAlertNotification) {
	defer close(fudChannel)

	for newMessage := range newMessageCh {
		log.Println("Got a new message:", newMessage.Author.UserName, " - ", newMessage.Text, "parent to:", newMessage.ParentTweet.Text, " grandparent:", newMessage.GrandParentTweet.Text)

		// Check if user has been through detailed analysis before
		isDetailAnalyzed := dbService.IsUserDetailAnalyzed(newMessage.Author.ID)

		// Check if user is already known FUD user
		isKnownFUDUser := dbService.IsFUDUser(newMessage.Author.ID)

		if isKnownFUDUser {
			// Known FUD user - ask Claude for quick analysis before sending notification
			log.Printf("Known FUD user %s - performing quick analysis before notification", newMessage.Author.UserName)

			messages := ClaudeMessages{}
			// Add thread context in order: grandparent -> parent -> current
			if newMessage.GrandParentTweet.ID != "" {
				messages = append(messages, ClaudeMessage{ROLE_USER, "the main post is: " + newMessage.GrandParentTweet.Author + ":" + newMessage.GrandParentTweet.Text})
				messages = append(messages, ClaudeMessage{ROLE_USER, "reply in thread: " + newMessage.ParentTweet.Author + ":" + newMessage.ParentTweet.Text})
			} else {
				messages = append(messages, ClaudeMessage{ROLE_USER, "the main post is: " + newMessage.ParentTweet.Author + ":" + newMessage.ParentTweet.Text})
			}

			messages = append(messages, ClaudeMessage{ROLE_USER, "user reply being analyzed: " + newMessage.Author.UserName + ":" + newMessage.Text})
			messages = append(messages, ClaudeMessage{ROLE_ASSISTANT, "{"})
			systemTicker := os.Getenv(ENV_TWITTER_COMMUNITY_TICKER)
			resp, err := claudeApi.SendMessage(messages, fmt.Sprintf("%s\n<instruction>you must analyze %s user messages in the context of the full thread</instruction> \n this is a FUD user. be more attention for his message and his answers."+"\nthe system ticker is:"+systemTicker+", it cannot be used for any criteria or flag about decision FUD or not", string(systemPromptFirstStep), newMessage.Author.UserName))
			if err != nil {
				log.Printf("error claude quick analysis: %s", err)
				continue
			}

			aiDecision := FirstStepClaudeResponse{}
			err = json.Unmarshal([]byte("{"+resp.Content[0].Text), &aiDecision)
			if err != nil {
				log.Printf("error unmarshaling claude response: %s", err)
				continue
			}

			if aiDecision.IsFud {
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

				// Create quick FUD alert notification (with basic data from first step)
				alert := FUDAlertNotification{
					FUDMessageID:          newMessage.TweetID,
					FUDUserID:             newMessage.Author.ID,
					FUDUsername:           newMessage.Author.UserName,
					ThreadID:              newMessage.ReplyTweetID,
					DetectedAt:            time.Now().Format(time.RFC3339),
					AlertSeverity:         "medium", // Default for known FUD users
					FUDType:               FUD_TYPE,
					FUDProbability:        0, // Convert percentage to decimal
					MessagePreview:        newMessage.Text,
					RecommendedAction:     "MONITOR_ACTIVITY",
					KeyEvidence:           []string{"Known FUD user"},
					DecisionReason:        "Quick analysis of known FUD user activity",
					OriginalPostText:      originalPostText,
					OriginalPostAuthor:    originalPostAuthor,
					ParentPostText:        parentPostText,
					ParentPostAuthor:      parentPostAuthor,
					GrandParentPostText:   grandParentPostText,
					GrandParentPostAuthor: grandParentPostAuthor,
					HasThreadContext:      hasThreadContext,
				}
				log.Printf("Sending quick notification for known FUD user %s", newMessage.Author.UserName)
				notificationCh <- alert
			} else {
				log.Printf("Known FUD user %s - message not FUD, ignoring", newMessage.Author.UserName)
			}
			continue
		}

		if !isDetailAnalyzed {
			// New user - send to detailed analysis
			log.Printf("New user %s - sending directly to detailed analysis", newMessage.Author.UserName)
			userStatusManager.SetUserAnalyzing(newMessage.Author.ID, newMessage.Author.UserName)
			fudChannel <- newMessage
			continue
		}

		// Existing user (not FUD) - standard first step analysis
		log.Printf("Existing user %s - performing first step analysis", newMessage.Author.UserName)
		messages := ClaudeMessages{}

		// Add thread context in order: grandparent -> parent -> current
		if newMessage.GrandParentTweet.ID != "" {
			messages = append(messages, ClaudeMessage{ROLE_USER, "the main post is: " + newMessage.GrandParentTweet.Author + ":" + newMessage.GrandParentTweet.Text})
			messages = append(messages, ClaudeMessage{ROLE_USER, "reply in thread: " + newMessage.ParentTweet.Author + ":" + newMessage.ParentTweet.Text})
		} else {
			messages = append(messages, ClaudeMessage{ROLE_USER, "the main post is: " + newMessage.ParentTweet.Author + ":" + newMessage.ParentTweet.Text})
		}

		messages = append(messages, ClaudeMessage{ROLE_USER, "user reply being analyzed: " + newMessage.Author.UserName + ":" + newMessage.Text})
		messages = append(messages, ClaudeMessage{ROLE_ASSISTANT, "{"})

		resp, err := claudeApi.SendMessage(messages, fmt.Sprintf("%s\n<instruction>you must analyze %s user messages in the context of the full thread</instruction>", string(systemPromptFirstStep), newMessage.Author.UserName))
		if err != nil {
			log.Printf("error claude: %s", err)
			continue
		}

		aiDecision := FirstStepClaudeResponse{}
		err = json.Unmarshal([]byte("{"+resp.Content[0].Text), &aiDecision)
		if err != nil {
			log.Printf("error unmarshaling claude response: %s; raw: %s", err, "{"+resp.Content[0].Text)
			continue
		}

		if aiDecision.IsFud {
			// Send to detailed analysis
			log.Printf("First step flagged user %s as FUD - sending to detailed analysis", newMessage.Author.UserName)
			userStatusManager.SetUserAnalyzing(newMessage.Author.ID, newMessage.Author.UserName)
			fudChannel <- newMessage
		} else {
			log.Printf("First step - user %s message not FUD, ignoring", newMessage.Author.UserName)
		}
	}
}
