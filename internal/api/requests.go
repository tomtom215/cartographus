// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package api provides HTTP request validation structs with go-playground/validator tags.
// These structs are used to validate incoming API request parameters before processing.
//
// The validation tags follow the go-playground/validator v10 syntax:
//   - required: field must be present and non-zero
//   - min,max: numeric or string length bounds
//   - oneof: value must be one of the specified options
//   - datetime: value must match the specified time format
//   - base64url: value must be valid URL-safe base64 encoded
//   - latitude,longitude: geographic coordinate validation
//   - omitempty: skip validation if field is empty/zero
//
// Example usage:
//
//	req := PlaybacksRequest{
//	    Limit:  getIntParam(r, "limit", 100),
//	    Offset: getIntParam(r, "offset", 0),
//	    Cursor: r.URL.Query().Get("cursor"),
//	}
//	if err := validateRequest(&req); err != nil {
//	    respondError(w, http.StatusBadRequest, err.Code, err.Message, nil)
//	    return
//	}
package api

// PlaybacksRequest represents the validated query parameters for the /playbacks endpoint.
// It supports both cursor-based (recommended) and offset-based (legacy) pagination.
//
// Fields:
//   - Limit: Results per page (1-1000, default from config)
//   - Offset: Legacy offset for backward compatibility (0-1000000)
//   - Cursor: Base64url-encoded cursor for efficient pagination
type PlaybacksRequest struct {
	Limit  int    `validate:"min=1,max=1000"`
	Offset int    `validate:"min=0,max=1000000"`
	Cursor string `validate:"omitempty,base64url"`
}

// LocationsRequest represents the validated query parameters for the /locations endpoint.
// It supports date filtering via either start_date/end_date or days parameter.
//
// Fields:
//   - Limit: Maximum locations to return (1-1000)
//   - Days: Filter by last N days (1-3650, alternative to date range)
//   - StartDate: Start of date range (RFC3339 format)
//   - EndDate: End of date range (RFC3339 format)
//   - Users: Comma-separated list of usernames to filter
//   - MediaTypes: Comma-separated list of media types to filter
type LocationsRequest struct {
	Limit      int    `validate:"min=1,max=1000"`
	Days       int    `validate:"omitempty,min=1,max=3650"`
	StartDate  string `validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	EndDate    string `validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	Users      string // Comma-separated, no validation needed
	MediaTypes string // Comma-separated, no validation needed
}

// LoginRequestValidation represents the validated request body for the /auth/login endpoint.
// Note: This is named differently from models.LoginRequest to avoid conflicts.
//
// Fields:
//   - Username: Required user login name
//   - Password: Required user password
//   - RememberMe: Optional flag to extend session duration
type LoginRequestValidation struct {
	Username   string `validate:"required,min=1"`
	Password   string `validate:"required,min=1"`
	RememberMe bool
}

// CreateBackupRequestValidation represents the validated request body for POST /backup.
// Note: Named differently from CreateBackupRequest in handlers_backup.go to avoid conflicts.
//
// Fields:
//   - Type: Backup type (optional, one of: full, database, config)
//   - Notes: Optional description (max 500 characters)
type CreateBackupRequestValidation struct {
	Type  string `validate:"omitempty,oneof=full database config"`
	Notes string `validate:"omitempty,max=500"`
}

// RestoreBackupRequestValidation represents the validated request body for POST /backups/{id}/restore.
// All fields are optional boolean flags with sensible defaults.
type RestoreBackupRequestValidation struct {
	ValidateOnly           bool
	CreatePreRestoreBackup bool
	RestoreDatabase        bool
	RestoreConfig          bool
	ForceRestore           bool
	VerifyAfterRestore     bool
}

// SetRetentionPolicyRequestValidation represents the validated request body for PUT /backup/retention.
// All numeric fields must be non-negative.
//
// Fields:
//   - MinCount: Minimum backups to retain (>= 0)
//   - MaxCount: Maximum backups to keep (>= 0)
//   - MaxAgeDays: Maximum age in days (>= 0)
//   - KeepRecentHours: Keep backups from last N hours (>= 0)
//   - KeepDailyForDays: Keep daily backups for N days (>= 0)
//   - KeepWeeklyForWeeks: Keep weekly backups for N weeks (>= 0)
//   - KeepMonthlyForMonths: Keep monthly backups for N months (>= 0)
type SetRetentionPolicyRequestValidation struct {
	MinCount             int `validate:"min=0"`
	MaxCount             int `validate:"min=0"`
	MaxAgeDays           int `validate:"min=0"`
	KeepRecentHours      int `validate:"min=0"`
	KeepDailyForDays     int `validate:"min=0"`
	KeepWeeklyForWeeks   int `validate:"min=0"`
	KeepMonthlyForMonths int `validate:"min=0"`
}

