// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
)

// =====================================================
// OIDC Flow Tests
// ADR-0015: Zero Trust Authentication & Authorization
// =====================================================

// Test PKCE code verifier generation
func TestGeneratePKCECodeVerifier(t *testing.T) {
	verifier, err := GeneratePKCECodeVerifier()
	if err != nil {
		t.Fatalf("GeneratePKCECodeVerifier() error = %v", err)
	}

	// Code verifier must be 43-128 characters (RFC 7636)
	if len(verifier) < 43 || len(verifier) > 128 {
		t.Errorf("verifier length = %d, want 43-128", len(verifier))
	}

	// Verifier must be URL-safe (base64url without padding)
	for _, c := range verifier {
		if !isURLSafeChar(c) {
			t.Errorf("verifier contains non-URL-safe character: %c", c)
		}
	}
}

// Test PKCE code challenge generation
func TestGeneratePKCECodeChallenge(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	expected := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	challenge := GeneratePKCECodeChallenge(verifier)

	if challenge != expected {
		t.Errorf("challenge = %s, want %s", challenge, expected)
	}
}

// Test state parameter generation
func TestGenerateStateParameter(t *testing.T) {
	state, err := GenerateStateParameter()
	if err != nil {
		t.Fatalf("GenerateStateParameter() error = %v", err)
	}

	// State should be at least 32 bytes (256 bits) of entropy
	if len(state) < 32 {
		t.Errorf("state length = %d, want >= 32", len(state))
	}
}

// Test OIDCFlow creation
func TestNewOIDCFlow(t *testing.T) {
	config := &OIDCFlowConfig{
		IssuerURL:            "https://auth.example.com",
		ClientID:             "test-client",
		ClientSecret:         "test-secret",
		RedirectURL:          "https://app.example.com/callback",
		Scopes:               []string{"openid", "profile", "email"},
		PKCEEnabled:          true,
		StateTTL:             10 * time.Minute,
		SessionDuration:      24 * time.Hour,
		PostLoginRedirectURL: "/dashboard",
	}

	store := NewMemoryOIDCStateStore()
	flow := NewOIDCFlow(config, store)

	if flow == nil {
		t.Fatal("NewOIDCFlow() returned nil")
	}
}

// Test OIDC authorization URL generation
func TestOIDCFlow_AuthorizationURL(t *testing.T) {
	config := &OIDCFlowConfig{
		IssuerURL:   "https://auth.example.com",
		ClientID:    "test-client",
		RedirectURL: "https://app.example.com/callback",
		Scopes:      []string{"openid", "profile", "email"},
		PKCEEnabled: true,
		StateTTL:    10 * time.Minute,
	}

	store := NewMemoryOIDCStateStore()
	flow := NewOIDCFlow(config, store)
	flow.SetAuthorizationEndpoint("https://auth.example.com/authorize")

	ctx := context.Background()
	authURL, err := flow.AuthorizationURL(ctx, "/after-login")
	if err != nil {
		t.Fatalf("AuthorizationURL() error = %v", err)
	}

	// Parse the URL
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	// Verify base URL
	if u.Scheme != "https" || u.Host != "auth.example.com" || u.Path != "/authorize" {
		t.Errorf("base URL = %s://%s%s, want https://auth.example.com/authorize", u.Scheme, u.Host, u.Path)
	}

	// Verify query parameters
	q := u.Query()
	if q.Get("response_type") != "code" {
		t.Errorf("response_type = %s, want code", q.Get("response_type"))
	}
	if q.Get("client_id") != "test-client" {
		t.Errorf("client_id = %s, want test-client", q.Get("client_id"))
	}
	if q.Get("redirect_uri") != "https://app.example.com/callback" {
		t.Errorf("redirect_uri = %s, want https://app.example.com/callback", q.Get("redirect_uri"))
	}
	if !strings.Contains(q.Get("scope"), "openid") {
		t.Errorf("scope = %s, should contain openid", q.Get("scope"))
	}
	if q.Get("state") == "" {
		t.Error("state parameter is empty")
	}
	if q.Get("code_challenge") == "" {
		t.Error("code_challenge parameter is empty (PKCE enabled)")
	}
	if q.Get("code_challenge_method") != "S256" {
		t.Errorf("code_challenge_method = %s, want S256", q.Get("code_challenge_method"))
	}
}

