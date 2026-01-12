// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/middleware"
	"github.com/tomtom215/cartographus/internal/models"
)

// Test helpers to reduce cyclomatic complexity

// assertStatusCode checks HTTP response status code
func assertStatusCode(t *testing.T, got, want int, testName string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: expected status %d, got %d", testName, want, got)
	}
}

// decodeAPIResponse decodes and validates API response
func decodeAPIResponse(t *testing.T, w *httptest.ResponseRecorder, testName string) *models.APIResponse {
	t.Helper()
	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("%s: failed to decode response: %v", testName, err)
	}
	return &response
}

// assertResponseSuccess checks if response status is success
func assertResponseSuccess(t *testing.T, response *models.APIResponse, testName string) {
	t.Helper()
	if response.Status != "success" {
		t.Errorf("%s: expected status 'success', got '%s'", testName, response.Status)
	}
}

// assertMapData extracts and validates response data as map
func assertMapData(t *testing.T, response *models.APIResponse, testName string) map[string]interface{} {
	t.Helper()
	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("%s: response data is not a map", testName)
	}
	return data
}

// assertArrayLength checks array length
func assertArrayLength(t *testing.T, arr []interface{}, expected int, testName string) {
	t.Helper()
	if len(arr) != expected {
		t.Errorf("%s: expected %d items, got %d", testName, expected, len(arr))
	}
}

// executeRequest executes an HTTP request and returns the recorder
func executeRequest(handler http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}

// assertValidJSONResponse validates JSON response structure and status
func assertValidJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, testName string) *models.APIResponse {
	t.Helper()
	assertStatusCode(t, w.Code, expectedStatus, testName)
	response := decodeAPIResponse(t, w, testName)
	assertResponseSuccess(t, response, testName)
	return response
}

// extractEventsArray extracts events array from response data map
func extractEventsArray(t *testing.T, response *models.APIResponse, testName string) []interface{} {
	t.Helper()
	dataMap := assertMapData(t, response, testName)
	events, ok := dataMap["events"].([]interface{})
	if !ok {
		t.Fatalf("%s: response data.events is not an array", testName)
	}
	return events
}

// assertPaginationResponse validates paginated response structure
func assertPaginationResponse(t *testing.T, w *httptest.ResponseRecorder, expectedEventCount int, testName string) *models.APIResponse {
	t.Helper()
	response := assertValidJSONResponse(t, w, http.StatusOK, testName)
	events := extractEventsArray(t, response, testName)
	assertArrayLength(t, events, expectedEventCount, testName+" events")
	return response
}

// assertEmptyEventsResponse validates response with empty events array
func assertEmptyEventsResponse(t *testing.T, w *httptest.ResponseRecorder, testName string) {
	t.Helper()
	response := assertValidJSONResponse(t, w, http.StatusOK, testName)
	events := extractEventsArray(t, response, testName)
	assertArrayLength(t, events, 0, testName+" events")
}

// testEndpointWithParams tests endpoint with specific query parameters
func testEndpointWithParams(t *testing.T, handler http.HandlerFunc, path, params string, expectedStatus int, testName string) *httptest.ResponseRecorder {
	t.Helper()
	url := path
	if params != "" {
		url = path + "?" + params
	}
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := executeRequest(handler, req)
	assertStatusCode(t, w.Code, expectedStatus, testName)
	return w
}

// setupTestDBForAPI creates a new in-memory test database for API handler tests
func setupTestDBForAPI(t *testing.T) *database.DB {
	t.Helper()
	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, err := database.New(cfg, 0.0, 0.0)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	return db
}

