// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// =============================================================================
// Mock DLQ Store
// =============================================================================

type mockDLQStore struct {
	entries          []DLQEntryInternal
	stats            DLQStatsInternal
	maxRetries       int
	retryErr         error
	retryAllErr      error
	retryAllCount    int
	cleanupCount     int
}

func (m *mockDLQStore) ListEntries() []DLQEntryInternal {
	return m.entries
}

func (m *mockDLQStore) GetEntry(eventID string) *DLQEntryInternal {
	for i := range m.entries {
		if m.entries[i].EventID == eventID {
			return &m.entries[i]
		}
	}
	return nil
}

func (m *mockDLQStore) RemoveEntry(eventID string) bool {
	for i := range m.entries {
		if m.entries[i].EventID == eventID {
			m.entries = append(m.entries[:i], m.entries[i+1:]...)
			return true
		}
	}
	return false
}

func (m *mockDLQStore) GetPendingRetries() []DLQEntryInternal {
	var pending []DLQEntryInternal
	for _, e := range m.entries {
		if e.RetryCount < m.maxRetries && time.Now().After(e.NextRetry) {
			pending = append(pending, e)
		}
	}
	return pending
}

func (m *mockDLQStore) Stats() DLQStatsInternal {
	return m.stats
}

func (m *mockDLQStore) RetryEntry(_ string) error {
	return m.retryErr
}

func (m *mockDLQStore) RetryAllPending() (int, error) {
	if m.retryAllErr != nil {
		return 0, m.retryAllErr
	}
	return m.retryAllCount, nil
}

func (m *mockDLQStore) GetMaxRetries() int {
	return m.maxRetries
}

func (m *mockDLQStore) Cleanup() int {
	return m.cleanupCount
}

// =============================================================================
// NewDLQHandlers Tests
// =============================================================================

func TestNewDLQHandlers(t *testing.T) {
	t.Parallel()

	store := &mockDLQStore{}
	handlers := NewDLQHandlers(store, 5)

	if handlers == nil {
		t.Fatal("NewDLQHandlers returned nil")
	}

	if handlers.store != store {
		t.Error("Store not properly assigned")
	}

	if handlers.maxRetries != 5 {
		t.Errorf("Expected maxRetries 5, got %d", handlers.maxRetries)
	}
}

// =============================================================================
// ListEntries Tests
// =============================================================================

func TestDLQListEntries_Success(t *testing.T) {
	t.Parallel()

	now := time.Now()
	store := &mockDLQStore{
		entries: []DLQEntryInternal{
			{
				EventID:      "evt-001",
				MessageID:    "msg-001",
				Source:       "tautulli",
				Username:     "user1",
				RetryCount:   0,
				FirstFailure: now,
				LastFailure:  now,
				Category:     "database",
			},
			{
				EventID:      "evt-002",
				MessageID:    "msg-002",
				Source:       "plex",
				Username:     "user2",
				RetryCount:   1,
				FirstFailure: now.Add(-time.Hour),
				LastFailure:  now,
				Category:     "timeout",
			},
		},
		maxRetries: 5,
	}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dlq/entries", nil)
	w := httptest.NewRecorder()

	handlers.ListEntries(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "ListEntries")

	var response DLQEntriesResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(response.Entries))
	}

	if response.Total != 2 {
		t.Errorf("Expected total 2, got %d", response.Total)
	}
}

