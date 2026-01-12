// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
service.go - Authorization Service Layer

This file provides a high-level authorization service that integrates
the Casbin enforcer with the database RBAC layer.

Key Features:
  - CanAccess: Check if subject can perform action on resource
  - CanAccessOwnData: Check if user can access another user's data
  - IsAdmin: Check if user has admin role
  - GetEffectiveRole: Get user's effective role from database
  - AssignRole: Assign role to user (admin-only)
  - RevokeRole: Revoke role from user (admin-only)

Role Hierarchy (enforced by Casbin):
  - viewer: Default, read-only access to own data
  - editor: Write access, inherits viewer
  - admin: Full access, inherits editor

Thread Safety:
  - All role operations go through database layer (mutex protected)
  - Cache invalidation handled automatically

ADR-0015: Zero Trust Authentication & Authorization
*/

package authz

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// Service errors
var (
	// ErrNotAuthorized is returned when an action is denied.
	ErrNotAuthorized = errors.New("not authorized")

	// ErrAdminRequired is returned when an admin-only action is attempted by non-admin.
	ErrAdminRequired = errors.New("admin role required")

	// ErrSelfRoleChange is returned when a user tries to change their own role.
	ErrSelfRoleChange = errors.New("cannot modify own role")

	// ErrNilSubject is returned when AuthSubject is nil.
	ErrNilSubject = errors.New("auth subject is nil")

	// ErrInvalidRole is returned when an invalid role is specified.
	ErrInvalidRole = errors.New("invalid role")
)

// RoleProvider defines the interface for role lookup operations.
// This abstraction allows the service to be tested without a real database.
type RoleProvider interface {
	GetUserRole(ctx context.Context, userID string) (*models.UserRole, error)
	GetEffectiveRole(ctx context.Context, userID string) (string, error)
	SetUserRole(ctx context.Context, role *models.UserRole, actorID, actorUsername, reason string) (*models.UserRole, error)
	DeleteUserRole(ctx context.Context, userID, actorID, actorUsername, reason string) error
	AuditRoleChange(ctx context.Context, entry *models.RoleAuditEntry) error
	IsUserAdmin(ctx context.Context, userID string) (bool, error)
}

// ServiceConfig holds configuration for the authorization service.
type ServiceConfig struct {
	// DefaultRole is assigned to users without explicit roles.
	DefaultRole string

	// CacheEnabled enables role caching for performance.
	CacheEnabled bool

	// CacheTTL is how long to cache role lookups.
	CacheTTL time.Duration

	// AuditEnabled enables audit logging for authorization decisions.
	AuditEnabled bool
}

// DefaultServiceConfig returns default configuration.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		DefaultRole:  models.RoleViewer,
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
		AuditEnabled: true,
	}
}

// Service provides authorization services integrating Casbin with database roles.
type Service struct {
	enforcer    *Enforcer
	db          RoleProvider
	config      *ServiceConfig
	roleCache   map[string]*roleCacheEntry
	roleCacheMu sync.RWMutex
	stopChan    chan struct{}
	stopOnce    sync.Once
	auditLogger *AuditLogger
}

// roleCacheEntry holds a cached role lookup.
type roleCacheEntry struct {
	role      string
	expiresAt time.Time
}

// NewService creates a new authorization service.
//
// Parameters:
//   - enforcer: The Casbin enforcer for policy evaluation
//   - db: Database connection for role persistence
//   - config: Service configuration (nil uses defaults)
//
// Returns:
//   - The initialized service
//   - Error if initialization fails
func NewService(enforcer *Enforcer, db RoleProvider, config *ServiceConfig) (*Service, error) {
	if enforcer == nil {
		return nil, errors.New("enforcer is required")
	}
	if db == nil {
		return nil, errors.New("database is required")
	}
	if config == nil {
		config = DefaultServiceConfig()
	}

	// Create audit logger if audit is enabled
	var auditLogger *AuditLogger
	if config.AuditEnabled {
		auditLogger = NewAuditLogger(&AuditLoggerConfig{
			Enabled:       true,
			LogAllowed:    true,
			LogDenied:     true,
			SampleRate:    1.0,
			BufferSize:    1000,
			FlushInterval: 5 * time.Second,
		})
	}

	s := &Service{
		enforcer:    enforcer,
		db:          db,
		config:      config,
		roleCache:   make(map[string]*roleCacheEntry),
		stopChan:    make(chan struct{}),
		auditLogger: auditLogger,
	}

	// Start cache cleanup if caching enabled
	if config.CacheEnabled && config.CacheTTL > 0 {
		go s.cacheCleanup()
	}

	return s, nil
}