// setupTestHandlerWithDB creates a handler with real DB and mock dependencies
// Note: sync is set to nil - tests that need sync functionality should use handlers_test.go
func setupTestHandlerWithDB(t *testing.T, db *database.DB) *Handler {
	t.Helper()
	cfg := &config.Config{
		API: config.APIConfig{
			DefaultPageSize: 100,
			MaxPageSize:     1000,
		},
		Server: config.ServerConfig{
			Latitude:  40.7128,
			Longitude: -74.0060,
		},
		Security: config.SecurityConfig{
			CORSOrigins: []string{"*"},
		},
	}

	return &Handler{
		db:        db,
		sync:      nil, // sync is nil - tests that need it should skip
		client:    &MockTautulliClient{},
		config:    cfg,
		startTime: time.Now(),
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}
}

// insertTestPlaybacks inserts test playback data using exported database methods
func insertTestPlaybacks(t *testing.T, db *database.DB, count int) {
	t.Helper()
	now := time.Now()

	users := []string{"user1", "user2", "user3"}
	mediaTypes := []string{"movie", "episode", "track"}

	for i := 0; i < count; i++ {
		user := users[i%len(users)]
		mediaType := mediaTypes[i%len(mediaTypes)]
		startedAt := now.Add(-time.Duration(i) * time.Hour)
		stoppedAt := startedAt.Add(30 * time.Minute)

		event := &models.PlaybackEvent{
			ID:              uuid.New(),
			SessionKey:      uuid.New().String(),
			StartedAt:       startedAt,
			StoppedAt:       &stoppedAt,
			UserID:          (i % 3) + 1,
			Username:        user,
			IPAddress:       "192.168.1." + string(rune('1'+i%3)),
			MediaType:       mediaType,
			Title:           "Test Title " + string(rune('A'+i%26)),
			PercentComplete: 100,
		}

		if err := db.InsertPlaybackEvent(event); err != nil {
			t.Logf("Warning: Failed to insert playback event %d: %v", i, err)
		}
	}
}

// TestStats_WithDB tests the Stats handler with real database
func TestStats_WithDB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	insertTestPlaybacks(t, db, 10)
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	w := httptest.NewRecorder()

	handler.Stats(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "TestStats_WithDB")
	response := decodeAPIResponse(t, w, "TestStats_WithDB")
	assertResponseSuccess(t, response, "TestStats_WithDB")
	data := assertMapData(t, response, "TestStats_WithDB")

	if data["total_playbacks"] == nil {
		t.Error("Expected total_playbacks in response")
	}
}

// TestStats_EmptyDB tests the Stats handler with empty database
func TestStats_EmptyDB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	w := httptest.NewRecorder()

	handler.Stats(w, req)

	assertStatusCode(t, w.Code, http.StatusOK, "TestStats_EmptyDB")
	response := decodeAPIResponse(t, w, "TestStats_EmptyDB")
	assertResponseSuccess(t, response, "TestStats_EmptyDB")
}

// TestPlaybacks_WithDB tests paginated playback retrieval with real database
func TestPlaybacks_WithDB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	insertTestPlaybacks(t, db, 20)
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/playbacks?limit=10&offset=0", nil)
	w := executeRequest(handler.Playbacks, req)
	assertPaginationResponse(t, w, 10, "TestPlaybacks_WithDB")
}

