package main

import (
	"fmt"
	"strings"
	"time"
)

type NotificationFormatter struct{}

type FUDAlertNotification struct {
	FUDMessageID      string   `json:"fud_message_id"`
	FUDUserID         string   `json:"fud_user_id"`
	FUDUsername       string   `json:"fud_username"`
	ThreadID          string   `json:"thread_id"`
	DetectedAt        string   `json:"detected_at"`
	AlertSeverity     string   `json:"alert_severity"`
	FUDType           string   `json:"fud_type"`
	FUDProbability    float64  `json:"fud_probability"`
	MessagePreview    string   `json:"message_preview"`
	RecommendedAction string   `json:"recommended_action"`
	KeyEvidence       []string `json:"key_evidence"`
	DecisionReason    string   `json:"decision_reason"`
	UserSummary       string   `json:"user_summary"` // Short conclusion about user type
	// Thread context fields
	OriginalPostText      string `json:"original_post_text"`
	OriginalPostAuthor    string `json:"original_post_author"`
	ParentPostText        string `json:"parent_post_text"`
	ParentPostAuthor      string `json:"parent_post_author"`
	GrandParentPostText   string `json:"grandparent_post_text"`
	GrandParentPostAuthor string `json:"grandparent_post_author"`
	HasThreadContext      bool   `json:"has_thread_context"`
	// Target chat for notification (optional)
	TargetChatID int64 `json:"target_chat_id,omitempty"` // If set, send only to this chat
}

func NewNotificationFormatter() *NotificationFormatter {
	return &NotificationFormatter{}
}

func (nf *NotificationFormatter) FormatForTelegram(alert FUDAlertNotification) string {
	severityEmoji := nf.getSeverityEmoji(alert.AlertSeverity)
	typeEmoji := nf.getFUDTypeEmoji(alert.FUDType)

	// Build context section if available
	contextSection := ""
	if alert.HasThreadContext {
		if alert.GrandParentPostText != "" {
			// Show grandparent -> parent -> current structure
			contextSection = fmt.Sprintf(`

📄 <b>Thread Context:</b>
<b>Root:</b> <i>%s</i> - @%s
<b>Reply:</b> <i>%s</i> - @%s`,
				nf.truncateText(alert.GrandParentPostText, 80),
				alert.GrandParentPostAuthor,
				nf.truncateText(alert.ParentPostText, 80),
				alert.ParentPostAuthor)
		} else if alert.OriginalPostText != "" || alert.ParentPostText != "" {
			// Show parent -> current structure
			postText := alert.OriginalPostText
			postAuthor := alert.OriginalPostAuthor
			if postText == "" {
				postText = alert.ParentPostText
				postAuthor = alert.ParentPostAuthor
			}
			contextSection = fmt.Sprintf(`

📄 <b>Original Post Context:</b>
<i>%s</i> - @%s`,
				nf.truncateText(postText, 100),
				postAuthor)
		}
	}

	// Determine if this is a FUD alert or clean analysis
	isFUDAlert := !strings.Contains(alert.FUDType, "manual_analysis_clean") && alert.FUDType != "none"

	var alertTitle, typeSection string
	if isFUDAlert {
		alertTitle = fmt.Sprintf("%s <b>FUD ALERT - %s SEVERITY</b>", severityEmoji, strings.ToUpper(alert.AlertSeverity))
		typeSection = fmt.Sprintf("%s <b>Attack Type:</b> %s\n👤 <b>User Profile:</b> %s", typeEmoji, nf.formatFUDType(alert.FUDType), alert.UserSummary)
	} else {
		alertTitle = fmt.Sprintf("✅ <b>ANALYSIS COMPLETE - USER CLEAN</b>")
		typeSection = fmt.Sprintf("👤 <b>User Type:</b> %s", alert.UserSummary)
	}

	message := fmt.Sprintf(`%s

%s
🎯 <b>User:</b> @%s
📊 <b>Confidence:</b> %.0f%%
⚡ <b>Action:</b> %s

💬 <b>Message:</b>
<i>%s</i>%s

🔗 <b>Links:</b>
• <a href="https://twitter.com/%s/status/%s">Message</a>
• <a href="https://twitter.com/user/status/%s">Original Thread</a>

🔍 <b>Investigation:</b>
• /history_%s - View recent messages
• /export_%s - Export full history

⏰ <b>Detected:</b> %s
🆔 <b>IDs:</b> User: %s | Tweet: %s`,
		alertTitle,
		typeSection,
		alert.FUDUsername,
		alert.FUDProbability*100,
		alert.RecommendedAction,
		nf.truncateText(alert.MessagePreview, 120),
		contextSection,
		alert.FUDUsername, alert.FUDMessageID,
		alert.ThreadID,
		alert.FUDUsername, alert.FUDUsername,
		nf.formatTime(alert.DetectedAt),
		alert.FUDUserID, alert.FUDMessageID)

	return message
}

