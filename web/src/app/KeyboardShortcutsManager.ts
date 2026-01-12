// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * KeyboardShortcutsManager - Handles keyboard shortcuts modal
 *
 * Features:
 * - Shows/hides keyboard shortcuts modal on '?' key press
 * - Accessible modal with focus trapping
 * - Closes on Escape, overlay click, or close button
 * - Global keyboard shortcut listener
 */

import { createLogger } from '../lib/logger';

const logger = createLogger('KeyboardShortcutsManager');

export class KeyboardShortcutsManager {
    private modal: HTMLElement | null;
    private closeButton: HTMLElement | null;
    private isOpen: boolean = false;
    private previouslyFocusedElement: HTMLElement | null = null;

    // Store bound event handler references for cleanup
    private boundHandleKeyDown: (e: KeyboardEvent) => void;
    private boundHandleCloseClick: () => void;
    private boundHandleOverlayClick: (e: MouseEvent) => void;

    constructor() {
        this.modal = document.getElementById('keyboard-shortcuts-modal');
        this.closeButton = this.modal?.querySelector('.modal-close') || null;

        // Bind handlers for cleanup
        this.boundHandleKeyDown = this.handleKeyDown.bind(this);
        this.boundHandleCloseClick = () => this.close();
        this.boundHandleOverlayClick = (e: MouseEvent) => {
            if (e.target === this.modal) {
                this.close();
            }
        };
    }

    /**
     * Initialize the keyboard shortcuts manager
     */
    init(): void {
        this.setupEventListeners();
    }

    /**
     * Set up all event listeners
     */
    private setupEventListeners(): void {
        // Global keyboard listener for '?' key
        document.addEventListener('keydown', this.boundHandleKeyDown);

        // Close button click
        if (this.closeButton) {
            this.closeButton.addEventListener('click', this.boundHandleCloseClick);
        }

        // Overlay click to close
        if (this.modal) {
            this.modal.addEventListener('click', this.boundHandleOverlayClick);
        }
    }

    /**
     * Handle keydown events
     */
    private handleKeyDown(e: KeyboardEvent): void {
        // Don't trigger if user is typing in an input
        if (e.target instanceof HTMLInputElement ||
            e.target instanceof HTMLTextAreaElement ||
            e.target instanceof HTMLSelectElement) {
            return;
        }

        // '?' key opens modal
        if (e.key === '?' && !this.isOpen) {
            e.preventDefault();
            this.open();
            return;
        }

        // Escape closes modal if open
        if (e.key === 'Escape' && this.isOpen) {
            e.preventDefault();
            this.close();
            return;
        }

        // Handle Tab key for focus trapping when modal is open
        if (e.key === 'Tab' && this.isOpen) {
            this.handleTabKey(e);
        }
    }

    /**
     * Open the keyboard shortcuts modal
     */
    open(): void {
        if (!this.modal) return;

        // Store currently focused element to restore later
        this.previouslyFocusedElement = document.activeElement as HTMLElement;

        // Show modal - CSS handles display via visibility/opacity
        // Clear any inline styles that might conflict with CSS
        this.modal.style.removeProperty('visibility');
        this.modal.style.removeProperty('display');
        this.modal.classList.add('visible');
        this.modal.setAttribute('aria-hidden', 'false');
        this.isOpen = true;

        // Focus the close button for accessibility
        setTimeout(() => {
            this.closeButton?.focus();
        }, 100);

        logger.debug('Keyboard shortcuts modal opened');
    }

    /**
     * Close the keyboard shortcuts modal
     */
    close(): void {
        if (!this.modal) return;

        // Hide modal immediately - set inline styles for instant hide
        // This ensures Playwright tests see the modal as hidden immediately
        this.modal.classList.remove('visible');
        this.modal.setAttribute('aria-hidden', 'true');
        this.modal.style.visibility = 'hidden';
        this.modal.style.opacity = '0';
        this.isOpen = false;

        // Restore focus to previously focused element
        if (this.previouslyFocusedElement) {
            this.previouslyFocusedElement.focus();
            this.previouslyFocusedElement = null;
        }

        logger.debug('Keyboard shortcuts modal closed');
    }

    /**
     * Handle Tab key for focus trapping within modal
     */
    private handleTabKey(e: KeyboardEvent): void {
        if (!this.modal) return;

        const focusableElements = this.modal.querySelectorAll(
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
     * Check if modal is currently open
     */
    isModalOpen(): boolean {
        return this.isOpen;
    }

    /**
     * Destroy the manager and clean up all event listeners
     */
    destroy(): void {
        // Remove document-level keyboard listener
        document.removeEventListener('keydown', this.boundHandleKeyDown);

        // Remove close button listener
        if (this.closeButton) {
            this.closeButton.removeEventListener('click', this.boundHandleCloseClick);
            this.closeButton = null;
        }

        // Remove overlay click listener
        if (this.modal) {
            this.modal.removeEventListener('click', this.boundHandleOverlayClick);
        }

        this.modal = null;
        logger.debug('KeyboardShortcutsManager destroyed');
    }
}

export default KeyboardShortcutsManager;
