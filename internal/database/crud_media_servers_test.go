// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
)

func setupTestDBForMediaServers(t *testing.T) *DB {
	t.Helper()

	// CRITICAL: Acquire semaphore to prevent concurrent DuckDB operations
	// See database_test.go for detailed explanation of why this is required
	testDBSemaphore <- struct{}{}
	t.Cleanup(func() {
		<-testDBSemaphore
	})

	db, err := New(&config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true,
	}, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	return db
}

func TestCreateMediaServer(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		server  *models.MediaServer
		wantErr error
	}{
		{
			name: "create plex server",
			server: &models.MediaServer{
				Platform:       "plex",
				Name:           "My Plex Server",
				URLEncrypted:   "encrypted-url",
				TokenEncrypted: "encrypted-token",
				ServerID:       "plex-001",
				Enabled:        true,
				Source:         models.ServerSourceUI,
			},
			wantErr: nil,
		},
		{
			name: "create jellyfin server",
			server: &models.MediaServer{
				Platform:       "jellyfin",
				Name:           "My Jellyfin Server",
				URLEncrypted:   "encrypted-url-jf",
				TokenEncrypted: "encrypted-token-jf",
				ServerID:       "jellyfin-001",
				Enabled:        true,
				Source:         models.ServerSourceUI,
			},
			wantErr: nil,
		},
		{
			name: "create emby server",
			server: &models.MediaServer{
				Platform:       "emby",
				Name:           "My Emby Server",
				URLEncrypted:   "encrypted-url-emby",
				TokenEncrypted: "encrypted-token-emby",
				ServerID:       "emby-001",
				Enabled:        true,
				Source:         models.ServerSourceUI,
			},
			wantErr: nil,
		},
		{
			name: "create tautulli server",
			server: &models.MediaServer{
				Platform:       "tautulli",
				Name:           "My Tautulli Server",
				URLEncrypted:   "encrypted-url-tau",
				TokenEncrypted: "encrypted-token-tau",
				ServerID:       "tautulli-001",
				Enabled:        true,
				Source:         models.ServerSourceUI,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.CreateMediaServer(ctx, tt.server)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("CreateMediaServer() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr == nil {
				// Verify the server was created
				if tt.server.ID == "" {
					t.Error("CreateMediaServer() did not set ID")
				}
				if tt.server.CreatedAt.IsZero() {
					t.Error("CreateMediaServer() did not set CreatedAt")
				}
			}
		})
	}
}

func TestCreateMediaServer_DuplicateServerID(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create first server
	server1 := &models.MediaServer{
		Platform:       "plex",
		Name:           "Server 1",
		URLEncrypted:   "url-1",
		TokenEncrypted: "token-1",
		ServerID:       "duplicate-id",
		Enabled:        true,
		Source:         models.ServerSourceUI,
	}
	if err := db.CreateMediaServer(ctx, server1); err != nil {
		t.Fatalf("Failed to create first server: %v", err)
	}

	// Try to create second server with same ServerID
	server2 := &models.MediaServer{
		Platform:       "plex",
		Name:           "Server 2",
		URLEncrypted:   "url-2",
		TokenEncrypted: "token-2",
		ServerID:       "duplicate-id",
		Enabled:        true,
		Source:         models.ServerSourceUI,
	}
	err := db.CreateMediaServer(ctx, server2)
	if !errors.Is(err, ErrServerIDConflict) {
		t.Errorf("CreateMediaServer() with duplicate ServerID error = %v, want %v", err, ErrServerIDConflict)
	}
}

