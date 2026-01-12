// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// methodValidationTest defines a test case for HTTP method validation
type methodValidationTest struct {
	name    string
	path    string
	handler func(h *Handler) http.HandlerFunc
	methods []string // methods that should return 405
}

// TestMethodNotAllowed consolidates all HTTP method validation tests into a single table-driven test.
// This significantly reduces cyclomatic complexity while maintaining the same test coverage.
func TestMethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{}

	// Standard disallowed methods for GET-only endpoints
	standardMethods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	tests := []methodValidationTest{
		// Core analytics endpoints
		{
			name:    "AnalyticsTrends",
			path:    "/api/v1/analytics/trends",
			handler: func(h *Handler) http.HandlerFunc { return h.AnalyticsTrends },
			methods: standardMethods,
		},
		{
			name:    "AnalyticsGeographic",
			path:    "/api/v1/analytics/geographic",
			handler: func(h *Handler) http.HandlerFunc { return h.AnalyticsGeographic },
			methods: standardMethods,
		},
		{
			name:    "AnalyticsUsers",
			path:    "/api/v1/analytics/users",
			handler: func(h *Handler) http.HandlerFunc { return h.AnalyticsUsers },
			methods: standardMethods,
		},
		{
			name:    "AnalyticsAbandonment",
			path:    "/api/v1/analytics/abandonment",
			handler: func(h *Handler) http.HandlerFunc { return h.AnalyticsAbandonment },
			methods: standardMethods,
		},
		// Health endpoints
		{
			name:    "Health",
			path:    "/api/v1/health",
			handler: func(h *Handler) http.HandlerFunc { return h.Health },
			methods: standardMethods,
		},
		// Core data endpoints
		{
			name:    "Stats",
			path:    "/api/v1/stats",
			handler: func(h *Handler) http.HandlerFunc { return h.Stats },
			methods: standardMethods,
		},
		{
			name:    "Playbacks",
			path:    "/api/v1/playbacks",
			handler: func(h *Handler) http.HandlerFunc { return h.Playbacks },
			methods: standardMethods,
		},
		{
			name:    "Locations",
			path:    "/api/v1/locations",
			handler: func(h *Handler) http.HandlerFunc { return h.Locations },
			methods: standardMethods,
		},
		{
			name:    "Users",
			path:    "/api/v1/users",
			handler: func(h *Handler) http.HandlerFunc { return h.Users },
			methods: standardMethods,
		},
		{
			name:    "MediaTypes",
			path:    "/api/v1/media-types",
			handler: func(h *Handler) http.HandlerFunc { return h.MediaTypes },
			methods: standardMethods,
		},
		// Tautulli proxy endpoints (use 3 methods since some may allow PATCH)
		{
			name:    "TautulliActivity",
			path:    "/api/v1/tautulli/activity",
			handler: func(h *Handler) http.HandlerFunc { return h.TautulliActivity },
			methods: []string{http.MethodPost, http.MethodPut, http.MethodDelete},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, method := range tt.methods {
				t.Run(method, func(t *testing.T) {
					req := httptest.NewRequest(method, tt.path, nil)
					w := httptest.NewRecorder()

					tt.handler(handler)(w, req)

					if w.Code != http.StatusMethodNotAllowed {
						t.Errorf("%s %s: expected status 405, got %d", method, tt.path, w.Code)
					}
				})
			}
		})
	}
}

