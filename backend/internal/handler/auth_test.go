package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/XiaomaiTX/go-react-starter/internal/middleware"
	"github.com/XiaomaiTX/go-react-starter/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type fakeOIDCAuth struct {
	secure      bool
	authorize   string
	state       string
	session     string
	expiresAt   time.Time
	revoked     string
	cancelled   string
	completeErr error
}

func (f *fakeOIDCAuth) BeginLogin(context.Context) (string, string, error) {
	return f.authorize, f.state, nil
}

func (f *fakeOIDCAuth) CancelLogin(_ context.Context, state, cookieState string) error {
	f.cancelled = state + ":" + cookieState
	return nil
}

func (f *fakeOIDCAuth) CompleteLogin(_ context.Context, state, cookieState, code string) (string, time.Time, error) {
	if state != f.state || cookieState != f.state || code != "test-code" {
		return "", time.Time{}, errUnexpectedCallback
	}
	return f.session, f.expiresAt, f.completeErr
}

func (f *fakeOIDCAuth) RevokeSession(_ context.Context, session string) error {
	f.revoked = session
	return nil
}

func (f *fakeOIDCAuth) SecureCookies() bool { return f.secure }

func TestOIDCAuthRoutesSetAndRevokeSecureCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	expiresAt := time.Now().Add(12 * time.Hour).UTC().Truncate(time.Second)
	auth := &fakeOIDCAuth{
		secure: true, authorize: "https://idp.example.test/authorize?state=state-value",
		state: "state-value", session: "opaque-session-value", expiresAt: expiresAt,
	}
	engine := gin.New()
	RegisterAuthRoutes(engine, auth, zap.NewNop())

	login := httptest.NewRecorder()
	engine.ServeHTTP(login, httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil))
	if login.Code != http.StatusFound || login.Header().Get("Location") != auth.authorize {
		t.Fatalf("login status=%d location=%q", login.Code, login.Header().Get("Location"))
	}
	stateCookie := findCookie(t, login.Result().Cookies(), oidcStateCookieName)
	if stateCookie.Value != auth.state || stateCookie.Path != "/api/v1/auth/callback" || !stateCookie.HttpOnly || !stateCookie.Secure || stateCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected state cookie: %+v", stateCookie)
	}

	callbackRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state=state-value&code=test-code", nil)
	callbackRequest.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: auth.state})
	callback := httptest.NewRecorder()
	engine.ServeHTTP(callback, callbackRequest)
	if callback.Code != http.StatusFound || callback.Header().Get("Location") != "/" {
		t.Fatalf("callback status=%d location=%q", callback.Code, callback.Header().Get("Location"))
	}
	sessionCookie := findCookie(t, callback.Result().Cookies(), middleware.SessionCookieName)
	if sessionCookie.Value != auth.session || sessionCookie.Path != "/api/v1" || !sessionCookie.HttpOnly || !sessionCookie.Secure || sessionCookie.SameSite != http.SameSiteLaxMode || !sessionCookie.Expires.Equal(expiresAt) {
		t.Fatalf("unexpected Tailor session cookie: %+v", sessionCookie)
	}
	clearedStateCookie := findCookie(t, callback.Result().Cookies(), oidcStateCookieName)
	if clearedStateCookie.MaxAge >= 0 || !clearedStateCookie.Secure || !clearedStateCookie.HttpOnly {
		t.Fatalf("state cookie was not cleared securely: %+v", clearedStateCookie)
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutRequest.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: auth.session})
	logout := httptest.NewRecorder()
	engine.ServeHTTP(logout, logoutRequest)
	if logout.Code != http.StatusNoContent || auth.revoked != auth.session {
		t.Fatalf("logout status=%d revoked=%q", logout.Code, auth.revoked)
	}
	clearedSessionCookie := findCookie(t, logout.Result().Cookies(), middleware.SessionCookieName)
	if clearedSessionCookie.MaxAge >= 0 || clearedSessionCookie.Path != "/api/v1" || !clearedSessionCookie.Secure || !clearedSessionCookie.HttpOnly || clearedSessionCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie was not cleared securely: %+v", clearedSessionCookie)
	}
}

func TestOIDCAuthCallbackFailureReturnsToLoginAndClearsState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core, logs := observer.New(zap.WarnLevel)
	auth := &fakeOIDCAuth{state: "state-value", secure: false, completeErr: &service.OIDCLoginFailure{Stage: "nonce"}}
	engine := gin.New()
	RegisterAuthRoutes(engine, auth, zap.New(core))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state=state-value&code=test-code", nil)
	request.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: auth.state})
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusFound || recorder.Header().Get("Location") != "/login?error=sso" {
		t.Fatalf("status=%d location=%q", recorder.Code, recorder.Header().Get("Location"))
	}
	clearedStateCookie := findCookie(t, recorder.Result().Cookies(), oidcStateCookieName)
	if clearedStateCookie.MaxAge >= 0 || clearedStateCookie.Secure {
		t.Fatalf("unexpected state cookie on HTTP redirect: %+v", clearedStateCookie)
	}
	entries := logs.All()
	if len(entries) != 1 || entries[0].ContextMap()["stage"] != "nonce" {
		t.Fatalf("callback failure stage was not logged: %+v", entries)
	}
}

type unexpectedCallbackError struct{}

func (unexpectedCallbackError) Error() string { return "unexpected callback" }

var errUnexpectedCallback error = unexpectedCallbackError{}

func findCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %q not found in %+v", name, cookies)
	return nil
}
