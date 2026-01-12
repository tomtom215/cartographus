// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package config

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// mockDB implements ServerConfigManagerDB for testing
type mockDB struct {
	servers []models.MediaServer
	getErr  error
	listErr error
	getByID map[string]*models.MediaServer
}

func (m *mockDB) ListMediaServers(ctx context.Context, platform string, enabledOnly bool) ([]models.MediaServer, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}

	if platform == "" && !enabledOnly {
		return m.servers, nil
	}

	result := make([]models.MediaServer, 0)
	for _, s := range m.servers {
		if platform != "" && s.Platform != platform {
			continue
		}
		if enabledOnly && !s.Enabled {
			continue
		}
		result = append(result, s)
	}
	return result, nil
}

func (m *mockDB) GetMediaServer(ctx context.Context, id string) (*models.MediaServer, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.getByID != nil {
		if s, ok := m.getByID[id]; ok {
			return s, nil
		}
	}
	for _, s := range m.servers {
		if s.ID == id {
			return &s, nil
		}
	}
	return nil, errors.New("not found")
}

func TestNewServerConfigManager(t *testing.T) {
	encryptor, _ := NewCredentialEncryptor("test-secret-key-for-testing")
	cfg := &Config{}

	tests := []struct {
		name      string
		db        ServerConfigManagerDB
		encryptor *CredentialEncryptor
		config    *Config
		wantErr   error
	}{
		{
			name:      "valid parameters",
			db:        &mockDB{},
			encryptor: encryptor,
			config:    cfg,
			wantErr:   nil,
		},
		{
			name:      "nil database",
			db:        nil,
			encryptor: encryptor,
			config:    cfg,
			wantErr:   ErrNilDatabase,
		},
		{
			name:      "nil encryptor",
			db:        &mockDB{},
			encryptor: nil,
			config:    cfg,
			wantErr:   ErrNilEncryptor,
		},
		{
			name:      "nil config",
			db:        &mockDB{},
			encryptor: encryptor,
			config:    nil,
			wantErr:   ErrNilConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := NewServerConfigManager(tt.db, tt.encryptor, tt.config)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewServerConfigManager() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && mgr == nil {
				t.Error("NewServerConfigManager() returned nil manager for valid input")
			}
		})
	}
}

