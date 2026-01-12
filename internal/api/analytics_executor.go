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

// AnalyticsQueryExecutor encapsulates the common pattern for analytics query handlers.
// It implements a cache-first execution flow that reduces handler complexity by ~60%:
//
//  1. Build filter from query parameters (date range, users, media types, etc.)
//  2. Check cache for existing results (5-minute TTL)
//  3. Execute query if cache miss
//  4. Cache the result for subsequent requests
//  5. Respond with JSON including metadata (query time, cached status)
//
// This executor is used by 11 analytics handlers and eliminates ~400 lines of
// repetitive cache-checking and response-building code across handlers_analytics.go.
//
// Example usage:
//
//	executor := NewAnalyticsQueryExecutor(h)
//	executor.ExecuteSimple(w, r, "AnalyticsTrends", func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
//	    return h.db.GetPlaybackTrends(ctx, filter)
//	})
type AnalyticsQueryExecutor struct {
	handler *Handler
}

// NewAnalyticsQueryExecutor creates a new analytics query executor instance.
//
// Parameters:
//   - h: Handler instance providing access to database, cache, and config
//
// Returns a configured executor ready to process analytics queries with
// automatic caching, error handling, and response formatting.
func NewAnalyticsQueryExecutor(h *Handler) *AnalyticsQueryExecutor {
	return &AnalyticsQueryExecutor{handler: h}
}

// AnalyticsQueryFunc is a function type for executing analytics queries.
// It receives a context for cancellation and a filter for query constraints,
// returning the query result (typically a struct or slice) or an error.
//
// The result must be JSON-serializable as it will be cached and returned in
// an APIResponse wrapper with metadata.
//
// Example implementations:
//   - Simple: func(ctx, filter) { return db.GetPlaybackTrends(ctx, filter) }
//   - With transformation: func(ctx, filter) { trends, _, err := db.GetPlaybackTrends(ctx, filter); return trends, err }
type AnalyticsQueryFunc func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error)

// ExecuteSimple executes a simple analytics query with automatic caching.
// Use this method for queries that only require the standard LocationStatsFilter
// (no additional parameters like limit, interval, or comparison type).
//
// Parameters:
//   - w: HTTP response writer
//   - r: HTTP request containing query parameters for filter building
//   - cacheKeyPrefix: Unique identifier for this query type (e.g., "AnalyticsTrends")
//   - queryFunc: Function that executes the actual database query
//
// The method automatically:
//   - Builds a LocationStatsFilter from query parameters
//   - Generates a cache key from prefix + filter
//   - Returns cached data if available (with Cached: true in metadata)
//   - Executes queryFunc on cache miss
//   - Caches successful results with 5-minute TTL
//   - Responds with JSON including query time metrics
//
// Cache hits return immediately with 0ms query time. Cache misses include
// actual query execution time in milliseconds.
//
// Example:
//
//	executor.ExecuteSimple(w, r, "AnalyticsBinge",
//	    func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
//	        return h.db.GetBingeAnalytics(ctx, filter)
//	    })
func (e *AnalyticsQueryExecutor) ExecuteSimple(
	w http.ResponseWriter,
	r *http.Request,
	cacheKeyPrefix string,
	queryFunc AnalyticsQueryFunc,
) {
	// Check if database is available (protects against nil pointer in queryFunc)
	if e.handler.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return
	}

	start := time.Now()
	filter := e.handler.buildFilter(r)

	// Generate cache key
	cacheKey := cache.GenerateKey(cacheKeyPrefix, filter)

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
	data, err := queryFunc(r.Context(), filter)
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

