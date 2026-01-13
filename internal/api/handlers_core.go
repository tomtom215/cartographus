// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/goccy/go-json"
	ws "github.com/tomtom215/cartographus/internal/websocket"

	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// This file contains core API endpoints for the Cartographus application.
// These handlers provide essential data access for the frontend dashboard including
// statistics, playback history, location data, user information, and WebSocket connections.
//
// Endpoints in this file:
//   - Stats: Dashboard statistics and summary metrics
//   - Playbacks: Playback history with cursor-based pagination
//   - Locations: Geographic location data with filtering
//   - Users: Unique user list
//   - MediaTypes: Available media types
//   - ServerInfo: Server configuration and version information
//   - TriggerSync: Manual synchronization trigger
//   - WebSocket: Real-time update connection
//   - Login: JWT authentication
//
// All handlers follow a consistent pattern:
//  1. Method validation (GET/POST)
//  2. Parameter parsing and validation
//  3. Database query with context
//  4. JSON response with metadata

// requireMethod validates HTTP method and returns true if valid, false if error was sent
func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return false
	}
	return true
}

// requireDB checks database availability and returns true if available, false if error was sent
func (h *Handler) requireDB(w http.ResponseWriter) bool {
	if h.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return false
	}
	return true
}

// Stats returns dashboard statistics including total playbacks, unique locations,
// unique users, top countries, recent activity, and last sync time.
//
// Method: GET
// Path: /api/v1/stats
//
// Response:
//   - 200: Statistics retrieved successfully
//   - 405: Method not allowed (non-GET request)
//   - 500: Database error
//   - 503: Database not available
//
// The response includes query execution time in metadata for performance monitoring.
// Last sync time is populated from the sync manager if available.
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) || !h.requireDB(w) {
		return
	}

	start := time.Now()

	stats, err := h.db.GetStats(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve statistics", err)
		return
	}

	if h.sync != nil {
		lastSync := h.sync.LastSyncTime()
		if !lastSync.IsZero() {
			stats.LastSyncTime = &lastSync
		}
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

// Playbacks handles playback history requests with cursor-based pagination
//
// @Summary Get playback history
// @Description Returns paginated playback event history with cursor-based pagination for efficient large dataset handling. Supports both cursor-based (recommended) and offset-based (legacy) pagination.
// @Tags Core
// @Accept json
// @Produce json
// @Param limit query int false "Number of results per page (1-1000)" default(100) minimum(1) maximum(1000)
// @Param cursor query string false "Cursor for next page (from previous response's next_cursor). Use this instead of offset for efficient pagination."
// @Param offset query int false "LEGACY: Number of results to skip (0-1000000). Prefer cursor for large datasets." default(0) minimum(0) maximum(1000000)
// @Success 200 {object} models.APIResponse{data=models.PlaybacksResponse} "Playback events retrieved successfully with pagination info"
// @Failure 400 {object} models.APIResponse "Invalid parameters"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /playbacks [get]
func (h *Handler) Playbacks(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	params, err := h.parsePlaybacksParams(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), err)
		return
	}

	if !h.requireDB(w) {
		return
	}

	start := time.Now()
	h.handlePlaybacksPagination(w, r, params, start)
}

// parsePlaybacksParams extracts and validates playback request parameters
func (h *Handler) parsePlaybacksParams(r *http.Request) (*playbacksParams, error) {
	defaultPageSize, maxPageSize := h.getPageSizeConfig()

	limit := getIntParam(r, "limit", defaultPageSize)
	offset := getIntParam(r, "offset", 0)
	cursorParam := r.URL.Query().Get("cursor")

	// Validate request structure
	req := PlaybacksRequest{
		Limit:  limit,
		Offset: offset,
		Cursor: cursorParam,
	}
	if apiErr := validateRequest(&req); apiErr != nil {
		return nil, fmt.Errorf("%s: %s", apiErr.Code, apiErr.Message)
	}

	// Validate limit against dynamic config
	if limit > maxPageSize {
		return nil, fmt.Errorf("limit must be between 1 and %d", maxPageSize)
	}

	// Decode cursor if provided
	var cursor *models.PlaybackCursor
	if cursorParam != "" {
		var err error
		if cursor, err = decodeCursor(cursorParam); err != nil {
			return nil, fmt.Errorf("invalid cursor format: %w", err)
		}
	}

	return &playbacksParams{
		limit:       limit,
		offset:      offset,
		cursorParam: cursorParam,
		cursor:      cursor,
	}, nil
}

