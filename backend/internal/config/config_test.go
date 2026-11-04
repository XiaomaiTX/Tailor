package config

import "testing"

func TestLoadAcceptsEnvironmentConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/starter?sslmode=disable")
	t.Setenv("REDIS_URL", "redis://:pass@localhost:6379/0")
	t.Setenv("CORS_ORIGINS", "http://localhost:5173, http://localhost:8080")
	t.Setenv("API_PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIPort != 9090 || len(cfg.CORSOrigins) != 2 || cfg.LogLevel != "debug" {
		t.Fatalf("意外配置：%+v", cfg)
	}
}

func TestLoadRejectsMissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("CORS_ORIGINS", "http://localhost:5173")
	if _, err := Load(); err == nil {
		t.Fatal("缺少数据库 URL 时应失败")
	}
}
