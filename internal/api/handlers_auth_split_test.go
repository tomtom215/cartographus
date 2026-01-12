// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func TestLogin_Success(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			AuthMode:       "jwt",
			JWTSecret:      "test_secret_with_at_least_32_characters_for_testing",
			AdminUsername:  "admin",
			AdminPassword:  "password123",
			SessionTimeout: 24 * time.Hour,
		},
	}

	jwtManager, err := auth.NewJWTManager(&cfg.Security)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	handler := &Handler{
		config:     cfg,
		jwtManager: jwtManager,
	}

	loginReq := models.LoginRequest{
		Username:   "admin",
		Password:   "password123",
		RememberMe: false,
	}

	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Response data is not a map")
	}

	if data["token"] == nil || data["token"] == "" {
		t.Error("Expected token in response")
	}

	if data["username"] != "admin" {
		t.Errorf("Expected username 'admin', got '%v'", data["username"])
	}

	cookies := w.Result().Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == "token" {
			found = true
			if !cookie.HttpOnly {
				t.Error("Expected cookie to be HttpOnly")
			}
			if cookie.SameSite != http.SameSiteStrictMode {
				t.Error("Expected cookie SameSite to be Strict")
			}
		}
	}
	if !found {
		t.Error("Expected token cookie in response")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			AuthMode:       "jwt",
			JWTSecret:      "test_secret_with_at_least_32_characters_for_testing",
			AdminUsername:  "admin",
			AdminPassword:  "password123",
			SessionTimeout: 24 * time.Hour,
		},
	}

	jwtManager, err := auth.NewJWTManager(&cfg.Security)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	handler := &Handler{
		config:     cfg,
		jwtManager: jwtManager,
	}

	tests := []struct {
		name     string
		username string
		password string
	}{
		{"Wrong password", "admin", "wrongpassword"},
		{"Wrong username", "wronguser", "password123"},
		{"Both wrong", "wronguser", "wrongpassword"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loginReq := models.LoginRequest{
				Username:   tt.username,
				Password:   tt.password,
				RememberMe: false,
			}

			body, _ := json.Marshal(loginReq)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.Login(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", w.Code)
			}

			var response models.APIResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Status != "error" {
				t.Errorf("Expected status 'error', got '%s'", response.Status)
			}

			if response.Error == nil {
				t.Error("Expected error in response")
			}
		})
	}
}

func TestLogin_MissingFields(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			AuthMode:       "jwt",
			JWTSecret:      "test_secret_with_at_least_32_characters_for_testing",
			AdminUsername:  "admin",
			AdminPassword:  "password123",
			SessionTimeout: 24 * time.Hour,
		},
	}

	jwtManager, err := auth.NewJWTManager(&cfg.Security)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	handler := &Handler{
		config:     cfg,
		jwtManager: jwtManager,
	}

	tests := []struct {
		name     string
		username string
		password string
	}{
		{"Missing username", "", "password123"},
		{"Missing password", "admin", ""},
		{"Both missing", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loginReq := models.LoginRequest{
				Username:   tt.username,
				Password:   tt.password,
				RememberMe: false,
			}

			body, _ := json.Marshal(loginReq)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.Login(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}

			var response models.APIResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Status != "error" {
				t.Errorf("Expected status 'error', got '%s'", response.Status)
			}
		})
	}
}

func TestLogin_InvalidJSON(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			AuthMode:       "jwt",
			JWTSecret:      "test_secret_with_at_least_32_characters_for_testing",
			AdminUsername:  "admin",
			AdminPassword:  "password123",
			SessionTimeout: 24 * time.Hour,
		},
	}

	jwtManager, err := auth.NewJWTManager(&cfg.Security)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	handler := &Handler{
		config:     cfg,
		jwtManager: jwtManager,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", response.Status)
	}
}

func TestLogin_AuthModeNone(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			AuthMode: "none",
		},
	}

	handler := &Handler{
		config:     cfg,
		jwtManager: nil,
	}

	loginReq := models.LoginRequest{
		Username:   "admin",
		Password:   "password123",
		RememberMe: false,
	}

	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.Login(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", response.Status)
	}
}

func TestLogin_MethodNotAllowed(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			AuthMode:       "jwt",
			JWTSecret:      "test_secret_with_at_least_32_characters_for_testing",
			AdminUsername:  "admin",
			AdminPassword:  "password123",
			SessionTimeout: 24 * time.Hour,
		},
	}

	jwtManager, err := auth.NewJWTManager(&cfg.Security)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	handler := &Handler{
		config:     cfg,
		jwtManager: jwtManager,
	}

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/auth/login", nil)
			w := httptest.NewRecorder()
			handler.Login(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

func TestTautulliUserLogins_Success(t *testing.T) {
	mockClient := &MockTautulliClient{
		GetUserLoginsFunc: func(ctx context.Context, userID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUserLogins, error) {
			return &tautulli.TautulliUserLogins{
				Response: tautulli.TautulliUserLoginsResponse{
					Result: "success",
					Data: tautulli.TautulliUserLoginsData{
						RecordsTotal:    500,
						RecordsFiltered: 250,
						Draw:            1,
						Data: []tautulli.TautulliUserLoginsRow{
							{
								Timestamp:    1700000000,
								Time:         "2023-11-15 10:00:00",
								UserID:       1,
								Username:     "testuser",
								FriendlyName: "Test User",
								IPAddress:    "192.168.1.100",
								Host:         "home-network",
								UserAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
								OS:           "Windows 10",
								Browser:      "Chrome 119",
								Success:      1,
							},
							{
								Timestamp:    1699000000,
								Time:         "2023-11-10 15:30:00",
								UserID:       1,
								Username:     "testuser",
								FriendlyName: "Test User",
								IPAddress:    "10.0.0.5",
								Host:         "mobile-network",
								UserAgent:    "Plex/8.5.0 (Android 13)",
								OS:           "Android",
								Browser:      "Plex App",
								Success:      1,
							},
						},
					},
				},
			}, nil
		},
	}

	handler := &Handler{
		client: mockClient,
		cache:  cache.New(5 * time.Minute),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tautulli/user-logins?user_id=1&start=0&length=10", nil)
	w := httptest.NewRecorder()

	handler.TautulliUserLogins(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}
