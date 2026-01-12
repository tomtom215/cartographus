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

// TestApplyRetentionPolicy tests retention policy application
func TestApplyRetentionPolicy(t *testing.T) {
	t.Parallel()

	t.Run("nil metadata", func(t *testing.T) {
		cfg := &Config{Enabled: false}
		manager, _ := NewManager(cfg, nil)
		manager.metadata = nil

		err := manager.ApplyRetentionPolicy(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty backups", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := env.newTestConfig()
		manager, _ := NewManager(cfg, nil)

		err := manager.ApplyRetentionPolicy(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("deletes old backups", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		os.MkdirAll(env.backupDir, 0750)
		oldBackupFile := filepath.Join(env.backupDir, "old-backup.tar.gz")
		os.WriteFile(oldBackupFile, []byte("old"), 0644)

		cfg := &Config{
			Enabled:   true,
			BackupDir: env.backupDir,
			Retention: RetentionPolicy{
				MinCount:   1,
				MaxCount:   5,
				MaxAgeDays: 30,
			},
		}

		manager, _ := NewManager(cfg, nil)

		now := time.Now()
		manager.metadata.Backups = []*Backup{
			{ID: "new", Status: StatusCompleted, CreatedAt: now, FilePath: "/nonexistent"},
			{ID: "old", Status: StatusCompleted, CreatedAt: now.AddDate(0, 0, -60), FilePath: oldBackupFile},
		}

		err := manager.ApplyRetentionPolicy(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(manager.metadata.Backups) > 1 {
			t.Error("old backup should have been deleted")
		}
	})
}

// TestGetRetentionPreview tests retention preview
func TestGetRetentionPreview(t *testing.T) {
	t.Parallel()

	t.Run("nil metadata", func(t *testing.T) {
		cfg := &Config{Enabled: false}
		manager, _ := NewManager(cfg, nil)
		manager.metadata = nil

		preview, err := manager.GetRetentionPreview()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(preview.WouldDelete) != 0 || len(preview.WouldKeep) != 0 {
			t.Error("expected empty preview")
		}
	})

	t.Run("with backups", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.Close()

		cfg := &Config{
			Enabled:   true,
			BackupDir: env.backupDir,
			Retention: RetentionPolicy{
				MinCount:        2,
				MaxCount:        5,
				KeepRecentHours: 24,
			},
		}

		manager, _ := NewManager(cfg, nil)

		now := time.Now()
		manager.metadata.Backups = []*Backup{
			{ID: "1", Status: StatusCompleted, Type: TypeFull, CreatedAt: now, FileSize: 100},
			{ID: "2", Status: StatusCompleted, Type: TypeFull, CreatedAt: now.Add(-time.Hour), FileSize: 200},
		}

		preview, err := manager.GetRetentionPreview()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if preview.KeptCount < 2 {
			t.Error("expected at least 2 kept backups")
		}
	})
}

// TestSetRetentionPolicy_Validation tests retention policy validation
func TestSetRetentionPolicy_Validation(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	cfg := env.newTestConfig()
	manager, _ := NewManager(cfg, nil)

	tests := []struct {
		name    string
		policy  RetentionPolicy
		wantErr bool
	}{
		{
			name:    "min count zero",
			policy:  RetentionPolicy{MinCount: 0},
			wantErr: true,
		},
		{
			name:    "max count less than min count",
			policy:  RetentionPolicy{MinCount: 5, MaxCount: 3},
			wantErr: true,
		},
		{
			name:    "valid policy",
			policy:  RetentionPolicy{MinCount: 3, MaxCount: 10},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.SetRetentionPolicy(tt.policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetRetentionPolicy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCleanupCorruptedBackups tests corrupted backup cleanup
func TestCleanupCorruptedBackups(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	os.MkdirAll(env.backupDir, 0750)
	validFile := filepath.Join(env.backupDir, "valid.tar.gz")
	os.WriteFile(validFile, []byte("valid content"), 0644)

	cfg := env.newTestConfig()
	manager, _ := NewManager(cfg, nil)

	checksum, _ := manager.calculateFileChecksum(validFile)

	manager.metadata.Backups = []*Backup{
		{ID: "valid", Status: StatusCompleted, FilePath: validFile, Checksum: checksum},
		{ID: "missing-file", Status: StatusCompleted, FilePath: "/nonexistent/file.tar.gz", Checksum: "xxx"},
		{ID: "in-progress", Status: StatusInProgress, FilePath: "/some/path.tar.gz"},
		{ID: "wrong-checksum", Status: StatusCompleted, FilePath: validFile, Checksum: "wrong"},
	}

	count, err := manager.CleanupCorruptedBackups(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 2 { // missing-file and wrong-checksum
		t.Errorf("expected 2 corrupted backups cleaned up, got %d", count)
	}
}

// TestSelectBackups tests backup selection for retention
func TestSelectBackups(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	cfg := env.newTestConfig()
	manager, _ := NewManager(cfg, nil)
	now := time.Now()

	backups := []*Backup{
		{ID: "today-1", Type: TypeFull, CreatedAt: now, Status: StatusCompleted},
		{ID: "today-2", Type: TypeDatabase, CreatedAt: now.Add(-time.Hour), Status: StatusCompleted},
		{ID: "yesterday", Type: TypeFull, CreatedAt: now.AddDate(0, 0, -1), Status: StatusCompleted},
		{ID: "last-week", Type: TypeFull, CreatedAt: now.AddDate(0, 0, -7), Status: StatusCompleted},
		{ID: "last-month", Type: TypeFull, CreatedAt: now.AddDate(0, -1, 0), Status: StatusCompleted},
	}

	t.Run("selectDailyBackups", func(t *testing.T) {
		selected := manager.selectDailyBackups(backups, 7, now)
		if len(selected) == 0 {
			t.Error("expected at least some daily backups")
		}
	})

	t.Run("selectWeeklyBackups", func(t *testing.T) {
		selected := manager.selectWeeklyBackups(backups, 4, now)
		if len(selected) == 0 {
			t.Error("expected at least some weekly backups")
		}
	})

	t.Run("selectMonthlyBackups", func(t *testing.T) {
		selected := manager.selectMonthlyBackups(backups, 6, now)
		if len(selected) == 0 {
			t.Error("expected at least some monthly backups")
		}
	})
}

// TestBuildKeepReasons tests the buildKeepReasons method
func TestBuildKeepReasons(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	cfg := env.newTestConfig()
	manager, _ := NewManager(cfg, nil)
	now := time.Now()

	backups := []*Backup{
		{ID: "recent-1", Type: TypeFull, CreatedAt: now.Add(-1 * time.Hour), Status: StatusCompleted},
		{ID: "recent-2", Type: TypeFull, CreatedAt: now.Add(-2 * time.Hour), Status: StatusCompleted},
		{ID: "yesterday", Type: TypeFull, CreatedAt: now.AddDate(0, 0, -1), Status: StatusCompleted},
	}

	t.Run("minimum count protection", func(t *testing.T) {
		policy := RetentionPolicy{
			MinCount: 2,
		}

		keepReasons := manager.buildKeepReasons(backups, policy, now)

		// First two backups should have "minimum count protection"
		if len(keepReasons["recent-1"]) == 0 {
			t.Error("expected reasons for recent-1")
		}
		if len(keepReasons["recent-2"]) == 0 {
			t.Error("expected reasons for recent-2")
		}
	})

	t.Run("keep recent hours", func(t *testing.T) {
		policy := RetentionPolicy{
			MinCount:        1,
			KeepRecentHours: 24,
		}

		keepReasons := manager.buildKeepReasons(backups, policy, now)

		// All backups within 24 hours should be kept
		for _, b := range backups[:2] {
			if len(keepReasons[b.ID]) == 0 {
				t.Errorf("expected reasons for %s", b.ID)
			}
		}
	})

	t.Run("daily backup retention", func(t *testing.T) {
		policy := RetentionPolicy{
			MinCount:         1,
			KeepDailyForDays: 7,
		}

		keepReasons := manager.buildKeepReasons(backups, policy, now)

		// Should have at least one reason for each day's backup
		if len(keepReasons) == 0 {
			t.Error("expected some keep reasons for daily backups")
		}
	})

	t.Run("weekly backup retention", func(t *testing.T) {
		policy := RetentionPolicy{
			MinCount:           1,
			KeepWeeklyForWeeks: 4,
		}

		keepReasons := manager.buildKeepReasons(backups, policy, now)

		if len(keepReasons) == 0 {
			t.Error("expected some keep reasons for weekly backups")
		}
	})

	t.Run("monthly backup retention", func(t *testing.T) {
		policy := RetentionPolicy{
			MinCount:             1,
			KeepMonthlyForMonths: 6,
		}

		keepReasons := manager.buildKeepReasons(backups, policy, now)

		if len(keepReasons) == 0 {
			t.Error("expected some keep reasons for monthly backups")
		}
	})
}

// TestGetDeleteReason tests the getDeleteReason function
func TestGetDeleteReason(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name      string
		backup    *Backup
		policy    RetentionPolicy
		expectStr string
	}{
		{
			name:      "older than max age",
			backup:    &Backup{CreatedAt: now.AddDate(0, 0, -60)},
			policy:    RetentionPolicy{MaxAgeDays: 30},
			expectStr: "older than 30 days",
		},
		{
			name:      "no rule matched",
			backup:    &Backup{CreatedAt: now.AddDate(0, 0, -10)},
			policy:    RetentionPolicy{MaxAgeDays: 0}, // No max age
			expectStr: "no retention rule matched",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := getDeleteReason(tt.backup, tt.policy, now)
			if reason != tt.expectStr {
				t.Errorf("getDeleteReason() = %s, want %s", reason, tt.expectStr)
			}
		})
	}
}

// TestBuildPreviewItems tests the buildPreviewItems function
func TestBuildPreviewItems(t *testing.T) {
	t.Parallel()

	now := time.Now()
	backups := []*Backup{
		{ID: "keep-1", Type: TypeFull, CreatedAt: now, FileSize: 1000},
		{ID: "keep-2", Type: TypeFull, CreatedAt: now.Add(-time.Hour), FileSize: 2000},
		{ID: "delete-1", Type: TypeDatabase, CreatedAt: now.AddDate(0, 0, -60), FileSize: 500},
	}

	keepReasons := map[string][]string{
		"keep-1": {"minimum count protection"},
		"keep-2": {"within last 24 hours"},
	}

	policy := RetentionPolicy{MaxAgeDays: 30}
	preview := &RetentionPreview{
		WouldDelete: make([]*BackupPreviewItem, 0),
		WouldKeep:   make([]*BackupPreviewItem, 0),
	}

	buildPreviewItems(backups, keepReasons, policy, now, preview)

	if preview.KeptCount != 2 {
		t.Errorf("expected KeptCount=2, got %d", preview.KeptCount)
	}
	if preview.DeletedCount != 1 {
		t.Errorf("expected DeletedCount=1, got %d", preview.DeletedCount)
	}
	if preview.TotalKeptSize != 3000 {
		t.Errorf("expected TotalKeptSize=3000, got %d", preview.TotalKeptSize)
	}
	if preview.TotalDeletedSize != 500 {
		t.Errorf("expected TotalDeletedSize=500, got %d", preview.TotalDeletedSize)
	}
}

// TestEnforceMaxCount tests the enforceMaxCount method
func TestEnforceMaxCount(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	cfg := env.newTestConfig()
	manager, _ := NewManager(cfg, nil)
	now := time.Now()

	t.Run("no max count limit", func(t *testing.T) {
		backups := []*Backup{
			{ID: "1", CreatedAt: now},
			{ID: "2", CreatedAt: now.Add(-time.Hour)},
		}
		keepSet := map[string]bool{"1": true, "2": true}
		policy := RetentionPolicy{MaxCount: 0, MinCount: 1}

		toDelete := manager.enforceMaxCount(backups, keepSet, policy)
		if len(toDelete) != 0 {
			t.Error("expected no deletions when MaxCount=0")
		}
	})

	t.Run("within max count", func(t *testing.T) {
		backups := []*Backup{
			{ID: "1", CreatedAt: now},
			{ID: "2", CreatedAt: now.Add(-time.Hour)},
		}
		keepSet := map[string]bool{"1": true, "2": true}
		policy := RetentionPolicy{MaxCount: 5, MinCount: 1}

		toDelete := manager.enforceMaxCount(backups, keepSet, policy)
		if len(toDelete) != 0 {
			t.Error("expected no deletions when within MaxCount")
		}
	})

	t.Run("exceeds max count", func(t *testing.T) {
		backups := []*Backup{
			{ID: "1", CreatedAt: now},
			{ID: "2", CreatedAt: now.Add(-time.Hour)},
			{ID: "3", CreatedAt: now.Add(-2 * time.Hour)},
			{ID: "4", CreatedAt: now.Add(-3 * time.Hour)},
		}
		keepSet := map[string]bool{"1": true, "2": true, "3": true, "4": true}
		policy := RetentionPolicy{MaxCount: 2, MinCount: 1}

		toDelete := manager.enforceMaxCount(backups, keepSet, policy)
		// Should delete oldest backups that exceed MaxCount, respecting MinCount
		if len(toDelete) == 0 {
			t.Error("expected some deletions when exceeding MaxCount")
		}
	})
}

// TestIdentifyBackupsToDelete tests the identifyBackupsToDelete method
func TestIdentifyBackupsToDelete(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	cfg := env.newTestConfig()
	manager, _ := NewManager(cfg, nil)
	now := time.Now()

	t.Run("delete old backups", func(t *testing.T) {
		backups := []*Backup{
			{ID: "recent", CreatedAt: now},
			{ID: "old", CreatedAt: now.AddDate(0, 0, -60)},
		}
		keepSet := map[string]bool{"recent": true}
		policy := RetentionPolicy{MaxAgeDays: 30}

		toDelete := manager.identifyBackupsToDelete(backups, keepSet, policy, now)
		if len(toDelete) != 1 {
			t.Errorf("expected 1 deletion, got %d", len(toDelete))
		}
		if toDelete[0].ID != "old" {
			t.Error("expected 'old' backup to be deleted")
		}
	})

	t.Run("keep protected backups", func(t *testing.T) {
		backups := []*Backup{
			{ID: "protected", CreatedAt: now.AddDate(0, 0, -60)},
		}
		keepSet := map[string]bool{"protected": true}
		policy := RetentionPolicy{MaxAgeDays: 30}

		toDelete := manager.identifyBackupsToDelete(backups, keepSet, policy, now)
		if len(toDelete) != 0 {
			t.Error("expected no deletions for protected backups")
		}
	})
}

// TestApplyRetentionRules tests the applyRetentionRules method
func TestApplyRetentionRules(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	cfg := env.newTestConfig()
	manager, _ := NewManager(cfg, nil)
	now := time.Now()

	backups := []*Backup{
		{ID: "1", Type: TypeFull, CreatedAt: now},
		{ID: "2", Type: TypeFull, CreatedAt: now.Add(-time.Hour)},
		{ID: "3", Type: TypeFull, CreatedAt: now.AddDate(0, 0, -1)},
	}

	t.Run("all rules combined", func(t *testing.T) {
		policy := RetentionPolicy{
			MinCount:             2,
			KeepRecentHours:      24,
			KeepDailyForDays:     7,
			KeepWeeklyForWeeks:   4,
			KeepMonthlyForMonths: 6,
		}

		keepSet := manager.applyRetentionRules(backups, policy, now)

		// All backups should be kept due to various rules
		if len(keepSet) < 2 {
			t.Error("expected at least 2 backups to be kept")
		}
	})
}

// TestGetCompletedBackupsSorted tests the getCompletedBackupsSorted method
func TestGetCompletedBackupsSorted(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	defer env.Close()

	cfg := env.newTestConfig()
	manager, _ := NewManager(cfg, nil)
	now := time.Now()

	manager.metadata.Backups = []*Backup{
		{ID: "oldest", Status: StatusCompleted, CreatedAt: now.AddDate(0, 0, -2)},
		{ID: "newest", Status: StatusCompleted, CreatedAt: now},
		{ID: "middle", Status: StatusCompleted, CreatedAt: now.AddDate(0, 0, -1)},
		{ID: "failed", Status: StatusFailed, CreatedAt: now.Add(time.Hour)},
	}

	completed := manager.getCompletedBackupsSorted()

	if len(completed) != 3 {
		t.Errorf("expected 3 completed backups, got %d", len(completed))
	}

	// Should be sorted newest first
	if completed[0].ID != "newest" {
		t.Error("expected newest backup first")
	}
	if completed[1].ID != "middle" {
		t.Error("expected middle backup second")
	}
	if completed[2].ID != "oldest" {
		t.Error("expected oldest backup last")
	}
}
