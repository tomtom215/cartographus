// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OIDC support.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/golang-jwt/jwt/v5"
)

// MockOIDCServer is a mock OIDC server for testing.
// It provides all the endpoints needed for OIDC authentication:
// - Discovery endpoint (/.well-known/openid-configuration)
// - JWKS endpoint (/jwks)
// - Authorization endpoint (/authorize)
// - Token endpoint (/token)
// - Userinfo endpoint (/userinfo)
type MockOIDCServer struct {
	Server *httptest.Server

	// Configuration
	Issuer       string
	ClientID     string
	ClientSecret string

	// AllowedRedirectURIs is a list of allowed redirect URI patterns.
	// For security, the mock server only redirects to URIs that match these patterns.
	// Patterns can be exact URIs or use "*" as a wildcard for localhost ports.
	// Default: ["http://localhost:*", "http://127.0.0.1:*"]
	AllowedRedirectURIs []string

	// RSA key pair for signing tokens
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	keyID      string

	// State for authorization codes
	mu              sync.Mutex
	authCodes       map[string]*mockAuthCode
	refreshTokens   map[string]*mockRefreshToken
	expectedState   string
	expectedNonce   string
	authCodeTimeout time.Duration

	// Configurable response behavior
	TokenExpiresIn int // Seconds until token expires (default: 3600)
	IDTokenClaims  IDTokenClaimsBuilder
}

// mockAuthCode represents an authorization code.
type mockAuthCode struct {
	Code         string
	ClientID     string
	RedirectURI  string
	CodeVerifier string
	Nonce        string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	Used         bool
	Claims       map[string]interface{}
}

// mockRefreshToken represents a refresh token.
type mockRefreshToken struct {
	Token     string
	UserID    string
	ClientID  string
	Scopes    []string
	CreatedAt time.Time
	ExpiresAt time.Time
	Claims    map[string]interface{}
}

// IDTokenClaimsBuilder allows customizing ID token claims.
type IDTokenClaimsBuilder struct {
	Subject           string
	Email             string
	EmailVerified     bool
	Name              string
	PreferredUsername string
	Roles             []string
	Groups            []string
	CustomClaims      map[string]interface{}
}

// NewMockOIDCServer creates a new mock OIDC server.
func NewMockOIDCServer(clientID, clientSecret string) (*MockOIDCServer, error) {
	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate RSA key: %w", err)
	}

	keyID := generateRandomString(16)

	mock := &MockOIDCServer{
		ClientID:        clientID,
		ClientSecret:    clientSecret,
		privateKey:      privateKey,
		publicKey:       &privateKey.PublicKey,
		keyID:           keyID,
		authCodes:       make(map[string]*mockAuthCode),
		refreshTokens:   make(map[string]*mockRefreshToken),
		authCodeTimeout: 5 * time.Minute,
		TokenExpiresIn:  3600,
		// Default allowed redirect URIs for testing - only localhost/loopback
		AllowedRedirectURIs: []string{
			"http://localhost:*",
			"http://127.0.0.1:*",
			"https://localhost:*",
			"https://127.0.0.1:*",
		},
		IDTokenClaims: IDTokenClaimsBuilder{
			Subject:           "user123",
			Email:             "user@example.com",
			EmailVerified:     true,
			Name:              "Test User",
			PreferredUsername: "testuser",
			Roles:             []string{"viewer"},
			Groups:            []string{"users"},
		},
	}

	// Create HTTP handler
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", mock.handleDiscovery)
	mux.HandleFunc("/jwks", mock.handleJWKS)
	mux.HandleFunc("/.well-known/jwks.json", mock.handleJWKS) // For back-channel logout (ADR-0015 Phase 4B.3)
	mux.HandleFunc("/authorize", mock.handleAuthorize)
	mux.HandleFunc("/token", mock.handleToken)
	mux.HandleFunc("/userinfo", mock.handleUserinfo)

	mock.Server = httptest.NewServer(mux)
	mock.Issuer = mock.Server.URL

	return mock, nil
}

// Close shuts down the mock server.
func (m *MockOIDCServer) Close() {
	if m.Server != nil {
		m.Server.Close()
	}
}

// AddAllowedRedirectURI adds a redirect URI pattern to the allowed list.
// Patterns can include "*" as a wildcard for port numbers.
func (m *MockOIDCServer) AddAllowedRedirectURI(pattern string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AllowedRedirectURIs = append(m.AllowedRedirectURIs, pattern)
}

