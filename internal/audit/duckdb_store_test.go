// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build integration

package audit

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/goccy/go-json"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory DuckDB: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func TestDuckDBStore_CreateTable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	err := store.CreateTable(ctx)
	if err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	// Verify table exists
	var tableName string
	err = db.QueryRowContext(ctx, "SELECT table_name FROM information_schema.tables WHERE table_name = 'audit_events'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Table audit_events does not exist: %v", err)
	}
	if tableName != "audit_events" {
		t.Errorf("Expected table name 'audit_events', got '%s'", tableName)
	}
}

func TestDuckDBStore_Save(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	event := &Event{
		ID:        "test-event-1",
		Timestamp: time.Now().UTC(),
		Type:      EventTypeAuthSuccess,
		Severity:  SeverityInfo,
		Outcome:   OutcomeSuccess,
		Actor: Actor{
			ID:         "user-123",
			Type:       "user",
			Name:       "testuser",
			Roles:      []string{"admin", "viewer"},
			SessionID:  "session-abc",
			AuthMethod: "jwt",
		},
		Target: &Target{
			ID:   "resource-456",
			Type: "config",
			Name: "auth_settings",
		},
		Source: Source{
			IPAddress: "192.168.1.100",
			UserAgent: "Mozilla/5.0",
			Hostname:  "localhost",
			Port:      8080,
		},
		Action:        "login",
		Description:   "User logged in successfully",
		Metadata:      json.RawMessage(`{"method":"password"}`),
		CorrelationID: "corr-789",
		RequestID:     "req-xyz",
	}

	err := store.Save(ctx, event)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify event was saved
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_events WHERE id = ?", event.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query saved event: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 event, got %d", count)
	}
}

func TestDuckDBStore_Get(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	// Save an event first
	originalEvent := &Event{
		ID:        "test-get-event",
		Timestamp: time.Now().UTC().Truncate(time.Microsecond),
		Type:      EventTypeAuthFailure,
		Severity:  SeverityWarning,
		Outcome:   OutcomeFailure,
		Actor: Actor{
			ID:   "user-456",
			Type: "user",
			Name: "baduser",
		},
		Source: Source{
			IPAddress: "10.0.0.1",
		},
		Action:      "login",
		Description: "Invalid password",
	}

	if err := store.Save(ctx, originalEvent); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Get the event
	retrieved, err := store.Get(ctx, originalEvent.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Verify fields
	if retrieved.ID != originalEvent.ID {
		t.Errorf("ID mismatch: expected %s, got %s", originalEvent.ID, retrieved.ID)
	}
	if retrieved.Type != originalEvent.Type {
		t.Errorf("Type mismatch: expected %s, got %s", originalEvent.Type, retrieved.Type)
	}
	if retrieved.Severity != originalEvent.Severity {
		t.Errorf("Severity mismatch: expected %s, got %s", originalEvent.Severity, retrieved.Severity)
	}
	if retrieved.Actor.ID != originalEvent.Actor.ID {
		t.Errorf("Actor.ID mismatch: expected %s, got %s", originalEvent.Actor.ID, retrieved.Actor.ID)
	}
}

func TestDuckDBStore_Get_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	_, err := store.Get(ctx, "nonexistent-id")
	if err == nil {
		t.Error("Expected error for nonexistent event, got nil")
	}
}