// TestServerInfo_MethodNotAllowed tests ServerInfo separately as it needs config
func TestServerInfo_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		config: &config.Config{},
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/server-info", nil)
			w := httptest.NewRecorder()

			handler.ServerInfo(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsUsers_LimitValidation tests limit parameter validation
func TestAnalyticsUsers_LimitValidation(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	tests := []struct {
		name           string
		limitParam     string
		expectedStatus int
	}{
		{"Limit too small", "0", http.StatusBadRequest},
		{"Limit too large", "101", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/analytics/users?limit=" + tt.limitParam
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsUsers(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response models.APIResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Status != "error" {
				t.Errorf("Expected status 'error', got '%s'", response.Status)
			}
		})
	}
}

// MockTautulliClient provides a mock implementation for testing Tautulli API handlers.
// Uses function fields to allow test-specific behavior injection.
type MockTautulliClient struct {
	GetActivityFunc         func(ctx context.Context, sessionKey string) (*tautulli.TautulliActivity, error)
	GetMetadataFunc         func(ctx context.Context, ratingKey string) (*tautulli.TautulliMetadata, error)
	GetUserFunc             func(ctx context.Context, userID int) (*tautulli.TautulliUser, error)
	GetUsersFunc            func(ctx context.Context) (*tautulli.TautulliUsers, error)
	GetLibraryUserStatsFunc func(ctx context.Context, sectionID int, grouping int) (*tautulli.TautulliLibraryUserStats, error)
	GetRecentlyAddedFunc    func(ctx context.Context, count int, start int, mediaType string, sectionID int) (*tautulli.TautulliRecentlyAdded, error)
	GetLibrariesFunc        func(ctx context.Context) (*tautulli.TautulliLibraries, error)
	GetLibraryFunc          func(ctx context.Context, sectionID int) (*tautulli.TautulliLibrary, error)
	GetServerInfoFunc       func(ctx context.Context) (*tautulli.TautulliServerInfo, error)
	GetSyncedItemsFunc      func(ctx context.Context, machineID string, userID int) (*tautulli.TautulliSyncedItems, error)
	TerminateSessionFunc    func(ctx context.Context, sessionID string, message string) (*tautulli.TautulliTerminateSession, error)

	// Priority 1: Analytics Dashboard Completion (8 endpoints)
	GetPlaysBySourceResolutionFunc func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysBySourceResolution, error)
	GetPlaysByStreamResolutionFunc func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByStreamResolution, error)
	GetPlaysByTop10PlatformsFunc   func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Platforms, error)
	GetPlaysByTop10UsersFunc       func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Users, error)
	GetPlaysPerMonthFunc           func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysPerMonth, error)
	GetUserPlayerStatsFunc         func(ctx context.Context, userID int) (*tautulli.TautulliUserPlayerStats, error)
	GetUserWatchTimeStatsFunc      func(ctx context.Context, userID int, queryDays string) (*tautulli.TautulliUserWatchTimeStats, error)
	GetItemUserStatsFunc           func(ctx context.Context, ratingKey string, grouping int) (*tautulli.TautulliItemUserStats, error)

	// Priority 2: Library-Specific Analytics (4 endpoints)
	GetLibrariesTableFunc        func(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliLibrariesTable, error)
	GetLibraryMediaInfoFunc      func(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int) (*tautulli.TautulliLibraryMediaInfo, error)
	GetLibraryWatchTimeStatsFunc func(ctx context.Context, sectionID int, grouping int, queryDays string) (*tautulli.TautulliLibraryWatchTimeStats, error)
	GetChildrenMetadataFunc      func(ctx context.Context, ratingKey string, mediaType string) (*tautulli.TautulliChildrenMetadata, error)

	// Priority 1: User Geography & Management (3 endpoints)
	GetUserIPsFunc    func(ctx context.Context, userID int) (*tautulli.TautulliUserIPs, error)
	GetUsersTableFunc func(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUsersTable, error)
	GetUserLoginsFunc func(ctx context.Context, userID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUserLogins, error)

	// Priority 2: Enhanced Metadata & Export (4 endpoints)
	GetStreamDataFunc   func(ctx context.Context, rowID int, sessionKey string) (*tautulli.TautulliStreamData, error)
	GetLibraryNamesFunc func(ctx context.Context) (*tautulli.TautulliLibraryNames, error)
	ExportMetadataFunc  func(ctx context.Context, sectionID int, exportType string, userID int, ratingKey string, fileFormat string) (*tautulli.TautulliExportMetadata, error)
	GetExportFieldsFunc func(ctx context.Context, mediaType string) (*tautulli.TautulliExportFields, error)

	// Priority 3: Advanced Analytics & Metadata (5 endpoints)
	GetStreamTypeByTop10UsersFunc     func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliStreamTypeByTop10Users, error)
	GetStreamTypeByTop10PlatformsFunc func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliStreamTypeByTop10Platforms, error)
	SearchFunc                        func(ctx context.Context, query string, limit int) (*tautulli.TautulliSearch, error)
	GetNewRatingKeysFunc              func(ctx context.Context, ratingKey string) (*tautulli.TautulliNewRatingKeys, error)
	GetOldRatingKeysFunc              func(ctx context.Context, ratingKey string) (*tautulli.TautulliOldRatingKeys, error)
	GetCollectionsTableFunc           func(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliCollectionsTable, error)
	GetPlaylistsTableFunc             func(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliPlaylistsTable, error)
	GetServerFriendlyNameFunc         func(ctx context.Context) (*tautulli.TautulliServerFriendlyName, error)
	GetServerIDFunc                   func(ctx context.Context) (*tautulli.TautulliServerID, error)
	GetServerIdentityFunc             func(ctx context.Context) (*tautulli.TautulliServerIdentity, error)

	// Data Export Management (3 endpoints)
	GetExportsTableFunc func(ctx context.Context, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliExportsTable, error)
	DownloadExportFunc  func(ctx context.Context, exportID int) (*tautulli.TautulliDownloadExport, error)
	DeleteExportFunc    func(ctx context.Context, exportID int) (*tautulli.TautulliDeleteExport, error)

	// Server Management (5 endpoints)
	GetTautulliInfoFunc func(ctx context.Context) (*tautulli.TautulliTautulliInfo, error)
	GetServerPrefFunc   func(ctx context.Context, pref string) (*tautulli.TautulliServerPref, error)
	GetServerListFunc   func(ctx context.Context) (*tautulli.TautulliServerList, error)
	GetServersInfoFunc  func(ctx context.Context) (*tautulli.TautulliServersInfo, error)
	GetPMSUpdateFunc    func(ctx context.Context) (*tautulli.TautulliPMSUpdate, error)

	// Additional mock functions for testing
	PingFunc                             func(ctx context.Context) error
	GetHistorySinceFunc                  func(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error)
	GetGeoIPLookupFunc                   func(ctx context.Context, ipAddress string) (*tautulli.TautulliGeoIP, error)
	GetHomeStatsFunc                     func(ctx context.Context, timeRange int, statsType string, statsCount int) (*tautulli.TautulliHomeStats, error)
	GetPlaysByDateFunc                   func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByDate, error)
	GetPlaysByDayOfWeekFunc              func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByDayOfWeek, error)
	GetPlaysByHourOfDayFunc              func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByHourOfDay, error)
	GetPlaysByStreamTypeFunc             func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByStreamType, error)
	GetConcurrentStreamsByStreamTypeFunc func(ctx context.Context, timeRange int, userID int) (*tautulli.TautulliConcurrentStreamsByStreamType, error)
	GetItemWatchTimeStatsFunc            func(ctx context.Context, ratingKey string, grouping int, queryDays string) (*tautulli.TautulliItemWatchTimeStats, error)
}

