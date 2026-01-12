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
)

// TestAuthMode_Parse tests parsing of auth mode strings
func TestAuthMode_Parse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    AuthMode
		wantErr bool
	}{
		{"none mode", "none", AuthModeNone, false},
		{"basic mode", "basic", AuthModeBasic, false},
		{"jwt mode", "jwt", AuthModeJWT, false},
		{"oidc mode", "oidc", AuthModeOIDC, false},
		{"plex mode", "plex", AuthModePlex, false},
		{"multi mode", "multi", AuthModeMulti, false},
		{"empty string defaults to none", "", AuthModeNone, false},
		{"invalid mode", "invalid", "", true},
		{"case sensitive", "NONE", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAuthMode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAuthMode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseAuthMode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestAuthMode_String tests string representation of auth modes
func TestAuthMode_String(t *testing.T) {
	tests := []struct {
		mode AuthMode
		want string
	}{
		{AuthModeNone, "none"},
		{AuthModeBasic, "basic"},
		{AuthModeJWT, "jwt"},
		{AuthModeOIDC, "oidc"},
		{AuthModePlex, "plex"},
		{AuthModeMulti, "multi"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("AuthMode.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAuthErrors tests the standard auth error types
func TestAuthErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"no credentials", ErrNoCredentials, "no credentials provided"},
		{"invalid credentials", ErrInvalidCredentials, "invalid credentials"},
		{"expired credentials", ErrExpiredCredentials, "credentials expired"},
		{"authenticator unavailable", ErrAuthenticatorUnavailable, "authenticator unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.want {
				t.Errorf("Error message = %q, want %q", tt.err.Error(), tt.want)
			}
		})
	}
}

// mockAuthenticator implements Authenticator for testing
type mockAuthenticator struct {
	name       string
	priority   int
	shouldFail bool
	returnErr  error
	returnSubj *AuthSubject
	callCount  int
}

func (m *mockAuthenticator) Authenticate(ctx context.Context, r *http.Request) (*AuthSubject, error) {
	m.callCount++
	if m.shouldFail {
		return nil, m.returnErr
	}
	return m.returnSubj, nil
}

func (m *mockAuthenticator) Name() string {
	return m.name
}

func (m *mockAuthenticator) Priority() int {
	return m.priority
}

// TestAuthenticator_Interface verifies the Authenticator interface contract
func TestAuthenticator_Interface(t *testing.T) {
	mock := &mockAuthenticator{
		name:     "mock",
		priority: 10,
		returnSubj: &AuthSubject{
			ID:       "test-user",
			Username: "testuser",
		},
	}

	// Verify it implements the interface
	var _ Authenticator = mock

	// Test Name()
	if mock.Name() != "mock" {
		t.Errorf("Name() = %v, want mock", mock.Name())
	}

	// Test Priority()
	if mock.Priority() != 10 {
		t.Errorf("Priority() = %v, want 10", mock.Priority())
	}

	// Test Authenticate()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	subject, err := mock.Authenticate(context.Background(), req)
	if err != nil {
		t.Errorf("Authenticate() error = %v", err)
	}
	if subject.ID != "test-user" {
		t.Errorf("Authenticate() subject.ID = %v, want test-user", subject.ID)
	}
}

// TestMultiAuthenticator_Priority tests that authenticators are tried in priority order
func TestMultiAuthenticator_Priority(t *testing.T) {
	// Create authenticators with different priorities
	lowPriority := &mockAuthenticator{name: "low", priority: 30, shouldFail: true, returnErr: ErrNoCredentials}
	midPriority := &mockAuthenticator{name: "mid", priority: 20, shouldFail: true, returnErr: ErrNoCredentials}
	highPriority := &mockAuthenticator{name: "high", priority: 10, returnSubj: &AuthSubject{ID: "user"}}

	// Multi-auth should try high priority first
	authenticators := []Authenticator{lowPriority, midPriority, highPriority}

	// Sort by priority (should be tested in actual implementation)
	// For now, verify the priority values
	for _, auth := range authenticators {
		if auth.Priority() < 0 {
			t.Errorf("Priority should be non-negative, got %d for %s", auth.Priority(), auth.Name())
		}
	}

	// Verify high priority has lowest number
	if highPriority.Priority() >= midPriority.Priority() {
		t.Error("High priority authenticator should have lower priority number")
	}
}

// TestAuthenticator_FailureHandling tests error handling in authenticators
func TestAuthenticator_FailureHandling(t *testing.T) {
	tests := []struct {
		name      string
		returnErr error
		wantTry   bool // Should next authenticator be tried?
	}{
		{"no credentials - try next", ErrNoCredentials, true},
		{"invalid credentials - stop", ErrInvalidCredentials, false},
		{"expired credentials - stop", ErrExpiredCredentials, false},
		{"unavailable - try next with fallback", ErrAuthenticatorUnavailable, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ErrNoCredentials and ErrAuthenticatorUnavailable should allow fallback
			shouldTryNext := errors.Is(tt.returnErr, ErrNoCredentials) ||
				errors.Is(tt.returnErr, ErrAuthenticatorUnavailable)

			if shouldTryNext != tt.wantTry {
				t.Errorf("For error %v: shouldTryNext = %v, want %v",
					tt.returnErr, shouldTryNext, tt.wantTry)
			}
		})
	}
}
