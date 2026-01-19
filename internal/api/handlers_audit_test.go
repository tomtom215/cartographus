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
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/tomtom215/cartographus/internal/audit"
)

// =============================================================================
// Mock Audit Store
// =============================================================================

type mockAuditStore struct {
	events   []audit.Event
	stats    *audit.Stats
	queryErr error
	countErr error
	getErr   error
	statsErr error
}

func (m *mockAuditStore) Query(_ context.Context, _ audit.QueryFilter) ([]audit.Event, error) {
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	return m.events, nil
}

func (m *mockAuditStore) Count(_ context.Context, _ audit.QueryFilter) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return int64(len(m.events)), nil
}

func (m *mockAuditStore) Get(_ context.Context, id string) (*audit.Event, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for i := range m.events {
		if m.events[i].ID == id {
			return &m.events[i], nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockAuditStore) GetStats(_ context.Context) (*audit.Stats, error) {
	if m.statsErr != nil {
		return nil, m.statsErr
	}
	return m.stats, nil
}

// =============================================================================
// NewAuditHandlers Tests
// =============================================================================

func TestNewAuditHandlers(t *testing.T) {
	t.Parallel()

	store := &mockAuditStore{}
	handlers := NewAuditHandlers(nil, store)

	if handlers == nil {
		t.Fatal("NewAuditHandlers returned nil")
	}

	if handlers.store != store {
		t.Error("Store not properly assigned")
	}
}

// =============================================================================
// ListEvents Tests
// =============================================================================

func TestAuditListEvents_Success(t *testing.T) {
	t.Parallel()

	now := time.Now()
	store := &mockAuditStore{
		events: []audit.Event{
			{
				ID:        "evt-001",
				Timestamp: now,
				Type:      audit.EventTypeAuthSuccess,
				Severity:  audit.SeverityInfo,
				Actor:     audit.Actor{ID: "user-1", Type: "user"},
			},
			{
				ID:        "evt-002",
				Timestamp: now.Add(-time.Hour),
				Type:      audit.EventTypeAuthFailure,
				Severity:  audit.SeverityWarning,
				Actor:     audit.Actor{ID: "user-2", Type: "user"},
			},
		},
	}
	handlers := NewAuditHandlers(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events", nil)
	w := httptest.NewRecorder()

	handlers.ListEvents(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "ListEvents")

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	events, ok := response["events"].([]interface{})
	if !ok {
		t.Fatal("Response events is not an array")
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	if response["total"].(float64) != 2 {
		t.Errorf("Expected total 2, got %v", response["total"])
	}
}

func TestAuditListEvents_WithFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "with_limit",
			query: "limit=10",
		},
		{
			name:  "with_offset",
			query: "offset=5",
		},
		{
			name:  "with_type_filter",
			query: "type=auth.success",
		},
		{
			name:  "with_severity_filter",
			query: "severity=info",
		},
		{
			name:  "with_outcome_filter",
			query: "outcome=success",
		},
		{
			name:  "with_actor_id",
			query: "actor_id=user-123",
		},
		{
			name:  "with_actor_type",
			query: "actor_type=user",
		},
		{
			name:  "with_target_id",
			query: "target_id=resource-456",
		},
		{
			name:  "with_target_type",
			query: "target_type=playback",
		},
		{
			name:  "with_source_ip",
			query: "source_ip=192.168.1.1",
		},
		{
			name:  "with_search",
			query: "search=login",
		},
		{
			name:  "with_correlation_id",
			query: "correlation_id=corr-123",
		},
		{
			name:  "with_request_id",
			query: "request_id=req-456",
		},
		{
			name:  "with_order_by",
			query: "order_by=timestamp",
		},
		{
			name:  "with_order_asc",
			query: "order_direction=asc",
		},
		{
			name:  "with_time_range",
			query: "start_time=2026-01-01T00:00:00Z&end_time=2026-12-31T23:59:59Z",
		},
		{
			name:  "with_multiple_types",
			query: "type=auth.success&type=auth.failure",
		},
		{
			name:  "with_multiple_severities",
			query: "severity=info&severity=warning",
		},
		{
			name:  "with_multiple_outcomes",
			query: "outcome=success&outcome=failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := &mockAuditStore{events: []audit.Event{}}
			handlers := NewAuditHandlers(nil, store)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events?"+tt.query, nil)
			w := httptest.NewRecorder()

			handlers.ListEvents(w, req)

			assertStatusCode(t, w.Code, http.StatusOK, tt.name)
		})
	}
}

