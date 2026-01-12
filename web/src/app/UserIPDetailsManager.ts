// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * User IP Details Manager
 *
 * Displays detailed IP history for users including:
 * - IP addresses with geolocation
 * - Last seen timestamps
 * - Platform and player information
 * - Play count per IP
 */

import { API } from '../lib/api';
import type { ToastManager } from '../lib/toast';
import type { TautulliUserIP } from '../lib/types';
import { createLogger } from '../lib/logger';

const logger = createLogger('UserIPDetailsManager');

interface UserIPDetailsOptions {
    api: API;
    containerId: string;
    toastManager?: ToastManager;
}

export class UserIPDetailsManager {
    private api: API;
    private containerId: string;
    private toastManager: ToastManager | null = null;
    private initialized = false;
    private currentUserId: number | null = null;
    private currentUsername: string = '';
    private ipDetails: TautulliUserIP[] = [];
    private sortField: keyof TautulliUserIP = 'last_seen';
    private sortDirection: 'asc' | 'desc' = 'desc';

    constructor(options: UserIPDetailsOptions) {
        this.api = options.api;
        this.containerId = options.containerId;
        if (options.toastManager) {
            this.toastManager = options.toastManager;
        }
    }

    /**
     * Set toast manager for notifications
     */
    setToastManager(toast: ToastManager): void {
        this.toastManager = toast;
    }

    /**
     * Initialize the manager
     */
    async init(): Promise<void> {
        if (this.initialized) return;
        this.renderEmptyState();
        this.initialized = true;
    }

    /**
     * Load IP details for a specific user
     */
    async loadUserIPDetails(userId: number, username: string): Promise<void> {
        this.currentUserId = userId;
        this.currentUsername = username;

        try {
            this.showLoadingState();
            this.ipDetails = await this.api.getUserIPDetails(userId);
            this.sortData();
            this.render();
        } catch (error) {
            logger.error('Failed to load user IP details:', error);
            this.showError('Failed to load IP details');
        }
    }

    /**
     * Sort data by field
     */
    setSortField(field: keyof TautulliUserIP): void {
        if (this.sortField === field) {
            this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc';
        } else {
            this.sortField = field;
            this.sortDirection = 'desc';
        }
        this.sortData();
        this.render();
    }

    /**
     * Sort the IP details array
     */
    private sortData(): void {
        this.ipDetails.sort((a, b) => {
            const aVal = a[this.sortField];
            const bVal = b[this.sortField];

            if (aVal === undefined || aVal === null) return 1;
            if (bVal === undefined || bVal === null) return -1;

            let comparison = 0;
            if (typeof aVal === 'number' && typeof bVal === 'number') {
                comparison = aVal - bVal;
            } else if (typeof aVal === 'string' && typeof bVal === 'string') {
                comparison = aVal.localeCompare(bVal);
            }

            return this.sortDirection === 'asc' ? comparison : -comparison;
        });
    }

    /**
     * Render the IP details view
     */
    private render(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        if (this.ipDetails.length === 0) {
            container.innerHTML = `
                <div class="user-ip-details" data-testid="user-ip-details">
                    <div class="ip-header">
                        <h3>IP History: ${this.escapeHtml(this.currentUsername)}</h3>
                        <button class="btn-close" id="ip-details-close" aria-label="Close">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <line x1="18" y1="6" x2="6" y2="18"/>
                                <line x1="6" y1="6" x2="18" y2="18"/>
                            </svg>
                        </button>
                    </div>
                    <div class="empty-state" data-testid="ip-empty">
                        <p>No IP history found for this user</p>
                    </div>
                </div>
            `;
            this.attachEventListeners();
            return;
        }

        // Calculate summary stats
        const uniqueCountries = new Set(this.ipDetails.map(ip => ip.country).filter(Boolean)).size;
        const totalPlays = this.ipDetails.reduce((sum, ip) => sum + (ip.play_count || 0), 0);
        const mostRecentIP = this.ipDetails.reduce((latest, ip) =>
            (ip.last_seen || 0) > (latest.last_seen || 0) ? ip : latest
        , this.ipDetails[0]);

        container.innerHTML = `
            <div class="user-ip-details" data-testid="user-ip-details">
                <div class="ip-header">
                    <h3>IP History: ${this.escapeHtml(this.currentUsername)}</h3>
                    <button class="btn-close" id="ip-details-close" aria-label="Close">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <line x1="18" y1="6" x2="6" y2="18"/>
                            <line x1="6" y1="6" x2="18" y2="18"/>
                        </svg>
                    </button>
                </div>

                <div class="ip-summary">
                    <div class="summary-stat">
                        <span class="stat-value">${this.ipDetails.length}</span>
                        <span class="stat-label">Unique IPs</span>
                    </div>
                    <div class="summary-stat">
                        <span class="stat-value">${uniqueCountries}</span>
                        <span class="stat-label">Countries</span>
                    </div>
                    <div class="summary-stat">
                        <span class="stat-value">${totalPlays.toLocaleString()}</span>
                        <span class="stat-label">Total Plays</span>
                    </div>
                    ${mostRecentIP.last_seen ? `
                        <div class="summary-stat">
                            <span class="stat-value">${this.formatDate(mostRecentIP.last_seen)}</span>
                            <span class="stat-label">Last Seen</span>
                        </div>
                    ` : ''}
                </div>

                <div class="ip-table-container">
                    <table class="ip-table" data-testid="ip-table">
                        <thead>
                            <tr>
                                ${this.renderSortableHeader('ip_address', 'IP Address')}
                                ${this.renderSortableHeader('city', 'Location')}
                                ${this.renderSortableHeader('platform', 'Platform')}
                                ${this.renderSortableHeader('player', 'Player')}
                                ${this.renderSortableHeader('play_count', 'Plays')}
                                ${this.renderSortableHeader('last_seen', 'Last Seen')}
                            </tr>
                        </thead>
                        <tbody>
                            ${this.ipDetails.map(ip => this.renderIPRow(ip)).join('')}
                        </tbody>
                    </table>
                </div>
            </div>
        `;

        this.attachEventListeners();
    }

