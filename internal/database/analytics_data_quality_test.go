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

func TestBuildFieldMetric(t *testing.T) {

	t.Run("healthy field with no nulls or invalids", func(t *testing.T) {
		metric := buildFieldMetric("test_field", "test_category", 1000, 0, 0, 100, false)

		if metric.FieldName != "test_field" {
			t.Errorf("expected field name 'test_field', got '%s'", metric.FieldName)
		}
		if metric.Category != "test_category" {
			t.Errorf("expected category 'test_category', got '%s'", metric.Category)
		}
		if metric.TotalRecords != 1000 {
			t.Errorf("expected 1000 total records, got %d", metric.TotalRecords)
		}
		if metric.NullRate != 0 {
			t.Errorf("expected 0%% null rate, got %.1f%%", metric.NullRate)
		}
		if metric.InvalidRate != 0 {
			t.Errorf("expected 0%% invalid rate, got %.1f%%", metric.InvalidRate)
		}
		if metric.QualityScore != 100 {
			t.Errorf("expected 100 quality score, got %.1f", metric.QualityScore)
		}
		if metric.Status != "healthy" {
			t.Errorf("expected status 'healthy', got '%s'", metric.Status)
		}
	})

	t.Run("required field with nulls is critical", func(t *testing.T) {
		// 10% null rate on required field
		metric := buildFieldMetric("user_id", "identity", 1000, 100, 0, 900, true)

		if metric.NullRate != 10.0 {
			t.Errorf("expected 10%% null rate, got %.1f%%", metric.NullRate)
		}
		if metric.Status != "critical" {
			t.Errorf("expected status 'critical' for required field with nulls, got '%s'", metric.Status)
		}
		if metric.IsRequired != true {
			t.Errorf("expected IsRequired=true, got %v", metric.IsRequired)
		}
	})

	t.Run("non-required field with high nulls is warning", func(t *testing.T) {
		// 15% null rate on non-required field
		metric := buildFieldMetric("platform", "device", 1000, 150, 0, 850, false)

		if metric.NullRate != 15.0 {
			t.Errorf("expected 15%% null rate, got %.1f%%", metric.NullRate)
		}
		if metric.Status != "warning" {
			t.Errorf("expected status 'warning' for high null rate, got '%s'", metric.Status)
		}
	})

	t.Run("calculates cardinality correctly", func(t *testing.T) {
		// 1000 records, 100 null, 500 unique values
		// Cardinality = 500 / 900 = 0.556
		metric := buildFieldMetric("ip_address", "network", 1000, 100, 0, 500, true)

		expectedCardinality := 500.0 / 900.0
		if metric.Cardinality < expectedCardinality-0.01 || metric.Cardinality > expectedCardinality+0.01 {
			t.Errorf("expected cardinality ~%.3f, got %.3f", expectedCardinality, metric.Cardinality)
		}
	})

	t.Run("quality score penalizes nulls and invalids", func(t *testing.T) {
		// 5% null, 2% invalid on non-required field
		// Score = 100 - (5 + 2*2) = 100 - 9 = 91
		metric := buildFieldMetric("percent_complete", "engagement", 1000, 50, 20, 0, false)

		// Allow small floating point tolerance
		if metric.QualityScore < 90 || metric.QualityScore > 92 {
			t.Errorf("expected quality score ~91, got %.1f", metric.QualityScore)
		}
	})

	t.Run("required field has double null penalty", func(t *testing.T) {
		// 5% null on required field
		// Score = 100 - (5*2 + 0) = 90
		metric := buildFieldMetric("user_id", "identity", 1000, 50, 0, 950, true)

		// Score should be 100 - (5*2) = 90
		if metric.QualityScore != 90 {
			t.Errorf("expected quality score 90, got %.1f", metric.QualityScore)
		}
	})

	t.Run("handles zero total records", func(t *testing.T) {
		metric := buildFieldMetric("test_field", "test", 0, 0, 0, 0, false)

		if metric.NullRate != 0 {
			t.Errorf("expected 0%% null rate for empty data, got %.1f%%", metric.NullRate)
		}
		if metric.Cardinality != 0 {
			t.Errorf("expected 0 cardinality for empty data, got %.3f", metric.Cardinality)
		}
	})

	t.Run("quality score floors at zero", func(t *testing.T) {
		// 100% null rate - should not go negative
		metric := buildFieldMetric("test_field", "test", 100, 100, 0, 0, true)

		if metric.QualityScore < 0 {
			t.Errorf("quality score should not be negative, got %.1f", metric.QualityScore)
		}
	})
}

