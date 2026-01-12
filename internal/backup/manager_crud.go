// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
manager_crud.go - Backup CRUD Operations

This file provides Create, Read, Update, and Delete operations for backup
management, including backup creation, listing, retrieval, and deletion.

Backup Creation Flow:
 1. Initialize backup record with UUID, type, trigger, and metadata
 2. Generate timestamped filename (backup-{type}-{timestamp}-{id}.tar.gz)
 3. Create archive using tar writer (delegated to manager_archive.go)
 4. Calculate SHA-256 checksum for integrity verification
 5. Update status to completed and save metadata
 6. Trigger completion callback for notification

Supported Triggers:
  - TriggerManual: User-initiated backup via API
  - TriggerScheduled: Automatic backup from scheduler
  - TriggerPreSync: Backup before Tautulli data sync
  - TriggerPreRestore: Backup before restore operation

Listing and Filtering:
  - Filter by type (full, database, config)
  - Filter by status (in_progress, completed, failed)
  - Filter by trigger source
  - Filter by date range
  - Pagination with offset and limit
  - Sortable by creation time (asc/desc)

Thread Safety:
All metadata operations use sync.RWMutex for safe concurrent access.
*/

//nolint:staticcheck // File documentation, not package doc
package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
)

// CreateBackup creates a new backup
func (m *Manager) CreateBackup(ctx context.Context, backupType BackupType, notes string) (*Backup, error) {
	return m.createBackupWithTrigger(ctx, backupType, TriggerManual, notes)
}

// CreatePreSyncBackup creates a backup before a sync operation
func (m *Manager) CreatePreSyncBackup(ctx context.Context) (*Backup, error) {
	if !m.cfg.Schedule.PreSyncBackup {
		return nil, nil // Pre-sync backups disabled
	}
	return m.createBackupWithTrigger(ctx, TypeDatabase, TriggerPreSync, "Pre-sync snapshot")
}

// createBackupWithTrigger creates a backup with the specified trigger
func (m *Manager) createBackupWithTrigger(ctx context.Context, backupType BackupType, trigger BackupTrigger, notes string) (*Backup, error) {
	if !m.cfg.Enabled {
		return nil, fmt.Errorf("backups are disabled")
	}

	startTime := time.Now()

	// Create backup record
	backup := m.initializeBackupRecord(backupType, trigger, notes, startTime)

	// Generate backup filename
	backup.FilePath = m.generateBackupFilePath(backupType, startTime, backup.ID)

	// Create the backup file
	if err := m.createBackupArchive(ctx, backup, backupType); err != nil {
		return m.handleBackupError(backup, startTime, err)
	}

	// Calculate checksum
	checksum, err := m.calculateFileChecksum(backup.FilePath)
	if err != nil {
		return m.handleBackupError(backup, startTime, fmt.Errorf("failed to calculate checksum: %w", err))
	}
	backup.Checksum = checksum

	// Get file size
	fileInfo, err := os.Stat(backup.FilePath)
	if err != nil {
		return m.handleBackupError(backup, startTime, fmt.Errorf("failed to stat backup file: %w", err))
	}
	backup.FileSize = fileInfo.Size()

	// Mark as completed
	backup.Status = StatusCompleted
	completedAt := time.Now()
	backup.CompletedAt = &completedAt
	backup.Duration = time.Since(startTime)

	// Save backup metadata
	m.saveBackup(backup)

	// Call completion callback
	if m.onBackupComplete != nil {
		m.onBackupComplete(backup)
	}

	return backup, nil
}

// initializeBackupRecord creates a new backup record with initial values
func (m *Manager) initializeBackupRecord(backupType BackupType, trigger BackupTrigger, notes string, startTime time.Time) *Backup {
	return &Backup{
		ID:         uuid.New().String(),
		Type:       backupType,
		Status:     StatusInProgress,
		Trigger:    trigger,
		CreatedAt:  startTime,
		Notes:      notes,
		AppVersion: AppVersion,
		Compressed: m.cfg.Compression.Enabled,
		Encrypted:  m.cfg.Encryption.Enabled,
		Contents: BackupContents{
			Files: make([]BackupFile, 0),
		},
	}
}

