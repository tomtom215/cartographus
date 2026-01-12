// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/auth"
)

// =====================================================
// Zero Trust Routes Tests
// ADR-0015: Zero Trust Authentication & Authorization
// =====================================================

func TestZeroTrustRouter_RegisterRoutes(t *testing.T) {
	// Create minimal test setup
	sessionStore := auth.NewMemorySessionStore()
	sessionMW := auth.NewSessionMiddleware(sessionStore, nil)
	authMiddleware := auth.NewMiddleware(nil, nil, string(auth.AuthModeNone), 100, time.Minute, true, nil, nil, "", "")

	flowConfig := &auth.FlowHandlersConfig{
		SessionDuration:          24 * time.Hour,
		DefaultPostLoginRedirect: "/",
	}
	flowHandlers := auth.NewFlowHandlers(nil, nil, sessionStore, sessionMW, flowConfig)

	router := NewZeroTrustRouter(&ZeroTrustRouterConfig{
		FlowHandlers:   flowHandlers,
		SessionMW:      sessionMW,
		AuthMiddleware: authMiddleware,
	})

	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	// Test that routes are registered
	tests := []struct {
		method string
		path   string
	}{
		{"GET", "/api/auth/oidc/login"},
		{"GET", "/api/auth/oidc/callback"},
		{"GET", "/api/auth/plex/login"},
		{"GET", "/api/auth/plex/poll"},
		{"POST", "/api/auth/plex/callback"},
		{"GET", "/api/auth/userinfo"},
		{"POST", "/api/auth/logout"},
		{"POST", "/api/auth/logout/all"},
		{"GET", "/api/auth/sessions"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			// Should not be 404 (route exists)
			// Various status codes are acceptable depending on the endpoint
			if w.Code == http.StatusNotFound {
				t.Errorf("Route %s %s should be registered", tt.method, tt.path)
			}
		})
	}
}

