package main

import (
	"log"
	"os"
	"sync"

	"github.com/grutapig/hackaton/twitterapi"
)

// Application holds all services and manages the application lifecycle
type Application struct {
	config                 *Config
	channels               *Channels
	claudeAPI              *ClaudeApi
	twitterAPI             *twitterapi.TwitterAPIService
	databaseService        *DatabaseService
	loggingService         *LoggingService
	telegramService        *TelegramService
	twitterBotService      *TwitterBotService
	cleanupScheduler       *CleanupScheduler
	systemPromptFirstStep  []byte
	systemPromptSecondStep []byte
}

// NewApplication creates a new application instance
func NewApplication(
	config *Config,
	channels *Channels,
	claudeAPI *ClaudeApi,
	twitterAPI *twitterapi.TwitterAPIService,
	databaseService *DatabaseService,
	loggingService *LoggingService,
	telegramService *TelegramService,
	twitterBotService *TwitterBotService,
	cleanupScheduler *CleanupScheduler,
) (*Application, error) {
	// Load system prompts
	systemPromptFirstStep, err := os.ReadFile(PROMPT_FILE_STEP1)
	if err != nil {
		return nil, err
	}

	systemPromptSecondStep, err := os.ReadFile(PROMPT_FILE_STEP2)
	if err != nil {
		return nil, err
	}

	return &Application{
		config:                 config,
		channels:               channels,
		claudeAPI:              claudeAPI,
		twitterAPI:             twitterAPI,
		databaseService:        databaseService,
		loggingService:         loggingService,
		telegramService:        telegramService,
		twitterBotService:      twitterBotService,
		cleanupScheduler:       cleanupScheduler,
		systemPromptFirstStep:  systemPromptFirstStep,
		systemPromptSecondStep: systemPromptSecondStep,
	}, nil
}

// Initialize performs application initialization
func (app *Application) Initialize() error {
	log.Println("Database service initialized successfully")
	log.Println("Twitter bot service initialized successfully")
	log.Println("Logging service initialized successfully")

	// Start cleanup scheduler
	app.cleanupScheduler.Start()

	// Check if we need to clear analysis flags on startup
	if app.config.ClearAnalysisOnStart {
		log.Println("Clearing all analysis flags on startup...")
		err := app.databaseService.ClearAllAnalysisFlags()
		if err != nil {
			log.Printf("Warning: Failed to clear analysis flags: %v", err)
		} else {
			log.Println("Successfully cleared all analysis flags")
		}
	}

	// User status management is now handled by database - no periodic saves needed

	// Initialize data (CSV import or community loading)
	log.Println("Initializing data...")
	initializeData(app.databaseService, app.twitterAPI)

	// Start Telegram service
	app.telegramService.StartListening()

	return nil
}

// Run starts the application
func (app *Application) Run() error {
	wg := sync.WaitGroup{}

	// Start monitoring for new messages in community
	wg.Add(1)
	go func() {
		defer wg.Done()
		MonitoringHandler(app.twitterAPI, app.channels.NewMessageCh, app.databaseService, app.loggingService)
	}()

	// Broadcast messages to multiple channels
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(app.channels.FirstStepCh)
		defer close(app.channels.TwitterBotCh)
		for message := range app.channels.NewMessageCh {
			// Send to first step handler
			select {
			case app.channels.FirstStepCh <- message:
			default:
				log.Printf("First step channel full, skipping message %s", message.TweetID)
			}
			// Send to twitter bot handler
			select {
			case app.channels.TwitterBotCh <- message:
			default:
				log.Printf("Twitter bot channel full, skipping message %s", message.TweetID)
			}
		}
	}()

	// Handle new message first step
	wg.Add(1)
	go func() {
		defer wg.Done()
		FirstStepHandler(app.channels.FirstStepCh, app.channels.FudCh, app.claudeAPI, app.systemPromptFirstStep, app.databaseService, app.loggingService, app.channels.NotificationCh)
	}()

	// Handle fud messages with dynamic routing
	wg.Add(1)
	go func() {
		defer wg.Done()
		for newMessage := range app.channels.FudCh {
			log.Printf("Second step processing for user %s", newMessage.Author.UserName)
			SecondStepHandler(newMessage, app.channels.NotificationCh, app.twitterAPI, app.claudeAPI, app.systemPromptSecondStep, app.config.Ticker, app.databaseService, app.loggingService)
		}
	}()

	// Notification handler
	wg.Add(1)
	go func() {
		defer wg.Done()
		NotificationHandler(app.channels.NotificationCh, app.telegramService)
	}()

	// Twitter bot mention listener
	wg.Add(1)
	go func() {
		defer wg.Done()
		app.twitterBotService.StartMentionListener(app.channels.TwitterBotCh)
	}()

	wg.Wait()
	return nil
}

// Shutdown performs graceful shutdown
func (app *Application) Shutdown() {
	log.Println("Shutting down application...")

	// Stop cleanup scheduler
	app.cleanupScheduler.Stop()

	// Close database connections
	app.databaseService.Close()
	app.loggingService.Close()

	log.Println("Application shutdown completed")
}
