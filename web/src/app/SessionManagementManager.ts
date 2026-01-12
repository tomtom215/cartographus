// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * SessionManagementManager - Active Session Management UI
 *
 * Provides a dedicated section for managing active sessions:
 * - View all active sessions (browser, OAuth, API tokens)
 * - See session details (provider, creation time, last activity)
 * - Revoke individual sessions remotely
 * - "Sign out everywhere" functionality
 *
 * ADR-0015: Zero Trust Authentication & Authorization
 *
 * Security considerations:
 * - Users can only see and revoke their own sessions
 * - Current session is clearly marked to prevent accidental logout
 * - Confirmation required for "sign out everywhere"
 */

import type { API } from '../lib/api';
import type { ToastManager } from '../lib/toast';
import type { SessionInfo } from '../lib/types/auth';
import { createLogger } from '../lib/logger';

const logger = createLogger('SessionManagement');

/**
 * Session display with parsed device info
 */
interface SessionDisplay extends SessionInfo {
    /** Friendly device name parsed from user agent */
    device_name?: string;
    /** Relative time since last activity */
    last_activity?: string;
    /** Is this session currently being revoked */
    revoking?: boolean;
}

export class SessionManagementManager {
    private api: API;
    private toastManager: ToastManager | null = null;
    private container: HTMLElement | null = null;
    private sessions: SessionDisplay[] = [];
    private abortController: AbortController | null = null;

    constructor(api: API) {
        this.api = api;
    }

    /**
     * Set toast manager for notifications
     */
    setToastManager(toastManager: ToastManager): void {
        this.toastManager = toastManager;
    }

    /**
     * Initialize the session management section
     */
    async initialize(containerId: string): Promise<void> {
        this.container = document.getElementById(containerId);
        if (!this.container) {
            logger.warn(`Container #${containerId} not found`);
            return;
        }

        this.render();
        await this.loadSessions();
        this.setupEventListeners();
        logger.debug('SessionManagementManager initialized');
    }

    /**
     * Load active sessions from API
     */
    private async loadSessions(): Promise<void> {
        this.setLoading(true);

        try {
            const response = await this.api.getSessions();
            this.sessions = response.sessions.map(session => ({
                ...session,
                device_name: this.parseDeviceName(session.user_agent),
                last_activity: session.last_accessed_at
                    ? this.formatRelativeTime(new Date(session.last_accessed_at))
                    : undefined,
            }));
            this.updateSessionsList();
        } catch (error) {
            logger.error('Failed to load sessions:', error);
            this.showError('Failed to load sessions. Please try again.');
        } finally {
            this.setLoading(false);
        }
    }

    /**
     * Render the session management panel
     */
    private render(): void {
        if (!this.container) return;

        this.container.innerHTML = `
            <div class="session-management-panel">
                <!-- Header -->
                <div class="session-management-header">
                    <div class="session-header-content">
                        <h4 class="session-title">
                            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
                                <line x1="16" y1="2" x2="16" y2="6"/>
                                <line x1="8" y1="2" x2="8" y2="6"/>
                                <line x1="3" y1="10" x2="21" y2="10"/>
                            </svg>
                            Active Sessions
                        </h4>
                        <p class="session-subtitle">
                            Manage your active login sessions across devices
                        </p>
                    </div>
                    <button class="session-refresh-btn" id="refresh-sessions-btn" title="Refresh sessions">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="23 4 23 10 17 10"/>
                            <polyline points="1 20 1 14 7 14"/>
                            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
                        </svg>
                    </button>
                </div>

                <!-- Sessions List -->
                <div class="sessions-list" id="sessions-list">
                    <div class="sessions-loading" id="sessions-loading">
                        <div class="sessions-spinner"></div>
                        <span>Loading sessions...</span>
                    </div>
                </div>

                <!-- Sign Out Everywhere -->
                <div class="session-actions">
                    <button class="session-logout-all-btn" id="logout-all-btn">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
                            <polyline points="16 17 21 12 16 7"/>
                            <line x1="21" y1="12" x2="9" y2="12"/>
                        </svg>
                        Sign Out Everywhere
                    </button>
                    <p class="session-action-note">
                        This will log you out of all devices, including this one.
                    </p>
                </div>
            </div>
        `;
    }

    /**
     * Update the sessions list with current data
     */
    private updateSessionsList(): void {
        const listContainer = document.getElementById('sessions-list');
        if (!listContainer) return;

        if (this.sessions.length === 0) {
            listContainer.innerHTML = `
                <div class="sessions-empty">
                    <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                        <circle cx="12" cy="12" r="10"/>
                        <line x1="12" y1="8" x2="12" y2="12"/>
                        <line x1="12" y1="16" x2="12.01" y2="16"/>
                    </svg>
                    <p>No active sessions found</p>
                </div>
            `;
            return;
        }

        listContainer.innerHTML = this.sessions.map(session => this.renderSessionItem(session)).join('');
    }

