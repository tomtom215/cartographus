// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * SyncProgressComponent - Reusable Progress UI Component
 *
 * A production-grade progress display component for sync operations.
 * Features:
 * - Real-time progress bar with percentage
 * - Detailed statistics (processed, imported, skipped, errors)
 * - Human-readable ETA and elapsed time
 * - Expandable error log
 * - Accessibility compliant (WCAG 2.1 AA)
 *
 * Can be used for:
 * - Tautulli database import
 * - Plex historical sync
 * - Server sync operations
 */

import { createLogger } from '../lib/logger';
import type {
    SyncProgress,
    SyncStatus,
    SyncProgressDisplayConfig,
    SyncError,
} from '../lib/types/sync';
import {
    formatDuration,
    formatNumber,
    formatETA,
    DEFAULT_STATUS_LABELS,
} from '../lib/types/sync';

const logger = createLogger('SyncProgressComponent');

// ============================================================================
// Default Configuration
// ============================================================================

const DEFAULT_DISPLAY_CONFIG: Required<SyncProgressDisplayConfig> = {
    showProgressBar: true,
    showDetailedStats: true,
    showErrorLog: true,
    maxErrorsDisplayed: 10,
    showETA: true,
    statusLabels: DEFAULT_STATUS_LABELS,
};

// ============================================================================
// Component Class
// ============================================================================

/**
 * Reusable sync progress display component.
 *
 * @example
 * ```typescript
 * const progressComponent = new SyncProgressComponent('import-progress', {
 *     showDetailedStats: true,
 *     showErrorLog: true,
 * });
 *
 * // Update with new progress data
 * progressComponent.update(progressData);
 *
 * // Clean up when done
 * progressComponent.destroy();
 * ```
 */
export class SyncProgressComponent {
    private container: HTMLElement | null = null;
    private config: Required<SyncProgressDisplayConfig>;
    private currentProgress: SyncProgress | null = null;
    private errorLogExpanded = false;
    private initialized = false;

    // Event handler references for cleanup
    private toggleErrorLogHandler: (() => void) | null = null;

    constructor(
        private containerId: string,
        config: SyncProgressDisplayConfig = {}
    ) {
        this.config = { ...DEFAULT_DISPLAY_CONFIG, ...config };
    }

    // ========================================================================
    // Lifecycle
    // ========================================================================

    /**
     * Initialize the component and render initial state.
     */
    init(): void {
        this.container = document.getElementById(this.containerId);
        if (!this.container) {
            logger.warn('Container not found, creating', { containerId: this.containerId });
            this.createContainer();
        }

        this.initialized = true;
        this.render();
        this.setupEventListeners();
        logger.debug('SyncProgressComponent initialized', { containerId: this.containerId });
    }

    /**
     * Create the container if it doesn't exist.
     */
    private createContainer(): void {
        const container = document.createElement('div');
        container.id = this.containerId;
        container.className = 'sync-progress-container';
        container.setAttribute('role', 'region');
        container.setAttribute('aria-label', 'Sync progress');
        document.body.appendChild(container);
        this.container = container;
    }

    /**
     * Clean up resources.
     */
    destroy(): void {
        this.removeEventListeners();
        this.toggleErrorLogHandler = null;
        this.container = null;
        this.currentProgress = null;
        this.initialized = false;
    }

    // ========================================================================
    // Event Handling
    // ========================================================================

    /**
     * Set up event listeners with stored references for cleanup.
     */
    private setupEventListeners(): void {
        if (!this.container) return;

        this.toggleErrorLogHandler = () => {
            this.errorLogExpanded = !this.errorLogExpanded;
            this.render();
        };

        // Use event delegation for error log toggle
        this.container.addEventListener('click', (e) => {
            const target = e.target as HTMLElement;
            if (target.closest('.sync-error-toggle')) {
                this.toggleErrorLogHandler?.();
            }
        });
    }

    /**
     * Remove event listeners for cleanup.
     */
    private removeEventListeners(): void {
        // Event listeners are removed when container is destroyed
    }

    // ========================================================================
    // Update & Render
    // ========================================================================

    /**
     * Update the component with new progress data.
     */
    update(progress: SyncProgress | null): void {
        const prevProgress = this.currentProgress;
        this.currentProgress = progress;

        // Only re-render if there are meaningful changes
        if (!this.hasSignificantChange(prevProgress, progress)) {
            return;
        }

        this.render();

        // Announce status changes for screen readers
        if (prevProgress?.status !== progress?.status && progress) {
            this.announceStatusChange(progress.status);
        }
    }

