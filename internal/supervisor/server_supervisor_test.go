// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package supervisor

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
)

// testLogger creates a logger for testing that minimizes output
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// mockConfigDB implements config.ServerConfigManagerDB for testing
type mockConfigDB struct {
	servers []models.MediaServer
	listErr error
}

func (m *mockConfigDB) ListMediaServers(ctx context.Context, platform string, enabledOnly bool) ([]models.MediaServer, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.servers, nil
}

func (m *mockConfigDB) GetMediaServer(ctx context.Context, id string) (*models.MediaServer, error) {
	for _, s := range m.servers {
		if s.ID == id {
			return &s, nil
		}
	}
	return nil, errors.New("not found")
}

func TestNewServerSupervisor(t *testing.T) {
	// Create a minimal supervisor tree for testing
	tree, err := NewSupervisorTree(testLogger(), DefaultTreeConfig())
	if err != nil {
		t.Fatalf("Failed to create supervisor tree: %v", err)
	}

	// Create a config manager
	encryptor, _ := config.NewCredentialEncryptor("test-secret-key-for-testing")
	db := &mockConfigDB{}
	cfg := &config.Config{}
	configMgr, _ := config.NewServerConfigManager(db, encryptor, cfg)

	tests := []struct {
		name      string
		tree      *SupervisorTree
		configMgr *config.ServerConfigManager
		wantErr   error
	}{
		{
			name:      "valid parameters",
			tree:      tree,
			configMgr: configMgr,
			wantErr:   nil,
		},
		{
			name:      "nil tree",
			tree:      nil,
			configMgr: configMgr,
			wantErr:   ErrNilSupervisorTree,
		},
		{
			name:      "nil config manager",
			tree:      tree,
			configMgr: nil,
			wantErr:   ErrNilConfigManager,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sup, err := NewServerSupervisor(tt.tree, tt.configMgr, nil, nil, nil, nil)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewServerSupervisor() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && sup == nil {
				t.Error("NewServerSupervisor() returned nil for valid input")
			}
		})
	}
}

func TestServerSupervisor_AddServer(t *testing.T) {
	tree, _ := NewSupervisorTree(testLogger(), DefaultTreeConfig())
	encryptor, _ := config.NewCredentialEncryptor("test-secret-key-for-testing")
	db := &mockConfigDB{}
	cfg := &config.Config{}
	configMgr, _ := config.NewServerConfigManager(db, encryptor, cfg)
	sup, _ := NewServerSupervisor(tree, configMgr, nil, nil, nil, nil)

	ctx := context.Background()

	// Add a Plex server
	plexCfg := &config.UnifiedServerConfig{
		ID:       "test-plex-1",
		ServerID: "plex-server-1",
		Platform: "plex",
		Name:     "Test Plex",
		URL:      "http://plex:32400",
		Token:    "test-token",
		Enabled:  true,
	}

	err := sup.AddServer(ctx, plexCfg)
	if err != nil {
		t.Errorf("AddServer() error = %v", err)
	}

	// Check if running
	if !sup.IsServerRunning("plex-server-1") {
		t.Error("Server should be running after AddServer")
	}

	// Try adding the same server again
	err = sup.AddServer(ctx, plexCfg)
	if !errors.Is(err, ErrServerAlreadyExists) {
		t.Errorf("AddServer() duplicate error = %v, want ErrServerAlreadyExists", err)
	}

	// Add nil config
	err = sup.AddServer(ctx, nil)
	if err == nil {
		t.Error("AddServer(nil) should return error")
	}
}

