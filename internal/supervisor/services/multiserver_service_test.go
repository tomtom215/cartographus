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

// mockJellyfinManager is a test double for JellyfinStartStopManager interface.
type mockJellyfinManager struct {
	startErr     error
	stopErr      error
	startBlocks  bool
	startCount   atomic.Int32
	stopCount    atomic.Int32
	startStarted chan struct{}
	stopCh       chan struct{}
}

func newMockJellyfinManager() *mockJellyfinManager {
	return &mockJellyfinManager{
		startStarted: make(chan struct{}, 1),
		stopCh:       make(chan struct{}),
	}
}

func (m *mockJellyfinManager) Start(ctx context.Context) error {
	m.startCount.Add(1)

	// Signal that we've started
	if m.startStarted != nil {
		select {
		case m.startStarted <- struct{}{}:
		default:
		}
	}

	// Return error immediately if set
	if m.startErr != nil {
		return m.startErr
	}

	// If blocking, wait until context canceled or stopped
	if m.startBlocks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.stopCh:
			return nil
		}
	}

	return nil
}

func (m *mockJellyfinManager) Stop() error {
	m.stopCount.Add(1)

	// Unblock Start if it's blocking
	if m.stopCh != nil {
		select {
		case m.stopCh <- struct{}{}:
		default:
		}
	}

	if m.stopErr != nil {
		return m.stopErr
	}
	return nil
}

func (m *mockJellyfinManager) StartCallCount() int {
	return int(m.startCount.Load())
}

func (m *mockJellyfinManager) StopCallCount() int {
	return int(m.stopCount.Load())
}

// mockEmbyManager is a test double for EmbyStartStopManager interface.
type mockEmbyManager struct {
	startErr     error
	stopErr      error
	startBlocks  bool
	startCount   atomic.Int32
	stopCount    atomic.Int32
	startStarted chan struct{}
	stopCh       chan struct{}
}

func newMockEmbyManager() *mockEmbyManager {
	return &mockEmbyManager{
		startStarted: make(chan struct{}, 1),
		stopCh:       make(chan struct{}),
	}
}

func (m *mockEmbyManager) Start(ctx context.Context) error {
	m.startCount.Add(1)

	// Signal that we've started
	if m.startStarted != nil {
		select {
		case m.startStarted <- struct{}{}:
		default:
		}
	}

	// Return error immediately if set
	if m.startErr != nil {
		return m.startErr
	}

	// If blocking, wait until context canceled or stopped
	if m.startBlocks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.stopCh:
			return nil
		}
	}

	return nil
}

func (m *mockEmbyManager) Stop() error {
	m.stopCount.Add(1)

	// Unblock Start if it's blocking
	if m.stopCh != nil {
		select {
		case m.stopCh <- struct{}{}:
		default:
		}
	}

	if m.stopErr != nil {
		return m.stopErr
	}
	return nil
}

func (m *mockEmbyManager) StartCallCount() int {
	return int(m.startCount.Load())
}

func (m *mockEmbyManager) StopCallCount() int {
	return int(m.stopCount.Load())
}

// --- Test: JellyfinService implements suture.Service ---

func TestJellyfinService_Interface(t *testing.T) {
	t.Parallel()

	// Verify JellyfinService implements suture.Service
	var _ suture.Service = (*JellyfinService)(nil)
}

// --- Test: EmbyService implements suture.Service ---

func TestEmbyService_Interface(t *testing.T) {
	t.Parallel()

	// Verify EmbyService implements suture.Service
	var _ suture.Service = (*EmbyService)(nil)
}

// --- Test: NewJellyfinService ---

func TestNewJellyfinService(t *testing.T) {
	t.Parallel()

	manager := newMockJellyfinManager()
	svc := NewJellyfinService(manager)

	if svc == nil {
		t.Fatal("NewJellyfinService() = nil, want non-nil")
	}

	if svc.manager != manager {
		t.Error("manager not assigned correctly")
	}

	if svc.name != "jellyfin-manager" {
		t.Errorf("expected name 'jellyfin-manager', got %q", svc.name)
	}
}

