// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * SchedulesRenderer - Newsletter Schedule Management
 *
 * CRUD operations for newsletter schedules with cron configuration.
 */

import { createLogger } from '../../lib/logger';
import type { API } from '../../lib/api';
import type {
    NewsletterSchedule,
    NewsletterTemplate,
    DeliveryChannel,
    CreateScheduleRequest,
    UpdateScheduleRequest,
} from '../../lib/types/newsletter';
import { CHANNEL_CONFIG } from '../../lib/types/newsletter';
import { BaseNewsletterRenderer, NewsletterConfig } from './BaseNewsletterRenderer';

const logger = createLogger('NewsletterSchedulesRenderer');

/** Common cron presets */
const CRON_PRESETS = [
    { label: 'Daily at 9 AM', value: '0 9 * * *' },
    { label: 'Weekly on Monday at 9 AM', value: '0 9 * * 1' },
    { label: 'Weekly on Sunday at 9 AM', value: '0 9 * * 0' },
    { label: 'First of month at 9 AM', value: '0 9 1 * *' },
    { label: 'Every 6 hours', value: '0 */6 * * *' },
    { label: 'Custom', value: 'custom' },
];

/** Common timezones */
const TIMEZONES = [
    'UTC',
    'America/New_York',
    'America/Chicago',
    'America/Denver',
    'America/Los_Angeles',
    'Europe/London',
    'Europe/Paris',
    'Europe/Berlin',
    'Asia/Tokyo',
    'Asia/Shanghai',
    'Australia/Sydney',
];

export class SchedulesRenderer extends BaseNewsletterRenderer {
    private schedules: NewsletterSchedule[] = [];
    private templates: NewsletterTemplate[] = [];

    constructor(api: API, config: NewsletterConfig) {
        super(api, config);
    }

    // =========================================================================
    // Public API
    // =========================================================================

    setupEventListeners(): void {
        // Refresh button
        document.getElementById('newsletter-schedules-refresh-btn')?.addEventListener('click', () => {
            this.load();
        });

        // Create button
        document.getElementById('newsletter-schedules-create-btn')?.addEventListener('click', () => {
            this.showCreateModal();
        });

        // Enabled filter
        document.getElementById('newsletter-schedules-enabled-filter')?.addEventListener('change', (e) => {
            const value = (e.target as HTMLSelectElement).value;
            this.filterByEnabled(value === '' ? undefined : value === 'true');
        });
    }

    async load(): Promise<void> {
        try {
            const [schedulesResult, templatesResult] = await Promise.all([
                this.api.getNewsletterSchedules({ limit: this.config.maxTableEntries }),
                this.api.getNewsletterTemplates({ active: true }),
            ]);

            this.schedules = schedulesResult.schedules || [];
            this.templates = templatesResult.templates || [];

            this.renderTable();
        } catch (error) {
            logger.error('Failed to load newsletter schedules:', error);
        }
    }

    // =========================================================================
    // Table Rendering
    // =========================================================================

    private renderTable(): void {
        const tbody = document.getElementById('newsletter-schedules-tbody');
        if (!tbody) return;

        if (this.schedules.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" class="table-empty">No schedules found. Create your first schedule to automate newsletter delivery.</td></tr>';
            return;
        }

        tbody.innerHTML = this.schedules.map(schedule => `
            <tr class="newsletter-schedule-row" data-id="${schedule.id}" data-enabled="${schedule.is_enabled}">
                <td class="schedule-name">
                    <div class="schedule-name-cell">
                        <span class="schedule-title">${this.escapeHtml(schedule.name)}</span>
                    </div>
                    ${schedule.description ? `<span class="schedule-description">${this.escapeHtml(schedule.description)}</span>` : ''}
                </td>
                <td class="schedule-template">${this.escapeHtml(schedule.template_name || '--')}</td>
                <td class="schedule-timing">
                    <span class="cron-display">${this.formatCronExpression(schedule.cron_expression)}</span>
                    <small class="timezone-display">${schedule.timezone}</small>
                </td>
                <td class="schedule-channels">${this.renderChannelBadges(schedule.channels)}</td>
                <td class="schedule-recipients">${schedule.recipients.length} recipients</td>
                <td class="schedule-status">
                    <label class="toggle-switch">
                        <input type="checkbox" ${schedule.is_enabled ? 'checked' : ''}
                               data-action="toggle-enabled" data-id="${schedule.id}">
                        <span class="toggle-slider"></span>
                    </label>
                </td>
                <td class="schedule-last-run">
                    ${schedule.last_run_at ? `
                        <span class="last-run-time">${this.formatTimeAgo(new Date(schedule.last_run_at))}</span>
                        ${this.renderStatusBadge(schedule.last_run_status!)}
                    ` : '<span class="never-run">Never</span>'}
                </td>
                <td class="schedule-actions">
                    <button class="btn-action btn-trigger" data-action="trigger" data-id="${schedule.id}"
                            title="Send Now" ${!schedule.is_enabled ? 'disabled' : ''}>
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polygon points="5 3 19 12 5 21 5 3"/>
                        </svg>
                    </button>
                    <button class="btn-action btn-edit" data-action="edit" data-id="${schedule.id}" title="Edit">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                            <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                        </svg>
                    </button>
                    <button class="btn-action btn-delete" data-action="delete" data-id="${schedule.id}" title="Delete">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="3 6 5 6 21 6"/>
                            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                        </svg>
                    </button>
                </td>
            </tr>
        `).join('');

        // Add action handlers
        this.attachTableActions();
    }

