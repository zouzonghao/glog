# AGENTS.md - Guidelines for AI Coding Agents

Guidelines for AI agents working on the Glog codebase, a lightweight Go blog system.

## Build & Run Commands

```bash
# Development
make run              # Run in development mode
go run .              # Alternative development run
go mod tidy           # Update dependencies

# Production Build
make build-platform-with-cleanup                      # Current platform
GOOS=linux GOARCH=amd64 make build-platform-with-cleanup
make release-all                                      # All platforms
make clean                                            # Clean artifacts

# Testing
go test ./...                                         # Run all tests
go test ./internal/services/...                       # Tests in a package
go test -v ./... -run TestName                        # Single test by name
go test -v ./internal/handlers -run TestBlogHandler   # Single test in package
go test -cover ./...                                  # With coverage
go test -race ./...                                   # With race detector
```

## Project Structure

```
├── main.go              # Entry point, router setup
├── internal/
│   ├── constants/       # Application-wide constants
│   ├── handlers/        # HTTP request handlers (Gin)
│   ├── services/        # Business logic layer
│   ├── repository/      # Data access layer (GORM)
│   ├── models/          # Data models/structs
│   ├── tasks/           # Background tasks (scheduler)
│   └── utils/           # Utility functions
├── static/              # Static assets (CSS, JS, images)
├── templates/           # HTML templates
└── Makefile             # Build automation
```

## Code Style

### Formatting
- Use `gofmt` (tabs for indentation), run `go fmt ./...` before committing
- Line length: ~120 characters (soft limit)

### Import Organization
Three groups separated by blank lines:
```go
import (
    "fmt"                          // 1. Standard library

    "glog/internal/services"       // 2. Internal packages

    "github.com/gin-gonic/gin"     // 3. External dependencies
)
```

### Naming Conventions
- **Types/Functions**: PascalCase exported, camelCase unexported
- **Receivers**: Short lowercase: `r` (repo), `s` (service), `h` (handler)
- **Constructors**: `NewXxxService()`, `NewXxxHandler()`, `NewXxxRepository()`
- **Constants**: PascalCase with category prefix: `SettingPassword`, `ContextKeyIsLoggedIn`

### Handler Pattern
```go
type BlogHandler struct {
    postService *services.PostService
}

func NewBlogHandler(postService *services.PostService) *BlogHandler {
    return &BlogHandler{postService: postService}
}

func (h *BlogHandler) Index(c *gin.Context) {
    posts, total, err := h.postService.GetPostsPage(page, pageSize, isLoggedIn)
    if err != nil {
        render(c, http.StatusInternalServerError, "404.html", gin.H{"error": "加载文章失败"})
        return
    }
    render(c, http.StatusOK, "index.html", gin.H{"posts": posts})
}
```

### Service/Repository Pattern
```go
type PostService struct {
    repo           *repository.PostRepository
    settingService *SettingService
}

func (r *PostRepository) FindByID(id uint) (*models.Post, error) {
    var post models.Post
    err := r.db.First(&post, id).Error
    return &post, err
}
```

## Error Handling

```go
// Error wrapping with %w
return nil, fmt.Errorf("渲染文章失败: %w", err)

// Sentinel errors for expected conditions
var ErrBackupNoChange = errors.New("backup: no changes detected")

// HTTP error responses (consistent structure)
c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的文章 ID"})

// Success responses
c.JSON(http.StatusOK, gin.H{"status": "success", "message": "文章已保存！", "post_id": post.ID})
```

## Constants Usage

Always use constants from `internal/constants/keys.go`:
```go
constants.ContextKeyIsLoggedIn    // Context keys
constants.SessionKeyAuthenticated // Session keys
constants.SettingPassword         // Setting keys
constants.SettingOpenAIBaseURL
```

## Database (GORM)

```go
err := r.db.Create(post).Error                                        // Create
err := r.db.Save(post).Error                                          // Update all
err := r.db.Model(&models.Post{}).Where("id = ?", id).Updates(f).Error // Partial update
query := r.db.Where("is_private = ?", false).Order("published_at desc")
err := query.Offset((page-1) * pageSize).Limit(pageSize).Find(&posts).Error
```

## Comments

- Chinese for business logic comments (this is a Chinese-language project)
- English for GoDoc-style documentation on exported types/functions

```go
// AIService handles interactions with an OpenAI compatible API.
type AIService struct { ... }

// 摘要严格限制50字以内，需简短精炼
```

## Template Rendering

Use the shared `render()` helper:
```go
render(c, http.StatusOK, "template.html", gin.H{"data": value})
```

## Concurrency

For background AI operations, use goroutines with locking:
```go
s.LockPost(post.ID)
go func() {
    defer s.UnlockPost(post.ID)
    // Background work
}()
```

## Common Gotchas

1. **Timezone**: Always use `Asia/Shanghai` for time operations
2. **CGO**: Pure-Go SQLite (`glebarez/sqlite`), no CGO required
3. **Build Tags**: Use `-tags release` for production to embed assets
4. **Assets**: Static assets embedded in release builds via `assets_prod.go`
