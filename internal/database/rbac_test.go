// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
rbac_test.go - Tests for RBAC Database Operations

This file provides comprehensive tests for Role-Based Access Control operations
including user role management and audit logging.

Test Coverage:
  - GetUserRole: Retrieve user roles (success, not found, expired)
  - GetEffectiveRole: Default role handling
  - SetUserRole: Create, update, validation
  - DeleteUserRole: Soft delete, audit logging
  - ListUserRoles: Filtering by active/role
  - AuditRoleChange: Audit log creation
  - GetRoleAuditLog: Audit log retrieval with filtering
  - ExpireRoles: Automatic role expiration
  - Thread safety: Concurrent role operations
  - Edge cases: Special characters, Unicode
*/

package database

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/models"
)

// TestGetUserRole tests retrieving user roles
func TestGetUserRole(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	t.Run("returns error when role not found", func(t *testing.T) {
		_, err := db.GetUserRole(ctx, "nonexistent-user")
		if !errors.Is(err, ErrRoleNotFound) {
			t.Errorf("Expected ErrRoleNotFound, got %v", err)
		}
	})

	t.Run("returns role when exists", func(t *testing.T) {
		role := models.NewUserRole("test-user-1", "testuser", models.RoleEditor, "admin")
		_, err := db.SetUserRole(ctx, role, "admin", "admin", "test")
		if err != nil {
			t.Fatalf("SetUserRole failed: %v", err)
		}

		found, err := db.GetUserRole(ctx, "test-user-1")
		if err != nil {
			t.Fatalf("GetUserRole failed: %v", err)
		}

		if found.UserID != "test-user-1" {
			t.Errorf("Expected UserID=test-user-1, got %s", found.UserID)
		}
		if found.Role != models.RoleEditor {
			t.Errorf("Expected Role=editor, got %s", found.Role)
		}
		if !found.IsActive {
			t.Error("Expected IsActive=true")
		}
	})

	t.Run("returns error for expired role", func(t *testing.T) {
		// Create role with past expiration
		pastTime := time.Now().Add(-time.Hour)
		role := models.NewUserRole("expired-user", "expired", models.RoleEditor, "admin")
		role.ExpiresAt = &pastTime

		_, err := db.SetUserRole(ctx, role, "admin", "admin", "test")
		if err != nil {
			t.Fatalf("SetUserRole failed: %v", err)
		}

		_, err = db.GetUserRole(ctx, "expired-user")
		if !errors.Is(err, ErrRoleNotFound) {
			t.Errorf("Expected ErrRoleNotFound for expired role, got %v", err)
		}
	})

	t.Run("returns role with future expiration", func(t *testing.T) {
		futureTime := time.Now().Add(24 * time.Hour)
		role := models.NewUserRole("future-user", "future", models.RoleEditor, "admin")
		role.ExpiresAt = &futureTime

		_, err := db.SetUserRole(ctx, role, "admin", "admin", "test")
		if err != nil {
			t.Fatalf("SetUserRole failed: %v", err)
		}

		found, err := db.GetUserRole(ctx, "future-user")
		if err != nil {
			t.Fatalf("GetUserRole failed: %v", err)
		}

		if found.ExpiresAt == nil {
			t.Error("Expected ExpiresAt to be set")
		}
	})
}

// TestGetEffectiveRole tests the effective role retrieval with defaults
func TestGetEffectiveRole(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	t.Run("returns viewer as default for unknown user", func(t *testing.T) {
		role, err := db.GetEffectiveRole(ctx, "unknown-user")
		if err != nil {
			t.Fatalf("GetEffectiveRole failed: %v", err)
		}

		if role != models.RoleViewer {
			t.Errorf("Expected viewer as default, got %s", role)
		}
	})

	t.Run("returns assigned role for known user", func(t *testing.T) {
		userRole := models.NewUserRole("admin-user", "admin", models.RoleAdmin, "system")
		_, err := db.SetUserRole(ctx, userRole, "system", "system", "test")
		if err != nil {
			t.Fatalf("SetUserRole failed: %v", err)
		}

		role, err := db.GetEffectiveRole(ctx, "admin-user")
		if err != nil {
			t.Fatalf("GetEffectiveRole failed: %v", err)
		}

		if role != models.RoleAdmin {
			t.Errorf("Expected admin, got %s", role)
		}
	})

	t.Run("returns viewer for expired role", func(t *testing.T) {
		pastTime := time.Now().Add(-time.Hour)
		userRole := models.NewUserRole("effective-expired", "expired", models.RoleAdmin, "system")
		userRole.ExpiresAt = &pastTime

		_, err := db.SetUserRole(ctx, userRole, "system", "system", "test")
		if err != nil {
			t.Fatalf("SetUserRole failed: %v", err)
		}

		role, err := db.GetEffectiveRole(ctx, "effective-expired")
		if err != nil {
			t.Fatalf("GetEffectiveRole failed: %v", err)
		}

		if role != models.RoleViewer {
			t.Errorf("Expected viewer for expired role, got %s", role)
		}
	})
}

