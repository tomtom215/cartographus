// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Metadata Deep-Dive Manager
 *
 * Displays rich metadata for media items including:
 * - Basic info (title, year, studio, etc.)
 * - Ratings (critic, audience, user)
 * - Summary and tagline
 * - Credits (directors, writers, actors)
 * - Genres, labels, collections
 * - Media info (resolution, codecs, etc.)
 */

import { API } from '../lib/api';
import type { ToastManager } from '../lib/toast';
import type { TautulliFullMetadataData, TautulliMediaInfo } from '../lib/types';
import { createLogger } from '../lib/logger';

const logger = createLogger('MetadataDeepDiveManager');

interface MetadataDeepDiveOptions {
    api: API;
    containerId: string;
    toastManager?: ToastManager;
}

export class MetadataDeepDiveManager {
    private api: API;
    private containerId: string;
    private toastManager: ToastManager | null = null;
    private initialized = false;

    // State
    private currentRatingKey: string = '';
    private metadata: TautulliFullMetadataData | null = null;
    private loading = false;
    private error: string | null = null;

    constructor(options: MetadataDeepDiveOptions) {
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
     * Load metadata for a media item
     */
    async loadMetadata(ratingKey: string): Promise<void> {
        this.currentRatingKey = ratingKey;
        this.loading = true;
        this.error = null;
        this.metadata = null;
        this.render();

        try {
            this.metadata = await this.api.getMetadata(ratingKey);
            this.loading = false;
            this.render();
        } catch (err) {
            logger.error('Failed to load metadata:', err);
            this.loading = false;
            this.error = 'Failed to load metadata';
            this.render();
        }
    }

    /**
     * Clear the current metadata
     */
    clear(): void {
        this.currentRatingKey = '';
        this.metadata = null;
        this.error = null;
        this.render();
    }

    /**
     * Main render method
     */
    private render(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        container.innerHTML = `
            <div class="metadata-deep-dive">
                <div class="metadata-header">
                    <h3>Media Metadata</h3>
                    <p class="hint">Enter a rating key to view detailed metadata</p>
                </div>

                <div class="metadata-search">
                    <div class="search-row">
                        <input type="text"
                               id="metadata-rating-key"
                               class="form-input"
                               placeholder="Enter rating key (e.g., 12345)"
                               value="${this.escapeHtml(this.currentRatingKey)}">
                        <button class="btn-primary" id="metadata-search-btn" ${this.loading ? 'disabled' : ''}>
                            ${this.loading ? 'Loading...' : 'Load Metadata'}
                        </button>
                    </div>
                </div>

                ${this.renderContent()}
            </div>
        `;

        this.attachEventListeners();
    }

    /**
     * Render main content
     */
    private renderContent(): string {
        if (this.loading) {
            return `
                <div class="metadata-content loading">
                    <div class="spinner"></div>
                    <p>Loading metadata...</p>
                </div>
            `;
        }

        if (this.error) {
            return `
                <div class="metadata-content error">
                    <p class="error-message">${this.escapeHtml(this.error)}</p>
                    <button class="btn-retry" id="metadata-retry">Retry</button>
                </div>
            `;
        }

        if (!this.metadata) {
            return `
                <div class="metadata-content empty">
                    <p>Enter a rating key above to view metadata</p>
                </div>
            `;
        }

        return `
            <div class="metadata-content" data-testid="metadata-content">
                ${this.renderBasicInfo()}
                ${this.renderRatings()}
                ${this.renderSummary()}
                ${this.renderCredits()}
                ${this.renderCategories()}
                ${this.renderMediaInfo()}
                ${this.renderDates()}
            </div>
        `;
    }

    /**
     * Render basic info section
     */
    private renderBasicInfo(): string {
        const m = this.metadata!;
        const fullTitle = this.getFullTitle();

        return `
            <div class="metadata-section basic-info" data-testid="basic-info">
                <h4>Basic Information</h4>
                <div class="info-grid">
                    <div class="info-item">
                        <span class="label">Title</span>
                        <span class="value">${this.escapeHtml(fullTitle)}</span>
                    </div>
                    ${m.original_title ? `
                        <div class="info-item">
                            <span class="label">Original Title</span>
                            <span class="value">${this.escapeHtml(m.original_title)}</span>
                        </div>
                    ` : ''}
                    ${m.year ? `
                        <div class="info-item">
                            <span class="label">Year</span>
                            <span class="value">${m.year}</span>
                        </div>
                    ` : ''}
                    ${m.studio ? `
                        <div class="info-item">
                            <span class="label">Studio</span>
                            <span class="value">${this.escapeHtml(m.studio)}</span>
                        </div>
                    ` : ''}
                    ${m.content_rating ? `
                        <div class="info-item">
                            <span class="label">Content Rating</span>
                            <span class="value badge">${this.escapeHtml(m.content_rating)}</span>
                        </div>
                    ` : ''}
                    ${m.duration ? `
                        <div class="info-item">
                            <span class="label">Duration</span>
                            <span class="value">${this.formatDuration(m.duration)}</span>
                        </div>
                    ` : ''}
                    <div class="info-item">
                        <span class="label">Rating Key</span>
                        <span class="value code">${this.escapeHtml(m.rating_key)}</span>
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Render ratings section
     */
    private renderRatings(): string {
        const m = this.metadata!;
        const hasRatings = m.rating || m.audience_rating || m.user_rating;

        if (!hasRatings) return '';

        return `
            <div class="metadata-section ratings" data-testid="ratings">
                <h4>Ratings</h4>
                <div class="ratings-grid">
                    ${m.rating ? `
                        <div class="rating-item">
                            <div class="rating-value">${m.rating.toFixed(1)}</div>
                            <div class="rating-label">Critic Rating</div>
                        </div>
                    ` : ''}
                    ${m.audience_rating ? `
                        <div class="rating-item">
                            <div class="rating-value">${m.audience_rating.toFixed(1)}</div>
                            <div class="rating-label">Audience Rating</div>
                        </div>
                    ` : ''}
                    ${m.user_rating ? `
                        <div class="rating-item user">
                            <div class="rating-value">${m.user_rating.toFixed(1)}</div>
                            <div class="rating-label">Your Rating</div>
                        </div>
                    ` : ''}
                </div>
            </div>
        `;
    }

    /**
     * Render summary section
     */
    private renderSummary(): string {
        const m = this.metadata!;

        if (!m.summary && !m.tagline) return '';

        return `
            <div class="metadata-section summary" data-testid="summary">
                <h4>Summary</h4>
                ${m.tagline ? `<p class="tagline">"${this.escapeHtml(m.tagline)}"</p>` : ''}
                ${m.summary ? `<p class="description">${this.escapeHtml(m.summary)}</p>` : ''}
            </div>
        `;
    }

    /**
     * Render credits section
     */
    private renderCredits(): string {
        const m = this.metadata!;
        const hasCredits = (m.directors?.length > 0) || (m.writers?.length > 0) || (m.actors?.length > 0);

        if (!hasCredits) return '';

        return `
            <div class="metadata-section credits" data-testid="credits">
                <h4>Credits</h4>
                <div class="credits-grid">
                    ${m.directors?.length > 0 ? `
                        <div class="credit-group">
                            <span class="credit-label">Directors</span>
                            <div class="credit-list">
                                ${m.directors.map(d => `<span class="credit-item">${this.escapeHtml(d)}</span>`).join('')}
                            </div>
                        </div>
                    ` : ''}
                    ${m.writers?.length > 0 ? `
                        <div class="credit-group">
                            <span class="credit-label">Writers</span>
                            <div class="credit-list">
                                ${m.writers.map(w => `<span class="credit-item">${this.escapeHtml(w)}</span>`).join('')}
                            </div>
                        </div>
                    ` : ''}
                    ${m.actors?.length > 0 ? `
                        <div class="credit-group">
                            <span class="credit-label">Cast</span>
                            <div class="credit-list">
                                ${m.actors.slice(0, 10).map(a => `<span class="credit-item">${this.escapeHtml(a)}</span>`).join('')}
                                ${m.actors.length > 10 ? `<span class="credit-more">+${m.actors.length - 10} more</span>` : ''}
                            </div>
                        </div>
                    ` : ''}
                </div>
            </div>
        `;
    }

    /**
     * Render categories section (genres, labels, collections)
     */
    private renderCategories(): string {
        const m = this.metadata!;
        const hasCategories = (m.genres?.length > 0) || (m.labels?.length > 0) || (m.collections?.length > 0);

        if (!hasCategories) return '';

        return `
            <div class="metadata-section categories" data-testid="categories">
                <h4>Categories</h4>
                <div class="categories-grid">
                    ${m.genres?.length > 0 ? `
                        <div class="category-group">
                            <span class="category-label">Genres</span>
                            <div class="tag-list">
                                ${m.genres.map(g => `<span class="tag genre">${this.escapeHtml(g)}</span>`).join('')}
                            </div>
                        </div>
                    ` : ''}
                    ${m.collections?.length > 0 ? `
                        <div class="category-group">
                            <span class="category-label">Collections</span>
                            <div class="tag-list">
                                ${m.collections.map(c => `<span class="tag collection">${this.escapeHtml(c)}</span>`).join('')}
                            </div>
                        </div>
                    ` : ''}
                    ${m.labels?.length > 0 ? `
                        <div class="category-group">
                            <span class="category-label">Labels</span>
                            <div class="tag-list">
                                ${m.labels.map(l => `<span class="tag label">${this.escapeHtml(l)}</span>`).join('')}
                            </div>
                        </div>
                    ` : ''}
                </div>
            </div>
        `;
    }

    /**
     * Render media info section
     */
    private renderMediaInfo(): string {
        const m = this.metadata!;

        if (!m.media_info || m.media_info.length === 0) return '';

        return `
            <div class="metadata-section media-info" data-testid="media-info">
                <h4>Media Information</h4>
                <div class="media-info-list">
                    ${m.media_info.map((mi, idx) => this.renderMediaInfoItem(mi, idx)).join('')}
                </div>
            </div>
        `;
    }

    /**
     * Render a single media info item
     */
    private renderMediaInfoItem(mi: TautulliMediaInfo, index: number): string {
        const videoInfo = [
            mi.video_resolution,
            mi.video_codec,
            mi.video_profile,
            mi.video_bit_depth ? `${mi.video_bit_depth}-bit` : null,
            mi.video_framerate
        ].filter(Boolean).join(' / ');

        const audioInfo = [
            mi.audio_codec,
            mi.audio_channel_layout || (mi.audio_channels ? `${mi.audio_channels}ch` : null)
        ].filter(Boolean).join(' / ');

        return `
            <div class="media-info-item" data-index="${index}">
                <div class="media-info-header">
                    <span class="media-version">Version ${index + 1}</span>
                    <span class="media-container">${mi.container.toUpperCase()}</span>
                    ${mi.optimized_version ? '<span class="optimized-badge">Optimized</span>' : ''}
                </div>
                <div class="media-info-details">
                    ${mi.width && mi.height ? `
                        <div class="detail-row">
                            <span class="detail-label">Resolution</span>
                            <span class="detail-value">${mi.width}x${mi.height}</span>
                        </div>
                    ` : ''}
                    ${videoInfo ? `
                        <div class="detail-row">
                            <span class="detail-label">Video</span>
                            <span class="detail-value">${this.escapeHtml(videoInfo)}</span>
                        </div>
                    ` : ''}
                    ${audioInfo ? `
                        <div class="detail-row">
                            <span class="detail-label">Audio</span>
                            <span class="detail-value">${this.escapeHtml(audioInfo)}</span>
                        </div>
                    ` : ''}
                    ${mi.bitrate ? `
                        <div class="detail-row">
                            <span class="detail-label">Bitrate</span>
                            <span class="detail-value">${this.formatBitrate(mi.bitrate)}</span>
                        </div>
                    ` : ''}
                </div>
            </div>
        `;
    }

    /**
     * Render dates section
     */
    private renderDates(): string {
        const m = this.metadata!;
        const hasDates = m.originally_available_at || m.added_at || m.updated_at || m.last_viewed_at;

        if (!hasDates) return '';

        return `
            <div class="metadata-section dates" data-testid="dates">
                <h4>Dates</h4>
                <div class="dates-grid">
                    ${m.originally_available_at ? `
                        <div class="date-item">
                            <span class="date-label">Release Date</span>
                            <span class="date-value">${this.escapeHtml(m.originally_available_at)}</span>
                        </div>
                    ` : ''}
                    ${m.added_at ? `
                        <div class="date-item">
                            <span class="date-label">Added to Library</span>
                            <span class="date-value">${this.formatDate(m.added_at)}</span>
                        </div>
                    ` : ''}
                    ${m.updated_at ? `
                        <div class="date-item">
                            <span class="date-label">Last Updated</span>
                            <span class="date-value">${this.formatDate(m.updated_at)}</span>
                        </div>
                    ` : ''}
                    ${m.last_viewed_at ? `
                        <div class="date-item">
                            <span class="date-label">Last Viewed</span>
                            <span class="date-value">${this.formatDate(m.last_viewed_at)}</span>
                        </div>
                    ` : ''}
                </div>
            </div>
        `;
    }

    /**
     * Get full title including parent/grandparent
     */
    private getFullTitle(): string {
        const m = this.metadata!;
        const parts = [m.grandparent_title, m.parent_title, m.title].filter(Boolean);
        return parts.join(' - ') || 'Unknown';
    }

    /**
     * Attach event listeners
     */
    private attachEventListeners(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        // Search button
        const searchBtn = container.querySelector('#metadata-search-btn');
        if (searchBtn) {
            searchBtn.addEventListener('click', () => this.handleSearch());
        }

        // Enter key
        const input = container.querySelector('#metadata-rating-key') as HTMLInputElement;
        if (input) {
            input.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    this.handleSearch();
                }
            });
        }

        // Retry button
        const retryBtn = container.querySelector('#metadata-retry');
        if (retryBtn) {
            retryBtn.addEventListener('click', () => {
                if (this.currentRatingKey) {
                    this.loadMetadata(this.currentRatingKey);
                }
            });
        }
    }

    /**
     * Handle search action
     */
    private handleSearch(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        const input = container.querySelector('#metadata-rating-key') as HTMLInputElement;
        const ratingKey = input?.value?.trim();

        if (!ratingKey) {
            this.toastManager?.warning('Please enter a rating key', 'Input Required', 3000);
            return;
        }

        this.loadMetadata(ratingKey);
    }

    /**
     * Format duration in milliseconds to human readable
     */
    private formatDuration(ms: number): string {
        const hours = Math.floor(ms / 3600000);
        const minutes = Math.floor((ms % 3600000) / 60000);

        if (hours > 0) {
            return `${hours}h ${minutes}m`;
        }
        return `${minutes}m`;
    }

    /**
     * Format bitrate to human readable
     */
    private formatBitrate(kbps: number): string {
        if (kbps >= 1000) {
            return `${(kbps / 1000).toFixed(1)} Mbps`;
        }
        return `${kbps} kbps`;
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
        this.metadata = null;
    }
}
