package http

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/time/rate"

	ilogger "github.com/your-org/ventopanel/internal/infra/logger"
)

// ── Per-IP rate limiter ───────────────────────────────────────────────────────

type ipLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

func newIPLimiter(r rate.Limit, b int) *ipLimiter {
	return &ipLimiter{limiters: make(map[string]*rate.Limiter), r: r, b: b}
}

func (l *ipLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	if lim, ok := l.limiters[ip]; ok {
		return lim
	}
	lim := rate.NewLimiter(l.r, l.b)
	l.limiters[ip] = lim
	return lim
}

// globalLimiter — 60 req/min per IP across all endpoints.
var globalLimiter = newIPLimiter(rate.Every(time.Second), 20)

// loginLimiter — 5 req/min per IP, protects /auth/login and /auth/mfa.
var loginLimiter = newIPLimiter(rate.Every(12*time.Second), 5)

// RateLimitMiddleware applies a per-IP token-bucket rate limit.
// Use globalLimiter for general routes, loginLimiter for auth routes.
func RateLimitMiddleware(lim *ipLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !lim.get(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests — please slow down",
			})
			return
		}
		c.Next()
	}
}

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
