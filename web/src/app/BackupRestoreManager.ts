// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * BackupRestoreManager - Manage database backups and restore
 *
 * Features:
 * - List existing backups
 * - Create new backups (full, database, config)
 * - Delete backups
 * - Download backups
 * - Restore from backup
 * - Display backup statistics
 */

import type { API, Backup, BackupStats, CreateBackupRequest } from '../lib/api';
import type { ToastManager } from '../lib/toast';
import type { ConfirmationDialogManager } from './ConfirmationDialogManager';
import { escapeHtml } from '../lib/sanitize';
import { createLogger } from '../lib/logger';
import { getRoleGuard } from '../lib/auth/RoleGuard';

const logger = createLogger('BackupRestoreManager');

export class BackupRestoreManager {
    private api: API;
    private toastManager: ToastManager | null = null;
    private confirmationManager: ConfirmationDialogManager | null = null;
    private backups: Backup[] = [];
    private stats: BackupStats | null = null;

    // DOM elements
    private container: HTMLElement | null = null;
    private backupList: HTMLElement | null = null;
    private statsContainer: HTMLElement | null = null;
    private createButton: HTMLButtonElement | null = null;
    private createDialog: HTMLElement | null = null;
    private restoreDialog: HTMLElement | null = null;

    // Event handler references for cleanup
    private createButtonClickHandler: (() => void) | null = null;
    private confirmButtonClickHandler: (() => void) | null = null;
    private cancelButtonClickHandler: (() => void) | null = null;
    private confirmRestoreClickHandler: (() => void) | null = null;
    private cancelRestoreClickHandler: (() => void) | null = null;
    private createDialogClickHandler: ((e: Event) => void) | null = null;
    private restoreDialogClickHandler: ((e: Event) => void) | null = null;
    private documentKeydownHandler: ((e: KeyboardEvent) => void) | null = null;
    private backupListClickHandler: ((e: Event) => void) | null = null;

    // Current backup being restored
    private pendingRestoreBackupId: string | null = null;

    constructor(api: API) {
        this.api = api;
    }

    /**
     * Initialize the backup manager
     * RBAC: Backup operations require admin role
     */
    init(containerId: string = 'backup-section'): void {
        this.container = document.getElementById(containerId);
        if (!this.container) {
            logger.error('Backup section container not found');
            return;
        }

        // RBAC Phase 4: Check admin permission
        const roleGuard = getRoleGuard();
        if (!roleGuard.canAccess('backup', 'read')) {
            logger.warn('[RBAC] User lacks permission to access backup functionality');
            this.container.innerHTML = `
                <div class="access-denied-message">
                    <p>You do not have permission to access backup management.</p>
                    <p>This feature requires administrator privileges.</p>
                </div>
            `;
            return;
        }

        this.bindElements();
        this.setupEventListeners();
        this.applyRoleBasedVisibility();
        this.loadData();
        logger.debug('BackupRestoreManager initialized');
    }

    /**
     * Apply role-based visibility to backup UI elements.
     * RBAC Phase 4: Frontend Role Integration
     */
    private applyRoleBasedVisibility(): void {
        const roleGuard = getRoleGuard();

        // Disable write operations for non-admins
        if (!roleGuard.canAccess('backup', 'write')) {
            // Disable create backup button
            if (this.createButton) {
                this.createButton.disabled = true;
                this.createButton.title = 'Admin role required to create backups';
            }
        }
    }

    /**
     * Set toast manager reference
     */
    setToastManager(toast: ToastManager): void {
        this.toastManager = toast;
    }

    /**
     * Set confirmation dialog manager reference
     */
    setConfirmationManager(confirmation: ConfirmationDialogManager): void {
        this.confirmationManager = confirmation;
    }

    /**
     * Bind to DOM elements
     */
    private bindElements(): void {
        this.backupList = document.getElementById('backup-list');
        this.statsContainer = document.getElementById('backup-stats');
        this.createButton = document.getElementById('btn-create-backup') as HTMLButtonElement;
        this.createDialog = document.getElementById('create-backup-dialog');
        this.restoreDialog = document.getElementById('restore-backup-dialog');
    }

