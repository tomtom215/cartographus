// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
handler_context.go - Request Context Helpers for Authorization

This file provides helpers for extracting and using authentication context
in API handlers. It integrates with the authz service layer (Phase 2) to
provide easy-to-use authorization checks.

Key Features:
  - HandlerContext: Extracts authenticated user from request
  - User-scoped access checks: CanAccessUser, RequireAccessToUser
  - Admin checks: RequireAdmin, RequireEditor

Usage:
    func (h *Handler) SomeHandler(w http.ResponseWriter, r *http.Request) {
        hctx := GetHandlerContext(r)
        if hctx == nil {
            respondError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
            return
        }

        // Check if user can access target user's data
        if !hctx.CanAccessUser(targetUserID) {
            respondError(w, http.StatusForbidden, "FORBIDDEN", "Access denied", nil)
            return
        }
        // ... proceed with handler logic
    }

ADR-0015: Zero Trust Authentication & Authorization
RBAC Phase 3: API Handler Authorization
*/

package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/authz"
	"github.com/tomtom215/cartographus/internal/models"
)

// HandlerContext provides request-scoped authorization context for handlers.
// It encapsulates the authenticated user's identity and role information,
// providing convenient methods for authorization checks.
type HandlerContext struct {
	// Subject is the authenticated user from the request context.
	// May be nil for unauthenticated requests.
	Subject *auth.AuthSubject

	// UserID is the unique identifier for the authenticated user.
	// Empty string for unauthenticated requests.
	UserID string

	// Username is the display name for the authenticated user.
	// Empty string for unauthenticated requests.
	Username string

	// IsAdmin indicates whether the user has admin role.
	// Checked against both token roles and database-assigned roles.
	IsAdmin bool

	// IsEditor indicates whether the user has editor role or higher.
	// Editors inherit viewer permissions and can modify data.
	IsEditor bool

	// EffectiveRole is the user's effective role (viewer, editor, admin).
	// Combines token roles with database-assigned roles.
	EffectiveRole string

	// RequestID is the unique identifier for this request.
	// Useful for logging and debugging.
	RequestID string

	// authzService holds reference to the authorization service.
	// Used for dynamic authorization checks.
	authzService *authz.Service

	// ctx holds the request context for database operations.
	ctx context.Context
}

// GetHandlerContext extracts the authentication context from an HTTP request.
// Returns nil if no authentication is present (for unauthenticated endpoints).
//
// This function:
//  1. Retrieves the AuthSubject from the request context
//  2. Extracts user identity (ID, username)
//  3. Determines admin/editor status from token roles
//  4. Sets the request ID for logging
//
// For database role lookups (more accurate), use GetHandlerContextWithAuthz.
func GetHandlerContext(r *http.Request) *HandlerContext {
	subject := auth.GetAuthSubject(r.Context())

	hctx := &HandlerContext{
		Subject:   subject,
		RequestID: r.Header.Get("X-Request-ID"),
		ctx:       r.Context(),
	}

	if subject != nil {
		hctx.UserID = subject.ID
		hctx.Username = subject.Username

		// Check roles from token
		hctx.IsAdmin = subject.HasRole(models.RoleAdmin)
		hctx.IsEditor = subject.HasRole(models.RoleEditor) || subject.HasRole(models.RoleAdmin)

		// Set effective role based on token
		if hctx.IsAdmin {
			hctx.EffectiveRole = models.RoleAdmin
		} else if hctx.IsEditor {
			hctx.EffectiveRole = models.RoleEditor
		} else {
			hctx.EffectiveRole = models.RoleViewer
		}
	}

	return hctx
}

