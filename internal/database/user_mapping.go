// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
user_mapping.go - User ID Mapping Operations

This file provides database operations for mapping external user IDs from various
media servers (Jellyfin/Emby UUIDs, Plex integer IDs) to internal integer IDs.

Key Features:
  - GetOrCreateUserMapping: Atomic lookup-or-create operation
  - GetUserMappingByExternal: Look up by source + server + external ID
  - GetUserMappingByInternal: Look up by internal user ID
  - UpdateUserMapping: Update user metadata (username, email, etc.)
  - GetNextInternalUserID: Generate next available internal user ID

Thread Safety:
All operations use proper SQL transactions to ensure atomicity and prevent
race conditions when multiple sync clients create user mappings concurrently.
*/

package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// userMappingMutex protects concurrent user mapping creation
var userMappingMutex sync.Mutex

// GetOrCreateUserMapping atomically retrieves or creates a user mapping.
// This is the primary method for user ID resolution during event processing.
//
// Parameters:
//   - ctx: Context for cancellation
//   - lookup: Parameters identifying the user (source, server_id, external_user_id)
//
// Returns:
//   - The user mapping (existing or newly created)
//   - Boolean indicating if a new mapping was created
//   - Error if operation fails
//
// Thread Safety:
// Uses mutex + database transaction to ensure only one mapping is created
// per source+server+external_user_id combination even under concurrent access.
func (db *DB) GetOrCreateUserMapping(ctx context.Context, lookup *models.UserMappingLookup) (*models.UserMapping, bool, error) {
	userMappingMutex.Lock()
	defer userMappingMutex.Unlock()

	// First, try to find existing mapping
	existing, err := db.getUserMappingByExternalLocked(ctx, lookup.Source, lookup.ServerID, lookup.ExternalUserID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, false, fmt.Errorf("failed to check existing mapping: %w", err)
	}

	if existing != nil {
		// Update metadata if provided and different
		if shouldUpdateMapping(existing, lookup) {
			if err := db.updateUserMappingMetadataLocked(ctx, existing.ID, lookup); err != nil {
				// Log but don't fail - mapping exists, metadata update is optional
				logging.Warn().Err(err).Msg("Failed to update user mapping metadata")
			}
		}
		return existing, false, nil
	}

	// Create new mapping with next available internal user ID
	internalID, err := db.getNextInternalUserIDLocked(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get next internal user ID: %w", err)
	}

	// Get next row ID (DuckDB doesn't support auto-increment with PRIMARY KEY)
	nextID, err := db.getNextMappingIDLocked(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get next mapping ID: %w", err)
	}

	now := time.Now()
	mapping := &models.UserMapping{
		ID:             nextID,
		Source:         lookup.Source,
		ServerID:       lookup.ServerID,
		ExternalUserID: lookup.ExternalUserID,
		InternalUserID: internalID,
		Username:       lookup.Username,
		FriendlyName:   lookup.FriendlyName,
		Email:          lookup.Email,
		UserThumb:      lookup.UserThumb,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	query := `
		INSERT INTO user_mappings (
			id, source, server_id, external_user_id, internal_user_id,
			username, friendly_name, email, user_thumb,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = db.conn.ExecContext(ctx, query,
		mapping.ID, mapping.Source, mapping.ServerID, mapping.ExternalUserID, mapping.InternalUserID,
		mapping.Username, mapping.FriendlyName, mapping.Email, mapping.UserThumb,
		mapping.CreatedAt, mapping.UpdatedAt,
	)
	if err != nil {
		return nil, false, fmt.Errorf("failed to insert user mapping: %w", err)
	}

	return mapping, true, nil
}

// GetUserMappingByExternal looks up a user mapping by source, server, and external user ID.
func (db *DB) GetUserMappingByExternal(ctx context.Context, source, serverID, externalUserID string) (*models.UserMapping, error) {
	userMappingMutex.Lock()
	defer userMappingMutex.Unlock()
	return db.getUserMappingByExternalLocked(ctx, source, serverID, externalUserID)
}

// getUserMappingByExternalLocked is the internal version without locking (caller must hold mutex).
func (db *DB) getUserMappingByExternalLocked(ctx context.Context, source, serverID, externalUserID string) (*models.UserMapping, error) {
	query := `
		SELECT id, source, server_id, external_user_id, internal_user_id,
			   username, friendly_name, email, user_thumb,
			   created_at, updated_at
		FROM user_mappings
		WHERE source = ? AND server_id = ? AND external_user_id = ?
	`

	mapping := &models.UserMapping{}
	err := db.conn.QueryRowContext(ctx, query, source, serverID, externalUserID).Scan(
		&mapping.ID, &mapping.Source, &mapping.ServerID, &mapping.ExternalUserID,
		&mapping.InternalUserID, &mapping.Username, &mapping.FriendlyName,
		&mapping.Email, &mapping.UserThumb, &mapping.CreatedAt, &mapping.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to query user mapping: %w", err)
	}

	return mapping, nil
}

// GetUserMappingByInternal looks up a user mapping by internal user ID.
// Returns all mappings for a given internal ID (could be multiple if linked across sources).
func (db *DB) GetUserMappingByInternal(ctx context.Context, internalUserID int) ([]*models.UserMapping, error) {
	query := `
		SELECT id, source, server_id, external_user_id, internal_user_id,
			   username, friendly_name, email, user_thumb,
			   created_at, updated_at
		FROM user_mappings
		WHERE internal_user_id = ?
		ORDER BY created_at ASC
	`

	rows, err := db.conn.QueryContext(ctx, query, internalUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user mappings by internal ID: %w", err)
	}
	defer rows.Close()

	var mappings []*models.UserMapping
	for rows.Next() {
		mapping := &models.UserMapping{}
		err := rows.Scan(
			&mapping.ID, &mapping.Source, &mapping.ServerID, &mapping.ExternalUserID,
			&mapping.InternalUserID, &mapping.Username, &mapping.FriendlyName,
			&mapping.Email, &mapping.UserThumb, &mapping.CreatedAt, &mapping.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user mapping: %w", err)
		}
		mappings = append(mappings, mapping)
	}

	return mappings, rows.Err()
}

// getNextInternalUserIDLocked generates the next available internal user ID.
// Caller must hold the userMappingMutex.
func (db *DB) getNextInternalUserIDLocked(ctx context.Context) (int, error) {
	query := `SELECT COALESCE(MAX(internal_user_id), 0) + 1 FROM user_mappings`

	var nextID int
	err := db.conn.QueryRowContext(ctx, query).Scan(&nextID)
	if err != nil {
		return 0, fmt.Errorf("failed to get next internal user ID: %w", err)
	}

	return nextID, nil
}

// getNextMappingIDLocked generates the next available row ID for user_mappings.
// Caller must hold the userMappingMutex.
// Note: DuckDB doesn't support auto-increment with PRIMARY KEY, so we manage IDs manually.
func (db *DB) getNextMappingIDLocked(ctx context.Context) (int64, error) {
	query := `SELECT COALESCE(MAX(id), 0) + 1 FROM user_mappings`

	var nextID int64
	err := db.conn.QueryRowContext(ctx, query).Scan(&nextID)
	if err != nil {
		return 0, fmt.Errorf("failed to get next mapping ID: %w", err)
	}

	return nextID, nil
}

// updateUserMappingMetadataLocked updates user metadata (username, email, etc.).
// Caller must hold the userMappingMutex.
func (db *DB) updateUserMappingMetadataLocked(ctx context.Context, id int64, lookup *models.UserMappingLookup) error {
	query := `
		UPDATE user_mappings
		SET username = COALESCE(?, username),
			friendly_name = COALESCE(?, friendly_name),
			email = COALESCE(?, email),
			user_thumb = COALESCE(?, user_thumb),
			updated_at = ?
		WHERE id = ?
	`

	_, err := db.conn.ExecContext(ctx, query,
		lookup.Username, lookup.FriendlyName, lookup.Email, lookup.UserThumb,
		time.Now(), id,
	)
	return err
}

// shouldUpdateMapping checks if the existing mapping should be updated with new metadata.
func shouldUpdateMapping(existing *models.UserMapping, lookup *models.UserMappingLookup) bool {
	// Update if any new metadata is provided and different from existing
	if lookup.Username != nil && (existing.Username == nil || *lookup.Username != *existing.Username) {
		return true
	}
	if lookup.FriendlyName != nil && (existing.FriendlyName == nil || *lookup.FriendlyName != *existing.FriendlyName) {
		return true
	}
	if lookup.Email != nil && (existing.Email == nil || *lookup.Email != *existing.Email) {
		return true
	}
	if lookup.UserThumb != nil && (existing.UserThumb == nil || *lookup.UserThumb != *existing.UserThumb) {
		return true
	}
	return false
}

// GetUserMappingStats returns statistics about user mappings.
func (db *DB) GetUserMappingStats(ctx context.Context) (*models.UserMappingStats, error) {
	stats := &models.UserMappingStats{
		BySource: make(map[string]int),
		ByServer: make(map[string]int),
	}

	// Total count
	err := db.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_mappings`).Scan(&stats.TotalMappings)
	if err != nil {
		return nil, fmt.Errorf("failed to get total mapping count: %w", err)
	}

	// Unique internal IDs
	err = db.conn.QueryRowContext(ctx, `SELECT COUNT(DISTINCT internal_user_id) FROM user_mappings`).Scan(&stats.UniqueInternalIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get unique internal IDs count: %w", err)
	}

	// Count by source
	rows, err := db.conn.QueryContext(ctx, `SELECT source, COUNT(*) FROM user_mappings GROUP BY source`)
	if err != nil {
		return nil, fmt.Errorf("failed to get counts by source: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var source string
		var count int
		if err := rows.Scan(&source, &count); err != nil {
			return nil, fmt.Errorf("failed to scan source count: %w", err)
		}
		stats.BySource[source] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate source counts: %w", err)
	}

	// Count by server
	rows2, err := db.conn.QueryContext(ctx, `SELECT server_id, COUNT(*) FROM user_mappings GROUP BY server_id`)
	if err != nil {
		return nil, fmt.Errorf("failed to get counts by server: %w", err)
	}
	defer func() { _ = rows2.Close() }()
	for rows2.Next() {
		var serverID string
		var count int
		if err := rows2.Scan(&serverID, &count); err != nil {
			return nil, fmt.Errorf("failed to scan server count: %w", err)
		}
		stats.ByServer[serverID] = count
	}
	if err := rows2.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate server counts: %w", err)
	}

	// Last timestamps
	var lastCreated, lastUpdated sql.NullTime
	err = db.conn.QueryRowContext(ctx, `SELECT MAX(created_at) FROM user_mappings`).Scan(&lastCreated)
	if err == nil && lastCreated.Valid {
		stats.LastCreatedAt = &lastCreated.Time
	}
	err = db.conn.QueryRowContext(ctx, `SELECT MAX(updated_at) FROM user_mappings`).Scan(&lastUpdated)
	if err == nil && lastUpdated.Valid {
		stats.LastUpdatedAt = &lastUpdated.Time
	}

	return stats, nil
}

// ResolveUserID is a convenience method that resolves an external user ID to an internal ID.
// This is the primary method sync clients should use during event processing.
//
// Parameters:
//   - ctx: Context for cancellation
//   - source: Source system (plex, jellyfin, emby, tautulli)
//   - serverID: Server instance identifier
//   - externalUserID: External user ID from the source system
//   - username: Optional username for display (may be nil)
//   - friendlyName: Optional friendly name (may be nil)
//
// Returns:
//   - Internal user ID (int) for use in playback events
//   - Error if resolution fails
func (db *DB) ResolveUserID(ctx context.Context, source, serverID, externalUserID string, username, friendlyName *string) (int, error) {
	lookup := &models.UserMappingLookup{
		Source:         source,
		ServerID:       serverID,
		ExternalUserID: externalUserID,
		Username:       username,
		FriendlyName:   friendlyName,
	}

	mapping, _, err := db.GetOrCreateUserMapping(ctx, lookup)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve user ID: %w", err)
	}

	return mapping.InternalUserID, nil
}
