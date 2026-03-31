# AGENTS.md - Guidelines for AI Coding Agents

Guidelines for AI agents working on the Glog codebase, a lightweight Go blog system.

## Build & Run Commands

```bash
# Development
make run              # Run in development mode
go run .              # Alternative development run
go run . --unsafe     # Run without HTTPS (for local dev)
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
├── main.go              # Entry point, router setup, session config
├── internal/
│   ├── constants/       # Application-wide constants (keys.go)
│   ├── handlers/        # HTTP request handlers (Gin)
│   │   ├── admin.go     # Admin panel, backup/restore
│   │   ├── auth.go      # Login/logout
│   │   ├── middleware.go # Session, auth, cookie cleanup
│   │   └── sync_handler.go # WebDAV sync endpoints
│   ├── services/        # Business logic layer
│   │   ├── post_service.go   # Post CRUD, backup restore
│   │   ├── setting_service.go # Settings with cache
│   │   └── sync_service.go   # WebDAV sync logic
│   ├── repository/      # Data access layer (GORM)
│   │   ├── post_repo.go  # Post queries, batch operations
│   │   └── setting_repo.go
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
- **No comments** unless explicitly requested

### Import Organization
Three groups separated by blank lines:
```go
import (
    "errors"
    "fmt"

    "glog/internal/constants"
    "glog/internal/services"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)
```

### Naming Conventions
- **Types/Functions**: PascalCase exported, camelCase unexported
- **Receivers**: Short lowercase: `r` (repo), `s` (service), `h` (handler)
- **Constructors**: `NewXxxService()`, `NewXxxHandler()`, `NewXxxRepository()`
- **Constants**: PascalCase with category prefix: `SettingPassword`, `ContextKeyIsLoggedIn`

## Security Best Practices

### Type Assertions
Always use safe type assertions to prevent panic:
```go
// Correct
isLoggedIn, _ := isLoggedInValue.(bool)
if settingsMap, ok := settings.(map[string]string); ok {
    // use settingsMap
}

// Wrong - can panic
isLoggedIn := isLoggedInValue.(bool)
```

### Password Comparison
Use `crypto/subtle.ConstantTimeCompare` for password comparison:
```go
import "crypto/subtle"

if subtle.ConstantTimeCompare([]byte(submitted), []byte(stored)) != 1 {
    // password mismatch
}
```

### Session Management
- Session cookie path must be `/` to avoid multiple cookies
- Use `CleanSessionCookie` middleware to handle duplicate session cookies
- Clear session before setting new values on login

### Settings Whitelist
Always validate settings keys against a whitelist:
```go
allowedSettings := map[string]bool{
    constants.SettingPassword:       true,
    constants.SettingWebdavURL:      true,
    // ...
}
for key := range c.Request.PostForm {
    if !allowedSettings[key] {
        continue
    }
}
```

## Database Operations (GORM)

### Basic Operations
```go
err := r.db.Create(post).Error
err := r.db.Save(post).Error
err := r.db.Model(&models.Post{}).Where("id = ?", id).Updates(fields).Error
```

### Batch Upsert (for sync/restore)
Use `clause.OnConflict` for efficient batch operations:
```go
import "gorm.io/gorm/clause"

func (r *PostRepository) UpsertAllPosts(posts []models.Post) error {
    return r.db.Transaction(func(tx *gorm.DB) error {
        return tx.Clauses(clause.OnConflict{
            Columns:   []clause.Column{{Name: "slug"}},
            DoUpdates: clause.AssignmentColumns([]string{"title", "content", ...}),
        }).Create(&posts).Error
    })
}
```

### Transaction for Replace All
```go
func (r *PostRepository) ReplaceAllPosts(posts []models.Post) error {
    return r.db.Transaction(func(tx *gorm.DB) error {
        if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Post{}).Error; err != nil {
            return err
        }
        if len(posts) > 0 {
            return tx.Create(&posts).Error
        }
        return nil
    })
}
```

### Error Handling with GORM
```go
import "errors"
import "gorm.io/gorm"

