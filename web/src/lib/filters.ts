// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * FilterManager - Handles all filter state management and URL synchronization
 *
 * Responsibilities:
 * - Filter data loading (users, media types)
 * - Filter dropdown population
 * - Filter state building from UI elements
 * - Debounced filter updates (300ms)
 * - URL query parameter synchronization
 * - Filter restoration from URL on page load
 * - Multi-select feedback (visual indicators on active filters)
 *
 * Extracted from index.ts to reduce complexity (refactoring v1.44)
 */

import type { API, LocationFilter } from './api';
import { createLogger } from './logger';

const logger = createLogger('Filters');

/**
 * Callback type for when filters change and data needs to be reloaded
 */
export type FilterChangeCallback = () => Promise<void>;

/**
 * Filter DOM elements interface
 */
interface FilterElements {
    filterDays: HTMLSelectElement | null;
    filterStartDate: HTMLInputElement | null;
    filterEndDate: HTMLInputElement | null;
    filterUsers: HTMLSelectElement | null;
    filterMediaTypes: HTMLSelectElement | null;
}

/**
 * Date range result from quick date calculations
 */
interface DateRange {
    startDate: Date;
    endDate: Date;
}

export class FilterManager {
    private api: API;
    private availableUsers: string[] = [];
    private availableMediaTypes: string[] = [];
    private filterDebounceTimer: number | null = null;
    private onFilterChangeCallback: FilterChangeCallback | null = null;
    private announcerElement: HTMLElement | null = null;
    /** Container for filter badges */
    private badgesContainer: HTMLElement | null = null;

    constructor(api: API) {
        this.api = api;
        this.announcerElement = document.getElementById('filter-announcer');
        this.badgesContainer = document.getElementById('filter-badges');
    }

    /**
     * Get all filter DOM elements in one call to reduce repetition
     */
    private getFilterElements(): FilterElements {
        return {
            filterDays: document.getElementById('filter-days') as HTMLSelectElement,
            filterStartDate: document.getElementById('filter-start-date') as HTMLInputElement,
            filterEndDate: document.getElementById('filter-end-date') as HTMLInputElement,
            filterUsers: document.getElementById('filter-users') as HTMLSelectElement,
            filterMediaTypes: document.getElementById('filter-media-types') as HTMLSelectElement,
        };
    }

    /**
     * Announce filter change to screen readers
     * Uses aria-live region to notify users of filter state changes
     */
    private announceFilterChange(message: string): void {
        if (!this.announcerElement) {
            this.announcerElement = document.getElementById('filter-announcer');
        }

        if (this.announcerElement) {
            // Clear and update to trigger announcement
            this.announcerElement.textContent = '';
            setTimeout(() => {
                if (this.announcerElement) {
                    this.announcerElement.textContent = message;
                }
            }, 100);
        }
    }

    /**
     * Build human-readable summary of current filter state for announcements
     */
    private buildFilterSummary(): string {
        const filter = this.buildFilter();
        const parts: string[] = [];

        if (filter.days) {
            parts.push(`last ${filter.days} days`);
        }
        if (filter.start_date && filter.end_date) {
            const start = new Date(filter.start_date).toLocaleDateString();
            const end = new Date(filter.end_date).toLocaleDateString();
            parts.push(`${start} to ${end}`);
        } else if (filter.start_date) {
            const start = new Date(filter.start_date).toLocaleDateString();
            parts.push(`from ${start}`);
        }
        if (filter.users && filter.users.length > 0) {
            parts.push(`user: ${filter.users.join(', ')}`);
        }
        if (filter.media_types && filter.media_types.length > 0) {
            parts.push(`media: ${filter.media_types.join(', ')}`);
        }

        if (parts.length === 0) {
            return 'Filters applied: all data';
        }

        return `Filters applied: ${parts.join(', ')}. Loading data.`;
    }

    /**
     * Set the callback to be invoked when filters change
     * This callback should reload all data with the new filter
     */
    setFilterChangeCallback(callback: FilterChangeCallback): void {
        this.onFilterChangeCallback = callback;
    }

    /**
     * Load filter data from API (users and media types)
     */
    async loadFiltersData(): Promise<void> {
        try {
            const [users, mediaTypes] = await Promise.all([
                this.api.getUsers(),
                this.api.getMediaTypes()
            ]);

            this.availableUsers = users;
            this.availableMediaTypes = mediaTypes;

            this.populateFilterDropdowns();
        } catch (error) {
            logger.error('Failed to load filter data:', error);
        }
    }