    /**
     * Check if progress has changed significantly enough to re-render.
     */
    private hasSignificantChange(prev: SyncProgress | null, current: SyncProgress | null): boolean {
        if (!prev && !current) return false;
        if (!prev || !current) return true;
        if (prev.status !== current.status) return true;
        if (Math.abs(prev.progress_percent - current.progress_percent) >= 1) return true;
        if (prev.error_count !== current.error_count) return true;
        return false;
    }

    /**
     * Announce status change for screen readers.
     */
    private announceStatusChange(status: SyncStatus): void {
        const label = this.config.statusLabels[status] || status;
        const announcement = document.createElement('div');
        announcement.setAttribute('role', 'status');
        announcement.setAttribute('aria-live', 'polite');
        announcement.className = 'sr-only';
        announcement.textContent = `Sync status: ${label}`;
        this.container?.appendChild(announcement);
        setTimeout(() => announcement.remove(), 1000);
    }

    /**
     * Main render method.
     */
    private render(): void {
        if (!this.container || !this.initialized) return;

        const progress = this.currentProgress;

        if (!progress) {
            this.container.innerHTML = this.renderIdle();
            return;
        }

        const statusClass = this.getStatusClass(progress.status);

        this.container.innerHTML = `
            <div class="sync-progress-panel ${statusClass}">
                ${this.renderStatusHeader(progress)}
                ${this.config.showProgressBar && progress.status === 'running' ? this.renderProgressBar(progress) : ''}
                ${this.config.showDetailedStats ? this.renderStats(progress) : ''}
                ${this.config.showETA && progress.status === 'running' ? this.renderETA(progress) : ''}
                ${this.config.showErrorLog && progress.error_count > 0 ? this.renderErrorLog(progress) : ''}
            </div>
        `;
    }

    /**
     * Render idle state.
     */
    private renderIdle(): string {
        return `
            <div class="sync-progress-panel sync-status-idle">
                <div class="sync-status-header">
                    <span class="sync-status-icon">${this.getStatusIcon('idle')}</span>
                    <span class="sync-status-text">Ready to start</span>
                </div>
            </div>
        `;
    }

    /**
     * Render status header.
     */
    private renderStatusHeader(progress: SyncProgress): string {
        const statusLabel = this.config.statusLabels[progress.status] || progress.status;
        const statusIcon = this.getStatusIcon(progress.status);

        return `
            <div class="sync-status-header">
                <span class="sync-status-icon" aria-hidden="true">${statusIcon}</span>
                <span class="sync-status-text">${this.escapeHtml(statusLabel)}</span>
                ${progress.dry_run ? '<span class="sync-dry-run-badge">Dry Run</span>' : ''}
            </div>
        `;
    }

    /**
     * Render progress bar with accessibility attributes.
     */
    private renderProgressBar(progress: SyncProgress): string {
        const percent = Math.min(100, Math.max(0, progress.progress_percent));
        const percentFormatted = percent.toFixed(1);

        return `
            <div class="sync-progress-bar-container">
                <div class="sync-progress-bar"
                     role="progressbar"
                     aria-valuenow="${percent}"
                     aria-valuemin="0"
                     aria-valuemax="100"
                     aria-label="Sync progress: ${percentFormatted}%">
                    <div class="sync-progress-bar-fill" style="width: ${percent}%"></div>
                </div>
                <span class="sync-progress-percent">${percentFormatted}%</span>
            </div>
        `;
    }

    /**
     * Render detailed statistics.
     */
    private renderStats(progress: SyncProgress): string {
        const showRate = progress.status === 'running' && progress.records_per_second > 0;

        return `
            <div class="sync-stats-grid" aria-label="Sync statistics">
                <div class="sync-stat">
                    <span class="sync-stat-label">Progress</span>
                    <span class="sync-stat-value" data-testid="progress-count">
                        ${formatNumber(progress.processed_records)} / ${formatNumber(progress.total_records)}
                    </span>
                </div>
                <div class="sync-stat">
                    <span class="sync-stat-label">Imported</span>
                    <span class="sync-stat-value sync-stat-success" data-testid="imported-count">
                        ${formatNumber(progress.imported_records)}
                    </span>
                </div>
                <div class="sync-stat">
                    <span class="sync-stat-label">Skipped</span>
                    <span class="sync-stat-value sync-stat-warning" data-testid="skipped-count">
                        ${formatNumber(progress.skipped_records)}
                    </span>
                </div>
                <div class="sync-stat">
                    <span class="sync-stat-label">Errors</span>
                    <span class="sync-stat-value ${progress.error_count > 0 ? 'sync-stat-error' : ''}" data-testid="error-count">
                        ${formatNumber(progress.error_count)}
                    </span>
                </div>
                ${showRate ? `
                    <div class="sync-stat">
                        <span class="sync-stat-label">Speed</span>
                        <span class="sync-stat-value" data-testid="speed">
                            ${progress.records_per_second.toFixed(1)} rec/s
                        </span>
                    </div>
                ` : ''}
                <div class="sync-stat">
                    <span class="sync-stat-label">Elapsed</span>
                    <span class="sync-stat-value" data-testid="elapsed">
                        ${formatDuration(progress.elapsed_seconds)}
                    </span>
                </div>
            </div>
        `;
    }

