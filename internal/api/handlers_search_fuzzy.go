// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/models"
)

// FuzzySearchResponse contains fuzzy search results with metadata
type FuzzySearchResponse struct {
	Results     []database.FuzzySearchResult `json:"results"`
	Count       int                          `json:"count"`
	FuzzySearch bool                         `json:"fuzzy_search"` // true if RapidFuzz was used
}

// FuzzyUserSearchResponse contains fuzzy user search results with metadata
type FuzzyUserSearchResponse struct {
	Results     []database.UserSearchResult `json:"results"`
	Count       int                         `json:"count"`
	FuzzySearch bool                        `json:"fuzzy_search"` // true if RapidFuzz was used
}

// FuzzySearch handles fuzzy search requests against local DuckDB playback data.
// Uses RapidFuzz extension when available, falls back to exact LIKE matching.
//
// @Summary Search playback content with fuzzy matching
// @Description Performs fuzzy string matching search against playback history.
//
//	When RapidFuzz extension is available, returns similarity-scored results.
//	Falls back to exact LIKE matching when extension is unavailable.
//
// @Tags Search
// @Accept json
// @Produce json
// @Param q query string true "Search query (1-200 characters)"
// @Param min_score query int false "Minimum similarity score (0-100, default 70)"
// @Param limit query int false "Maximum results (1-100, default 20)"
// @Success 200 {object} models.APIResponse{data=FuzzySearchResponse} "Search results"
// @Failure 400 {object} models.APIResponse "Invalid parameters"
// @Failure 500 {object} models.APIResponse "Database error"
// @Failure 503 {object} models.APIResponse "Database unavailable"
// @Router /api/v1/search/fuzzy [get]
func (h *Handler) FuzzySearch(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) || !h.requireDB(w) {
		return
	}

	start := time.Now()

	// Parse query parameter (required)
	query := r.URL.Query().Get("q")
	if query == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Query parameter 'q' is required", nil)
		return
	}
	if len(query) > 200 {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Query parameter 'q' must be 200 characters or less", nil)
		return
	}

	// Parse min_score parameter (optional, default 70)
	minScore := 70
	if minScoreStr := r.URL.Query().Get("min_score"); minScoreStr != "" {
		parsed, err := strconv.Atoi(minScoreStr)
		if err != nil || parsed < 0 || parsed > 100 {
			respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "min_score must be an integer between 0 and 100", nil)
			return
		}
		minScore = parsed
	}

	// Parse limit parameter (optional, default 20)
	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed < 1 || parsed > 100 {
			respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "limit must be an integer between 1 and 100", nil)
			return
		}
		limit = parsed
	}

	// Perform fuzzy search
	results, err := h.db.FuzzySearchPlaybacks(r.Context(), query, minScore, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Search failed", err)
		return
	}

	// Handle nil results
	if results == nil {
		results = []database.FuzzySearchResult{}
	}

	response := FuzzySearchResponse{
		Results:     results,
		Count:       len(results),
		FuzzySearch: h.db.IsRapidFuzzAvailable(),
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

// FuzzySearchUsers handles fuzzy user search requests against local DuckDB data.
// Uses RapidFuzz extension when available, falls back to exact LIKE matching.
//
// @Summary Search users with fuzzy matching
// @Description Performs fuzzy string matching search against usernames and friendly names.
//
//	When RapidFuzz extension is available, returns similarity-scored results.
//	Falls back to exact LIKE matching when extension is unavailable.
//
// @Tags Search
// @Accept json
// @Produce json
// @Param q query string true "Search query (1-200 characters)"
// @Param min_score query int false "Minimum similarity score (0-100, default 70)"
// @Param limit query int false "Maximum results (1-100, default 20)"
// @Success 200 {object} models.APIResponse{data=FuzzyUserSearchResponse} "User search results"
// @Failure 400 {object} models.APIResponse "Invalid parameters"
// @Failure 500 {object} models.APIResponse "Database error"
// @Failure 503 {object} models.APIResponse "Database unavailable"
// @Router /api/v1/search/users [get]
func (h *Handler) FuzzySearchUsers(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) || !h.requireDB(w) {
		return
	}

	start := time.Now()

	// Parse query parameter (required)
	query := r.URL.Query().Get("q")
	if query == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Query parameter 'q' is required", nil)
		return
	}
	if len(query) > 200 {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Query parameter 'q' must be 200 characters or less", nil)
		return
	}

	// Parse min_score parameter (optional, default 70)
	minScore := 70
	if minScoreStr := r.URL.Query().Get("min_score"); minScoreStr != "" {
		parsed, err := strconv.Atoi(minScoreStr)
		if err != nil || parsed < 0 || parsed > 100 {
			respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "min_score must be an integer between 0 and 100", nil)
			return
		}
		minScore = parsed
	}

	// Parse limit parameter (optional, default 20)
	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed < 1 || parsed > 100 {
			respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "limit must be an integer between 1 and 100", nil)
			return
		}
		limit = parsed
	}

	// Perform fuzzy user search
	results, err := h.db.FuzzySearchUsers(r.Context(), query, minScore, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "User search failed", err)
		return
	}

	// Handle nil results
	if results == nil {
		results = []database.UserSearchResult{}
	}

	response := FuzzyUserSearchResponse{
		Results:     results,
		Count:       len(results),
		FuzzySearch: h.db.IsRapidFuzzAvailable(),
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
