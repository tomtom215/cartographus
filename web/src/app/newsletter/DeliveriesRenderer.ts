// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * DeliveriesRenderer - Newsletter Delivery History
 *
 * View and filter newsletter delivery history and analytics.
 */

import { createLogger } from '../../lib/logger';
import type { API } from '../../lib/api';
import type {
    NewsletterDelivery,
    DeliveryStatus,
    DeliveryChannel,
} from '../../lib/types/newsletter';
import { BaseNewsletterRenderer, NewsletterConfig } from './BaseNewsletterRenderer';

const logger = createLogger('NewsletterDeliveriesRenderer');

export class DeliveriesRenderer extends BaseNewsletterRenderer {
    private deliveries: NewsletterDelivery[] = [];
    private filterStatus: DeliveryStatus | '' = '';
    private filterChannel: DeliveryChannel | '' = '';
    private page = 0;
    private hasMore = false;

    constructor(api: API, config: NewsletterConfig) {
        super(api, config);
    }

    // =========================================================================
    // Public API
    // =========================================================================

    setupEventListeners(): void {
        // Refresh button
        document.getElementById('newsletter-deliveries-refresh-btn')?.addEventListener('click', () => {
            this.page = 0;
            this.load();
        });

        // Status filter
        document.getElementById('newsletter-deliveries-status-filter')?.addEventListener('change', (e) => {
            this.filterStatus = (e.target as HTMLSelectElement).value as DeliveryStatus | '';
            this.page = 0;
            this.load();
        });

        // Channel filter
        document.getElementById('newsletter-deliveries-channel-filter')?.addEventListener('change', (e) => {
            this.filterChannel = (e.target as HTMLSelectElement).value as DeliveryChannel | '';
            this.page = 0;
            this.load();
        });

        // Pagination
        document.getElementById('newsletter-deliveries-prev-btn')?.addEventListener('click', () => {
            if (this.page > 0) {
                this.page--;
                this.load();
            }
        });

        document.getElementById('newsletter-deliveries-next-btn')?.addEventListener('click', () => {
            if (this.hasMore) {
                this.page++;
                this.load();
            }
        });
    }

    async load(): Promise<void> {
        try {
            const result = await this.api.getNewsletterDeliveries({
                status: this.filterStatus || undefined,
                channel: this.filterChannel || undefined,
                limit: this.config.maxTableEntries,
                offset: this.page * this.config.maxTableEntries,
            });

            this.deliveries = result.deliveries || [];
            this.hasMore = result.pagination?.has_more || false;

            this.renderTable();
            this.updatePagination();
        } catch (error) {
            logger.error('Failed to load newsletter deliveries:', error);
        }
    }

    // =========================================================================
    // Table Rendering
    // =========================================================================

