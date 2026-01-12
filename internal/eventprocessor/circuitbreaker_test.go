// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package eventprocessor

import (
	"errors"
	"testing"
	"time"

	gobreaker "github.com/sony/gobreaker/v2"
)

func TestNewCircuitBreaker(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig("test-breaker")
	cb := NewCircuitBreaker(cfg)

	if cb == nil {
		t.Fatal("Expected non-nil circuit breaker")
	}
	if cb.Name() != "test-breaker" {
		t.Errorf("Expected name=test-breaker, got %s", cb.Name())
	}
}

func TestCircuitBreakerState(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig("test-breaker")
	cb := NewCircuitBreaker(cfg)

	state := CircuitBreakerState(cb)
	if state != "closed" {
		t.Errorf("Expected initial state=closed, got %s", state)
	}
}

func TestExecuteWithBreaker(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		cfg := DefaultCircuitBreakerConfig("success-test")
		cb := NewCircuitBreaker(cfg)

		result, err := ExecuteWithBreaker(cb, func() (interface{}, error) {
			return "success", nil
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result != "success" {
			t.Errorf("Expected 'success', got %v", result)
		}
	})

	t.Run("failed execution", func(t *testing.T) {
		cfg := DefaultCircuitBreakerConfig("failure-test")
		cb := NewCircuitBreaker(cfg)

		expectedErr := errors.New("test error")
		_, err := ExecuteWithBreaker(cb, func() (interface{}, error) {
			return nil, expectedErr
		})

		if !errors.Is(err, expectedErr) {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("circuit opens after failures", func(t *testing.T) {
		cfg := CircuitBreakerConfig{
			Name:             "open-test",
			MaxRequests:      1,
			Interval:         1 * time.Second,
			Timeout:          1 * time.Second,
			FailureThreshold: 2, // Open after 2 consecutive failures
		}
		cb := NewCircuitBreaker(cfg)

		testErr := errors.New("fail")

		// First failure
		_, _ = ExecuteWithBreaker(cb, func() (interface{}, error) {
			return nil, testErr
		})

		// Second failure - should trip the breaker
		_, _ = ExecuteWithBreaker(cb, func() (interface{}, error) {
			return nil, testErr
		})

		// Third call - should fail with circuit open
		_, err := ExecuteWithBreaker(cb, func() (interface{}, error) {
			return "should not execute", nil
		})

		if !errors.Is(err, gobreaker.ErrOpenState) {
			t.Errorf("Expected ErrOpenState, got %v", err)
		}
	})
}

func TestCircuitBreakerRecovery(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Name:             "recovery-test",
		MaxRequests:      1,
		Interval:         100 * time.Millisecond,
		Timeout:          100 * time.Millisecond, // Short timeout for testing
		FailureThreshold: 1,
	}
	cb := NewCircuitBreaker(cfg)

	// Trigger circuit open
	_, _ = ExecuteWithBreaker(cb, func() (interface{}, error) {
		return nil, errors.New("fail")
	})

	// Verify circuit is open
	_, err := ExecuteWithBreaker(cb, func() (interface{}, error) {
		return "test", nil
	})
	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Errorf("Expected ErrOpenState, got %v", err)
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Now in half-open state, should allow one request
	result, err := ExecuteWithBreaker(cb, func() (interface{}, error) {
		return "recovered", nil
	})
	if err != nil {
		t.Errorf("Unexpected error after recovery: %v", err)
	}
	if result != "recovered" {
		t.Errorf("Expected 'recovered', got %v", result)
	}

	// Circuit should be closed again
	state := CircuitBreakerState(cb)
	if state != "closed" {
		t.Errorf("Expected state=closed after recovery, got %s", state)
	}
}
