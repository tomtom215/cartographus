// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// MockDatabase implements DatabaseInterface for testing
type MockDatabase struct {
	path             string
	playbackCount    int64
	geolocationCount int64
	checkpointError  error
	closeError       error
}

func (m *MockDatabase) GetDatabasePath() string {
	return m.path
}

func (m *MockDatabase) GetRecordCounts(ctx context.Context) (playbacks int64, geolocations int64, err error) {
	return m.playbackCount, m.geolocationCount, nil
}

func (m *MockDatabase) Checkpoint(ctx context.Context) error {
	return m.checkpointError
}

func (m *MockDatabase) Close() error {
	return m.closeError
}

// TestDefaultRetentionPolicy tests the default retention policy values
func TestDefaultRetentionPolicy(t *testing.T) {
	policy := DefaultRetentionPolicy()

	if policy.MinCount != 3 {
		t.Errorf("expected MinCount=3, got %d", policy.MinCount)
	}
	if policy.MaxCount != 50 {
		t.Errorf("expected MaxCount=50, got %d", policy.MaxCount)
	}
	if policy.MaxAgeDays != 90 {
		t.Errorf("expected MaxAgeDays=90, got %d", policy.MaxAgeDays)
	}
	if policy.KeepRecentHours != 24 {
		t.Errorf("expected KeepRecentHours=24, got %d", policy.KeepRecentHours)
	}
	if policy.KeepDailyForDays != 7 {
		t.Errorf("expected KeepDailyForDays=7, got %d", policy.KeepDailyForDays)
	}
	if policy.KeepWeeklyForWeeks != 4 {
		t.Errorf("expected KeepWeeklyForWeeks=4, got %d", policy.KeepWeeklyForWeeks)
	}
	if policy.KeepMonthlyForMonths != 6 {
		t.Errorf("expected KeepMonthlyForMonths=6, got %d", policy.KeepMonthlyForMonths)
	}
}

