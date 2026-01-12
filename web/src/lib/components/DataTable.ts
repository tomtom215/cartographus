// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * DataTable - Accessible, Reusable Data Table Component
 *
 * Features:
 * - Sortable columns (click or keyboard)
 * - Client-side filtering/search
 * - Pagination with configurable page size
 * - Row selection (single/multi)
 * - Full keyboard navigation (arrow keys, Home, End, Page Up/Down)
 * - WCAG 2.1 AA compliant ARIA attributes
 * - Screen reader announcements via live region
 * - Custom cell rendering
 * - Responsive design
 */

/**
 * Column definition for the data table
 */
export interface DataTableColumn<T = unknown> {
    /** Unique column key (matches data property) */
    key: string;
    /** Display header text */
    header: string;
    /** Whether column is sortable (default: true) */
    sortable?: boolean;
    /** Custom cell renderer */
    render?: (value: unknown, row: T, rowIndex: number) => string;
    /** Column width (CSS value) */
    width?: string;
    /** Text alignment */
    align?: 'left' | 'center' | 'right';
    /** Whether column is hidden */
    hidden?: boolean;
    /** ARIA label for header */
    ariaLabel?: string;
}

/**
 * Data table configuration options
 */
export interface DataTableOptions<T = unknown> {
    /** Table container element or ID */
    container: HTMLElement | string;
    /** Column definitions */
    columns: DataTableColumn<T>[];
    /** Initial data */
    data?: T[];
    /** Enable pagination (default: true) */
    pagination?: boolean;
    /** Items per page (default: 10) */
    pageSize?: number;
    /** Available page size options */
    pageSizeOptions?: number[];
    /** Enable row selection */
    selectable?: boolean;
    /** Multi-select mode */
    multiSelect?: boolean;
    /** Enable search/filter input */
    searchable?: boolean;
    /** Search placeholder text */
    searchPlaceholder?: string;
    /** Columns to search (default: all) */
    searchColumns?: string[];
    /** Custom row key extractor */
    getRowKey?: (row: T) => string | number;
    /** Empty state message */
    emptyMessage?: string;
    /** Table caption for accessibility */
    caption?: string;
    /** CSS class for the table */
    className?: string;
    /** Callbacks */
    onSelectionChange?: (selected: T[]) => void;
    onSort?: (column: string, direction: 'asc' | 'desc') => void;
    onPageChange?: (page: number) => void;
    onRowClick?: (row: T, index: number) => void;
}

/**
 * Sort state
 */
interface SortState {
    column: string;
    direction: 'asc' | 'desc';
}

/**
 * Accessible Data Table Component
 */
export class DataTable<T extends Record<string, unknown> = Record<string, unknown>> {
    private container: HTMLElement;
    private options: Required<DataTableOptions<T>>;
    private data: T[] = [];
    private filteredData: T[] = [];
    private sortState: SortState | null = null;
    private currentPage: number = 1;
    private selectedRows: Set<string | number> = new Set();
    private searchQuery: string = '';
    private focusedRowIndex: number = -1;
    private focusedColIndex: number = 0;
    private abortController: AbortController | null = null;
    private liveRegion: HTMLElement | null = null;

    constructor(options: DataTableOptions<T>) {
        // Resolve container
        const container = typeof options.container === 'string'
            ? document.getElementById(options.container)
            : options.container;

        if (!container) {
            throw new Error(`DataTable: Container not found`);
        }

        this.container = container;

        // Apply defaults
        this.options = {
            container: this.container,
            columns: options.columns,
            data: options.data || [],
            pagination: options.pagination ?? true,
            pageSize: options.pageSize || 10,
            pageSizeOptions: options.pageSizeOptions || [10, 25, 50, 100],
            selectable: options.selectable ?? false,
            multiSelect: options.multiSelect ?? false,
            searchable: options.searchable ?? false,
            searchPlaceholder: options.searchPlaceholder || 'Search...',
            searchColumns: options.searchColumns || options.columns.map(c => c.key),
            getRowKey: options.getRowKey || ((_row: T, index?: number) => index ?? 0) as (row: T) => string | number,
            emptyMessage: options.emptyMessage || 'No data available',
            caption: options.caption || '',
            className: options.className || '',
            onSelectionChange: options.onSelectionChange || (() => {}),
            onSort: options.onSort || (() => {}),
            onPageChange: options.onPageChange || (() => {}),
            onRowClick: options.onRowClick || (() => {}),
        };

        this.data = [...this.options.data];
        this.filteredData = [...this.data];

        this.init();
    }