// TestPlaybacks_Pagination_WithDB tests pagination with offset using real database
func TestPlaybacks_Pagination_WithDB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	insertTestPlaybacks(t, db, 30)
	handler := setupTestHandlerWithDB(t, db)

	// Get first page (offset=0 uses cursor-based internally)
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/playbacks?limit=10&offset=0", nil)
	w1 := httptest.NewRecorder()
	handler.Playbacks(w1, req1)

	var resp1 models.APIResponse
	if err := json.NewDecoder(w1.Body).Decode(&resp1); err != nil {
		t.Fatalf("Failed to decode first page response: %v", err)
	}

	// Extract events from first page
	dataMap1, ok := resp1.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Response data is not a map (expected PlaybacksResponse structure)")
	}
	events1, ok := dataMap1["events"].([]interface{})
	if !ok {
		t.Fatal("Response data.events is not an array")
	}

	if len(events1) != 10 {
		t.Errorf("Expected 10 results in first page, got %d", len(events1))
	}

	// Get next cursor from pagination info for second page
	pagination, ok := dataMap1["pagination"].(map[string]interface{})
	if !ok {
		t.Fatal("Response pagination is not a map")
	}
	nextCursor, _ := pagination["next_cursor"].(string)

	// Get second page using cursor
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/playbacks?limit=10&cursor="+nextCursor, nil)
	w2 := httptest.NewRecorder()
	handler.Playbacks(w2, req2)

	var resp2 models.APIResponse
	if err := json.NewDecoder(w2.Body).Decode(&resp2); err != nil {
		t.Fatalf("Failed to decode second page response: %v", err)
	}

	dataMap2, ok := resp2.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Second page response data is not a map")
	}
	events2, ok := dataMap2["events"].([]interface{})
	if !ok {
		t.Fatal("Second page response data.events is not an array")
	}

	if len(events2) != 10 {
		t.Errorf("Expected 10 results in second page, got %d", len(events2))
	}
}

// TestPlaybacks_EmptyDB tests playbacks endpoint with empty database
func TestPlaybacks_EmptyDB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/playbacks", nil)
	w := executeRequest(handler.Playbacks, req)
	assertEmptyEventsResponse(t, w, "TestPlaybacks_EmptyDB")
}

// TestPlaybacks_InvalidLimit tests validation for invalid limit values
func TestPlaybacks_InvalidLimit_WithDB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	tests := []struct {
		name   string
		limit  string
		expect int
	}{
		{"Limit too high", "2000", http.StatusBadRequest},
		{"Negative limit", "-1", http.StatusBadRequest},
		{"Zero limit", "0", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEndpointWithParams(t, handler.Playbacks, "/api/v1/playbacks", "limit="+tt.limit, tt.expect, tt.name)
		})
	}
}

// TestPlaybacks_InvalidOffset tests validation for invalid offset values
func TestPlaybacks_InvalidOffset_WithDB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	tests := []struct {
		name   string
		offset string
		expect int
	}{
		{"Negative offset", "-1", http.StatusBadRequest},
		{"Offset too large", "2000000", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEndpointWithParams(t, handler.Playbacks, "/api/v1/playbacks", "offset="+tt.offset, tt.expect, tt.name)
		})
	}
}

// TestLocations_WithDB tests location statistics retrieval with real database
func TestLocations_WithDB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	insertTestPlaybacks(t, db, 15)
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/locations", nil)
	w := executeRequest(handler.Locations, req)
	assertValidJSONResponse(t, w, http.StatusOK, "TestLocations_WithDB")
}

// TestLocations_WithDateFilter tests location filtering by date range
func TestLocations_WithDateFilter_DB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	insertTestPlaybacks(t, db, 10)
	handler := setupTestHandlerWithDB(t, db)

	startDate := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	testEndpointWithParams(t, handler.Locations, "/api/v1/locations", "start_date="+url.QueryEscape(startDate), http.StatusOK, "TestLocations_WithDateFilter_DB")
}

// TestLocations_InvalidStartDate tests date validation
func TestLocations_InvalidStartDate_DB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)
	testEndpointWithParams(t, handler.Locations, "/api/v1/locations", "start_date=invalid", http.StatusBadRequest, "TestLocations_InvalidStartDate_DB")
}

// TestLocations_WithDaysFilter tests the days filter shorthand
func TestLocations_WithDaysFilter_DB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	insertTestPlaybacks(t, db, 10)
	handler := setupTestHandlerWithDB(t, db)
	testEndpointWithParams(t, handler.Locations, "/api/v1/locations", "days=7", http.StatusOK, "TestLocations_WithDaysFilter_DB")
}

// TestLocations_InvalidDays_DB tests days validation
func TestLocations_InvalidDays_DB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	tests := []struct {
		name string
		days string
	}{
		{"Days too small", "0"},
		{"Days too large", "4000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEndpointWithParams(t, handler.Locations, "/api/v1/locations", "days="+tt.days, http.StatusBadRequest, tt.name)
		})
	}
}

