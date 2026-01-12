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
)

// =====================================================
// Session Cookie Middleware Tests
// ADR-0015: Zero Trust Authentication & Authorization
// =====================================================

func TestSessionMiddleware_ValidSession(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		SessionTTL: 24 * time.Hour,
	})

	// Create a session
	session := &Session{
		ID:        "session-123",
		UserID:    "user-abc",
		Username:  "testuser",
		Email:     "test@example.com",
		Roles:     []string{"viewer"},
		Provider:  "oidc",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Create(context.Background(), session); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create a handler that checks for AuthSubject
	var capturedSubject *AuthSubject
	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedSubject = GetAuthSubject(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// Create request with session cookie
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: session.ID,
	})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if capturedSubject == nil {
		t.Fatal("AuthSubject should be set in context")
	}
	if capturedSubject.ID != session.UserID {
		t.Errorf("AuthSubject.ID = %s, want %s", capturedSubject.ID, session.UserID)
	}
	if capturedSubject.Username != session.Username {
		t.Errorf("AuthSubject.Username = %s, want %s", capturedSubject.Username, session.Username)
	}
	if capturedSubject.SessionID != session.ID {
		t.Errorf("AuthSubject.SessionID = %s, want %s", capturedSubject.SessionID, session.ID)
	}
}

func TestSessionMiddleware_NoSession(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		SessionTTL: 24 * time.Hour,
	})

	// Create a handler that checks for AuthSubject
	var capturedSubject *AuthSubject
	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedSubject = GetAuthSubject(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// Create request without session cookie
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Without RequireAuth, request should proceed but without AuthSubject
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if capturedSubject != nil {
		t.Error("AuthSubject should be nil when no session")
	}
}

func TestSessionMiddleware_InvalidSession(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		SessionTTL: 24 * time.Hour,
	})

	var capturedSubject *AuthSubject
	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedSubject = GetAuthSubject(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// Create request with invalid session cookie
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: "nonexistent-session",
	})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should proceed without AuthSubject
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if capturedSubject != nil {
		t.Error("AuthSubject should be nil for invalid session")
	}
}

func TestSessionMiddleware_ExpiredSession(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		SessionTTL: 24 * time.Hour,
	})

	// Create an expired session
	session := &Session{
		ID:        "expired-session",
		UserID:    "user-abc",
		Username:  "testuser",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}
	if err := store.Create(context.Background(), session); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	var capturedSubject *AuthSubject
	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedSubject = GetAuthSubject(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: session.ID,
	})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should proceed without AuthSubject for expired session
	if capturedSubject != nil {
		t.Error("AuthSubject should be nil for expired session")
	}
}

func TestSessionMiddleware_RequireAuth_Valid(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		SessionTTL: 24 * time.Hour,
	})

	// Create a session
	session := &Session{
		ID:        "session-123",
		UserID:    "user-abc",
		Username:  "testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Create(context.Background(), session); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	handler := mw.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: session.ID,
	})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSessionMiddleware_RequireAuth_NoSession(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		SessionTTL: 24 * time.Hour,
	})

	handler := mw.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestSessionMiddleware_RequireRole_Valid(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		SessionTTL: 24 * time.Hour,
	})

	// Create a session with admin role
	session := &Session{
		ID:        "session-123",
		UserID:    "user-abc",
		Username:  "admin",
		Roles:     []string{"admin"},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Create(context.Background(), session); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	handler := mw.RequireRole("admin", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: session.ID,
	})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSessionMiddleware_RequireRole_Forbidden(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		SessionTTL: 24 * time.Hour,
	})

	// Create a session with viewer role
	session := &Session{
		ID:        "session-123",
		UserID:    "user-abc",
		Username:  "viewer",
		Roles:     []string{"viewer"},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Create(context.Background(), session); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	handler := mw.RequireRole("admin", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: session.ID,
	})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestSessionMiddleware_RequireAnyRole(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		SessionTTL: 24 * time.Hour,
	})

	// Create a session with editor role
	session := &Session{
		ID:        "session-123",
		UserID:    "user-abc",
		Username:  "editor",
		Roles:     []string{"editor"},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Create(context.Background(), session); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	handler := mw.RequireAnyRole([]string{"admin", "editor"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: session.ID,
	})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSessionMiddleware_TouchesSession(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName:     "session",
		SessionTTL:     24 * time.Hour,
		SlidingSession: true,
	})

	// Create a session
	originalExpiry := time.Now().Add(1 * time.Hour)
	session := &Session{
		ID:             "session-123",
		UserID:         "user-abc",
		Username:       "testuser",
		CreatedAt:      time.Now().Add(-10 * time.Minute),
		ExpiresAt:      originalExpiry,
		LastAccessedAt: time.Now().Add(-10 * time.Minute),
	}
	if err := store.Create(context.Background(), session); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: session.ID,
	})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify session was touched (expiry extended)
	updated, err := store.Get(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	// With sliding sessions, expiry should be extended
	if !updated.ExpiresAt.After(originalExpiry) {
		t.Error("Session expiry should be extended with sliding sessions")
	}
}

