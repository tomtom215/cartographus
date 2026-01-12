// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// TestReadBodyForError tests the utility function that reads response body for error reporting
func TestReadBodyForError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    io.Reader
		expected string
	}{
		{
			name:     "normal body content",
			input:    strings.NewReader("error message body"),
			expected: "error message body",
		},
		{
			name:     "empty body",
			input:    strings.NewReader(""),
			expected: "",
		},
		{
			name:     "JSON error response",
			input:    strings.NewReader(`{"error": "something went wrong"}`),
			expected: `{"error": "something went wrong"}`,
		},
		{
			name:     "large body content",
			input:    strings.NewReader(strings.Repeat("x", 10000)),
			expected: strings.Repeat("x", 10000),
		},
		{
			name:     "body with special characters",
			input:    strings.NewReader("Error: <html>&amp;special</html>"),
			expected: "Error: <html>&amp;special</html>",
		},
		{
			name:     "failing reader",
			input:    &failingReader{},
			expected: "(failed to read response body)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := readBodyForError(tt.input)
			if string(result) != tt.expected {
				t.Errorf("readBodyForError() = %q, want %q", string(result), tt.expected)
			}
		})
	}
}

// failingReader is a reader that always fails
type failingReader struct{}

func (f *failingReader) Read(_ []byte) (int, error) {
	return 0, errors.New("simulated read failure")
}

// TestDoRequestWithRateLimit tests the rate limiting functionality
func TestDoRequestWithRateLimit(t *testing.T) {
	t.Run("successful request on first try", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}))
		defer server.Close()

		cfg := &config.TautulliConfig{
			URL:    server.URL,
			APIKey: "test-key",
		}
		client := NewTautulliClient(cfg)

		resp, err := client.doRequestWithRateLimit(context.Background(), server.URL+"/test")
		if err != nil {
			t.Fatalf("doRequestWithRateLimit() error = %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	})

	t.Run("rate limit with retry success", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount < 3 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success after retry"))
		}))
		defer server.Close()

		cfg := &config.TautulliConfig{
			URL:    server.URL,
			APIKey: "test-key",
		}
		client := NewTautulliClient(cfg)
		// Use very short retry delay for testing
		client.retryBaseDelay = 1 * time.Millisecond

		resp, err := client.doRequestWithRateLimit(context.Background(), server.URL+"/test")
		if err != nil {
			t.Fatalf("doRequestWithRateLimit() error = %v", err)
		}
		defer resp.Body.Close()

		if attemptCount != 3 {
			t.Errorf("attempt count = %d, want 3", attemptCount)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	})

	t.Run("rate limit max retries exceeded", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer server.Close()

		cfg := &config.TautulliConfig{
			URL:    server.URL,
			APIKey: "test-key",
		}
		client := NewTautulliClient(cfg)
		client.retryBaseDelay = 1 * time.Millisecond
		client.maxRetries = 3

		resp, err := client.doRequestWithRateLimit(context.Background(), server.URL+"/test")
		if resp != nil {
			resp.Body.Close()
		}
		if err == nil {
			t.Fatal("Expected error after max retries exceeded")
		}
		if !strings.Contains(err.Error(), "rate limit exceeded") {
			t.Errorf("Error should mention rate limit, got: %v", err)
		}
		// Should have tried maxRetries + 1 times (initial + retries)
		if attemptCount != 4 {
			t.Errorf("attempt count = %d, want 4", attemptCount)
		}
	})

	t.Run("rate limit with Retry-After header", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount < 2 {
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		cfg := &config.TautulliConfig{
			URL:    server.URL,
			APIKey: "test-key",
		}
		client := NewTautulliClient(cfg)
		// Note: With Retry-After header, the delay should be 1s as specified in header
		// For testing, we accept this might take a bit longer

		resp, err := client.doRequestWithRateLimit(context.Background(), server.URL+"/test")
		if err != nil {
			t.Fatalf("doRequestWithRateLimit() error = %v", err)
		}
		defer resp.Body.Close()

		if attemptCount != 2 {
			t.Errorf("attempt count = %d, want 2", attemptCount)
		}
	})

	t.Run("non-429 error responses pass through", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		cfg := &config.TautulliConfig{
			URL:    server.URL,
			APIKey: "test-key",
		}
		client := NewTautulliClient(cfg)

		resp, err := client.doRequestWithRateLimit(context.Background(), server.URL+"/test")
		if err != nil {
			t.Fatalf("doRequestWithRateLimit() error = %v", err)
		}
		defer resp.Body.Close()

		// Non-429 errors should pass through without retry
		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
		}
	})

	t.Run("network failure", func(t *testing.T) {
		cfg := &config.TautulliConfig{
			URL:    "http://localhost:9999",
			APIKey: "test-key",
		}
		client := NewTautulliClient(cfg)

		resp, err := client.doRequestWithRateLimit(context.Background(), "http://localhost:9999/nonexistent")
		if resp != nil {
			resp.Body.Close()
		}
		if err == nil {
			t.Fatal("Expected error for network failure")
		}
		if !strings.Contains(err.Error(), "HTTP request failed") {
			t.Errorf("Error should mention HTTP request failed, got: %v", err)
		}
	})
}

