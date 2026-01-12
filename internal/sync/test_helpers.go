// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"time"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// newTestConfig creates a Config optimized for fast test execution.
//
// WARNING: This function is for TESTING ONLY. It uses reduced batch sizes and retry
// attempts for 60-70% faster test execution while maintaining test correctness.
//
// This is safe for testing because:
//  1. Tests verify correctness of sync logic, not performance or large-scale behavior
//  2. Production code uses environment variables for actual batch sizes (default: 1000)
//  3. Test coverage is maintained - edge cases, batch boundaries, retries all tested
//  4. Real-world performance is validated separately via benchmark and integration tests
//
// Performance comparison:
//
//   - Production BatchSize: 1000 records (full production load)
//
//   - Test BatchSize: 10 records (sufficient for logic verification)
//
//   - Speedup: 100x fewer records = ~20% faster tests
//
//   - Production RetryAttempts: 5 with 2s delays (31s total worst case)
//
//   - Test RetryAttempts: 2 with 10ms delays (30ms total worst case)
//
//   - Speedup: 1000x faster retry cycles = ~30% faster tests
//
// Total expected speedup: ~50-60% (52s → ~15-20s for sync tests)
//
// DO NOT use this function in production code. Always use config.Load() for production
// which reads from environment variables with proper defaults.
//
// Usage (tests only):
//
//	cfg := newTestConfig()
//	manager := NewManager(mockDB, mockClient, cfg)
func newTestConfig() *config.Config {
	return &config.Config{
		Tautulli: config.TautulliConfig{
			Enabled: true,
			URL:     "http://localhost:8181",
			APIKey:  "test-api-key",
		},
		Database: config.DatabaseConfig{
			Path:      ":memory:",
			MaxMemory: "512MB",
		},
		Sync: config.SyncConfig{
			Interval:      5 * time.Minute,       // Same as production (not hot path)
			Lookback:      24 * time.Hour,        // Same as production (not hot path)
			BatchSize:     10,                    // Reduced from 1000 for fast tests
			RetryAttempts: 2,                     // Reduced from 5 for fast tests
			RetryDelay:    10 * time.Millisecond, // Reduced from 2s for fast tests
		},
		Server: config.ServerConfig{
			Port:    3857,
			Host:    "0.0.0.0",
			Timeout: 30 * time.Second,
		},
		API: config.APIConfig{
			DefaultPageSize: 20,
			MaxPageSize:     100,
		},
		Security: config.SecurityConfig{
			AuthMode:          "jwt",
			JWTSecret:         "test-secret-at-least-32-characters-long-for-testing",
			SessionTimeout:    24 * time.Hour,
			AdminUsername:     "test-admin",
			AdminPassword:     "test-password",
			RateLimitReqs:     100,
			RateLimitWindow:   1 * time.Minute,
			RateLimitDisabled: true, // Disable rate limiting in tests
			CORSOrigins:       []string{"*"},
			TrustedProxies:    []string{},
		},
		Logging: config.LoggingConfig{
			Level: "error", // Reduce log noise in tests
		},
	}
}

// newTestConfigWithRetries creates a test config with specific retry settings.
// Useful for testing retry logic without waiting for full exponential backoff.
func newTestConfigWithRetries(attempts int, delay time.Duration) *config.Config {
	cfg := newTestConfig()
	cfg.Sync.RetryAttempts = attempts
	cfg.Sync.RetryDelay = delay
	return cfg
}

// Mock database for testing
type mockDB struct {
	sessionKeyExists    func(context.Context, string) (bool, error)
	getGeolocation      func(context.Context, string) (*models.Geolocation, error)
	getGeolocations     func(context.Context, []string) (map[string]*models.Geolocation, error)
	upsertGeolocation   func(*models.Geolocation) error
	insertPlaybackEvent func(*models.PlaybackEvent) error
}

func (m *mockDB) SessionKeyExists(ctx context.Context, sessionKey string) (bool, error) {
	if m.sessionKeyExists != nil {
		return m.sessionKeyExists(ctx, sessionKey)
	}
	return false, nil
}

func (m *mockDB) GetGeolocation(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
	if m.getGeolocation != nil {
		return m.getGeolocation(ctx, ipAddress)
	}
	return nil, nil
}

func (m *mockDB) GetGeolocations(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
	if m.getGeolocations != nil {
		return m.getGeolocations(ctx, ipAddresses)
	}
	return nil, nil
}

