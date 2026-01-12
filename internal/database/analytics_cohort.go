// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package database provides data access and analytics functionality for the Cartographus application.
// This file contains cohort retention analytics following industry best practices from
// Mixpanel, Amplitude, and similar product analytics platforms.
//
// Cohort retention analysis helps understand:
// - When users typically stop using the service (churn points)
// - Which cohorts have best/worst retention (identify successful periods)
// - Overall user engagement health over time
//
// The implementation uses DuckDB window functions for efficient cohort calculations.
package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// CohortRetentionConfig configures cohort analysis parameters
type CohortRetentionConfig struct {
	// MaxWeeks is the maximum number of weeks to track per cohort (default: 12)
	MaxWeeks int

	// MinCohortSize is the minimum users required to include a cohort (default: 3)
	MinCohortSize int

	// Granularity is "week" or "month" (default: "week")
	Granularity string
}

// DefaultCohortConfig returns sensible default configuration
func DefaultCohortConfig() CohortRetentionConfig {
	return CohortRetentionConfig{
		MaxWeeks:      12,
		MinCohortSize: 3,
		Granularity:   "week",
	}
}

// GetCohortRetentionAnalytics calculates cohort retention metrics
func (db *DB) GetCohortRetentionAnalytics(ctx context.Context, filter LocationStatsFilter, config CohortRetentionConfig) (*models.CohortRetentionAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	startTime := time.Now()

	// Apply defaults for zero values
	if config.MaxWeeks == 0 {
		config.MaxWeeks = 12
	}
	if config.MinCohortSize == 0 {
		config.MinCohortSize = 3
	}
	if config.Granularity == "" {
		config.Granularity = "week"
	}

	// Build filter conditions
	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereClause := "1=1"
	if len(whereClauses) > 0 {
		whereClause = join(whereClauses, " AND ")
	}

	// Generate deterministic query hash for reproducibility
	queryHash := generateCohortQueryHash(filter, config)

	// Execute cohort query
	cohortData, eventCount, err := db.executeCohortQuery(ctx, whereClause, args, config)
	if err != nil {
		return nil, fmt.Errorf("failed to execute cohort query: %w", err)
	}

	// Calculate summary statistics
	summary := calculateCohortSummary(cohortData)

	// Build retention curve
	retentionCurve := buildRetentionCurve(cohortData, config.MaxWeeks)

	// Determine data range
	dataRangeStart, dataRangeEnd := getDataRange(filter)

	return &models.CohortRetentionAnalytics{
		Cohorts:        cohortData,
		Summary:        summary,
		RetentionCurve: retentionCurve,
		Metadata: models.CohortQueryMetadata{
			QueryHash:         queryHash,
			DataRangeStart:    dataRangeStart,
			DataRangeEnd:      dataRangeEnd,
			CohortGranularity: config.Granularity,
			MaxWeeksTracked:   config.MaxWeeks,
			EventCount:        eventCount,
			GeneratedAt:       time.Now(),
			QueryTimeMs:       time.Since(startTime).Milliseconds(),
			Cached:            false,
		},
	}, nil
}

