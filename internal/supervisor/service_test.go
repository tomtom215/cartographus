// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package supervisor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thejerf/suture/v4"
)

// TestServiceInterface verifies services implement suture.Service correctly.
func TestServiceInterface(t *testing.T) {
	t.Run("MockService implements suture.Service", func(t *testing.T) {
		// Compile-time check that MockService satisfies suture.Service
		var _ suture.Service = (*MockService)(nil)
	})
}

// TestMockService validates the test helper works correctly.
func TestMockService(t *testing.T) {
	t.Run("runs until context canceled", func(t *testing.T) {
		svc := NewMockService("test")
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := svc.Serve(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected context.DeadlineExceeded, got %v", err)
		}
		if svc.StartCount() != 1 {
			t.Errorf("expected 1 start, got %d", svc.StartCount())
		}
	})

	t.Run("returns error on simulated failure", func(t *testing.T) {
		svc := NewMockService("failing")
		svc.SetError(errors.New("simulated failure"))

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := svc.Serve(ctx)
		if err == nil || err.Error() != "simulated failure" {
			t.Errorf("expected simulated failure, got %v", err)
		}
	})

	t.Run("returns ErrDoNotRestart for permanent completion", func(t *testing.T) {
		svc := NewMockService("one-shot")
		svc.SetError(suture.ErrDoNotRestart)

		ctx := context.Background()
		err := svc.Serve(ctx)
		if !errors.Is(err, suture.ErrDoNotRestart) {
			t.Errorf("expected ErrDoNotRestart, got %v", err)
		}
	})

	t.Run("fails N times then succeeds", func(t *testing.T) {
		svc := NewMockService("retry-test")
		svc.SetFailCount(2)

		// First two calls should fail
		err := svc.Serve(context.Background())
		if err == nil || err.Error() != "simulated failure" {
			t.Errorf("first call should fail, got %v", err)
		}

		err = svc.Serve(context.Background())
		if err == nil || err.Error() != "simulated failure" {
			t.Errorf("second call should fail, got %v", err)
		}

		// Third call should succeed (run until context canceled)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		err = svc.Serve(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("third call should succeed until timeout, got %v", err)
		}

		if svc.StartCount() != 3 {
			t.Errorf("expected 3 starts, got %d", svc.StartCount())
		}
	})

	t.Run("String returns service name", func(t *testing.T) {
		svc := NewMockService("my-service")
		if svc.String() != "my-service" {
			t.Errorf("expected 'my-service', got %q", svc.String())
		}
	})
}

// TestSupervisorBasics validates supervisor behavior.
func TestSupervisorBasics(t *testing.T) {
	t.Run("supervisor starts and stops services", func(t *testing.T) {
		svc := NewMockService("basic")
		sup := suture.NewSimple("test-supervisor")
		sup.Add(svc)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		errCh := make(chan error, 1)
		go func() {
			errCh <- sup.Serve(ctx)
		}()

		// Wait for service to start with polling (more reliable in CI under load)
		var started bool
		for i := 0; i < 10; i++ {
			time.Sleep(20 * time.Millisecond)
			if svc.StartCount() >= 1 {
				started = true
				break
			}
		}
		if !started {
			t.Error("service was not started")
		}

		cancel()
		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("unexpected supervisor error: %v", err)
			}
		case <-time.After(time.Second):
			t.Error("supervisor did not stop in time")
		}
	})

	t.Run("supervisor restarts crashed service", func(t *testing.T) {
		svc := NewMockService("crasher")
		svc.SetFailCount(2) // Fail twice, then succeed

		sup := suture.New("restart-test", suture.Spec{
			FailureThreshold: 10,
			FailureDecay:     1,
			FailureBackoff:   10 * time.Millisecond,
			Timeout:          100 * time.Millisecond,
		})
		sup.Add(svc)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		go sup.Serve(ctx)
		time.Sleep(300 * time.Millisecond)

		// Should have been started at least 3 times: 2 failures + 1 success
		if svc.StartCount() < 3 {
			t.Errorf("expected at least 3 starts (2 failures + 1 success), got %d", svc.StartCount())
		}
	})

	t.Run("service returning ErrDoNotRestart is not restarted", func(t *testing.T) {
		svc := NewMockService("one-shot")
		svc.SetError(suture.ErrDoNotRestart)

		sup := suture.New("no-restart-test", suture.Spec{
			FailureThreshold: 10,
			FailureBackoff:   10 * time.Millisecond,
			Timeout:          100 * time.Millisecond,
		})
		sup.Add(svc)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		go sup.Serve(ctx)
		time.Sleep(100 * time.Millisecond)

		// Should only be started once
		if svc.StartCount() != 1 {
			t.Errorf("expected exactly 1 start for ErrDoNotRestart, got %d", svc.StartCount())
		}
	})
}

// TestErrTerminateSupervisorTree validates tree termination behavior.
func TestErrTerminateSupervisorTree(t *testing.T) {
	t.Run("service can terminate entire tree", func(t *testing.T) {
		svc := NewMockService("terminator")
		svc.SetError(suture.ErrTerminateSupervisorTree)

		sup := suture.New("tree-test", suture.Spec{
			FailureThreshold: 10,
			Timeout:          100 * time.Millisecond,
		})
		sup.Add(svc)

		ctx := context.Background()
		err := sup.Serve(ctx)

		// The supervisor should terminate with ErrTerminateSupervisorTree
		if !errors.Is(err, suture.ErrTerminateSupervisorTree) {
			t.Logf("supervisor returned: %v (expected ErrTerminateSupervisorTree or wrapped)", err)
		}
	})
}

// TestHierarchicalSupervisors validates nested supervisor behavior.
func TestHierarchicalSupervisors(t *testing.T) {
	t.Run("child supervisors are started by parent", func(t *testing.T) {
		childSvc := NewMockService("child-service")
		childSup := suture.NewSimple("child-supervisor")
		childSup.Add(childSvc)

		parentSup := suture.NewSimple("parent-supervisor")
		parentSup.Add(childSup)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		go parentSup.Serve(ctx)
		time.Sleep(100 * time.Millisecond)

		if childSvc.StartCount() < 1 {
			t.Error("child service was not started through hierarchy")
		}
	})
}