func TestGetMediaServer(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create a server
	server := &models.MediaServer{
		Platform:               "plex",
		Name:                   "Test Server",
		URLEncrypted:           "encrypted-url",
		TokenEncrypted:         "encrypted-token",
		ServerID:               "test-server-id",
		Enabled:                true,
		RealtimeEnabled:        true,
		WebhooksEnabled:        true,
		SessionPollingEnabled:  true,
		SessionPollingInterval: "30s",
		Source:                 models.ServerSourceUI,
		CreatedBy:              "user-123",
	}
	if err := db.CreateMediaServer(ctx, server); err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test getting the server
	t.Run("get existing server", func(t *testing.T) {
		got, err := db.GetMediaServer(ctx, server.ID)
		if err != nil {
			t.Errorf("GetMediaServer() error = %v", err)
		}
		if got == nil {
			t.Fatal("GetMediaServer() returned nil")
		}
		if got.Platform != server.Platform {
			t.Errorf("GetMediaServer() platform = %v, want %v", got.Platform, server.Platform)
		}
		if got.Name != server.Name {
			t.Errorf("GetMediaServer() name = %v, want %v", got.Name, server.Name)
		}
		if got.ServerID != server.ServerID {
			t.Errorf("GetMediaServer() server_id = %v, want %v", got.ServerID, server.ServerID)
		}
	})

	t.Run("get non-existing server", func(t *testing.T) {
		_, err := db.GetMediaServer(ctx, "non-existent-id")
		if !errors.Is(err, ErrServerNotFound) {
			t.Errorf("GetMediaServer() error = %v, want %v", err, ErrServerNotFound)
		}
	})
}

func TestGetMediaServerByServerID(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create a server
	server := &models.MediaServer{
		Platform:       "jellyfin",
		Name:           "Jellyfin Server",
		URLEncrypted:   "encrypted-url",
		TokenEncrypted: "encrypted-token",
		ServerID:       "unique-server-id",
		Enabled:        true,
		Source:         models.ServerSourceUI,
	}
	if err := db.CreateMediaServer(ctx, server); err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	t.Run("get by server_id", func(t *testing.T) {
		got, err := db.GetMediaServerByServerID(ctx, "unique-server-id")
		if err != nil {
			t.Errorf("GetMediaServerByServerID() error = %v", err)
		}
		if got == nil {
			t.Fatal("GetMediaServerByServerID() returned nil")
		}
		if got.ID != server.ID {
			t.Errorf("GetMediaServerByServerID() ID = %v, want %v", got.ID, server.ID)
		}
	})

	t.Run("get non-existing server_id", func(t *testing.T) {
		_, err := db.GetMediaServerByServerID(ctx, "non-existent-server-id")
		if !errors.Is(err, ErrServerNotFound) {
			t.Errorf("GetMediaServerByServerID() error = %v, want %v", err, ErrServerNotFound)
		}
	})
}

func TestListMediaServers(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create servers of different platforms
	servers := []*models.MediaServer{
		{Platform: "plex", Name: "Plex 1", URLEncrypted: "u1", TokenEncrypted: "t1", ServerID: "p1", Enabled: true, Source: models.ServerSourceUI},
		{Platform: "plex", Name: "Plex 2", URLEncrypted: "u2", TokenEncrypted: "t2", ServerID: "p2", Enabled: false, Source: models.ServerSourceUI},
		{Platform: "jellyfin", Name: "Jellyfin 1", URLEncrypted: "u3", TokenEncrypted: "t3", ServerID: "j1", Enabled: true, Source: models.ServerSourceUI},
		{Platform: "emby", Name: "Emby 1", URLEncrypted: "u4", TokenEncrypted: "t4", ServerID: "e1", Enabled: true, Source: models.ServerSourceUI},
	}
	for _, s := range servers {
		if err := db.CreateMediaServer(ctx, s); err != nil {
			t.Fatalf("Failed to create server %s: %v", s.Name, err)
		}
	}

	t.Run("list all servers", func(t *testing.T) {
		got, err := db.ListMediaServers(ctx, "", false)
		if err != nil {
			t.Errorf("ListMediaServers() error = %v", err)
		}
		if len(got) != 4 {
			t.Errorf("ListMediaServers() returned %d servers, want 4", len(got))
		}
	})

	t.Run("filter by platform", func(t *testing.T) {
		got, err := db.ListMediaServers(ctx, "plex", false)
		if err != nil {
			t.Errorf("ListMediaServers() error = %v", err)
		}
		if len(got) != 2 {
			t.Errorf("ListMediaServers(plex) returned %d servers, want 2", len(got))
		}
	})

	t.Run("filter enabled only", func(t *testing.T) {
		got, err := db.ListMediaServers(ctx, "", true)
		if err != nil {
			t.Errorf("ListMediaServers() error = %v", err)
		}
		if len(got) != 3 {
			t.Errorf("ListMediaServers(enabledOnly) returned %d servers, want 3", len(got))
		}
	})

	t.Run("filter platform and enabled", func(t *testing.T) {
		got, err := db.ListMediaServers(ctx, "plex", true)
		if err != nil {
			t.Errorf("ListMediaServers() error = %v", err)
		}
		if len(got) != 1 {
			t.Errorf("ListMediaServers(plex, enabledOnly) returned %d servers, want 1", len(got))
		}
	})
}