// playbacksParams holds validated playback request parameters
type playbacksParams struct {
	limit       int
	offset      int
	cursorParam string
	cursor      *models.PlaybackCursor
}

// handlePlaybacksPagination routes to appropriate pagination handler
func (h *Handler) handlePlaybacksPagination(w http.ResponseWriter, r *http.Request, params *playbacksParams, start time.Time) {
	if params.cursorParam != "" {
		h.handleCursorPagination(w, r, params.limit, params.cursor, start)
		return
	}

	if params.offset == 0 {
		h.handleFirstPagePagination(w, r, params.limit, start)
		return
	}

	h.handleOffsetPagination(w, r, params.limit, params.offset, start)
}

// handleCursorPagination handles cursor-based pagination
func (h *Handler) handleCursorPagination(w http.ResponseWriter, r *http.Request, limit int, cursor *models.PlaybackCursor, start time.Time) {
	events, nextCursor, hasMore, err := h.db.GetPlaybackEventsWithCursor(r.Context(), limit, cursor)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve playback events", err)
		return
	}

	response := buildPlaybacksResponse(events, limit, hasMore, nextCursor)
	h.respondWithPlaybacks(w, response, start)
}

// handleFirstPagePagination handles first page using cursor-based method internally
func (h *Handler) handleFirstPagePagination(w http.ResponseWriter, r *http.Request, limit int, start time.Time) {
	events, nextCursor, hasMore, err := h.db.GetPlaybackEventsWithCursor(r.Context(), limit, nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve playback events", err)
		return
	}

	response := buildPlaybacksResponse(events, limit, hasMore, nextCursor)
	h.respondWithPlaybacks(w, response, start)
}

// handleOffsetPagination handles legacy offset-based pagination
func (h *Handler) handleOffsetPagination(w http.ResponseWriter, r *http.Request, limit, offset int, start time.Time) {
	events, err := h.db.GetPlaybackEvents(r.Context(), limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve playback events", err)
		return
	}

	hasMore := len(events) == limit // Approximate for offset mode
	response := buildPlaybacksResponse(events, limit, hasMore, nil)
	h.respondWithPlaybacks(w, response, start)
}

// getPageSizeConfig returns page size configuration with safe defaults
func (h *Handler) getPageSizeConfig() (defaultPageSize, maxPageSize int) {
	defaultPageSize, maxPageSize = 100, 1000
	if h.config != nil {
		defaultPageSize = h.config.API.DefaultPageSize
		maxPageSize = h.config.API.MaxPageSize
	}
	return defaultPageSize, maxPageSize
}

// encodeCursor encodes a PlaybackCursor to a base64 string for API transport
func encodeCursor(cursor *models.PlaybackCursor) string {
	data, err := json.Marshal(cursor)
	if err != nil {
		// Should never happen with a simple struct, but return empty if it does
		return ""
	}
	return base64.URLEncoding.EncodeToString(data)
}

// decodeCursor decodes a base64 cursor string back to a PlaybackCursor
func decodeCursor(encoded string) (*models.PlaybackCursor, error) {
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 encoding: %w", err)
	}

	var cursor models.PlaybackCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, fmt.Errorf("invalid cursor JSON: %w", err)
	}

	return &cursor, nil
}

// buildPlaybacksResponse creates a standardized PlaybacksResponse from events and pagination info
func buildPlaybacksResponse(events []models.PlaybackEvent, limit int, hasMore bool, nextCursor *models.PlaybackCursor) models.PlaybacksResponse {
	if events == nil {
		events = []models.PlaybackEvent{}
	}

	var nextCursorStr *string
	if nextCursor != nil {
		encoded := encodeCursor(nextCursor)
		nextCursorStr = &encoded
	}

	return models.PlaybacksResponse{
		Events: events,
		Pagination: models.PaginationInfo{
			Limit:      limit,
			HasMore:    hasMore,
			NextCursor: nextCursorStr,
		},
	}
}