func TestDLQListEntries_WithPagination(t *testing.T) {
	t.Parallel()

	now := time.Now()
	entries := make([]DLQEntryInternal, 100)
	for i := range entries {
		entries[i] = DLQEntryInternal{
			EventID:      "evt-" + string(rune('0'+i/10)) + string(rune('0'+i%10)),
			FirstFailure: now,
			LastFailure:  now,
		}
	}

	store := &mockDLQStore{
		entries:    entries,
		maxRetries: 5,
	}
	handlers := NewDLQHandlers(store, 5)

	tests := []struct {
		name          string
		query         string
		expectedCount int
	}{
		{
			name:          "default_limit",
			query:         "",
			expectedCount: 50, // Default limit
		},
		{
			name:          "custom_limit",
			query:         "limit=10",
			expectedCount: 10,
		},
		{
			name:          "with_offset",
			query:         "limit=10&offset=95",
			expectedCount: 5, // Only 5 entries left after offset
		},
		{
			name:          "max_limit",
			query:         "limit=1000",
			expectedCount: 100, // Capped at actual entries
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/dlq/entries?"+tt.query, nil)
			w := httptest.NewRecorder()

			handlers.ListEntries(w, req)

			assertStatusCode(t, w.Code, http.StatusOK, tt.name)

			var response DLQEntriesResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if len(response.Entries) != tt.expectedCount {
				t.Errorf("Expected %d entries, got %d", tt.expectedCount, len(response.Entries))
			}
		})
	}
}

func TestDLQListEntries_WithFilters(t *testing.T) {
	t.Parallel()

	now := time.Now()
	store := &mockDLQStore{
		entries: []DLQEntryInternal{
			{
				EventID:    "evt-001",
				Category:   "database",
				RetryCount: 0,
				FirstFailure: now,
				LastFailure:  now,
			},
			{
				EventID:    "evt-002",
				Category:   "timeout",
				RetryCount: 3,
				FirstFailure: now,
				LastFailure:  now,
			},
			{
				EventID:    "evt-003",
				Category:   "database",
				RetryCount: 5, // At max retries
				FirstFailure: now,
				LastFailure:  now,
			},
		},
		maxRetries: 5,
	}
	handlers := NewDLQHandlers(store, 5)

	tests := []struct {
		name          string
		query         string
		expectedCount int
	}{
		{
			name:          "filter_by_category",
			query:         "category=database",
			expectedCount: 2,
		},
		{
			name:          "filter_by_status_pending",
			query:         "status=pending",
			expectedCount: 1, // RetryCount 0
		},
		{
			name:          "filter_by_status_retrying",
			query:         "status=retrying",
			expectedCount: 1, // RetryCount > 0 && < max
		},
		{
			name:          "filter_by_status_permanent",
			query:         "status=permanent",
			expectedCount: 1, // RetryCount >= max
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/dlq/entries?"+tt.query, nil)
			w := httptest.NewRecorder()

			handlers.ListEntries(w, req)

			assertStatusCode(t, w.Code, http.StatusOK, tt.name)

			var response DLQEntriesResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if len(response.Entries) != tt.expectedCount {
				t.Errorf("Expected %d entries, got %d", tt.expectedCount, len(response.Entries))
			}
		})
	}
}

func TestDLQListEntries_Empty(t *testing.T) {
	t.Parallel()

	store := &mockDLQStore{
		entries:    []DLQEntryInternal{},
		maxRetries: 5,
	}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dlq/entries", nil)
	w := httptest.NewRecorder()

	handlers.ListEntries(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "ListEntries_Empty")

	var response DLQEntriesResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Total != 0 {
		t.Errorf("Expected total 0, got %d", response.Total)
	}
}

// =============================================================================
// GetEntry Tests
// =============================================================================

func TestDLQGetEntry_Success(t *testing.T) {
	t.Parallel()

	now := time.Now()
	store := &mockDLQStore{
		entries: []DLQEntryInternal{
			{
				EventID:       "evt-001",
				MessageID:     "msg-001",
				Source:        "tautulli",
				Username:      "testuser",
				MediaTitle:    "Test Movie",
				OriginalError: "connection timeout",
				RetryCount:    2,
				FirstFailure:  now.Add(-time.Hour),
				LastFailure:   now,
				Category:      "timeout",
			},
		},
		maxRetries: 5,
	}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dlq/entries/evt-001", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "evt-001")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handlers.GetEntry(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "GetEntry")

	var response DLQEntry
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.EventID != "evt-001" {
		t.Errorf("Expected EventID 'evt-001', got '%s'", response.EventID)
	}

	if response.Status != "retrying" {
		t.Errorf("Expected Status 'retrying', got '%s'", response.Status)
	}

	if response.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries 5, got %d", response.MaxRetries)
	}
}

