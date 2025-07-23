package main

type Config struct {
	TelegramAPIKey     string
	TelegramChatIds    []int64
	TwitterAPIKey      string
	TwitterBotTag      string
	ClaudeAPIKey       string
	ProxyDSN           string
	TwitterAuth        string
	TwitterCSRFToken   string
	TwitterCookie      string
	TwitterReverseAuth string
	UserSearchPages    int
}
