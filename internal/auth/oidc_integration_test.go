// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OIDC support.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"testing"
	"time"
)

func TestOIDCFlow_IntegrationWithMockServer(t *testing.T) {
	// Create mock OIDC server
	mockServer, err := NewMockOIDCServer("test-client", "test-secret")
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer mockServer.Close()

	// Create OIDC flow
	config := &OIDCFlowConfig{
		IssuerURL:       mockServer.Issuer,
		ClientID:        "test-client",
		ClientSecret:    "test-secret",
		RedirectURL:     "http://localhost:3857/callback",
		Scopes:          []string{"openid", "profile", "email"},
		PKCEEnabled:     true,
		StateTTL:        10 * time.Minute,
		SessionDuration: 24 * time.Hour,
		JWKSCacheTTL:    15 * time.Minute,
		NonceEnabled:    true,
		UsernameClaims:  []string{"preferred_username", "name", "email"},
	}

	stateStore := NewMemoryOIDCStateStore()
	flow := NewOIDCFlow(config, stateStore)

	ctx := context.Background()

	// Test endpoint discovery
	t.Run("DiscoverEndpoints", func(t *testing.T) {
		err := flow.DiscoverEndpoints(ctx)
		if err != nil {
			t.Fatalf("DiscoverEndpoints failed: %v", err)
		}

		// Verify JWKS cache was initialized
		if flow.GetJWKSCache() == nil {
			t.Error("JWKS cache should be initialized after discovery")
		}

		// Verify ID token validator was initialized
		if flow.GetIDTokenValidator() == nil {
			t.Error("ID token validator should be initialized after discovery")
		}
	})

	// Test authorization URL generation
	t.Run("AuthorizationURL", func(t *testing.T) {
		authURL, err := flow.AuthorizationURL(ctx, "/dashboard")
		if err != nil {
			t.Fatalf("AuthorizationURL failed: %v", err)
		}

		if authURL == "" {
			t.Error("Authorization URL should not be empty")
		}

		// URL should contain required parameters
		if !contains(authURL, "response_type=code") {
			t.Error("Authorization URL should contain response_type=code")
		}
		if !contains(authURL, "client_id=test-client") {
			t.Error("Authorization URL should contain client_id")
		}
		if !contains(authURL, "state=") {
			t.Error("Authorization URL should contain state")
		}
		if !contains(authURL, "nonce=") {
			t.Error("Authorization URL should contain nonce (NonceEnabled=true)")
		}
		if !contains(authURL, "code_challenge=") {
			t.Error("Authorization URL should contain code_challenge (PKCE enabled)")
		}
	})

	// Test full authorization code flow
	t.Run("FullAuthorizationCodeFlow", func(t *testing.T) {
		// Generate authorization URL to create state
		authURL, err := flow.AuthorizationURL(ctx, "/dashboard")
		if err != nil {
			t.Fatalf("AuthorizationURL failed: %v", err)
		}

		// Extract state from URL
		state := extractQueryParam(authURL, "state")
		if state == "" {
			t.Fatal("Failed to extract state from authorization URL")
		}

		// Get the stored state data to get the nonce
		stateData, err := stateStore.Get(ctx, state)
		if err != nil {
			t.Fatalf("Failed to get state data: %v", err)
		}

		// Create authorization code with matching nonce
		code := mockServer.CreateAuthorizationCode(
			"http://localhost:3857/callback",
			stateData.CodeVerifier,
			stateData.Nonce,
		)

		// Re-store the state (it was consumed by Get)
		if err := stateStore.Store(ctx, state, stateData); err != nil {
			t.Fatalf("Failed to re-store state: %v", err)
		}

		// Handle callback
		result, err := flow.HandleCallback(ctx, code, state)
		if err != nil {
			t.Fatalf("HandleCallback failed: %v", err)
		}

		// Verify tokens
		if result.AccessToken == "" {
			t.Error("Access token should not be empty")
		}
		if result.RefreshToken == "" {
			t.Error("Refresh token should not be empty")
		}
		if result.IDToken == "" {
			t.Error("ID token should not be empty")
		}
		if result.PostLoginRedirect != "/dashboard" {
			t.Errorf("PostLoginRedirect = %q, want %q", result.PostLoginRedirect, "/dashboard")
		}

		// Verify subject was parsed from ID token
		if result.Subject == nil {
			t.Fatal("Subject should be parsed from ID token")
		}
		if result.Subject.ID != "user123" {
			t.Errorf("Subject.ID = %q, want %q", result.Subject.ID, "user123")
		}
		if result.Subject.Email != "user@example.com" {
			t.Errorf("Subject.Email = %q, want %q", result.Subject.Email, "user@example.com")
		}
		if result.Subject.Username != "testuser" {
			t.Errorf("Subject.Username = %q, want %q", result.Subject.Username, "testuser")
		}
	})
}

