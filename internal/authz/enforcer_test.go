// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package authz

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// =====================================================
// Test Helpers
// =====================================================

// setupEnforcer creates an enforcer with default config and registers cleanup.
func setupEnforcer(t *testing.T) *Enforcer {
	t.Helper()
	ctx := context.Background()
	enforcer, err := NewEnforcer(ctx, nil)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	t.Cleanup(func() { enforcer.Close() })
	return enforcer
}

// setupEnforcerWithCache creates an enforcer with caching enabled.
func setupEnforcerWithCache(t *testing.T) *Enforcer {
	t.Helper()
	ctx := context.Background()
	config := &EnforcerConfig{CacheEnabled: true}
	enforcer, err := NewEnforcer(ctx, config)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	t.Cleanup(func() { enforcer.Close() })
	return enforcer
}

// setupEnforcerWithConfig creates an enforcer with custom config.
func setupEnforcerWithConfig(t *testing.T, config *EnforcerConfig) *Enforcer {
	t.Helper()
	ctx := context.Background()
	enforcer, err := NewEnforcer(ctx, config)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	t.Cleanup(func() { enforcer.Close() })
	return enforcer
}

// setupTempPolicyDir creates a temp directory with a policy file and returns the path.
func setupTempPolicyDir(t *testing.T, policyContent string) (tmpDir, policyPath string) {
	t.Helper()
	var err error
	tmpDir, err = os.MkdirTemp("", "authz-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	policyPath = filepath.Join(tmpDir, "policy.csv")
	if policyContent != "" {
		if err := os.WriteFile(policyPath, []byte(policyContent), 0644); err != nil {
			t.Fatalf("Failed to write policy file: %v", err)
		}
	}
	return tmpDir, policyPath
}

// assertEnforce checks that enforcement returns expected result.
func assertEnforce(t *testing.T, enforcer *Enforcer, subject, object, action string, want bool) {
	t.Helper()
	got, err := enforcer.Enforce(subject, object, action)
	if err != nil {
		t.Errorf("Enforce(%q, %q, %q) error = %v", subject, object, action, err)
		return
	}
	if got != want {
		t.Errorf("Enforce(%q, %q, %q) = %v, want %v", subject, object, action, got, want)
	}
}

// =====================================================
// Tests
// =====================================================

// TestEnforcer_Creation tests enforcer initialization
func TestEnforcer_Creation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		config  *EnforcerConfig
		wantErr bool
	}{
		{
			name:    "nil config uses defaults",
			config:  nil,
			wantErr: false,
		},
		{
			name: "custom config",
			config: &EnforcerConfig{
				DefaultRole:  "viewer",
				CacheEnabled: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enforcer, err := NewEnforcer(ctx, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEnforcer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && enforcer == nil {
				t.Error("NewEnforcer() returned nil enforcer")
			}
			if enforcer != nil {
				enforcer.Close()
			}
		})
	}
}

// TestEnforcer_BasicRBAC tests basic RBAC enforcement
func TestEnforcer_BasicRBAC(t *testing.T) {
	enforcer := setupEnforcer(t)

	tests := []struct {
		name    string
		subject string
		object  string
		action  string
		want    bool
	}{
		// Admin should have full access
		{"admin can access admin endpoint", "admin", "/api/admin/users", "GET", true},
		{"admin can delete users", "admin", "/api/users/123", "DELETE", true},
		{"admin can access all endpoints", "admin", "/api/maps", "*", true},

		// Viewer has limited access
		{"viewer can read maps", "viewer", "/api/maps", "GET", true},
		{"viewer cannot create maps", "viewer", "/api/maps", "POST", false},
		{"viewer cannot access admin", "viewer", "/api/admin/users", "GET", false},

		// Editor has intermediate access
		{"editor can read maps", "editor", "/api/maps", "GET", true},
		{"editor can create maps", "editor", "/api/maps", "POST", true},
		{"editor cannot access admin", "editor", "/api/admin/users", "GET", false},

		// Unknown role
		{"unknown role denied", "unknown", "/api/maps", "GET", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEnforce(t, enforcer, tt.subject, tt.object, tt.action, tt.want)
		})
	}
}