    private attachTableActions(): void {
        const tbody = document.getElementById('newsletter-schedules-tbody');
        if (!tbody) return;

        // Toggle enabled
        tbody.querySelectorAll('[data-action="toggle-enabled"]').forEach(input => {
            input.addEventListener('change', async (e) => {
                const id = (e.currentTarget as HTMLElement).getAttribute('data-id');
                const enabled = (e.currentTarget as HTMLInputElement).checked;
                if (!id) return;

                try {
                    if (enabled) {
                        await this.api.enableNewsletterSchedule(id);
                    } else {
                        await this.api.disableNewsletterSchedule(id);
                    }
                } catch (error) {
                    logger.error('Failed to toggle schedule:', error);
                    (e.currentTarget as HTMLInputElement).checked = !enabled;
                }
            });
        });

        // Other actions
        tbody.querySelectorAll('[data-action]:not([data-action="toggle-enabled"])').forEach(btn => {
            btn.addEventListener('click', async (e) => {
                const action = (e.currentTarget as HTMLElement).getAttribute('data-action');
                const id = (e.currentTarget as HTMLElement).getAttribute('data-id');
                if (!id) return;

                switch (action) {
                    case 'trigger':
                        await this.triggerSchedule(id);
                        break;
                    case 'edit':
                        await this.showEditModal(id);
                        break;
                    case 'delete':
                        await this.confirmDelete(id);
                        break;
                }
            });
        });
    }

    private filterByEnabled(enabled: boolean | undefined): void {
        const rows = document.querySelectorAll('.newsletter-schedule-row');
        rows.forEach(row => {
            if (enabled === undefined) {
                (row as HTMLElement).style.display = '';
            } else {
                const isEnabled = row.getAttribute('data-enabled') === 'true';
                (row as HTMLElement).style.display = isEnabled === enabled ? '' : 'none';
            }
        });
    }

    // =========================================================================
    // Actions
    // =========================================================================

    private async triggerSchedule(id: string): Promise<void> {
        try {
            await this.api.triggerNewsletterSchedule(id);
            // Show success notification
            this.showNotification('Newsletter delivery triggered successfully');
            await this.load();
        } catch (error) {
            logger.error('Failed to trigger schedule:', error);
            this.showNotification('Failed to trigger newsletter delivery', 'error');
        }
    }

    private showNotification(message: string, type: 'success' | 'error' = 'success'): void {
        // Simple notification - could be enhanced with a proper notification system
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.textContent = message;
        document.body.appendChild(notification);
        setTimeout(() => notification.remove(), 3000);
    }

    // =========================================================================
    // Create Modal
    // =========================================================================

    private showCreateModal(): void {
        if (this.templates.length === 0) {
            this.showNotification('Please create a template first before creating a schedule', 'error');
            return;
        }

        const content = this.buildScheduleForm(null);
        const modal = this.createModal('newsletter-schedule-form-modal', 'Create Newsletter Schedule', content);

        this.showModal(modal);
        this.attachFormHandlers(modal, null);
    }

