package tasks

import (
	"errors"
	"fmt"
	"glog/internal/constants"
	"glog/internal/services"
	"log"
	"runtime/debug"
	"strconv"
	"sync"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron           *cron.Cron
	settingService *services.SettingService
	backupService  *services.BackupService
	mu             sync.Mutex
}

func NewScheduler(settingService *services.SettingService, backupService *services.BackupService) *Scheduler {
	return &Scheduler{
		cron:           cron.New(),
		settingService: settingService,
		backupService:  backupService,
	}
}

func (s *Scheduler) Start() {
	log.Println("定时备份调度器正在初始化...")
	s.ReloadTasks()
	s.cron.Start()
}

func (s *Scheduler) ReloadTasks() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop the old cron scheduler and create a new one
	if s.cron != nil {
		s.cron.Stop()
	}
	s.cron = cron.New()

	settings, err := s.settingService.GetAllSettings()
	if err != nil {
		log.Printf("无法加载设置以重载调度器: %v", err)
		return
	}

	// --- GitHub Scheduler ---
	s.addBackupTask(settings, constants.SettingGithubInterval, "GitHub", func() error {
		repo := settings[constants.SettingGithubRepo]
		branch := settings[constants.SettingGithubBranch]
		token := settings[constants.SettingGithubToken]
		if repo == "" || branch == "" || token == "" {
			return errors.New("备份配置不完整")
		}
		return s.backupService.BackupToGithub(repo, branch, token)
	})

	// --- WebDAV Scheduler ---
	s.addBackupTask(settings, constants.SettingWebdavInterval, "WebDAV", func() error {
		url := settings[constants.SettingWebdavURL]
		user := settings[constants.SettingWebdavUser]
		password := settings[constants.SettingWebdavPassword]
		if url == "" {
			return errors.New("URL 未配置")
		}
		return s.backupService.BackupToWebdav(url, user, password)
	})

	if len(s.cron.Entries()) > 0 {
		s.cron.Start()
		log.Println("定时任务已重载并启动。")
	} else {
		log.Println("没有活动的定时任务。")
	}
}

func (s *Scheduler) addBackupTask(settings map[string]string, intervalKey, taskName string, backupFunc func() error) {
	intervalStr := settings[intervalKey]
	if intervalStr == "" {
		return
	}

	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval <= 0 {
		return
	}

	spec := fmt.Sprintf("@every %dh", interval)
	job := func() {
		log.Printf("开始执行 %s 定时备份...", taskName)
		err := backupFunc()
		if err != nil {
			if errors.Is(err, services.ErrBackupNoChange) {
				log.Printf("%s 备份检查：数据无变化，无需备份。", taskName)
			} else {
				log.Printf("%s 定时备份失败: %v", taskName, err)
			}
		} else {
			log.Printf("%s 定时备份成功！", taskName)
		}
	}

	_, err = s.cron.AddFunc(spec, recoveryWrapper(job))
	if err != nil {
		log.Printf("添加 %s 备份任务失败: %v", taskName, err)
	} else {
		log.Printf("已成功安排 %s 备份任务，每 %d 小时执行一次。", taskName, interval)
	}
}

func recoveryWrapper(job func()) func() {
	return func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("定时任务执行时发生 panic: %v\n%s", r, debug.Stack())
			}
		}()
		job()
	}
}
