// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// This file contains spatial analytics and export endpoints
// Total: 11 methods (2 export + 9 spatial)

// TileCoordinates represents parsed vector tile coordinates
type TileCoordinates struct {
	Z, X, Y int
}

// GeoJSON type definitions (extracted from inline definitions)
type GeoJSONGeometry struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

type GeoJSONProperties struct {
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
	AvgCompletion float64   `json:"avg_completion"`
	Country       string    `json:"country"`
	Region        *string   `json:"region,omitempty"`
	City          *string   `json:"city,omitempty"`
	PlaybackCount int       `json:"playback_count"`
	UniqueUsers   int       `json:"unique_users"`
}

type GeoJSONFeature struct {
	Type       string            `json:"type"`
	Geometry   GeoJSONGeometry   `json:"geometry"`
	Properties GeoJSONProperties `json:"properties"`
}

type GeoJSONFeatureCollection struct {
	Type     string           `json:"type"`
	Features []GeoJSONFeature `json:"features"`
}

// ParseTileCoordinates extracts and validates tile coordinates from URL path
// Expected format: /api/v1/tiles/{z}/{x}/{y}.pbf
func ParseTileCoordinates(path string) (*TileCoordinates, string, error) {
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 7 {
		return nil, "INVALID_PATH", fmt.Errorf("invalid tile path format")
	}

	z, err := strconv.Atoi(pathParts[4])
	if err != nil {
		return nil, "INVALID_ZOOM", fmt.Errorf("invalid zoom level")
	}

	x, err := strconv.Atoi(pathParts[5])
	if err != nil {
		return nil, "INVALID_X", fmt.Errorf("invalid X coordinate")
	}

	yStr := strings.TrimSuffix(pathParts[6], ".pbf")
	y, err := strconv.Atoi(yStr)
	if err != nil {
		return nil, "INVALID_Y", fmt.Errorf("invalid Y coordinate")
	}

	// Validate tile coordinates
	maxTile := int(math.Pow(2, float64(z))) - 1
	if z < 0 || z > 22 || x < 0 || x > maxTile || y < 0 || y > maxTile {
		return nil, "INVALID_COORDINATES", fmt.Errorf("tile coordinates out of range")
	}

	return &TileCoordinates{Z: z, X: x, Y: y}, "", nil
}

// optionalString returns the escaped CSV value for a string pointer, or empty string if nil.
func optionalString(s *string) string {
	if s == nil {
		return ""
	}
	return escapeCSV(*s)
}

// optionalInt returns the string representation of an int pointer, or empty string if nil.
func optionalInt(i *int) string {
	if i == nil {
		return ""
	}
	return strconv.Itoa(*i)
}

// optionalTime returns the RFC3339 formatted time pointer, or empty string if nil.
func optionalTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// parseExportFilter parses filter parameters for export endpoints.
// Returns the filter and any validation error message.
func parseExportFilter(r *http.Request) (database.LocationStatsFilter, string) {
	filter := database.LocationStatsFilter{}

	// Parse and validate date filter
	if errMsg := validateAndApplyDateFilter(r, &filter); errMsg != "" {
		return filter, errMsg
	}

	// Validate days range if provided
	if errMsg := validateDaysParam(r); errMsg != "" {
		return filter, errMsg
	}

	// Apply comma-separated filters
	applyCommaSeparatedFilters(r, &filter)

	return filter, ""
}

// validateAndApplyDateFilter parses date filter and returns error message if invalid
func validateAndApplyDateFilter(r *http.Request, filter *database.LocationStatsFilter) string {
	err := parseDateFilter(r, filter)
	if err == nil {
		return ""
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, "invalid start_date format") {
		return "Invalid start_date format. Use RFC3339 format"
	}
	if strings.Contains(errMsg, "invalid end_date format") {
		return "Invalid end_date format. Use RFC3339 format"
	}
	return errMsg
}

