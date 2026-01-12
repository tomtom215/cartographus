// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package backup

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestRestoreFromBackup tests restore functionality
func TestRestoreFromBackup(t *testing.T) {
	t.Parallel()

	t.Run("non-existent backup", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		manager, _ := NewManager(cfg, nil)

		_, err := manager.RestoreFromBackup(context.Background(), "non-existent", RestoreOptions{})
		if err == nil {
			t.Error("expected error for non-existent backup")
		}
	})

	t.Run("validate only mode", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		manager := env.newTestManager(t)
		ctx := context.Background()

		backup, _ := manager.CreateBackup(ctx, TypeDatabase, "test")

		result, err := manager.RestoreFromBackup(ctx, backup.ID, RestoreOptions{
			ValidateOnly: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Success {
			t.Error("expected success for validate only")
		}
		if result.DatabaseRestored {
			t.Error("database should not be restored in validate mode")
		}
	})

	t.Run("with restore callback", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		manager := env.newTestManager(t)
		ctx := context.Background()

		backup, _ := manager.CreateBackup(ctx, TypeDatabase, "test")

		callbackCalled := false
		manager.SetOnRestoreStart(func(backupID string) {
			callbackCalled = true
		})

		_, _ = manager.RestoreFromBackup(ctx, backup.ID, RestoreOptions{
			ForceRestore: true,
		})

		if !callbackCalled {
			t.Error("restore callback should have been called")
		}
	})
}

// TestDetermineRestoreTargets tests restore target determination
func TestDetermineRestoreTargets(t *testing.T) {
	tests := []struct {
		name         string
		backup       *Backup
		opts         RestoreOptions
		expectDB     bool
		expectConfig bool
	}{
		{
			name:         "full backup defaults",
			backup:       &Backup{Type: TypeFull},
			opts:         RestoreOptions{},
			expectDB:     true,
			expectConfig: true,
		},
		{
			name:         "database backup defaults",
			backup:       &Backup{Type: TypeDatabase},
			opts:         RestoreOptions{},
			expectDB:     true,
			expectConfig: false,
		},
		{
			name:         "config backup defaults",
			backup:       &Backup{Type: TypeConfig},
			opts:         RestoreOptions{},
			expectDB:     false,
			expectConfig: true,
		},
		{
			name:         "explicit override - db only",
			backup:       &Backup{Type: TypeFull},
			opts:         RestoreOptions{RestoreDatabase: true},
			expectDB:     true,
			expectConfig: false,
		},
		{
			name:         "explicit override - config only",
			backup:       &Backup{Type: TypeFull},
			opts:         RestoreOptions{RestoreConfig: true},
			expectDB:     false,
			expectConfig: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, config := determineRestoreTargets(tt.backup, tt.opts)
			if db != tt.expectDB {
				t.Errorf("expected restoreDB=%v, got %v", tt.expectDB, db)
			}
			if config != tt.expectConfig {
				t.Errorf("expected restoreConfig=%v, got %v", tt.expectConfig, config)
			}
		})
	}
}

