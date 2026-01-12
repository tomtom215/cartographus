// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package authz

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/auth"
)

// =====================================================
// Policy Management API Tests
// ADR-0015: Zero Trust Authentication & Authorization
// =====================================================

func TestPolicyHandlers_ListRoles(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	handlers := NewPolicyHandlers(enforcer)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/roles", nil)
	w := httptest.NewRecorder()
	handlers.ListRoles(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp RolesResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should include at least viewer, editor, admin
	roleNames := make(map[string]bool)
	for _, role := range resp.Roles {
		roleNames[role.Name] = true
	}

	expectedRoles := []string{"viewer", "editor", "admin"}
	for _, expected := range expectedRoles {
		if !roleNames[expected] {
			t.Errorf("Expected role %q not found in response", expected)
		}
	}
}

func TestPolicyHandlers_GetRolePermissions(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	handlers := NewPolicyHandlers(enforcer)

	tests := []struct {
		name         string
		role         string
		wantStatus   int
		wantMinPerms int
	}{
		{
			name:         "viewer role permissions",
			role:         "viewer",
			wantStatus:   http.StatusOK,
			wantMinPerms: 1, // At least read permission
		},
		{
			name:         "editor role permissions",
			role:         "editor",
			wantStatus:   http.StatusOK,
			wantMinPerms: 1,
		},
		{
			name:         "admin role permissions",
			role:         "admin",
			wantStatus:   http.StatusOK,
			wantMinPerms: 1,
		},
		{
			name:       "unknown role",
			role:       "unknown_role",
			wantStatus: http.StatusOK, // Returns empty permissions
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/roles/"+tt.role+"/permissions", nil)
			w := httptest.NewRecorder()
			handlers.GetRolePermissions(w, req, tt.role)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantMinPerms > 0 {
				var resp PermissionsResponse
				err = json.NewDecoder(w.Body).Decode(&resp)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if len(resp.Permissions) < tt.wantMinPerms {
					t.Errorf("permissions count = %d, want at least %d", len(resp.Permissions), tt.wantMinPerms)
				}
			}
		})
	}
}

func TestPolicyHandlers_CheckPermission(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	handlers := NewPolicyHandlers(enforcer)

	tests := []struct {
		name        string
		subject     *auth.AuthSubject
		body        string
		wantStatus  int
		wantAllowed bool
	}{
		{
			name: "viewer can read",
			subject: &auth.AuthSubject{
				ID:    "user-1",
				Roles: []string{"viewer"},
			},
			body:        `{"object": "/api/sessions", "action": "read"}`,
			wantStatus:  http.StatusOK,
			wantAllowed: true,
		},
		{
			name: "viewer cannot write",
			subject: &auth.AuthSubject{
				ID:    "user-1",
				Roles: []string{"viewer"},
			},
			body:        `{"object": "/api/sessions", "action": "write"}`,
			wantStatus:  http.StatusOK,
			wantAllowed: false,
		},
		{
			name: "editor can write",
			subject: &auth.AuthSubject{
				ID:    "user-2",
				Roles: []string{"editor"},
			},
			body:        `{"object": "/api/sessions", "action": "write"}`,
			wantStatus:  http.StatusOK,
			wantAllowed: true,
		},
		{
			name: "admin can delete",
			subject: &auth.AuthSubject{
				ID:    "user-3",
				Roles: []string{"admin"},
			},
			body:        `{"object": "/api/admin/users", "action": "delete"}`,
			wantStatus:  http.StatusOK,
			wantAllowed: true,
		},
		{
			name:        "unauthenticated request",
			subject:     nil,
			body:        `{"object": "/api/sessions", "action": "read"}`,
			wantStatus:  http.StatusUnauthorized,
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/auth/check", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			if tt.subject != nil {
				ctx := context.WithValue(req.Context(), auth.AuthSubjectContextKey, tt.subject)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handlers.CheckPermission(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var resp CheckPermissionResponse
				err = json.NewDecoder(w.Body).Decode(&resp)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if resp.Allowed != tt.wantAllowed {
					t.Errorf("allowed = %v, want %v", resp.Allowed, tt.wantAllowed)
				}
			}
		})
	}
}

