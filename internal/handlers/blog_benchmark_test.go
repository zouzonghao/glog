package handlers

import (
	"glog/internal/models"
	"glog/internal/repository"
	"glog/internal/services"
	"glog/internal/utils"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

var (
	testRouter *gin.Engine
	once       sync.Once
)

// setupTestRouter initializes a gin router with all the necessary dependencies for testing.
// It also changes the working directory to the project root to ensure relative paths work correctly.
func setupTestRouter() *gin.Engine {
	once.Do(func() {
		// --- Change working directory to project root ---
		_, b, _, _ := runtime.Caller(0)
		root := filepath.Join(filepath.Dir(b), "../../") // Move up two directories from internal/handlers
		if err := os.Chdir(root); err != nil {
			panic("Failed to change dir to project root: " + err.Error())
		}
		// --- End of directory change ---

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

		testRouter = r
	})
	return testRouter
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
