// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * DataFreshnessManager - Manages data freshness indicators and auto-refresh
 *
 * Features:
 * - Stale data warning when data is outdated
 * - Auto-refresh toggle control
 * - Loading progress bar
 *
 * @see /docs/working/UI_UX_AUDIT.md
 */

import { SafeStorage } from '../lib/utils/SafeStorage';

/**
 * Window interface extension for data freshness E2E testing
 */
declare global {
    interface Window {
        __lastDataUpdate?: number;
        __autoRefreshEnabled?: boolean;
    }
}

export interface DataFreshnessConfig {
    /** Stale threshold in milliseconds (default: 5 minutes) */
    staleThreshold: number;
    /** Auto-refresh interval in milliseconds (default: 60 seconds) */
    refreshInterval: number;
    /** Check interval for stale data in milliseconds (default: 30 seconds) */
    checkInterval: number;
}

const DEFAULT_CONFIG: DataFreshnessConfig = {
    staleThreshold: 5 * 60 * 1000,  // 5 minutes
    refreshInterval: 60 * 1000,     // 60 seconds
    checkInterval: 30 * 1000,       // 30 seconds
};

export class DataFreshnessManager {
    private config: DataFreshnessConfig;
    private lastUpdateTime: number = 0;
    private autoRefreshEnabled: boolean = true;
    private autoRefreshTimer: number | null = null;
    private staleCheckTimer: number | null = null;
    private staleDismissed: boolean = false;
    private refreshCallback: (() => void) | null = null;

    // DOM elements
    private progressBar: HTMLElement | null = null;
    private staleWarning: HTMLElement | null = null;
    private staleTimeDisplay: HTMLElement | null = null;
    private autoRefreshToggle: HTMLInputElement | null = null;
    private refreshIntervalDisplay: HTMLElement | null = null;

    constructor(config: Partial<DataFreshnessConfig> = {}) {
        this.config = { ...DEFAULT_CONFIG, ...config };
        this.loadPersistedState();
    }

    /**
     * Initialize the manager and bind to DOM elements
     */
    init(): void {
        this.bindElements();
        this.setupEventListeners();
        this.updateToggleUI();
        this.startStaleCheck();

        // Start auto-refresh if enabled
        if (this.autoRefreshEnabled) {
            this.startAutoRefresh();
        }

        // Expose for global access (for stale warning testing)
        window.__lastDataUpdate = this.lastUpdateTime;
    }

    /**
     * Set the callback function for refresh actions
     */
    setRefreshCallback(callback: () => void): void {
        this.refreshCallback = callback;
    }

    /**
     * Bind to DOM elements
     */
    private bindElements(): void {
        this.progressBar = document.getElementById('global-progress-bar');
        this.staleWarning = document.getElementById('stale-data-warning');
        this.staleTimeDisplay = document.getElementById('stale-data-time');
        this.autoRefreshToggle = document.getElementById('auto-refresh-toggle') as HTMLInputElement;
        this.refreshIntervalDisplay = document.getElementById('refresh-interval-display');
    }

