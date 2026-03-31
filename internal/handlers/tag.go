package handlers

import (
	"glog/internal/constants"
	"glog/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TagHandler struct {
	postService *services.PostService
}

func NewTagHandler(postService *services.PostService) *TagHandler {
	return &TagHandler{postService: postService}
}

func (h *TagHandler) ListTags(c *gin.Context) {
	isLoggedInValue, exists := c.Get(constants.ContextKeyIsLoggedIn)
	isLoggedIn, _ := isLoggedInValue.(bool)
	isLoggedIn = exists && isLoggedIn

	tags, err := h.postService.GetAllTags(isLoggedIn)
	if err != nil {
		render(c, http.StatusInternalServerError, "404.html", gin.H{
			"error": "加载标签失败",
		})
		return
	}

	render(c, http.StatusOK, "tags.html", gin.H{
		"tags": tags,
	})
}
