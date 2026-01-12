// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * ServerManagementManager - Server management dashboard
 *
 * Features:
 * - Display detailed server information
 * - Show Tautulli connection info
 * - Multi-server list display
 * - PMS update check
 * - NATS health monitoring
 */

import type { API } from '../lib/api';
import type { ToastManager } from '../lib/toast';
import { createLogger } from '../lib/logger';
import { getRoleGuard } from '../lib/auth/RoleGuard';

const logger = createLogger('ServerManagementManager');

// Tautulli Info interface
export interface TautulliInfo {
    tautulli_version: string;
    tautulli_platform: string;
    tautulli_branch: string;
    tautulli_install_type: string;
    tautulli_remote_access: number;
    tautulli_update_available: boolean;
}

// Server list item interface
export interface ServerListItem {
    name: string;
    machine_identifier: string;
    host: string;
    port: number;
    ssl: number;
    is_cloud: number;
    platform: string;
    version: string;
}

// Server identity interface
export interface ServerIdentity {
    machine_identifier: string;
    version: string;
}

// PMS Update interface
export interface PMSUpdate {
    update_available: boolean;
    release_date?: string;
    version?: string;
    download_url?: string;
}

// NATS Health interface
export interface NATSHealth {
    status: string;
    connected: boolean;
    jetstream_enabled?: boolean;
    streams?: number;
    consumers?: number;
    server_id?: string;
    version?: string;
    error?: string;
}

export class ServerManagementManager {
    private api: API;
    private toastManager: ToastManager | null = null;
    private initialized = false;

    // Data
    private tautulliInfo: TautulliInfo | null = null;
    private serverList: ServerListItem[] = [];
    private pmsUpdate: PMSUpdate | null = null;
    private natsHealth: NATSHealth | null = null;

    // DOM elements
    private container: HTMLElement | null = null;

    constructor(api: API) {
        this.api = api;
    }

    /**
     * Initialize the server management manager
     * RBAC: Viewers can see server info, only admins can modify sync settings
     */
    init(containerId: string = 'server-management-section'): void {
        this.container = document.getElementById(containerId);
        if (!this.container) {
            logger.warn('Server management section not found, creating', { containerId });
            this.createContainer();
        }

        // RBAC Phase 4: Check read permission for server info
        const roleGuard = getRoleGuard();
        if (!roleGuard.canAccess('server', 'read')) {
            logger.warn('[RBAC] User lacks permission to view server management');
            if (this.container) {
                this.container.innerHTML = `
                    <div class="access-denied-message">
                        <p>You do not have permission to view server management.</p>
                    </div>
                `;
            }
            return;
        }

        this.setupEventListeners();
        this.applyRoleBasedVisibility();
        this.initialized = true;
        logger.debug('ServerManagementManager initialized');

        // Load initial data
        this.loadData();
    }

    /**
     * Apply role-based visibility to server management UI elements.
     * RBAC Phase 4: Frontend Role Integration
     *
     * - Viewers can see server info but cannot trigger sync operations
     * - Admins can control sync and server settings
     */
    private applyRoleBasedVisibility(): void {
        const roleGuard = getRoleGuard();

        // Disable admin-only operations for non-admin users
        if (!roleGuard.canAccess('server', 'admin')) {
            // Disable sync button
            const syncButton = document.getElementById('btn-trigger-sync') as HTMLButtonElement;
            if (syncButton) {
                syncButton.disabled = true;
                syncButton.title = 'Admin role required to trigger sync';
            }

            // Add read-only class to container
            this.container?.classList.add('rbac-read-only');
        }
    }

    /**
     * Create the server management container if it doesn't exist
     */
    private createContainer(): void {
        const serverHealth = document.getElementById('server-health');
        if (!serverHealth) return;

        const section = document.createElement('div');
        section.id = 'server-management-section';
        section.className = 'server-management-section';
        section.setAttribute('aria-label', 'Server management');
        section.innerHTML = this.renderInitialHTML();

        // Insert after server-health
        serverHealth.parentNode?.insertBefore(section, serverHealth.nextSibling);
        this.container = section;
    }

