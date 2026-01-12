// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package api provides HTTP handlers for the Cartographus API.
// ADR-0015: Zero Trust Authentication & Authorization
package api

import (
	"net/http"
	"strings"

	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/authz"
)

// ZeroTrustRouter handles Zero Trust authentication and authorization routes.
type ZeroTrustRouter struct {
	flowHandlers   *auth.FlowHandlers
	policyHandlers *authz.PolicyHandlers
	sessionMW      *auth.SessionMiddleware
	authMiddleware *auth.Middleware
}

// ZeroTrustRouterConfig holds configuration for the Zero Trust router.
type ZeroTrustRouterConfig struct {
	FlowHandlers   *auth.FlowHandlers
	PolicyHandlers *authz.PolicyHandlers
	SessionMW      *auth.SessionMiddleware
	AuthMiddleware *auth.Middleware
}

// NewZeroTrustRouter creates a new Zero Trust router.
func NewZeroTrustRouter(config *ZeroTrustRouterConfig) *ZeroTrustRouter {
	return &ZeroTrustRouter{
		flowHandlers:   config.FlowHandlers,
		policyHandlers: config.PolicyHandlers,
		sessionMW:      config.SessionMW,
		authMiddleware: config.AuthMiddleware,
	}
}

// RegisterRoutes registers Zero Trust auth routes on the given mux.
func (r *ZeroTrustRouter) RegisterRoutes(mux *http.ServeMux) {
	if r.flowHandlers == nil && r.policyHandlers == nil {
		return
	}

	// ========================
	// Authentication Routes
	// ========================

	if r.flowHandlers != nil {
		// OIDC Authentication
		// GET /api/auth/oidc/login - Initiates OIDC login flow
		mux.HandleFunc("GET /api/auth/oidc/login",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(r.flowHandlers.OIDCLogin),
			),
		)

		// GET /api/auth/oidc/callback - OIDC callback handler
		mux.HandleFunc("GET /api/auth/oidc/callback",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(r.flowHandlers.OIDCCallback),
			),
		)

		// POST /api/auth/oidc/refresh - Refresh OIDC tokens (Phase 4A.2)
		mux.HandleFunc("POST /api/auth/oidc/refresh",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(r.flowHandlers.OIDCRefresh),
			),
		)

		// POST /api/auth/oidc/logout - RP-initiated logout (Phase 4B.2)
		mux.HandleFunc("POST /api/auth/oidc/logout",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(
					r.sessionMW.Authenticate(http.HandlerFunc(r.flowHandlers.OIDCLogout)).ServeHTTP,
				),
			),
		)

		// POST /api/auth/oidc/backchannel-logout - Back-channel logout (Phase 4B.3)
		// Note: No rate limiting or session auth - IdP server-to-server call
		mux.HandleFunc("POST /api/auth/oidc/backchannel-logout",
			r.authMiddleware.CORS(r.flowHandlers.BackChannelLogout),
		)

		// Plex Authentication
		// GET /api/auth/plex/login - Initiates Plex PIN-based login
		mux.HandleFunc("GET /api/auth/plex/login",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(r.flowHandlers.PlexLogin),
			),
		)

		// GET /api/auth/plex/poll - Polls for Plex PIN authorization
		mux.HandleFunc("GET /api/auth/plex/poll",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(r.flowHandlers.PlexPoll),
			),
		)

		// POST /api/auth/plex/callback - Completes Plex authentication
		mux.HandleFunc("POST /api/auth/plex/callback",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(r.flowHandlers.PlexCallback),
			),
		)

		// User Info & Session Management
		// GET /api/auth/userinfo - Get authenticated user info
		mux.HandleFunc("GET /api/auth/userinfo",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(
					r.sessionMW.Authenticate(http.HandlerFunc(r.flowHandlers.UserInfo)).ServeHTTP,
				),
			),
		)

		// POST /api/auth/logout - Logout current session
		mux.HandleFunc("POST /api/auth/logout",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(
					r.sessionMW.Authenticate(http.HandlerFunc(r.flowHandlers.Logout)).ServeHTTP,
				),
			),
		)

		// POST /api/auth/logout/all - Logout all sessions
		mux.HandleFunc("POST /api/auth/logout/all",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(
					r.sessionMW.RequireAuth(http.HandlerFunc(r.flowHandlers.LogoutAll)).ServeHTTP,
				),
			),
		)

		// GET /api/auth/sessions - List all sessions for current user
		mux.HandleFunc("GET /api/auth/sessions",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(
					r.sessionMW.RequireAuth(http.HandlerFunc(r.flowHandlers.Sessions)).ServeHTTP,
				),
			),
		)

		// DELETE /api/auth/sessions/{id} - Revoke specific session
		mux.HandleFunc("DELETE /api/auth/sessions/",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(
					r.sessionMW.RequireAuth(http.HandlerFunc(r.handleRevokeSession)).ServeHTTP,
				),
			),
		)
	}

	// ========================
	// Authorization Routes
	// ========================

	if r.policyHandlers != nil {
		// GET /api/admin/roles - List all roles (public)
		mux.HandleFunc("GET /api/admin/roles",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(r.policyHandlers.ListRoles),
			),
		)

		// GET /api/admin/roles/{role}/permissions - Get role permissions (public)
		mux.HandleFunc("GET /api/admin/roles/",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(r.handleRolePermissions),
			),
		)

		// POST /api/auth/check - Check permission
		mux.HandleFunc("POST /api/auth/check",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(
					r.sessionMW.RequireAuth(http.HandlerFunc(r.policyHandlers.CheckPermission)).ServeHTTP,
				),
			),
		)

		// GET /api/auth/roles - Get current user's roles
		mux.HandleFunc("GET /api/auth/roles",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(
					r.sessionMW.RequireAuth(http.HandlerFunc(r.policyHandlers.GetUserRoles)).ServeHTTP,
				),
			),
		)

		// POST /api/admin/roles/assign - Assign role to user (admin only)
		mux.HandleFunc("POST /api/admin/roles/assign",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(
					r.sessionMW.RequireRole("admin", http.HandlerFunc(r.policyHandlers.AssignRole)).ServeHTTP,
				),
			),
		)

		// POST /api/admin/roles/revoke - Revoke role from user (admin only)
		mux.HandleFunc("POST /api/admin/roles/revoke",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(
					r.sessionMW.RequireRole("admin", http.HandlerFunc(r.policyHandlers.RevokeRole)).ServeHTTP,
				),
			),
		)

		// GET /api/admin/policies - Get all policies (admin only)
		mux.HandleFunc("GET /api/admin/policies",
			r.authMiddleware.CORS(
				r.authMiddleware.RateLimit(
					r.sessionMW.RequireRole("admin", http.HandlerFunc(r.policyHandlers.GetPolicies)).ServeHTTP,
				),
			),
		)
	}
}

