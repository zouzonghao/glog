package services

import (
	"fmt"
	"glog/internal/constants"
	"glog/internal/models"
	"glog/internal/repository"
	"glog/internal/utils"
	"glog/internal/utils/segmenter"
	"html/template"
	"regexp"
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

// processAndRenderContent handles the logic of splitting content by "<!--more-->"
// and rendering it into the final HTML structure.
func (s *PostService) processAndRenderContent(md string) (string, error) {
	separatorRegex := regexp.MustCompile(`<!--\s*more\s*-->`)
	parts := separatorRegex.Split(md, 2)

	if len(parts) > 1 {
		summaryMd := parts[0]
		bodyMd := parts[1]

		summaryHtml, err := utils.RenderMarkdown(summaryMd)
		if err != nil {
			return "", fmt.Errorf("摘要渲染失败: %w", err)
		}

		bodyHtml, err := utils.RenderMarkdown(bodyMd)
		if err != nil {
			return "", fmt.Errorf("正文渲染失败: %w", err)
		}

		// Combine summary (wrapped in blockquote) and body.
		// The summaryHtml can contain its own <p> tags, which is valid inside a blockquote.
		finalHtml := fmt.Sprintf("<blockquote class=\"post-summary\">%s</blockquote>%s", summaryHtml, bodyHtml)
		return finalHtml, nil
	}

	// No separator, just render the whole content
	fullHtml, err := utils.RenderMarkdown(md)
	if err != nil {
		return "", fmt.Errorf("全文渲染失败: %w", err)
	}
	return string(fullHtml), nil
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

func (s *PostService) CreatePost(title, content string, isPrivate bool, aiSummary bool, publishedAt time.Time) (*models.Post, bool, error) {
	if title == "" {
		title = "未命名标题"
	}

	sanitizedContent := content
	excerpt := utils.GenerateExcerpt(sanitizedContent, 150)

	slugStr, err := s.generateUniqueSlug(title, 0)
	if err != nil {
		return nil, false, err
	}

	htmlContent, err := s.processAndRenderContent(sanitizedContent)
	if err != nil {
		return nil, false, err
	}

	post := &models.Post{
		Title:       title,
		Slug:        slugStr,
		Content:     sanitizedContent,
		ContentHTML: htmlContent,
		Excerpt:     excerpt,
		IsPrivate:   isPrivate,
		PublishedAt: publishedAt,
	}

	err = s.repo.Create(post)
	if err != nil {
		return nil, false, err
	}

	// Update FTS index
	segmentedTitle := segmenter.SegmentTextForIndex(post.Title)
	segmentedContent := segmenter.SegmentTextForIndex(post.Content)
	err = s.repo.UpdateFtsIndex(post.ID, segmentedTitle, segmentedContent)
	if err != nil {
		fmt.Printf("更新 FTS 索引失败 for post ID %d: %v\n", post.ID, err)
	}

	aiTriggered := false
	if aiSummary {
		// 只有在没有手动摘要时才触发AI
		separator := "<!--more-->"
		if !strings.Contains(content, separator) || len(strings.TrimSpace(strings.SplitN(content, separator, 2)[0])) == 0 {
			aiTriggered = true
			s.LockPost(post.ID)
			go func() {
				defer s.UnlockPost(post.ID)
				settings, err := s.settingService.GetAllSettings()
				if err != nil {
					fmt.Printf("获取 AI 设置失败 for post ID %d: %v\n", post.ID, err)
					return
				}
				baseURL := settings[constants.SettingOpenAIBaseURL]
				token := settings[constants.SettingOpenAIToken]
				model := settings[constants.SettingOpenAIModel]

				aiResp, err := s.aiService.GenerateSummaryAndTitle(post.Content, title == "未命名标题", baseURL, token, model)
				if err != nil {
					fmt.Printf("AI 摘要生成失败 for post ID %d: %v\n", post.ID, err)
					return
				}

				updateMap := make(map[string]interface{})
				contentChanged := false

				if aiResp.Summary != "" {
					updateMap["excerpt"] = aiResp.Summary

					originalContent := post.Content
					summary := aiResp.Summary
					var newContent string

					if strings.Contains(originalContent, separator) {
						parts := strings.SplitN(originalContent, separator, 2)
						before := strings.TrimSpace(parts[0])
						if len(before) == 0 {
							newContent = fmt.Sprintf("%s\n\n%s%s", summary, separator, parts[1])
							contentChanged = true
						}
					} else {
						newContent = fmt.Sprintf("%s\n\n%s\n\n%s", summary, separator, originalContent)
						contentChanged = true
					}

					if contentChanged {
						updateMap["content"] = newContent
						newHtmlContent, err := s.processAndRenderContent(newContent)
						if err == nil {
							updateMap["content_html"] = newHtmlContent
						} else {
							fmt.Printf("为 AI 生成的内容重新渲染 HTML 失败 for post ID %d: %v\n", post.ID, err)
						}
					}
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

				logMessage := fmt.Sprintf("AI 日志 (Post ID: %d): ", post.ID)
				if aiResp.Title != "" && aiResp.Summary != "" {
					logMessage += fmt.Sprintf("生成了标题 '%s' 和摘要 '%.50s...'", aiResp.Title, aiResp.Summary)
				} else if aiResp.Summary != "" {
					logMessage += fmt.Sprintf("生成了摘要 '%.50s...'", aiResp.Summary)
				} else {
					logMessage += "AI 未返回有效内容。"
				}

				if contentChanged {
					logMessage += " -> 已更新正文。"
				} else if strings.Contains(post.Content, "<!--more-->") {
					logMessage += " -> 正文未更新 (用户已填写摘要)。"
				}
				fmt.Println(logMessage)

				if len(updateMap) > 0 {
					if err := s.repo.UpdateFields(post.ID, updateMap); err != nil {
						fmt.Printf("用 AI 生成的内容更新文章失败 for post ID %d: %v\n", post.ID, err)
					} else {
						if aiResp.Title != "" {
							segmentedTitle := segmenter.SegmentTextForIndex(aiResp.Title)
							s.repo.UpdateFtsIndex(post.ID, segmentedTitle, segmentedContent)
						}
					}
				}
			}()
		}
	}

	return post, aiTriggered, nil
}

func (s *PostService) UpdatePost(id uint, title, content string, isPrivate bool, aiSummary bool, publishedAt time.Time) (*models.Post, bool, error) {
	post, err := s.repo.FindByID(id)
	if err != nil {
		return nil, false, err
	}

	if title == "" {
		title = "未命名标题"
	}

	sanitizedContent := content
	htmlContent, err := s.processAndRenderContent(sanitizedContent)
	if err != nil {
		return nil, false, err
	}

	if post.Title != title {
		newSlug, err := s.generateUniqueSlug(title, id)
		if err != nil {
			return nil, false, err
		}
		post.Slug = newSlug
	}

	post.Title = title
	post.Content = sanitizedContent
	post.ContentHTML = htmlContent
	post.Excerpt = utils.GenerateExcerpt(sanitizedContent, 150)
	post.IsPrivate = isPrivate
	post.PublishedAt = publishedAt

	err = s.repo.Update(post)
	if err != nil {
		return nil, false, err
	}

	// Update FTS index
	segmentedTitle := segmenter.SegmentTextForIndex(post.Title)
	segmentedContent := segmenter.SegmentTextForIndex(post.Content)
	err = s.repo.UpdateFtsIndex(post.ID, segmentedTitle, segmentedContent)
	if err != nil {
		fmt.Printf("更新 FTS 索引失败 for post ID %d: %v\n", post.ID, err)
	}

	aiTriggered := false
	if aiSummary {
		// 只有在没有手动摘要时才触发AI
		separator := "<!--more-->"
		if !strings.Contains(content, separator) || len(strings.TrimSpace(strings.SplitN(content, separator, 2)[0])) == 0 {
			aiTriggered = true
			s.LockPost(post.ID)
			go func() {
				defer s.UnlockPost(post.ID)
				settings, err := s.settingService.GetAllSettings()
				if err != nil {
					fmt.Printf("获取 AI 设置失败 for post ID %d: %v\n", post.ID, err)
					return
				}
				baseURL := settings[constants.SettingOpenAIBaseURL]
				token := settings[constants.SettingOpenAIToken]
				model := settings[constants.SettingOpenAIModel]

				aiResp, err := s.aiService.GenerateSummaryAndTitle(post.Content, title == "未命名标题", baseURL, token, model)
				if err != nil {
					fmt.Printf("AI 摘要生成失败 for post ID %d: %v\n", post.ID, err)
					return
				}

				updateMap := make(map[string]interface{})
				contentChanged := false

				if aiResp.Summary != "" {
					updateMap["excerpt"] = aiResp.Summary

					originalContent := post.Content
					summary := aiResp.Summary
					var newContent string

					if strings.Contains(originalContent, separator) {
						parts := strings.SplitN(originalContent, separator, 2)
						before := strings.TrimSpace(parts[0])
						if len(before) == 0 {
							newContent = fmt.Sprintf("%s\n\n%s%s", summary, separator, parts[1])
							contentChanged = true
						}
					} else {
						newContent = fmt.Sprintf("%s\n\n%s\n\n%s", summary, separator, originalContent)
						contentChanged = true
					}

					if contentChanged {
						updateMap["content"] = newContent
						newHtmlContent, err := s.processAndRenderContent(newContent)
						if err == nil {
							updateMap["content_html"] = newHtmlContent
						} else {
							fmt.Printf("为 AI 生成的内容重新渲染 HTML 失败 for post ID %d: %v\n", post.ID, err)
						}
					}
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

				logMessage := fmt.Sprintf("AI 日志 (Post ID: %d): ", post.ID)
				if aiResp.Title != "" && aiResp.Summary != "" {
					logMessage += fmt.Sprintf("生成了标题 '%s' 和摘要 '%.50s...'", aiResp.Title, aiResp.Summary)
				} else if aiResp.Summary != "" {
					logMessage += fmt.Sprintf("生成了摘要 '%.50s...'", aiResp.Summary)
				} else {
					logMessage += "AI 未返回有效内容。"
				}

				if contentChanged {
					logMessage += " -> 已更新正文。"
				} else if strings.Contains(post.Content, "<!--more-->") {
					logMessage += " -> 正文未更新 (用户已填写摘要)。"
				}
				fmt.Println(logMessage)

				if len(updateMap) > 0 {
					if err := s.repo.UpdateFields(post.ID, updateMap); err != nil {
						fmt.Printf("用 AI 生成的内容更新文章失败 for post ID %d: %v\n", post.ID, err)
					} else {
						if aiResp.Title != "" {
							segmentedTitle := segmenter.SegmentTextForIndex(aiResp.Title)
							s.repo.UpdateFtsIndex(post.ID, segmentedTitle, segmentedContent)
						}
					}
				}
			}()
		}
	}

	return post, aiTriggered, nil
}

func (s *PostService) DeletePost(id uint) error {
	err := s.repo.DeleteFtsIndex(id)
	if err != nil {
		fmt.Printf("删除 FTS 索引失败 for post ID %d: %v\n", id, err)
	}
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
	return s.renderPost(post)
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
		renderedPost, err := s.renderPost(&post)
		if err != nil {
			return nil, 0, fmt.Errorf("渲染文章失败 ID %d: %w", post.ID, err)
		}
		renderedPosts[i] = *renderedPost
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
	segmentedQuery := segmenter.SegmentTextForQuery(query)
	posts, err := s.repo.SearchPage(segmentedQuery, page, pageSize, isLoggedIn)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountByQuery(segmentedQuery, isLoggedIn)
	if err != nil {
		return nil, 0, err
	}

	renderedPosts := make([]models.RenderedPost, len(posts))
	for i, post := range posts {
		renderedPost, err := s.renderPost(&post)
		if err != nil {
			return nil, 0, fmt.Errorf("渲染文章失败 ID %d: %w", post.ID, err)
		}
		renderedPosts[i] = *renderedPost
	}

	return renderedPosts, int(total), nil
}

func (s *PostService) renderPost(post *models.Post) (*models.RenderedPost, error) {
	renderedPost := &models.RenderedPost{
		ID:          post.ID,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
		PublishedAt: post.PublishedAt,
		Title:       post.Title,
		Slug:        post.Slug,
		Summary:     "", // Summary is now part of the Body
		Body:        template.HTML(post.ContentHTML),
		Excerpt:     post.Excerpt,
		IsPrivate:   post.IsPrivate,
	}
	return renderedPost, nil
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
			txService := NewPostService(txRepo, s.settingService, s.aiService)
			_, _, err := txService.CreatePost(p.Title, p.Content, p.IsPrivate, false, p.PublishedAt)
			if err != nil {
				return fmt.Errorf("导入文章 '%s' 失败: %w", p.Title, err)
			}
		}
		return nil
	})
}

// BatchUpdatePosts performs a batch operation on multiple posts.
func (s *PostService) BatchUpdatePosts(ids []uint, action string, isPrivate bool) error {
	switch action {
	case "delete":
		// Also delete from FTS index
		for _, id := range ids {
			if err := s.repo.DeleteFtsIndex(id); err != nil {
				// Log the error but continue trying to delete other posts
				fmt.Printf("删除 FTS 索引失败 for post ID %d: %v\n", id, err)
			}
		}
		return s.repo.DeleteByIDs(ids)
	case "set-private":
		return s.repo.UpdatePrivacyByIDs(ids, isPrivate)
	default:
		return fmt.Errorf("不支持的操作: %s", action)
	}
}