// Test OIDC authorization URL without PKCE
func TestOIDCFlow_AuthorizationURL_NoPKCE(t *testing.T) {
	config := &OIDCFlowConfig{
		IssuerURL:   "https://auth.example.com",
		ClientID:    "test-client",
		RedirectURL: "https://app.example.com/callback",
		Scopes:      []string{"openid", "profile"},
		PKCEEnabled: false,
		StateTTL:    10 * time.Minute,
	}

	store := NewMemoryOIDCStateStore()
	flow := NewOIDCFlow(config, store)
	flow.SetAuthorizationEndpoint("https://auth.example.com/authorize")

	ctx := context.Background()
	authURL, err := flow.AuthorizationURL(ctx, "")
	if err != nil {
		t.Fatalf("AuthorizationURL() error = %v", err)
	}

	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	q := u.Query()
	if q.Get("code_challenge") != "" {
		t.Error("code_challenge should be empty when PKCE disabled")
	}
	if q.Get("code_challenge_method") != "" {
		t.Error("code_challenge_method should be empty when PKCE disabled")
	}
}

// Test state validation
func TestOIDCFlow_ValidateState(t *testing.T) {
	config := &OIDCFlowConfig{
		IssuerURL:   "https://auth.example.com",
		ClientID:    "test-client",
		RedirectURL: "https://app.example.com/callback",
		Scopes:      []string{"openid"},
		PKCEEnabled: true,
		StateTTL:    10 * time.Minute,
	}

	store := NewMemoryOIDCStateStore()
	flow := NewOIDCFlow(config, store)
	flow.SetAuthorizationEndpoint("https://auth.example.com/authorize")

	ctx := context.Background()
	authURL, err := flow.AuthorizationURL(ctx, "/dashboard")
	if err != nil {
		t.Fatalf("AuthorizationURL() error = %v", err)
	}

	// Extract state from URL
	u, _ := url.Parse(authURL)
	state := u.Query().Get("state")

	// Validate the state
	stateData, err := flow.ValidateState(ctx, state)
	if err != nil {
		t.Fatalf("ValidateState() error = %v", err)
	}

	if stateData.PostLoginRedirect != "/dashboard" {
		t.Errorf("PostLoginRedirect = %s, want /dashboard", stateData.PostLoginRedirect)
	}
	if stateData.CodeVerifier == "" && config.PKCEEnabled {
		t.Error("CodeVerifier should be set when PKCE enabled")
	}
}

// Test state validation with invalid state
func TestOIDCFlow_ValidateState_Invalid(t *testing.T) {
	config := &OIDCFlowConfig{
		IssuerURL:   "https://auth.example.com",
		ClientID:    "test-client",
		RedirectURL: "https://app.example.com/callback",
		Scopes:      []string{"openid"},
		StateTTL:    10 * time.Minute,
	}

	store := NewMemoryOIDCStateStore()
	flow := NewOIDCFlow(config, store)

	ctx := context.Background()
	_, err := flow.ValidateState(ctx, "invalid-state")
	if err == nil {
		t.Error("ValidateState() should fail for invalid state")
	}
}

// Test state expiration
func TestOIDCFlow_ValidateState_Expired(t *testing.T) {
	config := &OIDCFlowConfig{
		IssuerURL:   "https://auth.example.com",
		ClientID:    "test-client",
		RedirectURL: "https://app.example.com/callback",
		Scopes:      []string{"openid"},
		StateTTL:    1 * time.Millisecond, // Very short TTL
	}

	store := NewMemoryOIDCStateStore()
	flow := NewOIDCFlow(config, store)
	flow.SetAuthorizationEndpoint("https://auth.example.com/authorize")

	ctx := context.Background()
	authURL, err := flow.AuthorizationURL(ctx, "")
	if err != nil {
		t.Fatalf("AuthorizationURL() error = %v", err)
	}

	// Extract state
	u, _ := url.Parse(authURL)
	state := u.Query().Get("state")

	// Wait for state to expire
	time.Sleep(10 * time.Millisecond)

	_, err = flow.ValidateState(ctx, state)
	if err == nil {
		t.Error("ValidateState() should fail for expired state")
	}
}