// validateDaysParam validates the days parameter if present
func validateDaysParam(r *http.Request) string {
	daysStr := r.URL.Query().Get("days")
	if daysStr == "" {
		return ""
	}

	days := getIntParam(r, "days", 0)
	if days < 1 || days > 3650 {
		return "Days must be between 1 and 3650 (10 years)"
	}
	return ""
}

// Note: applyCommaSeparatedFilters is defined in handlers_core.go

// buildCSVRow builds a CSV row from a PlaybackEvent using the helper functions.
// Note: watched_at is an alias for started_at (for E2E test compatibility)
func buildCSVRow(event *models.PlaybackEvent) string {
	return event.ID.String() + "," +
		escapeCSV(event.SessionKey) + "," +
		event.StartedAt.Format(time.RFC3339) + "," +
		optionalTime(event.StoppedAt) + "," +
		event.StartedAt.Format(time.RFC3339) + "," + // watched_at (alias for started_at)
		strconv.Itoa(event.UserID) + "," +
		escapeCSV(event.Username) + "," +
		escapeCSV(event.IPAddress) + "," +
		escapeCSV(event.MediaType) + "," +
		escapeCSV(event.Title) + "," +
		optionalString(event.ParentTitle) + "," +
		optionalString(event.GrandparentTitle) + "," +
		escapeCSV(event.Platform) + "," +
		escapeCSV(event.Player) + "," +
		escapeCSV(event.LocationType) + "," +
		strconv.Itoa(event.PercentComplete) + "," +
		strconv.Itoa(event.PausedCounter) + "," +
		optionalString(event.TranscodeDecision) + "," +
		optionalString(event.VideoResolution) + "," +
		optionalString(event.VideoCodec) + "," +
		optionalString(event.AudioCodec) + "," +
		optionalInt(event.SectionID) + "," +
		optionalString(event.LibraryName) + "," +
		optionalString(event.ContentRating) + "," +
		optionalInt(event.PlayDuration) + "," +
		optionalInt(event.Year) + "," +
		event.CreatedAt.Format(time.RFC3339) + "\n"
}

func (h *Handler) ExportPlaybacksCSV(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	// Parse parameters
	limit := getIntParam(r, "limit", 10000)
	offset := getIntParam(r, "offset", 0)

	// Use validator for struct validation
	req := ExportPlaybacksCSVRequest{
		Limit:  limit,
		Offset: offset,
	}
	if apiErr := validateRequest(&req); apiErr != nil {
		respondError(w, http.StatusBadRequest, apiErr.Code, apiErr.Message, nil)
		return
	}

	// Check if database is available AFTER validation (service errors = 503)
	if !h.requireDB(w) {
		return
	}

	events, err := h.db.GetPlaybackEvents(r.Context(), limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve playback events", err)
		return
	}

	// Set headers for CSV download
	timestamp := time.Now().Format("20060102-150405")
	filename := "cartographus-playbacks-" + timestamp + ".csv"
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Cache-Control", "no-cache")

	// Write CSV header
	// Note: watched_at is an alias for started_at (for E2E test compatibility)
	header := "id,session_key,started_at,stopped_at,watched_at,user_id,username,ip_address,media_type,title,parent_title,grandparent_title,platform,player,location_type,percent_complete,paused_counter,transcode_decision,video_resolution,video_codec,audio_codec,section_id,library_name,content_rating,play_duration,year,created_at\n"
	if _, err := w.Write([]byte(header)); err != nil {
		logging.Error().Err(err).Msg("Failed to write CSV header")
		return
	}

	// Write CSV rows using helper function (use index to avoid copying 648-byte structs)
	for i := range events {
		if _, err := w.Write([]byte(buildCSVRow(&events[i]))); err != nil {
			logging.Error().Err(err).Msg("Failed to write CSV row")
			return
		}
	}
}

// buildGeoJSONFeature creates a GeoJSON feature from a location stat
func buildGeoJSONFeature(loc *models.LocationStats) GeoJSONFeature {
	return GeoJSONFeature{
		Type: "Feature",
		Geometry: GeoJSONGeometry{
			Type:        "Point",
			Coordinates: []float64{loc.Longitude, loc.Latitude},
		},
		Properties: GeoJSONProperties{
			Country:       loc.Country,
			Region:        loc.Region,
			City:          loc.City,
			PlaybackCount: loc.PlaybackCount,
			UniqueUsers:   loc.UniqueUsers,
			FirstSeen:     loc.FirstSeen,
			LastSeen:      loc.LastSeen,
			AvgCompletion: loc.AvgCompletion,
		},
	}
}

