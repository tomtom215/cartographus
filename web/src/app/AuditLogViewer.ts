// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * AuditLogViewer - Security Audit Log Viewing UI
 *
 * Provides a dedicated admin interface for viewing and exporting security audit logs:
 * - View all audit events (authentication, authorization, detection, admin actions)
 * - Filter by event type, severity, outcome, actor, time range
 * - Full-text search across events
 * - Export to JSON or CEF format (for SIEM integration)
 * - Pagination and sorting
 * - Event detail view modal
 *
 * ADR-0015: Zero Trust Authentication & Authorization
 *
 * Security considerations:
 * - Admin-only access (requires admin role)
 * - Audit events are read-only
 * - Export includes all visible events for compliance
 */

import type { API } from '../lib/api';
import type { ToastManager } from '../lib/toast';
import type {
    AuditEvent,
    AuditEventType,
    AuditSeverity,
    AuditOutcome,
    AuditEventFilter,
    AuditStats,
} from '../lib/types/audit';
import { createLogger } from '../lib/logger';

const logger = createLogger('AuditLogViewer');

/** Default page size for audit events */
const DEFAULT_PAGE_SIZE = 50;

/** Event type display configuration */
const EVENT_TYPE_CONFIG: Record<string, { label: string; category: string; icon: string }> = {
    'auth.success': { label: 'Login Success', category: 'Authentication', icon: 'check-circle' },
    'auth.failure': { label: 'Login Failed', category: 'Authentication', icon: 'x-circle' },
    'auth.lockout': { label: 'Account Lockout', category: 'Authentication', icon: 'lock' },
    'auth.unlock': { label: 'Account Unlock', category: 'Authentication', icon: 'unlock' },
    'auth.logout': { label: 'Logout', category: 'Authentication', icon: 'log-out' },
    'auth.logout_all': { label: 'Logout All Sessions', category: 'Authentication', icon: 'log-out' },
    'auth.session_created': { label: 'Session Created', category: 'Authentication', icon: 'plus-circle' },
    'auth.session_expired': { label: 'Session Expired', category: 'Authentication', icon: 'clock' },
    'auth.token_revoked': { label: 'Token Revoked', category: 'Authentication', icon: 'slash' },
    'authz.granted': { label: 'Access Granted', category: 'Authorization', icon: 'check' },
    'authz.denied': { label: 'Access Denied', category: 'Authorization', icon: 'x' },
    'detection.alert': { label: 'Security Alert', category: 'Detection', icon: 'alert-triangle' },
    'detection.acknowledged': { label: 'Alert Acknowledged', category: 'Detection', icon: 'check-square' },
    'detection.rule_changed': { label: 'Rule Changed', category: 'Detection', icon: 'settings' },
    'user.created': { label: 'User Created', category: 'User Management', icon: 'user-plus' },
    'user.modified': { label: 'User Modified', category: 'User Management', icon: 'edit' },
    'user.deleted': { label: 'User Deleted', category: 'User Management', icon: 'user-minus' },
    'user.role_assigned': { label: 'Role Assigned', category: 'User Management', icon: 'shield' },
    'user.role_revoked': { label: 'Role Revoked', category: 'User Management', icon: 'shield-off' },
    'config.changed': { label: 'Config Changed', category: 'Administration', icon: 'sliders' },
    'data.export': { label: 'Data Export', category: 'Data', icon: 'download' },
    'data.import': { label: 'Data Import', category: 'Data', icon: 'upload' },
    'data.backup': { label: 'Backup Created', category: 'Data', icon: 'save' },
    'admin.action': { label: 'Admin Action', category: 'Administration', icon: 'terminal' },
};

/** Severity display configuration */
const SEVERITY_CONFIG: Record<AuditSeverity, { label: string; color: string }> = {
    'debug': { label: 'Debug', color: 'var(--text-muted)' },
    'info': { label: 'Info', color: 'var(--color-info)' },
    'warning': { label: 'Warning', color: 'var(--color-warning)' },
    'error': { label: 'Error', color: 'var(--color-error)' },
    'critical': { label: 'Critical', color: 'var(--color-critical, #d73a4a)' },
};

/** Outcome display configuration */
const OUTCOME_CONFIG: Record<AuditOutcome, { label: string; color: string }> = {
    'success': { label: 'Success', color: 'var(--color-success)' },
    'failure': { label: 'Failure', color: 'var(--color-error)' },
    'unknown': { label: 'Unknown', color: 'var(--text-muted)' },
};

export class AuditLogViewer {
    private api: API;
    private toastManager: ToastManager | null = null;
    private container: HTMLElement | null = null;
    private events: AuditEvent[] = [];
    private stats: AuditStats | null = null;
    private totalEvents: number = 0;
    private currentPage: number = 1;
    private pageSize: number = DEFAULT_PAGE_SIZE;
    private abortController: AbortController | null = null;
    private selectedEvent: AuditEvent | null = null;

    // Current filter state
    private filter: AuditEventFilter = {
        limit: DEFAULT_PAGE_SIZE,
        offset: 0,
        order_by: 'timestamp',
        order_direction: 'desc',
    };

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
     * Initialize the audit log viewer
     */
    async initialize(containerId: string): Promise<void> {
        this.container = document.getElementById(containerId);
        if (!this.container) {
            logger.warn(`Container #${containerId} not found`);
            return;
        }

        this.render();
        await this.loadStats();
        await this.loadEvents();
        this.setupEventListeners();
        logger.debug('AuditLogViewer initialized');
    }

