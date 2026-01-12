// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality using Zitadel OIDC.
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
//
// This file contains comprehensive tests for ZitadelRelyingParty.
package auth

import (
	"testing"
	"time"
)

// TestZitadelRPConfig_Validate tests configuration validation.
func TestZitadelRPConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    *ZitadelRPConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid_config_all_required_fields",
			config: &ZitadelRPConfig{
				IssuerURL:   "https://auth.example.com",
				ClientID:    "my-client",
				RedirectURL: "https://app.example.com/callback",
				Scopes:      []string{"openid", "profile", "email"},
			},
			wantError: false,
		},
		{
			name: "valid_config_minimal",
			config: &ZitadelRPConfig{
				IssuerURL:   "https://auth.example.com",
				ClientID:    "my-client",
				RedirectURL: "https://app.example.com/callback",
				// Scopes empty - will be set by SetDefaults
			},
			wantError: false,
		},
		{
			name: "missing_issuer_url",
			config: &ZitadelRPConfig{
				ClientID:    "my-client",
				RedirectURL: "https://app.example.com/callback",
			},
			wantError: true,
			errorMsg:  "issuer_url is required",
		},
		{
			name: "missing_client_id",
			config: &ZitadelRPConfig{
				IssuerURL:   "https://auth.example.com",
				RedirectURL: "https://app.example.com/callback",
			},
			wantError: true,
			errorMsg:  "client_id is required",
		},
		{
			name: "missing_redirect_url",
			config: &ZitadelRPConfig{
				IssuerURL: "https://auth.example.com",
				ClientID:  "my-client",
			},
			wantError: true,
			errorMsg:  "redirect_url is required",
		},
		{
			name: "scopes_without_openid",
			config: &ZitadelRPConfig{
				IssuerURL:   "https://auth.example.com",
				ClientID:    "my-client",
				RedirectURL: "https://app.example.com/callback",
				Scopes:      []string{"profile", "email"}, // missing openid
			},
			wantError: true,
			errorMsg:  "scopes must include 'openid'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestZitadelRPConfig_SetDefaults tests default value application.
func TestZitadelRPConfig_SetDefaults(t *testing.T) {
	t.Run("sets_default_scopes", func(t *testing.T) {
		config := &ZitadelRPConfig{
			IssuerURL:   "https://auth.example.com",
			ClientID:    "my-client",
			RedirectURL: "https://app.example.com/callback",
		}

		config.SetDefaults()

		if len(config.Scopes) != 3 {
			t.Errorf("expected 3 scopes, got %d", len(config.Scopes))
		}
		expectedScopes := []string{"openid", "profile", "email"}
		for i, scope := range expectedScopes {
			if config.Scopes[i] != scope {
				t.Errorf("expected scope %q at index %d, got %q", scope, i, config.Scopes[i])
			}
		}
	})

	t.Run("sets_default_http_client", func(t *testing.T) {
		config := &ZitadelRPConfig{
			IssuerURL:   "https://auth.example.com",
			ClientID:    "my-client",
			RedirectURL: "https://app.example.com/callback",
		}

		config.SetDefaults()

		if config.HTTPClient == nil {
			t.Error("expected HTTPClient to be set")
		}
		if config.HTTPClient.Timeout != 30*time.Second {
			t.Errorf("expected 30s timeout, got %v", config.HTTPClient.Timeout)
		}
	})

	t.Run("sets_default_claims_mapping", func(t *testing.T) {
		config := &ZitadelRPConfig{
			IssuerURL:   "https://auth.example.com",
			ClientID:    "my-client",
			RedirectURL: "https://app.example.com/callback",
		}

		config.SetDefaults()

		if config.ClaimsMapping.RolesClaim != "roles" {
			t.Errorf("expected RolesClaim 'roles', got %q", config.ClaimsMapping.RolesClaim)
		}
		if config.ClaimsMapping.GroupsClaim != "groups" {
			t.Errorf("expected GroupsClaim 'groups', got %q", config.ClaimsMapping.GroupsClaim)
		}
		expectedUsernameClaims := []string{"preferred_username", "name", "email"}
		if len(config.ClaimsMapping.UsernameClaims) != len(expectedUsernameClaims) {
			t.Errorf("expected %d username claims, got %d", len(expectedUsernameClaims), len(config.ClaimsMapping.UsernameClaims))
		}
	})

	t.Run("sets_default_roles", func(t *testing.T) {
		config := &ZitadelRPConfig{
			IssuerURL:   "https://auth.example.com",
			ClientID:    "my-client",
			RedirectURL: "https://app.example.com/callback",
		}

		config.SetDefaults()

		if len(config.DefaultRoles) != 1 || config.DefaultRoles[0] != "viewer" {
			t.Errorf("expected default roles [viewer], got %v", config.DefaultRoles)
		}
	})

	t.Run("preserves_existing_values", func(t *testing.T) {
		config := &ZitadelRPConfig{
			IssuerURL:    "https://auth.example.com",
			ClientID:     "my-client",
			RedirectURL:  "https://app.example.com/callback",
			Scopes:       []string{"openid", "custom"},
			DefaultRoles: []string{"admin"},
			ClaimsMapping: ZitadelClaimsMappingConfig{
				RolesClaim: "custom_roles",
			},
		}

		config.SetDefaults()

		// Scopes should be preserved
		if len(config.Scopes) != 2 || config.Scopes[1] != "custom" {
			t.Errorf("scopes should be preserved, got %v", config.Scopes)
		}
		// DefaultRoles should be preserved
		if len(config.DefaultRoles) != 1 || config.DefaultRoles[0] != "admin" {
			t.Errorf("default roles should be preserved, got %v", config.DefaultRoles)
		}
		// RolesClaim should be preserved
		if config.ClaimsMapping.RolesClaim != "custom_roles" {
			t.Errorf("roles claim should be preserved, got %q", config.ClaimsMapping.RolesClaim)
		}
	})

	t.Run("idempotent_operation", func(t *testing.T) {
		config := &ZitadelRPConfig{
			IssuerURL:   "https://auth.example.com",
			ClientID:    "my-client",
			RedirectURL: "https://app.example.com/callback",
		}

		config.SetDefaults()
		firstClient := config.HTTPClient
		config.SetDefaults()

		// Second call should not change the client
		if config.HTTPClient != firstClient {
			t.Error("SetDefaults should be idempotent")
		}
	})
}

// TestZitadelRPConfig_Clone tests deep copy functionality.
func TestZitadelRPConfig_Clone(t *testing.T) {
	original := &ZitadelRPConfig{
		IssuerURL:             "https://auth.example.com",
		ClientID:              "my-client",
		ClientSecret:          "secret",
		RedirectURL:           "https://app.example.com/callback",
		Scopes:                []string{"openid", "profile"},
		PKCEEnabled:           true,
		DefaultRoles:          []string{"viewer", "user"},
		PostLogoutRedirectURI: "https://app.example.com",
		ClaimsMapping: ZitadelClaimsMappingConfig{
			RolesClaim:     "custom_roles",
			GroupsClaim:    "custom_groups",
			UsernameClaims: []string{"email", "name"},
		},
	}

	clone := original.Clone()

	// Verify values are copied
	if clone.IssuerURL != original.IssuerURL {
		t.Errorf("IssuerURL mismatch: got %q, want %q", clone.IssuerURL, original.IssuerURL)
	}
	if clone.ClientID != original.ClientID {
		t.Errorf("ClientID mismatch")
	}
	if clone.ClientSecret != original.ClientSecret {
		t.Errorf("ClientSecret mismatch")
	}
	if clone.PKCEEnabled != original.PKCEEnabled {
		t.Errorf("PKCEEnabled mismatch")
	}

	// Verify slices are deep copied
	if &clone.Scopes == &original.Scopes {
		t.Error("Scopes should be deep copied, not same reference")
	}
	if len(clone.Scopes) != len(original.Scopes) {
		t.Errorf("Scopes length mismatch: got %d, want %d", len(clone.Scopes), len(original.Scopes))
	}

	// Modify clone, verify original is unchanged
	clone.Scopes = append(clone.Scopes, "modified")
	if len(original.Scopes) != 2 {
		t.Error("modifying clone affected original")
	}

	// Verify claims mapping is deep copied
	clone.ClaimsMapping.RolesClaim = "modified_roles"
	if original.ClaimsMapping.RolesClaim != "custom_roles" {
		t.Error("modifying clone claims mapping affected original")
	}
}

// TestZitadelClaimsMappingConfig tests claims mapping configuration.
func TestZitadelClaimsMappingConfig(t *testing.T) {
	t.Run("empty_config", func(t *testing.T) {
		config := ZitadelClaimsMappingConfig{}

		if config.RolesClaim != "" {
			t.Errorf("expected empty RolesClaim, got %q", config.RolesClaim)
		}
		if config.GroupsClaim != "" {
			t.Errorf("expected empty GroupsClaim, got %q", config.GroupsClaim)
		}
		if len(config.UsernameClaims) != 0 {
			t.Errorf("expected empty UsernameClaims, got %v", config.UsernameClaims)
		}
	})

	t.Run("fully_configured", func(t *testing.T) {
		config := ZitadelClaimsMappingConfig{
			RolesClaim:     "realm_access.roles",
			GroupsClaim:    "org_groups",
			UsernameClaims: []string{"upn", "email"},
		}

		if config.RolesClaim != "realm_access.roles" {
			t.Errorf("RolesClaim mismatch")
		}
		if config.GroupsClaim != "org_groups" {
			t.Errorf("GroupsClaim mismatch")
		}
		if len(config.UsernameClaims) != 2 {
			t.Errorf("expected 2 username claims")
		}
	})
}

// TestNewZitadelRelyingParty_NilConfig tests nil config handling.
func TestNewZitadelRelyingParty_NilConfig(t *testing.T) {
	_, err := NewZitadelRelyingParty(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
	if err.Error() != "config is required" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestNewZitadelRelyingParty_InvalidConfig tests invalid config handling.
func TestNewZitadelRelyingParty_InvalidConfig(t *testing.T) {
	config := &ZitadelRPConfig{
		// Missing required fields
	}

	_, err := NewZitadelRelyingParty(context.Background(), config)
	if err == nil {
		t.Error("expected error for invalid config")
	}
}