// TestDefaultScheduleConfig tests the default schedule configuration
func TestDefaultScheduleConfig(t *testing.T) {
	config := DefaultScheduleConfig()

	if !config.Enabled {
		t.Error("expected Enabled=true")
	}
	if config.Interval != 24*time.Hour {
		t.Errorf("expected Interval=24h, got %s", config.Interval)
	}
	if config.PreferredHour != 2 {
		t.Errorf("expected PreferredHour=2, got %d", config.PreferredHour)
	}
	if config.BackupType != TypeFull {
		t.Errorf("expected BackupType=full, got %s", config.BackupType)
	}
	if config.PreSyncBackup {
		t.Error("expected PreSyncBackup=false")
	}
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Enabled:   true,
				BackupDir: "/tmp/backups",
				Schedule: ScheduleConfig{
					Enabled:       true,
					Interval:      24 * time.Hour,
					PreferredHour: 2,
					BackupType:    TypeFull,
				},
				Retention: RetentionPolicy{
					MinCount: 3,
					MaxCount: 50,
				},
				Compression: CompressionConfig{
					Enabled:   true,
					Level:     6,
					Algorithm: "gzip",
				},
			},
			wantErr: false,
		},
		{
			name: "disabled backups - skip validation",
			config: &Config{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "invalid backup dir - relative path",
			config: &Config{
				Enabled:   true,
				BackupDir: "relative/path",
				Schedule:  ScheduleConfig{Enabled: false},
				Retention: RetentionPolicy{MinCount: 1},
			},
			wantErr: true,
		},
		{
			name: "invalid schedule interval",
			config: &Config{
				Enabled:   true,
				BackupDir: "/tmp/backups",
				Schedule: ScheduleConfig{
					Enabled:  true,
					Interval: 30 * time.Minute, // Too short
				},
				Retention: RetentionPolicy{MinCount: 1},
			},
			wantErr: true,
		},
		{
			name: "invalid preferred hour",
			config: &Config{
				Enabled:   true,
				BackupDir: "/tmp/backups",
				Schedule: ScheduleConfig{
					Enabled:       true,
					Interval:      24 * time.Hour,
					PreferredHour: 25, // Invalid
				},
				Retention: RetentionPolicy{MinCount: 1},
			},
			wantErr: true,
		},
		{
			name: "invalid min count",
			config: &Config{
				Enabled:   true,
				BackupDir: "/tmp/backups",
				Schedule:  ScheduleConfig{Enabled: false},
				Retention: RetentionPolicy{MinCount: 0}, // Invalid
			},
			wantErr: true,
		},
		{
			name: "max count less than min count",
			config: &Config{
				Enabled:   true,
				BackupDir: "/tmp/backups",
				Schedule:  ScheduleConfig{Enabled: false},
				Retention: RetentionPolicy{
					MinCount: 5,
					MaxCount: 3, // Less than MinCount
				},
			},
			wantErr: true,
		},
		{
			name: "invalid compression level",
			config: &Config{
				Enabled:   true,
				BackupDir: "/tmp/backups",
				Schedule:  ScheduleConfig{Enabled: false},
				Retention: RetentionPolicy{MinCount: 1},
				Compression: CompressionConfig{
					Enabled: true,
					Level:   10, // Invalid
				},
			},
			wantErr: true,
		},
		{
			name: "invalid compression algorithm",
			config: &Config{
				Enabled:   true,
				BackupDir: "/tmp/backups",
				Schedule:  ScheduleConfig{Enabled: false},
				Retention: RetentionPolicy{MinCount: 1},
				Compression: CompressionConfig{
					Enabled:   true,
					Level:     6,
					Algorithm: "lz4", // Not supported
				},
			},
			wantErr: true,
		},
		{
			name: "encryption key too short",
			config: &Config{
				Enabled:   true,
				BackupDir: "/tmp/backups",
				Schedule:  ScheduleConfig{Enabled: false},
				Retention: RetentionPolicy{MinCount: 1},
				Encryption: EncryptionConfig{
					Enabled: true,
					Key:     "short", // Too short
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestManagerCreation tests backup manager creation
func TestManagerCreation(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock database file
	dbPath := filepath.Join(tempDir, "test.duckdb")
	if err := os.WriteFile(dbPath, []byte("test data"), 0644); err != nil {
		t.Fatalf("failed to create mock db file: %v", err)
	}

	mockDB := &MockDatabase{
		path:             dbPath,
		playbackCount:    100,
		geolocationCount: 50,
	}

	cfg := &Config{
		Enabled:   true,
		BackupDir: filepath.Join(tempDir, "backups"),
		Schedule:  DefaultScheduleConfig(),
		Retention: DefaultRetentionPolicy(),
		Compression: CompressionConfig{
			Enabled:   true,
			Level:     6,
			Algorithm: "gzip",
		},
	}

	manager, err := NewManager(cfg, mockDB)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	if manager == nil {
		t.Fatal("manager is nil")
	}

	// Check backup directory was created
	if _, err := os.Stat(cfg.BackupDir); os.IsNotExist(err) {
		t.Error("backup directory was not created")
	}
}

// TestCreateBackup tests backup creation
func TestCreateBackup(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock database file
	dbPath := filepath.Join(tempDir, "test.duckdb")
	if err := os.WriteFile(dbPath, []byte("test database content"), 0644); err != nil {
		t.Fatalf("failed to create mock db file: %v", err)
	}

	mockDB := &MockDatabase{
		path:             dbPath,
		playbackCount:    100,
		geolocationCount: 50,
	}

	cfg := &Config{
		Enabled:   true,
		BackupDir: filepath.Join(tempDir, "backups"),
		Schedule:  ScheduleConfig{Enabled: false},
		Retention: DefaultRetentionPolicy(),
		Compression: CompressionConfig{
			Enabled:   true,
			Level:     6,
			Algorithm: "gzip",
		},
	}

	manager, err := NewManager(cfg, mockDB)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Create a backup
	ctx := context.Background()
	backup, err := manager.CreateBackup(ctx, TypeFull, "Test backup")
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	// Verify backup metadata
	if backup.ID == "" {
		t.Error("backup ID is empty")
	}
	if backup.Type != TypeFull {
		t.Errorf("expected type=full, got %s", backup.Type)
	}
	if backup.Status != StatusCompleted {
		t.Errorf("expected status=completed, got %s", backup.Status)
	}
	if backup.Trigger != TriggerManual {
		t.Errorf("expected trigger=manual, got %s", backup.Trigger)
	}
	if backup.Notes != "Test backup" {
		t.Errorf("expected notes='Test backup', got '%s'", backup.Notes)
	}
	if backup.Compressed != true {
		t.Error("expected compressed=true")
	}
	if backup.FileSize == 0 {
		t.Error("file size is 0")
	}
	if backup.Checksum == "" {
		t.Error("checksum is empty")
	}

	// Verify file exists
	if _, err := os.Stat(backup.FilePath); os.IsNotExist(err) {
		t.Error("backup file does not exist")
	}
}

// TestListBackups tests backup listing with filters
func TestListBackups(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock database file
	dbPath := filepath.Join(tempDir, "test.duckdb")
	if err := os.WriteFile(dbPath, []byte("test data"), 0644); err != nil {
		t.Fatalf("failed to create mock db file: %v", err)
	}

	mockDB := &MockDatabase{
		path:             dbPath,
		playbackCount:    100,
		geolocationCount: 50,
	}

	cfg := &Config{
		Enabled:   true,
		BackupDir: filepath.Join(tempDir, "backups"),
		Schedule:  ScheduleConfig{Enabled: false},
		Retention: DefaultRetentionPolicy(),
		Compression: CompressionConfig{
			Enabled:   true,
			Level:     6,
			Algorithm: "gzip",
		},
	}

	manager, err := NewManager(cfg, mockDB)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Create multiple backups
	_, err = manager.CreateBackup(ctx, TypeFull, "Full backup 1")
	if err != nil {
		t.Fatalf("failed to create backup 1: %v", err)
	}

	_, err = manager.CreateBackup(ctx, TypeDatabase, "DB backup 1")
	if err != nil {
		t.Fatalf("failed to create backup 2: %v", err)
	}

	_, err = manager.CreateBackup(ctx, TypeConfig, "Config backup 1")
	if err != nil {
		t.Fatalf("failed to create backup 3: %v", err)
	}

	// List all backups
	backups, err := manager.ListBackups(BackupListOptions{
		Limit:    100,
		SortDesc: true,
	})
	if err != nil {
		t.Fatalf("failed to list backups: %v", err)
	}

	if len(backups) != 3 {
		t.Errorf("expected 3 backups, got %d", len(backups))
	}

	// Filter by type
	typeFilter := TypeFull
	backups, err = manager.ListBackups(BackupListOptions{
		Type:     &typeFilter,
		Limit:    100,
		SortDesc: true,
	})
	if err != nil {
		t.Fatalf("failed to list backups with type filter: %v", err)
	}

	if len(backups) != 1 {
		t.Errorf("expected 1 full backup, got %d", len(backups))
	}

	// Test pagination
	backups, err = manager.ListBackups(BackupListOptions{
		Limit:    2,
		Offset:   0,
		SortDesc: true,
	})
	if err != nil {
		t.Fatalf("failed to list backups with pagination: %v", err)
	}

	if len(backups) != 2 {
		t.Errorf("expected 2 backups with limit=2, got %d", len(backups))
	}
}

// TestValidateBackup tests backup validation
func TestValidateBackup(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock database file
	dbPath := filepath.Join(tempDir, "test.duckdb")
	if err := os.WriteFile(dbPath, []byte("test data"), 0644); err != nil {
		t.Fatalf("failed to create mock db file: %v", err)
	}

	mockDB := &MockDatabase{
		path:             dbPath,
		playbackCount:    100,
		geolocationCount: 50,
	}

	cfg := &Config{
		Enabled:   true,
		BackupDir: filepath.Join(tempDir, "backups"),
		Schedule:  ScheduleConfig{Enabled: false},
		Retention: DefaultRetentionPolicy(),
		Compression: CompressionConfig{
			Enabled:   true,
			Level:     6,
			Algorithm: "gzip",
		},
	}

	manager, err := NewManager(cfg, mockDB)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Create a backup
	backup, err := manager.CreateBackup(ctx, TypeFull, "Test backup")
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	// Validate the backup
	result, err := manager.ValidateBackup(backup.ID)
	if err != nil {
		t.Fatalf("failed to validate backup: %v", err)
	}

	if !result.Valid {
		t.Errorf("expected backup to be valid, errors: %v", result.Errors)
	}
	if !result.ChecksumValid {
		t.Error("expected checksum to be valid")
	}
	if !result.ArchiveReadable {
		t.Error("expected archive to be readable")
	}
}

// TestDeleteBackup tests backup deletion
func TestDeleteBackup(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock database file
	dbPath := filepath.Join(tempDir, "test.duckdb")
	if err := os.WriteFile(dbPath, []byte("test data"), 0644); err != nil {
		t.Fatalf("failed to create mock db file: %v", err)
	}

	mockDB := &MockDatabase{
		path:             dbPath,
		playbackCount:    100,
		geolocationCount: 50,
	}

	cfg := &Config{
		Enabled:   true,
		BackupDir: filepath.Join(tempDir, "backups"),
		Schedule:  ScheduleConfig{Enabled: false},
		Retention: DefaultRetentionPolicy(),
		Compression: CompressionConfig{
			Enabled:   true,
			Level:     6,
			Algorithm: "gzip",
		},
	}

	manager, err := NewManager(cfg, mockDB)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Create a backup
	backup, err := manager.CreateBackup(ctx, TypeFull, "Test backup")
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	backupPath := backup.FilePath

	// Verify file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatal("backup file does not exist before delete")
	}

	// Delete the backup
	if err := manager.DeleteBackup(backup.ID); err != nil {
		t.Fatalf("failed to delete backup: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("backup file still exists after delete")
	}

	// Verify backup is not in list
	backups, _ := manager.ListBackups(BackupListOptions{Limit: 100})
	for _, b := range backups {
		if b.ID == backup.ID {
			t.Error("deleted backup still in list")
		}
	}
}

// TestGetStats tests backup statistics
func TestGetStats(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock database file
	dbPath := filepath.Join(tempDir, "test.duckdb")
	if err := os.WriteFile(dbPath, []byte("test data"), 0644); err != nil {
		t.Fatalf("failed to create mock db file: %v", err)
	}

	mockDB := &MockDatabase{
		path:             dbPath,
		playbackCount:    100,
		geolocationCount: 50,
	}

	cfg := &Config{
		Enabled:   true,
		BackupDir: filepath.Join(tempDir, "backups"),
		Schedule:  ScheduleConfig{Enabled: false},
		Retention: DefaultRetentionPolicy(),
		Compression: CompressionConfig{
			Enabled:   true,
			Level:     6,
			Algorithm: "gzip",
		},
	}

	manager, err := NewManager(cfg, mockDB)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Create some backups
	_, _ = manager.CreateBackup(ctx, TypeFull, "Backup 1")
	_, _ = manager.CreateBackup(ctx, TypeDatabase, "Backup 2")

	// Get stats
	stats, err := manager.GetStats()
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if stats.TotalCount != 2 {
		t.Errorf("expected TotalCount=2, got %d", stats.TotalCount)
	}
	if stats.CountByType[TypeFull] != 1 {
		t.Errorf("expected 1 full backup, got %d", stats.CountByType[TypeFull])
	}
	if stats.CountByType[TypeDatabase] != 1 {
		t.Errorf("expected 1 database backup, got %d", stats.CountByType[TypeDatabase])
	}
	if stats.SuccessRate != 100.0 {
		t.Errorf("expected SuccessRate=100.0, got %f", stats.SuccessRate)
	}
}

// TestRetentionPolicy tests retention policy application
func TestRetentionPolicy(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock database file
	dbPath := filepath.Join(tempDir, "test.duckdb")
	if err := os.WriteFile(dbPath, []byte("test data"), 0644); err != nil {
		t.Fatalf("failed to create mock db file: %v", err)
	}

	mockDB := &MockDatabase{
		path:             dbPath,
		playbackCount:    100,
		geolocationCount: 50,
	}

	cfg := &Config{
		Enabled:   true,
		BackupDir: filepath.Join(tempDir, "backups"),
		Schedule:  ScheduleConfig{Enabled: false},
		Retention: RetentionPolicy{
			MinCount:        2,
			MaxCount:        5,
			MaxAgeDays:      30,
			KeepRecentHours: 24,
		},
		Compression: CompressionConfig{
			Enabled:   true,
			Level:     6,
			Algorithm: "gzip",
		},
	}

	manager, err := NewManager(cfg, mockDB)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Get retention policy
	policy := manager.GetRetentionPolicy()
	if policy.MinCount != 2 {
		t.Errorf("expected MinCount=2, got %d", policy.MinCount)
	}

	// Update retention policy
	newPolicy := RetentionPolicy{
		MinCount:   3,
		MaxCount:   10,
		MaxAgeDays: 60,
	}

	if err := manager.SetRetentionPolicy(newPolicy); err != nil {
		t.Fatalf("failed to set retention policy: %v", err)
	}

	policy = manager.GetRetentionPolicy()
	if policy.MinCount != 3 {
		t.Errorf("expected MinCount=3 after update, got %d", policy.MinCount)
	}
}

// TestBackupTypes tests different backup types
func TestBackupTypes(t *testing.T) {
	tests := []struct {
		name     string
		backup   BackupType
		expected string
	}{
		{"full backup", TypeFull, "full"},
		{"database backup", TypeDatabase, "database"},
		{"config backup", TypeConfig, "config"},
		{"incremental backup", TypeIncremental, "incremental"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.backup) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.backup)
			}
		})
	}
}