// TestEnforcer_RoleManagement tests dynamic role assignment
func TestEnforcer_RoleManagement(t *testing.T) {
	enforcer := setupEnforcer(t)
	userID := "user-12345"

	// Initially user has no roles
	roles, err := enforcer.GetRolesForUser(userID)
	if err != nil {
		t.Fatalf("GetRolesForUser() error = %v", err)
	}
	if len(roles) != 0 {
		t.Errorf("New user should have no roles, got %v", roles)
	}

	// Add admin role
	added, err := enforcer.AddRoleForUser(userID, "admin")
	if err != nil {
		t.Fatalf("AddRoleForUser() error = %v", err)
	}
	if !added {
		t.Error("AddRoleForUser() should return true for new assignment")
	}

	// Verify role was added
	roles, err = enforcer.GetRolesForUser(userID)
	if err != nil {
		t.Fatalf("GetRolesForUser() error = %v", err)
	}
	if len(roles) != 1 || roles[0] != "admin" {
		t.Errorf("User should have admin role, got %v", roles)
	}

	// User should now have admin permissions
	assertEnforce(t, enforcer, userID, "/api/admin/users", "GET", true)

	// Remove role
	removed, err := enforcer.DeleteRoleForUser(userID, "admin")
	if err != nil {
		t.Fatalf("DeleteRoleForUser() error = %v", err)
	}
	if !removed {
		t.Error("DeleteRoleForUser() should return true")
	}

	// User should no longer have admin permissions
	assertEnforce(t, enforcer, userID, "/api/admin/users", "GET", false)
}

// TestEnforcer_EnforceWithRoles tests enforcement with provided roles
func TestEnforcer_EnforceWithRoles(t *testing.T) {
	enforcer := setupEnforcer(t)

	tests := []struct {
		name    string
		subject string
		roles   []string
		object  string
		action  string
		want    bool
	}{
		{
			name:    "user with admin role",
			subject: "user-123",
			roles:   []string{"admin"},
			object:  "/api/admin/users",
			action:  "GET",
			want:    true,
		},
		{
			name:    "user with viewer role",
			subject: "user-456",
			roles:   []string{"viewer"},
			object:  "/api/maps",
			action:  "GET",
			want:    true,
		},
		{
			name:    "user with viewer role cannot create",
			subject: "user-789",
			roles:   []string{"viewer"},
			object:  "/api/maps",
			action:  "POST",
			want:    false,
		},
		{
			name:    "user with multiple roles",
			subject: "user-multi",
			roles:   []string{"viewer", "editor"},
			object:  "/api/maps",
			action:  "POST",
			want:    true, // editor can POST
		},
		{
			name:    "user with no roles gets default role",
			subject: "user-noroles",
			roles:   []string{},
			object:  "/api/maps",
			action:  "GET",
			want:    true, // default role (viewer) is applied, viewer can GET /api/maps
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := enforcer.EnforceWithRoles(tt.subject, tt.roles, tt.object, tt.action)
			if err != nil {
				t.Errorf("EnforceWithRoles() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("EnforceWithRoles(%q, %v, %q, %q) = %v, want %v",
					tt.subject, tt.roles, tt.object, tt.action, got, tt.want)
			}
		})
	}
}

// TestEnforcer_PathMatching tests wildcard path matching
func TestEnforcer_PathMatching(t *testing.T) {
	enforcer := setupEnforcer(t)

	tests := []struct {
		name   string
		object string
		want   bool
	}{
		// These assume policy allows viewer to GET /api/maps/*
		{"exact path", "/api/maps", true},
		{"single segment wildcard", "/api/maps/123", true},
		{"multi segment", "/api/maps/123/details", true},
		{"different base path", "/api/users", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := enforcer.Enforce("viewer", tt.object, "GET")
			if err != nil {
				t.Errorf("Enforce() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Enforce(viewer, %q, GET) = %v, want %v",
					tt.object, got, tt.want)
			}
		})
	}
}

// TestEnforcer_CacheInvalidation tests that cache is invalidated on policy changes
func TestEnforcer_CacheInvalidation(t *testing.T) {
	enforcer := setupEnforcerWithCache(t)
	userID := "cache-test-user"

	// First check - should cache the result
	allowed1, _ := enforcer.Enforce(userID, "/api/maps", "GET")

	// Add role
	enforcer.AddRoleForUser(userID, "admin")

	// Second check - cache should be invalidated, new result
	allowed2, _ := enforcer.Enforce(userID, "/api/admin/users", "GET")

	// The user should now have admin access
	if !allowed2 && allowed1 == allowed2 {
		t.Error("Cache was not invalidated after role change")
	}
}

