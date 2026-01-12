// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package authz

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/models"
)

// mockRoleProvider implements RoleProvider for testing.
type mockRoleProvider struct {
	mu           sync.Mutex
	roles        map[string]*models.UserRole
	auditEntries []*models.RoleAuditEntry
	getError     error
	setError     error
	deleteError  error
	auditError   error
}

func newMockRoleProvider() *mockRoleProvider {
	return &mockRoleProvider{
		roles:        make(map[string]*models.UserRole),
		auditEntries: make([]*models.RoleAuditEntry, 0),
	}
}

func (m *mockRoleProvider) GetUserRole(_ context.Context, userID string) (*models.UserRole, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.getError != nil {
		return nil, m.getError
	}

	role, ok := m.roles[userID]
	if !ok {
		return nil, database.ErrRoleNotFound
	}
	return role, nil
}

func (m *mockRoleProvider) GetEffectiveRole(ctx context.Context, userID string) (string, error) {
	role, err := m.GetUserRole(ctx, userID)
	if err != nil {
		if errors.Is(err, database.ErrRoleNotFound) {
			return models.RoleViewer, nil
		}
		return "", err
	}
	return role.Role, nil
}

func (m *mockRoleProvider) SetUserRole(_ context.Context, role *models.UserRole, _, _, _ string) (*models.UserRole, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.setError != nil {
		return nil, m.setError
	}

	m.roles[role.UserID] = role
	return role, nil
}

func (m *mockRoleProvider) DeleteUserRole(_ context.Context, userID, _, _, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.deleteError != nil {
		return m.deleteError
	}

	if _, ok := m.roles[userID]; !ok {
		return database.ErrRoleNotFound
	}

	delete(m.roles, userID)
	return nil
}

func (m *mockRoleProvider) AuditRoleChange(_ context.Context, entry *models.RoleAuditEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.auditError != nil {
		return m.auditError
	}

	m.auditEntries = append(m.auditEntries, entry)
	return nil
}

func (m *mockRoleProvider) IsUserAdmin(ctx context.Context, userID string) (bool, error) {
	role, err := m.GetEffectiveRole(ctx, userID)
	if err != nil {
		return false, err
	}
	return role == models.RoleAdmin, nil
}

func (m *mockRoleProvider) setRole(userID, username, role string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.roles[userID] = &models.UserRole{
		UserID:     userID,
		Username:   username,
		Role:       role,
		IsActive:   true,
		AssignedAt: time.Now(),
	}
}

func setupTestService(t *testing.T) (*Service, *mockRoleProvider, *Enforcer) {
	t.Helper()

	// Create enforcer with default config
	ctx := context.Background()
	enforcer, err := NewEnforcer(ctx, nil)
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}

	// Create mock provider
	mockDB := newMockRoleProvider()

	// Create service with caching disabled for predictable tests
	config := &ServiceConfig{
		DefaultRole:  models.RoleViewer,
		CacheEnabled: false,
		AuditEnabled: true,
	}

	service, err := NewService(enforcer, mockDB, config)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	return service, mockDB, enforcer
}

// TestNewService tests service creation.
func TestNewService(t *testing.T) {
	ctx := context.Background()
	enforcer, _ := NewEnforcer(ctx, nil)
	mockDB := newMockRoleProvider()

	tests := []struct {
		name     string
		enforcer *Enforcer
		db       RoleProvider
		config   *ServiceConfig
		wantErr  bool
	}{
		{
			name:     "success with defaults",
			enforcer: enforcer,
			db:       mockDB,
			config:   nil,
			wantErr:  false,
		},
		{
			name:     "success with custom config",
			enforcer: enforcer,
			db:       mockDB,
			config:   DefaultServiceConfig(),
			wantErr:  false,
		},
		{
			name:     "nil enforcer",
			enforcer: nil,
			db:       mockDB,
			config:   nil,
			wantErr:  true,
		},
		{
			name:     "nil database",
			enforcer: enforcer,
			db:       nil,
			config:   nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewService(tt.enforcer, tt.db, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && service == nil {
				t.Error("NewService() returned nil service without error")
			}
			if service != nil {
				service.Close()
			}
		})
	}
}

