// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * AuditRenderer - Audit Log Page
 *
 * Security audit event log with filtering and export.
 */

import { createLogger } from '../../lib/logger';
import type { API } from '../../lib/api';
import type { AuditEvent, AuditStats, AuditEventFilter } from '../../lib/types';
import { BaseRenderer, GovernanceConfig, SEVERITY_COLORS } from './BaseRenderer';

const logger = createLogger('AuditRenderer');

export class AuditRenderer extends BaseRenderer {
    private events: AuditEvent[] = [];
    private stats: AuditStats | null = null;
    private filter: AuditEventFilter = { limit: 50 };
    private page = 0;

    constructor(api: API, config: GovernanceConfig) {
        super(api, config);
    }

    // =========================================================================
    // Public API
    // =========================================================================

    setupEventListeners(): void {
        document.getElementById('audit-refresh-btn')?.addEventListener('click', () => {
            this.load();
        });

        document.getElementById('audit-export-btn')?.addEventListener('click', () => {
            this.exportLog();
        });

        document.getElementById('audit-type-filter')?.addEventListener('change', (e) => {
            const value = (e.target as HTMLSelectElement).value;
            this.filter.types = value ? [value as any] : undefined;
            this.page = 0;
            this.load();
        });

        document.getElementById('audit-severity-filter')?.addEventListener('change', (e) => {
            const value = (e.target as HTMLSelectElement).value;
            this.filter.severities = value ? [value as any] : undefined;
            this.page = 0;
            this.load();
        });

        document.getElementById('audit-outcome-filter')?.addEventListener('change', (e) => {
            const value = (e.target as HTMLSelectElement).value;
            this.filter.outcomes = value ? [value as any] : undefined;
            this.page = 0;
            this.load();
        });

        document.getElementById('audit-search')?.addEventListener('input', this.debounce((e: Event) => {
            const value = (e.target as HTMLInputElement).value;
            this.filter.search = value || undefined;
            this.page = 0;
            this.load();
        }, 300));

        document.getElementById('audit-prev-btn')?.addEventListener('click', () => {
            if (this.page > 0) {
                this.page--;
                this.load();
            }
        });

        document.getElementById('audit-next-btn')?.addEventListener('click', () => {
            this.page++;
            this.load();
        });
    }

    async load(): Promise<void> {
        try {
            const [statsResult, eventsResult] = await Promise.all([
                this.api.getAuditStats().catch(() => null),
                this.api.getAuditEvents({
                    ...this.filter,
                    offset: this.page * (this.filter.limit || 50),
                }).catch(() => ({ events: [], total: 0, limit: 50, offset: 0 })),
            ]);

            this.stats = statsResult;
            this.events = eventsResult.events;

            // Store total for pagination
            const totalEl = document.getElementById('audit-total-count');
            if (totalEl) {
                totalEl.setAttribute('data-total', String(eventsResult.total));
            }

            this.updateDisplay();
        } catch (error) {
            logger.error('Failed to load audit data:', error);
        }
    }

    // =========================================================================
    // Display Updates
    // =========================================================================

    private updateDisplay(): void {
        // Update stats grid
        if (this.stats) {
            this.setElementText('audit-total-events', this.stats.total_events.toLocaleString());

            // Calculate events by category
            const authEvents = (this.stats.events_by_type['auth.success'] || 0) +
                (this.stats.events_by_type['auth.failure'] || 0) +
                (this.stats.events_by_type['auth.lockout'] || 0);
            const detectionEvents = (this.stats.events_by_type['detection.alert'] || 0) +
                (this.stats.events_by_type['detection.acknowledged'] || 0);
            const adminEvents = (this.stats.events_by_type['admin.action'] || 0) +
                (this.stats.events_by_type['user.created'] || 0) +
                (this.stats.events_by_type['user.modified'] || 0) +
                (this.stats.events_by_type['config.changed'] || 0);

            this.setElementText('audit-auth-events', authEvents.toLocaleString());
            this.setElementText('audit-detection-events', detectionEvents.toLocaleString());
            this.setElementText('audit-admin-events', adminEvents.toLocaleString());

            // Critical and warning counts
            const criticalCount = this.stats.events_by_severity?.critical || 0;
            const warningCount = this.stats.events_by_severity?.warning || 0;
            this.setElementText('audit-critical-count', criticalCount.toLocaleString());
            this.setElementText('audit-warning-count', warningCount.toLocaleString());
        }

        this.updatePagination();
        this.renderTable();
    }

