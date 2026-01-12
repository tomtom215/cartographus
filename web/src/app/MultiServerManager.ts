// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * MultiServerManager - Multi-server management UI (ADR-0026 Phase 4)
 *
 * Features:
 * - Display all configured servers (env + DB)
 * - Add/Edit/Delete UI-managed servers
 * - Connection testing with real-time feedback
 * - Server status indicators
 * - Enable/disable toggles
 * - RBAC-protected actions
 *
 * Reference: ADR-0026 Multi-Server Management UI
 */

import type { API } from '../lib/api';
import type { ToastManager } from '../lib/toast';
import type {
    MediaServerStatus,
    MediaServerListResponse,
    MediaServerPlatform,
    CreateMediaServerRequest,
    UpdateMediaServerRequest,
    MediaServerResponse,
} from '../lib/types/server';
import { PLATFORM_INFO, STATUS_INFO } from '../lib/types/server';
import { createLogger } from '../lib/logger';
import { getRoleGuard } from '../lib/auth/RoleGuard';

const logger = createLogger('MultiServerManager');

/**
 * Modal mode for add/edit
 */
type ModalMode = 'add' | 'edit';

/**
 * Platform icon SVG lookup
 */
const PLATFORM_ICONS: Record<MediaServerPlatform, string> = {
    plex: `<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
        <path d="M12 0L1.75 6v12L12 24l10.25-6V6L12 0zm0 4.5L18.5 9v6L12 19.5 5.5 15V9L12 4.5z"/>
    </svg>`,
    jellyfin: `<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
    </svg>`,
    emby: `<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
        <circle cx="12" cy="12" r="10"/>
    </svg>`,
    tautulli: `<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
        <path d="M12 2l9 4.5v9L12 20l-9-4.5v-9L12 2z"/>
    </svg>`
};

/**
 * Time format thresholds (in seconds)
 */
const TIME_THRESHOLDS = {
    MINUTE: 60,
    HOUR: 3600,
    DAY: 86400,
    WEEK: 604800
};

/**
 * Server being edited
 */
interface EditingServer {
    id?: string;
    platform: MediaServerPlatform;
    name: string;
    url: string;
    token: string;
    realtime_enabled: boolean;
    webhooks_enabled: boolean;
    session_polling_enabled: boolean;
    session_polling_interval: string;
}

export class MultiServerManager {
    private api: API;
    private toastManager: ToastManager | null = null;
    private initialized = false;

    // State
    private servers: MediaServerStatus[] = [];
    private serverCounts = {
        total: 0,
        connected: 0,
        syncing: 0,
        error: 0
    };
    private isLoading = false;
    private editingServer: EditingServer | null = null;
    private modalMode: ModalMode = 'add';
    private testInProgress = false;
    private deleteServerId: string | null = null;

    // DOM elements
    private container: HTMLElement | null = null;

    constructor(api: API) {
        this.api = api;
    }

    /**
     * Initialize the multi-server manager
     */
    init(containerId: string = 'multi-server-section'): void {
        this.container = document.getElementById(containerId);
        if (!this.container) {
            logger.warn('Multi-server section not found, creating', { containerId });
            this.createContainer(containerId);
        }

        // RBAC: Check read permission
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

        this.render();
        this.setupEventListeners();
        this.initialized = true;
        logger.debug('MultiServerManager initialized');

        // Load initial data
        this.loadServers();
    }

    /**
     * Set toast manager reference
     */
    setToastManager(toast: ToastManager): void {
        this.toastManager = toast;
    }

    /**
     * Create the container if it doesn't exist
     */
    private createContainer(containerId: string): void {
        // Try to find a suitable parent element
        const parent = document.getElementById('settings-content') ||
                       document.getElementById('admin-content') ||
                       document.querySelector('.main-content') ||
                       document.body;

        const section = document.createElement('div');
        section.id = containerId;
        section.className = 'multi-server-section';
        section.setAttribute('aria-label', 'Multi-server management');

        parent.appendChild(section);
        this.container = section;
    }

    /**
     * Main render method
     */
    private render(): void {
        if (!this.container) return;

        const roleGuard = getRoleGuard();
        const canManage = roleGuard.canAccess('server', 'admin');

        this.container.innerHTML = `
            <div class="multi-server-panel">
                ${this.renderHeader(canManage)}
                ${this.renderStatusSummary()}
                ${this.renderServerList(canManage)}
                ${this.renderModal()}
                ${this.renderDeleteConfirmModal()}
            </div>
        `;
    }