func TestServerSupervisor_AddServer_AllPlatforms(t *testing.T) {
	tree, _ := NewSupervisorTree(testLogger(), DefaultTreeConfig())
	encryptor, _ := config.NewCredentialEncryptor("test-secret-key-for-testing")
	db := &mockConfigDB{}
	cfg := &config.Config{}
	configMgr, _ := config.NewServerConfigManager(db, encryptor, cfg)
	sup, _ := NewServerSupervisor(tree, configMgr, nil, nil, nil, nil)

	ctx := context.Background()

	platforms := []struct {
		platform string
		valid    bool
	}{
		{"plex", true},
		{"jellyfin", true},
		{"emby", true},
		{"tautulli", true},
		{"invalid", false},
	}

	for _, p := range platforms {
		t.Run(p.platform, func(t *testing.T) {
			cfg := &config.UnifiedServerConfig{
				ID:       "test-" + p.platform,
				ServerID: p.platform + "-server",
				Platform: p.platform,
				Name:     "Test " + p.platform,
				URL:      "http://server:8080",
				Token:    "token",
				Enabled:  true,
			}

			err := sup.AddServer(ctx, cfg)
			if p.valid {
				if err != nil {
					t.Errorf("AddServer(%s) unexpected error = %v", p.platform, err)
				}
			} else {
				if err == nil {
					t.Errorf("AddServer(%s) should return error for invalid platform", p.platform)
				}
			}
		})
	}
}

func TestServerSupervisor_RemoveServer(t *testing.T) {
	tree, _ := NewSupervisorTree(testLogger(), DefaultTreeConfig())
	encryptor, _ := config.NewCredentialEncryptor("test-secret-key-for-testing")
	db := &mockConfigDB{}
	cfg := &config.Config{}
	configMgr, _ := config.NewServerConfigManager(db, encryptor, cfg)
	sup, _ := NewServerSupervisor(tree, configMgr, nil, nil, nil, nil)

	// Start the supervisor tree in background (required for remove operations)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = tree.Serve(ctx) }()
	time.Sleep(50 * time.Millisecond) // Allow supervisor to start

	// Add a server first
	serverCfg := &config.UnifiedServerConfig{
		ID:       "test-plex-1",
		ServerID: "plex-server-1",
		Platform: "plex",
		Name:     "Test Plex",
		URL:      "http://plex:32400",
		Token:    "test-token",
		Enabled:  true,
	}
	_ = sup.AddServer(ctx, serverCfg)

	// Remove the server
	err := sup.RemoveServer(ctx, "plex-server-1")
	if err != nil {
		t.Errorf("RemoveServer() error = %v", err)
	}

	// Check it's no longer running
	if sup.IsServerRunning("plex-server-1") {
		t.Error("Server should not be running after RemoveServer")
	}

	// Try removing non-existent server
	err = sup.RemoveServer(ctx, "nonexistent")
	if !errors.Is(err, ErrServerNotRunning) {
		t.Errorf("RemoveServer(nonexistent) error = %v, want ErrServerNotRunning", err)
	}
}

func TestServerSupervisor_UpdateServer(t *testing.T) {
	tree, _ := NewSupervisorTree(testLogger(), DefaultTreeConfig())
	encryptor, _ := config.NewCredentialEncryptor("test-secret-key-for-testing")
	db := &mockConfigDB{}
	cfg := &config.Config{}
	configMgr, _ := config.NewServerConfigManager(db, encryptor, cfg)
	sup, _ := NewServerSupervisor(tree, configMgr, nil, nil, nil, nil)

	// Start the supervisor tree in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = tree.Serve(ctx) }()
	time.Sleep(50 * time.Millisecond)

	// Add a server first
	serverCfg := &config.UnifiedServerConfig{
		ID:       "test-plex-1",
		ServerID: "plex-server-1",
		Platform: "plex",
		Name:     "Test Plex",
		URL:      "http://plex:32400",
		Token:    "test-token",
		Enabled:  true,
	}
	_ = sup.AddServer(ctx, serverCfg)

	// Update the server
	updatedCfg := &config.UnifiedServerConfig{
		ID:       "test-plex-1",
		ServerID: "plex-server-1",
		Platform: "plex",
		Name:     "Updated Plex",
		URL:      "http://plex-new:32400",
		Token:    "new-token",
		Enabled:  true,
	}

	err := sup.UpdateServer(ctx, updatedCfg)
	if err != nil {
		t.Errorf("UpdateServer() error = %v", err)
	}

	// Check it's still running
	if !sup.IsServerRunning("plex-server-1") {
		t.Error("Server should still be running after UpdateServer")
	}

	// Update non-existent server (should add it)
	newCfg := &config.UnifiedServerConfig{
		ID:       "test-plex-2",
		ServerID: "plex-server-2",
		Platform: "plex",
		Name:     "New Plex",
		URL:      "http://plex2:32400",
		Token:    "token2",
		Enabled:  true,
	}

	err = sup.UpdateServer(ctx, newCfg)
	if err != nil {
		t.Errorf("UpdateServer() for new server error = %v", err)
	}

	if !sup.IsServerRunning("plex-server-2") {
		t.Error("New server should be running after UpdateServer")
	}
}

