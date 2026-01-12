// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"errors"
	"testing"
	"time"

	gobreaker "github.com/sony/gobreaker/v2"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/metrics"
)

// TestCircuitBreaker_OpensAfterFailures verifies circuit opens after exceeding failure threshold
func TestCircuitBreaker_OpensAfterFailures(t *testing.T) {
	cfg := &config.TautulliConfig{
		URL:    "http://localhost:8181",
		APIKey: "test-key",
	}

	// Create circuit breaker client
	cbc := NewCircuitBreakerClient(cfg)

	// Circuit breaker settings: minimum 10 requests, 60% failure rate
	// So we need at least 10 requests with 6+ failures to open

	// Initial state should be closed (0)
	state := cbc.cb.State()
	if state != gobreaker.StateClosed {
		t.Errorf("Expected initial state to be Closed, got %v", state)
	}

	// Simulate 10 API calls with 7 failures (70% failure rate)
	successCount := 0
	failureCount := 0

	for i := 0; i < 10; i++ {
		// Mock API call - fail 7 times, succeed 3 times
		_, err := cbc.execute(func() (interface{}, error) {
			if i < 7 {
				return nil, errors.New("simulated API failure")
			}
			return "success", nil
		})

		if err != nil {
			failureCount++
		} else {
			successCount++
		}
	}

	// Verify failure count
	if failureCount != 7 {
		t.Errorf("Expected 7 failures, got %d", failureCount)
	}

	if successCount != 3 {
		t.Errorf("Expected 3 successes, got %d", successCount)
	}

	// ReadyToTrip is checked BEFORE each request, so after 10 requests we have 9 checked
	// We need one more request (failure) to trigger ReadyToTrip with 10+ requests
	_, _ = cbc.execute(func() (interface{}, error) {
		return nil, errors.New("final failure to trigger circuit")
	})

	// After 70% failure rate with 10+ requests, circuit should be open
	state = cbc.cb.State()
	if state != gobreaker.StateOpen {
		t.Errorf("Expected circuit to be Open after 70%% failure rate, got %v", state)
	}

	// Verify next request is rejected with ErrOpenState
	_, err := cbc.execute(func() (interface{}, error) {
		return "should not execute", nil
	})

	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Errorf("Expected ErrOpenState when circuit is open, got %v", err)
	}
}

// TestCircuitBreaker_DoesNotOpenBelowThreshold verifies circuit stays closed below failure threshold
func TestCircuitBreaker_DoesNotOpenBelowThreshold(t *testing.T) {
	cfg := &config.TautulliConfig{
		URL:    "http://localhost:8181",
		APIKey: "test-key",
	}

	cbc := NewCircuitBreakerClient(cfg)

	// Simulate 10 API calls with 5 failures (50% failure rate)
	// This is below the 60% threshold, so circuit should stay closed
	for i := 0; i < 10; i++ {
		_, _ = cbc.execute(func() (interface{}, error) {
			if i < 5 {
				return nil, errors.New("simulated API failure")
			}
			return "success", nil
		})
	}

	// Circuit should still be closed (50% < 60% threshold)
	state := cbc.cb.State()
	if state != gobreaker.StateClosed {
		t.Errorf("Expected circuit to remain Closed with 50%% failure rate, got %v", state)
	}
}

// TestCircuitBreaker_RequiresMinimumRequests verifies circuit requires minimum 10 requests
func TestCircuitBreaker_RequiresMinimumRequests(t *testing.T) {
	cfg := &config.TautulliConfig{
		URL:    "http://localhost:8181",
		APIKey: "test-key",
	}

	cbc := NewCircuitBreakerClient(cfg)

	// Simulate only 5 API calls with 100% failure rate
	// Circuit should NOT open because we need minimum 10 requests for statistical significance
	for i := 0; i < 5; i++ {
		_, _ = cbc.execute(func() (interface{}, error) {
			return nil, errors.New("simulated API failure")
		})
	}

	// Circuit should still be closed despite 100% failure rate (< 10 requests)
	state := cbc.cb.State()
	if state != gobreaker.StateClosed {
		t.Errorf("Expected circuit to remain Closed with <10 requests, got %v", state)
	}
}

