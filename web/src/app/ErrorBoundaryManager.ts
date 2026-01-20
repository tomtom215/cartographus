// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * ErrorBoundaryManager - Global error handling with recovery UI
 *
 * Features:
 * - Displays error overlay when critical failures occur
 * - Retry functionality for data loading failures
 * - Network error detection
 * - Graceful degradation for partial failures
 * - Screen reader announcements (ARIA)
 * - Focus management for accessibility
 *
 * @see /docs/working/UI_UX_AUDIT.md
 */

import { createLogger } from '../lib/logger';

const logger = createLogger('ErrorBoundaryManager');

/**
 * Window interface extension for E2E testing
 */
declare global {
    interface Window {
        __E2E_ERROR_BOUNDARY_THRESHOLD__?: number;
    }
}

export interface ErrorBoundaryConfig {
    /** Number of retries before showing persistent error */
    maxRetries: number;
    /** Delay between automatic retries in ms */
    retryDelay: number;
    /** Show full overlay only after this many consecutive failures */
    overlayThreshold: number;
}

export interface ErrorContext {
    /** Type of error (network, api, runtime, unknown) */
    type: 'network' | 'api' | 'runtime' | 'unknown';
    /** Error message to display */
    message: string;
    /** Original error object */
    error?: Error;
    /** Whether this error can be recovered with retry */
    recoverable: boolean;
    /** Component/area that failed */
    component?: string;
    /** If true, bypass threshold and show overlay immediately */
    critical?: boolean;
}

const DEFAULT_CONFIG: ErrorBoundaryConfig = {
    maxRetries: 3,
    retryDelay: 2000,
    overlayThreshold: 3,
};

export class ErrorBoundaryManager {
    private config: ErrorBoundaryConfig;
    private consecutiveFailures: number = 0;
    private isRetrying: boolean = false;
    private retryCallback: (() => Promise<void>) | null = null;
    private previouslyFocusedElement: HTMLElement | null = null;
    private partialFailures: Set<string> = new Set();

    // DOM elements
    private overlay: HTMLElement | null = null;
    private title: HTMLElement | null = null;
    private message: HTMLElement | null = null;
    private retryButton: HTMLButtonElement | null = null;
    private dismissButton: HTMLButtonElement | null = null;

    // Event handler references for cleanup
    private retryButtonClickHandler: (() => void) | null = null;
    private dismissButtonClickHandler: (() => void) | null = null;
    private overlayClickHandler: ((e: MouseEvent) => void) | null = null;
    private documentKeydownHandler: ((e: KeyboardEvent) => void) | null = null;
    private windowOnlineHandler: (() => void) | null = null;
    private windowOfflineHandler: (() => void) | null = null;
    private unhandledRejectionHandler: ((e: PromiseRejectionEvent) => void) | null = null;
    private windowErrorHandler: ((e: ErrorEvent) => void) | null = null;

    constructor(config: Partial<ErrorBoundaryConfig> = {}) {
        this.config = { ...DEFAULT_CONFIG, ...config };

        // E2E Test Support: Allow tests to override the overlay threshold
        // This enables tests to trigger the error overlay immediately (threshold=1)
        // without waiting for 3 consecutive failures (default threshold)
        const e2eThreshold = window.__E2E_ERROR_BOUNDARY_THRESHOLD__;
        if (typeof e2eThreshold === 'number' && e2eThreshold >= 1) {
            this.config.overlayThreshold = e2eThreshold;
            logger.debug(`E2E: overlayThreshold set to ${e2eThreshold}`);
        }
    }

    /**
     * Initialize the error boundary manager
     */
    init(): void {
        this.bindElements();
        this.setupEventListeners();
        this.setupGlobalErrorHandlers();
        logger.debug('ErrorBoundaryManager initialized');
    }

    /**
     * Set the callback function for retry actions
     */
    setRetryCallback(callback: () => Promise<void>): void {
        this.retryCallback = callback;
    }

    /**
     * Bind to DOM elements
     */
    private bindElements(): void {
        this.overlay = document.getElementById('error-boundary-overlay');
        this.title = document.getElementById('error-boundary-title');
        this.message = document.getElementById('error-boundary-message');
        this.retryButton = document.getElementById('error-boundary-retry') as HTMLButtonElement;
        this.dismissButton = document.getElementById('error-boundary-dismiss') as HTMLButtonElement;
    }

    /**
     * Set up event listeners
     */
    private setupEventListeners(): void {
        // Retry button
        if (this.retryButton) {
            this.retryButtonClickHandler = () => this.handleRetry();
            this.retryButton.addEventListener('click', this.retryButtonClickHandler);
        }

        // Dismiss button
        if (this.dismissButton) {
            this.dismissButtonClickHandler = () => this.dismiss();
            this.dismissButton.addEventListener('click', this.dismissButtonClickHandler);
        }

        // Overlay click to dismiss (if dismissible)
        if (this.overlay) {
            this.overlayClickHandler = (e: MouseEvent) => {
                if (e.target === this.overlay) {
                    this.dismiss();
                }
            };
            this.overlay.addEventListener('click', this.overlayClickHandler);
        }

        // Keyboard events for accessibility
        this.documentKeydownHandler = (e: KeyboardEvent) => this.handleKeyDown(e);
        document.addEventListener('keydown', this.documentKeydownHandler);

        // Network status monitoring
        this.windowOnlineHandler = () => this.handleOnline();
        this.windowOfflineHandler = () => this.handleOffline();
        window.addEventListener('online', this.windowOnlineHandler);
        window.addEventListener('offline', this.windowOfflineHandler);
    }