// Test callback handler with mock token exchange
func TestOIDCFlow_HandleCallback(t *testing.T) {
	// Create a mock OIDC token endpoint
	tokenEndpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Verify request parameters
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if r.Form.Get("grant_type") != "authorization_code" {
			http.Error(w, "invalid grant_type", http.StatusBadRequest)
			return
		}

		// Return mock tokens
		resp := map[string]interface{}{
			"access_token":  "mock-access-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"refresh_token": "mock-refresh-token",
			"id_token":      createMockIDToken(t),
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer tokenEndpoint.Close()

	config := &OIDCFlowConfig{
		IssuerURL:       "https://auth.example.com",
		ClientID:        "test-client",
		ClientSecret:    "test-secret",
		RedirectURL:     "https://app.example.com/callback",
		Scopes:          []string{"openid", "profile", "email"},
		PKCEEnabled:     true,
		StateTTL:        10 * time.Minute,
		SessionDuration: 24 * time.Hour,
	}

	store := NewMemoryOIDCStateStore()
	flow := NewOIDCFlow(config, store)
	flow.SetAuthorizationEndpoint("https://auth.example.com/authorize")
	flow.SetTokenEndpoint(tokenEndpoint.URL)

	ctx := context.Background()

	// Generate authorization URL to create state
	authURL, err := flow.AuthorizationURL(ctx, "/dashboard")
	if err != nil {
		t.Fatalf("AuthorizationURL() error = %v", err)
	}

	// Extract state
	u, _ := url.Parse(authURL)
	state := u.Query().Get("state")

	// Handle callback
	result, err := flow.HandleCallback(ctx, "mock-auth-code", state)
	if err != nil {
		t.Fatalf("HandleCallback() error = %v", err)
	}

	if result.AccessToken != "mock-access-token" {
		t.Errorf("AccessToken = %s, want mock-access-token", result.AccessToken)
	}
	if result.RefreshToken != "mock-refresh-token" {
		t.Errorf("RefreshToken = %s, want mock-refresh-token", result.RefreshToken)
	}
	if result.PostLoginRedirect != "/dashboard" {
		t.Errorf("PostLoginRedirect = %s, want /dashboard", result.PostLoginRedirect)
	}
}

// Test callback with invalid state
func TestOIDCFlow_HandleCallback_InvalidState(t *testing.T) {
	config := &OIDCFlowConfig{
		IssuerURL:   "https://auth.example.com",
		ClientID:    "test-client",
		RedirectURL: "https://app.example.com/callback",
		Scopes:      []string{"openid"},
		StateTTL:    10 * time.Minute,
	}

	store := NewMemoryOIDCStateStore()
	flow := NewOIDCFlow(config, store)

	ctx := context.Background()
	_, err := flow.HandleCallback(ctx, "auth-code", "invalid-state")
	if err == nil {
		t.Error("HandleCallback() should fail with invalid state")
	}
}

// Test token refresh
func TestOIDCFlow_RefreshToken(t *testing.T) {
	// Create a mock token endpoint
	tokenEndpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if r.Form.Get("grant_type") != "refresh_token" {
			http.Error(w, "invalid grant_type", http.StatusBadRequest)
			return
		}

		resp := map[string]interface{}{
			"access_token":  "new-access-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"refresh_token": "new-refresh-token",
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer tokenEndpoint.Close()

	config := &OIDCFlowConfig{
		IssuerURL:    "https://auth.example.com",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "https://app.example.com/callback",
		Scopes:       []string{"openid"},
		StateTTL:     10 * time.Minute,
	}

	store := NewMemoryOIDCStateStore()
	flow := NewOIDCFlow(config, store)
	flow.SetTokenEndpoint(tokenEndpoint.URL)

	ctx := context.Background()
	result, err := flow.RefreshToken(ctx, "old-refresh-token")
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}

	if result.AccessToken != "new-access-token" {
		t.Errorf("AccessToken = %s, want new-access-token", result.AccessToken)
	}
	if result.RefreshToken != "new-refresh-token" {
		t.Errorf("RefreshToken = %s, want new-refresh-token", result.RefreshToken)
	}
}

// Test memory state store
func TestMemoryOIDCStateStore(t *testing.T) {
	store := NewMemoryOIDCStateStore()
	ctx := context.Background()

	// Test Store
	state := &OIDCStateData{
		CodeVerifier:      "test-verifier",
		PostLoginRedirect: "/dashboard",
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(10 * time.Minute),
	}

	stateKey := "test-state-key"
	err := store.Store(ctx, stateKey, state)
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Test Get
	retrieved, err := store.Get(ctx, stateKey)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.CodeVerifier != state.CodeVerifier {
		t.Errorf("CodeVerifier = %s, want %s", retrieved.CodeVerifier, state.CodeVerifier)
	}

	// Test Delete
	err = store.Delete(ctx, stateKey)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted
	_, err = store.Get(ctx, stateKey)
	if err == nil {
		t.Error("Get() should fail after Delete()")
	}
}

// Test state store cleanup
func TestMemoryOIDCStateStore_Cleanup(t *testing.T) {
	store := NewMemoryOIDCStateStore()
	ctx := context.Background()

	// Store expired state
	expiredState := &OIDCStateData{
		CodeVerifier: "expired-verifier",
		CreatedAt:    time.Now().Add(-20 * time.Minute),
		ExpiresAt:    time.Now().Add(-10 * time.Minute), // Expired
	}
	if err := store.Store(ctx, "expired-state", expiredState); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Store valid state
	validState := &OIDCStateData{
		CodeVerifier: "valid-verifier",
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}
	if err := store.Store(ctx, "valid-state", validState); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Cleanup
	count, err := store.CleanupExpired(ctx)
	if err != nil {
		t.Fatalf("CleanupExpired() error = %v", err)
	}
	if count != 1 {
		t.Errorf("CleanupExpired() count = %d, want 1", count)
	}

	// Verify expired is gone
	_, err = store.Get(ctx, "expired-state")
	if err == nil {
		t.Error("expired state should be cleaned up")
	}

	// Verify valid still exists
	_, err = store.Get(ctx, "valid-state")
	if err != nil {
		t.Errorf("valid state should still exist: %v", err)
	}
}

// Helper function to check if character is URL-safe
func isURLSafeChar(c rune) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_'
}