func TestDuckDBStore_Query(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	// Save multiple events
	now := time.Now().UTC()
	events := []*Event{
		{
			ID:          "event-1",
			Timestamp:   now.Add(-2 * time.Hour),
			Type:        EventTypeAuthSuccess,
			Severity:    SeverityInfo,
			Outcome:     OutcomeSuccess,
			Actor:       Actor{ID: "user-1", Type: "user"},
			Source:      Source{IPAddress: "192.168.1.1"},
			Action:      "login",
			Description: "Login success",
		},
		{
			ID:          "event-2",
			Timestamp:   now.Add(-1 * time.Hour),
			Type:        EventTypeAuthFailure,
			Severity:    SeverityWarning,
			Outcome:     OutcomeFailure,
			Actor:       Actor{ID: "user-2", Type: "user"},
			Source:      Source{IPAddress: "192.168.1.2"},
			Action:      "login",
			Description: "Login failed",
		},
		{
			ID:          "event-3",
			Timestamp:   now,
			Type:        EventTypeConfigChanged,
			Severity:    SeverityCritical,
			Outcome:     OutcomeSuccess,
			Actor:       Actor{ID: "user-1", Type: "admin"},
			Source:      Source{IPAddress: "192.168.1.1"},
			Action:      "update",
			Description: "Config changed",
		},
	}

	for _, e := range events {
		if err := store.Save(ctx, e); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// Test query with type filter
	results, err := store.Query(ctx, QueryFilter{
		Types: []EventType{EventTypeAuthSuccess},
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Test query with severity filter
	results, err = store.Query(ctx, QueryFilter{
		Severities: []Severity{SeverityWarning, SeverityCritical},
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Test query with actor filter
	results, err = store.Query(ctx, QueryFilter{
		ActorID: "user-1",
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results for user-1, got %d", len(results))
	}

	// Test query with limit
	results, err = store.Query(ctx, QueryFilter{
		Limit: 2,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results with limit, got %d", len(results))
	}
}

func TestDuckDBStore_Query_TextSearch(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	events := []*Event{
		{
			ID:          "event-search-1",
			Timestamp:   time.Now().UTC(),
			Type:        EventTypeAuthSuccess,
			Severity:    SeverityInfo,
			Outcome:     OutcomeSuccess,
			Actor:       Actor{ID: "user-1", Type: "user"},
			Source:      Source{IPAddress: "192.168.1.1"},
			Action:      "login",
			Description: "User successfully authenticated via SSO",
		},
		{
			ID:          "event-search-2",
			Timestamp:   time.Now().UTC(),
			Type:        EventTypeConfigChanged,
			Severity:    SeverityWarning,
			Outcome:     OutcomeSuccess,
			Actor:       Actor{ID: "admin-1", Type: "admin"},
			Source:      Source{IPAddress: "192.168.1.2"},
			Action:      "update_config",
			Description: "Security settings modified",
		},
	}

	for _, e := range events {
		if err := store.Save(ctx, e); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// Search for "SSO"
	results, err := store.Query(ctx, QueryFilter{
		SearchText: "SSO",
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'SSO', got %d", len(results))
	}

	// Search for "security" (case insensitive)
	results, err = store.Query(ctx, QueryFilter{
		SearchText: "security",
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'security', got %d", len(results))
	}
}

func TestDuckDBStore_Count(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	// Save some events
	for i := 0; i < 5; i++ {
		event := &Event{
			ID:          "count-event-" + string(rune('A'+i)),
			Timestamp:   time.Now().UTC(),
			Type:        EventTypeAuthSuccess,
			Severity:    SeverityInfo,
			Outcome:     OutcomeSuccess,
			Actor:       Actor{ID: "user-1", Type: "user"},
			Source:      Source{IPAddress: "192.168.1.1"},
			Action:      "test",
			Description: "Test event",
		}
		if err := store.Save(ctx, event); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// Count all
	count, err := store.Count(ctx, QueryFilter{})
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected count 5, got %d", count)
	}

	// Count with filter
	count, err = store.Count(ctx, QueryFilter{
		Types: []EventType{EventTypeAuthSuccess},
	})
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected count 5 with type filter, got %d", count)
	}
}

func TestDuckDBStore_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	now := time.Now().UTC()

	// Save events with different timestamps
	events := []*Event{
		{
			ID:          "old-event-1",
			Timestamp:   now.Add(-48 * time.Hour),
			Type:        EventTypeAuthSuccess,
			Severity:    SeverityInfo,
			Outcome:     OutcomeSuccess,
			Actor:       Actor{ID: "user-1", Type: "user"},
			Source:      Source{IPAddress: "192.168.1.1"},
			Action:      "test",
			Description: "Old event 1",
		},
		{
			ID:          "old-event-2",
			Timestamp:   now.Add(-36 * time.Hour),
			Type:        EventTypeAuthSuccess,
			Severity:    SeverityInfo,
			Outcome:     OutcomeSuccess,
			Actor:       Actor{ID: "user-1", Type: "user"},
			Source:      Source{IPAddress: "192.168.1.1"},
			Action:      "test",
			Description: "Old event 2",
		},
		{
			ID:          "recent-event",
			Timestamp:   now.Add(-1 * time.Hour),
			Type:        EventTypeAuthSuccess,
			Severity:    SeverityInfo,
			Outcome:     OutcomeSuccess,
			Actor:       Actor{ID: "user-1", Type: "user"},
			Source:      Source{IPAddress: "192.168.1.1"},
			Action:      "test",
			Description: "Recent event",
		},
	}

	for _, e := range events {
		if err := store.Save(ctx, e); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// Delete events older than 24 hours
	deleted, err := store.Delete(ctx, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if deleted != 2 {
		t.Errorf("Expected 2 deleted, got %d", deleted)
	}

	// Verify only recent event remains
	count, err := store.Count(ctx, QueryFilter{})
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 remaining event, got %d", count)
	}
}

func TestDuckDBStore_GetStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	// Save events with different types and severities
	events := []*Event{
		{
			ID: "stats-1", Timestamp: time.Now().UTC(),
			Type: EventTypeAuthSuccess, Severity: SeverityInfo, Outcome: OutcomeSuccess,
			Actor: Actor{ID: "u1", Type: "user"}, Source: Source{IPAddress: "1.1.1.1"},
			Action: "test", Description: "test",
		},
		{
			ID: "stats-2", Timestamp: time.Now().UTC(),
			Type: EventTypeAuthSuccess, Severity: SeverityInfo, Outcome: OutcomeSuccess,
			Actor: Actor{ID: "u2", Type: "user"}, Source: Source{IPAddress: "1.1.1.2"},
			Action: "test", Description: "test",
		},
		{
			ID: "stats-3", Timestamp: time.Now().UTC(),
			Type: EventTypeAuthFailure, Severity: SeverityWarning, Outcome: OutcomeFailure,
			Actor: Actor{ID: "u3", Type: "user"}, Source: Source{IPAddress: "1.1.1.3"},
			Action: "test", Description: "test",
		},
	}

	for _, e := range events {
		if err := store.Save(ctx, e); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	stats, err := store.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalEvents != 3 {
		t.Errorf("Expected TotalEvents 3, got %d", stats.TotalEvents)
	}

	if stats.EventsByType[string(EventTypeAuthSuccess)] != 2 {
		t.Errorf("Expected 2 auth.success events, got %d", stats.EventsByType[string(EventTypeAuthSuccess)])
	}

	if stats.EventsBySeverity[string(SeverityInfo)] != 2 {
		t.Errorf("Expected 2 info severity events, got %d", stats.EventsBySeverity[string(SeverityInfo)])
	}

	if stats.OldestEvent == nil || stats.NewestEvent == nil {
		t.Error("Expected OldestEvent and NewestEvent to be set")
	}
}

func TestDuckDBStore_Save_NilEvent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	err := store.Save(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil event, got nil")
	}
}

func TestDuckDBStore_Query_TimeRange(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	now := time.Now().UTC()

	events := []*Event{
		{
			ID:          "time-1",
			Timestamp:   now.Add(-72 * time.Hour),
			Type:        EventTypeAuthSuccess,
			Severity:    SeverityInfo,
			Outcome:     OutcomeSuccess,
			Actor:       Actor{ID: "u1", Type: "user"},
			Source:      Source{IPAddress: "1.1.1.1"},
			Action:      "test",
			Description: "72 hours ago",
		},
		{
			ID:          "time-2",
			Timestamp:   now.Add(-24 * time.Hour),
			Type:        EventTypeAuthSuccess,
			Severity:    SeverityInfo,
			Outcome:     OutcomeSuccess,
			Actor:       Actor{ID: "u1", Type: "user"},
			Source:      Source{IPAddress: "1.1.1.1"},
			Action:      "test",
			Description: "24 hours ago",
		},
		{
			ID:          "time-3",
			Timestamp:   now.Add(-1 * time.Hour),
			Type:        EventTypeAuthSuccess,
			Severity:    SeverityInfo,
			Outcome:     OutcomeSuccess,
			Actor:       Actor{ID: "u1", Type: "user"},
			Source:      Source{IPAddress: "1.1.1.1"},
			Action:      "test",
			Description: "1 hour ago",
		},
	}

	for _, e := range events {
		if err := store.Save(ctx, e); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// Query last 48 hours
	startTime := now.Add(-48 * time.Hour)
	results, err := store.Query(ctx, QueryFilter{
		StartTime: &startTime,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results for last 48 hours, got %d", len(results))
	}

	// Query between 48 and 12 hours ago
	startTime = now.Add(-48 * time.Hour)
	endTime := now.Add(-12 * time.Hour)
	results, err = store.Query(ctx, QueryFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 48-12 hours range, got %d", len(results))
	}
}
