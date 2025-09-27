package handlers

import (
	"glog/internal/constants"
	"glog/internal/models"
	"glog/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	postService *services.PostService
}

func NewSearchHandler(postService *services.PostService) *SearchHandler {
	return &SearchHandler{postService: postService}
}

func (h *SearchHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	isLoggedInValue, exists := c.Get(constants.ContextKeyIsLoggedIn)
	isLoggedIn := exists && isLoggedInValue.(bool)

	posts, total, err := h.postService.SearchPostsPage(query, page, pageSize, isLoggedIn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
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
		// This requires importing the "time" package
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
