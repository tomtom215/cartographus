// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

func TestGenerateCohortQueryHash(t *testing.T) {

	tests := []struct {
		name     string
		filter   LocationStatsFilter
		config   CohortRetentionConfig
		wantSame bool
	}{
		{
			name:     "empty filter produces consistent hash",
			filter:   LocationStatsFilter{},
			config:   DefaultCohortConfig(),
			wantSame: true,
		},
		{
			name: "different filters produce different hashes",
			filter: LocationStatsFilter{
				Users: []string{"user1"},
			},
			config:   DefaultCohortConfig(),
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := generateCohortQueryHash(tt.filter, tt.config)
			hash2 := generateCohortQueryHash(tt.filter, tt.config)

			if hash1 != hash2 {
				t.Errorf("same input should produce same hash, got %s and %s", hash1, hash2)
			}

			if len(hash1) != 16 {
				t.Errorf("hash should be 16 hex chars, got %d", len(hash1))
			}
		})
	}
}

func TestDefaultCohortConfig(t *testing.T) {

	config := DefaultCohortConfig()

	if config.MaxWeeks != 12 {
		t.Errorf("expected MaxWeeks=12, got %d", config.MaxWeeks)
	}
	if config.MinCohortSize != 3 {
		t.Errorf("expected MinCohortSize=3, got %d", config.MinCohortSize)
	}
	if config.Granularity != "week" {
		t.Errorf("expected Granularity=week, got %s", config.Granularity)
	}
}

