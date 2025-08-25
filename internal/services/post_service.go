package services

import (
	"fmt"
	"glog/internal/models"
	"glog/internal/repository"
	"glog/internal/utils"
	"glog/internal/utils/segmenter"
	"html/template"
	"strings"
	"sync"
	"time"

	"github.com/gosimple/slug"
	"gorm.io/gorm"
)

// PostLockManager manages locks on posts to prevent concurrent AI processing.
var (
	postLocks   = make(map[uint]bool)
	postLocksMu sync.Mutex
)

type PostService struct {
	repo           *repository.PostRepository
	settingService *SettingService
	aiService      *AIService
}

func NewPostService(repo *repository.PostRepository, settingService *SettingService, aiService *AIService) *PostService {
	return &PostService{
		repo:           repo,
		settingService: settingService,
		aiService:      aiService,
	}
}

// LockPost locks a post for processing.
func (s *PostService) LockPost(postID uint) {
	postLocksMu.Lock()
	defer postLocksMu.Unlock()
	postLocks[postID] = true
}

// UnlockPost unlocks a post after processing.
func (s *PostService) UnlockPost(postID uint) {
	postLocksMu.Lock()
	defer postLocksMu.Unlock()
	delete(postLocks, postID)
}

// CheckPostLock checks if a post is currently locked.
func (s *PostService) CheckPostLock(postID uint) bool {
	postLocksMu.Lock()
	defer postLocksMu.Unlock()
	return postLocks[postID]
}

func (s *PostService) CreatePost(title, content string, isPrivate bool, aiSummary bool, publishedAt time.Time) (*models.Post, error) {
	if title == "" {
		title = "未命名标题"
	}

	sanitizedContent := content

	// Generate excerpt
	excerpt := utils.GenerateExcerpt(sanitizedContent, 150)

	// Generate slug
	slugStr, err := s.generateUniqueSlug(title, 0)
	if err != nil {
		return nil, err
	}

	htmlContent, err := utils.RenderMarkdown(sanitizedContent)
	if err != nil {
		return nil, err
	}

	post := &models.Post{
		Title:       title,
		Slug:        slugStr,
		Content:     sanitizedContent,
		ContentHTML: string(htmlContent),
		Excerpt:     excerpt,
		IsPrivate:   isPrivate,
		PublishedAt: publishedAt,
	}

	err = s.repo.Create(post)
	if err != nil {
		return nil, err
	}

	// Update FTS index
	segmentedTitle := segmenter.SegmentTextForIndex(post.Title)
	segmentedContent := segmenter.SegmentTextForIndex(post.Content)
	err = s.repo.UpdateFtsIndex(post.ID, segmentedTitle, segmentedContent)
	if err != nil {
		// Log the error but don't fail the whole operation
		fmt.Printf("更新 FTS 索引失败 for post ID %d: %v\n", post.ID, err)
	}

	// Trigger AI summary generation if requested
	if aiSummary {
		s.LockPost(post.ID)
		go func() {
			defer s.UnlockPost(post.ID)
			aiResp, err := s.aiService.GenerateSummaryAndTitle(post.Content, title == "未命名标题", "", "", "")
			if err != nil {
				fmt.Printf("AI 摘要生成失败 for post ID %d: %v\n", post.ID, err)
				return
			}

			updateMap := make(map[string]interface{})
			if aiResp.Summary != "" {
				updateMap["excerpt"] = aiResp.Summary
			}
			if aiResp.Title != "" && aiResp.Title != post.Title {
				updateMap["title"] = aiResp.Title
				newSlug, slugErr := s.generateUniqueSlug(aiResp.Title, post.ID)
				if slugErr == nil {
					updateMap["slug"] = newSlug
				} else {
					fmt.Printf("为 AI 生成的标题更新 slug 失败 for post ID %d: %v\n", post.ID, slugErr)
				}
			}

			if len(updateMap) > 0 {
				if err := s.repo.UpdateFields(post.ID, updateMap); err != nil {
					fmt.Printf("用 AI 生成的内容更新文章失败 for post ID %d: %v\n", post.ID, err)
				} else {
					// Re-index FTS if title was updated
					if aiResp.Title != "" {
						segmentedTitle := segmenter.SegmentTextForIndex(aiResp.Title)
						s.repo.UpdateFtsIndex(post.ID, segmentedTitle, segmentedContent)
					}
				}
			}
		}()
	}

	return post, nil
}

