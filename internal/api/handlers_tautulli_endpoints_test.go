// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// setupTestHandlerWithMockClient creates a test handler with a mocked Tautulli client
func setupTestHandlerWithMockClient(t *testing.T, client *MockTautulliClient) *Handler {
	t.Helper()
	c := cache.New(5 * time.Minute)
	return &Handler{
		db:     nil, // nil db - tests should only test handler logic
		cache:  c,
		client: client,
	}
}

// tautulliEndpointTest defines a compact test case for Tautulli endpoints
type tautulliEndpointTest struct {
	name        string
	handlerFunc func(*Handler, http.ResponseWriter, *http.Request)
	httpMethod  string // Optional, defaults to GET
	queryParams string
	setupMock   func(*MockTautulliClient)
	// validateResp is optional - if nil, uses default success validation
	validateResp func(*testing.T, *models.APIResponse)
}

// defaultSuccessValidation is the standard validation for successful responses
func defaultSuccessValidation(t *testing.T, resp *models.APIResponse) {
	t.Helper()
	if resp.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", resp.Status)
	}
}

// runTautulliEndpointTests runs a slice of tautulliEndpointTest using a common pattern
func runTautulliEndpointTests(t *testing.T, tests []tautulliEndpointTest, expectedStatus int) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockTautulliClient{}
			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			handler := setupTestHandlerWithMockClient(t, mockClient)

			url := "/api/v1/test"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}
			method := tt.httpMethod
			if method == "" {
				method = http.MethodGet
			}
			req := httptest.NewRequest(method, url, nil)
			w := httptest.NewRecorder()

			tt.handlerFunc(handler, w, req)

			if w.Code != expectedStatus {
				t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}

			if expectedStatus == http.StatusOK {
				var response models.APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				validate := tt.validateResp
				if validate == nil {
					validate = defaultSuccessValidation
				}
				validate(t, &response)
			}
		})
	}
}

