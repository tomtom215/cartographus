// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package models defines data structures used throughout the Cartographus application.

package models

import "time"

// UserMapping maps external user IDs from various media servers to internal integer IDs.
// This enables consistent user tracking across all media server sources (Plex, Jellyfin, Emby, Tautulli)
// and allows correlating the same person across different platforms.
//
// Key Use Cases:
//   - Jellyfin/Emby: Use UUID strings for user IDs, need mapping to integers
//   - Plex: Uses integer IDs but may differ across servers
//   - Multi-server: Same user may have different IDs on different servers
//   - Cross-platform: Optionally link same person across Plex/Jellyfin/Emby
//
// The internal_user_id is auto-assigned on first encounter and used consistently
// throughout the application for analytics, deduplication, and correlation.
type UserMapping struct {
	ID             int64     `json:"id"`
	Source         string    `json:"source"`           // plex, jellyfin, emby, tautulli
	ServerID       string    `json:"server_id"`        // Server instance identifier
	ExternalUserID string    `json:"external_user_id"` // External ID (UUID for Jellyfin/Emby, int string for Plex)
	InternalUserID int       `json:"internal_user_id"` // Auto-assigned internal integer ID
	Username       *string   `json:"username,omitempty"`
	FriendlyName   *string   `json:"friendly_name,omitempty"`
	Email          *string   `json:"email,omitempty"`
	UserThumb      *string   `json:"user_thumb,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// UserMappingLookup represents the parameters needed to look up or create a user mapping.
type UserMappingLookup struct {
	Source         string  // plex, jellyfin, emby, tautulli
	ServerID       string  // Server instance identifier
	ExternalUserID string  // External ID from the source system
	Username       *string // Optional: Username for display
	FriendlyName   *string // Optional: Friendly name for display
	Email          *string // Optional: User email
	UserThumb      *string // Optional: User avatar URL
}

// UserMappingStats provides statistics about user mappings.
type UserMappingStats struct {
	TotalMappings     int            `json:"total_mappings"`
	BySource          map[string]int `json:"by_source"`    // Count per source
	ByServer          map[string]int `json:"by_server"`    // Count per server
	UniqueInternalIDs int            `json:"unique_users"` // Unique internal user IDs
	LastCreatedAt     *time.Time     `json:"last_created_at,omitempty"`
	LastUpdatedAt     *time.Time     `json:"last_updated_at,omitempty"`
}
