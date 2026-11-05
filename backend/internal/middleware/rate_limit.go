package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/XiaomaiTX/go-react-starter/internal/problem"
	"github.com/gin-gonic/gin"
)

type requestWindow struct {
	started time.Time
	count   int
}

type requestLimiter struct {
	mu      sync.Mutex
	windows map[string]requestWindow
}

// APIRateLimit bounds authenticated mutations and OIDC entry points per user or
// network peer. It is process-local; deployments with multiple API replicas
// should also apply an edge or shared-store limit.
func APIRateLimit() gin.HandlerFunc {
	limiter := &requestLimiter{windows: make(map[string]requestWindow)}
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		loginAttempt := path == "/api/v1/auth/login" || path == "/api/v1/auth/callback"
		if !loginAttempt && !isUnsafeMethod(c.Request.Method) {
			c.Next()
			return
		}

		limit := 60
		key := Principal(c.Request.Context())
		if loginAttempt || key == "" {
			limit = 20
			key = remoteHost(c.Request.RemoteAddr)
		}
		if loginAttempt {
			key = "oidc:" + key
		} else {
			key = "mutation:" + key
		}

		now := time.Now()
		if !limiter.allow(key, limit, now) {
			c.Header("Retry-After", "60")
			problem.Write(c, problem.New(http.StatusTooManyRequests, "rate-limit", "请求过于频繁", nil))
			c.Abort()
			return
		}
		c.Next()
	}
}

func (limiter *requestLimiter) allow(key string, limit int, now time.Time) bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if len(limiter.windows) >= 4096 {
		for candidate, window := range limiter.windows {
			if now.Sub(window.started) >= time.Minute {
				delete(limiter.windows, candidate)
			}
		}
	}
	window, ok := limiter.windows[key]
	if !ok && len(limiter.windows) >= 4096 {
		return false
	}
	if !ok || now.Sub(window.started) >= time.Minute || now.Before(window.started) {
		limiter.windows[key] = requestWindow{started: now, count: 1}
		return true
	}
	if window.count >= limit {
		return false
	}
	window.count++
	limiter.windows[key] = window
	return true
}

func remoteHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil && strings.TrimSpace(host) != "" {
		return host
	}
	return remoteAddr
}