func (s *PostService) UpdatePost(id uint, title, content string, isPrivate bool, aiSummary bool, publishedAt time.Time) (*models.Post, error) {
	post, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if title == "" {
		title = "未命名标题"
	}

	// Sanitize content
	sanitizedContent := content

	htmlContent, err := utils.RenderMarkdown(sanitizedContent)
	if err != nil {
		return nil, err
	}

	// If title has changed, generate a new unique slug
	if post.Title != title {
		newSlug, err := s.generateUniqueSlug(title, id)
		if err != nil {
			return nil, err
		}
		post.Slug = newSlug
	}

	post.Title = title
	post.Content = sanitizedContent
	post.ContentHTML = string(htmlContent)
	post.Excerpt = utils.GenerateExcerpt(sanitizedContent, 150)
	post.IsPrivate = isPrivate
	post.PublishedAt = publishedAt

	err = s.repo.Update(post)
	if err != nil {
		return nil, err
	}

	// Update FTS index
	segmentedTitle := segmenter.SegmentTextForIndex(post.Title)
	segmentedContent := segmenter.SegmentTextForIndex(post.Content)
	err = s.repo.UpdateFtsIndex(post.ID, segmentedTitle, segmentedContent)
	if err != nil {
		fmt.Printf("更新 FTS 索引失败 for post ID %d: %v\n", post.ID, err)
	}

	// Trigger AI summary generation if requested
	if aiSummary {
		s.LockPost(post.ID)
		go func() {
			defer s.UnlockPost(post.ID)
			aiResp, err := s.aiService.GenerateSummaryAndTitle(post.Content, title == "未命名标题", "", "", "")
			if err != nil {
				fmt.Printf("AI 摘要生成失败 for post ID %d: %v\n", post.ID, err)
				return
			}

			updateMap := make(map[string]interface{})
			if aiResp.Summary != "" {
				updateMap["excerpt"] = aiResp.Summary
			}
			if aiResp.Title != "" && aiResp.Title != post.Title {
				updateMap["title"] = aiResp.Title
				newSlug, slugErr := s.generateUniqueSlug(aiResp.Title, post.ID)
				if slugErr == nil {
					updateMap["slug"] = newSlug
				} else {
					fmt.Printf("为 AI 生成的标题更新 slug 失败 for post ID %d: %v\n", post.ID, slugErr)
				}
			}

			if len(updateMap) > 0 {
				if err := s.repo.UpdateFields(post.ID, updateMap); err != nil {
					fmt.Printf("用 AI 生成的内容更新文章失败 for post ID %d: %v\n", post.ID, err)
				} else {
					// Re-index FTS if title was updated
					if aiResp.Title != "" {
						segmentedTitle := segmenter.SegmentTextForIndex(aiResp.Title)
						s.repo.UpdateFtsIndex(post.ID, segmentedTitle, segmentedContent)
					}
				}
			}
		}()
	}

	return post, nil
}

func (s *PostService) DeletePost(id uint) error {
	// First, delete the FTS index entry
	err := s.repo.DeleteFtsIndex(id)
	if err != nil {
		// Log the error but continue with post deletion
		fmt.Printf("删除 FTS 索引失败 for post ID %d: %v\n", id, err)
	}
	// Then, delete the post itself
	return s.repo.Delete(id)
}

func (s *PostService) GetPostByID(id uint) (*models.Post, error) {
	return s.repo.FindByID(id)
}

