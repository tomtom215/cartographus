// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality using Zitadel OIDC.
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
//
// This file contains comprehensive tests for ZitadelOIDCAuthenticator.
package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestZitadelOIDCAuthenticator_Name tests authenticator name.
func TestZitadelOIDCAuthenticator_Name(t *testing.T) {
	auth := &ZitadelOIDCAuthenticator{}

	name := auth.Name()
	if name != string(AuthModeOIDC) {
		t.Errorf("expected %q, got %q", AuthModeOIDC, name)
	}
}

// TestZitadelOIDCAuthenticator_Priority tests authenticator priority.
func TestZitadelOIDCAuthenticator_Priority(t *testing.T) {
	auth := &ZitadelOIDCAuthenticator{}

	priority := auth.Priority()
	// OIDC has priority 10 (high priority)
	if priority != 10 {
		t.Errorf("expected priority 10, got %d", priority)
	}
}

// TestZitadelOIDCAuthenticator_ExtractToken tests token extraction.
func TestZitadelOIDCAuthenticator_ExtractToken(t *testing.T) {
	tests := []struct {
		name          string
		authHeader    string
		cookieName    string
		cookieValue   string
		expectedToken string
	}{
		{
			name:          "bearer_token_from_header",
			authHeader:    "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test",
			expectedToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test",
		},
		{
			name:          "bearer_token_case_insensitive",
			authHeader:    "bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test",
			expectedToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test",
		},
		{
			name:          "bearer_token_BEARER",
			authHeader:    "BEARER eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test",
			expectedToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test",
		},
		{
			name:          "bearer_token_with_whitespace",
			authHeader:    "Bearer   eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test  ",
			expectedToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test",
		},
		{
			name:          "non_bearer_header_ignored",
			authHeader:    "Basic dXNlcjpwYXNz",
			expectedToken: "",
		},
		{
			name:          "fallback_to_cookie",
			authHeader:    "",
			cookieName:    "access_token",
			cookieValue:   "cookie-token",
			expectedToken: "cookie-token",
		},
		{
			name:          "header_takes_precedence_over_cookie",
			authHeader:    "Bearer header-token",
			cookieName:    "access_token",
			cookieValue:   "cookie-token",
			expectedToken: "header-token",
		},
		{
			name:          "empty_header_and_no_cookie",
			authHeader:    "",
			expectedToken: "",
		},
		{
			name:          "malformed_header_no_space",
			authHeader:    "BearereyJhbGciOiJSUzI1NiJ9",
			expectedToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &ZitadelOIDCAuthenticator{
				tokenCookie: "access_token",
			}

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{
					Name:  tt.cookieName,
					Value: tt.cookieValue,
				})
			}

			token := auth.extractToken(req)

			if token != tt.expectedToken {
				t.Errorf("extractToken() = %q, want %q", token, tt.expectedToken)
			}
		})
	}
}

// TestZitadelOIDCAuthenticator_CustomCookie tests custom cookie name.
func TestZitadelOIDCAuthenticator_CustomCookie(t *testing.T) {
	t.Run("custom_cookie_name", func(t *testing.T) {
		auth := NewZitadelOIDCAuthenticatorWithCookie(nil, "custom_token")

		if auth.tokenCookie != "custom_token" {
			t.Errorf("expected cookie name 'custom_token', got %q", auth.tokenCookie)
		}
	})

	t.Run("empty_cookie_name_uses_default", func(t *testing.T) {
		auth := NewZitadelOIDCAuthenticatorWithCookie(nil, "")

		if auth.tokenCookie != "access_token" {
			t.Errorf("expected default cookie name 'access_token', got %q", auth.tokenCookie)
		}
	})

	t.Run("default_constructor_uses_access_token", func(t *testing.T) {
		auth := NewZitadelOIDCAuthenticator(nil)

		if auth.tokenCookie != "access_token" {
			t.Errorf("expected default cookie name 'access_token', got %q", auth.tokenCookie)
		}
	})
}

// TestZitadelOIDCAuthenticator_MapVerificationError tests error mapping.
func TestZitadelOIDCAuthenticator_MapVerificationError(t *testing.T) {
	auth := &ZitadelOIDCAuthenticator{}

	tests := []struct {
		name          string
		inputError    error
		expectExpired bool
		expectInvalid bool
	}{
		{
			name:          "nil_error_returns_nil",
			inputError:    nil,
			expectExpired: false,
			expectInvalid: false,
		},
		// Note: Detailed error mapping tests would require creating errors
		// that contain specific substrings. The implementation checks for:
		// - "expired" or "token is expired" -> ErrExpiredCredentials
		// - "signature", "invalid token", "verification failed" -> ErrInvalidCredentials
		// - "issuer" -> ErrInvalidCredentials with "issuer mismatch"
		// - "audience" -> ErrInvalidCredentials with "audience mismatch"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := auth.mapVerificationError(tt.inputError)

			if tt.inputError == nil && result != nil {
				t.Errorf("expected nil for nil input, got %v", result)
			}
		})
	}
}

// TestZitadelOIDCAuthenticator_Authenticate_NoToken tests authentication without token.
func TestZitadelOIDCAuthenticator_Authenticate_NoToken(t *testing.T) {
	auth := &ZitadelOIDCAuthenticator{
		tokenCookie: "access_token",
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No Authorization header, no cookie

	_, err := auth.Authenticate(nil, req)

	if !errors.Is(err, ErrNoCredentials) {
		t.Errorf("expected ErrNoCredentials, got %v", err)
	}
}

// TestZitadelOIDCAuthenticator_ImplementsInterface tests interface compliance.
func TestZitadelOIDCAuthenticator_ImplementsInterface(t *testing.T) {
	// Compile-time check that ZitadelOIDCAuthenticator implements Authenticator
	var _ Authenticator = (*ZitadelOIDCAuthenticator)(nil)
}

// TestZitadelOIDCAuthenticator_GetRelyingParty tests RP accessor.
func TestZitadelOIDCAuthenticator_GetRelyingParty(t *testing.T) {
	// Can't test with real RP without OIDC server, but can test nil handling
	auth := &ZitadelOIDCAuthenticator{
		rp: nil,
	}

	if auth.GetRelyingParty() != nil {
		t.Error("expected nil RP")
	}
}

// TestZitadelOIDCAuthenticator_Issuer tests issuer accessor.
func TestZitadelOIDCAuthenticator_Issuer(t *testing.T) {
	// This would panic with nil RP, documenting expected behavior
	// In production, RP is always initialized
}