    private renderTable(): void {
        const tbody = document.getElementById('newsletter-deliveries-tbody');
        if (!tbody) return;

        if (this.deliveries.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" class="table-empty">No delivery history found.</td></tr>';
            return;
        }

        tbody.innerHTML = this.deliveries.map(delivery => `
            <tr class="newsletter-delivery-row" data-id="${delivery.id}">
                <td class="delivery-time">
                    <span class="delivery-date">${this.formatTimestamp(delivery.started_at)}</span>
                </td>
                <td class="delivery-schedule">
                    <div class="delivery-info">
                        <span class="schedule-name">${this.escapeHtml(delivery.schedule_name || '--')}</span>
                        <span class="template-name">${this.escapeHtml(delivery.template_name || '--')}</span>
                    </div>
                </td>
                <td class="delivery-channel">${this.renderChannelBadge(delivery.channel)}</td>
                <td class="delivery-status">${this.renderStatusBadge(delivery.status)}</td>
                <td class="delivery-recipients">
                    <div class="recipients-stats">
                        <span class="recipients-count">${delivery.recipients_delivered}/${delivery.recipients_total}</span>
                        ${delivery.recipients_failed > 0 ? `
                            <span class="recipients-failed">${delivery.recipients_failed} failed</span>
                        ` : ''}
                    </div>
                    <div class="recipients-bar">
                        <div class="recipients-bar-success" style="width: ${this.formatPercent(delivery.recipients_delivered, delivery.recipients_total)}"></div>
                        ${delivery.recipients_failed > 0 ? `
                            <div class="recipients-bar-failed" style="width: ${this.formatPercent(delivery.recipients_failed, delivery.recipients_total)}"></div>
                        ` : ''}
                    </div>
                </td>
                <td class="delivery-duration">
                    ${delivery.duration_ms ? `${(delivery.duration_ms / 1000).toFixed(1)}s` : '--'}
                </td>
                <td class="delivery-content">
                    ${delivery.content_stats ? `
                        <span class="content-count">${delivery.content_stats.total_items} items</span>
                    ` : '--'}
                </td>
                <td class="delivery-actions">
                    <button class="btn-action btn-details" data-action="details" data-id="${delivery.id}" title="View Details">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <circle cx="12" cy="12" r="10"/>
                            <path d="M12 16v-4"/>
                            <path d="M12 8h.01"/>
                        </svg>
                    </button>
                </td>
            </tr>
        `).join('');

        // Add action handlers
        this.attachTableActions();
    }

    private attachTableActions(): void {
        const tbody = document.getElementById('newsletter-deliveries-tbody');
        if (!tbody) return;

        tbody.querySelectorAll('[data-action="details"]').forEach(btn => {
            btn.addEventListener('click', async (e) => {
                const id = (e.currentTarget as HTMLElement).getAttribute('data-id');
                if (id) {
                    await this.showDetailsModal(id);
                }
            });
        });
    }

    private updatePagination(): void {
        const pageInfo = document.getElementById('newsletter-deliveries-page-info');
        const prevBtn = document.getElementById('newsletter-deliveries-prev-btn') as HTMLButtonElement;
        const nextBtn = document.getElementById('newsletter-deliveries-next-btn') as HTMLButtonElement;

        if (pageInfo) {
            pageInfo.textContent = `Page ${this.page + 1}`;
        }

        if (prevBtn) {
            prevBtn.disabled = this.page === 0;
        }

        if (nextBtn) {
            nextBtn.disabled = !this.hasMore;
        }
    }

    // =========================================================================
    // Details Modal
    // =========================================================================

