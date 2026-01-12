// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality using Zitadel OIDC.
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
//
// This file implements the OIDC authorization code flow using the certified
// Zitadel OIDC library. It provides:
//   - Authorization URL generation with PKCE
//   - Authorization code exchange
//   - Token refresh
//   - State and nonce validation
//   - RP-initiated logout
//
// The flow is designed to be a drop-in replacement for the custom OIDCFlow
// while using Zitadel's certified implementations internally.
//
// References:
//   - https://github.com/zitadel/oidc
//   - RFC 7636 (PKCE)
//   - https://openid.net/specs/openid-connect-core-1_0.html
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/zitadel/oidc/v3/pkg/client/rp"
	"github.com/zitadel/oidc/v3/pkg/oidc"

	"github.com/tomtom215/cartographus/internal/logging"
)

// ZitadelOIDCFlow manages the OIDC authorization code flow with PKCE support.
// This is a wrapper around Zitadel's certified Relying Party that provides
// the same interface as the original OIDCFlow for compatibility.
//
// Key features:
//   - Certified PKCE implementation (RFC 7636)
//   - State parameter with TTL
//   - Nonce validation for ID tokens
//   - Token refresh support
//   - RP-initiated logout
//   - Back-channel logout support
type ZitadelOIDCFlow struct {
	rp         *ZitadelRelyingParty
	stateStore ZitadelStateStore
	config     *ZitadelFlowConfig

	// Back-channel logout support (ADR-0015 Phase 4B.3)
	// These are created lazily when needed for backwards compatibility.
	idTokenValidator *IDTokenValidator
	jwksCache        *JWKSCache
}

// ZitadelFlowConfig holds configuration specific to the flow.
// This extends the RP configuration with flow-specific settings.
type ZitadelFlowConfig struct {
	// StateTTL is how long state parameters are valid.
	// Default: 10 minutes
	StateTTL time.Duration

	// SessionDuration is how long sessions are valid after authentication.
	// Default: 24 hours
	SessionDuration time.Duration

	// DefaultPostLoginRedirect is the default redirect after login.
	// Default: "/"
	DefaultPostLoginRedirect string

	// ErrorRedirectURL is where to redirect on errors.
	// Default: "/login?error="
	ErrorRedirectURL string

	// NonceEnabled enables nonce generation for ID token validation.
	// Default: true
	NonceEnabled bool
}

// DefaultZitadelFlowConfig returns sensible defaults for the flow configuration.
func DefaultZitadelFlowConfig() *ZitadelFlowConfig {
	return &ZitadelFlowConfig{
		StateTTL:                 10 * time.Minute,
		SessionDuration:          24 * time.Hour,
		DefaultPostLoginRedirect: "/",
		ErrorRedirectURL:         "/login?error=",
		NonceEnabled:             true,
	}
}

// ZitadelStateStore defines the interface for storing OIDC state data.
// This is compatible with the existing OIDCStateStore interface.
type ZitadelStateStore interface {
	// Store saves state data with the given key.
	Store(ctx context.Context, key string, state *ZitadelStateData) error

	// Get retrieves state data by key.
	Get(ctx context.Context, key string) (*ZitadelStateData, error)

	// Delete removes state data by key.
	Delete(ctx context.Context, key string) error

	// CleanupExpired removes all expired states.
	CleanupExpired(ctx context.Context) (int, error)
}

// ZitadelStateData holds state information during the authorization flow.
// This is compatible with OIDCStateData for migration purposes.
type ZitadelStateData struct {
	// CodeVerifier is the PKCE code verifier (if PKCE enabled).
	CodeVerifier string

	// PostLoginRedirect is where to redirect after successful login.
	PostLoginRedirect string

	// Nonce is the nonce for ID token validation.
	Nonce string

	// CreatedAt is when the state was created.
	CreatedAt time.Time

	// ExpiresAt is when the state expires.
	ExpiresAt time.Time
}

// IsExpired checks if the state has expired.
func (s *ZitadelStateData) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// ZitadelMemoryStateStore is an in-memory implementation of ZitadelStateStore.
// Thread-safe for concurrent access.
type ZitadelMemoryStateStore struct {
	mu     sync.RWMutex
	states map[string]*ZitadelStateData
}

