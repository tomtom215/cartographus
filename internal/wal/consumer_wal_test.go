// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal

package wal

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
)

// testMediaEvent is a simplified event structure for testing
type testMediaEvent struct {
	EventID   string    `json:"event_id"`
	Source    string    `json:"source"`
	UserID    int       `json:"user_id"`
	Title     string    `json:"title"`
	StartedAt time.Time `json:"started_at"`
}

// testConsumerWALConfig returns a configuration suitable for testing
func testConsumerWALConfig(path string) *ConsumerWALConfig {
	return &ConsumerWALConfig{
		Path:             path,
		SyncWrites:       false, // Disable for faster tests
		EntryTTL:         time.Hour,
		MaxRetries:       5,
		RetryInterval:    time.Second,
		RetryBackoff:     time.Second,
		MemTableSize:     16 * 1024 * 1024, // 16MB - larger for BadgerDB requirements
		ValueLogFileSize: 16 * 1024 * 1024, // 16MB
		NumCompactors:    2,
		Compression:      false,
		CloseTimeout:     5 * time.Second,
		LeaseDuration:    30 * time.Second, // Minimum for durable leasing
	}
}

func TestConsumerWAL_WriteAndConfirm(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := testConsumerWALConfig(filepath.Join(dir, "consumer-wal"))

	wal, err := OpenConsumerWAL(cfg)
	if err != nil {
		t.Fatalf("OpenConsumerWAL failed: %v", err)
	}
	defer func() {
		if err := wal.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	}()

	ctx := context.Background()

	// Create test event
	event := &testMediaEvent{
		EventID:   "test-event-1",
		Source:    "tautulli",
		UserID:    42,
		Title:     "Test Movie",
		StartedAt: time.Now(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	transactionID := GenerateTransactionID("tautulli", "test-event-1")

	// Write to WAL
	entryID, err := wal.Write(ctx, payload, transactionID, "playback.tautulli.movie", "msg-123")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if entryID == "" {
		t.Error("Expected non-empty entry ID")
	}

	// Verify entry is pending
	entry, err := wal.GetEntry(ctx, entryID)
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}

	if entry.TransactionID != transactionID {
		t.Errorf("Expected transaction ID %s, got %s", transactionID, entry.TransactionID)
	}

	if entry.Confirmed {
		t.Error("Entry should not be confirmed yet")
	}

	// Confirm the entry
	if err := wal.Confirm(ctx, entryID); err != nil {
		t.Fatalf("Confirm failed: %v", err)
	}

	// Verify entry is no longer pending
	_, err = wal.GetEntry(ctx, entryID)
	if !errors.Is(err, ErrEntryNotFound) {
		t.Errorf("Expected ErrEntryNotFound, got %v", err)
	}

	// Verify stats
	stats := wal.Stats()
	if stats.TotalWrites != 1 {
		t.Errorf("Expected 1 write, got %d", stats.TotalWrites)
	}
	if stats.TotalConfirms != 1 {
		t.Errorf("Expected 1 confirm, got %d", stats.TotalConfirms)
	}
}

func TestConsumerWAL_GetPending(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := testConsumerWALConfig(filepath.Join(dir, "consumer-wal"))

	wal, err := OpenConsumerWAL(cfg)
	if err != nil {
		t.Fatalf("OpenConsumerWAL failed: %v", err)
	}
	defer wal.Close()

	ctx := context.Background()

	// Write multiple events
	for i := 0; i < 3; i++ {
		event := &testMediaEvent{
			EventID: "event-" + string(rune('a'+i)),
			Source:  "tautulli",
			UserID:  i + 1,
		}

		payload, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		transactionID := GenerateTransactionID("tautulli", event.EventID)
		_, err = wal.Write(ctx, payload, transactionID, "playback.tautulli.movie", "")
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	// Get pending entries
	entries, err := wal.GetPending(ctx)
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("Expected 3 pending entries, got %d", len(entries))
	}
}

func TestConsumerWAL_MarkFailed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := testConsumerWALConfig(filepath.Join(dir, "consumer-wal"))

	wal, err := OpenConsumerWAL(cfg)
	if err != nil {
		t.Fatalf("OpenConsumerWAL failed: %v", err)
	}
	defer wal.Close()

	ctx := context.Background()

	// Write an event
	event := &testMediaEvent{EventID: "failed-event", Source: "tautulli"}
	payload, _ := json.Marshal(event)
	transactionID := GenerateTransactionID("tautulli", "failed-event")
	entryID, err := wal.Write(ctx, payload, transactionID, "", "")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Mark as failed
	err = wal.MarkFailed(ctx, entryID, "max retries exceeded")
	if err != nil {
		t.Fatalf("MarkFailed failed: %v", err)
	}

	// Verify it's no longer in pending
	_, err = wal.GetEntry(ctx, entryID)
	if !errors.Is(err, ErrEntryNotFound) {
		t.Errorf("Expected ErrEntryNotFound, got %v", err)
	}

	// Verify failure metrics
	stats := wal.Stats()
	if stats.TotalFailures != 1 {
		t.Errorf("Expected 1 failure, got %d", stats.TotalFailures)
	}
}

