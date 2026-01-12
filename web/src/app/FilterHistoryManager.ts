// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * FilterHistoryManager - Undo/Redo for Filter Changes
 *
 * Tracks filter state changes and provides undo/redo functionality.
 *
 * Features:
 * - Maintains history stack of filter states
 * - Keyboard shortcuts (Ctrl+Z, Ctrl+Shift+Z)
 * - Visual undo/redo buttons
 * - Configurable history limit
 * - Screen reader announcements for state changes
 */

import type { LocationFilter } from '../lib/api';
import { createLogger } from '../lib/logger';

const logger = createLogger('FilterHistoryManager');

/**
 * Snapshot of filter state at a point in time
 */
interface FilterSnapshot {
    days?: number;
    start_date?: string;
    end_date?: string;
    users?: string[];
    media_types?: string[];
    timestamp: number;
}

/**
 * Callback for when filter state should be restored
 */
export type FilterRestoreCallback = (filter: Partial<LocationFilter>) => void;

export class FilterHistoryManager {
    private history: FilterSnapshot[] = [];
    private currentIndex = -1;
    private maxHistory = 50;
    private restoreCallback: FilterRestoreCallback | null = null;
    private isRestoring = false;
    private undoButton: HTMLButtonElement | null = null;
    private redoButton: HTMLButtonElement | null = null;
    private announcerElement: HTMLElement | null = null;

    // Store bound event handler references for cleanup
    private boundHandleKeyDown: (e: KeyboardEvent) => void;
    private boundHandleUndo: () => void;
    private boundHandleRedo: () => void;

    constructor() {
        // Bind handlers for cleanup
        this.boundHandleKeyDown = this.handleKeyboardShortcut.bind(this);
        this.boundHandleUndo = () => this.undo();
        this.boundHandleRedo = () => this.redo();

        this.setupKeyboardShortcuts();
        this.setupUI();
        this.announcerElement = document.getElementById('filter-announcer');
    }

    /**
     * Set callback to restore filter state
     */
    setRestoreCallback(callback: FilterRestoreCallback): void {
        this.restoreCallback = callback;
    }

    /**
     * Record current filter state to history
     * Called whenever filters change
     */
    recordState(filter: LocationFilter): void {
        // Don't record if we're in the middle of restoring
        if (this.isRestoring) {
            return;
        }

        const snapshot: FilterSnapshot = {
            days: filter.days,
            start_date: filter.start_date,
            end_date: filter.end_date,
            users: filter.users ? [...filter.users] : undefined,
            media_types: filter.media_types ? [...filter.media_types] : undefined,
            timestamp: Date.now()
        };

        // Check if this is the same as the current state (no change)
        if (this.currentIndex >= 0) {
            const current = this.history[this.currentIndex];
            if (this.areSnapshotsEqual(current, snapshot)) {
                return;
            }
        }

        // If we're not at the end of history, truncate forward history
        if (this.currentIndex < this.history.length - 1) {
            this.history = this.history.slice(0, this.currentIndex + 1);
        }

        // Add new state
        this.history.push(snapshot);
        this.currentIndex = this.history.length - 1;

        // Enforce max history limit
        if (this.history.length > this.maxHistory) {
            const trimCount = this.history.length - this.maxHistory;
            this.history = this.history.slice(trimCount);
            this.currentIndex = Math.max(0, this.currentIndex - trimCount);
        }

        this.updateButtonStates();
    }

    /**
     * Check if two snapshots are equivalent
     */
    private areSnapshotsEqual(a: FilterSnapshot, b: FilterSnapshot): boolean {
        if (a.days !== b.days) return false;
        if (a.start_date !== b.start_date) return false;
        if (a.end_date !== b.end_date) return false;

        // Compare users arrays
        const aUsers = a.users || [];
        const bUsers = b.users || [];
        if (aUsers.length !== bUsers.length) return false;
        if (!aUsers.every((u, i) => u === bUsers[i])) return false;

        // Compare media_types arrays
        const aMedia = a.media_types || [];
        const bMedia = b.media_types || [];
        if (aMedia.length !== bMedia.length) return false;
        if (!aMedia.every((m, i) => m === bMedia[i])) return false;

        return true;
    }

    /**
     * Undo last filter change
     */
    undo(): boolean {
        if (!this.canUndo()) {
            return false;
        }

        this.currentIndex--;
        this.restoreCurrentState();
        this.announce('Filter change undone');
        return true;
    }

    /**
     * Redo previously undone filter change
     */
    redo(): boolean {
        if (!this.canRedo()) {
            return false;
        }

        this.currentIndex++;
        this.restoreCurrentState();
        this.announce('Filter change redone');
        return true;
    }

    /**
     * Check if undo is available
     */
    canUndo(): boolean {
        return this.currentIndex > 0;
    }

    /**
     * Check if redo is available
     */
    canRedo(): boolean {
        return this.currentIndex < this.history.length - 1;
    }

    /**
     * Restore current history state to filters
     */
    private restoreCurrentState(): void {
        if (this.currentIndex < 0 || this.currentIndex >= this.history.length) {
            return;
        }

        const snapshot = this.history[this.currentIndex];

        if (this.restoreCallback) {
            this.isRestoring = true;
            this.restoreCallback({
                days: snapshot.days,
                start_date: snapshot.start_date,
                end_date: snapshot.end_date,
                users: snapshot.users,
                media_types: snapshot.media_types
            });
            // Small delay to ensure filter change events don't re-record
            setTimeout(() => {
                this.isRestoring = false;
            }, 350); // Slightly longer than debounce
        }

        this.updateButtonStates();
    }