func TestDLQGetEntry_MissingID(t *testing.T) {
	t.Parallel()

	store := &mockDLQStore{}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dlq/entries/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handlers.GetEntry(w, req)

	assertStatusCode(t, w.Code, http.StatusBadRequest, "GetEntry_MissingID")
}

func TestDLQGetEntry_NotFound(t *testing.T) {
	t.Parallel()

	store := &mockDLQStore{
		entries: []DLQEntryInternal{},
	}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dlq/entries/non-existent", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "non-existent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handlers.GetEntry(w, req)

	assertStatusCode(t, w.Code, http.StatusNotFound, "GetEntry_NotFound")
}

// =============================================================================
// RetryEntry Tests
// =============================================================================

func TestDLQRetryEntry_Success(t *testing.T) {
	t.Parallel()

	now := time.Now()
	store := &mockDLQStore{
		entries: []DLQEntryInternal{
			{
				EventID:      "evt-001",
				FirstFailure: now,
				LastFailure:  now,
			},
		},
		maxRetries: 5,
	}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dlq/entries/evt-001/retry", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "evt-001")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handlers.RetryEntry(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "RetryEntry")

	var response DLQRetryResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("Expected Success to be true")
	}

	if response.RetriedCount != 1 {
		t.Errorf("Expected RetriedCount 1, got %d", response.RetriedCount)
	}
}

func TestDLQRetryEntry_MissingID(t *testing.T) {
	t.Parallel()

	store := &mockDLQStore{}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dlq/entries//retry", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handlers.RetryEntry(w, req)

	assertStatusCode(t, w.Code, http.StatusBadRequest, "RetryEntry_MissingID")
}

func TestDLQRetryEntry_NotFound(t *testing.T) {
	t.Parallel()

	store := &mockDLQStore{
		entries: []DLQEntryInternal{},
	}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dlq/entries/non-existent/retry", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "non-existent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handlers.RetryEntry(w, req)

	assertStatusCode(t, w.Code, http.StatusNotFound, "RetryEntry_NotFound")
}

func TestDLQRetryEntry_Error(t *testing.T) {
	t.Parallel()

	now := time.Now()
	store := &mockDLQStore{
		entries: []DLQEntryInternal{
			{EventID: "evt-001", FirstFailure: now, LastFailure: now},
		},
		retryErr: errors.New("retry failed"),
	}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dlq/entries/evt-001/retry", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "evt-001")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handlers.RetryEntry(w, req)

	assertStatusCode(t, w.Code, http.StatusInternalServerError, "RetryEntry_Error")
}

// =============================================================================
// DeleteEntry Tests
// =============================================================================

func TestDLQDeleteEntry_Success(t *testing.T) {
	t.Parallel()

	now := time.Now()
	store := &mockDLQStore{
		entries: []DLQEntryInternal{
			{EventID: "evt-001", FirstFailure: now, LastFailure: now},
		},
	}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/dlq/entries/evt-001", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "evt-001")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handlers.DeleteEntry(w, req)

	assertStatusCode(t, w.Code, http.StatusNoContent, "DeleteEntry")
}

func TestDLQDeleteEntry_MissingID(t *testing.T) {
	t.Parallel()

	store := &mockDLQStore{}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/dlq/entries/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handlers.DeleteEntry(w, req)

	assertStatusCode(t, w.Code, http.StatusBadRequest, "DeleteEntry_MissingID")
}

func TestDLQDeleteEntry_NotFound(t *testing.T) {
	t.Parallel()

	store := &mockDLQStore{
		entries: []DLQEntryInternal{},
	}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/dlq/entries/non-existent", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "non-existent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handlers.DeleteEntry(w, req)

	assertStatusCode(t, w.Code, http.StatusNotFound, "DeleteEntry_NotFound")
}

