// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package database provides data access and analytics functionality for the Cartographus application.
// This file contains hardware transcode and HDR analytics implementation (v1.43 - API Coverage Audit).
package database

import (
	"context"
	"fmt"
)

// HardwareTranscodeStats represents aggregated hardware transcode statistics
type HardwareTranscodeStats struct {
	// Overview counts
	TotalSessions       int     `json:"total_sessions"`
	HWTranscodeSessions int     `json:"hw_transcode_sessions"`
	SWTranscodeSessions int     `json:"sw_transcode_sessions"`
	DirectPlaySessions  int     `json:"direct_play_sessions"`
	HWPercentage        float64 `json:"hw_percentage"`

	// Decoder statistics
	DecoderStats []HWCodecStats `json:"decoder_stats"`
	// Encoder statistics
	EncoderStats []HWCodecStats `json:"encoder_stats"`
	// Full pipeline (both HW decode and encode)
	FullPipelineCount int     `json:"full_pipeline_count"`
	FullPipelinePct   float64 `json:"full_pipeline_pct"`

	// Throttling statistics
	ThrottledSessions   int     `json:"throttled_sessions"`
	ThrottledPercentage float64 `json:"throttled_percentage"`
}

// HWCodecStats represents statistics for a specific hardware codec
type HWCodecStats struct {
	CodecName    string  `json:"codec_name"`  // e.g., "nvdec", "qsv", "vaapi"
	CodecTitle   string  `json:"codec_title"` // e.g., "NVIDIA NVDEC", "Intel Quick Sync"
	SessionCount int     `json:"session_count"`
	Percentage   float64 `json:"percentage"`
}

// HDRContentStats represents HDR content playback statistics
type HDRContentStats struct {
	// Overview counts
	TotalSessions int     `json:"total_sessions"`
	HDRSessions   int     `json:"hdr_sessions"`
	SDRSessions   int     `json:"sdr_sessions"`
	HDRPercentage float64 `json:"hdr_percentage"`

	// HDR format breakdown
	HDRFormats []HDRFormatStats `json:"hdr_formats"`

	// Color space distribution
	ColorSpaces []ColorSpaceStats `json:"color_spaces"`

	// Color primaries distribution
	ColorPrimaries []ColorPrimariesStats `json:"color_primaries"`
}

// HDRFormatStats represents statistics for a specific HDR format
type HDRFormatStats struct {
	Format       string  `json:"format"` // e.g., "HDR10", "HDR10+", "Dolby Vision", "HLG"
	SessionCount int     `json:"session_count"`
	Percentage   float64 `json:"percentage"`
}

// ColorSpaceStats represents statistics for a color space
type ColorSpaceStats struct {
	ColorSpace   string  `json:"color_space"` // e.g., "bt2020nc", "bt709"
	SessionCount int     `json:"session_count"`
	Percentage   float64 `json:"percentage"`
}

// ColorPrimariesStats represents statistics for color primaries
type ColorPrimariesStats struct {
	Primaries    string  `json:"primaries"` // e.g., "bt2020", "bt709"
	SessionCount int     `json:"session_count"`
	Percentage   float64 `json:"percentage"`
}

