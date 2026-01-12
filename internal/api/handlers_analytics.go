// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/models"
)

// This file contains all custom analytics endpoints for the Cartographus dashboard.
// These handlers provide advanced analytics beyond basic Tautulli data, including
// trend analysis, geographic insights, user engagement metrics, and content statistics.
//
// Analytics Endpoints (11 total):
//   - AnalyticsTrends: Time-series playback trends with automatic interval detection
//   - AnalyticsGeographic: 14 parallel geographic queries (cities, countries, heatmaps)
//   - AnalyticsBinge: Binge watching pattern analysis
//   - AnalyticsBandwidth: Bandwidth and streaming quality metrics
//   - AnalyticsBitrate: Bitrate distribution and quality analysis
//   - AnalyticsWatchParties: Concurrent viewing (watch party) detection
//   - AnalyticsAbandonment: Content abandonment rate analysis
//   - AnalyticsUsers: Top users by various metrics
//   - AnalyticsPopular: Popular content ranking
//   - AnalyticsUserEngagement: Per-user engagement scoring
//   - AnalyticsComparative: Period-over-period comparison
//   - AnalyticsTemporalHeatmap: Geographic activity over time
//
// All analytics endpoints support the standard 14+ filter dimensions and include
// 5-minute caching for performance. Cache is invalidated on sync completion.
//
// Performance Characteristics:
//   - AnalyticsGeographic: 14 concurrent DB queries (~50ms total with parallelization)
//   - Simple analytics: Single query (~20ms typical)
//   - All endpoints: <100ms p95 with caching

// AnalyticsTrends returns playback count trends over time with automatic interval detection.
//
// Method: GET
// Path: /api/v1/analytics/trends
//
// The endpoint automatically selects the best time interval based on the date range:
//   - <7 days: hourly intervals
//   - 7-60 days: daily intervals
//   - 60-365 days: weekly intervals
//   - >365 days: monthly intervals
//
// Query Parameters: Standard filter dimensions (users, media_types, platforms, etc.)
//
// Response: TrendsResponse with PlaybackTrend array and selected interval string.
func (h *Handler) AnalyticsTrends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsTrends", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		trends, interval, err := h.db.GetPlaybackTrends(ctx, filter)
		if err != nil {
			return nil, err
		}
		if trends == nil {
			trends = []models.PlaybackTrend{}
		}
		return models.TrendsResponse{
			PlaybackTrends: trends,
			Interval:       interval,
		}, nil
	})
}

// AnalyticsGeographic handles geographic analytics requests with caching and parallelization
//
// checkCacheAndReturnIfHit checks cache and returns early if data is found
//
//nolint:gocyclo // Complexity is due to parallel query orchestration, inherent to purpose
func (h *Handler) checkCacheAndReturnIfHit(w http.ResponseWriter, cacheKey string, start time.Time) bool {
	if cached, found := h.cache.Get(cacheKey); found {
		if response, ok := cached.(models.GeographicResponse); ok {
			respondJSON(w, http.StatusOK, &models.APIResponse{
				Status: "success",
				Data:   response,
				Metadata: models.Metadata{
					Timestamp:   time.Now(),
					QueryTimeMS: time.Since(start).Milliseconds(),
				},
			})
			return true
		}
	}
	return false
}

// geoQuery represents a geographic analytics query to be executed in parallel
type geoQuery struct {
	name  string
	query func() (interface{}, error)
}

// assertSliceResult performs type assertion for slice results, returning empty slice if nil
func assertSliceResult[T any](result interface{}, name string) ([]T, error) {
	if result == nil {
		return []T{}, nil
	}
	typed, ok := result.([]T)
	if !ok {
		return nil, fmt.Errorf("type assertion failed for %s", name)
	}
	return typed, nil
}

// assertStructResult performs type assertion for struct results
func assertStructResult[T any](result interface{}, name string) (T, error) {
	var zero T
	if result == nil {
		return zero, nil
	}
	typed, ok := result.(T)
	if !ok {
		return zero, fmt.Errorf("type assertion failed for %s", name)
	}
	return typed, nil
}

// executeQueriesInParallel executes queries concurrently and returns results or first error
func executeQueriesInParallel(queries []geoQuery) ([]interface{}, error) {
	results := make([]interface{}, len(queries))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, q := range queries {
		wg.Add(1)
		go func(idx int, query geoQuery) {
			defer wg.Done()
			data, err := query.query()
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("failed to retrieve %s: %w", query.name, err)
				}
				mu.Unlock()
				return
			}
			results[idx] = data
		}(i, q)
	}

	wg.Wait()
	return results, firstErr
}

