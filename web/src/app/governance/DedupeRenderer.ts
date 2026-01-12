// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * DedupeRenderer - Deduplication Audit Page
 *
 * Dedupe audit log with confirm/restore actions (ADR-0022).
 */

import type { API } from '../../lib/api';
import type { DedupeAuditEntry, DedupeAuditStats, DedupeAuditFilter } from '../../lib/types';
import { BaseRenderer, GovernanceConfig, REASON_NAMES, LAYER_NAMES, STATUS_CONFIG } from './BaseRenderer';
import { createLogger } from '../../lib/logger';

const logger = createLogger('DedupeRenderer');

export class DedupeRenderer extends BaseRenderer {
    private stats: DedupeAuditStats | null = null;
    private entries: DedupeAuditEntry[] = [];
    private filter: DedupeAuditFilter = {};

    constructor(api: API, config: GovernanceConfig) {
        super(api, config);
    }

    // =========================================================================
    // Data Getters
    // =========================================================================

    getStats(): DedupeAuditStats | null {
        return this.stats;
    }

    getEntries(): DedupeAuditEntry[] {
        return this.entries;
    }

    // =========================================================================
    // Public API
    // =========================================================================

    setupEventListeners(): void {
        // Status filter
        const statusFilter = document.getElementById('dedupe-filter-status') as HTMLSelectElement | null;
        statusFilter?.addEventListener('change', () => {
            this.filter.status = statusFilter.value as DedupeAuditFilter['status'] || undefined;
            this.loadEntries();
        });

        // Reason filter
        const reasonFilter = document.getElementById('dedupe-filter-reason') as HTMLSelectElement | null;
        reasonFilter?.addEventListener('change', () => {
            this.filter.reason = reasonFilter.value as DedupeAuditFilter['reason'] || undefined;
            this.loadEntries();
        });

        // Source filter
        const sourceFilter = document.getElementById('dedupe-filter-source') as HTMLSelectElement | null;
        sourceFilter?.addEventListener('change', () => {
            this.filter.source = sourceFilter.value || undefined;
            this.loadEntries();
        });

        // Refresh button
        document.getElementById('dedupe-refresh-btn')?.addEventListener('click', () => {
            this.loadStats();
            this.loadEntries();
        });

        // Export button
        document.getElementById('dedupe-export-btn')?.addEventListener('click', () => {
            this.exportToCSV();
        });
    }

    async load(): Promise<void> {
        await Promise.all([this.loadStats(), this.loadEntries()]);
    }

    // =========================================================================
    // Data Loading
    // =========================================================================

    async loadStats(): Promise<void> {
        try {
            this.stats = await this.api.getDedupeAuditStats();
            this.updateStatsDisplay();
        } catch (error) {
            logger.error('Failed to load stats:', error);
        }
    }

    async loadEntries(): Promise<void> {
        try {
            const response = await this.api.getDedupeAuditEntries({
                ...this.filter,
                limit: this.config.maxTableEntries,
            });
            this.entries = response.entries;
            this.renderTable();
        } catch (error) {
            logger.error('Failed to load entries:', error);
        }
    }

    // =========================================================================
    // Display Updates
    // =========================================================================

    private updateStatsDisplay(): void {
        if (!this.stats) return;

        this.setElementText('dedupe-total', this.stats.total_deduped.toLocaleString());
        this.setElementText('dedupe-pending', this.stats.pending_review.toLocaleString());
        this.setElementText('dedupe-confirmed', this.stats.user_confirmed.toLocaleString());
        this.setElementText('dedupe-restored', this.stats.user_restored.toLocaleString());
        this.setElementText('dedupe-accuracy', `${(this.stats.accuracy_rate * 100).toFixed(1)}%`);
    }

    private renderTable(): void {
        const tbody = document.getElementById('dedupe-audit-tbody');
        if (!tbody) return;

        if (this.entries.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="table-empty">No dedupe entries found</td></tr>';
            return;
        }

        tbody.innerHTML = this.entries.map(entry => {
            const statusConfig = STATUS_CONFIG[entry.status] || STATUS_CONFIG.auto_dedupe;
            return `
                <tr data-entry-id="${entry.id}">
                    <td>${this.formatTimestamp(entry.timestamp)}</td>
                    <td>${this.escapeHtml(entry.username || 'Unknown')}</td>
                    <td>${this.escapeHtml(entry.discarded_source)}</td>
                    <td>${REASON_NAMES[entry.dedupe_reason] || entry.dedupe_reason}</td>
                    <td>${LAYER_NAMES[entry.dedupe_layer] || entry.dedupe_layer}</td>
                    <td>
                        <span class="status-badge" style="background: ${statusConfig.color}">
                            ${statusConfig.icon} ${statusConfig.name}
                        </span>
                    </td>
                    <td class="actions-cell">
                        ${entry.status === 'auto_dedupe' ? `
                            <button class="btn-action btn-confirm" data-action="confirm" data-id="${entry.id}" title="Confirm as correct">
                                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <polyline points="20 6 9 17 4 12"/>
                                </svg>
                            </button>
                            <button class="btn-action btn-restore" data-action="restore" data-id="${entry.id}" title="Restore event">
                                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
                                    <path d="M3 3v5h5"/>
                                </svg>
                            </button>
                        ` : ''}
                    </td>
                </tr>
            `;
        }).join('');

        // Add action listeners
        tbody.querySelectorAll('[data-action]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const action = (e.currentTarget as HTMLElement).getAttribute('data-action');
                const id = (e.currentTarget as HTMLElement).getAttribute('data-id');
                if (action === 'confirm' && id) {
                    this.confirmEntry(id);
                } else if (action === 'restore' && id) {
                    this.restoreEntry(id);
                }
            });
        });
    }

    // =========================================================================
    // Actions
    // =========================================================================

    private async confirmEntry(id: string): Promise<void> {
        try {
            await this.api.confirmDedupeEntry(id, { resolved_by: 'user' });
            await this.loadEntries();
            await this.loadStats();
        } catch (error) {
            logger.error('Failed to confirm entry:', error);
        }
    }

    private async restoreEntry(id: string): Promise<void> {
        try {
            await this.api.restoreDedupeEntry(id, { resolved_by: 'user' });
            await this.loadEntries();
            await this.loadStats();
        } catch (error) {
            logger.error('Failed to restore entry:', error);
        }
    }

    private exportToCSV(): void {
        const url = this.api.getDedupeAuditExportUrl(this.filter);
        window.open(url, '_blank');
    }
}
