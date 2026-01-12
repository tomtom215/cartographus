// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package backup provides comprehensive backup and restore functionality for Cartographus.
//
// The backup package implements a production-ready disaster recovery system with:
//   - Full backups (database + configuration)
//   - Database-only backups
//   - Configuration-only backups
//   - Automatic scheduled backups with retention policies
//   - Pre-sync snapshots for data safety
//   - Compression (gzip) and integrity verification (SHA-256)
//   - Point-in-time restore capabilities
//
// Backup Types:
//
//	Full:        Complete backup including database, configuration, and metadata
//	Database:    DuckDB database files only (cartographus.duckdb + WAL)
//	Config:      Application configuration (sanitized - no secrets)
//	Incremental: Changes since last backup (future enhancement)
//
// Architecture:
//
//	┌──────────────┐     ┌─────────────────┐     ┌──────────────┐
//	│   Scheduler  │────▶│  BackupManager  │────▶│   Storage    │
//	└──────────────┘     └─────────────────┘     └──────────────┘
//	                            │                       │
//	                            ▼                       ▼
//	                     ┌──────────────┐      ┌──────────────┐
//	                     │   DuckDB     │      │   Archives   │
//	                     │   Database   │      │   (.tar.gz)  │
//	                     └──────────────┘      └──────────────┘
//
// Usage:
//
//	manager := backup.NewManager(cfg, db)
//	manager.Start(ctx)  // Start scheduled backups
//
//	// Manual backup
//	result, err := manager.CreateBackup(ctx, backup.TypeFull, "Manual backup")
//
//	// Restore from backup
//	err := manager.RestoreFromBackup(ctx, backupID, backup.RestoreOptions{
//	    ValidateOnly: false,
//	    StopServices: true,
//	})
package backup

import (
	"time"
)

// BackupType defines the type of backup to create
type BackupType string

const (
	// TypeFull creates a complete backup including database and configuration
	TypeFull BackupType = "full"

	// TypeDatabase creates a backup of the DuckDB database files only
	TypeDatabase BackupType = "database"

	// TypeConfig creates a backup of application configuration (sanitized)
	TypeConfig BackupType = "config"

	// TypeIncremental creates an incremental backup since the last full backup (future)
	TypeIncremental BackupType = "incremental"
)

// BackupStatus represents the current state of a backup
type BackupStatus string

const (
	// StatusPending indicates the backup is queued but not started
	StatusPending BackupStatus = "pending"

	// StatusInProgress indicates the backup is currently running
	StatusInProgress BackupStatus = "in_progress"

	// StatusCompleted indicates the backup finished successfully
	StatusCompleted BackupStatus = "completed"

	// StatusFailed indicates the backup failed
	StatusFailed BackupStatus = "failed"

	// StatusCorrupted indicates the backup file is corrupted (checksum mismatch)
	StatusCorrupted BackupStatus = "corrupted"
)

// BackupTrigger indicates what initiated the backup
type BackupTrigger string

const (
	// TriggerManual indicates the backup was triggered by user request
	TriggerManual BackupTrigger = "manual"

	// TriggerScheduled indicates the backup was triggered by the scheduler
	TriggerScheduled BackupTrigger = "scheduled"

	// TriggerPreSync indicates the backup was triggered before a sync operation
	TriggerPreSync BackupTrigger = "pre_sync"

	// TriggerPreRestore indicates the backup was triggered before a restore operation
	TriggerPreRestore BackupTrigger = "pre_restore"

	// TriggerRetention indicates the backup was triggered by retention policy
	TriggerRetention BackupTrigger = "retention"
)

// Backup represents metadata about a backup
type Backup struct {
	// Unique identifier for the backup
	ID string `json:"id"`

	// Type of backup (full, database, config)
	Type BackupType `json:"type"`

	// Current status of the backup
	Status BackupStatus `json:"status"`

	// What triggered this backup
	Trigger BackupTrigger `json:"trigger"`

	// When the backup was created
	CreatedAt time.Time `json:"created_at"`

	// When the backup completed (or failed)
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Duration of the backup operation
	Duration time.Duration `json:"duration_ms"`

	// Path to the backup file
	FilePath string `json:"file_path"`

	// Size of the backup file in bytes
	FileSize int64 `json:"file_size"`

	// SHA-256 checksum of the backup file
	Checksum string `json:"checksum"`

	// Whether the backup is compressed
	Compressed bool `json:"compressed"`

	// Whether the backup is encrypted
	Encrypted bool `json:"encrypted"`

	// Application version at time of backup
	AppVersion string `json:"app_version"`

	// Database schema version
	DBVersion string `json:"db_version"`

	// Number of records in the database at backup time
	RecordCount int64 `json:"record_count"`

	// User-provided notes about the backup
	Notes string `json:"notes,omitempty"`

	// Error message if backup failed
	Error string `json:"error,omitempty"`

	// Detailed backup contents
	Contents BackupContents `json:"contents"`
}

