// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// AuthMode represents the authentication strategy.
type AuthMode string

const (
	// AuthModeNone disables authentication
	AuthModeNone AuthMode = "none"

	// AuthModeBasic uses HTTP Basic Authentication
	AuthModeBasic AuthMode = "basic"

	// AuthModeJWT uses JWT Bearer tokens
	AuthModeJWT AuthMode = "jwt"

	// AuthModeOIDC uses OpenID Connect
	AuthModeOIDC AuthMode = "oidc"

	// AuthModePlex uses Plex OAuth 2.0
	AuthModePlex AuthMode = "plex"

	// AuthModeMulti tries multiple auth methods in order
	// Order: OIDC -> Plex -> JWT -> Basic
	AuthModeMulti AuthMode = "multi"
)

// ParseAuthMode converts a string to AuthMode.
func ParseAuthMode(s string) (AuthMode, error) {
	switch s {
	case "none", "":
		return AuthModeNone, nil
	case "basic":
		return AuthModeBasic, nil
	case "jwt":
		return AuthModeJWT, nil
	case string(AuthModeOIDC):
		return AuthModeOIDC, nil
	case "plex":
		return AuthModePlex, nil
	case "multi":
		return AuthModeMulti, nil
	default:
		return "", errors.New("invalid auth mode: " + s)
	}
}

// String returns the string representation of AuthMode.
func (m AuthMode) String() string {
	return string(m)
}

// Standard authentication errors
var (
	// ErrNoCredentials indicates no credentials were provided.
	ErrNoCredentials = errors.New("no credentials provided")

	// ErrInvalidCredentials indicates credentials were invalid.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrExpiredCredentials indicates credentials have expired.
	ErrExpiredCredentials = errors.New("credentials expired")

	// ErrAuthenticatorUnavailable indicates the auth provider is unreachable.
	ErrAuthenticatorUnavailable = errors.New("authenticator unavailable")
)

// Authenticator defines the interface for authentication providers.
type Authenticator interface {
	// Authenticate extracts and validates credentials from the request.
	// Returns AuthSubject on success, error on failure.
	Authenticate(ctx context.Context, r *http.Request) (*AuthSubject, error)

	// Name returns the authenticator's name for logging.
	Name() string

	// Priority returns the authenticator's priority for multi-mode.
	// Lower values are tried first.
	Priority() int
}

// AuthSubject represents an authenticated user/entity.
// This struct normalizes claims from different auth sources (JWT, OIDC, Basic, Plex).
type AuthSubject struct {
	// ID is the unique identifier for this subject.
	// For OIDC: the 'sub' claim
	// For JWT: the 'sub' claim or username
	// For Basic: the username
	// For Plex: the Plex user ID
	ID string `json:"id"`

	// Username is the human-readable username.
	// For OIDC: 'preferred_username' or 'name' claim
	// For Basic: the username
	// For Plex: Plex username
	Username string `json:"username"`

	// Email is the subject's email address (if available).
	// For OIDC: 'email' claim
	Email string `json:"email,omitempty"`

	// EmailVerified indicates if the email has been verified.
	// For OIDC: 'email_verified' claim
	EmailVerified bool `json:"email_verified,omitempty"`

	// Roles contains the subject's assigned roles.
	// Used by Casbin for authorization.
	Roles []string `json:"roles,omitempty"`

	// Groups contains group memberships (OIDC 'groups' claim).
	Groups []string `json:"groups,omitempty"`

	// Issuer identifies the auth source.
	// For OIDC: the 'iss' claim
	// For JWT: the 'iss' claim or "local"
	// For Basic: "local"
	// For Plex: "plex.tv"
	Issuer string `json:"issuer,omitempty"`

	// AuthMethod indicates how the subject was authenticated.
	AuthMethod AuthMode `json:"auth_method"`

	// IssuedAt is when the authentication token was issued.
	IssuedAt int64 `json:"issued_at,omitempty"`

	// ExpiresAt is when the authentication expires.
	ExpiresAt int64 `json:"expires_at,omitempty"`

	// SessionID is the unique session identifier (for OIDC/Plex).
	SessionID string `json:"session_id,omitempty"`

	// RawClaims contains the original claims (for debugging/extensibility).
	// Not exposed in JSON by default.
	RawClaims map[string]interface{} `json:"-"`

	// Provider is the authentication provider name (e.g., "oidc", "plex", "jwt", "basic").
	// This is a string representation of AuthMethod for session storage.
	Provider string `json:"provider,omitempty"`

	// Metadata contains additional key-value data for the session.
	// Can be used to store provider-specific data like tokens, refresh tokens, etc.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// HasRole checks if the subject has a specific role.
func (s *AuthSubject) HasRole(role string) bool {
	if role == "" {
		return false
	}
	for _, r := range s.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the subject has any of the specified roles.
func (s *AuthSubject) HasAnyRole(roles ...string) bool {
	if len(roles) == 0 {
		return false
	}
	for _, role := range roles {
		if s.HasRole(role) {
			return true
		}
	}
	return false
}

// IsExpired checks if the authentication has expired.
func (s *AuthSubject) IsExpired() bool {
	if s.ExpiresAt == 0 {
		return false // No expiry set
	}
	return time.Now().Unix() > s.ExpiresAt
}

// AuthSubjectFromClaims creates an AuthSubject from existing JWT Claims.
// This provides backwards compatibility with the existing auth system.
func AuthSubjectFromClaims(claims *Claims) *AuthSubject {
	if claims == nil {
		return nil
	}

	subject := &AuthSubject{
		ID:         claims.Username, // Use username as ID for JWT
		Username:   claims.Username,
		AuthMethod: AuthModeJWT,
		Issuer:     "local",
	}

	// Add role if present
	if claims.Role != "" {
		subject.Roles = []string{claims.Role}
	}

	// Extract timestamps from RegisteredClaims if available
	if claims.ExpiresAt != nil {
		subject.ExpiresAt = claims.ExpiresAt.Unix()
	}
	if claims.IssuedAt != nil {
		subject.IssuedAt = claims.IssuedAt.Unix()
	}

	return subject
}

// ToClaims converts an AuthSubject back to Claims for backwards compatibility.
// This allows the new auth system to work with existing handlers that expect Claims.
func (s *AuthSubject) ToClaims() *Claims {
	if s == nil {
		return nil
	}

	claims := &Claims{
		Username: s.Username,
	}

	// Use first role as the primary role
	if len(s.Roles) > 0 {
		claims.Role = s.Roles[0]
	}

	return claims
}
