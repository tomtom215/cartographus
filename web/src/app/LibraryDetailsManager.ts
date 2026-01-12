// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Library Details Manager
 *
 * Enhanced library details display integrating 5 Tautulli endpoints:
 * - get_libraries_table: Full library table with usage stats
 * - get_library_user_stats: User activity per library
 * - get_library_media_info: Media quality and technical specs
 * - get_library_watch_time_stats: Time-based watch statistics
 */

import { API } from '../lib/api';
import type { ToastManager } from '../lib/toast';
import type {
    TautulliLibraryTableItem,
    TautulliLibraryTableData,
    TautulliLibraryUserStatsItem,
    TautulliLibraryMediaInfoItem,
    TautulliLibraryMediaInfoData,
    TautulliLibraryWatchTimeStatsItem
} from '../lib/types';
import { createLogger } from '../lib/logger';

const logger = createLogger('LibraryDetailsManager');

interface LibraryDetailsOptions {
    api: API;
    containerId: string;
    toastManager?: ToastManager;
}

type ViewTab = 'overview' | 'users' | 'media' | 'watchtime';

export class LibraryDetailsManager {
    private api: API;
    private containerId: string;
    private toastManager: ToastManager | null = null;
    private initialized = false;

    // Data state
    private libraries: TautulliLibraryTableItem[] = [];
    private selectedLibraryId: number | null = null;
    private currentTab: ViewTab = 'overview';

    // Library-specific data
    private userStats: TautulliLibraryUserStatsItem[] = [];
    private mediaInfo: TautulliLibraryMediaInfoData | null = null;
    private watchTimeStats: TautulliLibraryWatchTimeStatsItem[] = [];

    // Pagination for media info
    private mediaPage = 0;
    private mediaPageSize = 25;
    private mediaSortColumn = 'added_at';
    private mediaSortDir: 'asc' | 'desc' = 'desc';

    // Bound event handler for event delegation (prevents memory leaks)
    private readonly boundHandleContainerClick: (e: Event) => void;

