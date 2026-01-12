// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * BaseNewsletterRenderer - Base class for newsletter page renderers
 *
 * Provides shared utilities for all newsletter sub-page renderers.
 */

import type { API } from '../../lib/api';
import {
    NEWSLETTER_TYPES,
    CHANNEL_CONFIG,
    STATUS_COLORS as NEWSLETTER_STATUS_COLORS,
    STATUS_NAMES as NEWSLETTER_STATUS_NAMES,
} from '../../lib/types/newsletter';
import type {
    NewsletterType,
    DeliveryChannel,
    DeliveryStatus,
} from '../../lib/types/newsletter';

/** Configuration for newsletter renderers */
export interface NewsletterConfig {
    autoRefreshMs: number;
    maxTableEntries: number;
}

export const DEFAULT_NEWSLETTER_CONFIG: NewsletterConfig = {
    autoRefreshMs: 60000, // 1 minute
    maxTableEntries: 50,
};

/**
 * Base renderer class with shared utilities for newsletter pages
 */
export abstract class BaseNewsletterRenderer {
    protected api: API;
    protected config: NewsletterConfig;

    constructor(api: API, config: NewsletterConfig = DEFAULT_NEWSLETTER_CONFIG) {
        this.api = api;
        this.config = config;
    }

    /**
     * Load data for this page
     */
    abstract load(): Promise<void>;

    /**
     * Setup event listeners for this page
     */
    abstract setupEventListeners(): void;

    // =========================================================================
    // DOM Utilities
    // =========================================================================

    protected setElementText(id: string, text: string): void {
        const el = document.getElementById(id);
        if (el) el.textContent = text;
    }

    protected setElementHTML(id: string, html: string): void {
        const el = document.getElementById(id);
        if (el) el.innerHTML = html;
    }