// TestSetUserRole tests creating and updating roles
func TestSetUserRole(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	t.Run("creates new role", func(t *testing.T) {
		role := models.NewUserRole("new-user", "newuser", models.RoleEditor, "admin")

		created, err := db.SetUserRole(ctx, role, "admin", "admin", "test creation")
		if err != nil {
			t.Fatalf("SetUserRole failed: %v", err)
		}

		if created.ID == 0 {
			t.Error("Expected ID to be set")
		}
		if created.UserID != "new-user" {
			t.Errorf("Expected UserID=new-user, got %s", created.UserID)
		}
		if created.Role != models.RoleEditor {
			t.Errorf("Expected Role=editor, got %s", created.Role)
		}
	})

	t.Run("updates existing role", func(t *testing.T) {
		// Create initial role
		role := models.NewUserRole("update-user", "updateuser", models.RoleViewer, "admin")
		created, err := db.SetUserRole(ctx, role, "admin", "admin", "initial")
		if err != nil {
			t.Fatalf("Initial SetUserRole failed: %v", err)
		}

		// Update to editor
		role.Role = models.RoleEditor
		updated, err := db.SetUserRole(ctx, role, "admin", "admin", "promoted")
		if err != nil {
			t.Fatalf("Update SetUserRole failed: %v", err)
		}

		if updated.ID != created.ID {
			t.Errorf("Expected same ID after update, got %d vs %d", created.ID, updated.ID)
		}
		if updated.Role != models.RoleEditor {
			t.Errorf("Expected Role=editor after update, got %s", updated.Role)
		}
	})

	t.Run("rejects invalid role", func(t *testing.T) {
		role := &models.UserRole{
			UserID:   "invalid-role-user",
			Username: "invalid",
			Role:     "superadmin", // Invalid role
		}

		_, err := db.SetUserRole(ctx, role, "admin", "admin", "test")
		if !errors.Is(err, ErrInvalidRole) {
			t.Errorf("Expected ErrInvalidRole, got %v", err)
		}
	})

	t.Run("sets expiration correctly", func(t *testing.T) {
		futureTime := time.Now().Add(7 * 24 * time.Hour) // 7 days
		role := models.NewUserRole("expiring-user", "expiring", models.RoleEditor, "admin")
		role.ExpiresAt = &futureTime

		created, err := db.SetUserRole(ctx, role, "admin", "admin", "temporary access")
		if err != nil {
			t.Fatalf("SetUserRole failed: %v", err)
		}

		if created.ExpiresAt == nil {
			t.Error("Expected ExpiresAt to be set")
		}
	})

	t.Run("creates audit log entry", func(t *testing.T) {
		role := models.NewUserRole("audit-user", "audituser", models.RoleEditor, "admin")

		_, err := db.SetUserRole(ctx, role, "admin", "adminuser", "audit test")
		if err != nil {
			t.Fatalf("SetUserRole failed: %v", err)
		}

		// Check audit log
		entries, err := db.GetRoleAuditLog(ctx, "audit-user", 10, 0)
		if err != nil {
			t.Fatalf("GetRoleAuditLog failed: %v", err)
		}

		if len(entries) == 0 {
			t.Error("Expected audit log entry to be created")
		}

		entry := entries[0]
		if entry.Action != models.AuditActionAssign {
			t.Errorf("Expected action=assign, got %s", entry.Action)
		}
		if entry.NewRole != models.RoleEditor {
			t.Errorf("Expected new_role=editor, got %s", entry.NewRole)
		}
		if entry.TargetUserID != "audit-user" {
			t.Errorf("Expected target_user_id=audit-user, got %s", entry.TargetUserID)
		}
	})
}

