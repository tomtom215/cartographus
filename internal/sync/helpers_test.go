// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
)

// TestRetryWithBackoff tests the retry mechanism with exponential backoff
func TestRetryWithBackoff(t *testing.T) {
	tests := []struct {
		name           string
		retryAttempts  int
		retryDelay     time.Duration
		failCount      int  // Number of times to fail before success
		expectedCalls  int  // Expected number of calls to the function
		expectError    bool // Whether we expect an error
		expectedErrMsg string
	}{
		{
			name:          "success on first try",
			retryAttempts: 3,
			retryDelay:    1 * time.Millisecond,
			failCount:     0,
			expectedCalls: 1,
			expectError:   false,
		},
		{
			name:          "success after one retry",
			retryAttempts: 3,
			retryDelay:    1 * time.Millisecond,
			failCount:     1,
			expectedCalls: 2,
			expectError:   false,
		},
		{
			name:          "success on last attempt",
			retryAttempts: 3,
			retryDelay:    1 * time.Millisecond,
			failCount:     2,
			expectedCalls: 3,
			expectError:   false,
		},
		{
			name:           "failure after all retries exhausted",
			retryAttempts:  3,
			retryDelay:     1 * time.Millisecond,
			failCount:      5, // More failures than retries
			expectedCalls:  3,
			expectError:    true,
			expectedErrMsg: "max retry attempts reached",
		},
		{
			name:          "single retry attempt succeeds first try",
			retryAttempts: 1,
			retryDelay:    1 * time.Millisecond,
			failCount:     0,
			expectedCalls: 1,
			expectError:   false,
		},
		{
			name:           "single retry attempt fails",
			retryAttempts:  1,
			retryDelay:     1 * time.Millisecond,
			failCount:      1,
			expectedCalls:  1,
			expectError:    true,
			expectedErrMsg: "max retry attempts reached",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create manager with test config
			cfg := &config.Config{
				Sync: config.SyncConfig{
					RetryAttempts: tt.retryAttempts,
					RetryDelay:    tt.retryDelay,
				},
			}
			m := &Manager{cfg: cfg}

			callCount := 0
			testErr := errors.New("transient error")

			err := m.retryWithBackoff(context.Background(), func() error {
				callCount++
				if callCount <= tt.failCount {
					return testErr
				}
				return nil
			})

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.expectedErrMsg != "" && !strings.Contains(err.Error(), tt.expectedErrMsg) {
					t.Errorf("error message mismatch: got %q, want substring %q", err.Error(), tt.expectedErrMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if callCount != tt.expectedCalls {
				t.Errorf("expected %d calls, got %d", tt.expectedCalls, callCount)
			}
		})
	}
}

// TestRetryWithBackoffExponentialDelay verifies exponential backoff timing
func TestRetryWithBackoffExponentialDelay(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			RetryAttempts: 3,
			RetryDelay:    10 * time.Millisecond,
		},
	}
	m := &Manager{cfg: cfg}

	callCount := 0
	callTimes := make([]time.Time, 0)

	start := time.Now()
	_ = m.retryWithBackoff(context.Background(), func() error {
		callTimes = append(callTimes, time.Now())
		callCount++
		if callCount < 3 {
			return errors.New("fail")
		}
		return nil
	})

	// Verify we had 3 calls
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}

	// Verify delays were approximately exponential
	// First delay should be ~10ms, second should be ~20ms
	// Note: CI environments can have significant scheduling delays, so we use generous bounds
	if len(callTimes) >= 2 {
		firstDelay := callTimes[1].Sub(callTimes[0])
		if firstDelay < 8*time.Millisecond || firstDelay > 500*time.Millisecond {
			t.Errorf("first delay out of range: %v (expected 8ms-500ms)", firstDelay)
		}
	}

	totalDuration := time.Since(start)
	// Total time should be reasonable (at least 10ms + 20ms = 30ms for retries)
	if totalDuration < 20*time.Millisecond {
		t.Errorf("total duration too short: %v (expected at least 20ms)", totalDuration)
	}
}

// TestRetryWithBackoffContextCancellation verifies that retryWithBackoff respects context cancellation
func TestRetryWithBackoffContextCancellation(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			RetryAttempts: 10,                     // Many attempts
			RetryDelay:    100 * time.Millisecond, // Long delay
		},
	}
	m := &Manager{cfg: cfg}

	// Create a context that we'll cancel during the backoff wait
	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	startTime := time.Now()

	// Start retryWithBackoff in a goroutine so we can cancel the context
	errChan := make(chan error, 1)
	go func() {
		err := m.retryWithBackoff(ctx, func() error {
			callCount++
			// Cancel context after first call to trigger cancellation during backoff
			if callCount == 1 {
				cancel()
			}
			return errors.New("always fail")
		})
		errChan <- err
	}()

	// Wait for the result
	err := <-errChan
	elapsed := time.Since(startTime)

	// Should have returned quickly due to context cancellation
	if elapsed > 50*time.Millisecond {
		t.Errorf("took too long: %v (expected quick return on context cancel)", elapsed)
	}

	// Should have received context.Canceled error
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}

	// Should have only made 1 call before cancellation took effect
	if callCount != 1 {
		t.Errorf("expected 1 call before cancellation, got %d", callCount)
	}
}