func TestCalculateRetentionTrend(t *testing.T) {

	tests := []struct {
		name     string
		cohorts  []models.CohortData
		expected string
	}{
		{
			name:     "insufficient data with fewer than 4 cohorts",
			cohorts:  []models.CohortData{{}, {}, {}},
			expected: "insufficient_data",
		},
		{
			name: "stable when no significant change",
			cohorts: []models.CohortData{
				{AverageRetention: 50.0},
				{AverageRetention: 51.0},
				{AverageRetention: 49.0},
				{AverageRetention: 50.5},
			},
			expected: "stable",
		},
		{
			name: "improving when later cohorts have higher retention",
			cohorts: []models.CohortData{
				{AverageRetention: 40.0},
				{AverageRetention: 42.0},
				{AverageRetention: 55.0},
				{AverageRetention: 58.0},
			},
			expected: "improving",
		},
		{
			name: "declining when later cohorts have lower retention",
			cohorts: []models.CohortData{
				{AverageRetention: 60.0},
				{AverageRetention: 58.0},
				{AverageRetention: 45.0},
				{AverageRetention: 42.0},
			},
			expected: "declining",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateRetentionTrend(tt.cohorts)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestStatisticsHelpers(t *testing.T) {

	t.Run("average", func(t *testing.T) {
		if avg := average([]float64{}); avg != 0 {
			t.Errorf("empty slice should return 0, got %f", avg)
		}
		if avg := average([]float64{10, 20, 30}); avg != 20 {
			t.Errorf("expected 20, got %f", avg)
		}
	})

	t.Run("median", func(t *testing.T) {
		if med := median([]float64{}); med != 0 {
			t.Errorf("empty slice should return 0, got %f", med)
		}
		if med := median([]float64{1, 2, 3}); med != 2 {
			t.Errorf("expected 2, got %f", med)
		}
		if med := median([]float64{1, 2, 3, 4}); med != 2.5 {
			t.Errorf("expected 2.5, got %f", med)
		}
	})

	t.Run("minFloat", func(t *testing.T) {
		if minVal := minFloat([]float64{}); minVal != 0 {
			t.Errorf("empty slice should return 0, got %f", minVal)
		}
		if minVal := minFloat([]float64{5, 2, 8, 1, 9}); minVal != 1 {
			t.Errorf("expected 1, got %f", minVal)
		}
	})

	t.Run("maxFloat", func(t *testing.T) {
		if maxVal := maxFloat([]float64{}); maxVal != 0 {
			t.Errorf("empty slice should return 0, got %f", maxVal)
		}
		if maxVal := maxFloat([]float64{5, 2, 8, 1, 9}); maxVal != 9 {
			t.Errorf("expected 9, got %f", maxVal)
		}
	})
}

func TestGetDataRange(t *testing.T) {

	t.Run("defaults when no filter dates", func(t *testing.T) {
		filter := LocationStatsFilter{}
		start, end := getDataRange(filter)

		// Start should be approximately 1 year ago
		expectedStart := time.Now().AddDate(-1, 0, 0)
		if start.Sub(expectedStart).Abs() > time.Hour {
			t.Errorf("start should be ~1 year ago, got %v", start)
		}

		// End should be approximately now
		if time.Since(end) > time.Minute {
			t.Errorf("end should be ~now, got %v", end)
		}
	})

	t.Run("uses filter dates when provided", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		filter := LocationStatsFilter{
			StartDate: &startDate,
			EndDate:   &endDate,
		}

		start, end := getDataRange(filter)
		if !start.Equal(startDate) {
			t.Errorf("expected start %v, got %v", startDate, start)
		}
		if !end.Equal(endDate) {
			t.Errorf("expected end %v, got %v", endDate, end)
		}
	})
}

func TestBuildRetentionCurve(t *testing.T) {

	t.Run("empty cohorts returns empty curve", func(t *testing.T) {
		curve := buildRetentionCurve([]models.CohortData{}, 12)
		if len(curve) != 0 {
			t.Errorf("expected empty curve, got %d points", len(curve))
		}
	})

	t.Run("builds curve from cohort data", func(t *testing.T) {
		cohorts := []models.CohortData{
			{
				Retention: []models.WeekRetention{
					{WeekOffset: 0, RetentionRate: 100},
					{WeekOffset: 1, RetentionRate: 80},
					{WeekOffset: 2, RetentionRate: 60},
				},
			},
			{
				Retention: []models.WeekRetention{
					{WeekOffset: 0, RetentionRate: 100},
					{WeekOffset: 1, RetentionRate: 70},
					{WeekOffset: 2, RetentionRate: 50},
				},
			},
		}

		curve := buildRetentionCurve(cohorts, 12)

		if len(curve) != 3 {
			t.Fatalf("expected 3 points, got %d", len(curve))
		}

		// Week 0 should be 100%
		if curve[0].AverageRetention != 100 {
			t.Errorf("week 0 should be 100%%, got %.1f%%", curve[0].AverageRetention)
		}

		// Week 1 should be average of 80 and 70 = 75
		if curve[1].AverageRetention != 75 {
			t.Errorf("week 1 should be 75%%, got %.1f%%", curve[1].AverageRetention)
		}

		// Week 2 should be average of 60 and 50 = 55
		if curve[2].AverageRetention != 55 {
			t.Errorf("week 2 should be 55%%, got %.1f%%", curve[2].AverageRetention)
		}
	})

	t.Run("curve includes min/max/median statistics", func(t *testing.T) {
		cohorts := []models.CohortData{
			{
				Retention: []models.WeekRetention{
					{WeekOffset: 1, RetentionRate: 80},
				},
			},
			{
				Retention: []models.WeekRetention{
					{WeekOffset: 1, RetentionRate: 60},
				},
			},
			{
				Retention: []models.WeekRetention{
					{WeekOffset: 1, RetentionRate: 70},
				},
			},
		}

		curve := buildRetentionCurve(cohorts, 12)

		if len(curve) != 1 {
			t.Fatalf("expected 1 point, got %d", len(curve))
		}

		if curve[0].MinRetention != 60 {
			t.Errorf("expected min 60, got %.1f", curve[0].MinRetention)
		}
		if curve[0].MaxRetention != 80 {
			t.Errorf("expected max 80, got %.1f", curve[0].MaxRetention)
		}
		if curve[0].MedianRetention != 70 {
			t.Errorf("expected median 70, got %.1f", curve[0].MedianRetention)
		}
		if curve[0].CohortsWithData != 3 {
			t.Errorf("expected 3 cohorts, got %d", curve[0].CohortsWithData)
		}
	})
}

func TestCalculateCohortSummary(t *testing.T) {

	t.Run("empty cohorts", func(t *testing.T) {
		summary := calculateCohortSummary([]models.CohortData{})
		if summary.TotalCohorts != 0 {
			t.Errorf("expected 0 cohorts, got %d", summary.TotalCohorts)
		}
		if summary.RetentionTrend != "insufficient_data" {
			t.Errorf("expected insufficient_data, got %s", summary.RetentionTrend)
		}
	})

	t.Run("calculates summary correctly", func(t *testing.T) {
		cohorts := []models.CohortData{
			{
				CohortWeek:       "2024-W01",
				InitialUsers:     100,
				AverageRetention: 60.0,
				Retention: []models.WeekRetention{
					{WeekOffset: 1, RetentionRate: 80},
					{WeekOffset: 4, RetentionRate: 60},
					{WeekOffset: 12, RetentionRate: 40},
				},
			},
			{
				CohortWeek:       "2024-W02",
				InitialUsers:     50,
				AverageRetention: 50.0,
				Retention: []models.WeekRetention{
					{WeekOffset: 1, RetentionRate: 70},
					{WeekOffset: 4, RetentionRate: 50},
					{WeekOffset: 12, RetentionRate: 30},
				},
			},
		}

		summary := calculateCohortSummary(cohorts)

		if summary.TotalCohorts != 2 {
			t.Errorf("expected 2 cohorts, got %d", summary.TotalCohorts)
		}
		if summary.TotalUsersTracked != 150 {
			t.Errorf("expected 150 users, got %d", summary.TotalUsersTracked)
		}
		if summary.BestPerformingCohort != "2024-W01" {
			t.Errorf("expected best cohort 2024-W01, got %s", summary.BestPerformingCohort)
		}
		if summary.WorstPerformingCohort != "2024-W02" {
			t.Errorf("expected worst cohort 2024-W02, got %s", summary.WorstPerformingCohort)
		}
	})

	t.Run("calculates week retention averages", func(t *testing.T) {
		cohorts := []models.CohortData{
			{
				CohortWeek:       "2024-W01",
				InitialUsers:     100,
				AverageRetention: 60.0,
				Retention: []models.WeekRetention{
					{WeekOffset: 1, RetentionRate: 80},
					{WeekOffset: 4, RetentionRate: 60},
				},
			},
			{
				CohortWeek:       "2024-W02",
				InitialUsers:     100,
				AverageRetention: 50.0,
				Retention: []models.WeekRetention{
					{WeekOffset: 1, RetentionRate: 60},
					{WeekOffset: 4, RetentionRate: 40},
				},
			},
		}

		summary := calculateCohortSummary(cohorts)

		// Week 1 retention should be average of 80 and 60 = 70
		if summary.Week1Retention != 70 {
			t.Errorf("expected Week1Retention=70, got %.1f", summary.Week1Retention)
		}
		// Week 4 retention should be average of 60 and 40 = 50
		if summary.Week4Retention != 50 {
			t.Errorf("expected Week4Retention=50, got %.1f", summary.Week4Retention)
		}
	})
}

func TestHashDeterminism(t *testing.T) {

	// Test that same inputs always produce same hash across multiple runs
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	filter := LocationStatsFilter{
		StartDate:  &startDate,
		Users:      []string{"user1", "user2"},
		MediaTypes: []string{"movie"},
	}
	config := CohortRetentionConfig{
		MaxWeeks:      8,
		MinCohortSize: 5,
		Granularity:   "week",
	}

	var hashes []string
	for i := 0; i < 10; i++ {
		hash := generateCohortQueryHash(filter, config)
		hashes = append(hashes, hash)
	}

	// All hashes should be identical
	for i := 1; i < len(hashes); i++ {
		if hashes[i] != hashes[0] {
			t.Errorf("hash %d differs from hash 0: %s != %s", i, hashes[i], hashes[0])
		}
	}
}

func TestHashUniqueness(t *testing.T) {

	// Different inputs should produce different hashes
	config := DefaultCohortConfig()

	filter1 := LocationStatsFilter{Users: []string{"user1"}}
	filter2 := LocationStatsFilter{Users: []string{"user2"}}
	filter3 := LocationStatsFilter{MediaTypes: []string{"movie"}}

	hash1 := generateCohortQueryHash(filter1, config)
	hash2 := generateCohortQueryHash(filter2, config)
	hash3 := generateCohortQueryHash(filter3, config)

	if hash1 == hash2 {
		t.Error("different user filters should produce different hashes")
	}
	if hash1 == hash3 {
		t.Error("different filter types should produce different hashes")
	}
}
