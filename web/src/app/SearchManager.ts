// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Search Manager
 *
 * Provides search functionality for the media library:
 * - Search input with debouncing
 * - Search results display with thumbnails
 * - Click to view metadata
 * - Filter by media type
 * - Fuzzy search with RapidFuzz extension (when available)
 */

import { API } from '../lib/api';
import type { ToastManager } from '../lib/toast';
import type { TautulliSearchResult, TautulliSearchData, FuzzySearchResult } from '../lib/types';
import { createLogger } from '../lib/logger';

const logger = createLogger('SearchManager');

interface SearchManagerOptions {
    api: API;
    containerId: string;
    toastManager?: ToastManager;
    onSelectResult?: (ratingKey: string) => void;
}

export class SearchManager {
    private api: API;
    private containerId: string;
    private toastManager: ToastManager | null = null;
    private onSelectResult: ((ratingKey: string) => void) | null = null;
    private initialized = false;

    // State
    private query: string = '';
    private results: TautulliSearchResult[] = [];
    private resultsCount: number = 0;
    private loading = false;
    private error: string | null = null;
    private searchTimeout: number | null = null;
    private filterType: string = 'all';
    private isFuzzySearch: boolean = false;

    constructor(options: SearchManagerOptions) {
        this.api = options.api;
        this.containerId = options.containerId;
        if (options.toastManager) {
            this.toastManager = options.toastManager;
        }
        if (options.onSelectResult) {
            this.onSelectResult = options.onSelectResult;
        }
    }

    /**
     * Set toast manager
     */
    setToastManager(toast: ToastManager): void {
        this.toastManager = toast;
    }

