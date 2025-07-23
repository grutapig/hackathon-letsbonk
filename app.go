package main

import (
	"context"
	"github.com/grutapig/hackaton/claude"
	"log"
	"os"
	"sync"

	"github.com/grutapig/hackaton/twitterapi"
)

type Application struct {
	config                 *Config
	channels               *Channels
	claudeAPI              *claude.ClaudeApi
	twitterAPI             *twitterapi.TwitterAPIService
	databaseService        *DatabaseService
	loggingService         *LoggingService
	telegramService        *TelegramService
	twitterBotService      *TwitterBotService
	cleanupScheduler       *CleanupScheduler
	systemPromptFirstStep  []byte
	systemPromptSecondStep []byte
}

func NewApplication(
	config *Config,
	channels *Channels,
	claudeAPI *claude.ClaudeApi,
	twitterAPI *twitterapi.TwitterAPIService,
	databaseService *DatabaseService,
	loggingService *LoggingService,
	telegramService *TelegramService,
	twitterBotService *TwitterBotService,
	cleanupScheduler *CleanupScheduler,
) (*Application, error) {

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

func (app *Application) Initialize() error {
	log.Println("Database service initialized successfully")
	log.Println("Twitter bot service initialized successfully")
	log.Println("Logging service initialized successfully")

	app.cleanupScheduler.Start()

	if app.config.ClearAnalysisOnStart {
		log.Println("Clearing all analysis flags on startup...")
		err := app.databaseService.ClearAllAnalysisFlags()
		if err != nil {
			log.Printf("Warning: Failed to clear analysis flags: %v", err)
		} else {
			log.Println("Successfully cleared all analysis flags")
		}
	}

	log.Println("Initializing data...")
	initializeData(app.databaseService, app.twitterAPI)
	go app.twitterBotService.StartMonitoring(context.Background())
	app.telegramService.StartListening()

	return nil
}

func (app *Application) Run() error {
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		MonitoringHandler(app.twitterAPI, app.channels.NewMessageCh, app.databaseService, app.loggingService)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(app.channels.FirstStepCh)
		defer close(app.channels.TwitterBotCh)
		for message := range app.channels.NewMessageCh {

			select {
			case app.channels.FirstStepCh <- message:
			default:
				log.Printf("First step channel full, skipping message %s", message.TweetID)
			}

			select {
			case app.channels.TwitterBotCh <- message:
			default:
				log.Printf("Twitter bot channel full, skipping message %s", message.TweetID)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		FirstStepHandler(app.channels.FirstStepCh, app.channels.FudCh, app.claudeAPI, app.systemPromptFirstStep, app.databaseService, app.loggingService, app.channels.NotificationCh)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for newMessage := range app.channels.FudCh {
			log.Printf("Second step processing for user %s", newMessage.Author.UserName)
			SecondStepHandler(newMessage, app.channels.NotificationCh, app.twitterAPI, app.claudeAPI, app.systemPromptSecondStep, app.config.Ticker, app.databaseService, app.loggingService)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		NotificationHandler(app.channels.NotificationCh, app.telegramService)
	}()

	wg.Wait()
	return nil
}

func (app *Application) Shutdown() {
	log.Println("Shutting down application...")

	app.cleanupScheduler.Stop()

	app.databaseService.Close()
	app.loggingService.Close()

	log.Println("Application shutdown completed")
}
