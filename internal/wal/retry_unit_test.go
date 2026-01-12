// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal

package wal

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// TestCalculateBackoff_EdgeCases tests calculateBackoff with various input values.
func TestCalculateBackoff_EdgeCases(t *testing.T) {
	cfg := createTestConfig(t)
	cfg.RetryBackoff = 1 * time.Second

	wal, err := Open(&cfg)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer wal.Close()

	pub := &mockPublisher{}
	retryLoop := NewRetryLoop(wal, pub)

	tests := []struct {
		name         string
		attempts     int
		expectedMin  time.Duration
		expectedMax  time.Duration
		expectMaxCap bool
		description  string
	}{
		{
			name:         "zero attempts",
			attempts:     0,
			expectedMin:  1 * time.Second,
			expectedMax:  1 * time.Second,
			expectMaxCap: false,
			description:  "base * 2^0 = base",
		},
		{
			name:         "one attempt",
			attempts:     1,
			expectedMin:  2 * time.Second,
			expectedMax:  2 * time.Second,
			expectMaxCap: false,
			description:  "base * 2^1 = 2 * base",
		},
		{
			name:         "two attempts",
			attempts:     2,
			expectedMin:  4 * time.Second,
			expectedMax:  4 * time.Second,
			expectMaxCap: false,
			description:  "base * 2^2 = 4 * base",
		},
		{
			name:         "five attempts",
			attempts:     5,
			expectedMin:  32 * time.Second,
			expectedMax:  32 * time.Second,
			expectMaxCap: false,
			description:  "base * 2^5 = 32 * base",
		},
		{
			name:         "ten attempts",
			attempts:     10,
			expectedMin:  5 * time.Minute, // Capped at 5 minutes
			expectedMax:  5 * time.Minute,
			expectMaxCap: true,
			description:  "base * 2^10 = 1024s > 5min, capped",
		},
		{
			name:         "twenty attempts",
			attempts:     20,
			expectedMin:  5 * time.Minute, // Capped at 5 minutes
			expectedMax:  5 * time.Minute,
			expectMaxCap: true,
			description:  "Exponential overflow capped at 5min",
		},
		{
			name:         "large attempts",
			attempts:     100,
			expectedMin:  5 * time.Minute, // Capped at 5 minutes
			expectedMax:  5 * time.Minute,
			expectMaxCap: true,
			description:  "Very large attempts still capped at 5min",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backoff := retryLoop.calculateBackoff(tt.attempts)

			if backoff < tt.expectedMin || backoff > tt.expectedMax {
				t.Errorf("%s: expected backoff between %v and %v, got %v",
					tt.description, tt.expectedMin, tt.expectedMax, backoff)
			}

			// Verify 5-minute cap is enforced
			if tt.expectMaxCap && backoff != 5*time.Minute {
				t.Errorf("Expected backoff to be capped at 5 minutes, got %v", backoff)
			}
		})
	}
}

// TestCalculateBackoff_FiveMinuteCap tests that backoff is always capped at 5 minutes.
func TestCalculateBackoff_FiveMinuteCap(t *testing.T) {
	cfg := createTestConfig(t)
	cfg.RetryBackoff = 10 * time.Second // Larger base to reach cap faster

	wal, err := Open(&cfg)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer wal.Close()

	pub := &mockPublisher{}
	retryLoop := NewRetryLoop(wal, pub)

	maxCap := 5 * time.Minute

	// Test a range of attempts that should all hit the cap
	for attempts := 5; attempts < 50; attempts++ {
		backoff := retryLoop.calculateBackoff(attempts)
		if backoff > maxCap {
			t.Errorf("Backoff for %d attempts (%v) exceeds 5 minute cap", attempts, backoff)
		}
		if backoff > maxCap+time.Millisecond {
			t.Errorf("Backoff for %d attempts exceeds cap: %v", attempts, backoff)
		}
	}
}

