// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tomtom215/cartographus/internal/models"
)

// =============================================================================
// DedupeAuditList Tests
// =============================================================================

func TestDedupeAuditList_EmptyDatabase(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dedupe/audit", nil)
	w := httptest.NewRecorder()

	handler.DedupeAuditList(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "DedupeAuditList_Empty")

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Response data is not a map")
	}

	// entries might be nil or empty array when database is empty
	entries, _ := dataMap["entries"].([]interface{})
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}

	totalCount, ok := dataMap["total_count"].(float64)
	if !ok {
		t.Fatal("total_count is not a number")
	}

	if int(totalCount) != 0 {
		t.Errorf("Expected total_count 0, got %d", int(totalCount))
	}
}

func TestDedupeAuditList_InvalidParameters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		query          string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid_user_id",
			query:          "user_id=invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid user_id parameter",
		},
		{
			name:           "invalid_limit_negative",
			query:          "limit=-1",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid limit",
		},
		{
			name:           "invalid_limit_zero",
			query:          "limit=0",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid limit",
		},
		{
			name:           "invalid_limit_too_large",
			query:          "limit=9999",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid limit",
		},
		{
			name:           "invalid_offset_negative",
			query:          "offset=-1",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid offset",
		},
		{
			name:           "invalid_offset_non_numeric",
			query:          "offset=abc",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid offset",
		},
		{
			name:           "invalid_from_timestamp",
			query:          "from=not-a-timestamp",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid 'from' timestamp",
		},
		{
			name:           "invalid_to_timestamp",
			query:          "to=not-a-timestamp",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid 'to' timestamp",
		},
		{
			name:           "invalid_limit_non_numeric",
			query:          "limit=abc",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()

			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/dedupe/audit?"+tt.query, nil)
			w := httptest.NewRecorder()

			handler.DedupeAuditList(w, req)

			assertStatusCode(t, w.Code, tt.expectedStatus, tt.name)

			if !strings.Contains(w.Body.String(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, w.Body.String())
			}
		})
	}
}

func TestDedupeAuditList_ValidPagination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "valid_limit",
			query: "limit=50",
		},
		{
			name:  "valid_limit_max",
			query: "limit=1000",
		},
		{
			name:  "valid_offset",
			query: "offset=10",
		},
		{
			name:  "valid_limit_and_offset",
			query: "limit=50&offset=100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()

			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/dedupe/audit?"+tt.query, nil)
			w := httptest.NewRecorder()

			handler.DedupeAuditList(w, req)

			assertStatusCode(t, w.Code, http.StatusOK, tt.name)
		})
	}
}

func TestDedupeAuditList_WithValidTimeFilters(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	// Test with valid time range
	query := "from=" + yesterday.Format(time.RFC3339) + "&to=" + tomorrow.Format(time.RFC3339)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dedupe/audit?"+query, nil)
	w := httptest.NewRecorder()

	handler.DedupeAuditList(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "TimeFilter")
}

func TestDedupeAuditList_WithStringFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "filter_by_source",
			query: "source=tautulli",
		},
		{
			name:  "filter_by_status",
			query: "status=auto_dedupe",
		},
		{
			name:  "filter_by_reason",
			query: "reason=timestamp_match",
		},
		{
			name:  "filter_by_layer",
			query: "layer=cross_source",
		},
		{
			name:  "multiple_filters",
			query: "source=plex&status=user_confirmed&layer=same_source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()

			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/dedupe/audit?"+tt.query, nil)
			w := httptest.NewRecorder()

			handler.DedupeAuditList(w, req)

			assertStatusCode(t, w.Code, http.StatusOK, tt.name)
		})
	}
}

// =============================================================================
// DedupeAuditGet Tests
// =============================================================================

func TestDedupeAuditGet_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	nonExistentID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dedupe/audit/"+nonExistentID.String(), nil)
	req.SetPathValue("id", nonExistentID.String())
	w := httptest.NewRecorder()

	handler.DedupeAuditGet(w, req)

	assertStatusCode(t, w.Code, http.StatusNotFound, "DedupeAuditGet_NotFound")
}

func TestDedupeAuditGet_InvalidID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	tests := []struct {
		name string
		id   string
	}{
		{
			name: "invalid_uuid_format",
			id:   "invalid-uuid",
		},
		{
			name: "partial_uuid",
			id:   "12345",
		},
		{
			name: "too_long",
			id:   "12345678-1234-1234-1234-123456789012-extra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/api/v1/dedupe/audit/"+tt.id, nil)
			req.SetPathValue("id", tt.id)
			w := httptest.NewRecorder()

			handler.DedupeAuditGet(w, req)

			assertStatusCode(t, w.Code, http.StatusBadRequest, tt.name)
		})
	}
}

