// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// ===================================================================================================
// proxyTautulliRequest Generic Function Tests
// ===================================================================================================

// mockTautulliResponse simulates a typical Tautulli API response
type mockTautulliResponse struct {
	Response struct {
		Data interface{} `json:"data"`
	} `json:"response"`
}

func TestProxyTautulliRequest_HTTPMethodValidation(t *testing.T) {
	tests := []struct {
		name           string
		configMethod   string
		requestMethod  string
		expectedStatus int
	}{
		{
			name:           "GET allowed when config is GET",
			configMethod:   http.MethodGet,
			requestMethod:  http.MethodGet,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST rejected when config is GET",
			configMethod:   http.MethodGet,
			requestMethod:  http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "default GET when config is empty",
			configMethod:   "",
			requestMethod:  http.MethodGet,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST allowed when config is POST",
			configMethod:   http.MethodPost,
			requestMethod:  http.MethodPost,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET rejected when config is POST",
			configMethod:   http.MethodPost,
			requestMethod:  http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock handler with mock client (required for nil check)
			h := &Handler{
				cache:  cache.New(5 * time.Minute),
				client: &MockTautulliClient{}, // Required to pass nil client check
			}

			// Create config for simple NoParams -> mockTautulliResponse
			config := TautulliProxyConfig[NoParams, *mockTautulliResponse]{
				CacheName:  "",
				HTTPMethod: tt.configMethod,
				ParseParams: func(r *http.Request) (NoParams, error) {
					return NoParams{}, nil
				},
				CallClient: func(ctx context.Context, h *Handler, params NoParams) (*mockTautulliResponse, error) {
					return &mockTautulliResponse{}, nil
				},
				ExtractData: func(response *mockTautulliResponse) interface{} {
					return response.Response.Data
				},
				ErrorMessage: "Test error",
			}

			// Create request with specified method
			req := httptest.NewRequest(tt.requestMethod, "/test", nil)
			w := httptest.NewRecorder()

			// Execute
			proxyTautulliRequest(h, w, req, config)

			// Verify status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestProxyTautulliRequest_ParameterParsing(t *testing.T) {
	tests := []struct {
		name           string
		queryString    string
		parseFunc      func(r *http.Request) (StandardTimeRangeParams, error)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:        "successful parameter parsing",
			queryString: "time_range=30&y_axis=plays&user_id=1&grouping=0",
			parseFunc: func(r *http.Request) (StandardTimeRangeParams, error) {
				return parseStandardTimeRangeParams(r)
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:        "parameter parsing error",
			queryString: "invalid=data",
			parseFunc: func(r *http.Request) (StandardTimeRangeParams, error) {
				return StandardTimeRangeParams{}, errors.New("invalid parameters")
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name:        "missing optional parameters uses defaults",
			queryString: "",
			parseFunc: func(r *http.Request) (StandardTimeRangeParams, error) {
				return parseStandardTimeRangeParams(r)
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				cache:  cache.New(5 * time.Minute),
				client: &MockTautulliClient{}, // Required to pass nil client check
			}

			config := TautulliProxyConfig[StandardTimeRangeParams, *mockTautulliResponse]{
				CacheName:   "",
				HTTPMethod:  http.MethodGet,
				ParseParams: tt.parseFunc,
				CallClient: func(ctx context.Context, h *Handler, params StandardTimeRangeParams) (*mockTautulliResponse, error) {
					return &mockTautulliResponse{}, nil
				},
				ExtractData: func(response *mockTautulliResponse) interface{} {
					return response.Response.Data
				},
				ErrorMessage: "Test error",
			}

			url := "/test"
			if tt.queryString != "" {
				url += "?" + tt.queryString
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			proxyTautulliRequest(h, w, req, config)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError && w.Code != http.StatusBadRequest {
				t.Error("Expected error response")
			}
		})
	}
}

func TestProxyTautulliRequest_CacheBehavior(t *testing.T) {
	tests := []struct {
		name           string
		cacheName      string
		cacheEnabled   bool
		setupCache     func(*cache.Cache)
		expectedCached bool
	}{
		{
			name:         "cache hit returns cached data",
			cacheName:    "test_cache",
			cacheEnabled: true,
			setupCache: func(c *cache.Cache) {
				// Pre-populate cache with valid response
				params := StandardTimeRangeParams{TimeRange: 30, YAxis: "plays"}
				response := &mockTautulliResponse{}
				response.Response.Data = "cached_data"
				key := cache.GenerateKey("test_cache", params)
				c.Set(key, response)
			},
			expectedCached: true,
		},
		{
			name:         "cache miss fetches fresh data",
			cacheName:    "test_cache",
			cacheEnabled: true,
			setupCache: func(c *cache.Cache) {
				// Cache is empty
			},
			expectedCached: false,
		},
		{
			name:           "cache disabled always fetches fresh data",
			cacheName:      "",
			cacheEnabled:   false,
			setupCache:     nil,
			expectedCached: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c *cache.Cache
			if tt.cacheEnabled {
				c = cache.New(5 * time.Minute)
				if tt.setupCache != nil {
					tt.setupCache(c)
				}
			}

			h := &Handler{
				cache:  c,
				client: &MockTautulliClient{}, // Required to pass nil client check
			}

			clientCalled := false
			config := TautulliProxyConfig[StandardTimeRangeParams, *mockTautulliResponse]{
				CacheName:  tt.cacheName,
				HTTPMethod: http.MethodGet,
				ParseParams: func(r *http.Request) (StandardTimeRangeParams, error) {
					return StandardTimeRangeParams{TimeRange: 30, YAxis: "plays"}, nil
				},
				CallClient: func(ctx context.Context, h *Handler, params StandardTimeRangeParams) (*mockTautulliResponse, error) {
					clientCalled = true
					response := &mockTautulliResponse{}
					response.Response.Data = "fresh_data"
					return response, nil
				},
				ExtractData: func(response *mockTautulliResponse) interface{} {
					return response.Response.Data
				},
				ErrorMessage: "Test error",
			}

			req := httptest.NewRequest(http.MethodGet, "/test?time_range=30&y_axis=plays", nil)
			w := httptest.NewRecorder()

			proxyTautulliRequest(h, w, req, config)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			// If cache hit expected, client should NOT be called
			if tt.expectedCached && clientCalled {
				t.Error("Expected cache hit, but client was called")
			}

			// If cache miss expected, client SHOULD be called
			if !tt.expectedCached && !clientCalled {
				t.Error("Expected cache miss and client call, but client was not called")
			}
		})
	}
}

func TestProxyTautulliRequest_SafeTypeAssertion(t *testing.T) {
	tests := []struct {
		name               string
		cachedValue        interface{}
		expectedClientCall bool
	}{
		{
			name: "correct type in cache returns cached data",
			cachedValue: &mockTautulliResponse{
				Response: struct {
					Data interface{} `json:"data"`
				}{Data: "cached_data"},
			},
			expectedClientCall: false,
		},
		{
			name:               "wrong type in cache falls through to fresh fetch",
			cachedValue:        "wrong_type",
			expectedClientCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := cache.New(5 * time.Minute)

			// Pre-populate cache with test value
			params := StandardTimeRangeParams{TimeRange: 30, YAxis: "plays"}
			key := cache.GenerateKey("test_cache", params)
			c.Set(key, tt.cachedValue)

			h := &Handler{
				cache:  c,
				client: &MockTautulliClient{}, // Required to pass nil client check
			}

			clientCalled := false
			config := TautulliProxyConfig[StandardTimeRangeParams, *mockTautulliResponse]{
				CacheName:  "test_cache",
				HTTPMethod: http.MethodGet,
				ParseParams: func(r *http.Request) (StandardTimeRangeParams, error) {
					return StandardTimeRangeParams{TimeRange: 30, YAxis: "plays"}, nil
				},
				CallClient: func(ctx context.Context, h *Handler, params StandardTimeRangeParams) (*mockTautulliResponse, error) {
					clientCalled = true
					response := &mockTautulliResponse{}
					response.Response.Data = "fresh_data"
					return response, nil
				},
				ExtractData: func(response *mockTautulliResponse) interface{} {
					return response.Response.Data
				},
				ErrorMessage: "Test error",
			}

			req := httptest.NewRequest(http.MethodGet, "/test?time_range=30&y_axis=plays", nil)
			w := httptest.NewRecorder()

			proxyTautulliRequest(h, w, req, config)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			if clientCalled != tt.expectedClientCall {
				t.Errorf("Expected clientCalled=%v, got %v", tt.expectedClientCall, clientCalled)
			}
		})
	}
}

func TestProxyTautulliRequest_ClientCallErrors(t *testing.T) {
	tests := []struct {
		name           string
		clientError    error
		expectedStatus int
	}{
		{
			name:           "client call success",
			clientError:    nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "client call error returns 500",
			clientError:    errors.New("tautulli connection failed"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				cache:  cache.New(5 * time.Minute),
				client: &MockTautulliClient{}, // Required to pass nil client check
			}

			config := TautulliProxyConfig[NoParams, *mockTautulliResponse]{
				CacheName:  "",
				HTTPMethod: http.MethodGet,
				ParseParams: func(r *http.Request) (NoParams, error) {
					return NoParams{}, nil
				},
				CallClient: func(ctx context.Context, h *Handler, params NoParams) (*mockTautulliResponse, error) {
					if tt.clientError != nil {
						return nil, tt.clientError
					}
					return &mockTautulliResponse{}, nil
				},
				ExtractData: func(response *mockTautulliResponse) interface{} {
					return response.Response.Data
				},
				ErrorMessage: "Tautulli API error",
			}

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			proxyTautulliRequest(h, w, req, config)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestProxyTautulliRequest_NilClient(t *testing.T) {
	// Test that nil client returns 503 Service Unavailable
	h := &Handler{
		cache:  cache.New(5 * time.Minute),
		client: nil, // Deliberately nil to test the check
	}

	config := TautulliProxyConfig[NoParams, *mockTautulliResponse]{
		CacheName:  "",
		HTTPMethod: http.MethodGet,
		ParseParams: func(r *http.Request) (NoParams, error) {
			return NoParams{}, nil
		},
		CallClient: func(ctx context.Context, h *Handler, params NoParams) (*mockTautulliResponse, error) {
			return &mockTautulliResponse{}, nil
		},
		ExtractData: func(response *mockTautulliResponse) interface{} {
			return response.Response.Data
		},
		ErrorMessage: "Test error",
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	proxyTautulliRequest(h, w, req, config)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for nil client, got %d", w.Code)
	}
}

func TestProxyTautulliRequest_NilResponse(t *testing.T) {
	// Test that CallClient returning (nil, nil) returns 500 Internal Server Error
	h := &Handler{
		cache:  cache.New(5 * time.Minute),
		client: &MockTautulliClient{},
	}

	config := TautulliProxyConfig[NoParams, *mockTautulliResponse]{
		CacheName:  "",
		HTTPMethod: http.MethodGet,
		ParseParams: func(r *http.Request) (NoParams, error) {
			return NoParams{}, nil
		},
		CallClient: func(ctx context.Context, h *Handler, params NoParams) (*mockTautulliResponse, error) {
			// Simulate client returning nil response without error
			return nil, nil
		},
		ExtractData: func(response *mockTautulliResponse) interface{} {
			// This should never be called due to nil response check
			return response.Response.Data
		},
		ErrorMessage: "Test error",
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	proxyTautulliRequest(h, w, req, config)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for nil response, got %d", w.Code)
	}
}

// ===================================================================================================
// Parameter Parser Tests
// ===================================================================================================

func TestParseStandardTimeRangeParams(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected StandardTimeRangeParams
	}{
		{
			name:  "all parameters provided",
			query: "time_range=60&y_axis=duration&user_id=5&grouping=1",
			expected: StandardTimeRangeParams{
				TimeRange: 60,
				YAxis:     "duration",
				UserID:    5,
				Grouping:  1,
			},
		},
		{
			name:  "defaults applied",
			query: "",
			expected: StandardTimeRangeParams{
				TimeRange: 30,
				YAxis:     "plays",
				UserID:    0,
				Grouping:  0,
			},
		},
		{
			name:  "partial parameters with defaults",
			query: "time_range=90",
			expected: StandardTimeRangeParams{
				TimeRange: 90,
				YAxis:     "plays",
				UserID:    0,
				Grouping:  0,
			},
		},
		{
			name:  "y_axis only",
			query: "y_axis=duration",
			expected: StandardTimeRangeParams{
				TimeRange: 30,
				YAxis:     "duration",
				UserID:    0,
				Grouping:  0,
			},
		},
		{
			name:  "user_id only",
			query: "user_id=10",
			expected: StandardTimeRangeParams{
				TimeRange: 30,
				YAxis:     "plays",
				UserID:    10,
				Grouping:  0,
			},
		},
		{
			name:  "invalid time_range uses default",
			query: "time_range=abc",
			expected: StandardTimeRangeParams{
				TimeRange: 30,
				YAxis:     "plays",
				UserID:    0,
				Grouping:  0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			result, err := parseStandardTimeRangeParams(req)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestParseTimeRangeUserIDParams(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected TwoIntParams
	}{
		{
			name:  "both parameters provided",
			query: "time_range=60&user_id=5",
			expected: TwoIntParams{
				Param1: 60,
				Param2: 5,
			},
		},
		{
			name:  "defaults applied",
			query: "",
			expected: TwoIntParams{
				Param1: 30,
				Param2: 0,
			},
		},
		{
			name:  "time_range only",
			query: "time_range=90",
			expected: TwoIntParams{
				Param1: 90,
				Param2: 0,
			},
		},
		{
			name:  "user_id only",
			query: "user_id=10",
			expected: TwoIntParams{
				Param1: 30,
				Param2: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			result, err := parseTimeRangeUserIDParams(req)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestParseHomeStatsParams(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected HomeStatsParams
	}{
		{
			name:  "all parameters provided",
			query: "time_range=60&stats_type=duration&stats_count=20",
			expected: HomeStatsParams{
				TimeRange:  60,
				StatsType:  "duration",
				StatsCount: 20,
			},
		},
		{
			name:  "defaults applied",
			query: "",
			expected: HomeStatsParams{
				TimeRange:  30,
				StatsType:  "plays",
				StatsCount: 10,
			},
		},
		{
			name:  "stats_type only",
			query: "stats_type=concurrent",
			expected: HomeStatsParams{
				TimeRange:  30,
				StatsType:  "concurrent",
				StatsCount: 10,
			},
		},
		{
			name:  "stats_count only",
			query: "stats_count=15",
			expected: HomeStatsParams{
				TimeRange:  30,
				StatsType:  "plays",
				StatsCount: 15,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			result, err := parseHomeStatsParams(req)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestParseChildrenMetadataParamsTyped(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expected    ChildrenMetadataParams
		expectError bool
	}{
		{
			name:  "both parameters provided",
			query: "rating_key=12345&media_type=movie",
			expected: ChildrenMetadataParams{
				RatingKey: "12345",
				MediaType: "movie",
			},
			expectError: false,
		},
		{
			name:  "rating_key only (media_type optional)",
			query: "rating_key=12345",
			expected: ChildrenMetadataParams{
				RatingKey: "12345",
				MediaType: "",
			},
			expectError: false,
		},
		{
			name:        "missing required rating_key",
			query:       "media_type=movie",
			expected:    ChildrenMetadataParams{},
			expectError: true,
		},
		{
			name:        "empty rating_key",
			query:       "rating_key=&media_type=movie",
			expected:    ChildrenMetadataParams{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			result, err := parseChildrenMetadataParamsTyped(req)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %+v, got %+v", tt.expected, result)
				}
			}
		})
	}
}

func TestParseSearchParamsTyped(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expected    SearchParams
		expectError bool
	}{
		{
			name:  "both parameters provided",
			query: "query=star+wars&limit=50",
			expected: SearchParams{
				Query: "star wars",
				Limit: 50,
			},
			expectError: false,
		},
		{
			name:  "query only uses default limit",
			query: "query=test",
			expected: SearchParams{
				Query: "test",
				Limit: 25,
			},
			expectError: false,
		},
		{
			name:        "missing required query",
			query:       "limit=10",
			expected:    SearchParams{},
			expectError: true,
		},
		{
			name:        "empty query",
			query:       "query=",
			expected:    SearchParams{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			result, err := parseSearchParamsTyped(req)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result.Query != tt.expected.Query || result.Limit != tt.expected.Limit {
					t.Errorf("Expected %+v, got %+v", tt.expected, result)
				}
			}
		})
	}
}

func TestParseLibrariesParamsTyped(t *testing.T) {
	// NoParams parser should always succeed
	req := httptest.NewRequest(http.MethodGet, "/test?any=params", nil)
	result, err := parseLibrariesParamsTyped(req)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := NoParams{}
	if result != expected {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

func TestParseTerminateSessionParamsTyped(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expected    TerminateSessionParams
		expectError bool
	}{
		{
			name:  "both parameters provided",
			query: "session_id=abc123&message=Server+maintenance",
			expected: TerminateSessionParams{
				SessionID: "abc123",
				Message:   "Server maintenance",
			},
			expectError: false,
		},
		{
			name:  "session_id only (message optional)",
			query: "session_id=abc123",
			expected: TerminateSessionParams{
				SessionID: "abc123",
				Message:   "",
			},
			expectError: false,
		},
		{
			name:        "missing required session_id",
			query:       "message=test",
			expected:    TerminateSessionParams{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			result, err := parseTerminateSessionParamsTyped(req)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %+v, got %+v", tt.expected, result)
				}
			}
		})
	}
}

func TestParseUserParamsTyped(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expected    SingleIntParam
		expectError bool
	}{
		{
			name:  "valid user_id",
			query: "user_id=42",
			expected: SingleIntParam{
				Value: 42,
			},
			expectError: false,
		},
		{
			name:        "missing user_id",
			query:       "",
			expected:    SingleIntParam{},
			expectError: true,
		},
		{
			name:        "zero user_id",
			query:       "user_id=0",
			expected:    SingleIntParam{},
			expectError: true,
		},
		{
			name:        "invalid user_id",
			query:       "user_id=abc",
			expected:    SingleIntParam{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			result, err := parseUserParamsTyped(req)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %+v, got %+v", tt.expected, result)
				}
			}
		})
	}
}

// ===================================================================================================
// Integration Tests with Real Handlers
// ===================================================================================================

func TestProxyTautulliRequest_Integration(t *testing.T) {
	// Create a minimal handler with cache and mock client
	c := cache.New(5 * time.Minute)

	h := &Handler{
		cache:  c,
		client: &MockTautulliClient{}, // Required to pass nil client check
	}

	// Test with actual HomeStats config pattern
	config := TautulliProxyConfig[HomeStatsParams, *tautulli.TautulliHomeStats]{
		CacheName:   "home_stats",
		HTTPMethod:  http.MethodGet,
		ParseParams: parseHomeStatsParams,
		CallClient: func(ctx context.Context, h *Handler, params HomeStatsParams) (*tautulli.TautulliHomeStats, error) {
			// Mock successful response
			return &tautulli.TautulliHomeStats{
				Response: tautulli.TautulliHomeStatsResponse{
					Result:  "success",
					Message: nil,
					Data:    []tautulli.TautulliHomeStatRow{},
				},
			}, nil
		},
		ExtractData: func(response *tautulli.TautulliHomeStats) interface{} {
			return response.Response.Data
		},
		ErrorMessage: "Failed to fetch home stats",
	}

	// First request - should call client
	req1 := httptest.NewRequest(http.MethodGet, "/test?time_range=30&stats_type=plays&stats_count=10", nil)
	w1 := httptest.NewRecorder()
	proxyTautulliRequest(h, w1, req1, config)

	if w1.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w1.Code)
	}

	// Second identical request - should hit cache
	req2 := httptest.NewRequest(http.MethodGet, "/test?time_range=30&stats_type=plays&stats_count=10", nil)
	w2 := httptest.NewRecorder()
	proxyTautulliRequest(h, w2, req2, config)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w2.Code)
	}

	// Both responses should have proper JSON format
	if w1.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected application/json content type")
	}
	if w2.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected application/json content type")
	}
}

// ===================================================================================================
// Benchmarks
// ===================================================================================================

func BenchmarkProxyTautulliRequest_WithCache(b *testing.B) {
	h := &Handler{
		cache:  cache.New(5 * time.Minute),
		client: &MockTautulliClient{}, // Required to pass nil client check
	}

	config := TautulliProxyConfig[NoParams, *mockTautulliResponse]{
		CacheName:  "bench_cache",
		HTTPMethod: http.MethodGet,
		ParseParams: func(r *http.Request) (NoParams, error) {
			return NoParams{}, nil
		},
		CallClient: func(ctx context.Context, h *Handler, params NoParams) (*mockTautulliResponse, error) {
			return &mockTautulliResponse{}, nil
		},
		ExtractData: func(response *mockTautulliResponse) interface{} {
			return response.Response.Data
		},
		ErrorMessage: "Test error",
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		proxyTautulliRequest(h, w, req, config)
	}
}

func BenchmarkParseStandardTimeRangeParams(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/test?time_range=60&y_axis=duration&user_id=5&grouping=1", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parseStandardTimeRangeParams(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