// TestGetEffectiveRole tests role lookup.
func TestGetEffectiveRole(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer service.Close()

	ctx := context.Background()

	t.Run("returns viewer for empty userID", func(t *testing.T) {
		role, err := service.GetEffectiveRole(ctx, "")
		if err != nil {
			t.Errorf("GetEffectiveRole() error = %v", err)
		}
		if role != models.RoleViewer {
			t.Errorf("GetEffectiveRole() = %v, want %v", role, models.RoleViewer)
		}
	})

	t.Run("returns viewer for unknown user", func(t *testing.T) {
		role, err := service.GetEffectiveRole(ctx, "unknown")
		if err != nil {
			t.Errorf("GetEffectiveRole() error = %v", err)
		}
		if role != models.RoleViewer {
			t.Errorf("GetEffectiveRole() = %v, want %v", role, models.RoleViewer)
		}
	})

	t.Run("returns assigned role", func(t *testing.T) {
		mockDB.setRole("admin-user", "Admin", models.RoleAdmin)

		role, err := service.GetEffectiveRole(ctx, "admin-user")
		if err != nil {
			t.Errorf("GetEffectiveRole() error = %v", err)
		}
		if role != models.RoleAdmin {
			t.Errorf("GetEffectiveRole() = %v, want %v", role, models.RoleAdmin)
		}
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		mockDB.getError = errors.New("database error")
		defer func() { mockDB.getError = nil }()

		_, err := service.GetEffectiveRole(ctx, "any-user")
		if err == nil {
			t.Error("GetEffectiveRole() expected error, got nil")
		}
	})
}

// TestCanAccess tests policy enforcement.
func TestCanAccess(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer service.Close()

	ctx := context.Background()

	t.Run("nil subject uses default role", func(t *testing.T) {
		// Viewers can read /api/*
		allowed, err := service.CanAccess(ctx, nil, "/api/health", "read")
		if err != nil {
			t.Errorf("CanAccess() error = %v", err)
		}
		if !allowed {
			t.Error("CanAccess() = false, want true for viewer reading")
		}
	})

	t.Run("admin can access admin endpoints", func(t *testing.T) {
		mockDB.setRole("admin-id", "Admin", models.RoleAdmin)

		subject := &auth.AuthSubject{
			ID:       "admin-id",
			Username: "Admin",
		}

		allowed, err := service.CanAccess(ctx, subject, "/api/admin/users", "delete")
		if err != nil {
			t.Errorf("CanAccess() error = %v", err)
		}
		if !allowed {
			t.Error("CanAccess() = false, want true for admin")
		}
	})

	t.Run("viewer denied write access", func(t *testing.T) {
		subject := &auth.AuthSubject{
			ID:       "viewer-id",
			Username: "Viewer",
			Roles:    []string{models.RoleViewer},
		}

		allowed, err := service.CanAccess(ctx, subject, "/api/admin/config", "write")
		if err != nil {
			t.Errorf("CanAccess() error = %v", err)
		}
		if allowed {
			t.Error("CanAccess() = true, want false for viewer writing to admin")
		}
	})

	t.Run("subject with token role", func(t *testing.T) {
		subject := &auth.AuthSubject{
			ID:       "token-admin",
			Username: "TokenAdmin",
			Roles:    []string{models.RoleAdmin},
		}

		allowed, err := service.CanAccess(ctx, subject, "/api/admin/users", "*")
		if err != nil {
			t.Errorf("CanAccess() error = %v", err)
		}
		if !allowed {
			t.Error("CanAccess() = false, want true for admin with token role")
		}
	})

	t.Run("database role combines with token roles", func(t *testing.T) {
		mockDB.setRole("combo-user", "Combo", models.RoleAdmin)

		subject := &auth.AuthSubject{
			ID:       "combo-user",
			Username: "Combo",
			Roles:    []string{models.RoleViewer}, // Token says viewer
		}

		// Database says admin, should be allowed
		allowed, err := service.CanAccess(ctx, subject, "/api/admin/config", "*")
		if err != nil {
			t.Errorf("CanAccess() error = %v", err)
		}
		if !allowed {
			t.Error("CanAccess() = false, want true (database role is admin)")
		}
	})
}

