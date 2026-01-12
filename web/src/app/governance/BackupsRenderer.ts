// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * BackupsRenderer - Backups Management Page
 *
 * Backup creation, validation, restoration, retention policy management,
 * and automatic backup scheduling.
 */

import type { API } from '../../lib/api';
import type { Backup, BackupStats, RetentionPolicy, ScheduleConfig } from '../../lib/types';
import { BaseRenderer, GovernanceConfig } from './BaseRenderer';
import { createLogger } from '../../lib/logger';

const logger = createLogger('BackupsRenderer');

export class BackupsRenderer extends BaseRenderer {
    private backups: Backup[] = [];
    private stats: BackupStats | null = null;
    private retentionPolicy: RetentionPolicy | null = null;
    private scheduleConfig: ScheduleConfig | null = null;

    constructor(api: API, config: GovernanceConfig) {
        super(api, config);
    }

    // =========================================================================
    // Data Getters
    // =========================================================================

    getStats(): BackupStats | null {
        return this.stats;
    }

    // =========================================================================
    // Public API
    // =========================================================================

    setupEventListeners(): void {
        document.getElementById('backups-refresh-btn')?.addEventListener('click', () => {
            this.load();
        });

        document.getElementById('backups-create-btn')?.addEventListener('click', () => {
            this.openCreateDialog();
        });

        document.getElementById('retention-preview-btn')?.addEventListener('click', () => {
            this.previewRetention();
        });

        document.getElementById('retention-apply-btn')?.addEventListener('click', () => {
            this.applyRetention();
        });

        // Schedule event listeners
        document.getElementById('schedule-save-btn')?.addEventListener('click', () => {
            this.saveScheduleConfig();
        });

        document.getElementById('schedule-trigger-btn')?.addEventListener('click', () => {
            this.triggerBackup();
        });

        document.getElementById('schedule-enabled')?.addEventListener('change', () => {
            this.updateScheduleFieldsState();
        });
    }

    async load(): Promise<void> {
        try {
            const [backups, stats, retention, schedule] = await Promise.all([
                this.api.listBackups().catch(() => []),
                this.api.getBackupStats().catch(() => null),
                this.api.getRetentionPolicy().catch(() => null),
                this.api.getScheduleConfig().catch(() => null),
            ]);

            this.backups = backups;
            this.stats = stats;
            this.retentionPolicy = retention;
            this.scheduleConfig = schedule;

            this.updateDisplay();
        } catch (error) {
            logger.error('Failed to load backups data', { error });
        }
    }

    // =========================================================================
    // Display Updates
    // =========================================================================

    private updateDisplay(): void {
        // Update stats
        if (this.stats) {
            this.setElementText('backup-total-count', this.stats.total_backups.toString());
            this.setElementText('backup-total-size', this.formatBytes(this.stats.total_size_bytes));
            this.setElementText('backup-full-count', this.stats.full_backups.toString());
            this.setElementText('backup-db-count', this.stats.database_backups.toString());
            this.setElementText('backup-config-count', this.stats.config_backups.toString());
            this.setElementText('backup-last-backup', this.stats.newest_backup
                ? this.formatTimeAgo(new Date(this.stats.newest_backup))
                : 'Never');
        }

        // Update retention policy display
        if (this.retentionPolicy) {
            this.setElementText('retention-min-count', this.retentionPolicy.min_count.toString());
            this.setElementText('retention-max-count', this.retentionPolicy.max_count === 0 ? 'Unlimited' : this.retentionPolicy.max_count.toString());
            this.setElementText('retention-max-age', this.retentionPolicy.max_age_days === 0 ? 'Unlimited' : `${this.retentionPolicy.max_age_days} days`);
            this.setElementText('retention-daily', `${this.retentionPolicy.keep_daily_for_days} days`);
            this.setElementText('retention-weekly', `${this.retentionPolicy.keep_weekly_for_weeks} weeks`);
            this.setElementText('retention-monthly', `${this.retentionPolicy.keep_monthly_for_months} months`);
        }

        // Update schedule display
        this.updateScheduleDisplay();

        this.renderTable();
    }

