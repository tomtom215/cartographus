// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Library Content Manager
 *
 * Manages display of:
 * - Collections: Browse and filter Plex collections
 * - Playlists: Browse playlists with type filtering
 */

import { API } from '../lib/api';
import { createLogger } from '../lib/logger';
import type { ToastManager } from '../lib/toast';

const logger = createLogger('LibraryContentManager');
import type {
    TautulliCollectionItem,
    TautulliCollectionsTableData,
    TautulliPlaylistItem,
    TautulliPlaylistsTableData,
    TautulliLibraryNameItem
} from '../lib/types';

export type ContentViewMode = 'collections' | 'playlists';

interface ContentManagerOptions {
    api: API;
    containerId: string;
    toastManager?: ToastManager;
}

export class LibraryContentManager {
    private api: API;
    private containerId: string;
    private toastManager: ToastManager | null = null;
    private initialized = false;
    private currentMode: ContentViewMode = 'collections';

    // Collections state
    private collections: TautulliCollectionItem[] = [];
    private collectionsTotal = 0;
    private collectionsPage = 0;
    private collectionsPageSize = 25;
    private collectionsSectionId: number | undefined = undefined;

    // Playlists state
    private playlists: TautulliPlaylistItem[] = [];
    private playlistsTotal = 0;
    private playlistsPage = 0;
    private playlistsPageSize = 25;
    private playlistsFilter: string = 'all'; // 'all', 'video', 'audio', 'photo'

    // Library list for filtering
    private libraries: TautulliLibraryNameItem[] = [];

    constructor(options: ContentManagerOptions) {
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

        try {
            // Load libraries for filtering
            this.libraries = await this.api.getTautulliLibraryNames();

            // Load initial data based on mode
            await this.loadCurrentView();

            this.initialized = true;
        } catch (error) {
            logger.error('Failed to initialize LibraryContentManager:', error);
            this.showError('Failed to initialize content manager');
        }
    }

    /**
     * Switch between collections and playlists view
     */
    async setViewMode(mode: ContentViewMode): Promise<void> {
        if (this.currentMode === mode) return;
        this.currentMode = mode;
        await this.loadCurrentView();
    }

    /**
     * Get current view mode
     */
    getViewMode(): ContentViewMode {
        return this.currentMode;
    }

    /**
     * Load current view data
     */
    private async loadCurrentView(): Promise<void> {
        if (this.currentMode === 'collections') {
            await this.loadCollections();
        } else {
            await this.loadPlaylists();
        }
    }

    /**
     * Load collections data
     */
    async loadCollections(): Promise<void> {
        try {
            const start = this.collectionsPage * this.collectionsPageSize;
            const data: TautulliCollectionsTableData = await this.api.getCollectionsTable(
                this.collectionsSectionId,
                start,
                this.collectionsPageSize
            );

            this.collections = data.data;
            this.collectionsTotal = data.recordsTotal;

            this.renderCollections();
        } catch (error) {
            logger.error('Failed to load collections:', error);
            this.showError('Failed to load collections');
        }
    }

    /**
     * Load playlists data
     */
    async loadPlaylists(): Promise<void> {
        try {
            const start = this.playlistsPage * this.playlistsPageSize;
            const data: TautulliPlaylistsTableData = await this.api.getPlaylistsTable(
                start,
                this.playlistsPageSize
            );

            // Apply client-side filter
            let filteredData = data.data;
            if (this.playlistsFilter !== 'all') {
                filteredData = data.data.filter(p => p.playlist_type === this.playlistsFilter);
            }

            this.playlists = filteredData;
            this.playlistsTotal = this.playlistsFilter === 'all'
                ? data.recordsTotal
                : filteredData.length;

            this.renderPlaylists();
        } catch (error) {
            logger.error('Failed to load playlists:', error);
            this.showError('Failed to load playlists');
        }
    }

    /**
     * Filter collections by library section
     */
    async filterCollectionsByLibrary(sectionId?: number): Promise<void> {
        this.collectionsSectionId = sectionId;
        this.collectionsPage = 0;
        await this.loadCollections();
    }

