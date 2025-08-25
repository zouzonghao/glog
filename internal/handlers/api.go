package handlers

import (
	"glog/internal/models"
	"glog/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type APIHandler struct {
	postService *services.PostService
}

func NewAPIHandler(postService *services.PostService) *APIHandler {
	return &APIHandler{
		postService: postService,
	}
}

// CreatePost handles the API request to create a new post.
func (h *APIHandler) CreatePost(c *gin.Context) {
	var post models.Post
	if err := c.ShouldBindJSON(&post); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// For API creation, we don't trigger AI summary by default.
	// PublishedAt will be set by the service if not provided.
	createdPost, err := h.postService.CreatePost(post.Title, post.Content, post.IsPrivate, false, post.PublishedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createdPost)
}

// FindPosts handles the API request to find posts.
func (h *APIHandler) FindPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	query := c.Query("query")

	var posts []models.RenderedPost
	var total int64
	var err error

	if query != "" {
		renderedPosts, totalInt, searchErr := h.postService.SearchPostsPage(query, page, pageSize, true)
		if searchErr != nil {
			err = searchErr
		} else {
			posts = renderedPosts
			total = int64(totalInt)
		}
	} else {
		renderedPosts, totalInt, pageErr := h.postService.GetPostsPage(page, pageSize, true)
		if pageErr != nil {
			err = pageErr
		} else {
			posts = renderedPosts
			total = int64(totalInt)
		}
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"posts": posts,
		"total": total,
	})
}
