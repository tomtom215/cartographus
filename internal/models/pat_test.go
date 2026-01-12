// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"testing"
	"time"
)

func TestTokenScope_IsValidScope(t *testing.T) {
	tests := []struct {
		scope TokenScope
		valid bool
	}{
		{ScopeReadAnalytics, true},
		{ScopeReadUsers, true},
		{ScopeReadPlaybacks, true},
		{ScopeReadExport, true},
		{ScopeReadDetection, true},
		{ScopeReadWrapped, true},
		{ScopeReadSpatial, true},
		{ScopeReadLibraries, true},
		{ScopeWritePlaybacks, true},
		{ScopeWriteDetection, true},
		{ScopeWriteWrapped, true},
		{ScopeAdmin, true},
		{TokenScope("invalid:scope"), false},
		{TokenScope(""), false},
		{TokenScope("read"), false},
		{TokenScope("write"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.scope), func(t *testing.T) {
			if got := IsValidScope(tt.scope); got != tt.valid {
				t.Errorf("IsValidScope(%q) = %v, want %v", tt.scope, got, tt.valid)
			}
		})
	}
}

func TestAllScopes(t *testing.T) {
	scopes := AllScopes()

	// Should have 12 scopes (8 read + 3 write + 1 admin)
	expectedCount := 12
	if len(scopes) != expectedCount {
		t.Errorf("AllScopes() returned %d scopes, expected %d", len(scopes), expectedCount)
	}

	// All returned scopes should be valid
	for _, scope := range scopes {
		if !IsValidScope(scope) {
			t.Errorf("AllScopes() returned invalid scope: %s", scope)
		}
	}

	// Should include admin
	hasAdmin := false
	for _, scope := range scopes {
		if scope == ScopeAdmin {
			hasAdmin = true
			break
		}
	}
	if !hasAdmin {
		t.Error("AllScopes() should include admin scope")
	}
}

func TestReadOnlyScopes(t *testing.T) {
	scopes := ReadOnlyScopes()

	// Should have 8 read scopes
	expectedCount := 8
	if len(scopes) != expectedCount {
		t.Errorf("ReadOnlyScopes() returned %d scopes, expected %d", len(scopes), expectedCount)
	}

	// All should be read scopes
	for _, scope := range scopes {
		if scope == ScopeAdmin {
			t.Error("ReadOnlyScopes() should not include admin scope")
		}
		if scope == ScopeWritePlaybacks || scope == ScopeWriteDetection || scope == ScopeWriteWrapped {
			t.Errorf("ReadOnlyScopes() should not include write scope: %s", scope)
		}
	}
}

func TestStandardScopes(t *testing.T) {
	scopes := StandardScopes()

	// Should have 11 scopes (all except admin)
	expectedCount := 11
	if len(scopes) != expectedCount {
		t.Errorf("StandardScopes() returned %d scopes, expected %d", len(scopes), expectedCount)
	}

	// Should not include admin
	for _, scope := range scopes {
		if scope == ScopeAdmin {
			t.Error("StandardScopes() should not include admin scope")
		}
	}
}

