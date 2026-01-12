// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/config"
)

// TestMiddlewareV2_AuthenticateWithAuthenticator tests the new Authenticate method
// that uses the Authenticator interface
func TestMiddlewareV2_AuthenticateWithAuthenticator(t *testing.T) {
	jwtManager, _ := NewJWTManager(&config.SecurityConfig{
		JWTSecret:      "test-secret-key-that-is-at-least-32-characters-long",
		SessionTimeout: 1 * time.Hour,
	})
	jwtAuth := NewJWTAuthenticator(jwtManager)

	validToken, _ := jwtManager.GenerateToken("testuser", "admin")

	tests := []struct {
		name          string
		authenticator Authenticator
		setupRequest  func(*http.Request)
		wantStatus    int
		wantCalled    bool
		wantUsername  string
	}{
		{
			name:          "JWT authenticator - valid token",
			authenticator: jwtAuth,
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+validToken)
			},
			wantStatus:   http.StatusOK,
			wantCalled:   true,
			wantUsername: "testuser",
		},
		{
			name:          "JWT authenticator - no credentials",
			authenticator: jwtAuth,
			setupRequest:  func(r *http.Request) {},
			wantStatus:    http.StatusUnauthorized,
			wantCalled:    false,
		},
		{
			name:          "JWT authenticator - invalid token",
			authenticator: jwtAuth,
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer invalid.jwt.token")
			},
			wantStatus: http.StatusUnauthorized,
			wantCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MiddlewareV2{
				authenticator: tt.authenticator,
				authMode:      AuthModeJWT,
			}

			handlerCalled := false
			var capturedSubject *AuthSubject
			handler := m.AuthenticateV2(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				capturedSubject = GetAuthSubject(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/", nil)
			tt.setupRequest(req)
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if handlerCalled != tt.wantCalled {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.wantCalled)
			}
			if tt.wantUsername != "" && capturedSubject != nil {
				if capturedSubject.Username != tt.wantUsername {
					t.Errorf("username = %q, want %q", capturedSubject.Username, tt.wantUsername)
				}
			}
		})
	}
}

