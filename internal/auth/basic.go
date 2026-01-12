// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// BasicAuthManager handles HTTP Basic Authentication with secure password verification
type BasicAuthManager struct {
	username     string
	passwordHash []byte // bcrypt hash of password
}

// NewBasicAuthManager creates a new Basic Auth manager with bcrypt-hashed password
// The password is hashed at initialization to avoid hashing on every request
func NewBasicAuthManager(username, password string) (*BasicAuthManager, error) {
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if password == "" {
		return nil, fmt.Errorf("password is required")
	}
	if len(password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters for security")
	}

	// Hash the password using bcrypt (cost factor 12 for good security/performance balance)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	return &BasicAuthManager{
		username:     username,
		passwordHash: hash,
	}, nil
}

// ValidateCredentials validates HTTP Basic Auth credentials
// Uses constant-time comparison to prevent timing attacks
// Returns username if valid, error if invalid
func (m *BasicAuthManager) ValidateCredentials(authHeader string) (string, error) {
	// Check for "Basic " prefix
	if !strings.HasPrefix(authHeader, "Basic ") {
		return "", fmt.Errorf("invalid authorization header format")
	}

	// Decode Base64 encoded credentials
	encodedCredentials := strings.TrimPrefix(authHeader, "Basic ")
	credentials, err := base64.StdEncoding.DecodeString(encodedCredentials)
	if err != nil {
		return "", fmt.Errorf("failed to decode credentials")
	}

	// Split username:password
	parts := strings.SplitN(string(credentials), ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid credentials format")
	}

	providedUsername := parts[0]
	providedPassword := parts[1]

	// Validate credentials using constant-time comparison
	if !m.validateUsernamePassword(providedUsername, providedPassword) {
		return "", fmt.Errorf("invalid username or password")
	}

	return providedUsername, nil
}

// validateUsernamePassword performs constant-time comparison of credentials
// This prevents timing attacks by ensuring comparison takes the same time
// regardless of which character differs
func (m *BasicAuthManager) validateUsernamePassword(username, password string) bool {
	// Use constant-time comparison for username to prevent timing attacks
	usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(m.username)) == 1

	// Use bcrypt's CompareHashAndPassword which is timing-safe
	// Note: bcrypt.CompareHashAndPassword is already timing-safe by design
	passwordMatch := bcrypt.CompareHashAndPassword(m.passwordHash, []byte(password)) == nil

	// Both must match; using & instead of && to prevent short-circuit evaluation
	// This ensures both comparisons happen regardless of username result
	return usernameMatch && passwordMatch
}

// GetWWWAuthenticateHeader returns the WWW-Authenticate header value
// This is required by HTTP spec to be sent with 401 Unauthorized responses
func (m *BasicAuthManager) GetWWWAuthenticateHeader() string {
	return `Basic realm="Cartographus", charset="UTF-8"`
}