    private updateScheduleDisplay(): void {
        if (!this.scheduleConfig) return;

        // Set enabled checkbox
        const enabledCheckbox = document.getElementById('schedule-enabled') as HTMLInputElement | null;
        if (enabledCheckbox) {
            enabledCheckbox.checked = this.scheduleConfig.enabled;
        }

        // Set interval
        const intervalSelect = document.getElementById('schedule-interval') as HTMLSelectElement | null;
        if (intervalSelect) {
            intervalSelect.value = this.scheduleConfig.interval_hours.toString();
        }

        // Set preferred hour
        const hourSelect = document.getElementById('schedule-preferred-hour') as HTMLSelectElement | null;
        if (hourSelect) {
            hourSelect.value = this.scheduleConfig.preferred_hour.toString();
        }

        // Set backup type
        const typeSelect = document.getElementById('schedule-backup-type') as HTMLSelectElement | null;
        if (typeSelect) {
            typeSelect.value = this.scheduleConfig.backup_type;
        }

        // Set pre-sync checkbox
        const preSyncCheckbox = document.getElementById('schedule-pre-sync') as HTMLInputElement | null;
        if (preSyncCheckbox) {
            preSyncCheckbox.checked = this.scheduleConfig.pre_sync_backup;
        }

        // Update next backup display
        this.updateNextBackupDisplay();

        // Update form fields state
        this.updateScheduleFieldsState();
    }

    private updateNextBackupDisplay(): void {
        const nextValue = document.getElementById('schedule-next-value');
        if (!nextValue) return;

        if (this.scheduleConfig?.enabled) {
            // Calculate approximate next backup time based on interval and preferred hour
            const nextBackup = this.calculateNextBackupTime(
                this.scheduleConfig.interval_hours,
                this.scheduleConfig.preferred_hour
            );
            nextValue.textContent = this.formatFutureTime(nextBackup);
        } else {
            nextValue.textContent = 'Disabled';
        }
    }

    private formatFutureTime(date: Date): string {
        const now = new Date();
        const diffMs = date.getTime() - now.getTime();
        const diffMins = Math.floor(diffMs / 60000);
        const diffHours = Math.floor(diffMins / 60);
        const diffDays = Math.floor(diffHours / 24);

        if (diffMins < 60) {
            return `in ${diffMins} min`;
        }
        if (diffHours < 24) {
            return `in ${diffHours}h`;
        }
        if (diffDays === 1) {
            return 'tomorrow';
        }
        return `in ${diffDays} days`;
    }

    private calculateNextBackupTime(intervalHours: number, preferredHour: number): Date {
        const now = new Date();

        if (intervalHours >= 24) {
            // Daily or longer - use preferred hour
            const next = new Date(now);
            next.setHours(preferredHour, 0, 0, 0);

            if (next <= now) {
                next.setDate(next.getDate() + 1);
            }

            // Add additional days if interval is more than 24h
            if (intervalHours > 24) {
                const additionalDays = Math.floor(intervalHours / 24) - 1;
                next.setDate(next.getDate() + additionalDays);
            }

            return next;
        }

        // Shorter interval - add to current time
        const next = new Date(now.getTime() + intervalHours * 60 * 60 * 1000);
        return next;
    }

    private updateScheduleFieldsState(): void {
        const enabledCheckbox = document.getElementById('schedule-enabled') as HTMLInputElement | null;
        const isEnabled = enabledCheckbox?.checked ?? false;

        const fields = [
            'schedule-interval',
            'schedule-preferred-hour',
            'schedule-backup-type',
            'schedule-pre-sync'
        ];

        fields.forEach(id => {
            const element = document.getElementById(id) as HTMLInputElement | HTMLSelectElement | null;
            if (element) {
                element.disabled = !isEnabled;
            }
        });
    }