    /**
     * Load audit statistics
     */
    private async loadStats(): Promise<void> {
        try {
            this.stats = await this.api.getAuditStats();
            this.updateStatsDisplay();
        } catch (error) {
            logger.error('Failed to load audit stats:', error);
        }
    }

    /**
     * Load audit events with current filter
     */
    private async loadEvents(): Promise<void> {
        this.setLoading(true);

        try {
            const response = await this.api.getAuditEvents(this.filter);
            this.events = response.events;
            this.totalEvents = response.total;
            this.updateEventsList();
            this.updatePagination();
        } catch (error) {
            logger.error('Failed to load audit events:', error);
            this.showError('Failed to load audit events. Please try again.');
        } finally {
            this.setLoading(false);
        }
    }

    /**
     * Render the audit log viewer panel
     */
    private render(): void {
        if (!this.container) return;

        this.container.innerHTML = `
            <div class="audit-log-viewer">
                ${this.renderHeader()}
                ${this.renderStats()}
                ${this.renderFilters()}
                ${this.renderEventsTable()}
                ${this.renderPagination()}
            </div>
            ${this.renderModal()}
        `;
    }

    /**
     * Render header section
     */
    private renderHeader(): string {
        return `
            <div class="audit-header">
                <div class="audit-header-content">
                    <h3 class="audit-title">
                        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                            <polyline points="14 2 14 8 20 8"/>
                            <line x1="16" y1="13" x2="8" y2="13"/>
                            <line x1="16" y1="17" x2="8" y2="17"/>
                            <polyline points="10 9 9 9 8 9"/>
                        </svg>
                        Security Audit Log
                    </h3>
                    <p class="audit-subtitle">
                        View and export security events for compliance and forensic analysis
                    </p>
                </div>
                <div class="audit-header-actions">
                    ${this.renderRefreshButton()}
                    ${this.renderExportDropdown()}
                </div>
            </div>
        `;
    }

    /**
     * Render refresh button
     */
    private renderRefreshButton(): string {
        return `
            <button class="audit-refresh-btn" id="audit-refresh-btn" title="Refresh">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="23 4 23 10 17 10"/>
                    <polyline points="1 20 1 14 7 14"/>
                    <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
                </svg>
            </button>
        `;
    }

    /**
     * Render export dropdown
     */
    private renderExportDropdown(): string {
        return `
            <div class="audit-export-dropdown">
                <button class="audit-export-btn" id="audit-export-btn" title="Export">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                        <polyline points="7 10 12 15 17 10"/>
                        <line x1="12" y1="15" x2="12" y2="3"/>
                    </svg>
                    Export
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="6 9 12 15 18 9"/>
                    </svg>
                </button>
                <div class="audit-export-menu" id="audit-export-menu">
                    <button class="audit-export-option" data-format="json">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="4 7 4 4 20 4 20 7"/>
                            <line x1="9" y1="20" x2="15" y2="20"/>
                            <line x1="12" y1="4" x2="12" y2="20"/>
                        </svg>
                        Export as JSON
                    </button>
                    <button class="audit-export-option" data-format="cef">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                            <line x1="3" y1="9" x2="21" y2="9"/>
                            <line x1="9" y1="21" x2="9" y2="9"/>
                        </svg>
                        Export as CEF (SIEM)
                    </button>
                </div>
            </div>
        `;
    }

    /**
     * Render stats section
     */
    private renderStats(): string {
        return `
            <div class="audit-stats" id="audit-stats">
                <div class="audit-stat-card">
                    <div class="audit-stat-value" id="stat-total">-</div>
                    <div class="audit-stat-label">Total Events</div>
                </div>
                <div class="audit-stat-card audit-stat-success">
                    <div class="audit-stat-value" id="stat-success">-</div>
                    <div class="audit-stat-label">Successful</div>
                </div>
                <div class="audit-stat-card audit-stat-failure">
                    <div class="audit-stat-value" id="stat-failure">-</div>
                    <div class="audit-stat-label">Failed</div>
                </div>
                <div class="audit-stat-card audit-stat-warning">
                    <div class="audit-stat-value" id="stat-warnings">-</div>
                    <div class="audit-stat-label">Warnings+</div>
                </div>
            </div>
        `;
    }

    /**
     * Render filters section
     */
    private renderFilters(): string {
        return `
            <div class="audit-filters">
                ${this.renderSearchInput()}
                ${this.renderFilterRow()}
            </div>
        `;
    }

    /**
     * Render search input
     */
    private renderSearchInput(): string {
        return `
            <div class="audit-search-wrapper">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="11" cy="11" r="8"/>
                    <line x1="21" y1="21" x2="16.65" y2="16.65"/>
                </svg>
                <input
                    type="text"
                    id="audit-search"
                    class="audit-search-input"
                    placeholder="Search events..."
                />
            </div>
        `;
    }

    /**
     * Render filter row with dropdowns
     */
    private renderFilterRow(): string {
        return `
            <div class="audit-filter-row">
                ${this.renderTypeFilter()}
                ${this.renderSeverityFilter()}
                ${this.renderOutcomeFilter()}
                ${this.renderDateRange()}
                ${this.renderClearButton()}
            </div>
        `;
    }