func TestUpdateMediaServer(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create a server
	server := &models.MediaServer{
		Platform:       "plex",
		Name:           "Original Name",
		URLEncrypted:   "original-url",
		TokenEncrypted: "original-token",
		ServerID:       "update-test-id",
		Enabled:        true,
		Source:         models.ServerSourceUI,
	}
	if err := db.CreateMediaServer(ctx, server); err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	t.Run("update server", func(t *testing.T) {
		server.Name = "Updated Name"
		server.Enabled = false
		server.RealtimeEnabled = true

		err := db.UpdateMediaServer(ctx, server)
		if err != nil {
			t.Errorf("UpdateMediaServer() error = %v", err)
		}

		// Verify update
		got, _ := db.GetMediaServer(ctx, server.ID)
		if got.Name != "Updated Name" {
			t.Errorf("UpdateMediaServer() name = %v, want %v", got.Name, "Updated Name")
		}
		if got.Enabled {
			t.Error("UpdateMediaServer() enabled should be false")
		}
		if !got.RealtimeEnabled {
			t.Error("UpdateMediaServer() realtime_enabled should be true")
		}
	})

	t.Run("update non-existing server", func(t *testing.T) {
		nonExistent := &models.MediaServer{
			ID:             "non-existent-id",
			Platform:       "plex",
			Name:           "Name",
			URLEncrypted:   "url",
			TokenEncrypted: "token",
		}
		err := db.UpdateMediaServer(ctx, nonExistent)
		if !errors.Is(err, ErrServerNotFound) {
			t.Errorf("UpdateMediaServer() error = %v, want %v", err, ErrServerNotFound)
		}
	})
}

func TestUpdateMediaServer_Immutable(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create an env-var server (immutable)
	server := &models.MediaServer{
		Platform:       "plex",
		Name:           "Env Server",
		URLEncrypted:   "url",
		TokenEncrypted: "token",
		ServerID:       "env-server-id",
		Enabled:        true,
		Source:         models.ServerSourceEnv,
	}
	if err := db.CreateMediaServer(ctx, server); err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Try to update it
	server.Name = "New Name"
	err := db.UpdateMediaServer(ctx, server)
	if !errors.Is(err, ErrImmutableServer) {
		t.Errorf("UpdateMediaServer() error = %v, want %v", err, ErrImmutableServer)
	}
}

func TestDeleteMediaServer(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create a UI server (deletable)
	server := &models.MediaServer{
		Platform:       "plex",
		Name:           "To Delete",
		URLEncrypted:   "url",
		TokenEncrypted: "token",
		ServerID:       "delete-test-id",
		Enabled:        true,
		Source:         models.ServerSourceUI,
	}
	if err := db.CreateMediaServer(ctx, server); err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	t.Run("delete existing server", func(t *testing.T) {
		err := db.DeleteMediaServer(ctx, server.ID)
		if err != nil {
			t.Errorf("DeleteMediaServer() error = %v", err)
		}

		// Verify deletion
		_, err = db.GetMediaServer(ctx, server.ID)
		if !errors.Is(err, ErrServerNotFound) {
			t.Error("DeleteMediaServer() server should be deleted")
		}
	})

	t.Run("delete non-existing server", func(t *testing.T) {
		err := db.DeleteMediaServer(ctx, "non-existent-id")
		if !errors.Is(err, ErrServerNotFound) {
			t.Errorf("DeleteMediaServer() error = %v, want %v", err, ErrServerNotFound)
		}
	})
}

