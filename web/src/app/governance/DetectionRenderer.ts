// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * DetectionRenderer - Detection Alerts Page
 *
 * Security alerts and detection rules management (ADR-0020).
 */

import type { API } from '../../lib/api';
import type { DetectionAlert, DetectionAlertStats, DetectionRule, DetectionRuleType } from '../../lib/types';
import { BaseRenderer, GovernanceConfig, SEVERITY_COLORS } from './BaseRenderer';
import { createLogger } from '../../lib/logger';

const logger = createLogger('DetectionRenderer');

export class DetectionRenderer extends BaseRenderer {
    private stats: DetectionAlertStats | null = null;
    private alerts: DetectionAlert[] = [];
    private rules: DetectionRule[] = [];

    constructor(api: API, config: GovernanceConfig) {
        super(api, config);
    }

    // =========================================================================
    // Data Getters
    // =========================================================================

    getStats(): DetectionAlertStats | null {
        return this.stats;
    }

    getAlerts(): DetectionAlert[] {
        return this.alerts;
    }

    // =========================================================================
    // Public API
    // =========================================================================

    setupEventListeners(): void {
        document.getElementById('detection-refresh-btn')?.addEventListener('click', () => {
            this.load();
        });
    }

    async load(): Promise<void> {
        try {
            const [stats, alertsResponse, rulesResponse] = await Promise.all([
                this.api.getDetectionAlertStats(),
                this.api.getDetectionAlerts({ limit: this.config.maxTableEntries }),
                this.api.getDetectionRules(),
            ]);

            this.stats = stats;
            this.alerts = alertsResponse.alerts;
            this.rules = rulesResponse.rules;

            this.updateDisplay();
        } catch (error) {
            logger.error('Failed to load detection data:', error);
        }
    }

    // =========================================================================
    // Display Updates
    // =========================================================================

    private updateDisplay(): void {
        if (this.stats) {
            this.setElementText('detection-critical', this.stats.by_severity.critical.toString());
            this.setElementText('detection-high', (this.stats.by_severity.warning || 0).toString());
            this.setElementText('detection-medium', (this.stats.by_severity.info || 0).toString());
            this.setElementText('detection-low', '0');
        }

        this.renderAlertsTable();
        this.renderRulesGrid();
    }

    private renderAlertsTable(): void {
        const tbody = document.getElementById('detection-alerts-tbody');
        if (!tbody) return;

        if (this.alerts.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="table-empty">No security alerts</td></tr>';
            return;
        }

        tbody.innerHTML = this.alerts.map(alert => `
            <tr data-alert-id="${alert.id}">
                <td>${this.formatTimestamp(alert.created_at)}</td>
                <td>${this.formatRuleType(alert.rule_type)}</td>
                <td>${this.escapeHtml(alert.username)}</td>
                <td>
                    <span class="severity-badge" style="background: ${SEVERITY_COLORS[alert.severity] || '#666'}">
                        ${alert.severity}
                    </span>
                </td>
                <td>${this.escapeHtml(alert.message).slice(0, 50)}...</td>
                <td>${alert.acknowledged ? 'Acknowledged' : 'Active'}</td>
                <td class="actions-cell">
                    ${!alert.acknowledged ? `
                        <button class="btn-action btn-acknowledge" data-action="acknowledge" data-id="${alert.id}" title="Acknowledge">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <polyline points="20 6 9 17 4 12"/>
                            </svg>
                        </button>
                    ` : ''}
                </td>
            </tr>
        `).join('');

        // Add action listeners
        tbody.querySelectorAll('[data-action="acknowledge"]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const id = (e.currentTarget as HTMLElement).getAttribute('data-id');
                if (id) {
                    this.acknowledgeAlert(parseInt(id, 10));
                }
            });
        });
    }

    private renderRulesGrid(): void {
        const grid = document.getElementById('detection-rules-grid');
        if (!grid) return;

        if (this.rules.length === 0) {
            grid.innerHTML = '<div class="rules-empty">No detection rules configured</div>';
            return;
        }

        grid.innerHTML = this.rules.map(rule => `
            <div class="rule-card ${rule.enabled ? 'rule-enabled' : 'rule-disabled'}">
                <div class="rule-header">
                    <span class="rule-name">${this.escapeHtml(rule.name)}</span>
                    <label class="rule-toggle">
                        <input type="checkbox" ${rule.enabled ? 'checked' : ''} data-rule-type="${rule.rule_type}">
                        <span class="toggle-slider"></span>
                    </label>
                </div>
                <div class="rule-type">${this.formatRuleType(rule.rule_type)}</div>
            </div>
        `).join('');

        // Add toggle listeners
        grid.querySelectorAll('input[type="checkbox"]').forEach(checkbox => {
            checkbox.addEventListener('change', (e) => {
                const ruleType = (e.target as HTMLInputElement).getAttribute('data-rule-type') as DetectionRuleType | null;
                const enabled = (e.target as HTMLInputElement).checked;
                if (ruleType) {
                    this.toggleRule(ruleType, enabled);
                }
            });
        });
    }

    // =========================================================================
    // Actions
    // =========================================================================

    private async acknowledgeAlert(id: number): Promise<void> {
        try {
            await this.api.acknowledgeDetectionAlert(id, 'user');
            await this.load();
        } catch (error) {
            logger.error('Failed to acknowledge alert:', error);
        }
    }

    private async toggleRule(ruleType: DetectionRuleType, enabled: boolean): Promise<void> {
        try {
            await this.api.setDetectionRuleEnabled(ruleType, enabled);
            await this.load();
        } catch (error) {
            logger.error('Failed to toggle rule:', error);
        }
    }
}
