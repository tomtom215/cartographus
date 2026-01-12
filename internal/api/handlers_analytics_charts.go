// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package api provides HTTP handlers for the Cartographus application.
// This file contains handlers for advanced chart analytics endpoints:
//   - Content Flow (Sankey): Show -> Season -> Episode viewing journeys
//   - User Content Overlap (Chord): User-user content similarity
//   - User Profile (Radar): Multi-dimensional user engagement
//   - Library Utilization (Treemap): Hierarchical library usage
//   - Calendar Heatmap: Daily activity patterns
//   - Bump Chart: Content ranking changes over time
package api

import (
	"context"
	"net/http"

	"github.com/tomtom215/cartographus/internal/database"
)

// AnalyticsContentFlow returns content flow data for Sankey diagram visualization.
//
// Method: GET
// Path: /api/v1/analytics/content-flow
//
// This endpoint provides viewing journey data from shows to seasons to episodes,
// enabling visualization of content consumption patterns and drop-off analysis.
//
// Use Cases:
//   - Visualize how viewers progress through TV series
//   - Identify drop-off points in multi-season shows
//   - Understand binge-watching patterns
//
// Query Parameters: Standard filter dimensions (users, media_types, date range)
//
// Response: ContentFlowAnalytics with Sankey nodes, links, and journey data
//
// SECURITY (RBAC):
//   - Admins: See all users' content journeys
//   - Regular users: See only their own viewing journeys
//
// Deterministic: Query hash in metadata for cache validation
// Auditable: Execution time in metadata
// Traceable: Total flows and show counts in response
// Observable: Full metadata with node/link statistics
func (h *Handler) AnalyticsContentFlow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsContentFlow", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return h.db.GetContentFlowAnalytics(ctx, filter)
	})
}

// AnalyticsUserOverlap returns user-user content overlap for Chord diagram.
//
// Method: GET
// Path: /api/v1/analytics/user-overlap
//
// This endpoint calculates Jaccard similarity between users based on shared
// content consumption, enabling visualization of viewing taste clusters.
//
// Use Cases:
//   - Identify users with similar viewing tastes
//   - Discover viewing communities within your user base
//   - Support collaborative filtering recommendations
//
// Query Parameters: Standard filter dimensions (date range, users)
//
// Response: UserContentOverlapAnalytics with similarity matrix and top pairs
//
// SECURITY (RBAC):
//   - ADMIN ONLY: This endpoint exposes cross-user viewing patterns
//   - Regular users receive 403 Forbidden
//
// Privacy: User IDs are pseudonymized in public deployments
// Deterministic: Jaccard similarity calculation is deterministic
// Auditable: User counts and connection counts in response
// Observable: Full metadata with cluster statistics
func (h *Handler) AnalyticsUserOverlap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteAdminOnly(w, r, "AnalyticsUserOverlap", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return h.db.GetUserContentOverlapAnalytics(ctx, filter)
	})
}

// AnalyticsUserProfile returns multi-dimensional user engagement scores for Radar chart.
//
// Method: GET
// Path: /api/v1/analytics/user-profile
//
// This endpoint calculates user engagement across six dimensions:
//   - Watch Time: Total viewing hours
//   - Completion: Average completion rate
//   - Diversity: Genre/content variety
//   - Quality: Stream quality preferences
//   - Discovery: New content exploration rate
//   - Social: Watch party participation
//
// Use Cases:
//   - Compare user engagement profiles
//   - Identify power users and casual viewers
//   - Segment users for targeted recommendations
//
// Query Parameters: Standard filter dimensions (date range, users)
//
// Response: UserProfileAnalytics with scores for each user on each dimension
//
// SECURITY (RBAC):
//   - Admins: See all users' engagement profiles
//   - Regular users: See only their own profile
//
// Deterministic: Normalized scores (0-100) for consistent visualization
// Auditable: User counts and average profiles in response
// Observable: Full metadata with ranking information
func (h *Handler) AnalyticsUserProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsUserProfile", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return h.db.GetUserProfileAnalytics(ctx, filter)
	})
}

// AnalyticsLibraryUtilization returns hierarchical library usage for Treemap.
//
// Method: GET
// Path: /api/v1/analytics/library-utilization
//
// This endpoint provides hierarchical library usage data organized as:
// Library -> Section Type -> Genre -> Content
//
// Use Cases:
//   - Visualize library usage at a glance
//   - Identify underutilized content areas
//   - Optimize library organization
//
// Query Parameters: Standard filter dimensions (date range, libraries)
//
// Response: LibraryUtilizationAnalytics with hierarchical tree data
//
// SECURITY (RBAC):
//   - Admins: See global library utilization
//   - Regular users: See only their own usage patterns
//
// Deterministic: Consistent tree structure for same data
// Auditable: Total content and utilization rates in response
// Observable: Full metadata with hierarchy statistics
func (h *Handler) AnalyticsLibraryUtilization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsLibraryUtilization", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return h.db.GetLibraryUtilizationAnalytics(ctx, filter)
	})
}

// AnalyticsCalendarHeatmap returns daily activity for calendar heatmap visualization.
//
// Method: GET
// Path: /api/v1/analytics/calendar-heatmap
//
// This endpoint provides GitHub-style contribution graph data showing
// daily viewing activity with normalized intensity values.
//
// Use Cases:
//   - Visualize viewing patterns over time
//   - Identify seasonal viewing trends
//   - Track engagement streaks
//
// Query Parameters: Standard filter dimensions (date range, users)
//
// Response: CalendarHeatmapAnalytics with daily activity and streak data
//
// SECURITY (RBAC):
//   - Admins: See global activity patterns
//   - Regular users: See only their own daily activity
//
// Deterministic: Consistent intensity normalization
// Auditable: Activity statistics and streak counts in response
// Observable: Full metadata with daily averages
func (h *Handler) AnalyticsCalendarHeatmap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsCalendarHeatmap", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return h.db.GetCalendarHeatmapAnalytics(ctx, filter)
	})
}

// AnalyticsBumpChart returns content ranking changes for Bump chart visualization.
//
// Method: GET
// Path: /api/v1/analytics/bump-chart
//
// This endpoint provides content ranking data over time, showing how
// the top 10 content items change positions week-by-week.
//
// Use Cases:
//   - Track trending content over time
//   - Identify viral content moments
//   - Analyze content lifecycle patterns
//
// Query Parameters: Standard filter dimensions (date range, media types)
//
// Response: BumpChartAnalytics with ranking data per time period
//
// SECURITY (RBAC):
//   - ADMIN ONLY: This endpoint provides global ranking insights
//   - Regular users receive 403 Forbidden
//
// Deterministic: Consistent ranking calculation
// Auditable: Period counts and ranking entries in response
// Observable: Full metadata with movers and shakers
func (h *Handler) AnalyticsBumpChart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteAdminOnly(w, r, "AnalyticsBumpChart", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return h.db.GetBumpChartAnalytics(ctx, filter)
	})
}
