// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * OverviewRenderer - Overview Dashboard Page
 *
 * Key metrics dashboard for the Data Governance tab.
 */

import { createLogger } from '../../lib/logger';
import type { API } from '../../lib/api';
import type { DedupeAuditStats, DetectionAlertStats, DedupeAuditEntry, DetectionAlert, DLQStats } from '../../lib/types';
import { BaseRenderer, GovernanceConfig, REASON_NAMES } from './BaseRenderer';

const logger = createLogger('OverviewRenderer');

/** Sync source status */
export interface SyncSourceStatus {
    name: string;
    status: 'healthy' | 'degraded' | 'error' | 'disconnected';
    lastSync: Date | null;
    eventsToday: number;
}

export class OverviewRenderer extends BaseRenderer {
    // Cached data from other renderers (shared via setters)
    private dedupeStats: DedupeAuditStats | null = null;
    private detectionStats: DetectionAlertStats | null = null;
    private dedupeEntries: DedupeAuditEntry[] = [];
    private detectionAlerts: DetectionAlert[] = [];
    private syncSources: SyncSourceStatus[] = [];
    private dlqStats: DLQStats | null = null;

    constructor(api: API, config: GovernanceConfig) {
        super(api, config);
    }

    // =========================================================================
    // Data Setters (for sharing data from other renderers)
    // =========================================================================

    setDedupeStats(stats: DedupeAuditStats | null): void {
        this.dedupeStats = stats;
    }

    setDetectionStats(stats: DetectionAlertStats | null): void {
        this.detectionStats = stats;
    }

    setDedupeEntries(entries: DedupeAuditEntry[]): void {
        this.dedupeEntries = entries;
    }

    setDetectionAlerts(alerts: DetectionAlert[]): void {
        this.detectionAlerts = alerts;
    }

    setSyncSources(sources: SyncSourceStatus[]): void {
        this.syncSources = sources;
    }

    setDLQStats(stats: DLQStats | null): void {
        this.dlqStats = stats;
    }

    getSyncSources(): SyncSourceStatus[] {
        return this.syncSources;
    }

    // =========================================================================
    // Public API
    // =========================================================================

    setupEventListeners(): void {
        // Overview has no user interactions - it's display-only
    }

    async load(): Promise<void> {
        try {
            // Load all metrics in parallel
            const [dedupeStats, detectionStats] = await Promise.all([
                this.api.getDedupeAuditStats().catch(() => null),
                this.api.getDetectionAlertStats().catch(() => null),
                this.loadSyncStatus(),
            ]);

            this.dedupeStats = dedupeStats;
            this.detectionStats = detectionStats;

            this.updateMetrics();
        } catch (error) {
            logger.error('Failed to load overview data:', error);
        }
    }

    // =========================================================================
    // Sync Status
    // =========================================================================

    async loadSyncStatus(): Promise<SyncSourceStatus[]> {
        // This would need a dedicated sync status endpoint
        // For now, return placeholder data
        this.syncSources = [
            { name: 'Tautulli', status: 'healthy', lastSync: new Date(), eventsToday: 0 },
            { name: 'Plex', status: 'healthy', lastSync: new Date(), eventsToday: 0 },
            { name: 'Jellyfin', status: 'disconnected', lastSync: null, eventsToday: 0 },
            { name: 'Emby', status: 'disconnected', lastSync: null, eventsToday: 0 },
        ];
        return this.syncSources;
    }

    // =========================================================================
    // Display Updates
    // =========================================================================