existingPost, err := s.repo.FindBySlugIgnorePrivacy(slug)
if err != nil {
    if !errors.Is(err, gorm.ErrRecordNotFound) {
        return fmt.Errorf("查询失败: %w", err)
    }
    // Record not found - create new
}
```

## Backup & Restore

### Backup Format
- ZIP file containing `backup.json` with posts and settings
- Encrypted with AES-256 using site password
- Includes both posts and settings for full site backup

### Restore Logic
1. Use `ReplaceAllPosts` for complete replacement (transactional)
2. Use `UpsertAllPosts` for WebDAV sync (merge with existing)
3. Filter out sensitive settings: `password` (if empty), `db_modified_at`
4. Always use transactions for data integrity

### WebDAV Sync
- Timestamp stored as nanoseconds in `db_modified_at` setting
- Local timestamp updated to remote timestamp after download (not `time.Now()`)
- Cleanup old backups based on `sync_max_backups` setting

#### Timestamp Logic
- Remote backup filename: `{timestamp_nanos}.zip` (e.g., `1704067200000000000.zip`)
- When uploading: filename uses `localTimestamp` from `db_modified_at`
- When syncing: `getRemoteTimestamp()` parses filename to get remote timestamp
- No need to update `db_modified_at` after upload because:
  - Remote timestamp comes from filename (equals upload-time local timestamp)
  - If no local changes, `localTimestamp == remoteTimestamp` → skip sync
  - If local changes exist, `localTimestamp > remoteTimestamp` → upload

#### Download & Restore Logic
- When downloading: delete all local posts first, then restore from remote
- After restore: `db_modified_at` must be set to `remoteTimestamp` (from filename)
- Never use `time.Now()` for `db_modified_at` after download - this would cause:
  - Local timestamp > remote timestamp
  - Next sync would incorrectly trigger upload, overwriting remote data

## Time Handling

### UTC Storage
- All times stored as UTC in database
- Frontend converts UTC to local time for display
- Frontend converts local time to UTC before submission

### Frontend Time Conversion (editor.js)
```javascript
// Display: UTC -> Local
const utcDate = new Date(utcTimeStr + 'Z');
publishedAtInput.value = localFormat(utcDate);

// Submit: Local -> UTC
const localDate = new Date(localTimeStr.replace(' ', 'T'));
formData.set('published_at', utcFormat(localDate));
```

## Error Handling

```go
// Error wrapping with %w
return nil, fmt.Errorf("渲染文章失败: %w", err)

// Sentinel errors for expected conditions
var ErrSyncNotConfigured = errors.New("WebDAV 同步未配置")

// HTTP error responses (consistent structure)
c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "无效的文章 ID"})

// Success responses
c.JSON(http.StatusOK, gin.H{"status": "success", "message": "文章已保存！", "post_id": post.ID})
```

## Constants Usage

Always use constants from `internal/constants/keys.go`:
```go
// Context keys
constants.ContextKeyIsLoggedIn
constants.ContextKeySettings

// Session keys
constants.SessionKeyAuthenticated

// Setting keys
constants.SettingPassword
constants.SettingWebdavURL
constants.SettingWebdavUser
constants.SettingWebdavPassword
constants.SettingSyncInterval
constants.SettingSyncMaxBackups
constants.SettingDBModifiedAt
```

## Template Rendering

Use the shared `render()` helper:
```go
render(c, http.StatusOK, "template.html", gin.H{"data": value})
```

## Common Gotchas

1. **Timezone**: All times in UTC, frontend handles conversion
2. **CGO**: Pure-Go SQLite (`glebarez/sqlite`), no CGO required
3. **Build Tags**: Use `-tags release` for production to embed assets
4. **Assets**: Static assets embedded in release builds via `assets_prod.go`
5. **Session Cookies**: Must set `Path: "/"` to avoid multiple cookies
6. **Variable Shadowing**: Be careful with `:=` in nested scopes
7. **Type Assertions**: Always use safe assertion with `,ok` pattern
