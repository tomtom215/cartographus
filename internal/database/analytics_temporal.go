// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// GetTemporalHeatmap generates time-series geographic heatmap data with configurable time intervals
// (hour, day, week, month) for temporal animation and playback pattern visualization.
func (db *DB) GetTemporalHeatmap(ctx context.Context, filter LocationStatsFilter, interval string) (*models.TemporalHeatmapResponse, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Validate interval and build SQL
	bucketSQL, err := buildTemporalBucketSQL(interval)
	if err != nil {
		return nil, err
	}

	// Query and scan temporal heatmap data
	bucketMap, bucketCounts, minTime, maxTime, err := db.queryTemporalHeatmapData(ctx, filter, bucketSQL)
	if err != nil {
		return nil, err
	}

	// Convert map to sorted buckets array
	buckets, totalCount := sortAndConstructBuckets(bucketMap, bucketCounts, interval)

	// Fill in missing buckets for smoother animation
	if len(buckets) > 0 {
		buckets = fillMissingBuckets(buckets, interval)
	}

	return &models.TemporalHeatmapResponse{
		Interval:   interval,
		Buckets:    buckets,
		TotalCount: totalCount,
		StartDate:  minTime,
		EndDate:    maxTime,
	}, nil
}

// buildTemporalBucketSQL validates the interval and returns the corresponding DuckDB DATE_TRUNC expression
// for time bucketing. Supported intervals: hour, day, week, month.
func buildTemporalBucketSQL(interval string) (string, error) {
	switch interval {
	case "hour":
		return "DATE_TRUNC('hour', p.started_at)", nil
	case "day":
		return "DATE_TRUNC('day', p.started_at)", nil
	case "week":
		return "DATE_TRUNC('week', p.started_at)", nil
	case "month":
		return "DATE_TRUNC('month', p.started_at)", nil
	default:
		return "", fmt.Errorf("invalid interval: must be hour, day, week, or month")
	}
}

// queryTemporalHeatmapData executes the temporal heatmap query and scans results into maps
// tracking points by time bucket, along with min/max timestamps for the time range.
func (db *DB) queryTemporalHeatmapData(ctx context.Context, filter LocationStatsFilter, bucketSQL string) (map[time.Time][]models.TemporalHeatmapPoint, map[time.Time]int, time.Time, time.Time, error) {
	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " AND " + join(whereClauses, " AND ")
	}

	query := fmt.Sprintf(`
	SELECT
		%s as time_bucket,
		g.latitude,
		g.longitude,
		COUNT(*) as weight
	FROM playback_events p
	JOIN geolocations g ON p.ip_address = g.ip_address
	WHERE 1=1%s
	GROUP BY time_bucket, g.latitude, g.longitude
	ORDER BY time_bucket, weight DESC`, bucketSQL, whereSQL)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, time.Time{}, time.Time{}, fmt.Errorf("failed to query temporal heatmap: %w", err)
	}
	defer rows.Close()

	bucketMap := make(map[time.Time][]models.TemporalHeatmapPoint)
	bucketCounts := make(map[time.Time]int)
	var minTime, maxTime time.Time

	for rows.Next() {
		var bucketTime time.Time
		var lat, lon float64
		var weight int

		if err := rows.Scan(&bucketTime, &lat, &lon, &weight); err != nil {
			return nil, nil, time.Time{}, time.Time{}, fmt.Errorf("failed to scan temporal heatmap row: %w", err)
		}

		if minTime.IsZero() || bucketTime.Before(minTime) {
			minTime = bucketTime
		}
		if maxTime.IsZero() || bucketTime.After(maxTime) {
			maxTime = bucketTime
		}

		bucketMap[bucketTime] = append(bucketMap[bucketTime], models.TemporalHeatmapPoint{
			Latitude:  lat,
			Longitude: lon,
			Weight:    weight,
		})
		bucketCounts[bucketTime] += weight
	}

	if err = rows.Err(); err != nil {
		return nil, nil, time.Time{}, time.Time{}, fmt.Errorf("error iterating temporal heatmap rows: %w", err)
	}

	return bucketMap, bucketCounts, minTime, maxTime, nil
}

// sortAndConstructBuckets sorts time buckets chronologically and constructs the final bucket array
// with calculated end times, labels, and aggregated counts.
func sortAndConstructBuckets(bucketMap map[time.Time][]models.TemporalHeatmapPoint, bucketCounts map[time.Time]int, interval string) ([]models.TemporalHeatmapBucket, int) {
	var bucketTimes []time.Time
	for key := range bucketMap {
		bucketTimes = append(bucketTimes, key)
	}

	sort.Slice(bucketTimes, func(i, j int) bool {
		return bucketTimes[i].Before(bucketTimes[j])
	})

	var buckets []models.TemporalHeatmapBucket
	totalCount := 0

	for _, bucketTime := range bucketTimes {
		endTime := calculateBucketEndTime(bucketTime, interval)
		label := generateBucketLabel(bucketTime, interval)

		bucket := models.TemporalHeatmapBucket{
			StartTime: bucketTime,
			EndTime:   endTime,
			Label:     label,
			Points:    bucketMap[bucketTime],
			Count:     bucketCounts[bucketTime],
		}
		buckets = append(buckets, bucket)
		totalCount += bucketCounts[bucketTime]
	}

	return buckets, totalCount
}

// calculateBucketEndTime computes the end time for a bucket based on the interval
// (hour adds 1 hour, day adds 24 hours, week adds 7 days, month adds 1 month).
func calculateBucketEndTime(startTime time.Time, interval string) time.Time {
	switch interval {
	case "hour":
		return startTime.Add(time.Hour)
	case "day":
		return startTime.Add(24 * time.Hour)
	case "week":
		return startTime.Add(7 * 24 * time.Hour)
	case "month":
		return startTime.AddDate(0, 1, 0)
	default:
		return startTime
	}
}

// generateBucketLabel creates human-readable labels for time buckets
func generateBucketLabel(t time.Time, interval string) string {
	switch interval {
	case "hour":
		return t.Format("Mon 3PM")
	case "day":
		return t.Format("Jan 2")
	case "week":
		return fmt.Sprintf("Week of %s", t.Format("Jan 2"))
	case "month":
		return t.Format("Jan 2006")
	default:
		return t.Format(time.RFC3339)
	}
}

// fillMissingBuckets fills gaps in time series for smoother animations by inserting
// empty buckets between discontinuous time periods.
func fillMissingBuckets(buckets []models.TemporalHeatmapBucket, interval string) []models.TemporalHeatmapBucket {
	if len(buckets) < 2 {
		return buckets
	}

	result := []models.TemporalHeatmapBucket{buckets[0]}

	for i := 1; i < len(buckets); i++ {
		prev := result[len(result)-1]
		curr := buckets[i]

		expectedNext := calculateBucketEndTime(prev.StartTime, interval)

		// Fill gaps with empty buckets
		for !expectedNext.Equal(curr.StartTime) && expectedNext.Before(curr.StartTime) {
			emptyBucket := models.TemporalHeatmapBucket{
				StartTime: expectedNext,
				EndTime:   calculateBucketEndTime(expectedNext, interval),
				Label:     generateBucketLabel(expectedNext, interval),
				Points:    []models.TemporalHeatmapPoint{},
				Count:     0,
			}
			result = append(result, emptyBucket)
			expectedNext = calculateBucketEndTime(expectedNext, interval)
		}

		result = append(result, curr)
	}

	return result
}
