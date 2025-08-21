package handlers

import (
	"glog/internal/services"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

type BlogHandler struct {
	postService *services.PostService
}

func NewBlogHandler(postService *services.PostService) *BlogHandler {
	return &BlogHandler{postService: postService}
}

func (h *BlogHandler) Index(c *gin.Context) {
	posts, err := h.postService.GetAllPublishedPosts()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to load posts",
		})
		return
	}

	// 每次请求都显式解析正确的模板文件组合
	tmpl, err := template.ParseFiles("templates/base.html", "templates/index.html")
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to parse templates",
		})
		return
	}

	// 使用解析好的模板执行渲染
	tmpl.ExecuteTemplate(c.Writer, "base.html", gin.H{
		"posts": posts,
	})
}

func (h *BlogHandler) ShowPost(c *gin.Context) {
	slug := c.Param("slug")

	// GetRenderedPostBySlug 现在返回 *models.RenderedPost
	post, err := h.postService.GetRenderedPostBySlug(slug)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Post not found",
		})
		return
	}

	// 每次请求都显式解析正确的模板文件组合
	tmpl, err := template.ParseFiles("templates/base.html", "templates/post.html")
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to parse templates",
		})
		return
	}

	// 使用解析好的模板执行渲染
	tmpl.ExecuteTemplate(c.Writer, "base.html", gin.H{
		"post": post,
	})
}
