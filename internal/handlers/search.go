package handlers

import (
	"glog/internal/services"
	"html/template"
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

	posts, err := h.postService.SearchPosts(query)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Search failed",
		})
		return
	}

	// 每次请求都显式解析正确的模板文件组合
	tmpl, err := template.ParseFiles("templates/base.html", "templates/search.html")
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to parse templates",
		})
		return
	}

	// 使用解析好的模板执行渲染
	tmpl.ExecuteTemplate(c.Writer, "base.html", gin.H{
		"posts": posts,
		"query": query,
	})
}