// TestIsReadyForRetry_BoundaryConditions tests isReadyForRetry with edge cases.
func TestIsReadyForRetry_BoundaryConditions(t *testing.T) {
	cfg := createFastTestConfig(t)
	cfg.RetryBackoff = 100 * time.Millisecond

	wal, err := OpenForTesting(&cfg)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer wal.Close()

	pub := &mockPublisher{}
	retryLoop := NewRetryLoop(wal, pub)

	t.Run("zero LastAttemptAt always ready", func(t *testing.T) {
		entry := &Entry{
			ID:            "test-1",
			CreatedAt:     time.Now(),
			LastAttemptAt: time.Time{}, // Zero value
			Attempts:      0,
		}
		if !retryLoop.isReadyForRetry(entry) {
			t.Error("Entry with zero LastAttemptAt should be ready for retry")
		}

		// Even with many attempts, zero LastAttemptAt means ready
		entry.Attempts = 10
		if !retryLoop.isReadyForRetry(entry) {
			t.Error("Entry with zero LastAttemptAt should be ready regardless of attempts")
		}
	})

	t.Run("just created not ready", func(t *testing.T) {
		now := time.Now()
		entry := &Entry{
			ID:            "test-2",
			CreatedAt:     now,
			LastAttemptAt: now, // Just attempted
			Attempts:      1,
		}
		// Should not be ready because backoff hasn't elapsed
		if retryLoop.isReadyForRetry(entry) {
			t.Error("Entry just attempted should not be ready for retry")
		}
	})

	t.Run("backoff elapsed ready", func(t *testing.T) {
		// Create entry with LastAttemptAt in the past
		entry := &Entry{
			ID:            "test-3",
			CreatedAt:     time.Now().Add(-1 * time.Hour),
			LastAttemptAt: time.Now().Add(-1 * time.Second), // 1 second ago
			Attempts:      0,                                // backoff = 100ms for 0 attempts
		}
		// For 0 attempts, backoff = 100ms. 1 second elapsed > 100ms
		if !retryLoop.isReadyForRetry(entry) {
			t.Error("Entry should be ready after backoff has elapsed")
		}
	})

	t.Run("backoff not yet elapsed", func(t *testing.T) {
		// Create entry with recent LastAttemptAt
		entry := &Entry{
			ID:            "test-4",
			CreatedAt:     time.Now(),
			LastAttemptAt: time.Now().Add(-10 * time.Millisecond), // 10ms ago
			Attempts:      5,                                      // backoff = 100ms * 2^5 = 3.2s
		}
		// Backoff for 5 attempts is 3.2 seconds, only 10ms has elapsed
		if retryLoop.isReadyForRetry(entry) {
			t.Error("Entry should not be ready when backoff hasn't elapsed")
		}
	})
}

// TestRetryLoop_ProcessEntry_Canceled tests context cancellation during processing.
func TestRetryLoop_ProcessEntry_Canceled(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	pub := &mockPublisher{}
	retryLoop := NewRetryLoop(wal, pub)

	ctx, cancel := context.WithCancel(context.Background())
	if err := retryLoop.Start(ctx); err != nil {
		t.Fatalf("Failed to start retry loop: %v", err)
	}

	// Write an entry
	event := createTestEvent("cancel-test")
	if _, err := wal.Write(ctx, event); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Cancel immediately
	cancel()

	// Wait for stop
	retryLoop.Stop()

	// Verify the loop stopped cleanly
	if retryLoop.IsRunning() {
		t.Error("RetryLoop should not be running after Stop")
	}
}

// TestRetryLoop_ConcurrentStartStop tests concurrent Start/Stop operations.
func TestRetryLoop_ConcurrentStartStop(t *testing.T) {
	wal := setupFastWAL(t)
	defer wal.Close()

	pub := &mockPublisher{}
	retryLoop := NewRetryLoop(wal, pub)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const goroutines = 10
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if err := retryLoop.Start(ctx); err != nil {
					t.Errorf("Start failed: %v", err)
				}
				time.Sleep(time.Millisecond)
				retryLoop.Stop()
			}
		}()
	}

	wg.Wait()
	// Should not panic or deadlock
}

// errorPublisher is a publisher that always returns an error.
type errorPublisher struct {
	errorMsg string
}

func (e *errorPublisher) PublishEntry(ctx context.Context, entry *Entry) error {
	return errors.New(e.errorMsg)
}

