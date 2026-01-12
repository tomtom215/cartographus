// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Data Sync Types
 *
 * TypeScript interfaces for the Data Sync UI feature.
 * Supports Tautulli database import, Plex historical sync,
 * and per-server sync status tracking.
 */

/**
 * Sync operation status
 */
export type SyncStatus = 'idle' | 'running' | 'completed' | 'error' | 'cancelled';

/**
 * Type of sync operation
 */
export type SyncOperationType = 'tautulli_import' | 'plex_historical' | 'server_sync';

/**
 * Progress data for a sync operation
 */
export interface SyncProgress {
    /** Current status of the operation */
    status: SyncStatus;
    /** Total number of records to process */
    total_records: number;
    /** Number of records processed so far */
    processed_records: number;
    /** Number of records successfully imported */
    imported_records: number;
    /** Number of records skipped (validation failures) */
    skipped_records: number;
    /** Number of errors encountered */
    error_count: number;
    /** Progress as percentage (0-100) */
    progress_percent: number;
    /** Current processing rate (records per second) */
    records_per_second: number;
    /** Elapsed time in seconds */
    elapsed_seconds: number;
    /** Estimated time remaining in seconds */
    estimated_remaining_seconds: number;
    /** ISO 8601 timestamp when operation started */
    start_time?: string;
    /** ID of the last successfully processed record (for resume) */
    last_processed_id?: number;
    /** Whether this is a dry run (validate without importing) */
    dry_run?: boolean;
    /** Recent errors encountered */
    errors?: SyncError[];
}

/**
 * Error encountered during sync
 */
export interface SyncError {
    /** ISO 8601 timestamp when error occurred */
    timestamp: string;
    /** ID of the record that caused the error (if applicable) */
    record_id?: number;
    /** Human-readable error message */
    message: string;
    /** Whether the error is recoverable (operation can continue) */
    recoverable: boolean;
}

/**
 * Request to start a Tautulli database import
 */
export interface TautulliImportRequest {
    /** Path to the Tautulli SQLite database file */
    db_path?: string;
    /** Whether to resume from the last saved position */
    resume?: boolean;
    /** Whether to validate without actually importing */
    dry_run?: boolean;
}

/**
 * Response from Tautulli import operations
 */
export interface TautulliImportResponse {
    /** Whether the operation was successful */
    success: boolean;
    /** Human-readable message */
    message?: string;
    /** Error message if failed */
    error?: string;
    /** Current progress statistics */
    stats?: SyncProgress;
}

/**
 * Response from database validation
 */
export interface TautulliValidateResponse {
    /** Whether validation passed */
    success: boolean;
    /** Error message if validation failed */
    error?: string;
    /** Total number of records in database */
    total_records?: number;
    /** Date range of data */
    date_range?: {
        earliest: string;
        latest: string;
    };
    /** Number of unique users */
    unique_users?: number;
    /** Breakdown by media type */
    media_types?: Record<string, number>;
}

/**
 * Request to start Plex historical sync
 */
export interface PlexHistoricalRequest {
    /** Number of days to sync back (default: 30) */
    days_back?: number;
    /** Specific library IDs to sync (empty = all libraries) */
    library_ids?: string[];
}

/**
 * Response from Plex historical sync
 */
export interface PlexHistoricalResponse {
    /** Whether the operation started successfully */
    success: boolean;
    /** Human-readable message */
    message?: string;
    /** Error message if failed */
    error?: string;
    /** Correlation ID for tracking */
    correlation_id?: string;
}

/**
 * Combined sync status response
 */
export interface SyncStatusResponse {
    /** Tautulli import status (if any) */
    tautulli_import?: SyncProgress;
    /** Plex historical sync status (if any) */
    plex_historical?: SyncProgress;
    /** Per-server sync status */
    server_syncs?: Record<string, SyncProgress>;
}

/**
 * WebSocket message for sync progress updates
 */