    private async showDetailsModal(id: string): Promise<void> {
        try {
            const delivery = await this.api.getNewsletterDelivery(id);

            const content = `
                <div class="delivery-details-container">
                    <div class="details-grid">
                        <div class="detail-row">
                            <label>Delivery ID:</label>
                            <span class="monospace">${delivery.id}</span>
                        </div>
                        <div class="detail-row">
                            <label>Started:</label>
                            <span>${this.formatTimestamp(delivery.started_at)}</span>
                        </div>
                        ${delivery.completed_at ? `
                            <div class="detail-row">
                                <label>Completed:</label>
                                <span>${this.formatTimestamp(delivery.completed_at)}</span>
                            </div>
                        ` : ''}
                        <div class="detail-row">
                            <label>Duration:</label>
                            <span>${delivery.duration_ms ? `${(delivery.duration_ms / 1000).toFixed(2)} seconds` : '--'}</span>
                        </div>
                        <div class="detail-row">
                            <label>Status:</label>
                            ${this.renderStatusBadge(delivery.status)}
                        </div>
                        <div class="detail-row">
                            <label>Channel:</label>
                            ${this.renderChannelBadge(delivery.channel)}
                        </div>
                    </div>

                    <div class="details-section">
                        <h4>Template</h4>
                        <div class="detail-row">
                            <label>Name:</label>
                            <span>${this.escapeHtml(delivery.template_name || '--')}</span>
                        </div>
                        <div class="detail-row">
                            <label>Version:</label>
                            <span>v${delivery.template_version}</span>
                        </div>
                        ${delivery.rendered_subject ? `
                            <div class="detail-row">
                                <label>Subject:</label>
                                <span>${this.escapeHtml(delivery.rendered_subject)}</span>
                            </div>
                        ` : ''}
                    </div>

                    <div class="details-section">
                        <h4>Recipients</h4>
                        <div class="recipients-summary">
                            <div class="summary-stat">
                                <span class="stat-value">${delivery.recipients_total}</span>
                                <span class="stat-label">Total</span>
                            </div>
                            <div class="summary-stat success">
                                <span class="stat-value">${delivery.recipients_delivered}</span>
                                <span class="stat-label">Delivered</span>
                            </div>
                            <div class="summary-stat error">
                                <span class="stat-value">${delivery.recipients_failed}</span>
                                <span class="stat-label">Failed</span>
                            </div>
                        </div>
                        ${delivery.recipient_details && delivery.recipient_details.length > 0 ? `
                            <table class="recipient-details-table">
                                <thead>
                                    <tr>
                                        <th>Recipient</th>
                                        <th>Status</th>
                                        <th>Time</th>
                                        <th>Error</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    ${delivery.recipient_details.slice(0, 20).map(r => `
                                        <tr>
                                            <td>${this.escapeHtml(r.recipient_id)}</td>
                                            <td>${this.renderStatusBadge(r.status)}</td>
                                            <td>${r.delivered_at ? this.formatTimestamp(r.delivered_at) : '--'}</td>
                                            <td class="error-cell">${r.error ? this.escapeHtml(r.error) : '--'}</td>
                                        </tr>
                                    `).join('')}
                                    ${delivery.recipient_details.length > 20 ? `
                                        <tr>
                                            <td colspan="4" class="more-recipients">
                                                ... and ${delivery.recipient_details.length - 20} more recipients
                                            </td>
                                        </tr>
                                    ` : ''}
                                </tbody>
                            </table>
                        ` : ''}
                    </div>

                    ${delivery.content_stats ? `
                        <div class="details-section">
                            <h4>Content</h4>
                            <div class="content-stats">
                                <div class="stat-item">
                                    <span class="stat-value">${delivery.content_stats.movies_count}</span>
                                    <span class="stat-label">Movies</span>
                                </div>
                                <div class="stat-item">
                                    <span class="stat-value">${delivery.content_stats.shows_count}</span>
                                    <span class="stat-label">Shows</span>
                                </div>
                                <div class="stat-item">
                                    <span class="stat-value">${delivery.content_stats.episodes_count}</span>
                                    <span class="stat-label">Episodes</span>
                                </div>
                            </div>
                        </div>
                    ` : ''}

                    ${delivery.error_message ? `
                        <div class="details-section error-section">
                            <h4>Error</h4>
                            <div class="error-message">${this.escapeHtml(delivery.error_message)}</div>
                            ${delivery.error_details ? `
                                <pre class="error-details">${this.escapeHtml(JSON.stringify(delivery.error_details, null, 2))}</pre>
                            ` : ''}
                        </div>
                    ` : ''}

                    <div class="details-section">
                        <h4>Trigger</h4>
                        <div class="detail-row">
                            <label>Triggered by:</label>
                            <span>${this.escapeHtml(delivery.triggered_by)}</span>
                        </div>
                        ${delivery.triggered_by_user_id ? `
                            <div class="detail-row">
                                <label>User ID:</label>
                                <span>${this.escapeHtml(delivery.triggered_by_user_id)}</span>
                            </div>
                        ` : ''}
                    </div>
                </div>
            `;

            const modal = this.createModal('newsletter-delivery-details-modal', 'Delivery Details', content);
            this.showModal(modal);
        } catch (error) {
            logger.error('Failed to load delivery details:', error);
        }
    }
}