// Ping returns nil (healthy) or calls PingFunc if set
func (m *MockTautulliClient) Ping(ctx context.Context) error {
	if m.PingFunc != nil {
		return m.PingFunc(ctx)
	}
	return nil
}

// GetHistorySince returns test history data or calls GetHistorySinceFunc if set
func (m *MockTautulliClient) GetHistorySince(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
	if m.GetHistorySinceFunc != nil {
		return m.GetHistorySinceFunc(ctx, since, start, length)
	}
	return &tautulli.TautulliHistory{Response: tautulli.TautulliHistoryResponse{Result: "success"}}, nil
}

// GetGeoIPLookup returns geo lookup data or calls GetGeoIPLookupFunc if set
func (m *MockTautulliClient) GetGeoIPLookup(ctx context.Context, ipAddress string) (*tautulli.TautulliGeoIP, error) {
	if m.GetGeoIPLookupFunc != nil {
		return m.GetGeoIPLookupFunc(ctx, ipAddress)
	}
	return &tautulli.TautulliGeoIP{Response: tautulli.TautulliGeoIPResponse{Result: "success"}}, nil
}

// GetHomeStats returns home stats or calls GetHomeStatsFunc if set
func (m *MockTautulliClient) GetHomeStats(ctx context.Context, timeRange int, statsType string, statsCount int) (*tautulli.TautulliHomeStats, error) {
	if m.GetHomeStatsFunc != nil {
		return m.GetHomeStatsFunc(ctx, timeRange, statsType, statsCount)
	}
	return &tautulli.TautulliHomeStats{Response: tautulli.TautulliHomeStatsResponse{Result: "success"}}, nil
}

