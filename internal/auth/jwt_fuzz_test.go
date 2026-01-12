// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"testing"
	"time"
	"unicode/utf8"

	"github.com/tomtom215/cartographus/internal/config"
)

// FuzzJWTValidateToken tests JWT token validation against malformed, tampered, and malicious inputs
func FuzzJWTValidateToken(f *testing.F) {
	// Create JWT manager with test secret
	cfg := &config.SecurityConfig{
		JWTSecret:      "test-secret-key-for-fuzzing-at-least-32-chars-long",
		SessionTimeout: 24 * time.Hour,
	}
	manager, err := NewJWTManager(cfg)
	if err != nil {
		f.Fatal(err)
	}

	// Seed corpus with known valid and invalid tokens
	validToken, _ := manager.GenerateToken("testuser", "admin")
	f.Add(validToken)                                                                                     // Valid token
	f.Add("")                                                                                             // Empty string
	f.Add("invalid.token.here")                                                                           // Simple malformed
	f.Add("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFkbWluIiwicm9sZSI6ImFkbWluIn0.invalid") // Invalid signature
	f.Add("eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1c2VybmFtZSI6ImFkbWluIiwicm9sZSI6ImFkbWluIn0.")         // Algorithm: none attack
	f.Add("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFkbWluIn0.sig")                         // Algorithm confusion (RS256)
	f.Add("..." + validToken)                                                                             // Prepended data
	f.Add(validToken + "...")                                                                             // Appended data
	f.Add(validToken[:len(validToken)-5])                                                                 // Truncated
	f.Add("Bearer " + validToken)                                                                         // With Bearer prefix
	f.Add("\x00" + validToken)                                                                            // Null byte prefix
	f.Add(validToken + "\x00")                                                                            // Null byte suffix

	f.Fuzz(func(t *testing.T, tokenString string) {
		// Validation should never panic, regardless of input
		claims, err := manager.ValidateToken(tokenString)

		// Valid tokens should have non-nil claims
		if err == nil && claims == nil {
			t.Error("ValidateToken returned nil error but nil claims")
		}

		// If claims are returned, they should have valid username
		if claims != nil {
			if claims.Username == "" {
				t.Error("ValidateToken returned claims with empty username")
			}
		}

		// Edge case: tokens with embedded null bytes should always fail
		for i := 0; i < len(tokenString); i++ {
			if tokenString[i] == 0 {
				if err == nil {
					t.Error("ValidateToken accepted token with null byte")
				}
				break
			}
		}
	})
}

// FuzzJWTGenerateToken tests token generation with various username/role combinations
func FuzzJWTGenerateToken(f *testing.F) {
	cfg := &config.SecurityConfig{
		JWTSecret:      "test-secret-key-for-fuzzing-at-least-32-chars-long",
		SessionTimeout: 24 * time.Hour,
	}
	manager, err := NewJWTManager(cfg)
	if err != nil {
		f.Fatal(err)
	}

	// Seed corpus with typical and edge case values
	f.Add("admin", "admin")
	f.Add("user", "user")
	f.Add("", "")                              // Empty username/role
	f.Add("user@example.com", "admin")         // Email as username
	f.Add("user\x00name", "role")              // Null byte in username
	f.Add("user", "role\x00")                  // Null byte in role
	f.Add("user;DROP TABLE users;--", "admin") // SQL injection attempt
	f.Add("<script>alert('xss')</script>", "") // XSS attempt
	f.Add("user' OR '1'='1", "admin")          // SQL injection variant
	f.Add("admin\nadmin", "role\nrole")        // Newline injection
	f.Add(string(make([]byte, 1000)), "admin") // Very long username
	f.Add("admin", string(make([]byte, 1000))) // Very long role

	f.Fuzz(func(t *testing.T, username, role string) {
		// Token generation should never panic
		token, err := manager.GenerateToken(username, role)

		if err != nil {
			// Errors are acceptable for some inputs
			return
		}

		// Generated token should not be empty
		if token == "" {
			t.Error("GenerateToken returned empty token without error")
		}

		// Generated token should be valid when parsed back
		claims, err := manager.ValidateToken(token)
		if err != nil {
			t.Errorf("Generated token failed validation: %v", err)
			return
		}

		// Claims should match input
		// Note: Invalid UTF-8 sequences may not round-trip correctly through JSON encoding
		// JSON will replace invalid UTF-8 with the replacement character (U+FFFD)
		if claims.Username != username {
			// Check if input was valid UTF-8
			if utf8Valid := utf8.ValidString(username); utf8Valid {
				t.Errorf("Username mismatch for valid UTF-8: got %q, want %q", claims.Username, username)
			}
			// For invalid UTF-8, we expect the username to be sanitized
		}
		if claims.Role != role {
			if utf8Valid := utf8.ValidString(role); utf8Valid {
				t.Errorf("Role mismatch for valid UTF-8: got %q, want %q", claims.Role, role)
			}
		}

		// Token should have valid expiration
		if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) {
			t.Error("Generated token has invalid expiration")
		}
	})
}
