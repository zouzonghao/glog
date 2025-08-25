package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"glog/internal/models"
	"glog/internal/services"
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
)

type AdminHandler struct {
	postService    *services.PostService
	settingService *services.SettingService
	aiService      *services.AIService
}

func NewAdminHandler(postService *services.PostService, settingService *services.SettingService, aiService *services.AIService) *AdminHandler {
	return &AdminHandler{
		postService:    postService,
		settingService: settingService,
		aiService:      aiService,
	}
}

func (h *AdminHandler) ListPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	query := c.Query("query")
	status := c.DefaultQuery("status", "all")
	pageSize := 10

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
	flashes := session.Flashes("success")
	session.Save() // Clear flashes after reading

	render(c, http.StatusOK, "admin.html", gin.H{
		"posts":      posts,
		"Pagination": pagination,
		"Query":      query,
		"Status":     status,
		"Flashes":    flashes,
	})
}

func (h *AdminHandler) NewPost(c *gin.Context) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Now().In(loc).Format("2006-01-02 15:04")
	render(c, http.StatusOK, "editor.html", gin.H{
		"post": nil, // Pass a nil post to indicate a new post
		"now":  now,
	})
}

func (h *AdminHandler) Editor(c *gin.Context) {
	idStr := c.Query("id")
	status := c.Query("status") // For feedback from non-AJAX fallbacks if any

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

	// Check for lock before proceeding
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

	if idStr == "" || idStr == "0" {
		post, err = h.postService.CreatePost(title, content, isPrivate, aiSummary, publishedAt)
	} else {
		id, _ := strconv.ParseUint(idStr, 10, 64)
		post, err = h.postService.UpdatePost(uint(id), title, content, isPrivate, aiSummary, publishedAt)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "保存文章失败: " + err.Error(),
		})
		return
	}

	message := "文章已保存！"
	if aiSummary && title == "未命名标题" {
		message = "文章已保存，AI正在生成标题和摘要，请稍后刷新查看..."
	} else if aiSummary {
		message = "文章已保存，AI摘要正在生成中..."
	}

	response := gin.H{
		"status":  "success",
		"message": message,
		"post_id": post.ID,
	}

	// Only return slug if AI is not going to rename the post
	if !(aiSummary && title == "未命名标题") {
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
	// The render function will automatically inject settings from the context.
	render(c, http.StatusOK, "settings.html", gin.H{})
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
			// Special handling for password fields: only update if not empty
			if (key == "password" || key == "openai_token") && value == "" {
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

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "设置已成功保存！"})
}

func (h *AdminHandler) TestAISettings(c *gin.Context) {
	baseURL := c.PostForm("openai_base_url")
	token := c.PostForm("openai_token")
	model := c.PostForm("openai_model")

	finalToken := token
	if finalToken == "" {
		settings, err := h.settingService.GetAllSettings()
		if err == nil {
			finalToken = settings["openai_token"]
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

func (h *AdminHandler) BackupPosts(c *gin.Context) {
	posts, err := h.postService.GetAllPostsForBackup()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "获取文章失败: " + err.Error()})
		return
	}

	jsonData, err := json.MarshalIndent(posts, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "JSON 序列化失败: " + err.Error()})
		return
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)
	zipFile, err := zipWriter.Create("backup.json")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "创建 ZIP 文件失败: " + err.Error()})
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

func (h *AdminHandler) UploadPosts(c *gin.Context) {
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

	var jsonReader io.Reader = src

	// Handle ZIP file
	if strings.HasSuffix(file.Filename, ".zip") {
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

		if len(zipReader.File) == 0 || zipReader.File[0].Name != "backup.json" {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "ZIP 文件中未找到 backup.json"})
			return
		}

		jsonFile, err := zipReader.File[0].Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "打开 backup.json 失败: " + err.Error()})
			return
		}
		defer jsonFile.Close()
		jsonReader = jsonFile
	}

	var posts []models.PostBackup
	if err := json.NewDecoder(jsonReader).Decode(&posts); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "解析 JSON 数据失败: " + err.Error()})
		return
	}

	if err := h.postService.CreatePostsFromBackup(posts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "导入文章失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": fmt.Sprintf("成功导入 %d 篇文章！", len(posts))})
}
