// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	stdSync "sync"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func TestManager_ConcurrentStartCalls(t *testing.T) {
	// NOT parallel - tests concurrency explicitly

	cfg := newTestConfig()

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

	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
	}

	manager := NewManager(mockDb, nil, mockClient, cfg, nil)

	// Try to start multiple times concurrently
	const numConcurrent = 5
	var wg stdSync.WaitGroup
	wg.Add(numConcurrent)

	errors := make([]error, numConcurrent)
	for i := 0; i < numConcurrent; i++ {
		go func(index int) {
			defer wg.Done()
			errors[index] = manager.Start(context.Background())
		}(i)
	}

	wg.Wait()

	// Clean up
	_ = manager.Stop()

	// Exactly one Start() should succeed, the rest should error
	successCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		}
	}

	if successCount != 1 {
		t.Errorf("Expected exactly 1 successful Start(), got %d", successCount)
	}
}

// TestManager_StopDuringSync tests that calling Stop() during an active sync doesn't cause panics

func TestManager_StopDuringSync(t *testing.T) {
	// NOT parallel - tests goroutine coordination

	cfg := newTestConfig()

	syncStarted := make(chan struct{})
	syncCanFinish := make(chan struct{})
	var syncStartedOnce stdSync.Once // Ensure channel is only closed once

	mockClient := &mockTautulliClient{
		getHistorySince: func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
			// Signal that sync has started (only once)
			syncStartedOnce.Do(func() {
				close(syncStarted)
			})
			<-syncCanFinish // Wait for test to call Stop()

			return &tautulli.TautulliHistory{
				Response: tautulli.TautulliHistoryResponse{
					Result: "success",
					Data:   tautulli.TautulliHistoryData{Data: []tautulli.TautulliHistoryRecord{}},
				},
			}, nil
		},
	}

	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
	}

	manager := NewManager(mockDb, nil, mockClient, cfg, nil)

	// Start manager
	err := manager.Start(context.Background())
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}

	// Trigger a sync
	go func() {
		_ = manager.TriggerSync()
	}()

	// Wait for sync to start
	<-syncStarted

	// Allow syncs to finish by closing the channel
	close(syncCanFinish)

	// Stop manager (goroutines can now complete)
	err = manager.Stop()

	if err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}
}

// TestManager_RaceConditions is a test that should be run with -race flag
// Run with: go test -race ./internal/sync -run TestManager_RaceConditions

func TestManager_RaceConditions(t *testing.T) {
	// NOT parallel - tests race conditions explicitly

	cfg := newTestConfig()

	mockClient := &mockTautulliClient{
		getHistorySince: func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
			// Simulate variable latency
			time.Sleep(time.Duration(1+start%5) * time.Millisecond)

			return &tautulli.TautulliHistory{
				Response: tautulli.TautulliHistoryResponse{
					Result: "success",
					Data:   tautulli.TautulliHistoryData{Data: []tautulli.TautulliHistoryRecord{}},
				},
			}, nil
		},
		getGeoIPLookup: func(ctx context.Context, ipAddress string) (*tautulli.TautulliGeoIP, error) {
			return &tautulli.TautulliGeoIP{
				Response: tautulli.TautulliGeoIPResponse{
					Result: "success",
					Data: tautulli.TautulliGeoIPData{
						Latitude:  37.7749,
						Longitude: -122.4194,
						Country:   "US",
					},
				},
			}, nil
		},
	}

	mockDb := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocation: func(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
			return nil, nil
		},
		upsertGeolocation: func(geo *models.Geolocation) error {
			return nil
		},
		getGeolocations: func(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
			// Return empty map - all IPs will be fetched individually
			return make(map[string]*models.Geolocation), nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			return nil
		},
	}

	manager := NewManager(mockDb, nil, mockClient, cfg, nil)

	// Start manager
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}

	// Perform various concurrent operations
	var wg stdSync.WaitGroup

	// Concurrent TriggerSync calls
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = manager.TriggerSync()
		}()
	}

	// Concurrent LastSyncTime reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = manager.LastSyncTime()
		}()
	}

	wg.Wait()

	// Stop manager
	err = manager.Stop()
	if err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}
}
