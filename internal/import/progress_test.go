// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//nolint:revive // package name with underscore is intentional for clarity
package tautulli_import

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func TestInMemoryProgress(t *testing.T) {
	t.Run("saves and loads progress", func(t *testing.T) {
		progress := NewInMemoryProgress()
		ctx := context.Background()

		stats := &ImportStats{
			TotalRecords:    1000,
			Processed:       500,
			Imported:        480,
			Skipped:         15,
			Errors:          5,
			StartTime:       time.Now().Add(-5 * time.Minute),
			LastProcessedID: 500,
		}

		if err := progress.Save(ctx, stats); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		loaded, err := progress.Load(ctx)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if loaded.TotalRecords != stats.TotalRecords {
			t.Errorf("TotalRecords = %d, want %d", loaded.TotalRecords, stats.TotalRecords)
		}
		if loaded.Processed != stats.Processed {
			t.Errorf("Processed = %d, want %d", loaded.Processed, stats.Processed)
		}
		if loaded.LastProcessedID != stats.LastProcessedID {
			t.Errorf("LastProcessedID = %d, want %d", loaded.LastProcessedID, stats.LastProcessedID)
		}
	})

	t.Run("returns nil when no progress saved", func(t *testing.T) {
		progress := NewInMemoryProgress()
		ctx := context.Background()

		loaded, err := progress.Load(ctx)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded != nil {
			t.Errorf("Load() = %v, want nil", loaded)
		}
	})

	t.Run("clears progress", func(t *testing.T) {
		progress := NewInMemoryProgress()
		ctx := context.Background()

		stats := &ImportStats{
			TotalRecords:    1000,
			Processed:       500,
			LastProcessedID: 500,
		}

		if err := progress.Save(ctx, stats); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		if err := progress.Clear(ctx); err != nil {
			t.Fatalf("Clear() error = %v", err)
		}

		loaded, err := progress.Load(ctx)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded != nil {
			t.Errorf("Load() after Clear() = %v, want nil", loaded)
		}
	})

	t.Run("save does not modify original stats", func(t *testing.T) {
		progress := NewInMemoryProgress()
		ctx := context.Background()

		stats := &ImportStats{
			Processed: 100,
		}

		if err := progress.Save(ctx, stats); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Modify original
		stats.Processed = 200

		// Load should return original value
		loaded, _ := progress.Load(ctx)
		if loaded.Processed != 100 {
			t.Errorf("Loaded Processed = %d, want 100 (original)", loaded.Processed)
		}
	})
}

func TestImportStats(t *testing.T) {
	t.Run("Duration calculates correctly for running import", func(t *testing.T) {
		stats := &ImportStats{
			StartTime: time.Now().Add(-5 * time.Minute),
			// EndTime is zero (import still running)
		}

		duration := stats.Duration()

		// Should be approximately 5 minutes (allowing some margin)
		if duration < 4*time.Minute || duration > 6*time.Minute {
			t.Errorf("Duration() = %v, want ~5 minutes", duration)
		}
	})

	t.Run("Duration calculates correctly for completed import", func(t *testing.T) {
		start := time.Now().Add(-10 * time.Minute)
		end := start.Add(5 * time.Minute)

		stats := &ImportStats{
			StartTime: start,
			EndTime:   end,
		}

		duration := stats.Duration()

		if duration != 5*time.Minute {
			t.Errorf("Duration() = %v, want 5 minutes", duration)
		}
	})

	t.Run("Progress calculates correctly", func(t *testing.T) {
		tests := []struct {
			total    int64
			done     int64
			expected float64
		}{
			{1000, 0, 0},
			{1000, 500, 50},
			{1000, 1000, 100},
			{0, 0, 0}, // Edge case: empty database
		}

		for _, tt := range tests {
			stats := &ImportStats{
				TotalRecords: tt.total,
				Processed:    tt.done,
			}

			progress := stats.Progress()

			if progress != tt.expected {
				t.Errorf("Progress() with %d/%d = %f, want %f", tt.done, tt.total, progress, tt.expected)
			}
		}
	})

	t.Run("RecordsPerSecond calculates correctly", func(t *testing.T) {
		start := time.Now().Add(-10 * time.Second)
		end := time.Now()

		stats := &ImportStats{
			Processed: 100,
			StartTime: start,
			EndTime:   end,
		}

		rate := stats.RecordsPerSecond()

		// Should be approximately 10 records/second
		if rate < 9 || rate > 11 {
			t.Errorf("RecordsPerSecond() = %f, want ~10", rate)
		}
	})

	t.Run("RecordsPerSecond returns 0 for zero duration", func(t *testing.T) {
		now := time.Now()
		stats := &ImportStats{
			Processed: 100,
			StartTime: now,
			EndTime:   now,
		}

		rate := stats.RecordsPerSecond()

		if rate != 0 {
			t.Errorf("RecordsPerSecond() = %f, want 0 for zero duration", rate)
		}
	})
}

