package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/XiaomaiTX/go-react-starter/internal/config"
	"github.com/XiaomaiTX/go-react-starter/internal/model"
	"github.com/XiaomaiTX/go-react-starter/internal/repository"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const oidcStateTTL = 10 * time.Minute

var ErrInvalidOIDCLogin = errors.New("invalid OIDC login")

type OIDCLoginFailure struct{ Stage string }

func (e *OIDCLoginFailure) Error() string { return ErrInvalidOIDCLogin.Error() }
func (e *OIDCLoginFailure) Unwrap() error { return ErrInvalidOIDCLogin }

func OIDCFailureStage(err error) string {
	var failure *OIDCLoginFailure
	if errors.As(err, &failure) && failure.Stage != "" {
		return failure.Stage
	}
	return "internal"
}

func invalidOIDCLogin(stage string) error { return &OIDCLoginFailure{Stage: stage} }

type HeadscaleUserRepository interface {
	Users(context.Context) ([]model.HeadscaleUser, error)
}

type OIDCService struct {
	provider       *oidc.Provider
	oauth          oauth2.Config
	verifier       *oidc.IDTokenVerifier
	httpClient     *http.Client
	authRepository repository.AuthRepository
	userRepository HeadscaleUserRepository
	sessionTTL     time.Duration
	cookieSecure   bool
}

func NewOIDCService(ctx context.Context, cfg config.Config, authRepository repository.AuthRepository, userRepository HeadscaleUserRepository) (*OIDCService, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	return newOIDCServiceWithHTTPClient(ctx, cfg, authRepository, userRepository, client)
}

func newOIDCServiceWithHTTPClient(ctx context.Context, cfg config.Config, authRepository repository.AuthRepository, userRepository HeadscaleUserRepository, client *http.Client) (*OIDCService, error) {
	discoveryContext, cancel := context.WithTimeout(oidc.ClientContext(ctx, client), 10*time.Second)
	defer cancel()
	provider, err := oidc.NewProvider(discoveryContext, cfg.OIDCIssuerURL)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery failed: %w", err)
	}
	endpoint := provider.Endpoint()
	if endpoint.AuthURL == "" || endpoint.TokenURL == "" {
		return nil, errors.New("OIDC provider must advertise authorization and token endpoints")
	}
	redirect, err := url.Parse(cfg.OIDCRedirectURL)
	if err != nil {
		return nil, errors.New("OIDC redirect URL is invalid")
	}
	sessionTTL := cfg.OIDCSessionTTL
	if sessionTTL <= 0 {
		sessionTTL = 12 * time.Hour
	}
	return &OIDCService{
		provider: provider,
		oauth: oauth2.Config{
			ClientID: cfg.OIDCClientID, ClientSecret: cfg.OIDCClientSecret,
			Endpoint: endpoint, RedirectURL: cfg.OIDCRedirectURL,
			Scopes: []string{oidc.ScopeOpenID, "email", "profile"},
		},
		verifier:       provider.Verifier(&oidc.Config{ClientID: cfg.OIDCClientID}),
		httpClient:     client,
		authRepository: authRepository,
		userRepository: userRepository,
		sessionTTL:     sessionTTL,
		cookieSecure:   redirect.Scheme == "https",
	}, nil
}

func (s *OIDCService) BeginLogin(ctx context.Context) (authorizationURL, state string, err error) {
	if err := s.authRepository.PurgeExpiredAuthRecords(ctx, time.Now()); err != nil {
		return "", "", err
	}
	state, err = randomValue(32)
	if err != nil {
		return "", "", err
	}
	nonce, err := randomValue(32)
	if err != nil {
		return "", "", err
	}
	verifier := oauth2.GenerateVerifier()
	now := time.Now()
	if err := s.authRepository.SaveLoginState(ctx, model.OIDCLoginState{
		StateHash: hashValue(state), Nonce: nonce, CodeVerifier: verifier,
		CreatedAt: now, ExpiresAt: now.Add(oidcStateTTL),
	}); err != nil {
		return "", "", err
	}
	return s.oauth.AuthCodeURL(state,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.S256ChallengeOption(verifier),
	), state, nil
}

func (s *OIDCService) CancelLogin(ctx context.Context, state, cookieState string) error {
	if !validState(state, cookieState) {
		return invalidOIDCLogin("state")
	}
	_, err := s.authRepository.ConsumeLoginState(ctx, hashValue(state), time.Now())
	if err != nil {
		return invalidOIDCLogin("state_consume")
	}
	return nil
}

