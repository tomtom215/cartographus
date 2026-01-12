// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package supervisor

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

// MockService is a test helper that implements suture.Service.
// It provides control over service behavior for testing supervisor functionality.
type MockService struct {
	name       string
	startCount atomic.Int32
	stopCount  atomic.Int32
	failCount  atomic.Int32
	maxFails   int32
	err        error
	mu         sync.Mutex
}

// NewMockService creates a new mock service for testing.
func NewMockService(name string) *MockService {
	return &MockService{name: name}
}

// Serve implements suture.Service.
// The method signature matches suture v4's Service interface exactly:
// Serve(ctx context.Context) error
func (m *MockService) Serve(ctx context.Context) error {
	m.startCount.Add(1)
	defer m.stopCount.Add(1)

	m.mu.Lock()
	err := m.err
	maxFails := m.maxFails
	m.mu.Unlock()

	// If we have a fail count, fail that many times before succeeding
	if maxFails > 0 {
		current := m.failCount.Add(1)
		if current <= maxFails {
			return errors.New("simulated failure")
		}
	}

	// If error is set, return it immediately
	if err != nil {
		return err
	}

	// Otherwise, run until context is canceled
	<-ctx.Done()
	return ctx.Err()
}

// SetError configures the service to return this error immediately.
// Useful for testing error propagation and restart behavior.
func (m *MockService) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// SetFailCount configures the service to fail N times before succeeding.
// Each call to Serve will fail until the fail count is exhausted.
func (m *MockService) SetFailCount(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxFails = int32(n)
}

// StartCount returns how many times Serve was called.
func (m *MockService) StartCount() int32 {
	return m.startCount.Load()
}

// StopCount returns how many times Serve returned.
func (m *MockService) StopCount() int32 {
	return m.stopCount.Load()
}

// String implements fmt.Stringer for logging.
// Suture uses this to identify services in log messages.
func (m *MockService) String() string {
	return m.name
}
