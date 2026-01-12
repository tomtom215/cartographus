// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupTestDuckDBDLQStore creates a DLQ store with an in-memory DuckDB for testing.
func setupTestDuckDBDLQStore(t *testing.T) *DuckDBDLQStore {
	t.Helper()

	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("Failed to open in-memory DuckDB: %v", err)
	}

	store := NewDuckDBDLQStore(db)
	ctx := context.Background()
	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("Failed to create DLQ table: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return store
}

func TestDuckDBDLQStore_SaveAndGet(t *testing.T) {
	t.Parallel()
	store := setupTestDuckDBDLQStore(t)
	ctx := context.Background()

	// Create a test event
	event := &MediaEvent{
		EventID:  "test-event-1",
		Source:   "test",
		UserID:   1,
		Username: "testuser",
		Title:    "Test Movie",
	}

	entry := &DLQEntry{
		Event:         event,
		MessageID:     "msg-123",
		OriginalError: "connection timeout",
		LastError:     "connection timeout",
		RetryCount:    0,
		FirstFailure:  time.Now().UTC().Truncate(time.Microsecond),
		LastFailure:   time.Now().UTC().Truncate(time.Microsecond),
		NextRetry:     time.Now().Add(time.Minute).UTC().Truncate(time.Microsecond),
		Category:      ErrorCategoryConnection,
	}

	// Save entry
	err := store.Save(ctx, entry)
	if err != nil {
		t.Fatalf("Failed to save entry: %v", err)
	}

	// Get entry
	retrieved, err := store.Get(ctx, "test-event-1")
	if err != nil {
		t.Fatalf("Failed to get entry: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected entry to be retrieved")
	}

	// Verify fields
	if retrieved.Event.EventID != entry.Event.EventID {
		t.Errorf("EventID mismatch: got %s, want %s", retrieved.Event.EventID, entry.Event.EventID)
	}
	if retrieved.MessageID != entry.MessageID {
		t.Errorf("MessageID mismatch: got %s, want %s", retrieved.MessageID, entry.MessageID)
	}
	if retrieved.OriginalError != entry.OriginalError {
		t.Errorf("OriginalError mismatch: got %s, want %s", retrieved.OriginalError, entry.OriginalError)
	}
	if retrieved.RetryCount != entry.RetryCount {
		t.Errorf("RetryCount mismatch: got %d, want %d", retrieved.RetryCount, entry.RetryCount)
	}
	if retrieved.Category != entry.Category {
		t.Errorf("Category mismatch: got %v, want %v", retrieved.Category, entry.Category)
	}
}

func TestDuckDBDLQStore_Update(t *testing.T) {
	t.Parallel()
	store := setupTestDuckDBDLQStore(t)
	ctx := context.Background()

	event := &MediaEvent{
		EventID: "test-event-2",
		Source:  "test",
	}

	entry := &DLQEntry{
		Event:         event,
		MessageID:     "msg-456",
		OriginalError: "timeout",
		LastError:     "timeout",
		RetryCount:    0,
		FirstFailure:  time.Now().UTC().Truncate(time.Microsecond),
		LastFailure:   time.Now().UTC().Truncate(time.Microsecond),
		NextRetry:     time.Now().Add(time.Minute).UTC().Truncate(time.Microsecond),
		Category:      ErrorCategoryTimeout,
	}

	// Save initial entry
	if err := store.Save(ctx, entry); err != nil {
		t.Fatalf("Failed to save entry: %v", err)
	}

	// Update entry
	entry.RetryCount = 3
	entry.LastError = "still timing out"
	entry.LastFailure = time.Now().UTC().Truncate(time.Microsecond)
	entry.NextRetry = time.Now().Add(5 * time.Minute).UTC().Truncate(time.Microsecond)

	if err := store.Update(ctx, entry); err != nil {
		t.Fatalf("Failed to update entry: %v", err)
	}

	// Verify update
	retrieved, err := store.Get(ctx, "test-event-2")
	if err != nil {
		t.Fatalf("Failed to get updated entry: %v", err)
	}

	if retrieved.RetryCount != 3 {
		t.Errorf("RetryCount not updated: got %d, want 3", retrieved.RetryCount)
	}
	if retrieved.LastError != "still timing out" {
		t.Errorf("LastError not updated: got %s, want 'still timing out'", retrieved.LastError)
	}
}

