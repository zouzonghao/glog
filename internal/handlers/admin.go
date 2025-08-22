package handlers

import (
	"glog/internal/models"
	"glog/internal/services"
	"glog/internal/utils"
	"math"
	"net/http"
	"strconv"

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
	pageSize := 10 // 每页显示10篇文章

	posts, total, err := h.postService.GetPostsPage(page, pageSize)

	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load posts")
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	pagination := utils.GeneratePagination(page, totalPages)

	render(c, http.StatusOK, "admin.html", gin.H{
		"posts":      posts,
		"Pagination": pagination,
		"Query":      "", // Was removed, keep the key for the template
	})
}

func (h *AdminHandler) NewPost(c *gin.Context) {
	render(c, http.StatusOK, "editor.html", gin.H{
		"post": nil, // Pass a nil post to indicate a new post
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
	published := c.PostForm("published") == "on"
	isPrivate := c.PostForm("is_private") == "on"
	aiSummary := c.PostForm("ai_summary") == "on"

	// Check for lock before proceeding
	if idStr != "" && idStr != "0" {
		id, _ := strconv.ParseUint(idStr, 10, 64)
		if h.postService.CheckPostLock(uint(id)) {
			c.JSON(http.StatusConflict, gin.H{
				"status":  "locked",
				"message": "正在生成AI摘要，文章已锁定，请稍候再试。",
			})
			return
		}
	}

	var post *models.Post
	var err error

	if idStr == "" || idStr == "0" {
		post, err = h.postService.CreatePost(title, content, published, isPrivate, aiSummary)
	} else {
		id, _ := strconv.ParseUint(idStr, 10, 64)
		post, err = h.postService.UpdatePost(uint(id), title, content, published, isPrivate, aiSummary)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "保存文章失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "文章已保存。",
		"post_id": post.ID,
		"slug":    post.Slug,
	})
}

func (h *AdminHandler) DeletePost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid post ID")
		return
	}

	err = h.postService.DeletePost(uint(id))
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to delete post")
		return
	}

	c.Redirect(http.StatusFound, "/admin")
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

	if token == "" {
		// If the token field is empty, try to get the existing token from settings
		// This allows testing without re-entering the token
		settings, err := h.settingService.GetAllSettings()
		if err == nil {
			token = settings["openai_token"]
		}
	}

	testContent := "这是一个用于测试AI摘要功能的文本。"
	_, err := h.aiService.GenerateSummaryAndTitle(testContent, false, baseURL, token, model)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "error", "message": "测试失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "测试成功！连接和配置均有效。"})
}
