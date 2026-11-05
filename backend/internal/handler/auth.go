package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/XiaomaiTX/go-react-starter/internal/middleware"
	"github.com/XiaomaiTX/go-react-starter/internal/problem"
	"github.com/XiaomaiTX/go-react-starter/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const oidcStateCookieName = "tailor_oidc_state"

type OIDCAuth interface {
	BeginLogin(context.Context) (authorizationURL, state string, err error)
	CancelLogin(context.Context, string, string) error
	CompleteLogin(context.Context, string, string, string) (string, time.Time, error)
	RevokeSession(context.Context, string) error
	SecureCookies() bool
}

func RegisterAuthRoutes(engine *gin.Engine, auth OIDCAuth, logger *zap.Logger) {
	group := engine.Group("/api/v1/auth")
	group.GET("/login", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		authorizationURL, state, err := auth.BeginLogin(c.Request.Context())
		if err != nil {
			problem.Write(c, problem.Unavailable("OIDC 登录暂时不可用"))
			c.Abort()
			return
		}
		http.SetCookie(c.Writer, &http.Cookie{
			Name: oidcStateCookieName, Value: state, Path: "/api/v1/auth/callback",
			Expires: time.Now().Add(10 * time.Minute), MaxAge: 600,
			HttpOnly: true, Secure: auth.SecureCookies(), SameSite: http.SameSiteLaxMode,
		})
		c.Redirect(http.StatusFound, authorizationURL)
	})
	group.GET("/callback", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		state, _ := c.Cookie(oidcStateCookieName)
		queryState := c.Query("state")
		if c.Query("error") != "" {
			stage := "provider_rejected"
			if err := auth.CancelLogin(c.Request.Context(), queryState, state); err != nil {
				stage = service.OIDCFailureStage(err)
			}
			logger.Warn("OIDC callback rejected",
				zap.String("stage", stage),
				zap.String("request_id", middleware.GetRequestID(c)),
			)
			clearOIDCStateCookie(c, auth.SecureCookies())
			c.Redirect(http.StatusFound, "/login?error=sso")
			return
		}
		session, expiresAt, err := auth.CompleteLogin(c.Request.Context(), queryState, state, c.Query("code"))
		clearOIDCStateCookie(c, auth.SecureCookies())
		if err != nil {
			logger.Warn("OIDC callback rejected",
				zap.String("stage", service.OIDCFailureStage(err)),
				zap.String("request_id", middleware.GetRequestID(c)),
			)
			c.Redirect(http.StatusFound, "/login?error=sso")
			return
		}
		maxAge := int(time.Until(expiresAt).Seconds())
		if maxAge < 1 {
			maxAge = 1
		}
		http.SetCookie(c.Writer, &http.Cookie{
			Name: middleware.SessionCookieName, Value: session, Path: "/api/v1",
			Expires: expiresAt, MaxAge: maxAge,
			HttpOnly: true, Secure: auth.SecureCookies(), SameSite: http.SameSiteLaxMode,
		})
		c.Redirect(http.StatusFound, "/")
	})
	group.POST("/logout", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		rawSession, _ := c.Cookie(middleware.SessionCookieName)
		if err := auth.RevokeSession(c.Request.Context(), rawSession); err != nil {
			problem.Write(c, problem.Unavailable("无法结束登录会话"))
			c.Abort()
			return
		}
		http.SetCookie(c.Writer, &http.Cookie{
			Name: middleware.SessionCookieName, Value: "", Path: "/api/v1",
			Expires: time.Unix(1, 0), MaxAge: -1,
			HttpOnly: true, Secure: auth.SecureCookies(), SameSite: http.SameSiteLaxMode,
		})
		c.Status(http.StatusNoContent)
	})
}

func clearOIDCStateCookie(c *gin.Context, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: oidcStateCookieName, Value: "", Path: "/api/v1/auth/callback",
		Expires: time.Unix(1, 0), MaxAge: -1,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
}
