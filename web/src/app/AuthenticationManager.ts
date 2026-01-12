// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * AuthenticationManager - Handles user authentication and session management
 *
 * Manages:
 * - JWT authentication (username/password)
 * - Plex PIN-based authentication (works without HTTPS/public domain)
 * - Session persistence (remember me)
 * - Login/logout lifecycle
 * - Error handling and user feedback
 * - Inline form validation (WCAG 2.1 AA compliant)
 *
 * Authentication Modes:
 * - JWT: Traditional username/password authentication
 * - Plex PIN: Single sign-on via Plex account (popup-based, no redirect required)
 */

import type { API } from '../lib/api';
import {
    PlexPINAuthenticator,
    cancelPlexAuth,
    type PlexAuthStatus,
    type PlexPINConfig,
} from '../lib/auth/plex-pin';
import { FormValidator, validators } from '../lib/form-validation';
import { createLogger } from '../lib/logger';

const logger = createLogger('Auth');

export class AuthenticationManager {
    private loginForm: HTMLFormElement | null = null;
    private loginError: HTMLElement | null = null;
    private loginButton: HTMLButtonElement | null = null;
    private plexOAuthButton: HTMLButtonElement | null = null;
    private formValidator: FormValidator | null = null;
    // AbortController for clean event listener removal
    private abortController: AbortController | null = null;
    // Production-grade Plex PIN authenticator with durability
    private plexAuthenticator: PlexPINAuthenticator;

    constructor(
        private api: API,
        private onLoginSuccess: () => void,
        private onLogoutComplete: () => void
    ) {
        this.plexAuthenticator = new PlexPINAuthenticator();
        this.initializeDOM();
    }

    /**
     * Initialize DOM element references
     */
    private initializeDOM(): void {
        this.loginForm = document.getElementById('login-form') as HTMLFormElement;
        this.loginError = document.getElementById('login-error');
        this.loginButton = document.getElementById('btn-login') as HTMLButtonElement;
        this.plexOAuthButton = document.getElementById('btn-plex-oauth') as HTMLButtonElement;
    }

