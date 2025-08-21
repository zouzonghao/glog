package main

import (
	"log"

	"glog/internal/handlers"
	"glog/internal/repository"
	"glog/internal/services"
	"glog/internal/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化数据库
	db, err := utils.InitDatabase()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// 初始化依赖
	postRepo := repository.NewPostRepository(db)
	postService := services.NewPostService(postRepo)

	blogHandler := handlers.NewBlogHandler(postService)
	adminHandler := handlers.NewAdminHandler(postService)
	searchHandler := handlers.NewSearchHandler(postService)

	// 设置Gin路由
	r := gin.Default()

	// 加载模板
	// 模板将在每个处理器中单独加载

	// 静态文件服务
	r.Static("/static", "./static")

	// 博客路由
	r.GET("/", blogHandler.Index)
	r.GET("/post/:slug", blogHandler.ShowPost)
	r.GET("/search", searchHandler.Search)

	// 后台路由
	admin := r.Group("/admin")
	{
		admin.GET("/", adminHandler.ListPosts) // 后台文章列表
		admin.GET("/editor", adminHandler.Editor)
		admin.POST("/save", adminHandler.SavePost)
		admin.GET("/delete/:id", adminHandler.DeletePost) // 删除文章
	}

	// 启动服务器
	log.Println("Server starting on :8080")
	r.Run(":8080")
}