    /**
     * Filter playlists by type
     */
    async filterPlaylistsByType(type: string): Promise<void> {
        this.playlistsFilter = type;
        this.playlistsPage = 0;
        await this.loadPlaylists();
    }

    /**
     * Navigate to collections page
     */
    async goToCollectionsPage(page: number): Promise<void> {
        const maxPage = Math.ceil(this.collectionsTotal / this.collectionsPageSize) - 1;
        this.collectionsPage = Math.max(0, Math.min(page, maxPage));
        await this.loadCollections();
    }

    /**
     * Navigate to playlists page
     */
    async goToPlaylistsPage(page: number): Promise<void> {
        const maxPage = Math.ceil(this.playlistsTotal / this.playlistsPageSize) - 1;
        this.playlistsPage = Math.max(0, Math.min(page, maxPage));
        await this.loadPlaylists();
    }

    /**
     * Render collections view
     */
    private renderCollections(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        const totalPages = Math.ceil(this.collectionsTotal / this.collectionsPageSize);

        container.innerHTML = `
            <div class="library-content-view" data-testid="collections-view">
                <div class="content-header">
                    <div class="view-tabs">
                        <button class="tab-btn active" data-mode="collections">Collections</button>
                        <button class="tab-btn" data-mode="playlists">Playlists</button>
                    </div>
                    <div class="content-filters">
                        <select id="collections-library-filter" class="filter-select" aria-label="Filter by library">
                            <option value="">All Libraries</option>
                            ${this.libraries.map(lib =>
                                `<option value="${lib.section_id}" ${this.collectionsSectionId === lib.section_id ? 'selected' : ''}>
                                    ${this.escapeHtml(lib.section_name)}
                                </option>`
                            ).join('')}
                        </select>
                    </div>
                </div>

                <div class="content-stats">
                    <span class="stat-item">Total: <strong>${this.collectionsTotal}</strong> collections</span>
                    <span class="stat-item">Showing: <strong>${this.collections.length}</strong></span>
                </div>

                ${this.collections.length === 0 ? `
                    <div class="empty-state" data-testid="collections-empty">
                        <p>No collections found</p>
                        ${this.collectionsSectionId ? '<p class="empty-hint">Try selecting a different library or "All Libraries"</p>' : ''}
                    </div>
                ` : `
                    <div class="content-grid collections-grid" data-testid="collections-grid">
                        ${this.collections.map(collection => this.renderCollectionCard(collection)).join('')}
                    </div>
                `}

                ${totalPages > 1 ? this.renderPagination(this.collectionsPage, totalPages, 'collections') : ''}
            </div>
        `;

        this.attachEventListeners();
    }

    /**
     * Render single collection card
     */
    private renderCollectionCard(collection: TautulliCollectionItem): string {
        const yearDisplay = collection.min_year
            ? (collection.min_year === collection.max_year
                ? collection.min_year.toString()
                : `${collection.min_year} - ${collection.max_year}`)
            : '';

        return `
            <div class="content-card collection-card" data-rating-key="${collection.rating_key}" data-testid="collection-card">
                <div class="card-thumbnail">
                    ${collection.thumb
                        ? `<img src="${this.escapeHtml(collection.thumb)}" alt="${this.escapeHtml(collection.title)}" loading="lazy">`
                        : '<div class="placeholder-thumb"><span>No Image</span></div>'
                    }
                </div>
                <div class="card-info">
                    <h4 class="card-title" title="${this.escapeHtml(collection.title)}">${this.escapeHtml(collection.title)}</h4>
                    <div class="card-meta">
                        <span class="meta-item item-count">${collection.child_count} items</span>
                        ${yearDisplay ? `<span class="meta-item year-range">${yearDisplay}</span>` : ''}
                        ${collection.content_rating ? `<span class="meta-item content-rating">${this.escapeHtml(collection.content_rating)}</span>` : ''}
                    </div>
                    ${collection.summary ? `<p class="card-summary">${this.escapeHtml(this.truncate(collection.summary, 100))}</p>` : ''}
                    ${collection.labels && collection.labels.length > 0 ? `
                        <div class="card-labels">
                            ${collection.labels.slice(0, 3).map(label =>
                                `<span class="label-tag">${this.escapeHtml(label)}</span>`
                            ).join('')}
                            ${collection.labels.length > 3 ? `<span class="label-more">+${collection.labels.length - 3}</span>` : ''}
                        </div>
                    ` : ''}
                </div>
            </div>
        `;
    }

