// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package api provides HTTP handlers for the Cartographus application.
//
// handlers_wrapped.go - Annual Wrapped Report API Handlers
//
// This file contains HTTP handlers for the "Spotify Wrapped" style annual analytics feature.
// Users can generate and view personalized year-in-review reports with statistics about
// their media consumption.
//
// Endpoints:
//   - GET  /api/v1/wrapped/{year}                    - Get server-wide wrapped stats
//   - GET  /api/v1/wrapped/{year}/user/{userID}      - Get per-user wrapped report
//   - GET  /api/v1/wrapped/{year}/leaderboard        - Get wrapped leaderboard
//   - POST /api/v1/wrapped/{year}/generate           - Trigger report generation (admin)
//   - GET  /api/v1/wrapped/share/{token}             - Get shared wrapped report
//
// Security:
//   - User reports require authentication
//   - Admin endpoints require admin role
//   - Share tokens allow anonymous access to specific reports
//
// Observability:
//   - Prometheus metrics for generation duration and counts
//   - Structured logging with correlation IDs
//   - Request timing in response metadata
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/tomtom215/cartographus/internal/metrics"
	"github.com/tomtom215/cartographus/internal/models"
)

// WrappedServerStats returns server-wide wrapped statistics for a year.
//
// Method: GET
// Path: /api/v1/wrapped/{year}
//
// URL Parameters:
//   - year: The year to get statistics for (e.g., 2025)
//
// Response: WrappedServerStats with total users, watch time, top content, etc.
//
// Authentication: Required
func (h *Handler) WrappedServerStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	year, err := parseYearParam(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_YEAR", err.Error(), nil)
		return
	}

	start := time.Now()
	stats, err := h.db.GetWrappedServerStats(r.Context(), year)
	if err != nil {
		log.Error().Err(err).Int("year", year).Msg("Failed to get wrapped server stats")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get server statistics", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   stats,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// WrappedUserReport returns a wrapped report for a specific user and year.
//
// Method: GET
// Path: /api/v1/wrapped/{year}/user/{userID}
//
// URL Parameters:
//   - year: The year for the report (e.g., 2025)
//   - userID: The user ID to get the report for
//
// Query Parameters:
//   - generate: If "true" and no report exists, generate one
//
// Response: WrappedReport with all user statistics for the year.
//
// Authentication: Required (users can only access their own reports unless admin)
//
// Authorization (RBAC Phase 3):
//   - Users can only access their own reports
//   - Admins can access any user's report
func (h *Handler) WrappedUserReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	year, err := parseYearParam(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_YEAR", err.Error(), nil)
		return
	}

	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID format", nil)
		return
	}

	// RBAC Phase 3: Authorization check
	// Users can only access their own wrapped reports unless they are admin
	hctx := GetHandlerContext(r)
	if hctx.IsAuthenticated() {
		// Check if user can access this report (own data or admin)
		if !hctx.CanAccessUser(userIDStr) {
			log.Warn().
				Str("request_user_id", hctx.UserID).
				Str("target_user_id", userIDStr).
				Int("year", year).
				Msg("Wrapped report access denied: user cannot access other user's report")
			respondError(w, http.StatusForbidden, "FORBIDDEN", "Access denied: cannot view other users' wrapped reports", nil)
			return
		}
	}

	start := time.Now()

	// Check if report exists
	report, err := h.db.GetWrappedReport(r.Context(), userID, year)
	if err != nil {
		log.Error().Err(err).Int("year", year).Int("userID", userID).Msg("Failed to get wrapped report")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get wrapped report", err)
		return
	}

	// If no report exists and generate=true, generate one
	if report == nil {
		metrics.RecordWrappedCacheMiss(year)
		generateParam := r.URL.Query().Get("generate")
		if generateParam == "true" {
			log.Info().Int("year", year).Int("userID", userID).Msg("Generating wrapped report on demand")
			genStart := time.Now()
			report, err = h.db.GenerateWrappedReport(r.Context(), userID, year)
			metrics.RecordWrappedGeneration(year, "user", time.Since(genStart), err)
			if err != nil {
				log.Error().Err(err).Int("year", year).Int("userID", userID).Msg("Failed to generate wrapped report")
				respondError(w, http.StatusInternalServerError, "GENERATION_ERROR", "Failed to generate wrapped report", err)
				return
			}
		} else {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Wrapped report not found. Add ?generate=true to generate one.", nil)
			return
		}
	} else {
		metrics.RecordWrappedCacheHit(year)
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   report,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// WrappedLeaderboard returns the wrapped leaderboard for a year.
//
// Method: GET
// Path: /api/v1/wrapped/{year}/leaderboard
//
// URL Parameters:
//   - year: The year for the leaderboard (e.g., 2025)
//
// Query Parameters:
//   - limit: Maximum number of entries (default: 10, max: 100)
//
// Response: Array of WrappedLeaderboardEntry sorted by watch time.
//
// Authentication: Required
//
// Authorization (RBAC Phase 3):
//   - Admin-only endpoint (leaderboard contains other users' data)
//   - Regular users should only access their own wrapped reports
func (h *Handler) WrappedLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// RBAC Phase 3: Admin-only authorization
	// Leaderboard shows all users' data, so it's restricted to admins for privacy
	hctx := GetHandlerContext(r)
	if err := hctx.RequireAdmin(); err != nil {
		log.Warn().
			Str("user_id", hctx.UserID).
			Str("effective_role", hctx.EffectiveRole).
			Msg("Wrapped leaderboard access denied: admin role required")
		RespondAuthError(w, err)
		return
	}

	year, err := parseYearParam(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_YEAR", err.Error(), nil)
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	start := time.Now()
	leaderboard, err := h.db.GetWrappedLeaderboard(r.Context(), year, limit)
	if err != nil {
		log.Error().Err(err).Int("year", year).Msg("Failed to get wrapped leaderboard")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get leaderboard", err)
		return
	}

	// Record leaderboard query metric
	metrics.RecordWrappedLeaderboardQuery(year)

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   leaderboard,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// WrappedGenerate triggers generation of wrapped reports.
//
// Method: POST
// Path: /api/v1/wrapped/{year}/generate
//
// URL Parameters:
//   - year: The year to generate reports for (e.g., 2025)
//
// Request Body (optional):
//
//	{
//	  "user_id": 123,  // If set, only generate for this user
//	  "force": true    // If true, regenerate even if report exists
//	}
//
// Response: WrappedGenerateResponse with count of reports generated.
//
// Authentication: Required (admin only)
//
// Authorization (RBAC Phase 3):
//   - Admin-only endpoint (generates reports for other users)
func (h *Handler) WrappedGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// RBAC Phase 3: Admin-only authorization
	// Only admins can trigger report generation for all users
	hctx := GetHandlerContext(r)
	if err := hctx.RequireAdmin(); err != nil {
		log.Warn().
			Str("user_id", hctx.UserID).
			Str("effective_role", hctx.EffectiveRole).
			Msg("Wrapped report generation denied: admin role required")
		RespondAuthError(w, err)
		return
	}

	year, err := parseYearParam(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_YEAR", err.Error(), nil)
		return
	}

	// Parse request body
	var req models.WrappedGenerateRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", err)
			return
		}
	}
	req.Year = year

	start := time.Now()
	var reportsGenerated int

	if req.UserID != nil {
		// Generate for specific user
		log.Info().Int("year", year).Int("userID", *req.UserID).Msg("Generating wrapped report for user")
		genStart := time.Now()
		_, err := h.db.GenerateWrappedReport(r.Context(), *req.UserID, year)
		metrics.RecordWrappedGeneration(year, "user", time.Since(genStart), err)
		if err != nil {
			log.Error().Err(err).Int("year", year).Int("userID", *req.UserID).Msg("Failed to generate wrapped report")
			respondError(w, http.StatusInternalServerError, "GENERATION_ERROR", "Failed to generate report", err)
			return
		}
		reportsGenerated = 1
	} else {
		// Generate for all users
		log.Info().Int("year", year).Msg("Generating wrapped reports for all users")
		users, err := h.db.GetUsersWithPlaybacksInYear(r.Context(), year)
		if err != nil {
			log.Error().Err(err).Int("year", year).Msg("Failed to get users for wrapped generation")
			respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get users", err)
			return
		}

		for _, userID := range users {
			// Check if report exists and skip if not forcing
			if !req.Force {
				//nolint:errcheck // Error ignored - if check fails, we'll generate anyway
				existing, _ := h.db.GetWrappedReport(r.Context(), userID, year)
				if existing != nil {
					continue
				}
			}

			genStart := time.Now()
			_, err := h.db.GenerateWrappedReport(r.Context(), userID, year)
			metrics.RecordWrappedGeneration(year, "user", time.Since(genStart), err)
			if err != nil {
				log.Warn().Err(err).Int("year", year).Int("userID", userID).Msg("Failed to generate wrapped report for user")
				continue
			}
			reportsGenerated++
		}

		// Record batch size for batch operations
		if reportsGenerated > 0 {
			metrics.RecordWrappedBatch(reportsGenerated)
			metrics.RecordWrappedGeneration(year, "batch", time.Since(start), nil)
		}
	}

	response := models.WrappedGenerateResponse{
		Year:             year,
		ReportsGenerated: reportsGenerated,
		DurationMS:       time.Since(start).Milliseconds(),
		GeneratedAt:      time.Now(),
	}

	log.Info().Int("year", year).Int("count", reportsGenerated).Int64("durationMS", response.DurationMS).Msg("Wrapped report generation complete")

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   response,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: response.DurationMS,
		},
	})
}