func TestDuckDBDLQStore_Delete(t *testing.T) {
	t.Parallel()
	store := setupTestDuckDBDLQStore(t)
	ctx := context.Background()

	event := &MediaEvent{
		EventID: "test-event-3",
		Source:  "test",
	}

	entry := &DLQEntry{
		Event:         event,
		MessageID:     "msg-789",
		OriginalError: "error",
		LastError:     "error",
		RetryCount:    0,
		FirstFailure:  time.Now().UTC(),
		LastFailure:   time.Now().UTC(),
		NextRetry:     time.Now().Add(time.Minute).UTC(),
		Category:      ErrorCategoryUnknown,
	}

	// Save entry
	if err := store.Save(ctx, entry); err != nil {
		t.Fatalf("Failed to save entry: %v", err)
	}

	// Verify it exists
	retrieved, err := store.Get(ctx, "test-event-3")
	if err != nil {
		t.Fatalf("Failed to get entry: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Entry should exist before delete")
	}

	// Delete entry
	if err := store.Delete(ctx, "test-event-3"); err != nil {
		t.Fatalf("Failed to delete entry: %v", err)
	}

	// Verify deletion
	retrieved, err = store.Get(ctx, "test-event-3")
	if err != nil {
		t.Fatalf("Get after delete returned error: %v", err)
	}
	if retrieved != nil {
		t.Error("Entry should not exist after delete")
	}
}

func TestDuckDBDLQStore_List(t *testing.T) {
	t.Parallel()
	store := setupTestDuckDBDLQStore(t)
	ctx := context.Background()

	// Create multiple entries
	for i := 0; i < 3; i++ {
		event := &MediaEvent{
			EventID: "list-event-" + string(rune('a'+i)),
			Source:  "test",
		}

		entry := &DLQEntry{
			Event:         event,
			MessageID:     "msg-list-" + string(rune('a'+i)),
			OriginalError: "error",
			LastError:     "error",
			RetryCount:    i,
			FirstFailure:  time.Now().Add(time.Duration(i) * time.Minute).UTC(),
			LastFailure:   time.Now().Add(time.Duration(i) * time.Minute).UTC(),
			NextRetry:     time.Now().Add(time.Duration(i+1) * time.Minute).UTC(),
			Category:      ErrorCategoryUnknown,
		}

		if err := store.Save(ctx, entry); err != nil {
			t.Fatalf("Failed to save entry %d: %v", i, err)
		}
	}

	// List all entries
	entries, err := store.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list entries: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}
}

func TestDuckDBDLQStore_Count(t *testing.T) {
	t.Parallel()
	store := setupTestDuckDBDLQStore(t)
	ctx := context.Background()

	// Initial count should be 0
	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}

	// Add entries
	for i := 0; i < 5; i++ {
		event := &MediaEvent{
			EventID: "count-event-" + string(rune('a'+i)),
			Source:  "test",
		}

		entry := &DLQEntry{
			Event:         event,
			MessageID:     "msg-" + string(rune('a'+i)),
			OriginalError: "error",
			LastError:     "error",
			RetryCount:    0,
			FirstFailure:  time.Now().UTC(),
			LastFailure:   time.Now().UTC(),
			NextRetry:     time.Now().Add(time.Minute).UTC(),
			Category:      ErrorCategoryUnknown,
		}

		if err := store.Save(ctx, entry); err != nil {
			t.Fatalf("Failed to save entry: %v", err)
		}
	}

	// Check count
	count, err = store.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected count 5, got %d", count)
	}
}

