// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package backup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNewManager_EdgeCases tests manager creation edge cases
func TestNewManager_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("nil config", func(t *testing.T) {
		_, err := NewManager(nil, nil)
		if err == nil {
			t.Error("expected error for nil config")
		}
	})

	t.Run("disabled backups skip dir creation", func(t *testing.T) {
		cfg := &Config{
			Enabled:   false,
			BackupDir: "/nonexistent/path",
		}

		manager, err := NewManager(cfg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if manager == nil {
			t.Error("manager should not be nil")
		}
	})

	t.Run("nil database allowed", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := &Config{
			Enabled:   true,
			BackupDir: env.backupDir,
			Retention: DefaultRetentionPolicy(),
		}

		manager, err := NewManager(cfg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if manager == nil {
			t.Error("manager should not be nil")
		}
	})

	t.Run("load existing metadata", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		os.MkdirAll(env.backupDir, 0750)
		metadataContent := `{"backups":[],"retention":{"min_count":5}}`
		os.WriteFile(filepath.Join(env.backupDir, "metadata.json"), []byte(metadataContent), 0600)

		cfg := &Config{
			Enabled:   true,
			BackupDir: env.backupDir,
			Retention: DefaultRetentionPolicy(),
		}

		manager, err := NewManager(cfg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if manager.metadata == nil {
			t.Error("metadata should be loaded")
		}
	})
}

// TestManagerStartStop tests scheduler start/stop
func TestManagerStartStop(t *testing.T) {
	env := newTestEnv(t)
	defer env.Close()

	cfg := &Config{
		Enabled:   true,
		BackupDir: env.backupDir,
		Schedule: ScheduleConfig{
			Enabled:       true,
			Interval:      time.Hour,
			PreferredHour: 2,
			BackupType:    TypeFull,
		},
		Retention: DefaultRetentionPolicy(),
	}

	manager, _ := NewManager(cfg, env.mockDB)
	ctx := context.Background()

	t.Run("start scheduler", func(t *testing.T) {
		err := manager.Start(ctx)
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		// Second start should fail
		err = manager.Start(ctx)
		if err == nil {
			t.Error("expected error on second Start()")
		}
	})

	t.Run("stop scheduler", func(t *testing.T) {
		err := manager.Stop()
		if err != nil {
			t.Fatalf("Stop() error = %v", err)
		}

		// Second stop should be no-op
		err = manager.Stop()
		if err != nil {
			t.Fatalf("second Stop() error = %v", err)
		}
	})

	t.Run("start disabled scheduler", func(t *testing.T) {
		cfg2 := &Config{
			Enabled:   true,
			BackupDir: filepath.Join(env.tempDir, "backups2"),
			Schedule:  ScheduleConfig{Enabled: false},
			Retention: DefaultRetentionPolicy(),
		}
		manager2, _ := NewManager(cfg2, env.mockDB)

		err := manager2.Start(ctx)
		if err != nil {
			t.Fatalf("Start() with disabled schedule should not error: %v", err)
		}
	})

	t.Run("start with disabled backups", func(t *testing.T) {
		cfg3 := &Config{
			Enabled:   false,
			BackupDir: filepath.Join(env.tempDir, "backups3"),
		}
		manager3, _ := NewManager(cfg3, nil)

		err := manager3.Start(ctx)
		if err != nil {
			t.Fatalf("Start() with disabled backups should not error: %v", err)
		}
	})
}

// TestCreateBackup_EdgeCases tests backup creation edge cases
func TestCreateBackup_EdgeCases(t *testing.T) {
	t.Run("backup disabled", func(t *testing.T) {
		cfg := &Config{Enabled: false}
		manager, _ := NewManager(cfg, nil)

		_, err := manager.CreateBackup(context.Background(), TypeFull, "test")
		if err == nil {
			t.Error("expected error when backups disabled")
		}
	})

	t.Run("nil database for database backup", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		manager, _ := NewManager(cfg, nil)

		backup, err := manager.CreateBackup(context.Background(), TypeDatabase, "test")
		if err == nil {
			t.Error("expected error for nil database")
		}
		if backup != nil && backup.Status != StatusFailed {
			t.Error("backup should be marked as failed")
		}
	})

	t.Run("config only backup without database", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		manager, _ := NewManager(cfg, nil)

		backup, err := manager.CreateBackup(context.Background(), TypeConfig, "test")
		if err != nil {
			t.Fatalf("config backup should succeed without database: %v", err)
		}
		if backup.Status != StatusCompleted {
			t.Error("backup should be completed")
		}
	})

	t.Run("checkpoint error is warning only", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		env.mockDB.checkpointError = errors.New("checkpoint failed")
		cfg := env.newTestConfig()
		manager, _ := NewManager(cfg, env.mockDB)

		backup, err := manager.CreateBackup(context.Background(), TypeDatabase, "test")
		if err != nil {
			t.Fatalf("backup should succeed despite checkpoint error: %v", err)
		}
		if backup.Status != StatusCompleted {
			t.Error("backup should be completed")
		}
	})

	t.Run("uncompressed backup", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		cfg.Compression.Enabled = false
		manager, _ := NewManager(cfg, env.mockDB)

		backup, err := manager.CreateBackup(context.Background(), TypeDatabase, "test")
		if err != nil {
			t.Fatalf("uncompressed backup failed: %v", err)
		}
		if backup.Compressed {
			t.Error("backup should not be compressed")
		}
		if !strings.HasSuffix(backup.FilePath, ".tar") {
			t.Error("uncompressed backup should have .tar extension")
		}
	})

	t.Run("backup with callback", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		manager, _ := NewManager(cfg, env.mockDB)

		callbackCalled := false
		manager.SetOnBackupComplete(func(backup *Backup) {
			callbackCalled = true
		})

		_, err := manager.CreateBackup(context.Background(), TypeDatabase, "test")
		if err != nil {
			t.Fatalf("backup failed: %v", err)
		}
		if !callbackCalled {
			t.Error("callback should have been called")
		}
	})
}