func (nf *NotificationFormatter) FormatForTelegramWithDetail(alert FUDAlertNotification, notificationID string) string {
	severityEmoji := nf.getSeverityEmoji(alert.AlertSeverity)
	typeEmoji := nf.getFUDTypeEmoji(alert.FUDType)

	// Determine if this is a FUD alert or clean analysis
	isFUDAlert := !strings.Contains(alert.FUDType, "manual_analysis_clean") && alert.FUDType != "none"

	var alertTitle, typeSection string
	if isFUDAlert {
		alertTitle = fmt.Sprintf("%s <b>FUD ALERT - %s SEVERITY</b>", severityEmoji, strings.ToUpper(alert.AlertSeverity))
		typeSection = fmt.Sprintf("%s <b>Attack Type:</b> %s\n👤 <b>User Profile:</b> %s", typeEmoji, nf.formatFUDType(alert.FUDType), alert.UserSummary)
	} else {
		alertTitle = fmt.Sprintf("✅ <b>ANALYSIS COMPLETE - USER CLEAN</b>")
		typeSection = fmt.Sprintf("👤 <b>User Type:</b> %s", alert.UserSummary)
	}

	message := fmt.Sprintf(`%s

%s
🎯 <b>User:</b> @%s
📊 <b>Confidence:</b> %.0f%%
⚡ <b>Action:</b> %s

💬 <b>Message Preview:</b>
<i>%s</i>

🔗 <b>Quick Links:</b>
• <a href="https://twitter.com/%s/status/%s">Message</a>
• <a href="https://twitter.com/user/status/%s">Original Thread</a>

🔍 <b>Investigation Commands:</b>
• /detail_%s - Detailed analysis
• /history_%s - View recent messages  
• /export_%s - Export full history

⏰ <b>Detected:</b> %s`,
		alertTitle,
		typeSection,
		alert.FUDUsername,
		alert.FUDProbability*100,
		alert.RecommendedAction,
		nf.truncateText(alert.MessagePreview, 120),
		alert.FUDUsername, alert.FUDMessageID,
		alert.ThreadID,
		notificationID, alert.FUDUsername, alert.FUDUsername,
		nf.formatTime(alert.DetectedAt))

	return message
}

func (nf *NotificationFormatter) FormatDetailedView(alert FUDAlertNotification) string {
	severityEmoji := nf.getSeverityEmoji(alert.AlertSeverity)
	typeEmoji := nf.getFUDTypeEmoji(alert.FUDType)

	// Format key evidence
	var evidenceList string
	for i, evidence := range alert.KeyEvidence {
		evidenceList += fmt.Sprintf("  %d. %s\n", i+1, evidence)
	}
	if evidenceList == "" {
		evidenceList = "  No specific evidence provided\n"
	}

	// Build thread context section for detailed view
	threadContextSection := ""
	if alert.HasThreadContext {
		if alert.GrandParentPostText != "" {
			// Show full thread: grandparent -> parent -> current
			threadContextSection = fmt.Sprintf(`

📄 <b>FULL THREAD CONTEXT</b>
🏠 <b>Root Post:</b> @%s
📝 <i>%s</i>

💬 <b>Parent Reply:</b> @%s
📝 <i>%s</i>`,
				alert.GrandParentPostAuthor,
				alert.GrandParentPostText,
				alert.ParentPostAuthor,
				alert.ParentPostText)
		} else if alert.OriginalPostText != "" || alert.ParentPostText != "" {
			// Show single parent context
			postText := alert.OriginalPostText
			postAuthor := alert.OriginalPostAuthor
			if postText == "" {
				postText = alert.ParentPostText
				postAuthor = alert.ParentPostAuthor
			}
			threadContextSection = fmt.Sprintf(`

📄 <b>ORIGINAL POST (FULL TEXT)</b>
👤 Author: @%s
📝 Content: <i>%s</i>`,
				postAuthor,
				postText)
		}
	}

	// Determine if this is a FUD alert or clean analysis
	isFUDAlert := !strings.Contains(alert.FUDType, "manual_analysis_clean") && alert.FUDType != "none"

	var analysisTitle, classificationSection string
	if isFUDAlert {
		analysisTitle = fmt.Sprintf("%s <b>DETAILED FUD ANALYSIS</b>", severityEmoji)
		classificationSection = fmt.Sprintf(`🏷️ <b>CLASSIFICATION</b>
%s Type: %s
🎯 Target User: @%s (ID: %s)
📊 Confidence Level: %.1f%%
🚨 Risk Level: %s
⚡ Recommended Action: %s`, typeEmoji, nf.formatFUDType(alert.FUDType), alert.FUDUsername, alert.FUDUserID, alert.FUDProbability*100, strings.ToUpper(alert.AlertSeverity), alert.RecommendedAction)
	} else {
		analysisTitle = fmt.Sprintf("✅ <b>DETAILED USER ANALYSIS - CLEAN</b>")
		classificationSection = fmt.Sprintf(`👤 <b>USER CLASSIFICATION</b>
✅ Status: Not a FUD user
👤 User Type: %s
🎯 Analyzed User: @%s (ID: %s)
📊 Confidence Level: %.1f%%
⚡ Recommended Action: %s`, alert.UserSummary, alert.FUDUsername, alert.FUDUserID, alert.FUDProbability*100, alert.RecommendedAction)
	}

	var messageTitle string
	if isFUDAlert {
		messageTitle = "💬 <b>FUD MESSAGE (FULL TEXT)</b>"
	} else {
		messageTitle = "💬 <b>ANALYZED MESSAGE (FULL TEXT)</b>"
	}

	message := fmt.Sprintf(`%s

%s

%s
<i>%s</i>%s

🔍 <b>KEY EVIDENCE</b>
%s

🧠 <b>AI DECISION REASONING</b>
<i>%s</i>

🔗 <b>INVESTIGATION LINKS</b>
• <a href="https://twitter.com/%s/status/%s">View Message</a>
• <a href="https://twitter.com/user/status/%s">View Original Thread</a>
• <a href="https://twitter.com/%s">User Profile</a>

🕵️ <b>INVESTIGATION COMMANDS</b>
• /history_%s - View recent messages
• /export_%s - Export full history

📅 <b>DETECTION METADATA</b>
Detected At: %s
Message ID: %s
Thread ID: %s
User ID: %s`,
		analysisTitle,
		classificationSection,
		messageTitle,
		alert.MessagePreview,
		threadContextSection,
		evidenceList,
		alert.DecisionReason,
		alert.FUDUsername, alert.FUDMessageID,
		alert.ThreadID,
		alert.FUDUsername,
		alert.FUDUsername, alert.FUDUsername,
		nf.formatTime(alert.DetectedAt),
		alert.FUDMessageID,
		alert.ThreadID,
		alert.FUDUserID)

	return message
}

