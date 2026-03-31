package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"glog/internal/constants"
	"glog/internal/models"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yeka/zip"
)

var (
	ErrSyncNoChange      = errors.New("同步:数据无变化")
	ErrSyncNotConfigured = errors.New("同步：WebDAV 未配置")
	zipFileRegex         = regexp.MustCompile(`/(\d+)\.zip$`)
)

type SyncResult struct {
	Action      string   // "upload", "download", "merge", "none"
	LocalCount  int      // 本地文章数
	RemoteCount int      // 远程文章数
	Uploaded    int      // 上传文章数
	Downloaded  int      // 下载文章数
	Conflicts   []string // 冲突的文章 slug
}

type SyncService struct {
	postService    *PostService
	settingService *SettingService
	client         *http.Client
}

func NewSyncService(postService *PostService, settingService *SettingService) *SyncService {
	return &SyncService{
		postService:    postService,
		settingService: settingService,
		client: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
				},
			},
		},
	}
}

func (s *SyncService) Sync() (*SyncResult, error) {
	baseURL, user, password, err := s.getWebdavConfig()
	if err != nil {
		return nil, err
	}

	passwordHash, err := s.settingService.GetSetting(constants.SettingPassword)
	if err != nil || passwordHash == "" {
		return nil, errors.New("站点密码未设置")
	}

	remoteTimestamp, err := s.getRemoteTimestamp(baseURL, user, password)
	if err != nil {
		return nil, fmt.Errorf("获取远程时间戳失败: %w", err)
	}

	localTimestamp := s.getDBModifiedAt()

	localPosts, err := s.postService.GetAllPostsForBackup()
	if err != nil {
		return nil, fmt.Errorf("获取本地文章失败: %w", err)
	}

	result := &SyncResult{
		LocalCount: len(localPosts),
	}

	if remoteTimestamp.IsZero() {
		if err := s.uploadAll(baseURL, user, password, passwordHash, localPosts, localTimestamp); err != nil {
			return nil, err
		}
		result.Action = "upload"
		result.Uploaded = len(localPosts)
		return result, nil
	}

	if localTimestamp.Equal(remoteTimestamp) {
		result.Action = "none"
		return result, ErrSyncNoChange
	}

	if localTimestamp.After(remoteTimestamp) {
		if err := s.uploadAll(baseURL, user, password, passwordHash, localPosts, localTimestamp); err != nil {
			return nil, err
		}
		result.Action = "upload"
		result.Uploaded = len(localPosts)
		return result, nil
	}

	remotePosts, err := s.downloadAndDecrypt(baseURL, user, password, passwordHash)
	if err != nil {
		return nil, fmt.Errorf("下载远程数据失败: %w", err)
	}

	result.RemoteCount = len(remotePosts)

	merged, conflicts := s.mergePosts(localPosts, remotePosts)
	result.Conflicts = conflicts

	if err := s.importPosts(merged); err != nil {
		return nil, fmt.Errorf("导入文章失败: %w", err)
	}

	s.updateDBModifiedAt(remoteTimestamp)

	result.Action = "download"
	result.Downloaded = len(merged)
	return result, nil
}

func (s *SyncService) getWebdavConfig() (baseURL, user, password string, err error) {
	settings, err := s.settingService.GetAllSettings()
	if err != nil {
		return "", "", "", err
	}

	baseURL = settings[constants.SettingWebdavURL]
	if baseURL == "" {
		return "", "", "", ErrSyncNotConfigured
	}

	user = settings[constants.SettingWebdavUser]
	password = settings[constants.SettingWebdavPassword]
	return baseURL, user, password, nil
}

func (s *SyncService) getDBModifiedAt() time.Time {
	val, err := s.settingService.GetSetting(constants.SettingDBModifiedAt)
	if err != nil || val == "" {
		return time.Time{}
	}

	nano, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return time.Time{}
	}

	return time.Unix(0, nano)
}

func (s *SyncService) updateDBModifiedAt(t time.Time) {
	s.settingService.UpdateSettings(map[string]string{
		constants.SettingDBModifiedAt: strconv.FormatInt(t.UnixNano(), 10),
	})
}

type propfindResponse struct {
	XMLName   xml.Name   `xml:"multistatus"`
	Responses []response `xml:"response"`
}

type response struct {
	Href string `xml:"href"`
}

