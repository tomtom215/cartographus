// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Export Manager
 *
 * Manages Tautulli export functionality:
 * - View existing exports
 * - Create new exports
 * - Download exports
 * - Delete exports
 */

import { API } from '../lib/api';
import type { ToastManager } from '../lib/toast';
import type { ConfirmationDialogManager } from './ConfirmationDialogManager';
import type {
    TautulliExportsTableRow,
    TautulliExportsTableData,
    TautulliExportFieldItem,
    TautulliLibraryNameItem
} from '../lib/types';
import { createLogger } from '../lib/logger';

const logger = createLogger('ExportManager');

interface ExportManagerOptions {
    api: API;
    containerId: string;
    toastManager?: ToastManager;
    confirmationManager?: ConfirmationDialogManager;
}

type ViewMode = 'list' | 'create';

export class ExportManager {
    private api: API;
    private containerId: string;
    private toastManager: ToastManager | null = null;
    private confirmationManager: ConfirmationDialogManager | null = null;
    private initialized = false;

    // Event handler references for cleanup
    private containerClickHandler: ((e: Event) => void) | null = null;
    private containerChangeHandler: ((e: Event) => void) | null = null;

    // View state
    private viewMode: ViewMode = 'list';

    // Exports list state
    private exports: TautulliExportsTableRow[] = [];
    private exportsTotal = 0;
    private currentPage = 0;
    private pageSize = 25;
    private sortColumn = 'timestamp';
    private sortDir: 'asc' | 'desc' = 'desc';

    // Create export state
    private libraries: TautulliLibraryNameItem[] = [];
    private exportFields: TautulliExportFieldItem[] = [];
    private selectedLibrary: number | null = null;
    private selectedMediaType: string = 'movie';
    private selectedFormat: string = 'csv';
    private selectedFields: string[] = [];

    constructor(options: ExportManagerOptions) {
        this.api = options.api;
        this.containerId = options.containerId;
        if (options.toastManager) {
            this.toastManager = options.toastManager;
        }
        if (options.confirmationManager) {
            this.confirmationManager = options.confirmationManager;
        }
    }

    /**
     * Set toast manager
     */
    setToastManager(toast: ToastManager): void {
        this.toastManager = toast;
    }

    /**
     * Set confirmation dialog manager
     */
    setConfirmationManager(manager: ConfirmationDialogManager): void {
        this.confirmationManager = manager;
    }

    /**
     * Initialize the manager
     */
    async init(): Promise<void> {
        if (this.initialized) return;

        try {
            // Load initial data
            await Promise.all([
                this.loadExports(),
                this.loadLibraries()
            ]);

            this.initialized = true;
        } catch (error) {
            logger.error('Failed to initialize ExportManager', { error });
            this.showError('Failed to initialize export manager');
        }
    }

    /**
     * Load exports list
     */
    private async loadExports(): Promise<void> {
        try {
            const data: TautulliExportsTableData = await this.api.getExportsTable(
                this.currentPage * this.pageSize,
                this.pageSize,
                this.sortColumn,
                this.sortDir
            );

            this.exports = data.data;
            this.exportsTotal = data.recordsTotal;

            this.render();
        } catch (error) {
            logger.error('Failed to load exports', { error });
            throw error;
        }
    }

    /**
     * Load libraries for export creation
     */
    private async loadLibraries(): Promise<void> {
        try {
            this.libraries = await this.api.getTautulliLibraryNames();
            if (this.libraries.length > 0) {
                this.selectedLibrary = this.libraries[0].section_id;
                this.selectedMediaType = this.libraries[0].section_type || 'movie';
            }
        } catch (error) {
            logger.error('Failed to load libraries', { error });
        }
    }

    /**
     * Load export fields for selected media type
     */
    private async loadExportFields(): Promise<void> {
        if (!this.selectedMediaType) return;

        try {
            this.exportFields = await this.api.getExportFields(this.selectedMediaType);
            // Select all fields by default
            this.selectedFields = this.exportFields.map(f => f.field_name);
        } catch (error) {
            logger.error('Failed to load export fields', { error });
            this.exportFields = [];
        }
    }

    /**
     * Switch view mode
     */
    async setViewMode(mode: ViewMode): Promise<void> {
        this.viewMode = mode;
        if (mode === 'create' && this.exportFields.length === 0) {
            await this.loadExportFields();
        }
        this.render();
    }

