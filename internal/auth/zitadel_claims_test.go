// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality using Zitadel OIDC.
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
//
// This file contains comprehensive tests for Zitadel claims mapping.
package auth

import (
	"testing"

	"github.com/zitadel/oidc/v3/pkg/oidc"
)

// TestMapZitadelClaimsToAuthSubject tests ID token claims mapping.
func TestMapZitadelClaimsToAuthSubject(t *testing.T) {
	t.Run("nil_claims_returns_nil", func(t *testing.T) {
		result := MapZitadelClaimsToAuthSubject(nil, nil, nil)
		if result != nil {
			t.Error("expected nil for nil claims")
		}
	})

	t.Run("basic_claims_mapping", func(t *testing.T) {
		claims := &oidc.IDTokenClaims{
			TokenClaims: oidc.TokenClaims{
				Subject: "user-123",
				Issuer:  "https://auth.example.com",
			},
			UserInfoEmail: oidc.UserInfoEmail{
				Email:         "user@example.com",
				EmailVerified: true,
			},
			UserInfoProfile: oidc.UserInfoProfile{
				PreferredUsername: "testuser",
			},
		}
		config := &ZitadelClaimsMappingConfig{
			RolesClaim:     "roles",
			GroupsClaim:    "groups",
			UsernameClaims: []string{"preferred_username", "email"},
		}

		result := MapZitadelClaimsToAuthSubject(claims, config, nil)

		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.ID != "user-123" {
			t.Errorf("ID mismatch: got %q, want %q", result.ID, "user-123")
		}
		if result.Email != "user@example.com" {
			t.Errorf("Email mismatch: got %q, want %q", result.Email, "user@example.com")
		}
		if result.Issuer != "https://auth.example.com" {
			t.Errorf("Issuer mismatch: got %q, want %q", result.Issuer, "https://auth.example.com")
		}
		if result.AuthMethod != AuthModeOIDC {
			t.Errorf("AuthMethod mismatch: got %q, want %q", result.AuthMethod, AuthModeOIDC)
		}
		if result.Provider != "oidc" {
			t.Errorf("Provider mismatch: got %q, want %q", result.Provider, "oidc")
		}
		if result.Username != "testuser" {
			t.Errorf("Username mismatch: got %q, want %q", result.Username, "testuser")
		}
		if !result.EmailVerified {
			t.Error("EmailVerified should be true")
		}
	})

	t.Run("username_fallback_chain", func(t *testing.T) {
		testCases := []struct {
			name             string
			preferredUser    string
			userName         string
			userEmail        string
			expectedUsername string
		}{
			{
				name:             "uses_preferred_username",
				preferredUser:    "preferred",
				userName:         "name",
				userEmail:        "email@test.com",
				expectedUsername: "preferred",
			},
			{
				name:             "falls_back_to_name",
				preferredUser:    "",
				userName:         "name",
				userEmail:        "email@test.com",
				expectedUsername: "name",
			},
			{
				name:             "falls_back_to_email",
				preferredUser:    "",
				userName:         "",
				userEmail:        "email@test.com",
				expectedUsername: "email@test.com",
			},
			{
				name:             "falls_back_to_subject",
				preferredUser:    "",
				userName:         "",
				userEmail:        "",
				expectedUsername: "sub-id",
			},
		}

		config := &ZitadelClaimsMappingConfig{
			UsernameClaims: []string{"preferred_username", "name", "email"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				claims := &oidc.IDTokenClaims{
					TokenClaims: oidc.TokenClaims{
						Subject: "sub-id",
					},
					UserInfoProfile: oidc.UserInfoProfile{
						PreferredUsername: tc.preferredUser,
						Name:              tc.userName,
					},
					UserInfoEmail: oidc.UserInfoEmail{
						Email: tc.userEmail,
					},
				}

				result := MapZitadelClaimsToAuthSubject(claims, config, nil)

				if result.Username != tc.expectedUsername {
					t.Errorf("expected username %q, got %q", tc.expectedUsername, result.Username)
				}
			})
		}
	})

	t.Run("default_roles_applied_when_no_roles_in_token", func(t *testing.T) {
		claims := &oidc.IDTokenClaims{
			TokenClaims: oidc.TokenClaims{
				Subject: "user-123",
			},
		}
		config := &ZitadelClaimsMappingConfig{
			RolesClaim: "roles",
		}
		defaultRoles := []string{"viewer", "basic"}

		result := MapZitadelClaimsToAuthSubject(claims, config, defaultRoles)

		if len(result.Roles) != 2 {
			t.Errorf("expected 2 roles, got %d", len(result.Roles))
		}
		if result.Roles[0] != "viewer" || result.Roles[1] != "basic" {
			t.Errorf("unexpected roles: %v", result.Roles)
		}
	})

	t.Run("roles_from_token_override_defaults", func(t *testing.T) {
		claims := &oidc.IDTokenClaims{
			TokenClaims: oidc.TokenClaims{
				Subject: "user-123",
			},
			Claims: map[string]interface{}{
				"roles": []interface{}{"admin", "editor"},
			},
		}
		config := &ZitadelClaimsMappingConfig{
			RolesClaim: "roles",
		}
		defaultRoles := []string{"viewer"} // Should be ignored

		result := MapZitadelClaimsToAuthSubject(claims, config, defaultRoles)

		if len(result.Roles) != 2 {
			t.Errorf("expected 2 roles from token, got %d", len(result.Roles))
		}
		if result.Roles[0] != "admin" || result.Roles[1] != "editor" {
			t.Errorf("unexpected roles: %v", result.Roles)
		}
	})

	t.Run("groups_extracted_from_claims", func(t *testing.T) {
		claims := &oidc.IDTokenClaims{
			TokenClaims: oidc.TokenClaims{
				Subject: "user-123",
			},
			Claims: map[string]interface{}{
				"groups": []interface{}{"engineering", "backend"},
			},
		}
		config := &ZitadelClaimsMappingConfig{
			GroupsClaim: "groups",
		}

		result := MapZitadelClaimsToAuthSubject(claims, config, nil)

		if len(result.Groups) != 2 {
			t.Errorf("expected 2 groups, got %d", len(result.Groups))
		}
		if result.Groups[0] != "engineering" || result.Groups[1] != "backend" {
			t.Errorf("unexpected groups: %v", result.Groups)
		}
	})

	t.Run("raw_claims_stored", func(t *testing.T) {
		claims := &oidc.IDTokenClaims{
			TokenClaims: oidc.TokenClaims{
				Subject: "user-123",
			},
			Claims: map[string]interface{}{
				"custom_claim": "custom_value",
			},
		}
		config := &ZitadelClaimsMappingConfig{}

		result := MapZitadelClaimsToAuthSubject(claims, config, nil)

		if result.RawClaims == nil {
			t.Fatal("expected RawClaims to be set")
		}
		if result.RawClaims["custom_claim"] != "custom_value" {
			t.Errorf("custom claim not preserved in RawClaims")
		}
	})
}

