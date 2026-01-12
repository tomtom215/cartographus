// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package database provides comprehensive data access and analytics functionality
// for the Cartographus application.
//
// # Overview
//
// This package serves as the data layer between the application and DuckDB,
// providing type-safe query execution, transaction management, and advanced
// analytics capabilities for Plex media server playback data.
//
// # Architecture
//
// The package is organized into several domain-specific files:
//
// Core Database Operations:
//   - database.go: Core database lifecycle (connection, initialization, cleanup)
//   - database_extensions.go: DuckDB extension installation (spatial, h3, inet, icu, json)
//   - database_schema.go: Table creation, migrations, and index management
//   - database_connection.go: Connection recovery with exponential backoff and pool configuration
//   - database_cache.go: Prepared statement caching and vector tile caching with TTL
//   - database_utils.go: Profiling, context management, and backup interface
//   - crud.go: Basic CRUD operations for playback events and geolocations
//   - filter.go: Filter building and WHERE clause construction
//
// Analytics Operations:
//   - analytics_binge.go: Binge-watching pattern detection and analysis
//   - analytics_bandwidth.go: Network bandwidth usage analytics
//   - analytics_engagement.go: User engagement, popular content, and watch parties
//   - analytics_comparative.go: Comparative analytics and content abandonment
//   - analytics_distribution.go: Distribution statistics (codecs, platforms, etc.)
//   - analytics_temporal.go: Time-based heatmap and temporal patterns
//   - analytics_trends.go: Trend analysis over time
//   - analytics_spatial.go: Geographic and spatial analytics
//   - analytics_helpers.go: Shared analytics helper functions
//   - database_new_analytics.go: Specialized analytics (HDR, audio, pauses, etc.)
//   - vector_tiles.go: Mapbox Vector Tile (MVT) generation for map visualization
//
// # Database Technology
//
// The package uses DuckDB (not SQLite) as its analytics database:
//   - OLAP-optimized for analytical queries
//   - Native spatial extension for geographic queries
//   - Advanced SQL features (window functions, CTEs, PERCENTILE_CONT)
//   - CGO-based driver (github.com/duckdb/duckdb-go/v2)
//
// # Key Features
//
// Analytics:
//   - Binge-watching detection (3+ episodes within 6 hours)
//   - Bandwidth usage calculation and trending
//   - User engagement scoring and ranking
//   - Content popularity and abandonment analysis
//   - Watch party detection (concurrent streams)
//   - Comparative period-over-period metrics
//   - Geographic playback distribution
//
// Performance:
//   - Connection recovery with exponential backoff
//   - Context-based query cancellation
//   - Prepared statement reuse via connection pool
//   - Spatial indexing for geographic queries
//   - Efficient aggregation with CTEs and window functions
//
// # Usage Examples
//
// Basic CRUD:
//
//	db, err := database.New(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
//
//	// Insert playback event
//	event := &models.PlaybackEvent{...}
//	if err := db.InsertPlaybackEvent(ctx, event); err != nil {
//	    log.Printf("Insert failed: %v", err)
//	}
//
// Analytics:
//
//	// Get binge-watching analytics
//	filter := database.LocationStatsFilter{
//	    StartDate: &startDate,
//	    EndDate:   &endDate,
//	}
//	bingeStats, err := db.GetBingeAnalytics(ctx, filter)
//	if err != nil {
//	    log.Printf("Analytics failed: %v", err)
//	}
//
// Filtering:
//
//	// Build complex WHERE clause
//	filter := database.LocationStatsFilter{
//	    StartDate:  &startDate,
//	    EndDate:    &endDate,
//	    Users:      []string{"alice", "bob"},
//	    MediaTypes: []string{"movie", "episode"},
//	}
//	// Filter automatically builds parameterized queries
//
// # Concurrency
//
// All exported methods are safe for concurrent use. The package handles:
//   - Connection pooling via DuckDB driver
//   - Automatic reconnection on connection failures
//   - Context-based cancellation for long-running queries
//   - Thread-safe query execution
//
// # Error Handling
//
// The package follows Go error handling best practices:
//   - Errors are wrapped with context using fmt.Errorf with %w
//   - Connection errors trigger automatic reconnection attempts
//   - Query timeouts are enforced via context deadlines
//   - All database errors are propagated to callers
//
// # Testing
//
// The package includes comprehensive test coverage:
//   - Unit tests for all CRUD operations
//   - Integration tests for analytics functions
//   - Benchmark tests for performance validation
//   - Fuzz tests for filter SQL injection prevention
//   - Concurrent access tests with race detector
//
// Test coverage: 90.2% (as of 2025-11-23)
//
// # Performance Targets
//
//   - CRUD operations: <10ms (p95)
//   - Analytics queries: <100ms (p95)
//   - Spatial queries: <30ms (p95)
//   - Memory usage: <512MB under normal load
//   - Connection recovery: <5s for automatic reconnection
//
// # Package Dependencies
//
// Internal dependencies:
//   - internal/models: Data model definitions
//   - internal/bandwidth: Bandwidth calculation utilities
//   - internal/database/query: SQL query building helpers
//
// External dependencies:
//   - github.com/duckdb/duckdb-go/v2: DuckDB driver (CGO-based)
//
// # Maintainer Notes
//
// When adding new analytics:
//  1. Add method to appropriate analytics_*.go file based on domain
//  2. Use LocationStatsFilter for consistent parameter handling
//  3. Add comprehensive godoc comments with complexity notes
//  4. Include benchmark tests in database_bench_test.go
//  5. Test with race detector: go test -race ./internal/database
//  6. Verify query performance with EXPLAIN ANALYZE
//
// When modifying queries:
//  1. Use parameterized queries (? placeholders) to prevent SQL injection
//  2. Prefer CTEs for complex multi-step queries
//  3. Add indexes for frequently filtered columns
//  4. Test with realistic data volumes (10k+ events)
//  5. Profile with pprof if performance degrades
//
// For spatial queries:
//  1. Use ST_MakePoint(lon, lat) for point creation
//  2. Use ST_Distance for distance calculations
//  3. Leverage spatial indexes (RTREE) for performance
//  4. Remember: DuckDB uses lon, lat order (not lat, lon)
//
// # See Also
//
//   - ARCHITECTURE.md: System architecture and design decisions
//   - docs/GO_METRICS_ANALYSIS.md: Code quality metrics and refactoring roadmap
//   - TESTING.md: Testing strategy and E2E test coverage
package database