func TestServerSupervisor_GetServerStatus(t *testing.T) {
	tree, _ := NewSupervisorTree(testLogger(), DefaultTreeConfig())
	encryptor, _ := config.NewCredentialEncryptor("test-secret-key-for-testing")
	db := &mockConfigDB{}
	cfg := &config.Config{}
	configMgr, _ := config.NewServerConfigManager(db, encryptor, cfg)
	sup, _ := NewServerSupervisor(tree, configMgr, nil, nil, nil, nil)

	ctx := context.Background()

	// Add a server
	serverCfg := &config.UnifiedServerConfig{
		ID:       "test-plex-1",
		ServerID: "plex-server-1",
		Platform: "plex",
		Name:     "Test Plex",
		URL:      "http://plex:32400",
		Token:    "test-token",
		Enabled:  true,
		Status:   "connected",
	}
	_ = sup.AddServer(ctx, serverCfg)

	// Get status
	status, err := sup.GetServerStatus("plex-server-1")
	if err != nil {
		t.Errorf("GetServerStatus() error = %v", err)
	}

	if status == nil {
		t.Fatal("GetServerStatus() returned nil status")
	}

	if status.ServerID != "plex-server-1" {
		t.Errorf("ServerID = %s, want plex-server-1", status.ServerID)
	}
	if status.Platform != "plex" {
		t.Errorf("Platform = %s, want plex", status.Platform)
	}
	if !status.Running {
		t.Error("Running should be true")
	}
	if status.StartedAt == nil {
		t.Error("StartedAt should not be nil")
	}

	// Get status for non-existent server
	_, err = sup.GetServerStatus("nonexistent")
	if !errors.Is(err, ErrServerNotRunning) {
		t.Errorf("GetServerStatus(nonexistent) error = %v, want ErrServerNotRunning", err)
	}
}

func TestServerSupervisor_GetAllServerStatuses(t *testing.T) {
	tree, _ := NewSupervisorTree(testLogger(), DefaultTreeConfig())
	encryptor, _ := config.NewCredentialEncryptor("test-secret-key-for-testing")
	db := &mockConfigDB{}
	cfg := &config.Config{}
	configMgr, _ := config.NewServerConfigManager(db, encryptor, cfg)
	sup, _ := NewServerSupervisor(tree, configMgr, nil, nil, nil, nil)

	ctx := context.Background()

	// Add multiple servers
	servers := []struct {
		serverID string
		platform string
	}{
		{"plex-1", "plex"},
		{"jellyfin-1", "jellyfin"},
		{"emby-1", "emby"},
	}

	for _, s := range servers {
		cfg := &config.UnifiedServerConfig{
			ID:       "test-" + s.serverID,
			ServerID: s.serverID,
			Platform: s.platform,
			Name:     "Test " + s.platform,
			URL:      "http://server:8080",
			Token:    "token",
			Enabled:  true,
		}
		_ = sup.AddServer(ctx, cfg)
	}

	// Get all statuses
	statuses := sup.GetAllServerStatuses()
	if len(statuses) != len(servers) {
		t.Errorf("GetAllServerStatuses() got %d, want %d", len(statuses), len(servers))
	}

	// All should be running
	for _, status := range statuses {
		if !status.Running {
			t.Errorf("Server %s should be running", status.ServerID)
		}
	}
}