    /**
     * Initialize the data table
     */
    private init(): void {
        this.abortController = new AbortController();
        this.createLiveRegion();
        this.render();
        this.setupEventListeners();
    }

    /**
     * Create live region for screen reader announcements
     */
    private createLiveRegion(): void {
        this.liveRegion = document.createElement('div');
        this.liveRegion.setAttribute('role', 'status');
        this.liveRegion.setAttribute('aria-live', 'polite');
        this.liveRegion.setAttribute('aria-atomic', 'true');
        this.liveRegion.className = 'sr-only';
        this.container.appendChild(this.liveRegion);
    }

    /**
     * Announce message to screen readers
     */
    private announce(message: string): void {
        if (this.liveRegion) {
            this.liveRegion.textContent = '';
            // Small delay ensures announcement is triggered
            setTimeout(() => {
                if (this.liveRegion) {
                    this.liveRegion.textContent = message;
                }
            }, 50);
        }
    }

    /**
     * Render the complete table
     */
    private render(): void {
        const visibleColumns = this.getVisibleColumns();
        const paginatedData = this.getPaginatedData();
        const totalPages = this.getTotalPages();

        this.container.innerHTML = `
            <div class="data-table-container ${this.options.className}">
                ${this.renderSearchIfEnabled()}
                ${this.renderTableInfo()}
                ${this.renderTableWrapper(visibleColumns, paginatedData)}
                ${this.renderPaginationIfEnabled(totalPages)}
            </div>
        `;

        // Re-attach live region (it was replaced by innerHTML)
        this.createLiveRegion();
    }

    /**
     * Get visible columns
     */
    private getVisibleColumns(): DataTableColumn<T>[] {
        return this.options.columns.filter(c => !c.hidden);
    }

    /**
     * Render search if enabled
     */
    private renderSearchIfEnabled(): string {
        return this.options.searchable ? this.renderSearch() : '';
    }

    /**
     * Render pagination if enabled
     */
    private renderPaginationIfEnabled(totalPages: number): string {
        return this.options.pagination && totalPages > 1 ? this.renderPagination(totalPages) : '';
    }

    /**
     * Render table wrapper
     */
    private renderTableWrapper(visibleColumns: DataTableColumn<T>[], paginatedData: T[]): string {
        return `
            <div class="data-table-wrapper" role="region" aria-label="Data table" tabindex="0">
                <table class="data-table" role="grid" aria-rowcount="${this.filteredData.length}">
                    ${this.renderTableCaption()}
                    ${this.renderTableHead(visibleColumns)}
                    ${this.renderTableBody(visibleColumns, paginatedData)}
                </table>
            </div>
        `;
    }

    /**
     * Render table caption
     */
    private renderTableCaption(): string {
        return this.options.caption
            ? `<caption class="sr-only">${this.escapeHtml(this.options.caption)}</caption>`
            : '';
    }

    /**
     * Render table head
     */
    private renderTableHead(visibleColumns: DataTableColumn<T>[]): string {
        return `
            <thead>
                <tr role="row">
                    ${this.options.selectable ? this.renderSelectAllHeader() : ''}
                    ${visibleColumns.map((col, i) => this.renderHeader(col, i)).join('')}
                </tr>
            </thead>
        `;
    }

    /**
     * Render table body
     */
    private renderTableBody(visibleColumns: DataTableColumn<T>[], paginatedData: T[]): string {
        const colspan = visibleColumns.length + (this.options.selectable ? 1 : 0);
        const rows = paginatedData.length > 0
            ? paginatedData.map((row, i) => this.renderRow(row, i, visibleColumns)).join('')
            : this.renderEmptyState(colspan);

        return `<tbody>${rows}</tbody>`;
    }

    /**
     * Render search input
     */
    private renderSearch(): string {
        return `
            <div class="data-table-search">
                <label for="dt-search" class="sr-only">Search table</label>
                <div class="search-input-wrapper">
                    ${this.renderSearchIcon()}
                    ${this.renderSearchInput()}
                    ${this.renderSearchClearButton()}
                </div>
            </div>
        `;
    }

    /**
     * Render search icon
     */
    private renderSearchIcon(): string {
        return `
            <svg class="search-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                <circle cx="11" cy="11" r="8"/>
                <path d="M21 21l-4.35-4.35"/>
            </svg>
        `;
    }

    /**
     * Render search input field
     */
    private renderSearchInput(): string {
        return `
            <input
                type="search"
                id="dt-search"
                class="data-table-search-input"
                placeholder="${this.escapeHtml(this.options.searchPlaceholder)}"
                value="${this.escapeHtml(this.searchQuery)}"
                aria-controls="data-table-body"
            />
        `;
    }

