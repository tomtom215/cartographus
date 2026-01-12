// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OIDC support.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func TestNewMemoryJTITracker(t *testing.T) {
	tracker := NewMemoryJTITracker()
	if tracker == nil {
		t.Fatal("Expected tracker to be created")
	}
	if tracker.entries == nil {
		t.Error("Expected entries map to be initialized")
	}
}

func TestMemoryJTITracker_CheckAndStore(t *testing.T) {
	ctx := context.Background()

	t.Run("store_new_jti", func(t *testing.T) {
		tracker := NewMemoryJTITracker()
		entry := &JTIEntry{
			JTI:     "jti-123",
			Issuer:  "https://idp.example.com",
			Subject: "user-456",
		}

		err := tracker.CheckAndStore(ctx, entry, time.Hour)
		if err != nil {
			t.Fatalf("CheckAndStore failed: %v", err)
		}

		// Verify it was stored
		used, err := tracker.IsUsed(ctx, "jti-123")
		if err != nil {
			t.Fatalf("IsUsed failed: %v", err)
		}
		if !used {
			t.Error("Expected JTI to be marked as used")
		}
	})

	t.Run("detect_replay_attack", func(t *testing.T) {
		tracker := NewMemoryJTITracker()
		entry := &JTIEntry{
			JTI:     "jti-replay",
			Issuer:  "https://idp.example.com",
			Subject: "user-456",
		}

		// First store should succeed
		err := tracker.CheckAndStore(ctx, entry, time.Hour)
		if err != nil {
			t.Fatalf("First CheckAndStore failed: %v", err)
		}

		// Second store should fail (replay attack)
		err = tracker.CheckAndStore(ctx, entry, time.Hour)
		if !errors.Is(err, ErrJTIAlreadyUsed) {
			t.Errorf("Expected ErrJTIAlreadyUsed, got: %v", err)
		}
	})

	t.Run("allow_reuse_after_expiry", func(t *testing.T) {
		tracker := NewMemoryJTITracker()
		entry := &JTIEntry{
			JTI:     "jti-expiring",
			Issuer:  "https://idp.example.com",
			Subject: "user-456",
		}

		// Store with short TTL
		err := tracker.CheckAndStore(ctx, entry, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("First CheckAndStore failed: %v", err)
		}

		// Wait for expiry
		time.Sleep(100 * time.Millisecond)

		// Should succeed after expiry
		err = tracker.CheckAndStore(ctx, entry, time.Hour)
		if err != nil {
			t.Errorf("CheckAndStore should succeed after expiry: %v", err)
		}
	})

	t.Run("closed_tracker_returns_error", func(t *testing.T) {
		tracker := NewMemoryJTITracker()
		tracker.Close()

		entry := &JTIEntry{JTI: "jti-closed"}
		err := tracker.CheckAndStore(ctx, entry, time.Hour)
		if !errors.Is(err, ErrJTIStoreClosed) {
			t.Errorf("Expected ErrJTIStoreClosed, got: %v", err)
		}
	})
}

func TestMemoryJTITracker_IsUsed(t *testing.T) {
	ctx := context.Background()

	t.Run("unused_jti", func(t *testing.T) {
		tracker := NewMemoryJTITracker()

		used, err := tracker.IsUsed(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("IsUsed failed: %v", err)
		}
		if used {
			t.Error("Expected unused JTI to return false")
		}
	})

	t.Run("used_jti", func(t *testing.T) {
		tracker := NewMemoryJTITracker()
		entry := &JTIEntry{JTI: "jti-used"}
		_ = tracker.CheckAndStore(ctx, entry, time.Hour)

		used, err := tracker.IsUsed(ctx, "jti-used")
		if err != nil {
			t.Fatalf("IsUsed failed: %v", err)
		}
		if !used {
			t.Error("Expected used JTI to return true")
		}
	})

	t.Run("expired_jti", func(t *testing.T) {
		tracker := NewMemoryJTITracker()
		entry := &JTIEntry{JTI: "jti-expired"}
		_ = tracker.CheckAndStore(ctx, entry, 50*time.Millisecond)

		time.Sleep(100 * time.Millisecond)

		used, err := tracker.IsUsed(ctx, "jti-expired")
		if err != nil {
			t.Fatalf("IsUsed failed: %v", err)
		}
		if used {
			t.Error("Expected expired JTI to return false")
		}
	})
}

