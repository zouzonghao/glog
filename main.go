package main

import (
	"flag"
	"glog/internal/handlers"
	"glog/internal/repository"
	"glog/internal/services"
	"glog/internal/tasks"
	"glog/internal/utils"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

var IsRelease bool
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
	add("index_cards.html", "base.html", "index_cards.html", "_pagination.html")
	add("post.html", "base.html", "post.html")
	add("admin.html", "base.html", "admin.html", "_pagination.html")
	add("editor.html", "base.html", "editor.html")
	add("settings.html", "base.html", "settings.html")
	add("login.html", "base.html", "login.html")
	add("search.html", "base.html", "search.html", "_pagination.html")
	add("search_cards.html", "base.html", "search_cards.html", "_pagination.html")
	add("404.html", "base.html", "404.html")

	return r
}

func main() {
	if IsRelease {
		gin.SetMode(gin.ReleaseMode)
	}

	unsafe := flag.Bool("unsafe", false, "allow insecure cookies")
	flag.Parse()

	db, err := utils.InitDatabase()
	if err != nil {
		log.Fatal("初始化数据库失败：", err)
	}

	postRepo := repository.NewPostRepository(db)
	settingRepo := repository.NewSettingRepository(db)

	settingService := services.NewSettingService(settingRepo)

	aiService := services.NewAIService()
	postService := services.NewPostService(postRepo, settingService, aiService)
	backupService := services.NewBackupService(postService, settingService)
	scheduler := tasks.NewScheduler(settingService, backupService)

	blogHandler := handlers.NewBlogHandler(postService)
	adminHandler := handlers.NewAdminHandler(postService, settingService, aiService, backupService, scheduler)
	searchHandler := handlers.NewSearchHandler(postService)
	authHandler := handlers.NewAuthHandler(settingService)
	apiHandler := handlers.NewAPIHandler(postService)

	r := gin.Default()
	r.HTMLRender = createRenderer()

	store := cookie.NewStore([]byte("secret-key-should-be-changed"))
	store.Options(sessions.Options{
		HttpOnly: true,
		Secure:   !*unsafe,
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("glog_session", store))

	r.Use(handlers.SettingsMiddleware(settingService))

	staticGroup := r.Group("/static")
	staticGroup.Use(handlers.CacheControlMiddleware())
	staticGroup.StaticFS("/", http.FS(staticFS))

	r.GET("/favicon.ico", func(c *gin.Context) {
		c.File("./static/pic/favicon.ico")
	})

	r.GET("/", blogHandler.Index)
	r.GET("/post/:slug", blogHandler.ShowPost)
	r.GET("/search", searchHandler.Search)

	r.GET("/login", authHandler.ShowLoginPage)
	r.POST("/login", authHandler.Login)
	r.GET("/logout", authHandler.Logout)

	admin := r.Group("/admin")
	admin.Use(handlers.AuthMiddleware())
	{
		admin.GET("/", adminHandler.ListPosts)
		admin.GET("/new", adminHandler.NewPost)
		admin.GET("/editor", adminHandler.Editor)
		admin.POST("/save", adminHandler.SavePost)
		admin.POST("/delete/:id", adminHandler.DeletePost)
		admin.POST("/posts/batch-update", adminHandler.BatchUpdatePosts)
	}

	settings := r.Group("/admin/setting")
	settings.Use(handlers.AuthMiddleware())
	{
		settings.GET("/", adminHandler.ShowSettingsPage)
		settings.POST("/", adminHandler.UpdateSettings)
		settings.POST("/test-ai", adminHandler.TestAISettings)
		settings.GET("/backup", adminHandler.BackupSite)
		settings.POST("/upload", adminHandler.UploadBackup)
		settings.POST("/test-github", adminHandler.TestGithubSettings)
		settings.POST("/test-webdav", adminHandler.TestWebdavSettings)
		settings.POST("/backup-github-now", adminHandler.BackupToGithubNow)
		settings.POST("/backup-webdav-now", adminHandler.BackupToWebdavNow)
	}
	api := r.Group("/api/v1")
	api.Use(handlers.APIAuthMiddleware(settingService))
	{
		api.POST("/posts", apiHandler.CreatePost)
		api.GET("/posts", apiHandler.FindPosts)
	}

	r.NoRoute(blogHandler.NotFound)

	go scheduler.Start()

	log.Println("服务器启动于 :37371")
	r.Run(":37371")
}