    /**
     * Render header section
     */
    private renderHeader(canManage: boolean): string {
        return `
            <div class="multi-server-header">
                <div class="multi-server-title">
                    <h3>Media Servers</h3>
                    <span class="server-count-badge" data-testid="server-count">${this.serverCounts.total} servers</span>
                </div>
                <div class="multi-server-actions">
                    <button id="btn-refresh-servers" type="button" class="btn-icon"
                            aria-label="Refresh servers" title="Refresh" data-testid="refresh-servers">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="23 4 23 10 17 10"/>
                            <polyline points="1 20 1 14 7 14"/>
                            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
                        </svg>
                    </button>
                    ${canManage ? `
                        <button id="btn-add-server" type="button" class="btn-primary"
                                aria-label="Add server" data-testid="add-server">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <line x1="12" y1="5" x2="12" y2="19"/>
                                <line x1="5" y1="12" x2="19" y2="12"/>
                            </svg>
                            Add Server
                        </button>
                    ` : ''}
                </div>
            </div>
        `;
    }

    /**
     * Render status summary cards
     */
    private renderStatusSummary(): string {
        return `
            <div class="server-status-summary" data-testid="status-summary">
                <div class="status-card status-connected" data-testid="connected-count">
                    <span class="status-count">${this.serverCounts.connected}</span>
                    <span class="status-label">Connected</span>
                </div>
                <div class="status-card status-syncing" data-testid="syncing-count">
                    <span class="status-count">${this.serverCounts.syncing}</span>
                    <span class="status-label">Syncing</span>
                </div>
                <div class="status-card status-error" data-testid="error-count">
                    <span class="status-count">${this.serverCounts.error}</span>
                    <span class="status-label">Errors</span>
                </div>
            </div>
        `;
    }

    /**
     * Render server list
     */
    private renderServerList(canManage: boolean): string {
        if (this.isLoading) {
            return `
                <div class="server-list-loading" data-testid="server-list-loading">
                    <div class="loading-spinner"></div>
                    <span>Loading servers...</span>
                </div>
            `;
        }

        if (this.servers.length === 0) {
            return `
                <div class="server-list-empty" data-testid="server-list-empty">
                    <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                        <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
                        <line x1="8" y1="21" x2="16" y2="21"/>
                        <line x1="12" y1="17" x2="12" y2="21"/>
                    </svg>
                    <p>No servers configured</p>
                    ${canManage ? '<p class="hint">Click "Add Server" to connect a media server</p>' : ''}
                </div>
            `;
        }

        return `
            <div class="server-list" role="list" aria-label="Media servers" data-testid="server-list">
                ${this.servers.map(server => this.renderServerCard(server, canManage)).join('')}
            </div>
        `;
    }

    /**
     * Render individual server card
     */
    private renderServerCard(server: MediaServerStatus, canManage: boolean): string {
        const platformInfo = PLATFORM_INFO[server.platform];
        const statusInfo = STATUS_INFO[server.status];
        const isEnvServer = server.source === 'env';

        return `
            <div class="server-card" data-server-id="${this.escapeHtml(server.id)}"
                 data-testid="server-card-${this.escapeHtml(server.id)}" role="listitem">
                ${this.renderServerCardHeader(server, platformInfo, statusInfo)}
                ${this.renderServerCardDetails(server)}
                ${this.renderServerCardFooter(server, isEnvServer, canManage)}
            </div>
        `;
    }

    /**
     * Render server card header section
     */
    private renderServerCardHeader(
        server: MediaServerStatus,
        platformInfo: typeof PLATFORM_INFO[keyof typeof PLATFORM_INFO],
        statusInfo: typeof STATUS_INFO[keyof typeof STATUS_INFO]
    ): string {
        return `
            <div class="server-card-header">
                <div class="server-info">
                    <div class="server-platform" style="color: ${platformInfo.color}">
                        ${this.renderPlatformIcon(server.platform)}
                        <span class="platform-name">${platformInfo.name}</span>
                    </div>
                    <h4 class="server-name" data-testid="server-name">${this.escapeHtml(server.name)}</h4>
                    <span class="server-url" data-testid="server-url">${this.escapeHtml(server.url)}</span>
                </div>
                <div class="server-status">
                    <span class="status-indicator" style="background-color: ${statusInfo.color}"
                          title="${statusInfo.label}" data-testid="status-indicator"></span>
                    <span class="status-text" data-testid="status-text">${statusInfo.label}</span>
                </div>
            </div>
        `;
    }