    /**
     * Setup keyboard shortcuts for undo/redo
     */
    private setupKeyboardShortcuts(): void {
        document.addEventListener('keydown', this.boundHandleKeyDown);
    }

    /**
     * Handle keyboard shortcut for undo/redo
     */
    private handleKeyboardShortcut(e: KeyboardEvent): void {
        // Check for Ctrl/Cmd+Z (undo) and Ctrl/Cmd+Shift+Z (redo)
        const isCtrlOrCmd = e.ctrlKey || e.metaKey;

        if (!isCtrlOrCmd || e.key.toLowerCase() !== 'z') {
            return;
        }

        // Don't interfere with text inputs
        const target = e.target as HTMLElement;
        if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
            return;
        }

        e.preventDefault();

        if (e.shiftKey) {
            this.redo();
        } else {
            this.undo();
        }
    }

    /**
     * Setup undo/redo buttons in the UI
     */
    private setupUI(): void {
        // Find or create the undo/redo container
        const filterControls = document.getElementById('filter-controls');
        if (!filterControls) {
            // Try to find a suitable location
            const filterPanel = document.getElementById('filters');
            if (!filterPanel) return;

            // Look for an existing undo/redo container
            let container = document.getElementById('filter-history-controls');
            if (!container) {
                // Create container
                container = document.createElement('div');
                container.id = 'filter-history-controls';
                container.className = 'filter-history-controls';

                // Insert after filter badges or at start of filter panel
                const badgesContainer = document.getElementById('filter-badges');
                if (badgesContainer) {
                    badgesContainer.after(container);
                } else {
                    filterPanel.insertBefore(container, filterPanel.firstChild);
                }
            }

            // Create undo button
            this.undoButton = document.createElement('button');
            this.undoButton.id = 'filter-undo';
            this.undoButton.className = 'filter-history-btn';
            this.undoButton.type = 'button';
            this.undoButton.disabled = true;
            this.undoButton.setAttribute('aria-label', 'Undo filter change (Ctrl+Z)');
            this.undoButton.setAttribute('title', 'Undo (Ctrl+Z)');
            this.undoButton.innerHTML = '<span class="history-icon">&#x21A9;</span><span class="history-label">Undo</span>';
            this.undoButton.addEventListener('click', this.boundHandleUndo);

            // Create redo button
            this.redoButton = document.createElement('button');
            this.redoButton.id = 'filter-redo';
            this.redoButton.className = 'filter-history-btn';
            this.redoButton.type = 'button';
            this.redoButton.disabled = true;
            this.redoButton.setAttribute('aria-label', 'Redo filter change (Ctrl+Shift+Z)');
            this.redoButton.setAttribute('title', 'Redo (Ctrl+Shift+Z)');
            this.redoButton.innerHTML = '<span class="history-icon">&#x21AA;</span><span class="history-label">Redo</span>';
            this.redoButton.addEventListener('click', this.boundHandleRedo);

            container.appendChild(this.undoButton);
            container.appendChild(this.redoButton);
        } else {
            // Use existing buttons if present
            this.undoButton = document.getElementById('filter-undo') as HTMLButtonElement;
            this.redoButton = document.getElementById('filter-redo') as HTMLButtonElement;

            if (this.undoButton) {
                this.undoButton.addEventListener('click', this.boundHandleUndo);
            }
            if (this.redoButton) {
                this.redoButton.addEventListener('click', this.boundHandleRedo);
            }
        }
    }

    /**
     * Update button enabled/disabled states
     */
    private updateButtonStates(): void {
        if (this.undoButton) {
            this.undoButton.disabled = !this.canUndo();
            this.undoButton.classList.toggle('has-history', this.canUndo());
        }

        if (this.redoButton) {
            this.redoButton.disabled = !this.canRedo();
            this.redoButton.classList.toggle('has-history', this.canRedo());
        }
    }

    /**
     * Announce state change to screen readers
     */
    private announce(message: string): void {
        if (!this.announcerElement) {
            this.announcerElement = document.getElementById('filter-announcer');
        }

        if (this.announcerElement) {
            this.announcerElement.textContent = '';
            setTimeout(() => {
                if (this.announcerElement) {
                    this.announcerElement.textContent = message;
                }
            }, 100);
        }
    }

    /**
     * Clear all history (e.g., on page navigation)
     */
    clearHistory(): void {
        this.history = [];
        this.currentIndex = -1;
        this.updateButtonStates();
    }

    /**
     * Get current history size (for debugging/testing)
     */
    getHistorySize(): number {
        return this.history.length;
    }

    /**
     * Get current position in history (for debugging/testing)
     */
    getCurrentIndex(): number {
        return this.currentIndex;
    }

    /**
     * Cleanup event listeners and DOM elements
     */
    destroy(): void {
        // Remove keyboard listener
        document.removeEventListener('keydown', this.boundHandleKeyDown);

        // Remove button listeners
        if (this.undoButton) {
            this.undoButton.removeEventListener('click', this.boundHandleUndo);
        }
        if (this.redoButton) {
            this.redoButton.removeEventListener('click', this.boundHandleRedo);
        }

        // Remove buttons if we created them
        const container = document.getElementById('filter-history-controls');
        if (container) {
            container.remove();
        }

        // Clear references
        this.undoButton = null;
        this.redoButton = null;
        this.announcerElement = null;
        this.restoreCallback = null;

        logger.info('FilterHistoryManager destroyed');
    }
}
