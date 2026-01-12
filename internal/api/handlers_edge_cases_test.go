// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/models"
)

// TestRespondJSON_WithUnmarshalableData tests respondJSON with data that fails to marshal
func TestRespondJSON_WithUnmarshalableData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "channel",
			data: make(chan int),
		},
		{
			name: "function",
			data: func() {},
		},
		{
			name: "complex number",
			data: complex(1, 2),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			response := &models.APIResponse{
				Status: "success",
				Data:   tt.data,
			}

			respondJSON(w, http.StatusOK, response)

			// Should return 500 due to marshaling error
			if w.Code != http.StatusInternalServerError {
				t.Errorf("Expected status 500 for unmarshalable data, got %d", w.Code)
			}
		})
	}
}

// TestRespondJSON_WithSpecialFloatValues tests handling of special float values
func TestRespondJSON_WithSpecialFloatValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value float64
	}{
		{
			name:  "positive infinity",
			value: math.Inf(1),
		},
		{
			name:  "negative infinity",
			value: math.Inf(-1),
		},
		{
			name:  "NaN",
			value: math.NaN(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			response := &models.APIResponse{
				Status: "success",
				Data: map[string]interface{}{
					"value": tt.value,
				},
			}

			respondJSON(w, http.StatusOK, response)

			// JSON encoding of Inf/NaN should fail
			if w.Code != http.StatusInternalServerError {
				t.Errorf("Expected status 500 for %s, got %d", tt.name, w.Code)
			}
		})
	}
}

// TestStats_NilDatabase tests Stats handler with nil database
func TestStats_NilDatabase(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db: nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	w := httptest.NewRecorder()

	handler.Stats(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for nil database, got %d", w.Code)
	}

	var response models.APIResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Error == nil || response.Error.Code != "SERVICE_ERROR" {
		t.Error("Expected SERVICE_ERROR for nil database")
	}
}

// TestStats_WithNilSync tests Stats when sync manager is nil
func TestStats_WithNilSync(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := &Handler{
		db:   db,
		sync: nil, // Nil sync manager
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	w := httptest.NewRecorder()

	handler.Stats(w, req)

	// Should still succeed with nil sync
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 with nil sync, got %d", w.Code)
	}
}

// TestLocations_ExtremeLimit tests Locations with boundary limit values
func TestLocations_ExtremeLimit(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := &Handler{
		db: db,
	}

	tests := []struct {
		name       string
		limit      string
		wantStatus int
	}{
		{
			name:       "zero limit",
			limit:      "0",
			wantStatus: http.StatusBadRequest, // Validation rejects < 1
		},
		{
			name:       "negative limit",
			limit:      "-1",
			wantStatus: http.StatusBadRequest, // Validation rejects < 1
		},
		{
			name:       "minimum valid",
			limit:      "1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "maximum valid",
			limit:      "1000",
			wantStatus: http.StatusOK,
		},
		{
			name:       "over maximum",
			limit:      "1001",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/locations?limit="+tt.limit, nil)
			w := httptest.NewRecorder()

			handler.Locations(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d for %s, got %d", tt.wantStatus, tt.name, w.Code)
			}
		})
	}
}

// TestLocations_DaysRangeValidation tests days parameter validation
func TestLocations_DaysRangeValidation(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := &Handler{
		db: db,
	}

	tests := []struct {
		name       string
		days       string
		wantStatus int
	}{
		{
			name:       "zero days",
			days:       "0",
			wantStatus: http.StatusBadRequest, // Validation rejects < 1
		},
		{
			name:       "minimum valid",
			days:       "1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "maximum valid",
			days:       "3650",
			wantStatus: http.StatusOK,
		},
		{
			name:       "over maximum",
			days:       "3651",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "negative days",
			days:       "-1",
			wantStatus: http.StatusBadRequest, // Validation rejects < 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/locations?days="+tt.days, nil)
			w := httptest.NewRecorder()

			handler.Locations(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d for %s, got %d", tt.wantStatus, tt.name, w.Code)
			}
		})
	}
}

// TestLocations_NilDatabase tests Locations with nil database
func TestLocations_NilDatabase(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db: nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/locations", nil)
	w := httptest.NewRecorder()

	handler.Locations(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for nil database, got %d", w.Code)
	}
}

// TestTriggerSync_MethodNotAllowed tests TriggerSync with invalid HTTP methods
func TestTriggerSync_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{}

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/sync/trigger", nil)
			w := httptest.NewRecorder()

			handler.TriggerSync(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestTriggerSync_NilSyncManager tests TriggerSync when sync manager is nil
func TestTriggerSync_NilSyncManager(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		sync: nil,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/trigger", nil)
	w := httptest.NewRecorder()

	handler.TriggerSync(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for nil sync manager, got %d", w.Code)
	}

	var response models.APIResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Error == nil || response.Error.Code != "SERVICE_ERROR" {
		t.Error("Expected SERVICE_ERROR for nil sync manager")
	}
}

// TestOnSyncCompleted_NilDatabase tests OnSyncCompleted when database is nil
func TestOnSyncCompleted_NilDatabase(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db: nil,
	}

	// Should not panic with nil database
	handler.OnSyncCompleted(10, 0)
}
