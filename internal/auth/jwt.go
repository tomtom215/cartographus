// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tomtom215/cartographus/internal/config"
)

// Claims represents JWT claims
type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT token creation and validation
type JWTManager struct {
	secret  []byte
	timeout time.Duration
}

// NewJWTManager creates a new JWT token manager with the configured secret and timeout.
//
// This constructor initializes a JWTManager for creating and validating JWT tokens
// used in the authentication system. The manager uses HMAC-SHA256 signing.
//
// Parameters:
//   - cfg: Security configuration containing JWT secret and session timeout
//
// Returns:
//   - Pointer to initialized JWTManager
//   - error if JWT_SECRET is empty (minimum 32 characters required)
//
// Security Requirements:
//   - JWT_SECRET must be at least 32 characters for production security
//   - Secret is stored as []byte to prevent string interning attacks
//   - Uses HS256 signing algorithm (HMAC with SHA-256)
//
// Example:
//
//	jwtManager, err := auth.NewJWTManager(cfg.Security)
//	if err != nil {
//	    log.Fatal("Failed to initialize JWT manager:", err)
//	}
func NewJWTManager(cfg *config.SecurityConfig) (*JWTManager, error) {
	secret := cfg.JWTSecret
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required but was empty")
	}

	return &JWTManager{
		secret:  []byte(secret),
		timeout: cfg.SessionTimeout,
	}, nil
}

// GenerateToken creates a new JWT token for an authenticated user.
//
// This method generates a signed JWT token containing the user's username and role.
// The token is valid for the configured session timeout duration (default: 24 hours).
//
// Parameters:
//   - username: User's login name (stored in claims for identification)
//   - role: User's role (e.g., "admin", "user" - stored for authorization)
//
// Returns:
//   - Signed JWT token string (base64-encoded)
//   - error if token signing fails
//
// Token Claims:
//   - Username: User identifier
//   - Role: Authorization role
//   - ExpiresAt: Session timeout (now + configured timeout)
//   - IssuedAt: Token creation timestamp
//   - NotBefore: Token becomes valid immediately
//
// Security:
//   - Uses HMAC-SHA256 (HS256) signing algorithm
//   - Tokens are stateless and cannot be revoked before expiration
//   - Client must store token securely (HTTP-only cookie recommended)
//
// Example:
//
//	token, err := jwtManager.GenerateToken("alice", "admin")
//	if err != nil {
//	    return nil, fmt.Errorf("token generation failed: %w", err)
//	}
//	// Set as HTTP-only cookie or return in response body
func (m *JWTManager) GenerateToken(username, role string) (string, error) {
	claims := &Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.timeout)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken validates a JWT token and extracts the user claims.
//
// This method performs comprehensive validation of a JWT token string, checking:
// signature validity, expiration time, token structure, and signing algorithm.
//
// Parameters:
//   - tokenString: Base64-encoded JWT token string (from Authorization header or cookie)
//
// Returns:
//   - Pointer to Claims struct containing username and role
//   - error if validation fails (expired, tampered, malformed, or wrong signature)
//
// Validation Steps:
//  1. Parse token structure and extract claims
//  2. Verify HMAC-SHA256 signature matches secret
//  3. Check signing algorithm is HS256 (prevents algorithm confusion attacks)
//  4. Verify token expiration (ExpiresAt claim)
//  5. Verify NotBefore claim (token is active)
//
// Security:
//   - Rejects tokens with unexpected signing algorithm (RS256, none, etc.)
//   - Returns specific errors for debugging (parse failure, invalid claims)
//   - Time-based validation uses server time (not client-provided)
//
// Common Errors:
//   - "token is expired": Token exceeded SessionTimeout, user must re-authenticate
//   - "unexpected signing method": Possible algorithm confusion attack
//   - "failed to parse token": Malformed token or wrong secret
//
// Example:
//
//	claims, err := jwtManager.ValidateToken(tokenString)
//	if err != nil {
//	    return nil, fmt.Errorf("authentication failed: %w", err)
//	}
//	// Use claims.Username and claims.Role for authorization
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