// respondWithPlaybacks sends a successful playback response with timing metadata
func (h *Handler) respondWithPlaybacks(w http.ResponseWriter, response models.PlaybacksResponse, start time.Time) {
	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   response,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// Locations handles location statistics requests with advanced filtering
//
// @Summary Get location statistics
// @Description Returns aggregated playback statistics by geographic location with support for 14+ filter dimensions
// @Tags Core
// @Accept json
// @Produce json
// @Param limit query int false "Maximum number of locations to return (1-1000)" default(100) minimum(1) maximum(1000)
// @Param start_date query string false "Start date filter (RFC3339 format)" example("2025-01-01T00:00:00Z")
// @Param end_date query string false "End date filter (RFC3339 format)" example("2025-12-31T23:59:59Z")
// @Param days query int false "Filter by last N days (1-3650, alternative to start_date)" minimum(1) maximum(3650)
// @Param users query string false "Comma-separated list of usernames" example("user1,user2")
// @Param media_types query string false "Comma-separated list of media types" example("movie,episode")
// @Param platforms query string false "Comma-separated list of platforms" example("Android,iOS")
// @Param players query string false "Comma-separated list of players" example("Plex for Android,Plex Web")
// @Param transcode_decisions query string false "Comma-separated transcode decisions" example("transcode,direct play")
// @Param video_resolutions query string false "Comma-separated video resolutions" example("1080,720")
// @Param video_codecs query string false "Comma-separated video codecs" example("h264,hevc")
// @Param audio_codecs query string false "Comma-separated audio codecs" example("aac,ac3")
// @Param libraries query string false "Comma-separated library names" example("Movies,TV Shows")
// @Param content_ratings query string false "Comma-separated content ratings" example("PG,PG-13,R")
// @Param years query string false "Comma-separated release years" example("2023,2024")
// @Param location_types query string false "Comma-separated location types" example("lan,wan")
// @Success 200 {object} models.APIResponse{data=[]models.LocationStats} "Location statistics retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid parameters"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /locations [get]
func (h *Handler) Locations(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	filter, err := h.parseLocationsFilter(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), err)
		return
	}

	if !h.requireDB(w) {
		return
	}

	start := time.Now()
	h.fetchAndRespondLocations(w, r, filter, start)
}

// parseLocationsFilter extracts and validates location filter parameters
func (h *Handler) parseLocationsFilter(r *http.Request) (database.LocationStatsFilter, error) {
	defaultPageSize, maxPageSize := h.getPageSizeConfig()

	limit := getIntParam(r, "limit", defaultPageSize)
	days := getIntParam(r, "days", 0)

	// Validate request structure
	req := LocationsRequest{
		Limit:     limit,
		Days:      days,
		StartDate: r.URL.Query().Get("start_date"),
		EndDate:   r.URL.Query().Get("end_date"),
		Users:     r.URL.Query().Get("users"),
	}
	if apiErr := validateRequest(&req); apiErr != nil {
		return database.LocationStatsFilter{}, fmt.Errorf("%s: %s", apiErr.Code, apiErr.Message)
	}

	// Validate limit against dynamic config
	if limit > maxPageSize {
		return database.LocationStatsFilter{}, fmt.Errorf("limit must be between 1 and %d", maxPageSize)
	}

	// Validate days when explicitly provided
	if err := h.validateDaysParam(r, days); err != nil {
		return database.LocationStatsFilter{}, err
	}

	// Build and populate filter
	filter := database.LocationStatsFilter{Limit: limit}
	if err := parseDateFilter(r, &filter); err != nil {
		return database.LocationStatsFilter{}, err
	}

	h.applyLocationFilters(r, &filter)
	return filter, nil
}

// validateDaysParam validates the days query parameter
func (h *Handler) validateDaysParam(r *http.Request, days int) error {
	daysStr := r.URL.Query().Get("days")
	if daysStr == "" {
		return nil
	}

	if days < 1 || days > 3650 {
		return fmt.Errorf("days must be between 1 and 3650")
	}
	return nil
}

