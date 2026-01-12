// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"
)

// =====================================================
// Flow Handlers Test Helpers
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
// =====================================================
// These helpers are shared across all flow handler test files.
// Updated to use ZitadelOIDCFlow for testing.

// testSetup holds common test dependencies.
type testSetup struct {
	sessionStore SessionStore
	sessionMW    *SessionMiddleware
	oidcFlow     *ZitadelOIDCFlow // Updated for Zitadel integration
	plexFlow     *PlexFlow
	handlers     *FlowHandlers
}

// setupBasicHandlers creates handlers without OIDC or Plex configured.
func setupBasicHandlers(t *testing.T) *testSetup {
	t.Helper()
	sessionStore := NewMemorySessionStore()
	sessionMW := NewSessionMiddleware(sessionStore, nil)
	handlers := NewFlowHandlers(nil, nil, sessionStore, sessionMW, nil)
	return &testSetup{
		sessionStore: sessionStore,
		sessionMW:    sessionMW,
		handlers:     handlers,
	}
}

// setupOIDCHandlers creates handlers with OIDC configured using a mock server.
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
// Uses ZitadelOIDCFlow with a mock OIDC server for testing.
func setupOIDCHandlers(t *testing.T) *testSetup {
	t.Helper()

	// Create mock OIDC server for discovery
	mockServer, err := NewMockOIDCServer("test-client", "test-secret")
	if err != nil {
		t.Fatalf("create mock server: %v", err)
	}
	t.Cleanup(func() { mockServer.Close() })

	// Create OIDC flow configuration
	oidcConfig := &OIDCFlowConfig{
		IssuerURL:    mockServer.Issuer,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  mockServer.Issuer + "/callback",
		Scopes:       []string{"openid", "profile"},
		PKCEEnabled:  true,
		StateTTL:     10 * time.Minute,
		NonceEnabled: true,
	}

	// Create Zitadel OIDC flow with state store
	stateStore := NewZitadelMemoryStateStore()
	oidcFlow, err := NewZitadelOIDCFlowFromConfig(context.Background(), oidcConfig, stateStore)
	if err != nil {
		t.Fatalf("create Zitadel OIDC flow: %v", err)
	}

	sessionStore := NewMemorySessionStore()
	sessionMW := NewSessionMiddleware(sessionStore, nil)
	handlers := NewFlowHandlers(oidcFlow, nil, sessionStore, sessionMW, nil)

	return &testSetup{
		sessionStore: sessionStore,
		sessionMW:    sessionMW,
		oidcFlow:     oidcFlow,
		handlers:     handlers,
	}
}

// createAuthContext creates a context with an authenticated subject.
func createAuthContext(subject *AuthSubject) context.Context {
	return context.WithValue(context.Background(), AuthSubjectContextKey, subject)
}

// createTestSession creates a test session in the store.
func createTestSession(t *testing.T, store SessionStore, session *Session) {
	t.Helper()
	if err := store.Create(context.Background(), session); err != nil {
		t.Fatalf("create session: %v", err)
	}
}

// assertStatusCode checks the response status code.
func assertStatusCode(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Errorf("status = %d, want %d. Body: %s", w.Code, want, w.Body.String())
	}
}

// setupOIDCWithEndSession creates handlers with OIDC and end_session_endpoint configured.
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
// Uses ZitadelOIDCFlow with a mock OIDC server that supports logout.
func setupOIDCWithEndSession(t *testing.T, postLogoutRedirectURI string) *testSetup {
	t.Helper()

	// Create mock OIDC server for discovery
	mockServer, err := NewMockOIDCServer("test-client", "test-secret")
	if err != nil {
		t.Fatalf("create mock server: %v", err)
	}
	t.Cleanup(func() { mockServer.Close() })

	// Create OIDC flow configuration
	oidcConfig := &OIDCFlowConfig{
		IssuerURL:    mockServer.Issuer,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  mockServer.Issuer + "/callback",
		Scopes:       []string{"openid", "profile"},
		StateTTL:     10 * time.Minute,
		NonceEnabled: true,
	}

	// Create Zitadel OIDC flow with state store
	stateStore := NewZitadelMemoryStateStore()
	oidcFlow, err := NewZitadelOIDCFlowFromConfig(context.Background(), oidcConfig, stateStore)
	if err != nil {
		t.Fatalf("create Zitadel OIDC flow: %v", err)
	}

	sessionStore := NewMemorySessionStore()
	sessionMW := NewSessionMiddleware(sessionStore, nil)

	var config *FlowHandlersConfig
	if postLogoutRedirectURI != "" {
		config = &FlowHandlersConfig{
			PostLogoutRedirectURI: postLogoutRedirectURI,
		}
	}
	handlers := NewFlowHandlers(oidcFlow, nil, sessionStore, sessionMW, config)

	return &testSetup{
		sessionStore: sessionStore,
		sessionMW:    sessionMW,
		oidcFlow:     oidcFlow,
		handlers:     handlers,
	}
}

