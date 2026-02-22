package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type ImagePromptResponse struct {
	CoreTheme  []string `json:"core_theme"`
	FullPrompt string   `json:"full_prompt"`
}

// AI 服务配置常量
const (
	aiServiceDefaultTimeout = 3 * time.Minute  // 基础模型（摘要生成等）默认超时时间
	aiServiceCoverTimeout   = 10 * time.Minute // 封面生成超时时间
)

// AIService 处理与 OpenAI 兼容 API 的交互。
type AIService struct {
	Client *http.Client
}

// NewAIService 创建新的 AIService 实例。
func NewAIService() *AIService {
	return &AIService{
		Client: &http.Client{Timeout: aiServiceDefaultTimeout},
	}
}

// OpenAI API 请求结构
type openAIRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAI API 响应结构
type openAIResponse struct {
	Choices []choice `json:"choices"`
}

type choice struct {
	Message message `json:"message"`
}

// AIResponse 定义 AI JSON 响应的结构。
type AIResponse struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// ImageAPI 请求结构
type imageAPIRequest struct {
	Prompt   string `json:"prompt"`
	Model    string `json:"model"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	Seed     int64  `json:"seed,omitempty"`
	Steps    int    `json:"steps,omitempty"`
}

// ImageAPI 响应结构
type imageAPIResponse struct {
	Status   string `json:"status"`
	ImageURL string `json:"image_url"`
	Error    string `json:"error"`
}

// ImageAPIModelInfo 定义 ImageAPI 单个模型的结构。
type ImageAPIModelInfo struct {
	Name string `json:"name"`
}

// ImageAPIProviderInfo 定义提供商及其模型的结构。
type ImageAPIProviderInfo struct {
	Provider string              `json:"provider"`
	Models   []ImageAPIModelInfo `json:"models"`
}

// GenerateSummaryAndTitle 为给定内容生成摘要，可选生成标题。
func (s *AIService) GenerateSummaryAndTitle(ctx context.Context, content string, needsTitle bool, baseURL, token, model string) (*AIResponse, error) {
	log.Printf("开始请求 AI 生成摘要和标题 (Model: %s)...", model)
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

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewBuffer(jsonData))
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
	// AI 可能将 JSON 放在代码块中返回，需要去除代码块标记
	rawJSON = strings.TrimPrefix(rawJSON, "```json\n")
	rawJSON = strings.TrimSuffix(rawJSON, "\n```")

	if err := json.Unmarshal([]byte(rawJSON), &aiResp); err != nil {
		log.Printf("无法解析 AI 响应 JSON。原始内容: %s", rawJSON)
		return nil, fmt.Errorf("无法解析 AI 响应 JSON: %w", err)
	}

	log.Println("AI 摘要和标题生成成功")
	return &aiResp, nil
}

func (s *AIService) GenerateCover(ctx context.Context, prompt, content, openAIBaseURL, openAIToken, openAIModel, imageAPIURL, imageAPIToken, imageAPIModel string) (string, error) {
	log.Printf("开始生成 AI 封面 (Model: %s)...", imageAPIModel)
	trimmedPrompt := strings.TrimSpace(prompt)
	if strings.HasPrefix(trimmedPrompt, "http://") || strings.HasPrefix(trimmedPrompt, "https://") {
		return trimmedPrompt, nil
	}

	var englishPrompt string
	var err error

	if trimmedPrompt != "" {
		// 用户提供了提示词，翻译为英文
		englishPrompt, err = s.translateToEnglish(ctx, trimmedPrompt, openAIBaseURL, openAIToken, openAIModel)
		if err != nil {
			return "", fmt.Errorf("翻译提示词失败: %w", err)
		}
	} else {
		// 提示词为空，根据文章内容生成
		imagePrompt, err := s.generateImagePromptFromContent(ctx, content, openAIBaseURL, openAIToken, openAIModel)
		if err != nil {
			return "", fmt.Errorf("根据文章内容生成提示词失败: %w", err)
		}
		englishPrompt = imagePrompt.FullPrompt
	}

	if englishPrompt == "" {
		return "", errors.New("生成的英文提示词为空")
	}

	imageURL, err := s.generateImageFromAPI(ctx, englishPrompt, imageAPIURL, imageAPIToken, imageAPIModel)
	if err != nil {
		log.Printf("AI 封面生成失败: %v", err)
		return "", err
	}
	log.Printf("AI 封面生成成功: %s", imageURL)
	return imageURL, nil
}

func (s *AIService) generateImageFromAPI(ctx context.Context, prompt, apiURL, apiToken, apiModel string) (string, error) {
	log.Printf("调用 ImageAPI 生成图片 (Model: %s, Prompt 长度: %d)", apiModel, len(prompt))
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
	req, err := http.NewRequestWithContext(ctx, "POST", generateURL, bytes.NewBuffer(jsonData))
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

func (s *AIService) generateImagePromptFromContent(ctx context.Context, content, baseURL, token, model string) (*ImagePromptResponse, error) {
	log.Printf("正在根据文章内容生成 AI 绘图提示词 (Model: %s)...", model)
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

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewBuffer(jsonData))
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

func (s *AIService) translateToEnglish(ctx context.Context, text, baseURL, token, model string) (string, error) {
	log.Printf("正在将提示词翻译为英文 (Model: %s)...", model)
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

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewBuffer(jsonData))
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

// TestImageAPI 通过获取可用模型列表来测试 ImageAPI 连接。
func (s *AIService) TestImageAPI(ctx context.Context, apiURL, apiToken string) ([]ImageAPIProviderInfo, error) {
	if apiURL == "" || apiToken == "" {
		return nil, errors.New("ImageAPI URL 或 Token 未配置！")
	}

	// 构建获取模型列表的请求 URL
	modelsURL := strings.TrimSuffix(apiURL, "/") + "/api/v1/models"

	req, err := http.NewRequestWithContext(ctx, "GET", modelsURL, nil)
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