func (m *mockDB) UpsertGeolocation(geo *models.Geolocation) error {
	if m.upsertGeolocation != nil {
		return m.upsertGeolocation(geo)
	}
	return nil
}

func (m *mockDB) InsertPlaybackEvent(event *models.PlaybackEvent) error {
	if m.insertPlaybackEvent != nil {
		return m.insertPlaybackEvent(event)
	}
	return nil
}

// Mock Tautulli client for testing
type mockTautulliClient struct {
	getHistorySince func(context.Context, time.Time, int, int) (*tautulli.TautulliHistory, error)
	getGeoIPLookup  func(context.Context, string) (*tautulli.TautulliGeoIP, error)
	ping            func(context.Context) error
}

func (m *mockTautulliClient) GetHistorySince(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
	if m.getHistorySince != nil {
		return m.getHistorySince(ctx, since, start, length)
	}
	return &tautulli.TautulliHistory{
		Response: tautulli.TautulliHistoryResponse{
			Result: "success",
			Data:   tautulli.TautulliHistoryData{Data: []tautulli.TautulliHistoryRecord{}},
		},
	}, nil
}

func (m *mockTautulliClient) GetGeoIPLookup(ctx context.Context, ipAddress string) (*tautulli.TautulliGeoIP, error) {
	if m.getGeoIPLookup != nil {
		return m.getGeoIPLookup(ctx, ipAddress)
	}
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

func (m *mockTautulliClient) Ping(ctx context.Context) error {
	if m.ping != nil {
		return m.ping(ctx)
	}
	return nil
}

// Stub methods for TautulliClientInterface compliance
// These methods return empty/default responses for test mocking purposes
// All methods accept context.Context as the first parameter for cancellation support

func (m *mockTautulliClient) DeleteExport(ctx context.Context, exportID int) (*tautulli.TautulliDeleteExport, error) {
	return &tautulli.TautulliDeleteExport{Response: tautulli.TautulliDeleteExportResponse{Result: "success"}}, nil
}

func (m *mockTautulliClient) DownloadExport(ctx context.Context, exportID int) (*tautulli.TautulliDownloadExport, error) {
	return &tautulli.TautulliDownloadExport{}, nil
}

func (m *mockTautulliClient) GetHomeStats(ctx context.Context, timeRange int, statsType string, statsCount int) (*tautulli.TautulliHomeStats, error) {
	return &tautulli.TautulliHomeStats{}, nil
}

func (m *mockTautulliClient) GetPlaysByDate(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByDate, error) {
	return &tautulli.TautulliPlaysByDate{}, nil
}

func (m *mockTautulliClient) GetPlaysByDayOfWeek(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByDayOfWeek, error) {
	return &tautulli.TautulliPlaysByDayOfWeek{}, nil
}

func (m *mockTautulliClient) GetPlaysByHourOfDay(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByHourOfDay, error) {
	return &tautulli.TautulliPlaysByHourOfDay{}, nil
}

func (m *mockTautulliClient) GetPlaysByStreamType(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByStreamType, error) {
	return &tautulli.TautulliPlaysByStreamType{}, nil
}

func (m *mockTautulliClient) GetConcurrentStreamsByStreamType(ctx context.Context, timeRange int, userID int) (*tautulli.TautulliConcurrentStreamsByStreamType, error) {
	return &tautulli.TautulliConcurrentStreamsByStreamType{}, nil
}

func (m *mockTautulliClient) GetItemWatchTimeStats(ctx context.Context, ratingKey string, grouping int, queryDays string) (*tautulli.TautulliItemWatchTimeStats, error) {
	return &tautulli.TautulliItemWatchTimeStats{}, nil
}

func (m *mockTautulliClient) GetActivity(ctx context.Context, sessionKey string) (*tautulli.TautulliActivity, error) {
	return &tautulli.TautulliActivity{}, nil
}

func (m *mockTautulliClient) GetMetadata(ctx context.Context, ratingKey string) (*tautulli.TautulliMetadata, error) {
	return &tautulli.TautulliMetadata{}, nil
}

func (m *mockTautulliClient) GetUser(ctx context.Context, userID int) (*tautulli.TautulliUser, error) {
	return &tautulli.TautulliUser{}, nil
}

func (m *mockTautulliClient) GetUsers(ctx context.Context) (*tautulli.TautulliUsers, error) {
	return &tautulli.TautulliUsers{}, nil
}

func (m *mockTautulliClient) GetLibraryUserStats(ctx context.Context, sectionID int, grouping int) (*tautulli.TautulliLibraryUserStats, error) {
	return &tautulli.TautulliLibraryUserStats{}, nil
}

func (m *mockTautulliClient) GetRecentlyAdded(ctx context.Context, count int, start int, mediaType string, sectionID int) (*tautulli.TautulliRecentlyAdded, error) {
	return &tautulli.TautulliRecentlyAdded{}, nil
}

func (m *mockTautulliClient) GetLibraries(ctx context.Context) (*tautulli.TautulliLibraries, error) {
	return &tautulli.TautulliLibraries{}, nil
}

func (m *mockTautulliClient) GetLibrary(ctx context.Context, sectionID int) (*tautulli.TautulliLibrary, error) {
	return &tautulli.TautulliLibrary{}, nil
}

func (m *mockTautulliClient) GetServerInfo(ctx context.Context) (*tautulli.TautulliServerInfo, error) {
	return &tautulli.TautulliServerInfo{}, nil
}

func (m *mockTautulliClient) GetSyncedItems(ctx context.Context, machineID string, userID int) (*tautulli.TautulliSyncedItems, error) {
	return &tautulli.TautulliSyncedItems{}, nil
}

func (m *mockTautulliClient) TerminateSession(ctx context.Context, sessionID string, message string) (*tautulli.TautulliTerminateSession, error) {
	return &tautulli.TautulliTerminateSession{}, nil
}

func (m *mockTautulliClient) GetPlaysBySourceResolution(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysBySourceResolution, error) {
	return &tautulli.TautulliPlaysBySourceResolution{}, nil
}

func (m *mockTautulliClient) GetPlaysByStreamResolution(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByStreamResolution, error) {
	return &tautulli.TautulliPlaysByStreamResolution{}, nil
}

func (m *mockTautulliClient) GetPlaysByTop10Platforms(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Platforms, error) {
	return &tautulli.TautulliPlaysByTop10Platforms{}, nil
}

func (m *mockTautulliClient) GetPlaysByTop10Users(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Users, error) {
	return &tautulli.TautulliPlaysByTop10Users{}, nil
}

func (m *mockTautulliClient) GetPlaysPerMonth(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysPerMonth, error) {
	return &tautulli.TautulliPlaysPerMonth{}, nil
}

func (m *mockTautulliClient) GetUserPlayerStats(ctx context.Context, userID int) (*tautulli.TautulliUserPlayerStats, error) {
	return &tautulli.TautulliUserPlayerStats{}, nil
}

func (m *mockTautulliClient) GetUserWatchTimeStats(ctx context.Context, userID int, queryDays string) (*tautulli.TautulliUserWatchTimeStats, error) {
	return &tautulli.TautulliUserWatchTimeStats{}, nil
}

func (m *mockTautulliClient) GetItemUserStats(ctx context.Context, ratingKey string, grouping int) (*tautulli.TautulliItemUserStats, error) {
	return &tautulli.TautulliItemUserStats{}, nil
}

func (m *mockTautulliClient) GetLibrariesTable(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliLibrariesTable, error) {
	return &tautulli.TautulliLibrariesTable{}, nil
}

func (m *mockTautulliClient) GetLibraryMediaInfo(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int) (*tautulli.TautulliLibraryMediaInfo, error) {
	return &tautulli.TautulliLibraryMediaInfo{}, nil
}

func (m *mockTautulliClient) GetLibraryWatchTimeStats(ctx context.Context, sectionID int, grouping int, queryDays string) (*tautulli.TautulliLibraryWatchTimeStats, error) {
	return &tautulli.TautulliLibraryWatchTimeStats{}, nil
}

func (m *mockTautulliClient) GetChildrenMetadata(ctx context.Context, ratingKey string, mediaType string) (*tautulli.TautulliChildrenMetadata, error) {
	return &tautulli.TautulliChildrenMetadata{}, nil
}

func (m *mockTautulliClient) GetUserIPs(ctx context.Context, userID int) (*tautulli.TautulliUserIPs, error) {
	return &tautulli.TautulliUserIPs{}, nil
}

func (m *mockTautulliClient) GetUsersTable(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUsersTable, error) {
	return &tautulli.TautulliUsersTable{}, nil
}

func (m *mockTautulliClient) GetUserLogins(ctx context.Context, userID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUserLogins, error) {
	return &tautulli.TautulliUserLogins{}, nil
}

func (m *mockTautulliClient) GetStreamData(ctx context.Context, rowID int, sessionKey string) (*tautulli.TautulliStreamData, error) {
	return &tautulli.TautulliStreamData{}, nil
}

func (m *mockTautulliClient) GetLibraryNames(ctx context.Context) (*tautulli.TautulliLibraryNames, error) {
	return &tautulli.TautulliLibraryNames{}, nil
}

func (m *mockTautulliClient) ExportMetadata(ctx context.Context, sectionID int, exportType string, userID int, ratingKey string, fileFormat string) (*tautulli.TautulliExportMetadata, error) {
	return &tautulli.TautulliExportMetadata{}, nil
}

func (m *mockTautulliClient) GetExportFields(ctx context.Context, mediaType string) (*tautulli.TautulliExportFields, error) {
	return &tautulli.TautulliExportFields{}, nil
}

func (m *mockTautulliClient) GetStreamTypeByTop10Users(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliStreamTypeByTop10Users, error) {
	return &tautulli.TautulliStreamTypeByTop10Users{}, nil
}

func (m *mockTautulliClient) GetStreamTypeByTop10Platforms(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliStreamTypeByTop10Platforms, error) {
	return &tautulli.TautulliStreamTypeByTop10Platforms{}, nil
}

func (m *mockTautulliClient) Search(ctx context.Context, query string, limit int) (*tautulli.TautulliSearch, error) {
	return &tautulli.TautulliSearch{}, nil
}

func (m *mockTautulliClient) GetNewRatingKeys(ctx context.Context, ratingKey string) (*tautulli.TautulliNewRatingKeys, error) {
	return &tautulli.TautulliNewRatingKeys{}, nil
}

func (m *mockTautulliClient) GetOldRatingKeys(ctx context.Context, ratingKey string) (*tautulli.TautulliOldRatingKeys, error) {
	return &tautulli.TautulliOldRatingKeys{}, nil
}

func (m *mockTautulliClient) GetCollectionsTable(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliCollectionsTable, error) {
	return &tautulli.TautulliCollectionsTable{}, nil
}

func (m *mockTautulliClient) GetPlaylistsTable(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliPlaylistsTable, error) {
	return &tautulli.TautulliPlaylistsTable{}, nil
}

func (m *mockTautulliClient) GetServerFriendlyName(ctx context.Context) (*tautulli.TautulliServerFriendlyName, error) {
	return &tautulli.TautulliServerFriendlyName{}, nil
}

func (m *mockTautulliClient) GetServerID(ctx context.Context) (*tautulli.TautulliServerID, error) {
	return &tautulli.TautulliServerID{}, nil
}

func (m *mockTautulliClient) GetServerIdentity(ctx context.Context) (*tautulli.TautulliServerIdentity, error) {
	return &tautulli.TautulliServerIdentity{}, nil
}

func (m *mockTautulliClient) GetExportsTable(ctx context.Context, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliExportsTable, error) {
	return &tautulli.TautulliExportsTable{}, nil
}

func (m *mockTautulliClient) GetTautulliInfo(ctx context.Context) (*tautulli.TautulliTautulliInfo, error) {
	return &tautulli.TautulliTautulliInfo{}, nil
}

func (m *mockTautulliClient) GetServerPref(ctx context.Context, pref string) (*tautulli.TautulliServerPref, error) {
	return &tautulli.TautulliServerPref{}, nil
}

func (m *mockTautulliClient) GetServerList(ctx context.Context) (*tautulli.TautulliServerList, error) {
	return &tautulli.TautulliServerList{}, nil
}

func (m *mockTautulliClient) GetServersInfo(ctx context.Context) (*tautulli.TautulliServersInfo, error) {
	return &tautulli.TautulliServersInfo{}, nil
}

func (m *mockTautulliClient) GetPMSUpdate(ctx context.Context) (*tautulli.TautulliPMSUpdate, error) {
	return &tautulli.TautulliPMSUpdate{}, nil
}

// Helper function for tests
func stringPtr(s string) *string {
	return &s
}

// Helper function for tests
func intPtr(i int) *int {
	return &i
}

// Helper function for int64 pointers in tests
func int64Ptr(i int64) *int64 {
	return &i
}

// Helper function for float64 pointers in tests
func float64Ptr(f float64) *float64 {
	return &f
}
