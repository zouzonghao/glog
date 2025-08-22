package main

import (
	"log"

	"glog/internal/handlers"
	"glog/internal/repository"
	"glog/internal/services"
	"glog/internal/utils"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func createRenderer() multitemplate.Renderer {
	r := multitemplate.NewRenderer()
	r.AddFromFiles("index.html", "templates/base.html", "templates/index.html", "templates/_pagination.html")
	r.AddFromFiles("post.html", "templates/base.html", "templates/post.html")
	r.AddFromFiles("admin.html", "templates/base.html", "templates/admin.html", "templates/_pagination.html")
	r.AddFromFiles("editor.html", "templates/base.html", "templates/editor.html")
	r.AddFromFiles("settings.html", "templates/base.html", "templates/settings.html")
	r.AddFromFiles("login.html", "templates/base.html", "templates/login.html")
	r.AddFromFiles("search.html", "templates/base.html", "templates/search.html", "templates/_pagination.html")
	r.AddFromFiles("404.html", "templates/base.html", "templates/404.html")
	// Assuming error.html exists and is used by handlers
	r.AddFromFiles("error.html", "templates/base.html", "templates/error.html")
	return r
}

func main() {
	// 初始化数据库
	db, err := utils.InitDatabase()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// 初始化依赖
	postRepo := repository.NewPostRepository(db)
	settingRepo := repository.NewSettingRepository(db)

	settingService := services.NewSettingService(settingRepo)
	aiService := services.NewAIService()
	postService := services.NewPostService(postRepo, settingService, aiService)

	blogHandler := handlers.NewBlogHandler(postService)
	adminHandler := handlers.NewAdminHandler(postService, settingService, aiService)
	searchHandler := handlers.NewSearchHandler(postService)
	authHandler := handlers.NewAuthHandler(settingService)

	// 设置Gin路由
	r := gin.Default()
	r.HTMLRender = createRenderer()

	// 设置会话中间件
	store := cookie.NewStore([]byte("secret-key-should-be-changed"))
	r.Use(sessions.Sessions("glog_session", store))

	// 加载设置的中间件
	r.Use(handlers.SettingsMiddleware(settingService))

	// 静态文件服务
	r.Static("/static", "./static")

	// 博客路由
	r.GET("/", blogHandler.Index)
	r.GET("/post/:slug", blogHandler.ShowPost)
	r.GET("/search", searchHandler.Search)

	// 认证路由
	r.GET("/login", authHandler.ShowLoginPage)
	r.POST("/login", authHandler.Login)
	r.GET("/logout", authHandler.Logout)

	// 后台路由
	admin := r.Group("/admin")
	admin.Use(handlers.AuthMiddleware())
	{
		admin.GET("/", adminHandler.ListPosts)
		admin.GET("/new", adminHandler.NewPost)
		admin.GET("/editor", adminHandler.Editor)
		admin.POST("/save", adminHandler.SavePost)
		admin.GET("/delete/:id", adminHandler.DeletePost)
	}

	settings := r.Group("/settings")
	settings.Use(handlers.AuthMiddleware())
	{
		settings.GET("/", adminHandler.ShowSettingsPage)
		settings.POST("/", adminHandler.UpdateSettings)
		settings.POST("/test-ai", adminHandler.TestAISettings)
	}

	// 404处理
	r.NoRoute(blogHandler.NotFound)

	// 启动服务器
	log.Println("Server starting on :8080")
	r.Run(":8080")
}