// isRedirectURIAllowed checks if the given redirect URI matches any allowed pattern.
// This prevents open redirect vulnerabilities by validating against a whitelist.
func (m *MockOIDCServer) isRedirectURIAllowed(redirectURI string) bool {
	parsedURI, err := url.Parse(redirectURI)
	if err != nil {
		return false
	}

	for _, pattern := range m.AllowedRedirectURIs {
		if matchesRedirectPattern(parsedURI, pattern) {
			return true
		}
	}
	return false
}

// matchesRedirectPattern checks if a parsed URI matches an allowed pattern.
// Patterns can use "*" as a wildcard for the port portion.
func matchesRedirectPattern(uri *url.URL, pattern string) bool {
	// Parse the pattern
	patternURL, err := url.Parse(pattern)
	if err != nil {
		return false
	}

	// Check scheme
	if uri.Scheme != patternURL.Scheme {
		return false
	}

	// Check host (without port)
	uriHost := uri.Hostname()
	patternHost := patternURL.Hostname()
	if uriHost != patternHost {
		return false
	}

	// Check port - "*" matches any port
	patternPort := patternURL.Port()
	if patternPort != "*" && patternPort != uri.Port() {
		return false
	}

	// Check path - if pattern has a path, it must match
	if patternURL.Path != "" && patternURL.Path != "/" {
		if !strings.HasPrefix(uri.Path, patternURL.Path) {
			return false
		}
	}

	return true
}

// SetExpectedState sets the expected state parameter for authorization.
func (m *MockOIDCServer) SetExpectedState(state string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expectedState = state
}

// SetExpectedNonce sets the expected nonce parameter for ID token validation.
func (m *MockOIDCServer) SetExpectedNonce(nonce string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expectedNonce = nonce
}

// CreateAuthorizationCode creates an authorization code for testing.
func (m *MockOIDCServer) CreateAuthorizationCode(redirectURI, codeVerifier, nonce string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	code := generateRandomString(32)
	now := time.Now()

	m.authCodes[code] = &mockAuthCode{
		Code:         code,
		ClientID:     m.ClientID,
		RedirectURI:  redirectURI,
		CodeVerifier: codeVerifier,
		Nonce:        nonce,
		CreatedAt:    now,
		ExpiresAt:    now.Add(m.authCodeTimeout),
		Used:         false,
		Claims:       m.buildClaimsMap(),
	}

	return code
}

// GetKeyID returns the key ID used for signing.
func (m *MockOIDCServer) GetKeyID() string {
	return m.keyID
}

// GetPublicKey returns the public key for verification.
func (m *MockOIDCServer) GetPublicKey() *rsa.PublicKey {
	return m.publicKey
}

// buildClaimsMap builds the claims map from IDTokenClaimsBuilder.
func (m *MockOIDCServer) buildClaimsMap() map[string]interface{} {
	claims := map[string]interface{}{
		"sub":                m.IDTokenClaims.Subject,
		"email":              m.IDTokenClaims.Email,
		"email_verified":     m.IDTokenClaims.EmailVerified,
		"name":               m.IDTokenClaims.Name,
		"preferred_username": m.IDTokenClaims.PreferredUsername,
	}

	if len(m.IDTokenClaims.Roles) > 0 {
		claims["roles"] = m.IDTokenClaims.Roles
	}
	if len(m.IDTokenClaims.Groups) > 0 {
		claims["groups"] = m.IDTokenClaims.Groups
	}

	for k, v := range m.IDTokenClaims.CustomClaims {
		claims[k] = v
	}

	return claims
}

// handleDiscovery handles the OIDC discovery endpoint.
func (m *MockOIDCServer) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	discovery := map[string]interface{}{
		"issuer":                 m.Issuer,
		"authorization_endpoint": m.Issuer + "/authorize",
		"token_endpoint":         m.Issuer + "/token",
		"userinfo_endpoint":      m.Issuer + "/userinfo",
		"jwks_uri":               m.Issuer + "/jwks",
		"end_session_endpoint":   m.Issuer + "/logout", // Phase 4B: RP-Initiated Logout
		"response_types_supported": []string{
			"code",
			"id_token",
			"token id_token",
			"code id_token",
		},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "roles", "groups"},
		"claims_supported": []string{
			"sub", "iss", "aud", "exp", "iat", "nonce",
			"name", "email", "email_verified", "preferred_username",
			"roles", "groups",
		},
		"code_challenge_methods_supported": []string{"S256", "plain"},
		// Phase 4B: Back-channel logout support
		"backchannel_logout_supported":         true,
		"backchannel_logout_session_supported": true,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(discovery); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleJWKS handles the JWKS endpoint.
