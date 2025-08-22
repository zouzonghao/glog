package handlers

import (
	"glog/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type APIHandler struct {
	postService *services.PostService
}

func NewAPIHandler(postService *services.PostService) *APIHandler {
	return &APIHandler{postService: postService}
}

type CreatePostRequest struct {
	Title     string `json:"title" binding:"required"`
	Content   string `json:"content" binding:"required"`
	Published bool   `json:"published"`
	WithAI    bool   `json:"with_ai"`
	IsPrivate bool   `json:"is_private"`
}

func (h *APIHandler) CreatePost(c *gin.Context) {
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// For API creation, we don't need AI summary.
	post, err := h.postService.CreatePost(req.Title, req.Content, req.Published, req.IsPrivate, req.WithAI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建文章失败"})
		return
	}

	c.JSON(http.StatusCreated, post)
}

func (h *APIHandler) FindPosts(c *gin.Context) {
	query := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "15"))

	var (
		posts interface{}
		total int64
		err   error
	)

	if query != "" {
		posts, total, err = h.postService.SearchPublishedPostsPage(query, page, pageSize, true)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "搜索文章失败"})
			return
		}
	} else {
		posts, total, err = h.postService.GetPublishedPostsPage(page, pageSize, true)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文章失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"posts": posts,
		"total": total,
	})
}
