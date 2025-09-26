package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"glog/internal/constants"
	"glog/internal/models"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v39/github"
	"github.com/yeka/zip"
	"golang.org/x/oauth2"
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

	delete(settings, constants.SettingGithubLastBackupHash)
	delete(settings, constants.SettingWebdavLastBackupHash)

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

func (s *BackupService) BackupToGithub(repoName, branch, token string) error {
	backupData, newHash, err := s.generateBackupDataAndHash()
	if err != nil {
		return err
	}

	lastHash, _ := s.SettingService.GetSetting(constants.SettingGithubLastBackupHash)
	if newHash == lastHash {
		return ErrBackupNoChange
	}

	backupContent, err := s.createEncryptedBackup(backupData)
	if err != nil {
		return fmt.Errorf("创建备份文件失败: %w", err)
	}

	parts := strings.Split(repoName, "/")
	if len(parts) != 2 {
		return fmt.Errorf("无效的仓库名格式")
	}
	owner, repo := parts[0], parts[1]
	path := fmt.Sprintf("glog_backup_%s.zip", time.Now().Format("20060102150405"))
	message := "Automated backup from Glog"

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	opts := &github.RepositoryContentFileOptions{
		Message: &message,
		Content: backupContent,
		Branch:  &branch,
	}

	_, _, err = client.Repositories.CreateFile(ctx, owner, repo, path, opts)
	if err != nil {
		fileContent, _, _, getErr := client.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{Ref: branch})
		if getErr != nil {
			return fmt.Errorf("创建文件失败，并且无法获取现有文件信息: %w", err)
		}
		opts.SHA = fileContent.SHA
		_, _, updateErr := client.Repositories.UpdateFile(ctx, owner, repo, path, opts)
		if updateErr != nil {
			return fmt.Errorf("尝试更新 GitHub 文件也失败了: %w", updateErr)
		}
	}

	return s.SettingService.UpdateSettings(map[string]string{
		constants.SettingGithubLastBackupHash: newHash,
	})
}

func (s *BackupService) BackupToWebdav(url, user, password string) error {
	backupData, newHash, err := s.generateBackupDataAndHash()
	if err != nil {
		return err
	}

	lastHash, _ := s.SettingService.GetSetting(constants.SettingWebdavLastBackupHash)
	if newHash == lastHash {
		return ErrBackupNoChange
	}

	backupContent, err := s.createEncryptedBackup(backupData)
	if err != nil {
		return fmt.Errorf("创建备份文件失败: %w", err)
	}

	fileName := fmt.Sprintf("glog_backup_%s.zip", time.Now().Format("20060102150405"))
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	fullURL := url + fileName

	req, err := http.NewRequest(http.MethodPut, fullURL, bytes.NewReader(backupContent))
	if err != nil {
		return fmt.Errorf("创建 WebDAV 请求失败: %w", err)
	}

	if user != "" && password != "" {
		req.SetBasicAuth(user, password)
	}
	req.Header.Set("Content-Type", "application/zip")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("上传到 WebDAV 服务器失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("WebDAV 服务器返回错误状态: %s, 响应: %s", resp.Status, string(body))
	}

	return s.SettingService.UpdateSettings(map[string]string{
		constants.SettingWebdavLastBackupHash: newHash,
	})
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

func (s *BackupService) TestGithubConnection(repoName, token string) error {
	if repoName == "" || token == "" {
		return fmt.Errorf("仓库名和 Token 不能为空")
	}

	parts := strings.Split(repoName, "/")
	if len(parts) != 2 {
		return fmt.Errorf("无效的仓库名格式，应为 'user/repo'")
	}
	owner, repo := parts[0], parts[1]

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	_, _, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		_, _, userErr := client.Users.Get(ctx, "")
		if userErr != nil {
			return fmt.Errorf("无法访问 GitHub 仓库，且 Token 无效: %v", userErr)
		}
		return fmt.Errorf("无法访问 GitHub 仓库 (请检查仓库名称和权限): %v", err)
	}

	return nil
}

func (s *BackupService) TestWebdavConnection(url, user, password string) error {
	if url == "" {
		return fmt.Errorf("服务器地址不能为空")
	}

	req, err := http.NewRequest("OPTIONS", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	if user != "" && password != "" {
		req.SetBasicAuth(user, password)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("连接 WebDAV 服务器失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("WebDAV 服务器返回错误状态: %s", resp.Status)
}
