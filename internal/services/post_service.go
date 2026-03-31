package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"glog/internal/constants"
	"glog/internal/models"
	"glog/internal/repository"
	"glog/internal/utils"
	"html/template"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gosimple/slug"
	"gorm.io/gorm"
)

const maxTagLength = 20

var (
	tagSplitRegex  = regexp.MustCompile(`[,，、;\s]+`)
	tagCleanRegex  = regexp.MustCompile(`[^\p{Han}\w\-\+\#\.]+`)
	separatorRegex = regexp.MustCompile(`<!--\s*more\s*-->`)
)

func normalizeTags(input string) string {
	if input == "" {
		return ""
	}
	parts := tagSplitRegex.Split(input, -1)
	var result []string
	for _, part := range parts {
		normalized := normalizeSingleTag(part)
		if normalized != "" {
			result = append(result, normalized)
		}
	}
	return strings.Join(result, ",")
}

func normalizeSingleTag(tag string) string {
	tag = strings.TrimSpace(tag)
	tag = strings.ToLower(tag)
	tag = tagCleanRegex.ReplaceAllString(tag, "")
	if utf8.RuneCountInString(tag) > maxTagLength {
		tag = string([]rune(tag)[:maxTagLength])
	}
	return tag
}

var keywordSplitRegex = regexp.MustCompile(`[\s,，]+`)

func extractKeywords(query string) []string {
	keywords := keywordSplitRegex.Split(strings.TrimSpace(query), -1)
	var cleaned []string
	for _, kw := range keywords {
		if kw != "" {
			cleaned = append(cleaned, kw)
		}
	}
	return cleaned
}

type PostService struct {
	repo           *repository.PostRepository
	settingService *SettingService
	tagsCache      *tagsCache
}

type tagsCache struct {
	mu       sync.RWMutex
	data     []string
	expireAt time.Time
	maxAge   time.Duration
	maxSize  int
}

func newTagsCache(maxAge time.Duration, maxSize int) *tagsCache {
	return &tagsCache{
		maxAge:  maxAge,
		maxSize: maxSize,
	}
}

func (c *tagsCache) Get() ([]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.data == nil || time.Now().After(c.expireAt) {
		return nil, false
	}
	return c.data, true
}

func (c *tagsCache) Set(tags []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(tags) > c.maxSize {
		tags = append([]string(nil), tags[:c.maxSize]...)
	}
	c.data = tags
	c.expireAt = time.Now().Add(c.maxAge)
}

func (c *tagsCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = nil
	c.expireAt = time.Time{}
}

func NewPostService(repo *repository.PostRepository, settingService *SettingService) *PostService {
	return &PostService{
		repo:           repo,
		settingService: settingService,
		tagsCache:      newTagsCache(30*time.Minute, 1000),
	}
}

func (s *PostService) TouchDBModified() {
	s.settingService.UpdateSettings(map[string]string{
		constants.SettingDBModifiedAt: strconv.FormatInt(time.Now().UnixNano(), 10),
	})
}

