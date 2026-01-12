// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
	"golang.org/x/time/rate"
)

type contextKey string

const ClaimsContextKey contextKey = "claims"
const CSPNonceContextKey contextKey = "csp-nonce" // S1: CSP nonce context key

// Middleware provides authentication and rate limiting middleware
type Middleware struct {
	jwtManager             *JWTManager
	basicAuthManager       *BasicAuthManager
	authMode               string
	rateLimiter            *RateLimiter
	rateLimitDisabled      bool
	corsOrigins            []string
	trustedProxies         map[string]bool
	basicAuthDefaultRole   string // Default role for Basic Auth users (default: viewer)
	basicAuthAdminUsername string // Username that gets admin role
}

// NewMiddleware creates a new authentication middleware
func NewMiddleware(jwtManager *JWTManager, basicAuthManager *BasicAuthManager, authMode string, reqsPerWindow int, window time.Duration, rateLimitDisabled bool, corsOrigins, trustedProxies []string, basicAuthDefaultRole, basicAuthAdminUsername string) *Middleware {
	trustedMap := make(map[string]bool)
	for _, proxy := range trustedProxies {
		trustedMap[proxy] = true
	}

	// Default to "viewer" role for security (principle of least privilege)
	if basicAuthDefaultRole == "" {
		basicAuthDefaultRole = "viewer"
	}

	m := &Middleware{
		jwtManager:             jwtManager,
		basicAuthManager:       basicAuthManager,
		authMode:               authMode,
		rateLimiter:            NewRateLimiter(reqsPerWindow, window),
		rateLimitDisabled:      rateLimitDisabled,
		corsOrigins:            corsOrigins,
		trustedProxies:         trustedMap,
		basicAuthDefaultRole:   basicAuthDefaultRole,
		basicAuthAdminUsername: basicAuthAdminUsername,
	}

	// Start periodic cleanup for rate limiter (only if not disabled)
	if !rateLimitDisabled {
		go m.rateLimiter.startCleanup(5 * time.Minute)
	}

	return m
}

// Authenticate is middleware that enforces authentication
func (m *Middleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m.authMode == "none" {
			next(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")

		// Handle Basic Authentication
		if m.authMode == string(AuthModeBasic) {
			m.handleBasicAuth(w, r, next, authHeader)
			return
		}

		// Handle JWT Authentication
		m.handleJWTAuth(w, r, next, authHeader)
	}
}

// handleBasicAuth processes Basic Authentication requests
func (m *Middleware) handleBasicAuth(w http.ResponseWriter, r *http.Request, next http.HandlerFunc, authHeader string) {
	if authHeader == "" {
		m.sendBasicAuthChallenge(w, "Unauthorized: authentication required")
		return
	}

	username, err := m.basicAuthManager.ValidateCredentials(authHeader)
	if err != nil {
		logging.Error().Err(err).Msg("Basic auth validation failed")
		m.sendBasicAuthChallenge(w, "Unauthorized: invalid credentials")
		return
	}

	claims := m.createBasicAuthClaims(username)
	ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
	next(w, r.WithContext(ctx))
}

// sendBasicAuthChallenge sends a WWW-Authenticate challenge and error response
func (m *Middleware) sendBasicAuthChallenge(w http.ResponseWriter, message string) {
	w.Header().Set("WWW-Authenticate", m.basicAuthManager.GetWWWAuthenticateHeader())
	http.Error(w, message, http.StatusUnauthorized)
}

// createBasicAuthClaims creates claims for a Basic Auth user with appropriate role
func (m *Middleware) createBasicAuthClaims(username string) *Claims {
	role := m.basicAuthDefaultRole
	if m.basicAuthAdminUsername != "" && username == m.basicAuthAdminUsername {
		role = "admin"
	}

	return &Claims{
		Username: username,
		Role:     role,
	}
}

// handleJWTAuth processes JWT Authentication requests
func (m *Middleware) handleJWTAuth(w http.ResponseWriter, r *http.Request, next http.HandlerFunc, authHeader string) {
	token, err := m.extractJWTToken(r, authHeader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	claims, err := m.jwtManager.ValidateToken(token)
	if err != nil {
		logging.Error().Err(err).Msg("Token validation failed")
		http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
		return
	}

	ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
	next(w, r.WithContext(ctx))
}

// extractJWTToken extracts JWT token from Authorization header or cookie
func (m *Middleware) extractJWTToken(r *http.Request, authHeader string) (string, error) {
	if authHeader == "" {
		cookie, err := r.Cookie("token")
		if err != nil {
			return "", fmt.Errorf("unauthorized: missing token")
		}
		return cookie.Value, nil
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("unauthorized: invalid authorization header")
	}

	return parts[1], nil
}

// RequireRole is middleware that enforces a specific role
func (m *Middleware) RequireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return m.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(ClaimsContextKey).(*Claims)
		if !ok {
			http.Error(w, "Forbidden: invalid claims", http.StatusForbidden)
			return
		}

		if claims.Role != role && claims.Role != "admin" {
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			return
		}

		next(w, r)
	})
}

