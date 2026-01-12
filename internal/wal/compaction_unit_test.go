// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal

package wal

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestCompactor_DeleteConfirmedEntries tests that confirmed entries are deleted.
func TestCompactor_DeleteConfirmedEntries(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	ctx := context.Background()

	// Write and confirm entries
	ids := writeAndConfirmEvents(ctx, t, wal, 5)

	// Verify confirmed count
	stats := wal.Stats()
	if stats.ConfirmedCount != 5 {
		t.Errorf("Expected 5 confirmed, got %d", stats.ConfirmedCount)
	}

	// Run compaction
	compactor := NewCompactor(wal)
	if err := compactor.RunNow(); err != nil {
		t.Fatalf("RunNow failed: %v", err)
	}

	// Verify confirmed entries were deleted
	statsAfter := wal.Stats()
	if statsAfter.ConfirmedCount != 0 {
		t.Errorf("Expected 0 confirmed after compaction, got %d", statsAfter.ConfirmedCount)
	}

	// Verify entries are gone (we can check pending for IDs)
	for _, id := range ids {
		// The entries should be fully removed
		entries, _ := wal.GetPending(ctx)
		for _, entry := range entries {
			if entry.ID == id {
				t.Errorf("Entry %s should be removed after confirmation and compaction", id)
			}
		}
	}
}

// TestCompactor_DeleteExpiredEntries tests that expired pending entries are deleted.
// Note: BadgerDB native TTL causes entries to become invisible after expiration.
// This test uses a longer TTL to verify that entries can be written and then
// naturally expire via BadgerDB's TTL mechanism.
func TestCompactor_DeleteExpiredEntries(t *testing.T) {
	cfg := createFastTestConfig(t)
	// Use 2s TTL - long enough to verify writes even in slow CI environments
	cfg.EntryTTL = 2 * time.Second

	wal, err := OpenForTesting(&cfg)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer wal.Close()

	ctx := context.Background()

	// Write entries
	writeTestEvents(ctx, t, wal, 3)

	// Verify entries are written (before they expire)
	entriesBefore, err := wal.GetPending(ctx)
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}
	if len(entriesBefore) != 3 {
		t.Fatalf("Expected 3 pending entries after write, got %d", len(entriesBefore))
	}

	// Wait for entries to expire via BadgerDB native TTL
	time.Sleep(2500 * time.Millisecond)

	// After native TTL expires, entries should be invisible to GetPending
	entriesExpired, err := wal.GetPending(ctx)
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}
	// BadgerDB native TTL makes expired entries invisible
	if len(entriesExpired) != 0 {
		t.Errorf("Expected 0 entries after TTL expiration, got %d", len(entriesExpired))
	}

	// Run compaction to trigger GC cleanup of expired entries
	compactor := NewCompactor(wal)
	if err := compactor.RunNow(); err != nil {
		t.Fatalf("RunNow failed: %v", err)
	}

	// Verify no entries remain
	entriesAfter, err := wal.GetPending(ctx)
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}
	if len(entriesAfter) != 0 {
		t.Errorf("Expected 0 pending after compaction, got %d", len(entriesAfter))
	}
}

