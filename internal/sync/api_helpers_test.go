// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goccy/go-json"
)

// TestNewAPIRequest tests the API request builder constructor
func TestNewAPIRequest(t *testing.T) {
	req := newAPIRequest("get_history")

	if req.cmd != "get_history" {
		t.Errorf("cmd: expected get_history, got %s", req.cmd)
	}

	if req.params == nil {
		t.Error("params should not be nil")
	}

	if len(req.params) != 0 {
		t.Errorf("params should be empty, got %d items", len(req.params))
	}
}

// TestAPIRequestAddParam tests adding string parameters
func TestAPIRequestAddParam(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		expectSet bool
	}{
		{
			name:      "non-empty value is added",
			key:       "user_id",
			value:     "123",
			expectSet: true,
		},
		{
			name:      "empty value is not added",
			key:       "user_id",
			value:     "",
			expectSet: false,
		},
		{
			name:      "whitespace value is added",
			key:       "search",
			value:     "  ",
			expectSet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newAPIRequest("test_cmd")
			result := req.addParam(tt.key, tt.value)

			// Verify fluent interface
			if result != req {
				t.Error("addParam should return the same request for chaining")
			}

			val, exists := req.params[tt.key]
			if tt.expectSet {
				if !exists {
					t.Errorf("param %s should exist", tt.key)
				} else if val != tt.value {
					t.Errorf("param %s: expected %q, got %q", tt.key, tt.value, val)
				}
			} else {
				if exists {
					t.Errorf("param %s should not exist", tt.key)
				}
			}
		})
	}
}

// TestAPIRequestAddIntParam tests adding integer parameters
func TestAPIRequestAddIntParam(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     int
		expectSet bool
		expected  string
	}{
		{
			name:      "positive int is added",
			key:       "user_id",
			value:     123,
			expectSet: true,
			expected:  "123",
		},
		{
			name:      "zero is not added",
			key:       "user_id",
			value:     0,
			expectSet: false,
		},
		{
			name:      "negative is not added",
			key:       "user_id",
			value:     -1,
			expectSet: false,
		},
		{
			name:      "large positive is added",
			key:       "time_range",
			value:     99999999,
			expectSet: true,
			expected:  "99999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newAPIRequest("test_cmd")
			result := req.addIntParam(tt.key, tt.value)

			// Verify fluent interface
			if result != req {
				t.Error("addIntParam should return the same request for chaining")
			}

			val, exists := req.params[tt.key]
			if tt.expectSet {
				if !exists {
					t.Errorf("param %s should exist", tt.key)
				} else if val != tt.expected {
					t.Errorf("param %s: expected %q, got %q", tt.key, tt.expected, val)
				}
			} else {
				if exists {
					t.Errorf("param %s should not exist", tt.key)
				}
			}
		})
	}
}

// TestAPIRequestAddIntParamZero tests adding integer parameters including zero
func TestAPIRequestAddIntParamZero(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     int
		expectSet bool
		expected  string
	}{
		{
			name:      "positive int is added",
			key:       "grouping",
			value:     1,
			expectSet: true,
			expected:  "1",
		},
		{
			name:      "zero is added (unlike addIntParam)",
			key:       "grouping",
			value:     0,
			expectSet: true,
			expected:  "0",
		},
		{
			name:      "negative is not added",
			key:       "grouping",
			value:     -1,
			expectSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newAPIRequest("test_cmd")
			result := req.addIntParamZero(tt.key, tt.value)

			// Verify fluent interface
			if result != req {
				t.Error("addIntParamZero should return the same request for chaining")
			}

			val, exists := req.params[tt.key]
			if tt.expectSet {
				if !exists {
					t.Errorf("param %s should exist", tt.key)
				} else if val != tt.expected {
					t.Errorf("param %s: expected %q, got %q", tt.key, tt.expected, val)
				}
			} else {
				if exists {
					t.Errorf("param %s should not exist", tt.key)
				}
			}
		})
	}
}

