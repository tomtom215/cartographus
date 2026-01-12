// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// retryWithBackoff executes a function with exponential backoff on failure.
// The context is used for cancellation during backoff waits.
// If the context is canceled during a wait, the function returns immediately with the context error.
func (m *Manager) retryWithBackoff(ctx context.Context, fn func() error) error {
	var err error
	delay := m.cfg.Sync.RetryDelay

	for attempt := 0; attempt < m.cfg.Sync.RetryAttempts; attempt++ {
		// Check context before attempting operation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err = fn()
		if err == nil {
			return nil
		}

		if attempt < m.cfg.Sync.RetryAttempts-1 {
			logging.Warn().Err(err).Int("attempt", attempt+1).Int("max_attempts", m.cfg.Sync.RetryAttempts).Dur("delay", delay).Msg("Retry attempt")
			// Use cancellable wait instead of time.Sleep
			select {
			case <-time.After(delay):
				// Continue to next attempt
			case <-ctx.Done():
				return ctx.Err()
			}
			delay *= 2
		}
	}

	return fmt.Errorf("max retry attempts reached: %w", err)
}

// stringToPtr converts a non-empty string to a pointer, returns nil for empty strings
func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// intToPtr converts a positive integer to a pointer, returns nil for zero or negative
func intToPtr(i int) *int {
	if i <= 0 {
		return nil
	}
	return &i
}

// mapStringField is a helper to set optional string fields
func mapStringField(value string, target **string) {
	if value != "" && value != "N/A" {
		*target = &value
	}
}

// mapStringPtrField is a helper to set optional string fields from a pointer source
// Used for nullable JSON fields that come from Tautulli API as *string
func mapStringPtrField(value *string, target **string) {
	if value != nil && *value != "" && *value != "N/A" {
		*target = value
	}
}

// mapIntPtrToStringField converts a *int to string and sets the optional string field
// Used for nullable int fields like RatingKey where Tautulli can return null
func mapIntPtrToStringField(value *int, target **string) {
	if value != nil && *value > 0 {
		s := fmt.Sprintf("%d", *value)
		*target = &s
	}
}

// mapIntField is a helper to set optional int fields
func mapIntField(value int, target **int) {
	if value > 0 {
		*target = &value
	}
}

// mapSignedIntPtrField is a helper to set optional int fields from a pointer source
// that can be >= 0 (episode 0, season 0 are valid)
// Used for nullable JSON fields that come from Tautulli API as *int
func mapSignedIntPtrField(value *int, target **int) {
	if value != nil && *value >= 0 {
		*target = value
	}
}

// mapFloat64ToIntPtrField converts a *float64 to *int (truncating to integer)
// Used for fields like watched_status that Tautulli returns as float (0.75 = 75%)
func mapFloat64ToIntPtrField(value *float64, target **int) {
	if value != nil && *value >= 0 {
		intVal := int(*value)
		*target = &intVal
	}
}

// mapInt64Field is a helper to set optional int64 fields
func mapInt64Field(value int64, target **int64) {
	if value > 0 {
		*target = &value
	}
}

// mapSignedIntField is a helper to set optional int fields that can be >= 0
func mapSignedIntField(value int, target **int) {
	if value >= 0 {
		*target = &value
	}
}

// mapInt64PtrField is a helper to set optional int64 fields from a pointer source
// Used for nullable JSON fields that come from Tautulli API as *int64
func mapInt64PtrField(value *int64, target **int64) {
	if value != nil && *value > 0 {
		*target = value
	}
}

// mapIntPtrField is a helper to set optional int fields from a pointer source
// Used for nullable JSON fields that come from Tautulli API as *int
func mapIntPtrField(value *int, target **int) {
	if value != nil && *value > 0 {
		*target = value
	}
}

// getUsername safely extracts username from session user
func getUsername(user *models.PlexSessionUser) string {
	if user != nil {
		return user.Title
	}
	return "Unknown"
}

// getPlayerName safely extracts player name from session player
func getPlayerName(player *models.PlexSessionPlayer) string {
	if player != nil {
		if player.Title != "" {
			return player.Title
		}
		return player.Product
	}
	return "Unknown"
}

// getEffectiveSessionKey returns the session key to use for a Tautulli history record.
// Tautulli API returns NULL AS session_key for historical records (only active sessions have real values).
// This function uses RowID as a fallback identifier to ensure unique session keys for deduplication.
func getEffectiveSessionKey(record *tautulli.TautulliHistoryRecord) string {
	// SessionKey is now a pointer due to nullable JSON
	if record.SessionKey != nil && *record.SessionKey != "" {
		return *record.SessionKey
	}
	// RowID is now a pointer due to nullable JSON - use 0 if nil
	rowID := 0
	if record.RowID != nil {
		rowID = *record.RowID
	}
	return fmt.Sprintf("tautulli-%d", rowID)
}

// generatePlaybackEventCorrelationKey generates a correlation key for a PlaybackEvent.
// This is used in the FALLBACK path when the NATS publisher is unavailable.
//
// CRITICAL (v2.3): This MUST match the format used by MediaEvent.GenerateCorrelationKey()
// in internal/eventprocessor/events.go to ensure cross-source deduplication works correctly.
//
// Format (v2.3): {source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}:{session_key}
// Example: tautulli:default:12345:54321:device123:2024-01-15T10:30:00:tautulli-12345
//
// Components:
//   - source: Media server type (plex, jellyfin, emby, tautulli)
//   - server_id: Server instance identifier (defaults to "default")
//   - user_id: Internal user ID
//   - rating_key: Media content identifier (or title as fallback)
//   - machine_id: Device identifier (or "unknown" if not available)
//   - time_bucket: StartedAt with second precision (handles clock skew)
//   - session_key: Source-specific session identifier (GUARANTEES UNIQUENESS)
//
// The session_key is included to GUARANTEE uniqueness and prevent data loss from
// correlation key collisions. Cross-source dedup uses getCrossSourceKey() which
// extracts only the content-based portion (parts 2-6).
func generatePlaybackEventCorrelationKey(event *models.PlaybackEvent) string {
	// Source defaults to "unknown" if not set
	source := event.Source
	if source == "" {
		source = "unknown"
	}

	// ServerID defaults to "default" - matches MediaEvent.GenerateCorrelationKey()
	serverID := "default"

	// User ID is required and always available
	userID := event.UserID

	// Rating key is the primary content identifier
	ratingKey := ""
	if event.RatingKey != nil && *event.RatingKey != "" {
		ratingKey = *event.RatingKey
	} else {
		// Fallback to title-based key if no rating key
		ratingKey = event.Title
	}

	// MachineID identifies the device - critical for multi-device support
	// When empty, use "unknown" to ensure consistent key format
	machineID := "unknown"
	if event.MachineID != nil && *event.MachineID != "" {
		machineID = *event.MachineID
	}

	// Use exact timestamp (second precision) for correlation key
	// This prevents false deduplication of different sessions within the same time window
	// For cross-source matching, identical playbacks have identical started_at
	timeBucket := event.StartedAt.UTC().Format("2006-01-02T15:04:05")

	// SessionKey is the source-specific session identifier
	// CRITICAL: This guarantees uniqueness and prevents data loss from collisions
	sessionKey := event.SessionKey
	if sessionKey == "" {
		// Use ID as fallback if no session key
		sessionKey = event.ID.String()
	}

	// Format MUST match MediaEvent.GenerateCorrelationKey() in events.go
	return fmt.Sprintf("%s:%s:%d:%s:%s:%s:%s", source, serverID, userID, ratingKey, machineID, timeBucket, sessionKey)
}
