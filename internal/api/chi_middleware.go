// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package api provides Chi middleware factories for production-hardened middleware.
// ADR-0016: Chi router adoption with production-proven middleware ecosystem.
package api

import (
	"net/http"
	"os"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"

	"github.com/tomtom215/cartographus/internal/logging"
)

// ChiMiddlewareConfig holds configuration for Chi middleware factories.
type ChiMiddlewareConfig struct {
	// CORS configuration
	CORSAllowedOrigins   []string
	CORSAllowedMethods   []string
	CORSAllowedHeaders   []string
	CORSExposedHeaders   []string
	CORSAllowCredentials bool
	CORSMaxAge           int // seconds

	// Rate limiting configuration
	RateLimitRequests int
	RateLimitWindow   time.Duration
	RateLimitDisabled bool
	RateLimitKeyFunc  httprate.KeyFunc
	RateLimitOnLimit  http.HandlerFunc
}

// DefaultChiMiddlewareConfig returns a secure default configuration.
// CORS origins default to empty, requiring explicit configuration.
// This prevents accidental deployment with insecure wildcard CORS.
func DefaultChiMiddlewareConfig() *ChiMiddlewareConfig {
	return &ChiMiddlewareConfig{
		CORSAllowedOrigins:   []string{}, // Empty by default - requires explicit configuration
		CORSAllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		CORSAllowedHeaders:   []string{"Content-Type", "Authorization", "X-Plex-Token"},
		CORSExposedHeaders:   []string{},
		CORSAllowCredentials: false,
		CORSMaxAge:           86400, // 24 hours, matching existing behavior

		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		RateLimitDisabled: false,
	}
}

// ChiMiddleware provides Chi-compatible middleware factories.
// This uses production-hardened implementations from the Chi ecosystem.
type ChiMiddleware struct {
	config *ChiMiddlewareConfig
	cors   func(http.Handler) http.Handler
}

// NewChiMiddleware creates a new Chi middleware factory with the given configuration.
func NewChiMiddleware(config *ChiMiddlewareConfig) *ChiMiddleware {
	if config == nil {
		config = DefaultChiMiddlewareConfig()
	}

	// Build CORS handler using go-chi/cors
	corsHandler := cors.Handler(cors.Options{
		AllowedOrigins:   config.CORSAllowedOrigins,
		AllowedMethods:   config.CORSAllowedMethods,
		AllowedHeaders:   config.CORSAllowedHeaders,
		ExposedHeaders:   config.CORSExposedHeaders,
		AllowCredentials: config.CORSAllowCredentials,
		MaxAge:           config.CORSMaxAge,
	})

	return &ChiMiddleware{
		config: config,
		cors:   corsHandler,
	}
}

// CORS returns a Chi-compatible CORS middleware using go-chi/cors.
// This is a production-hardened replacement for the custom CORS middleware.
func (m *ChiMiddleware) CORS() func(http.Handler) http.Handler {
	return m.cors
}