    /**
     * Render search clear button
     */
    private renderSearchClearButton(): string {
        if (!this.searchQuery) return '';

        return `
            <button type="button" class="search-clear-btn" aria-label="Clear search">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                    <line x1="18" y1="6" x2="6" y2="18"/>
                    <line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
            </button>
        `;
    }

    /**
     * Render table info (showing X of Y)
     */
    private renderTableInfo(): string {
        const total = this.filteredData.length;
        if (total === 0) return '';

        const start = (this.currentPage - 1) * this.options.pageSize + 1;
        const end = Math.min(this.currentPage * this.options.pageSize, total);
        const filterInfo = this.getFilterInfo();

        return `
            <div class="data-table-info" aria-live="polite">
                Showing ${start} to ${end} of ${total} entries${filterInfo}
            </div>
        `;
    }

    /**
     * Get filter info text
     */
    private getFilterInfo(): string {
        const total = this.filteredData.length;
        const originalTotal = this.data.length;

        return total !== originalTotal
            ? ` (filtered from ${originalTotal} entries)`
            : '';
    }

    /**
     * Render select all checkbox header
     */
    private renderSelectAllHeader(): string {
        if (!this.options.multiSelect) {
            return '<th class="select-cell" scope="col"><span class="sr-only">Select</span></th>';
        }

        const { allSelected, someSelected } = this.getSelectAllState();

        return `
            <th class="select-cell" scope="col">
                <input
                    type="checkbox"
                    class="select-all-checkbox"
                    aria-label="Select all rows"
                    ${allSelected ? 'checked' : ''}
                    ${someSelected ? 'data-indeterminate="true"' : ''}
                />
            </th>
        `;
    }

    /**
     * Get select all checkbox state
     */
    private getSelectAllState(): { allSelected: boolean; someSelected: boolean } {
        const allSelected = this.filteredData.length > 0 &&
            this.filteredData.every((row, i) => this.selectedRows.has(this.getRowKey(row, i)));
        const someSelected = this.selectedRows.size > 0 && !allSelected;

        return { allSelected, someSelected };
    }

    /**
     * Render column header
     */
    private renderHeader(col: DataTableColumn<T>, index: number): string {
        const sortable = col.sortable !== false;
        const isSorted = this.sortState?.column === col.key;
        const sortDir = isSorted ? this.sortState!.direction : null;

        const style = col.width ? `width: ${col.width};` : '';
        const alignClass = col.align ? `text-${col.align}` : '';
        const sortedClass = isSorted ? 'sorted' : '';
        const sortableClass = sortable ? 'sortable' : '';

        return `
            <th
                scope="col"
                class="data-table-header ${sortableClass} ${sortedClass} ${alignClass}"
                data-column="${col.key}"
                data-col-index="${index}"
                style="${style}"
                ${this.getHeaderAriaAttributes(col, sortable, isSorted, sortDir)}
            >
                <span class="header-content">
                    <span class="header-text">${this.escapeHtml(col.header)}</span>
                    ${sortable ? this.renderSortIndicator(isSorted, sortDir) : ''}
                </span>
            </th>
        `;
    }

    /**
     * Get ARIA attributes for header
     */
    private getHeaderAriaAttributes(
        col: DataTableColumn<T>,
        sortable: boolean,
        isSorted: boolean,
        sortDir: 'asc' | 'desc' | null
    ): string {
        if (!sortable) return '';

        const ariaSort = isSorted ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none';
        const ariaLabel = `${col.ariaLabel || col.header}, sortable`;

        return `
            role="columnheader"
            tabindex="0"
            aria-sort="${ariaSort}"
            aria-label="${ariaLabel}"
        `;
    }

    /**
     * Render sort indicator
     */
    private renderSortIndicator(isSorted: boolean, sortDir: 'asc' | 'desc' | null): string {
        let indicator = '⇅';
        if (isSorted) {
            indicator = sortDir === 'asc' ? '▲' : '▼';
        }

        return `<span class="sort-indicator" aria-hidden="true">${indicator}</span>`;
    }

