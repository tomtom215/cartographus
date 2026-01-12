// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/models"
)

// ApproximateStatsResponse wraps approximate stats with API response metadata
type ApproximateStatsResponse struct {
	Stats        *database.ApproximateStats `json:"stats"`
	DataSketches bool                       `json:"datasketches_available"` // true if DataSketches extension is available
}

// ApproximateDistinctResponse contains approximate distinct count result
type ApproximateDistinctResponse struct {
	Column        string `json:"column"`
	Count         int64  `json:"count"`
	IsApproximate bool   `json:"is_approximate"` // true if DataSketches was used
}

// ApproximatePercentileResponse contains approximate percentile result
type ApproximatePercentileResponse struct {
	Column        string  `json:"column"`
	Percentile    float64 `json:"percentile"`
	Value         float64 `json:"value"`
	IsApproximate bool    `json:"is_approximate"` // true if DataSketches was used
}

// ApproximateStats returns approximate analytics metrics using DataSketches when available.
// Falls back to exact calculations when extension is unavailable.
//
// @Summary Get approximate analytics statistics
// @Description Returns approximate distinct counts (using HyperLogLog) and percentiles (using KLL sketches).
//
//	When DataSketches extension is available, queries are O(1) space complexity.
//	Falls back to exact calculations (COUNT DISTINCT, PERCENTILE_CONT) when unavailable.
//
// @Tags Analytics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date filter (RFC3339 format)"
// @Param end_date query string false "End date filter (RFC3339 format)"
// @Param users query string false "Comma-separated list of usernames to filter"
// @Param media_types query string false "Comma-separated list of media types (movie, episode, track)"
// @Success 200 {object} models.APIResponse{data=ApproximateStatsResponse} "Approximate statistics"
// @Failure 400 {object} models.APIResponse "Invalid parameters"
// @Failure 500 {object} models.APIResponse "Database error"
// @Failure 503 {object} models.APIResponse "Database unavailable"
// @Router /api/v1/analytics/approximate [get]
func (h *Handler) ApproximateStats(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) || !h.requireDB(w) {
		return
	}

	start := time.Now()

	// Parse filter parameters
	filter, err := parseApproximateStatsFilter(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	// Get approximate stats
	stats, err := h.db.GetApproximateStats(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get approximate statistics", err)
		return
	}

	response := ApproximateStatsResponse{
		Stats:        stats,
		DataSketches: h.db.IsDataSketchesAvailable(),
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   response,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// ApproximateDistinctCount returns approximate count of distinct values in a specified column.
// Uses HyperLogLog when DataSketches is available, falls back to exact COUNT(DISTINCT).
//
// @Summary Get approximate distinct count for a column
// @Description Returns approximate distinct count using HyperLogLog with ~2% error bound.
//
//	Falls back to exact COUNT(DISTINCT) when DataSketches is unavailable.
//
// @Tags Analytics
// @Accept json
// @Produce json
// @Param column query string true "Column name (username, title, ip_address, city, country, platform, player, rating_key, media_type)"
// @Param start_date query string false "Start date filter (RFC3339 format)"
// @Param end_date query string false "End date filter (RFC3339 format)"
// @Param users query string false "Comma-separated list of usernames to filter"
// @Param media_types query string false "Comma-separated list of media types"
// @Success 200 {object} models.APIResponse{data=ApproximateDistinctResponse} "Approximate distinct count"
// @Failure 400 {object} models.APIResponse "Invalid parameters"
// @Failure 500 {object} models.APIResponse "Database error"
// @Failure 503 {object} models.APIResponse "Database unavailable"
// @Router /api/v1/analytics/approximate/distinct [get]
func (h *Handler) ApproximateDistinctCount(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) || !h.requireDB(w) {
		return
	}

	start := time.Now()

	// Parse column parameter (required)
	column := r.URL.Query().Get("column")
	if column == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Query parameter 'column' is required", nil)
		return
	}

	// Parse filter parameters
	filter, err := parseApproximateStatsFilter(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	// Get approximate distinct count
	count, isApproximate, err := h.db.ApproximateDistinctCount(r.Context(), column, filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get distinct count", err)
		return
	}

	response := ApproximateDistinctResponse{
		Column:        column,
		Count:         count,
		IsApproximate: isApproximate,
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   response,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// ApproximatePercentile returns approximate percentile of a numeric column.
// Uses KLL sketches when DataSketches is available, falls back to exact PERCENTILE_CONT.
//
// @Summary Get approximate percentile for a column
// @Description Returns approximate percentile using KLL sketches.
//
//	Falls back to exact PERCENTILE_CONT when DataSketches is unavailable.
//
// @Tags Analytics
// @Accept json
// @Produce json
// @Param column query string true "Column name (duration, percent_complete, paused_counter)"
// @Param percentile query number true "Percentile value between 0 and 1 (e.g., 0.50 for median, 0.95 for p95)"
// @Param start_date query string false "Start date filter (RFC3339 format)"
// @Param end_date query string false "End date filter (RFC3339 format)"
// @Param users query string false "Comma-separated list of usernames to filter"
// @Param media_types query string false "Comma-separated list of media types"
// @Success 200 {object} models.APIResponse{data=ApproximatePercentileResponse} "Approximate percentile"
// @Failure 400 {object} models.APIResponse "Invalid parameters"
// @Failure 500 {object} models.APIResponse "Database error"
// @Failure 503 {object} models.APIResponse "Database unavailable"
// @Router /api/v1/analytics/approximate/percentile [get]
func (h *Handler) ApproximatePercentile(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) || !h.requireDB(w) {
		return
	}

	start := time.Now()

	// Parse column parameter (required)
	column := r.URL.Query().Get("column")
	if column == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Query parameter 'column' is required", nil)
		return
	}

	// Parse percentile parameter (required)
	percentileStr := r.URL.Query().Get("percentile")
	if percentileStr == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Query parameter 'percentile' is required", nil)
		return
	}
	percentile, err := strconv.ParseFloat(percentileStr, 64)
	if err != nil || percentile < 0 || percentile > 1 {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "percentile must be a number between 0 and 1", nil)
		return
	}

	// Parse filter parameters
	filter, err := parseApproximateStatsFilter(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	// Get approximate percentile
	value, isApproximate, err := h.db.ApproximatePercentile(r.Context(), column, percentile, filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get percentile", err)
		return
	}

	response := ApproximatePercentileResponse{
		Column:        column,
		Percentile:    percentile,
		Value:         value,
		IsApproximate: isApproximate,
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   response,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// parseApproximateStatsFilter parses filter parameters from request
func parseApproximateStatsFilter(r *http.Request) (database.ApproximateStatsFilter, error) {
	var filter database.ApproximateStatsFilter

	// Parse start_date
	if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
		startDate, err := time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			return filter, err
		}
		filter.StartDate = &startDate
	}

	// Parse end_date
	if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
		endDate, err := time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			return filter, err
		}
		filter.EndDate = &endDate
	}

	// Parse users (comma-separated)
	if usersStr := r.URL.Query().Get("users"); usersStr != "" {
		filter.Users = strings.Split(usersStr, ",")
	}

	// Parse media_types (comma-separated)
	if mediaTypesStr := r.URL.Query().Get("media_types"); mediaTypesStr != "" {
		filter.MediaTypes = strings.Split(mediaTypesStr, ",")
	}

	return filter, nil
}