// GetHardwareTranscodeStats retrieves hardware transcode statistics
func (db *DB) GetHardwareTranscodeStats(ctx context.Context, filter LocationStatsFilter) (*HardwareTranscodeStats, error) {
	whereClause, args := buildFilterWhereClause(filter)

	stats := &HardwareTranscodeStats{}

	// Get total and HW transcode counts
	// Use COALESCE to handle empty database (SUM returns NULL when no rows)
	overviewQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_sessions,
			COALESCE(SUM(CASE WHEN transcode_hw_encoding = 1 OR transcode_hw_decoding = 1 THEN 1 ELSE 0 END), 0) as hw_sessions,
			COALESCE(SUM(CASE WHEN (transcode_hw_encoding IS NULL OR transcode_hw_encoding = 0)
			         AND (transcode_hw_decoding IS NULL OR transcode_hw_decoding = 0)
			         AND transcode_decision = 'transcode' THEN 1 ELSE 0 END), 0) as sw_sessions,
			COALESCE(SUM(CASE WHEN transcode_decision = 'direct play' OR transcode_decision IS NULL THEN 1 ELSE 0 END), 0) as direct_play,
			COALESCE(SUM(CASE WHEN transcode_hw_full_pipeline = 1 THEN 1 ELSE 0 END), 0) as full_pipeline,
			COALESCE(SUM(CASE WHEN transcode_throttled = 1 THEN 1 ELSE 0 END), 0) as throttled
		FROM playback_events
		WHERE %s
	`, whereClause)

	var totalSessions, hwSessions, swSessions, directPlay, fullPipeline, throttled int
	err := db.conn.QueryRowContext(ctx, overviewQuery, args...).Scan(
		&totalSessions, &hwSessions, &swSessions, &directPlay, &fullPipeline, &throttled,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query hardware transcode overview: %w", err)
	}

	stats.TotalSessions = totalSessions
	stats.HWTranscodeSessions = hwSessions
	stats.SWTranscodeSessions = swSessions
	stats.DirectPlaySessions = directPlay
	stats.FullPipelineCount = fullPipeline
	stats.ThrottledSessions = throttled

	if totalSessions > 0 {
		stats.HWPercentage = float64(hwSessions) / float64(totalSessions) * 100
		stats.FullPipelinePct = float64(fullPipeline) / float64(totalSessions) * 100
		stats.ThrottledPercentage = float64(throttled) / float64(totalSessions) * 100
	}

	// Get decoder statistics
	decoderQuery := fmt.Sprintf(`
		SELECT
			COALESCE(transcode_hw_decode, 'unknown') as codec,
			COALESCE(transcode_hw_decode_title, 'Unknown') as codec_title,
			COUNT(*) as session_count
		FROM playback_events
		WHERE %s AND transcode_hw_decoding = 1 AND transcode_hw_decode IS NOT NULL AND transcode_hw_decode != ''
		GROUP BY codec, codec_title
		ORDER BY session_count DESC
		LIMIT 10
	`, whereClause)

	decoderRows, err := db.conn.QueryContext(ctx, decoderQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query decoder stats: %w", err)
	}
	defer func() { _ = decoderRows.Close() }()

	for decoderRows.Next() {
		var codec, title string
		var count int
		if err := decoderRows.Scan(&codec, &title, &count); err != nil {
			return nil, fmt.Errorf("failed to scan decoder stats: %w", err)
		}
		pct := 0.0
		if hwSessions > 0 {
			pct = float64(count) / float64(hwSessions) * 100
		}
		stats.DecoderStats = append(stats.DecoderStats, HWCodecStats{
			CodecName:    codec,
			CodecTitle:   title,
			SessionCount: count,
			Percentage:   pct,
		})
	}
	if err := decoderRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating decoder stats: %w", err)
	}

	// Get encoder statistics
	encoderQuery := fmt.Sprintf(`
		SELECT
			COALESCE(transcode_hw_encode, 'unknown') as codec,
			COALESCE(transcode_hw_encode_title, 'Unknown') as codec_title,
			COUNT(*) as session_count
		FROM playback_events
		WHERE %s AND transcode_hw_encoding = 1 AND transcode_hw_encode IS NOT NULL AND transcode_hw_encode != ''
		GROUP BY codec, codec_title
		ORDER BY session_count DESC
		LIMIT 10
	`, whereClause)

	encoderRows, err := db.conn.QueryContext(ctx, encoderQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query encoder stats: %w", err)
	}
	defer func() { _ = encoderRows.Close() }()

	for encoderRows.Next() {
		var codec, title string
		var count int
		if err := encoderRows.Scan(&codec, &title, &count); err != nil {
			return nil, fmt.Errorf("failed to scan encoder stats: %w", err)
		}
		pct := 0.0
		if hwSessions > 0 {
			pct = float64(count) / float64(hwSessions) * 100
		}
		stats.EncoderStats = append(stats.EncoderStats, HWCodecStats{
			CodecName:    codec,
			CodecTitle:   title,
			SessionCount: count,
			Percentage:   pct,
		})
	}
	if err := encoderRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating encoder stats: %w", err)
	}

	return stats, nil
}

// GetHDRContentStats retrieves HDR content playback statistics
func (db *DB) GetHDRContentStats(ctx context.Context, filter LocationStatsFilter) (*HDRContentStats, error) {
	whereClause, args := buildFilterWhereClause(filter)

	stats := &HDRContentStats{}

	// Query overview counts
	totalSessions, hdrSessions, err := db.queryHDROverview(ctx, whereClause, args)
	if err != nil {
		return nil, err
	}

	stats.TotalSessions = totalSessions
	stats.HDRSessions = hdrSessions
	stats.SDRSessions = totalSessions - hdrSessions
	if totalSessions > 0 {
		stats.HDRPercentage = float64(hdrSessions) / float64(totalSessions) * 100
	}

	// Query HDR format breakdown
	stats.HDRFormats, err = db.queryHDRFormats(ctx, whereClause, args, totalSessions)
	if err != nil {
		return nil, err
	}

	// Query color space distribution
	stats.ColorSpaces, err = db.queryColorSpaces(ctx, whereClause, args, totalSessions)
	if err != nil {
		return nil, err
	}

	// Query color primaries distribution
	stats.ColorPrimaries, err = db.queryColorPrimaries(ctx, whereClause, args, totalSessions)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// queryHDROverview queries HDR vs SDR session counts
func (db *DB) queryHDROverview(ctx context.Context, whereClause string, args []interface{}) (int, int, error) {
	// Use COALESCE to handle empty database (SUM returns NULL when no rows)
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_sessions,
			COALESCE(SUM(CASE
				WHEN video_color_primaries = 'bt2020'
				     OR video_color_trc IN ('smpte2084', 'arib-std-b67')
				     OR video_dynamic_range IN ('HDR', 'HDR10', 'HDR10+', 'Dolby Vision', 'HLG')
				THEN 1 ELSE 0
			END), 0) as hdr_sessions
		FROM playback_events
		WHERE %s
	`, whereClause)

	var totalSessions, hdrSessions int
	err := db.conn.QueryRowContext(ctx, query, args...).Scan(&totalSessions, &hdrSessions)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query HDR overview: %w", err)
	}
	return totalSessions, hdrSessions, nil
}

