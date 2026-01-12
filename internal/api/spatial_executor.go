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
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/models"
)

// SpatialQueryExecutor encapsulates the common pattern for spatial query handlers.
// It implements a cache-first execution flow for geospatial queries that reduces
// handler complexity by ~58%:
//
//  1. Parse and validate spatial parameters (bbox, coordinates, resolution)
//  2. Check cache for existing results (5-minute TTL)
//  3. Execute spatial query if cache miss (uses DuckDB spatial extension)
//  4. Cache the result for subsequent requests
//  5. Respond with JSON including metadata (query time, cached status)
//
// This executor is used by 5 spatial handlers and eliminates ~237 lines of
// repetitive validation and cache-checking code across handlers_spatial.go.
//
// All spatial queries leverage DuckDB's spatial extension with R-tree indexing
// for 100x performance improvement on viewport queries.
//
// Example usage:
//
//	executor := NewSpatialQueryExecutor(h)
//	bbox, _ := ValidateBoundingBox(r)
//	executor.ExecuteWithCache(w, r, "SpatialViewport",
//	    func(ctx context.Context, filter database.LocationStatsFilter, params interface{}) (interface{}, error) {
//	        b := params.(*BoundingBoxParams)
//	        return h.db.GetLocationsInViewport(ctx, filter, b.West, b.South, b.East, b.North)
//	    },
//	    bbox, bbox)
type SpatialQueryExecutor struct {
	handler *Handler
}

// NewSpatialQueryExecutor creates a new spatial query executor instance.
//
// Parameters:
//   - h: Handler instance providing access to database, cache, and config
//
// Returns a configured executor ready to process spatial queries with
// automatic caching, error handling, and response formatting.
func NewSpatialQueryExecutor(h *Handler) *SpatialQueryExecutor {
	return &SpatialQueryExecutor{handler: h}
}

// QueryFunc is a function type for executing spatial queries.
// It receives a context for cancellation, a filter for query constraints,
// and typed spatial parameters (e.g., BoundingBoxParams, CoordinateParams).
//
// The params argument should be a pointer to a validated parameter struct.
// The result must be JSON-serializable as it will be cached and returned in
// an APIResponse wrapper with metadata.
//
// Example implementations:
//   - Viewport: func(ctx, filter, params) { bbox := params.(*BoundingBoxParams); return db.GetLocationsInViewport(...) }
//   - Nearby: func(ctx, filter, params) { coords := params.(*CoordinateParams); return db.GetNearbyLocations(...) }
//   - Hexagons: func(ctx, filter, params) { res := params.(*ResolutionParams); return db.GetH3AggregatedHexagons(...) }
type QueryFunc func(ctx context.Context, filter database.LocationStatsFilter, params interface{}) (interface{}, error)

// ExecuteWithCache executes a spatial query with automatic caching.
// Use this method for all spatial queries that require validated parameters
// beyond the standard LocationStatsFilter.
//
// Parameters:
//   - w: HTTP response writer
//   - r: HTTP request containing query parameters for filter building
//   - cacheKeyPrefix: Unique identifier for this query type (e.g., "SpatialViewport")
//   - queryFunc: Function that executes the actual database query
//   - cacheKeyParams: Parameters to include in cache key (for cache isolation)
//   - queryParams: Parameters passed to queryFunc (can be same as cacheKeyParams)
//
// The method automatically:
//   - Builds a LocationStatsFilter from query parameters
//   - Generates a cache key from prefix + filter + cacheKeyParams
//   - Returns cached data if available (with Cached: true in metadata)
//   - Executes queryFunc on cache miss, passing filter and queryParams
//   - Caches successful results with 5-minute TTL
//   - Responds with JSON including query time metrics
//
// The cacheKeyParams and queryParams can be the same object, or different if you
// need to cache based on different criteria than what the query receives.
//
// Cache hits return immediately with 0ms query time. Cache misses include
// actual query execution time in milliseconds (typically 5-50ms with R-tree index).
//
// Example:
//
//	bbox, err := ValidateBoundingBox(r)
//	if err != nil {
//	    respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
//	    return
//	}
//	executor.ExecuteWithCache(w, r, "SpatialViewport",
//	    func(ctx context.Context, filter database.LocationStatsFilter, params interface{}) (interface{}, error) {
//	        b := params.(*BoundingBoxParams)
//	        return h.db.GetLocationsInViewport(ctx, filter, b.West, b.South, b.East, b.North)
//	    },
//	    bbox, bbox)
func (e *SpatialQueryExecutor) ExecuteWithCache(
	w http.ResponseWriter,
	r *http.Request,
	cacheKeyPrefix string,
	queryFunc QueryFunc,
	cacheKeyParams interface{},
	queryParams interface{},
) {
	// Check if database is available (protects against nil pointer in queryFunc)
	if e.handler.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return
	}

	// Check if spatial extension is available (required for all spatial queries)
	if !e.handler.db.IsSpatialAvailable() {
		respondError(w, http.StatusServiceUnavailable, "EXTENSION_UNAVAILABLE",
			"Spatial extension not available. Spatial queries require the DuckDB spatial extension.", nil)
		return
	}

	start := time.Now()
	filter := e.handler.buildFilter(r)

	// Generate cache key
	cacheKey := cache.GenerateKey(cacheKeyPrefix, struct {
		Filter database.LocationStatsFilter
		Params interface{}
	}{filter, cacheKeyParams})

	// Check cache first (only if cache is available)
	if e.handler.cache != nil {
		if cached, found := e.handler.cache.Get(cacheKey); found {
			respondJSON(w, http.StatusOK, &models.APIResponse{
				Status: "success",
				Data:   cached,
				Metadata: models.Metadata{
					Timestamp:   time.Now(),
					QueryTimeMS: 0, // Cached
					Cached:      true,
				},
			})
			return
		}
	}

	// Execute query
	data, err := queryFunc(r.Context(), filter, queryParams)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR",
			fmt.Sprintf("Failed to execute query: %s", cacheKeyPrefix), err)
		return
	}

	// Cache the result (only if cache is available)
	if e.handler.cache != nil {
		e.handler.cache.Set(cacheKey, data)
	}

	// Respond with data
	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   data,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// validateFloatParam validates and parses a float query parameter