func TestMemoryJTITracker_CleanupExpired(t *testing.T) {
	ctx := context.Background()
	tracker := NewMemoryJTITracker()

	// Add some entries with different TTLs
	_ = tracker.CheckAndStore(ctx, &JTIEntry{JTI: "jti-short-1"}, 50*time.Millisecond)
	_ = tracker.CheckAndStore(ctx, &JTIEntry{JTI: "jti-short-2"}, 50*time.Millisecond)
	_ = tracker.CheckAndStore(ctx, &JTIEntry{JTI: "jti-long"}, time.Hour)

	// Verify all are stored
	size, _ := tracker.Size(ctx)
	if size != 3 {
		t.Errorf("Expected 3 entries, got %d", size)
	}

	// Wait for short-lived entries to expire
	time.Sleep(100 * time.Millisecond)

	// Cleanup
	count, err := tracker.CleanupExpired(ctx)
	if err != nil {
		t.Fatalf("CleanupExpired failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 entries cleaned up, got %d", count)
	}

	// Verify only long-lived entry remains
	size, _ = tracker.Size(ctx)
	if size != 1 {
		t.Errorf("Expected 1 entry remaining, got %d", size)
	}

	// Verify the long-lived entry is still there
	used, _ := tracker.IsUsed(ctx, "jti-long")
	if !used {
		t.Error("Expected jti-long to still be present")
	}
}

func TestMemoryJTITracker_Size(t *testing.T) {
	ctx := context.Background()
	tracker := NewMemoryJTITracker()

	// Initially empty
	size, err := tracker.Size(ctx)
	if err != nil {
		t.Fatalf("Size failed: %v", err)
	}
	if size != 0 {
		t.Errorf("Expected size 0, got %d", size)
	}

	// Add entries
	for i := 0; i < 5; i++ {
		entry := &JTIEntry{JTI: string(rune('a' + i))}
		_ = tracker.CheckAndStore(ctx, entry, time.Hour)
	}

	size, err = tracker.Size(ctx)
	if err != nil {
		t.Fatalf("Size failed: %v", err)
	}
	if size != 5 {
		t.Errorf("Expected size 5, got %d", size)
	}
}

func TestMemoryJTITracker_Close(t *testing.T) {
	ctx := context.Background()
	tracker := NewMemoryJTITracker()

	// Add an entry
	_ = tracker.CheckAndStore(ctx, &JTIEntry{JTI: "jti-before-close"}, time.Hour)

	// Close
	err := tracker.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// All operations should fail
	_, err = tracker.IsUsed(ctx, "jti-before-close")
	if !errors.Is(err, ErrJTIStoreClosed) {
		t.Error("Expected ErrJTIStoreClosed from IsUsed after close")
	}

	_, err = tracker.Size(ctx)
	if !errors.Is(err, ErrJTIStoreClosed) {
		t.Error("Expected ErrJTIStoreClosed from Size after close")
	}

	_, err = tracker.CleanupExpired(ctx)
	if !errors.Is(err, ErrJTIStoreClosed) {
		t.Error("Expected ErrJTIStoreClosed from CleanupExpired after close")
	}
}

func TestMemoryJTITracker_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	tracker := NewMemoryJTITracker()

	var wg sync.WaitGroup
	errors := make(chan error, 200)

	// Concurrent writes with unique JTIs
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			entry := &JTIEntry{
				JTI:     string(rune('a'+idx%26)) + string(rune('0'+idx/26)),
				Issuer:  "https://idp.example.com",
				Subject: "user",
			}
			if err := tracker.CheckAndStore(ctx, entry, time.Hour); err != nil {
				errors <- err
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			jti := string(rune('a'+idx%26)) + string(rune('0'+idx/26))
			_, err := tracker.IsUsed(ctx, jti)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}
}

