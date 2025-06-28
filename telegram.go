package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TelegramService struct {
	apiKey        string
	client        *http.Client
	chatIDs       map[int64]bool
	chatMutex     sync.RWMutex
	lastOffset    int64
	isRunning     bool
	notifications map[string]FUDAlertNotification
	notifMutex    sync.RWMutex
	formatter     *NotificationFormatter
}

type TelegramUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  struct {
		MessageID int64 `json:"message_id"`
		From      struct {
			ID        int64  `json:"id"`
			IsBot     bool   `json:"is_bot"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name,omitempty"`
			Username  string `json:"username,omitempty"`
		} `json:"from"`
		Chat struct {
			ID    int64  `json:"id"`
			Type  string `json:"type"`
			Title string `json:"title,omitempty"`
		} `json:"chat"`
		Date int64  `json:"date"`
		Text string `json:"text"`
	} `json:"message"`
}

type TelegramResponse struct {
	OK     bool             `json:"ok"`
	Result []TelegramUpdate `json:"result"`
	Error  *TelegramError   `json:"error,omitempty"`
}

type TelegramError struct {
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

type TelegramSendMessageRequest struct {
	ChatID         int64  `json:"chat_id"`
	Text           string `json:"text"`
	ParseMode      string `json:"parse_mode,omitempty"`
	DisablePreview bool   `json:"disable_web_page_preview,omitempty"`
}

func NewTelegramService(apiKey string, proxyDSN string, initialChatIDs string, formatter *NotificationFormatter) (*TelegramService, error) {
	transport := &http.Transport{}
	if proxyDSN != "" {
		proxyURL, err := url.Parse(proxyDSN)
		if err != nil {
			return nil, fmt.Errorf("telegram service proxy dsn error: %s", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	service := &TelegramService{
		apiKey:        apiKey,
		client:        client,
		chatIDs:       make(map[int64]bool),
		lastOffset:    0,
		isRunning:     false,
		notifications: make(map[string]FUDAlertNotification),
		formatter:     formatter,
	}

	// Add initial chat IDs if provided (comma-separated)
	if initialChatIDs != "" {
		chatIDStrings := strings.Split(initialChatIDs, ",")
		for _, chatIDStr := range chatIDStrings {
			chatIDStr = strings.TrimSpace(chatIDStr) // Remove spaces
			if chatIDStr != "" {
				if chatID, err := strconv.ParseInt(chatIDStr, 10, 64); err == nil {
					service.chatIDs[chatID] = true
					log.Printf("Added initial Telegram chat ID: %d", chatID)
				} else {
					log.Printf("Warning: Invalid chat ID format: %s", chatIDStr)
				}
			}
		}
	}

	return service, nil
}

func (t *TelegramService) StartListening() {
	if t.isRunning {
		return
	}
	t.isRunning = true

	go func() {
		for t.isRunning {
			err := t.processUpdates()
			if err != nil {
				log.Printf("Error processing Telegram updates: %v", err)
			}
			time.Sleep(2 * time.Second)
		}
	}()

	log.Println("Telegram service started listening for updates")
}

func (t *TelegramService) StopListening() {
	t.isRunning = false
	log.Println("Telegram service stopped listening")
}

func (t *TelegramService) processUpdates() error {
	updates, err := t.getUpdates()
	if err != nil {
		return err
	}

	for _, update := range updates {
		t.lastOffset = update.UpdateID + 1

		// Add new chat ID if not exists
		chatID := update.Message.Chat.ID
		t.chatMutex.Lock()
		if !t.chatIDs[chatID] {
			t.chatIDs[chatID] = true
			log.Printf("New Telegram chat registered: %d (from: %s)", chatID, update.Message.From.FirstName)

			// Send chat info as response
			info := fmt.Sprintf("✅ Chat registered!\nChat ID: %d\nUser: %s %s\nUsername: @%s",
				chatID,
				update.Message.From.FirstName,
				update.Message.From.LastName,
				update.Message.From.Username)

			go t.SendMessage(chatID, info)
		}
		t.chatMutex.Unlock()

		// Handle commands and messages
		if update.Message.Text != "" {
			if strings.HasPrefix(update.Message.Text, "/detail_") {
				t.handleDetailCommand(chatID, update.Message.Text)
			} else {
				response := fmt.Sprintf("📋 Your Chat Info:\nChat ID: %d\nMessage: %s\nRegistered chats: %d",
					chatID,
					update.Message.Text,
					len(t.chatIDs))

				go t.SendMessage(chatID, response)
			}
		}
	}

	return nil
}

func (t *TelegramService) getUpdates() ([]TelegramUpdate, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=1", t.apiKey, t.lastOffset)

	resp, err := t.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var telegramResp TelegramResponse
	err = json.Unmarshal(body, &telegramResp)
	if err != nil {
		return nil, err
	}

	if !telegramResp.OK {
		return nil, fmt.Errorf("telegram API error: %v", telegramResp.Error)
	}

	return telegramResp.Result, nil
}

func (t *TelegramService) SendMessage(chatID int64, text string) error {
	reqBody := TelegramSendMessageRequest{
		ChatID:         chatID,
		Text:           text,
		ParseMode:      "HTML",
		DisablePreview: true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.apiKey)
	resp, err := t.client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram send message failed: %s", string(body))
	}

	return nil
}

func (t *TelegramService) BroadcastMessage(text string) error {
	t.chatMutex.RLock()
	defer t.chatMutex.RUnlock()

	if len(t.chatIDs) == 0 {
		log.Println("No registered Telegram chats to broadcast to")
		return nil
	}

	var errors []error
	for chatID := range t.chatIDs {
		err := t.SendMessage(chatID, text)
		if err != nil {
			log.Printf("Failed to send message to chat %d: %v", chatID, err)
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send to %d chats", len(errors))
	}

	log.Printf("Successfully broadcasted message to %d chats", len(t.chatIDs))
	return nil
}

func (t *TelegramService) GetRegisteredChats() []int64 {
	t.chatMutex.RLock()
	defer t.chatMutex.RUnlock()

	var chats []int64
	for chatID := range t.chatIDs {
		chats = append(chats, chatID)
	}
	return chats
}

func (t *TelegramService) generateNotificationID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (t *TelegramService) StoreAndBroadcastNotification(alert FUDAlertNotification) error {
	// Generate unique ID and store notification
	notificationID := t.generateNotificationID()

	t.notifMutex.Lock()
	t.notifications[notificationID] = alert
	t.notifMutex.Unlock()

	// Format message with detail command
	telegramMessage := t.formatter.FormatForTelegramWithDetail(alert, notificationID)

	// Broadcast to all chats
	return t.BroadcastMessage(telegramMessage)
}

func (t *TelegramService) handleDetailCommand(chatID int64, command string) {
	// Extract notification ID from command "/detail_12345abc"
	parts := strings.Split(command, "_")
	if len(parts) != 2 {
		t.SendMessage(chatID, "❌ Invalid command format. Use /detail_<id>")
		return
	}

	notificationID := parts[1]

	t.notifMutex.RLock()
	alert, exists := t.notifications[notificationID]
	t.notifMutex.RUnlock()

	if !exists {
		t.SendMessage(chatID, "❌ Notification not found or expired.")
		return
	}

	// Send detailed information
	detailMessage := t.formatter.FormatDetailedView(alert)
	t.SendMessage(chatID, detailMessage)
}
