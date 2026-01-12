// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OIDC support.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
)

// OIDC Flow errors
var (
	// ErrInvalidState indicates the state parameter is invalid or expired.
	ErrInvalidState = errors.New("invalid or expired state parameter")

	// ErrStateNotFound indicates the state was not found in the store.
	ErrStateNotFound = errors.New("state not found")

	// ErrStateExpired indicates the state has expired.
	ErrStateExpired = errors.New("state expired")

	// ErrTokenExchangeFailed indicates token exchange failed.
	ErrTokenExchangeFailed = errors.New("token exchange failed")
)

// OIDCFlowConfig holds configuration for the OIDC authorization flow.
type OIDCFlowConfig struct {
	// IssuerURL is the OIDC provider's issuer URL.
	IssuerURL string

	// ClientID is the OAuth2 client identifier.
	ClientID string

	// ClientSecret is the OAuth2 client secret (optional for public clients).
	ClientSecret string

	// RedirectURL is the callback URL for authorization code flow.
	RedirectURL string

	// Scopes to request from the OIDC provider.
	Scopes []string

	// PKCEEnabled enables Proof Key for Code Exchange (RFC 7636).
	PKCEEnabled bool

	// StateTTL is how long state parameters are valid.
	StateTTL time.Duration

	// SessionDuration is how long sessions are valid after authentication.
	SessionDuration time.Duration

	// PostLoginRedirectURL is the default redirect after login.
	PostLoginRedirectURL string

	// HTTPClient for making requests (optional, uses default if nil).
	HTTPClient *http.Client

	// JWKSCacheTTL is how long to cache JWKS keys.
	JWKSCacheTTL time.Duration

	// ClockSkew is the allowed clock difference for token validation.
	ClockSkew time.Duration

	// RolesClaim is the claim name for roles (default: "roles").
	RolesClaim string

	// GroupsClaim is the claim name for groups (default: "groups").
	GroupsClaim string

	// UsernameClaims is the list of claims to try for username.
	UsernameClaims []string

	// DefaultRoles are roles assigned if no roles in token.
	DefaultRoles []string

	// NonceEnabled enables nonce generation for ID token validation.
	NonceEnabled bool
}

// OIDCStateData holds state information during the authorization flow.
type OIDCStateData struct {
	// CodeVerifier is the PKCE code verifier (if PKCE enabled).
	CodeVerifier string

	// PostLoginRedirect is where to redirect after successful login.
	PostLoginRedirect string

	// Nonce is an optional nonce for ID token validation.
	Nonce string

	// CreatedAt is when the state was created.
	CreatedAt time.Time

	// ExpiresAt is when the state expires.
	ExpiresAt time.Time
}

// IsExpired checks if the state has expired.
func (s *OIDCStateData) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// OIDCStateStore defines the interface for storing OIDC state data.
type OIDCStateStore interface {
	// Store saves state data with the given key.
	Store(ctx context.Context, key string, state *OIDCStateData) error

	// Get retrieves state data by key.
	Get(ctx context.Context, key string) (*OIDCStateData, error)

	// Delete removes state data by key.
	Delete(ctx context.Context, key string) error

	// CleanupExpired removes all expired states.
	CleanupExpired(ctx context.Context) (int, error)
}

// MemoryOIDCStateStore is an in-memory implementation of OIDCStateStore.
type MemoryOIDCStateStore struct {
	mu     sync.RWMutex
	states map[string]*OIDCStateData
}

// NewMemoryOIDCStateStore creates a new in-memory state store.
func NewMemoryOIDCStateStore() *MemoryOIDCStateStore {
	return &MemoryOIDCStateStore{
		states: make(map[string]*OIDCStateData),
	}
}

// Store saves state data with the given key.
func (s *MemoryOIDCStateStore) Store(ctx context.Context, key string, state *OIDCStateData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Deep copy
	stored := &OIDCStateData{
		CodeVerifier:      state.CodeVerifier,
		PostLoginRedirect: state.PostLoginRedirect,
		Nonce:             state.Nonce,
		CreatedAt:         state.CreatedAt,
		ExpiresAt:         state.ExpiresAt,
	}
	s.states[key] = stored
	return nil
}

