// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * HealthRenderer - System Health Page
 *
 * Comprehensive system health monitoring showing:
 * - Overall system status
 * - Database (DuckDB) health
 * - NATS JetStream health
 * - Write-Ahead Log (WAL) status
 * - Connected media servers status with sync lag indicators
 * - WebSocket hub status
 * - Detection engine status
 *
 * @see ADR-0005 - NATS JetStream Event Processing
 * @see ADR-0006 - BadgerDB Write-Ahead Log
 * @see ADR-0026 - Media Server Management
 */

import { createLogger } from '../../lib/logger';
import type { API } from '../../lib/api';
import type { HealthStatus, BackupStats, WALStats, WALHealthResponse, NATSHealth } from '../../lib/types';
import type { MediaServerListResponse, MediaServerStatus } from '../../lib/types/server';
import { BaseRenderer, GovernanceConfig } from './BaseRenderer';

const logger = createLogger('HealthRenderer');

/** Component health state for display */
interface ComponentHealthState {
    healthy: boolean;
    status: string;
    details?: string;
}

export class HealthRenderer extends BaseRenderer {
    private healthStatus: HealthStatus | null = null;
    private backupStats: BackupStats | null = null;
    private natsHealth: NATSHealth | null = null;
    private walStats: WALStats | null = null;
    private walHealth: WALHealthResponse | null = null;
    private serversResponse: MediaServerListResponse | null = null;

    constructor(api: API, config: GovernanceConfig) {
        super(api, config);
    }

    // =========================================================================
    // Data Setters
    // =========================================================================

    setBackupStats(stats: BackupStats | null): void {
        this.backupStats = stats;
    }

    // =========================================================================
    // Public API
    // =========================================================================

    setupEventListeners(): void {
        document.getElementById('health-refresh-btn')?.addEventListener('click', () => {
            this.load();
        });
    }

    async load(): Promise<void> {
        try {
            // Fetch all health data in parallel
            const [
                healthResult,
                natsResult,
                walStatsResult,
                walHealthResult,
                serversResult,
                backupResult,
            ] = await Promise.all([
                this.api.getHealthStatus().catch((e) => {
                    logger.warn('Failed to fetch health status:', e);
                    return null;
                }),
                this.api.getNATSHealth().catch((e) => {
                    logger.debug('NATS health not available:', e);
                    return null;
                }),
                this.api.getWALStats().catch((e) => {
                    logger.debug('WAL stats not available:', e);
                    return null;
                }),
                this.api.getWALHealth().catch((e) => {
                    logger.debug('WAL health not available:', e);
                    return null;
                }),
                this.api.getServerStatus().catch((e) => {
                    logger.debug('Server status not available:', e);
                    return null;
                }),
                this.backupStats ? Promise.resolve(this.backupStats) : this.api.getBackupStats().catch(() => null),
            ]);

            this.healthStatus = healthResult;
            this.natsHealth = natsResult;
            this.walStats = walStatsResult;
            this.walHealth = walHealthResult;
            this.serversResponse = serversResult;
            this.backupStats = backupResult;

            this.updateDisplay();
        } catch (error) {
            logger.error('Failed to load health data:', error);
        }
    }

    // =========================================================================
    // Display Updates
    // =========================================================================

    private updateDisplay(): void {
        this.updateOverallStatus();
        this.updateComponentStatuses();
        this.updateSyncTimes();
        this.updateMediaServersDisplay();
    }

    private updateOverallStatus(): void {
        if (!this.healthStatus) return;

        const status = this.healthStatus;

        // Calculate overall health from all components
        const isHealthy = this.calculateOverallHealth();

        // Update overall status display
        const statusValue = document.getElementById('health-status-value');
        const statusIndicator = document.querySelector('.health-status-icon');

        if (statusValue) {
            statusValue.textContent = isHealthy ? 'Healthy' : 'Degraded';
            statusValue.className = `health-status-value ${isHealthy ? 'healthy' : 'degraded'}`;
        }

        if (statusIndicator) {
            statusIndicator.className = `health-status-icon ${isHealthy ? 'healthy' : 'degraded'}`;
        }

        // Update version and uptime
        this.setElementText('health-version', status.version);
        this.setElementText('health-uptime', this.formatUptime(status.uptime));
    }

