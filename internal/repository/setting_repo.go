package repository

import (
	"glog/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SettingRepository struct {
	db *gorm.DB
}

func NewSettingRepository(db *gorm.DB) *SettingRepository {
	return &SettingRepository{db: db}
}

// GetSettingByKey retrieves a single setting by its key.
func (r *SettingRepository) GetSettingByKey(key string) (*models.Setting, error) {
	var setting models.Setting
	if err := r.db.Where("key = ?", key).First(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

// GetAllSettings retrieves all settings as a map.
func (r *SettingRepository) GetAllSettings() (map[string]string, error) {
	var settings []models.Setting
	if err := r.db.Find(&settings).Error; err != nil {
		return nil, err
	}

	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.Key] = s.Value
	}
	return settingsMap, nil
}

// UpdateSetting updates or creates a setting.
func (r *SettingRepository) UpdateSetting(key, value string) error {
	setting := models.Setting{Key: key, Value: value}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&setting).Error
}
