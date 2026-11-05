package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type sessionResolverFunc func(context.Context, string) (string, bool, error)

func (f sessionResolverFunc) ResolveSession(ctx context.Context, token string) (string, bool, error) {
	return f(ctx, token)
}

func TestAuthSessionRejectsAndAbortsUnauthenticatedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	engine := gin.New()
	engine.Use(AuthSession(sessionResolverFunc(func(context.Context, string) (string, bool, error) {
		t.Fatal("resolver should not be called without session cookie")
		return "", false, nil
	}), nil, true))
	engine.GET("/api/v1/relay/dashboard", func(c *gin.Context) {
		called = true
		c.JSON(http.StatusForbidden, gin.H{"title": "second response"})
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/relay/dashboard", nil))
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not one JSON value: %v; body=%q", err, recorder.Body.String())
	}
	if recorder.Code != http.StatusUnauthorized || called {
		t.Fatalf("status=%d called=%v, want 401 and handler not called", recorder.Code, called)
	}
	if body["status"] != float64(http.StatusUnauthorized) {
		t.Fatalf("unexpected problem response: %v", body)
	}
}

func TestAuthSessionAllowsHealthAndResolvesPrincipal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resolver := sessionResolverFunc(func(_ context.Context, token string) (string, bool, error) {
		if token != "session-token" {
			t.Fatalf("token=%q", token)
		}
		return "alice", true, nil
	})
	engine := gin.New()
	engine.Use(AuthSession(resolver, nil, false))
	engine.GET("/api/v1/health/live", func(c *gin.Context) { c.Status(http.StatusOK) })
	engine.GET("/api/v1/relay/me", func(c *gin.Context) { c.String(http.StatusOK, Principal(c.Request.Context())) })

	health := httptest.NewRecorder()
	engine.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status=%d", health.Code)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/relay/me", nil)
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "session-token"})
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "alice" {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func TestAuthSessionRejectsUntrustedMutationOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(AuthSession(nil, []string{"https://tailor.example.test"}, true))
	engine.POST("/api/v1/auth/logout", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	request.Header.Set("Origin", "https://evil.example.test")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", recorder.Code)
	}
}