func TestMemoryJTITracker_EntryMetadata(t *testing.T) {
	ctx := context.Background()
	tracker := NewMemoryJTITracker()

	entry := &JTIEntry{
		JTI:       "jti-with-metadata",
		Issuer:    "https://idp.example.com",
		Subject:   "user-123",
		SessionID: "session-456",
		SourceIP:  "192.168.1.100",
	}

	err := tracker.CheckAndStore(ctx, entry, time.Hour)
	if err != nil {
		t.Fatalf("CheckAndStore failed: %v", err)
	}

	// Verify metadata was stored
	tracker.mu.RLock()
	stored := tracker.entries["jti-with-metadata"]
	tracker.mu.RUnlock()

	if stored == nil {
		t.Fatal("Expected entry to be stored")
	}
	if stored.Issuer != "https://idp.example.com" {
		t.Errorf("Issuer mismatch: %q", stored.Issuer)
	}
	if stored.Subject != "user-123" {
		t.Errorf("Subject mismatch: %q", stored.Subject)
	}
	if stored.SessionID != "session-456" {
		t.Errorf("SessionID mismatch: %q", stored.SessionID)
	}
	if stored.SourceIP != "192.168.1.100" {
		t.Errorf("SourceIP mismatch: %q", stored.SourceIP)
	}
	if stored.FirstSeen.IsZero() {
		t.Error("FirstSeen should be set")
	}
	if stored.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be set")
	}
}

func TestStartJTICleanupRoutine(t *testing.T) {
	ctx := context.Background()
	tracker := NewMemoryJTITracker()

	// Add short-lived entries
	_ = tracker.CheckAndStore(ctx, &JTIEntry{JTI: "jti-1"}, 50*time.Millisecond)
	_ = tracker.CheckAndStore(ctx, &JTIEntry{JTI: "jti-2"}, 50*time.Millisecond)

	// Start cleanup routine
	done := StartJTICleanupRoutine(tracker, 30*time.Millisecond)

	// Wait for entries to expire and cleanup to run
	time.Sleep(150 * time.Millisecond)

	// Stop cleanup routine
	close(done)

	// Verify entries were cleaned up
	size, _ := tracker.Size(ctx)
	if size != 0 {
		t.Errorf("Expected 0 entries after cleanup, got %d", size)
	}
}

func TestJTIEntry_Fields(t *testing.T) {
	now := time.Now()
	entry := JTIEntry{
		JTI:       "test-jti",
		Issuer:    "https://issuer.example.com",
		Subject:   "user@example.com",
		SessionID: "session-xyz",
		FirstSeen: now,
		ExpiresAt: now.Add(time.Hour),
		SourceIP:  "10.0.0.1",
	}

	if entry.JTI != "test-jti" {
		t.Errorf("JTI mismatch: %q", entry.JTI)
	}
	if entry.Issuer != "https://issuer.example.com" {
		t.Errorf("Issuer mismatch: %q", entry.Issuer)
	}
	if entry.Subject != "user@example.com" {
		t.Errorf("Subject mismatch: %q", entry.Subject)
	}
	if entry.SessionID != "session-xyz" {
		t.Errorf("SessionID mismatch: %q", entry.SessionID)
	}
	if entry.SourceIP != "10.0.0.1" {
		t.Errorf("SourceIP mismatch: %q", entry.SourceIP)
	}
	if entry.FirstSeen != now {
		t.Errorf("FirstSeen mismatch")
	}
	if entry.ExpiresAt != now.Add(time.Hour) {
		t.Errorf("ExpiresAt mismatch")
	}
}

func TestJTITracker_Interface(t *testing.T) {
	// Verify MemoryJTITracker implements JTITracker interface
	var _ JTITracker = (*MemoryJTITracker)(nil)
	var _ JTITracker = (*BadgerJTITracker)(nil)
}

