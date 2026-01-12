// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// =============================================================================
// Dedupe Audit Log CRUD Operations (v2.2 - ADR-0022)
// =============================================================================

// InsertDedupeAuditEntry inserts a new deduplication audit entry.
// Called when an event is deduplicated to create an audit trail.
func (db *DB) InsertDedupeAuditEntry(ctx context.Context, entry *models.DedupeAuditEntry) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	if entry.Status == "" {
		entry.Status = "auto_dedupe"
	}

	query := `INSERT INTO dedupe_audit_log (
		id, timestamp, discarded_event_id, discarded_session_key, discarded_correlation_key,
		discarded_source, discarded_started_at, discarded_raw_payload,
		matched_event_id, matched_session_key, matched_correlation_key, matched_source,
		dedupe_reason, dedupe_layer, similarity_score,
		user_id, username, media_type, title, rating_key,
		status, resolved_by, resolved_at, resolution_notes, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.conn.ExecContext(ctx, query,
		entry.ID, entry.Timestamp, entry.DiscardedEventID, entry.DiscardedSessionKey, entry.DiscardedCorrelationKey,
		entry.DiscardedSource, entry.DiscardedStartedAt, entry.DiscardedRawPayload,
		entry.MatchedEventID, entry.MatchedSessionKey, entry.MatchedCorrelationKey, entry.MatchedSource,
		entry.DedupeReason, entry.DedupeLayer, entry.SimilarityScore,
		entry.UserID, entry.Username, entry.MediaType, entry.Title, entry.RatingKey,
		entry.Status, entry.ResolvedBy, entry.ResolvedAt, entry.ResolutionNotes, entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert dedupe audit entry: %w", err)
	}
	return nil
}

// GetDedupeAuditEntry retrieves a specific dedupe audit entry by ID.
func (db *DB) GetDedupeAuditEntry(ctx context.Context, id uuid.UUID) (*models.DedupeAuditEntry, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `SELECT
		id, timestamp, discarded_event_id, discarded_session_key, discarded_correlation_key,
		discarded_source, discarded_started_at, discarded_raw_payload,
		matched_event_id, matched_session_key, matched_correlation_key, matched_source,
		dedupe_reason, dedupe_layer, similarity_score,
		user_id, username, media_type, title, rating_key,
		status, resolved_by, resolved_at, resolution_notes, created_at
	FROM dedupe_audit_log WHERE id = ?`

	row := db.conn.QueryRowContext(ctx, query, id)
	entry := &models.DedupeAuditEntry{}

	err := row.Scan(
		&entry.ID, &entry.Timestamp, &entry.DiscardedEventID, &entry.DiscardedSessionKey, &entry.DiscardedCorrelationKey,
		&entry.DiscardedSource, &entry.DiscardedStartedAt, &entry.DiscardedRawPayload,
		&entry.MatchedEventID, &entry.MatchedSessionKey, &entry.MatchedCorrelationKey, &entry.MatchedSource,
		&entry.DedupeReason, &entry.DedupeLayer, &entry.SimilarityScore,
		&entry.UserID, &entry.Username, &entry.MediaType, &entry.Title, &entry.RatingKey,
		&entry.Status, &entry.ResolvedBy, &entry.ResolvedAt, &entry.ResolutionNotes, &entry.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("dedupe audit entry not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get dedupe audit entry: %w", err)
	}
	return entry, nil
}

// DedupeAuditFilter contains filter options for listing dedupe audit entries.
type DedupeAuditFilter struct {
	UserID       *int
	Source       string
	Status       string
	DedupeReason string
	DedupeLayer  string
	FromTime     *time.Time
	ToTime       *time.Time
	Limit        int
	Offset       int
}

// buildWhereClause builds the WHERE clause and args for dedupe audit queries.
func (filter DedupeAuditFilter) buildWhereClause() (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if filter.UserID != nil {
		conditions = append(conditions, "user_id = ?")
		args = append(args, *filter.UserID)
	}
	if filter.Source != "" {
		conditions = append(conditions, "discarded_source = ?")
		args = append(args, filter.Source)
	}
	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.DedupeReason != "" {
		conditions = append(conditions, "dedupe_reason = ?")
		args = append(args, filter.DedupeReason)
	}
	if filter.DedupeLayer != "" {
		conditions = append(conditions, "dedupe_layer = ?")
		args = append(args, filter.DedupeLayer)
	}
	if filter.FromTime != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, *filter.FromTime)
	}
	if filter.ToTime != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, *filter.ToTime)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	return whereClause, args
}

// getPaginationDefaults returns normalized limit and offset values.
func (filter DedupeAuditFilter) getPaginationDefaults() (int, int) {
	limit := filter.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// ListDedupeAuditEntries lists dedupe audit entries with optional filtering.
func (db *DB) ListDedupeAuditEntries(ctx context.Context, filter DedupeAuditFilter) ([]*models.DedupeAuditEntry, int64, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	whereClause, args := filter.buildWhereClause()

	// Count total matching entries
	countQuery := "SELECT COUNT(*) FROM dedupe_audit_log" + whereClause
	var totalCount int64
	if err := db.conn.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count dedupe audit entries: %w", err)
	}

	limit, offset := filter.getPaginationDefaults()

	// Build main query
	query := `SELECT
		id, timestamp, discarded_event_id, discarded_session_key, discarded_correlation_key,
		discarded_source, discarded_started_at, discarded_raw_payload,
		matched_event_id, matched_session_key, matched_correlation_key, matched_source,
		dedupe_reason, dedupe_layer, similarity_score,
		user_id, username, media_type, title, rating_key,
		status, resolved_by, resolved_at, resolution_notes, created_at
	FROM dedupe_audit_log` + whereClause + ` ORDER BY timestamp DESC LIMIT ? OFFSET ?`

	// Add pagination args
	args = append(args, limit, offset)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list dedupe audit entries: %w", err)
	}
	defer rows.Close()

	var entries []*models.DedupeAuditEntry
	for rows.Next() {
		entry := &models.DedupeAuditEntry{}
		err := rows.Scan(
			&entry.ID, &entry.Timestamp, &entry.DiscardedEventID, &entry.DiscardedSessionKey, &entry.DiscardedCorrelationKey,
			&entry.DiscardedSource, &entry.DiscardedStartedAt, &entry.DiscardedRawPayload,
			&entry.MatchedEventID, &entry.MatchedSessionKey, &entry.MatchedCorrelationKey, &entry.MatchedSource,
			&entry.DedupeReason, &entry.DedupeLayer, &entry.SimilarityScore,
			&entry.UserID, &entry.Username, &entry.MediaType, &entry.Title, &entry.RatingKey,
			&entry.Status, &entry.ResolvedBy, &entry.ResolvedAt, &entry.ResolutionNotes, &entry.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan dedupe audit entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating dedupe audit entries: %w", err)
	}

	return entries, totalCount, nil
}

// UpdateDedupeAuditStatus updates the status of a dedupe audit entry.
// Used when a user confirms or restores a deduplicated event.
func (db *DB) UpdateDedupeAuditStatus(ctx context.Context, id uuid.UUID, status, resolvedBy, notes string) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	now := time.Now()
	query := `UPDATE dedupe_audit_log SET
		status = ?,
		resolved_by = ?,
		resolved_at = ?,
		resolution_notes = ?
	WHERE id = ?`

	result, err := db.conn.ExecContext(ctx, query, status, resolvedBy, now, notes, id)
	if err != nil {
		return fmt.Errorf("failed to update dedupe audit status: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("dedupe audit entry not found: %s", id)
	}

	return nil
}

// queryGroupByCounts executes a GROUP BY query and returns a map of key -> count.
func (db *DB) queryGroupByCounts(ctx context.Context, query string) (map[string]int64, error) {
	result := make(map[string]int64)

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var count int64
		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}
		result[key] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// GetDedupeAuditStats returns aggregate statistics for the dedupe audit dashboard.
func (db *DB) GetDedupeAuditStats(ctx context.Context) (*models.DedupeAuditStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	stats := &models.DedupeAuditStats{
		DedupeByReason: make(map[string]int64),
		DedupeByLayer:  make(map[string]int64),
		DedupeBySource: make(map[string]int64),
	}

	// Get status counts and calculate totals
	if err := db.populateDedupeStatusCounts(ctx, stats); err != nil {
		return nil, err
	}

	// Counts by reason
	reasonCounts, err := db.queryGroupByCounts(ctx, `SELECT dedupe_reason, COUNT(*) FROM dedupe_audit_log GROUP BY dedupe_reason`)
	if err != nil {
		return nil, fmt.Errorf("failed to get reason counts: %w", err)
	}
	stats.DedupeByReason = reasonCounts

	// Counts by layer
	layerCounts, err := db.queryGroupByCounts(ctx, `SELECT dedupe_layer, COUNT(*) FROM dedupe_audit_log GROUP BY dedupe_layer`)
	if err != nil {
		return nil, fmt.Errorf("failed to get layer counts: %w", err)
	}
	stats.DedupeByLayer = layerCounts

	// Counts by source
	sourceCounts, err := db.queryGroupByCounts(ctx, `SELECT discarded_source, COUNT(*) FROM dedupe_audit_log GROUP BY discarded_source`)
	if err != nil {
		return nil, fmt.Errorf("failed to get source counts: %w", err)
	}
	stats.DedupeBySource = sourceCounts

	// Time-based counts
	if err := db.populateDedupeTimeCounts(ctx, stats); err != nil {
		return nil, err
	}

	return stats, nil
}

// populateDedupeStatusCounts populates status-related counts in stats.
func (db *DB) populateDedupeStatusCounts(ctx context.Context, stats *models.DedupeAuditStats) error {
	statusCounts, err := db.queryGroupByCounts(ctx, `SELECT status, COUNT(*) FROM dedupe_audit_log GROUP BY status`)
	if err != nil {
		return fmt.Errorf("failed to get status counts: %w", err)
	}

	for status, count := range statusCounts {
		stats.TotalDeduped += count
		switch status {
		case "auto_dedupe":
			stats.PendingReview = count
		case "user_confirmed":
			stats.UserConfirmed = count
		case "user_restored":
			stats.UserRestored = count
		}
	}

	// Calculate accuracy rate (confirmed / (confirmed + restored))
	if stats.UserConfirmed+stats.UserRestored > 0 {
		stats.AccuracyRate = float64(stats.UserConfirmed) / float64(stats.UserConfirmed+stats.UserRestored) * 100
	}

	return nil
}

// populateDedupeTimeCounts populates time-based counts in stats.
func (db *DB) populateDedupeTimeCounts(ctx context.Context, stats *models.DedupeAuditStats) error {
	now := time.Now()
	last24h := now.Add(-24 * time.Hour)
	last7d := now.Add(-7 * 24 * time.Hour)
	last30d := now.Add(-30 * 24 * time.Hour)

	timeQuery := `SELECT
		(SELECT COUNT(*) FROM dedupe_audit_log WHERE timestamp >= ?) as last_24h,
		(SELECT COUNT(*) FROM dedupe_audit_log WHERE timestamp >= ?) as last_7d,
		(SELECT COUNT(*) FROM dedupe_audit_log WHERE timestamp >= ?) as last_30d`

	if err := db.conn.QueryRowContext(ctx, timeQuery, last24h, last7d, last30d).Scan(
		&stats.Last24Hours, &stats.Last7Days, &stats.Last30Days,
	); err != nil {
		return fmt.Errorf("failed to get time-based counts: %w", err)
	}

	return nil
}

// CleanupDedupeAuditEntries removes old resolved entries based on retention policy.
// Returns the number of entries deleted.
func (db *DB) CleanupDedupeAuditEntries(ctx context.Context, retentionDays int) (int64, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	if retentionDays <= 0 {
		retentionDays = 90 // Default retention
	}

	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)

	// Only delete entries that have been resolved (confirmed or restored)
	query := `DELETE FROM dedupe_audit_log
		WHERE status IN ('user_confirmed', 'user_restored')
		AND resolved_at IS NOT NULL
		AND resolved_at < ?`

	result, err := db.conn.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup dedupe audit entries: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	if affected > 0 {
		logging.Info().
			Int64("affected", affected).
			Int("retention_days", retentionDays).
			Msg("Cleaned up old dedupe audit entries")
	}

	return affected, nil
}
