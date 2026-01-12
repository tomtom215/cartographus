// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * ConfirmationDialogManager - Handles confirmation dialogs for destructive actions
 *
 * Features:
 * - Shows modal confirmation before destructive actions
 * - Focus trapping for accessibility
 * - Keyboard navigation (Escape to close)
 * - Screen reader support with ARIA attributes
 * - Returns Promise to await user decision
 */

import { createLogger } from '../lib/logger';

const logger = createLogger('ConfirmationDialogManager');

export interface ConfirmationOptions {
    title: string;
    message: string;
    confirmText?: string;
    cancelText?: string;
    confirmButtonClass?: string;
}

export class ConfirmationDialogManager {
    private dialog: HTMLElement | null;
    private titleElement: HTMLElement | null;
    private messageElement: HTMLElement | null;
    private confirmButton: HTMLButtonElement | null;
    private cancelButton: HTMLButtonElement | null;
    private isOpen: boolean = false;
    private previouslyFocusedElement: HTMLElement | null = null;
    private resolvePromise: ((confirmed: boolean) => void) | null = null;
    // AbortController for clean event listener removal
    private abortController: AbortController | null = null;
    // Track active timeouts
    private activeTimeouts: Set<ReturnType<typeof setTimeout>> = new Set();

    constructor() {
        this.dialog = document.getElementById('confirmation-dialog');
        this.titleElement = document.getElementById('confirmation-dialog-title');
        this.messageElement = document.getElementById('confirmation-dialog-message');
        this.confirmButton = document.getElementById('confirmation-dialog-confirm') as HTMLButtonElement;
        this.cancelButton = document.getElementById('confirmation-dialog-cancel') as HTMLButtonElement;

        this.setupEventListeners();
    }

    /**
     * Initialize the confirmation dialog manager
     */
    init(): void {
        // Already initialized in constructor
        logger.log('ConfirmationDialogManager initialized');
    }

    /**
     * Set up all event listeners with AbortController for clean removal
     */
    private setupEventListeners(): void {
        // Create AbortController for cleanup
        this.abortController = new AbortController();
        const signal = this.abortController.signal;

        // Confirm button click
        this.confirmButton?.addEventListener('click', () => this.handleConfirm(), { signal });

        // Cancel button click
        this.cancelButton?.addEventListener('click', () => this.handleCancel(), { signal });

        // Overlay click to close
        this.dialog?.addEventListener('click', (e) => {
            if (e.target === this.dialog) {
                this.handleCancel();
            }
        }, { signal });

        // Keyboard events
        document.addEventListener('keydown', (e) => this.handleKeyDown(e), { signal });
    }

    /**
     * Handle keyboard events when dialog is open
     */
    private handleKeyDown(e: KeyboardEvent): void {
        if (!this.isOpen) return;

        // Escape closes dialog
        if (e.key === 'Escape') {
            e.preventDefault();
            this.handleCancel();
            return;
        }

        // Tab key for focus trapping
        if (e.key === 'Tab') {
            this.handleTabKey(e);
        }
    }

    /**
     * Handle Tab key for focus trapping within dialog
     */
    private handleTabKey(e: KeyboardEvent): void {
        if (!this.dialog) return;

        const focusableElements = this.dialog.querySelectorAll(
            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );

        if (focusableElements.length === 0) return;

        const firstElement = focusableElements[0] as HTMLElement;
        const lastElement = focusableElements[focusableElements.length - 1] as HTMLElement;

        if (e.shiftKey) {
            // Shift + Tab: If on first element, go to last
            if (document.activeElement === firstElement) {
                e.preventDefault();
                lastElement.focus();
            }
        } else {
            // Tab: If on last element, go to first
            if (document.activeElement === lastElement) {
                e.preventDefault();
                firstElement.focus();
            }
        }
    }

    /**
     * Show the confirmation dialog with custom options
     * Returns a Promise that resolves to true (confirmed) or false (cancelled)
     */
    show(options: ConfirmationOptions): Promise<boolean> {
        return new Promise((resolve) => {
            if (!this.dialog) {
                // If dialog doesn't exist, auto-confirm for graceful degradation
                resolve(true);
                return;
            }

            this.resolvePromise = resolve;

            // Update dialog content
            if (this.titleElement) {
                this.titleElement.textContent = options.title;
            }
            if (this.messageElement) {
                this.messageElement.textContent = options.message;
            }
            if (this.confirmButton) {
                this.confirmButton.textContent = options.confirmText || 'Confirm';
            }
            if (this.cancelButton) {
                this.cancelButton.textContent = options.cancelText || 'Cancel';
            }

            // Store currently focused element to restore later
            this.previouslyFocusedElement = document.activeElement as HTMLElement;

            // Show dialog
            this.dialog.style.display = 'flex';
            this.dialog.classList.add('visible');
            this.dialog.setAttribute('aria-hidden', 'false');
            this.isOpen = true;

            // Focus the confirm button for accessibility
            const focusTimeoutId = setTimeout(() => {
                this.activeTimeouts.delete(focusTimeoutId);
                this.confirmButton?.focus();
            }, 100);
            this.activeTimeouts.add(focusTimeoutId);
        });
    }

    /**
     * Handle confirm action
     */
    private handleConfirm(): void {
        this.close();
        if (this.resolvePromise) {
            this.resolvePromise(true);
            this.resolvePromise = null;
        }
    }

    /**
     * Handle cancel action
     */
    private handleCancel(): void {
        this.close();
        if (this.resolvePromise) {
            this.resolvePromise(false);
            this.resolvePromise = null;
        }
    }

    /**
     * Close the dialog
     */
    private close(): void {
        if (!this.dialog) return;

        // Hide dialog
        this.dialog.classList.remove('visible');
        this.dialog.setAttribute('aria-hidden', 'true');
        this.isOpen = false;

        // Wait for transition to complete before hiding
        const hideTimeoutId = setTimeout(() => {
            this.activeTimeouts.delete(hideTimeoutId);
            if (this.dialog) {
                this.dialog.style.display = 'none';
            }
        }, 200);
        this.activeTimeouts.add(hideTimeoutId);

        // Restore focus to previously focused element
        if (this.previouslyFocusedElement) {
            this.previouslyFocusedElement.focus();
            this.previouslyFocusedElement = null;
        }
    }

    /**
     * Check if dialog is currently open
     */
    isDialogOpen(): boolean {
        return this.isOpen;
    }

    /**
     * Clean up event listeners and resources
     */
    destroy(): void {
        // Abort all event listeners
        if (this.abortController) {
            this.abortController.abort();
            this.abortController = null;
        }

        // Clear all active timeouts
        for (const timeoutId of this.activeTimeouts) {
            clearTimeout(timeoutId);
        }
        this.activeTimeouts.clear();

        // Reject any pending promise
        if (this.resolvePromise) {
            this.resolvePromise(false);
            this.resolvePromise = null;
        }

        // Clear DOM references
        this.dialog = null;
        this.titleElement = null;
        this.messageElement = null;
        this.confirmButton = null;
        this.cancelButton = null;
        this.previouslyFocusedElement = null;

        logger.log('ConfirmationDialogManager destroyed');
    }
}

export default ConfirmationDialogManager;
