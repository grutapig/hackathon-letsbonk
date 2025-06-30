package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"
)

const USER_STATUS_FILE = "user_status.json"

type UserStatus string

const (
	STATUS_UNKNOWN       UserStatus = "unknown"
	STATUS_CLEAN         UserStatus = "clean"
	STATUS_FUD_CONFIRMED UserStatus = "fud_confirmed"
	STATUS_ANALYZING     UserStatus = "analyzing"
)

type UserInfo struct {
	UserID          string     `json:"user_id"`
	Username        string     `json:"username"`
	Status          UserStatus `json:"status"`
	FUDType         string     `json:"fud_type,omitempty"`
	FUDProbability  float64    `json:"fud_probability,omitempty"`
	LastAnalyzedAt  string     `json:"last_analyzed_at"`
	LastMessageID   string     `json:"last_message_id"`
	AnalysisCount   int        `json:"analysis_count"`
	FUDMessageCount int        `json:"fud_message_count"`
}

type UserStatusManager struct {
	users     map[string]*UserInfo
	mutex     sync.RWMutex
	isRunning bool
}

func NewUserStatusManager() *UserStatusManager {
	manager := &UserStatusManager{
		users:     make(map[string]*UserInfo),
		isRunning: false,
	}

	// Load existing data from file
	manager.loadFromFile()

	return manager
}

func (usm *UserStatusManager) StartPeriodicSave() {
	if usm.isRunning {
		return
	}
	usm.isRunning = true

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for usm.isRunning {
			select {
			case <-ticker.C:
				err := usm.saveToFile()
				if err != nil {
					log.Printf("Error saving user status: %v", err)
				}
			}
		}
	}()

	log.Println("User status manager started periodic save")
}

func (usm *UserStatusManager) StopPeriodicSave() {
	usm.isRunning = false
	// Final save before stopping
	usm.saveToFile()
	log.Println("User status manager stopped periodic save")
}

func (usm *UserStatusManager) GetUserStatus(userID string) UserStatus {
	usm.mutex.RLock()
	defer usm.mutex.RUnlock()

	if user, exists := usm.users[userID]; exists {
		return user.Status
	}
	return STATUS_UNKNOWN
}

func (usm *UserStatusManager) GetUserInfo(userID string) *UserInfo {
	usm.mutex.RLock()
	defer usm.mutex.RUnlock()

	if user, exists := usm.users[userID]; exists {
		// Return a copy to avoid race conditions
		userCopy := *user
		return &userCopy
	}
	return nil
}

func (usm *UserStatusManager) SetUserAnalyzing(userID, username string) {
	usm.mutex.Lock()
	defer usm.mutex.Unlock()

	if user, exists := usm.users[userID]; exists {
		user.Status = STATUS_ANALYZING
		user.LastAnalyzedAt = time.Now().Format(time.RFC3339)
		user.AnalysisCount++
	} else {
		usm.users[userID] = &UserInfo{
			UserID:         userID,
			Username:       username,
			Status:         STATUS_ANALYZING,
			LastAnalyzedAt: time.Now().Format(time.RFC3339),
			AnalysisCount:  1,
		}
	}
}

func (usm *UserStatusManager) UpdateUserAfterAnalysis(userID, username string, aiDecision SecondStepClaudeResponse, messageID string) {
	usm.mutex.Lock()
	defer usm.mutex.Unlock()

	user, exists := usm.users[userID]
	if !exists {
		user = &UserInfo{
			UserID:   userID,
			Username: username,
		}
		usm.users[userID] = user
	}

	user.Username = username // Update username in case it changed
	user.LastAnalyzedAt = time.Now().Format(time.RFC3339)
	user.LastMessageID = messageID
	user.AnalysisCount++

	if aiDecision.IsFUDUser || aiDecision.IsFUDAttack {
		user.Status = STATUS_FUD_CONFIRMED
		user.FUDType = aiDecision.FUDType
		user.FUDProbability = aiDecision.FUDProbability
		user.FUDMessageCount++
	} else {
		user.Status = STATUS_CLEAN
	}
}

func (usm *UserStatusManager) MarkUserAsFUD(userID, username, messageID string, fudType string, probability float64) {
	usm.mutex.Lock()
	defer usm.mutex.Unlock()

	user, exists := usm.users[userID]
	if !exists {
		user = &UserInfo{
			UserID:   userID,
			Username: username,
		}
		usm.users[userID] = user
	}

	user.Status = STATUS_FUD_CONFIRMED
	user.FUDType = fudType
	user.FUDProbability = probability
	user.LastMessageID = messageID
	user.LastAnalyzedAt = time.Now().Format(time.RFC3339)
	user.FUDMessageCount++
}

func (usm *UserStatusManager) IsFUDUser(userID string) bool {
	return usm.GetUserStatus(userID) == STATUS_FUD_CONFIRMED
}

func (usm *UserStatusManager) IsUserBeingAnalyzed(userID string) bool {
	return usm.GetUserStatus(userID) == STATUS_ANALYZING
}

func (usm *UserStatusManager) GetFUDFriendsAnalysis(usernames []string) (int, int, []string) {
	usm.mutex.RLock()
	defer usm.mutex.RUnlock()

	totalFriends := len(usernames)
	fudFriends := 0
	fudFriendsList := make([]string, 0)

	for _, username := range usernames {
		// Find user by username
		for _, user := range usm.users {
			if user.Username == username && user.Status == STATUS_FUD_CONFIRMED {
				fudFriends++
				fudFriendsList = append(fudFriendsList, fmt.Sprintf("%s (%s, %.1f%%)",
					username, user.FUDType, user.FUDProbability*100))
				break
			}
		}
	}

	return totalFriends, fudFriends, fudFriendsList
}

func (usm *UserStatusManager) GetStats() map[string]int {
	usm.mutex.RLock()
	defer usm.mutex.RUnlock()

	stats := map[string]int{
		"total_users":   len(usm.users),
		"fud_confirmed": 0,
		"clean_users":   0,
		"analyzing":     0,
		"unknown":       0,
	}

	for _, user := range usm.users {
		switch user.Status {
		case STATUS_FUD_CONFIRMED:
			stats["fud_confirmed"]++
		case STATUS_CLEAN:
			stats["clean_users"]++
		case STATUS_ANALYZING:
			stats["analyzing"]++
		case STATUS_UNKNOWN:
			stats["unknown"]++
		}
	}

	return stats
}

func (usm *UserStatusManager) loadFromFile() error {
	if _, err := os.Stat(USER_STATUS_FILE); os.IsNotExist(err) {
		// File doesn't exist, start with empty map
		log.Println("User status file doesn't exist, starting with empty data")
		return nil
	}

	data, err := ioutil.ReadFile(USER_STATUS_FILE)
	if err != nil {
		log.Printf("Error reading user status file: %v", err)
		return err
	}

	var users map[string]*UserInfo
	err = json.Unmarshal(data, &users)
	if err != nil {
		log.Printf("Error unmarshaling user status: %v", err)
		return err
	}

	usm.mutex.Lock()
	usm.users = users
	usm.mutex.Unlock()

	log.Printf("Loaded %d users from status file", len(users))
	return nil
}

func (usm *UserStatusManager) saveToFile() error {
	usm.mutex.RLock()
	data, err := json.MarshalIndent(usm.users, "", "  ")
	usm.mutex.RUnlock()

	if err != nil {
		return err
	}

	return ioutil.WriteFile(USER_STATUS_FILE, data, 0644)
}