// TestTautulliClientInitialization tests client initialization edge cases
func TestTautulliClientInitialization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cfg            *config.TautulliConfig
		expectedURL    string
		expectedAPIKey string
	}{
		{
			name: "standard config",
			cfg: &config.TautulliConfig{
				URL:    "http://localhost:8181",
				APIKey: "abcdef123456",
			},
			expectedURL:    "http://localhost:8181",
			expectedAPIKey: "abcdef123456",
		},
		{
			name: "HTTPS URL",
			cfg: &config.TautulliConfig{
				URL:    "https://tautulli.example.com",
				APIKey: "secure-key-789",
			},
			expectedURL:    "https://tautulli.example.com",
			expectedAPIKey: "secure-key-789",
		},
		{
			name: "URL with trailing slash",
			cfg: &config.TautulliConfig{
				URL:    "http://localhost:8181/",
				APIKey: "test-key",
			},
			expectedURL:    "http://localhost:8181/",
			expectedAPIKey: "test-key",
		},
		{
			name: "empty API key",
			cfg: &config.TautulliConfig{
				URL:    "http://localhost:8181",
				APIKey: "",
			},
			expectedURL:    "http://localhost:8181",
			expectedAPIKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := NewTautulliClient(tt.cfg)

			if client == nil {
				t.Fatal("NewTautulliClient returned nil")
			}
			if client.baseURL != tt.expectedURL {
				t.Errorf("baseURL = %q, want %q", client.baseURL, tt.expectedURL)
			}
			if client.apiKey != tt.expectedAPIKey {
				t.Errorf("apiKey = %q, want %q", client.apiKey, tt.expectedAPIKey)
			}
			if client.client == nil {
				t.Error("HTTP client should not be nil")
			}
			if client.maxRetries != 5 {
				t.Errorf("maxRetries = %d, want 5", client.maxRetries)
			}
			if client.retryBaseDelay != 1*time.Second {
				t.Errorf("retryBaseDelay = %v, want 1s", client.retryBaseDelay)
			}
		})
	}
}