    /**
     * Render sortable table header
     */
    private renderSortableHeader(field: keyof TautulliUserIP, label: string): string {
        const isActive = this.sortField === field;
        const direction = isActive ? this.sortDirection : 'none';
        const arrow = isActive
            ? (this.sortDirection === 'asc' ? '&#9650;' : '&#9660;')
            : '';

        return `
            <th class="sortable ${isActive ? 'active' : ''}"
                data-sort="${field}"
                aria-sort="${direction}">
                <span>${label}</span>
                <span class="sort-arrow">${arrow}</span>
            </th>
        `;
    }

    /**
     * Render a single IP row
     */
    private renderIPRow(ip: TautulliUserIP): string {
        const location = [ip.city, ip.region, ip.country]
            .filter(Boolean)
            .join(', ') || 'Unknown';

        const countryFlag = ip.code ? this.getCountryFlag(ip.code) : '';

        return `
            <tr data-ip="${this.escapeHtml(ip.ip_address)}">
                <td class="ip-address">
                    <code>${this.escapeHtml(ip.ip_address)}</code>
                    ${ip.host ? `<span class="ip-host">${this.escapeHtml(ip.host)}</span>` : ''}
                </td>
                <td class="ip-location">
                    ${countryFlag ? `<span class="country-flag">${countryFlag}</span>` : ''}
                    <span>${this.escapeHtml(location)}</span>
                    ${ip.latitude && ip.longitude ? `
                        <span class="ip-coords" title="Coordinates">(${ip.latitude.toFixed(2)}, ${ip.longitude.toFixed(2)})</span>
                    ` : ''}
                </td>
                <td class="ip-platform">${ip.platform ? this.escapeHtml(ip.platform) : '-'}</td>
                <td class="ip-player">${ip.player ? this.escapeHtml(ip.player) : '-'}</td>
                <td class="ip-plays">${ip.play_count?.toLocaleString() || '-'}</td>
                <td class="ip-last-seen">${ip.last_seen ? this.formatDateTime(ip.last_seen) : '-'}</td>
            </tr>
        `;
    }

    /**
     * Render empty state
     */
    private renderEmptyState(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        container.innerHTML = `
            <div class="user-ip-details empty" data-testid="user-ip-details">
                <div class="empty-state" data-testid="ip-select-prompt">
                    <p>Select a user to view their IP history</p>
                </div>
            </div>
        `;
    }

    /**
     * Show loading state
     */
    private showLoadingState(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        container.innerHTML = `
            <div class="user-ip-details loading" data-testid="user-ip-details">
                <div class="loading-state">
                    <div class="spinner"></div>
                    <p>Loading IP history...</p>
                </div>
            </div>
        `;
    }

    /**
     * Attach event listeners
     */
    private attachEventListeners(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        // Close button
        const closeBtn = container.querySelector('#ip-details-close');
        if (closeBtn) {
            closeBtn.addEventListener('click', () => {
                this.currentUserId = null;
                this.currentUsername = '';
                this.ipDetails = [];
                this.renderEmptyState();
            });
        }

        // Sortable headers
        container.querySelectorAll('th.sortable').forEach(th => {
            th.addEventListener('click', (e) => {
                const field = (e.currentTarget as HTMLElement).getAttribute('data-sort') as keyof TautulliUserIP;
                if (field) {
                    this.setSortField(field);
                }
            });
        });
    }

    /**
     * Get country flag emoji from country code
     */
    private getCountryFlag(code: string): string {
        if (!code || code.length !== 2) return '';
        const codePoints = code
            .toUpperCase()
            .split('')
            .map(char => 127397 + char.charCodeAt(0));
        return String.fromCodePoint(...codePoints);
    }

    /**
     * Format timestamp to date string
     */
    private formatDate(timestamp: number): string {
        const date = new Date(timestamp * 1000);
        return date.toLocaleDateString();
    }

    /**
     * Format timestamp to date/time string
     */
    private formatDateTime(timestamp: number): string {
        const date = new Date(timestamp * 1000);
        return date.toLocaleString();
    }

    /**
     * Escape HTML to prevent XSS
     */
    private escapeHtml(unsafe: string): string {
        return unsafe
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#039;');
    }

    /**
     * Show error notification
     */
    private showError(message: string): void {
        const container = document.getElementById(this.containerId);
        if (container) {
            container.innerHTML = `
                <div class="user-ip-details error" data-testid="user-ip-details">
                    <div class="error-state">
                        <p>${this.escapeHtml(message)}</p>
                        <button class="btn-retry" id="ip-retry">Retry</button>
                    </div>
                </div>
            `;

            const retryBtn = container.querySelector('#ip-retry');
            if (retryBtn && this.currentUserId) {
                retryBtn.addEventListener('click', () => {
                    this.loadUserIPDetails(this.currentUserId!, this.currentUsername);
                });
            }
        }

        if (this.toastManager) {
            this.toastManager.error(message, 'Error', 5000);
        }
    }

    /**
     * Clean up resources
     */
    destroy(): void {
        this.initialized = false;
        this.ipDetails = [];
        this.currentUserId = null;
    }
}
