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

func setupTestCheckpointDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	return db
}

func TestCheckpointStore_CreateTable(t *testing.T) {
	db := setupTestCheckpointDB(t)
	defer db.Close()

	store := NewCheckpointStore(db)
	ctx := context.Background()

	err := store.CreateTable(ctx)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Verify table exists
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM replay_checkpoints").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query table: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 rows, got %d", count)
	}
}

func TestCheckpointStore_SaveAndGet(t *testing.T) {
	db := setupTestCheckpointDB(t)
	defer db.Close()

	store := NewCheckpointStore(db)
	ctx := context.Background()

	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Create checkpoint
	cp := &Checkpoint{
		ConsumerName:   "test-consumer",
		StreamName:     "MEDIA_EVENTS",
		LastSequence:   12345,
		LastTimestamp:  time.Now().UTC().Truncate(time.Microsecond),
		ProcessedCount: 1000,
		ErrorCount:     5,
		Status:         "running",
		ReplayMode:     "sequence",
		StartSequence:  100,
	}

	// Save
	err := store.Save(ctx, cp)
	if err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	// Retrieve
	retrieved, err := store.Get(ctx, cp.ConsumerName, cp.StreamName)
	if err != nil {
		t.Fatalf("Failed to get checkpoint: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected checkpoint, got nil")
	}

	// Verify fields
	if retrieved.ConsumerName != cp.ConsumerName {
		t.Errorf("ConsumerName mismatch: got %s, want %s", retrieved.ConsumerName, cp.ConsumerName)
	}
	if retrieved.StreamName != cp.StreamName {
		t.Errorf("StreamName mismatch: got %s, want %s", retrieved.StreamName, cp.StreamName)
	}
	if retrieved.LastSequence != cp.LastSequence {
		t.Errorf("LastSequence mismatch: got %d, want %d", retrieved.LastSequence, cp.LastSequence)
	}
	if retrieved.ProcessedCount != cp.ProcessedCount {
		t.Errorf("ProcessedCount mismatch: got %d, want %d", retrieved.ProcessedCount, cp.ProcessedCount)
	}
	if retrieved.Status != cp.Status {
		t.Errorf("Status mismatch: got %s, want %s", retrieved.Status, cp.Status)
	}
}

func TestCheckpointStore_Update(t *testing.T) {
	db := setupTestCheckpointDB(t)
	defer db.Close()

	store := NewCheckpointStore(db)
	ctx := context.Background()

	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Create initial checkpoint
	cp := &Checkpoint{
		ConsumerName:   "test-consumer",
		StreamName:     "MEDIA_EVENTS",
		LastSequence:   100,
		ProcessedCount: 100,
		Status:         "running",
	}

	if err := store.Save(ctx, cp); err != nil {
		t.Fatalf("Failed to save initial checkpoint: %v", err)
	}

	// Update
	cp.LastSequence = 200
	cp.ProcessedCount = 200
	cp.Status = "completed"

	if err := store.Save(ctx, cp); err != nil {
		t.Fatalf("Failed to update checkpoint: %v", err)
	}

	// Verify update
	retrieved, err := store.Get(ctx, cp.ConsumerName, cp.StreamName)
	if err != nil {
		t.Fatalf("Failed to get checkpoint: %v", err)
	}
	if retrieved.LastSequence != 200 {
		t.Errorf("LastSequence not updated: got %d, want 200", retrieved.LastSequence)
	}
	if retrieved.Status != "completed" {
		t.Errorf("Status not updated: got %s, want completed", retrieved.Status)
	}
}