// Get retrieves state data by key.
func (s *MemoryOIDCStateStore) Get(ctx context.Context, key string) (*OIDCStateData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.states[key]
	if !ok {
		return nil, ErrStateNotFound
	}

	if state.IsExpired() {
		return nil, ErrStateExpired
	}

	// Return a copy
	return &OIDCStateData{
		CodeVerifier:      state.CodeVerifier,
		PostLoginRedirect: state.PostLoginRedirect,
		Nonce:             state.Nonce,
		CreatedAt:         state.CreatedAt,
		ExpiresAt:         state.ExpiresAt,
	}, nil
}

// Delete removes state data by key.
func (s *MemoryOIDCStateStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, key)
	return nil
}

// CleanupExpired removes all expired states.
func (s *MemoryOIDCStateStore) CleanupExpired(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for key, state := range s.states {
		if state.IsExpired() {
			delete(s.states, key)
			count++
		}
	}
	return count, nil
}

// OIDCFlow manages the OIDC authorization code flow with PKCE support.
type OIDCFlow struct {
	config *OIDCFlowConfig
	store  OIDCStateStore
	client *http.Client

	// Discovered endpoints (can be set manually for testing)
	authorizationEndpoint string
	tokenEndpoint         string
	userinfoEndpoint      string
	jwksURI               string
	endSessionEndpoint    string // OIDC RP-Initiated Logout (Phase 4B)

	// JWKS cache for ID token validation
	jwksCache *JWKSCache

	// ID token validator
	idTokenValidator *IDTokenValidator
}

// NewOIDCFlow creates a new OIDC flow manager.
func NewOIDCFlow(config *OIDCFlowConfig, store OIDCStateStore) *OIDCFlow {
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &OIDCFlow{
		config: config,
		store:  store,
		client: client,
	}
}

// SetAuthorizationEndpoint sets the authorization endpoint URL.
func (f *OIDCFlow) SetAuthorizationEndpoint(url string) {
	f.authorizationEndpoint = url
}

// SetTokenEndpoint sets the token endpoint URL.
func (f *OIDCFlow) SetTokenEndpoint(url string) {
	f.tokenEndpoint = url
}

// SetUserinfoEndpoint sets the userinfo endpoint URL.
func (f *OIDCFlow) SetUserinfoEndpoint(url string) {
	f.userinfoEndpoint = url
}

// SetJWKSURI sets the JWKS URI.
func (f *OIDCFlow) SetJWKSURI(uri string) {
	f.jwksURI = uri
}

// SetEndSessionEndpoint sets the end session endpoint URL.
// ADR-0015 Phase 4B: OIDC RP-Initiated Logout
func (f *OIDCFlow) SetEndSessionEndpoint(url string) {
	f.endSessionEndpoint = url
}

// GetEndSessionEndpoint returns the end session endpoint URL.
// Returns empty string if not available (IdP doesn't support RP-initiated logout).
// ADR-0015 Phase 4B: OIDC RP-Initiated Logout
func (f *OIDCFlow) GetEndSessionEndpoint() string {
	return f.endSessionEndpoint
}

// SetJWKSCache sets the JWKS cache (for testing or manual configuration).
func (f *OIDCFlow) SetJWKSCache(cache *JWKSCache) {
	f.jwksCache = cache
}

// SetIDTokenValidator sets the ID token validator (for testing or manual configuration).
func (f *OIDCFlow) SetIDTokenValidator(validator *IDTokenValidator) {
	f.idTokenValidator = validator
}

// GetJWKSCache returns the JWKS cache.
func (f *OIDCFlow) GetJWKSCache() *JWKSCache {
	return f.jwksCache
}

// GetIDTokenValidator returns the ID token validator.
func (f *OIDCFlow) GetIDTokenValidator() *IDTokenValidator {
	return f.idTokenValidator
}

