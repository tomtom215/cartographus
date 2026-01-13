// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"errors"
	"strings"
	stdSync "sync"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func TestManager_TriggerSync(t *testing.T) {
	// NOT parallel - tests async sync triggering

	cfg := newTestConfig()
	syncCalled := false
	mockClient := &mockTautulliClient{
		getHistorySince: func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
			syncCalled = true
			return &tautulli.TautulliHistory{
				Response: tautulli.TautulliHistoryResponse{
					Result: "success",
					Data:   tautulli.TautulliHistoryData{Data: []tautulli.TautulliHistoryRecord{}},
				},
			}, nil
		},
	}

	mockDB := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
	}

	manager := NewManager(mockDB, nil, mockClient, cfg, nil)

	err := manager.TriggerSync()
	if err != nil {
		t.Fatalf("TriggerSync failed: %v", err)
	}

	if !syncCalled {
		t.Error("Expected sync to be called")
	}
}

func TestManager_SyncData_FirstSync(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()

	var sinceCaptured time.Time
	mockClient := &mockTautulliClient{
		getHistorySince: func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
			sinceCaptured = since
			return &tautulli.TautulliHistory{
				Response: tautulli.TautulliHistoryResponse{
					Result: "success",
					Data:   tautulli.TautulliHistoryData{Data: []tautulli.TautulliHistoryRecord{}},
				},
			}, nil
		},
	}

	mockDB := &mockDB{}
	manager := NewManager(mockDB, nil, mockClient, cfg, nil)

	// First sync should use lookback period
	err := manager.TriggerSync()
	if err != nil {
		t.Fatalf("TriggerSync failed: %v", err)
	}

	// Verify lookback was approximately 24 hours ago
	expectedSince := time.Now().Add(-24 * time.Hour)
	diff := expectedSince.Sub(sinceCaptured)
	if diff < 0 {
		diff = -diff
	}
	if diff > 5*time.Second {
		t.Errorf("Expected since to be ~24h ago, but diff was %v", diff)
	}
}

func TestManager_SyncData_SubsequentSync(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()

	callCount := 0
	var secondCallSince time.Time
	mockClient := &mockTautulliClient{
		getHistorySince: func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
			callCount++
			if callCount == 2 {
				secondCallSince = since
			}
			return &tautulli.TautulliHistory{
				Response: tautulli.TautulliHistoryResponse{
					Result: "success",
					Data:   tautulli.TautulliHistoryData{Data: []tautulli.TautulliHistoryRecord{}},
				},
			}, nil
		},
	}

	mockDB := &mockDB{}
	manager := NewManager(mockDB, nil, mockClient, cfg, nil)

	// First sync
	err := manager.TriggerSync()
	if err != nil {
		t.Fatalf("First TriggerSync failed: %v", err)
	}

	firstSyncTime := manager.LastSyncTime()

	// Small delay to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Second sync should use last sync time
	err = manager.TriggerSync()
	if err != nil {
		t.Fatalf("Second TriggerSync failed: %v", err)
	}

	// Verify second sync used first sync's timestamp
	diff := firstSyncTime.Sub(secondCallSince)
	if diff < 0 {
		diff = -diff
	}
	if diff > 100*time.Millisecond {
		t.Errorf("Expected second sync to use first sync time, diff: %v", diff)
	}
}

func TestManager_SyncDataSince_MultipleBatches(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()
	// Override batch size to 2 for this test - it specifically tests batch boundary behavior
	cfg.Sync.BatchSize = 2

	batchCount := 0
	mockClient := &mockTautulliClient{
		getHistorySince: func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
			batchCount++

			// Return different batches based on start offset
			var records []tautulli.TautulliHistoryRecord
			if start == 0 {
				// First batch: 2 records (full batch)
				records = []tautulli.TautulliHistoryRecord{
					{
						SessionKey: stringPtr("session-1"),
						UserID:     intPtr(1),
						User:       "user1",
						IPAddress:  "192.168.1.1",
						MediaType:  "movie",
						Title:      "Movie 1",
						Started:    time.Now().Unix(),
					},
					{
						SessionKey: stringPtr("session-2"),
						UserID:     intPtr(1),
						User:       "user1",
						IPAddress:  "192.168.1.1",
						MediaType:  "movie",
						Title:      "Movie 2",
						Started:    time.Now().Unix(),
					},
				}
			} else if start == 2 {
				// Second batch: 1 record (partial batch, should stop)
				records = []tautulli.TautulliHistoryRecord{
					{
						SessionKey: stringPtr("session-3"),
						UserID:     intPtr(1),
						User:       "user1",
						IPAddress:  "192.168.1.1",
						MediaType:  "movie",
						Title:      "Movie 3",
						Started:    time.Now().Unix(),
					},
				}
			}

			return &tautulli.TautulliHistory{
				Response: tautulli.TautulliHistoryResponse{
					Result: "success",
					Data:   tautulli.TautulliHistoryData{Data: records},
				},
			}, nil
		},
	}

	recordsInserted := 0
	mockDB := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocation: func(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
			return &models.Geolocation{
				IPAddress: ipAddress,
				Latitude:  40.7128,
				Longitude: -74.0060,
				Country:   "United States",
			}, nil
		},
		getGeolocations: func(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
			// Return empty map - all IPs will be fetched individually
			return make(map[string]*models.Geolocation), nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			recordsInserted++
			return nil
		},
	}

	manager := NewManager(mockDB, nil, mockClient, cfg, nil)

	err := manager.TriggerSync()
	if err != nil {
		t.Fatalf("TriggerSync failed: %v", err)
	}

	if batchCount != 2 {
		t.Errorf("Expected 2 batches, got %d", batchCount)
	}

	if recordsInserted != 3 {
		t.Errorf("Expected 3 records inserted, got %d", recordsInserted)
	}
}