func TestDedupeAuditGet_MissingID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dedupe/audit/", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	handler.DedupeAuditGet(w, req)

	assertStatusCode(t, w.Code, http.StatusBadRequest, "DedupeAuditGet_MissingID")
}

// =============================================================================
// DedupeAuditStats Tests
// =============================================================================

func TestDedupeAuditStats_EmptyDatabase(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dedupe/audit/stats", nil)
	w := httptest.NewRecorder()

	handler.DedupeAuditStats(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "DedupeAuditStats_Empty")

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	// Verify stats structure
	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Response data is not a map")
	}

	if _, ok := dataMap["total_deduped"]; !ok {
		t.Error("Expected 'total_deduped' in stats")
	}
}

// =============================================================================
// DedupeAuditConfirm Tests
// =============================================================================

func TestDedupeAuditConfirm_InvalidID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dedupe/audit/invalid-uuid/confirm", nil)
	req.SetPathValue("id", "invalid-uuid")
	w := httptest.NewRecorder()

	handler.DedupeAuditConfirm(w, req)

	assertStatusCode(t, w.Code, http.StatusBadRequest, "DedupeAuditConfirm_InvalidID")
}

func TestDedupeAuditConfirm_MissingID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dedupe/audit//confirm", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	handler.DedupeAuditConfirm(w, req)

	assertStatusCode(t, w.Code, http.StatusBadRequest, "DedupeAuditConfirm_MissingID")
}

func TestDedupeAuditConfirm_NonExistentEntry(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	nonExistentID := uuid.New()
	body := DedupeAuditActionRequest{
		ResolvedBy: "admin_user",
		Notes:      "Test confirmation",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dedupe/audit/"+nonExistentID.String()+"/confirm", bytes.NewReader(bodyBytes))
	req.SetPathValue("id", nonExistentID.String())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.DedupeAuditConfirm(w, req)

	// Should fail to update non-existent entry
	assertStatusCode(t, w.Code, http.StatusInternalServerError, "DedupeAuditConfirm_NonExistent")
}

func TestDedupeAuditConfirm_EmptyBody(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	// Using valid but non-existent UUID
	testID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dedupe/audit/"+testID.String()+"/confirm", nil)
	req.SetPathValue("id", testID.String())
	w := httptest.NewRecorder()

	handler.DedupeAuditConfirm(w, req)

	// Should accept empty body (use defaults) but fail on non-existent entry
	// The point is it doesn't fail on JSON parsing
	if w.Code == http.StatusBadRequest && strings.Contains(w.Body.String(), "INVALID_JSON") {
		t.Error("Should not fail on empty body - defaults should be used")
	}
}

func TestDedupeAuditConfirm_InvalidJSON(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	testID := uuid.New()
	// Invalid JSON is handled gracefully (empty defaults used)
	invalidBody := bytes.NewReader([]byte(`{invalid json`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dedupe/audit/"+testID.String()+"/confirm", invalidBody)
	req.SetPathValue("id", testID.String())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.DedupeAuditConfirm(w, req)

	// Handler gracefully handles invalid JSON by using defaults
	// So it shouldn't be a BadRequest for JSON, but might fail on other operations
	if w.Code == http.StatusBadRequest && strings.Contains(w.Body.String(), "INVALID_JSON") {
		t.Error("Handler should gracefully handle invalid JSON by using defaults")
	}
}

// =============================================================================
// DedupeAuditRestore Tests
// =============================================================================

func TestDedupeAuditRestore_InvalidID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dedupe/audit/invalid-uuid/restore", nil)
	req.SetPathValue("id", "invalid-uuid")
	w := httptest.NewRecorder()

	handler.DedupeAuditRestore(w, req)

	assertStatusCode(t, w.Code, http.StatusBadRequest, "DedupeAuditRestore_InvalidID")
}

func TestDedupeAuditRestore_MissingID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dedupe/audit//restore", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	handler.DedupeAuditRestore(w, req)

	assertStatusCode(t, w.Code, http.StatusBadRequest, "DedupeAuditRestore_MissingID")
}

func TestDedupeAuditRestore_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	nonExistentID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dedupe/audit/"+nonExistentID.String()+"/restore", nil)
	req.SetPathValue("id", nonExistentID.String())
	w := httptest.NewRecorder()

	handler.DedupeAuditRestore(w, req)

	assertStatusCode(t, w.Code, http.StatusNotFound, "DedupeAuditRestore_NotFound")
}

// =============================================================================
// DedupeAuditExport Tests
// =============================================================================

