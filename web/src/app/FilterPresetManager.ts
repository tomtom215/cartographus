// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * FilterPresetManager - Manage saved filter presets
 *
 * Features:
 * - Save current filter settings as named presets
 * - Load and apply saved presets
 * - Delete presets
 * - Rename presets
 * - localStorage persistence
 * - Accessible UI with ARIA attributes
 */

import type { FilterManager } from '../lib/filters';
import type { LocationFilter } from '../lib/api';
import type { ToastManager } from '../lib/toast';
import { createLogger } from '../lib/logger';
import { SafeStorage } from '../lib/utils/SafeStorage';

/**
 * Preset data structure
 */
export interface FilterPreset {
    id: string;
    name: string;
    filter: LocationFilter;
    createdAt: string;
}

/**
 * Filter preset manager configuration
 */
export interface FilterPresetConfig {
    maxPresets: number;
    storageKey: string;
}

const DEFAULT_CONFIG: FilterPresetConfig = {
    maxPresets: 20,
    storageKey: 'filter-presets',
};

const logger = createLogger('FilterPresetManager');

export class FilterPresetManager {
    private config: FilterPresetConfig;
    private presets: FilterPreset[] = [];
    private filterManager: FilterManager | null = null;
    private toastManager: ToastManager | null = null;

    // DOM elements
    private saveButton: HTMLButtonElement | null = null;
    private saveDialog: HTMLElement | null = null;
    private nameInput: HTMLInputElement | null = null;
    private confirmSaveButton: HTMLButtonElement | null = null;
    private cancelSaveButton: HTMLButtonElement | null = null;
    private presetList: HTMLElement | null = null;
    private presetEmptyState: HTMLElement | null = null;
    private previouslyFocusedElement: HTMLElement | null = null;

    // Event handler references for cleanup
    private saveButtonClickHandler: (() => void) | null = null;
    private confirmSaveClickHandler: (() => void) | null = null;
    private cancelSaveClickHandler: (() => void) | null = null;
    private saveDialogClickHandler: ((e: Event) => void) | null = null;
    private nameInputKeydownHandler: ((e: KeyboardEvent) => void) | null = null;
    private documentKeydownHandler: ((e: KeyboardEvent) => void) | null = null;
    private presetListClickHandler: ((e: Event) => void) | null = null;
    private presetListKeydownHandler: ((e: KeyboardEvent) => void) | null = null;

    constructor(config: Partial<FilterPresetConfig> = {}) {
        this.config = { ...DEFAULT_CONFIG, ...config };
        this.loadPresets();
    }

    /**
     * Initialize the preset manager
     */
    init(): void {
        this.bindElements();
        this.setupEventListeners();
        this.renderPresetList();
        logger.debug('FilterPresetManager initialized');
    }

    /**
     * Set the filter manager reference
     */
    setFilterManager(filterManager: FilterManager): void {
        this.filterManager = filterManager;
    }

    /**
     * Set the toast manager reference
     */
    setToastManager(toastManager: ToastManager): void {
        this.toastManager = toastManager;
    }

    /**
     * Bind to DOM elements
     */
    private bindElements(): void {
        this.saveButton = document.getElementById('btn-save-preset') as HTMLButtonElement;
        this.saveDialog = document.getElementById('save-preset-dialog');
        this.nameInput = document.getElementById('preset-name-input') as HTMLInputElement;
        this.confirmSaveButton = document.getElementById('btn-confirm-save-preset') as HTMLButtonElement;
        this.cancelSaveButton = document.getElementById('btn-cancel-save-preset') as HTMLButtonElement;
        this.presetList = document.getElementById('preset-list');
        this.presetEmptyState = document.getElementById('preset-list-empty');
    }

    /**
     * Set up event listeners with stored references for cleanup
     */
    private setupEventListeners(): void {
        // Save button click
        if (this.saveButton) {
            this.saveButtonClickHandler = () => this.openSaveDialog();
            this.saveButton.addEventListener('click', this.saveButtonClickHandler);
        }

        // Confirm save
        if (this.confirmSaveButton) {
            this.confirmSaveClickHandler = () => this.handleSavePreset();
            this.confirmSaveButton.addEventListener('click', this.confirmSaveClickHandler);
        }

        // Cancel save
        if (this.cancelSaveButton) {
            this.cancelSaveClickHandler = () => this.closeSaveDialog();
            this.cancelSaveButton.addEventListener('click', this.cancelSaveClickHandler);
        }

        // Dialog overlay click
        if (this.saveDialog) {
            this.saveDialogClickHandler = (e: Event) => {
                if (e.target === this.saveDialog) {
                    this.closeSaveDialog();
                }
            };
            this.saveDialog.addEventListener('click', this.saveDialogClickHandler);
        }

        // Name input Enter key
        if (this.nameInput) {
            this.nameInputKeydownHandler = (e: KeyboardEvent) => {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    this.handleSavePreset();
                }
            };
            this.nameInput.addEventListener('keydown', this.nameInputKeydownHandler);
        }

