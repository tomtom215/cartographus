// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
Package auth provides authentication, authorization, and security middleware.

This package implements JWT and Basic Authentication, rate limiting, CORS, and
security headers for the Cartographus application. It serves as the security
layer between incoming HTTP requests and the API handlers.

Key Components:

  - JWTManager: Token generation and validation using HMAC-SHA256
  - BasicAuthManager: HTTP Basic Authentication with bcrypt password hashing
  - Middleware: HTTP middleware for authentication, rate limiting, and CORS
  - RateLimiter: Token bucket rate limiter (100 req/min per IP)
  - Security Headers: CSP, HSTS, X-Frame-Options, etc.

Authentication Modes:

The application supports two authentication modes (configured via AUTH_MODE):

1. JWT Mode (default):
  - Token-based authentication with configurable expiry (default: 24h)
  - Tokens stored in HTTP-only cookies for XSS protection
  - Refresh token support (future enhancement)

2. Basic Auth Mode:
  - Username/password authentication with bcrypt hashing
  - Password requirements: minimum 8 characters, complexity rules
  - Suitable for simple deployments without frontend complexity

Usage Example - JWT:

	import (
	    "github.com/tomtom215/cartographus/internal/auth"
	    "github.com/tomtom215/cartographus/internal/config"
	)

	// Create JWT manager
	jwtManager := auth.NewJWTManager(
	    config.JWTSecret,        // Min 32 characters
	    config.AdminUsername,
	    config.AdminPasswordHash,
	    24 * time.Hour,          // Token expiry
	)

	// Generate token
	token, err := jwtManager.GenerateToken("admin", "admin")
	if err != nil {
	    log.Fatal(err)
	}

	// Validate token
	claims, err := jwtManager.ValidateToken(token)

Usage Example - Middleware:

	// Create middleware
	middleware := auth.NewMiddleware(
	    jwtManager,
	    basicAuthManager,
	    "jwt",               // Auth mode
	    100,                 // Requests per window
	    time.Minute,         // Window duration
	    false,               // Rate limit disabled?
	    []string{"*"},       // CORS origins
	    []string{},          // Trusted proxies
	)

	// Protect endpoint
	http.HandleFunc("/api/v1/protected",
	    middleware.CORS(
	        middleware.RateLimit(
	            middleware.Authenticate(handler),
	        ),
	    ),
	)

Security Features:

  - Password Hashing: bcrypt with cost 10 (60ms per hash)
  - Token Signing: HMAC-SHA256 with 256-bit secret
  - Rate Limiting: Token bucket algorithm (100 req/min per IP, configurable)
  - CORS: Configurable origins with credentials support
  - CSP: Nonce-based Content Security Policy (v1.12)
  - Security Headers: HSTS, X-Frame-Options, X-Content-Type-Options
  - IP Extraction: X-Forwarded-For with trusted proxy validation

Rate Limiting:

The rate limiter uses a token bucket algorithm with automatic cleanup:
  - Default: 100 requests per minute per IP address
  - Cleanup interval: Every 5 minutes
  - Response: HTTP 429 Too Many Requests with Retry-After header
  - Bypass: Trusted proxy IPs are exempt from rate limiting

Thread Safety:

All components are thread-safe:
  - RateLimiter uses sync.RWMutex for concurrent access
  - Middleware is stateless and goroutine-safe
  - JWTManager and BasicAuthManager are read-only after initialization

Performance:

  - JWT validation: <1ms (HMAC verification)
  - bcrypt hashing: ~60ms (cost 10)
  - Rate limiter: O(1) lookup with periodic cleanup
  - Memory footprint: ~1KB per tracked IP address

Security Best Practices:

  - JWT secrets must be ≥32 characters (enforced)
  - Passwords must be ≥8 characters (enforced)
  - Tokens expire after 24 hours (configurable)
  - HTTP-only cookies prevent XSS attacks
  - CSP headers prevent inline script execution
  - Rate limiting prevents brute force attacks

See Also:

  - internal/api: HTTP handlers protected by middleware
  - internal/config: Authentication configuration
  - internal/middleware: Additional middleware components
*/
package auth