// --- Test: NewJellyfinServiceWithName ---

func TestNewJellyfinServiceWithName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		customName   string
		expectedName string
	}{
		{
			name:         "simple custom name",
			customName:   "jellyfin-home",
			expectedName: "jellyfin-home",
		},
		{
			name:         "auto-generated server ID",
			customName:   "jellyfin-abc12345",
			expectedName: "jellyfin-abc12345",
		},
		{
			name:         "descriptive name",
			customName:   "jellyfin-living-room-server",
			expectedName: "jellyfin-living-room-server",
		},
		{
			name:         "empty name",
			customName:   "",
			expectedName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			manager := newMockJellyfinManager()
			svc := NewJellyfinServiceWithName(manager, tt.customName)

			if svc == nil {
				t.Fatal("NewJellyfinServiceWithName() = nil, want non-nil")
			}

			if svc.manager != manager {
				t.Error("manager not assigned correctly")
			}

			if svc.name != tt.expectedName {
				t.Errorf("expected name %q, got %q", tt.expectedName, svc.name)
			}
		})
	}
}

// --- Test: JellyfinService.Serve ---

func TestJellyfinService_Serve(t *testing.T) {
	t.Parallel()

	t.Run("calls manager Start and waits for context", func(t *testing.T) {
		t.Parallel()

		manager := newMockJellyfinManager()
		svc := NewJellyfinService(manager)

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)

		go func() {
			errCh <- svc.Serve(ctx)
		}()

		// Wait for Start to be called
		select {
		case <-manager.startStarted:
		case <-time.After(time.Second):
			t.Fatal("manager Start was not called")
		}

		// Cancel context to trigger shutdown
		cancel()

		select {
		case err := <-errCh:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("Serve() error = %v, want context.Canceled", err)
			}
		case <-time.After(time.Second):
			t.Error("Serve() did not return after context cancellation")
		}

		if manager.StartCallCount() != 1 {
			t.Errorf("Start called %d times, want 1", manager.StartCallCount())
		}
		if manager.StopCallCount() != 1 {
			t.Errorf("Stop called %d times, want 1", manager.StopCallCount())
		}
	})

	t.Run("propagates Start error", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("jellyfin connection failed")
		manager := newMockJellyfinManager()
		manager.startErr = expectedErr
		svc := NewJellyfinService(manager)

		err := svc.Serve(context.Background())

		if err == nil {
			t.Fatal("Serve() error = nil, want error")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("Serve() error = %v, want error wrapping %v", err, expectedErr)
		}
		// Stop should not be called if Start fails
		if manager.StopCallCount() != 0 {
			t.Errorf("Stop called %d times, want 0 (Start failed)", manager.StopCallCount())
		}
	})

	t.Run("propagates Stop error", func(t *testing.T) {
		t.Parallel()

		stopErr := errors.New("jellyfin stop failed")
		manager := newMockJellyfinManager()
		manager.stopErr = stopErr
		svc := NewJellyfinService(manager)

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)

		go func() {
			errCh <- svc.Serve(ctx)
		}()

		// Wait for Start to be called
		select {
		case <-manager.startStarted:
		case <-time.After(time.Second):
			t.Fatal("manager Start was not called")
		}

		// Cancel to trigger Stop
		cancel()

		select {
		case err := <-errCh:
			if !errors.Is(err, stopErr) {
				t.Errorf("Serve() error = %v, want error wrapping %v", err, stopErr)
			}
		case <-time.After(time.Second):
			t.Error("Serve() did not return")
		}
	})

	t.Run("returns on context deadline exceeded", func(t *testing.T) {
		t.Parallel()

		manager := newMockJellyfinManager()
		svc := NewJellyfinService(manager)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := svc.Serve(ctx)

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Serve() error = %v, want context.DeadlineExceeded", err)
		}
	})
}

// --- Test: JellyfinService.String ---

