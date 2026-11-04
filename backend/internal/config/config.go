package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	APIPort     int
	LogLevel    string
	DatabaseURL string
	RedisURL    string
	CORSOrigins []string
}

func Load() (Config, error) {
	port, err := optionalPort("API_PORT", 8080)
	if err != nil {
		return Config{}, err
	}
	databaseURL, err := requiredURL("DATABASE_URL", "postgres", "postgresql")
	if err != nil {
		return Config{}, err
	}
	redisURL, err := requiredURL("REDIS_URL", "redis", "rediss")
	if err != nil {
		return Config{}, err
	}
	origins := splitCSV(os.Getenv("CORS_ORIGINS"))
	if len(origins) == 0 {
		return Config{}, fmt.Errorf("CORS_ORIGINS 不能为空")
	}
	level := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	if level == "" {
		level = "info"
	}
	if level != "debug" && level != "info" && level != "warn" && level != "error" {
		return Config{}, fmt.Errorf("LOG_LEVEL 必须为 debug、info、warn 或 error")
	}
	return Config{APIPort: port, LogLevel: level, DatabaseURL: databaseURL, RedisURL: redisURL, CORSOrigins: origins}, nil
}

func optionalPort(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	port, err := strconv.Atoi(raw)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("%s 必须是 1 到 65535 之间的端口", key)
	}
	return port, nil
}

func requiredURL(key string, allowedSchemes ...string) (string, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return "", fmt.Errorf("%s 不能为空", key)
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("%s 必须是有效 URL", key)
	}
	for _, scheme := range allowedSchemes {
		if parsed.Scheme == scheme {
			return raw, nil
		}
	}
	return "", fmt.Errorf("%s 使用了不支持的协议", key)
}

func splitCSV(value string) []string {
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}
