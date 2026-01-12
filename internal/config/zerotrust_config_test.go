// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

// =====================================================
// Zero Trust Authentication & Authorization Config Tests
// ADR-0015: Zero Trust Authentication & Authorization
// =====================================================

func TestLoad_ZeroTrustAuthModes(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid OIDC auth mode",
			envVars: map[string]string{
				"TAUTULLI_URL":      "http://localhost:8181",
				"TAUTULLI_API_KEY":  "test_api_key_12345678",
				"AUTH_MODE":         "oidc",
				"OIDC_ISSUER_URL":   "https://auth.example.com",
				"OIDC_CLIENT_ID":    "my-client-id",
				"OIDC_REDIRECT_URL": "http://localhost:3857/auth/callback",
			},
			wantErr: false,
		},
		{
			name: "valid Plex auth mode",
			envVars: map[string]string{
				"TAUTULLI_URL":           "http://localhost:8181",
				"TAUTULLI_API_KEY":       "test_api_key_12345678",
				"AUTH_MODE":              "plex",
				"PLEX_AUTH_CLIENT_ID":    "plex-client-id-12345",
				"PLEX_AUTH_REDIRECT_URI": "http://localhost:3857/auth/plex/callback",
			},
			wantErr: false,
		},
		{
			name: "valid multi auth mode with JWT",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "multi",
				"JWT_SECRET":       "this_is_a_very_long_secret_key_with_more_than_32_characters",
				"ADMIN_USERNAME":   "admin",
				"ADMIN_PASSWORD":   "password123",
			},
			wantErr: false,
		},
		{
			name: "OIDC auth mode requires issuer URL",
			envVars: map[string]string{
				"TAUTULLI_URL":      "http://localhost:8181",
				"TAUTULLI_API_KEY":  "test_api_key_12345678",
				"AUTH_MODE":         "oidc",
				"OIDC_CLIENT_ID":    "my-client-id",
				"OIDC_REDIRECT_URL": "http://localhost:3857/auth/callback",
			},
			wantErr: true,
			errMsg:  "OIDC_ISSUER_URL is required when AUTH_MODE is oidc",
		},
		{
			name: "OIDC auth mode requires client ID",
			envVars: map[string]string{
				"TAUTULLI_URL":      "http://localhost:8181",
				"TAUTULLI_API_KEY":  "test_api_key_12345678",
				"AUTH_MODE":         "oidc",
				"OIDC_ISSUER_URL":   "https://auth.example.com",
				"OIDC_REDIRECT_URL": "http://localhost:3857/auth/callback",
			},
			wantErr: true,
			errMsg:  "OIDC_CLIENT_ID is required when AUTH_MODE is oidc",
		},
		{
			name: "OIDC auth mode requires redirect URL",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "oidc",
				"OIDC_ISSUER_URL":  "https://auth.example.com",
				"OIDC_CLIENT_ID":   "my-client-id",
			},
			wantErr: true,
			errMsg:  "OIDC_REDIRECT_URL is required when AUTH_MODE is oidc",
		},
		{
			name: "OIDC issuer URL must be valid",
			envVars: map[string]string{
				"TAUTULLI_URL":      "http://localhost:8181",
				"TAUTULLI_API_KEY":  "test_api_key_12345678",
				"AUTH_MODE":         "oidc",
				"OIDC_ISSUER_URL":   "not-a-valid-url",
				"OIDC_CLIENT_ID":    "my-client-id",
				"OIDC_REDIRECT_URL": "http://localhost:3857/auth/callback",
			},
			wantErr: true,
			errMsg:  "OIDC_ISSUER_URL is invalid",
		},
		{
			name: "Plex auth mode requires client ID",
			envVars: map[string]string{
				"TAUTULLI_URL":           "http://localhost:8181",
				"TAUTULLI_API_KEY":       "test_api_key_12345678",
				"AUTH_MODE":              "plex",
				"PLEX_AUTH_REDIRECT_URI": "http://localhost:3857/auth/plex/callback",
			},
			wantErr: true,
			errMsg:  "PLEX_AUTH_CLIENT_ID is required when AUTH_MODE is plex",
		},
		{
			name: "Plex auth mode requires redirect URI",
			envVars: map[string]string{
				"TAUTULLI_URL":        "http://localhost:8181",
				"TAUTULLI_API_KEY":    "test_api_key_12345678",
				"AUTH_MODE":           "plex",
				"PLEX_AUTH_CLIENT_ID": "plex-client-id-12345",
			},
			wantErr: true,
			errMsg:  "PLEX_AUTH_REDIRECT_URI is required when AUTH_MODE is plex",
		},
		{
			name: "multi auth mode requires at least one authenticator",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "multi",
			},
			wantErr: true,
			errMsg:  "multi auth mode requires at least one authenticator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t, tt.envVars)
			defer cleanup()

			cfg, err := Load()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Load() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Load() unexpected error = %v", err)
				}
				if cfg == nil {
					t.Error("Load() returned nil config")
				}
			}
		})
	}
}