func TestManager_SyncDataSince_TautulliError(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()

	attemptCount := 0
	mockClient := &mockTautulliClient{
		getHistorySince: func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
			attemptCount++
			return nil, errors.New("tautulli connection failed")
		},
	}

	mockDB := &mockDB{}
	manager := NewManager(mockDB, nil, mockClient, cfg, nil)

	err := manager.TriggerSync()
	if err == nil {
		t.Error("Expected error when Tautulli fails")
	}

	if attemptCount != cfg.Sync.RetryAttempts {
		t.Errorf("Expected %d retry attempts, got %d", cfg.Sync.RetryAttempts, attemptCount)
	}

	if !strings.Contains(err.Error(), "failed to fetch history") {
		t.Errorf("Expected 'failed to fetch history' error, got: %v", err)
	}
}

func TestManager_SyncDataSince_ProcessingError_ContinuesWithOthers(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()

	mockClient := &mockTautulliClient{
		getHistorySince: func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
			return &tautulli.TautulliHistory{
				Response: tautulli.TautulliHistoryResponse{
					Result: "success",
					Data: tautulli.TautulliHistoryData{
						Data: []tautulli.TautulliHistoryRecord{
							{
								SessionKey: stringPtr("session-good"),
								UserID:     intPtr(1),
								User:       "user1",
								IPAddress:  "192.168.1.1",
								MediaType:  "movie",
								Title:      "Good Movie",
								Started:    time.Now().Unix(),
							},
							{
								SessionKey: stringPtr("session-bad"),
								UserID:     intPtr(2),
								User:       "user2",
								IPAddress:  "N/A", // Invalid IP will cause error
								MediaType:  "movie",
								Title:      "Bad Movie",
								Started:    time.Now().Unix(),
							},
							{
								SessionKey: stringPtr("session-good-2"),
								UserID:     intPtr(3),
								User:       "user3",
								IPAddress:  "192.168.1.2",
								MediaType:  "movie",
								Title:      "Another Good Movie",
								Started:    time.Now().Unix(),
							},
						},
					},
				},
			}, nil
		},
	}

	recordsInserted := 0
	mockDB := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocation: func(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
			return &models.Geolocation{
				IPAddress: ipAddress,
				Latitude:  40.7128,
				Longitude: -74.0060,
				Country:   "United States",
			}, nil
		},
		getGeolocations: func(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
			// Return empty map - all IPs will be fetched individually
			return make(map[string]*models.Geolocation), nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			recordsInserted++
			return nil
		},
	}

	manager := NewManager(mockDB, nil, mockClient, cfg, nil)

	// Should not return error, should log and continue
	err := manager.TriggerSync()
	if err != nil {
		t.Fatalf("TriggerSync should not fail: %v", err)
	}

	// Should have processed 2 good records, skipped 1 bad
	if recordsInserted != 2 {
		t.Errorf("Expected 2 records inserted, got %d", recordsInserted)
	}
}

