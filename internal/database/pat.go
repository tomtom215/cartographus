// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package database provides database operations for the Cartographus application.
//
// pat.go - Personal Access Token Database Operations
//
// This file contains CRUD operations for Personal Access Tokens (PATs).
// PATs enable programmatic API access with scoped permissions.
//
// Security:
//   - Token hashes are stored, never plaintext tokens
//   - All operations are parameterized (SQL injection safe)
//   - Usage logging provides audit trail
//
// Performance:
//   - Indexed on user_id, token_prefix for fast lookups
//   - Batch operations for stats queries
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/tomtom215/cartographus/internal/models"
)

// CreatePAT creates a new Personal Access Token in the database.
func (db *DB) CreatePAT(ctx context.Context, token *models.PersonalAccessToken) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	scopesJSON, err := json.Marshal(token.Scopes)
	if err != nil {
		return fmt.Errorf("failed to marshal scopes: %w", err)
	}

	ipAllowlistSQL, err := marshalIPAllowlist(token.IPAllowlist)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO personal_access_tokens (
			id, user_id, username, name, description,
			token_prefix, token_hash, scopes,
			expires_at, ip_allowlist, created_at,
			last_used_at, last_used_ip, use_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = db.conn.ExecContext(ctx, query,
		token.ID, token.UserID, token.Username, token.Name, token.Description,
		token.TokenPrefix, token.TokenHash, string(scopesJSON),
		token.ExpiresAt, ipAllowlistSQL, token.CreatedAt,
		token.LastUsedAt, token.LastUsedIP, token.UseCount,
	)
	if err != nil {
		return fmt.Errorf("failed to insert PAT: %w", err)
	}

	return nil
}

// GetPATByID retrieves a PAT by its ID.
func (db *DB) GetPATByID(ctx context.Context, id string) (*models.PersonalAccessToken, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `
		SELECT
			id, user_id, username, name, description,
			token_prefix, token_hash, scopes::VARCHAR,
			expires_at, last_used_at, last_used_ip, use_count,
			ip_allowlist::VARCHAR, created_at,
			revoked_at, revoked_by, revoke_reason
		FROM personal_access_tokens
		WHERE id = ?
	`

	row := db.conn.QueryRowContext(ctx, query, id)
	return scanPAT(row)
}