    /**
     * Render a single session item
     */
    private renderSessionItem(session: SessionDisplay): string {
        const providerIcon = this.getProviderIcon(session.provider);
        const providerLabel = this.getProviderLabel(session.provider);
        const createdAt = new Date(session.created_at).toLocaleString();
        const deviceName = session.device_name || 'Unknown device';

        return `
            <div class="session-item ${session.current ? 'session-current' : ''}" data-session-id="${session.id}">
                <div class="session-icon">
                    ${providerIcon}
                </div>
                <div class="session-details">
                    <div class="session-device">
                        ${deviceName}
                        ${session.current ? '<span class="session-badge">Current</span>' : ''}
                    </div>
                    <div class="session-meta">
                        <span class="session-provider">${providerLabel}</span>
                        <span class="session-separator">&#8226;</span>
                        <span class="session-time" title="Created: ${createdAt}">
                            ${session.last_activity || createdAt}
                        </span>
                    </div>
                </div>
                <button
                    class="session-revoke-btn"
                    data-session-id="${session.id}"
                    ${session.current ? 'title="This is your current session"' : 'title="Revoke this session"'}
                    ${session.revoking ? 'disabled' : ''}
                >
                    ${session.revoking ? `
                        <svg class="session-spinner-small" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <circle cx="12" cy="12" r="10"/>
                        </svg>
                    ` : `
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <line x1="18" y1="6" x2="6" y2="18"/>
                            <line x1="6" y1="6" x2="18" y2="18"/>
                        </svg>
                    `}
                </button>
            </div>
        `;
    }

    /**
     * Setup event listeners
     */
    private setupEventListeners(): void {
        if (!this.container) return;

        this.abortController = new AbortController();
        const signal = this.abortController.signal;

        // Refresh button
        const refreshBtn = this.container.querySelector('#refresh-sessions-btn');
        refreshBtn?.addEventListener('click', () => this.loadSessions(), { signal });

        // Logout all button
        const logoutAllBtn = this.container.querySelector('#logout-all-btn');
        logoutAllBtn?.addEventListener('click', () => this.confirmLogoutAll(), { signal });

        // Revoke buttons (using event delegation)
        const sessionsList = this.container.querySelector('#sessions-list');
        sessionsList?.addEventListener('click', (e) => {
            const target = e.target as HTMLElement;
            const revokeBtn = target.closest('.session-revoke-btn') as HTMLButtonElement;
            if (revokeBtn) {
                const sessionId = revokeBtn.dataset.sessionId;
                if (sessionId) {
                    this.handleRevokeSession(sessionId);
                }
            }
        }, { signal });
    }

    /**
     * Handle revoking a single session
     */
    private async handleRevokeSession(sessionId: string): Promise<void> {
        const session = this.sessions.find(s => s.id === sessionId);
        if (!session) return;

        // Warn if revoking current session
        if (session.current) {
            const confirmed = confirm(
                'You are about to revoke your current session.\n\n' +
                'This will log you out immediately. Are you sure?'
            );
            if (!confirmed) return;
        }

        // Mark as revoking
        session.revoking = true;
        this.updateSessionsList();

        try {
            await this.api.revokeSession(sessionId);

            // If current session, reload page
            if (session.current) {
                this.toastManager?.success('Session revoked. Logging out...');
                setTimeout(() => window.location.reload(), 1000);
                return;
            }

            // Remove from list
            this.sessions = this.sessions.filter(s => s.id !== sessionId);
            this.updateSessionsList();
            this.toastManager?.success('Session revoked successfully');
        } catch (error) {
            logger.error('Failed to revoke session:', error);
            session.revoking = false;
            this.updateSessionsList();
            this.toastManager?.error('Failed to revoke session');
        }
    }

    /**
     * Confirm and execute logout from all sessions
     */
    private async confirmLogoutAll(): Promise<void> {
        const confirmed = confirm(
            'Sign out everywhere?\n\n' +
            `This will log you out of all ${this.sessions.length} session(s), ` +
            'including this device.\n\n' +
            'You will need to log in again.'
        );

        if (!confirmed) return;

        try {
            const logoutAllBtn = document.getElementById('logout-all-btn') as HTMLButtonElement;
            if (logoutAllBtn) {
                logoutAllBtn.disabled = true;
                logoutAllBtn.textContent = 'Signing out...';
            }

            const response = await this.api.logoutAll();
            this.toastManager?.success(`Signed out of ${response.sessions_count || 'all'} sessions`);

            // Reload page to show login
            setTimeout(() => window.location.reload(), 1000);
        } catch (error) {
            logger.error('Failed to logout all:', error);
            this.toastManager?.error('Failed to sign out everywhere');
            const logoutAllBtn = document.getElementById('logout-all-btn') as HTMLButtonElement;
            if (logoutAllBtn) {
                logoutAllBtn.disabled = false;
                logoutAllBtn.innerHTML = `
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
                        <polyline points="16 17 21 12 16 7"/>
                        <line x1="21" y1="12" x2="9" y2="12"/>
                    </svg>
                    Sign Out Everywhere
                `;
            }
        }
    }