// setupPlexHandlers creates handlers with Plex flow configured.
func setupPlexHandlers(t *testing.T, pinServer *httptest.Server) *testSetup {
	t.Helper()
	plexConfig := &PlexFlowConfig{
		ClientID:   "test-client",
		Product:    "Test App",
		PINTimeout: 5 * time.Minute,
	}

	plexStore := NewMemoryPlexPINStore()
	plexFlow := NewPlexFlow(plexConfig, plexStore)
	if pinServer != nil {
		plexFlow.SetPINCheckEndpoint(pinServer.URL + "/pins")
	}

	sessionStore := NewMemorySessionStore()
	sessionMW := NewSessionMiddleware(sessionStore, nil)
	handlers := NewFlowHandlers(nil, plexFlow, sessionStore, sessionMW, nil)

	return &testSetup{
		sessionStore: sessionStore,
		sessionMW:    sessionMW,
		plexFlow:     plexFlow,
		handlers:     handlers,
	}
}

// createMultipleSessions creates n sessions for a user.
func createMultipleSessions(t *testing.T, store SessionStore, userID, username string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		session := &Session{
			ID:        "session-" + string(rune('a'+i)),
			UserID:    userID,
			Username:  username,
			Provider:  "oidc",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		if err := store.Create(context.Background(), session); err != nil {
			t.Fatalf("create session: %v", err)
		}
	}
}

// createZitadelFlowFromMock creates a ZitadelOIDCFlow from a MockOIDCServer.
// This is a helper for tests that need to create flows with specific mock behaviors.
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
func createZitadelFlowFromMock(t *testing.T, mockServer *MockOIDCServer) *ZitadelOIDCFlow {
	t.Helper()

	oidcConfig := &OIDCFlowConfig{
		IssuerURL:    mockServer.Issuer,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  mockServer.Issuer + "/callback",
		Scopes:       []string{"openid", "profile"},
		StateTTL:     10 * time.Minute,
		NonceEnabled: true,
	}

	stateStore := NewZitadelMemoryStateStore()
	oidcFlow, err := NewZitadelOIDCFlowFromConfig(context.Background(), oidcConfig, stateStore)
	if err != nil {
		t.Fatalf("create Zitadel OIDC flow: %v", err)
	}

	return oidcFlow
}

// createTestMockServerAndFlow creates a MockOIDCServer and ZitadelOIDCFlow for testing.
// The caller should close the mock server when done.
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
func createTestMockServerAndFlow(t *testing.T) (*MockOIDCServer, *ZitadelOIDCFlow) {
	t.Helper()

	mockServer, err := NewMockOIDCServer("test-client", "test-secret")
	if err != nil {
		t.Fatalf("create mock server: %v", err)
	}

	oidcFlow := createZitadelFlowFromMock(t, mockServer)
	return mockServer, oidcFlow
}

// setupMockOIDCWithDiscovery creates a mock OIDC server and configures the flow with discovery.
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
// Uses ZitadelOIDCFlow with automatic OIDC discovery.
func setupMockOIDCWithDiscovery(t *testing.T) (*MockOIDCServer, *testSetup) {
	t.Helper()
	mockServer, err := NewMockOIDCServer("test-client", "test-secret")
	if err != nil {
		t.Fatalf("create mock server: %v", err)
	}

	oidcConfig := &OIDCFlowConfig{
		IssuerURL:    mockServer.Issuer,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  mockServer.Issuer + "/callback",
		Scopes:       []string{"openid", "profile"},
		StateTTL:     10 * time.Minute,
		NonceEnabled: true,
	}

	// Create Zitadel OIDC flow with state store
	// Discovery happens automatically during NewZitadelOIDCFlowFromConfig
	stateStore := NewZitadelMemoryStateStore()
	oidcFlow, err := NewZitadelOIDCFlowFromConfig(context.Background(), oidcConfig, stateStore)
	if err != nil {
		mockServer.Close()
		t.Fatalf("create Zitadel OIDC flow: %v", err)
	}

	sessionStore := NewMemorySessionStore()
	sessionMW := NewSessionMiddleware(sessionStore, nil)
	handlers := NewFlowHandlers(oidcFlow, nil, sessionStore, sessionMW, nil)

	return mockServer, &testSetup{
		sessionStore: sessionStore,
		sessionMW:    sessionMW,
		oidcFlow:     oidcFlow,
		handlers:     handlers,
	}
}
