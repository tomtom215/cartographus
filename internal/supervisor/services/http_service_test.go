// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package services

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thejerf/suture/v4"
)

// mockHTTPServer is a test double for HTTPServer interface.
type mockHTTPServer struct {
	listenAndServeErr    error
	listenAndServeBlock  bool
	shutdownErr          error
	listenAndServeCount  atomic.Int32
	shutdownCount        atomic.Int32
	listenAndServeCalled chan struct{}
	stopCh               chan struct{}
}

func newMockHTTPServer() *mockHTTPServer {
	return &mockHTTPServer{
		listenAndServeCalled: make(chan struct{}, 1),
		stopCh:               make(chan struct{}),
	}
}

func (m *mockHTTPServer) ListenAndServe() error {
	m.listenAndServeCount.Add(1)

	// Signal that we've started
	select {
	case m.listenAndServeCalled <- struct{}{}:
	default:
	}

	// Return error immediately if set
	if m.listenAndServeErr != nil {
		return m.listenAndServeErr
	}

	// If blocking, wait until stopped
	if m.listenAndServeBlock {
		<-m.stopCh
		return http.ErrServerClosed
	}

	return nil
}

func (m *mockHTTPServer) Shutdown(ctx context.Context) error {
	m.shutdownCount.Add(1)

	// Unblock ListenAndServe if it's blocking
	close(m.stopCh)

	if m.shutdownErr != nil {
		return m.shutdownErr
	}
	return nil
}

func (m *mockHTTPServer) ListenAndServeCallCount() int {
	return int(m.listenAndServeCount.Load())
}

func (m *mockHTTPServer) ShutdownCallCount() int {
	return int(m.shutdownCount.Load())
}

func TestHTTPServerService_Interface(t *testing.T) {
	// Verify HTTPServerService implements suture.Service
	var _ suture.Service = (*HTTPServerService)(nil)
}

func TestNewHTTPServerService(t *testing.T) {
	server := newMockHTTPServer()
	svc := NewHTTPServerService(server, 10*time.Second)

	if svc == nil {
		t.Fatal("NewHTTPServerService returned nil")
	}
	if svc.server != server {
		t.Error("server not assigned correctly")
	}
	if svc.shutdownTimeout != 10*time.Second {
		t.Errorf("expected shutdown timeout 10s, got %v", svc.shutdownTimeout)
	}
	if svc.name != "http-server" {
		t.Errorf("expected name 'http-server', got %q", svc.name)
	}
}

func TestNewHTTPServerService_DefaultTimeout(t *testing.T) {
	server := newMockHTTPServer()

	// Test zero timeout gets default
	svc := NewHTTPServerService(server, 0)
	if svc.shutdownTimeout != 10*time.Second {
		t.Errorf("expected default timeout 10s, got %v", svc.shutdownTimeout)
	}

	// Test negative timeout gets default
	svc = NewHTTPServerService(server, -5*time.Second)
	if svc.shutdownTimeout != 10*time.Second {
		t.Errorf("expected default timeout 10s, got %v", svc.shutdownTimeout)
	}
}

func TestHTTPServerService_Serve(t *testing.T) {
	t.Run("shuts down gracefully on context cancellation", func(t *testing.T) {
		server := newMockHTTPServer()
		server.listenAndServeBlock = true
		svc := NewHTTPServerService(server, time.Second)

		ctx, cancel := context.WithCancel(context.Background())

		errCh := make(chan error, 1)
		go func() {
			errCh <- svc.Serve(ctx)
		}()

		// Wait for server to start
		select {
		case <-server.listenAndServeCalled:
		case <-time.After(time.Second):
			t.Fatal("server did not start")
		}

		// Cancel context
		cancel()

		select {
		case err := <-errCh:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("expected context.Canceled, got %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("Serve did not return after context cancellation")
		}

		if server.ListenAndServeCallCount() != 1 {
			t.Errorf("expected 1 ListenAndServe call, got %d", server.ListenAndServeCallCount())
		}
		if server.ShutdownCallCount() != 1 {
			t.Errorf("expected 1 Shutdown call, got %d", server.ShutdownCallCount())
		}
	})

	t.Run("returns error on startup failure", func(t *testing.T) {
		expectedErr := errors.New("bind: address already in use")
		server := newMockHTTPServer()
		server.listenAndServeErr = expectedErr
		svc := NewHTTPServerService(server, time.Second)

		ctx := context.Background()
		err := svc.Serve(ctx)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error containing %v, got %v", expectedErr, err)
		}
	})

	t.Run("returns shutdown error if shutdown fails", func(t *testing.T) {
		shutdownErr := errors.New("shutdown timeout")
		server := newMockHTTPServer()
		server.listenAndServeBlock = true
		server.shutdownErr = shutdownErr
		svc := NewHTTPServerService(server, time.Second)

		ctx, cancel := context.WithCancel(context.Background())

		errCh := make(chan error, 1)
		go func() {
			errCh <- svc.Serve(ctx)
		}()

		// Wait for server to start
		<-server.listenAndServeCalled

		// Cancel context
		cancel()

		select {
		case err := <-errCh:
			if !errors.Is(err, shutdownErr) {
				t.Errorf("expected shutdown error, got %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("Serve did not return")
		}
	})
}

func TestHTTPServerService_String(t *testing.T) {
	server := newMockHTTPServer()
	svc := NewHTTPServerService(server, time.Second)

	if svc.String() != "http-server" {
		t.Errorf("expected 'http-server', got %q", svc.String())
	}
}

func TestHTTPServerService_WithSupervisor(t *testing.T) {
	server := newMockHTTPServer()
	server.listenAndServeBlock = true
	svc := NewHTTPServerService(server, time.Second)

	sup := suture.New("test-sup", suture.Spec{
		FailureThreshold: 3,
		FailureBackoff:   10 * time.Millisecond,
		Timeout:          2 * time.Second,
	})
	sup.Add(svc)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	errCh := sup.ServeBackground(ctx)

	// Wait for server to start
	select {
	case <-server.listenAndServeCalled:
	case <-time.After(time.Second):
		t.Fatal("server did not start")
	}

	if server.ListenAndServeCallCount() < 1 {
		t.Error("server ListenAndServe was not called")
	}

	cancel()
	<-errCh

	// Verify shutdown was called
	if server.ShutdownCallCount() < 1 {
		t.Error("server Shutdown was not called")
	}
}
