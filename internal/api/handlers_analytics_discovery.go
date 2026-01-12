// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package api provides HTTP handlers for the Cartographus application.
// This file contains analytics handlers for device migration tracking and content discovery.
package api

import (
	"context"
	"net/http"

	"github.com/tomtom215/cartographus/internal/database"
)

// AnalyticsDeviceMigration returns comprehensive device migration analytics.
//
// Method: GET
// Path: /api/v1/analytics/device-migration
//
// This endpoint provides device and platform migration analytics including:
//   - Summary statistics (multi-device users, migration counts)
//   - Top user device profiles with platform history
//   - Recent platform migration events
//   - Platform adoption trends over time
//   - Common platform transition paths
//   - Current platform distribution
//
// The endpoint uses LAG() window functions to detect platform switches and
// calculates metrics like permanent switches vs temporary device usage.
//
// Query Parameters: Standard filter dimensions (users, media_types, platforms, etc.)
//
// Response: DeviceMigrationAnalytics with comprehensive migration data and metadata
//
// Deterministic: Query hash in metadata for cache validation
// Auditable: Execution time and event counts in metadata
// Traceable: Date range and platform counts in metadata
// Observable: Full metadata with query execution details
func (h *Handler) AnalyticsDeviceMigration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsDeviceMigration", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return h.db.GetDeviceMigrationAnalytics(ctx, filter)
	})
}

// AnalyticsContentDiscovery returns comprehensive content discovery analytics.
//
// Method: GET
// Path: /api/v1/analytics/content-discovery
//
// This endpoint provides content discovery and time-to-first-watch analytics including:
//   - Summary statistics (discovery rates, average time to first watch)
//   - Time bucket distribution (0-24h, 1-7d, 7-30d, 30-90d, 90d+)
//   - Early adopters (users who discover content quickly)
//   - Recently discovered content with discovery velocity
//   - Stale content (added but never watched)
//   - Per-library discovery statistics
//   - Discovery trends over time
//
// The endpoint uses added_at timestamps to calculate time between content
// addition and first playback, identifying content engagement patterns.
//
// Query Parameters: Standard filter dimensions (users, media_types, platforms, etc.)
//
// Response: ContentDiscoveryAnalytics with comprehensive discovery data and metadata
//
// Deterministic: Query hash in metadata for cache validation
// Auditable: Execution time and event counts in metadata
// Traceable: Date range and content counts in metadata
// Observable: Full metadata with discovery thresholds
func (h *Handler) AnalyticsContentDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsContentDiscovery", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return h.db.GetContentDiscoveryAnalytics(ctx, filter)
	})
}