    /**
     * Render table row
     */
    private renderRow(row: T, rowIndex: number, columns: DataTableColumn<T>[]): string {
        const actualIndex = (this.currentPage - 1) * this.options.pageSize + rowIndex;
        const rowKey = this.getRowKey(row, actualIndex);
        const isSelected = this.selectedRows.has(rowKey);
        const isFocused = this.focusedRowIndex === rowIndex;
        const rowClasses = this.getRowClasses(isSelected, isFocused);

        return `
            <tr
                class="${rowClasses}"
                data-row-key="${rowKey}"
                data-row-index="${rowIndex}"
                role="row"
                aria-rowindex="${actualIndex + 2}"
                aria-selected="${isSelected}"
                tabindex="${isFocused ? '0' : '-1'}"
            >
                ${this.renderRowSelectCell(rowKey, isSelected)}
                ${columns.map((col, colIndex) => this.renderCell(row, col, rowIndex, colIndex)).join('')}
            </tr>
        `;
    }

    /**
     * Get row CSS classes
     */
    private getRowClasses(isSelected: boolean, isFocused: boolean): string {
        const classes = ['data-table-row'];
        if (isSelected) classes.push('selected');
        if (isFocused) classes.push('focused');
        return classes.join(' ');
    }

    /**
     * Render row select cell if selectable
     */
    private renderRowSelectCell(rowKey: string | number, isSelected: boolean): string {
        return this.options.selectable ? this.renderSelectCell(rowKey, isSelected) : '';
    }

    /**
     * Render selection cell
     */
    private renderSelectCell(rowKey: string | number, isSelected: boolean): string {
        const inputType = this.options.multiSelect ? 'checkbox' : 'radio';
        return `
            <td class="select-cell">
                <input
                    type="${inputType}"
                    class="row-select"
                    name="table-row-select"
                    aria-label="Select row"
                    data-row-key="${rowKey}"
                    ${isSelected ? 'checked' : ''}
                />
            </td>
        `;
    }

    /**
     * Render table cell
     */
    private renderCell(row: T, col: DataTableColumn<T>, rowIndex: number, colIndex: number): string {
        const value = row[col.key];
        const alignClass = col.align ? `text-${col.align}` : '';
        const isFocused = this.focusedRowIndex === rowIndex && this.focusedColIndex === colIndex;
        const cellContent = this.getCellContent(value, col, row, rowIndex);

        return `
            <td
                class="data-table-cell ${alignClass} ${isFocused ? 'focused' : ''}"
                data-column="${col.key}"
                data-col-index="${colIndex}"
                tabindex="${isFocused ? '0' : '-1'}"
            >
                ${cellContent}
            </td>
        `;
    }

    /**
     * Get cell content based on value and column configuration
     */
    private getCellContent(value: unknown, col: DataTableColumn<T>, row: T, rowIndex: number): string {
        if (col.render) {
            return col.render(value, row, rowIndex);
        }

        if (value === null || value === undefined) {
            return '-';
        }

        if (typeof value === 'number') {
            return value.toLocaleString();
        }

        return this.escapeHtml(String(value));
    }

    /**
     * Render empty state
     */
    private renderEmptyState(colspan: number): string {
        return `
            <tr class="empty-row">
                <td colspan="${colspan}" class="empty-cell">
                    <div class="empty-state">
                        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
                            <path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                        </svg>
                        <p>${this.escapeHtml(this.options.emptyMessage)}</p>
                        ${this.searchQuery ? '<p class="empty-hint">Try adjusting your search terms</p>' : ''}
                    </div>
                </td>
            </tr>
        `;
    }

    /**
     * Render pagination controls
     */
    private renderPagination(totalPages: number): string {
        const pages = this.getPageNumbers(totalPages);
        const hasPrev = this.currentPage > 1;
        const hasNext = this.currentPage < totalPages;

        return `
            <nav class="data-table-pagination" aria-label="Table pagination">
                ${this.renderPageSizeSelector()}
                <div class="pagination-controls" role="group" aria-label="Pagination">
                    ${this.renderFirstPageButton(hasPrev)}
                    ${this.renderPrevPageButton(hasPrev)}
                    <div class="pagination-pages">
                        ${pages.map(page => this.renderPageButton(page)).join('')}
                    </div>
                    ${this.renderNextPageButton(hasNext, totalPages)}
                    ${this.renderLastPageButton(hasNext, totalPages)}
                </div>
            </nav>
        `;
    }

    /**
     * Render page size selector
     */
    private renderPageSizeSelector(): string {
        return `
            <div class="pagination-size">
                <label for="page-size-select">Show:</label>
                <select id="page-size-select" class="page-size-select" aria-label="Items per page">
                    ${this.options.pageSizeOptions.map(size => `
                        <option value="${size}" ${size === this.options.pageSize ? 'selected' : ''}>
                            ${size}
                        </option>
                    `).join('')}
                </select>
            </div>
        `;
    }

