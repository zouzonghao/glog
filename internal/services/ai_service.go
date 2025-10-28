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

	"glog/internal/utils"
)

type ImagePromptResponse struct {
	CoreTheme  []string `json:"core_theme"`
	FullPrompt string   `json:"full_prompt"`
}

// AIService handles interactions with an OpenAI compatible API.
type AIService struct {
	Client *http.Client
}

// NewAIService creates a new AIService.
func NewAIService() *AIService {
	return &AIService{
		Client: &http.Client{Timeout: 180 * time.Second}, // Increased timeout for AI generation
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

// ImageAPI request structure
type imageAPIRequest struct {
	Prompt   string `json:"prompt"`
	Model    string `json:"model"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	Seed     int64  `json:"seed,omitempty"`
	Steps    int    `json:"steps,omitempty"`
}

// ImageAPI response structure
type imageAPIResponse struct {
	Status   string `json:"status"`
	ImageURL string `json:"image_url"`
	Error    string `json:"error"`
}

// ImageAPIModelInfo defines the structure for a single model from the ImageAPI.
type ImageAPIModelInfo struct {
	Name string `json:"name"`
}

// ImageAPIProviderInfo defines the structure for a provider and its models.
type ImageAPIProviderInfo struct {
	Provider string              `json:"provider"`
	Models   []ImageAPIModelInfo `json:"models"`
}

// GenerateSummaryAndTitle generates a summary and optionally a title for the given content.
func (s *AIService) GenerateSummaryAndTitle(content string, needsTitle bool, baseURL, token, model string) (*AIResponse, error) {
	if baseURL == "" || token == "" || model == "" {
		return nil, errors.New("AI 接口未配置！")
	}

	prompt := "请为以下文章生成摘要。"
	if needsTitle {
		prompt = "请为以下文章生成标题和摘要。"
	}
	prompt += "摘要严格限制50字以内，需简短精炼。请严格按照以下JSON格式返回，不要添加任何额外的解释或说明文字：\n"
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
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求至 AI API 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AI API 返回非 200 状态码 %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("解码 AI API 响应失败: %w", err)
	}

	if len(apiResp.Choices) == 0 || apiResp.Choices[0].Message.Content == "" {
		return nil, errors.New("AI API 返回无效回复")
	}

	var aiResp AIResponse
	rawJSON := apiResp.Choices[0].Message.Content
	// It's possible the AI returns the JSON inside a code block, so we trim it.
	rawJSON = strings.TrimPrefix(rawJSON, "```json\n")
	rawJSON = strings.TrimSuffix(rawJSON, "\n```")

	if err := json.Unmarshal([]byte(rawJSON), &aiResp); err != nil {
		log.Printf("无法解析 AI 响应 JSON。原始内容: %s", rawJSON)
		return nil, fmt.Errorf("无法解析 AI 响应 JSON: %w", err)
	}

	return &aiResp, nil
}

func (s *AIService) GenerateCover(postTitle, prompt, content, openAIBaseURL, openAIToken, openAIModel, imageAPIURL, imageAPIToken, imageAPIModel string) (string, error) {
	trimmedPrompt := strings.TrimSpace(prompt)
	if strings.HasPrefix(trimmedPrompt, "http://") || strings.HasPrefix(trimmedPrompt, "https://") {
		utils.AILog("文章 '%s': 已直接使用提供的链接作为封面", postTitle)
		return trimmedPrompt, nil
	}

	var englishPrompt string
	var err error

	if trimmedPrompt != "" {
		// User provided a prompt, translate it
		englishPrompt, err = s.translateToEnglish(trimmedPrompt, openAIBaseURL, openAIToken, openAIModel)
		if err != nil {
			return "", fmt.Errorf("翻译提示词失败: %w", err)
		}
	} else {
		// Prompt is empty, generate prompt from article content
		imagePrompt, err := s.generateImagePromptFromContent(content, openAIBaseURL, openAIToken, openAIModel)
		if err != nil {
			return "", fmt.Errorf("根据文章内容生成提示词失败: %w", err)
		}
		englishPrompt = imagePrompt.FullPrompt
	}

	if englishPrompt == "" {
		return "", errors.New("生成的英文提示词为空")
	}

	imageURL, err := s.generateImageFromAPI(englishPrompt, imageAPIURL, imageAPIToken, imageAPIModel)
	if err != nil {
		return "", err
	}
	utils.AILog("文章 '%s': AI封面生成成功", postTitle)
	return imageURL, nil
}

func (s *AIService) generateImageFromAPI(prompt, apiURL, apiToken, apiModel string) (string, error) {
	if apiURL == "" || apiToken == "" || apiModel == "" {
		return "", errors.New("ImageAPI 未配置！")
	}

	reqBody := imageAPIRequest{
		Prompt: prompt,
		Model:  apiModel,
		Width:  1024,
		Height: 1024,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化 ImageAPI 请求体失败: %w", err)
	}

	generateURL := strings.TrimSuffix(apiURL, "/") + "/api/v1/generate"
	req, err := http.NewRequest("POST", generateURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建 ImageAPI 请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiToken)

	resp, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求至 ImageAPI 失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 ImageAPI 响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ImageAPI 返回非 200 状态码 %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp imageAPIResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return "", fmt.Errorf("解码 ImageAPI 响应失败: %w. 响应原文: %s", err, string(bodyBytes))
	}

	if apiResp.Status != "success" {
		return "", fmt.Errorf("ImageAPI 返回错误: %s", apiResp.Error)
	}

	if apiResp.ImageURL == "" {
		return "", errors.New("ImageAPI 未返回图片 URL")
	}

	return apiResp.ImageURL, nil
}

func (s *AIService) generateImagePromptFromContent(content, baseURL, token, model string) (*ImagePromptResponse, error) {
	if baseURL == "" || token == "" || model == "" {
		return nil, errors.New("AI 接口未配置！")
	}

	prompt := `
Please analyze the following article and extract its core theme and key concepts.
Based on the analysis, generate a concise, concrete, and vivid English phrase that can be used as a prompt for an image generation model.
The final output must be in JSON format, containing the core theme and the full prompt.
Template for the full prompt: "[Generated English Phrase], 4k, Exquisite composition, professional artistic quality, worthy of a gallery, cinematic composition."
The JSON object must have the following keys:
- core_theme: An array of strings containing the English phrases extracted.
- full_prompt: A string containing the complete, human-readable prompt.
Your final output must be a single, valid JSON object. Do not include any explanations or text outside of the JSON object itself.
Now, please perform the above tasks for the following article:
` + content

	reqBody := openAIRequest{
		Model: model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求至 AI API 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AI API 返回非 200 状态码 %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("解码 AI API 响应失败: %w", err)
	}

	if len(apiResp.Choices) == 0 || apiResp.Choices[0].Message.Content == "" {
		return nil, errors.New("AI API 返回无效回复")
	}

	var promptResp ImagePromptResponse
	rawJSON := apiResp.Choices[0].Message.Content
	rawJSON = strings.TrimPrefix(rawJSON, "```json\n")
	rawJSON = strings.TrimSuffix(rawJSON, "\n```")

	if err := json.Unmarshal([]byte(rawJSON), &promptResp); err != nil {
		log.Printf("无法解析 AI 响应 JSON。原始内容: %s", rawJSON)
		return nil, fmt.Errorf("无法解析 AI 响应 JSON: %w", err)
	}

	return &promptResp, nil
}

func (s *AIService) translateToEnglish(text, baseURL, token, model string) (string, error) {
	if baseURL == "" || token == "" || model == "" {
		return "", errors.New("AI 接口未配置！")
	}

	prompt := "Translate the following text to English, output only the translated content, without any other characters or explanations:\n\n" + text

	reqBody := openAIRequest{
		Model: model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求体失败: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求至 AI API 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("AI API 返回非 200 状态码 %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("解码 AI API 响应失败: %w", err)
	}

	if len(apiResp.Choices) == 0 || apiResp.Choices[0].Message.Content == "" {
		return "", errors.New("AI API 返回无效回复")
	}

	return apiResp.Choices[0].Message.Content, nil
}

// TestImageAPI tests the connection to the ImageAPI by fetching the available models.
func (s *AIService) TestImageAPI(apiURL, apiToken string) ([]ImageAPIProviderInfo, error) {
	if apiURL == "" || apiToken == "" {
		return nil, errors.New("ImageAPI URL 或 Token 未配置！")
	}

	// Construct the request URL for fetching models
	modelsURL := strings.TrimSuffix(apiURL, "/") + "/api/v1/models"

	req, err := http.NewRequest("GET", modelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建 ImageAPI 测试请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求至 ImageAPI 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ImageAPI 返回非 200 状态码 %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Try to decode the response to ensure it's a valid JSON array (list of models)
	var models []ImageAPIProviderInfo
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("解码 ImageAPI 模型列表响应失败: %w", err)
	}

	return models, nil
}