// NewZitadelMemoryStateStore creates a new in-memory state store.
func NewZitadelMemoryStateStore() *ZitadelMemoryStateStore {
	return &ZitadelMemoryStateStore{
		states: make(map[string]*ZitadelStateData),
	}
}

// Store saves state data with the given key.
func (s *ZitadelMemoryStateStore) Store(ctx context.Context, key string, state *ZitadelStateData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Deep copy to prevent mutation
	stored := &ZitadelStateData{
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
func (s *ZitadelMemoryStateStore) Get(ctx context.Context, key string) (*ZitadelStateData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.states[key]
	if !ok {
		return nil, ErrStateNotFound
	}

	if state.IsExpired() {
		return nil, ErrStateExpired
	}

	// Return a copy to prevent mutation
	return &ZitadelStateData{
		CodeVerifier:      state.CodeVerifier,
		PostLoginRedirect: state.PostLoginRedirect,
		Nonce:             state.Nonce,
		CreatedAt:         state.CreatedAt,
		ExpiresAt:         state.ExpiresAt,
	}, nil
}

// Delete removes state data by key.
func (s *ZitadelMemoryStateStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, key)
	return nil
}

// CleanupExpired removes all expired states.
func (s *ZitadelMemoryStateStore) CleanupExpired(ctx context.Context) (int, error) {
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

// Compile-time interface assertion
var _ ZitadelStateStore = (*ZitadelMemoryStateStore)(nil)

// NewZitadelOIDCFlow creates a new OIDC flow manager using Zitadel.
//
// Parameters:
//   - relyingParty: The Zitadel Relying Party (must be initialized)
//   - stateStore: Store for state parameters (use NewZitadelMemoryStateStore)
//   - config: Flow-specific configuration (use DefaultZitadelFlowConfig)
//
// Returns:
//   - Configured flow manager
func NewZitadelOIDCFlow(
	relyingParty *ZitadelRelyingParty,
	stateStore ZitadelStateStore,
	config *ZitadelFlowConfig,
) *ZitadelOIDCFlow {
	if config == nil {
		config = DefaultZitadelFlowConfig()
	}
	return &ZitadelOIDCFlow{
		rp:         relyingParty,
		stateStore: stateStore,
		config:     config,
	}
}

// ZitadelTokenResult holds the result of a token exchange.
// This is compatible with OIDCTokenResult for migration purposes.
type ZitadelTokenResult struct {
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

// AuthorizationURL generates the authorization URL with state and PKCE.
// The user should be redirected to this URL to initiate login.
//
// Parameters:
//   - ctx: Context for state storage
//   - postLoginRedirect: Where to redirect after successful login
//
// Returns:
//   - Authorization URL to redirect user to
//   - Error if state cannot be stored
func (f *ZitadelOIDCFlow) AuthorizationURL(ctx context.Context, postLoginRedirect string) (string, error) {
	// Generate state parameter
	stateKey, err := generateSecureRandom(32)
	if err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}

	// Prepare state data
	now := time.Now()
	stateData := &ZitadelStateData{
		PostLoginRedirect: postLoginRedirect,
		CreatedAt:         now,
		ExpiresAt:         now.Add(f.config.StateTTL),
	}

	// Generate nonce if enabled
	if f.config.NonceEnabled {
		nonce, err := generateSecureRandom(32)
		if err != nil {
			return "", fmt.Errorf("generate nonce: %w", err)
		}
		stateData.Nonce = nonce
	}

	// Build authorization URL using Zitadel
	// Zitadel handles PKCE internally when configured
	authURL := rp.AuthURL(stateKey, f.rp.RelyingParty())

	// Add nonce to the URL if enabled
	if stateData.Nonce != "" {
		parsedURL, err := url.Parse(authURL)
		if err != nil {
			return "", fmt.Errorf("parse auth URL: %w", err)
		}
		query := parsedURL.Query()
		query.Set("nonce", stateData.Nonce)
		parsedURL.RawQuery = query.Encode()
		authURL = parsedURL.String()
	}

	// Store state
	if err := f.stateStore.Store(ctx, stateKey, stateData); err != nil {
		return "", fmt.Errorf("store state: %w", err)
	}

	logging.Debug().
		Str("state", stateKey[:8]+"...").
		Bool("nonce", f.config.NonceEnabled).
		Msg("Generated OIDC authorization URL")

	return authURL, nil
}

// HandleCallback handles the authorization code callback.
// This validates the state, exchanges the code for tokens, and validates the ID token.
//
// Parameters:
//   - ctx: Context for HTTP requests
//   - code: Authorization code from callback
//   - state: State parameter from callback
//
// Returns:
//   - Token result including parsed claims
//   - Error if validation or exchange fails
func (f *ZitadelOIDCFlow) HandleCallback(ctx context.Context, code, state string) (*ZitadelTokenResult, error) {
	// Validate and consume state
	stateData, err := f.validateAndConsumeState(ctx, state)
	if err != nil {
		return nil, err
	}

	// Exchange code for tokens using Zitadel
	tokens, err := rp.CodeExchange[*oidc.IDTokenClaims](ctx, code, f.rp.RelyingParty())
	if err != nil {
		logging.Error().Err(err).Msg("Token exchange failed")
		return nil, fmt.Errorf("%w: %s", ErrTokenExchangeFailed, err.Error())
	}

	// Build result
	result := &ZitadelTokenResult{
		AccessToken:       tokens.AccessToken,
		RefreshToken:      tokens.RefreshToken,
		TokenType:         tokens.TokenType,
		PostLoginRedirect: stateData.PostLoginRedirect,
	}

	// Set expiry if available
	if !tokens.Expiry.IsZero() {
		result.ExpiresIn = int(time.Until(tokens.Expiry).Seconds())
	}

	// Extract ID token if present
	if tokens.IDToken != "" {
		result.IDToken = tokens.IDToken
	}

	// Validate nonce if enabled
	if f.config.NonceEnabled && stateData.Nonce != "" {
		if tokens.IDTokenClaims == nil {
			return nil, fmt.Errorf("%w: no ID token claims for nonce validation", ErrInvalidCredentials)
		}
		if tokens.IDTokenClaims.Nonce != stateData.Nonce {
			logging.Warn().
				Str("expected", stateData.Nonce[:8]+"...").
				Str("got", tokens.IDTokenClaims.Nonce[:min(8, len(tokens.IDTokenClaims.Nonce))]+"...").
				Msg("Nonce mismatch")
			return nil, fmt.Errorf("%w: nonce mismatch", ErrInvalidCredentials)
		}
	}

	// Map claims to AuthSubject if we have ID token claims
	if tokens.IDTokenClaims != nil {
		claimsMapping := f.rp.GetClaimsMapping()
		result.Subject = MapZitadelClaimsToAuthSubject(
			tokens.IDTokenClaims,
			&claimsMapping,
			f.rp.GetDefaultRoles(),
		)
	}

	logging.Info().
		Str("user", result.Subject.Username).
		Int("expires_in", result.ExpiresIn).
		Msg("OIDC token exchange successful")

	return result, nil
}

// validateAndConsumeState validates and removes the state from the store.
// The state is consumed (deleted) after successful validation to prevent replay.
func (f *ZitadelOIDCFlow) validateAndConsumeState(ctx context.Context, state string) (*ZitadelStateData, error) {
	stateData, err := f.stateStore.Get(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidState, err.Error())
	}

	// Delete state to prevent replay attacks (single use)
	if err := f.stateStore.Delete(ctx, state); err != nil {
		// Log but don't fail - state was already validated
		logging.Warn().Err(err).Msg("Failed to delete state after validation")
	}

	return stateData, nil
}

// RefreshToken exchanges a refresh token for new tokens.
//
// Parameters:
//   - ctx: Context for HTTP requests
//   - refreshToken: The refresh token to exchange
//
// Returns:
//   - New token result
//   - Error if refresh fails
func (f *ZitadelOIDCFlow) RefreshToken(ctx context.Context, refreshToken string) (*ZitadelTokenResult, error) {
	// Use Zitadel's refresh token endpoint
	tokens, err := rp.RefreshTokens[*oidc.IDTokenClaims](ctx, f.rp.RelyingParty(), refreshToken, "", "")
	if err != nil {
		logging.Debug().Err(err).Msg("Token refresh failed")
		return nil, fmt.Errorf("refresh failed: %w", err)
	}

	result := &ZitadelTokenResult{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
	}

	if !tokens.Expiry.IsZero() {
		result.ExpiresIn = int(time.Until(tokens.Expiry).Seconds())
	}

	if tokens.IDToken != "" {
		result.IDToken = tokens.IDToken
	}

	if tokens.IDTokenClaims != nil {
		claimsMapping := f.rp.GetClaimsMapping()
		result.Subject = MapZitadelClaimsToAuthSubject(
			tokens.IDTokenClaims,
			&claimsMapping,
			f.rp.GetDefaultRoles(),
		)
	}

	logging.Debug().
		Int("expires_in", result.ExpiresIn).
		Msg("Token refresh successful")

	return result, nil
}

// GetEndSessionEndpoint returns the IdP's end session endpoint.
// Returns empty string if RP-initiated logout is not supported.
func (f *ZitadelOIDCFlow) GetEndSessionEndpoint() string {
	return f.rp.GetEndSessionEndpoint()
}

// BuildLogoutURL constructs the IdP logout URL with query parameters.
//
// Parameters:
//   - idTokenHint: The ID token for session identification
//   - postLogoutRedirectURI: Where to redirect after logout
//
// Returns:
//   - Complete logout URL
//   - Error if endpoint is not available or URL is malformed
func (f *ZitadelOIDCFlow) BuildLogoutURL(idTokenHint, postLogoutRedirectURI string) (string, error) {
	endSessionEndpoint := f.GetEndSessionEndpoint()
	if endSessionEndpoint == "" {
		return "", fmt.Errorf("end session endpoint not available")
	}

	logoutURL, err := url.Parse(endSessionEndpoint)
	if err != nil {
		return "", fmt.Errorf("parse end session URL: %w", err)
	}

	params := logoutURL.Query()
	if idTokenHint != "" {
		params.Set("id_token_hint", idTokenHint)
	}
	if postLogoutRedirectURI != "" {
		params.Set("post_logout_redirect_uri", postLogoutRedirectURI)
	}

	// Add state for security
	state, err := generateSecureRandom(16)
	if err == nil {
		params.Set("state", state)
	}

	logoutURL.RawQuery = params.Encode()
	return logoutURL.String(), nil
}

// GetRelyingParty returns the underlying Zitadel Relying Party.
func (f *ZitadelOIDCFlow) GetRelyingParty() *ZitadelRelyingParty {
	return f.rp
}

// GetConfig returns the flow configuration.
func (f *ZitadelOIDCFlow) GetConfig() *ZitadelFlowConfig {
	return f.config
}

// GetIDTokenValidator returns the ID token validator for back-channel logout.
// ADR-0015 Phase 4B.3: Back-channel logout JWT validation.
// The validator is created lazily on first access.
func (f *ZitadelOIDCFlow) GetIDTokenValidator() *IDTokenValidator {
	if f.idTokenValidator == nil && f.rp != nil {
		f.initializeBackChannelComponents()
	}
	return f.idTokenValidator
}

// GetJWKSCache returns the JWKS cache for back-channel logout.
// ADR-0015 Phase 4B.3: Back-channel logout JWT signature verification.
// The cache is created lazily on first access.
func (f *ZitadelOIDCFlow) GetJWKSCache() *JWKSCache {
	if f.jwksCache == nil && f.rp != nil {
		f.initializeBackChannelComponents()
	}
	return f.jwksCache
}

// initializeBackChannelComponents creates the JWKS cache and ID token validator
// for back-channel logout support. This is done lazily to avoid overhead
// when back-channel logout is not used.
func (f *ZitadelOIDCFlow) initializeBackChannelComponents() {
	if f.rp == nil {
		return
	}

	// Get JWKS URI from the discovery document
	jwksURI := f.rp.GetJWKSURI()
	if jwksURI == "" {
		logging.Warn().Msg("JWKS URI not available, back-channel logout will not work")
		return
	}

	// Create JWKS cache with 1 hour TTL
	f.jwksCache = NewJWKSCache(jwksURI, nil, time.Hour)

	// Create ID token validation config
	claimsMapping := f.rp.GetClaimsMapping()
	validationConfig := &IDTokenValidationConfig{
		Issuer:         f.rp.Issuer(),
		ClientID:       f.rp.GetClientID(),
		RolesClaim:     claimsMapping.RolesClaim,
		GroupsClaim:    claimsMapping.GroupsClaim,
		UsernameClaims: claimsMapping.UsernameClaims,
	}

	// Create ID token validator
	f.idTokenValidator = NewIDTokenValidator(validationConfig, f.jwksCache)

	logging.Debug().
		Str("issuer", validationConfig.Issuer).
		Str("jwks_uri", jwksURI).
		Msg("Initialized back-channel logout components")
}

// DiscoverEndpoints is a no-op for ZitadelOIDCFlow since discovery
// happens during RelyingParty creation. This method exists for
// compatibility with the original OIDCFlow interface.
func (f *ZitadelOIDCFlow) DiscoverEndpoints(ctx context.Context) error {
	// Discovery already happened in NewZitadelRelyingParty
	// Verify endpoints are available
	if f.rp.GetAuthURL() == "" {
		return fmt.Errorf("authorization endpoint not discovered")
	}
	return nil
}

// generateSecureRandom generates a cryptographically secure random string.
// The output is base64url encoded without padding.
func generateSecureRandom(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// NewZitadelOIDCFlowFromConfig creates a flow from OIDCFlowConfig.
// This is a convenience function for integration with the existing configuration.
//
// Parameters:
//   - ctx: Context for OIDC discovery
//   - cfg: Existing OIDCFlowConfig from the application
//   - stateStore: Store for state parameters
//
// Returns:
//   - Configured flow or error
func NewZitadelOIDCFlowFromConfig(
	ctx context.Context,
	cfg *OIDCFlowConfig,
	stateStore ZitadelStateStore,
) (*ZitadelOIDCFlow, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Map to ZitadelRPConfig
	rpConfig := &ZitadelRPConfig{
		IssuerURL:             cfg.IssuerURL,
		ClientID:              cfg.ClientID,
		ClientSecret:          cfg.ClientSecret,
		RedirectURL:           cfg.RedirectURL,
		Scopes:                cfg.Scopes,
		PKCEEnabled:           cfg.PKCEEnabled,
		PostLogoutRedirectURI: cfg.PostLoginRedirectURL, // Note: different field name
		ClaimsMapping: ZitadelClaimsMappingConfig{
			RolesClaim:     cfg.RolesClaim,
			GroupsClaim:    cfg.GroupsClaim,
			UsernameClaims: cfg.UsernameClaims,
		},
		DefaultRoles: cfg.DefaultRoles,
	}

	// Use configured HTTP client or timeout
	if cfg.HTTPClient != nil {
		rpConfig.HTTPClient = cfg.HTTPClient
	}

	// Create the Relying Party
	relyingParty, err := NewZitadelRelyingParty(ctx, rpConfig)
	if err != nil {
		return nil, fmt.Errorf("create OIDC client: %w", err)
	}

	// Map flow config
	flowConfig := &ZitadelFlowConfig{
		StateTTL:                 cfg.StateTTL,
		SessionDuration:          cfg.SessionDuration,
		DefaultPostLoginRedirect: cfg.PostLoginRedirectURL,
		NonceEnabled:             cfg.NonceEnabled,
	}

	// Apply defaults
	if flowConfig.StateTTL == 0 {
		flowConfig.StateTTL = 10 * time.Minute
	}
	if flowConfig.SessionDuration == 0 {
		flowConfig.SessionDuration = 24 * time.Hour
	}
	if flowConfig.DefaultPostLoginRedirect == "" {
		flowConfig.DefaultPostLoginRedirect = "/"
	}

	return NewZitadelOIDCFlow(relyingParty, stateStore, flowConfig), nil
}

// Note: OIDCFlowConfig is defined in oidc_flow.go