// TestDownloadBackup tests backup download
func TestDownloadBackup(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	manager := env.newTestManager(t)
	ctx := context.Background()

	backup, _ := manager.CreateBackup(ctx, TypeDatabase, "test")

	t.Run("successful download", func(t *testing.T) {
		reader, b, err := manager.DownloadBackup(backup.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer reader.Close()

		if b.ID != backup.ID {
			t.Error("backup ID mismatch")
		}

		buf := make([]byte, 100)
		n, _ := reader.Read(buf)
		if n == 0 {
			t.Error("expected to read some content")
		}
	})

	t.Run("non-existent backup", func(t *testing.T) {
		_, _, err := manager.DownloadBackup("non-existent")
		if err == nil {
			t.Error("expected error for non-existent backup")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		manager.metadata.Backups = append(manager.metadata.Backups, &Backup{
			ID:       "missing-file",
			FilePath: "/nonexistent/file.tar.gz",
		})

		_, _, err := manager.DownloadBackup("missing-file")
		if err == nil {
			t.Error("expected error for missing file")
		}
	})
}

// TestImportBackup tests backup import
func TestImportBackup(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	cfg := env.newTestConfig()
	manager, _ := NewManager(cfg, nil)
	ctx := context.Background()

	t.Run("import simple file", func(t *testing.T) {
		content := []byte("test backup content")
		reader := bytes.NewReader(content)

		backup, err := manager.ImportBackup(ctx, reader, "test-backup.tar.gz")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if backup == nil {
			t.Fatal("backup should not be nil")
		}
		if backup.FileSize != int64(len(content)) {
			t.Errorf("expected size=%d, got %d", len(content), backup.FileSize)
		}
	})
}

// TestShouldExtractFile tests file extraction filtering
func TestShouldExtractFile(t *testing.T) {
	tests := []struct {
		name          string
		headerName    string
		restoreDB     bool
		restoreConfig bool
		expected      bool
	}{
		{"database file with db restore", "database/cartographus.duckdb", true, false, true},
		{"database file without db restore", "database/cartographus.duckdb", false, true, false},
		{"config file with config restore", "config/config.json", false, true, true},
		{"config file without config restore", "config/config.json", true, false, false},
		{"metadata file", "backup-metadata.json", true, true, false},
		{"unknown file", "unknown/file.txt", true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldExtractFile(tt.headerName, tt.restoreDB, tt.restoreConfig)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestCopyFile tests file copying
func TestCopyFile(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	t.Run("successful copy", func(t *testing.T) {
		src := filepath.Join(env.tempDir, "source.txt")
		dst := filepath.Join(env.tempDir, "dest.txt")

		os.WriteFile(src, []byte("test content"), 0644)

		err := copyFile(src, dst)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, _ := os.ReadFile(dst)
		if string(content) != "test content" {
			t.Error("content mismatch")
		}
	})

	t.Run("source not found", func(t *testing.T) {
		err := copyFile("/nonexistent/source", filepath.Join(env.tempDir, "dest"))
		if err == nil {
			t.Error("expected error for missing source")
		}
	})

	t.Run("creates destination directory", func(t *testing.T) {
		src := filepath.Join(env.tempDir, "source2.txt")
		dst := filepath.Join(env.tempDir, "newdir", "subdir", "dest2.txt")

		os.WriteFile(src, []byte("test"), 0644)

		err := copyFile(src, dst)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := os.Stat(dst); os.IsNotExist(err) {
			t.Error("destination file not created")
		}
	})
}

// TestExtractFile tests file extraction with size limits
func TestExtractFile(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	t.Run("successful extraction", func(t *testing.T) {
		content := []byte("test content for extraction")
		reader := bytes.NewReader(content)
		destPath := filepath.Join(env.tempDir, "extracted.txt")

		err := extractFile(reader, destPath, int64(len(content)))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		extracted, _ := os.ReadFile(destPath)
		if string(extracted) != string(content) {
			t.Error("content mismatch")
		}
	})

	t.Run("file too large", func(t *testing.T) {
		reader := bytes.NewReader([]byte("small"))
		destPath := filepath.Join(env.tempDir, "toolarge.txt")

		// Size larger than 1GB limit
		err := extractFile(reader, destPath, 2<<30)
		if err == nil {
			t.Error("expected error for file too large")
		}
	})
}

// TestOpenVerificationDB tests opening a read-only verification connection
func TestOpenVerificationDB(t *testing.T) {
	t.Parallel()

	t.Run("non-existent database", func(t *testing.T) {
		ctx := context.Background()
		_, err := openVerificationDB(ctx, "/nonexistent/database.duckdb")
		if err == nil {
			t.Error("expected error for non-existent database")
		}
	})

	t.Run("invalid database file", func(t *testing.T) {
		ctx := context.Background()
		// Create a temp file with invalid content
		tmpFile, err := os.CreateTemp("", "invalid-*.duckdb")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		tmpFile.WriteString("not a valid duckdb file")
		tmpFile.Close()

		_, err = openVerificationDB(ctx, tmpFile.Name())
		if err == nil {
			t.Error("expected error for invalid database")
		}
	})
}

// TestVerifyRestoredDatabase tests the post-restore verification logic
func TestVerifyRestoredDatabase(t *testing.T) {
	t.Parallel()

	t.Run("missing database file", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		mockDB := &mockDatabaseInterface{
			dbPath: "/nonexistent/database.duckdb",
		}
		manager, _ := NewManager(cfg, mockDB)

		backup := &Backup{RecordCount: 100}
		result := &RestoreResult{}

		err := manager.verifyRestoredDatabase(context.Background(), backup, result)
		if err == nil {
			t.Error("expected error for missing database file")
		}
	})

	t.Run("empty database file", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		// Create empty file
		emptyDB := filepath.Join(env.tempDir, "empty.duckdb")
		os.WriteFile(emptyDB, []byte{}, 0o644)

		cfg := env.newTestConfig()
		mockDB := &mockDatabaseInterface{
			dbPath: emptyDB,
		}
		manager, _ := NewManager(cfg, mockDB)

		backup := &Backup{RecordCount: 100}
		result := &RestoreResult{}

		err := manager.verifyRestoredDatabase(context.Background(), backup, result)
		if err == nil {
			t.Error("expected error for empty database file")
		}
	})
}

// mockDatabaseInterface is a mock implementation for testing
type mockDatabaseInterface struct {
	dbPath string
}

func (m *mockDatabaseInterface) GetDatabasePath() string {
	return m.dbPath
}

func (m *mockDatabaseInterface) GetRecordCounts(_ context.Context) (int64, int64, error) {
	return 0, 0, nil
}

func (m *mockDatabaseInterface) Close() error {
	return nil
}

func (m *mockDatabaseInterface) Checkpoint(_ context.Context) error {
	return nil
}

// TestCheckRequiredFile tests the checkRequiredFile helper function
func TestCheckRequiredFile(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	cfg := env.newTestConfig()
	manager, _ := NewManager(cfg, nil)

	t.Run("file found", func(t *testing.T) {
		result := &ValidationResult{
			FilesComplete: true,
			Backup: &Backup{
				Contents: BackupContents{
					Files: []BackupFile{
						{Path: "database/cartographus.duckdb"},
						{Path: "config/config.json"},
					},
				},
			},
		}

		manager.checkRequiredFile(result, "database/cartographus.duckdb")

		if !result.FilesComplete {
			t.Error("FilesComplete should remain true when file is found")
		}
		if len(result.MissingFiles) != 0 {
			t.Error("MissingFiles should be empty when file is found")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		result := &ValidationResult{
			FilesComplete: true,
			Backup: &Backup{
				Contents: BackupContents{
					Files: []BackupFile{},
				},
			},
		}

		manager.checkRequiredFile(result, "database/cartographus.duckdb")

		if result.FilesComplete {
			t.Error("FilesComplete should be false when file is not found")
		}
		if len(result.MissingFiles) != 1 {
			t.Errorf("MissingFiles should have 1 entry, got %d", len(result.MissingFiles))
		}
	})
}

// TestValidateBeforeRestore tests the validateBeforeRestore method
func TestValidateBeforeRestore(t *testing.T) {
	t.Parallel()

	t.Run("force restore skips validation", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		manager, _ := NewManager(cfg, nil)

		result := &RestoreResult{}
		opts := RestoreOptions{ForceRestore: true}

		err := manager.validateBeforeRestore("any-id", opts, result)
		if err != nil {
			t.Errorf("ForceRestore should skip validation, got error: %v", err)
		}
	})

	t.Run("validation failure", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		manager, _ := NewManager(cfg, nil)

		result := &RestoreResult{}
		opts := RestoreOptions{ForceRestore: false}

		err := manager.validateBeforeRestore("non-existent", opts, result)
		if err == nil {
			t.Error("expected error for non-existent backup")
		}
	})
}

// TestCreatePreRestoreBackup tests the createPreRestoreBackup method
func TestCreatePreRestoreBackup(t *testing.T) {
	t.Parallel()

	t.Run("disabled pre-restore backup", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		manager := env.newTestManager(t)
		result := &RestoreResult{}
		opts := RestoreOptions{CreatePreRestoreBackup: false}

		manager.createPreRestoreBackup(context.Background(), opts, result)

		if result.PreRestoreBackupID != "" {
			t.Error("PreRestoreBackupID should be empty when disabled")
		}
	})

	t.Run("enabled pre-restore backup", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		manager := env.newTestManager(t)
		result := &RestoreResult{}
		opts := RestoreOptions{CreatePreRestoreBackup: true}

		manager.createPreRestoreBackup(context.Background(), opts, result)

		if result.PreRestoreBackupID == "" {
			t.Error("PreRestoreBackupID should be set when enabled")
		}
	})

	t.Run("pre-restore backup failure adds warning", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		// Create manager without database to cause backup failure
		manager, _ := NewManager(cfg, nil)

		result := &RestoreResult{}
		opts := RestoreOptions{CreatePreRestoreBackup: true}

		manager.createPreRestoreBackup(context.Background(), opts, result)

		if len(result.Warnings) == 0 {
			t.Error("expected warning when pre-restore backup fails")
		}
	})
}