func TestErrJTIAlreadyUsed(t *testing.T) {
	err := ErrJTIAlreadyUsed
	if err.Error() != "JTI already used (replay attack prevented)" {
		t.Errorf("Unexpected error message: %q", err.Error())
	}
}

func TestErrJTIStoreClosed(t *testing.T) {
	err := ErrJTIStoreClosed
	if err.Error() != "JTI store is closed" {
		t.Errorf("Unexpected error message: %q", err.Error())
	}
}

// TestMemoryJTITracker_ReplayWithDifferentMetadata tests that replay detection
// works even when metadata differs between requests.
func TestMemoryJTITracker_ReplayWithDifferentMetadata(t *testing.T) {
	ctx := context.Background()
	tracker := NewMemoryJTITracker()

	// First request
	entry1 := &JTIEntry{
		JTI:      "same-jti",
		Issuer:   "https://idp.example.com",
		Subject:  "user-1",
		SourceIP: "192.168.1.1",
	}
	err := tracker.CheckAndStore(ctx, entry1, time.Hour)
	if err != nil {
		t.Fatalf("First store failed: %v", err)
	}

	// Second request with same JTI but different metadata (attack from different IP)
	entry2 := &JTIEntry{
		JTI:      "same-jti", // Same JTI
		Issuer:   "https://idp.example.com",
		Subject:  "user-1",
		SourceIP: "10.0.0.1", // Different IP
	}
	err = tracker.CheckAndStore(ctx, entry2, time.Hour)
	if !errors.Is(err, ErrJTIAlreadyUsed) {
		t.Errorf("Expected replay detection even with different metadata: %v", err)
	}
}

// BenchmarkMemoryJTITracker_CheckAndStore measures performance of JTI storage.
func BenchmarkMemoryJTITracker_CheckAndStore(b *testing.B) {
	ctx := context.Background()
	tracker := NewMemoryJTITracker()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry := &JTIEntry{
			JTI:     string(rune(i % 1000000)),
			Issuer:  "https://idp.example.com",
			Subject: "user",
		}
		_ = tracker.CheckAndStore(ctx, entry, time.Hour)
	}
}

// BenchmarkMemoryJTITracker_IsUsed measures lookup performance.
func BenchmarkMemoryJTITracker_IsUsed(b *testing.B) {
	ctx := context.Background()
	tracker := NewMemoryJTITracker()

	// Pre-populate with entries
	for i := 0; i < 10000; i++ {
		entry := &JTIEntry{JTI: string(rune(i))}
		_ = tracker.CheckAndStore(ctx, entry, time.Hour)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tracker.IsUsed(ctx, string(rune(i%10000)))
	}
}

// ============================
// BadgerJTITracker Tests
// ============================

// setupBadgerJTITracker creates a BadgerDB in-memory instance for testing.
func setupBadgerJTITracker(t *testing.T) (*BadgerJTITracker, func()) {
	t.Helper()

	// Use in-memory storage for testing
	opts := badger.DefaultOptions("").WithInMemory(true).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("Failed to open badger: %v", err)
	}

	tracker := NewBadgerJTITracker(db, "test:")
	cleanup := func() {
		tracker.Close()
		db.Close()
	}
	return tracker, cleanup
}

func TestNewBadgerJTITracker(t *testing.T) {
	opts := badger.DefaultOptions("").WithInMemory(true).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("Failed to open badger: %v", err)
	}
	defer db.Close()

	t.Run("with_default_prefix", func(t *testing.T) {
		tracker := NewBadgerJTITracker(db, "")
		defer tracker.Close()

		if tracker == nil {
			t.Fatal("Expected tracker to be created")
		}
		if string(tracker.prefix) != "jti:" {
			t.Errorf("Expected default prefix 'jti:', got %q", string(tracker.prefix))
		}
	})

	t.Run("with_custom_prefix", func(t *testing.T) {
		tracker := NewBadgerJTITracker(db, "custom:")
		defer tracker.Close()

		if string(tracker.prefix) != "custom:" {
			t.Errorf("Expected prefix 'custom:', got %q", string(tracker.prefix))
		}
	})
}