// =============================================================================
// GetStats Tests
// =============================================================================

func TestDLQGetStats_Success(t *testing.T) {
	t.Parallel()

	now := time.Now()
	store := &mockDLQStore{
		entries: []DLQEntryInternal{
			{EventID: "evt-001", RetryCount: 0, FirstFailure: now, LastFailure: now},
			{EventID: "evt-002", RetryCount: 2, FirstFailure: now, LastFailure: now},
			{EventID: "evt-003", RetryCount: 5, FirstFailure: now, LastFailure: now},
		},
		stats: DLQStatsInternal{
			TotalEntries:      3,
			TotalAdded:        10,
			TotalRemoved:      7,
			TotalRetries:      15,
			TotalExpired:      2,
			OldestEntry:       now.Add(-24 * time.Hour),
			NewestEntry:       now,
			EntriesByCategory: map[string]int64{"database": 2, "timeout": 1},
		},
		maxRetries: 5,
	}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dlq/stats", nil)
	w := httptest.NewRecorder()

	handlers.GetStats(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "GetStats")

	var response DLQStats
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.TotalEntries != 3 {
		t.Errorf("Expected TotalEntries 3, got %d", response.TotalEntries)
	}

	if response.TotalAdded != 10 {
		t.Errorf("Expected TotalAdded 10, got %d", response.TotalAdded)
	}

	if response.OldestEntryAge == nil {
		t.Error("Expected OldestEntryAge to be set")
	}

	if response.NewestEntryAge == nil {
		t.Error("Expected NewestEntryAge to be set")
	}

	// Verify entries by status
	if response.EntriesByStatus["pending"] != 1 {
		t.Errorf("Expected 1 pending entry, got %d", response.EntriesByStatus["pending"])
	}
	if response.EntriesByStatus["retrying"] != 1 {
		t.Errorf("Expected 1 retrying entry, got %d", response.EntriesByStatus["retrying"])
	}
	if response.EntriesByStatus["permanent"] != 1 {
		t.Errorf("Expected 1 permanent entry, got %d", response.EntriesByStatus["permanent"])
	}
}

func TestDLQGetStats_Empty(t *testing.T) {
	t.Parallel()

	store := &mockDLQStore{
		entries: []DLQEntryInternal{},
		stats: DLQStatsInternal{
			TotalEntries:      0,
			EntriesByCategory: map[string]int64{},
		},
		maxRetries: 5,
	}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dlq/stats", nil)
	w := httptest.NewRecorder()

	handlers.GetStats(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "GetStats_Empty")

	var response DLQStats
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.TotalEntries != 0 {
		t.Errorf("Expected TotalEntries 0, got %d", response.TotalEntries)
	}

	// Oldest/Newest should be nil for empty store
	if response.OldestEntryAge != nil {
		t.Error("Expected OldestEntryAge to be nil for empty store")
	}
}

// =============================================================================
// RetryAllPending Tests
// =============================================================================

func TestDLQRetryAllPending_Success(t *testing.T) {
	t.Parallel()

	store := &mockDLQStore{
		retryAllCount: 5,
	}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dlq/retry-all", nil)
	w := httptest.NewRecorder()

	handlers.RetryAllPending(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "RetryAllPending")

	var response DLQRetryResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("Expected Success to be true")
	}

	if response.RetriedCount != 5 {
		t.Errorf("Expected RetriedCount 5, got %d", response.RetriedCount)
	}
}

func TestDLQRetryAllPending_Error(t *testing.T) {
	t.Parallel()

	store := &mockDLQStore{
		retryAllErr: errors.New("retry failed"),
	}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dlq/retry-all", nil)
	w := httptest.NewRecorder()

	handlers.RetryAllPending(w, req)

	assertStatusCode(t, w.Code, http.StatusInternalServerError, "RetryAllPending_Error")
}

// =============================================================================
// Cleanup Tests
// =============================================================================

