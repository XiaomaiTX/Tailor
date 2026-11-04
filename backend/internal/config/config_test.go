package config

import "testing"

func TestLoadAcceptsEnvironmentConfiguration(t *testing.T) {
	t.Setenv("DATABASE_PATH", "./data/test.db")
	t.Setenv("CORS_ORIGINS", "http://localhost:5173, http://localhost:8080")
	t.Setenv("API_PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIPort != 9090 || cfg.DatabasePath != "./data/test.db" || len(cfg.CORSOrigins) != 2 || cfg.LogLevel != "debug" {
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