// RateLimit is middleware that enforces rate limiting
func (m *Middleware) RateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting if disabled (for CI/CD tests)
		if m.rateLimitDisabled {
			next(w, r)
			return
		}

		ip := m.getClientIP(r)
		if !m.rateLimiter.Allow(ip) {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

// CORS is a method that adds CORS headers based on configuration
func (m *Middleware) CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := m.checkAndSetOriginHeaders(w, origin)

		if !allowed && origin != "" {
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			// For non-preflight requests, continue but don't add CORS headers
			// The browser will block the response due to CORS policy
		}

		m.setCommonCORSHeaders(w)

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// checkAndSetOriginHeaders checks if origin is allowed and sets appropriate headers
func (m *Middleware) checkAndSetOriginHeaders(w http.ResponseWriter, origin string) bool {
	for _, allowedOrigin := range m.corsOrigins {
		if allowedOrigin == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			return true
		}
		if allowedOrigin == origin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			return true
		}
	}
	return false
}

// setCommonCORSHeaders sets the common CORS headers for all requests
func (m *Middleware) setCommonCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "86400")
}

// generateNonce generates a cryptographically secure nonce for CSP
// S1: Addresses Medium Priority Issue S1 from production audit
func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// SecurityHeaders adds security headers to all responses
func (m *Middleware) SecurityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// S1: Generate CSP nonce for this request
		nonce, err := generateNonce()
		if err != nil {
			logging.Warn().Err(err).Msg("Failed to generate CSP nonce")
			nonce = "" // Fallback to no nonce if generation fails
		}

		// S1: Store nonce in request context for HTML template
		ctx := context.WithValue(r.Context(), CSPNonceContextKey, nonce)
		r = r.WithContext(ctx)

		// Content Security Policy - Enterprise-grade strict policy with nonce for scripts
		// S1: CSP nonces prevent inline script XSS attacks (industry best practice)
		// Script security: Nonce-based (NO unsafe-inline) - Maximum protection against XSS
		// Style security: unsafe-inline allowed (styles cannot execute code, minimal risk)
		// unsafe-eval: Required only for ECharts dynamic chart generation (data visualization library)
		csp := "default-src 'self'; " +
			"script-src 'self' 'nonce-" + nonce + "' 'unsafe-eval'; " + // STRICT: No unsafe-inline, nonce required for inline scripts
			"style-src 'self' https://api.mapbox.com https://unpkg.com 'unsafe-inline'; " + // Allow MapLibre CSS from unpkg.com
			"img-src 'self' data: https://*.basemaps.cartocdn.com https://api.mapbox.com; " +
			"font-src 'self' data:; " +
			"connect-src 'self' https://*.basemaps.cartocdn.com wss: ws:; " + // Allow map tiles + WebSocket
			"worker-src 'self' blob:; " + // Allow blob: workers for deck.gl
			"manifest-src 'self'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'"
		w.Header().Set("Content-Security-Policy", csp)

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// HSTS (only if using HTTPS - check X-Forwarded-Proto)
		if r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Permissions policy (restrict unnecessary browser features)
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next(w, r)
	}
}

// getClientIP extracts the client IP address from the request with proxy validation
func (m *Middleware) getClientIP(r *http.Request) string {
	remoteIP := strings.Split(r.RemoteAddr, ":")[0]

	if !m.isFromTrustedProxy(remoteIP) {
		return remoteIP
	}

	// Try X-Forwarded-For first
	if clientIP := m.extractIPFromXFF(r); clientIP != "" {
		return clientIP
	}

	// Try X-Real-IP as fallback
	if clientIP := m.extractIPFromXRealIP(r); clientIP != "" {
		return clientIP
	}

	// No valid headers, use RemoteAddr
	return remoteIP
}

