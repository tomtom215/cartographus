// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
rbac.go - RBAC Database Operations

This file provides database operations for Role-Based Access Control (RBAC),
including user role management and audit logging.

Key Operations:
  - GetUserRole: Retrieve a user's role by user ID
  - SetUserRole: Create or update a user's role
  - DeleteUserRole: Remove a user's role (soft delete via is_active)
  - ListUserRoles: List all user roles with optional filtering
  - AuditRoleChange: Record a role change in the audit log
  - GetRoleAuditLog: Retrieve audit log entries

Thread Safety:
All operations use proper mutex locking to ensure atomicity and prevent
race conditions when multiple requests modify roles concurrently.

Role Hierarchy (enforced by Casbin, not this file):
  - viewer: Default, read-only access to own data
  - editor: Write access, inherits viewer
  - admin: Full access, inherits editor
*/

package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// rbacMutex protects concurrent role operations
var rbacMutex sync.Mutex

// ErrRoleNotFound is returned when a user role is not found.
var ErrRoleNotFound = errors.New("role not found")

// ErrInvalidRole is returned when an invalid role name is provided.
var ErrInvalidRole = errors.New("invalid role")

// scanUserRoleRow scans a database row into a UserRole struct, handling nullable fields.
func scanUserRoleRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.UserRole, error) {
	role := &models.UserRole{}
	var assignedBy sql.NullString
	var expiresAt sql.NullTime
	var metadata sql.NullString

	err := scanner.Scan(
		&role.ID, &role.UserID, &role.Username, &role.Role,
		&assignedBy, &role.AssignedAt, &expiresAt, &role.IsActive, &metadata,
	)
	if err != nil {
		return nil, err
	}

	applyNullableUserRoleFields(role, assignedBy, expiresAt, metadata)
	return role, nil
}

// applyNullableUserRoleFields applies nullable fields to a UserRole.
func applyNullableUserRoleFields(role *models.UserRole, assignedBy sql.NullString, expiresAt sql.NullTime, metadata sql.NullString) {
	if assignedBy.Valid {
		role.AssignedBy = assignedBy.String
	}
	if expiresAt.Valid {
		role.ExpiresAt = &expiresAt.Time
	}
	if metadata.Valid {
		role.Metadata = &metadata.String
	}
}

// scanAuditEntryRow scans a database row into a RoleAuditEntry struct, handling nullable fields.
func scanAuditEntryRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.RoleAuditEntry, error) {
	entry := &models.RoleAuditEntry{}
	var actorUsername, targetUsername, oldRole, newRole sql.NullString
	var reason, ipAddress, userAgent, sessionID sql.NullString

	err := scanner.Scan(
		&entry.ID, &entry.Timestamp, &entry.ActorID, &actorUsername, &entry.Action,
		&entry.TargetUserID, &targetUsername, &oldRole, &newRole,
		&reason, &ipAddress, &userAgent, &sessionID,
	)
	if err != nil {
		return nil, err
	}

	applyNullableAuditFields(entry, actorUsername, targetUsername, oldRole, newRole, reason, ipAddress, userAgent, sessionID)
	return entry, nil
}

// applyNullableAuditFields applies nullable fields to a RoleAuditEntry.
func applyNullableAuditFields(entry *models.RoleAuditEntry, actorUsername, targetUsername, oldRole, newRole, reason, ipAddress, userAgent, sessionID sql.NullString) {
	if actorUsername.Valid {
		entry.ActorUsername = actorUsername.String
	}
	if targetUsername.Valid {
		entry.TargetUsername = targetUsername.String
	}
	if oldRole.Valid {
		entry.OldRole = oldRole.String
	}
	if newRole.Valid {
		entry.NewRole = newRole.String
	}
	if reason.Valid {
		entry.Reason = reason.String
	}
	if ipAddress.Valid {
		entry.IPAddress = ipAddress.String
	}
	if userAgent.Valid {
		entry.UserAgent = userAgent.String
	}
	if sessionID.Valid {
		entry.SessionID = sessionID.String
	}
}

