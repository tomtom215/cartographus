// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
Package api provides the HTTP REST API layer for Cartographus.

This package implements 100+ HTTP endpoints for managing playback data, analytics,
geospatial visualizations, and Tautulli proxy functionality. It serves as the primary
interface between the frontend web application and the backend services.

Key Components:

  - Router: HTTP route configuration and middleware stack integration
  - Handler: Request handlers for all API endpoints (~100 routes)
  - Response formatting: Standardized JSON responses with metadata
  - Error handling: Consistent error responses with appropriate HTTP status codes
  - Authentication integration: JWT and Basic Auth support via middleware
  - Rate limiting: Token bucket rate limiter to prevent abuse
  - CORS: Cross-Origin Resource Sharing for frontend compatibility

API Categories:

The API is organized into the following categories:

1. Core Endpoints (/api/v1/):
  - Health checks (health/live, health/ready)
  - Authentication (auth/login)
  - Statistics and aggregations (stats, playbacks, locations, users)

2. Analytics Endpoints (/api/v1/analytics/):
  - Playback trends and temporal analytics
  - User engagement and watch party detection
  - Bandwidth and quality analytics
  - Binge-watching patterns
  - Content abandonment analysis

3. Spatial Endpoints (/api/v1/spatial/):
  - Vector tile generation (MVT format) for map rendering
  - GeoJSON export and streaming
  - Spatial queries (viewport, nearby, hexagons, arcs)
  - GeoParquet export for data science workflows

4. Tautulli Proxy Endpoints (/api/v1/tautulli/):
  - 54 proxy endpoints to Tautulli API
  - Library management, user stats, server info
  - Play history and activity monitoring

5. WebSocket Endpoint (/ws):
  - Real-time playback notifications
  - Sync completion broadcasts
  - Live activity streaming

Usage Example:

	import (
	    "github.com/tomtom215/cartographus/internal/api"
	    "github.com/tomtom215/cartographus/internal/auth"
	    "github.com/tomtom215/cartographus/internal/database"
	)

	// Create dependencies
	db, _ := database.New(config)
	middleware := auth.NewMiddleware(jwtManager, basicAuthManager, config)

	// Create handler and router
	handler := api.NewHandler(db, cache, syncManager, wsHub)
	router := api.NewRouter(handler, middleware)

	// Setup routes and start server (Chi router - ADR-0016)
	http.ListenAndServe(":3857", router.SetupChi())

Performance Characteristics:

  - Response times: p95 <100ms for most endpoints (target)
  - Caching: 5-minute TTL for analytics endpoints
  - Streaming: Supports chunked transfer encoding for large exports
  - Compression: Gzip middleware for responses >1KB

Thread Safety:

All handlers are thread-safe and designed for concurrent request handling.
Shared resources (database, cache, WebSocket hub) are protected by their
respective synchronization primitives.

Security:

  - CSP headers with nonce-based script whitelisting
  - JWT token validation on protected routes
  - Rate limiting (100 req/min per IP)
  - Input validation and sanitization
  - SQL injection prevention via parameterized queries

See Also:

  - internal/auth: Authentication and authorization
  - internal/database: Data access layer
  - internal/models: Request/response data structures
  - internal/middleware: HTTP middleware components
*/
package api