// Test all 54 Tautulli proxy handlers using table-driven tests
func TestTautulliHandlers_AllEndpoints(t *testing.T) {
	t.Parallel()

	tests := []tautulliEndpointTest{
		// Core Tautulli Analytics Endpoints
		{
			name:        "TautulliHomeStats - success",
			handlerFunc: (*Handler).TautulliHomeStats,
			queryParams: "time_range=30&stats_type=plays&stats_count=10",
			setupMock: func(m *MockTautulliClient) {
				m.GetHomeStatsFunc = func(ctx context.Context, timeRange int, statsType string, statsCount int) (*tautulli.TautulliHomeStats, error) {
					return &tautulli.TautulliHomeStats{Response: tautulli.TautulliHomeStatsResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliPlaysByDate - success",
			handlerFunc: (*Handler).TautulliPlaysByDate,
			queryParams: "time_range=30&y_axis=plays&user_id=0&grouping=0",
		},
		{
			name:        "TautulliPlaysByDayOfWeek - success",
			handlerFunc: (*Handler).TautulliPlaysByDayOfWeek,
			queryParams: "time_range=30&y_axis=plays&user_id=0&grouping=0",
		},
		{
			name:        "TautulliPlaysByHourOfDay - success",
			handlerFunc: (*Handler).TautulliPlaysByHourOfDay,
			queryParams: "time_range=30&y_axis=plays&user_id=0&grouping=0",
		},
		{
			name:        "TautulliPlaysByStreamType - success",
			handlerFunc: (*Handler).TautulliPlaysByStreamType,
			queryParams: "time_range=30&y_axis=plays&user_id=0&grouping=0",
		},
		{
			name:        "TautulliConcurrentStreamsByStreamType - success",
			handlerFunc: (*Handler).TautulliConcurrentStreamsByStreamType,
			queryParams: "time_range=30&user_id=0",
		},
		{
			name:        "TautulliItemWatchTimeStats - success",
			handlerFunc: (*Handler).TautulliItemWatchTimeStats,
			queryParams: "rating_key=12345&grouping=0&query_days=30",
		},
		{
			name:        "TautulliActivity - success",
			handlerFunc: (*Handler).TautulliActivity,
			queryParams: "session_key=abc123",
			setupMock: func(m *MockTautulliClient) {
				m.GetActivityFunc = func(ctx context.Context, sessionKey string) (*tautulli.TautulliActivity, error) {
					return &tautulli.TautulliActivity{Response: tautulli.TautulliActivityResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliMetadata - success",
			handlerFunc: (*Handler).TautulliMetadata,
			queryParams: "rating_key=12345",
			setupMock: func(m *MockTautulliClient) {
				m.GetMetadataFunc = func(ctx context.Context, ratingKey string) (*tautulli.TautulliMetadata, error) {
					return &tautulli.TautulliMetadata{Response: tautulli.TautulliMetadataResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliUser - success",
			handlerFunc: (*Handler).TautulliUser,
			queryParams: "user_id=1",
			setupMock: func(m *MockTautulliClient) {
				m.GetUserFunc = func(ctx context.Context, userID int) (*tautulli.TautulliUser, error) {
					return &tautulli.TautulliUser{Response: tautulli.TautulliUserResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliUsers - success",
			handlerFunc: (*Handler).TautulliUsers,
			queryParams: "",
			setupMock: func(m *MockTautulliClient) {
				m.GetUsersFunc = func(ctx context.Context) (*tautulli.TautulliUsers, error) {
					return &tautulli.TautulliUsers{Response: tautulli.TautulliUsersResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliLibraryUserStats - success",
			handlerFunc: (*Handler).TautulliLibraryUserStats,
			queryParams: "section_id=1&grouping=0",
			setupMock: func(m *MockTautulliClient) {
				m.GetLibraryUserStatsFunc = func(ctx context.Context, sectionID int, grouping int) (*tautulli.TautulliLibraryUserStats, error) {
					return &tautulli.TautulliLibraryUserStats{Response: tautulli.TautulliLibraryUserStatsResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliRecentlyAdded - success",
			handlerFunc: (*Handler).TautulliRecentlyAdded,
			queryParams: "count=25&start=0&media_type=movie&section_id=1",
			setupMock: func(m *MockTautulliClient) {
				m.GetRecentlyAddedFunc = func(ctx context.Context, count int, start int, mediaType string, sectionID int) (*tautulli.TautulliRecentlyAdded, error) {
					return &tautulli.TautulliRecentlyAdded{Response: tautulli.TautulliRecentlyAddedResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliLibraries - success",
			handlerFunc: (*Handler).TautulliLibraries,
			setupMock: func(m *MockTautulliClient) {
				m.GetLibrariesFunc = func(ctx context.Context) (*tautulli.TautulliLibraries, error) {
					return &tautulli.TautulliLibraries{Response: tautulli.TautulliLibrariesResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliLibrary - success",
			handlerFunc: (*Handler).TautulliLibrary,
			queryParams: "section_id=1",
			setupMock: func(m *MockTautulliClient) {
				m.GetLibraryFunc = func(ctx context.Context, sectionID int) (*tautulli.TautulliLibrary, error) {
					return &tautulli.TautulliLibrary{Response: tautulli.TautulliLibraryResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliServerInfo - success",
			handlerFunc: (*Handler).TautulliServerInfo,
			setupMock: func(m *MockTautulliClient) {
				m.GetServerInfoFunc = func(ctx context.Context) (*tautulli.TautulliServerInfo, error) {
					return &tautulli.TautulliServerInfo{Response: tautulli.TautulliServerInfoResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliSyncedItems - success",
			handlerFunc: (*Handler).TautulliSyncedItems,
			queryParams: "machine_id=abc123&user_id=1",
			setupMock: func(m *MockTautulliClient) {
				m.GetSyncedItemsFunc = func(ctx context.Context, machineID string, userID int) (*tautulli.TautulliSyncedItems, error) {
					return &tautulli.TautulliSyncedItems{Response: tautulli.TautulliSyncedItemsResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliTerminateSession - success",
			handlerFunc: (*Handler).TautulliTerminateSession,
			httpMethod:  http.MethodPost,
			queryParams: "session_id=abc123&message=Test",
			setupMock: func(m *MockTautulliClient) {
				m.TerminateSessionFunc = func(ctx context.Context, sessionID string, message string) (*tautulli.TautulliTerminateSession, error) {
					return &tautulli.TautulliTerminateSession{Response: tautulli.TautulliTerminateSessionResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliPlaysBySourceResolution - success",
			handlerFunc: (*Handler).TautulliPlaysBySourceResolution,
			queryParams: "time_range=30&y_axis=plays&user_id=0&grouping=0",
			setupMock: func(m *MockTautulliClient) {
				m.GetPlaysBySourceResolutionFunc = func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysBySourceResolution, error) {
					return &tautulli.TautulliPlaysBySourceResolution{Response: tautulli.TautulliPlaysBySourceResolutionResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliPlaysByStreamResolution - success",
			handlerFunc: (*Handler).TautulliPlaysByStreamResolution,
			queryParams: "time_range=30&y_axis=plays&user_id=0&grouping=0",
			setupMock: func(m *MockTautulliClient) {
				m.GetPlaysByStreamResolutionFunc = func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByStreamResolution, error) {
					return &tautulli.TautulliPlaysByStreamResolution{Response: tautulli.TautulliPlaysByStreamResolutionResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliPlaysByTop10Platforms - success",
			handlerFunc: (*Handler).TautulliPlaysByTop10Platforms,
			queryParams: "time_range=30&y_axis=plays&user_id=0&grouping=0",
			setupMock: func(m *MockTautulliClient) {
				m.GetPlaysByTop10PlatformsFunc = func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Platforms, error) {
					return &tautulli.TautulliPlaysByTop10Platforms{Response: tautulli.TautulliPlaysByTop10PlatformsResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliPlaysByTop10Users - success",
			handlerFunc: (*Handler).TautulliPlaysByTop10Users,
			queryParams: "time_range=30&y_axis=plays&user_id=0&grouping=0",
			setupMock: func(m *MockTautulliClient) {
				m.GetPlaysByTop10UsersFunc = func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Users, error) {
					return &tautulli.TautulliPlaysByTop10Users{Response: tautulli.TautulliPlaysByTop10UsersResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliPlaysPerMonth - success",
			handlerFunc: (*Handler).TautulliPlaysPerMonth,
			queryParams: "time_range=30&y_axis=plays&user_id=0&grouping=0",
			setupMock: func(m *MockTautulliClient) {
				m.GetPlaysPerMonthFunc = func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysPerMonth, error) {
					return &tautulli.TautulliPlaysPerMonth{Response: tautulli.TautulliPlaysPerMonthResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliUserPlayerStats - success",
			handlerFunc: (*Handler).TautulliUserPlayerStats,
			queryParams: "user_id=1",
			setupMock: func(m *MockTautulliClient) {
				m.GetUserPlayerStatsFunc = func(ctx context.Context, userID int) (*tautulli.TautulliUserPlayerStats, error) {
					return &tautulli.TautulliUserPlayerStats{Response: tautulli.TautulliUserPlayerStatsResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliUserWatchTimeStats - success",
			handlerFunc: (*Handler).TautulliUserWatchTimeStats,
			queryParams: "user_id=1&query_days=30",
			setupMock: func(m *MockTautulliClient) {
				m.GetUserWatchTimeStatsFunc = func(ctx context.Context, userID int, queryDays string) (*tautulli.TautulliUserWatchTimeStats, error) {
					return &tautulli.TautulliUserWatchTimeStats{Response: tautulli.TautulliUserWatchTimeStatsResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliItemUserStats - success",
			handlerFunc: (*Handler).TautulliItemUserStats,
			queryParams: "rating_key=12345&grouping=0",
			setupMock: func(m *MockTautulliClient) {
				m.GetItemUserStatsFunc = func(ctx context.Context, ratingKey string, grouping int) (*tautulli.TautulliItemUserStats, error) {
					return &tautulli.TautulliItemUserStats{Response: tautulli.TautulliItemUserStatsResponse{Result: "success"}}, nil
				}
			},
		},
		// Library Management Endpoints
		{
			name:        "TautulliLibrariesTable - success",
			handlerFunc: (*Handler).TautulliLibrariesTable,
			queryParams: "grouping=0&order_column=section_name&order_dir=asc&start=0&length=25",
			setupMock: func(m *MockTautulliClient) {
				m.GetLibrariesTableFunc = func(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliLibrariesTable, error) {
					return &tautulli.TautulliLibrariesTable{Response: tautulli.TautulliLibrariesTableResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliLibraryMediaInfo - success",
			handlerFunc: (*Handler).TautulliLibraryMediaInfo,
			queryParams: "section_id=1&order_column=title&order_dir=asc&start=0&length=25",
			setupMock: func(m *MockTautulliClient) {
				m.GetLibraryMediaInfoFunc = func(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int) (*tautulli.TautulliLibraryMediaInfo, error) {
					return &tautulli.TautulliLibraryMediaInfo{Response: tautulli.TautulliLibraryMediaInfoResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliLibraryWatchTimeStats - success",
			handlerFunc: (*Handler).TautulliLibraryWatchTimeStats,
			queryParams: "section_id=1&grouping=0&query_days=30",
			setupMock: func(m *MockTautulliClient) {
				m.GetLibraryWatchTimeStatsFunc = func(ctx context.Context, sectionID int, grouping int, queryDays string) (*tautulli.TautulliLibraryWatchTimeStats, error) {
					return &tautulli.TautulliLibraryWatchTimeStats{Response: tautulli.TautulliLibraryWatchTimeStatsResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliChildrenMetadata - success",
			handlerFunc: (*Handler).TautulliChildrenMetadata,
			queryParams: "rating_key=12345&media_type=episode",
			setupMock: func(m *MockTautulliClient) {
				m.GetChildrenMetadataFunc = func(ctx context.Context, ratingKey string, mediaType string) (*tautulli.TautulliChildrenMetadata, error) {
					return &tautulli.TautulliChildrenMetadata{Response: tautulli.TautulliChildrenMetadataResponse{Result: "success"}}, nil
				}
			},
		},
		// User Management Endpoints
		{
			name:        "TautulliUserIPs - success",
			handlerFunc: (*Handler).TautulliUserIPs,
			queryParams: "user_id=1",
			setupMock: func(m *MockTautulliClient) {
				m.GetUserIPsFunc = func(ctx context.Context, userID int) (*tautulli.TautulliUserIPs, error) {
					return &tautulli.TautulliUserIPs{Response: tautulli.TautulliUserIPsResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliUsersTable - success",
			handlerFunc: (*Handler).TautulliUsersTable,
			queryParams: "grouping=0&order_column=username&order_dir=asc&start=0&length=25",
			setupMock: func(m *MockTautulliClient) {
				m.GetUsersTableFunc = func(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUsersTable, error) {
					return &tautulli.TautulliUsersTable{Response: tautulli.TautulliUsersTableResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliUserLogins - success",
			handlerFunc: (*Handler).TautulliUserLogins,
			queryParams: "user_id=1&order_column=date&order_dir=desc&start=0&length=25",
			setupMock: func(m *MockTautulliClient) {
				m.GetUserLoginsFunc = func(ctx context.Context, userID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUserLogins, error) {
					return &tautulli.TautulliUserLogins{Response: tautulli.TautulliUserLoginsResponse{Result: "success"}}, nil
				}
			},
		},
		// Metadata and Export Endpoints
		{
			name:        "TautulliStreamData - success",
			handlerFunc: (*Handler).TautulliStreamData,
			queryParams: "row_id=1&session_key=abc123",
			setupMock: func(m *MockTautulliClient) {
				m.GetStreamDataFunc = func(ctx context.Context, rowID int, sessionKey string) (*tautulli.TautulliStreamData, error) {
					return &tautulli.TautulliStreamData{Response: tautulli.TautulliStreamDataResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliLibraryNames - success",
			handlerFunc: (*Handler).TautulliLibraryNames,
			setupMock: func(m *MockTautulliClient) {
				m.GetLibraryNamesFunc = func(ctx context.Context) (*tautulli.TautulliLibraryNames, error) {
					return &tautulli.TautulliLibraryNames{Response: tautulli.TautulliLibraryNamesResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliExportMetadata - success",
			handlerFunc: (*Handler).TautulliExportMetadata,
			queryParams: "section_id=1&export_type=collection&user_id=0&rating_key=&file_format=csv",
			setupMock: func(m *MockTautulliClient) {
				m.ExportMetadataFunc = func(ctx context.Context, sectionID int, exportType string, userID int, ratingKey string, fileFormat string) (*tautulli.TautulliExportMetadata, error) {
					return &tautulli.TautulliExportMetadata{Response: tautulli.TautulliExportMetadataResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliExportFields - success",
			handlerFunc: (*Handler).TautulliExportFields,
			queryParams: "media_type=movie",
			setupMock: func(m *MockTautulliClient) {
				m.GetExportFieldsFunc = func(ctx context.Context, mediaType string) (*tautulli.TautulliExportFields, error) {
					return &tautulli.TautulliExportFields{Response: tautulli.TautulliExportFieldsResponse{Result: "success"}}, nil
				}
			},
		},
		// Advanced Analytics Endpoints
		{
			name:        "TautulliStreamTypeByTop10Users - success",
			handlerFunc: (*Handler).TautulliStreamTypeByTop10Users,
			queryParams: "time_range=30&y_axis=plays&user_id=0&grouping=0",
			setupMock: func(m *MockTautulliClient) {
				m.GetStreamTypeByTop10UsersFunc = func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliStreamTypeByTop10Users, error) {
					return &tautulli.TautulliStreamTypeByTop10Users{Response: tautulli.TautulliStreamTypeByTop10UsersResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliStreamTypeByTop10Platforms - success",
			handlerFunc: (*Handler).TautulliStreamTypeByTop10Platforms,
			queryParams: "time_range=30&y_axis=plays&user_id=0&grouping=0",
			setupMock: func(m *MockTautulliClient) {
				m.GetStreamTypeByTop10PlatformsFunc = func(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliStreamTypeByTop10Platforms, error) {
					return &tautulli.TautulliStreamTypeByTop10Platforms{Response: tautulli.TautulliStreamTypeByTop10PlatformsResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliSearch - success",
			handlerFunc: (*Handler).TautulliSearch,
			queryParams: "query=test&limit=10",
			setupMock: func(m *MockTautulliClient) {
				m.SearchFunc = func(ctx context.Context, query string, limit int) (*tautulli.TautulliSearch, error) {
					return &tautulli.TautulliSearch{Response: tautulli.TautulliSearchResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliNewRatingKeys - success",
			handlerFunc: (*Handler).TautulliNewRatingKeys,
			queryParams: "rating_key=12345",
			setupMock: func(m *MockTautulliClient) {
				m.GetNewRatingKeysFunc = func(ctx context.Context, ratingKey string) (*tautulli.TautulliNewRatingKeys, error) {
					return &tautulli.TautulliNewRatingKeys{Response: tautulli.TautulliNewRatingKeysResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliOldRatingKeys - success",
			handlerFunc: (*Handler).TautulliOldRatingKeys,
			queryParams: "rating_key=12345",
			setupMock: func(m *MockTautulliClient) {
				m.GetOldRatingKeysFunc = func(ctx context.Context, ratingKey string) (*tautulli.TautulliOldRatingKeys, error) {
					return &tautulli.TautulliOldRatingKeys{Response: tautulli.TautulliOldRatingKeysResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliCollectionsTable - success",
			handlerFunc: (*Handler).TautulliCollectionsTable,
			queryParams: "section_id=1&order_column=title&order_dir=asc&start=0&length=25",
			setupMock: func(m *MockTautulliClient) {
				m.GetCollectionsTableFunc = func(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliCollectionsTable, error) {
					return &tautulli.TautulliCollectionsTable{Response: tautulli.TautulliCollectionsTableResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliPlaylistsTable - success",
			handlerFunc: (*Handler).TautulliPlaylistsTable,
			queryParams: "section_id=1&order_column=title&order_dir=asc&start=0&length=25",
			setupMock: func(m *MockTautulliClient) {
				m.GetPlaylistsTableFunc = func(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliPlaylistsTable, error) {
					return &tautulli.TautulliPlaylistsTable{Response: tautulli.TautulliPlaylistsTableResponse{Result: "success"}}, nil
				}
			},
		},
		// Server Information Endpoints
		{
			name:        "TautulliServerFriendlyName - success",
			handlerFunc: (*Handler).TautulliServerFriendlyName,
			setupMock: func(m *MockTautulliClient) {
				m.GetServerFriendlyNameFunc = func(ctx context.Context) (*tautulli.TautulliServerFriendlyName, error) {
					return &tautulli.TautulliServerFriendlyName{Response: tautulli.TautulliServerFriendlyNameResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliServerID - success",
			handlerFunc: (*Handler).TautulliServerID,
			setupMock: func(m *MockTautulliClient) {
				m.GetServerIDFunc = func(ctx context.Context) (*tautulli.TautulliServerID, error) {
					return &tautulli.TautulliServerID{Response: tautulli.TautulliServerIDResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliServerIdentity - success",
			handlerFunc: (*Handler).TautulliServerIdentity,
			setupMock: func(m *MockTautulliClient) {
				m.GetServerIdentityFunc = func(ctx context.Context) (*tautulli.TautulliServerIdentity, error) {
					return &tautulli.TautulliServerIdentity{Response: tautulli.TautulliServerIdentityResponse{Result: "success"}}, nil
				}
			},
		},
		// Export Management Endpoints
		{
			name:        "TautulliExportsTable - success",
			handlerFunc: (*Handler).TautulliExportsTable,
			queryParams: "order_column=timestamp&order_dir=desc&start=0&length=25",
			setupMock: func(m *MockTautulliClient) {
				m.GetExportsTableFunc = func(ctx context.Context, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliExportsTable, error) {
					return &tautulli.TautulliExportsTable{Response: tautulli.TautulliExportsTableResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliDownloadExport - success",
			handlerFunc: (*Handler).TautulliDownloadExport,
			queryParams: "export_id=1",
			setupMock: func(m *MockTautulliClient) {
				m.DownloadExportFunc = func(ctx context.Context, exportID int) (*tautulli.TautulliDownloadExport, error) {
					return &tautulli.TautulliDownloadExport{Response: tautulli.TautulliDownloadExportResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliDeleteExport - success",
			handlerFunc: (*Handler).TautulliDeleteExport,
			queryParams: "export_id=1",
			setupMock: func(m *MockTautulliClient) {
				m.DeleteExportFunc = func(ctx context.Context, exportID int) (*tautulli.TautulliDeleteExport, error) {
					return &tautulli.TautulliDeleteExport{Response: tautulli.TautulliDeleteExportResponse{Result: "success"}}, nil
				}
			},
		},
		// Server Management Endpoints
		{
			name:        "TautulliTautulliInfo - success",
			handlerFunc: (*Handler).TautulliTautulliInfo,
			setupMock: func(m *MockTautulliClient) {
				m.GetTautulliInfoFunc = func(ctx context.Context) (*tautulli.TautulliTautulliInfo, error) {
					return &tautulli.TautulliTautulliInfo{Response: tautulli.TautulliTautulliInfoResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliServerPref - success",
			handlerFunc: (*Handler).TautulliServerPref,
			queryParams: "pref=FriendlyName",
			setupMock: func(m *MockTautulliClient) {
				m.GetServerPrefFunc = func(ctx context.Context, pref string) (*tautulli.TautulliServerPref, error) {
					return &tautulli.TautulliServerPref{Response: tautulli.TautulliServerPrefResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliServerList - success",
			handlerFunc: (*Handler).TautulliServerList,
			setupMock: func(m *MockTautulliClient) {
				m.GetServerListFunc = func(ctx context.Context) (*tautulli.TautulliServerList, error) {
					return &tautulli.TautulliServerList{Response: tautulli.TautulliServerListResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliServersInfo - success",
			handlerFunc: (*Handler).TautulliServersInfo,
			setupMock: func(m *MockTautulliClient) {
				m.GetServersInfoFunc = func(ctx context.Context) (*tautulli.TautulliServersInfo, error) {
					return &tautulli.TautulliServersInfo{Response: tautulli.TautulliServersInfoResponse{Result: "success"}}, nil
				}
			},
		},
		{
			name:        "TautulliPMSUpdate - success",
			handlerFunc: (*Handler).TautulliPMSUpdate,
			setupMock: func(m *MockTautulliClient) {
				m.GetPMSUpdateFunc = func(ctx context.Context) (*tautulli.TautulliPMSUpdate, error) {
					return &tautulli.TautulliPMSUpdate{Response: tautulli.TautulliPMSUpdateResponse{Result: "success"}}, nil
				}
			},
		},
	}

	runTautulliEndpointTests(t, tests, http.StatusOK)
}

// TestTautulliHandlers_ErrorCases tests error handling in Tautulli handlers
func TestTautulliHandlers_ErrorCases(t *testing.T) {
	t.Parallel()

	tests := []tautulliEndpointTest{
		{
			name:        "TautulliActivity - client error",
			handlerFunc: (*Handler).TautulliActivity,
			queryParams: "session_key=abc123",
			setupMock: func(m *MockTautulliClient) {
				m.GetActivityFunc = func(ctx context.Context, sessionKey string) (*tautulli.TautulliActivity, error) {
					return nil, fmt.Errorf("client error")
				}
			},
		},
		{
			name:        "TautulliMetadata - missing rating_key",
			handlerFunc: (*Handler).TautulliMetadata,
			setupMock: func(m *MockTautulliClient) {
				m.GetMetadataFunc = func(ctx context.Context, ratingKey string) (*tautulli.TautulliMetadata, error) {
					return nil, nil
				}
			},
		},
		{
			name:        "TautulliUser - invalid user_id",
			handlerFunc: (*Handler).TautulliUser,
			queryParams: "user_id=invalid",
			setupMock: func(m *MockTautulliClient) {
				m.GetUserFunc = func(ctx context.Context, userID int) (*tautulli.TautulliUser, error) {
					return nil, nil
				}
			},
		},
	}

	// First test expects internal server error (client error)
	t.Run(tests[0].name, func(t *testing.T) {
		mockClient := &MockTautulliClient{}
		tests[0].setupMock(mockClient)
		handler := setupTestHandlerWithMockClient(t, mockClient)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/test?"+tests[0].queryParams, nil)
		w := httptest.NewRecorder()
		tests[0].handlerFunc(handler, w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})

	// Remaining tests expect bad request (validation errors)
	for _, tt := range tests[1:] {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockTautulliClient{}
			tt.setupMock(mockClient)
			handler := setupTestHandlerWithMockClient(t, mockClient)
			url := "/api/v1/test"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			tt.handlerFunc(handler, w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}
		})
	}
}

// TestTautulliHandlers_CacheBehavior tests caching in Tautulli handlers
func TestTautulliHandlers_CacheBehavior(t *testing.T) {
	t.Parallel()

	t.Run("TautulliHomeStats - cache hit", func(t *testing.T) {
		callCount := 0
		mockClient := &MockTautulliClient{}
		mockClient.GetHomeStatsFunc = func(ctx context.Context, timeRange int, statsType string, statsCount int) (*tautulli.TautulliHomeStats, error) {
			callCount++
			return &tautulli.TautulliHomeStats{Response: tautulli.TautulliHomeStatsResponse{Result: "success"}}, nil
		}

		handler := setupTestHandlerWithMockClient(t, mockClient)

		// First call - should call client
		req1 := httptest.NewRequest(http.MethodGet, "/api/v1/test?time_range=30&stats_type=plays&stats_count=10", nil)
		w1 := httptest.NewRecorder()
		handler.TautulliHomeStats(w1, req1)

		if callCount != 1 {
			t.Errorf("Expected 1 client call, got %d", callCount)
		}

		// Second call with same parameters - should use cache
		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/test?time_range=30&stats_type=plays&stats_count=10", nil)
		w2 := httptest.NewRecorder()
		handler.TautulliHomeStats(w2, req2)

		if callCount != 1 {
			t.Errorf("Expected 1 client call (cached), got %d", callCount)
		}
	})
}