    /**
     * Sort exports by column
     */
    async sortBy(column: string): Promise<void> {
        if (this.sortColumn === column) {
            this.sortDir = this.sortDir === 'asc' ? 'desc' : 'asc';
        } else {
            this.sortColumn = column;
            this.sortDir = 'desc';
        }
        this.currentPage = 0;
        await this.loadExports();
    }

    /**
     * Navigate to page
     */
    async goToPage(page: number): Promise<void> {
        const maxPage = Math.ceil(this.exportsTotal / this.pageSize) - 1;
        this.currentPage = Math.max(0, Math.min(page, maxPage));
        await this.loadExports();
    }

    /**
     * Create a new export
     */
    async createExport(): Promise<void> {
        if (!this.selectedLibrary) {
            this.showError('Please select a library');
            return;
        }

        try {
            const result = await this.api.createExport(
                this.selectedLibrary,
                'library', // Default export type
                this.selectedFormat
            );

            if (result.status === 'success' || result.export_id) {
                this.toastManager?.success(
                    'Export started successfully',
                    'Export Created',
                    5000
                );
                this.viewMode = 'list';
                await this.loadExports();
            } else if (result.error) {
                this.showError(result.error);
            }
        } catch (error) {
            logger.error('Failed to create export', { error });
            this.showError('Failed to create export');
        }
    }

    /**
     * Download an export
     */
    downloadExport(exportId: number): void {
        const url = this.api.getExportDownloadUrl(exportId);
        window.open(url, '_blank');
    }

    /**
     * Delete an export with confirmation
     */
    async confirmDeleteExport(exportRow: TautulliExportsTableRow): Promise<void> {
        const message = `Are you sure you want to delete this export?${exportRow.title ? `\n"${exportRow.title}"` : ''}`;

        if (this.confirmationManager) {
            const confirmed = await this.confirmationManager.show({
                title: 'Delete Export',
                message,
                confirmText: 'Delete',
                cancelText: 'Cancel',
                confirmButtonClass: 'btn-danger'
            });
            if (confirmed) {
                await this.deleteExport(exportRow.id);
            }
        } else {
            if (confirm(message)) {
                await this.deleteExport(exportRow.id);
            }
        }
    }

    /**
     * Delete an export
     */
    private async deleteExport(exportId: number): Promise<void> {
        try {
            await this.api.deleteExport(exportId);
            this.toastManager?.success('Export deleted', 'Deleted', 3000);
            await this.loadExports();
        } catch (error) {
            logger.error('Failed to delete export', { error, exportId });
            this.showError('Failed to delete export');
        }
    }

    /**
     * Main render method
     */
    private render(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        if (this.viewMode === 'list') {
            this.renderExportsList(container);
        } else {
            this.renderCreateExport(container);
        }

        this.attachEventListeners();
    }

    /**
     * Render exports list view
     */
    private renderExportsList(container: HTMLElement): void {
        const totalPages = Math.ceil(this.exportsTotal / this.pageSize);

        container.innerHTML = `
            <div class="export-manager" data-testid="exports-list">
                <div class="export-header">
                    <h3>Exports</h3>
                    <button class="btn-primary" id="create-export-btn">
                        + New Export
                    </button>
                </div>

                <div class="export-stats">
                    <span>Total: <strong>${this.exportsTotal}</strong> exports</span>
                </div>

                ${this.exports.length === 0 ? `
                    <div class="empty-state" data-testid="exports-empty">
                        <p>No exports found</p>
                        <p class="empty-hint">Click "New Export" to create one</p>
                    </div>
                ` : `
                    <div class="table-container">
                        <table class="export-table" data-testid="exports-table">
                            <thead>
                                <tr>
                                    ${this.renderSortableHeader('timestamp', 'Date')}
                                    ${this.renderSortableHeader('title', 'Title')}
                                    ${this.renderSortableHeader('export_type', 'Type')}
                                    ${this.renderSortableHeader('file_format', 'Format')}
                                    ${this.renderSortableHeader('file_size', 'Size')}
                                    <th>Status</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                ${this.exports.map(exp => this.renderExportRow(exp)).join('')}
                            </tbody>
                        </table>
                    </div>

                    ${totalPages > 1 ? this.renderPagination(totalPages) : ''}
                `}
            </div>
        `;
    }

    /**
     * Render sortable header
     */
    private renderSortableHeader(column: string, label: string): string {
        const isActive = this.sortColumn === column;
        const arrow = isActive
            ? (this.sortDir === 'asc' ? '&#9650;' : '&#9660;')
            : '';

        return `
            <th class="sortable ${isActive ? 'active' : ''}" data-sort="${column}">
                <span>${label}</span>
                <span class="sort-arrow">${arrow}</span>
            </th>
        `;
    }

