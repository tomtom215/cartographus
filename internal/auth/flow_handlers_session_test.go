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
// Session Management Tests
// ADR-0015: Zero Trust Authentication & Authorization
// =====================================================
// Tests for session listing, revocation, logout, and logout all.

func TestFlowHandlers_Sessions(t *testing.T) {
	setup := setupBasicHandlers(t)
	userID := "user-abc"
	createMultipleSessions(t, setup.sessionStore, userID, "testuser", 3)

	subject := &AuthSubject{
		ID:        userID,
		Username:  "testuser",
		SessionID: "session-a",
	}
	ctx := createAuthContext(subject)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/sessions", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	setup.handlers.Sessions(w, req)

	assertStatusCode(t, w, http.StatusOK)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	sessions, ok := resp["sessions"].([]interface{})
	if !ok {
		t.Fatal("Response should contain sessions array")
	}
	if len(sessions) != 3 {
		t.Errorf("sessions count = %d, want 3", len(sessions))
	}
}

func TestFlowHandlers_Sessions_Unauthenticated(t *testing.T) {
	setup := setupBasicHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/sessions", nil)
	w := httptest.NewRecorder()

	setup.handlers.Sessions(w, req)

	assertStatusCode(t, w, http.StatusUnauthorized)
}

// =====================================================
// Session Revocation Tests
// =====================================================

