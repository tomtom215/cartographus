// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/config"
)

// TestRateLimiting_HTTP429_ExponentialBackoff tests that the client retries
// with exponential backoff when receiving HTTP 429 responses
func TestRateLimiting_HTTP429_ExponentialBackoff(t *testing.T) {
	attemptCount := atomic.Int32{}
	startTime := time.Now()
	var attemptTimes []time.Time

	// Create mock server that returns 429 for first 2 attempts, then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attemptCount.Add(1)
		attemptTimes = append(attemptTimes, time.Now())

		if count <= 2 {
			// Return 429 for first 2 attempts
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"response":{"result":"error","message":"Rate limit exceeded"}}`))
			return
		}

		// Success on 3rd attempt
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"response":{"result":"success","message":null}}`))
	}))
	defer server.Close()

	// Create client
	cfg := &config.TautulliConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	client := NewTautulliClient(cfg)

	// Test Ping (which uses doRequestWithRateLimit)
	err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Expected ping to succeed after retries, got error: %v", err)
	}

	// Verify retry count
	finalCount := attemptCount.Load()
	if finalCount != 3 {
		t.Errorf("Expected 3 attempts (2 failures + 1 success), got %d", finalCount)
	}

	// Verify exponential backoff timing (1s, 2s)
	if len(attemptTimes) >= 3 {
		// First retry should wait ~1 second
		delay1 := attemptTimes[1].Sub(attemptTimes[0])
		if delay1 < 900*time.Millisecond || delay1 > 1200*time.Millisecond {
			t.Errorf("Expected first retry delay ~1s, got %v", delay1)
		}

		// Second retry should wait ~2 seconds
		delay2 := attemptTimes[2].Sub(attemptTimes[1])
		if delay2 < 1800*time.Millisecond || delay2 > 2300*time.Millisecond {
			t.Errorf("Expected second retry delay ~2s, got %v", delay2)
		}
	}

	// Verify total time is at least 3 seconds (1s + 2s)
	totalTime := time.Since(startTime)
	if totalTime < 2800*time.Millisecond {
		t.Errorf("Expected total time >= 3s with exponential backoff, got %v", totalTime)
	}
}