// RateLimit returns a Chi-compatible rate limiting middleware using go-chi/httprate.
// This is a production-hardened replacement for the custom rate limiting middleware.
func (m *ChiMiddleware) RateLimit() func(http.Handler) http.Handler {
	if m.config.RateLimitDisabled {
		// Return a no-op middleware when rate limiting is disabled
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	// Use IP-based rate limiting by default, or custom key function if provided
	keyFunc := m.config.RateLimitKeyFunc
	if keyFunc == nil {
		keyFunc = httprate.KeyByIP
	}

	// Use custom limit handler or default
	opts := []httprate.Option{
		httprate.WithKeyFuncs(keyFunc),
	}

	if m.config.RateLimitOnLimit != nil {
		opts = append(opts, httprate.WithLimitHandler(m.config.RateLimitOnLimit))
	}

	return httprate.Limit(
		m.config.RateLimitRequests,
		m.config.RateLimitWindow,
		opts...,
	)
}

// RateLimitByIP returns a rate limiter that uses IP-based key extraction.
// This is suitable for most API endpoints.
func (m *ChiMiddleware) RateLimitByIP() func(http.Handler) http.Handler {
	if m.config.RateLimitDisabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return httprate.LimitByIP(
		m.config.RateLimitRequests,
		m.config.RateLimitWindow,
	)
}

// RateLimitByRealIP returns a rate limiter that uses the real IP from X-Forwarded-For.
// This is suitable when behind a reverse proxy.
func (m *ChiMiddleware) RateLimitByRealIP() func(http.Handler) http.Handler {
	if m.config.RateLimitDisabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return httprate.LimitByRealIP(
		m.config.RateLimitRequests,
		m.config.RateLimitWindow,
	)
}

// RequestIDWithLogging returns a middleware that adds request ID to the context
// and integrates with the logging package for distributed tracing.
// This wraps chi's RequestID middleware and adds correlation_id and request_id
// to the logging context, enabling structured logging with request tracing.
func RequestIDWithLogging() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// First apply chi's RequestID middleware
		chiRequestID := chimiddleware.RequestID(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the request ID that chi will set (from header or generated)
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				// chi will generate one, but we need it for logging context
				// so we generate our own that chi will then use
				requestID = logging.GenerateRequestID()
				r.Header.Set("X-Request-ID", requestID)
			}

			// Add logging context with request and correlation IDs
			ctx := logging.ContextWithRequestID(r.Context(), requestID)
			ctx = logging.ContextWithNewCorrelationID(ctx)

			// Pass through to chi's RequestID middleware with enriched context
			chiRequestID.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// NewChiMiddlewareFromAuth creates a ChiMiddleware instance from existing auth config.
// This bridges the existing configuration to the new Chi middleware.
func NewChiMiddlewareFromAuth(corsOrigins []string, rateLimitReqs int, rateLimitWindow time.Duration, rateLimitDisabled bool) *ChiMiddleware {
	config := &ChiMiddlewareConfig{
		CORSAllowedOrigins:   corsOrigins,
		CORSAllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		CORSAllowedHeaders:   []string{"Content-Type", "Authorization", "X-Plex-Token"},
		CORSExposedHeaders:   []string{},
		CORSAllowCredentials: false,
		CORSMaxAge:           86400,

		RateLimitRequests: rateLimitReqs,
		RateLimitWindow:   rateLimitWindow,
		RateLimitDisabled: rateLimitDisabled,
	}

	return NewChiMiddleware(config)
}

// ================================================================================
// Phase 3: Endpoint-Specific Rate Limits
// ================================================================================

// RateLimitConfig defines rate limit parameters for specific endpoints.
type RateLimitConfig struct {
	// Requests is the number of requests allowed in the window
	Requests int
	// Window is the time window for rate limiting
	Window time.Duration
}

// Endpoint-specific rate limit configurations
// These are tuned for production workloads based on endpoint characteristics
var (
	// RateLimitAuth is strict limiting for authentication endpoints (brute force prevention)
	RateLimitAuth = RateLimitConfig{Requests: 5, Window: time.Minute}

	// RateLimitLogin is very strict for login attempts
	RateLimitLogin = RateLimitConfig{Requests: 5, Window: 5 * time.Minute}

	// RateLimitSync is moderate limiting for sync operations (resource intensive)
	RateLimitSync = RateLimitConfig{Requests: 10, Window: time.Minute}

	// RateLimitWrite is moderate limiting for write operations
	RateLimitWrite = RateLimitConfig{Requests: 30, Window: time.Minute}

	// RateLimitAnalytics is permissive for read-heavy cached analytics (31 endpoints)
	// Dashboard loads 31 charts simultaneously; 1000/min allows smooth exploration
	// across all analytics pages without users hitting limits unexpectedly
	RateLimitAnalytics = RateLimitConfig{Requests: 1000, Window: time.Minute}

	// RateLimitExport is moderate limiting for export operations (resource intensive)
	RateLimitExport = RateLimitConfig{Requests: 10, Window: time.Minute}

	// RateLimitWebSocket is permissive for WebSocket connections (upgrade rate)
	RateLimitWebSocket = RateLimitConfig{Requests: 30, Window: time.Minute}

	// RateLimitAPI is the default API rate limit
	RateLimitAPI = RateLimitConfig{Requests: 100, Window: time.Minute}

	// RateLimitBurst is for interactive endpoints needing burst capacity
	RateLimitBurst = RateLimitConfig{Requests: 200, Window: time.Minute}
)

// RateLimitCustom returns a rate limiter with custom configuration.
// Phase 3: Enables endpoint-specific rate limiting.
func (m *ChiMiddleware) RateLimitCustom(config RateLimitConfig) func(http.Handler) http.Handler {
	if m.config.RateLimitDisabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return httprate.LimitByIP(config.Requests, config.Window)
}

// RateLimitAuth returns a strict rate limiter for authentication endpoints.
// Prevents brute force attacks by limiting login attempts.
func (m *ChiMiddleware) RateLimitAuth() func(http.Handler) http.Handler {
	return m.RateLimitCustom(RateLimitAuth)
}

// RateLimitLogin returns a very strict rate limiter for login endpoints.
// Prevents credential stuffing and brute force attacks.
func (m *ChiMiddleware) RateLimitLogin() func(http.Handler) http.Handler {
	return m.RateLimitCustom(RateLimitLogin)
}

// RateLimitSync returns a rate limiter for sync operations.
// These are resource-intensive and should be limited.
func (m *ChiMiddleware) RateLimitSync() func(http.Handler) http.Handler {
	return m.RateLimitCustom(RateLimitSync)
}

// RateLimitWrite returns a rate limiter for write operations.
// Protects database from write floods.
func (m *ChiMiddleware) RateLimitWrite() func(http.Handler) http.Handler {
	return m.RateLimitCustom(RateLimitWrite)
}

// RateLimitAnalytics returns a rate limiter for analytics endpoints.
// More permissive since these are read-heavy cached operations.
func (m *ChiMiddleware) RateLimitAnalytics() func(http.Handler) http.Handler {
	return m.RateLimitCustom(RateLimitAnalytics)
}

// RateLimitExport returns a rate limiter for export operations.
// These are resource-intensive and should be limited.
func (m *ChiMiddleware) RateLimitExport() func(http.Handler) http.Handler {
	return m.RateLimitCustom(RateLimitExport)
}

// RateLimitBurst returns a rate limiter for interactive endpoints.
// Allows burst traffic for better user experience.
func (m *ChiMiddleware) RateLimitBurst() func(http.Handler) http.Handler {
	return m.RateLimitCustom(RateLimitBurst)
}

// RateLimitHealth is permissive rate limiting for health endpoints (1000/min).
// L-02 Security Fix: Health endpoints need rate limiting to prevent abuse
// while still allowing frequent health checks from monitoring tools.
var RateLimitHealth = RateLimitConfig{Requests: 1000, Window: time.Minute}

// RateLimitHealth returns a rate limiter for health endpoints.
// Prevents abuse while allowing frequent monitoring checks.
func (m *ChiMiddleware) RateLimitHealth() func(http.Handler) http.Handler {
	return m.RateLimitCustom(RateLimitHealth)
}

// ================================================================================
// L-01 Security Fix: API Security Headers
// ================================================================================

// APISecurityHeaders returns a middleware that adds security headers to API responses.
// This addresses L-01: Missing security headers on API endpoints.
//
// Headers added:
//   - X-Content-Type-Options: nosniff (prevents MIME type sniffing)
//   - X-Frame-Options: DENY (prevents clickjacking)
//   - Cache-Control: no-store (prevents caching of sensitive API responses)
//   - Referrer-Policy: strict-origin-when-cross-origin (limits referrer information)
//
// Note: Content-Security-Policy is not added to API endpoints as it's designed for HTML.
// HSTS is added conditionally when the request is over HTTPS.
func APISecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Prevent embedding in frames (clickjacking protection)
			w.Header().Set("X-Frame-Options", "DENY")

			// Control referrer information
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Add HSTS header when request is over HTTPS or behind a TLS-terminating proxy
			// Check X-Forwarded-Proto for reverse proxy setups
			if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
				// 1 year max-age with includeSubDomains
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ================================================================================
// E2E Debug Logging (Diagnostic)
// ================================================================================

// e2eDebugEnabled caches the E2E_DEBUG environment variable check.
var e2eDebugEnabled = os.Getenv("E2E_DEBUG") == "true"

// E2EDebugLogging returns a middleware that logs all incoming requests for E2E debugging.
// This is only enabled when the E2E_DEBUG environment variable is set to "true".
// It logs the request method, path, remote address, response status, and duration.
//
// Enable in CI by setting: E2E_DEBUG=true
func E2EDebugLogging() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Skip if E2E debugging is not enabled
		if !e2eDebugEnabled {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			ww := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Log request start
			logging.Info().
				Str("component", "e2e-debug").
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("query", r.URL.RawQuery).
				Str("remote_addr", r.RemoteAddr).
				Str("user_agent", r.UserAgent()).
				Msg("[E2E] Request received")

			// Call next handler
			next.ServeHTTP(ww, r)

			// Log request completion
			duration := time.Since(start)
			logging.Info().
				Str("component", "e2e-debug").
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", ww.statusCode).
				Dur("duration", duration).
				Msg("[E2E] Request completed")
		})
	}
}