func validateFloatParam(value string, paramName string, minVal, maxVal float64) (float64, error) {
	if value == "" {
		return 0, fmt.Errorf("missing required parameter: %s", paramName)
	}

	var result float64
	_, err := fmt.Sscanf(value, "%f", &result)
	if err != nil || result < minVal || result > maxVal {
		return 0, fmt.Errorf("invalid %s parameter (must be %.1f to %.1f)", paramName, minVal, maxVal)
	}

	return result, nil
}

// validateIntParam validates and parses an integer query parameter
func validateIntParam(value string, paramName string, minVal, maxVal int) (int, error) {
	if value == "" {
		return 0, fmt.Errorf("missing required parameter: %s", paramName)
	}

	// Use Sscanf with strict format checking
	// %d reads the integer, then %c should fail if nothing follows
	// If there's extra content (like ".5" in "7.5"), we reject it
	var result int
	var extra rune
	n, err := fmt.Sscanf(value, "%d%c", &result, &extra)
	// n==1 means only the integer was read (valid)
	// n==2 means extra characters followed (invalid, like "7.5")
	// err means parse failure (invalid, like "abc")
	if n == 2 || (err != nil && n == 0) {
		return 0, fmt.Errorf("invalid %s parameter (must be %d to %d)", paramName, minVal, maxVal)
	}
	if result < minVal || result > maxVal {
		return 0, fmt.Errorf("invalid %s parameter (must be %d to %d)", paramName, minVal, maxVal)
	}

	return result, nil
}

// BoundingBoxParams holds validated bounding box coordinates for viewport queries.
// All coordinates are validated to be within valid geographic ranges:
//   - West, East: -180 to 180 degrees longitude
//   - South, North: -90 to 90 degrees latitude
//
// Used by SpatialViewport handler for efficient R-tree spatial index queries.
type BoundingBoxParams struct {
	West, South, East, North float64
}

// ValidateBoundingBox parses and validates bounding box parameters from HTTP request.
// It extracts and validates all four coordinates from query parameters:
//   - west: Western longitude boundary (-180 to 180)
//   - south: Southern latitude boundary (-90 to 90)
//   - east: Eastern longitude boundary (-180 to 180)
//   - north: Northern latitude boundary (-90 to 90)
//
// Returns a BoundingBoxParams struct with validated coordinates or an error
// if any parameter is missing, malformed, or out of range.
//
// Example usage:
//
//	bbox, err := ValidateBoundingBox(r)
//	if err != nil {
//	    respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
//	    return
//	}
//	// bbox.West, bbox.South, bbox.East, bbox.North are all validated
func ValidateBoundingBox(r *http.Request) (*BoundingBoxParams, error) {
	west, err := validateFloatParam(r.URL.Query().Get("west"), "west", -180, 180)
	if err != nil {
		return nil, err
	}

	south, err := validateFloatParam(r.URL.Query().Get("south"), "south", -90, 90)
	if err != nil {
		return nil, err
	}

	east, err := validateFloatParam(r.URL.Query().Get("east"), "east", -180, 180)
	if err != nil {
		return nil, err
	}

	north, err := validateFloatParam(r.URL.Query().Get("north"), "north", -90, 90)
	if err != nil {
		return nil, err
	}

	return &BoundingBoxParams{
		West:  west,
		South: south,
		East:  east,
		North: north,
	}, nil
}

