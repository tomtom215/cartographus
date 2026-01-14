// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ========================================
// ApproximateStats Tests
// ========================================

func TestApproximateStats_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(method, "/api/v1/analytics/approximate", nil)
			rec := httptest.NewRecorder()

			handler.ApproximateStats(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
			}
		})
	}
}

func TestApproximateStats_Success(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/approximate", nil)
	rec := httptest.NewRecorder()

	handler.ApproximateStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "success" {
		t.Error("Expected status=success")
	}
}

func TestApproximateStats_WithFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		query  string
		status int
	}{
		{
			name:   "with_valid_start_date",
			query:  "start_date=2024-01-01T00:00:00Z",
			status: http.StatusOK,
		},
		{
			name:   "with_valid_end_date",
			query:  "end_date=2024-12-31T23:59:59Z",
			status: http.StatusOK,
		},
		{
			name:   "with_date_range",
			query:  "start_date=2024-01-01T00:00:00Z&end_date=2024-12-31T23:59:59Z",
			status: http.StatusOK,
		},
		{
			name:   "with_users_filter",
			query:  "users=user1,user2,user3",
			status: http.StatusOK,
		},
		{
			name:   "with_media_types_filter",
			query:  "media_types=movie,episode",
			status: http.StatusOK,
		},
		{
			name:   "with_all_filters",
			query:  "start_date=2024-01-01T00:00:00Z&end_date=2024-12-31T23:59:59Z&users=user1&media_types=movie",
			status: http.StatusOK,
		},
		{
			name:   "invalid_start_date",
			query:  "start_date=invalid-date",
			status: http.StatusBadRequest,
		},
		{
			name:   "invalid_end_date",
			query:  "end_date=not-a-date",
			status: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/approximate?"+tt.query, nil)
			rec := httptest.NewRecorder()

			handler.ApproximateStats(rec, req)

			if rec.Code != tt.status {
				t.Errorf("Expected status %d, got %d", tt.status, rec.Code)
			}
		})
	}
}

// ========================================
// ApproximateDistinctCount Tests
// ========================================

func TestApproximateDistinctCount_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(method, "/api/v1/analytics/approximate/distinct?column=username", nil)
			rec := httptest.NewRecorder()

			handler.ApproximateDistinctCount(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
			}
		})
	}
}

func TestApproximateDistinctCount_MissingColumn(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/approximate/distinct", nil)
	rec := httptest.NewRecorder()

	handler.ApproximateDistinctCount(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected error object in response")
	}

	if errObj["code"] != "VALIDATION_ERROR" {
		t.Errorf("Expected error code VALIDATION_ERROR, got %v", errObj["code"])
	}
}

func TestApproximateDistinctCount_ValidColumns(t *testing.T) {
	t.Parallel()

	columns := []string{"username", "title", "ip_address", "city", "country", "platform", "player", "rating_key", "media_type"}

	for _, column := range columns {
		t.Run(column, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/approximate/distinct?column="+column, nil)
			rec := httptest.NewRecorder()

			handler.ApproximateDistinctCount(rec, req)

			// Should either succeed or return database error (not validation error)
			if rec.Code != http.StatusOK && rec.Code != http.StatusInternalServerError {
				t.Errorf("Expected status 200 or 500, got %d", rec.Code)
			}
		})
	}
}

func TestApproximateDistinctCount_WithFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		query  string
		status int
	}{
		{
			name:   "with_valid_date_range",
			query:  "column=username&start_date=2024-01-01T00:00:00Z&end_date=2024-12-31T23:59:59Z",
			status: http.StatusOK,
		},
		{
			name:   "with_users",
			query:  "column=title&users=user1,user2",
			status: http.StatusOK,
		},
		{
			name:   "with_media_types",
			query:  "column=platform&media_types=movie,episode",
			status: http.StatusOK,
		},
		{
			name:   "invalid_start_date",
			query:  "column=username&start_date=bad-date",
			status: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/approximate/distinct?"+tt.query, nil)
			rec := httptest.NewRecorder()

			handler.ApproximateDistinctCount(rec, req)

			// Accept OK or InternalServerError for valid queries (DB might not have column)
			if tt.status == http.StatusOK {
				if rec.Code != http.StatusOK && rec.Code != http.StatusInternalServerError {
					t.Errorf("Expected status 200 or 500, got %d", rec.Code)
				}
			} else if rec.Code != tt.status {
				t.Errorf("Expected status %d, got %d", tt.status, rec.Code)
			}
		})
	}
}

// ========================================
// ApproximatePercentile Tests
// ========================================

func TestApproximatePercentile_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(method, "/api/v1/analytics/approximate/percentile?column=duration&percentile=0.50", nil)
			rec := httptest.NewRecorder()

			handler.ApproximatePercentile(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
			}
		})
	}
}

func TestApproximatePercentile_MissingColumn(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/approximate/percentile?percentile=0.50", nil)
	rec := httptest.NewRecorder()

	handler.ApproximatePercentile(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected error object in response")
	}

	if errObj["message"] != "Query parameter 'column' is required" {
		t.Errorf("Expected column required error, got %v", errObj["message"])
	}
}

