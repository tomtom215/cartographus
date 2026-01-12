// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLoadConfig tests loading configuration from environment variables
func TestLoadConfig(t *testing.T) {
	t.Parallel()

	t.Run("default values", func(t *testing.T) {
		cleanup := clearEnvVars(t, []string{
			"BACKUP_ENABLED", "BACKUP_DIR", "BACKUP_SCHEDULE_ENABLED",
			"BACKUP_INTERVAL", "BACKUP_PREFERRED_HOUR", "BACKUP_TYPE",
		})
		defer cleanup()

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if !cfg.Enabled {
			t.Error("expected Enabled=true by default")
		}
		if cfg.BackupDir != "/data/backups" {
			t.Errorf("expected BackupDir=/data/backups, got %s", cfg.BackupDir)
		}
	})

	t.Run("custom values from env", func(t *testing.T) {
		cleanup := setEnvVars(t, map[string]string{"BACKUP_ENABLED": "false"})
		defer cleanup()

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if cfg.Enabled {
			t.Error("expected Enabled=false from env")
		}
	})

	t.Run("invalid config returns error", func(t *testing.T) {
		cleanup := setEnvVars(t, map[string]string{
			"BACKUP_ENABLED": "true",
			"BACKUP_DIR":     "relative/path", // Invalid
		})
		defer cleanup()

		_, err := LoadConfig()
		if err == nil {
			t.Error("expected error for invalid config")
		}
	})
}

// TestEnvHelpers tests environment variable helper functions
func TestEnvHelpers(t *testing.T) {
	t.Parallel()

	envHelperTests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "getEnv with value",
			testFunc: func(t *testing.T) {
				cleanup := setEnvVars(t, map[string]string{"TEST_ENV_VAR": "custom_value"})
				defer cleanup()

				result := getEnv("TEST_ENV_VAR", "default")
				if result != "custom_value" {
					t.Errorf("expected custom_value, got %s", result)
				}
			},
		},
		{
			name: "getEnv with default",
			testFunc: func(t *testing.T) {
				os.Unsetenv("TEST_ENV_VAR_MISSING")
				result := getEnv("TEST_ENV_VAR_MISSING", "default_value")
				if result != "default_value" {
					t.Errorf("expected default_value, got %s", result)
				}
			},
		},
		{
			name: "getIntEnv with valid int",
			testFunc: func(t *testing.T) {
				cleanup := setEnvVars(t, map[string]string{"TEST_INT_VAR": "42"})
				defer cleanup()

				result := getIntEnv("TEST_INT_VAR", 10)
				if result != 42 {
					t.Errorf("expected 42, got %d", result)
				}
			},
		},
		{
			name: "getIntEnv with invalid int",
			testFunc: func(t *testing.T) {
				cleanup := setEnvVars(t, map[string]string{"TEST_INT_VAR_INVALID": "not_a_number"})
				defer cleanup()

				result := getIntEnv("TEST_INT_VAR_INVALID", 10)
				if result != 10 {
					t.Errorf("expected default 10, got %d", result)
				}
			},
		},
		{
			name: "getDurationEnv with valid duration",
			testFunc: func(t *testing.T) {
				cleanup := setEnvVars(t, map[string]string{"TEST_DUR_VAR": "2h30m"})
				defer cleanup()

				result := getDurationEnv("TEST_DUR_VAR", time.Hour)
				expected := 2*time.Hour + 30*time.Minute
				if result != expected {
					t.Errorf("expected %v, got %v", expected, result)
				}
			},
		},
		{
			name: "getDurationEnv with invalid duration",
			testFunc: func(t *testing.T) {
				cleanup := setEnvVars(t, map[string]string{"TEST_DUR_VAR_INVALID": "invalid"})
				defer cleanup()

				result := getDurationEnv("TEST_DUR_VAR_INVALID", time.Hour)
				if result != time.Hour {
					t.Errorf("expected default 1h, got %v", result)
				}
			},
		},
	}

	for _, tt := range envHelperTests {
		t.Run(tt.name, tt.testFunc)
	}
}

