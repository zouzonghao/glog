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
	"glog/internal/utils"
	"glog/internal/utils/segmenter"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
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

	if err := c.Request.ParseForm(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的表单数据"})
		return
	}

	for key, values := range c.Request.PostForm {
		if len(values) > 0 {
			value := values[0]
			if (key == constants.SettingPassword || key == constants.SettingOpenAIToken || key == constants.SettingGithubToken || key == constants.SettingWebdavPassword) && value == "" {
				continue
			}
			settingsToUpdate[key] = value
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
	query := c.Query("query")
	status := c.DefaultQuery("status", "all")

	searchQuery := query
	if searchQuery != "" {
		searchQuery = segmenter.SegmentTextForQuery(searchQuery)
	}
	posts, total, err := h.postService.GetPostsPageByAdmin(page, pageSize, searchQuery, status)
	if err != nil {
		c.String(http.StatusInternalServerError, "加载文章失败")
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	pagination := utils.GeneratePagination(page, totalPages)

	session := sessions.Default(c)
	flashes := session.Flashes(constants.SessionKeySuccessFlash)
	session.Save()

	render(c, http.StatusOK, "admin.html", gin.H{
		"posts":           posts,
		"Pagination":      pagination,
		"Query":           query,
		"Status":          status,
		"Flashes":         flashes,
		"PageSize":        pageSize,
		"PageSizeOptions": []int{10, 20, 50},
	})
}

func (h *AdminHandler) NewPost(c *gin.Context) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Now().In(loc).Format("2006-01-02 15:04")
	render(c, http.StatusOK, "editor.html", gin.H{
		"post": nil,
		"now":  now,
	})
}

func (h *AdminHandler) Editor(c *gin.Context) {
	idStr := c.Query("id")
	status := c.Query("status")

	if idStr == "" {
		c.Redirect(http.StatusFound, "/admin")
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin")
		return
	}

	post, err := h.postService.GetPostByID(uint(id))
	if err != nil {
		c.Redirect(http.StatusFound, "/admin")
		return
	}

	render(c, http.StatusOK, "editor.html", gin.H{
		"post":   post,
		"status": status,
	})
}

func (h *AdminHandler) SavePost(c *gin.Context) {
	idStr := c.PostForm("id")
	title := c.PostForm("title")
	content := c.PostForm("content")
	publishedAtStr := c.PostForm("published_at")
	isPrivate := c.PostForm("is_private") == "on"
	aiSummary := c.PostForm("ai_summary") == "on"

	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "服务器时间配置错误"})
		return
	}
	publishedAt, err := time.ParseInLocation("2006-01-02 15:04", publishedAtStr, loc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的发布时间格式"})
		return
	}

	if idStr != "" && idStr != "0" {
		id, _ := strconv.ParseUint(idStr, 10, 64)
		if h.postService.CheckPostLock(uint(id)) {
			c.JSON(http.StatusConflict, gin.H{
				"status":  "locked",
				"message": "正在生成AI摘要，文章已锁定，请稍候再试...",
			})
			return
		}
	}

	var post *models.Post
	var aiTriggered bool

	if idStr == "" || idStr == "0" {
		post, aiTriggered, err = h.postService.CreatePost(title, content, isPrivate, aiSummary, publishedAt)
	} else {
		id, _ := strconv.ParseUint(idStr, 10, 64)
		post, aiTriggered, err = h.postService.UpdatePost(uint(id), title, content, isPrivate, aiSummary, publishedAt)
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
	if aiTriggered && title == "未命名标题" {
		message = "文章已保存，AI正在生成标题和摘要，请稍后刷新查看..."
	} else if aiTriggered {
		message = "文章已保存，AI摘要正在生成中..."
	}

	response := gin.H{
		"status":  "success",
		"message": message,
		"post_id": post.ID,
	}

	if !(aiTriggered && title == "未命名标题") {
		response["slug"] = post.Slug
	}

	c.JSON(http.StatusOK, response)
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

func (h *AdminHandler) ShowSettingsPage(c *gin.Context) {
	render(c, http.StatusOK, "settings.html", gin.H{})
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

	if strings.Contains(contentType, "application/json") {
		if err := c.ShouldBindJSON(&backupData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "解析 JSON 数据失败: " + err.Error()})
			return
		}
	} else {
		file, err := c.FormFile("backup")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "获取上传文件失败: " + err.Error()})
			return
		}

		password := c.PostForm("password")
		if password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "请输入备份文件密码。"})
			return
		}

		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "打开上传文件失败: " + err.Error()})
			return
		}
		defer src.Close()

		fileBytes, err := io.ReadAll(src)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "读取上传文件失败: " + err.Error()})
			return
		}

		zipReader, err := zip.NewReader(bytes.NewReader(fileBytes), int64(len(fileBytes)))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的 ZIP 文件: " + err.Error()})
			return
		}

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

		if err := json.NewDecoder(jsonFile).Decode(&backupData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "解析 JSON 数据失败: " + err.Error()})
			return
		}
	}

	if err := h.restoreFromBackupData(&backupData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": fmt.Sprintf("恢复成功！导入 %d 篇文章并更新了站点设置。", len(backupData.Posts))})
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