    /**
     * Render server card details section
     */
    private renderServerCardDetails(server: MediaServerStatus): string {
        return `
            <div class="server-details">
                ${this.renderServerFeatures(server)}
                ${this.renderServerSyncInfo(server)}
                ${this.renderServerError(server)}
            </div>
        `;
    }

    /**
     * Render server features badges
     */
    private renderServerFeatures(server: MediaServerStatus): string {
        const features = [
            server.realtime_enabled ? '<span class="feature-badge" title="Real-time updates enabled">Real-time</span>' : '',
            server.webhooks_enabled ? '<span class="feature-badge" title="Webhooks enabled">Webhooks</span>' : '',
            server.session_polling_enabled ? '<span class="feature-badge" title="Session polling enabled">Polling</span>' : ''
        ].filter(Boolean);

        return features.length > 0 ? `<div class="server-features">${features.join('')}</div>` : '';
    }

    /**
     * Render server sync info
     */
    private renderServerSyncInfo(server: MediaServerStatus): string {
        if (!server.last_sync_at) return '';

        return `
            <div class="server-sync-info">
                <span class="sync-label">Last sync:</span>
                <span class="sync-time" data-testid="last-sync">${this.formatRelativeTime(server.last_sync_at)}</span>
            </div>
        `;
    }

    /**
     * Render server error
     */
    private renderServerError(server: MediaServerStatus): string {
        if (!server.last_error) return '';

        return `
            <div class="server-error" data-testid="server-error">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <line x1="12" y1="8" x2="12" y2="12"/>
                    <line x1="12" y1="16" x2="12.01" y2="16"/>
                </svg>
                <span>${this.escapeHtml(server.last_error)}</span>
            </div>
        `;
    }

    /**
     * Render server card footer section
     */
    private renderServerCardFooter(server: MediaServerStatus, isEnvServer: boolean, canManage: boolean): string {
        return `
            <div class="server-card-footer">
                ${this.renderServerSource(isEnvServer)}
                ${this.renderServerActions(server, isEnvServer, canManage)}
            </div>
        `;
    }

    /**
     * Render server source badge
     */
    private renderServerSource(isEnvServer: boolean): string {
        if (isEnvServer) {
            return `
                <div class="server-source">
                    <span class="source-badge source-env" title="Configured via environment variables" data-testid="source-env">
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
                            <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                        </svg>
                        Environment
                    </span>
                </div>
            `;
        }
        return `
            <div class="server-source">
                <span class="source-badge source-ui" title="Added via UI" data-testid="source-ui">
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                        <line x1="3" y1="9" x2="21" y2="9"/>
                        <line x1="9" y1="21" x2="9" y2="9"/>
                    </svg>
                    UI
                </span>
            </div>
        `;
    }

    /**
     * Render server action buttons
     */
    private renderServerActions(server: MediaServerStatus, isEnvServer: boolean, canManage: boolean): string {
        if (!canManage) return '<div class="server-actions"></div>';

        const syncButton = `
            <button type="button" class="btn-icon btn-sync"
                    data-action="sync" data-server-id="${this.escapeHtml(server.id)}"
                    title="Trigger sync" aria-label="Trigger sync" data-testid="btn-sync">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="23 4 23 10 17 10"/>
                    <path d="M21 14a9 9 0 1 1-2.17-5.83"/>
                </svg>
            </button>
        `;

        const managementButtons = !isEnvServer ? this.renderManagementButtons(server) : '';

        return `<div class="server-actions">${syncButton}${managementButtons}</div>`;
    }

    /**
     * Render server management buttons (edit, delete, toggle)
     */
    private renderManagementButtons(server: MediaServerStatus): string {
        return `
            <label class="toggle-switch" title="${server.enabled ? 'Disable server' : 'Enable server'}">
                <input type="checkbox"
                       data-action="toggle"
                       data-server-id="${this.escapeHtml(server.id)}"
                       data-testid="toggle-enabled"
                       ${server.enabled ? 'checked' : ''}>
                <span class="toggle-slider"></span>
            </label>
            <button type="button" class="btn-icon btn-edit"
                    data-action="edit" data-server-id="${this.escapeHtml(server.id)}"
                    title="Edit server" aria-label="Edit server" data-testid="btn-edit">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                </svg>
            </button>
            <button type="button" class="btn-icon btn-delete"
                    data-action="delete" data-server-id="${this.escapeHtml(server.id)}"
                    title="Delete server" aria-label="Delete server" data-testid="btn-delete">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6"/>
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                </svg>
            </button>
        `;
    }

