// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication middleware with support for multiple auth modes.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
)

// AuthSubjectContextKey is the context key for AuthSubject.
const AuthSubjectContextKey contextKey = "auth_subject"

// MiddlewareV2Config holds configuration for MiddlewareV2.
type MiddlewareV2Config struct {
	// AuthMode specifies the authentication mode.
	AuthMode AuthMode

	// JWTManager for JWT authentication.
	JWTManager *JWTManager

	// BasicAuthManager for Basic authentication.
	BasicAuthManager *BasicAuthManager

	// BasicAuthDefaultRole is the role assigned to Basic auth users (except admin).
	BasicAuthDefaultRole string

	// BasicAuthAdminUsername is the username that receives admin role.
	// This user bypasses BasicAuthDefaultRole and gets admin privileges.
	BasicAuthAdminUsername string

	// OIDCAuthenticator for OIDC authentication.
	OIDCAuthenticator *OIDCAuthenticator

	// PlexAuthenticator for Plex authentication.
	PlexAuthenticator *PlexAuthenticator

	// ReqsPerWindow is the rate limit requests per window.
	ReqsPerWindow int

	// Window is the rate limit window duration.
	Window time.Duration

	// RateLimitDisabled disables rate limiting (for testing).
	RateLimitDisabled bool

	// CORSOrigins is the list of allowed CORS origins.
	CORSOrigins []string

	// TrustedProxies is the list of trusted proxy IPs.
	TrustedProxies []string
}

// MiddlewareV2 provides authentication middleware using the Authenticator interface.
// It supports multiple authentication modes: none, basic, jwt, oidc, plex, multi.
type MiddlewareV2 struct {
	authenticator     Authenticator
	authMode          AuthMode
	rateLimiter       *RateLimiter
	rateLimitDisabled bool
	corsOrigins       []string
	trustedProxies    map[string]bool

	// For backwards compatibility with existing handlers
	jwtManager       *JWTManager
	basicAuthManager *BasicAuthManager
}

// NewMiddlewareV2 creates a new authentication middleware with the given configuration.
func NewMiddlewareV2(cfg *MiddlewareV2Config) (*MiddlewareV2, error) {
	trustedMap := make(map[string]bool)
	for _, proxy := range cfg.TrustedProxies {
		trustedMap[proxy] = true
	}

	m := &MiddlewareV2{
		authMode:          cfg.AuthMode,
		rateLimiter:       NewRateLimiter(cfg.ReqsPerWindow, cfg.Window),
		rateLimitDisabled: cfg.RateLimitDisabled,
		corsOrigins:       cfg.CORSOrigins,
		trustedProxies:    trustedMap,
		jwtManager:        cfg.JWTManager,
		basicAuthManager:  cfg.BasicAuthManager,
	}

	// Build authenticator based on mode
	authenticator, err := m.buildAuthenticator(cfg)
	if err != nil {
		return nil, err
	}
	m.authenticator = authenticator

	// Start periodic cleanup for rate limiter (only if not disabled)
	if !cfg.RateLimitDisabled {
		go m.rateLimiter.startCleanup(5 * time.Minute)
	}

	return m, nil
}

// buildAuthenticator creates the appropriate authenticator based on the auth mode.
func (m *MiddlewareV2) buildAuthenticator(cfg *MiddlewareV2Config) (Authenticator, error) {
	switch cfg.AuthMode {
	case AuthModeNone:
		return nil, nil

	case AuthModeJWT:
		if cfg.JWTManager == nil {
			return nil, errors.New("JWT manager required for jwt auth mode")
		}
		return NewJWTAuthenticator(cfg.JWTManager), nil

	case AuthModeBasic:
		if cfg.BasicAuthManager == nil {
			return nil, errors.New("Basic auth manager required for basic auth mode")
		}
		basicConfig := &BasicAuthenticatorConfig{
			DefaultRole:   cfg.BasicAuthDefaultRole,
			AdminUsername: cfg.BasicAuthAdminUsername,
		}
		return NewBasicAuthenticator(cfg.BasicAuthManager, basicConfig), nil

	case AuthModeOIDC:
		if cfg.OIDCAuthenticator == nil {
			return nil, errors.New("OIDC authenticator required for oidc auth mode")
		}
		return cfg.OIDCAuthenticator, nil

	case AuthModePlex:
		if cfg.PlexAuthenticator == nil {
			return nil, errors.New("Plex authenticator required for plex auth mode")
		}
		return cfg.PlexAuthenticator, nil

	case AuthModeMulti:
		return m.buildMultiAuthenticator(cfg)

	default:
		return nil, errors.New("unsupported auth mode: " + string(cfg.AuthMode))
	}
}

// buildMultiAuthenticator creates a multi-authenticator from available authenticators.
func (m *MiddlewareV2) buildMultiAuthenticator(cfg *MiddlewareV2Config) (*MultiAuthenticator, error) {
	var authenticators []Authenticator

	// Add authenticators in priority order (priority is handled by MultiAuthenticator)
	if cfg.OIDCAuthenticator != nil {
		authenticators = append(authenticators, cfg.OIDCAuthenticator)
	}

	if cfg.PlexAuthenticator != nil {
		authenticators = append(authenticators, cfg.PlexAuthenticator)
	}

	if cfg.JWTManager != nil {
		authenticators = append(authenticators, NewJWTAuthenticator(cfg.JWTManager))
	}

	if cfg.BasicAuthManager != nil {
		basicConfig := &BasicAuthenticatorConfig{
			DefaultRole:   cfg.BasicAuthDefaultRole,
			AdminUsername: cfg.BasicAuthAdminUsername,
		}
		authenticators = append(authenticators, NewBasicAuthenticator(cfg.BasicAuthManager, basicConfig))
	}

	if len(authenticators) == 0 {
		return nil, errors.New("multi auth mode requires at least one authenticator")
	}

	return NewMultiAuthenticator(authenticators...), nil
}

