// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// =====================================================
// CSRF Middleware Tests
// ADR-0015: Zero Trust Authentication - Phase 4D.1
// =====================================================

func TestDefaultCSRFConfig(t *testing.T) {
	config := DefaultCSRFConfig()

	if config.CookieName != "_csrf" {
		t.Errorf("CookieName = %s, want _csrf", config.CookieName)
	}
	if config.HeaderName != "X-CSRF-Token" {
		t.Errorf("HeaderName = %s, want X-CSRF-Token", config.HeaderName)
	}
	if config.FormFieldName != "csrf_token" {
		t.Errorf("FormFieldName = %s, want csrf_token", config.FormFieldName)
	}
	if config.TokenLength != 32 {
		t.Errorf("TokenLength = %d, want 32", config.TokenLength)
	}
	if config.TokenTTL != 24*time.Hour {
		t.Errorf("TokenTTL = %v, want 24h", config.TokenTTL)
	}
	if !config.CookieSecure {
		t.Error("CookieSecure should default to true")
	}
	if config.CookieHTTPOnly {
		t.Error("CookieHTTPOnly should default to false for CSRF")
	}
}

func TestCSRFMiddleware_NewWithNilConfig(t *testing.T) {
	mw := NewCSRFMiddleware(nil)

	if mw == nil {
		t.Fatal("middleware should not be nil")
	}
	if mw.config.CookieName != "_csrf" {
		t.Errorf("CookieName = %s, want _csrf", mw.config.CookieName)
	}
}

func TestCSRFMiddleware_ExemptMethods(t *testing.T) {
	mw := NewCSRFMiddleware(nil)

	exemptMethods := []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace}
	for _, method := range exemptMethods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/data", nil)
			w := httptest.NewRecorder()

			called := false
			handler := mw.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			}))

			handler.ServeHTTP(w, req)

			if !called {
				t.Errorf("%s request should pass through without CSRF token", method)
			}
			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
			}
		})
	}
}