// GetHandlerContextWithAuthz extracts authentication context and performs
// database role lookup for accurate authorization checks.
//
// This function extends GetHandlerContext by:
//  1. Querying the database for the user's assigned role
//  2. Combining token roles with database roles
//  3. Using the authz service for consistent role evaluation
//
// Use this when you need accurate role information that includes
// database-assigned roles (not just JWT/token roles).
func GetHandlerContextWithAuthz(r *http.Request, authzSvc *authz.Service) *HandlerContext {
	hctx := GetHandlerContext(r)

	if authzSvc == nil || hctx.Subject == nil {
		return hctx
	}

	hctx.authzService = authzSvc

	// Query database for effective role (combines token + database roles)
	effectiveRole, err := authzSvc.GetEffectiveRole(r.Context(), hctx.UserID)
	if err == nil && effectiveRole != "" {
		hctx.EffectiveRole = effectiveRole

		// Update admin/editor flags based on effective role
		hctx.IsAdmin = effectiveRole == models.RoleAdmin
		hctx.IsEditor = effectiveRole == models.RoleEditor || effectiveRole == models.RoleAdmin
	}

	return hctx
}

// IsAuthenticated returns true if the request has valid authentication.
func (hctx *HandlerContext) IsAuthenticated() bool {
	return hctx != nil && hctx.Subject != nil
}

// HasRole checks if the user has a specific role.
// Role hierarchy: admin > editor > viewer
// Admins have all roles, editors have editor and viewer roles.
//
// Parameters:
//   - role: The role to check (viewer, editor, admin)
//
// Returns:
//   - true if the user has the specified role (or higher)
//   - false if not authenticated or role not granted
func (hctx *HandlerContext) HasRole(role string) bool {
	if hctx == nil || hctx.Subject == nil {
		return false
	}

	switch role {
	case models.RoleAdmin:
		return hctx.IsAdmin
	case models.RoleEditor:
		return hctx.IsEditor || hctx.IsAdmin
	case models.RoleViewer:
		// All authenticated users are at least viewers
		return true
	default:
		return false
	}
}

// CanAccessUser checks if the current user can access another user's data.
// Users can access their own data; admins can access all users' data.
//
// Parameters:
//   - targetUserID: The ID of the user whose data is being accessed
//
// Returns:
//   - true if access is allowed
//   - false if access is denied or user is not authenticated
func (hctx *HandlerContext) CanAccessUser(targetUserID string) bool {
	if hctx == nil || hctx.Subject == nil {
		return false
	}

	// Users can always access their own data
	if hctx.UserID == targetUserID {
		return true
	}

	// Admins can access all data
	return hctx.IsAdmin
}

// CanAccessUserWithAuthz performs an authorization check using the authz service.
// This is more accurate than CanAccessUser as it includes database role lookups.
//
// Parameters:
//   - targetUserID: The ID of the user whose data is being accessed
//
// Returns:
//   - true if access is allowed
//   - false if access is denied
func (hctx *HandlerContext) CanAccessUserWithAuthz(targetUserID string) bool {
	if hctx == nil || hctx.Subject == nil {
		return false
	}

	// If no authz service, fall back to simple check
	if hctx.authzService == nil {
		return hctx.CanAccessUser(targetUserID)
	}

	// Use authz service for comprehensive check
	canAccess, err := hctx.authzService.CanAccessOwnData(hctx.ctx, hctx.Subject, targetUserID)
	if err != nil {
		return false
	}

	return canAccess
}

// RequireAdmin returns an error if the user is not an admin.
// Useful for handler-level authorization in admin-only endpoints.
//
// Returns:
//   - nil if user is admin
//   - ErrNotAuthenticated if not authenticated
//   - ErrNotAuthorized if not admin
func (hctx *HandlerContext) RequireAdmin() error {
	if hctx == nil || hctx.Subject == nil {
		return ErrNotAuthenticated
	}

	// Check with authz service if available
	if hctx.authzService != nil {
		return hctx.authzService.RequireAdmin(hctx.ctx, hctx.Subject)
	}

	// Fall back to local check
	if !hctx.IsAdmin {
		return ErrNotAuthorized
	}

	return nil
}

// RequireEditor returns an error if the user is not an editor or admin.
// Useful for handler-level authorization in editor endpoints.
//
// Returns:
//   - nil if user is editor or admin
//   - ErrNotAuthenticated if not authenticated
//   - ErrNotAuthorized if not editor or admin
func (hctx *HandlerContext) RequireEditor() error {
	if hctx == nil || hctx.Subject == nil {
		return ErrNotAuthenticated
	}

	// Check with authz service if available
	if hctx.authzService != nil {
		return hctx.authzService.RequireEditor(hctx.ctx, hctx.Subject)
	}

	// Fall back to local check
	if !hctx.IsEditor {
		return ErrNotAuthorized
	}

	return nil
}

