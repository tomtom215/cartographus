// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"testing"
	"time"
)

// TestOIDCConfig_Validate tests OIDC configuration validation
func TestOIDCConfig_Validate(t *testing.T) {
	validConfig := func() *OIDCConfig {
		return &OIDCConfig{
			IssuerURL:   "https://auth.example.com",
			ClientID:    "test-client",
			RedirectURL: "http://localhost:8080/auth/callback",
			Scopes:      []string{"openid", "profile", "email"},
		}
	}

	tests := []struct {
		name    string
		modify  func(*OIDCConfig)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			modify:  func(c *OIDCConfig) {},
			wantErr: false,
		},
		{
			name: "missing issuer URL",
			modify: func(c *OIDCConfig) {
				c.IssuerURL = ""
			},
			wantErr: true,
			errMsg:  "issuer URL is required",
		},
		{
			name: "invalid issuer URL scheme",
			modify: func(c *OIDCConfig) {
				c.IssuerURL = "ftp://auth.example.com"
			},
			wantErr: true,
			errMsg:  "must use http or https",
		},
		{
			name: "missing client ID",
			modify: func(c *OIDCConfig) {
				c.ClientID = ""
			},
			wantErr: true,
			errMsg:  "client ID is required",
		},
		{
			name: "missing redirect URL",
			modify: func(c *OIDCConfig) {
				c.RedirectURL = ""
			},
			wantErr: true,
			errMsg:  "redirect URL is required",
		},
		{
			name: "invalid redirect URL",
			modify: func(c *OIDCConfig) {
				c.RedirectURL = "://invalid"
			},
			wantErr: true,
			errMsg:  "invalid redirect URL",
		},
		{
			name: "missing openid scope",
			modify: func(c *OIDCConfig) {
				c.Scopes = []string{"profile", "email"}
			},
			wantErr: true,
			errMsg:  "'openid' scope is required",
		},
		{
			name: "invalid session secret length",
			modify: func(c *OIDCConfig) {
				c.Session.Secret = []byte("too-short")
			},
			wantErr: true,
			errMsg:  "session secret must be exactly 32 bytes",
		},
		{
			name: "valid session secret length",
			modify: func(c *OIDCConfig) {
				c.Session.Secret = make([]byte, 32)
			},
			wantErr: false,
		},
		{
			name: "http issuer URL (allowed for development)",
			modify: func(c *OIDCConfig) {
				c.IssuerURL = "http://localhost:8081"
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := validConfig()
			tt.modify(config)

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !containsSubstring(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

// TestDefaultOIDCConfig verifies default configuration values
func TestDefaultOIDCConfig(t *testing.T) {
	config := DefaultOIDCConfig()

	// Check default scopes
	if len(config.Scopes) != 3 {
		t.Errorf("Default scopes count = %d, want 3", len(config.Scopes))
	}

	hasOpenID := false
	for _, scope := range config.Scopes {
		if scope == "openid" {
			hasOpenID = true
		}
	}
	if !hasOpenID {
		t.Error("Default scopes should include 'openid'")
	}

	// Check PKCE enabled by default
	if !config.PKCEEnabled {
		t.Error("PKCE should be enabled by default")
	}

	// Check JWKS cache enabled by default
	if !config.JWKSCache.Enabled {
		t.Error("JWKS cache should be enabled by default")
	}

	// Check session defaults
	if config.Session.MaxAge != 24*time.Hour {
		t.Errorf("Session.MaxAge = %v, want 24h", config.Session.MaxAge)
	}

	if config.Session.CookieName != "tautulli_session" {
		t.Errorf("Session.CookieName = %q, want %q", config.Session.CookieName, "tautulli_session")
	}

	if !config.Session.CookieSecure {
		t.Error("Session.CookieSecure should be true by default")
	}

	if !config.Session.CookieHTTPOnly {
		t.Error("Session.CookieHTTPOnly should be true by default")
	}
}

// TestSessionSecret_Generation tests secure session secret generation
func TestSessionSecret_Generation(t *testing.T) {
	secret1, err := GenerateSessionSecret()
	if err != nil {
		t.Fatalf("GenerateSessionSecret() error = %v", err)
	}

	if len(secret1) != 32 {
		t.Errorf("Session secret length = %d, want 32", len(secret1))
	}

	// Generate another secret - should be different
	secret2, err := GenerateSessionSecret()
	if err != nil {
		t.Fatalf("GenerateSessionSecret() error = %v", err)
	}

	if string(secret1) == string(secret2) {
		t.Error("Generated secrets should be unique")
	}
}

// TestSessionSecret_Encoding tests base64 encoding/decoding of session secrets
func TestSessionSecret_Encoding(t *testing.T) {
	original, err := GenerateSessionSecret()
	if err != nil {
		t.Fatalf("GenerateSessionSecret() error = %v", err)
	}

	// Encode
	encoded := EncodeSessionSecret(original)
	if encoded == "" {
		t.Error("EncodeSessionSecret() returned empty string")
	}

	// Decode
	decoded, err := DecodeSessionSecret(encoded)
	if err != nil {
		t.Fatalf("DecodeSessionSecret() error = %v", err)
	}

	if string(decoded) != string(original) {
		t.Error("Decoded secret does not match original")
	}
}

// TestClaimsMapping_Defaults tests default claims mapping configuration
func TestClaimsMapping_Defaults(t *testing.T) {
	config := DefaultOIDCConfig()

	if config.ClaimsMapping.SubjectClaim != "sub" {
		t.Errorf("SubjectClaim = %q, want %q", config.ClaimsMapping.SubjectClaim, "sub")
	}

	expectedUsernameClaims := []string{"preferred_username", "name", "email"}
	if len(config.ClaimsMapping.UsernameClaims) != len(expectedUsernameClaims) {
		t.Errorf("UsernameClaims length = %d, want %d",
			len(config.ClaimsMapping.UsernameClaims), len(expectedUsernameClaims))
	}

	if config.ClaimsMapping.EmailClaim != "email" {
		t.Errorf("EmailClaim = %q, want %q", config.ClaimsMapping.EmailClaim, "email")
	}

	if config.ClaimsMapping.GroupsClaim != "groups" {
		t.Errorf("GroupsClaim = %q, want %q", config.ClaimsMapping.GroupsClaim, "groups")
	}
}

// containsSubstring helper function
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