func TestCheckpointStore_List(t *testing.T) {
	db := setupTestCheckpointDB(t)
	defer db.Close()

	store := NewCheckpointStore(db)
	ctx := context.Background()

	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Create multiple checkpoints
	checkpoints := []*Checkpoint{
		{ConsumerName: "consumer-1", StreamName: "STREAM_A", Status: "running", LastSequence: 100},
		{ConsumerName: "consumer-2", StreamName: "STREAM_A", Status: "completed", LastSequence: 200},
		{ConsumerName: "consumer-3", StreamName: "STREAM_B", Status: "running", LastSequence: 300},
	}

	for _, cp := range checkpoints {
		if err := store.Save(ctx, cp); err != nil {
			t.Fatalf("Failed to save checkpoint: %v", err)
		}
	}

	// List all
	all, err := store.List(ctx, "")
	if err != nil {
		t.Fatalf("Failed to list all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("Expected 3 checkpoints, got %d", len(all))
	}

	// List by status
	running, err := store.List(ctx, "running")
	if err != nil {
		t.Fatalf("Failed to list running: %v", err)
	}
	if len(running) != 2 {
		t.Errorf("Expected 2 running checkpoints, got %d", len(running))
	}
}

func TestCheckpointStore_Delete(t *testing.T) {
	db := setupTestCheckpointDB(t)
	defer db.Close()

	store := NewCheckpointStore(db)
	ctx := context.Background()

	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Create checkpoint
	cp := &Checkpoint{
		ConsumerName: "test-consumer",
		StreamName:   "MEDIA_EVENTS",
		Status:       "running",
	}

	if err := store.Save(ctx, cp); err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	// Get the ID
	retrieved, err := store.Get(ctx, cp.ConsumerName, cp.StreamName)
	if err != nil {
		t.Fatalf("Failed to get checkpoint: %v", err)
	}

	// Delete
	if err := store.Delete(ctx, retrieved.ID); err != nil {
		t.Fatalf("Failed to delete checkpoint: %v", err)
	}

	// Verify deleted
	deleted, err := store.Get(ctx, cp.ConsumerName, cp.StreamName)
	if err != nil {
		t.Fatalf("Failed to get after delete: %v", err)
	}
	if deleted != nil {
		t.Error("Expected nil after delete")
	}
}

func TestCheckpointStore_GetNotFound(t *testing.T) {
	db := setupTestCheckpointDB(t)
	defer db.Close()

	store := NewCheckpointStore(db)
	ctx := context.Background()

	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Get non-existent
	cp, err := store.Get(ctx, "nonexistent", "nonexistent")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if cp != nil {
		t.Error("Expected nil for non-existent checkpoint")
	}
}

func TestCheckpointStore_GetLastForStream(t *testing.T) {
	db := setupTestCheckpointDB(t)
	defer db.Close()

	store := NewCheckpointStore(db)
	ctx := context.Background()

	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Create checkpoints for same stream
	checkpoints := []*Checkpoint{
		{ConsumerName: "consumer-1", StreamName: "MEDIA_EVENTS", LastSequence: 100, Status: "completed"},
		{ConsumerName: "consumer-2", StreamName: "MEDIA_EVENTS", LastSequence: 300, Status: "completed"},
		{ConsumerName: "consumer-3", StreamName: "MEDIA_EVENTS", LastSequence: 200, Status: "running"},
	}

	for _, cp := range checkpoints {
		if err := store.Save(ctx, cp); err != nil {
			t.Fatalf("Failed to save checkpoint: %v", err)
		}
	}

	// Get last (highest sequence)
	last, err := store.GetLastForStream(ctx, "MEDIA_EVENTS")
	if err != nil {
		t.Fatalf("Failed to get last: %v", err)
	}
	if last == nil {
		t.Fatal("Expected checkpoint, got nil")
	}
	if last.LastSequence != 300 {
		t.Errorf("Expected highest sequence 300, got %d", last.LastSequence)
	}
}

func TestReplayMode_String(t *testing.T) {
	tests := []struct {
		mode ReplayMode
		want string
	}{
		{ReplayModeNew, "new"},
		{ReplayModeAll, "all"},
		{ReplayModeSequence, "sequence"},
		{ReplayModeTime, "time"},
		{ReplayModeLastAcked, "last_acked"},
		{ReplayMode(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.mode.String()
		if got != tt.want {
			t.Errorf("ReplayMode(%d).String() = %s, want %s", tt.mode, got, tt.want)
		}
	}
}
