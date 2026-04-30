package http

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	ilogger "github.com/your-org/ventopanel/internal/infra/logger"
)

// CORSMiddleware allows browser-direct requests (e.g. from EventSource/WebSocket
// connecting to port 8080 while the UI is served from port 3000). Reflected-origin
// is used so that both http and https frontends are supported without wildcards
// breaking credentialed requests.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = "*"
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, X-User-ID, X-Team-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

func LoggerMiddleware(logger ilogger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		logger.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("duration", time.Since(startedAt)).
			Msg("http request")
	}
}
