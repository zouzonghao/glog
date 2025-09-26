package handlers

import (
	"glog/internal/constants"
	"glog/internal/services"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	settingService *services.SettingService
}

func NewAuthHandler(settingService *services.SettingService) *AuthHandler {
	return &AuthHandler{settingService: settingService}
}

func (h *AuthHandler) ShowLoginPage(c *gin.Context) {
	render(c, http.StatusOK, "login.html", gin.H{})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var payload struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "无效的请求",
		})
		return
	}

	session := sessions.Default(c)

	adminPassword, err := h.settingService.GetSetting(constants.SettingPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "服务器内部错误",
		})
		return
	}

	if payload.Password != adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "密码错误，请重新输入！",
		})
		return
	}

	session.Set(constants.SessionKeyAuthenticated, true)
	session.Save()
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/login")
}
