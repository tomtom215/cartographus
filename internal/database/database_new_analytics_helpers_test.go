// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"testing"
)

// TestCalculatePercentage tests the percentage calculation helper
func TestCalculatePercentage(t *testing.T) {
	tests := []struct {
		name        string
		numerator   int
		denominator int
		expected    float64
	}{
		{
			name:        "simple percentage",
			numerator:   50,
			denominator: 100,
			expected:    50.0,
		},
		{
			name:        "zero numerator",
			numerator:   0,
			denominator: 100,
			expected:    0.0,
		},
		{
			name:        "zero denominator - division by zero protection",
			numerator:   50,
			denominator: 0,
			expected:    0.0,
		},
		{
			name:        "both zero",
			numerator:   0,
			denominator: 0,
			expected:    0.0,
		},
		{
			name:        "100 percent",
			numerator:   100,
			denominator: 100,
			expected:    100.0,
		},
		{
			name:        "over 100 percent",
			numerator:   150,
			denominator: 100,
			expected:    150.0,
		},
		{
			name:        "small fraction",
			numerator:   1,
			denominator: 1000,
			expected:    0.1,
		},
		{
			name:        "large numbers",
			numerator:   1000000,
			denominator: 2000000,
			expected:    50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculatePercentage(tt.numerator, tt.denominator)
			if result != tt.expected {
				t.Errorf("calculatePercentage(%d, %d) = %f, want %f",
					tt.numerator, tt.denominator, result, tt.expected)
			}
		})
	}
}

// TestBuildWhereClause tests WHERE clause construction
func TestBuildWhereClause(t *testing.T) {
	tests := []struct {
		name         string
		whereClauses []string
		expected     string
	}{
		{
			name:         "empty clauses",
			whereClauses: []string{},
			expected:     "",
		},
		{
			name:         "nil clauses",
			whereClauses: nil,
			expected:     "",
		},
		{
			name:         "single clause",
			whereClauses: []string{"status = 'active'"},
			expected:     "WHERE status = 'active'",
		},
		{
			name:         "two clauses",
			whereClauses: []string{"status = 'active'", "user_id = 1"},
			expected:     "WHERE status = 'active' AND user_id = 1",
		},
		{
			name:         "three clauses",
			whereClauses: []string{"a = 1", "b = 2", "c = 3"},
			expected:     "WHERE a = 1 AND b = 2 AND c = 3",
		},
		{
			name:         "complex conditions",
			whereClauses: []string{"date >= '2025-01-01'", "username IN ('alice', 'bob')", "media_type = 'movie'"},
			expected:     "WHERE date >= '2025-01-01' AND username IN ('alice', 'bob') AND media_type = 'movie'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildWhereClause(tt.whereClauses)
			if result != tt.expected {
				t.Errorf("buildWhereClause(%v) = %q, want %q", tt.whereClauses, result, tt.expected)
			}
		})
	}
}

// TestBuildAndWhereClause tests AND-prefixed WHERE clause construction
func TestBuildAndWhereClause(t *testing.T) {
	tests := []struct {
		name         string
		whereClauses []string
		expected     string
	}{
		{
			name:         "empty clauses",
			whereClauses: []string{},
			expected:     "",
		},
		{
			name:         "nil clauses",
			whereClauses: nil,
			expected:     "",
		},
		{
			name:         "single clause",
			whereClauses: []string{"status = 'active'"},
			expected:     "status = 'active'",
		},
		{
			name:         "two clauses",
			whereClauses: []string{"status = 'active'", "user_id = 1"},
			expected:     "status = 'active' AND user_id = 1",
		},
		{
			name:         "multiple clauses",
			whereClauses: []string{"a = 1", "b = 2", "c = 3", "d = 4"},
			expected:     "a = 1 AND b = 2 AND c = 3 AND d = 4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildAndWhereClause(tt.whereClauses)
			if result != tt.expected {
				t.Errorf("buildAndWhereClause(%v) = %q, want %q", tt.whereClauses, result, tt.expected)
			}
		})
	}
}

