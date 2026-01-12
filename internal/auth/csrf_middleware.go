// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including CSRF protection.
// ADR-0015: Zero Trust Authentication & Authorization
// Phase 4D.1: CSRF Protection Middleware
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
)

// CSRF protection errors
var (
	// ErrCSRFTokenMissing indicates no CSRF token was provided.
	ErrCSRFTokenMissing = errors.New("CSRF token missing")

	// ErrCSRFTokenInvalid indicates the CSRF token is invalid or doesn't match.
	ErrCSRFTokenInvalid = errors.New("CSRF token invalid")

	// ErrCSRFTokenExpired indicates the CSRF token has expired.
	ErrCSRFTokenExpired = errors.New("CSRF token expired")
)

// CSRFConfig holds configuration for CSRF protection middleware.
type CSRFConfig struct {
	// CookieName is the name of the CSRF cookie (default: "_csrf").
	CookieName string

	// HeaderName is the HTTP header name for CSRF token (default: "X-CSRF-Token").
	HeaderName string

	// FormFieldName is the form field name for CSRF token (default: "csrf_token").
	FormFieldName string

	// CookiePath is the path for the CSRF cookie (default: "/").
	CookiePath string

	// CookieDomain is the domain for the CSRF cookie.
	CookieDomain string

	// CookieSecure sets the Secure flag on the cookie (default: true).
	CookieSecure bool

	// CookieHTTPOnly sets the HttpOnly flag (default: false for CSRF - needs JS access).
	CookieHTTPOnly bool

	// CookieSameSite sets the SameSite attribute (default: Strict).
	CookieSameSite http.SameSite

	// TokenLength is the byte length of the CSRF token (default: 32).
	TokenLength int

	// TokenTTL is how long tokens are valid (default: 24h).
	TokenTTL time.Duration

	// ExemptPaths are paths that don't require CSRF protection.
	// Use this for public endpoints or server-to-server APIs.
	ExemptPaths []string

	// ExemptMethods are HTTP methods that don't require CSRF protection.
	// Default: GET, HEAD, OPTIONS, TRACE (safe methods per RFC 7231).
	ExemptMethods []string

	// ErrorHandler is called when CSRF validation fails.
	// If nil, returns 403 Forbidden with JSON error.
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

	// TrustedOrigins are origins allowed to make cross-origin requests.
	// If empty, origin checking is disabled.
	TrustedOrigins []string
}

// DefaultCSRFConfig returns sensible defaults for CSRF protection.
func DefaultCSRFConfig() *CSRFConfig {
	return &CSRFConfig{
		CookieName:     "_csrf",
		HeaderName:     "X-CSRF-Token",
		FormFieldName:  "csrf_token",
		CookiePath:     "/",
		CookieSecure:   true,
		CookieHTTPOnly: false, // CSRF tokens need JS access for SPA
		CookieSameSite: http.SameSiteStrictMode,
		TokenLength:    32,
		TokenTTL:       24 * time.Hour,
		ExemptMethods:  []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace},
	}
}

// CSRFMiddleware provides Cross-Site Request Forgery protection.
// It uses the double-submit cookie pattern where a token is stored in both
// a cookie and must be submitted in a header or form field.
type CSRFMiddleware struct {
	config *CSRFConfig
	tokens *csrfTokenStore
}

// csrfTokenStore provides thread-safe token storage.
type csrfTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]*csrfTokenData
}

type csrfTokenData struct {
	token     string
	createdAt time.Time
	expiresAt time.Time
}

// newCSRFTokenStore creates a new token store.
func newCSRFTokenStore() *csrfTokenStore {
	return &csrfTokenStore{
		tokens: make(map[string]*csrfTokenData),
	}
}

// NewCSRFMiddleware creates a new CSRF protection middleware.
func NewCSRFMiddleware(config *CSRFConfig) *CSRFMiddleware {
	if config == nil {
		config = DefaultCSRFConfig()
	}

	// Set defaults if not provided
	if config.CookieName == "" {
		config.CookieName = "_csrf"
	}
	if config.HeaderName == "" {
		config.HeaderName = "X-CSRF-Token"
	}
	if config.FormFieldName == "" {
		config.FormFieldName = "csrf_token"
	}
	if config.TokenLength == 0 {
		config.TokenLength = 32
	}
	if config.TokenTTL == 0 {
		config.TokenTTL = 24 * time.Hour
	}
	if len(config.ExemptMethods) == 0 {
		config.ExemptMethods = []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace}
	}

	return &CSRFMiddleware{
		config: config,
		tokens: newCSRFTokenStore(),
	}
}

