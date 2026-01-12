// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
)

// ===================================================================================================
// Validator Integration Tests
// These tests verify that the go-playground/validator v10 integration works correctly
// across all refactored handlers, ensuring proper error codes and message formats.
// ===================================================================================================

// setupValidationTestHandler creates a handler for validation integration tests
func setupValidationTestHandler(t *testing.T) *Handler {
	t.Helper()
	return &Handler{
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 100,
				MaxPageSize:     1000,
			},
		},
		cache: cache.New(5 * time.Minute),
	}
}

// assertValidationError verifies that the response is a proper validation error
func assertValidationError(t *testing.T, w *httptest.ResponseRecorder, expectedCode int) {
	t.Helper()

	if w.Code != expectedCode {
		t.Errorf("status = %d, want %d. Body: %s", w.Code, expectedCode, w.Body.String())
	}

	var response models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Status != "error" {
		t.Errorf("response.Status = %s, want error", response.Status)
	}

	if response.Error == nil {
		t.Fatal("response.Error is nil, expected VALIDATION_ERROR")
	}

	if response.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("response.Error.Code = %s, want VALIDATION_ERROR", response.Error.Code)
	}

	if response.Error.Message == "" {
		t.Error("response.Error.Message is empty, expected descriptive message")
	}
}

// ===================================================================================================
// PlaybacksRequest Validation Integration Tests
// ===================================================================================================

func TestPlaybacksValidation_LimitValidation(t *testing.T) {
	t.Parallel()
	handler := setupValidationTestHandler(t)

	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{"limit zero", "limit=0", http.StatusBadRequest},
		{"limit negative", "limit=-1", http.StatusBadRequest},
		{"limit exceeds struct max", "limit=1001", http.StatusBadRequest},
		{"limit valid min", "limit=1", http.StatusServiceUnavailable}, // 503 because no DB
		{"limit valid max", "limit=1000", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/playbacks?"+tt.query, nil)
			w := httptest.NewRecorder()
			handler.Playbacks(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusBadRequest {
				assertValidationError(t, w, http.StatusBadRequest)
			}
		})
	}
}

func TestPlaybacksValidation_OffsetValidation(t *testing.T) {
	t.Parallel()
	handler := setupValidationTestHandler(t)

	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{"offset negative", "offset=-1", http.StatusBadRequest},
		{"offset exceeds max", "offset=1000001", http.StatusBadRequest},
		{"offset valid zero", "offset=0", http.StatusServiceUnavailable},
		{"offset valid max", "offset=1000000", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/playbacks?"+tt.query, nil)
			w := httptest.NewRecorder()
			handler.Playbacks(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusBadRequest {
				assertValidationError(t, w, http.StatusBadRequest)
			}
		})
	}
}

func TestPlaybacksValidation_CursorValidation(t *testing.T) {
	t.Parallel()
	handler := setupValidationTestHandler(t)

	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{"cursor invalid base64", "cursor=not-valid-base64!!!", http.StatusBadRequest},
		{"cursor with spaces", "cursor=abc%20def", http.StatusBadRequest},
		{"cursor empty", "", http.StatusServiceUnavailable}, // Valid - no cursor
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/playbacks"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			handler.Playbacks(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusBadRequest {
				assertValidationError(t, w, http.StatusBadRequest)
			}
		})
	}
}

// ===================================================================================================
// LocationsRequest Validation Integration Tests
// ===================================================================================================

func TestLocationsValidation_DaysValidation(t *testing.T) {
	t.Parallel()
	handler := setupValidationTestHandler(t)

	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{"days negative", "days=-1", http.StatusBadRequest},
		{"days exceeds max", "days=3651", http.StatusBadRequest},
		{"days valid min", "days=1", http.StatusServiceUnavailable},
		{"days valid max", "days=3650", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/locations?"+tt.query, nil)
			w := httptest.NewRecorder()
			handler.Locations(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusBadRequest {
				assertValidationError(t, w, http.StatusBadRequest)
			}
		})
	}
}

