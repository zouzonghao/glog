package handlers

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// formatSize converts bytes to a human-readable string (B, KB, MB).
func formatSize(size int) string {
	const (
		B  = 1
		KB = 1024 * B
		MB = 1024 * KB
	)
	floatSize := float64(size)
	switch {
	case size >= MB:
		return fmt.Sprintf("%.2fMB", floatSize/MB)
	case size >= KB:
		return fmt.Sprintf("%.2fKB", floatSize/KB)
	default:
		return fmt.Sprintf("%dB", size)
	}
}

// JSONLoggerMiddleware creates a custom JSON-like logger middleware.
func JSONLoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// your custom format
		return fmt.Sprintf("[GIN] %s | %3d | %10v | %13s | %-7s | %-7s %#v\n",
			param.TimeStamp.Format(time.RFC3339),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			formatSize(param.BodySize),
			param.Method,
			param.Path,
		)
	})
}