// ExecuteUserScoped executes an analytics query with automatic user-scoping for RBAC.
// For non-admin users, the query is automatically filtered to only show their own data.
// Admins see all data without restriction.
//
// SECURITY: This method enforces the RBAC policy where regular users can only
// see their own analytics data. This prevents information leakage between users.
//
// Parameters:
//   - w: HTTP response writer
//   - r: HTTP request containing query parameters and auth context
//   - cacheKeyPrefix: Unique identifier for this query type
//   - queryFunc: Function that executes the actual database query
//
// The method automatically:
//   - Extracts user identity from request context
//   - For non-admins: Injects user filter into LocationStatsFilter.Users
//   - For admins: No filter restriction (sees all data)
//   - Includes user ID in cache key for per-user caching
//
// Example:
//
//	executor.ExecuteUserScoped(w, r, "AnalyticsTrends",
//	    func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
//	        return h.db.GetPlaybackTrends(ctx, filter)
//	    })
func (e *AnalyticsQueryExecutor) ExecuteUserScoped(
	w http.ResponseWriter,
	r *http.Request,
	cacheKeyPrefix string,
	queryFunc AnalyticsQueryFunc,
) {
	// Check if database is available
	if e.handler.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return
	}

	// Get handler context for RBAC
	hctx := GetHandlerContext(r)

	start := time.Now()
	filter := e.handler.buildFilter(r)

	// RBAC: For non-admins, scope to their own data only
	userScope := ""
	if hctx != nil && hctx.IsAuthenticated() && !hctx.IsAdmin {
		// Non-admin user: restrict to their own data
		userScope = hctx.Username
		if userScope != "" {
			// Override any user filter with just the current user
			filter.Users = []string{userScope}
		}
	}

	// Generate cache key (includes user scope for per-user caching)
	cacheKey := cache.GenerateKey(cacheKeyPrefix, struct {
		Filter    database.LocationStatsFilter
		UserScope string
	}{filter, userScope})

	// Check cache first
	if e.handler.cache != nil {
		if cached, found := e.handler.cache.Get(cacheKey); found {
			respondJSON(w, http.StatusOK, &models.APIResponse{
				Status: "success",
				Data:   cached,
				Metadata: models.Metadata{
					Timestamp:   time.Now(),
					QueryTimeMS: 0,
					Cached:      true,
				},
			})
			return
		}
	}

	// Execute query
	data, err := queryFunc(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR",
			fmt.Sprintf("Failed to execute query: %s", cacheKeyPrefix), err)
		return
	}

	// Cache the result
	if e.handler.cache != nil {
		e.handler.cache.Set(cacheKey, data)
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   data,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// ExecuteAdminOnly executes an analytics query that is restricted to admin users only.
// Non-admin users receive a 403 Forbidden response.
//
// SECURITY: This method enforces admin-only access for sensitive analytics that
// expose cross-user data or global insights that should not be visible to regular users.
//
// Use cases:
//   - User overlap/chord diagrams (exposes all users' viewing patterns)
//   - Bump charts (global ranking insights)
//   - Any analytics that aggregate across multiple users
//
// Parameters:
//   - w: HTTP response writer
//   - r: HTTP request containing query parameters and auth context
//   - cacheKeyPrefix: Unique identifier for this query type
//   - queryFunc: Function that executes the actual database query
//
// Example:
//
//	executor.ExecuteAdminOnly(w, r, "AnalyticsUserOverlap",
//	    func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
//	        return h.db.GetUserContentOverlapAnalytics(ctx, filter)
//	    })
func (e *AnalyticsQueryExecutor) ExecuteAdminOnly(
	w http.ResponseWriter,
	r *http.Request,
	cacheKeyPrefix string,
	queryFunc AnalyticsQueryFunc,
) {
	// RBAC: Require admin role (check BEFORE database availability)
	hctx := GetHandlerContext(r)
	if hctx == nil || !hctx.IsAuthenticated() {
		respondError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
		return
	}
	if !hctx.IsAdmin {
		respondError(w, http.StatusForbidden, "ADMIN_REQUIRED",
			"Admin role required to access this analytics endpoint", nil)
		return
	}

	// Check if database is available (after RBAC passes)
	if e.handler.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return
	}

	// Admin user: proceed with full data access
	start := time.Now()
	filter := e.handler.buildFilter(r)

	cacheKey := cache.GenerateKey(cacheKeyPrefix, filter)

	if e.handler.cache != nil {
		if cached, found := e.handler.cache.Get(cacheKey); found {
			respondJSON(w, http.StatusOK, &models.APIResponse{
				Status: "success",
				Data:   cached,
				Metadata: models.Metadata{
					Timestamp:   time.Now(),
					QueryTimeMS: 0,
					Cached:      true,
				},
			})
			return
		}
	}

	data, err := queryFunc(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR",
			fmt.Sprintf("Failed to execute query: %s", cacheKeyPrefix), err)
		return
	}

	if e.handler.cache != nil {
		e.handler.cache.Set(cacheKey, data)
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   data,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}