    /**
     * Helper to populate a single dropdown with options
     */
    private populateDropdown(select: HTMLSelectElement | null, items: string[], defaultLabel: string): void {
        if (!select || items.length === 0) return;

        select.innerHTML = `<option value="">${defaultLabel}</option>`;
        items.forEach(item => {
            const option = document.createElement('option');
            option.value = item;
            option.textContent = item;
            select.appendChild(option);
        });
    }

    /**
     * Populate filter dropdown UI elements with available values
     */
    private populateFilterDropdowns(): void {
        const { filterUsers, filterMediaTypes } = this.getFilterElements();
        this.populateDropdown(filterUsers, this.availableUsers, 'All Users');
        this.populateDropdown(filterMediaTypes, this.availableMediaTypes, 'All Media Types');
    }

    /**
     * Build a LocationFilter object from current UI filter values
     */
    buildFilter(): LocationFilter {
        const filter: LocationFilter = { limit: 1000 };
        const { filterDays, filterStartDate, filterEndDate, filterUsers, filterMediaTypes } = this.getFilterElements();

        // Date range handling: custom date takes precedence over preset days
        if (filterStartDate?.value) {
            filter.start_date = new Date(filterStartDate.value).toISOString();
        } else if (filterDays?.value) {
            const days = parseInt(filterDays.value, 10);
            if (days > 0) {
                filter.days = days;
            }
        }

        if (filterEndDate?.value) {
            filter.end_date = new Date(filterEndDate.value).toISOString();
        }

        if (filterUsers?.value) {
            filter.users = [filterUsers.value];
        }

        if (filterMediaTypes?.value) {
            filter.media_types = [filterMediaTypes.value];
        }

        return filter;
    }

    /**
     * Handle filter change with 300ms debounce
     * Updates URL, announces to screen readers, updates badges,
     * updates active states, and triggers data reload
     */
    onFilterChange(): void {
        if (this.filterDebounceTimer !== null) {
            window.clearTimeout(this.filterDebounceTimer);
        }

        this.filterDebounceTimer = window.setTimeout(() => {
            this.updateURLFromFilters();

            // Update filter badges to reflect current state
            this.updateFilterBadges();

            // Update visual active states on filter elements
            this.updateFilterActiveStates();

            // Announce filter change to screen readers
            const summary = this.buildFilterSummary();
            this.announceFilterChange(summary);

            if (this.onFilterChangeCallback) {
                this.onFilterChangeCallback();
            }
            this.filterDebounceTimer = null;
        }, 300);
    }

    /**
     * Clear all filters to defaults (90 days)
     * Announces reset to screen readers, clears badges, resets active states
     */
    clearFilters(): void {
        const { filterDays, filterStartDate, filterEndDate, filterUsers, filterMediaTypes } = this.getFilterElements();

        if (filterDays) filterDays.value = '90';
        if (filterStartDate) filterStartDate.value = '';
        if (filterEndDate) filterEndDate.value = '';
        if (filterUsers) filterUsers.value = '';
        if (filterMediaTypes) filterMediaTypes.value = '';

        this.updateURLFromFilters();
        this.updateFilterBadges();
        this.updateFilterActiveStates();
        this.updateQuickDateActiveState('');

        this.announceFilterChange('Filters cleared. Showing last 90 days of data.');

        if (this.onFilterChangeCallback) {
            this.onFilterChangeCallback();
        }
    }

    /**
     * Sync current filter values to URL query parameters
     */
    private updateURLFromFilters(): void {
        const filter = this.buildFilter();
        const params = new URLSearchParams();

        if (filter.days) params.set('days', filter.days.toString());
        if (filter.start_date) params.set('start_date', filter.start_date);
        if (filter.end_date) params.set('end_date', filter.end_date);
        if (filter.users && filter.users.length > 0) params.set('users', filter.users.join(','));
        if (filter.media_types && filter.media_types.length > 0) params.set('media_types', filter.media_types.join(','));

        const newURL = params.toString() ? `${window.location.pathname}?${params}` : window.location.pathname;
        window.history.replaceState({}, '', newURL);
    }