    /**
     * Set up global error handlers
     */
    private setupGlobalErrorHandlers(): void {
        // Catch unhandled promise rejections
        this.unhandledRejectionHandler = (event: PromiseRejectionEvent) => {
            logger.error('Unhandled rejection:', event.reason);
            // Don't show overlay for every unhandled promise - just log
            // This prevents overwhelming the user with error dialogs
        };
        window.addEventListener('unhandledrejection', this.unhandledRejectionHandler);

        // Catch uncaught exceptions
        this.windowErrorHandler = (event: ErrorEvent) => {
            logger.error('Uncaught error:', event.error);
            // Don't show overlay for every error - just log
            // Runtime errors shouldn't crash the app visually
        };
        window.addEventListener('error', this.windowErrorHandler);
    }

    /**
     * Handle keyboard events when overlay is visible
     */
    private handleKeyDown(e: KeyboardEvent): void {
        if (!this.isOverlayVisible()) return;

        // Escape to dismiss (if dismissible)
        if (e.key === 'Escape') {
            e.preventDefault();
            this.dismiss();
            return;
        }

        // Tab key for focus trapping
        if (e.key === 'Tab') {
            this.handleTabKey(e);
        }
    }

    /**
     * Handle Tab key for focus trapping within overlay
     */
    private handleTabKey(e: KeyboardEvent): void {
        if (!this.overlay) return;

        const focusableElements = this.overlay.querySelectorAll(
            'button:not(:disabled), [href], input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])'
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
     * Handle network coming back online
     */
    private handleOnline(): void {
        logger.debug('Network restored');
        // If showing network error, auto-retry
        if (this.isOverlayVisible()) {
            this.handleRetry();
        }
        // Clear network-related partial failures
        this.partialFailures.delete('network');
    }

    /**
     * Handle network going offline
     */
    private handleOffline(): void {
        logger.debug('Network lost');
        this.show({
            type: 'network',
            message: 'Network connection lost. Please check your internet connection and try again.',
            recoverable: true,
            component: 'network'
        });
    }

    /**
     * Report an error to the boundary
     * This is the main entry point for error handling
     */
    reportError(context: ErrorContext): void {
        logger.error(`Error reported [${context.type}]:`, context.message, context.error);

        // Track consecutive failures
        this.consecutiveFailures++;

        // For partial failures, track component
        if (context.component) {
            this.partialFailures.add(context.component);
        }

        // Show overlay immediately for critical errors (bypass threshold)
        // or if we've exceeded the threshold for non-critical errors
        if (context.critical || this.consecutiveFailures >= this.config.overlayThreshold) {
            this.show(context);
        }
    }

    /**
     * Report a successful operation (resets failure count)
     */
    reportSuccess(): void {
        this.consecutiveFailures = 0;
        this.partialFailures.clear();
        this.hide();
    }

    /**
     * Report a partial failure (some data loaded, some failed)
     * Doesn't show full overlay, but tracks the failure
     */
    reportPartialFailure(component: string, message: string): void {
        logger.warn(`Partial failure [${component}]:`, message);
        this.partialFailures.add(component);
        // Don't show full overlay for partial failures
        // Toast notifications should be used instead
    }

    /**
     * Check if any partial failures exist
     */
    hasPartialFailures(): boolean {
        return this.partialFailures.size > 0;
    }

    /**
     * Get list of failed components
     */
    getFailedComponents(): string[] {
        return Array.from(this.partialFailures);
    }

    /**
     * Show the error overlay
     */
    show(context: ErrorContext): void {
        if (!this.overlay || !this.title || !this.message) {
            logger.error('Error boundary elements not found');
            return;
        }

        // Store currently focused element
        this.previouslyFocusedElement = document.activeElement as HTMLElement;

        // Update content based on error type
        const titles: Record<ErrorContext['type'], string> = {
            network: 'Connection Error',
            api: 'Unable to Load Data',
            runtime: 'Application Error',
            unknown: 'Something Went Wrong'
        };

        this.title.textContent = titles[context.type] || titles.unknown;
        this.message.textContent = context.message;

        // Set data attribute for styling/testing
        this.overlay.setAttribute('data-error-type', context.type);

        // Show/hide retry button based on recoverability
        if (this.retryButton) {
            this.retryButton.style.display = context.recoverable ? 'inline-flex' : 'none';
            this.retryButton.disabled = false;
            this.retryButton.textContent = 'Retry';
        }

        // Show overlay
        this.overlay.style.display = 'flex';
        this.overlay.classList.add('visible');
        this.overlay.setAttribute('aria-hidden', 'false');

        // Focus retry or dismiss button for accessibility
        setTimeout(() => {
            if (context.recoverable && this.retryButton) {
                this.retryButton.focus();
            } else if (this.dismissButton) {
                this.dismissButton.focus();
            }
        }, 100);

        logger.debug('Error overlay shown');
    }

    /**
     * Hide the error overlay
     */
    hide(): void {
        if (!this.overlay) return;

        this.overlay.classList.remove('visible');
        this.overlay.setAttribute('aria-hidden', 'true');

        // Wait for transition before hiding
        setTimeout(() => {
            if (this.overlay) {
                this.overlay.style.display = 'none';
            }
        }, 200);

        // Restore focus
        if (this.previouslyFocusedElement) {
            this.previouslyFocusedElement.focus();
            this.previouslyFocusedElement = null;
        }
    }

    /**
     * Dismiss the error overlay (user action)
     */
    dismiss(): void {
        this.hide();
        // Reset failure count so user can continue
        // (They explicitly dismissed, indicating they want to proceed)
        this.consecutiveFailures = 0;
    }

    /**
     * Handle retry button click
     */
    private async handleRetry(): Promise<void> {
        if (this.isRetrying || !this.retryCallback) return;

        this.isRetrying = true;

        // Update button state
        if (this.retryButton) {
            this.retryButton.disabled = true;
            this.retryButton.textContent = 'Retrying...';
        }

        try {
            await this.retryCallback();
            // Success - hide overlay and reset
            this.consecutiveFailures = 0;
            this.hide();
        } catch (error) {
            logger.error('Retry failed:', error);
            // Update message to indicate retry failure
            if (this.message) {
                this.message.textContent = 'Retry failed. Please check your connection and try again.';
            }
        } finally {
            this.isRetrying = false;
            if (this.retryButton) {
                this.retryButton.disabled = false;
                this.retryButton.textContent = 'Retry';
            }
        }
    }

    /**
     * Check if overlay is currently visible
     */
    isOverlayVisible(): boolean {
        return this.overlay?.classList.contains('visible') ?? false;
    }

    /**
     * Create error context from an Error object
     */
    static createContext(error: Error | unknown, component?: string): ErrorContext {
        // Data-load errors use standard threshold (not critical)
        // This allows graceful degradation on first failure, showing error overlay
        // only after multiple consecutive failures (default: 3)
        const isCritical = false;

        // Network errors
        if (error instanceof TypeError && error.message.includes('fetch')) {
            return {
                type: 'network',
                message: 'Unable to connect to the server. Please check your network connection.',
                error: error as Error,
                recoverable: true,
                component,
                critical: isCritical
            };
        }

        // API errors (HTTP status codes)
        if (error instanceof Error && error.message.includes('HTTP')) {
            return {
                type: 'api',
                message: 'The server returned an error. Please try again later.',
                error,
                recoverable: true,
                component,
                critical: isCritical
            };
        }

        // Generic runtime errors
        if (error instanceof Error) {
            return {
                type: 'runtime',
                message: 'An unexpected error occurred. Please refresh the page.',
                error,
                recoverable: false,
                component,
                critical: isCritical
            };
        }

        // Unknown errors
        return {
            type: 'unknown',
            message: 'Something went wrong. Please try again.',
            recoverable: true,
            component,
            critical: isCritical
        };
    }

    /**
     * Remove all event listeners for cleanup
     */
    private removeEventListeners(): void {
        // Remove retry button handler
        if (this.retryButton && this.retryButtonClickHandler) {
            this.retryButton.removeEventListener('click', this.retryButtonClickHandler);
            this.retryButtonClickHandler = null;
        }

        // Remove dismiss button handler
        if (this.dismissButton && this.dismissButtonClickHandler) {
            this.dismissButton.removeEventListener('click', this.dismissButtonClickHandler);
            this.dismissButtonClickHandler = null;
        }

        // Remove overlay click handler
        if (this.overlay && this.overlayClickHandler) {
            this.overlay.removeEventListener('click', this.overlayClickHandler);
            this.overlayClickHandler = null;
        }

        // Remove document keydown handler
        if (this.documentKeydownHandler) {
            document.removeEventListener('keydown', this.documentKeydownHandler);
            this.documentKeydownHandler = null;
        }

        // Remove window online/offline handlers
        if (this.windowOnlineHandler) {
            window.removeEventListener('online', this.windowOnlineHandler);
            this.windowOnlineHandler = null;
        }

        if (this.windowOfflineHandler) {
            window.removeEventListener('offline', this.windowOfflineHandler);
            this.windowOfflineHandler = null;
        }

        // Remove global error handlers
        if (this.unhandledRejectionHandler) {
            window.removeEventListener('unhandledrejection', this.unhandledRejectionHandler);
            this.unhandledRejectionHandler = null;
        }

        if (this.windowErrorHandler) {
            window.removeEventListener('error', this.windowErrorHandler);
            this.windowErrorHandler = null;
        }
    }

    /**
     * Destroy the error boundary manager and clean up resources
     */
    destroy(): void {
        this.removeEventListeners();
        this.hide();
    }
}

export default ErrorBoundaryManager;