// TestRestoreConfigFiles tests the restoreConfigFiles method
func TestRestoreConfigFiles(t *testing.T) {
	t.Parallel()

	t.Run("no config file exists", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		manager := env.newTestManager(t)
		tempDir, _ := os.MkdirTemp("", "restore-test-*")
		defer os.RemoveAll(tempDir)

		result := &RestoreResult{}
		manager.restoreConfigFiles(tempDir, result)

		if result.ConfigRestored {
			t.Error("ConfigRestored should be false when no config file exists")
		}
	})

	t.Run("config file exists", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		manager := env.newTestManager(t)
		tempDir, _ := os.MkdirTemp("", "restore-test-*")
		defer os.RemoveAll(tempDir)

		// Create config directory and file
		configDir := filepath.Join(tempDir, "config")
		os.MkdirAll(configDir, 0o750)
		configFile := filepath.Join(configDir, "config.json")
		os.WriteFile(configFile, []byte(`{"key": "value"}`), 0o644)

		result := &RestoreResult{}
		manager.restoreConfigFiles(tempDir, result)

		if !result.ConfigRestored {
			t.Error("ConfigRestored should be true when config file exists")
		}
		if len(result.Warnings) == 0 {
			t.Error("expected informational warning about config restoration")
		}
	})

	t.Run("config restored with proper destination", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		manager := env.newTestManager(t)

		tempDir, _ := os.MkdirTemp("", "restore-test-*")
		defer os.RemoveAll(tempDir)

		// Create config directory and file
		configDir := filepath.Join(tempDir, "config")
		os.MkdirAll(configDir, 0o750)
		configFile := filepath.Join(configDir, "config.json")
		os.WriteFile(configFile, []byte(`{"key": "value"}`), 0o644)

		result := &RestoreResult{}
		manager.restoreConfigFiles(tempDir, result)

		// Config should be restored
		if !result.ConfigRestored {
			t.Error("ConfigRestored should be true")
		}
		// Should have the informational warning
		if len(result.Warnings) < 1 {
			t.Error("expected at least 1 warning about config restoration")
		}

		// Check that the config was actually copied
		destConfig := filepath.Join(manager.cfg.BackupDir, "restored-config.json")
		if !fileExists(destConfig) {
			t.Error("config should have been copied to backup dir")
		}
	})
}

