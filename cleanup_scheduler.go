package main

import (
	"log"
	"time"
)

type CleanupScheduler struct {
	loggingService *LoggingService
	ticker         *time.Ticker
	stopChan       chan bool
}

func NewCleanupScheduler(loggingService *LoggingService) *CleanupScheduler {
	return &CleanupScheduler{
		loggingService: loggingService,
		stopChan:       make(chan bool),
	}
}

func (cs *CleanupScheduler) Start() {
	log.Printf("🧹 Starting cleanup scheduler - will run daily at midnight")

	now := time.Now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	durationUntilMidnight := nextMidnight.Sub(now)

	firstRunTimer := time.NewTimer(durationUntilMidnight)

	go func() {

		select {
		case <-firstRunTimer.C:
			log.Printf("🧹 Running first cleanup at midnight")
			cs.runCleanup()
		case <-cs.stopChan:
			firstRunTimer.Stop()
			return
		}

		cs.ticker = time.NewTicker(24 * time.Hour)
		defer cs.ticker.Stop()

		for {
			select {
			case <-cs.ticker.C:
				log.Printf("🧹 Running daily cleanup")
				cs.runCleanup()
			case <-cs.stopChan:
				log.Printf("🧹 Cleanup scheduler stopped")
				return
			}
		}
	}()
}

func (cs *CleanupScheduler) Stop() {
	log.Printf("🧹 Stopping cleanup scheduler")
	close(cs.stopChan)
	if cs.ticker != nil {
		cs.ticker.Stop()
	}
}

func (cs *CleanupScheduler) runCleanup() {
	log.Printf("🧹 Starting scheduled cleanup of old log records")

	err := cs.loggingService.CleanupOldLogs(30)
	if err != nil {
		log.Printf("❌ Error during cleanup: %v", err)
		return
	}

	err = cs.loggingService.VacuumDatabase()
	if err != nil {
		log.Printf("❌ Error during VACUUM: %v", err)
		return
	}

	stats, err := cs.loggingService.GetDatabaseStats()
	if err != nil {
		log.Printf("❌ Error getting database stats: %v", err)
		return
	}

	log.Printf("✅ Cleanup completed successfully")
	log.Printf("📊 Database stats after cleanup: %+v", stats)
}

func (cs *CleanupScheduler) RunCleanupNow() {
	log.Printf("🧹 Running manual cleanup")
	cs.runCleanup()
}