// handleRevokeSession extracts session ID from path and calls RevokeSession.
func (r *ZeroTrustRouter) handleRevokeSession(w http.ResponseWriter, req *http.Request) {
	// Extract session ID from path: /api/auth/sessions/{id}
	path := req.URL.Path
	prefix := "/api/auth/sessions/"
	if !strings.HasPrefix(path, prefix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	sessionID := strings.TrimPrefix(path, prefix)
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	r.flowHandlers.RevokeSession(w, req, sessionID)
}

// handleRolePermissions extracts role from path and calls GetRolePermissions.
func (r *ZeroTrustRouter) handleRolePermissions(w http.ResponseWriter, req *http.Request) {
	// Extract role from path: /api/admin/roles/{role}/permissions
	path := req.URL.Path
	prefix := "/api/admin/roles/"
	if !strings.HasPrefix(path, prefix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	remainder := strings.TrimPrefix(path, prefix)
	parts := strings.Split(remainder, "/")
	if len(parts) < 2 || parts[1] != "permissions" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	role := parts[0]
	if role == "" {
		http.Error(w, "Role required", http.StatusBadRequest)
		return
	}

	r.policyHandlers.GetRolePermissions(w, req, role)
}

// RegisterZeroTrustRoutes registers Zero Trust routes on an http.ServeMux.
//
// Deprecated: This function is used only by tests. Production code uses
// router.SetupChi() with registerChiZeroTrustRoutes() (ADR-0016).
func RegisterZeroTrustRoutes(
	mux *http.ServeMux,
	flowHandlers *auth.FlowHandlers,
	policyHandlers *authz.PolicyHandlers,
	sessionMW *auth.SessionMiddleware,
	authMiddleware *auth.Middleware,
) {
	router := NewZeroTrustRouter(&ZeroTrustRouterConfig{
		FlowHandlers:   flowHandlers,
		PolicyHandlers: policyHandlers,
		SessionMW:      sessionMW,
		AuthMiddleware: authMiddleware,
	})
	router.RegisterRoutes(mux)
}
