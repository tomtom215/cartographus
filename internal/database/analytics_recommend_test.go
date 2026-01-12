// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"testing"
)

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single value",
			input:    "Action",
			expected: []string{"Action"},
		},
		{
			name:     "multiple values",
			input:    "Action, Comedy, Drama",
			expected: []string{"Action", "Comedy", "Drama"},
		},
		{
			name:     "with extra whitespace",
			input:    "  Action  ,  Comedy  ,  Drama  ",
			expected: []string{"Action", "Comedy", "Drama"},
		},
		{
			name:     "empty values filtered",
			input:    "Action,,Comedy,  ,Drama",
			expected: []string{"Action", "Comedy", "Drama"},
		},
		{
			name:     "single value with whitespace",
			input:    "  Action  ",
			expected: []string{"Action"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitAndTrim(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("splitAndTrim(%q) returned %d items, want %d", tt.input, len(result), len(tt.expected))
				return
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("splitAndTrim(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

// Note: Integration tests for recommendation queries would require a test database.
// These are covered by the integration test suite using testcontainers.
// See: internal/testinfra/duckdb_test.go for patterns.
