package handlers

import (
	"errors"
	"glog/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SyncHandler struct {
	syncService *services.SyncService
}

func NewSyncHandler(syncService *services.SyncService) *SyncHandler {
	return &SyncHandler{
		syncService: syncService,
	}
}

func (h *SyncHandler) Sync(c *gin.Context) {
	result, err := h.syncService.Sync()
	if err != nil {
		if errors.Is(err, services.ErrSyncNoChange) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "info",
				"message": "数据无变化，无需同步",
				"result":  result,
			})
			return
		}
		if errors.Is(err, services.ErrSyncNotConfigured) {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "WebDAV 未配置",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "同步失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": h.buildSyncMessage(result),
		"result":  result,
	})
}

func (h *SyncHandler) buildSyncMessage(result *services.SyncResult) string {
	switch result.Action {
	case "upload":
		return "已上传本地数据到远程"
	case "download":
		msg := "已从远程同步数据"
		if len(result.Conflicts) > 0 {
			msg += "，存在冲突已按时间戳合并"
		}
		return msg
	case "merge":
		return "数据已合并"
	default:
		return "同步完成"
	}
}

func (h *SyncHandler) TestConnection(c *gin.Context) {
	status, err := h.syncService.TestConnection()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "连接测试失败: " + err.Error(),
		})
		return
	}

	message := "连接成功！"
	if status["status"] != "no_remote" {
		message = "同步状态获取成功！"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": message,
		"data":    status,
	})
}