// TestJoinWithAnd tests the AND joining helper
func TestJoinWithAnd(t *testing.T) {
	tests := []struct {
		name     string
		clauses  []string
		expected string
	}{
		{
			name:     "empty",
			clauses:  []string{},
			expected: "",
		},
		{
			name:     "single",
			clauses:  []string{"a"},
			expected: "a",
		},
		{
			name:     "two",
			clauses:  []string{"a", "b"},
			expected: "a AND b",
		},
		{
			name:     "three",
			clauses:  []string{"x = 1", "y = 2", "z = 3"},
			expected: "x = 1 AND y = 2 AND z = 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinWithAnd(tt.clauses)
			if result != tt.expected {
				t.Errorf("joinWithAnd(%v) = %q, want %q", tt.clauses, result, tt.expected)
			}
		})
	}
}

// TestAppendWhereCondition tests condition appending
func TestAppendWhereCondition(t *testing.T) {
	tests := []struct {
		name      string
		baseWhere string
		condition string
		expected  string
	}{
		{
			name:      "empty base where",
			baseWhere: "",
			condition: "status = 'active'",
			expected:  "WHERE status = 'active'",
		},
		{
			name:      "existing where clause",
			baseWhere: "WHERE id = 1",
			condition: "status = 'active'",
			expected:  "WHERE id = 1 AND status = 'active'",
		},
		{
			name:      "complex base where",
			baseWhere: "WHERE a = 1 AND b = 2",
			condition: "c = 3",
			expected:  "WHERE a = 1 AND b = 2 AND c = 3",
		},
		{
			name:      "in clause condition",
			baseWhere: "WHERE type = 'movie'",
			condition: "user_id IN (1, 2, 3)",
			expected:  "WHERE type = 'movie' AND user_id IN (1, 2, 3)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appendWhereCondition(tt.baseWhere, tt.condition)
			if result != tt.expected {
				t.Errorf("appendWhereCondition(%q, %q) = %q, want %q",
					tt.baseWhere, tt.condition, result, tt.expected)
			}
		})
	}
}

// TestCalculateAdoptionRate tests adoption rate calculation
func TestCalculateAdoptionRate(t *testing.T) {
	tests := []struct {
		name         string
		adoptedCount int
		total        int
		expected     float64
	}{
		{
			name:         "50 percent adoption",
			adoptedCount: 50,
			total:        100,
			expected:     50.0,
		},
		{
			name:         "full adoption",
			adoptedCount: 100,
			total:        100,
			expected:     100.0,
		},
		{
			name:         "no adoption",
			adoptedCount: 0,
			total:        100,
			expected:     0.0,
		},
		{
			name:         "zero total - division by zero protection",
			adoptedCount: 50,
			total:        0,
			expected:     0.0,
		},
		{
			name:         "small adoption rate",
			adoptedCount: 1,
			total:        1000,
			expected:     0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateAdoptionRate(tt.adoptedCount, tt.total)
			if result != tt.expected {
				t.Errorf("calculateAdoptionRate(%d, %d) = %f, want %f",
					tt.adoptedCount, tt.total, result, tt.expected)
			}
		})
	}
}