export interface SyncProgressMessage {
    type: 'sync_progress';
    data: {
        /** Type of sync operation */
        operation: SyncOperationType;
        /** Current status */
        status: SyncStatus;
        /** Server ID (for server_sync operations) */
        server_id?: string;
        /** Progress data */
        progress: SyncProgress;
        /** Human-readable status message */
        message?: string;
        /** Error message if applicable */
        error?: string;
        /** Correlation ID for tracing */
        correlation_id: string;
    };
}

/**
 * Configuration for sync manager
 */
export interface SyncManagerConfig {
    /** Polling interval when WebSocket is disconnected (ms) */
    pollingInterval?: number;
    /** Maximum age of persisted state before cleanup (ms) */
    maxStateAge?: number;
    /** Callback when sync status changes */
    onStatusChange?: (operation: SyncOperationType, progress: SyncProgress) => void;
    /** Callback when an error occurs */
    onError?: (operation: SyncOperationType, error: Error) => void;
    /** Callback when sync completes */
    onComplete?: (operation: SyncOperationType, progress: SyncProgress) => void;
}

/**
 * Persisted sync state for durability
 */
export interface PersistedSyncState {
    /** Correlation ID for this sync session */
    correlationId: string;
    /** Type of operation */
    operation: SyncOperationType;
    /** Current status when persisted */
    status: SyncStatus;
    /** Last known progress */
    progress: SyncProgress;
    /** Timestamp when state was persisted */
    persistedAt: number;
    /** Timestamp when state expires */
    expiresAt: number;
}

/**
 * UI display configuration for sync progress
 */
export interface SyncProgressDisplayConfig {
    /** Whether to show the progress bar */
    showProgressBar?: boolean;
    /** Whether to show detailed statistics */
    showDetailedStats?: boolean;
    /** Whether to show error log */
    showErrorLog?: boolean;
    /** Maximum number of errors to display */
    maxErrorsDisplayed?: number;
    /** Whether to show ETA */
    showETA?: boolean;
    /** Custom labels for status messages */
    statusLabels?: Partial<Record<SyncStatus, string>>;
}

/**
 * Default status labels for display
 */
export const DEFAULT_STATUS_LABELS: Record<SyncStatus, string> = {
    idle: 'Ready',
    running: 'In Progress',
    completed: 'Completed',
    error: 'Error',
    cancelled: 'Cancelled',
};

/**
 * Default labels for operation types
 */
export const OPERATION_LABELS: Record<SyncOperationType, string> = {
    tautulli_import: 'Tautulli Database Import',
    plex_historical: 'Plex Historical Sync',
    server_sync: 'Server Sync',
};

/**
 * Formats seconds into human-readable duration
 * @param seconds Number of seconds
 * @returns Formatted duration string (e.g., "2m 30s", "1h 15m")
 */
export function formatDuration(seconds: number): string {
    if (!seconds || seconds < 0) return '0s';

    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = Math.floor(seconds % 60);

    if (hours > 0) {
        return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`;
    }
    if (minutes > 0) {
        return secs > 0 ? `${minutes}m ${secs}s` : `${minutes}m`;
    }
    return `${secs}s`;
}

/**
 * Formats a number with locale-specific separators
 * @param num Number to format
 * @returns Formatted number string
 */
export function formatNumber(num: number): string {
    return num.toLocaleString();
}

/**
 * Calculates human-readable ETA string
 * @param estimatedSecondsRemaining Estimated seconds remaining
 * @returns Human-readable ETA (e.g., "~5 minutes", "< 1 minute")
 */
export function formatETA(estimatedSecondsRemaining: number): string {
    if (!estimatedSecondsRemaining || estimatedSecondsRemaining < 0) {
        return 'calculating...';
    }

    if (estimatedSecondsRemaining < 60) {
        return '< 1 minute';
    }

    const minutes = Math.ceil(estimatedSecondsRemaining / 60);
    if (minutes === 1) {
        return '~1 minute';
    }
    if (minutes < 60) {
        return `~${minutes} minutes`;
    }

    const hours = Math.floor(minutes / 60);
    const remainingMinutes = minutes % 60;
    if (remainingMinutes === 0) {
        return `~${hours} hour${hours > 1 ? 's' : ''}`;
    }
    return `~${hours}h ${remainingMinutes}m`;
}