// TestCanAccessOwnData tests user-scoped access.
func TestCanAccessOwnData(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer service.Close()

	ctx := context.Background()

	t.Run("nil subject returns error", func(t *testing.T) {
		_, err := service.CanAccessOwnData(ctx, nil, "any-user")
		if !errors.Is(err, ErrNilSubject) {
			t.Errorf("CanAccessOwnData() error = %v, want ErrNilSubject", err)
		}
	})

	t.Run("user can access own data", func(t *testing.T) {
		subject := &auth.AuthSubject{
			ID:       "user-123",
			Username: "User123",
		}

		allowed, err := service.CanAccessOwnData(ctx, subject, "user-123")
		if err != nil {
			t.Errorf("CanAccessOwnData() error = %v", err)
		}
		if !allowed {
			t.Error("CanAccessOwnData() = false, want true for own data")
		}
	})

	t.Run("viewer cannot access other user data", func(t *testing.T) {
		subject := &auth.AuthSubject{
			ID:       "user-123",
			Username: "User123",
		}

		allowed, err := service.CanAccessOwnData(ctx, subject, "other-user")
		if err != nil {
			t.Errorf("CanAccessOwnData() error = %v", err)
		}
		if allowed {
			t.Error("CanAccessOwnData() = true, want false for other user's data")
		}
	})

	t.Run("admin can access any user data", func(t *testing.T) {
		mockDB.setRole("admin-id", "Admin", models.RoleAdmin)

		subject := &auth.AuthSubject{
			ID:       "admin-id",
			Username: "Admin",
		}

		allowed, err := service.CanAccessOwnData(ctx, subject, "other-user")
		if err != nil {
			t.Errorf("CanAccessOwnData() error = %v", err)
		}
		if !allowed {
			t.Error("CanAccessOwnData() = false, want true for admin")
		}
	})
}

// TestIsAdmin tests admin check.
func TestIsAdmin(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer service.Close()

	ctx := context.Background()

	t.Run("nil subject returns false", func(t *testing.T) {
		isAdmin, err := service.IsAdmin(ctx, nil)
		if err != nil {
			t.Errorf("IsAdmin() error = %v", err)
		}
		if isAdmin {
			t.Error("IsAdmin() = true, want false for nil subject")
		}
	})

	t.Run("subject with admin token role", func(t *testing.T) {
		subject := &auth.AuthSubject{
			ID:       "token-admin",
			Username: "TokenAdmin",
			Roles:    []string{models.RoleAdmin},
		}

		isAdmin, err := service.IsAdmin(ctx, subject)
		if err != nil {
			t.Errorf("IsAdmin() error = %v", err)
		}
		if !isAdmin {
			t.Error("IsAdmin() = false, want true")
		}
	})

	t.Run("subject with admin database role", func(t *testing.T) {
		mockDB.setRole("db-admin", "DBAdmin", models.RoleAdmin)

		subject := &auth.AuthSubject{
			ID:       "db-admin",
			Username: "DBAdmin",
		}

		isAdmin, err := service.IsAdmin(ctx, subject)
		if err != nil {
			t.Errorf("IsAdmin() error = %v", err)
		}
		if !isAdmin {
			t.Error("IsAdmin() = false, want true")
		}
	})

	t.Run("regular user is not admin", func(t *testing.T) {
		subject := &auth.AuthSubject{
			ID:       "regular-user",
			Username: "Regular",
		}

		isAdmin, err := service.IsAdmin(ctx, subject)
		if err != nil {
			t.Errorf("IsAdmin() error = %v", err)
		}
		if isAdmin {
			t.Error("IsAdmin() = true, want false")
		}
	})
}

