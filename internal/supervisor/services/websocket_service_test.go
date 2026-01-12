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

// mockContextHub is a test double for ContextHub interface.
type mockContextHub struct {
	runErr      error
	runCount    atomic.Int32
	runDuration time.Duration
}

func (m *mockContextHub) RunWithContext(ctx context.Context) error {
	m.runCount.Add(1)
	if m.runErr != nil {
		return m.runErr
	}
	if m.runDuration > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.runDuration):
			return nil
		}
	}
	<-ctx.Done()
	return ctx.Err()
}

func (m *mockContextHub) RunCount() int {
	return int(m.runCount.Load())
}

func TestWebSocketHubService_Interface(t *testing.T) {
	// Verify WebSocketHubService implements suture.Service
	var _ suture.Service = (*WebSocketHubService)(nil)
}

func TestNewWebSocketHubService(t *testing.T) {
	hub := &mockContextHub{}
	svc := NewWebSocketHubService(hub)

	if svc == nil {
		t.Fatal("NewWebSocketHubService returned nil")
	}
	if svc.hub != hub {
		t.Error("hub not assigned correctly")
	}
	if svc.name != "websocket-hub" {
		t.Errorf("expected name 'websocket-hub', got %q", svc.name)
	}
}

func TestWebSocketHubService_Serve(t *testing.T) {
	t.Run("returns context error on cancellation", func(t *testing.T) {
		hub := &mockContextHub{}
		svc := NewWebSocketHubService(hub)

		ctx, cancel := context.WithCancel(context.Background())

		errCh := make(chan error, 1)
		go func() {
			errCh <- svc.Serve(ctx)
		}()

		time.Sleep(20 * time.Millisecond)
		cancel()

		select {
		case err := <-errCh:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("expected context.Canceled, got %v", err)
			}
		case <-time.After(time.Second):
			t.Error("Serve did not return after context cancellation")
		}

		if hub.RunCount() != 1 {
			t.Errorf("expected 1 run, got %d", hub.RunCount())
		}
	})

	t.Run("returns context error on deadline", func(t *testing.T) {
		hub := &mockContextHub{}
		svc := NewWebSocketHubService(hub)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := svc.Serve(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected context.DeadlineExceeded, got %v", err)
		}
	})

	t.Run("propagates hub errors", func(t *testing.T) {
		expectedErr := errors.New("hub startup error")
		hub := &mockContextHub{runErr: expectedErr}
		svc := NewWebSocketHubService(hub)

		ctx := context.Background()
		err := svc.Serve(ctx)

		if !errors.Is(err, expectedErr) {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})
}

func TestWebSocketHubService_String(t *testing.T) {
	hub := &mockContextHub{}
	svc := NewWebSocketHubService(hub)

	if svc.String() != "websocket-hub" {
		t.Errorf("expected 'websocket-hub', got %q", svc.String())
	}
}

func TestWebSocketHubService_WithSupervisor(t *testing.T) {
	hub := &mockContextHub{}
	svc := NewWebSocketHubService(hub)

	sup := suture.New("test-sup", suture.Spec{
		FailureThreshold: 3,
		FailureBackoff:   10 * time.Millisecond,
		Timeout:          100 * time.Millisecond,
	})
	sup.Add(svc)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	errCh := sup.ServeBackground(ctx)

	// Wait for hub to start with polling (more reliable in CI under load)
	var started bool
	for i := 0; i < 10; i++ {
		time.Sleep(20 * time.Millisecond)
		if hub.RunCount() >= 1 {
			started = true
			break
		}
	}

	if !started {
		t.Error("hub RunWithContext was not called")
	}

	cancel()
	<-errCh
}