    /**
     * Set up event listeners with stored references for cleanup
     */
    private setupEventListeners(): void {
        // Create backup button
        if (this.createButton) {
            this.createButtonClickHandler = () => this.openCreateDialog();
            this.createButton.addEventListener('click', this.createButtonClickHandler);
        }

        // Create dialog buttons
        const confirmButton = document.getElementById('btn-confirm-backup');
        if (confirmButton) {
            this.confirmButtonClickHandler = () => this.handleCreateBackup();
            confirmButton.addEventListener('click', this.confirmButtonClickHandler);
        }

        const cancelButton = document.getElementById('btn-cancel-backup');
        if (cancelButton) {
            this.cancelButtonClickHandler = () => this.closeCreateDialog();
            cancelButton.addEventListener('click', this.cancelButtonClickHandler);
        }

        // Restore dialog buttons
        const confirmRestoreButton = document.getElementById('btn-confirm-restore');
        if (confirmRestoreButton) {
            this.confirmRestoreClickHandler = () => this.handleRestoreBackup();
            confirmRestoreButton.addEventListener('click', this.confirmRestoreClickHandler);
        }

        const cancelRestoreButton = document.getElementById('btn-cancel-restore');
        if (cancelRestoreButton) {
            this.cancelRestoreClickHandler = () => this.closeRestoreDialog();
            cancelRestoreButton.addEventListener('click', this.cancelRestoreClickHandler);
        }

        // Dialog overlay clicks
        if (this.createDialog) {
            this.createDialogClickHandler = (e: Event) => {
                if (e.target === this.createDialog) {
                    this.closeCreateDialog();
                }
            };
            this.createDialog.addEventListener('click', this.createDialogClickHandler);
        }

        if (this.restoreDialog) {
            this.restoreDialogClickHandler = (e: Event) => {
                if (e.target === this.restoreDialog) {
                    this.closeRestoreDialog();
                }
            };
            this.restoreDialog.addEventListener('click', this.restoreDialogClickHandler);
        }

        // Escape key (document level - critical to clean up!)
        this.documentKeydownHandler = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                if (this.isCreateDialogOpen()) {
                    this.closeCreateDialog();
                }
                if (this.isRestoreDialogOpen()) {
                    this.closeRestoreDialog();
                }
            }
        };
        document.addEventListener('keydown', this.documentKeydownHandler);

        // Delegated click handler for backup list (prevents leak on re-render)
        if (this.backupList) {
            this.backupListClickHandler = (e: Event) => {
                const target = e.target as HTMLElement;
                const actionButton = target.closest('[data-action]') as HTMLElement;
                if (!actionButton) return;

                e.stopPropagation();
                const action = actionButton.getAttribute('data-action');
                const backupItem = actionButton.closest('.backup-item') as HTMLElement;
                const backupId = backupItem?.getAttribute('data-backup-id');

                if (!backupId) return;
                const backup = this.backups.find(b => b.id === backupId);
                if (!backup) return;

                switch (action) {
                    case 'download':
                        this.downloadBackup(backup.id);
                        break;
                    case 'delete':
                        this.confirmDelete(backup);
                        break;
                    case 'restore':
                        this.confirmRestore(backup);
                        break;
                }
            };
            this.backupList.addEventListener('click', this.backupListClickHandler);
        }
    }

    /**
     * Remove event listeners for cleanup
     */
    private removeEventListeners(): void {
        // Create button
        if (this.createButton && this.createButtonClickHandler) {
            this.createButton.removeEventListener('click', this.createButtonClickHandler);
        }

        // Create dialog buttons
        const confirmButton = document.getElementById('btn-confirm-backup');
        if (confirmButton && this.confirmButtonClickHandler) {
            confirmButton.removeEventListener('click', this.confirmButtonClickHandler);
        }

        const cancelButton = document.getElementById('btn-cancel-backup');
        if (cancelButton && this.cancelButtonClickHandler) {
            cancelButton.removeEventListener('click', this.cancelButtonClickHandler);
        }

        // Restore dialog buttons
        const confirmRestoreButton = document.getElementById('btn-confirm-restore');
        if (confirmRestoreButton && this.confirmRestoreClickHandler) {
            confirmRestoreButton.removeEventListener('click', this.confirmRestoreClickHandler);
        }

        const cancelRestoreButton = document.getElementById('btn-cancel-restore');
        if (cancelRestoreButton && this.cancelRestoreClickHandler) {
            cancelRestoreButton.removeEventListener('click', this.cancelRestoreClickHandler);
        }

        // Dialog overlays
        if (this.createDialog && this.createDialogClickHandler) {
            this.createDialog.removeEventListener('click', this.createDialogClickHandler);
        }

        if (this.restoreDialog && this.restoreDialogClickHandler) {
            this.restoreDialog.removeEventListener('click', this.restoreDialogClickHandler);
        }

        // Document keydown (critical!)
        if (this.documentKeydownHandler) {
            document.removeEventListener('keydown', this.documentKeydownHandler);
        }

        // Backup list
        if (this.backupList && this.backupListClickHandler) {
            this.backupList.removeEventListener('click', this.backupListClickHandler);
        }
    }

    /**
     * Load backup data from API
     */
    async loadData(): Promise<void> {
        try {
            const [backups, stats] = await Promise.all([
                this.api.listBackups().catch(() => []),
                this.api.getBackupStats().catch(() => null)
            ]);

            this.backups = backups;
            this.stats = stats;

            this.renderStats();
            this.renderBackupList();
        } catch (error) {
            logger.error('Failed to load backup data:', error);
            this.toastManager?.error('Failed to load backup data');
        }
    }

    /**
     * Render backup statistics
     */
    private renderStats(): void {
        if (!this.statsContainer || !this.stats) return;

        this.statsContainer.innerHTML = `
            <div class="backup-stat-item">
                <span class="backup-stat-value">${this.stats.total_backups}</span>
                <span class="backup-stat-label">Total Backups</span>
            </div>
            <div class="backup-stat-item">
                <span class="backup-stat-value">${this.formatBytes(this.stats.total_size_bytes)}</span>
                <span class="backup-stat-label">Total Size</span>
            </div>
            <div class="backup-stat-item">
                <span class="backup-stat-value">${this.stats.full_backups}</span>
                <span class="backup-stat-label">Full Backups</span>
            </div>
            <div class="backup-stat-item">
                <span class="backup-stat-value">${this.stats.database_backups}</span>
                <span class="backup-stat-label">DB Backups</span>
            </div>
        `;
    }

    /**
     * Render backup list
     * Note: Event listeners are handled via delegation in setupEventListeners()
     */
    private renderBackupList(): void {
        if (!this.backupList) return;

        if (this.backups.length === 0) {
            this.backupList.innerHTML = `
                <div class="backup-empty-state">
                    <p>No backups found</p>
                    <p class="backup-empty-hint">Create your first backup to protect your data.</p>
                </div>
            `;
            return;
        }

        // Render list - event handlers are delegated, no individual attachment needed
        this.backupList.innerHTML = this.backups.map(backup => this.renderBackupItem(backup)).join('');
    }

    /**
     * Render a single backup item
     */
    private renderBackupItem(backup: Backup): string {
        const date = new Date(backup.created_at);
        const typeLabel = backup.type.charAt(0).toUpperCase() + backup.type.slice(1);

        return `
            <div class="backup-item" data-backup-id="${backup.id}">
                <div class="backup-info">
                    <div class="backup-header">
                        <span class="backup-type backup-type-${backup.type}">${typeLabel}</span>
                        <span class="backup-date">${date.toLocaleDateString()} ${date.toLocaleTimeString()}</span>
                    </div>
                    <div class="backup-details">
                        <span class="backup-size">${this.formatBytes(backup.size_bytes)}</span>
                        ${backup.database_records ? `<span class="backup-records">${backup.database_records.toLocaleString()} records</span>` : ''}
                        ${backup.notes ? `<span class="backup-notes">${escapeHtml(backup.notes)}</span>` : ''}
                    </div>
                </div>
                <div class="backup-actions">
                    <button type="button" class="backup-action-btn backup-download-btn" data-action="download" aria-label="Download backup">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                            <polyline points="7 10 12 15 17 10"/>
                            <line x1="12" y1="15" x2="12" y2="3"/>
                        </svg>
                    </button>
                    <button type="button" class="backup-action-btn backup-restore-btn" data-action="restore" aria-label="Restore from backup">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="1 4 1 10 7 10"/>
                            <path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/>
                        </svg>
                    </button>
                    <button type="button" class="backup-action-btn backup-delete-btn" data-action="delete" aria-label="Delete backup">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="3 6 5 6 21 6"/>
                            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                        </svg>
                    </button>
                </div>
            </div>
        `;
    }

    /**
     * Open create backup dialog
     */
    private openCreateDialog(): void {
        if (!this.createDialog) return;

        this.createDialog.style.display = 'flex';
        this.createDialog.classList.add('visible');
        this.createDialog.setAttribute('aria-hidden', 'false');

        // Reset form
        const typeSelect = document.getElementById('backup-type-select') as HTMLSelectElement;
        const notesInput = document.getElementById('backup-notes-input') as HTMLInputElement;
        if (typeSelect) typeSelect.value = 'full';
        if (notesInput) notesInput.value = '';

        // Focus first input
        setTimeout(() => typeSelect?.focus(), 100);
    }

    /**
     * Close create backup dialog
     */
    private closeCreateDialog(): void {
        if (!this.createDialog) return;

        this.createDialog.classList.remove('visible');
        this.createDialog.setAttribute('aria-hidden', 'true');

        setTimeout(() => {
            if (this.createDialog) {
                this.createDialog.style.display = 'none';
            }
        }, 200);
    }

    /**
     * Check if create dialog is open
     */
    private isCreateDialogOpen(): boolean {
        return this.createDialog?.classList.contains('visible') ?? false;
    }

    /**
     * Check if restore dialog is open
     */
    private isRestoreDialogOpen(): boolean {
        return this.restoreDialog?.classList.contains('visible') ?? false;
    }

    /**
     * Open restore backup dialog
     */
    private openRestoreDialog(backup: Backup): void {
        if (!this.restoreDialog) return;

        this.pendingRestoreBackupId = backup.id;

        // Update dialog info
        const infoElement = document.getElementById('restore-backup-info');
        if (infoElement) {
            const date = new Date(backup.created_at).toLocaleDateString();
            infoElement.textContent = `Restore from backup created on ${date}. This will replace your current data.`;
        }

        // Reset checkboxes to default checked state
        const dbCheckbox = document.getElementById('restore-database-checkbox') as HTMLInputElement;
        const preBackupCheckbox = document.getElementById('restore-pre-backup-checkbox') as HTMLInputElement;
        const verifyCheckbox = document.getElementById('restore-verify-checkbox') as HTMLInputElement;
        if (dbCheckbox) dbCheckbox.checked = true;
        if (preBackupCheckbox) preBackupCheckbox.checked = true;
        if (verifyCheckbox) verifyCheckbox.checked = true;

        this.restoreDialog.style.display = 'flex';
        this.restoreDialog.classList.add('visible');
        this.restoreDialog.setAttribute('aria-hidden', 'false');

        // Focus first checkbox
        setTimeout(() => dbCheckbox?.focus(), 100);
    }

    /**
     * Close restore backup dialog
     */
    private closeRestoreDialog(): void {
        if (!this.restoreDialog) return;

        this.pendingRestoreBackupId = null;
        this.restoreDialog.classList.remove('visible');
        this.restoreDialog.setAttribute('aria-hidden', 'true');

        setTimeout(() => {
            if (this.restoreDialog) {
                this.restoreDialog.style.display = 'none';
            }
        }, 200);
    }

    /**
     * Handle restore backup from dialog
     */
    private async handleRestoreBackup(): Promise<void> {
        if (!this.pendingRestoreBackupId) return;

        // Get options from checkboxes
        const dbCheckbox = document.getElementById('restore-database-checkbox') as HTMLInputElement;
        const preBackupCheckbox = document.getElementById('restore-pre-backup-checkbox') as HTMLInputElement;
        const verifyCheckbox = document.getElementById('restore-verify-checkbox') as HTMLInputElement;

        const backupId = this.pendingRestoreBackupId;
        const options = {
            restore_database: dbCheckbox?.checked ?? true,
            create_pre_restore_backup: preBackupCheckbox?.checked ?? true,
            verify_after_restore: verifyCheckbox?.checked ?? true
        };

        this.closeRestoreDialog();
        await this.restoreBackup(backupId, options);
    }

    /**
     * Handle create backup
     */
    private async handleCreateBackup(): Promise<void> {
        const typeSelect = document.getElementById('backup-type-select') as HTMLSelectElement;
        const notesInput = document.getElementById('backup-notes-input') as HTMLInputElement;

        const request: CreateBackupRequest = {
            type: (typeSelect?.value as 'full' | 'database' | 'config') || 'full',
            notes: notesInput?.value || undefined
        };

        this.closeCreateDialog();

        try {
            this.toastManager?.info('Creating backup...', '', 3000);
            await this.api.createBackup(request);
            this.toastManager?.success('Backup created successfully');
            await this.loadData();
        } catch (error) {
            logger.error('Failed to create backup:', error);
            this.toastManager?.error('Failed to create backup');
        }
    }

    /**
     * Download a backup
     */
    private downloadBackup(backupId: string): void {
        const url = this.api.getBackupDownloadUrl(backupId);
        window.open(url, '_blank');
    }

    /**
     * Show delete confirmation
     */
    private async confirmDelete(backup: Backup): Promise<void> {
        const dateStr = new Date(backup.created_at).toLocaleDateString();

        if (this.confirmationManager) {
            const confirmed = await this.confirmationManager.show({
                title: 'Delete Backup',
                message: `Are you sure you want to delete this backup from ${dateStr}? This action cannot be undone.`,
                confirmText: 'Delete',
                confirmButtonClass: 'btn-danger'
            });
            if (confirmed) {
                await this.deleteBackup(backup.id);
            }
        } else {
            // Fallback to native confirm
            if (confirm(`Delete backup from ${dateStr}?`)) {
                await this.deleteBackup(backup.id);
            }
        }
    }

    /**
     * Delete a backup
     */
    private async deleteBackup(backupId: string): Promise<void> {
        try {
            await this.api.deleteBackup(backupId);
            this.toastManager?.success('Backup deleted');
            await this.loadData();
        } catch (error) {
            logger.error('Failed to delete backup:', error);
            this.toastManager?.error('Failed to delete backup');
        }
    }

    /**
     * Show restore confirmation dialog
     */
    private confirmRestore(backup: Backup): void {
        // Use dedicated restore dialog with options
        this.openRestoreDialog(backup);
    }

    /**
     * Restore from a backup
     */
    private async restoreBackup(backupId: string, options?: {
        restore_database?: boolean;
        create_pre_restore_backup?: boolean;
        verify_after_restore?: boolean;
    }): Promise<void> {
        try {
            this.toastManager?.info('Restoring backup...', '', 5000);
            const result = await this.api.restoreBackup(backupId, {
                create_pre_restore_backup: options?.create_pre_restore_backup ?? true,
                restore_database: options?.restore_database ?? true,
                verify_after_restore: options?.verify_after_restore ?? true
            });

            if (result.success) {
                this.toastManager?.success('Backup restored successfully. Please refresh the page.');
            } else {
                this.toastManager?.error('Restore completed with issues');
            }
        } catch (error) {
            logger.error('Failed to restore backup:', error);
            this.toastManager?.error('Failed to restore backup');
        }
    }

    /**
     * Format bytes to human readable string
     */
    private formatBytes(bytes: number): string {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    }

    /**
     * Refresh backup list
     */
    async refresh(): Promise<void> {
        await this.loadData();
    }

    /**
     * Clean up resources and event listeners
     */
    destroy(): void {
        this.removeEventListeners();
        this.createButtonClickHandler = null;
        this.confirmButtonClickHandler = null;
        this.cancelButtonClickHandler = null;
        this.confirmRestoreClickHandler = null;
        this.cancelRestoreClickHandler = null;
        this.createDialogClickHandler = null;
        this.restoreDialogClickHandler = null;
        this.documentKeydownHandler = null;
        this.backupListClickHandler = null;
        this.backups = [];
        this.stats = null;
    }
}

export default BackupRestoreManager;
