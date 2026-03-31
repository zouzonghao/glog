package handlers

import (
	"glog/internal/constants"
	"glog/internal/services"
	"glog/internal/utils"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
)

type BlogHandler struct {
	postService *services.PostService
}

func NewBlogHandler(postService *services.PostService) *BlogHandler {
	return &BlogHandler{postService: postService}
}

func (h *BlogHandler) Index(c *gin.Context) {
	view := GetViewPreference(c)

	page := utils.ParsePage(c)
	pageSize := 10

	isLoggedInValue, exists := c.Get(constants.ContextKeyIsLoggedIn)
	isLoggedIn, _ := isLoggedInValue.(bool)
	isLoggedIn = exists && isLoggedIn
	posts, total, err := h.postService.GetPostsPage(page, pageSize, isLoggedIn)
	if err != nil {
		render(c, http.StatusInternalServerError, "404.html", gin.H{
			"error": "加载文章失败",
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	pagination := utils.GeneratePagination(page, totalPages)

	// 根据视图选择渲染的模板
	templateName := "index.html"
	if view == "cards" {
		templateName = "index_cards.html"
	}

	render(c, http.StatusOK, templateName, gin.H{
		"posts":      posts,
		"Pagination": pagination,
		"View":       view, // 将视图名称传递给模板
		"is_index":   true, // 标记这是首页
	})
}

func (h *BlogHandler) ShowPost(c *gin.Context) {
	slug := c.Param("slug")
	isLoggedInVal, _ := c.Get(constants.ContextKeyIsLoggedIn)
	isLoggedIn := isLoggedInVal != nil && isLoggedInVal.(bool)

	post, err := h.postService.GetPostBySlug(slug, isLoggedIn)
	if err != nil {
		// Render custom 404 page
		render(c, http.StatusNotFound, "404.html", gin.H{})
		return
	}

	render(c, http.StatusOK, "post.html", gin.H{
		"post": post,
	})
}

func (h *BlogHandler) NotFound(c *gin.Context) {
	render(c, http.StatusNotFound, "404.html", gin.H{})
}