// SetScheduleConfigRequestValidation represents the validated request body for PUT /backup/schedule.
// Configures automatic backup scheduling.
//
// Fields:
//   - Enabled: Whether scheduled backups are enabled
//   - IntervalHours: Hours between backups (1-720, i.e., 1 hour to 30 days)
//   - PreferredHour: Hour of day for daily+ backups (0-23)
//   - BackupType: Type of backup to create (full, database, config)
//   - PreSyncBackup: Create backup before sync operations
type SetScheduleConfigRequestValidation struct {
	Enabled       bool
	IntervalHours int    `validate:"min=1,max=720"`
	PreferredHour int    `validate:"min=0,max=23"`
	BackupType    string `validate:"omitempty,oneof=full database config"`
	PreSyncBackup bool
}

// SpatialViewportRequest represents the validated query parameters for /spatial/viewport.
// All bounding box coordinates are required and must be valid geographic coordinates.
//
// Fields:
//   - West: Western longitude boundary (-180 to 180)
//   - South: Southern latitude boundary (-90 to 90)
//   - East: Eastern longitude boundary (-180 to 180)
//   - North: Northern latitude boundary (-90 to 90)
//   - StartDate: Optional start date filter (validated by parseDateFilter)
//   - EndDate: Optional end date filter (validated by parseDateFilter)
type SpatialViewportRequest struct {
	West      float64 `validate:"min=-180,max=180"`
	South     float64 `validate:"min=-90,max=90"`
	East      float64 `validate:"min=-180,max=180"`
	North     float64 `validate:"min=-90,max=90"`
	StartDate string  // Date validation done by parseDateFilter
	EndDate   string  // Date validation done by parseDateFilter
}

// SpatialHexagonsRequest represents the validated query parameters for /spatial/hexagons.
// Resolution determines the size of H3 hexagons for spatial aggregation.
//
// Fields:
//   - Resolution: H3 resolution level (0-15, default 7)
//   - StartDate: Optional start date filter (validated by parseDateFilter)
//   - EndDate: Optional end date filter (validated by parseDateFilter)
type SpatialHexagonsRequest struct {
	Resolution int    `validate:"min=0,max=15"`
	StartDate  string // Date validation done by parseDateFilter
	EndDate    string // Date validation done by parseDateFilter
}

// SpatialNearbyRequest represents the validated query parameters for /spatial/nearby.
// It finds locations within a specified radius of a center point.
//
// Fields:
//   - Lat: Center latitude (-90 to 90)
//   - Lon: Center longitude (-180 to 180)
//   - Radius: Search radius in kilometers (1-20000, default 100)
//   - StartDate: Optional start date filter (validated by parseDateFilter)
//   - EndDate: Optional end date filter (validated by parseDateFilter)
type SpatialNearbyRequest struct {
	Lat       float64 `validate:"latitude"`
	Lon       float64 `validate:"longitude"`
	Radius    float64 `validate:"min=1,max=20000"`
	StartDate string  // Date validation done by parseDateFilter
	EndDate   string  // Date validation done by parseDateFilter
}

// SpatialTemporalDensityRequest represents the validated query parameters for /spatial/temporal-density.
// It returns temporal-spatial playback density with rolling aggregations.
//
// Fields:
//   - Interval: Time interval (hour, day, week, month)
//   - Resolution: H3 resolution (6-8, default 7)
//   - StartDate: Optional start date filter (validated by parseDateFilter)
//   - EndDate: Optional end date filter (validated by parseDateFilter)
type SpatialTemporalDensityRequest struct {
	Interval   string `validate:"omitempty,oneof=hour day week month"`
	Resolution int    `validate:"min=6,max=8"`
	StartDate  string // Date validation done by parseDateFilter
	EndDate    string // Date validation done by parseDateFilter
}

// ExportPlaybacksCSVRequest represents the validated query parameters for /export/playbacks/csv.
// Supports higher limits than regular pagination for bulk exports.
//
// Fields:
//   - Limit: Maximum records to export (1-100000)
//   - Offset: Starting offset (0-1000000)
type ExportPlaybacksCSVRequest struct {
	Limit  int `validate:"min=1,max=100000"`
	Offset int `validate:"min=0,max=1000000"`
}

// AnalyticsRequest represents common validated query parameters for analytics endpoints.
// Used across multiple analytics handlers with consistent validation.
//
// Fields:
//   - StartDate: Optional start date filter (validated by parseDateFilter)
//   - EndDate: Optional end date filter (validated by parseDateFilter)
//   - Days: Filter by last N days (1-3650)
//   - Users: Comma-separated list of usernames
//   - MediaTypes: Comma-separated list of media types
type AnalyticsRequest struct {
	StartDate  string // Date validation done by parseDateFilter
	EndDate    string // Date validation done by parseDateFilter
	Days       int    `validate:"omitempty,min=1,max=3650"`
	Users      string // Comma-separated, no validation needed
	MediaTypes string // Comma-separated, no validation needed
}

// TileRequest represents the validated path parameters for /tiles/{z}/{x}/{y}.pbf.
// Tile coordinates are validated against zoom-level bounds.
//
// Fields:
//   - Z: Zoom level (0-22)
//   - X: Tile X coordinate (0 to 2^z - 1)
//   - Y: Tile Y coordinate (0 to 2^z - 1)
type TileRequest struct {
	Z int `validate:"min=0,max=22"`
	X int `validate:"min=0"`
	Y int `validate:"min=0"`
}
