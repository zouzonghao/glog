package handlers

import (
	"fmt"
	"glog/internal/constants"
	"glog/internal/services"
	"glog/internal/utils"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type BlogHandler struct {
	postService *services.PostService
}

func NewBlogHandler(postService *services.PostService) *BlogHandler {
	return &BlogHandler{postService: postService}
}

func (h *BlogHandler) Index(c *gin.Context) {
	// 视图切换逻辑
	// 1. 从查询参数获取 (最高优先级)
	view := c.Query("view")

	// 2. 如果查询参数没有，从 cookie 获取
	if view == "" {
		cookie, err := c.Cookie("view")
		if err == nil {
			view = cookie
		}
	}

	// 3. 如果都没有，根据 User-Agent 判断
	if view == "" {
		userAgent := c.Request.UserAgent()
		// 简单的移动端判断逻辑
		if strings.Contains(strings.ToLower(userAgent), "mobile") || strings.Contains(strings.ToLower(userAgent), "android") || strings.Contains(strings.ToLower(userAgent), "iphone") {
			view = "cards"
		} else {
			view = "list"
		}
	}

	// 规范化 view 的值，防止注入等问题
	if view != "cards" {
		view = "list"
	}

	// 设置 cookie，以便记住用户的选择
	// 域名设置为根路径，有效期设置为一年
	c.SetCookie("view", view, 3600*24*365, "/", "", false, true)

	// 使用 Link 响应头预加载关键资源
	// 这是一个比 HTTP/2 Server Push 更现代、更受浏览器支持的方案
	header := c.Writer.Header()
	header.Add("Link", fmt.Sprintf(`</static/css/style.css>; rel=preload; as=style`))
	header.Add("Link", fmt.Sprintf(`</static/js/main.js>; rel=preload; as=script`))
	// 如果是卡片视图，额外预加载其专属资源
	if view == "cards" {
		header.Add("Link", fmt.Sprintf(`</static/css/cards.css>; rel=preload; as=style`))
		header.Add("Link", fmt.Sprintf(`</static/js/cards.js>; rel=preload; as=script`))
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := 10 // 每页显示10篇文章

	isLoggedInValue, exists := c.Get(constants.ContextKeyIsLoggedIn)
	isLoggedIn := exists && isLoggedInValue.(bool)
	posts, total, err := h.postService.GetPostsPage(page, pageSize, isLoggedIn)
	if err != nil {
		render(c, http.StatusInternalServerError, "404.html", gin.H{
			"error": "加载文章失败",
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	pagination := utils.GeneratePagination(page, totalPages)

	// 根据视图选择渲染的模板
	templateName := "index.html"
	if view == "cards" {
		templateName = "index_cards.html"
	}

	render(c, http.StatusOK, templateName, gin.H{
		"posts":      posts,
		"Pagination": pagination,
		"View":       view, // 将视图名称传递给模板
		"is_index":   true, // 标记这是首页
	})
}

func (h *BlogHandler) ShowPost(c *gin.Context) {
	slug := c.Param("slug")
	isLoggedIn, _ := c.Get(constants.ContextKeyIsLoggedIn)

	post, err := h.postService.GetPostBySlug(slug, isLoggedIn.(bool))
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