    private updatePagination(): void {
        const totalEl = document.getElementById('audit-total-count');
        const total = parseInt(totalEl?.getAttribute('data-total') || '0', 10);
        const pageSize = this.filter.limit || 50;
        const totalPages = Math.ceil(total / pageSize);
        const currentPage = this.page + 1;

        this.setElementText('audit-page-info', `Page ${currentPage} of ${totalPages || 1}`);

        // Enable/disable pagination buttons
        const prevBtn = document.getElementById('audit-prev-btn') as HTMLButtonElement | null;
        const nextBtn = document.getElementById('audit-next-btn') as HTMLButtonElement | null;

        if (prevBtn) {
            prevBtn.disabled = this.page === 0;
        }
        if (nextBtn) {
            nextBtn.disabled = currentPage >= totalPages;
        }
    }

    private renderTable(): void {
        const tbody = document.getElementById('audit-events-tbody');
        if (!tbody) return;

        if (this.events.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" class="table-empty">No audit events found</td></tr>';
            return;
        }

        const outcomeIcons: Record<string, string> = {
            success: '\u2714',
            failure: '\u2718',
            unknown: '\u2753',
        };

        tbody.innerHTML = this.events.map(event => {
            const severityColor = SEVERITY_COLORS[event.severity] || '#666';
            const outcomeIcon = outcomeIcons[event.outcome] || '';

            return `
                <tr data-event-id="${event.id}" class="audit-row audit-${event.severity}">
                    <td class="audit-time">${this.formatTimestamp(event.timestamp)}</td>
                    <td class="audit-type">
                        <span class="audit-type-badge">${this.formatAuditType(event.type)}</span>
                    </td>
                    <td class="audit-actor">
                        ${event.actor ? `
                            <span class="actor-info">
                                <span class="actor-type">${event.actor.type}</span>
                                <span class="actor-id">${this.escapeHtml(event.actor.name || event.actor.id)}</span>
                            </span>
                        ` : '--'}
                    </td>
                    <td class="audit-target">
                        ${event.target ? `
                            <span class="target-info">
                                <span class="target-type">${event.target.type}</span>
                                <span class="target-id">${this.escapeHtml(event.target.name || event.target.id)}</span>
                            </span>
                        ` : '--'}
                    </td>
                    <td class="audit-severity">
                        <span class="severity-badge" style="background: ${severityColor}">${event.severity}</span>
                    </td>
                    <td class="audit-outcome">
                        <span class="outcome-badge outcome-${event.outcome}">${outcomeIcon} ${event.outcome}</span>
                    </td>
                    <td class="audit-source">
                        ${event.source?.ip_address || '--'}
                    </td>
                    <td class="audit-details">
                        <button class="btn-action btn-details" data-action="details" data-id="${event.id}" title="View Details">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <circle cx="12" cy="12" r="10"/>
                                <path d="M12 16v-4"/>
                                <path d="M12 8h.01"/>
                            </svg>
                        </button>
                    </td>
                </tr>
            `;
        }).join('');

        // Add details click handlers
        tbody.querySelectorAll('[data-action="details"]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const id = (e.currentTarget as HTMLElement).getAttribute('data-id');
                if (id) {
                    this.showEventDetails(id);
                }
            });
        });
    }

    private formatAuditType(type: string): string {
        return type.split('.').map(part =>
            part.charAt(0).toUpperCase() + part.slice(1)
        ).join(' ');
    }

    // =========================================================================
    // Event Details Dialog
    // =========================================================================

    private showEventDetails(eventId: string): void {
        const event = this.events.find(e => e.id === eventId);
        if (!event) return;

        // Create or get details dialog
        let dialog = document.getElementById('audit-details-dialog');
        if (!dialog) {
            dialog = document.createElement('div');
            dialog.id = 'audit-details-dialog';
            dialog.className = 'modal-overlay';
            document.body.appendChild(dialog);
        }

        dialog.innerHTML = `
            <div class="modal-content audit-details-modal">
                <div class="modal-header">
                    <h3>Audit Event Details</h3>
                    <button class="modal-close" aria-label="Close">&times;</button>
                </div>
                <div class="modal-body">
                    <div class="audit-detail-grid">
                        <div class="audit-detail-row">
                            <label>Event ID:</label>
                            <span class="audit-detail-value">${event.id}</span>
                        </div>
                        <div class="audit-detail-row">
                            <label>Timestamp:</label>
                            <span class="audit-detail-value">${this.formatTimestamp(event.timestamp)}</span>
                        </div>
                        <div class="audit-detail-row">
                            <label>Type:</label>
                            <span class="audit-detail-value">${this.formatAuditType(event.type)}</span>
                        </div>
                        <div class="audit-detail-row">
                            <label>Severity:</label>
                            <span class="audit-detail-value severity-${event.severity}">${event.severity}</span>
                        </div>
                        <div class="audit-detail-row">
                            <label>Outcome:</label>
                            <span class="audit-detail-value outcome-${event.outcome}">${event.outcome}</span>
                        </div>
                        ${event.description ? `
                            <div class="audit-detail-row full-width">
                                <label>Description:</label>
                                <span class="audit-detail-value">${this.escapeHtml(event.description)}</span>
                            </div>
                        ` : ''}
                        ${event.actor ? `
                            <div class="audit-detail-section">
                                <h4>Actor</h4>
                                <div class="audit-detail-row">
                                    <label>Type:</label>
                                    <span>${event.actor.type}</span>
                                </div>
                                <div class="audit-detail-row">
                                    <label>ID:</label>
                                    <span>${this.escapeHtml(event.actor.id)}</span>
                                </div>
                                ${event.actor.name ? `
                                    <div class="audit-detail-row">
                                        <label>Name:</label>
                                        <span>${this.escapeHtml(event.actor.name)}</span>
                                    </div>
                                ` : ''}
                            </div>
                        ` : ''}
                        ${event.target ? `
                            <div class="audit-detail-section">
                                <h4>Target</h4>
                                <div class="audit-detail-row">
                                    <label>Type:</label>
                                    <span>${event.target.type}</span>
                                </div>
                                <div class="audit-detail-row">
                                    <label>ID:</label>
                                    <span>${this.escapeHtml(event.target.id)}</span>
                                </div>
                                ${event.target.name ? `
                                    <div class="audit-detail-row">
                                        <label>Name:</label>
                                        <span>${this.escapeHtml(event.target.name)}</span>
                                    </div>
                                ` : ''}
                            </div>
                        ` : ''}
                        ${event.source ? `
                            <div class="audit-detail-section">
                                <h4>Source</h4>
                                <div class="audit-detail-row">
                                    <label>IP Address:</label>
                                    <span>${event.source.ip_address || '--'}</span>
                                </div>
                                ${event.source.user_agent ? `
                                    <div class="audit-detail-row full-width">
                                        <label>User Agent:</label>
                                        <span>${this.escapeHtml(event.source.user_agent)}</span>
                                    </div>
                                ` : ''}
                                ${event.source.geo ? `
                                    <div class="audit-detail-row">
                                        <label>Location:</label>
                                        <span>${this.escapeHtml([event.source.geo.city, event.source.geo.region, event.source.geo.country].filter(Boolean).join(', '))}</span>
                                    </div>
                                ` : ''}
                            </div>
                        ` : ''}
                        ${event.metadata ? `
                            <div class="audit-detail-section full-width">
                                <h4>Metadata</h4>
                                <pre class="audit-context-json">${this.escapeHtml(typeof event.metadata === 'string' ? event.metadata : JSON.stringify(event.metadata, null, 2))}</pre>
                            </div>
                        ` : ''}
                        ${event.correlation_id || event.request_id ? `
                            <div class="audit-detail-section">
                                <h4>Tracing</h4>
                                ${event.correlation_id ? `
                                    <div class="audit-detail-row">
                                        <label>Correlation ID:</label>
                                        <span class="monospace">${event.correlation_id}</span>
                                    </div>
                                ` : ''}
                                ${event.request_id ? `
                                    <div class="audit-detail-row">
                                        <label>Request ID:</label>
                                        <span class="monospace">${event.request_id}</span>
                                    </div>
                                ` : ''}
                            </div>
                        ` : ''}
                    </div>
                </div>
            </div>
        `;

        dialog.style.display = 'flex';

        // Close button handler
        const closeBtn = dialog.querySelector('.modal-close');
        closeBtn?.addEventListener('click', () => {
            dialog!.style.display = 'none';
        });

        // Click outside to close
        dialog.addEventListener('click', (e) => {
            if (e.target === dialog) {
                dialog!.style.display = 'none';
            }
        });
    }

    // =========================================================================
    // Export
    // =========================================================================

    private exportLog(): void {
        const formatValue = (document.getElementById('audit-export-format') as HTMLSelectElement)?.value || 'json';
        const format = formatValue === 'cef' ? 'cef' : 'json';
        const url = this.api.getAuditExportUrl(format, this.filter);
        window.open(url, '_blank');
    }
}
