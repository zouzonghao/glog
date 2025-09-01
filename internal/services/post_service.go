package services

import (
	"encoding/json"
	"fmt"
	"glog/internal/constants"
	"glog/internal/models"
	"glog/internal/repository"
	"glog/internal/utils"
	"html/template"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gosimple/slug"
)

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
		finalHtml := fmt.Sprintf("<blockquote class=\"post-summary\">%s</blockquote>%s", summaryHtml, bodyHtml)
		return finalHtml, nil
	}

	fullHtml, err := utils.RenderMarkdown(md)
	if err != nil {
		return "", fmt.Errorf("全文渲染失败: %w", err)
	}
	return string(fullHtml), nil
}

func (s *PostService) LockPost(postID uint) {
	postLocksMu.Lock()
	defer postLocksMu.Unlock()
	postLocks[postID] = true
}

func (s *PostService) UnlockPost(postID uint) {
	postLocksMu.Lock()
	defer postLocksMu.Unlock()
	delete(postLocks, postID)
}

func (s *PostService) CheckPostLock(postID uint) bool {
	postLocksMu.Lock()
	defer postLocksMu.Unlock()
	return postLocks[postID]
}

func (s *PostService) CreatePost(title, content string, isPrivate bool, aiSummary bool, publishedAt time.Time) (*models.Post, bool, error) {
	if title == "" {
		title = "未命名标题"
	}

	excerpt := utils.GenerateExcerpt(content, 150)

	slugStr, err := s.generateUniqueSlug(title, 0)
	if err != nil {
		return nil, false, err
	}

	htmlContent, err := s.processAndRenderContent(content)
	if err != nil {
		return nil, false, err
	}

	post := &models.Post{
		Title:       title,
		Slug:        slugStr,
		Content:     content,
		ContentHTML: htmlContent,
		Excerpt:     excerpt,
		IsPrivate:   isPrivate,
		PublishedAt: publishedAt,
	}

	err = s.repo.Create(post)
	if err != nil {
		return nil, false, err
	}

	aiTriggered := false
	if aiSummary {
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
					var newContent string
					if strings.Contains(post.Content, separator) {
						parts := strings.SplitN(post.Content, separator, 2)
						if len(strings.TrimSpace(parts[0])) == 0 {
							newContent = fmt.Sprintf("%s\n\n%s%s", aiResp.Summary, separator, parts[1])
							contentChanged = true
						}
					} else {
						newContent = fmt.Sprintf("%s\n\n%s\n\n%s", aiResp.Summary, separator, post.Content)
						contentChanged = true
					}

					if contentChanged {
						updateMap["content"] = newContent
						newHtmlContent, err := s.processAndRenderContent(newContent)
						if err == nil {
							updateMap["content_html"] = newHtmlContent
						}
					}
				}
				if aiResp.Title != "" && aiResp.Title != post.Title {
					updateMap["title"] = aiResp.Title
					newSlug, slugErr := s.generateUniqueSlug(aiResp.Title, post.ID)
					if slugErr == nil {
						updateMap["slug"] = newSlug
					}
				}

				if len(updateMap) > 0 {
					if err := s.repo.UpdateFields(post.ID, updateMap); err != nil {
						fmt.Printf("用 AI 生成的内容更新文章失败 for post ID %d: %v\n", post.ID, err)
					}
				}
			}()
		}
	}

	return post, aiTriggered, nil
}

