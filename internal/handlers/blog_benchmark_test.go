package handlers

import (
	"glog/internal/models"
	"glog/internal/repository"
	"glog/internal/services"
	"glog/internal/utils"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

// setupTestRouter initializes a gin router with all the necessary dependencies for testing.
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	db, err := utils.InitDatabase()
	if err != nil {
		panic("Failed to initialize database for testing: " + err.Error())
	}

	postRepo := repository.NewPostRepository(db)
	settingRepo := repository.NewSettingRepository(db)

	settingService := services.NewSettingService(settingRepo)
	aiService := services.NewAIService()
	postService := services.NewPostService(postRepo, settingService, aiService)

	blogHandler := NewBlogHandler(postService)
	searchHandler := NewSearchHandler(postService)

	r := gin.New()
	r.GET("/", blogHandler.Index)
	r.GET("/post/:slug", blogHandler.ShowPost)
	r.GET("/search", searchHandler.Search)

	return r
}

// BenchmarkGetIndex performs a benchmark test on the Index handler.
func BenchmarkGetIndex(b *testing.B) {
	router := setupTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		router.ServeHTTP(w, req)
	}
}

// BenchmarkGetPost performs a benchmark test on the ShowPost handler with random posts.
func BenchmarkGetPost(b *testing.B) {
	router := setupTestRouter()

	// Pre-fetch a list of valid post slugs to avoid 404s in benchmark
	db, _ := utils.InitDatabase()
	var slugs []string
	db.Model(&models.Post{}).Select("slug").Find(&slugs)
	if len(slugs) == 0 {
		b.Fatal("No posts found in the database. Please seed the database first.")
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Pick a random slug for each request
		randomSlug := slugs[rand.Intn(len(slugs))]
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/post/"+randomSlug, nil)
		router.ServeHTTP(w, req)
	}
}

// BenchmarkSearch performs a benchmark test on the Search handler.
func BenchmarkSearch(b *testing.B) {
	router := setupTestRouter()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Search for a common term that should exist in many posts
		query := "性能测试文章 " + strconv.Itoa(rand.Intn(1000)+1)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/search?q="+query, nil)
		router.ServeHTTP(w, req)
	}
}
