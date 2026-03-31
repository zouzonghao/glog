package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"glog/internal/constants"
	"glog/internal/models"
	"glog/internal/services"
	"glog/internal/tasks"
	"glog/internal/utils"
	"io"
	"math"
	"net/http"
	"os"
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
	backupService  *services.BackupService
	scheduler      *tasks.Scheduler
}

func NewAdminHandler(postService *services.PostService, settingService *services.SettingService, backupService *services.BackupService, scheduler *tasks.Scheduler) *AdminHandler {
	return &AdminHandler{
		postService:    postService,
		settingService: settingService,
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

	allowedSettings := map[string]bool{
		constants.SettingPassword:        true,
		constants.SettingFavicon:         true,
		constants.SettingSiteTitle:       true,
		constants.SettingSiteDescription: true,
		constants.SettingCoverPrefix:     true,
		constants.SettingWebdavURL:       true,
		constants.SettingWebdavUser:      true,
		constants.SettingWebdavPassword:  true,
		constants.SettingSyncInterval:    true,
		constants.SettingSyncMaxBackups:  true,
	}

	for key, values := range c.Request.PostForm {
		if !allowedSettings[key] {
			continue
		}
		if len(values) > 0 {
			value := values[0]
			if (key == constants.SettingPassword || key == constants.SettingWebdavPassword) && value == "" {
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
	page, pageSize := utils.ParsePagination(c, 10)
	query := c.Query("q")
	if query == "" {
		query = c.Query("query")
	}
	status := c.DefaultQuery("status", "all")

	posts, total, err := h.postService.GetPostsPageByAdmin(page, pageSize, query, status)
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
	now := time.Now().UTC().Format("2006-01-02 15:04")
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
	tag := c.PostForm("tag")
	publishedAtStr := c.PostForm("published_at")
	isPrivate := c.PostForm("is_private") == "on"

	publishedAt, err := time.Parse("2006-01-02 15:04", publishedAtStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的发布时间格式"})
		return
	}

	var post *models.Post

	if idStr == "" || idStr == "0" {
		post, err = h.postService.CreatePost(title, content, tag, isPrivate, publishedAt)
	} else {
		id, parseErr := strconv.ParseUint(idStr, 10, 64)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的文章 ID"})
			return
		}
		post, err = h.postService.UpdatePost(uint(id), title, content, tag, isPrivate, publishedAt)
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

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "文章已保存！",
		"post_id": post.ID,
		"slug":    post.Slug,
	})
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
	settings, err := h.settingService.GetAllSettings()
	if err != nil {
		render(c, http.StatusInternalServerError, "settings.html", gin.H{
			"error": "无法加载设置",
		})
		return
	}

	data := make(gin.H)
	for k, v := range settings {
		data[k] = v
	}

	render(c, http.StatusOK, "settings.html", data)
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

	if strings.Contains(contentType, "application/json") {
		postCount, err := h.postService.CreatePostsFromBackupStream(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": fmt.Sprintf("恢复成功！导入 %d 篇文章并更新了站点设置。", postCount)})
		return
	}

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

	postCount, err := h.postService.CreatePostsFromBackupStream(jsonFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": fmt.Sprintf("恢复成功！导入 %d 篇文章并更新了站点设置。", postCount)})
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
