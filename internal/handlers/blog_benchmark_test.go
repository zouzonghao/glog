package handlers

import (
	"glog/internal/models"
	"glog/internal/repository"
	"glog/internal/services"
	"glog/internal/utils"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

var testRouter *gin.Engine

// TestMain is executed before any other test in this package.
// It's used here to set up the global test environment, specifically
// to change the working directory to the project root.
func TestMain(m *testing.M) {
	// Change working directory to project root
	// This is crucial for the segmenter to find its dictionary files in dev mode.
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	// We expect to be in internal/handlers, so we go up two levels.
	projectRoot := filepath.Join(dir, "..", "..")
	if err := os.Chdir(projectRoot); err != nil {
		log.Fatalf("Failed to change directory to project root: %v", err)
	}

	// Now that the CWD is correct, setup the router
	setupTestRouter()

	// Run all tests
	os.Exit(m.Run())
}

// setupTestRouter initializes a gin router with all the necessary dependencies for testing.
// This function should only be called once by TestMain.
func setupTestRouter() {
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
}

// BenchmarkGetIndex performs a benchmark test on the Index handler.
func BenchmarkGetIndex(b *testing.B) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		testRouter.ServeHTTP(w, req)
	}
}

// BenchmarkGetPost performs a benchmark test on the ShowPost handler with random posts.
func BenchmarkGetPost(b *testing.B) {
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
		testRouter.ServeHTTP(w, req)
	}
}

// BenchmarkSearch performs a benchmark test on the Search handler.
func BenchmarkSearch(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Search for a common term that should exist in many posts
		query := "性能测试文章 " + strconv.Itoa(rand.Intn(1000)+1)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/search?q="+query, nil)
		testRouter.ServeHTTP(w, req)
	}
}
