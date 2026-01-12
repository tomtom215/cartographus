// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package models provides data structures for the Cartographus application.
// This file contains models for Personal Access Tokens (PATs) - programmatic API access.
package models

import (
	"time"
)

// TokenScope represents a permission scope for a PAT.
type TokenScope string

// Token scopes define granular permissions for API access.
const (
	// Read scopes
	ScopeReadAnalytics TokenScope = "read:analytics"
	ScopeReadUsers     TokenScope = "read:users"
	ScopeReadPlaybacks TokenScope = "read:playbacks"
	ScopeReadExport    TokenScope = "read:export"
	ScopeReadDetection TokenScope = "read:detection"
	ScopeReadWrapped   TokenScope = "read:wrapped"
	ScopeReadSpatial   TokenScope = "read:spatial"
	ScopeReadLibraries TokenScope = "read:libraries"

	// Write scopes
	ScopeWritePlaybacks TokenScope = "write:playbacks"
	ScopeWriteDetection TokenScope = "write:detection"
	ScopeWriteWrapped   TokenScope = "write:wrapped"

	// Admin scope (all permissions)
	ScopeAdmin TokenScope = "admin"
)

// AllScopes returns all available token scopes.
func AllScopes() []TokenScope {
	return []TokenScope{
		ScopeReadAnalytics,
		ScopeReadUsers,
		ScopeReadPlaybacks,
		ScopeReadExport,
		ScopeReadDetection,
		ScopeReadWrapped,
		ScopeReadSpatial,
		ScopeReadLibraries,
		ScopeWritePlaybacks,
		ScopeWriteDetection,
		ScopeWriteWrapped,
		ScopeAdmin,
	}
}

// ReadOnlyScopes returns all read-only scopes.
func ReadOnlyScopes() []TokenScope {
	return []TokenScope{
		ScopeReadAnalytics,
		ScopeReadUsers,
		ScopeReadPlaybacks,
		ScopeReadExport,
		ScopeReadDetection,
		ScopeReadWrapped,
		ScopeReadSpatial,
		ScopeReadLibraries,
	}
}

// StandardScopes returns read and write scopes (no admin).
func StandardScopes() []TokenScope {
	return []TokenScope{
		ScopeReadAnalytics,
		ScopeReadUsers,
		ScopeReadPlaybacks,
		ScopeReadExport,
		ScopeReadDetection,
		ScopeReadWrapped,
		ScopeReadSpatial,
		ScopeReadLibraries,
		ScopeWritePlaybacks,
		ScopeWriteDetection,
		ScopeWriteWrapped,
	}
}

// IsValidScope checks if a scope is valid.
func IsValidScope(scope TokenScope) bool {
	for _, s := range AllScopes() {
		if s == scope {
			return true
		}
	}
	return false
}

// PersonalAccessToken represents a PAT for programmatic API access.
// Tokens are similar to GitHub PATs - they can be scoped, have expiration,
// and support IP allowlisting.
//
// Security:
//   - Token hash is stored, never the plaintext token
//   - Token prefix is stored for identification (first 8 chars)
//   - Tokens can be revoked at any time
//   - Expiration is enforced on every request
type PersonalAccessToken struct {
	// Identification
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`

	// Token (prefix only stored, hash for validation)
	TokenPrefix string `json:"token_prefix"` // First 8 chars for identification
	TokenHash   string `json:"-"`            // bcrypt hash, never exposed in JSON

	// Permissions
	Scopes []TokenScope `json:"scopes"`

	// Expiration
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// Usage tracking
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	LastUsedIP string     `json:"last_used_ip,omitempty"`
	UseCount   int        `json:"use_count"`

	// IP restriction (optional)
	IPAllowlist []string `json:"ip_allowlist,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`

	// Revocation
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	RevokedBy    string     `json:"revoked_by,omitempty"`
	RevokeReason string     `json:"revoke_reason,omitempty"`
}

// IsExpired checks if the token has expired.
func (t *PersonalAccessToken) IsExpired() bool {
	if t.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*t.ExpiresAt)
}