// buildGeoJSONCollection creates a GeoJSON FeatureCollection from location stats
func buildGeoJSONCollection(locations []models.LocationStats) GeoJSONFeatureCollection {
	features := make([]GeoJSONFeature, 0, len(locations))
	for i := range locations {
		features = append(features, buildGeoJSONFeature(&locations[i]))
	}

	return GeoJSONFeatureCollection{
		Type:     "FeatureCollection",
		Features: features,
	}
}

// setGeoJSONDownloadHeaders sets appropriate headers for GeoJSON file download
func setGeoJSONDownloadHeaders(w http.ResponseWriter) {
	timestamp := time.Now().Format("20060102-150405")
	filename := "cartographus-locations-" + timestamp + ".geojson"
	w.Header().Set("Content-Type", "application/geo+json")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Cache-Control", "no-cache")
}

// ExportLocationsGeoJSON exports location statistics as GeoJSON
//
// @Summary Export locations as GeoJSON
// @Description Exports location statistics in GeoJSON format for use in GIS applications and mapping tools
// @Tags Export
// @Accept json
// @Produce application/geo+json
// @Success 200 {file} file "GeoJSON file download"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /export/locations/geojson [get]
func (h *Handler) ExportLocationsGeoJSON(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) || !h.requireDB(w) {
		return
	}

	filter := h.buildFilter(r)

	locations, err := h.db.GetLocationStatsFiltered(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve location statistics", err)
		return
	}

	featureCollection := buildGeoJSONCollection(locations)
	setGeoJSONDownloadHeaders(w)

	// Write GeoJSON (headers already sent, can only log errors)
	if err := json.NewEncoder(w).Encode(featureCollection); err != nil {
		logging.Error().Err(err).Msg("Failed to encode GeoJSON")
	}
}

func (h *Handler) SpatialHexagons(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	// Validate resolution parameter using existing validator
	resParams, err := ValidateResolution(r, 7)
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	// Additional validation using struct validator for date filters
	req := SpatialHexagonsRequest{
		Resolution: resParams.Resolution,
		StartDate:  r.URL.Query().Get("start_date"),
		EndDate:    r.URL.Query().Get("end_date"),
	}
	if apiErr := validateRequest(&req); apiErr != nil {
		respondError(w, http.StatusBadRequest, apiErr.Code, apiErr.Message, nil)
		return
	}

	// Execute query with caching
	executor := NewSpatialQueryExecutor(h)
	executor.ExecuteWithCache(w, r, "SpatialHexagons",
		func(ctx context.Context, filter database.LocationStatsFilter, params interface{}) (interface{}, error) {
			resolution := params.(*ResolutionParams).Resolution
			hexagons, err := h.db.GetH3AggregatedHexagons(ctx, filter, resolution)
			if err != nil {
				return nil, err
			}
			if hexagons == nil {
				return []models.H3HexagonStats{}, nil
			}
			return hexagons, nil
		},
		resParams,
		resParams,
	)
}

// ServerLocation holds server coordinates for arc calculations
type ServerLocation struct {
	Lat, Lon float64
}

// validateServerLocation checks if server location is configured
func (h *Handler) validateServerLocation() (*ServerLocation, error) {
	serverLat := h.config.Server.Latitude
	serverLon := h.config.Server.Longitude
	if serverLat == 0.0 && serverLon == 0.0 {
		return nil, fmt.Errorf("server location not configured (set SERVER_LATITUDE and SERVER_LONGITUDE)")
	}
	return &ServerLocation{Lat: serverLat, Lon: serverLon}, nil
}

