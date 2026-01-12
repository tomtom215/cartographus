// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"
)

// =====================================================
// Auth API Handlers Tests
// ADR-0015: Zero Trust Authentication & Authorization
// =====================================================

func TestAuthHandlers_UserInfo_Authenticated(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil)

	// Create a session
	session := &Session{
		ID:        "session-123",
		UserID:    "user-abc",
		Username:  "testuser",
		Email:     "test@example.com",
		Roles:     []string{"viewer", "editor"},
		Groups:    []string{"users"},
		Provider:  "oidc",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err := store.Create(context.Background(), session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create request with session in context
	req := httptest.NewRequest(http.MethodGet, "/api/auth/userinfo", nil)
	subject := session.ToAuthSubject()
	ctx := context.WithValue(req.Context(), AuthSubjectContextKey, subject)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.UserInfo(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify response
	var resp UserInfoResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.ID != session.UserID {
		t.Errorf("ID = %v, want %v", resp.ID, session.UserID)
	}
	if resp.Username != session.Username {
		t.Errorf("Username = %v, want %v", resp.Username, session.Username)
	}
	if resp.Email != session.Email {
		t.Errorf("Email = %v, want %v", resp.Email, session.Email)
	}
	if len(resp.Roles) != len(session.Roles) {
		t.Errorf("Roles length = %v, want %v", len(resp.Roles), len(session.Roles))
	}
}

func TestAuthHandlers_UserInfo_Unauthenticated(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil)

	// Create request without session in context
	req := httptest.NewRequest(http.MethodGet, "/api/auth/userinfo", nil)
	w := httptest.NewRecorder()
	handlers.UserInfo(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthHandlers_Logout_WithSession(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil)

	// Create a session
	session := &Session{
		ID:        "session-123",
		UserID:    "user-abc",
		Username:  "testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err := store.Create(context.Background(), session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create request with session in context
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	subject := session.ToAuthSubject()
	subject.SessionID = session.ID
	ctx := context.WithValue(req.Context(), AuthSubjectContextKey, subject)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.Logout(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify session is deleted
	_, err = store.Get(context.Background(), session.ID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Session should be deleted, got error = %v", err)
	}
}

func TestAuthHandlers_Logout_WithoutSession(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil)

	// Create request without session in context
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	handlers.Logout(w, req)

	// Logout should still return OK even without session
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthHandlers_LogoutAll_WithSession(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil)

	// Create multiple sessions for the same user
	userID := "user-abc"
	for i := 0; i < 3; i++ {
		session := &Session{
			ID:        "session-" + string(rune('a'+i)),
			UserID:    userID,
			Username:  "testuser",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		err := store.Create(context.Background(), session)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
	}

	// Create request with one of the sessions in context
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout/all", nil)
	subject := &AuthSubject{
		ID:        userID,
		Username:  "testuser",
		SessionID: "session-a",
	}
	ctx := context.WithValue(req.Context(), AuthSubjectContextKey, subject)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.LogoutAll(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify all sessions are deleted
	sessions, err := store.GetByUserID(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetByUserID() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("All sessions should be deleted, got %d remaining", len(sessions))
	}
}

func TestAuthHandlers_Sessions_ListUserSessions(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil)

	// Create multiple sessions for the same user
	userID := "user-abc"
	for i := 0; i < 3; i++ {
		session := &Session{
			ID:        "session-" + string(rune('a'+i)),
			UserID:    userID,
			Username:  "testuser",
			Provider:  "oidc",
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		err := store.Create(context.Background(), session)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
	}

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/auth/sessions", nil)
	subject := &AuthSubject{
		ID:        userID,
		Username:  "testuser",
		SessionID: "session-a",
	}
	ctx := context.WithValue(req.Context(), AuthSubjectContextKey, subject)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.Sessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify response
	var resp SessionsResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Sessions) != 3 {
		t.Errorf("Sessions count = %d, want 3", len(resp.Sessions))
	}
}

func TestAuthHandlers_RevokeSession_Success(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil)

	// Create a session
	userID := "user-abc"
	session := &Session{
		ID:        "session-to-revoke",
		UserID:    userID,
		Username:  "testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err := store.Create(context.Background(), session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create request
	req := httptest.NewRequest(http.MethodDelete, "/api/auth/sessions/session-to-revoke", nil)
	subject := &AuthSubject{
		ID:        userID,
		Username:  "testuser",
		SessionID: "current-session",
	}
	ctx := context.WithValue(req.Context(), AuthSubjectContextKey, subject)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.RevokeSession(w, req, "session-to-revoke")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify session is deleted
	_, err = store.Get(context.Background(), "session-to-revoke")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Session should be deleted, got error = %v", err)
	}
}

func TestAuthHandlers_RevokeSession_NotOwned(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil)

	// Create a session for another user
	session := &Session{
		ID:        "other-user-session",
		UserID:    "other-user",
		Username:  "otheruser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err := store.Create(context.Background(), session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Try to revoke as a different user
	req := httptest.NewRequest(http.MethodDelete, "/api/auth/sessions/other-user-session", nil)
	subject := &AuthSubject{
		ID:        "user-abc",
		Username:  "testuser",
		SessionID: "current-session",
	}
	ctx := context.WithValue(req.Context(), AuthSubjectContextKey, subject)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.RevokeSession(w, req, "other-user-session")

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}

	// Verify session still exists
	_, err = store.Get(context.Background(), "other-user-session")
	if err != nil {
		t.Errorf("Session should still exist, got error = %v", err)
	}
}

func TestAuthHandlers_RevokeSession_AdminCanRevokeAny(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil)

	// Create a session for another user
	session := &Session{
		ID:        "other-user-session",
		UserID:    "other-user",
		Username:  "otheruser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err := store.Create(context.Background(), session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Admin user can revoke any session
	req := httptest.NewRequest(http.MethodDelete, "/api/auth/sessions/other-user-session", nil)
	subject := &AuthSubject{
		ID:        "admin-user",
		Username:  "admin",
		Roles:     []string{"admin"},
		SessionID: "admin-session",
	}
	ctx := context.WithValue(req.Context(), AuthSubjectContextKey, subject)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.RevokeSession(w, req, "other-user-session")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify session is deleted
	_, err = store.Get(context.Background(), "other-user-session")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Session should be deleted, got error = %v", err)
	}
}

// Response types for handler tests
type UserInfoResponse struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email,omitempty"`
	Roles    []string `json:"roles,omitempty"`
	Groups   []string `json:"groups,omitempty"`
	Provider string   `json:"provider,omitempty"`
}