func (m *MockOIDCServer) handleJWKS(w http.ResponseWriter, r *http.Request) {
	// Encode the public key components
	n := base64.RawURLEncoding.EncodeToString(m.publicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(m.publicKey.E)).Bytes())

	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"kid": m.keyID,
				"use": "sig",
				"alg": "RS256",
				"n":   n,
				"e":   e,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jwks); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleAuthorize handles the authorization endpoint.
func (m *MockOIDCServer) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	// This is typically called by the browser, but we can use it to generate codes
	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")
	nonce := r.URL.Query().Get("nonce")
	codeChallenge := r.URL.Query().Get("code_challenge")
	_ = codeChallenge // Not used in mock, but logged

	if clientID != m.ClientID {
		http.Error(w, "Invalid client_id", http.StatusBadRequest)
		return
	}

	// Validate redirect_uri is a valid URL to prevent open redirect vulnerabilities
	parsedURI, err := url.Parse(redirectURI)
	if err != nil || parsedURI.Scheme == "" || parsedURI.Host == "" {
		http.Error(w, "Invalid redirect_uri", http.StatusBadRequest)
		return
	}

	// Validate redirect_uri against allowed patterns to prevent open redirect attacks
	if !m.isRedirectURIAllowed(redirectURI) {
		http.Error(w, "redirect_uri not in allowed list", http.StatusBadRequest)
		return
	}

	// Create authorization code
	code := m.CreateAuthorizationCode(redirectURI, "", nonce)

	// Build redirect URL safely using url.Values to prevent injection
	// Security: redirectURI is validated via isRedirectURIAllowed() (line 377) which checks
	// against AllowedRedirectURIs whitelist. Only pre-configured patterns are accepted.
	// codeql[go/unvalidated-url-redirection]: False positive - redirect_uri validated against whitelist
	query := parsedURI.Query()
	query.Set("code", code)
	query.Set("state", state)
	parsedURI.RawQuery = query.Encode()
	http.Redirect(w, r, parsedURI.String(), http.StatusFound) //nolint:gosec // redirect_uri validated via isRedirectURIAllowed
}

// handleToken handles the token endpoint.
func (m *MockOIDCServer) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	grantType := r.FormValue("grant_type")

	switch grantType {
	case "authorization_code":
		m.handleAuthorizationCodeGrant(w, r)
	case "refresh_token": //nolint:goconst // OAuth2 grant type, not metadata key
		m.handleRefreshTokenGrant(w, r)
	default:
		http.Error(w, "unsupported_grant_type", http.StatusBadRequest)
	}
}

// handleAuthorizationCodeGrant handles the authorization code grant.
func (m *MockOIDCServer) handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	redirectURI := r.FormValue("redirect_uri")
	codeVerifier := r.FormValue("code_verifier")

	m.mu.Lock()
	authCode, ok := m.authCodes[code]
	if !ok {
		m.mu.Unlock()
		m.sendTokenError(w, "invalid_grant", "Authorization code not found")
		return
	}

	// Mark as used
	if authCode.Used {
		m.mu.Unlock()
		m.sendTokenError(w, "invalid_grant", "Authorization code already used")
		return
	}
	authCode.Used = true

	// Validate
	if time.Now().After(authCode.ExpiresAt) {
		m.mu.Unlock()
		m.sendTokenError(w, "invalid_grant", "Authorization code expired")
		return
	}

	if clientID != m.ClientID {
		m.mu.Unlock()
		m.sendTokenError(w, "invalid_client", "Invalid client_id")
		return
	}

	if m.ClientSecret != "" && clientSecret != m.ClientSecret {
		m.mu.Unlock()
		m.sendTokenError(w, "invalid_client", "Invalid client_secret")
		return
	}

	if redirectURI != authCode.RedirectURI {
		m.mu.Unlock()
		m.sendTokenError(w, "invalid_grant", "Invalid redirect_uri")
		return
	}

	// Validate PKCE code verifier if present
	if authCode.CodeVerifier != "" && codeVerifier != authCode.CodeVerifier {
		// In a real implementation, we'd verify S256 challenge
		// For testing, we just check if verifier is present
		if codeVerifier == "" {
			m.mu.Unlock()
			m.sendTokenError(w, "invalid_grant", "Code verifier required")
			return
		}
	}

	claims := authCode.Claims
	nonce := authCode.Nonce
	m.mu.Unlock()

	// Generate tokens
	m.sendTokenResponse(w, claims, nonce)
}

