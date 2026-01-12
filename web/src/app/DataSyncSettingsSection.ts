// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * DataSyncSettingsSection - Data Sync UI for Settings Panel
 *
 * Integrates data synchronization controls into the Settings panel.
 * Supports:
 * - Tautulli database import (start, stop, resume, validate)
 * - Plex historical sync
 * - Per-server sync status monitoring
 *
 * Features:
 * - Real-time progress updates via WebSocket
 * - Persistent state across page refreshes
 * - RBAC enforcement (admin only)
 * - Accessibility compliant (WCAG 2.1 AA)
 */

import { createLogger } from '../lib/logger';
import { getRoleGuard } from '../lib/auth/RoleGuard';
import type { ToastManager } from '../lib/toast';
import type { SyncManager } from './SyncManager';
import { SyncProgressComponent } from './SyncProgressComponent';
import type {
    SyncProgress,
    SyncOperationType,
    TautulliValidateResponse,
} from '../lib/types/sync';
import { formatNumber, OPERATION_LABELS } from '../lib/types/sync';

const logger = createLogger('DataSyncSettingsSection');

// ============================================================================
// Configuration
// ============================================================================

interface DataSyncSectionConfig {
    /** ID of the container element */
    containerId?: string;
    /** Days back options for Plex historical sync */
    daysBackOptions?: number[];
    /** Default days back value */
    defaultDaysBack?: number;
}

const DEFAULT_CONFIG: Required<DataSyncSectionConfig> = {
    containerId: 'data-sync-section',
    daysBackOptions: [7, 14, 30, 60, 90, 180, 365],
    defaultDaysBack: 30,
};

// ============================================================================
// Component Class
// ============================================================================

/**
 * Data Sync Settings Section component.
 *
 * @example
 * ```typescript
 * const syncSection = new DataSyncSettingsSection(syncManager);
 * syncSection.setToastManager(toastManager);
 * syncSection.init();
 * ```
 */
export class DataSyncSettingsSection {
    private container: HTMLElement | null = null;
    private config: Required<DataSyncSectionConfig>;
    private toastManager: ToastManager | null = null;

    // Child components
    private tautulliProgressComponent: SyncProgressComponent | null = null;
    private plexProgressComponent: SyncProgressComponent | null = null;

    // State
    private isExpanded = false;
    private validationResult: TautulliValidateResponse | null = null;
    private isValidating = false;
    private isStarting = false;

    // Event handler references for cleanup
    private panelClickHandler: ((e: Event) => void) | null = null;
    private panelChangeHandler: ((e: Event) => void) | null = null;
    private panelSubmitHandler: ((e: Event) => void) | null = null;

    constructor(
        private syncManager: SyncManager,
        config: DataSyncSectionConfig = {}
    ) {
        this.config = { ...DEFAULT_CONFIG, ...config };
    }

    // ========================================================================
    // Lifecycle
    // ========================================================================

    /**
     * Set toast manager reference.
     */
    setToastManager(toast: ToastManager): void {
        this.toastManager = toast;
    }

    /**
     * Initialize the component.
     */
    init(): void {
        this.container = document.getElementById(this.config.containerId);
        if (!this.container) {
            logger.warn('Container not found, creating', { containerId: this.config.containerId });
            this.createContainer();
        }

        // Check RBAC - admin only
        const roleGuard = getRoleGuard();
        if (!roleGuard.canAccess('sync', 'admin')) {
            logger.warn('[RBAC] User lacks permission to access data sync');
            if (this.container) {
                this.container.innerHTML = `
                    <div class="settings-section data-sync-section">
                        <div class="access-denied-message">
                            <p>You do not have permission to access data synchronization.</p>
                        </div>
                    </div>
                `;
            }
            return;
        }

        this.render();
        this.setupEventListeners();
        this.initProgressComponents();

        // Set up callbacks on sync manager
        this.syncManager.setCallbacks({
            onStatusChange: (operation, progress) => this.handleStatusChange(operation, progress),
            onComplete: (operation, progress) => this.handleComplete(operation, progress),
            onError: (operation, error) => this.handleError(operation, error),
        });

        logger.debug('DataSyncSettingsSection initialized');
    }