    constructor(options: LibraryDetailsOptions) {
        this.api = options.api;
        this.containerId = options.containerId;
        if (options.toastManager) {
            this.toastManager = options.toastManager;
        }

        // Bind event handler once to ensure consistent reference for add/remove
        this.boundHandleContainerClick = this.handleContainerClick.bind(this);
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
            // Attach container event listener once during initialization
            this.attachContainerEventListener();
            await this.loadLibraries();
            this.initialized = true;
        } catch (error) {
            logger.error('Failed to initialize LibraryDetailsManager:', error);
            this.showError('Failed to load libraries');
        }
    }

    /**
     * Attach container event listener once (event delegation pattern)
     * This prevents memory leaks from adding listeners on every render()
     */
    private attachContainerEventListener(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        container.addEventListener('click', this.boundHandleContainerClick);
    }

    /**
     * Handle all click events via event delegation
     * Single listener handles all interactive elements based on target
     */
    private handleContainerClick(e: Event): void {
        const target = e.target as HTMLElement;
        if (!target) return;

        // Find closest interactive element
        const libraryCard = target.closest('.library-card') as HTMLElement;
        const backBtn = target.closest('#library-back');
        const tabBtn = target.closest('.tab-btn') as HTMLElement;
        const sortableHeader = target.closest('th.sortable') as HTMLElement;
        const pageBtn = target.closest('.pagination .page-btn:not([disabled])') as HTMLElement;

        // Library card click - navigate to library details
        if (libraryCard) {
            const sectionId = parseInt(libraryCard.getAttribute('data-section-id') || '0', 10);
            if (sectionId) {
                this.selectLibrary(sectionId);
            }
            return;
        }

        // Back button click - return to library list
        if (backBtn) {
            this.selectedLibraryId = null;
            this.render();
            return;
        }

        // Tab button click - switch tabs
        if (tabBtn) {
            const tab = tabBtn.getAttribute('data-tab') as ViewTab;
            if (tab) {
                this.setTab(tab);
            }
            return;
        }

        // Sortable header click - sort media table
        if (sortableHeader) {
            const column = sortableHeader.getAttribute('data-sort');
            if (column) {
                this.sortMedia(column);
            }
            return;
        }

        // Pagination button click - navigate pages
        if (pageBtn) {
            const page = parseInt(pageBtn.getAttribute('data-page') || '0', 10);
            this.goToMediaPage(page);
            return;
        }
    }

    /**
     * Load all libraries
     */
    private async loadLibraries(): Promise<void> {
        try {
            const data: TautulliLibraryTableData = await this.api.getLibrariesTable();
            this.libraries = data.data;
            this.render();
        } catch (error) {
            logger.error('Failed to load libraries:', error);
            throw error;
        }
    }

    /**
     * Select a library and load its details
     */
    async selectLibrary(sectionId: number): Promise<void> {
        this.selectedLibraryId = sectionId;
        this.currentTab = 'overview';
        await this.loadLibraryDetails();
    }

    /**
     * Load all details for selected library
     */
    private async loadLibraryDetails(): Promise<void> {
        if (!this.selectedLibraryId) return;

        try {
            // Load all data in parallel
            const [userStats, mediaInfo, watchTimeStats] = await Promise.all([
                this.api.getLibraryUserStats(this.selectedLibraryId),
                this.api.getLibraryMediaInfo(
                    this.selectedLibraryId,
                    this.mediaPage * this.mediaPageSize,
                    this.mediaPageSize,
                    this.mediaSortColumn,
                    this.mediaSortDir
                ),
                this.api.getLibraryWatchTimeStats(this.selectedLibraryId)
            ]);

            this.userStats = userStats;
            this.mediaInfo = mediaInfo;
            this.watchTimeStats = watchTimeStats;

            this.render();
        } catch (error) {
            logger.error('Failed to load library details:', error);
            this.showError('Failed to load library details');
        }
    }

    /**
     * Switch tab view
     */
    setTab(tab: ViewTab): void {
        this.currentTab = tab;
        this.render();
    }

    /**
     * Sort media by column
     */
    async sortMedia(column: string): Promise<void> {
        if (this.mediaSortColumn === column) {
            this.mediaSortDir = this.mediaSortDir === 'asc' ? 'desc' : 'asc';
        } else {
            this.mediaSortColumn = column;
            this.mediaSortDir = 'desc';
        }
        this.mediaPage = 0;
        await this.loadMediaInfo();
    }

    /**
     * Navigate media pages
     */
    async goToMediaPage(page: number): Promise<void> {
        if (!this.mediaInfo) return;
        const maxPage = Math.ceil(this.mediaInfo.recordsTotal / this.mediaPageSize) - 1;
        this.mediaPage = Math.max(0, Math.min(page, maxPage));
        await this.loadMediaInfo();
    }

    /**
     * Load media info only
     */
    private async loadMediaInfo(): Promise<void> {
        if (!this.selectedLibraryId) return;

        try {
            this.mediaInfo = await this.api.getLibraryMediaInfo(
                this.selectedLibraryId,
                this.mediaPage * this.mediaPageSize,
                this.mediaPageSize,
                this.mediaSortColumn,
                this.mediaSortDir
            );
            this.render();
        } catch (error) {
            logger.error('Failed to load media info:', error);
        }
    }

    /**
     * Main render method
     * Note: Event listeners are NOT attached here - they are attached once
     * in init() using event delegation to prevent memory leaks.
     */
    private render(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        if (!this.selectedLibraryId) {
            this.renderLibraryList(container);
        } else {
            this.renderLibraryDetails(container);
        }
        // Event delegation is set up once in init() - no per-render listener attachment
    }

    /**
     * Render library list
     */
    private renderLibraryList(container: HTMLElement): void {
        const totalPlays = this.libraries.reduce((sum, lib) => sum + lib.plays, 0);
        const totalDuration = this.libraries.reduce((sum, lib) => sum + lib.duration, 0);

        container.innerHTML = `
            <div class="library-details-view" data-testid="library-list">
                <div class="library-header">
                    <h3>Libraries</h3>
                    <div class="library-summary">
                        <span>${this.libraries.length} libraries</span>
                        <span>${totalPlays.toLocaleString()} total plays</span>
                        <span>${this.formatDuration(totalDuration)} watched</span>
                    </div>
                </div>

                <div class="library-grid">
                    ${this.libraries.map(lib => this.renderLibraryCard(lib)).join('')}
                </div>
            </div>
        `;
    }

    /**
     * Render individual library card
     */
    private renderLibraryCard(lib: TautulliLibraryTableItem): string {
        const typeIcon = this.getLibraryTypeIcon(lib.section_type);
        const isActive = lib.is_active === 1;

        return `
            <div class="library-card ${!isActive ? 'inactive' : ''}"
                 data-section-id="${lib.section_id}"
                 data-testid="library-card">
                <div class="library-card-header">
                    <span class="library-type-icon">${typeIcon}</span>
                    <h4>${this.escapeHtml(lib.section_name)}</h4>
                    ${!isActive ? '<span class="inactive-badge">Inactive</span>' : ''}
                </div>
                <div class="library-card-stats">
                    <div class="stat">
                        <span class="stat-value">${lib.count.toLocaleString()}</span>
                        <span class="stat-label">Items</span>
                    </div>
                    <div class="stat">
                        <span class="stat-value">${lib.plays.toLocaleString()}</span>
                        <span class="stat-label">Plays</span>
                    </div>
                    <div class="stat">
                        <span class="stat-value">${this.formatDuration(lib.duration)}</span>
                        <span class="stat-label">Watch Time</span>
                    </div>
                </div>
                ${lib.last_accessed ? `
                    <div class="library-card-footer">
                        Last accessed: ${this.formatDate(lib.last_accessed)}
                    </div>
                ` : ''}
            </div>
        `;
    }

    /**
     * Render library details view
     */
    private renderLibraryDetails(container: HTMLElement): void {
        const library = this.libraries.find(lib => lib.section_id === this.selectedLibraryId);
        if (!library) return;

        container.innerHTML = `
            <div class="library-details-view" data-testid="library-details">
                <div class="library-detail-header">
                    <button class="btn-back" id="library-back" aria-label="Back to libraries">
                        &larr; Back
                    </button>
                    <div class="library-title">
                        <span class="library-type-icon">${this.getLibraryTypeIcon(library.section_type)}</span>
                        <h3>${this.escapeHtml(library.section_name)}</h3>
                    </div>
                </div>

                <div class="library-tabs">
                    ${this.renderTab('overview', 'Overview')}
                    ${this.renderTab('users', 'Users')}
                    ${this.renderTab('media', 'Media')}
                    ${this.renderTab('watchtime', 'Watch Time')}
                </div>

                <div class="library-tab-content">
                    ${this.renderCurrentTab(library)}
                </div>
            </div>
        `;
    }

    /**
     * Render tab button
     */
    private renderTab(tab: ViewTab, label: string): string {
        return `
            <button class="tab-btn ${this.currentTab === tab ? 'active' : ''}"
                    data-tab="${tab}">
                ${label}
            </button>
        `;
    }

    /**
     * Render current tab content
     */
    private renderCurrentTab(library: TautulliLibraryTableItem): string {
        switch (this.currentTab) {
            case 'overview':
                return this.renderOverviewTab(library);
            case 'users':
                return this.renderUsersTab();
            case 'media':
                return this.renderMediaTab();
            case 'watchtime':
                return this.renderWatchTimeTab();
            default:
                return '';
        }
    }

    /**
     * Render overview tab
     */
    private renderOverviewTab(library: TautulliLibraryTableItem): string {
        return `
            <div class="overview-tab" data-testid="overview-tab">
                <div class="overview-grid">
                    <div class="overview-card">
                        <h4>Library Statistics</h4>
                        <div class="overview-stats">
                            <div class="stat-row">
                                <span>Type</span>
                                <span>${library.section_type}</span>
                            </div>
                            <div class="stat-row">
                                <span>Items</span>
                                <span>${library.count.toLocaleString()}</span>
                            </div>
                            ${library.parent_count ? `
                                <div class="stat-row">
                                    <span>Parent Count</span>
                                    <span>${library.parent_count.toLocaleString()}</span>
                                </div>
                            ` : ''}
                            ${library.child_count ? `
                                <div class="stat-row">
                                    <span>Child Count</span>
                                    <span>${library.child_count.toLocaleString()}</span>
                                </div>
                            ` : ''}
                            <div class="stat-row">
                                <span>Total Plays</span>
                                <span>${library.plays.toLocaleString()}</span>
                            </div>
                            <div class="stat-row">
                                <span>Total Watch Time</span>
                                <span>${this.formatDuration(library.duration)}</span>
                            </div>
                        </div>
                    </div>

                    <div class="overview-card">
                        <h4>Settings</h4>
                        <div class="overview-stats">
                            <div class="stat-row">
                                <span>Status</span>
                                <span class="${library.is_active ? 'status-active' : 'status-inactive'}">
                                    ${library.is_active ? 'Active' : 'Inactive'}
                                </span>
                            </div>
                            <div class="stat-row">
                                <span>Notifications</span>
                                <span>${library.do_notify ? 'Enabled' : 'Disabled'}</span>
                            </div>
                            <div class="stat-row">
                                <span>History</span>
                                <span>${library.keep_history ? 'Enabled' : 'Disabled'}</span>
                            </div>
                            <div class="stat-row">
                                <span>Agent</span>
                                <span>${this.escapeHtml(library.agent)}</span>
                            </div>
                        </div>
                    </div>

                    <div class="overview-card">
                        <h4>Top Users</h4>
                        ${this.userStats.length > 0 ? `
                            <div class="top-users-list">
                                ${this.userStats.slice(0, 5).map((user, idx) => `
                                    <div class="top-user">
                                        <span class="rank">#${idx + 1}</span>
                                        <span class="name">${this.escapeHtml(user.friendly_name || user.username)}</span>
                                        <span class="plays">${user.total_plays} plays</span>
                                    </div>
                                `).join('')}
                            </div>
                        ` : '<p class="empty-text">No user data</p>'}
                    </div>

                    <div class="overview-card">
                        <h4>Recent Activity</h4>
                        <div class="overview-stats">
                            ${this.watchTimeStats.map(stat => `
                                <div class="stat-row">
                                    <span>${stat.query_days === 0 ? 'All Time' : `Last ${stat.query_days} days`}</span>
                                    <span>${stat.total_plays} plays</span>
                                </div>
                            `).join('')}
                        </div>
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Render users tab
     */
    private renderUsersTab(): string {
        if (this.userStats.length === 0) {
            return `
                <div class="users-tab" data-testid="users-tab">
                    <div class="empty-state">
                        <p>No user activity for this library</p>
                    </div>
                </div>
            `;
        }

        return `
            <div class="users-tab" data-testid="users-tab">
                <table class="data-table">
                    <thead>
                        <tr>
                            <th>User</th>
                            <th>Total Plays</th>
                            <th>Total Duration</th>
                            <th>Last Watch</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${this.userStats.map(user => `
                            <tr>
                                <td>
                                    <div class="user-cell">
                                        ${user.user_thumb ? `<img src="${this.escapeHtml(user.user_thumb)}" alt="" class="user-thumb">` : ''}
                                        <span>${this.escapeHtml(user.friendly_name || user.username)}</span>
                                    </div>
                                </td>
                                <td>${user.total_plays.toLocaleString()}</td>
                                <td>${this.formatDuration(user.total_duration)}</td>
                                <td>${user.last_watch ? this.formatDate(user.last_watch) : '-'}</td>
                            </tr>
                        `).join('')}
                    </tbody>
                </table>
            </div>
        `;
    }

    /**
     * Render media tab
     */
    private renderMediaTab(): string {
        if (!this.mediaInfo || this.mediaInfo.data.length === 0) {
            return `
                <div class="media-tab" data-testid="media-tab">
                    <div class="empty-state">
                        <p>No media items in this library</p>
                    </div>
                </div>
            `;
        }

        const totalPages = Math.ceil(this.mediaInfo.recordsTotal / this.mediaPageSize);

        return `
            <div class="media-tab" data-testid="media-tab">
                <div class="media-summary">
                    <span>${this.mediaInfo.recordsTotal.toLocaleString()} items</span>
                    ${this.mediaInfo.total_file_size ? `
                        <span>Total: ${this.formatFileSize(this.mediaInfo.total_file_size)}</span>
                    ` : ''}
                </div>

                <div class="table-container">
                    <table class="data-table">
                        <thead>
                            <tr>
                                ${this.renderSortableHeader('title', 'Title')}
                                ${this.renderSortableHeader('year', 'Year')}
                                ${this.renderSortableHeader('video_resolution', 'Resolution')}
                                ${this.renderSortableHeader('video_codec', 'Video')}
                                ${this.renderSortableHeader('audio_codec', 'Audio')}
                                ${this.renderSortableHeader('file_size', 'Size')}
                                ${this.renderSortableHeader('play_count', 'Plays')}
                                ${this.renderSortableHeader('added_at', 'Added')}
                            </tr>
                        </thead>
                        <tbody>
                            ${this.mediaInfo.data.map(item => this.renderMediaRow(item)).join('')}
                        </tbody>
                    </table>
                </div>

                ${totalPages > 1 ? this.renderPagination(this.mediaPage, totalPages) : ''}
            </div>
        `;
    }

    /**
     * Render sortable table header
     */
    private renderSortableHeader(column: string, label: string): string {
        const isActive = this.mediaSortColumn === column;
        const arrow = isActive
            ? (this.mediaSortDir === 'asc' ? '&#9650;' : '&#9660;')
            : '';

        return `
            <th class="sortable ${isActive ? 'active' : ''}" data-sort="${column}">
                <span>${label}</span>
                <span class="sort-arrow">${arrow}</span>
            </th>
        `;
    }

    /**
     * Render media row
     */
    private renderMediaRow(item: TautulliLibraryMediaInfoItem): string {
        return `
            <tr>
                <td class="title-cell">${this.escapeHtml(item.title)}</td>
                <td>${item.year || '-'}</td>
                <td>${item.video_resolution || '-'}</td>
                <td>${item.video_codec || '-'}</td>
                <td>${item.audio_codec || '-'}</td>
                <td>${item.file_size ? this.formatFileSize(item.file_size) : '-'}</td>
                <td>${item.play_count}</td>
                <td>${item.added_at ? this.formatDate(item.added_at) : '-'}</td>
            </tr>
        `;
    }

    /**
     * Render watch time tab
     */
    private renderWatchTimeTab(): string {
        if (this.watchTimeStats.length === 0) {
            return `
                <div class="watchtime-tab" data-testid="watchtime-tab">
                    <div class="empty-state">
                        <p>No watch time data available</p>
                    </div>
                </div>
            `;
        }

        return `
            <div class="watchtime-tab" data-testid="watchtime-tab">
                <div class="watchtime-grid">
                    ${this.watchTimeStats.map(stat => `
                        <div class="watchtime-card">
                            <h4>${stat.query_days === 0 ? 'All Time' : `Last ${stat.query_days} Days`}</h4>
                            <div class="watchtime-stats">
                                <div class="stat">
                                    <span class="stat-value">${stat.total_plays.toLocaleString()}</span>
                                    <span class="stat-label">Plays</span>
                                </div>
                                <div class="stat">
                                    <span class="stat-value">${this.formatDuration(stat.total_duration)}</span>
                                    <span class="stat-label">Duration</span>
                                </div>
                            </div>
                        </div>
                    `).join('')}
                </div>
            </div>
        `;
    }

    /**
     * Render pagination
     */
    private renderPagination(currentPage: number, totalPages: number): string {
        return `
            <div class="pagination" data-testid="media-pagination">
                <button class="page-btn" data-page="${currentPage - 1}" ${currentPage === 0 ? 'disabled' : ''}>
                    &laquo;
                </button>
                <span class="page-info">Page ${currentPage + 1} of ${totalPages}</span>
                <button class="page-btn" data-page="${currentPage + 1}" ${currentPage >= totalPages - 1 ? 'disabled' : ''}>
                    &raquo;
                </button>
            </div>
        `;
    }

    /**
     * Get icon for library type
     */
    private getLibraryTypeIcon(type: string): string {
        switch (type.toLowerCase()) {
            case 'movie':
                return '&#127909;';
            case 'show':
                return '&#128250;';
            case 'artist':
            case 'album':
            case 'track':
                return '&#127925;';
            case 'photo':
                return '&#128247;';
            default:
                return '&#128193;';
        }
    }

    /**
     * Format duration in seconds to human readable
     */
    private formatDuration(seconds: number): string {
        if (seconds === 0) return '0m';
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);

        if (hours > 24) {
            const days = Math.floor(hours / 24);
            return `${days}d ${hours % 24}h`;
        }
        if (hours > 0) {
            return `${hours}h ${minutes}m`;
        }
        return `${minutes}m`;
    }

    /**
     * Format file size
     */
    private formatFileSize(bytes: number): string {
        const units = ['B', 'KB', 'MB', 'GB', 'TB'];
        let unitIndex = 0;
        let size = bytes;

        while (size >= 1024 && unitIndex < units.length - 1) {
            size /= 1024;
            unitIndex++;
        }

        return `${size.toFixed(1)} ${units[unitIndex]}`;
    }

    /**
     * Format timestamp to date
     */
    private formatDate(timestamp: number): string {
        const date = new Date(timestamp * 1000);
        return date.toLocaleDateString();
    }

    /**
     * Escape HTML
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
        if (this.toastManager) {
            this.toastManager.error(message, 'Error', 5000);
        }
    }

    /**
     * Clean up resources and remove event listeners
     */
    destroy(): void {
        // Remove the container event listener
        const container = document.getElementById(this.containerId);
        if (container) {
            container.removeEventListener('click', this.boundHandleContainerClick);
        }

        // Reset state
        this.initialized = false;
        this.libraries = [];
        this.selectedLibraryId = null;
        this.userStats = [];
        this.mediaInfo = null;
        this.watchTimeStats = [];
        this.mediaPage = 0;
    }
}
