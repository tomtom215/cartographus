// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package models provides data structures for the Cartographus application.
// This file contains cohort retention analytics models for user engagement tracking.
package models

import "time"

// CohortRetentionAnalytics represents comprehensive cohort retention analysis
// following industry best practices from Mixpanel, Amplitude, and similar platforms.
type CohortRetentionAnalytics struct {
	// Cohorts contains all cohort data with retention percentages
	Cohorts []CohortData `json:"cohorts"`

	// Summary provides aggregate retention statistics
	Summary CohortRetentionSummary `json:"summary"`

	// RetentionCurve provides week-by-week average retention for visualization
	RetentionCurve []RetentionPoint `json:"retention_curve"`

	// Metadata provides query provenance information
	Metadata CohortQueryMetadata `json:"metadata"`
}

// CohortData represents a single cohort (users who started in a specific period)
type CohortData struct {
	// CohortWeek is the ISO week when this cohort first appeared (YYYY-Www format)
	CohortWeek string `json:"cohort_week"`

	// CohortStartDate is the first day of the cohort week
	CohortStartDate time.Time `json:"cohort_start_date"`

	// InitialUsers is the count of unique users who first played in this week
	InitialUsers int `json:"initial_users"`

	// Retention is a map of week offsets (0, 1, 2, ...) to retention data
	// Week 0 is always 100% (the cohort definition week)
	Retention []WeekRetention `json:"retention"`

	// AverageRetention is the mean retention rate across all tracked weeks (excluding week 0)
	AverageRetention float64 `json:"average_retention"`

	// ChurnRate is 100 - AverageRetention
	ChurnRate float64 `json:"churn_rate"`
}

// WeekRetention represents retention data for a specific week offset
type WeekRetention struct {
	// WeekOffset is the number of weeks since cohort formation (0 = same week)
	WeekOffset int `json:"week_offset"`

	// ActiveUsers is the count of users from this cohort active in this week
	ActiveUsers int `json:"active_users"`

	// RetentionRate is (ActiveUsers / InitialUsers) * 100
	RetentionRate float64 `json:"retention_rate"`

	// WeekDate is the actual date of this retention week
	WeekDate time.Time `json:"week_date"`
}

// CohortRetentionSummary provides aggregate statistics across all cohorts
type CohortRetentionSummary struct {
	// TotalCohorts is the number of weekly cohorts analyzed
	TotalCohorts int `json:"total_cohorts"`

	// TotalUsersTracked is the sum of all initial cohort users
	TotalUsersTracked int `json:"total_users_tracked"`

	// Week1Retention is the average retention rate at week 1 across all cohorts
	Week1Retention float64 `json:"week1_retention"`

	// Week4Retention is the average retention rate at week 4 across all cohorts
	Week4Retention float64 `json:"week4_retention"`

	// Week12Retention is the average retention rate at week 12 (3 months) across all cohorts
	Week12Retention float64 `json:"week12_retention"`

	// MedianRetentionWeek1 is the median retention at week 1 (more robust than mean)
	MedianRetentionWeek1 float64 `json:"median_retention_week1"`

	// BestPerformingCohort is the cohort week with highest average retention
	BestPerformingCohort string `json:"best_performing_cohort"`

	// WorstPerformingCohort is the cohort week with lowest average retention
	WorstPerformingCohort string `json:"worst_performing_cohort"`

	// OverallAverageRetention is the average retention rate across all cohorts and weeks
	OverallAverageRetention float64 `json:"overall_average_retention"`

	// RetentionTrend indicates if retention is "improving", "declining", or "stable"
	RetentionTrend string `json:"retention_trend"`
}

// RetentionPoint represents a single point on the aggregate retention curve
type RetentionPoint struct {
	// WeekOffset is the number of weeks since cohort formation
	WeekOffset int `json:"week_offset"`

	// AverageRetention is the mean retention rate across all cohorts at this week
	AverageRetention float64 `json:"average_retention"`

	// MedianRetention is the median retention rate (more robust to outliers)
	MedianRetention float64 `json:"median_retention"`

	// MinRetention is the lowest retention rate among cohorts at this week
	MinRetention float64 `json:"min_retention"`

	// MaxRetention is the highest retention rate among cohorts at this week
	MaxRetention float64 `json:"max_retention"`

	// CohortsWithData is the number of cohorts that have data for this week offset
	CohortsWithData int `json:"cohorts_with_data"`
}

// CohortQueryMetadata provides provenance and auditability information
type CohortQueryMetadata struct {
	// QueryHash is a deterministic hash of the query parameters for reproducibility
	QueryHash string `json:"query_hash"`

	// DataRangeStart is the earliest data point used
	DataRangeStart time.Time `json:"data_range_start"`

	// DataRangeEnd is the latest data point used
	DataRangeEnd time.Time `json:"data_range_end"`

	// CohortGranularity is the cohort grouping period ("week" or "month")
	CohortGranularity string `json:"cohort_granularity"`

	// MaxWeeksTracked is the maximum number of weeks tracked per cohort
	MaxWeeksTracked int `json:"max_weeks_tracked"`

	// EventCount is the total number of playback events analyzed
	EventCount int64 `json:"event_count"`

	// GeneratedAt is when this analysis was generated
	GeneratedAt time.Time `json:"generated_at"`

	// QueryTimeMs is how long the query took to execute
	QueryTimeMs int64 `json:"query_time_ms"`

	// Cached indicates if this result was served from cache
	Cached bool `json:"cached"`
}