func TestServerConfigManager_GetAllServers(t *testing.T) {
	encryptor, _ := NewCredentialEncryptor("test-secret-key-for-testing")

	// Create encrypted test data
	encURL, _ := encryptor.Encrypt("http://db-plex:32400")
	encToken, _ := encryptor.Encrypt("db-token-123")

	tests := []struct {
		name      string
		envConfig *Config
		dbServers []models.MediaServer
		dbListErr error
		wantCount int
		wantErr   bool
	}{
		{
			name: "env vars only",
			envConfig: &Config{
				Plex: PlexConfig{
					Enabled:  true,
					ServerID: "plex-env-1",
					URL:      "http://plex:32400",
					Token:    "env-token",
				},
			},
			dbServers: nil,
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "db servers only",
			envConfig: &Config{},
			dbServers: []models.MediaServer{
				{
					ID:             "db-1",
					ServerID:       "plex-db-1",
					Platform:       models.ServerPlatformPlex,
					Name:           "DB Plex",
					URLEncrypted:   encURL,
					TokenEncrypted: encToken,
					Enabled:        true,
					Source:         models.ServerSourceUI,
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "env vars and db servers merged",
			envConfig: &Config{
				Plex: PlexConfig{
					Enabled:  true,
					ServerID: "plex-env-1",
					URL:      "http://plex:32400",
					Token:    "env-token",
				},
			},
			dbServers: []models.MediaServer{
				{
					ID:             "db-1",
					ServerID:       "plex-db-1",
					Platform:       models.ServerPlatformPlex,
					Name:           "DB Plex",
					URLEncrypted:   encURL,
					TokenEncrypted: encToken,
					Enabled:        true,
					Source:         models.ServerSourceUI,
				},
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "db server with same serverID as env var is skipped",
			envConfig: &Config{
				Plex: PlexConfig{
					Enabled:  true,
					ServerID: "plex-same-id", // Same as DB server
					URL:      "http://plex:32400",
					Token:    "env-token",
				},
			},
			dbServers: []models.MediaServer{
				{
					ID:             "db-1",
					ServerID:       "plex-same-id", // Same as env var
					Platform:       models.ServerPlatformPlex,
					Name:           "DB Plex",
					URLEncrypted:   encURL,
					TokenEncrypted: encToken,
					Enabled:        true,
					Source:         models.ServerSourceUI,
				},
			},
			wantCount: 1, // Only env var, DB skipped due to duplicate
			wantErr:   false,
		},
		{
			name:      "db list error",
			envConfig: &Config{},
			dbServers: nil,
			dbListErr: errors.New("database error"),
			wantCount: 0,
			wantErr:   true,
		},
		{
			name: "multiple platforms",
			envConfig: &Config{
				Plex: PlexConfig{
					Enabled:  true,
					ServerID: "plex-1",
					URL:      "http://plex:32400",
					Token:    "plex-token",
				},
				Jellyfin: JellyfinConfig{
					Enabled:  true,
					ServerID: "jelly-1",
					URL:      "http://jellyfin:8096",
					APIKey:   "jelly-key",
				},
				Tautulli: TautulliConfig{
					Enabled:  true,
					ServerID: "taut-1",
					URL:      "http://tautulli:8181",
					APIKey:   "taut-key",
				},
			},
			dbServers: nil,
			wantCount: 3,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &mockDB{
				servers: tt.dbServers,
				listErr: tt.dbListErr,
			}
			mgr, _ := NewServerConfigManager(db, encryptor, tt.envConfig)

			servers, err := mgr.GetAllServers(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllServers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(servers) != tt.wantCount {
				t.Errorf("GetAllServers() got %d servers, want %d", len(servers), tt.wantCount)
			}
		})
	}
}

func TestServerConfigManager_GetEnabledServers(t *testing.T) {
	encryptor, _ := NewCredentialEncryptor("test-secret-key-for-testing")
	encURL, _ := encryptor.Encrypt("http://db-plex:32400")
	encToken, _ := encryptor.Encrypt("db-token-123")

	cfg := &Config{
		Plex: PlexConfig{
			Enabled:  true,
			ServerID: "plex-env-1",
			URL:      "http://plex:32400",
			Token:    "env-token",
		},
	}

	db := &mockDB{
		servers: []models.MediaServer{
			{
				ID:             "db-1",
				ServerID:       "plex-db-1",
				Platform:       models.ServerPlatformPlex,
				Name:           "Enabled DB Plex",
				URLEncrypted:   encURL,
				TokenEncrypted: encToken,
				Enabled:        true,
				Source:         models.ServerSourceUI,
			},
			{
				ID:             "db-2",
				ServerID:       "plex-db-2",
				Platform:       models.ServerPlatformPlex,
				Name:           "Disabled DB Plex",
				URLEncrypted:   encURL,
				TokenEncrypted: encToken,
				Enabled:        false, // Disabled
				Source:         models.ServerSourceUI,
			},
		},
	}

	mgr, _ := NewServerConfigManager(db, encryptor, cfg)
	servers, err := mgr.GetEnabledServers(context.Background())
	if err != nil {
		t.Fatalf("GetEnabledServers() error = %v", err)
	}

	// Should have 2: 1 env + 1 enabled DB (disabled is filtered)
	if len(servers) != 2 {
		t.Errorf("GetEnabledServers() got %d servers, want 2", len(servers))
	}

	for _, s := range servers {
		if !s.Enabled {
			t.Errorf("GetEnabledServers() returned disabled server: %s", s.Name)
		}
	}
}

func TestServerConfigManager_GetServersByPlatform(t *testing.T) {
	encryptor, _ := NewCredentialEncryptor("test-secret-key-for-testing")
	encURL, _ := encryptor.Encrypt("http://server:8096")
	encToken, _ := encryptor.Encrypt("token-123")

	cfg := &Config{
		Plex: PlexConfig{
			Enabled:  true,
			ServerID: "plex-1",
			URL:      "http://plex:32400",
			Token:    "plex-token",
		},
		Jellyfin: JellyfinConfig{
			Enabled:  true,
			ServerID: "jelly-1",
			URL:      "http://jellyfin:8096",
			APIKey:   "jelly-key",
		},
	}

	db := &mockDB{
		servers: []models.MediaServer{
			{
				ID:             "db-jelly",
				ServerID:       "jelly-db-1",
				Platform:       models.ServerPlatformJellyfin,
				Name:           "DB Jellyfin",
				URLEncrypted:   encURL,
				TokenEncrypted: encToken,
				Enabled:        true,
				Source:         models.ServerSourceUI,
			},
		},
	}

	mgr, _ := NewServerConfigManager(db, encryptor, cfg)

	tests := []struct {
		platform  string
		wantCount int
	}{
		{models.ServerPlatformPlex, 1},
		{models.ServerPlatformJellyfin, 2}, // 1 env + 1 db
		{models.ServerPlatformEmby, 0},
		{models.ServerPlatformTautulli, 0},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			servers, err := mgr.GetServersByPlatform(context.Background(), tt.platform)
			if err != nil {
				t.Fatalf("GetServersByPlatform(%s) error = %v", tt.platform, err)
			}
			if len(servers) != tt.wantCount {
				t.Errorf("GetServersByPlatform(%s) got %d, want %d", tt.platform, len(servers), tt.wantCount)
			}
		})
	}
}

func TestServerConfigManager_GetServer(t *testing.T) {
	encryptor, _ := NewCredentialEncryptor("test-secret-key-for-testing")
	encURL, _ := encryptor.Encrypt("http://db-plex:32400")
	encToken, _ := encryptor.Encrypt("db-token-123")

	cfg := &Config{
		Plex: PlexConfig{
			Enabled:  true,
			ServerID: "plex-env-1",
			URL:      "http://plex:32400",
			Token:    "env-token",
		},
	}

	dbServer := &models.MediaServer{
		ID:             "db-server-id",
		ServerID:       "plex-db-1",
		Platform:       models.ServerPlatformPlex,
		Name:           "DB Plex",
		URLEncrypted:   encURL,
		TokenEncrypted: encToken,
		Enabled:        true,
		Source:         models.ServerSourceUI,
	}

	db := &mockDB{
		servers: []models.MediaServer{*dbServer},
		getByID: map[string]*models.MediaServer{
			"db-server-id": dbServer,
		},
	}

	mgr, _ := NewServerConfigManager(db, encryptor, cfg)

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "find env var server",
			id:      "env-plex-0",
			wantErr: false,
		},
		{
			name:    "find db server",
			id:      "db-server-id",
			wantErr: false,
		},
		{
			name:    "not found",
			id:      "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := mgr.GetServer(context.Background(), tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetServer(%s) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
			if !tt.wantErr && server == nil {
				t.Errorf("GetServer(%s) returned nil server", tt.id)
			}
		})
	}
}

