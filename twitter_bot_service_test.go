package main

import (
	"context"
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/joho/godotenv"
	"os"
	"testing"
)

func TestNewTwitterBotService(t *testing.T) {
	godotenv.Load()
	twitApi := twitterapi.NewTwitterAPIService(os.Getenv(ENV_TWITTER_API_KEY), os.Getenv(ENV_TWITTER_API_BASE_URL), os.Getenv(ENV_PROXY_DSN))
	twitterBotService := NewTwitterBotService(twitApi)
	twitterBotService.StartMonitoring(context.Background())
}
