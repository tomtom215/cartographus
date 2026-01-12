// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
Package models defines data structures for the Cartographus application.

This package contains all data models used throughout the application, including
database schemas, API request/response structures, Tautulli API response models,
and internal data transfer objects. It serves as the single source of truth for
data structure definitions.

Key Components:

  - PlaybackEvent: Core database model for playback history (68 fields, v1.8 enrichment)
  - Geolocation: Geographic location data from Tautulli GeoIP lookups
  - APIResponse: Standardized API response wrapper
  - LocationStatsFilter: Comprehensive filter for analytics queries (14+ dimensions)
  - Tautulli Models: 54 struct types for Tautulli API responses

Model Categories:

1. Database Models:
  - PlaybackEvent: Playback history with metadata enrichment
  - Geolocation: IP address geolocation data
  - Stats: Global statistics aggregations

2. API Request/Response Models:
  - APIResponse: Standard response wrapper
  - APIError: Error details
  - Metadata: Response metadata (timestamp, query time)

3. Filter Models:
  - LocationStatsFilter: 14+ filter dimensions (users, media types, platforms, etc.)
  - Supports date ranges, multi-select, and spatial filtering

4. Tautulli API Models:
  - TautulliHistoryRecord: Playback history from Tautulli
  - TautulliActivity: Current streaming activity
  - 52 additional models for Tautulli API endpoints

5. Analytics Models:
  - BingeAnalytics: Binge-watching detection results
  - BandwidthAnalytics: Network usage patterns
  - UserEngagement: User activity metrics
  - 17 additional analytics result types

Usage Example - Database Models:

	import "github.com/tomtom215/cartographus/internal/models"

	// Create playback event
	event := &models.PlaybackEvent{
	    SessionKey:   "abc123",
	    Username:     "alice",
	    IPAddress:    "203.0.113.42",
	    Title:        "Inception",
	    WatchedAt:    time.Now(),
	    // ... 63 more fields
	}

	// Insert into database
	db.InsertPlaybackEvent(event)

Usage Example - API Response:

	import "github.com/tomtom215/cartographus/internal/models"

	// Success response
	response := models.APIResponse{
	    Status: "success",
	    Data: map[string]interface{}{
	        "total": 1000,
	        "results": events,
	    },
	    Metadata: &models.Metadata{
	        Timestamp:   time.Now(),
	        QueryTimeMs: 45,
	    },
	}

	json.NewEncoder(w).Encode(response)

	// Error response
	errorResponse := models.APIResponse{
	    Status: "error",
	    Error: &models.APIError{
	        Code:    "VALIDATION_ERROR",
	        Message: "Invalid date range",
	        Details: map[string]interface{}{
	            "field": "start_date",
	        },
	    },
	}

Usage Example - Filters:

	import "github.com/tomtom215/cartographus/internal/models"

	// Create filter
	filter := models.LocationStatsFilter{
	    StartDate:  time.Now().AddDate(0, 0, -30),
	    EndDate:    time.Now(),
	    Users:      []string{"alice", "bob"},
	    MediaTypes: []string{"movie", "episode"},
	    MinPlays:   5,
	    SortBy:     "play_count",
	    SortOrder:  "desc",
	    Limit:      100,
	}

	// Apply filter in query
	results := db.GetLocationStatsFiltered(filter)

PlaybackEvent Enrichment (v1.8):

The PlaybackEvent model was extended with 15 metadata fields for advanced analytics:

  - Content IDs: rating_key, parent_rating_key, grandparent_rating_key
  - Episode tracking: media_index, parent_media_index
  - External IDs: guid (IMDB, TVDB, TMDB)
  - Metadata: original_title, full_title, originally_available_at
  - Watch status: watched_status (0-100%)
  - Media: thumb (poster URL)
  - People: directors, writers, actors (comma-separated)
  - Classification: genres (comma-separated)

These fields enable:
  - Binge-watching detection (sequential episodes)
  - Genre and cast analytics
  - Content completion tracking
  - External system integration

Filter Dimensions (LocationStatsFilter):

The comprehensive filter supports 14+ dimensions:

  - Temporal: start_date, end_date, days (last N days)
  - Users: users (multi-select), exclude_users
  - Content: media_types, titles, libraries
  - Technical: platforms, players, quality_profiles
  - Geographic: countries, cities, ip_addresses
  - Aggregation: min_plays, min_duration
  - Sorting: sort_by, sort_order
  - Pagination: limit, offset

Thread Safety:

All models are:
  - Immutable after creation (pass-by-value or pointers)
  - Safe for concurrent read access
  - No internal mutexes needed (data structures only)

JSON Marshaling:

All models support JSON serialization:
  - Struct tags for field naming (camelCase for API, snake_case for DB)
  - Omitempty tags for optional fields
  - Time.Time uses RFC3339 format
  - Custom marshalers for complex types

See Also:

  - internal/database: Database operations using these models
  - internal/api: API handlers returning these models
  - internal/sync: Tautulli API client using Tautulli models
*/
package models
