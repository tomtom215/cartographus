// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
)

// =====================================================
// Flow Handlers Tests - Core & Utility Functions
// ADR-0015: Zero Trust Authentication & Authorization
// =====================================================
// This file contains tests for UserInfo, Config, and token parsing utilities.
// OIDC tests are in flow_handlers_oidc_test.go
// Plex tests are in flow_handlers_plex_test.go
// Session tests are in flow_handlers_session_test.go
// Test helpers are in flow_handlers_test_helpers.go

// =====================================================
// UserInfo Handler Tests
// =====================================================

func TestFlowHandlers_UserInfo_Authenticated(t *testing.T) {
	setup := setupBasicHandlers(t)

	subject := &AuthSubject{
		ID:       "user-123",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"viewer", "editor"},
		Provider: "oidc",
	}
	ctx := createAuthContext(subject)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/userinfo", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	setup.handlers.UserInfo(w, req)

	assertStatusCode(t, w, http.StatusOK)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp["username"] != "testuser" {
		t.Errorf("username = %s, want testuser", resp["username"])
	}
}

func TestFlowHandlers_UserInfo_Unauthenticated(t *testing.T) {
	setup := setupBasicHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/userinfo", nil)
	w := httptest.NewRecorder()

	setup.handlers.UserInfo(w, req)

	assertStatusCode(t, w, http.StatusUnauthorized)
}

// =====================================================
// Configuration Tests
// =====================================================

func TestDefaultFlowHandlersConfig(t *testing.T) {
	config := DefaultFlowHandlersConfig()

	if config.SessionDuration != 24*time.Hour {
		t.Errorf("SessionDuration = %v, want 24h", config.SessionDuration)
	}
	if config.DefaultPostLoginRedirect != "/" {
		t.Errorf("DefaultPostLoginRedirect = %s, want /", config.DefaultPostLoginRedirect)
	}
}

// =====================================================
// Token Parsing Utility Tests
// =====================================================

// createMockLogoutToken creates a mock logout token for testing.
// NOTE: This is a simplified mock - real tokens would be signed JWTs.
func createMockLogoutToken(iss, sub, aud, sid string) string {
	// Create header
	header := `{"alg":"RS256","typ":"JWT"}`

	// Create payload with events claim
	payload := map[string]interface{}{
		"iss": iss,
		"sub": sub,
		"aud": aud,
		"iat": time.Now().Unix(),
		"jti": "test-jti-123",
		"events": map[string]interface{}{
			"http://schemas.openid.net/event/backchannel-logout": map[string]interface{}{},
		},
	}
	if sid != "" {
		payload["sid"] = sid
	}

	payloadBytes, _ := json.Marshal(payload)

	// Create mock token (header.payload.signature)
	headerEncoded := encodeBase64URLNoPad([]byte(header))
	payloadEncoded := encodeBase64URLNoPad(payloadBytes)

	return headerEncoded + "." + payloadEncoded + ".mock-signature"
}

// encodeBase64URLNoPad encodes bytes to base64url without padding.
func encodeBase64URLNoPad(data []byte) string {
	encoded := base64.URLEncoding.EncodeToString(data)
	// Remove padding
	return strings.TrimRight(encoded, "=")
}

func TestParseLogoutTokenClaims(t *testing.T) {
	token := createMockLogoutToken("https://auth.example.com", "user123", "test-client", "session-abc")

	claims, err := parseLogoutTokenClaims(token)
	if err != nil {
		t.Fatalf("parseLogoutTokenClaims error: %v", err)
	}

	if claims.Issuer != "https://auth.example.com" {
		t.Errorf("issuer = %s, want https://auth.example.com", claims.Issuer)
	}
	if claims.Subject != "user123" {
		t.Errorf("subject = %s, want user123", claims.Subject)
	}
	if claims.SessionID != "session-abc" {
		t.Errorf("session_id = %s, want session-abc", claims.SessionID)
	}
	if len(claims.Audience) != 1 || claims.Audience[0] != "test-client" {
		t.Errorf("audience = %v, want [test-client]", claims.Audience)
	}
}

func TestParseLogoutTokenClaims_InvalidToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"single part", "header-only"},
		{"two parts", "header.payload"},
		{"invalid base64 payload", "header.!!!invalid!!!.signature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseLogoutTokenClaims(tt.token)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestParseLogoutTokenClaims_MissingEvents(t *testing.T) {
	// Create token without events claim
	header := `{"alg":"RS256","typ":"JWT"}`
	payload := map[string]interface{}{
		"iss": "https://auth.example.com",
		"sub": "user123",
		"aud": "test-client",
		"iat": time.Now().Unix(),
		"jti": "test-jti-123",
		// Missing "events" claim
	}

	payloadBytes, _ := json.Marshal(payload)
	headerEncoded := encodeBase64URLNoPad([]byte(header))
	payloadEncoded := encodeBase64URLNoPad(payloadBytes)
	token := headerEncoded + "." + payloadEncoded + ".mock-signature"

	_, err := parseLogoutTokenClaims(token)
	if err == nil {
		t.Error("expected error for missing events claim")
	}
	if !strings.Contains(err.Error(), "events") {
		t.Errorf("expected error about events, got: %v", err)
	}
}