func TestIDTokenValidator_Integration(t *testing.T) {
	// Create mock OIDC server
	mockServer, err := NewMockOIDCServer("test-client", "test-secret")
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer mockServer.Close()

	// Create JWKS cache
	jwksCache := NewJWKSCache(mockServer.Server.URL+"/jwks", nil, 15*time.Minute)

	// Create validator
	validatorConfig := &IDTokenValidationConfig{
		Issuer:         mockServer.Issuer,
		ClientID:       "test-client",
		ClockSkew:      1 * time.Minute,
		RolesClaim:     "roles",
		GroupsClaim:    "groups",
		UsernameClaims: []string{"preferred_username", "name", "email"},
	}
	validator := NewIDTokenValidator(validatorConfig, jwksCache)

	ctx := context.Background()

	t.Run("ValidateValidToken", func(t *testing.T) {
		// Generate valid token
		token, err := mockServer.GenerateValidIDToken("")
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		claims, err := validator.ValidateAndParse(ctx, token, "")
		if err != nil {
			t.Fatalf("ValidateAndParse failed: %v", err)
		}

		if claims.Subject != "user123" {
			t.Errorf("Subject = %q, want %q", claims.Subject, "user123")
		}
		if claims.Email != "user@example.com" {
			t.Errorf("Email = %q, want %q", claims.Email, "user@example.com")
		}
	})

	t.Run("ValidateTokenWithNonce", func(t *testing.T) {
		nonce := "test-nonce-123"
		token, err := mockServer.GenerateValidIDToken(nonce)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		claims, err := validator.ValidateAndParse(ctx, token, nonce)
		if err != nil {
			t.Fatalf("ValidateAndParse failed: %v", err)
		}

		if claims.Nonce != nonce {
			t.Errorf("Nonce = %q, want %q", claims.Nonce, nonce)
		}
	})

	t.Run("RejectTokenWithWrongNonce", func(t *testing.T) {
		token, err := mockServer.GenerateValidIDToken("correct-nonce")
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		_, err = validator.ValidateAndParse(ctx, token, "wrong-nonce")
		if err == nil {
			t.Error("Expected error for wrong nonce")
		}
	})

	t.Run("RejectExpiredToken", func(t *testing.T) {
		token, err := mockServer.GenerateExpiredIDToken()
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		_, err = validator.ValidateAndParse(ctx, token, "")
		if err == nil {
			t.Error("Expected error for expired token")
		}
	})

	t.Run("RejectTokenWithWrongAudience", func(t *testing.T) {
		token, err := mockServer.GenerateIDTokenWithWrongAudience()
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		_, err = validator.ValidateAndParse(ctx, token, "")
		if err == nil {
			t.Error("Expected error for wrong audience")
		}
	})

	t.Run("RejectTokenWithWrongIssuer", func(t *testing.T) {
		token, err := mockServer.GenerateIDTokenWithWrongIssuer()
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		_, err = validator.ValidateAndParse(ctx, token, "")
		if err == nil {
			t.Error("Expected error for wrong issuer")
		}
	})
}

func TestIDTokenClaims_ToAuthSubject(t *testing.T) {
	claims := &IDTokenClaims{
		Subject:           "user123",
		Issuer:            "https://auth.example.com",
		Email:             "user@example.com",
		EmailVerified:     true,
		Name:              "Test User",
		PreferredUsername: "testuser",
		Roles:             []string{"admin", "viewer"},
		Groups:            []string{"developers"},
		IssuedAt:          time.Now().Unix(),
		ExpiresAt:         time.Now().Add(time.Hour).Unix(),
		RawClaims: map[string]interface{}{
			"custom_claim": "custom_value",
		},
	}

	usernameClaims := []string{"preferred_username", "name", "email"}
	subject := claims.ToAuthSubject(usernameClaims)

	if subject.ID != "user123" {
		t.Errorf("ID = %q, want %q", subject.ID, "user123")
	}
	if subject.Username != "testuser" {
		t.Errorf("Username = %q, want %q", subject.Username, "testuser")
	}
	if subject.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", subject.Email, "user@example.com")
	}
	if !subject.EmailVerified {
		t.Error("EmailVerified should be true")
	}
	if len(subject.Roles) != 2 {
		t.Errorf("Roles count = %d, want 2", len(subject.Roles))
	}
	if subject.AuthMethod != AuthModeOIDC {
		t.Errorf("AuthMethod = %q, want %q", subject.AuthMethod, AuthModeOIDC)
	}
	if subject.Provider != "oidc" {
		t.Errorf("Provider = %q, want %q", subject.Provider, "oidc")
	}
}

func TestJWKSCache_Integration(t *testing.T) {
	// Create mock OIDC server
	mockServer, err := NewMockOIDCServer("test-client", "test-secret")
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer mockServer.Close()

	ctx := context.Background()

	t.Run("FetchAndCacheKeys", func(t *testing.T) {
		cache := NewJWKSCache(mockServer.Server.URL+"/jwks", nil, 15*time.Minute)

		// Fetch the key
		key, err := cache.GetKey(ctx, mockServer.GetKeyID())
		if err != nil {
			t.Fatalf("GetKey failed: %v", err)
		}

		if key == nil {
			t.Error("Key should not be nil")
		}

		// Verify it matches the server's public key
		if key.N.Cmp(mockServer.GetPublicKey().N) != 0 {
			t.Error("Key N component doesn't match")
		}
		if key.E != mockServer.GetPublicKey().E {
			t.Error("Key E component doesn't match")
		}
	})

	t.Run("KeyNotFound", func(t *testing.T) {
		cache := NewJWKSCache(mockServer.Server.URL+"/jwks", nil, 15*time.Minute)

		_, err := cache.GetKey(ctx, "non-existent-key")
		if err == nil {
			t.Error("Expected error for non-existent key")
		}
	})
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func extractQueryParam(url, param string) string {
	// Simple query parameter extraction
	parts := splitString(url, "?")
	if len(parts) < 2 {
		return ""
	}

	query := parts[1]
	params := splitString(query, "&")

	for _, p := range params {
		kv := splitString(p, "=")
		if len(kv) == 2 && kv[0] == param {
			return kv[1]
		}
	}

	return ""
}

func splitString(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}