// Close stops the service and cleans up resources.
func (s *Service) Close() {
	s.stopOnce.Do(func() {
		close(s.stopChan)
		if s.auditLogger != nil {
			s.auditLogger.Close()
		}
	})
}

// cacheCleanup periodically removes expired cache entries.
func (s *Service) cacheCleanup() {
	ticker := time.NewTicker(s.config.CacheTTL)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.roleCacheMu.Lock()
			now := time.Now()
			for key, entry := range s.roleCache {
				if now.After(entry.expiresAt) {
					delete(s.roleCache, key)
				}
			}
			s.roleCacheMu.Unlock()
		}
	}
}

// getCachedRole retrieves a role from cache.
func (s *Service) getCachedRole(userID string) (string, bool) {
	if !s.config.CacheEnabled {
		return "", false
	}

	s.roleCacheMu.RLock()
	defer s.roleCacheMu.RUnlock()

	entry, ok := s.roleCache[userID]
	if !ok {
		return "", false
	}

	if time.Now().After(entry.expiresAt) {
		return "", false
	}

	return entry.role, true
}

// setCachedRole stores a role in cache.
func (s *Service) setCachedRole(userID, role string) {
	if !s.config.CacheEnabled {
		return
	}

	s.roleCacheMu.Lock()
	defer s.roleCacheMu.Unlock()

	s.roleCache[userID] = &roleCacheEntry{
		role:      role,
		expiresAt: time.Now().Add(s.config.CacheTTL),
	}
}

// invalidateRoleCache removes a user's cached role.
func (s *Service) invalidateRoleCache(userID string) {
	s.roleCacheMu.Lock()
	defer s.roleCacheMu.Unlock()
	delete(s.roleCache, userID)
}

// GetEffectiveRole returns the effective role for a user.
// Checks cache first, then database, defaults to viewer if not found.
//
// Parameters:
//   - ctx: Context for cancellation
//   - userID: The user's unique identifier
//
// Returns:
//   - The user's effective role (viewer, editor, or admin)
//   - Error if database lookup fails (not for "not found")
func (s *Service) GetEffectiveRole(ctx context.Context, userID string) (string, error) {
	if userID == "" {
		return s.config.DefaultRole, nil
	}

	// Check cache first
	if role, ok := s.getCachedRole(userID); ok {
		return role, nil
	}

	// Query database
	role, err := s.db.GetEffectiveRole(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get effective role: %w", err)
	}

	// Cache the result
	s.setCachedRole(userID, role)

	return role, nil
}

// getAllRolesForSubject combines token roles with database role for a subject.
// Returns all roles and the effective (highest privilege) role.
func (s *Service) getAllRolesForSubject(ctx context.Context, subject *auth.AuthSubject) (allRoles []string, effectiveRole string, err error) {
	// Get database role for the user
	dbRole, err := s.GetEffectiveRole(ctx, subject.ID)
	if err != nil {
		RecordAuthzError("role_lookup_error")
		return nil, "", err
	}

	// Combine token roles with database role
	allRoles = make([]string, 0, len(subject.Roles)+1)
	allRoles = append(allRoles, subject.Roles...)

	// Add database role if not already present
	hasDBRole := false
	for _, r := range subject.Roles {
		if r == dbRole {
			hasDBRole = true
			break
		}
	}
	if !hasDBRole && dbRole != "" {
		allRoles = append(allRoles, dbRole)
	}

	// Determine effective role for metrics (highest privilege role)
	effectiveRole = s.getHighestRole(allRoles)

	return allRoles, effectiveRole, nil
}