// TestMapZitadelUserInfoToAuthSubject tests userinfo claims mapping.
func TestMapZitadelUserInfoToAuthSubject(t *testing.T) {
	t.Run("nil_userinfo_returns_nil", func(t *testing.T) {
		result := MapZitadelUserInfoToAuthSubject(nil, "", nil, nil)
		if result != nil {
			t.Error("expected nil for nil userinfo")
		}
	})

	t.Run("basic_userinfo_mapping", func(t *testing.T) {
		userInfo := &oidc.UserInfo{
			Subject: "user-456",
			UserInfoEmail: oidc.UserInfoEmail{
				Email:         "test@example.com",
				EmailVerified: true,
			},
			UserInfoProfile: oidc.UserInfoProfile{
				PreferredUsername: "testuser",
				Name:              "Test User",
			},
		}
		config := &ZitadelClaimsMappingConfig{
			UsernameClaims: []string{"preferred_username"},
		}

		result := MapZitadelUserInfoToAuthSubject(userInfo, "https://issuer.example.com", config, nil)

		if result.ID != "user-456" {
			t.Errorf("ID mismatch: got %q", result.ID)
		}
		if result.Email != "test@example.com" {
			t.Errorf("Email mismatch")
		}
		if result.Issuer != "https://issuer.example.com" {
			t.Errorf("Issuer mismatch")
		}
		if result.Username != "testuser" {
			t.Errorf("Username mismatch: got %q", result.Username)
		}
	})
}