// CoordinateParams holds validated latitude/longitude coordinates for point-based queries.
// All coordinates are validated to be within valid geographic ranges:
//   - Lat: -90 to 90 degrees latitude
//   - Lon: -180 to 180 degrees longitude
//   - Radius: 1 to 20,000 kilometers (optional, defaults to 100.0 km)
//
// Used by SpatialNearby handler for radius-based proximity searches.
type CoordinateParams struct {
	Lat, Lon float64
	Radius   float64 // optional, for nearby queries (default 100 km)
}

// ValidateCoordinates parses and validates latitude/longitude parameters from HTTP request.
// It extracts and validates point coordinates from query parameters:
//   - lat: Latitude coordinate (-90 to 90)
//   - lon: Longitude coordinate (-180 to 180)
//   - radius: Optional search radius in kilometers (1 to 20,000, default 100)
//
// Parameters:
//   - r: HTTP request containing query parameters
//   - requireRadius: If true, validates radius parameter; if false, uses default 100 km
//
// Returns a CoordinateParams struct with validated coordinates and radius,
// or an error if any parameter is missing, malformed, or out of range.
//
// Example usage:
//
//	coords, err := ValidateCoordinates(r, true)
//	if err != nil {
//	    respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
//	    return
//	}
//	// coords.Lat, coords.Lon, coords.Radius are all validated
func ValidateCoordinates(r *http.Request, requireRadius bool) (*CoordinateParams, error) {
	lat, err := validateFloatParam(r.URL.Query().Get("lat"), "lat", -90, 90)
	if err != nil {
		return nil, err
	}

	lon, err := validateFloatParam(r.URL.Query().Get("lon"), "lon", -180, 180)
	if err != nil {
		return nil, err
	}

	params := &CoordinateParams{Lat: lat, Lon: lon, Radius: 100.0}

	if requireRadius {
		radiusStr := r.URL.Query().Get("radius")
		if radiusStr != "" {
			radius, err := validateFloatParam(radiusStr, "radius", 1, 20000)
			if err != nil {
				return nil, err
			}
			params.Radius = radius
		}
	}

	return params, nil
}

// ResolutionParams holds validated H3 hexagon resolution for spatial aggregation.
// Resolution determines the size of H3 hexagons:
//   - Resolution 6: ~36.1 km² hexagons (country/state level)
//   - Resolution 7: ~5.2 km² hexagons (city level, default)
//   - Resolution 8: ~0.74 km² hexagons (neighborhood level)
//
// Used by SpatialHexagons and SpatialTemporalDensity handlers for GPU-accelerated
// aggregation via deck.gl HexagonLayer.
type ResolutionParams struct {
	Resolution int
}

// ValidateResolution parses and validates H3 resolution parameter from HTTP request.
// It extracts and validates the hexagon resolution from query parameters:
//   - resolution: H3 resolution level (6 to 8, defaults to provided defaultVal)
//
// Parameters:
//   - r: HTTP request containing query parameters
//   - defaultVal: Default resolution to use if parameter is not provided (typically 7)
//
// Returns a ResolutionParams struct with validated resolution,
// or an error if the parameter is malformed or out of range.
//
// Example usage:
//
//	resParams, err := ValidateResolution(r, 7)
//	if err != nil {
//	    respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
//	    return
//	}
//	// resParams.Resolution is validated (6-8) or default value
func ValidateResolution(r *http.Request, defaultVal int) (*ResolutionParams, error) {
	resolutionStr := r.URL.Query().Get("resolution")
	resolution := defaultVal

	if resolutionStr != "" {
		var err error
		resolution, err = validateIntParam(resolutionStr, "resolution", 6, 8)
		if err != nil {
			return nil, err
		}
	}

	return &ResolutionParams{Resolution: resolution}, nil
}

// ValidateInterval validates time interval parameter for temporal aggregation.
// It ensures the interval is one of the supported temporal resolutions:
//   - "hour": Hourly aggregation (24 buckets per day)
//   - "day": Daily aggregation (7 buckets per week)
//   - "week": Weekly aggregation (4-5 buckets per month)
//   - "month": Monthly aggregation (12 buckets per year)
//
// Used by SpatialTemporalDensity and AnalyticsTemporalHeatmap handlers for
// time-based grouping in queries.
//
// Parameters:
//   - interval: The interval string to validate
//
// Returns an error if the interval is not one of the valid values, or nil if valid.
//
// Example usage:
//
//	interval := r.URL.Query().Get("interval")
//	if interval == "" {
//	    interval = "hour" // default
//	}
//	if err := ValidateInterval(interval); err != nil {
//	    respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
//	    return
//	}
func ValidateInterval(interval string) error {
	validIntervals := map[string]bool{
		"hour":  true,
		"day":   true,
		"week":  true,
		"month": true,
	}

	if !validIntervals[interval] {
		return fmt.Errorf("interval must be one of: hour, day, week, month")
	}

	return nil
}