    /**
     * Render initial HTML structure
     */
    private renderInitialHTML(): string {
        return `
            <div class="server-management-header">
                <h3>Server Management</h3>
                <button id="btn-refresh-server-info" type="button" class="btn-refresh-server" aria-label="Refresh server information">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="23 4 23 10 17 10"/>
                        <polyline points="1 20 1 14 7 14"/>
                        <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
                    </svg>
                    Refresh
                </button>
            </div>

            <!-- Tautulli Info Card -->
            <div id="tautulli-info-card" class="server-card tautulli-card">
                <div class="server-card-header">
                    <h4>Tautulli</h4>
                    <span id="tautulli-status" class="server-status status-loading">●</span>
                </div>
                <div class="server-details">
                    <div class="server-detail-row">
                        <span class="server-detail-label">Version:</span>
                        <span class="server-detail-value" id="tautulli-version">Loading...</span>
                    </div>
                    <div class="server-detail-row">
                        <span class="server-detail-label">Platform:</span>
                        <span class="server-detail-value" id="tautulli-platform">-</span>
                    </div>
                    <div class="server-detail-row">
                        <span class="server-detail-label">Branch:</span>
                        <span class="server-detail-value" id="tautulli-branch">-</span>
                    </div>
                    <div class="server-detail-row">
                        <span class="server-detail-label">Install Type:</span>
                        <span class="server-detail-value" id="tautulli-install-type">-</span>
                    </div>
                    <div class="server-detail-row">
                        <span class="server-detail-label">Remote Access:</span>
                        <span class="server-detail-value" id="tautulli-remote-access">-</span>
                    </div>
                </div>
            </div>

            <!-- PMS Update Check -->
            <div id="pms-update-section" class="server-update-section">
                <div class="server-card-header">
                    <h4>PMS Update Status</h4>
                    <button id="btn-check-update" type="button" class="btn-check-update" aria-label="Check for updates">
                        Check Updates
                    </button>
                </div>
                <div id="pms-update-notice" class="pms-update-notice" style="display: none;">
                    <div class="update-icon">
                        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                            <polyline points="7 10 12 15 17 10"/>
                            <line x1="12" y1="15" x2="12" y2="3"/>
                        </svg>
                    </div>
                    <div class="update-info">
                        <strong>Update Available!</strong>
                        <span id="pms-update-version">Version: -</span>
                        <span id="pms-update-date">Release Date: -</span>
                    </div>
                </div>
                <div id="pms-no-update" class="pms-no-update">
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
                        <polyline points="22 4 12 14.01 9 11.01"/>
                    </svg>
                    <span>Plex Media Server is up to date</span>
                </div>
            </div>

            <!-- Server List Section -->
            <div id="server-list-section" class="server-list-section">
                <h4>Connected Servers</h4>
                <div id="server-list-container" class="server-list-container" role="list" aria-label="Server list">
                    <div class="server-list-loading">Loading servers...</div>
                </div>
            </div>

            <!-- NATS Health Section -->
            <div id="nats-health-section" class="nats-health-section">
                <div class="server-card-header">
                    <h4>NATS Messaging</h4>
                    <span id="nats-connection-status" class="server-status status-loading">●</span>
                </div>
                <div id="nats-health-content" class="nats-health-content">
                    <div class="server-details">
                        <div class="server-detail-row">
                            <span class="server-detail-label">Status:</span>
                            <span class="server-detail-value" id="nats-status-text">Loading...</span>
                        </div>
                        <div class="server-detail-row">
                            <span class="server-detail-label">JetStream:</span>
                            <span class="server-detail-value" id="nats-jetstream-status">-</span>
                        </div>
                        <div class="server-detail-row">
                            <span class="server-detail-label">Streams:</span>
                            <span class="server-detail-value" id="nats-streams-count">-</span>
                        </div>
                        <div class="server-detail-row">
                            <span class="server-detail-label">Consumers:</span>
                            <span class="server-detail-value" id="nats-consumers-count">-</span>
                        </div>
                        <div class="server-detail-row">
                            <span class="server-detail-label">Version:</span>
                            <span class="server-detail-value" id="nats-version">-</span>
                        </div>
                    </div>
                </div>
                <div id="nats-unavailable" class="nats-unavailable" style="display: none;">
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="12" cy="12" r="10"/>
                        <line x1="15" y1="9" x2="9" y2="15"/>
                        <line x1="9" y1="9" x2="15" y2="15"/>
                    </svg>
                    <span>NATS messaging not configured</span>
                </div>
            </div>
        `;
    }

    /**
     * Set toast manager reference
     */
    setToastManager(toast: ToastManager): void {
        this.toastManager = toast;
    }

    /**
     * Set up event listeners
     */
    private setupEventListeners(): void {
        // Refresh button
        document.getElementById('btn-refresh-server-info')?.addEventListener('click', () => {
            this.loadData();
        });

        // Check update button
        document.getElementById('btn-check-update')?.addEventListener('click', () => {
            this.checkPMSUpdate();
        });
    }

