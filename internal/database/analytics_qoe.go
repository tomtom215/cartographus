// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package database provides data access and analytics functionality for the Cartographus application.
// This file contains Quality of Experience (QoE) analytics following Netflix and industry standards.
//
// QoE metrics help server administrators understand:
// - User experience quality (abandonment, buffering indicators)
// - Stream quality issues (degradation, transcoding overhead)
// - Platform-specific problems
// - Areas for infrastructure improvement
//
// Key metrics implemented:
// - EBVS (Exit Before Video Starts): Sessions with 0% completion and <10s duration
// - Quality Degradation: Source quality higher than delivered quality
// - Transcode Rate: Percentage requiring transcoding
// - Pause Rate: Indicator of potential buffering issues
// - Completion Rate: Overall engagement quality
package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// GetQoEDashboard returns comprehensive Quality of Experience metrics
func (db *DB) GetQoEDashboard(ctx context.Context, filter LocationStatsFilter) (*models.QoEDashboard, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	startTime := time.Now()

	// Build filter conditions
	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereClause := "1=1"
	if len(whereClauses) > 0 {
		whereClause = join(whereClauses, " AND ")
	}

	// Execute all queries in parallel for performance
	var (
		summary      models.QoESummary
		trends       []models.QoETrendPoint
		byPlatform   []models.QoEByPlatform
		byTranscode  []models.QoEByTranscode
		issues       []models.QoEIssue
		summaryErr   error
		trendsErr    error
		platformErr  error
		transcodeErr error
		wg           sync.WaitGroup
	)

	wg.Add(4)

	go func() {
		defer wg.Done()
		summary, summaryErr = db.getQoESummary(ctx, whereClause, args)
	}()

	go func() {
		defer wg.Done()
		trends, trendsErr = db.getQoETrends(ctx, whereClause, args, filter)
	}()

	go func() {
		defer wg.Done()
		byPlatform, platformErr = db.getQoEByPlatform(ctx, whereClause, args)
	}()

	go func() {
		defer wg.Done()
		byTranscode, transcodeErr = db.getQoEByTranscode(ctx, whereClause, args)
	}()

	wg.Wait()

	// Check for errors
	if summaryErr != nil {
		return nil, fmt.Errorf("QoE summary query failed: %w", summaryErr)
	}
	if trendsErr != nil {
		return nil, fmt.Errorf("QoE trends query failed: %w", trendsErr)
	}
	if platformErr != nil {
		return nil, fmt.Errorf("QoE platform query failed: %w", platformErr)
	}
	if transcodeErr != nil {
		return nil, fmt.Errorf("QoE transcode query failed: %w", transcodeErr)
	}

	// Generate issues from the data
	issues = generateQoEIssues(&summary, byPlatform)

	// Generate query hash
	queryHash := generateQoEQueryHash(filter)

	// Get data range
	dataRangeStart, dataRangeEnd := getDataRange(filter)

	// Determine trend interval
	trendInterval := "day"
	if dataRangeEnd.Sub(dataRangeStart) < 7*24*time.Hour {
		trendInterval = "hour"
	}

	return &models.QoEDashboard{
		Summary:             summary,
		Trends:              trends,
		ByPlatform:          byPlatform,
		ByTranscodeDecision: byTranscode,
		TopIssues:           issues,
		Metadata: models.QoEQueryMetadata{
			QueryHash:      queryHash,
			DataRangeStart: dataRangeStart,
			DataRangeEnd:   dataRangeEnd,
			TrendInterval:  trendInterval,
			EventCount:     summary.TotalSessions,
			GeneratedAt:    time.Now(),
			QueryTimeMs:    time.Since(startTime).Milliseconds(),
			Cached:         false,
		},
	}, nil
}