func TestUnifiedServerConfig_ToPlexConfig(t *testing.T) {
	cfg := &UnifiedServerConfig{
		ID:                     "test-id",
		ServerID:               "plex-123",
		Platform:               models.ServerPlatformPlex,
		Name:                   "Test Plex",
		URL:                    "http://plex:32400",
		Token:                  "test-token",
		RealtimeEnabled:        true,
		WebhooksEnabled:        true,
		SessionPollingEnabled:  true,
		SessionPollingInterval: 30 * time.Second,
		Settings: map[string]any{
			"historical_sync":          true,
			"sync_days_back":           float64(90),
			"transcode_monitoring":     true,
			"buffer_health_monitoring": true,
		},
	}

	plexCfg := cfg.ToPlexConfig()
	if plexCfg == nil {
		t.Fatal("ToPlexConfig() returned nil")
	}

	if plexCfg.ServerID != cfg.ServerID {
		t.Errorf("ServerID = %s, want %s", plexCfg.ServerID, cfg.ServerID)
	}
	if plexCfg.URL != cfg.URL {
		t.Errorf("URL = %s, want %s", plexCfg.URL, cfg.URL)
	}
	if plexCfg.Token != cfg.Token {
		t.Errorf("Token = %s, want %s", plexCfg.Token, cfg.Token)
	}
	if !plexCfg.RealtimeEnabled {
		t.Error("RealtimeEnabled should be true")
	}
	if !plexCfg.HistoricalSync {
		t.Error("HistoricalSync should be true from settings")
	}
	if plexCfg.SyncDaysBack != 90 {
		t.Errorf("SyncDaysBack = %d, want 90", plexCfg.SyncDaysBack)
	}
}

func TestUnifiedServerConfig_ToPlexConfig_WrongPlatform(t *testing.T) {
	cfg := &UnifiedServerConfig{
		Platform: models.ServerPlatformJellyfin, // Wrong platform
	}

	if cfg.ToPlexConfig() != nil {
		t.Error("ToPlexConfig() should return nil for non-Plex platform")
	}
}

