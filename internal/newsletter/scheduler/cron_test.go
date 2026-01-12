// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package scheduler

import (
	"testing"
	"time"
)

func TestParseCron(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{
			name:    "daily at 9am",
			expr:    "0 9 * * *",
			wantErr: false,
		},
		{
			name:    "every 5 minutes",
			expr:    "*/5 * * * *",
			wantErr: false,
		},
		{
			name:    "monday at 9am",
			expr:    "0 9 * * 1",
			wantErr: false,
		},
		{
			name:    "first of month at midnight",
			expr:    "0 0 1 * *",
			wantErr: false,
		},
		{
			name:    "every hour on weekdays",
			expr:    "0 * * * 1-5",
			wantErr: false,
		},
		{
			name:    "multiple specific minutes",
			expr:    "0,15,30,45 * * * *",
			wantErr: false,
		},
		{
			name:    "too few fields",
			expr:    "0 9 * *",
			wantErr: true,
		},
		{
			name:    "too many fields",
			expr:    "0 9 * * * *",
			wantErr: true,
		},
		{
			name:    "invalid minute",
			expr:    "60 9 * * *",
			wantErr: true,
		},
		{
			name:    "invalid hour",
			expr:    "0 24 * * *",
			wantErr: true,
		},
		{
			name:    "invalid step",
			expr:    "*/0 * * * *",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCron(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCron() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCronExpression_NextRun(t *testing.T) {
	loc := time.UTC

	tests := []struct {
		name     string
		expr     string
		after    time.Time
		expected time.Time
	}{
		{
			name:     "daily at 9am from 8am",
			expr:     "0 9 * * *",
			after:    time.Date(2024, 1, 1, 8, 0, 0, 0, loc),
			expected: time.Date(2024, 1, 1, 9, 0, 0, 0, loc),
		},
		{
			name:     "daily at 9am from 10am (next day)",
			expr:     "0 9 * * *",
			after:    time.Date(2024, 1, 1, 10, 0, 0, 0, loc),
			expected: time.Date(2024, 1, 2, 9, 0, 0, 0, loc),
		},
		{
			name:     "every 5 minutes from :01",
			expr:     "*/5 * * * *",
			after:    time.Date(2024, 1, 1, 12, 1, 0, 0, loc),
			expected: time.Date(2024, 1, 1, 12, 5, 0, 0, loc),
		},
		{
			name:     "every 5 minutes from :05",
			expr:     "*/5 * * * *",
			after:    time.Date(2024, 1, 1, 12, 5, 0, 0, loc),
			expected: time.Date(2024, 1, 1, 12, 10, 0, 0, loc),
		},
		{
			name:     "monday at 9am from sunday",
			expr:     "0 9 * * 1",
			after:    time.Date(2024, 1, 7, 10, 0, 0, 0, loc), // Sunday Jan 7, 2024
			expected: time.Date(2024, 1, 8, 9, 0, 0, 0, loc),  // Monday Jan 8, 2024
		},
		{
			name:     "first of month from 15th",
			expr:     "0 0 1 * *",
			after:    time.Date(2024, 1, 15, 0, 0, 0, 0, loc),
			expected: time.Date(2024, 2, 1, 0, 0, 0, 0, loc),
		},
		{
			name:     "at minute 30 from minute 20",
			expr:     "30 * * * *",
			after:    time.Date(2024, 1, 1, 12, 20, 0, 0, loc),
			expected: time.Date(2024, 1, 1, 12, 30, 0, 0, loc),
		},
		{
			name:     "at minute 30 from minute 35 (next hour)",
			expr:     "30 * * * *",
			after:    time.Date(2024, 1, 1, 12, 35, 0, 0, loc),
			expected: time.Date(2024, 1, 1, 13, 30, 0, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cron, err := ParseCron(tt.expr)
			if err != nil {
				t.Fatalf("ParseCron() error = %v", err)
			}

			got := cron.NextRun(tt.after, loc)
			if !got.Equal(tt.expected) {
				t.Errorf("NextRun() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestCronExpression_NextRun_Timezone(t *testing.T) {
	// Test with different timezone
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("America/New_York timezone not available")
	}

	// Parse cron for 9am daily
	cron, err := ParseCron("0 9 * * *")
	if err != nil {
		t.Fatalf("ParseCron() error = %v", err)
	}

	// After 8am in New York
	after := time.Date(2024, 1, 1, 8, 0, 0, 0, newYork)
	next := cron.NextRun(after, newYork)

	// Should be 9am New York time
	expected := time.Date(2024, 1, 1, 9, 0, 0, 0, newYork)
	if !next.Equal(expected) {
		t.Errorf("NextRun() = %v, expected %v", next, expected)
	}
}

func TestCalculateNextRun(t *testing.T) {
	after := time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		cronExpr string
		timezone string
		wantErr  bool
	}{
		{
			name:     "valid expression UTC",
			cronExpr: "0 9 * * *",
			timezone: "",
			wantErr:  false,
		},
		{
			name:     "valid expression with timezone",
			cronExpr: "0 9 * * *",
			timezone: "America/New_York",
			wantErr:  false,
		},
		{
			name:     "invalid expression",
			cronExpr: "invalid",
			timezone: "",
			wantErr:  true,
		},
		{
			name:     "invalid timezone",
			cronExpr: "0 9 * * *",
			timezone: "Invalid/Timezone",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateNextRun(tt.cronExpr, after, tt.timezone)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateNextRun() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseField(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		min     int
		max     int
		want    []int
		wantErr bool
	}{
		{
			name:    "wildcard",
			field:   "*",
			min:     0,
			max:     5,
			want:    []int{0, 1, 2, 3, 4, 5},
			wantErr: false,
		},
		{
			name:    "single value",
			field:   "5",
			min:     0,
			max:     59,
			want:    []int{5},
			wantErr: false,
		},
		{
			name:    "range",
			field:   "1-5",
			min:     0,
			max:     10,
			want:    []int{1, 2, 3, 4, 5},
			wantErr: false,
		},
		{
			name:    "step from start",
			field:   "*/15",
			min:     0,
			max:     59,
			want:    []int{0, 15, 30, 45},
			wantErr: false,
		},
		{
			name:    "step in range",
			field:   "0-30/10",
			min:     0,
			max:     59,
			want:    []int{0, 10, 20, 30},
			wantErr: false,
		},
		{
			name:    "list",
			field:   "1,3,5",
			min:     0,
			max:     10,
			want:    []int{1, 3, 5},
			wantErr: false,
		},
		{
			name:    "value out of range",
			field:   "60",
			min:     0,
			max:     59,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid range",
			field:   "10-5",
			min:     0,
			max:     59,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseField(tt.field, tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseField() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !equalIntSlices(got, tt.want) {
				t.Errorf("parseField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCronExpression_Matches(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		time    time.Time
		matches bool
	}{
		{
			name:    "exact match",
			expr:    "30 9 15 1 *",
			time:    time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC),
			matches: true,
		},
		{
			name:    "minute mismatch",
			expr:    "30 9 15 1 *",
			time:    time.Date(2024, 1, 15, 9, 31, 0, 0, time.UTC),
			matches: false,
		},
		{
			name:    "hour mismatch",
			expr:    "30 9 15 1 *",
			time:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			matches: false,
		},
		{
			name:    "day of week match (Monday)",
			expr:    "0 9 * * 1",
			time:    time.Date(2024, 1, 8, 9, 0, 0, 0, time.UTC), // Monday
			matches: true,
		},
		{
			name:    "day of week mismatch (Tuesday)",
			expr:    "0 9 * * 1",
			time:    time.Date(2024, 1, 9, 9, 0, 0, 0, time.UTC), // Tuesday
			matches: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cron, err := ParseCron(tt.expr)
			if err != nil {
				t.Fatalf("ParseCron() error = %v", err)
			}

			if got := cron.matches(tt.time); got != tt.matches {
				t.Errorf("matches() = %v, want %v", got, tt.matches)
			}
		})
	}
}

// equalIntSlices compares two int slices for equality.
func equalIntSlices(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