// TestDefaultEnforcerConfig verifies default configuration values
func TestDefaultEnforcerConfig(t *testing.T) {
	config := DefaultEnforcerConfig()

	if config == nil {
		t.Fatal("DefaultEnforcerConfig() returned nil")
	}
	if !config.AutoReload {
		t.Error("AutoReload should be true by default")
	}
	if config.ReloadInterval != 30*time.Second {
		t.Errorf("ReloadInterval = %v, want 30s", config.ReloadInterval)
	}
	if config.DefaultRole != "viewer" {
		t.Errorf("DefaultRole = %q, want 'viewer'", config.DefaultRole)
	}
	if !config.CacheEnabled {
		t.Error("CacheEnabled should be true by default")
	}
	if config.CacheTTL != 5*time.Minute {
		t.Errorf("CacheTTL = %v, want 5m", config.CacheTTL)
	}
}

// TestEnforcer_GetUsersForRole tests retrieving users with a specific role
func TestEnforcer_GetUsersForRole(t *testing.T) {
	enforcer := setupEnforcer(t)

	// Add some users to the admin role
	enforcer.AddRoleForUser("user-admin-1", "admin")
	enforcer.AddRoleForUser("user-admin-2", "admin")
	enforcer.AddRoleForUser("user-viewer-1", "viewer")

	// Get users for admin role
	users, err := enforcer.GetUsersForRole("admin")
	if err != nil {
		t.Fatalf("GetUsersForRole() error = %v", err)
	}

	// Should have at least 2 admin users
	if len(users) < 2 {
		t.Errorf("Expected at least 2 admin users, got %d", len(users))
	}

	// Verify specific users are in the list
	userMap := make(map[string]bool)
	for _, u := range users {
		userMap[u] = true
	}
	if !userMap["user-admin-1"] {
		t.Error("user-admin-1 should be in admin role")
	}
	if !userMap["user-admin-2"] {
		t.Error("user-admin-2 should be in admin role")
	}
}

// TestEnforcer_AddPolicy tests adding new policy rules
func TestEnforcer_AddPolicy(t *testing.T) {
	enforcer := setupEnforcer(t)

	// Add a custom policy for a specific user
	added, err := enforcer.AddPolicy("custom-user", "/api/custom", "GET")
	if err != nil {
		t.Fatalf("AddPolicy() error = %v", err)
	}
	if !added {
		t.Error("AddPolicy() should return true for new policy")
	}

	// Verify the policy works
	allowed, err := enforcer.Enforce("custom-user", "/api/custom", "GET")
	if err != nil {
		t.Fatalf("Enforce() error = %v", err)
	}
	if !allowed {
		t.Error("custom-user should have access after AddPolicy")
	}

	// Adding the same policy again should return false (already exists)
	added, err = enforcer.AddPolicy("custom-user", "/api/custom", "GET")
	if err != nil {
		t.Fatalf("AddPolicy() error = %v", err)
	}
	if added {
		t.Error("AddPolicy() should return false for duplicate policy")
	}
}

// TestEnforcer_RemovePolicy tests removing policy rules
func TestEnforcer_RemovePolicy(t *testing.T) {
	enforcer := setupEnforcer(t)

	// Add a policy first
	enforcer.AddPolicy("remove-test-user", "/api/removable", "GET")

	// Verify it works
	allowed, _ := enforcer.Enforce("remove-test-user", "/api/removable", "GET")
	if !allowed {
		t.Error("Policy should be active before removal")
	}

	// Remove the policy
	removed, err := enforcer.RemovePolicy("remove-test-user", "/api/removable", "GET")
	if err != nil {
		t.Fatalf("RemovePolicy() error = %v", err)
	}
	if !removed {
		t.Error("RemovePolicy() should return true")
	}

	// Verify it no longer works
	allowed, _ = enforcer.Enforce("remove-test-user", "/api/removable", "GET")
	if allowed {
		t.Error("Policy should be inactive after removal")
	}

	// Removing non-existent policy should return false
	removed, err = enforcer.RemovePolicy("non-existent", "/api/nothing", "GET")
	if err != nil {
		t.Fatalf("RemovePolicy() error = %v", err)
	}
	if removed {
		t.Error("RemovePolicy() should return false for non-existent policy")
	}
}

// TestEnforcer_GetPolicy tests retrieving all policy rules
func TestEnforcer_GetPolicy(t *testing.T) {
	enforcer := setupEnforcer(t)

	// Get all policies (should include embedded policies)
	policies := enforcer.GetPolicy()

	// Should have some policies from embedded policy
	if len(policies) == 0 {
		t.Error("GetPolicy() should return policies from embedded policy")
	}

	// Each policy should have 3 elements: subject, object, action
	for i, policy := range policies {
		if len(policy) < 3 {
			t.Errorf("Policy %d has %d elements, want at least 3", i, len(policy))
		}
	}
}

