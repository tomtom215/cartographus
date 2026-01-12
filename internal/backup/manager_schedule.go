// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
manager_schedule.go - Backup Schedule Configuration

This file provides methods to get and update the backup schedule configuration
at runtime, allowing users to modify scheduling settings through the API.

Thread Safety:
All configuration updates are protected by mutex locks and persist to metadata.
Changing the schedule while the scheduler is running will take effect on the
next backup cycle.
*/

//nolint:staticcheck // File documentation, not package doc
package backup

import (
	"context"
	"fmt"
	"time"
)

// GetScheduleConfig returns the current schedule configuration
func (m *Manager) GetScheduleConfig() ScheduleConfig {
	return m.cfg.Schedule
}

// SetScheduleConfig updates the schedule configuration
func (m *Manager) SetScheduleConfig(ctx context.Context, schedule ScheduleConfig) error {
	// Validate the new schedule
	if err := m.validateSchedule(schedule); err != nil {
		return err
	}

	// Stop the current scheduler if running
	wasRunning := m.isRunning()
	if wasRunning {
		if err := m.Stop(); err != nil {
			return fmt.Errorf("failed to stop scheduler: %w", err)
		}
	}

	// Update configuration
	m.cfg.Schedule = schedule

	// Persist to metadata
	m.metadataMu.Lock()
	if err := m.saveMetadataLocked(); err != nil {
		m.metadataMu.Unlock()
		return fmt.Errorf("failed to save schedule config: %w", err)
	}
	m.metadataMu.Unlock()

	// Restart scheduler if it was running and is still enabled
	if wasRunning && schedule.Enabled {
		if err := m.Start(ctx); err != nil {
			return fmt.Errorf("failed to restart scheduler: %w", err)
		}
	}

	return nil
}

// GetNextScheduledBackup returns the time of the next scheduled backup
func (m *Manager) GetNextScheduledBackup() *time.Time {
	m.metadataMu.RLock()
	defer m.metadataMu.RUnlock()

	if m.metadata != nil {
		return m.metadata.NextScheduled
	}
	return nil
}

// GetLastScheduledBackup returns the time of the last scheduled backup
func (m *Manager) GetLastScheduledBackup() *time.Time {
	m.metadataMu.RLock()
	defer m.metadataMu.RUnlock()

	if m.metadata != nil {
		return m.metadata.LastScheduled
	}
	return nil
}

// isRunning returns whether the scheduler is currently running
func (m *Manager) isRunning() bool {
	m.runningMu.Lock()
	defer m.runningMu.Unlock()
	return m.running
}

// validateSchedule validates a schedule configuration
func (m *Manager) validateSchedule(schedule ScheduleConfig) error {
	if !schedule.Enabled {
		return nil // No validation needed if scheduling is disabled
	}

	if schedule.Interval < time.Hour {
		return fmt.Errorf("interval must be at least 1 hour, got: %s", schedule.Interval)
	}

	if schedule.PreferredHour < 0 || schedule.PreferredHour > 23 {
		return fmt.Errorf("preferred_hour must be between 0 and 23, got: %d", schedule.PreferredHour)
	}

	if schedule.BackupType != TypeFull && schedule.BackupType != TypeDatabase && schedule.BackupType != TypeConfig {
		return fmt.Errorf("backup_type must be one of: full, database, config")
	}

	return nil
}

// TriggerScheduledBackup manually triggers a scheduled backup
func (m *Manager) TriggerScheduledBackup(ctx context.Context) (*Backup, error) {
	if !m.cfg.Enabled {
		return nil, fmt.Errorf("backup functionality is not enabled")
	}

	return m.CreateBackup(ctx, m.cfg.Schedule.BackupType, "Manually triggered scheduled backup")
}