    /**
     * Render type filter dropdown
     */
    private renderTypeFilter(): string {
        return `
            <select id="audit-filter-type" class="audit-filter-select">
                <option value="">All Event Types</option>
                <optgroup label="Authentication">
                    <option value="auth.success">Login Success</option>
                    <option value="auth.failure">Login Failed</option>
                    <option value="auth.lockout">Account Lockout</option>
                    <option value="auth.unlock">Account Unlock</option>
                    <option value="auth.logout">Logout</option>
                    <option value="auth.logout_all">Logout All</option>
                    <option value="auth.session_created">Session Created</option>
                    <option value="auth.session_expired">Session Expired</option>
                    <option value="auth.token_revoked">Token Revoked</option>
                </optgroup>
                <optgroup label="Authorization">
                    <option value="authz.granted">Access Granted</option>
                    <option value="authz.denied">Access Denied</option>
                </optgroup>
                <optgroup label="Detection">
                    <option value="detection.alert">Security Alert</option>
                    <option value="detection.acknowledged">Alert Acknowledged</option>
                    <option value="detection.rule_changed">Rule Changed</option>
                </optgroup>
                <optgroup label="User Management">
                    <option value="user.created">User Created</option>
                    <option value="user.modified">User Modified</option>
                    <option value="user.deleted">User Deleted</option>
                    <option value="user.role_assigned">Role Assigned</option>
                    <option value="user.role_revoked">Role Revoked</option>
                </optgroup>
                <optgroup label="Administration">
                    <option value="config.changed">Config Changed</option>
                    <option value="admin.action">Admin Action</option>
                </optgroup>
                <optgroup label="Data">
                    <option value="data.export">Data Export</option>
                    <option value="data.import">Data Import</option>
                    <option value="data.backup">Data Backup</option>
                </optgroup>
            </select>
        `;
    }

    /**
     * Render severity filter dropdown
     */
    private renderSeverityFilter(): string {
        return `
            <select id="audit-filter-severity" class="audit-filter-select">
                <option value="">All Severities</option>
                <option value="critical">Critical</option>
                <option value="error">Error</option>
                <option value="warning">Warning</option>
                <option value="info">Info</option>
                <option value="debug">Debug</option>
            </select>
        `;
    }

    /**
     * Render outcome filter dropdown
     */
    private renderOutcomeFilter(): string {
        return `
            <select id="audit-filter-outcome" class="audit-filter-select">
                <option value="">All Outcomes</option>
                <option value="success">Success</option>
                <option value="failure">Failure</option>
                <option value="unknown">Unknown</option>
            </select>
        `;
    }

    /**
     * Render date range inputs
     */
    private renderDateRange(): string {
        return `
            <div class="audit-date-range">
                <input
                    type="date"
                    id="audit-date-start"
                    class="audit-date-input"
                    title="Start date"
                />
                <span class="audit-date-separator">to</span>
                <input
                    type="date"
                    id="audit-date-end"
                    class="audit-date-input"
                    title="End date"
                />
            </div>
        `;
    }

    /**
     * Render clear filters button
     */
    private renderClearButton(): string {
        return `
            <button class="audit-filter-clear-btn" id="audit-filter-clear" title="Clear filters">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="18" y1="6" x2="6" y2="18"/>
                    <line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
                Clear
            </button>
        `;
    }

    /**
     * Render events table
     */
    private renderEventsTable(): string {
        return `
            <div class="audit-events-container">
                <table class="audit-events-table">
                    <thead>
                        <tr>
                            <th class="audit-col-time">
                                <button class="audit-sort-btn" data-column="timestamp">
                                    Time
                                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                        <polyline points="6 9 12 15 18 9"/>
                                    </svg>
                                </button>
                            </th>
                            <th class="audit-col-type">Type</th>
                            <th class="audit-col-severity">Severity</th>
                            <th class="audit-col-outcome">Outcome</th>
                            <th class="audit-col-actor">Actor</th>
                            <th class="audit-col-action">Action</th>
                            <th class="audit-col-source">Source IP</th>
                            <th class="audit-col-details"></th>
                        </tr>
                    </thead>
                    <tbody id="audit-events-body">
                        <tr class="audit-loading-row">
                            <td colspan="8">
                                <div class="audit-loading">
                                    <div class="audit-spinner"></div>
                                    <span>Loading events...</span>
                                </div>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        `;
    }

    /**
     * Render pagination controls
     */
    private renderPagination(): string {
        return `
            <div class="audit-pagination" id="audit-pagination">
                <div class="audit-pagination-info">
                    Showing <span id="pagination-start">0</span>-<span id="pagination-end">0</span>
                    of <span id="pagination-total">0</span> events
                </div>
                <div class="audit-pagination-controls">
                    <button class="audit-page-btn" id="audit-page-first" title="First page">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="11 17 6 12 11 7"/>
                            <polyline points="18 17 13 12 18 7"/>
                        </svg>
                    </button>
                    <button class="audit-page-btn" id="audit-page-prev" title="Previous page">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="15 18 9 12 15 6"/>
                        </svg>
                    </button>
                    <span class="audit-page-indicator">
                        Page <span id="current-page">1</span> of <span id="total-pages">1</span>
                    </span>
                    <button class="audit-page-btn" id="audit-page-next" title="Next page">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="9 18 15 12 9 6"/>
                        </svg>
                    </button>
                    <button class="audit-page-btn" id="audit-page-last" title="Last page">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="13 17 18 12 13 7"/>
                            <polyline points="6 17 11 12 6 7"/>
                        </svg>
                    </button>
                </div>
            </div>
        `;
    }