// TestIsEditor tests editor check.
func TestIsEditor(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer service.Close()

	ctx := context.Background()

	t.Run("nil subject returns false", func(t *testing.T) {
		isEditor, err := service.IsEditor(ctx, nil)
		if err != nil {
			t.Errorf("IsEditor() error = %v", err)
		}
		if isEditor {
			t.Error("IsEditor() = true, want false for nil subject")
		}
	})

	t.Run("editor role returns true", func(t *testing.T) {
		subject := &auth.AuthSubject{
			ID:       "editor-user",
			Username: "Editor",
			Roles:    []string{models.RoleEditor},
		}

		isEditor, err := service.IsEditor(ctx, subject)
		if err != nil {
			t.Errorf("IsEditor() error = %v", err)
		}
		if !isEditor {
			t.Error("IsEditor() = false, want true")
		}
	})

	t.Run("admin role also returns true", func(t *testing.T) {
		subject := &auth.AuthSubject{
			ID:       "admin-user",
			Username: "Admin",
			Roles:    []string{models.RoleAdmin},
		}

		isEditor, err := service.IsEditor(ctx, subject)
		if err != nil {
			t.Errorf("IsEditor() error = %v", err)
		}
		if !isEditor {
			t.Error("IsEditor() = false, want true for admin")
		}
	})

	t.Run("database editor role", func(t *testing.T) {
		mockDB.setRole("db-editor", "DBEditor", models.RoleEditor)

		subject := &auth.AuthSubject{
			ID:       "db-editor",
			Username: "DBEditor",
		}

		isEditor, err := service.IsEditor(ctx, subject)
		if err != nil {
			t.Errorf("IsEditor() error = %v", err)
		}
		if !isEditor {
			t.Error("IsEditor() = false, want true")
		}
	})

	t.Run("viewer is not editor", func(t *testing.T) {
		subject := &auth.AuthSubject{
			ID:       "viewer-user",
			Username: "Viewer",
		}

		isEditor, err := service.IsEditor(ctx, subject)
		if err != nil {
			t.Errorf("IsEditor() error = %v", err)
		}
		if isEditor {
			t.Error("IsEditor() = true, want false")
		}
	})
}

// TestAssignRole tests role assignment.
func TestAssignRole(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer service.Close()

	ctx := context.Background()

	adminSubject := &auth.AuthSubject{
		ID:       "admin-id",
		Username: "Admin",
		Roles:    []string{models.RoleAdmin},
	}

	t.Run("nil actor returns error", func(t *testing.T) {
		err := service.AssignRole(ctx, nil, "target", "Target", models.RoleEditor, "")
		if !errors.Is(err, ErrNilSubject) {
			t.Errorf("AssignRole() error = %v, want ErrNilSubject", err)
		}
	})

	t.Run("invalid role returns error", func(t *testing.T) {
		err := service.AssignRole(ctx, adminSubject, "target", "Target", "invalid-role", "")
		if !errors.Is(err, ErrInvalidRole) {
			t.Errorf("AssignRole() error = %v, want ErrInvalidRole", err)
		}
	})

	t.Run("self role change denied", func(t *testing.T) {
		err := service.AssignRole(ctx, adminSubject, adminSubject.ID, adminSubject.Username, models.RoleEditor, "")
		if !errors.Is(err, ErrSelfRoleChange) {
			t.Errorf("AssignRole() error = %v, want ErrSelfRoleChange", err)
		}
	})

	t.Run("non-admin cannot assign roles", func(t *testing.T) {
		viewer := &auth.AuthSubject{
			ID:       "viewer-id",
			Username: "Viewer",
		}

		err := service.AssignRole(ctx, viewer, "target", "Target", models.RoleEditor, "")
		if !errors.Is(err, ErrAdminRequired) {
			t.Errorf("AssignRole() error = %v, want ErrAdminRequired", err)
		}
	})

	t.Run("admin can assign role", func(t *testing.T) {
		err := service.AssignRole(ctx, adminSubject, "new-editor", "NewEditor", models.RoleEditor, "promotion")
		if err != nil {
			t.Errorf("AssignRole() error = %v", err)
		}

		// Verify role was set
		role, _ := service.GetEffectiveRole(ctx, "new-editor")
		if role != models.RoleEditor {
			t.Errorf("GetEffectiveRole() = %v, want %v", role, models.RoleEditor)
		}
	})

	t.Run("database error propagates", func(t *testing.T) {
		mockDB.setError = errors.New("db error")
		defer func() { mockDB.setError = nil }()

		err := service.AssignRole(ctx, adminSubject, "target", "Target", models.RoleEditor, "")
		if err == nil {
			t.Error("AssignRole() expected error, got nil")
		}
	})
}