// TestDeleteUserRole tests soft-deleting roles
func TestDeleteUserRole(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	t.Run("soft deletes existing role", func(t *testing.T) {
		// Create role
		role := models.NewUserRole("delete-user", "deleteuser", models.RoleEditor, "admin")
		_, err := db.SetUserRole(ctx, role, "admin", "admin", "test")
		if err != nil {
			t.Fatalf("SetUserRole failed: %v", err)
		}

		// Delete role
		err = db.DeleteUserRole(ctx, "delete-user", "admin", "admin", "test deletion")
		if err != nil {
			t.Fatalf("DeleteUserRole failed: %v", err)
		}

		// Verify not found
		_, err = db.GetUserRole(ctx, "delete-user")
		if !errors.Is(err, ErrRoleNotFound) {
			t.Errorf("Expected ErrRoleNotFound after deletion, got %v", err)
		}
	})

	t.Run("returns error for nonexistent user", func(t *testing.T) {
		err := db.DeleteUserRole(ctx, "nonexistent-delete", "admin", "admin", "test")
		if err == nil {
			t.Error("Expected error when deleting nonexistent role")
		}
	})

	t.Run("creates audit log entry on deletion", func(t *testing.T) {
		// Create role
		role := models.NewUserRole("delete-audit-user", "deleteaudit", models.RoleAdmin, "admin")
		_, err := db.SetUserRole(ctx, role, "admin", "admin", "test")
		if err != nil {
			t.Fatalf("SetUserRole failed: %v", err)
		}

		// Delete role
		err = db.DeleteUserRole(ctx, "delete-audit-user", "admin", "adminuser", "revoked access")
		if err != nil {
			t.Fatalf("DeleteUserRole failed: %v", err)
		}

		// Check audit log
		entries, err := db.GetRoleAuditLog(ctx, "delete-audit-user", 10, 0)
		if err != nil {
			t.Fatalf("GetRoleAuditLog failed: %v", err)
		}

		// Should have at least one revoke entry
		found := false
		for _, entry := range entries {
			if entry.Action == models.AuditActionRevoke {
				found = true
				if entry.OldRole != models.RoleAdmin {
					t.Errorf("Expected old_role=admin, got %s", entry.OldRole)
				}
				break
			}
		}
		if !found {
			t.Error("Expected revoke audit entry")
		}
	})
}

// TestListUserRoles tests listing roles with filtering
func TestListUserRoles(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	// Create test data
	testRoles := []struct {
		userID   string
		username string
		role     string
	}{
		{"list-user-1", "user1", models.RoleViewer},
		{"list-user-2", "user2", models.RoleEditor},
		{"list-user-3", "user3", models.RoleAdmin},
		{"list-user-4", "user4", models.RoleViewer},
	}

	for _, tr := range testRoles {
		role := models.NewUserRole(tr.userID, tr.username, tr.role, "admin")
		_, err := db.SetUserRole(ctx, role, "admin", "admin", "test")
		if err != nil {
			t.Fatalf("SetUserRole failed: %v", err)
		}
	}

	t.Run("lists all roles", func(t *testing.T) {
		roles, err := db.ListUserRoles(ctx, false, "")
		if err != nil {
			t.Fatalf("ListUserRoles failed: %v", err)
		}

		if len(roles) < 4 {
			t.Errorf("Expected at least 4 roles, got %d", len(roles))
		}
	})

	t.Run("filters by active only", func(t *testing.T) {
		// Delete one role
		_ = db.DeleteUserRole(ctx, "list-user-1", "admin", "admin", "test")

		activeRoles, err := db.ListUserRoles(ctx, true, "")
		if err != nil {
			t.Fatalf("ListUserRoles failed: %v", err)
		}

		// Verify deleted role is not in active list
		for _, r := range activeRoles {
			if r.UserID == "list-user-1" {
				t.Error("Deleted role should not appear in active list")
			}
		}
	})

	t.Run("filters by role type", func(t *testing.T) {
		viewerRoles, err := db.ListUserRoles(ctx, true, models.RoleViewer)
		if err != nil {
			t.Fatalf("ListUserRoles failed: %v", err)
		}

		for _, r := range viewerRoles {
			if r.Role != models.RoleViewer {
				t.Errorf("Expected only viewer roles, got %s", r.Role)
			}
		}
	})
}