func TestFlowHandlers_RevokeSession(t *testing.T) {
	setup := setupBasicHandlers(t)

	userID := "user-abc"
	session := &Session{
		ID:        "session-to-revoke",
		UserID:    userID,
		Username:  "testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	createTestSession(t, setup.sessionStore, session)

	subject := &AuthSubject{
		ID:        userID,
		Username:  "testuser",
		SessionID: "other-session",
	}
	ctx := createAuthContext(subject)
	req := httptest.NewRequest(http.MethodDelete, "/api/auth/sessions/session-to-revoke", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	setup.handlers.RevokeSession(w, req, "session-to-revoke")

	assertStatusCode(t, w, http.StatusOK)

	_, err := setup.sessionStore.Get(context.Background(), session.ID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Error("Session should be deleted after revoke")
	}
}

func TestFlowHandlers_RevokeSession_Forbidden(t *testing.T) {
	setup := setupBasicHandlers(t)

	session := &Session{
		ID:        "other-user-session",
		UserID:    "other-user",
		Username:  "otheruser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	createTestSession(t, setup.sessionStore, session)

	subject := &AuthSubject{
		ID:        "user-abc",
		Username:  "testuser",
		Roles:     []string{"viewer"},
		SessionID: "my-session",
	}
	ctx := createAuthContext(subject)
	req := httptest.NewRequest(http.MethodDelete, "/api/auth/sessions/other-user-session", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	setup.handlers.RevokeSession(w, req, "other-user-session")

	assertStatusCode(t, w, http.StatusForbidden)
}

func TestFlowHandlers_RevokeSession_NotFound(t *testing.T) {
	setup := setupBasicHandlers(t)

	subject := &AuthSubject{
		ID:        "user-abc",
		Username:  "testuser",
		SessionID: "my-session",
	}
	ctx := createAuthContext(subject)
	req := httptest.NewRequest(http.MethodDelete, "/api/auth/sessions/nonexistent", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	setup.handlers.RevokeSession(w, req, "nonexistent")

	assertStatusCode(t, w, http.StatusNotFound)
}

func TestFlowHandlers_RevokeSession_AdminCanRevokeOthers(t *testing.T) {
	setup := setupBasicHandlers(t)

	otherSession := &Session{
		ID:        "other-user-session",
		UserID:    "other-user",
		Username:  "otheruser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	createTestSession(t, setup.sessionStore, otherSession)

	subject := &AuthSubject{
		ID:        "admin-user",
		Username:  "admin",
		Roles:     []string{"admin"},
		SessionID: "admin-session",
	}
	ctx := createAuthContext(subject)
	req := httptest.NewRequest(http.MethodDelete, "/api/auth/sessions/other-user-session", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	setup.handlers.RevokeSession(w, req, "other-user-session")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify session was deleted
	_, err := setup.sessionStore.Get(context.Background(), "other-user-session")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Error("session should be deleted")
	}
}

func TestFlowHandlers_RevokeSession_Unauthenticated(t *testing.T) {
	setup := setupBasicHandlers(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/auth/sessions/some-session", nil)
	w := httptest.NewRecorder()

	setup.handlers.RevokeSession(w, req, "some-session")

	assertStatusCode(t, w, http.StatusUnauthorized)
}

// =====================================================
// Logout Tests
// =====================================================

func TestFlowHandlers_Logout(t *testing.T) {
	setup := setupBasicHandlers(t)

	session := &Session{
		ID:        "session-123",
		UserID:    "user-abc",
		Username:  "testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	createTestSession(t, setup.sessionStore, session)

	subject := &AuthSubject{
		ID:        "user-abc",
		Username:  "testuser",
		SessionID: session.ID,
	}
	ctx := createAuthContext(subject)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	setup.handlers.Logout(w, req)

	assertStatusCode(t, w, http.StatusOK)

	_, err := setup.sessionStore.Get(context.Background(), session.ID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Error("Session should be deleted after logout")
	}

	cookies := w.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "session" && cookie.MaxAge != -1 {
			t.Error("Session cookie should be cleared")
		}
	}
}

func TestFlowHandlers_Logout_NoSession(t *testing.T) {
	setup := setupBasicHandlers(t)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	w := httptest.NewRecorder()

	setup.handlers.Logout(w, req)

	// Should still return OK (clears cookie)
	assertStatusCode(t, w, http.StatusOK)
}

// =====================================================
// Logout All Tests
// =====================================================

func TestFlowHandlers_LogoutAll_Success(t *testing.T) {
	setup := setupBasicHandlers(t)
	userID := "user-abc"
	createMultipleSessions(t, setup.sessionStore, userID, "testuser", 3)

	subject := &AuthSubject{
		ID:        userID,
		Username:  "testuser",
		SessionID: "session-a",
	}
	ctx := createAuthContext(subject)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout/all", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	setup.handlers.LogoutAll(w, req)

	assertStatusCode(t, w, http.StatusOK)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp["sessions_count"].(float64) != 3 {
		t.Errorf("sessions_count = %v, want 3", resp["sessions_count"])
	}

	// Verify all sessions are deleted
	sessions, err := setup.sessionStore.GetByUserID(context.Background(), userID)
	if err != nil {
		t.Fatalf("get sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected all sessions deleted, got %d", len(sessions))
	}
}

func TestFlowHandlers_LogoutAll_Unauthenticated(t *testing.T) {
	setup := setupBasicHandlers(t)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout/all", nil)
	w := httptest.NewRecorder()

	setup.handlers.LogoutAll(w, req)

	assertStatusCode(t, w, http.StatusUnauthorized)
}

// =====================================================
// Session Cookie Extraction Tests
// =====================================================

func TestFlowHandlers_ExtractSessionIDFromCookie(t *testing.T) {
	sessionStore := NewMemorySessionStore()
	sessionMW := NewSessionMiddleware(sessionStore, &SessionMiddlewareConfig{
		CookieName: "custom_session",
	})

	handlers := NewFlowHandlers(nil, nil, sessionStore, sessionMW, nil)

	t.Run("cookie present", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "custom_session",
			Value: "session-xyz",
		})

		sessionID := handlers.extractSessionIDFromCookie(req)
		if sessionID != "session-xyz" {
			t.Errorf("sessionID = %s, want session-xyz", sessionID)
		}
	})

	t.Run("cookie absent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		sessionID := handlers.extractSessionIDFromCookie(req)
		if sessionID != "" {
			t.Errorf("sessionID = %s, want empty", sessionID)
		}
	})

	t.Run("default cookie name", func(t *testing.T) {
		// Create handler without session middleware to test default
		handlersNoMW := NewFlowHandlers(nil, nil, sessionStore, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "tautulli_session",
			Value: "session-default",
		})

		sessionID := handlersNoMW.extractSessionIDFromCookie(req)
		if sessionID != "session-default" {
			t.Errorf("sessionID = %s, want session-default", sessionID)
		}
	})
}
