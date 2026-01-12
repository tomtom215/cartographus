// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * DetectionRulesManager - Configure detection rules (ADR-0020)
 *
 * Features:
 * - View and configure detection rules
 * - Enable/disable individual rules
 * - Adjust rule thresholds and parameters
 * - View rule descriptions and metrics
 *
 * Reference: ADR-0020 Detection Rules Engine
 */

import type { API } from '../lib/api';
import { createLogger } from '../lib/logger';
import type {
  DetectionRule,
  DetectionRuleType,
  DetectionMetrics,
} from '../lib/types';

const logger = createLogger('DetectionRules');

/** Rule display metadata */
interface RuleMetadata {
  name: string;
  description: string;
  icon: string;
  configFields: ConfigField[];
}

/** Configuration field definition */
interface ConfigField {
  key: string;
  label: string;
  type: 'number' | 'text' | 'select' | 'countries';
  unit?: string;
  min?: number;
  max?: number;
  options?: { value: string; label: string }[];
}

/** Rule metadata definitions */
const RULE_METADATA: Record<DetectionRuleType, RuleMetadata> = {
  impossible_travel: {
    name: 'Impossible Travel Detection',
    description:
      'Detects when a user appears to be streaming from two geographically distant locations within a time frame that would be impossible to travel.',
    icon: '\u2708',
    configFields: [
      { key: 'max_speed_kmh', label: 'Max Travel Speed', type: 'number', unit: 'km/h', min: 100, max: 2000 },
      { key: 'min_distance_km', label: 'Min Distance', type: 'number', unit: 'km', min: 10, max: 500 },
      { key: 'min_time_delta_minutes', label: 'Min Time Between Events', type: 'number', unit: 'min', min: 1, max: 60 },
    ],
  },
  concurrent_streams: {
    name: 'Concurrent Stream Limits',
    description:
      'Monitors the number of simultaneous streams per user and alerts when the limit is exceeded.',
    icon: '\u23F5',
    configFields: [
      { key: 'default_limit', label: 'Default Stream Limit', type: 'number', min: 1, max: 20 },
    ],
  },
  device_velocity: {
    name: 'Device IP Velocity',
    description:
      'Detects devices that appear on many different IP addresses within a short time window, indicating potential credential sharing.',
    icon: '\u26A1',
    configFields: [
      { key: 'window_minutes', label: 'Time Window', type: 'number', unit: 'min', min: 5, max: 120 },
      { key: 'max_unique_ips', label: 'Max Unique IPs', type: 'number', min: 2, max: 20 },
    ],
  },
  geo_restriction: {
    name: 'Geographic Restrictions',
    description:
      'Block or allow access based on the geographic location of the streaming device.',
    icon: '\u1F6AB',
    configFields: [
      { key: 'blocked_countries', label: 'Blocked Countries', type: 'countries' },
      { key: 'allowed_countries', label: 'Allowed Countries', type: 'countries' },
    ],
  },
  simultaneous_locations: {
    name: 'Simultaneous Locations',
    description:
      'Detects when a user has active streams from multiple distant locations at the same time.',
    icon: '\u1F4CD',
    configFields: [
      { key: 'window_minutes', label: 'Time Window', type: 'number', unit: 'min', min: 5, max: 120 },
      { key: 'min_distance_km', label: 'Min Distance', type: 'number', unit: 'km', min: 10, max: 500 },
    ],
  },
  user_agent_anomaly: {
    name: 'User Agent Anomaly Detection',
    description:
      'Detects unusual user agent patterns including rapid platform switches, suspicious automation tools, and new device/client combinations.',
    icon: '\u1F4F1',
    configFields: [
      { key: 'window_minutes', label: 'Time Window', type: 'number', unit: 'min', min: 5, max: 120 },
      { key: 'min_history_for_anomaly', label: 'Min History', type: 'number', min: 1, max: 10 },
    ],
  },
  vpn_usage: {
    name: 'VPN Usage Detection',
    description:
      'Detects when users stream via known VPN providers, which may indicate location spoofing or policy violations.',
    icon: '\u1F510',
    configFields: [],
  },
};

export class DetectionRulesManager {
  private api: API;
  private rules: DetectionRule[] = [];
  private metrics: DetectionMetrics | null = null;
  private isLoading: boolean = false;
  private pendingChanges: Map<DetectionRuleType, Partial<DetectionRule>> = new Map();

  constructor(api: API) {
    this.api = api;
  }

  /**
   * Initialize the rules manager
   */
  async init(): Promise<void> {
    logger.debug('[ADR-0020] DetectionRulesManager initialized');
    this.setupEventListeners();
    await this.loadRules();
  }

  /**
   * Set up DOM event listeners
   */
  private setupEventListeners(): void {
    // Save button
    const saveBtn = document.getElementById('detection-rules-save');
    saveBtn?.addEventListener('click', () => this.saveChanges());

    // Reset button
    const resetBtn = document.getElementById('detection-rules-reset');
    resetBtn?.addEventListener('click', () => this.resetChanges());
  }