    /**
     * Render add/edit modal
     */
    private renderModal(): string {
        const isEditing = this.modalMode === 'edit';

        return `
            <div id="server-modal" class="modal-overlay" style="display: none;"
                 role="dialog" aria-modal="true" data-testid="server-modal">
                <div class="modal-content server-modal-content">
                    ${this.renderModalHeader(isEditing)}
                    ${this.renderModalForm(isEditing)}
                </div>
            </div>
        `;
    }

    /**
     * Render modal header
     */
    private renderModalHeader(isEditing: boolean): string {
        return `
            <div class="modal-header">
                <h3>${isEditing ? 'Edit Server' : 'Add Server'}</h3>
                <button type="button" class="btn-icon btn-close" id="btn-close-modal" aria-label="Close">
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <line x1="18" y1="6" x2="6" y2="18"/>
                        <line x1="6" y1="6" x2="18" y2="18"/>
                    </svg>
                </button>
            </div>
        `;
    }

    /**
     * Render modal form
     */
    private renderModalForm(isEditing: boolean): string {
        return `
            <form id="server-form" class="server-form" data-testid="server-form">
                ${this.renderFormFields(isEditing)}
                ${this.renderFormOptions()}
                ${this.renderTestResult()}
                ${this.renderModalFooter(isEditing)}
            </form>
        `;
    }

    /**
     * Render form fields (platform, name, url, token)
     */
    private renderFormFields(isEditing: boolean): string {
        const server = this.editingServer;

        return `
            <div class="form-group">
                <label for="server-platform">Platform</label>
                <select id="server-platform" name="platform" required ${isEditing ? 'disabled' : ''}
                        data-testid="select-platform">
                    <option value="">Select a platform...</option>
                    <option value="plex" ${server?.platform === 'plex' ? 'selected' : ''}>Plex</option>
                    <option value="jellyfin" ${server?.platform === 'jellyfin' ? 'selected' : ''}>Jellyfin</option>
                    <option value="emby" ${server?.platform === 'emby' ? 'selected' : ''}>Emby</option>
                    <option value="tautulli" ${server?.platform === 'tautulli' ? 'selected' : ''}>Tautulli</option>
                </select>
            </div>

            <div class="form-group">
                <label for="server-name">Display Name</label>
                <input type="text" id="server-name" name="name"
                       value="${this.escapeHtml(server?.name || '')}"
                       placeholder="e.g., Main Plex Server" required minlength="1" maxlength="100"
                       data-testid="input-name">
            </div>

            <div class="form-group">
                <label for="server-url">Server URL</label>
                <input type="url" id="server-url" name="url"
                       value="${this.escapeHtml(server?.url || '')}"
                       placeholder="e.g., http://localhost:32400" required
                       data-testid="input-url">
                <span class="form-hint">Include protocol (http/https) and port</span>
            </div>

            <div class="form-group">
                <label for="server-token">API Token</label>
                <div class="input-with-button">
                    <input type="password" id="server-token" name="token"
                           value="${this.escapeHtml(server?.token || '')}"
                           placeholder="${isEditing ? 'Leave blank to keep current' : 'Enter API token'}"
                           ${isEditing ? '' : 'required'} minlength="8"
                           data-testid="input-token">
                    <button type="button" class="btn-icon" id="btn-toggle-token"
                            aria-label="Toggle token visibility">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                            <circle cx="12" cy="12" r="3"/>
                        </svg>
                    </button>
                </div>
            </div>
        `;
    }