// GetPlaysByDate returns plays by date data or calls GetPlaysByDateFunc if set
func (m *MockTautulliClient) GetPlaysByDate(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByDate, error) {
	if m.GetPlaysByDateFunc != nil {
		return m.GetPlaysByDateFunc(ctx, timeRange, yAxis, userID, grouping)
	}
	return &tautulli.TautulliPlaysByDate{Response: tautulli.TautulliPlaysByDateResponse{Result: "success"}}, nil
}

// GetPlaysByDayOfWeek returns plays by day of week data
func (m *MockTautulliClient) GetPlaysByDayOfWeek(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByDayOfWeek, error) {
	if m.GetPlaysByDayOfWeekFunc != nil {
		return m.GetPlaysByDayOfWeekFunc(ctx, timeRange, yAxis, userID, grouping)
	}
	return &tautulli.TautulliPlaysByDayOfWeek{Response: tautulli.TautulliPlaysByDayOfWeekResponse{Result: "success"}}, nil
}

// GetPlaysByHourOfDay returns plays by hour of day data
func (m *MockTautulliClient) GetPlaysByHourOfDay(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByHourOfDay, error) {
	if m.GetPlaysByHourOfDayFunc != nil {
		return m.GetPlaysByHourOfDayFunc(ctx, timeRange, yAxis, userID, grouping)
	}
	return &tautulli.TautulliPlaysByHourOfDay{Response: tautulli.TautulliPlaysByHourOfDayResponse{Result: "success"}}, nil
}

// GetPlaysByStreamType returns plays by stream type data
func (m *MockTautulliClient) GetPlaysByStreamType(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByStreamType, error) {
	if m.GetPlaysByStreamTypeFunc != nil {
		return m.GetPlaysByStreamTypeFunc(ctx, timeRange, yAxis, userID, grouping)
	}
	return &tautulli.TautulliPlaysByStreamType{Response: tautulli.TautulliPlaysByStreamTypeResponse{Result: "success"}}, nil
}

// GetConcurrentStreamsByStreamType returns concurrent streams data
func (m *MockTautulliClient) GetConcurrentStreamsByStreamType(ctx context.Context, timeRange int, userID int) (*tautulli.TautulliConcurrentStreamsByStreamType, error) {
	if m.GetConcurrentStreamsByStreamTypeFunc != nil {
		return m.GetConcurrentStreamsByStreamTypeFunc(ctx, timeRange, userID)
	}
	return &tautulli.TautulliConcurrentStreamsByStreamType{Response: tautulli.TautulliConcurrentStreamsByStreamTypeResponse{Result: "success"}}, nil
}

// GetItemWatchTimeStats returns item watch time stats
func (m *MockTautulliClient) GetItemWatchTimeStats(ctx context.Context, ratingKey string, grouping int, queryDays string) (*tautulli.TautulliItemWatchTimeStats, error) {
	if m.GetItemWatchTimeStatsFunc != nil {
		return m.GetItemWatchTimeStatsFunc(ctx, ratingKey, grouping, queryDays)
	}
	return &tautulli.TautulliItemWatchTimeStats{Response: tautulli.TautulliItemWatchTimeStatsResponse{Result: "success"}}, nil
}

// GetActivity returns activity data or calls GetActivityFunc if set
func (m *MockTautulliClient) GetActivity(ctx context.Context, sessionKey string) (*tautulli.TautulliActivity, error) {
	if m.GetActivityFunc != nil {
		return m.GetActivityFunc(ctx, sessionKey)
	}
	return nil, nil
}

// GetMetadata returns metadata or calls GetMetadataFunc if set
func (m *MockTautulliClient) GetMetadata(ctx context.Context, ratingKey string) (*tautulli.TautulliMetadata, error) {
	if m.GetMetadataFunc != nil {
		return m.GetMetadataFunc(ctx, ratingKey)
	}
	return nil, nil
}

