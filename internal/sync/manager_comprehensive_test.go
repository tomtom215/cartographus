// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
	ws "github.com/tomtom215/cartographus/internal/websocket"
)

// MockTautulliClient is a minimal mock implementing TautulliClientInterface for comprehensive tests
// All methods accept context.Context as the first parameter for cancellation support
type MockTautulliClient struct{}

func (m *MockTautulliClient) Ping(ctx context.Context) error {
	return nil
}

func (m *MockTautulliClient) GetHistorySince(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
	return &tautulli.TautulliHistory{
		Response: tautulli.TautulliHistoryResponse{
			Result: "success",
			Data:   tautulli.TautulliHistoryData{Data: []tautulli.TautulliHistoryRecord{}},
		},
	}, nil
}

func (m *MockTautulliClient) GetGeoIPLookup(ctx context.Context, ipAddress string) (*tautulli.TautulliGeoIP, error) {
	return &tautulli.TautulliGeoIP{
		Response: tautulli.TautulliGeoIPResponse{
			Result: "success",
			Data: tautulli.TautulliGeoIPData{
				Country:   "United States",
				Latitude:  37.7749,
				Longitude: -122.4194,
			},
		},
	}, nil
}

// Stub methods for remaining interface requirements (not used in comprehensive tests)
func (m *MockTautulliClient) GetHomeStats(ctx context.Context, timeRange int, statsType string, statsCount int) (*tautulli.TautulliHomeStats, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetPlaysByDate(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByDate, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetPlaysByDayOfWeek(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByDayOfWeek, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetPlaysByHourOfDay(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByHourOfDay, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetPlaysByStreamType(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByStreamType, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetConcurrentStreamsByStreamType(ctx context.Context, timeRange int, userID int) (*tautulli.TautulliConcurrentStreamsByStreamType, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetItemWatchTimeStats(ctx context.Context, ratingKey string, grouping int, queryDays string) (*tautulli.TautulliItemWatchTimeStats, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetActivity(ctx context.Context, sessionKey string) (*tautulli.TautulliActivity, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetMetadata(ctx context.Context, ratingKey string) (*tautulli.TautulliMetadata, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetUser(ctx context.Context, userID int) (*tautulli.TautulliUser, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetUsers(ctx context.Context) (*tautulli.TautulliUsers, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetLibraryUserStats(ctx context.Context, sectionID int, grouping int) (*tautulli.TautulliLibraryUserStats, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetRecentlyAdded(ctx context.Context, count int, start int, mediaType string, sectionID int) (*tautulli.TautulliRecentlyAdded, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetLibraries(ctx context.Context) (*tautulli.TautulliLibraries, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetLibrary(ctx context.Context, sectionID int) (*tautulli.TautulliLibrary, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetServerInfo(ctx context.Context) (*tautulli.TautulliServerInfo, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetSyncedItems(ctx context.Context, machineID string, userID int) (*tautulli.TautulliSyncedItems, error) {
	return nil, nil
}
func (m *MockTautulliClient) TerminateSession(ctx context.Context, sessionID string, message string) (*tautulli.TautulliTerminateSession, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetPlaysBySourceResolution(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysBySourceResolution, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetPlaysByStreamResolution(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByStreamResolution, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetPlaysByTop10Platforms(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Platforms, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetPlaysByTop10Users(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Users, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetPlaysPerMonth(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysPerMonth, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetUserPlayerStats(ctx context.Context, userID int) (*tautulli.TautulliUserPlayerStats, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetUserWatchTimeStats(ctx context.Context, userID int, queryDays string) (*tautulli.TautulliUserWatchTimeStats, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetItemUserStats(ctx context.Context, ratingKey string, grouping int) (*tautulli.TautulliItemUserStats, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetLibrariesTable(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliLibrariesTable, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetLibraryMediaInfo(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int) (*tautulli.TautulliLibraryMediaInfo, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetLibraryWatchTimeStats(ctx context.Context, sectionID int, grouping int, queryDays string) (*tautulli.TautulliLibraryWatchTimeStats, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetChildrenMetadata(ctx context.Context, ratingKey string, mediaType string) (*tautulli.TautulliChildrenMetadata, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetUserIPs(ctx context.Context, userID int) (*tautulli.TautulliUserIPs, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetUsersTable(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUsersTable, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetUserLogins(ctx context.Context, userID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUserLogins, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetStreamData(ctx context.Context, rowID int, sessionKey string) (*tautulli.TautulliStreamData, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetLibraryNames(ctx context.Context) (*tautulli.TautulliLibraryNames, error) {
	return nil, nil
}
func (m *MockTautulliClient) ExportMetadata(ctx context.Context, sectionID int, exportType string, userID int, ratingKey string, fileFormat string) (*tautulli.TautulliExportMetadata, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetExportFields(ctx context.Context, mediaType string) (*tautulli.TautulliExportFields, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetStreamTypeByTop10Users(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliStreamTypeByTop10Users, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetStreamTypeByTop10Platforms(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliStreamTypeByTop10Platforms, error) {
	return nil, nil
}
func (m *MockTautulliClient) Search(ctx context.Context, query string, limit int) (*tautulli.TautulliSearch, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetNewRatingKeys(ctx context.Context, ratingKey string) (*tautulli.TautulliNewRatingKeys, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetOldRatingKeys(ctx context.Context, ratingKey string) (*tautulli.TautulliOldRatingKeys, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetCollectionsTable(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliCollectionsTable, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetPlaylistsTable(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliPlaylistsTable, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetServerFriendlyName(ctx context.Context) (*tautulli.TautulliServerFriendlyName, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetServerID(ctx context.Context) (*tautulli.TautulliServerID, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetServerIdentity(ctx context.Context) (*tautulli.TautulliServerIdentity, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetExportsTable(ctx context.Context, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliExportsTable, error) {
	return nil, nil
}
func (m *MockTautulliClient) DownloadExport(ctx context.Context, exportID int) (*tautulli.TautulliDownloadExport, error) {
	return nil, nil
}
func (m *MockTautulliClient) DeleteExport(ctx context.Context, exportID int) (*tautulli.TautulliDeleteExport, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetTautulliInfo(ctx context.Context) (*tautulli.TautulliTautulliInfo, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetServerPref(ctx context.Context, pref string) (*tautulli.TautulliServerPref, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetServerList(ctx context.Context) (*tautulli.TautulliServerList, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetServersInfo(ctx context.Context) (*tautulli.TautulliServersInfo, error) {
	return nil, nil
}
func (m *MockTautulliClient) GetPMSUpdate(ctx context.Context) (*tautulli.TautulliPMSUpdate, error) {
	return nil, nil
}

// setupTestManager creates a test sync manager with mock dependencies
func setupTestManager(t *testing.T) (*Manager, *database.DB) {
	// Create test database
	dbCfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, err := database.New(dbCfg, 0.0, 0.0)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create WebSocket hub
	wsHub := ws.NewHub()
	go wsHub.Run()

	// Create mock client
	mockClient := &MockTautulliClient{}

	// Create full config with sync settings
	cfg := &config.Config{
		Tautulli: config.TautulliConfig{
			Enabled: true,
			URL:     "http://localhost:8181",
			APIKey:  "test-api-key",
		},
		Sync: config.SyncConfig{
			Interval:      30 * time.Minute,
			Lookback:      24 * time.Hour,
			BatchSize:     1000,
			RetryAttempts: 5,
			RetryDelay:    2 * time.Second,
		},
	}

	// Create manager with correct signature
	manager := NewManager(db, nil, mockClient, cfg, wsHub)

	return manager, db
}

// TestNewManagerComprehensive tests the manager constructor
func TestNewManagerComprehensive(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.client == nil {
		t.Error("Manager client is nil")
	}

	if manager.db == nil {
		t.Error("Manager db is nil")
	}

	if manager.wsHub == nil {
		t.Error("Manager wsHub is nil")
	}

	// Note: interval is now stored in cfg.Sync.Interval, not as a direct field
	// This test verifies manager was created successfully
}

// TestManagerStart tests starting the sync manager
func TestManagerStart(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()

	// Start should not block
	ctx := context.Background()
	done := make(chan bool)
	go func() {
		manager.Start(ctx)
		done <- true
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Should be running
	manager.mu.RLock()
	running := manager.running
	manager.mu.RUnlock()

	if !running {
		t.Error("Manager is not running after Start()")
	}

	// Stop after verifying it started (prevents race with defer)
	manager.Stop()
}

// TestManagerStop tests stopping the sync manager
func TestManagerStop(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()

	// Start the manager
	go manager.Start(context.Background())
	time.Sleep(100 * time.Millisecond)

	// Stop should work
	manager.Stop()

	// Give it a moment to stop
	time.Sleep(100 * time.Millisecond)

	manager.mu.RLock()
	running := manager.running
	manager.mu.RUnlock()

	if running {
		t.Error("Manager is still running after Stop()")
	}
}

// TestManagerStopWithoutStart tests stopping a manager that was never started
func TestManagerStopWithoutStart(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()

	// Stopping a non-running manager should not panic
	manager.Stop()

	manager.mu.RLock()
	running := manager.running
	manager.mu.RUnlock()

	if running {
		t.Error("Manager should not be running")
	}
}

// TestManagerTriggerSync tests manual sync triggering
func TestManagerTriggerSync(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()
	defer manager.Stop()

	// Start the manager
	go manager.Start(context.Background())
	time.Sleep(100 * time.Millisecond)

	// Trigger a sync
	err := manager.TriggerSync()
	if err != nil {
		t.Errorf("TriggerSync() error = %v", err)
	}

	// Wait for sync to complete
	time.Sleep(200 * time.Millisecond)
}

// TestManagerSetOnSyncCompleted tests setting sync completion callback
func TestManagerSetOnSyncCompleted(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()

	var callbackCalled atomic.Bool
	manager.SetOnSyncCompleted(func(newRecords int, durationMs int64) {
		callbackCalled.Store(true)
	})

	// Trigger a sync
	go manager.Start(context.Background())
	time.Sleep(100 * time.Millisecond)

	err := manager.TriggerSync()
	if err != nil {
		t.Fatalf("TriggerSync() error = %v", err)
	}

	// Wait for callback
	time.Sleep(500 * time.Millisecond)
	manager.Stop()

	if !callbackCalled.Load() {
		t.Error("OnSyncCompleted callback was not called")
	}
}

// TestManagerConcurrentSyncs tests that concurrent sync triggers are handled safely
func TestManagerConcurrentSyncs(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()
	defer manager.Stop()

	go manager.Start(context.Background())
	time.Sleep(100 * time.Millisecond)

	// Trigger multiple syncs concurrently
	const numSyncs = 5
	done := make(chan error, numSyncs)

	for i := 0; i < numSyncs; i++ {
		go func() {
			done <- manager.TriggerSync()
		}()
	}

	// Collect results
	for i := 0; i < numSyncs; i++ {
		err := <-done
		// Some may succeed, some may fail if already syncing
		// The important thing is no panic
		_ = err
	}
}

// TestManagerInterval tests that sync interval is respected
func TestManagerInterval(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping interval test in short mode")
	}

	// Create manager with very short interval for testing
	dbCfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, err := database.New(dbCfg, 0.0, 0.0)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	wsHub := ws.NewHub()
	go wsHub.Run()

	mockClient := &MockTautulliClient{}

	// Create full config with 500ms interval for testing
	cfg := &config.Config{
		Tautulli: config.TautulliConfig{
			Enabled: true,
			URL:     "http://localhost:8181",
			APIKey:  "test-api-key",
		},
		Sync: config.SyncConfig{
			Interval:      500 * time.Millisecond,
			Lookback:      24 * time.Hour,
			BatchSize:     1000,
			RetryAttempts: 5,
			RetryDelay:    2 * time.Second,
		},
	}

	manager := NewManager(db, nil, mockClient, cfg, wsHub)

	var syncCount atomic.Int32
	manager.SetOnSyncCompleted(func(newRecords int, durationMs int64) {
		syncCount.Add(1)
	})

	go manager.Start(context.Background())
	defer manager.Stop()

	// Wait for ~3 intervals
	time.Sleep(1600 * time.Millisecond)

	// Should have synced at least 2 times (initial + 1-2 interval syncs)
	if syncCount.Load() < 2 {
		t.Errorf("Expected at least 2 syncs, got %d", syncCount.Load())
	}
}

// TestManagerStopDuringSync tests stopping manager while sync is in progress
func TestManagerStopDuringSync(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()

	go manager.Start(context.Background())
	time.Sleep(100 * time.Millisecond)

	// Trigger sync
	err := manager.TriggerSync()
	if err != nil {
		t.Fatalf("TriggerSync() error = %v", err)
	}

	// Stop immediately (might be during sync)
	manager.Stop()

	// Should not panic
	time.Sleep(100 * time.Millisecond)
}

// TestManagerMultipleStarts tests starting manager multiple times
func TestManagerMultipleStarts(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()
	defer manager.Stop()

	// First start
	go manager.Start(context.Background())
	time.Sleep(100 * time.Millisecond)

	manager.mu.RLock()
	running1 := manager.running
	manager.mu.RUnlock()

	if !running1 {
		t.Error("Manager should be running after first Start()")
	}

	// Second start should be ignored
	go manager.Start(context.Background())
	time.Sleep(100 * time.Millisecond)

	manager.mu.RLock()
	running2 := manager.running
	manager.mu.RUnlock()

	if !running2 {
		t.Error("Manager should still be running after second Start()")
	}
}

// TestManagerNilCallbacks tests that nil callbacks don't cause panics
func TestManagerNilCallbacks(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()

	// Don't set any callbacks
	go manager.Start(context.Background())
	time.Sleep(100 * time.Millisecond)

	// Trigger sync - should not panic even without callbacks
	err := manager.TriggerSync()
	if err != nil {
		t.Fatalf("TriggerSync() error = %v", err)
	}

	time.Sleep(300 * time.Millisecond)
	manager.Stop()
}

// TestManagerErrorHandling tests error handling during sync
func TestManagerErrorHandling(t *testing.T) {
	// Create manager with a minimal client
	manager, db := setupTestManager(t)
	defer db.Close()

	go manager.Start(context.Background())
	defer manager.Stop()

	time.Sleep(100 * time.Millisecond)

	err := manager.TriggerSync()
	if err != nil {
		// Error is expected if sync encounters issues
		t.Logf("TriggerSync() returned error: %v", err)
	}

	// Wait for sync to complete
	time.Sleep(500 * time.Millisecond)

	// The test passes if we don't panic
}

// TestManagerStatePersistence tests that manager state is maintained across operations
func TestManagerStatePersistence(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()

	// Set callbacks
	var completedCalls atomic.Int32
	manager.SetOnSyncCompleted(func(newRecords int, durationMs int64) {
		completedCalls.Add(1)
	})

	go manager.Start(context.Background())
	time.Sleep(100 * time.Millisecond)

	// First sync
	err := manager.TriggerSync()
	if err != nil {
		t.Fatalf("First TriggerSync() error = %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	// Second sync
	err = manager.TriggerSync()
	if err != nil {
		t.Fatalf("Second TriggerSync() error = %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	manager.Stop()

	// Callbacks should have been called for each sync
	if completedCalls.Load() < 2 {
		t.Errorf("Expected at least 2 completed calls, got %d", completedCalls.Load())
	}
}