// Helper function to create a mock ID token for testing
func createMockIDToken(t *testing.T) string {
	t.Helper()

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{
		"iss": "https://auth.example.com",
		"sub": "user-123",
		"aud": "test-client",
		"exp": 9999999999,
		"iat": 1234567890,
		"preferred_username": "testuser",
		"email": "test@example.com",
		"email_verified": true,
		"roles": ["viewer", "editor"]
	}`))

	// For testing, we use an unsigned token (alg: none)
	return header + "." + payload + "."
}

// Test helper for code challenge calculation
func TestPKCECodeChallengeCalculation(t *testing.T) {
	// RFC 7636 Appendix B test vector
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

	// Calculate expected challenge
	hash := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(hash[:])

	challenge := GeneratePKCECodeChallenge(verifier)

	if challenge != expected {
		t.Errorf("challenge = %s, want %s", challenge, expected)
	}
}

// =====================================================
// Phase 4B.1: End Session Endpoint Tests
// =====================================================

func TestOIDCFlow_DiscoverEndpoints_EndSession(t *testing.T) {
	// Create mock server that returns discovery with end_session_endpoint
	var serverURL string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			w.Header().Set("Content-Type", "application/json")
			//nolint:errcheck // test
			json.NewEncoder(w).Encode(map[string]interface{}{
				"issuer":                 serverURL,
				"authorization_endpoint": serverURL + "/authorize",
				"token_endpoint":         serverURL + "/token",
				"jwks_uri":               serverURL + "/jwks",
				"end_session_endpoint":   serverURL + "/logout",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()
	serverURL = mockServer.URL

	config := &OIDCFlowConfig{
		IssuerURL: mockServer.URL,
		ClientID:  "test-client",
	}

	stateStore := NewMemoryOIDCStateStore()
	flow := NewOIDCFlow(config, stateStore)

	// Discover endpoints
	if err := flow.DiscoverEndpoints(context.Background()); err != nil {
		t.Fatalf("DiscoverEndpoints error: %v", err)
	}

	// Verify end_session_endpoint was parsed
	endSessionEndpoint := flow.GetEndSessionEndpoint()
	expectedEndpoint := mockServer.URL + "/logout"
	if endSessionEndpoint != expectedEndpoint {
		t.Errorf("GetEndSessionEndpoint = %s, want %s", endSessionEndpoint, expectedEndpoint)
	}
}

func TestOIDCFlow_DiscoverEndpoints_NoEndSession(t *testing.T) {
	// Create mock server that returns discovery WITHOUT end_session_endpoint
	var serverURL string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			w.Header().Set("Content-Type", "application/json")
			//nolint:errcheck // test
			json.NewEncoder(w).Encode(map[string]interface{}{
				"issuer":                 serverURL,
				"authorization_endpoint": serverURL + "/authorize",
				"token_endpoint":         serverURL + "/token",
				"jwks_uri":               serverURL + "/jwks",
				// No end_session_endpoint
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()
	serverURL = mockServer.URL

	config := &OIDCFlowConfig{
		IssuerURL: mockServer.URL,
		ClientID:  "test-client",
	}

	stateStore := NewMemoryOIDCStateStore()
	flow := NewOIDCFlow(config, stateStore)

	// Discover endpoints
	if err := flow.DiscoverEndpoints(context.Background()); err != nil {
		t.Fatalf("DiscoverEndpoints error: %v", err)
	}

	// Verify end_session_endpoint is empty (not nil, just empty string)
	endSessionEndpoint := flow.GetEndSessionEndpoint()
	if endSessionEndpoint != "" {
		t.Errorf("GetEndSessionEndpoint = %s, want empty string", endSessionEndpoint)
	}
}

func TestOIDCFlow_SetEndSessionEndpoint(t *testing.T) {
	config := &OIDCFlowConfig{
		IssuerURL: "https://auth.example.com",
		ClientID:  "test-client",
	}

	stateStore := NewMemoryOIDCStateStore()
	flow := NewOIDCFlow(config, stateStore)

	// Set endpoint manually
	flow.SetEndSessionEndpoint("https://auth.example.com/logout")

	// Verify it was set
	if flow.GetEndSessionEndpoint() != "https://auth.example.com/logout" {
		t.Errorf("GetEndSessionEndpoint = %s, want https://auth.example.com/logout", flow.GetEndSessionEndpoint())
	}
}