// GetUser returns user data or calls GetUserFunc if set
func (m *MockTautulliClient) GetUser(ctx context.Context, userID int) (*tautulli.TautulliUser, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(ctx, userID)
	}
	return nil, nil
}

// GetUsers returns all users or calls GetUsersFunc if set
func (m *MockTautulliClient) GetUsers(ctx context.Context) (*tautulli.TautulliUsers, error) {
	if m.GetUsersFunc != nil {
		return m.GetUsersFunc(ctx)
	}
	return nil, nil
}

// GetLibraryUserStats returns library user stats
func (m *MockTautulliClient) GetLibraryUserStats(ctx context.Context, sectionID int, grouping int) (*tautulli.TautulliLibraryUserStats, error) {
	if m.GetLibraryUserStatsFunc != nil {
		return m.GetLibraryUserStatsFunc(ctx, sectionID, grouping)
	}
	return nil, nil
}

// GetRecentlyAdded returns recently added data
func (m *MockTautulliClient) GetRecentlyAdded(ctx context.Context, count int, start int, mediaType string, sectionID int) (*tautulli.TautulliRecentlyAdded, error) {
	if m.GetRecentlyAddedFunc != nil {
		return m.GetRecentlyAddedFunc(ctx, count, start, mediaType, sectionID)
	}
	return nil, nil
}

// GetLibraries returns libraries list
func (m *MockTautulliClient) GetLibraries(ctx context.Context) (*tautulli.TautulliLibraries, error) {
	if m.GetLibrariesFunc != nil {
		return m.GetLibrariesFunc(ctx)
	}
	return nil, nil
}

// GetLibrary returns library data
func (m *MockTautulliClient) GetLibrary(ctx context.Context, sectionID int) (*tautulli.TautulliLibrary, error) {
	if m.GetLibraryFunc != nil {
		return m.GetLibraryFunc(ctx, sectionID)
	}
	return nil, nil
}

// GetServerInfo returns server info
func (m *MockTautulliClient) GetServerInfo(ctx context.Context) (*tautulli.TautulliServerInfo, error) {
	if m.GetServerInfoFunc != nil {
		return m.GetServerInfoFunc(ctx)
	}
	return nil, nil
}

// GetSyncedItems returns synced items
func (m *MockTautulliClient) GetSyncedItems(ctx context.Context, machineID string, userID int) (*tautulli.TautulliSyncedItems, error) {
	if m.GetSyncedItemsFunc != nil {
		return m.GetSyncedItemsFunc(ctx, machineID, userID)
	}
	return nil, nil
}

// TerminateSession terminates a session
func (m *MockTautulliClient) TerminateSession(ctx context.Context, sessionID string, message string) (*tautulli.TautulliTerminateSession, error) {
	if m.TerminateSessionFunc != nil {
		return m.TerminateSessionFunc(ctx, sessionID, message)
	}
	return nil, nil
}