// convertToNullableFields converts pointer fields to database nullable types.
func convertToNullableFields(expiresAt *time.Time, metadata *string) (interface{}, interface{}) {
	var expiresAtVal interface{}
	if expiresAt != nil {
		expiresAtVal = *expiresAt
	}

	var metadataVal interface{}
	if metadata != nil {
		metadataVal = *metadata
	}

	return expiresAtVal, metadataVal
}

// GetUserRole retrieves a user's active role by user ID.
// Returns ErrRoleNotFound if no active role exists for the user.
// Expired roles are treated as not found.
//
// Parameters:
//   - ctx: Context for cancellation
//   - userID: The user's unique identifier
//
// Returns:
//   - The user's role record if found and effective
//   - ErrRoleNotFound if no active, non-expired role exists
//   - Error if database operation fails
func (db *DB) GetUserRole(ctx context.Context, userID string) (*models.UserRole, error) {
	query := `
		SELECT id, user_id, username, role, assigned_by, assigned_at,
		       expires_at, is_active, CAST(metadata AS TEXT) as metadata
		FROM user_roles
		WHERE user_id = ? AND is_active = TRUE
		ORDER BY assigned_at DESC
		LIMIT 1
	`

	row := db.conn.QueryRowContext(ctx, query, userID)
	role, err := scanUserRoleRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to query user role: %w", err)
	}

	// Check if role is expired
	if role.IsExpired() {
		return nil, ErrRoleNotFound
	}

	return role, nil
}

// GetEffectiveRole returns the effective role for a user, defaulting to "viewer" if none found.
// This is the primary method for authorization checks.
func (db *DB) GetEffectiveRole(ctx context.Context, userID string) (string, error) {
	role, err := db.GetUserRole(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			return models.RoleViewer, nil // Default to viewer
		}
		return "", err
	}
	return role.Role, nil
}

// SetUserRole creates or updates a user's role.
// If the user already has a role, it is updated. Otherwise, a new role is created.
// An audit log entry is automatically created for the change.
//
// Parameters:
//   - ctx: Context for cancellation
//   - role: The role to set (must have valid UserID, Username, Role)
//   - actorID: The user ID of who is making this change (for audit)
//   - actorUsername: The username of who is making this change
//   - reason: Optional reason for the change
//
// Returns:
//   - The created/updated role record
//   - Error if validation fails or database operation fails
func (db *DB) SetUserRole(ctx context.Context, role *models.UserRole, actorID, actorUsername, reason string) (*models.UserRole, error) {
	if !models.IsValidRole(role.Role) {
		return nil, ErrInvalidRole
	}

	rbacMutex.Lock()
	defer rbacMutex.Unlock()

	existing, err := db.getUserRoleLocked(ctx, role.UserID)
	if err != nil && !errors.Is(err, ErrRoleNotFound) {
		return nil, fmt.Errorf("failed to check existing role: %w", err)
	}

	oldRole, action := db.upsertUserRoleLocked(ctx, role, existing)
	if action == "" {
		return nil, fmt.Errorf("failed to set user role")
	}

	db.auditRoleSetOperation(ctx, actorID, actorUsername, action, role, oldRole, reason)
	return role, nil
}

// upsertUserRoleLocked creates or updates a user role.
// Returns the old role and action taken (assign or update), or empty string on error.
// Caller must hold rbacMutex.
func (db *DB) upsertUserRoleLocked(ctx context.Context, role *models.UserRole, existing *models.UserRole) (string, string) {
	if existing != nil {
		if err := db.updateUserRoleLocked(ctx, existing.ID, role); err != nil {
			logging.Error().Err(err).Msg("Failed to update user role")
			return "", ""
		}
		role.ID = existing.ID
		role.AssignedAt = existing.AssignedAt
		return existing.Role, models.AuditActionUpdate
	}

	if err := db.createUserRoleLocked(ctx, role); err != nil {
		logging.Error().Err(err).Msg("Failed to create user role")
		return "", ""
	}
	return "", models.AuditActionAssign
}