func TestUnifiedServerConfig_ToJellyfinConfig(t *testing.T) {
	cfg := &UnifiedServerConfig{
		ID:                     "test-id",
		ServerID:               "jelly-123",
		Platform:               models.ServerPlatformJellyfin,
		Name:                   "Test Jellyfin",
		URL:                    "http://jellyfin:8096",
		Token:                  "test-api-key",
		RealtimeEnabled:        true,
		SessionPollingEnabled:  true,
		SessionPollingInterval: 45 * time.Second,
		Settings: map[string]any{
			"user_id":        "user-123",
			"webhook_secret": "secret",
		},
	}

	jellyCfg := cfg.ToJellyfinConfig()
	if jellyCfg == nil {
		t.Fatal("ToJellyfinConfig() returned nil")
	}

	if jellyCfg.ServerID != cfg.ServerID {
		t.Errorf("ServerID = %s, want %s", jellyCfg.ServerID, cfg.ServerID)
	}
	if jellyCfg.APIKey != cfg.Token {
		t.Errorf("APIKey = %s, want %s", jellyCfg.APIKey, cfg.Token)
	}
	if jellyCfg.UserID != "user-123" {
		t.Errorf("UserID = %s, want user-123", jellyCfg.UserID)
	}
}

func TestUnifiedServerConfig_ToEmbyConfig(t *testing.T) {
	cfg := &UnifiedServerConfig{
		ID:              "test-id",
		ServerID:        "emby-123",
		Platform:        models.ServerPlatformEmby,
		Name:            "Test Emby",
		URL:             "http://emby:8096",
		Token:           "test-api-key",
		RealtimeEnabled: true,
		Settings: map[string]any{
			"user_id": "emby-user-123",
		},
	}

	embyCfg := cfg.ToEmbyConfig()
	if embyCfg == nil {
		t.Fatal("ToEmbyConfig() returned nil")
	}

	if embyCfg.ServerID != cfg.ServerID {
		t.Errorf("ServerID = %s, want %s", embyCfg.ServerID, cfg.ServerID)
	}
	if embyCfg.APIKey != cfg.Token {
		t.Errorf("APIKey = %s, want %s", embyCfg.APIKey, cfg.Token)
	}
	if embyCfg.UserID != "emby-user-123" {
		t.Errorf("UserID = %s, want emby-user-123", embyCfg.UserID)
	}
}

func TestUnifiedServerConfig_ToTautulliConfig(t *testing.T) {
	cfg := &UnifiedServerConfig{
		ID:       "test-id",
		ServerID: "taut-123",
		Platform: models.ServerPlatformTautulli,
		Name:     "Test Tautulli",
		URL:      "http://tautulli:8181",
		Token:    "test-api-key",
	}

	tautCfg := cfg.ToTautulliConfig()
	if tautCfg == nil {
		t.Fatal("ToTautulliConfig() returned nil")
	}

	if tautCfg.ServerID != cfg.ServerID {
		t.Errorf("ServerID = %s, want %s", tautCfg.ServerID, cfg.ServerID)
	}
	if tautCfg.URL != cfg.URL {
		t.Errorf("URL = %s, want %s", tautCfg.URL, cfg.URL)
	}
	if tautCfg.APIKey != cfg.Token {
		t.Errorf("APIKey = %s, want %s", tautCfg.APIKey, cfg.Token)
	}
}

func TestUnifiedServerConfig_ToTautulliConfig_WrongPlatform(t *testing.T) {
	cfg := &UnifiedServerConfig{
		Platform: models.ServerPlatformPlex, // Wrong platform
	}

	if cfg.ToTautulliConfig() != nil {
		t.Error("ToTautulliConfig() should return nil for non-Tautulli platform")
	}
}

func TestServerConfigManager_LoadEnvVarServers_PlexServers(t *testing.T) {
	encryptor, _ := NewCredentialEncryptor("test-secret-key-for-testing")

	// Test with PlexServers array (multiple servers)
	cfg := &Config{
		PlexServers: []PlexConfig{
			{
				Enabled:  true,
				ServerID: "plex-1",
				URL:      "http://plex1:32400",
				Token:    "token1",
			},
			{
				Enabled:  true,
				ServerID: "plex-2",
				URL:      "http://plex2:32400",
				Token:    "token2",
			},
		},
	}

	db := &mockDB{}
	mgr, _ := NewServerConfigManager(db, encryptor, cfg)

	servers, err := mgr.GetAllServers(context.Background())
	if err != nil {
		t.Fatalf("GetAllServers() error = %v", err)
	}

	if len(servers) != 2 {
		t.Errorf("GetAllServers() got %d servers, want 2", len(servers))
	}

	// Verify all are marked as immutable
	for _, s := range servers {
		if !s.Immutable {
			t.Errorf("Server %s should be immutable", s.Name)
		}
		if s.Source != models.ServerSourceEnv {
			t.Errorf("Server %s source = %s, want env", s.Name, s.Source)
		}
	}
}

