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
	"time"

	"github.com/gosimple/slug"
)

type PostService struct {
	repo           *repository.PostRepository
	settingService *SettingService
}

func NewPostService(repo *repository.PostRepository, settingService *SettingService) *PostService {
	return &PostService{
		repo:           repo,
		settingService: settingService,
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

func (s *PostService) CreatePost(title, content string, isPrivate bool, publishedAt time.Time) (*models.Post, error) {
	if title == "" {
		title = "未命名标题"
	}

	excerpt := utils.GenerateExcerpt(content, 150)
	coverURL := utils.ExtractFirstImageURL(content)

	slugStr, err := s.generateUniqueSlug(title, 0)
	if err != nil {
		return nil, err
	}

	htmlContent, err := s.processAndRenderContent(content)
	if err != nil {
		return nil, err
	}

	post := &models.Post{
		Title:       title,
		Slug:        slugStr,
		Content:     content,
		ContentHTML: htmlContent,
		Excerpt:     excerpt,
		Cover:       coverURL,
		IsPrivate:   isPrivate,
		PublishedAt: publishedAt,
	}

	err = s.repo.Create(post)
	if err != nil {
		return nil, err
	}

	return post, nil
}

func (s *PostService) UpdatePost(id uint, title, content string, isPrivate bool, publishedAt time.Time) (*models.Post, error) {
	if strings.TrimSpace(content) == "" {
		return nil, s.DeletePost(id)
	}
	post, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if title == "" {
		title = "未命名标题"
	}

	htmlContent, err := s.processAndRenderContent(content)
	if err != nil {
		return nil, err
	}

	if post.Title != title {
		newSlug, err := s.generateUniqueSlug(title, id)
		if err != nil {
			return nil, err
		}
		post.Slug = newSlug
	}

	post.Title = title
	post.Content = content
	post.ContentHTML = htmlContent
	post.Excerpt = utils.GenerateExcerpt(content, 150)
	post.Cover = utils.ExtractFirstImageURL(content)
	post.IsPrivate = isPrivate
	post.PublishedAt = publishedAt

	err = s.repo.Update(post)
	if err != nil {
		return nil, err
	}

	return post, nil
}

func (s *PostService) DeletePost(id uint) error {
	return s.repo.Delete(id)
}

func (s *PostService) UpdateExcerptByID(id uint, excerpt string) error {
	return s.repo.UpdateFields(id, map[string]interface{}{
		"excerpt": excerpt,
	})
}

func (s *PostService) UpdateCoverByID(id uint, coverURL string) error {
	return s.repo.UpdateFields(id, map[string]interface{}{
		"cover": coverURL,
	})
}

func (s *PostService) GetPostByID(id uint) (*models.Post, error) {
	post, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (s *PostService) GetPostBySlug(slug string, isLoggedIn bool) (*models.RenderedPost, error) {
	post, err := s.repo.FindBySlug(slug, isLoggedIn)
	if err != nil {
		return nil, err
	}
	renderedPost, err := s.renderPost(post)
	if err != nil {
		return nil, err
	}
	s.applyCoverPrefix(renderedPost)
	return renderedPost, nil
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

	s.applyCoverPrefix(renderedPosts)
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
	re := regexp.MustCompile(`[\s,，]+`)
	keywords := re.Split(strings.TrimSpace(query), -1)
	var cleanedKeywords []string
	for _, keyword := range keywords {
		if keyword != "" {
			cleanedKeywords = append(cleanedKeywords, keyword)
		}
	}

	if len(cleanedKeywords) == 0 {
		return []models.RenderedPost{}, 0, nil
	}

	posts, err := s.repo.SearchPageByLike(cleanedKeywords, page, pageSize, isLoggedIn)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountByQueryByLike(cleanedKeywords, isLoggedIn)
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

	s.applyCoverPrefix(renderedPosts)
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
		Cover:       post.Cover,
		Body:        template.HTML(post.ContentHTML),
		Excerpt:     post.Excerpt,
		IsPrivate:   post.IsPrivate,
	}
	return renderedPost, nil
}

func (s *PostService) generateUniqueSlug(title string, postID uint) (string, error) {
	return s.generateUniqueSlugWithUsed(title, postID, nil)
}

func (s *PostService) generateUniqueSlugWithUsed(title string, postID uint, usedSlugs map[string]bool) (string, error) {
	baseSlug := slug.Make(title)
	if baseSlug == "" {
		baseSlug = "untitled"
	}
	finalSlug := baseSlug
	counter := 1
	for {
		var exists bool
		var err error

		// Check if it's in the provided map first (batch import case)
		if usedSlugs != nil && usedSlugs[finalSlug] {
			exists = true
		} else {
			// Then check the database
			if postID == 0 {
				exists, err = s.repo.CheckSlugExists(finalSlug)
			} else {
				exists, err = s.repo.CheckSlugExistsForOtherPost(finalSlug, postID)
			}
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
			Cover:       p.Cover,
			Content:     p.Content,
			IsPrivate:   p.IsPrivate,
			PublishedAt: p.PublishedAt,
		}
	}
	return backupPosts, nil
}

func (s *PostService) CreatePostsFromBackup(posts []models.PostBackup) error {
	newPosts := make([]models.Post, 0, len(posts))
	usedSlugs := make(map[string]bool)
	for _, p := range posts {
		slugStr, err := s.generateUniqueSlugWithUsed(p.Title, 0, usedSlugs)
		if err != nil {
			return fmt.Errorf("为导入的文章 '%s' 生成 slug 失败: %w", p.Title, err)
		}
		usedSlugs[slugStr] = true

		htmlContent, err := s.processAndRenderContent(p.Content)
		if err != nil {
			return fmt.Errorf("为导入的文章 '%s' 渲染 HTML 失败: %w", p.Title, err)
		}
		cover := p.Cover
		if cover == "" {
			cover = utils.ExtractFirstImageURL(p.Content)
		}
		newPosts = append(newPosts, models.Post{
			Title:       p.Title,
			Slug:        slugStr,
			Content:     p.Content,
			ContentHTML: htmlContent,
			IsPrivate:   p.IsPrivate,
			PublishedAt: p.PublishedAt,
			Excerpt:     utils.GenerateExcerpt(p.Content, 150),
			Cover:       cover,
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

func (s *PostService) applyCoverPrefix(data interface{}) {
	coverPrefix, err := s.settingService.GetSetting(constants.SettingCoverPrefix)
	if err != nil || coverPrefix == "" {
		return
	}

	switch v := data.(type) {
	case *models.RenderedPost:
		if v.Cover != "" && !strings.HasSuffix(v.Cover, ".avif") {
			v.Cover = coverPrefix + v.Cover
		}
	case []models.RenderedPost:
		for i := range v {
			if v[i].Cover != "" && !strings.HasSuffix(v[i].Cover, ".avif") {
				v[i].Cover = coverPrefix + v[i].Cover
			}
		}
	}
}
