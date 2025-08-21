package handlers

import (
	"glog/internal/models"
	"glog/internal/services"
	"html/template"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	postService *services.PostService
}

func NewAdminHandler(postService *services.PostService) *AdminHandler {
	return &AdminHandler{postService: postService}
}

func (h *AdminHandler) ListPosts(c *gin.Context) {
	posts, err := h.postService.GetAllPosts()
	if err != nil {
		// In a real app, you'd have a proper error template
		c.String(http.StatusInternalServerError, "Failed to load posts")
		return
	}

	tmpl, err := template.ParseFiles("templates/base.html", "templates/admin.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to parse templates")
		return
	}
	tmpl.ExecuteTemplate(c.Writer, "base.html", gin.H{"posts": posts})
}

func (h *AdminHandler) Editor(c *gin.Context) {
	idStr := c.Query("id")
	var post *models.Post
	var err error

	if idStr != "" {
		id, _ := strconv.ParseUint(idStr, 10, 64)
		post, err = h.postService.GetPostByID(uint(id))
		if err != nil {
			c.Redirect(http.StatusFound, "/admin")
			return
		}
	}

	tmpl, err := template.ParseFiles("templates/base.html", "templates/editor.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to parse templates")
		return
	}
	tmpl.ExecuteTemplate(c.Writer, "base.html", gin.H{
		"post": post,
	})
}

func (h *AdminHandler) SavePost(c *gin.Context) {
	idStr := c.PostForm("id")
	title := c.PostForm("title")
	content := c.PostForm("content")
	published := c.PostForm("published") == "on"

	var post *models.Post
	var err error

	if idStr == "" || idStr == "0" {
		post, err = h.postService.CreatePost(title, content, published)
	} else {
		id, _ := strconv.ParseUint(idStr, 10, 64)
		post, err = h.postService.UpdatePost(uint(id), title, content, published)
	}

	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to save post")
		return
	}

	if published {
		c.Redirect(http.StatusFound, "/post/"+post.Slug)
	} else {
		c.Redirect(http.StatusFound, "/admin/editor?id="+strconv.Itoa(int(post.ID)))
	}
}

func (h *AdminHandler) DeletePost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid post ID")
		return
	}

	err = h.postService.DeletePost(uint(id))
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to delete post")
		return
	}

	c.Redirect(http.StatusFound, "/admin")
}
