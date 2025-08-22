package handlers

import (
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
	// 从会话中获取并清除 flash 消息
	session := sessions.Default(c)
	flashes := session.Flashes("error")
	session.Save() // 确保 flash 消息被清除

	render(c, http.StatusOK, "login.html", gin.H{
		"error": flashes,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	session := sessions.Default(c)
	submittedPassword := c.PostForm("password")

	adminPassword, err := h.settingService.GetSetting("password")
	if err != nil {
		session.AddFlash("服务器内部错误", "error")
		session.Save()
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if submittedPassword != adminPassword {
		session.AddFlash("密码错误！请重新输入。", "error")
		session.Save()
		c.Redirect(http.StatusFound, "/login")
		return
	}

	session.Set("authenticated", true)
	session.Save()
	c.Redirect(http.StatusFound, "/admin/")
}

func (h *AuthHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/login")
}