// TestEnforcer_GetFilteredPolicy tests filtered policy retrieval
func TestEnforcer_GetFilteredPolicy(t *testing.T) {
	enforcer := setupEnforcer(t)

	// Filter by subject (field index 0)
	adminPolicies := enforcer.GetFilteredPolicy(0, "admin")
	if len(adminPolicies) == 0 {
		t.Error("GetFilteredPolicy() should return admin policies")
	}

	// All returned policies should have admin as subject
	for _, policy := range adminPolicies {
		if len(policy) > 0 && policy[0] != "admin" {
			t.Errorf("Filtered policy has subject %q, want 'admin'", policy[0])
		}
	}

	// Filter viewer policies
	viewerPolicies := enforcer.GetFilteredPolicy(0, "viewer")
	if len(viewerPolicies) == 0 {
		t.Error("GetFilteredPolicy() should return viewer policies")
	}
}

// TestEnforcer_GetGroupingPolicy tests retrieving role inheritance rules
func TestEnforcer_GetGroupingPolicy(t *testing.T) {
	enforcer := setupEnforcer(t)

	// Add some grouping policies (role assignments)
	enforcer.AddGroupingPolicy("group-test-user", "editor")

	// Get all grouping policies
	groupings := enforcer.GetGroupingPolicy()

	// Should have at least one grouping (the one we just added + embedded)
	if len(groupings) == 0 {
		t.Error("GetGroupingPolicy() should return grouping policies")
	}

	// Each grouping should have at least 2 elements: subject, role
	for i, grouping := range groupings {
		if len(grouping) < 2 {
			t.Errorf("Grouping %d has %d elements, want at least 2", i, len(grouping))
		}
	}
}

// TestEnforcer_AddGroupingPolicy tests adding role assignments
func TestEnforcer_AddGroupingPolicy(t *testing.T) {
	enforcer := setupEnforcer(t)

	// Add a grouping policy
	err := enforcer.AddGroupingPolicy("grouping-test-user", "admin")
	if err != nil {
		t.Fatalf("AddGroupingPolicy() error = %v", err)
	}

	// Verify user has the role
	roles, _ := enforcer.GetRolesForUser("grouping-test-user")
	found := false
	for _, r := range roles {
		if r == "admin" {
			found = true
			break
		}
	}
	if !found {
		t.Error("User should have admin role after AddGroupingPolicy")
	}
}

// TestEnforcer_RemoveGroupingPolicy tests removing role assignments
func TestEnforcer_RemoveGroupingPolicy(t *testing.T) {
	enforcer := setupEnforcer(t)

	// Add then remove a grouping policy
	enforcer.AddGroupingPolicy("remove-grouping-user", "editor")

	err := enforcer.RemoveGroupingPolicy("remove-grouping-user", "editor")
	if err != nil {
		t.Fatalf("RemoveGroupingPolicy() error = %v", err)
	}

	// Verify user no longer has the role
	roles, _ := enforcer.GetRolesForUser("remove-grouping-user")
	for _, r := range roles {
		if r == "editor" {
			t.Error("User should not have editor role after RemoveGroupingPolicy")
		}
	}
}

// TestEnforcer_CacheDisabled tests enforcer without cache
func TestEnforcer_CacheDisabled(t *testing.T) {
	config := &EnforcerConfig{CacheEnabled: false}
	enforcer := setupEnforcerWithConfig(t, config)

	// Basic enforcement should work without cache
	assertEnforce(t, enforcer, "viewer", "/api/maps", "GET", true)
}

// TestFileExists tests the fileExists helper function
func TestFileExists(t *testing.T) {
	// Test with existing file (this test file)
	if !fileExists("enforcer_test.go") {
		t.Error("fileExists() should return true for existing file")
	}

	// Test with non-existing file
	if fileExists("non-existent-file-12345.txt") {
		t.Error("fileExists() should return false for non-existing file")
	}

	// Test with empty path
	if fileExists("") {
		t.Error("fileExists() should return false for empty path")
	}
}

// TestEnforcer_SavePolicy_NoAdapter tests SavePolicy with no file adapter
// ADR-0015 Phase 4 Continuation: Graceful handling of missing adapter
func TestEnforcer_SavePolicy_NoAdapter(t *testing.T) {
	enforcer := setupEnforcer(t) // nil config uses embedded policy, no file adapter

	// SavePolicy should return ErrNoAdapter instead of panicking
	err := enforcer.SavePolicy()
	if err == nil {
		t.Error("SavePolicy() should return error with no adapter")
	}
	if !errors.Is(err, ErrNoAdapter) {
		t.Errorf("SavePolicy() error = %v, want ErrNoAdapter", err)
	}
}

