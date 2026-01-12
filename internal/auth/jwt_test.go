// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/config"
)

func TestNewJWTManager(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.SecurityConfig
		wantErr bool
	}{
		{
			name: "valid secret",
			cfg: &config.SecurityConfig{
				JWTSecret:      "this_is_a_very_long_secret_key_with_32_plus_characters",
				SessionTimeout: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "empty secret",
			cfg: &config.SecurityConfig{
				JWTSecret:      "",
				SessionTimeout: 24 * time.Hour,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewJWTManager(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Error("NewJWTManager() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("NewJWTManager() unexpected error = %v", err)
				return
			}
			if manager == nil {
				t.Error("NewJWTManager() returned nil manager")
			}
		})
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	cfg := &config.SecurityConfig{
		JWTSecret:      "this_is_a_very_long_secret_key_for_testing_purposes_12345",
		SessionTimeout: 1 * time.Hour,
	}

	manager, err := NewJWTManager(cfg)
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}

	tests := []struct {
		name     string
		username string
		role     string
	}{
		{
			name:     "valid token",
			username: "testuser",
			role:     "admin",
		},
		{
			name:     "another valid token",
			username: "anotheruser",
			role:     "user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate token
			token, err := manager.GenerateToken(tt.username, tt.role)
			if err != nil {
				t.Errorf("GenerateToken() error = %v", err)
				return
			}
			if token == "" {
				t.Error("GenerateToken() returned empty token")
				return
			}

			// Validate token
			claims, err := manager.ValidateToken(token)
			if err != nil {
				t.Errorf("ValidateToken() error = %v", err)
				return
			}
			if claims == nil {
				t.Error("ValidateToken() returned nil claims")
				return
			}
			if claims.Username != tt.username {
				t.Errorf("ValidateToken() username = %v, want %v", claims.Username, tt.username)
			}
			if claims.Role != tt.role {
				t.Errorf("ValidateToken() role = %v, want %v", claims.Role, tt.role)
			}
		})
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	cfg := &config.SecurityConfig{
		JWTSecret:      "this_is_a_very_long_secret_key_for_testing_purposes_12345",
		SessionTimeout: 1 * time.Hour,
	}

	manager, err := NewJWTManager(cfg)
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "invalid token format",
			token: "invalid.token.format",
		},
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "malformed token",
			token: "not_a_jwt_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.ValidateToken(tt.token)
			if err == nil {
				t.Error("ValidateToken() expected error for invalid token, got nil")
			}
			if claims != nil {
				t.Error("ValidateToken() expected nil claims for invalid token")
			}
		})
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	cfg1 := &config.SecurityConfig{
		JWTSecret:      "first_secret_key_that_is_long_enough_for_testing_12345",
		SessionTimeout: 1 * time.Hour,
	}
	cfg2 := &config.SecurityConfig{
		JWTSecret:      "second_secret_key_that_is_different_from_first_12345",
		SessionTimeout: 1 * time.Hour,
	}

	manager1, err := NewJWTManager(cfg1)
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}

	manager2, err := NewJWTManager(cfg2)
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}

	// Generate token with first manager
	token, err := manager1.GenerateToken("testuser", "admin")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Try to validate with second manager (different secret)
	claims, err := manager2.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken() expected error when using wrong secret, got nil")
	}
	if claims != nil {
		t.Error("ValidateToken() expected nil claims when using wrong secret")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	cfg := &config.SecurityConfig{
		JWTSecret:      "secret_key_for_expiration_test_that_is_long_enough_12345",
		SessionTimeout: -1 * time.Hour, // Already expired
	}

	manager, err := NewJWTManager(cfg)
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}

	// Generate already-expired token
	token, err := manager.GenerateToken("testuser", "admin")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Try to validate expired token
	claims, err := manager.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken() expected error for expired token, got nil")
	}
	if claims != nil {
		t.Error("ValidateToken() expected nil claims for expired token")
	}
}