// TestCreatePreSyncBackup tests pre-sync backup creation
func TestCreatePreSyncBackup(t *testing.T) {
	t.Parallel()

	t.Run("disabled pre-sync backup", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		cfg.Schedule.PreSyncBackup = false
		manager, _ := NewManager(cfg, nil)

		backup, err := manager.CreatePreSyncBackup(context.Background())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if backup != nil {
			t.Error("backup should be nil when pre-sync disabled")
		}
	})

	t.Run("enabled pre-sync backup", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		cfg.Schedule.PreSyncBackup = true
		manager, _ := NewManager(cfg, env.mockDB)

		backup, err := manager.CreatePreSyncBackup(context.Background())
		if err != nil {
			t.Fatalf("pre-sync backup failed: %v", err)
		}
		if backup == nil {
			t.Fatal("backup should not be nil")
		}
		if backup.Trigger != TriggerPreSync {
			t.Errorf("expected trigger=pre_sync, got %s", backup.Trigger)
		}
	})
}

// TestGetBackup tests backup retrieval
func TestGetBackup(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	manager := env.newTestManager(t)
	ctx := context.Background()

	created, _ := manager.CreateBackup(ctx, TypeFull, "test")

	t.Run("existing backup", func(t *testing.T) {
		backup, err := manager.GetBackup(created.ID)
		if err != nil {
			t.Fatalf("GetBackup() error = %v", err)
		}
		if backup.ID != created.ID {
			t.Error("backup ID mismatch")
		}
	})

	t.Run("non-existent backup", func(t *testing.T) {
		_, err := manager.GetBackup("non-existent-id")
		if err == nil {
			t.Error("expected error for non-existent backup")
		}
	})
}

// TestListBackups_EdgeCases tests backup listing edge cases
func TestListBackups_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("nil metadata", func(t *testing.T) {
		cfg := &Config{Enabled: false}
		manager, _ := NewManager(cfg, nil)
		manager.metadata = nil

		backups, err := manager.ListBackups(BackupListOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(backups) != 0 {
			t.Error("expected empty list")
		}
	})

	t.Run("offset beyond list", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		manager, _ := NewManager(cfg, nil)

		backups, err := manager.ListBackups(BackupListOptions{Offset: 100, Limit: 10})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(backups) != 0 {
			t.Error("expected empty list")
		}
	})

	t.Run("sort ascending", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		manager := env.newTestManager(t)
		ctx := context.Background()

		manager.CreateBackup(ctx, TypeFull, "first")
		time.Sleep(10 * time.Millisecond)
		manager.CreateBackup(ctx, TypeFull, "second")

		backups, _ := manager.ListBackups(BackupListOptions{SortDesc: false, Limit: 10})

		if len(backups) >= 2 && backups[0].Notes != "first" {
			t.Error("expected ascending sort")
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		manager := env.newTestManager(t)
		manager.CreateBackup(context.Background(), TypeFull, "test")

		status := StatusCompleted
		backups, _ := manager.ListBackups(BackupListOptions{Status: &status, Limit: 10})

		for _, b := range backups {
			if b.Status != StatusCompleted {
				t.Error("expected only completed backups")
			}
		}
	})

	t.Run("filter by date range", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		manager, _ := NewManager(cfg, nil)

		now := time.Now()
		manager.metadata.Backups = []*Backup{
			{ID: "1", Status: StatusCompleted, CreatedAt: now.Add(-48 * time.Hour)},
			{ID: "2", Status: StatusCompleted, CreatedAt: now.Add(-24 * time.Hour)},
			{ID: "3", Status: StatusCompleted, CreatedAt: now},
		}

		startDate := now.Add(-36 * time.Hour)
		endDate := now.Add(-12 * time.Hour)

		backups, _ := manager.ListBackups(BackupListOptions{
			StartDate: &startDate,
			EndDate:   &endDate,
			Limit:     10,
		})

		if len(backups) != 1 {
			t.Errorf("expected 1 backup in date range, got %d", len(backups))
		}
	})
}