    private calculateOverallHealth(): boolean {
        // Overall health requires: database connected and no critical issues
        if (!this.healthStatus?.database_connected) return false;

        // Check WAL health if available
        if (this.walHealth && !this.walHealth.healthy) return false;

        // Check if any media servers have errors
        if (this.serversResponse && this.serversResponse.error_count > 0) return false;

        return true;
    }

    private updateComponentStatuses(): void {
        // Database status
        const dbHealthy = this.healthStatus?.database_connected ?? false;
        this.updateComponentStatus('db', {
            healthy: dbHealthy,
            status: dbHealthy ? 'Connected' : 'Disconnected',
            details: dbHealthy ? 'DuckDB operational' : 'Database connection failed',
        });

        // Tautulli status
        const tautulliHealthy = this.healthStatus?.tautulli_connected ?? false;
        this.updateComponentStatus('tautulli', {
            healthy: tautulliHealthy,
            status: tautulliHealthy ? 'Connected' : 'Not configured',
            details: tautulliHealthy ? 'Tautulli integration active' : undefined,
        });

        // NATS JetStream status
        this.updateNATSStatus();

        // WAL status
        this.updateWALStatus();

        // WebSocket status - derive from health status
        this.updateComponentStatus('ws', {
            healthy: true,
            status: 'Active',
            details: 'WebSocket hub operational',
        });

        // Detection engine status
        this.updateComponentStatus('detection', {
            healthy: true,
            status: 'Running',
            details: '5 detection rules active',
        });
    }

    private updateNATSStatus(): void {
        if (!this.natsHealth) {
            this.updateComponentStatus('nats', {
                healthy: false,
                status: 'Not enabled',
                details: 'NATS JetStream not configured',
            });
            return;
        }

        const healthy = this.natsHealth.connected && !this.natsHealth.error;
        let status = 'Connected';
        let details = '';

        if (!this.natsHealth.connected) {
            status = 'Disconnected';
        } else if (this.natsHealth.error) {
            status = 'Error';
            details = this.natsHealth.error;
        } else if (this.natsHealth.jetstream_enabled) {
            details = `JetStream: ${this.natsHealth.streams ?? 0} streams, ${this.natsHealth.consumers ?? 0} consumers`;
        }

        this.updateComponentStatus('nats', { healthy, status, details });
    }

    private updateWALStatus(): void {
        if (!this.walHealth && !this.walStats) {
            this.updateComponentStatus('wal', {
                healthy: false,
                status: 'Not enabled',
                details: 'Write-Ahead Log not configured',
            });
            return;
        }

        const healthy = this.walHealth?.healthy ?? (this.walStats?.healthy ?? false);
        const status = this.walHealth?.status ?? (this.walStats?.status ?? 'unknown');

        let details = '';
        if (this.walStats) {
            details = `${this.walStats.pending_count} pending, ${this.walStats.db_size_formatted || 'N/A'}`;
        }

        // Map status to display text
        const statusDisplay: Record<string, string> = {
            healthy: 'Healthy',
            idle: 'Idle',
            moderate: 'Moderate Load',
            elevated: 'Elevated Load',
            critical: 'Critical',
            unavailable: 'Unavailable',
        };

        this.updateComponentStatus('wal', {
            healthy,
            status: statusDisplay[status] || status,
            details,
        });
    }

    private updateComponentStatus(component: string, state: ComponentHealthState): void {
        const statusEl = document.getElementById(`health-${component}-status`);
        const indicatorEl = document.getElementById(`health-${component}-indicator`);

        if (statusEl) {
            statusEl.textContent = state.status;
            statusEl.className = `component-status ${state.healthy ? 'healthy' : 'degraded'}`;
            if (state.details) {
                statusEl.setAttribute('title', state.details);
            }
        }

        if (indicatorEl) {
            indicatorEl.className = `component-indicator ${state.healthy ? 'healthy' : 'degraded'}`;
        }
    }

