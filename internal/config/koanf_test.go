// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDefaultConfig verifies that defaultConfig() returns proper defaults
func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	// Tautulli defaults (empty - required fields)
	if cfg.Tautulli.URL != "" {
		t.Errorf("Tautulli.URL should be empty by default, got %q", cfg.Tautulli.URL)
	}
	if cfg.Tautulli.APIKey != "" {
		t.Errorf("Tautulli.APIKey should be empty by default, got %q", cfg.Tautulli.APIKey)
	}

	// Plex defaults (disabled)
	if cfg.Plex.Enabled != false {
		t.Errorf("Plex.Enabled should be false by default")
	}
	if cfg.Plex.SyncDaysBack != 365 {
		t.Errorf("Plex.SyncDaysBack = %d, want 365", cfg.Plex.SyncDaysBack)
	}
	if cfg.Plex.SyncInterval != 24*time.Hour {
		t.Errorf("Plex.SyncInterval = %v, want 24h", cfg.Plex.SyncInterval)
	}

	// NATS defaults (enabled)
	if cfg.NATS.Enabled != true {
		t.Errorf("NATS.Enabled should be true by default")
	}
	if cfg.NATS.EventSourcing != true {
		t.Errorf("NATS.EventSourcing should be true by default")
	}
	if cfg.NATS.URL != "nats://127.0.0.1:4222" {
		t.Errorf("NATS.URL = %q, want nats://127.0.0.1:4222", cfg.NATS.URL)
	}
	if cfg.NATS.MaxMemory != 1<<30 {
		t.Errorf("NATS.MaxMemory = %d, want 1GB", cfg.NATS.MaxMemory)
	}
	if cfg.NATS.MaxStore != 10<<30 {
		t.Errorf("NATS.MaxStore = %d, want 10GB", cfg.NATS.MaxStore)
	}

	// Database defaults
	if cfg.Database.Path != "/data/cartographus.duckdb" {
		t.Errorf("Database.Path = %q, want /data/cartographus.duckdb", cfg.Database.Path)
	}
	if cfg.Database.MaxMemory != "2GB" {
		t.Errorf("Database.MaxMemory = %q, want 2GB", cfg.Database.MaxMemory)
	}

	// Sync defaults
	if cfg.Sync.Interval != 5*time.Minute {
		t.Errorf("Sync.Interval = %v, want 5m", cfg.Sync.Interval)
	}
	if cfg.Sync.BatchSize != 1000 {
		t.Errorf("Sync.BatchSize = %d, want 1000", cfg.Sync.BatchSize)
	}

	// Server defaults
	if cfg.Server.Port != 3857 {
		t.Errorf("Server.Port = %d, want 3857", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %q, want 0.0.0.0", cfg.Server.Host)
	}

	// API defaults
	if cfg.API.DefaultPageSize != 20 {
		t.Errorf("API.DefaultPageSize = %d, want 20", cfg.API.DefaultPageSize)
	}
	if cfg.API.MaxPageSize != 100 {
		t.Errorf("API.MaxPageSize = %d, want 100", cfg.API.MaxPageSize)
	}

	// Security defaults
	if cfg.Security.AuthMode != "jwt" {
		t.Errorf("Security.AuthMode = %q, want jwt", cfg.Security.AuthMode)
	}
	if cfg.Security.RateLimitReqs != 100 {
		t.Errorf("Security.RateLimitReqs = %d, want 100", cfg.Security.RateLimitReqs)
	}
	if len(cfg.Security.CORSOrigins) != 1 || cfg.Security.CORSOrigins[0] != "*" {
		t.Errorf("Security.CORSOrigins = %v, want [*]", cfg.Security.CORSOrigins)
	}

	// Logging defaults
	if cfg.Logging.Level != "info" {
		t.Errorf("Logging.Level = %q, want info", cfg.Logging.Level)
	}
}