func TestDedupeAuditExport_Empty(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dedupe/audit/export", nil)
	w := httptest.NewRecorder()

	handler.DedupeAuditExport(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "DedupeAuditExport_Empty")

	// Verify CSV content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/csv" {
		t.Errorf("Expected Content-Type 'text/csv', got '%s'", contentType)
	}

	// Verify Content-Disposition header
	disposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "attachment") {
		t.Errorf("Expected Content-Disposition to contain 'attachment', got '%s'", disposition)
	}

	// Verify CSV has header
	body := w.Body.String()
	if !strings.Contains(body, "id,timestamp,discarded_event_id") {
		t.Error("Expected CSV header with id,timestamp,discarded_event_id columns")
	}
}

func TestDedupeAuditExport_WithFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "filter_by_user_id",
			query: "user_id=1",
		},
		{
			name:  "filter_by_source",
			query: "source=tautulli",
		},
		{
			name:  "filter_by_status",
			query: "status=auto_dedupe",
		},
		{
			name:  "filter_by_reason",
			query: "reason=timestamp_match",
		},
		{
			name:  "filter_by_layer",
			query: "layer=cross_source",
		},
		{
			name:  "filter_with_time_range",
			query: "from=2026-01-01T00:00:00Z&to=2026-12-31T23:59:59Z",
		},
		{
			name:  "multiple_filters",
			query: "source=plex&status=user_confirmed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()

			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/dedupe/audit/export?"+tt.query, nil)
			w := httptest.NewRecorder()

			handler.DedupeAuditExport(w, req)

			assertStatusCode(t, w.Code, http.StatusOK, tt.name)

			// Verify CSV content type
			if w.Header().Get("Content-Type") != "text/csv" {
				t.Errorf("Expected Content-Type 'text/csv'")
			}
		})
	}
}

// =============================================================================
// dedupeMetadata Helper Tests
// =============================================================================

func TestDedupeMetadata(t *testing.T) {
	t.Parallel()

	start := time.Now()
	time.Sleep(10 * time.Millisecond) // Small delay to ensure measurable duration

	metadata := dedupeMetadata(start)

	if metadata.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}

	if metadata.QueryTimeMS < 10 {
		t.Errorf("Expected QueryTimeMS >= 10, got %d", metadata.QueryTimeMS)
	}
}

func TestDedupeMetadata_ImmediateReturn(t *testing.T) {
	t.Parallel()

	start := time.Now()
	metadata := dedupeMetadata(start)

	if metadata.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}

	// Query time should be >= 0 (might be 0 if very fast)
	if metadata.QueryTimeMS < 0 {
		t.Errorf("Expected QueryTimeMS >= 0, got %d", metadata.QueryTimeMS)
	}
}

// =============================================================================
// Response Structure Tests
// =============================================================================

func TestDedupeAuditListResponse_Structure(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dedupe/audit", nil)
	w := httptest.NewRecorder()

	handler.DedupeAuditList(w, req)

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response structure
	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Response data is not a map")
	}

	// Check required fields exist
	requiredFields := []string{"entries", "total_count", "limit", "offset"}
	for _, field := range requiredFields {
		if _, ok := dataMap[field]; !ok {
			t.Errorf("Missing required field '%s' in response", field)
		}
	}

	// Verify metadata
	if response.Metadata.Timestamp.IsZero() {
		t.Error("Expected non-zero metadata timestamp")
	}
}

func TestDedupeAuditStatsResponse_Structure(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dedupe/audit/stats", nil)
	w := httptest.NewRecorder()

	handler.DedupeAuditStats(w, req)

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response structure
	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Response data is not a map")
	}

	// Check total_deduped exists (key stat field)
	if _, ok := dataMap["total_deduped"]; !ok {
		t.Error("Missing 'total_deduped' in stats response")
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkDedupeAuditList_EmptyDB(b *testing.B) {
	db := setupTestDBForAPI(&testing.T{})
	defer db.Close()

	handler := setupTestHandlerWithDB(&testing.T{}, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dedupe/audit", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.DedupeAuditList(w, req)
	}
}

func BenchmarkDedupeAuditStats(b *testing.B) {
	db := setupTestDBForAPI(&testing.T{})
	defer db.Close()

	handler := setupTestHandlerWithDB(&testing.T{}, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dedupe/audit/stats", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.DedupeAuditStats(w, req)
	}
}

func BenchmarkDedupeAuditExport_EmptyDB(b *testing.B) {
	db := setupTestDBForAPI(&testing.T{})
	defer db.Close()

	handler := setupTestHandlerWithDB(&testing.T{}, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dedupe/audit/export", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.DedupeAuditExport(w, req)
	}
}
