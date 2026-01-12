// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including Basic auth support.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"net/http"
	"strings"
)

// BasicAuthenticatorConfig holds configuration for the Basic authenticator.
type BasicAuthenticatorConfig struct {
	// DefaultRole is assigned to Basic-authenticated users who are not the admin user.
	// If empty, defaults to "viewer" for security (principle of least privilege).
	DefaultRole string

	// AdminUsername is the username that should receive admin role.
	// This user bypasses the default role and gets admin privileges directly.
	AdminUsername string
}

// BasicAuthenticator implements the Authenticator interface for HTTP Basic Authentication.
// It wraps the existing BasicAuthManager to provide a consistent interface.
//
// RBAC Integration:
// - The configured admin user (AdminUsername) receives "admin" role
// - All other users receive the DefaultRole (typically "viewer")
// - Users can be elevated via database role assignments (/api/admin/roles/assign)
// - The authorization service merges token roles + database roles
type BasicAuthenticator struct {
	manager       *BasicAuthManager
	defaultRole   string
	adminUsername string
}

// NewBasicAuthenticator creates a new Basic authenticator.
func NewBasicAuthenticator(manager *BasicAuthManager, config *BasicAuthenticatorConfig) *BasicAuthenticator {
	defaultRole := "viewer" // Default to viewer for security (least privilege)
	adminUsername := ""

	if config != nil {
		if config.DefaultRole != "" {
			defaultRole = config.DefaultRole
		}
		adminUsername = config.AdminUsername
	}

	return &BasicAuthenticator{
		manager:       manager,
		defaultRole:   defaultRole,
		adminUsername: adminUsername,
	}
}

// Authenticate extracts and validates Basic auth credentials from the request.
func (a *BasicAuthenticator) Authenticate(ctx context.Context, r *http.Request) (*AuthSubject, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, ErrNoCredentials
	}

	// Check for Basic scheme
	if !strings.HasPrefix(authHeader, "Basic ") {
		return nil, ErrNoCredentials
	}

	// Validate credentials using the manager
	username, err := a.manager.ValidateCredentials(authHeader)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Determine role based on username
	// Admin user gets admin role, all others get default role
	role := a.defaultRole
	if a.adminUsername != "" && username == a.adminUsername {
		role = "admin"
	}

	// Build AuthSubject
	subject := &AuthSubject{
		ID:         username,
		Username:   username,
		AuthMethod: AuthModeBasic,
		Issuer:     "local",
		Roles:      []string{role},
	}

	return subject, nil
}

// Name returns the authenticator name.
func (a *BasicAuthenticator) Name() string {
	return string(AuthModeBasic)
}

// Priority returns the authenticator priority (lower = higher priority).
// Basic auth has priority 25 (lowest, after JWT at 20).
func (a *BasicAuthenticator) Priority() int {
	return 25
}

// GetWWWAuthenticateHeader returns the WWW-Authenticate header value.
// This is used by middleware to send the proper 401 response.
func (a *BasicAuthenticator) GetWWWAuthenticateHeader() string {
	return a.manager.GetWWWAuthenticateHeader()
}
