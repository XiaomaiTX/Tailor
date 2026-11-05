package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	APIPort             int
	LogLevel            string
	DatabasePath        string
	CORSOrigins         []string
	HeadscaleURL        string
	HeadscaleKey        string
	OIDCIssuerURL       string
	OIDCClientID        string
	OIDCClientSecret    string
	OIDCRedirectURL     string
	OIDCSessionTTL      time.Duration
	PolicyWrites        bool
	SelfServiceEnabled  bool
	BootstrapAdmins     []string
	AdminGroups         []string
	ReviewerGroups      []string
	ApplicantGroups     []string
	ApplicantMode       string
	SyncIntervalSeconds int
}

func Load() (Config, error) {
	port, err := optionalPort("API_PORT", 8080)
	if err != nil {
		return Config{}, err
	}
	databasePath := strings.TrimSpace(os.Getenv("DATABASE_PATH"))
	if databasePath == "" {
		return Config{}, fmt.Errorf("DATABASE_PATH 不能为空")
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
	oidcIssuerURL := strings.TrimSpace(os.Getenv("OIDC_ISSUER_URL"))
	oidcClientID := strings.TrimSpace(os.Getenv("OIDC_CLIENT_ID"))
	oidcClientSecret := strings.TrimSpace(os.Getenv("OIDC_CLIENT_SECRET"))
	oidcRedirectURL := strings.TrimSpace(os.Getenv("OIDC_REDIRECT_URL"))
	if oidcIssuerURL == "" || oidcClientID == "" || oidcClientSecret == "" || oidcRedirectURL == "" {
		return Config{}, fmt.Errorf("OIDC_ISSUER_URL、OIDC_CLIENT_ID、OIDC_CLIENT_SECRET 和 OIDC_REDIRECT_URL 均为必填项")
	}
	if err := validateOIDCURL("OIDC_ISSUER_URL", oidcIssuerURL, true); err != nil {
		return Config{}, err
	}
	if err := validateOIDCURL("OIDC_REDIRECT_URL", oidcRedirectURL, false); err != nil {
		return Config{}, err
	}
	redirect, _ := url.Parse(oidcRedirectURL)
	if redirect.Path != "/api/v1/auth/callback" || redirect.RawQuery != "" {
		return Config{}, fmt.Errorf("OIDC_REDIRECT_URL 必须是 /api/v1/auth/callback 的精确回调 URL")
	}
	policyWrites, err := optionalBool("PEER_RELAY_POLICY_WRITES", false)
	if err != nil {
		return Config{}, err
	}
	selfServiceEnabled, err := optionalBool("PEER_RELAY_SELF_SERVICE_ENABLED", true)
	if err != nil {
		return Config{}, err
	}
	applicantMode := strings.ToLower(strings.TrimSpace(os.Getenv("PEER_RELAY_APPLICANT_MODE")))
	if applicantMode == "" {
		applicantMode = "all"
	}
	if applicantMode != "all" && applicantMode != "groups" && applicantMode != "disabled" {
		return Config{}, fmt.Errorf("PEER_RELAY_APPLICANT_MODE 必须为 all、groups 或 disabled")
	}
	interval, err := optionalInt("HEADSCALE_SYNC_INTERVAL_SECONDS", 60, 10, 3600)
	if err != nil {
		return Config{}, err
	}
	return Config{
		APIPort: port, LogLevel: level, DatabasePath: databasePath, CORSOrigins: origins,
		HeadscaleURL:        strings.TrimRight(strings.TrimSpace(os.Getenv("HEADSCALE_URL")), "/"),
		HeadscaleKey:        strings.TrimSpace(os.Getenv("HEADSCALE_API_KEY")),
		OIDCIssuerURL:       oidcIssuerURL,
		OIDCClientID:        oidcClientID,
		OIDCClientSecret:    oidcClientSecret,
		OIDCRedirectURL:     oidcRedirectURL,
		OIDCSessionTTL:      12 * time.Hour,
		PolicyWrites:        policyWrites,
		SelfServiceEnabled:  selfServiceEnabled,
		BootstrapAdmins:     splitCSV(os.Getenv("PEER_RELAY_BOOTSTRAP_ADMINS")),
		AdminGroups:         configuredGroups("PEER_RELAY_ADMIN_GROUPS", "group:relay-admin"),
		ReviewerGroups:      configuredGroups("PEER_RELAY_REVIEWER_GROUPS", "group:relay-reviewer"),
		ApplicantGroups:     configuredGroups("PEER_RELAY_APPLICANT_GROUPS", "group:relay-applicant"),
		ApplicantMode:       applicantMode,
		SyncIntervalSeconds: interval,
	}, nil
}

func validateOIDCURL(key, raw string, issuer bool) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("%s 必须是有效的绝对 HTTP(S) URL", key)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("%s 必须使用 HTTPS（本地回环地址可使用 HTTP）", key)
	}
	if parsed.Scheme == "http" && parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "::1" {
		return fmt.Errorf("%s 仅允许 HTTPS（本地回环地址可使用 HTTP）", key)
	}
	if issuer && parsed.RawQuery != "" {
		return fmt.Errorf("%s 不能包含查询参数", key)
	}
	return nil
}

func optionalBool(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s 必须是布尔值", key)
	}
	return value, nil
}

func optionalInt(key string, fallback, minimum, maximum int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return 0, fmt.Errorf("%s 必须在 %d 到 %d 之间", key, minimum, maximum)
	}
	return value, nil
}

func configuredGroups(key, fallback string) []string {
	value := splitCSV(os.Getenv(key))
	if len(value) == 0 {
		return []string{fallback}
	}
	return value
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