// TestArrayEndpoints_WithDB tests list-based endpoints with table-driven tests
func TestArrayEndpoints_WithDB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	insertTestPlaybacks(t, db, 15)
	handler := setupTestHandlerWithDB(t, db)

	tests := []struct {
		name      string
		path      string
		handler   http.HandlerFunc
		minLength int
	}{
		{"Users list", "/api/v1/users", handler.Users, 1},
		{"Media types list", "/api/v1/media-types", handler.MediaTypes, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := executeRequest(tt.handler, req)

			assertStatusCode(t, w.Code, http.StatusOK, tt.name)
			response := decodeAPIResponse(t, w, tt.name)
			assertResponseSuccess(t, response, tt.name)

			data, ok := response.Data.([]interface{})
			if !ok {
				t.Fatalf("%s: Response data is not an array", tt.name)
			}
			if len(data) < tt.minLength {
				t.Errorf("%s: Expected at least %d items, got %d", tt.name, tt.minLength, len(data))
			}
		})
	}
}

// TestArrayEndpoints_EmptyDB tests list-based endpoints with empty database
func TestArrayEndpoints_EmptyDB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	tests := []struct {
		name    string
		path    string
		handler http.HandlerFunc
	}{
		{"Users empty", "/api/v1/users", handler.Users},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := executeRequest(tt.handler, req)

			assertStatusCode(t, w.Code, http.StatusOK, tt.name)
			response := decodeAPIResponse(t, w, tt.name)

			data, ok := response.Data.([]interface{})
			if !ok {
				t.Fatalf("%s: Response data is not an array", tt.name)
			}
			if len(data) != 0 {
				t.Errorf("%s: Expected empty array, got %d items", tt.name, len(data))
			}
		})
	}
}

// TestServerInfo_WithDB tests server info retrieval
func TestServerInfo_WithDB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/server-info", nil)
	w := httptest.NewRecorder()

	handler.ServerInfo(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Response data is not a map")
	}

	if data["has_location"] != true {
		t.Error("Expected has_location to be true")
	}

	if data["latitude"] != 40.7128 {
		t.Errorf("Expected latitude 40.7128, got %v", data["latitude"])
	}

	if data["longitude"] != -74.0060 {
		t.Errorf("Expected longitude -74.0060, got %v", data["longitude"])
	}
}

// TestTriggerSync_WithDB tests sync trigger endpoint
// Note: This test is skipped because it requires a real sync.Manager,
// which cannot be easily mocked without modifying production code.
// The TriggerSync handler is tested in handlers_test.go with a mock client.
func TestTriggerSync_WithDB(t *testing.T) {
	t.Skip("Skipped: TriggerSync requires sync.Manager which cannot be mocked in this test setup")
}

// TestClearCache tests cache clearing
func TestClearCache_WithDB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	// Add something to cache
	handler.cache.Set("test_key", "test_value")

	// Clear cache
	handler.ClearCache()

	// Verify cache is empty
	if _, found := handler.cache.Get("test_key"); found {
		t.Error("Expected cache to be empty after ClearCache")
	}
}

// TestClearCache_NilCache tests clearing nil cache
func TestClearCache_NilCache(t *testing.T) {
	t.Parallel()
	handler := &Handler{cache: nil}

	// Should not panic
	handler.ClearCache()
}

// TestGetCacheStats tests cache stats retrieval
func TestGetCacheStats_WithDB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	// Add and retrieve from cache
	handler.cache.Set("test_key", "test_value")
	handler.cache.Get("test_key")
	handler.cache.Get("nonexistent")

	stats := handler.GetCacheStats()

	if stats.Hits < 1 {
		t.Errorf("Expected at least 1 hit, got %d", stats.Hits)
	}
	if stats.Misses < 1 {
		t.Errorf("Expected at least 1 miss, got %d", stats.Misses)
	}
}

