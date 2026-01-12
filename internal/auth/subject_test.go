// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"testing"
	"time"
)

// TestAuthSubject_Creation tests the creation of AuthSubject
func TestAuthSubject_Creation(t *testing.T) {
	tests := []struct {
		name    string
		subject AuthSubject
		wantID  string
	}{
		{
			name: "basic subject creation",
			subject: AuthSubject{
				ID:       "user-123",
				Username: "testuser",
				Email:    "test@example.com",
				Roles:    []string{"viewer"},
			},
			wantID: "user-123",
		},
		{
			name: "subject with multiple roles",
			subject: AuthSubject{
				ID:       "admin-456",
				Username: "admin",
				Roles:    []string{"admin", "editor", "viewer"},
			},
			wantID: "admin-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.subject.ID != tt.wantID {
				t.Errorf("AuthSubject.ID = %v, want %v", tt.subject.ID, tt.wantID)
			}
		})
	}
}

// TestAuthSubject_HasRole tests role checking functionality
func TestAuthSubject_HasRole(t *testing.T) {
	subject := AuthSubject{
		ID:       "user-123",
		Username: "testuser",
		Roles:    []string{"admin", "viewer"},
	}

	tests := []struct {
		name string
		role string
		want bool
	}{
		{"has admin role", "admin", true},
		{"has viewer role", "viewer", true},
		{"does not have editor role", "editor", false},
		{"empty role", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := subject.HasRole(tt.role); got != tt.want {
				t.Errorf("HasRole(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

// TestAuthSubject_HasAnyRole tests checking for any of multiple roles
func TestAuthSubject_HasAnyRole(t *testing.T) {
	subject := AuthSubject{
		ID:    "user-123",
		Roles: []string{"viewer"},
	}

	tests := []struct {
		name  string
		roles []string
		want  bool
	}{
		{"has one of the roles", []string{"admin", "viewer"}, true},
		{"has none of the roles", []string{"admin", "editor"}, false},
		{"empty roles list", []string{}, false},
		{"exact match", []string{"viewer"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := subject.HasAnyRole(tt.roles...); got != tt.want {
				t.Errorf("HasAnyRole(%v) = %v, want %v", tt.roles, got, tt.want)
			}
		})
	}
}

// TestAuthSubject_IsExpired tests expiration checking
func TestAuthSubject_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt int64
		want      bool
	}{
		{"not expired (future)", time.Now().Add(1 * time.Hour).Unix(), false},
		{"expired (past)", time.Now().Add(-1 * time.Hour).Unix(), true},
		{"no expiry set", 0, false},
		{"expires exactly now", time.Now().Unix(), false}, // edge case: equal is not expired
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject := AuthSubject{
				ID:        "user-123",
				ExpiresAt: tt.expiresAt,
			}
			if got := subject.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAuthSubject_FromClaims tests conversion from Claims to AuthSubject
func TestAuthSubject_FromClaims(t *testing.T) {
	claims := &Claims{
		Username: "testuser",
		Role:     "admin",
	}

	subject := AuthSubjectFromClaims(claims)

	if subject.Username != claims.Username {
		t.Errorf("Username = %v, want %v", subject.Username, claims.Username)
	}

	if !subject.HasRole(claims.Role) {
		t.Errorf("Subject should have role %q", claims.Role)
	}

	if subject.AuthMethod != AuthModeJWT {
		t.Errorf("AuthMethod = %v, want %v", subject.AuthMethod, AuthModeJWT)
	}
}

// TestAuthSubject_ToClaims tests conversion from AuthSubject back to Claims
func TestAuthSubject_ToClaims(t *testing.T) {
	subject := AuthSubject{
		ID:       "user-123",
		Username: "testuser",
		Roles:    []string{"admin"},
	}

	claims := subject.ToClaims()

	if claims.Username != subject.Username {
		t.Errorf("Claims.Username = %v, want %v", claims.Username, subject.Username)
	}

	if claims.Role != "admin" {
		t.Errorf("Claims.Role = %v, want %v", claims.Role, "admin")
	}
}