    /**
     * Render playlists view
     */
    private renderPlaylists(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        const totalPages = Math.ceil(this.playlistsTotal / this.playlistsPageSize);

        container.innerHTML = `
            <div class="library-content-view" data-testid="playlists-view">
                <div class="content-header">
                    <div class="view-tabs">
                        <button class="tab-btn" data-mode="collections">Collections</button>
                        <button class="tab-btn active" data-mode="playlists">Playlists</button>
                    </div>
                    <div class="content-filters">
                        <select id="playlists-type-filter" class="filter-select" aria-label="Filter by type">
                            <option value="all" ${this.playlistsFilter === 'all' ? 'selected' : ''}>All Types</option>
                            <option value="video" ${this.playlistsFilter === 'video' ? 'selected' : ''}>Video</option>
                            <option value="audio" ${this.playlistsFilter === 'audio' ? 'selected' : ''}>Audio</option>
                            <option value="photo" ${this.playlistsFilter === 'photo' ? 'selected' : ''}>Photo</option>
                        </select>
                    </div>
                </div>

                <div class="content-stats">
                    <span class="stat-item">Total: <strong>${this.playlistsTotal}</strong> playlists</span>
                    <span class="stat-item">Showing: <strong>${this.playlists.length}</strong></span>
                </div>

                ${this.playlists.length === 0 ? `
                    <div class="empty-state" data-testid="playlists-empty">
                        <p>No playlists found</p>
                        ${this.playlistsFilter !== 'all' ? '<p class="empty-hint">Try selecting "All Types"</p>' : ''}
                    </div>
                ` : `
                    <div class="content-grid playlists-grid" data-testid="playlists-grid">
                        ${this.playlists.map(playlist => this.renderPlaylistCard(playlist)).join('')}
                    </div>
                `}

                ${totalPages > 1 ? this.renderPagination(this.playlistsPage, totalPages, 'playlists') : ''}
            </div>
        `;

        this.attachEventListeners();
    }

    /**
     * Render single playlist card
     */
    private renderPlaylistCard(playlist: TautulliPlaylistItem): string {
        const typeIcon = this.getPlaylistTypeIcon(playlist.playlist_type);
        const durationDisplay = playlist.duration ? this.formatDuration(playlist.duration) : '';

        return `
            <div class="content-card playlist-card" data-rating-key="${playlist.rating_key}" data-testid="playlist-card">
                <div class="card-thumbnail">
                    ${playlist.thumb || playlist.composite
                        ? `<img src="${this.escapeHtml(playlist.thumb || playlist.composite || '')}" alt="${this.escapeHtml(playlist.title)}" loading="lazy">`
                        : '<div class="placeholder-thumb"><span>No Image</span></div>'
                    }
                    <span class="type-badge ${playlist.playlist_type}">${typeIcon}</span>
                </div>
                <div class="card-info">
                    <h4 class="card-title" title="${this.escapeHtml(playlist.title)}">${this.escapeHtml(playlist.title)}</h4>
                    <div class="card-meta">
                        <span class="meta-item item-count">${playlist.leaf_count} items</span>
                        ${durationDisplay ? `<span class="meta-item duration">${durationDisplay}</span>` : ''}
                        ${playlist.smart ? '<span class="meta-item smart-badge">Smart</span>' : ''}
                    </div>
                    ${playlist.summary ? `<p class="card-summary">${this.escapeHtml(this.truncate(playlist.summary, 100))}</p>` : ''}
                    ${playlist.username ? `<p class="card-owner">By: ${this.escapeHtml(playlist.username)}</p>` : ''}
                </div>
            </div>
        `;
    }

