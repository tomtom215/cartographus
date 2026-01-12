// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * OverviewRenderer - Newsletter Overview/Stats Dashboard
 *
 * Displays overall newsletter statistics and recent activity.
 */

import { createLogger } from '../../lib/logger';
import type { API } from '../../lib/api';
import type { NewsletterStats, NewsletterDelivery } from '../../lib/types/newsletter';
import { BaseNewsletterRenderer, NewsletterConfig } from './BaseNewsletterRenderer';

const logger = createLogger('NewsletterOverviewRenderer');

export class OverviewRenderer extends BaseNewsletterRenderer {
    private stats: NewsletterStats | null = null;
    private recentDeliveries: NewsletterDelivery[] = [];

    constructor(api: API, config: NewsletterConfig) {
        super(api, config);
    }

    // =========================================================================
    // Public API
    // =========================================================================

    setupEventListeners(): void {
        document.getElementById('newsletter-overview-refresh-btn')?.addEventListener('click', () => {
            this.load();
        });
    }

    async load(): Promise<void> {
        try {
            const [statsResult, deliveriesResult] = await Promise.all([
                this.api.getNewsletterStats().catch(() => null),
                this.api.getNewsletterDeliveries({ limit: 10 }).catch(() => ({ deliveries: [] })),
            ]);

            this.stats = statsResult;
            this.recentDeliveries = deliveriesResult.deliveries || [];

            this.updateDisplay();
        } catch (error) {
            logger.error('Failed to load newsletter overview:', error);
        }
    }

    // =========================================================================
    // Display Updates
    // =========================================================================

    private updateDisplay(): void {
        if (this.stats) {
            // Update stat cards
            this.setElementText('newsletter-total-templates', this.formatNumber(this.stats.total_templates));
            this.setElementText('newsletter-active-templates', this.formatNumber(this.stats.active_templates));
            this.setElementText('newsletter-total-schedules', this.formatNumber(this.stats.total_schedules));
            this.setElementText('newsletter-enabled-schedules', this.formatNumber(this.stats.enabled_schedules));
            this.setElementText('newsletter-total-deliveries', this.formatNumber(this.stats.total_deliveries));
            this.setElementText('newsletter-successful-deliveries', this.formatNumber(this.stats.successful_deliveries));
            this.setElementText('newsletter-failed-deliveries', this.formatNumber(this.stats.failed_deliveries));

            // Success rate
            const successRate = this.stats.total_deliveries > 0
                ? ((this.stats.successful_deliveries / this.stats.total_deliveries) * 100).toFixed(1)
                : '0';
            this.setElementText('newsletter-success-rate', `${successRate}%`);

            // Recent activity
            this.setElementText('newsletter-last-7-days', this.formatNumber(this.stats.last_7_days_deliveries));
            this.setElementText('newsletter-last-30-days', this.formatNumber(this.stats.last_30_days_deliveries));

            // Channel distribution
            this.renderChannelDistribution();
        }

        // Render recent deliveries
        this.renderRecentDeliveries();
    }

    private renderChannelDistribution(): void {
        const container = document.getElementById('newsletter-channel-distribution');
        if (!container || !this.stats) return;

        const channels = this.stats.deliveries_by_channel;
        const total = Object.values(channels).reduce((sum, count) => sum + count, 0);

        if (total === 0) {
            container.innerHTML = '<p class="empty-state">No deliveries yet</p>';
            return;
        }

        const entries = Object.entries(channels)
            .sort(([, a], [, b]) => b - a);

        container.innerHTML = entries.map(([channel, count]) => {
            const percentage = ((count / total) * 100).toFixed(1);
            return `
                <div class="channel-stat-row">
                    <span class="channel-label">${this.renderChannelBadge(channel as any)}</span>
                    <div class="channel-bar-container">
                        <div class="channel-bar" style="width: ${percentage}%; background: ${this.getChannelColor(channel as any)}"></div>
                    </div>
                    <span class="channel-count">${this.formatNumber(count)} (${percentage}%)</span>
                </div>
            `;
        }).join('');
    }

    private renderRecentDeliveries(): void {
        const tbody = document.getElementById('newsletter-recent-deliveries-tbody');
        if (!tbody) return;

        if (this.recentDeliveries.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="table-empty">No recent deliveries</td></tr>';
            return;
        }

        tbody.innerHTML = this.recentDeliveries.map(delivery => `
            <tr class="newsletter-delivery-row" data-id="${delivery.id}">
                <td class="delivery-time">${this.formatTimestamp(delivery.started_at)}</td>
                <td class="delivery-schedule">${this.escapeHtml(delivery.schedule_name || delivery.template_name || '--')}</td>
                <td class="delivery-channel">${this.renderChannelBadge(delivery.channel)}</td>
                <td class="delivery-recipients">
                    ${delivery.recipients_delivered}/${delivery.recipients_total}
                    ${delivery.recipients_failed > 0 ? `<span class="failed-count">(${delivery.recipients_failed} failed)</span>` : ''}
                </td>
                <td class="delivery-status">${this.renderStatusBadge(delivery.status)}</td>
                <td class="delivery-duration">
                    ${delivery.duration_ms ? `${(delivery.duration_ms / 1000).toFixed(1)}s` : '--'}
                </td>
            </tr>
        `).join('');
    }
}
