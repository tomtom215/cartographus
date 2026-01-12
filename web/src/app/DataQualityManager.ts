// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * DataQualityManager - Displays data quality indicators
 *
 * Shows system health, sync status, and data completeness indicators.
 * Provides visibility into data freshness and connectivity status.
 */

import type { API, HealthStatus } from '../lib/api';
import { createLogger } from '../lib/logger';

const logger = createLogger('DataQualityManager');

export interface DataQualityConfig {
    /** Refresh interval in milliseconds (default: 60 seconds) */
    refreshInterval: number;
}

const DEFAULT_CONFIG: DataQualityConfig = {
    refreshInterval: 60 * 1000, // 60 seconds
};

export class DataQualityManager {
    private api: API;
    private config: DataQualityConfig;
    private refreshTimer: number | null = null;
    private lastStatus: HealthStatus | null = null;

    // DOM elements
    private container: HTMLElement | null = null;
    private statusIndicator: HTMLElement | null = null;
    private syncTimeDisplay: HTMLElement | null = null;
    private connectionStatus: HTMLElement | null = null;

    constructor(api: API, config: Partial<DataQualityConfig> = {}) {
        this.api = api;
        this.config = { ...DEFAULT_CONFIG, ...config };
    }

    /**
     * Initialize the manager and bind to DOM elements
     */
    init(): void {
        this.bindElements();
        this.fetchAndUpdate();
        this.startRefreshTimer();
    }

    /**
     * Bind to DOM elements
     */
    private bindElements(): void {
        this.container = document.getElementById('data-quality-indicator');
        this.statusIndicator = document.getElementById('data-quality-status');
        this.syncTimeDisplay = document.getElementById('data-quality-sync-time');
        this.connectionStatus = document.getElementById('data-quality-connection');
    }

    /**
     * Fetch health status and update the UI
     */
    async fetchAndUpdate(): Promise<void> {
        try {
            const status = await this.api.getHealthStatus();
            this.lastStatus = status;
            this.updateUI(status);
        } catch (error) {
            logger.error('Failed to fetch health status:', error);
            this.showError();
        }
    }

    /**
     * Update the UI with health status
     */
    private updateUI(status: HealthStatus): void {
        if (!this.container) return;

        // Show the container
        this.container.classList.remove('hidden');

        // Update status indicator
        if (this.statusIndicator) {
            this.statusIndicator.className = 'data-quality-status';
            this.statusIndicator.classList.add(`status-${status.status}`);
            this.statusIndicator.setAttribute('aria-label',
                `System status: ${status.status}`
            );

            // Update status text
            const statusText = this.statusIndicator.querySelector('.status-text');
            if (statusText) {
                statusText.textContent = status.status === 'healthy'
                    ? 'Healthy'
                    : 'Degraded';
            }
        }

        // Update sync time
        if (this.syncTimeDisplay) {
            if (status.last_sync_time) {
                const syncDate = new Date(status.last_sync_time);
                const timeAgo = this.formatTimeAgo(syncDate);
                this.syncTimeDisplay.textContent = `Synced ${timeAgo}`;
                this.syncTimeDisplay.setAttribute('title', syncDate.toLocaleString());
            } else {
                this.syncTimeDisplay.textContent = 'Never synced';
            }
        }

        // Update connection status
        if (this.connectionStatus) {
            const dbIcon = this.connectionStatus.querySelector('.connection-db');
            const tautulliIcon = this.connectionStatus.querySelector('.connection-tautulli');

            if (dbIcon) {
                dbIcon.classList.toggle('connected', status.database_connected);
                dbIcon.setAttribute('aria-label',
                    `Database: ${status.database_connected ? 'connected' : 'disconnected'}`
                );
            }

            if (tautulliIcon) {
                tautulliIcon.classList.toggle('connected', status.tautulli_connected);
                tautulliIcon.setAttribute('aria-label',
                    `Tautulli: ${status.tautulli_connected ? 'connected' : 'disconnected'}`
                );
            }
        }
    }

    /**
     * Show error state when health check fails
     */
    private showError(): void {
        if (!this.container) return;

        this.container.classList.remove('hidden');

        if (this.statusIndicator) {
            this.statusIndicator.className = 'data-quality-status status-error';
            const statusText = this.statusIndicator.querySelector('.status-text');
            if (statusText) {
                statusText.textContent = 'Error';
            }
        }

        if (this.syncTimeDisplay) {
            this.syncTimeDisplay.textContent = 'Unable to check';
        }
    }

    /**
     * Format a date as relative time (e.g., "5 minutes ago")
     */
    private formatTimeAgo(date: Date): string {
        const now = Date.now();
        const diff = now - date.getTime();

        const minutes = Math.floor(diff / 60000);
        const hours = Math.floor(diff / 3600000);
        const days = Math.floor(diff / 86400000);

        if (minutes < 1) return 'just now';
        if (minutes === 1) return '1 minute ago';
        if (minutes < 60) return `${minutes} minutes ago`;
        if (hours === 1) return '1 hour ago';
        if (hours < 24) return `${hours} hours ago`;
        if (days === 1) return '1 day ago';
        return `${days} days ago`;
    }

    /**
     * Start the refresh timer
     */
    private startRefreshTimer(): void {
        this.stopRefreshTimer();
        this.refreshTimer = window.setInterval(() => {
            this.fetchAndUpdate();
        }, this.config.refreshInterval);
    }

    /**
     * Stop the refresh timer
     */
    private stopRefreshTimer(): void {
        if (this.refreshTimer !== null) {
            window.clearInterval(this.refreshTimer);
            this.refreshTimer = null;
        }
    }

    /**
     * Get the last known status
     */
    getLastStatus(): HealthStatus | null {
        return this.lastStatus;
    }

    /**
     * Force a refresh of the health status
     */
    async refresh(): Promise<void> {
        await this.fetchAndUpdate();
    }

    /**
     * Cleanup and destroy the manager
     */
    destroy(): void {
        this.stopRefreshTimer();
    }
}