// queryHDRFormats queries HDR format breakdown (HDR10, HDR10+, Dolby Vision, HLG)
func (db *DB) queryHDRFormats(ctx context.Context, whereClause string, args []interface{}, totalSessions int) ([]HDRFormatStats, error) {
	query := fmt.Sprintf(`
		SELECT
			CASE
				WHEN video_dynamic_range = 'Dolby Vision' OR video_dynamic_range = 'DV' THEN 'Dolby Vision'
				WHEN video_dynamic_range = 'HDR10+' THEN 'HDR10+'
				WHEN video_color_trc = 'smpte2084' THEN 'HDR10'
				WHEN video_color_trc = 'arib-std-b67' THEN 'HLG'
				WHEN video_dynamic_range = 'HDR' THEN 'HDR (Generic)'
				ELSE 'SDR'
			END as format,
			COUNT(*) as session_count
		FROM playback_events
		WHERE %s
		GROUP BY format
		ORDER BY session_count DESC
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query HDR formats: %w", err)
	}
	defer rows.Close()

	var formats []HDRFormatStats
	for rows.Next() {
		var format string
		var count int
		if err := rows.Scan(&format, &count); err != nil {
			return nil, fmt.Errorf("failed to scan HDR format: %w", err)
		}
		formats = append(formats, HDRFormatStats{
			Format:       format,
			SessionCount: count,
			Percentage:   calcPercentage(count, totalSessions),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating HDR formats: %w", err)
	}
	return formats, nil
}

// queryColorSpaces queries color space distribution
func (db *DB) queryColorSpaces(ctx context.Context, whereClause string, args []interface{}, totalSessions int) ([]ColorSpaceStats, error) {
	query := fmt.Sprintf(`
		SELECT
			COALESCE(video_color_space, 'unknown') as color_space,
			COUNT(*) as session_count
		FROM playback_events
		WHERE %s AND video_color_space IS NOT NULL AND video_color_space != ''
		GROUP BY color_space
		ORDER BY session_count DESC
		LIMIT 10
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query color spaces: %w", err)
	}
	defer rows.Close()

	var spaces []ColorSpaceStats
	for rows.Next() {
		var colorSpace string
		var count int
		if err := rows.Scan(&colorSpace, &count); err != nil {
			return nil, fmt.Errorf("failed to scan color space: %w", err)
		}
		spaces = append(spaces, ColorSpaceStats{
			ColorSpace:   colorSpace,
			SessionCount: count,
			Percentage:   calcPercentage(count, totalSessions),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating color spaces: %w", err)
	}
	return spaces, nil
}

// queryColorPrimaries queries color primaries distribution
func (db *DB) queryColorPrimaries(ctx context.Context, whereClause string, args []interface{}, totalSessions int) ([]ColorPrimariesStats, error) {
	query := fmt.Sprintf(`
		SELECT
			COALESCE(video_color_primaries, 'unknown') as primaries,
			COUNT(*) as session_count
		FROM playback_events
		WHERE %s AND video_color_primaries IS NOT NULL AND video_color_primaries != ''
		GROUP BY primaries
		ORDER BY session_count DESC
		LIMIT 10
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query color primaries: %w", err)
	}
	defer rows.Close()

	var primaries []ColorPrimariesStats
	for rows.Next() {
		var p string
		var count int
		if err := rows.Scan(&p, &count); err != nil {
			return nil, fmt.Errorf("failed to scan color primaries: %w", err)
		}
		primaries = append(primaries, ColorPrimariesStats{
			Primaries:    p,
			SessionCount: count,
			Percentage:   calcPercentage(count, totalSessions),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating color primaries: %w", err)
	}
	return primaries, nil
}

// calcPercentage calculates percentage safely (returns 0 if total is 0)
func calcPercentage(count, total int) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(count) / float64(total) * 100
}

// GetHardwareTranscodeTrends retrieves hardware transcode usage over time
func (db *DB) GetHardwareTranscodeTrends(ctx context.Context, filter LocationStatsFilter) ([]HWTranscodeTrend, error) {
	whereClause, args := buildFilterWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT
			DATE_TRUNC('day', started_at) as day,
			COUNT(*) as total_sessions,
			SUM(CASE WHEN transcode_hw_encoding = 1 OR transcode_hw_decoding = 1 THEN 1 ELSE 0 END) as hw_sessions,
			SUM(CASE WHEN transcode_hw_full_pipeline = 1 THEN 1 ELSE 0 END) as full_pipeline_sessions
		FROM playback_events
		WHERE %s
		GROUP BY day
		ORDER BY day DESC
		LIMIT 30
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query HW transcode trends: %w", err)
	}
	defer rows.Close()

	var trends []HWTranscodeTrend
	for rows.Next() {
		var trend HWTranscodeTrend
		if err := rows.Scan(&trend.Day, &trend.TotalSessions, &trend.HWSessions, &trend.FullPipelineSessions); err != nil {
			return nil, fmt.Errorf("failed to scan HW transcode trend: %w", err)
		}
		if trend.TotalSessions > 0 {
			trend.HWPercentage = float64(trend.HWSessions) / float64(trend.TotalSessions) * 100
		}
		trends = append(trends, trend)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating HW transcode trends: %w", err)
	}

	return trends, nil
}

// HWTranscodeTrend represents hardware transcode usage for a single day
type HWTranscodeTrend struct {
	Day                  string  `json:"day"`
	TotalSessions        int     `json:"total_sessions"`
	HWSessions           int     `json:"hw_sessions"`
	FullPipelineSessions int     `json:"full_pipeline_sessions"`
	HWPercentage         float64 `json:"hw_percentage"`
}
