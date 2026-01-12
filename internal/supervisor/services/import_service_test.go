// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// mockImporter implements ImporterInterface for testing.
type mockImporter struct {
	mu           sync.Mutex
	running      bool
	importErr    error
	importStats  interface{}
	importCalled bool
	stopCalled   bool
	importDelay  time.Duration
}

func newMockImporter() *mockImporter {
	return &mockImporter{
		importStats: map[string]int{"imported": 0},
	}
}

func (m *mockImporter) Import(ctx context.Context) (interface{}, error) {
	m.mu.Lock()
	m.importCalled = true
	m.running = true
	m.mu.Unlock()

	if m.importDelay > 0 {
		select {
		case <-time.After(m.importDelay):
		case <-ctx.Done():
			m.mu.Lock()
			m.running = false
			m.mu.Unlock()
			return m.importStats, ctx.Err()
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = false

	if m.importErr != nil {
		return nil, m.importErr
	}
	return m.importStats, nil
}

func (m *mockImporter) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

func (m *mockImporter) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopCalled = true
	if !m.running {
		return errors.New("no import in progress")
	}
	m.running = false
	return nil
}

func (m *mockImporter) setRunning(running bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = running
}

func (m *mockImporter) wasImportCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.importCalled
}

func (m *mockImporter) wasStopCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopCalled
}

// --- Tests ---

func TestNewImportService(t *testing.T) {
	importer := newMockImporter()

	t.Run("creates service with autoStart=true", func(t *testing.T) {
		svc := NewImportService(importer, true)

		if svc == nil {
			t.Fatal("NewImportService() returned nil")
		}
		if svc.name != "tautulli-import" {
			t.Errorf("name = %q, want 'tautulli-import'", svc.name)
		}
		if !svc.autoStart {
			t.Error("autoStart should be true")
		}
	})

	t.Run("creates service with autoStart=false", func(t *testing.T) {
		svc := NewImportService(importer, false)

		if svc == nil {
			t.Fatal("NewImportService() returned nil")
		}
		if svc.autoStart {
			t.Error("autoStart should be false")
		}
	})
}

func TestImportService_Serve_AutoStart(t *testing.T) {
	importer := newMockImporter()
	importer.importDelay = 50 * time.Millisecond

	svc := NewImportService(importer, true)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := svc.Serve(ctx)

	// Should return context error after import completes and waiting for shutdown
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Serve() error = %v, want %v", err, context.DeadlineExceeded)
	}

	// Import should have been called
	if !importer.wasImportCalled() {
		t.Error("Import() should have been called in autoStart mode")
	}
}

func TestImportService_Serve_AutoStart_ImportError(t *testing.T) {
	importer := newMockImporter()
	importer.importErr = errors.New("import failed")

	svc := NewImportService(importer, true)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := svc.Serve(ctx)

	// Should return wrapped import error
	if err == nil {
		t.Fatal("Serve() should return error when import fails")
	}
	if err.Error() != "import failed: import failed" {
		t.Errorf("Serve() error = %q, want 'import failed: import failed'", err.Error())
	}
}

func TestImportService_Serve_OnDemand(t *testing.T) {
	importer := newMockImporter()

	svc := NewImportService(importer, false)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := svc.Serve(ctx)

	// Should return context error (shutdown)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Serve() error = %v, want %v", err, context.DeadlineExceeded)
	}

	// Import should NOT have been called
	if importer.wasImportCalled() {
		t.Error("Import() should not be called in on-demand mode")
	}
}

func TestImportService_Serve_OnDemand_StopsRunningImport(t *testing.T) {
	importer := newMockImporter()
	importer.importDelay = 10 * time.Second // Long-running import

	svc := NewImportService(importer, false)

	ctx, cancel := context.WithCancel(context.Background())

	// Channel to signal when Serve has started and is waiting
	serveStarted := make(chan struct{})

	// Start Serve in background
	done := make(chan error, 1)
	go func() {
		// Signal that Serve is about to start
		close(serveStarted)
		done <- svc.Serve(ctx)
	}()

	// Wait for Serve to start
	<-serveStarted
	// Give Serve time to reach the <-ctx.Done() line
	time.Sleep(10 * time.Millisecond)

	// Simulate an import starting (e.g., triggered via API)
	importer.setRunning(true)

	// Poll to verify running state is set before canceling
	deadline := time.Now().Add(time.Second)
	for !importer.IsRunning() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !importer.IsRunning() {
		t.Fatal("importer.IsRunning() should be true before shutdown")
	}

	// Cancel context to trigger shutdown
	cancel()

	// Wait for Serve to finish
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Serve() error = %v, want %v", err, context.Canceled)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve() didn't stop within timeout")
	}

	// Stop should have been called
	if !importer.wasStopCalled() {
		t.Error("Stop() should be called when import is running during shutdown")
	}
}

func TestImportService_Serve_Shutdown(t *testing.T) {
	importer := newMockImporter()
	importer.importDelay = 10 * time.Second // Long-running import

	svc := NewImportService(importer, true)

	ctx, cancel := context.WithCancel(context.Background())

	// Start Serve in background
	done := make(chan error, 1)
	go func() {
		done <- svc.Serve(ctx)
	}()

	// Wait a bit for import to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for Serve to finish
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Serve() error = %v, want %v", err, context.Canceled)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve() didn't stop within timeout")
	}
}

func TestImportService_String(t *testing.T) {
	importer := newMockImporter()
	svc := NewImportService(importer, false)

	name := svc.String()

	if name != "tautulli-import" {
		t.Errorf("String() = %q, want 'tautulli-import'", name)
	}
}

func TestImportService_Importer(t *testing.T) {
	importer := newMockImporter()
	svc := NewImportService(importer, false)

	got := svc.Importer()

	if got != importer {
		t.Error("Importer() should return the underlying importer")
	}
}