// generateBackupFilePath generates the file path for a backup
func (m *Manager) generateBackupFilePath(backupType BackupType, startTime time.Time, backupID string) string {
	timestamp := startTime.Format("20060102-150405")
	filename := fmt.Sprintf("backup-%s-%s-%s", backupType, timestamp, backupID[:8])
	if m.cfg.Compression.Enabled {
		filename += ".tar.gz"
	} else {
		filename += ".tar"
	}
	return filepath.Join(m.cfg.BackupDir, filename)
}

// handleBackupError marks a backup as failed and saves it
func (m *Manager) handleBackupError(backup *Backup, startTime time.Time, err error) (*Backup, error) {
	backup.Status = StatusFailed
	backup.Error = err.Error()
	completedAt := time.Now()
	backup.CompletedAt = &completedAt
	backup.Duration = time.Since(startTime)
	m.saveBackup(backup)
	return backup, err
}

// ListBackups returns a list of backups with optional filtering
//
//nolint:gocyclo // Filter and pagination logic with multiple conditions
func (m *Manager) ListBackups(opts BackupListOptions) ([]*Backup, error) {
	m.metadataMu.RLock()
	defer m.metadataMu.RUnlock()

	if m.metadata == nil {
		return []*Backup{}, nil
	}

	// Filter backups
	filtered := m.filterBackups(opts)

	// Sort by created_at
	sort.Slice(filtered, func(i, j int) bool {
		if opts.SortDesc {
			return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
		}
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})

	// Apply pagination
	return m.applyPagination(filtered, opts), nil
}

// filterBackups filters backups based on the provided options
func (m *Manager) filterBackups(opts BackupListOptions) []*Backup {
	var filtered []*Backup
	for _, b := range m.metadata.Backups {
		if m.matchesFilter(b, opts) {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

// matchesFilter checks if a backup matches the filter options
func (m *Manager) matchesFilter(b *Backup, opts BackupListOptions) bool {
	if opts.Type != nil && b.Type != *opts.Type {
		return false
	}
	if opts.Status != nil && b.Status != *opts.Status {
		return false
	}
	if opts.Trigger != nil && b.Trigger != *opts.Trigger {
		return false
	}
	if opts.StartDate != nil && b.CreatedAt.Before(*opts.StartDate) {
		return false
	}
	if opts.EndDate != nil && b.CreatedAt.After(*opts.EndDate) {
		return false
	}
	return true
}

// applyPagination applies offset and limit to the filtered backups
func (m *Manager) applyPagination(filtered []*Backup, opts BackupListOptions) []*Backup {
	if opts.Offset > 0 && opts.Offset < len(filtered) {
		filtered = filtered[opts.Offset:]
	} else if opts.Offset >= len(filtered) {
		return []*Backup{}
	}

	if opts.Limit > 0 && opts.Limit < len(filtered) {
		filtered = filtered[:opts.Limit]
	}

	return filtered
}

// GetBackup returns a specific backup by ID
func (m *Manager) GetBackup(backupID string) (*Backup, error) {
	m.metadataMu.RLock()
	defer m.metadataMu.RUnlock()

	for _, b := range m.metadata.Backups {
		if b.ID == backupID {
			return b, nil
		}
	}

	return nil, fmt.Errorf("backup not found: %s", backupID)
}

// DeleteBackup deletes a backup
func (m *Manager) DeleteBackup(backupID string) error {
	m.metadataMu.Lock()
	defer m.metadataMu.Unlock()

	// Find the backup
	backup, idx := m.findBackupLocked(backupID)
	if backup == nil {
		return fmt.Errorf("backup not found: %s", backupID)
	}

	// Delete the backup file
	if fileExists(backup.FilePath) {
		if err := os.Remove(backup.FilePath); err != nil {
			return fmt.Errorf("failed to delete backup file: %w", err)
		}
	}

	// Remove from metadata
	m.metadata.Backups = append(m.metadata.Backups[:idx], m.metadata.Backups[idx+1:]...)

	// Save metadata
	return m.saveMetadataLocked()
}

// findBackupLocked finds a backup by ID (must be called with lock held)
func (m *Manager) findBackupLocked(backupID string) (*Backup, int) {
	for i, b := range m.metadata.Backups {
		if b.ID == backupID {
			return b, i
		}
	}
	return nil, -1
}