// BackupContents describes what's included in the backup
type BackupContents struct {
	// Database file information
	Database *DatabaseBackupInfo `json:"database,omitempty"`

	// Configuration backup information
	Config *ConfigBackupInfo `json:"config,omitempty"`

	// List of files included in the backup
	Files []BackupFile `json:"files"`
}

// DatabaseBackupInfo contains database-specific backup metadata
type DatabaseBackupInfo struct {
	// Path to the database file
	Path string `json:"path"`

	// Size of the database file
	Size int64 `json:"size"`

	// Whether WAL file was included
	WALIncluded bool `json:"wal_included"`

	// Size of the WAL file (if included)
	WALSize int64 `json:"wal_size,omitempty"`

	// Number of playback events
	PlaybackCount int64 `json:"playback_count"`

	// Number of geolocations
	GeolocationCount int64 `json:"geolocation_count"`

	// Database extensions that were loaded
	Extensions []string `json:"extensions"`

	// DuckDB version
	DuckDBVersion string `json:"duckdb_version"`
}

// ConfigBackupInfo contains configuration backup metadata
type ConfigBackupInfo struct {
	// Number of configuration values backed up
	ValueCount int `json:"value_count"`

	// Whether secrets were included (should always be false)
	IncludesSecrets bool `json:"includes_secrets"`

	// Configuration categories included
	Categories []string `json:"categories"`
}

// BackupFile represents a file included in the backup archive
type BackupFile struct {
	// Path within the archive
	Path string `json:"path"`

	// Original file path on disk
	OriginalPath string `json:"original_path"`

	// File size in bytes
	Size int64 `json:"size"`

	// File modification time
	ModTime time.Time `json:"mod_time"`

	// SHA-256 checksum of the individual file
	Checksum string `json:"checksum"`
}