func (s *SyncService) getRemoteTimestamp(baseURL, user, password string) (time.Time, error) {
	syncPath := strings.TrimSuffix(baseURL, "/") + "/"

	req, err := http.NewRequest("PROPFIND", syncPath, bytes.NewReader([]byte(propfindBody)))
	if err != nil {
		return time.Time{}, err
	}
	req.SetBasicAuth(user, password)
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	resp, err := s.client.Do(req)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusConflict {
		return time.Time{}, nil
	}

	if resp.StatusCode != http.StatusMultiStatus && resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("PROPFIND 失败: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return time.Time{}, err
	}

	var pf propfindResponse
	if err := xml.Unmarshal(body, &pf); err != nil {
		return time.Time{}, err
	}

	var maxTimestamp int64

	for _, r := range pf.Responses {
		decodedHref, err := url.PathUnescape(r.Href)
		if err != nil {
			continue
		}

		matches := zipFileRegex.FindStringSubmatch(decodedHref)
		if len(matches) == 2 {
			timestamp, err := strconv.ParseInt(matches[1], 10, 64)
			if err == nil && timestamp > maxTimestamp {
				maxTimestamp = timestamp
			}
		}
	}

	if maxTimestamp == 0 {
		return time.Time{}, nil
	}

	return time.Unix(0, maxTimestamp), nil
}

const propfindBody = `<?xml version="1.0" encoding="utf-8"?>
<propfind xmlns="DAV:">
  <prop>
    <displayname/>
    <getlastmodified/>
  </prop>
</propfind>`

func (s *SyncService) uploadAll(baseURL, user, password, passwordHash string, posts []models.PostBackup, timestamp time.Time) error {
	syncPath := strings.TrimSuffix(baseURL, "/")

	zipFileName := fmt.Sprintf("%d.zip", timestamp.UnixNano())
	remotePath := syncPath + "/" + zipFileName

	if err := s.uploadStream(baseURL, user, password, passwordHash, posts, remotePath); err != nil {
		return fmt.Errorf("上传失败: %w", err)
	}

	if err := s.cleanupOldBackups(baseURL, user, password); err != nil {
		return fmt.Errorf("清理旧备份失败: %w", err)
	}

	return nil
}

func (s *SyncService) uploadStream(baseURL, user, password, passwordHash string, posts []models.PostBackup, remotePath string) error {
	pr, pw := io.Pipe()

	go func() {
		var err error
		defer func() {
			if err != nil {
				pw.CloseWithError(err)
			} else {
				pw.Close()
			}
		}()

		zipWriter := zip.NewWriter(pw)
		defer func() {
			if cerr := zipWriter.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}()

		var jsonData []byte
		jsonData, err = json.Marshal(posts)
		if err != nil {
			return
		}

		var writer io.Writer
		writer, err = zipWriter.Encrypt("posts.json", passwordHash, zip.AES256Encryption)
		if err != nil {
			return
		}

		_, err = writer.Write(jsonData)
		if err != nil {
			return
		}
	}()

	req, err := http.NewRequest(http.MethodPut, remotePath, pr)
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, password)
	req.Header.Set("Content-Type", "application/zip")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("上传失败: %s, %s", resp.Status, string(body))
	}

	return nil
}

func (s *SyncService) cleanupOldBackups(baseURL, user, password string) error {
	maxBackups := 100
	val, err := s.settingService.GetSetting(constants.SettingSyncMaxBackups)
	if err == nil && val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			maxBackups = n
		}
	}

	syncPath := strings.TrimSuffix(baseURL, "/") + "/"

	req, err := http.NewRequest("PROPFIND", syncPath, bytes.NewReader([]byte(propfindBody)))
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, password)
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var pf propfindResponse
	if err := xml.Unmarshal(body, &pf); err != nil {
		return err
	}

	type backupFile struct {
		path      string
		timestamp int64
	}

	var backups []backupFile

	for _, r := range pf.Responses {
		decodedHref, err := url.PathUnescape(r.Href)
		if err != nil {
			continue
		}

		matches := zipFileRegex.FindStringSubmatch(decodedHref)
		if len(matches) == 2 {
			timestamp, err := strconv.ParseInt(matches[1], 10, 64)
			if err == nil {
				backups = append(backups, backupFile{
					path:      decodedHref,
					timestamp: timestamp,
				})
			}
		}
	}

	if len(backups) <= maxBackups {
		return nil
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].timestamp > backups[j].timestamp
	})

	baseURLParsed, _ := url.Parse(baseURL)

	for i := maxBackups; i < len(backups); i++ {
		var deleteURL string
		backupPath := backups[i].path

		if strings.HasPrefix(backupPath, "http://") || strings.HasPrefix(backupPath, "https://") {
			deleteURL = backupPath
		} else if strings.HasPrefix(backupPath, "/") {
			deleteURL = baseURLParsed.Scheme + "://" + baseURLParsed.Host + backupPath
		} else {
			deleteURL = strings.TrimSuffix(baseURL, "/") + "/" + backupPath
		}

		req, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
		if err != nil {
			continue
		}
		req.SetBasicAuth(user, password)

		resp, err := s.client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
	}

	return nil
}