    /**
     * Render first page button
     */
    private renderFirstPageButton(enabled: boolean): string {
        return `
            <button
                type="button"
                class="pagination-btn pagination-first"
                aria-label="Go to first page"
                ${!enabled ? 'disabled' : ''}
                data-page="1"
            >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                    <polyline points="11 17 6 12 11 7"/>
                    <polyline points="18 17 13 12 18 7"/>
                </svg>
            </button>
        `;
    }

    /**
     * Render previous page button
     */
    private renderPrevPageButton(enabled: boolean): string {
        return `
            <button
                type="button"
                class="pagination-btn pagination-prev"
                aria-label="Go to previous page"
                ${!enabled ? 'disabled' : ''}
                data-page="${this.currentPage - 1}"
            >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                    <polyline points="15 18 9 12 15 6"/>
                </svg>
            </button>
        `;
    }

    /**
     * Render next page button
     */
    private renderNextPageButton(enabled: boolean, _totalPages: number): string {
        return `
            <button
                type="button"
                class="pagination-btn pagination-next"
                aria-label="Go to next page"
                ${!enabled ? 'disabled' : ''}
                data-page="${this.currentPage + 1}"
            >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                    <polyline points="9 18 15 12 9 6"/>
                </svg>
            </button>
        `;
    }

    /**
     * Render last page button
     */
    private renderLastPageButton(enabled: boolean, totalPages: number): string {
        return `
            <button
                type="button"
                class="pagination-btn pagination-last"
                aria-label="Go to last page"
                ${!enabled ? 'disabled' : ''}
                data-page="${totalPages}"
            >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                    <polyline points="13 17 18 12 13 7"/>
                    <polyline points="6 17 11 12 6 7"/>
                </svg>
            </button>
        `;
    }

    /**
     * Render page button or ellipsis
     */
    private renderPageButton(page: number | string): string {
        if (page === '...') {
            return '<span class="pagination-ellipsis">...</span>';
        }

        const pageNum = page as number;
        const isCurrent = pageNum === this.currentPage;

        return `
            <button
                type="button"
                class="pagination-btn pagination-page ${isCurrent ? 'current' : ''}"
                aria-label="Page ${pageNum}"
                aria-current="${isCurrent ? 'page' : 'false'}"
                data-page="${pageNum}"
            >
                ${pageNum}
            </button>
        `;
    }

    /**
     * Get page numbers for pagination (with ellipsis)
     */
    private getPageNumbers(totalPages: number): (number | string)[] {
        const pages: (number | string)[] = [];
        const current = this.currentPage;
        const delta = 2; // Pages to show around current

        const start = Math.max(1, current - delta);
        const end = Math.min(totalPages, current + delta);

        this.addFirstPageAndEllipsis(pages, start);
        this.addMiddlePages(pages, start, end);
        this.addLastPageAndEllipsis(pages, end, totalPages);

        return pages;
    }

    /**
     * Add first page and ellipsis if needed
     */
    private addFirstPageAndEllipsis(pages: (number | string)[], start: number): void {
        if (start <= 1) return;

        pages.push(1);
        if (start > 2) {
            pages.push('...');
        }
    }

    /**
     * Add middle page numbers
     */
    private addMiddlePages(pages: (number | string)[], start: number, end: number): void {
        for (let i = start; i <= end; i++) {
            pages.push(i);
        }
    }

    /**
     * Add last page and ellipsis if needed
     */
    private addLastPageAndEllipsis(pages: (number | string)[], end: number, totalPages: number): void {
        if (end >= totalPages) return;

        if (end < totalPages - 1) {
            pages.push('...');
        }
        pages.push(totalPages);
    }

    /**
     * Setup event listeners
     */
    private setupEventListeners(): void {
        const signal = this.abortController!.signal;

        // Delegated event handling
        this.container.addEventListener('click', (e) => this.handleClick(e), { signal });
        this.container.addEventListener('keydown', (e) => this.handleKeydown(e), { signal });
        this.container.addEventListener('input', (e) => this.handleInput(e), { signal });
        this.container.addEventListener('change', (e) => this.handleChange(e), { signal });
    }

    /**
     * Handle click events
     */
    private handleClick(e: Event): void {
        const target = e.target as HTMLElement;

        if (this.handleSortHeaderClick(target)) return;
        if (this.handleRowClick(target)) return;
        if (this.handlePaginationClick(target)) return;
        this.handleSearchClearClick(target);
    }

    /**
     * Handle sort header click
     */
    private handleSortHeaderClick(target: HTMLElement): boolean {
        const header = target.closest('.data-table-header.sortable') as HTMLElement;
        if (!header) return false;

        const column = header.dataset.column;
        if (column) {
            this.sort(column);
        }
        return true;
    }

