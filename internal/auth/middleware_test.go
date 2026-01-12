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

	"github.com/tomtom215/cartographus/internal/config"
)

// testJWTConfig returns a standard test security config for JWT
func testJWTConfig() *config.SecurityConfig {
	return &config.SecurityConfig{
		JWTSecret:      "test-secret-key-that-is-at-least-32-characters-long",
		SessionTimeout: 1 * time.Hour,
	}
}

// setupBasicAuthMiddleware creates a middleware configured for Basic authentication
// The provided username is set as the admin username for RBAC testing
func setupBasicAuthMiddleware(t *testing.T, username, password string) *Middleware {
	basicAuthManager, err := newBasicAuthManagerForTest(username, password)
	if err != nil {
		t.Fatalf("Failed to create basic auth manager: %v", err)
	}
	return &Middleware{
		authMode:               "basic",
		basicAuthManager:       basicAuthManager,
		basicAuthDefaultRole:   "viewer",
		basicAuthAdminUsername: username, // The test username gets admin role for testing
	}
}

// makeBasicAuthHeader creates a Basic Auth header value
func makeBasicAuthHeader(username, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}

func TestIsValidIP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv4 localhost", "127.0.0.1", true},
		{"valid IPv6", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
		{"valid IPv6 short", "::1", true},
		{"invalid with spaces", "192.168. 1.1", false},
		{"invalid empty", "", false},
		{"invalid format", "not_an_ip", true}, // Simple validation allows this
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isValidIP(tt.ip); got != tt.want {
				t.Errorf("isValidIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestRateLimiter(t *testing.T) {
	t.Run("basic rate limiting", func(t *testing.T) {
		limiter := NewRateLimiter(2, 1*time.Second)
		ip := "192.168.1.1"

		if !limiter.Allow(ip) {
			t.Error("First request should be allowed")
		}
		if !limiter.Allow(ip) {
			t.Error("Second request should be allowed")
		}
		if limiter.Allow(ip) {
			t.Error("Third request should be denied")
		}

		time.Sleep(1100 * time.Millisecond)
		if !limiter.Allow(ip) {
			t.Error("Request after reset should be allowed")
		}
	})

	t.Run("multiple IPs rate limited independently", func(t *testing.T) {
		limiter := NewRateLimiter(1, 1*time.Second)

		if !limiter.Allow("192.168.1.1") || !limiter.Allow("192.168.1.2") {
			t.Error("First request from each IP should be allowed")
		}
		if limiter.Allow("192.168.1.1") || limiter.Allow("192.168.1.2") {
			t.Error("Second request from each IP should be denied")
		}
	})

	t.Run("cleanup removes old limiters", func(t *testing.T) {
		limiter := NewRateLimiter(100, 1*time.Minute)
		for _, ip := range []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"} {
			limiter.Allow(ip)
		}

		if len(limiter.limiters) != 3 {
			t.Errorf("Expected 3 limiters, got %d", len(limiter.limiters))
		}

		limiter.mu.Lock()
		for ip := range limiter.limiters {
			limiter.limiters[ip].lastAccess = time.Now().Add(-2 * time.Hour)
		}
		limiter.mu.Unlock()

		limiter.cleanup()

		limiter.mu.RLock()
		count := len(limiter.limiters)
		limiter.mu.RUnlock()

		if count != 0 {
			t.Errorf("Expected 0 limiters after cleanup, got %d", count)
		}
	})

	t.Run("stop cleanup gracefully", func(t *testing.T) {
		limiter := NewRateLimiter(100, 1*time.Minute)
		go limiter.startCleanup(100 * time.Millisecond)
		time.Sleep(50 * time.Millisecond)
		limiter.Stop()
		time.Sleep(200 * time.Millisecond)
	})
}

func TestMiddleware_getClientIP(t *testing.T) {
	tests := []struct {
		name           string
		trustedProxies map[string]bool
		remoteAddr     string
		xffHeader      string
		xriHeader      string
		want           string
	}{
		{
			name:           "IPv4 with port direct",
			trustedProxies: map[string]bool{},
			remoteAddr:     "192.168.1.1:12345",
			want:           "192.168.1.1",
		},
		{
			name:           "IPv4 without port direct",
			trustedProxies: map[string]bool{},
			remoteAddr:     "192.168.1.1",
			want:           "192.168.1.1",
		},
		{
			name:           "XFF from trusted proxy",
			trustedProxies: map[string]bool{"10.0.0.1": true},
			remoteAddr:     "10.0.0.1:12345",
			xffHeader:      "192.168.1.100",
			want:           "192.168.1.100",
		},
		{
			name:           "XFF multiple IPs from trusted proxy",
			trustedProxies: map[string]bool{"10.0.0.1": true},
			remoteAddr:     "10.0.0.1:12345",
			xffHeader:      "192.168.1.100, 10.0.0.2",
			want:           "192.168.1.100",
		},
		{
			name:           "X-Real-IP from trusted proxy",
			trustedProxies: map[string]bool{"10.0.0.1": true},
			remoteAddr:     "10.0.0.1:12345",
			xriHeader:      "192.168.1.101",
			want:           "192.168.1.101",
		},
		{
			name:           "XFF takes precedence over X-Real-IP",
			trustedProxies: map[string]bool{"10.0.0.1": true},
			remoteAddr:     "10.0.0.1:12345",
			xffHeader:      "192.168.1.100",
			xriHeader:      "192.168.1.101",
			want:           "192.168.1.100",
		},
		{
			name:           "untrusted proxy ignores headers",
			trustedProxies: map[string]bool{"10.0.0.1": true},
			remoteAddr:     "192.168.1.50:12345",
			xffHeader:      "10.0.0.100",
			want:           "192.168.1.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Middleware{trustedProxies: tt.trustedProxies}
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xffHeader != "" {
				req.Header.Set("X-Forwarded-For", tt.xffHeader)
			}
			if tt.xriHeader != "" {
				req.Header.Set("X-Real-IP", tt.xriHeader)
			}

			if got := m.getClientIP(req); got != tt.want {
				t.Errorf("getClientIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMiddleware_CORS(t *testing.T) {
	tests := []struct {
		name          string
		corsOrigins   []string
		requestOrigin string
		wantAllowed   bool
		wantHeader    string
	}{
		{"wildcard allows all", []string{"*"}, "https://example.com", true, "*"},
		{"specific origin allowed", []string{"https://example.com"}, "https://example.com", true, "https://example.com"},
		{"origin not in list", []string{"https://example.com"}, "https://evil.com", false, ""},
		{"multiple origins one matches", []string{"https://example.com", "https://another.com"}, "https://another.com", true, "https://another.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Middleware{corsOrigins: tt.corsOrigins}
			handler := m.CORS(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

			req := httptest.NewRequest("OPTIONS", "/", nil)
			req.Header.Set("Origin", tt.requestOrigin)
			w := httptest.NewRecorder()
			handler(w, req)

			if tt.wantAllowed {
				if w.Code == http.StatusForbidden {
					t.Error("CORS() blocked allowed origin")
				}
				if got := w.Header().Get("Access-Control-Allow-Origin"); got != tt.wantHeader {
					t.Errorf("CORS() header = %v, want %v", got, tt.wantHeader)
				}
			} else if w.Code != http.StatusForbidden {
				t.Errorf("CORS() allowed forbidden origin, got status %d", w.Code)
			}
		})
	}
}

func TestMiddleware_Authenticate_JWT(t *testing.T) {
	jwtManager, _ := NewJWTManager(testJWTConfig())
	validToken, _ := jwtManager.GenerateToken("testuser", "admin")

	tests := []struct {
		name         string
		authMode     string
		authHeader   string
		cookie       *http.Cookie
		wantStatus   int
		wantCalled   bool
		wantUsername string
	}{
		{
			name:       "no auth mode passes",
			authMode:   "none",
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "missing token returns 401",
			authMode:   "jwt",
			wantStatus: http.StatusUnauthorized,
			wantCalled: false,
		},
		{
			name:         "valid token in header",
			authMode:     "jwt",
			authHeader:   "Bearer " + validToken,
			wantStatus:   http.StatusOK,
			wantCalled:   true,
			wantUsername: "testuser",
		},
		{
			name:         "valid token in cookie",
			authMode:     "jwt",
			cookie:       &http.Cookie{Name: "token", Value: validToken},
			wantStatus:   http.StatusOK,
			wantCalled:   true,
			wantUsername: "testuser",
		},
		{
			name:       "invalid token returns 401",
			authMode:   "jwt",
			cookie:     &http.Cookie{Name: "token", Value: "invalid.jwt.token"},
			wantStatus: http.StatusUnauthorized,
			wantCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Middleware{authMode: tt.authMode, jwtManager: jwtManager}

			handlerCalled := false
			var capturedUsername string
			handler := m.Authenticate(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				if claims, ok := r.Context().Value(ClaimsContextKey).(*Claims); ok {
					capturedUsername = claims.Username
				}
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if handlerCalled != tt.wantCalled {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.wantCalled)
			}
			if tt.wantUsername != "" && capturedUsername != tt.wantUsername {
				t.Errorf("username = %q, want %q", capturedUsername, tt.wantUsername)
			}
		})
	}
}

func TestMiddleware_Authenticate_BasicAuth(t *testing.T) {
	m := setupBasicAuthMiddleware(t, "admin", "securepass123")

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantCalled bool
	}{
		{"missing credentials", "", http.StatusUnauthorized, false},
		{"valid credentials", makeBasicAuthHeader("admin", "securepass123"), http.StatusOK, true},
		{"wrong password", makeBasicAuthHeader("admin", "wrongpassword"), http.StatusUnauthorized, false},
		{"wrong username", makeBasicAuthHeader("hacker", "securepass123"), http.StatusUnauthorized, false},
		{"both wrong", makeBasicAuthHeader("hacker", "wrongpass"), http.StatusUnauthorized, false},
		{"empty password", makeBasicAuthHeader("admin", ""), http.StatusUnauthorized, false},
		{"empty username", makeBasicAuthHeader("", "securepass123"), http.StatusUnauthorized, false},
		{"case sensitive username", makeBasicAuthHeader("Admin", "securepass123"), http.StatusUnauthorized, false},
		{"case sensitive password", makeBasicAuthHeader("admin", "SecurePass123"), http.StatusUnauthorized, false},
		{"missing Basic prefix", base64.StdEncoding.EncodeToString([]byte("admin:securepass123")), http.StatusUnauthorized, false},
		{"invalid base64", "Basic !!invalid!!", http.StatusUnauthorized, false},
		{"missing colon", "Basic " + base64.StdEncoding.EncodeToString([]byte("adminsecurepass123")), http.StatusUnauthorized, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := m.Authenticate(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if handlerCalled != tt.wantCalled {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.wantCalled)
			}
			if tt.wantStatus == http.StatusUnauthorized {
				if wwwAuth := w.Header().Get("WWW-Authenticate"); wwwAuth == "" || !strings.Contains(wwwAuth, "Basic") {
					t.Error("Expected WWW-Authenticate header with Basic scheme")
				}
			}
		})
	}
}

func TestMiddleware_RequireRole(t *testing.T) {
	jwtManager, _ := NewJWTManager(&config.SecurityConfig{
		JWTSecret:      "test_secret_key_that_is_long_enough_for_testing_12345",
		SessionTimeout: 24 * time.Hour,
	})
	middleware := NewMiddleware(jwtManager, nil, "jwt", 100, 1*time.Minute, false, []string{"*"}, []string{}, "", "")

	tests := []struct {
		name         string
		requiredRole string
		userRole     string
		username     string
		wantStatus   int
		wantCalled   bool
	}{
		{"admin can access admin endpoint", "admin", "admin", "admin_user", http.StatusOK, true},
		{"admin can access user endpoint", "user", "admin", "admin_user", http.StatusOK, true},
		{"user cannot access admin endpoint", "admin", "user", "regular_user", http.StatusForbidden, false},
		{"user can access user endpoint", "user", "user", "regular_user", http.StatusOK, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := middleware.jwtManager.GenerateToken(tt.username, tt.userRole)
			if err != nil {
				t.Fatalf("GenerateToken() error = %v", err)
			}

			handlerCalled := false
			handler := middleware.RequireRole(tt.requiredRole, func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.AddCookie(&http.Cookie{Name: "token", Value: token})
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

	t.Run("missing token returns 401", func(t *testing.T) {
		handler := middleware.RequireRole("admin", func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		})
		req := httptest.NewRequest("GET", "/admin", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("basic auth admin can access", func(t *testing.T) {
		basicMiddleware := setupBasicAuthMiddleware(t, "admin", "securepass123")
		handlerCalled := false
		handler := basicMiddleware.RequireRole("admin", func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", makeBasicAuthHeader("admin", "securepass123"))
		w := httptest.NewRecorder()
		handler(w, req)

		if !handlerCalled || w.Code != http.StatusOK {
			t.Error("Basic Auth admin should be able to access admin endpoint")
		}
	})
}

func TestMiddleware_SecurityHeaders(t *testing.T) {
	m := &Middleware{}
	handler := m.SecurityHeaders(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	expectedHeaders := []string{
		"Content-Security-Policy",
		"X-Frame-Options",
		"X-Content-Type-Options",
		"Referrer-Policy",
		"Permissions-Policy",
	}

	for _, header := range expectedHeaders {
		if w.Header().Get(header) == "" {
			t.Errorf("Expected header %s to be set", header)
		}
	}

	csp := w.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "default-src") {
		t.Error("CSP should contain default-src")
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("X-Frame-Options = %s, want DENY", w.Header().Get("X-Frame-Options"))
	}

	t.Run("HSTS with HTTPS", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		w := httptest.NewRecorder()
		handler(w, req)

		hsts := w.Header().Get("Strict-Transport-Security")
		if hsts == "" || !strings.Contains(hsts, "max-age=31536000") {
			t.Errorf("HSTS header missing or incorrect: %s", hsts)
		}
	})
}

func TestMiddleware_RateLimit(t *testing.T) {
	t.Run("allowed when under limit", func(t *testing.T) {
		m := &Middleware{rateLimiter: NewRateLimiter(10, 1*time.Second), trustedProxies: make(map[string]bool)}
		handlerCalled := false
		handler := m.RateLimit(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler(w, req)

		if !handlerCalled || w.Code != http.StatusOK {
			t.Error("Request should be allowed when under rate limit")
		}
	})

	t.Run("exceeded returns 429", func(t *testing.T) {
		m := &Middleware{rateLimiter: NewRateLimiter(1, 1*time.Second), trustedProxies: make(map[string]bool)}
		handler := m.RateLimit(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"

		w1 := httptest.NewRecorder()
		handler(w1, req)
		if w1.Code != http.StatusOK {
			t.Errorf("First request: status = %d, want %d", w1.Code, http.StatusOK)
		}

		w2 := httptest.NewRecorder()
		handler(w2, req)
		if w2.Code != http.StatusTooManyRequests {
			t.Errorf("Second request: status = %d, want %d", w2.Code, http.StatusTooManyRequests)
		}
	})
}

func TestNewMiddleware(t *testing.T) {
	jwtManager, _ := NewJWTManager(testJWTConfig())
	m := NewMiddleware(jwtManager, nil, "jwt", 100, 1*time.Minute, false, []string{"*"}, []string{"10.0.0.1", "10.0.0.2"}, "", "")

	if m == nil {
		t.Fatal("NewMiddleware returned nil")
	}
	if m.authMode != "jwt" {
		t.Errorf("authMode = %q, want 'jwt'", m.authMode)
	}
	if len(m.corsOrigins) != 1 {
		t.Errorf("len(corsOrigins) = %d, want 1", len(m.corsOrigins))
	}
	if len(m.trustedProxies) != 2 {
		t.Errorf("len(trustedProxies) = %d, want 2", len(m.trustedProxies))
	}
	if !m.trustedProxies["10.0.0.1"] {
		t.Error("Expected 10.0.0.1 to be trusted")
	}
}

func TestMiddleware_GetCORSOrigins(t *testing.T) {
	jwtManager, _ := NewJWTManager(testJWTConfig())
	corsOrigins := []string{"https://example.com", "https://app.example.com"}
	m := NewMiddleware(jwtManager, nil, "jwt", 100, 1*time.Minute, false, corsOrigins, nil, "", "")

	got := m.GetCORSOrigins()
	if len(got) != 2 {
		t.Errorf("len(GetCORSOrigins()) = %d, want 2", len(got))
	}
	if got[0] != "https://example.com" {
		t.Errorf("GetCORSOrigins()[0] = %s, want https://example.com", got[0])
	}
	if got[1] != "https://app.example.com" {
		t.Errorf("GetCORSOrigins()[1] = %s, want https://app.example.com", got[1])
	}
}

func TestMiddleware_GetRateLimitConfig(t *testing.T) {
	jwtManager, _ := NewJWTManager(testJWTConfig())

	t.Run("rate limiting enabled", func(t *testing.T) {
		m := NewMiddleware(jwtManager, nil, "jwt", 150, 1*time.Minute, false, []string{"*"}, nil, "", "")

		reqsPerWindow, disabled := m.GetRateLimitConfig()
		if reqsPerWindow != 150 {
			t.Errorf("reqsPerWindow = %d, want 150", reqsPerWindow)
		}
		if disabled {
			t.Error("expected disabled=false")
		}
	})

	t.Run("rate limiting disabled", func(t *testing.T) {
		m := NewMiddleware(jwtManager, nil, "jwt", 0, 1*time.Minute, true, []string{"*"}, nil, "", "")

		_, disabled := m.GetRateLimitConfig()
		if !disabled {
			t.Error("expected disabled=true")
		}
	})
}

func TestMiddleware_GetRateLimitWindow(t *testing.T) {
	jwtManager, _ := NewJWTManager(testJWTConfig())
	m := NewMiddleware(jwtManager, nil, "jwt", 100, 1*time.Minute, false, []string{"*"}, nil, "", "")

	window := m.GetRateLimitWindow()
	// The function returns 1 minute as the standard window
	if window != 1*time.Minute {
		t.Errorf("GetRateLimitWindow() = %v, want 1m", window)
	}
}
