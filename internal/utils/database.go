package utils

import (
	"glog/internal/models"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func InitDatabase() (*gorm.DB, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		exePath, err := os.Executable()
		if err != nil {
			return nil, err
		}
		dbPath = filepath.Join(filepath.Dir(exePath), "glog.db")
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