// TestGetRoleStats tests statistics retrieval
func TestGetRoleStats(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	t.Run("returns stats for empty database", func(t *testing.T) {
		stats, err := db.GetRoleStats(ctx)
		if err != nil {
			t.Fatalf("GetRoleStats failed: %v", err)
		}

		if stats.TotalUsers != 0 {
			t.Errorf("Expected 0 total users, got %d", stats.TotalUsers)
		}
		if stats.ActiveRoles != 0 {
			t.Errorf("Expected 0 active roles, got %d", stats.ActiveRoles)
		}
	})

	t.Run("returns correct stats after creating roles", func(t *testing.T) {
		// Create various roles
		roles := []struct {
			userID   string
			username string
			role     string
		}{
			{"stats-user-1", "user1", models.RoleViewer},
			{"stats-user-2", "user2", models.RoleViewer},
			{"stats-user-3", "user3", models.RoleEditor},
			{"stats-user-4", "user4", models.RoleAdmin},
		}

		for _, r := range roles {
			role := models.NewUserRole(r.userID, r.username, r.role, "admin")
			_, err := db.SetUserRole(ctx, role, "admin", "admin", "test")
			if err != nil {
				t.Fatalf("SetUserRole failed: %v", err)
			}
		}

		stats, err := db.GetRoleStats(ctx)
		if err != nil {
			t.Fatalf("GetRoleStats failed: %v", err)
		}

		if stats.TotalUsers < 4 {
			t.Errorf("Expected at least 4 total users, got %d", stats.TotalUsers)
		}
		if stats.ByRole[models.RoleViewer] < 2 {
			t.Errorf("Expected at least 2 viewers, got %d", stats.ByRole[models.RoleViewer])
		}
		if stats.ByRole[models.RoleEditor] < 1 {
			t.Errorf("Expected at least 1 editor, got %d", stats.ByRole[models.RoleEditor])
		}
		if stats.ByRole[models.RoleAdmin] < 1 {
			t.Errorf("Expected at least 1 admin, got %d", stats.ByRole[models.RoleAdmin])
		}
	})
}

// TestAuditRoleChange tests direct audit logging
func TestAuditRoleChange(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	t.Run("creates audit entry", func(t *testing.T) {
		entry := models.NewRoleAuditEntry("actor-1", "actor", models.AuditActionAssign, "target-1", "target")
		entry.NewRole = models.RoleEditor
		entry.Reason = "test reason"
		entry.IPAddress = "192.168.1.1"
		entry.UserAgent = "TestAgent/1.0"
		entry.SessionID = "session-123"

		err := db.AuditRoleChange(ctx, entry)
		if err != nil {
			t.Fatalf("AuditRoleChange failed: %v", err)
		}

		// Verify entry was created
		entries, err := db.GetRoleAuditLog(ctx, "target-1", 10, 0)
		if err != nil {
			t.Fatalf("GetRoleAuditLog failed: %v", err)
		}

		if len(entries) == 0 {
			t.Fatal("Expected at least one audit entry")
		}

		found := entries[0]
		if found.ActorID != "actor-1" {
			t.Errorf("Expected ActorID=actor-1, got %s", found.ActorID)
		}
		if found.Action != models.AuditActionAssign {
			t.Errorf("Expected Action=assign, got %s", found.Action)
		}
		if found.NewRole != models.RoleEditor {
			t.Errorf("Expected NewRole=editor, got %s", found.NewRole)
		}
		if found.IPAddress != "192.168.1.1" {
			t.Errorf("Expected IPAddress=192.168.1.1, got %s", found.IPAddress)
		}
	})

	t.Run("generates UUID if not provided", func(t *testing.T) {
		entry := &models.RoleAuditEntry{
			ActorID:      "actor-2",
			Action:       models.AuditActionRevoke,
			TargetUserID: "target-2",
		}

		err := db.AuditRoleChange(ctx, entry)
		if err != nil {
			t.Fatalf("AuditRoleChange failed: %v", err)
		}

		if entry.ID == uuid.Nil {
			t.Error("Expected UUID to be generated")
		}
	})
}

