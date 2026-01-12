// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package eventprocessor

import (
	gobreaker "github.com/sony/gobreaker/v2"
)

// NewCircuitBreaker creates a circuit breaker with the given configuration.
// Uses gobreaker v2.4.0 generic API with interface{} type parameter for flexibility.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *gobreaker.CircuitBreaker[interface{}] {
	settings := gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cfg.FailureThreshold
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			// State change callback - can be extended for logging/metrics
		},
	}

	return gobreaker.NewCircuitBreaker[interface{}](settings)
}

// CircuitBreakerState converts gobreaker.State to a string for monitoring.
func CircuitBreakerState(cb *gobreaker.CircuitBreaker[interface{}]) string {
	return cb.State().String()
}

// ExecuteWithBreaker wraps a function with circuit breaker protection.
// Returns the result and any error from the function or circuit breaker.
func ExecuteWithBreaker(cb *gobreaker.CircuitBreaker[interface{}], fn func() (interface{}, error)) (interface{}, error) {
	return cb.Execute(fn)
}