    /**
     * Render pagination controls
     */
    private renderPagination(currentPage: number, totalPages: number, type: 'collections' | 'playlists'): string {
        const pages: string[] = [];
        const maxVisible = 5;
        let startPage = Math.max(0, currentPage - Math.floor(maxVisible / 2));
        const endPage = Math.min(totalPages - 1, startPage + maxVisible - 1);

        if (endPage - startPage < maxVisible - 1) {
            startPage = Math.max(0, endPage - maxVisible + 1);
        }

        for (let i = startPage; i <= endPage; i++) {
            pages.push(`
                <button class="page-btn ${i === currentPage ? 'active' : ''}"
                        data-page="${i}"
                        data-type="${type}"
                        ${i === currentPage ? 'aria-current="page"' : ''}>
                    ${i + 1}
                </button>
            `);
        }

        return `
            <div class="pagination" data-testid="${type}-pagination">
                <button class="page-btn nav-btn" data-page="${currentPage - 1}" data-type="${type}"
                        ${currentPage === 0 ? 'disabled' : ''} aria-label="Previous page">
                    &laquo;
                </button>
                ${startPage > 0 ? '<span class="page-ellipsis">...</span>' : ''}
                ${pages.join('')}
                ${endPage < totalPages - 1 ? '<span class="page-ellipsis">...</span>' : ''}
                <button class="page-btn nav-btn" data-page="${currentPage + 1}" data-type="${type}"
                        ${currentPage >= totalPages - 1 ? 'disabled' : ''} aria-label="Next page">
                    &raquo;
                </button>
            </div>
        `;
    }

    /**
     * Attach event listeners
     */
    private attachEventListeners(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        // View mode tabs
        container.querySelectorAll('.tab-btn').forEach(btn => {
            btn.addEventListener('click', async (e) => {
                const mode = (e.currentTarget as HTMLElement).getAttribute('data-mode') as ContentViewMode;
                if (mode) {
                    await this.setViewMode(mode);
                }
            });
        });

        // Library filter for collections
        const libraryFilter = container.querySelector('#collections-library-filter');
        if (libraryFilter) {
            libraryFilter.addEventListener('change', async (e) => {
                const value = (e.target as HTMLSelectElement).value;
                await this.filterCollectionsByLibrary(value ? parseInt(value, 10) : undefined);
            });
        }

        // Type filter for playlists
        const typeFilter = container.querySelector('#playlists-type-filter');
        if (typeFilter) {
            typeFilter.addEventListener('change', async (e) => {
                const value = (e.target as HTMLSelectElement).value;
                await this.filterPlaylistsByType(value);
            });
        }

        // Pagination buttons
        container.querySelectorAll('.page-btn:not([disabled])').forEach(btn => {
            btn.addEventListener('click', async (e) => {
                const page = parseInt((e.currentTarget as HTMLElement).getAttribute('data-page') || '0', 10);
                const type = (e.currentTarget as HTMLElement).getAttribute('data-type');
                if (type === 'collections') {
                    await this.goToCollectionsPage(page);
                } else if (type === 'playlists') {
                    await this.goToPlaylistsPage(page);
                }
            });
        });
    }

    /**
     * Get icon for playlist type
     */
    private getPlaylistTypeIcon(type: string): string {
        switch (type) {
            case 'video':
                return '&#127909;'; // Film
            case 'audio':
                return '&#127925;'; // Music
            case 'photo':
                return '&#128247;'; // Camera
            default:
                return '&#128196;'; // Document
        }
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
     * Truncate text with ellipsis
     */
    private truncate(text: string, maxLength: number): string {
        if (text.length <= maxLength) return text;
        return text.substring(0, maxLength - 3) + '...';
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
     * Show error toast
     */
    private showError(message: string): void {
        if (this.toastManager) {
            this.toastManager.error(message, 'Error', 5000);
        }
    }

    /**
     * Clean up resources
     */
    destroy(): void {
        this.initialized = false;
        this.collections = [];
        this.playlists = [];
    }
}
