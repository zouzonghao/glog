package utils

import (
	"glog/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func InitDatabase(dbPath string) (*gorm.DB, error) {
	if dbPath == "" {
		dbPath = "blog.db"
	}
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 自动迁移模式
	err = db.AutoMigrate(&models.Post{}, &models.Setting{})
	if err != nil {
		return nil, err
	}

	// --- FTS5 Setup ---
	// 1. Create FTS virtual table
	// We are using a normal FTS table, not a contentless one,
	// because we need to pass pre-segmented text from our Go application.
	// Triggers are also removed as the application layer will handle synchronization.
	ftsTableSQL := `
	CREATE VIRTUAL TABLE IF NOT EXISTS posts_fts USING fts5(
		title,
		content
	);`
	if err := db.Exec(ftsTableSQL).Error; err != nil {
		return nil, err
	}

	// Seed the database with initial settings
	if err := seedSettings(db); err != nil {
		return nil, err
	}

	return db, nil
}

// seedSettings populates the database with default settings if they don't exist.
func seedSettings(db *gorm.DB) error {
	defaultSettings := map[string]string{
		"password":         "admin",
		"site_logo":        "",
		"site_description": "由 Glog 驱动的博客",
		"openai_base_url":  "",
		"openai_token":     "",
		"openai_model":     "gemini-2.5-flash",
		"search_engine":    "like", // "like" or "fts5"
	}

	for key, value := range defaultSettings {
		setting := models.Setting{Key: key}
		result := db.FirstOrCreate(&setting, models.Setting{Key: key})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			// Only set the value if the record was just created
			setting.Value = value
			db.Save(&setting)
		}
	}

	return nil
}