// TestGetRoleAuditLog tests audit log retrieval with filtering
func TestGetRoleAuditLog(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	// Create test audit entries
	for i := 0; i < 15; i++ {
		entry := models.NewRoleAuditEntry("actor", "actor", models.AuditActionAssign, "audit-log-user", "user")
		entry.NewRole = models.RoleEditor
		_ = db.AuditRoleChange(ctx, entry)
	}

	// Create entries for different user
	for i := 0; i < 5; i++ {
		entry := models.NewRoleAuditEntry("actor", "actor", models.AuditActionAssign, "other-user", "other")
		entry.NewRole = models.RoleViewer
		_ = db.AuditRoleChange(ctx, entry)
	}

	t.Run("retrieves all entries without filter", func(t *testing.T) {
		entries, err := db.GetRoleAuditLog(ctx, "", 0, 0)
		if err != nil {
			t.Fatalf("GetRoleAuditLog failed: %v", err)
		}

		if len(entries) < 20 {
			t.Errorf("Expected at least 20 entries, got %d", len(entries))
		}
	})

	t.Run("filters by user ID", func(t *testing.T) {
		entries, err := db.GetRoleAuditLog(ctx, "audit-log-user", 0, 0)
		if err != nil {
			t.Fatalf("GetRoleAuditLog failed: %v", err)
		}

		for _, e := range entries {
			if e.TargetUserID != "audit-log-user" {
				t.Errorf("Expected TargetUserID=audit-log-user, got %s", e.TargetUserID)
			}
		}
	})

	t.Run("applies limit", func(t *testing.T) {
		entries, err := db.GetRoleAuditLog(ctx, "", 5, 0)
		if err != nil {
			t.Fatalf("GetRoleAuditLog failed: %v", err)
		}

		if len(entries) > 5 {
			t.Errorf("Expected at most 5 entries, got %d", len(entries))
		}
	})

	t.Run("applies offset", func(t *testing.T) {
		// Get first 5
		first, _ := db.GetRoleAuditLog(ctx, "audit-log-user", 5, 0)
		// Get next 5
		second, _ := db.GetRoleAuditLog(ctx, "audit-log-user", 5, 5)

		if len(first) > 0 && len(second) > 0 {
			if first[0].ID == second[0].ID {
				t.Error("Expected different entries with offset")
			}
		}
	})

	t.Run("returns entries in descending timestamp order", func(t *testing.T) {
		entries, err := db.GetRoleAuditLog(ctx, "", 10, 0)
		if err != nil {
			t.Fatalf("GetRoleAuditLog failed: %v", err)
		}

		for i := 1; i < len(entries); i++ {
			if entries[i].Timestamp.After(entries[i-1].Timestamp) {
				t.Error("Expected entries in descending timestamp order")
			}
		}
	})
}

// TestGetAuditLogCount tests counting audit entries
func TestGetAuditLogCount(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	t.Run("returns 0 for empty log", func(t *testing.T) {
		count, err := db.GetAuditLogCount(ctx, "")
		if err != nil {
			t.Fatalf("GetAuditLogCount failed: %v", err)
		}

		if count != 0 {
			t.Errorf("Expected count=0 for empty log, got %d", count)
		}
	})

	t.Run("returns correct count", func(t *testing.T) {
		// Create 5 entries
		for i := 0; i < 5; i++ {
			entry := models.NewRoleAuditEntry("actor", "actor", models.AuditActionAssign, "count-user", "user")
			_ = db.AuditRoleChange(ctx, entry)
		}

		count, err := db.GetAuditLogCount(ctx, "count-user")
		if err != nil {
			t.Fatalf("GetAuditLogCount failed: %v", err)
		}

		if count != 5 {
			t.Errorf("Expected count=5, got %d", count)
		}
	})

	t.Run("filters by user ID", func(t *testing.T) {
		// Create entries for different user
		for i := 0; i < 3; i++ {
			entry := models.NewRoleAuditEntry("actor", "actor", models.AuditActionAssign, "count-other", "other")
			_ = db.AuditRoleChange(ctx, entry)
		}

		count, err := db.GetAuditLogCount(ctx, "count-other")
		if err != nil {
			t.Fatalf("GetAuditLogCount failed: %v", err)
		}

		if count != 3 {
			t.Errorf("Expected count=3, got %d", count)
		}
	})
}

