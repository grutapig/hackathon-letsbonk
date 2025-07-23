package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type ClaudeApi struct {
	apiKey      string
	client      *http.Client
	model       string
	maxTokens   int
	temperature float32
}

const ROLE_USER = "user"
const ROLE_ASSISTANT = "assistant"

const CLAUDE_MODEL = "claude-sonnet-4-0"
const CLAUDE_API_URL = "https://api.anthropic.com/v1/messages"
const DEFAULT_TEMPERATURE = 0.01
const MAX_TOKENS = 64000
const DEFAULT_MAX_TOKENS = 1000

type ClaudeMessageRequest struct {
	Model         string         `json:"model"`
	System        string         `json:"system"`
	Messages      ClaudeMessages `json:"messages"`
	MaxTokens     int            `json:"max_tokens"`
	Temperature   float32        `json:"temperature,omitempty"`
	StopSequences []string       `json:"stop_sequences,omitempty"`
}

type ClaudeMessages []ClaudeMessage

type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type Content struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type ClaudeMessageResponse struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Role         string    `json:"role"`
	Content      []Content `json:"content"`
	Model        string    `json:"model"`
	StopReason   string    `json:"stop_reason"`
	StopSequence *string   `json:"stop_sequence"`
	Usage        Usage     `json:"usage"`
}

type ClaudeMessageErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
	Type string `json:"type"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func NewClaudeClient(apiKey string, proxyDSN string, defaultModel string) (api *ClaudeApi, err error) {
	transport := &http.Transport{}
	if proxyDSN != "" {
		proxyURL, err := url.Parse(proxyDSN)
		if err != nil {
			return nil, fmt.Errorf("new claude client proxy dsn error: %s", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	client := &http.Client{
		Transport: transport,
	}
	api = &ClaudeApi{
		apiKey:      apiKey,
		client:      client,
		model:       defaultModel,
		maxTokens:   DEFAULT_MAX_TOKENS,
		temperature: DEFAULT_TEMPERATURE,
	}
	return api, nil
}

func (c *ClaudeApi) SendMessage(claudeMessages ClaudeMessages, systemMessage string) (*ClaudeMessageResponse, error) {
	request := ClaudeMessageRequest{
		Model:       c.model,
		System:      systemMessage,
		Messages:    claudeMessages,
		MaxTokens:   min(c.maxTokens, MAX_TOKENS),
		Temperature: c.temperature,
	}
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", CLAUDE_API_URL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		var respData ClaudeMessageErrorResponse
		err = json.Unmarshal(body, &respData)
		if err != nil {
			return nil, fmt.Errorf("claude SendMessage status code non 200, %d, unmarshall err: %s, body: %s", resp.StatusCode, err, string(body))
		}
		return nil, fmt.Errorf("claude SendMessage status not 200(%d) error: message: %s, type: %s", resp.StatusCode, respData.Error.Message, respData.Error.Type)
	}

	var respData ClaudeMessageResponse
	err = json.Unmarshal(body, &respData)
	if err != nil {
		return nil, fmt.Errorf("claude SendMessage unmarshall err: %s, body: %s", err, string(body))
	}

	return &respData, nil
}
