package services

import (
	"context"
	"errors"
	"fmt"
	"glog/internal/constants"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// CoverTask* 表示内存中的任务生命周期状态，仅用于任务服务内部。
// 数据库持久化状态请使用 CoverStatus* 常量。
const (
	CoverTaskPending   = "pending"
	CoverTaskRunning   = "running"
	CoverTaskSuccess   = "success"
	CoverTaskFailed    = "failed"
	CoverTaskCancelled = "cancelled"
)

// 封面任务配置常量
const (
	defaultCoverTaskTimeout    = 10 * time.Minute // 封面任务默认超时时间
	coverTaskCleanupInterval   = 3 * time.Minute  // 任务清理检查间隔
	coverTaskRetentionDuration = 20 * time.Minute // 终态任务保留时长

	danglingCoverTaskFailedError = "服务重启导致封面任务中断，请重新生成。"
	shutdownCancelledError       = "服务正在关闭，封面任务已取消"
)

type CoverTask struct {
	ID           string    `json:"id"`
	PostID       uint      `json:"post_id"`
	Status       string    `json:"status"`
	Cover        string    `json:"cover,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CoverTaskService struct {
	mu                sync.RWMutex
	tasks             map[string]*CoverTask
	currentTaskByPost map[uint]string
	cancelFuncs       map[string]context.CancelFunc
	seq               uint64
	taskTimeout       time.Duration

	rootCtx    context.Context
	rootCancel context.CancelFunc
	wg         sync.WaitGroup
	stopOnce   sync.Once

	settingService *SettingService
	aiService      *AIService
	postService    *PostService
}

func NewCoverTaskService(settingService *SettingService, aiService *AIService, postService *PostService) *CoverTaskService {
	rootCtx, rootCancel := context.WithCancel(context.Background())
	s := &CoverTaskService{
		tasks:             make(map[string]*CoverTask),
		currentTaskByPost: make(map[uint]string),
		cancelFuncs:       make(map[string]context.CancelFunc),
		taskTimeout:       defaultCoverTaskTimeout,
		rootCtx:           rootCtx,
		rootCancel:        rootCancel,
		settingService:    settingService,
		aiService:         aiService,
		postService:       postService,
	}

	s.wg.Add(1)
	go s.cleanupLoop()
	return s
}

func (s *CoverTaskService) RecoverDanglingTasks() error {
	log.Println("开始执行封面任务状态恢复检查...")

	count, err := s.postService.ResolveDanglingCoverTasks(danglingCoverTaskFailedError)
	if err != nil {
		return err
	}
	legacyCount, legacyErr := s.postService.ResolveLegacyCoverMarkers(danglingCoverTaskFailedError)
	if legacyErr != nil {
		return legacyErr
	}
	if count > 0 {
		log.Printf("已修复 %d 条遗留的封面生成状态", count)
	}
	if legacyCount > 0 {
		log.Printf("已清理 %d 条历史 marker 封面数据", legacyCount)
	}
	if count == 0 && legacyCount == 0 {
		log.Println("封面任务状态恢复检查完成，无需修复。")
	}
	return nil
}

func (s *CoverTaskService) CreateTask(postID uint, prompt, content string) (*CoverTask, error) {
	if s.rootCtx.Err() != nil {
		return nil, errors.New("服务正在关闭，无法创建新的封面任务")
	}

	now := time.Now()
	taskID := fmt.Sprintf("cover-%d-%d", now.UnixNano(), atomic.AddUint64(&s.seq, 1))
	task := &CoverTask{
		ID:        taskID,
		PostID:    postID,
		Status:    CoverTaskPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	ctx, cancel := context.WithCancel(s.rootCtx)

	var oldTaskID string
	var oldCancel context.CancelFunc
	var oldTask *CoverTask

	s.mu.Lock()
	if s.rootCtx.Err() != nil {
		s.mu.Unlock()
		cancel()
		return nil, errors.New("服务正在关闭，无法创建新的封面任务")
	}

	if postID > 0 {
		if currentID, ok := s.currentTaskByPost[postID]; ok {
			oldTaskID = currentID
			oldCancel = s.cancelFuncs[currentID]
			oldTask = s.tasks[currentID]
		}
		s.currentTaskByPost[postID] = taskID
	}

	s.cancelFuncs[taskID] = cancel
	s.tasks[taskID] = task

	if postID > 0 {
		if err := s.postService.StartCoverTask(postID, taskID); err != nil {
			delete(s.cancelFuncs, taskID)
			delete(s.tasks, taskID)
			if oldTaskID != "" {
				s.currentTaskByPost[postID] = oldTaskID
			} else {
				delete(s.currentTaskByPost, postID)
			}
			s.mu.Unlock()
			cancel()
			return nil, fmt.Errorf("写入封面任务状态失败: %w", err)
		}

		if oldTaskID != "" {
			delete(s.cancelFuncs, oldTaskID)
			if oldTask != nil && (oldTask.Status == CoverTaskPending || oldTask.Status == CoverTaskRunning) {
				oldTask.Status = CoverTaskCancelled
				oldTask.ErrorMessage = "已被新的封面任务替代"
				oldTask.UpdatedAt = now
			}
		}
	}
	s.wg.Add(1)
	s.mu.Unlock()

	if oldCancel != nil {
		oldCancel()
	}
	if oldTaskID != "" && postID > 0 {
		if _, err := s.postService.FailCoverTaskIfCurrent(postID, oldTaskID, "已被新的封面任务替代", CoverStatusCancelled); err != nil {
			log.Printf("取消旧封面任务状态写库失败 (task_id=%s): %v", oldTaskID, err)
		}
	}

	go func() {
		defer s.wg.Done()
		s.runTask(ctx, taskID, postID, prompt, content)
	}()

	log.Printf("封面任务创建成功 (task_id=%s, post_id=%d)", taskID, postID)
	return cloneCoverTask(task), nil
}

func (s *CoverTaskService) CancelTaskByPostID(postID uint) {
	var taskID string
	var cancel context.CancelFunc

	s.mu.Lock()
	if currentID, ok := s.currentTaskByPost[postID]; ok {
		taskID = currentID
		cancel = s.cancelFuncs[currentID]
		delete(s.cancelFuncs, currentID)
		delete(s.currentTaskByPost, postID)

		if task, exists := s.tasks[currentID]; exists &&
			(task.Status == CoverTaskPending || task.Status == CoverTaskRunning) {
			task.Status = CoverTaskCancelled
			task.ErrorMessage = "用户手动更新了封面，任务已取消"
			task.UpdatedAt = time.Now()
		}
	}
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if taskID != "" {
		if _, err := s.postService.FailCoverTaskIfCurrent(postID, taskID, "用户手动更新了封面，任务已取消", CoverStatusCancelled); err != nil {
			log.Printf("取消封面任务状态写库失败 (task_id=%s): %v", taskID, err)
		}
	}
}

func (s *CoverTaskService) Shutdown(ctx context.Context) error {
	s.stopOnce.Do(func() {
		s.rootCancel()

		type cancelItem struct {
			taskID string
			postID uint
			cancel context.CancelFunc
		}
		var items []cancelItem

		s.mu.Lock()
		now := time.Now()
		for taskID, cancel := range s.cancelFuncs {
			postID := uint(0)
			if task, ok := s.tasks[taskID]; ok {
				postID = task.PostID
				if task.Status == CoverTaskPending || task.Status == CoverTaskRunning {
					task.Status = CoverTaskCancelled
					task.ErrorMessage = shutdownCancelledError
					task.UpdatedAt = now
				}
			}
			items = append(items, cancelItem{
				taskID: taskID,
				postID: postID,
				cancel: cancel,
			})
		}
		s.cancelFuncs = make(map[string]context.CancelFunc)
		s.currentTaskByPost = make(map[uint]string)
		s.mu.Unlock()

		for _, item := range items {
			if item.cancel != nil {
				item.cancel()
			}
			if item.postID > 0 {
				if _, err := s.postService.FailCoverTaskIfCurrent(item.postID, item.taskID, shutdownCancelledError, CoverStatusCancelled); err != nil {
					log.Printf("服务关闭时写入封面任务取消状态失败 (task_id=%s): %v", item.taskID, err)
				}
			}
		}
	})

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *CoverTaskService) GetTask(taskID string) (*CoverTask, bool) {
	s.mu.RLock()
	task, ok := s.tasks[taskID]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}

	return cloneCoverTask(task), true
}

func (s *CoverTaskService) runTask(ctx context.Context, taskID string, postID uint, prompt, content string) {
	s.markRunning(taskID)

	defer func() {
		s.mu.Lock()
		delete(s.cancelFuncs, taskID)
		s.mu.Unlock()
	}()

	if !s.isCurrentTask(taskID, postID) {
		log.Printf("封面任务已被替换，跳过执行 (task_id=%s, post_id=%d)", taskID, postID)
		s.markCancelled(taskID, "任务已被更新任务替换")
		return
	}

	taskCtx, cancel := context.WithTimeout(ctx, s.taskTimeout)
	defer cancel()

	settings, err := s.settingService.GetAllSettings()
	if err != nil {
		s.markAndPersistFailed(taskID, postID, "获取 AI 设置失败: "+err.Error(), CoverStatusFailed)
		return
	}

	coverURL, err := s.aiService.GenerateCover(
		taskCtx,
		prompt,
		content,
		settings[constants.SettingOpenAIBaseURL],
		settings[constants.SettingOpenAIToken],
		settings[constants.SettingOpenAIModel],
		settings[constants.SettingImageAPIURL],
		settings[constants.SettingImageAPIToken],
		settings[constants.SettingImageAPIModel],
	)
	if err != nil {
		if taskCtx.Err() == context.Canceled {
			s.markAndPersistFailed(taskID, postID, "任务已取消", CoverStatusCancelled)
			return
		}
		if taskCtx.Err() == context.DeadlineExceeded {
			s.markAndPersistFailed(taskID, postID, "AI 封面生成超时，请稍后重试", CoverStatusFailed)
			return
		}
		s.markAndPersistFailed(taskID, postID, "AI 封面生成失败: "+err.Error(), CoverStatusFailed)
		return
	}

	if postID > 0 {
		if !s.isCurrentTask(taskID, postID) {
			log.Printf("封面任务结果被跳过 (task_id=%s, post_id=%d)", taskID, postID)
			s.markCancelled(taskID, "任务结果已被更新的封面状态忽略")
			return
		}

		updated, updateErr := s.postService.FinishCoverTaskIfCurrent(postID, taskID, coverURL)
		if updateErr != nil {
			s.markAndPersistFailed(taskID, postID, "写入文章封面失败: "+updateErr.Error(), CoverStatusFailed)
			return
		}
		if !updated {
			log.Printf("封面任务结果 CAS 未命中，结果已忽略 (task_id=%s, post_id=%d)", taskID, postID)
			s.markCancelled(taskID, "任务结果已被更新的封面状态忽略")
			return
		}
	}

	s.markSuccess(taskID, postID, coverURL)
}

func (s *CoverTaskService) markAndPersistFailed(taskID string, postID uint, message, failedStatus string) {
	if strings.TrimSpace(failedStatus) == CoverStatusCancelled {
		s.markCancelled(taskID, message)
	} else {
		s.markFailed(taskID, postID, message)
	}

	if postID == 0 {
		return
	}

	updated, err := s.postService.FailCoverTaskIfCurrent(postID, taskID, message, failedStatus)
	if err != nil {
		log.Printf("更新封面任务失败状态失败 (task_id=%s): %v", taskID, err)
		return
	}
	if !updated {
		log.Printf("封面任务失败状态 CAS 未命中，状态已忽略 (task_id=%s, post_id=%d)", taskID, postID)
		s.markCancelled(taskID, "任务结果已被更新的封面状态忽略")
	}
}

func (s *CoverTaskService) isCurrentTask(taskID string, postID uint) bool {
	if postID == 0 {
		return true
	}

	s.mu.RLock()
	currentTaskID, ok := s.currentTaskByPost[postID]
	s.mu.RUnlock()
	return ok && currentTaskID == taskID
}

func (s *CoverTaskService) markRunning(taskID string) {
	s.updateTask(taskID, func(task *CoverTask, now time.Time) {
		task.Status = CoverTaskRunning
		task.UpdatedAt = now
	})
}

func (s *CoverTaskService) markSuccess(taskID string, postID uint, coverURL string) {
	s.updateTask(taskID, func(task *CoverTask, now time.Time) {
		task.Status = CoverTaskSuccess
		task.Cover = coverURL
		task.ErrorMessage = ""
		task.UpdatedAt = now
		if postID > 0 && s.currentTaskByPost[postID] == taskID {
			delete(s.currentTaskByPost, postID)
		}
	})
}

func (s *CoverTaskService) markFailed(taskID string, postID uint, message string) {
	s.updateTask(taskID, func(task *CoverTask, now time.Time) {
		task.Status = CoverTaskFailed
		task.ErrorMessage = message
		task.UpdatedAt = now
		if postID > 0 && s.currentTaskByPost[postID] == taskID {
			delete(s.currentTaskByPost, postID)
		}
	})
}

func (s *CoverTaskService) markCancelled(taskID string, message string) {
	s.updateTask(taskID, func(task *CoverTask, now time.Time) {
		task.Status = CoverTaskCancelled
		if strings.TrimSpace(message) == "" {
			task.ErrorMessage = "已被新的封面任务替代"
		} else {
			task.ErrorMessage = message
		}
		task.UpdatedAt = now
		if task.PostID > 0 && s.currentTaskByPost[task.PostID] == taskID {
			delete(s.currentTaskByPost, task.PostID)
		}
	})
}

func (s *CoverTaskService) updateTask(taskID string, updater func(task *CoverTask, now time.Time)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return
	}

	updater(task, time.Now())
}

func (s *CoverTaskService) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(coverTaskCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupTasks()
		case <-s.rootCtx.Done():
			return
		}
	}
}

func (s *CoverTaskService) cleanupTasks() {
	cutoff := time.Now().Add(-coverTaskRetentionDuration)

	s.mu.Lock()
	defer s.mu.Unlock()

	for id, task := range s.tasks {
		if (task.Status == CoverTaskSuccess || task.Status == CoverTaskFailed || task.Status == CoverTaskCancelled) && task.UpdatedAt.Before(cutoff) {
			delete(s.tasks, id)
			delete(s.cancelFuncs, id)
			if task.PostID > 0 && s.currentTaskByPost[task.PostID] == id {
				delete(s.currentTaskByPost, task.PostID)
			}
		}
	}

	// 防御性清理：移除指向不存在任务或终态任务的索引，避免残留引用长期占用内存
	for postID, taskID := range s.currentTaskByPost {
		task, exists := s.tasks[taskID]
		if !exists {
			delete(s.currentTaskByPost, postID)
			delete(s.cancelFuncs, taskID)
			continue
		}
		if task.Status == CoverTaskSuccess || task.Status == CoverTaskFailed || task.Status == CoverTaskCancelled {
			delete(s.currentTaskByPost, postID)
			delete(s.cancelFuncs, taskID)
		}
	}
}

func cloneCoverTask(task *CoverTask) *CoverTask {
	if task == nil {
		return nil
	}
	copy := *task
	return &copy
}