func TestSessionMiddleware_HeaderToken(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		HeaderName: "X-Session-Token",
		SessionTTL: 24 * time.Hour,
	})

	// Create a session
	session := &Session{
		ID:        "session-123",
		UserID:    "user-abc",
		Username:  "testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Create(context.Background(), session); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	var capturedSubject *AuthSubject
	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedSubject = GetAuthSubject(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// Create request with session in header (not cookie)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Session-Token", session.ID)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if capturedSubject == nil {
		t.Fatal("AuthSubject should be set when using header token")
	}
	if capturedSubject.SessionID != session.ID {
		t.Errorf("AuthSubject.SessionID = %s, want %s", capturedSubject.SessionID, session.ID)
	}
}

func TestSessionMiddlewareConfig_Defaults(t *testing.T) {
	config := DefaultSessionMiddlewareConfig()

	if config.CookieName != "session" {
		t.Errorf("CookieName = %s, want session", config.CookieName)
	}
	if config.SessionTTL != 24*time.Hour {
		t.Errorf("SessionTTL = %v, want 24h", config.SessionTTL)
	}
	if !config.CookieSecure {
		t.Error("CookieSecure should default to true")
	}
	if !config.CookieHTTPOnly {
		t.Error("CookieHTTPOnly should default to true")
	}
}

// =====================================================
// Phase 4A.3: Session Fixation Protection Tests
// =====================================================

func TestSessionMiddleware_CreateSessionWithOldID_DeletesOldSession(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		SessionTTL: 24 * time.Hour,
	})

	// Create an old session
	oldSession := &Session{
		ID:        "old-session-id",
		UserID:    "user-abc",
		Username:  "testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Create(context.Background(), oldSession); err != nil {
		t.Fatalf("Failed to create old session: %v", err)
	}

	// Verify old session exists
	_, err := store.Get(context.Background(), "old-session-id")
	if err != nil {
		t.Fatalf("Old session should exist: %v", err)
	}

	// Create new session with old session ID
	subject := &AuthSubject{
		ID:       "user-abc",
		Username: "testuser",
		Roles:    []string{"viewer"},
	}

	w := httptest.NewRecorder()
	newSession, err := mw.CreateSessionWithOldID(context.Background(), w, subject, "old-session-id")
	if err != nil {
		t.Fatalf("CreateSessionWithOldID error: %v", err)
	}

	// Verify new session was created with different ID
	if newSession.ID == "old-session-id" {
		t.Error("New session should have different ID than old session")
	}

	// Verify old session was deleted
	_, err = store.Get(context.Background(), "old-session-id")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Old session should be deleted, got error: %v", err)
	}

	// Verify new session exists
	_, err = store.Get(context.Background(), newSession.ID)
	if err != nil {
		t.Fatalf("New session should exist: %v", err)
	}
}

