package services

import (
	"fmt"
	"glog/internal/models"
	"glog/internal/repository"
	"glog/internal/utils"
	"glog/internal/utils/segmenter"
	"html/template"
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
	baseSlug := slug.Make(title)
	finalSlug, err := s.findAvailableSlug(baseSlug, 0)
	if err != nil {
		return nil, err
	}

	renderedHTML, err := utils.RenderMarkdown(content)
	if err != nil {
		return nil, fmt.Errorf("markdown render failed: %w", err)
	}

	post := &models.Post{
		Title:       title,
		Slug:        finalSlug,
		Content:     content,
		ContentHTML: string(renderedHTML),
		Excerpt:     utils.GenerateExcerpt(content, maxExcerptLength),
		Published:   published,
		IsPrivate:   isPrivate,
	}

	if published {
		post.PublishedAt = time.Now()
	}

	if err := s.repo.Create(post); err != nil {
		return nil, err
	}

	go s.asyncPostSaveOperations(post, aiSummary)

	return post, nil
}

func (s *PostService) UpdatePost(id uint, title, content string, published bool, isPrivate bool, aiSummary bool) (*models.Post, error) {
	post, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if post.Title != title {
		baseSlug := slug.Make(title)
		post.Slug, err = s.findAvailableSlug(baseSlug, id)
		if err != nil {
			return nil, err
		}
	}

	renderedHTML, err := utils.RenderMarkdown(content)
	if err != nil {
		return nil, fmt.Errorf("markdown render failed: %w", err)
	}

	post.Title = title
	post.Content = content
	post.ContentHTML = string(renderedHTML)
	post.IsPrivate = isPrivate
	post.Excerpt = utils.GenerateExcerpt(content, maxExcerptLength)

	if !post.Published && published {
		post.PublishedAt = time.Now()
	}
	post.Published = published

	if err := s.repo.Update(post); err != nil {
		return nil, err
	}

	go s.asyncPostSaveOperations(post, aiSummary)

	return post, nil
}

// asyncPostSaveOperations handles asynchronous post-save operations like FTS indexing and AI summary.
func (s *PostService) asyncPostSaveOperations(post *models.Post, aiSummary bool) {
	// 首先，生成 AI 摘要，这会修改数据库中的文章
	s.generateAndSaveAISummary(post, aiSummary)

	// 摘要保存后，获取文章的最新版本以确保我们使用的是最终内容
	latestPost, err := s.repo.FindByID(post.ID)
	if err != nil {
		log.Printf("获取最新文章以更新 FTS 索引失败，文章 ID %d: %v", post.ID, err)
		return
	}

	// 现在，使用最终的、完整的内容（可能包含 AI 摘要）来更新 FTS 索引
	log.Printf("使用最终内容为文章 ID 更新 FTS 索引 %d", latestPost.ID)
	segmentedTitle := segmenter.SegmentTextForIndex(latestPost.Title)
	segmentedContent := segmenter.SegmentTextForIndex(latestPost.Content)
	if err := s.repo.UpdateFtsIndex(latestPost.ID, segmentedTitle, segmentedContent); err != nil {
		log.Printf("使用最终内容更新 FTS 索引失败，文章 ID %d: %v", latestPost.ID, err)
	}
}

