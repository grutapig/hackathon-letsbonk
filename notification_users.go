package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

type NotificationUsersManager struct {
	users    map[string]bool
	filePath string
	mutex    sync.RWMutex
}

func NewNotificationUsersManager(filePath string) *NotificationUsersManager {
	return &NotificationUsersManager{
		users:    make(map[string]bool),
		filePath: filePath,
	}
}

// LoadUsers loads users from environment variable and file
func (n *NotificationUsersManager) LoadUsers(envUsers string) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	// Clear existing users
	n.users = make(map[string]bool)

	// Load from environment variable first
	if envUsers != "" {
		envUsersList := strings.Split(envUsers, ",")
		for _, user := range envUsersList {
			user = strings.TrimSpace(user)
			if user != "" {
				n.users[user] = true
			}
		}
	}

	// Load from file if exists
	if _, err := os.Stat(n.filePath); err == nil {
		file, err := os.Open(n.filePath)
		if err != nil {
			return fmt.Errorf("failed to open notification users file: %w", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			user := strings.TrimSpace(scanner.Text())
			if user != "" && !strings.HasPrefix(user, "#") { // Skip comments
				n.users[user] = true
			}
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read notification users file: %w", err)
		}
	}

	return nil
}

// AddUser adds a new user to the notification list and saves to file
func (n *NotificationUsersManager) AddUser(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	n.mutex.Lock()
	defer n.mutex.Unlock()

	// Check if user already exists
	if n.users[username] {
		return nil // User already exists, no need to add
	}

	// Add user to memory
	n.users[username] = true

	// Append to file
	file, err := os.OpenFile(n.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open notification users file for writing: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(username + "\n")
	if err != nil {
		return fmt.Errorf("failed to write user to file: %w", err)
	}

	return nil
}

// HasUser checks if a user is in the notification list
func (n *NotificationUsersManager) HasUser(username string) bool {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	return n.users[username]
}

// GetAllUsers returns all users in the notification list
func (n *NotificationUsersManager) GetAllUsers() []string {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	users := make([]string, 0, len(n.users))
	for user := range n.users {
		users = append(users, user)
	}
	return users
}

// GetUserCount returns the number of users in the notification list
func (n *NotificationUsersManager) GetUserCount() int {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	return len(n.users)
}

// SaveAllUsers saves all current users to file (overwrites existing file)
func (n *NotificationUsersManager) SaveAllUsers() error {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	file, err := os.Create(n.filePath)
	if err != nil {
		return fmt.Errorf("failed to create notification users file: %w", err)
	}
	defer file.Close()

	// Write header comment
	_, err = file.WriteString("# Notification Users List\n")
	if err != nil {
		return fmt.Errorf("failed to write header to file: %w", err)
	}

	// Write all users
	for user := range n.users {
		_, err = file.WriteString(user + "\n")
		if err != nil {
			return fmt.Errorf("failed to write user to file: %w", err)
		}
	}

	return nil
}
