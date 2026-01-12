// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package services

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thejerf/suture/v4"
)

// MockNATSComponents simulates the NATSComponents for testing.
// Implements the NATSComponentsRunner interface defined in nats_service.go.
type MockNATSComponents struct {
	running  atomic.Bool
	started  atomic.Bool
	startErr error
}

func NewMockNATSComponents() *MockNATSComponents {
	return &MockNATSComponents{}
}

func (m *MockNATSComponents) Start(ctx context.Context) error {
	if m.startErr != nil {
		return m.startErr
	}
	m.started.Store(true)
	m.running.Store(true)
	return nil
}

func (m *MockNATSComponents) Shutdown(ctx context.Context) {
	m.running.Store(false)
}

func (m *MockNATSComponents) IsRunning() bool {
	return m.running.Load()
}

func (m *MockNATSComponents) SetStartError(err error) {
	m.startErr = err
}

func TestNATSComponentsService(t *testing.T) {
	t.Run("implements suture.Service interface", func(t *testing.T) {
		var _ suture.Service = (*NATSComponentsService)(nil)
	})

	t.Run("starts underlying NATS components", func(t *testing.T) {
		mock := NewMockNATSComponents()
		svc := NewNATSComponentsService(mock)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- svc.Serve(ctx)
		}()

		// Wait for service to start with polling (more reliable in CI under load)
		var started bool
		for i := 0; i < 10; i++ {
			time.Sleep(20 * time.Millisecond)
			if mock.started.Load() {
				started = true
				break
			}
		}

		if !started {
			t.Error("NATS components should have been started")
		}
		if !mock.IsRunning() {
			t.Error("NATS components should be running")
		}

		cancel()
		<-done
	})

	t.Run("stops components on context cancellation", func(t *testing.T) {
		mock := NewMockNATSComponents()
		svc := NewNATSComponentsService(mock)

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error, 1)
		go func() {
			done <- svc.Serve(ctx)
		}()

		// Wait for start with polling (more reliable in CI under load)
		for i := 0; i < 10; i++ {
			time.Sleep(20 * time.Millisecond)
			if mock.started.Load() {
				break
			}
		}
		cancel()

		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("expected context.Canceled, got %v", err)
			}
		case <-time.After(time.Second):
			t.Error("service did not stop in time")
		}

		if mock.IsRunning() {
			t.Error("NATS components should have been stopped")
		}
	})

	t.Run("propagates start error for restart", func(t *testing.T) {
		mock := NewMockNATSComponents()
		mock.SetStartError(errors.New("NATS connection refused"))
		svc := NewNATSComponentsService(mock)

		err := svc.Serve(context.Background())
		if err == nil {
			t.Error("expected error to be propagated")
		}
		if !errors.Is(err, mock.startErr) && err.Error() != "NATS components start failed: NATS connection refused" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("String returns service name", func(t *testing.T) {
		mock := NewMockNATSComponents()
		svc := NewNATSComponentsService(mock)

		if svc.String() != "nats-components" {
			t.Errorf("expected 'nats-components', got '%s'", svc.String())
		}
	})
}

func TestNATSComponentsServiceWithTimeout(t *testing.T) {
	t.Run("respects shutdown timeout", func(t *testing.T) {
		mock := NewMockNATSComponents()
		timeout := 5 * time.Second
		svc := NewNATSComponentsServiceWithTimeout(mock, timeout)

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error, 1)
		go func() {
			done <- svc.Serve(ctx)
		}()

		// Wait for start with polling (more reliable in CI under load)
		for i := 0; i < 10; i++ {
			time.Sleep(20 * time.Millisecond)
			if mock.started.Load() {
				break
			}
		}
		cancel()

		select {
		case <-done:
			// Success
		case <-time.After(time.Second):
			t.Error("service did not stop in time")
		}
	})
}