    /**
     * Render export row
     */
    private renderExportRow(exp: TautulliExportsTableRow): string {
        const isComplete = exp.complete === 1;
        const statusClass = isComplete ? 'status-complete' : 'status-pending';
        const statusText = isComplete ? 'Complete' : 'Processing';

        return `
            <tr data-export-id="${exp.id}">
                <td>${this.formatDate(exp.timestamp)}</td>
                <td class="title-cell">${this.escapeHtml(exp.title || exp.export_type)}</td>
                <td>${this.escapeHtml(exp.export_type)}</td>
                <td><span class="format-badge">${exp.file_format.toUpperCase()}</span></td>
                <td>${this.formatFileSize(exp.file_size)}</td>
                <td><span class="status-badge ${statusClass}">${statusText}</span></td>
                <td class="actions-cell">
                    ${isComplete ? `
                        <button class="btn-icon btn-download" data-export-id="${exp.id}" title="Download">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                                <polyline points="7 10 12 15 17 10"/>
                                <line x1="12" y1="15" x2="12" y2="3"/>
                            </svg>
                        </button>
                    ` : ''}
                    <button class="btn-icon btn-delete" data-export-id="${exp.id}" title="Delete">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="3 6 5 6 21 6"/>
                            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                        </svg>
                    </button>
                </td>
            </tr>
        `;
    }

    /**
     * Render pagination
     */
    private renderPagination(totalPages: number): string {
        return `
            <div class="pagination" data-testid="exports-pagination">
                <button class="page-btn" data-page="${this.currentPage - 1}"
                        ${this.currentPage === 0 ? 'disabled' : ''}>
                    &laquo;
                </button>
                <span class="page-info">Page ${this.currentPage + 1} of ${totalPages}</span>
                <button class="page-btn" data-page="${this.currentPage + 1}"
                        ${this.currentPage >= totalPages - 1 ? 'disabled' : ''}>
                    &raquo;
                </button>
            </div>
        `;
    }