// TestRetryWithBackoffContextAlreadyCanceled verifies behavior when context is already canceled
func TestRetryWithBackoffContextAlreadyCanceled(t *testing.T) {
	cfg := &config.Config{
		Sync: config.SyncConfig{
			RetryAttempts: 3,
			RetryDelay:    100 * time.Millisecond,
		},
	}
	m := &Manager{cfg: cfg}

	// Create an already-canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	callCount := 0
	err := m.retryWithBackoff(ctx, func() error {
		callCount++
		return nil
	})

	// Should return immediately with context.Canceled
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}

	// Should not have called the function at all
	if callCount != 0 {
		t.Errorf("expected 0 calls with canceled context, got %d", callCount)
	}
}

// TestStringToPtr tests the stringToPtr helper function
func TestStringToPtr(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		wantVal string
	}{
		{
			name:    "empty string returns nil",
			input:   "",
			wantNil: true,
		},
		{
			name:    "non-empty string returns pointer",
			input:   "hello",
			wantNil: false,
			wantVal: "hello",
		},
		{
			name:    "whitespace string returns pointer",
			input:   "   ",
			wantNil: false,
			wantVal: "   ",
		},
		{
			name:    "special characters",
			input:   "hello\nworld\t!",
			wantNil: false,
			wantVal: "hello\nworld\t!",
		},
		{
			name:    "unicode string",
			input:   "Hello, 世界!",
			wantNil: false,
			wantVal: "Hello, 世界!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringToPtr(tt.input)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %q", *result)
				}
			} else {
				if result == nil {
					t.Error("expected non-nil pointer, got nil")
				} else if *result != tt.wantVal {
					t.Errorf("expected %q, got %q", tt.wantVal, *result)
				}
			}
		})
	}
}

// TestIntToPtr tests the intToPtr helper function
func TestIntToPtr(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		wantNil bool
		wantVal int
	}{
		{
			name:    "zero returns nil",
			input:   0,
			wantNil: true,
		},
		{
			name:    "negative returns nil",
			input:   -1,
			wantNil: true,
		},
		{
			name:    "negative large returns nil",
			input:   -1000000,
			wantNil: true,
		},
		{
			name:    "positive returns pointer",
			input:   42,
			wantNil: false,
			wantVal: 42,
		},
		{
			name:    "one returns pointer",
			input:   1,
			wantNil: false,
			wantVal: 1,
		},
		{
			name:    "large positive returns pointer",
			input:   999999999,
			wantNil: false,
			wantVal: 999999999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := intToPtr(tt.input)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %d", *result)
				}
			} else {
				if result == nil {
					t.Error("expected non-nil pointer, got nil")
				} else if *result != tt.wantVal {
					t.Errorf("expected %d, got %d", tt.wantVal, *result)
				}
			}
		})
	}
}

// TestMapStringField tests the mapStringField helper function
func TestMapStringField(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		expectNil   bool
		expectedVal string
	}{
		{
			name:      "empty string does not set target",
			value:     "",
			expectNil: true,
		},
		{
			name:      "N/A does not set target",
			value:     "N/A",
			expectNil: true,
		},
		{
			name:        "valid string sets target",
			value:       "test value",
			expectNil:   false,
			expectedVal: "test value",
		},
		{
			name:        "whitespace string sets target",
			value:       "  ",
			expectNil:   false,
			expectedVal: "  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target *string
			mapStringField(tt.value, &target)

			if tt.expectNil {
				if target != nil {
					t.Errorf("expected nil, got %q", *target)
				}
			} else {
				if target == nil {
					t.Error("expected non-nil pointer, got nil")
				} else if *target != tt.expectedVal {
					t.Errorf("expected %q, got %q", tt.expectedVal, *target)
				}
			}
		})
	}
}

// TestMapIntField tests the mapIntField helper function
func TestMapIntField(t *testing.T) {
	tests := []struct {
		name        string
		value       int
		expectNil   bool
		expectedVal int
	}{
		{
			name:      "zero does not set target",
			value:     0,
			expectNil: true,
		},
		{
			name:      "negative does not set target",
			value:     -5,
			expectNil: true,
		},
		{
			name:        "positive sets target",
			value:       100,
			expectNil:   false,
			expectedVal: 100,
		},
		{
			name:        "one sets target",
			value:       1,
			expectNil:   false,
			expectedVal: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target *int
			mapIntField(tt.value, &target)

			if tt.expectNil {
				if target != nil {
					t.Errorf("expected nil, got %d", *target)
				}
			} else {
				if target == nil {
					t.Error("expected non-nil pointer, got nil")
				} else if *target != tt.expectedVal {
					t.Errorf("expected %d, got %d", tt.expectedVal, *target)
				}
			}
		})
	}
}