func TestServerSupervisor_StopAll(t *testing.T) {
	tree, _ := NewSupervisorTree(testLogger(), DefaultTreeConfig())
	encryptor, _ := config.NewCredentialEncryptor("test-secret-key-for-testing")
	db := &mockConfigDB{}
	cfg := &config.Config{}
	configMgr, _ := config.NewServerConfigManager(db, encryptor, cfg)
	sup, _ := NewServerSupervisor(tree, configMgr, nil, nil, nil, nil)

	// Start the supervisor tree in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = tree.Serve(ctx) }()
	time.Sleep(50 * time.Millisecond)

	// Add servers
	for i := 0; i < 3; i++ {
		cfg := &config.UnifiedServerConfig{
			ID:       "test-plex-" + string(rune('0'+i)),
			ServerID: "plex-" + string(rune('0'+i)),
			Platform: "plex",
			Name:     "Test Plex",
			URL:      "http://plex:32400",
			Token:    "token",
			Enabled:  true,
		}
		_ = sup.AddServer(ctx, cfg)
	}

	// Stop all
	err := sup.StopAll(ctx)
	if err != nil {
		t.Errorf("StopAll() error = %v", err)
	}

	// Check none are running
	statuses := sup.GetAllServerStatuses()
	if len(statuses) != 0 {
		t.Errorf("GetAllServerStatuses() after StopAll got %d, want 0", len(statuses))
	}
}

func TestServerSupervisor_StartAll(t *testing.T) {
	tree, _ := NewSupervisorTree(testLogger(), DefaultTreeConfig())
	encryptor, _ := config.NewCredentialEncryptor("test-secret-key-for-testing")

	// Encrypt test data
	encURL, _ := encryptor.Encrypt("http://plex:32400")
	encToken, _ := encryptor.Encrypt("test-token")

	// Create mock DB with servers
	db := &mockConfigDB{
		servers: []models.MediaServer{
			{
				ID:             "db-1",
				ServerID:       "plex-from-db",
				Platform:       "plex",
				Name:           "DB Plex",
				URLEncrypted:   encURL,
				TokenEncrypted: encToken,
				Enabled:        true,
				Source:         "ui",
			},
		},
	}

	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled:  true,
			ServerID: "plex-from-env",
			URL:      "http://plex-env:32400",
			Token:    "env-token",
		},
	}

	configMgr, _ := config.NewServerConfigManager(db, encryptor, cfg)
	sup, _ := NewServerSupervisor(tree, configMgr, nil, nil, nil, nil)

	ctx := context.Background()

	// Start all
	err := sup.StartAll(ctx)
	if err != nil {
		t.Errorf("StartAll() error = %v", err)
	}

	// Check servers are running
	statuses := sup.GetAllServerStatuses()
	if len(statuses) != 2 {
		t.Errorf("GetAllServerStatuses() after StartAll got %d, want 2", len(statuses))
	}
}

func TestServerSupervisor_StartAll_LoadError(t *testing.T) {
	tree, _ := NewSupervisorTree(testLogger(), DefaultTreeConfig())
	encryptor, _ := config.NewCredentialEncryptor("test-secret-key-for-testing")

	// Create mock DB that returns error
	db := &mockConfigDB{
		listErr: errors.New("database error"),
	}

	cfg := &config.Config{}
	configMgr, _ := config.NewServerConfigManager(db, encryptor, cfg)
	sup, _ := NewServerSupervisor(tree, configMgr, nil, nil, nil, nil)

	ctx := context.Background()

	err := sup.StartAll(ctx)
	if err == nil {
		t.Error("StartAll() should return error when config loading fails")
	}
}

