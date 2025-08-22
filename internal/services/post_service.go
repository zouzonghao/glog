package services

import (
	"fmt"
	"glog/internal/models"
	"glog/internal/repository"
	"glog/internal/utils"
	"log"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gosimple/slug"
)

const (
	excerptSeparator = "<!--more-->"
	maxContentForAI  = 2000
	maxExcerptLength = 150 // SEO-friendly length for meta description
)

type PostService struct {
	repo           *repository.PostRepository
	settingService *SettingService
	aiService      *AIService
	postLocks      *sync.Map // Use sync.Map for concurrent access
}

func NewPostService(repo *repository.PostRepository, settingService *SettingService, aiService *AIService) *PostService {
	return &PostService{
		repo:           repo,
		settingService: settingService,
		aiService:      aiService,
		postLocks:      &sync.Map{},
	}
}

// CheckPostLock checks if a post is currently locked for AI processing.
func (s *PostService) CheckPostLock(id uint) bool {
	_, locked := s.postLocks.Load(id)
	return locked
}

func (s *PostService) CreatePost(title, content string, published bool, isPrivate bool, aiSummary bool) (*models.Post, error) {
	// First, save the post to get an ID
	baseSlug := slug.Make(title)
	finalSlug, err := s.findAvailableSlug(baseSlug, 0)
	if err != nil {
		return nil, err
	}

	post := &models.Post{
		Title:     title,
		Slug:      finalSlug,
		Content:   content,
		Excerpt:   utils.GenerateExcerpt(content, maxExcerptLength), // Generate a simple excerpt for meta initially
		Published: published,
		IsPrivate: isPrivate,
	}

	if published {
		post.PublishedAt = time.Now()
	}

	if err := s.repo.Create(post); err != nil {
		return nil, err
	}

	// After saving, check and generate AI summary asynchronously
	go s.generateAndSaveAISummary(post, aiSummary)

	return post, nil
}

func (s *PostService) UpdatePost(id uint, title, content string, published bool, isPrivate bool, aiSummary bool) (*models.Post, error) {
	post, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// If title is changed, update slug
	if post.Title != title {
		baseSlug := slug.Make(title)
		post.Slug, err = s.findAvailableSlug(baseSlug, id)
		if err != nil {
			return nil, err
		}
	}

	post.Title = title
	post.Content = content
	post.IsPrivate = isPrivate
	post.Excerpt = utils.GenerateExcerpt(content, maxExcerptLength) // Keep updating simple excerpt

	if !post.Published && published {
		post.PublishedAt = time.Now()
	}
	post.Published = published

	if err := s.repo.Update(post); err != nil {
		return nil, err
	}

	// After saving, check and generate AI summary asynchronously
	go s.generateAndSaveAISummary(post, aiSummary)

	return post, nil
}

// generateAndSaveAISummary is the core async logic for AI summary generation.
func (s *PostService) generateAndSaveAISummary(post *models.Post, aiSummary bool) {
	// Check if AI summary generation is needed
	if !s.isAISummaryNeeded(post.Content, aiSummary) {
		return
	}

	// Lock the post
	s.postLocks.Store(post.ID, true)
	defer s.postLocks.Delete(post.ID)

	log.Printf("Starting AI summary generation for post ID %d", post.ID)

	// Get AI settings
	settings, err := s.settingService.GetAllSettings()
	if err != nil {
		log.Printf("Error getting settings for AI summary (post ID %d): %v", post.ID, err)
		return
	}
	baseURL := settings["openai_base_url"]
	token := settings["openai_token"]
	model := settings["openai_model"]

	if baseURL == "" || token == "" || model == "" {
		log.Printf("AI settings not configured, skipping summary generation for post ID %d", post.ID)
		return
	}

	// Prepare content for AI
	contentForAI := post.Title + "\n\n" + post.Content
	if utf8.RuneCountInString(contentForAI) > maxContentForAI {
		runes := []rune(contentForAI)
		contentForAI = string(runes[:maxContentForAI])
	}

	needsTitle := post.Title == "未命名"
	aiResp, err := s.aiService.GenerateSummaryAndTitle(contentForAI, needsTitle, baseURL, token, model)
	if err != nil {
		log.Printf("Error generating AI content for post ID %d: %v", post.ID, err)
		return
	}
	summary := "AI摘要：" + aiResp.Summary

	// Update the post with the new summary and title
	latestPost, err := s.repo.FindByID(post.ID)
	if err != nil {
		log.Printf("Error fetching latest post data for ID %d: %v", post.ID, err)
		return
	}

	if needsTitle && aiResp.Title != "" {
		latestPost.Title = aiResp.Title
		baseSlug := slug.Make(aiResp.Title)
		latestPost.Slug, err = s.findAvailableSlug(baseSlug, latestPost.ID)
		if err != nil {
			log.Printf("Error generating slug for new title for post ID %d: %v", post.ID, err)
		}
	}

	// Also update the Excerpt field with the AI-generated summary
	latestPost.Excerpt = utils.GenerateExcerpt(summary, maxExcerptLength)

	// Smartly insert the summary into the main content
	parts := strings.SplitN(latestPost.Content, excerptSeparator, 2)
	if len(parts) == 2 {
		// If separator exists, replace the content before it
		body := parts[1]
		latestPost.Content = summary + "\n\n" + excerptSeparator + body
	} else {
		// If no separator, prepend to the whole content.
		latestPost.Content = summary + "\n\n" + excerptSeparator + "\n\n" + latestPost.Content
	}

	if err := s.repo.Update(latestPost); err != nil {
		log.Printf("Error saving AI summary for post ID %d: %v", post.ID, err)
		return
	}

	log.Printf("Successfully generated and saved AI summary for post ID %d", post.ID)
}