func TestZeroTrustRouter_SessionRequired(t *testing.T) {
	sessionStore := auth.NewMemorySessionStore()
	sessionMW := auth.NewSessionMiddleware(sessionStore, nil)
	authMiddleware := auth.NewMiddleware(nil, nil, string(auth.AuthModeNone), 100, time.Minute, true, nil, nil, "", "")

	flowConfig := &auth.FlowHandlersConfig{
		SessionDuration: 24 * time.Hour,
	}
	flowHandlers := auth.NewFlowHandlers(nil, nil, sessionStore, sessionMW, flowConfig)

	router := NewZeroTrustRouter(&ZeroTrustRouterConfig{
		FlowHandlers:   flowHandlers,
		SessionMW:      sessionMW,
		AuthMiddleware: authMiddleware,
	})

	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	// These endpoints require authentication
	protectedEndpoints := []struct {
		method string
		path   string
	}{
		{"POST", "/api/auth/logout/all"},
		{"GET", "/api/auth/sessions"},
	}

	for _, tt := range protectedEndpoints {
		t.Run(tt.method+" "+tt.path+" requires auth", func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d (unauthorized)", w.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestZeroTrustRouter_WithValidSession(t *testing.T) {
	sessionStore := auth.NewMemorySessionStore()
	sessionMW := auth.NewSessionMiddleware(sessionStore, nil)
	authMiddleware := auth.NewMiddleware(nil, nil, string(auth.AuthModeNone), 100, time.Minute, true, nil, nil, "", "")

	flowConfig := &auth.FlowHandlersConfig{
		SessionDuration: 24 * time.Hour,
	}
	flowHandlers := auth.NewFlowHandlers(nil, nil, sessionStore, sessionMW, flowConfig)

	router := NewZeroTrustRouter(&ZeroTrustRouterConfig{
		FlowHandlers:   flowHandlers,
		SessionMW:      sessionMW,
		AuthMiddleware: authMiddleware,
	})

	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	// Create a session
	session := &auth.Session{
		ID:        "test-session",
		UserID:    "user-123",
		Username:  "testuser",
		Email:     "test@example.com",
		Roles:     []string{"viewer"},
		Provider:  "test",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := sessionStore.Create(context.Background(), session); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Test userinfo with valid session
	req := httptest.NewRequest("GET", "/api/auth/userinfo", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: session.ID,
	})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["username"] != "testuser" {
		t.Errorf("username = %v, want testuser", resp["username"])
	}
}

func TestZeroTrustRouter_SessionsEndpoint(t *testing.T) {
	sessionStore := auth.NewMemorySessionStore()
	sessionMW := auth.NewSessionMiddleware(sessionStore, nil)
	authMiddleware := auth.NewMiddleware(nil, nil, string(auth.AuthModeNone), 100, time.Minute, true, nil, nil, "", "")

	flowHandlers := auth.NewFlowHandlers(nil, nil, sessionStore, sessionMW, nil)

	router := NewZeroTrustRouter(&ZeroTrustRouterConfig{
		FlowHandlers:   flowHandlers,
		SessionMW:      sessionMW,
		AuthMiddleware: authMiddleware,
	})

	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	// Create multiple sessions for a user
	userID := "user-123"
	for i := 0; i < 3; i++ {
		session := &auth.Session{
			ID:        "session-" + string(rune('a'+i)),
			UserID:    userID,
			Username:  "testuser",
			Provider:  "test",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		if err := sessionStore.Create(context.Background(), session); err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
	}

	req := httptest.NewRequest("GET", "/api/auth/sessions", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: "session-a",
	})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

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

func TestZeroTrustRouter_LogoutEndpoint(t *testing.T) {
	sessionStore := auth.NewMemorySessionStore()
	sessionMW := auth.NewSessionMiddleware(sessionStore, nil)
	authMiddleware := auth.NewMiddleware(nil, nil, string(auth.AuthModeNone), 100, time.Minute, true, nil, nil, "", "")

	flowHandlers := auth.NewFlowHandlers(nil, nil, sessionStore, sessionMW, nil)

	router := NewZeroTrustRouter(&ZeroTrustRouterConfig{
		FlowHandlers:   flowHandlers,
		SessionMW:      sessionMW,
		AuthMiddleware: authMiddleware,
	})

	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	// Create a session
	session := &auth.Session{
		ID:        "session-to-logout",
		UserID:    "user-123",
		Username:  "testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := sessionStore.Create(context.Background(), session); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: session.ID,
	})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify session was deleted
	_, err := sessionStore.Get(context.Background(), session.ID)
	if !errors.Is(err, auth.ErrSessionNotFound) {
		t.Error("Session should be deleted after logout")
	}
}

func TestZeroTrustRouter_RevokeSessionEndpoint(t *testing.T) {
	sessionStore := auth.NewMemorySessionStore()
	sessionMW := auth.NewSessionMiddleware(sessionStore, nil)
	authMiddleware := auth.NewMiddleware(nil, nil, string(auth.AuthModeNone), 100, time.Minute, true, nil, nil, "", "")

	flowHandlers := auth.NewFlowHandlers(nil, nil, sessionStore, sessionMW, nil)

	router := NewZeroTrustRouter(&ZeroTrustRouterConfig{
		FlowHandlers:   flowHandlers,
		SessionMW:      sessionMW,
		AuthMiddleware: authMiddleware,
	})

	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	userID := "user-123"

	// Create session to revoke
	targetSession := &auth.Session{
		ID:        "session-to-revoke",
		UserID:    userID,
		Username:  "testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := sessionStore.Create(context.Background(), targetSession); err != nil {
		t.Fatalf("Failed to create target session: %v", err)
	}

	// Create current session
	currentSession := &auth.Session{
		ID:        "current-session",
		UserID:    userID,
		Username:  "testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := sessionStore.Create(context.Background(), currentSession); err != nil {
		t.Fatalf("Failed to create current session: %v", err)
	}

	req := httptest.NewRequest("DELETE", "/api/auth/sessions/session-to-revoke", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: currentSession.ID,
	})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify target session was deleted
	_, err := sessionStore.Get(context.Background(), targetSession.ID)
	if !errors.Is(err, auth.ErrSessionNotFound) {
		t.Error("Target session should be deleted after revoke")
	}

	// Current session should still exist
	_, err = sessionStore.Get(context.Background(), currentSession.ID)
	if err != nil {
		t.Error("Current session should still exist")
	}
}

func TestZeroTrustRouter_OIDCLoginNotConfigured(t *testing.T) {
	sessionStore := auth.NewMemorySessionStore()
	sessionMW := auth.NewSessionMiddleware(sessionStore, nil)
	authMiddleware := auth.NewMiddleware(nil, nil, string(auth.AuthModeNone), 100, time.Minute, true, nil, nil, "", "")

	// FlowHandlers without OIDC flow configured
	flowHandlers := auth.NewFlowHandlers(nil, nil, sessionStore, sessionMW, nil)

	router := NewZeroTrustRouter(&ZeroTrustRouterConfig{
		FlowHandlers:   flowHandlers,
		SessionMW:      sessionMW,
		AuthMiddleware: authMiddleware,
	})

	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/auth/oidc/login", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d (service unavailable)", w.Code, http.StatusServiceUnavailable)
	}
}

