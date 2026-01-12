// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/models"
)

// addAdminContext adds an admin user to the request context for tests.
// RBAC Phase 3: Tests for admin-only endpoints need authentication context.
func addAdminContext(r *http.Request) *http.Request {
	subject := &auth.AuthSubject{
		ID:       "test-admin",
		Username: "admin",
		Roles:    []string{models.RoleAdmin},
	}
	ctx := context.WithValue(r.Context(), auth.AuthSubjectContextKey, subject)
	return r.WithContext(ctx)
}

// TestWrappedServerStats tests the server-wide wrapped statistics endpoint.
func TestWrappedServerStats(t *testing.T) {
	handler := setupTestHandler(t)

	testCases := []struct {
		name           string
		year           string
		expectedStatus int
	}{
		{
			name:           "Valid year",
			year:           "2025",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid year format",
			year:           "invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Year out of range - too old",
			year:           "1999",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Year out of range - too far in future",
			year:           "3000",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/{year}", handler.WrappedServerStats)

			req := httptest.NewRequest(http.MethodGet, "/"+tc.year, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Status = %d, want %d, body: %s", w.Code, tc.expectedStatus, w.Body.String())
			}
		})
	}
}

// TestWrappedServerStats_MethodNotAllowed tests that POST is rejected.
func TestWrappedServerStats_MethodNotAllowed(t *testing.T) {
	handler := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/{year}", handler.WrappedServerStats)

	req := httptest.NewRequest(http.MethodPost, "/2025", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// TestWrappedUserReport tests the per-user wrapped report endpoint.
func TestWrappedUserReport(t *testing.T) {
	handler := setupTestHandler(t)

	testCases := []struct {
		name           string
		year           string
		userID         string
		generate       string
		expectedStatus int
	}{
		{
			name:           "User with no report, no generate flag",
			year:           "2025",
			userID:         "1",
			generate:       "",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid user ID",
			year:           "2025",
			userID:         "invalid",
			generate:       "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid year",
			year:           "invalid",
			userID:         "1",
			generate:       "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "User with generate flag - no data",
			year:           "2025",
			userID:         "999",
			generate:       "true",
			expectedStatus: http.StatusInternalServerError, // No data for user
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/{year}/user/{userID}", handler.WrappedUserReport)

			url := "/" + tc.year + "/user/" + tc.userID
			if tc.generate != "" {
				url += "?generate=" + tc.generate
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Status = %d, want %d, body: %s", w.Code, tc.expectedStatus, w.Body.String())
			}
		})
	}
}

// TestWrappedUserReport_MethodNotAllowed tests that POST is rejected.
func TestWrappedUserReport_MethodNotAllowed(t *testing.T) {
	handler := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/{year}/user/{userID}", handler.WrappedUserReport)

	req := httptest.NewRequest(http.MethodPost, "/2025/user/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// TestWrappedLeaderboard tests the leaderboard endpoint.
// RBAC Phase 3: Leaderboard is admin-only.
func TestWrappedLeaderboard(t *testing.T) {
	handler := setupTestHandler(t)

	testCases := []struct {
		name           string
		year           string
		limit          string
		expectedStatus int
	}{
		{
			name:           "Valid year, default limit",
			year:           "2025",
			limit:          "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Custom limit",
			year:           "2025",
			limit:          "5",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid limit ignored",
			year:           "2025",
			limit:          "invalid",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Limit too high capped at 100",
			year:           "2025",
			limit:          "500",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid year",
			year:           "invalid",
			limit:          "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/{year}/leaderboard", handler.WrappedLeaderboard)

			url := "/" + tc.year + "/leaderboard"
			if tc.limit != "" {
				url += "?limit=" + tc.limit
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			// RBAC Phase 3: Add admin context for admin-only endpoint
			req = addAdminContext(req)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Status = %d, want %d, body: %s", w.Code, tc.expectedStatus, w.Body.String())
			}
		})
	}
}

// TestWrappedLeaderboard_Unauthorized tests that non-admin users cannot access leaderboard.
// RBAC Phase 3: Leaderboard is restricted to admins for privacy.
func TestWrappedLeaderboard_Unauthorized(t *testing.T) {
	handler := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/{year}/leaderboard", handler.WrappedLeaderboard)

	// Request without authentication context
	req := httptest.NewRequest(http.MethodGet, "/2025/leaderboard", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Should return 401 Unauthorized (not authenticated)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// TestWrappedLeaderboard_MethodNotAllowed tests that POST is rejected.
func TestWrappedLeaderboard_MethodNotAllowed(t *testing.T) {
	handler := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/{year}/leaderboard", handler.WrappedLeaderboard)

	req := httptest.NewRequest(http.MethodPost, "/2025/leaderboard", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// TestWrappedGenerate tests the report generation endpoint.
// RBAC Phase 3: Generate is admin-only.
func TestWrappedGenerate(t *testing.T) {
	handler := setupTestHandler(t)

	testCases := []struct {
		name           string
		year           string
		body           interface{}
		expectedStatus int
	}{
		{
			name:           "Generate for all users - empty year",
			year:           "2020",
			body:           nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid year",
			year:           "invalid",
			body:           nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Year too old",
			year:           "1999",
			body:           nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/{year}/generate", handler.WrappedGenerate)

			var body []byte
			if tc.body != nil {
				body, _ = json.Marshal(tc.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/"+tc.year+"/generate", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			// RBAC Phase 3: Add admin context for admin-only endpoint
			req = addAdminContext(req)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Status = %d, want %d, body: %s", w.Code, tc.expectedStatus, w.Body.String())
			}

			if w.Code == http.StatusOK {
				var resp models.APIResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp.Status != "success" {
					t.Errorf("Expected success status, got %s", resp.Status)
				}
			}
		})
	}
}

// TestWrappedGenerate_Unauthorized tests that non-admin users cannot generate reports.
// RBAC Phase 3: Generate is restricted to admins.
func TestWrappedGenerate_Unauthorized(t *testing.T) {
	handler := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/{year}/generate", handler.WrappedGenerate)

	// Request without authentication context
	req := httptest.NewRequest(http.MethodPost, "/2025/generate", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Should return 401 Unauthorized (not authenticated)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// TestWrappedGenerate_MethodNotAllowed tests that GET is rejected.
func TestWrappedGenerate_MethodNotAllowed(t *testing.T) {
	handler := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/{year}/generate", handler.WrappedGenerate)

	req := httptest.NewRequest(http.MethodGet, "/2025/generate", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// TestWrappedGenerate_InvalidJSON tests invalid JSON body handling.
// RBAC Phase 3: Generate is admin-only.
func TestWrappedGenerate_InvalidJSON(t *testing.T) {
	handler := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/{year}/generate", handler.WrappedGenerate)

	req := httptest.NewRequest(http.MethodPost, "/2025/generate", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = 12 // Set content length to trigger JSON parsing
	// RBAC Phase 3: Add admin context for admin-only endpoint
	req = addAdminContext(req)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestWrappedShare tests the share token endpoint.
func TestWrappedShare(t *testing.T) {
	handler := setupTestHandler(t)

	testCases := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{
			name:           "Invalid share token",
			token:          "invalid-token",
			expectedStatus: http.StatusNotFound,
		},
		// Note: Empty token returns 404 from chi router (empty path segment)
		// so we don't test for 400 here
		{
			name:           "Random UUID token not found",
			token:          "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/share/{token}", handler.WrappedShare)

			url := "/share/" + tc.token

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Status = %d, want %d, body: %s", w.Code, tc.expectedStatus, w.Body.String())
			}
		})
	}
}

// TestWrappedShare_MethodNotAllowed tests that POST is rejected.
func TestWrappedShare_MethodNotAllowed(t *testing.T) {
	handler := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/share/{token}", handler.WrappedShare)

	req := httptest.NewRequest(http.MethodPost, "/share/some-token", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// TestParseYearParam tests the year parameter parsing.
func TestParseYearParam(t *testing.T) {
	currentYear := time.Now().Year()

	testCases := []struct {
		name        string
		year        string
		expectError bool
		expected    int
	}{
		{"Valid year 2025", "2025", false, 2025},
		{"Valid year 2000", "2000", false, 2000},
		{"Valid current year", "2026", false, 2026},
		{"Invalid format", "invalid", true, 0},
		{"Year too old", "1999", true, 0},
		{"Year too far in future", "3000", true, 0},
		{"Empty", "", true, 0},
		{"Next year valid", string(rune('2')) + "027", false, 2027}, // Next year is valid
	}

	// Adjust expected value for current year test
	for i := range testCases {
		if testCases[i].name == "Valid current year" {
			testCases[i].year = string([]byte{'2', '0', byte('0' + (currentYear/10)%10), byte('0' + currentYear%10)})
			testCases[i].expected = currentYear
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/{year}", func(w http.ResponseWriter, r *http.Request) {
				year, err := parseYearParam(r)
				if tc.expectError {
					if err == nil {
						t.Errorf("Expected error for year %s", tc.year)
					}
				} else {
					if err != nil {
						t.Errorf("Unexpected error: %v", err)
					}
					if year != tc.expected {
						t.Errorf("Year = %d, want %d", year, tc.expected)
					}
				}
			})

			req := httptest.NewRequest(http.MethodGet, "/"+tc.year, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
		})
	}
}

// TestWrappedResponseFormat tests that responses have correct format.
func TestWrappedResponseFormat(t *testing.T) {
	handler := setupTestHandler(t)

	t.Run("Server stats response format", func(t *testing.T) {
		r := chi.NewRouter()
		r.Get("/{year}", handler.WrappedServerStats)

		req := httptest.NewRequest(http.MethodGet, "/2025", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp models.APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Status != "success" {
			t.Errorf("Status = %s, want success", resp.Status)
		}
		if resp.Metadata.Timestamp.IsZero() {
			t.Error("Metadata.Timestamp should not be zero")
		}
		if resp.Metadata.QueryTimeMS < 0 {
			t.Error("Metadata.QueryTimeMS should be non-negative")
		}
	})

	t.Run("Leaderboard response format", func(t *testing.T) {
		r := chi.NewRouter()
		r.Get("/{year}/leaderboard", handler.WrappedLeaderboard)

		req := httptest.NewRequest(http.MethodGet, "/2025/leaderboard", nil)
		// RBAC Phase 3: Add admin context for admin-only endpoint
		req = addAdminContext(req)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp models.APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Status != "success" {
			t.Errorf("Status = %s, want success", resp.Status)
		}
	})

	t.Run("Generate response format", func(t *testing.T) {
		r := chi.NewRouter()
		r.Post("/{year}/generate", handler.WrappedGenerate)

		req := httptest.NewRequest(http.MethodPost, "/2020/generate", nil)
		req.Header.Set("Content-Type", "application/json")
		// RBAC Phase 3: Add admin context for admin-only endpoint
		req = addAdminContext(req)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp models.APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Status != "success" {
			t.Errorf("Status = %s, want success", resp.Status)
		}

		// Parse the generate response data
		dataBytes, _ := json.Marshal(resp.Data)
		var genResp models.WrappedGenerateResponse
		if err := json.Unmarshal(dataBytes, &genResp); err != nil {
			t.Fatalf("Failed to parse generate response: %v", err)
		}

		if genResp.Year != 2020 {
			t.Errorf("Year = %d, want 2020", genResp.Year)
		}
		if genResp.DurationMS < 0 {
			t.Error("DurationMS should be non-negative")
		}
		if genResp.GeneratedAt.IsZero() {
			t.Error("GeneratedAt should not be zero")
		}
	})
}

// TestWrappedErrorFormat tests that error responses have correct format.
func TestWrappedErrorFormat(t *testing.T) {
	handler := setupTestHandler(t)

	testCases := []struct {
		name      string
		path      string
		handler   http.HandlerFunc
		errorCode string
	}{
		{"Invalid year error", "/invalid", handler.WrappedServerStats, "INVALID_YEAR"},
		{"Not found error", "/2025/user/999?generate=false", handler.WrappedUserReport, "NOT_FOUND"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/{year}", handler.WrappedServerStats)
			r.Get("/{year}/user/{userID}", handler.WrappedUserReport)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code >= 200 && w.Code < 300 {
				t.Fatalf("Expected error status, got %d", w.Code)
			}

			var resp models.APIResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if resp.Status != "error" {
				t.Errorf("Status = %s, want error", resp.Status)
			}
			if resp.Error == nil {
				t.Error("Error should not be nil")
			} else {
				if resp.Error.Code != tc.errorCode {
					t.Errorf("Error.Code = %s, want %s", resp.Error.Code, tc.errorCode)
				}
				if resp.Error.Message == "" {
					t.Error("Error.Message should not be empty")
				}
			}
		})
	}
}