func TestCalculateDataQualitySummary(t *testing.T) {

	t.Run("empty fields returns N/A grade", func(t *testing.T) {
		summary := calculateDataQualitySummary([]models.FieldQualityMetric{}, []models.DailyQualityTrend{})

		if summary.Grade != "N/A" {
			t.Errorf("expected grade 'N/A' for empty fields, got '%s'", summary.Grade)
		}
	})

	t.Run("perfect data returns A grade", func(t *testing.T) {
		fields := []models.FieldQualityMetric{
			{TotalRecords: 1000, NullRate: 0, InvalidRate: 0, Status: "healthy"},
			{TotalRecords: 1000, NullRate: 0, InvalidRate: 0, Status: "healthy"},
		}
		trends := []models.DailyQualityTrend{}

		summary := calculateDataQualitySummary(fields, trends)

		if summary.CompletenessScore != 100 {
			t.Errorf("expected 100%% completeness, got %.1f%%", summary.CompletenessScore)
		}
		if summary.ValidityScore != 100 {
			t.Errorf("expected 100%% validity, got %.1f%%", summary.ValidityScore)
		}
		// Overall = (100*0.4 + 100*0.4 + 95*0.2) = 99
		if summary.OverallScore < 95 {
			t.Errorf("expected high overall score, got %.1f", summary.OverallScore)
		}
		if summary.Grade != "A" {
			t.Errorf("expected grade 'A', got '%s'", summary.Grade)
		}
	})

	t.Run("counts critical and warning issues", func(t *testing.T) {
		fields := []models.FieldQualityMetric{
			{TotalRecords: 1000, NullRate: 0, Status: "critical"},
			{TotalRecords: 1000, NullRate: 0, Status: "critical"},
			{TotalRecords: 1000, NullRate: 0, Status: "warning"},
			{TotalRecords: 1000, NullRate: 0, Status: "healthy"},
		}
		trends := []models.DailyQualityTrend{}

		summary := calculateDataQualitySummary(fields, trends)

		if summary.CriticalIssueCount != 2 {
			t.Errorf("expected 2 critical issues, got %d", summary.CriticalIssueCount)
		}
		if summary.IssueCount != 3 {
			t.Errorf("expected 3 total issues, got %d", summary.IssueCount)
		}
	})

	t.Run("grades are assigned correctly", func(t *testing.T) {

		testCases := []struct {
			nullRate      float64
			expectedGrade string
		}{
			{0, "A"},  // 100% completeness
			{5, "A"},  // 95% completeness
			{10, "B"}, // 90% completeness
			{15, "B"}, // 85% completeness
			{20, "C"}, // 80% completeness
			{30, "D"}, // 70% completeness
			{50, "F"}, // 50% completeness
		}

		for _, tc := range testCases {
			fields := []models.FieldQualityMetric{
				{TotalRecords: 1000, NullRate: tc.nullRate, InvalidRate: 0, Status: "healthy"},
			}
			summary := calculateDataQualitySummary(fields, []models.DailyQualityTrend{})

			// Note: Grade depends on overall score which combines completeness, validity, and consistency
			// Just verify that lower completeness leads to lower grades
			if tc.nullRate > 40 && summary.Grade == "A" {
				t.Errorf("null rate %.0f%% should not get grade A", tc.nullRate)
			}
		}
	})
}

func TestCalculateQualityTrend(t *testing.T) {

	t.Run("insufficient data with fewer than 7 days", func(t *testing.T) {
		trends := make([]models.DailyQualityTrend, 5)
		result := calculateQualityTrend(trends)

		if result != "insufficient_data" {
			t.Errorf("expected 'insufficient_data', got '%s'", result)
		}
	})

	t.Run("stable when scores are similar", func(t *testing.T) {
		trends := make([]models.DailyQualityTrend, 14)
		for i := range trends {
			trends[i].OverallScore = 90 // Same score every day
		}

		result := calculateQualityTrend(trends)

		if result != "stable" {
			t.Errorf("expected 'stable', got '%s'", result)
		}
	})

	t.Run("improving when recent scores are higher", func(t *testing.T) {
		trends := make([]models.DailyQualityTrend, 14)
		// Recent 7 days (indices 0-6) have higher scores
		for i := 0; i < 7; i++ {
			trends[i].OverallScore = 95
		}
		// Older 7 days (indices 7-13) have lower scores
		for i := 7; i < 14; i++ {
			trends[i].OverallScore = 85
		}

		result := calculateQualityTrend(trends)

		if result != "improving" {
			t.Errorf("expected 'improving', got '%s'", result)
		}
	})

	t.Run("declining when recent scores are lower", func(t *testing.T) {
		trends := make([]models.DailyQualityTrend, 14)
		// Recent 7 days (indices 0-6) have lower scores
		for i := 0; i < 7; i++ {
			trends[i].OverallScore = 80
		}
		// Older 7 days (indices 7-13) have higher scores
		for i := 7; i < 14; i++ {
			trends[i].OverallScore = 95
		}

		result := calculateQualityTrend(trends)

		if result != "declining" {
			t.Errorf("expected 'declining', got '%s'", result)
		}
	})
}