    /**
     * Render form options (checkboxes and polling interval)
     */
    private renderFormOptions(): string {
        const server = this.editingServer;

        return `
            <div class="form-group-row">
                <div class="form-group checkbox-group">
                    <label class="checkbox-label">
                        <input type="checkbox" id="server-realtime" name="realtime_enabled"
                               ${server?.realtime_enabled ? 'checked' : ''}
                               data-testid="checkbox-realtime">
                        <span>Enable real-time updates</span>
                    </label>
                </div>
                <div class="form-group checkbox-group">
                    <label class="checkbox-label">
                        <input type="checkbox" id="server-webhooks" name="webhooks_enabled"
                               ${server?.webhooks_enabled ? 'checked' : ''}
                               data-testid="checkbox-webhooks">
                        <span>Enable webhooks</span>
                    </label>
                </div>
            </div>

            <div class="form-group-row">
                <div class="form-group checkbox-group">
                    <label class="checkbox-label">
                        <input type="checkbox" id="server-polling" name="session_polling_enabled"
                               ${server?.session_polling_enabled ? 'checked' : ''}
                               data-testid="checkbox-polling">
                        <span>Enable session polling</span>
                    </label>
                </div>
                <div class="form-group">
                    <label for="server-polling-interval">Polling Interval</label>
                    <select id="server-polling-interval" name="session_polling_interval"
                            data-testid="select-polling-interval">
                        <option value="15s" ${server?.session_polling_interval === '15s' ? 'selected' : ''}>15 seconds</option>
                        <option value="30s" ${!server?.session_polling_interval || server?.session_polling_interval === '30s' ? 'selected' : ''}>30 seconds</option>
                        <option value="1m" ${server?.session_polling_interval === '1m' ? 'selected' : ''}>1 minute</option>
                        <option value="5m" ${server?.session_polling_interval === '5m' ? 'selected' : ''}>5 minutes</option>
                    </select>
                </div>
            </div>
        `;
    }

    /**
     * Render test result container
     */
    private renderTestResult(): string {
        return `
            <div id="test-result" class="test-result" style="display: none;" data-testid="test-result">
                <div class="test-result-content">
                    <span class="test-icon"></span>
                    <span class="test-message"></span>
                </div>
            </div>
        `;
    }

    /**
     * Render modal footer
     */
    private renderModalFooter(isEditing: boolean): string {
        return `
            <div class="modal-footer">
                <button type="button" class="btn-secondary" id="btn-test-connection"
                        data-testid="btn-test-connection"
                        ${this.testInProgress ? 'disabled' : ''}>
                    ${this.testInProgress ? 'Testing...' : 'Test Connection'}
                </button>
                <div class="modal-footer-right">
                    <button type="button" class="btn-secondary" id="btn-cancel-modal"
                            data-testid="btn-cancel">Cancel</button>
                    <button type="submit" class="btn-primary" id="btn-save-server"
                            data-testid="btn-save">
                        ${isEditing ? 'Save Changes' : 'Add Server'}
                    </button>
                </div>
            </div>
        `;
    }

