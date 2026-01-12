// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * LineageRenderer - Data Lineage Page
 *
 * Event tracing and data lineage visualization.
 */

import { createLogger } from '../../lib/logger';
import type { API } from '../../lib/api';
import type { DedupeAuditEntry } from '../../lib/types';
import { BaseRenderer, GovernanceConfig, REASON_NAMES, STATUS_CONFIG } from './BaseRenderer';

const logger = createLogger('LineageRenderer');

export class LineageRenderer extends BaseRenderer {
    private dedupeEntries: DedupeAuditEntry[] = [];

    constructor(api: API, config: GovernanceConfig) {
        super(api, config);
    }

    // =========================================================================
    // Data Setters
    // =========================================================================

    setDedupeEntries(entries: DedupeAuditEntry[]): void {
        this.dedupeEntries = entries;
    }

    // =========================================================================
    // Public API
    // =========================================================================

    setupEventListeners(): void {
        document.getElementById('lineage-lookup-btn')?.addEventListener('click', () => {
            const input = document.getElementById('lineage-event-id') as HTMLInputElement;
            if (input?.value) {
                this.traceEvent(input.value);
            }
        });
    }

    async load(): Promise<void> {
        const modificationsEl = document.getElementById('lineage-modifications');
        if (modificationsEl) {
            // Load recent modifications from dedupe entries
            if (this.dedupeEntries.length === 0) {
                try {
                    const response = await this.api.getDedupeAuditEntries({ limit: 50 });
                    this.dedupeEntries = response.entries;
                } catch (error) {
                    logger.error('Failed to load dedupe entries:', error);
                }
            }

            const restoredEntries = this.dedupeEntries.filter(e => e.status === 'user_restored');
            if (restoredEntries.length === 0) {
                modificationsEl.innerHTML = '<div class="modifications-empty">No recent modifications</div>';
            } else {
                modificationsEl.innerHTML = restoredEntries.slice(0, 10).map(entry => `
                    <div class="modification-item">
                        <div class="modification-time">${this.formatTimestamp(entry.resolved_at || entry.timestamp)}</div>
                        <div class="modification-action">Event restored by ${entry.resolved_by || 'user'}</div>
                        <div class="modification-details">Event ID: ${entry.discarded_event_id}</div>
                    </div>
                `).join('');
            }
        }
    }

    // =========================================================================
    // Event Tracing
    // =========================================================================

    async traceEvent(eventId: string): Promise<void> {
        const resultsEl = document.getElementById('lineage-results');
        const eventInfoEl = document.getElementById('lineage-event-info');
        const timelineEl = document.getElementById('lineage-timeline');

        if (!resultsEl || !eventInfoEl || !timelineEl) return;

        resultsEl.style.display = 'block';
        eventInfoEl.innerHTML = '<div class="lineage-loading">Tracing event...</div>';

        try {
            // Try to find the event in dedupe audit
            const dedupeEntry = this.dedupeEntries.find(e =>
                e.discarded_event_id === eventId || e.matched_event_id === eventId
            );

            if (dedupeEntry) {
                eventInfoEl.innerHTML = `
                    <div class="lineage-event-card">
                        <h4>Event Found in Dedupe Audit</h4>
                        <div class="lineage-field"><strong>Event ID:</strong> ${eventId}</div>
                        <div class="lineage-field"><strong>User:</strong> ${this.escapeHtml(dedupeEntry.username || 'Unknown')}</div>
                        <div class="lineage-field"><strong>Source:</strong> ${dedupeEntry.discarded_source}</div>
                        <div class="lineage-field"><strong>Status:</strong> ${STATUS_CONFIG[dedupeEntry.status]?.name || dedupeEntry.status}</div>
                    </div>
                `;

                timelineEl.innerHTML = `
                    <div class="lineage-timeline-item">
                        <div class="timeline-marker"></div>
                        <div class="timeline-content">
                            <div class="timeline-title">Event Created</div>
                            <div class="timeline-time">${this.formatTimestamp(dedupeEntry.discarded_started_at || dedupeEntry.timestamp)}</div>
                        </div>
                    </div>
                    <div class="lineage-timeline-item">
                        <div class="timeline-marker"></div>
                        <div class="timeline-content">
                            <div class="timeline-title">Deduplicated (${REASON_NAMES[dedupeEntry.dedupe_reason]})</div>
                            <div class="timeline-time">${this.formatTimestamp(dedupeEntry.timestamp)}</div>
                        </div>
                    </div>
                    ${dedupeEntry.resolved_at ? `
                        <div class="lineage-timeline-item">
                            <div class="timeline-marker"></div>
                            <div class="timeline-content">
                                <div class="timeline-title">${dedupeEntry.status === 'user_restored' ? 'Restored' : 'Confirmed'}</div>
                                <div class="timeline-time">${this.formatTimestamp(dedupeEntry.resolved_at)}</div>
                            </div>
                        </div>
                    ` : ''}
                `;
            } else {
                eventInfoEl.innerHTML = `
                    <div class="lineage-not-found">
                        <p>Event not found in audit trail.</p>
                        <p>The event may be active in the database or was processed without deduplication.</p>
                    </div>
                `;
                timelineEl.innerHTML = '';
            }
        } catch (error) {
            logger.error('Failed to trace event:', error);
            eventInfoEl.innerHTML = '<div class="lineage-error">Failed to trace event</div>';
        }
    }
}