// DiscoverEndpoints fetches OIDC discovery document and sets endpoints.
func (f *OIDCFlow) DiscoverEndpoints(ctx context.Context) error {
	discoveryURL := strings.TrimSuffix(f.config.IssuerURL, "/") + "/.well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("create discovery request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("discovery request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("discovery returned status %d", resp.StatusCode)
		}
		return fmt.Errorf("discovery returned status %d: %s", resp.StatusCode, string(body))
	}

	var discovery struct {
		Issuer                string `json:"issuer"`
		AuthorizationEndpoint string `json:"authorization_endpoint"`
		TokenEndpoint         string `json:"token_endpoint"`
		UserinfoEndpoint      string `json:"userinfo_endpoint"`
		JWKSURI               string `json:"jwks_uri"`
		EndSessionEndpoint    string `json:"end_session_endpoint"` // OIDC RP-Initiated Logout (Phase 4B)
	}

	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return fmt.Errorf("decode discovery document: %w", err)
	}

	f.authorizationEndpoint = discovery.AuthorizationEndpoint
	f.tokenEndpoint = discovery.TokenEndpoint
	f.userinfoEndpoint = discovery.UserinfoEndpoint
	f.jwksURI = discovery.JWKSURI
	f.endSessionEndpoint = discovery.EndSessionEndpoint // Phase 4B

	// Initialize JWKS cache if we have a JWKS URI
	if f.jwksURI != "" {
		jwksTTL := f.config.JWKSCacheTTL
		if jwksTTL == 0 {
			jwksTTL = 15 * time.Minute
		}
		f.jwksCache = NewJWKSCache(f.jwksURI, f.client, jwksTTL)

		// Initialize ID token validator
		// Use discovered issuer if config issuer is empty
		issuer := f.config.IssuerURL
		if discovery.Issuer != "" {
			issuer = discovery.Issuer
		}

		validationConfig := &IDTokenValidationConfig{
			Issuer:         issuer,
			ClientID:       f.config.ClientID,
			ClockSkew:      f.config.ClockSkew,
			RolesClaim:     f.config.RolesClaim,
			GroupsClaim:    f.config.GroupsClaim,
			UsernameClaims: f.config.UsernameClaims,
			DefaultRoles:   f.config.DefaultRoles,
		}
		f.idTokenValidator = NewIDTokenValidator(validationConfig, f.jwksCache)
	}

	return nil
}

// AuthorizationURL generates the authorization URL with state and optional PKCE.
func (f *OIDCFlow) AuthorizationURL(ctx context.Context, postLoginRedirect string) (string, error) {
	// Generate state parameter
	stateKey, err := GenerateStateParameter()
	if err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}

	// Prepare state data
	now := time.Now()
	stateData := &OIDCStateData{
		PostLoginRedirect: postLoginRedirect,
		CreatedAt:         now,
		ExpiresAt:         now.Add(f.config.StateTTL),
	}

	// Build URL parameters
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", f.config.ClientID)
	params.Set("redirect_uri", f.config.RedirectURL)
	params.Set("scope", strings.Join(f.config.Scopes, " "))
	params.Set("state", stateKey)

	// Add PKCE if enabled
	if f.config.PKCEEnabled {
		codeVerifier, err := GeneratePKCECodeVerifier()
		if err != nil {
			return "", fmt.Errorf("generate code verifier: %w", err)
		}

		stateData.CodeVerifier = codeVerifier
		codeChallenge := GeneratePKCECodeChallenge(codeVerifier)

		params.Set("code_challenge", codeChallenge)
		params.Set("code_challenge_method", "S256")
	}

	// Add nonce if enabled (for ID token validation)
	if f.config.NonceEnabled {
		nonce, err := GenerateNonce()
		if err != nil {
			return "", fmt.Errorf("generate nonce: %w", err)
		}
		stateData.Nonce = nonce
		params.Set("nonce", nonce)
	}

	// Store state
	if err := f.store.Store(ctx, stateKey, stateData); err != nil {
		return "", fmt.Errorf("store state: %w", err)
	}

	// Build authorization URL
	authURL := f.authorizationEndpoint + "?" + params.Encode()
	return authURL, nil
}

