package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

// GenerateExcerpt generates a summary for the given content using an AI model.
func (s *AIService) GenerateExcerpt(content, baseURL, token, model string) (string, error) {
	if baseURL == "" || token == "" || model == "" {
		return "", errors.New("AI settings are not configured")
	}

	// New, more creative prompt as requested by the user
	prompt := fmt.Sprintf("请为以下文章（标题和内容）生成一个独树一帜、能吸引人点击的中文摘要，摘要需要高度概括文章核心内容，字数在150字以内。请直接返回摘要本身，不要包含任何多余的文字说明或标题。文章内容如下:\n\n%s", content)

	reqBody := openAIRequest{
		Model: model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to AI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("AI API returned non-200 status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("failed to decode AI API response: %w", err)
	}

	if len(apiResp.Choices) == 0 || apiResp.Choices[0].Message.Content == "" {
		return "", errors.New("AI API returned no choices or an empty message")
	}

	return apiResp.Choices[0].Message.Content, nil
}
