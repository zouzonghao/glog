package services

import (
	"glog/internal/repository"
)

type SettingService struct {
	repo *repository.SettingRepository
}

func NewSettingService(repo *repository.SettingRepository) *SettingService {
	return &SettingService{repo: repo}
}

// GetAllSettings retrieves all settings as a map.
func (s *SettingService) GetAllSettings() (map[string]string, error) {
	return s.repo.GetAllSettings()
}

// UpdateSettings updates multiple settings at once.
func (s *SettingService) UpdateSettings(settings map[string]string) error {
	for key, value := range settings {
		if err := s.repo.UpdateSetting(key, value); err != nil {
			return err
		}
	}
	return nil
}

// GetSetting retrieves a single setting value by its key.
func (s *SettingService) GetSetting(key string) (string, error) {
	setting, err := s.repo.GetSettingByKey(key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}