    /**
     * Set callback for result selection
     */
    setOnSelectResult(callback: (ratingKey: string) => void): void {
        this.onSelectResult = callback;
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
     * Perform search with debouncing
     */
    private debouncedSearch(): void {
        if (this.searchTimeout) {
            clearTimeout(this.searchTimeout);
        }
        this.searchTimeout = window.setTimeout(() => {
            this.performSearch();
        }, 300);
    }

    /**
     * Perform the actual search
     * Tries fuzzy search first (local DuckDB with RapidFuzz), falls back to Tautulli API
     */
    private async performSearch(): Promise<void> {
        const trimmedQuery = this.query.trim();

        if (!trimmedQuery) {
            this.results = [];
            this.resultsCount = 0;
            this.isFuzzySearch = false;
            this.render();
            return;
        }

        this.loading = true;
        this.error = null;
        this.render();

        try {
            // Try fuzzy search first (local DuckDB)
            const fuzzyResponse = await this.api.fuzzySearch(trimmedQuery, { minScore: 60, limit: 50 });

            if (fuzzyResponse.results && fuzzyResponse.results.length > 0) {
                // Use fuzzy search results
                this.results = this.mapFuzzyResultsToTautulli(fuzzyResponse.results);
                this.resultsCount = fuzzyResponse.count;
                this.isFuzzySearch = fuzzyResponse.fuzzy_search;
                this.loading = false;
                this.render();
                return;
            }

            // Fall back to Tautulli search if no local results
            const data: TautulliSearchData = await this.api.search(trimmedQuery, 50);
            this.results = data.results || [];
            this.resultsCount = data.results_count || 0;
            this.isFuzzySearch = false;
            this.loading = false;
            this.render();
        } catch (err) {
            // If fuzzy search fails, try Tautulli as fallback
            try {
                const data: TautulliSearchData = await this.api.search(trimmedQuery, 50);
                this.results = data.results || [];
                this.resultsCount = data.results_count || 0;
                this.isFuzzySearch = false;
                this.loading = false;
                this.render();
            } catch (fallbackErr) {
                logger.error('Search failed:', fallbackErr);
                this.loading = false;
                this.error = 'Search failed. Please try again.';
                this.toastManager?.error('Search failed', 'Error', 3000);
                this.render();
            }
        }
    }

    /**
     * Map fuzzy search results to TautulliSearchResult format
     */
    private mapFuzzyResultsToTautulli(fuzzyResults: FuzzySearchResult[]): TautulliSearchResult[] {
        return fuzzyResults.map(result => ({
            type: result.media_type,
            rating_key: result.id,
            title: result.title,
            year: result.year || 0,
            thumb: result.thumb || '',
            score: result.score,
            library: '',
            library_id: 0,
            media_type: result.media_type,
            summary: '',
            grandparent_title: result.grandparent_title,
            parent_title: result.parent_title,
        }));
    }

    /**
     * Set search query
     */
    setQuery(query: string): void {
        this.query = query;
        this.debouncedSearch();
    }

    /**
     * Set media type filter
     */
    setFilterType(type: string): void {
        this.filterType = type;
        this.render();
    }

    /**
     * Get filtered results
     */
    private getFilteredResults(): TautulliSearchResult[] {
        if (this.filterType === 'all') {
            return this.results;
        }
        return this.results.filter(r => r.media_type === this.filterType || r.type === this.filterType);
    }

    /**
     * Clear search
     */
    clear(): void {
        this.query = '';
        this.results = [];
        this.resultsCount = 0;
        this.error = null;
        this.render();
    }

    /**
     * Main render method
     */
    private render(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        const filteredResults = this.getFilteredResults();
        const mediaTypes = this.getUniqueMediaTypes();

        container.innerHTML = `
            <div class="search-manager">
                <div class="search-header">
                    <h3>Search Library</h3>
                </div>

                <div class="search-form">
                    <div class="search-input-wrapper">
                        <svg class="search-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <circle cx="11" cy="11" r="8"/>
                            <path d="M21 21l-4.35-4.35"/>
                        </svg>
                        <input type="text"
                               id="search-input"
                               class="search-input"
                               placeholder="Search movies, shows, music..."
                               value="${this.escapeHtml(this.query)}"
                               autocomplete="off">
                        ${this.query ? `
                            <button class="btn-clear" id="search-clear" title="Clear">
                                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <line x1="18" y1="6" x2="6" y2="18"/>
                                    <line x1="6" y1="6" x2="18" y2="18"/>
                                </svg>
                            </button>
                        ` : ''}
                    </div>

                    ${mediaTypes.length > 1 ? `
                        <div class="search-filters">
                            <button class="filter-btn ${this.filterType === 'all' ? 'active' : ''}"
                                    data-filter="all">All</button>
                            ${mediaTypes.map(type => `
                                <button class="filter-btn ${this.filterType === type ? 'active' : ''}"
                                        data-filter="${type}">
                                    ${this.getMediaTypeLabel(type)}
                                </button>
                            `).join('')}
                        </div>
                    ` : ''}
                </div>

                ${this.renderContent(filteredResults)}
            </div>
        `;

        this.attachEventListeners();
    }

    /**
     * Render content area
     */
    private renderContent(filteredResults: TautulliSearchResult[]): string {
        if (this.loading) {
            return `
                <div class="search-content loading">
                    <div class="spinner"></div>
                    <p>Searching...</p>
                </div>
            `;
        }

        if (this.error) {
            return `
                <div class="search-content error">
                    <p class="error-message">${this.escapeHtml(this.error)}</p>
                    <button class="btn-retry" id="search-retry">Retry</button>
                </div>
            `;
        }

        if (!this.query.trim()) {
            return `
                <div class="search-content empty">
                    <div class="empty-icon">
                        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                            <circle cx="11" cy="11" r="8"/>
                            <path d="M21 21l-4.35-4.35"/>
                        </svg>
                    </div>
                    <p>Enter a search term to find media</p>
                </div>
            `;
        }

        if (filteredResults.length === 0) {
            return `
                <div class="search-content empty" data-testid="no-results">
                    <p>No results found for "${this.escapeHtml(this.query)}"</p>
                    ${this.filterType !== 'all' ? `
                        <p class="hint">Try removing the filter or using different keywords</p>
                    ` : ''}
                </div>
            `;
        }

        return `
            <div class="search-content" data-testid="search-results">
                <div class="results-header">
                    <span class="results-count">${this.resultsCount} result${this.resultsCount !== 1 ? 's' : ''}</span>
                    ${this.isFuzzySearch ? '<span class="fuzzy-badge" title="Using fuzzy string matching">fuzzy</span>' : ''}
                    ${filteredResults.length !== this.resultsCount ? `
                        <span class="filtered-count">(showing ${filteredResults.length})</span>
                    ` : ''}
                </div>
                <div class="results-list">
                    ${filteredResults.map(result => this.renderResultItem(result)).join('')}
                </div>
            </div>
        `;
    }

    /**
     * Render a single search result
     */
    private renderResultItem(result: TautulliSearchResult): string {
        const title = this.getFullTitle(result);
        const typeIcon = this.getMediaTypeIcon(result.media_type || result.type);

        return `
            <div class="result-item" data-rating-key="${result.rating_key}" data-testid="search-result">
                <div class="result-thumb">
                    ${result.thumb ? `
                        <img src="${this.escapeHtml(result.thumb)}" alt="${this.escapeHtml(result.title)}"
                             class="result-thumb-img">
                        <div class="thumb-placeholder" style="display: none;">
                            ${typeIcon}
                        </div>
                    ` : `
                        <div class="thumb-placeholder">
                            ${typeIcon}
                        </div>
                    `}
                </div>
                <div class="result-info">
                    <div class="result-title">${this.escapeHtml(title)}</div>
                    <div class="result-meta">
                        <span class="result-type">${this.getMediaTypeLabel(result.media_type || result.type)}</span>
                        ${result.year ? `<span class="result-year">${result.year}</span>` : ''}
                        ${result.library ? `<span class="result-library">${this.escapeHtml(result.library)}</span>` : ''}
                    </div>
                    ${result.summary ? `
                        <p class="result-summary">${this.escapeHtml(this.truncateSummary(result.summary, 150))}</p>
                    ` : ''}
                </div>
                <div class="result-actions">
                    <button class="btn-view" data-rating-key="${result.rating_key}" title="View details">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M9 18l6-6-6-6"/>
                        </svg>
                    </button>
                </div>
            </div>
        `;
    }

    /**
     * Get unique media types from results
     */
    private getUniqueMediaTypes(): string[] {
        const types = new Set<string>();
        this.results.forEach(r => {
            if (r.media_type) types.add(r.media_type);
            else if (r.type) types.add(r.type);
        });
        return Array.from(types).sort();
    }

    /**
     * Get full title including parent titles
     */
    private getFullTitle(result: TautulliSearchResult): string {
        const parts = [];
        if (result.grandparent_title) parts.push(result.grandparent_title);
        if (result.parent_title) parts.push(result.parent_title);
        parts.push(result.title);
        return parts.join(' - ');
    }

    /**
     * Get label for media type
     */
    private getMediaTypeLabel(type: string): string {
        const labels: Record<string, string> = {
            movie: 'Movie',
            show: 'TV Show',
            season: 'Season',
            episode: 'Episode',
            artist: 'Artist',
            album: 'Album',
            track: 'Track',
            photo: 'Photo'
        };
        return labels[type] || type.charAt(0).toUpperCase() + type.slice(1);
    }

    /**
     * Get icon for media type
     */
    private getMediaTypeIcon(type: string): string {
        const icons: Record<string, string> = {
            movie: '🎬',
            show: '📺',
            season: '📁',
            episode: '📄',
            artist: '🎤',
            album: '💿',
            track: '🎵',
            photo: '📷'
        };
        return icons[type] || '📄';
    }

    /**
     * Truncate summary text
     */
    private truncateSummary(text: string, maxLength: number): string {
        if (text.length <= maxLength) return text;
        return text.substring(0, maxLength).trim() + '...';
    }

    /**
     * Attach event listeners
     */
    private attachEventListeners(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        // Search input
        const input = container.querySelector('#search-input') as HTMLInputElement;
        if (input) {
            input.addEventListener('input', () => {
                this.query = input.value;
                this.debouncedSearch();
            });

            // Focus input
            input.focus();
        }

        // Clear button
        const clearBtn = container.querySelector('#search-clear');
        if (clearBtn) {
            clearBtn.addEventListener('click', () => {
                this.clear();
            });
        }

        // Retry button
        const retryBtn = container.querySelector('#search-retry');
        if (retryBtn) {
            retryBtn.addEventListener('click', () => {
                this.performSearch();
            });
        }

        // Filter buttons
        container.querySelectorAll('.filter-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                const filter = btn.getAttribute('data-filter') || 'all';
                this.setFilterType(filter);
            });
        });

        // Result items
        container.querySelectorAll('.result-item').forEach(item => {
            item.addEventListener('click', () => {
                const ratingKey = item.getAttribute('data-rating-key');
                if (ratingKey && this.onSelectResult) {
                    this.onSelectResult(ratingKey);
                }
            });
        });

        // View buttons
        container.querySelectorAll('.btn-view').forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.stopPropagation();
                const ratingKey = btn.getAttribute('data-rating-key');
                if (ratingKey && this.onSelectResult) {
                    this.onSelectResult(ratingKey);
                }
            });
        });

        // Image error handling (CSP-compliant - replaces inline onerror)
        container.querySelectorAll('.result-thumb-img').forEach(img => {
            img.addEventListener('error', () => {
                (img as HTMLElement).style.display = 'none';
                const placeholder = (img as HTMLElement).nextElementSibling as HTMLElement;
                if (placeholder) placeholder.style.display = 'flex';
            });
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
        if (this.searchTimeout) {
            clearTimeout(this.searchTimeout);
        }
        this.initialized = false;
        this.results = [];
    }
}
