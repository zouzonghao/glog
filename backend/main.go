package main

import (
	"flag"
	"glog/internal/handlers"
	"glog/internal/repository"
	"glog/internal/services"
	"glog/internal/tasks"
	"glog/internal/utils"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var IsRelease bool

func main() {
	if IsRelease {
		gin.SetMode(gin.ReleaseMode)
	}

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	unsafe := flag.Bool("unsafe", false, "allow insecure cookies")
	flag.Parse()

	utils.InitAILogger()
	defer utils.CloseAILogger()

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		utils.CloseAILogger()
		os.Exit(0)
	}()

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

	r := gin.Default()

	// CORS Middleware
	config := cors.DefaultConfig()
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:4321" // 开发环境默认值
	}
	config.AllowOrigins = []string{frontendURL}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	config.AllowCredentials = true
	r.Use(cors.New(config))

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "dev-secret-key-should-be-changed" // 开发环境默认密钥
	}
	store := cookie.NewStore([]byte(sessionSecret))

	cookieDomain := os.Getenv("COOKIE_DOMAIN")
	if cookieDomain == "" {
		cookieDomain = "localhost" // 开发环境默认值
	}
	store.Options(sessions.Options{
		Path:     "/",
		Domain:   cookieDomain,
		HttpOnly: true,
		Secure:   !*unsafe,
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("glog_session", store))

	r.Use(handlers.SettingsMiddleware(settingService))

	// Serve frontend static assets
	r.StaticFS("/assets", http.Dir("./static/assets"))

	// API Routes
	api := r.Group("/api")
	{
		// Auth
		api.POST("/login", authHandler.Login)
		api.POST("/logout", authHandler.Logout)
		api.GET("/logout", authHandler.Logout) // Also accept GET for simple link-based logout
		api.GET("/auth/status", authHandler.AuthStatus)

		// Blog Posts
		api.GET("/posts", blogHandler.GetPosts)
		api.GET("/posts/:slug", blogHandler.GetPostBySlug)
		api.GET("/search", searchHandler.Search)

		// Admin required
		admin := api.Group("/")
		admin.Use(handlers.AuthMiddleware())
		{
			// Posts
			admin.POST("/posts", adminHandler.SavePost)         // Create
			admin.PUT("/posts/:id", adminHandler.SavePost)      // Update
			admin.DELETE("/posts/:id", adminHandler.DeletePost) // Delete
			admin.POST("/posts/batch-update", adminHandler.BatchUpdatePosts)
			admin.GET("/admin/posts", adminHandler.ListPosts)       // Admin list posts
			admin.GET("/admin/posts/:id", adminHandler.GetPostByID) // Admin get post by id

			// Settings
			settings := admin.Group("/settings")
			{
				settings.GET("", adminHandler.GetSettings)
				settings.GET("/", adminHandler.GetSettings)
				settings.POST("/", adminHandler.UpdateSettings)
				settings.POST("/test-ai", adminHandler.TestAISettings)
				settings.POST("/test-imageapi", adminHandler.TestImageAPIHandler)
				settings.POST("/test-github", adminHandler.TestGithubSettings)
				settings.POST("/test-webdav", adminHandler.TestWebdavSettings)
			}

			// Backup
			backup := admin.Group("/backup")
			{
				backup.GET("/", adminHandler.BackupSite)
				backup.POST("/upload", adminHandler.UploadBackup)
				backup.POST("/github-now", adminHandler.BackupToGithubNow)
				backup.POST("/webdav-now", adminHandler.BackupToWebdavNow)
			}

			// AI Logs
			aiLogs := admin.Group("/ai-logs")
			{
				aiLogs.GET("/", adminHandler.GetAILogs)
				aiLogs.POST("/clear", adminHandler.ClearAILogs)
			}
		}

	}

	r.NoRoute(func(c *gin.Context) {
		// For any route not matched by the API, try to serve a corresponding HTML file from Astro's build.
		// This supports Astro's file-based routing for pages like /post/some-slug -> /post/some-slug.html
		// We must check if the file exists to avoid breaking API routes.
		filePath := "./static" + c.Request.URL.Path
		if c.Request.URL.Path == "/" {
			filePath = "./static/index.html"
		} else if _, err := os.Stat(filePath + ".html"); err == nil {
			filePath = filePath + ".html"
		} else {
			// Fallback to index.html for client-side routing or 404.
			filePath = "./static/index.html"
		}

		// Check if the file exists before serving
		if _, err := os.Stat(filePath); err == nil {
			c.File(filePath)
		} else {
			// If no file matches, it's a true 404 for the API.
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		}
	})

	go scheduler.Start()

	log.Println("服务器启动于 :37371")
	r.Run(":37371")
}
