package main

import (
	"log"
)

func NotificationHandler(notificationCh chan FUDAlertNotification, telegramService *TelegramService) {
	for alert := range notificationCh {
		log.Printf("FUD Alert: %s (@%s) - %s", alert.FUDType, alert.FUDUsername, alert.AlertSeverity)

		if alert.TargetChatID != 0 {

			telegramMessage := telegramService.formatter.FormatForTelegramWithDetail(alert, "")
			err := telegramService.SendMessage(alert.TargetChatID, telegramMessage)
			if err != nil {
				log.Printf("Failed to send targeted Telegram notification to chat %d: %v", alert.TargetChatID, err)
			} else {
				log.Printf("Sent targeted notification for @%s to chat %d", alert.FUDUsername, alert.TargetChatID)
			}
		} else {

			err := telegramService.StoreAndBroadcastNotification(alert)
			if err != nil {
				log.Printf("Failed to send Telegram notification: %v", err)
			}
		}
	}
}
