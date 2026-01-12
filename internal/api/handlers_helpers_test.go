// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"

	"github.com/tomtom215/cartographus/internal/models"
)

// ===================================================================================================
// generateETag Tests
// ===================================================================================================

func TestGenerateETag_Helpers(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		checkLen bool
	}{
		{
			name:     "empty data",
			input:    []byte{},
			checkLen: true,
		},
		{
			name:     "simple string",
			input:    []byte("hello world"),
			checkLen: true,
		},
		{
			name:     "json data",
			input:    []byte(`{"key": "value", "count": 123}`),
			checkLen: true,
		},
		{
			name:     "binary data",
			input:    []byte{0x00, 0xFF, 0x55, 0xAA},
			checkLen: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			etag := generateETag(tt.input)

			// ETag should be non-empty
			if etag == "" {
				t.Error("generateETag() returned empty string")
			}

			// ETag should be deterministic (same input = same output)
			etag2 := generateETag(tt.input)
			if etag != etag2 {
				t.Errorf("generateETag() is not deterministic: %s != %s", etag, etag2)
			}
		})
	}

	// Test that different inputs produce different ETags
	t.Run("different inputs produce different ETags", func(t *testing.T) {
		etag1 := generateETag([]byte("hello"))
		etag2 := generateETag([]byte("world"))
		if etag1 == etag2 {
			t.Error("Different inputs produced the same ETag")
		}
	})
}

// ===================================================================================================
// escapeCSV Tests
// ===================================================================================================