// getQoESummary calculates aggregate QoE metrics
func (db *DB) getQoESummary(ctx context.Context, whereClause string, args []interface{}) (models.QoESummary, error) {
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) AS total_sessions,

			-- EBVS: Exit Before Video Starts (0%% completion AND <10s duration)
			COALESCE(SUM(CASE WHEN percent_complete = 0 AND COALESCE(play_duration, 0) < 10 THEN 1 ELSE 0 END), 0) AS ebvs_count,

			-- Quality Degradation: source resolution > stream resolution
			COALESCE(SUM(CASE
				WHEN video_resolution IS NOT NULL
				AND stream_video_resolution IS NOT NULL
				AND video_resolution != stream_video_resolution
				AND (
					(video_resolution = '4k' AND stream_video_resolution IN ('1080', '720', '480', 'sd')) OR
					(video_resolution = '1080' AND stream_video_resolution IN ('720', '480', 'sd')) OR
					(video_resolution = '720' AND stream_video_resolution IN ('480', 'sd'))
				)
				THEN 1 ELSE 0 END), 0) AS quality_degrade_count,

			-- Transcode vs Direct Play
			COALESCE(SUM(CASE WHEN LOWER(transcode_decision) = 'transcode' THEN 1 ELSE 0 END), 0) AS transcode_count,
			COALESCE(SUM(CASE WHEN LOWER(transcode_decision) = 'direct play' THEN 1 ELSE 0 END), 0) AS direct_play_count,

			-- Completion metrics
			COALESCE(AVG(percent_complete), 0) AS avg_completion,
			COALESCE(SUM(CASE WHEN percent_complete >= 80 THEN 1 ELSE 0 END), 0) AS high_completion_count,

			-- Pause metrics (potential buffering indicator)
			COALESCE(SUM(CASE WHEN paused_counter > 0 THEN 1 ELSE 0 END), 0) AS pause_session_count,
			COALESCE(AVG(CASE WHEN paused_counter IS NOT NULL THEN paused_counter ELSE 0 END), 0) AS avg_pause_count,

			-- Connection metrics
			COALESCE(SUM(CASE WHEN relayed = 1 OR relay = 1 THEN 1 ELSE 0 END), 0) AS relayed_count,
			COALESCE(SUM(CASE WHEN secure = 1 THEN 1 ELSE 0 END), 0) AS secure_count,

			-- Bitrate metrics
			COALESCE(AVG(CASE WHEN stream_bitrate > 0 THEN stream_bitrate ELSE NULL END), 0) AS avg_bitrate,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY CASE WHEN stream_bitrate > 0 THEN stream_bitrate ELSE NULL END), 0) AS bitrate_p50,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY CASE WHEN stream_bitrate > 0 THEN stream_bitrate ELSE NULL END), 0) AS bitrate_p95

		FROM playback_events
		WHERE %s
	`, whereClause)

	var summary models.QoESummary
	var highCompletionCount, pauseSessionCount int64
	var avgBitrate, bitrateP50, bitrateP95 float64

	err := db.conn.QueryRowContext(ctx, query, args...).Scan(
		&summary.TotalSessions,
		&summary.EBVSCount,
		&summary.QualityDegradeCount,
		&summary.TranscodeCount,
		&summary.DirectPlayCount,
		&summary.AvgCompletion,
		&highCompletionCount,
		&pauseSessionCount,
		&summary.AvgPauseCount,
		&summary.RelayedRate,          // Temporarily store count here
		&summary.SecureConnectionRate, // Temporarily store count here
		&avgBitrate,
		&bitrateP50,
		&bitrateP95,
	)
	if err != nil {
		return summary, fmt.Errorf("scan QoE summary: %w", err)
	}

	// Calculate rates
	if summary.TotalSessions > 0 {
		summary.EBVSRate = float64(summary.EBVSCount) / float64(summary.TotalSessions) * 100.0
		summary.QualityDegradeRate = float64(summary.QualityDegradeCount) / float64(summary.TotalSessions) * 100.0
		summary.TranscodeRate = float64(summary.TranscodeCount) / float64(summary.TotalSessions) * 100.0
		summary.DirectPlayRate = float64(summary.DirectPlayCount) / float64(summary.TotalSessions) * 100.0
		summary.HighCompletionRate = float64(highCompletionCount) / float64(summary.TotalSessions) * 100.0
		summary.PauseRate = float64(pauseSessionCount) / float64(summary.TotalSessions) * 100.0

		// Convert stored counts to rates
		relayedCount := int64(summary.RelayedRate)
		secureCount := int64(summary.SecureConnectionRate)
		summary.RelayedRate = float64(relayedCount) / float64(summary.TotalSessions) * 100.0
		summary.SecureConnectionRate = float64(secureCount) / float64(summary.TotalSessions) * 100.0
	}

	// Convert bitrate to Mbps
	summary.AvgBitrateMbps = avgBitrate / 1000.0
	summary.BitrateP50Mbps = bitrateP50 / 1000.0
	summary.BitrateP95Mbps = bitrateP95 / 1000.0

	// Calculate QoE Score (0-100)
	// Formula: 100 - (EBVS_penalty + QualityDegrade_penalty + Pause_penalty + LowCompletion_penalty)
	ebvsPenalty := summary.EBVSRate * 5 // High weight: EBVS is severe
	qualityPenalty := summary.QualityDegradeRate * 2
	pausePenalty := summary.PauseRate * 1
	completionPenalty := (100 - summary.AvgCompletion) * 0.3

	summary.QoEScore = 100 - (ebvsPenalty + qualityPenalty + pausePenalty + completionPenalty)
	if summary.QoEScore < 0 {
		summary.QoEScore = 0
	}
	if summary.QoEScore > 100 {
		summary.QoEScore = 100
	}

	// Assign grade
	summary.QoEGrade = getQoEGrade(summary.QoEScore)

	return summary, nil
}

// getQoETrends calculates QoE metrics over time
func (db *DB) getQoETrends(ctx context.Context, whereClause string, args []interface{}, filter LocationStatsFilter) ([]models.QoETrendPoint, error) {
	// Determine interval based on date range
	interval := "day"
	dataRangeStart, dataRangeEnd := getDataRange(filter)
	if dataRangeEnd.Sub(dataRangeStart) < 7*24*time.Hour {
		interval = "hour"
	}

	query := fmt.Sprintf(`
		SELECT
			DATE_TRUNC('%s', started_at) AS time_bucket,
			COUNT(*) AS session_count,
			COALESCE(SUM(CASE WHEN percent_complete = 0 AND COALESCE(play_duration, 0) < 10 THEN 1 ELSE 0 END) * 100.0 / NULLIF(COUNT(*), 0), 0) AS ebvs_rate,
			COALESCE(SUM(CASE
				WHEN video_resolution IS NOT NULL AND stream_video_resolution IS NOT NULL
				AND video_resolution != stream_video_resolution THEN 1 ELSE 0 END) * 100.0 / NULLIF(COUNT(*), 0), 0) AS quality_degrade_rate,
			COALESCE(SUM(CASE WHEN LOWER(transcode_decision) = 'transcode' THEN 1 ELSE 0 END) * 100.0 / NULLIF(COUNT(*), 0), 0) AS transcode_rate,
			COALESCE(AVG(percent_complete), 0) AS avg_completion,
			COALESCE(AVG(CASE WHEN stream_bitrate > 0 THEN stream_bitrate ELSE NULL END) / 1000.0, 0) AS avg_bitrate_mbps
		FROM playback_events
		WHERE %s
		GROUP BY time_bucket
		ORDER BY time_bucket
	`, interval, whereClause)

	var trends []models.QoETrendPoint
	err := db.queryAndScan(ctx, query, args, func(rows *sql.Rows) error {
		var point models.QoETrendPoint
		if err := rows.Scan(
			&point.Timestamp,
			&point.SessionCount,
			&point.EBVSRate,
			&point.QualityDegradeRate,
			&point.TranscodeRate,
			&point.AvgCompletion,
			&point.AvgBitrateMbps,
		); err != nil {
			return err
		}

		// Calculate QoE score for this point
		ebvsPenalty := point.EBVSRate * 5
		qualityPenalty := point.QualityDegradeRate * 2
		completionPenalty := (100 - point.AvgCompletion) * 0.3
		point.QoEScore = 100 - (ebvsPenalty + qualityPenalty + completionPenalty)
		if point.QoEScore < 0 {
			point.QoEScore = 0
		}
		if point.QoEScore > 100 {
			point.QoEScore = 100
		}

		trends = append(trends, point)
		return nil
	})

	return trends, err
}

// getQoEByPlatform breaks down QoE by platform
func (db *DB) getQoEByPlatform(ctx context.Context, whereClause string, args []interface{}) ([]models.QoEByPlatform, error) {
	query := fmt.Sprintf(`
		WITH platform_stats AS (
			SELECT
				COALESCE(platform, 'Unknown') AS platform,
				COUNT(*) AS session_count,
				SUM(CASE WHEN percent_complete = 0 AND COALESCE(play_duration, 0) < 10 THEN 1 ELSE 0 END) AS ebvs_count,
				SUM(CASE
					WHEN video_resolution IS NOT NULL AND stream_video_resolution IS NOT NULL
					AND video_resolution != stream_video_resolution THEN 1 ELSE 0 END) AS quality_degrade_count,
				SUM(CASE WHEN LOWER(transcode_decision) = 'transcode' THEN 1 ELSE 0 END) AS transcode_count,
				SUM(CASE WHEN LOWER(transcode_decision) = 'direct play' THEN 1 ELSE 0 END) AS direct_play_count,
				AVG(percent_complete) AS avg_completion,
				AVG(CASE WHEN stream_bitrate > 0 THEN stream_bitrate ELSE NULL END) AS avg_bitrate
			FROM playback_events
			WHERE %s
			GROUP BY platform
		),
		total AS (
			SELECT SUM(session_count) AS total_sessions FROM platform_stats
		)
		SELECT
			ps.platform,
			ps.session_count,
			COALESCE(ps.session_count * 100.0 / NULLIF(t.total_sessions, 0), 0) AS session_percentage,
			COALESCE(ps.ebvs_count * 100.0 / NULLIF(ps.session_count, 0), 0) AS ebvs_rate,
			COALESCE(ps.quality_degrade_count * 100.0 / NULLIF(ps.session_count, 0), 0) AS quality_degrade_rate,
			COALESCE(ps.transcode_count * 100.0 / NULLIF(ps.session_count, 0), 0) AS transcode_rate,
			COALESCE(ps.direct_play_count * 100.0 / NULLIF(ps.session_count, 0), 0) AS direct_play_rate,
			COALESCE(ps.avg_completion, 0) AS avg_completion,
			COALESCE(ps.avg_bitrate / 1000.0, 0) AS avg_bitrate_mbps
		FROM platform_stats ps
		CROSS JOIN total t
		ORDER BY ps.session_count DESC
		LIMIT 20
	`, whereClause)

	var platforms []models.QoEByPlatform
	err := db.queryAndScan(ctx, query, args, func(rows *sql.Rows) error {
		var p models.QoEByPlatform
		if err := rows.Scan(
			&p.Platform,
			&p.SessionCount,
			&p.SessionPercentage,
			&p.EBVSRate,
			&p.QualityDegradeRate,
			&p.TranscodeRate,
			&p.DirectPlayRate,
			&p.AvgCompletion,
			&p.AvgBitrateMbps,
		); err != nil {
			return err
		}

		// Calculate QoE score for this platform
		ebvsPenalty := p.EBVSRate * 5
		qualityPenalty := p.QualityDegradeRate * 2
		completionPenalty := (100 - p.AvgCompletion) * 0.3
		p.QoEScore = 100 - (ebvsPenalty + qualityPenalty + completionPenalty)
		if p.QoEScore < 0 {
			p.QoEScore = 0
		}
		if p.QoEScore > 100 {
			p.QoEScore = 100
		}
		p.QoEGrade = getQoEGrade(p.QoEScore)

		platforms = append(platforms, p)
		return nil
	})

	return platforms, err
}

// getQoEByTranscode breaks down QoE by transcode decision
func (db *DB) getQoEByTranscode(ctx context.Context, whereClause string, args []interface{}) ([]models.QoEByTranscode, error) {
	query := fmt.Sprintf(`
		WITH transcode_stats AS (
			SELECT
				COALESCE(LOWER(transcode_decision), 'unknown') AS transcode_decision,
				COUNT(*) AS session_count,
				SUM(CASE WHEN percent_complete = 0 AND COALESCE(play_duration, 0) < 10 THEN 1 ELSE 0 END) AS ebvs_count,
				AVG(percent_complete) AS avg_completion,
				AVG(CASE WHEN stream_bitrate > 0 THEN stream_bitrate ELSE NULL END) AS avg_bitrate
			FROM playback_events
			WHERE %s
			GROUP BY transcode_decision
		),
		total AS (
			SELECT SUM(session_count) AS total_sessions FROM transcode_stats
		)
		SELECT
			ts.transcode_decision,
			ts.session_count,
			COALESCE(ts.session_count * 100.0 / NULLIF(t.total_sessions, 0), 0) AS session_percentage,
			COALESCE(ts.ebvs_count * 100.0 / NULLIF(ts.session_count, 0), 0) AS ebvs_rate,
			COALESCE(ts.avg_completion, 0) AS avg_completion,
			COALESCE(ts.avg_bitrate / 1000.0, 0) AS avg_bitrate_mbps
		FROM transcode_stats ts
		CROSS JOIN total t
		ORDER BY ts.session_count DESC
	`, whereClause)

	var transcodes []models.QoEByTranscode
	err := db.queryAndScan(ctx, query, args, func(rows *sql.Rows) error {
		var t models.QoEByTranscode
		if err := rows.Scan(
			&t.TranscodeDecision,
			&t.SessionCount,
			&t.SessionPercentage,
			&t.EBVSRate,
			&t.AvgCompletion,
			&t.AvgBitrateMbps,
		); err != nil {
			return err
		}

		// Calculate QoE score
		ebvsPenalty := t.EBVSRate * 5
		completionPenalty := (100 - t.AvgCompletion) * 0.3
		t.QoEScore = 100 - (ebvsPenalty + completionPenalty)
		if t.QoEScore < 0 {
			t.QoEScore = 0
		}
		if t.QoEScore > 100 {
			t.QoEScore = 100
		}

		transcodes = append(transcodes, t)
		return nil
	})

	return transcodes, err
}

// generateQoEIssues creates a list of detected quality issues
func generateQoEIssues(summary *models.QoESummary, platforms []models.QoEByPlatform) []models.QoEIssue {
	var issues []models.QoEIssue

	// Check EBVS rate
	if summary.EBVSRate > 5 {
		issues = append(issues, models.QoEIssue{
			IssueType:        "high_ebvs",
			Severity:         getSeverity(summary.EBVSRate, 5, 10),
			Title:            "High Exit Before Video Starts Rate",
			Description:      fmt.Sprintf("%.1f%% of sessions exit before video starts (%.0f sessions)", summary.EBVSRate, float64(summary.EBVSCount)),
			AffectedSessions: summary.EBVSCount,
			ImpactPercentage: summary.EBVSRate,
			Recommendation:   "Check server startup time, network latency, and client application performance",
		})
	}

	// Check quality degradation
	if summary.QualityDegradeRate > 20 {
		issues = append(issues, models.QoEIssue{
			IssueType:        "quality_degradation",
			Severity:         getSeverity(summary.QualityDegradeRate, 20, 40),
			Title:            "High Quality Degradation Rate",
			Description:      fmt.Sprintf("%.1f%% of sessions experience quality reduction (%.0f sessions)", summary.QualityDegradeRate, float64(summary.QualityDegradeCount)),
			AffectedSessions: summary.QualityDegradeCount,
			ImpactPercentage: summary.QualityDegradeRate,
			Recommendation:   "Consider optimizing library for common device capabilities or upgrading network bandwidth",
		})
	}

	// Check transcode rate
	if summary.TranscodeRate > 50 {
		issues = append(issues, models.QoEIssue{
			IssueType:        "high_transcode",
			Severity:         getSeverity(summary.TranscodeRate, 50, 75),
			Title:            "High Transcoding Rate",
			Description:      fmt.Sprintf("%.1f%% of sessions require transcoding (%.0f sessions)", summary.TranscodeRate, float64(summary.TranscodeCount)),
			AffectedSessions: summary.TranscodeCount,
			ImpactPercentage: summary.TranscodeRate,
			Recommendation:   "Add media in formats compatible with your clients (H.264 for broad compatibility) or upgrade server CPU",
		})
	}

	// Check pause rate (potential buffering indicator)
	if summary.PauseRate > 30 {
		issues = append(issues, models.QoEIssue{
			IssueType:        "high_pause",
			Severity:         "warning",
			Title:            "High Pause Event Rate",
			Description:      fmt.Sprintf("%.1f%% of sessions have pause events (avg %.1f pauses per session)", summary.PauseRate, summary.AvgPauseCount),
			AffectedSessions: int64(float64(summary.TotalSessions) * summary.PauseRate / 100),
			ImpactPercentage: summary.PauseRate,
			Recommendation:   "This may indicate buffering issues. Check bandwidth, server performance, and CDN configuration",
		})
	}

	// Check completion rate
	if summary.AvgCompletion < 50 {
		issues = append(issues, models.QoEIssue{
			IssueType:        "low_completion",
			Severity:         getSeverity(50-summary.AvgCompletion, 20, 35),
			Title:            "Low Average Completion Rate",
			Description:      fmt.Sprintf("Average completion is only %.1f%%", summary.AvgCompletion),
			AffectedSessions: summary.TotalSessions,
			ImpactPercentage: 100 - summary.AvgCompletion,
			Recommendation:   "Investigate content quality, playback issues, or user engagement factors",
		})
	}

	// Check for platform-specific issues
	for _, p := range platforms {
		if p.EBVSRate > 10 && p.SessionCount > 50 {
			issues = append(issues, models.QoEIssue{
				IssueType:        "platform_ebvs",
				Severity:         "warning",
				Title:            fmt.Sprintf("High EBVS on %s", p.Platform),
				Description:      fmt.Sprintf("%s has %.1f%% EBVS rate (%d sessions)", p.Platform, p.EBVSRate, p.SessionCount),
				AffectedSessions: int64(float64(p.SessionCount) * p.EBVSRate / 100),
				ImpactPercentage: p.EBVSRate,
				Recommendation:   fmt.Sprintf("Investigate %s client app performance and compatibility", p.Platform),
				RelatedDimension: fmt.Sprintf("platform:%s", p.Platform),
			})
		}
	}

	return issues
}

// getQoEGrade converts a QoE score to a letter grade
func getQoEGrade(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}

// getSeverity determines issue severity based on thresholds
func getSeverity(value, warningThreshold, criticalThreshold float64) string {
	if value >= criticalThreshold {
		return "critical"
	}
	if value >= warningThreshold {
		return "warning"
	}
	return "info"
}

// generateQoEQueryHash creates a deterministic hash for reproducibility
func generateQoEQueryHash(filter LocationStatsFilter) string {
	canonical := "qoe|"
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
	if len(filter.Platforms) > 0 {
		canonical += fmt.Sprintf("platforms=%v|", filter.Platforms)
	}

	hash := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(hash[:8])
}