    /**
     * Load all server data
     */
    async loadData(): Promise<void> {
        if (!this.initialized) return;

        try {
            await Promise.all([
                this.loadTautulliInfo(),
                this.loadServerList(),
                this.loadPMSUpdate(),
                this.loadNATSHealth()
            ]);
        } catch (error) {
            logger.error('Failed to load server data', { error });
        }
    }

    /**
     * Load Tautulli info
     */
    private async loadTautulliInfo(): Promise<void> {
        try {
            this.tautulliInfo = await this.api.getTautulliInfo();
            this.renderTautulliInfo();
        } catch (error) {
            logger.error('Failed to load Tautulli info', { error });
            this.renderTautulliError();
        }
    }

    /**
     * Render Tautulli info
     */
    private renderTautulliInfo(): void {
        if (!this.tautulliInfo) return;

        const statusEl = document.getElementById('tautulli-status');
        const versionEl = document.getElementById('tautulli-version');
        const platformEl = document.getElementById('tautulli-platform');
        const branchEl = document.getElementById('tautulli-branch');
        const installTypeEl = document.getElementById('tautulli-install-type');
        const remoteAccessEl = document.getElementById('tautulli-remote-access');

        if (statusEl) {
            statusEl.className = 'server-status status-online';
        }
        if (versionEl) {
            versionEl.textContent = this.tautulliInfo.tautulli_version || '-';
        }
        if (platformEl) {
            platformEl.textContent = this.tautulliInfo.tautulli_platform || '-';
        }
        if (branchEl) {
            branchEl.textContent = this.tautulliInfo.tautulli_branch || '-';
        }
        if (installTypeEl) {
            installTypeEl.textContent = this.tautulliInfo.tautulli_install_type || '-';
        }
        if (remoteAccessEl) {
            remoteAccessEl.textContent = this.tautulliInfo.tautulli_remote_access ? 'Enabled' : 'Disabled';
        }
    }

    /**
     * Render Tautulli error state
     */
    private renderTautulliError(): void {
        const statusEl = document.getElementById('tautulli-status');
        const versionEl = document.getElementById('tautulli-version');

        if (statusEl) {
            statusEl.className = 'server-status status-error';
        }
        if (versionEl) {
            versionEl.textContent = 'Error loading';
        }
    }

    /**
     * Load server list
     */
    private async loadServerList(): Promise<void> {
        try {
            this.serverList = await this.api.getTautulliServerList();
            this.renderServerList();
        } catch (error) {
            logger.error('Failed to load server list', { error });
            this.renderServerListError();
        }
    }

    /**
     * Render server list
     */
    private renderServerList(): void {
        const container = document.getElementById('server-list-container');
        if (!container) return;

        if (this.serverList.length === 0) {
            container.innerHTML = `
                <div class="server-list-empty">
                    <p>No servers found</p>
                </div>
            `;
            return;
        }

        container.innerHTML = this.serverList.map(server => `
            <div class="server-list-item" role="listitem">
                <div class="server-list-item-header">
                    <span class="server-list-name">${this.escapeHtml(server.name)}</span>
                    <span class="server-list-status ${server.is_cloud ? 'status-cloud' : 'status-local'}">
                        ${server.is_cloud ? 'Cloud' : 'Local'}
                    </span>
                </div>
                <div class="server-list-details">
                    <div class="server-list-detail">
                        <span class="server-list-label">Host:</span>
                        <span class="server-list-value">${this.escapeHtml(server.host)}:${server.port}</span>
                    </div>
                    <div class="server-list-detail">
                        <span class="server-list-label">Platform:</span>
                        <span class="server-list-value">${this.escapeHtml(server.platform)}</span>
                    </div>
                    <div class="server-list-detail">
                        <span class="server-list-label">Version:</span>
                        <span class="server-list-value">${this.escapeHtml(server.version)}</span>
                    </div>
                    <div class="server-list-detail">
                        <span class="server-list-label">SSL:</span>
                        <span class="server-list-value">${server.ssl ? 'Enabled' : 'Disabled'}</span>
                    </div>
                </div>
            </div>
        `).join('');
    }

    /**
     * Render server list error
     */
    private renderServerListError(): void {
        const container = document.getElementById('server-list-container');
        if (!container) return;

        container.innerHTML = `
            <div class="server-list-error server-error-state">
                <p>Failed to load server list</p>
            </div>
        `;
    }

    /**
     * Load PMS update status
     */
    private async loadPMSUpdate(): Promise<void> {
        try {
            this.pmsUpdate = await this.api.getTautulliPMSUpdate();
            this.renderPMSUpdate();
        } catch (error) {
            logger.error('Failed to load PMS update', { error });
        }
    }