// GenerateNonce generates a cryptographically secure nonce for ID token validation.
func GenerateNonce() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// ValidateState validates and retrieves the state data.
// The state is consumed (deleted) after successful validation.
func (f *OIDCFlow) ValidateState(ctx context.Context, state string) (*OIDCStateData, error) {
	stateData, err := f.store.Get(ctx, state)
	if err != nil {
		if errors.Is(err, ErrStateNotFound) || errors.Is(err, ErrStateExpired) {
			return nil, ErrInvalidState
		}
		return nil, fmt.Errorf("get state: %w", err)
	}

	// Delete state to prevent replay attacks (single use)
	// Error is intentionally ignored - state was already validated and deletion is a cleanup step
	_ = f.store.Delete(ctx, state)

	return stateData, nil
}

// OIDCTokenResult holds the result of a token exchange.
type OIDCTokenResult struct {
	// AccessToken is the OAuth2 access token.
	AccessToken string

	// RefreshToken is the OAuth2 refresh token (if provided).
	RefreshToken string

	// IDToken is the OIDC ID token (if provided).
	IDToken string

	// TokenType is typically "Bearer".
	TokenType string

	// ExpiresIn is the token lifetime in seconds.
	ExpiresIn int

	// PostLoginRedirect is where to redirect after login.
	PostLoginRedirect string

	// Subject contains the authenticated user info (parsed from ID token).
	Subject *AuthSubject
}

// HandleCallback handles the authorization code callback.
func (f *OIDCFlow) HandleCallback(ctx context.Context, code, state string) (*OIDCTokenResult, error) {
	// Validate state
	stateData, err := f.ValidateState(ctx, state)
	if err != nil {
		return nil, err
	}

	// Exchange code for tokens
	result, err := f.exchangeCode(ctx, code, stateData.CodeVerifier)
	if err != nil {
		return nil, err
	}

	result.PostLoginRedirect = stateData.PostLoginRedirect

	// Validate and parse ID token if present and validator is configured
	if result.IDToken != "" && f.idTokenValidator != nil {
		claims, err := f.idTokenValidator.ValidateAndParse(ctx, result.IDToken, stateData.Nonce)
		if err != nil {
			return nil, fmt.Errorf("ID token validation failed: %w", err)
		}

		// Convert claims to AuthSubject
		result.Subject = claims.ToAuthSubject(f.config.UsernameClaims)
	}

	return result, nil
}

// exchangeCode exchanges the authorization code for tokens.
func (f *OIDCFlow) exchangeCode(ctx context.Context, code, codeVerifier string) (*OIDCTokenResult, error) {
	// Build request body
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", f.config.RedirectURL)
	data.Set("client_id", f.config.ClientID)

	if f.config.ClientSecret != "" {
		data.Set("client_secret", f.config.ClientSecret)
	}

	if codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
	}

	// Make request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("%w: status %d", ErrTokenExchangeFailed, resp.StatusCode)
		}
		return nil, fmt.Errorf("%w: status %d: %s", ErrTokenExchangeFailed, resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	return &OIDCTokenResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		IDToken:      tokenResp.IDToken,
		TokenType:    tokenResp.TokenType,
		ExpiresIn:    tokenResp.ExpiresIn,
	}, nil
}

// RefreshToken exchanges a refresh token for new tokens.
func (f *OIDCFlow) RefreshToken(ctx context.Context, refreshToken string) (*OIDCTokenResult, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", f.config.ClientID)

	if f.config.ClientSecret != "" {
		data.Set("client_secret", f.config.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("refresh failed: status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("refresh failed: status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode refresh response: %w", err)
	}

	return &OIDCTokenResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		IDToken:      tokenResp.IDToken,
		TokenType:    tokenResp.TokenType,
		ExpiresIn:    tokenResp.ExpiresIn,
	}, nil
}

// GeneratePKCECodeVerifier generates a cryptographically random code verifier.
// The verifier is 43-128 characters using URL-safe base64 encoding (RFC 7636).
func GeneratePKCECodeVerifier() (string, error) {
	// 32 bytes = 256 bits of entropy, encodes to 43 base64url characters
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// GeneratePKCECodeChallenge generates a code challenge from a verifier using S256.
func GeneratePKCECodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// GenerateStateParameter generates a cryptographically secure state parameter.
func GenerateStateParameter() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
