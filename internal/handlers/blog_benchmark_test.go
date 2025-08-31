package handlers

import (
	"glog/internal/constants"
	"glog/internal/models"
	"glog/internal/repository"
	"glog/internal/services"
	"glog/internal/utils"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
)

var (
	testRouter *gin.Engine
	once       sync.Once
)

// createTestRenderer creates a multitemplate renderer for testing.
func createTestRenderer() multitemplate.Renderer {
	r := multitemplate.NewRenderer()
	_, b, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatalf("Failed to get current file path")
	}
	projectRoot := filepath.Join(filepath.Dir(b), "..", "..")
	templatesDir := filepath.Join(projectRoot, "templates")

	add := func(name string, files ...string) {
		for i, f := range files {
			files[i] = filepath.Join(templatesDir, f)
		}
		tpl, err := template.ParseFiles(files...)
		if err != nil {
			log.Fatalf("Failed to parse template %s: %v", name, err)
		}
		r.Add(name, tpl)
	}

	add("index.html", "base.html", "index.html", "_pagination.html")
	add("post.html", "base.html", "post.html")
	add("search.html", "base.html", "search.html", "_pagination.html")

	return r
}

// setupTestRouter initializes a gin router with all the necessary dependencies for testing.
// It's wrapped with sync.Once to ensure it only runs once for all benchmarks.
func setupTestRouter() *gin.Engine {
	once.Do(func() {
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
		r.HTMLRender = createTestRenderer()

		// *** THIS IS THE FIX for the search panic ***
		// Add a mock middleware to set the login status for every request.
		r.Use(func(c *gin.Context) {
			c.Set(constants.ContextKeyIsLoggedIn, false)
			c.Next()
		})

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

	db, _ := utils.InitDatabase()
	var slugs []string
	db.Model(&models.Post{}).Select("slug").Find(&slugs)
	if len(slugs) == 0 {
		b.Fatal("No posts found in the database. Please seed the database first.")
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
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
		query := "性能测试文章 " + strconv.Itoa(rand.Intn(1000)+1)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/search?q="+query, nil)
		router.ServeHTTP(w, req)
	}
}