func TestCSRFMiddleware_RequiresTokenForPOST(t *testing.T) {
	mw := NewCSRFMiddleware(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/data", nil)
	w := httptest.NewRecorder()

	called := false
	handler := mw.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	handler.ServeHTTP(w, req)

	if called {
		t.Error("POST request without CSRF token should be blocked")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddleware_TokenInHeader(t *testing.T) {
	mw := NewCSRFMiddleware(nil)

	// First, get a token via GET request
	getReq := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	getW := httptest.NewRecorder()

	mw.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(getW, getReq)

	// Extract token from cookie
	cookies := getW.Result().Cookies()
	var token string
	for _, cookie := range cookies {
		if cookie.Name == "_csrf" {
			token = cookie.Value
			break
		}
	}

	if token == "" {
		t.Fatal("CSRF token cookie should be set")
	}

	// Make POST request with token in header
	postReq := httptest.NewRequest(http.MethodPost, "/api/data", nil)
	postReq.Header.Set("X-CSRF-Token", token)
	postReq.AddCookie(&http.Cookie{Name: "_csrf", Value: token})
	postW := httptest.NewRecorder()

	called := false
	mw.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(postW, postReq)

	if !called {
		t.Error("POST request with valid CSRF token should succeed")
	}
	if postW.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", postW.Code, http.StatusOK)
	}
}

func TestCSRFMiddleware_TokenInFormField(t *testing.T) {
	mw := NewCSRFMiddleware(nil)

	// Get a token
	token := mw.GetToken(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	// Make POST request with token in form
	form := url.Values{}
	form.Set("csrf_token", token)
	postReq := httptest.NewRequest(http.MethodPost, "/api/data", strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(&http.Cookie{Name: "_csrf", Value: token})
	postW := httptest.NewRecorder()

	called := false
	mw.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(postW, postReq)

	if !called {
		t.Error("POST request with valid CSRF token in form should succeed")
	}
}

func TestCSRFMiddleware_InvalidToken(t *testing.T) {
	mw := NewCSRFMiddleware(nil)

	// Get a valid token to store
	validToken := mw.GetToken(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	// Make POST request with wrong token
	postReq := httptest.NewRequest(http.MethodPost, "/api/data", nil)
	postReq.Header.Set("X-CSRF-Token", "wrong-token")
	postReq.AddCookie(&http.Cookie{Name: "_csrf", Value: validToken})
	postW := httptest.NewRecorder()

	called := false
	mw.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(postW, postReq)

	if called {
		t.Error("POST request with invalid CSRF token should be blocked")
	}
	if postW.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", postW.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddleware_ExemptPaths(t *testing.T) {
	mw := NewCSRFMiddleware(&CSRFConfig{
		ExemptPaths: []string{"/api/auth/oidc/backchannel-logout", "/webhooks/"},
	})

	tests := []struct {
		name   string
		path   string
		method string
		exempt bool
	}{
		{"backchannel logout", "/api/auth/oidc/backchannel-logout", http.MethodPost, true},
		{"webhook", "/webhooks/plex", http.MethodPost, true},
		{"regular API", "/api/data", http.MethodPost, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			called := false
			handler := mw.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			}))

			handler.ServeHTTP(w, req)

			if tt.exempt && !called {
				t.Errorf("exempt path %s should pass through", tt.path)
			}
			if !tt.exempt && called {
				t.Errorf("non-exempt path %s should be blocked without token", tt.path)
			}
		})
	}
}

func TestCSRFMiddleware_DELETE_RequiresToken(t *testing.T) {
	mw := NewCSRFMiddleware(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/data/123", nil)
	w := httptest.NewRecorder()

	called := false
	handler := mw.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	handler.ServeHTTP(w, req)

	if called {
		t.Error("DELETE request without CSRF token should be blocked")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddleware_PUT_RequiresToken(t *testing.T) {
	mw := NewCSRFMiddleware(nil)

	req := httptest.NewRequest(http.MethodPut, "/api/data/123", nil)
	w := httptest.NewRecorder()

	called := false
	handler := mw.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	handler.ServeHTTP(w, req)

	if called {
		t.Error("PUT request without CSRF token should be blocked")
	}
}

func TestCSRFMiddleware_CustomErrorHandler(t *testing.T) {
	customHandlerCalled := false
	mw := NewCSRFMiddleware(&CSRFConfig{
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			customHandlerCalled = true
			w.WriteHeader(http.StatusTeapot) // Custom status
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/data", nil)
	w := httptest.NewRecorder()

	mw.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(w, req)

	if !customHandlerCalled {
		t.Error("custom error handler should be called")
	}
	if w.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTeapot)
	}
}

func TestCSRFMiddleware_GetToken(t *testing.T) {
	mw := NewCSRFMiddleware(nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	token := mw.GetToken(w, r)

	if token == "" {
		t.Error("GetToken should return a token")
	}
	if len(token) < 32 {
		t.Errorf("token length = %d, should be at least 32", len(token))
	}

	// Verify cookie was set
	cookies := w.Result().Cookies()
	foundCookie := false
	for _, cookie := range cookies {
		if cookie.Name == "_csrf" && cookie.Value == token {
			foundCookie = true
			break
		}
	}
	if !foundCookie {
		t.Error("CSRF cookie should be set")
	}
}

func TestCSRFMiddleware_TokenReuse(t *testing.T) {
	mw := NewCSRFMiddleware(nil)

	// Get initial token
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodGet, "/", nil)
	token1 := mw.GetToken(w1, r1)

	// Simulate subsequent request with existing cookie
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.AddCookie(&http.Cookie{Name: "_csrf", Value: token1})
	token2 := mw.GetToken(w2, r2)

	// Should return same token if still valid
	if token2 != token1 {
		t.Errorf("token should be reused: got %s, want %s", token2, token1)
	}
}

func TestCSRFTokenStore_CleanupExpired(t *testing.T) {
	store := newCSRFTokenStore()

	// Add some tokens with short TTL
	store.store("token1", 50*time.Millisecond)
	store.store("token2", 50*time.Millisecond)
	store.store("token3", 1*time.Hour) // This one should persist

	// Wait for short tokens to expire
	time.Sleep(100 * time.Millisecond)

	// Run cleanup
	count := store.CleanupExpired()

	if count != 2 {
		t.Errorf("cleanup count = %d, want 2", count)
	}

	// Verify token3 still exists
	if !store.isValid("token3") {
		t.Error("token3 should still be valid")
	}
}

func TestCSRFMiddleware_StartCleanupRoutine(t *testing.T) {
	mw := NewCSRFMiddleware(&CSRFConfig{
		TokenTTL: 50 * time.Millisecond,
	})

	// Generate a token
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	token := mw.GetToken(w, r)

	// Verify token is valid
	if !mw.tokens.isValid(token) {
		t.Error("token should be valid initially")
	}

	// Start cleanup routine
	done := mw.StartCleanupRoutine(50 * time.Millisecond)
	defer close(done)

	// Wait for token to expire and cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Token should be cleaned up
	if mw.tokens.isValid(token) {
		t.Error("expired token should be cleaned up")
	}
}

func TestCSRFMiddleware_MissingCookieButHasHeader(t *testing.T) {
	mw := NewCSRFMiddleware(nil)

	// POST without cookie but with header
	req := httptest.NewRequest(http.MethodPost, "/api/data", nil)
	req.Header.Set("X-CSRF-Token", "some-token")
	w := httptest.NewRecorder()

	called := false
	mw.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(w, req)

	if called {
		t.Error("request without CSRF cookie should be blocked")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddleware_CookieSettings(t *testing.T) {
	mw := NewCSRFMiddleware(&CSRFConfig{
		CookieName:     "custom_csrf",
		CookiePath:     "/api",
		CookieDomain:   "example.com",
		CookieSecure:   true,
		CookieHTTPOnly: false,
		CookieSameSite: http.SameSiteStrictMode,
		TokenTTL:       1 * time.Hour,
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	mw.GetToken(w, r)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected cookie to be set")
	}

	cookie := cookies[0]
	if cookie.Name != "custom_csrf" {
		t.Errorf("cookie name = %s, want custom_csrf", cookie.Name)
	}
	if cookie.Path != "/api" {
		t.Errorf("cookie path = %s, want /api", cookie.Path)
	}
	if cookie.Domain != "example.com" {
		t.Errorf("cookie domain = %s, want example.com", cookie.Domain)
	}
	if !cookie.Secure {
		t.Error("cookie Secure should be true")
	}
	if cookie.HttpOnly {
		t.Error("cookie HttpOnly should be false for CSRF")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("cookie SameSite = %v, want Strict", cookie.SameSite)
	}
}

func TestCSRFMiddleware_ErrorMessages(t *testing.T) {
	mw := NewCSRFMiddleware(nil)

	tests := []struct {
		name        string
		setupReq    func() *http.Request
		expectedMsg string
	}{
		{
			name: "missing token",
			setupReq: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/api/data", nil)
			},
			expectedMsg: "CSRF token missing",
		},
		{
			name: "invalid token",
			setupReq: func() *http.Request {
				// Store a valid token first
				mw.GetToken(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

				req := httptest.NewRequest(http.MethodPost, "/api/data", nil)
				req.AddCookie(&http.Cookie{Name: "_csrf", Value: "valid-stored-token"})
				req.Header.Set("X-CSRF-Token", "different-token")
				return req
			},
			expectedMsg: "CSRF token invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupReq()
			w := httptest.NewRecorder()

			mw.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(w, req)

			body := w.Body.String()
			if !strings.Contains(body, tt.expectedMsg) {
				t.Errorf("response body = %s, should contain %s", body, tt.expectedMsg)
			}
		})
	}
}