// TestGetBoolEnv tests boolean environment variable parsing
func TestGetBoolEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		value      string
		defaultVal bool
		expected   bool
	}{
		{"true lowercase", "true", false, true},
		{"True mixed", "True", false, true},
		{"TRUE uppercase", "TRUE", false, true},
		{"1", "1", false, true},
		{"false lowercase", "false", true, false},
		{"False mixed", "False", true, false},
		{"FALSE uppercase", "FALSE", true, false},
		{"0", "0", true, false},
		{"invalid uses default true", "maybe", true, true},
		{"invalid uses default false", "maybe", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setEnvVars(t, map[string]string{"TEST_BOOL_VAR": tt.value})
			defer cleanup()

			result := getBoolEnv("TEST_BOOL_VAR", tt.defaultVal)
			if result != tt.expected {
				t.Errorf("expected %v for %q, got %v", tt.expected, tt.value, result)
			}
		})
	}
}

// TestConfigValidateEdgeCases tests additional config validation edge cases
func TestConfigValidateEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "empty backup dir",
			cfg: &Config{
				Enabled:   true,
				BackupDir: "",
				Retention: RetentionPolicy{MinCount: 1},
			},
			wantErr: true,
		},
		{
			name: "negative preferred hour",
			cfg: &Config{
				Enabled:   true,
				BackupDir: "/tmp/backups",
				Schedule: ScheduleConfig{
					Enabled:       true,
					Interval:      24 * time.Hour,
					PreferredHour: -1,
				},
				Retention: RetentionPolicy{MinCount: 1},
			},
			wantErr: true,
		},
		{
			name: "invalid backup type",
			cfg: &Config{
				Enabled:   true,
				BackupDir: "/tmp/backups",
				Schedule: ScheduleConfig{
					Enabled:    true,
					Interval:   24 * time.Hour,
					BackupType: BackupType("invalid"),
				},
				Retention: RetentionPolicy{MinCount: 1},
			},
			wantErr: true,
		},
		{
			name: "valid zstd compression",
			cfg: &Config{
				Enabled:   true,
				BackupDir: "/tmp/backups",
				Schedule:  ScheduleConfig{Enabled: false},
				Retention: RetentionPolicy{MinCount: 1},
				Compression: CompressionConfig{
					Enabled:   true,
					Level:     5,
					Algorithm: "zstd",
				},
			},
			wantErr: false,
		},
		{
			name: "compression level too low",
			cfg: &Config{
				Enabled:   true,
				BackupDir: "/tmp/backups",
				Schedule:  ScheduleConfig{Enabled: false},
				Retention: RetentionPolicy{MinCount: 1},
				Compression: CompressionConfig{
					Enabled:   true,
					Level:     0,
					Algorithm: "gzip",
				},
			},
			wantErr: true,
		},
		{
			name: "valid encryption config",
			cfg: &Config{
				Enabled:   true,
				BackupDir: "/tmp/backups",
				Schedule:  ScheduleConfig{Enabled: false},
				Retention: RetentionPolicy{MinCount: 1},
				Encryption: EncryptionConfig{
					Enabled: true,
					Key:     "this-is-a-32-char-encryption-key",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestEnsureBackupDir tests backup directory creation
func TestEnsureBackupDir(t *testing.T) {
	t.Parallel()

	t.Run("create new directory", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "backup-test-*")
		defer os.RemoveAll(tempDir)

		cfg := &Config{
			BackupDir: filepath.Join(tempDir, "new", "backups"),
		}

		if err := cfg.EnsureBackupDir(); err != nil {
			t.Fatalf("EnsureBackupDir() error = %v", err)
		}

		if _, err := os.Stat(cfg.BackupDir); os.IsNotExist(err) {
			t.Error("directory was not created")
		}
	})

	t.Run("existing directory", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "backup-test-*")
		defer os.RemoveAll(tempDir)

		cfg := &Config{BackupDir: tempDir}

		if err := cfg.EnsureBackupDir(); err != nil {
			t.Fatalf("EnsureBackupDir() error = %v", err)
		}
	})
}