// TestRateLimiting_HTTP429_MaxRetriesExceeded tests that the client returns
// an error after exceeding max retries
func TestRateLimiting_HTTP429_MaxRetriesExceeded(t *testing.T) {
	attemptCount := atomic.Int32{}

	// Create mock server that always returns 429
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"response":{"result":"error","message":"Rate limit exceeded"}}`))
	}))
	defer server.Close()

	// Create client
	cfg := &config.TautulliConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	client := NewTautulliClient(cfg)

	// Test Ping (should fail after max retries)
	err := client.Ping(context.Background())
	if err == nil {
		t.Fatal("Expected error after exceeding max retries, got nil")
	}

	// Verify error message mentions rate limit
	expectedErrMsg := "rate limit exceeded after"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error containing '%s', got: %v", expectedErrMsg, err)
	}

	// Verify max retries were attempted (maxRetries=5, so 6 total attempts: 0,1,2,3,4,5)
	finalCount := attemptCount.Load()
	if finalCount != 6 {
		t.Errorf("Expected 6 attempts (maxRetries=5 + 1 initial), got %d", finalCount)
	}
}

// TestRateLimiting_HTTP429_RetryAfterHeader tests that the client respects
// the Retry-After header when provided
func TestRateLimiting_HTTP429_RetryAfterHeader(t *testing.T) {
	attemptCount := atomic.Int32{}
	var attemptTimes []time.Time

	// Create mock server that returns 429 with Retry-After header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attemptCount.Add(1)
		attemptTimes = append(attemptTimes, time.Now())

		if count == 1 {
			// First attempt: return 429 with Retry-After: 2 seconds
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"response":{"result":"error","message":"Rate limit exceeded"}}`))
			return
		}

		// Second attempt: success
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"response":{"result":"success","message":null}}`))
	}))
	defer server.Close()

	// Create client
	cfg := &config.TautulliConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	client := NewTautulliClient(cfg)

	// Test Ping
	err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Expected ping to succeed after retry, got error: %v", err)
	}

	// Verify 2 attempts
	finalCount := attemptCount.Load()
	if finalCount != 2 {
		t.Errorf("Expected 2 attempts, got %d", finalCount)
	}

	// Verify Retry-After header was respected (~2 seconds delay)
	if len(attemptTimes) >= 2 {
		delay := attemptTimes[1].Sub(attemptTimes[0])
		if delay < 1900*time.Millisecond || delay > 2200*time.Millisecond {
			t.Errorf("Expected retry delay ~2s (from Retry-After header), got %v", delay)
		}
	}
}

// TestRateLimiting_OtherHTTPErrors_NoRetry tests that non-429 HTTP errors
// are not retried (fail fast)
func TestRateLimiting_OtherHTTPErrors_NoRetry(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		errorMsg   string
	}{
		{"HTTP 500", http.StatusInternalServerError, "Internal Server Error"},
		{"HTTP 503", http.StatusServiceUnavailable, "Service Unavailable"},
		{"HTTP 404", http.StatusNotFound, "Not Found"},
		{"HTTP 401", http.StatusUnauthorized, "Unauthorized"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			attemptCount := atomic.Int32{}

			// Create mock server that returns error status
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attemptCount.Add(1)
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(fmt.Sprintf(`{"error":"%s"}`, tc.errorMsg)))
			}))
			defer server.Close()

			// Create client
			cfg := &config.TautulliConfig{
				URL:    server.URL,
				APIKey: "test-key",
			}
			client := NewTautulliClient(cfg)

			// Test Ping (should fail immediately, no retries)
			err := client.Ping(context.Background())
			if err == nil {
				t.Fatal("Expected error for non-200 status, got nil")
			}

			// Verify only 1 attempt (no retries for non-429 errors)
			finalCount := attemptCount.Load()
			if finalCount != 1 {
				t.Errorf("Expected 1 attempt (no retries for non-429), got %d", finalCount)
			}
		})
	}
}

// TestRateLimiting_SuccessOnFirstAttempt tests that successful requests
// complete immediately without retries
func TestRateLimiting_SuccessOnFirstAttempt(t *testing.T) {
	attemptCount := atomic.Int32{}

	// Create mock server that succeeds immediately
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount.Add(1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"response":{"result":"success","message":null}}`))
	}))
	defer server.Close()

	// Create client
	cfg := &config.TautulliConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	client := NewTautulliClient(cfg)

	// Test Ping
	err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Expected ping to succeed, got error: %v", err)
	}

	// Verify only 1 attempt (no retries needed)
	finalCount := attemptCount.Load()
	if finalCount != 1 {
		t.Errorf("Expected 1 attempt for successful request, got %d", finalCount)
	}
}

// TestRateLimiting_NetworkError_NoRetry tests that network errors are not retried
// (handled by outer retry logic in sync manager)
func TestRateLimiting_NetworkError_NoRetry(t *testing.T) {
	// Create client with invalid URL
	cfg := &config.TautulliConfig{
		URL:    "http://invalid-host-that-does-not-exist-12345.com",
		APIKey: "test-key",
	}
	client := NewTautulliClient(cfg)

	// Test Ping (should fail immediately with network error or connection error)
	err := client.Ping(context.Background())
	if err == nil {
		t.Fatal("Expected error from invalid host, got nil")
	}

	// Verify error contains either network error indicators or Tautulli error wrapper
	// (behavior varies by environment: DNS resolution may fail or succeed with error response)
	errMsg := err.Error()
	hasError := strings.Contains(errMsg, "failed to ping Tautulli") ||
		strings.Contains(errMsg, "Tautulli ping failed") ||
		strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "connection refused")

	if !hasError {
		t.Errorf("Expected error to indicate connection/network failure, got: %v", err)
	}
}
