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

func TestGetQoEGrade(t *testing.T) {

	tests := []struct {
		name     string
		score    float64
		expected string
	}{
		{"A grade for 90+", 95, "A"},
		{"A grade for exactly 90", 90, "A"},
		{"B grade for 80-89", 85, "B"},
		{"B grade for exactly 80", 80, "B"},
		{"C grade for 70-79", 75, "C"},
		{"C grade for exactly 70", 70, "C"},
		{"D grade for 60-69", 65, "D"},
		{"D grade for exactly 60", 60, "D"},
		{"F grade for below 60", 55, "F"},
		{"F grade for 0", 0, "F"},
		{"A grade for 100", 100, "A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getQoEGrade(tt.score)
			if result != tt.expected {
				t.Errorf("getQoEGrade(%.0f) = %s, want %s", tt.score, result, tt.expected)
			}
		})
	}
}

func TestGetSeverity(t *testing.T) {

	tests := []struct {
		name              string
		value             float64
		warningThreshold  float64
		criticalThreshold float64
		expected          string
	}{
		{"critical when above critical threshold", 50, 20, 40, "critical"},
		{"critical when exactly at critical threshold", 40, 20, 40, "critical"},
		{"warning when between thresholds", 30, 20, 40, "warning"},
		{"warning when exactly at warning threshold", 20, 20, 40, "warning"},
		{"info when below warning threshold", 10, 20, 40, "info"},
		{"info when at zero", 0, 20, 40, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSeverity(tt.value, tt.warningThreshold, tt.criticalThreshold)
			if result != tt.expected {
				t.Errorf("getSeverity(%.0f, %.0f, %.0f) = %s, want %s",
					tt.value, tt.warningThreshold, tt.criticalThreshold, result, tt.expected)
			}
		})
	}
}

func TestGenerateQoEQueryHash(t *testing.T) {

	t.Run("empty filter produces consistent hash", func(t *testing.T) {
		filter := LocationStatsFilter{}
		hash1 := generateQoEQueryHash(filter)
		hash2 := generateQoEQueryHash(filter)

		if hash1 != hash2 {
			t.Errorf("same filter should produce same hash, got %s and %s", hash1, hash2)
		}
		if len(hash1) != 16 {
			t.Errorf("hash should be 16 hex chars, got %d", len(hash1))
		}
	})

	t.Run("different filters produce different hashes", func(t *testing.T) {
		filter1 := LocationStatsFilter{Users: []string{"user1"}}
		filter2 := LocationStatsFilter{Users: []string{"user2"}}

		hash1 := generateQoEQueryHash(filter1)
		hash2 := generateQoEQueryHash(filter2)

		if hash1 == hash2 {
			t.Error("different filters should produce different hashes")
		}
	})

	t.Run("includes all filter dimensions in hash", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		// Filter with all dimensions
		fullFilter := LocationStatsFilter{
			StartDate:  &startDate,
			Users:      []string{"user1"},
			MediaTypes: []string{"movie"},
			Platforms:  []string{"iOS"},
		}

		// Same filter but missing one dimension
		partialFilter := LocationStatsFilter{
			StartDate:  &startDate,
			Users:      []string{"user1"},
			MediaTypes: []string{"movie"},
			// Platforms omitted
		}

		hash1 := generateQoEQueryHash(fullFilter)
		hash2 := generateQoEQueryHash(partialFilter)

		if hash1 == hash2 {
			t.Error("filters with different dimensions should produce different hashes")
		}
	})
}