// TestRevokeRole tests role revocation.
func TestRevokeRole(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer service.Close()

	ctx := context.Background()

	adminSubject := &auth.AuthSubject{
		ID:       "admin-id",
		Username: "Admin",
		Roles:    []string{models.RoleAdmin},
	}

	t.Run("nil actor returns error", func(t *testing.T) {
		err := service.RevokeRole(ctx, nil, "target", "")
		if !errors.Is(err, ErrNilSubject) {
			t.Errorf("RevokeRole() error = %v, want ErrNilSubject", err)
		}
	})

	t.Run("self role revoke denied", func(t *testing.T) {
		err := service.RevokeRole(ctx, adminSubject, adminSubject.ID, "")
		if !errors.Is(err, ErrSelfRoleChange) {
			t.Errorf("RevokeRole() error = %v, want ErrSelfRoleChange", err)
		}
	})

	t.Run("non-admin cannot revoke roles", func(t *testing.T) {
		viewer := &auth.AuthSubject{
			ID:       "viewer-id",
			Username: "Viewer",
		}

		err := service.RevokeRole(ctx, viewer, "target", "")
		if !errors.Is(err, ErrAdminRequired) {
			t.Errorf("RevokeRole() error = %v, want ErrAdminRequired", err)
		}
	})

	t.Run("admin can revoke role", func(t *testing.T) {
		mockDB.setRole("editor-to-revoke", "Editor", models.RoleEditor)

		err := service.RevokeRole(ctx, adminSubject, "editor-to-revoke", "demotion")
		if err != nil {
			t.Errorf("RevokeRole() error = %v", err)
		}

		// Verify role reverted to default
		role, _ := service.GetEffectiveRole(ctx, "editor-to-revoke")
		if role != models.RoleViewer {
			t.Errorf("GetEffectiveRole() = %v, want %v (default)", role, models.RoleViewer)
		}
	})

	t.Run("revoke non-existent role succeeds", func(t *testing.T) {
		err := service.RevokeRole(ctx, adminSubject, "never-had-role", "cleanup")
		if err != nil {
			t.Errorf("RevokeRole() error = %v, want nil for non-existent role", err)
		}
	})

	t.Run("database error propagates", func(t *testing.T) {
		mockDB.setRole("has-role", "HasRole", models.RoleEditor)
		mockDB.deleteError = errors.New("db error")
		defer func() { mockDB.deleteError = nil }()

		err := service.RevokeRole(ctx, adminSubject, "has-role", "")
		if err == nil {
			t.Error("RevokeRole() expected error, got nil")
		}
	})
}