    /**
     * Set up login form and OAuth button event listeners with AbortController.
     * Also checks for and resumes any interrupted Plex auth flows.
     */
    setupLoginListeners(): void {
        if (!this.loginForm) return;

        // Create AbortController for cleanup
        this.abortController = new AbortController();
        const signal = this.abortController.signal;

        // Initialize form validation
        this.setupFormValidation();

        // JWT login form submission
        this.loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            await this.handleLogin();
        }, { signal });

        // Plex PIN-based "Login with Plex" button
        // Uses popup-based auth that works without HTTPS or public domain
        if (this.plexOAuthButton) {
            this.plexOAuthButton.addEventListener('click', async () => {
                await this.handlePlexAuth();
            }, { signal });

            // Check for pending auth flow that can be resumed (e.g., after page refresh)
            this.checkAndResumePendingAuth();
        }
    }

    /**
     * Check for and resume any pending Plex authentication flow.
     * Called automatically on page load to provide seamless recovery.
     */
    private async checkAndResumePendingAuth(): Promise<void> {
        if (!this.plexAuthenticator.hasPendingAuth()) {
            return;
        }

        const pendingInfo = this.plexAuthenticator.getPendingAuthInfo();
        if (!pendingInfo) {
            return;
        }

        logger.info('Found pending Plex auth flow, resuming...', {
            correlationId: pendingInfo.correlationId,
            ageMs: Date.now() - pendingInfo.createdAt,
        });

        // Show resuming state to user
        if (this.plexOAuthButton) {
            this.plexOAuthButton.disabled = true;
            this.plexOAuthButton.textContent = 'Resuming...';
        }

        try {
            await this.plexAuthenticator.resume(this.getPlexAuthConfig());
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Plex login failed';
            logger.error('Failed to resume Plex auth:', error);
            this.showLoginError(message);
        } finally {
            if (this.plexOAuthButton) {
                this.plexOAuthButton.disabled = false;
                this.plexOAuthButton.textContent = 'Sign in with Plex';
            }
        }
    }

    /**
     * Handle Plex PIN-based authentication
     */
    private async handlePlexAuth(): Promise<void> {
        try {
            this.plexOAuthButton!.disabled = true;
            this.updatePlexButtonStatus('requesting_pin');

            await this.plexAuthenticator.authenticate(this.getPlexAuthConfig());
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Plex login failed';
            logger.error('Plex PIN auth failed:', error);
            this.showLoginError(message);
        } finally {
            this.plexOAuthButton!.disabled = false;
            this.plexOAuthButton!.textContent = 'Sign in with Plex';
        }
    }

    /**
     * Get the Plex authentication configuration
     */
    private getPlexAuthConfig(): PlexPINConfig {
        return {
            onStatusChange: (status) => this.updatePlexButtonStatus(status),
            onSuccess: (user) => {
                logger.info('Plex login successful:', user.username);
                this.onLoginSuccess();
            },
            onError: (error) => {
                logger.error('Plex login failed:', error);
                this.showLoginError(error.message);
            },
        };
    }

    /**
     * Update the Plex button text based on auth status
     */
    private updatePlexButtonStatus(status: PlexAuthStatus): void {
        if (!this.plexOAuthButton) return;

        const statusMessages: Record<PlexAuthStatus, string> = {
            idle: 'Sign in with Plex',
            requesting_pin: 'Connecting to Plex...',
            waiting_for_user: 'Waiting for approval...',
            checking_authorization: 'Checking...',
            completing_auth: 'Completing login...',
            success: 'Success!',
            error: 'Sign in with Plex',
            timeout: 'Sign in with Plex',
            cancelled: 'Sign in with Plex',
        };

        this.plexOAuthButton.textContent = statusMessages[status] || 'Sign in with Plex';
    }

    /**
     * Set up form validation for login fields
     * Uses WCAG 2.1 AA compliant inline validation
     */
    private setupFormValidation(): void {
        if (!this.loginForm) return;

        this.formValidator = new FormValidator(this.loginForm, {
            validateOnBlur: true,
            validateOnInput: false, // Avoid noise during typing
            debounceMs: 300,
        });

        // Username validation
        this.formValidator.addField('username', validators.combine(
            validators.required('Username is required'),
            validators.minLength(1, 'Username is required')
        ));

        // Password validation - basic for login (full NIST validation on registration)
        this.formValidator.addField('password', validators.combine(
            validators.required('Password is required'),
            validators.minLength(1, 'Password is required')
        ));
    }

    /**
     * Handle JWT login form submission
     */
    private async handleLogin(): Promise<void> {
        if (!this.loginButton) return;

        // Validate form before submission
        if (this.formValidator) {
            const result = this.formValidator.validate();
            if (!result.isValid) {
                // Focus first invalid field
                this.formValidator.focusFirstInvalidField();
                return;
            }
        }

        const username = (document.getElementById('username') as HTMLInputElement).value.trim();
        const password = (document.getElementById('password') as HTMLInputElement).value;
        const rememberMe = (document.getElementById('remember-me') as HTMLInputElement)?.checked ?? false;

        this.showLoginError('');
        this.loginButton.disabled = true;
        this.loginButton.textContent = 'Signing in...';
        this.loginForm?.classList.add('form-loading');

        try {
            await this.api.login({ username, password, remember_me: rememberMe });
            this.onLoginSuccess();
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Login failed';
            this.showLoginError(message);

            // Set server-side validation error on appropriate field
            if (message.toLowerCase().includes('username') || message.toLowerCase().includes('user not found')) {
                this.formValidator?.setError('username', message);
            } else if (message.toLowerCase().includes('password') || message.toLowerCase().includes('invalid credentials')) {
                this.formValidator?.setError('password', 'Invalid username or password');
            }
        } finally {
            if (this.loginButton) {
                this.loginButton.disabled = false;
                this.loginButton.textContent = 'Sign In';
            }
            this.loginForm?.classList.remove('form-loading');
        }
    }

    /**
     * Handle user logout
     * Cleans up session and triggers app cleanup via callback
     */
    handleLogout(): void {
        this.api.logout();
        this.onLogoutComplete();
    }

    /**
     * Display login error message
     */
    private showLoginError(message: string): void {
        if (!this.loginError) return;

        if (message) {
            this.loginError.textContent = message;
            this.loginError.classList.add('show');
        } else {
            this.loginError.classList.remove('show');
        }
    }

    /**
     * Check if user is currently authenticated
     */
    isAuthenticated(): boolean {
        return this.api.isAuthenticated();
    }

    /**
     * Clean up event listeners and form validation
     */
    destroy(): void {
        // Cancel any ongoing Plex auth flow
        cancelPlexAuth();

        if (this.abortController) {
            this.abortController.abort();
            this.abortController = null;
        }
        if (this.formValidator) {
            this.formValidator.destroy();
            this.formValidator = null;
        }
        this.loginForm = null;
        this.loginError = null;
        this.loginButton = null;
        this.plexOAuthButton = null;
    }
}