func (s *SyncService) downloadAndDecrypt(baseURL, user, password, passwordHash string) ([]models.PostBackup, error) {
	syncPath := strings.TrimSuffix(baseURL, "/") + "/"

	remoteTimestamp, err := s.getRemoteTimestamp(baseURL, user, password)
	if err != nil {
		return nil, err
	}

	if remoteTimestamp.IsZero() {
		return nil, errors.New("远程没有备份数据")
	}

	zipFileName := fmt.Sprintf("%d.zip", remoteTimestamp.UnixNano())
	downloadURL := syncPath + zipFileName

	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, password)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载失败: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("无效的 ZIP 文件: %w", err)
	}

	if len(reader.File) == 0 {
		return nil, errors.New("ZIP 文件为空")
	}

	zipFile := reader.File[0]
	zipFile.SetPassword(passwordHash)

	rc, err := zipFile.Open()
	if err != nil {
		return nil, fmt.Errorf("解密失败，请检查站点密码: %w", err)
	}
	defer rc.Close()

	var posts []models.PostBackup
	if err := json.NewDecoder(rc).Decode(&posts); err != nil {
		return nil, fmt.Errorf("解析数据失败: %w", err)
	}

	return posts, nil
}

func (s *SyncService) mergePosts(local, remote []models.PostBackup) ([]models.PostBackup, []string) {
	merged := make(map[string]models.PostBackup)
	localMap := make(map[string]models.PostBackup)
	remoteMap := make(map[string]models.PostBackup)

	for _, p := range local {
		localMap[p.Slug] = p
		merged[p.Slug] = p
	}

	for _, p := range remote {
		remoteMap[p.Slug] = p
	}

	var conflicts []string
	for slug, remotePost := range remoteMap {
		if localPost, exists := localMap[slug]; exists {
			if !s.postsEqual(localPost, remotePost) {
				conflicts = append(conflicts, slug)
				if remotePost.PublishedAt.After(localPost.PublishedAt) {
					merged[slug] = remotePost
				}
			}
		} else {
			merged[slug] = remotePost
		}
	}

	result := make([]models.PostBackup, 0, len(merged))
	for _, p := range merged {
		result = append(result, p)
	}

	return result, conflicts
}

func (s *SyncService) postsEqual(a, b models.PostBackup) bool {
	return a.Title == b.Title &&
		a.Tag == b.Tag &&
		a.Cover == b.Cover &&
		a.Content == b.Content &&
		a.IsPrivate == b.IsPrivate &&
		a.PublishedAt.Equal(b.PublishedAt)
}

func (s *SyncService) importPosts(posts []models.PostBackup) error {
	return s.postService.UpsertPostsFromBackup(posts)
}

func (s *SyncService) TestConnection() (map[string]interface{}, error) {
	baseURL, user, password, err := s.getWebdavConfig()
	if err != nil {
		return nil, err
	}

	syncPath := strings.TrimSuffix(baseURL, "/") + "/"

	req, err := http.NewRequest("PROPFIND", syncPath, bytes.NewReader([]byte(propfindBody)))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, password)
	req.Header.Set("Depth", "0")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("认证失败")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("连接失败: %d", resp.StatusCode)
	}

	remoteTimestamp, err := s.getRemoteTimestamp(baseURL, user, password)
	if err != nil {
		return nil, fmt.Errorf("获取远程状态失败: %w", err)
	}

	localTimestamp := s.getDBModifiedAt()

	status := "synced"
	if remoteTimestamp.IsZero() {
		status = "no_remote"
	} else if localTimestamp.After(remoteTimestamp) {
		status = "local_ahead"
	} else if remoteTimestamp.After(localTimestamp) {
		status = "remote_ahead"
	}

	return map[string]interface{}{
		"configured":         true,
		"status":             status,
		"local_modified_at":  localTimestamp.Format(time.RFC3339),
		"remote_modified_at": remoteTimestamp.Format(time.RFC3339),
	}, nil
}