func TestDuckDBDLQStore_DeleteExpired(t *testing.T) {
	t.Parallel()
	store := setupTestDuckDBDLQStore(t)
	ctx := context.Background()

	now := time.Now().UTC()

	// Create entries at different times
	oldEntry := &DLQEntry{
		Event:         &MediaEvent{EventID: "old-event", Source: "test"},
		MessageID:     "msg-old",
		OriginalError: "error",
		LastError:     "error",
		RetryCount:    0,
		FirstFailure:  now.Add(-48 * time.Hour), // 2 days ago
		LastFailure:   now.Add(-48 * time.Hour),
		NextRetry:     now.Add(-47 * time.Hour),
		Category:      ErrorCategoryUnknown,
	}

	newEntry := &DLQEntry{
		Event:         &MediaEvent{EventID: "new-event", Source: "test"},
		MessageID:     "msg-new",
		OriginalError: "error",
		LastError:     "error",
		RetryCount:    0,
		FirstFailure:  now.Add(-1 * time.Hour), // 1 hour ago
		LastFailure:   now.Add(-1 * time.Hour),
		NextRetry:     now,
		Category:      ErrorCategoryUnknown,
	}

	if err := store.Save(ctx, oldEntry); err != nil {
		t.Fatalf("Failed to save old entry: %v", err)
	}
	if err := store.Save(ctx, newEntry); err != nil {
		t.Fatalf("Failed to save new entry: %v", err)
	}

	// Delete entries older than 24 hours
	cutoff := now.Add(-24 * time.Hour)
	deleted, err := store.DeleteExpired(ctx, cutoff)
	if err != nil {
		t.Fatalf("Failed to delete expired: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Expected 1 deleted, got %d", deleted)
	}

	// Verify old entry is gone
	old, _ := store.Get(ctx, "old-event")
	if old != nil {
		t.Error("Old entry should have been deleted")
	}

	// Verify new entry still exists
	retrieved, _ := store.Get(ctx, "new-event")
	if retrieved == nil {
		t.Error("New entry should still exist")
	}
}

func TestDuckDBDLQStore_SaveNilEntry(t *testing.T) {
	t.Parallel()
	store := setupTestDuckDBDLQStore(t)
	ctx := context.Background()

	err := store.Save(ctx, nil)
	if err == nil {
		t.Error("Expected error when saving nil entry")
	}
}

func TestDuckDBDLQStore_UpsertBehavior(t *testing.T) {
	t.Parallel()
	store := setupTestDuckDBDLQStore(t)
	ctx := context.Background()

	event := &MediaEvent{
		EventID: "upsert-event",
		Source:  "test",
	}

	entry := &DLQEntry{
		Event:         event,
		MessageID:     "msg-upsert",
		OriginalError: "original error",
		LastError:     "original error",
		RetryCount:    0,
		FirstFailure:  time.Now().UTC(),
		LastFailure:   time.Now().UTC(),
		NextRetry:     time.Now().Add(time.Minute).UTC(),
		Category:      ErrorCategoryUnknown,
	}

	// Save initial
	if err := store.Save(ctx, entry); err != nil {
		t.Fatalf("Failed to save initial entry: %v", err)
	}

	// Save again with updated values (upsert)
	entry.RetryCount = 5
	entry.LastError = "updated error"

	if err := store.Save(ctx, entry); err != nil {
		t.Fatalf("Failed to upsert entry: %v", err)
	}

	// Verify upsert worked
	retrieved, err := store.Get(ctx, "upsert-event")
	if err != nil {
		t.Fatalf("Failed to get entry: %v", err)
	}

	if retrieved.RetryCount != 5 {
		t.Errorf("RetryCount not updated via upsert: got %d, want 5", retrieved.RetryCount)
	}
	if retrieved.LastError != "updated error" {
		t.Errorf("LastError not updated via upsert")
	}
}
