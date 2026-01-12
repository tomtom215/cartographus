// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including session management.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
)

// SessionMiddlewareConfig holds configuration for the session middleware.
type SessionMiddlewareConfig struct {
	// CookieName is the name of the session cookie.
	CookieName string

	// HeaderName is an optional header to read the session token from.
	// If set, the header takes priority over the cookie.
	HeaderName string

	// SessionTTL is the session time-to-live.
	SessionTTL time.Duration

	// SlidingSession enables session expiry extension on each request.
	SlidingSession bool

	// CookiePath is the path for the session cookie.
	CookiePath string

	// CookieDomain is the domain for the session cookie.
	CookieDomain string

	// CookieSecure sets the Secure flag on the cookie.
	CookieSecure bool

	// CookieHTTPOnly sets the HttpOnly flag on the cookie.
	CookieHTTPOnly bool

	// CookieSameSite sets the SameSite attribute.
	CookieSameSite http.SameSite
}

// DefaultSessionMiddlewareConfig returns sensible defaults.
func DefaultSessionMiddlewareConfig() *SessionMiddlewareConfig {
	return &SessionMiddlewareConfig{
		CookieName:     "session",
		SessionTTL:     24 * time.Hour,
		SlidingSession: true,
		CookiePath:     "/",
		CookieSecure:   true,
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteLaxMode,
	}
}

// SessionMiddleware provides session-based authentication middleware.
type SessionMiddleware struct {
	store  SessionStore
	config *SessionMiddlewareConfig
}

// NewSessionMiddleware creates a new session middleware.
func NewSessionMiddleware(store SessionStore, config *SessionMiddlewareConfig) *SessionMiddleware {
	if config == nil {
		config = DefaultSessionMiddlewareConfig()
	}
	return &SessionMiddleware{
		store:  store,
		config: config,
	}
}

// Authenticate is a middleware that extracts and validates the session from
// the request cookie or header. If valid, it sets the AuthSubject in the
// request context. If no session is found, the request continues without
// an AuthSubject (use RequireAuth for protected routes).
func (m *SessionMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract session ID
		sessionID := m.extractSessionID(r)
		if sessionID == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Look up session
		session, err := m.store.Get(r.Context(), sessionID)
		if err != nil {
			// Session not found or expired - continue without auth
			if !errors.Is(err, ErrSessionNotFound) && !errors.Is(err, ErrSessionExpired) {
				logging.Error().Err(err).Msg("Session lookup error")
			}
			next.ServeHTTP(w, r)
			return
		}

		// Touch session to extend expiry (sliding sessions)
		if m.config.SlidingSession {
			newExpiry := time.Now().Add(m.config.SessionTTL)
			if touchErr := m.store.Touch(r.Context(), sessionID, newExpiry); touchErr != nil {
				logging.Error().Err(touchErr).Msg("Failed to touch session")
			}
		}

		// Convert session to AuthSubject and add to context
		subject := session.ToAuthSubject()
		subject.SessionID = session.ID

		ctx := context.WithValue(r.Context(), AuthSubjectContextKey, subject)

		// Also set Claims for backwards compatibility
		if claims := subject.ToClaims(); claims != nil {
			ctx = context.WithValue(ctx, ClaimsContextKey, claims)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuth is a middleware that requires a valid session.
// Returns 401 Unauthorized if no valid session is present.
func (m *SessionMiddleware) RequireAuth(next http.Handler) http.Handler {
	return m.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subject := GetAuthSubject(r.Context())
		if subject == nil {
			http.Error(w, "Unauthorized: authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// RequireRole is a middleware that requires the user to have a specific role.
// Returns 401 if not authenticated, 403 if authenticated but missing role.
func (m *SessionMiddleware) RequireRole(role string, next http.Handler) http.Handler {
	return m.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subject := GetAuthSubject(r.Context())
		if subject == nil {
			http.Error(w, "Unauthorized: authentication required", http.StatusUnauthorized)
			return
		}

		if !subject.HasRole(role) {
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}))
}

// RequireAnyRole is a middleware that requires the user to have any of the specified roles.
func (m *SessionMiddleware) RequireAnyRole(roles []string, next http.Handler) http.Handler {
	return m.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subject := GetAuthSubject(r.Context())
		if subject == nil {
			http.Error(w, "Unauthorized: authentication required", http.StatusUnauthorized)
			return
		}

		if !subject.HasAnyRole(roles...) {
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}))
}

// extractSessionID extracts the session ID from the request.
// Priority: Header > Cookie
func (m *SessionMiddleware) extractSessionID(r *http.Request) string {
	// Check header first (if configured)
	if m.config.HeaderName != "" {
		if headerValue := r.Header.Get(m.config.HeaderName); headerValue != "" {
			return headerValue
		}
	}

	// Check cookie
	cookie, err := r.Cookie(m.config.CookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}

// SetSessionCookie sets the session cookie on the response.
func (m *SessionMiddleware) SetSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.config.CookieName,
		Value:    sessionID,
		Path:     m.config.CookiePath,
		Domain:   m.config.CookieDomain,
		MaxAge:   int(m.config.SessionTTL.Seconds()),
		Secure:   m.config.CookieSecure,
		HttpOnly: m.config.CookieHTTPOnly,
		SameSite: m.config.CookieSameSite,
	})
}

// ClearSessionCookie clears the session cookie.
func (m *SessionMiddleware) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.config.CookieName,
		Value:    "",
		Path:     m.config.CookiePath,
		Domain:   m.config.CookieDomain,
		MaxAge:   -1,
		Secure:   m.config.CookieSecure,
		HttpOnly: m.config.CookieHTTPOnly,
		SameSite: m.config.CookieSameSite,
	})
}

// CreateSession creates a new session for the subject and sets the cookie.
// If oldSessionID is provided, the old session is deleted first to prevent session fixation.
func (m *SessionMiddleware) CreateSession(ctx context.Context, w http.ResponseWriter, subject *AuthSubject) (*Session, error) {
	return m.CreateSessionWithOldID(ctx, w, subject, "")
}

// CreateSessionWithOldID creates a new session and deletes any existing session.
// This provides protection against session fixation attacks by ensuring a fresh
// session ID is generated after successful authentication.
// ADR-0015 Phase 4A.3: Session Fixation Protection
func (m *SessionMiddleware) CreateSessionWithOldID(ctx context.Context, w http.ResponseWriter, subject *AuthSubject, oldSessionID string) (*Session, error) {
	// Delete old session if provided (session fixation protection)
	if oldSessionID != "" {
		// Best effort deletion - ignore errors
		//nolint:errcheck // non-critical cleanup
		m.store.Delete(ctx, oldSessionID)
	}

	session := NewSession(subject, m.config.SessionTTL)

	if err := m.store.Create(ctx, session); err != nil {
		return nil, err
	}

	m.SetSessionCookie(w, session.ID)
	return session, nil
}

// DestroySession destroys the session and clears the cookie.
func (m *SessionMiddleware) DestroySession(ctx context.Context, w http.ResponseWriter, sessionID string) error {
	if err := m.store.Delete(ctx, sessionID); err != nil {
		return err
	}
	m.ClearSessionCookie(w)
	return nil
}

// GetCookieName returns the configured session cookie name.
// Used for session fixation protection (Phase 4A.3).
func (m *SessionMiddleware) GetCookieName() string {
	return m.config.CookieName
}
