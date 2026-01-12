// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
export type ToastType = 'info' | 'success' | 'warning' | 'error';

/**
 * Toast duration constants for accessibility
 * WCAG 2.2.1 recommends giving users enough time to read content
 * These values are in milliseconds
 */
export const TOAST_DURATION = {
    /** Short duration for brief confirmations (6 seconds) */
    SHORT: 6000,
    /** Default duration for most messages (8 seconds) */
    DEFAULT: 8000,
    /** Long duration for complex messages (12 seconds) */
    LONG: 12000,
    /** Persistent - requires user dismissal */
    PERSISTENT: 0,
} as const;

export interface ToastOptions {
    message: string;
    type?: ToastType;
    duration?: number; // milliseconds, 0 = persistent (default: 8000 for accessibility)
    title?: string;
}

interface Toast extends ToastOptions {
    id: string;
    type: ToastType;
    duration: number;
}

export class ToastManager {
    private container: HTMLElement | null = null;
    private toasts: Map<string, Toast> = new Map();
    private nextId: number = 1;
    /** Track recent messages for deduplication (message -> timestamp) */
    private recentMessages: Map<string, number> = new Map();
    /** Deduplication window in milliseconds (default: 3 seconds) */
    private deduplicationWindow: number = 3000;
    /** Callback to notify notification center of new toasts */
    private notificationCallback: ((type: ToastType, message: string, title?: string) => void) | null = null;

    constructor() {
        this.createContainer();
    }

    /**
     * Set callback to notify notification center of new toasts
     */
    setNotificationCallback(callback: (type: ToastType, message: string, title?: string) => void): void {
        this.notificationCallback = callback;
    }

    /**
     * Create the toast container element
     */
    private createContainer(): void {
        this.container = document.createElement('div');
        this.container.id = 'toast-container';
        this.container.className = 'toast-container';
        this.container.setAttribute('role', 'region');
        this.container.setAttribute('aria-label', 'Notifications');
        this.container.setAttribute('aria-live', 'polite');
        document.body.appendChild(this.container);
    }

    /**
     * Show a toast notification
     * Implements deduplication to prevent identical messages from appearing multiple times
     */
    show(options: ToastOptions): string {
        const messageKey = this.getMessageKey(options);
        const now = Date.now();

        // Check for duplicate message within deduplication window
        const lastShown = this.recentMessages.get(messageKey);
        if (lastShown && (now - lastShown) < this.deduplicationWindow) {
            // Message was shown recently, skip duplicate
            // Return empty string to indicate no toast was shown
            return '';
        }

        // Clean up old entries from recentMessages
        this.cleanupRecentMessages(now);

        // Track this message
        this.recentMessages.set(messageKey, now);

        const toast: Toast = {
            id: `toast-${this.nextId++}`,
            message: options.message,
            type: options.type || 'info',
            duration: options.duration ?? TOAST_DURATION.DEFAULT,
            title: options.title,
        };

        this.toasts.set(toast.id, toast);
        this.renderToast(toast);

        // Notify notification center of new toast
        if (this.notificationCallback) {
            this.notificationCallback(toast.type, toast.message, toast.title);
        }

        // Auto-dismiss after duration (unless duration is 0)
        if (toast.duration > 0) {
            setTimeout(() => {
                this.dismiss(toast.id);
            }, toast.duration);
        }

        return toast.id;
    }

    /**
     * Generate a key for message deduplication
     * Combines type and message to identify duplicates
     */
    private getMessageKey(options: ToastOptions): string {
        return `${options.type || 'info'}:${options.message}`;
    }

    /**
     * Clean up old entries from recentMessages map
     */
    private cleanupRecentMessages(now: number): void {
        const cutoff = now - this.deduplicationWindow;
        for (const [key, timestamp] of this.recentMessages) {
            if (timestamp < cutoff) {
                this.recentMessages.delete(key);
            }
        }
    }

