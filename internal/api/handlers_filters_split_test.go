// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBuildFilter_StartDate(t *testing.T) {
	handler := &Handler{}

	tests := []struct {
		name      string
		queryStr  string
		expectNil bool
	}{
		{
			name:      "Valid RFC3339 start date",
			queryStr:  "start_date=2025-01-01T00:00:00Z",
			expectNil: false,
		},
		{
			name:      "Invalid start date format",
			queryStr:  "start_date=invalid",
			expectNil: true,
		},
		{
			name:      "No start date",
			queryStr:  "",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/trends?"+tt.queryStr, nil)
			filter := handler.buildFilter(req)

			if tt.expectNil && filter.StartDate != nil {
				t.Error("Expected StartDate to be nil")
			}
			if !tt.expectNil && filter.StartDate == nil {
				t.Error("Expected StartDate to be set")
			}
		})
	}
}

func TestBuildFilter_Days(t *testing.T) {
	handler := &Handler{}

	tests := []struct {
		name      string
		days      string
		expectNil bool
	}{
		{
			name:      "Valid days parameter",
			days:      "7",
			expectNil: false,
		},
		{
			name:      "Days at minimum",
			days:      "1",
			expectNil: false,
		},
		{
			name:      "Days at maximum",
			days:      "3650",
			expectNil: false,
		},
		{
			name:      "Days too small",
			days:      "0",
			expectNil: true,
		},
		{
			name:      "Days too large",
			days:      "3651",
			expectNil: true,
		},
		{
			name:      "Invalid days format",
			days:      "invalid",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/trends?days="+tt.days, nil)
			filter := handler.buildFilter(req)

			if tt.expectNil && filter.StartDate != nil {
				t.Errorf("Expected StartDate to be nil for days=%s", tt.days)
			}
			if !tt.expectNil && filter.StartDate == nil {
				t.Errorf("Expected StartDate to be set for days=%s", tt.days)
			}
		})
	}
}

func TestBuildFilter_EndDate(t *testing.T) {
	handler := &Handler{}

	tests := []struct {
		name      string
		queryStr  string
		expectNil bool
	}{
		{
			name:      "Valid RFC3339 end date",
			queryStr:  "end_date=2025-12-31T23:59:59Z",
			expectNil: false,
		},
		{
			name:      "Invalid end date format",
			queryStr:  "end_date=invalid",
			expectNil: true,
		},
		{
			name:      "No end date",
			queryStr:  "",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/trends?"+tt.queryStr, nil)
			filter := handler.buildFilter(req)

			if tt.expectNil && filter.EndDate != nil {
				t.Error("Expected EndDate to be nil")
			}
			if !tt.expectNil && filter.EndDate == nil {
				t.Error("Expected EndDate to be set")
			}
		})
	}
}

func TestBuildFilter_Users(t *testing.T) {
	handler := &Handler{}

	tests := []struct {
		name          string
		usersParam    string
		expectedCount int
	}{
		{
			name:          "Single user",
			usersParam:    "user1",
			expectedCount: 1,
		},
		{
			name:          "Multiple users",
			usersParam:    "user1,user2,user3",
			expectedCount: 3,
		},
		{
			name:          "No users",
			usersParam:    "",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/analytics/trends"
			if tt.usersParam != "" {
				url += "?users=" + tt.usersParam
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			filter := handler.buildFilter(req)

			if len(filter.Users) != tt.expectedCount {
				t.Errorf("Expected %d users, got %d", tt.expectedCount, len(filter.Users))
			}
		})
	}
}

func TestBuildFilter_MediaTypes(t *testing.T) {
	handler := &Handler{}

	tests := []struct {
		name          string
		mediaTypes    string
		expectedCount int
	}{
		{
			name:          "Single media type",
			mediaTypes:    "movie",
			expectedCount: 1,
		},
		{
			name:          "Multiple media types",
			mediaTypes:    "movie,episode,track",
			expectedCount: 3,
		},
		{
			name:          "No media types",
			mediaTypes:    "",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/analytics/trends"
			if tt.mediaTypes != "" {
				url += "?media_types=" + tt.mediaTypes
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			filter := handler.buildFilter(req)

			if len(filter.MediaTypes) != tt.expectedCount {
				t.Errorf("Expected %d media types, got %d", tt.expectedCount, len(filter.MediaTypes))
			}
		})
	}
}

func TestBuildFilter_CombinedFilters(t *testing.T) {
	handler := &Handler{}

	queryStr := "start_date=2025-01-01T00:00:00Z&end_date=2025-12-31T23:59:59Z&users=user1,user2&media_types=movie,episode"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/trends?"+queryStr, nil)
	filter := handler.buildFilter(req)

	if filter.StartDate == nil {
		t.Error("Expected StartDate to be set")
	}
	if filter.EndDate == nil {
		t.Error("Expected EndDate to be set")
	}
	if len(filter.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(filter.Users))
	}
	if len(filter.MediaTypes) != 2 {
		t.Errorf("Expected 2 media types, got %d", len(filter.MediaTypes))
	}
	if filter.Limit != 1000 {
		t.Errorf("Expected default limit 1000, got %d", filter.Limit)
	}
}

func TestBuildFilter_DaysOverridesStartDate(t *testing.T) {
	handler := &Handler{}

	// If both start_date and days are provided, start_date takes precedence (it's checked first)
	queryStr := "start_date=2025-01-01T00:00:00Z&days=30"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/trends?"+queryStr, nil)
	filter := handler.buildFilter(req)

	if filter.StartDate == nil {
		t.Error("Expected StartDate to be set")
	}

	// Start date should be from explicit start_date, not days calculation
	expectedDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if !filter.StartDate.Equal(expectedDate) {
		t.Errorf("Expected start date %v, got %v", expectedDate, filter.StartDate)
	}
}

func TestGetIntParam_ValidValues(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		defValue int
		expected int
	}{
		{
			name:     "Valid positive integer",
			value:    "10",
			defValue: 5,
			expected: 10,
		},
		{
			name:     "Valid zero",
			value:    "0",
			defValue: 5,
			expected: 0,
		},
		{
			name:     "Invalid value uses default",
			value:    "invalid",
			defValue: 5,
			expected: 5,
		},
		{
			name:     "Negative value",
			value:    "-10",
			defValue: 5,
			expected: -10,
		},
		{
			name:     "Empty value uses default",
			value:    "",
			defValue: 5,
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.value != "" {
				url += "?param=" + tt.value
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			result := getIntParam(req, "param", tt.defValue)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestParseCommaSeparated_ValidInputs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Single value",
			input:    "value1",
			expected: []string{"value1"},
		},
		{
			name:     "Multiple values",
			input:    "value1,value2,value3",
			expected: []string{"value1", "value2", "value3"},
		},
		{
			name:     "Values with spaces",
			input:    "value1, value2, value3",
			expected: []string{"value1", "value2", "value3"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Single comma",
			input:    ",",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparated(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d values, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected value[%d] = %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}