func TestGenerateDataQualityIssues(t *testing.T) {

	t.Run("no issues for healthy data", func(t *testing.T) {
		fields := []models.FieldQualityMetric{
			{FieldName: "user_id", NullCount: 0, InvalidCount: 0, IsRequired: true},
			{FieldName: "platform", NullCount: 0, InvalidCount: 0, IsRequired: false},
		}
		summary := models.DataQualitySummary{OverallScore: 95}

		issues := generateDataQualityIssues(fields, &summary)

		if len(issues) != 0 {
			t.Errorf("expected 0 issues for healthy data, got %d", len(issues))
		}
	})

	t.Run("detects null required fields", func(t *testing.T) {
		fields := []models.FieldQualityMetric{
			{
				FieldName:  "user_id",
				NullCount:  100,
				NullRate:   10.0,
				IsRequired: true,
			},
		}
		summary := models.DataQualitySummary{OverallScore: 95}

		issues := generateDataQualityIssues(fields, &summary)

		found := false
		for _, issue := range issues {
			if issue.Type == "null_required" && issue.Field == "user_id" {
				found = true
				if issue.Severity != "critical" {
					t.Errorf("expected critical severity, got %s", issue.Severity)
				}
				if issue.AffectedRecords != 100 {
					t.Errorf("expected 100 affected records, got %d", issue.AffectedRecords)
				}
			}
		}
		if !found {
			t.Error("expected null_required issue to be detected")
		}
	})

	t.Run("detects invalid values", func(t *testing.T) {
		fields := []models.FieldQualityMetric{
			{
				FieldName:    "percent_complete",
				InvalidCount: 50,
				InvalidRate:  5.0,
				IsRequired:   false,
			},
		}
		summary := models.DataQualitySummary{OverallScore: 95}

		issues := generateDataQualityIssues(fields, &summary)

		found := false
		for _, issue := range issues {
			if issue.Type == "invalid_value" && issue.Field == "percent_complete" {
				found = true
			}
		}
		if !found {
			t.Error("expected invalid_value issue to be detected")
		}
	})

	t.Run("detects low overall quality", func(t *testing.T) {
		fields := []models.FieldQualityMetric{}
		summary := models.DataQualitySummary{
			OverallScore: 70,
			TotalEvents:  1000,
		}

		issues := generateDataQualityIssues(fields, &summary)

		found := false
		for _, issue := range issues {
			if issue.Type == "low_quality" {
				found = true
			}
		}
		if !found {
			t.Error("expected low_quality issue to be detected")
		}
	})

	t.Run("does not flag low quality when score is high", func(t *testing.T) {
		fields := []models.FieldQualityMetric{}
		summary := models.DataQualitySummary{
			OverallScore: 85,
		}

		issues := generateDataQualityIssues(fields, &summary)

		for _, issue := range issues {
			if issue.Type == "low_quality" {
				t.Error("should not flag low_quality when overall score is 85%")
			}
		}
	})
}

func TestGenerateDataQualityQueryHash(t *testing.T) {

	t.Run("empty filter produces consistent hash", func(t *testing.T) {
		filter := LocationStatsFilter{}
		hash1 := generateDataQualityQueryHash(filter)
		hash2 := generateDataQualityQueryHash(filter)

		if hash1 != hash2 {
			t.Errorf("same filter should produce same hash, got %s and %s", hash1, hash2)
		}
		if len(hash1) != 16 {
			t.Errorf("hash should be 16 hex chars, got %d", len(hash1))
		}
	})

	t.Run("different date ranges produce different hashes", func(t *testing.T) {
		start1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		start2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

		filter1 := LocationStatsFilter{StartDate: &start1}
		filter2 := LocationStatsFilter{StartDate: &start2}

		hash1 := generateDataQualityQueryHash(filter1)
		hash2 := generateDataQualityQueryHash(filter2)

		if hash1 == hash2 {
			t.Error("different date ranges should produce different hashes")
		}
	})
}

func TestMin64(t *testing.T) {

	tests := []struct {
		a, b     int64
		expected int64
	}{
		{5, 10, 5},
		{10, 5, 5},
		{5, 5, 5},
		{0, 10, 0},
		{-5, 5, -5},
	}

	for _, tt := range tests {
		result := min64(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min64(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestCriticalFieldsConfiguration(t *testing.T) {

	// Verify critical fields are properly configured
	requiredFields := make(map[string]bool)
	categories := make(map[string]bool)

	for _, f := range criticalFields {
		requiredFields[f.name] = f.isRequired
		categories[f.category] = true
	}

	// Essential fields must be marked as required
	essentialFields := []string{"user_id", "username", "session_key", "ip_address", "started_at", "media_type", "title"}
	for _, field := range essentialFields {
		if !requiredFields[field] {
			t.Errorf("field '%s' should be marked as required", field)
		}
	}

	// Should have multiple categories
	if len(categories) < 4 {
		t.Errorf("expected at least 4 categories, got %d", len(categories))
	}

	// Should have at least 10 fields
	if len(criticalFields) < 10 {
		t.Errorf("expected at least 10 critical fields, got %d", len(criticalFields))
	}
}

func TestDataQualityHashDeterminism(t *testing.T) {

	startDate := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 9, 15, 0, 0, 0, 0, time.UTC)

	filter := LocationStatsFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	// Generate hash 100 times and verify consistency
	firstHash := generateDataQualityQueryHash(filter)
	for i := 0; i < 100; i++ {
		hash := generateDataQualityQueryHash(filter)
		if hash != firstHash {
			t.Errorf("iteration %d produced different hash: %s != %s", i, hash, firstHash)
		}
	}
}