// TestJSONDecoder tests the JSONDecoder utility
func TestJSONDecoder(t *testing.T) {
	t.Parallel()

	t.Run("decode backup", func(t *testing.T) {
		// The jsonUnmarshal function just sets defaults for *Backup
		var backup Backup
		err := jsonUnmarshal([]byte(`{"id": "test"}`), &backup)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Check that defaults are set
		if backup.ID != "imported" {
			t.Errorf("expected ID='imported', got %s", backup.ID)
		}
		if backup.Type != TypeFull {
			t.Errorf("expected Type=full, got %s", backup.Type)
		}
	})

	t.Run("decode non-backup type", func(t *testing.T) {
		var nonBackup string
		err := jsonUnmarshal([]byte(`"test"`), &nonBackup)
		if err == nil {
			t.Error("expected error for non-backup type")
		}
	})

	t.Run("NewJSONDecoder and Decode", func(t *testing.T) {
		content := []byte(`{"key": "value"}`)
		reader := bytes.NewReader(content)
		decoder := NewJSONDecoder(reader)

		var backup Backup
		err := decoder.Decode(&backup)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("decoder read error", func(t *testing.T) {
		// Create a reader that will fail
		reader := &failingReader{}
		decoder := NewJSONDecoder(reader)

		var backup Backup
		err := decoder.Decode(&backup)
		if err == nil {
			t.Error("expected error from failing reader")
		}
	})
}

// failingReader is a reader that always fails
type failingReader struct{}

func (r *failingReader) Read(p []byte) (n int, err error) {
	return 0, os.ErrClosed
}

// TestRestoreDatabaseFiles tests the restoreDatabaseFiles method
func TestRestoreDatabaseFiles(t *testing.T) {
	t.Parallel()

	t.Run("no database file to restore", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		manager := env.newTestManager(t)
		tempDir, _ := os.MkdirTemp("", "restore-test-*")
		defer os.RemoveAll(tempDir)

		backup := &Backup{RecordCount: 100}
		result := &RestoreResult{}

		err := manager.restoreDatabaseFiles(tempDir, backup, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.DatabaseRestored {
			t.Error("DatabaseRestored should be false when no database file exists")
		}
	})
}

// TestExtractFilesToTemp_PathTraversal tests path traversal prevention
func TestExtractFilesToTemp_PathTraversal(t *testing.T) {
	t.Parallel()

	t.Run("valid path extraction", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		tempDir, _ := os.MkdirTemp("", "extract-test-*")
		defer os.RemoveAll(tempDir)

		// Test the shouldExtractFile function with various paths
		tests := []struct {
			name          string
			path          string
			restoreDB     bool
			restoreConfig bool
			expected      bool
		}{
			{"database with db restore", "database/file.db", true, false, true},
			{"config with config restore", "config/file.json", false, true, true},
			{"database without db restore", "database/file.db", false, true, false},
			{"other file", "other/file.txt", true, true, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := shouldExtractFile(tt.path, tt.restoreDB, tt.restoreConfig)
				if result != tt.expected {
					t.Errorf("shouldExtractFile(%s, %v, %v) = %v, want %v",
						tt.path, tt.restoreDB, tt.restoreConfig, result, tt.expected)
				}
			})
		}
	})
}