    /**
     * Handle row click
     */
    private handleRowClick(target: HTMLElement): boolean {
        const row = target.closest('.data-table-row') as HTMLElement;
        if (!row || target.closest('.row-select, .select-cell')) return false;

        const rowIndex = parseInt(row.dataset.rowIndex || '0', 10);
        const actualIndex = (this.currentPage - 1) * this.options.pageSize + rowIndex;

        if (actualIndex < this.filteredData.length) {
            this.options.onRowClick(this.filteredData[actualIndex], actualIndex);
        }
        return true;
    }

    /**
     * Handle pagination button click
     */
    private handlePaginationClick(target: HTMLElement): boolean {
        const pageBtn = target.closest('.pagination-btn[data-page]') as HTMLElement;
        if (!pageBtn || pageBtn.hasAttribute('disabled')) return false;

        const page = parseInt(pageBtn.dataset.page || '1', 10);
        this.goToPage(page);
        return true;
    }

    /**
     * Handle search clear button click
     */
    private handleSearchClearClick(target: HTMLElement): void {
        if (!target.closest('.search-clear-btn')) return;

        this.setSearch('');
        const searchInput = this.container.querySelector('.data-table-search-input') as HTMLInputElement;
        if (searchInput) {
            searchInput.value = '';
            searchInput.focus();
        }
    }

    /**
     * Handle keyboard events
     */
    private handleKeydown(e: KeyboardEvent): void {
        const target = e.target as HTMLElement;

        // Header keyboard sorting
        if (target.classList.contains('data-table-header') && target.classList.contains('sortable')) {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                const column = target.dataset.column;
                if (column) {
                    this.sort(column);
                }
            }
            return;
        }