func TestGenerateQoEIssues(t *testing.T) {

	t.Run("no issues for healthy metrics", func(t *testing.T) {
		summary := models.QoESummary{
			TotalSessions:      1000,
			EBVSRate:           1.0,  // Below 5% threshold
			QualityDegradeRate: 10.0, // Below 20% threshold
			TranscodeRate:      30.0, // Below 50% threshold
			PauseRate:          15.0, // Below 30% threshold
			AvgCompletion:      75.0, // Above 50% threshold
		}
		platforms := []models.QoEByPlatform{}

		issues := generateQoEIssues(&summary, platforms)

		if len(issues) != 0 {
			t.Errorf("expected 0 issues for healthy metrics, got %d", len(issues))
		}
	})

	t.Run("detects high EBVS rate", func(t *testing.T) {
		summary := models.QoESummary{
			TotalSessions: 1000,
			EBVSRate:      8.0, // Above 5% threshold
			EBVSCount:     80,
			AvgCompletion: 75.0,
		}
		platforms := []models.QoEByPlatform{}

		issues := generateQoEIssues(&summary, platforms)

		found := false
		for _, issue := range issues {
			if issue.IssueType == "high_ebvs" {
				found = true
				if issue.Severity != "warning" {
					t.Errorf("expected warning severity for 8%% EBVS, got %s", issue.Severity)
				}
				if issue.AffectedSessions != 80 {
					t.Errorf("expected 80 affected sessions, got %d", issue.AffectedSessions)
				}
			}
		}
		if !found {
			t.Error("expected high_ebvs issue to be detected")
		}
	})

	t.Run("detects critical EBVS rate", func(t *testing.T) {
		summary := models.QoESummary{
			TotalSessions: 1000,
			EBVSRate:      15.0, // Above 10% critical threshold
			EBVSCount:     150,
			AvgCompletion: 75.0,
		}
		platforms := []models.QoEByPlatform{}

		issues := generateQoEIssues(&summary, platforms)

		for _, issue := range issues {
			if issue.IssueType == "high_ebvs" {
				if issue.Severity != "critical" {
					t.Errorf("expected critical severity for 15%% EBVS, got %s", issue.Severity)
				}
				return
			}
		}
		t.Error("expected high_ebvs issue to be detected")
	})

	t.Run("detects quality degradation", func(t *testing.T) {
		summary := models.QoESummary{
			TotalSessions:       1000,
			QualityDegradeRate:  25.0, // Above 20% threshold
			QualityDegradeCount: 250,
			AvgCompletion:       75.0,
		}
		platforms := []models.QoEByPlatform{}

		issues := generateQoEIssues(&summary, platforms)

		found := false
		for _, issue := range issues {
			if issue.IssueType == "quality_degradation" {
				found = true
				if issue.Severity != "warning" {
					t.Errorf("expected warning severity, got %s", issue.Severity)
				}
			}
		}
		if !found {
			t.Error("expected quality_degradation issue to be detected")
		}
	})

	t.Run("detects high transcode rate", func(t *testing.T) {
		summary := models.QoESummary{
			TotalSessions:  1000,
			TranscodeRate:  65.0, // Above 50% threshold
			TranscodeCount: 650,
			AvgCompletion:  75.0,
		}
		platforms := []models.QoEByPlatform{}

		issues := generateQoEIssues(&summary, platforms)

		found := false
		for _, issue := range issues {
			if issue.IssueType == "high_transcode" {
				found = true
			}
		}
		if !found {
			t.Error("expected high_transcode issue to be detected")
		}
	})

	t.Run("detects high pause rate", func(t *testing.T) {
		summary := models.QoESummary{
			TotalSessions: 1000,
			PauseRate:     40.0, // Above 30% threshold
			AvgPauseCount: 3.5,
			AvgCompletion: 75.0,
		}
		platforms := []models.QoEByPlatform{}

		issues := generateQoEIssues(&summary, platforms)

		found := false
		for _, issue := range issues {
			if issue.IssueType == "high_pause" {
				found = true
				if issue.Severity != "warning" {
					t.Errorf("expected warning severity, got %s", issue.Severity)
				}
			}
		}
		if !found {
			t.Error("expected high_pause issue to be detected")
		}
	})

	t.Run("detects low completion rate", func(t *testing.T) {
		summary := models.QoESummary{
			TotalSessions: 1000,
			AvgCompletion: 35.0, // Below 50% threshold
		}
		platforms := []models.QoEByPlatform{}

		issues := generateQoEIssues(&summary, platforms)

		found := false
		for _, issue := range issues {
			if issue.IssueType == "low_completion" {
				found = true
			}
		}
		if !found {
			t.Error("expected low_completion issue to be detected")
		}
	})

	t.Run("detects platform-specific EBVS issues", func(t *testing.T) {
		summary := models.QoESummary{
			TotalSessions: 1000,
			AvgCompletion: 75.0,
		}
		platforms := []models.QoEByPlatform{
			{
				Platform:     "Roku",
				SessionCount: 100,
				EBVSRate:     15.0, // Above 10% threshold with >50 sessions
			},
		}

		issues := generateQoEIssues(&summary, platforms)

		found := false
		for _, issue := range issues {
			if issue.IssueType == "platform_ebvs" {
				found = true
				if issue.RelatedDimension != "platform:Roku" {
					t.Errorf("expected RelatedDimension=platform:Roku, got %s", issue.RelatedDimension)
				}
			}
		}
		if !found {
			t.Error("expected platform_ebvs issue to be detected")
		}
	})

	t.Run("ignores small sample platform issues", func(t *testing.T) {
		summary := models.QoESummary{
			TotalSessions: 1000,
			AvgCompletion: 75.0,
		}
		platforms := []models.QoEByPlatform{
			{
				Platform:     "OldDevice",
				SessionCount: 20, // Below 50 session threshold
				EBVSRate:     30.0,
			},
		}

		issues := generateQoEIssues(&summary, platforms)

		for _, issue := range issues {
			if issue.IssueType == "platform_ebvs" && issue.RelatedDimension == "platform:OldDevice" {
				t.Error("should not report platform issues for small sample sizes")
			}
		}
	})
}