// executeCohortQuery runs the cohort retention SQL query
func (db *DB) executeCohortQuery(ctx context.Context, whereClause string, args []interface{}, config CohortRetentionConfig) ([]models.CohortData, int64, error) {
	// Step 1: Find each user's first activity (cohort assignment)
	// Step 2: For each cohort, count active users per week offset
	// Step 3: Calculate retention rates
	query := fmt.Sprintf(`
		WITH user_first_activity AS (
			-- Assign each user to their cohort based on first activity
			SELECT
				user_id,
				DATE_TRUNC('week', MIN(started_at)) AS cohort_week
			FROM playback_events
			WHERE %s
			GROUP BY user_id
		),
		user_weekly_activity AS (
			-- Get all weeks where each user was active
			SELECT DISTINCT
				user_id,
				DATE_TRUNC('week', started_at) AS activity_week
			FROM playback_events
			WHERE %s
		),
		cohort_retention AS (
			-- Join to calculate week offset and retention
			SELECT
				ufa.cohort_week,
				DATEDIFF('week', ufa.cohort_week, uwa.activity_week) AS week_offset,
				COUNT(DISTINCT uwa.user_id) AS active_users
			FROM user_first_activity ufa
			JOIN user_weekly_activity uwa ON ufa.user_id = uwa.user_id
			WHERE DATEDIFF('week', ufa.cohort_week, uwa.activity_week) >= 0
				AND DATEDIFF('week', ufa.cohort_week, uwa.activity_week) <= ?
			GROUP BY ufa.cohort_week, week_offset
		),
		cohort_sizes AS (
			-- Get initial size of each cohort
			SELECT
				cohort_week,
				COUNT(DISTINCT user_id) AS initial_users
			FROM user_first_activity
			GROUP BY cohort_week
			HAVING COUNT(DISTINCT user_id) >= ?
		),
		event_count AS (
			SELECT COUNT(*) AS total FROM playback_events WHERE %s
		)
		SELECT
			cs.cohort_week,
			cs.initial_users,
			cr.week_offset,
			cr.active_users,
			(SELECT total FROM event_count) AS event_count
		FROM cohort_sizes cs
		JOIN cohort_retention cr ON cs.cohort_week = cr.cohort_week
		ORDER BY cs.cohort_week, cr.week_offset
	`, whereClause, whereClause, whereClause)

	// Build full args (whereClause used 3 times, plus maxWeeks and minCohortSize)
	fullArgs := append([]interface{}{}, args...)
	fullArgs = append(fullArgs, args...)
	fullArgs = append(fullArgs, config.MaxWeeks)
	fullArgs = append(fullArgs, config.MinCohortSize)
	fullArgs = append(fullArgs, args...)

	rows, err := db.conn.QueryContext(ctx, query, fullArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query cohort data: %w", err)
	}
	defer rows.Close()

	// Parse results into cohort structure
	cohortMap := make(map[string]*models.CohortData)
	var eventCount int64

	for rows.Next() {
		var cohortWeek time.Time
		var initialUsers, weekOffset, activeUsers int
		var evtCount int64

		if err := rows.Scan(&cohortWeek, &initialUsers, &weekOffset, &activeUsers, &evtCount); err != nil {
			return nil, 0, fmt.Errorf("scan cohort row: %w", err)
		}

		eventCount = evtCount
		cohortKey := cohortWeek.Format("2006-W02")

		if _, exists := cohortMap[cohortKey]; !exists {
			cohortMap[cohortKey] = &models.CohortData{
				CohortWeek:      cohortKey,
				CohortStartDate: cohortWeek,
				InitialUsers:    initialUsers,
				Retention:       make([]models.WeekRetention, 0, config.MaxWeeks+1),
			}
		}

		retentionRate := 0.0
		if initialUsers > 0 {
			retentionRate = float64(activeUsers) / float64(initialUsers) * 100.0
		}

		cohortMap[cohortKey].Retention = append(cohortMap[cohortKey].Retention, models.WeekRetention{
			WeekOffset:    weekOffset,
			ActiveUsers:   activeUsers,
			RetentionRate: retentionRate,
			WeekDate:      cohortWeek.AddDate(0, 0, weekOffset*7),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate cohort rows: %w", err)
	}

	// Convert map to sorted slice
	cohorts := make([]models.CohortData, 0, len(cohortMap))
	for _, cohort := range cohortMap {
		// Calculate average retention (excluding week 0)
		var totalRetention float64
		var retentionPoints int
		for _, r := range cohort.Retention {
			if r.WeekOffset > 0 {
				totalRetention += r.RetentionRate
				retentionPoints++
			}
		}
		if retentionPoints > 0 {
			cohort.AverageRetention = totalRetention / float64(retentionPoints)
			cohort.ChurnRate = 100.0 - cohort.AverageRetention
		}
		cohorts = append(cohorts, *cohort)
	}

	// Sort by cohort week
	sort.Slice(cohorts, func(i, j int) bool {
		return cohorts[i].CohortStartDate.Before(cohorts[j].CohortStartDate)
	})

	return cohorts, eventCount, nil
}

// calculateCohortSummary computes aggregate statistics across all cohorts
func calculateCohortSummary(cohorts []models.CohortData) models.CohortRetentionSummary {
	summary := models.CohortRetentionSummary{
		TotalCohorts: len(cohorts),
	}

	if len(cohorts) == 0 {
		summary.RetentionTrend = "insufficient_data"
		return summary
	}

	var week1Rates, week4Rates, week12Rates []float64
	var allRetentionRates []float64
	var bestRetention, worstRetention float64 = 0, 100
	var bestCohort, worstCohort string

	for _, cohort := range cohorts {
		summary.TotalUsersTracked += cohort.InitialUsers

		for _, r := range cohort.Retention {
			if r.WeekOffset == 1 {
				week1Rates = append(week1Rates, r.RetentionRate)
			}
			if r.WeekOffset == 4 {
				week4Rates = append(week4Rates, r.RetentionRate)
			}
			if r.WeekOffset == 12 {
				week12Rates = append(week12Rates, r.RetentionRate)
			}
			if r.WeekOffset > 0 {
				allRetentionRates = append(allRetentionRates, r.RetentionRate)
			}
		}

		if cohort.AverageRetention > bestRetention {
			bestRetention = cohort.AverageRetention
			bestCohort = cohort.CohortWeek
		}
		if cohort.AverageRetention < worstRetention {
			worstRetention = cohort.AverageRetention
			worstCohort = cohort.CohortWeek
		}
	}

	// Calculate averages
	summary.Week1Retention = average(week1Rates)
	summary.Week4Retention = average(week4Rates)
	summary.Week12Retention = average(week12Rates)
	summary.MedianRetentionWeek1 = median(week1Rates)
	summary.OverallAverageRetention = average(allRetentionRates)
	summary.BestPerformingCohort = bestCohort
	summary.WorstPerformingCohort = worstCohort

	// Determine trend (compare first half to second half of cohorts)
	summary.RetentionTrend = calculateRetentionTrend(cohorts)

	return summary
}

// buildRetentionCurve creates aggregate retention curve data
func buildRetentionCurve(cohorts []models.CohortData, maxWeeks int) []models.RetentionPoint {
	if len(cohorts) == 0 {
		return []models.RetentionPoint{}
	}

	// Collect retention rates per week offset
	weekData := make(map[int][]float64)
	for _, cohort := range cohorts {
		for _, r := range cohort.Retention {
			if r.WeekOffset <= maxWeeks {
				weekData[r.WeekOffset] = append(weekData[r.WeekOffset], r.RetentionRate)
			}
		}
	}

	// Build curve points
	curve := make([]models.RetentionPoint, 0, maxWeeks+1)
	for week := 0; week <= maxWeeks; week++ {
		rates := weekData[week]
		if len(rates) == 0 {
			continue
		}

		point := models.RetentionPoint{
			WeekOffset:       week,
			AverageRetention: average(rates),
			MedianRetention:  median(rates),
			MinRetention:     minFloat(rates),
			MaxRetention:     maxFloat(rates),
			CohortsWithData:  len(rates),
		}
		curve = append(curve, point)
	}

	return curve
}

// calculateRetentionTrend compares recent cohorts to older ones
func calculateRetentionTrend(cohorts []models.CohortData) string {
	if len(cohorts) < 4 {
		return "insufficient_data"
	}

	midpoint := len(cohorts) / 2
	var earlyAvg, lateAvg float64
	var earlyCount, lateCount int

	for i, cohort := range cohorts {
		if i < midpoint {
			earlyAvg += cohort.AverageRetention
			earlyCount++
		} else {
			lateAvg += cohort.AverageRetention
			lateCount++
		}
	}

	if earlyCount > 0 {
		earlyAvg /= float64(earlyCount)
	}
	if lateCount > 0 {
		lateAvg /= float64(lateCount)
	}

	diff := lateAvg - earlyAvg
	if diff > 5 {
		return "improving"
	}
	if diff < -5 {
		return "declining"
	}
	return "stable"
}

// generateCohortQueryHash creates a deterministic hash for query reproducibility
func generateCohortQueryHash(filter LocationStatsFilter, config CohortRetentionConfig) string {
	// Create a canonical representation of the query parameters
	canonical := fmt.Sprintf("cohort|max_weeks=%d|min_size=%d|granularity=%s|",
		config.MaxWeeks, config.MinCohortSize, config.Granularity)

	if filter.StartDate != nil {
		canonical += fmt.Sprintf("start=%s|", filter.StartDate.Format(time.RFC3339))
	}
	if filter.EndDate != nil {
		canonical += fmt.Sprintf("end=%s|", filter.EndDate.Format(time.RFC3339))
	}
	if len(filter.Users) > 0 {
		canonical += fmt.Sprintf("users=%v|", filter.Users)
	}
	if len(filter.MediaTypes) > 0 {
		canonical += fmt.Sprintf("media_types=%v|", filter.MediaTypes)
	}

	hash := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(hash[:8]) // First 8 bytes = 16 hex chars
}

// getDataRange extracts the effective date range from filter
func getDataRange(filter LocationStatsFilter) (time.Time, time.Time) {
	start := time.Now().AddDate(-1, 0, 0) // Default: 1 year ago
	end := time.Now()

	if filter.StartDate != nil {
		start = *filter.StartDate
	}
	if filter.EndDate != nil {
		end = *filter.EndDate
	}

	return start, end
}

// Helper functions for statistics

func average(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func median(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func minFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	result := vals[0]
	for _, v := range vals[1:] {
		if v < result {
			result = v
		}
	}
	return result
}

func maxFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	result := vals[0]
	for _, v := range vals[1:] {
		if v > result {
			result = v
		}
	}
	return result
}

// Ensure the DB type satisfies the interface if needed
var _ sql.DB
