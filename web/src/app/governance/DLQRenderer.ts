// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * DLQRenderer - Failed Events (Dead Letter Queue) Page
 *
 * Dead letter queue management with retry and delete actions.
 */

import type { API } from '../../lib/api';
import type { DLQEntry, DLQStats, DLQEntryFilter } from '../../lib/types';
import { BaseRenderer, GovernanceConfig } from './BaseRenderer';
import { createLogger } from '../../lib/logger';

const logger = createLogger('DLQRenderer');

export class DLQRenderer extends BaseRenderer {
    private entries: DLQEntry[] = [];
    private stats: DLQStats | null = null;
    private filter: DLQEntryFilter = { limit: 50 };
    private page = 0;

    constructor(api: API, config: GovernanceConfig) {
        super(api, config);
    }

    // =========================================================================
    // Data Getters
    // =========================================================================

    getStats(): DLQStats | null {
        return this.stats;
    }

    getEntries(): DLQEntry[] {
        return this.entries;
    }

    // =========================================================================
    // Public API
    // =========================================================================

    setupEventListeners(): void {
        document.getElementById('failed-refresh-btn')?.addEventListener('click', () => {
            this.load();
        });

        document.getElementById('failed-retry-all-btn')?.addEventListener('click', () => {
            this.retryAll();
        });
    }

    async load(): Promise<void> {
        try {
            const [statsResult, entriesResult] = await Promise.all([
                this.api.getDLQStats().catch(() => null),
                this.api.getDLQEntries({
                    ...this.filter,
                    offset: this.page * (this.filter.limit || 50),
                }).catch(() => ({ entries: [], total: 0, limit: 50, offset: 0 })),
            ]);

            this.stats = statsResult;
            this.entries = entriesResult.entries;

            // Store total for pagination
            const totalEl = document.getElementById('failed-total-count');
            if (totalEl) {
                totalEl.setAttribute('data-total', String(entriesResult.total));
            }

            this.updateDisplay();
        } catch (error) {
            logger.error('Failed to load DLQ data', { error });
        }
    }

    // =========================================================================
    // Display Updates
    // =========================================================================

    private updateDisplay(): void {
        // Update stats
        if (this.stats) {
            this.setElementText('failed-total', this.stats.total_entries.toString());
            this.setElementText('failed-pending', (this.stats.entries_by_status?.pending || 0).toString());
            this.setElementText('failed-retried', this.stats.total_retries.toString());
            this.setElementText('failed-permanent', (this.stats.entries_by_status?.permanent || 0).toString());
        } else {
            this.setElementText('failed-total', this.entries.length.toString());
            this.setElementText('failed-pending', '0');
            this.setElementText('failed-retried', '0');
            this.setElementText('failed-permanent', '0');
        }

        this.renderTable();
    }

    private renderTable(): void {
        const tbody = document.getElementById('failed-events-tbody');
        if (!tbody) return;

        if (this.entries.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="table-empty">No failed events in queue</td></tr>';
            return;
        }

        const statusColors: Record<string, string> = {
            pending: '#f59e0b',
            retrying: '#3b82f6',
            permanent: '#ef4444',
        };

        tbody.innerHTML = this.entries.map(entry => {
            const statusColor = statusColors[entry.status] || '#666';
            const isRetryable = entry.status !== 'permanent';

            return `
                <tr data-event-id="${entry.event_id}" class="dlq-row dlq-${entry.status}">
                    <td class="dlq-time">${this.formatTimestamp(entry.first_failure)}</td>
                    <td class="dlq-source">${this.escapeHtml(entry.source)}</td>
                    <td class="dlq-info">
                        ${entry.username ? `<span class="dlq-user">${this.escapeHtml(entry.username)}</span>` : ''}
                        ${entry.media_title ? `<span class="dlq-media">${this.escapeHtml(entry.media_title)}</span>` : ''}
                    </td>
                    <td class="dlq-error" title="${this.escapeHtml(entry.last_error)}">
                        <span class="category-badge category-${entry.category}">${entry.category}</span>
                    </td>
                    <td class="dlq-retries">${entry.retry_count}/${entry.max_retries}</td>
                    <td class="dlq-status">
                        <span class="status-badge" style="background: ${statusColor}">${entry.status}</span>
                    </td>
                    <td class="actions-cell">
                        ${isRetryable ? `
                            <button class="btn-action btn-retry" data-action="retry" data-id="${entry.event_id}" title="Retry">
                                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <polyline points="23 4 23 10 17 10"/>
                                    <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
                                </svg>
                            </button>
                        ` : ''}
                        <button class="btn-action btn-delete" data-action="delete" data-id="${entry.event_id}" title="Remove">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M3 6h18"/>
                                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/>
                                <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                            </svg>
                        </button>
                    </td>
                </tr>
            `;
        }).join('');

        // Add click handlers for actions
        tbody.querySelectorAll('[data-action="retry"]').forEach(btn => {
            btn.addEventListener('click', async (e) => {
                const id = (e.currentTarget as HTMLElement).getAttribute('data-id');
                if (id) {
                    await this.retryEntry(id);
                }
            });
        });

        tbody.querySelectorAll('[data-action="delete"]').forEach(btn => {
            btn.addEventListener('click', async (e) => {
                const id = (e.currentTarget as HTMLElement).getAttribute('data-id');
                if (id) {
                    await this.deleteEntry(id);
                }
            });
        });
    }

    // =========================================================================
    // Actions
    // =========================================================================

    private async retryEntry(eventId: string): Promise<void> {
        try {
            const result = await this.api.retryDLQEntry(eventId);
            if (result.success) {
                logger.debug('Entry queued for retry', { eventId });
                await this.load();
            }
        } catch (error) {
            logger.error('Failed to retry entry', { error, eventId });
        }
    }

    private async deleteEntry(eventId: string): Promise<void> {
        try {
            await this.api.deleteDLQEntry(eventId);
            logger.debug('Entry deleted', { eventId });
            await this.load();
        } catch (error) {
            logger.error('Failed to delete entry', { error, eventId });
        }
    }

    private async retryAll(): Promise<void> {
        try {
            const result = await this.api.retryAllDLQEntries();
            logger.debug('Retry all result', { result });
            if (result.success) {
                await this.load();
            }
        } catch (error) {
            logger.error('Failed to retry all entries', { error });
        }
    }
}