// TestTautulliAnalyticsEndpoints tests various analytics endpoint methods
func TestTautulliAnalyticsEndpoints(t *testing.T) {
	// Test successful GetHomeStats
	t.Run("GetHomeStats success", func(t *testing.T) {
		server := createMockServer(t, "get_home_stats", tautulli.TautulliHomeStats{
			Response: tautulli.TautulliHomeStatsResponse{Result: "success", Data: []tautulli.TautulliHomeStatRow{}},
		})
		defer server.Close()

		client := NewTautulliClient(&config.TautulliConfig{URL: server.URL, APIKey: "test-key"})
		stats, err := client.GetHomeStats(context.Background(), 30, "plays", 10)
		clientAssertNoError(t, err, "GetHomeStats")
		clientAssertNotNil(t, stats, "stats")
	})

	// Test GetHomeStats error responses
	t.Run("GetHomeStats error response", func(t *testing.T) {
		server := createMockServer(t, "get_home_stats", tautulli.TautulliHomeStats{
			Response: tautulli.TautulliHomeStatsResponse{Result: "error", Message: stringPtr("Database error")},
		})
		defer server.Close()

		client := NewTautulliClient(&config.TautulliConfig{URL: server.URL, APIKey: "test-key"})
		_, err := client.GetHomeStats(context.Background(), 30, "plays", 10)
		clientAssertErrorContains(t, err, "Database error")
	})

	t.Run("GetHomeStats HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("service unavailable"))
		}))
		defer server.Close()

		client := NewTautulliClient(&config.TautulliConfig{URL: server.URL, APIKey: "test-key"})
		_, err := client.GetHomeStats(context.Background(), 30, "plays", 10)
		clientAssertErrorContains(t, err, "503")
	})

	// Table-driven tests for successful analytics endpoints
	successTests := []struct {
		name    string
		cmd     string
		runTest func(*TautulliClient) error
	}{
		{
			name: "GetPlaysByDate",
			cmd:  "get_plays_by_date",
			runTest: func(c *TautulliClient) error {
				_, err := c.GetPlaysByDate(context.Background(), 30, "plays", 0, 1)
				return err
			},
		},
		{
			name: "GetPlaysByDayOfWeek",
			cmd:  "get_plays_by_dayofweek",
			runTest: func(c *TautulliClient) error {
				_, err := c.GetPlaysByDayOfWeek(context.Background(), 30, "plays", 0, 1)
				return err
			},
		},
		{
			name: "GetPlaysByHourOfDay",
			cmd:  "get_plays_by_hourofday",
			runTest: func(c *TautulliClient) error {
				_, err := c.GetPlaysByHourOfDay(context.Background(), 30, "plays", 0, 1)
				return err
			},
		},
		{
			name: "GetPlaysByStreamType",
			cmd:  "get_plays_by_stream_type",
			runTest: func(c *TautulliClient) error {
				_, err := c.GetPlaysByStreamType(context.Background(), 30, "plays", 0, 1)
				return err
			},
		},
		{
			name: "GetConcurrentStreamsByStreamType",
			cmd:  "get_concurrent_streams_by_stream_type",
			runTest: func(c *TautulliClient) error {
				_, err := c.GetConcurrentStreamsByStreamType(context.Background(), 30, 0)
				return err
			},
		},
		{
			name: "GetItemWatchTimeStats",
			cmd:  "get_item_watch_time_stats",
			runTest: func(c *TautulliClient) error {
				_, err := c.GetItemWatchTimeStats(context.Background(), "12345", 1, "30")
				return err
			},
		},
	}

	for _, tt := range successTests {
		t.Run(tt.name+" success", func(t *testing.T) {
			server := createGenericSuccessServer(tt.cmd)
			defer server.Close()

			client := NewTautulliClient(&config.TautulliConfig{URL: server.URL, APIKey: "test-key"})
			if err := tt.runTest(client); err != nil {
				t.Fatalf("%s() error = %v", tt.name, err)
			}
		})
	}
}