// SpatialArcs returns distance-weighted user→server connection arcs
//
// @Summary Get distance-weighted connection arcs
// @Description Returns playback locations with geodesic distances and arc weights for visualization
// @Tags Spatial Analytics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param days query int false "Number of days to include (alternative to start_date)" minimum(1) maximum(3650)
// @Param users query string false "Comma-separated usernames"
// @Param media_types query string false "Comma-separated media types"
// @Param platforms query string false "Comma-separated platforms"
// @Param players query string false "Comma-separated players"
// @Success 200 {object} models.APIResponse{data=[]models.ArcStats} "Distance-weighted arcs retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid parameters"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /api/v1/spatial/arcs [get]
func (h *Handler) SpatialArcs(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	serverLoc, err := h.validateServerLocation()
	if err != nil {
		respondError(w, http.StatusBadRequest, "CONFIGURATION_ERROR", err.Error(), nil)
		return
	}

	executor := NewSpatialQueryExecutor(h)
	executor.ExecuteWithCache(w, r, "SpatialArcs",
		func(ctx context.Context, filter database.LocationStatsFilter, params interface{}) (interface{}, error) {
			loc := params.(*ServerLocation)
			arcs, err := h.db.GetDistanceWeightedArcs(ctx, filter, loc.Lat, loc.Lon)
			if err != nil {
				return nil, err
			}
			if arcs == nil {
				return []models.ArcStats{}, nil
			}
			return arcs, nil
		},
		serverLoc,
		serverLoc,
	)
}

// SpatialViewport returns locations within a geographic bounding box (100x faster with R-tree)
//
// @Summary Get locations in viewport
// @Description Returns playback locations within a bounding box using spatial index for fast pan/zoom
// @Tags Spatial Analytics
// @Accept json
// @Produce json
// @Param west query number true "Western longitude boundary" minimum(-180) maximum(180)
// @Param south query number true "Southern latitude boundary" minimum(-90) maximum(90)
// @Param east query number true "Eastern longitude boundary" minimum(-180) maximum(180)
// @Param north query number true "Northern latitude boundary" minimum(-90) maximum(90)
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param days query int false "Number of days to include (alternative to start_date)" minimum(1) maximum(3650)
// @Param users query string false "Comma-separated usernames"
// @Param media_types query string false "Comma-separated media types"
// @Param platforms query string false "Comma-separated platforms"
// @Param players query string false "Comma-separated players"
// @Success 200 {object} models.APIResponse{data=[]models.LocationStats} "Viewport locations retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid parameters"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /api/v1/spatial/viewport [get]
func (h *Handler) SpatialViewport(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	// Validate bounding box parameters using validator
	bbox, err := ValidateBoundingBox(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	// Additional validation using struct validator for consistency
	req := SpatialViewportRequest{
		West:      bbox.West,
		South:     bbox.South,
		East:      bbox.East,
		North:     bbox.North,
		StartDate: r.URL.Query().Get("start_date"),
		EndDate:   r.URL.Query().Get("end_date"),
	}
	if apiErr := validateRequest(&req); apiErr != nil {
		respondError(w, http.StatusBadRequest, apiErr.Code, apiErr.Message, nil)
		return
	}

	// Execute query with caching
	executor := NewSpatialQueryExecutor(h)
	executor.ExecuteWithCache(w, r, "SpatialViewport",
		func(ctx context.Context, filter database.LocationStatsFilter, params interface{}) (interface{}, error) {
			bbox := params.(*BoundingBoxParams)
			locations, err := h.db.GetLocationsInViewport(ctx, filter, bbox.West, bbox.South, bbox.East, bbox.North)
			if err != nil {
				return nil, err
			}
			if locations == nil {
				return []models.LocationStats{}, nil
			}
			return locations, nil
		},
		bbox,
		bbox,
	)
}

// TemporalDensityParams holds parameters for temporal density queries
type TemporalDensityParams struct {
	Interval   string
	Resolution int
}

// validateTemporalDensityParams validates interval and resolution parameters
func validateTemporalDensityParams(r *http.Request) (*TemporalDensityParams, error) {
	// Parse and validate interval (default "hour")
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "hour"
	}
	if err := ValidateInterval(interval); err != nil {
		return nil, err
	}

	// Validate resolution parameter
	resParams, err := ValidateResolution(r, 7)
	if err != nil {
		return nil, err
	}

	// Additional struct validation for date filters
	req := SpatialTemporalDensityRequest{
		Interval:   interval,
		Resolution: resParams.Resolution,
		StartDate:  r.URL.Query().Get("start_date"),
		EndDate:    r.URL.Query().Get("end_date"),
	}
	if apiErr := validateRequest(&req); apiErr != nil {
		return nil, fmt.Errorf("%s: %s", apiErr.Code, apiErr.Message)
	}

	return &TemporalDensityParams{
		Interval:   interval,
		Resolution: resParams.Resolution,
	}, nil
}

