// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal

package wal

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testEvent is a simple struct for testing WAL operations.
// It avoids importing eventprocessor to prevent circular imports.
type testEvent struct {
	EventID   string    `json:"event_id"`
	Source    string    `json:"source"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	MediaType string    `json:"media_type"`
	Title     string    `json:"title"`
	StartedAt time.Time `json:"started_at"`
}

// Test helpers

func createTestConfig(t *testing.T) Config {
	t.Helper()
	tmpDir := t.TempDir()
	return Config{
		Enabled:          true,
		Path:             filepath.Join(tmpDir, "wal"),
		SyncWrites:       false, // Faster tests without fsync
		RetryInterval:    1 * time.Second,
		MaxRetries:       3,
		RetryBackoff:     1 * time.Second,
		CompactInterval:  1 * time.Minute,
		EntryTTL:         1 * time.Hour,
		MemTableSize:     16 * 1024 * 1024, // 16MB for tests (BadgerDB minimum)
		ValueLogFileSize: 16 * 1024 * 1024, // 16MB for tests
		NumCompactors:    2,                // BadgerDB minimum
		LeaseDuration:    30 * time.Second, // Minimum for durable leasing
	}
}

// createFastTestConfig creates a config with fast intervals for testing.
// This config is NOT valid for Open() but works with OpenForTesting().
func createFastTestConfig(t *testing.T) Config {
	t.Helper()
	tmpDir := t.TempDir()
	return Config{
		Enabled:          true,
		Path:             filepath.Join(tmpDir, "wal"),
		SyncWrites:       false,
		RetryInterval:    50 * time.Millisecond,
		MaxRetries:       3,
		RetryBackoff:     1 * time.Millisecond,
		CompactInterval:  50 * time.Millisecond,
		EntryTTL:         1 * time.Hour,
		MemTableSize:     16 * 1024 * 1024, // 16MB for tests (BadgerDB minimum)
		ValueLogFileSize: 16 * 1024 * 1024, // 16MB for tests
		NumCompactors:    2,
		LeaseDuration:    30 * time.Second, // Minimum for durable leasing
	}
}

func createTestEvent(id string) *testEvent {
	return &testEvent{
		EventID:   id,
		Source:    "test",
		UserID:    123,
		Username:  "testuser",
		MediaType: "movie",
		Title:     "Test Movie " + id,
		StartedAt: time.Now(),
	}
}

// setupWAL creates a WAL with standard test config and returns the concrete type.
// The caller should defer wal.Close().
func setupWAL(t *testing.T) *BadgerWAL {
	t.Helper()
	cfg := createTestConfig(t)
	wal, err := Open(&cfg)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	return wal
}

// setupFastWAL creates a WAL with fast test config for timing-sensitive tests.
// The caller should defer wal.Close().
func setupFastWAL(t *testing.T) *BadgerWAL {
	t.Helper()
	cfg := createFastTestConfig(t)
	wal, err := OpenForTesting(&cfg)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	return wal
}

// writeTestEvents writes n events to the WAL and returns their IDs.
func writeTestEvents(ctx context.Context, t *testing.T, wal WAL, n int) []string {
	t.Helper()
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		event := createTestEvent("test-" + string(rune('1'+i)))
		id, err := wal.Write(ctx, event)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		ids[i] = id
	}
	return ids
}

// writeAndConfirmEvents writes n events, confirms them, and returns their IDs.
func writeAndConfirmEvents(ctx context.Context, t *testing.T, wal WAL, n int) []string {
	t.Helper()
	ids := writeTestEvents(ctx, t, wal, n)
	for _, id := range ids {
		if err := wal.Confirm(ctx, id); err != nil {
			t.Fatalf("Confirm failed: %v", err)
		}
	}
	return ids
}

// assertPendingCount checks that GetPending returns the expected count.
func assertPendingCount(ctx context.Context, t *testing.T, wal WAL, expected int) {
	t.Helper()
	entries, err := wal.GetPending(ctx)
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}
	if len(entries) != expected {
		t.Errorf("Expected %d pending entries, got %d", expected, len(entries))
	}
}

// mockPublisher implements Publisher for testing
type mockPublisher struct {
	publishCount atomic.Int32
	failCount    atomic.Int32
	failUntil    atomic.Int32
	mu           sync.Mutex
	published    []*testEvent
}

func (m *mockPublisher) PublishEntry(ctx context.Context, entry *Entry) error {
	m.publishCount.Add(1)
	if m.failUntil.Load() > 0 {
		m.failUntil.Add(-1)
		m.failCount.Add(1)
		return context.DeadlineExceeded
	}
	// Unmarshal the payload to get the event
	var event testEvent
	if err := entry.UnmarshalPayload(&event); err != nil {
		return err
	}
	m.mu.Lock()
	m.published = append(m.published, &event)
	m.mu.Unlock()
	return nil
}

func (m *mockPublisher) setFailures(n int) {
	m.failUntil.Store(int32(n))
}

// TestWAL_Write tests basic write operations
func TestWAL_Write(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	ctx := context.Background()
	event := createTestEvent("test-1")

	entryID, err := wal.Write(ctx, event)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if entryID == "" {
		t.Error("Expected non-empty entry ID")
	}

	entries, err := wal.GetPending(ctx)
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("Expected 1 pending entry, got %d", len(entries))
	}
	if entries[0].ID != entryID {
		t.Errorf("Entry ID mismatch: got %s, want %s", entries[0].ID, entryID)
	}
}

// TestWAL_Write_NilEvent tests that nil events are rejected
func TestWAL_Write_NilEvent(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	_, err := wal.Write(context.Background(), nil)
	if !errors.Is(err, ErrNilEvent) {
		t.Errorf("Expected ErrNilEvent, got %v", err)
	}
}

// TestWAL_Write_Concurrent tests concurrent write operations
func TestWAL_Write_Concurrent(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	ctx := context.Background()
	const numWriters = 10
	const writesPerWorker = 100

	var wg sync.WaitGroup
	errChan := make(chan error, numWriters*writesPerWorker)

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < writesPerWorker; j++ {
				event := createTestEvent("event-" + string(rune('a'+workerID)) + "-" + string(rune('0'+j%10)))
				if _, err := wal.Write(ctx, event); err != nil {
					errChan <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("Write error: %v", err)
	}

	assertPendingCount(ctx, t, wal, numWriters*writesPerWorker)
}

// TestWAL_Confirm tests confirming an entry
func TestWAL_Confirm(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	ctx := context.Background()
	ids := writeTestEvents(ctx, t, wal, 1)

	if err := wal.Confirm(ctx, ids[0]); err != nil {
		t.Fatalf("Confirm failed: %v", err)
	}

	assertPendingCount(ctx, t, wal, 0)
}

// TestWAL_Confirm_Errors tests error cases for Confirm
func TestWAL_Confirm_Errors(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	tests := []struct {
		name    string
		entryID string
		wantErr error
	}{
		{"non-existent ID", "non-existent-id", ErrEntryNotFound},
		{"empty ID", "", ErrEmptyEntryID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wal.Confirm(context.Background(), tt.entryID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Confirm(%q) error = %v, want %v", tt.entryID, err, tt.wantErr)
			}
		})
	}
}

// TestWAL_GetPending tests retrieving pending entries
func TestWAL_GetPending(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	ctx := context.Background()
	writeTestEvents(ctx, t, wal, 5)
	assertPendingCount(ctx, t, wal, 5)
}

// TestWAL_GetPending_Empty tests GetPending with no entries
func TestWAL_GetPending_Empty(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	assertPendingCount(context.Background(), t, wal, 0)
}

// TestWAL_UpdateAttempt tests updating attempt count
func TestWAL_UpdateAttempt(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	ctx := context.Background()
	ids := writeTestEvents(ctx, t, wal, 1)

	if err := wal.UpdateAttempt(ctx, ids[0], "test error"); err != nil {
		t.Fatalf("UpdateAttempt failed: %v", err)
	}

	entries, err := wal.GetPending(ctx)
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
	if entries[0].Attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", entries[0].Attempts)
	}
	if entries[0].LastError != "test error" {
		t.Errorf("Expected 'test error', got '%s'", entries[0].LastError)
	}
}

// TestWAL_DeleteEntry tests deleting an entry
func TestWAL_DeleteEntry(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	ctx := context.Background()
	ids := writeTestEvents(ctx, t, wal, 1)

	if err := wal.DeleteEntry(ctx, ids[0]); err != nil {
		t.Fatalf("DeleteEntry failed: %v", err)
	}

	assertPendingCount(ctx, t, wal, 0)
}

// TestWAL_Stats tests statistics retrieval
func TestWAL_Stats(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	ctx := context.Background()
	writeTestEvents(ctx, t, wal, 3)

	stats := wal.Stats()
	if stats.PendingCount != 3 {
		t.Errorf("Expected 3 pending, got %d", stats.PendingCount)
	}
	if stats.TotalWrites != 3 {
		t.Errorf("Expected 3 total writes, got %d", stats.TotalWrites)
	}
}

// TestWAL_Close tests graceful shutdown
func TestWAL_Close(t *testing.T) {
	wal := setupWAL(t)

	ctx := context.Background()
	event := createTestEvent("test-1")
	if _, err := wal.Write(ctx, event); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if err := wal.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Operations should fail after close
	_, err := wal.Write(ctx, event)
	if !errors.Is(err, ErrWALClosed) {
		t.Errorf("Expected ErrWALClosed, got %v", err)
	}
}

// TestWAL_Recovery tests recovery of pending entries
func TestWAL_Recovery(t *testing.T) {
	cfg := createTestConfig(t)
	walPath := cfg.Path

	// Create WAL and write events
	wal1, err := Open(&cfg)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		event := createTestEvent("test-" + string(rune('1'+i)))
		if _, err := wal1.Write(ctx, event); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	// Close WAL (simulating crash)
	if err := wal1.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Reopen WAL
	cfg.Path = walPath
	wal2, err := Open(&cfg)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer wal2.Close()

	// Recover
	pub := &mockPublisher{}
	result, err := wal2.RecoverPending(ctx, pub)
	if err != nil {
		t.Fatalf("RecoverPending failed: %v", err)
	}

	if result.TotalPending != 5 {
		t.Errorf("Expected 5 pending, got %d", result.TotalPending)
	}

	if result.Recovered != 5 {
		t.Errorf("Expected 5 recovered, got %d", result.Recovered)
	}

	// Verify entries are confirmed
	entries, err := wal2.GetPending(ctx)
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 pending after recovery, got %d", len(entries))
	}
}

// TestWAL_Recovery_WithFailures tests recovery with publish failures
func TestWAL_Recovery_WithFailures(t *testing.T) {
	cfg := createTestConfig(t)
	cfg.MaxRetries = 5 // Allow more retries

	wal, err := Open(&cfg)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer wal.Close()

	ctx := context.Background()

	// Write events
	for i := 0; i < 5; i++ {
		event := createTestEvent("test-" + string(rune('1'+i)))
		if _, err := wal.Write(ctx, event); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	// Create publisher that fails first 2 attempts
	pub := &mockPublisher{}
	pub.setFailures(2)

	result, err := wal.RecoverPending(ctx, pub)
	if err != nil {
		t.Fatalf("RecoverPending failed: %v", err)
	}

	if result.Recovered != 3 {
		t.Errorf("Expected 3 recovered, got %d", result.Recovered)
	}

	if result.Failed != 2 {
		t.Errorf("Expected 2 failed, got %d", result.Failed)
	}
}

// TestWAL_RetryLoop tests the background retry loop
func TestWAL_RetryLoop(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	writeTestEvents(ctx, t, wal, 3)

	pub := &mockPublisher{}
	retryLoop := NewRetryLoop(wal, pub)
	if err := retryLoop.Start(ctx); err != nil {
		t.Fatalf("Failed to start retry loop: %v", err)
	}
	defer retryLoop.Stop()

	time.Sleep(200 * time.Millisecond)

	if pub.publishCount.Load() != 3 {
		t.Errorf("Expected 3 publishes, got %d", pub.publishCount.Load())
	}
}

// TestWAL_RetryLoop_MaxRetries tests retry limit enforcement
func TestWAL_RetryLoop_MaxRetries(t *testing.T) {
	cfg := createFastTestConfig(t)
	cfg.MaxRetries = 2

	wal, err := OpenForTesting(&cfg)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer wal.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Write an event
	event := createTestEvent("test-1")
	entryID, err := wal.Write(ctx, event)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Manually set attempt count to max
	if err := wal.UpdateAttempt(ctx, entryID, "error 1"); err != nil {
		t.Fatalf("UpdateAttempt failed: %v", err)
	}
	if err := wal.UpdateAttempt(ctx, entryID, "error 2"); err != nil {
		t.Fatalf("UpdateAttempt failed: %v", err)
	}

	// Create always-failing publisher
	pub := &mockPublisher{}
	pub.setFailures(1000)

	// Start retry loop
	retryLoop := NewRetryLoop(wal, pub)
	if err := retryLoop.Start(ctx); err != nil {
		t.Fatalf("Failed to start retry loop: %v", err)
	}
	defer retryLoop.Stop()

	// Wait for retry loop
	time.Sleep(200 * time.Millisecond)

	// Entry should be deleted (max retries exceeded)
	entries, err := wal.GetPending(ctx)
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected entry to be deleted after max retries, got %d pending", len(entries))
	}
}

// TestWAL_Compaction tests compaction of confirmed entries
func TestWAL_Compaction(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	writeAndConfirmEvents(ctx, t, wal, 5)

	statsBefore := wal.Stats()
	if statsBefore.ConfirmedCount != 5 {
		t.Errorf("Expected 5 confirmed before compaction, got %d", statsBefore.ConfirmedCount)
	}

	compactor := NewCompactor(wal)
	if err := compactor.Start(ctx); err != nil {
		t.Fatalf("Failed to start compactor: %v", err)
	}
	defer compactor.Stop()

	time.Sleep(200 * time.Millisecond)

	statsAfter := wal.Stats()
	if statsAfter.ConfirmedCount != 0 {
		t.Errorf("Expected 0 confirmed after compaction, got %d", statsAfter.ConfirmedCount)
	}
}

// TestConfig_Validate tests configuration validation
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "disabled config",
			config:  Config{Enabled: false},
			wantErr: false,
		},
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "empty path",
			config: func() Config {
				c := DefaultConfig()
				c.Path = ""
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid retry interval",
			config: func() Config {
				c := DefaultConfig()
				c.RetryInterval = 100 * time.Millisecond
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid max retries",
			config: func() Config {
				c := DefaultConfig()
				c.MaxRetries = 0
				return c
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestConfig_LoadConfig tests loading from environment
func TestConfig_LoadConfig(t *testing.T) {
	// Set test environment variables
	os.Setenv("WAL_ENABLED", "true")
	os.Setenv("WAL_PATH", "/test/wal")
	os.Setenv("WAL_SYNC_WRITES", "false")
	os.Setenv("WAL_RETRY_INTERVAL", "60s")
	os.Setenv("WAL_MAX_RETRIES", "50")
	defer func() {
		os.Unsetenv("WAL_ENABLED")
		os.Unsetenv("WAL_PATH")
		os.Unsetenv("WAL_SYNC_WRITES")
		os.Unsetenv("WAL_RETRY_INTERVAL")
		os.Unsetenv("WAL_MAX_RETRIES")
	}()

	cfg := LoadConfig()

	if !cfg.Enabled {
		t.Error("Expected Enabled=true")
	}
	if cfg.Path != "/test/wal" {
		t.Errorf("Expected Path=/test/wal, got %s", cfg.Path)
	}
	if cfg.SyncWrites {
		t.Error("Expected SyncWrites=false")
	}
	if cfg.RetryInterval != 60*time.Second {
		t.Errorf("Expected RetryInterval=60s, got %v", cfg.RetryInterval)
	}
	if cfg.MaxRetries != 50 {
		t.Errorf("Expected MaxRetries=50, got %d", cfg.MaxRetries)
	}
}

// Durable Leasing Tests

// TestWAL_TryClaimEntryDurable tests durable leasing for concurrent processing prevention.
func TestWAL_TryClaimEntryDurable(t *testing.T) {
	t.Parallel()
	w := setupWAL(t)
	defer w.Close()
	ctx := context.Background()

	// Write an entry
	event := createTestEvent("lease-test-1")
	id, err := w.Write(ctx, event)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// First claim should succeed
	claimed, err := w.TryClaimEntryDurable(ctx, id, "processor-1")
	if err != nil {
		t.Fatalf("TryClaimEntryDurable failed: %v", err)
	}
	if !claimed {
		t.Error("First claim should succeed")
	}

	// Second claim by different processor should fail (lease not expired)
	claimed, err = w.TryClaimEntryDurable(ctx, id, "processor-2")
	if err != nil {
		t.Fatalf("TryClaimEntryDurable failed: %v", err)
	}
	if claimed {
		t.Error("Second claim should fail when lease is active")
	}

	// Same processor can reclaim
	claimed, err = w.TryClaimEntryDurable(ctx, id, "processor-1")
	if err != nil {
		t.Fatalf("TryClaimEntryDurable failed: %v", err)
	}
	if !claimed {
		t.Error("Same processor should be able to reclaim")
	}
}

// TestWAL_TryClaimEntryDurable_ExpiredLease tests that expired leases can be reclaimed.
func TestWAL_TryClaimEntryDurable_ExpiredLease(t *testing.T) {
	t.Parallel()

	// Create config with short lease duration for testing
	tmpDir := t.TempDir()
	cfg := Config{
		Enabled:          true,
		Path:             tmpDir,
		SyncWrites:       false,
		RetryInterval:    1 * time.Second,
		MaxRetries:       3,
		RetryBackoff:     1 * time.Second,
		CompactInterval:  1 * time.Minute,
		EntryTTL:         1 * time.Hour,
		MemTableSize:     16 * 1024 * 1024,
		ValueLogFileSize: 16 * 1024 * 1024,
		NumCompactors:    2,
		LeaseDuration:    100 * time.Millisecond, // Very short for testing
	}
	w, err := OpenForTesting(&cfg)
	if err != nil {
		t.Fatalf("OpenForTesting failed: %v", err)
	}
	defer w.Close()
	ctx := context.Background()

	// Write an entry
	event := createTestEvent("lease-expire-test")
	id, err := w.Write(ctx, event)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// First claim
	claimed, err := w.TryClaimEntryDurable(ctx, id, "processor-1")
	if err != nil || !claimed {
		t.Fatalf("First claim should succeed: claimed=%v, err=%v", claimed, err)
	}

	// Wait for lease to expire
	time.Sleep(150 * time.Millisecond)

	// Second processor can now claim
	claimed, err = w.TryClaimEntryDurable(ctx, id, "processor-2")
	if err != nil {
		t.Fatalf("TryClaimEntryDurable failed: %v", err)
	}
	if !claimed {
		t.Error("Should be able to claim after lease expiry")
	}
}

// TestWAL_ReleaseLeaseDurable tests explicit lease release.
func TestWAL_ReleaseLeaseDurable(t *testing.T) {
	t.Parallel()
	w := setupWAL(t)
	defer w.Close()
	ctx := context.Background()

	// Write an entry
	event := createTestEvent("release-test")
	id, err := w.Write(ctx, event)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Claim the entry
	claimed, err := w.TryClaimEntryDurable(ctx, id, "processor-1")
	if err != nil || !claimed {
		t.Fatalf("Initial claim failed: claimed=%v, err=%v", claimed, err)
	}

	// Release the lease
	err = w.ReleaseLeaseDurable(ctx, id)
	if err != nil {
		t.Fatalf("ReleaseLeaseDurable failed: %v", err)
	}

	// Another processor can now claim immediately
	claimed, err = w.TryClaimEntryDurable(ctx, id, "processor-2")
	if err != nil {
		t.Fatalf("TryClaimEntryDurable failed: %v", err)
	}
	if !claimed {
		t.Error("Should be able to claim after explicit release")
	}
}

// TestWAL_TryClaimEntryDurable_NonExistent tests claiming non-existent entry.
func TestWAL_TryClaimEntryDurable_NonExistent(t *testing.T) {
	t.Parallel()
	w := setupWAL(t)
	defer w.Close()
	ctx := context.Background()

	claimed, err := w.TryClaimEntryDurable(ctx, "non-existent-id", "processor-1")
	if err == nil {
		t.Error("Expected error for non-existent entry")
	}
	if claimed {
		t.Error("Should not claim non-existent entry")
	}
}

// Benchmark tests

// setupBenchWAL creates a WAL for benchmark tests.
func setupBenchWAL(b *testing.B) *BadgerWAL {
	b.Helper()
	cfg := Config{
		Enabled:          true,
		Path:             filepath.Join(b.TempDir(), "wal"),
		SyncWrites:       false,
		RetryInterval:    30 * time.Second,
		MaxRetries:       100,
		RetryBackoff:     5 * time.Second,
		CompactInterval:  1 * time.Hour,
		EntryTTL:         1 * time.Hour,
		MemTableSize:     16 * 1024 * 1024,
		ValueLogFileSize: 64 * 1024 * 1024,
		NumCompactors:    2,
		LeaseDuration:    2 * time.Minute, // Production-like for benchmarks
	}
	wal, err := Open(&cfg)
	if err != nil {
		b.Fatalf("Failed to open WAL: %v", err)
	}
	return wal
}

func BenchmarkWAL_Write(b *testing.B) {
	wal := setupBenchWAL(b)
	defer wal.Close()

	ctx := context.Background()
	event := createTestEvent("benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := wal.Write(ctx, event); err != nil {
			b.Fatalf("Write failed: %v", err)
		}
	}
}

func BenchmarkWAL_Write_Concurrent(b *testing.B) {
	wal := setupBenchWAL(b)
	defer wal.Close()

	ctx := context.Background()
	event := createTestEvent("benchmark")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := wal.Write(ctx, event); err != nil {
				b.Fatalf("Write failed: %v", err)
			}
		}
	})
}

func BenchmarkWAL_Confirm(b *testing.B) {
	wal := setupBenchWAL(b)
	defer wal.Close()

	ctx := context.Background()
	event := createTestEvent("benchmark")

	entryIDs := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		id, err := wal.Write(ctx, event)
		if err != nil {
			b.Fatalf("Write failed: %v", err)
		}
		entryIDs[i] = id
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := wal.Confirm(ctx, entryIDs[i]); err != nil {
			b.Fatalf("Confirm failed: %v", err)
		}
	}
}

// TestWAL_GetPendingStream tests the streaming GetPending method
func TestWAL_GetPendingStream(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	ctx := context.Background()
	expectedCount := 10
	writeTestEvents(ctx, t, wal, expectedCount)

	entryCh, errCh := wal.GetPendingStream(ctx)

	var count int
	for entry := range entryCh {
		if entry == nil {
			t.Error("Received nil entry")
			continue
		}
		count++
	}

	if err := <-errCh; err != nil {
		t.Errorf("Stream error: %v", err)
	}
	if count != expectedCount {
		t.Errorf("Expected %d entries from stream, got %d", expectedCount, count)
	}
}

// TestWAL_GetPendingStream_Cancellation tests context cancellation
func TestWAL_GetPendingStream_Cancellation(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	writeTestEvents(context.Background(), t, wal, 100)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	entryCh, errCh := wal.GetPendingStream(ctx)

	count := 0
	for entry := range entryCh {
		if entry != nil {
			count++
		}
		if count >= 5 {
			cancel()
			break
		}
	}

	for range entryCh {
	}

	err := <-errCh
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Logf("Stream error (expected context.Canceled): %v", err)
	}

	if count < 5 {
		t.Errorf("Expected at least 5 entries before cancel, got %d", count)
	}
}

// TestWAL_RecoverPendingStream tests streaming recovery
func TestWAL_RecoverPendingStream(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	ctx := context.Background()
	writeTestEvents(ctx, t, wal, 5)

	pub := &mockPublisher{}
	result, err := wal.RecoverPendingStream(ctx, pub)
	if err != nil {
		t.Fatalf("RecoverPendingStream failed: %v", err)
	}

	if result.TotalPending != 5 {
		t.Errorf("Expected 5 total pending, got %d", result.TotalPending)
	}
	if result.Recovered != 5 {
		t.Errorf("Expected 5 recovered, got %d", result.Recovered)
	}

	assertPendingCount(ctx, t, wal, 0)
}

// TestWAL_CloseTimeout tests that close respects timeout
func TestWAL_CloseTimeout(t *testing.T) {
	cfg := createFastTestConfig(t)
	cfg.CloseTimeout = 5 * time.Second // Normal timeout

	wal, err := OpenForTesting(&cfg)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}

	// Normal close should succeed
	if err := wal.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Double close should be idempotent
	if err := wal.Close(); err != nil {
		t.Errorf("Second close should not fail: %v", err)
	}
}

// TestConfig_NewFields tests the new configuration fields
func TestConfig_NewFields(t *testing.T) {
	cfg := DefaultConfig()

	// Verify new defaults
	if !cfg.Compression {
		t.Error("Expected Compression=true by default")
	}
	if cfg.GCRatio != 0.5 {
		t.Errorf("Expected GCRatio=0.5, got %f", cfg.GCRatio)
	}
	if cfg.CloseTimeout != 30*time.Second {
		t.Errorf("Expected CloseTimeout=30s, got %v", cfg.CloseTimeout)
	}
	if cfg.NumMemtables != 5 {
		t.Errorf("Expected NumMemtables=5, got %d", cfg.NumMemtables)
	}
	if cfg.BlockCacheSize != 256*1024*1024 {
		t.Errorf("Expected BlockCacheSize=256MB, got %d", cfg.BlockCacheSize)
	}
}

// TestConfig_LoadNewEnvVars tests loading new environment variables
func TestConfig_LoadNewEnvVars(t *testing.T) {
	// Set test environment variables
	os.Setenv("WAL_COMPRESSION", "false")
	os.Setenv("WAL_GC_RATIO", "0.7")
	os.Setenv("WAL_CLOSE_TIMEOUT", "60s")
	os.Setenv("WAL_NUM_MEMTABLES", "10")
	os.Setenv("WAL_BLOCK_CACHE_SIZE", "536870912") // 512MB
	defer func() {
		os.Unsetenv("WAL_COMPRESSION")
		os.Unsetenv("WAL_GC_RATIO")
		os.Unsetenv("WAL_CLOSE_TIMEOUT")
		os.Unsetenv("WAL_NUM_MEMTABLES")
		os.Unsetenv("WAL_BLOCK_CACHE_SIZE")
	}()

	cfg := LoadConfig()

	if cfg.Compression {
		t.Error("Expected Compression=false")
	}
	if cfg.GCRatio != 0.7 {
		t.Errorf("Expected GCRatio=0.7, got %f", cfg.GCRatio)
	}
	if cfg.CloseTimeout != 60*time.Second {
		t.Errorf("Expected CloseTimeout=60s, got %v", cfg.CloseTimeout)
	}
	if cfg.NumMemtables != 10 {
		t.Errorf("Expected NumMemtables=10, got %d", cfg.NumMemtables)
	}
	if cfg.BlockCacheSize != 536870912 {
		t.Errorf("Expected BlockCacheSize=512MB, got %d", cfg.BlockCacheSize)
	}
}

// TestWAL_Compression tests that compression can be enabled
func TestWAL_Compression(t *testing.T) {
	cfg := createTestConfig(t)
	cfg.Compression = true

	wal, err := Open(&cfg)
	if err != nil {
		t.Fatalf("Failed to open WAL with compression: %v", err)
	}
	defer wal.Close()

	ctx := context.Background()
	writeTestEvents(ctx, t, wal, 5)
	assertPendingCount(ctx, t, wal, 5)
}

// TestCompactor_IsRunning tests the IsRunning method.
func TestCompactor_IsRunning(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	compactor := NewCompactor(wal)

	if compactor.IsRunning() {
		t.Error("Compactor should not be running before Start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := compactor.Start(ctx); err != nil {
		t.Fatalf("Failed to start compactor: %v", err)
	}

	if !compactor.IsRunning() {
		t.Error("Compactor should be running after Start")
	}

	compactor.Stop()

	if compactor.IsRunning() {
		t.Error("Compactor should not be running after Stop")
	}
}

// TestCompactor_RunNow tests immediate compaction.
func TestCompactor_RunNow(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	ctx := context.Background()
	writeAndConfirmEvents(ctx, t, wal, 3)

	statsBefore := wal.Stats()
	if statsBefore.ConfirmedCount != 3 {
		t.Errorf("Expected 3 confirmed before compaction, got %d", statsBefore.ConfirmedCount)
	}

	compactor := NewCompactor(wal)
	if err := compactor.RunNow(); err != nil {
		t.Fatalf("RunNow failed: %v", err)
	}

	statsAfter := wal.Stats()
	if statsAfter.ConfirmedCount != 0 {
		t.Errorf("Expected 0 confirmed after RunNow, got %d", statsAfter.ConfirmedCount)
	}
}

// TestCompactor_GetStats tests statistics retrieval.
func TestCompactor_GetStats(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	ctx := context.Background()
	writeAndConfirmEvents(ctx, t, wal, 5)

	compactor := NewCompactor(wal)

	statsBefore := compactor.GetStats()
	if !statsBefore.LastRun.IsZero() {
		t.Error("LastRun should be zero before any compaction")
	}

	if err := compactor.RunNow(); err != nil {
		t.Fatalf("RunNow failed: %v", err)
	}

	statsAfter := compactor.GetStats()
	if statsAfter.LastRun.IsZero() {
		t.Error("LastRun should not be zero after compaction")
	}
	if statsAfter.LastEntriesCount != 5 {
		t.Errorf("Expected 5 entries compacted, got %d", statsAfter.LastEntriesCount)
	}
}

// TestRetryLoop_IsRunning tests the IsRunning method.
func TestRetryLoop_IsRunning(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	pub := &mockPublisher{}
	retryLoop := NewRetryLoop(wal, pub)

	if retryLoop.IsRunning() {
		t.Error("RetryLoop should not be running before Start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := retryLoop.Start(ctx); err != nil {
		t.Fatalf("Failed to start retry loop: %v", err)
	}

	if !retryLoop.IsRunning() {
		t.Error("RetryLoop should be running after Start")
	}

	retryLoop.Stop()

	if retryLoop.IsRunning() {
		t.Error("RetryLoop should not be running after Stop")
	}
}

// TestRetryLoop_GetStats tests statistics retrieval.
func TestRetryLoop_GetStats(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	ctx := context.Background()
	writeTestEvents(ctx, t, wal, 3)

	pub := &mockPublisher{}
	retryLoop := NewRetryLoop(wal, pub)

	stats := retryLoop.GetStats()
	if stats.PendingCount != 3 {
		t.Errorf("Expected 3 pending, got %d", stats.PendingCount)
	}
}

// TestRetryLoop_GetStats_WithAttempts tests stats with retry attempts.
func TestRetryLoop_GetStats_WithAttempts(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	ctx := context.Background()
	ids := writeTestEvents(ctx, t, wal, 1)

	for i := 0; i < 3; i++ {
		if err := wal.UpdateAttempt(ctx, ids[0], "error"); err != nil {
			t.Fatalf("UpdateAttempt failed: %v", err)
		}
	}

	pub := &mockPublisher{}
	retryLoop := NewRetryLoop(wal, pub)

	stats := retryLoop.GetStats()
	if stats.PendingCount != 1 {
		t.Errorf("Expected 1 pending, got %d", stats.PendingCount)
	}
	if stats.TotalAttempts != 3 {
		t.Errorf("Expected 3 total attempts, got %d", stats.TotalAttempts)
	}
	if stats.MaxAttempts != 3 {
		t.Errorf("Expected 3 max attempts, got %d", stats.MaxAttempts)
	}
	if stats.OldestEntry.IsZero() {
		t.Error("OldestEntry should not be zero")
	}
}

// TestWAL_DB tests the DB() accessor method.
func TestWAL_DB(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	if wal.DB() == nil {
		t.Error("DB() should not return nil")
	}
}

// TestWAL_SetMetricsCallback tests setting a metrics callback.
func TestWAL_SetMetricsCallback(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	var callbackInvoked bool
	wal.SetMetricsCallback(func(stats Stats) {
		callbackInvoked = true
	})

	if callbackInvoked {
		t.Log("Callback was invoked during setup")
	}
}

// TestConfig_Error tests the ConfigError.Error method.
func TestConfig_Error(t *testing.T) {
	tests := []struct {
		name  string
		cerr  *ConfigError
		check func(msg string) bool
	}{
		{
			name:  "empty path",
			cerr:  &ConfigError{Field: "Path", Message: "cannot be empty"},
			check: func(msg string) bool { return msg == "WAL config error: Path: cannot be empty" },
		},
		{
			name:  "invalid retry interval",
			cerr:  &ConfigError{Field: "RetryInterval", Message: "must be at least 1s"},
			check: func(msg string) bool { return msg == "WAL config error: RetryInterval: must be at least 1s" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.cerr.Error()
			if !tt.check(errMsg) {
				t.Errorf("Unexpected error message: %s", errMsg)
			}
		})
	}
}

// TestConfig_Validate_AdditionalFields tests additional config validation.
func TestConfig_Validate_AdditionalFields(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Config)
	}{
		{"short CompactInterval", func(c *Config) { c.CompactInterval = 500 * time.Millisecond }},
		{"short EntryTTL", func(c *Config) { c.EntryTTL = 30 * time.Second }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Errorf("Expected validation error for %s", tt.name)
			}
		})
	}
}

// TestRetryLoop_Idempotent tests that start/stop are idempotent.
func TestRetryLoop_Idempotent(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	pub := &mockPublisher{}
	retryLoop := NewRetryLoop(wal, pub)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Double start should be idempotent
	if err := retryLoop.Start(ctx); err != nil {
		t.Fatalf("First start failed: %v", err)
	}
	if err := retryLoop.Start(ctx); err != nil {
		t.Fatalf("Second start should not fail: %v", err)
	}

	// Double stop should be idempotent
	retryLoop.Stop()
	retryLoop.Stop()
}

// TestCompactor_Idempotent tests that start/stop are idempotent.
func TestCompactor_Idempotent(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	compactor := NewCompactor(wal)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Double start should be idempotent
	if err := compactor.Start(ctx); err != nil {
		t.Fatalf("First start failed: %v", err)
	}
	if err := compactor.Start(ctx); err != nil {
		t.Fatalf("Second start should not fail: %v", err)
	}

	// Double stop should be idempotent
	compactor.Stop()
	compactor.Stop()
}

// TestWAL_DeleteEntry_EdgeCases tests edge cases for DeleteEntry.
func TestWAL_DeleteEntry_EdgeCases(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		entryID string
	}{
		{"empty ID", ""},
		{"non-existent ID", "non-existent-id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wal.DeleteEntry(ctx, tt.entryID)
			if err != nil {
				t.Errorf("DeleteEntry(%q) should succeed silently, got %v", tt.entryID, err)
			}
		})
	}
}

// TestWAL_UpdateAttempt_Errors tests error cases for UpdateAttempt.
func TestWAL_UpdateAttempt_Errors(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	tests := []struct {
		name    string
		entryID string
	}{
		{"empty ID", ""},
		{"non-existent ID", "non-existent-id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wal.UpdateAttempt(context.Background(), tt.entryID, "error")
			if !errors.Is(err, ErrEntryNotFound) {
				t.Errorf("UpdateAttempt(%q) error = %v, want ErrEntryNotFound", tt.entryID, err)
			}
		})
	}
}