// TestRequireAdmin tests admin requirement check.
func TestRequireAdmin(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer service.Close()

	ctx := context.Background()

	t.Run("nil subject", func(t *testing.T) {
		err := service.RequireAdmin(ctx, nil)
		if !errors.Is(err, ErrNilSubject) {
			t.Errorf("RequireAdmin() error = %v, want ErrNilSubject", err)
		}
	})

	t.Run("admin passes", func(t *testing.T) {
		mockDB.setRole("admin-id", "Admin", models.RoleAdmin)
		subject := &auth.AuthSubject{ID: "admin-id", Username: "Admin"}

		err := service.RequireAdmin(ctx, subject)
		if err != nil {
			t.Errorf("RequireAdmin() error = %v, want nil", err)
		}
	})

	t.Run("viewer fails", func(t *testing.T) {
		subject := &auth.AuthSubject{ID: "viewer-id", Username: "Viewer"}

		err := service.RequireAdmin(ctx, subject)
		if !errors.Is(err, ErrAdminRequired) {
			t.Errorf("RequireAdmin() error = %v, want ErrAdminRequired", err)
		}
	})
}

// TestRequireEditor tests editor requirement check.
func TestRequireEditor(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer service.Close()

	ctx := context.Background()

	t.Run("nil subject", func(t *testing.T) {
		err := service.RequireEditor(ctx, nil)
		if !errors.Is(err, ErrNilSubject) {
			t.Errorf("RequireEditor() error = %v, want ErrNilSubject", err)
		}
	})

	t.Run("editor passes", func(t *testing.T) {
		mockDB.setRole("editor-id", "Editor", models.RoleEditor)
		subject := &auth.AuthSubject{ID: "editor-id", Username: "Editor"}

		err := service.RequireEditor(ctx, subject)
		if err != nil {
			t.Errorf("RequireEditor() error = %v, want nil", err)
		}
	})

	t.Run("admin passes", func(t *testing.T) {
		subject := &auth.AuthSubject{ID: "admin-id", Username: "Admin", Roles: []string{models.RoleAdmin}}

		err := service.RequireEditor(ctx, subject)
		if err != nil {
			t.Errorf("RequireEditor() error = %v, want nil for admin", err)
		}
	})

	t.Run("viewer fails", func(t *testing.T) {
		subject := &auth.AuthSubject{ID: "viewer-id", Username: "Viewer"}

		err := service.RequireEditor(ctx, subject)
		if !errors.Is(err, ErrNotAuthorized) {
			t.Errorf("RequireEditor() error = %v, want ErrNotAuthorized", err)
		}
	})
}

// TestRequireAccessToUser tests user data access check.
func TestRequireAccessToUser(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer service.Close()

	ctx := context.Background()

	t.Run("user accessing own data", func(t *testing.T) {
		subject := &auth.AuthSubject{ID: "user-id", Username: "User"}

		err := service.RequireAccessToUser(ctx, subject, "user-id")
		if err != nil {
			t.Errorf("RequireAccessToUser() error = %v, want nil", err)
		}
	})

	t.Run("user accessing other data", func(t *testing.T) {
		subject := &auth.AuthSubject{ID: "user-id", Username: "User"}

		err := service.RequireAccessToUser(ctx, subject, "other-id")
		if !errors.Is(err, ErrNotAuthorized) {
			t.Errorf("RequireAccessToUser() error = %v, want ErrNotAuthorized", err)
		}
	})

	t.Run("admin accessing any data", func(t *testing.T) {
		mockDB.setRole("admin-id", "Admin", models.RoleAdmin)
		subject := &auth.AuthSubject{ID: "admin-id", Username: "Admin"}

		err := service.RequireAccessToUser(ctx, subject, "any-user")
		if err != nil {
			t.Errorf("RequireAccessToUser() error = %v, want nil for admin", err)
		}
	})
}

