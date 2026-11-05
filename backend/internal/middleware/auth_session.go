package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/XiaomaiTX/go-react-starter/internal/problem"
	"github.com/gin-gonic/gin"
)

const SessionCookieName = "tailor_session"

type SessionResolver interface {
	ResolveSession(context.Context, string) (string, bool, error)
}

type identityKey struct{}

func Principal(ctx context.Context) string {
	value, _ := ctx.Value(identityKey{}).(string)
	return value
}

func AuthSession(resolver SessionResolver, origins []string, secureCookie bool) gin.HandlerFunc {
	allowedOrigins := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowedOrigins[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		if isUnsafeMethod(c.Request.Method) {
			origin := c.GetHeader("Origin")
			if origin != "" {
				if _, ok := allowedOrigins[origin]; !ok {
					problem.Write(c, problem.Forbidden())
					c.Abort()
					return
				}
			}
		}
		if isPublicPath(c.Request.URL.Path) || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		if resolver == nil {
			problem.Write(c, problem.Unavailable("OIDC 身份服务未配置"))
			c.Abort()
			return
		}
		rawSession, err := c.Cookie(SessionCookieName)
		if err != nil || rawSession == "" {
			writeUnauthorized(c, secureCookie, false)
			return
		}
		username, valid, err := resolver.ResolveSession(c.Request.Context(), rawSession)
		if err != nil {
			problem.Write(c, problem.Unavailable("认证服务暂时不可用"))
			c.Abort()
			return
		}
		if !valid || strings.TrimSpace(username) == "" {
			writeUnauthorized(c, secureCookie, true)
			return
		}
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), identityKey{}, username))
		c.Next()
	}
}

func writeUnauthorized(c *gin.Context, secureCookie, clearCookie bool) {
	if clearCookie {
		http.SetCookie(c.Writer, &http.Cookie{
			Name: SessionCookieName, Value: "", Path: "/api/v1",
			Expires: time.Unix(1, 0), MaxAge: -1,
			HttpOnly: true, Secure: secureCookie, SameSite: http.SameSiteLaxMode,
		})
	}
	problem.Write(c, problem.Unauthorized())
	c.Abort()
}

func isPublicPath(path string) bool {
	switch path {
	case "/api/v1/health/live", "/api/v1/health/ready",
		"/api/v1/auth/login", "/api/v1/auth/callback", "/api/v1/auth/logout":
		return true
	default:
		return false
	}
}

func isUnsafeMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}
