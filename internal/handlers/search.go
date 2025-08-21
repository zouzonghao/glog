package handlers

import (
	"glog/internal/services"
	"net/http"

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

	isLoggedIn, _ := c.Get("IsLoggedIn")

	posts, err := h.postService.SearchPosts(query, isLoggedIn.(bool))
	if err != nil {
		render(c, http.StatusInternalServerError, "error.html", gin.H{
			"error": "Search failed",
		})
		return
	}

	render(c, http.StatusOK, "search.html", gin.H{
		"posts": posts,
		"query": query,
	})
}