    /**
     * Render ETA section.
     */
    private renderETA(progress: SyncProgress): string {
        const eta = formatETA(progress.estimated_remaining_seconds);

        return `
            <div class="sync-eta" aria-live="polite">
                <span class="sync-eta-label">Estimated time remaining:</span>
                <span class="sync-eta-value" data-testid="eta">${eta}</span>
            </div>
        `;
    }

    /**
     * Render expandable error log.
     */
    private renderErrorLog(progress: SyncProgress): string {
        const errors = progress.errors || [];
        const displayErrors = errors.slice(0, this.config.maxErrorsDisplayed);
        const hasMore = errors.length > this.config.maxErrorsDisplayed;

        return `
            <div class="sync-error-section">
                <button type="button" class="sync-error-toggle" aria-expanded="${this.errorLogExpanded}">
                    ${this.getExpandIcon(this.errorLogExpanded)}
                    <span>Errors (${formatNumber(progress.error_count)})</span>
                </button>
                ${this.errorLogExpanded ? `
                    <div class="sync-error-log" role="log" aria-label="Error log">
                        <ul class="sync-error-list">
                            ${displayErrors.map((error) => this.renderErrorItem(error)).join('')}
                        </ul>
                        ${hasMore ? `
                            <p class="sync-error-more">
                                ... and ${formatNumber(errors.length - this.config.maxErrorsDisplayed)} more errors
                            </p>
                        ` : ''}
                    </div>
                ` : ''}
            </div>
        `;
    }

    /**
     * Render a single error item.
     */
    private renderErrorItem(error: SyncError): string {
        const time = new Date(error.timestamp).toLocaleTimeString();
        const recordInfo = error.record_id ? ` (Record ${error.record_id})` : '';

        return `
            <li class="sync-error-item ${error.recoverable ? 'sync-error-recoverable' : 'sync-error-fatal'}">
                <span class="sync-error-time">${time}</span>
                <span class="sync-error-message">${this.escapeHtml(error.message)}${recordInfo}</span>
            </li>
        `;
    }

    // ========================================================================
    // Helpers
    // ========================================================================

    /**
     * Get CSS class for status.
     */
    private getStatusClass(status: SyncStatus): string {
        const classes: Record<SyncStatus, string> = {
            idle: 'sync-status-idle',
            running: 'sync-status-running',
            completed: 'sync-status-completed',
            error: 'sync-status-error',
            cancelled: 'sync-status-cancelled',
        };
        return classes[status] || 'sync-status-idle';
    }

    /**
     * Get status icon SVG.
     */
    private getStatusIcon(status: SyncStatus): string {
        switch (status) {
            case 'running':
                return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="sync-icon-spinning">
                    <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
                </svg>`;
            case 'completed':
                return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
                    <polyline points="22 4 12 14.01 9 11.01"/>
                </svg>`;
            case 'error':
                return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <line x1="15" y1="9" x2="9" y2="15"/>
                    <line x1="9" y1="9" x2="15" y2="15"/>
                </svg>`;
            case 'cancelled':
                return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <line x1="8" y1="12" x2="16" y2="12"/>
                </svg>`;
            default:
                return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                </svg>`;
        }
    }

    /**
     * Get expand/collapse icon.
     */
    private getExpandIcon(expanded: boolean): string {
        if (expanded) {
            return `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="6 9 12 15 18 9"/>
            </svg>`;
        }
        return `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="9 18 15 12 9 6"/>
        </svg>`;
    }

    /**
     * Escape HTML to prevent XSS.
     */
    private escapeHtml(text: string): string {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

export default SyncProgressComponent;