    /**
     * Helper to set a date input from URL parameter
     */
    private setDateFromParam(element: HTMLInputElement | null, paramValue: string): void {
        if (!element) return;
        const date = new Date(paramValue);
        element.value = date.toISOString().split('T')[0];
    }

    /**
     * Helper to set a select from URL parameter (handles comma-separated values)
     */
    private setSelectFromParam(element: HTMLSelectElement | null, paramValue: string): void {
        if (!element) return;
        const values = paramValue.split(',');
        if (values.length === 1) {
            element.value = values[0];
        }
    }

    /**
     * Load filter values from URL query parameters on page load
     */
    loadDataFromURL(): void {
        const params = new URLSearchParams(window.location.search);
        const { filterDays, filterStartDate, filterEndDate, filterUsers, filterMediaTypes } = this.getFilterElements();

        if (params.has('days') && filterDays) {
            filterDays.value = params.get('days')!;
        }

        if (params.has('start_date')) {
            this.setDateFromParam(filterStartDate, params.get('start_date')!);
        }

        if (params.has('end_date')) {
            this.setDateFromParam(filterEndDate, params.get('end_date')!);
        }

        if (params.has('users')) {
            this.setSelectFromParam(filterUsers, params.get('users')!);
        }

        if (params.has('media_types')) {
            this.setSelectFromParam(filterMediaTypes, params.get('media_types')!);
        }
    }

    /**
     * Initialize quick date picker buttons
     * Sets up event listeners for Today, Yesterday, This Week, This Month buttons
     */
    setupQuickDateButtons(): void {
        const todayBtn = document.getElementById('quick-date-today');
        const yesterdayBtn = document.getElementById('quick-date-yesterday');
        const weekBtn = document.getElementById('quick-date-week');
        const monthBtn = document.getElementById('quick-date-month');

        todayBtn?.addEventListener('click', () => this.setQuickDate('today'));
        yesterdayBtn?.addEventListener('click', () => this.setQuickDate('yesterday'));
        weekBtn?.addEventListener('click', () => this.setQuickDate('week'));
        monthBtn?.addEventListener('click', () => this.setQuickDate('month'));
    }

    /**
     * Calculate date range for quick date options
     */
    private calculateQuickDateRange(option: 'today' | 'yesterday' | 'week' | 'month'): DateRange {
        const today = new Date();

        const dateCalculators: Record<string, () => DateRange> = {
            today: () => ({
                startDate: new Date(today),
                endDate: new Date(today)
            }),
            yesterday: () => {
                const date = new Date(today);
                date.setDate(date.getDate() - 1);
                return { startDate: date, endDate: new Date(date) };
            },
            week: () => {
                const startDate = new Date(today);
                const dayOfWeek = startDate.getDay();
                startDate.setDate(startDate.getDate() - dayOfWeek);
                return { startDate, endDate: new Date(today) };
            },
            month: () => ({
                startDate: new Date(today.getFullYear(), today.getMonth(), 1),
                endDate: new Date(today)
            })
        };

        return dateCalculators[option]();
    }

    /**
     * Format date as YYYY-MM-DD string
     */
    private formatDateForInput(date: Date): string {
        return date.toISOString().split('T')[0];
    }

    /**
     * Set date range based on quick select option
     */
    private setQuickDate(option: 'today' | 'yesterday' | 'week' | 'month'): void {
        const { filterDays, filterStartDate, filterEndDate } = this.getFilterElements();

        if (!filterStartDate || !filterEndDate) return;

        const { startDate, endDate } = this.calculateQuickDateRange(option);

        filterStartDate.value = this.formatDateForInput(startDate);
        filterEndDate.value = this.formatDateForInput(endDate);

        if (filterDays) {
            filterDays.value = '';
        }

        this.updateQuickDateActiveState(option);
        this.onFilterChange();
    }

    /**
     * Update the active state of quick date buttons
     */
    private updateQuickDateActiveState(activeOption: string): void {
        const buttons = ['today', 'yesterday', 'week', 'month'];
        buttons.forEach(btn => {
            const element = document.getElementById(`quick-date-${btn}`);
            if (element) {
                element.classList.toggle('active', btn === activeOption);
            }
        });
    }

    // =========================================================================
    // Filter Badges/Chips
    // Displays active filters as removable badges for better UX
    // =========================================================================