// SpatialTemporalDensity returns temporal-spatial playback density with rolling aggregations
//
// @Summary Get temporal-spatial density
// @Description Returns playback density over time and space with rolling averages for smooth animation
// @Tags Spatial Analytics
// @Accept json
// @Produce json
// @Param interval query string false "Time interval (hour, day, week, month)" default(hour)
// @Param resolution query int false "H3 resolution (6=country, 7=city, 8=neighborhood)" default(7) minimum(6) maximum(8)
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param days query int false "Number of days to include (alternative to start_date)" minimum(1) maximum(3650)
// @Param users query string false "Comma-separated usernames"
// @Param media_types query string false "Comma-separated media types"
// @Param platforms query string false "Comma-separated platforms"
// @Param players query string false "Comma-separated players"
// @Success 200 {object} models.APIResponse{data=[]models.TemporalSpatialPoint} "Temporal-spatial density retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid parameters"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /api/v1/spatial/temporal-density [get]
func (h *Handler) SpatialTemporalDensity(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	params, err := validateTemporalDensityParams(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	executor := NewSpatialQueryExecutor(h)
	executor.ExecuteWithCache(w, r, "SpatialTemporalDensity",
		func(ctx context.Context, filter database.LocationStatsFilter, p interface{}) (interface{}, error) {
			tdParams := p.(*TemporalDensityParams)
			points, err := h.db.GetTemporalSpatialDensity(ctx, filter, tdParams.Interval, tdParams.Resolution)
			if err != nil {
				return nil, err
			}
			if points == nil {
				return []models.TemporalSpatialPoint{}, nil
			}
			return points, nil
		},
		params,
		params,
	)
}

// SpatialNearby returns locations within a specified radius of a point
//
// @Summary Get nearby locations
// @Description Returns playback locations within a specified radius using spatial index for fast proximity search
// @Tags Spatial Analytics
// @Accept json
// @Produce json
// @Param lat query number true "Center latitude" minimum(-90) maximum(90)
// @Param lon query number true "Center longitude" minimum(-180) maximum(180)
// @Param radius query number false "Search radius in kilometers" default(100) minimum(1) maximum(20000)
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param days query int false "Number of days to include (alternative to start_date)" minimum(1) maximum(3650)
// @Param users query string false "Comma-separated usernames"
// @Param media_types query string false "Comma-separated media types"
// @Param platforms query string false "Comma-separated platforms"
// @Param players query string false "Comma-separated players"
// @Success 200 {object} models.APIResponse{data=[]models.LocationStats} "Nearby locations retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid parameters"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /api/v1/spatial/nearby [get]
func (h *Handler) SpatialNearby(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	// Validate coordinates and radius using existing validator
	coords, err := ValidateCoordinates(r, true)
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	// Additional validation using struct validator for date filters
	req := SpatialNearbyRequest{
		Lat:       coords.Lat,
		Lon:       coords.Lon,
		Radius:    coords.Radius,
		StartDate: r.URL.Query().Get("start_date"),
		EndDate:   r.URL.Query().Get("end_date"),
	}
	if apiErr := validateRequest(&req); apiErr != nil {
		respondError(w, http.StatusBadRequest, apiErr.Code, apiErr.Message, nil)
		return
	}

	// Execute query with caching
	executor := NewSpatialQueryExecutor(h)
	executor.ExecuteWithCache(w, r, "SpatialNearby",
		func(ctx context.Context, filter database.LocationStatsFilter, params interface{}) (interface{}, error) {
			coords := params.(*CoordinateParams)
			locations, err := h.db.GetNearbyLocations(ctx, coords.Lat, coords.Lon, coords.Radius, filter)
			if err != nil {
				return nil, err
			}
			if locations == nil {
				return []models.LocationStats{}, nil
			}
			return locations, nil
		},
		coords,
		coords,
	)
}

// exportConfig holds configuration for file export operations
type exportConfig struct {
	fileExtension string
	contentType   string
	exportFunc    func(ctx context.Context, outputPath string, filter database.LocationStatsFilter) error
}

// handleFileExport is a common handler for file-based exports (GeoParquet, GeoJSON)
func (h *Handler) handleFileExport(w http.ResponseWriter, r *http.Request, config exportConfig) {
	filter, errMsg := parseExportFilter(r)
	if errMsg != "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", errMsg, nil)
		return
	}

	if !h.requireDB(w) {
		return
	}

	start := time.Now()
	filename := fmt.Sprintf("cartographus-locations-%s.%s", time.Now().Format("2006-01-02-150405"), config.fileExtension)
	outputPath := fmt.Sprintf("/tmp/%s", filename)

	// Ensure cleanup of temporary file even on panic
	defer func() {
		_ = os.Remove(outputPath)
	}()

	if err := config.exportFunc(r.Context(), outputPath, filter); err != nil {
		respondError(w, http.StatusInternalServerError, "EXPORT_ERROR",
			fmt.Sprintf("Failed to export %s", config.fileExtension), err)
		return
	}

	// Set response headers for file download
	w.Header().Set("Content-Type", config.contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("X-Export-Time-MS", fmt.Sprintf("%d", time.Since(start).Milliseconds()))

	http.ServeFile(w, r, outputPath)
}

