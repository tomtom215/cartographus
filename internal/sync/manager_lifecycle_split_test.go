// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func TestNewManager(t *testing.T) {
	t.Parallel() // Safe - isolated test with no shared state

	cfg := newTestConfig()
	mockClient := &mockTautulliClient{}
	manager := NewManager(nil, nil, mockClient, cfg, nil)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.cfg != cfg {
		t.Error("Config not set correctly")
	}

	if manager.running {
		t.Error("Manager should not be running initially")
	}

	if manager.stopChan == nil {
		t.Error("Stop channel not initialized")
	}
}

func TestManager_SetOnSyncCompleted(t *testing.T) {
	t.Parallel() // Safe - isolated test with no shared state

	cfg := newTestConfig()
	manager := NewManager(nil, nil, nil, cfg, nil)

	callbackCalled := false
	callback := func(newRecords int, durationMs int64) {
		callbackCalled = true
		if newRecords != 42 {
			t.Errorf("Expected newRecords=42, got %d", newRecords)
		}
		if durationMs != 1000 {
			t.Errorf("Expected durationMs=1000, got %d", durationMs)
		}
	}

	manager.SetOnSyncCompleted(callback)

	// Trigger callback
	manager.mu.RLock()
	if manager.onSyncCompleted != nil {
		manager.onSyncCompleted(42, 1000)
	}
	manager.mu.RUnlock()

	if !callbackCalled {
		t.Error("Callback was not called")
	}
}

func TestManager_LastSyncTime(t *testing.T) {
	t.Parallel() // Safe - isolated test with no shared state

	cfg := newTestConfig()
	manager := NewManager(nil, nil, nil, cfg, nil)

	// Initially should be zero time
	if !manager.LastSyncTime().IsZero() {
		t.Error("Expected zero time initially")
	}

	// Set last sync time
	now := time.Now()
	manager.mu.Lock()
	manager.lastSync = now
	manager.mu.Unlock()

	lastSync := manager.LastSyncTime()
	if !lastSync.Equal(now) {
		t.Errorf("Expected last sync time %v, got %v", now, lastSync)
	}
}

func TestManager_StartStop(t *testing.T) {
	// NOT parallel - tests goroutine lifecycle with timing

	cfg := newTestConfig()
	cfg.Sync.Interval = 1 * time.Second // Override for faster test

	mockClient := &mockTautulliClient{
		getHistorySince: func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
			return &tautulli.TautulliHistory{
				Response: tautulli.TautulliHistoryResponse{
					Result: "success",
					Data:   tautulli.TautulliHistoryData{Data: []tautulli.TautulliHistoryRecord{}},
				},
			}, nil
		},
	}

	manager := NewManager(nil, nil, mockClient, cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test starting manager
	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}

	// Verify running state
	manager.mu.RLock()
	running := manager.running
	manager.mu.RUnlock()

	if !running {
		t.Error("Manager should be running after Start()")
	}

	// Test starting already running manager
	err = manager.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running manager")
	}

	// Wait a bit for goroutines to start
	time.Sleep(100 * time.Millisecond)

	// Test stopping manager
	err = manager.Stop()
	if err != nil {
		t.Fatalf("Failed to stop manager: %v", err)
	}

	// Verify stopped state
	manager.mu.RLock()
	running = manager.running
	manager.mu.RUnlock()

	if running {
		t.Error("Manager should not be running after Stop()")
	}

	// Test stopping already stopped manager
	err = manager.Stop()
	if err == nil {
		t.Error("Expected error when stopping already stopped manager")
	}
}
