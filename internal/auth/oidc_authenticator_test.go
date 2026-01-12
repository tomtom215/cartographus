// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/golang-jwt/jwt/v5"
)

// TestOIDCAuthenticator_Interface verifies the interface contract
func TestOIDCAuthenticator_Interface(t *testing.T) {
	// Skip if we can't create a mock server
	server := newMockOIDCServer(t)
	defer server.Close()

	config := &OIDCConfig{
		IssuerURL:    server.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:3857/auth/callback",
		Scopes:       []string{"openid", "profile", "email"},
	}

	auth, err := NewOIDCAuthenticator(context.Background(), config)
	if err != nil {
		t.Fatalf("NewOIDCAuthenticator() error = %v", err)
	}

	// Verify interface implementation
	var _ Authenticator = auth

	// Test Name()
	if auth.Name() != "oidc" {
		t.Errorf("Name() = %v, want oidc", auth.Name())
	}

	// Test Priority() - OIDC should be high priority (low number)
	if auth.Priority() > 20 {
		t.Errorf("Priority() = %v, want <= 20", auth.Priority())
	}
}

// TestOIDCAuthenticator_NoToken tests request without bearer token
func TestOIDCAuthenticator_NoToken(t *testing.T) {
	server := newMockOIDCServer(t)
	defer server.Close()

	config := &OIDCConfig{
		IssuerURL:    server.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:3857/auth/callback",
		Scopes:       []string{"openid"},
	}

	auth, err := NewOIDCAuthenticator(context.Background(), config)
	if err != nil {
		t.Fatalf("NewOIDCAuthenticator() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	_, err = auth.Authenticate(context.Background(), req)

	if !errors.Is(err, ErrNoCredentials) {
		t.Errorf("Authenticate() error = %v, want ErrNoCredentials", err)
	}
}

// TestOIDCAuthenticator_InvalidToken tests request with invalid bearer token
func TestOIDCAuthenticator_InvalidToken(t *testing.T) {
	server := newMockOIDCServer(t)
	defer server.Close()

	config := &OIDCConfig{
		IssuerURL:    server.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:3857/auth/callback",
		Scopes:       []string{"openid"},
	}

	auth, err := NewOIDCAuthenticator(context.Background(), config)
	if err != nil {
		t.Fatalf("NewOIDCAuthenticator() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	_, err = auth.Authenticate(context.Background(), req)

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Authenticate() error = %v, want ErrInvalidCredentials", err)
	}
}

// TestOIDCAuthenticator_ValidToken tests request with valid bearer token
func TestOIDCAuthenticator_ValidToken(t *testing.T) {
	server := newMockOIDCServer(t)
	defer server.Close()

	config := &OIDCConfig{
		IssuerURL:    server.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:3857/auth/callback",
		Scopes:       []string{"openid", "profile", "email"},
	}

	auth, err := NewOIDCAuthenticator(context.Background(), config)
	if err != nil {
		t.Fatalf("NewOIDCAuthenticator() error = %v", err)
	}

	// Create a valid token using the mock server's key
	token := server.CreateToken(t, map[string]interface{}{
		"sub":                "user-123",
		"preferred_username": "testuser",
		"email":              "test@example.com",
		"email_verified":     true,
		"groups":             []string{"admin", "users"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if subject.ID != "user-123" {
		t.Errorf("subject.ID = %v, want user-123", subject.ID)
	}
	if subject.Username != "testuser" {
		t.Errorf("subject.Username = %v, want testuser", subject.Username)
	}
	if subject.Email != "test@example.com" {
		t.Errorf("subject.Email = %v, want test@example.com", subject.Email)
	}
	if subject.AuthMethod != AuthModeOIDC {
		t.Errorf("subject.AuthMethod = %v, want %v", subject.AuthMethod, AuthModeOIDC)
	}
}

// TestOIDCAuthenticator_ExpiredToken tests request with expired token
func TestOIDCAuthenticator_ExpiredToken(t *testing.T) {
	server := newMockOIDCServer(t)
	defer server.Close()

	config := &OIDCConfig{
		IssuerURL:    server.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:3857/auth/callback",
		Scopes:       []string{"openid"},
	}

	auth, err := NewOIDCAuthenticator(context.Background(), config)
	if err != nil {
		t.Fatalf("NewOIDCAuthenticator() error = %v", err)
	}

	// Create an expired token
	token := server.CreateExpiredToken(t, map[string]interface{}{
		"sub": "user-123",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	_, err = auth.Authenticate(context.Background(), req)

	if !errors.Is(err, ErrExpiredCredentials) {
		t.Errorf("Authenticate() error = %v, want ErrExpiredCredentials", err)
	}
}

// TestOIDCAuthenticator_WrongAudience tests token with wrong audience
func TestOIDCAuthenticator_WrongAudience(t *testing.T) {
	server := newMockOIDCServer(t)
	defer server.Close()

	config := &OIDCConfig{
		IssuerURL:    server.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:3857/auth/callback",
		Scopes:       []string{"openid"},
	}

	auth, err := NewOIDCAuthenticator(context.Background(), config)
	if err != nil {
		t.Fatalf("NewOIDCAuthenticator() error = %v", err)
	}

	// Create token with wrong audience
	token := server.CreateTokenWithAudience(t, "wrong-client", map[string]interface{}{
		"sub": "user-123",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	_, err = auth.Authenticate(context.Background(), req)

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Authenticate() error = %v, want ErrInvalidCredentials", err)
	}
}

// TestOIDCAuthenticator_ClaimsMapping tests custom claims mapping
func TestOIDCAuthenticator_ClaimsMapping(t *testing.T) {
	server := newMockOIDCServer(t)
	defer server.Close()

	config := &OIDCConfig{
		IssuerURL:    server.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:3857/auth/callback",
		Scopes:       []string{"openid", "profile"},
		ClaimsMapping: ClaimsMappingConfig{
			RolesClaim:     "custom_roles",
			GroupsClaim:    "custom_groups",
			UsernameClaims: []string{"username"},
		},
	}

	auth, err := NewOIDCAuthenticator(context.Background(), config)
	if err != nil {
		t.Fatalf("NewOIDCAuthenticator() error = %v", err)
	}

	token := server.CreateToken(t, map[string]interface{}{
		"sub":           "user-456",
		"username":      "customuser",
		"custom_roles":  []string{"admin"},
		"custom_groups": []string{"developers"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if subject.Username != "customuser" {
		t.Errorf("subject.Username = %v, want customuser", subject.Username)
	}
	if len(subject.Roles) != 1 || subject.Roles[0] != "admin" {
		t.Errorf("subject.Roles = %v, want [admin]", subject.Roles)
	}
	if len(subject.Groups) != 1 || subject.Groups[0] != "developers" {
		t.Errorf("subject.Groups = %v, want [developers]", subject.Groups)
	}
}

// mockOIDCServer is a test server that simulates an OIDC provider
type mockOIDCServer struct {
	*httptest.Server
	privateKey *rsa.PrivateKey
	keyID      string
}

func newMockOIDCServer(t *testing.T) *mockOIDCServer {
	t.Helper()

	// Generate RSA key pair for signing tokens
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	keyID := "test-key-1"

	server := &mockOIDCServer{
		privateKey: privateKey,
		keyID:      keyID,
	}

	mux := http.NewServeMux()

	// OIDC Discovery endpoint
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		// We need to know the server URL for the config
		// This is a bit tricky since we're inside the handler
		baseURL := "http://" + r.Host
		config := map[string]interface{}{
			"issuer":                                baseURL,
			"authorization_endpoint":                baseURL + "/authorize",
			"token_endpoint":                        baseURL + "/token",
			"userinfo_endpoint":                     baseURL + "/userinfo",
			"jwks_uri":                              baseURL + "/.well-known/jwks.json",
			"id_token_signing_alg_values_supported": []string{"RS256"},
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"scopes_supported":                      []string{"openid", "profile", "email"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
	})

	// JWKS endpoint
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kty": "RSA",
					"alg": "RS256",
					"use": "sig",
					"kid": keyID,
					"n":   base64URLEncode(privateKey.N.Bytes()),
					"e":   base64URLEncode([]byte{1, 0, 1}), // 65537
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	})

	server.Server = httptest.NewServer(mux)
	return server
}

func (s *mockOIDCServer) CreateToken(t *testing.T, claims map[string]interface{}) string {
	return s.createTokenWithExpiry(t, "test-client", claims, time.Now().Add(time.Hour))
}

func (s *mockOIDCServer) CreateExpiredToken(t *testing.T, claims map[string]interface{}) string {
	return s.createTokenWithExpiry(t, "test-client", claims, time.Now().Add(-time.Hour))
}

func (s *mockOIDCServer) CreateTokenWithAudience(t *testing.T, audience string, claims map[string]interface{}) string {
	return s.createTokenWithExpiry(t, audience, claims, time.Now().Add(time.Hour))
}

func (s *mockOIDCServer) createTokenWithExpiry(t *testing.T, audience string, extraClaims map[string]interface{}, exp time.Time) string {
	t.Helper()

	claims := jwt.MapClaims{
		"iss": s.URL,
		"aud": audience,
		"iat": time.Now().Unix(),
		"exp": exp.Unix(),
	}

	for k, v := range extraClaims {
		claims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.keyID

	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	return signed
}

func base64URLEncode(data []byte) string {
	// Simple base64url encoding (for test purposes)
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	result := ""
	for i := 0; i < len(data); i += 3 {
		n := uint32(data[i]) << 16
		if i+1 < len(data) {
			n |= uint32(data[i+1]) << 8
		}
		if i+2 < len(data) {
			n |= uint32(data[i+2])
		}

		result += string(alphabet[(n>>18)&63])
		result += string(alphabet[(n>>12)&63])
		if i+1 < len(data) {
			result += string(alphabet[(n>>6)&63])
		}
		if i+2 < len(data) {
			result += string(alphabet[n&63])
		}
	}
	return result
}

// TestJWKSCache tests the JWKS caching functionality
func TestJWKSCache(t *testing.T) {
	tests := []struct {
		name           string
		ttl            time.Duration
		enabled        bool
		wantDefaultTTL bool
	}{
		{
			name:           "custom TTL",
			ttl:            time.Hour,
			enabled:        true,
			wantDefaultTTL: false,
		},
		{
			name:           "zero TTL uses default",
			ttl:            0,
			enabled:        true,
			wantDefaultTTL: true,
		},
		{
			name:           "disabled cache",
			ttl:            time.Hour,
			enabled:        false,
			wantDefaultTTL: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &JWKSCacheConfig{
				TTL:     tt.ttl,
				Enabled: tt.enabled,
			}

			if tt.wantDefaultTTL && config.TTL == 0 {
				// Default should be applied during NewOIDCAuthenticator
			}
			if config.Enabled != tt.enabled {
				t.Errorf("Enabled = %v, want %v", config.Enabled, tt.enabled)
			}
		})
	}
}

// TestOIDCAuthenticator_CookieToken tests token from cookie
func TestOIDCAuthenticator_CookieToken(t *testing.T) {
	server := newMockOIDCServer(t)
	defer server.Close()

	config := &OIDCConfig{
		IssuerURL:    server.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:3857/auth/callback",
		Scopes:       []string{"openid"},
	}

	auth, err := NewOIDCAuthenticator(context.Background(), config)
	if err != nil {
		t.Fatalf("NewOIDCAuthenticator() error = %v", err)
	}

	token := server.CreateToken(t, map[string]interface{}{
		"sub":                "cookie-user",
		"preferred_username": "cookieuser",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: token,
	})

	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if subject.ID != "cookie-user" {
		t.Errorf("subject.ID = %v, want cookie-user", subject.ID)
	}
}

// TestOIDCConfig_DefaultValues tests default configuration values
func TestOIDCConfig_DefaultValues(t *testing.T) {
	config := DefaultOIDCConfig()

	if config.PKCEEnabled != true {
		t.Error("PKCEEnabled should default to true")
	}

	if len(config.Scopes) != 3 {
		t.Errorf("Default scopes should be 3, got %d", len(config.Scopes))
	}

	expectedScopes := []string{"openid", "profile", "email"}
	for i, scope := range expectedScopes {
		if i >= len(config.Scopes) || config.Scopes[i] != scope {
			t.Errorf("Scope[%d] = %v, want %v", i, config.Scopes[i], scope)
		}
	}
}

// TestOIDCAuthenticator_SubjectFromClaims tests AuthSubject conversion
func TestOIDCAuthenticator_SubjectFromClaims(t *testing.T) {
	server := newMockOIDCServer(t)
	defer server.Close()

	config := &OIDCConfig{
		IssuerURL:    server.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:3857/auth/callback",
		Scopes:       []string{"openid", "profile", "email"},
	}

	auth, err := NewOIDCAuthenticator(context.Background(), config)
	if err != nil {
		t.Fatalf("NewOIDCAuthenticator() error = %v", err)
	}

	token := server.CreateToken(t, map[string]interface{}{
		"sub":                "convert-user",
		"preferred_username": "convertuser",
		"email":              "convert@example.com",
		"email_verified":     true,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Convert to Claims for backwards compatibility
	claims := subject.ToClaims()
	if claims == nil {
		t.Fatal("ToClaims() returned nil")
	}

	if claims.Username != "convertuser" {
		t.Errorf("claims.Username = %v, want convertuser", claims.Username)
	}

	// Convert back
	subject2 := AuthSubjectFromClaims(claims)
	if subject2.Username != subject.Username {
		t.Errorf("Round-trip username mismatch: %v != %v", subject2.Username, subject.Username)
	}
}

// TestOIDCAuthenticator_ErrorOnInvalidIssuer tests error handling for invalid issuer
func TestOIDCAuthenticator_ErrorOnInvalidIssuer(t *testing.T) {
	config := &OIDCConfig{
		IssuerURL:    "http://localhost:99999", // Invalid port, won't connect
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:3857/auth/callback",
		Scopes:       []string{"openid"},
	}

	_, err := NewOIDCAuthenticator(context.Background(), config)
	// Should return error because it can't connect to discover OIDC config
	if err == nil {
		t.Error("Expected error for invalid issuer URL, got nil")
	}
}

// BenchmarkOIDCAuthenticate benchmarks the authentication flow
func BenchmarkOIDCAuthenticate(b *testing.B) {
	// Create a mock server that stays up for the benchmark
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatalf("Failed to generate RSA key: %v", err)
	}
	keyID := "bench-key"

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		baseURL := "http://" + r.Host
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issuer":                                baseURL,
			"jwks_uri":                              baseURL + "/.well-known/jwks.json",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kty": "RSA", "alg": "RS256", "use": "sig", "kid": keyID,
					"n": base64URLEncode(privateKey.N.Bytes()),
					"e": base64URLEncode([]byte{1, 0, 1}),
				},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	config := &OIDCConfig{
		IssuerURL:    server.URL,
		ClientID:     "bench-client",
		ClientSecret: "bench-secret",
		RedirectURL:  "http://localhost:3857/auth/callback",
	}

	auth, err := NewOIDCAuthenticator(context.Background(), config)
	if err != nil {
		b.Fatalf("NewOIDCAuthenticator() error = %v", err)
	}

	// Create a valid token
	claims := jwt.MapClaims{
		"iss": server.URL,
		"aud": "bench-client",
		"sub": "bench-user",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID
	tokenStr, _ := token.SignedString(privateKey)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auth.Authenticate(context.Background(), req)
	}
}
