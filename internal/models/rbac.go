// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
rbac.go - Role-Based Access Control Models

This file defines data structures for RBAC user role management and audit logging.

Key Structures:
  - UserRole: Persistent role assignment for a user
  - RoleAuditEntry: Audit log entry for role changes
  - RoleConstants: Standard role names (viewer, editor, admin)

Role Hierarchy:
  - viewer: Default role, read-only access to own data
  - editor: Can write/modify data (inherits viewer)
  - admin: Full access including user management (inherits editor)

Usage:
  - Database operations in internal/database/rbac.go
  - Authorization service in internal/authz/service.go
  - API authorization in internal/api/handlers_*.go
*/

package models

import (
	"time"

	"github.com/google/uuid"
)

// Role constants define the standard roles in the system.
// These align with the Casbin policy definitions in internal/authz/policy.csv.
const (
	// RoleViewer is the default role with read-only access to own data.
	RoleViewer = "viewer"

	// RoleEditor can write/modify data and inherits viewer permissions.
	RoleEditor = "editor"

	// RoleAdmin has full access including user management and inherits editor permissions.
	RoleAdmin = "admin"
)

// ValidRoles contains all valid role names for validation.
var ValidRoles = []string{RoleViewer, RoleEditor, RoleAdmin}

// IsValidRole checks if a role name is valid.
func IsValidRole(role string) bool {
	for _, r := range ValidRoles {
		if r == role {
			return true
		}
	}
	return false
}

// UserRole represents a user's role assignment in the system.
// Roles are persistent and stored in the database for lookup during authorization.
//
// Key Features:
//   - Unique constraint on (user_id, role) allows multiple roles per user if needed
//   - ExpiresAt supports time-limited role assignments
//   - IsActive allows soft-disable without deletion
//   - Metadata stores optional JSON data (e.g., assignment context)
type UserRole struct {
	// ID is the primary key (auto-generated, not auto-increment in DuckDB)
	ID int64 `json:"id"`

	// UserID is the unique identifier for the user (from auth system)
	UserID string `json:"user_id"`

	// Username is the display name for the user
	Username string `json:"username"`

	// Role is the assigned role (viewer, editor, admin)
	Role string `json:"role"`

	// AssignedBy is the user ID who assigned this role (empty for system assignments)
	AssignedBy string `json:"assigned_by,omitempty"`

	// AssignedAt is when the role was assigned
	AssignedAt time.Time `json:"assigned_at"`

	// ExpiresAt is when the role expires (nil means no expiration)
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// IsActive indicates if the role is currently active
	IsActive bool `json:"is_active"`

	// Metadata contains optional JSON data for the role assignment
	Metadata *string `json:"metadata,omitempty"`
}

// IsExpired checks if the role has expired.
func (ur *UserRole) IsExpired() bool {
	if ur.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*ur.ExpiresAt)
}

// IsEffective checks if the role is currently effective (active and not expired).
func (ur *UserRole) IsEffective() bool {
	return ur.IsActive && !ur.IsExpired()
}

// RoleAuditEntry records a role change event for audit purposes.
// All role assignments, revocations, and modifications are logged.
//
// Key Features:
//   - Immutable once created (append-only audit log)
//   - Captures actor, target, action, and context
//   - Supports both automated and manual role changes
type RoleAuditEntry struct {
	// ID is the primary key (UUID for global uniqueness)
	ID uuid.UUID `json:"id"`

	// Timestamp is when the action occurred
	Timestamp time.Time `json:"timestamp"`

	// ActorID is the user who performed the action (system for automated changes)
	ActorID string `json:"actor_id"`

	// ActorUsername is the display name of the actor
	ActorUsername string `json:"actor_username,omitempty"`

	// Action is the type of change (assign, revoke, update, expire)
	Action string `json:"action"`

	// TargetUserID is the user whose role was changed
	TargetUserID string `json:"target_user_id"`

	// TargetUsername is the display name of the target user
	TargetUsername string `json:"target_username,omitempty"`

	// OldRole is the previous role (empty for new assignments)
	OldRole string `json:"old_role,omitempty"`

	// NewRole is the new role (empty for revocations)
	NewRole string `json:"new_role,omitempty"`

	// Reason is an optional explanation for the change
	Reason string `json:"reason,omitempty"`

	// IPAddress is the client IP address (for web requests)
	IPAddress string `json:"ip_address,omitempty"`

	// UserAgent is the client user agent (for web requests)
	UserAgent string `json:"user_agent,omitempty"`

	// SessionID is the session identifier (for web requests)
	SessionID string `json:"session_id,omitempty"`
}

// AuditAction constants define the types of audit log entries.
const (
	// AuditActionAssign indicates a new role was assigned.
	AuditActionAssign = "assign"

	// AuditActionRevoke indicates a role was revoked.
	AuditActionRevoke = "revoke"

	// AuditActionUpdate indicates a role was updated (e.g., expiration changed).
	AuditActionUpdate = "update"

	// AuditActionExpire indicates a role expired automatically.
	AuditActionExpire = "expire"
)

// RoleStats provides statistics about role assignments.
type RoleStats struct {
	// TotalUsers is the number of users with any role
	TotalUsers int `json:"total_users"`

	// ByRole is the count of users per role
	ByRole map[string]int `json:"by_role"`

	// ActiveRoles is the count of currently effective roles
	ActiveRoles int `json:"active_roles"`

	// ExpiredRoles is the count of expired role assignments
	ExpiredRoles int `json:"expired_roles"`

	// InactiveRoles is the count of deactivated role assignments
	InactiveRoles int `json:"inactive_roles"`
}

// NewUserRole creates a new UserRole with default values.
func NewUserRole(userID, username, role, assignedBy string) *UserRole {
	return &UserRole{
		UserID:     userID,
		Username:   username,
		Role:       role,
		AssignedBy: assignedBy,
		AssignedAt: time.Now(),
		IsActive:   true,
	}
}

// NewRoleAuditEntry creates a new RoleAuditEntry with default values.
func NewRoleAuditEntry(actorID, actorUsername, action, targetUserID, targetUsername string) *RoleAuditEntry {
	return &RoleAuditEntry{
		ID:             uuid.New(),
		Timestamp:      time.Now(),
		ActorID:        actorID,
		ActorUsername:  actorUsername,
		Action:         action,
		TargetUserID:   targetUserID,
		TargetUsername: targetUsername,
	}
}
