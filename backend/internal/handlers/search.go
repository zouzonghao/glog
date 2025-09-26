package handlers

import (
	"glog/internal/constants"
	"glog/internal/services"
	"glog/internal/utils"
	"math"
	"net/http"
	"strconv"
	"strings"

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

	// --- 视图切换逻辑 Start ---
	view := c.Query("view")
	if view == "" {
		cookie, err := c.Cookie("view")
		if err == nil {
			view = cookie
		}
	}
	if view == "" {
		userAgent := c.Request.UserAgent()
		if strings.Contains(strings.ToLower(userAgent), "mobile") || strings.Contains(strings.ToLower(userAgent), "android") || strings.Contains(strings.ToLower(userAgent), "iphone") {
			view = "cards"
		} else {
			view = "list"
		}
	}
	if view != "cards" {
		view = "list"
	}
	c.SetCookie("view", view, 3600*24*365, "/", "", false, true)
	// --- 视图切换逻辑 End ---

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := 10 // 与首页保持一致

	isLoggedIn, _ := c.Get(constants.ContextKeyIsLoggedIn)

	posts, total, err := h.postService.SearchPostsPage(query, page, pageSize, isLoggedIn.(bool))
	if err != nil {
		render(c, http.StatusInternalServerError, "404.html", gin.H{
			"error": "Search failed",
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	pagination := utils.GeneratePagination(page, totalPages)

	// 根据视图选择渲染的模板
	templateName := "search.html"
	if view == "cards" {
		templateName = "search_cards.html"
	}

	render(c, http.StatusOK, templateName, gin.H{
		"posts":      posts,
		"query":      query,
		"Pagination": pagination,
		"View":       view, // 将视图名称传递给模板
	})
}
