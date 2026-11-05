package config

import "testing"

func TestLoadAcceptsEnvironmentConfiguration(t *testing.T) {
	t.Setenv("DATABASE_PATH", "./data/test.db")
	t.Setenv("CORS_ORIGINS", "http://localhost:5173, http://localhost:8080")
	t.Setenv("API_PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("PEER_RELAY_SELF_SERVICE_ENABLED", "")
	t.Setenv("OIDC_ISSUER_URL", "https://login.example.test")
	t.Setenv("OIDC_CLIENT_ID", "tailor-client")
	t.Setenv("OIDC_CLIENT_SECRET", "tailor-secret")
	t.Setenv("OIDC_REDIRECT_URL", "https://tailor.example.test/api/v1/auth/callback")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIPort != 9090 || cfg.DatabasePath != "./data/test.db" || len(cfg.CORSOrigins) != 2 || cfg.LogLevel != "debug" || !cfg.SelfServiceEnabled || cfg.OIDCSessionTTL.Hours() != 12 {
		t.Fatalf("意外配置：%+v", cfg)
	}
}

func TestLoadRejectsMissingDatabasePath(t *testing.T) {
	t.Setenv("DATABASE_PATH", "")
	t.Setenv("CORS_ORIGINS", "http://localhost:5173")
	if _, err := Load(); err == nil {
		t.Fatal("缺少数据库路径时应失败")
	}
}

func TestLoadRequiresOIDCConfiguration(t *testing.T) {
	t.Setenv("DATABASE_PATH", "./data/test.db")
	t.Setenv("CORS_ORIGINS", "http://localhost:5173")
	for _, key := range []string{"OIDC_ISSUER_URL", "OIDC_CLIENT_ID", "OIDC_CLIENT_SECRET", "OIDC_REDIRECT_URL"} {
		t.Setenv(key, "")
	}
	if _, err := Load(); err == nil {
		t.Fatal("OIDC 配置缺失时应失败")
	}
}

func TestLoadRejectsInsecureNonLocalOIDCURL(t *testing.T) {
	t.Setenv("DATABASE_PATH", "./data/test.db")
	t.Setenv("CORS_ORIGINS", "http://localhost:5173")
	t.Setenv("OIDC_ISSUER_URL", "http://login.example.test")
	t.Setenv("OIDC_CLIENT_ID", "tailor-client")
	t.Setenv("OIDC_CLIENT_SECRET", "tailor-secret")
	t.Setenv("OIDC_REDIRECT_URL", "https://tailor.example.test/api/v1/auth/callback")
	if _, err := Load(); err == nil {
		t.Fatal("非本地 OIDC URL 使用 HTTP 时应失败")
	}
}

func TestLoadRejectsOIDCRedirectURLOutsideCallbackRoute(t *testing.T) {
	t.Setenv("DATABASE_PATH", "./data/test.db")
	t.Setenv("CORS_ORIGINS", "http://localhost:5173")
	t.Setenv("OIDC_ISSUER_URL", "https://login.example.test")
	t.Setenv("OIDC_CLIENT_ID", "tailor-client")
	t.Setenv("OIDC_CLIENT_SECRET", "tailor-secret")
	t.Setenv("OIDC_REDIRECT_URL", "https://tailor.example.test/other/callback")
	if _, err := Load(); err == nil {
		t.Fatal("non-callback OIDC redirect path should be rejected")
	}
}
