package main

import (
	"context"
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/grutapig/hackaton/twitterapi_reverse"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestNewTwitterBotService(t *testing.T) {
	godotenv.Load()
	twitterAPIService := twitterapi.NewTwitterAPIService(os.Getenv(ENV_TWITTER_API_KEY), os.Getenv(ENV_TWITTER_API_BASE_URL), os.Getenv(ENV_PROXY_DSN))
	databaseService, err := NewDatabaseService(os.Getenv(ENV_DATABASE_NAME))
	assert.NoError(t, err)
	auth := twitterapi_reverse.NewTwitterAuth(os.Getenv(ENV_TWITTER_REVERSE_AUTHORIZATION), os.Getenv(ENV_TWITTER_REVERSE_CSRF_TOKEN), os.Getenv(ENV_TWITTER_REVERSE_COOKIE))
	twitterReverseService := twitterapi_reverse.NewTwitterReverseApi(auth, os.Getenv(twitterapi.ENV_PROXY_DSN), false)

	claude, err := NewClaudeClient(os.Getenv(ENV_CLAUDE_API_KEY), os.Getenv(ENV_PROXY_CLAUDE_DSN), CLAUDE_MODEL)
	twitterBotService := NewTwitterBotService(twitterAPIService, twitterReverseService, databaseService, claude)
	twitterBotService.StartMonitoring(context.Background())
}
