// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import { API, TautulliRecentlyAddedItem, TautulliLibraryDetail } from './api';
import { VirtualScrollManager } from '../app/VirtualScrollManager';
import { createLogger } from './logger';

const logger = createLogger('RecentlyAddedDashboard');

/**
 * RecentlyAddedDashboardManager
 *
 * Manages the Recently Added content dashboard with enhanced pagination.
 *
 * Enhancements:
 * - Load All option (value=0)
 * - Jump to specific page
 * - First/Last page buttons
 * - More page size options (10, 25, 50, 100, 250, All)
 *
 * Enhancement:
 * - Virtual scrolling for large datasets (100+ items)
 * - Only renders visible items plus buffer for performance
 */

/** Threshold for enabling virtual scrolling */
const VIRTUAL_SCROLL_THRESHOLD = 100;
/** Estimated height of each grid row in pixels */
const ITEM_ROW_HEIGHT = 320;

export class RecentlyAddedDashboardManager {
    private api: API;
    private currentPage = 0;
    private itemsPerPage = 25;
    private totalRecords = 0;
    private currentMediaType = '';
    private currentLibrary = 0;
    private libraries: TautulliLibraryDetail[] = [];
    private loadAllMode = false;
    private virtualScrollManager: VirtualScrollManager | null = null;
    private cachedItems: TautulliRecentlyAddedItem[] = [];
    private useVirtualScroll = false;

    constructor(api: API) {
        this.api = api;
    }

    async init(): Promise<void> {
        await this.loadLibraries();
        this.setupEventListeners();
        await this.loadRecentlyAdded();
    }

    destroy(): void {
        // Cleanup virtual scroll manager
        if (this.virtualScrollManager) {
            this.virtualScrollManager.destroy();
            this.virtualScrollManager = null;
        }
        this.cachedItems = [];
    }

    private async loadLibraries(): Promise<void> {
        try {
            this.libraries = await this.api.getTautulliLibraries();
            this.populateLibraryFilter();
        } catch (error) {
            logger.error('Failed to load libraries:', error);
        }
    }

    private populateLibraryFilter(): void {
        const select = document.getElementById('recently-added-library') as HTMLSelectElement;
        if (!select) return;

        // Clear existing options except first
        while (select.options.length > 1) {
            select.remove(1);
        }

        // Add library options
        this.libraries.forEach(lib => {
            const option = document.createElement('option');
            option.value = lib.section_id.toString();
            option.textContent = `${lib.section_name} (${lib.section_type})`;
            select.appendChild(option);
        });
    }