// Analytics Dashboard methods - return nil or call function if set
func (m *MockTautulliClient) GetPlaysBySourceResolution(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysBySourceResolution, error) {
	if m.GetPlaysBySourceResolutionFunc != nil {
		return m.GetPlaysBySourceResolutionFunc(ctx, timeRange, yAxis, userID, grouping)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetPlaysByStreamResolution(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByStreamResolution, error) {
	if m.GetPlaysByStreamResolutionFunc != nil {
		return m.GetPlaysByStreamResolutionFunc(ctx, timeRange, yAxis, userID, grouping)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetPlaysByTop10Platforms(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Platforms, error) {
	if m.GetPlaysByTop10PlatformsFunc != nil {
		return m.GetPlaysByTop10PlatformsFunc(ctx, timeRange, yAxis, userID, grouping)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetPlaysByTop10Users(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Users, error) {
	if m.GetPlaysByTop10UsersFunc != nil {
		return m.GetPlaysByTop10UsersFunc(ctx, timeRange, yAxis, userID, grouping)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetPlaysPerMonth(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysPerMonth, error) {
	if m.GetPlaysPerMonthFunc != nil {
		return m.GetPlaysPerMonthFunc(ctx, timeRange, yAxis, userID, grouping)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetUserPlayerStats(ctx context.Context, userID int) (*tautulli.TautulliUserPlayerStats, error) {
	if m.GetUserPlayerStatsFunc != nil {
		return m.GetUserPlayerStatsFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetUserWatchTimeStats(ctx context.Context, userID int, queryDays string) (*tautulli.TautulliUserWatchTimeStats, error) {
	if m.GetUserWatchTimeStatsFunc != nil {
		return m.GetUserWatchTimeStatsFunc(ctx, userID, queryDays)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetItemUserStats(ctx context.Context, ratingKey string, grouping int) (*tautulli.TautulliItemUserStats, error) {
	if m.GetItemUserStatsFunc != nil {
		return m.GetItemUserStatsFunc(ctx, ratingKey, grouping)
	}
	return nil, nil
}

// Library-Specific Analytics methods
func (m *MockTautulliClient) GetLibrariesTable(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliLibrariesTable, error) {
	if m.GetLibrariesTableFunc != nil {
		return m.GetLibrariesTableFunc(ctx, grouping, orderColumn, orderDir, start, length, search)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetLibraryMediaInfo(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int) (*tautulli.TautulliLibraryMediaInfo, error) {
	if m.GetLibraryMediaInfoFunc != nil {
		return m.GetLibraryMediaInfoFunc(ctx, sectionID, orderColumn, orderDir, start, length)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetLibraryWatchTimeStats(ctx context.Context, sectionID int, grouping int, queryDays string) (*tautulli.TautulliLibraryWatchTimeStats, error) {
	if m.GetLibraryWatchTimeStatsFunc != nil {
		return m.GetLibraryWatchTimeStatsFunc(ctx, sectionID, grouping, queryDays)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetChildrenMetadata(ctx context.Context, ratingKey string, mediaType string) (*tautulli.TautulliChildrenMetadata, error) {
	if m.GetChildrenMetadataFunc != nil {
		return m.GetChildrenMetadataFunc(ctx, ratingKey, mediaType)
	}
	return nil, nil
}

// User Geography & Management methods
func (m *MockTautulliClient) GetUserIPs(ctx context.Context, userID int) (*tautulli.TautulliUserIPs, error) {
	if m.GetUserIPsFunc != nil {
		return m.GetUserIPsFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetUsersTable(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUsersTable, error) {
	if m.GetUsersTableFunc != nil {
		return m.GetUsersTableFunc(ctx, grouping, orderColumn, orderDir, start, length, search)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetUserLogins(ctx context.Context, userID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUserLogins, error) {
	if m.GetUserLoginsFunc != nil {
		return m.GetUserLoginsFunc(ctx, userID, orderColumn, orderDir, start, length, search)
	}
	return nil, nil
}

// Enhanced Metadata & Export methods
func (m *MockTautulliClient) GetStreamData(ctx context.Context, rowID int, sessionKey string) (*tautulli.TautulliStreamData, error) {
	if m.GetStreamDataFunc != nil {
		return m.GetStreamDataFunc(ctx, rowID, sessionKey)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetLibraryNames(ctx context.Context) (*tautulli.TautulliLibraryNames, error) {
	if m.GetLibraryNamesFunc != nil {
		return m.GetLibraryNamesFunc(ctx)
	}
	return nil, nil
}

func (m *MockTautulliClient) ExportMetadata(ctx context.Context, sectionID int, exportType string, userID int, ratingKey string, fileFormat string) (*tautulli.TautulliExportMetadata, error) {
	if m.ExportMetadataFunc != nil {
		return m.ExportMetadataFunc(ctx, sectionID, exportType, userID, ratingKey, fileFormat)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetExportFields(ctx context.Context, mediaType string) (*tautulli.TautulliExportFields, error) {
	if m.GetExportFieldsFunc != nil {
		return m.GetExportFieldsFunc(ctx, mediaType)
	}
	return nil, nil
}

// Advanced Analytics & Metadata methods
func (m *MockTautulliClient) GetStreamTypeByTop10Users(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliStreamTypeByTop10Users, error) {
	if m.GetStreamTypeByTop10UsersFunc != nil {
		return m.GetStreamTypeByTop10UsersFunc(ctx, timeRange, yAxis, userID, grouping)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetStreamTypeByTop10Platforms(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliStreamTypeByTop10Platforms, error) {
	if m.GetStreamTypeByTop10PlatformsFunc != nil {
		return m.GetStreamTypeByTop10PlatformsFunc(ctx, timeRange, yAxis, userID, grouping)
	}
	return nil, nil
}

func (m *MockTautulliClient) Search(ctx context.Context, query string, limit int) (*tautulli.TautulliSearch, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query, limit)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetNewRatingKeys(ctx context.Context, ratingKey string) (*tautulli.TautulliNewRatingKeys, error) {
	if m.GetNewRatingKeysFunc != nil {
		return m.GetNewRatingKeysFunc(ctx, ratingKey)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetOldRatingKeys(ctx context.Context, ratingKey string) (*tautulli.TautulliOldRatingKeys, error) {
	if m.GetOldRatingKeysFunc != nil {
		return m.GetOldRatingKeysFunc(ctx, ratingKey)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetCollectionsTable(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliCollectionsTable, error) {
	if m.GetCollectionsTableFunc != nil {
		return m.GetCollectionsTableFunc(ctx, sectionID, orderColumn, orderDir, start, length, search)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetPlaylistsTable(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliPlaylistsTable, error) {
	if m.GetPlaylistsTableFunc != nil {
		return m.GetPlaylistsTableFunc(ctx, sectionID, orderColumn, orderDir, start, length, search)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetServerFriendlyName(ctx context.Context) (*tautulli.TautulliServerFriendlyName, error) {
	if m.GetServerFriendlyNameFunc != nil {
		return m.GetServerFriendlyNameFunc(ctx)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetServerID(ctx context.Context) (*tautulli.TautulliServerID, error) {
	if m.GetServerIDFunc != nil {
		return m.GetServerIDFunc(ctx)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetServerIdentity(ctx context.Context) (*tautulli.TautulliServerIdentity, error) {
	if m.GetServerIdentityFunc != nil {
		return m.GetServerIdentityFunc(ctx)
	}
	return nil, nil
}

// Data Export Management methods
func (m *MockTautulliClient) GetExportsTable(ctx context.Context, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliExportsTable, error) {
	if m.GetExportsTableFunc != nil {
		return m.GetExportsTableFunc(ctx, orderColumn, orderDir, start, length, search)
	}
	return nil, nil
}

func (m *MockTautulliClient) DownloadExport(ctx context.Context, exportID int) (*tautulli.TautulliDownloadExport, error) {
	if m.DownloadExportFunc != nil {
		return m.DownloadExportFunc(ctx, exportID)
	}
	return nil, nil
}

func (m *MockTautulliClient) DeleteExport(ctx context.Context, exportID int) (*tautulli.TautulliDeleteExport, error) {
	if m.DeleteExportFunc != nil {
		return m.DeleteExportFunc(ctx, exportID)
	}
	return nil, nil
}

// Server Management methods
func (m *MockTautulliClient) GetTautulliInfo(ctx context.Context) (*tautulli.TautulliTautulliInfo, error) {
	if m.GetTautulliInfoFunc != nil {
		return m.GetTautulliInfoFunc(ctx)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetServerPref(ctx context.Context, pref string) (*tautulli.TautulliServerPref, error) {
	if m.GetServerPrefFunc != nil {
		return m.GetServerPrefFunc(ctx, pref)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetServerList(ctx context.Context) (*tautulli.TautulliServerList, error) {
	if m.GetServerListFunc != nil {
		return m.GetServerListFunc(ctx)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetServersInfo(ctx context.Context) (*tautulli.TautulliServersInfo, error) {
	if m.GetServersInfoFunc != nil {
		return m.GetServersInfoFunc(ctx)
	}
	return nil, nil
}

func (m *MockTautulliClient) GetPMSUpdate(ctx context.Context) (*tautulli.TautulliPMSUpdate, error) {
	if m.GetPMSUpdateFunc != nil {
		return m.GetPMSUpdateFunc(ctx)
	}
	return nil, nil
}
