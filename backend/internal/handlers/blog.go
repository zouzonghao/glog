package handlers

import (
	"glog/internal/constants"
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

// GetPosts handles the request to get a paginated list of posts.
func (h *BlogHandler) GetPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := 10 // Or get from query param

	isLoggedInValue, exists := c.Get(constants.ContextKeyIsLoggedIn)
	isLoggedIn := exists && isLoggedInValue.(bool)

	posts, total, err := h.postService.GetPostsPage(page, pageSize, isLoggedIn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load posts"})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	pagination := utils.GeneratePagination(page, totalPages)

	c.JSON(http.StatusOK, gin.H{
		"posts":      posts,
		"pagination": pagination,
		"total":      total,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": totalPages,
	})
}

// GetPostBySlug handles the request to get a single post by its slug.
func (h *BlogHandler) GetPostBySlug(c *gin.Context) {
	slug := c.Param("slug")
	isLoggedInValue, exists := c.Get(constants.ContextKeyIsLoggedIn)
	isLoggedIn := exists && isLoggedInValue.(bool)

	post, err := h.postService.GetPostBySlug(slug, isLoggedIn)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	c.JSON(http.StatusOK, post)
}