// TestBackupStatus tests backup status values
func TestBackupStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   BackupStatus
		expected string
	}{
		{"pending", StatusPending, "pending"},
		{"in progress", StatusInProgress, "in_progress"},
		{"completed", StatusCompleted, "completed"},
		{"failed", StatusFailed, "failed"},
		{"corrupted", StatusCorrupted, "corrupted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.status)
			}
		})
	}
}

// TestBackupTrigger tests backup trigger values
func TestBackupTrigger(t *testing.T) {
	tests := []struct {
		name     string
		trigger  BackupTrigger
		expected string
	}{
		{"manual", TriggerManual, "manual"},
		{"scheduled", TriggerScheduled, "scheduled"},
		{"pre-sync", TriggerPreSync, "pre_sync"},
		{"pre-restore", TriggerPreRestore, "pre_restore"},
		{"retention", TriggerRetention, "retention"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.trigger) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.trigger)
			}
		})
	}
}

// TestCalculateNextBackupTime tests next backup time calculation
func TestCalculateNextBackupTime(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mockDB := &MockDatabase{path: filepath.Join(tempDir, "test.duckdb")}
	if err := os.WriteFile(mockDB.path, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write mock db: %v", err)
	}

	cfg := &Config{
		Enabled:   true,
		BackupDir: filepath.Join(tempDir, "backups"),
		Schedule: ScheduleConfig{
			Enabled:       true,
			Interval:      24 * time.Hour,
			PreferredHour: 2, // 2 AM
			BackupType:    TypeFull,
		},
		Retention: DefaultRetentionPolicy(),
	}

	manager, err := NewManager(cfg, mockDB)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	nextBackup := manager.calculateNextBackupTime()

	// The next backup should be in the future
	if !nextBackup.After(time.Now()) {
		t.Error("next backup time should be in the future")
	}

	// The hour should match preferred hour
	if nextBackup.Hour() != 2 {
		t.Errorf("expected hour=2, got %d", nextBackup.Hour())
	}
}
