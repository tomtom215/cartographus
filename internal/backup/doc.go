// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package backup provides database and configuration backup/restore functionality
// for the Cartographus application with comprehensive retention policies,
// compression, and optional encryption support.
//
// # Overview
//
// The backup package implements a complete backup solution with the following features:
//   - Full, database-only, and config-only backup types
//   - GZIP and ZSTD compression support
//   - AES-256 encryption (optional)
//   - Flexible retention policies with GFS (Grandfather-Father-Son) strategy
//   - Scheduled backups with preferred hour configuration
//   - Pre-sync backup hooks for data protection
//   - Checksum verification for backup integrity
//
// # Architecture
//
// The backup system consists of several components:
//
//	Manager - Orchestrates backup operations and manages metadata
//	Config  - Configuration loaded from environment variables
//	Backup  - Individual backup metadata and status tracking
//	RetentionPolicy - Rules for automatic backup cleanup
//
// # Backup Types
//
// Three backup types are supported:
//
//	TypeFull     - Complete backup including database and config
//	TypeDatabase - DuckDB database file only
//	TypeConfig   - Configuration files only
//
// # Retention Policy
//
// The retention policy uses a Grandfather-Father-Son (GFS) strategy:
//
//	MinCount          - Minimum backups to always keep (protection floor)
//	MaxCount          - Maximum backups to store (hard ceiling)
//	MaxAgeDays        - Delete backups older than this
//	KeepRecentHours   - Keep all backups from the last N hours
//	KeepDailyForDays  - Keep one backup per day for N days
//	KeepWeeklyForWeeks - Keep one backup per week for N weeks
//	KeepMonthlyForMonths - Keep one backup per month for N months
//
// # Usage
//
// Basic usage:
//
//	// Load configuration from environment
//	cfg, err := backup.LoadConfig()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Create manager with database interface
//	manager, err := backup.NewManager(cfg, db)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Start scheduler (if enabled)
//	if err := manager.Start(ctx); err != nil {
//		log.Fatal(err)
//	}
//	defer manager.Stop()
//
//	// Create manual backup
//	backup, err := manager.CreateBackup(ctx, backup.TypeFull, "Manual backup")
//	if err != nil {
//		log.Printf("Backup failed: %v", err)
//	}
//
// # Environment Variables
//
// Core settings:
//
//	BACKUP_ENABLED               - Enable backup functionality (default: true)
//	BACKUP_DIR                   - Directory to store backups (default: /data/backups)
//
// Schedule settings:
//
//	BACKUP_SCHEDULE_ENABLED      - Enable automatic scheduling (default: true)
//	BACKUP_INTERVAL              - Time between backups (default: 24h)
//	BACKUP_PREFERRED_HOUR        - Preferred hour for backups 0-23 (default: 2)
//	BACKUP_TYPE                  - Default backup type: full, database, config (default: full)
//	BACKUP_PRE_SYNC              - Create backup before each sync (default: false)
//
// Retention settings:
//
//	BACKUP_RETENTION_MIN_COUNT   - Minimum backups to keep (default: 3)
//	BACKUP_RETENTION_MAX_COUNT   - Maximum backups to keep (default: 50)
//	BACKUP_RETENTION_MAX_DAYS    - Delete backups older than N days (default: 90)
//	BACKUP_RETENTION_KEEP_RECENT_HOURS - Keep all backups from last N hours (default: 24)
//	BACKUP_RETENTION_KEEP_DAILY_DAYS   - Keep daily backups for N days (default: 7)
//	BACKUP_RETENTION_KEEP_WEEKLY_WEEKS - Keep weekly backups for N weeks (default: 4)
//	BACKUP_RETENTION_KEEP_MONTHLY_MONTHS - Keep monthly backups for N months (default: 6)
//
// Compression settings:
//
//	BACKUP_COMPRESSION_ENABLED   - Enable compression (default: true)
//	BACKUP_COMPRESSION_LEVEL     - Compression level 1-9 (default: 6)
//	BACKUP_COMPRESSION_ALGORITHM - Algorithm: gzip, zstd (default: gzip)
//
// Encryption settings:
//
//	BACKUP_ENCRYPTION_ENABLED    - Enable AES-256 encryption (default: false)
//	BACKUP_ENCRYPTION_KEY        - Encryption key (min 32 chars)
//	BACKUP_ENCRYPTION_KEY_ID     - Key identifier for rotation
//
// Notification settings:
//
//	BACKUP_NOTIFY_SUCCESS        - Notify on successful backup (default: false)
//	BACKUP_NOTIFY_FAILURE        - Notify on failed backup (default: true)
//	BACKUP_NOTIFY_CLEANUP        - Notify on retention cleanup (default: false)
//	BACKUP_WEBHOOK_URL           - Webhook URL for notifications
//
// # Thread Safety
//
// The Manager is safe for concurrent use. All metadata operations are protected
// by sync.RWMutex. The scheduler runs in a separate goroutine and can be
// started/stopped safely from any goroutine.
//
// # Backup File Format
//
// Backup files are stored as compressed tar archives with the naming convention:
//
//	{type}_{timestamp}_{uuid}.tar.gz  (or .tar.zst for ZSTD)
//
// Each backup includes:
//   - The database file (for full/database types)
//   - Configuration files (for full/config types)
//   - A manifest.json with backup metadata
//
// # Restore Process
//
// Restoration follows these steps:
//  1. Validate backup file integrity (checksum verification)
//  2. Extract backup to temporary directory
//  3. Stop active database connections
//  4. Replace existing database/config files
//  5. Restart database connections
//
// Pre-restore callbacks allow applications to prepare for the restore operation.
package backup
