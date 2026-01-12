// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// TestDLQEntry_Creation tests DLQ entry creation with proper fields.
func TestDLQEntry_Creation(t *testing.T) {
	t.Parallel()

	event := NewMediaEvent(SourcePlex)
	event.UserID = 1
	event.Username = "testuser"
	event.MediaType = MediaTypeMovie
	event.Title = "Test Movie"

	originalErr := errors.New("database connection failed")
	entry := NewDLQEntry(event, originalErr, "test-message-id")

	if entry.Event == nil {
		t.Fatal("Entry.Event should not be nil")
	}
	if entry.Event.EventID != event.EventID {
		t.Errorf("Entry.Event.EventID = %s, want %s", entry.Event.EventID, event.EventID)
	}
	if entry.OriginalError != originalErr.Error() {
		t.Errorf("Entry.OriginalError = %s, want %s", entry.OriginalError, originalErr.Error())
	}
	if entry.MessageID != "test-message-id" {
		t.Errorf("Entry.MessageID = %s, want test-message-id", entry.MessageID)
	}
	if entry.RetryCount != 0 {
		t.Errorf("Entry.RetryCount = %d, want 0", entry.RetryCount)
	}
	if entry.FirstFailure.IsZero() {
		t.Error("Entry.FirstFailure should be set")
	}
	if entry.LastFailure.IsZero() {
		t.Error("Entry.LastFailure should be set")
	}
}

// TestRetryableError_Identification tests error type identification.
func TestRetryableError_Identification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		err         error
		isRetryable bool
		isPermanent bool
	}{
		{
			name:        "retryable error",
			err:         NewRetryableError("connection timeout", nil),
			isRetryable: true,
			isPermanent: false,
		},
		{
			name:        "permanent error",
			err:         NewPermanentError("invalid JSON format", nil),
			isRetryable: false,
			isPermanent: true,
		},
		{
			name:        "wrapped retryable",
			err:         NewRetryableError("db error", errors.New("connection refused")),
			isRetryable: true,
			isPermanent: false,
		},
		{
			name:        "regular error",
			err:         errors.New("some error"),
			isRetryable: false,
			isPermanent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryableError(tt.err); got != tt.isRetryable {
				t.Errorf("IsRetryableError() = %v, want %v", got, tt.isRetryable)
			}
			if got := IsPermanentError(tt.err); got != tt.isPermanent {
				t.Errorf("IsPermanentError() = %v, want %v", got, tt.isPermanent)
			}
		})
	}
}

// TestDLQHandler_NewDLQHandler tests handler creation with config validation.
func TestDLQHandler_NewDLQHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     DLQConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     DefaultDLQConfig(),
			wantErr: false,
		},
		{
			name: "zero max retries",
			cfg: DLQConfig{
				MaxRetries:     0,
				MaxEntries:     1000,
				RetentionTime:  24 * time.Hour,
				InitialBackoff: time.Second,
				MaxBackoff:     time.Minute,
			},
			wantErr: true,
		},
		{
			name: "zero max entries",
			cfg: DLQConfig{
				MaxRetries:     3,
				MaxEntries:     0,
				RetentionTime:  24 * time.Hour,
				InitialBackoff: time.Second,
				MaxBackoff:     time.Minute,
			},
			wantErr: true,
		},
		{
			name: "zero initial backoff",
			cfg: DLQConfig{
				MaxRetries:     3,
				MaxEntries:     1000,
				RetentionTime:  24 * time.Hour,
				InitialBackoff: 0,
				MaxBackoff:     time.Minute,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewDLQHandler(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDLQHandler() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && handler == nil {
				t.Error("NewDLQHandler() returned nil handler for valid config")
			}
		})
	}
}

// TestDLQHandler_AddEntry tests adding entries to DLQ.
func TestDLQHandler_AddEntry(t *testing.T) {
	t.Parallel()

	cfg := DefaultDLQConfig()
	handler, err := NewDLQHandler(cfg)
	if err != nil {
		t.Fatalf("NewDLQHandler() error = %v", err)
	}

	event := NewMediaEvent(SourcePlex)
	event.UserID = 1
	event.Username = "user1"
	event.MediaType = MediaTypeMovie
	event.Title = "Movie 1"

	entry := handler.AddEntry(event, errors.New("test error"), "msg-1")

	if entry == nil {
		t.Fatal("AddEntry() returned nil")
	}

	stats := handler.Stats()
	if stats.TotalEntries != 1 {
		t.Errorf("Stats().TotalEntries = %d, want 1", stats.TotalEntries)
	}
}