func (s *PostService) processAndRenderContent(md string) (string, error) {
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

func (s *PostService) CreatePost(title, content, tag string, isPrivate bool, publishedAt time.Time) (*models.Post, error) {
	if title == "" {
		title = "未命名标题"
	}

	tag = normalizeTags(tag)

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
		Tag:         tag,
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
	s.tagsCache.Clear()
	s.TouchDBModified()

	return post, nil
}

func (s *PostService) UpdatePost(id uint, title, content, tag string, isPrivate bool, publishedAt time.Time) (*models.Post, error) {
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

	tag = normalizeTags(tag)

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
	post.Tag = tag
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
	s.tagsCache.Clear()
	s.TouchDBModified()

	return post, nil
}

func (s *PostService) DeletePost(id uint) error {
	err := s.repo.Delete(id)
	if err != nil {
		return err
	}
	s.tagsCache.Clear()
	s.TouchDBModified()
	return nil
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

	renderedPosts, err := s.renderPosts(posts)
	if err != nil {
		return nil, 0, err
	}
	return renderedPosts, int(total), nil
}

func (s *PostService) UpsertPostFromBackup(p models.PostBackup) error {
	existingPost, err := s.repo.FindBySlugIgnorePrivacy(p.Slug)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("查询文章失败: %w", err)
		}

		htmlContent, renderErr := s.processAndRenderContent(p.Content)
		if renderErr != nil {
			return fmt.Errorf("渲染 HTML 失败: %w", renderErr)
		}

		cover := p.Cover
		if cover == "" {
			cover = utils.ExtractFirstImageURL(p.Content)
		}

		newPost := &models.Post{
			Title:       p.Title,
			Slug:        p.Slug,
			Tag:         normalizeTags(p.Tag),
			Content:     p.Content,
			ContentHTML: htmlContent,
			IsPrivate:   p.IsPrivate,
			PublishedAt: p.PublishedAt,
			Excerpt:     utils.GenerateExcerpt(p.Content, 150),
			Cover:       cover,
		}

		if err := s.repo.Create(newPost); err != nil {
			return fmt.Errorf("创建文章失败: %w", err)
		}

		s.tagsCache.Clear()
		return nil
	}

	htmlContent, renderErr := s.processAndRenderContent(p.Content)
	if renderErr != nil {
		return fmt.Errorf("渲染 HTML 失败: %w", renderErr)
	}

	cover := p.Cover
	if cover == "" {
		cover = utils.ExtractFirstImageURL(p.Content)
	}

	existingPost.Title = p.Title
	existingPost.Tag = normalizeTags(p.Tag)
	existingPost.Content = p.Content
	existingPost.ContentHTML = htmlContent
	existingPost.IsPrivate = p.IsPrivate
	existingPost.PublishedAt = p.PublishedAt
	existingPost.Excerpt = utils.GenerateExcerpt(p.Content, 150)
	existingPost.Cover = cover

	if err := s.repo.Update(existingPost); err != nil {
		return fmt.Errorf("更新文章失败: %w", err)
	}

	s.tagsCache.Clear()
	return nil
}

func (s *PostService) UpsertPostsFromBackup(posts []models.PostBackup) error {
	if len(posts) == 0 {
		return nil
	}

	processedPosts := make([]models.Post, 0, len(posts))
	for _, p := range posts {
		htmlContent, err := s.processAndRenderContent(p.Content)
		if err != nil {
			return fmt.Errorf("渲染文章 '%s' HTML 失败: %w", p.Title, err)
		}

		cover := p.Cover
		if cover == "" {
			cover = utils.ExtractFirstImageURL(p.Content)
		}

		processedPosts = append(processedPosts, models.Post{
			Title:       p.Title,
			Slug:        p.Slug,
			Tag:         normalizeTags(p.Tag),
			Content:     p.Content,
			ContentHTML: htmlContent,
			IsPrivate:   p.IsPrivate,
			PublishedAt: p.PublishedAt,
			Excerpt:     utils.GenerateExcerpt(p.Content, 150),
			Cover:       cover,
		})
	}

	if err := s.repo.UpsertAllPosts(processedPosts); err != nil {
		return fmt.Errorf("批量导入文章失败: %w", err)
	}

	s.tagsCache.Clear()
	return nil
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
	cleanedKeywords := extractKeywords(query)

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

	renderedPosts, err := s.renderPosts(posts)
	if err != nil {
		return nil, 0, err
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
			go s.repo.UpdateFields(post.ID, map[string]interface{}{"content_html": html})
		}
	}

	renderedPost := &models.RenderedPost{
		ID:          post.ID,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
		PublishedAt: post.PublishedAt,
		Title:       post.Title,
		Slug:        post.Slug,
		Tag:         post.Tag,
		Cover:       post.Cover,
		Body:        template.HTML(post.ContentHTML),
		Excerpt:     post.Excerpt,
		IsPrivate:   post.IsPrivate,
	}
	return renderedPost, nil
}

func (s *PostService) renderPosts(posts []models.Post) ([]models.RenderedPost, error) {
	rendered := make([]models.RenderedPost, len(posts))
	for i, post := range posts {
		r, err := s.renderPost(&post)
		if err != nil {
			return nil, fmt.Errorf("渲染文章失败 ID %d: %w", post.ID, err)
		}
		rendered[i] = *r
	}
	s.applyCoverPrefix(rendered)
	return rendered, nil
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
			Slug:        p.Slug,
			Title:       p.Title,
			Tag:         p.Tag,
			Cover:       p.Cover,
			Content:     p.Content,
			IsPrivate:   p.IsPrivate,
			PublishedAt: p.PublishedAt,
		}
	}
	return backupPosts, nil
}