// actorInfo holds actor information for audit logging.
type actorInfo struct {
	id       string
	username string
}

// getActorInfoForAudit extracts actor information from subject for audit logging.
func getActorInfoForAudit(subject *auth.AuthSubject) actorInfo {
	if subject != nil {
		return actorInfo{
			id:       subject.ID,
			username: subject.Username,
		}
	}
	return actorInfo{
		id:       "anonymous",
		username: "anonymous",
	}
}

// getReason returns the reason string for denied access.
func getReason(allowed bool) string {
	if !allowed {
		return "Insufficient permissions"
	}
	return ""
}

// CanAccess checks if the subject can perform the action on the resource.
// Uses both the subject's token roles and database-assigned roles.
//
// Parameters:
//   - ctx: Context for cancellation
//   - subject: The authenticated user (can be nil for anonymous)
//   - resource: The resource path (e.g., "/api/v1/playbacks")
//   - action: The action (e.g., "GET", "POST", "read", "write")
//
// Returns:
//   - true if access is allowed
//   - false with nil error if access is denied
//   - false with error if evaluation failed
func (s *Service) CanAccess(ctx context.Context, subject *auth.AuthSubject, resource, action string) (bool, error) {
	start := time.Now()

	// Handle anonymous vs authenticated access
	allowed, effectiveRole, allRoles, cacheHit, err := s.evaluateAccess(ctx, subject, resource, action)
	if err != nil {
		RecordAuthzError("enforcer_error")
		return false, err
	}

	duration := time.Since(start)

	// Record metrics
	RecordAuthzDecision(effectiveRole, resource, action, allowed, duration, cacheHit)

	// Log audit event
	s.logAuditDecision(subject, effectiveRole, allRoles, resource, action, allowed, duration, cacheHit)

	return allowed, nil
}

// evaluateAccess performs the actual access evaluation for anonymous or authenticated subjects.
func (s *Service) evaluateAccess(ctx context.Context, subject *auth.AuthSubject, resource, action string) (allowed bool, effectiveRole string, allRoles []string, cacheHit bool, err error) {
	// Handle anonymous access
	if subject == nil {
		return s.evaluateAnonymousAccess(resource, action)
	}

	// Handle authenticated access
	return s.evaluateAuthenticatedAccess(ctx, subject, resource, action)
}

// evaluateAnonymousAccess handles access evaluation for anonymous users.
func (s *Service) evaluateAnonymousAccess(resource, action string) (allowed bool, effectiveRole string, allRoles []string, cacheHit bool, err error) {
	effectiveRole = s.config.DefaultRole
	allRoles = []string{effectiveRole}
	allowed, err = s.enforcer.Enforce(s.config.DefaultRole, resource, action)
	return allowed, effectiveRole, allRoles, false, err
}

// evaluateAuthenticatedAccess handles access evaluation for authenticated users.
func (s *Service) evaluateAuthenticatedAccess(ctx context.Context, subject *auth.AuthSubject, resource, action string) (allowed bool, effectiveRole string, allRoles []string, cacheHit bool, err error) {
	// Get all roles for the subject
	allRoles, effectiveRole, err = s.getAllRolesForSubject(ctx, subject)
	if err != nil {
		return false, "", nil, false, err
	}

	// Check with enforcer using all roles
	allowed, err = s.enforcer.EnforceWithRoles(subject.ID, allRoles, resource, action)
	if err != nil {
		return false, effectiveRole, allRoles, false, err
	}

	// Check if this was a cache hit (enforcer tracks this internally)
	_, cacheHit = s.enforcer.cache.get(subject.ID, resource, action)

	return allowed, effectiveRole, allRoles, cacheHit, nil
}