        // Global keyboard events for dialog (critical to clean up!)
        this.documentKeydownHandler = (e: KeyboardEvent) => this.handleKeyDown(e);
        document.addEventListener('keydown', this.documentKeydownHandler);

        // Delegated event handlers for preset list (prevents leaks on re-render)
        if (this.presetList) {
            this.presetListClickHandler = (e: Event) => {
                const target = e.target as HTMLElement;

                // Check for delete button
                const deleteBtn = target.closest('[data-action="delete"]') as HTMLElement;
                if (deleteBtn) {
                    e.stopPropagation();
                    const presetItem = deleteBtn.closest('[data-preset-id]') as HTMLElement;
                    const presetId = presetItem?.getAttribute('data-preset-id');
                    if (presetId) {
                        this.deletePreset(presetId);
                    }
                    return;
                }

                // Check for preset item click
                const presetItem = target.closest('[data-preset-id]') as HTMLElement;
                if (presetItem) {
                    const presetId = presetItem.getAttribute('data-preset-id');
                    if (presetId) {
                        this.loadPreset(presetId);
                    }
                }
            };
            this.presetList.addEventListener('click', this.presetListClickHandler);

            this.presetListKeydownHandler = (e: KeyboardEvent) => {
                if (e.key !== 'Enter' && e.key !== ' ') return;

                const target = e.target as HTMLElement;
                const presetItem = target.closest('[data-preset-id]') as HTMLElement;
                if (presetItem) {
                    e.preventDefault();
                    const presetId = presetItem.getAttribute('data-preset-id');
                    if (presetId) {
                        this.loadPreset(presetId);
                    }
                }
            };
            this.presetList.addEventListener('keydown', this.presetListKeydownHandler);
        }
    }

    /**
     * Remove event listeners for cleanup
     */
    private removeEventListeners(): void {
        if (this.saveButton && this.saveButtonClickHandler) {
            this.saveButton.removeEventListener('click', this.saveButtonClickHandler);
        }
        if (this.confirmSaveButton && this.confirmSaveClickHandler) {
            this.confirmSaveButton.removeEventListener('click', this.confirmSaveClickHandler);
        }
        if (this.cancelSaveButton && this.cancelSaveClickHandler) {
            this.cancelSaveButton.removeEventListener('click', this.cancelSaveClickHandler);
        }
        if (this.saveDialog && this.saveDialogClickHandler) {
            this.saveDialog.removeEventListener('click', this.saveDialogClickHandler);
        }
        if (this.nameInput && this.nameInputKeydownHandler) {
            this.nameInput.removeEventListener('keydown', this.nameInputKeydownHandler);
        }
        if (this.documentKeydownHandler) {
            document.removeEventListener('keydown', this.documentKeydownHandler);
        }
        if (this.presetList) {
            if (this.presetListClickHandler) {
                this.presetList.removeEventListener('click', this.presetListClickHandler);
            }
            if (this.presetListKeydownHandler) {
                this.presetList.removeEventListener('keydown', this.presetListKeydownHandler);
            }
        }
    }

    /**
     * Handle global keyboard events
     */
    private handleKeyDown(e: KeyboardEvent): void {
        if (!this.isDialogOpen()) return;

        if (e.key === 'Escape') {
            e.preventDefault();
            this.closeSaveDialog();
        }

        if (e.key === 'Tab') {
            this.handleTabKey(e);
        }
    }

    /**
     * Handle Tab key for focus trapping
     */
    private handleTabKey(e: KeyboardEvent): void {
        if (!this.saveDialog) return;

        const focusableElements = this.saveDialog.querySelectorAll(
            'input:not(:disabled), button:not(:disabled), [tabindex]:not([tabindex="-1"])'
        );

        if (focusableElements.length === 0) return;

        const firstElement = focusableElements[0] as HTMLElement;
        const lastElement = focusableElements[focusableElements.length - 1] as HTMLElement;

        if (e.shiftKey) {
            if (document.activeElement === firstElement) {
                e.preventDefault();
                lastElement.focus();
            }
        } else {
            if (document.activeElement === lastElement) {
                e.preventDefault();
                firstElement.focus();
            }
        }
    }

    /**
     * Check if dialog is open
     */
    isDialogOpen(): boolean {
        return this.saveDialog?.classList.contains('visible') ?? false;
    }

    /**
     * Open the save preset dialog
     */
    openSaveDialog(): void {
        if (!this.saveDialog || !this.nameInput) return;

        // Store currently focused element
        this.previouslyFocusedElement = document.activeElement as HTMLElement;

        // Generate default name
        const defaultName = `Preset ${this.presets.length + 1}`;
        this.nameInput.value = defaultName;

        // Clear any error state
        this.nameInput.classList.remove('error');
        const errorEl = this.saveDialog.querySelector('.preset-name-error');
        if (errorEl) errorEl.textContent = '';

        // Show dialog
        this.saveDialog.style.display = 'flex';
        this.saveDialog.classList.add('visible');
        this.saveDialog.setAttribute('aria-hidden', 'false');

        // Focus name input
        setTimeout(() => {
            this.nameInput?.select();
            this.nameInput?.focus();
        }, 100);
    }

    /**
     * Close the save preset dialog
     */
    closeSaveDialog(): void {
        if (!this.saveDialog) return;

        this.saveDialog.classList.remove('visible');
        this.saveDialog.setAttribute('aria-hidden', 'true');

        setTimeout(() => {
            if (this.saveDialog) {
                this.saveDialog.style.display = 'none';
            }
        }, 200);

        // Restore focus
        if (this.previouslyFocusedElement) {
            this.previouslyFocusedElement.focus();
            this.previouslyFocusedElement = null;
        }
    }

    /**
     * Handle save preset action
     */
    private handleSavePreset(): void {
        if (!this.nameInput || !this.filterManager) return;

        let name = this.nameInput.value.trim();

        // Use default name if empty
        if (!name) {
            name = `Preset ${this.presets.length + 1}`;
        }

        // Check for duplicates
        if (this.presets.some(p => p.name === name)) {
            // Auto-rename with suffix
            let suffix = 2;
            while (this.presets.some(p => p.name === `${name} (${suffix})`)) {
                suffix++;
            }
            name = `${name} (${suffix})`;
        }

        // Check max presets
        if (this.presets.length >= this.config.maxPresets) {
            this.showError(`Maximum ${this.config.maxPresets} presets allowed`);
            return;
        }

        // Get current filter state
        const filter = this.filterManager.buildFilter();

        // Create preset
        const preset: FilterPreset = {
            id: this.generateId(),
            name,
            filter,
            createdAt: new Date().toISOString()
        };

        // Add and save
        this.presets.push(preset);
        this.savePresets();
        this.renderPresetList();

        // Close dialog
        this.closeSaveDialog();

        // Show toast
        this.toastManager?.success(`Preset "${name}" saved`, '', 3000);
    }

    /**
     * Load preset and apply filters
     */
    loadPreset(presetId: string): void {
        const preset = this.presets.find(p => p.id === presetId);
        if (!preset) {
            logger.error('Preset not found:', presetId);
            return;
        }

        this.applyFilter(preset.filter);
        this.toastManager?.info(`Loaded preset "${preset.name}"`, '', 2000);
    }

    /**
     * Apply filter to UI
     */
    private applyFilter(filter: LocationFilter): void {
        // Apply days filter
        const filterDays = document.getElementById('filter-days') as HTMLSelectElement;
        if (filterDays && filter.days) {
            filterDays.value = String(filter.days);
        } else if (filterDays && !filter.start_date) {
            // Reset to default if no days and no custom date
            filterDays.value = '90';
        }

        // Apply date filters
        const filterStartDate = document.getElementById('filter-start-date') as HTMLInputElement;
        if (filterStartDate) {
            if (filter.start_date) {
                filterStartDate.value = filter.start_date.split('T')[0];
            } else {
                filterStartDate.value = '';
            }
        }

        const filterEndDate = document.getElementById('filter-end-date') as HTMLInputElement;
        if (filterEndDate) {
            if (filter.end_date) {
                filterEndDate.value = filter.end_date.split('T')[0];
            } else {
                filterEndDate.value = '';
            }
        }

        // Apply user filter
        const filterUsers = document.getElementById('filter-users') as HTMLSelectElement;
        if (filterUsers) {
            if (filter.users && filter.users.length > 0) {
                filterUsers.value = filter.users[0];
            } else {
                filterUsers.value = '';
            }
        }

        // Apply media type filter
        const filterMediaTypes = document.getElementById('filter-media-types') as HTMLSelectElement;
        if (filterMediaTypes) {
            if (filter.media_types && filter.media_types.length > 0) {
                filterMediaTypes.value = filter.media_types[0];
            } else {
                filterMediaTypes.value = '';
            }
        }

        // Trigger filter change
        if (this.filterManager) {
            this.filterManager.onFilterChange();
        }
    }

    /**
     * Delete a preset
     */
    deletePreset(presetId: string): void {
        const index = this.presets.findIndex(p => p.id === presetId);
        if (index === -1) return;

        const preset = this.presets[index];
        this.presets.splice(index, 1);
        this.savePresets();
        this.renderPresetList();

        this.toastManager?.info(`Preset "${preset.name}" deleted`, '', 2000);
    }

    /**
     * Rename a preset
     */
    renamePreset(presetId: string, newName: string): void {
        const preset = this.presets.find(p => p.id === presetId);
        if (!preset) return;

        const oldName = preset.name;
        preset.name = newName.trim() || oldName;
        this.savePresets();
        this.renderPresetList();

        if (preset.name !== oldName) {
            this.toastManager?.info(`Preset renamed to "${preset.name}"`, '', 2000);
        }
    }

    /**
     * Render the preset list
     */
    private renderPresetList(): void {
        if (!this.presetList) return;

        // Clear existing items (except empty state)
        const existingItems = this.presetList.querySelectorAll('.preset-item');
        existingItems.forEach(item => item.remove());

        // Show/hide empty state
        if (this.presetEmptyState) {
            this.presetEmptyState.style.display = this.presets.length === 0 ? 'block' : 'none';
        }

        // Render presets
        this.presets.forEach(preset => {
            const item = this.createPresetItem(preset);
            this.presetList!.appendChild(item);
        });

        // Update count badge
        this.updateCountBadge();
    }

    /**
     * Create a preset list item
     * Note: Event listeners are handled via delegation in setupEventListeners()
     */
    private createPresetItem(preset: FilterPreset): HTMLElement {
        const item = document.createElement('div');
        item.className = 'preset-item';
        item.setAttribute('data-preset-name', preset.name);
        item.setAttribute('data-preset-id', preset.id);
        item.setAttribute('role', 'button');
        item.setAttribute('tabindex', '0');
        item.setAttribute('aria-label', `Load preset: ${preset.name}`);

        // Name
        const name = document.createElement('span');
        name.className = 'preset-name';
        name.textContent = preset.name;
        item.appendChild(name);

        // Actions container
        const actions = document.createElement('div');
        actions.className = 'preset-actions';

        // Delete button - events handled via delegation
        const deleteBtn = document.createElement('button');
        deleteBtn.type = 'button';
        deleteBtn.className = 'preset-delete-btn';
        deleteBtn.setAttribute('data-action', 'delete');
        deleteBtn.setAttribute('aria-label', `Delete preset: ${preset.name}`);
        deleteBtn.innerHTML = `
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"/>
                <line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
        `;
        actions.appendChild(deleteBtn);

        item.appendChild(actions);

        // Click and keyboard events handled via delegation on presetList

        return item;
    }

    /**
     * Update the preset count badge
     */
    private updateCountBadge(): void {
        const badge = document.querySelector('.preset-count-badge');
        if (badge) {
            if (this.presets.length > 0) {
                badge.textContent = String(this.presets.length);
                (badge as HTMLElement).style.display = 'inline-flex';
            } else {
                (badge as HTMLElement).style.display = 'none';
            }
        }
    }

    /**
     * Show error message in dialog
     */
    private showError(message: string): void {
        const errorEl = this.saveDialog?.querySelector('.preset-name-error');
        if (errorEl) {
            errorEl.textContent = message;
        }
        this.nameInput?.classList.add('error');
    }

    /**
     * Generate unique ID
     */
    private generateId(): string {
        return `preset-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    }

    /**
     * Load presets from localStorage
     */
    private loadPresets(): void {
        const stored = SafeStorage.getJSON<FilterPreset[]>(this.config.storageKey, []);
        if (Array.isArray(stored)) {
            this.presets = stored;
        }
    }

    /**
     * Save presets to localStorage
     */
    private savePresets(): void {
        SafeStorage.setJSON(this.config.storageKey, this.presets);
    }

    /**
     * Get all presets (for testing)
     */
    getPresets(): FilterPreset[] {
        return [...this.presets];
    }

    /**
     * Clean up resources and event listeners
     */
    destroy(): void {
        this.removeEventListeners();
        this.saveButtonClickHandler = null;
        this.confirmSaveClickHandler = null;
        this.cancelSaveClickHandler = null;
        this.saveDialogClickHandler = null;
        this.nameInputKeydownHandler = null;
        this.documentKeydownHandler = null;
        this.presetListClickHandler = null;
        this.presetListKeydownHandler = null;
        this.presets = [];
    }
}

export default FilterPresetManager;