func TestLoad_OIDCConfigValues(t *testing.T) {
	os.Clearenv()

	envVars := map[string]string{
		"TAUTULLI_URL":         "http://localhost:8181",
		"TAUTULLI_API_KEY":     "test_api_key_12345678",
		"AUTH_MODE":            "oidc",
		"OIDC_ISSUER_URL":      "https://auth.example.com",
		"OIDC_CLIENT_ID":       "my-client-id",
		"OIDC_CLIENT_SECRET":   "my-client-secret",
		"OIDC_REDIRECT_URL":    "http://localhost:3857/auth/callback",
		"OIDC_SCOPES":          "openid,profile,email,groups",
		"OIDC_PKCE_ENABLED":    "true",
		"OIDC_JWKS_CACHE_TTL":  "30m",
		"OIDC_SESSION_MAX_AGE": "12h",
		"OIDC_SESSION_SECRET":  "32-byte-secret-for-session-test",
		"OIDC_COOKIE_NAME":     "my_session",
		"OIDC_COOKIE_SECURE":   "true",
		"OIDC_ROLES_CLAIM":     "realm_access.roles",
		"OIDC_DEFAULT_ROLES":   "viewer",
		"OIDC_USERNAME_CLAIMS": "preferred_username,name,email",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify OIDC configuration
	assertStringEqual(t, cfg.Security.OIDC.IssuerURL, "https://auth.example.com", "Security.OIDC.IssuerURL")
	assertStringEqual(t, cfg.Security.OIDC.ClientID, "my-client-id", "Security.OIDC.ClientID")
	assertStringEqual(t, cfg.Security.OIDC.ClientSecret, "my-client-secret", "Security.OIDC.ClientSecret")
	assertStringEqual(t, cfg.Security.OIDC.RedirectURL, "http://localhost:3857/auth/callback", "Security.OIDC.RedirectURL")
	assertBoolEqual(t, cfg.Security.OIDC.PKCEEnabled, true, "Security.OIDC.PKCEEnabled")
	assertDurationEqual(t, cfg.Security.OIDC.JWKSCacheTTL, 30*time.Minute, "Security.OIDC.JWKSCacheTTL")
	assertDurationEqual(t, cfg.Security.OIDC.SessionMaxAge, 12*time.Hour, "Security.OIDC.SessionMaxAge")
	assertStringEqual(t, cfg.Security.OIDC.CookieName, "my_session", "Security.OIDC.CookieName")
	assertBoolEqual(t, cfg.Security.OIDC.CookieSecure, true, "Security.OIDC.CookieSecure")
	assertStringEqual(t, cfg.Security.OIDC.RolesClaim, "realm_access.roles", "Security.OIDC.RolesClaim")

	// Verify scopes
	expectedScopes := []string{"openid", "profile", "email", "groups"}
	if len(cfg.Security.OIDC.Scopes) != len(expectedScopes) {
		t.Errorf("Security.OIDC.Scopes length = %v, want %v", len(cfg.Security.OIDC.Scopes), len(expectedScopes))
	}

	// Verify default roles
	if len(cfg.Security.OIDC.DefaultRoles) != 1 || cfg.Security.OIDC.DefaultRoles[0] != "viewer" {
		t.Errorf("Security.OIDC.DefaultRoles = %v, want [viewer]", cfg.Security.OIDC.DefaultRoles)
	}
}

func TestLoad_OIDCDefaultValues(t *testing.T) {
	os.Clearenv()

	envVars := map[string]string{
		"TAUTULLI_URL":      "http://localhost:8181",
		"TAUTULLI_API_KEY":  "test_api_key_12345678",
		"AUTH_MODE":         "oidc",
		"OIDC_ISSUER_URL":   "https://auth.example.com",
		"OIDC_CLIENT_ID":    "my-client-id",
		"OIDC_REDIRECT_URL": "http://localhost:3857/auth/callback",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify OIDC default values
	assertBoolEqual(t, cfg.Security.OIDC.PKCEEnabled, true, "Security.OIDC.PKCEEnabled")
	assertDurationEqual(t, cfg.Security.OIDC.JWKSCacheTTL, 1*time.Hour, "Security.OIDC.JWKSCacheTTL")
	assertDurationEqual(t, cfg.Security.OIDC.SessionMaxAge, 24*time.Hour, "Security.OIDC.SessionMaxAge")
	assertStringEqual(t, cfg.Security.OIDC.CookieName, "tautulli_session", "Security.OIDC.CookieName")
	assertBoolEqual(t, cfg.Security.OIDC.CookieSecure, true, "Security.OIDC.CookieSecure")

	// Verify default scopes include openid
	hasOpenID := false
	for _, scope := range cfg.Security.OIDC.Scopes {
		if scope == "openid" {
			hasOpenID = true
			break
		}
	}
	if !hasOpenID {
		t.Errorf("Security.OIDC.Scopes should include 'openid' by default, got %v", cfg.Security.OIDC.Scopes)
	}
}

func TestLoad_PlexAuthConfigValues(t *testing.T) {
	os.Clearenv()

	envVars := map[string]string{
		"TAUTULLI_URL":             "http://localhost:8181",
		"TAUTULLI_API_KEY":         "test_api_key_12345678",
		"AUTH_MODE":                "plex",
		"PLEX_AUTH_CLIENT_ID":      "plex-client-id-12345",
		"PLEX_AUTH_CLIENT_SECRET":  "plex-client-secret",
		"PLEX_AUTH_REDIRECT_URI":   "http://localhost:3857/auth/plex/callback",
		"PLEX_AUTH_DEFAULT_ROLES":  "viewer,plex_user",
		"PLEX_AUTH_PLEX_PASS_ROLE": "premium",
		"PLEX_AUTH_TIMEOUT":        "60s",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify Plex Auth configuration
	assertStringEqual(t, cfg.Security.PlexAuth.ClientID, "plex-client-id-12345", "Security.PlexAuth.ClientID")
	assertStringEqual(t, cfg.Security.PlexAuth.ClientSecret, "plex-client-secret", "Security.PlexAuth.ClientSecret")
	assertStringEqual(t, cfg.Security.PlexAuth.RedirectURI, "http://localhost:3857/auth/plex/callback", "Security.PlexAuth.RedirectURI")
	assertStringEqual(t, cfg.Security.PlexAuth.PlexPassRole, "premium", "Security.PlexAuth.PlexPassRole")
	assertDurationEqual(t, cfg.Security.PlexAuth.Timeout, 60*time.Second, "Security.PlexAuth.Timeout")

	// Verify default roles
	if len(cfg.Security.PlexAuth.DefaultRoles) != 2 {
		t.Errorf("Security.PlexAuth.DefaultRoles length = %v, want 2", len(cfg.Security.PlexAuth.DefaultRoles))
	}
}

func TestLoad_PlexAuthDefaultValues(t *testing.T) {
	os.Clearenv()

	envVars := map[string]string{
		"TAUTULLI_URL":           "http://localhost:8181",
		"TAUTULLI_API_KEY":       "test_api_key_12345678",
		"AUTH_MODE":              "plex",
		"PLEX_AUTH_CLIENT_ID":    "plex-client-id-12345",
		"PLEX_AUTH_REDIRECT_URI": "http://localhost:3857/auth/plex/callback",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify Plex Auth default values
	assertDurationEqual(t, cfg.Security.PlexAuth.Timeout, 30*time.Second, "Security.PlexAuth.Timeout")

	// Default roles should include viewer
	hasViewer := false
	for _, role := range cfg.Security.PlexAuth.DefaultRoles {
		if role == "viewer" {
			hasViewer = true
			break
		}
	}
	if !hasViewer {
		t.Errorf("Security.PlexAuth.DefaultRoles should include 'viewer' by default, got %v", cfg.Security.PlexAuth.DefaultRoles)
	}
}

func TestLoad_CasbinConfigValues(t *testing.T) {
	os.Clearenv()

	envVars := map[string]string{
		"TAUTULLI_URL":           "http://localhost:8181",
		"TAUTULLI_API_KEY":       "test_api_key_12345678",
		"AUTH_MODE":              "none",
		"CASBIN_MODEL_PATH":      "/custom/model.conf",
		"CASBIN_POLICY_PATH":     "/custom/policy.csv",
		"CASBIN_DEFAULT_ROLE":    "guest",
		"CASBIN_AUTO_RELOAD":     "true",
		"CASBIN_RELOAD_INTERVAL": "1m",
		"CASBIN_CACHE_ENABLED":   "true",
		"CASBIN_CACHE_TTL":       "10m",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify Casbin configuration
	assertStringEqual(t, cfg.Security.Casbin.ModelPath, "/custom/model.conf", "Security.Casbin.ModelPath")
	assertStringEqual(t, cfg.Security.Casbin.PolicyPath, "/custom/policy.csv", "Security.Casbin.PolicyPath")
	assertStringEqual(t, cfg.Security.Casbin.DefaultRole, "guest", "Security.Casbin.DefaultRole")
	assertBoolEqual(t, cfg.Security.Casbin.AutoReload, true, "Security.Casbin.AutoReload")
	assertDurationEqual(t, cfg.Security.Casbin.ReloadInterval, 1*time.Minute, "Security.Casbin.ReloadInterval")
	assertBoolEqual(t, cfg.Security.Casbin.CacheEnabled, true, "Security.Casbin.CacheEnabled")
	assertDurationEqual(t, cfg.Security.Casbin.CacheTTL, 10*time.Minute, "Security.Casbin.CacheTTL")
}

func TestLoad_CasbinDefaultValues(t *testing.T) {
	os.Clearenv()

	envVars := map[string]string{
		"TAUTULLI_URL":     "http://localhost:8181",
		"TAUTULLI_API_KEY": "test_api_key_12345678",
		"AUTH_MODE":        "none",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify Casbin default values
	assertStringEqual(t, cfg.Security.Casbin.DefaultRole, "viewer", "Security.Casbin.DefaultRole")
	assertBoolEqual(t, cfg.Security.Casbin.AutoReload, true, "Security.Casbin.AutoReload")
	assertDurationEqual(t, cfg.Security.Casbin.ReloadInterval, 30*time.Second, "Security.Casbin.ReloadInterval")
	assertBoolEqual(t, cfg.Security.Casbin.CacheEnabled, true, "Security.Casbin.CacheEnabled")
	assertDurationEqual(t, cfg.Security.Casbin.CacheTTL, 5*time.Minute, "Security.Casbin.CacheTTL")
}

func TestLoad_MultiAuthModeConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name: "multi mode with JWT and Basic",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "multi",
				"JWT_SECRET":       "this_is_a_very_long_secret_key_with_more_than_32_characters",
				"ADMIN_USERNAME":   "admin",
				"ADMIN_PASSWORD":   "password123",
			},
			wantErr: false,
		},
		{
			name: "multi mode with OIDC",
			envVars: map[string]string{
				"TAUTULLI_URL":      "http://localhost:8181",
				"TAUTULLI_API_KEY":  "test_api_key_12345678",
				"AUTH_MODE":         "multi",
				"OIDC_ISSUER_URL":   "https://auth.example.com",
				"OIDC_CLIENT_ID":    "my-client-id",
				"OIDC_REDIRECT_URL": "http://localhost:3857/auth/callback",
			},
			wantErr: false,
		},
		{
			name: "multi mode with Plex",
			envVars: map[string]string{
				"TAUTULLI_URL":           "http://localhost:8181",
				"TAUTULLI_API_KEY":       "test_api_key_12345678",
				"AUTH_MODE":              "multi",
				"PLEX_AUTH_CLIENT_ID":    "plex-client-id-12345",
				"PLEX_AUTH_REDIRECT_URI": "http://localhost:3857/auth/plex/callback",
			},
			wantErr: false,
		},
		{
			name: "multi mode with all authenticators",
			envVars: map[string]string{
				"TAUTULLI_URL":           "http://localhost:8181",
				"TAUTULLI_API_KEY":       "test_api_key_12345678",
				"AUTH_MODE":              "multi",
				"JWT_SECRET":             "this_is_a_very_long_secret_key_with_more_than_32_characters",
				"ADMIN_USERNAME":         "admin",
				"ADMIN_PASSWORD":         "password123",
				"OIDC_ISSUER_URL":        "https://auth.example.com",
				"OIDC_CLIENT_ID":         "my-client-id",
				"OIDC_REDIRECT_URL":      "http://localhost:3857/auth/callback",
				"PLEX_AUTH_CLIENT_ID":    "plex-client-id-12345",
				"PLEX_AUTH_REDIRECT_URI": "http://localhost:3857/auth/plex/callback",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t, tt.envVars)
			defer cleanup()

			cfg, err := Load()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Load() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Load() unexpected error = %v", err)
				}
				if cfg == nil {
					t.Error("Load() returned nil config")
				}
			}
		})
	}
}

func TestValidateOIDCURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid HTTPS URL",
			url:     "https://auth.example.com",
			wantErr: false,
		},
		{
			name:    "valid HTTPS with path",
			url:     "https://auth.example.com/realms/myrealm",
			wantErr: false,
		},
		{
			name:    "valid HTTP localhost (for development)",
			url:     "http://localhost:8080",
			wantErr: false,
		},
		{
			name:    "invalid scheme",
			url:     "ftp://auth.example.com",
			wantErr: true,
			errMsg:  "scheme must be http or https",
		},
		{
			name:    "missing scheme",
			url:     "auth.example.com",
			wantErr: true,
			errMsg:  "scheme must be http or https",
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
			errMsg:  "scheme must be http or https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOIDCIssuerURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateOIDCIssuerURL(%q) expected error containing %q, got nil", tt.url, tt.errMsg)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateOIDCIssuerURL(%q) error = %v, want error containing %q", tt.url, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateOIDCIssuerURL(%q) unexpected error = %v", tt.url, err)
				}
			}
		})
	}
}
