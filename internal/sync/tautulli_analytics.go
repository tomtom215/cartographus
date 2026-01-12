// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
tautulli_analytics.go - Tautulli Analytics and Reporting Methods

This file provides methods for retrieving analytics and reporting data from
Tautulli, used primarily for charting and dashboard visualizations.

Analytics Methods:
  - GetHomeStats(): Dashboard statistics (top users, media, platforms)
  - GetPlaysByDate(): Daily playback counts for trend charts
  - GetPlaysByDayOfWeek(): Weekly distribution patterns
  - GetPlaysByHourOfDay(): Hourly activity patterns
  - GetPlaysBySourceResolution(): Resolution breakdown
  - GetPlaysByStreamResolution(): Streaming resolution analysis
  - GetPlaysByStreamType(): Direct play vs transcode distribution
  - GetPlaysByTopTenPlatforms(): Platform usage ranking
  - GetPlaysByTopTenUsers(): Top user activity

Time Range Parameters:
Most analytics methods accept a timeRange parameter (in days):
  - 0: All time
  - 7: Last week
  - 30: Last month
  - 90: Last quarter
  - 365: Last year

Y-Axis Options:
The yAxis parameter determines the metric being measured:
  - "plays": Number of playback events
  - "duration": Total watch time in seconds

These methods power the 47 charts across the 6 analytics dashboard pages.
*/

//nolint:staticcheck // File documentation, not package doc
package sync