    /**
     * Render event detail modal
     */
    private renderModal(): string {
        return `
            <div class="audit-modal-overlay" id="audit-event-modal">
                <div class="audit-modal">
                    <div class="audit-modal-header">
                        <h4 class="audit-modal-title">Event Details</h4>
                        <button class="audit-modal-close" id="audit-modal-close" title="Close">
                            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <line x1="18" y1="6" x2="6" y2="18"/>
                                <line x1="6" y1="6" x2="18" y2="18"/>
                            </svg>
                        </button>
                    </div>
                    <div class="audit-modal-body" id="audit-modal-body">
                        <!-- Event details will be rendered here -->
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Update the stats display
     */
    private updateStatsDisplay(): void {
        if (!this.stats) return;

        const warningCount = this.calculateWarningPlusCount();

        const statUpdates = [
            { id: 'stat-total', value: this.stats.total_events },
            { id: 'stat-success', value: this.stats.events_by_outcome?.success || 0 },
            { id: 'stat-failure', value: this.stats.events_by_outcome?.failure || 0 },
            { id: 'stat-warnings', value: warningCount },
        ];

        statUpdates.forEach(({ id, value }) => {
            const element = document.getElementById(id);
            if (element) {
                element.textContent = this.formatNumber(value);
            }
        });
    }

    /**
     * Calculate warning+ count (warning + error + critical)
     */
    private calculateWarningPlusCount(): number {
        if (!this.stats?.events_by_severity) return 0;

        return (this.stats.events_by_severity.warning || 0) +
               (this.stats.events_by_severity.error || 0) +
               (this.stats.events_by_severity.critical || 0);
    }

    /**
     * Update the events list with current data
     */
    private updateEventsList(): void {
        const tbody = document.getElementById('audit-events-body');
        if (!tbody) return;

        if (this.events.length === 0) {
            tbody.innerHTML = `
                <tr class="audit-empty-row">
                    <td colspan="8">
                        <div class="audit-empty">
                            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                                <circle cx="12" cy="12" r="10"/>
                                <line x1="12" y1="8" x2="12" y2="12"/>
                                <line x1="12" y1="16" x2="12.01" y2="16"/>
                            </svg>
                            <p>No audit events found</p>
                            <span>Try adjusting your filters</span>
                        </div>
                    </td>
                </tr>
            `;
            return;
        }

        tbody.innerHTML = this.events.map(event => this.renderEventRow(event)).join('');
    }

    /**
     * Render a single event row
     */
    private renderEventRow(event: AuditEvent): string {
        const typeConfig = EVENT_TYPE_CONFIG[event.type] || {
            label: event.type,
            category: 'Other',
            icon: 'circle'
        };
        const severityConfig = SEVERITY_CONFIG[event.severity];
        const outcomeConfig = OUTCOME_CONFIG[event.outcome];
        const timestamp = new Date(event.timestamp);
        const formattedTime = this.formatDateTime(timestamp);
        const relativeTime = this.formatRelativeTime(timestamp);

        return `
            <tr class="audit-event-row" data-event-id="${event.id}">
                <td class="audit-col-time" title="${formattedTime}">
                    <span class="audit-time-relative">${relativeTime}</span>
                    <span class="audit-time-full">${this.formatTimeShort(timestamp)}</span>
                </td>
                <td class="audit-col-type">
                    <span class="audit-type-badge" data-category="${typeConfig.category}">
                        ${typeConfig.label}
                    </span>
                </td>
                <td class="audit-col-severity">
                    <span class="audit-severity-badge audit-severity-${event.severity}"
                          style="--severity-color: ${severityConfig.color}">
                        ${severityConfig.label}
                    </span>
                </td>
                <td class="audit-col-outcome">
                    <span class="audit-outcome-badge audit-outcome-${event.outcome}"
                          style="--outcome-color: ${outcomeConfig.color}">
                        ${outcomeConfig.label}
                    </span>
                </td>
                <td class="audit-col-actor">
                    <div class="audit-actor">
                        <span class="audit-actor-name" title="${event.actor.id}">
                            ${event.actor.name || event.actor.id || 'System'}
                        </span>
                        ${event.actor.type !== 'user' ? `
                            <span class="audit-actor-type">${event.actor.type}</span>
                        ` : ''}
                    </div>
                </td>
                <td class="audit-col-action">
                    <span class="audit-action" title="${event.description}">
                        ${event.action}
                    </span>
                </td>
                <td class="audit-col-source">
                    <span class="audit-ip" title="${event.source.user_agent || 'Unknown user agent'}">
                        ${event.source.ip_address || '-'}
                    </span>
                </td>
                <td class="audit-col-details">
                    <button class="audit-details-btn" data-event-id="${event.id}" title="View details">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <circle cx="12" cy="12" r="3"/>
                            <path d="M12 1v6m0 6v10M4.22 4.22l4.24 4.24m7.08 7.08l4.24 4.24M1 12h6m6 0h10M4.22 19.78l4.24-4.24m7.08-7.08l4.24-4.24"/>
                        </svg>
                    </button>
                </td>
            </tr>
        `;
    }

    /**
     * Update pagination controls
     */
    private updatePagination(): void {
        const totalPages = Math.max(1, Math.ceil(this.totalEvents / this.pageSize));
        const start = Math.min(this.filter.offset! + 1, this.totalEvents);
        const end = Math.min(this.filter.offset! + this.pageSize, this.totalEvents);

        this.updatePaginationInfo(start, end, totalPages);
        this.updatePaginationButtons(totalPages);
    }

    /**
     * Update pagination info display
     */
    private updatePaginationInfo(start: number, end: number, totalPages: number): void {
        const updates = [
            { id: 'pagination-start', value: start },
            { id: 'pagination-end', value: end },
            { id: 'pagination-total', value: this.totalEvents },
            { id: 'current-page', value: this.currentPage },
            { id: 'total-pages', value: totalPages },
        ];

        updates.forEach(({ id, value }) => {
            const element = document.getElementById(id);
            if (element) {
                element.textContent = String(value);
            }
        });
    }

    /**
     * Update pagination button states
     */
    private updatePaginationButtons(totalPages: number): void {
        const buttonStates = [
            { id: 'audit-page-first', disabled: this.currentPage <= 1 },
            { id: 'audit-page-prev', disabled: this.currentPage <= 1 },
            { id: 'audit-page-next', disabled: this.currentPage >= totalPages },
            { id: 'audit-page-last', disabled: this.currentPage >= totalPages },
        ];

        buttonStates.forEach(({ id, disabled }) => {
            const button = document.getElementById(id) as HTMLButtonElement;
            if (button) {
                button.disabled = disabled;
            }
        });
    }

    /**
     * Setup event listeners
     */
    private setupEventListeners(): void {
        if (!this.container) return;

        this.abortController = new AbortController();
        const signal = this.abortController.signal;

        this.setupHeaderListeners(signal);
        this.setupFilterListeners(signal);
        this.setupPaginationListeners(signal);
        this.setupModalListeners(signal);
        this.setupSortListeners(signal);
        this.setupEventDetailsListeners(signal);
    }

    /**
     * Setup header button listeners (refresh, export)
     */
    private setupHeaderListeners(signal: AbortSignal): void {
        const refreshBtn = this.container?.querySelector('#audit-refresh-btn');
        refreshBtn?.addEventListener('click', () => this.refresh(), { signal });

        const exportBtn = this.container?.querySelector('#audit-export-btn');
        const exportMenu = this.container?.querySelector('#audit-export-menu');

        exportBtn?.addEventListener('click', (e) => {
            e.stopPropagation();
            exportMenu?.classList.toggle('show');
        }, { signal });

        const exportOptions = this.container?.querySelectorAll('.audit-export-option');
        exportOptions?.forEach(option => {
            option.addEventListener('click', (e) => {
                const format = (e.currentTarget as HTMLElement).dataset.format as 'json' | 'cef';
                this.handleExport(format);
                exportMenu?.classList.remove('show');
            }, { signal });
        });

        document.addEventListener('click', () => {
            exportMenu?.classList.remove('show');
        }, { signal });
    }

    /**
     * Setup filter input listeners
     */
    private setupFilterListeners(signal: AbortSignal): void {
        this.setupSearchListener(signal);
        this.setupSelectFilters(signal);
        this.setupDateFilters(signal);

        const clearBtn = this.container?.querySelector('#audit-filter-clear');
        clearBtn?.addEventListener('click', () => this.clearFilters(), { signal });
    }

    /**
     * Setup search input listener with debouncing
     */
    private setupSearchListener(signal: AbortSignal): void {
        const searchInput = this.container?.querySelector('#audit-search') as HTMLInputElement;
        let searchTimeout: ReturnType<typeof setTimeout>;

        searchInput?.addEventListener('input', () => {
            clearTimeout(searchTimeout);
            searchTimeout = setTimeout(() => {
                this.filter.search = searchInput.value || undefined;
                this.resetToFirstPage();
                this.loadEvents();
            }, 300);
        }, { signal });
    }

    /**
     * Setup select filter listeners
     */
    private setupSelectFilters(signal: AbortSignal): void {
        const typeFilter = this.container?.querySelector('#audit-filter-type') as HTMLSelectElement;
        typeFilter?.addEventListener('change', () => {
            this.filter.types = typeFilter.value ? [typeFilter.value as AuditEventType] : undefined;
            this.resetToFirstPage();
            this.loadEvents();
        }, { signal });

        const severityFilter = this.container?.querySelector('#audit-filter-severity') as HTMLSelectElement;
        severityFilter?.addEventListener('change', () => {
            this.filter.severities = severityFilter.value ? [severityFilter.value as AuditSeverity] : undefined;
            this.resetToFirstPage();
            this.loadEvents();
        }, { signal });

        const outcomeFilter = this.container?.querySelector('#audit-filter-outcome') as HTMLSelectElement;
        outcomeFilter?.addEventListener('change', () => {
            this.filter.outcomes = outcomeFilter.value ? [outcomeFilter.value as AuditOutcome] : undefined;
            this.resetToFirstPage();
            this.loadEvents();
        }, { signal });
    }

    /**
     * Setup date range filter listeners
     */
    private setupDateFilters(signal: AbortSignal): void {
        const startDate = this.container?.querySelector('#audit-date-start') as HTMLInputElement;
        const endDate = this.container?.querySelector('#audit-date-end') as HTMLInputElement;

        startDate?.addEventListener('change', () => {
            this.filter.start_time = startDate.value ? new Date(startDate.value).toISOString() : undefined;
            this.resetToFirstPage();
            this.loadEvents();
        }, { signal });

        endDate?.addEventListener('change', () => {
            this.filter.end_time = this.getEndOfDayISO(endDate.value);
            this.resetToFirstPage();
            this.loadEvents();
        }, { signal });
    }

    /**
     * Setup pagination button listeners
     */
    private setupPaginationListeners(signal: AbortSignal): void {
        const paginationButtons = [
            { id: 'audit-page-first', action: () => this.goToPage(1) },
            { id: 'audit-page-prev', action: () => this.goToPage(this.currentPage - 1) },
            { id: 'audit-page-next', action: () => this.goToPage(this.currentPage + 1) },
            { id: 'audit-page-last', action: () => this.goToPage(Math.ceil(this.totalEvents / this.pageSize)) },
        ];

        paginationButtons.forEach(({ id, action }) => {
            const btn = this.container?.querySelector(`#${id}`);
            btn?.addEventListener('click', action, { signal });
        });
    }

    /**
     * Setup modal close listeners
     */
    private setupModalListeners(signal: AbortSignal): void {
        const modalClose = this.container?.querySelector('#audit-modal-close');
        modalClose?.addEventListener('click', () => this.closeModal(), { signal });

        const modalOverlay = this.container?.querySelector('#audit-event-modal');
        modalOverlay?.addEventListener('click', (e) => {
            if (e.target === modalOverlay) {
                this.closeModal();
            }
        }, { signal });

        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && this.selectedEvent) {
                this.closeModal();
            }
        }, { signal });
    }

    /**
     * Setup event details button listeners
     */
    private setupEventDetailsListeners(signal: AbortSignal): void {
        const eventsBody = this.container?.querySelector('#audit-events-body');
        eventsBody?.addEventListener('click', (e) => {
            const target = e.target as HTMLElement;
            const detailsBtn = target.closest('.audit-details-btn') as HTMLButtonElement;

            if (detailsBtn) {
                const eventId = detailsBtn.dataset.eventId;
                if (eventId) {
                    this.showEventDetails(eventId);
                }
            }
        }, { signal });
    }

    /**
     * Setup sort button listeners
     */
    private setupSortListeners(signal: AbortSignal): void {
        const sortBtns = this.container?.querySelectorAll('.audit-sort-btn');
        sortBtns?.forEach(btn => {
            btn.addEventListener('click', () => {
                const column = (btn as HTMLElement).dataset.column;
                if (column) {
                    this.toggleSort(column);
                }
            }, { signal });
        });
    }

    /**
     * Helper to reset pagination to first page
     */
    private resetToFirstPage(): void {
        this.currentPage = 1;
        this.filter.offset = 0;
    }

    /**
     * Helper to get end of day ISO string
     */
    private getEndOfDayISO(dateValue: string): string | undefined {
        if (!dateValue) return undefined;

        const date = new Date(dateValue);
        date.setHours(23, 59, 59, 999);
        return date.toISOString();
    }

    /**
     * Go to a specific page
     */
    private async goToPage(page: number): Promise<void> {
        const totalPages = Math.ceil(this.totalEvents / this.pageSize);
        if (page < 1 || page > totalPages) return;

        this.currentPage = page;
        this.filter.offset = (page - 1) * this.pageSize;
        await this.loadEvents();
    }

    /**
     * Toggle sort direction for a column
     */
    private async toggleSort(column: string): Promise<void> {
        if (this.filter.order_by === column) {
            this.filter.order_direction = this.filter.order_direction === 'desc' ? 'asc' : 'desc';
        } else {
            this.filter.order_by = column;
            this.filter.order_direction = 'desc';
        }
        await this.loadEvents();
    }

    /**
     * Clear all filters
     */
    private async clearFilters(): Promise<void> {
        this.filter = {
            limit: DEFAULT_PAGE_SIZE,
            offset: 0,
            order_by: 'timestamp',
            order_direction: 'desc',
        };
        this.currentPage = 1;

        // Reset UI
        this.resetFilterInputs();
        await this.loadEvents();
    }

    /**
     * Reset all filter input values
     */
    private resetFilterInputs(): void {
        const filterIds = [
            'audit-search',
            'audit-filter-type',
            'audit-filter-severity',
            'audit-filter-outcome',
            'audit-date-start',
            'audit-date-end',
        ];

        filterIds.forEach(id => {
            const input = this.container?.querySelector(`#${id}`) as HTMLInputElement | HTMLSelectElement;
            if (input) {
                input.value = '';
            }
        });
    }

    /**
     * Show event details modal
     */
    private async showEventDetails(eventId: string): Promise<void> {
        const event = this.events.find(e => e.id === eventId);
        if (!event) {
            // Try to fetch from API
            try {
                this.selectedEvent = await this.api.getAuditEvent(eventId);
            } catch (error) {
                logger.error('Failed to load event details:', error);
                this.toastManager?.error('Failed to load event details');
                return;
            }
        } else {
            this.selectedEvent = event;
        }

        const modalBody = document.getElementById('audit-modal-body');
        if (modalBody) {
            modalBody.innerHTML = this.renderEventDetails(this.selectedEvent);
        }

        const modal = document.getElementById('audit-event-modal');
        modal?.classList.add('show');
    }

    /**
     * Render event details for modal
     */
    private renderEventDetails(event: AuditEvent): string {
        return `
            <div class="audit-detail-grid">
                ${this.renderEventInfoSection(event)}
                ${this.renderActorSection(event)}
                ${this.renderSourceSection(event)}
                ${this.renderTargetSection(event)}
                ${this.renderCorrelationSection(event)}
                ${this.renderMetadataSection(event)}
            </div>
        `;
    }

    /**
     * Render event information section
     */
    private renderEventInfoSection(event: AuditEvent): string {
        const typeConfig = EVENT_TYPE_CONFIG[event.type] || { label: event.type, category: 'Other' };
        const severityConfig = SEVERITY_CONFIG[event.severity];
        const outcomeConfig = OUTCOME_CONFIG[event.outcome];
        const timestamp = new Date(event.timestamp);

        return `
            <div class="audit-detail-section">
                <h5 class="audit-detail-section-title">Event Information</h5>
                <div class="audit-detail-row">
                    <span class="audit-detail-label">Event ID</span>
                    <span class="audit-detail-value audit-detail-mono">${event.id}</span>
                </div>
                <div class="audit-detail-row">
                    <span class="audit-detail-label">Timestamp</span>
                    <span class="audit-detail-value">${this.formatDateTime(timestamp)}</span>
                </div>
                <div class="audit-detail-row">
                    <span class="audit-detail-label">Type</span>
                    <span class="audit-detail-value">
                        <span class="audit-type-badge">${typeConfig.label}</span>
                        <span class="audit-detail-category">(${typeConfig.category})</span>
                    </span>
                </div>
                <div class="audit-detail-row">
                    <span class="audit-detail-label">Severity</span>
                    <span class="audit-detail-value">
                        <span class="audit-severity-badge audit-severity-${event.severity}"
                              style="--severity-color: ${severityConfig.color}">
                            ${severityConfig.label}
                        </span>
                    </span>
                </div>
                <div class="audit-detail-row">
                    <span class="audit-detail-label">Outcome</span>
                    <span class="audit-detail-value">
                        <span class="audit-outcome-badge audit-outcome-${event.outcome}"
                              style="--outcome-color: ${outcomeConfig.color}">
                            ${outcomeConfig.label}
                        </span>
                    </span>
                </div>
                <div class="audit-detail-row">
                    <span class="audit-detail-label">Action</span>
                    <span class="audit-detail-value">${event.action}</span>
                </div>
                <div class="audit-detail-row">
                    <span class="audit-detail-label">Description</span>
                    <span class="audit-detail-value">${event.description}</span>
                </div>
            </div>
        `;
    }

    /**
     * Render actor section
     */
    private renderActorSection(event: AuditEvent): string {
        const optionalRows = [
            { condition: event.actor.roles?.length, label: 'Roles', value: event.actor.roles?.join(', ') },
            { condition: event.actor.auth_method, label: 'Auth Method', value: event.actor.auth_method },
            { condition: event.actor.session_id, label: 'Session ID', value: event.actor.session_id, mono: true },
        ];

        const optionalHtml = optionalRows
            .filter(row => row.condition)
            .map(row => `
                <div class="audit-detail-row">
                    <span class="audit-detail-label">${row.label}</span>
                    <span class="audit-detail-value ${row.mono ? 'audit-detail-mono' : ''}">${row.value}</span>
                </div>
            `)
            .join('');

        return `
            <div class="audit-detail-section">
                <h5 class="audit-detail-section-title">Actor</h5>
                <div class="audit-detail-row">
                    <span class="audit-detail-label">ID</span>
                    <span class="audit-detail-value audit-detail-mono">${event.actor.id || '-'}</span>
                </div>
                <div class="audit-detail-row">
                    <span class="audit-detail-label">Name</span>
                    <span class="audit-detail-value">${event.actor.name || '-'}</span>
                </div>
                <div class="audit-detail-row">
                    <span class="audit-detail-label">Type</span>
                    <span class="audit-detail-value">${event.actor.type}</span>
                </div>
                ${optionalHtml}
            </div>
        `;
    }

    /**
     * Render source section
     */
    private renderSourceSection(event: AuditEvent): string {
        const optionalRows = [
            { condition: event.source.hostname, label: 'Hostname', value: event.source.hostname },
            { condition: event.source.user_agent, label: 'User Agent', value: event.source.user_agent, wrap: true },
        ];

        const optionalHtml = optionalRows
            .filter(row => row.condition)
            .map(row => `
                <div class="audit-detail-row">
                    <span class="audit-detail-label">${row.label}</span>
                    <span class="audit-detail-value ${row.wrap ? 'audit-detail-wrap' : ''}">${row.value}</span>
                </div>
            `)
            .join('');

        const geoHtml = event.source.geo ? `
            <div class="audit-detail-row">
                <span class="audit-detail-label">Location</span>
                <span class="audit-detail-value">
                    ${[event.source.geo.city, event.source.geo.region, event.source.geo.country]
                        .filter(Boolean).join(', ') || '-'}
                </span>
            </div>
        ` : '';

        return `
            <div class="audit-detail-section">
                <h5 class="audit-detail-section-title">Source</h5>
                <div class="audit-detail-row">
                    <span class="audit-detail-label">IP Address</span>
                    <span class="audit-detail-value audit-detail-mono">${event.source.ip_address || '-'}</span>
                </div>
                ${optionalHtml}
                ${geoHtml}
            </div>
        `;
    }

    /**
     * Render target section
     */
    private renderTargetSection(event: AuditEvent): string {
        if (!event.target) return '';

        const nameRow = event.target.name ? `
            <div class="audit-detail-row">
                <span class="audit-detail-label">Name</span>
                <span class="audit-detail-value">${event.target.name}</span>
            </div>
        ` : '';

        return `
            <div class="audit-detail-section">
                <h5 class="audit-detail-section-title">Target</h5>
                <div class="audit-detail-row">
                    <span class="audit-detail-label">ID</span>
                    <span class="audit-detail-value audit-detail-mono">${event.target.id}</span>
                </div>
                <div class="audit-detail-row">
                    <span class="audit-detail-label">Type</span>
                    <span class="audit-detail-value">${event.target.type}</span>
                </div>
                ${nameRow}
            </div>
        `;
    }

    /**
     * Render correlation section
     */
    private renderCorrelationSection(event: AuditEvent): string {
        const hasCorrelation = event.request_id || event.correlation_id;

        const requestIdRow = event.request_id ? `
            <div class="audit-detail-row">
                <span class="audit-detail-label">Request ID</span>
                <span class="audit-detail-value audit-detail-mono">${event.request_id}</span>
            </div>
        ` : '';

        const correlationIdRow = event.correlation_id ? `
            <div class="audit-detail-row">
                <span class="audit-detail-label">Correlation ID</span>
                <span class="audit-detail-value audit-detail-mono">${event.correlation_id}</span>
            </div>
        ` : '';

        const emptyRow = !hasCorrelation ? `
            <div class="audit-detail-row">
                <span class="audit-detail-value audit-detail-muted">No correlation data</span>
            </div>
        ` : '';

        return `
            <div class="audit-detail-section">
                <h5 class="audit-detail-section-title">Correlation</h5>
                ${requestIdRow}
                ${correlationIdRow}
                ${emptyRow}
            </div>
        `;
    }

    /**
     * Render metadata section
     */
    private renderMetadataSection(event: AuditEvent): string {
        if (!event.metadata || Object.keys(event.metadata).length === 0) {
            return '';
        }

        return `
            <div class="audit-detail-section audit-detail-full-width">
                <h5 class="audit-detail-section-title">Metadata</h5>
                <pre class="audit-detail-json">${JSON.stringify(event.metadata, null, 2)}</pre>
            </div>
        `;
    }

    /**
     * Close the event detail modal
     */
    private closeModal(): void {
        const modal = document.getElementById('audit-event-modal');
        modal?.classList.remove('show');
        this.selectedEvent = null;
    }

    /**
     * Handle export
     */
    private handleExport(format: 'json' | 'cef'): void {
        try {
            const url = this.api.getAuditExportUrl(format, this.filter);
            window.open(url, '_blank');
            this.toastManager?.success(`Exporting audit log as ${format.toUpperCase()}`);
        } catch (error) {
            logger.error('Failed to export audit log:', error);
            this.toastManager?.error('Failed to export audit log');
        }
    }

    /**
     * Set loading state
     */
    private setLoading(loading: boolean): void {
        const tbody = document.getElementById('audit-events-body');
        if (!tbody) return;

        if (loading) {
            tbody.innerHTML = `
                <tr class="audit-loading-row">
                    <td colspan="8">
                        <div class="audit-loading">
                            <div class="audit-spinner"></div>
                            <span>Loading events...</span>
                        </div>
                    </td>
                </tr>
            `;
        }
    }

    /**
     * Show error message
     */
    private showError(message: string): void {
        const tbody = document.getElementById('audit-events-body');
        if (!tbody) return;

        tbody.innerHTML = `
            <tr class="audit-error-row">
                <td colspan="8">
                    <div class="audit-error">
                        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <circle cx="12" cy="12" r="10"/>
                            <line x1="12" y1="8" x2="12" y2="12"/>
                            <line x1="12" y1="16" x2="12.01" y2="16"/>
                        </svg>
                        <p>${message}</p>
                        <button class="audit-retry-btn" id="audit-retry-btn">Try Again</button>
                    </div>
                </td>
            </tr>
        `;

        const retryBtn = document.getElementById('audit-retry-btn');
        retryBtn?.addEventListener('click', () => this.loadEvents());
    }

    /**
     * Format number with locale
     */
    private formatNumber(num: number): string {
        return num.toLocaleString();
    }

    /**
     * Format date/time for display
     */
    private formatDateTime(date: Date): string {
        return date.toLocaleString(undefined, {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
        });
    }

    /**
     * Format time for table (short)
     */
    private formatTimeShort(date: Date): string {
        return date.toLocaleTimeString(undefined, {
            hour: '2-digit',
            minute: '2-digit',
        });
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

        const timeRanges = [
            { threshold: 1, format: () => 'Just now' },
            { threshold: 60, format: () => `${diffMins}m ago` },
            { threshold: 24 * 60, format: () => `${diffHours}h ago` },
            { threshold: 7 * 24 * 60, format: () => `${diffDays}d ago` },
        ];

        for (const range of timeRanges) {
            if (diffMins < range.threshold) {
                return range.format();
            }
        }

        return date.toLocaleDateString();
    }

    /**
     * Refresh the audit log
     */
    async refresh(): Promise<void> {
        await Promise.all([this.loadStats(), this.loadEvents()]);
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
        this.events = [];
        this.stats = null;
        this.selectedEvent = null;
        this.toastManager = null;
    }
}

// Export singleton factory
let auditLogViewerInstance: AuditLogViewer | null = null;

export function getAuditLogViewer(api: API): AuditLogViewer {
    if (!auditLogViewerInstance) {
        auditLogViewerInstance = new AuditLogViewer(api);
    }
    return auditLogViewerInstance;
}
