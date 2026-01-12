// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/models"
)

// AnalyticsResolutionMismatch handles resolution mismatch analytics requests
func (h *Handler) AnalyticsResolutionMismatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	start := time.Now()
	filter := h.buildFilter(r)

	// Generate cache key from filter parameters
	cacheKey := cache.GenerateKey("AnalyticsResolutionMismatch", filter)

	// Check cache first
	if cached, found := h.cache.Get(cacheKey); found {
		if response, ok := cached.(*models.ResolutionMismatchAnalytics); ok {
			respondJSON(w, http.StatusOK, &models.APIResponse{
				Status: "success",
				Data:   response,
				Metadata: models.Metadata{
					Timestamp:   time.Now(),
					QueryTimeMS: 0, // Cached response
				},
			})
			return
		}
	}

	// Check if database is available
	if h.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return
	}

	analytics, err := h.db.GetResolutionMismatchAnalytics(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve resolution mismatch analytics", err)
		return
	}

	// Cache the result
	h.cache.Set(cacheKey, analytics)

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   analytics,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// AnalyticsHDR handles HDR and dynamic range analytics requests
func (h *Handler) AnalyticsHDR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	start := time.Now()
	filter := h.buildFilter(r)

	// Generate cache key from filter parameters
	cacheKey := cache.GenerateKey("AnalyticsHDR", filter)

	// Check cache first
	if cached, found := h.cache.Get(cacheKey); found {
		if response, ok := cached.(*models.HDRAnalytics); ok {
			respondJSON(w, http.StatusOK, &models.APIResponse{
				Status: "success",
				Data:   response,
				Metadata: models.Metadata{
					Timestamp:   time.Now(),
					QueryTimeMS: 0, // Cached response
				},
			})
			return
		}
	}

	// Check if database is available
	if h.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return
	}

	analytics, err := h.db.GetHDRAnalytics(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve HDR analytics", err)
		return
	}

	// Cache the result
	h.cache.Set(cacheKey, analytics)

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   analytics,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// AnalyticsAudio handles audio quality analytics requests
func (h *Handler) AnalyticsAudio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	start := time.Now()
	filter := h.buildFilter(r)

	// Generate cache key from filter parameters
	cacheKey := cache.GenerateKey("AnalyticsAudio", filter)

	// Check cache first
	if cached, found := h.cache.Get(cacheKey); found {
		if response, ok := cached.(*models.AudioAnalytics); ok {
			respondJSON(w, http.StatusOK, &models.APIResponse{
				Status: "success",
				Data:   response,
				Metadata: models.Metadata{
					Timestamp:   time.Now(),
					QueryTimeMS: 0, // Cached response
				},
			})
			return
		}
	}

	// Check if database is available
	if h.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return
	}

	analytics, err := h.db.GetAudioAnalytics(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve audio analytics", err)
		return
	}

	// Cache the result
	h.cache.Set(cacheKey, analytics)

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   analytics,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// AnalyticsSubtitles handles subtitle usage analytics requests
func (h *Handler) AnalyticsSubtitles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	start := time.Now()
	filter := h.buildFilter(r)

	// Generate cache key from filter parameters
	cacheKey := cache.GenerateKey("AnalyticsSubtitles", filter)

	// Check cache first
	if cached, found := h.cache.Get(cacheKey); found {
		if response, ok := cached.(*models.SubtitleAnalytics); ok {
			respondJSON(w, http.StatusOK, &models.APIResponse{
				Status: "success",
				Data:   response,
				Metadata: models.Metadata{
					Timestamp:   time.Now(),
					QueryTimeMS: 0, // Cached response
				},
			})
			return
		}
	}

	// Check if database is available
	if h.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return
	}

	analytics, err := h.db.GetSubtitleAnalytics(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve subtitle analytics", err)
		return
	}

	// Cache the result
	h.cache.Set(cacheKey, analytics)

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   analytics,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// AnalyticsFrameRate handles frame rate analytics requests
func (h *Handler) AnalyticsFrameRate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	start := time.Now()
	filter := h.buildFilter(r)

	// Generate cache key from filter parameters
	cacheKey := cache.GenerateKey("AnalyticsFrameRate", filter)

	// Check cache first
	if cached, found := h.cache.Get(cacheKey); found {
		if response, ok := cached.(*models.FrameRateAnalytics); ok {
			respondJSON(w, http.StatusOK, &models.APIResponse{
				Status: "success",
				Data:   response,
				Metadata: models.Metadata{
					Timestamp:   time.Now(),
					QueryTimeMS: 0, // Cached response
				},
			})
			return
		}
	}

	// Check if database is available
	if h.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return
	}

	analytics, err := h.db.GetFrameRateAnalytics(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve frame rate analytics", err)
		return
	}

	// Cache the result
	h.cache.Set(cacheKey, analytics)

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   analytics,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// AnalyticsContainer handles container format analytics requests
func (h *Handler) AnalyticsContainer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	start := time.Now()
	filter := h.buildFilter(r)

	// Generate cache key from filter parameters
	cacheKey := cache.GenerateKey("AnalyticsContainer", filter)

	// Check cache first
	if cached, found := h.cache.Get(cacheKey); found {
		if response, ok := cached.(*models.ContainerAnalytics); ok {
			respondJSON(w, http.StatusOK, &models.APIResponse{
				Status: "success",
				Data:   response,
				Metadata: models.Metadata{
					Timestamp:   time.Now(),
					QueryTimeMS: 0, // Cached response
				},
			})
			return
		}
	}

	// Check if database is available
	if h.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return
	}

	analytics, err := h.db.GetContainerAnalytics(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve container analytics", err)
		return
	}

	// Cache the result
	h.cache.Set(cacheKey, analytics)

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   analytics,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// AnalyticsConnectionSecurity handles connection security analytics requests
func (h *Handler) AnalyticsConnectionSecurity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	start := time.Now()
	filter := h.buildFilter(r)

	// Generate cache key from filter parameters
	cacheKey := cache.GenerateKey("AnalyticsConnectionSecurity", filter)

	// Check cache first
	if cached, found := h.cache.Get(cacheKey); found {
		if response, ok := cached.(*models.ConnectionSecurityAnalytics); ok {
			respondJSON(w, http.StatusOK, &models.APIResponse{
				Status: "success",
				Data:   response,
				Metadata: models.Metadata{
					Timestamp:   time.Now(),
					QueryTimeMS: 0, // Cached response
				},
			})
			return
		}
	}

	// Check if database is available
	if h.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return
	}

	analytics, err := h.db.GetConnectionSecurityAnalytics(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve connection security analytics", err)
		return
	}

	// Cache the result
	h.cache.Set(cacheKey, analytics)

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   analytics,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// AnalyticsPausePatterns handles pause pattern analytics requests
func (h *Handler) AnalyticsPausePatterns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	start := time.Now()
	filter := h.buildFilter(r)

	// Generate cache key from filter parameters
	cacheKey := cache.GenerateKey("AnalyticsPausePatterns", filter)

	// Check cache first
	if cached, found := h.cache.Get(cacheKey); found {
		if response, ok := cached.(*models.PausePatternAnalytics); ok {
			respondJSON(w, http.StatusOK, &models.APIResponse{
				Status: "success",
				Data:   response,
				Metadata: models.Metadata{
					Timestamp:   time.Now(),
					QueryTimeMS: 0, // Cached response
				},
			})
			return
		}
	}

	// Check if database is available
	if h.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return
	}

	analytics, err := h.db.GetPausePatternAnalytics(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve pause pattern analytics", err)
		return
	}

	// Cache the result
	h.cache.Set(cacheKey, analytics)

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   analytics,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// AnalyticsLibrary handles library-specific analytics requests
func (h *Handler) AnalyticsLibrary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	start := time.Now()
	filter := h.buildFilter(r)

	// Get section_id from query parameter
	sectionIDStr := r.URL.Query().Get("section_id")
	if sectionIDStr == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "section_id parameter is required", nil)
		return
	}

	sectionID := 0
	if _, err := fmt.Sscanf(sectionIDStr, "%d", &sectionID); err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid section_id parameter", err)
		return
	}

	// Generate cache key from filter parameters
	cacheKey := cache.GenerateKey("AnalyticsLibrary", struct {
		SectionID int
		Filter    interface{}
	}{SectionID: sectionID, Filter: filter})

	// Check cache first
	if cached, found := h.cache.Get(cacheKey); found {
		if response, ok := cached.(*models.LibraryAnalytics); ok {
			respondJSON(w, http.StatusOK, &models.APIResponse{
				Status: "success",
				Data:   response,
				Metadata: models.Metadata{
					Timestamp:   time.Now(),
					QueryTimeMS: 0, // Cached response
				},
			})
			return
		}
	}

	// Check if database is available
	if h.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return
	}

	analytics, err := h.db.GetLibraryAnalytics(r.Context(), sectionID, filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve library analytics", err)
		return
	}

	// Cache the result
	h.cache.Set(cacheKey, analytics)

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   analytics,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// AnalyticsConcurrentStreams handles concurrent streams analytics requests
func (h *Handler) AnalyticsConcurrentStreams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// Validate interval parameter FIRST (validation errors = 400)
	interval := r.URL.Query().Get("interval")
	if interval != "" {
		validIntervals := map[string]bool{"hour": true, "day": true, "week": true, "month": true}
		if !validIntervals[interval] {
			respondError(w, http.StatusBadRequest, "VALIDATION_ERROR",
				"Invalid interval parameter: must be one of 'hour', 'day', 'week', 'month'", nil)
			return
		}
	}

	// Check if database is available AFTER validation (service errors = 503)
	if h.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return
	}

	start := time.Now()
	filter := h.buildFilter(r)

	// Generate cache key from filter parameters
	cacheKey := cache.GenerateKey("AnalyticsConcurrentStreams", filter)

	// Check cache first (only if cache is available)
	if h.cache != nil {
		if cached, found := h.cache.Get(cacheKey); found {
			if response, ok := cached.(*models.ConcurrentStreamsAnalytics); ok {
				respondJSON(w, http.StatusOK, &models.APIResponse{
					Status: "success",
					Data:   response,
					Metadata: models.Metadata{
						Timestamp:   time.Now(),
						QueryTimeMS: 0, // Cached response
					},
				})
				return
			}
		}
	}

	// Use a dedicated timeout context for this potentially slow query
	// This prevents client disconnection from canceling the query mid-execution
	// and ensures the query completes or times out gracefully
	queryCtx, queryCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()

	analytics, err := h.db.GetConcurrentStreamsAnalytics(queryCtx, filter)
	if err != nil {
		// Check for context cancellation errors
		if queryCtx.Err() == context.DeadlineExceeded {
			respondError(w, http.StatusGatewayTimeout, "QUERY_TIMEOUT", "Concurrent streams query timed out", err)
			return
		}
		if queryCtx.Err() == context.Canceled {
			respondError(w, http.StatusServiceUnavailable, "QUERY_CANCELED", "Concurrent streams query was canceled", err)
			return
		}
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve concurrent streams analytics", err)
		return
	}

	// Cache the result (only if cache is available)
	if h.cache != nil {
		h.cache.Set(cacheKey, analytics)
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   analytics,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}