// AuthenticateV2 is middleware that enforces authentication using the Authenticator interface.
func (m *MiddlewareV2) AuthenticateV2(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// No authentication required
		if m.authMode == AuthModeNone {
			next(w, r)
			return
		}

		// Authenticate using the configured authenticator
		subject, err := m.authenticator.Authenticate(r.Context(), r)
		if err != nil {
			m.handleAuthError(w, err)
			return
		}

		// Add AuthSubject to context
		ctx := context.WithValue(r.Context(), AuthSubjectContextKey, subject)

		// Add Claims to context for backwards compatibility
		claims := subject.ToClaims()
		ctx = context.WithValue(ctx, ClaimsContextKey, claims)

		next(w, r.WithContext(ctx))
	}
}

// handleAuthError sends the appropriate HTTP error response for auth errors.
func (m *MiddlewareV2) handleAuthError(w http.ResponseWriter, err error) {
	// Log the error
	logging.Error().Err(err).Msg("Authentication failed")

	// Determine error type and respond accordingly
	switch {
	case errors.Is(err, ErrNoCredentials):
		// If using Basic auth, send WWW-Authenticate header
		if basicAuth, ok := m.authenticator.(*BasicAuthenticator); ok {
			w.Header().Set("WWW-Authenticate", basicAuth.GetWWWAuthenticateHeader())
		} else if m.basicAuthManager != nil {
			w.Header().Set("WWW-Authenticate", m.basicAuthManager.GetWWWAuthenticateHeader())
		}
		http.Error(w, "Unauthorized: authentication required", http.StatusUnauthorized)

	case errors.Is(err, ErrInvalidCredentials):
		http.Error(w, "Unauthorized: invalid credentials", http.StatusUnauthorized)

	case errors.Is(err, ErrExpiredCredentials):
		http.Error(w, "Unauthorized: credentials expired", http.StatusUnauthorized)

	case errors.Is(err, ErrAuthenticatorUnavailable):
		http.Error(w, "Service unavailable: authentication service unavailable", http.StatusServiceUnavailable)

	default:
		http.Error(w, "Unauthorized: authentication failed", http.StatusUnauthorized)
	}
}

// RequireRoleV2 is middleware that enforces a specific role using AuthSubject.
func (m *MiddlewareV2) RequireRoleV2(role string, next http.HandlerFunc) http.HandlerFunc {
	return m.AuthenticateV2(func(w http.ResponseWriter, r *http.Request) {
		subject := GetAuthSubject(r.Context())
		if subject == nil {
			http.Error(w, "Forbidden: invalid authentication", http.StatusForbidden)
			return
		}

		// Admin role has access to everything
		if subject.HasRole("admin") {
			next(w, r)
			return
		}

		// Check if user has the required role
		if !subject.HasRole(role) {
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			return
		}

		next(w, r)
	})
}

// RequireAnyRoleV2 is middleware that enforces any of the specified roles.
func (m *MiddlewareV2) RequireAnyRoleV2(roles []string, next http.HandlerFunc) http.HandlerFunc {
	return m.AuthenticateV2(func(w http.ResponseWriter, r *http.Request) {
		subject := GetAuthSubject(r.Context())
		if subject == nil {
			http.Error(w, "Forbidden: invalid authentication", http.StatusForbidden)
			return
		}

		// Admin role has access to everything
		if subject.HasRole("admin") {
			next(w, r)
			return
		}

		// Check if user has any of the required roles
		if !subject.HasAnyRole(roles...) {
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			return
		}

		next(w, r)
	})
}

// GetAuthSubject retrieves the AuthSubject from the request context.
func GetAuthSubject(ctx context.Context) *AuthSubject {
	subject, ok := ctx.Value(AuthSubjectContextKey).(*AuthSubject)
	if !ok {
		return nil
	}
	return subject
}

// GetClaims retrieves the Claims from the request context (for backwards compatibility).
func GetClaims(ctx context.Context) *Claims {
	claims, ok := ctx.Value(ClaimsContextKey).(*Claims)
	if !ok {
		return nil
	}
	return claims
}

// RateLimit is middleware that enforces rate limiting.
func (m *MiddlewareV2) RateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting if disabled
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

// CORS is middleware that adds CORS headers based on configuration.
func (m *MiddlewareV2) CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range m.corsOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				if allowedOrigin == "*" {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
				}
				break
			}
		}

		if !allowed && origin != "" {
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Plex-Token")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// getClientIP extracts the client IP address from the request.
func (m *MiddlewareV2) getClientIP(r *http.Request) string {
	// Import the function from middleware.go
	// This is duplicated for now to avoid breaking changes
	remoteIP := r.RemoteAddr
	if idx := len(remoteIP) - 1; idx >= 0 {
		for i := idx; i >= 0; i-- {
			if remoteIP[i] == ':' {
				remoteIP = remoteIP[:i]
				break
			}
		}
	}

	if len(m.trustedProxies) > 0 && m.trustedProxies[remoteIP] {
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			// Get first IP in the chain
			for i := 0; i < len(xff); i++ {
				if xff[i] == ',' {
					return xff[:i]
				}
			}
			return xff
		}

		xri := r.Header.Get("X-Real-IP")
		if xri != "" {
			return xri
		}
	}

	return remoteIP
}