func TestDeleteMediaServer_Immutable(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create an env-var server (immutable)
	server := &models.MediaServer{
		Platform:       "plex",
		Name:           "Env Server",
		URLEncrypted:   "url",
		TokenEncrypted: "token",
		ServerID:       "env-delete-test",
		Enabled:        true,
		Source:         models.ServerSourceEnv,
	}
	if err := db.CreateMediaServer(ctx, server); err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Try to delete it
	err := db.DeleteMediaServer(ctx, server.ID)
	if !errors.Is(err, ErrImmutableServer) {
		t.Errorf("DeleteMediaServer() error = %v, want %v", err, ErrImmutableServer)
	}
}

func TestEnableDisableMediaServer(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create a server
	server := &models.MediaServer{
		Platform:       "plex",
		Name:           "Enable Test",
		URLEncrypted:   "url",
		TokenEncrypted: "token",
		ServerID:       "enable-test-id",
		Enabled:        true,
		Source:         models.ServerSourceUI,
	}
	if err := db.CreateMediaServer(ctx, server); err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	t.Run("disable server", func(t *testing.T) {
		err := db.DisableMediaServer(ctx, server.ID)
		if err != nil {
			t.Errorf("DisableMediaServer() error = %v", err)
		}

		got, _ := db.GetMediaServer(ctx, server.ID)
		if got.Enabled {
			t.Error("DisableMediaServer() server should be disabled")
		}
	})

	t.Run("enable server", func(t *testing.T) {
		err := db.EnableMediaServer(ctx, server.ID)
		if err != nil {
			t.Errorf("EnableMediaServer() error = %v", err)
		}

		got, _ := db.GetMediaServer(ctx, server.ID)
		if !got.Enabled {
			t.Error("EnableMediaServer() server should be enabled")
		}
	})

	t.Run("enable non-existing server", func(t *testing.T) {
		err := db.EnableMediaServer(ctx, "non-existent")
		if !errors.Is(err, ErrServerNotFound) {
			t.Errorf("EnableMediaServer() error = %v, want %v", err, ErrServerNotFound)
		}
	})
}

func TestUpdateMediaServerSyncStatus(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create a server
	server := &models.MediaServer{
		Platform:       "plex",
		Name:           "Sync Status Test",
		URLEncrypted:   "url",
		TokenEncrypted: "token",
		ServerID:       "sync-status-test",
		Enabled:        true,
		Source:         models.ServerSourceUI,
	}
	if err := db.CreateMediaServer(ctx, server); err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	t.Run("update sync status success", func(t *testing.T) {
		err := db.UpdateMediaServerSyncStatus(ctx, server.ID, "completed", "")
		if err != nil {
			t.Errorf("UpdateMediaServerSyncStatus() error = %v", err)
		}

		got, _ := db.GetMediaServer(ctx, server.ID)
		if got.LastSyncStatus != "completed" {
			t.Errorf("UpdateMediaServerSyncStatus() status = %v, want 'completed'", got.LastSyncStatus)
		}
		if got.LastSyncAt == nil {
			t.Error("UpdateMediaServerSyncStatus() should set LastSyncAt")
		}
		if got.LastError != "" {
			t.Errorf("UpdateMediaServerSyncStatus() should clear LastError, got %v", got.LastError)
		}
	})

	t.Run("update sync status with error", func(t *testing.T) {
		err := db.UpdateMediaServerSyncStatus(ctx, server.ID, "error", "Connection refused")
		if err != nil {
			t.Errorf("UpdateMediaServerSyncStatus() error = %v", err)
		}

		got, _ := db.GetMediaServer(ctx, server.ID)
		if got.LastSyncStatus != "error" {
			t.Errorf("UpdateMediaServerSyncStatus() status = %v, want 'error'", got.LastSyncStatus)
		}
		if got.LastError != "Connection refused" {
			t.Errorf("UpdateMediaServerSyncStatus() error = %v, want 'Connection refused'", got.LastError)
		}
		if got.LastErrorAt == nil {
			t.Error("UpdateMediaServerSyncStatus() should set LastErrorAt")
		}
	})
}

