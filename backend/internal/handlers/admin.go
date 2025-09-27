package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"glog/internal/constants"
	"glog/internal/models"
	"glog/internal/services"
	"glog/internal/tasks"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yeka/zip"
)

type AdminHandler struct {
	postService    *services.PostService
	settingService *services.SettingService
	aiService      *services.AIService
	backupService  *services.BackupService
	scheduler      *tasks.Scheduler
}

func NewAdminHandler(postService *services.PostService, settingService *services.SettingService, aiService *services.AIService, backupService *services.BackupService, scheduler *tasks.Scheduler) *AdminHandler {
	return &AdminHandler{
		postService:    postService,
		settingService: settingService,
		aiService:      aiService,
		backupService:  backupService,
		scheduler:      scheduler,
	}
}

func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	settingsToUpdate := make(map[string]string)
	if err := c.ShouldBindJSON(&settingsToUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的数据格式: " + err.Error()})
		return
	}

	// Filter out empty sensitive fields to avoid overwriting them
	for key, value := range settingsToUpdate {
		if (key == constants.SettingPassword || key == constants.SettingOpenAIToken || key == constants.SettingGithubToken || key == constants.SettingWebdavPassword || key == constants.SettingImageAPIToken) && value == "" {
			delete(settingsToUpdate, key)
		}
	}

	err := h.settingService.UpdateSettings(settingsToUpdate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "更新设置失败"})
		return
	}

	go h.scheduler.ReloadTasks()

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "设置已成功保存！"})
}