    /**
     * Show an info toast
     */
    info(message: string, title?: string, duration?: number): string {
        return this.show({ message, type: 'info', title, duration });
    }

    /**
     * Show a success toast
     */
    success(message: string, title?: string, duration?: number): string {
        return this.show({ message, type: 'success', title, duration });
    }

    /**
     * Show a warning toast
     */
    warning(message: string, title?: string, duration?: number): string {
        return this.show({ message, type: 'warning', title, duration });
    }

    /**
     * Show an error toast
     */
    error(message: string, title?: string, duration?: number): string {
        return this.show({ message, type: 'error', title, duration });
    }

    /**
     * Dismiss a toast by ID
     */
    dismiss(id: string): void {
        const toast = this.toasts.get(id);
        if (!toast) return;

        const element = document.getElementById(id);
        if (element) {
            // Add fade-out animation
            element.classList.add('toast-exit');
            setTimeout(() => {
                element.remove();
                this.toasts.delete(id);
            }, 300);
        } else {
            this.toasts.delete(id);
        }
    }

    /**
     * Dismiss all toasts
     */
    dismissAll(): void {
        this.toasts.forEach((_, id) => this.dismiss(id));
    }

    /**
     * Render a toast element
     */
    private renderToast(toast: Toast): void {
        if (!this.container) return;

        const toastElement = document.createElement('div');
        toastElement.id = toast.id;
        toastElement.className = `toast toast-${toast.type}`;
        toastElement.setAttribute('role', 'alert');
        toastElement.setAttribute('aria-atomic', 'true');

        // Toast icon
        const icon = this.getIcon(toast.type);
        const iconElement = document.createElement('div');
        iconElement.className = 'toast-icon';
        iconElement.innerHTML = icon;

        // Toast content
        const contentElement = document.createElement('div');
        contentElement.className = 'toast-content';

        if (toast.title) {
            const titleElement = document.createElement('div');
            titleElement.className = 'toast-title';
            titleElement.textContent = toast.title;
            contentElement.appendChild(titleElement);
        }

        const messageElement = document.createElement('div');
        messageElement.className = 'toast-message';
        messageElement.textContent = toast.message;
        contentElement.appendChild(messageElement);

        // Close button
        const closeButton = document.createElement('button');
        closeButton.className = 'toast-close';
        closeButton.setAttribute('aria-label', 'Close notification');
        closeButton.innerHTML = '&times;';
        closeButton.onclick = () => this.dismiss(toast.id);

        toastElement.appendChild(iconElement);
        toastElement.appendChild(contentElement);
        toastElement.appendChild(closeButton);

        this.container.appendChild(toastElement);

        // Trigger animation
        requestAnimationFrame(() => {
            toastElement.classList.add('toast-enter');
        });
    }

    /**
     * Get icon SVG for toast type
     */
    private getIcon(type: ToastType): string {
        switch (type) {
            case 'success':
                return `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M20 6L9 17l-5-5"/>
                </svg>`;
            case 'error':
                return `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <line x1="15" y1="9" x2="9" y2="15"/>
                    <line x1="9" y1="9" x2="15" y2="15"/>
                </svg>`;
            case 'warning':
                return `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
                    <line x1="12" y1="9" x2="12" y2="13"/>
                    <line x1="12" y1="17" x2="12.01" y2="17"/>
                </svg>`;
            case 'info':
            default:
                return `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <line x1="12" y1="16" x2="12" y2="12"/>
                    <line x1="12" y1="8" x2="12.01" y2="8"/>
                </svg>`;
        }
    }

    /**
     * Cleanup resources to prevent memory leaks
     * Dismisses all toasts and removes the container from DOM
     */
    destroy(): void {
        // Dismiss all active toasts
        this.dismissAll();

        // Remove container from DOM
        if (this.container) {
            this.container.remove();
            this.container = null;
        }

        // Clear internal state
        this.toasts.clear();
        this.recentMessages.clear();
        this.notificationCallback = null;
    }
}