// executeParallelGeographicQueries executes all 14 geographic analytics queries in parallel
func (h *Handler) executeParallelGeographicQueries(ctx context.Context, filter database.LocationStatsFilter) (*models.GeographicResponse, error) {
	queries := []geoQuery{
		{"top cities", func() (interface{}, error) { return h.db.GetTopCities(ctx, filter, 10) }},
		{"top countries", func() (interface{}, error) { return h.db.GetTopCountries(ctx, filter, 10) }},
		{"media type distribution", func() (interface{}, error) { return h.db.GetMediaTypeDistribution(ctx, filter) }},
		{"viewing hours heatmap", func() (interface{}, error) { return h.db.GetViewingHoursHeatmap(ctx, filter) }},
		{"platform distribution", func() (interface{}, error) { return h.db.GetPlatformDistribution(ctx, filter) }},
		{"player distribution", func() (interface{}, error) { return h.db.GetPlayerDistribution(ctx, filter) }},
		{"content completion stats", func() (interface{}, error) { return h.db.GetContentCompletionStats(ctx, filter) }},
		{"transcode distribution", func() (interface{}, error) { return h.db.GetTranscodeDistribution(ctx, filter) }},
		{"resolution distribution", func() (interface{}, error) { return h.db.GetResolutionDistribution(ctx, filter) }},
		{"codec distribution", func() (interface{}, error) { return h.db.GetCodecDistribution(ctx, filter) }},
		{"library distribution", func() (interface{}, error) { return h.db.GetLibraryStats(ctx, filter) }},
		{"rating distribution", func() (interface{}, error) { return h.db.GetRatingDistribution(ctx, filter) }},
		{"duration stats", func() (interface{}, error) { return h.db.GetDurationStats(ctx, filter) }},
		{"year distribution", func() (interface{}, error) { return h.db.GetYearDistribution(ctx, filter, 10) }},
	}

	results, err := executeQueriesInParallel(queries)
	if err != nil {
		return nil, err
	}

	return h.buildGeographicResponse(results)
}

// buildGeographicResponse constructs GeographicResponse from query results with type assertions
func (h *Handler) buildGeographicResponse(results []interface{}) (*models.GeographicResponse, error) {
	topCities, err := assertSliceResult[models.CityStats](results[0], "top cities")
	if err != nil {
		return nil, err
	}
	topCountries, err := assertSliceResult[models.CountryStats](results[1], "top countries")
	if err != nil {
		return nil, err
	}
	mediaTypes, err := assertSliceResult[models.MediaTypeStats](results[2], "media types")
	if err != nil {
		return nil, err
	}
	heatmap, err := assertSliceResult[models.ViewingHoursHeatmap](results[3], "heatmap")
	if err != nil {
		return nil, err
	}
	platforms, err := assertSliceResult[models.PlatformStats](results[4], "platforms")
	if err != nil {
		return nil, err
	}
	players, err := assertSliceResult[models.PlayerStats](results[5], "players")
	if err != nil {
		return nil, err
	}
	completionStats, err := assertStructResult[models.ContentCompletionStats](results[6], "completion stats")
	if err != nil {
		return nil, err
	}
	transcodeDistribution, err := assertSliceResult[models.TranscodeStats](results[7], "transcode distribution")
	if err != nil {
		return nil, err
	}
	resolutionDistribution, err := assertSliceResult[models.ResolutionStats](results[8], "resolution distribution")
	if err != nil {
		return nil, err
	}
	codecDistribution, err := assertSliceResult[models.CodecStats](results[9], "codec distribution")
	if err != nil {
		return nil, err
	}
	libraryDistribution, err := assertSliceResult[models.LibraryStats](results[10], "library distribution")
	if err != nil {
		return nil, err
	}
	ratingDistribution, err := assertSliceResult[models.RatingStats](results[11], "rating distribution")
	if err != nil {
		return nil, err
	}
	durationStats, err := assertStructResult[models.DurationStats](results[12], "duration stats")
	if err != nil {
		return nil, err
	}
	yearDistribution, err := assertSliceResult[models.YearStats](results[13], "year distribution")
	if err != nil {
		return nil, err
	}

	return &models.GeographicResponse{
		TopCities:              topCities,
		TopCountries:           topCountries,
		MediaTypeDistribution:  mediaTypes,
		ViewingHoursHeatmap:    heatmap,
		PlatformDistribution:   platforms,
		PlayerDistribution:     players,
		ContentCompletionStats: completionStats,
		TranscodeDistribution:  transcodeDistribution,
		ResolutionDistribution: resolutionDistribution,
		CodecDistribution:      codecDistribution,
		LibraryDistribution:    libraryDistribution,
		RatingDistribution:     ratingDistribution,
		DurationStats:          durationStats,
		YearDistribution:       yearDistribution,
	}, nil
}

