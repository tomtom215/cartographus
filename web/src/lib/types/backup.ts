// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Backup and restore types
 */

export interface Backup {
    id: string;
    type: 'full' | 'database' | 'config';
    filename: string;
    size_bytes: number;
    created_at: string;
    notes?: string;
    database_records?: number;
    is_valid?: boolean;
}

export interface BackupStats {
    total_backups: number;
    total_size_bytes: number;
    oldest_backup?: string;
    newest_backup?: string;
    full_backups: number;
    database_backups: number;
    config_backups: number;
}

export interface CreateBackupRequest {
    type?: 'full' | 'database' | 'config';
    notes?: string;
}

export interface RestoreBackupRequest {
    validate_only?: boolean;
    create_pre_restore_backup?: boolean;
    restore_database?: boolean;
    restore_config?: boolean;
    force_restore?: boolean;
    verify_after_restore?: boolean;
}

export interface RestoreResult {
    success: boolean;
    pre_restore_backup_id?: string;
    records_restored?: number;
    warnings?: string[];
}

/**
 * Backup Retention Policy types (Task 29)
 * Implements GFS (Grandfather-Father-Son) retention strategy
 */

export interface RetentionPolicy {
    /** Keep at least this many backups regardless of age */
    min_count: number;
    /** Maximum number of backups to keep (0 = unlimited) */
    max_count: number;
    /** Maximum age of backups in days (0 = unlimited) */
    max_age_days: number;
    /** Keep all backups from the last N hours */
    keep_recent_hours: number;
    /** Keep at least one backup per day for N days */
    keep_daily_for_days: number;
    /** Keep at least one backup per week for N weeks */
    keep_weekly_for_weeks: number;
    /** Keep at least one backup per month for N months */
    keep_monthly_for_months: number;
}

export interface BackupPreviewItem {
    id: string;
    type: 'full' | 'database' | 'config';
    created_at: string;
    file_size: number;
    reasons: string[];
}

export interface RetentionPreview {
    would_delete: BackupPreviewItem[];
    would_keep: BackupPreviewItem[];
    deleted_count: number;
    kept_count: number;
    total_deleted_size: number;
    total_kept_size: number;
}

export interface RetentionApplyResult {
    deleted_count: number;
    freed_bytes: number;
    deleted_ids: string[];
}

/**
 * Backup Schedule Configuration
 * Controls automatic backup scheduling
 */

export interface ScheduleConfig {
    /** Whether scheduled backups are enabled */
    enabled: boolean;
    /** Hours between backups (1-720) */
    interval_hours: number;
    /** Hour of day for daily+ backups (0-23) */
    preferred_hour: number;
    /** Type of backup to create */
    backup_type: 'full' | 'database' | 'config';
    /** Create backup before sync operations */
    pre_sync_backup: boolean;
}

export interface SetScheduleConfigRequest {
    enabled: boolean;
    interval_hours: number;
    preferred_hour: number;
    backup_type?: 'full' | 'database' | 'config';
    pre_sync_backup: boolean;
}