// auditRoleSetOperation creates an audit log entry for a role set operation.
// Caller must hold rbacMutex.
func (db *DB) auditRoleSetOperation(ctx context.Context, actorID, actorUsername, action string, role *models.UserRole, oldRole, reason string) {
	auditEntry := models.NewRoleAuditEntry(actorID, actorUsername, action, role.UserID, role.Username)
	auditEntry.OldRole = oldRole
	auditEntry.NewRole = role.Role
	auditEntry.Reason = reason

	if err := db.auditRoleChangeLocked(ctx, auditEntry); err != nil {
		logging.Warn().Err(err).Msg("Failed to audit role change")
	}
}

// getUserRoleLocked retrieves a user's role without acquiring the mutex.
// Caller must hold rbacMutex.
func (db *DB) getUserRoleLocked(ctx context.Context, userID string) (*models.UserRole, error) {
	query := `
		SELECT id, user_id, username, role, assigned_by, assigned_at,
		       expires_at, is_active, CAST(metadata AS TEXT) as metadata
		FROM user_roles
		WHERE user_id = ? AND is_active = TRUE
		ORDER BY assigned_at DESC
		LIMIT 1
	`

	row := db.conn.QueryRowContext(ctx, query, userID)
	role, err := scanUserRoleRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to query user role: %w", err)
	}

	return role, nil
}

