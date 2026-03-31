package handlers

import (
	"glog/internal/services"
	"glog/internal/utils"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	maxExcerptLength = 500
	maxCoverLength   = 2048
)

type APIHandler struct {
	postService *services.PostService
}

func NewAPIHandler(postService *services.PostService) *APIHandler {
	return &APIHandler{
		postService: postService,
	}
}

type PostListItem struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Excerpt     string `json:"excerpt"`
	HasCover    bool   `json:"has_cover"`
	Cover       string `json:"cover,omitempty"`
	PublishedAt string `json:"published_at"`
	IsPrivate   bool   `json:"is_private"`
}

type PostContent struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Content     string `json:"content"`
	Cover       string `json:"cover,omitempty"`
	PublishedAt string `json:"published_at"`
}

type UpdateExcerptRequest struct {
	Excerpt string `json:"excerpt"`
}

type UpdateCoverRequest struct {
	Cover string `json:"cover"`
}

func (h *APIHandler) GetPosts(c *gin.Context) {
	page, pageSize := utils.ParsePagination(c, 10)

	posts, total, err := h.postService.GetPostsPage(page, pageSize, true)
	if err != nil {
		log.Printf("API error: GetPostsPage: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "获取文章列表失败"})
		return
	}

	items := make([]PostListItem, len(posts))
	for i, post := range posts {
		items[i] = PostListItem{
			ID:          post.ID,
			Title:       post.Title,
			Slug:        post.Slug,
			Excerpt:     post.Excerpt,
			HasCover:    post.Cover != "",
			Cover:       post.Cover,
			PublishedAt: post.PublishedAt.Format("2006-01-02 15:04:05"),
			IsPrivate:   post.IsPrivate,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "获取文章列表成功",
		"data": gin.H{
			"posts":     items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func (h *APIHandler) GetPost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的文章 ID"})
		return
	}

	post, err := h.postService.GetPostByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "文章不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "获取文章成功",
		"data": PostContent{
			ID:          post.ID,
			Title:       post.Title,
			Slug:        post.Slug,
			Content:     post.Content,
			Cover:       post.Cover,
			PublishedAt: post.PublishedAt.Format("2006-01-02 15:04:05"),
		},
	})
}

func (h *APIHandler) UpdateExcerpt(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的文章 ID"})
		return
	}

	var req UpdateExcerptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "请求格式错误"})
		return
	}

	if len(req.Excerpt) > maxExcerptLength {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "摘要长度不能超过 500 字符"})
		return
	}

	if err := h.postService.UpdateExcerptByID(uint(id), req.Excerpt); err != nil {
		log.Printf("API error: UpdateExcerptByID: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "更新摘要失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "摘要更新成功",
		"data":    gin.H{"id": id},
	})
}

func (h *APIHandler) UpdateCover(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的文章 ID"})
		return
	}

	var req UpdateCoverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "请求格式错误"})
		return
	}

	if len(req.Cover) > maxCoverLength {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "封面 URL 长度不能超过 2048 字符"})
		return
	}

	if err := h.postService.UpdateCoverByID(uint(id), req.Cover); err != nil {
		log.Printf("API error: UpdateCoverByID: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "更新封面失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "封面更新成功",
		"data":    gin.H{"id": id},
	})
}
