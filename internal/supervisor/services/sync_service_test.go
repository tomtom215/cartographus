// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package services

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thejerf/suture/v4"
)

// MockSyncManager simulates the sync.Manager for testing.
// It matches the StartStopManager interface.
type MockSyncManager struct {
	started    atomic.Bool
	stopped    atomic.Bool
	startError error
	stopError  error
}

func (m *MockSyncManager) Start(ctx context.Context) error {
	if m.startError != nil {
		return m.startError
	}
	m.started.Store(true)
	return nil
}

func (m *MockSyncManager) Stop() error {
	m.stopped.Store(true)
	return m.stopError
}

func TestSyncServiceInterface(t *testing.T) {
	t.Run("implements suture.Service", func(t *testing.T) {
		var _ suture.Service = (*SyncService)(nil)
	})
}

func TestSyncService(t *testing.T) {
	t.Run("starts underlying sync manager", func(t *testing.T) {
		mockMgr := &MockSyncManager{}
		svc := NewSyncService(mockMgr)

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
			if mockMgr.started.Load() {
				started = true
				break
			}
		}
		if !started {
			t.Error("sync manager was not started")
		}

		// Let context expire
		<-done
	})

	t.Run("stops manager on context cancellation", func(t *testing.T) {
		mockMgr := &MockSyncManager{}
		svc := NewSyncService(mockMgr)

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error, 1)
		go func() {
			done <- svc.Serve(ctx)
		}()

		// Wait for start with polling (more reliable in CI under load)
		for i := 0; i < 10; i++ {
			time.Sleep(20 * time.Millisecond)
			if mockMgr.started.Load() {
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

		if !mockMgr.stopped.Load() {
			t.Error("sync manager was not stopped")
		}
	})

	t.Run("propagates start error for restart", func(t *testing.T) {
		expectedErr := errors.New("tautulli connection failed")
		mockMgr := &MockSyncManager{
			startError: expectedErr,
		}
		svc := NewSyncService(mockMgr)

		err := svc.Serve(context.Background())
		if err == nil {
			t.Error("expected error to be propagated")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected wrapped tautulli error, got %v", err)
		}

		// Manager should not be marked as started
		if mockMgr.started.Load() {
			t.Error("manager should not be started on error")
		}
	})

	t.Run("handles stop error gracefully", func(t *testing.T) {
		mockMgr := &MockSyncManager{
			stopError: errors.New("stop failed"),
		}
		svc := NewSyncService(mockMgr)

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error, 1)
		go func() {
			done <- svc.Serve(ctx)
		}()

		// Wait for start with polling (more reliable in CI under load)
		for i := 0; i < 10; i++ {
			time.Sleep(20 * time.Millisecond)
			if mockMgr.started.Load() {
				break
			}
		}
		cancel()

		err := <-done
		// Should still get an error (wrapped stop error)
		if err == nil {
			t.Error("expected error from stop failure")
		}
	})

	t.Run("String returns service name", func(t *testing.T) {
		svc := NewSyncService(&MockSyncManager{})
		if svc.String() != "sync-manager" {
			t.Errorf("expected 'sync-manager', got %q", svc.String())
		}
	})
}

func TestSyncServiceWithSupervisor(t *testing.T) {
	t.Run("supervisor restarts on start failure", func(t *testing.T) {
		startCount := atomic.Int32{}

		mockMgr := &restartableMockManager{
			startCount: &startCount,
			failUntil:  2, // Fail first 2 starts
		}
		svc := NewSyncService(mockMgr)

		sup := suture.New("sync-test", suture.Spec{
			FailureThreshold: 10,
			FailureBackoff:   10 * time.Millisecond,
			Timeout:          100 * time.Millisecond,
		})
		sup.Add(svc)

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		go func() {
			if err := sup.Serve(ctx); err != nil && err != context.DeadlineExceeded && err != context.Canceled {
				t.Logf("Supervisor serve error (expected during test): %v", err)
			}
		}()
		time.Sleep(200 * time.Millisecond)

		// Should have been started at least 3 times (2 failures + 1 success)
		if startCount.Load() < 3 {
			t.Errorf("expected at least 3 start attempts, got %d", startCount.Load())
		}
	})
}

// restartableMockManager fails the first N starts, then succeeds
type restartableMockManager struct {
	startCount *atomic.Int32
	stopCount  atomic.Int32
	failUntil  int32
}

func (m *restartableMockManager) Start(ctx context.Context) error {
	count := m.startCount.Add(1)
	if count <= m.failUntil {
		return errors.New("simulated start failure")
	}
	return nil
}

func (m *restartableMockManager) Stop() error {
	m.stopCount.Add(1)
	return nil
}