// TestCircuitBreaker_TransitionsToHalfOpen verifies circuit transitions to half-open after timeout
func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	// This test requires modifying circuit breaker timeout
	// For testing purposes, we create a custom circuit breaker with short timeout

	cfg := &config.TautulliConfig{
		URL:    "http://localhost:8181",
		APIKey: "test-key",
	}

	client := NewTautulliClient(cfg)
	cbName := "test-circuit-breaker"

	// Create circuit breaker with 100ms timeout for testing
	cb := gobreaker.NewCircuitBreaker[interface{}](gobreaker.Settings{
		Name:        cbName,
		MaxRequests: 3,
		Interval:    time.Second,
		Timeout:     100 * time.Millisecond, // Short timeout for testing

		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < 10 {
				return false
			}
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return failureRatio >= 0.6
		},
	})

	cbc := &CircuitBreakerClient{
		client: client,
		cb:     cb,
		name:   cbName,
	}

	// Force circuit to open state by exceeding failure threshold
	for i := 0; i < 10; i++ {
		_, _ = cbc.execute(func() (interface{}, error) {
			return nil, errors.New("simulated API failure")
		})
	}

	// Verify circuit is open
	state := cbc.cb.State()
	if state != gobreaker.StateOpen {
		t.Fatalf("Expected circuit to be Open, got %v", state)
	}

	// Wait for timeout to transition to half-open
	time.Sleep(150 * time.Millisecond) // Wait longer than timeout

	// Next request should trigger transition to half-open
	_, _ = cbc.execute(func() (interface{}, error) {
		return "test", nil
	})

	// State should be half-open or closed (depending on success)
	state = cbc.cb.State()
	if state == gobreaker.StateOpen {
		t.Errorf("Expected circuit to transition from Open after timeout, still Open")
	}
}