// ExecuteWithParam executes an analytics query with a single additional parameter.
// Use this method for queries that require both a LocationStatsFilter AND an
// additional parameter (e.g., limit, interval, comparison_type).
//
// Parameters:
//   - w: HTTP response writer
//   - r: HTTP request containing query parameters for filter building
//   - cacheKeyPrefix: Unique identifier for this query type (e.g., "AnalyticsUsers")
//   - queryFunc: Function that executes the database query with both filter and param
//   - param: Additional parameter passed to queryFunc (e.g., limit int, interval string)
//
// The method automatically:
//   - Builds a LocationStatsFilter from query parameters
//   - Generates a cache key from prefix + filter + param
//   - Returns cached data if available (with Cached: true in metadata)
//   - Executes queryFunc on cache miss, passing both filter and param
//   - Caches successful results with 5-minute TTL
//   - Responds with JSON including query time metrics
//
// The param is included in the cache key to ensure different parameter values
// are cached separately (e.g., limit=10 vs limit=50).
//
// Example usage:
//
//	// Validate limit parameter
//	limit := getIntParam(r, "limit", 10)
//	if limit < 1 || limit > 100 {
//	    respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid limit", nil)
//	    return
//	}
//
//	// Execute with parameter
//	executor.ExecuteWithParam(w, r, "AnalyticsUsers",
//	    func(ctx context.Context, filter database.LocationStatsFilter, param interface{}) (interface{}, error) {
//	        lmt := param.(int)
//	        return h.db.GetTopUsers(ctx, filter, lmt)
//	    },
//	    limit)
func (e *AnalyticsQueryExecutor) ExecuteWithParam(
	w http.ResponseWriter,
	r *http.Request,
	cacheKeyPrefix string,
	queryFunc func(ctx context.Context, filter database.LocationStatsFilter, param interface{}) (interface{}, error),
	param interface{},
) {
	// Check if database is available (protects against nil pointer in queryFunc)
	if e.handler.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return
	}

	start := time.Now()
	filter := e.handler.buildFilter(r)

	// Generate cache key
	cacheKey := cache.GenerateKey(cacheKeyPrefix, struct {
		Filter database.LocationStatsFilter
		Param  interface{}
	}{filter, param})

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
	data, err := queryFunc(r.Context(), filter, param)
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

// ExecuteWithParamUserScoped executes an analytics query with user-scoping and an additional parameter.
// Combines the functionality of ExecuteUserScoped and ExecuteWithParam.
//
// SECURITY: For non-admin users, the query is automatically filtered to only show their own data.
//
// Parameters:
//   - w: HTTP response writer
//   - r: HTTP request containing query parameters and auth context
//   - cacheKeyPrefix: Unique identifier for this query type
//   - queryFunc: Function that executes the database query with both filter and param
//   - param: Additional parameter passed to queryFunc
//
// Example:
//
//	executor.ExecuteWithParamUserScoped(w, r, "AnalyticsUsers",
//	    func(ctx context.Context, filter database.LocationStatsFilter, param interface{}) (interface{}, error) {
//	        lmt := param.(int)
//	        return h.db.GetTopUsers(ctx, filter, lmt)
//	    },
//	    limit)
func (e *AnalyticsQueryExecutor) ExecuteWithParamUserScoped(
	w http.ResponseWriter,
	r *http.Request,
	cacheKeyPrefix string,
	queryFunc func(ctx context.Context, filter database.LocationStatsFilter, param interface{}) (interface{}, error),
	param interface{},
) {
	if e.handler.db == nil {
		respondError(w, http.StatusServiceUnavailable, "SERVICE_ERROR", "Database not available", nil)
		return
	}

	hctx := GetHandlerContext(r)

	start := time.Now()
	filter := e.handler.buildFilter(r)

	// RBAC: For non-admins, scope to their own data only
	userScope := ""
	if hctx != nil && hctx.IsAuthenticated() && !hctx.IsAdmin {
		userScope = hctx.Username
		if userScope != "" {
			filter.Users = []string{userScope}
		}
	}

	cacheKey := cache.GenerateKey(cacheKeyPrefix, struct {
		Filter    database.LocationStatsFilter
		Param     interface{}
		UserScope string
	}{filter, param, userScope})

	if e.handler.cache != nil {
		if cached, found := e.handler.cache.Get(cacheKey); found {
			respondJSON(w, http.StatusOK, &models.APIResponse{
				Status: "success",
				Data:   cached,
				Metadata: models.Metadata{
					Timestamp:   time.Now(),
					QueryTimeMS: 0,
					Cached:      true,
				},
			})
			return
		}
	}

	data, err := queryFunc(r.Context(), filter, param)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR",
			fmt.Sprintf("Failed to execute query: %s", cacheKeyPrefix), err)
		return
	}

	if e.handler.cache != nil {
		e.handler.cache.Set(cacheKey, data)
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   data,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: time.Since(start).Milliseconds(),
		},
	})
}