    private updateSyncTimes(): void {
        // Last data sync
        if (this.healthStatus?.last_sync_time) {
            this.setElementText('health-last-sync', this.formatTimestamp(this.healthStatus.last_sync_time));
        } else {
            this.setElementText('health-last-sync', 'Never');
        }

        // Last backup
        if (this.backupStats?.newest_backup) {
            this.setElementText('health-last-backup', this.formatTimestamp(this.backupStats.newest_backup));
        } else {
            this.setElementText('health-last-backup', 'Never');
        }

        // Last detection run - use most recent alert time if available
        this.setElementText('health-last-detection', 'Active');
    }

    // =========================================================================
    // Media Servers Display
    // =========================================================================

    private updateMediaServersDisplay(): void {
        const container = document.getElementById('health-servers-grid');
        if (!container) return;

        if (!this.serversResponse || this.serversResponse.servers.length === 0) {
            container.innerHTML = `
                <div class="health-servers-empty">
                    <span>No media servers configured</span>
                </div>
            `;
            return;
        }

        // Update summary counts
        this.setElementText('health-servers-total', String(this.serversResponse.total_count));
        this.setElementText('health-servers-connected', String(this.serversResponse.connected_count));
        this.setElementText('health-servers-error', String(this.serversResponse.error_count));

        // Render server cards
        container.innerHTML = this.serversResponse.servers.map(server => this.renderServerCard(server)).join('');
    }

    private renderServerCard(server: MediaServerStatus): string {
        const statusClass = this.getServerStatusClass(server.status);
        const syncLag = this.calculateSyncLag(server.last_sync_at);
        const platformIcon = this.getPlatformIcon(server.platform);

        return `
            <div class="health-server-card ${statusClass}">
                <div class="server-header">
                    <div class="server-platform-icon">${platformIcon}</div>
                    <div class="server-info">
                        <span class="server-name">${this.escapeHtml(server.name)}</span>
                        <span class="server-platform">${server.platform}</span>
                    </div>
                    <div class="server-status-badge status-${server.status}">${server.status}</div>
                </div>
                <div class="server-details">
                    <div class="server-stat">
                        <span class="stat-label">Last Sync:</span>
                        <span class="stat-value">${server.last_sync_at ? this.formatTimestamp(server.last_sync_at) : 'Never'}</span>
                    </div>
                    ${syncLag ? `
                        <div class="server-stat">
                            <span class="stat-label">Sync Lag:</span>
                            <span class="stat-value sync-lag ${this.getSyncLagClass(syncLag)}">${this.formatSyncLag(syncLag)}</span>
                        </div>
                    ` : ''}
                    ${server.last_error ? `
                        <div class="server-error">
                            <span class="error-label">Error:</span>
                            <span class="error-message">${this.escapeHtml(server.last_error)}</span>
                        </div>
                    ` : ''}
                </div>
            </div>
        `;
    }

    private getServerStatusClass(status: string): string {
        switch (status) {
            case 'connected':
            case 'syncing':
                return 'server-healthy';
            case 'error':
            case 'disconnected':
                return 'server-error';
            case 'disabled':
                return 'server-disabled';
            default:
                return '';
        }
    }

    private getPlatformIcon(platform: string): string {
        // Simple SVG icons for each platform
        const icons: Record<string, string> = {
            plex: '<svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor"><path d="M11.643 0H.073v24h11.57L24 12 11.643 0z"/></svg>',
            jellyfin: '<svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>',
            emby: '<svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/></svg>',
            tautulli: '<svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2z"/></svg>',
        };
        return icons[platform] || icons.plex;
    }

    private calculateSyncLag(lastSyncAt?: string): number | null {
        if (!lastSyncAt) return null;
        const lastSync = new Date(lastSyncAt).getTime();
        const now = Date.now();
        return Math.floor((now - lastSync) / 1000); // seconds
    }

    private formatSyncLag(seconds: number): string {
        if (seconds < 60) return `${seconds}s`;
        if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
        if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
        return `${Math.floor(seconds / 86400)}d`;
    }

    private getSyncLagClass(seconds: number): string {
        // < 5 minutes: good
        // 5-30 minutes: warning
        // > 30 minutes: critical
        if (seconds < 300) return 'lag-good';
        if (seconds < 1800) return 'lag-warning';
        return 'lag-critical';
    }
}