// TestGetCacheStats_NilCache tests stats with nil cache
func TestGetCacheStats_NilCache(t *testing.T) {
	t.Parallel()
	handler := &Handler{cache: nil}

	stats := handler.GetCacheStats()

	// Should return zero stats without panicking
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Error("Expected zero stats for nil cache")
	}
}

// TestGetPerformanceStats tests performance stats retrieval
func TestGetPerformanceStats_WithDB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)
	stats := handler.GetPerformanceStats()

	// Initially should be empty or nil
	if stats == nil {
		// This is acceptable for a new handler
		return
	}
}

// TestGetPerformanceStats_NilMonitor tests stats with nil monitor
func TestGetPerformanceStats_NilMonitor(t *testing.T) {
	t.Parallel()
	handler := &Handler{perfMon: nil}

	stats := handler.GetPerformanceStats()

	if stats != nil {
		t.Error("Expected nil stats for nil monitor")
	}
}

// TestHealthEndpoints_WithDB tests health endpoints with table-driven tests
func TestHealthEndpoints_WithDB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	tests := []struct {
		name           string
		path           string
		handler        http.HandlerFunc
		allowedCodes   []int
		expectedStatus string // expected status field in response
		requiredKey    string
		expectedVal    interface{} // nil means just check key exists
	}{
		{"Health check", "/api/v1/health", handler.Health, []int{http.StatusOK}, "success", "database_connected", nil},
		{"Liveness probe", "/api/v1/health/live", handler.HealthLive, []int{http.StatusOK}, "success", "alive", true},
		{"Readiness probe", "/api/v1/health/ready", handler.HealthReady, []int{http.StatusOK, http.StatusServiceUnavailable}, "ready", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := executeRequest(tt.handler, req)

			// Check if status code is in allowed list
			validCode := false
			for _, code := range tt.allowedCodes {
				if w.Code == code {
					validCode = true
					break
				}
			}
			if !validCode {
				t.Errorf("%s: unexpected status %d, expected one of %v", tt.name, w.Code, tt.allowedCodes)
			}

			// Only validate response body if we expect success
			if w.Code == http.StatusOK {
				response := decodeAPIResponse(t, w, tt.name)
				if response.Status != tt.expectedStatus {
					t.Errorf("%s: expected status '%s', got '%s'", tt.name, tt.expectedStatus, response.Status)
				}

				if tt.requiredKey != "" {
					data := assertMapData(t, response, tt.name)
					val, exists := data[tt.requiredKey]
					if !exists {
						t.Errorf("%s: expected %s in response", tt.name, tt.requiredKey)
					}
					if tt.expectedVal != nil && val != tt.expectedVal {
						t.Errorf("%s: expected %s=%v, got %v", tt.name, tt.requiredKey, tt.expectedVal, val)
					}
				}
			}
		})
	}
}

// TestOnSyncCompleted tests the sync completion callback
func TestOnSyncCompleted(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	// Add something to cache
	handler.cache.Set("test_key", "test_value")

	// Trigger sync completion
	handler.OnSyncCompleted(100, 500)

	// Verify cache was cleared
	if _, found := handler.cache.Get("test_key"); found {
		t.Error("Expected cache to be cleared after sync completion")
	}
}

// TestAnalytics endpoints with DB

// TestAnalyticsEndpoints_WithDB tests analytics endpoints with table-driven tests
func TestAnalyticsEndpoints_WithDB(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	insertTestPlaybacks(t, db, 25)
	handler := setupTestHandlerWithDB(t, db)

	tests := []struct {
		name    string
		path    string
		handler http.HandlerFunc
	}{
		{"Trends analytics", "/api/v1/analytics/trends?days=7", handler.AnalyticsTrends},
		{"Geographic analytics", "/api/v1/analytics/geographic", handler.AnalyticsGeographic},
		{"User analytics", "/api/v1/analytics/users", handler.AnalyticsUsers},
		{"Popular analytics", "/api/v1/analytics/popular", handler.AnalyticsPopular},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := executeRequest(tt.handler, req)
			assertStatusCode(t, w.Code, http.StatusOK, tt.name)
			response := decodeAPIResponse(t, w, tt.name)
			assertResponseSuccess(t, response, tt.name)
		})
	}
}