    /**
     * Check for PMS updates
     */
    private async checkPMSUpdate(): Promise<void> {
        this.toastManager?.info('Checking for updates...');
        try {
            this.pmsUpdate = await this.api.getTautulliPMSUpdate();
            this.renderPMSUpdate();

            if (this.pmsUpdate?.update_available) {
                this.toastManager?.success(`Update available: ${this.pmsUpdate.version}`);
            } else {
                this.toastManager?.success('Plex Media Server is up to date');
            }
        } catch (error) {
            logger.error('Failed to check PMS update', { error });
            this.toastManager?.error('Failed to check for updates');
        }
    }

    /**
     * Render PMS update status
     */
    private renderPMSUpdate(): void {
        const noticeEl = document.getElementById('pms-update-notice');
        const noUpdateEl = document.getElementById('pms-no-update');
        const versionEl = document.getElementById('pms-update-version');
        const dateEl = document.getElementById('pms-update-date');

        if (!noticeEl || !noUpdateEl) return;

        if (this.pmsUpdate?.update_available) {
            noticeEl.style.display = 'flex';
            noUpdateEl.style.display = 'none';

            if (versionEl && this.pmsUpdate.version) {
                versionEl.textContent = `Version: ${this.pmsUpdate.version}`;
            }
            if (dateEl && this.pmsUpdate.release_date) {
                dateEl.textContent = `Release Date: ${this.pmsUpdate.release_date}`;
            }
        } else {
            noticeEl.style.display = 'none';
            noUpdateEl.style.display = 'flex';
        }
    }

    /**
     * Load NATS health status
     */
    private async loadNATSHealth(): Promise<void> {
        try {
            this.natsHealth = await this.api.getNATSHealth();
            this.renderNATSHealth();
        } catch (error) {
            logger.error('Failed to load NATS health', { error });
            this.renderNATSUnavailable();
        }
    }

    /**
     * Render NATS health
     */
    private renderNATSHealth(): void {
        const contentEl = document.getElementById('nats-health-content');
        const unavailableEl = document.getElementById('nats-unavailable');
        const statusIndicator = document.getElementById('nats-connection-status');
        const statusText = document.getElementById('nats-status-text');
        const jetstreamEl = document.getElementById('nats-jetstream-status');
        const streamsEl = document.getElementById('nats-streams-count');
        const consumersEl = document.getElementById('nats-consumers-count');
        const versionEl = document.getElementById('nats-version');

        if (!this.natsHealth || !this.natsHealth.connected) {
            this.renderNATSUnavailable();
            return;
        }

        if (contentEl) contentEl.style.display = 'block';
        if (unavailableEl) unavailableEl.style.display = 'none';

        if (statusIndicator) {
            statusIndicator.className = `server-status ${this.natsHealth.status === 'healthy' ? 'status-online' : 'status-warning'}`;
        }
        if (statusText) {
            statusText.textContent = this.natsHealth.status || 'Unknown';
        }
        if (jetstreamEl) {
            jetstreamEl.textContent = this.natsHealth.jetstream_enabled ? 'Enabled' : 'Disabled';
        }
        if (streamsEl) {
            streamsEl.textContent = this.natsHealth.streams?.toString() || '0';
        }
        if (consumersEl) {
            consumersEl.textContent = this.natsHealth.consumers?.toString() || '0';
        }
        if (versionEl) {
            versionEl.textContent = this.natsHealth.version || '-';
        }
    }

    /**
     * Render NATS unavailable state
     */
    private renderNATSUnavailable(): void {
        const contentEl = document.getElementById('nats-health-content');
        const unavailableEl = document.getElementById('nats-unavailable');
        const statusIndicator = document.getElementById('nats-connection-status');

        if (contentEl) contentEl.style.display = 'none';
        if (unavailableEl) unavailableEl.style.display = 'flex';
        if (statusIndicator) {
            statusIndicator.className = 'server-status status-offline';
        }
    }

    /**
     * Escape HTML to prevent XSS
     */
    private escapeHtml(text: string): string {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    /**
     * Refresh all data
     */
    async refresh(): Promise<void> {
        await this.loadData();
    }

    /**
     * Cleanup resources to prevent memory leaks
     * Clears state and references
     */
    destroy(): void {
        this.tautulliInfo = null;
        this.serverList = [];
        this.pmsUpdate = null;
        this.natsHealth = null;
        this.toastManager = null;
        this.container = null;
        this.initialized = false;
    }
}

export default ServerManagementManager;