func TestSessionMiddleware_CreateSessionWithOldID_NoOldSession(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		SessionTTL: 24 * time.Hour,
	})

	subject := &AuthSubject{
		ID:       "user-abc",
		Username: "testuser",
		Roles:    []string{"viewer"},
	}

	// Create session without old session ID
	w := httptest.NewRecorder()
	newSession, err := mw.CreateSessionWithOldID(context.Background(), w, subject, "")
	if err != nil {
		t.Fatalf("CreateSessionWithOldID error: %v", err)
	}

	// Verify session was created
	if newSession == nil {
		t.Fatal("Session should be created")
	}

	_, err = store.Get(context.Background(), newSession.ID)
	if err != nil {
		t.Fatalf("New session should exist: %v", err)
	}
}

func TestSessionMiddleware_CreateSessionWithOldID_PreservesSubjectData(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		SessionTTL: 24 * time.Hour,
	})

	subject := &AuthSubject{
		ID:       "user-abc",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"viewer", "editor"},
		Groups:   []string{"team-a", "team-b"},
		Metadata: map[string]string{
			"access_token":  "token123",
			"refresh_token": "refresh123",
		},
	}

	w := httptest.NewRecorder()
	session, err := mw.CreateSessionWithOldID(context.Background(), w, subject, "")
	if err != nil {
		t.Fatalf("CreateSessionWithOldID error: %v", err)
	}

	// Verify all subject data is preserved
	if session.UserID != "user-abc" {
		t.Errorf("UserID = %s, want user-abc", session.UserID)
	}
	if session.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", session.Username)
	}
	if session.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", session.Email)
	}
	if len(session.Roles) != 2 {
		t.Errorf("Roles count = %d, want 2", len(session.Roles))
	}
	if len(session.Groups) != 2 {
		t.Errorf("Groups count = %d, want 2", len(session.Groups))
	}
	if session.Metadata["access_token"] != "token123" {
		t.Errorf("access_token = %s, want token123", session.Metadata["access_token"])
	}
}

func TestSessionMiddleware_GetCookieName(t *testing.T) {
	mw := NewSessionMiddleware(NewMemorySessionStore(), &SessionMiddlewareConfig{
		CookieName: "custom_session_cookie",
		SessionTTL: 24 * time.Hour,
	})

	if mw.GetCookieName() != "custom_session_cookie" {
		t.Errorf("GetCookieName = %s, want custom_session_cookie", mw.GetCookieName())
	}
}

// =====================================================
// Phase 4C: Additional Coverage Tests
// =====================================================

func TestSessionMiddleware_CreateSession(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		SessionTTL: 24 * time.Hour,
	})

	subject := &AuthSubject{
		ID:       "user-123",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"viewer"},
	}

	w := httptest.NewRecorder()
	session, err := mw.CreateSession(context.Background(), w, subject)
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}

	// Verify session was created
	if session == nil {
		t.Fatal("session should not be nil")
	}
	if session.UserID != subject.ID {
		t.Errorf("UserID = %s, want %s", session.UserID, subject.ID)
	}

	// Verify cookie was set
	cookies := w.Result().Cookies()
	foundCookie := false
	for _, cookie := range cookies {
		if cookie.Name == "session" && cookie.Value == session.ID {
			foundCookie = true
			break
		}
	}
	if !foundCookie {
		t.Error("session cookie should be set")
	}

	// Verify session exists in store
	_, err = store.Get(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("session should exist in store: %v", err)
	}
}