// ExportGeoParquet exports location data to GeoParquet format
// GET /api/v1/export/geoparquet?start_date=...&end_date=...&users=...&media_types=...
// This addresses Medium Priority Issue M2 from the production audit
func (h *Handler) ExportGeoParquet(w http.ResponseWriter, r *http.Request) {
	// Validate filter params first
	filter, errMsg := parseExportFilter(r)
	if errMsg != "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", errMsg, nil)
		return
	}

	// Check DB availability before checking extensions
	if !h.requireDB(w) {
		return
	}

	// Check if spatial extension is available (required for GeoParquet export)
	if !h.db.IsSpatialAvailable() {
		respondError(w, http.StatusServiceUnavailable, "EXTENSION_UNAVAILABLE",
			"Spatial extension not available. GeoParquet export requires the DuckDB spatial extension.", nil)
		return
	}

	// Reuse common export logic (skip validation since already done)
	start := time.Now()
	filename := fmt.Sprintf("cartographus-locations-%s.parquet", time.Now().Format("2006-01-02-150405"))
	outputPath := fmt.Sprintf("/tmp/%s", filename)

	defer func() {
		_ = os.Remove(outputPath)
	}()

	if err := h.db.ExportGeoParquet(r.Context(), outputPath, filter); err != nil {
		respondError(w, http.StatusInternalServerError, "EXPORT_ERROR", "Failed to export parquet", err)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("X-Export-Time-MS", fmt.Sprintf("%d", time.Since(start).Milliseconds()))

	http.ServeFile(w, r, outputPath)
}

// ExportGeoJSON exports location data to GeoJSON format
// GET /api/v1/export/geojson?start_date=...&end_date=...&users=...&media_types=...
func (h *Handler) ExportGeoJSON(w http.ResponseWriter, r *http.Request) {
	h.handleFileExport(w, r, exportConfig{
		fileExtension: "json",
		contentType:   "application/geo+json",
		exportFunc:    h.db.ExportGeoJSON,
	})
}