    private renderTable(): void {
        const tbody = document.getElementById('backup-list-tbody');
        if (!tbody) return;

        if (this.backups.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" class="table-empty">No backups found</td></tr>';
            return;
        }

        tbody.innerHTML = this.backups.map(backup => {
            const typeClass = `backup-type-${backup.type}`;
            const validIcon = backup.is_valid !== false ? '\u2714' : '\u2718';
            const validClass = backup.is_valid !== false ? 'valid' : 'invalid';

            return `
                <tr data-backup-id="${backup.id}">
                    <td>${this.formatTimestamp(backup.created_at)}</td>
                    <td><span class="backup-type-badge ${typeClass}">${backup.type}</span></td>
                    <td class="backup-filename">${this.escapeHtml(backup.filename)}</td>
                    <td>${this.formatBytes(backup.size_bytes)}</td>
                    <td>${backup.database_records?.toLocaleString() || '--'}</td>
                    <td><span class="backup-valid ${validClass}">${validIcon}</span></td>
                    <td class="backup-notes">${backup.notes ? this.escapeHtml(backup.notes) : '--'}</td>
                    <td class="actions-cell">
                        <button class="btn-action btn-download" data-action="download" data-id="${backup.id}" title="Download">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                                <polyline points="7 10 12 15 17 10"/>
                                <line x1="12" y1="15" x2="12" y2="3"/>
                            </svg>
                        </button>
                        <button class="btn-action btn-validate" data-action="validate" data-id="${backup.id}" title="Validate">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M9 12l2 2 4-4"/>
                                <path d="M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0z"/>
                            </svg>
                        </button>
                        <button class="btn-action btn-restore" data-action="restore" data-id="${backup.id}" title="Restore">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
                                <path d="M3 3v5h5"/>
                            </svg>
                        </button>
                        <button class="btn-action btn-delete btn-danger" data-action="delete" data-id="${backup.id}" title="Delete">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <polyline points="3 6 5 6 21 6"/>
                                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                            </svg>
                        </button>
                    </td>
                </tr>
            `;
        }).join('');

        // Add action listeners
        tbody.querySelectorAll('[data-action]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const action = (e.currentTarget as HTMLElement).getAttribute('data-action');
                const id = (e.currentTarget as HTMLElement).getAttribute('data-id');
                if (!id) return;

                switch (action) {
                    case 'download':
                        this.downloadBackup(id);
                        break;
                    case 'validate':
                        this.validateBackup(id);
                        break;
                    case 'restore':
                        this.openRestoreDialog(id);
                        break;
                    case 'delete':
                        this.deleteBackup(id);
                        break;
                }
            });
        });
    }

    // =========================================================================
    // Actions
    // =========================================================================

    private downloadBackup(backupId: string): void {
        const url = this.api.getBackupDownloadUrl(backupId);
        window.open(url, '_blank');
    }

    private async validateBackup(backupId: string): Promise<void> {
        try {
            const result = await this.api.validateBackup(backupId);
            if (result.is_valid) {
                logger.debug('Backup is valid', { backupId });
            } else {
                logger.warn('Backup validation issues', { backupId, issues: result.issues });
            }
            await this.load();
        } catch (error) {
            logger.error('Failed to validate backup', { error, backupId });
        }
    }

    private async deleteBackup(backupId: string): Promise<void> {
        if (!confirm('Are you sure you want to delete this backup? This action cannot be undone.')) {
            return;
        }

        try {
            await this.api.deleteBackup(backupId);
            await this.load();
        } catch (error) {
            logger.error('Failed to delete backup', { error, backupId });
        }
    }

    private openCreateDialog(): void {
        const dialog = document.getElementById('create-backup-dialog');
        if (dialog) {
            dialog.style.display = 'flex';
            dialog.setAttribute('aria-hidden', 'false');
        }
    }

    private openRestoreDialog(backupId: string): void {
        const dialog = document.getElementById('restore-backup-dialog');
        if (dialog) {
            dialog.setAttribute('data-backup-id', backupId);
            dialog.style.display = 'flex';
            dialog.setAttribute('aria-hidden', 'false');
        }
    }

    private async previewRetention(): Promise<void> {
        try {
            const preview = await this.api.getRetentionPreview();
            const message = `Retention preview:\n- Would delete: ${preview.deleted_count} backups\n- Would keep: ${preview.kept_count} backups\n- Space freed: ${this.formatBytes(preview.total_deleted_size)}`;
            alert(message);
        } catch (error) {
            logger.error('Failed to preview retention', { error });
        }
    }

    private async applyRetention(): Promise<void> {
        if (!confirm('This will permanently delete backups according to the retention policy. Continue?')) {
            return;
        }

        try {
            const result = await this.api.applyRetention();
            logger.debug('Retention applied', {
                deletedCount: result.deleted_count,
                freedBytes: result.freed_bytes
            });
            await this.load();
        } catch (error) {
            logger.error('Failed to apply retention', { error });
        }
    }

    // =========================================================================
    // Schedule Actions
    // =========================================================================

    private async saveScheduleConfig(): Promise<void> {
        const enabledCheckbox = document.getElementById('schedule-enabled') as HTMLInputElement | null;
        const intervalSelect = document.getElementById('schedule-interval') as HTMLSelectElement | null;
        const hourSelect = document.getElementById('schedule-preferred-hour') as HTMLSelectElement | null;
        const typeSelect = document.getElementById('schedule-backup-type') as HTMLSelectElement | null;
        const preSyncCheckbox = document.getElementById('schedule-pre-sync') as HTMLInputElement | null;

        const config = {
            enabled: enabledCheckbox?.checked ?? false,
            interval_hours: parseInt(intervalSelect?.value ?? '24', 10),
            preferred_hour: parseInt(hourSelect?.value ?? '2', 10),
            backup_type: (typeSelect?.value ?? 'full') as 'full' | 'database' | 'config',
            pre_sync_backup: preSyncCheckbox?.checked ?? false,
        };

        try {
            const saveBtn = document.getElementById('schedule-save-btn') as HTMLButtonElement | null;
            if (saveBtn) {
                saveBtn.disabled = true;
                saveBtn.textContent = 'Saving...';
            }

            await this.api.setScheduleConfig(config);
            this.scheduleConfig = { ...config, backup_type: config.backup_type };

            logger.info('Schedule config saved', { config });

            // Update display to reflect changes
            this.updateNextBackupDisplay();

            if (saveBtn) {
                saveBtn.textContent = 'Saved!';
                setTimeout(() => {
                    saveBtn.disabled = false;
                    saveBtn.textContent = 'Save Schedule';
                }, 2000);
            }
        } catch (error) {
            logger.error('Failed to save schedule config', { error });
            alert('Failed to save schedule configuration. Please try again.');

            const saveBtn = document.getElementById('schedule-save-btn') as HTMLButtonElement | null;
            if (saveBtn) {
                saveBtn.disabled = false;
                saveBtn.textContent = 'Save Schedule';
            }
        }
    }

    private async triggerBackup(): Promise<void> {
        const triggerBtn = document.getElementById('schedule-trigger-btn') as HTMLButtonElement | null;

        try {
            if (triggerBtn) {
                triggerBtn.disabled = true;
                triggerBtn.textContent = 'Running...';
            }

            const backup = await this.api.triggerScheduledBackup();
            logger.info('Backup triggered successfully', { backupId: backup.id });

            // Reload to show new backup
            await this.load();

            if (triggerBtn) {
                triggerBtn.textContent = 'Done!';
                setTimeout(() => {
                    triggerBtn.disabled = false;
                    triggerBtn.textContent = 'Run Backup Now';
                }, 2000);
            }
        } catch (error) {
            logger.error('Failed to trigger backup', { error });
            alert('Failed to trigger backup. Please try again.');

            if (triggerBtn) {
                triggerBtn.disabled = false;
                triggerBtn.textContent = 'Run Backup Now';
            }
        }
    }
}