    /**
     * Render delete confirmation modal
     */
    private renderDeleteConfirmModal(): string {
        return `
            <div id="delete-confirm-modal" class="modal-overlay" style="display: none;"
                 role="dialog" aria-modal="true" data-testid="delete-modal">
                <div class="modal-content delete-confirm-content">
                    <div class="modal-header">
                        <h3>Delete Server</h3>
                        <button type="button" class="btn-icon btn-close" id="btn-close-delete-modal" aria-label="Close">
                            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <line x1="18" y1="6" x2="6" y2="18"/>
                                <line x1="6" y1="6" x2="18" y2="18"/>
                            </svg>
                        </button>
                    </div>
                    <div class="modal-body">
                        <p>Are you sure you want to delete this server?</p>
                        <p class="delete-server-name" id="delete-server-name" data-testid="delete-server-name"></p>
                        <p class="delete-warning">This action cannot be undone.</p>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn-secondary" id="btn-cancel-delete"
                                data-testid="btn-cancel-delete">Cancel</button>
                        <button type="button" class="btn-danger" id="btn-confirm-delete"
                                data-testid="btn-confirm-delete">Delete</button>
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Setup event listeners
     */
    private setupEventListeners(): void {
        if (!this.container) return;

        // Use event delegation for all actions
        this.container.addEventListener('click', (e) => this.handleClick(e));
        this.container.addEventListener('submit', (e) => this.handleSubmit(e));
    }

    /**
     * Handle click events with delegation
     */
    private handleClick(e: Event): void {
        const target = e.target as HTMLElement;
        const button = target.closest('button');
        const checkbox = target.closest('input[type="checkbox"]');

        if (button) {
            this.handleButtonClick(button);
        }

        if (checkbox) {
            this.handleCheckboxClick(checkbox as HTMLInputElement);
        }
    }

    /**
     * Handle button clicks
     */
    private handleButtonClick(button: HTMLElement): void {
        const buttonId = button.id;
        const action = button.dataset.action;
        const serverId = button.dataset.serverId;

        // Handle buttons by ID
        this.handleButtonById(buttonId);

        // Handle buttons by action
        if (action && serverId) {
            this.handleServerAction(action, serverId);
        }
    }

    /**
     * Handle button actions by ID
     */
    private handleButtonById(buttonId: string): void {
        const handlers: Record<string, () => void> = {
            'btn-refresh-servers': () => this.loadServers(),
            'btn-add-server': () => this.showAddModal(),
            'btn-close-modal': () => this.closeModal(),
            'btn-cancel-modal': () => this.closeModal(),
            'btn-test-connection': () => this.testConnection(),
            'btn-toggle-token': () => this.toggleTokenVisibility(),
            'btn-close-delete-modal': () => this.closeDeleteModal(),
            'btn-cancel-delete': () => this.closeDeleteModal(),
            'btn-confirm-delete': () => this.confirmDelete()
        };

        const handler = handlers[buttonId];
        if (handler) {
            handler();
        }
    }

    /**
     * Handle server-specific actions
     */
    private handleServerAction(action: string, serverId: string): void {
        const actions: Record<string, (id: string) => void> = {
            'sync': (id) => this.triggerSync(id),
            'edit': (id) => this.showEditModal(id),
            'delete': (id) => this.showDeleteModal(id)
        };

        const handler = actions[action];
        if (handler) {
            handler(serverId);
        }
    }

    /**
     * Handle checkbox clicks
     */
    private handleCheckboxClick(checkbox: HTMLInputElement): void {
        const action = checkbox.dataset.action;
        const serverId = checkbox.dataset.serverId;

        if (action === 'toggle' && serverId) {
            this.toggleServer(serverId, checkbox.checked);
        }
    }

    /**
     * Handle form submission
     */
    private handleSubmit(e: Event): void {
        e.preventDefault();
        if ((e.target as HTMLElement).id === 'server-form') {
            this.saveServer();
        }
    }

    /**
     * Load servers from API
     */
    async loadServers(): Promise<void> {
        if (!this.initialized) return;

        this.isLoading = true;
        this.render();
        this.setupEventListeners();

        try {
            const response: MediaServerListResponse = await this.api.getServerStatus();
            this.servers = response.servers;
            this.serverCounts = {
                total: response.total_count,
                connected: response.connected_count,
                syncing: response.syncing_count,
                error: response.error_count
            };
            logger.debug('Servers loaded', { count: this.servers.length });
        } catch (error) {
            logger.error('Failed to load servers', { error });
            this.toastManager?.error('Failed to load servers');
        } finally {
            this.isLoading = false;
            this.render();
            this.setupEventListeners();
        }
    }

    /**
     * Show add server modal
     */
    private showAddModal(): void {
        this.modalMode = 'add';
        this.editingServer = {
            platform: 'plex' as MediaServerPlatform,
            name: '',
            url: '',
            token: '',
            realtime_enabled: false,
            webhooks_enabled: false,
            session_polling_enabled: false,
            session_polling_interval: '30s'
        };
        this.render();
        this.setupEventListeners();
        this.showModal();
    }

    /**
     * Show edit server modal
     */
    private async showEditModal(serverId: string): Promise<void> {
        try {
            const server: MediaServerResponse = await this.api.getServer(serverId);
            this.modalMode = 'edit';
            this.editingServer = {
                id: server.id,
                platform: server.platform as MediaServerPlatform,
                name: server.name,
                url: server.url,
                token: '', // Don't pre-fill token for security
                realtime_enabled: server.realtime_enabled,
                webhooks_enabled: server.webhooks_enabled,
                session_polling_enabled: server.session_polling_enabled,
                session_polling_interval: server.session_polling_interval || '30s'
            };
            this.render();
            this.setupEventListeners();
            this.showModal();
        } catch (error) {
            logger.error('Failed to load server for editing', { error, serverId });
            this.toastManager?.error('Failed to load server details');
        }
    }

    /**
     * Show modal
     */
    private showModal(): void {
        const modal = document.getElementById('server-modal');
        if (modal) {
            modal.style.display = 'flex';
            // Focus first input
            const firstInput = modal.querySelector('select:not([disabled]), input') as HTMLElement;
            firstInput?.focus();
        }
    }

    /**
     * Close modal
     */
    private closeModal(): void {
        const modal = document.getElementById('server-modal');
        if (modal) {
            modal.style.display = 'none';
        }
        this.editingServer = null;
    }

    /**
     * Test connection
     */
    private async testConnection(): Promise<void> {
        const form = document.getElementById('server-form') as HTMLFormElement;
        if (!form) return;

        const connectionParams = this.getConnectionParams(form);
        if (!connectionParams) {
            this.showTestResult(false, 'Please fill in platform, URL, and token');
            return;
        }

        this.testInProgress = true;
        this.updateTestButton('Testing...');

        try {
            const result = await this.api.testServerConnection(connectionParams);
            this.handleTestResult(result);
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Connection test failed';
            this.showTestResult(false, message);
        } finally {
            this.testInProgress = false;
            this.updateTestButton('Test Connection');
        }
    }

    /**
     * Get connection parameters from form
     */
    private getConnectionParams(form: HTMLFormElement): { platform: MediaServerPlatform; url: string; token: string } | null {
        const platform = (form.querySelector('#server-platform') as HTMLSelectElement)?.value as MediaServerPlatform;
        const url = (form.querySelector('#server-url') as HTMLInputElement)?.value;
        const token = (form.querySelector('#server-token') as HTMLInputElement)?.value;

        if (!platform || !url || !token) {
            return null;
        }

        return { platform, url, token };
    }

    /**
     * Handle test connection result
     */
    private handleTestResult(result: { success: boolean; server_name?: string; version?: string; error?: string }): void {
        if (result.success) {
            const serverInfo = `Connected! Server: ${result.server_name || 'Unknown'}, Version: ${result.version || 'Unknown'}`;
            this.showTestResult(true, serverInfo);
        } else {
            this.showTestResult(false, result.error || 'Connection failed');
        }
    }

    /**
     * Update test button text
     */
    private updateTestButton(text: string): void {
        const testBtn = document.getElementById('btn-test-connection');
        if (testBtn) {
            testBtn.textContent = text;
        }
    }

    /**
     * Show test result in modal
     */
    private showTestResult(success: boolean, message: string): void {
        const resultDiv = document.getElementById('test-result');
        if (!resultDiv) return;

        resultDiv.style.display = 'block';
        resultDiv.className = `test-result ${success ? 'test-success' : 'test-error'}`;
        resultDiv.innerHTML = `
            <div class="test-result-content">
                <span class="test-icon">${this.getTestResultIcon(success)}</span>
                <span class="test-message">${this.escapeHtml(message)}</span>
            </div>
        `;
    }

    /**
     * Get test result icon SVG
     */
    private getTestResultIcon(success: boolean): string {
        if (success) {
            return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
                <polyline points="22 4 12 14.01 9 11.01"/>
            </svg>`;
        }
        return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="15" y1="9" x2="9" y2="15"/>
            <line x1="9" y1="9" x2="15" y2="15"/>
        </svg>`;
    }

    /**
     * Toggle token visibility
     */
    private toggleTokenVisibility(): void {
        const tokenInput = document.getElementById('server-token') as HTMLInputElement;
        if (tokenInput) {
            tokenInput.type = tokenInput.type === 'password' ? 'text' : 'password';
        }
    }

    /**
     * Save server (create or update)
     */
    private async saveServer(): Promise<void> {
        const form = document.getElementById('server-form') as HTMLFormElement;
        if (!form) return;

        const formData = new FormData(form);
        const isEditing = this.modalMode === 'edit';

        try {
            if (isEditing && this.editingServer?.id) {
                await this.updateExistingServer(formData);
            } else {
                await this.createNewServer(formData);
            }

            this.closeModal();
            await this.loadServers();
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Failed to save server';
            logger.error('Failed to save server', { error });
            this.toastManager?.error(message);
        }
    }

    /**
     * Update existing server
     */
    private async updateExistingServer(formData: FormData): Promise<void> {
        if (!this.editingServer?.id) return;

        const updateRequest = this.buildUpdateRequest(formData);
        await this.api.updateServer(this.editingServer.id, updateRequest);
        this.toastManager?.success('Server updated successfully');
    }

    /**
     * Create new server
     */
    private async createNewServer(formData: FormData): Promise<void> {
        const createRequest = this.buildCreateRequest(formData);
        await this.api.createServer(createRequest);
        this.toastManager?.success('Server added successfully');
    }

    /**
     * Build update request from form data
     */
    private buildUpdateRequest(formData: FormData): UpdateMediaServerRequest {
        const request: UpdateMediaServerRequest = {
            name: formData.get('name') as string,
            url: formData.get('url') as string,
            realtime_enabled: formData.get('realtime_enabled') === 'on',
            webhooks_enabled: formData.get('webhooks_enabled') === 'on',
            session_polling_enabled: formData.get('session_polling_enabled') === 'on',
            session_polling_interval: formData.get('session_polling_interval') as string
        };

        // Only include token if provided
        const token = formData.get('token') as string;
        if (token) {
            request.token = token;
        }

        return request;
    }

    /**
     * Build create request from form data
     */
    private buildCreateRequest(formData: FormData): CreateMediaServerRequest {
        return {
            platform: formData.get('platform') as MediaServerPlatform,
            name: formData.get('name') as string,
            url: formData.get('url') as string,
            token: formData.get('token') as string,
            realtime_enabled: formData.get('realtime_enabled') === 'on',
            webhooks_enabled: formData.get('webhooks_enabled') === 'on',
            session_polling_enabled: formData.get('session_polling_enabled') === 'on',
            session_polling_interval: formData.get('session_polling_interval') as string
        };
    }

    /**
     * Toggle server enabled state
     */
    private async toggleServer(serverId: string, enabled: boolean): Promise<void> {
        try {
            await this.api.setServerEnabled(serverId, enabled);
            this.toastManager?.success(enabled ? 'Server enabled' : 'Server disabled');
            await this.loadServers();
        } catch (error) {
            logger.error('Failed to toggle server', { error, serverId });
            this.toastManager?.error('Failed to update server');
            await this.loadServers(); // Reload to revert checkbox state
        }
    }

    /**
     * Trigger sync for server
     */
    private async triggerSync(serverId: string): Promise<void> {
        try {
            await this.api.triggerServerSync(serverId, false);
            this.toastManager?.success('Sync triggered');
            // Delay reload to allow sync to start
            setTimeout(() => this.loadServers(), 1000);
        } catch (error) {
            logger.error('Failed to trigger sync', { error, serverId });
            this.toastManager?.error('Failed to trigger sync');
        }
    }

    /**
     * Show delete confirmation modal
     */
    private showDeleteModal(serverId: string): void {
        const server = this.servers.find(s => s.id === serverId);
        if (!server) return;

        this.deleteServerId = serverId;

        const modal = document.getElementById('delete-confirm-modal');
        const serverNameEl = document.getElementById('delete-server-name');

        if (modal) modal.style.display = 'flex';
        if (serverNameEl) serverNameEl.textContent = server.name;
    }

    /**
     * Close delete modal
     */
    private closeDeleteModal(): void {
        const modal = document.getElementById('delete-confirm-modal');
        if (modal) modal.style.display = 'none';
        this.deleteServerId = null;
    }

    /**
     * Confirm delete
     */
    private async confirmDelete(): Promise<void> {
        if (!this.deleteServerId) return;

        try {
            await this.api.deleteServer(this.deleteServerId);
            this.toastManager?.success('Server deleted');
            this.closeDeleteModal();
            await this.loadServers();
        } catch (error) {
            logger.error('Failed to delete server', { error, serverId: this.deleteServerId });
            this.toastManager?.error('Failed to delete server');
        }
    }

    /**
     * Render platform icon SVG
     */
    private renderPlatformIcon(platform: MediaServerPlatform): string {
        return PLATFORM_ICONS[platform] || `<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <rect x="2" y="3" width="20" height="14" rx="2"/>
        </svg>`;
    }

    /**
     * Format relative time
     */
    private formatRelativeTime(dateString: string): string {
        const date = new Date(dateString);
        const now = new Date();
        const diffSec = Math.floor((now.getTime() - date.getTime()) / 1000);

        if (diffSec < TIME_THRESHOLDS.MINUTE) return 'just now';
        if (diffSec < TIME_THRESHOLDS.HOUR) return `${Math.floor(diffSec / 60)}m ago`;
        if (diffSec < TIME_THRESHOLDS.DAY) return `${Math.floor(diffSec / 3600)}h ago`;
        if (diffSec < TIME_THRESHOLDS.WEEK) return `${Math.floor(diffSec / 86400)}d ago`;

        return date.toLocaleDateString();
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
        await this.loadServers();
    }

    /**
     * Cleanup resources to prevent memory leaks
     * Clears state and references
     */
    destroy(): void {
        this.servers = [];
        this.toastManager = null;
        this.editingServer = null;
        this.container = null;
        this.initialized = false;
    }
}

export default MultiServerManager;
