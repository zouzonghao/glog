package utils

import (
	"glog/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func InitDatabase() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("blog.db"), &gorm.Config{})
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
	ftsTableSQL := `
	CREATE VIRTUAL TABLE IF NOT EXISTS posts_fts USING fts5(
		title,
		content,
		content='posts',
		content_rowid='id'
	);`
	if err := db.Exec(ftsTableSQL).Error; err != nil {
		return nil, err
	}

	// 2. Create triggers to keep FTS table synchronized with posts table
	triggers := []string{
		`
		CREATE TRIGGER IF NOT EXISTS posts_ai AFTER INSERT ON posts BEGIN
			INSERT INTO posts_fts(rowid, title, content) VALUES (new.id, new.title, new.content);
		END;`,
		`
		CREATE TRIGGER IF NOT EXISTS posts_ad AFTER DELETE ON posts BEGIN
			INSERT INTO posts_fts(posts_fts, rowid, title, content) VALUES ('delete', old.id, old.title, old.content);
		END;`,
		`
		CREATE TRIGGER IF NOT EXISTS posts_au AFTER UPDATE ON posts BEGIN
			INSERT INTO posts_fts(posts_fts, rowid, title, content) VALUES ('delete', old.id, old.title, old.content);
			INSERT INTO posts_fts(rowid, title, content) VALUES (new.id, new.title, new.content);
		END;`,
	}

	for _, trigger := range triggers {
		if err := db.Exec(trigger).Error; err != nil {
			return nil, err
		}
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
		"openai_model":     "gpt-3.5-turbo",
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
