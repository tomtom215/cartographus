// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/config"
)

// testSecurityConfig creates a security config for testing JWT authenticator
func testSecurityConfig() *config.SecurityConfig {
	return &config.SecurityConfig{
		JWTSecret:      "test-secret-key-that-is-at-least-32-characters-long",
		SessionTimeout: 1 * time.Hour,
	}
}

func TestJWTAuthenticator_Authenticate_Success(t *testing.T) {
	jwtManager, err := NewJWTManager(testSecurityConfig())
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	auth := NewJWTAuthenticator(jwtManager)

	// Generate a valid token
	token, err := jwtManager.GenerateToken("testuser", "admin")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	tests := []struct {
		name         string
		setupRequest func(*http.Request)
		wantUsername string
		wantRole     string
	}{
		{
			name: "valid token in Authorization header",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+token)
			},
			wantUsername: "testuser",
			wantRole:     "admin",
		},
		{
			name: "valid token in cookie",
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: "token", Value: token})
			},
			wantUsername: "testuser",
			wantRole:     "admin",
		},
		{
			name: "authorization header takes precedence over cookie",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+token)
				// Add a different (invalid) token in cookie
				r.AddCookie(&http.Cookie{Name: "token", Value: "invalid-token"})
			},
			wantUsername: "testuser",
			wantRole:     "admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setupRequest(req)

			subject, err := auth.Authenticate(context.Background(), req)
			if err != nil {
				t.Errorf("Authenticate() error = %v", err)
				return
			}

			if subject.Username != tt.wantUsername {
				t.Errorf("Authenticate() username = %v, want %v", subject.Username, tt.wantUsername)
			}

			if !subject.HasRole(tt.wantRole) {
				t.Errorf("Authenticate() should have role %v, has %v", tt.wantRole, subject.Roles)
			}

			if subject.AuthMethod != AuthModeJWT {
				t.Errorf("Authenticate() AuthMethod = %v, want %v", subject.AuthMethod, AuthModeJWT)
			}

			if subject.Issuer != "local" {
				t.Errorf("Authenticate() Issuer = %v, want local", subject.Issuer)
			}
		})
	}
}

func TestJWTAuthenticator_Authenticate_Errors(t *testing.T) {
	jwtManager, err := NewJWTManager(testSecurityConfig())
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	auth := NewJWTAuthenticator(jwtManager)

	tests := []struct {
		name         string
		setupRequest func(*http.Request)
		wantErr      error
	}{
		{
			name:         "no credentials",
			setupRequest: func(r *http.Request) {},
			wantErr:      ErrNoCredentials,
		},
		{
			name: "invalid token",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer invalid.jwt.token")
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name: "malformed authorization header - no Bearer",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "invalid-token")
			},
			wantErr: ErrNoCredentials,
		},
		{
			name: "malformed authorization header - wrong scheme",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
			},
			wantErr: ErrNoCredentials,
		},
		{
			name: "empty bearer token",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer ")
			},
			wantErr: ErrNoCredentials,
		},
		{
			name: "empty cookie value",
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: "token", Value: ""})
			},
			wantErr: ErrNoCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setupRequest(req)

			_, err := auth.Authenticate(context.Background(), req)
			if err == nil {
				t.Errorf("Authenticate() expected error %v, got nil", tt.wantErr)
				return
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Authenticate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestJWTAuthenticator_Authenticate_ExpiredToken(t *testing.T) {
	// Create a JWT manager with very short expiry
	shortExpiryConfig := &config.SecurityConfig{
		JWTSecret:      "test-secret-key-that-is-at-least-32-characters-long",
		SessionTimeout: 1 * time.Millisecond,
	}

	jwtManager, err := NewJWTManager(shortExpiryConfig)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	auth := NewJWTAuthenticator(jwtManager)

	// Generate a token
	token, err := jwtManager.GenerateToken("testuser", "admin")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	_, err = auth.Authenticate(context.Background(), req)
	if !errors.Is(err, ErrExpiredCredentials) {
		t.Errorf("Authenticate() error = %v, want %v", err, ErrExpiredCredentials)
	}
}

func TestJWTAuthenticator_Name(t *testing.T) {
	jwtManager, _ := NewJWTManager(testSecurityConfig())
	auth := NewJWTAuthenticator(jwtManager)

	if auth.Name() != string(AuthModeJWT) {
		t.Errorf("Name() = %v, want %v", auth.Name(), AuthModeJWT)
	}
}

func TestJWTAuthenticator_Priority(t *testing.T) {
	jwtManager, _ := NewJWTManager(testSecurityConfig())
	auth := NewJWTAuthenticator(jwtManager)

	// JWT should have priority 20 (between Plex at 15 and Basic at 25)
	if auth.Priority() != 20 {
		t.Errorf("Priority() = %v, want 20", auth.Priority())
	}
}

func TestJWTAuthenticator_ImplementsInterface(t *testing.T) {
	jwtManager, _ := NewJWTManager(testSecurityConfig())
	auth := NewJWTAuthenticator(jwtManager)

	// Verify it implements the Authenticator interface
	var _ Authenticator = auth
}

func TestJWTAuthenticator_SubjectConversion(t *testing.T) {
	jwtManager, _ := NewJWTManager(testSecurityConfig())
	auth := NewJWTAuthenticator(jwtManager)

	token, _ := jwtManager.GenerateToken("testuser", "editor")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Convert back to Claims for backwards compatibility
	claims := subject.ToClaims()
	if claims.Username != "testuser" {
		t.Errorf("ToClaims() username = %v, want testuser", claims.Username)
	}
	if claims.Role != "editor" {
		t.Errorf("ToClaims() role = %v, want editor", claims.Role)
	}
}

func TestJWTAuthenticator_CaseInsensitiveBearer(t *testing.T) {
	jwtManager, _ := NewJWTManager(testSecurityConfig())
	auth := NewJWTAuthenticator(jwtManager)

	token, _ := jwtManager.GenerateToken("testuser", "admin")

	// Test case-insensitive "Bearer" prefix
	schemes := []string{"Bearer", "bearer", "BEARER", "BeArEr"}

	for _, scheme := range schemes {
		t.Run(scheme, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", scheme+" "+token)

			subject, err := auth.Authenticate(context.Background(), req)
			if err != nil {
				t.Errorf("Authenticate() with scheme %q error = %v", scheme, err)
				return
			}

			if subject.Username != "testuser" {
				t.Errorf("Authenticate() username = %v, want testuser", subject.Username)
			}
		})
	}
}
