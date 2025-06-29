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
}

func NewNotificationFormatter() *NotificationFormatter {
	return &NotificationFormatter{}
}

func (nf *NotificationFormatter) FormatForTelegram(alert FUDAlertNotification) string {
	severityEmoji := nf.getSeverityEmoji(alert.AlertSeverity)
	typeEmoji := nf.getFUDTypeEmoji(alert.FUDType)

	message := fmt.Sprintf(`%s <b>FUD ALERT - %s SEVERITY</b>

%s <b>Attack Type:</b> %s
🎯 <b>User:</b> @%s
📊 <b>Confidence:</b> %.0f%%
⚡ <b>Action:</b> %s

💬 <b>Message Preview:</b>
<i>%s</i>

🔗 <b>Links:</b>
• <a href="https://twitter.com/%s/status/%s">FUD Message</a>
• <a href="https://twitter.com/user/status/%s">Original Thread</a>

⏰ <b>Detected:</b> %s
🆔 <b>IDs:</b> User: %s | Tweet: %s`,
		severityEmoji, strings.ToUpper(alert.AlertSeverity),
		typeEmoji, nf.formatFUDType(alert.FUDType),
		alert.FUDUsername,
		alert.FUDProbability*100,
		alert.RecommendedAction,
		nf.truncateText(alert.MessagePreview, 150),
		alert.FUDUsername, alert.FUDMessageID,
		alert.ThreadID,
		nf.formatTime(alert.DetectedAt),
		alert.FUDUserID, alert.FUDMessageID)

	return message
}

func (nf *NotificationFormatter) FormatForTelegramWithDetail(alert FUDAlertNotification, notificationID string) string {
	severityEmoji := nf.getSeverityEmoji(alert.AlertSeverity)
	typeEmoji := nf.getFUDTypeEmoji(alert.FUDType)

	message := fmt.Sprintf(`%s <b>FUD ALERT - %s SEVERITY</b>

%s <b>Attack Type:</b> %s
🎯 <b>User:</b> @%s
📊 <b>Confidence:</b> %.0f%%
⚡ <b>Action:</b> %s

💬 <b>Message Preview:</b>
<i>%s</i>

🔗 <b>Quick Links:</b>
• <a href="https://twitter.com/%s/status/%s">FUD Message</a>
• <a href="https://twitter.com/user/status/%s">Original Thread</a>

⏰ <b>Detected:</b> %s

📋 <b>For detailed analysis, use:</b> /detail_%s`,
		severityEmoji, strings.ToUpper(alert.AlertSeverity),
		typeEmoji, nf.formatFUDType(alert.FUDType),
		alert.FUDUsername,
		alert.FUDProbability*100,
		alert.RecommendedAction,
		nf.truncateText(alert.MessagePreview, 120),
		alert.FUDUsername, alert.FUDMessageID,
		alert.ThreadID,
		nf.formatTime(alert.DetectedAt),
		notificationID)

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

	message := fmt.Sprintf(`%s <b>DETAILED FUD ANALYSIS</b>

🏷️ <b>CLASSIFICATION</b>
%s Type: %s
🎯 Target User: @%s (ID: %s)
📊 Confidence Level: %.1f%%
🚨 Risk Level: %s
⚡ Recommended Action: %s

📝 <b>FULL MESSAGE TEXT</b>
<i>%s</i>

🔍 <b>KEY EVIDENCE</b>
%s

🧠 <b>AI DECISION REASONING</b>
<i>%s</i>

🔗 <b>INVESTIGATION LINKS</b>
• <a href="https://twitter.com/%s/status/%s">View FUD Message</a>
• <a href="https://twitter.com/user/status/%s">View Original Thread</a>
• <a href="https://twitter.com/%s">User Profile</a>

📅 <b>DETECTION METADATA</b>
Detected At: %s
FUD Message ID: %s
Thread ID: %s
User ID: %s`,
		severityEmoji,
		typeEmoji, nf.formatFUDType(alert.FUDType),
		alert.FUDUsername, alert.FUDUserID,
		alert.FUDProbability*100,
		strings.ToUpper(alert.AlertSeverity),
		alert.RecommendedAction,
		alert.MessagePreview,
		evidenceList,
		alert.DecisionReason,
		alert.FUDUsername, alert.FUDMessageID,
		alert.ThreadID,
		alert.FUDUsername,
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