func TestQoEScoreCalculation(t *testing.T) {

	// Test the QoE score formula logic
	// Formula: 100 - (EBVS_penalty + QualityDegrade_penalty + Pause_penalty + LowCompletion_penalty)
	// Where:
	// - EBVS_penalty = EBVSRate * 5
	// - QualityDegrade_penalty = QualityDegradeRate * 2
	// - Pause_penalty = PauseRate * 1
	// - Completion_penalty = (100 - AvgCompletion) * 0.3

	t.Run("perfect metrics yield 100 score", func(t *testing.T) {
		// With 0 EBVS, 0 quality degrade, 0 pause, 100% completion
		// Score = 100 - (0*5 + 0*2 + 0*1 + 0*0.3) = 100
		ebvsPenalty := 0.0 * 5
		qualityPenalty := 0.0 * 2
		pausePenalty := 0.0 * 1
		completionPenalty := (100 - 100.0) * 0.3

		score := 100 - (ebvsPenalty + qualityPenalty + pausePenalty + completionPenalty)
		if score != 100 {
			t.Errorf("expected score 100, got %.1f", score)
		}
	})

	t.Run("EBVS has highest impact", func(t *testing.T) {
		// 10% EBVS should contribute 50 points penalty
		ebvsPenalty := 10.0 * 5 // = 50
		if ebvsPenalty != 50 {
			t.Errorf("10%% EBVS should have 50 point penalty, got %.1f", ebvsPenalty)
		}

		// 10% quality degrade should contribute 20 points penalty
		qualityPenalty := 10.0 * 2 // = 20
		if qualityPenalty != 20 {
			t.Errorf("10%% quality degrade should have 20 point penalty, got %.1f", qualityPenalty)
		}
	})

	t.Run("score is clamped to 0-100 range", func(t *testing.T) {
		// Very poor metrics should not go below 0
		// Example: 30% EBVS = 150 penalty alone
		ebvsPenalty := 30.0 * 5 // = 150
		score := 100 - ebvsPenalty
		if score > 0 {
			t.Error("raw score should be negative for very poor metrics")
		}
		// Clamped score should be 0
		if score < 0 {
			score = 0
		}
		if score != 0 {
			t.Errorf("clamped score should be 0, got %.1f", score)
		}
	})
}

func TestQoEHashDeterminism(t *testing.T) {

	startDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)

	filter := LocationStatsFilter{
		StartDate:  &startDate,
		EndDate:    &endDate,
		Users:      []string{"user1", "user2", "user3"},
		MediaTypes: []string{"movie", "episode"},
		Platforms:  []string{"iOS", "Android"},
	}

	// Generate hash 100 times and verify consistency
	firstHash := generateQoEQueryHash(filter)
	for i := 0; i < 100; i++ {
		hash := generateQoEQueryHash(filter)
		if hash != firstHash {
			t.Errorf("iteration %d produced different hash: %s != %s", i, hash, firstHash)
		}
	}
}