// RequireAccessToUser returns an error if the user cannot access the target user's data.
// Users can access their own data; admins can access all users' data.
//
// Parameters:
//   - targetUserID: The ID of the user whose data is being accessed
//
// Returns:
//   - nil if access is allowed
//   - ErrNotAuthenticated if not authenticated
//   - ErrNotAuthorized if access is denied
func (hctx *HandlerContext) RequireAccessToUser(targetUserID string) error {
	if hctx == nil || hctx.Subject == nil {
		return ErrNotAuthenticated
	}

	// Check with authz service if available
	if hctx.authzService != nil {
		return hctx.authzService.RequireAccessToUser(hctx.ctx, hctx.Subject, targetUserID)
	}

	// Fall back to local check
	if !hctx.CanAccessUser(targetUserID) {
		return ErrNotAuthorized
	}

	return nil
}

// FilterByUser returns a SQL WHERE clause condition for user-scoped queries.
// For regular users, returns "user_id = '<userID>'".
// For admins, returns "1=1" (no filter, all data visible).
//
// Parameters:
//   - userIDColumn: The column name to filter on (e.g., "user_id", "player_user_id")
//
// Returns:
//   - SQL WHERE condition string
//
// Example:
//
//	whereClause := hctx.FilterByUser("user_id")
//	query := fmt.Sprintf("SELECT * FROM playbacks WHERE %s", whereClause)
func (hctx *HandlerContext) FilterByUser(userIDColumn string) string {
	if hctx == nil || hctx.Subject == nil {
		// No authentication - return impossible condition
		return "1=0"
	}

	// Admins see all data
	if hctx.IsAdmin {
		return "1=1"
	}

	// Regular users see only their own data
	return userIDColumn + " = '" + hctx.UserID + "'"
}

// GetUserIDForQuery returns the user ID to use in database queries.
// For regular users, returns their own user ID.
// For admins, returns empty string (indicating all users).
//
// This is useful when building queries that should show:
//   - All data for admins
//   - Only user's own data for regular users
func (hctx *HandlerContext) GetUserIDForQuery() string {
	if hctx == nil || hctx.Subject == nil {
		return ""
	}

	// Admins see all data (empty = no filter)
	if hctx.IsAdmin {
		return ""
	}

	// Regular users see only their own data
	return hctx.UserID
}

// Handler authorization errors
var (
	// ErrNotAuthenticated is returned when authentication is required but not present.
	ErrNotAuthenticated = &AuthError{
		Code:       "AUTH_REQUIRED",
		Message:    "Authentication required",
		StatusCode: 401,
	}

	// ErrNotAuthorized is returned when the user lacks permission for the action.
	ErrNotAuthorized = &AuthError{
		Code:       "FORBIDDEN",
		Message:    "Access denied: insufficient permissions",
		StatusCode: 403,
	}
)

// AuthError represents a structured error for authorization failures.
// This is separate from APIError (in response.go) to avoid conflicts.
type AuthError struct {
	Code       string
	Message    string
	StatusCode int
}

func (e *AuthError) Error() string {
	return e.Message
}

// RespondAuthError writes an authorization error response.
// Use this helper to consistently respond to auth failures.
func RespondAuthError(w http.ResponseWriter, err error) {
	var authErr *AuthError
	if errors.As(err, &authErr) {
		respondError(w, authErr.StatusCode, authErr.Code, authErr.Message, nil)
		return
	}

	// Handle authz service errors using errors.Is for wrapped error support
	switch {
	case errors.Is(err, authz.ErrNilSubject):
		respondError(w, 401, "AUTH_REQUIRED", "Authentication required", nil)
	case errors.Is(err, authz.ErrAdminRequired):
		respondError(w, 403, "ADMIN_REQUIRED", "Admin role required", nil)
	case errors.Is(err, authz.ErrNotAuthorized):
		respondError(w, 403, "FORBIDDEN", "Access denied: insufficient permissions", nil)
	default:
		respondError(w, 403, "FORBIDDEN", "Access denied", err)
	}
}