// TestErrorContext tests error context wrapping
func TestErrorContext(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		result := errorContext("test operation", nil)
		if result != nil {
			t.Errorf("errorContext with nil error should return nil, got %v", result)
		}
	})

	t.Run("wraps error with context", func(t *testing.T) {
		originalErr := &testError{msg: "original error"}
		result := errorContext("test operation", originalErr)

		if result == nil {
			t.Fatal("errorContext should return wrapped error")
		}

		expected := "test operation: original error"
		if result.Error() != expected {
			t.Errorf("errorContext error message = %q, want %q", result.Error(), expected)
		}
	})

	t.Run("different operations", func(t *testing.T) {
		originalErr := &testError{msg: "db connection failed"}

		operations := []struct {
			op       string
			expected string
		}{
			{"fetching users", "fetching users: db connection failed"},
			{"updating record", "updating record: db connection failed"},
			{"deleting playback", "deleting playback: db connection failed"},
		}

		for _, op := range operations {
			result := errorContext(op.op, originalErr)
			if result.Error() != op.expected {
				t.Errorf("errorContext(%q, err) = %q, want %q", op.op, result.Error(), op.expected)
			}
		}
	})
}

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// TestParseAggregatedList tests parsing of aggregated lists
func TestParseAggregatedList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single item",
			input:    "item1",
			expected: []string{"item1"},
		},
		{
			name:     "comma separated",
			input:    "item1,item2,item3",
			expected: []string{"item1", "item2", "item3"},
		},
		{
			name:     "with spaces",
			input:    "item1, item2, item3",
			expected: []string{"item1", "item2", "item3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAggregatedList(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("parseAggregatedList(%q) returned %d items, want %d",
					tt.input, len(result), len(tt.expected))
				return
			}

			for i, item := range result {
				if item != tt.expected[i] {
					t.Errorf("parseAggregatedList(%q)[%d] = %q, want %q",
						tt.input, i, item, tt.expected[i])
				}
			}
		})
	}
}

// TestGenerateCapacityRecommendation tests capacity recommendation generation based on peak concurrent streams
func TestGenerateCapacityRecommendation(t *testing.T) {
	tests := []struct {
		name         string
		peak         float64
		avg          float64
		expectedText string
	}{
		{
			name:         "light usage - zero streams",
			peak:         0,
			avg:          0,
			expectedText: "Light usage",
		},
		{
			name:         "light usage - 1 stream",
			peak:         1,
			avg:          0.5,
			expectedText: "Light usage",
		},
		{
			name:         "light usage - 2 streams",
			peak:         2,
			avg:          1,
			expectedText: "Light usage",
		},
		{
			name:         "moderate usage - 3 streams",
			peak:         3,
			avg:          2,
			expectedText: "Moderate usage",
		},
		{
			name:         "moderate usage - 5 streams",
			peak:         5,
			avg:          3,
			expectedText: "Moderate usage",
		},
		{
			name:         "heavy usage - 6 streams",
			peak:         6,
			avg:          4,
			expectedText: "Heavy usage",
		},
		{
			name:         "heavy usage - 10 streams",
			peak:         10,
			avg:          6,
			expectedText: "Heavy usage",
		},
		{
			name:         "very heavy usage - 11 streams",
			peak:         11,
			avg:          8,
			expectedText: "Very heavy usage",
		},
		{
			name:         "very heavy usage - 20 streams",
			peak:         20,
			avg:          12,
			expectedText: "Very heavy usage",
		},
		{
			name:         "extreme usage - 21 streams",
			peak:         21,
			avg:          15,
			expectedText: "Extreme usage",
		},
		{
			name:         "extreme usage - 50 streams",
			peak:         50,
			avg:          30,
			expectedText: "Extreme usage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateCapacityRecommendation(tt.peak, tt.avg)
			if !stringContains(result, tt.expectedText) {
				t.Errorf("generateCapacityRecommendation(%f, %f) = %q, expected to contain %q",
					tt.peak, tt.avg, result, tt.expectedText)
			}
		})
	}
}

// Benchmark tests
func BenchmarkCalculatePercentage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		calculatePercentage(50, 100)
	}
}

func BenchmarkBuildWhereClause(b *testing.B) {
	clauses := []string{"a = 1", "b = 2", "c = 3"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildWhereClause(clauses)
	}
}

func BenchmarkJoinWithAnd(b *testing.B) {
	clauses := []string{"a = 1", "b = 2", "c = 3", "d = 4", "e = 5"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		joinWithAnd(clauses)
	}
}

func BenchmarkAppendWhereCondition(b *testing.B) {
	baseWhere := "WHERE id = 1 AND status = 'active'"
	condition := "user_id = 100"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		appendWhereCondition(baseWhere, condition)
	}
}
