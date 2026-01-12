// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"time"
)

// H3HexagonStats represents aggregated playback data for an H3 hexagon
// Uses DuckDB's native H3 spatial indexing for fast hierarchical aggregation
type H3HexagonStats struct {
	H3Index           uint64  `json:"h3_index"`            // H3 hexagon index
	Latitude          float64 `json:"latitude"`            // Hexagon center latitude
	Longitude         float64 `json:"longitude"`           // Hexagon center longitude
	PlaybackCount     int     `json:"playback_count"`      // Total playbacks in hexagon
	UniqueUsers       int     `json:"unique_users"`        // Unique users in hexagon
	AvgCompletion     float64 `json:"avg_completion"`      // Average completion percentage
	TotalWatchMinutes int     `json:"total_watch_minutes"` // Total watch time in minutes
}

// ArcStats represents user→server connection arc with geodesic distance
// Uses DuckDB's ST_Distance_Sphere for accurate great-circle distance calculations
type ArcStats struct {
	UserLatitude    float64 `json:"user_latitude"`    // User location latitude
	UserLongitude   float64 `json:"user_longitude"`   // User location longitude
	ServerLatitude  float64 `json:"server_latitude"`  // Server location latitude
	ServerLongitude float64 `json:"server_longitude"` // Server location longitude
	City            string  `json:"city"`             // User city
	Country         string  `json:"country"`          // User country
	DistanceKm      float64 `json:"distance_km"`      // Geodesic distance in kilometers
	PlaybackCount   int     `json:"playback_count"`   // Number of playbacks from this location
	UniqueUsers     int     `json:"unique_users"`     // Unique users from this location
	AvgCompletion   float64 `json:"avg_completion"`   // Average completion percentage
	Weight          float64 `json:"weight"`           // Arc visual weight (playback * distance factor)
}

// TemporalSpatialPoint represents a point in both time and space with aggregated metrics
// Uses DuckDB window functions for rolling averages and cumulative aggregations
type TemporalSpatialPoint struct {
	TimeBucket          time.Time `json:"time_bucket"`           // Time bucket (hour/day/week/month)
	H3Index             uint64    `json:"h3_index"`              // H3 hexagon index
	Latitude            float64   `json:"latitude"`              // Hexagon center latitude
	Longitude           float64   `json:"longitude"`             // Hexagon center longitude
	PlaybackCount       int       `json:"playback_count"`        // Playbacks in this time-space bucket
	UniqueUsers         int       `json:"unique_users"`          // Unique users in this time-space bucket
	RollingAvgPlaybacks float64   `json:"rolling_avg_playbacks"` // 7-period rolling average
	CumulativePlaybacks int       `json:"cumulative_playbacks"`  // Cumulative playbacks up to this time
}

// ViewportBounds represents a geographic bounding box for viewport queries
// Used with DuckDB's ST_MakeEnvelope and R-tree spatial index
type ViewportBounds struct {
	West  float64 `json:"west"`  // Western longitude bound
	South float64 `json:"south"` // Southern latitude bound
	East  float64 `json:"east"`  // Eastern longitude bound
	North float64 `json:"north"` // Northern latitude bound
}

// ProximityQuery represents a proximity search centered on a point
// Used with DuckDB's ST_DWithin for fast radius queries
type ProximityQuery struct {
	Latitude  float64 `json:"latitude"`  // Center point latitude
	Longitude float64 `json:"longitude"` // Center point longitude
	RadiusKm  float64 `json:"radius_km"` // Search radius in kilometers
}
