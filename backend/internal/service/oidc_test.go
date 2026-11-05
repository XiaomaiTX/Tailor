package service

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/XiaomaiTX/go-react-starter/internal/config"
	"github.com/XiaomaiTX/go-react-starter/internal/model"
	"github.com/XiaomaiTX/go-react-starter/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type oidcTestUserRepository []model.HeadscaleUser

func (r oidcTestUserRepository) Users(context.Context) ([]model.HeadscaleUser, error) {
	return r, nil
}

type oidcTestCase struct {
	name          string
	mutateClaims  func(map[string]any)
	mutateToken   func(string) string
	users         []model.HeadscaleUser
	wrongState    bool
	wantLoginFail bool
}

func TestOIDCLoginVerifiesClaimsAndUniquelyMapsHeadscaleEmail(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	tests := []oidcTestCase{
		{
			name:  "verified email maps case-insensitively and creates session",
			users: []model.HeadscaleUser{{ID: "user-1", Username: "alice", Email: "alice@example.test"}},
		},
		{
			name:          "state cookie mismatch",
			wrongState:    true,
			wantLoginFail: true,
			users:         []model.HeadscaleUser{{ID: "user-1", Username: "alice", Email: "alice@example.test"}},
		},
		{
			name:          "nonce mismatch",
			mutateClaims:  func(claims map[string]any) { claims["nonce"] = "wrong-nonce" },
			wantLoginFail: true,
			users:         []model.HeadscaleUser{{ID: "user-1", Username: "alice", Email: "alice@example.test"}},
		},
		{
			name:          "signature mismatch",
			mutateToken:   tamperJWTSignature,
			wantLoginFail: true,
			users:         []model.HeadscaleUser{{ID: "user-1", Username: "alice", Email: "alice@example.test"}},
		},
		{
			name:          "issuer mismatch",
			mutateClaims:  func(claims map[string]any) { claims["iss"] = "https://other.example.test" },
			wantLoginFail: true,
			users:         []model.HeadscaleUser{{ID: "user-1", Username: "alice", Email: "alice@example.test"}},
		},
		{
			name:          "audience mismatch",
			mutateClaims:  func(claims map[string]any) { claims["aud"] = []string{"other-client"} },
			wantLoginFail: true,
			users:         []model.HeadscaleUser{{ID: "user-1", Username: "alice", Email: "alice@example.test"}},
		},
		{
			name: "expired ID token",
			mutateClaims: func(claims map[string]any) {
				claims["exp"] = time.Now().Add(-time.Minute).Unix()
			},
			wantLoginFail: true,
			users:         []model.HeadscaleUser{{ID: "user-1", Username: "alice", Email: "alice@example.test"}},
		},
		{
			name:          "unverified email",
			mutateClaims:  func(claims map[string]any) { claims["email_verified"] = false },
			wantLoginFail: true,
			users:         []model.HeadscaleUser{{ID: "user-1", Username: "alice", Email: "alice@example.test"}},
		},
		{
			name:          "Headscale email has no match",
			wantLoginFail: true,
			users:         []model.HeadscaleUser{{ID: "user-1", Username: "alice", Email: "someone@example.test"}},
		},
		{
			name: "Headscale email has multiple matches",
			users: []model.HeadscaleUser{
				{ID: "user-1", Username: "alice", Email: "alice@example.test"},
				{ID: "user-2", Username: "alice2", Email: "ALICE@example.test"},
			},
			wantLoginFail: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, authRepo, expectedNonce, expectedChallenge := newOIDCTestService(t, privateKey, test)

			authorizationURL, state, err := service.BeginLogin(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			parsedAuthorizationURL, err := url.Parse(authorizationURL)
			if err != nil {
				t.Fatal(err)
			}
			if parsedAuthorizationURL.Query().Get("state") != state {
				t.Fatal("authorization request is missing its state")
			}
			*expectedNonce = parsedAuthorizationURL.Query().Get("nonce")
			*expectedChallenge = parsedAuthorizationURL.Query().Get("code_challenge")
			if parsedAuthorizationURL.Query().Get("nonce") == "" || parsedAuthorizationURL.Query().Get("code_challenge_method") != "S256" || parsedAuthorizationURL.Query().Get("code_challenge") == "" {
				t.Fatalf("authorization request lacks nonce or PKCE: %s", authorizationURL)
			}

			cookieState := state
			if test.wrongState {
				cookieState += "wrong"
			}
			rawSession, expiresAt, err := service.CompleteLogin(context.Background(), state, cookieState, "authorization-code")
			if test.wantLoginFail {
				if err == nil || rawSession != "" {
					t.Fatalf("login unexpectedly succeeded: session=%q expires=%v", rawSession, expiresAt)
				}
				return
			}
			if err != nil {
				t.Fatalf("login failed: %v", err)
			}
			if len(rawSession) < 40 || !expiresAt.After(time.Now()) {
				t.Fatalf("invalid session result: token length=%d expires=%v", len(rawSession), expiresAt)
			}
			username, valid, err := service.ResolveSession(context.Background(), rawSession)
			if err != nil || !valid || username != "alice" {
				t.Fatalf("resolved username=%q valid=%v err=%v", username, valid, err)
			}
			if _, err := authRepo.AuthSession(context.Background(), hashValue(rawSession), time.Now()); err != nil {
				t.Fatalf("session token hash was not persisted: %v", err)
			}
			if strings.Contains(rawSession, "alice") {
				t.Fatal("session token must be opaque")
			}
		})
	}
}