func TestJellyfinService_String(t *testing.T) {
	t.Parallel()

	t.Run("default name", func(t *testing.T) {
		t.Parallel()

		manager := newMockJellyfinManager()
		svc := NewJellyfinService(manager)

		if got := svc.String(); got != "jellyfin-manager" {
			t.Errorf("String() = %q, want 'jellyfin-manager'", got)
		}
	})

	t.Run("custom name", func(t *testing.T) {
		t.Parallel()

		manager := newMockJellyfinManager()
		svc := NewJellyfinServiceWithName(manager, "jellyfin-home-server")

		if got := svc.String(); got != "jellyfin-home-server" {
			t.Errorf("String() = %q, want 'jellyfin-home-server'", got)
		}
	})
}

// --- Test: JellyfinService with Suture supervisor ---

func TestJellyfinService_WithSupervisor(t *testing.T) {
	t.Parallel()

	manager := newMockJellyfinManager()
	svc := NewJellyfinService(manager)

	sup := suture.New("test-sup", suture.Spec{
		FailureThreshold: 3,
		FailureBackoff:   10 * time.Millisecond,
		Timeout:          2 * time.Second,
	})
	sup.Add(svc)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	errCh := sup.ServeBackground(ctx)

	// Wait for manager to start
	select {
	case <-manager.startStarted:
	case <-time.After(time.Second):
		t.Fatal("manager did not start under supervisor")
	}

	if manager.StartCallCount() < 1 {
		t.Error("Start was not called")
	}

	cancel()
	<-errCh

	// Verify Stop was called during shutdown
	if manager.StopCallCount() < 1 {
		t.Error("Stop was not called during supervisor shutdown")
	}
}

func TestJellyfinService_RestartOnError(t *testing.T) {
	t.Parallel()

	manager := newMockJellyfinManager()
	manager.startErr = errors.New("transient error")
	svc := NewJellyfinService(manager)

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
	if manager.StartCallCount() < 2 {
		t.Errorf("expected multiple restarts, got %d runs", manager.StartCallCount())
	}
}

// --- Test: NewEmbyService ---

func TestNewEmbyService(t *testing.T) {
	t.Parallel()

	manager := newMockEmbyManager()
	svc := NewEmbyService(manager)

	if svc == nil {
		t.Fatal("NewEmbyService() = nil, want non-nil")
	}

	if svc.manager != manager {
		t.Error("manager not assigned correctly")
	}

	if svc.name != "emby-manager" {
		t.Errorf("expected name 'emby-manager', got %q", svc.name)
	}
}

// --- Test: NewEmbyServiceWithName ---

func TestNewEmbyServiceWithName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		customName   string
		expectedName string
	}{
		{
			name:         "simple custom name",
			customName:   "emby-home",
			expectedName: "emby-home",
		},
		{
			name:         "auto-generated server ID",
			customName:   "emby-def67890",
			expectedName: "emby-def67890",
		},
		{
			name:         "descriptive name",
			customName:   "emby-bedroom-server",
			expectedName: "emby-bedroom-server",
		},
		{
			name:         "empty name",
			customName:   "",
			expectedName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			manager := newMockEmbyManager()
			svc := NewEmbyServiceWithName(manager, tt.customName)

			if svc == nil {
				t.Fatal("NewEmbyServiceWithName() = nil, want non-nil")
			}

			if svc.manager != manager {
				t.Error("manager not assigned correctly")
			}

			if svc.name != tt.expectedName {
				t.Errorf("expected name %q, got %q", tt.expectedName, svc.name)
			}
		})
	}
}

// --- Test: EmbyService.Serve ---