// TestEnvTransformFunc verifies environment variable name transformations
func TestEnvTransformFunc(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Tautulli
		{"TAUTULLI_URL", "tautulli.url"},
		{"TAUTULLI_API_KEY", "tautulli.api_key"},

		// Plex
		{"ENABLE_PLEX_SYNC", "plex.enabled"},
		{"PLEX_URL", "plex.url"},
		{"PLEX_TOKEN", "plex.token"},
		{"PLEX_SYNC_DAYS_BACK", "plex.sync_days_back"},
		{"ENABLE_PLEX_REALTIME", "plex.realtime_enabled"},
		{"ENABLE_BUFFER_HEALTH_MONITORING", "plex.buffer_health_monitoring"},

		// NATS
		{"NATS_ENABLED", "nats.enabled"},
		{"NATS_URL", "nats.url"},
		{"NATS_EMBEDDED", "nats.embedded_server"},
		{"NATS_MAX_MEMORY", "nats.max_memory"},
		{"NATS_RETENTION_DAYS", "nats.stream_retention_days"},

		// Database
		{"DUCKDB_PATH", "database.path"},
		{"DUCKDB_MAX_MEMORY", "database.max_memory"},
		{"SEED_MOCK_DATA", "database.seed_mock_data"},

		// Server
		{"HTTP_PORT", "server.port"},
		{"HTTP_HOST", "server.host"},
		{"HTTP_TIMEOUT", "server.timeout"},
		{"SERVER_LATITUDE", "server.latitude"},

		// Security
		{"AUTH_MODE", "security.auth_mode"},
		{"JWT_SECRET", "security.jwt_secret"},
		{"ADMIN_USERNAME", "security.admin_username"},
		{"RATE_LIMIT_REQUESTS", "security.rate_limit_reqs"},
		{"DISABLE_RATE_LIMIT", "security.rate_limit_disabled"},

		// Logging
		{"LOG_LEVEL", "logging.level"},

		// Unknown (should return empty)
		{"RANDOM_VAR", ""},
		{"PATH", ""},
		{"HOME", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := envTransformFunc(tt.input)
			if result != tt.expected {
				t.Errorf("envTransformFunc(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestFindConfigFile verifies config file discovery
func TestFindConfigFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save original working directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	t.Run("no config file exists", func(t *testing.T) {
		os.Unsetenv(ConfigPathEnvVar)
		result := findConfigFile()
		if result != "" {
			t.Errorf("findConfigFile() = %q, want empty string", result)
		}
	})

	t.Run("config.yaml exists", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte("test: true"), 0644); err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}
		defer os.Remove(configPath)

		os.Unsetenv(ConfigPathEnvVar)
		result := findConfigFile()
		if result != "config.yaml" {
			t.Errorf("findConfigFile() = %q, want config.yaml", result)
		}
	})

	t.Run("CONFIG_PATH env var takes precedence", func(t *testing.T) {
		// Create a custom config file
		customPath := filepath.Join(tmpDir, "custom_config.yaml")
		if err := os.WriteFile(customPath, []byte("test: true"), 0644); err != nil {
			t.Fatalf("Failed to create custom config file: %v", err)
		}
		defer os.Remove(customPath)

		os.Setenv(ConfigPathEnvVar, customPath)
		defer os.Unsetenv(ConfigPathEnvVar)

		result := findConfigFile()
		if result != customPath {
			t.Errorf("findConfigFile() = %q, want %q", result, customPath)
		}
	})

	t.Run("CONFIG_PATH env var with non-existent file", func(t *testing.T) {
		os.Setenv(ConfigPathEnvVar, "/non/existent/config.yaml")
		defer os.Unsetenv(ConfigPathEnvVar)

		result := findConfigFile()
		// Should fall back to default paths (which don't exist in temp dir)
		if result != "" {
			t.Errorf("findConfigFile() = %q, want empty string", result)
		}
	})
}

// TestLoadWithKoanfEnvVars tests loading configuration from environment variables
func TestLoadWithKoanfEnvVars(t *testing.T) {
	// Clear all environment variables
	os.Clearenv()

	// Set required variables
	os.Setenv("TAUTULLI_URL", "http://test.local:8181")
	os.Setenv("TAUTULLI_API_KEY", "test_api_key_12345")
	os.Setenv("AUTH_MODE", "none")

	// Set some custom values to override defaults
	os.Setenv("HTTP_PORT", "9000")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("SYNC_BATCH_SIZE", "500")

	cfg, err := LoadWithKoanf()
	if err != nil {
		t.Fatalf("LoadWithKoanf() error = %v", err)
	}

	// Verify required values
	if cfg.Tautulli.URL != "http://test.local:8181" {
		t.Errorf("Tautulli.URL = %q, want http://test.local:8181", cfg.Tautulli.URL)
	}
	if cfg.Tautulli.APIKey != "test_api_key_12345" {
		t.Errorf("Tautulli.APIKey = %q, want test_api_key_12345", cfg.Tautulli.APIKey)
	}

	// Verify custom overrides
	if cfg.Server.Port != 9000 {
		t.Errorf("Server.Port = %d, want 9000", cfg.Server.Port)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %q, want debug", cfg.Logging.Level)
	}
	if cfg.Sync.BatchSize != 500 {
		t.Errorf("Sync.BatchSize = %d, want 500", cfg.Sync.BatchSize)
	}

	// Verify defaults are still applied for unset values
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %q, want 0.0.0.0 (default)", cfg.Server.Host)
	}
	if cfg.Database.MaxMemory != "2GB" {
		t.Errorf("Database.MaxMemory = %q, want 2GB (default)", cfg.Database.MaxMemory)
	}
}