    /**
     * Update filter badges to reflect current filter state
     * Creates badge elements for each active filter
     */
    updateFilterBadges(): void {
        if (!this.badgesContainer) {
            this.badgesContainer = document.getElementById('filter-badges');
        }
        if (!this.badgesContainer) return;

        // Clear existing badges but preserve placeholder for accessibility
        this.badgesContainer.innerHTML = '<span class="filter-badges-placeholder visually-hidden" aria-hidden="true">No active filters</span>';

        const filter = this.buildFilter();

        // Days filter badge (only show if not default 90 days)
        if (filter.days && filter.days !== 90) {
            this.createBadge('days', 'Time', `Last ${filter.days} days`);
        }

        // Custom date range badge
        if (filter.start_date) {
            const startDate = new Date(filter.start_date).toLocaleDateString();
            const endDate = filter.end_date ? new Date(filter.end_date).toLocaleDateString() : 'Now';
            this.createBadge('dateRange', 'Dates', `${startDate} - ${endDate}`);
        }

        // User filter badge
        if (filter.users && filter.users.length > 0) {
            this.createBadge('user', 'User', filter.users[0]);
        }

        // Media type filter badge
        if (filter.media_types && filter.media_types.length > 0) {
            this.createBadge('mediaType', 'Media', filter.media_types[0]);
        }
    }