    /**
     * Create container if it doesn't exist.
     */
    private createContainer(): void {
        const settingsContent = document.getElementById('settings-content') ||
            document.querySelector('.settings-sections');

        if (!settingsContent) {
            logger.error('Could not find settings content area');
            return;
        }

        const section = document.createElement('div');
        section.id = this.config.containerId;
        settingsContent.appendChild(section);
        this.container = section;
    }

    /**
     * Initialize child progress components.
     */
    private initProgressComponents(): void {
        // Initialize progress components after render
        setTimeout(() => {
            this.tautulliProgressComponent = new SyncProgressComponent('tautulli-import-progress', {
                showDetailedStats: true,
                showErrorLog: true,
                maxErrorsDisplayed: 5,
            });
            this.tautulliProgressComponent.init();
            this.tautulliProgressComponent.update(this.syncManager.getTautulliImportStatus());

            this.plexProgressComponent = new SyncProgressComponent('plex-historical-progress', {
                showDetailedStats: true,
                showErrorLog: true,
                maxErrorsDisplayed: 5,
            });
            this.plexProgressComponent.init();
            this.plexProgressComponent.update(this.syncManager.getPlexHistoricalStatus());
        }, 0);
    }

    /**
     * Clean up resources.
     */
    destroy(): void {
        this.removeEventListeners();
        this.tautulliProgressComponent?.destroy();
        this.plexProgressComponent?.destroy();
        this.container = null;
    }

    // ========================================================================
    // Event Handling
    // ========================================================================

    /**
     * Set up event listeners.
     */
    private setupEventListeners(): void {
        if (!this.container) return;

        // Delegated click handler
        this.panelClickHandler = (e: Event) => {
            const target = e.target as HTMLElement;
            this.handleClick(target);
        };
        this.container.addEventListener('click', this.panelClickHandler);

        // Delegated change handler
        this.panelChangeHandler = (e: Event) => {
            const target = e.target as HTMLInputElement;
            this.handleChange(target);
        };
        this.container.addEventListener('change', this.panelChangeHandler);

        // Form submit handler
        this.panelSubmitHandler = (e: Event) => {
            e.preventDefault();
        };
        this.container.addEventListener('submit', this.panelSubmitHandler);
    }

    /**
     * Remove event listeners.
     */
    private removeEventListeners(): void {
        if (!this.container) return;

        if (this.panelClickHandler) {
            this.container.removeEventListener('click', this.panelClickHandler);
        }
        if (this.panelChangeHandler) {
            this.container.removeEventListener('change', this.panelChangeHandler);
        }
        if (this.panelSubmitHandler) {
            this.container.removeEventListener('submit', this.panelSubmitHandler);
        }
    }

    /**
     * Handle click events via delegation.
     */
    private handleClick(target: HTMLElement): void {
        const button = target.closest('button');
        if (!button) return;

        const action = button.dataset.action;
        if (!action) return;

        switch (action) {
            case 'toggle-section':
                this.isExpanded = !this.isExpanded;
                this.render();
                this.setupEventListeners();
                break;
            case 'validate-tautulli':
                this.validateTautulliDatabase();
                break;
            case 'start-tautulli-import':
                this.startTautulliImport();
                break;
            case 'stop-tautulli-import':
                this.stopTautulliImport();
                break;
            case 'start-plex-historical':
                this.startPlexHistoricalSync();
                break;
        }
    }

    /**
     * Handle change events via delegation.
     */
    private handleChange(_target: HTMLInputElement): void {
        // Handle checkbox and select changes if needed
    }

    // ========================================================================
    // Sync Operations
    // ========================================================================

    /**
     * Validate Tautulli database.
     */
    private async validateTautulliDatabase(): Promise<void> {
        const dbPathInput = document.getElementById('tautulli-db-path') as HTMLInputElement;
        const dbPath = dbPathInput?.value?.trim();

        if (!dbPath) {
            this.toastManager?.error('Please enter a database path');
            return;
        }

        this.isValidating = true;
        this.updateValidateButton();

        try {
            this.validationResult = await this.syncManager.validateTautulliDatabase(dbPath);

            if (this.validationResult.success) {
                this.toastManager?.success('Database validation successful');
            } else {
                this.toastManager?.error(this.validationResult.error || 'Validation failed');
            }

            this.render();
            this.setupEventListeners();
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Validation failed';
            this.toastManager?.error(message);
            logger.error('Database validation failed', { error });
        } finally {
            this.isValidating = false;
            this.updateValidateButton();
        }
    }