    /**
     * Setup event listeners
     */
    private setupEventListeners(): void {
        // Auto-refresh toggle
        if (this.autoRefreshToggle) {
            this.autoRefreshToggle.addEventListener('change', () => {
                this.autoRefreshEnabled = this.autoRefreshToggle!.checked;
                this.savePersistedState();

                if (this.autoRefreshEnabled) {
                    this.startAutoRefresh();
                } else {
                    this.stopAutoRefresh();
                }
            });
        }

        // Stale warning dismiss button
        const dismissBtn = this.staleWarning?.querySelector('.stale-warning-dismiss');
        if (dismissBtn) {
            dismissBtn.addEventListener('click', () => {
                this.dismissStaleWarning();
            });
        }

        // Stale warning refresh button
        const refreshBtn = this.staleWarning?.querySelector('.stale-warning-refresh');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                this.triggerRefresh();
            });
        }
    }

    /**
     * Load persisted state from localStorage
     */
    private loadPersistedState(): void {
        const savedAutoRefresh = SafeStorage.getItem('auto-refresh-enabled');
        if (savedAutoRefresh !== null) {
            this.autoRefreshEnabled = savedAutoRefresh === 'true';
        }

        const savedLastUpdate = SafeStorage.getItem('last-data-update');
        if (savedLastUpdate) {
            this.lastUpdateTime = parseInt(savedLastUpdate, 10);
        }
    }

    /**
     * Save persisted state to localStorage
     */
    private savePersistedState(): void {
        SafeStorage.setItem('auto-refresh-enabled', String(this.autoRefreshEnabled));
        SafeStorage.setItem('last-data-update', String(this.lastUpdateTime));
    }

    /**
     * Update toggle UI to match current state
     */
    private updateToggleUI(): void {
        if (this.autoRefreshToggle) {
            this.autoRefreshToggle.checked = this.autoRefreshEnabled;
        }

        // Update interval display
        if (this.refreshIntervalDisplay) {
            const seconds = Math.round(this.config.refreshInterval / 1000);
            this.refreshIntervalDisplay.textContent = seconds >= 60
                ? `${Math.round(seconds / 60)}m`
                : `${seconds}s`;
        }

        // Update global state for testing
        window.__autoRefreshEnabled = this.autoRefreshEnabled;
    }

    /**
     * Mark data as updated (call after successful data load)
     */
    markDataUpdated(): void {
        this.lastUpdateTime = Date.now();
        this.staleDismissed = false;
        this.savePersistedState();
        this.hideStaleWarning();

        // Update global state for testing
        window.__lastDataUpdate = this.lastUpdateTime;
    }

    /**
     * Start the auto-refresh timer
     */
    startAutoRefresh(): void {
        this.stopAutoRefresh();

        if (!this.autoRefreshEnabled) return;

        this.autoRefreshTimer = window.setInterval(() => {
            if (this.refreshCallback) {
                this.refreshCallback();
            }
        }, this.config.refreshInterval);
    }

    /**
     * Stop the auto-refresh timer
     */
    stopAutoRefresh(): void {
        if (this.autoRefreshTimer !== null) {
            window.clearInterval(this.autoRefreshTimer);
            this.autoRefreshTimer = null;
        }
    }

    /**
     * Start the stale data check timer
     */
    private startStaleCheck(): void {
        this.staleCheckTimer = window.setInterval(() => {
            this.checkStaleData();
        }, this.config.checkInterval);

        // Initial check
        this.checkStaleData();
    }

    /**
     * Check if data is stale and show warning if needed
     */
    private checkStaleData(): void {
        if (this.staleDismissed) return;
        if (this.lastUpdateTime === 0) return; // No data loaded yet

        const now = Date.now();
        const timeSinceUpdate = now - this.lastUpdateTime;

        if (timeSinceUpdate > this.config.staleThreshold) {
            this.showStaleWarning(timeSinceUpdate);
        } else {
            this.hideStaleWarning();
        }
    }

    /**
     * Show the stale data warning
     */
    private showStaleWarning(timeSinceUpdate: number): void {
        if (!this.staleWarning || !this.staleTimeDisplay) return;

        // Format time since update
        const minutes = Math.round(timeSinceUpdate / 60000);
        const timeText = minutes === 1
            ? '1 minute ago'
            : minutes < 60
                ? `${minutes} minutes ago`
                : `${Math.round(minutes / 60)} hours ago`;

        this.staleTimeDisplay.textContent = timeText;
        this.staleWarning.classList.remove('hidden');
    }

    /**
     * Hide the stale data warning
     */
    private hideStaleWarning(): void {
        if (!this.staleWarning) return;
        this.staleWarning.classList.add('hidden');
    }

    /**
     * Dismiss the stale warning until next data refresh
     */
    private dismissStaleWarning(): void {
        this.staleDismissed = true;
        this.hideStaleWarning();
    }

    /**
     * Trigger a manual refresh
     */
    private triggerRefresh(): void {
        this.hideStaleWarning();
        if (this.refreshCallback) {
            this.refreshCallback();
        }
    }

    // =========================================================================
    // Progress Bar Methods
    // =========================================================================

    /**
     * Show the global progress bar
     */
    showProgress(): void {
        if (!this.progressBar) return;
        this.progressBar.classList.remove('hidden');
        this.progressBar.classList.remove('determinate');
    }

    /**
     * Hide the global progress bar
     */
    hideProgress(): void {
        if (!this.progressBar) return;
        this.progressBar.classList.add('hidden');
    }

    /**
     * Set determinate progress value (0-100)
     */
    setProgress(percent: number): void {
        if (!this.progressBar) return;

        this.progressBar.classList.remove('hidden');
        this.progressBar.classList.add('determinate');

        const fill = this.progressBar.querySelector('.progress-fill') as HTMLElement;
        if (fill) {
            fill.style.width = `${Math.min(100, Math.max(0, percent))}%`;
        }

        this.progressBar.setAttribute('aria-valuenow', String(Math.round(percent)));
    }

    // =========================================================================
    // Cleanup
    // =========================================================================

    /**
     * Destroy the manager and cleanup timers
     */
    destroy(): void {
        this.stopAutoRefresh();

        if (this.staleCheckTimer !== null) {
            window.clearInterval(this.staleCheckTimer);
            this.staleCheckTimer = null;
        }
    }
}