// applyLocationFilters applies comma-separated filter parameters
func (h *Handler) applyLocationFilters(r *http.Request, filter *database.LocationStatsFilter) {
	if usersParam := r.URL.Query().Get("users"); usersParam != "" {
		filter.Users = parseCommaSeparated(usersParam)
	}

	if mediaTypesParam := r.URL.Query().Get("media_types"); mediaTypesParam != "" {
		filter.MediaTypes = parseCommaSeparated(mediaTypesParam)
	}
}

// fetchAndRespondLocations queries database and sends response
func (h *Handler) fetchAndRespondLocations(w http.ResponseWriter, r *http.Request, filter database.LocationStatsFilter, start time.Time) {
	locations, err := h.db.GetLocationStatsFiltered(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve location statistics", err)
		return
	}

	if locations == nil {
		locations = []models.LocationStats{}
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   locations,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// TriggerSync handles manual sync trigger requests
//
// @Summary Trigger data synchronization
// @Description Manually triggers a sync with Tautulli to fetch latest playback data. Requires JWT authentication.
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 202 {object} models.APIResponse "Sync triggered successfully"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 409 {object} models.APIResponse "Sync already in progress"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /sync [post]
func (h *Handler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	if h.sync == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Sync manager not available", nil)
		return
	}

	go func() {
		if err := h.sync.TriggerSync(); err != nil {
			logging.Error().Err(err).Msg("Manual sync failed")
		}
	}()

	respondJSON(w, http.StatusAccepted, &models.APIResponse{
		Status: "success",
		Data:   map[string]string{"message": "Sync triggered"},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// Users handles requests for list of unique users
//
// @Summary Get list of unique users
// @Description Returns a list of all unique usernames that have playback history
// @Tags Core
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]string} "List of usernames retrieved successfully"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /users [get]
func (h *Handler) Users(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) || !h.requireDB(w) {
		return
	}

	start := time.Now()

	users, err := h.db.GetUniqueUsers(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve users", err)
		return
	}

	if users == nil {
		users = []string{}
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   users,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// ServerInfo handles requests for server location information
//
// @Summary Get server location
// @Description Returns the physical location of the Plex server for visualization purposes (configured via SERVER_LATITUDE and SERVER_LONGITUDE environment variables)
// @Tags Core
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse{data=object{latitude=float64,longitude=float64,has_location=bool}} "Server location retrieved successfully"
// @Router /server-info [get]
func (h *Handler) ServerInfo(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	type ServerInfoResponse struct {
		Latitude    float64 `json:"latitude"`
		Longitude   float64 `json:"longitude"`
		HasLocation bool    `json:"has_location"`
	}

	var latitude, longitude float64
	var hasLocation bool

	if h.config != nil {
		latitude = h.config.Server.Latitude
		longitude = h.config.Server.Longitude
		hasLocation = latitude != 0.0 || longitude != 0.0
	}

	info := ServerInfoResponse{
		Latitude:    latitude,
		Longitude:   longitude,
		HasLocation: hasLocation,
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   info,
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// MediaTypes handles requests for list of unique media types
//
// @Summary Get list of unique media types
// @Description Returns a list of all unique media types (movie, episode, track, etc.) found in playback history
// @Tags Core
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]string} "List of media types retrieved successfully"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /media-types [get]
func (h *Handler) MediaTypes(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) || !h.requireDB(w) {
		return
	}

	start := time.Now()

	mediaTypes, err := h.db.GetUniqueMediaTypes(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve media types", err)
		return
	}

	if mediaTypes == nil {
		mediaTypes = []string{}
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   mediaTypes,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// AnalyticsTrends handles analytics trends requests

// parseDateFilter parses start_date/days and end_date from query parameters
func parseDateFilter(r *http.Request, filter *database.LocationStatsFilter) error {
	if err := parseStartDateFilter(r, filter); err != nil {
		return err
	}
	return parseEndDateFilter(r, filter)
}

// parseStartDateFilter parses start_date or days parameter
func parseStartDateFilter(r *http.Request, filter *database.LocationStatsFilter) error {
	startDateStr := r.URL.Query().Get("start_date")
	if startDateStr != "" {
		return parseStartDate(startDateStr, filter)
	}

	daysStr := r.URL.Query().Get("days")
	if daysStr != "" {
		return parseDaysFilter(r, filter)
	}

	return nil
}

// parseStartDate parses RFC3339 start date
func parseStartDate(startDateStr string, filter *database.LocationStatsFilter) error {
	startDate, err := time.Parse(time.RFC3339, startDateStr)
	if err != nil {
		return fmt.Errorf("invalid start_date format: %w", err)
	}
	filter.StartDate = &startDate
	return nil
}

// parseDaysFilter converts days parameter to start date
func parseDaysFilter(r *http.Request, filter *database.LocationStatsFilter) error {
	days := getIntParam(r, "days", 0)
	if days < 1 || days > 3650 {
		return nil
	}

	since := time.Now().AddDate(0, 0, -days)
	filter.StartDate = &since
	return nil
}

// parseEndDateFilter parses end_date parameter
func parseEndDateFilter(r *http.Request, filter *database.LocationStatsFilter) error {
	endDateStr := r.URL.Query().Get("end_date")
	if endDateStr == "" {
		return nil
	}

	endDate, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		return fmt.Errorf("invalid end_date format: %w", err)
	}
	filter.EndDate = &endDate
	return nil
}

// applyCommaSeparatedFilters applies comma-separated query parameters to filter fields
func applyCommaSeparatedFilters(r *http.Request, filter *database.LocationStatsFilter) {
	filterMap := map[string]*[]string{
		"users":               &filter.Users,
		"media_types":         &filter.MediaTypes,
		"platforms":           &filter.Platforms,
		"players":             &filter.Players,
		"transcode_decisions": &filter.TranscodeDecisions,
		"video_resolutions":   &filter.VideoResolutions,
		"video_codecs":        &filter.VideoCodecs,
		"audio_codecs":        &filter.AudioCodecs,
		"libraries":           &filter.Libraries,
		"content_ratings":     &filter.ContentRatings,
		"location_types":      &filter.LocationTypes,
		"server_ids":          &filter.ServerIDs, // v2.1: Multi-server support
	}

	for paramName, filterField := range filterMap {
		if paramValue := r.URL.Query().Get(paramName); paramValue != "" {
			*filterField = parseCommaSeparated(paramValue)
		}
	}

	// Handle years separately (integer slice)
	if yearsParam := r.URL.Query().Get("years"); yearsParam != "" {
		filter.Years = parseCommaSeparatedInts(yearsParam)
	}
}

func (h *Handler) buildFilter(r *http.Request) database.LocationStatsFilter {
	filter := database.LocationStatsFilter{
		Limit: 1000,
	}

	// Parse date filters (silently ignore errors for backward compatibility)
	//nolint:errcheck // Intentionally ignoring errors for backward compatibility
	_ = parseDateFilter(r, &filter)

	// Apply all comma-separated filters
	applyCommaSeparatedFilters(r, &filter)

	return filter
}

// WebSocket handles WebSocket connections
//
// @Summary Establish WebSocket connection
// @Description Establishes a WebSocket connection for real-time playback notifications and statistics updates
// @Tags Realtime
// @Accept json
// @Produce json
// @Success 101 {string} string "Switching Protocols"
// @Failure 400 {string} string "Bad Request"
// @Failure 503 {string} string "WebSocket hub not available"
// @Router /ws [get]
func (h *Handler) WebSocket(w http.ResponseWriter, r *http.Request) {
	// Check if WebSocket hub is available
	if h.wsHub == nil {
		logging.Warn().Msg("WebSocket connection rejected: hub not initialized")
		respondError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "WebSocket service unavailable", nil)
		return
	}

	upgrader := h.getUpgrader()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.Error().Err(err).Msg("WebSocket upgrade error")
		return
	}

	client := ws.NewClient(h.wsHub, conn)
	h.wsHub.Register <- client
	client.Start()
}

// Login handles user authentication requests
//
// @Summary Authenticate user
// @Description Authenticates user with username and password, returns JWT token in HTTP-only cookie
// @Tags Auth
// @Accept json
// @Produce json
// @Param credentials body object{username=string,password=string} true "Login credentials"
// @Success 200 {object} models.APIResponse{data=object{token=string,expires_at=string}} "Authentication successful"
// @Failure 400 {object} models.APIResponse "Invalid request body"
// @Failure 401 {object} models.APIResponse "Invalid credentials"
// @Failure 403 {object} models.APIResponse "Authentication disabled"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	req, err := h.parseAndValidateLoginRequest(w, r)
	if err != nil {
		return
	}

	if !h.validateAuthConfiguration(w) {
		return
	}

	if !h.authenticateCredentials(w, req) {
		return
	}

	h.generateAndSendToken(w, r, req)
}

