package handlers

import (
	"bytes"
	"encoding/json"
	"glog/internal/constants"
	"glog/internal/models"
	"glog/internal/repository"
	"glog/internal/services"
	"glog/internal/utils"
	"html/template"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"github.com/yeka/zip"
	"gorm.io/gorm"
)

// testContext holds all shared resources for the benchmark tests.
type testContext struct {
	router     *gin.Engine
	db         *gorm.DB
	backupData []byte
}

var tCtx testContext

// TestMain sets up the entire test suite before any benchmarks are run.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	// Setup router and database
	setupTestRouterAndDB()

	// Create a large backup file for the restore benchmark
	if err := createTestBackupFile(); err != nil {
		log.Fatalf("Failed to create test backup file: %v", err)
	}

	// Run all benchmarks
	os.Exit(m.Run())
}

// setupTestRouterAndDB initializes the router and DB connection once.
func setupTestRouterAndDB() {
	// --- Get project root and construct absolute DB path ---
	_, b, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(b), "..", "..")
	dbPath := filepath.Join(projectRoot, "blog.db")
	os.Setenv("DB_PATH", dbPath)
	// --- End of path construction ---

	db, err := utils.InitDatabase() // Use absolute path
	if err != nil {
		panic("Failed to initialize database for testing: " + err.Error())
	}
	tCtx.db = db

	postRepo := repository.NewPostRepository(tCtx.db)
	settingRepo := repository.NewSettingRepository(tCtx.db)

	settingService := services.NewSettingService(settingRepo)
	aiService := services.NewAIService()
	postService := services.NewPostService(postRepo, settingService, aiService)
	backupService := services.NewBackupService(postService, settingService)

	blogHandler := NewBlogHandler(postService)
	searchHandler := NewSearchHandler(postService)
	adminHandler := NewAdminHandler(postService, settingService, aiService, backupService, nil)

	r := gin.New()
	r.HTMLRender = createTestRenderer()

	r.Use(func(c *gin.Context) {
		c.Set(constants.ContextKeyIsLoggedIn, false)
		c.Next()
	})

	r.GET("/", blogHandler.Index)
	r.GET("/post/:slug", blogHandler.ShowPost)
	r.GET("/search", searchHandler.Search)
	r.POST("/admin/upload", adminHandler.UploadBackup)

	tCtx.router = r
}

// createTestBackupFile generates a large backup file from the seeded data.
func createTestBackupFile() error {
	postRepo := repository.NewPostRepository(tCtx.db)
	settingRepo := repository.NewSettingRepository(tCtx.db)
	postService := services.NewPostService(postRepo, nil, nil)
	settingService := services.NewSettingService(settingRepo)

	posts, err := postService.GetAllPostsForBackup()
	if err != nil {
		return err
	}
	settings, err := settingService.GetAllSettings()
	if err != nil {
		return err
	}

	backupData := &models.SiteBackup{Posts: posts, Settings: settings}
	jsonData, err := json.Marshal(backupData)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)
	zipFile, err := zipWriter.Encrypt("backup.json", "admin", zip.AES256Encryption)
	if err != nil {
		return err
	}
	_, err = zipFile.Write(jsonData)
	if err != nil {
		return err
	}
	zipWriter.Close()

	tCtx.backupData = buf.Bytes()
	log.Printf("Created a test backup file with size: %d bytes", len(tCtx.backupData))
	return nil
}

func BenchmarkGetIndex(b *testing.B) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tCtx.router.ServeHTTP(w, req)
	}
}

func BenchmarkGetPost(b *testing.B) {
	var slugs []string
	tCtx.db.Model(&models.Post{}).Select("slug").Find(&slugs)
	if len(slugs) == 0 {
		b.Fatal("No posts found in the database. Please seed the database first.")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		randomSlug := slugs[rand.Intn(len(slugs))]
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/post/"+randomSlug, nil)
		tCtx.router.ServeHTTP(w, req)
	}
}

func BenchmarkSearch(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := "性能测试文章 " + strconv.Itoa(rand.Intn(1000)+1)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/search?q="+query, nil)
		tCtx.router.ServeHTTP(w, req)
	}
}

func BenchmarkRestoreBackup(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("backup", "test_backup.zip")
		part.Write(tCtx.backupData)
		writer.Close()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/admin/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		tCtx.router.ServeHTTP(w, req)
	}
}

func createTestRenderer() multitemplate.Renderer {
	r := multitemplate.NewRenderer()
	_, f, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatalf("Failed to get current file path")
	}
	projectRoot := filepath.Join(filepath.Dir(f), "..", "..")
	templatesDir := filepath.Join(projectRoot, "templates")

	add := func(name string, files ...string) {
		for i, file := range files {
			files[i] = filepath.Join(templatesDir, file)
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
	add("admin.html", "base.html", "admin.html", "_pagination.html")

	return r
}