func TestConsumerWAL_UpdateAttempt(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := testConsumerWALConfig(filepath.Join(dir, "consumer-wal"))

	wal, err := OpenConsumerWAL(cfg)
	if err != nil {
		t.Fatalf("OpenConsumerWAL failed: %v", err)
	}
	defer wal.Close()

	ctx := context.Background()

	// Write an event
	event := &testMediaEvent{EventID: "retry-event", Source: "tautulli"}
	payload, _ := json.Marshal(event)
	transactionID := GenerateTransactionID("tautulli", "retry-event")
	entryID, err := wal.Write(ctx, payload, transactionID, "", "")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Update attempt count multiple times
	for i := 0; i < 3; i++ {
		err = wal.UpdateAttempt(ctx, entryID, "connection refused")
		if err != nil {
			t.Fatalf("UpdateAttempt failed: %v", err)
		}
	}

	// Verify attempt count
	entry, err := wal.GetEntry(ctx, entryID)
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}

	if entry.Attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", entry.Attempts)
	}

	if entry.LastError != "connection refused" {
		t.Errorf("Expected 'connection refused', got %s", entry.LastError)
	}
}

func TestConsumerWAL_CleanupConfirmed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := testConsumerWALConfig(filepath.Join(dir, "consumer-wal"))

	wal, err := OpenConsumerWAL(cfg)
	if err != nil {
		t.Fatalf("OpenConsumerWAL failed: %v", err)
	}
	defer wal.Close()

	ctx := context.Background()

	// Write and confirm multiple events
	for i := 0; i < 5; i++ {
		event := &testMediaEvent{EventID: "cleanup-event-" + string(rune('a'+i)), Source: "tautulli"}
		payload, _ := json.Marshal(event)
		transactionID := GenerateTransactionID("tautulli", event.EventID)
		entryID, err := wal.Write(ctx, payload, transactionID, "", "")
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		if err := wal.Confirm(ctx, entryID); err != nil {
			t.Fatalf("Confirm failed: %v", err)
		}
	}

	// Run cleanup
	deleted, err := wal.CleanupConfirmed(ctx)
	if err != nil {
		t.Fatalf("CleanupConfirmed failed: %v", err)
	}

	if deleted != 5 {
		t.Errorf("Expected 5 deleted, got %d", deleted)
	}
}

func TestConsumerWAL_ClosedWALErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := testConsumerWALConfig(filepath.Join(dir, "consumer-wal"))

	wal, err := OpenConsumerWAL(cfg)
	if err != nil {
		t.Fatalf("OpenConsumerWAL failed: %v", err)
	}

	// Close the WAL
	if err := wal.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	ctx := context.Background()

	// All operations should fail with ErrWALClosed
	_, err = wal.Write(ctx, []byte("{}"), "txn-1", "", "")
	if !errors.Is(err, ErrWALClosed) {
		t.Errorf("Write: expected ErrWALClosed, got %v", err)
	}

	err = wal.Confirm(ctx, "entry-1")
	if !errors.Is(err, ErrWALClosed) {
		t.Errorf("Confirm: expected ErrWALClosed, got %v", err)
	}

	_, err = wal.GetPending(ctx)
	if !errors.Is(err, ErrWALClosed) {
		t.Errorf("GetPending: expected ErrWALClosed, got %v", err)
	}
}