    /**
     * Parse device name from user agent
     */
    private parseDeviceName(userAgent?: string): string {
        if (!userAgent) return 'Unknown device';

        // Common patterns
        if (/iPhone/i.test(userAgent)) return 'iPhone';
        if (/iPad/i.test(userAgent)) return 'iPad';
        if (/Android/i.test(userAgent)) {
            if (/Mobile/i.test(userAgent)) return 'Android Phone';
            return 'Android Tablet';
        }
        if (/Windows/i.test(userAgent)) {
            if (/Edge/i.test(userAgent)) return 'Windows (Edge)';
            if (/Chrome/i.test(userAgent)) return 'Windows (Chrome)';
            if (/Firefox/i.test(userAgent)) return 'Windows (Firefox)';
            return 'Windows PC';
        }
        if (/Mac OS X/i.test(userAgent)) {
            if (/Safari/i.test(userAgent) && !/Chrome/i.test(userAgent)) return 'Mac (Safari)';
            if (/Chrome/i.test(userAgent)) return 'Mac (Chrome)';
            if (/Firefox/i.test(userAgent)) return 'Mac (Firefox)';
            return 'Mac';
        }
        if (/Linux/i.test(userAgent)) {
            if (/Chrome/i.test(userAgent)) return 'Linux (Chrome)';
            if (/Firefox/i.test(userAgent)) return 'Linux (Firefox)';
            return 'Linux';
        }
        if (/CrOS/i.test(userAgent)) return 'Chromebook';

        // API clients
        if (/curl/i.test(userAgent)) return 'API Client (curl)';
        if (/python/i.test(userAgent)) return 'API Client (Python)';

        return 'Unknown device';
    }

    /**
     * Get icon for auth provider
     */
    private getProviderIcon(provider: string): string {
        switch (provider) {
            case 'plex':
                return `<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M12 2L2 22h20L12 2zm0 4.5l6.5 13h-13L12 6.5z"/>
                </svg>`;
            case 'oidc':
                return `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="3"/>
                    <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
                </svg>`;
            case 'jwt':
            case 'basic':
                return `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
                    <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                </svg>`;
            default:
                return `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <circle cx="12" cy="12" r="3"/>
                </svg>`;
        }
    }

    /**
     * Get label for auth provider
     */
    private getProviderLabel(provider: string): string {
        switch (provider) {
            case 'plex': return 'Plex OAuth';
            case 'oidc': return 'SSO (OIDC)';
            case 'jwt': return 'Username/Password';
            case 'basic': return 'Basic Auth';
            default: return provider;
        }
    }

    /**
     * Format relative time
     */
    private formatRelativeTime(date: Date): string {
        const now = new Date();
        const diffMs = now.getTime() - date.getTime();
        const diffMins = Math.floor(diffMs / 60000);
        const diffHours = Math.floor(diffMs / 3600000);
        const diffDays = Math.floor(diffMs / 86400000);

        if (diffMins < 1) return 'Just now';
        if (diffMins < 60) return `${diffMins}m ago`;
        if (diffHours < 24) return `${diffHours}h ago`;
        if (diffDays < 7) return `${diffDays}d ago`;
        return date.toLocaleDateString();
    }

    /**
     * Set loading state
     */
    private setLoading(loading: boolean): void {
        const loadingEl = document.getElementById('sessions-loading');
        const listEl = document.getElementById('sessions-list');

        if (loading) {
            if (loadingEl) loadingEl.style.display = 'flex';
            if (listEl) listEl.innerHTML = '';
        } else {
            if (loadingEl) loadingEl.style.display = 'none';
        }
    }

    /**
     * Show error message
     */
    private showError(message: string): void {
        const listContainer = document.getElementById('sessions-list');
        if (!listContainer) return;

        listContainer.innerHTML = `
            <div class="sessions-error">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <line x1="12" y1="8" x2="12" y2="12"/>
                    <line x1="12" y1="16" x2="12.01" y2="16"/>
                </svg>
                <p>${message}</p>
                <button class="session-retry-btn" id="retry-sessions-btn">Try Again</button>
            </div>
        `;

        const retryBtn = document.getElementById('retry-sessions-btn');
        retryBtn?.addEventListener('click', () => this.loadSessions());
    }

    /**
     * Refresh session list
     */
    async refresh(): Promise<void> {
        await this.loadSessions();
    }

    /**
     * Clean up event listeners and resources
     */
    destroy(): void {
        if (this.abortController) {
            this.abortController.abort();
            this.abortController = null;
        }
        this.container = null;
        this.sessions = [];
        this.toastManager = null;
    }
}

// Export singleton factory
let sessionManagementManagerInstance: SessionManagementManager | null = null;

export function getSessionManagementManager(api: API): SessionManagementManager {
    if (!sessionManagementManagerInstance) {
        sessionManagementManagerInstance = new SessionManagementManager(api);
    }
    return sessionManagementManagerInstance;
}