// TestGetUserRoleInfo tests role info retrieval.
func TestGetUserRoleInfo(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer service.Close()

	ctx := context.Background()

	t.Run("nil subject", func(t *testing.T) {
		_, err := service.GetUserRoleInfo(ctx, nil, "target")
		if !errors.Is(err, ErrNilSubject) {
			t.Errorf("GetUserRoleInfo() error = %v, want ErrNilSubject", err)
		}
	})

	t.Run("user can see own role", func(t *testing.T) {
		mockDB.setRole("user-id", "User", models.RoleEditor)
		subject := &auth.AuthSubject{ID: "user-id", Username: "User"}

		role, err := service.GetUserRoleInfo(ctx, subject, "user-id")
		if err != nil {
			t.Errorf("GetUserRoleInfo() error = %v", err)
		}
		if role == nil {
			t.Error("GetUserRoleInfo() = nil, want role")
		}
		if role.Role != models.RoleEditor {
			t.Errorf("GetUserRoleInfo().Role = %v, want %v", role.Role, models.RoleEditor)
		}
	})

	t.Run("viewer cannot see other roles", func(t *testing.T) {
		mockDB.setRole("other-id", "Other", models.RoleEditor)
		subject := &auth.AuthSubject{ID: "viewer-id", Username: "Viewer"}

		_, err := service.GetUserRoleInfo(ctx, subject, "other-id")
		if !errors.Is(err, ErrAdminRequired) {
			t.Errorf("GetUserRoleInfo() error = %v, want ErrAdminRequired", err)
		}
	})

	t.Run("admin can see any role", func(t *testing.T) {
		mockDB.setRole("admin-id", "Admin", models.RoleAdmin)
		mockDB.setRole("other-id", "Other", models.RoleEditor)
		subject := &auth.AuthSubject{ID: "admin-id", Username: "Admin"}

		role, err := service.GetUserRoleInfo(ctx, subject, "other-id")
		if err != nil {
			t.Errorf("GetUserRoleInfo() error = %v", err)
		}
		if role == nil || role.Role != models.RoleEditor {
			t.Errorf("GetUserRoleInfo() unexpected role: %v", role)
		}
	})

	t.Run("returns nil for user with default role", func(t *testing.T) {
		mockDB.setRole("admin-id", "Admin", models.RoleAdmin)
		subject := &auth.AuthSubject{ID: "admin-id", Username: "Admin"}

		role, err := service.GetUserRoleInfo(ctx, subject, "no-explicit-role")
		if err != nil {
			t.Errorf("GetUserRoleInfo() error = %v", err)
		}
		if role != nil {
			t.Errorf("GetUserRoleInfo() = %v, want nil for default role", role)
		}
	})
}

// TestUpdateRole tests role update.
func TestUpdateRole(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer service.Close()

	ctx := context.Background()

	adminSubject := &auth.AuthSubject{
		ID:       "admin-id",
		Username: "Admin",
		Roles:    []string{models.RoleAdmin},
	}

	t.Run("update role calls assign role", func(t *testing.T) {
		mockDB.setRole("target-id", "Target", models.RoleViewer)

		err := service.UpdateRole(ctx, adminSubject, "target-id", "Target", models.RoleEditor, "update")
		if err != nil {
			t.Errorf("UpdateRole() error = %v", err)
		}

		role, _ := service.GetEffectiveRole(ctx, "target-id")
		if role != models.RoleEditor {
			t.Errorf("GetEffectiveRole() = %v, want %v", role, models.RoleEditor)
		}
	})
}

// TestGetEnforcer tests enforcer accessor.
func TestGetEnforcer(t *testing.T) {
	service, _, enforcer := setupTestService(t)
	defer service.Close()

	got := service.GetEnforcer()
	if got != enforcer {
		t.Error("GetEnforcer() returned different enforcer")
	}
}