        // Table navigation
        const row = target.closest('.data-table-row') as HTMLElement;
        if (row || target.classList.contains('data-table-cell')) {
            this.handleTableNavigation(e, row || target.closest('.data-table-row') as HTMLElement);
        }
    }

    /**
     * Handle table keyboard navigation
     */
    private handleTableNavigation(e: KeyboardEvent, currentRow: HTMLElement): void {
        const key = e.key;
        const rowIndex = parseInt(currentRow?.dataset.rowIndex || '0', 10);
        const paginatedData = this.getPaginatedData();

        const handlers: Record<string, () => void> = {
            'ArrowDown': () => this.handleArrowDown(e, rowIndex, paginatedData.length),
            'ArrowUp': () => this.handleArrowUp(e, rowIndex),
            'Home': () => this.handleHome(e),
            'End': () => this.handleEnd(e, paginatedData),
            'PageDown': () => this.handlePageDown(e),
            'PageUp': () => this.handlePageUp(e),
            'Enter': () => this.handleEnterOrSpace(e, currentRow),
            ' ': () => this.handleEnterOrSpace(e, currentRow),
        };

        const handler = handlers[key];
        if (handler) {
            handler();
        }
    }

    /**
     * Handle arrow down navigation
     */
    private handleArrowDown(e: KeyboardEvent, rowIndex: number, dataLength: number): void {
        e.preventDefault();
        if (rowIndex < dataLength - 1) {
            this.focusRow(rowIndex + 1);
            return;
        }

        if (this.currentPage < this.getTotalPages()) {
            this.goToPage(this.currentPage + 1);
            this.focusRow(0);
        }
    }

    /**
     * Handle arrow up navigation
     */
    private handleArrowUp(e: KeyboardEvent, rowIndex: number): void {
        e.preventDefault();
        if (rowIndex > 0) {
            this.focusRow(rowIndex - 1);
            return;
        }

        if (this.currentPage > 1) {
            this.goToPage(this.currentPage - 1);
            this.focusRow(this.options.pageSize - 1);
        }
    }

    /**
     * Handle home key navigation
     */
    private handleHome(e: KeyboardEvent): void {
        e.preventDefault();
        if (e.ctrlKey) {
            this.goToPage(1);
        }
        this.focusRow(0);
    }

    /**
     * Handle end key navigation
     */
    private handleEnd(e: KeyboardEvent, paginatedData: T[]): void {
        e.preventDefault();
        if (e.ctrlKey) {
            this.goToPage(this.getTotalPages());
            const lastPageData = this.getPaginatedData();
            this.focusRow(lastPageData.length - 1);
            return;
        }
        this.focusRow(paginatedData.length - 1);
    }

    /**
     * Handle page down navigation
     */
    private handlePageDown(e: KeyboardEvent): void {
        e.preventDefault();
        if (this.currentPage < this.getTotalPages()) {
            this.goToPage(this.currentPage + 1);
            this.focusRow(0);
        }
    }

    /**
     * Handle page up navigation
     */
    private handlePageUp(e: KeyboardEvent): void {
        e.preventDefault();
        if (this.currentPage > 1) {
            this.goToPage(this.currentPage - 1);
            this.focusRow(0);
        }
    }

    /**
     * Handle enter or space key for selection
     */
    private handleEnterOrSpace(e: KeyboardEvent, currentRow: HTMLElement): void {
        e.preventDefault();
        if (!this.options.selectable) return;

        const rowKey = currentRow?.dataset.rowKey;
        if (rowKey) {
            this.toggleRowSelection(rowKey);
        }
    }

    /**
     * Focus a specific row
     */
    private focusRow(rowIndex: number): void {
        this.focusedRowIndex = rowIndex;
        const row = this.container.querySelector(`[data-row-index="${rowIndex}"]`) as HTMLElement;
        if (row) {
            row.focus();
            row.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
        }
    }

    /**
     * Handle input events (search)
     */
    private handleInput(e: Event): void {
        const target = e.target as HTMLInputElement;

        if (target.classList.contains('data-table-search-input')) {
            this.setSearch(target.value);
        }
    }

    /**
     * Handle change events (selection, page size)
     */
    private handleChange(e: Event): void {
        const target = e.target as HTMLInputElement | HTMLSelectElement;

        if (this.handleRowSelectionChange(target)) return;
        if (this.handleSelectAllChange(target)) return;
        this.handlePageSizeChange(target);
    }

    /**
     * Handle row selection change
     */
    private handleRowSelectionChange(target: HTMLInputElement | HTMLSelectElement): boolean {
        if (!target.classList.contains('row-select')) return false;

        const rowKey = target.dataset.rowKey;
        if (rowKey) {
            this.toggleRowSelection(rowKey);
        }
        return true;
    }

    /**
     * Handle select all change
     */
    private handleSelectAllChange(target: HTMLInputElement | HTMLSelectElement): boolean {
        if (!target.classList.contains('select-all-checkbox')) return false;

        this.toggleSelectAll((target as HTMLInputElement).checked);
        return true;
    }

    /**
     * Handle page size change
     */
    private handlePageSizeChange(target: HTMLInputElement | HTMLSelectElement): void {
        if (!target.classList.contains('page-size-select')) return;

        const newSize = parseInt(target.value, 10);
        this.setPageSize(newSize);
    }

    /**
     * Sort by column
     */
    sort(column: string): void {
        let direction: 'asc' | 'desc' = 'asc';

        if (this.sortState?.column === column) {
            direction = this.sortState.direction === 'asc' ? 'desc' : 'asc';
        }

        this.sortState = { column, direction };
        this.applySorting();
        this.currentPage = 1;
        this.render();
        this.setupEventListeners();

        const col = this.options.columns.find(c => c.key === column);
        this.announce(`Sorted by ${col?.header || column}, ${direction === 'asc' ? 'ascending' : 'descending'}`);
        this.options.onSort(column, direction);
    }

    /**
     * Apply sorting to filtered data
     */
    private applySorting(): void {
        if (!this.sortState) return;

        const { column, direction } = this.sortState;
        this.filteredData.sort((a, b) => this.compareValues(a[column], b[column], direction));
    }

    /**
     * Compare two values for sorting
     */
    private compareValues(valA: unknown, valB: unknown, direction: 'asc' | 'desc'): number {
        const nullResult = this.compareNullValues(valA, valB, direction);
        if (nullResult !== null) return nullResult;

        if (typeof valA === 'number' && typeof valB === 'number') {
            return this.compareNumbers(valA, valB, direction);
        }

        return this.compareStrings(valA, valB, direction);
    }

    /**
     * Compare null/undefined values
     */
    private compareNullValues(valA: unknown, valB: unknown, direction: 'asc' | 'desc'): number | null {
        if (valA == null && valB == null) return 0;
        if (valA == null) return direction === 'asc' ? 1 : -1;
        if (valB == null) return direction === 'asc' ? -1 : 1;
        return null;
    }

    /**
     * Compare numeric values
     */
    private compareNumbers(valA: number, valB: number, direction: 'asc' | 'desc'): number {
        return direction === 'asc' ? valA - valB : valB - valA;
    }

    /**
     * Compare string values
     */
    private compareStrings(valA: unknown, valB: unknown, direction: 'asc' | 'desc'): number {
        const strA = String(valA).toLowerCase();
        const strB = String(valB).toLowerCase();
        const comparison = strA.localeCompare(strB);
        return direction === 'asc' ? comparison : -comparison;
    }

    /**
     * Set search query
     */
    setSearch(query: string): void {
        this.searchQuery = query.trim().toLowerCase();
        this.applyFiltering();
        this.currentPage = 1;
        this.render();
        this.setupEventListeners();

        if (query) {
            this.announce(`${this.filteredData.length} results found`);
        } else {
            this.announce(`Showing all ${this.data.length} entries`);
        }
    }

    /**
     * Apply filtering based on search query
     */
    private applyFiltering(): void {
        if (!this.searchQuery) {
            this.filteredData = [...this.data];
        } else {
            this.filteredData = this.data.filter(row => {
                return this.options.searchColumns.some(key => {
                    const value = row[key];
                    if (value == null) return false;
                    return String(value).toLowerCase().includes(this.searchQuery);
                });
            });
        }

        // Re-apply sorting after filtering
        if (this.sortState) {
            this.applySorting();
        }
    }

    /**
     * Go to specific page
     */
    goToPage(page: number): void {
        const totalPages = this.getTotalPages();
        const newPage = Math.max(1, Math.min(page, totalPages));

        if (newPage !== this.currentPage) {
            this.currentPage = newPage;
            this.focusedRowIndex = 0;
            this.render();
            this.setupEventListeners();

            this.announce(`Page ${newPage} of ${totalPages}`);
            this.options.onPageChange(newPage);
        }
    }

    /**
     * Set page size
     */
    setPageSize(size: number): void {
        this.options.pageSize = size;
        this.currentPage = 1;
        this.render();
        this.setupEventListeners();

        this.announce(`Showing ${size} items per page`);
    }

    /**
     * Toggle row selection
     */
    toggleRowSelection(rowKey: string | number): void {
        if (!this.options.multiSelect) {
            // Single select - clear others
            this.selectedRows.clear();
        }

        if (this.selectedRows.has(rowKey)) {
            this.selectedRows.delete(rowKey);
        } else {
            this.selectedRows.add(rowKey);
        }

        this.render();
        this.setupEventListeners();

        const selectedData = this.getSelectedData();
        this.announce(`${selectedData.length} row${selectedData.length !== 1 ? 's' : ''} selected`);
        this.options.onSelectionChange(selectedData);
    }

    /**
     * Toggle select all
     */
    toggleSelectAll(selectAll: boolean): void {
        if (selectAll) {
            this.filteredData.forEach((row, i) => {
                this.selectedRows.add(this.getRowKey(row, i));
            });
        } else {
            this.selectedRows.clear();
        }

        this.render();
        this.setupEventListeners();

        const selectedData = this.getSelectedData();
        this.announce(selectAll
            ? `All ${selectedData.length} rows selected`
            : 'Selection cleared'
        );
        this.options.onSelectionChange(selectedData);
    }

    /**
     * Get selected data
     */
    getSelectedData(): T[] {
        return this.filteredData.filter((row, i) =>
            this.selectedRows.has(this.getRowKey(row, i))
        );
    }

    /**
     * Get row key
     */
    private getRowKey(row: T, index: number): string | number {
        return this.options.getRowKey(row) ?? index;
    }

    /**
     * Get paginated data
     */
    private getPaginatedData(): T[] {
        if (!this.options.pagination) return this.filteredData;

        const start = (this.currentPage - 1) * this.options.pageSize;
        const end = start + this.options.pageSize;
        return this.filteredData.slice(start, end);
    }

    /**
     * Get total pages
     */
    private getTotalPages(): number {
        return Math.ceil(this.filteredData.length / this.options.pageSize);
    }

    /**
     * Update table data
     */
    setData(data: T[]): void {
        this.data = [...data];
        this.applyFiltering();
        this.currentPage = 1;
        this.selectedRows.clear();
        this.render();
        this.setupEventListeners();

        this.announce(`Table updated with ${data.length} entries`);
    }

    /**
     * Get current data
     */
    getData(): T[] {
        return [...this.data];
    }

    /**
     * Escape HTML
     */
    private escapeHtml(text: string): string {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    /**
     * Refresh the table
     */
    refresh(): void {
        this.render();
        this.setupEventListeners();
    }

    /**
     * Destroy the table
     */
    destroy(): void {
        if (this.abortController) {
            this.abortController.abort();
            this.abortController = null;
        }
        this.container.innerHTML = '';
        this.data = [];
        this.filteredData = [];
        this.selectedRows.clear();
        this.liveRegion = null;
    }
}

export default DataTable;