// createUserRoleLocked creates a new user role.
// Caller must hold rbacMutex.
func (db *DB) createUserRoleLocked(ctx context.Context, role *models.UserRole) error {
	// Get next ID (DuckDB doesn't support auto-increment with PRIMARY KEY)
	nextID, err := db.getNextRoleIDLocked(ctx)
	if err != nil {
		return fmt.Errorf("failed to get next role ID: %w", err)
	}

	role.ID = nextID
	if role.AssignedAt.IsZero() {
		role.AssignedAt = time.Now()
	}
	if !role.IsActive {
		role.IsActive = true // New roles are active by default
	}

	query := `
		INSERT INTO user_roles (id, user_id, username, role, assigned_by, assigned_at, expires_at, is_active, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	expiresAt, metadata := convertToNullableFields(role.ExpiresAt, role.Metadata)

	_, err = db.conn.ExecContext(ctx, query,
		role.ID, role.UserID, role.Username, role.Role,
		role.AssignedBy, role.AssignedAt, expiresAt, role.IsActive, metadata,
	)
	return err
}

// updateUserRoleLocked updates an existing user role.
// Caller must hold rbacMutex.
func (db *DB) updateUserRoleLocked(ctx context.Context, id int64, role *models.UserRole) error {
	query := `
		UPDATE user_roles
		SET role = ?, assigned_by = ?, expires_at = ?, is_active = ?, metadata = ?
		WHERE id = ?
	`

	expiresAt, metadata := convertToNullableFields(role.ExpiresAt, role.Metadata)

	_, err := db.conn.ExecContext(ctx, query,
		role.Role, role.AssignedBy, expiresAt, role.IsActive, metadata, id,
	)
	return err
}

// getNextRoleIDLocked generates the next available role ID.
// Caller must hold rbacMutex.
func (db *DB) getNextRoleIDLocked(ctx context.Context) (int64, error) {
	query := `SELECT COALESCE(MAX(id), 0) + 1 FROM user_roles`

	var nextID int64
	err := db.conn.QueryRowContext(ctx, query).Scan(&nextID)
	if err != nil {
		return 0, fmt.Errorf("failed to get next role ID: %w", err)
	}

	return nextID, nil
}

// DeleteUserRole soft-deletes a user's role by setting is_active to false.
// An audit log entry is automatically created for the change.
//
// Parameters:
//   - ctx: Context for cancellation
//   - userID: The user whose role to delete
//   - actorID: The user ID of who is making this change
//   - actorUsername: The username of who is making this change
//   - reason: Optional reason for the deletion
//
// Returns:
//   - Error if user has no role or database operation fails
func (db *DB) DeleteUserRole(ctx context.Context, userID, actorID, actorUsername, reason string) error {
	rbacMutex.Lock()
	defer rbacMutex.Unlock()

	// Get existing role for audit
	existing, err := db.getUserRoleLocked(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get existing role: %w", err)
	}

	// Soft delete by setting is_active to false
	query := `UPDATE user_roles SET is_active = FALSE WHERE id = ?`
	_, err = db.conn.ExecContext(ctx, query, existing.ID)
	if err != nil {
		return fmt.Errorf("failed to delete user role: %w", err)
	}

	// Audit the change
	auditEntry := models.NewRoleAuditEntry(actorID, actorUsername, models.AuditActionRevoke, userID, existing.Username)
	auditEntry.OldRole = existing.Role
	auditEntry.Reason = reason

	if err := db.auditRoleChangeLocked(ctx, auditEntry); err != nil {
		logging.Warn().Err(err).Msg("Failed to audit role deletion")
	}

	return nil
}

// ListUserRoles retrieves all user roles with optional filtering.
//
// Parameters:
//   - ctx: Context for cancellation
//   - activeOnly: If true, only return active roles
//   - roleFilter: If non-empty, only return roles matching this value
//
// Returns:
//   - List of user roles
//   - Error if database operation fails
func (db *DB) ListUserRoles(ctx context.Context, activeOnly bool, roleFilter string) ([]*models.UserRole, error) {
	query, args := buildUserRolesQuery(activeOnly, roleFilter)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list user roles: %w", err)
	}
	defer rows.Close()

	var roles []*models.UserRole
	for rows.Next() {
		role, err := scanUserRoleRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, rows.Err()
}

// buildUserRolesQuery constructs the query for listing user roles with filters.
func buildUserRolesQuery(activeOnly bool, roleFilter string) (string, []interface{}) {
	query := `
		SELECT id, user_id, username, role, assigned_by, assigned_at,
		       expires_at, is_active, CAST(metadata AS TEXT) as metadata
		FROM user_roles
		WHERE 1=1
	`
	args := []interface{}{}

	if activeOnly {
		query += ` AND is_active = TRUE`
	}
	if roleFilter != "" {
		query += ` AND role = ?`
		args = append(args, roleFilter)
	}

	query += ` ORDER BY assigned_at DESC`
	return query, args
}

// GetRoleStats returns statistics about role assignments.
func (db *DB) GetRoleStats(ctx context.Context) (*models.RoleStats, error) {
	stats := &models.RoleStats{
		ByRole: make(map[string]int),
	}

	// Gather all statistics
	if err := db.populateTotalUsers(ctx, stats); err != nil {
		return nil, err
	}
	if err := db.populateRoleCounts(ctx, stats); err != nil {
		return nil, err
	}
	if err := db.populateCountsByRole(ctx, stats); err != nil {
		return nil, err
	}

	return stats, nil
}

// populateTotalUsers sets the total number of unique users.
func (db *DB) populateTotalUsers(ctx context.Context, stats *models.RoleStats) error {
	query := `SELECT COUNT(DISTINCT user_id) FROM user_roles`
	if err := db.conn.QueryRowContext(ctx, query).Scan(&stats.TotalUsers); err != nil {
		return fmt.Errorf("failed to get total users: %w", err)
	}
	return nil
}

// populateRoleCounts sets the active, expired, and inactive role counts.
func (db *DB) populateRoleCounts(ctx context.Context, stats *models.RoleStats) error {
	type countQuery struct {
		query  string
		target *int
		errMsg string
	}

	queries := []countQuery{
		{
			query:  `SELECT COUNT(*) FROM user_roles WHERE is_active = TRUE`,
			target: &stats.ActiveRoles,
			errMsg: "failed to get active roles count",
		},
		{
			query: `SELECT COUNT(*) FROM user_roles
				WHERE is_active = TRUE AND expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP`,
			target: &stats.ExpiredRoles,
			errMsg: "failed to get expired roles count",
		},
		{
			query:  `SELECT COUNT(*) FROM user_roles WHERE is_active = FALSE`,
			target: &stats.InactiveRoles,
			errMsg: "failed to get inactive roles count",
		},
	}

	for _, q := range queries {
		if err := db.conn.QueryRowContext(ctx, q.query).Scan(q.target); err != nil {
			return fmt.Errorf("%s: %w", q.errMsg, err)
		}
	}

	return nil
}

// populateCountsByRole sets the counts grouped by role type.
func (db *DB) populateCountsByRole(ctx context.Context, stats *models.RoleStats) error {
	query := `SELECT role, COUNT(*) FROM user_roles WHERE is_active = TRUE GROUP BY role`
	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to get counts by role: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var role string
		var count int
		if err := rows.Scan(&role, &count); err != nil {
			return fmt.Errorf("failed to scan role count: %w", err)
		}
		stats.ByRole[role] = count
	}

	return rows.Err()
}

// AuditRoleChange records a role change in the audit log.
// This is the public API for recording role changes from external sources.
func (db *DB) AuditRoleChange(ctx context.Context, entry *models.RoleAuditEntry) error {
	rbacMutex.Lock()
	defer rbacMutex.Unlock()
	return db.auditRoleChangeLocked(ctx, entry)
}

// auditRoleChangeLocked records a role change in the audit log.
// Caller must hold rbacMutex.
func (db *DB) auditRoleChangeLocked(ctx context.Context, entry *models.RoleAuditEntry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	query := `
		INSERT INTO role_audit_log (
			id, timestamp, actor_id, actor_username, action,
			target_user_id, target_username, old_role, new_role,
			reason, ip_address, user_agent, session_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.conn.ExecContext(ctx, query,
		entry.ID, entry.Timestamp, entry.ActorID, entry.ActorUsername, entry.Action,
		entry.TargetUserID, entry.TargetUsername, entry.OldRole, entry.NewRole,
		entry.Reason, entry.IPAddress, entry.UserAgent, entry.SessionID,
	)
	if err != nil {
		return fmt.Errorf("failed to insert audit entry: %w", err)
	}

	return nil
}

// GetRoleAuditLog retrieves audit log entries with optional filtering.
//
// Parameters:
//   - ctx: Context for cancellation
//   - userID: If non-empty, only return entries for this target user
//   - limit: Maximum number of entries to return (0 = no limit)
//   - offset: Number of entries to skip
//
// Returns:
//   - List of audit log entries
//   - Error if database operation fails
func (db *DB) GetRoleAuditLog(ctx context.Context, userID string, limit, offset int) ([]*models.RoleAuditEntry, error) {
	query := buildAuditLogQuery(userID, limit, offset)
	args := buildAuditLogArgs(userID)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit log: %w", err)
	}
	defer rows.Close()

	var entries []*models.RoleAuditEntry
	for rows.Next() {
		entry, err := scanAuditEntryRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// buildAuditLogQuery constructs the query for retrieving audit log entries.
func buildAuditLogQuery(userID string, limit, offset int) string {
	query := `
		SELECT id, timestamp, actor_id, actor_username, action,
		       target_user_id, target_username, old_role, new_role,
		       reason, ip_address, user_agent, session_id
		FROM role_audit_log
	`

	if userID != "" {
		query += ` WHERE target_user_id = ?`
	}

	query += ` ORDER BY timestamp DESC`

	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}
	if offset > 0 {
		query += fmt.Sprintf(` OFFSET %d`, offset)
	}

	return query
}

// buildAuditLogArgs constructs the arguments for the audit log query.
func buildAuditLogArgs(userID string) []interface{} {
	args := []interface{}{}
	if userID != "" {
		args = append(args, userID)
	}
	return args
}

// GetAuditLogCount returns the total number of audit log entries,
// optionally filtered by target user.
func (db *DB) GetAuditLogCount(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM role_audit_log`
	args := []interface{}{}

	if userID != "" {
		query += ` WHERE target_user_id = ?`
		args = append(args, userID)
	}

	var count int
	err := db.conn.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count audit entries: %w", err)
	}

	return count, nil
}

// ExpireRoles finds and marks expired roles as inactive.
// This is intended to be called periodically (e.g., by a background job).
// Returns the number of roles that were expired.
func (db *DB) ExpireRoles(ctx context.Context, systemActorID string) (int, error) {
	rbacMutex.Lock()
	defer rbacMutex.Unlock()

	expiredRoles, err := db.findExpiredRolesLocked(ctx)
	if err != nil {
		return 0, err
	}

	db.processExpiredRoles(ctx, expiredRoles, systemActorID)
	return len(expiredRoles), nil
}

// expiredRoleInfo holds information about an expired role.
type expiredRoleInfo struct {
	ID       int64
	UserID   string
	Username string
	Role     string
}

// findExpiredRolesLocked queries for roles that are active but past expiration.
// Caller must hold rbacMutex.
func (db *DB) findExpiredRolesLocked(ctx context.Context) ([]expiredRoleInfo, error) {
	query := `
		SELECT id, user_id, username, role
		FROM user_roles
		WHERE is_active = TRUE AND expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP
	`

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to find expired roles: %w", err)
	}
	defer rows.Close()

	var expiredRoles []expiredRoleInfo
	for rows.Next() {
		var r expiredRoleInfo
		if err := rows.Scan(&r.ID, &r.UserID, &r.Username, &r.Role); err != nil {
			return nil, fmt.Errorf("failed to scan expired role: %w", err)
		}
		expiredRoles = append(expiredRoles, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate expired roles: %w", err)
	}

	return expiredRoles, nil
}

// processExpiredRoles deactivates each expired role and creates audit entries.
// Caller must hold rbacMutex.
func (db *DB) processExpiredRoles(ctx context.Context, expiredRoles []expiredRoleInfo, systemActorID string) {
	for _, r := range expiredRoles {
		if err := db.deactivateRoleLocked(ctx, r.ID); err != nil {
			logging.Warn().Err(err).Int64("role_id", r.ID).Msg("Failed to deactivate expired role")
			continue
		}

		auditEntry := createExpirationAuditEntry(systemActorID, r)
		if err := db.auditRoleChangeLocked(ctx, auditEntry); err != nil {
			logging.Warn().Err(err).Str("user_id", r.UserID).Msg("Failed to audit role expiration")
		}
	}
}

// deactivateRoleLocked marks a role as inactive.
// Caller must hold rbacMutex.
func (db *DB) deactivateRoleLocked(ctx context.Context, roleID int64) error {
	query := `UPDATE user_roles SET is_active = FALSE WHERE id = ?`
	_, err := db.conn.ExecContext(ctx, query, roleID)
	return err
}

// createExpirationAuditEntry creates an audit entry for a role expiration.
func createExpirationAuditEntry(systemActorID string, role expiredRoleInfo) *models.RoleAuditEntry {
	return &models.RoleAuditEntry{
		ID:             uuid.New(),
		Timestamp:      time.Now(),
		ActorID:        systemActorID,
		ActorUsername:  "system",
		Action:         models.AuditActionExpire,
		TargetUserID:   role.UserID,
		TargetUsername: role.Username,
		OldRole:        role.Role,
		Reason:         "Role expired automatically",
	}
}

// IsUserAdmin checks if a user has the admin role.
// This is a convenience method for common authorization checks.
func (db *DB) IsUserAdmin(ctx context.Context, userID string) (bool, error) {
	role, err := db.GetEffectiveRole(ctx, userID)
	if err != nil {
		return false, err
	}
	return role == models.RoleAdmin, nil
}
