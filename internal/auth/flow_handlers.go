// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OAuth flow handlers.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// FlowHandlers provides HTTP handlers for OIDC and Plex OAuth flows.
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
// Uses certified Zitadel OIDC library for OpenID Connect authentication.
type FlowHandlers struct {
	oidcFlow          *ZitadelOIDCFlow
	plexFlow          *PlexFlow
	sessionStore      SessionStore
	sessionMiddleware *SessionMiddleware
	config            *FlowHandlersConfig
	auditLogger       *OIDCAuditLogger // Production-grade audit logging
	jtiTracker        JTITracker       // JTI tracking for back-channel logout replay prevention
}

// FlowHandlersConfig holds configuration for the flow handlers.
type FlowHandlersConfig struct {
	// SessionDuration is how long sessions are valid.
	SessionDuration time.Duration

	// DefaultPostLoginRedirect is the default redirect after login.
	DefaultPostLoginRedirect string

	// ErrorRedirectURL is where to redirect on errors.
	ErrorRedirectURL string

	// AllowInsecureCookies allows non-HTTPS cookies (for development).
	AllowInsecureCookies bool

	// PostLogoutRedirectURI is where to redirect after OIDC logout.
	// ADR-0015 Phase 4B: RP-Initiated Logout
	PostLogoutRedirectURI string
}

// DefaultFlowHandlersConfig returns sensible defaults.
func DefaultFlowHandlersConfig() *FlowHandlersConfig {
	return &FlowHandlersConfig{
		SessionDuration:          24 * time.Hour,
		DefaultPostLoginRedirect: "/",
		ErrorRedirectURL:         "/login?error=",
		PostLogoutRedirectURI:    "/",
	}
}

// NewFlowHandlers creates a new FlowHandlers instance.
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
// Uses certified Zitadel OIDC library for OpenID Connect authentication.
//
// Parameters:
//   - oidcFlow: The Zitadel OIDC flow handler (can be nil if OIDC disabled)
//   - plexFlow: The Plex OAuth flow handler (can be nil if Plex auth disabled)
//   - sessionStore: Store for session management
//   - sessionMiddleware: Middleware for session cookie handling
//   - config: Handler configuration (uses defaults if nil)
func NewFlowHandlers(
	oidcFlow *ZitadelOIDCFlow,
	plexFlow *PlexFlow,
	sessionStore SessionStore,
	sessionMiddleware *SessionMiddleware,
	config *FlowHandlersConfig,
) *FlowHandlers {
	if config == nil {
		config = DefaultFlowHandlersConfig()
	}
	return &FlowHandlers{
		oidcFlow:          oidcFlow,
		plexFlow:          plexFlow,
		sessionStore:      sessionStore,
		sessionMiddleware: sessionMiddleware,
		config:            config,
	}
}

// SetAuditLogger sets the audit logger for authentication events.
// This enables production-grade audit logging for OIDC operations.
// ADR-0015: Zero Trust Authentication (Zitadel Amendment)
func (h *FlowHandlers) SetAuditLogger(logger *OIDCAuditLogger) {
	h.auditLogger = logger
}

// GetAuditLogger returns the audit logger (for external components).
func (h *FlowHandlers) GetAuditLogger() *OIDCAuditLogger {
	return h.auditLogger
}

// SetJTITracker sets the JTI tracker for back-channel logout replay prevention.
// When set, back-channel logout tokens are checked for JTI reuse to prevent replay attacks.
// ADR-0015: Zero Trust Authentication - Security Enhancement
func (h *FlowHandlers) SetJTITracker(tracker JTITracker) {
	h.jtiTracker = tracker
}

// GetJTITracker returns the JTI tracker (for external components).
func (h *FlowHandlers) GetJTITracker() JTITracker {
	return h.jtiTracker
}

// ========================
// Common Handlers
// ========================

// UserInfo returns information about the authenticated user.
// GET /api/auth/userinfo
func (h *FlowHandlers) UserInfo(w http.ResponseWriter, r *http.Request) {
	subject := GetAuthSubject(r.Context())
	if subject == nil {
		http.Error(w, "Unauthorized: not authenticated", http.StatusUnauthorized)
		return
	}

	response := buildUserInfoResponse(subject)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logging.Error().Err(err).Msg("Failed to encode userinfo response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// buildUserInfoResponse builds the user info response map.
func buildUserInfoResponse(subject *AuthSubject) map[string]interface{} {
	response := map[string]interface{}{
		"id":       subject.ID,
		"username": subject.Username,
	}

	if subject.Email != "" {
		response["email"] = subject.Email
	}
	if len(subject.Roles) > 0 {
		response["roles"] = subject.Roles
	}
	if len(subject.Groups) > 0 {
		response["groups"] = subject.Groups
	}
	if subject.Provider != "" {
		response["provider"] = subject.Provider
	}

	return response
}

// Logout destroys the current session.
// POST /api/auth/logout
func (h *FlowHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	subject := GetAuthSubject(r.Context())
	if subject != nil && subject.SessionID != "" {
		if err := h.sessionMiddleware.DestroySession(r.Context(), w, subject.SessionID); err != nil {
			logging.Error().Err(err).Str("session_id", subject.SessionID).Msg("Failed to destroy session")
		}
	} else {
		// Clear cookie anyway
		h.sessionMiddleware.ClearSessionCookie(w)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out successfully",
	}); err != nil {
		logging.Error().Err(err).Msg("Failed to encode logout response")
	}
}

// LogoutAll destroys all sessions for the current user.
// POST /api/auth/logout/all
func (h *FlowHandlers) LogoutAll(w http.ResponseWriter, r *http.Request) {
	subject := GetAuthSubject(r.Context())
	if subject == nil {
		http.Error(w, "Unauthorized: not authenticated", http.StatusUnauthorized)
		return
	}

	count, err := h.sessionStore.DeleteByUserID(r.Context(), subject.ID)
	if err != nil {
		logging.Error().Err(err).Str("user_id", subject.ID).Msg("Failed to delete sessions for user")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Clear cookie
	h.sessionMiddleware.ClearSessionCookie(w)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"message":        "All sessions logged out successfully",
		"sessions_count": count,
	}); err != nil {
		logging.Error().Err(err).Msg("Failed to encode logout-all response")
	}
}