func (nf *NotificationFormatter) FormatForTwitterDM(alert FUDAlertNotification) string {
	severityEmoji := nf.getSeverityEmoji(alert.AlertSeverity)

	message := fmt.Sprintf(`%s FUD ALERT - %s

User: @%s (%s)
Type: %s (%.0f%% confidence)
Action: %s

Message: "%s"

Links:
- FUD: https://twitter.com/%s/status/%s  
- Thread: https://twitter.com/user/status/%s

Time: %s`,
		severityEmoji, strings.ToUpper(alert.AlertSeverity),
		alert.FUDUsername, alert.FUDUserID,
		nf.formatFUDType(alert.FUDType), alert.FUDProbability*100,
		alert.RecommendedAction,
		nf.truncateText(alert.MessagePreview, 100),
		alert.FUDUsername, alert.FUDMessageID,
		alert.ThreadID,
		nf.formatTime(alert.DetectedAt))

	return message
}

func (nf *NotificationFormatter) getSeverityEmoji(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "🚨🔥"
	case "high":
		return "🚨"
	case "medium":
		return "⚠️"
	case "low":
		return "ℹ️"
	default:
		return "❓"
	}
}

func (nf *NotificationFormatter) getFUDTypeEmoji(fudType string) string {
	switch {
	case strings.Contains(fudType, "trojan_horse"):
		return "🐴"
	case strings.Contains(fudType, "direct_attack"):
		return "⚔️"
	case strings.Contains(fudType, "statistical"):
		return "📊"
	case strings.Contains(fudType, "escalation"):
		return "📈"
	case strings.Contains(fudType, "dramatic_exit"):
		return "🎭"
	case strings.Contains(fudType, "casual"):
		return "💭"
	default:
		return "🎯"
	}
}

func (nf *NotificationFormatter) formatFUDType(fudType string) string {
	// Convert snake_case to Title Case
	words := strings.Split(fudType, "_")
	for i, word := range words {
		words[i] = strings.Title(word)
	}
	return strings.Join(words, " ")
}

func (nf *NotificationFormatter) truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength-3] + "..."
}

func (nf *NotificationFormatter) formatTime(timeStr string) string {
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t.Format("2006-01-02 15:04:05 UTC")
	}
	return timeStr
}

func (nf *NotificationFormatter) mapRiskLevelToSeverity(riskLevel string) string {
	switch strings.ToLower(riskLevel) {
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

func (nf *NotificationFormatter) getRecommendedAction(aiDecision SecondStepClaudeResponse) string {
	switch strings.ToLower(aiDecision.UserRiskLevel) {
	case "critical":
		if strings.Contains(strings.ToLower(aiDecision.FUDType), "professional") {
			return "IMMEDIATE_BAN"
		}
		return "URGENT_REVIEW"
	case "high":
		return "ESCALATE_TO_ADMIN"
	case "medium":
		return "MONITOR_CLOSELY"
	case "low":
		return "LOG_AND_WATCH"
	default:
		return "REVIEW_NEEDED"
	}
}