func TestCreateMediaServerAudit(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	audit := &models.MediaServerAudit{
		ServerID:  "server-123",
		Action:    models.ServerAuditActionCreate,
		UserID:    "user-456",
		Username:  "admin",
		Changes:   `{"name": "New Server"}`,
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	err := db.CreateMediaServerAudit(ctx, audit)
	if err != nil {
		t.Errorf("CreateMediaServerAudit() error = %v", err)
	}

	if audit.ID == "" {
		t.Error("CreateMediaServerAudit() should set ID")
	}
	if audit.CreatedAt.IsZero() {
		t.Error("CreateMediaServerAudit() should set CreatedAt")
	}
}

func TestListMediaServerAuditLogs(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create some audit logs
	serverID := "audit-test-server"
	for i := 0; i < 5; i++ {
		audit := &models.MediaServerAudit{
			ServerID:  serverID,
			Action:    models.ServerAuditActionUpdate,
			UserID:    "user-456",
			Username:  "admin",
			Changes:   `{}`,
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour),
		}
		if err := db.CreateMediaServerAudit(ctx, audit); err != nil {
			t.Fatalf("Failed to create audit log: %v", err)
		}
	}

	t.Run("list audit logs", func(t *testing.T) {
		logs, err := db.ListMediaServerAuditLogs(ctx, serverID, 10)
		if err != nil {
			t.Errorf("ListMediaServerAuditLogs() error = %v", err)
		}
		if len(logs) != 5 {
			t.Errorf("ListMediaServerAuditLogs() returned %d logs, want 5", len(logs))
		}
	})

	t.Run("list with limit", func(t *testing.T) {
		logs, err := db.ListMediaServerAuditLogs(ctx, serverID, 3)
		if err != nil {
			t.Errorf("ListMediaServerAuditLogs() error = %v", err)
		}
		if len(logs) != 3 {
			t.Errorf("ListMediaServerAuditLogs() with limit returned %d logs, want 3", len(logs))
		}
	})
}

func TestCountMediaServers(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create servers of different platforms
	servers := []*models.MediaServer{
		{Platform: "plex", Name: "P1", URLEncrypted: "u1", TokenEncrypted: "t1", ServerID: "p1", Source: models.ServerSourceUI},
		{Platform: "plex", Name: "P2", URLEncrypted: "u2", TokenEncrypted: "t2", ServerID: "p2", Source: models.ServerSourceUI},
		{Platform: "jellyfin", Name: "J1", URLEncrypted: "u3", TokenEncrypted: "t3", ServerID: "j1", Source: models.ServerSourceUI},
	}
	for _, s := range servers {
		if err := db.CreateMediaServer(ctx, s); err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}
	}

	counts, err := db.CountMediaServers(ctx)
	if err != nil {
		t.Errorf("CountMediaServers() error = %v", err)
	}

	if counts["plex"] != 2 {
		t.Errorf("CountMediaServers() plex = %d, want 2", counts["plex"])
	}
	if counts["jellyfin"] != 1 {
		t.Errorf("CountMediaServers() jellyfin = %d, want 1", counts["jellyfin"])
	}
}

