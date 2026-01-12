// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"fmt"

	"github.com/tomtom215/cartographus/internal/models"
)

// GetHDRAnalytics returns HDR and dynamic range analytics including format distribution,
// HDR adoption rate, tone mapping events, HDR-capable devices, and content statistics by format.
func (db *DB) GetHDRAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.HDRAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereClause := buildWhereClause(whereClauses)

	// Get total playbacks
	total, err := db.getTotalPlaybacks(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get total playbacks", err)
	}

	// Return empty analytics if no data
	if total == 0 {
		return &models.HDRAnalytics{
			TotalPlaybacks:     0,
			HDRAdoptionRate:    0,
			FormatDistribution: []models.DynamicRangeDistribution{},
			ToneMappingEvents:  []models.ToneMappingEvent{},
		}, nil
	}

	// Get format distribution with HDR adoption rate
	formatDist, hdrAdoptionRate, err := db.getHDRFormatDistribution(ctx, whereClause, args, total)
	if err != nil {
		return nil, errorContext("get format distribution", err)
	}

	// Get tone mapping events
	toneMappingEvents, err := db.getToneMappingEvents(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get tone mapping events", err)
	}

	// Get HDR capable devices
	hdrDevices, err := db.getHDRCapableDevices(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get HDR capable devices", err)
	}

	// Get content by format
	contentByFormat, err := db.getContentByFormat(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get content by format", err)
	}

	return &models.HDRAnalytics{
		TotalPlaybacks:     total,
		FormatDistribution: formatDist,
		HDRAdoptionRate:    hdrAdoptionRate,
		ToneMappingEvents:  toneMappingEvents,
		HDRCapableDevices:  hdrDevices,
		ContentByFormat:    contentByFormat,
	}, nil
}

// getHDRFormatDistribution retrieves HDR format distribution and calculates HDR adoption rate
func (db *DB) getHDRFormatDistribution(ctx context.Context, whereClause string, args []interface{}, total int) ([]models.DynamicRangeDistribution, float64, error) {
	query := fmt.Sprintf(`
		SELECT
			COALESCE(video_dynamic_range, 'SDR') as dynamic_range,
			COUNT(*) as playback_count,
			(COUNT(*) * 100.0 / %d) as percentage,
			COUNT(DISTINCT username) as unique_users
		FROM playback_events
		%s
		GROUP BY video_dynamic_range
		ORDER BY playback_count DESC
	`, total, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query format distribution: %w", err)
	}
	defer rows.Close()

	var formatDist []models.DynamicRangeDistribution
	hdrCount := 0
	for rows.Next() {
		var d models.DynamicRangeDistribution
		err := rows.Scan(&d.DynamicRange, &d.PlaybackCount, &d.Percentage, &d.UniqueUsers)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan format row: %w", err)
		}
		formatDist = append(formatDist, d)
		if d.DynamicRange != "SDR" && d.DynamicRange != "" {
			hdrCount += d.PlaybackCount
		}
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating rows: %w", err)
	}

	hdrAdoptionRate := calculateAdoptionRate(hdrCount, total)

	return formatDist, hdrAdoptionRate, nil
}

// getToneMappingEvents retrieves tone mapping events where HDR content is transcoded to SDR
func (db *DB) getToneMappingEvents(ctx context.Context, whereClause string, args []interface{}) ([]models.ToneMappingEvent, error) {
	toneMappingCondition := "video_dynamic_range IS NOT NULL AND video_dynamic_range != 'SDR' AND (stream_video_decision = 'transcode' OR transcode_decision = 'transcode')"
	toneMappingWhere := appendWhereCondition(whereClause, toneMappingCondition)

	query := fmt.Sprintf(`
		SELECT
			video_dynamic_range as source_format,
			'SDR' as stream_format,
			COUNT(*) as occurrence_count,
			string_agg(DISTINCT username, ',') as users,
			string_agg(DISTINCT platform, ',') as platforms
		FROM playback_events
		%s
		GROUP BY video_dynamic_range
		ORDER BY occurrence_count DESC
	`, toneMappingWhere)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tone mapping: %w", err)
	}
	defer rows.Close()

	var toneMappingEvents []models.ToneMappingEvent
	for rows.Next() {
		var tm models.ToneMappingEvent
		var usersStr, platformsStr string
		err := rows.Scan(&tm.SourceFormat, &tm.StreamFormat, &tm.OccurrenceCount, &usersStr, &platformsStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tone mapping row: %w", err)
		}
		tm.AffectedUsers = parseAggregatedList(usersStr)
		tm.Platforms = parseAggregatedList(platformsStr)
		toneMappingEvents = append(toneMappingEvents, tm)
	}

	return toneMappingEvents, rows.Err()
}

