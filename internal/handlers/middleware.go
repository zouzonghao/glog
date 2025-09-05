package handlers

import (
	"log"
	"net/http"
	"strings"

	"glog/internal/constants"
	"glog/internal/services"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// CacheControlMiddleware adds Cache-Control headers to static assets.
func CacheControlMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Header("Cache-Control", "public, max-age=31536000") // Cache for 1 year
		}
		c.Next()
	}
}

// PageCacheMiddleware adds Cache-Control headers to public pages.
func PageCacheMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=3600") // Cache for 1 hour
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

		if parts[1] != adminPassword {
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
			// Log the error but don't block the request.
			// The application can run with default settings.
			log.Printf("无法加载设置: %v", err)
			c.Set("settings", make(map[string]string))
		} else {
			c.Set(constants.ContextKeySettings, settings)
		}

		// Also, add the login status to the context for the template.
		session := sessions.Default(c)
		isLoggedIn := session.Get(constants.SessionKeyAuthenticated)
		c.Set(constants.ContextKeyIsLoggedIn, isLoggedIn != nil && isLoggedIn.(bool))

		c.Next()
	}
}

// render is a helper function to render templates with common data.
func render(c *gin.Context, status int, templateName string, data gin.H) {
	// Get settings from context
	settings, exists := c.Get(constants.ContextKeySettings)
	if exists {
		// Merge settings into the data map
		for key, value := range settings.(map[string]string) {
			if _, ok := data[key]; !ok { // Don't overwrite existing data
				if key == constants.SettingFavicon {
					data[key] = value
				} else {
					data[key] = value
				}
			}
		}
	}

	// Get login status from context
	isLoggedIn, exists := c.Get(constants.ContextKeyIsLoggedIn)
	if exists {
		data["IsLoggedIn"] = isLoggedIn
	}

	c.HTML(status, templateName, data)
}