func (s *OIDCService) CompleteLogin(ctx context.Context, state, cookieState, code string) (string, time.Time, error) {
	if !validState(state, cookieState) || strings.TrimSpace(code) == "" {
		return "", time.Time{}, invalidOIDCLogin("state_or_code")
	}
	loginState, err := s.authRepository.ConsumeLoginState(ctx, hashValue(state), time.Now())
	if err != nil {
		return "", time.Time{}, invalidOIDCLogin("state_consume")
	}
	ctx = oidc.ClientContext(ctx, s.httpClient)
	token, err := s.oauth.Exchange(ctx, code, oauth2.VerifierOption(loginState.CodeVerifier))
	if err != nil {
		return "", time.Time{}, invalidOIDCLogin("token_exchange")
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return "", time.Time{}, invalidOIDCLogin("id_token_missing")
	}
	idToken, err := s.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return "", time.Time{}, invalidOIDCLogin("id_token_validation")
	}
	if idToken.Subject == "" {
		return "", time.Time{}, invalidOIDCLogin("subject")
	}
	if subtle.ConstantTimeCompare([]byte(idToken.Nonce), []byte(loginState.Nonce)) != 1 {
		return "", time.Time{}, invalidOIDCLogin("nonce")
	}
	email, err := s.verifiedEmail(ctx, token, idToken)
	if err != nil {
		return "", time.Time{}, err
	}
	users, err := s.userRepository.Users(ctx)
	if err != nil {
		return "", time.Time{}, invalidOIDCLogin("headscale_lookup")
	}
	matched := model.HeadscaleUser{}
	for _, user := range users {
		if normalizeEmail(user.Email) != email {
			continue
		}
		if matched.ID != "" {
			return "", time.Time{}, invalidOIDCLogin("headscale_email_ambiguous")
		}
		matched = user
	}
	if matched.ID == "" {
		return "", time.Time{}, invalidOIDCLogin("headscale_email_no_match")
	}

	rawSession, err := randomValue(32)
	if err != nil {
		return "", time.Time{}, invalidOIDCLogin("session_token")
	}
	createdAt := time.Now()
	expiresAt := createdAt.Add(s.sessionTTL)
	if err := s.authRepository.CreateAuthSession(ctx, model.AuthSession{
		TokenHash: hashValue(rawSession), HeadscaleUserID: matched.ID,
		CreatedAt: createdAt, ExpiresAt: expiresAt,
	}); err != nil {
		return "", time.Time{}, invalidOIDCLogin("session_storage")
	}
	return rawSession, expiresAt, nil
}

func (s *OIDCService) verifiedEmail(ctx context.Context, token *oauth2.Token, idToken *oidc.IDToken) (string, error) {
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return "", invalidOIDCLogin("email_claim")
	}
	if claims.EmailVerified && normalizeEmail(claims.Email) != "" {
		return normalizeEmail(claims.Email), nil
	}
	userInfo, err := s.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil || userInfo.Subject != idToken.Subject || !userInfo.EmailVerified {
		return "", invalidOIDCLogin("verified_email")
	}
	email := normalizeEmail(userInfo.Email)
	if email == "" {
		return "", invalidOIDCLogin("verified_email")
	}
	return email, nil
}

func (s *OIDCService) ResolveSession(ctx context.Context, rawSession string) (string, bool, error) {
	if rawSession == "" {
		return "", false, nil
	}
	session, err := s.authRepository.AuthSession(ctx, hashValue(rawSession), time.Now())
	if errors.Is(err, repository.ErrAuthRecordNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	users, err := s.userRepository.Users(ctx)
	if err != nil {
		return "", false, err
	}
	for _, user := range users {
		if user.ID == session.HeadscaleUserID && strings.TrimSpace(user.Username) != "" {
			return user.Username, true, nil
		}
	}
	return "", false, nil
}

func (s *OIDCService) RevokeSession(ctx context.Context, rawSession string) error {
	if rawSession == "" {
		return nil
	}
	return s.authRepository.DeleteAuthSession(ctx, hashValue(rawSession))
}

func (s *OIDCService) SecureCookies() bool { return s.cookieSecure }

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validState(state, cookieState string) bool {
	return state != "" && len(state) == len(cookieState) && subtle.ConstantTimeCompare([]byte(state), []byte(cookieState)) == 1
}

func hashValue(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func randomValue(bytes int) (string, error) {
	value := make([]byte, bytes)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