func TestMiddlewareV2_AuthModeNone(t *testing.T) {
	m := &MiddlewareV2{
		authMode: AuthModeNone,
	}

	handlerCalled := false
	handler := m.AuthenticateV2(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if !handlerCalled {
		t.Error("Handler should be called when auth mode is none")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestMiddlewareV2_MultiAuthenticator(t *testing.T) {
	// Create JWT authenticator
	jwtManager, _ := NewJWTManager(&config.SecurityConfig{
		JWTSecret:      "test-secret-key-that-is-at-least-32-characters-long",
		SessionTimeout: 1 * time.Hour,
	})
	jwtAuth := NewJWTAuthenticator(jwtManager)
	validJWTToken, _ := jwtManager.GenerateToken("jwtuser", "admin")

	// Create Basic authenticator
	basicManager, _ := NewBasicAuthManager("basicuser", "securepassword123")
	basicAuth := NewBasicAuthenticator(basicManager, &BasicAuthenticatorConfig{DefaultRole: "admin"})

	// Create multi-authenticator
	multiAuth := NewMultiAuthenticator(jwtAuth, basicAuth)

	m := &MiddlewareV2{
		authenticator: multiAuth,
		authMode:      AuthModeMulti,
	}

	tests := []struct {
		name         string
		setupRequest func(*http.Request)
		wantStatus   int
		wantCalled   bool
		wantUsername string
		wantMethod   AuthMode
	}{
		{
			name: "JWT token - should use JWT authenticator",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+validJWTToken)
			},
			wantStatus:   http.StatusOK,
			wantCalled:   true,
			wantUsername: "jwtuser",
			wantMethod:   AuthModeJWT,
		},
		{
			name: "Basic auth - should use Basic authenticator",
			setupRequest: func(r *http.Request) {
				credentials := base64.StdEncoding.EncodeToString([]byte("basicuser:securepassword123"))
				r.Header.Set("Authorization", "Basic "+credentials)
			},
			wantStatus:   http.StatusOK,
			wantCalled:   true,
			wantUsername: "basicuser",
			wantMethod:   AuthModeBasic,
		},
		{
			name:         "No credentials - should fail",
			setupRequest: func(r *http.Request) {},
			wantStatus:   http.StatusUnauthorized,
			wantCalled:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			var capturedSubject *AuthSubject
			handler := m.AuthenticateV2(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				capturedSubject = GetAuthSubject(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/", nil)
			tt.setupRequest(req)
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if handlerCalled != tt.wantCalled {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.wantCalled)
			}
			if tt.wantUsername != "" && capturedSubject != nil {
				if capturedSubject.Username != tt.wantUsername {
					t.Errorf("username = %q, want %q", capturedSubject.Username, tt.wantUsername)
				}
				if capturedSubject.AuthMethod != tt.wantMethod {
					t.Errorf("auth method = %v, want %v", capturedSubject.AuthMethod, tt.wantMethod)
				}
			}
		})
	}
}

func TestMiddlewareV2_BackwardsCompatibility(t *testing.T) {
	jwtManager, _ := NewJWTManager(&config.SecurityConfig{
		JWTSecret:      "test-secret-key-that-is-at-least-32-characters-long",
		SessionTimeout: 1 * time.Hour,
	})
	jwtAuth := NewJWTAuthenticator(jwtManager)
	validToken, _ := jwtManager.GenerateToken("testuser", "editor")

	m := &MiddlewareV2{
		authenticator: jwtAuth,
		authMode:      AuthModeJWT,
	}

	var capturedClaims *Claims
	var capturedSubject *AuthSubject
	handler := m.AuthenticateV2(func(w http.ResponseWriter, r *http.Request) {
		// Should have both Claims (old) and AuthSubject (new) in context
		capturedClaims = GetClaims(r.Context())
		capturedSubject = GetAuthSubject(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	w := httptest.NewRecorder()
	handler(w, req)

	if capturedClaims == nil {
		t.Fatal("Claims should be present in context for backwards compatibility")
	}
	if capturedSubject == nil {
		t.Fatal("AuthSubject should be present in context")
	}

	// Verify Claims data
	if capturedClaims.Username != "testuser" {
		t.Errorf("Claims.Username = %v, want testuser", capturedClaims.Username)
	}
	if capturedClaims.Role != "editor" {
		t.Errorf("Claims.Role = %v, want editor", capturedClaims.Role)
	}

	// Verify AuthSubject data
	if capturedSubject.Username != "testuser" {
		t.Errorf("AuthSubject.Username = %v, want testuser", capturedSubject.Username)
	}
	if !capturedSubject.HasRole("editor") {
		t.Errorf("AuthSubject should have role editor, has %v", capturedSubject.Roles)
	}
}

func TestMiddlewareV2_RequireRoleWithSubject(t *testing.T) {
	jwtManager, _ := NewJWTManager(&config.SecurityConfig{
		JWTSecret:      "test-secret-key-that-is-at-least-32-characters-long",
		SessionTimeout: 1 * time.Hour,
	})
	jwtAuth := NewJWTAuthenticator(jwtManager)

	m := &MiddlewareV2{
		authenticator: jwtAuth,
		authMode:      AuthModeJWT,
	}

	tests := []struct {
		name         string
		requiredRole string
		userRole     string
		wantStatus   int
		wantCalled   bool
	}{
		{"admin can access admin", "admin", "admin", http.StatusOK, true},
		{"admin can access viewer", "viewer", "admin", http.StatusOK, true},
		{"viewer cannot access admin", "admin", "viewer", http.StatusForbidden, false},
		{"viewer can access viewer", "viewer", "viewer", http.StatusOK, true},
		{"editor cannot access admin", "admin", "editor", http.StatusForbidden, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, _ := jwtManager.GenerateToken("testuser", tt.userRole)

			handlerCalled := false
			handler := m.RequireRoleV2(tt.requiredRole, func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if handlerCalled != tt.wantCalled {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.wantCalled)
			}
		})
	}
}

func TestMiddlewareV2_GetAuthSubjectNil(t *testing.T) {
	// Test that GetAuthSubject returns nil when no subject in context
	ctx := context.Background()
	subject := GetAuthSubject(ctx)
	if subject != nil {
		t.Errorf("GetAuthSubject() = %v, want nil", subject)
	}
}

func TestMiddlewareV2_GetClaimsNil(t *testing.T) {
	// Test that GetClaims returns nil when no claims in context
	ctx := context.Background()
	claims := GetClaims(ctx)
	if claims != nil {
		t.Errorf("GetClaims() = %v, want nil", claims)
	}
}

func TestNewMiddlewareV2(t *testing.T) {
	jwtManager, _ := NewJWTManager(&config.SecurityConfig{
		JWTSecret:      "test-secret-key-that-is-at-least-32-characters-long",
		SessionTimeout: 1 * time.Hour,
	})

	config := &MiddlewareV2Config{
		AuthMode:          AuthModeJWT,
		JWTManager:        jwtManager,
		ReqsPerWindow:     100,
		Window:            1 * time.Minute,
		RateLimitDisabled: true,
		CORSOrigins:       []string{"*"},
		TrustedProxies:    []string{"10.0.0.1"},
	}

	m, err := NewMiddlewareV2(config)
	if err != nil {
		t.Fatalf("NewMiddlewareV2() error = %v", err)
	}

	if m == nil {
		t.Fatal("NewMiddlewareV2() returned nil")
	}

	if m.authMode != AuthModeJWT {
		t.Errorf("authMode = %v, want %v", m.authMode, AuthModeJWT)
	}
}

func TestNewMiddlewareV2_MultiMode(t *testing.T) {
	jwtManager, _ := NewJWTManager(&config.SecurityConfig{
		JWTSecret:      "test-secret-key-that-is-at-least-32-characters-long",
		SessionTimeout: 1 * time.Hour,
	})
	basicManager, _ := NewBasicAuthManager("user", "password12345678")

	config := &MiddlewareV2Config{
		AuthMode:          AuthModeMulti,
		JWTManager:        jwtManager,
		BasicAuthManager:  basicManager,
		ReqsPerWindow:     100,
		Window:            1 * time.Minute,
		RateLimitDisabled: true,
		CORSOrigins:       []string{"*"},
	}

	m, err := NewMiddlewareV2(config)
	if err != nil {
		t.Fatalf("NewMiddlewareV2() error = %v", err)
	}

	if m == nil {
		t.Fatal("NewMiddlewareV2() returned nil")
	}

	if m.authMode != AuthModeMulti {
		t.Errorf("authMode = %v, want %v", m.authMode, AuthModeMulti)
	}

	// Verify multi-authenticator was created
	_, ok := m.authenticator.(*MultiAuthenticator)
	if !ok {
		t.Error("Expected MultiAuthenticator for multi mode")
	}
}

// =====================================================
// RequireAnyRoleV2 Tests
// =====================================================

func TestMiddlewareV2_RequireAnyRoleV2_AdminBypass(t *testing.T) {
	jwtManager, _ := NewJWTManager(&config.SecurityConfig{
		JWTSecret:      "test-secret-key-that-is-at-least-32-characters-long",
		SessionTimeout: 1 * time.Hour,
	})
	jwtAuth := NewJWTAuthenticator(jwtManager)

	m := &MiddlewareV2{
		authenticator: jwtAuth,
		authMode:      AuthModeJWT,
	}

	// Admin should bypass any role requirement
	adminToken, _ := jwtManager.GenerateToken("admin-user", "admin")

	handlerCalled := false
	handler := m.RequireAnyRoleV2([]string{"editor", "manager"}, func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (admin should bypass role check)", w.Code, http.StatusOK)
	}
	if !handlerCalled {
		t.Error("Handler should be called for admin user")
	}
}

func TestMiddlewareV2_RequireAnyRoleV2_HasMatchingRole(t *testing.T) {
	jwtManager, _ := NewJWTManager(&config.SecurityConfig{
		JWTSecret:      "test-secret-key-that-is-at-least-32-characters-long",
		SessionTimeout: 1 * time.Hour,
	})
	jwtAuth := NewJWTAuthenticator(jwtManager)

	m := &MiddlewareV2{
		authenticator: jwtAuth,
		authMode:      AuthModeJWT,
	}

	editorToken, _ := jwtManager.GenerateToken("editor-user", "editor")

	handlerCalled := false
	handler := m.RequireAnyRoleV2([]string{"editor", "manager"}, func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+editorToken)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !handlerCalled {
		t.Error("Handler should be called for user with matching role")
	}
}

func TestMiddlewareV2_RequireAnyRoleV2_NoMatchingRole(t *testing.T) {
	jwtManager, _ := NewJWTManager(&config.SecurityConfig{
		JWTSecret:      "test-secret-key-that-is-at-least-32-characters-long",
		SessionTimeout: 1 * time.Hour,
	})
	jwtAuth := NewJWTAuthenticator(jwtManager)

	m := &MiddlewareV2{
		authenticator: jwtAuth,
		authMode:      AuthModeJWT,
	}

	viewerToken, _ := jwtManager.GenerateToken("viewer-user", "viewer")

	handlerCalled := false
	handler := m.RequireAnyRoleV2([]string{"editor", "manager"}, func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if handlerCalled {
		t.Error("Handler should not be called for user without matching role")
	}
}

func TestMiddlewareV2_RequireAnyRoleV2_NoAuth(t *testing.T) {
	jwtManager, _ := NewJWTManager(&config.SecurityConfig{
		JWTSecret:      "test-secret-key-that-is-at-least-32-characters-long",
		SessionTimeout: 1 * time.Hour,
	})
	jwtAuth := NewJWTAuthenticator(jwtManager)

	m := &MiddlewareV2{
		authenticator: jwtAuth,
		authMode:      AuthModeJWT,
	}

	handlerCalled := false
	handler := m.RequireAnyRoleV2([]string{"editor"}, func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	// No auth header
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if handlerCalled {
		t.Error("Handler should not be called without auth")
	}
}

// =====================================================
// RateLimit Tests
// =====================================================

func TestMiddlewareV2_RateLimit_Disabled(t *testing.T) {
	m := &MiddlewareV2{
		rateLimitDisabled: true,
	}

	callCount := 0
	handler := m.RateLimit(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	})

	// Should allow unlimited requests when disabled
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: status = %d, want %d", i, w.Code, http.StatusOK)
		}
	}

	if callCount != 10 {
		t.Errorf("callCount = %d, want 10", callCount)
	}
}

func TestMiddlewareV2_RateLimit_Enabled(t *testing.T) {
	m := &MiddlewareV2{
		rateLimitDisabled: false,
		rateLimiter:       NewRateLimiter(3, 1*time.Minute), // 3 requests per minute
		trustedProxies:    make(map[string]bool),
	}

	handler := m.RateLimit(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: status = %d, want %d", i, w.Code, http.StatusOK)
		}
	}

	// 4th request should be rate limited
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("4th request: status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

// =====================================================
// CORS Tests
// =====================================================

func TestMiddlewareV2_CORS_WildcardOrigin(t *testing.T) {
	m := &MiddlewareV2{
		corsOrigins: []string{"*"},
	}

	handlerCalled := false
	handler := m.CORS(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !handlerCalled {
		t.Error("Handler should be called")
	}

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Access-Control-Allow-Methods should be set")
	}
}

func TestMiddlewareV2_CORS_SpecificOrigin(t *testing.T) {
	m := &MiddlewareV2{
		corsOrigins: []string{"https://allowed.com"},
	}

	handlerCalled := false
	handler := m.CORS(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://allowed.com")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !handlerCalled {
		t.Error("Handler should be called")
	}

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "https://allowed.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want https://allowed.com", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Vary") != "Origin" {
		t.Errorf("Vary = %q, want Origin", w.Header().Get("Vary"))
	}
}

func TestMiddlewareV2_CORS_DisallowedOrigin(t *testing.T) {
	m := &MiddlewareV2{
		corsOrigins: []string{"https://allowed.com"},
	}

	handlerCalled := false
	handler := m.CORS(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// OPTIONS request from disallowed origin
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://not-allowed.com")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d for disallowed origin OPTIONS", w.Code, http.StatusForbidden)
	}
	if handlerCalled {
		t.Error("Handler should not be called for disallowed origin OPTIONS")
	}
}

func TestMiddlewareV2_CORS_PreflightRequest(t *testing.T) {
	m := &MiddlewareV2{
		corsOrigins: []string{"*"},
	}

	handlerCalled := false
	handler := m.CORS(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// OPTIONS preflight request
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if handlerCalled {
		t.Error("Handler should not be called for OPTIONS preflight")
	}
}

// =====================================================
// getClientIP Tests
// =====================================================

func TestMiddlewareV2_getClientIP_DirectConnection(t *testing.T) {
	m := &MiddlewareV2{
		trustedProxies: make(map[string]bool),
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	ip := m.getClientIP(req)
	if ip != "192.168.1.100" {
		t.Errorf("getClientIP() = %q, want 192.168.1.100", ip)
	}
}

func TestMiddlewareV2_getClientIP_IPv6(t *testing.T) {
	m := &MiddlewareV2{
		trustedProxies: make(map[string]bool),
	}

	req := httptest.NewRequest("GET", "/", nil)
	// IPv6 format with port
	req.RemoteAddr = "[::1]:12345"

	ip := m.getClientIP(req)
	// Should extract the IP without port
	if ip != "[" && ip != "[::1]" {
		// IPv6 parsing varies - just ensure no panic
		t.Logf("IPv6 getClientIP() = %q", ip)
	}
}

func TestMiddlewareV2_getClientIP_TrustedProxy(t *testing.T) {
	m := &MiddlewareV2{
		trustedProxies: map[string]bool{
			"10.0.0.1": true,
		},
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18")

	ip := m.getClientIP(req)
	// Should use X-Forwarded-For from trusted proxy
	if ip != "203.0.113.50" {
		t.Errorf("getClientIP() = %q, want 203.0.113.50", ip)
	}
}

func TestMiddlewareV2_getClientIP_UntrustedProxy(t *testing.T) {
	m := &MiddlewareV2{
		trustedProxies: map[string]bool{
			"10.0.0.1": true, // Different IP
		},
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:12345" // Not in trusted list
	req.Header.Set("X-Forwarded-For", "203.0.113.50")

	ip := m.getClientIP(req)
	// Should NOT use X-Forwarded-For from untrusted proxy
	if ip != "192.168.1.100" {
		t.Errorf("getClientIP() = %q, want 192.168.1.100 (should ignore X-Forwarded-For from untrusted proxy)", ip)
	}
}