func TestDLQCleanup_Success(t *testing.T) {
	t.Parallel()

	store := &mockDLQStore{
		cleanupCount: 3,
	}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dlq/cleanup", nil)
	w := httptest.NewRecorder()

	handlers.Cleanup(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "Cleanup")

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["cleaned_count"].(float64) != 3 {
		t.Errorf("Expected cleaned_count 3, got %v", response["cleaned_count"])
	}
}

// =============================================================================
// GetCategories Tests
// =============================================================================

func TestDLQGetCategories_Success(t *testing.T) {
	t.Parallel()

	store := &mockDLQStore{}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dlq/categories", nil)
	w := httptest.NewRecorder()

	handlers.GetCategories(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "GetCategories")

	var response map[string][]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	categories, ok := response["categories"]
	if !ok {
		t.Fatal("Expected 'categories' in response")
	}

	expectedCategories := []string{"unknown", "connection", "timeout", "validation", "database", "capacity"}
	for _, expected := range expectedCategories {
		found := false
		for _, c := range categories {
			if c == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected category '%s' not found in response", expected)
		}
	}
}

// =============================================================================
// getStatus Tests
// =============================================================================

func TestDLQGetStatus(t *testing.T) {
	t.Parallel()

	handlers := &DLQHandlers{maxRetries: 5}

	tests := []struct {
		name       string
		retryCount int
		expected   string
	}{
		{
			name:       "pending",
			retryCount: 0,
			expected:   "pending",
		},
		{
			name:       "retrying_1",
			retryCount: 1,
			expected:   "retrying",
		},
		{
			name:       "retrying_4",
			retryCount: 4,
			expected:   "retrying",
		},
		{
			name:       "permanent_at_max",
			retryCount: 5,
			expected:   "permanent",
		},
		{
			name:       "permanent_over_max",
			retryCount: 10,
			expected:   "permanent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := handlers.getStatus(tt.retryCount)
			if status != tt.expected {
				t.Errorf("Expected status '%s', got '%s'", tt.expected, status)
			}
		})
	}
}

// =============================================================================
// convertEntry Tests
// =============================================================================

func TestDLQConvertEntry(t *testing.T) {
	t.Parallel()

	now := time.Now()
	handlers := &DLQHandlers{maxRetries: 5}

	internal := &DLQEntryInternal{
		EventID:       "evt-001",
		MessageID:     "msg-001",
		Source:        "tautulli",
		Username:      "testuser",
		MediaTitle:    "Test Movie",
		OriginalError: "original error",
		LastError:     "last error",
		RetryCount:    2,
		FirstFailure:  now.Add(-time.Hour),
		LastFailure:   now,
		NextRetry:     now.Add(time.Minute),
		Category:      "database",
	}

	entry := handlers.convertEntry(internal)

	if entry.EventID != "evt-001" {
		t.Errorf("Expected EventID 'evt-001', got '%s'", entry.EventID)
	}

	if entry.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries 5, got %d", entry.MaxRetries)
	}

	if entry.Status != "retrying" {
		t.Errorf("Expected Status 'retrying', got '%s'", entry.Status)
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkDLQListEntries(b *testing.B) {
	now := time.Now()
	entries := make([]DLQEntryInternal, 1000)
	for i := range entries {
		entries[i] = DLQEntryInternal{
			EventID:      "evt-" + string(rune('0'+i/100)) + string(rune('0'+(i/10)%10)) + string(rune('0'+i%10)),
			FirstFailure: now,
			LastFailure:  now,
		}
	}

	store := &mockDLQStore{entries: entries, maxRetries: 5}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dlq/entries", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handlers.ListEntries(w, req)
	}
}

func BenchmarkDLQGetStats(b *testing.B) {
	store := &mockDLQStore{
		stats: DLQStatsInternal{
			TotalEntries: 100,
		},
		maxRetries: 5,
	}
	handlers := NewDLQHandlers(store, 5)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dlq/stats", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handlers.GetStats(w, req)
	}
}