func TestBadgerJTITracker_CheckAndStore(t *testing.T) {
	ctx := context.Background()

	t.Run("store_new_jti", func(t *testing.T) {
		tracker, cleanup := setupBadgerJTITracker(t)
		defer cleanup()

		entry := &JTIEntry{
			JTI:     "badger-jti-123",
			Issuer:  "https://idp.example.com",
			Subject: "user-456",
		}

		err := tracker.CheckAndStore(ctx, entry, time.Hour)
		if err != nil {
			t.Fatalf("CheckAndStore failed: %v", err)
		}

		// Verify it was stored
		used, err := tracker.IsUsed(ctx, "badger-jti-123")
		if err != nil {
			t.Fatalf("IsUsed failed: %v", err)
		}
		if !used {
			t.Error("Expected JTI to be marked as used")
		}
	})

	t.Run("detect_replay_attack", func(t *testing.T) {
		tracker, cleanup := setupBadgerJTITracker(t)
		defer cleanup()

		entry := &JTIEntry{
			JTI:     "badger-jti-replay",
			Issuer:  "https://idp.example.com",
			Subject: "user-456",
		}

		// First store should succeed
		err := tracker.CheckAndStore(ctx, entry, time.Hour)
		if err != nil {
			t.Fatalf("First CheckAndStore failed: %v", err)
		}

		// Second store should fail (replay attack)
		err = tracker.CheckAndStore(ctx, entry, time.Hour)
		if !errors.Is(err, ErrJTIAlreadyUsed) {
			t.Errorf("Expected ErrJTIAlreadyUsed, got: %v", err)
		}
	})

	t.Run("allow_reuse_after_expiry", func(t *testing.T) {
		tracker, cleanup := setupBadgerJTITracker(t)
		defer cleanup()

		entry := &JTIEntry{
			JTI:     "badger-jti-expiring",
			Issuer:  "https://idp.example.com",
			Subject: "user-456",
		}

		// Store with short TTL
		err := tracker.CheckAndStore(ctx, entry, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("First CheckAndStore failed: %v", err)
		}

		// Wait for expiry
		time.Sleep(100 * time.Millisecond)

		// Should succeed after expiry
		err = tracker.CheckAndStore(ctx, entry, time.Hour)
		if err != nil {
			t.Errorf("CheckAndStore should succeed after expiry: %v", err)
		}
	})

	t.Run("closed_tracker_returns_error", func(t *testing.T) {
		tracker, cleanup := setupBadgerJTITracker(t)
		cleanup() // Close early

		entry := &JTIEntry{JTI: "badger-jti-closed"}
		err := tracker.CheckAndStore(ctx, entry, time.Hour)
		if !errors.Is(err, ErrJTIStoreClosed) {
			t.Errorf("Expected ErrJTIStoreClosed, got: %v", err)
		}
	})
}

func TestBadgerJTITracker_IsUsed(t *testing.T) {
	ctx := context.Background()

	t.Run("unused_jti", func(t *testing.T) {
		tracker, cleanup := setupBadgerJTITracker(t)
		defer cleanup()

		used, err := tracker.IsUsed(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("IsUsed failed: %v", err)
		}
		if used {
			t.Error("Expected unused JTI to return false")
		}
	})

	t.Run("used_jti", func(t *testing.T) {
		tracker, cleanup := setupBadgerJTITracker(t)
		defer cleanup()

		entry := &JTIEntry{JTI: "badger-jti-used"}
		_ = tracker.CheckAndStore(ctx, entry, time.Hour)

		used, err := tracker.IsUsed(ctx, "badger-jti-used")
		if err != nil {
			t.Fatalf("IsUsed failed: %v", err)
		}
		if !used {
			t.Error("Expected used JTI to return true")
		}
	})

	t.Run("closed_tracker_returns_error", func(t *testing.T) {
		tracker, cleanup := setupBadgerJTITracker(t)
		cleanup() // Close early

		_, err := tracker.IsUsed(ctx, "any")
		if !errors.Is(err, ErrJTIStoreClosed) {
			t.Errorf("Expected ErrJTIStoreClosed, got: %v", err)
		}
	})
}