func (h *AdminHandler) ListPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if pageSize <= 0 {
		pageSize = 10
	}
	query := c.Query("q")
	status := c.DefaultQuery("status", "all")

	posts, total, err := h.postService.GetPostsPageByAdmin(page, pageSize, query, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "加载文章失败"})
		return
	}

	// --- Start: Define structured and correctly cased JSON responses ---
	type PostResponseForAdmin struct {
		ID          uint   `json:"id"`
		PublishedAt string `json:"published_at"`
		Title       string `json:"title"`
		Slug        string `json:"slug"`
		Cover       string `json:"cover"`
		Excerpt     string `json:"excerpt"`
		IsPrivate   bool   `json:"is_private"`
	}

	type PaginationResponse struct {
		CurrentPage  int  `json:"currentPage"`
		TotalPages   int  `json:"totalPages"`
		TotalRecords int  `json:"totalRecords"`
		PageSize     int  `json:"pageSize"`
		HasPrev      bool `json:"hasPrev"`
		HasNext      bool `json:"hasNext"`
	}

	type AdminPostsResponse struct {
		Posts      []PostResponseForAdmin `json:"posts"`
		Pagination PaginationResponse     `json:"pagination"`
	}
	// --- End: Define structured JSON responses ---

	postResponses := make([]PostResponseForAdmin, len(posts))
	for i, post := range posts {
		var publishedAtStr string
		if !post.PublishedAt.IsZero() {
			publishedAtStr = post.PublishedAt.Format(time.RFC3339)
		}
		postResponses[i] = PostResponseForAdmin{
			ID:          post.ID,
			PublishedAt: publishedAtStr,
			Title:       post.Title,
			Slug:        post.Slug,
			Cover:       post.Cover,
			Excerpt:     post.Excerpt,
			IsPrivate:   post.IsPrivate,
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	response := AdminPostsResponse{
		Posts: postResponses,
		Pagination: PaginationResponse{
			CurrentPage:  page,
			TotalPages:   totalPages,
			TotalRecords: int(total),
			PageSize:     pageSize,
			HasPrev:      page > 1,
			HasNext:      page < totalPages,
		},
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) SavePost(c *gin.Context) {
	var req struct {
		ID            uint   `json:"id"`
		Title         string `json:"title"`
		Content       string `json:"content"`
		PublishedAt   string `json:"published_at"`
		IsPrivate     bool   `json:"is_private"`
		AISummary     bool   `json:"ai_summary"`
		AICover       bool   `json:"ai_cover"`
		AICoverPrompt string `json:"ai_cover_prompt"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的数据格式: " + err.Error()})
		return
	}

	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "服务器时间配置错误"})
		return
	}
	publishedAt, err := time.ParseInLocation("2006-01-02 15:04", req.PublishedAt, loc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的发布时间格式"})
		return
	}

	if req.ID != 0 {
		if h.postService.CheckPostLock(req.ID) {
			c.JSON(http.StatusConflict, gin.H{
				"status":  "locked",
				"message": "正在生成AI内容，文章已锁定，请稍候再试...",
			})
			return
		}
	}

	var post *models.Post
	var aiTriggered bool

	if req.ID == 0 {
		post, aiTriggered, err = h.postService.CreatePost(req.Title, req.Content, req.IsPrivate, req.AISummary, req.AICover, req.AICoverPrompt, publishedAt)
	} else {
		post, aiTriggered, err = h.postService.UpdatePost(req.ID, req.Title, req.Content, req.IsPrivate, req.AISummary, req.AICover, req.AICoverPrompt, publishedAt)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "保存文章失败: " + err.Error(),
		})
		return
	}

	if post == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "deleted",
			"message": "文章内容为空，已自动删除。",
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "保存文章失败: " + err.Error(),
		})
		return
	}

	message := "文章已保存！"
	if aiTriggered {
		message = "文章已保存，AI 内容正在生成中，请稍后刷新查看..."
	}

	response := gin.H{
		"status":  "success",
		"message": message,
		"post_id": post.ID,
	}

	if !(aiTriggered && req.Title == "未命名标题") {
		response["slug"] = post.Slug
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) GetPostByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的文章 ID"})
		return
	}

	post, err := h.postService.GetPostByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "文章未找到"})
		return
	}

	postResponse := models.PostDetailResponse{
		ID:          post.ID,
		PublishedAt: post.PublishedAt,
		Title:       post.Title,
		Slug:        post.Slug,
		Cover:       post.Cover,
		Content:     post.Content,
		IsPrivate:   post.IsPrivate,
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "post": postResponse})
}

func (h *AdminHandler) DeletePost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的文章 ID"})
		return
	}

	err = h.postService.DeletePost(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "删除文章失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "文章已成功删除"})
}

func (h *AdminHandler) GetSettings(c *gin.Context) {
	settings, err := h.settingService.GetAllSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "无法加载设置"})
		return
	}
	// 密码等敏感信息不返回
	delete(settings, constants.SettingPassword)
	delete(settings, constants.SettingOpenAIToken)
	delete(settings, constants.SettingGithubToken)
	delete(settings, constants.SettingWebdavPassword)
	delete(settings, constants.SettingImageAPIToken)

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"settings": settings,
	})
}

func (h *AdminHandler) TestAISettings(c *gin.Context) {
	baseURL := c.PostForm(constants.SettingOpenAIBaseURL)
	token := c.PostForm(constants.SettingOpenAIToken)
	model := c.PostForm(constants.SettingOpenAIModel)

	finalToken := token
	if finalToken == "" {
		settings, err := h.settingService.GetAllSettings()
		if err == nil {
			finalToken = settings[constants.SettingOpenAIToken]
		}
	}

	testContent := "这是一个用于测试AI摘要功能的文本。"
	_, err := h.aiService.GenerateSummaryAndTitle(testContent, false, baseURL, finalToken, model)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "error", "message": "测试失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "测试成功！连接和配置均有效。"})
}

func (h *AdminHandler) TestImageAPIHandler(c *gin.Context) {
	apiURL := c.PostForm(constants.SettingImageAPIURL)
	apiToken := c.PostForm(constants.SettingImageAPIToken)

	if apiToken == "" {
		settings, err := h.settingService.GetAllSettings()
		if err == nil {
			apiToken = settings[constants.SettingImageAPIToken]
		}
	}

	models, err := h.aiService.TestImageAPI(apiURL, apiToken)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "error", "message": "连接失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "连接成功！",
		"models":  models,
	})
}

func (h *AdminHandler) BackupSite(c *gin.Context) {
	password, err := h.settingService.GetSetting(constants.SettingPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "获取站点密码失败: " + err.Error()})
		return
	}
	if password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "请先设置站点密码，备份文件需要加密。"})
		return
	}

	posts, err := h.postService.GetAllPostsForBackup()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "获取文章失败: " + err.Error()})
		return
	}

	settings, err := h.settingService.GetAllSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "获取设置失败: " + err.Error()})
		return
	}

	backupData := models.SiteBackup{
		Posts:    posts,
		Settings: settings,
	}

	jsonData, err := json.MarshalIndent(backupData, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "JSON 序列化失败: " + err.Error()})
		return
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)
	zipFile, err := zipWriter.Encrypt("backup.json", password, zip.AES256Encryption)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "创建加密 ZIP 文件失败: " + err.Error()})
		return
	}
	_, err = zipFile.Write(jsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "写入 ZIP 文件失败: " + err.Error()})
		return
	}
	zipWriter.Close()

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=glog_backup_%s.zip", time.Now().Format("20060102150405")))
	c.Data(http.StatusOK, "application/zip", buf.Bytes())
}

func (h *AdminHandler) UploadBackup(c *gin.Context) {
	contentType := c.GetHeader("Content-Type")
	var backupData models.SiteBackup
	var postCount int

	if strings.Contains(contentType, "application/json") {
		if err := c.ShouldBindJSON(&backupData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "解析 JSON 数据失败: " + err.Error()})
			return
		}
		if err := h.restoreFromBackupData(&backupData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
			return
		}
		postCount = len(backupData.Posts)
	} else {
		password := c.PostForm("password")
		if password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "请输入备份文件密码。"})
			return
		}

		file, err := c.FormFile("backup")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "获取上传文件失败: " + err.Error()})
			return
		}

		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "打开上传文件失败: " + err.Error()})
			return
		}
		defer src.Close()

		tempFile, err := os.CreateTemp("", "glog-backup-*.zip")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "创建临时文件失败: " + err.Error()})
			return
		}
		defer os.Remove(tempFile.Name())

		_, err = io.Copy(tempFile, src)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "保存上传文件失败: " + err.Error()})
			return
		}

		zipReader, err := zip.OpenReader(tempFile.Name())
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的 ZIP 文件: " + err.Error()})
			return
		}
		defer zipReader.Close()

		if len(zipReader.File) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "空的 ZIP 文件。"})
			return
		}

		backupFile := zipReader.File[0]
		backupFile.SetPassword(password)

		jsonFile, err := backupFile.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "打开 backup.json 失败，请检查密码是否正确。"})
			return
		}
		defer jsonFile.Close()

		importedCount, err := h.postService.CreatePostsFromBackupStream(jsonFile)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
			return
		}
		postCount = importedCount
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": fmt.Sprintf("恢复成功！导入 %d 篇文章并更新了站点设置。", postCount)})
}

func (h *AdminHandler) restoreFromBackupData(backupData *models.SiteBackup) error {
	if len(backupData.Settings) > 0 {
		if newPass, ok := backupData.Settings[constants.SettingPassword]; !ok || newPass == "" {
			delete(backupData.Settings, constants.SettingPassword)
		}

		if err := h.settingService.UpdateSettings(backupData.Settings); err != nil {
			return fmt.Errorf("恢复设置失败: %w", err)
		}
	}

	if err := h.postService.CreatePostsFromBackup(backupData.Posts); err != nil {
		return fmt.Errorf("导入文章失败: %w", err)
	}

	return nil
}

type BatchUpdateRequest struct {
	IDs       []uint `json:"ids"`
	Action    string `json:"action"`
	IsPrivate bool   `json:"is_private"`
}

func (h *AdminHandler) BatchUpdatePosts(c *gin.Context) {
	var req BatchUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的请求数据: " + err.Error()})
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "请至少选择一篇文章"})
		return
	}

	err := h.postService.BatchUpdatePosts(req.IDs, req.Action, req.IsPrivate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "操作失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "操作成功！"})
}

func (h *AdminHandler) TestGithubSettings(c *gin.Context) {
	repo := c.PostForm(constants.SettingGithubRepo)
	token := c.PostForm(constants.SettingGithubToken)

	finalToken := token
	if finalToken == "" {
		settings, err := h.settingService.GetAllSettings()
		if err == nil {
			finalToken = settings[constants.SettingGithubToken]
		}
	}

	err := h.backupService.TestGithubConnection(repo, finalToken)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "error", "message": "GitHub 连接测试失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "GitHub 连接成功！"})
}

func (h *AdminHandler) TestWebdavSettings(c *gin.Context) {
	url := c.PostForm(constants.SettingWebdavURL)
	user := c.PostForm(constants.SettingWebdavUser)
	password := c.PostForm(constants.SettingWebdavPassword)

	finalPassword := password
	if finalPassword == "" {
		settings, err := h.settingService.GetAllSettings()
		if err == nil {
			finalPassword = settings[constants.SettingWebdavPassword]
		}
	}

	err := h.backupService.TestWebdavConnection(url, user, finalPassword)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "error", "message": "WebDAV 连接测试失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "WebDAV 连接成功！"})
}

func (h *AdminHandler) BackupToGithubNow(c *gin.Context) {
	settings, err := h.settingService.GetAllSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "获取设置失败: " + err.Error()})
		return
	}

	repo := settings[constants.SettingGithubRepo]
	branch := settings[constants.SettingGithubBranch]
	token := settings[constants.SettingGithubToken]

	if repo == "" || branch == "" || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "GitHub 备份配置不完整，请先保存设置。"})
		return
	}

	err = h.backupService.BackupToGithub(repo, branch, token)
	if err != nil {
		if errors.Is(err, services.ErrBackupNoChange) {
			c.JSON(http.StatusOK, gin.H{"status": "info", "message": "数据无变化，无需备份。"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "执行 GitHub 备份失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "已成功触发 GitHub 备份！"})
}

func (h *AdminHandler) BackupToWebdavNow(c *gin.Context) {
	settings, err := h.settingService.GetAllSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "获取设置失败: " + err.Error()})
		return
	}

	url := settings[constants.SettingWebdavURL]
	user := settings[constants.SettingWebdavUser]
	password := settings[constants.SettingWebdavPassword]

	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "WebDAV URL 未配置，请先保存设置。"})
		return
	}

	err = h.backupService.BackupToWebdav(url, user, password)
	if err != nil {
		if errors.Is(err, services.ErrBackupNoChange) {
			c.JSON(http.StatusOK, gin.H{"status": "info", "message": "数据无变化，无需备份。"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "执行 WebDAV 备份失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "已成功触发 WebDAV 备份！"})
}

func (h *AdminHandler) GetAILogs(c *gin.Context) {
	logs, err := os.ReadFile("ai.log")
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, gin.H{"status": "success", "logs": "暂无 AI 日志。"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "无法读取 AI 日志文件: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "logs": string(logs)})
}

func (h *AdminHandler) ClearAILogs(c *gin.Context) {
	err := os.Truncate("ai.log", 0)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, gin.H{"status": "success", "message": "日志文件不存在，无需清除。"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "无法清除 AI 日志文件: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "AI 日志已成功清除！"})
}

func (h *AdminHandler) GetPublicSettings(c *gin.Context) {
	settings, err := h.settingService.GetAllSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "无法加载设置"})
		return
	}

	publicSettings := map[string]string{
		"site_title":       settings["site_title"],
		"site_description": settings["site_description"],
		"favicon":          settings["favicon"],
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"settings": publicSettings,
	})
}