// TestRecovery_PublisherErrors tests recovery when publisher returns errors.
func TestRecovery_PublisherErrors(t *testing.T) {
	cfg := createTestConfig(t)
	cfg.MaxRetries = 5

	wal, err := Open(&cfg)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer wal.Close()

	ctx := context.Background()

	// Write entries
	for i := 0; i < 3; i++ {
		event := createTestEvent("error-test-" + string(rune('a'+i)))
		if _, err := wal.Write(ctx, event); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	// Use error publisher
	pub := &errorPublisher{errorMsg: "simulated publish failure"}

	result, err := wal.RecoverPending(ctx, pub)
	if err != nil {
		t.Fatalf("RecoverPending failed: %v", err)
	}

	// All entries should fail
	if result.Failed != 3 {
		t.Errorf("Expected 3 failed entries, got %d", result.Failed)
	}
	if result.Recovered != 0 {
		t.Errorf("Expected 0 recovered entries, got %d", result.Recovered)
	}

	// Verify attempt counts were updated
	entries, err := wal.GetPending(ctx)
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}
	for _, entry := range entries {
		if entry.Attempts != 1 {
			t.Errorf("Expected 1 attempt, got %d", entry.Attempts)
		}
		if entry.LastError != "simulated publish failure" {
			t.Errorf("Expected error message, got %s", entry.LastError)
		}
	}
}

// TestRecovery_NilPublisher tests that nil publisher is rejected.
func TestRecovery_NilPublisher(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	_, err := wal.RecoverPending(context.Background(), nil)
	if err == nil {
		t.Error("Expected error for nil publisher")
	}

	_, err = wal.RecoverPendingStream(context.Background(), nil)
	if err == nil {
		t.Error("Expected error for nil publisher in stream recovery")
	}
}

// TestRecovery_ContextCancellation tests that recovery stops on context cancellation.
func TestRecovery_ContextCancellation(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	ctx := context.Background()

	// Write many entries
	for i := 0; i < 10; i++ {
		event := createTestEvent("cancel-" + string(rune('a'+i)))
		if _, err := wal.Write(ctx, event); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	// Create a slow publisher and cancel context
	slowPub := PublisherFunc(func(ctx context.Context, entry *Entry) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := wal.RecoverPending(ctx, slowPub)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected deadline exceeded error, got %v", err)
	}

	// Some entries may have been processed before cancellation
	if result.TotalPending != 10 {
		t.Errorf("Expected 10 total pending, got %d", result.TotalPending)
	}
}

// TestRecovery_ExpiredEntries tests that expired entries are properly removed.
// We disable native TTL (EntryTTL=0) so entries never expire at the BadgerDB level,
// but then set a very short TTL for the recovery check to consider them expired.
func TestRecovery_ExpiredEntries(t *testing.T) {
	cfg := createFastTestConfig(t)
	// Start with no native TTL so entries persist
	cfg.EntryTTL = 0

	wal, err := OpenForTesting(&cfg)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer wal.Close()

	ctx := context.Background()

	// Write entry - no native TTL means it persists
	event := createTestEvent("expired-test")
	if _, err := wal.Write(ctx, event); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify entry was written
	entries, err := wal.GetPending(ctx)
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 pending entry, got %d", len(entries))
	}

	// Now change the WAL config to have a very short TTL for recovery check
	// This simulates entries that were written with a longer TTL but have since aged
	wal.config.EntryTTL = 1 * time.Nanosecond

	// Small delay to ensure time.Since(CreatedAt) > 1ns
	time.Sleep(1 * time.Millisecond)

	// Recovery will see the entry and mark it as expired
	// (time.Since(CreatedAt) > 1ns is true)
	pub := &mockPublisher{}
	result, err := wal.RecoverPending(ctx, pub)
	if err != nil {
		t.Fatalf("RecoverPending failed: %v", err)
	}

	// Entry should be expired (age > TTL)
	if result.Expired != 1 {
		t.Errorf("Expected 1 expired entry, got %d (TotalPending=%d)", result.Expired, result.TotalPending)
	}
	if result.Recovered != 0 {
		t.Errorf("Expected 0 recovered entries, got %d", result.Recovered)
	}

	// Publisher should not have been called
	if pub.publishCount.Load() != 0 {
		t.Errorf("Publisher should not be called for expired entries, got %d calls", pub.publishCount.Load())
	}
}