// TestTautulliAnalyticsErrors tests error handling for analytics endpoints
func TestTautulliAnalyticsErrors(t *testing.T) {
	// Table-driven tests for API errors
	apiErrorTests := []struct {
		name          string
		errorMessage  string
		expectedInErr string
		runTest       func(*TautulliClient) error
	}{
		{
			name:          "GetPlaysByDate API error",
			errorMessage:  "Invalid time range",
			expectedInErr: "Invalid time range",
			runTest: func(c *TautulliClient) error {
				_, err := c.GetPlaysByDate(context.Background(), -1, "plays", 0, 1)
				return err
			},
		},
		{
			name:          "GetConcurrentStreamsByStreamType API error",
			errorMessage:  "Permission denied",
			expectedInErr: "Permission denied",
			runTest: func(c *TautulliClient) error {
				_, err := c.GetConcurrentStreamsByStreamType(context.Background(), 30, 0)
				return err
			},
		},
	}

	for _, tt := range apiErrorTests {
		t.Run(tt.name, func(t *testing.T) {
			server := createErrorServer(tt.errorMessage)
			defer server.Close()

			client := NewTautulliClient(&config.TautulliConfig{URL: server.URL, APIKey: "test-key"})
			err := tt.runTest(client)
			clientAssertErrorContains(t, err, tt.expectedInErr)
		})
	}

	// Test JSON decode error
	t.Run("GetPlaysByDayOfWeek JSON decode error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{invalid json`))
		}))
		defer server.Close()

		client := NewTautulliClient(&config.TautulliConfig{URL: server.URL, APIKey: "test-key"})
		_, err := client.GetPlaysByDayOfWeek(context.Background(), 30, "plays", 0, 1)
		clientAssertErrorContains(t, err, "failed to decode")
	})

	// Test network error
	t.Run("GetPlaysByHourOfDay network error", func(t *testing.T) {
		client := NewTautulliClient(&config.TautulliConfig{URL: "http://localhost:9999", APIKey: "test-key"})
		_, err := client.GetPlaysByHourOfDay(context.Background(), 30, "plays", 0, 1)
		if err == nil {
			t.Fatal("Expected error for network failure")
		}
	})

	// Test error with no message
	t.Run("GetPlaysByStreamType with no message", func(t *testing.T) {
		server := createErrorServer("")
		defer server.Close()

		client := NewTautulliClient(&config.TautulliConfig{URL: server.URL, APIKey: "test-key"})
		_, err := client.GetPlaysByStreamType(context.Background(), 30, "plays", 0, 1)
		clientAssertErrorContains(t, err, "unknown error")
	})

	// Test HTTP 500 error
	t.Run("GetItemWatchTimeStats HTTP 500 error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal server error"))
		}))
		defer server.Close()

		client := NewTautulliClient(&config.TautulliConfig{URL: server.URL, APIKey: "test-key"})
		_, err := client.GetItemWatchTimeStats(context.Background(), "12345", 1, "30")
		clientAssertErrorContains(t, err, "500")
	})
}

// TestReadBodyForErrorIntegration tests readBodyForError with real http.Response bodies
func TestReadBodyForErrorIntegration(t *testing.T) {
	t.Parallel()

	t.Run("HTTP response body", func(t *testing.T) {
		body := io.NopCloser(bytes.NewBufferString("API rate limit exceeded"))
		result := readBodyForError(body)
		if string(result) != "API rate limit exceeded" {
			t.Errorf("readBodyForError() = %q, want %q", string(result), "API rate limit exceeded")
		}
	})

	t.Run("empty HTTP response body", func(t *testing.T) {
		body := io.NopCloser(bytes.NewBufferString(""))
		result := readBodyForError(body)
		if string(result) != "" {
			t.Errorf("readBodyForError() = %q, want empty string", string(result))
		}
	})

	t.Run("multiline error body", func(t *testing.T) {
		multiline := "Error occurred:\n- Invalid parameter\n- Missing API key"
		body := io.NopCloser(bytes.NewBufferString(multiline))
		result := readBodyForError(body)
		if string(result) != multiline {
			t.Errorf("readBodyForError() = %q, want %q", string(result), multiline)
		}
	})

	t.Run("large body is truncated at 64KB", func(t *testing.T) {
		// Create a body larger than maxErrorBodySize (64KB)
		largeBody := strings.Repeat("x", 100000) // 100KB
		body := io.NopCloser(bytes.NewBufferString(largeBody))
		result := readBodyForError(body)

		// Should be truncated to 64KB + truncation message
		expectedLen := maxErrorBodySize + len("\n... (truncated)")
		if len(result) != expectedLen {
			t.Errorf("readBodyForError() length = %d, want %d (64KB + truncation indicator)", len(result), expectedLen)
		}

		// Should end with truncation indicator
		if !bytes.HasSuffix(result, []byte("\n... (truncated)")) {
			t.Errorf("readBodyForError() should end with truncation indicator")
		}
	})

	t.Run("body exactly at 64KB is truncated", func(t *testing.T) {
		// Create a body exactly at the limit
		exactBody := strings.Repeat("x", maxErrorBodySize)
		body := io.NopCloser(bytes.NewBufferString(exactBody))
		result := readBodyForError(body)

		// Should be truncated (we hit the limit exactly)
		if !bytes.HasSuffix(result, []byte("\n... (truncated)")) {
			t.Errorf("readBodyForError() should indicate truncation when hitting exact limit")
		}
	})

	t.Run("body under 64KB is not truncated", func(t *testing.T) {
		// Create a body under the limit
		underBody := strings.Repeat("x", maxErrorBodySize-1)
		body := io.NopCloser(bytes.NewBufferString(underBody))
		result := readBodyForError(body)

		// Should not be truncated
		if bytes.HasSuffix(result, []byte("\n... (truncated)")) {
			t.Errorf("readBodyForError() should not truncate bodies under 64KB")
		}
		if len(result) != maxErrorBodySize-1 {
			t.Errorf("readBodyForError() length = %d, want %d", len(result), maxErrorBodySize-1)
		}
	})
}

