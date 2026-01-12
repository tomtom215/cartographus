// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
manager_scheduler.go - Backup Scheduling

This file implements the automatic backup scheduling functionality,
allowing backups to be created at regular intervals without manual intervention.

Scheduling Features:
  - Configurable backup intervals (hourly, daily, weekly)
  - Preferred hour for daily+ backups (e.g., run at 3 AM)
  - Automatic retention policy enforcement after each backup
  - Graceful shutdown via context cancellation or stop signal

Timer Logic:
  - For intervals >= 24h: Uses preferred hour, scheduling for next occurrence
  - For shorter intervals: Simply adds interval to current time
  - Timer is reset after each backup completes

Integration:
The scheduler is started via Manager.Start() and stopped via Manager.Stop().
It runs independently and calls Manager.CreateBackup() internally.
*/

//nolint:staticcheck // File documentation, not package doc
package backup

import (
	"context"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
)

// runScheduler runs the backup scheduler loop
func (m *Manager) runScheduler(ctx context.Context) {
	defer m.schedulerWg.Done()

	// Calculate time until next scheduled backup
	nextBackup := m.calculateNextBackupTime()
	m.metadataMu.Lock()
	m.metadata.NextScheduled = &nextBackup
	m.saveMetadataLocked() //nolint:errcheck // Non-critical in scheduler
	m.metadataMu.Unlock()

	timer := time.NewTimer(time.Until(nextBackup))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.schedulerStop:
			return
		case <-timer.C:
			// Time to create a backup
			backup, err := m.CreateBackup(ctx, m.cfg.Schedule.BackupType, "Scheduled backup")
			if err != nil {
				logging.Error().Err(err).Msg("Scheduled backup failed")
			} else {
				logging.Info().Str("backup_id", backup.ID).Msg("Scheduled backup completed")
			}

			// Apply retention policy
			if err := m.ApplyRetentionPolicy(ctx); err != nil {
				logging.Error().Err(err).Msg("Retention policy application failed")
			}

			// Calculate next backup time
			nextBackup = m.calculateNextBackupTime()
			m.metadataMu.Lock()
			now := time.Now()
			m.metadata.LastScheduled = &now
			m.metadata.NextScheduled = &nextBackup
			m.saveMetadataLocked() //nolint:errcheck // Non-critical in scheduler
			m.metadataMu.Unlock()

			timer.Reset(time.Until(nextBackup))
		}
	}
}

// calculateNextBackupTime determines when the next scheduled backup should run
func (m *Manager) calculateNextBackupTime() time.Time {
	now := time.Now()
	interval := m.cfg.Schedule.Interval

	if interval >= 24*time.Hour {
		// Daily or longer - use preferred hour
		next := time.Date(now.Year(), now.Month(), now.Day(),
			m.cfg.Schedule.PreferredHour, 0, 0, 0, now.Location())

		// If we've already passed the preferred hour today, schedule for tomorrow
		if next.Before(now) {
			next = next.Add(24 * time.Hour)
		}

		// Add additional days if interval is more than 24h
		if interval > 24*time.Hour {
			days := int(interval.Hours() / 24)
			next = next.Add(time.Duration(days-1) * 24 * time.Hour)
		}

		return next
	}

	// Shorter interval - just add interval to now
	return now.Add(interval)
}
