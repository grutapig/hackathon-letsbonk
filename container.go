package main

import (
	"fmt"
	"github.com/grutapig/hackaton/twitterapi_reverse"
	"os"

	"github.com/grutapig/hackaton/twitterapi"
	"go.uber.org/dig"
)

type Config struct {
	ClaudeAPIKey         string
	ProxyClaudeDSN       string
	TwitterAPIKey        string
	TwitterAPIBaseURL    string
	ProxyDSN             string
	TelegramAPIKey       string
	TelegramAdminChatID  string
	DatabaseName         string
	LoggingDBPath        string
	TwitterBotTag        string
	TwitterAuth          string
	Ticker               string
	ClearAnalysisOnStart bool
}

type Channels struct {
	NewMessageCh   chan twitterapi.NewMessage
	FirstStepCh    chan twitterapi.NewMessage
	TwitterBotCh   chan twitterapi.NewMessage
	FudCh          chan twitterapi.NewMessage
	NotificationCh chan FUDAlertNotification
}

func ProvideConfig() (*Config, error) {
	ticker := os.Getenv(ENV_TWITTER_COMMUNITY_TICKER)
	if ticker == "" {
		return nil, fmt.Errorf("ticker should be set .env: %s", ENV_TWITTER_COMMUNITY_TICKER)
	}

	botTag := os.Getenv(ENV_TWITTER_BOT_TAG)
	if botTag == "" {
		return nil, fmt.Errorf("ENV_TWITTER_BOT_TAG environment variable is not set")
	}

	authSession := os.Getenv(ENV_TWITTER_AUTH)
	if authSession == "" {
		return nil, fmt.Errorf("ENV_TWITTER_AUTH environment variable is not set")
	}

	dbName := os.Getenv(ENV_DATABASE_NAME)
	if dbName == "" {
		dbName = "hackathon.db"
	}

	loggingDBPath := os.Getenv(ENV_LOGGING_DATABASE_PATH)
	if loggingDBPath == "" {
		loggingDBPath = "logs.db"
	}

	return &Config{
		ClaudeAPIKey:         os.Getenv(ENV_CLAUDE_API_KEY),
		ProxyClaudeDSN:       os.Getenv(ENV_PROXY_CLAUDE_DSN),
		TwitterAPIKey:        os.Getenv(ENV_TWITTER_API_KEY),
		TwitterAPIBaseURL:    os.Getenv(ENV_TWITTER_API_BASE_URL),
		ProxyDSN:             os.Getenv(ENV_PROXY_DSN),
		TelegramAPIKey:       os.Getenv(ENV_TELEGRAM_API_KEY),
		TelegramAdminChatID:  os.Getenv(ENV_TELEGRAM_ADMIN_CHAT_ID),
		DatabaseName:         dbName,
		LoggingDBPath:        loggingDBPath,
		TwitterBotTag:        botTag,
		TwitterAuth:          authSession,
		Ticker:               ticker,
		ClearAnalysisOnStart: os.Getenv(ENV_CLEAR_ANALYSIS_ON_START) == "true",
	}, nil
}

func ProvideChannels() *Channels {
	return &Channels{
		NewMessageCh:   make(chan twitterapi.NewMessage, 10),
		FirstStepCh:    make(chan twitterapi.NewMessage, 10),
		TwitterBotCh:   make(chan twitterapi.NewMessage, 10),
		FudCh:          make(chan twitterapi.NewMessage, 30),
		NotificationCh: make(chan FUDAlertNotification, 30),
	}
}

func ProvideClaudeAPI(config *Config) (*ClaudeApi, error) {
	return NewClaudeClient(config.ClaudeAPIKey, config.ProxyClaudeDSN, CLAUDE_MODEL)
}

func ProvideTwitterAPI(config *Config) *twitterapi.TwitterAPIService {
	return twitterapi.NewTwitterAPIService(config.TwitterAPIKey, config.TwitterAPIBaseURL, config.ProxyDSN)
}
func ProvideTwitterReverseAPI(config *Config) *twitterapi_reverse.TwitterReverseService {
	auth := twitterapi_reverse.NewTwitterAuth(os.Getenv(ENV_TWITTER_REVERSE_AUTHORIZATION), os.Getenv(ENV_TWITTER_REVERSE_CSRF_TOKEN), os.Getenv(ENV_TWITTER_REVERSE_COOKIE))

	return twitterapi_reverse.NewTwitterReverseApi(auth, config.ProxyDSN, false)
}

func ProvideDatabaseService(config *Config) (*DatabaseService, error) {
	return NewDatabaseService(config.DatabaseName)
}

func ProvideLoggingService(config *Config) (*LoggingService, error) {
	return NewLoggingService(config.LoggingDBPath)
}
func ProvideTwitterBotService(twitterapiService *twitterapi.TwitterAPIService, dbService *DatabaseService, claudeApi *ClaudeApi, twitterReverseService *twitterapi_reverse.TwitterReverseService) (*TwitterBotService, error) {
	return NewTwitterBotService(twitterapiService, twitterReverseService, dbService, claudeApi), nil
}

func ProvideNotificationFormatter() *NotificationFormatter {
	return NewNotificationFormatter()
}

func ProvideTelegramService(config *Config, formatter *NotificationFormatter, dbService *DatabaseService, channels *Channels) (*TelegramService, error) {
	return NewTelegramService(config.TelegramAPIKey, config.ProxyDSN, config.TelegramAdminChatID, formatter, dbService, channels.FudCh)
}

func ProvideCleanupScheduler(loggingService *LoggingService) *CleanupScheduler {
	return NewCleanupScheduler(loggingService)
}

func BuildContainer() (*dig.Container, error) {
	container := dig.New()

	if err := container.Provide(ProvideConfig); err != nil {
		return nil, fmt.Errorf("failed to provide config: %w", err)
	}

	if err := container.Provide(ProvideChannels); err != nil {
		return nil, fmt.Errorf("failed to provide channels: %w", err)
	}

	if err := container.Provide(ProvideClaudeAPI); err != nil {
		return nil, fmt.Errorf("failed to provide Claude API: %w", err)
	}

	if err := container.Provide(ProvideTwitterAPI); err != nil {
		return nil, fmt.Errorf("failed to provide Twitter API: %w", err)
	}
	if err := container.Provide(ProvideTwitterReverseAPI); err != nil {
		return nil, fmt.Errorf("failed to provide Twitter API: %w", err)
	}

	if err := container.Provide(ProvideDatabaseService); err != nil {
		return nil, fmt.Errorf("failed to provide database service: %w", err)
	}

	if err := container.Provide(ProvideLoggingService); err != nil {
		return nil, fmt.Errorf("failed to provide logging service: %w", err)
	}

	if err := container.Provide(ProvideTwitterBotService); err != nil {
		return nil, fmt.Errorf("failed to provide twitterbot service: %w", err)
	}

	if err := container.Provide(ProvideNotificationFormatter); err != nil {
		return nil, fmt.Errorf("failed to provide notification formatter: %w", err)
	}

	if err := container.Provide(ProvideTelegramService); err != nil {
		return nil, fmt.Errorf("failed to provide Telegram service: %w", err)
	}

	if err := container.Provide(ProvideCleanupScheduler); err != nil {
		return nil, fmt.Errorf("failed to provide cleanup scheduler: %w", err)
	}

	if err := container.Provide(NewApplication); err != nil {
		return nil, fmt.Errorf("failed to provide application: %w", err)
	}

	return container, nil
}
