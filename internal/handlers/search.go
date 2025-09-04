package handlers

import (
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
		c.Redirect(http.StatusFound, "/")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := 10 // 与首页保持一致

	isLoggedIn, _ := c.Get("IsLoggedIn")

	posts, total, err := h.postService.SearchPublishedPostsPage(query, page, pageSize, isLoggedIn.(bool))
	if err != nil {
		render(c, http.StatusInternalServerError, "404.html", gin.H{
			"error": "Search failed",
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	pagination := utils.GeneratePagination(page, totalPages)

	render(c, http.StatusOK, "search.html", gin.H{
		"posts":      posts,
		"query":      query,
		"Pagination": pagination,
	})
}