// TestLoadWithKoanfConfigFile tests loading configuration from a YAML file
func TestLoadWithKoanfConfigFile(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test config file
	configContent := `
tautulli:
  url: "http://config-file.local:8181"
  api_key: "config_file_api_key"

server:
  port: 8888
  host: "127.0.0.1"

security:
  auth_mode: "none"

logging:
  level: "warn"
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Clear environment and set CONFIG_PATH
	os.Clearenv()
	os.Setenv(ConfigPathEnvVar, configPath)

	cfg, err := LoadWithKoanf()
	if err != nil {
		t.Fatalf("LoadWithKoanf() error = %v", err)
	}

	// Verify values from config file
	if cfg.Tautulli.URL != "http://config-file.local:8181" {
		t.Errorf("Tautulli.URL = %q, want http://config-file.local:8181", cfg.Tautulli.URL)
	}
	if cfg.Server.Port != 8888 {
		t.Errorf("Server.Port = %d, want 8888", cfg.Server.Port)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host = %q, want 127.0.0.1", cfg.Server.Host)
	}
	if cfg.Logging.Level != "warn" {
		t.Errorf("Logging.Level = %q, want warn", cfg.Logging.Level)
	}

	// Verify defaults are still applied for unset values
	if cfg.Database.Path != "/data/cartographus.duckdb" {
		t.Errorf("Database.Path = %q, want /data/cartographus.duckdb (default)", cfg.Database.Path)
	}
}

// TestLoadWithKoanfEnvOverridesFile tests that env vars override config file
func TestLoadWithKoanfEnvOverridesFile(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test config file with some values
	configContent := `
tautulli:
  url: "http://config-file.local:8181"
  api_key: "config_file_api_key"

server:
  port: 8888

security:
  auth_mode: "none"

logging:
  level: "warn"
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Clear environment and set CONFIG_PATH + override values
	os.Clearenv()
	os.Setenv(ConfigPathEnvVar, configPath)
	os.Setenv("HTTP_PORT", "9999")                // Override port from config file
	os.Setenv("LOG_LEVEL", "error")               // Override log level from config file
	os.Setenv("DUCKDB_PATH", "/custom/db.duckdb") // Override a default value

	cfg, err := LoadWithKoanf()
	if err != nil {
		t.Fatalf("LoadWithKoanf() error = %v", err)
	}

	// Verify values from config file (not overridden by env)
	if cfg.Tautulli.URL != "http://config-file.local:8181" {
		t.Errorf("Tautulli.URL = %q, want http://config-file.local:8181 (from file)", cfg.Tautulli.URL)
	}

	// Verify env vars override config file
	if cfg.Server.Port != 9999 {
		t.Errorf("Server.Port = %d, want 9999 (env override)", cfg.Server.Port)
	}
	if cfg.Logging.Level != "error" {
		t.Errorf("Logging.Level = %q, want error (env override)", cfg.Logging.Level)
	}

	// Verify env vars override defaults
	if cfg.Database.Path != "/custom/db.duckdb" {
		t.Errorf("Database.Path = %q, want /custom/db.duckdb (env override)", cfg.Database.Path)
	}
}