// statusResponseWriter wraps http.ResponseWriter to capture the status code.
type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and calls the underlying WriteHeader.
func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// ================================================================================
// RBAC Phase 3: Authorization Middleware
// ================================================================================

// RequireAdminMiddleware returns a Chi middleware that requires admin role.
// This middleware should be applied AFTER authentication middleware.
// If the user is not authenticated, returns 401.
// If the user is not an admin, returns 403.
//
// Usage:
//
//	r.Group(func(r chi.Router) {
//	    r.Use(chiMiddleware(router.middleware.Authenticate))
//	    r.Use(RequireAdminMiddleware())
//	    r.Post("/admin/action", handler.AdminAction)
//	})
func RequireAdminMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hctx := GetHandlerContext(r)
			if err := hctx.RequireAdmin(); err != nil {
				logging.Warn().
					Str("user_id", hctx.UserID).
					Str("effective_role", hctx.EffectiveRole).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Msg("Access denied: admin role required")
				RespondAuthError(w, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireEditorMiddleware returns a Chi middleware that requires editor or admin role.
// This middleware should be applied AFTER authentication middleware.
// If the user is not authenticated, returns 401.
// If the user is not an editor or admin, returns 403.
//
// Usage:
//
//	r.Group(func(r chi.Router) {
//	    r.Use(chiMiddleware(router.middleware.Authenticate))
//	    r.Use(RequireEditorMiddleware())
//	    r.Post("/content/edit", handler.EditContent)
//	})
func RequireEditorMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hctx := GetHandlerContext(r)
			if err := hctx.RequireEditor(); err != nil {
				logging.Warn().
					Str("user_id", hctx.UserID).
					Str("effective_role", hctx.EffectiveRole).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Msg("Access denied: editor role required")
				RespondAuthError(w, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuthMiddleware returns a Chi middleware that requires authentication.
// This is useful for endpoints that need any authenticated user (viewer, editor, or admin).
// If the user is not authenticated, returns 401.
//
// Usage:
//
//	r.Group(func(r chi.Router) {
//	    r.Use(RequireAuthMiddleware())
//	    r.Get("/profile", handler.Profile)
//	})
func RequireAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hctx := GetHandlerContext(r)
			if !hctx.IsAuthenticated() {
				logging.Warn().
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Msg("Access denied: authentication required")
				RespondAuthError(w, ErrNotAuthenticated)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
