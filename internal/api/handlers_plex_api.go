// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// PlexBandwidthStatistics retrieves bandwidth usage statistics from Plex
//
// Endpoint: GET /api/v1/plex/statistics/bandwidth
//
// Query Parameters:
//   - timespan: Optional time aggregation period in seconds
//
// Response: PlexBandwidthResponse with device, account, and bandwidth data
//
// Errors:
//   - 503 Service Unavailable: Plex integration not enabled
//   - 500 Internal Server Error: Failed to fetch from Plex
func (h *Handler) PlexBandwidthStatistics(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check if Plex is enabled
	if h.sync == nil || !h.sync.IsPlexEnabled() {
		respondError(w, http.StatusServiceUnavailable, "PLEX_DISABLED", "Plex integration is not enabled", nil)
		return
	}

	// Parse optional timespan parameter
	var timespan *int
	if ts := r.URL.Query().Get("timespan"); ts != "" {
		if v, err := strconv.Atoi(ts); err == nil {
			timespan = &v
		}
	}

	// Fetch bandwidth statistics
	stats, err := h.sync.GetPlexBandwidthStatistics(r.Context(), timespan)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PLEX_ERROR", "Failed to fetch bandwidth statistics", err)
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

// PlexLibrarySections retrieves all library sections from Plex
//
// Endpoint: GET /api/v1/plex/library/sections
//
// Response: PlexLibrarySectionsResponse with all configured library sections
//
// Errors:
//   - 503 Service Unavailable: Plex integration not enabled
//   - 500 Internal Server Error: Failed to fetch from Plex
func (h *Handler) PlexLibrarySections(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check if Plex is enabled
	if h.sync == nil || !h.sync.IsPlexEnabled() {
		respondError(w, http.StatusServiceUnavailable, "PLEX_DISABLED", "Plex integration is not enabled", nil)
		return
	}

	// Fetch library sections
	sections, err := h.sync.GetPlexLibrarySections(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PLEX_ERROR", "Failed to fetch library sections", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   sections,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PlexLibrarySectionContent retrieves content from a specific library section
//
// Endpoint: GET /api/v1/plex/library/sections/{key}/all
//
// Path Parameters:
//   - key: Library section key (e.g., "1" for Movies)
//
// Query Parameters:
//   - start: Pagination start offset (optional)
//   - size: Number of items to return (optional)
//
// Response: PlexLibrarySectionContentResponse with media items
//
// Errors:
//   - 400 Bad Request: Missing section key
//   - 503 Service Unavailable: Plex integration not enabled
//   - 500 Internal Server Error: Failed to fetch from Plex
func (h *Handler) PlexLibrarySectionContent(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check if Plex is enabled
	if h.sync == nil || !h.sync.IsPlexEnabled() {
		respondError(w, http.StatusServiceUnavailable, "PLEX_DISABLED", "Plex integration is not enabled", nil)
		return
	}

	// Extract section key from URL path
	sectionKey := r.PathValue("key")
	if sectionKey == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "section key is required", nil)
		return
	}

	// Parse optional pagination parameters
	var startOffset, size *int
	if s := r.URL.Query().Get("start"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			startOffset = &v
		}
	}
	if s := r.URL.Query().Get("size"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			size = &v
		}
	}

	// Fetch section content
	content, err := h.sync.GetPlexLibrarySectionContent(r.Context(), sectionKey, startOffset, size)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PLEX_ERROR", "Failed to fetch section content", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   content,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PlexLibrarySectionRecentlyAdded retrieves recently added content from a library section
//
// Endpoint: GET /api/v1/plex/library/sections/{key}/recentlyAdded
//
// Path Parameters:
//   - key: Library section key (e.g., "1" for Movies)
//
// Query Parameters:
//   - size: Number of items to return (optional)
//
// Response: PlexLibrarySectionContentResponse with recently added items
//
// Errors:
//   - 400 Bad Request: Missing section key
//   - 503 Service Unavailable: Plex integration not enabled
//   - 500 Internal Server Error: Failed to fetch from Plex
func (h *Handler) PlexLibrarySectionRecentlyAdded(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check if Plex is enabled
	if h.sync == nil || !h.sync.IsPlexEnabled() {
		respondError(w, http.StatusServiceUnavailable, "PLEX_DISABLED", "Plex integration is not enabled", nil)
		return
	}

	// Extract section key from URL path
	sectionKey := r.PathValue("key")
	if sectionKey == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "section key is required", nil)
		return
	}

	// Parse optional size parameter
	var size *int
	if s := r.URL.Query().Get("size"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			size = &v
		}
	}

	// Fetch recently added content
	content, err := h.sync.GetPlexLibrarySectionRecentlyAdded(r.Context(), sectionKey, size)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PLEX_ERROR", "Failed to fetch recently added content", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   content,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PlexActivities retrieves current server activities from Plex