// logAuditDecision logs an audit event for an access decision.
func (s *Service) logAuditDecision(subject *auth.AuthSubject, effectiveRole string, allRoles []string, resource, action string, allowed bool, duration time.Duration, cacheHit bool) {
	if s.auditLogger == nil {
		return
	}

	actor := getActorInfoForAudit(subject)

	s.auditLogger.LogDecision(&AuditEvent{
		ActorID:       actor.id,
		ActorUsername: actor.username,
		ActorRole:     effectiveRole,
		ActorRoles:    allRoles,
		Resource:      resource,
		Action:        action,
		Decision:      allowed,
		Reason:        getReason(allowed),
		Duration:      duration,
		CacheHit:      cacheHit,
	})
}

// getHighestRole returns the highest privilege role from a list.
func (s *Service) getHighestRole(roles []string) string {
	// Priority: admin > editor > viewer
	// Check highest privilege first and return early
	for _, role := range roles {
		if role == models.RoleAdmin {
			return models.RoleAdmin
		}
	}

	for _, role := range roles {
		if role == models.RoleEditor {
			return models.RoleEditor
		}
	}

	for _, role := range roles {
		if role == models.RoleViewer {
			return models.RoleViewer
		}
	}

	return s.config.DefaultRole
}

// CanAccessOwnData checks if a user can access another user's data.
// Regular users can only access their own data; admins can access all.
//
// Parameters:
//   - ctx: Context for cancellation
//   - subject: The authenticated user making the request
//   - targetUserID: The user whose data is being accessed
//
// Returns:
//   - true if the subject can access the target's data
//   - false if access is denied
//   - error if subject is nil or role lookup fails
func (s *Service) CanAccessOwnData(ctx context.Context, subject *auth.AuthSubject, targetUserID string) (bool, error) {
	if subject == nil {
		return false, ErrNilSubject
	}

	// User can always access their own data
	if subject.ID == targetUserID {
		return true, nil
	}

	// Check if user is admin (can access all data)
	isAdmin, err := s.IsAdmin(ctx, subject)
	if err != nil {
		return false, err
	}

	return isAdmin, nil
}

// IsAdmin checks if the subject has admin role.
// Checks both token roles and database-assigned roles.
//
// Parameters:
//   - ctx: Context for cancellation
//   - subject: The authenticated user
//
// Returns:
//   - true if user is an admin
//   - false if not an admin or on error
func (s *Service) IsAdmin(ctx context.Context, subject *auth.AuthSubject) (bool, error) {
	if subject == nil {
		return false, nil
	}

	// Check token roles first (faster)
	for _, role := range subject.Roles {
		if role == models.RoleAdmin {
			return true, nil
		}
	}

	// Check database role
	dbRole, err := s.GetEffectiveRole(ctx, subject.ID)
	if err != nil {
		return false, err
	}

	return dbRole == models.RoleAdmin, nil
}

// IsEditor checks if the subject has editor or higher role.
// Editors inherit viewer permissions and can modify data.
//
// Parameters:
//   - ctx: Context for cancellation
//   - subject: The authenticated user
//
// Returns:
//   - true if user is an editor or admin
//   - false otherwise
func (s *Service) IsEditor(ctx context.Context, subject *auth.AuthSubject) (bool, error) {
	if subject == nil {
		return false, nil
	}

	// Check token roles first
	for _, role := range subject.Roles {
		if role == models.RoleEditor || role == models.RoleAdmin {
			return true, nil
		}
	}

	// Check database role
	dbRole, err := s.GetEffectiveRole(ctx, subject.ID)
	if err != nil {
		return false, err
	}

	return dbRole == models.RoleEditor || dbRole == models.RoleAdmin, nil
}

// validateRoleChange performs common validation for role assignment/revocation operations.
// Returns error if validation fails.
func (s *Service) validateRoleChange(ctx context.Context, actor *auth.AuthSubject, targetUserID string) error {
	if actor == nil {
		return ErrNilSubject
	}

	// Prevent self role change
	if actor.ID == targetUserID {
		return ErrSelfRoleChange
	}

	// Verify actor is admin
	isAdmin, err := s.IsAdmin(ctx, actor)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}
	if !isAdmin {
		return ErrAdminRequired
	}

	return nil
}