func TestBadgerJTITracker_Size(t *testing.T) {
	ctx := context.Background()
	tracker, cleanup := setupBadgerJTITracker(t)
	defer cleanup()

	// Initially empty
	size, err := tracker.Size(ctx)
	if err != nil {
		t.Fatalf("Size failed: %v", err)
	}
	if size != 0 {
		t.Errorf("Expected size 0, got %d", size)
	}

	// Add entries
	for i := 0; i < 5; i++ {
		entry := &JTIEntry{JTI: "badger-jti-" + string(rune('a'+i))}
		_ = tracker.CheckAndStore(ctx, entry, time.Hour)
	}

	size, err = tracker.Size(ctx)
	if err != nil {
		t.Fatalf("Size failed: %v", err)
	}
	if size != 5 {
		t.Errorf("Expected size 5, got %d", size)
	}
}

func TestBadgerJTITracker_CleanupExpired(t *testing.T) {
	ctx := context.Background()
	tracker, cleanup := setupBadgerJTITracker(t)
	defer cleanup()

	// Add some entries with different TTLs
	_ = tracker.CheckAndStore(ctx, &JTIEntry{JTI: "badger-short-1"}, 50*time.Millisecond)
	_ = tracker.CheckAndStore(ctx, &JTIEntry{JTI: "badger-short-2"}, 50*time.Millisecond)
	_ = tracker.CheckAndStore(ctx, &JTIEntry{JTI: "badger-long"}, time.Hour)

	// Wait for short-lived entries to expire
	time.Sleep(100 * time.Millisecond)

	// Cleanup - BadgerDB may also clean up expired entries via its native TTL
	// mechanism, so we just verify no error occurs
	_, err := tracker.CleanupExpired(ctx)
	if err != nil {
		t.Fatalf("CleanupExpired failed: %v", err)
	}

	// The long-lived entry should still be usable
	used, err := tracker.IsUsed(ctx, "badger-long")
	if err != nil {
		t.Fatalf("IsUsed failed: %v", err)
	}
	if !used {
		t.Error("Expected badger-long to still be usable")
	}

	// The short-lived entries should be expired (not usable)
	used, err = tracker.IsUsed(ctx, "badger-short-1")
	if err != nil {
		t.Fatalf("IsUsed failed: %v", err)
	}
	if used {
		t.Error("Expected badger-short-1 to be expired")
	}
}

func TestBadgerJTITracker_Close(t *testing.T) {
	ctx := context.Background()
	tracker, cleanup := setupBadgerJTITracker(t)
	defer cleanup()

	// Add an entry
	_ = tracker.CheckAndStore(ctx, &JTIEntry{JTI: "badger-before-close"}, time.Hour)

	// Close
	err := tracker.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// All operations should fail
	_, err = tracker.IsUsed(ctx, "badger-before-close")
	if !errors.Is(err, ErrJTIStoreClosed) {
		t.Error("Expected ErrJTIStoreClosed from IsUsed after close")
	}

	_, err = tracker.Size(ctx)
	if !errors.Is(err, ErrJTIStoreClosed) {
		t.Error("Expected ErrJTIStoreClosed from Size after close")
	}

	_, err = tracker.CleanupExpired(ctx)
	if !errors.Is(err, ErrJTIStoreClosed) {
		t.Error("Expected ErrJTIStoreClosed from CleanupExpired after close")
	}
}

func TestBadgerJTITracker_makeKey(t *testing.T) {
	opts := badger.DefaultOptions("").WithInMemory(true).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("Failed to open badger: %v", err)
	}
	defer db.Close()

	tracker := NewBadgerJTITracker(db, "test:")
	defer tracker.Close()

	key := tracker.makeKey("my-jti-value")
	expected := []byte("test:my-jti-value")

	if string(key) != string(expected) {
		t.Errorf("makeKey() = %q, want %q", string(key), string(expected))
	}
}
