// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package backup

import (
	"os"
	"path/filepath"
	"testing"
)

// testEnv holds the common test environment setup
type testEnv struct {
	tempDir   string
	backupDir string
	dbPath    string
	mockDB    *MockDatabase
	cleanup   func()
}

// newTestEnv creates a new test environment with temp directory and mock database
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.duckdb")
	if err := os.WriteFile(dbPath, []byte("test database content"), 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to create mock db file: %v", err)
	}

	return &testEnv{
		tempDir:   tempDir,
		backupDir: filepath.Join(tempDir, "backups"),
		dbPath:    dbPath,
		mockDB: &MockDatabase{
			path:             dbPath,
			playbackCount:    100,
			geolocationCount: 50,
		},
		cleanup: func() { os.RemoveAll(tempDir) },
	}
}

// Close cleans up the test environment
func (e *testEnv) Close() {
	if e.cleanup != nil {
		e.cleanup()
	}
}

// newTestConfig creates a default test configuration
func (e *testEnv) newTestConfig() *Config {
	return &Config{
		Enabled:   true,
		BackupDir: e.backupDir,
		Schedule:  ScheduleConfig{Enabled: false},
		Retention: DefaultRetentionPolicy(),
		Compression: CompressionConfig{
			Enabled:   true,
			Level:     6,
			Algorithm: "gzip",
		},
	}
}

// newTestManager creates a test manager with the default config
func (e *testEnv) newTestManager(t *testing.T) *Manager {
	t.Helper()
	cfg := e.newTestConfig()
	manager, err := NewManager(cfg, e.mockDB)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	return manager
}

// setEnvVars sets environment variables and returns a cleanup function
func setEnvVars(t *testing.T, vars map[string]string) func() {
	t.Helper()
	originalVars := make(map[string]string)

	for key, value := range vars {
		originalVars[key] = os.Getenv(key)
		os.Setenv(key, value)
	}

	return func() {
		for key, original := range originalVars {
			if original == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, original)
			}
		}
	}
}

// clearEnvVars clears the specified environment variables and returns a cleanup function
func clearEnvVars(t *testing.T, keys []string) func() {
	t.Helper()
	originalVars := make(map[string]string)

	for _, key := range keys {
		originalVars[key] = os.Getenv(key)
		os.Unsetenv(key)
	}

	return func() {
		for key, original := range originalVars {
			if original != "" {
				os.Setenv(key, original)
			}
		}
	}
}