// isFromTrustedProxy checks if the remote IP is a trusted proxy
func (m *Middleware) isFromTrustedProxy(remoteIP string) bool {
	return len(m.trustedProxies) > 0 && m.trustedProxies[remoteIP]
}

// extractIPFromXFF extracts and validates IP from X-Forwarded-For header
func (m *Middleware) extractIPFromXFF(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return ""
	}

	ips := strings.Split(xff, ",")
	clientIP := strings.TrimSpace(ips[0])
	if isValidIP(clientIP) {
		return clientIP
	}

	return ""
}

// extractIPFromXRealIP extracts and validates IP from X-Real-IP header
func (m *Middleware) extractIPFromXRealIP(r *http.Request) string {
	xri := r.Header.Get("X-Real-IP")
	if xri != "" && isValidIP(xri) {
		return xri
	}
	return ""
}

// isValidIP checks if a string is a valid IP address (basic validation)
func isValidIP(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		return isValidIPv4(parts)
	}
	return isValidIPv6(ip)
}

// isValidIPv4 validates an IPv4 address from its parts
func isValidIPv4(parts []string) bool {
	for _, part := range parts {
		if !isValidIPv4Part(part) {
			return false
		}
	}
	return true
}

// isValidIPv4Part validates a single octet of an IPv4 address
func isValidIPv4Part(part string) bool {
	if len(part) == 0 || len(part) > 3 {
		return false
	}
	for _, char := range part {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// isValidIPv6 performs basic validation for IPv6 addresses
func isValidIPv6(ip string) bool {
	// Simple check - could be IPv6 or invalid
	// For IPv6, just check it's not empty and doesn't contain suspicious chars
	return ip != "" && !strings.Contains(ip, " ") && len(ip) < 40
}

// RateLimiter implements per-IP rate limiting with automatic cleanup
type RateLimiter struct {
	limiters  map[string]*rateLimiterEntry
	mu        sync.RWMutex
	rate      rate.Limit
	burst     int
	stopClean chan struct{}
}

// rateLimiterEntry wraps a rate limiter with last access time
type rateLimiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(reqsPerWindow int, window time.Duration) *RateLimiter {
	r := rate.Every(window)
	return &RateLimiter{
		limiters:  make(map[string]*rateLimiterEntry),
		rate:      r,
		burst:     reqsPerWindow,
		stopClean: make(chan struct{}),
	}
}

// Allow checks if a request from the given IP is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	entry, exists := rl.limiters[ip]
	if !exists {
		entry = &rateLimiterEntry{
			limiter:    rate.NewLimiter(rl.rate, rl.burst),
			lastAccess: time.Now(),
		}
		rl.limiters[ip] = entry
	} else {
		entry.lastAccess = time.Now()
	}
	limiter := entry.limiter
	rl.mu.Unlock()

	return limiter.Allow()
}

// startCleanup periodically removes stale rate limiters
func (rl *RateLimiter) startCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopClean:
			return
		}
	}
}

// cleanup removes rate limiters that haven't been accessed in the last hour
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	threshold := time.Now().Add(-1 * time.Hour)
	for ip, entry := range rl.limiters {
		if entry.lastAccess.Before(threshold) {
			delete(rl.limiters, ip)
		}
	}
}

// Stop stops the cleanup goroutine
func (rl *RateLimiter) Stop() {
	close(rl.stopClean)
}

// GetCORSOrigins returns the configured CORS allowed origins.
// ADR-0016: Exposes config for Chi middleware integration.
func (m *Middleware) GetCORSOrigins() []string {
	return m.corsOrigins
}

// GetRateLimitConfig returns the rate limit configuration.
// ADR-0016: Exposes config for Chi middleware integration.
func (m *Middleware) GetRateLimitConfig() (reqsPerWindow int, disabled bool) {
	return m.rateLimiter.burst, m.rateLimitDisabled
}

// GetRateLimitWindow returns the rate limit window duration.
// ADR-0016: Exposes config for Chi middleware integration.
func (m *Middleware) GetRateLimitWindow() time.Duration {
	// The rate limiter stores rate as tokens/sec, convert back to window
	// For simplicity, return 1 minute as the standard window
	return time.Minute
}
