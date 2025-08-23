package main

import (
	"flag"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"glog/internal/handlers"
	"glog/internal/repository"
	"glog/internal/services"
	"glog/internal/utils"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

// Global filesystems that will be populated by either assets_dev.go or assets_prod.go at startup.
var templatesFS fs.FS
var staticFS fs.FS

func createRenderer() multitemplate.Renderer {
	r := multitemplate.NewRenderer()

	add := func(name string, files ...string) {
		tpl, err := template.ParseFS(templatesFS, files...)
		if err != nil {
			log.Fatalf("解析模板失败： %s: %v", name, err)
		}
		r.Add(name, tpl)
	}

	add("index.html", "base.html", "index.html", "_pagination.html")
	add("post.html", "base.html", "post.html")
	add("admin.html", "base.html", "admin.html", "_pagination.html")
	add("editor.html", "base.html", "editor.html")
	add("settings.html", "base.html", "settings.html")
	add("login.html", "base.html", "login.html")
	add("search.html", "base.html", "search.html", "_pagination.html")
	add("404.html", "base.html", "404.html")
	add("error.html", "base.html", "error.html")

	return r
}

func main() {
	// Asset loading is now handled automatically by build tags.
	unsafe := flag.Bool("unsafe", false, "allow insecure cookies")
	flag.Parse()

	// 初始化数据库
	db, err := utils.InitDatabase()
	if err != nil {
		log.Fatal("初始化数据库失败：", err)
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
	apiHandler := handlers.NewAPIHandler(postService)

	// 设置Gin路由
	r := gin.Default()
	r.HTMLRender = createRenderer()

	// 设置会话中间件
	store := cookie.NewStore([]byte("secret-key-should-be-changed"))
	store.Options(sessions.Options{
		HttpOnly: true,
		Secure:   !*unsafe,
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("glog_session", store))

	// 加载设置的中间件
	r.Use(handlers.SettingsMiddleware(settingService))

	// 静态文件服务
	r.StaticFS("/static", http.FS(staticFS))

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
		admin.POST("/delete/:id", adminHandler.DeletePost)
	}

	settings := r.Group("/settings")
	settings.Use(handlers.AuthMiddleware())
	{
		settings.GET("/", adminHandler.ShowSettingsPage)
		settings.POST("/", adminHandler.UpdateSettings)
		settings.POST("/test-ai", adminHandler.TestAISettings)
	}
	// API 路由
	api := r.Group("/api/v1")
	api.Use(handlers.APIAuthMiddleware(settingService))
	{
		api.POST("/posts", apiHandler.CreatePost)
		api.GET("/posts", apiHandler.FindPosts)
	}

	// 404处理
	r.NoRoute(blogHandler.NotFound)

	// 启动服务器
	log.Println("服务器启动于 :37371")
	r.Run(":37371")
}