    // =========================================================================
    // Edit Modal
    // =========================================================================

    private async showEditModal(id: string): Promise<void> {
        try {
            const schedule = await this.api.getNewsletterSchedule(id);
            const content = this.buildScheduleForm(schedule);
            const modal = this.createModal('newsletter-schedule-form-modal', 'Edit Newsletter Schedule', content);

            this.showModal(modal);
            this.attachFormHandlers(modal, schedule);
        } catch (error) {
            logger.error('Failed to load schedule:', error);
        }
    }

    // =========================================================================
    // Delete Confirmation
    // =========================================================================

    private async confirmDelete(id: string): Promise<void> {
        const schedule = this.schedules.find(s => s.id === id);
        if (!schedule) return;

        const content = `
            <p>Are you sure you want to delete the schedule "<strong>${this.escapeHtml(schedule.name)}</strong>"?</p>
            <p class="warning-text">This action cannot be undone.</p>
            <div class="modal-actions">
                <button type="button" class="btn btn-secondary" data-action="cancel">Cancel</button>
                <button type="button" class="btn btn-danger" data-action="confirm">Delete</button>
            </div>
        `;

        const modal = this.createModal('newsletter-delete-confirm-modal', 'Delete Schedule', content);
        this.showModal(modal);

        modal.querySelector('[data-action="cancel"]')?.addEventListener('click', () => {
            this.closeModal('newsletter-delete-confirm-modal');
        });

        modal.querySelector('[data-action="confirm"]')?.addEventListener('click', async () => {
            try {
                await this.api.deleteNewsletterSchedule(id);
                this.closeModal('newsletter-delete-confirm-modal');
                await this.load();
            } catch (error) {
                logger.error('Failed to delete schedule:', error);
            }
        });
    }

    // =========================================================================
    // Form Building
    // =========================================================================

    private buildScheduleForm(schedule: NewsletterSchedule | null): string {
        const templateOptions = this.templates
            .map(t => `<option value="${t.id}" ${schedule?.template_id === t.id ? 'selected' : ''}>${this.escapeHtml(t.name)}</option>`)
            .join('');

        const cronPresetOptions = CRON_PRESETS
            .map(p => {
                const isCustom = schedule && !CRON_PRESETS.some(preset => preset.value === schedule.cron_expression);
                const selected = schedule?.cron_expression === p.value || (isCustom && p.value === 'custom');
                return `<option value="${p.value}" ${selected ? 'selected' : ''}>${p.label}</option>`;
            })
            .join('');

        const timezoneOptions = TIMEZONES
            .map(tz => `<option value="${tz}" ${schedule?.timezone === tz ? 'selected' : ''}>${tz}</option>`)
            .join('');

        const channelCheckboxes = Object.entries(CHANNEL_CONFIG)
            .map(([channel, config]) => `
                <label class="checkbox-label">
                    <input type="checkbox" name="channels" value="${channel}"
                           ${schedule?.channels.includes(channel as DeliveryChannel) ? 'checked' : ''}>
                    ${config.icon} ${config.label}
                </label>
            `)
            .join('');

        const isCustomCron = schedule && !CRON_PRESETS.some(p => p.value === schedule.cron_expression);

        return `
            <form id="newsletter-schedule-form" class="newsletter-form">
                <div class="form-row">
                    <div class="form-group">
                        <label for="schedule-name">Name *</label>
                        <input type="text" id="schedule-name" name="name" required
                               value="${schedule ? this.escapeHtml(schedule.name) : ''}"
                               placeholder="e.g., Weekly Digest">
                    </div>
                    <div class="form-group">
                        <label for="schedule-template">Template *</label>
                        <select id="schedule-template" name="template_id" required>
                            <option value="">Select template...</option>
                            ${templateOptions}
                        </select>
                    </div>
                </div>

                <div class="form-group">
                    <label for="schedule-description">Description</label>
                    <input type="text" id="schedule-description" name="description"
                           value="${schedule?.description ? this.escapeHtml(schedule.description) : ''}"
                           placeholder="Brief description of this schedule">
                </div>

                <div class="form-section">
                    <h4>Schedule Timing</h4>
                    <div class="form-row">
                        <div class="form-group">
                            <label for="schedule-cron-preset">Frequency</label>
                            <select id="schedule-cron-preset" name="cron_preset">
                                ${cronPresetOptions}
                            </select>
                        </div>
                        <div class="form-group">
                            <label for="schedule-timezone">Timezone</label>
                            <select id="schedule-timezone" name="timezone">
                                ${timezoneOptions}
                            </select>
                        </div>
                    </div>
                    <div class="form-group" id="custom-cron-group" style="display: ${isCustomCron ? 'block' : 'none'}">
                        <label for="schedule-cron">Custom Cron Expression</label>
                        <input type="text" id="schedule-cron" name="cron_expression"
                               value="${schedule?.cron_expression || ''}"
                               placeholder="0 9 * * 1">
                        <small class="form-hint">Format: minute hour day-of-month month day-of-week</small>
                    </div>
                </div>

                <div class="form-section">
                    <h4>Delivery Channels *</h4>
                    <div class="checkbox-group">
                        ${channelCheckboxes}
                    </div>
                </div>

                <div class="form-section">
                    <h4>Recipients</h4>
                    <div class="form-group">
                        <label for="schedule-recipients">Email Addresses / User IDs</label>
                        <textarea id="schedule-recipients" name="recipients" rows="4"
                                  placeholder="Enter one recipient per line">${schedule?.recipients.join('\n') || ''}</textarea>
                        <small class="form-hint">Enter email addresses or user IDs, one per line</small>
                    </div>
                </div>

                <div class="form-row">
                    <div class="form-group">
                        <label>
                            <input type="checkbox" id="schedule-enabled" name="is_enabled"
                                   ${schedule?.is_enabled !== false ? 'checked' : ''}>
                            Enabled
                        </label>
                    </div>
                </div>

                <div class="form-actions">
                    <button type="button" class="btn btn-secondary" data-action="cancel">Cancel</button>
                    <button type="submit" class="btn btn-primary">${schedule ? 'Update' : 'Create'} Schedule</button>
                </div>
            </form>
        `;
    }