func TestZeroTrustRouter_PlexLoginNotConfigured(t *testing.T) {
	sessionStore := auth.NewMemorySessionStore()
	sessionMW := auth.NewSessionMiddleware(sessionStore, nil)
	authMiddleware := auth.NewMiddleware(nil, nil, string(auth.AuthModeNone), 100, time.Minute, true, nil, nil, "", "")

	// FlowHandlers without Plex flow configured
	flowHandlers := auth.NewFlowHandlers(nil, nil, sessionStore, sessionMW, nil)

	router := NewZeroTrustRouter(&ZeroTrustRouterConfig{
		FlowHandlers:   flowHandlers,
		SessionMW:      sessionMW,
		AuthMiddleware: authMiddleware,
	})

	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/auth/plex/login", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d (service unavailable)", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleRevokeSession_InvalidPath(t *testing.T) {
	sessionStore := auth.NewMemorySessionStore()
	sessionMW := auth.NewSessionMiddleware(sessionStore, nil)
	authMiddleware := auth.NewMiddleware(nil, nil, string(auth.AuthModeNone), 100, time.Minute, true, nil, nil, "", "")

	flowHandlers := auth.NewFlowHandlers(nil, nil, sessionStore, sessionMW, nil)

	router := &ZeroTrustRouter{
		flowHandlers:   flowHandlers,
		sessionMW:      sessionMW,
		authMiddleware: authMiddleware,
	}

	// Test with invalid path
	req := httptest.NewRequest("DELETE", "/api/auth/invalid/path", nil)
	w := httptest.NewRecorder()

	router.handleRevokeSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRevokeSession_MissingSessionID(t *testing.T) {
	sessionStore := auth.NewMemorySessionStore()
	sessionMW := auth.NewSessionMiddleware(sessionStore, nil)
	authMiddleware := auth.NewMiddleware(nil, nil, string(auth.AuthModeNone), 100, time.Minute, true, nil, nil, "", "")

	flowHandlers := auth.NewFlowHandlers(nil, nil, sessionStore, sessionMW, nil)

	router := &ZeroTrustRouter{
		flowHandlers:   flowHandlers,
		sessionMW:      sessionMW,
		authMiddleware: authMiddleware,
	}

	// Test with empty session ID
	req := httptest.NewRequest("DELETE", "/api/auth/sessions/", nil)

	// Add auth context
	session := &auth.Session{
		ID:        "current-session",
		UserID:    "user-123",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	sessionStore.Create(context.Background(), session)

	subject := &auth.AuthSubject{
		ID:        "user-123",
		SessionID: session.ID,
	}
	ctx := context.WithValue(req.Context(), auth.AuthSubjectContextKey, subject)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	router.handleRevokeSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRegisterZeroTrustRoutes_Helper(t *testing.T) {
	sessionStore := auth.NewMemorySessionStore()
	sessionMW := auth.NewSessionMiddleware(sessionStore, nil)
	authMiddleware := auth.NewMiddleware(nil, nil, string(auth.AuthModeNone), 100, time.Minute, true, nil, nil, "", "")
	flowHandlers := auth.NewFlowHandlers(nil, nil, sessionStore, sessionMW, nil)

	mux := http.NewServeMux()
	RegisterZeroTrustRoutes(mux, flowHandlers, nil, sessionMW, authMiddleware)

	// Verify routes are registered
	req := httptest.NewRequest("GET", "/api/auth/userinfo", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should not be 404 (though it might be 401 unauthorized)
	if w.Code == http.StatusNotFound {
		t.Error("Routes should be registered via helper function")
	}
}

func TestZeroTrustRouter_NilHandlers(t *testing.T) {
	router := NewZeroTrustRouter(&ZeroTrustRouterConfig{
		FlowHandlers:   nil,
		PolicyHandlers: nil,
		SessionMW:      nil,
		AuthMiddleware: nil,
	})

	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	// Should not panic with nil handlers
	req := httptest.NewRequest("GET", "/api/auth/userinfo", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// With no routes registered, should get 404
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d (no routes registered)", w.Code, http.StatusNotFound)
	}
}

func TestHandleRolePermissions_InvalidPath(t *testing.T) {
	router := &ZeroTrustRouter{}

	tests := []struct {
		name string
		path string
	}{
		{"invalid prefix", "/api/invalid/path"},
		{"missing permissions", "/api/admin/roles/admin"},
		{"empty role", "/api/admin/roles//permissions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router.handleRolePermissions(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestZeroTrustRouter_Integration(t *testing.T) {
	// Create full test setup with session management (OIDC requires live IdP discovery)
	// ADR-0015: Zitadel OIDC requires live OIDC discovery, so we test graceful handling
	// when OIDC is not configured (nil flow).
	sessionStore := auth.NewMemorySessionStore()
	sessionMW := auth.NewSessionMiddleware(sessionStore, &auth.SessionMiddlewareConfig{
		CookieName:     "session",
		SessionTTL:     24 * time.Hour,
		SlidingSession: true,
	})
	authMiddleware := auth.NewMiddleware(nil, nil, string(auth.AuthModeNone), 100, time.Minute, true, nil, nil, "", "")

	// Create flow handlers without OIDC (nil) - Zitadel requires live IdP discovery
	// which cannot be done in unit tests. OIDC endpoints should return 503 Service Unavailable.
	flowHandlers := auth.NewFlowHandlers(nil, nil, sessionStore, sessionMW, nil)

	router := NewZeroTrustRouter(&ZeroTrustRouterConfig{
		FlowHandlers:   flowHandlers,
		SessionMW:      sessionMW,
		AuthMiddleware: authMiddleware,
	})

	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	// Test OIDC login returns 503 when OIDC is not configured
	// ADR-0015: Zitadel OIDC flow returns 503 Service Unavailable when not initialized
	req := httptest.NewRequest("GET", "/api/auth/oidc/login?redirect_uri=/dashboard", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("OIDC login status = %d, want %d (OIDC not configured)", w.Code, http.StatusServiceUnavailable)
	}

	// Test userinfo endpoint (should work even without OIDC)
	req = httptest.NewRequest("GET", "/api/auth/userinfo", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should return 401 Unauthorized when not authenticated
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Userinfo status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