func TestProgressSummary(t *testing.T) {
	t.Run("ToSummary sets correct status", func(t *testing.T) {
		tests := []struct {
			name     string
			stats    *ImportStats
			running  bool
			expected string
		}{
			{
				name:     "running import",
				stats:    &ImportStats{StartTime: time.Now()},
				running:  true,
				expected: "running",
			},
			{
				name:     "completed import",
				stats:    &ImportStats{StartTime: time.Now().Add(-time.Hour), EndTime: time.Now()},
				running:  false,
				expected: "completed",
			},
			{
				name:     "pending import",
				stats:    &ImportStats{StartTime: time.Now()},
				running:  false,
				expected: "pending",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				summary := tt.stats.ToSummary(tt.running)

				if summary.Status != tt.expected {
					t.Errorf("ToSummary().Status = %s, want %s", summary.Status, tt.expected)
				}
			})
		}
	})

	t.Run("ToSummary calculates estimated remaining time", func(t *testing.T) {
		stats := &ImportStats{
			TotalRecords: 1000,
			Processed:    500,
			StartTime:    time.Now().Add(-50 * time.Second), // 10 records/second
		}

		summary := stats.ToSummary(true)

		// 500 remaining at 10/sec = 50 seconds remaining
		if summary.EstimatedRemain < 40 || summary.EstimatedRemain > 60 {
			t.Errorf("EstimatedRemain = %f, want ~50 seconds", summary.EstimatedRemain)
		}
	})
}

