// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * CookieConsentManager
 *
 * GDPR-compliant cookie consent banner implementation.
 * Shows a consent banner for first-time visitors and stores their preference.
 *
 * Features:
 * - Cookie consent banner with Accept/Decline options
 * - localStorage persistence of user choice
 * - Respects Do Not Track browser setting
 * - Accessible with keyboard navigation and ARIA attributes
 * - Themed to match application styles
 */

import { createLogger } from '../lib/logger';
import { SafeStorage } from '../lib/utils/SafeStorage';

const logger = createLogger('CookieConsentManager');

export interface CookiePreferences {
    necessary: boolean; // Always true - essential cookies
    analytics: boolean; // Analytics cookies (optional)
    preferences: boolean; // User preference cookies (optional)
    consentGiven: boolean;
    consentDate: string;
}

export class CookieConsentManager {
    private banner: HTMLElement | null = null;
    private readonly STORAGE_KEY = 'cartographus-cookie-consent';
    private preferences: CookiePreferences = {
        necessary: true,
        analytics: false,
        preferences: false,
        consentGiven: false,
        consentDate: '',
    };
    // AbortController for clean event listener removal
    private abortController: AbortController | null = null;
    // Track active timeouts
    private activeTimeouts: Set<ReturnType<typeof setTimeout>> = new Set();

    constructor() {
        this.loadPreferences();
        this.initialize();
    }

    /**
     * Initialize the cookie consent manager
     */
    private initialize(): void {
        // Check if consent has already been given
        if (this.preferences.consentGiven) {
            logger.debug('[CookieConsent] User has already given consent');
            return;
        }

        // Check Do Not Track setting
        if (this.isDoNotTrackEnabled()) {
            logger.debug('[CookieConsent] Do Not Track is enabled, skipping analytics cookies');
            this.preferences.analytics = false;
            this.preferences.preferences = false;
        }

        // Show the banner
        this.createBanner();
    }

    /**
     * Check if Do Not Track is enabled in the browser
     */
    private isDoNotTrackEnabled(): boolean {
        const dnt =
            navigator.doNotTrack ||
            (window as unknown as { doNotTrack?: string }).doNotTrack ||
            (navigator as unknown as { msDoNotTrack?: string }).msDoNotTrack;
        return dnt === '1' || dnt === 'yes';
    }

    /**
     * Load preferences from localStorage
     */
    private loadPreferences(): void {
        const stored = SafeStorage.getJSON<CookiePreferences | null>(this.STORAGE_KEY, null);
        if (stored) {
            this.preferences = stored;
        }
    }

    /**
     * Save preferences to localStorage
     */
    private savePreferences(): void {
        SafeStorage.setJSON(this.STORAGE_KEY, this.preferences);
    }