// cacheAndRespondSuccess stores the response in cache and returns JSON success response
func (h *Handler) cacheAndRespondSuccess(w http.ResponseWriter, cacheKey string, response *models.GeographicResponse, start time.Time) {
	// Store in cache
	h.cache.Set(cacheKey, *response)

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   *response,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// AnalyticsGeographic handles GET /api/v1/analytics/geographic requests
func (h *Handler) AnalyticsGeographic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// Check if database is available
	if h.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return
	}

	start := time.Now()
	filter := h.buildFilter(r)

	// Generate cache key from filter parameters
	cacheKey := cache.GenerateKey("AnalyticsGeographic", filter)

	// Check cache first - early return if hit
	if h.checkCacheAndReturnIfHit(w, cacheKey, start) {
		return
	}

	// Cache miss - execute parallel queries
	response, err := h.executeParallelGeographicQueries(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", err.Error(), err)
		return
	}

	// Cache and return success response
	h.cacheAndRespondSuccess(w, cacheKey, response, start)
}

// validateLimitParam validates and returns the limit parameter within bounds
func (h *Handler) validateLimitParam(r *http.Request, defaultLimit, maxLimit int) (int, error) {
	limit := getIntParam(r, "limit", defaultLimit)
	if limit < 1 || limit > maxLimit {
		return 0, fmt.Errorf("Limit must be between 1 and %d", maxLimit)
	}
	return limit, nil
}

// validateStringParam validates and returns a string parameter from allowed values
func validateStringParam(r *http.Request, paramName, defaultValue string, validValues map[string]bool) (string, error) {
	value := r.URL.Query().Get(paramName)
	if value == "" {
		return defaultValue, nil
	}
	if !validValues[value] {
		return "", fmt.Errorf("Invalid %s parameter", paramName)
	}
	return value, nil
}

// AnalyticsUsers handles user analytics requests
func (h *Handler) AnalyticsUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	limit, err := h.validateLimitParam(r, 10, h.config.API.MaxPageSize)
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteWithParamUserScoped(w, r, "AnalyticsUsers",
		func(ctx context.Context, filter database.LocationStatsFilter, param interface{}) (interface{}, error) {
			lmt, ok := param.(int)
			if !ok {
				return nil, fmt.Errorf("invalid parameter type: expected int")
			}
			topUsers, err := h.db.GetTopUsers(ctx, filter, lmt)
			if err != nil {
				return nil, err
			}
			if topUsers == nil {
				topUsers = []models.UserActivity{}
			}
			return models.UsersResponse{TopUsers: topUsers}, nil
		},
		limit,
	)
}

// AnalyticsBinge retrieves binge-watching analytics
func (h *Handler) AnalyticsBinge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsBinge", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return h.db.GetBingeAnalytics(ctx, filter)
	})
}

// AnalyticsBandwidth retrieves bandwidth usage analytics
func (h *Handler) AnalyticsBandwidth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsBandwidth", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return h.db.GetBandwidthAnalytics(ctx, filter)
	})
}

// AnalyticsBitrate retrieves bitrate and bandwidth analytics (v1.42 - Phase 2.2)
// Tracks bitrate at 3 levels (source, transcode, network) for network bottleneck identification
func (h *Handler) AnalyticsBitrate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsBitrate", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return h.db.GetBitrateAnalytics(ctx, filter)
	})
}

// AnalyticsPopular retrieves popular content analytics
func (h *Handler) AnalyticsPopular(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	limit, err := h.validateLimitParam(r, 10, 50)
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteWithParamUserScoped(w, r, "AnalyticsPopular",
		func(ctx context.Context, filter database.LocationStatsFilter, param interface{}) (interface{}, error) {
			lmt, ok := param.(int)
			if !ok {
				return nil, fmt.Errorf("invalid parameter type: expected int")
			}
			return h.db.GetPopularContent(ctx, filter, lmt)
		},
		limit,
	)
}

// AnalyticsWatchParties returns watch party detection analytics
func (h *Handler) AnalyticsWatchParties(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsWatchParties", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return h.db.GetWatchParties(ctx, filter)
	})
}

// AnalyticsUserEngagement retrieves user engagement analytics
func (h *Handler) AnalyticsUserEngagement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	limit, err := h.validateLimitParam(r, 10, 100)
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteWithParamUserScoped(w, r, "AnalyticsUserEngagement",
		func(ctx context.Context, filter database.LocationStatsFilter, param interface{}) (interface{}, error) {
			lmt, ok := param.(int)
			if !ok {
				return nil, fmt.Errorf("invalid parameter type: expected int")
			}
			return h.db.GetUserEngagement(ctx, filter, lmt)
		},
		limit,
	)
}

// AnalyticsAbandonment retrieves content abandonment and drop-off rate analytics
func (h *Handler) AnalyticsAbandonment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsAbandonment", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return h.db.GetContentAbandonmentAnalytics(ctx, filter)
	})
}