func (s *PostService) UpdatePost(id uint, title, content string, isPrivate bool, aiSummary bool, publishedAt time.Time) (*models.Post, bool, error) {
	if strings.TrimSpace(content) == "" {
		return nil, false, s.DeletePost(id)
	}
	post, err := s.repo.FindByID(id)
	if err != nil {
		return nil, false, err
	}

	if title == "" {
		title = "未命名标题"
	}

	htmlContent, err := s.processAndRenderContent(content)
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
	post.Content = content
	post.ContentHTML = htmlContent
	post.Excerpt = utils.GenerateExcerpt(content, 150)
	post.IsPrivate = isPrivate
	post.PublishedAt = publishedAt

	err = s.repo.Update(post)
	if err != nil {
		return nil, false, err
	}

	aiTriggered := false
	if aiSummary {
		separator := "<!--more-->"
		if !strings.Contains(content, separator) || len(strings.TrimSpace(strings.SplitN(content, separator, 2)[0])) == 0 {
			aiTriggered = true
			s.LockPost(post.ID)
			go func() {
				defer s.UnlockPost(post.ID)
				settings, err := s.settingService.GetAllSettings()
				if err != nil {
					return
				}
				baseURL := settings[constants.SettingOpenAIBaseURL]
				token := settings[constants.SettingOpenAIToken]
				model := settings[constants.SettingOpenAIModel]

				aiResp, err := s.aiService.GenerateSummaryAndTitle(post.Content, title == "未命名标题", baseURL, token, model)
				if err != nil {
					return
				}

				updateMap := make(map[string]interface{})
				contentChanged := false

				if aiResp.Summary != "" {
					updateMap["excerpt"] = aiResp.Summary
					var newContent string
					if strings.Contains(post.Content, separator) {
						parts := strings.SplitN(post.Content, separator, 2)
						if len(strings.TrimSpace(parts[0])) == 0 {
							newContent = fmt.Sprintf("%s\n\n%s%s", aiResp.Summary, separator, parts[1])
							contentChanged = true
						}
					} else {
						newContent = fmt.Sprintf("%s\n\n%s\n\n%s", aiResp.Summary, separator, post.Content)
						contentChanged = true
					}

					if contentChanged {
						updateMap["content"] = newContent
						newHtmlContent, err := s.processAndRenderContent(newContent)
						if err == nil {
							updateMap["content_html"] = newHtmlContent
						}
					}
				}
				if aiResp.Title != "" && aiResp.Title != post.Title {
					updateMap["title"] = aiResp.Title
					newSlug, slugErr := s.generateUniqueSlug(aiResp.Title, post.ID)
					if slugErr == nil {
						updateMap["slug"] = newSlug
					}
				}

				if len(updateMap) > 0 {
					if err := s.repo.UpdateFields(post.ID, updateMap); err != nil {
						fmt.Printf("用 AI 生成的内容更新文章失败 for post ID %d: %v\n", post.ID, err)
					}
				}
			}()
		}
	}

	return post, aiTriggered, nil
}

func (s *PostService) DeletePost(id uint) error {
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
	posts, err := s.repo.SearchPageByLike(query, page, pageSize, isLoggedIn)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountByQueryByLike(query, isLoggedIn)
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
	if post.ContentHTML == "" && post.Content != "" {
		html, err := s.processAndRenderContent(post.Content)
		if err != nil {
			fmt.Printf("按需渲染 Markdown 失败 for post ID %d: %v\n", post.ID, err)
		} else {
			post.ContentHTML = html
		}
	}

	renderedPost := &models.RenderedPost{
		ID:          post.ID,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
		PublishedAt: post.PublishedAt,
		Title:       post.Title,
		Slug:        post.Slug,
		Body:        template.HTML(post.ContentHTML),
		Excerpt:     post.Excerpt,
		IsPrivate:   post.IsPrivate,
	}
	return renderedPost, nil
}

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

func (s *PostService) CreatePostsFromBackup(posts []models.PostBackup) error {
	newPosts := make([]models.Post, 0, len(posts))
	for _, p := range posts {
		slugStr, err := s.generateUniqueSlug(p.Title, 0)
		if err != nil {
			return fmt.Errorf("为导入的文章 '%s' 生成 slug 失败: %w", p.Title, err)
		}
		htmlContent, err := s.processAndRenderContent(p.Content)
		if err != nil {
			return fmt.Errorf("为导入的文章 '%s' 渲染 HTML 失败: %w", p.Title, err)
		}
		newPosts = append(newPosts, models.Post{
			Title:       p.Title,
			Slug:        slugStr,
			Content:     p.Content,
			ContentHTML: htmlContent,
			IsPrivate:   p.IsPrivate,
			PublishedAt: p.PublishedAt,
			Excerpt:     utils.GenerateExcerpt(p.Content, 150),
		})
	}

	if err := s.repo.CreateBatchFromBackup(newPosts); err != nil {
		return fmt.Errorf("批量导入文章失败: %w", err)
	}

	return nil
}

func (s *PostService) CreatePostsFromBackupStream(backupReader io.Reader) (int, error) {
	var backupData models.SiteBackup
	if err := json.NewDecoder(backupReader).Decode(&backupData); err != nil {
		return 0, fmt.Errorf("解析备份 JSON 数据失败: %w", err)
	}

	if err := s.CreatePostsFromBackup(backupData.Posts); err != nil {
		return 0, err
	}

	if len(backupData.Settings) > 0 {
		if newPass, ok := backupData.Settings[constants.SettingPassword]; !ok || newPass == "" {
			delete(backupData.Settings, constants.SettingPassword)
		}
		if err := s.settingService.UpdateSettings(backupData.Settings); err != nil {
			return 0, fmt.Errorf("恢复设置失败: %w", err)
		}
	}

	return len(backupData.Posts), nil
}

func (s *PostService) BatchUpdatePosts(ids []uint, action string, isPrivate bool) error {
	switch action {
	case "delete":
		return s.repo.DeleteByIDs(ids)
	case "set-private":
		return s.repo.UpdatePrivacyByIDs(ids, isPrivate)
	default:
		return fmt.Errorf("不支持的操作: %s", action)
	}
}