func (s *PostService) CreatePostsFromBackupStream(backupReader io.Reader) (int, error) {
	var backupData models.SiteBackup
	if err := json.NewDecoder(backupReader).Decode(&backupData); err != nil {
		return 0, fmt.Errorf("解析备份 JSON 数据失败: %w", err)
	}

	return s.RestoreFromBackupData(&backupData)
}

func (s *PostService) RestoreFromBackupData(backupData *models.SiteBackup) (int, error) {
	newPosts := make([]models.Post, 0, len(backupData.Posts))
	usedSlugs := make(map[string]bool)
	for _, p := range backupData.Posts {
		slugStr, err := s.generateUniqueSlugWithUsed(p.Title, 0, usedSlugs)
		if err != nil {
			return 0, fmt.Errorf("为导入的文章 '%s' 生成 slug 失败: %w", p.Title, err)
		}
		usedSlugs[slugStr] = true

		htmlContent, err := s.processAndRenderContent(p.Content)
		if err != nil {
			return 0, fmt.Errorf("为导入的文章 '%s' 渲染 HTML 失败: %w", p.Title, err)
		}
		cover := p.Cover
		if cover == "" {
			cover = utils.ExtractFirstImageURL(p.Content)
		}
		newPosts = append(newPosts, models.Post{
			Title:       p.Title,
			Slug:        slugStr,
			Tag:         normalizeTags(p.Tag),
			Content:     p.Content,
			ContentHTML: htmlContent,
			IsPrivate:   p.IsPrivate,
			PublishedAt: p.PublishedAt,
			Excerpt:     utils.GenerateExcerpt(p.Content, 150),
			Cover:       cover,
		})
	}

	if err := s.repo.ReplaceAllPosts(newPosts); err != nil {
		return 0, fmt.Errorf("导入文章失败: %w", err)
	}

	s.tagsCache.Clear()

	if len(backupData.Settings) > 0 {
		settings := backupData.Settings
		if newPass, ok := settings[constants.SettingPassword]; !ok || newPass == "" {
			delete(settings, constants.SettingPassword)
		}
		delete(settings, constants.SettingDBModifiedAt)
		if err := s.settingService.UpdateSettings(settings); err != nil {
			return len(newPosts), fmt.Errorf("文章已恢复，但设置恢复失败: %w", err)
		}
	}

	return len(newPosts), nil
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

func (s *PostService) GetPostsPageByTag(tag string, page, pageSize int, isLoggedIn bool) ([]models.RenderedPost, int, error) {
	tag = normalizeSingleTag(tag)
	if tag == "" {
		return nil, 0, nil
	}
	posts, err := s.repo.FindPageByTag(tag, page, pageSize, isLoggedIn)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountByTag(tag, isLoggedIn)
	if err != nil {
		return nil, 0, err
	}

	renderedPosts, err := s.renderPosts(posts)
	if err != nil {
		return nil, 0, err
	}
	return renderedPosts, int(total), nil
}

func (s *PostService) GetAllTags(isLoggedIn bool) ([]string, error) {
	if !isLoggedIn {
		if tags, ok := s.tagsCache.Get(); ok {
			return tags, nil
		}
	}
	tags, err := s.repo.GetAllTags(isLoggedIn)
	if err != nil {
		return nil, err
	}
	if !isLoggedIn {
		s.tagsCache.Set(tags)
	}
	return tags, nil
}

func (s *PostService) SearchPostsPageByTag(query string, tag string, page, pageSize int, isLoggedIn bool) ([]models.RenderedPost, int, error) {
	tag = normalizeSingleTag(tag)
	if tag == "" {
		return nil, 0, nil
	}
	cleanedKeywords := extractKeywords(query)

	if len(cleanedKeywords) == 0 {
		return s.GetPostsPageByTag(tag, page, pageSize, isLoggedIn)
	}

	posts, err := s.repo.SearchPageByLikeAndTag(cleanedKeywords, tag, page, pageSize, isLoggedIn)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountByQueryByLikeAndTag(cleanedKeywords, tag, isLoggedIn)
	if err != nil {
		return nil, 0, err
	}

	renderedPosts, err := s.renderPosts(posts)
	if err != nil {
		return nil, 0, err
	}
	return renderedPosts, int(total), nil
}