// TestEnforcer_LoadPolicy_NoAdapter tests LoadPolicy with no file adapter
// ADR-0015 Phase 4 Continuation: Graceful handling of missing adapter
func TestEnforcer_LoadPolicy_NoAdapter(t *testing.T) {
	enforcer := setupEnforcer(t) // nil config uses embedded policy, no file adapter

	// LoadPolicy should return ErrNoAdapter instead of panicking
	err := enforcer.LoadPolicy()
	if err == nil {
		t.Error("LoadPolicy() should return error with no adapter")
	}
	if !errors.Is(err, ErrNoAdapter) {
		t.Errorf("LoadPolicy() error = %v, want ErrNoAdapter", err)
	}
}

// TestEnforcer_Close tests cleanup
func TestEnforcer_Close(t *testing.T) {
	ctx := context.Background()
	enforcer, err := NewEnforcer(ctx, nil)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}

	// Close should not panic
	enforcer.Close()

	// ADR-0015 Phase 4 Continuation: Close should be idempotent
	// Calling Close twice should not panic (cache.stop uses sync.Once)
	enforcer.Close()
	enforcer.Close() // Third call should also be safe
}

// TestEnforcer_InvalidModelPath tests with invalid model path
func TestEnforcer_InvalidModelPath(t *testing.T) {
	ctx := context.Background()
	config := &EnforcerConfig{
		ModelPath: "non-existent-model.conf",
	}
	// Should fall back to embedded model
	enforcer, err := NewEnforcer(ctx, config)
	if err != nil {
		t.Fatalf("NewEnforcer() should use embedded model when file not found: %v", err)
	}
	defer enforcer.Close()

	// Basic enforcement should work
	allowed, _ := enforcer.Enforce("admin", "/api/admin/users", "GET")
	if !allowed {
		t.Error("Admin should have access with embedded model fallback")
	}
}

// =====================================================
// File-Based Policy Tests
// ADR-0015: Zero Trust - Policy persistence
// =====================================================

func TestEnforcer_FileBasedPolicy(t *testing.T) {
	policyContent := `p, admin, /api/*, *
p, editor, /api/*, GET
p, editor, /api/*, POST
p, viewer, /api/sessions, GET
g, editor, viewer
g, admin, editor
`
	_, policyPath := setupTempPolicyDir(t, policyContent)

	config := &EnforcerConfig{
		PolicyPath:   policyPath,
		CacheEnabled: true,
	}
	enforcer := setupEnforcerWithConfig(t, config)

	// Verify policies loaded from file
	assertEnforce(t, enforcer, "admin", "/api/users", "DELETE", true)
	assertEnforce(t, enforcer, "viewer", "/api/sessions", "GET", true)
}

func TestEnforcer_SavePolicy_WithFileAdapter(t *testing.T) {
	// Create temporary directory for policy files
	tmpDir, err := os.MkdirTemp("", "authz-save-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create initial policy file
	policyPath := filepath.Join(tmpDir, "policy.csv")
	initialPolicy := `p, admin, /api/*, *
p, viewer, /api/sessions, GET
`
	err = os.WriteFile(policyPath, []byte(initialPolicy), 0644)
	if err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	// Create enforcer with file-based policy
	ctx := context.Background()
	config := &EnforcerConfig{
		PolicyPath:   policyPath,
		CacheEnabled: false, // Disable cache to ensure fresh reads
		AutoReload:   false,
	}

	enforcer, err := NewEnforcer(ctx, config)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	defer enforcer.Close()

	// Add a new policy
	added, err := enforcer.AddPolicy("editor", "/api/maps", "POST")
	if err != nil {
		t.Fatalf("AddPolicy() error = %v", err)
	}
	if !added {
		t.Error("AddPolicy() should return true for new policy")
	}

	// Save policy to file
	err = enforcer.SavePolicy()
	if err != nil {
		t.Fatalf("SavePolicy() error = %v", err)
	}

	// Verify policy was saved by reading the file
	savedContent, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatalf("Failed to read saved policy: %v", err)
	}

	// Check that editor policy is in the saved file
	if !contains(string(savedContent), "editor") {
		t.Error("Saved policy should contain editor rule")
	}
}