func newOIDCTestService(t *testing.T, signingKey *rsa.PrivateKey, test oidcTestCase) (*OIDCService, repository.AuthRepository, *string, *string) {
	t.Helper()
	issuer := "https://issuer.example.test"
	expectedChallenge := new(string)
	expectedNonce := new(string)
	transport := &oidcTestRoundTripper{
		issuer: issuer, signingKey: signingKey, test: test,
		expectedNonce: expectedNonce, expectedChallenge: expectedChallenge,
	}

	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "oidc.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.OIDCLoginState{}, &model.AuthSession{}); err != nil {
		t.Fatal(err)
	}
	authRepo := repository.NewAuthRepository(database)
	service, err := newOIDCServiceWithHTTPClient(context.Background(), config.Config{
		OIDCIssuerURL: issuer, OIDCClientID: "tailor-client", OIDCClientSecret: "test-secret",
		OIDCRedirectURL: "https://tailor.example.test/api/v1/auth/callback", OIDCSessionTTL: 12 * time.Hour,
	}, authRepo, oidcTestUserRepository(test.users), &http.Client{Transport: transport, Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("create OIDC service: %v", err)
	}
	return service, authRepo, expectedNonce, expectedChallenge
}

type oidcTestRoundTripper struct {
	issuer            string
	signingKey        *rsa.PrivateKey
	test              oidcTestCase
	expectedNonce     *string
	expectedChallenge *string
}

func (r *oidcTestRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	switch request.URL.Path {
	case "/.well-known/openid-configuration":
		return oidcTestJSONResponse(request, http.StatusOK, map[string]any{
			"issuer": r.issuer, "authorization_endpoint": r.issuer + "/authorize",
			"token_endpoint": r.issuer + "/token", "jwks_uri": r.issuer + "/keys",
		})
	case "/keys":
		modulus := base64.RawURLEncoding.EncodeToString(r.signingKey.N.Bytes())
		exponent := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(r.signingKey.E)).Bytes())
		return oidcTestJSONResponse(request, http.StatusOK, map[string]any{"keys": []any{map[string]string{
			"kty": "RSA", "use": "sig", "kid": "test-key", "alg": "RS256", "n": modulus, "e": exponent,
		}}})
	case "/token":
		body, err := io.ReadAll(request.Body)
		if err != nil {
			return nil, err
		}
		form, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, err
		}
		verifier := form.Get("code_verifier")
		challengeDigest := sha256.Sum256([]byte(verifier))
		challenge := base64.RawURLEncoding.EncodeToString(challengeDigest[:])
		if form.Get("code") != "authorization-code" || verifier == "" || challenge != *r.expectedChallenge {
			return oidcTestJSONResponse(request, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
		}
		claims := map[string]any{
			"iss": r.issuer, "sub": "oidc-subject-1", "aud": "tailor-client",
			"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
			"nonce": *r.expectedNonce, "email": " ALICE@EXAMPLE.TEST ", "email_verified": true,
		}
		if r.test.mutateClaims != nil {
			r.test.mutateClaims(claims)
		}
		idToken, err := signTestIDToken(r.signingKey, claims)
		if err != nil {
			return nil, err
		}
		if r.test.mutateToken != nil {
			idToken = r.test.mutateToken(idToken)
		}
		return oidcTestJSONResponse(request, http.StatusOK, map[string]any{
			"access_token": "provider-access-token", "token_type": "Bearer",
			"expires_in": 3600, "id_token": idToken,
		})
	default:
		return oidcTestJSONResponse(request, http.StatusNotFound, map[string]string{"error": "not_found"})
	}
}

func oidcTestJSONResponse(request *http.Request, status int, value any) (*http.Response, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: status, Status: http.StatusText(status), Header: http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(bytes.NewReader(body)), Request: request,
	}, nil
}

func signTestIDToken(key *rsa.PrivateKey, claims map[string]any) (string, error) {
	header, err := json.Marshal(map[string]string{"alg": "RS256", "kid": "test-key", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	message := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return message + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func tamperJWTSignature(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[2] == "" {
		return token + "x"
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signature) == 0 {
		return token + "x"
	}
	signature[0] ^= 0xff
	parts[2] = base64.RawURLEncoding.EncodeToString(signature)
	return strings.Join(parts, ".")
}