func TestParseLogoutTokenClaims_WrongEventType(t *testing.T) {
	// Create token with wrong event type
	header := `{"alg":"RS256","typ":"JWT"}`
	payload := map[string]interface{}{
		"iss": "https://auth.example.com",
		"sub": "user123",
		"aud": "test-client",
		"iat": time.Now().Unix(),
		"jti": "test-jti-123",
		"events": map[string]interface{}{
			"http://wrong.event/type": map[string]interface{}{},
		},
	}

	payloadBytes, _ := json.Marshal(payload)
	headerEncoded := encodeBase64URLNoPad([]byte(header))
	payloadEncoded := encodeBase64URLNoPad(payloadBytes)
	token := headerEncoded + "." + payloadEncoded + ".mock-signature"

	_, err := parseLogoutTokenClaims(token)
	if err == nil {
		t.Error("expected error for wrong event type")
	}
	if !strings.Contains(err.Error(), "back-channel logout") {
		t.Errorf("expected error about back-channel logout event, got: %v", err)
	}
}

func TestParseLogoutTokenClaims_ArrayAudience(t *testing.T) {
	// Create token with array audience
	header := `{"alg":"RS256","typ":"JWT"}`
	payload := map[string]interface{}{
		"iss": "https://auth.example.com",
		"sub": "user123",
		"aud": []string{"client1", "client2", "client3"},
		"iat": time.Now().Unix(),
		"jti": "test-jti-123",
		"events": map[string]interface{}{
			"http://schemas.openid.net/event/backchannel-logout": map[string]interface{}{},
		},
	}

	payloadBytes, _ := json.Marshal(payload)
	headerEncoded := encodeBase64URLNoPad([]byte(header))
	payloadEncoded := encodeBase64URLNoPad(payloadBytes)
	token := headerEncoded + "." + payloadEncoded + ".mock-signature"

	claims, err := parseLogoutTokenClaims(token)
	if err != nil {
		t.Fatalf("parseLogoutTokenClaims error: %v", err)
	}

	if len(claims.Audience) != 3 {
		t.Errorf("audience length = %d, want 3", len(claims.Audience))
	}
}

func TestSplitToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected []string
	}{
		{"valid token", "a.b.c", []string{"a", "b", "c"}},
		{"empty parts", "...", []string{"", "", ""}},
		{"single part", "abc", []string{"abc"}},
		{"two parts", "a.b", []string{"a", "b"}},
		{"four parts", "a.b.c.d", []string{"a", "b", "c", "d"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitToken(tt.token)
			if len(result) != len(tt.expected) {
				t.Errorf("len = %d, want %d", len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("part[%d] = %s, want %s", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestDecodeBase64URL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no padding needed", "dGVzdA", "test"},
		{"one padding", "dGVzdGE", "testa"},
		{"two padding", "dGVzdGFi", "testab"},
		{"standard case", "SGVsbG8gV29ybGQ", "Hello World"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decodeBase64URL(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("result = %s, want %s", string(result), tt.expected)
			}
		})
	}
}

// =====================================================
// Setter/Getter Tests
// =====================================================

func TestFlowHandlers_SetAuditLogger(t *testing.T) {
	setup := setupBasicHandlers(t)

	// Initially nil
	if setup.handlers.GetAuditLogger() != nil {
		t.Error("expected nil audit logger initially")
	}

	// Create and set a mock audit logger
	auditLogger := &OIDCAuditLogger{}
	setup.handlers.SetAuditLogger(auditLogger)

	// Should return the set logger
	if setup.handlers.GetAuditLogger() != auditLogger {
		t.Error("GetAuditLogger should return the set logger")
	}

	// Set to nil
	setup.handlers.SetAuditLogger(nil)
	if setup.handlers.GetAuditLogger() != nil {
		t.Error("expected nil after setting nil")
	}
}

func TestFlowHandlers_SetJTITracker(t *testing.T) {
	setup := setupBasicHandlers(t)

	// Initially nil
	if setup.handlers.GetJTITracker() != nil {
		t.Error("expected nil JTI tracker initially")
	}

	// Create and set a tracker
	tracker := NewMemoryJTITracker()
	setup.handlers.SetJTITracker(tracker)

	// Should return the set tracker
	if setup.handlers.GetJTITracker() != tracker {
		t.Error("GetJTITracker should return the set tracker")
	}

	// Set to nil
	setup.handlers.SetJTITracker(nil)
	if setup.handlers.GetJTITracker() != nil {
		t.Error("expected nil after setting nil")
	}
}