// streamingFeature is a lightweight feature for streaming (subset of full GeoJSON)
type streamingFeature struct {
	Type       string                 `json:"type"`
	Geometry   map[string]interface{} `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}

// buildStreamingFeature creates a lightweight GeoJSON feature for streaming
func buildStreamingFeature(location *models.LocationStats) streamingFeature {
	return streamingFeature{
		Type: "Feature",
		Geometry: map[string]interface{}{
			"type":        "Point",
			"coordinates": []float64{location.Longitude, location.Latitude},
		},
		Properties: map[string]interface{}{
			"country":        location.Country,
			"city":           location.City,
			"playback_count": location.PlaybackCount,
		},
	}
}

// setupStreamingResponse configures headers and validates streaming support
func setupStreamingResponse(w http.ResponseWriter) (http.Flusher, error) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}
	return flusher, nil
}

// streamGeoJSONFeatures streams features as GeoJSON with batched flushing
func streamGeoJSONFeatures(w http.ResponseWriter, flusher http.Flusher, locations []models.LocationStats) {
	const batchSize = 100

	_, _ = fmt.Fprintf(w, `{"type":"FeatureCollection","features":[`)
	flusher.Flush()

	for i := range locations {
		if i > 0 {
			_, _ = fmt.Fprintf(w, ",")
		}

		feature := buildStreamingFeature(&locations[i])
		featureJSON, err := json.Marshal(feature)
		if err != nil {
			continue // Skip malformed features
		}
		_, _ = w.Write(featureJSON)

		if (i+1)%batchSize == 0 {
			flusher.Flush()
		}
	}

	_, _ = fmt.Fprintf(w, `]}`)
	flusher.Flush()
}

// StreamLocationsGeoJSON streams location data as GeoJSON with chunked transfer encoding
// This addresses Medium Priority Issue M12 from the production audit
// GET /api/v1/stream/locations-geojson?start_date=...&end_date=...
// Handles 100k+ locations without memory spikes by streaming JSON incrementally
func (h *Handler) StreamLocationsGeoJSON(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}

	filter := database.LocationStatsFilter{}
	if err := parseDateFilter(r, &filter); err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	flusher, err := setupStreamingResponse(w)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "SERVER_ERROR", err.Error(), nil)
		return
	}

	locations, err := h.db.GetLocationStatsFiltered(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch locations", err)
		return
	}

	streamGeoJSONFeatures(w, flusher, locations)
}

// setVectorTileHeaders sets appropriate headers for MVT response
func setVectorTileHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Header().Set("Content-Encoding", "gzip")              // MVT is typically gzipped
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache tiles for 1 hour
	w.Header().Set("Access-Control-Allow-Origin", "*")      // Allow cross-origin tile requests
}

// parseTileRequest validates tile coordinates and filter parameters
func parseTileRequest(r *http.Request) (*TileCoordinates, database.LocationStatsFilter, error) {
	coords, errCode, err := ParseTileCoordinates(r.URL.Path)
	if err != nil {
		return nil, database.LocationStatsFilter{}, fmt.Errorf("%s: %s", errCode, err.Error())
	}

	filter := database.LocationStatsFilter{}
	if err := parseDateFilter(r, &filter); err != nil {
		return nil, database.LocationStatsFilter{}, err
	}

	return coords, filter, nil
}

// GetVectorTile serves Mapbox Vector Tiles (MVT) for map visualization
// This addresses Medium Priority Issue M10 from the production audit
// GET /api/v1/tiles/{z}/{x}/{y}.pbf
// Handles 1M+ locations smoothly through tile-based data delivery
func (h *Handler) GetVectorTile(w http.ResponseWriter, r *http.Request) {
	coords, filter, err := parseTileRequest(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	if !h.requireDB(w) {
		return
	}

	mvtData, err := h.db.GenerateVectorTile(r.Context(), coords.Z, coords.X, coords.Y, filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "TILE_GENERATION_ERROR", "Failed to generate tile", err)
		return
	}

	setVectorTileHeaders(w)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(mvtData)
}

// Data Export Management Endpoints (Priority 3 - 3 endpoints)

// TautulliExportsTable handles requests for paginated export history table