// TestDLQHandler_GetEntry tests retrieving entries.
func TestDLQHandler_GetEntry(t *testing.T) {
	t.Parallel()

	cfg := DefaultDLQConfig()
	handler, _ := NewDLQHandler(cfg)

	event := NewMediaEvent(SourcePlex)
	event.UserID = 1
	event.Username = "user1"
	event.MediaType = MediaTypeMovie
	event.Title = "Movie 1"

	handler.AddEntry(event, errors.New("test error"), "msg-1")

	// Get existing entry
	entry := handler.GetEntry(event.EventID)
	if entry == nil {
		t.Fatal("GetEntry() returned nil for existing entry")
	}
	if entry.Event.EventID != event.EventID {
		t.Errorf("GetEntry().Event.EventID = %s, want %s", entry.Event.EventID, event.EventID)
	}

	// Get non-existing entry
	nonExistent := handler.GetEntry("non-existent")
	if nonExistent != nil {
		t.Error("GetEntry() should return nil for non-existent entry")
	}
}

// TestDLQHandler_IncrementRetry tests retry count increment.
func TestDLQHandler_IncrementRetry(t *testing.T) {
	t.Parallel()

	cfg := DefaultDLQConfig()
	cfg.MaxRetries = 3
	handler, _ := NewDLQHandler(cfg)

	event := NewMediaEvent(SourcePlex)
	event.UserID = 1
	event.Username = "user1"
	event.MediaType = MediaTypeMovie
	event.Title = "Movie 1"

	handler.AddEntry(event, errors.New("test error"), "msg-1")

	// First increment
	canRetry := handler.IncrementRetry(event.EventID, errors.New("retry error 1"))
	if !canRetry {
		t.Error("IncrementRetry() should return true when under max retries")
	}

	entry := handler.GetEntry(event.EventID)
	if entry.RetryCount != 1 {
		t.Errorf("RetryCount = %d, want 1", entry.RetryCount)
	}

	// Second increment
	canRetry = handler.IncrementRetry(event.EventID, errors.New("retry error 2"))
	if !canRetry {
		t.Error("IncrementRetry() should return true when under max retries")
	}

	// Third increment (reaches max)
	canRetry = handler.IncrementRetry(event.EventID, errors.New("retry error 3"))
	if canRetry {
		t.Error("IncrementRetry() should return false when max retries reached")
	}
}

// TestDLQHandler_RemoveEntry tests removing entries from DLQ.
func TestDLQHandler_RemoveEntry(t *testing.T) {
	t.Parallel()

	cfg := DefaultDLQConfig()
	handler, _ := NewDLQHandler(cfg)

	event := NewMediaEvent(SourcePlex)
	event.UserID = 1
	event.Username = "user1"
	event.MediaType = MediaTypeMovie
	event.Title = "Movie 1"

	handler.AddEntry(event, errors.New("test error"), "msg-1")

	// Remove existing entry
	removed := handler.RemoveEntry(event.EventID)
	if !removed {
		t.Error("RemoveEntry() should return true for existing entry")
	}

	stats := handler.Stats()
	if stats.TotalEntries != 0 {
		t.Errorf("Stats().TotalEntries = %d, want 0 after removal", stats.TotalEntries)
	}

	// Remove non-existing entry
	removed = handler.RemoveEntry("non-existent")
	if removed {
		t.Error("RemoveEntry() should return false for non-existent entry")
	}
}