func TestServerSupervisor_IsServerRunning(t *testing.T) {
	tree, _ := NewSupervisorTree(testLogger(), DefaultTreeConfig())
	encryptor, _ := config.NewCredentialEncryptor("test-secret-key-for-testing")
	db := &mockConfigDB{}
	cfg := &config.Config{}
	configMgr, _ := config.NewServerConfigManager(db, encryptor, cfg)
	sup, _ := NewServerSupervisor(tree, configMgr, nil, nil, nil, nil)

	// Start the supervisor tree in background (required for remove operations)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = tree.Serve(ctx) }()
	time.Sleep(50 * time.Millisecond)

	// Not running initially
	if sup.IsServerRunning("test-server") {
		t.Error("Server should not be running initially")
	}

	// Add server
	serverCfg := &config.UnifiedServerConfig{
		ID:       "test-plex-1",
		ServerID: "test-server",
		Platform: "plex",
		Name:     "Test",
		URL:      "http://plex:32400",
		Token:    "token",
		Enabled:  true,
	}
	_ = sup.AddServer(ctx, serverCfg)

	// Now running
	if !sup.IsServerRunning("test-server") {
		t.Error("Server should be running after AddServer")
	}

	// Remove
	_ = sup.RemoveServer(ctx, "test-server")

	// Not running anymore
	if sup.IsServerRunning("test-server") {
		t.Error("Server should not be running after RemoveServer")
	}
}

func TestServerSupervisorConfig_Defaults(t *testing.T) {
	cfg := DefaultServerSupervisorConfig()

	if cfg.StartupTimeout != 30*time.Second {
		t.Errorf("StartupTimeout = %v, want 30s", cfg.StartupTimeout)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 10s", cfg.ShutdownTimeout)
	}
}

func TestServerStatus_Fields(t *testing.T) {
	now := time.Now()
	status := ServerStatus{
		ServerID:       "test-id",
		Platform:       "plex",
		Name:           "Test Server",
		Running:        true,
		Status:         "connected",
		LastSyncAt:     &now,
		LastSyncStatus: "success",
		LastError:      "",
		LastErrorAt:    nil,
		StartedAt:      &now,
	}

	// Verify all fields to satisfy govet unusedwrite
	if status.ServerID != "test-id" {
		t.Errorf("ServerID = %s, want test-id", status.ServerID)
	}
	if status.Platform != "plex" {
		t.Errorf("Platform = %s, want plex", status.Platform)
	}
	if status.Name != "Test Server" {
		t.Errorf("Name = %s, want Test Server", status.Name)
	}
	if !status.Running {
		t.Error("Running should be true")
	}
	if status.Status != "connected" {
		t.Errorf("Status = %s, want connected", status.Status)
	}
	if status.LastSyncAt == nil || !status.LastSyncAt.Equal(now) {
		t.Error("LastSyncAt should equal now")
	}
	if status.LastSyncStatus != "success" {
		t.Errorf("LastSyncStatus = %s, want success", status.LastSyncStatus)
	}
	if status.LastError != "" {
		t.Errorf("LastError = %s, want empty", status.LastError)
	}
	if status.LastErrorAt != nil {
		t.Error("LastErrorAt should be nil")
	}
	if status.StartedAt == nil || !status.StartedAt.Equal(now) {
		t.Error("StartedAt should equal now")
	}
}

func TestServerSupervisor_getServerKey(t *testing.T) {
	tree, _ := NewSupervisorTree(testLogger(), DefaultTreeConfig())
	encryptor, _ := config.NewCredentialEncryptor("test-secret-key-for-testing")
	db := &mockConfigDB{}
	cfg := &config.Config{}
	configMgr, _ := config.NewServerConfigManager(db, encryptor, cfg)
	sup, _ := NewServerSupervisor(tree, configMgr, nil, nil, nil, nil)

	tests := []struct {
		name    string
		cfg     *config.UnifiedServerConfig
		wantKey string
	}{
		{
			name: "uses ServerID when present",
			cfg: &config.UnifiedServerConfig{
				ID:       "config-id",
				ServerID: "server-id",
			},
			wantKey: "server-id",
		},
		{
			name: "falls back to ID when ServerID empty",
			cfg: &config.UnifiedServerConfig{
				ID:       "config-id",
				ServerID: "",
			},
			wantKey: "config-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := sup.getServerKey(tt.cfg)
			if key != tt.wantKey {
				t.Errorf("getServerKey() = %s, want %s", key, tt.wantKey)
			}
		})
	}
}
