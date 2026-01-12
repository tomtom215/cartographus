// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package tautulli provides data models for Tautulli API responses.
//
// This package contains Go struct definitions for all Tautulli API endpoints
// used by the Cartographus application. Each struct is carefully designed to
// match the Tautulli API v2 response format with appropriate JSON tags and
// field documentation.
//
// # Overview
//
// The package provides 54+ struct types organized by domain:
//
// Activity & Sessions:
//   - TautulliActivity: Current streaming activity (get_activity)
//   - TautulliActivitySession: Individual active session details
//
// Playback History:
//   - TautulliHistory: Paginated playback history (get_history)
//   - TautulliHistoryRecord: Individual playback event
//
// Library Management:
//   - TautulliLibrary: Library metadata (get_libraries)
//   - TautulliLibraryDetails: Detailed library information
//   - TautulliLibraryNames: Library names for filters
//
// User Management:
//   - TautulliUser: User account information (get_user)
//   - TautulliUserStats: User statistics and preferences
//   - TautulliUserWatchTimeStats: Watch time analytics
//
// Server Information:
//   - TautulliServerInfo: Plex server details (get_server_info)
//   - TautulliServerList: Available Plex servers
//   - TautulliSettings: Tautulli configuration
//
// Media Metadata:
//   - TautulliMetadata: Item metadata (get_metadata)
//   - TautulliMetadataChildren: Child items (episodes, tracks)
//   - TautulliRatingKey: Rating key lookups
//
// Content Discovery:
//   - TautulliRecentlyAdded: Recently added content
//   - TautulliHomeStats: Dashboard statistics
//   - TautulliSearch: Search results
//
// Stream Analysis:
//   - TautulliStreamData: Detailed stream information
//   - TautulliStreamType: Stream type enumeration
//   - TautulliResolution: Video resolution data
//
// Collections & Playlists:
//   - TautulliCollection: Collection metadata
//   - TautulliPlaylist: Playlist content
//
// Geographic Data:
//   - TautulliGeoIP: IP geolocation lookup results
//
// # Response Envelope
//
// All Tautulli API responses follow a standard envelope format:
//
//	{
//	    "response": {
//	        "result": "success",
//	        "message": null,
//	        "data": { ... }
//	    }
//	}
//
// Each struct in this package follows this pattern with a Response wrapper
// and Data payload.
//
// # Usage Example
//
// Parsing Tautulli API responses:
//
//	var activity tautulli.TautulliActivity
//	if err := json.Unmarshal(responseBody, &activity); err != nil {
//	    return err
//	}
//
//	if activity.Response.Result != "success" {
//	    return fmt.Errorf("API error: %s", *activity.Response.Message)
//	}
//
//	for _, session := range activity.Response.Data.Sessions {
//	    log.Printf("User %s watching %s", session.FriendlyName, session.Title)
//	}
//
// # Field Naming Conventions
//
// JSON fields use snake_case to match the Tautulli API:
//
//	type Session struct {
//	    SessionKey       string `json:"session_key"`
//	    FriendlyName     string `json:"friendly_name"`
//	    TranscodeDecision string `json:"transcode_decision"`
//	}
//
// # Optional Fields
//
// Optional fields use pointers or omitempty tags:
//
//	Message *string `json:"message,omitempty"`  // Pointer for nullable
//	Email   string  `json:"email,omitempty"`    // Empty string = not present
//
// # Numeric Types
//
// Tautulli uses different numeric types for different fields:
//   - IDs: Usually int (user_id, rating_key)
//   - Timestamps: int64 (Unix epoch)
//   - Durations: int (seconds)
//   - Bitrates: int (kbps)
//   - Percentages: float64 or int (0-100)
//
// # Version Compatibility
//
// These models are compatible with Tautulli API v2. Field additions in
// Tautulli updates are handled gracefully - unknown fields are ignored
// by Go's JSON decoder.
//
// # Thread Safety
//
// All structs are value types with no internal synchronization. They are
// safe for concurrent read access. For concurrent writes, use appropriate
// synchronization or create separate instances.
//
// # See Also
//
//   - internal/sync: TautulliClient using these models
//   - internal/models: Main application models (PlaybackEvent)
//   - internal/api: Tautulli proxy endpoints
//   - https://github.com/Tautulli/Tautulli/wiki/Tautulli-API-Reference: API documentation
package tautulli