// Protect is a middleware that provides CSRF protection.
// It sets a CSRF token cookie and validates the token on protected requests.
func (m *CSRFMiddleware) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path is exempt
		if m.isExemptPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Check if method is exempt (safe methods)
		if m.isExemptMethod(r.Method) {
			// Still set/refresh the token for GET requests
			m.ensureToken(w, r)
			next.ServeHTTP(w, r)
			return
		}

		// Validate CSRF token for state-changing requests
		if err := m.validateToken(r); err != nil {
			m.handleError(w, r, err)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ensureToken ensures a CSRF token exists in the cookie.
func (m *CSRFMiddleware) ensureToken(w http.ResponseWriter, r *http.Request) {
	// Check if token cookie exists
	cookie, err := r.Cookie(m.config.CookieName)
	if err == nil && cookie.Value != "" {
		// Validate existing token
		if m.tokens.isValid(cookie.Value) {
			return // Token exists and is valid
		}
	}

	// Generate new token
	token, err := m.generateToken()
	if err != nil {
		logging.Error().Err(err).Msg("CSRF: failed to generate token")
		return
	}

	// Store token
	m.tokens.store(token, m.config.TokenTTL)

	// Set cookie
	m.setTokenCookie(w, token)
}

// validateToken validates the CSRF token from the request.
func (m *CSRFMiddleware) validateToken(r *http.Request) error {
	// Get token from cookie
	cookieToken := m.getTokenFromCookie(r)
	if cookieToken == "" {
		return ErrCSRFTokenMissing
	}

	// Get token from request (header or form)
	requestToken := m.getTokenFromRequest(r)
	if requestToken == "" {
		return ErrCSRFTokenMissing
	}

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(cookieToken), []byte(requestToken)) != 1 {
		return ErrCSRFTokenInvalid
	}

	// Validate token exists and is not expired
	if !m.tokens.isValid(cookieToken) {
		return ErrCSRFTokenExpired
	}

	return nil
}

// getTokenFromCookie extracts the CSRF token from the cookie.
func (m *CSRFMiddleware) getTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(m.config.CookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// getTokenFromRequest extracts the CSRF token from header or form.
func (m *CSRFMiddleware) getTokenFromRequest(r *http.Request) string {
	// Try header first
	token := r.Header.Get(m.config.HeaderName)
	if token != "" {
		return token
	}

	// Try form field
	if r.Method == http.MethodPost {
		// Parse form if not already parsed
		if r.PostForm == nil {
			//nolint:errcheck // best effort form parsing
			r.ParseForm()
		}
		token = r.FormValue(m.config.FormFieldName)
		if token != "" {
			return token
		}
	}

	return ""
}

// generateToken generates a cryptographically secure CSRF token.
func (m *CSRFMiddleware) generateToken() (string, error) {
	bytes := make([]byte, m.config.TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// setTokenCookie sets the CSRF token cookie.
func (m *CSRFMiddleware) setTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.config.CookieName,
		Value:    token,
		Path:     m.config.CookiePath,
		Domain:   m.config.CookieDomain,
		MaxAge:   int(m.config.TokenTTL.Seconds()),
		Secure:   m.config.CookieSecure,
		HttpOnly: m.config.CookieHTTPOnly,
		SameSite: m.config.CookieSameSite,
	})
}

// isExemptPath checks if the path is exempt from CSRF protection.
func (m *CSRFMiddleware) isExemptPath(path string) bool {
	for _, exempt := range m.config.ExemptPaths {
		if strings.HasPrefix(path, exempt) {
			return true
		}
	}
	return false
}

// isExemptMethod checks if the HTTP method is exempt from CSRF protection.
func (m *CSRFMiddleware) isExemptMethod(method string) bool {
	for _, exempt := range m.config.ExemptMethods {
		if strings.EqualFold(method, exempt) {
			return true
		}
	}
	return false
}

// handleError handles CSRF validation errors.
func (m *CSRFMiddleware) handleError(w http.ResponseWriter, r *http.Request, err error) {
	if m.config.ErrorHandler != nil {
		m.config.ErrorHandler(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)

	var errMsg string
	switch {
	case errors.Is(err, ErrCSRFTokenMissing):
		errMsg = "CSRF token missing"
	case errors.Is(err, ErrCSRFTokenInvalid):
		errMsg = "CSRF token invalid"
	case errors.Is(err, ErrCSRFTokenExpired):
		errMsg = "CSRF token expired"
	default:
		errMsg = "CSRF validation failed"
	}

	//nolint:errcheck // error response
	w.Write([]byte(`{"error":"csrf_failed","error_description":"` + errMsg + `"}`))
}

// GetToken returns a valid CSRF token for the response.
// Use this to include the token in responses for SPAs.
func (m *CSRFMiddleware) GetToken(w http.ResponseWriter, r *http.Request) string {
	// Check existing token
	cookie, err := r.Cookie(m.config.CookieName)
	if err == nil && cookie.Value != "" {
		if m.tokens.isValid(cookie.Value) {
			return cookie.Value
		}
	}

	// Generate new token
	token, err := m.generateToken()
	if err != nil {
		logging.Error().Err(err).Msg("CSRF: failed to generate token")
		return ""
	}

	// Store and set cookie
	m.tokens.store(token, m.config.TokenTTL)
	m.setTokenCookie(w, token)

	return token
}

// Token store methods

func (s *csrfTokenStore) store(token string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.tokens[token] = &csrfTokenData{
		token:     token,
		createdAt: now,
		expiresAt: now.Add(ttl),
	}
}

func (s *csrfTokenStore) isValid(token string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.tokens[token]
	if !ok {
		return false
	}

	return time.Now().Before(data.expiresAt)
}

// CleanupExpired removes all expired tokens from the store.
// Call this periodically to prevent memory leaks.
func (s *csrfTokenStore) CleanupExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	now := time.Now()
	for token, data := range s.tokens {
		if now.After(data.expiresAt) {
			delete(s.tokens, token)
			count++
		}
	}
	return count
}

// StartCleanupRoutine starts a background routine to clean up expired tokens.
func (m *CSRFMiddleware) StartCleanupRoutine(interval time.Duration) chan struct{} {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.tokens.CleanupExpired()
			case <-done:
				return
			}
		}
	}()
	return done
}