  /**
   * Load rules from the API
   */
  async loadRules(): Promise<void> {
    if (this.isLoading) return;

    this.isLoading = true;
    this.showLoading();

    try {
      const [rulesResponse, metrics] = await Promise.all([
        this.api.getDetectionRules(),
        this.api.getDetectionMetrics().catch(() => null),
      ]);

      this.rules = rulesResponse.rules || [];
      this.metrics = metrics;
      this.pendingChanges.clear();

      this.updateDisplay();
    } catch (error) {
      logger.error('[ADR-0020] Failed to load detection rules:', error);
      this.showError();
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * Show loading state
   */
  private showLoading(): void {
    const container = document.getElementById('detection-rules-content');
    if (container) {
      container.innerHTML = `
        <div class="detection-rules-loading">
          <div class="detection-rules-spinner"></div>
          <span>Loading detection rules...</span>
        </div>
      `;
    }
  }

  /**
   * Show error state
   */
  private showError(): void {
    const container = document.getElementById('detection-rules-content');
    if (container) {
      container.innerHTML = `
        <div class="detection-rules-empty">
          <span class="detection-rules-icon" aria-hidden="true">&#x26A0;</span>
          <span>Unable to load detection rules</span>
        </div>
      `;
    }
  }

  /**
   * Update the display with current rules
   */
  private updateDisplay(): void {
    const container = document.getElementById('detection-rules-content');
    if (!container) return;

    if (this.rules.length === 0) {
      container.innerHTML = `
        <div class="detection-rules-empty">
          <span class="detection-rules-icon" aria-hidden="true">&#x2139;</span>
          <span>No detection rules configured</span>
        </div>
      `;
      return;
    }

    const html = this.rules.map((rule) => this.renderRuleCard(rule)).join('');
    container.innerHTML = html;

    // Set up event listeners for rule cards
    this.setupRuleCardListeners();
    this.updateSaveButton();
  }

  /**
   * Render a single rule card
   */
  private renderRuleCard(rule: DetectionRule): string {
    const metadata = RULE_METADATA[rule.rule_type];
    if (!metadata) return '';

    const config = rule.config as Record<string, unknown>;
    const detectorMetrics = this.metrics?.detector_metrics[rule.rule_type];

    const configHtml = metadata.configFields
      .map((field) => this.renderConfigField(rule.rule_type, field, config[field.key]))
      .join('');

    const metricsHtml = detectorMetrics
      ? `
      <div class="detection-rule-metrics">
        <span class="detection-rule-metric">
          <span class="detection-rule-metric-value">${detectorMetrics.events_checked}</span>
          <span class="detection-rule-metric-label">Events Checked</span>
        </span>
        <span class="detection-rule-metric">
          <span class="detection-rule-metric-value">${detectorMetrics.alerts_generated}</span>
          <span class="detection-rule-metric-label">Alerts Generated</span>
        </span>
      </div>
    `
      : '';

    return `
      <div class="detection-rule-card ${rule.enabled ? '' : 'detection-rule-disabled'}"
           data-rule-type="${rule.rule_type}">
        <div class="detection-rule-header">
          <span class="detection-rule-icon" aria-hidden="true">${metadata.icon}</span>
          <div class="detection-rule-title-row">
            <span class="detection-rule-name">${this.escapeHtml(metadata.name)}</span>
            <label class="detection-rule-toggle">
              <input type="checkbox" ${rule.enabled ? 'checked' : ''}
                     class="detection-rule-enabled" data-rule-type="${rule.rule_type}">
              <span class="detection-rule-toggle-slider"></span>
            </label>
          </div>
        </div>
        <p class="detection-rule-description">${this.escapeHtml(metadata.description)}</p>
        <div class="detection-rule-config">
          ${configHtml}
        </div>
        ${metricsHtml}
      </div>
    `;
  }

  /**
   * Render a configuration field
   */
  private renderConfigField(
    ruleType: DetectionRuleType,
    field: ConfigField,
    value: unknown
  ): string {
    const fieldId = `${ruleType}-${field.key}`;
    const currentValue = value !== undefined ? String(value) : '';

    let inputHtml = '';

    switch (field.type) {
      case 'number':
        inputHtml = `
          <input type="number" id="${fieldId}" class="detection-config-input"
                 data-rule-type="${ruleType}" data-config-key="${field.key}"
                 value="${currentValue}"
                 ${field.min !== undefined ? `min="${field.min}"` : ''}
                 ${field.max !== undefined ? `max="${field.max}"` : ''}>
          ${field.unit ? `<span class="detection-config-unit">${field.unit}</span>` : ''}
        `;
        break;

      case 'text':
        inputHtml = `
          <input type="text" id="${fieldId}" class="detection-config-input"
                 data-rule-type="${ruleType}" data-config-key="${field.key}"
                 value="${this.escapeHtml(currentValue)}">
        `;
        break;

      case 'select':
        const options = field.options?.map(
          (opt) =>
            `<option value="${opt.value}" ${currentValue === opt.value ? 'selected' : ''}>${opt.label}</option>`
        ).join('') || '';
        inputHtml = `
          <select id="${fieldId}" class="detection-config-input"
                  data-rule-type="${ruleType}" data-config-key="${field.key}">
            ${options}
          </select>
        `;
        break;

      case 'countries':
        const countries = Array.isArray(value) ? value : [];
        inputHtml = `
          <input type="text" id="${fieldId}" class="detection-config-input"
                 data-rule-type="${ruleType}" data-config-key="${field.key}"
                 value="${this.escapeHtml(countries.join(', '))}"
                 placeholder="US, CA, GB">
          <span class="detection-config-hint">Comma-separated country codes</span>
        `;
        break;
    }

    return `
      <div class="detection-config-field">
        <label for="${fieldId}" class="detection-config-label">${field.label}</label>
        <div class="detection-config-input-wrapper">
          ${inputHtml}
        </div>
      </div>
    `;
  }

  /**
   * Set up event listeners for rule cards
   */
  private setupRuleCardListeners(): void {
    // Toggle listeners
    document.querySelectorAll('.detection-rule-enabled').forEach((checkbox) => {
      checkbox.addEventListener('change', (e) => {
        const target = e.target as HTMLInputElement;
        const ruleType = target.dataset.ruleType as DetectionRuleType;
        this.trackChange(ruleType, { enabled: target.checked });
      });
    });

    // Config input listeners
    document.querySelectorAll('.detection-config-input').forEach((input) => {
      input.addEventListener('change', (e) => {
        const target = e.target as HTMLInputElement;
        const ruleType = target.dataset.ruleType as DetectionRuleType;
        const configKey = target.dataset.configKey as string;

        let value: unknown;
        if (target.type === 'number') {
          value = parseFloat(target.value);
        } else if (configKey.includes('countries')) {
          value = target.value
            .split(',')
            .map((s) => s.trim().toUpperCase())
            .filter(Boolean);
        } else {
          value = target.value;
        }

        const rule = this.rules.find((r) => r.rule_type === ruleType);
        const currentConfig = (rule?.config as Record<string, unknown>) || {};
        const newConfig = { ...currentConfig, [configKey]: value };

        this.trackChange(ruleType, { config: newConfig });
      });
    });
  }

  /**
   * Track a pending change
   */
  private trackChange(ruleType: DetectionRuleType, change: Partial<DetectionRule>): void {
    const existing = this.pendingChanges.get(ruleType) || {};
    this.pendingChanges.set(ruleType, { ...existing, ...change });
    this.updateSaveButton();
  }

  /**
   * Update the save button state
   */
  private updateSaveButton(): void {
    const saveBtn = document.getElementById('detection-rules-save') as HTMLButtonElement;
    if (saveBtn) {
      saveBtn.disabled = this.pendingChanges.size === 0;
    }
  }

  /**
   * Save all pending changes
   */
  async saveChanges(): Promise<void> {
    if (this.pendingChanges.size === 0) return;

    const saveBtn = document.getElementById('detection-rules-save') as HTMLButtonElement;
    if (saveBtn) {
      saveBtn.disabled = true;
      saveBtn.textContent = 'Saving...';
    }

    try {
      for (const [ruleType, changes] of this.pendingChanges) {
        if (changes.enabled !== undefined) {
          await this.api.setDetectionRuleEnabled(ruleType, changes.enabled);
        }
        if (changes.config) {
          await this.api.updateDetectionRule(ruleType, {
            enabled: changes.enabled ?? this.rules.find((r) => r.rule_type === ruleType)?.enabled ?? true,
            config: changes.config as Record<string, unknown>,
          });
        }
      }

      this.pendingChanges.clear();

      // Reload rules to get updated state
      await this.loadRules();

      // Show success notification
      window.dispatchEvent(
        new CustomEvent('notification:add', {
          detail: {
            type: 'success',
            title: 'Rules Saved',
            message: 'Detection rules have been updated.',
          },
        })
      );
    } catch (error) {
      logger.error('[ADR-0020] Failed to save rules:', error);
      window.dispatchEvent(
        new CustomEvent('notification:add', {
          detail: {
            type: 'error',
            title: 'Error',
            message: 'Failed to save detection rules. Please try again.',
          },
        })
      );
    } finally {
      if (saveBtn) {
        saveBtn.textContent = 'Save Changes';
        this.updateSaveButton();
      }
    }
  }

  /**
   * Reset pending changes
   */
  resetChanges(): void {
    this.pendingChanges.clear();
    this.updateDisplay();
  }

  /**
   * Escape HTML
   */
  private escapeHtml(str: string): string {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  /**
   * Get current rules
   */
  getRules(): DetectionRule[] {
    return this.rules;
  }

  /**
   * Cleanup resources to prevent memory leaks
   * Clears internal state and pending changes
   */
  destroy(): void {
    this.rules = [];
    this.metrics = null;
    this.pendingChanges.clear();
  }
}

export default DetectionRulesManager;