// TestExpireRoles tests automatic role expiration
func TestExpireRoles(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	t.Run("expires past-due roles", func(t *testing.T) {
		// Create role with past expiration
		pastTime := time.Now().Add(-time.Hour)
		role := models.NewUserRole("expire-test-user", "expiretest", models.RoleEditor, "admin")
		role.ExpiresAt = &pastTime

		_, err := db.SetUserRole(ctx, role, "admin", "admin", "test")
		if err != nil {
			t.Fatalf("SetUserRole failed: %v", err)
		}

		// Run expiration
		count, err := db.ExpireRoles(ctx, "system")
		if err != nil {
			t.Fatalf("ExpireRoles failed: %v", err)
		}

		if count < 1 {
			t.Errorf("Expected at least 1 expired role, got %d", count)
		}

		// Verify role is now inactive
		_, err = db.GetUserRole(ctx, "expire-test-user")
		if !errors.Is(err, ErrRoleNotFound) {
			t.Errorf("Expected ErrRoleNotFound after expiration, got %v", err)
		}
	})

	t.Run("does not expire future roles", func(t *testing.T) {
		// Create role with future expiration
		futureTime := time.Now().Add(24 * time.Hour)
		role := models.NewUserRole("future-expire-user", "futureexpire", models.RoleEditor, "admin")
		role.ExpiresAt = &futureTime

		_, err := db.SetUserRole(ctx, role, "admin", "admin", "test")
		if err != nil {
			t.Fatalf("SetUserRole failed: %v", err)
		}

		// Run expiration
		_, _ = db.ExpireRoles(ctx, "system")

		// Verify role is still active
		found, err := db.GetUserRole(ctx, "future-expire-user")
		if err != nil {
			t.Fatalf("GetUserRole failed: %v", err)
		}

		if !found.IsActive {
			t.Error("Future role should still be active")
		}
	})

	t.Run("creates audit entry for expired roles", func(t *testing.T) {
		// Create role with past expiration
		pastTime := time.Now().Add(-2 * time.Hour)
		role := models.NewUserRole("expire-audit-user", "expireaudit", models.RoleAdmin, "admin")
		role.ExpiresAt = &pastTime

		_, err := db.SetUserRole(ctx, role, "admin", "admin", "test")
		if err != nil {
			t.Fatalf("SetUserRole failed: %v", err)
		}

		// Run expiration
		_, _ = db.ExpireRoles(ctx, "system")

		// Check for audit entry
		entries, err := db.GetRoleAuditLog(ctx, "expire-audit-user", 10, 0)
		if err != nil {
			t.Fatalf("GetRoleAuditLog failed: %v", err)
		}

		found := false
		for _, e := range entries {
			if e.Action == models.AuditActionExpire {
				found = true
				if e.ActorID != "system" {
					t.Errorf("Expected ActorID=system, got %s", e.ActorID)
				}
				break
			}
		}

		if !found {
			t.Error("Expected expire audit entry")
		}
	})
}

// TestIsUserAdmin tests the admin check helper
func TestIsUserAdmin(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	t.Run("returns false for unknown user", func(t *testing.T) {
		isAdmin, err := db.IsUserAdmin(ctx, "unknown-admin-check")
		if err != nil {
			t.Fatalf("IsUserAdmin failed: %v", err)
		}

		if isAdmin {
			t.Error("Expected false for unknown user")
		}
	})

	t.Run("returns true for admin user", func(t *testing.T) {
		role := models.NewUserRole("is-admin-user", "isadmin", models.RoleAdmin, "system")
		_, err := db.SetUserRole(ctx, role, "system", "system", "test")
		if err != nil {
			t.Fatalf("SetUserRole failed: %v", err)
		}

		isAdmin, err := db.IsUserAdmin(ctx, "is-admin-user")
		if err != nil {
			t.Fatalf("IsUserAdmin failed: %v", err)
		}

		if !isAdmin {
			t.Error("Expected true for admin user")
		}
	})

	t.Run("returns false for non-admin user", func(t *testing.T) {
		role := models.NewUserRole("not-admin-user", "notadmin", models.RoleEditor, "system")
		_, err := db.SetUserRole(ctx, role, "system", "system", "test")
		if err != nil {
			t.Fatalf("SetUserRole failed: %v", err)
		}

		isAdmin, err := db.IsUserAdmin(ctx, "not-admin-user")
		if err != nil {
			t.Fatalf("IsUserAdmin failed: %v", err)
		}

		if isAdmin {
			t.Error("Expected false for editor user")
		}
	})
}