func TestAuditListEvents_QueryError(t *testing.T) {
	t.Parallel()

	store := &mockAuditStore{
		queryErr: errors.New("database error"),
	}
	handlers := NewAuditHandlers(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events", nil)
	w := httptest.NewRecorder()

	handlers.ListEvents(w, req)

	assertStatusCode(t, w.Code, http.StatusInternalServerError, "ListEvents_QueryError")
}

func TestAuditListEvents_CountError(t *testing.T) {
	t.Parallel()

	store := &mockAuditStore{
		events:   []audit.Event{{ID: "evt-1"}},
		countErr: errors.New("count error"),
	}
	handlers := NewAuditHandlers(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events", nil)
	w := httptest.NewRecorder()

	handlers.ListEvents(w, req)

	// Should still succeed, using len(events) as fallback
	assertStatusCode(t, w.Code, http.StatusOK, "ListEvents_CountError")
}

// =============================================================================
// GetEvent Tests
// =============================================================================

func TestAuditGetEvent_Success(t *testing.T) {
	t.Parallel()

	store := &mockAuditStore{
		events: []audit.Event{
			{
				ID:        "evt-001",
				Timestamp: time.Now(),
				Type:      audit.EventTypeAuthSuccess,
			},
		},
	}
	handlers := NewAuditHandlers(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events/evt-001", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "evt-001")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handlers.GetEvent(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "GetEvent")
}

func TestAuditGetEvent_MissingID(t *testing.T) {
	t.Parallel()

	store := &mockAuditStore{}
	handlers := NewAuditHandlers(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handlers.GetEvent(w, req)

	assertStatusCode(t, w.Code, http.StatusBadRequest, "GetEvent_MissingID")
}

func TestAuditGetEvent_NotFound(t *testing.T) {
	t.Parallel()

	store := &mockAuditStore{
		events: []audit.Event{},
	}
	handlers := NewAuditHandlers(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events/non-existent", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "non-existent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handlers.GetEvent(w, req)

	assertStatusCode(t, w.Code, http.StatusNotFound, "GetEvent_NotFound")
}

// =============================================================================
// GetStats Tests
// =============================================================================

func TestAuditGetStats_Success(t *testing.T) {
	t.Parallel()

	store := &mockAuditStore{
		stats: &audit.Stats{
			TotalEvents:      100,
			EventsByType:     map[string]int64{string(audit.EventTypeAuthSuccess): 50, string(audit.EventTypeAuthFailure): 50},
			EventsBySeverity: map[string]int64{string(audit.SeverityInfo): 80, string(audit.SeverityWarning): 20},
		},
	}
	handlers := NewAuditHandlers(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/stats", nil)
	w := httptest.NewRecorder()

	handlers.GetStats(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "GetStats")

	var response audit.Stats
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.TotalEvents != 100 {
		t.Errorf("Expected TotalEvents 100, got %d", response.TotalEvents)
	}
}

func TestAuditGetStats_Error(t *testing.T) {
	t.Parallel()

	store := &mockAuditStore{
		statsErr: errors.New("stats error"),
	}
	handlers := NewAuditHandlers(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/stats", nil)
	w := httptest.NewRecorder()

	handlers.GetStats(w, req)

	assertStatusCode(t, w.Code, http.StatusInternalServerError, "GetStats_Error")
}

// =============================================================================
// GetTypes Tests
// =============================================================================

func TestAuditGetTypes_Success(t *testing.T) {
	t.Parallel()

	store := &mockAuditStore{}
	handlers := NewAuditHandlers(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/types", nil)
	w := httptest.NewRecorder()

	handlers.GetTypes(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "GetTypes")

	var response map[string][]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	types, ok := response["types"]
	if !ok {
		t.Fatal("Expected 'types' in response")
	}

	// Verify some expected types are present
	expectedTypes := []string{"auth.success", "auth.failure", "auth.logout", "config.changed"}
	for _, expected := range expectedTypes {
		found := false
		for _, t := range types {
			if t == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected type '%s' not found in response", expected)
		}
	}
}

// =============================================================================
// GetSeverities Tests
// =============================================================================

func TestAuditGetSeverities_Success(t *testing.T) {
	t.Parallel()

	store := &mockAuditStore{}
	handlers := NewAuditHandlers(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/severities", nil)
	w := httptest.NewRecorder()

	handlers.GetSeverities(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "GetSeverities")

	var response map[string][]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	severities, ok := response["severities"]
	if !ok {
		t.Fatal("Expected 'severities' in response")
	}

	// Verify all expected severities are present
	expectedSeverities := []string{"debug", "info", "warning", "error", "critical"}
	for _, expected := range expectedSeverities {
		found := false
		for _, s := range severities {
			if s == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected severity '%s' not found in response", expected)
		}
	}
}

// =============================================================================
// ExportEvents Tests
// =============================================================================

func TestAuditExportEvents_JSON(t *testing.T) {
	t.Parallel()

	store := &mockAuditStore{
		events: []audit.Event{
			{
				ID:        "evt-001",
				Timestamp: time.Now(),
				Type:      audit.EventTypeAuthSuccess,
			},
		},
	}
	handlers := NewAuditHandlers(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/export?format=json", nil)
	w := httptest.NewRecorder()

	handlers.ExportEvents(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "ExportEvents_JSON")

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	disposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "audit-events.json") {
		t.Errorf("Expected filename 'audit-events.json' in Content-Disposition, got '%s'", disposition)
	}
}

func TestAuditExportEvents_CEF(t *testing.T) {
	t.Parallel()

	store := &mockAuditStore{
		events: []audit.Event{
			{
				ID:        "evt-001",
				Timestamp: time.Now(),
				Type:      audit.EventTypeAuthSuccess,
				Severity:  audit.SeverityInfo,
			},
		},
	}
	handlers := NewAuditHandlers(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/export?format=cef", nil)
	w := httptest.NewRecorder()

	handlers.ExportEvents(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "ExportEvents_CEF")

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Expected Content-Type 'text/plain', got '%s'", contentType)
	}

	disposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "audit-events.cef") {
		t.Errorf("Expected filename 'audit-events.cef' in Content-Disposition, got '%s'", disposition)
	}
}

func TestAuditExportEvents_DefaultFormat(t *testing.T) {
	t.Parallel()

	store := &mockAuditStore{events: []audit.Event{}}
	handlers := NewAuditHandlers(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/export", nil)
	w := httptest.NewRecorder()

	handlers.ExportEvents(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "ExportEvents_Default")

	// Default should be JSON
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

func TestAuditExportEvents_WithFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "with_time_range",
			query: "start_time=2026-01-01T00:00:00Z&end_time=2026-12-31T23:59:59Z",
		},
		{
			name:  "with_type_filter",
			query: "type=auth.success",
		},
		{
			name:  "with_multiple_types",
			query: "type=auth.success&type=auth.failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := &mockAuditStore{events: []audit.Event{}}
			handlers := NewAuditHandlers(nil, store)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/export?"+tt.query, nil)
			w := httptest.NewRecorder()

			handlers.ExportEvents(w, req)

			assertStatusCode(t, w.Code, http.StatusOK, tt.name)
		})
	}
}

func TestAuditExportEvents_QueryError(t *testing.T) {
	t.Parallel()

	store := &mockAuditStore{
		queryErr: errors.New("database error"),
	}
	handlers := NewAuditHandlers(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/export", nil)
	w := httptest.NewRecorder()

	handlers.ExportEvents(w, req)

	assertStatusCode(t, w.Code, http.StatusInternalServerError, "ExportEvents_QueryError")
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkAuditListEvents(b *testing.B) {
	events := make([]audit.Event, 100)
	for i := range events {
		events[i] = audit.Event{
			ID:        "evt-" + string(rune('0'+i%10)),
			Timestamp: time.Now(),
			Type:      audit.EventTypeAuthSuccess,
		}
	}

	store := &mockAuditStore{events: events}
	handlers := NewAuditHandlers(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handlers.ListEvents(w, req)
	}
}

func BenchmarkAuditGetStats(b *testing.B) {
	store := &mockAuditStore{
		stats: &audit.Stats{
			TotalEvents: 1000,
		},
	}
	handlers := NewAuditHandlers(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/stats", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handlers.GetStats(w, req)
	}
}
