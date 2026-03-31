package services

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"glog/internal/constants"
	"glog/internal/models"
	"sort"

	"github.com/yeka/zip"
)

var ErrBackupNoChange = errors.New("数据无变化，无需备份")

type BackupService struct {
	PostService    *PostService
	SettingService *SettingService
}

func NewBackupService(postService *PostService, settingService *SettingService) *BackupService {
	return &BackupService{
		PostService:    postService,
		SettingService: settingService,
	}
}

func (s *BackupService) generateBackupDataAndHash() (*models.SiteBackup, string, error) {
	posts, err := s.PostService.GetAllPostsForBackup()
	if err != nil {
		return nil, "", fmt.Errorf("获取文章失败: %w", err)
	}

	settings, err := s.SettingService.GetAllSettings()
	if err != nil {
		return nil, "", fmt.Errorf("获取设置失败: %w", err)
	}

	delete(settings, constants.SettingDBModifiedAt)

	backupData := &models.SiteBackup{
		Posts:    posts,
		Settings: settings,
	}

	keys := make([]string, 0, len(settings))
	for k := range settings {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	stableSettings := make(map[string]string)
	for _, k := range keys {
		stableSettings[k] = settings[k]
	}
	backupData.Settings = stableSettings

	jsonData, err := json.Marshal(backupData)
	if err != nil {
		return nil, "", fmt.Errorf("JSON 序列化失败: %w", err)
	}

	hash := sha256.Sum256(jsonData)
	return backupData, hex.EncodeToString(hash[:]), nil
}

func (s *BackupService) createEncryptedBackup(backupData *models.SiteBackup) ([]byte, error) {
	password, err := s.SettingService.GetSetting(constants.SettingPassword)
	if err != nil {
		return nil, fmt.Errorf("获取站点密码失败: %w", err)
	}
	if password == "" {
		return nil, fmt.Errorf("站点密码未设置，无法创建加密备份")
	}

	jsonData, err := json.MarshalIndent(backupData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("JSON 序列化失败: %w", err)
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)
	zipFile, err := zipWriter.Encrypt("backup.json", password, zip.AES256Encryption)
	if err != nil {
		return nil, fmt.Errorf("创建加密 ZIP 文件失败: %w", err)
	}
	_, err = zipFile.Write(jsonData)
	if err != nil {
		return nil, fmt.Errorf("写入 ZIP 文件失败: %w", err)
	}
	zipWriter.Close()

	return buf.Bytes(), nil
}