// TestMapInt64Field tests the mapInt64Field helper function
func TestMapInt64Field(t *testing.T) {
	tests := []struct {
		name        string
		value       int64
		expectNil   bool
		expectedVal int64
	}{
		{
			name:      "zero does not set target",
			value:     0,
			expectNil: true,
		},
		{
			name:      "negative does not set target",
			value:     -5,
			expectNil: true,
		},
		{
			name:        "positive sets target",
			value:       1000000000,
			expectNil:   false,
			expectedVal: 1000000000,
		},
		{
			name:        "large value sets target",
			value:       9223372036854775807, // Max int64
			expectNil:   false,
			expectedVal: 9223372036854775807,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target *int64
			mapInt64Field(tt.value, &target)

			if tt.expectNil {
				if target != nil {
					t.Errorf("expected nil, got %d", *target)
				}
			} else {
				if target == nil {
					t.Error("expected non-nil pointer, got nil")
				} else if *target != tt.expectedVal {
					t.Errorf("expected %d, got %d", tt.expectedVal, *target)
				}
			}
		})
	}
}

// TestMapSignedIntField tests the mapSignedIntField helper function
func TestMapSignedIntField(t *testing.T) {
	tests := []struct {
		name        string
		value       int
		expectNil   bool
		expectedVal int
	}{
		{
			name:        "zero sets target (unlike mapIntField)",
			value:       0,
			expectNil:   false,
			expectedVal: 0,
		},
		{
			name:      "negative does not set target",
			value:     -1,
			expectNil: true,
		},
		{
			name:        "positive sets target",
			value:       100,
			expectNil:   false,
			expectedVal: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target *int
			mapSignedIntField(tt.value, &target)

			if tt.expectNil {
				if target != nil {
					t.Errorf("expected nil, got %d", *target)
				}
			} else {
				if target == nil {
					t.Error("expected non-nil pointer, got nil")
				} else if *target != tt.expectedVal {
					t.Errorf("expected %d, got %d", tt.expectedVal, *target)
				}
			}
		})
	}
}

// TestGetUsername tests the getUsername helper function
func TestGetUsername(t *testing.T) {
	tests := []struct {
		name     string
		user     *models.PlexSessionUser
		expected string
	}{
		{
			name:     "nil user returns Unknown",
			user:     nil,
			expected: "Unknown",
		},
		{
			name:     "user with title returns title",
			user:     &models.PlexSessionUser{Title: "TestUser"},
			expected: "TestUser",
		},
		{
			name:     "user with empty title returns empty string",
			user:     &models.PlexSessionUser{Title: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUsername(tt.user)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestGetPlayerName tests the getPlayerName helper function
func TestGetPlayerName(t *testing.T) {
	tests := []struct {
		name     string
		player   *models.PlexSessionPlayer
		expected string
	}{
		{
			name:     "nil player returns Unknown",
			player:   nil,
			expected: "Unknown",
		},
		{
			name:     "player with title returns title",
			player:   &models.PlexSessionPlayer{Title: "Living Room TV", Product: "Plex for Smart TVs"},
			expected: "Living Room TV",
		},
		{
			name:     "player with empty title returns product",
			player:   &models.PlexSessionPlayer{Title: "", Product: "Plex for iOS"},
			expected: "Plex for iOS",
		},
		{
			name:     "player with both empty returns empty",
			player:   &models.PlexSessionPlayer{Title: "", Product: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPlayerName(tt.player)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestNewTestConfig verifies test configuration helper
func TestNewTestConfig(t *testing.T) {
	cfg := newTestConfig()

	// Verify test configuration values
	if cfg.Sync.BatchSize != 10 {
		t.Errorf("expected BatchSize 10, got %d", cfg.Sync.BatchSize)
	}

	if cfg.Sync.RetryAttempts != 2 {
		t.Errorf("expected RetryAttempts 2, got %d", cfg.Sync.RetryAttempts)
	}

	if cfg.Sync.RetryDelay != 10*time.Millisecond {
		t.Errorf("expected RetryDelay 10ms, got %v", cfg.Sync.RetryDelay)
	}

	if cfg.Security.RateLimitDisabled != true {
		t.Error("expected RateLimitDisabled to be true")
	}
}

// TestNewTestConfigWithRetries verifies retry configuration customization
func TestNewTestConfigWithRetries(t *testing.T) {
	cfg := newTestConfigWithRetries(5, 50*time.Millisecond)

	if cfg.Sync.RetryAttempts != 5 {
		t.Errorf("expected RetryAttempts 5, got %d", cfg.Sync.RetryAttempts)
	}

	if cfg.Sync.RetryDelay != 50*time.Millisecond {
		t.Errorf("expected RetryDelay 50ms, got %v", cfg.Sync.RetryDelay)
	}

	// Verify other values are still set from base config
	if cfg.Sync.BatchSize != 10 {
		t.Errorf("expected BatchSize 10, got %d", cfg.Sync.BatchSize)
	}
}
