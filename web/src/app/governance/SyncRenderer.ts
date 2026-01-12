// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * SyncRenderer - Sync Status Page
 *
 * Event pipeline metrics and WAL statistics.
 */

import { createLogger } from '../../lib/logger';
import type { API } from '../../lib/api';
import type { WALStats, DedupeAuditStats, DLQStats } from '../../lib/types';
import { BaseRenderer, GovernanceConfig } from './BaseRenderer';
import type { SyncSourceStatus } from './OverviewRenderer';

const logger = createLogger('SyncRenderer');

export class SyncRenderer extends BaseRenderer {
    private walStats: WALStats | null = null;
    private syncSources: SyncSourceStatus[] = [];
    private dedupeStats: DedupeAuditStats | null = null;
    private dlqStats: DLQStats | null = null;

    constructor(api: API, config: GovernanceConfig) {
        super(api, config);
    }

    // =========================================================================
    // Data Setters
    // =========================================================================

    setSyncSources(sources: SyncSourceStatus[]): void {
        this.syncSources = sources;
    }

    setDedupeStats(stats: DedupeAuditStats | null): void {
        this.dedupeStats = stats;
    }

    setDLQStats(stats: DLQStats | null): void {
        this.dlqStats = stats;
    }

    getWALStats(): WALStats | null {
        return this.walStats;
    }

    // =========================================================================
    // Public API
    // =========================================================================

    setupEventListeners(): void {
        document.getElementById('sync-refresh-btn')?.addEventListener('click', () => {
            this.load();
        });
    }

    async load(): Promise<void> {
        await Promise.all([
            this.loadSyncStatus(),
            this.loadWALStats(),
        ]);
        this.updateDisplay();
    }

    // =========================================================================
    // Data Loading
    // =========================================================================

    private async loadSyncStatus(): Promise<void> {
        // This would need a dedicated sync status endpoint
        // For now, use placeholder data
        this.syncSources = [
            { name: 'Tautulli', status: 'healthy', lastSync: new Date(), eventsToday: 0 },
            { name: 'Plex', status: 'healthy', lastSync: new Date(), eventsToday: 0 },
            { name: 'Jellyfin', status: 'disconnected', lastSync: null, eventsToday: 0 },
            { name: 'Emby', status: 'disconnected', lastSync: null, eventsToday: 0 },
        ];
    }

    private async loadWALStats(): Promise<void> {
        try {
            this.walStats = await this.api.getWALStats().catch(() => null);
        } catch (error) {
            logger.error('Failed to load WAL stats:', error);
        }
    }

    // =========================================================================
    // Display Updates
    // =========================================================================

    private updateDisplay(): void {
        // Update source status cards
        this.syncSources.forEach(source => {
            const statusEl = document.getElementById(`sync-${source.name.toLowerCase()}-status`);
            const lastEl = document.getElementById(`sync-${source.name.toLowerCase()}-last`);
            const eventsEl = document.getElementById(`sync-${source.name.toLowerCase()}-events`);

            if (statusEl) statusEl.textContent = source.status;
            if (lastEl) lastEl.textContent = source.lastSync ? this.formatTimeAgo(source.lastSync) : 'Never';
            if (eventsEl) eventsEl.textContent = source.eventsToday.toString();
        });

        // Update pipeline metrics with WAL data
        this.setElementText('pipeline-received', this.walStats?.total_writes.toLocaleString() || '--');
        this.setElementText('pipeline-processed', this.walStats?.total_confirms.toLocaleString() || '--');
        this.setElementText('pipeline-deduplicated', this.dedupeStats?.total_deduped?.toLocaleString() || '--');
        this.setElementText('pipeline-failed', this.dlqStats?.total_entries.toString() || '0');
        this.setElementText('pipeline-pending', this.walStats?.pending_count.toLocaleString() || '0');

        // Update WAL-specific stats
        this.setElementText('wal-status', this.walStats?.status || '--');
        this.setElementText('wal-db-size', this.walStats?.db_size_formatted || '--');
        this.setElementText('wal-write-rate', this.walStats?.write_rate_per_min?.toFixed(1) || '--');
    }
}