// TestExtractZitadelStringSlice tests string slice extraction from claims.
func TestExtractZitadelStringSlice(t *testing.T) {
	t.Run("nil_claims_returns_nil", func(t *testing.T) {
		result := extractZitadelStringSlice(nil, "roles")
		if result != nil {
			t.Error("expected nil for nil claims")
		}
	})

	t.Run("empty_claim_name_returns_nil", func(t *testing.T) {
		claims := map[string]interface{}{"roles": []string{"admin"}}
		result := extractZitadelStringSlice(claims, "")
		if result != nil {
			t.Error("expected nil for empty claim name")
		}
	})

	t.Run("missing_claim_returns_nil", func(t *testing.T) {
		claims := map[string]interface{}{"other": "value"}
		result := extractZitadelStringSlice(claims, "roles")
		if result != nil {
			t.Error("expected nil for missing claim")
		}
	})

	t.Run("string_slice_type", func(t *testing.T) {
		claims := map[string]interface{}{
			"roles": []string{"admin", "editor"},
		}
		result := extractZitadelStringSlice(claims, "roles")

		if len(result) != 2 {
			t.Fatalf("expected 2 roles, got %d", len(result))
		}
		if result[0] != "admin" || result[1] != "editor" {
			t.Errorf("unexpected roles: %v", result)
		}
	})

	t.Run("interface_slice_type", func(t *testing.T) {
		claims := map[string]interface{}{
			"roles": []interface{}{"admin", "editor"},
		}
		result := extractZitadelStringSlice(claims, "roles")

		if len(result) != 2 {
			t.Fatalf("expected 2 roles, got %d", len(result))
		}
	})

	t.Run("single_string_value", func(t *testing.T) {
		claims := map[string]interface{}{
			"role": "admin",
		}
		result := extractZitadelStringSlice(claims, "role")

		if len(result) != 1 {
			t.Fatalf("expected 1 role, got %d", len(result))
		}
		if result[0] != "admin" {
			t.Errorf("expected 'admin', got %q", result[0])
		}
	})

	t.Run("interface_slice_with_non_strings_filtered", func(t *testing.T) {
		claims := map[string]interface{}{
			"roles": []interface{}{"admin", 123, "editor", true},
		}
		result := extractZitadelStringSlice(claims, "roles")

		if len(result) != 2 {
			t.Errorf("expected 2 string roles, got %d: %v", len(result), result)
		}
	})

	t.Run("empty_interface_slice_returns_nil", func(t *testing.T) {
		claims := map[string]interface{}{
			"roles": []interface{}{123, true}, // no strings
		}
		result := extractZitadelStringSlice(claims, "roles")

		if result != nil {
			t.Errorf("expected nil for no valid strings, got %v", result)
		}
	})

	t.Run("wrong_type_returns_nil", func(t *testing.T) {
		claims := map[string]interface{}{
			"roles": 12345, // not a slice or string
		}
		result := extractZitadelStringSlice(claims, "roles")

		if result != nil {
			t.Errorf("expected nil for wrong type, got %v", result)
		}
	})

	t.Run("returns_copy_not_reference", func(t *testing.T) {
		original := []string{"admin", "editor"}
		claims := map[string]interface{}{
			"roles": original,
		}
		result := extractZitadelStringSlice(claims, "roles")

		// Modify result
		result[0] = "modified"

		// Original should be unchanged
		if original[0] != "admin" {
			t.Error("modifying result affected original slice")
		}
	})
}

// TestMapZitadelTokensToMetadata tests token metadata creation.
func TestMapZitadelTokensToMetadata(t *testing.T) {
	t.Run("all_tokens_present", func(t *testing.T) {
		metadata := MapZitadelTokensToMetadata("access", "refresh", "id")

		if metadata["access_token"] != "access" {
			t.Error("access_token mismatch")
		}
		if metadata["refresh_token"] != "refresh" {
			t.Error("refresh_token mismatch")
		}
		if metadata["id_token"] != "id" {
			t.Error("id_token mismatch")
		}
	})

	t.Run("empty_tokens_omitted", func(t *testing.T) {
		metadata := MapZitadelTokensToMetadata("access", "", "")

		if _, ok := metadata["refresh_token"]; ok {
			t.Error("empty refresh_token should be omitted")
		}
		if _, ok := metadata["id_token"]; ok {
			t.Error("empty id_token should be omitted")
		}
		if len(metadata) != 1 {
			t.Errorf("expected 1 entry, got %d", len(metadata))
		}
	})

	t.Run("all_empty_returns_empty_map", func(t *testing.T) {
		metadata := MapZitadelTokensToMetadata("", "", "")

		if len(metadata) != 0 {
			t.Errorf("expected empty map, got %v", metadata)
		}
	})
}

// TestExtractTokensFromMetadata tests token extraction helpers.
func TestExtractTokensFromMetadata(t *testing.T) {
	t.Run("nil_metadata_returns_empty", func(t *testing.T) {
		if ExtractAccessTokenFromMetadata(nil) != "" {
			t.Error("expected empty for nil metadata")
		}
		if ExtractRefreshTokenFromMetadata(nil) != "" {
			t.Error("expected empty for nil metadata")
		}
		if ExtractIDTokenFromMetadata(nil) != "" {
			t.Error("expected empty for nil metadata")
		}
	})

	t.Run("extracts_existing_tokens", func(t *testing.T) {
		metadata := map[string]string{
			"access_token":  "access-value",
			"refresh_token": "refresh-value",
			"id_token":      "id-value",
		}

		if ExtractAccessTokenFromMetadata(metadata) != "access-value" {
			t.Error("access token mismatch")
		}
		if ExtractRefreshTokenFromMetadata(metadata) != "refresh-value" {
			t.Error("refresh token mismatch")
		}
		if ExtractIDTokenFromMetadata(metadata) != "id-value" {
			t.Error("id token mismatch")
		}
	})

	t.Run("missing_token_returns_empty", func(t *testing.T) {
		metadata := map[string]string{}

		if ExtractAccessTokenFromMetadata(metadata) != "" {
			t.Error("expected empty for missing token")
		}
	})
}