func TestPolicyHandlers_GetUserRoles(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	handlers := NewPolicyHandlers(enforcer)

	// Create request with auth subject
	subject := &auth.AuthSubject{
		ID:       "user-abc",
		Username: "testuser",
		Roles:    []string{"viewer", "editor"},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/roles", nil)
	ctx := context.WithValue(req.Context(), auth.AuthSubjectContextKey, subject)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.GetUserRoles(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp UserRolesResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Roles) != 2 {
		t.Errorf("roles count = %d, want 2", len(resp.Roles))
	}
}

func TestPolicyHandlers_AssignRole_AdminOnly(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	handlers := NewPolicyHandlers(enforcer)

	tests := []struct {
		name       string
		subject    *auth.AuthSubject
		body       string
		wantStatus int
	}{
		{
			name: "admin can assign role",
			subject: &auth.AuthSubject{
				ID:    "admin-user",
				Roles: []string{"admin"},
			},
			body:       `{"user_id": "user-123", "role": "editor"}`,
			wantStatus: http.StatusOK,
		},
		{
			name: "non-admin cannot assign role",
			subject: &auth.AuthSubject{
				ID:    "user-1",
				Roles: []string{"viewer"},
			},
			body:       `{"user_id": "user-123", "role": "editor"}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "unauthenticated cannot assign role",
			subject:    nil,
			body:       `{"user_id": "user-123", "role": "editor"}`,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/admin/roles/assign", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			if tt.subject != nil {
				ctx := context.WithValue(req.Context(), auth.AuthSubjectContextKey, tt.subject)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handlers.AssignRole(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestPolicyHandlers_RevokeRole_AdminOnly(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	handlers := NewPolicyHandlers(enforcer)

	// First assign a role to test revocation
	err = enforcer.AddGroupingPolicy("user-to-revoke", "editor")
	if err != nil {
		t.Fatalf("Failed to add initial role: %v", err)
	}

	tests := []struct {
		name       string
		subject    *auth.AuthSubject
		body       string
		wantStatus int
	}{
		{
			name: "admin can revoke role",
			subject: &auth.AuthSubject{
				ID:    "admin-user",
				Roles: []string{"admin"},
			},
			body:       `{"user_id": "user-to-revoke", "role": "editor"}`,
			wantStatus: http.StatusOK,
		},
		{
			name: "non-admin cannot revoke role",
			subject: &auth.AuthSubject{
				ID:    "user-1",
				Roles: []string{"viewer"},
			},
			body:       `{"user_id": "user-123", "role": "editor"}`,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/admin/roles/revoke", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			if tt.subject != nil {
				ctx := context.WithValue(req.Context(), auth.AuthSubjectContextKey, tt.subject)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handlers.RevokeRole(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// Response types for handler tests
type RolesResponse struct {
	Roles []RoleInfo `json:"roles"`
}

type RoleInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Inherits    []string `json:"inherits,omitempty"`
}

type PermissionsResponse struct {
	Permissions []Permission `json:"permissions"`
}

type Permission struct {
	Object string `json:"object"`
	Action string `json:"action"`
}

type CheckPermissionResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

type UserRolesResponse struct {
	Roles []string `json:"roles"`
}

type PoliciesResponse struct {
	Policies  []PolicyEntry   `json:"policies"`
	Groupings []GroupingEntry `json:"groupings"`
}

type PolicyEntry struct {
	Subject string `json:"subject"`
	Object  string `json:"object"`
	Action  string `json:"action"`
}

type GroupingEntry struct {
	Subject string `json:"subject"`
	Role    string `json:"role"`
}

// =====================================================
// GetPolicies Handler Tests
// =====================================================

func TestPolicyHandlers_GetPolicies(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	handlers := NewPolicyHandlers(enforcer)

	tests := []struct {
		name       string
		subject    *auth.AuthSubject
		wantStatus int
	}{
		{
			name: "admin can get policies",
			subject: &auth.AuthSubject{
				ID:    "admin-user",
				Roles: []string{"admin"},
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "non-admin cannot get policies",
			subject: &auth.AuthSubject{
				ID:    "user-1",
				Roles: []string{"viewer"},
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "unauthenticated cannot get policies",
			subject:    nil,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/policies", nil)

			if tt.subject != nil {
				ctx := context.WithValue(req.Context(), auth.AuthSubjectContextKey, tt.subject)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handlers.GetPolicies(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var resp PoliciesResponse
				err = json.NewDecoder(w.Body).Decode(&resp)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				// Should have at least some policies from embedded policy
				if len(resp.Policies) == 0 {
					t.Error("Expected at least one policy")
				}
			}
		})
	}
}

// =====================================================
// GetUserRoles Handler Edge Cases
// =====================================================

func TestPolicyHandlers_GetUserRoles_Unauthenticated(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	handlers := NewPolicyHandlers(enforcer)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/roles", nil)
	// No auth subject in context

	w := httptest.NewRecorder()
	handlers.GetUserRoles(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// =====================================================
// CheckPermission Handler Edge Cases
// =====================================================

func TestPolicyHandlers_CheckPermission_InvalidJSON(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	handlers := NewPolicyHandlers(enforcer)

	subject := &auth.AuthSubject{
		ID:    "user-1",
		Roles: []string{"viewer"},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/check", strings.NewReader("not valid json"))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), auth.AuthSubjectContextKey, subject)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.CheckPermission(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// =====================================================
// AssignRole Handler Edge Cases
// =====================================================

func TestPolicyHandlers_AssignRole_InvalidRole(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	handlers := NewPolicyHandlers(enforcer)

	subject := &auth.AuthSubject{
		ID:    "admin-user",
		Roles: []string{"admin"},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/roles/assign",
		strings.NewReader(`{"user_id": "user-123", "role": "invalid_role"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), auth.AuthSubjectContextKey, subject)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.AssignRole(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPolicyHandlers_AssignRole_MissingFields(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	handlers := NewPolicyHandlers(enforcer)

	subject := &auth.AuthSubject{
		ID:    "admin-user",
		Roles: []string{"admin"},
	}

	tests := []struct {
		name string
		body string
	}{
		{"missing user_id", `{"role": "editor"}`},
		{"missing role", `{"user_id": "user-123"}`},
		{"empty user_id", `{"user_id": "", "role": "editor"}`},
		{"empty role", `{"user_id": "user-123", "role": ""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/admin/roles/assign",
				strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), auth.AuthSubjectContextKey, subject)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handlers.AssignRole(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d for %s", w.Code, http.StatusBadRequest, tt.name)
			}
		})
	}
}

func TestPolicyHandlers_AssignRole_InvalidJSON(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	handlers := NewPolicyHandlers(enforcer)

	subject := &auth.AuthSubject{
		ID:    "admin-user",
		Roles: []string{"admin"},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/roles/assign",
		strings.NewReader("not valid json"))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), auth.AuthSubjectContextKey, subject)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.AssignRole(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// =====================================================
// RevokeRole Handler Edge Cases
// =====================================================

func TestPolicyHandlers_RevokeRole_MissingFields(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	handlers := NewPolicyHandlers(enforcer)

	subject := &auth.AuthSubject{
		ID:    "admin-user",
		Roles: []string{"admin"},
	}

	tests := []struct {
		name string
		body string
	}{
		{"missing user_id", `{"role": "editor"}`},
		{"missing role", `{"user_id": "user-123"}`},
		{"empty user_id", `{"user_id": "", "role": "editor"}`},
		{"empty role", `{"user_id": "user-123", "role": ""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/admin/roles/revoke",
				strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), auth.AuthSubjectContextKey, subject)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handlers.RevokeRole(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d for %s", w.Code, http.StatusBadRequest, tt.name)
			}
		})
	}
}

func TestPolicyHandlers_RevokeRole_InvalidJSON(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	handlers := NewPolicyHandlers(enforcer)

	subject := &auth.AuthSubject{
		ID:    "admin-user",
		Roles: []string{"admin"},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/roles/revoke",
		strings.NewReader("not valid json"))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), auth.AuthSubjectContextKey, subject)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.RevokeRole(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPolicyHandlers_RevokeRole_Unauthenticated(t *testing.T) {
	enforcer, err := NewEnforcer(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create enforcer: %v", err)
	}
	defer enforcer.Close()

	handlers := NewPolicyHandlers(enforcer)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/roles/revoke",
		strings.NewReader(`{"user_id": "user-123", "role": "editor"}`))
	req.Header.Set("Content-Type", "application/json")
	// No auth subject

	w := httptest.NewRecorder()
	handlers.RevokeRole(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