type SessionsResponse struct {
	Sessions []SessionInfo `json:"sessions"`
}

type SessionInfo struct {
	ID             string    `json:"id"`
	Provider       string    `json:"provider"`
	CreatedAt      time.Time `json:"created_at"`
	LastAccessedAt time.Time `json:"last_accessed_at,omitempty"`
	Current        bool      `json:"current"`
}

// =====================================================
// OIDCLogin Tests
// =====================================================

func TestAuthHandlers_OIDCLogin_Configured(t *testing.T) {
	store := NewMemorySessionStore()
	config := &AuthHandlersConfig{
		OIDCLoginURL: "https://auth.example.com/authorize?client_id=test",
	}
	handlers := NewAuthHandlers(store, config)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/login", nil)
	w := httptest.NewRecorder()
	handlers.OIDCLogin(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	if location != config.OIDCLoginURL {
		t.Errorf("Location = %q, want %q", location, config.OIDCLoginURL)
	}
}

func TestAuthHandlers_OIDCLogin_NotConfigured(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil) // No OIDC config

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/login", nil)
	w := httptest.NewRecorder()
	handlers.OIDCLogin(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// =====================================================
// PlexLogin Tests
// =====================================================

func TestAuthHandlers_PlexLogin_Configured(t *testing.T) {
	store := NewMemorySessionStore()
	config := &AuthHandlersConfig{
		PlexLoginURL: "https://app.plex.tv/auth#?clientID=test",
	}
	handlers := NewAuthHandlers(store, config)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/plex/login", nil)
	w := httptest.NewRecorder()
	handlers.PlexLogin(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	if location != config.PlexLoginURL {
		t.Errorf("Location = %q, want %q", location, config.PlexLoginURL)
	}
}

func TestAuthHandlers_PlexLogin_NotConfigured(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil) // No Plex config

	req := httptest.NewRequest(http.MethodGet, "/api/auth/plex/login", nil)
	w := httptest.NewRecorder()
	handlers.PlexLogin(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// =====================================================
// HealthCheck Tests
// =====================================================

func TestAuthHandlers_HealthCheck(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/health", nil)
	w := httptest.NewRecorder()
	handlers.HealthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	err := json.NewDecoder(w.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "healthy" {
		t.Errorf("status = %q, want %q", resp["status"], "healthy")
	}
}

// =====================================================
// LogoutAll Edge Cases
// =====================================================

func TestAuthHandlers_LogoutAll_Unauthenticated(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout-all", nil)
	// No auth subject in context
	w := httptest.NewRecorder()
	handlers.LogoutAll(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthHandlers_LogoutAll_NoSessions(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil)

	// Create request with subject but no sessions exist
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout-all", nil)
	subject := &AuthSubject{
		ID:        "user-no-sessions",
		Username:  "testuser",
		SessionID: "current-session",
	}
	ctx := context.WithValue(req.Context(), AuthSubjectContextKey, subject)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.LogoutAll(w, req)

	// Should still return OK even if no sessions to delete
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// =====================================================
// Sessions Edge Cases
// =====================================================

func TestAuthHandlers_Sessions_Unauthenticated(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/sessions", nil)
	// No auth subject in context
	w := httptest.NewRecorder()
	handlers.Sessions(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthHandlers_Sessions_EmptyList(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/sessions", nil)
	subject := &AuthSubject{
		ID:        "user-no-sessions",
		Username:  "testuser",
		SessionID: "current-session",
	}
	ctx := context.WithValue(req.Context(), AuthSubjectContextKey, subject)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.Sessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp SessionsResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Empty list is valid
	if resp.Sessions == nil {
		t.Error("Sessions should be initialized (not nil)")
	}
}

// =====================================================
// RevokeSession Edge Cases
// =====================================================

func TestAuthHandlers_RevokeSession_Unauthenticated(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/auth/sessions/session-123", nil)
	// No auth subject in context
	w := httptest.NewRecorder()
	handlers.RevokeSession(w, req, "session-123")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthHandlers_RevokeSession_NotFound(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/auth/sessions/nonexistent", nil)
	subject := &AuthSubject{
		ID:        "user-abc",
		Username:  "testuser",
		SessionID: "current-session",
		Roles:     []string{"admin"}, // Admin to bypass ownership check
	}
	ctx := context.WithValue(req.Context(), AuthSubjectContextKey, subject)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.RevokeSession(w, req, "nonexistent")

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestAuthHandlers_RevokeSession_CurrentSession(t *testing.T) {
	store := NewMemorySessionStore()
	handlers := NewAuthHandlers(store, nil)

	// Create a session
	session := &Session{
		ID:        "current-session",
		UserID:    "user-abc",
		Username:  "testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err := store.Create(context.Background(), session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Try to revoke the current session
	req := httptest.NewRequest(http.MethodDelete, "/api/auth/sessions/current-session", nil)
	subject := &AuthSubject{
		ID:        "user-abc",
		Username:  "testuser",
		SessionID: "current-session", // Same as the one being revoked
	}
	ctx := context.WithValue(req.Context(), AuthSubjectContextKey, subject)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.RevokeSession(w, req, "current-session")

	// Should still allow revoking current session
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (should allow revoking current session)", w.Code, http.StatusOK)
	}
}
