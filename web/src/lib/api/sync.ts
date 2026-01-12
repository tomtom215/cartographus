// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Sync API Module
 *
 * API client for data synchronization operations.
 * Supports Tautulli database import, Plex historical sync,
 * and sync status monitoring.
 */

import type {
    TautulliImportRequest,
    TautulliImportResponse,
    TautulliValidateResponse,
    PlexHistoricalRequest,
    PlexHistoricalResponse,
    SyncStatusResponse,
    SyncProgress,
} from '../types/sync';
import { BaseAPIClient } from './client';

/**
 * Sync API methods
 */
export class SyncAPI extends BaseAPIClient {
    // ========================================================================
    // Tautulli Import Operations
    // ========================================================================

    /**
     * Start a Tautulli database import.
     * The import runs in the background; use getImportStatus() to monitor progress.
     * Requires admin role.
     *
     * @param request - Import options
     * @returns Import start response
     */
    async startTautulliImport(request: TautulliImportRequest = {}): Promise<TautulliImportResponse> {
        const response = await this.fetch<TautulliImportResponse>('/import/tautulli', {
            method: 'POST',
            body: JSON.stringify(request),
        });
        return response.data;
    }

    /**
     * Get current Tautulli import status.
     * Returns progress statistics if an import is running or has completed.
     *
     * @returns Current import status
     */
    async getImportStatus(): Promise<TautulliImportResponse> {
        const response = await this.fetch<TautulliImportResponse>('/import/status');
        return response.data;
    }

    /**
     * Stop a running Tautulli import.
     * Progress is saved and can be resumed later.
     * Requires admin role.
     *
     * @returns Stop response with final statistics
     */
    async stopImport(): Promise<TautulliImportResponse> {
        const response = await this.fetch<TautulliImportResponse>('/import', {
            method: 'DELETE',
        });
        return response.data;
    }

    /**
     * Clear saved import progress.
     * Call this before starting a fresh import (non-resume).
     * Requires admin role.
     */
    async clearImportProgress(): Promise<TautulliImportResponse> {
        const response = await this.fetch<TautulliImportResponse>('/import/progress', {
            method: 'DELETE',
        });
        return response.data;
    }

    /**
     * Validate a Tautulli database without importing.
     * Returns database statistics like record count and date range.
     * Requires admin role.
     *
     * @param dbPath - Path to the Tautulli SQLite database
     * @returns Validation result with database statistics
     */
    async validateDatabase(dbPath: string): Promise<TautulliValidateResponse> {
        const response = await this.fetch<TautulliValidateResponse>('/import/validate', {
            method: 'POST',
            body: JSON.stringify({ db_path: dbPath }),
        });
        return response.data;
    }

    // ========================================================================
    // Plex Historical Sync Operations
    // ========================================================================

    /**
     * Start a Plex historical sync.
     * Syncs playback history from Plex servers directly.
     * Cannot run while a Tautulli import is active.
     * Requires admin role.
     *
     * @param request - Historical sync options
     * @returns Sync start response
     */
    async startPlexHistoricalSync(request: PlexHistoricalRequest = {}): Promise<PlexHistoricalResponse> {
        const response = await this.fetch<PlexHistoricalResponse>('/sync/plex/historical', {
            method: 'POST',
            body: JSON.stringify(request),
        });
        return response.data;
    }

    /**
     * Get Plex historical sync status.
     *
     * @returns Current sync status
     */
    async getPlexHistoricalStatus(): Promise<SyncProgress | null> {
        const response = await this.fetch<SyncStatusResponse>('/sync/status');
        return response.data.plex_historical || null;
    }

    // ========================================================================
    // Combined Sync Status
    // ========================================================================

    /**
     * Get status of all sync operations.
     * Returns combined status for Tautulli import, Plex historical,
     * and per-server sync operations.
     *
     * @returns Combined sync status
     */
    async getSyncStatus(): Promise<SyncStatusResponse> {
        const response = await this.fetch<SyncStatusResponse>('/sync/status');
        return response.data;
    }

    /**
     * Check if any sync operation is currently running.
     *
     * @returns True if any sync is in progress
     */
    async isAnySyncRunning(): Promise<boolean> {
        const status = await this.getSyncStatus();

        if (status.tautulli_import?.status === 'running') {
            return true;
        }
        if (status.plex_historical?.status === 'running') {
            return true;
        }
        if (status.server_syncs) {
            for (const serverSync of Object.values(status.server_syncs)) {
                if (serverSync.status === 'running') {
                    return true;
                }
            }
        }

        return false;
    }

    /**
     * Check if Tautulli import and Plex historical sync can run concurrently.
     * They use a mutex - only one can run at a time.
     *
     * @returns Object indicating availability
     */
    async checkSyncAvailability(): Promise<{
        tautulli_available: boolean;
        plex_historical_available: boolean;
        blocking_operation?: string;
    }> {
        const status = await this.getSyncStatus();

        const tautulliRunning = status.tautulli_import?.status === 'running';
        const plexHistoricalRunning = status.plex_historical?.status === 'running';

        if (tautulliRunning) {
            return {
                tautulli_available: false,
                plex_historical_available: false,
                blocking_operation: 'Tautulli import is currently running',
            };
        }

        if (plexHistoricalRunning) {
            return {
                tautulli_available: false,
                plex_historical_available: false,
                blocking_operation: 'Plex historical sync is currently running',
            };
        }

        return {
            tautulli_available: true,
            plex_historical_available: true,
        };
    }
}
