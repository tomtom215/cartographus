// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package api provides HTTP handlers for the Cartographus application.
// This file contains enhanced analytics endpoints for production-grade insights.
//
// Enhanced Analytics Endpoints:
//   - AnalyticsCohortRetention: Cohort-based user retention analysis
//   - AnalyticsQoE: Quality of Experience dashboard (Netflix-style metrics)
//   - AnalyticsDataQuality: Data quality monitoring for observability
//   - AnalyticsUserNetwork: Social viewing network graph
//
// All endpoints follow the same patterns as existing analytics:
//   - Standard 14+ filter dimensions
//   - 5-minute caching for performance
//   - Query provenance metadata for auditability
//   - Deterministic query hashes for reproducibility
package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/tomtom215/cartographus/internal/database"
)

// AnalyticsCohortRetention returns cohort-based user retention analysis.
//
// Method: GET
// Path: /api/v1/analytics/cohort-retention
//
// Query Parameters:
//   - Standard filter dimensions (start_date, end_date, users, media_types, etc.)
//   - max_weeks: Maximum weeks to track per cohort (default: 12, max: 52)
//   - min_cohort_size: Minimum users to include a cohort (default: 3)
//   - granularity: "week" or "month" (default: "week")
//
// Response: CohortRetentionAnalytics with cohort data, summary, retention curve, and metadata.
//
// This endpoint follows industry best practices from Mixpanel/Amplitude for cohort analysis.
// It helps understand:
//   - When users typically stop using the service (churn points)
//   - Which cohorts have best/worst retention
//   - Overall user engagement health over time
func (h *Handler) AnalyticsCohortRetention(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsCohortRetention", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		// Parse configuration from query parameters
		config := database.DefaultCohortConfig()

		if maxWeeks := r.URL.Query().Get("max_weeks"); maxWeeks != "" {
			if val, err := strconv.Atoi(maxWeeks); err == nil && val > 0 && val <= 52 {
				config.MaxWeeks = val
			}
		}

		if minSize := r.URL.Query().Get("min_cohort_size"); minSize != "" {
			if val, err := strconv.Atoi(minSize); err == nil && val > 0 {
				config.MinCohortSize = val
			}
		}

		if granularity := r.URL.Query().Get("granularity"); granularity == "month" || granularity == "week" {
			config.Granularity = granularity
		}

		return h.db.GetCohortRetentionAnalytics(ctx, filter, config)
	})
}

// AnalyticsQoE returns Quality of Experience dashboard metrics.
//
// Method: GET
// Path: /api/v1/analytics/qoe
//
// Query Parameters: Standard filter dimensions
//
// Response: QoEDashboard with summary, trends, platform breakdown, and issues.
//
// This endpoint implements Netflix-style QoE metrics:
//   - EBVS (Exit Before Video Starts): Sessions abandoned before playback
//   - Quality Degradation Rate: Source quality higher than delivered
//   - Transcode Rate: Percentage requiring server-side transcoding
//   - Pause Rate: Indicator of potential buffering issues
//   - QoE Score: Composite score (0-100) with letter grade
//
// Use this to understand and improve user experience quality.
func (h *Handler) AnalyticsQoE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsQoE", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return h.db.GetQoEDashboard(ctx, filter)
	})
}

// AnalyticsDataQuality returns data quality monitoring metrics.
//
// Method: GET
// Path: /api/v1/analytics/data-quality
//
// Query Parameters: Standard filter dimensions
//
// Response: DataQualityReport with summary, field metrics, trends, issues, and source breakdown.
//
// This endpoint provides production-grade observability for:
//   - Field completeness (null/empty rates for critical fields)
//   - Value validity (out-of-range, invalid values)
//   - Data consistency (duplicates, orphaned records)
//   - Source-specific quality metrics
//   - Daily quality trends
//
// Use this for:
//   - Ensuring analytics accuracy and trustworthiness
//   - Early detection of data pipeline issues
//   - Maintaining auditability and compliance
func (h *Handler) AnalyticsDataQuality(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsDataQuality", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return h.db.GetDataQualityReport(ctx, filter)
	})
}

// AnalyticsUserNetwork returns user relationship network graph data.
//
// Method: GET
// Path: /api/v1/analytics/user-network
//
// Query Parameters:
//   - Standard filter dimensions
//   - min_shared_sessions: Minimum shared sessions to create an edge (default: 2)
//   - min_content_overlap: Minimum Jaccard similarity for content overlap edge (default: 0.3)
//
// Response: UserNetworkGraph with nodes, edges, clusters, summary, and metadata.
//
// This endpoint visualizes user relationships based on:
//   - Shared viewing sessions (same IP, same time)
//   - Content overlap (similar taste profiles)
//   - Watch party participation (synchronized viewing)
//
// Use this for:
//   - Understanding social viewing patterns
//   - Identifying potential account sharing
//   - Discovering user communities
//   - Content recommendation opportunities
func (h *Handler) AnalyticsUserNetwork(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsUserNetwork", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		// Parse configuration
		minSharedSessions := 2
		if val := r.URL.Query().Get("min_shared_sessions"); val != "" {
			if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
				minSharedSessions = parsed
			}
		}

		minContentOverlap := 0.3
		if val := r.URL.Query().Get("min_content_overlap"); val != "" {
			if parsed, err := strconv.ParseFloat(val, 64); err == nil && parsed >= 0 && parsed <= 1 {
				minContentOverlap = parsed
			}
		}

		return h.db.GetUserNetworkGraph(ctx, filter, minSharedSessions, minContentOverlap)
	})
}