func TestServerConfigManager_DBDecryptionError(t *testing.T) {
	encryptor, _ := NewCredentialEncryptor("test-secret-key-for-testing")

	// DB server with invalid encrypted data
	db := &mockDB{
		servers: []models.MediaServer{
			{
				ID:             "db-bad",
				ServerID:       "bad-server",
				Platform:       models.ServerPlatformPlex,
				Name:           "Bad Encryption",
				URLEncrypted:   "not-valid-base64!!", // Invalid
				TokenEncrypted: "also-invalid",
				Enabled:        true,
				Source:         models.ServerSourceUI,
			},
		},
	}

	cfg := &Config{}
	mgr, _ := NewServerConfigManager(db, encryptor, cfg)

	// Should not return error, just skip the bad server
	servers, err := mgr.GetAllServers(context.Background())
	if err != nil {
		t.Fatalf("GetAllServers() error = %v", err)
	}

	// Bad server should be skipped
	if len(servers) != 0 {
		t.Errorf("GetAllServers() got %d servers, want 0 (bad server skipped)", len(servers))
	}
}

func TestServerConfigManager_ConvertDBServer_Status(t *testing.T) {
	encryptor, _ := NewCredentialEncryptor("test-secret-key-for-testing")
	encURL, _ := encryptor.Encrypt("http://server:32400")
	encToken, _ := encryptor.Encrypt("token")

	tests := []struct {
		name       string
		server     models.MediaServer
		wantStatus string
	}{
		{
			name: "disabled server",
			server: models.MediaServer{
				ID:             "test-1",
				Platform:       models.ServerPlatformPlex,
				URLEncrypted:   encURL,
				TokenEncrypted: encToken,
				Enabled:        false,
			},
			wantStatus: "disabled",
		},
		{
			name: "enabled server with error",
			server: models.MediaServer{
				ID:             "test-2",
				Platform:       models.ServerPlatformPlex,
				URLEncrypted:   encURL,
				TokenEncrypted: encToken,
				Enabled:        true,
				LastError:      "connection failed",
			},
			wantStatus: "error",
		},
		{
			name: "syncing server",
			server: models.MediaServer{
				ID:             "test-3",
				Platform:       models.ServerPlatformPlex,
				URLEncrypted:   encURL,
				TokenEncrypted: encToken,
				Enabled:        true,
				LastSyncStatus: "syncing",
			},
			wantStatus: "syncing",
		},
		{
			name: "connected server",
			server: models.MediaServer{
				ID:             "test-4",
				Platform:       models.ServerPlatformPlex,
				URLEncrypted:   encURL,
				TokenEncrypted: encToken,
				Enabled:        true,
			},
			wantStatus: "connected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &mockDB{servers: []models.MediaServer{tt.server}}
			cfg := &Config{}
			mgr, _ := NewServerConfigManager(db, encryptor, cfg)

			servers, _ := mgr.GetAllServers(context.Background())
			if len(servers) != 1 {
				t.Fatalf("Expected 1 server, got %d", len(servers))
			}
			if servers[0].Status != tt.wantStatus {
				t.Errorf("Status = %s, want %s", servers[0].Status, tt.wantStatus)
			}
		})
	}
}

func TestParseSettings(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
	}{
		{
			name:    "empty string",
			input:   "",
			wantLen: 0,
		},
		{
			name:    "empty json",
			input:   "{}",
			wantLen: 0,
		},
		{
			name:    "valid json (not parsed by parseSettings)",
			input:   `{"key": "value"}`,
			wantLen: 0, // parseSettings returns empty map as placeholder
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSettings(tt.input)
			if len(result) != tt.wantLen {
				t.Errorf("parseSettings(%q) len = %d, want %d", tt.input, len(result), tt.wantLen)
			}
		})
	}
}
