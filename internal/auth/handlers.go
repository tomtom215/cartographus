// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication middleware with support for multiple auth modes.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"errors"
	"net/http"

	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// AuthHandlersConfig holds configuration for the auth handlers.
type AuthHandlersConfig struct {
	// OIDCLoginURL is the URL to redirect to for OIDC login.
	OIDCLoginURL string

	// PlexLoginURL is the URL to redirect to for Plex login.
	PlexLoginURL string

	// PostLogoutRedirectURL is where to redirect after logout.
	PostLogoutRedirectURL string
}

// AuthHandlers provides HTTP handlers for authentication operations.
type AuthHandlers struct {
	sessionStore SessionStore
	config       *AuthHandlersConfig
}

// NewAuthHandlers creates a new AuthHandlers instance.
func NewAuthHandlers(store SessionStore, config *AuthHandlersConfig) *AuthHandlers {
	if config == nil {
		config = &AuthHandlersConfig{
			PostLogoutRedirectURL: "/",
		}
	}
	return &AuthHandlers{
		sessionStore: store,
		config:       config,
	}
}

// UserInfo returns information about the authenticated user.
// GET /api/auth/userinfo
func (h *AuthHandlers) UserInfo(w http.ResponseWriter, r *http.Request) {
	subject := GetAuthSubject(r.Context())
	if subject == nil {
		http.Error(w, "Unauthorized: not authenticated", http.StatusUnauthorized)
		return
	}

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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logging.Error().Err(err).Msg("Failed to encode userinfo response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Logout destroys the current session.
// POST /api/auth/logout
func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	subject := GetAuthSubject(r.Context())
	if subject != nil && subject.SessionID != "" {
		if err := h.sessionStore.Delete(r.Context(), subject.SessionID); err != nil {
			logging.Error().Err(err).Str("session_id", subject.SessionID).Msg("Failed to delete session")
		}
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

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
func (h *AuthHandlers) LogoutAll(w http.ResponseWriter, r *http.Request) {
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

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

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
func (h *AuthHandlers) Sessions(w http.ResponseWriter, r *http.Request) {
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
	sessionInfos := make([]map[string]interface{}, 0, len(sessions))
	for _, s := range sessions {
		info := map[string]interface{}{
			"id":         s.ID,
			"provider":   s.Provider,
			"created_at": s.CreatedAt,
			"current":    s.ID == subject.SessionID,
		}
		if !s.LastAccessedAt.IsZero() {
			info["last_accessed_at"] = s.LastAccessedAt
		}
		sessionInfos = append(sessionInfos, info)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"sessions": sessionInfos,
	}); err != nil {
		logging.Error().Err(err).Msg("Failed to encode sessions response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// RevokeSession revokes a specific session.
// DELETE /api/auth/sessions/:id
func (h *AuthHandlers) RevokeSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	subject := GetAuthSubject(r.Context())
	if subject == nil {
		http.Error(w, "Unauthorized: not authenticated", http.StatusUnauthorized)
		return
	}

	// Check if user owns the session or is admin
	session, err := h.sessionStore.Get(r.Context(), sessionID)
	if errors.Is(err, ErrSessionNotFound) || errors.Is(err, ErrSessionExpired) {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}
	if err != nil {
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

// OIDCLogin initiates the OIDC login flow.
// GET /api/auth/oidc/login
func (h *AuthHandlers) OIDCLogin(w http.ResponseWriter, r *http.Request) {
	if h.config.OIDCLoginURL == "" {
		http.Error(w, "OIDC login not configured", http.StatusServiceUnavailable)
		return
	}

	// In a full implementation, this would:
	// 1. Generate a state parameter for CSRF protection
	// 2. Store the state in a cookie or session
	// 3. Redirect to the OIDC provider's authorization endpoint

	http.Redirect(w, r, h.config.OIDCLoginURL, http.StatusFound)
}

// PlexLogin initiates the Plex OAuth login flow.
// GET /api/auth/plex/login
func (h *AuthHandlers) PlexLogin(w http.ResponseWriter, r *http.Request) {
	if h.config.PlexLoginURL == "" {
		http.Error(w, "Plex login not configured", http.StatusServiceUnavailable)
		return
	}

	// In a full implementation, this would:
	// 1. Generate a state parameter for CSRF protection
	// 2. Store the state in a cookie or session
	// 3. Redirect to Plex's authorization endpoint

	http.Redirect(w, r, h.config.PlexLoginURL, http.StatusFound)
}

// HealthCheck returns the auth service health status.
// GET /api/auth/health
func (h *AuthHandlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	}); err != nil {
		logging.Error().Err(err).Msg("Failed to encode health check response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