func TestIsUniqueConstraintError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "unique constraint error",
			err:  errors.New("UNIQUE constraint failed"),
			want: true,
		},
		{
			name: "duplicate key error",
			err:  errors.New("Duplicate key violation"),
			want: true,
		},
		{
			name: "regular error",
			err:  errors.New("connection refused"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUniqueConstraintError(tt.err)
			if got != tt.want {
				t.Errorf("isUniqueConstraintError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestJsonToString(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want string
	}{
		{
			name: "nil value",
			val:  nil,
			want: "{}",
		},
		{
			name: "string value",
			val:  `{"key": "value"}`,
			want: `{"key": "value"}`,
		},
		{
			name: "byte slice value",
			val:  []byte(`{"key": "value"}`),
			want: `{"key": "value"}`,
		},
		{
			name: "map value",
			val:  map[string]any{"key": "value"},
			want: `{"key":"value"}`,
		},
		{
			name: "empty map",
			val:  map[string]any{},
			want: "{}",
		},
		{
			name: "unknown type",
			val:  123,
			want: "{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonToString(tt.val)
			if got != tt.want {
				t.Errorf("jsonToString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUpdateMediaServerSyncStatus_NotFound(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	err := db.UpdateMediaServerSyncStatus(ctx, "non-existent-id", "completed", "")
	if !errors.Is(err, ErrServerNotFound) {
		t.Errorf("UpdateMediaServerSyncStatus() error = %v, want %v", err, ErrServerNotFound)
	}
}

func TestDisableMediaServer_NotFound(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	err := db.DisableMediaServer(ctx, "non-existent-id")
	if !errors.Is(err, ErrServerNotFound) {
		t.Errorf("DisableMediaServer() error = %v, want %v", err, ErrServerNotFound)
	}
}

func TestCreateMediaServer_WithAllFields(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	server := &models.MediaServer{
		Platform:               "plex",
		Name:                   "Full Test Server",
		URLEncrypted:           "encrypted-url",
		TokenEncrypted:         "encrypted-token",
		ServerID:               "full-test-id",
		Enabled:                true,
		Settings:               `{"custom":"setting"}`,
		RealtimeEnabled:        true,
		WebhooksEnabled:        true,
		SessionPollingEnabled:  true,
		SessionPollingInterval: "60s",
		Source:                 models.ServerSourceUI,
		CreatedBy:              "test-user",
	}

	err := db.CreateMediaServer(ctx, server)
	if err != nil {
		t.Fatalf("CreateMediaServer() error = %v", err)
	}

	// Verify all fields were saved
	got, err := db.GetMediaServer(ctx, server.ID)
	if err != nil {
		t.Fatalf("GetMediaServer() error = %v", err)
	}

	if got.Settings != `{"custom":"setting"}` {
		t.Errorf("Settings = %q, want %q", got.Settings, `{"custom":"setting"}`)
	}
	if got.SessionPollingInterval != "60s" {
		t.Errorf("SessionPollingInterval = %q, want %q", got.SessionPollingInterval, "60s")
	}
	if got.CreatedBy != "test-user" {
		t.Errorf("CreatedBy = %q, want %q", got.CreatedBy, "test-user")
	}
}

func TestListMediaServers_EmptyResult(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// List from empty database
	servers, err := db.ListMediaServers(ctx, "", false)
	if err != nil {
		t.Errorf("ListMediaServers() error = %v", err)
	}
	if servers == nil {
		t.Error("ListMediaServers() should return empty slice, not nil")
	}
	if len(servers) != 0 {
		t.Errorf("ListMediaServers() returned %d servers, want 0", len(servers))
	}
}

func TestListMediaServerAuditLogs_DefaultLimit(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create audit log
	audit := &models.MediaServerAudit{
		ServerID: "test-server",
		Action:   models.ServerAuditActionCreate,
		UserID:   "user-123",
		Username: "admin",
		Changes:  `{}`,
	}
	_ = db.CreateMediaServerAudit(ctx, audit)

	// List with default limit (0 -> 100)
	logs, err := db.ListMediaServerAuditLogs(ctx, "test-server", 0)
	if err != nil {
		t.Errorf("ListMediaServerAuditLogs() error = %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("ListMediaServerAuditLogs() returned %d logs, want 1", len(logs))
	}
}

func TestCountMediaServers_Empty(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	counts, err := db.CountMediaServers(ctx)
	if err != nil {
		t.Errorf("CountMediaServers() error = %v", err)
	}
	if len(counts) != 0 {
		t.Errorf("CountMediaServers() returned %d counts, want 0", len(counts))
	}
}

func TestUpdateMediaServer_ServerIDConflict(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create two servers
	server1 := &models.MediaServer{
		Platform:       "plex",
		Name:           "Server 1",
		URLEncrypted:   "url-1",
		TokenEncrypted: "token-1",
		ServerID:       "server-id-1",
		Source:         models.ServerSourceUI,
	}
	server2 := &models.MediaServer{
		Platform:       "plex",
		Name:           "Server 2",
		URLEncrypted:   "url-2",
		TokenEncrypted: "token-2",
		ServerID:       "server-id-2",
		Source:         models.ServerSourceUI,
	}
	_ = db.CreateMediaServer(ctx, server1)
	_ = db.CreateMediaServer(ctx, server2)

	// Try to update server2 with server1's ServerID
	server2.ServerID = "server-id-1"
	err := db.UpdateMediaServer(ctx, server2)
	if !errors.Is(err, ErrServerIDConflict) {
		t.Errorf("UpdateMediaServer() error = %v, want %v", err, ErrServerIDConflict)
	}
}

func TestDeleteMediaServer_EnvServer(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create an env-sourced server (immutable)
	server := &models.MediaServer{
		Platform:       "plex",
		Name:           "Env Server",
		URLEncrypted:   "encrypted-url",
		TokenEncrypted: "encrypted-token",
		ServerID:       "env-server-id",
		Source:         models.ServerSourceEnv,
	}
	if err := db.CreateMediaServer(ctx, server); err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Try to delete - should fail with ErrImmutableServer
	err := db.DeleteMediaServer(ctx, server.ID)
	if !errors.Is(err, ErrImmutableServer) {
		t.Errorf("DeleteMediaServer() error = %v, want %v", err, ErrImmutableServer)
	}
}

func TestUpdateMediaServer_EnvServer(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create an env-sourced server (immutable)
	server := &models.MediaServer{
		Platform:       "plex",
		Name:           "Env Server",
		URLEncrypted:   "encrypted-url",
		TokenEncrypted: "encrypted-token",
		ServerID:       "env-server-id",
		Source:         models.ServerSourceEnv,
	}
	if err := db.CreateMediaServer(ctx, server); err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Try to update - should fail with ErrImmutableServer
	server.Name = "Updated Name"
	err := db.UpdateMediaServer(ctx, server)
	if !errors.Is(err, ErrImmutableServer) {
		t.Errorf("UpdateMediaServer() error = %v, want %v", err, ErrImmutableServer)
	}
}

func TestUpdateMediaServer_NotFound(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Try to update non-existent server
	server := &models.MediaServer{
		ID:             "non-existent-id",
		Platform:       "plex",
		Name:           "Test Server",
		URLEncrypted:   "encrypted-url",
		TokenEncrypted: "encrypted-token",
		Source:         models.ServerSourceUI,
	}
	err := db.UpdateMediaServer(ctx, server)
	if !errors.Is(err, ErrServerNotFound) {
		t.Errorf("UpdateMediaServer() error = %v, want %v", err, ErrServerNotFound)
	}
}

func TestListMediaServers_WithFilters(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create servers with different platforms and enabled states
	servers := []*models.MediaServer{
		{
			Platform:       "plex",
			Name:           "Plex Enabled",
			URLEncrypted:   "url-1",
			TokenEncrypted: "token-1",
			ServerID:       "server-1",
			Enabled:        true,
			Source:         models.ServerSourceUI,
		},
		{
			Platform:       "plex",
			Name:           "Plex Disabled",
			URLEncrypted:   "url-2",
			TokenEncrypted: "token-2",
			ServerID:       "server-2",
			Enabled:        false,
			Source:         models.ServerSourceUI,
		},
		{
			Platform:       "jellyfin",
			Name:           "Jellyfin Enabled",
			URLEncrypted:   "url-3",
			TokenEncrypted: "token-3",
			ServerID:       "server-3",
			Enabled:        true,
			Source:         models.ServerSourceUI,
		},
	}
	for _, s := range servers {
		if err := db.CreateMediaServer(ctx, s); err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}
	}

	// Test filter by platform
	plexServers, err := db.ListMediaServers(ctx, "plex", false)
	if err != nil {
		t.Errorf("ListMediaServers(plex) error = %v", err)
	}
	if len(plexServers) != 2 {
		t.Errorf("ListMediaServers(plex) returned %d servers, want 2", len(plexServers))
	}

	// Test filter by enabled only
	enabledServers, err := db.ListMediaServers(ctx, "", true)
	if err != nil {
		t.Errorf("ListMediaServers(enabled) error = %v", err)
	}
	if len(enabledServers) != 2 {
		t.Errorf("ListMediaServers(enabled) returned %d servers, want 2", len(enabledServers))
	}

	// Test combined filters
	plexEnabled, err := db.ListMediaServers(ctx, "plex", true)
	if err != nil {
		t.Errorf("ListMediaServers(plex, enabled) error = %v", err)
	}
	if len(plexEnabled) != 1 {
		t.Errorf("ListMediaServers(plex, enabled) returned %d servers, want 1", len(plexEnabled))
	}
}

func TestSetServerEnabled_NotFound(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Test enable non-existent server
	err := db.EnableMediaServer(ctx, "non-existent-id")
	if !errors.Is(err, ErrServerNotFound) {
		t.Errorf("EnableMediaServer() error = %v, want %v", err, ErrServerNotFound)
	}
}

func TestCountMediaServers_MultiplePlatforms(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// Create servers with different platforms
	servers := []*models.MediaServer{
		{Platform: "plex", Name: "Plex 1", URLEncrypted: "url-1", TokenEncrypted: "token-1", ServerID: "server-1"},
		{Platform: "plex", Name: "Plex 2", URLEncrypted: "url-2", TokenEncrypted: "token-2", ServerID: "server-2"},
		{Platform: "jellyfin", Name: "Jellyfin", URLEncrypted: "url-3", TokenEncrypted: "token-3", ServerID: "server-3"},
		{Platform: "emby", Name: "Emby", URLEncrypted: "url-4", TokenEncrypted: "token-4", ServerID: "server-4"},
	}
	for _, s := range servers {
		if err := db.CreateMediaServer(ctx, s); err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}
	}

	counts, err := db.CountMediaServers(ctx)
	if err != nil {
		t.Fatalf("CountMediaServers() error = %v", err)
	}

	if counts["plex"] != 2 {
		t.Errorf("CountMediaServers() plex = %d, want 2", counts["plex"])
	}
	if counts["jellyfin"] != 1 {
		t.Errorf("CountMediaServers() jellyfin = %d, want 1", counts["jellyfin"])
	}
	if counts["emby"] != 1 {
		t.Errorf("CountMediaServers() emby = %d, want 1", counts["emby"])
	}
}

func TestListMediaServerAuditLogs_EmptyResult(t *testing.T) {
	db := setupTestDBForMediaServers(t)
	defer db.Close()

	ctx := context.Background()

	// List audit logs for non-existent server
	audits, err := db.ListMediaServerAuditLogs(ctx, "non-existent-server", 10)
	if err != nil {
		t.Errorf("ListMediaServerAuditLogs() error = %v", err)
	}
	if audits == nil {
		t.Error("ListMediaServerAuditLogs() should return empty slice, not nil")
	}
	if len(audits) != 0 {
		t.Errorf("ListMediaServerAuditLogs() returned %d audits, want 0", len(audits))
	}
}