// Sessions returns all active sessions for the current user.
// GET /api/auth/sessions
func (h *FlowHandlers) Sessions(w http.ResponseWriter, r *http.Request) {
	subject := GetAuthSubject(r.Context())
	if subject == nil {
		http.Error(w, "Unauthorized: not authenticated", http.StatusUnauthorized)
		return
	}

	sessions, err := h.sessionStore.GetByUserID(r.Context(), subject.ID)
	if err != nil {
		logging.Error().Err(err).Str("user_id", subject.ID).Msg("Failed to get sessions for user")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	sessionInfos := buildSessionInfoList(sessions, subject.SessionID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"sessions": sessionInfos,
	}); err != nil {
		logging.Error().Err(err).Msg("Failed to encode sessions response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// buildSessionInfoList converts sessions to response format.
func buildSessionInfoList(sessions []*Session, currentSessionID string) []map[string]interface{} {
	sessionInfos := make([]map[string]interface{}, 0, len(sessions))
	for _, s := range sessions {
		info := map[string]interface{}{
			"id":         s.ID,
			"provider":   s.Provider,
			"created_at": s.CreatedAt,
			"current":    s.ID == currentSessionID,
		}
		if !s.LastAccessedAt.IsZero() {
			info["last_accessed_at"] = s.LastAccessedAt
		}
		sessionInfos = append(sessionInfos, info)
	}
	return sessionInfos
}

// RevokeSession revokes a specific session.
// DELETE /api/auth/sessions/:id
func (h *FlowHandlers) RevokeSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	subject := GetAuthSubject(r.Context())
	if subject == nil {
		http.Error(w, "Unauthorized: not authenticated", http.StatusUnauthorized)
		return
	}

	// Check if user owns the session or is admin
	session, err := h.sessionStore.Get(r.Context(), sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) || errors.Is(err, ErrSessionExpired) {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}
		logging.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get session")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Only allow user to revoke their own sessions, or admin to revoke any
	if session.UserID != subject.ID && !subject.HasRole("admin") {
		http.Error(w, "Forbidden: cannot revoke other user's session", http.StatusForbidden)
		return
	}

	if err := h.sessionStore.Delete(r.Context(), sessionID); err != nil {
		logging.Error().Err(err).Str("session_id", sessionID).Msg("Failed to delete session")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "Session revoked successfully",
	}); err != nil {
		logging.Error().Err(err).Msg("Failed to encode revoke-session response")
	}
}

// ========================
// Helper Methods
// ========================

// extractSessionIDFromCookie extracts the session ID from the request cookie.
// Used for session fixation protection (Phase 4A.3).
func (h *FlowHandlers) extractSessionIDFromCookie(r *http.Request) string {
	// Get the cookie name from session middleware config
	cookieName := "tautulli_session" // default
	if h.sessionMiddleware != nil {
		cookieName = h.sessionMiddleware.GetCookieName()
	}

	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// getIDTokenHint extracts the ID token from the session for use as id_token_hint.
func (h *FlowHandlers) getIDTokenHint(ctx context.Context, subject *AuthSubject) string {
	if subject == nil || subject.SessionID == "" {
		return ""
	}
	session, err := h.sessionStore.Get(ctx, subject.SessionID)
	if err != nil || session == nil || session.Metadata == nil {
		return ""
	}
	return session.Metadata["id_token"]
}

// destroyLocalSession destroys the user's local session.
func (h *FlowHandlers) destroyLocalSession(ctx context.Context, w http.ResponseWriter, subject *AuthSubject) {
	if subject != nil && subject.SessionID != "" {
		if err := h.sessionMiddleware.DestroySession(ctx, w, subject.SessionID); err != nil {
			logging.Error().Err(err).Str("session_id", subject.SessionID).Msg("Failed to destroy session")
		}
		logging.Info().Str("user", subject.Username).Str("session_id", subject.SessionID).Msg("OIDC logout: local session destroyed")
	} else {
		h.sessionMiddleware.ClearSessionCookie(w)
	}
}

// isAJAXRequest checks if the request is an AJAX request.
func (h *FlowHandlers) isAJAXRequest(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return accept == "application/json" || r.Header.Get("X-Requested-With") == "XMLHttpRequest"
}