    /**
     * Update validate button state.
     */
    private updateValidateButton(): void {
        const btn = document.getElementById('btn-validate-tautulli');
        if (btn) {
            btn.textContent = this.isValidating ? 'Validating...' : 'Validate Database';
            (btn as HTMLButtonElement).disabled = this.isValidating;
        }
    }

    /**
     * Start Tautulli import.
     */
    private async startTautulliImport(): Promise<void> {
        const dbPathInput = document.getElementById('tautulli-db-path') as HTMLInputElement;
        const resumeCheckbox = document.getElementById('tautulli-resume') as HTMLInputElement;
        const dryRunCheckbox = document.getElementById('tautulli-dry-run') as HTMLInputElement;

        const dbPath = dbPathInput?.value?.trim();
        const resume = resumeCheckbox?.checked ?? false;
        const dryRun = dryRunCheckbox?.checked ?? false;

        if (!dbPath) {
            this.toastManager?.error('Please enter a database path');
            return;
        }

        // Check if Plex historical sync is running
        if (!this.syncManager.isTautulliImportAvailable()) {
            this.toastManager?.error('Cannot start import while Plex historical sync is running');
            return;
        }

        this.isStarting = true;
        this.updateStartButton();

        try {
            await this.syncManager.startTautulliImport({
                db_path: dbPath,
                resume,
                dry_run: dryRun,
            });

            this.toastManager?.success('Import started');
            this.render();
            this.setupEventListeners();
            this.initProgressComponents();
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Failed to start import';
            this.toastManager?.error(message);
            logger.error('Failed to start Tautulli import', { error });
        } finally {
            this.isStarting = false;
            this.updateStartButton();
        }
    }

    /**
     * Update start button state.
     */
    private updateStartButton(): void {
        const btn = document.getElementById('btn-start-tautulli-import');
        if (btn) {
            btn.textContent = this.isStarting ? 'Starting...' : 'Start Import';
            (btn as HTMLButtonElement).disabled = this.isStarting;
        }
    }

    /**
     * Stop Tautulli import.
     */
    private async stopTautulliImport(): Promise<void> {
        try {
            await this.syncManager.stopTautulliImport();
            this.toastManager?.success('Import stopped');
            this.render();
            this.setupEventListeners();
            this.initProgressComponents();
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Failed to stop import';
            this.toastManager?.error(message);
            logger.error('Failed to stop Tautulli import', { error });
        }
    }

    /**
     * Start Plex historical sync.
     */
    private async startPlexHistoricalSync(): Promise<void> {
        const daysSelect = document.getElementById('plex-days-back') as HTMLSelectElement;
        const daysBack = parseInt(daysSelect?.value || String(this.config.defaultDaysBack), 10);

        // Check if Tautulli import is running
        if (!this.syncManager.isPlexHistoricalAvailable()) {
            this.toastManager?.error('Cannot start sync while Tautulli import is running');
            return;
        }

        try {
            await this.syncManager.startPlexHistoricalSync({
                days_back: daysBack,
            });

            this.toastManager?.success('Plex historical sync started');
            this.render();
            this.setupEventListeners();
            this.initProgressComponents();
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Failed to start sync';
            this.toastManager?.error(message);
            logger.error('Failed to start Plex historical sync', { error });
        }
    }

    // ========================================================================
    // Callbacks
    // ========================================================================

    /**
     * Handle status change from sync manager.
     */
    private handleStatusChange(operation: SyncOperationType, progress: SyncProgress): void {
        switch (operation) {
            case 'tautulli_import':
                this.tautulliProgressComponent?.update(progress);
                break;
            case 'plex_historical':
                this.plexProgressComponent?.update(progress);
                break;
        }
    }

    /**
     * Handle sync completion.
     */
    private handleComplete(operation: SyncOperationType, progress: SyncProgress): void {
        const label = OPERATION_LABELS[operation];

        if (progress.status === 'completed') {
            this.toastManager?.success(`${label} completed: ${formatNumber(progress.imported_records)} records imported`);
        } else if (progress.status === 'error') {
            this.toastManager?.error(`${label} failed with ${progress.error_count} errors`);
        }

        // Re-render to update button states
        this.render();
        this.setupEventListeners();
        this.initProgressComponents();
    }

    /**
     * Handle sync error.
     */
    private handleError(operation: SyncOperationType, error: Error): void {
        logger.error('Sync operation error', { operation, error: error.message });
    }

