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
		"favicon":          "",
		"site_description": "由 Glog 驱动的博客",
	}

	for key, defaultValue := range defaultSettings {
		var setting models.Setting
		result := db.FirstOrCreate(&setting, models.Setting{Key: key})
		if result.Error != nil {
			return result.Error
		}
		if setting.Value == "" {
			setting.Value = defaultValue
			if err := db.Save(&setting).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
