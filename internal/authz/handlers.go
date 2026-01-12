// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package authz provides authorization middleware using Casbin.
// ADR-0015: Zero Trust Authentication & Authorization
package authz

import (
	"net/http"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/logging"
)

// PolicyHandlers provides HTTP handlers for policy management operations.
type PolicyHandlers struct {
	enforcer *Enforcer
}

// NewPolicyHandlers creates a new PolicyHandlers instance.
func NewPolicyHandlers(enforcer *Enforcer) *PolicyHandlers {
	return &PolicyHandlers{
		enforcer: enforcer,
	}
}

// ListRoles returns all available roles.
// GET /api/admin/roles
func (h *PolicyHandlers) ListRoles(w http.ResponseWriter, r *http.Request) {
	// Define the standard roles with their descriptions and hierarchy
	roles := []map[string]interface{}{
		{
			"name":        "viewer",
			"description": "Read-only access to resources",
			"inherits":    []string{},
		},
		{
			"name":        "editor",
			"description": "Can view and modify resources",
			"inherits":    []string{"viewer"},
		},
		{
			"name":        "admin",
			"description": "Full access to all resources",
			"inherits":    []string{"editor"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"roles": roles,
	}); err != nil {
		logging.Error().Err(err).Msg("Failed to encode roles response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// GetRolePermissions returns permissions for a specific role.
// GET /api/admin/roles/:role/permissions
func (h *PolicyHandlers) GetRolePermissions(w http.ResponseWriter, r *http.Request, role string) {
	// Get policies for the role
	policies := h.enforcer.GetFilteredPolicy(0, role)

	permissions := make([]map[string]string, 0, len(policies))
	for _, policy := range policies {
		if len(policy) >= 3 {
			permissions = append(permissions, map[string]string{
				"object": policy[1],
				"action": policy[2],
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"role":        role,
		"permissions": permissions,
	}); err != nil {
		logging.Error().Err(err).Msg("Failed to encode permissions response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// CheckPermission checks if the authenticated user has permission for an action.
// POST /api/auth/check
func (h *PolicyHandlers) CheckPermission(w http.ResponseWriter, r *http.Request) {
	subject := auth.GetAuthSubject(r.Context())
	if subject == nil {
		http.Error(w, "Unauthorized: not authenticated", http.StatusUnauthorized)
		return
	}

	var req struct {
		Object string `json:"object"`
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check permission using the enforcer
	allowed, err := h.enforcer.EnforceWithRoles(subject.ID, subject.Roles, req.Object, req.Action)
	if err != nil {
		logging.Error().Err(err).Msg("Permission check error")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	reason := ""
	if !allowed {
		reason = "Insufficient permissions"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"allowed": allowed,
		"reason":  reason,
	}); err != nil {
		logging.Error().Err(err).Msg("Failed to encode check-permission response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// GetUserRoles returns the authenticated user's roles.
// GET /api/auth/roles
func (h *PolicyHandlers) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	subject := auth.GetAuthSubject(r.Context())
	if subject == nil {
		http.Error(w, "Unauthorized: not authenticated", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"roles": subject.Roles,
	}); err != nil {
		logging.Error().Err(err).Msg("Failed to encode user-roles response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// AssignRole assigns a role to a user. Admin only.
// POST /api/admin/roles/assign
func (h *PolicyHandlers) AssignRole(w http.ResponseWriter, r *http.Request) {
	subject := auth.GetAuthSubject(r.Context())
	if subject == nil {
		http.Error(w, "Unauthorized: not authenticated", http.StatusUnauthorized)
		return
	}

	// Only admins can assign roles
	if !subject.HasRole("admin") {
		http.Error(w, "Forbidden: admin role required", http.StatusForbidden)
		return
	}

	var req struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" || req.Role == "" {
		http.Error(w, "user_id and role are required", http.StatusBadRequest)
		return
	}

	// Validate role exists
	validRoles := map[string]bool{"viewer": true, "editor": true, "admin": true}
	if !validRoles[req.Role] {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}

	// Add the role assignment (grouping policy in Casbin)
	err := h.enforcer.AddGroupingPolicy(req.UserID, req.Role)
	if err != nil {
		logging.Error().Err(err).Msg("Failed to assign role")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "Role assigned successfully",
	}); err != nil {
		logging.Error().Err(err).Msg("Failed to encode assign-role response")
	}
}

// RevokeRole revokes a role from a user. Admin only.
// POST /api/admin/roles/revoke
func (h *PolicyHandlers) RevokeRole(w http.ResponseWriter, r *http.Request) {
	subject := auth.GetAuthSubject(r.Context())
	if subject == nil {
		http.Error(w, "Unauthorized: not authenticated", http.StatusUnauthorized)
		return
	}

	// Only admins can revoke roles
	if !subject.HasRole("admin") {
		http.Error(w, "Forbidden: admin role required", http.StatusForbidden)
		return
	}

	var req struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" || req.Role == "" {
		http.Error(w, "user_id and role are required", http.StatusBadRequest)
		return
	}

	// Remove the role assignment (grouping policy in Casbin)
	err := h.enforcer.RemoveGroupingPolicy(req.UserID, req.Role)
	if err != nil {
		logging.Error().Err(err).Msg("Failed to revoke role")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "Role revoked successfully",
	}); err != nil {
		logging.Error().Err(err).Msg("Failed to encode revoke-role response")
	}
}

// GetPolicies returns all policies. Admin only.
// GET /api/admin/policies
func (h *PolicyHandlers) GetPolicies(w http.ResponseWriter, r *http.Request) {
	subject := auth.GetAuthSubject(r.Context())
	if subject == nil {
		http.Error(w, "Unauthorized: not authenticated", http.StatusUnauthorized)
		return
	}

	// Only admins can view policies
	if !subject.HasRole("admin") {
		http.Error(w, "Forbidden: admin role required", http.StatusForbidden)
		return
	}

	// Get all policies
	policies := h.enforcer.GetPolicy()

	policyList := make([]map[string]string, 0, len(policies))
	for _, policy := range policies {
		if len(policy) >= 3 {
			policyList = append(policyList, map[string]string{
				"subject": policy[0],
				"object":  policy[1],
				"action":  policy[2],
			})
		}
	}

	// Get all grouping policies (role assignments and hierarchy)
	groupings := h.enforcer.GetGroupingPolicy()

	groupingList := make([]map[string]string, 0, len(groupings))
	for _, grouping := range groupings {
		if len(grouping) >= 2 {
			groupingList = append(groupingList, map[string]string{
				"subject": grouping[0],
				"role":    grouping[1],
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"policies":  policyList,
		"groupings": groupingList,
	}); err != nil {
		logging.Error().Err(err).Msg("Failed to encode policies response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
