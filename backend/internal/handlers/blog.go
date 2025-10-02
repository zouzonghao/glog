package handlers

import (
	"glog/internal/constants"
	"glog/internal/models"
	"glog/internal/services"
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
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	isLoggedInValue, exists := c.Get(constants.ContextKeyIsLoggedIn)
	isLoggedIn := exists && isLoggedInValue.(bool)

	posts, total, err := h.postService.GetPostsPage(page, pageSize, isLoggedIn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load posts"})
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	pagination := models.Pagination{
		CurrentPage:  page,
		TotalPages:   totalPages,
		TotalRecords: total,
		PageSize:     pageSize,
		HasPrev:      page > 1,
		HasNext:      page < totalPages,
	}

	// Create a summary response to avoid sending full content in list views
	type PostSummaryResponse struct {
		ID          uint   `json:"id"`
		PublishedAt string `json:"published_at"`
		Title       string `json:"title"`
		Slug        string `json:"slug"`
		Cover       string `json:"cover"`
		Excerpt     string `json:"excerpt"`
		IsPrivate   bool   `json:"is_private"`
	}

	postSummaries := make([]PostSummaryResponse, len(posts))
	for i, post := range posts {
		var publishedAtStr string
		if !post.PublishedAt.IsZero() {
			publishedAtStr = post.PublishedAt.Format("2006-01-02T15:04:05Z")
		}
		postSummaries[i] = PostSummaryResponse{
			ID:          post.ID,
			PublishedAt: publishedAtStr,
			Title:       post.Title,
			Slug:        post.Slug,
			Cover:       post.Cover,
			Excerpt:     post.Excerpt,
			IsPrivate:   post.IsPrivate,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"posts":      postSummaries,
		"pagination": pagination,
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
