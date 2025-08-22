package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// AIService handles interactions with an OpenAI compatible API.
type AIService struct {
	Client *http.Client
}

// NewAIService creates a new AIService.
func NewAIService() *AIService {
	return &AIService{
		Client: &http.Client{Timeout: 120 * time.Second}, // Increased timeout for AI generation
	}
}

// OpenAI API request structure
type openAIRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAI API response structure
type openAIResponse struct {
	Choices []choice `json:"choices"`
}

type choice struct {
	Message message `json:"message"`
}

// AIResponse defines the structure for the JSON response from the AI.
type AIResponse struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// GenerateSummaryAndTitle generates a summary and optionally a title for the given content.
func (s *AIService) GenerateSummaryAndTitle(content string, needsTitle bool, baseURL, token, model string) (*AIResponse, error) {
	if baseURL == "" || token == "" || model == "" {
		return nil, errors.New("AI settings are not configured")
	}

	prompt := "请为以下文章生成摘要。"
	if needsTitle {
		prompt = "请为以下文章生成标题和摘要。"
	}
	prompt += "请严格按照以下JSON格式返回，不要添加任何额外的解释或说明文字：\n"
	prompt += "`{\"title\": \"生成的标题（如果需要）\", \"summary\": \"生成的摘要\"}`\n"
	prompt += "如果不需要生成标题，请将title字段留空。\n"
	prompt += "文章内容如下：\n\n" + content

	reqBody := openAIRequest{
		Model: model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to AI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AI API returned non-200 status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode AI API response: %w", err)
	}

	if len(apiResp.Choices) == 0 || apiResp.Choices[0].Message.Content == "" {
		return nil, errors.New("AI API returned no choices or an empty message")
	}

	var aiResp AIResponse
	rawJSON := apiResp.Choices[0].Message.Content
	// It's possible the AI returns the JSON inside a code block, so we trim it.
	rawJSON = strings.TrimPrefix(rawJSON, "```json\n")
	rawJSON = strings.TrimSuffix(rawJSON, "\n```")

	if err := json.Unmarshal([]byte(rawJSON), &aiResp); err != nil {
		log.Printf("Failed to unmarshal AI response JSON. Raw content: %s", rawJSON)
		return nil, fmt.Errorf("failed to unmarshal AI response JSON: %w", err)
	}

	return &aiResp, nil
}
