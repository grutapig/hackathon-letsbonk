package main

import (
	"log"
	"time"
)

// CleanupScheduler handles periodic cleanup of old log records
type CleanupScheduler struct {
	loggingService *LoggingService
	ticker         *time.Ticker
	stopChan       chan bool
}

// NewCleanupScheduler creates a new cleanup scheduler
func NewCleanupScheduler(loggingService *LoggingService) *CleanupScheduler {
	return &CleanupScheduler{
		loggingService: loggingService,
		stopChan:       make(chan bool),
	}
}

// Start starts the cleanup scheduler to run daily
func (cs *CleanupScheduler) Start() {
	log.Printf("🧹 Starting cleanup scheduler - will run daily at midnight")

	// Calculate duration until next midnight
	now := time.Now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	durationUntilMidnight := nextMidnight.Sub(now)

	// Start a timer for the first run at midnight
	firstRunTimer := time.NewTimer(durationUntilMidnight)

	// Start the routine
	go func() {
		// Wait for first midnight
		select {
		case <-firstRunTimer.C:
			log.Printf("🧹 Running first cleanup at midnight")
			cs.runCleanup()
		case <-cs.stopChan:
			firstRunTimer.Stop()
			return
		}

		// After first run, start daily ticker
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

// Stop stops the cleanup scheduler
func (cs *CleanupScheduler) Stop() {
	log.Printf("🧹 Stopping cleanup scheduler")
	close(cs.stopChan)
	if cs.ticker != nil {
		cs.ticker.Stop()
	}
}

// runCleanup performs the actual cleanup operation
func (cs *CleanupScheduler) runCleanup() {
	log.Printf("🧹 Starting scheduled cleanup of old log records")

	// Clean up records older than 30 days
	err := cs.loggingService.CleanupOldLogs(30)
	if err != nil {
		log.Printf("❌ Error during cleanup: %v", err)
		return
	}

	// Run VACUUM to reclaim space
	err = cs.loggingService.VacuumDatabase()
	if err != nil {
		log.Printf("❌ Error during VACUUM: %v", err)
		return
	}

	// Log database statistics after cleanup
	stats, err := cs.loggingService.GetDatabaseStats()
	if err != nil {
		log.Printf("❌ Error getting database stats: %v", err)
		return
	}

	log.Printf("✅ Cleanup completed successfully")
	log.Printf("📊 Database stats after cleanup: %+v", stats)
}

// RunCleanupNow runs cleanup immediately (for testing or manual trigger)
func (cs *CleanupScheduler) RunCleanupNow() {
	log.Printf("🧹 Running manual cleanup")
	cs.runCleanup()
}
