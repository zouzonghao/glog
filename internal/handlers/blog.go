package handlers

import (
	"glog/internal/services"
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
	isLoggedIn, _ := c.Get("IsLoggedIn")
	posts, err := h.postService.GetAllPublishedPosts(isLoggedIn.(bool))
	if err != nil {
		render(c, http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to load posts",
		})
		return
	}

	render(c, http.StatusOK, "index.html", gin.H{
		"posts": posts,
	})
}

func (h *BlogHandler) ShowPost(c *gin.Context) {
	slug := c.Param("slug")
	isLoggedIn, _ := c.Get("IsLoggedIn")

	post, err := h.postService.GetRenderedPostBySlug(slug, isLoggedIn.(bool))
	if err != nil {
		// Render custom 404 page
		render(c, http.StatusNotFound, "404.html", gin.H{})
		return
	}

	render(c, http.StatusOK, "post.html", gin.H{
		"post": post,
	})
}

func (h *BlogHandler) NotFound(c *gin.Context) {
	render(c, http.StatusNotFound, "404.html", gin.H{})
}
