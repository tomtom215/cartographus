// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/models"
)

// Media server errors
var (
	ErrServerNotFound   = errors.New("media server not found")
	ErrServerIDConflict = errors.New("server with this server_id already exists")
	ErrImmutableServer  = errors.New("cannot modify server configured via environment variables")
)

// CreateMediaServer creates a new media server configuration in the database.
// The URL and token should already be encrypted before calling this method.
func (db *DB) CreateMediaServer(ctx context.Context, server *models.MediaServer) error {
	if server.ID == "" {
		server.ID = uuid.New().String()
	}
	if server.CreatedAt.IsZero() {
		server.CreatedAt = time.Now()
	}
	server.UpdatedAt = server.CreatedAt
	if server.Source == "" {
		server.Source = models.ServerSourceUI
	}
	if server.SessionPollingInterval == "" {
		server.SessionPollingInterval = "30s"
	}
	if server.Settings == "" {
		server.Settings = "{}"
	}

	query := `INSERT INTO media_servers (
		id, platform, name, url_encrypted, token_encrypted, server_id,
		enabled, settings, realtime_enabled, webhooks_enabled,
		session_polling_enabled, session_polling_interval, source,
		created_by, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.conn.ExecContext(ctx, query,
		server.ID, server.Platform, server.Name, server.URLEncrypted, server.TokenEncrypted, server.ServerID,
		server.Enabled, server.Settings, server.RealtimeEnabled, server.WebhooksEnabled,
		server.SessionPollingEnabled, server.SessionPollingInterval, server.Source,
		server.CreatedBy, server.CreatedAt, server.UpdatedAt,
	)
	if err != nil {
		// Check for unique constraint violation on server_id
		if isUniqueConstraintError(err) {
			return ErrServerIDConflict
		}
		return fmt.Errorf("failed to create media server: %w", err)
	}

	return nil
}

// GetMediaServer retrieves a media server by ID.
func (db *DB) GetMediaServer(ctx context.Context, id string) (*models.MediaServer, error) {
	query := `SELECT
		id, platform, name, url_encrypted, token_encrypted, server_id,
		enabled, settings, realtime_enabled, webhooks_enabled,
		session_polling_enabled, session_polling_interval, source,
		created_by, created_at, updated_at,
		last_sync_at, last_sync_status, last_error, last_error_at
	FROM media_servers WHERE id = ?`

	row := db.conn.QueryRowContext(ctx, query, id)
	return scanMediaServer(row)
}

// GetMediaServerByServerID retrieves a media server by its unique server_id (for deduplication).
func (db *DB) GetMediaServerByServerID(ctx context.Context, serverID string) (*models.MediaServer, error) {
	query := `SELECT
		id, platform, name, url_encrypted, token_encrypted, server_id,
		enabled, settings, realtime_enabled, webhooks_enabled,
		session_polling_enabled, session_polling_interval, source,
		created_by, created_at, updated_at,
		last_sync_at, last_sync_status, last_error, last_error_at
	FROM media_servers WHERE server_id = ?`

	row := db.conn.QueryRowContext(ctx, query, serverID)
	return scanMediaServer(row)
}

// ListMediaServers retrieves all media servers, optionally filtered by platform.
func (db *DB) ListMediaServers(ctx context.Context, platform string, enabledOnly bool) ([]models.MediaServer, error) {
	query := `SELECT
		id, platform, name, url_encrypted, token_encrypted, server_id,
		enabled, settings, realtime_enabled, webhooks_enabled,
		session_polling_enabled, session_polling_interval, source,
		created_by, created_at, updated_at,
		last_sync_at, last_sync_status, last_error, last_error_at
	FROM media_servers WHERE 1=1`

	args := []any{}
	if platform != "" {
		query += " AND platform = ?"
		args = append(args, platform)
	}
	if enabledOnly {
		query += " AND enabled = true"
	}
	query += " ORDER BY created_at DESC"

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list media servers: %w", err)
	}
	defer rows.Close()

	servers := make([]models.MediaServer, 0)
	for rows.Next() {
		server, err := scanMediaServerRows(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan media server: %w", err)
		}
		servers = append(servers, *server)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating media servers: %w", err)
	}

	return servers, nil
}

// UpdateMediaServer updates an existing media server configuration.
func (db *DB) UpdateMediaServer(ctx context.Context, server *models.MediaServer) error {
	server.UpdatedAt = time.Now()

	query := `UPDATE media_servers SET
		name = ?, url_encrypted = ?, token_encrypted = ?, server_id = ?,
		enabled = ?, settings = ?, realtime_enabled = ?, webhooks_enabled = ?,
		session_polling_enabled = ?, session_polling_interval = ?,
		updated_at = ?
	WHERE id = ? AND source != 'env'`

	result, err := db.conn.ExecContext(ctx, query,
		server.Name, server.URLEncrypted, server.TokenEncrypted, server.ServerID,
		server.Enabled, server.Settings, server.RealtimeEnabled, server.WebhooksEnabled,
		server.SessionPollingEnabled, server.SessionPollingInterval,
		server.UpdatedAt, server.ID,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrServerIDConflict
		}
		return fmt.Errorf("failed to update media server: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		// Check if it exists but is immutable
		existing, err := db.GetMediaServer(ctx, server.ID)
		if err == nil && existing != nil && existing.Source == models.ServerSourceEnv {
			return ErrImmutableServer
		}
		return ErrServerNotFound
	}

	return nil
}

// UpdateMediaServerSyncStatus updates the sync status fields for a server.
func (db *DB) UpdateMediaServerSyncStatus(ctx context.Context, id string, status string, syncError string) error {
	now := time.Now()

	var query string
	var args []any

	if syncError != "" {
		query = `UPDATE media_servers SET
			last_sync_status = ?, last_error = ?, last_error_at = ?, updated_at = ?
		WHERE id = ?`
		args = []any{status, syncError, now, now, id}
	} else {
		query = `UPDATE media_servers SET
			last_sync_at = ?, last_sync_status = ?, last_error = NULL, last_error_at = NULL, updated_at = ?
		WHERE id = ?`
		args = []any{now, status, now, id}
	}

	result, err := db.conn.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrServerNotFound
	}

	return nil
}

// DeleteMediaServer removes a media server from the database.
// Only UI-added servers can be deleted (env-var servers are immutable).
func (db *DB) DeleteMediaServer(ctx context.Context, id string) error {
	// Check if server exists and is not immutable
	existing, err := db.GetMediaServer(ctx, id)
	if err != nil {
		if errors.Is(err, ErrServerNotFound) {
			return err
		}
		return fmt.Errorf("failed to check server existence: %w", err)
	}
	if existing.Source == models.ServerSourceEnv {
		return ErrImmutableServer
	}

	query := `DELETE FROM media_servers WHERE id = ? AND source != 'env'`
	result, err := db.conn.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete media server: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrServerNotFound
	}

	return nil
}

// EnableMediaServer enables a disabled media server.
func (db *DB) EnableMediaServer(ctx context.Context, id string) error {
	return db.setServerEnabled(ctx, id, true)
}

// DisableMediaServer disables an enabled media server.
func (db *DB) DisableMediaServer(ctx context.Context, id string) error {
	return db.setServerEnabled(ctx, id, false)
}

func (db *DB) setServerEnabled(ctx context.Context, id string, enabled bool) error {
	query := `UPDATE media_servers SET enabled = ?, updated_at = ? WHERE id = ?`
	result, err := db.conn.ExecContext(ctx, query, enabled, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update server enabled status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrServerNotFound
	}

	return nil
}

// CreateMediaServerAudit logs a server configuration change.
func (db *DB) CreateMediaServerAudit(ctx context.Context, audit *models.MediaServerAudit) error {
	if audit.ID == "" {
		audit.ID = uuid.New().String()
	}
	if audit.CreatedAt.IsZero() {
		audit.CreatedAt = time.Now()
	}

	query := `INSERT INTO media_server_audit (
		id, server_id, action, user_id, username, changes, ip_address, user_agent, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.conn.ExecContext(ctx, query,
		audit.ID, audit.ServerID, audit.Action, audit.UserID, audit.Username,
		audit.Changes, audit.IPAddress, audit.UserAgent, audit.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// ListMediaServerAuditLogs retrieves audit logs for a server.
func (db *DB) ListMediaServerAuditLogs(ctx context.Context, serverID string, limit int) ([]models.MediaServerAudit, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT
		id, server_id, action, user_id, username, changes, ip_address, user_agent, created_at
	FROM media_server_audit
	WHERE server_id = ?
	ORDER BY created_at DESC
	LIMIT ?`

	rows, err := db.conn.QueryContext(ctx, query, serverID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	audits := make([]models.MediaServerAudit, 0)
	for rows.Next() {
		var audit models.MediaServerAudit
		var changes any // DuckDB returns JSON as map[string]any
		var ipAddr, userAgent sql.NullString
		err := rows.Scan(
			&audit.ID, &audit.ServerID, &audit.Action, &audit.UserID, &audit.Username,
			&changes, &ipAddr, &userAgent, &audit.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		audit.Changes = jsonToString(changes)
		if ipAddr.Valid {
			audit.IPAddress = ipAddr.String
		}
		if userAgent.Valid {
			audit.UserAgent = userAgent.String
		}
		audits = append(audits, audit)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit logs: %w", err)
	}

	return audits, nil
}

// CountMediaServers returns the count of media servers by platform.
func (db *DB) CountMediaServers(ctx context.Context) (map[string]int, error) {
	query := `SELECT platform, COUNT(*) as count FROM media_servers GROUP BY platform`

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count servers: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var platform string
		var count int
		if err := rows.Scan(&platform, &count); err != nil {
			return nil, fmt.Errorf("failed to scan count: %w", err)
		}
		counts[platform] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating server counts: %w", err)
	}

	return counts, nil
}

// scanMediaServer scans a single row into a MediaServer struct.
func scanMediaServer(row *sql.Row) (*models.MediaServer, error) {
	var server models.MediaServer
	var serverID, createdBy sql.NullString
	var settings any // DuckDB returns JSON as map[string]any
	var lastSyncAt, lastErrorAt sql.NullTime
	var lastSyncStatus, lastError sql.NullString

	err := row.Scan(
		&server.ID, &server.Platform, &server.Name, &server.URLEncrypted, &server.TokenEncrypted, &serverID,
		&server.Enabled, &settings, &server.RealtimeEnabled, &server.WebhooksEnabled,
		&server.SessionPollingEnabled, &server.SessionPollingInterval, &server.Source,
		&createdBy, &server.CreatedAt, &server.UpdatedAt,
		&lastSyncAt, &lastSyncStatus, &lastError, &lastErrorAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrServerNotFound
		}
		return nil, fmt.Errorf("failed to scan media server: %w", err)
	}

	if serverID.Valid {
		server.ServerID = serverID.String
	}
	server.Settings = jsonToString(settings)
	if createdBy.Valid {
		server.CreatedBy = createdBy.String
	}
	if lastSyncAt.Valid {
		server.LastSyncAt = &lastSyncAt.Time
	}
	if lastSyncStatus.Valid {
		server.LastSyncStatus = lastSyncStatus.String
	}
	if lastError.Valid {
		server.LastError = lastError.String
	}
	if lastErrorAt.Valid {
		server.LastErrorAt = &lastErrorAt.Time
	}

	return &server, nil
}

// scanMediaServerRows scans rows into a MediaServer struct.
func scanMediaServerRows(rows *sql.Rows) (*models.MediaServer, error) {
	var server models.MediaServer
	var serverID, createdBy sql.NullString
	var settings any // DuckDB returns JSON as map[string]any
	var lastSyncAt, lastErrorAt sql.NullTime
	var lastSyncStatus, lastError sql.NullString

	err := rows.Scan(
		&server.ID, &server.Platform, &server.Name, &server.URLEncrypted, &server.TokenEncrypted, &serverID,
		&server.Enabled, &settings, &server.RealtimeEnabled, &server.WebhooksEnabled,
		&server.SessionPollingEnabled, &server.SessionPollingInterval, &server.Source,
		&createdBy, &server.CreatedAt, &server.UpdatedAt,
		&lastSyncAt, &lastSyncStatus, &lastError, &lastErrorAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan media server: %w", err)
	}

	if serverID.Valid {
		server.ServerID = serverID.String
	}
	server.Settings = jsonToString(settings)
	if createdBy.Valid {
		server.CreatedBy = createdBy.String
	}
	if lastSyncAt.Valid {
		server.LastSyncAt = &lastSyncAt.Time
	}
	if lastSyncStatus.Valid {
		server.LastSyncStatus = lastSyncStatus.String
	}
	if lastError.Valid {
		server.LastError = lastError.String
	}
	if lastErrorAt.Valid {
		server.LastErrorAt = &lastErrorAt.Time
	}

	return &server, nil
}

// jsonToString converts a DuckDB JSON value to a string.
func jsonToString(v any) string {
	if v == nil {
		return "{}"
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case map[string]any:
		// DuckDB returns JSON as map[string]any
		bytes, err := json.Marshal(val)
		if err != nil {
			return "{}"
		}
		return string(bytes)
	default:
		return "{}"
	}
}

// isUniqueConstraintError checks if an error is a unique constraint violation.
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	// DuckDB unique constraint error messages contain "UNIQUE constraint" or "Duplicate key"
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "unique constraint") || strings.Contains(errMsg, "duplicate key")
}
