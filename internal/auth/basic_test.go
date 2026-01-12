// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestNewBasicAuthManager(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		password    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid credentials",
			username:    "admin",
			password:    "securepassword123",
			expectError: false,
		},
		{
			name:        "minimum password length",
			username:    "admin",
			password:    "12345678", // exactly 8 chars
			expectError: false,
		},
		{
			name:        "empty username",
			username:    "",
			password:    "securepassword123",
			expectError: true,
			errorMsg:    "username is required",
		},
		{
			name:        "empty password",
			username:    "admin",
			password:    "",
			expectError: true,
			errorMsg:    "password is required",
		},
		{
			name:        "password too short",
			username:    "admin",
			password:    "1234567", // 7 chars
			expectError: true,
			errorMsg:    "password must be at least 8 characters",
		},
		{
			name:        "both empty",
			username:    "",
			password:    "",
			expectError: true,
			errorMsg:    "username is required",
		},
		{
			name:        "long credentials (under bcrypt 72-byte limit)",
			username:    "very_long_username_" + strings.Repeat("a", 30),
			password:    "very_long_password_" + strings.Repeat("b", 30), // 49 bytes total (under 72 limit)
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewBasicAuthManager(tt.username, tt.password)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				if manager != nil {
					t.Errorf("Expected nil manager on error, got %v", manager)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if manager == nil {
					t.Errorf("Expected non-nil manager")
				} else {
					// Verify password is hashed (not stored in plaintext)
					if len(manager.passwordHash) == 0 {
						t.Errorf("Password hash should not be empty")
					}
					if string(manager.passwordHash) == tt.password {
						t.Errorf("Password should be hashed, not stored in plaintext")
					}
					// Verify username is stored
					if manager.username != tt.username {
						t.Errorf("Expected username %s, got %s", tt.username, manager.username)
					}
				}
			}
		})
	}
}

func TestValidateCredentials(t *testing.T) {
	// Create a valid manager for testing (using fast MinCost for 125x speedup)
	manager, err := newBasicAuthManagerForTest("admin", "securepass123")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Helper function to create Basic Auth header
	makeAuthHeader := func(username, password string) string {
		credentials := username + ":" + password
		encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
		return "Basic " + encoded
	}

	tests := []struct {
		name        string
		authHeader  string
		expectValid bool
		expectUser  string
	}{
		{
			name:        "valid credentials",
			authHeader:  makeAuthHeader("admin", "securepass123"),
			expectValid: true,
			expectUser:  "admin",
		},
		{
			name:        "wrong password",
			authHeader:  makeAuthHeader("admin", "wrongpassword"),
			expectValid: false,
		},
		{
			name:        "wrong username",
			authHeader:  makeAuthHeader("hacker", "securepass123"),
			expectValid: false,
		},
		{
			name:        "both wrong",
			authHeader:  makeAuthHeader("hacker", "wrongpass"),
			expectValid: false,
		},
		{
			name:        "empty username",
			authHeader:  makeAuthHeader("", "securepass123"),
			expectValid: false,
		},
		{
			name:        "empty password",
			authHeader:  makeAuthHeader("admin", ""),
			expectValid: false,
		},
		{
			name:        "both empty",
			authHeader:  makeAuthHeader("", ""),
			expectValid: false,
		},
		{
			name:        "missing Basic prefix",
			authHeader:  base64.StdEncoding.EncodeToString([]byte("admin:securepass123")),
			expectValid: false,
		},
		{
			name:        "wrong scheme (Bearer)",
			authHeader:  "Bearer " + base64.StdEncoding.EncodeToString([]byte("admin:securepass123")),
			expectValid: false,
		},
		{
			name:        "invalid base64",
			authHeader:  "Basic !!invalid!!",
			expectValid: false,
		},
		{
			name:        "missing colon separator",
			authHeader:  "Basic " + base64.StdEncoding.EncodeToString([]byte("adminsecurepass123")),
			expectValid: false,
		},
		{
			name:        "multiple colons in credentials",
			authHeader:  makeAuthHeader("admin", "pass:with:colons"),
			expectValid: false, // Our user's password doesn't have colons
		},
		{
			name:        "case sensitive username",
			authHeader:  makeAuthHeader("Admin", "securepass123"), // capital A
			expectValid: false,
		},
		{
			name:        "case sensitive password",
			authHeader:  makeAuthHeader("admin", "SecurePass123"), // capital S and P
			expectValid: false,
		},
		{
			name:        "extra whitespace in header",
			authHeader:  "  Basic  " + base64.StdEncoding.EncodeToString([]byte("admin:securepass123")),
			expectValid: false, // Should not trim automatically
		},
		{
			name:        "empty header",
			authHeader:  "",
			expectValid: false,
		},
		{
			name:        "just 'Basic'",
			authHeader:  "Basic",
			expectValid: false,
		},
		{
			name:        "just 'Basic '",
			authHeader:  "Basic ",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username, err := manager.ValidateCredentials(tt.authHeader)

			if tt.expectValid {
				if err != nil {
					t.Errorf("Expected valid credentials, got error: %v", err)
				}
				if username != tt.expectUser {
					t.Errorf("Expected username %s, got %s", tt.expectUser, username)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for invalid credentials, got username: %s", username)
				}
				if username != "" {
					t.Errorf("Expected empty username on error, got %s", username)
				}
			}
		})
	}
}