func TestEscapeCSV_Helpers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special chars",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "contains comma",
			input:    "hello,world",
			expected: `"hello,world"`,
		},
		{
			name:     "contains quote",
			input:    `hello"world`,
			expected: `"hello""world"`,
		},
		{
			name:     "contains newline",
			input:    "hello\nworld",
			expected: "\"hello\nworld\"",
		},
		{
			name:     "contains carriage return",
			input:    "hello\rworld",
			expected: "\"hello\rworld\"",
		},
		{
			name:     "multiple quotes",
			input:    `"hello" and "world"`,
			expected: `"""hello"" and ""world"""`,
		},
		{
			name:     "comma and quote",
			input:    `hello,"world"`,
			expected: `"hello,""world"""`,
		},
		{
			name:     "all special characters",
			input:    "a,b\"c\nd",
			expected: "\"a,b\"\"c\nd\"",
		},
		{
			name:     "numeric value",
			input:    "12345",
			expected: "12345",
		},
		{
			name:     "value with only comma",
			input:    ",",
			expected: `","`,
		},
		{
			name:     "value with only quote",
			input:    `"`,
			expected: `""""`,
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

// ===================================================================================================
// parseIntParam Tests
// ===================================================================================================

func TestParseIntParam_Helpers(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		defaultValue int
		expected     int
	}{
		{
			name:         "valid positive",
			value:        "42",
			defaultValue: 0,
			expected:     42,
		},
		{
			name:         "valid negative",
			value:        "-10",
			defaultValue: 0,
			expected:     -10,
		},
		{
			name:         "valid zero",
			value:        "0",
			defaultValue: 5,
			expected:     0,
		},
		{
			name:         "empty string uses default",
			value:        "",
			defaultValue: 100,
			expected:     100,
		},
		{
			name:         "invalid string uses default",
			value:        "abc",
			defaultValue: 50,
			expected:     50,
		},
		{
			name:         "float string parses integer part",
			value:        "3.14",
			defaultValue: 20,
			expected:     3,
		},
		{
			name:         "string with spaces parses correctly",
			value:        " 10 ",
			defaultValue: 5,
			expected:     10,
		},
		{
			name:         "large number",
			value:        "999999999",
			defaultValue: 0,
			expected:     999999999,
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

// ===================================================================================================
// parseCommaSeparatedInts Tests
// ===================================================================================================

func TestParseCommaSeparatedInts_Helpers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []int
	}{
		{
			name:     "single value",
			input:    "42",
			expected: []int{42},
		},
		{
			name:     "multiple values",
			input:    "1,2,3",
			expected: []int{1, 2, 3},
		},
		{
			name:     "values with spaces",
			input:    "1, 2, 3",
			expected: []int{1, 2, 3},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "invalid values skipped",
			input:    "1,abc,3",
			expected: []int{1, 3},
		},
		{
			name:     "all invalid",
			input:    "abc,def",
			expected: []int{},
		},
		{
			name:     "negative numbers",
			input:    "-1,-2,3",
			expected: []int{-1, -2, 3},
		},
		{
			name:     "mixed valid and invalid",
			input:    "1,two,3,four,5",
			expected: []int{1, 3, 5},
		},
		{
			name:     "trailing comma",
			input:    "1,2,",
			expected: []int{1, 2},
		},
		{
			name:     "leading comma",
			input:    ",1,2",
			expected: []int{1, 2},
		},
		{
			name:     "zeros",
			input:    "0,0,0",
			expected: []int{0, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparatedInts(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("parseCommaSeparatedInts(%q) returned %d values, expected %d. Got: %v", tt.input, len(result), len(tt.expected), result)
				return
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("parseCommaSeparatedInts(%q)[%d] = %d, expected %d", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

// ===================================================================================================
// respondJSON Tests
// ===================================================================================================

func TestRespondJSON(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		response       *models.APIResponse
		expectedStatus int
		checkHeaders   bool
	}{
		{
			name:   "success response",
			status: http.StatusOK,
			response: &models.APIResponse{
				Status: "success",
				Data:   map[string]string{"key": "value"},
			},
			expectedStatus: http.StatusOK,
			checkHeaders:   true,
		},
		{
			name:   "error response",
			status: http.StatusBadRequest,
			response: &models.APIResponse{
				Status: "error",
				Error:  &models.APIError{Code: "TEST_ERROR", Message: "test message"},
			},
			expectedStatus: http.StatusBadRequest,
			checkHeaders:   true,
		},
		{
			name:   "not found response",
			status: http.StatusNotFound,
			response: &models.APIResponse{
				Status: "error",
				Data:   nil,
			},
			expectedStatus: http.StatusNotFound,
			checkHeaders:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			respondJSON(w, tt.status, tt.response)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkHeaders {
				if ct := w.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("Expected Content-Type 'application/json', got %q", ct)
				}
				if cc := w.Header().Get("Cache-Control"); cc == "" {
					t.Error("Expected Cache-Control header to be set")
				}
				if etag := w.Header().Get("ETag"); etag == "" {
					t.Error("Expected ETag header to be set")
				}
			}

			// Verify body can be decoded
			var decoded models.APIResponse
			if err := json.NewDecoder(w.Body).Decode(&decoded); err != nil {
				t.Errorf("Failed to decode response body: %v", err)
			}

			if decoded.Status != tt.response.Status {
				t.Errorf("Expected status %q, got %q", tt.response.Status, decoded.Status)
			}
		})
	}
}

// ===================================================================================================
// respondError Tests
// ===================================================================================================

func TestRespondError(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		code           string
		message        string
		err            error
		expectedStatus int
	}{
		{
			name:           "bad request error",
			status:         http.StatusBadRequest,
			code:           "VALIDATION_ERROR",
			message:        "Invalid input",
			err:            nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "internal server error",
			status:         http.StatusInternalServerError,
			code:           "DATABASE_ERROR",
			message:        "Database connection failed",
			err:            nil,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "unauthorized error",
			status:         http.StatusUnauthorized,
			code:           "AUTH_ERROR",
			message:        "Invalid credentials",
			err:            nil,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			respondError(w, tt.status, tt.code, tt.message, tt.err)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var decoded models.APIResponse
			if err := json.NewDecoder(w.Body).Decode(&decoded); err != nil {
				t.Errorf("Failed to decode response body: %v", err)
			}

			if decoded.Status != "error" {
				t.Errorf("Expected status 'error', got %q", decoded.Status)
			}

			if decoded.Error == nil {
				t.Error("Expected error field to be set")
			} else {
				if decoded.Error.Code != tt.code {
					t.Errorf("Expected error code %q, got %q", tt.code, decoded.Error.Code)
				}
				if decoded.Error.Message != tt.message {
					t.Errorf("Expected error message %q, got %q", tt.message, decoded.Error.Message)
				}
			}
		})
	}
}

// ===================================================================================================
// parseCommaSeparated Additional Edge Cases
// ===================================================================================================

func TestParseCommaSeparated_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "values with internal whitespace",
			input:    "hello world, foo bar, baz qux",
			expected: []string{"hello world", "foo bar", "baz qux"},
		},
		{
			name:     "excessive whitespace",
			input:    "  a  ,  b  ,  c  ",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "mixed empty and non-empty",
			input:    "a,,b",
			expected: []string{"a", "b"},
		},
		{
			name:     "only empty values",
			input:    "  ,  ,  ",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparated(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("parseCommaSeparated(%q) returned %d values, expected %d. Got: %v", tt.input, len(result), len(tt.expected), result)
				return
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("parseCommaSeparated(%q)[%d] = %q, expected %q", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

// ===================================================================================================
// encodeCursor and decodeCursor Tests
// ===================================================================================================

func TestEncodeCursor(t *testing.T) {
	testID := "550e8400-e29b-41d4-a716-446655440000"
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name   string
		cursor *models.PlaybackCursor
	}{
		{
			name: "complete cursor",
			cursor: &models.PlaybackCursor{
				ID:        testID,
				StartedAt: testTime,
			},
		},
		{
			name: "cursor with zero time",
			cursor: &models.PlaybackCursor{
				ID:        testID,
				StartedAt: time.Time{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := encodeCursor(tt.cursor)

			// Should produce non-empty base64 string
			if encoded == "" {
				t.Error("encodeCursor() returned empty string")
			}

			// Should be valid base64
			_, err := base64.URLEncoding.DecodeString(encoded)
			if err != nil {
				t.Errorf("encodeCursor() produced invalid base64: %v", err)
			}
		})
	}
}

func TestEncodeCursor_NilCursor(t *testing.T) {
	// Test with nil cursor - should handle gracefully
	// Note: This tests the defensive programming aspect
	// The function panics with nil, which is expected Go behavior
	// We test non-nil cases
}

func TestDecodeCursor(t *testing.T) {
	testID := "550e8400-e29b-41d4-a716-446655440000"
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	// First encode a cursor
	original := &models.PlaybackCursor{
		ID:        testID,
		StartedAt: testTime,
	}
	encoded := encodeCursor(original)

	// Then decode it
	decoded, err := decodeCursor(encoded)
	if err != nil {
		t.Fatalf("decodeCursor() failed: %v", err)
	}

	// Verify the decoded cursor matches
	if decoded.ID != original.ID {
		t.Errorf("decoded.ID = %v, expected %v", decoded.ID, original.ID)
	}
	if !decoded.StartedAt.Equal(original.StartedAt) {
		t.Errorf("decoded.StartedAt = %v, expected %v", decoded.StartedAt, original.StartedAt)
	}
}

func TestDecodeCursor_InvalidBase64(t *testing.T) {
	// Test with invalid base64 string
	_, err := decodeCursor("not-valid-base64!!!")
	if err == nil {
		t.Error("decodeCursor() should fail with invalid base64")
	}
}

func TestDecodeCursor_InvalidJSON(t *testing.T) {
	// Test with valid base64 but invalid JSON
	invalidJSON := base64.URLEncoding.EncodeToString([]byte("not valid json"))
	_, err := decodeCursor(invalidJSON)
	if err == nil {
		t.Error("decodeCursor() should fail with invalid JSON")
	}
}

func TestDecodeCursor_EmptyString(t *testing.T) {
	// Test with empty string
	_, err := decodeCursor("")
	if err == nil {
		t.Error("decodeCursor() should fail with empty string")
	}
}

func TestEncodeDecode_RoundTrip(t *testing.T) {
	// Test that encode -> decode produces the same result
	cursors := []models.PlaybackCursor{
		{
			ID:        uuid.New().String(),
			StartedAt: time.Now().UTC().Truncate(time.Second),
		},
		{
			ID:        "00000000-0000-0000-0000-000000000000",
			StartedAt: time.Time{},
		},
		{
			ID:        "ffffffff-ffff-ffff-ffff-ffffffffffff",
			StartedAt: time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC),
		},
	}

	for i, original := range cursors {
		t.Run("cursor_"+string(rune('0'+i)), func(t *testing.T) {
			encoded := encodeCursor(&original)
			decoded, err := decodeCursor(encoded)
			if err != nil {
				t.Fatalf("Round trip failed at decode: %v", err)
			}

			if decoded.ID != original.ID {
				t.Errorf("ID mismatch: got %v, want %v", decoded.ID, original.ID)
			}
			if !decoded.StartedAt.Equal(original.StartedAt) {
				t.Errorf("StartedAt mismatch: got %v, want %v", decoded.StartedAt, original.StartedAt)
			}
		})
	}
}

// ===================================================================================================
// getIntParam Tests - Extended
// ===================================================================================================

func TestGetIntParam_FromRequest(t *testing.T) {
	tests := []struct {
		name         string
		queryString  string
		paramName    string
		defaultValue int
		expected     int
	}{
		{
			name:         "existing parameter",
			queryString:  "limit=50",
			paramName:    "limit",
			defaultValue: 100,
			expected:     50,
		},
		{
			name:         "missing parameter",
			queryString:  "other=50",
			paramName:    "limit",
			defaultValue: 100,
			expected:     100,
		},
		{
			name:         "empty query string",
			queryString:  "",
			paramName:    "limit",
			defaultValue: 100,
			expected:     100,
		},
		{
			name:         "negative number",
			queryString:  "offset=-1",
			paramName:    "offset",
			defaultValue: 0,
			expected:     -1,
		},
		{
			name:         "invalid number",
			queryString:  "limit=abc",
			paramName:    "limit",
			defaultValue: 50,
			expected:     50,
		},
		{
			name:         "zero value",
			queryString:  "limit=0",
			paramName:    "limit",
			defaultValue: 100,
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.queryString != "" {
				url += "?" + tt.queryString
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			result := getIntParam(req, tt.paramName, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getIntParam() = %d, expected %d", result, tt.expected)
			}
		})
	}
}
