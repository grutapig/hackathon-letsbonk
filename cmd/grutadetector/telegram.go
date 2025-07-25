package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func sendTelegramMessage(message string) {
	for _, chatId := range telegramChatIds {
		msg := tgbotapi.NewMessage(chatId, message)
		msg.ParseMode = tgbotapi.ModeMarkdown

		_, err := bot.Send(msg)
		if err != nil {
			errorMsg := fmt.Sprintf("Error: %s... - %v", message[:min(5, len(message))], err)
			retryMsg := tgbotapi.NewMessage(chatId, errorMsg)
			bot.Send(retryMsg)
		}
	}
}