func TestLocationsValidation_DateValidation(t *testing.T) {
	t.Parallel()
	handler := setupValidationTestHandler(t)

	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{"start_date invalid format", "start_date=2025/01/15", http.StatusBadRequest},
		{"start_date date only", "start_date=2025-01-15", http.StatusBadRequest},
		{"start_date garbage", "start_date=not-a-date", http.StatusBadRequest},
		{"end_date invalid format", "end_date=2025/12/31", http.StatusBadRequest},
		{"start_date valid RFC3339", "start_date=2025-01-15T10:30:00Z", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/locations?"+tt.query, nil)
			w := httptest.NewRecorder()
			handler.Locations(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusBadRequest {
				assertValidationError(t, w, http.StatusBadRequest)
			}
		})
	}
}

// ===================================================================================================
// LoginRequestValidation Integration Tests
// ===================================================================================================

func TestLoginValidation_RequiredFields(t *testing.T) {
	t.Parallel()
	handler := setupValidationTestHandler(t)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"empty username", `{"username":"","password":"secret"}`, http.StatusBadRequest},
		{"empty password", `{"username":"admin","password":""}`, http.StatusBadRequest},
		{"both empty", `{"username":"","password":""}`, http.StatusBadRequest},
		{"missing username", `{"password":"secret"}`, http.StatusBadRequest},
		{"missing password", `{"username":"admin"}`, http.StatusBadRequest},
		{"both missing", `{}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler.Login(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusBadRequest {
				assertValidationError(t, w, http.StatusBadRequest)
			}
		})
	}
}

// ===================================================================================================
// ExportPlaybacksCSVRequest Validation Integration Tests
// ===================================================================================================

func TestExportPlaybacksCSVValidation_LimitValidation(t *testing.T) {
	t.Parallel()
	handler := setupValidationTestHandler(t)

	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{"limit zero", "limit=0", http.StatusBadRequest},
		{"limit negative", "limit=-1", http.StatusBadRequest},
		{"limit exceeds max", "limit=100001", http.StatusBadRequest},
		{"limit valid", "limit=10000", http.StatusServiceUnavailable},
		{"limit valid max", "limit=100000", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/export/playbacks/csv?"+tt.query, nil)
			w := httptest.NewRecorder()
			handler.ExportPlaybacksCSV(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusBadRequest {
				assertValidationError(t, w, http.StatusBadRequest)
			}
		})
	}
}

// ===================================================================================================
// Error Message Format Tests
// ===================================================================================================

func TestValidationErrorFormat(t *testing.T) {
	t.Parallel()
	handler := setupValidationTestHandler(t)

	// Test that error messages are user-friendly and contain field names
	tests := []struct {
		name           string
		url            string
		expectedFields []string // Fields that should be mentioned in error message
	}{
		{
			name:           "playbacks limit error mentions Limit",
			url:            "/api/v1/playbacks?limit=0",
			expectedFields: []string{"Limit"},
		},
		{
			name:           "playbacks offset error mentions Offset",
			url:            "/api/v1/playbacks?offset=-1",
			expectedFields: []string{"Offset"},
		},
		{
			name:           "locations days error mentions Days",
			url:            "/api/v1/locations?days=-5",
			expectedFields: []string{"Days"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			// Route to appropriate handler based on URL
			if contains(tt.url, "/playbacks") {
				handler.Playbacks(w, req)
			} else if contains(tt.url, "/locations") {
				handler.Locations(w, req)
			}

			var response models.APIResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if response.Error == nil {
				t.Fatal("Expected error response")
			}

			// Check that error message or details contain expected fields
			for _, field := range tt.expectedFields {
				found := contains(response.Error.Message, field)
				if !found && response.Error.Details != nil {
					// Check in details
					if f, ok := response.Error.Details["field"]; ok {
						found = f == field
					}
				}
				if !found {
					t.Errorf("Error should mention field %s. Got: %s", field, response.Error.Message)
				}
			}
		})
	}
}

// helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