// TestAPIRequestBuildURL tests URL building with parameters
func TestAPIRequestBuildURL(t *testing.T) {
	tests := []struct {
		name      string
		cmd       string
		params    map[string]string
		baseURL   string
		apiKey    string
		wantParts []string // Substrings that should be in the URL
	}{
		{
			name:    "basic URL with no extra params",
			cmd:     "get_history",
			params:  nil,
			baseURL: "http://localhost:8181",
			apiKey:  "abc123",
			wantParts: []string{
				"http://localhost:8181/api/v2?",
				"apikey=abc123",
				"cmd=get_history",
			},
		},
		{
			name:    "URL with additional params",
			cmd:     "get_home_stats",
			params:  map[string]string{"time_range": "30", "stats_type": "plays"},
			baseURL: "http://192.168.1.100:8181",
			apiKey:  "xyz789",
			wantParts: []string{
				"http://192.168.1.100:8181/api/v2?",
				"apikey=xyz789",
				"cmd=get_home_stats",
				"time_range=30",
				"stats_type=plays",
			},
		},
		{
			name:    "URL with special characters in params",
			cmd:     "search",
			params:  map[string]string{"query": "test query"},
			baseURL: "http://localhost:8181",
			apiKey:  "key123",
			wantParts: []string{
				"http://localhost:8181/api/v2?",
				"query=test+query", // URL encoded
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newAPIRequest(tt.cmd)
			for k, v := range tt.params {
				req.addParam(k, v)
			}

			url := req.buildURL(tt.baseURL, tt.apiKey)

			for _, part := range tt.wantParts {
				if !strings.Contains(url, part) {
					t.Errorf("URL missing expected part %q, got %s", part, url)
				}
			}
		})
	}
}

// TestAPIRequestChaining tests fluent interface chaining
func TestAPIRequestChaining(t *testing.T) {
	req := newAPIRequest("get_plays_by_date").
		addParam("y_axis", "duration").
		addIntParam("time_range", 30).
		addIntParam("user_id", 1).
		addIntParamZero("grouping", 0)

	// Verify all params are set
	if req.params["y_axis"] != "duration" {
		t.Error("y_axis not set correctly through chaining")
	}
	if req.params["time_range"] != "30" {
		t.Error("time_range not set correctly through chaining")
	}
	if req.params["user_id"] != "1" {
		t.Error("user_id not set correctly through chaining")
	}
	if req.params["grouping"] != "0" {
		t.Error("grouping not set correctly through chaining")
	}
}

// TestExecuteRequest tests HTTP request execution
func TestExecuteRequest(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		body          string
		expectError   bool
		expectBodyStr string
	}{
		{
			name:          "successful response",
			statusCode:    http.StatusOK,
			body:          `{"result": "success"}`,
			expectError:   false,
			expectBodyStr: `{"result": "success"}`,
		},
		{
			name:        "non-200 status",
			statusCode:  http.StatusInternalServerError,
			body:        `{"error": "internal error"}`,
			expectError: true,
		},
		{
			name:        "not found",
			statusCode:  http.StatusNotFound,
			body:        "Not Found",
			expectError: true,
		},
		{
			name:        "forbidden",
			statusCode:  http.StatusForbidden,
			body:        "Forbidden",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := &http.Client{}
			body, err := executeRequest(context.Background(), client, server.URL)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if body != nil {
					defer body.Close()
					content, _ := io.ReadAll(body)
					if string(content) != tt.expectBodyStr {
						t.Errorf("body mismatch: expected %q, got %q", tt.expectBodyStr, string(content))
					}
				}
			}
		})
	}
}

// TestExecuteRequestNetworkError tests network error handling
func TestExecuteRequestNetworkError(t *testing.T) {
	client := &http.Client{}
	// Invalid URL should cause network error
	_, err := executeRequest(context.Background(), client, "http://invalid.invalid.invalid:99999")

	if err == nil {
		t.Error("expected network error but got nil")
	}
}

// TestDecodeResponse tests JSON response decoding
func TestDecodeResponse(t *testing.T) {
	type testResponse struct {
		Result  string  `json:"result"`
		Message *string `json:"message,omitempty"`
		Data    struct {
			Value int `json:"value"`
		} `json:"data"`
	}

	tests := []struct {
		name        string
		body        string
		getResult   func(*testResponse) string
		getMessage  func(*testResponse) *string
		expectError bool
		expectValue int
	}{
		{
			name: "successful decode",
			body: `{"result": "success", "data": {"value": 42}}`,
			getResult: func(r *testResponse) string {
				return r.Result
			},
			getMessage: func(r *testResponse) *string {
				return r.Message
			},
			expectError: false,
			expectValue: 42,
		},
		{
			name: "failed result with message",
			body: `{"result": "error", "message": "something went wrong"}`,
			getResult: func(r *testResponse) string {
				return r.Result
			},
			getMessage: func(r *testResponse) *string {
				return r.Message
			},
			expectError: true,
		},
		{
			name: "failed result without message",
			body: `{"result": "error"}`,
			getResult: func(r *testResponse) string {
				return r.Result
			},
			getMessage: func(r *testResponse) *string {
				return r.Message
			},
			expectError: true,
		},
		{
			name: "invalid JSON",
			body: `not valid json`,
			getResult: func(r *testResponse) string {
				return r.Result
			},
			getMessage: func(r *testResponse) *string {
				return r.Message
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := io.NopCloser(strings.NewReader(tt.body))
			var result testResponse

			err := decodeResponse(body, &result, tt.getResult, tt.getMessage)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result.Data.Value != tt.expectValue {
					t.Errorf("expected value %d, got %d", tt.expectValue, result.Data.Value)
				}
			}
		})
	}
}