    updateMetrics(): void {
        // Total events (placeholder - would need a stats endpoint)
        const totalEventsEl = document.getElementById('governance-total-events');
        if (totalEventsEl) {
            totalEventsEl.textContent = '--';
        }

        // Dedupe rate
        const dedupeRateEl = document.getElementById('governance-dedupe-rate');
        if (dedupeRateEl && this.dedupeStats) {
            const rate = this.dedupeStats.accuracy_rate;
            dedupeRateEl.textContent = `${(rate * 100).toFixed(1)}%`;
        }

        // Active alerts
        const alertsCountEl = document.getElementById('governance-alerts-count');
        if (alertsCountEl && this.detectionStats) {
            alertsCountEl.textContent = this.detectionStats.unacknowledged.toString();
        }

        // Sync health
        const syncStatusEl = document.getElementById('governance-sync-status');
        if (syncStatusEl) {
            const healthyCount = this.syncSources.filter(s => s.status === 'healthy').length;
            const total = this.syncSources.length || 4;
            syncStatusEl.textContent = total > 0 ? `${healthyCount}/${total}` : '--';
        }

        // Failed events (from DLQ)
        const failedCountEl = document.getElementById('governance-failed-count');
        if (failedCountEl) {
            failedCountEl.textContent = this.dlqStats?.total_entries.toString() || '0';
        }

        // Data freshness
        const freshnessEl = document.getElementById('governance-data-freshness');
        if (freshnessEl) {
            const recentSync = this.syncSources.find(s => s.lastSync)?.lastSync;
            if (recentSync) {
                const ago = this.formatTimeAgo(recentSync);
                freshnessEl.textContent = ago;
            } else {
                freshnessEl.textContent = '--';
            }
        }

        this.updateActivityTimeline();
        this.updateDataSourcesGrid();
    }

    // =========================================================================
    // Activity Timeline
    // =========================================================================

    private updateActivityTimeline(): void {
        const timeline = document.getElementById('governance-activity-timeline');
        if (!timeline) return;

        // Build activity items from available data
        const activities: { time: Date; type: string; message: string }[] = [];

        // Add recent dedupe events
        if (this.dedupeEntries.length > 0) {
            this.dedupeEntries.slice(0, 5).forEach(entry => {
                activities.push({
                    time: new Date(entry.timestamp),
                    type: 'dedupe',
                    message: `Event deduplicated: ${entry.username || 'Unknown'} - ${REASON_NAMES[entry.dedupe_reason] || entry.dedupe_reason}`,
                });
            });
        }

        // Add recent detection alerts
        if (this.detectionAlerts.length > 0) {
            this.detectionAlerts.slice(0, 5).forEach(alert => {
                activities.push({
                    time: new Date(alert.created_at),
                    type: 'alert',
                    message: `Alert: ${alert.title}`,
                });
            });
        }

        // Sort by time
        activities.sort((a, b) => b.time.getTime() - a.time.getTime());

        if (activities.length === 0) {
            timeline.innerHTML = '<div class="timeline-empty">No recent activity</div>';
            return;
        }

        timeline.innerHTML = activities.slice(0, 10).map(activity => `
            <div class="timeline-item timeline-${activity.type}">
                <div class="timeline-time">${this.formatTimeAgo(activity.time)}</div>
                <div class="timeline-message">${this.escapeHtml(activity.message)}</div>
            </div>
        `).join('');
    }

    // =========================================================================
    // Data Sources Grid
    // =========================================================================

    private updateDataSourcesGrid(): void {
        const sourcesGrid = document.getElementById('governance-sources');
        if (!sourcesGrid) return;

        if (this.syncSources.length === 0) {
            sourcesGrid.innerHTML = '<div class="source-empty">No sources configured</div>';
            return;
        }

        sourcesGrid.innerHTML = this.syncSources.map(source => `
            <div class="source-card source-${source.status}">
                <div class="source-header">
                    <span class="source-name">${this.escapeHtml(source.name)}</span>
                    <span class="source-status-badge">${source.status}</span>
                </div>
                <div class="source-meta">
                    <span>Last sync: ${source.lastSync ? this.formatTimeAgo(source.lastSync) : 'Never'}</span>
                    <span>Events today: ${source.eventsToday}</span>
                </div>
            </div>
        `).join('');
    }
}