// TestRBACConcurrency tests thread safety of RBAC operations
func TestRBACConcurrency(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	t.Run("concurrent role updates for same user", func(t *testing.T) {
		const numGoroutines = 10

		// Create initial role
		role := models.NewUserRole("concurrent-user", "concurrent", models.RoleViewer, "admin")
		_, _ = db.SetUserRole(ctx, role, "admin", "admin", "initial")

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()
				newRole := models.NewUserRole("concurrent-user", "concurrent", models.RoleEditor, "admin")
				_, err := db.SetUserRole(ctx, newRole, "admin", "admin", "update")
				if err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent update error: %v", err)
		}

		// Verify final state is consistent
		finalRole, err := db.GetUserRole(ctx, "concurrent-user")
		if err != nil {
			t.Fatalf("GetUserRole failed: %v", err)
		}
		if finalRole.Role != models.RoleEditor {
			t.Errorf("Expected final role=editor, got %s", finalRole.Role)
		}
	})

	t.Run("concurrent creates for different users", func(t *testing.T) {
		const numUsers = 20
		var wg sync.WaitGroup
		results := make(chan int64, numUsers)

		for i := 0; i < numUsers; i++ {
			wg.Add(1)
			go func(userNum int) {
				defer wg.Done()
				userID := string(rune('A' + userNum))
				role := models.NewUserRole("concurrent-create-"+userID, "user"+userID, models.RoleViewer, "admin")
				created, err := db.SetUserRole(ctx, role, "admin", "admin", "test")
				if err != nil {
					t.Errorf("Concurrent create failed: %v", err)
					return
				}
				results <- created.ID
			}(i)
		}

		wg.Wait()
		close(results)

		// All IDs should be unique
		seen := make(map[int64]bool)
		for id := range results {
			if seen[id] {
				t.Errorf("Duplicate ID: %d", id)
			}
			seen[id] = true
		}
	})
}

// TestRBACEdgeCases tests edge cases and special characters
func TestRBACEdgeCases(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	t.Run("handles special characters in user ID", func(t *testing.T) {
		specialIDs := []string{
			"user@domain.com",
			"user+tag@example.com",
			"user/with/slashes",
			"user with spaces",
		}

		for _, userID := range specialIDs {
			role := models.NewUserRole(userID, "user", models.RoleViewer, "admin")
			_, err := db.SetUserRole(ctx, role, "admin", "admin", "test")
			if err != nil {
				t.Errorf("Failed for user ID %q: %v", userID, err)
				continue
			}

			// Verify round-trip
			found, err := db.GetUserRole(ctx, userID)
			if err != nil {
				t.Errorf("Lookup failed for user ID %q: %v", userID, err)
				continue
			}

			if found.UserID != userID {
				t.Errorf("Round-trip failed for %q", userID)
			}
		}
	})

	t.Run("handles Unicode in username", func(t *testing.T) {
		role := models.NewUserRole("unicode-user", "Пользователь", models.RoleViewer, "admin")
		created, err := db.SetUserRole(ctx, role, "admin", "admin", "test")
		if err != nil {
			t.Fatalf("Failed with Unicode: %v", err)
		}

		if created.Username != "Пользователь" {
			t.Error("Unicode username not preserved")
		}
	})

	t.Run("handles metadata JSON", func(t *testing.T) {
		metadata := `{"department":"engineering","team":"backend"}`
		role := models.NewUserRole("metadata-user", "metadata", models.RoleEditor, "admin")
		role.Metadata = &metadata

		_, err := db.SetUserRole(ctx, role, "admin", "admin", "test")
		if err != nil {
			t.Fatalf("SetUserRole with metadata failed: %v", err)
		}

		found, err := db.GetUserRole(ctx, "metadata-user")
		if err != nil {
			t.Fatalf("GetUserRole failed: %v", err)
		}

		if found.Metadata == nil || *found.Metadata != metadata {
			t.Error("Metadata not preserved")
		}
	})

	t.Run("handles empty reason in audit", func(t *testing.T) {
		entry := models.NewRoleAuditEntry("actor", "actor", models.AuditActionAssign, "target", "target")
		// Reason is empty

		err := db.AuditRoleChange(ctx, entry)
		if err != nil {
			t.Fatalf("AuditRoleChange with empty reason failed: %v", err)
		}
	})
}