// TestTimingSafety verifies that credential validation is timing-safe
// This is a basic test - true timing attack resistance requires statistical analysis
func TestTimingSafety(t *testing.T) {
	manager, err := newBasicAuthManagerForTest("admin", "securepassword123")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	makeAuthHeader := func(username, password string) string {
		credentials := username + ":" + password
		encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
		return "Basic " + encoded
	}

	// Test multiple wrong credentials to ensure no timing leaks
	testCases := []struct {
		username string
		password string
	}{
		{"admin", "wrong1"},
		{"admin", "wrong2"},
		{"wrong", "securepassword123"},
		{"wrong", "wrong"},
		{"a", "b"},
		{"", ""},
	}

	// All invalid attempts should complete in similar time
	// We're not doing statistical analysis here, just basic validation
	for _, tc := range testCases {
		t.Run(tc.username+":"+tc.password, func(t *testing.T) {
			start := time.Now()
			_, err := manager.ValidateCredentials(makeAuthHeader(tc.username, tc.password))
			duration := time.Since(start)

			if err == nil {
				t.Errorf("Expected error for invalid credentials")
			}

			// Basic sanity check - validation should complete within reasonable time
			// Note: bcrypt with cost 12 is intentionally slow (0.5-3s) to prevent brute force
			// This is a security feature, not a bug. We check for > 5s to catch true DoS issues.
			if duration > 5*time.Second {
				t.Errorf("Validation took too long: %v (possible DoS vector)", duration)
			}
		})
	}
}

// TestPasswordHashingSecurity verifies password is properly hashed
func TestPasswordHashingSecurity(t *testing.T) {
	password := "testpassword123"
	manager1, err := newBasicAuthManagerForTest("user1", password)
	if err != nil {
		t.Fatalf("Failed to create manager1: %v", err)
	}

	manager2, err := newBasicAuthManagerForTest("user2", password)
	if err != nil {
		t.Fatalf("Failed to create manager2: %v", err)
	}

	// Same password should produce different hashes (bcrypt uses salt)
	if string(manager1.passwordHash) == string(manager2.passwordHash) {
		t.Errorf("Same password produced identical hashes - salt not working")
	}

	// Hash should not be the plaintext password
	if string(manager1.passwordHash) == password {
		t.Errorf("Password stored in plaintext - not hashed!")
	}

	// Hash should look like a bcrypt hash (starts with $2a$ or $2b$)
	hashStr := string(manager1.passwordHash)
	if !strings.HasPrefix(hashStr, "$2a$") && !strings.HasPrefix(hashStr, "$2b$") {
		t.Errorf("Hash doesn't look like a bcrypt hash: %s", hashStr)
	}
}

// TestGetWWWAuthenticateHeader verifies proper header generation
func TestGetWWWAuthenticateHeader(t *testing.T) {
	manager, err := newBasicAuthManagerForTest("admin", "password123")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	header := manager.GetWWWAuthenticateHeader()

	// Should start with "Basic realm="
	if !strings.HasPrefix(header, "Basic realm=") {
		t.Errorf("Expected header to start with 'Basic realm=', got: %s", header)
	}

	// Should contain the realm name
	if !strings.Contains(header, "Cartographus") {
		t.Errorf("Expected header to contain 'Cartographus', got: %s", header)
	}

	// Should specify charset
	if !strings.Contains(header, "charset=") {
		t.Errorf("Expected header to contain charset specification, got: %s", header)
	}
}

// TestSpecialCharactersInCredentials tests handling of special characters
func TestSpecialCharactersInCredentials(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
	}{
		{"spaces in password", "admin", "pass word 123"},
		{"special chars in password", "admin", "p@$$w0rd!#%"},
		{"unicode in password", "admin", "пароль密码🔒"},
		{"spaces in username", "admin user", "password123"},
		{"special chars in username", "admin@example.com", "password123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := newBasicAuthManagerForTest(tt.username, tt.password)
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			// Create auth header
			credentials := tt.username + ":" + tt.password
			encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
			authHeader := "Basic " + encoded

			// Validate
			username, err := manager.ValidateCredentials(authHeader)
			if err != nil {
				t.Errorf("Failed to validate special characters: %v", err)
			}
			if username != tt.username {
				t.Errorf("Expected username %s, got %s", tt.username, username)
			}
		})
	}
}

// TestColonInPassword tests handling of passwords containing colons
func TestColonInPassword(t *testing.T) {
	manager, err := newBasicAuthManagerForTest("admin", "pass:word:123")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create auth header with colon in password
	credentials := "admin:pass:word:123"
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	authHeader := "Basic " + encoded

	username, err := manager.ValidateCredentials(authHeader)
	if err != nil {
		t.Errorf("Failed to validate password with colons: %v", err)
	}
	if username != "admin" {
		t.Errorf("Expected username 'admin', got %s", username)
	}
}