    private setupEventListeners(): void {
        // Media type filter
        const mediaTypeSelect = document.getElementById('recently-added-media-type') as HTMLSelectElement;
        mediaTypeSelect?.addEventListener('change', (e) => {
            this.currentMediaType = (e.target as HTMLSelectElement).value;
            this.currentPage = 0;
            this.loadRecentlyAdded();
        });

        // Library filter
        const librarySelect = document.getElementById('recently-added-library') as HTMLSelectElement;
        librarySelect?.addEventListener('change', (e) => {
            this.currentLibrary = parseInt((e.target as HTMLSelectElement).value) || 0;
            this.currentPage = 0;
            this.loadRecentlyAdded();
        });

        // Items per page (Handle Load All option)
        const countSelect = document.getElementById('recently-added-count') as HTMLSelectElement;
        countSelect?.addEventListener('change', (e) => {
            const value = parseInt((e.target as HTMLSelectElement).value);
            if (value === 0) {
                // Load All mode
                this.loadAllMode = true;
                this.itemsPerPage = 10000; // Large number to get all items
            } else {
                this.loadAllMode = false;
                this.itemsPerPage = value;
            }
            this.currentPage = 0;
            this.loadRecentlyAdded();
        });

        // Enhanced pagination controls
        // First page button
        document.getElementById('recently-added-first')?.addEventListener('click', () => {
            if (this.currentPage > 0) {
                this.currentPage = 0;
                this.loadRecentlyAdded();
            }
        });

        // Previous button
        document.getElementById('recently-added-prev')?.addEventListener('click', () => {
            if (this.currentPage > 0) {
                this.currentPage--;
                this.loadRecentlyAdded();
            }
        });

        // Next button
        document.getElementById('recently-added-next')?.addEventListener('click', () => {
            const totalPages = this.getTotalPages();
            if (this.currentPage < totalPages - 1) {
                this.currentPage++;
                this.loadRecentlyAdded();
            }
        });

        // Last page button
        document.getElementById('recently-added-last')?.addEventListener('click', () => {
            const totalPages = this.getTotalPages();
            if (this.currentPage < totalPages - 1) {
                this.currentPage = totalPages - 1;
                this.loadRecentlyAdded();
            }
        });

        // Page jump input
        const pageInput = document.getElementById('recently-added-page-input') as HTMLInputElement;
        const goButton = document.getElementById('recently-added-go');

        goButton?.addEventListener('click', () => {
            this.goToPage(pageInput);
        });

        pageInput?.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                this.goToPage(pageInput);
            }
        });
    }

    /**
     * Go to a specific page
     */
    private goToPage(input: HTMLInputElement): void {
        const pageNumber = parseInt(input.value);
        const totalPages = this.getTotalPages();

        if (isNaN(pageNumber) || pageNumber < 1) {
            input.value = '1';
            this.currentPage = 0;
        } else if (pageNumber > totalPages) {
            input.value = totalPages.toString();
            this.currentPage = totalPages - 1;
        } else {
            this.currentPage = pageNumber - 1;
        }

        this.loadRecentlyAdded();
    }

    /**
     * Get total pages based on current settings
     */
    private getTotalPages(): number {
        if (this.loadAllMode || this.totalRecords === 0) {
            return 1;
        }
        return Math.ceil(this.totalRecords / this.itemsPerPage);
    }

    private async loadRecentlyAdded(): Promise<void> {
        const loadingEl = document.getElementById('recently-added-loading');

        if (loadingEl) {
            loadingEl.style.display = 'block';
        }

        try {
            const start = this.loadAllMode ? 0 : this.currentPage * this.itemsPerPage;
            const count = this.loadAllMode ? 10000 : this.itemsPerPage;

            const data = await this.api.getTautulliRecentlyAdded(
                count,
                start,
                this.currentMediaType || undefined,
                this.currentLibrary || undefined
            );

            this.totalRecords = data.records_total;
            this.renderItems(data.recently_added);
            this.updatePagination(data.records_total);
        } catch (error) {
            logger.error('Failed to load recently added:', error);
            this.showError();
            // Initialize pagination with 0 records so controls are visible but disabled
            this.updatePagination(0);
        } finally {
            if (loadingEl) {
                loadingEl.style.display = 'none';
            }
        }
    }

    private renderItems(items: TautulliRecentlyAddedItem[]): void {
        const gridEl = document.getElementById('recently-added-grid');
        if (!gridEl) {
            logger.error('Recently added grid element not found');
            return;
        }

        // Store items for virtual scrolling
        this.cachedItems = items;

        // Determine if we should use virtual scrolling
        this.useVirtualScroll = items.length >= VIRTUAL_SCROLL_THRESHOLD;

        if (items.length === 0) {
            this.disableVirtualScroll(gridEl);
            gridEl.innerHTML = '<div class="empty-state"><p>No recently added content found</p></div>';
            return;
        }

        if (this.useVirtualScroll) {
            this.renderWithVirtualScroll(gridEl, items);
        } else {
            this.renderWithoutVirtualScroll(gridEl, items);
        }
    }

    /**
     * Render items with virtual scrolling for large datasets
     */
    private renderWithVirtualScroll(gridEl: HTMLElement, items: TautulliRecentlyAddedItem[]): void {
        // Calculate items per row based on container width
        const containerWidth = gridEl.clientWidth || 800;
        const itemWidth = 220; // Approximate card width including gap
        const itemsPerRow = Math.max(1, Math.floor(containerWidth / itemWidth));

        // Initialize or update virtual scroll manager
        if (!this.virtualScrollManager) {
            gridEl.innerHTML = '';
            gridEl.classList.add('virtual-scroll-enabled');

            this.virtualScrollManager = new VirtualScrollManager({
                container: gridEl,
                itemHeight: ITEM_ROW_HEIGHT,
                itemsPerRow: itemsPerRow,
                bufferRows: 2,
                totalItems: items.length,
                renderItem: (index: number) => this.renderVirtualItem(index),
                onRangeChange: (start, end) => {
                    logger.debug(`Virtual scroll range: ${start}-${end} of ${items.length}`);
                }
            });
        }

        // Enable virtual scrolling with current items
        this.virtualScrollManager.enable(items);

        // Show virtual scroll indicator
        this.showVirtualScrollIndicator(items.length);
    }

    /**
     * Render a single item for virtual scrolling
     */
    private renderVirtualItem(index: number): HTMLElement | null {
        const item = this.cachedItems[index];
        if (!item) return null;
        return this.createItemCard(item);
    }

    /**
     * Render items without virtual scrolling (small datasets)
     */
    private renderWithoutVirtualScroll(gridEl: HTMLElement, items: TautulliRecentlyAddedItem[]): void {
        this.disableVirtualScroll(gridEl);

        // Clear grid except loading element
        Array.from(gridEl.children).forEach(child => {
            if (child.id !== 'recently-added-loading') {
                child.remove();
            }
        });

        items.forEach(item => {
            const card = this.createItemCard(item);
            gridEl.appendChild(card);
        });

        // Hide virtual scroll indicator
        this.hideVirtualScrollIndicator();
    }

    /**
     * Disable virtual scrolling and cleanup
     */
    private disableVirtualScroll(gridEl: HTMLElement): void {
        if (this.virtualScrollManager) {
            this.virtualScrollManager.destroy();
            this.virtualScrollManager = null;
        }
        gridEl.classList.remove('virtual-scroll-enabled');
        this.hideVirtualScrollIndicator();
    }

    /**
     * Show indicator that virtual scrolling is active
     */
    private showVirtualScrollIndicator(totalItems: number): void {
        let indicator = document.getElementById('virtual-scroll-indicator');
        if (!indicator) {
            indicator = document.createElement('div');
            indicator.id = 'virtual-scroll-indicator';
            indicator.className = 'virtual-scroll-indicator';
            indicator.setAttribute('role', 'status');
            indicator.setAttribute('aria-live', 'polite');

            const paginationEl = document.getElementById('recently-added-pagination');
            if (paginationEl) {
                paginationEl.parentElement?.insertBefore(indicator, paginationEl);
            }
        }
        indicator.innerHTML = `
            <span class="virtual-scroll-icon" aria-hidden="true">&#x26A1;</span>
            <span>Virtual scrolling enabled (${totalItems} items)</span>
        `;
        indicator.style.display = 'flex';
    }

    /**
     * Hide virtual scroll indicator
     */
    private hideVirtualScrollIndicator(): void {
        const indicator = document.getElementById('virtual-scroll-indicator');
        if (indicator) {
            indicator.style.display = 'none';
        }
    }

    private createItemCard(item: TautulliRecentlyAddedItem): HTMLElement {
        const card = document.createElement('div');
        card.className = 'recently-added-card';

        const title = item.grandparent_title
            ? `${item.grandparent_title}`
            : item.title;

        const subtitle = item.parent_title
            ? `${item.parent_title} - ${item.title}`
            : item.media_type === 'show' ? item.title : '';

        const addedDate = new Date(item.added_at * 1000);
        const timeAgo = this.getTimeAgo(addedDate);

        // Create optimized image HTML with lazy loading
        const imageHTML = item.thumb
            ? `<img src="${this.escapeHtml(item.thumb)}" alt="${this.escapeHtml(title)}" loading="lazy" class="lazy-image">`
            : '<div class="placeholder-image">📺</div>';

        card.innerHTML = `
            <div class="recently-added-image">
                ${imageHTML}
            </div>
            <div class="recently-added-info">
                <div class="recently-added-title">${this.escapeHtml(title)}</div>
                ${subtitle ? `<div class="recently-added-subtitle">${this.escapeHtml(subtitle)}</div>` : ''}
                <div class="recently-added-meta">
                    <span class="media-type-badge">${item.media_type}</span>
                    ${item.year ? `<span>${item.year}</span>` : ''}
                    <span>${timeAgo}</span>
                </div>
                <div class="recently-added-library">${this.escapeHtml(item.library_name)}</div>
            </div>
        `;

        // CSP-compliant image error handling (replaces inline onerror)
        const img = card.querySelector('.lazy-image');
        if (img) {
            img.addEventListener('error', () => {
                img.classList.add('lazy-error');
            });
        }

        return card;
    }

    private getTimeAgo(date: Date): string {
        const seconds = Math.floor((Date.now() - date.getTime()) / 1000);

        const intervals = {
            year: 31536000,
            month: 2592000,
            week: 604800,
            day: 86400,
            hour: 3600,
            minute: 60
        };

        for (const [unit, secondsInUnit] of Object.entries(intervals)) {
            const interval = Math.floor(seconds / secondsInUnit);
            if (interval >= 1) {
                return `${interval} ${unit}${interval > 1 ? 's' : ''} ago`;
            }
        }

        return 'Just now';
    }

    /**
     * Enhanced pagination update with all controls
     */
    private updatePagination(totalRecords: number): void {
        this.totalRecords = totalRecords;

        const firstBtn = document.getElementById('recently-added-first') as HTMLButtonElement;
        const prevBtn = document.getElementById('recently-added-prev') as HTMLButtonElement;
        const nextBtn = document.getElementById('recently-added-next') as HTMLButtonElement;
        const lastBtn = document.getElementById('recently-added-last') as HTMLButtonElement;
        const pageInput = document.getElementById('recently-added-page-input') as HTMLInputElement;
        const pageTotal = document.getElementById('recently-added-page-total');
        const pageInfo = document.getElementById('recently-added-page-info');

        const totalPages = this.getTotalPages();
        const currentPage = this.currentPage + 1;

        // Update page input and total
        if (pageInput) {
            pageInput.value = currentPage.toString();
            pageInput.max = totalPages.toString();
        }

        if (pageTotal) {
            pageTotal.textContent = `/ ${totalPages}`;
        }

        // Update info text
        if (pageInfo) {
            if (this.loadAllMode) {
                pageInfo.textContent = `Showing all ${totalRecords} items`;
            } else {
                const startItem = this.currentPage * this.itemsPerPage + 1;
                const endItem = Math.min(startItem + this.itemsPerPage - 1, totalRecords);
                pageInfo.textContent = `${startItem}-${endItem} of ${totalRecords} items`;
            }
        }

        // Update button states
        const isFirstPage = this.currentPage === 0;
        const isLastPage = currentPage >= totalPages;

        if (firstBtn) {
            firstBtn.disabled = isFirstPage || this.loadAllMode;
        }

        if (prevBtn) {
            prevBtn.disabled = isFirstPage || this.loadAllMode;
        }

        if (nextBtn) {
            nextBtn.disabled = isLastPage || this.loadAllMode;
        }

        if (lastBtn) {
            lastBtn.disabled = isLastPage || this.loadAllMode;
        }

        // Hide pagination controls in Load All mode
        const paginationContainer = document.getElementById('recently-added-pagination');
        if (paginationContainer) {
            const jumpElements = paginationContainer.querySelectorAll('.pagination-jump, .pagination-btn:not(#recently-added-go)');
            jumpElements.forEach(el => {
                (el as HTMLElement).style.opacity = this.loadAllMode ? '0.5' : '1';
            });
        }
    }

    private escapeHtml(unsafe: string): string {
        return unsafe
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;")
            .replace(/'/g, "&#039;");
    }

    private showError(): void {
        const gridEl = document.getElementById('recently-added-grid');
        if (!gridEl) {
            logger.error('Recently added grid element not found');
            return;
        }

        // Check if error message already exists
        const existingError = gridEl.querySelector('.recently-added-error-message');
        if (!existingError) {
            // Clear loading element
            const loading = document.getElementById('recently-added-loading');
            if (loading) {
                loading.remove();
            }

            const errorDiv = document.createElement('div');
            errorDiv.className = 'recently-added-error-message error-state';
            errorDiv.style.cssText = 'padding: 40px; text-align: center;';
            errorDiv.innerHTML = `
                <p style="margin-bottom: 15px; color: #ef4444;">Failed to load recently added content</p>
                <button id="recently-added-retry" style="padding: 8px 16px; background: #ef4444; color: white; border: none; border-radius: 4px; cursor: pointer;">Retry</button>
            `;
            gridEl.appendChild(errorDiv);

            // Attach retry listener
            const retryButton = document.getElementById('recently-added-retry');
            if (retryButton) {
                retryButton.addEventListener('click', () => {
                    errorDiv.remove();
                    this.loadRecentlyAdded();
                });
            }
        }
    }
}
