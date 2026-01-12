// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
manager_helpers.go - Statistics and Utility Functions

This file provides backup statistics calculation and general utility functions
used throughout the backup package.

Statistics (BackupStats):
  - TotalCount: Total number of backups
  - TotalSizeBytes: Combined size of all backups
  - AverageBackupSize: Mean backup size
  - AverageDuration: Mean backup duration for successful backups
  - SuccessRate: Percentage of completed backups (0-100%)
  - CountByType: Breakdown by backup type (full/database/config)
  - CountByStatus: Breakdown by status (in_progress/completed/failed)
  - OldestBackup/NewestBackup: Date range of backups
  - NextScheduledBackup: Next automatic backup time

Helper Functions:
  - fileExists(): Check if a file exists on disk
  - getFileSize(): Get file size in bytes (0 on error)
  - containsFile(): Check if a file is in the validation result

Usage:
Statistics are calculated on-demand from metadata and not cached, ensuring
they always reflect the current state of backups.
*/

//nolint:staticcheck // File documentation, not package doc
package backup

import (
	"os"
	"time"
)

// GetStats returns backup statistics
func (m *Manager) GetStats() (*BackupStats, error) {
	m.metadataMu.RLock()
	defer m.metadataMu.RUnlock()

	stats := &BackupStats{
		CountByType:     make(map[BackupType]int),
		CountByStatus:   make(map[BackupStatus]int),
		RetentionPolicy: m.cfg.Retention,
	}

	if m.metadata == nil || len(m.metadata.Backups) == 0 {
		return stats, nil
	}

	m.calculateBackupStats(stats)

	if m.metadata.NextScheduled != nil {
		stats.NextScheduledBackup = m.metadata.NextScheduled
	}

	return stats, nil
}

// calculateBackupStats calculates statistics from backup metadata
func (m *Manager) calculateBackupStats(stats *BackupStats) {
	var totalDuration time.Duration
	var successCount int

	for _, b := range m.metadata.Backups {
		stats.TotalCount++
		stats.CountByType[b.Type]++
		stats.CountByStatus[b.Status]++
		stats.TotalSizeBytes += b.FileSize

		if b.Status == StatusCompleted {
			successCount++
			totalDuration += b.Duration
		}

		m.updateOldestNewest(stats, b)
	}

	m.calculateAverages(stats, successCount, totalDuration)
}

// updateOldestNewest tracks the oldest and newest backups
func (m *Manager) updateOldestNewest(stats *BackupStats, b *Backup) {
	if stats.OldestBackup == nil || b.CreatedAt.Before(*stats.OldestBackup) {
		stats.OldestBackup = &b.CreatedAt
		stats.OldestBackupSize = b.FileSize
	}
	if stats.NewestBackup == nil || b.CreatedAt.After(*stats.NewestBackup) {
		stats.NewestBackup = &b.CreatedAt
		stats.NewestBackupSize = b.FileSize
		stats.LastBackup = b
	}
}

// calculateAverages calculates average size, duration, and success rate
func (m *Manager) calculateAverages(stats *BackupStats, successCount int, totalDuration time.Duration) {
	if stats.TotalCount > 0 {
		stats.AverageBackupSize = stats.TotalSizeBytes / int64(stats.TotalCount)
		stats.SuccessRate = float64(successCount) / float64(stats.TotalCount) * 100
	}

	if successCount > 0 {
		stats.AverageDuration = totalDuration / time.Duration(successCount)
	}
}

// Helper functions

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func containsFile(result *ValidationResult, filename string) bool {
	for _, f := range result.Backup.Contents.Files {
		if f.Path == filename {
			return true
		}
	}
	return false
}
