// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including JWT support.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// JWTAuthenticator implements the Authenticator interface for JWT authentication.
// It wraps the existing JWTManager to provide a consistent interface.
type JWTAuthenticator struct {
	manager     *JWTManager
	tokenCookie string
}

// NewJWTAuthenticator creates a new JWT authenticator.
func NewJWTAuthenticator(manager *JWTManager) *JWTAuthenticator {
	return &JWTAuthenticator{
		manager:     manager,
		tokenCookie: "token",
	}
}

// Authenticate extracts and validates the JWT from the request.
func (a *JWTAuthenticator) Authenticate(ctx context.Context, r *http.Request) (*AuthSubject, error) {
	// Extract token from Authorization header or cookie
	tokenStr := a.extractToken(r)
	if tokenStr == "" {
		return nil, ErrNoCredentials
	}

	// Validate the token
	claims, err := a.manager.ValidateToken(tokenStr)
	if err != nil {
		// Check for expiration
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredCredentials
		}
		return nil, ErrInvalidCredentials
	}

	// Convert Claims to AuthSubject
	subject := AuthSubjectFromClaims(claims)
	return subject, nil
}

// Name returns the authenticator name.
func (a *JWTAuthenticator) Name() string {
	return string(AuthModeJWT)
}

// Priority returns the authenticator priority (lower = higher priority).
// JWT has priority 20, between Plex (15) and Basic (25).
func (a *JWTAuthenticator) Priority() int {
	return 20
}

// extractToken extracts the bearer token from Authorization header or cookie.
func (a *JWTAuthenticator) extractToken(r *http.Request) string {
	// Check Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			token := strings.TrimSpace(parts[1])
			if token != "" {
				return token
			}
		}
	}

	// Fall back to cookie
	cookie, err := r.Cookie(a.tokenCookie)
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}