    protected escapeHtml(str: string): string {
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    protected showElement(id: string): void {
        const el = document.getElementById(id);
        if (el) el.style.display = 'block';
    }

    protected hideElement(id: string): void {
        const el = document.getElementById(id);
        if (el) el.style.display = 'none';
    }

    // =========================================================================
    // Formatting Utilities
    // =========================================================================

    protected formatTimestamp(timestamp: string): string {
        try {
            const date = new Date(timestamp);
            return date.toLocaleString();
        } catch {
            return timestamp;
        }
    }

    protected formatTimeAgo(date: Date): string {
        const now = new Date();
        const diffMs = now.getTime() - date.getTime();
        const diffMins = Math.floor(diffMs / 60000);
        const diffHours = Math.floor(diffMins / 60);
        const diffDays = Math.floor(diffHours / 24);

        if (diffMins < 1) return 'just now';
        if (diffMins < 60) return `${diffMins}m ago`;
        if (diffHours < 24) return `${diffHours}h ago`;
        return `${diffDays}d ago`;
    }

    protected formatNewsletterType(type: NewsletterType): string {
        return NEWSLETTER_TYPES[type]?.label || type;
    }

    protected getNewsletterTypeIcon(type: NewsletterType): string {
        return NEWSLETTER_TYPES[type]?.icon || '';
    }

    protected getNewsletterTypeColor(type: NewsletterType): string {
        return NEWSLETTER_TYPES[type]?.color || '#666';
    }

    protected formatChannel(channel: DeliveryChannel): string {
        return CHANNEL_CONFIG[channel]?.label || channel;
    }

    protected getChannelIcon(channel: DeliveryChannel): string {
        return CHANNEL_CONFIG[channel]?.icon || '';
    }

    protected getChannelColor(channel: DeliveryChannel): string {
        return CHANNEL_CONFIG[channel]?.color || '#666';
    }

    protected formatStatus(status: DeliveryStatus): string {
        return NEWSLETTER_STATUS_NAMES[status] || status;
    }

    protected getStatusColor(status: DeliveryStatus): string {
        return NEWSLETTER_STATUS_COLORS[status] || '#666';
    }

    protected formatCronExpression(cron: string): string {
        // Simple cron expression parser for display
        const parts = cron.split(' ');
        if (parts.length < 5) return cron;

        const [minute, hour, dayOfMonth, month, dayOfWeek] = parts;

        // Common patterns
        if (cron === '0 9 * * 1') return 'Every Monday at 9:00 AM';
        if (cron === '0 9 * * *') return 'Daily at 9:00 AM';
        if (cron === '0 0 1 * *') return 'First of every month at midnight';
        if (cron === '0 9 * * 0') return 'Every Sunday at 9:00 AM';

        // Weekly
        if (dayOfWeek !== '*' && dayOfMonth === '*' && month === '*') {
            const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
            const dayName = days[parseInt(dayOfWeek, 10)] || dayOfWeek;
            return `Every ${dayName} at ${this.formatTime(hour, minute)}`;
        }

        // Daily
        if (dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
            return `Daily at ${this.formatTime(hour, minute)}`;
        }

        // Monthly
        if (month === '*' && dayOfWeek === '*') {
            const dayOrdinal = this.ordinal(parseInt(dayOfMonth, 10));
            return `${dayOrdinal} of every month at ${this.formatTime(hour, minute)}`;
        }

        return cron;
    }

    private formatTime(hour: string, minute: string): string {
        const h = parseInt(hour, 10);
        const m = parseInt(minute, 10);
        const ampm = h >= 12 ? 'PM' : 'AM';
        const displayHour = h % 12 || 12;
        const displayMinute = m.toString().padStart(2, '0');
        return `${displayHour}:${displayMinute} ${ampm}`;
    }

    private ordinal(n: number): string {
        const s = ['th', 'st', 'nd', 'rd'];
        const v = n % 100;
        return n + (s[(v - 20) % 10] || s[v] || s[0]);
    }

    protected formatNumber(num: number): string {
        return num.toLocaleString();
    }

    protected formatPercent(value: number, total: number): string {
        if (total === 0) return '0%';
        return `${((value / total) * 100).toFixed(1)}%`;
    }

    // =========================================================================
    // Input Utilities
    // =========================================================================

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    protected debounce<T extends (...args: any[]) => void>(
        fn: T,
        delay: number
    ): (...args: Parameters<T>) => void {
        let timeoutId: ReturnType<typeof setTimeout> | null = null;
        return (...args: Parameters<T>) => {
            if (timeoutId) {
                clearTimeout(timeoutId);
            }
            timeoutId = setTimeout(() => fn(...args), delay);
        };
    }

    // =========================================================================
    // Modal Utilities
    // =========================================================================

    protected createModal(id: string, title: string, content: string): HTMLElement {
        let modal = document.getElementById(id);
        if (!modal) {
            modal = document.createElement('div');
            modal.id = id;
            modal.className = 'modal-overlay';
            document.body.appendChild(modal);
        }

        modal.innerHTML = `
            <div class="modal-content newsletter-modal">
                <div class="modal-header">
                    <h3>${this.escapeHtml(title)}</h3>
                    <button class="modal-close" aria-label="Close">&times;</button>
                </div>
                <div class="modal-body">
                    ${content}
                </div>
            </div>
        `;

        return modal;
    }

    protected showModal(modal: HTMLElement): void {
        modal.style.display = 'flex';

        // Close button handler
        const closeBtn = modal.querySelector('.modal-close');
        closeBtn?.addEventListener('click', () => {
            modal.style.display = 'none';
        });

        // Click outside to close
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.style.display = 'none';
            }
        });
    }

    protected closeModal(id: string): void {
        const modal = document.getElementById(id);
        if (modal) {
            modal.style.display = 'none';
        }
    }

    // =========================================================================
    // Badge Rendering
    // =========================================================================

    protected renderTypeBadge(type: NewsletterType): string {
        const config = NEWSLETTER_TYPES[type];
        return `<span class="newsletter-type-badge" style="background: ${config?.color || '#666'}">${config?.icon || ''} ${config?.label || type}</span>`;
    }

    protected renderChannelBadge(channel: DeliveryChannel): string {
        const config = CHANNEL_CONFIG[channel];
        return `<span class="newsletter-channel-badge" style="background: ${config?.color || '#666'}">${config?.icon || ''} ${config?.label || channel}</span>`;
    }

    protected renderStatusBadge(status: DeliveryStatus): string {
        const color = NEWSLETTER_STATUS_COLORS[status] || '#666';
        const name = NEWSLETTER_STATUS_NAMES[status] || status;
        return `<span class="newsletter-status-badge" style="background: ${color}">${name}</span>`;
    }

    protected renderChannelBadges(channels: DeliveryChannel[]): string {
        return channels.map(ch => this.renderChannelBadge(ch)).join(' ');
    }
}