import (
	"context"

	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

func (c *TautulliClient) GetHomeStats(ctx context.Context, timeRange int, statsType string, statsCount int) (*tautulli.TautulliHomeStats, error) {
	req := newAPIRequest("get_home_stats").
		addIntParam("time_range", timeRange).
		addParam("stats_type", statsType).
		addIntParam("stats_count", statsCount)

	return executeAPIRequest(ctx, c, req,
		func(r *tautulli.TautulliHomeStats) string { return r.Response.Result },
		func(r *tautulli.TautulliHomeStats) *string { return r.Response.Message },
	)
}

// GetPlaysByDate retrieves playback graph data organized by date
func (c *TautulliClient) GetPlaysByDate(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByDate, error) {
	req := newAPIRequest("get_plays_by_date")
	addTimeRangeParams(req, timeRange, yAxis, userID, grouping)

	return executeAPIRequest(ctx, c, req,
		func(r *tautulli.TautulliPlaysByDate) string { return r.Response.Result },
		func(r *tautulli.TautulliPlaysByDate) *string { return r.Response.Message },
	)
}

// GetPlaysByDayOfWeek retrieves playback data organized by day of week
func (c *TautulliClient) GetPlaysByDayOfWeek(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByDayOfWeek, error) {
	req := newAPIRequest("get_plays_by_dayofweek")
	addTimeRangeParams(req, timeRange, yAxis, userID, grouping)

	return executeAPIRequest(ctx, c, req,
		func(r *tautulli.TautulliPlaysByDayOfWeek) string { return r.Response.Result },
		func(r *tautulli.TautulliPlaysByDayOfWeek) *string { return r.Response.Message },
	)
}

// GetPlaysByHourOfDay retrieves playback data organized by hour of day
func (c *TautulliClient) GetPlaysByHourOfDay(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByHourOfDay, error) {
	req := newAPIRequest("get_plays_by_hourofday")
	addTimeRangeParams(req, timeRange, yAxis, userID, grouping)

	return executeAPIRequest(ctx, c, req,
		func(r *tautulli.TautulliPlaysByHourOfDay) string { return r.Response.Result },
		func(r *tautulli.TautulliPlaysByHourOfDay) *string { return r.Response.Message },
	)
}

// GetPlaysByStreamType retrieves playback data organized by stream type
func (c *TautulliClient) GetPlaysByStreamType(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByStreamType, error) {
	req := newAPIRequest("get_plays_by_stream_type")
	addTimeRangeParams(req, timeRange, yAxis, userID, grouping)

	return executeAPIRequest(ctx, c, req,
		func(r *tautulli.TautulliPlaysByStreamType) string { return r.Response.Result },
		func(r *tautulli.TautulliPlaysByStreamType) *string { return r.Response.Message },
	)
}

// GetConcurrentStreamsByStreamType retrieves concurrent streams data organized by stream type
func (c *TautulliClient) GetConcurrentStreamsByStreamType(ctx context.Context, timeRange int, userID int) (*tautulli.TautulliConcurrentStreamsByStreamType, error) {
	req := newAPIRequest("get_concurrent_streams_by_stream_type").
		addIntParam("time_range", timeRange).
		addIntParam("user_id", userID)

	return executeAPIRequest(ctx, c, req,
		func(r *tautulli.TautulliConcurrentStreamsByStreamType) string { return r.Response.Result },
		func(r *tautulli.TautulliConcurrentStreamsByStreamType) *string { return r.Response.Message },
	)
}

// GetItemWatchTimeStats retrieves watch time statistics for a specific media item
func (c *TautulliClient) GetItemWatchTimeStats(ctx context.Context, ratingKey string, grouping int, queryDays string) (*tautulli.TautulliItemWatchTimeStats, error) {
	req := newAPIRequest("get_item_watch_time_stats").
		addParam("rating_key", ratingKey).
		addIntParamZero("grouping", grouping).
		addParam("query_days", queryDays)

	return executeAPIRequest(ctx, c, req,
		func(r *tautulli.TautulliItemWatchTimeStats) string { return r.Response.Result },
		func(r *tautulli.TautulliItemWatchTimeStats) *string { return r.Response.Message },
	)
}

func (c *TautulliClient) GetPlaysBySourceResolution(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysBySourceResolution, error) {
	req := newAPIRequest("get_plays_by_source_resolution")
	addTimeRangeParams(req, timeRange, yAxis, userID, grouping)

	return executeAPIRequest(ctx, c, req,
		func(r *tautulli.TautulliPlaysBySourceResolution) string { return r.Response.Result },
		func(r *tautulli.TautulliPlaysBySourceResolution) *string { return r.Response.Message },
	)
}

// GetPlaysByStreamResolution retrieves plays by streamed video resolution
func (c *TautulliClient) GetPlaysByStreamResolution(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByStreamResolution, error) {
	req := newAPIRequest("get_plays_by_stream_resolution")
	addTimeRangeParams(req, timeRange, yAxis, userID, grouping)

	return executeAPIRequest(ctx, c, req,
		func(r *tautulli.TautulliPlaysByStreamResolution) string { return r.Response.Result },
		func(r *tautulli.TautulliPlaysByStreamResolution) *string { return r.Response.Message },
	)
}

// GetPlaysByTop10Platforms retrieves top 10 platforms by play count
func (c *TautulliClient) GetPlaysByTop10Platforms(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Platforms, error) {
	req := newAPIRequest("get_plays_by_top_10_platforms")
	addTimeRangeParams(req, timeRange, yAxis, userID, grouping)

	return executeAPIRequest(ctx, c, req,
		func(r *tautulli.TautulliPlaysByTop10Platforms) string { return r.Response.Result },
		func(r *tautulli.TautulliPlaysByTop10Platforms) *string { return r.Response.Message },
	)
}

// GetPlaysByTop10Users retrieves top 10 users by play count
func (c *TautulliClient) GetPlaysByTop10Users(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Users, error) {
	req := newAPIRequest("get_plays_by_top_10_users")
	addTimeRangeParams(req, timeRange, yAxis, userID, grouping)

	return executeAPIRequest(ctx, c, req,
		func(r *tautulli.TautulliPlaysByTop10Users) string { return r.Response.Result },
		func(r *tautulli.TautulliPlaysByTop10Users) *string { return r.Response.Message },
	)
}

// GetPlaysPerMonth retrieves monthly playback statistics
func (c *TautulliClient) GetPlaysPerMonth(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysPerMonth, error) {
	req := newAPIRequest("get_plays_per_month")
	addTimeRangeParams(req, timeRange, yAxis, userID, grouping)

	return executeAPIRequest(ctx, c, req,
		func(r *tautulli.TautulliPlaysPerMonth) string { return r.Response.Result },
		func(r *tautulli.TautulliPlaysPerMonth) *string { return r.Response.Message },
	)
}

// GetUserPlayerStats retrieves player statistics for a specific user
func (c *TautulliClient) GetItemUserStats(ctx context.Context, ratingKey string, grouping int) (*tautulli.TautulliItemUserStats, error) {
	req := newAPIRequest("get_item_user_stats").
		addParam("rating_key", ratingKey).
		addIntParamZero("grouping", grouping)

	return executeAPIRequest(ctx, c, req,
		func(r *tautulli.TautulliItemUserStats) string { return r.Response.Result },
		func(r *tautulli.TautulliItemUserStats) *string { return r.Response.Message },
	)
}

// ==========================================
// Priority 2: Library-Specific Analytics (4 endpoints)
// ==========================================

// GetLibrariesTable retrieves library data in table format with pagination and sorting
func (c *TautulliClient) GetStreamTypeByTop10Users(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliStreamTypeByTop10Users, error) {
	req := newAPIRequest("get_stream_type_by_top_10_users")
	addTimeRangeParams(req, timeRange, yAxis, userID, grouping)

	return executeAPIRequest(ctx, c, req,
		func(r *tautulli.TautulliStreamTypeByTop10Users) string { return r.Response.Result },
		func(r *tautulli.TautulliStreamTypeByTop10Users) *string { return r.Response.Message },
	)
}

// GetStreamTypeByTop10Platforms retrieves streaming method breakdown by top 10 platforms
func (c *TautulliClient) GetStreamTypeByTop10Platforms(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliStreamTypeByTop10Platforms, error) {
	req := newAPIRequest("get_stream_type_by_top_10_platforms")
	addTimeRangeParams(req, timeRange, yAxis, userID, grouping)

	return executeAPIRequest(ctx, c, req,
		func(r *tautulli.TautulliStreamTypeByTop10Platforms) string { return r.Response.Result },
		func(r *tautulli.TautulliStreamTypeByTop10Platforms) *string { return r.Response.Message },
	)
}

// Search performs a global search across Tautulli data
