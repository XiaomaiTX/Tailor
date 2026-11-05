package headscale

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var ErrNotConfigured = errors.New("Headscale is not configured")

type User struct {
	ID       string
	Username string
	Email    string
}

type Node struct {
	ID          string
	UserID      string
	Username    string
	Hostname    string
	GivenName   string
	IPAddresses []string
	Tags        []string
	Online      *bool
	LastSeen    string
	Expiry      string
}

type Client interface {
	Configured() bool
	ListUsers(context.Context) ([]User, error)
	ListNodes(context.Context) ([]Node, error)
	GetPolicy(context.Context) (string, error)
	SetPolicy(context.Context, string) error
}

type HTTPClient struct {
	base   *url.URL
	apiKey string
	client *http.Client
}

func NewClient(rawURL, apiKey string) (*HTTPClient, error) {
	rawURL = strings.TrimSpace(rawURL)
	apiKey = strings.TrimSpace(apiKey)
	if rawURL == "" && apiKey == "" {
		return &HTTPClient{}, nil
	}
	if rawURL == "" || apiKey == "" {
		return nil, errors.New("HEADSCALE_URL 和 HEADSCALE_API_KEY 必须同时配置")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("HEADSCALE_URL 必须是有效的 HTTP(S) URL")
	}
	if parsed.Scheme == "http" {
		host := parsed.Hostname()
		ip := net.ParseIP(host)
		if !strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback()) {
			return nil, errors.New("HEADSCALE_URL 仅允许对本地回环地址使用 HTTP，其他地址必须使用 HTTPS")
		}
	}
	return &HTTPClient{
		base:   parsed,
		apiKey: apiKey,
		client: &http.Client{
			Timeout:       15 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

func (c *HTTPClient) Configured() bool { return c != nil && c.base != nil && c.apiKey != "" }

func (c *HTTPClient) ListUsers(ctx context.Context) ([]User, error) {
	var body json.RawMessage
	if err := c.get(ctx, "/api/v1/user", &body); err != nil {
		return nil, err
	}
	items, err := collection(body, "users", "user")
	if err != nil {
		return nil, fmt.Errorf("decode Headscale users: %w", err)
	}
	users := make([]User, 0, len(items))
	for _, item := range items {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(item, &fields); err != nil {
			return nil, err
		}
		user := User{ID: field(fields, "id", "userId"), Username: field(fields, "name", "username"), Email: field(fields, "email")}
		if user.Username == "" {
			user.Username = user.Email
		}
		if user.ID == "" || user.Username == "" {
			return nil, errors.New("Headscale returned a user without a stable id or name")
		}
		users = append(users, user)
	}
	return users, nil
}

func (c *HTTPClient) ListNodes(ctx context.Context) ([]Node, error) {
	var body json.RawMessage
	if err := c.get(ctx, "/api/v1/node", &body); err != nil {
		return nil, err
	}
	items, err := collection(body, "nodes", "node")
	if err != nil {
		return nil, fmt.Errorf("decode Headscale nodes: %w", err)
	}
	nodes := make([]Node, 0, len(items))
	for _, item := range items {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(item, &fields); err != nil {
			return nil, err
		}
		node := Node{
			ID:        field(fields, "id", "nodeId"),
			Hostname:  field(fields, "name", "hostname"),
			GivenName: field(fields, "givenName", "given_name"),
			LastSeen:  field(fields, "lastSeen", "last_seen"),
			Expiry:    field(fields, "expiry", "expiresAt", "expires_at"),
		}
		if node.ID == "" || node.Hostname == "" {
			return nil, errors.New("Headscale returned a node without a stable id or hostname")
		}
		var ips []string
		_ = json.Unmarshal(firstField(fields, "ipAddresses", "ip_addresses", "ips"), &ips)
		node.IPAddresses = ips
		var tags []string
		_ = json.Unmarshal(firstField(fields, "tags", "forcedTags", "forced_tags"), &tags)
		node.Tags = tags
		if raw := firstField(fields, "online"); len(raw) > 0 && string(raw) != "null" {
			var online bool
			if json.Unmarshal(raw, &online) == nil {
				node.Online = &online
			}
		}
		if raw := firstField(fields, "user"); len(raw) > 0 {
			var userFields map[string]json.RawMessage
			if json.Unmarshal(raw, &userFields) == nil {
				node.UserID = field(userFields, "id", "userId")
				node.Username = field(userFields, "name", "username")
			} else {
				node.UserID = stringValue(raw)
			}
		}
		if node.UserID == "" {
			node.UserID = field(fields, "userId", "user_id")
		}
		if node.Username == "" {
			node.Username = field(fields, "username")
		}
		if node.UserID == "" || node.Username == "" {
			return nil, errors.New("Headscale returned a node without an owner")
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (c *HTTPClient) GetPolicy(ctx context.Context) (string, error) {
	var response map[string]json.RawMessage
	if err := c.get(ctx, "/api/v1/policy", &response); err != nil {
		return "", err
	}
	policy := field(response, "policy", "acl")
	if policy == "" {
		return "", errors.New("Headscale returned an empty policy")
	}
	return policy, nil
}

func (c *HTTPClient) SetPolicy(ctx context.Context, policy string) error {
	if !c.Configured() {
		return ErrNotConfigured
	}
	body, err := json.Marshal(map[string]string{"policy": policy})
	if err != nil {
		return err
	}
	return c.request(ctx, http.MethodPut, "/api/v1/policy", bytes.NewReader(body), nil)
}

func (c *HTTPClient) get(ctx context.Context, path string, target any) error {
	if !c.Configured() {
		return ErrNotConfigured
	}
	return c.request(ctx, http.MethodGet, path, nil, target)
}

func (c *HTTPClient) request(ctx context.Context, method, path string, body io.Reader, target any) error {
	if !c.Configured() {
		return ErrNotConfigured
	}
	endpoint := *c.base
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + path
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("Headscale request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Headscale returned HTTP %d", response.StatusCode)
	}
	if target == nil {
		_, err := io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		return err
	}
	limited := io.LimitReader(response.Body, 10<<20)
	if err := json.NewDecoder(limited).Decode(target); err != nil {
		return fmt.Errorf("decode Headscale response: %w", err)
	}
	return nil
}

func collection(raw json.RawMessage, names ...string) ([]json.RawMessage, error) {
	var direct []json.RawMessage
	if err := json.Unmarshal(raw, &direct); err == nil {
		return direct, nil
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	for _, name := range names {
		if value, ok := envelope[name]; ok {
			if err := json.Unmarshal(value, &direct); err != nil {
				return nil, err
			}
			return direct, nil
		}
	}
	return nil, errors.New("Headscale collection response has an unexpected shape")
}

func firstField(fields map[string]json.RawMessage, names ...string) json.RawMessage {
	for _, name := range names {
		if value, ok := fields[name]; ok {
			return value
		}
	}
	return nil
}

func field(fields map[string]json.RawMessage, names ...string) string {
	return stringValue(firstField(fields, names...))
}

func stringValue(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var value string
	if json.Unmarshal(raw, &value) == nil {
		return strings.TrimSpace(value)
	}
	var number json.Number
	if json.Unmarshal(raw, &number) == nil {
		return number.String()
	}
	return ""
}

func ParseTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" || value == "0001-01-01T00:00:00Z" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return &parsed
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		parsed := time.Unix(seconds, 0).UTC()
		return &parsed
	}
	return nil
}