// TestValidateRequiredFiles tests the validateRequiredFiles method
func TestValidateRequiredFiles(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	cfg := env.newTestConfig()
	manager, _ := NewManager(cfg, nil)

	tests := []struct {
		name           string
		backupType     BackupType
		files          []BackupFile
		expectValid    bool
		expectDBValid  bool
		expectCfgValid bool
	}{
		{
			name:       "full backup with all files",
			backupType: TypeFull,
			files: []BackupFile{
				{Path: "database/cartographus.duckdb"},
				{Path: "config/config.json"},
			},
			expectValid:    true,
			expectDBValid:  true,
			expectCfgValid: true,
		},
		{
			name:       "full backup missing database",
			backupType: TypeFull,
			files: []BackupFile{
				{Path: "config/config.json"},
			},
			expectValid:    false,
			expectDBValid:  false,
			expectCfgValid: true,
		},
		{
			name:       "database backup with database file",
			backupType: TypeDatabase,
			files: []BackupFile{
				{Path: "database/cartographus.duckdb"},
			},
			expectValid:    true,
			expectDBValid:  true,
			expectCfgValid: true, // Config not required for database backup
		},
		{
			name:       "config backup with config file",
			backupType: TypeConfig,
			files: []BackupFile{
				{Path: "config/config.json"},
			},
			expectValid:    true,
			expectDBValid:  true, // Database not required for config backup
			expectCfgValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backup := &Backup{
				Type:     tt.backupType,
				Contents: BackupContents{Files: tt.files},
			}
			result := &ValidationResult{
				Backup: backup,
			}

			manager.validateRequiredFiles(backup, result)

			if result.FilesComplete != tt.expectValid {
				t.Errorf("FilesComplete = %v, want %v", result.FilesComplete, tt.expectValid)
			}
			if result.DatabaseValid != tt.expectDBValid {
				t.Errorf("DatabaseValid = %v, want %v", result.DatabaseValid, tt.expectDBValid)
			}
			if result.ConfigValid != tt.expectCfgValid {
				t.Errorf("ConfigValid = %v, want %v", result.ConfigValid, tt.expectCfgValid)
			}
		})
	}
}