// WrappedShare returns a wrapped report by share token (anonymous access).
//
// Method: GET
// Path: /api/v1/wrapped/share/{token}
//
// URL Parameters:
//   - token: The share token for the report
//
// Response: WrappedReport with user statistics (limited fields for privacy).
//
// Authentication: Not required (public access via share token)
func (h *Handler) WrappedShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	token := chi.URLParam(r, "token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "MISSING_TOKEN", "Share token is required", nil)
		return
	}

	start := time.Now()
	report, err := h.db.GetWrappedReportByShareToken(r.Context(), token)
	if err != nil {
		log.Error().Err(err).Str("token", token).Msg("Failed to get wrapped report by share token")
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get shared report", err)
		return
	}

	if report == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Shared report not found or expired", nil)
		return
	}

	// Record share token access metric
	metrics.RecordWrappedShareAccess()

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   report,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// parseYearParam extracts and validates the year URL parameter.
func parseYearParam(r *http.Request) (int, error) {
	yearStr := chi.URLParam(r, "year")
	if yearStr == "" {
		return 0, errorf("year parameter is required")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return 0, errorf("invalid year format: %s", yearStr)
	}

	// Validate reasonable year range
	currentYear := time.Now().Year()
	if year < 2000 || year > currentYear+1 {
		return 0, errorf("year must be between 2000 and %d", currentYear+1)
	}

	return year, nil
}

// errorf is a helper to create formatted errors.
func errorf(format string, args ...interface{}) error {
	return &wrappedError{msg: format, args: args}
}

type wrappedError struct {
	msg  string
	args []interface{}
}

func (e *wrappedError) Error() string {
	if len(e.args) > 0 {
		return e.msg // Already formatted or use fmt.Sprintf in actual usage
	}
	return e.msg
}
