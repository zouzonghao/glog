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
	pageSize := 10 // Keep consistent with other endpoints

	isLoggedInValue, exists := c.Get(constants.ContextKeyIsLoggedIn)
	isLoggedIn := exists && isLoggedInValue.(bool)

	posts, total, err := h.postService.SearchPostsPage(query, page, pageSize, isLoggedIn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
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