// invalidateCachesForUser invalidates both role cache and enforcer cache for a user.
func (s *Service) invalidateCachesForUser(userID string) {
	s.invalidateRoleCache(userID)
	s.enforcer.cache.invalidateUser(userID)
	RecordAuthzCacheInvalidation("role_change")
}

// AssignRole assigns a role to a user.
// Only admins can assign roles. Users cannot change their own role.
//
// Parameters:
//   - ctx: Context for cancellation
//   - actor: The admin performing the assignment
//   - targetUserID: The user receiving the role
//   - targetUsername: The username for display/audit
//   - role: The role to assign (viewer, editor, admin)
//   - reason: Optional reason for the assignment
//
// Returns:
//   - Error if actor is not admin, role is invalid, or database operation fails
func (s *Service) AssignRole(ctx context.Context, actor *auth.AuthSubject, targetUserID, targetUsername, role, reason string) error {
	// Validate role
	if !models.IsValidRole(role) {
		return ErrInvalidRole
	}

	// Validate actor and permissions
	if err := s.validateRoleChange(ctx, actor, targetUserID); err != nil {
		return err
	}

	// Create role assignment
	userRole := models.NewUserRole(targetUserID, targetUsername, role, actor.ID)

	// Save to database (this also creates audit entry)
	_, err := s.db.SetUserRole(ctx, userRole, actor.ID, actor.Username, reason)
	if err != nil {
		return fmt.Errorf("failed to set user role: %w", err)
	}

	// Invalidate caches
	s.invalidateCachesForUser(targetUserID)

	// Record metrics for role assignment
	RecordRoleAssignment(role, "assign")

	logging.Info().
		Str("actor_id", actor.ID).
		Str("actor_username", actor.Username).
		Str("target_user_id", targetUserID).
		Str("target_username", targetUsername).
		Str("role", role).
		Str("reason", reason).
		Msg("Role assigned")

	return nil
}

// RevokeRole removes a user's role assignment.
// Only admins can revoke roles. Users cannot revoke their own role.
//
// Parameters:
//   - ctx: Context for cancellation
//   - actor: The admin performing the revocation
//   - targetUserID: The user whose role is being revoked
//   - reason: Optional reason for the revocation
//
// Returns:
//   - Error if actor is not admin or database operation fails
func (s *Service) RevokeRole(ctx context.Context, actor *auth.AuthSubject, targetUserID, reason string) error {
	// Validate actor and permissions
	if err := s.validateRoleChange(ctx, actor, targetUserID); err != nil {
		return err
	}

	// Get old role for metrics before deletion (ignore error, we just want the role for metrics)
	oldRole, _ := s.GetEffectiveRole(ctx, targetUserID) //nolint:errcheck // intentionally ignored for metrics

	// Delete role from database
	err := s.db.DeleteUserRole(ctx, targetUserID, actor.ID, actor.Username, reason)
	if err != nil {
		// If role not found, that's not an error for revocation
		if errors.Is(err, database.ErrRoleNotFound) {
			return nil
		}
		return fmt.Errorf("failed to delete user role: %w", err)
	}

	// Invalidate caches
	s.invalidateCachesForUser(targetUserID)

	// Record metrics for role revocation
	if oldRole != "" {
		RecordRoleAssignment(oldRole, "revoke")
	}

	logging.Info().
		Str("actor_id", actor.ID).
		Str("actor_username", actor.Username).
		Str("target_user_id", targetUserID).
		Str("reason", reason).
		Msg("Role revoked")

	return nil
}

// UpdateRole updates a user's role.
// This is a convenience method that wraps AssignRole.
//
// Parameters:
//   - ctx: Context for cancellation
//   - actor: The admin performing the update
//   - targetUserID: The user whose role is being updated
//   - targetUsername: The username for display/audit
//   - newRole: The new role to assign
//   - reason: Optional reason for the update
//
// Returns:
//   - Error if actor is not admin, role is invalid, or database operation fails
func (s *Service) UpdateRole(ctx context.Context, actor *auth.AuthSubject, targetUserID, targetUsername, newRole, reason string) error {
	return s.AssignRole(ctx, actor, targetUserID, targetUsername, newRole, reason)
}

