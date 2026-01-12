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

// mockDetectionEngine implements DetectionEngine for testing.
type mockDetectionEngine struct {
	runErr     error
	runBlocks  bool
	runCount   atomic.Int32
	runStarted chan struct{}
	stopCh     chan struct{}
}

func newMockDetectionEngine() *mockDetectionEngine {
	return &mockDetectionEngine{
		runStarted: make(chan struct{}, 1),
		stopCh:     make(chan struct{}),
	}
}

func (m *mockDetectionEngine) RunWithContext(ctx context.Context) error {
	m.runCount.Add(1)

	// Signal that we've started
	select {
	case m.runStarted <- struct{}{}:
	default:
	}

	// Return error immediately if set
	if m.runErr != nil {
		return m.runErr
	}

	// If blocking, wait until context canceled or stopped
	if m.runBlocks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.stopCh:
			return nil
		}
	}

	return nil
}

func (m *mockDetectionEngine) RunCallCount() int {
	return int(m.runCount.Load())
}

func (m *mockDetectionEngine) Stop() {
	select {
	case m.stopCh <- struct{}{}:
	default:
	}
}

// --- Test: DetectionService implements suture.Service ---

func TestDetectionService_Interface(t *testing.T) {
	t.Parallel()

	// Verify DetectionService implements suture.Service
	var _ suture.Service = (*DetectionService)(nil)
}

// --- Test: NewDetectionService ---

func TestNewDetectionService(t *testing.T) {
	t.Parallel()

	engine := newMockDetectionEngine()
	svc := NewDetectionService(engine)

	if svc == nil {
		t.Fatal("NewDetectionService() = nil, want non-nil")
	}

	if svc.engine != engine {
		t.Error("engine not assigned correctly")
	}

	if svc.name != "detection-engine" {
		t.Errorf("expected name 'detection-engine', got %q", svc.name)
	}
}

// --- Test: DetectionService.Serve ---

func TestDetectionService_Serve(t *testing.T) {
	t.Parallel()

	t.Run("calls engine RunWithContext", func(t *testing.T) {
		t.Parallel()

		engine := newMockDetectionEngine()
		engine.runBlocks = true
		svc := NewDetectionService(engine)

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)

		go func() {
			errCh <- svc.Serve(ctx)
		}()

		// Wait for engine to start
		select {
		case <-engine.runStarted:
		case <-time.After(time.Second):
			t.Fatal("engine did not start")
		}

		// Cancel context
		cancel()

		select {
		case err := <-errCh:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("Serve() error = %v, want context.Canceled", err)
			}
		case <-time.After(time.Second):
			t.Error("Serve() did not return after context cancellation")
		}

		if engine.RunCallCount() != 1 {
			t.Errorf("RunWithContext called %d times, want 1", engine.RunCallCount())
		}
	})

	t.Run("propagates engine error", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("detection engine error")
		engine := newMockDetectionEngine()
		engine.runErr = expectedErr
		svc := NewDetectionService(engine)

		err := svc.Serve(context.Background())

		if !errors.Is(err, expectedErr) {
			t.Errorf("Serve() error = %v, want %v", err, expectedErr)
		}
	})

	t.Run("returns immediately when engine returns", func(t *testing.T) {
		t.Parallel()

		engine := newMockDetectionEngine()
		engine.runBlocks = false // Returns immediately
		svc := NewDetectionService(engine)

		done := make(chan struct{})
		go func() {
			_ = svc.Serve(context.Background())
			close(done)
		}()

		select {
		case <-done:
			// Expected
		case <-time.After(time.Second):
			t.Error("Serve() did not return when engine returned")
		}
	})
}

// --- Test: DetectionService.String ---

func TestDetectionService_String(t *testing.T) {
	t.Parallel()

	engine := newMockDetectionEngine()
	svc := NewDetectionService(engine)

	if got := svc.String(); got != "detection-engine" {
		t.Errorf("String() = %q, want 'detection-engine'", got)
	}
}

// --- Test: Integration with Suture supervisor ---

func TestDetectionService_WithSupervisor(t *testing.T) {
	t.Parallel()

	engine := newMockDetectionEngine()
	engine.runBlocks = true
	svc := NewDetectionService(engine)

	sup := suture.New("test-sup", suture.Spec{
		FailureThreshold: 3,
		FailureBackoff:   10 * time.Millisecond,
		Timeout:          2 * time.Second,
	})
	sup.Add(svc)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	errCh := sup.ServeBackground(ctx)

	// Wait for engine to start
	select {
	case <-engine.runStarted:
	case <-time.After(time.Second):
		t.Fatal("engine did not start under supervisor")
	}

	if engine.RunCallCount() < 1 {
		t.Error("RunWithContext was not called")
	}

	cancel()
	<-errCh
}

func TestDetectionService_RestartOnError(t *testing.T) {
	t.Parallel()

	engine := newMockDetectionEngine()
	engine.runErr = errors.New("transient error")
	svc := NewDetectionService(engine)

	sup := suture.New("test-sup", suture.Spec{
		FailureThreshold: 10,
		FailureBackoff:   5 * time.Millisecond,
		Timeout:          time.Second,
	})
	sup.Add(svc)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	errCh := sup.ServeBackground(ctx)
	<-errCh

	// Should have been restarted multiple times due to error
	if engine.RunCallCount() < 2 {
		t.Errorf("expected multiple restarts, got %d runs", engine.RunCallCount())
	}
}