// TestRecovery_MaxRetriesExceeded tests that entries exceeding max retries are removed.
func TestRecovery_MaxRetriesExceeded(t *testing.T) {
	cfg := createTestConfig(t)
	cfg.MaxRetries = 2

	wal, err := Open(&cfg)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer wal.Close()

	ctx := context.Background()

	// Write entry
	event := createTestEvent("max-retry-test")
	entryID, err := wal.Write(ctx, event)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Simulate max retries reached
	if err := wal.UpdateAttempt(ctx, entryID, "error 1"); err != nil {
		t.Fatalf("UpdateAttempt failed: %v", err)
	}
	if err := wal.UpdateAttempt(ctx, entryID, "error 2"); err != nil {
		t.Fatalf("UpdateAttempt failed: %v", err)
	}

	pub := &mockPublisher{}
	result, err := wal.RecoverPending(ctx, pub)
	if err != nil {
		t.Fatalf("RecoverPending failed: %v", err)
	}

	// Entry should be removed for exceeding max retries
	if result.Failed != 1 {
		t.Errorf("Expected 1 failed entry (max retries), got %d", result.Failed)
	}
	if result.Recovered != 0 {
		t.Errorf("Expected 0 recovered entries, got %d", result.Recovered)
	}
}

// TestPublisherFunc tests the PublisherFunc adapter.
func TestPublisherFunc(t *testing.T) {
	var called bool
	pub := PublisherFunc(func(ctx context.Context, entry *Entry) error {
		called = true
		return nil
	})

	err := pub.PublishEntry(context.Background(), &Entry{ID: "test"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !called {
		t.Error("PublisherFunc was not called")
	}
}

// TestPublisherFunc_WithError tests PublisherFunc returning an error.
func TestPublisherFunc_WithError(t *testing.T) {
	expectedErr := errors.New("test error")
	pub := PublisherFunc(func(ctx context.Context, entry *Entry) error {
		return expectedErr
	})

	err := pub.PublishEntry(context.Background(), &Entry{ID: "test"})
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

// TestRetryStats_EmptyWAL tests RetryStats with no pending entries.
func TestRetryStats_EmptyWAL(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	pub := &mockPublisher{}
	retryLoop := NewRetryLoop(wal, pub)

	stats := retryLoop.GetStats()
	if stats.PendingCount != 0 {
		t.Errorf("Expected 0 pending, got %d", stats.PendingCount)
	}
	if stats.TotalAttempts != 0 {
		t.Errorf("Expected 0 total attempts, got %d", stats.TotalAttempts)
	}
	if stats.MaxAttempts != 0 {
		t.Errorf("Expected 0 max attempts, got %d", stats.MaxAttempts)
	}
	if !stats.OldestEntry.IsZero() {
		t.Errorf("Expected zero oldest entry, got %v", stats.OldestEntry)
	}
}

// TestRecoveryResult_Fields tests that RecoveryResult captures all information.
func TestRecoveryResult_Fields(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	ctx := context.Background()

	// Write entries
	for i := 0; i < 3; i++ {
		event := createTestEvent("result-" + string(rune('a'+i)))
		if _, err := wal.Write(ctx, event); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	pub := &mockPublisher{}
	result, err := wal.RecoverPending(ctx, pub)
	if err != nil {
		t.Fatalf("RecoverPending failed: %v", err)
	}

	if result.TotalPending != 3 {
		t.Errorf("Expected TotalPending=3, got %d", result.TotalPending)
	}
	if result.Recovered != 3 {
		t.Errorf("Expected Recovered=3, got %d", result.Recovered)
	}
	if result.Failed != 0 {
		t.Errorf("Expected Failed=0, got %d", result.Failed)
	}
	if result.Expired != 0 {
		t.Errorf("Expected Expired=0, got %d", result.Expired)
	}
	if result.Duration == 0 {
		t.Error("Duration should be non-zero")
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %d", len(result.Errors))
	}
}

// TestRetryLoop_LeaseHolder tests that each retry loop has a unique lease holder.
func TestRetryLoop_LeaseHolder(t *testing.T) {
	wal := setupWAL(t)
	defer wal.Close()

	pub := &mockPublisher{}

	// Create multiple retry loops
	loop1 := NewRetryLoop(wal, pub)
	loop2 := NewRetryLoop(wal, pub)

	// They should have different lease holders
	if loop1.leaseHolder == loop2.leaseHolder {
		t.Error("Each retry loop should have a unique lease holder")
	}

	// Lease holders should have expected prefix
	if len(loop1.leaseHolder) < 10 {
		t.Errorf("Lease holder too short: %s", loop1.leaseHolder)
	}
}