// TestCompactor_ConcurrentCompaction tests concurrent compaction calls.
func TestCompactor_ConcurrentCompaction(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	ctx := context.Background()

	// Write and confirm entries
	writeAndConfirmEvents(ctx, t, wal, 10)

	compactor := NewCompactor(wal)

	const goroutines = 5
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				if err := compactor.RunNow(); err != nil {
					t.Errorf("RunNow failed: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	// Should not panic or deadlock, all entries should be deleted
	stats := wal.Stats()
	if stats.ConfirmedCount != 0 {
		t.Errorf("Expected 0 confirmed after concurrent compaction, got %d", stats.ConfirmedCount)
	}
}

// TestCompactor_DuringHeavyWrites tests compaction during heavy write load.
func TestCompactor_DuringHeavyWrites(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	compactor := NewCompactor(wal)
	if err := compactor.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer compactor.Stop()

	const writers = 5
	const writesPerWriter = 50

	var wg sync.WaitGroup
	errCh := make(chan error, writers*writesPerWriter)

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < writesPerWriter; j++ {
				event := createTestEvent("heavy-" + string(rune('a'+writerID)) + "-" + string(rune('0'+j%10)))
				entryID, err := wal.Write(ctx, event)
				if err != nil {
					errCh <- err
					continue
				}

				// Confirm half of them to trigger compaction
				if j%2 == 0 {
					if err := wal.Confirm(ctx, entryID); err != nil {
						errCh <- err
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	// Check for errors
	for err := range errCh {
		t.Errorf("Error during heavy writes: %v", err)
	}

	// Give compactor time to run
	time.Sleep(200 * time.Millisecond)

	// Should complete without panics or data corruption
}

// TestCompactor_StatsUpdate tests that compactor stats are updated correctly.
func TestCompactor_StatsUpdate(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	ctx := context.Background()

	compactor := NewCompactor(wal)

	// Initial stats should be zero
	stats := compactor.GetStats()
	if !stats.LastRun.IsZero() {
		t.Error("LastRun should be zero before any compaction")
	}
	if stats.LastEntriesCount != 0 {
		t.Error("LastEntriesCount should be 0 before any compaction")
	}

	// Write and confirm entries
	writeAndConfirmEvents(ctx, t, wal, 3)

	// Run compaction
	if err := compactor.RunNow(); err != nil {
		t.Fatalf("RunNow failed: %v", err)
	}

	// Stats should be updated
	statsAfter := compactor.GetStats()
	if statsAfter.LastRun.IsZero() {
		t.Error("LastRun should not be zero after compaction")
	}
	if statsAfter.LastEntriesCount != 3 {
		t.Errorf("Expected 3 entries compacted, got %d", statsAfter.LastEntriesCount)
	}
}

// TestCompactor_StartStopCycles tests multiple start/stop cycles.
func TestCompactor_StartStopCycles(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	compactor := NewCompactor(wal)

	for i := 0; i < 5; i++ {
		if err := compactor.Start(ctx); err != nil {
			t.Fatalf("Start cycle %d failed: %v", i, err)
		}
		if !compactor.IsRunning() {
			t.Errorf("Compactor should be running after start cycle %d", i)
		}

		time.Sleep(10 * time.Millisecond)

		compactor.Stop()
		if compactor.IsRunning() {
			t.Errorf("Compactor should not be running after stop cycle %d", i)
		}
	}
}

// TestCompactor_GracefulStop tests that Stop waits for running compaction.
func TestCompactor_GracefulStop(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	ctx := context.Background()

	// Write entries
	writeAndConfirmEvents(ctx, t, wal, 10)

	compactor := NewCompactor(wal)

	// Start and immediately stop
	if err := compactor.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Stop should complete cleanly
	compactor.Stop()

	if compactor.IsRunning() {
		t.Error("Compactor should not be running after Stop")
	}
}

// TestCompactor_ContextCancellation tests compaction stopping on context cancellation.
func TestCompactor_ContextCancellation(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	ctx, cancel := context.WithCancel(context.Background())

	compactor := NewCompactor(wal)
	if err := compactor.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Cancel context
	cancel()

	// Wait for compactor to stop
	time.Sleep(100 * time.Millisecond)

	// Verify Stop is idempotent after context cancellation
	compactor.Stop()
}

// TestCompactor_EmptyWAL tests compaction with no entries.
func TestCompactor_EmptyWAL(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	compactor := NewCompactor(wal)

	// Should not error on empty WAL
	if err := compactor.RunNow(); err != nil {
		t.Fatalf("RunNow on empty WAL failed: %v", err)
	}

	stats := compactor.GetStats()
	if stats.LastEntriesCount != 0 {
		t.Errorf("Expected 0 entries compacted on empty WAL, got %d", stats.LastEntriesCount)
	}
}

// TestCompactor_MixedEntries tests compaction with mix of pending and confirmed entries.
func TestCompactor_MixedEntries(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	ctx := context.Background()

	// Write pending entries
	pendingIDs := writeTestEvents(ctx, t, wal, 3)

	// Write and confirm entries
	writeAndConfirmEvents(ctx, t, wal, 2)

	// Verify counts
	stats := wal.Stats()
	if stats.PendingCount != 3 {
		t.Errorf("Expected 3 pending, got %d", stats.PendingCount)
	}
	if stats.ConfirmedCount != 2 {
		t.Errorf("Expected 2 confirmed, got %d", stats.ConfirmedCount)
	}

	// Run compaction
	compactor := NewCompactor(wal)
	if err := compactor.RunNow(); err != nil {
		t.Fatalf("RunNow failed: %v", err)
	}

	// Only confirmed entries should be removed
	statsAfter := wal.Stats()
	if statsAfter.PendingCount != 3 {
		t.Errorf("Expected 3 pending after compaction, got %d", statsAfter.PendingCount)
	}
	if statsAfter.ConfirmedCount != 0 {
		t.Errorf("Expected 0 confirmed after compaction, got %d", statsAfter.ConfirmedCount)
	}

	// Verify pending entries still exist
	entries, err := wal.GetPending(ctx)
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("Expected 3 pending entries, got %d", len(entries))
	}

	// Verify correct pending IDs
	for _, id := range pendingIDs {
		found := false
		for _, entry := range entries {
			if entry.ID == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Pending entry %s should still exist", id)
		}
	}
}

// TestCompactor_GCRuns tests that GC is run during compaction.
func TestCompactor_GCRuns(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	ctx := context.Background()

	// Write and confirm many entries to trigger GC
	writeAndConfirmEvents(ctx, t, wal, 100)

	compactor := NewCompactor(wal)
	if err := compactor.RunNow(); err != nil {
		t.Fatalf("RunNow failed: %v", err)
	}

	// GC should complete without error
	// (GC failure is logged but doesn't stop compaction)
}

// TestCompactor_StatsConcurrency tests concurrent access to GetStats.
func TestCompactor_StatsConcurrency(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	ctx := context.Background()

	compactor := NewCompactor(wal)
	if err := compactor.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer compactor.Stop()

	var wg sync.WaitGroup
	const goroutines = 10

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = compactor.GetStats()
				_ = compactor.IsRunning()
			}
		}()
	}

	wg.Wait()
	// Should not race or panic
}

// TestCompactorStats_Fields tests CompactorStats struct.
func TestCompactorStats_Fields(t *testing.T) {
	stats := CompactorStats{
		LastRun:          time.Now(),
		LastEntriesCount: 42,
	}

	if stats.LastRun.IsZero() {
		t.Error("LastRun should not be zero")
	}
	if stats.LastEntriesCount != 42 {
		t.Errorf("Expected 42 entries, got %d", stats.LastEntriesCount)
	}
}

// TestCompactor_RunDuringStart tests that RunNow works while compactor is running.
func TestCompactor_RunDuringStart(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	ctx := context.Background()

	compactor := NewCompactor(wal)
	if err := compactor.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer compactor.Stop()

	// Write and confirm entries
	writeAndConfirmEvents(ctx, t, wal, 5)

	// RunNow should work even with background loop running
	if err := compactor.RunNow(); err != nil {
		t.Fatalf("RunNow while running failed: %v", err)
	}

	// Entries should be compacted
	stats := wal.Stats()
	if stats.ConfirmedCount != 0 {
		t.Errorf("Expected 0 confirmed after RunNow, got %d", stats.ConfirmedCount)
	}
}