// GetPATsByUserID retrieves all PATs for a user.
func (db *DB) GetPATsByUserID(ctx context.Context, userID string) ([]models.PersonalAccessToken, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `
		SELECT
			id, user_id, username, name, description,
			token_prefix, token_hash, scopes::VARCHAR,
			expires_at, last_used_at, last_used_ip, use_count,
			ip_allowlist::VARCHAR, created_at,
			revoked_at, revoked_by, revoke_reason
		FROM personal_access_tokens
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := db.conn.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query PATs: %w", err)
	}
	defer rows.Close()

	var tokens []models.PersonalAccessToken
	for rows.Next() {
		token, err := scanPATRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan PAT: %w", err)
		}
		tokens = append(tokens, *token)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating PATs: %w", err)
	}

	if tokens == nil {
		tokens = []models.PersonalAccessToken{}
	}

	return tokens, nil
}

// GetPATByPrefix retrieves PATs matching a prefix (for validation optimization).
func (db *DB) GetPATByPrefix(ctx context.Context, prefix string) ([]models.PersonalAccessToken, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `
		SELECT
			id, user_id, username, name, description,
			token_prefix, token_hash, scopes::VARCHAR,
			expires_at, last_used_at, last_used_ip, use_count,
			ip_allowlist::VARCHAR, created_at,
			revoked_at, revoked_by, revoke_reason
		FROM personal_access_tokens
		WHERE token_prefix = ?
	`

	rows, err := db.conn.QueryContext(ctx, query, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to query PATs by prefix: %w", err)
	}
	defer rows.Close()

	var tokens []models.PersonalAccessToken
	for rows.Next() {
		token, err := scanPATRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan PAT: %w", err)
		}
		tokens = append(tokens, *token)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating PATs: %w", err)
	}

	return tokens, nil
}

// UpdatePAT updates a PAT in the database.
func (db *DB) UpdatePAT(ctx context.Context, token *models.PersonalAccessToken) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	scopesJSON, err := json.Marshal(token.Scopes)
	if err != nil {
		return fmt.Errorf("failed to marshal scopes: %w", err)
	}

	ipAllowlistSQL, err := marshalIPAllowlist(token.IPAllowlist)
	if err != nil {
		return err
	}

	query := `
		UPDATE personal_access_tokens SET
			name = ?,
			description = ?,
			token_prefix = ?,
			token_hash = ?,
			scopes = ?,
			expires_at = ?,
			last_used_at = ?,
			last_used_ip = ?,
			use_count = ?,
			ip_allowlist = ?
		WHERE id = ?
	`

	result, err := db.conn.ExecContext(ctx, query,
		token.Name, token.Description,
		token.TokenPrefix, token.TokenHash, string(scopesJSON),
		token.ExpiresAt, token.LastUsedAt, token.LastUsedIP,
		token.UseCount, ipAllowlistSQL, token.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update PAT: %w", err)
	}

	return checkRowsAffected(result, "PAT not found")
}

// RevokePAT revokes a PAT.
func (db *DB) RevokePAT(ctx context.Context, id string, revokedBy string, reason string) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `
		UPDATE personal_access_tokens SET
			revoked_at = ?,
			revoked_by = ?,
			revoke_reason = ?
		WHERE id = ? AND revoked_at IS NULL
	`

	result, err := db.conn.ExecContext(ctx, query, time.Now(), revokedBy, reason, id)
	if err != nil {
		return fmt.Errorf("failed to revoke PAT: %w", err)
	}

	return checkRowsAffected(result, "PAT not found or already revoked")
}

// DeletePAT permanently deletes a PAT.
func (db *DB) DeletePAT(ctx context.Context, id string) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `DELETE FROM personal_access_tokens WHERE id = ?`

	result, err := db.conn.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete PAT: %w", err)
	}

	return checkRowsAffected(result, "PAT not found")
}

// LogPATUsage logs a PAT usage event.
func (db *DB) LogPATUsage(ctx context.Context, log *models.PATUsageLog) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	if log.ID == "" {
		log.ID = uuid.New().String()
	}

	query := `
		INSERT INTO pat_usage_log (
			id, timestamp, token_id, user_id, action,
			endpoint, method, ip_address, user_agent,
			success, error_code, response_time_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.conn.ExecContext(ctx, query,
		log.ID, log.Timestamp, log.TokenID, log.UserID, log.Action,
		log.Endpoint, log.Method, log.IPAddress, log.UserAgent,
		log.Success, log.ErrorCode, log.ResponseTimeMS,
	)
	if err != nil {
		return fmt.Errorf("failed to log PAT usage: %w", err)
	}

	return nil
}

// GetPATUsageLogs retrieves usage logs for a token.
func (db *DB) GetPATUsageLogs(ctx context.Context, tokenID string, limit int) ([]models.PATUsageLog, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT
			id, timestamp, token_id, user_id, action,
			endpoint, method, ip_address, user_agent,
			success, error_code, response_time_ms
		FROM pat_usage_log
		WHERE token_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := db.conn.QueryContext(ctx, query, tokenID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query PAT usage logs: %w", err)
	}
	defer rows.Close()

	var logs []models.PATUsageLog
	for rows.Next() {
		log, err := scanUsageLogRow(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, *log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating PAT usage logs: %w", err)
	}

	if logs == nil {
		logs = []models.PATUsageLog{}
	}

	return logs, nil
}

// GetPATStats returns aggregated PAT statistics for a user.
func (db *DB) GetPATStats(ctx context.Context, userID string) (*models.PATStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `
		SELECT
			COUNT(*) AS total_tokens,
			COUNT(CASE WHEN revoked_at IS NULL AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP) THEN 1 END) AS active_tokens,
			COUNT(CASE WHEN expires_at IS NOT NULL AND expires_at <= CURRENT_TIMESTAMP AND revoked_at IS NULL THEN 1 END) AS expired_tokens,
			COUNT(CASE WHEN revoked_at IS NOT NULL THEN 1 END) AS revoked_tokens,
			COALESCE(SUM(use_count), 0) AS total_usage,
			MAX(last_used_at) AS last_used_at
		FROM personal_access_tokens
		WHERE user_id = ?
	`

	var stats models.PATStats
	var lastUsedAt sql.NullTime

	err := db.conn.QueryRowContext(ctx, query, userID).Scan(
		&stats.TotalTokens,
		&stats.ActiveTokens,
		&stats.ExpiredTokens,
		&stats.RevokedTokens,
		&stats.TotalUsage,
		&lastUsedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get PAT stats: %w", err)
	}

	if lastUsedAt.Valid {
		stats.LastUsedAt = lastUsedAt.Time
	}

	return &stats, nil
}

// scanPAT scans a single PAT from a row.
func scanPAT(row *sql.Row) (*models.PersonalAccessToken, error) {
	var scannedData patScanData

	err := row.Scan(
		&scannedData.id, &scannedData.userID, &scannedData.username, &scannedData.name, &scannedData.description,
		&scannedData.tokenPrefix, &scannedData.tokenHash, &scannedData.scopesJSON,
		&scannedData.expiresAt, &scannedData.lastUsedAt, &scannedData.lastUsedIP, &scannedData.useCount,
		&scannedData.ipAllowlistJSON, &scannedData.createdAt,
		&scannedData.revokedAt, &scannedData.revokedBy, &scannedData.revokeReason,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan PAT: %w", err)
	}

	return buildPATFromScanData(&scannedData)
}

// scanPATRow scans a single PAT from a row iterator.
func scanPATRow(rows *sql.Rows) (*models.PersonalAccessToken, error) {
	var scannedData patScanData

	err := rows.Scan(
		&scannedData.id, &scannedData.userID, &scannedData.username, &scannedData.name, &scannedData.description,
		&scannedData.tokenPrefix, &scannedData.tokenHash, &scannedData.scopesJSON,
		&scannedData.expiresAt, &scannedData.lastUsedAt, &scannedData.lastUsedIP, &scannedData.useCount,
		&scannedData.ipAllowlistJSON, &scannedData.createdAt,
		&scannedData.revokedAt, &scannedData.revokedBy, &scannedData.revokeReason,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan PAT: %w", err)
	}

	return buildPATFromScanData(&scannedData)
}

// patScanData holds scanned database values before conversion to models.PersonalAccessToken.
type patScanData struct {
	id, userID, username, name  string
	description                 sql.NullString
	tokenPrefix, tokenHash      string
	scopesJSON, ipAllowlistJSON sql.NullString
	expiresAt, lastUsedAt       sql.NullTime
	revokedAt                   sql.NullTime
	lastUsedIP                  sql.NullString
	revokedBy, revokeReason     sql.NullString
	useCount                    int
	createdAt                   time.Time
}

// buildPATFromScanData converts scanned database values into a PersonalAccessToken.
func buildPATFromScanData(data *patScanData) (*models.PersonalAccessToken, error) {
	token := &models.PersonalAccessToken{
		ID:          data.id,
		UserID:      data.userID,
		Username:    data.username,
		Name:        data.name,
		TokenPrefix: data.tokenPrefix,
		TokenHash:   data.tokenHash,
		UseCount:    data.useCount,
		CreatedAt:   data.createdAt,
	}

	// Populate nullable strings
	populateNullableStrings(token, data)

	// Populate nullable times
	populateNullableTimes(token, data)

	// Parse JSON fields
	if err := parseTokenJSON(token, data.scopesJSON, data.ipAllowlistJSON); err != nil {
		return nil, err
	}

	return token, nil
}

// populateNullableStrings sets nullable string fields on the token.
func populateNullableStrings(token *models.PersonalAccessToken, data *patScanData) {
	token.Description = data.description.String
	token.LastUsedIP = data.lastUsedIP.String
	token.RevokedBy = data.revokedBy.String
	token.RevokeReason = data.revokeReason.String
}

// populateNullableTimes sets nullable time fields on the token.
func populateNullableTimes(token *models.PersonalAccessToken, data *patScanData) {
	if data.expiresAt.Valid {
		token.ExpiresAt = &data.expiresAt.Time
	}
	if data.lastUsedAt.Valid {
		token.LastUsedAt = &data.lastUsedAt.Time
	}
	if data.revokedAt.Valid {
		token.RevokedAt = &data.revokedAt.Time
	}
}

// parseTokenJSON parses the scopes and IP allowlist JSON fields.
func parseTokenJSON(token *models.PersonalAccessToken, scopesJSON, ipAllowlistJSON sql.NullString) error {
	// Parse scopes JSON
	if scopesJSON.Valid && scopesJSON.String != "" {
		if err := json.Unmarshal([]byte(scopesJSON.String), &token.Scopes); err != nil {
			return fmt.Errorf("failed to parse scopes: %w", err)
		}
	}

	// Parse IP allowlist JSON
	if ipAllowlistJSON.Valid && ipAllowlistJSON.String != "" {
		if err := json.Unmarshal([]byte(ipAllowlistJSON.String), &token.IPAllowlist); err != nil {
			return fmt.Errorf("failed to parse ip_allowlist: %w", err)
		}
	}

	return nil
}

// scanUsageLogRow scans a single PAT usage log from a row iterator.
func scanUsageLogRow(rows *sql.Rows) (*models.PATUsageLog, error) {
	var log models.PATUsageLog
	var endpoint, method, ip, userAgent, errorCode sql.NullString
	var responseTime sql.NullInt64

	err := rows.Scan(
		&log.ID, &log.Timestamp, &log.TokenID, &log.UserID, &log.Action,
		&endpoint, &method, &ip, &userAgent,
		&log.Success, &errorCode, &responseTime,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan PAT usage log: %w", err)
	}

	log.Endpoint = endpoint.String
	log.Method = method.String
	log.IPAddress = ip.String
	log.UserAgent = userAgent.String
	log.ErrorCode = errorCode.String
	if responseTime.Valid {
		log.ResponseTimeMS = int(responseTime.Int64)
	}

	return &log, nil
}

// marshalIPAllowlist marshals an IP allowlist to a NullString for database storage.
func marshalIPAllowlist(ipAllowlist []string) (sql.NullString, error) {
	var ipAllowlistSQL sql.NullString
	if len(ipAllowlist) > 0 {
		ipAllowlistJSON, err := json.Marshal(ipAllowlist)
		if err != nil {
			return ipAllowlistSQL, fmt.Errorf("failed to marshal ip_allowlist: %w", err)
		}
		ipAllowlistSQL = sql.NullString{String: string(ipAllowlistJSON), Valid: true}
	}
	return ipAllowlistSQL, nil
}

// checkRowsAffected verifies that at least one row was affected by an operation.
func checkRowsAffected(result sql.Result, notFoundMsg string) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%s", notFoundMsg)
	}
	return nil
}