    // ========================================================================
    // Rendering
    // ========================================================================

    /**
     * Main render method.
     */
    private render(): void {
        if (!this.container) return;

        const tautulliStatus = this.syncManager.getTautulliImportStatus();
        const plexStatus = this.syncManager.getPlexHistoricalStatus();

        this.container.innerHTML = `
            <div class="settings-section data-sync-section">
                ${this.renderHeader()}
                ${this.isExpanded ? this.renderContent(tautulliStatus, plexStatus) : ''}
            </div>
        `;
    }

    /**
     * Render section header.
     */
    private renderHeader(): string {
        const tautulliStatus = this.syncManager.getTautulliImportStatus();
        const plexStatus = this.syncManager.getPlexHistoricalStatus();

        const summaryParts: string[] = [];
        if (tautulliStatus?.status === 'running') {
            summaryParts.push(`Tautulli: ${tautulliStatus.progress_percent.toFixed(0)}%`);
        } else if (tautulliStatus?.status === 'completed') {
            summaryParts.push('Tautulli: Complete');
        }
        if (plexStatus?.status === 'running') {
            summaryParts.push(`Plex: ${plexStatus.progress_percent.toFixed(0)}%`);
        }

        const summary = summaryParts.length > 0 ? summaryParts.join(' | ') : 'Ready';

        return `
            <div class="settings-section-header">
                <button type="button" class="settings-section-toggle" data-action="toggle-section"
                        aria-expanded="${this.isExpanded}" data-testid="toggle-data-sync">
                    <span class="settings-section-icon">${this.getExpandIcon(this.isExpanded)}</span>
                    <h4 class="settings-section-title">Data Sync</h4>
                    <span class="settings-section-summary">${summary}</span>
                </button>
            </div>
        `;
    }

    /**
     * Render section content.
     */
    private renderContent(tautulliStatus: SyncProgress | null, plexStatus: SyncProgress | null): string {
        return `
            <div class="settings-section-content" role="region" aria-label="Data sync settings">
                ${this.renderTautulliSection(tautulliStatus)}
                ${this.renderPlexHistoricalSection(plexStatus)}
            </div>
        `;
    }

    /**
     * Render Tautulli import section.
     */
    private renderTautulliSection(status: SyncProgress | null): string {
        const isRunning = status?.status === 'running';
        const canStart = !isRunning && this.syncManager.isTautulliImportAvailable();

        return `
            <div class="data-sync-subsection" data-testid="tautulli-import-section">
                <h5 class="data-sync-subsection-title">Tautulli Database Import</h5>
                <p class="data-sync-description">
                    Import playback history from your Tautulli SQLite database.
                </p>

                ${isRunning ? this.renderTautulliRunning() : this.renderTautulliForm(canStart)}

                <div id="tautulli-import-progress"></div>

                ${this.validationResult ? this.renderValidationResult() : ''}
            </div>
        `;
    }

    /**
     * Render Tautulli form when not running.
     */
    private renderTautulliForm(canStart: boolean): string {
        return `
            <form class="data-sync-form" data-testid="tautulli-import-form">
                <div class="form-group">
                    <label for="tautulli-db-path">Database Path</label>
                    <input type="text" id="tautulli-db-path" name="db_path"
                           placeholder="/config/tautulli.db"
                           class="form-input"
                           data-testid="tautulli-db-path">
                    <span class="form-hint">Path to the Tautulli SQLite database file on the server</span>
                </div>

                <div class="form-group-row">
                    <label class="checkbox-label">
                        <input type="checkbox" id="tautulli-resume" name="resume" data-testid="tautulli-resume">
                        <span>Resume from last position</span>
                    </label>
                    <label class="checkbox-label">
                        <input type="checkbox" id="tautulli-dry-run" name="dry_run" data-testid="tautulli-dry-run">
                        <span>Dry run (validate only)</span>
                    </label>
                </div>

                <div class="form-actions">
                    <button type="button" id="btn-validate-tautulli" class="btn-secondary"
                            data-action="validate-tautulli" data-testid="btn-validate-tautulli">
                        Validate Database
                    </button>
                    <button type="button" id="btn-start-tautulli-import" class="btn-primary"
                            data-action="start-tautulli-import"
                            ${!canStart ? 'disabled' : ''}
                            data-testid="btn-start-import">
                        Start Import
                    </button>
                </div>

                ${!canStart && !this.syncManager.isTautulliImportAvailable() ? `
                    <p class="form-warning">Cannot start import while Plex historical sync is running.</p>
                ` : ''}
            </form>
        `;
    }