func (s *PostService) GetPostBySlug(slug string, isLoggedIn bool) (*models.RenderedPost, error) {
	post, err := s.repo.FindBySlug(slug, isLoggedIn)
	if err != nil {
		return nil, err
	}
	return s.renderPost(post), nil
}

func (s *PostService) GetPostsPage(page, pageSize int, isLoggedIn bool) ([]models.RenderedPost, int, error) {
	posts, err := s.repo.FindPage(page, pageSize, isLoggedIn)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(isLoggedIn)
	if err != nil {
		return nil, 0, err
	}

	renderedPosts := make([]models.RenderedPost, len(posts))
	for i, post := range posts {
		renderedPosts[i] = *s.renderPost(&post)
	}

	return renderedPosts, int(total), nil
}

func (s *PostService) GetPostsPageByAdmin(page, pageSize int, query, status string) ([]models.Post, int, error) {
	posts, err := s.repo.FindAllByAdmin(page, pageSize, query, status)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountAllByAdmin(query, status)
	if err != nil {
		return nil, 0, err
	}
	return posts, int(total), nil
}

func (s *PostService) SearchPostsPage(query string, page, pageSize int, isLoggedIn bool) ([]models.RenderedPost, int, error) {
	posts, err := s.repo.SearchPage(query, page, pageSize, isLoggedIn)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountByQuery(query, isLoggedIn)
	if err != nil {
		return nil, 0, err
	}

	renderedPosts := make([]models.RenderedPost, len(posts))
	for i, post := range posts {
		renderedPosts[i] = *s.renderPost(&post)
	}

	return renderedPosts, int(total), nil
}

func (s *PostService) renderPost(post *models.Post) *models.RenderedPost {
	// Use a placeholder for the summary/body split
	const moreTag = "<!--more-->"
	content := post.ContentHTML
	summary := content
	body := ""

	if idx := strings.Index(content, moreTag); idx != -1 {
		summary = content[:idx]
		body = content[idx+len(moreTag):]
	}

	return &models.RenderedPost{
		ID:          post.ID,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
		PublishedAt: post.PublishedAt,
		Title:       post.Title,
		Slug:        post.Slug,
		Summary:     template.HTML(summary),
		Body:        template.HTML(body),
		Excerpt:     post.Excerpt,
		IsPrivate:   post.IsPrivate,
	}
}

// generateUniqueSlug checks for slug uniqueness and appends a counter if needed.
func (s *PostService) generateUniqueSlug(title string, postID uint) (string, error) {
	baseSlug := slug.Make(title)
	if baseSlug == "" {
		baseSlug = "untitled"
	}
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

// GetAllPostsForBackup retrieves all posts for backup.
func (s *PostService) GetAllPostsForBackup() ([]models.PostBackup, error) {
	posts, err := s.repo.FindAllForBackup()
	if err != nil {
		return nil, err
	}

	backupPosts := make([]models.PostBackup, len(posts))
	for i, p := range posts {
		backupPosts[i] = models.PostBackup{
			Title:       p.Title,
			Content:     p.Content,
			IsPrivate:   p.IsPrivate,
			PublishedAt: p.PublishedAt,
		}
	}
	return backupPosts, nil
}

// CreatePostsFromBackup creates posts from a backup file, ensuring business logic is applied.
func (s *PostService) CreatePostsFromBackup(posts []models.PostBackup) error {
	return s.repo.GetDB().Transaction(func(tx *gorm.DB) error {
		txRepo := repository.NewPostRepository(tx)

		for _, p := range posts {
			// We need a new service with the transactional repo to ensure proper slug generation
			// within the transaction.
			txService := NewPostService(txRepo, s.settingService, s.aiService)

			// We don't want AI summary on import.
			_, err := txService.CreatePost(p.Title, p.Content, p.IsPrivate, false, p.PublishedAt)
			if err != nil {
				// If any post fails, the transaction will be rolled back.
				return fmt.Errorf("导入文章 '%s' 失败: %w", p.Title, err)
			}
		}
		return nil
	})
}
