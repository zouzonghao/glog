package handlers

import (
	"glog/internal/services"
	"glog/internal/utils"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type BlogHandler struct {
	postService *services.PostService
}

func NewBlogHandler(postService *services.PostService) *BlogHandler {
	return &BlogHandler{postService: postService}
}

func (h *BlogHandler) Index(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := 10 // 每页显示15篇文章

	isLoggedInValue, exists := c.Get("IsLoggedIn")
	isLoggedIn := exists && isLoggedInValue.(bool)
	posts, total, err := h.postService.GetPostsPage(page, pageSize, isLoggedIn)
	if err != nil {
		render(c, http.StatusInternalServerError, "404.html", gin.H{
			"error": "加载文章失败",
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	pagination := utils.GeneratePagination(page, totalPages)

	render(c, http.StatusOK, "index.html", gin.H{
		"posts":      posts,
		"Pagination": pagination,
	})
}

func (h *BlogHandler) ShowPost(c *gin.Context) {
	slug := c.Param("slug")
	isLoggedIn, _ := c.Get("IsLoggedIn")

	post, err := h.postService.GetRenderedPostBySlug(slug, isLoggedIn.(bool))
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