// IsRevoked checks if the token has been revoked.
func (t *PersonalAccessToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

// IsActive checks if the token is active (not expired, not revoked).
func (t *PersonalAccessToken) IsActive() bool {
	return !t.IsExpired() && !t.IsRevoked()
}

// HasScope checks if the token has a specific scope.
func (t *PersonalAccessToken) HasScope(scope TokenScope) bool {
	for _, s := range t.Scopes {
		if s == ScopeAdmin || s == scope {
			return true
		}
	}
	return false
}

// HasAnyScope checks if the token has any of the given scopes.
func (t *PersonalAccessToken) HasAnyScope(scopes ...TokenScope) bool {
	for _, scope := range scopes {
		if t.HasScope(scope) {
			return true
		}
	}
	return false
}

// IsIPAllowed checks if an IP address is allowed for this token.
// Returns true if no allowlist is configured (all IPs allowed).
func (t *PersonalAccessToken) IsIPAllowed(ip string) bool {
	if len(t.IPAllowlist) == 0 {
		return true
	}
	for _, allowed := range t.IPAllowlist {
		if allowed == ip {
			return true
		}
		// TODO: Add CIDR matching support
	}
	return false
}

// CreatePATRequest represents a request to create a new PAT.
type CreatePATRequest struct {
	Name        string       `json:"name" validate:"required,min=1,max=100"`
	Description string       `json:"description,omitempty" validate:"max=500"`
	Scopes      []TokenScope `json:"scopes" validate:"required,min=1,dive"`
	ExpiresIn   *int         `json:"expires_in_days,omitempty" validate:"omitempty,min=1,max=365"`
	IPAllowlist []string     `json:"ip_allowlist,omitempty" validate:"omitempty,dive,ip"`
}

// CreatePATResponse represents the response when creating a PAT.
// IMPORTANT: The plaintext token is only returned ONCE during creation.
type CreatePATResponse struct {
	Token *PersonalAccessToken `json:"token"`
	// PlaintextToken is the full token value - only shown once!
	// Format: carto_pat_<base64-id>_<random-secret>
	PlaintextToken string `json:"plaintext_token"`
}

// UpdatePATRequest represents a request to update a PAT.
type UpdatePATRequest struct {
	Name        string       `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description string       `json:"description,omitempty" validate:"max=500"`
	Scopes      []TokenScope `json:"scopes,omitempty" validate:"omitempty,min=1,dive"`
	ExpiresIn   *int         `json:"expires_in_days,omitempty" validate:"omitempty,min=1,max=365"`
	IPAllowlist []string     `json:"ip_allowlist,omitempty" validate:"omitempty,dive,ip"`
}

// RevokePATRequest represents a request to revoke a PAT.
type RevokePATRequest struct {
	Reason string `json:"reason,omitempty" validate:"max=500"`
}

// ListPATsResponse represents the response when listing PATs.
type ListPATsResponse struct {
	Tokens     []PersonalAccessToken `json:"tokens"`
	TotalCount int                   `json:"total_count"`
}

// PATUsageLog represents a usage log entry for a PAT.
type PATUsageLog struct {
	ID             string    `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	TokenID        string    `json:"token_id"`
	UserID         string    `json:"user_id"`
	Action         string    `json:"action"` // "authenticate", "request", "revoke"
	Endpoint       string    `json:"endpoint,omitempty"`
	Method         string    `json:"method,omitempty"`
	IPAddress      string    `json:"ip_address,omitempty"`
	UserAgent      string    `json:"user_agent,omitempty"`
	Success        bool      `json:"success"`
	ErrorCode      string    `json:"error_code,omitempty"`
	ResponseTimeMS int       `json:"response_time_ms,omitempty"`
}

// PATStats represents aggregated statistics for a user's PATs.
type PATStats struct {
	TotalTokens   int       `json:"total_tokens"`
	ActiveTokens  int       `json:"active_tokens"`
	ExpiredTokens int       `json:"expired_tokens"`
	RevokedTokens int       `json:"revoked_tokens"`
	TotalUsage    int       `json:"total_usage"`
	LastUsedAt    time.Time `json:"last_used_at,omitempty"`
}

// TokenType constants for token format.
const (
	// TokenPrefix is the prefix for all Cartographus PATs.
	// Format: carto_pat_<base64-encoded-id>_<random-secret>
	TokenPrefixConst = "carto_pat_"
)