// generateAndSaveAISummary is the core async logic for AI summary generation.
func (s *PostService) generateAndSaveAISummary(post *models.Post, aiSummary bool) {
	if !s.isAISummaryNeeded(post.Content, aiSummary) {
		return
	}

	s.postLocks.Store(post.ID, true)
	defer s.postLocks.Delete(post.ID)

	log.Printf("开始为文章 ID 生成 AI 摘要 %d", post.ID)

	settings, err := s.settingService.GetAllSettings()
	if err != nil {
		log.Printf("获取 AI 摘要设置失败 (文章 ID %d): %v", post.ID, err)
		return
	}
	baseURL := settings["openai_base_url"]
	token := settings["openai_token"]
	model := settings["openai_model"]

	if baseURL == "" || token == "" || model == "" {
		log.Printf("AI 未配置，跳过为文章 ID 生成摘要 %d", post.ID)
		return
	}

	contentForAI := post.Title + "\n\n" + post.Content
	if utf8.RuneCountInString(contentForAI) > maxContentForAI {
		runes := []rune(contentForAI)
		contentForAI = string(runes[:maxContentForAI])
	}

	needsTitle := post.Title == "未命名标题"
	aiResp, err := s.aiService.GenerateSummaryAndTitle(contentForAI, needsTitle, baseURL, token, model)
	if err != nil {
		log.Printf("为文章 ID 生成 AI 内容失败 %d: %v", post.ID, err)
		return
	}
	summary := "AI摘要：" + aiResp.Summary

	latestPost, err := s.repo.FindByID(post.ID)
	if err != nil {
		log.Printf("获取文章最新数据失败，ID %d: %v", post.ID, err)
		return
	}

	if needsTitle && aiResp.Title != "" {
		latestPost.Title = aiResp.Title
		baseSlug := slug.Make(aiResp.Title)
		latestPost.Slug, err = s.findAvailableSlug(baseSlug, latestPost.ID)
		if err != nil {
			log.Printf("为文章新标题生成 slug 失败，文章 ID %d: %v", post.ID, err)
		}
	}

	latestPost.Excerpt = utils.GenerateExcerpt(summary, maxExcerptLength)

	parts := strings.SplitN(latestPost.Content, excerptSeparator, 2)
	if len(parts) == 2 {
		body := parts[1]
		latestPost.Content = summary + "\n\n" + excerptSeparator + body
	} else {
		latestPost.Content = summary + "\n\n" + excerptSeparator + "\n\n" + latestPost.Content
	}

	// Re-render content to include AI summary
	renderedHTML, err := utils.RenderMarkdown(latestPost.Content)
	if err != nil {
		log.Printf("重新渲染 AI 摘要内容失败，文章 ID %d: %v", post.ID, err)
		return
	}
	latestPost.ContentHTML = string(renderedHTML)

	if err := s.repo.Update(latestPost); err != nil {
		log.Printf("保存 AI 摘要失败，文章 ID %d: %v", post.ID, err)
		return
	}

	log.Printf("成功为文章 ID 生成并保存 AI 摘要 %d", post.ID)
}

// isAISummaryNeeded checks if the content requires an AI summary.
func (s *PostService) isAISummaryNeeded(content string, aiSummary bool) bool {
	if !aiSummary {
		return false
	}
	parts := strings.SplitN(content, excerptSeparator, 2)
	if len(parts) < 2 {
		return true
	}
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

	var renderedHTML string
	if post.ContentHTML == "" {
		// Fallback: Render markdown in real-time if ContentHTML is empty
		log.Printf("警告：文章 ID %d 的 ContentHTML 为空，正在实时渲染。", post.ID)
		renderedBytes, err := utils.RenderMarkdown(post.Content)
		if err != nil {
			// Even if render fails, return the rest of the post data
			log.Printf("实时渲染 Markdown 失败，文章 ID %d: %v", post.ID, err)
		} else {
			renderedHTML = string(renderedBytes)
			// Self-healing: Update the post in the background
			go func(p *models.Post, html string) {
				p.ContentHTML = html
				if err := s.repo.Update(p); err != nil {
					log.Printf("后台更新 ContentHTML 失败，文章 ID %d: %v", p.ID, err)
				}
			}(post, renderedHTML)
		}
	} else {
		renderedHTML = post.ContentHTML
	}

	var summaryHTML, bodyHTML template.HTML
	separatorHTML := "<!--more-->" // Use the raw separator for splitting HTML

	if strings.Contains(renderedHTML, separatorHTML) {
		parts := strings.SplitN(renderedHTML, separatorHTML, 2)
		summaryHTML = template.HTML(parts[0])
		bodyHTML = template.HTML(parts[1])
	} else {
		summaryHTML = ""
		bodyHTML = template.HTML(renderedHTML)
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

func (s *PostService) GetPublishedPostsPage(page, pageSize int, isLoggedIn bool) ([]models.Post, int64, error) {
	posts, err := s.repo.FindPublishedPage(page, pageSize, isLoggedIn)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountPublished(isLoggedIn)
	if err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

func (s *PostService) GetPostsPage(page, pageSize int, query, status string) ([]models.Post, int64, error) {
	posts, err := s.repo.FindAll(page, pageSize, query, status)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountAll(query, status)
	if err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

func (s *PostService) SearchPublishedPostsPage(query string, page, pageSize int, isLoggedIn bool) ([]models.Post, int64, error) {
	if query == "" {
		return []models.Post{}, 0, nil
	}

	ftsQuery := segmenter.SegmentTextForQuery(query)
	// If the query becomes empty after segmentation and filtering (e.g., only punctuation was entered),
	// return no results to avoid a database error from an empty MATCH clause.
	if ftsQuery == "" {
		return []models.Post{}, 0, nil
	}
	posts, err := s.repo.SearchPage(ftsQuery, page, pageSize, isLoggedIn)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountByQuery(ftsQuery, isLoggedIn)
	if err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

func (s *PostService) DeletePost(id uint) error {
	if err := s.repo.DeleteFtsIndex(id); err != nil {
		log.Printf("删除 FTS 索引失败，文章 ID %d: %v", id, err)
	}
	return s.repo.Delete(id)
}