    /**
     * Create a single filter badge element
     */
    private createBadge(filterType: string, label: string, value: string): void {
        if (!this.badgesContainer) return;

        const badge = document.createElement('div');
        badge.className = 'filter-badge';
        badge.setAttribute('data-filter-type', filterType);

        const labelSpan = document.createElement('span');
        labelSpan.className = 'filter-badge-label';
        labelSpan.textContent = label;

        const valueSpan = document.createElement('span');
        valueSpan.className = 'filter-badge-value';
        valueSpan.textContent = value;

        const removeButton = document.createElement('button');
        removeButton.className = 'filter-badge-remove';
        removeButton.setAttribute('type', 'button');
        removeButton.setAttribute('aria-label', `Remove ${label} filter: ${value}`);
        removeButton.innerHTML = '&times;';

        removeButton.addEventListener('click', () => this.removeBadgeFilter(filterType, badge));
        removeButton.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                this.removeBadgeFilter(filterType, badge);
            }
        });

        badge.appendChild(labelSpan);
        badge.appendChild(valueSpan);
        badge.appendChild(removeButton);

        this.badgesContainer.appendChild(badge);
    }

    /**
     * Helper to clear the days filter
     */
    private clearDaysFilter(elements: FilterElements): void {
        if (elements.filterDays) {
            elements.filterDays.value = '90';
        }
    }

    /**
     * Helper to clear the date range filter
     */
    private clearDateRangeFilter(elements: FilterElements): void {
        if (elements.filterStartDate) elements.filterStartDate.value = '';
        if (elements.filterEndDate) elements.filterEndDate.value = '';
        if (elements.filterDays) elements.filterDays.value = '90';
        this.updateQuickDateActiveState('');
    }

    /**
     * Helper to clear the user filter
     */
    private clearUserFilter(elements: FilterElements): void {
        if (elements.filterUsers) {
            elements.filterUsers.value = '';
        }
    }

    /**
     * Helper to clear the media type filter
     */
    private clearMediaTypeFilter(elements: FilterElements): void {
        if (elements.filterMediaTypes) {
            elements.filterMediaTypes.value = '';
        }
    }

    /**
     * Remove a specific filter when badge is clicked
     */
    private removeBadgeFilter(filterType: string, badgeElement: HTMLElement): void {
        badgeElement.classList.add('removing');

        setTimeout(() => {
            const elements = this.getFilterElements();

            const filterClearers: Record<string, () => void> = {
                days: () => this.clearDaysFilter(elements),
                dateRange: () => this.clearDateRangeFilter(elements),
                user: () => this.clearUserFilter(elements),
                mediaType: () => this.clearMediaTypeFilter(elements)
            };

            const clearer = filterClearers[filterType];
            if (clearer) {
                clearer();
            }

            this.announceFilterChange(`${filterType} filter removed`);
            badgeElement.remove();
            this.onFilterChange();
        }, 200);
    }

    /**
     * Initialize filter badges and active states on page load
     * Call this after loading initial filter values
     */
    initFilterBadges(): void {
        this.updateFilterBadges();
        this.updateFilterActiveStates();
    }

    // =========================================================================
    // Multi-Select Feedback
    // Visual indicators on filter dropdowns when selections are active
    // =========================================================================

    /**
     * Helper to update active state for a single filter element
     */
    private updateElementActiveState(element: HTMLElement | null, isActive: boolean): void {
        if (!element) return;
        element.classList.toggle('filter-active', isActive);
        element.closest('.filter-group')?.classList.toggle('has-selection', isActive);
    }

    /**
     * Update visual state of filter dropdowns to show active selections
     * Adds .filter-active class to selects and .has-selection to filter groups
     */
    updateFilterActiveStates(): void {
        const { filterDays, filterStartDate, filterEndDate, filterUsers, filterMediaTypes } = this.getFilterElements();

        this.updateElementActiveState(filterDays, !!(filterDays?.value && filterDays.value !== '90'));
        this.updateElementActiveState(filterStartDate, !!filterStartDate?.value);
        this.updateElementActiveState(filterEndDate, !!filterEndDate?.value);
        this.updateElementActiveState(filterUsers, !!filterUsers?.value);
        this.updateElementActiveState(filterMediaTypes, !!filterMediaTypes?.value);
    }

    /**
     * Get count of active filters for display
     */
    getActiveFilterCount(): number {
        const filter = this.buildFilter();
        let count = 0;

        if (filter.days && filter.days !== 90) count++;
        if (filter.start_date) count++;
        if (filter.users && filter.users.length > 0) count++;
        if (filter.media_types && filter.media_types.length > 0) count++;

        return count;
    }

    /**
     * Set date range from external source (e.g., chart brush selection)
     * Used by DateRangeBrushManager to apply brush-selected date ranges
     *
     * @param startDate - Start date in YYYY-MM-DD format
     * @param endDate - End date in YYYY-MM-DD format
     */
    setDateRange(startDate: string, endDate: string): void {
        const { filterDays, filterStartDate, filterEndDate } = this.getFilterElements();

        if (filterStartDate) filterStartDate.value = startDate;
        if (filterEndDate) filterEndDate.value = endDate;
        if (filterDays) filterDays.value = '';

        this.updateQuickDateActiveState('');
        this.onFilterChange();
    }

    /**
     * Helper to restore a date input value from filter state
     */
    private restoreDateInput(element: HTMLInputElement | null, isoDate?: string): void {
        if (!element) return;

        if (isoDate) {
            const date = new Date(isoDate);
            element.value = this.formatDateForInput(date);
        } else {
            element.value = '';
        }
    }

    /**
     * Helper to restore a select value from array filter
     */
    private restoreSelectFromArray(element: HTMLSelectElement | null, values?: string[]): void {
        if (!element) return;
        element.value = values && values.length > 0 ? values[0] : '';
    }

    /**
     * Restore filter state from a snapshot
     * Used by FilterHistoryManager for undo/redo
     *
     * @param filter - Partial filter state to restore
     */
    restoreFilterState(filter: Partial<LocationFilter>): void {
        const { filterDays, filterStartDate, filterEndDate, filterUsers, filterMediaTypes } = this.getFilterElements();

        if (filterDays) {
            filterDays.value = filter.days?.toString() || '90';
        }

        this.restoreDateInput(filterStartDate, filter.start_date);
        this.restoreDateInput(filterEndDate, filter.end_date);
        this.restoreSelectFromArray(filterUsers, filter.users);
        this.restoreSelectFromArray(filterMediaTypes, filter.media_types);

        this.updateQuickDateActiveState('');
        this.updateURLFromFilters();
        this.updateFilterBadges();
        this.updateFilterActiveStates();

        const summary = this.buildFilterSummary();
        this.announceFilterChange(summary);

        if (this.onFilterChangeCallback) {
            this.onFilterChangeCallback();
        }
    }

    /**
     * Cleanup resources to prevent memory leaks
     * Clears debounce timer and removes event listeners
     */
    destroy(): void {
        // Clear debounce timer
        if (this.filterDebounceTimer !== null) {
            window.clearTimeout(this.filterDebounceTimer);
            this.filterDebounceTimer = null;
        }

        // Clear callback reference
        this.onFilterChangeCallback = null;

        // Clear DOM element references
        this.announcerElement = null;
        this.badgesContainer = null;
    }
}