// parseAndValidateLoginRequest parses and validates login request body
func (h *Handler) parseAndValidateLoginRequest(w http.ResponseWriter, r *http.Request) (*models.LoginRequest, error) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err)
		return nil, err
	}

	validationReq := LoginRequestValidation{
		Username:   req.Username,
		Password:   req.Password,
		RememberMe: req.RememberMe,
	}
	if apiErr := validateRequest(&validationReq); apiErr != nil {
		respondError(w, http.StatusBadRequest, apiErr.Code, apiErr.Message, nil)
		return nil, fmt.Errorf("%s: %s", apiErr.Code, apiErr.Message)
	}

	return &req, nil
}

// validateAuthConfiguration checks if JWT authentication is properly configured
func (h *Handler) validateAuthConfiguration(w http.ResponseWriter) bool {
	if h.config == nil || h.config.Security.AuthMode != "jwt" {
		respondError(w, http.StatusForbidden, "AUTH_DISABLED", "Authentication is disabled", nil)
		return false
	}

	if h.jwtManager == nil {
		respondError(w, http.StatusInternalServerError, "AUTH_NOT_CONFIGURED", "JWT manager not initialized", nil)
		return false
	}

	return true
}

// authenticateCredentials verifies username and password
func (h *Handler) authenticateCredentials(w http.ResponseWriter, req *models.LoginRequest) bool {
	if req.Username != h.config.Security.AdminUsername || req.Password != h.config.Security.AdminPassword {
		respondError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password", nil)
		return false
	}
	return true
}

