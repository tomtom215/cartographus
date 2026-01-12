// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Accessibility utilities for screen reader support
 *
 * Provides functions to:
 * - Announce loading states
 * - Announce filter changes
 * - Announce chart data
 * - Manage focus
 *
 * Reference: UI/UX Audit Tasks
 * @see /docs/working/UI_UX_AUDIT.md
 */

/**
 * Announcement priority levels
 */
export type AnnouncementPriority = 'polite' | 'assertive';

/**
 * Global live region elements for screen reader announcements
 */
let loadingAnnouncer: HTMLElement | null = null;
let statusAnnouncer: HTMLElement | null = null;

/**
 * Initialize accessibility announcer elements
 * Creates hidden live regions for screen reader announcements
 */
export function initializeAccessibilityAnnouncers(): void {
    // Loading state announcer (assertive for important state changes)
    if (!document.getElementById('loading-announcer')) {
        loadingAnnouncer = document.createElement('div');
        loadingAnnouncer.id = 'loading-announcer';
        loadingAnnouncer.className = 'visually-hidden';
        loadingAnnouncer.setAttribute('role', 'status');
        loadingAnnouncer.setAttribute('aria-live', 'assertive');
        loadingAnnouncer.setAttribute('aria-atomic', 'true');
        document.body.appendChild(loadingAnnouncer);
    } else {
        loadingAnnouncer = document.getElementById('loading-announcer');
    }

    // General status announcer (polite for non-urgent updates)
    if (!document.getElementById('status-announcer')) {
        statusAnnouncer = document.createElement('div');
        statusAnnouncer.id = 'status-announcer';
        statusAnnouncer.className = 'visually-hidden';
        statusAnnouncer.setAttribute('role', 'status');
        statusAnnouncer.setAttribute('aria-live', 'polite');
        statusAnnouncer.setAttribute('aria-atomic', 'true');
        document.body.appendChild(statusAnnouncer);
    } else {
        statusAnnouncer = document.getElementById('status-announcer');
    }
}

/**
 * Announce a message to screen readers
 * @param message - The message to announce
 * @param priority - 'polite' (default) or 'assertive' for urgent announcements
 */
export function announceToScreenReader(message: string, priority: AnnouncementPriority = 'polite'): void {
    const announcer = priority === 'assertive' ? loadingAnnouncer : statusAnnouncer;

    // Fall back to looking up element if not initialized
    const targetAnnouncer = announcer || document.getElementById(
        priority === 'assertive' ? 'loading-announcer' : 'status-announcer'
    );

    if (!targetAnnouncer) {
        // Initialize if needed
        initializeAccessibilityAnnouncers();
        return announceToScreenReader(message, priority);
    }

    // Clear and update to trigger announcement
    targetAnnouncer.textContent = '';
    requestAnimationFrame(() => {
        targetAnnouncer.textContent = message;
    });
}

/**
 * Announce loading start
 * @param context - What is being loaded (e.g., 'map data', 'analytics')
 */
export function announceLoadingStart(context: string = 'data'): void {
    announceToScreenReader(`Loading ${context}`, 'assertive');
}

/**
 * Announce loading complete
 * @param context - What was loaded
 * @param itemCount - Optional count of items loaded
 */
export function announceLoadingComplete(context: string = 'data', itemCount?: number): void {
    let message = `${context} loaded`;
    if (itemCount !== undefined) {
        message = `${context} loaded: ${itemCount.toLocaleString()} ${itemCount === 1 ? 'item' : 'items'}`;
    }
    announceToScreenReader(message, 'polite');
}

/**
 * Announce loading error
 * @param context - What failed to load
 * @param error - Optional error message
 */
export function announceLoadingError(context: string = 'data', error?: string): void {
    const message = error
        ? `Error loading ${context}: ${error}`
        : `Error loading ${context}. Please try again.`;
    announceToScreenReader(message, 'assertive');
}

/**
 * Announce data refresh
 * @param context - What was refreshed
 */
export function announceDataRefresh(context: string = 'data'): void {
    announceToScreenReader(`${context} refreshed`, 'polite');
}

