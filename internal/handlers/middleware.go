package handlers

import (
	"log"
	"net/http"

	"glog/internal/services"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware checks if a user is authenticated via session flag.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		authenticated := session.Get("authenticated")

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
			log.Printf("Could not load settings: %v", err)
			c.Set("settings", make(map[string]string))
		} else {
			c.Set("settings", settings)
		}

		// Also, add the login status to the context for the template.
		session := sessions.Default(c)
		isLoggedIn := session.Get("authenticated")
		c.Set("IsLoggedIn", isLoggedIn != nil && isLoggedIn.(bool))

		c.Next()
	}
}

// render is a helper function to render templates with common data.
func render(c *gin.Context, status int, templateName string, data gin.H) {
	// Get settings from context
	settings, exists := c.Get("settings")
	if exists {
		// Merge settings into the data map
		for key, value := range settings.(map[string]string) {
			if _, ok := data[key]; !ok { // Don't overwrite existing data
				data[key] = value
			}
		}
	}

	// Get login status from context
	isLoggedIn, exists := c.Get("IsLoggedIn")
	if exists {
		data["IsLoggedIn"] = isLoggedIn
	}

	c.HTML(status, templateName, data)
}