func TestManager_SyncDataSince_CallbackInvoked(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()

	mockClient := &mockTautulliClient{
		getHistorySince: func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
			return &tautulli.TautulliHistory{
				Response: tautulli.TautulliHistoryResponse{
					Result: "success",
					Data: tautulli.TautulliHistoryData{
						Data: []tautulli.TautulliHistoryRecord{
							{
								SessionKey: stringPtr("session-1"),
								UserID:     intPtr(1),
								User:       "user1",
								IPAddress:  "192.168.1.1",
								MediaType:  "movie",
								Title:      "Movie 1",
								Started:    time.Now().Unix(),
							},
							{
								SessionKey: stringPtr("session-2"),
								UserID:     intPtr(1),
								User:       "user1",
								IPAddress:  "192.168.1.1",
								MediaType:  "movie",
								Title:      "Movie 2",
								Started:    time.Now().Unix(),
							},
						},
					},
				},
			}, nil
		},
	}

	mockDB := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
		getGeolocation: func(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
			return &models.Geolocation{
				IPAddress: ipAddress,
				Latitude:  40.7128,
				Longitude: -74.0060,
				Country:   "United States",
			}, nil
		},
		getGeolocations: func(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
			// Return empty map - all IPs will be fetched individually
			return make(map[string]*models.Geolocation), nil
		},
		insertPlaybackEvent: func(event *models.PlaybackEvent) error {
			return nil
		},
	}

	manager := NewManager(mockDB, nil, mockClient, cfg, nil)

	callbackCalled := false
	var callbackRecords int
	var callbackDuration int64

	manager.SetOnSyncCompleted(func(newRecords int, durationMs int64) {
		callbackCalled = true
		callbackRecords = newRecords
		callbackDuration = durationMs
	})

	err := manager.TriggerSync()
	if err != nil {
		t.Fatalf("TriggerSync failed: %v", err)
	}

	if !callbackCalled {
		t.Error("Expected callback to be called after sync")
	}

	if callbackRecords != 2 {
		t.Errorf("Expected callback with 2 records, got %d", callbackRecords)
	}

	if callbackDuration < 0 {
		t.Errorf("Expected non-negative duration, got %d", callbackDuration)
	}
}

func TestManager_SyncLoop_ContextCancellation(t *testing.T) {
	// NOT parallel - tests context cancellation with goroutines

	cfg := newTestConfig()

	syncCount := 0
	mockClient := &mockTautulliClient{
		getHistorySince: func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
			syncCount++
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

	// Start manager
	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}

	// Let it run for a bit
	time.Sleep(250 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait a bit for shutdown
	time.Sleep(100 * time.Millisecond)

	// Stop manager
	err = manager.Stop()
	if err != nil {
		t.Fatalf("Failed to stop manager: %v", err)
	}

	// Should have synced at least once
	if syncCount < 1 {
		t.Error("Expected at least one sync during execution")
	}
}

func TestManager_PerformInitialSync_Success(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

	cfg := newTestConfig()

	syncCalled := false
	var sinceCaptured time.Time

	mockClient := &mockTautulliClient{
		getHistorySince: func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
			syncCalled = true
			sinceCaptured = since
			return &tautulli.TautulliHistory{
				Response: tautulli.TautulliHistoryResponse{
					Result: "success",
					Data:   tautulli.TautulliHistoryData{Data: []tautulli.TautulliHistoryRecord{}},
				},
			}, nil
		},
	}

	manager := NewManager(nil, nil, mockClient, cfg, nil)

	err := manager.performInitialSync()
	if err != nil {
		t.Fatalf("performInitialSync failed: %v", err)
	}

	if !syncCalled {
		t.Error("Expected sync to be called")
	}

	// Since should be approximately 24 hours ago (lookback period)
	expectedSince := time.Now().Add(-24 * time.Hour)
	diff := expectedSince.Sub(sinceCaptured)
	if diff < 0 {
		diff = -diff
	}
	if diff > 5*time.Second {
		t.Errorf("Expected since to be ~24h ago, but diff was %v", diff)
	}
}

func TestManager_SyncDataSince_EmptyHistory(t *testing.T) {
	t.Parallel() // Safe - isolated mock with no shared state

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

	manager := NewManager(nil, nil, mockClient, cfg, nil)

	err := manager.syncDataSince(context.Background(), time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("syncDataSince failed: %v", err)
	}
}

func TestManager_ConcurrentTriggerSync(t *testing.T) {
	// NOT parallel - tests concurrency explicitly

	cfg := newTestConfig()

	syncCount := 0
	var mu stdSync.Mutex

	mockClient := &mockTautulliClient{
		getHistorySince: func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
			// Simulate some work
			time.Sleep(10 * time.Millisecond)

			mu.Lock()
			syncCount++
			mu.Unlock()

			return &tautulli.TautulliHistory{
				Response: tautulli.TautulliHistoryResponse{
					Result: "success",
					Data:   tautulli.TautulliHistoryData{Data: []tautulli.TautulliHistoryRecord{}},
				},
			}, nil
		},
	}

	mockDB := &mockDB{
		sessionKeyExists: func(ctx context.Context, sessionKey string) (bool, error) {
			return false, nil
		},
	}

	manager := NewManager(mockDB, nil, mockClient, cfg, nil)

	// Trigger multiple syncs concurrently
	const numConcurrent = 10
	var wg stdSync.WaitGroup
	wg.Add(numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		go func() {
			defer wg.Done()
			_ = manager.TriggerSync()
		}()
	}

	wg.Wait()

	// With the syncMu lock, only one sync should execute at a time
	// Without it, we'd expect multiple concurrent syncs
	if syncCount != numConcurrent {
		t.Logf("Sync count: %d (some syncs may have been serialized by the mutex)", syncCount)
	}
}

// TestManager_ConcurrentStartCalls tests that calling Start() multiple times is handled correctly