// TestAddTimeRangeParams tests the common parameter builder
func TestAddTimeRangeParams(t *testing.T) {
	tests := []struct {
		name      string
		timeRange int
		yAxis     string
		userID    int
		grouping  int
		expected  map[string]string
	}{
		{
			name:      "all params set",
			timeRange: 30,
			yAxis:     "duration",
			userID:    1,
			grouping:  0,
			expected: map[string]string{
				"time_range": "30",
				"y_axis":     "duration",
				"user_id":    "1",
				"grouping":   "0",
			},
		},
		{
			name:      "only time_range and y_axis",
			timeRange: 7,
			yAxis:     "plays",
			userID:    0,
			grouping:  -1,
			expected: map[string]string{
				"time_range": "7",
				"y_axis":     "plays",
			},
		},
		{
			name:      "zero values not set for time_range and user_id",
			timeRange: 0,
			yAxis:     "",
			userID:    0,
			grouping:  1,
			expected: map[string]string{
				"grouping": "1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newAPIRequest("test_cmd")
			addTimeRangeParams(req, tt.timeRange, tt.yAxis, tt.userID, tt.grouping)

			for k, v := range tt.expected {
				if req.params[k] != v {
					t.Errorf("param %s: expected %q, got %q", k, v, req.params[k])
				}
			}

			// Also check no extra params are set
			for k := range req.params {
				if _, exists := tt.expected[k]; !exists {
					t.Errorf("unexpected param %s set to %q", k, req.params[k])
				}
			}
		})
	}
}

// TestExecuteAPIRequest tests the generic API request executor
func TestExecuteAPIRequest(t *testing.T) {
	type testAPIResponse struct {
		Response struct {
			Result  string  `json:"result"`
			Message *string `json:"message,omitempty"`
			Data    struct {
				Value int `json:"value"`
			} `json:"data"`
		} `json:"response"`
	}

	tests := []struct {
		name          string
		serverHandler func(w http.ResponseWriter, r *http.Request)
		expectError   bool
		expectValue   int
	}{
		{
			name: "successful request",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				// Verify request parameters
				if r.URL.Query().Get("apikey") == "" {
					t.Error("apikey not set")
				}
				if r.URL.Query().Get("cmd") == "" {
					t.Error("cmd not set")
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(testAPIResponse{
					Response: struct {
						Result  string  `json:"result"`
						Message *string `json:"message,omitempty"`
						Data    struct {
							Value int `json:"value"`
						} `json:"data"`
					}{
						Result: "success",
						Data: struct {
							Value int `json:"value"`
						}{
							Value: 100,
						},
					},
				})
			},
			expectError: false,
			expectValue: 100,
		},
		{
			name: "server error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
			},
			expectError: true,
		},
		{
			name: "API error response",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				msg := "invalid parameter"
				json.NewEncoder(w).Encode(testAPIResponse{
					Response: struct {
						Result  string  `json:"result"`
						Message *string `json:"message,omitempty"`
						Data    struct {
							Value int `json:"value"`
						} `json:"data"`
					}{
						Result:  "error",
						Message: &msg,
					},
				})
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverHandler))
			defer server.Close()

			client := &TautulliClient{
				baseURL: server.URL,
				apiKey:  "test-api-key",
				client:  &http.Client{},
			}

			req := newAPIRequest("test_cmd").
				addParam("test_param", "test_value")

			result, err := executeAPIRequest[testAPIResponse](
				context.Background(),
				client,
				req,
				func(r *testAPIResponse) string { return r.Response.Result },
				func(r *testAPIResponse) *string { return r.Response.Message },
			)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result.Response.Data.Value != tt.expectValue {
					t.Errorf("expected value %d, got %d", tt.expectValue, result.Response.Data.Value)
				}
			}
		})
	}
}

// TestReadBodyForErrorComprehensive tests the error body reader comprehensively
func TestReadBodyForErrorComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		reader   io.Reader
		expected string
	}{
		{
			name:     "valid body",
			reader:   strings.NewReader("error message"),
			expected: "error message",
		},
		{
			name:     "empty body",
			reader:   strings.NewReader(""),
			expected: "",
		},
		{
			name:     "error reader",
			reader:   &errorReader{},
			expected: "(failed to read response body)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := readBodyForError(tt.reader)
			if string(result) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

// errorReader is a reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, context.DeadlineExceeded
}