// TestDLQHandler_MaxEntries tests DLQ entry limit enforcement.
func TestDLQHandler_MaxEntries(t *testing.T) {
	t.Parallel()

	cfg := DefaultDLQConfig()
	cfg.MaxEntries = 3
	handler, _ := NewDLQHandler(cfg)

	// Add more entries than max
	for i := 0; i < 5; i++ {
		event := NewMediaEvent(SourcePlex)
		event.UserID = i + 1
		event.Username = "user"
		event.MediaType = MediaTypeMovie
		event.Title = "Movie"
		handler.AddEntry(event, errors.New("test error"), "msg-"+string(rune('0'+i)))
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	stats := handler.Stats()
	if stats.TotalEntries > int64(cfg.MaxEntries) {
		t.Errorf("Stats().TotalEntries = %d, want <= %d (max)", stats.TotalEntries, cfg.MaxEntries)
	}
}

// TestRetryPolicy_CalculateBackoff tests exponential backoff calculation.
func TestRetryPolicy_CalculateBackoff(t *testing.T) {
	t.Parallel()

	policy := DefaultRetryPolicy()

	tests := []struct {
		retryCount int
		minBackoff time.Duration
		maxBackoff time.Duration
	}{
		{0, time.Second, 2 * time.Second},     // 1s * 2^0 = 1s (with jitter up to 2s)
		{1, 2 * time.Second, 4 * time.Second}, // 1s * 2^1 = 2s (with jitter up to 4s)
		{2, 4 * time.Second, 8 * time.Second}, // 1s * 2^2 = 4s (with jitter up to 8s)
		{10, time.Minute, 2 * time.Minute},    // Capped at MaxBackoff (1m)
	}

	for _, tt := range tests {
		t.Run("retry_"+string(rune('0'+tt.retryCount)), func(t *testing.T) {
			backoff := policy.CalculateBackoff(tt.retryCount)
			if backoff < tt.minBackoff/2 { // Allow for jitter
				t.Errorf("CalculateBackoff(%d) = %v, want >= %v", tt.retryCount, backoff, tt.minBackoff/2)
			}
			if backoff > tt.maxBackoff {
				t.Errorf("CalculateBackoff(%d) = %v, want <= %v", tt.retryCount, backoff, tt.maxBackoff)
			}
		})
	}
}

// TestRetryPolicy_ShouldRetry tests retry decision logic.
func TestRetryPolicy_ShouldRetry(t *testing.T) {
	t.Parallel()

	policy := DefaultRetryPolicy()

	tests := []struct {
		name       string
		err        error
		retryCount int
		want       bool
	}{
		{"retryable under limit", NewRetryableError("timeout", nil), 0, true},
		{"retryable at limit", NewRetryableError("timeout", nil), policy.MaxRetries, false},
		{"permanent error", NewPermanentError("invalid", nil), 0, false},
		{"regular error under limit", errors.New("unknown"), 0, true},
		{"regular error at limit", errors.New("unknown"), policy.MaxRetries, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := policy.ShouldRetry(tt.err, tt.retryCount); got != tt.want {
				t.Errorf("ShouldRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDLQHandler_GetPendingRetries tests retrieval of entries ready for retry.
func TestDLQHandler_GetPendingRetries(t *testing.T) {
	t.Parallel()

	cfg := DefaultDLQConfig()
	cfg.InitialBackoff = 10 * time.Millisecond
	handler, _ := NewDLQHandler(cfg)

	// Add an entry
	event := NewMediaEvent(SourcePlex)
	event.UserID = 1
	event.Username = "user1"
	event.MediaType = MediaTypeMovie
	event.Title = "Movie 1"

	handler.AddEntry(event, errors.New("test error"), "msg-1")

	// Immediately after add, entry should not be ready (backoff not elapsed)
	pending := handler.GetPendingRetries()
	if len(pending) != 0 {
		t.Errorf("GetPendingRetries() = %d entries, want 0 (backoff not elapsed)", len(pending))
	}

	// Wait for backoff to elapse
	time.Sleep(20 * time.Millisecond)

	pending = handler.GetPendingRetries()
	if len(pending) != 1 {
		t.Errorf("GetPendingRetries() = %d entries, want 1", len(pending))
	}
}

// TestDLQHandler_Stats tests statistics collection.
func TestDLQHandler_Stats(t *testing.T) {
	t.Parallel()

	cfg := DefaultDLQConfig()
	handler, _ := NewDLQHandler(cfg)

	// Add entries
	for i := 0; i < 3; i++ {
		event := NewMediaEvent(SourcePlex)
		event.UserID = i + 1
		event.Username = "user"
		event.MediaType = MediaTypeMovie
		event.Title = "Movie"
		handler.AddEntry(event, errors.New("test error"), "msg-"+string(rune('0'+i)))
	}

	// Increment retry on one
	event := NewMediaEvent(SourcePlex)
	event.EventID = handler.ListEntries()[0].Event.EventID
	handler.IncrementRetry(event.EventID, errors.New("retry"))

	// Remove one
	handler.RemoveEntry(handler.ListEntries()[0].Event.EventID)

	stats := handler.Stats()
	if stats.TotalEntries != 2 {
		t.Errorf("Stats().TotalEntries = %d, want 2", stats.TotalEntries)
	}
	if stats.TotalAdded != 3 {
		t.Errorf("Stats().TotalAdded = %d, want 3", stats.TotalAdded)
	}
	if stats.TotalRemoved != 1 {
		t.Errorf("Stats().TotalRemoved = %d, want 1", stats.TotalRemoved)
	}
	if stats.TotalRetries != 1 {
		t.Errorf("Stats().TotalRetries = %d, want 1", stats.TotalRetries)
	}
}

// TestDLQHandler_ConcurrentAccess tests thread safety.
func TestDLQHandler_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	cfg := DefaultDLQConfig()
	cfg.MaxEntries = 1000
	handler, _ := NewDLQHandler(cfg)

	const numGoroutines = 10
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				event := NewMediaEvent(SourcePlex)
				event.UserID = goroutineID*1000 + i
				event.Username = "user"
				event.MediaType = MediaTypeMovie
				event.Title = "Movie"

				// Add entry
				handler.AddEntry(event, errors.New("test"), "msg")

				// Try to get it
				_ = handler.GetEntry(event.EventID)

				// Maybe increment retry
				if i%2 == 0 {
					handler.IncrementRetry(event.EventID, errors.New("retry"))
				}

				// Maybe remove it
				if i%3 == 0 {
					handler.RemoveEntry(event.EventID)
				}
			}
		}(g)
	}

	wg.Wait()

	// Should not panic or have data races
	stats := handler.Stats()
	t.Logf("Final stats: entries=%d, added=%d, removed=%d, retries=%d",
		stats.TotalEntries, stats.TotalAdded, stats.TotalRemoved, stats.TotalRetries)
}

// TestDLQHandler_Cleanup tests expired entry cleanup.
func TestDLQHandler_Cleanup(t *testing.T) {
	t.Parallel()

	cfg := DefaultDLQConfig()
	cfg.RetentionTime = 50 * time.Millisecond
	handler, _ := NewDLQHandler(cfg)

	// Add entry
	event := NewMediaEvent(SourcePlex)
	event.UserID = 1
	event.Username = "user1"
	event.MediaType = MediaTypeMovie
	event.Title = "Movie 1"
	handler.AddEntry(event, errors.New("test error"), "msg-1")

	// Entry should exist
	if handler.GetEntry(event.EventID) == nil {
		t.Fatal("Entry should exist before cleanup")
	}

	// Wait for retention to expire
	time.Sleep(100 * time.Millisecond)

	// Run cleanup
	cleaned := handler.Cleanup()
	if cleaned != 1 {
		t.Errorf("Cleanup() = %d, want 1", cleaned)
	}

	// Entry should be gone
	if handler.GetEntry(event.EventID) != nil {
		t.Error("Entry should be removed after cleanup")
	}
}

// TestDLQConfig_Defaults tests default configuration values.
func TestDLQConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg := DefaultDLQConfig()

	if cfg.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", cfg.MaxRetries)
	}
	if cfg.MaxEntries != 10000 {
		t.Errorf("MaxEntries = %d, want 10000", cfg.MaxEntries)
	}
	if cfg.RetentionTime != 7*24*time.Hour {
		t.Errorf("RetentionTime = %v, want 7d", cfg.RetentionTime)
	}
	if cfg.InitialBackoff != time.Second {
		t.Errorf("InitialBackoff = %v, want 1s", cfg.InitialBackoff)
	}
	if cfg.MaxBackoff != time.Minute {
		t.Errorf("MaxBackoff = %v, want 1m", cfg.MaxBackoff)
	}
}

// TestErrorCategories tests error categorization for DLQ routing.
func TestErrorCategories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		category ErrorCategory
	}{
		{"connection error", NewRetryableError("connection refused", nil), ErrorCategoryConnection},
		{"timeout error", NewRetryableError("timeout", nil), ErrorCategoryTimeout},
		{"validation error", NewPermanentError("invalid field", nil), ErrorCategoryValidation},
		{"unknown error", errors.New("something went wrong"), ErrorCategoryUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var category ErrorCategory
			var retryErr *RetryableError
			var permErr *PermanentError

			if errors.As(tt.err, &retryErr) {
				category = retryErr.Category
			} else if errors.As(tt.err, &permErr) {
				category = permErr.Category
			} else {
				category = ErrorCategoryUnknown
			}

			if category != tt.category {
				t.Errorf("error category = %v, want %v", category, tt.category)
			}
		})
	}
}