func TestGenerateTransactionID(t *testing.T) {
	t.Parallel()

	txn1 := GenerateTransactionID("tautulli", "event-1")
	txn2 := GenerateTransactionID("tautulli", "event-1")

	// Even same source/event should have different transaction IDs due to atomic counter
	if txn1 == txn2 {
		t.Error("Transaction IDs should be unique due to atomic counter")
	}

	// Format should be source:event_id:counter
	// DETERMINISM: Uses atomic counter instead of timestamp for reproducible ordering
	if !strings.Contains(txn1, "tautulli:event-1:") {
		t.Errorf("Transaction ID format incorrect: %s", txn1)
	}
}

// mockRecoveryCallback implements RecoveryCallback for testing
type mockRecoveryCallback struct {
	existingTransactions map[string]bool
	insertedEvents       [][]byte
	failedEvents         []*ConsumerWALEntry
	insertError          error
}

func (m *mockRecoveryCallback) TransactionIDExists(ctx context.Context, transactionID string) (bool, error) {
	return m.existingTransactions[transactionID], nil
}

func (m *mockRecoveryCallback) InsertEvent(ctx context.Context, payload []byte, transactionID string) error {
	if m.insertError != nil {
		return m.insertError
	}
	m.insertedEvents = append(m.insertedEvents, payload)
	return nil
}

func (m *mockRecoveryCallback) InsertFailedEvent(ctx context.Context, entry *ConsumerWALEntry, reason string) error {
	m.failedEvents = append(m.failedEvents, entry)
	return nil
}