func TestEmbyService_Serve(t *testing.T) {
	t.Parallel()

	t.Run("calls manager Start and waits for context", func(t *testing.T) {
		t.Parallel()

		manager := newMockEmbyManager()
		svc := NewEmbyService(manager)

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)

		go func() {
			errCh <- svc.Serve(ctx)
		}()

		// Wait for Start to be called
		select {
		case <-manager.startStarted:
		case <-time.After(time.Second):
			t.Fatal("manager Start was not called")
		}

		// Cancel context to trigger shutdown
		cancel()

		select {
		case err := <-errCh:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("Serve() error = %v, want context.Canceled", err)
			}
		case <-time.After(time.Second):
			t.Error("Serve() did not return after context cancellation")
		}

		if manager.StartCallCount() != 1 {
			t.Errorf("Start called %d times, want 1", manager.StartCallCount())
		}
		if manager.StopCallCount() != 1 {
			t.Errorf("Stop called %d times, want 1", manager.StopCallCount())
		}
	})

	t.Run("propagates Start error", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("emby connection failed")
		manager := newMockEmbyManager()
		manager.startErr = expectedErr
		svc := NewEmbyService(manager)

		err := svc.Serve(context.Background())

		if err == nil {
			t.Fatal("Serve() error = nil, want error")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("Serve() error = %v, want error wrapping %v", err, expectedErr)
		}
		// Stop should not be called if Start fails
		if manager.StopCallCount() != 0 {
			t.Errorf("Stop called %d times, want 0 (Start failed)", manager.StopCallCount())
		}
	})

	t.Run("propagates Stop error", func(t *testing.T) {
		t.Parallel()

		stopErr := errors.New("emby stop failed")
		manager := newMockEmbyManager()
		manager.stopErr = stopErr
		svc := NewEmbyService(manager)

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)

		go func() {
			errCh <- svc.Serve(ctx)
		}()

		// Wait for Start to be called
		select {
		case <-manager.startStarted:
		case <-time.After(time.Second):
			t.Fatal("manager Start was not called")
		}

		// Cancel to trigger Stop
		cancel()

		select {
		case err := <-errCh:
			if !errors.Is(err, stopErr) {
				t.Errorf("Serve() error = %v, want error wrapping %v", err, stopErr)
			}
		case <-time.After(time.Second):
			t.Error("Serve() did not return")
		}
	})

	t.Run("returns on context deadline exceeded", func(t *testing.T) {
		t.Parallel()

		manager := newMockEmbyManager()
		svc := NewEmbyService(manager)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := svc.Serve(ctx)

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Serve() error = %v, want context.DeadlineExceeded", err)
		}
	})
}

// --- Test: EmbyService.String ---

func TestEmbyService_String(t *testing.T) {
	t.Parallel()

	t.Run("default name", func(t *testing.T) {
		t.Parallel()

		manager := newMockEmbyManager()
		svc := NewEmbyService(manager)

		if got := svc.String(); got != "emby-manager" {
			t.Errorf("String() = %q, want 'emby-manager'", got)
		}
	})

	t.Run("custom name", func(t *testing.T) {
		t.Parallel()

		manager := newMockEmbyManager()
		svc := NewEmbyServiceWithName(manager, "emby-home-server")

		if got := svc.String(); got != "emby-home-server" {
			t.Errorf("String() = %q, want 'emby-home-server'", got)
		}
	})
}

// --- Test: EmbyService with Suture supervisor ---

func TestEmbyService_WithSupervisor(t *testing.T) {
	t.Parallel()

	manager := newMockEmbyManager()
	svc := NewEmbyService(manager)

	sup := suture.New("test-sup", suture.Spec{
		FailureThreshold: 3,
		FailureBackoff:   10 * time.Millisecond,
		Timeout:          2 * time.Second,
	})
	sup.Add(svc)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	errCh := sup.ServeBackground(ctx)

	// Wait for manager to start
	select {
	case <-manager.startStarted:
	case <-time.After(time.Second):
		t.Fatal("manager did not start under supervisor")
	}

	if manager.StartCallCount() < 1 {
		t.Error("Start was not called")
	}

	cancel()
	<-errCh

	// Verify Stop was called during shutdown
	if manager.StopCallCount() < 1 {
		t.Error("Stop was not called during supervisor shutdown")
	}
}

func TestEmbyService_RestartOnError(t *testing.T) {
	t.Parallel()

	manager := newMockEmbyManager()
	manager.startErr = errors.New("transient error")
	svc := NewEmbyService(manager)

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
	if manager.StartCallCount() < 2 {
		t.Errorf("expected multiple restarts, got %d runs", manager.StartCallCount())
	}
}

