package handlers

import (
	"glog/internal/constants"
	"glog/internal/models"
	"glog/internal/services"
	"glog/internal/utils"
	"math"
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
		query = c.Query("query")
	}
	tag := c.Query("tag")

	if query == "" && tag == "" {
		c.Redirect(http.StatusFound, "/")
		return
	}

	view := GetViewPreference(c)

	page := utils.ParsePage(c)
	pageSize := 10

	isLoggedInVal, _ := c.Get(constants.ContextKeyIsLoggedIn)
	isLoggedIn := isLoggedInVal != nil && isLoggedInVal.(bool)

	var posts []models.RenderedPost
	var total int
	var err error

	if tag != "" && query != "" {
		posts, total, err = h.postService.SearchPostsPageByTag(query, tag, page, pageSize, isLoggedIn)
	} else if tag != "" {
		posts, total, err = h.postService.GetPostsPageByTag(tag, page, pageSize, isLoggedIn)
	} else {
		posts, total, err = h.postService.SearchPostsPage(query, page, pageSize, isLoggedIn)
	}

	if err != nil {
		render(c, http.StatusInternalServerError, "404.html", gin.H{
			"error": "Search failed",
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	pagination := utils.GeneratePagination(page, totalPages)

	templateName := "search.html"
	if view == "cards" {
		templateName = "search_cards.html"
	}

	pageTitle := query
	if tag != "" {
		if query != "" {
			pageTitle = tag + " / " + query
		} else {
			pageTitle = tag
		}
	}

	render(c, http.StatusOK, templateName, gin.H{
		"posts":      posts,
		"query":      query,
		"Query":      query,
		"tag":        tag,
		"pageTitle":  pageTitle,
		"Pagination": pagination,
		"View":       view,
	})
}
