// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Rating Keys Manager
 *
 * Tracks library changes by showing rating key mappings:
 * - View new rating keys for re-added media
 * - View old rating keys (reverse lookup)
 * - Track when media was updated
 */

import { createLogger } from '../lib/logger';
import { API } from '../lib/api';
import type { ToastManager } from '../lib/toast';
import type { TautulliRatingKeyMapping } from '../lib/types';

const logger = createLogger('RatingKeysManager');

interface RatingKeysManagerOptions {
    api: API;
    containerId: string;
    toastManager?: ToastManager;
}

type LookupDirection = 'new' | 'old';

export class RatingKeysManager {
    private api: API;
    private containerId: string;
    private toastManager: ToastManager | null = null;
    private initialized = false;

    // State
    private lookupDirection: LookupDirection = 'new';
    private currentRatingKey: string = '';
    private ratingKeyMappings: TautulliRatingKeyMapping[] = [];
    private loading = false;
    private error: string | null = null;

    constructor(options: RatingKeysManagerOptions) {
        this.api = options.api;
        this.containerId = options.containerId;
        if (options.toastManager) {
            this.toastManager = options.toastManager;
        }
    }

    /**
     * Set toast manager
     */
    setToastManager(toast: ToastManager): void {
        this.toastManager = toast;
    }

    /**
     * Initialize the manager
     */
    async init(): Promise<void> {
        if (this.initialized) return;
        this.render();
        this.initialized = true;
    }

    /**
     * Lookup rating keys for a media item
     */
    async lookupRatingKeys(ratingKey: string, direction: LookupDirection = 'new'): Promise<void> {
        this.currentRatingKey = ratingKey;
        this.lookupDirection = direction;
        this.loading = true;
        this.error = null;
        this.ratingKeyMappings = [];
        this.render();

        try {
            if (direction === 'new') {
                this.ratingKeyMappings = await this.api.getNewRatingKeys(ratingKey);
            } else {
                this.ratingKeyMappings = await this.api.getOldRatingKeys(ratingKey);
            }
            this.loading = false;
            this.render();
        } catch (err) {
            logger.error('Failed to lookup rating keys:', err);
            this.loading = false;
            this.error = 'Failed to lookup rating keys';
            this.render();
        }
    }

    /**
     * Clear results
     */
    clear(): void {
        this.currentRatingKey = '';
        this.ratingKeyMappings = [];
        this.error = null;
        this.render();
    }

    /**
     * Render the component
     */
    private render(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        container.innerHTML = `
            <div class="rating-keys-manager" data-testid="rating-keys-manager">
                <div class="rating-keys-header">
                    <h3>Rating Key Tracker</h3>
                    <p class="hint">Track library changes when media is re-added or refreshed</p>
                </div>

                <div class="rating-keys-form">
                    <div class="form-row">
                        <div class="form-group">
                            <label for="rating-key-input">Rating Key</label>
                            <input type="text"
                                   id="rating-key-input"
                                   class="form-input"
                                   placeholder="Enter rating key (e.g., 12345)"
                                   value="${this.escapeHtml(this.currentRatingKey)}">
                        </div>
                        <div class="form-group">
                            <label for="lookup-direction">Lookup Direction</label>
                            <select id="lookup-direction" class="form-select">
                                <option value="new" ${this.lookupDirection === 'new' ? 'selected' : ''}>
                                    Find New Keys (Forward)
                                </option>
                                <option value="old" ${this.lookupDirection === 'old' ? 'selected' : ''}>
                                    Find Old Keys (Reverse)
                                </option>
                            </select>
                        </div>
                        <button class="btn-primary" id="lookup-btn" ${this.loading ? 'disabled' : ''}>
                            ${this.loading ? 'Looking up...' : 'Lookup'}
                        </button>
                    </div>
                    <p class="direction-hint">
                        ${this.lookupDirection === 'new'
                            ? 'Find what new rating keys this item was assigned after being re-added'
                            : 'Find what old rating keys this item had before being re-added'}
                    </p>
                </div>

                ${this.renderResults()}
            </div>
        `;

        this.attachEventListeners();
    }

    /**
     * Render results section
     */
    private renderResults(): string {
        if (this.loading) {
            return `
                <div class="rating-keys-results loading">
                    <div class="spinner"></div>
                    <p>Looking up rating keys...</p>
                </div>
            `;
        }

        if (this.error) {
            return `
                <div class="rating-keys-results error">
                    <p class="error-message">${this.escapeHtml(this.error)}</p>
                    <button class="btn-retry" id="retry-btn">Retry</button>
                </div>
            `;
        }

        if (!this.currentRatingKey) {
            return `
                <div class="rating-keys-results empty">
                    <p>Enter a rating key to look up its history</p>
                </div>
            `;
        }

        if (this.ratingKeyMappings.length === 0) {
            return `
                <div class="rating-keys-results empty" data-testid="no-mappings">
                    <p>No rating key mappings found for "${this.escapeHtml(this.currentRatingKey)}"</p>
                    <p class="hint">This item may not have been re-added to the library</p>
                </div>
            `;
        }

        return `
            <div class="rating-keys-results" data-testid="rating-keys-results">
                <div class="results-header">
                    <h4>
                        ${this.lookupDirection === 'new' ? 'New' : 'Old'} Rating Keys
                        <span class="count">(${this.ratingKeyMappings.length} found)</span>
                    </h4>
                </div>
                <div class="results-table-container">
                    <table class="rating-keys-table" data-testid="rating-keys-table">
                        <thead>
                            <tr>
                                <th>Title</th>
                                <th>Media Type</th>
                                <th>Old Key</th>
                                <th>New Key</th>
                                <th>Updated</th>
                            </tr>
                        </thead>
                        <tbody>
                            ${this.ratingKeyMappings.map(mapping => this.renderMappingRow(mapping)).join('')}
                        </tbody>
                    </table>
                </div>
            </div>
        `;
    }