// createTestBadgerDB creates a temporary BadgerDB for testing.
func createTestBadgerDB(t *testing.T) (*badger.DB, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "badger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	opts := badger.DefaultOptions(filepath.Join(tmpDir, "badger"))
	opts.Logger = nil // Suppress badger logs during tests

	db, err := badger.Open(opts)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to open badger: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestBadgerProgress(t *testing.T) {
	t.Run("saves and loads progress", func(t *testing.T) {
		db, cleanup := createTestBadgerDB(t)
		defer cleanup()

		progress := NewBadgerProgress(db)
		ctx := context.Background()

		stats := &ImportStats{
			TotalRecords:    1000,
			Processed:       500,
			Imported:        480,
			Skipped:         15,
			Errors:          5,
			StartTime:       time.Now().Add(-5 * time.Minute),
			LastProcessedID: 500,
		}

		if err := progress.Save(ctx, stats); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		loaded, err := progress.Load(ctx)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if loaded.TotalRecords != stats.TotalRecords {
			t.Errorf("TotalRecords = %d, want %d", loaded.TotalRecords, stats.TotalRecords)
		}
		if loaded.Processed != stats.Processed {
			t.Errorf("Processed = %d, want %d", loaded.Processed, stats.Processed)
		}
		if loaded.Imported != stats.Imported {
			t.Errorf("Imported = %d, want %d", loaded.Imported, stats.Imported)
		}
		if loaded.LastProcessedID != stats.LastProcessedID {
			t.Errorf("LastProcessedID = %d, want %d", loaded.LastProcessedID, stats.LastProcessedID)
		}
	})

	t.Run("returns nil when no progress saved", func(t *testing.T) {
		db, cleanup := createTestBadgerDB(t)
		defer cleanup()

		progress := NewBadgerProgress(db)
		ctx := context.Background()

		loaded, err := progress.Load(ctx)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded != nil {
			t.Errorf("Load() = %v, want nil", loaded)
		}
	})

	t.Run("clears progress", func(t *testing.T) {
		db, cleanup := createTestBadgerDB(t)
		defer cleanup()

		progress := NewBadgerProgress(db)
		ctx := context.Background()

		stats := &ImportStats{
			TotalRecords:    1000,
			Processed:       500,
			LastProcessedID: 500,
			StartTime:       time.Now(),
		}

		if err := progress.Save(ctx, stats); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		if err := progress.Clear(ctx); err != nil {
			t.Fatalf("Clear() error = %v", err)
		}

		loaded, err := progress.Load(ctx)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded != nil {
			t.Errorf("Load() after Clear() = %v, want nil", loaded)
		}
	})

	t.Run("persists across sessions", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "badger-persist-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		dbPath := filepath.Join(tmpDir, "badger")

		// First session: save progress
		opts := badger.DefaultOptions(dbPath)
		opts.Logger = nil
		db1, err := badger.Open(opts)
		if err != nil {
			t.Fatalf("Failed to open badger (session 1): %v", err)
		}

		progress1 := NewBadgerProgress(db1)
		ctx := context.Background()

		stats := &ImportStats{
			TotalRecords:    1000,
			Processed:       500,
			Imported:        480,
			LastProcessedID: 500,
			StartTime:       time.Now().Add(-5 * time.Minute),
		}

		if err := progress1.Save(ctx, stats); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		db1.Close()

		// Second session: load progress
		db2, err := badger.Open(opts)
		if err != nil {
			t.Fatalf("Failed to open badger (session 2): %v", err)
		}
		defer db2.Close()

		progress2 := NewBadgerProgress(db2)

		loaded, err := progress2.Load(ctx)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded == nil {
			t.Fatal("Load() returned nil after restart")
		}

		if loaded.TotalRecords != stats.TotalRecords {
			t.Errorf("TotalRecords after restart = %d, want %d", loaded.TotalRecords, stats.TotalRecords)
		}
		if loaded.Processed != stats.Processed {
			t.Errorf("Processed after restart = %d, want %d", loaded.Processed, stats.Processed)
		}
		if loaded.LastProcessedID != stats.LastProcessedID {
			t.Errorf("LastProcessedID after restart = %d, want %d", loaded.LastProcessedID, stats.LastProcessedID)
		}
	})

	t.Run("clear on nonexistent key succeeds", func(t *testing.T) {
		db, cleanup := createTestBadgerDB(t)
		defer cleanup()

		progress := NewBadgerProgress(db)
		ctx := context.Background()

		// Clear without saving first should not error
		if err := progress.Clear(ctx); err != nil {
			t.Errorf("Clear() on empty db error = %v", err)
		}
	})

	t.Run("overwrites existing progress", func(t *testing.T) {
		db, cleanup := createTestBadgerDB(t)
		defer cleanup()

		progress := NewBadgerProgress(db)
		ctx := context.Background()

		// First save
		stats1 := &ImportStats{
			TotalRecords:    1000,
			Processed:       500,
			LastProcessedID: 500,
			StartTime:       time.Now(),
		}
		if err := progress.Save(ctx, stats1); err != nil {
			t.Fatalf("Save(1) error = %v", err)
		}

		// Second save with different values
		stats2 := &ImportStats{
			TotalRecords:    1000,
			Processed:       750,
			LastProcessedID: 750,
			StartTime:       time.Now(),
		}
		if err := progress.Save(ctx, stats2); err != nil {
			t.Fatalf("Save(2) error = %v", err)
		}

		// Load should return second save
		loaded, err := progress.Load(ctx)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded.Processed != 750 {
			t.Errorf("Processed = %d, want 750", loaded.Processed)
		}
		if loaded.LastProcessedID != 750 {
			t.Errorf("LastProcessedID = %d, want 750", loaded.LastProcessedID)
		}
	})
}