    /**
     * Create and display the cookie consent banner
     */
    private createBanner(): void {
        // Create banner element
        this.banner = document.createElement('div');
        this.banner.className = 'cookie-consent-banner';
        this.banner.setAttribute('role', 'dialog');
        this.banner.setAttribute('aria-label', 'Cookie consent');
        this.banner.setAttribute('aria-describedby', 'cookie-consent-description');

        this.banner.innerHTML = `
            <div class="cookie-consent-content">
                <div class="cookie-consent-text">
                    <h3 class="cookie-consent-title">Cookie Preferences</h3>
                    <p id="cookie-consent-description" class="cookie-consent-description">
                        We use cookies to enhance your experience. Essential cookies are required for the site to function.
                        Analytics cookies help us understand how you use the site. You can customize your preferences below.
                    </p>
                </div>
                <div class="cookie-consent-options">
                    <label class="cookie-option">
                        <input type="checkbox" id="cookie-necessary" checked disabled>
                        <span class="cookie-option-label">
                            <strong>Essential</strong>
                            <small>Required for basic functionality</small>
                        </span>
                    </label>
                    <label class="cookie-option">
                        <input type="checkbox" id="cookie-analytics">
                        <span class="cookie-option-label">
                            <strong>Analytics</strong>
                            <small>Help us improve the site</small>
                        </span>
                    </label>
                    <label class="cookie-option">
                        <input type="checkbox" id="cookie-preferences">
                        <span class="cookie-option-label">
                            <strong>Preferences</strong>
                            <small>Remember your settings</small>
                        </span>
                    </label>
                </div>
                <div class="cookie-consent-actions">
                    <button class="cookie-btn cookie-btn-decline" id="cookie-decline">
                        Essential Only
                    </button>
                    <button class="cookie-btn cookie-btn-accept" id="cookie-accept-selected">
                        Save Preferences
                    </button>
                    <button class="cookie-btn cookie-btn-accept-all" id="cookie-accept-all">
                        Accept All
                    </button>
                </div>
            </div>
        `;

        // Create AbortController for cleanup
        this.abortController = new AbortController();
        const signal = this.abortController.signal;

        // Add event listeners with abort signal
        const declineBtn = this.banner.querySelector('#cookie-decline') as HTMLButtonElement;
        const acceptSelectedBtn = this.banner.querySelector('#cookie-accept-selected') as HTMLButtonElement;
        const acceptAllBtn = this.banner.querySelector('#cookie-accept-all') as HTMLButtonElement;

        declineBtn?.addEventListener('click', () => this.handleDecline(), { signal });
        acceptSelectedBtn?.addEventListener('click', () => this.handleAcceptSelected(), { signal });
        acceptAllBtn?.addEventListener('click', () => this.handleAcceptAll(), { signal });

        // Handle keyboard navigation
        this.banner.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.handleDecline();
            }
        }, { signal });

        // Add to document
        document.body.appendChild(this.banner);

        // Focus the first interactive element
        const focusTimeoutId = setTimeout(() => {
            this.activeTimeouts.delete(focusTimeoutId);
            const firstCheckbox = this.banner?.querySelector('#cookie-analytics') as HTMLInputElement;
            firstCheckbox?.focus();
        }, 100);
        this.activeTimeouts.add(focusTimeoutId);

        logger.debug('[CookieConsent] Banner displayed');
    }

    /**
     * Handle declining optional cookies (essential only)
     */
    private handleDecline(): void {
        this.preferences = {
            necessary: true,
            analytics: false,
            preferences: false,
            consentGiven: true,
            consentDate: new Date().toISOString(),
        };
        this.savePreferences();
        this.hideBanner();
        logger.debug('[CookieConsent] User declined optional cookies');
    }

    /**
     * Handle accepting selected cookies
     */
    private handleAcceptSelected(): void {
        const analyticsCheckbox = this.banner?.querySelector('#cookie-analytics') as HTMLInputElement;
        const preferencesCheckbox = this.banner?.querySelector('#cookie-preferences') as HTMLInputElement;

        this.preferences = {
            necessary: true,
            analytics: analyticsCheckbox?.checked ?? false,
            preferences: preferencesCheckbox?.checked ?? false,
            consentGiven: true,
            consentDate: new Date().toISOString(),
        };
        this.savePreferences();
        this.hideBanner();
        logger.debug('[CookieConsent] User accepted selected cookies:', this.preferences);
    }

    /**
     * Handle accepting all cookies
     */
    private handleAcceptAll(): void {
        this.preferences = {
            necessary: true,
            analytics: true,
            preferences: true,
            consentGiven: true,
            consentDate: new Date().toISOString(),
        };
        this.savePreferences();
        this.hideBanner();
        logger.debug('[CookieConsent] User accepted all cookies');
    }

    /**
     * Hide and remove the banner
     */
    private hideBanner(): void {
        if (this.banner) {
            this.banner.classList.add('cookie-consent-hidden');
            const hideTimeoutId = setTimeout(() => {
                this.activeTimeouts.delete(hideTimeoutId);
                this.banner?.remove();
                this.banner = null;
            }, 300); // Match CSS transition duration
            this.activeTimeouts.add(hideTimeoutId);
        }
    }

    /**
     * Get current cookie preferences
     */
    getPreferences(): CookiePreferences {
        return { ...this.preferences };
    }

    /**
     * Check if analytics cookies are allowed
     */
    isAnalyticsAllowed(): boolean {
        return this.preferences.analytics;
    }

    /**
     * Check if preference cookies are allowed
     */
    isPreferencesAllowed(): boolean {
        return this.preferences.preferences;
    }

    /**
     * Reset consent (useful for settings page)
     */
    resetConsent(): void {
        SafeStorage.removeItem(this.STORAGE_KEY);
        this.preferences = {
            necessary: true,
            analytics: false,
            preferences: false,
            consentGiven: false,
            consentDate: '',
        };
        this.createBanner();
        logger.debug('[CookieConsent] Consent reset');
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

        // Remove banner if present
        if (this.banner) {
            this.banner.remove();
            this.banner = null;
        }

        logger.debug('[CookieConsent] CookieConsentManager destroyed');
    }
}