    /**
     * Render Tautulli running state.
     */
    private renderTautulliRunning(): string {
        return `
            <div class="data-sync-running">
                <div class="form-actions">
                    <button type="button" class="btn-danger"
                            data-action="stop-tautulli-import"
                            data-testid="btn-stop-import">
                        Stop Import
                    </button>
                </div>
            </div>
        `;
    }

    /**
     * Render validation result.
     */
    private renderValidationResult(): string {
        const result = this.validationResult;
        if (!result) return '';

        if (!result.success) {
            return `
                <div class="validation-result validation-error" data-testid="validation-error">
                    <span class="validation-icon">${this.getErrorIcon()}</span>
                    <span class="validation-message">${this.escapeHtml(result.error || 'Validation failed')}</span>
                </div>
            `;
        }

        return `
            <div class="validation-result validation-success" data-testid="validation-success">
                <span class="validation-icon">${this.getSuccessIcon()}</span>
                <div class="validation-details">
                    <p><strong>Records:</strong> ${formatNumber(result.total_records || 0)}</p>
                    ${result.date_range ? `
                        <p><strong>Date Range:</strong> ${result.date_range.earliest} to ${result.date_range.latest}</p>
                    ` : ''}
                    ${result.unique_users ? `
                        <p><strong>Unique Users:</strong> ${formatNumber(result.unique_users)}</p>
                    ` : ''}
                    ${result.media_types ? `
                        <p><strong>Media Types:</strong> ${Object.entries(result.media_types)
                            .map(([type, count]) => `${type}: ${formatNumber(count)}`)
                            .join(', ')}</p>
                    ` : ''}
                </div>
            </div>
        `;
    }

    /**
     * Render Plex historical sync section.
     */
    private renderPlexHistoricalSection(status: SyncProgress | null): string {
        const isRunning = status?.status === 'running';
        const canStart = !isRunning && this.syncManager.isPlexHistoricalAvailable();

        return `
            <div class="data-sync-subsection" data-testid="plex-historical-section">
                <h5 class="data-sync-subsection-title">Plex Historical Sync</h5>
                <p class="data-sync-description">
                    Sync playback history directly from your Plex server.
                </p>

                ${!isRunning ? `
                    <form class="data-sync-form" data-testid="plex-historical-form">
                        <div class="form-group">
                            <label for="plex-days-back">Days to sync back</label>
                            <select id="plex-days-back" name="days_back" class="form-select" data-testid="plex-days-back">
                                ${this.config.daysBackOptions.map(days => `
                                    <option value="${days}" ${days === this.config.defaultDaysBack ? 'selected' : ''}>
                                        ${days} days
                                    </option>
                                `).join('')}
                            </select>
                        </div>

                        <div class="form-actions">
                            <button type="button" class="btn-primary"
                                    data-action="start-plex-historical"
                                    ${!canStart ? 'disabled' : ''}
                                    data-testid="btn-start-plex-historical">
                                Start Historical Sync
                            </button>
                        </div>

                        ${!canStart && !this.syncManager.isPlexHistoricalAvailable() ? `
                            <p class="form-warning">Cannot start sync while Tautulli import is running.</p>
                        ` : ''}
                    </form>
                ` : ''}

                <div id="plex-historical-progress"></div>
            </div>
        `;
    }

    // ========================================================================
    // Helpers
    // ========================================================================

    /**
     * Get expand/collapse icon.
     */
    private getExpandIcon(expanded: boolean): string {
        if (expanded) {
            return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="6 9 12 15 18 9"/>
            </svg>`;
        }
        return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="9 18 15 12 9 6"/>
        </svg>`;
    }

    /**
     * Get success icon.
     */
    private getSuccessIcon(): string {
        return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
            <polyline points="22 4 12 14.01 9 11.01"/>
        </svg>`;
    }

    /**
     * Get error icon.
     */
    private getErrorIcon(): string {
        return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="15" y1="9" x2="9" y2="15"/>
            <line x1="9" y1="9" x2="15" y2="15"/>
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

export default DataSyncSettingsSection;
