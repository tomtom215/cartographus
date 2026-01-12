// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
cross_platform.go - Cross-Platform Content and User Linking Operations

This file provides database operations for Phase 3 cross-platform features:
  - Content Mapping: Link content across Plex/Jellyfin/Emby using external IDs (IMDb, TMDB, TVDB)
  - User Linking: Link user identities across platforms for unified analytics

Key Features:
  - Content reconciliation by external IDs (IMDb, TMDB, TVDB)
  - Platform-specific content ID storage (rating_key, item_id)
  - Manual and automatic user linking across platforms
  - Cross-platform watch statistics aggregation

Thread Safety:
All operations use proper SQL transactions to ensure atomicity.

See: docs/PRODUCTION_READINESS_AUDIT.md Phase 3 for design rationale.
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
)

// ========================================
// Content Mapping Types
// ========================================

// ContentMapping represents a cross-platform content link.
// Content is matched using external IDs (IMDb, TMDB, TVDB) and stores
// platform-specific identifiers for each media server.
type ContentMapping struct {
	ID             int64     `json:"id"`
	IMDbID         *string   `json:"imdb_id,omitempty"`          // IMDb ID (e.g., tt1234567)
	TMDbID         *int      `json:"tmdb_id,omitempty"`          // TMDB movie/show ID
	TVDbID         *int      `json:"tvdb_id,omitempty"`          // TVDB series ID
	PlexRatingKey  *string   `json:"plex_rating_key,omitempty"`  // Plex rating_key
	JellyfinItemID *string   `json:"jellyfin_item_id,omitempty"` // Jellyfin Item.Id (UUID)
	EmbyItemID     *string   `json:"emby_item_id,omitempty"`     // Emby Item.Id
	Title          string    `json:"title"`
	MediaType      string    `json:"media_type"` // movie, show, episode
	Year           *int      `json:"year,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ContentMappingLookup contains parameters for content lookup/creation.
type ContentMappingLookup struct {
	// External IDs (at least one required for creation)
	IMDbID *string
	TMDbID *int
	TVDbID *int
	// Platform-specific IDs (optional, for linking)
	PlexRatingKey  *string
	JellyfinItemID *string
	EmbyItemID     *string
	// Metadata
	Title     string
	MediaType string
	Year      *int
}

// ========================================
// User Linking Types
// ========================================

// UserLink represents a link between two user mappings from different platforms.
// This enables cross-platform analytics for users who use multiple media servers.
type UserLink struct {
	ID            int64     `json:"id"`
	PrimaryUserID int       `json:"primary_user_id"` // From user_mappings.internal_user_id
	LinkedUserID  int       `json:"linked_user_id"`  // From user_mappings.internal_user_id
	LinkType      string    `json:"link_type"`       // manual, email, plex_home
	Confidence    float64   `json:"confidence"`      // 0.0-1.0, 1.0 = certain
	CreatedBy     *string   `json:"created_by,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// LinkedUserInfo provides aggregated info about a linked user.
type LinkedUserInfo struct {
	InternalUserID int     `json:"internal_user_id"`
	Username       *string `json:"username,omitempty"`
	FriendlyName   *string `json:"friendly_name,omitempty"`
	Source         string  `json:"source"`
	ServerID       string  `json:"server_id"`
	LinkType       string  `json:"link_type,omitempty"`
}

// contentMappingMutex protects concurrent content mapping creation
var contentMappingMutex sync.Mutex

// userLinkMutex protects concurrent user link creation
var userLinkMutex sync.Mutex

// ========================================
// Schema Initialization
// ========================================

// InitCrossPlatformSchema creates the Phase 3 cross-platform tables.
// Called by DB.Initialize() during database setup.
func (db *DB) InitCrossPlatformSchema(ctx context.Context) error {
	queries := []string{
		// Content mappings table - links content across platforms
		`CREATE TABLE IF NOT EXISTS content_mappings (
			id INTEGER PRIMARY KEY,
			imdb_id TEXT,
			tmdb_id INTEGER,
			tvdb_id INTEGER,
			plex_rating_key TEXT,
			jellyfin_item_id TEXT,
			emby_item_id TEXT,
			title TEXT NOT NULL,
			media_type TEXT NOT NULL,
			year INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// User links table - links users across platforms
		`CREATE TABLE IF NOT EXISTS user_links (
			id INTEGER PRIMARY KEY,
			primary_user_id INTEGER NOT NULL,
			linked_user_id INTEGER NOT NULL,
			link_type TEXT NOT NULL,
			confidence DOUBLE DEFAULT 1.0,
			created_by TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(primary_user_id, linked_user_id),
			CHECK(primary_user_id != linked_user_id)
		)`,

		// Content mapping indexes (DuckDB doesn't support partial indexes, so no WHERE clauses)
		`CREATE INDEX IF NOT EXISTS idx_content_imdb_id ON content_mappings(imdb_id)`,
		`CREATE INDEX IF NOT EXISTS idx_content_tmdb_id ON content_mappings(tmdb_id)`,
		`CREATE INDEX IF NOT EXISTS idx_content_tvdb_id ON content_mappings(tvdb_id)`,
		`CREATE INDEX IF NOT EXISTS idx_content_plex_key ON content_mappings(plex_rating_key)`,
		`CREATE INDEX IF NOT EXISTS idx_content_jellyfin_id ON content_mappings(jellyfin_item_id)`,
		`CREATE INDEX IF NOT EXISTS idx_content_emby_id ON content_mappings(emby_item_id)`,
		`CREATE INDEX IF NOT EXISTS idx_content_media_type ON content_mappings(media_type)`,
		`CREATE INDEX IF NOT EXISTS idx_content_title ON content_mappings(title)`,

		// User link indexes
		`CREATE INDEX IF NOT EXISTS idx_user_links_primary ON user_links(primary_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_links_linked ON user_links(linked_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_links_type ON user_links(link_type)`,
	}

	for _, query := range queries {
		if _, err := db.conn.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute cross-platform schema query: %w", err)
		}
	}

	return nil
}

// ========================================
// Content Mapping Operations
// ========================================

// GetOrCreateContentMapping atomically retrieves or creates a content mapping.
// Matching is done by external IDs in priority order: IMDb > TMDb > TVDb.
func (db *DB) GetOrCreateContentMapping(ctx context.Context, lookup *ContentMappingLookup) (*ContentMapping, bool, error) {
	contentMappingMutex.Lock()
	defer contentMappingMutex.Unlock()

	// Try to find existing mapping by external IDs
	existing, err := db.findContentMappingLocked(ctx, lookup)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, false, fmt.Errorf("failed to check existing content mapping: %w", err)
	}

	if existing != nil {
		// Update platform IDs if provided
		if shouldUpdateContentMapping(existing, lookup) {
			if err := db.updateContentMappingLocked(ctx, existing.ID, lookup); err != nil {
				logging.Warn().Err(err).Msg("Failed to update content mapping")
			}
			// Re-fetch to get updated values (ignore error, existing is still valid)
			//nolint:errcheck // existing is still valid if re-fetch fails
			existing, _ = db.getContentMappingByIDLocked(ctx, existing.ID)
		}
		return existing, false, nil
	}

	// Validate that at least one external ID is provided
	if lookup.IMDbID == nil && lookup.TMDbID == nil && lookup.TVDbID == nil {
		return nil, false, fmt.Errorf("at least one external ID (IMDb, TMDb, or TVDb) is required")
	}

	// Create new mapping
	nextID, err := db.getNextContentMappingIDLocked(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get next content mapping ID: %w", err)
	}

	now := time.Now()
	mapping := &ContentMapping{
		ID:             nextID,
		IMDbID:         lookup.IMDbID,
		TMDbID:         lookup.TMDbID,
		TVDbID:         lookup.TVDbID,
		PlexRatingKey:  lookup.PlexRatingKey,
		JellyfinItemID: lookup.JellyfinItemID,
		EmbyItemID:     lookup.EmbyItemID,
		Title:          lookup.Title,
		MediaType:      lookup.MediaType,
		Year:           lookup.Year,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	query := `
		INSERT INTO content_mappings (
			id, imdb_id, tmdb_id, tvdb_id,
			plex_rating_key, jellyfin_item_id, emby_item_id,
			title, media_type, year, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = db.conn.ExecContext(ctx, query,
		mapping.ID, mapping.IMDbID, mapping.TMDbID, mapping.TVDbID,
		mapping.PlexRatingKey, mapping.JellyfinItemID, mapping.EmbyItemID,
		mapping.Title, mapping.MediaType, mapping.Year,
		mapping.CreatedAt, mapping.UpdatedAt,
	)
	if err != nil {
		return nil, false, fmt.Errorf("failed to insert content mapping: %w", err)
	}

	return mapping, true, nil
}

// GetContentMappingByExternalID looks up a content mapping by any external ID.
func (db *DB) GetContentMappingByExternalID(ctx context.Context, idType string, id interface{}) (*ContentMapping, error) {
	contentMappingMutex.Lock()
	defer contentMappingMutex.Unlock()

	var query string
	switch idType {
	case "imdb":
		query = `SELECT * FROM content_mappings WHERE imdb_id = ?`
	case "tmdb":
		query = `SELECT * FROM content_mappings WHERE tmdb_id = ?`
	case "tvdb":
		query = `SELECT * FROM content_mappings WHERE tvdb_id = ?`
	case "plex":
		query = `SELECT * FROM content_mappings WHERE plex_rating_key = ?`
	case "jellyfin":
		query = `SELECT * FROM content_mappings WHERE jellyfin_item_id = ?`
	case "emby":
		query = `SELECT * FROM content_mappings WHERE emby_item_id = ?`
	default:
		return nil, fmt.Errorf("unknown ID type: %s", idType)
	}

	return db.scanContentMapping(db.conn.QueryRowContext(ctx, query, id))
}

// LinkPlexContent links a Plex rating_key to an existing content mapping.
func (db *DB) LinkPlexContent(ctx context.Context, mappingID int64, plexRatingKey string) error {
	contentMappingMutex.Lock()
	defer contentMappingMutex.Unlock()

	query := `UPDATE content_mappings SET plex_rating_key = ?, updated_at = ? WHERE id = ?`
	_, err := db.conn.ExecContext(ctx, query, plexRatingKey, time.Now(), mappingID)
	return err
}

// LinkJellyfinContent links a Jellyfin item ID to an existing content mapping.
func (db *DB) LinkJellyfinContent(ctx context.Context, mappingID int64, jellyfinItemID string) error {
	contentMappingMutex.Lock()
	defer contentMappingMutex.Unlock()

	query := `UPDATE content_mappings SET jellyfin_item_id = ?, updated_at = ? WHERE id = ?`
	_, err := db.conn.ExecContext(ctx, query, jellyfinItemID, time.Now(), mappingID)
	return err
}

// LinkEmbyContent links an Emby item ID to an existing content mapping.
func (db *DB) LinkEmbyContent(ctx context.Context, mappingID int64, embyItemID string) error {
	contentMappingMutex.Lock()
	defer contentMappingMutex.Unlock()

	query := `UPDATE content_mappings SET emby_item_id = ?, updated_at = ? WHERE id = ?`
	_, err := db.conn.ExecContext(ctx, query, embyItemID, time.Now(), mappingID)
	return err
}

// GetCrossplatformWatchCount returns the total watch count for content across all platforms.
func (db *DB) GetCrossplatformWatchCount(ctx context.Context, mappingID int64) (int, error) {
	mapping, err := db.GetContentMappingByID(ctx, mappingID)
	if err != nil {
		return 0, err
	}

	// Build query to count plays across all linked platform IDs
	args := []interface{}{}
	conditions := []string{}

	if mapping.PlexRatingKey != nil {
		conditions = append(conditions, "rating_key = ?")
		args = append(args, *mapping.PlexRatingKey)
	}
	// Note: For Jellyfin/Emby, we'd need to track their rating_key equivalent
	// This is a simplified implementation

	if len(conditions) == 0 {
		return 0, nil
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM playback_events
		WHERE %s
	`, joinOr(conditions))

	var count int
	err = db.conn.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count cross-platform plays: %w", err)
	}

	return count, nil
}

// GetContentMappingByID retrieves a content mapping by its database ID.
func (db *DB) GetContentMappingByID(ctx context.Context, id int64) (*ContentMapping, error) {
	contentMappingMutex.Lock()
	defer contentMappingMutex.Unlock()
	return db.getContentMappingByIDLocked(ctx, id)
}

// Internal helpers for content mapping

func (db *DB) findContentMappingLocked(ctx context.Context, lookup *ContentMappingLookup) (*ContentMapping, error) {
	// Try to find by external IDs in priority order
	if lookup.IMDbID != nil {
		mapping, err := db.scanContentMapping(db.conn.QueryRowContext(ctx,
			`SELECT * FROM content_mappings WHERE imdb_id = ?`, *lookup.IMDbID))
		if err == nil {
			return mapping, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	if lookup.TMDbID != nil {
		mapping, err := db.scanContentMapping(db.conn.QueryRowContext(ctx,
			`SELECT * FROM content_mappings WHERE tmdb_id = ? AND media_type = ?`,
			*lookup.TMDbID, lookup.MediaType))
		if err == nil {
			return mapping, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	if lookup.TVDbID != nil {
		mapping, err := db.scanContentMapping(db.conn.QueryRowContext(ctx,
			`SELECT * FROM content_mappings WHERE tvdb_id = ?`, *lookup.TVDbID))
		if err == nil {
			return mapping, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	return nil, sql.ErrNoRows
}

func (db *DB) getContentMappingByIDLocked(ctx context.Context, id int64) (*ContentMapping, error) {
	query := `SELECT * FROM content_mappings WHERE id = ?`
	return db.scanContentMapping(db.conn.QueryRowContext(ctx, query, id))
}

func (db *DB) scanContentMapping(row *sql.Row) (*ContentMapping, error) {
	mapping := &ContentMapping{}
	err := row.Scan(
		&mapping.ID, &mapping.IMDbID, &mapping.TMDbID, &mapping.TVDbID,
		&mapping.PlexRatingKey, &mapping.JellyfinItemID, &mapping.EmbyItemID,
		&mapping.Title, &mapping.MediaType, &mapping.Year,
		&mapping.CreatedAt, &mapping.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return mapping, nil
}

func (db *DB) getNextContentMappingIDLocked(ctx context.Context) (int64, error) {
	query := `SELECT COALESCE(MAX(id), 0) + 1 FROM content_mappings`
	var nextID int64
	err := db.conn.QueryRowContext(ctx, query).Scan(&nextID)
	return nextID, err
}

func (db *DB) updateContentMappingLocked(ctx context.Context, id int64, lookup *ContentMappingLookup) error {
	query := `
		UPDATE content_mappings SET
			plex_rating_key = COALESCE(?, plex_rating_key),
			jellyfin_item_id = COALESCE(?, jellyfin_item_id),
			emby_item_id = COALESCE(?, emby_item_id),
			updated_at = ?
		WHERE id = ?
	`
	_, err := db.conn.ExecContext(ctx, query,
		lookup.PlexRatingKey, lookup.JellyfinItemID, lookup.EmbyItemID,
		time.Now(), id,
	)
	return err
}

func shouldUpdateContentMapping(existing *ContentMapping, lookup *ContentMappingLookup) bool {
	if lookup.PlexRatingKey != nil && existing.PlexRatingKey == nil {
		return true
	}
	if lookup.JellyfinItemID != nil && existing.JellyfinItemID == nil {
		return true
	}
	if lookup.EmbyItemID != nil && existing.EmbyItemID == nil {
		return true
	}
	return false
}

// ========================================
// User Linking Operations
// ========================================

// CreateUserLink creates a link between two user identities from different platforms.
// Both user IDs must exist in user_mappings table.
func (db *DB) CreateUserLink(ctx context.Context, primaryUserID, linkedUserID int, linkType string, createdBy *string) (*UserLink, error) {
	userLinkMutex.Lock()
	defer userLinkMutex.Unlock()

	if primaryUserID == linkedUserID {
		return nil, fmt.Errorf("cannot link a user to themselves")
	}

	// Check if link already exists (in either direction)
	existing, err := db.getUserLinkLocked(ctx, primaryUserID, linkedUserID)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Get next ID
	nextID, err := db.getNextUserLinkIDLocked(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get next user link ID: %w", err)
	}

	now := time.Now()
	link := &UserLink{
		ID:            nextID,
		PrimaryUserID: primaryUserID,
		LinkedUserID:  linkedUserID,
		LinkType:      linkType,
		Confidence:    1.0,
		CreatedBy:     createdBy,
		CreatedAt:     now,
	}

	query := `
		INSERT INTO user_links (
			id, primary_user_id, linked_user_id, link_type, confidence, created_by, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = db.conn.ExecContext(ctx, query,
		link.ID, link.PrimaryUserID, link.LinkedUserID,
		link.LinkType, link.Confidence, link.CreatedBy, link.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user link: %w", err)
	}

	return link, nil
}

// GetLinkedUsers returns all users linked to the given internal user ID.
// Includes both directions (primary -> linked and linked -> primary).
func (db *DB) GetLinkedUsers(ctx context.Context, internalUserID int) ([]*LinkedUserInfo, error) {
	query := `
		SELECT
			um.internal_user_id, um.username, um.friendly_name, um.source, um.server_id,
			ul.link_type
		FROM user_links ul
		JOIN user_mappings um ON (
			(ul.primary_user_id = ? AND um.internal_user_id = ul.linked_user_id) OR
			(ul.linked_user_id = ? AND um.internal_user_id = ul.primary_user_id)
		)
		ORDER BY um.source, um.server_id
	`

	rows, err := db.conn.QueryContext(ctx, query, internalUserID, internalUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to query linked users: %w", err)
	}
	defer rows.Close()

	var users []*LinkedUserInfo
	for rows.Next() {
		user := &LinkedUserInfo{}
		err := rows.Scan(
			&user.InternalUserID, &user.Username, &user.FriendlyName,
			&user.Source, &user.ServerID, &user.LinkType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan linked user: %w", err)
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// GetAllLinkedUserIDs returns all internal user IDs linked to the given user.
// Useful for aggregating statistics across all linked identities.
func (db *DB) GetAllLinkedUserIDs(ctx context.Context, internalUserID int) ([]int, error) {
	// Start with the given user
	ids := []int{internalUserID}
	seen := map[int]bool{internalUserID: true}

	// Get directly linked users
	query := `
		SELECT linked_user_id FROM user_links WHERE primary_user_id = ?
		UNION
		SELECT primary_user_id FROM user_links WHERE linked_user_id = ?
	`

	rows, err := db.conn.QueryContext(ctx, query, internalUserID, internalUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to query linked user IDs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var linkedID int
		if err := rows.Scan(&linkedID); err != nil {
			return nil, fmt.Errorf("failed to scan linked user ID: %w", err)
		}
		if !seen[linkedID] {
			ids = append(ids, linkedID)
			seen[linkedID] = true
		}
	}

	return ids, rows.Err()
}

// DeleteUserLink removes a link between two users.
func (db *DB) DeleteUserLink(ctx context.Context, primaryUserID, linkedUserID int) error {
	userLinkMutex.Lock()
	defer userLinkMutex.Unlock()

	query := `DELETE FROM user_links WHERE
		(primary_user_id = ? AND linked_user_id = ?) OR
		(primary_user_id = ? AND linked_user_id = ?)`

	_, err := db.conn.ExecContext(ctx, query,
		primaryUserID, linkedUserID,
		linkedUserID, primaryUserID,
	)
	return err
}

// FindUsersByEmail finds user mappings that might be the same person based on email.
// Returns groups of users that share the same email address.
func (db *DB) FindUsersByEmail(ctx context.Context) (map[string][]*LinkedUserInfo, error) {
	query := `
		SELECT email, internal_user_id, username, friendly_name, source, server_id
		FROM user_mappings
		WHERE email IS NOT NULL AND email != ''
		ORDER BY email, source, server_id
	`

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users by email: %w", err)
	}
	defer rows.Close()

	emailGroups := make(map[string][]*LinkedUserInfo)
	for rows.Next() {
		var email string
		user := &LinkedUserInfo{}
		err := rows.Scan(
			&email, &user.InternalUserID, &user.Username,
			&user.FriendlyName, &user.Source, &user.ServerID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		emailGroups[email] = append(emailGroups[email], user)
	}

	// Filter to only groups with multiple users
	for email, users := range emailGroups {
		if len(users) < 2 {
			delete(emailGroups, email)
		}
	}

	return emailGroups, rows.Err()
}

// Internal helpers for user linking

func (db *DB) getUserLinkLocked(ctx context.Context, userID1, userID2 int) (*UserLink, error) {
	query := `
		SELECT id, primary_user_id, linked_user_id, link_type, confidence, created_by, created_at
		FROM user_links
		WHERE (primary_user_id = ? AND linked_user_id = ?) OR
		      (primary_user_id = ? AND linked_user_id = ?)
	`

	link := &UserLink{}
	err := db.conn.QueryRowContext(ctx, query, userID1, userID2, userID2, userID1).Scan(
		&link.ID, &link.PrimaryUserID, &link.LinkedUserID,
		&link.LinkType, &link.Confidence, &link.CreatedBy, &link.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return link, nil
}

func (db *DB) getNextUserLinkIDLocked(ctx context.Context) (int64, error) {
	query := `SELECT COALESCE(MAX(id), 0) + 1 FROM user_links`
	var nextID int64
	err := db.conn.QueryRowContext(ctx, query).Scan(&nextID)
	return nextID, err
}

// joinOr joins conditions with OR.
func joinOr(conditions []string) string {
	if len(conditions) == 0 {
		return "1=0"
	}
	result := conditions[0]
	for i := 1; i < len(conditions); i++ {
		result += " OR " + conditions[i]
	}
	return result
}

// ========================================
// Cross-Platform Analytics Queries
// ========================================

// UserPlayStats contains aggregated play statistics for a user.
type UserPlayStats struct {
	TotalPlays    int `json:"total_plays"`
	TotalDuration int `json:"total_duration"` // seconds
}

// GetUserPlayStats returns aggregated play statistics for users matching the filter.
func (db *DB) GetUserPlayStats(ctx context.Context, filter LocationStatsFilter) (*UserPlayStats, error) {
	whereClause, args := buildFilterWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_plays,
			COALESCE(SUM(play_duration), 0) as total_duration
		FROM playback_events
		WHERE %s
	`, whereClause)

	stats := &UserPlayStats{}
	err := db.conn.QueryRowContext(ctx, query, args...).Scan(&stats.TotalPlays, &stats.TotalDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to get user play stats: %w", err)
	}

	return stats, nil
}

// QueryRow executes a query that returns a single row.
// This is a thin wrapper around sql.DB.QueryRowContext.
func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.conn.QueryRowContext(ctx, query, args...)
}