func TestEnforcer_LoadPolicy_WithFileAdapter(t *testing.T) {
	// Create temporary directory for policy files
	tmpDir, err := os.MkdirTemp("", "authz-load-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create initial policy file
	policyPath := filepath.Join(tmpDir, "policy.csv")
	initialPolicy := `p, admin, /api/*, *
`
	err = os.WriteFile(policyPath, []byte(initialPolicy), 0644)
	if err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	// Create enforcer with file-based policy
	ctx := context.Background()
	config := &EnforcerConfig{
		PolicyPath:   policyPath,
		CacheEnabled: true,
		AutoReload:   false,
	}

	enforcer, err := NewEnforcer(ctx, config)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	defer enforcer.Close()

	// Initially, only admin policy exists
	allowed, _ := enforcer.Enforce("viewer", "/api/maps", "GET")
	if allowed {
		t.Error("Viewer should not have access initially")
	}

	// Update policy file externally
	updatedPolicy := `p, admin, /api/*, *
p, viewer, /api/maps, GET
`
	err = os.WriteFile(policyPath, []byte(updatedPolicy), 0644)
	if err != nil {
		t.Fatalf("Failed to update policy file: %v", err)
	}

	// Reload policy
	err = enforcer.LoadPolicy()
	if err != nil {
		t.Fatalf("LoadPolicy() error = %v", err)
	}

	// Now viewer should have access
	allowed, _ = enforcer.Enforce("viewer", "/api/maps", "GET")
	if !allowed {
		t.Error("Viewer should have access after policy reload")
	}
}

func TestEnforcer_LoadPolicy_CacheCleared(t *testing.T) {
	// Create temporary directory for policy files
	tmpDir, err := os.MkdirTemp("", "authz-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create initial policy file
	policyPath := filepath.Join(tmpDir, "policy.csv")
	initialPolicy := `p, admin, /api/*, *
p, tester, /api/test, GET
`
	err = os.WriteFile(policyPath, []byte(initialPolicy), 0644)
	if err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	// Create enforcer with cache enabled
	ctx := context.Background()
	config := &EnforcerConfig{
		PolicyPath:   policyPath,
		CacheEnabled: true,
		AutoReload:   false,
	}

	enforcer, err := NewEnforcer(ctx, config)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	defer enforcer.Close()

	// Warm up cache
	allowed, _ := enforcer.Enforce("tester", "/api/test", "GET")
	if !allowed {
		t.Error("Tester should have access initially")
	}

	// Update policy file - remove tester access
	updatedPolicy := `p, admin, /api/*, *
`
	err = os.WriteFile(policyPath, []byte(updatedPolicy), 0644)
	if err != nil {
		t.Fatalf("Failed to update policy file: %v", err)
	}

	// Reload policy - this should clear cache
	err = enforcer.LoadPolicy()
	if err != nil {
		t.Fatalf("LoadPolicy() error = %v", err)
	}

	// Tester should no longer have access (cache was cleared)
	allowed, _ = enforcer.Enforce("tester", "/api/test", "GET")
	if allowed {
		t.Error("Tester should not have access after policy reload (cache should be cleared)")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =====================================================
// EnforceWithRoles Additional Tests
// =====================================================

func TestEnforcer_EnforceWithRoles_DirectPermission(t *testing.T) {
	ctx := context.Background()
	enforcer, err := NewEnforcer(ctx, nil)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	defer enforcer.Close()

	// Add a direct permission for a user (not via role)
	_, err = enforcer.AddPolicy("user-direct", "/api/special", "GET")
	if err != nil {
		t.Fatalf("AddPolicy() error = %v", err)
	}

	// User should have access via direct permission, not roles
	allowed, err := enforcer.EnforceWithRoles("user-direct", []string{}, "/api/special", "GET")
	if err != nil {
		t.Fatalf("EnforceWithRoles() error = %v", err)
	}
	if !allowed {
		t.Error("User should have access via direct permission")
	}
}

func TestEnforcer_EnforceWithRoles_NoDefaultRole(t *testing.T) {
	ctx := context.Background()
	config := &EnforcerConfig{
		DefaultRole:  "", // No default role
		CacheEnabled: true,
	}
	enforcer, err := NewEnforcer(ctx, config)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	defer enforcer.Close()

	// User with no roles and no default role should be denied
	allowed, err := enforcer.EnforceWithRoles("user-no-default", []string{}, "/api/maps", "GET")
	if err != nil {
		t.Fatalf("EnforceWithRoles() error = %v", err)
	}
	if allowed {
		t.Error("User with no roles and no default role should be denied")
	}
}

func TestEnforcer_EnforceWithRoles_MultipleRolesFirstMatch(t *testing.T) {
	ctx := context.Background()
	enforcer, err := NewEnforcer(ctx, nil)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	defer enforcer.Close()

	// Test with multiple roles where first role allows access
	allowed, err := enforcer.EnforceWithRoles("multi-role-user", []string{"admin", "viewer"}, "/api/admin/users", "DELETE")
	if err != nil {
		t.Fatalf("EnforceWithRoles() error = %v", err)
	}
	if !allowed {
		t.Error("User with admin role should have access")
	}

	// Test with multiple roles where second role allows access
	allowed, err = enforcer.EnforceWithRoles("multi-role-user", []string{"viewer", "admin"}, "/api/admin/users", "DELETE")
	if err != nil {
		t.Fatalf("EnforceWithRoles() error = %v", err)
	}
	if !allowed {
		t.Error("User with admin as second role should still have access")
	}
}

// =====================================================
// AddRoleForUser/DeleteRoleForUser Cache Tests
// =====================================================

func TestEnforcer_AddRoleForUser_CacheInvalidation(t *testing.T) {
	ctx := context.Background()
	config := &EnforcerConfig{
		CacheEnabled: true,
	}
	enforcer, err := NewEnforcer(ctx, config)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	defer enforcer.Close()

	userID := "cache-user-add"

	// First, cache a denial for this user
	allowed, _ := enforcer.Enforce(userID, "/api/admin/users", "GET")
	if allowed {
		t.Error("User should not have access initially")
	}

	// Add admin role - should invalidate cache
	_, err = enforcer.AddRoleForUser(userID, "admin")
	if err != nil {
		t.Fatalf("AddRoleForUser() error = %v", err)
	}

	// Now user should have access (cache was invalidated)
	allowed, _ = enforcer.Enforce(userID, "/api/admin/users", "GET")
	if !allowed {
		t.Error("User should have access after role added")
	}
}

func TestEnforcer_DeleteRoleForUser_CacheInvalidation(t *testing.T) {
	ctx := context.Background()
	config := &EnforcerConfig{
		CacheEnabled: true,
	}
	enforcer, err := NewEnforcer(ctx, config)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	defer enforcer.Close()

	userID := "cache-user-delete"

	// Add admin role first
	_, err = enforcer.AddRoleForUser(userID, "admin")
	if err != nil {
		t.Fatalf("AddRoleForUser() error = %v", err)
	}

	// Cache the allowed result
	allowed, _ := enforcer.Enforce(userID, "/api/admin/users", "GET")
	if !allowed {
		t.Error("User should have access with admin role")
	}

	// Remove admin role - should invalidate cache
	_, err = enforcer.DeleteRoleForUser(userID, "admin")
	if err != nil {
		t.Fatalf("DeleteRoleForUser() error = %v", err)
	}

	// Now user should not have access (cache was invalidated)
	allowed, _ = enforcer.Enforce(userID, "/api/admin/users", "GET")
	if allowed {
		t.Error("User should not have access after role removed")
	}
}

// =====================================================
// AddPolicy/RemovePolicy Cache Tests
// =====================================================

func TestEnforcer_AddPolicy_CacheCleared(t *testing.T) {
	ctx := context.Background()
	config := &EnforcerConfig{
		CacheEnabled: true,
	}
	enforcer, err := NewEnforcer(ctx, config)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	defer enforcer.Close()

	// Cache a denial
	allowed, _ := enforcer.Enforce("new-role", "/api/new", "GET")
	if allowed {
		t.Error("new-role should not have access initially")
	}

	// Add policy - should clear cache
	_, err = enforcer.AddPolicy("new-role", "/api/new", "GET")
	if err != nil {
		t.Fatalf("AddPolicy() error = %v", err)
	}

	// Now should have access
	allowed, _ = enforcer.Enforce("new-role", "/api/new", "GET")
	if !allowed {
		t.Error("new-role should have access after policy added")
	}
}

func TestEnforcer_RemovePolicy_CacheCleared(t *testing.T) {
	ctx := context.Background()
	config := &EnforcerConfig{
		CacheEnabled: true,
	}
	enforcer, err := NewEnforcer(ctx, config)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	defer enforcer.Close()

	// Add a policy
	_, err = enforcer.AddPolicy("temp-role", "/api/temp", "GET")
	if err != nil {
		t.Fatalf("AddPolicy() error = %v", err)
	}

	// Cache the allowed result
	allowed, _ := enforcer.Enforce("temp-role", "/api/temp", "GET")
	if !allowed {
		t.Error("temp-role should have access")
	}

	// Remove policy - should clear cache
	_, err = enforcer.RemovePolicy("temp-role", "/api/temp", "GET")
	if err != nil {
		t.Fatalf("RemovePolicy() error = %v", err)
	}

	// Now should not have access
	allowed, _ = enforcer.Enforce("temp-role", "/api/temp", "GET")
	if allowed {
		t.Error("temp-role should not have access after policy removed")
	}
}

// =====================================================
// AddGroupingPolicy/RemoveGroupingPolicy Cache Tests
// =====================================================

func TestEnforcer_AddGroupingPolicy_CacheInvalidation(t *testing.T) {
	ctx := context.Background()
	config := &EnforcerConfig{
		CacheEnabled: true,
	}
	enforcer, err := NewEnforcer(ctx, config)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	defer enforcer.Close()

	userID := "grouping-cache-user"

	// Cache a denial
	allowed, _ := enforcer.Enforce(userID, "/api/admin/users", "GET")
	if allowed {
		t.Error("User should not have access initially")
	}

	// Add grouping policy - should invalidate cache for user
	err = enforcer.AddGroupingPolicy(userID, "admin")
	if err != nil {
		t.Fatalf("AddGroupingPolicy() error = %v", err)
	}

	// Now should have access
	allowed, _ = enforcer.Enforce(userID, "/api/admin/users", "GET")
	if !allowed {
		t.Error("User should have access after grouping added")
	}
}

func TestEnforcer_RemoveGroupingPolicy_CacheInvalidation(t *testing.T) {
	ctx := context.Background()
	config := &EnforcerConfig{
		CacheEnabled: true,
	}
	enforcer, err := NewEnforcer(ctx, config)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	defer enforcer.Close()

	userID := "grouping-remove-user"

	// Add grouping first
	err = enforcer.AddGroupingPolicy(userID, "admin")
	if err != nil {
		t.Fatalf("AddGroupingPolicy() error = %v", err)
	}

	// Cache the allowed result
	allowed, _ := enforcer.Enforce(userID, "/api/admin/users", "GET")
	if !allowed {
		t.Error("User should have access with admin grouping")
	}

	// Remove grouping - should invalidate cache
	err = enforcer.RemoveGroupingPolicy(userID, "admin")
	if err != nil {
		t.Fatalf("RemoveGroupingPolicy() error = %v", err)
	}

	// Now should not have access
	allowed, _ = enforcer.Enforce(userID, "/api/admin/users", "GET")
	if allowed {
		t.Error("User should not have access after grouping removed")
	}
}

// =====================================================
// NewEnforcer Edge Cases
// =====================================================

func TestEnforcer_NewEnforcer_WithFileModel(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "authz-model-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create model file
	modelPath := filepath.Join(tmpDir, "model.conf")
	modelContent := `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
`
	err = os.WriteFile(modelPath, []byte(modelContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write model file: %v", err)
	}

	// Create enforcer with file-based model
	ctx := context.Background()
	config := &EnforcerConfig{
		ModelPath: modelPath,
	}

	enforcer, err := NewEnforcer(ctx, config)
	if err != nil {
		t.Fatalf("NewEnforcer() with file model error = %v", err)
	}
	defer enforcer.Close()

	// Add a policy and verify it works
	_, err = enforcer.AddPolicy("test-user", "/api/test", "GET")
	if err != nil {
		t.Fatalf("AddPolicy() error = %v", err)
	}

	allowed, err := enforcer.Enforce("test-user", "/api/test", "GET")
	if err != nil {
		t.Fatalf("Enforce() error = %v", err)
	}
	if !allowed {
		t.Error("User should have access with file model")
	}
}

func TestEnforcer_NewEnforcer_WithAutoReload(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "authz-reload-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create policy file
	policyPath := filepath.Join(tmpDir, "policy.csv")
	err = os.WriteFile(policyPath, []byte("p, admin, /api/*, *\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	// Create enforcer with auto-reload enabled
	ctx := context.Background()
	config := &EnforcerConfig{
		PolicyPath:     policyPath,
		AutoReload:     true,
		ReloadInterval: 100 * time.Millisecond,
	}

	enforcer, err := NewEnforcer(ctx, config)
	if err != nil {
		t.Fatalf("NewEnforcer() with auto-reload error = %v", err)
	}
	defer enforcer.Close()

	// Verify initial policy works
	allowed, _ := enforcer.Enforce("admin", "/api/test", "GET")
	if !allowed {
		t.Error("Admin should have access initially")
	}
}