func TestPersonalAccessToken_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{
			name:      "no expiration",
			expiresAt: nil,
			want:      false,
		},
		{
			name:      "future expiration",
			expiresAt: timePtr(time.Now().Add(24 * time.Hour)),
			want:      false,
		},
		{
			name:      "past expiration",
			expiresAt: timePtr(time.Now().Add(-24 * time.Hour)),
			want:      true,
		},
		{
			name:      "just expired",
			expiresAt: timePtr(time.Now().Add(-1 * time.Second)),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &PersonalAccessToken{
				ExpiresAt: tt.expiresAt,
			}
			if got := token.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPersonalAccessToken_IsRevoked(t *testing.T) {
	tests := []struct {
		name      string
		revokedAt *time.Time
		want      bool
	}{
		{
			name:      "not revoked",
			revokedAt: nil,
			want:      false,
		},
		{
			name:      "revoked",
			revokedAt: timePtr(time.Now()),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &PersonalAccessToken{
				RevokedAt: tt.revokedAt,
			}
			if got := token.IsRevoked(); got != tt.want {
				t.Errorf("IsRevoked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPersonalAccessToken_IsActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		expiresAt *time.Time
		revokedAt *time.Time
		want      bool
	}{
		{
			name:      "active - no expiry, not revoked",
			expiresAt: nil,
			revokedAt: nil,
			want:      true,
		},
		{
			name:      "active - future expiry, not revoked",
			expiresAt: timePtr(now.Add(24 * time.Hour)),
			revokedAt: nil,
			want:      true,
		},
		{
			name:      "inactive - expired",
			expiresAt: timePtr(now.Add(-24 * time.Hour)),
			revokedAt: nil,
			want:      false,
		},
		{
			name:      "inactive - revoked",
			expiresAt: nil,
			revokedAt: timePtr(now),
			want:      false,
		},
		{
			name:      "inactive - both expired and revoked",
			expiresAt: timePtr(now.Add(-24 * time.Hour)),
			revokedAt: timePtr(now),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &PersonalAccessToken{
				ExpiresAt: tt.expiresAt,
				RevokedAt: tt.revokedAt,
			}
			if got := token.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPersonalAccessToken_HasScope(t *testing.T) {
	tests := []struct {
		name   string
		scopes []TokenScope
		check  TokenScope
		want   bool
	}{
		{
			name:   "has scope",
			scopes: []TokenScope{ScopeReadAnalytics, ScopeReadUsers},
			check:  ScopeReadAnalytics,
			want:   true,
		},
		{
			name:   "missing scope",
			scopes: []TokenScope{ScopeReadAnalytics},
			check:  ScopeReadUsers,
			want:   false,
		},
		{
			name:   "admin grants all",
			scopes: []TokenScope{ScopeAdmin},
			check:  ScopeReadAnalytics,
			want:   true,
		},
		{
			name:   "admin grants write",
			scopes: []TokenScope{ScopeAdmin},
			check:  ScopeWritePlaybacks,
			want:   true,
		},
		{
			name:   "empty scopes",
			scopes: []TokenScope{},
			check:  ScopeReadAnalytics,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &PersonalAccessToken{
				Scopes: tt.scopes,
			}
			if got := token.HasScope(tt.check); got != tt.want {
				t.Errorf("HasScope(%s) = %v, want %v", tt.check, got, tt.want)
			}
		})
	}
}

func TestPersonalAccessToken_HasAnyScope(t *testing.T) {
	tests := []struct {
		name   string
		scopes []TokenScope
		check  []TokenScope
		want   bool
	}{
		{
			name:   "has first scope",
			scopes: []TokenScope{ScopeReadAnalytics},
			check:  []TokenScope{ScopeReadAnalytics, ScopeReadUsers},
			want:   true,
		},
		{
			name:   "has second scope",
			scopes: []TokenScope{ScopeReadUsers},
			check:  []TokenScope{ScopeReadAnalytics, ScopeReadUsers},
			want:   true,
		},
		{
			name:   "has none",
			scopes: []TokenScope{ScopeReadPlaybacks},
			check:  []TokenScope{ScopeReadAnalytics, ScopeReadUsers},
			want:   false,
		},
		{
			name:   "admin matches any",
			scopes: []TokenScope{ScopeAdmin},
			check:  []TokenScope{ScopeReadAnalytics, ScopeReadUsers},
			want:   true,
		},
		{
			name:   "empty check list",
			scopes: []TokenScope{ScopeReadAnalytics},
			check:  []TokenScope{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &PersonalAccessToken{
				Scopes: tt.scopes,
			}
			if got := token.HasAnyScope(tt.check...); got != tt.want {
				t.Errorf("HasAnyScope(%v) = %v, want %v", tt.check, got, tt.want)
			}
		})
	}
}

func TestPersonalAccessToken_IsIPAllowed(t *testing.T) {
	tests := []struct {
		name        string
		ipAllowlist []string
		checkIP     string
		want        bool
	}{
		{
			name:        "no allowlist allows all",
			ipAllowlist: nil,
			checkIP:     "192.168.1.1",
			want:        true,
		},
		{
			name:        "empty allowlist allows all",
			ipAllowlist: []string{},
			checkIP:     "192.168.1.1",
			want:        true,
		},
		{
			name:        "IP in allowlist",
			ipAllowlist: []string{"192.168.1.1", "10.0.0.1"},
			checkIP:     "192.168.1.1",
			want:        true,
		},
		{
			name:        "IP not in allowlist",
			ipAllowlist: []string{"192.168.1.1", "10.0.0.1"},
			checkIP:     "192.168.1.2",
			want:        false,
		},
		{
			name:        "IPv6 in allowlist",
			ipAllowlist: []string{"::1", "192.168.1.1"},
			checkIP:     "::1",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &PersonalAccessToken{
				IPAllowlist: tt.ipAllowlist,
			}
			if got := token.IsIPAllowed(tt.checkIP); got != tt.want {
				t.Errorf("IsIPAllowed(%s) = %v, want %v", tt.checkIP, got, tt.want)
			}
		})
	}
}

func TestTokenPrefixConst(t *testing.T) {
	expected := "carto_pat_"
	if TokenPrefixConst != expected {
		t.Errorf("TokenPrefixConst = %q, want %q", TokenPrefixConst, expected)
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