// TestDeleteBackup_EdgeCases tests backup deletion edge cases
func TestDeleteBackup_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("non-existent backup", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		manager, _ := NewManager(cfg, nil)

		err := manager.DeleteBackup("non-existent-id")
		if err == nil {
			t.Error("expected error for non-existent backup")
		}
	})

	t.Run("missing file still removes from metadata", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		manager, _ := NewManager(cfg, nil)

		manager.metadata.Backups = []*Backup{
			{ID: "test-id", FilePath: "/nonexistent/file.tar.gz", Status: StatusCompleted},
		}

		err := manager.DeleteBackup("test-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(manager.metadata.Backups) != 0 {
			t.Error("backup should be removed from metadata")
		}
	})
}

// TestValidateBackup_EdgeCases tests backup validation edge cases
func TestValidateBackup_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("missing file", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		manager, _ := NewManager(cfg, nil)
		manager.metadata.Backups = []*Backup{
			{ID: "test-id", FilePath: "/nonexistent/file.tar.gz", Status: StatusCompleted},
		}

		result, err := manager.ValidateBackup("test-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Valid {
			t.Error("expected invalid result for missing file")
		}
	})

	t.Run("checksum mismatch", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		os.MkdirAll(env.backupDir, 0750)
		backupFile := filepath.Join(env.backupDir, "test.tar.gz")
		os.WriteFile(backupFile, []byte("test content"), 0644)

		cfg := env.newTestConfig()
		manager, _ := NewManager(cfg, nil)
		manager.metadata.Backups = []*Backup{
			{
				ID:       "test-id",
				FilePath: backupFile,
				Checksum: "wrong-checksum",
				Status:   StatusCompleted,
				Type:     TypeFull,
				Contents: BackupContents{Files: []BackupFile{}},
			},
		}

		result, err := manager.ValidateBackup("test-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Valid {
			t.Error("expected invalid result for checksum mismatch")
		}
		if result.ChecksumValid {
			t.Error("expected checksum to be invalid")
		}
	})
}

// TestGetStats_EdgeCases tests statistics edge cases
func TestGetStats_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("nil metadata", func(t *testing.T) {
		cfg := &Config{Enabled: false}
		manager, _ := NewManager(cfg, nil)
		manager.metadata = nil

		stats, err := manager.GetStats()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stats.TotalCount != 0 {
			t.Error("expected 0 total count")
		}
	})

	t.Run("empty backups", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		manager, _ := NewManager(cfg, nil)

		stats, err := manager.GetStats()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stats.TotalCount != 0 {
			t.Error("expected 0 total count")
		}
		if stats.SuccessRate != 0 {
			t.Error("expected 0 success rate")
		}
	})

	t.Run("with failed backups", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		manager, _ := NewManager(cfg, nil)

		now := time.Now()
		manager.metadata.Backups = []*Backup{
			{ID: "1", Status: StatusCompleted, Type: TypeFull, FileSize: 1000, CreatedAt: now, Duration: time.Second},
			{ID: "2", Status: StatusFailed, Type: TypeFull, FileSize: 0, CreatedAt: now.Add(time.Hour)},
		}

		stats, _ := manager.GetStats()
		if stats.TotalCount != 2 {
			t.Errorf("expected 2 backups, got %d", stats.TotalCount)
		}
		if stats.SuccessRate != 50.0 {
			t.Errorf("expected 50%% success rate, got %f", stats.SuccessRate)
		}
	})
}

// TestCalculateNextBackupTime_EdgeCases tests next backup time calculation
func TestCalculateNextBackupTime_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("short interval", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		cfg.Schedule.Enabled = true
		cfg.Schedule.Interval = 2 * time.Hour
		manager, _ := NewManager(cfg, nil)

		next := manager.calculateNextBackupTime()
		expectedMin := time.Now().Add(2 * time.Hour).Add(-time.Second)
		expectedMax := time.Now().Add(2 * time.Hour).Add(time.Second)

		if next.Before(expectedMin) || next.After(expectedMax) {
			t.Error("next backup time should be approximately 2 hours from now")
		}
	})

	t.Run("multi-day interval", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		cfg.Schedule.Enabled = true
		cfg.Schedule.Interval = 48 * time.Hour
		cfg.Schedule.PreferredHour = 3
		manager, _ := NewManager(cfg, nil)

		next := manager.calculateNextBackupTime()
		if next.Hour() != 3 {
			t.Errorf("expected hour=3, got %d", next.Hour())
		}
	})
}