// Test helper functions

func TestEscapeCSV(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Simple string", "hello", "hello"},
		{"String with comma", "hello,world", "\"hello,world\""},
		{"String with quote", "hello\"world", "\"hello\"\"world\""},
		{"String with newline", "hello\nworld", "\"hello\nworld\""},
		{"Empty string", "", ""},
		{"String with multiple special chars", "a,b\"c\nd", "\"a,b\"\"c\nd\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeCSV(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGenerateETag(t *testing.T) {
	t.Parallel()

	// Same input should produce same ETag
	data := []byte("test data")
	etag1 := generateETag(data)
	etag2 := generateETag(data)

	if etag1 != etag2 {
		t.Errorf("Same input should produce same ETag: %s != %s", etag1, etag2)
	}

	// Different input should produce different ETag
	data2 := []byte("different data")
	etag3 := generateETag(data2)

	if etag1 == etag3 {
		t.Error("Different input should produce different ETag")
	}

	// Empty data should produce valid ETag
	emptyEtag := generateETag([]byte{})
	if emptyEtag == "" {
		t.Error("Empty data should produce non-empty ETag")
	}
}

func TestParseIntParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		value        string
		defaultValue int
		expected     int
	}{
		{"Valid number", "42", 0, 42},
		{"Empty string", "", 10, 10},
		{"Invalid string", "abc", 5, 5},
		{"Negative number", "-5", 0, -5},
		{"Zero", "0", 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIntParam(tt.value, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestParseCommaSeparatedInts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []int
	}{
		{"Single value", "42", []int{42}},
		{"Multiple values", "1,2,3", []int{1, 2, 3}},
		{"Empty string", "", nil},
		{"With invalid values", "1,abc,3", []int{1, 3}},
		{"With spaces", "1, 2, 3", []int{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparatedInts(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d items, got %d", len(tt.expected), len(result))
				return
			}
			for i, v := range tt.expected {
				if result[i] != v {
					t.Errorf("Expected %d at index %d, got %d", v, i, result[i])
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkStats_WithDB(b *testing.B) {
	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}
	db, _ := database.New(cfg, 0.0, 0.0)
	defer db.Close()

	handler := &Handler{
		db:        db,
		sync:      nil,
		client:    &MockTautulliClient{},
		config:    &config.Config{API: config.APIConfig{DefaultPageSize: 100, MaxPageSize: 1000}},
		startTime: time.Now(),
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.Stats(w, req)
	}
}

func BenchmarkPlaybacks_WithDB(b *testing.B) {
	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}
	db, _ := database.New(cfg, 0.0, 0.0)
	defer db.Close()

	handler := &Handler{
		db:        db,
		sync:      nil,
		client:    &MockTautulliClient{},
		config:    &config.Config{API: config.APIConfig{DefaultPageSize: 100, MaxPageSize: 1000}},
		startTime: time.Now(),
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/playbacks?limit=10", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.Playbacks(w, req)
	}
}

func BenchmarkEscapeCSV(b *testing.B) {
	input := "test,value\"with,special\ncharacters"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		escapeCSV(input)
	}
}

func BenchmarkGenerateETag(b *testing.B) {
	data := []byte(`{"status":"success","data":{"total_playbacks":1000,"unique_locations":50}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateETag(data)
	}
}

// Test context cancellation
func TestStats_ContextCancellation(t *testing.T) {
	t.Parallel()
	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := setupTestHandlerWithDB(t, db)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.Stats(w, req)

	// Should still return a response (may be error due to canceled context)
	// The key is that it doesn't hang
}