// getHDRCapableDevices retrieves platform HDR capabilities and usage statistics
func (db *DB) getHDRCapableDevices(ctx context.Context, whereClause string, args []interface{}) ([]models.HDRDeviceStats, error) {
	query := fmt.Sprintf(`
		SELECT
			platform,
			SUM(CASE WHEN video_dynamic_range != 'SDR' AND video_dynamic_range IS NOT NULL THEN 1 ELSE 0 END) as hdr_playbacks,
			SUM(CASE WHEN video_dynamic_range = 'SDR' OR video_dynamic_range IS NULL THEN 1 ELSE 0 END) as sdr_playbacks,
			BOOL_OR(video_dynamic_range != 'SDR' AND video_dynamic_range IS NOT NULL) as hdr_capable
		FROM playback_events
		%s
		GROUP BY platform
		ORDER BY hdr_playbacks DESC
		LIMIT 15
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query HDR devices: %w", err)
	}
	defer rows.Close()

	var hdrDevices []models.HDRDeviceStats
	for rows.Next() {
		var d models.HDRDeviceStats
		err := rows.Scan(&d.Platform, &d.HDRPlaybacks, &d.SDRPlaybacks, &d.HDRCapable)
		if err != nil {
			return nil, fmt.Errorf("failed to scan device row: %w", err)
		}
		hdrDevices = append(hdrDevices, d)
	}

	return hdrDevices, rows.Err()
}

// getContentByFormat retrieves content statistics grouped by dynamic range format
func (db *DB) getContentByFormat(ctx context.Context, whereClause string, args []interface{}) ([]models.ContentFormatStats, error) {
	query := fmt.Sprintf(`
		WITH title_counts AS (
			SELECT
				COALESCE(video_dynamic_range, 'SDR') as dynamic_range,
				title,
				COUNT(*) as title_count,
				ROW_NUMBER() OVER (PARTITION BY COALESCE(video_dynamic_range, 'SDR') ORDER BY COUNT(*) DESC) as rn
			FROM playback_events
			%s
			GROUP BY COALESCE(video_dynamic_range, 'SDR'), title
		),
		format_stats AS (
			SELECT
				COALESCE(video_dynamic_range, 'SDR') as dynamic_range,
				COUNT(DISTINCT title) as unique_content,
				AVG(percent_complete) as avg_completion
			FROM playback_events
			%s
			GROUP BY video_dynamic_range
		)
		SELECT
			fs.dynamic_range,
			fs.unique_content,
			fs.avg_completion,
			COALESCE(tc.title, '') as most_watched
		FROM format_stats fs
		LEFT JOIN title_counts tc ON fs.dynamic_range = tc.dynamic_range AND tc.rn = 1
		ORDER BY fs.unique_content DESC
	`, whereClause, whereClause)

	// Need to pass args twice since we use the WHERE clause twice in the CTE
	contentArgs := append(args, args...)
	rows, err := db.conn.QueryContext(ctx, query, contentArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query content by format: %w", err)
	}
	defer rows.Close()

	var contentByFormat []models.ContentFormatStats
	for rows.Next() {
		var c models.ContentFormatStats
		err := rows.Scan(&c.DynamicRange, &c.UniqueContent, &c.AvgCompletion, &c.MostWatchedTitle)
		if err != nil {
			return nil, fmt.Errorf("failed to scan content row: %w", err)
		}
		contentByFormat = append(contentByFormat, c)
	}

	return contentByFormat, rows.Err()
}
