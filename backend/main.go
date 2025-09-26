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
)

var IsRelease bool

func main() {
	if IsRelease {
		gin.SetMode(gin.ReleaseMode)
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
	apiHandler := handlers.NewAPIHandler(postService)

	r := gin.Default()

	// CORS Middleware
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"} // 在生产环境中应设置为你的前端域名
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	config.AllowCredentials = true
	r.Use(cors.New(config))

	store := cookie.NewStore([]byte("secret-key-should-be-changed"))
	store.Options(sessions.Options{
		HttpOnly: true,
		Secure:   !*unsafe,
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("glog_session", store))

	r.Use(handlers.SettingsMiddleware(settingService))

	// API Routes
	api := r.Group("/api")
	{
		// Auth
		api.POST("/login", authHandler.Login)
		api.POST("/logout", authHandler.Logout)

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

		// Legacy API for external tools
		v1 := api.Group("/v1")
		v1.Use(handlers.APIAuthMiddleware(settingService))
		{
			v1.POST("/posts", apiHandler.CreatePost)
			v1.GET("/posts", apiHandler.FindPosts)
		}
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
	})

	go scheduler.Start()

	log.Println("服务器启动于 :37371")
	r.Run(":37371")
}
