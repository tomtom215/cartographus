// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
)

// =====================================================
// Plex Flow Handler Tests
// ADR-0015: Zero Trust Authentication & Authorization
// =====================================================
// Tests for Plex login, poll, and callback handlers.

func TestFlowHandlers_PlexLogin(t *testing.T) {
	// Mock Plex PIN API
	pinServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":         12345,
			"code":       "ABC1",
			"expires_at": time.Now().Add(5 * time.Minute).Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer pinServer.Close()

	plexConfig := &PlexFlowConfig{
		ClientID:   "test-client",
		Product:    "Test App",
		PINTimeout: 5 * time.Minute,
	}

	plexStore := NewMemoryPlexPINStore()
	plexFlow := NewPlexFlow(plexConfig, plexStore)
	plexFlow.SetPINEndpoint(pinServer.URL)

	sessionStore := NewMemorySessionStore()
	sessionMW := NewSessionMiddleware(sessionStore, nil)

	handlers := NewFlowHandlers(nil, plexFlow, sessionStore, sessionMW, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/plex/login", nil)
	w := httptest.NewRecorder()

	handlers.PlexLogin(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["pin_id"] == nil {
		t.Error("Response should contain pin_id")
	}
	if resp["pin_code"] == nil {
		t.Error("Response should contain pin_code")
	}
	if resp["auth_url"] == nil {
		t.Error("Response should contain auth_url")
	}
}

func TestFlowHandlers_PlexLogin_NotConfigured(t *testing.T) {
	setup := setupBasicHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/plex/login", nil)
	w := httptest.NewRecorder()

	setup.handlers.PlexLogin(w, req)

	assertStatusCode(t, w, http.StatusServiceUnavailable)
}

// =====================================================
// Plex Poll Tests
// =====================================================

func TestFlowHandlers_PlexPoll_NotAuthorized(t *testing.T) {
	pinServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":         12345,
			"code":       "ABC1",
			"auth_token": nil,
			"expires_at": time.Now().Add(5 * time.Minute).Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer pinServer.Close()

	plexConfig := &PlexFlowConfig{
		ClientID:   "test-client",
		Product:    "Test App",
		PINTimeout: 5 * time.Minute,
	}

	plexStore := NewMemoryPlexPINStore()
	plexFlow := NewPlexFlow(plexConfig, plexStore)
	plexFlow.SetPINCheckEndpoint(pinServer.URL)

	sessionStore := NewMemorySessionStore()
	sessionMW := NewSessionMiddleware(sessionStore, nil)

	handlers := NewFlowHandlers(nil, plexFlow, sessionStore, sessionMW, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/plex/poll?pin_id=12345", nil)
	w := httptest.NewRecorder()

	handlers.PlexPoll(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["authorized"] != false {
		t.Error("authorized should be false")
	}
}

func TestFlowHandlers_PlexPoll_MissingPinID(t *testing.T) {
	plexConfig := &PlexFlowConfig{
		ClientID: "test-client",
		Product:  "Test App",
	}

	plexStore := NewMemoryPlexPINStore()
	plexFlow := NewPlexFlow(plexConfig, plexStore)

	sessionStore := NewMemorySessionStore()
	sessionMW := NewSessionMiddleware(sessionStore, nil)

	handlers := NewFlowHandlers(nil, plexFlow, sessionStore, sessionMW, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/plex/poll", nil)
	w := httptest.NewRecorder()

	handlers.PlexPoll(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestFlowHandlers_PlexPoll_InvalidPinID(t *testing.T) {
	setup := setupPlexHandlers(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/plex/poll?pin_id=not-a-number", nil)
	w := httptest.NewRecorder()

	setup.handlers.PlexPoll(w, req)

	assertStatusCode(t, w, http.StatusBadRequest)
}

func TestFlowHandlers_PlexPoll_NotConfigured(t *testing.T) {
	setup := setupBasicHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/plex/poll?pin_id=12345", nil)
	w := httptest.NewRecorder()

	setup.handlers.PlexPoll(w, req)

	assertStatusCode(t, w, http.StatusServiceUnavailable)
}

// =====================================================
// Plex Callback Tests
// =====================================================

func TestFlowHandlers_PlexCallback_Success(t *testing.T) {
	// Mock Plex PIN check and user info endpoints
	pinCheckServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pins/") {
			// PIN check - return authorized
			authToken := "plex-auth-token-123"
			resp := map[string]interface{}{
				"id":         12345,
				"code":       "ABC1",
				"auth_token": &authToken,
				"expires_at": time.Now().Add(5 * time.Minute).Format(time.RFC3339),
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/user" {
			// User info endpoint
			resp := map[string]interface{}{
				"user": map[string]interface{}{
					"id":       123,
					"uuid":     "uuid-123",
					"username": "plexuser",
					"email":    "plex@example.com",
					"subscription": map[string]interface{}{
						"active": true,
						"status": "Active",
						"plan":   "lifetime",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer pinCheckServer.Close()

	plexConfig := &PlexFlowConfig{
		ClientID:     "test-client",
		Product:      "Test App",
		PINTimeout:   5 * time.Minute,
		DefaultRoles: []string{"viewer"},
		PlexPassRole: "plexpass",
	}

	plexStore := NewMemoryPlexPINStore()
	plexFlow := NewPlexFlow(plexConfig, plexStore)
	plexFlow.SetPINCheckEndpoint(pinCheckServer.URL + "/pins")
	plexFlow.SetUserInfoEndpoint(pinCheckServer.URL + "/user")

	sessionStore := NewMemorySessionStore()
	sessionMW := NewSessionMiddleware(sessionStore, nil)

	config := &FlowHandlersConfig{
		DefaultPostLoginRedirect: "/",
	}
	handlers := NewFlowHandlers(nil, plexFlow, sessionStore, sessionMW, config)

	// Make callback request
	reqBody := strings.NewReader(`{"pin_id": 12345, "redirect_uri": "/dashboard"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/plex/callback", reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.PlexCallback(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp["success"] != true {
		t.Error("success should be true")
	}
	if resp["redirect_url"] != "/dashboard" {
		t.Errorf("redirect_url = %v, want /dashboard", resp["redirect_url"])
	}

	user := resp["user"].(map[string]interface{})
	if user["username"] != "plexuser" {
		t.Errorf("username = %v, want plexuser", user["username"])
	}
}

func TestFlowHandlers_PlexCallback_NotConfigured(t *testing.T) {
	sessionStore := NewMemorySessionStore()
	sessionMW := NewSessionMiddleware(sessionStore, nil)

	// No Plex flow configured
	handlers := NewFlowHandlers(nil, nil, sessionStore, sessionMW, nil)

	reqBody := strings.NewReader(`{"pin_id": 12345}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/plex/callback", reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.PlexCallback(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestFlowHandlers_PlexCallback_InvalidJSON(t *testing.T) {
	plexConfig := &PlexFlowConfig{
		ClientID: "test-client",
		Product:  "Test App",
	}

	plexStore := NewMemoryPlexPINStore()
	plexFlow := NewPlexFlow(plexConfig, plexStore)

	sessionStore := NewMemorySessionStore()
	sessionMW := NewSessionMiddleware(sessionStore, nil)

	handlers := NewFlowHandlers(nil, plexFlow, sessionStore, sessionMW, nil)

	reqBody := strings.NewReader(`not valid json`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/plex/callback", reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.PlexCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestFlowHandlers_PlexCallback_MissingPinID(t *testing.T) {
	plexConfig := &PlexFlowConfig{
		ClientID: "test-client",
		Product:  "Test App",
	}

	plexStore := NewMemoryPlexPINStore()
	plexFlow := NewPlexFlow(plexConfig, plexStore)

	sessionStore := NewMemorySessionStore()
	sessionMW := NewSessionMiddleware(sessionStore, nil)

	handlers := NewFlowHandlers(nil, plexFlow, sessionStore, sessionMW, nil)

	// pin_id is 0 (missing)
	reqBody := strings.NewReader(`{"redirect_uri": "/dashboard"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/plex/callback", reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.PlexCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestFlowHandlers_PlexCallback_PINNotFound(t *testing.T) {
	// Mock Plex PIN check endpoint that returns 404
	pinCheckServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer pinCheckServer.Close()

	plexConfig := &PlexFlowConfig{
		ClientID:   "test-client",
		Product:    "Test App",
		PINTimeout: 5 * time.Minute,
	}

	plexStore := NewMemoryPlexPINStore()
	plexFlow := NewPlexFlow(plexConfig, plexStore)
	plexFlow.SetPINCheckEndpoint(pinCheckServer.URL + "/pins")

	sessionStore := NewMemorySessionStore()
	sessionMW := NewSessionMiddleware(sessionStore, nil)

	handlers := NewFlowHandlers(nil, plexFlow, sessionStore, sessionMW, nil)

	reqBody := strings.NewReader(`{"pin_id": 99999}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/plex/callback", reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.PlexCallback(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestFlowHandlers_PlexCallback_PINNotAuthorized(t *testing.T) {
	// Mock Plex PIN check endpoint that returns unauthorized PIN
	pinCheckServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":         12345,
			"code":       "ABC1",
			"auth_token": nil, // Not authorized yet
			"expires_at": time.Now().Add(5 * time.Minute).Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer pinCheckServer.Close()

	plexConfig := &PlexFlowConfig{
		ClientID:   "test-client",
		Product:    "Test App",
		PINTimeout: 5 * time.Minute,
	}

	plexStore := NewMemoryPlexPINStore()
	plexFlow := NewPlexFlow(plexConfig, plexStore)
	plexFlow.SetPINCheckEndpoint(pinCheckServer.URL + "/pins")

	sessionStore := NewMemorySessionStore()
	sessionMW := NewSessionMiddleware(sessionStore, nil)

	handlers := NewFlowHandlers(nil, plexFlow, sessionStore, sessionMW, nil)

	reqBody := strings.NewReader(`{"pin_id": 12345}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/plex/callback", reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.PlexCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	body := w.Body.String()
	if !strings.Contains(body, "not yet authorized") {
		t.Errorf("body = %s, should contain 'not yet authorized'", body)
	}
}

func TestFlowHandlers_PlexCallback_UserInfoFailure(t *testing.T) {
	// Mock Plex endpoints - PIN is authorized but user info fails
	pinCheckServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pins/") {
			authToken := "plex-auth-token-123"
			resp := map[string]interface{}{
				"id":         12345,
				"code":       "ABC1",
				"auth_token": &authToken,
				"expires_at": time.Now().Add(5 * time.Minute).Format(time.RFC3339),
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/user" {
			// Return unauthorized
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		http.NotFound(w, r)
	}))
	defer pinCheckServer.Close()

	plexConfig := &PlexFlowConfig{
		ClientID:   "test-client",
		Product:    "Test App",
		PINTimeout: 5 * time.Minute,
	}

	plexStore := NewMemoryPlexPINStore()
	plexFlow := NewPlexFlow(plexConfig, plexStore)
	plexFlow.SetPINCheckEndpoint(pinCheckServer.URL + "/pins")
	plexFlow.SetUserInfoEndpoint(pinCheckServer.URL + "/user")

	sessionStore := NewMemorySessionStore()
	sessionMW := NewSessionMiddleware(sessionStore, nil)

	handlers := NewFlowHandlers(nil, plexFlow, sessionStore, sessionMW, nil)

	reqBody := strings.NewReader(`{"pin_id": 12345}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/plex/callback", reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.PlexCallback(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d. Body: %s", w.Code, http.StatusInternalServerError, w.Body.String())
	}
}

func TestFlowHandlers_PlexCallback_DefaultRedirect(t *testing.T) {
	// Mock Plex PIN check and user info endpoints
	pinCheckServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pins/") {
			authToken := "plex-auth-token-123"
			resp := map[string]interface{}{
				"id":         12345,
				"code":       "ABC1",
				"auth_token": &authToken,
				"expires_at": time.Now().Add(5 * time.Minute).Format(time.RFC3339),
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/user" {
			resp := map[string]interface{}{
				"user": map[string]interface{}{
					"id":           123,
					"uuid":         "uuid-123",
					"username":     "plexuser",
					"email":        "plex@example.com",
					"subscription": map[string]interface{}{"active": false},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer pinCheckServer.Close()

	plexConfig := &PlexFlowConfig{
		ClientID:   "test-client",
		Product:    "Test App",
		PINTimeout: 5 * time.Minute,
	}

	plexStore := NewMemoryPlexPINStore()
	plexFlow := NewPlexFlow(plexConfig, plexStore)
	plexFlow.SetPINCheckEndpoint(pinCheckServer.URL + "/pins")
	plexFlow.SetUserInfoEndpoint(pinCheckServer.URL + "/user")

	sessionStore := NewMemorySessionStore()
	sessionMW := NewSessionMiddleware(sessionStore, nil)

	config := &FlowHandlersConfig{
		DefaultPostLoginRedirect: "/home",
	}
	handlers := NewFlowHandlers(nil, plexFlow, sessionStore, sessionMW, config)

	// Request without redirect_uri - should use default
	reqBody := strings.NewReader(`{"pin_id": 12345}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/plex/callback", reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.PlexCallback(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Should use default redirect
	if resp["redirect_url"] != "/home" {
		t.Errorf("redirect_url = %v, want /home", resp["redirect_url"])
	}
}