func TestSessionMiddleware_RequireAnyRole_Unauthenticated(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		SessionTTL: 24 * time.Hour,
	})

	handler := mw.RequireAnyRole([]string{"admin", "editor"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request without authentication
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestSessionMiddleware_RequireAnyRole_Forbidden(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		SessionTTL: 24 * time.Hour,
	})

	// Create session with wrong role
	session := &Session{
		ID:        "session-123",
		UserID:    "user-abc",
		Username:  "viewer",
		Roles:     []string{"viewer"}, // Not admin or editor
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Create(context.Background(), session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	handler := mw.RequireAnyRole([]string{"admin", "editor"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: session.ID,
	})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestSessionMiddleware_RequireRole_Unauthenticated(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		SessionTTL: 24 * time.Hour,
	})

	handler := mw.RequireRole("admin", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request without authentication
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestSessionMiddleware_DestroySession(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName: "session",
		SessionTTL: 24 * time.Hour,
	})

	// Create a session
	session := &Session{
		ID:        "session-to-destroy",
		UserID:    "user-abc",
		Username:  "testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Create(context.Background(), session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	w := httptest.NewRecorder()
	err := mw.DestroySession(context.Background(), w, session.ID)
	if err != nil {
		t.Fatalf("DestroySession error: %v", err)
	}

	// Verify session was deleted
	_, err = store.Get(context.Background(), session.ID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("session should be deleted, got err: %v", err)
	}

	// Verify cookie was cleared
	cookies := w.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "session" && cookie.MaxAge != -1 {
			t.Error("session cookie should be cleared (MaxAge = -1)")
		}
	}
}

func TestSessionMiddleware_SetAndClearSessionCookie(t *testing.T) {
	store := NewMemorySessionStore()
	mw := NewSessionMiddleware(store, &SessionMiddlewareConfig{
		CookieName:     "test_session",
		CookiePath:     "/app",
		CookieDomain:   "example.com",
		SessionTTL:     24 * time.Hour,
		CookieSecure:   true,
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteStrictMode,
	})

	t.Run("set cookie", func(t *testing.T) {
		w := httptest.NewRecorder()
		mw.SetSessionCookie(w, "session-id-123")

		cookies := w.Result().Cookies()
		if len(cookies) == 0 {
			t.Fatal("expected cookie to be set")
		}

		cookie := cookies[0]
		if cookie.Name != "test_session" {
			t.Errorf("cookie name = %s, want test_session", cookie.Name)
		}
		if cookie.Value != "session-id-123" {
			t.Errorf("cookie value = %s, want session-id-123", cookie.Value)
		}
		if cookie.Path != "/app" {
			t.Errorf("cookie path = %s, want /app", cookie.Path)
		}
		if cookie.Domain != "example.com" {
			t.Errorf("cookie domain = %s, want example.com", cookie.Domain)
		}
		if !cookie.Secure {
			t.Error("cookie Secure should be true")
		}
		if !cookie.HttpOnly {
			t.Error("cookie HttpOnly should be true")
		}
	})

	t.Run("clear cookie", func(t *testing.T) {
		w := httptest.NewRecorder()
		mw.ClearSessionCookie(w)

		cookies := w.Result().Cookies()
		if len(cookies) == 0 {
			t.Fatal("expected cookie to be cleared")
		}

		cookie := cookies[0]
		if cookie.MaxAge != -1 {
			t.Errorf("cookie MaxAge = %d, want -1", cookie.MaxAge)
		}
	})
}

func TestMemorySessionStore_StartCleanupRoutine(t *testing.T) {
	store := NewMemorySessionStore()

	// Create an expired session
	expiredSession := &Session{
		ID:        "expired-session",
		UserID:    "user-abc",
		Username:  "testuser",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}
	if err := store.Create(context.Background(), expiredSession); err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Create a valid session
	validSession := &Session{
		ID:        "valid-session",
		UserID:    "user-xyz",
		Username:  "testuser2",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Create(context.Background(), validSession); err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Start cleanup routine with short interval
	done := store.StartCleanupRoutine(50 * time.Millisecond)
	defer close(done)

	// Wait for cleanup to run
	time.Sleep(150 * time.Millisecond)

	// Expired session should be gone (note: direct access bypasses expiry check)
	store.mu.RLock()
	_, expiredExists := store.sessions["expired-session"]
	_, validExists := store.sessions["valid-session"]
	store.mu.RUnlock()

	if expiredExists {
		t.Error("expired session should be deleted by cleanup routine")
	}
	if !validExists {
		t.Error("valid session should still exist")
	}
}