    /**
     * Render a single mapping row
     */
    private renderMappingRow(mapping: TautulliRatingKeyMapping): string {
        const mediaTypeIcon = this.getMediaTypeIcon(mapping.media_type);
        const updatedAt = mapping.updated_at
            ? this.formatDate(mapping.updated_at)
            : '-';

        return `
            <tr data-old-key="${this.escapeHtml(mapping.old_rating_key)}"
                data-new-key="${this.escapeHtml(mapping.new_rating_key)}">
                <td class="title-cell">
                    <span class="title">${this.escapeHtml(mapping.title || 'Unknown')}</span>
                </td>
                <td class="type-cell">
                    <span class="media-type-badge" title="${mapping.media_type}">
                        ${mediaTypeIcon} ${mapping.media_type}
                    </span>
                </td>
                <td class="key-cell old-key">
                    <code>${this.escapeHtml(mapping.old_rating_key)}</code>
                    <button class="btn-copy" data-copy="${mapping.old_rating_key}" title="Copy">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                        </svg>
                    </button>
                </td>
                <td class="key-cell new-key">
                    <code>${this.escapeHtml(mapping.new_rating_key)}</code>
                    <button class="btn-copy" data-copy="${mapping.new_rating_key}" title="Copy">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                        </svg>
                    </button>
                </td>
                <td class="date-cell">${updatedAt}</td>
            </tr>
        `;
    }

    /**
     * Get icon for media type
     */
    private getMediaTypeIcon(mediaType: string): string {
        const icons: Record<string, string> = {
            movie: '🎬',
            show: '📺',
            season: '📁',
            episode: '📄',
            artist: '🎵',
            album: '💿',
            track: '🎵'
        };
        return icons[mediaType] || '📄';
    }

    /**
     * Attach event listeners
     */
    private attachEventListeners(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        // Lookup button
        const lookupBtn = container.querySelector('#lookup-btn');
        if (lookupBtn) {
            lookupBtn.addEventListener('click', () => this.handleLookup());
        }

        // Enter key on input
        const input = container.querySelector('#rating-key-input') as HTMLInputElement;
        if (input) {
            input.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    this.handleLookup();
                }
            });
        }

        // Direction select
        const directionSelect = container.querySelector('#lookup-direction') as HTMLSelectElement;
        if (directionSelect) {
            directionSelect.addEventListener('change', () => {
                this.lookupDirection = directionSelect.value as LookupDirection;
            });
        }

        // Retry button
        const retryBtn = container.querySelector('#retry-btn');
        if (retryBtn) {
            retryBtn.addEventListener('click', () => {
                if (this.currentRatingKey) {
                    this.lookupRatingKeys(this.currentRatingKey, this.lookupDirection);
                }
            });
        }

        // Copy buttons
        container.querySelectorAll('.btn-copy').forEach(btn => {
            btn.addEventListener('click', async (e) => {
                e.stopPropagation();
                const value = btn.getAttribute('data-copy');
                if (value) {
                    await this.copyToClipboard(value);
                }
            });
        });
    }

    /**
     * Handle lookup action
     */
    private handleLookup(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        const input = container.querySelector('#rating-key-input') as HTMLInputElement;
        const ratingKey = input?.value?.trim();

        if (!ratingKey) {
            this.toastManager?.warning('Please enter a rating key', 'Input Required', 3000);
            return;
        }

        this.lookupRatingKeys(ratingKey, this.lookupDirection);
    }

    /**
     * Copy value to clipboard
     */
    private async copyToClipboard(value: string): Promise<void> {
        try {
            await navigator.clipboard.writeText(value);
            this.toastManager?.success('Copied to clipboard', 'Copied', 2000);
        } catch (err) {
            logger.error('Failed to copy to clipboard:', err);
            this.toastManager?.error('Failed to copy', 'Error', 3000);
        }
    }

    /**
     * Format timestamp to date string
     */
    private formatDate(timestamp: number): string {
        const date = new Date(timestamp * 1000);
        return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], {
            hour: '2-digit',
            minute: '2-digit'
        });
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
     * Clean up resources
     */
    destroy(): void {
        this.initialized = false;
        this.ratingKeyMappings = [];
    }
}