// TestLoadWithKoanfValidation tests that validation still works
func TestLoadWithKoanfValidation(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing TAUTULLI_URL when enabled",
			envVars: map[string]string{
				"TAUTULLI_ENABLED": "true",
				"TAUTULLI_API_KEY": "test_key",
				"AUTH_MODE":        "none",
			},
			wantErr: true,
			errMsg:  "TAUTULLI_URL is required when TAUTULLI_ENABLED=true",
		},
		{
			name: "missing TAUTULLI_API_KEY when enabled",
			envVars: map[string]string{
				"TAUTULLI_ENABLED": "true",
				"TAUTULLI_URL":     "http://localhost:8181",
				"AUTH_MODE":        "none",
			},
			wantErr: true,
			errMsg:  "TAUTULLI_API_KEY is required when TAUTULLI_ENABLED=true",
		},
		{
			name: "standalone mode - no Tautulli required",
			envVars: map[string]string{
				"AUTH_MODE": "none",
			},
			wantErr: false,
		},
		{
			name: "JWT mode requires JWT_SECRET",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_key",
				"AUTH_MODE":        "jwt",
			},
			wantErr: true,
			errMsg:  "JWT_SECRET is required",
		},
		{
			name: "valid configuration",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345",
				"AUTH_MODE":        "none",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			_, err := LoadWithKoanf()

			if tt.wantErr {
				if err == nil {
					t.Errorf("LoadWithKoanf() expected error containing %q, got nil", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("LoadWithKoanf() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestLoadBackwardCompatibility ensures Load() still works with legacy env vars
func TestLoadBackwardCompatibility(t *testing.T) {
	os.Clearenv()

	// Set up a complete configuration using legacy env var names
	envVars := map[string]string{
		"TAUTULLI_URL":                     "http://legacy.local:8181",
		"TAUTULLI_API_KEY":                 "legacy_api_key_here",
		"AUTH_MODE":                        "none",
		"ENABLE_PLEX_SYNC":                 "true",
		"PLEX_URL":                         "http://plex.local:32400",
		"PLEX_TOKEN":                       "legacy_plex_token_12345",
		"PLEX_SYNC_DAYS_BACK":              "180",
		"NATS_ENABLED":                     "false",
		"DUCKDB_PATH":                      "/legacy/db.duckdb",
		"DUCKDB_MAX_MEMORY":                "4GB",
		"SYNC_INTERVAL":                    "10m",
		"HTTP_PORT":                        "8080",
		"HTTP_HOST":                        "192.168.1.1",
		"API_DEFAULT_PAGE_SIZE":            "50",
		"LOG_LEVEL":                        "debug",
		"RATE_LIMIT_REQUESTS":              "200",
		"DISABLE_RATE_LIMIT":               "true",
		"ENABLE_BUFFER_HEALTH_MONITORING":  "true",
		"BUFFER_HEALTH_CRITICAL_THRESHOLD": "15.5",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify all values loaded correctly
	if cfg.Tautulli.URL != "http://legacy.local:8181" {
		t.Errorf("Tautulli.URL = %q, want http://legacy.local:8181", cfg.Tautulli.URL)
	}
	if cfg.Plex.Enabled != true {
		t.Errorf("Plex.Enabled = %v, want true", cfg.Plex.Enabled)
	}
	if cfg.Plex.SyncDaysBack != 180 {
		t.Errorf("Plex.SyncDaysBack = %d, want 180", cfg.Plex.SyncDaysBack)
	}
	if cfg.NATS.Enabled != false {
		t.Errorf("NATS.Enabled = %v, want false", cfg.NATS.Enabled)
	}
	if cfg.Database.Path != "/legacy/db.duckdb" {
		t.Errorf("Database.Path = %q, want /legacy/db.duckdb", cfg.Database.Path)
	}
	if cfg.Database.MaxMemory != "4GB" {
		t.Errorf("Database.MaxMemory = %q, want 4GB", cfg.Database.MaxMemory)
	}
	if cfg.Sync.Interval != 10*time.Minute {
		t.Errorf("Sync.Interval = %v, want 10m", cfg.Sync.Interval)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Server.Host != "192.168.1.1" {
		t.Errorf("Server.Host = %q, want 192.168.1.1", cfg.Server.Host)
	}
	if cfg.API.DefaultPageSize != 50 {
		t.Errorf("API.DefaultPageSize = %d, want 50", cfg.API.DefaultPageSize)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %q, want debug", cfg.Logging.Level)
	}
	if cfg.Security.RateLimitReqs != 200 {
		t.Errorf("Security.RateLimitReqs = %d, want 200", cfg.Security.RateLimitReqs)
	}
	if cfg.Security.RateLimitDisabled != true {
		t.Errorf("Security.RateLimitDisabled = %v, want true", cfg.Security.RateLimitDisabled)
	}
	if cfg.Plex.BufferHealthMonitoring != true {
		t.Errorf("Plex.BufferHealthMonitoring = %v, want true", cfg.Plex.BufferHealthMonitoring)
	}
	if cfg.Plex.BufferHealthCriticalThreshold != 15.5 {
		t.Errorf("Plex.BufferHealthCriticalThreshold = %v, want 15.5", cfg.Plex.BufferHealthCriticalThreshold)
	}
}

// TestGetKoanfInstance verifies we can get a Koanf instance for custom use
func TestGetKoanfInstance(t *testing.T) {
	k := GetKoanfInstance()
	if k == nil {
		t.Error("GetKoanfInstance() returned nil")
	}
}