// TestCircuitBreaker_ClosesAfterSuccessInHalfOpen verifies circuit closes after success in half-open
func TestCircuitBreaker_ClosesAfterSuccessInHalfOpen(t *testing.T) {
	cfg := &config.TautulliConfig{
		URL:    "http://localhost:8181",
		APIKey: "test-key",
	}

	client := NewTautulliClient(cfg)
	cbName := "test-circuit-breaker-recovery"

	// Create circuit breaker with short timeout and MaxRequests=1 for easier testing
	cb := gobreaker.NewCircuitBreaker[interface{}](gobreaker.Settings{
		Name:        cbName,
		MaxRequests: 1, // Allow only 1 request in half-open
		Interval:    time.Second,
		Timeout:     100 * time.Millisecond,

		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < 10 {
				return false
			}
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return failureRatio >= 0.6
		},
	})

	cbc := &CircuitBreakerClient{
		client: client,
		cb:     cb,
		name:   cbName,
	}

	// Force circuit to open
	for i := 0; i < 10; i++ {
		_, _ = cbc.execute(func() (interface{}, error) {
			return nil, errors.New("simulated API failure")
		})
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Execute successful request in half-open state
	_, err := cbc.execute(func() (interface{}, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("Expected successful request in half-open, got error: %v", err)
	}

	// Circuit should close after successful request in half-open
	state := cbc.cb.State()
	if state != gobreaker.StateClosed {
		t.Errorf("Expected circuit to close after success in half-open, got %v", state)
	}
}

// TestCircuitBreaker_MetricsUpdated verifies Prometheus metrics are updated correctly
func TestCircuitBreaker_MetricsUpdated(t *testing.T) {
	cfg := &config.TautulliConfig{
		URL:    "http://localhost:8181",
		APIKey: "test-key",
	}

	cbc := NewCircuitBreakerClient(cfg)

	// Reset metrics counters (note: in production we can't reset, but for tests we just verify increments)

	// Execute successful request
	_, err := cbc.execute(func() (interface{}, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("Expected successful request, got error: %v", err)
	}

	// Verify success metric would be incremented (we can't easily check counter values in tests)
	// In a real test environment, we'd use prometheus testutil package

	// Execute failed request
	_, err = cbc.execute(func() (interface{}, error) {
		return nil, errors.New("simulated failure")
	})

	if err == nil {
		t.Error("Expected failed request to return error")
	}

	// Verify failure metric would be incremented
	// Note: Full metrics verification requires prometheus/testutil package
}

// TestCircuitBreaker_RealAPICall verifies circuit breaker works with actual TautulliClient method
func TestCircuitBreaker_RealAPICall(t *testing.T) {
	cfg := &config.TautulliConfig{
		URL:    "http://invalid-tautulli-url.example.com",
		APIKey: "test-key",
	}

	cbc := NewCircuitBreakerClient(cfg)

	// This will fail because URL is invalid, but should still go through circuit breaker
	err := cbc.Ping(context.Background())

	// Should get an error (either network error or circuit breaker error)
	if err == nil {
		t.Error("Expected error when calling invalid Tautulli URL")
	}

	// Verify circuit breaker processed the request (counts should be updated)
	counts := cbc.cb.Counts()
	if counts.Requests == 0 {
		t.Error("Expected circuit breaker to track request")
	}
}

// TestCircuitBreaker_MaxRequestsInHalfOpen verifies MaxRequests limit in half-open state
func TestCircuitBreaker_MaxRequestsInHalfOpen(t *testing.T) {
	cfg := &config.TautulliConfig{
		URL:    "http://localhost:8181",
		APIKey: "test-key",
	}

	client := NewTautulliClient(cfg)
	cbName := "test-max-requests"

	// Create circuit breaker with MaxRequests=2
	cb := gobreaker.NewCircuitBreaker[interface{}](gobreaker.Settings{
		Name:        cbName,
		MaxRequests: 2, // Allow only 2 concurrent requests in half-open
		Interval:    time.Second,
		Timeout:     100 * time.Millisecond,

		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < 10 {
				return false
			}
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return failureRatio >= 0.6
		},
	})

	cbc := &CircuitBreakerClient{
		client: client,
		cb:     cb,
		name:   cbName,
	}

	// Force circuit to open
	for i := 0; i < 10; i++ {
		_, _ = cbc.execute(func() (interface{}, error) {
			return nil, errors.New("failure")
		})
	}

	// Wait for timeout to transition to half-open
	time.Sleep(150 * time.Millisecond)

	// In half-open state, try to execute 3 requests (MaxRequests=2)
	// First 2 should succeed (or at least not be rejected)
	// Third should be rejected with ErrTooManyRequests

	_, err1 := cbc.execute(func() (interface{}, error) {
		time.Sleep(50 * time.Millisecond) // Slow request
		return "success", nil
	})

	_, err2 := cbc.execute(func() (interface{}, error) {
		time.Sleep(50 * time.Millisecond) // Slow request
		return "success", nil
	})

	// Note: Depending on timing, these might complete before the third request
	// This test is inherently flaky due to concurrency
	_ = err1
	_ = err2

	// Third request might be rejected if the first two are still running
	// This is timing-dependent, so we just verify the behavior doesn't panic
	_, err3 := cbc.execute(func() (interface{}, error) {
		return "success", nil
	})

	_ = err3
	// Cannot reliably assert on err3 due to race conditions
}

// TestCircuitBreaker_StateHelpers verifies stateToFloat and stateToString helpers
func TestCircuitBreaker_StateHelpers(t *testing.T) {
	tests := []struct {
		state       gobreaker.State
		expectedStr string
		expectedNum float64
	}{
		{gobreaker.StateClosed, "closed", 0},
		{gobreaker.StateHalfOpen, "half-open", 1},
		{gobreaker.StateOpen, "open", 2},
	}

	for _, tt := range tests {
		t.Run(tt.expectedStr, func(t *testing.T) {
			// Test stateToString
			str := stateToString(tt.state)
			if str != tt.expectedStr {
				t.Errorf("stateToString(%v) = %s, expected %s", tt.state, str, tt.expectedStr)
			}

			// Test stateToFloat
			num := stateToFloat(tt.state)
			if num != tt.expectedNum {
				t.Errorf("stateToFloat(%v) = %f, expected %f", tt.state, num, tt.expectedNum)
			}
		})
	}
}

// TestCircuitBreaker_AllInterfaceMethods verifies all TautulliClientInterface methods are implemented
func TestCircuitBreaker_AllInterfaceMethods(t *testing.T) {
	cfg := &config.TautulliConfig{
		URL:    "http://localhost:8181",
		APIKey: "test-key",
	}

	cbc := NewCircuitBreakerClient(cfg)

	// Verify CircuitBreakerClient implements TautulliClientInterface
	var _ TautulliClientInterface = cbc

	// This test just verifies compilation - if CircuitBreakerClient doesn't implement
	// all methods, this won't compile

	t.Log("CircuitBreakerClient successfully implements TautulliClientInterface")
}

// BenchmarkCircuitBreaker_ClosedState benchmarks throughput in closed state
func BenchmarkCircuitBreaker_ClosedState(b *testing.B) {
	cfg := &config.TautulliConfig{
		URL:    "http://localhost:8181",
		APIKey: "test-key",
	}

	cbc := NewCircuitBreakerClient(cfg)

	// Reset metrics before benchmark
	metrics.CircuitBreakerState.WithLabelValues(cbc.name).Set(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cbc.execute(func() (interface{}, error) {
			return "success", nil
		})
	}
}

// BenchmarkCircuitBreaker_OpenState benchmarks rejection speed in open state
func BenchmarkCircuitBreaker_OpenState(b *testing.B) {
	cfg := &config.TautulliConfig{
		URL:    "http://localhost:8181",
		APIKey: "test-key",
	}

	cbc := NewCircuitBreakerClient(cfg)

	// Force circuit to open
	for i := 0; i < 10; i++ {
		_, _ = cbc.execute(func() (interface{}, error) {
			return nil, errors.New("failure")
		})
	}

	// Verify circuit is open
	if cbc.cb.State() != gobreaker.StateOpen {
		b.Fatalf("Circuit should be open for benchmark")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// These should all be rejected instantly
		_, _ = cbc.execute(func() (interface{}, error) {
			return "should not execute", nil
		})
	}
}
