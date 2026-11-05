package headscale

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestHTTPClientUsesBearerAPIAndDecodesHeadscaleCollections(t *testing.T) {
	policyText := `{"groups":{},"grants":[]}`
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Authorization") != "Bearer test-api-key" {
			return nil, errors.New("missing bearer")
		}
		status, body := http.StatusOK, ""
		switch request.URL.Path {
		case "/api/v1/user":
			body = `{"users":[{"id":"u1","name":"alice","email":"alice@example.com"}]}`
		case "/api/v1/node":
			body = `{"nodes":[{"id":7,"name":"laptop","user":{"id":"u1","name":"alice"},"ipAddresses":["100.64.0.10","fd7a:115c:a1e0::10"],"forcedTags":["tag:work"],"online":true}]}`
		case "/api/v1/policy":
			if request.Method == http.MethodGet {
				encoded, _ := json.Marshal(map[string]string{"policy": policyText})
				body = string(encoded)
				break
			}
			var requestBody map[string]string
			if err := json.NewDecoder(request.Body).Decode(&requestBody); err != nil || requestBody["policy"] == "" {
				status = http.StatusBadRequest
				break
			}
			policyText = requestBody["policy"]
			status = http.StatusNoContent
		default:
			status = http.StatusNotFound
		}
		return &http.Response{
			StatusCode: status,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    request,
		}, nil
	})
	base, _ := url.Parse("https://headscale.example.com")
	client := &HTTPClient{base: base, apiKey: "test-api-key", client: &http.Client{Transport: transport}}

	users, err := client.ListUsers(context.Background())
	if err != nil || len(users) != 1 || users[0].Username != "alice" || users[0].Email != "alice@example.com" {
		t.Fatalf("users=%+v err=%v", users, err)
	}
	nodes, err := client.ListNodes(context.Background())
	if err != nil || len(nodes) != 1 || nodes[0].ID != "7" || nodes[0].UserID != "u1" || len(nodes[0].IPAddresses) != 2 || len(nodes[0].Tags) != 1 {
		t.Fatalf("nodes=%+v err=%v", nodes, err)
	}
	current, err := client.GetPolicy(context.Background())
	if err != nil || current != `{"groups":{},"grants":[]}` {
		t.Fatalf("policy=%q err=%v", current, err)
	}
	updated := `{"groups":{},"grants":[{"src":["100.64.0.10"]}]}`
	if err := client.SetPolicy(context.Background(), updated); err != nil {
		t.Fatal(err)
	}
	current, err = client.GetPolicy(context.Background())
	if err != nil || current != updated {
		t.Fatalf("policy write body was not accepted: policy=%q err=%v", current, err)
	}
}

func TestNewClientRequiresURLAndKeyTogether(t *testing.T) {
	if _, err := NewClient("http://localhost:8080", ""); err == nil || !strings.Contains(err.Error(), "必须同时配置") {
		t.Fatalf("expected paired-configuration error, got %v", err)
	}
	client, err := NewClient("", "")
	if err != nil || client.Configured() {
		t.Fatalf("empty configuration should be disabled: client=%+v err=%v", client, err)
	}
}
