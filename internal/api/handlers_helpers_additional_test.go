// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/models"
)

// TestGenerateETag_Consistency tests that generateETag produces consistent results
func TestGenerateETag_Consistency(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		data []byte
	}{
		{
			name: "same data produces same etag",
			data: []byte("test data"),
		},
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "large data",
			data: make([]byte, 10000),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			etag1 := generateETag(tc.data)
			etag2 := generateETag(tc.data)

			if etag1 != etag2 {
				t.Errorf("generateETag produced inconsistent results: %s != %s", etag1, etag2)
			}

			if etag1 == "" {
				t.Error("generateETag produced empty string")
			}
		})
	}
}

// TestGenerateETag_Uniqueness tests that different data produces different eTags
func TestGenerateETag_Uniqueness(t *testing.T) {
	t.Parallel()

	testData := [][]byte{
		[]byte("data1"),
		[]byte("data2"),
		[]byte("data3"),
		{0x00},
		{0xFF},
	}

	eTags := make(map[string]bool)
	for i, data := range testData {
		etag := generateETag(data)
		if eTags[etag] {
			t.Errorf("Duplicate eTag found for index %d", i)
		}
		eTags[etag] = true
	}
}

// TestParseCommaSeparated_EmptyResults tests parseCommaSeparated with various empty inputs
func TestParseCommaSeparated_EmptyResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"only commas", ",,,,"},
		{"only spaces", "     "},
		{"commas and spaces", " , , , "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparated(tt.input)
			if len(result) > 0 {
				t.Errorf("Expected empty result for %q, got %v", tt.input, result)
			}
		})
	}
}

// TestParseCommaSeparated_SpecialCharacters tests parseCommaSeparated with special characters
func TestParseCommaSeparated_SpecialCharacters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "unicode characters",
			input:    "hello,世界,привет",
			expected: []string{"hello", "世界", "привет"},
		},
		{
			name:     "emojis",
			input:    "😀,😁,😂",
			expected: []string{"😀", "😁", "😂"},
		},
		{
			name:     "mixed unicode and ascii",
			input:    "test,テスト,123",
			expected: []string{"test", "テスト", "123"},
		},
		{
			name:     "newlines in values",
			input:    "hello\nworld,foo\nbar",
			expected: []string{"hello\nworld", "foo\nbar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparated(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d items, got %d", len(tt.expected), len(result))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("Item %d: expected %q, got %q", i, tt.expected[i], v)
				}
			}
		})
	}
}

// TestParseCommaSeparatedInts_EdgeCases tests parseCommaSeparatedInts with edge cases
func TestParseCommaSeparatedInts_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []int
	}{
		{
			name:     "very large numbers",
			input:    "2147483647,-2147483648,999999999",
			expected: []int{2147483647, -2147483648, 999999999},
		},
		{
			name:     "numbers with leading zeros",
			input:    "001,002,003",
			expected: []int{1, 2, 3},
		},
		{
			name:     "numbers with plus sign",
			input:    "+1,+2,+3",
			expected: []int{1, 2, 3},
		},
		{
			name:     "float numbers - only integer part parsed",
			input:    "1.5,2.9,3.1",
			expected: []int{},
		},
		{
			name:     "scientific notation - not parsed",
			input:    "1e5,2e3",
			expected: []int{},
		},
		{
			name:     "hex numbers - not parsed as hex",
			input:    "0x10,0xFF",
			expected: []int{}, // strconv.Atoi doesn't parse hex, so these are skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparatedInts(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d items, got %d. Got: %v", len(tt.expected), len(result), result)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("Item %d: expected %d, got %d", i, tt.expected[i], v)
				}
			}
		})
	}
}