// --- Test: Multiple servers in supervisor ---

func TestMultipleJellyfinServices(t *testing.T) {
	t.Parallel()

	mgr1 := newMockJellyfinManager()
	mgr2 := newMockJellyfinManager()
	mgr3 := newMockJellyfinManager()

	svc1 := NewJellyfinServiceWithName(mgr1, "jellyfin-home")
	svc2 := NewJellyfinServiceWithName(mgr2, "jellyfin-office")
	svc3 := NewJellyfinServiceWithName(mgr3, "jellyfin-basement")

	names := map[string]bool{
		svc1.String(): true,
		svc2.String(): true,
		svc3.String(): true,
	}

	// Verify all names are unique
	if len(names) != 3 {
		t.Error("service names are not unique")
	}

	// Verify specific names
	expectedNames := []string{"jellyfin-home", "jellyfin-office", "jellyfin-basement"}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("expected name %q not found in service names", name)
		}
	}
}

func TestMultipleEmbyServices(t *testing.T) {
	t.Parallel()

	mgr1 := newMockEmbyManager()
	mgr2 := newMockEmbyManager()
	mgr3 := newMockEmbyManager()

	svc1 := NewEmbyServiceWithName(mgr1, "emby-primary")
	svc2 := NewEmbyServiceWithName(mgr2, "emby-secondary")
	svc3 := NewEmbyServiceWithName(mgr3, "emby-backup")

	names := map[string]bool{
		svc1.String(): true,
		svc2.String(): true,
		svc3.String(): true,
	}

	// Verify all names are unique
	if len(names) != 3 {
		t.Error("service names are not unique")
	}

	// Verify specific names
	expectedNames := []string{"emby-primary", "emby-secondary", "emby-backup"}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("expected name %q not found in service names", name)
		}
	}
}

func TestMultipleServers_WithSupervisor(t *testing.T) {
	t.Parallel()

	jellyMgr1 := newMockJellyfinManager()
	jellyMgr2 := newMockJellyfinManager()
	embyMgr := newMockEmbyManager()

	jellySvc1 := NewJellyfinServiceWithName(jellyMgr1, "jellyfin-primary")
	jellySvc2 := NewJellyfinServiceWithName(jellyMgr2, "jellyfin-secondary")
	embySvc := NewEmbyServiceWithName(embyMgr, "emby-main")

	sup := suture.New("test-sup", suture.Spec{
		FailureThreshold: 3,
		FailureBackoff:   10 * time.Millisecond,
		Timeout:          2 * time.Second,
	})
	sup.Add(jellySvc1)
	sup.Add(jellySvc2)
	sup.Add(embySvc)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	errCh := sup.ServeBackground(ctx)

	// Wait for all managers to start
	select {
	case <-jellyMgr1.startStarted:
	case <-time.After(time.Second):
		t.Fatal("jellyMgr1 did not start")
	}

	select {
	case <-jellyMgr2.startStarted:
	case <-time.After(time.Second):
		t.Fatal("jellyMgr2 did not start")
	}

	select {
	case <-embyMgr.startStarted:
	case <-time.After(time.Second):
		t.Fatal("embyMgr did not start")
	}

	// Verify unique names
	names := map[string]bool{
		jellySvc1.String(): true,
		jellySvc2.String(): true,
		embySvc.String():   true,
	}
	if len(names) != 3 {
		t.Errorf("services should have 3 unique names, got %d", len(names))
	}

	cancel()
	<-errCh

	// Verify all were stopped
	if jellyMgr1.StopCallCount() < 1 {
		t.Error("jellyMgr1 Stop was not called")
	}
	if jellyMgr2.StopCallCount() < 1 {
		t.Error("jellyMgr2 Stop was not called")
	}
	if embyMgr.StopCallCount() < 1 {
		t.Error("embyMgr Stop was not called")
	}
}
