package main

import (
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestNewTelegramService(t *testing.T) {
	t.Skip()
	godotenv.Load()
	telegramService, err := NewTelegramService(os.Getenv(ENV_TELEGRAM_API_KEY), os.Getenv(ENV_PROXY_DSN), os.Getenv(ENV_TELEGRAM_ADMIN_CHAT_ID), nil, nil, nil)
	assert.NoError(t, err)
	telegramService.StartListening()
	err = telegramService.BroadcastMessage("hello")
	assert.NoError(t, err)
	ch := make(chan bool)
	<-ch
}