func TestConsumerWAL_RecoverOnStartup(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := testConsumerWALConfig(filepath.Join(dir, "consumer-wal"))

	wal, err := OpenConsumerWAL(cfg)
	if err != nil {
		t.Fatalf("OpenConsumerWAL failed: %v", err)
	}
	defer wal.Close()

	ctx := context.Background()

	// Write events that will be "recovered"
	var txnIDs []string
	for i := 0; i < 3; i++ {
		event := &testMediaEvent{EventID: "recovery-event-" + string(rune('a'+i)), Source: "tautulli"}
		payload, _ := json.Marshal(event)
		txnID := GenerateTransactionID("tautulli", event.EventID)
		txnIDs = append(txnIDs, txnID)
		_, err := wal.Write(ctx, payload, txnID, "", "")
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	// Simulate one already being in DuckDB
	callback := &mockRecoveryCallback{
		existingTransactions: map[string]bool{
			txnIDs[0]: true, // First one is already committed
		},
	}

	result, err := wal.RecoverOnStartup(ctx, callback)
	if err != nil {
		t.Fatalf("RecoverOnStartup failed: %v", err)
	}

	if result.TotalPending != 3 {
		t.Errorf("Expected 3 pending, got %d", result.TotalPending)
	}

	if result.AlreadyCommitted != 1 {
		t.Errorf("Expected 1 already committed, got %d", result.AlreadyCommitted)
	}

	if result.Recovered != 2 {
		t.Errorf("Expected 2 recovered, got %d", result.Recovered)
	}

	if len(callback.insertedEvents) != 2 {
		t.Errorf("Expected 2 inserted events, got %d", len(callback.insertedEvents))
	}
}

func TestConsumerWAL_ConfigValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cfg       ConsumerWALConfig
		expectErr bool
	}{
		{
			name: "valid config",
			cfg: ConsumerWALConfig{
				Path:          "/tmp/test-wal",
				NumCompactors: 2,
				MaxRetries:    5,
				LeaseDuration: 30 * time.Second,
			},
			expectErr: false,
		},
		{
			name: "empty path",
			cfg: ConsumerWALConfig{
				Path:          "",
				NumCompactors: 2,
				MaxRetries:    5,
			},
			expectErr: true,
		},
		{
			name: "too few compactors",
			cfg: ConsumerWALConfig{
				Path:          "/tmp/test-wal",
				NumCompactors: 1,
				MaxRetries:    5,
			},
			expectErr: true,
		},
		{
			name: "zero max retries",
			cfg: ConsumerWALConfig{
				Path:          "/tmp/test-wal",
				NumCompactors: 2,
				MaxRetries:    0,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// Durable Leasing Tests for Consumer WAL

// TestConsumerWAL_TryClaimEntryDurable tests durable leasing for concurrent processing prevention.
func TestConsumerWAL_TryClaimEntryDurable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := testConsumerWALConfig(filepath.Join(dir, "consumer-wal"))
	cwal, err := OpenConsumerWAL(cfg)
	if err != nil {
		t.Fatalf("OpenConsumerWAL failed: %v", err)
	}
	defer cwal.Close()
	ctx := context.Background()

	// Write an entry
	payload := json.RawMessage(`{"event_id":"test-1"}`)
	transactionID := GenerateTransactionID("tautulli", "test-1")
	id, err := cwal.Write(ctx, payload, transactionID, "playback.tautulli", "msg-1")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// First claim should succeed
	claimed, err := cwal.TryClaimEntryDurable(ctx, id, "processor-1")
	if err != nil {
		t.Fatalf("TryClaimEntryDurable failed: %v", err)
	}
	if !claimed {
		t.Error("First claim should succeed")
	}

	// Second claim by different processor should fail
	claimed, err = cwal.TryClaimEntryDurable(ctx, id, "processor-2")
	if err != nil {
		t.Fatalf("TryClaimEntryDurable failed: %v", err)
	}
	if claimed {
		t.Error("Second claim should fail when lease is active")
	}

	// Same processor can reclaim
	claimed, err = cwal.TryClaimEntryDurable(ctx, id, "processor-1")
	if err != nil {
		t.Fatalf("TryClaimEntryDurable failed: %v", err)
	}
	if !claimed {
		t.Error("Same processor should be able to reclaim")
	}
}

// TestConsumerWAL_ReleaseLeaseDurable tests explicit lease release.
func TestConsumerWAL_ReleaseLeaseDurable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := testConsumerWALConfig(filepath.Join(dir, "consumer-wal"))
	cwal, err := OpenConsumerWAL(cfg)
	if err != nil {
		t.Fatalf("OpenConsumerWAL failed: %v", err)
	}
	defer cwal.Close()
	ctx := context.Background()

	// Write an entry
	payload := json.RawMessage(`{"event_id":"release-test"}`)
	transactionID := GenerateTransactionID("plex", "release-test")
	id, err := cwal.Write(ctx, payload, transactionID, "playback.plex", "msg-2")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Claim the entry
	claimed, err := cwal.TryClaimEntryDurable(ctx, id, "processor-1")
	if err != nil || !claimed {
		t.Fatalf("Initial claim failed: claimed=%v, err=%v", claimed, err)
	}

	// Release the lease
	err = cwal.ReleaseLeaseDurable(ctx, id)
	if err != nil {
		t.Fatalf("ReleaseLeaseDurable failed: %v", err)
	}

	// Another processor can now claim immediately
	claimed, err = cwal.TryClaimEntryDurable(ctx, id, "processor-2")
	if err != nil {
		t.Fatalf("TryClaimEntryDurable failed: %v", err)
	}
	if !claimed {
		t.Error("Should be able to claim after explicit release")
	}
}

// TestConsumerWAL_ExtendLease tests lease extension.
func TestConsumerWAL_ExtendLease(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := testConsumerWALConfig(filepath.Join(dir, "consumer-wal"))
	cfg.LeaseDuration = 200 * time.Millisecond // Short for testing
	cwal, err := OpenConsumerWALForTesting(cfg)
	if err != nil {
		t.Fatalf("OpenConsumerWALForTesting failed: %v", err)
	}
	defer cwal.Close()
	ctx := context.Background()

	// Write an entry
	payload := json.RawMessage(`{"event_id":"extend-test"}`)
	transactionID := GenerateTransactionID("jellyfin", "extend-test")
	id, err := cwal.Write(ctx, payload, transactionID, "playback.jellyfin", "msg-3")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Claim the entry
	claimed, err := cwal.TryClaimEntryDurable(ctx, id, "processor-1")
	if err != nil || !claimed {
		t.Fatalf("Initial claim failed: claimed=%v, err=%v", claimed, err)
	}

	// Wait a bit but not until expiry
	time.Sleep(100 * time.Millisecond)

	// Extend the lease
	err = cwal.ExtendLease(ctx, id, "processor-1")
	if err != nil {
		t.Fatalf("ExtendLease failed: %v", err)
	}

	// Wait past original expiry
	time.Sleep(150 * time.Millisecond)

	// Another processor should still not be able to claim (lease was extended)
	claimed, err = cwal.TryClaimEntryDurable(ctx, id, "processor-2")
	if err != nil {
		t.Fatalf("TryClaimEntryDurable failed: %v", err)
	}
	if claimed {
		t.Error("Lease was extended, so second claim should fail")
	}
}