// AnalyticsComparative retrieves period-over-period comparison analytics
func (h *Handler) AnalyticsComparative(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	validTypes := map[string]bool{
		"week": true, "month": true, "quarter": true, "year": true, "custom": true,
	}
	comparisonType, err := validateStringParam(r, "comparison_type", "week", validTypes)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_PARAMETER",
			"Invalid comparison_type. Must be: week, month, quarter, year, or custom", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteWithParamUserScoped(w, r, "AnalyticsComparative",
		func(ctx context.Context, filter database.LocationStatsFilter, param interface{}) (interface{}, error) {
			cType, ok := param.(string)
			if !ok {
				return nil, fmt.Errorf("invalid parameter type: expected string")
			}
			return h.db.GetComparativeAnalytics(ctx, filter, cType)
		},
		comparisonType,
	)
}

// AnalyticsTemporalHeatmap handles temporal heatmap analytics requests
func (h *Handler) AnalyticsTemporalHeatmap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	validIntervals := map[string]bool{
		"hour": true, "day": true, "week": true, "month": true,
	}
	interval, err := validateStringParam(r, "interval", "day", validIntervals)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_PARAMETER",
			"Invalid interval. Must be: hour, day, week, or month", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteWithParamUserScoped(w, r, "AnalyticsTemporalHeatmap",
		func(ctx context.Context, filter database.LocationStatsFilter, param interface{}) (interface{}, error) {
			intv, ok := param.(string)
			if !ok {
				return nil, fmt.Errorf("invalid parameter type: expected string")
			}
			return h.db.GetTemporalHeatmap(ctx, filter, intv)
		},
		interval,
	)
}

// AnalyticsHardwareTranscode returns hardware transcode statistics (v1.43 - API Coverage Audit)
//
// Method: GET
// Path: /api/v1/analytics/hardware-transcode
//
// This endpoint provides GPU utilization insights including:
//   - Hardware vs software transcode ratio
//   - Decoder/encoder breakdown (NVIDIA NVENC, Intel Quick Sync, etc.)
//   - Full pipeline (HW decode + HW encode) statistics
//   - Throttling metrics
//
// Query Parameters: Standard filter dimensions (users, media_types, platforms, etc.)
//
// Response: HardwareTranscodeStats with decoder/encoder breakdown and percentages
func (h *Handler) AnalyticsHardwareTranscode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsHardwareTranscode", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return h.db.GetHardwareTranscodeStats(ctx, filter)
	})
}

// AnalyticsHDRContent returns HDR content playback statistics (v1.43 - API Coverage Audit)
//
// Method: GET
// Path: /api/v1/analytics/hdr-content
//
// This endpoint provides HDR content insights including:
//   - HDR vs SDR content ratio
//   - HDR format breakdown (HDR10, HDR10+, Dolby Vision, HLG)
//   - Color space distribution (bt2020nc, bt709, etc.)
//   - Color primaries distribution (bt2020, bt709, etc.)
//
// Query Parameters: Standard filter dimensions (users, media_types, platforms, etc.)
//
// Response: HDRContentStats with format breakdown and color metadata statistics
func (h *Handler) AnalyticsHDRContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsHDRContent", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return h.db.GetHDRContentStats(ctx, filter)
	})
}

// AnalyticsHardwareTranscodeTrends returns hardware transcode usage over time (v1.43 - API Coverage Audit)
//
// Method: GET
// Path: /api/v1/analytics/hardware-transcode/trends
//
// This endpoint provides daily trends of hardware transcode usage:
//   - Total sessions per day
//   - Hardware transcode sessions per day
//   - Full pipeline sessions per day
//   - Hardware percentage over time
//
// Query Parameters: Standard filter dimensions (users, media_types, platforms, etc.)
//
// Response: Array of HWTranscodeTrend with daily statistics for the last 30 days
func (h *Handler) AnalyticsHardwareTranscodeTrends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	executor := NewAnalyticsQueryExecutor(h)
	executor.ExecuteUserScoped(w, r, "AnalyticsHardwareTranscodeTrends", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		trends, err := h.db.GetHardwareTranscodeTrends(ctx, filter)
		if err != nil {
			return nil, err
		}
		if trends == nil {
			trends = []database.HWTranscodeTrend{}
		}
		return trends, nil
	})
}

// TautulliHomeStats handles Tautulli home statistics requests
//
// @Summary Get Tautulli homepage statistics
// @Description Returns pre-calculated analytics from Tautulli including top movies, TV shows, users, and platforms
// @Tags Tautulli Analytics
// @Accept json
// @Produce json
// @Param time_range query int false "Time range in days" default(30)
// @Param stats_type query string false "Statistics type (plays or duration)" default(plays)
// @Param stats_count query int false "Number of stats to return" default(10)
// @Success 200 {object} models.APIResponse{data=tautulli.TautulliHomeStats} "Home statistics retrieved successfully"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /api/v1/tautulli/home-stats [get]