// generateAndSendToken generates JWT token and sends response
func (h *Handler) generateAndSendToken(w http.ResponseWriter, r *http.Request, req *models.LoginRequest) {
	role := models.RoleAdmin
	userID := fmt.Sprintf("%s-001", req.Username)

	token, err := h.jwtManager.GenerateToken(req.Username, role)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "TOKEN_GENERATION_FAILED", "Failed to generate authentication token", err)
		return
	}

	expiresAt := time.Now().Add(h.config.Security.SessionTimeout)

	h.setAuthCookie(w, r, token, expiresAt)
	h.sendLoginResponse(w, token, expiresAt, req.Username, role, userID)
}

// setAuthCookie sets the authentication cookie
func (h *Handler) setAuthCookie(w http.ResponseWriter, r *http.Request, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})
}

// sendLoginResponse sends successful login response
func (h *Handler) sendLoginResponse(w http.ResponseWriter, token string, expiresAt time.Time, username string, role string, userID string) {
	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: models.LoginResponse{
			Token:     token,
			ExpiresAt: expiresAt,
			Username:  username,
			Role:      role,
			UserID:    userID,
		},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// ExportPlaybacksCSV exports playback events as CSV
//
// @Summary Export playback history as CSV
// @Description Exports complete playback history with all metadata to CSV format for external analysis
// @Tags Export
// @Accept json
// @Produce text/csv
// @Param limit query int false "Maximum number of records to export (1-100000)" default(10000) minimum(1) maximum(100000)
// @Success 200 {file} file "CSV file download"
// @Failure 400 {object} models.APIResponse "Invalid parameters"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /export/playbacks/csv [get]
//
//nolint:gocyclo // Complexity is due to handling many nullable CSV fields, logic is linear
