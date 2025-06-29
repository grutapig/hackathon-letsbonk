package main

import (
	"log"
)

// NotificationHandler handles FUD alert notifications
func NotificationHandler(notificationCh chan FUDAlertNotification, telegramService *TelegramService) {
	for alert := range notificationCh {
		log.Printf("FUD Alert: %s (@%s) - %s", alert.FUDType, alert.FUDUsername, alert.AlertSeverity)

		// Store and broadcast notification with detail command
		err := telegramService.StoreAndBroadcastNotification(alert)
		if err != nil {
			log.Printf("Failed to send Telegram notification: %v", err)
		}
	}
}
