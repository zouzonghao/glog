package handlers

import (
	"crypto/subtle"
	"log"
	"net/http"
	"strings"

	"glog/internal/constants"
	"glog/internal/services"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func CleanSessionCookie() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookieHeader := c.Request.Header.Get("Cookie")
		if cookieHeader != "" {
			cookies := strings.Split(cookieHeader, ";")
			var glogSessValues []string
			var otherCookies []string

			for _, cookie := range cookies {
				cookie = strings.TrimSpace(cookie)
				if strings.HasPrefix(cookie, "glog_sess=") {
					glogSessValues = append(glogSessValues, strings.TrimPrefix(cookie, "glog_sess="))
				} else {
					otherCookies = append(otherCookies, cookie)
				}
			}

			if len(glogSessValues) > 1 {
				var newCookies []string
				if len(otherCookies) > 0 {
					newCookies = append(newCookies, otherCookies...)
				}
				newCookies = append(newCookies, "glog_sess="+glogSessValues[len(glogSessValues)-1])
				c.Request.Header.Set("Cookie", strings.Join(newCookies, "; "))
			}
		}

		c.Next()

		setCookies := c.Writer.Header()["Set-Cookie"]
		lastGlogSessIndex := -1
		for i, sc := range setCookies {
			if strings.HasPrefix(sc, "glog_sess=") {
				lastGlogSessIndex = i
			}
		}

		if lastGlogSessIndex == -1 {
			return
		}

		filtered := make([]string, 0, len(setCookies))
		for i, sc := range setCookies {
			if strings.HasPrefix(sc, "glog_sess=") {
				if i == lastGlogSessIndex {
					filtered = append(filtered, sc)
				}
			} else {
				filtered = append(filtered, sc)
			}
		}

		if len(filtered) == 0 {
			c.Writer.Header().Del("Set-Cookie")
		} else {
			c.Writer.Header()["Set-Cookie"] = filtered
		}
	}
}

// CacheControlMiddleware adds Cache-Control headers to static assets.
func CacheControlMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Header("Cache-Control", "public, max-age=31536000") // Cache for 1 year
		}
		c.Next()
	}
}

// APIAuthMiddleware checks for a valid Bearer token.
func APIAuthMiddleware(settingService *services.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminPassword, err := settingService.GetSetting("password")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
			c.Abort()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "需要 Authorization 请求头"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization 请求头格式必须为 Bearer {token}"})
			c.Abort()
			return
		}

		if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(adminPassword)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的 token"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AuthMiddleware checks if a user is authenticated via session flag.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		authenticated := session.Get(constants.SessionKeyAuthenticated)

		if authenticated == nil || !authenticated.(bool) {
			// User is not logged in, redirect to login page.
			c.Redirect(http.StatusFound, "/login")
			c.Abort() // Prevent further processing
			return
		}

		// User is authenticated, proceed to the next handler.
		c.Next()
	}
}

// SettingsMiddleware loads settings from the database and adds them to the context.
func SettingsMiddleware(settingService *services.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		settings, err := settingService.GetAllSettings()
		if err != nil {
			log.Printf("无法加载设置: %v", err)
			c.Set(constants.ContextKeySettings, make(map[string]string))
		} else {
			c.Set(constants.ContextKeySettings, settings)
		}

		session := sessions.Default(c)
		isLoggedInValue := session.Get(constants.SessionKeyAuthenticated)
		isLoggedIn, _ := isLoggedInValue.(bool)
		c.Set(constants.ContextKeyIsLoggedIn, isLoggedIn)

		c.Next()
	}
}

func render(c *gin.Context, status int, templateName string, data gin.H) {
	settings, exists := c.Get(constants.ContextKeySettings)
	if exists {
		if settingsMap, ok := settings.(map[string]string); ok {
			for key, value := range settingsMap {
				if _, ok := data[key]; !ok {
					data[key] = value
				}
			}
		}
	}

	isLoggedIn, exists := c.Get(constants.ContextKeyIsLoggedIn)
	if exists {
		data["IsLoggedIn"] = isLoggedIn
	}

	c.HTML(status, templateName, data)
}

func GetViewPreference(c *gin.Context) string {
	view := c.Query("view")
	if view == "" {
		if cookie, err := c.Cookie("view"); err == nil {
			view = cookie
		}
	}
	if view == "" {
		ua := strings.ToLower(c.Request.UserAgent())
		if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
			view = "cards"
		} else {
			view = "list"
		}
	}
	if view != "cards" {
		view = "list"
	}
	c.SetCookie("view", view, 3600*24*365, "/", "", false, true)
	return view
}