// RequireAdmin returns an error if the subject is not an admin.
// Useful for handler-level authorization checks.
//
// Parameters:
//   - ctx: Context for cancellation
//   - subject: The authenticated user
//
// Returns:
//   - nil if user is admin
//   - ErrNilSubject if subject is nil
//   - ErrAdminRequired if user is not admin
func (s *Service) RequireAdmin(ctx context.Context, subject *auth.AuthSubject) error {
	if subject == nil {
		return ErrNilSubject
	}

	isAdmin, err := s.IsAdmin(ctx, subject)
	if err != nil {
		return err
	}
	if !isAdmin {
		return ErrAdminRequired
	}

	return nil
}

// RequireEditor returns an error if the subject is not an editor or admin.
// Useful for handler-level authorization checks.
//
// Parameters:
//   - ctx: Context for cancellation
//   - subject: The authenticated user
//
// Returns:
//   - nil if user is editor or admin
//   - ErrNilSubject if subject is nil
//   - ErrNotAuthorized if user is not editor or admin
func (s *Service) RequireEditor(ctx context.Context, subject *auth.AuthSubject) error {
	if subject == nil {
		return ErrNilSubject
	}

	isEditor, err := s.IsEditor(ctx, subject)
	if err != nil {
		return err
	}
	if !isEditor {
		return ErrNotAuthorized
	}

	return nil
}

// RequireAccessToUser returns an error if the subject cannot access the target user's data.
// Users can access their own data; admins can access all data.
//
// Parameters:
//   - ctx: Context for cancellation
//   - subject: The authenticated user
//   - targetUserID: The user whose data is being accessed
//
// Returns:
//   - nil if access is allowed
//   - ErrNilSubject if subject is nil
//   - ErrNotAuthorized if access is denied
func (s *Service) RequireAccessToUser(ctx context.Context, subject *auth.AuthSubject, targetUserID string) error {
	canAccess, err := s.CanAccessOwnData(ctx, subject, targetUserID)
	if err != nil {
		return err
	}
	if !canAccess {
		return ErrNotAuthorized
	}

	return nil
}

// GetUserRoleInfo retrieves detailed role information for a user.
// Returns nil if user has no explicit role assignment (uses default).
//
// Parameters:
//   - ctx: Context for cancellation
//   - subject: The admin requesting the info (for authorization)
//   - targetUserID: The user whose role info is requested
//
// Returns:
//   - UserRole if found
//   - nil if user has default role only
//   - Error if not authorized or database error
func (s *Service) GetUserRoleInfo(ctx context.Context, subject *auth.AuthSubject, targetUserID string) (*models.UserRole, error) {
	if subject == nil {
		return nil, ErrNilSubject
	}

	// Allow users to see their own role
	if subject.ID == targetUserID {
		return s.getUserRoleFromDB(ctx, targetUserID)
	}

	// Non-owners need admin access
	if err := s.RequireAdmin(ctx, subject); err != nil {
		return nil, err
	}

	return s.getUserRoleFromDB(ctx, targetUserID)
}

// getUserRoleFromDB retrieves a user role from the database.
// Returns nil if user has no explicit role assignment (no error).
func (s *Service) getUserRoleFromDB(ctx context.Context, targetUserID string) (*models.UserRole, error) {
	role, err := s.db.GetUserRole(ctx, targetUserID)
	if err != nil {
		if errors.Is(err, database.ErrRoleNotFound) {
			return nil, nil // No explicit role, uses default
		}
		return nil, err
	}

	return role, nil
}

// GetEnforcer returns the underlying Casbin enforcer for advanced use cases.
// This should be used sparingly; prefer the service methods for most operations.
func (s *Service) GetEnforcer() *Enforcer {
	return s.enforcer
}