// isAISummaryNeeded checks if the content requires an AI summary.
func (s *PostService) isAISummaryNeeded(content string, aiSummary bool) bool {
	if !aiSummary {
		return false
	}
	parts := strings.SplitN(content, excerptSeparator, 2)
	if len(parts) < 2 {
		// No separator, but user wants AI summary
		return true
	}
	// Check if the part before the separator is empty or just whitespace.
	return strings.TrimSpace(parts[0]) == ""
}

// findAvailableSlug checks for slug uniqueness and appends a counter if needed.
func (s *PostService) findAvailableSlug(baseSlug string, postID uint) (string, error) {
	finalSlug := baseSlug
	counter := 1
	for {
		var exists bool
		var err error
		if postID == 0 {
			exists, err = s.repo.CheckSlugExists(finalSlug)
		} else {
			exists, err = s.repo.CheckSlugExistsForOtherPost(finalSlug, postID)
		}

		if err != nil {
			return "", err
		}
		if !exists {
			break
		}
		finalSlug = fmt.Sprintf("%s-%d", baseSlug, counter)
		counter++
	}
	return finalSlug, nil
}

func (s *PostService) GetPostByID(id uint) (*models.Post, error) {
	return s.repo.FindByID(id)
}

func (s *PostService) GetRenderedPostBySlug(slug string, isLoggedIn bool) (*models.RenderedPost, error) {
	post, err := s.repo.FindBySlug(slug, isLoggedIn)
	if err != nil {
		return nil, err
	}

	var summaryMd, bodyMd string

	if strings.Contains(post.Content, excerptSeparator) {
		parts := strings.SplitN(post.Content, excerptSeparator, 2)
		summaryMd = parts[0]
		bodyMd = parts[1]
	} else {
		summaryMd = "" // No summary if no separator
		bodyMd = post.Content
	}

	summaryHTML, err := utils.RenderMarkdown(summaryMd)
	if err != nil {
		return nil, err
	}

	bodyHTML, err := utils.RenderMarkdown(bodyMd)
	if err != nil {
		return nil, err
	}

	renderedPost := &models.RenderedPost{
		Model:       post.Model,
		Title:       post.Title,
		Slug:        post.Slug,
		Summary:     summaryHTML,
		Body:        bodyHTML,
		Excerpt:     post.Excerpt,
		Published:   post.Published,
		IsPrivate:   post.IsPrivate,
		PublishedAt: post.PublishedAt,
	}

	return renderedPost, nil
}

func (s *PostService) GetAllPublishedPosts(isLoggedIn bool) ([]models.Post, error) {
	return s.repo.FindAllPublished(isLoggedIn)
}

func (s *PostService) GetPostsPage(page, pageSize int) ([]models.Post, int64, error) {
	posts, err := s.repo.FindAll(page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountAll()
	if err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

func formatFTSQuery(query string) string {
	keywords := strings.Fields(strings.ReplaceAll(query, ",", " "))
	for i, keyword := range keywords {
		keywords[i] = keyword + "*"
	}
	return strings.Join(keywords, " AND ")
}

func (s *PostService) SearchPostsPage(query string, page, pageSize int) ([]models.Post, int64, error) {
	ftsQuery := formatFTSQuery(query)
	if ftsQuery == "" {
		return []models.Post{}, 0, nil
	}
	posts, err := s.repo.SearchAll(ftsQuery, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountAllByQuery(ftsQuery)
	if err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

func (s *PostService) SearchPosts(query string, isLoggedIn bool) ([]models.Post, error) {
	ftsQuery := formatFTSQuery(query)
	if ftsQuery == "" {
		return []models.Post{}, nil
	}
	return s.repo.Search(ftsQuery, isLoggedIn)
}

func (s *PostService) DeletePost(id uint) error {
	return s.repo.Delete(id)
}