    /**
     * Render create export view
     */
    private renderCreateExport(container: HTMLElement): void {
        container.innerHTML = `
            <div class="export-manager" data-testid="create-export">
                <div class="export-header">
                    <button class="btn-back" id="back-to-list">
                        &larr; Back
                    </button>
                    <h3>Create Export</h3>
                </div>

                <div class="create-form">
                    <div class="form-group">
                        <label for="export-library">Library</label>
                        <select id="export-library" class="form-select">
                            ${this.libraries.map(lib => `
                                <option value="${lib.section_id}"
                                        data-type="${lib.section_type || 'movie'}"
                                        ${this.selectedLibrary === lib.section_id ? 'selected' : ''}>
                                    ${this.escapeHtml(lib.section_name)}
                                </option>
                            `).join('')}
                        </select>
                    </div>

                    <div class="form-group">
                        <label for="export-format">Format</label>
                        <select id="export-format" class="form-select">
                            <option value="csv" ${this.selectedFormat === 'csv' ? 'selected' : ''}>CSV</option>
                            <option value="json" ${this.selectedFormat === 'json' ? 'selected' : ''}>JSON</option>
                            <option value="xml" ${this.selectedFormat === 'xml' ? 'selected' : ''}>XML</option>
                            <option value="m3u" ${this.selectedFormat === 'm3u' ? 'selected' : ''}>M3U Playlist</option>
                        </select>
                    </div>

                    ${this.exportFields.length > 0 ? `
                        <div class="form-group">
                            <label>Fields to Include</label>
                            <div class="field-selection">
                                <div class="field-actions">
                                    <button class="btn-sm" id="select-all-fields">Select All</button>
                                    <button class="btn-sm" id="clear-all-fields">Clear All</button>
                                </div>
                                <div class="field-list" data-testid="export-fields">
                                    ${this.exportFields.map(field => `
                                        <label class="field-checkbox">
                                            <input type="checkbox" name="export_field" value="${field.field_name}"
                                                   ${this.selectedFields.includes(field.field_name) ? 'checked' : ''}>
                                            <span class="field-name">${this.escapeHtml(field.display_name)}</span>
                                            ${field.description ? `<span class="field-desc">${this.escapeHtml(field.description)}</span>` : ''}
                                        </label>
                                    `).join('')}
                                </div>
                            </div>
                        </div>
                    ` : ''}

                    <div class="form-actions">
                        <button class="btn-secondary" id="cancel-export">Cancel</button>
                        <button class="btn-primary" id="submit-export">Create Export</button>
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Attach event listeners using event delegation for proper cleanup
     */
    private attachEventListeners(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        // Remove existing handlers before attaching new ones
        this.removeEventListeners();

        // Delegated click handler for all clickable elements
        this.containerClickHandler = async (e: Event) => {
            const target = e.target as HTMLElement;
            const clickedElement = target.closest('[id], [data-sort], [data-page], [data-export-id]') as HTMLElement;
            if (!clickedElement) return;

            // Create export button
            if (clickedElement.id === 'create-export-btn') {
                await this.setViewMode('create');
                return;
            }

            // Back to list button
            if (clickedElement.id === 'back-to-list' || clickedElement.id === 'cancel-export') {
                await this.setViewMode('list');
                return;
            }

            // Submit export button
            if (clickedElement.id === 'submit-export') {
                await this.createExport();
                return;
            }

            // Select all fields
            if (clickedElement.id === 'select-all-fields') {
                this.selectedFields = this.exportFields.map(f => f.field_name);
                this.render();
                return;
            }

            // Clear all fields
            if (clickedElement.id === 'clear-all-fields') {
                this.selectedFields = [];
                this.render();
                return;
            }

            // Sortable headers
            const sortColumn = clickedElement.getAttribute('data-sort');
            if (sortColumn) {
                await this.sortBy(sortColumn);
                return;
            }

            // Pagination
            const page = clickedElement.getAttribute('data-page');
            if (page !== null && !clickedElement.hasAttribute('disabled')) {
                await this.goToPage(parseInt(page, 10));
                return;
            }

            // Download buttons
            if (clickedElement.classList.contains('btn-download')) {
                e.stopPropagation();
                const exportId = parseInt(clickedElement.getAttribute('data-export-id') || '0', 10);
                if (exportId) {
                    this.downloadExport(exportId);
                }
                return;
            }

            // Delete buttons
            if (clickedElement.classList.contains('btn-delete')) {
                e.stopPropagation();
                const exportId = parseInt(clickedElement.getAttribute('data-export-id') || '0', 10);
                if (exportId) {
                    const exp = this.exports.find(e => e.id === exportId);
                    if (exp) {
                        await this.confirmDeleteExport(exp);
                    }
                }
            }
        };

        // Delegated change handler for selects and checkboxes
        this.containerChangeHandler = async (e: Event) => {
            const target = e.target as HTMLElement;

            // Library select
            if (target.id === 'export-library') {
                const select = target as HTMLSelectElement;
                this.selectedLibrary = parseInt(select.value, 10);
                const option = select.selectedOptions[0];
                this.selectedMediaType = option.getAttribute('data-type') || 'movie';
                await this.loadExportFields();
                this.render();
                return;
            }

            // Format select
            if (target.id === 'export-format') {
                this.selectedFormat = (target as HTMLSelectElement).value;
                return;
            }

            // Field checkboxes
            if ((target as HTMLInputElement).name === 'export_field') {
                const checkbox = target as HTMLInputElement;
                if (checkbox.checked) {
                    this.selectedFields.push(checkbox.value);
                } else {
                    this.selectedFields = this.selectedFields.filter(f => f !== checkbox.value);
                }
            }
        };

        container.addEventListener('click', this.containerClickHandler);
        container.addEventListener('change', this.containerChangeHandler);
    }

    /**
     * Remove event listeners for cleanup
     */
    private removeEventListeners(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        if (this.containerClickHandler) {
            container.removeEventListener('click', this.containerClickHandler);
        }
        if (this.containerChangeHandler) {
            container.removeEventListener('change', this.containerChangeHandler);
        }
    }

    /**
     * Format timestamp to date string
     */
    private formatDate(timestamp: number): string {
        const date = new Date(timestamp * 1000);
        return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }

    /**
     * Format file size
     */
    private formatFileSize(bytes: number): string {
        if (bytes === 0) return '-';
        const units = ['B', 'KB', 'MB', 'GB'];
        let unitIndex = 0;
        let size = bytes;

        while (size >= 1024 && unitIndex < units.length - 1) {
            size /= 1024;
            unitIndex++;
        }

        return `${size.toFixed(1)} ${units[unitIndex]}`;
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
     * Clean up resources
     */
    destroy(): void {
        this.removeEventListeners();
        this.containerClickHandler = null;
        this.containerChangeHandler = null;
        this.initialized = false;
        this.exports = [];
    }
}