/**
 * Announce filter change
 * @param filterType - Type of filter changed
 * @param value - New filter value or description
 */
export function announceFilterChange(filterType: string, value: string): void {
    announceToScreenReader(`${filterType} filter set to ${value}`, 'polite');
}

/**
 * Announce filter cleared
 * @param filterType - Type of filter cleared, or 'all' for all filters
 */
export function announceFilterCleared(filterType: string = 'all'): void {
    const message = filterType === 'all'
        ? 'All filters cleared'
        : `${filterType} filter cleared`;
    announceToScreenReader(message, 'polite');
}

/**
 * Announce view change
 * @param viewName - Name of the new view
 */
export function announceViewChange(viewName: string): void {
    announceToScreenReader(`Switched to ${viewName} view`, 'polite');
}

/**
 * Announce chart data for screen readers
 * @param chartTitle - Title of the chart
 * @param dataDescription - Description of the data being shown
 */
export function announceChartData(chartTitle: string, dataDescription: string): void {
    announceToScreenReader(`${chartTitle}: ${dataDescription}`, 'polite');
}

/**
 * Manage focus for view switches
 * Moves focus to the main content of the new view
 * @param targetElement - Element to focus, or selector string
 */
export function manageFocusOnViewChange(targetElement: HTMLElement | string): void {
    let element: HTMLElement | null;

    if (typeof targetElement === 'string') {
        element = document.querySelector(targetElement);
    } else {
        element = targetElement;
    }

    if (element) {
        // Ensure element is focusable
        if (!element.hasAttribute('tabindex')) {
            element.setAttribute('tabindex', '-1');
        }

        // Small delay to ensure DOM is ready
        requestAnimationFrame(() => {
            element?.focus();
        });
    }
}

/**
 * Get current focus trap boundaries for modal dialogs
 * @param container - Container element of the modal
 * @returns Object with first and last focusable elements
 */
export function getFocusTrapBoundaries(container: HTMLElement): {
    first: HTMLElement | null;
    last: HTMLElement | null;
} {
    const focusableSelectors = [
        'button:not([disabled])',
        'a[href]',
        'input:not([disabled])',
        'select:not([disabled])',
        'textarea:not([disabled])',
        '[tabindex]:not([tabindex="-1"])',
    ].join(', ');

    const focusableElements = container.querySelectorAll<HTMLElement>(focusableSelectors);

    return {
        first: focusableElements[0] || null,
        last: focusableElements[focusableElements.length - 1] || null,
    };
}

/**
 * Create focus trap for modal dialogs
 * @param container - Container element of the modal
 * @returns Cleanup function to remove trap
 */
export function createFocusTrap(container: HTMLElement): () => void {
    const handleKeyDown = (e: KeyboardEvent) => {
        if (e.key !== 'Tab') return;

        const { first, last } = getFocusTrapBoundaries(container);

        if (!first || !last) return;

        if (e.shiftKey && document.activeElement === first) {
            e.preventDefault();
            last.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
            e.preventDefault();
            first.focus();
        }
    };

    container.addEventListener('keydown', handleKeyDown);

    // Focus first element
    const { first } = getFocusTrapBoundaries(container);
    first?.focus();

    // Return cleanup function
    return () => {
        container.removeEventListener('keydown', handleKeyDown);
    };
}

/**
 * Generate accessible description for chart data
 * @param chartType - Type of chart (bar, line, pie, etc.)
 * @param dataPoints - Number of data points
 * @param summary - Optional summary statistics
 */
export function generateChartDescription(
    chartType: string,
    dataPoints: number,
    summary?: { min?: number; max?: number; avg?: number }
): string {
    let description = `${chartType} chart with ${dataPoints} data points.`;

    if (summary) {
        const parts: string[] = [];
        if (summary.min !== undefined) parts.push(`minimum: ${summary.min.toLocaleString()}`);
        if (summary.max !== undefined) parts.push(`maximum: ${summary.max.toLocaleString()}`);
        if (summary.avg !== undefined) parts.push(`average: ${summary.avg.toLocaleString()}`);

        if (parts.length > 0) {
            description += ` ${parts.join(', ')}.`;
        }
    }

    return description;
}