// BackupListOptions provides filtering and pagination for backup listing
type BackupListOptions struct {
	// Filter by backup type
	Type *BackupType `json:"type,omitempty"`

	// Filter by backup status
	Status *BackupStatus `json:"status,omitempty"`

	// Filter by trigger
	Trigger *BackupTrigger `json:"trigger,omitempty"`

	// Filter by date range
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`

	// Pagination
	Limit  int `json:"limit"`
	Offset int `json:"offset"`

	// Sort order (newest first by default)
	SortDesc bool `json:"sort_desc"`
}

// RestoreOptions configures how a backup should be restored
type RestoreOptions struct {
	// Only validate the backup without restoring
	ValidateOnly bool `json:"validate_only"`

	// Create a backup before restoring (safety measure)
	CreatePreRestoreBackup bool `json:"create_pre_restore_backup"`

	// Whether to stop services during restore
	StopServices bool `json:"stop_services"`

	// Restore specific components only
	RestoreDatabase bool `json:"restore_database"`
	RestoreConfig   bool `json:"restore_config"`

	// Force restore even if checksums don't match (dangerous)
	ForceRestore bool `json:"force_restore"`

	// Verify restored data after restore
	VerifyAfterRestore bool `json:"verify_after_restore"`
}

// RestoreResult contains the result of a restore operation
type RestoreResult struct {
	// Whether the restore was successful
	Success bool `json:"success"`

	// The backup that was restored
	BackupID string `json:"backup_id"`

	// ID of the pre-restore backup (if created)
	PreRestoreBackupID string `json:"pre_restore_backup_id,omitempty"`

	// What was restored
	DatabaseRestored bool `json:"database_restored"`
	ConfigRestored   bool `json:"config_restored"`

	// Number of records restored
	RecordsRestored int64 `json:"records_restored"`

	// Duration of the restore operation
	Duration time.Duration `json:"duration_ms"`

	// Any warnings during restore
	Warnings []string `json:"warnings,omitempty"`

	// Error message if restore failed
	Error string `json:"error,omitempty"`

	// Whether application restart is required
	RestartRequired bool `json:"restart_required"`
}

// RetentionPolicy defines how backups should be retained
type RetentionPolicy struct {
	// Keep at least this many backups regardless of age
	MinCount int `json:"min_count"`

	// Maximum number of backups to keep (0 = unlimited)
	MaxCount int `json:"max_count"`

	// Maximum age of backups in days (0 = unlimited)
	MaxAgeDays int `json:"max_age_days"`

	// Keep all backups from the last N hours (for recent recovery)
	KeepRecentHours int `json:"keep_recent_hours"`

	// Keep at least one backup per day for the last N days
	KeepDailyForDays int `json:"keep_daily_for_days"`

	// Keep at least one backup per week for the last N weeks
	KeepWeeklyForWeeks int `json:"keep_weekly_for_weeks"`

	// Keep at least one backup per month for the last N months
	KeepMonthlyForMonths int `json:"keep_monthly_for_months"`
}

// DefaultRetentionPolicy returns a sensible default retention policy
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		MinCount:             3,  // Always keep at least 3 backups
		MaxCount:             50, // Maximum 50 backups
		MaxAgeDays:           90, // Delete backups older than 90 days
		KeepRecentHours:      24, // Keep all backups from last 24 hours
		KeepDailyForDays:     7,  // Keep daily backups for 7 days
		KeepWeeklyForWeeks:   4,  // Keep weekly backups for 4 weeks
		KeepMonthlyForMonths: 6,  // Keep monthly backups for 6 months
	}
}

// ScheduleConfig defines when automatic backups should run
type ScheduleConfig struct {
	// Enable automatic scheduled backups
	Enabled bool `json:"enabled"`

	// Backup interval (e.g., 24h for daily)
	Interval time.Duration `json:"interval"`

	// Time of day to run backups (hour in 24h format, 0-23)
	// Only used if Interval >= 24h
	PreferredHour int `json:"preferred_hour"`

	// Type of backup to create on schedule
	BackupType BackupType `json:"backup_type"`

	// Create backup before each sync operation
	PreSyncBackup bool `json:"pre_sync_backup"`
}

// DefaultScheduleConfig returns a sensible default schedule configuration
func DefaultScheduleConfig() ScheduleConfig {
	return ScheduleConfig{
		Enabled:       true,
		Interval:      24 * time.Hour,
		PreferredHour: 2, // 2 AM
		BackupType:    TypeFull,
		PreSyncBackup: false,
	}
}

// BackupStats contains statistics about the backup system
type BackupStats struct {
	// Total number of backups
	TotalCount int `json:"total_count"`

	// Breakdown by type
	CountByType map[BackupType]int `json:"count_by_type"`

	// Breakdown by status
	CountByStatus map[BackupStatus]int `json:"count_by_status"`

	// Total disk space used by backups
	TotalSizeBytes int64 `json:"total_size_bytes"`

	// Size of the oldest backup
	OldestBackupSize int64 `json:"oldest_backup_size"`

	// Size of the newest backup
	NewestBackupSize int64 `json:"newest_backup_size"`

	// Average backup size
	AverageBackupSize int64 `json:"average_backup_size"`

	// Date of oldest backup
	OldestBackup *time.Time `json:"oldest_backup,omitempty"`

	// Date of newest backup
	NewestBackup *time.Time `json:"newest_backup,omitempty"`

	// Average backup duration
	AverageDuration time.Duration `json:"average_duration_ms"`

	// Success rate (percentage)
	SuccessRate float64 `json:"success_rate"`

	// Last backup result
	LastBackup *Backup `json:"last_backup,omitempty"`

	// Next scheduled backup time
	NextScheduledBackup *time.Time `json:"next_scheduled_backup,omitempty"`

	// Retention policy in effect
	RetentionPolicy RetentionPolicy `json:"retention_policy"`
}

// ValidationResult contains the result of backup validation
type ValidationResult struct {
	// Whether the backup is valid
	Valid bool `json:"valid"`

	// Backup metadata
	Backup *Backup `json:"backup"`

	// Checksum verification result
	ChecksumValid bool `json:"checksum_valid"`

	// Expected checksum
	ExpectedChecksum string `json:"expected_checksum"`

	// Actual checksum
	ActualChecksum string `json:"actual_checksum"`

	// Whether the archive is readable
	ArchiveReadable bool `json:"archive_readable"`

	// Whether all expected files are present
	FilesComplete bool `json:"files_complete"`

	// Missing files (if any)
	MissingFiles []string `json:"missing_files,omitempty"`

	// Corrupted files (if any)
	CorruptedFiles []string `json:"corrupted_files,omitempty"`

	// Whether the database backup is valid
	DatabaseValid bool `json:"database_valid"`

	// Whether the config backup is valid
	ConfigValid bool `json:"config_valid"`

	// Validation errors
	Errors []string `json:"errors,omitempty"`

	// Validation warnings
	Warnings []string `json:"warnings,omitempty"`
}
