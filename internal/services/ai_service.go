package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type FluxPromptResponse struct {
	CoreTheme     []string `json:"core_theme"`
	FullPrompt    string   `json:"full_prompt"`
	EncodedPrompt string   `json:"encoded_prompt"`
}

type ImageUploadResponse struct {
	Ok  bool   `json:"ok"`
	Src string `json:"src"`
}

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
		resp.Body.Close()
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

func (s *AIService) GenerateCover(textForCover, openAIBaseURL, openAIToken, openAIModel, pollinationsToken, coverPrefix string) (string, error) {
	trimmedInput := strings.TrimSpace(textForCover)
	if strings.HasPrefix(trimmedInput, "http") {
		imageURL := trimmedInput
		if coverPrefix != "" {
			imageURL = coverPrefix + imageURL
		}

		req, err := http.NewRequest("GET", imageURL, nil)
		if err != nil {
			return "", fmt.Errorf("创建图片下载请求失败: %w", err)
		}
		// 模拟浏览器行为，明确表示接受 webp 和 avif 格式
		req.Header.Set("Accept", "image/webp,image/avif,image/apng,image/*,*/*;q=0.8")

		resp, err := s.Client.Do(req)
		if err != nil {
			return "", fmt.Errorf("下载提供的图片链接失败: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("下载图片链接返回非 200 状态码: %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		filename := getFilenameFromContentType(contentType)
		log.Printf("下载图片: URL=%s, Content-Type=%s, 生成文件名=%s", imageURL, contentType, filename)
		return s.uploadImage(resp.Body, filename)
	}

	var encodedPrompt string
	var err error

	if len(textForCover) < 500 { // Heuristic to decide if it's a prompt or content
		translatedPrompt, err := s.translateToEnglish(textForCover, openAIBaseURL, openAIToken, openAIModel)
		if err != nil {
			return "", fmt.Errorf("翻译提示词失败: %w", err)
		}
		encodedPrompt = url.QueryEscape(translatedPrompt)
	} else {
		fluxPrompt, err := s.generateFluxPrompt(textForCover, openAIBaseURL, openAIToken, openAIModel)
		if err != nil {
			return "", fmt.Errorf("生成 Flux 提示词失败: %w", err)
		}
		encodedPrompt = fluxPrompt.EncodedPrompt
	}

	imageReader, contentType, imageURL, err := s.generateImageFromPollinations(encodedPrompt, pollinationsToken, coverPrefix)
	if err != nil {
		return "", fmt.Errorf("从 Pollinations 生成图片失败: %w", err)
	}
	defer imageReader.Close()

	filename := getFilenameFromContentType(contentType)
	log.Printf("生成图片: URL=%s, Content-Type=%s, 生成文件名=%s", imageURL, contentType, filename)
	finalImageURL, err := s.uploadImage(imageReader, filename)
	if err != nil {
		return "", fmt.Errorf("上传图片失败: %w", err)
	}

	return finalImageURL, nil
}

func (s *AIService) generateFluxPrompt(content, baseURL, token, model string) (*FluxPromptResponse, error) {
	if baseURL == "" || token == "" || model == "" {
		return nil, errors.New("AI 接口未配置！")
	}

	prompt := `
Please follow the steps below to generate an image prompt for the Flux model based on the article I provide. The final output must be in JSON format.

Step 1: Analyze Article

Deeply read and understand the provided article to accurately extract its core theme and key concepts.
Generate a series of concrete English phrases representing this core theme.

Step 2: Construct Prompt

Embed the Core Words from Step 1 into the [Core Concept] placeholder of the template below.
Generate the final, full, human-readable English prompt.
Template:
[Core Concept], 8k, Exquisite composition, professional artistic quality, worthy of a gallery.

Step 3: Encode Prompt

Take the complete English prompt generated in Step 2 and URL-encode it.
Step 4: Format as JSON

Assemble the results from the previous steps into a single JSON object.
The JSON object must have the following keys:
core_theme: An array of strings containing the English phrases extracted in Step 1.
full_prompt: A string containing the complete, human-readable prompt from Step 2.
encoded_prompt: A string containing the URL-encoded prompt from Step 3.
Final Requirement:

Your final output must be a single, valid JSON object.
Do not include any explanations, comments, or any other text outside of the JSON object itself.
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
		resp.Body.Close()
		return nil, fmt.Errorf("AI API 返回非 200 状态码 %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("解码 AI API 响应失败: %w", err)
	}

	if len(apiResp.Choices) == 0 || apiResp.Choices[0].Message.Content == "" {
		return nil, errors.New("AI API 返回无效回复")
	}

	var fluxResp FluxPromptResponse
	rawJSON := apiResp.Choices[0].Message.Content
	rawJSON = strings.TrimPrefix(rawJSON, "```json\n")
	rawJSON = strings.TrimSuffix(rawJSON, "\n```")

	if err := json.Unmarshal([]byte(rawJSON), &fluxResp); err != nil {
		log.Printf("无法解析 AI 响应 JSON。原始内容: %s", rawJSON)
		return nil, fmt.Errorf("无法解析 AI 响应 JSON: %w", err)
	}

	return &fluxResp, nil
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
		resp.Body.Close()
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

func (s *AIService) generateImageFromPollinations(encodedPrompt, token, coverPrefix string) (io.ReadCloser, string, string, error) {
	if token == "" {
		return nil, "", "", errors.New("Pollinations token 未配置")
	}

	var lastErr error
	const maxRetries = 3
	var imageURL string

	for attempt := 1; attempt <= maxRetries; attempt++ {
		seed := rand.Intn(1000000)
		imageURL = fmt.Sprintf("https://image.pollinations.ai/prompt/%s?width=1024&height=1024&model=flux&nologo=true&seed=%d", encodedPrompt, seed)

		if coverPrefix != "" {
			imageURL = coverPrefix + imageURL
		}

		req, err := http.NewRequest("GET", imageURL, nil)
		if err != nil {
			return nil, "", "", fmt.Errorf("创建 Pollinations 请求失败: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		// 模拟浏览器行为，明确表示接受 webp  格式
		req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")

		resp, err := s.Client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("请求 Pollinations 失败 (attempt %d/%d): %w", attempt, maxRetries, err)
			time.Sleep(3 * time.Second) // Wait before retrying
			continue
		}

		if resp.StatusCode == http.StatusOK {
			contentType := resp.Header.Get("Content-Type")
			return resp.Body, contentType, imageURL, nil
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 524 {
			lastErr = fmt.Errorf("Pollinations API 返回 524 状态码 (attempt %d/%d), URL: %s, Body: %s", attempt, maxRetries, imageURL, string(bodyBytes))
			log.Println(lastErr)
			time.Sleep(3 * time.Second)
			continue
		}

		return nil, "", "", fmt.Errorf("Pollinations API 返回非 200 状态码: %d, URL: %s, Body: %s", resp.StatusCode, imageURL, string(bodyBytes))
	}

	return nil, "", "", fmt.Errorf("经过 %d 次尝试后，从 Pollinations 生成图片仍然失败: %w", maxRetries, lastErr)
}

func (s *AIService) uploadImage(imageReader io.Reader, filename string) (string, error) {
	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)
	errChan := make(chan error, 1)

	go func() {
		defer pipeWriter.Close()
		defer writer.Close()

		part, err := writer.CreateFormFile("image", filename)
		if err != nil {
			errChan <- fmt.Errorf("创建表单文件失败: %w", err)
			return
		}

		if _, err := io.Copy(part, imageReader); err != nil {
			errChan <- fmt.Errorf("复制图片数据到表单失败: %w", err)
			return
		}
		errChan <- nil
	}()

	req, err := http.NewRequest("POST", "https://i.111666.best/image", pipeReader)
	if err != nil {
		return "", fmt.Errorf("创建图片上传请求失败: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Auth-Token", "KtX9AlVOTFoo7ccJyAvRnPmUeZEckAd5")

	resp, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("上传图片请求失败: %w", err)
	}
	defer resp.Body.Close()

	if err := <-errChan; err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("图床 API 返回非 200 状态码: %d, Body: %s", resp.StatusCode, string(bodyBytes))
	}

	var uploadResp ImageUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", fmt.Errorf("解码图床响应失败: %w", err)
	}

	if !uploadResp.Ok {
		return "", errors.New("图床返回上传失败")
	}

	return "https://i.111666.best" + uploadResp.Src, nil
}

func getFilenameFromContentType(contentType string) string {
	// 优先处理常见类型以确保使用标准扩展名
	switch contentType {
	case "image/jpeg":
		return "cover.jpg"
	case "image/png":
		return "cover.png"
	case "image/webp":
		return "cover.webp"
	case "image/gif":
		return "cover.gif"
	case "image/avif":
		return "cover.avif"
	}

	// 对于其他类型，使用 mime 库，但做一些健壮性检查
	exts, err := mime.ExtensionsByType(contentType)
	if err != nil || len(exts) == 0 {
		// 最后的备选方案
		return "cover.jpg"
	}

	// 避免 .jpe 这样不常用的扩展名
	for _, ext := range exts {
		if ext == ".jpeg" || ext == ".jpg" {
			return "cover.jpg"
		}
	}

	// 如果没有找到常见的 jpeg 扩展名，再使用列表中的第一个
	return "cover" + exts[0]
}
