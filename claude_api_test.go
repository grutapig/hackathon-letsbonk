package main

import (
	"encoding/json"
	"github.com/grutapig/hackaton/claude"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestClaudeApi_SendMessage(t *testing.T) {
	err := godotenv.Load()
	assert.NoError(t, err)
	claudeApi, err := claude.NewClaudeClient(os.Getenv(ENV_CLAUDE_API_KEY), os.Getenv(ENV_PROXY_DSN), claude.CLAUDE_MODEL)
	assert.NoError(t, err)
	response, err := claudeApi.SendMessage(
		claude.ClaudeMessages{
			{claude.ROLE_USER, "hi solve this: 54+99"},
			{claude.ROLE_ASSISTANT, "{"},
		},
		"response JSON format {sum:365,param_first:1,param_second:2}")
	assert.NoError(t, err)
	assert.Greater(t, len(response.Content), 0)
	responseStruct := struct {
		Sum         int `json:"sum"`
		ParamFirst  int `json:"param_first"`
		ParamSecond int `json:"param_second"`
	}{}
	err = json.Unmarshal([]byte("{"+response.Content[0].Text), &responseStruct)
	assert.NoError(t, err)
	assert.Equal(t, responseStruct.Sum, 153)
	assert.Equal(t, responseStruct.ParamFirst, 54)
	assert.Equal(t, responseStruct.ParamSecond, 99)
}
