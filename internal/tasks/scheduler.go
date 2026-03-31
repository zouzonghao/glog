package tasks

import (
	"context"
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
	syncService    *services.SyncService
	mu             sync.Mutex
}

func NewScheduler(settingService *services.SettingService, syncService *services.SyncService) *Scheduler {
	return &Scheduler{
		cron:           cron.New(),
		settingService: settingService,
		syncService:    syncService,
	}
}

func (s *Scheduler) Start() {
	log.Println("定时同步调度器正在初始化...")
	s.ReloadTasks()
	s.cron.Start()
}

func (s *Scheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	if s.cron == nil {
		s.mu.Unlock()
		return nil
	}
	stopCtx := s.cron.Stop()
	s.mu.Unlock()

	select {
	case <-stopCtx.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Scheduler) ReloadTasks() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cron != nil {
		s.cron.Stop()
	}
	s.cron = cron.New()

	settings, err := s.settingService.GetAllSettings()
	if err != nil {
		log.Printf("无法加载设置以重载调度器: %v", err)
		return
	}

	s.addSyncTask(settings)

	if len(s.cron.Entries()) > 0 {
		s.cron.Start()
		log.Println("定时任务已重载并启动。")
	} else {
		log.Println("没有活动的定时任务。")
	}
}

func (s *Scheduler) addSyncTask(settings map[string]string) {
	intervalStr := settings[constants.SettingSyncInterval]
	if intervalStr == "" {
		return
	}

	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval <= 0 {
		return
	}

	spec := fmt.Sprintf("@every %dm", interval)
	job := func() {
		log.Printf("开始执行定时同步...")
		result, err := s.syncService.Sync()
		if err != nil {
			if errors.Is(err, services.ErrSyncNoChange) {
				log.Printf("同步检查：数据无变化，无需同步。")
			} else {
				log.Printf("定时同步失败: %v", err)
			}
		} else {
			log.Printf("定时同步成功！动作: %s, 本地: %d, 远程: %d", result.Action, result.LocalCount, result.RemoteCount)
		}
	}

	_, err = s.cron.AddFunc(spec, recoveryWrapper(job))
	if err != nil {
		log.Printf("添加同步任务失败: %v", err)
	} else {
		log.Printf("已成功安排同步任务，每 %d 分钟执行一次。", interval)
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