// Benchmark for readBodyForError
func BenchmarkReadBodyForError(b *testing.B) {
	smallBodyStr := "error message"
	mediumBodyStr := strings.Repeat("x", 10000) // 10KB
	largeBodyStr := strings.Repeat("x", 100000) // 100KB (exceeds 64KB limit)
	smallBody := strings.NewReader(smallBodyStr)
	mediumBody := strings.NewReader(mediumBodyStr)
	largeBody := strings.NewReader(largeBodyStr)

	b.Run("small_body", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			smallBody.Reset(smallBodyStr) // Reset reader position
			readBodyForError(smallBody)
		}
	})

	b.Run("medium_body", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			mediumBody.Reset(mediumBodyStr)
			readBodyForError(mediumBody)
		}
	})

	b.Run("large_body_truncated", func(b *testing.B) {
		// Tests truncation behavior with body > 64KB
		for i := 0; i < b.N; i++ {
			largeBody.Reset(largeBodyStr)
			readBodyForError(largeBody)
		}
	})
}

// Benchmark for doRequestWithRateLimit
func BenchmarkDoRequestWithRateLimit(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.TautulliConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	client := NewTautulliClient(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.doRequestWithRateLimit(context.Background(), server.URL+"/test")
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// Test helper functions for reducing complexity

// createMockServer creates a test server that validates cmd and returns mock data
func createMockServer(t *testing.T, expectedCmd string, mockData interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cmd := r.URL.Query().Get("cmd"); cmd != expectedCmd {
			t.Errorf("Expected cmd=%s, got %s", expectedCmd, cmd)
		}
		json.NewEncoder(w).Encode(mockData)
	}))
}

// createGenericSuccessServer creates a test server that returns a generic success response
func createGenericSuccessServer(cmd string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Determine the appropriate data type based on the command
		// Most analytics endpoints return objects with categories/series structure
		// but some (like get_item_watch_time_stats) return arrays
		var data interface{}
		switch cmd {
		case "get_item_watch_time_stats":
			// This endpoint returns an array of time period stats
			data = []interface{}{}
		default:
			// Most Tautulli analytics endpoints return objects with nested structure
			data = map[string]interface{}{}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"response": map[string]interface{}{
				"result": "success",
				"data":   data,
			},
		})
	}))
}

// clientAssertNoError is a test helper that fails if err is not nil
func clientAssertNoError(t *testing.T, err error, operation string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s() error = %v", operation, err)
	}
}

// clientAssertNotNil is a test helper that fails if value is nil
func clientAssertNotNil(t *testing.T, value interface{}, name string) {
	t.Helper()
	if value == nil {
		t.Errorf("Expected non-nil %s", name)
	}
}

// clientAssertErrorContains is a test helper that fails if error doesn't contain expected string
func clientAssertErrorContains(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("Error should contain %q, got: %v", expected, err)
	}
}

// createErrorServer creates a test server that returns an error response
func createErrorServer(errorMessage string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg *string
		if errorMessage != "" {
			msg = &errorMessage
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"response": map[string]interface{}{
				"result":  "error",
				"message": msg,
			},
		})
	}))
}