// TestCaching tests role caching behavior.
func TestCaching(t *testing.T) {
	ctx := context.Background()
	enforcer, _ := NewEnforcer(ctx, nil)
	mockDB := newMockRoleProvider()

	// Enable caching with short TTL
	config := &ServiceConfig{
		DefaultRole:  models.RoleViewer,
		CacheEnabled: true,
		CacheTTL:     100 * time.Millisecond,
		AuditEnabled: false,
	}

	service, err := NewService(enforcer, mockDB, config)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close()

	t.Run("caches role lookups", func(t *testing.T) {
		mockDB.setRole("cached-user", "Cached", models.RoleEditor)

		// First lookup - from database
		role1, _ := service.GetEffectiveRole(ctx, "cached-user")
		if role1 != models.RoleEditor {
			t.Errorf("First lookup = %v, want %v", role1, models.RoleEditor)
		}

		// Simulate database change (cache doesn't know)
		mockDB.setRole("cached-user", "Cached", models.RoleAdmin)

		// Second lookup - should be cached
		role2, _ := service.GetEffectiveRole(ctx, "cached-user")
		if role2 != models.RoleEditor {
			t.Errorf("Cached lookup = %v, want %v (cached)", role2, models.RoleEditor)
		}

		// Wait for cache expiry
		time.Sleep(150 * time.Millisecond)

		// Third lookup - should refresh from database
		role3, _ := service.GetEffectiveRole(ctx, "cached-user")
		if role3 != models.RoleAdmin {
			t.Errorf("Refreshed lookup = %v, want %v", role3, models.RoleAdmin)
		}
	})

	t.Run("invalidates cache on role change", func(t *testing.T) {
		mockDB.setRole("change-user", "Change", models.RoleViewer)

		// Cache the role
		service.GetEffectiveRole(ctx, "change-user")

		// Assign new role (should invalidate cache)
		admin := &auth.AuthSubject{ID: "admin-id", Username: "Admin", Roles: []string{models.RoleAdmin}}
		service.AssignRole(ctx, admin, "change-user", "Change", models.RoleEditor, "test")

		// Should get new role immediately
		role, _ := service.GetEffectiveRole(ctx, "change-user")
		if role != models.RoleEditor {
			t.Errorf("After assignment = %v, want %v", role, models.RoleEditor)
		}
	})
}

// TestDefaultServiceConfig tests default config values.
func TestDefaultServiceConfig(t *testing.T) {
	config := DefaultServiceConfig()

	if config.DefaultRole != models.RoleViewer {
		t.Errorf("DefaultRole = %v, want %v", config.DefaultRole, models.RoleViewer)
	}
	if !config.CacheEnabled {
		t.Error("CacheEnabled = false, want true")
	}
	if config.CacheTTL != 5*time.Minute {
		t.Errorf("CacheTTL = %v, want %v", config.CacheTTL, 5*time.Minute)
	}
	if !config.AuditEnabled {
		t.Error("AuditEnabled = false, want true")
	}
}

// TestServiceClose tests service cleanup.
func TestServiceClose(t *testing.T) {
	ctx := context.Background()
	enforcer, _ := NewEnforcer(ctx, nil)
	mockDB := newMockRoleProvider()

	config := &ServiceConfig{
		DefaultRole:  models.RoleViewer,
		CacheEnabled: true,
		CacheTTL:     1 * time.Second,
	}

	service, err := NewService(enforcer, mockDB, config)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	// Close should be safe to call multiple times
	service.Close()
	service.Close()
}

// TestConcurrentAccess tests thread safety.
func TestConcurrentAccess(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer service.Close()

	ctx := context.Background()

	// Set up initial roles
	mockDB.setRole("admin-id", "Admin", models.RoleAdmin)

	adminSubject := &auth.AuthSubject{
		ID:       "admin-id",
		Username: "Admin",
		Roles:    []string{models.RoleAdmin},
	}

	var wg sync.WaitGroup
	numGoroutines := 10

	t.Run("concurrent role lookups", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					_, err := service.GetEffectiveRole(ctx, "admin-id")
					if err != nil {
						t.Errorf("GetEffectiveRole() error = %v", err)
					}
				}
			}()
		}
		wg.Wait()
	})

	t.Run("concurrent role assignments", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				userID := "concurrent-user"
				role := models.RoleEditor
				if idx%2 == 0 {
					role = models.RoleViewer
				}
				_ = service.AssignRole(ctx, adminSubject, userID, "ConcurrentUser", role, "test")
			}(i)
		}
		wg.Wait()
	})
}
