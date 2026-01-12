// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OIDC support.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/audit"
)

// TestOIDCAuditLogger_NewOIDCAuditLogger tests audit logger creation.
func TestOIDCAuditLogger_NewOIDCAuditLogger(t *testing.T) {
	t.Run("with store and provider", func(t *testing.T) {
		store := audit.NewMemoryStore(100)
		logger := NewOIDCAuditLogger(store, "keycloak")

		if logger == nil {
			t.Fatal("Expected non-nil logger")
		}
		if logger.store != store {
			t.Error("Store not set correctly")
		}
		if logger.provider != "keycloak" {
			t.Errorf("Provider not set correctly: got %s, want keycloak", logger.provider)
		}
	})

	t.Run("with empty provider defaults to oidc", func(t *testing.T) {
		store := audit.NewMemoryStore(100)
		logger := NewOIDCAuditLogger(store, "")

		if logger.provider != "oidc" {
			t.Errorf("Expected default provider 'oidc', got: %s", logger.provider)
		}
	})

	t.Run("with nil store", func(t *testing.T) {
		logger := NewOIDCAuditLogger(nil, "test")

		if logger == nil {
			t.Fatal("Expected non-nil logger even with nil store")
		}
	})
}

// TestOIDCAuditLogger_LogLoginSuccess tests successful login event logging.
func TestOIDCAuditLogger_LogLoginSuccess(t *testing.T) {
	store := audit.NewMemoryStore(100)
	logger := NewOIDCAuditLogger(store, "keycloak")

	subject := &AuthSubject{
		ID:       "user-123",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"admin", "viewer"},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/oidc/callback", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("User-Agent", "TestBrowser/1.0")

	logger.LogLoginSuccess(context.Background(), req, subject, 150*time.Millisecond)

	// Verify event was stored
	if store.Len() != 1 {
		t.Fatalf("Expected 1 event, got: %d", store.Len())
	}

	events, err := store.Query(context.Background(), audit.QueryFilter{
		Types: []audit.EventType{audit.EventTypeAuthSuccess},
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 auth success event, got: %d", len(events))
	}

	event := events[0]
	if event.Actor.ID != "user-123" {
		t.Errorf("Actor ID mismatch: got %s, want user-123", event.Actor.ID)
	}
	if event.Actor.Name != "testuser" {
		t.Errorf("Actor Name mismatch: got %s, want testuser", event.Actor.Name)
	}
	if event.Outcome != audit.OutcomeSuccess {
		t.Errorf("Outcome mismatch: got %s, want success", event.Outcome)
	}
	if event.Source.IPAddress != "192.168.1.100" {
		t.Errorf("Source IP mismatch: got %s, want 192.168.1.100", event.Source.IPAddress)
	}
}

// TestOIDCAuditLogger_LogLoginFailure tests failed login event logging.
func TestOIDCAuditLogger_LogLoginFailure(t *testing.T) {
	store := audit.NewMemoryStore(100)
	logger := NewOIDCAuditLogger(store, "okta")

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/callback?error=access_denied", nil)
	req.RemoteAddr = "10.0.0.50:54321"

	logger.LogLoginFailure(context.Background(), req, "access_denied", "User denied access")

	events, err := store.Query(context.Background(), audit.QueryFilter{
		Types: []audit.EventType{audit.EventTypeAuthFailure},
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 auth failure event, got: %d", len(events))
	}

	event := events[0]
	if event.Outcome != audit.OutcomeFailure {
		t.Errorf("Outcome mismatch: got %s, want failure", event.Outcome)
	}
	if event.Severity != audit.SeverityWarning {
		t.Errorf("Severity mismatch: got %s, want warning", event.Severity)
	}
}

// TestOIDCAuditLogger_LogLogout tests logout event logging.
func TestOIDCAuditLogger_LogLogout(t *testing.T) {
	store := audit.NewMemoryStore(100)
	logger := NewOIDCAuditLogger(store, "auth0")

	req := httptest.NewRequest(http.MethodPost, "/api/auth/oidc/logout", nil)
	req.RemoteAddr = "172.16.0.1:8080"

	logger.LogLogout(context.Background(), req, "user-456", "john.doe", "session-abc123", true)

	events, err := store.Query(context.Background(), audit.QueryFilter{
		Types: []audit.EventType{audit.EventTypeLogout},
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 logout event, got: %d", len(events))
	}

	event := events[0]
	if event.Actor.ID != "user-456" {
		t.Errorf("Actor ID mismatch: got %s, want user-456", event.Actor.ID)
	}
	if event.Action != "oidc.logout" {
		t.Errorf("Action mismatch: got %s, want oidc.logout", event.Action)
	}
	if event.Target == nil || event.Target.ID != "session-abc123" {
		t.Error("Target session ID not set correctly")
	}
}

// TestOIDCAuditLogger_LogBackChannelLogout tests back-channel logout event logging.
func TestOIDCAuditLogger_LogBackChannelLogout(t *testing.T) {
	store := audit.NewMemoryStore(100)
	logger := NewOIDCAuditLogger(store, "keycloak")

	req := httptest.NewRequest(http.MethodPost, "/api/auth/oidc/backchannel-logout", nil)
	req.RemoteAddr = "10.10.10.10:443"

	logger.LogBackChannelLogout(context.Background(), req, "user-789", "session-xyz", 3)

	events, err := store.Query(context.Background(), audit.QueryFilter{
		Types: []audit.EventType{audit.EventTypeLogout},
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 logout event, got: %d", len(events))
	}

	event := events[0]
	if event.Actor.Type != "service" {
		t.Errorf("Actor Type mismatch: got %s, want service", event.Actor.Type)
	}
	if event.Action != "oidc.backchannel_logout" {
		t.Errorf("Action mismatch: got %s, want oidc.backchannel_logout", event.Action)
	}
	if event.Target == nil || event.Target.ID != "user-789" {
		t.Error("Target user ID not set correctly")
	}
}

// TestOIDCAuditLogger_LogBackChannelLogoutFailure tests failed back-channel logout event logging.
func TestOIDCAuditLogger_LogBackChannelLogoutFailure(t *testing.T) {
	store := audit.NewMemoryStore(100)
	logger := NewOIDCAuditLogger(store, "okta")

	req := httptest.NewRequest(http.MethodPost, "/api/auth/oidc/backchannel-logout", nil)

	logger.LogBackChannelLogoutFailure(context.Background(), req, "Invalid signature")

	events, err := store.Query(context.Background(), audit.QueryFilter{
		Types: []audit.EventType{audit.EventTypeAuthFailure},
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 auth failure event, got: %d", len(events))
	}

	event := events[0]
	if event.Action != "oidc.backchannel_logout" {
		t.Errorf("Action mismatch: got %s, want oidc.backchannel_logout", event.Action)
	}
}

// TestOIDCAuditLogger_LogTokenRefresh tests token refresh event logging.
func TestOIDCAuditLogger_LogTokenRefresh(t *testing.T) {
	store := audit.NewMemoryStore(100)
	logger := NewOIDCAuditLogger(store, "keycloak")

	req := httptest.NewRequest(http.MethodPost, "/api/auth/oidc/refresh", nil)

	t.Run("successful refresh", func(t *testing.T) {
		store.Clear()
		newExpiry := time.Now().Add(time.Hour)

		logger.LogTokenRefresh(context.Background(), req, "user-123", "testuser", true, 100*time.Millisecond, newExpiry)

		events, _ := store.Query(context.Background(), audit.DefaultQueryFilter())
		if len(events) != 1 {
			t.Fatalf("Expected 1 event, got: %d", len(events))
		}

		event := events[0]
		if event.Outcome != audit.OutcomeSuccess {
			t.Errorf("Outcome mismatch: got %s, want success", event.Outcome)
		}
	})

	t.Run("failed refresh", func(t *testing.T) {
		store.Clear()

		logger.LogTokenRefresh(context.Background(), req, "user-123", "testuser", false, 50*time.Millisecond, time.Time{})

		events, _ := store.Query(context.Background(), audit.DefaultQueryFilter())
		if len(events) != 1 {
			t.Fatalf("Expected 1 event, got: %d", len(events))
		}

		event := events[0]
		if event.Outcome != audit.OutcomeFailure {
			t.Errorf("Outcome mismatch: got %s, want failure", event.Outcome)
		}
	})
}

// TestOIDCAuditLogger_LogSessionCreated tests session creation event logging.
func TestOIDCAuditLogger_LogSessionCreated(t *testing.T) {
	store := audit.NewMemoryStore(100)
	logger := NewOIDCAuditLogger(store, "keycloak")

	session := &Session{
		ID:        "session-new-123",
		UserID:    "user-456",
		Username:  "alice",
		Roles:     []string{"admin"},
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/oidc/callback", nil)

	logger.LogSessionCreated(context.Background(), req, session)

	events, err := store.Query(context.Background(), audit.QueryFilter{
		Types: []audit.EventType{audit.EventTypeSessionCreated},
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 session created event, got: %d", len(events))
	}

	event := events[0]
	if event.Actor.ID != "user-456" {
		t.Errorf("Actor ID mismatch: got %s, want user-456", event.Actor.ID)
	}
	if event.Target == nil || event.Target.ID != "session-new-123" {
		t.Error("Target session ID not set correctly")
	}
}

// TestOIDCAuditLogger_NilStore tests that logger handles nil store gracefully.
func TestOIDCAuditLogger_NilStore(t *testing.T) {
	logger := NewOIDCAuditLogger(nil, "test")
	req := httptest.NewRequest(http.MethodPost, "/test", nil)

	// All these should not panic
	logger.LogLoginSuccess(context.Background(), req, &AuthSubject{ID: "test"}, time.Second)
	logger.LogLoginFailure(context.Background(), req, "error", "desc")
	logger.LogLogout(context.Background(), req, "user", "name", "session", false)
	logger.LogBackChannelLogout(context.Background(), req, "sub", "sid", 1)
	logger.LogBackChannelLogoutFailure(context.Background(), req, "reason")
	logger.LogTokenRefresh(context.Background(), req, "user", "name", true, time.Second, time.Now())
	logger.LogSessionCreated(context.Background(), req, &Session{ID: "test"})
}

// TestExtractSource tests source extraction from HTTP requests.
func TestExtractSource(t *testing.T) {
	t.Run("basic remote addr", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("User-Agent", "TestAgent/1.0")

		source := extractSource(req)

		if source.IPAddress != "192.168.1.1" {
			t.Errorf("IP mismatch: got %s, want 192.168.1.1", source.IPAddress)
		}
		if source.UserAgent != "TestAgent/1.0" {
			t.Errorf("UserAgent mismatch: got %s, want TestAgent/1.0", source.UserAgent)
		}
	})

	t.Run("X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18, 150.172.238.178")

		source := extractSource(req)

		// Should extract the first IP (original client)
		if source.IPAddress != "203.0.113.50" {
			t.Errorf("IP mismatch: got %s, want 203.0.113.50", source.IPAddress)
		}
	})

	t.Run("nil request", func(t *testing.T) {
		source := extractSource(nil)

		if source.IPAddress != "unknown" {
			t.Errorf("Expected 'unknown' for nil request, got: %s", source.IPAddress)
		}
	})
}

// TestGetRequestID tests request ID extraction.
func TestGetRequestID(t *testing.T) {
	t.Run("X-Request-ID header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "req-123-abc")

		id := getRequestID(req)

		if id != "req-123-abc" {
			t.Errorf("Request ID mismatch: got %s, want req-123-abc", id)
		}
	})

	t.Run("X-Correlation-ID header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Correlation-ID", "corr-456-def")

		id := getRequestID(req)

		if id != "corr-456-def" {
			t.Errorf("Correlation ID mismatch: got %s, want corr-456-def", id)
		}
	})

	t.Run("no headers returns empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		id := getRequestID(req)

		if id != "" {
			t.Errorf("Expected empty string for no headers, got: %s", id)
		}
	})
}

// TestGenerateEventID tests event ID generation.
func TestGenerateEventID(t *testing.T) {
	ids := make(map[string]bool)

	// Generate multiple IDs and verify uniqueness
	for i := 0; i < 100; i++ {
		id := generateEventID()

		if id == "" {
			t.Error("Generated empty event ID")
		}

		if len(id) != 32 { // 16 bytes hex encoded = 32 chars
			t.Errorf("Event ID length mismatch: got %d, want 32", len(id))
		}

		if ids[id] {
			t.Errorf("Duplicate event ID generated: %s", id)
		}
		ids[id] = true
	}
}