// TestModelHelpers tests the model helper functions
func TestModelHelpers(t *testing.T) {
	t.Run("IsValidRole validates correctly", func(t *testing.T) {
		validRoles := []string{models.RoleViewer, models.RoleEditor, models.RoleAdmin}
		for _, role := range validRoles {
			if !models.IsValidRole(role) {
				t.Errorf("Expected %s to be valid", role)
			}
		}

		invalidRoles := []string{"superadmin", "guest", "", "ADMIN"}
		for _, role := range invalidRoles {
			if models.IsValidRole(role) {
				t.Errorf("Expected %s to be invalid", role)
			}
		}
	})

	t.Run("UserRole.IsExpired works correctly", func(t *testing.T) {
		role := &models.UserRole{}

		// No expiration
		if role.IsExpired() {
			t.Error("Expected non-expired for nil ExpiresAt")
		}

		// Past expiration
		pastTime := time.Now().Add(-time.Hour)
		role.ExpiresAt = &pastTime
		if !role.IsExpired() {
			t.Error("Expected expired for past time")
		}

		// Future expiration
		futureTime := time.Now().Add(time.Hour)
		role.ExpiresAt = &futureTime
		if role.IsExpired() {
			t.Error("Expected non-expired for future time")
		}
	})

	t.Run("UserRole.IsEffective works correctly", func(t *testing.T) {
		// Active and not expired
		role := &models.UserRole{IsActive: true}
		if !role.IsEffective() {
			t.Error("Expected effective for active, non-expired role")
		}

		// Inactive
		role.IsActive = false
		if role.IsEffective() {
			t.Error("Expected not effective for inactive role")
		}

		// Active but expired
		role.IsActive = true
		pastTime := time.Now().Add(-time.Hour)
		role.ExpiresAt = &pastTime
		if role.IsEffective() {
			t.Error("Expected not effective for expired role")
		}
	})

	t.Run("NewUserRole creates valid role", func(t *testing.T) {
		role := models.NewUserRole("user-123", "testuser", models.RoleEditor, "admin-456")

		if role.UserID != "user-123" {
			t.Errorf("Expected UserID=user-123, got %s", role.UserID)
		}
		if role.Username != "testuser" {
			t.Errorf("Expected Username=testuser, got %s", role.Username)
		}
		if role.Role != models.RoleEditor {
			t.Errorf("Expected Role=editor, got %s", role.Role)
		}
		if role.AssignedBy != "admin-456" {
			t.Errorf("Expected AssignedBy=admin-456, got %s", role.AssignedBy)
		}
		if !role.IsActive {
			t.Error("Expected IsActive=true")
		}
		if role.AssignedAt.IsZero() {
			t.Error("Expected AssignedAt to be set")
		}
	})

	t.Run("NewRoleAuditEntry creates valid entry", func(t *testing.T) {
		entry := models.NewRoleAuditEntry("actor-1", "actor", models.AuditActionAssign, "target-1", "target")

		if entry.ID == uuid.Nil {
			t.Error("Expected UUID to be generated")
		}
		if entry.ActorID != "actor-1" {
			t.Errorf("Expected ActorID=actor-1, got %s", entry.ActorID)
		}
		if entry.Action != models.AuditActionAssign {
			t.Errorf("Expected Action=assign, got %s", entry.Action)
		}
		if entry.TargetUserID != "target-1" {
			t.Errorf("Expected TargetUserID=target-1, got %s", entry.TargetUserID)
		}
		if entry.Timestamp.IsZero() {
			t.Error("Expected Timestamp to be set")
		}
	})
}
