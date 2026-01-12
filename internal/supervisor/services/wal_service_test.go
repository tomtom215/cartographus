// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal

package services

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thejerf/suture/v4"
)

// MockWALComponent simulates a WAL component (RetryLoop or Compactor) for testing.
// Implements the WALStartStopper interface defined in wal_service.go.
type MockWALComponent struct {
	running  atomic.Bool
	started  atomic.Bool
	startErr error
}

func NewMockWALComponent() *MockWALComponent {
	return &MockWALComponent{}
}

func (m *MockWALComponent) Start(ctx context.Context) error {
	if m.startErr != nil {
		return m.startErr
	}
	m.started.Store(true)
	m.running.Store(true)
	return nil
}

func (m *MockWALComponent) Stop() {
	m.running.Store(false)
}

func (m *MockWALComponent) IsRunning() bool {
	return m.running.Load()
}

func (m *MockWALComponent) SetStartError(err error) {
	m.startErr = err
}

func TestWALRetryLoopService(t *testing.T) {
	t.Run("implements suture.Service interface", func(t *testing.T) {
		var _ suture.Service = (*WALRetryLoopService)(nil)
	})

	t.Run("starts underlying retry loop", func(t *testing.T) {
		mock := NewMockWALComponent()
		svc := NewWALRetryLoopService(mock)

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
			t.Error("retry loop should have been started")
		}
		if !mock.IsRunning() {
			t.Error("retry loop should be running")
		}

		cancel()
		<-done
	})

	t.Run("stops retry loop on context cancellation", func(t *testing.T) {
		mock := NewMockWALComponent()
		svc := NewWALRetryLoopService(mock)

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

		// Give a moment for Stop to be called
		time.Sleep(10 * time.Millisecond)
		if mock.IsRunning() {
			t.Error("retry loop should have been stopped")
		}
	})

	t.Run("propagates start error for restart", func(t *testing.T) {
		mock := NewMockWALComponent()
		mock.SetStartError(errors.New("BadgerDB open failed"))
		svc := NewWALRetryLoopService(mock)

		err := svc.Serve(context.Background())
		if err == nil {
			t.Error("expected error to be propagated")
		}
	})

	t.Run("String returns service name", func(t *testing.T) {
		mock := NewMockWALComponent()
		svc := NewWALRetryLoopService(mock)

		if svc.String() != "wal-retry-loop" {
			t.Errorf("expected 'wal-retry-loop', got '%s'", svc.String())
		}
	})
}

func TestWALCompactorService(t *testing.T) {
	t.Run("implements suture.Service interface", func(t *testing.T) {
		var _ suture.Service = (*WALCompactorService)(nil)
	})

	t.Run("starts underlying compactor", func(t *testing.T) {
		mock := NewMockWALComponent()
		svc := NewWALCompactorService(mock)

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
			t.Error("compactor should have been started")
		}
		if !mock.IsRunning() {
			t.Error("compactor should be running")
		}

		cancel()
		<-done
	})

	t.Run("stops compactor on context cancellation", func(t *testing.T) {
		mock := NewMockWALComponent()
		svc := NewWALCompactorService(mock)

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

		// Give a moment for Stop to be called
		time.Sleep(10 * time.Millisecond)
		if mock.IsRunning() {
			t.Error("compactor should have been stopped")
		}
	})

	t.Run("propagates start error for restart", func(t *testing.T) {
		mock := NewMockWALComponent()
		mock.SetStartError(errors.New("disk full"))
		svc := NewWALCompactorService(mock)

		err := svc.Serve(context.Background())
		if err == nil {
			t.Error("expected error to be propagated")
		}
	})

	t.Run("String returns service name", func(t *testing.T) {
		mock := NewMockWALComponent()
		svc := NewWALCompactorService(mock)

		if svc.String() != "wal-compactor" {
			t.Errorf("expected 'wal-compactor', got '%s'", svc.String())
		}
	})
}