// handleRefreshTokenGrant handles the refresh token grant.
func (m *MockOIDCServer) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.FormValue("refresh_token")
	clientID := r.FormValue("client_id")

	m.mu.Lock()
	stored, ok := m.refreshTokens[refreshToken]
	if !ok {
		m.mu.Unlock()
		m.sendTokenError(w, "invalid_grant", "Refresh token not found")
		return
	}

	if clientID != stored.ClientID {
		m.mu.Unlock()
		m.sendTokenError(w, "invalid_client", "Invalid client_id")
		return
	}

	if time.Now().After(stored.ExpiresAt) {
		m.mu.Unlock()
		m.sendTokenError(w, "invalid_grant", "Refresh token expired")
		return
	}

	claims := stored.Claims
	m.mu.Unlock()

	// Generate new tokens
	m.sendTokenResponse(w, claims, "")
}

// sendTokenResponse sends a successful token response.
func (m *MockOIDCServer) sendTokenResponse(w http.ResponseWriter, claims map[string]interface{}, nonce string) {
	now := time.Now()
	expiry := now.Add(time.Duration(m.TokenExpiresIn) * time.Second)

	// Generate access token (simple opaque token for mock)
	accessToken := generateRandomString(32)

	// Generate refresh token
	refreshToken := generateRandomString(32)

	// Store refresh token
	m.mu.Lock()
	sub, _ := claims["sub"].(string) //nolint:errcheck // type assertion ok value intentionally ignored in test mock
	m.refreshTokens[refreshToken] = &mockRefreshToken{
		Token:     refreshToken,
		UserID:    sub,
		ClientID:  m.ClientID,
		CreatedAt: now,
		ExpiresAt: now.Add(7 * 24 * time.Hour), // 7 days
		Claims:    claims,
	}
	m.mu.Unlock()

	// Generate ID token
	idToken, err := m.generateIDToken(claims, nonce, expiry)
	if err != nil {
		m.sendTokenError(w, "server_error", "Failed to generate ID token")
		return
	}

	response := map[string]interface{}{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    m.TokenExpiresIn,
		"refresh_token": refreshToken,
		"id_token":      idToken,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// generateIDToken generates a signed ID token.
func (m *MockOIDCServer) generateIDToken(claims map[string]interface{}, nonce string, expiry time.Time) (string, error) {
	now := time.Now()

	// Build JWT claims
	jwtClaims := jwt.MapClaims{
		"iss": m.Issuer,
		"sub": claims["sub"],
		"aud": m.ClientID,
		"exp": expiry.Unix(),
		"iat": now.Unix(),
	}

	// Add nonce if present
	if nonce != "" {
		jwtClaims["nonce"] = nonce
	}

	// Add additional claims
	for k, v := range claims {
		if k != "sub" { // sub already added
			jwtClaims[k] = v
		}
	}

	// Create token with RS256 signing
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwtClaims)
	token.Header["kid"] = m.keyID

	// Sign the token
	tokenString, err := token.SignedString(m.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}

// sendTokenError sends a token error response.
func (m *MockOIDCServer) sendTokenError(w http.ResponseWriter, errorCode, description string) {
	response := map[string]string{
		"error":             errorCode,
		"error_description": description,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	//nolint:errcheck // error response encoding failure is not recoverable in test mock
	json.NewEncoder(w).Encode(response)
}

// StoreRefreshToken stores a refresh token for testing purposes.
// This allows tests to pre-configure refresh tokens that can be used
// for token refresh operations.
// ADR-0015: Zero Trust Authentication (Zitadel Amendment)
func (m *MockOIDCServer) StoreRefreshToken(token, subject string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Include claims that are required by the Zitadel library for token refresh
	claims := map[string]interface{}{
		"sub":                subject,
		"preferred_username": "testuser",
		"email":              "testuser@example.com",
	}

	m.refreshTokens[token] = &mockRefreshToken{
		Token:     token,
		UserID:    subject,
		ClientID:  m.ClientID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 day expiry
		Claims:    claims,
	}
}

// handleUserinfo handles the userinfo endpoint.
func (m *MockOIDCServer) handleUserinfo(w http.ResponseWriter, r *http.Request) {
	// Check for Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// In a real implementation, we'd validate the access token
	// For testing, we just return the configured claims
	response := m.buildClaimsMap()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// GenerateExpiredIDToken generates an expired ID token for testing error cases.
func (m *MockOIDCServer) GenerateExpiredIDToken() (string, error) {
	claims := m.buildClaimsMap()
	expiry := time.Now().Add(-1 * time.Hour) // Already expired
	return m.generateIDToken(claims, "", expiry)
}

// GenerateIDTokenWithWrongAudience generates an ID token with wrong audience.
func (m *MockOIDCServer) GenerateIDTokenWithWrongAudience() (string, error) {
	claims := m.buildClaimsMap()
	expiry := time.Now().Add(time.Hour)

	// Create token with wrong audience
	jwtClaims := jwt.MapClaims{
		"iss": m.Issuer,
		"sub": claims["sub"],
		"aud": "wrong-client-id",
		"exp": expiry.Unix(),
		"iat": time.Now().Unix(),
	}

	for k, v := range claims {
		if k != "sub" {
			jwtClaims[k] = v
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwtClaims)
	token.Header["kid"] = m.keyID

	return token.SignedString(m.privateKey)
}

// GenerateIDTokenWithWrongIssuer generates an ID token with wrong issuer.
func (m *MockOIDCServer) GenerateIDTokenWithWrongIssuer() (string, error) {
	claims := m.buildClaimsMap()
	expiry := time.Now().Add(time.Hour)

	jwtClaims := jwt.MapClaims{
		"iss": "https://wrong-issuer.example.com",
		"sub": claims["sub"],
		"aud": m.ClientID,
		"exp": expiry.Unix(),
		"iat": time.Now().Unix(),
	}

	for k, v := range claims {
		if k != "sub" {
			jwtClaims[k] = v
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwtClaims)
	token.Header["kid"] = m.keyID

	return token.SignedString(m.privateKey)
}

// GenerateValidIDToken generates a valid ID token for testing.
func (m *MockOIDCServer) GenerateValidIDToken(nonce string) (string, error) {
	claims := m.buildClaimsMap()
	expiry := time.Now().Add(time.Hour)
	return m.generateIDToken(claims, nonce, expiry)
}

// generateRandomString generates a random string of the specified length.
func generateRandomString(length int) string {
	bytes := make([]byte, length)
	//nolint:errcheck // crypto/rand.Read error is negligible for test mock token generation
	rand.Read(bytes)
	return base64.RawURLEncoding.EncodeToString(bytes)[:length]
}

// GenerateLogoutToken generates a back-channel logout token.
// ADR-0015 Phase 4B.3: Test helper for back-channel logout
func (m *MockOIDCServer) GenerateLogoutToken(subject, sessionID string) (string, error) {
	return m.GenerateLogoutTokenForIssuer(m.Issuer, m.ClientID, subject, sessionID)
}

// GenerateLogoutTokenForIssuer generates a logout token with a custom issuer.
// This is used to test signature verification with forged tokens.
func (m *MockOIDCServer) GenerateLogoutTokenForIssuer(issuer, clientID, subject, sessionID string) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"iss": issuer,
		"sub": subject,
		"aud": clientID,
		"iat": now.Unix(),
		"jti": generateRandomString(16),
		"events": map[string]interface{}{
			"http://schemas.openid.net/event/backchannel-logout": map[string]interface{}{},
		},
	}

	if sessionID != "" {
		claims["sid"] = sessionID
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = m.keyID

	return token.SignedString(m.privateKey)
}

// GenerateLogoutTokenWithoutKid generates a logout token without kid header.
// Used to test rejection of tokens with missing kid.
func (m *MockOIDCServer) GenerateLogoutTokenWithoutKid(subject, sessionID string) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"iss": m.Issuer,
		"sub": subject,
		"aud": m.ClientID,
		"iat": now.Unix(),
		"jti": generateRandomString(16),
		"events": map[string]interface{}{
			"http://schemas.openid.net/event/backchannel-logout": map[string]interface{}{},
		},
	}

	if sessionID != "" {
		claims["sid"] = sessionID
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	// Don't set kid header

	return token.SignedString(m.privateKey)
}

// GenerateLogoutTokenWithAlgorithm generates a logout token with a specific algorithm.
// Used to test rejection of unsupported algorithms.
func (m *MockOIDCServer) GenerateLogoutTokenWithAlgorithm(alg, subject, sessionID string) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"iss": m.Issuer,
		"sub": subject,
		"aud": m.ClientID,
		"iat": now.Unix(),
		"jti": generateRandomString(16),
		"events": map[string]interface{}{
			"http://schemas.openid.net/event/backchannel-logout": map[string]interface{}{},
		},
	}

	if sessionID != "" {
		claims["sid"] = sessionID
	}

	var token *jwt.Token
	var signingKey interface{}

	switch alg {
	case "HS256":
		token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signingKey = []byte("test-secret-for-hs256")
	case "RS384":
		token = jwt.NewWithClaims(jwt.SigningMethodRS384, claims)
		signingKey = m.privateKey
	case "RS512":
		token = jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
		signingKey = m.privateKey
	default:
		token = jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		signingKey = m.privateKey
	}

	token.Header["kid"] = m.keyID

	return token.SignedString(signingKey)
}