//
// Endpoint: GET /api/v1/plex/activities
//
// Response: PlexActivitiesResponse with active server tasks
//
// Errors:
//   - 503 Service Unavailable: Plex integration not enabled
//   - 500 Internal Server Error: Failed to fetch from Plex
func (h *Handler) PlexActivities(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check if Plex is enabled
	if h.sync == nil || !h.sync.IsPlexEnabled() {
		respondError(w, http.StatusServiceUnavailable, "PLEX_DISABLED", "Plex integration is not enabled", nil)
		return
	}

	// Fetch activities
	activities, err := h.sync.GetPlexActivities(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PLEX_ERROR", "Failed to fetch activities", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   activities,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PlexSessions retrieves active playback sessions from Plex
//
// Endpoint: GET /api/v1/plex/sessions
//
// Response: PlexSessionsResponse with active playback sessions including transcode details
//
// Use Cases:
//   - Real-time monitoring of active streams
//   - Transcode session monitoring with hardware acceleration detection
//   - Quality metrics (source vs delivered resolution/codec)
//
// Errors:
//   - 503 Service Unavailable: Plex integration not enabled
//   - 500 Internal Server Error: Failed to fetch from Plex
func (h *Handler) PlexSessions(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check if Plex is enabled
	if h.sync == nil || !h.sync.IsPlexEnabled() {
		respondError(w, http.StatusServiceUnavailable, "PLEX_DISABLED", "Plex integration is not enabled", nil)
		return
	}

	// Fetch active sessions
	sessions, err := h.sync.GetPlexSessions(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PLEX_ERROR", "Failed to fetch active sessions", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   sessions,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PlexIdentity retrieves server identity information from Plex
//
// Endpoint: GET /api/v1/plex/identity
//
// Response: Server identity with machine identifier, version, and name
//
// Use Cases:
//   - Server identification for multi-server setups
//   - Version checking and compatibility verification
//   - Server health monitoring
//
// Errors:
//   - 503 Service Unavailable: Plex integration not enabled
//   - 500 Internal Server Error: Failed to fetch from Plex
func (h *Handler) PlexIdentity(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check if Plex is enabled
	if h.sync == nil || !h.sync.IsPlexEnabled() {
		respondError(w, http.StatusServiceUnavailable, "PLEX_DISABLED", "Plex integration is not enabled", nil)
		return
	}

	// Fetch server identity
	identity, err := h.sync.GetPlexIdentity(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PLEX_ERROR", "Failed to fetch server identity", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   identity,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PlexMetadata retrieves detailed metadata for a specific media item
//
// Endpoint: GET /api/v1/plex/library/metadata/{ratingKey}
//
// Path Parameters:
//   - ratingKey: Unique identifier for the media item
//
// Response: Detailed metadata including media info, streams, and related items
//
// Use Cases:
//   - Standalone catalog browsing without Tautulli
//   - Media details page for analytics dashboard
//   - Content discovery and recommendations
//
// Errors:
//   - 400 Bad Request: Missing rating key
//   - 503 Service Unavailable: Plex integration not enabled
//   - 500 Internal Server Error: Failed to fetch from Plex
func (h *Handler) PlexMetadata(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check if Plex is enabled
	if h.sync == nil || !h.sync.IsPlexEnabled() {
		respondError(w, http.StatusServiceUnavailable, "PLEX_DISABLED", "Plex integration is not enabled", nil)
		return
	}

	// Extract rating key from URL path
	ratingKey := r.PathValue("ratingKey")
	if ratingKey == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "rating key is required", nil)
		return
	}

	// Fetch metadata
	metadata, err := h.sync.GetPlexMetadata(r.Context(), ratingKey)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PLEX_ERROR", "Failed to fetch metadata", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   metadata,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PlexDevices retrieves connected devices from Plex
//
// Endpoint: GET /api/v1/plex/devices
//
// Response: List of devices that have connected to this Plex server
//
// Use Cases:
//   - Device analytics and tracking
//   - Platform distribution analysis
//   - Client version monitoring
//
// Errors:
//   - 503 Service Unavailable: Plex integration not enabled
//   - 500 Internal Server Error: Failed to fetch from Plex
func (h *Handler) PlexDevices(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check if Plex is enabled
	if h.sync == nil || !h.sync.IsPlexEnabled() {
		respondError(w, http.StatusServiceUnavailable, "PLEX_DISABLED", "Plex integration is not enabled", nil)
		return
	}

	// Fetch devices
	devices, err := h.sync.GetPlexDevices(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PLEX_ERROR", "Failed to fetch devices", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   devices,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PlexAccounts retrieves user accounts from Plex
//
// Endpoint: GET /api/v1/plex/accounts
//
// Response: List of user accounts (including managed users and home users)
//
// Use Cases:
//   - Standalone user management without Tautulli
//   - User analytics and preference tracking
//   - Access control monitoring
//
// Errors:
//   - 503 Service Unavailable: Plex integration not enabled
//   - 500 Internal Server Error: Failed to fetch from Plex
func (h *Handler) PlexAccounts(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check if Plex is enabled
	if h.sync == nil || !h.sync.IsPlexEnabled() {
		respondError(w, http.StatusServiceUnavailable, "PLEX_DISABLED", "Plex integration is not enabled", nil)
		return
	}

	// Fetch accounts
	accounts, err := h.sync.GetPlexAccounts(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PLEX_ERROR", "Failed to fetch accounts", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   accounts,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PlexOnDeck retrieves on-deck content from Plex
//
// Endpoint: GET /api/v1/plex/library/onDeck
//
// Response: List of on-deck items (partially watched content)
//
// Use Cases:
//   - Continue watching recommendations
//   - User engagement tracking
//   - Homepage content suggestions
//
// Errors:
//   - 503 Service Unavailable: Plex integration not enabled
//   - 500 Internal Server Error: Failed to fetch from Plex
func (h *Handler) PlexOnDeck(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check if Plex is enabled
	if h.sync == nil || !h.sync.IsPlexEnabled() {
		respondError(w, http.StatusServiceUnavailable, "PLEX_DISABLED", "Plex integration is not enabled", nil)
		return
	}

	// Fetch on-deck content
	onDeck, err := h.sync.GetPlexOnDeck(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PLEX_ERROR", "Failed to fetch on-deck content", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   onDeck,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PlexPlaylists retrieves playlists from Plex
//
// Endpoint: GET /api/v1/plex/playlists
//
// Response: List of user playlists
//
// Use Cases:
//   - Playlist analytics
//   - User preference insights
//   - Content curation tracking
//
// Errors:
//   - 503 Service Unavailable: Plex integration not enabled
//   - 500 Internal Server Error: Failed to fetch from Plex
func (h *Handler) PlexPlaylists(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check if Plex is enabled
	if h.sync == nil || !h.sync.IsPlexEnabled() {
		respondError(w, http.StatusServiceUnavailable, "PLEX_DISABLED", "Plex integration is not enabled", nil)
		return
	}

	// Fetch playlists
	playlists, err := h.sync.GetPlexPlaylists(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PLEX_ERROR", "Failed to fetch playlists", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   playlists,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PlexSearch performs a search within a library section
//
// Endpoint: GET /api/v1/plex/library/sections/{key}/search
//
// Path Parameters:
//   - key: Library section key (e.g., "1" for Movies)
//
// Query Parameters:
//   - query: Search query string (required)
//   - type: Media type filter (optional, e.g., 1=movie, 4=episode)
//
// Response: Search results matching the query
//
// Use Cases:
//   - Content discovery
//   - Quick access to specific media
//   - Standalone search without Tautulli
//
// Errors:
//   - 400 Bad Request: Missing section key or query
//   - 503 Service Unavailable: Plex integration not enabled
//   - 500 Internal Server Error: Failed to fetch from Plex
func (h *Handler) PlexSearch(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check if Plex is enabled
	if h.sync == nil || !h.sync.IsPlexEnabled() {
		respondError(w, http.StatusServiceUnavailable, "PLEX_DISABLED", "Plex integration is not enabled", nil)
		return
	}

	// Extract section key from URL path
	sectionKey := r.PathValue("key")
	if sectionKey == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "section key is required", nil)
		return
	}

	// Get query parameter
	query := r.URL.Query().Get("query")
	if query == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "query parameter is required", nil)
		return
	}

	// Parse optional type parameter
	var mediaType *int
	if t := r.URL.Query().Get("type"); t != "" {
		if v, err := strconv.Atoi(t); err == nil {
			mediaType = &v
		}
	}

	// Perform search
	results, err := h.sync.GetPlexSearch(r.Context(), sectionKey, query, mediaType)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PLEX_ERROR", "Failed to search", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   results,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PlexTranscodeSessions retrieves active transcode sessions
//
// Endpoint: GET /api/v1/plex/transcode/sessions
//
// Response: List of active transcode sessions with detailed metrics
//
// Use Cases:
//   - Transcode monitoring dashboard
//   - Server load analysis
//   - Hardware acceleration tracking
//
// Errors:
//   - 503 Service Unavailable: Plex integration not enabled
//   - 500 Internal Server Error: Failed to fetch from Plex
func (h *Handler) PlexTranscodeSessions(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check if Plex is enabled
	if h.sync == nil || !h.sync.IsPlexEnabled() {
		respondError(w, http.StatusServiceUnavailable, "PLEX_DISABLED", "Plex integration is not enabled", nil)
		return
	}

	// Fetch transcode sessions (same as GetPlexSessions but focused on transcode data)
	sessions, err := h.sync.GetPlexTranscodeSessions(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PLEX_ERROR", "Failed to fetch transcode sessions", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   sessions,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PlexCancelTranscode cancels an active transcode session
//
// Endpoint: DELETE /api/v1/plex/transcode/sessions/{sessionKey}
//
// Path Parameters:
//   - sessionKey: Transcode session key to cancel
//
// Response: Success confirmation
//
// Use Cases:
//   - Server load management
//   - Terminating runaway transcodes
//   - Administrative control
//
// Errors:
//   - 400 Bad Request: Missing session key
//   - 503 Service Unavailable: Plex integration not enabled
//   - 500 Internal Server Error: Failed to cancel transcode
func (h *Handler) PlexCancelTranscode(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check if Plex is enabled
	if h.sync == nil || !h.sync.IsPlexEnabled() {
		respondError(w, http.StatusServiceUnavailable, "PLEX_DISABLED", "Plex integration is not enabled", nil)
		return
	}

	// Extract session key from URL path
	sessionKey := r.PathValue("sessionKey")
	if sessionKey == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "session key is required", nil)
		return
	}

	// Cancel transcode
	err := h.sync.CancelPlexTranscode(r.Context(), sessionKey)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PLEX_ERROR", "Failed to cancel transcode", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"message":    "Transcode session canceled",
			"sessionKey": sessionKey,
		},
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// PlexServerCapabilities retrieves comprehensive server capabilities from Plex
//
// Endpoint: GET /api/v1/plex/capabilities
//
// Response: Server capabilities including feature flags, transcoder support,
// Plex Pass status, and available directories
//
// Use Cases:
//   - Server feature detection
//   - Plex Pass subscription verification
//   - Transcoder capability checking
//   - Hardware acceleration availability
//   - Live TV/DVR support detection
//
// Errors:
//   - 503 Service Unavailable: Plex integration not enabled
//   - 500 Internal Server Error: Failed to fetch from Plex
func (h *Handler) PlexServerCapabilities(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check if Plex is enabled
	if h.sync == nil || !h.sync.IsPlexEnabled() {
		respondError(w, http.StatusServiceUnavailable, "PLEX_DISABLED", "Plex integration is not enabled", nil)
		return
	}

	// Fetch server capabilities
	capabilities, err := h.sync.GetPlexServerCapabilities(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PLEX_ERROR", "Failed to fetch server capabilities", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   capabilities,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}
