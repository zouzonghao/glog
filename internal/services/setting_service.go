package services

import (
	"glog/internal/repository"
	"log"
	"sync"
)

type SettingService struct {
	repo         *repository.SettingRepository
	settings     map[string]string
	settingsLock sync.RWMutex
}

func NewSettingService(repo *repository.SettingRepository) *SettingService {
	s := &SettingService{
		repo:     repo,
		settings: make(map[string]string),
	}
	s.loadSettings()
	return s
}

func (s *SettingService) loadSettings() {
	s.settingsLock.Lock()
	defer s.settingsLock.Unlock()

	settings, err := s.repo.GetAllSettings()
	if err != nil {
		log.Printf("无法加载设置: %v", err)
		return
	}
	s.settings = settings
}

// GetAllSettings retrieves all settings as a map from the cache.
func (s *SettingService) GetAllSettings() (map[string]string, error) {
	s.settingsLock.RLock()
	defer s.settingsLock.RUnlock()

	// Return a copy to prevent modification of the cache from outside.
	settingsCopy := make(map[string]string)
	for key, value := range s.settings {
		settingsCopy[key] = value
	}
	return settingsCopy, nil
}

// UpdateSettings updates multiple settings at once and refreshes the cache.
func (s *SettingService) UpdateSettings(settings map[string]string) error {
	for key, value := range settings {
		if err := s.repo.UpdateSetting(key, value); err != nil {
			return err
		}
	}
	// Reload settings into cache after update
	s.loadSettings()
	return nil
}

// GetSetting retrieves a single setting value by its key from the cache.
func (s *SettingService) GetSetting(key string) (string, error) {
	s.settingsLock.RLock()
	defer s.settingsLock.RUnlock()
	return s.settings[key], nil
}