// TestParseIntParam_EdgeCases tests parseIntParam with edge cases
func TestParseIntParam_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		value        string
		defaultValue int
		expected     int
	}{
		{
			name:         "max int32",
			value:        "2147483647",
			defaultValue: 0,
			expected:     2147483647,
		},
		{
			name:         "min int32",
			value:        "-2147483648",
			defaultValue: 0,
			expected:     -2147483648,
		},
		{
			name:         "overflow - use default",
			value:        "999999999999999999999",
			defaultValue: 100,
			expected:     100,
		},
		{
			name:         "just whitespace",
			value:        "   ",
			defaultValue: 50,
			expected:     50,
		},
		{
			name:         "tab character",
			value:        "\t",
			defaultValue: 75,
			expected:     75,
		},
		{
			name:         "newline character",
			value:        "\n",
			defaultValue: 80,
			expected:     80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIntParam(tt.value, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("parseIntParam(%q, %d) = %d, expected %d", tt.value, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

// TestGetIntParam_EdgeCases tests getIntParam with edge cases
func TestGetIntParam_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		queryString  string
		paramName    string
		defaultValue int
		expected     int
	}{
		{
			name:         "multiple values - first used",
			queryString:  "limit=10&limit=20",
			paramName:    "limit",
			defaultValue: 100,
			expected:     10,
		},
		{
			name:         "url encoded value",
			queryString:  "limit=%2B10", // +10
			paramName:    "limit",
			defaultValue: 0,
			expected:     10,
		},
		{
			name:         "special characters in value",
			queryString:  "limit=10abc",
			paramName:    "limit",
			defaultValue: 50,
			expected:     50,
		},
		{
			name:         "max int value",
			queryString:  "limit=2147483647",
			paramName:    "limit",
			defaultValue: 0,
			expected:     2147483647,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test?" + tt.queryString
			req := httptest.NewRequest(http.MethodGet, url, nil)
			result := getIntParam(req, tt.paramName, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getIntParam() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

// TestEscapeCSV_ComplexScenarios tests escapeCSV with complex real-world scenarios
func TestEscapeCSV_ComplexScenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "email address",
			input:    "user@example.com",
			expected: "user@example.com",
		},
		{
			name:     "url with commas",
			input:    "http://example.com?a=1,2,3",
			expected: `"http://example.com?a=1,2,3"`,
		},
		{
			name:     "json string",
			input:    `{"key":"value","count":123}`,
			expected: `"{""key"":""value"",""count"":123}"`,
		},
		{
			name:     "multi-line with quotes and commas",
			input:    "Line 1,\"quoted\"\nLine 2",
			expected: "\"Line 1,\"\"quoted\"\"\nLine 2\"",
		},
		{
			name:     "windows path",
			input:    `C:\Users\Test\file.txt`,
			expected: `C:\Users\Test\file.txt`,
		},
		{
			name:     "unix path with comma",
			input:    `/home/user/file,test.txt`,
			expected: `"/home/user/file,test.txt"`,
		},
		{
			name:     "sql injection attempt",
			input:    `' OR '1'='1`,
			expected: `' OR '1'='1`,
		},
		{
			name:     "xss attempt",
			input:    `<script>alert("xss")</script>`,
			expected: `"<script>alert(""xss"")</script>"`,
		},
		{
			name:     "unicode with special chars",
			input:    "Hello, 世界!",
			expected: `"Hello, 世界!"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeCSV(tt.input)
			if result != tt.expected {
				t.Errorf("escapeCSV(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestRespondError_WithNilError tests respondError when err parameter is nil
func TestRespondError_WithNilError(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	respondError(w, http.StatusBadRequest, "TEST_ERROR", "test message", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Should still create error response
	var response models.APIResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Error == nil {
		t.Error("Expected error in response")
	}
}

// TestRespondError_WithError tests respondError when err parameter is provided
func TestRespondError_WithError(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	testErr := errors.New("detailed error message")
	respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "database failed", testErr)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var response models.APIResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Error == nil {
		t.Fatal("Expected error in response")
	}

	if response.Error.Code != "DATABASE_ERROR" {
		t.Errorf("Expected error code DATABASE_ERROR, got %s", response.Error.Code)
	}

	if response.Error.Message != "database failed" {
		t.Errorf("Expected message 'database failed', got %s", response.Error.Message)
	}
}

// TestRespondError_VariousStatusCodes tests respondError with different status codes
func TestRespondError_VariousStatusCodes(t *testing.T) {
	t.Parallel()

	statusCodes := []int{
		http.StatusBadRequest,          // 400
		http.StatusUnauthorized,        // 401
		http.StatusForbidden,           // 403
		http.StatusNotFound,            // 404
		http.StatusMethodNotAllowed,    // 405
		http.StatusConflict,            // 409
		http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
		http.StatusServiceUnavailable,  // 503
	}

	for _, statusCode := range statusCodes {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			w := httptest.NewRecorder()
			respondError(w, statusCode, "TEST_ERROR", "test message", nil)

			if w.Code != statusCode {
				t.Errorf("Expected status %d, got %d", statusCode, w.Code)
			}

			var response models.APIResponse
			json.NewDecoder(w.Body).Decode(&response)

			if response.Status != "error" {
				t.Errorf("Expected status 'error', got '%s'", response.Status)
			}
		})
	}
}

// BenchmarkParseCommaSeparated benchmarks comma-separated parsing
func BenchmarkParseCommaSeparated(b *testing.B) {
	input := "value1,value2,value3,value4,value5,value6,value7,value8,value9,value10"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseCommaSeparated(input)
	}
}

// BenchmarkParseCommaSeparatedInts benchmarks integer parsing
func BenchmarkParseCommaSeparatedInts(b *testing.B) {
	input := "1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseCommaSeparatedInts(input)
	}
}