    private attachFormHandlers(modal: HTMLElement, schedule: NewsletterSchedule | null): void {
        const form = modal.querySelector('#newsletter-schedule-form') as HTMLFormElement;
        if (!form) return;

        // Cron preset change
        const cronPreset = form.querySelector('#schedule-cron-preset') as HTMLSelectElement;
        const customCronGroup = form.querySelector('#custom-cron-group') as HTMLElement;
        const cronInput = form.querySelector('#schedule-cron') as HTMLInputElement;

        cronPreset?.addEventListener('change', () => {
            if (cronPreset.value === 'custom') {
                customCronGroup.style.display = 'block';
            } else {
                customCronGroup.style.display = 'none';
                cronInput.value = cronPreset.value;
            }
        });

        // Cancel
        form.querySelector('[data-action="cancel"]')?.addEventListener('click', () => {
            this.closeModal('newsletter-schedule-form-modal');
        });

        // Submit
        form.addEventListener('submit', async (e) => {
            e.preventDefault();

            const formData = new FormData(form);
            const channels = formData.getAll('channels') as DeliveryChannel[];
            const recipientsText = formData.get('recipients') as string;
            const recipients = recipientsText.split('\n').map(r => r.trim()).filter(r => r);

            if (channels.length === 0) {
                this.showNotification('Please select at least one delivery channel', 'error');
                return;
            }

            const cronPresetValue = formData.get('cron_preset') as string;
            const cronExpression = cronPresetValue === 'custom'
                ? formData.get('cron_expression') as string
                : cronPresetValue;

            const data: CreateScheduleRequest | UpdateScheduleRequest = {
                name: formData.get('name') as string,
                description: formData.get('description') as string || undefined,
                template_id: formData.get('template_id') as string,
                cron_expression: cronExpression,
                timezone: formData.get('timezone') as string,
                channels,
                recipients,
                is_enabled: (form.querySelector('#schedule-enabled') as HTMLInputElement)?.checked ?? true,
            };

            try {
                if (schedule) {
                    await this.api.updateNewsletterSchedule(schedule.id, data);
                } else {
                    await this.api.createNewsletterSchedule(data as CreateScheduleRequest);
                }

                this.closeModal('newsletter-schedule-form-modal');
                await this.load();
            } catch (error) {
                logger.error('Failed to save schedule:', error);
                this.showNotification('Failed to save schedule', 'error');
            }
        });
    }
}
