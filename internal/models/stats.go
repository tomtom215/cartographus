// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"time"
)

// Stats represents overall system statistics
type Stats struct {
	TotalPlaybacks    int            `json:"total_playbacks"`
	UniqueLocations   int            `json:"unique_locations"`
	UniqueUsers       int            `json:"unique_users"`
	TopCountries      []CountryStats `json:"top_countries"`
	RecentActivity    int            `json:"recent_activity"`
	LastSyncTime      *time.Time     `json:"last_sync_time,omitempty"`
	DatabaseSizeBytes int64          `json:"database_size_bytes"`
}

// CountryStats represents playback statistics for a country
type CountryStats struct {
	Country       string `json:"country"`
	PlaybackCount int    `json:"playback_count"`
	UniqueUsers   int    `json:"unique_users"`
}

// HealthStatus represents the health check response
type HealthStatus struct {
	Status            string     `json:"status"`
	Mode              string     `json:"mode"` // "standalone" (no Tautulli) or "tautulli" (with Tautulli)
	Version           string     `json:"version"`
	DatabaseConnected bool       `json:"database_connected"`
	TautulliConnected bool       `json:"tautulli_connected"`
	LastSyncTime      *time.Time `json:"last_sync_time,omitempty"`
	Uptime            float64    `json:"uptime_seconds"`
}

// SetupStatus represents the setup wizard status for first-time users.
// Used by the frontend onboarding wizard to guide users through configuration.
//
// Fields:
//   - Ready: Overall readiness - true if at least one data source is connected
//   - Database: Database connection status
//   - DataSources: Configuration status for each data source
//   - DataAvailable: Whether playback data exists in the database
//   - Recommendations: List of recommended next steps
//
// Example response (partially configured):
//
//	{
//	  "ready": true,
//	  "database": {"connected": true},
//	  "data_sources": {
//	    "tautulli": {"configured": true, "connected": true},
//	    "plex": {"configured": false}
//	  },
//	  "data_available": {"has_playbacks": true, "playback_count": 1234},
//	  "recommendations": ["Configure Plex direct for real-time updates"]
//	}
type SetupStatus struct {
	Ready           bool                  `json:"ready"`
	Database        SetupDatabaseStatus   `json:"database"`
	DataSources     SetupDataSources      `json:"data_sources"`
	DataAvailable   SetupDataAvailability `json:"data_available"`
	Recommendations []string              `json:"recommendations,omitempty"`
}

// SetupDatabaseStatus represents database connection status
type SetupDatabaseStatus struct {
	Connected bool `json:"connected"`
}

// SetupDataSources represents configuration status for all data sources
type SetupDataSources struct {
	Tautulli SetupDataSourceStatus  `json:"tautulli"`
	Plex     SetupMediaServerStatus `json:"plex"`
	Jellyfin SetupMediaServerStatus `json:"jellyfin"`
	Emby     SetupMediaServerStatus `json:"emby"`
	NATS     SetupOptionalFeature   `json:"nats"`
}

// SetupDataSourceStatus represents a single data source configuration status
type SetupDataSourceStatus struct {
	Configured bool   `json:"configured"`
	Connected  bool   `json:"connected,omitempty"`
	URL        string `json:"url,omitempty"`
	Error      string `json:"error,omitempty"`
}

// SetupMediaServerStatus represents media server configuration with server count
type SetupMediaServerStatus struct {
	Configured  bool   `json:"configured"`
	Connected   bool   `json:"connected,omitempty"`
	ServerCount int    `json:"server_count,omitempty"`
	Error       string `json:"error,omitempty"`
}

// SetupOptionalFeature represents optional feature configuration
type SetupOptionalFeature struct {
	Enabled   bool   `json:"enabled"`
	Connected bool   `json:"connected,omitempty"`
	Error     string `json:"error,omitempty"`
}

// SetupDataAvailability represents data availability in the database
type SetupDataAvailability struct {
	HasPlaybacks    bool  `json:"has_playbacks"`
	PlaybackCount   int64 `json:"playback_count"`
	HasGeolocations bool  `json:"has_geolocations"`
}

// PlaybackTrend represents playback count over time
type PlaybackTrend struct {
	Date          string `json:"date"`
	PlaybackCount int    `json:"playback_count"`
	UniqueUsers   int    `json:"unique_users"`
}

// UserActivity represents user playback statistics
type UserActivity struct {
	Username      string  `json:"username"`
	PlaybackCount int     `json:"playback_count"`
	TotalDuration int     `json:"total_duration_minutes"`
	AvgCompletion float64 `json:"avg_completion"`
	UniqueMedia   int     `json:"unique_media"`
}

// MediaTypeStats represents playback statistics by media type
type MediaTypeStats struct {
	MediaType     string `json:"media_type"`
	PlaybackCount int    `json:"playback_count"`
	UniqueUsers   int    `json:"unique_users"`
}

// CityStats represents playback statistics for a city
type CityStats struct {
	City          string `json:"city"`
	Country       string `json:"country"`
	PlaybackCount int    `json:"playback_count"`
	UniqueUsers   int    `json:"unique_users"`
}

// TrendsResponse represents the analytics trends endpoint response
type TrendsResponse struct {
	PlaybackTrends []PlaybackTrend `json:"playback_trends"`
	Interval       string          `json:"interval"`
}

// GeographicResponse represents the geographic analytics endpoint response
type GeographicResponse struct {
	TopCities              []CityStats            `json:"top_cities"`
	TopCountries           []CountryStats         `json:"top_countries"`
	MediaTypeDistribution  []MediaTypeStats       `json:"media_type_distribution"`
	ViewingHoursHeatmap    []ViewingHoursHeatmap  `json:"viewing_hours_heatmap"`
	PlatformDistribution   []PlatformStats        `json:"platform_distribution"`
	PlayerDistribution     []PlayerStats          `json:"player_distribution"`
	ContentCompletionStats ContentCompletionStats `json:"content_completion_stats"`
	TranscodeDistribution  []TranscodeStats       `json:"transcode_distribution"`
	ResolutionDistribution []ResolutionStats      `json:"resolution_distribution"`
	CodecDistribution      []CodecStats           `json:"codec_distribution"`
	LibraryDistribution    []LibraryStats         `json:"library_distribution"`
	RatingDistribution     []RatingStats          `json:"rating_distribution"`
	DurationStats          DurationStats          `json:"duration_stats"`
	YearDistribution       []YearStats            `json:"year_distribution"`
}

// UsersResponse represents the user analytics endpoint response
type UsersResponse struct {
	TopUsers []UserActivity `json:"top_users"`
}