func TestApproximatePercentile_MissingPercentile(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/approximate/percentile?column=duration", nil)
	rec := httptest.NewRecorder()

	handler.ApproximatePercentile(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected error object in response")
	}

	if errObj["message"] != "Query parameter 'percentile' is required" {
		t.Errorf("Expected percentile required error, got %v", errObj["message"])
	}
}

func TestApproximatePercentile_InvalidPercentile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		percentile string
	}{
		{name: "non_numeric", percentile: "abc"},
		{name: "negative", percentile: "-0.5"},
		{name: "greater_than_one", percentile: "1.5"},
		{name: "way_over", percentile: "100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/approximate/percentile?column=duration&percentile="+tt.percentile, nil)
			rec := httptest.NewRecorder()

			handler.ApproximatePercentile(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			errObj, ok := response["error"].(map[string]interface{})
			if !ok {
				t.Fatal("Expected error object in response")
			}

			if errObj["message"] != "percentile must be a number between 0 and 1" {
				t.Errorf("Expected percentile range error, got %v", errObj["message"])
			}
		})
	}
}

func TestApproximatePercentile_ValidPercentiles(t *testing.T) {
	t.Parallel()

	percentiles := []string{"0", "0.25", "0.50", "0.75", "0.90", "0.95", "0.99", "1"}

	for _, p := range percentiles {
		t.Run("percentile_"+p, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/approximate/percentile?column=duration&percentile="+p, nil)
			rec := httptest.NewRecorder()

			handler.ApproximatePercentile(rec, req)

			// Should either succeed or return database error (not validation error)
			if rec.Code != http.StatusOK && rec.Code != http.StatusInternalServerError {
				t.Errorf("Expected status 200 or 500, got %d", rec.Code)
			}
		})
	}
}

func TestApproximatePercentile_ValidColumns(t *testing.T) {
	t.Parallel()

	columns := []string{"duration", "percent_complete", "paused_counter"}

	for _, column := range columns {
		t.Run(column, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/approximate/percentile?column="+column+"&percentile=0.50", nil)
			rec := httptest.NewRecorder()

			handler.ApproximatePercentile(rec, req)

			// Should either succeed or return database error (not validation error)
			if rec.Code != http.StatusOK && rec.Code != http.StatusInternalServerError {
				t.Errorf("Expected status 200 or 500, got %d", rec.Code)
			}
		})
	}
}

func TestApproximatePercentile_WithFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		query  string
		status int
	}{
		{
			name:   "with_date_range",
			query:  "column=duration&percentile=0.95&start_date=2024-01-01T00:00:00Z&end_date=2024-12-31T23:59:59Z",
			status: http.StatusOK,
		},
		{
			name:   "with_users",
			query:  "column=percent_complete&percentile=0.50&users=user1,user2",
			status: http.StatusOK,
		},
		{
			name:   "invalid_date",
			query:  "column=duration&percentile=0.50&start_date=bad",
			status: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/approximate/percentile?"+tt.query, nil)
			rec := httptest.NewRecorder()

			handler.ApproximatePercentile(rec, req)

			if tt.status == http.StatusOK {
				if rec.Code != http.StatusOK && rec.Code != http.StatusInternalServerError {
					t.Errorf("Expected status 200 or 500, got %d", rec.Code)
				}
			} else if rec.Code != tt.status {
				t.Errorf("Expected status %d, got %d", tt.status, rec.Code)
			}
		})
	}
}

// ========================================
// Response Type Tests
// ========================================

func TestApproximateStatsResponse_JSONMarshaling(t *testing.T) {
	t.Parallel()

	resp := ApproximateStatsResponse{
		Stats:        nil,
		DataSketches: false,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded["datasketches_available"] != false {
		t.Error("Expected datasketches_available=false")
	}
}

func TestApproximateDistinctResponse_JSONMarshaling(t *testing.T) {
	t.Parallel()

	resp := ApproximateDistinctResponse{
		Column:        "username",
		Count:         100,
		IsApproximate: true,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded["column"] != "username" {
		t.Errorf("Expected column=username, got %v", decoded["column"])
	}
	if decoded["count"].(float64) != 100 {
		t.Errorf("Expected count=100, got %v", decoded["count"])
	}
	if decoded["is_approximate"] != true {
		t.Error("Expected is_approximate=true")
	}
}

func TestApproximatePercentileResponse_JSONMarshaling(t *testing.T) {
	t.Parallel()

	resp := ApproximatePercentileResponse{
		Column:        "duration",
		Percentile:    0.95,
		Value:         3600.5,
		IsApproximate: false,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded["column"] != "duration" {
		t.Errorf("Expected column=duration, got %v", decoded["column"])
	}
	if decoded["percentile"].(float64) != 0.95 {
		t.Errorf("Expected percentile=0.95, got %v", decoded["percentile"])
	}
	if decoded["value"].(float64) != 3600.5 {
		t.Errorf("Expected value=3600.5, got %v", decoded["value"])
	}
	if decoded["is_approximate"] != false {
		t.Error("Expected is_approximate=false")
	}
}
