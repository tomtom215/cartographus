// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * SecurityAlertsManager - Displays and manages detection alerts (ADR-0020)
 *
 * Features:
 * - Displays security alerts from the detection engine
 * - Shows impossible travel, concurrent streams, device velocity alerts
 * - Allows acknowledging alerts
 * - Real-time updates via WebSocket
 * - Displays alert statistics and metrics
 *
 * Reference: ADR-0020 Detection Rules Engine
 */

import type { API } from '../lib/api';
import type {
  DetectionAlert,
  DetectionAlertFilter,
  DetectionAlertStats,
  DetectionRuleType,
  DetectionSeverity,
} from '../lib/types';
import { createLogger } from '../lib/logger';
import { getRoleGuard } from '../lib/auth/RoleGuard';

const logger = createLogger('SecurityAlerts');

/** Alert display configuration */
interface AlertDisplayConfig {
  maxAlerts: number;
  autoRefreshMs: number;
  showAcknowledged: boolean;
}

const DEFAULT_CONFIG: AlertDisplayConfig = {
  maxAlerts: 20,
  autoRefreshMs: 30000, // 30 seconds
  showAcknowledged: false,
};

/** Rule type display names */
const RULE_TYPE_NAMES: Record<DetectionRuleType, string> = {
  impossible_travel: 'Impossible Travel',
  concurrent_streams: 'Concurrent Streams',
  device_velocity: 'Device IP Velocity',
  geo_restriction: 'Geographic Restriction',
  simultaneous_locations: 'Simultaneous Locations',
  user_agent_anomaly: 'User Agent Anomaly',
  vpn_usage: 'VPN Usage',
};

/** Rule type icons (Unicode) */
const RULE_TYPE_ICONS: Record<DetectionRuleType, string> = {
  impossible_travel: '\u2708', // airplane
  concurrent_streams: '\u23F5', // multiple streams
  device_velocity: '\u26A1', // lightning
  geo_restriction: '\u1F6AB', // prohibited
  simultaneous_locations: '\u1F4CD', // location pin
  user_agent_anomaly: '\u1F4F1', // mobile phone (device/agent)
  vpn_usage: '\u1F510', // lock with key (VPN)
};

/** Severity colors */
const SEVERITY_COLORS: Record<DetectionSeverity, string> = {
  critical: '#e74c3c',
  warning: '#f39c12',
  info: '#3498db',
};

export class SecurityAlertsManager {
  private api: API;
  private config: AlertDisplayConfig;
  private alerts: DetectionAlert[] = [];
  private stats: DetectionAlertStats | null = null;
  private isLoading: boolean = false;
  private refreshInterval: ReturnType<typeof setInterval> | null = null;
  private currentUsername: string | null = null;
  // AbortController for clean event listener removal (Task 22 - Fix Listener Leak)
  private abortController: AbortController | null = null;
  // Bound handler for event delegation on acknowledge buttons
  private boundAcknowledgeHandler: ((e: Event) => void) | null = null;

  constructor(api: API, config: Partial<AlertDisplayConfig> = {}) {
    this.api = api;
    this.config = { ...DEFAULT_CONFIG, ...config };
    this.currentUsername = api.getUsername();
    // Bind the acknowledge handler once for reuse
    this.boundAcknowledgeHandler = this.handleAcknowledgeClick.bind(this);
  }

  /**
   * Handle acknowledge button clicks via event delegation
   */
  private handleAcknowledgeClick(e: Event): void {
    const target = e.target as HTMLElement;
    if (target.classList.contains('security-alert-acknowledge')) {
      const alertId = parseInt(target.dataset.alertId || '0', 10);
      if (alertId) {
        this.acknowledgeAlert(alertId);
      }
    }
  }

  /**
   * Initialize the security alerts manager
   * RBAC: Viewers can see alerts, only admins can acknowledge
   */
  init(): void {
    logger.debug('SecurityAlertsManager initialized');

    // RBAC Phase 4: Check read permission for detection alerts
    const roleGuard = getRoleGuard();
    if (!roleGuard.canAccess('detection', 'read')) {
      logger.warn('[RBAC] User lacks permission to view security alerts');
      return;
    }

    this.setupEventListeners();
    this.applyRoleBasedVisibility();
    this.loadAlerts();
    this.startAutoRefresh();
  }

  /**
   * Apply role-based visibility to security alerts UI elements.
   * RBAC Phase 4: Frontend Role Integration
   *
   * - Viewers can see alerts but cannot acknowledge them
   * - Admins can acknowledge and manage alerts
   */
  private applyRoleBasedVisibility(): void {
    const roleGuard = getRoleGuard();

    // Hide acknowledge buttons for non-admin users
    if (!roleGuard.canAccess('detection', 'write')) {
      // Add CSS class to hide acknowledge buttons
      const alertsContainer = document.getElementById('security-alerts-list');
      alertsContainer?.classList.add('rbac-read-only');
    }
  }

  /**
   * Set up DOM event listeners
   * Uses AbortController for clean event listener removal (Task 22)
   */
  private setupEventListeners(): void {
    // Create AbortController for cleanup
    this.abortController = new AbortController();
    const signal = this.abortController.signal;

    // Listen for WebSocket detection alerts
    window.addEventListener(
      'ws:detection_alert',
      ((event: CustomEvent) => {
        this.handleNewAlert(event.detail);
      }) as EventListener,
      { signal }
    );

    // Listen for filter toggle
    const toggleBtn = document.getElementById('security-alerts-toggle-acknowledged');
    toggleBtn?.addEventListener(
      'click',
      () => {
        this.config.showAcknowledged = !this.config.showAcknowledged;
        toggleBtn.classList.toggle('active', this.config.showAcknowledged);
        this.loadAlerts();
      },
      { signal }
    );

    // Listen for refresh button
    const refreshBtn = document.getElementById('security-alerts-refresh');
    refreshBtn?.addEventListener('click', () => this.loadAlerts(), { signal });
  }

  /**
   * Start auto-refresh interval
   */
  private startAutoRefresh(): void {
    if (this.refreshInterval) {
      clearInterval(this.refreshInterval);
    }
    this.refreshInterval = setInterval(() => {
      this.loadAlerts();
    }, this.config.autoRefreshMs);
  }

  /**
   * Stop auto-refresh and clean up event listeners
   * Fixes memory leak by properly removing all event listeners (Task 22)
   */
  destroy(): void {
    // Clear the refresh interval
    if (this.refreshInterval) {
      clearInterval(this.refreshInterval);
      this.refreshInterval = null;
    }

    // Abort all event listeners registered with the AbortController
    if (this.abortController) {
      this.abortController.abort();
      this.abortController = null;
    }

    // Remove the acknowledge button handler
    const container = document.getElementById('security-alerts-content');
    if (container && this.boundAcknowledgeHandler) {
      container.removeEventListener('click', this.boundAcknowledgeHandler);
      container.innerHTML = '';
    }

    // Clear the handler reference
    this.boundAcknowledgeHandler = null;

    logger.debug('SecurityAlertsManager destroyed');
  }

  /**
   * Load alerts from the API
   */
  async loadAlerts(): Promise<void> {
    if (this.isLoading) return;

    this.isLoading = true;
    this.showLoading();

    try {
      const filter: DetectionAlertFilter = {
        limit: this.config.maxAlerts,
        order_by: 'created_at',
        order_direction: 'desc',
      };

      if (!this.config.showAcknowledged) {
        filter.acknowledged = false;
      }

      const [alertsResponse, stats] = await Promise.all([
        this.api.getDetectionAlerts(filter),
        this.api.getDetectionAlertStats().catch(() => null),
      ]);

      this.alerts = alertsResponse.alerts || [];
      this.stats = stats;

      this.updateDisplay();
    } catch (error) {
      logger.error('Failed to load alerts:', error);
      this.showError();
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * Handle a new alert from WebSocket
   */
  private handleNewAlert(alert: DetectionAlert): void {
    // Add to the beginning of the list
    this.alerts.unshift(alert);

    // Trim to max alerts
    if (this.alerts.length > this.config.maxAlerts) {
      this.alerts = this.alerts.slice(0, this.config.maxAlerts);
    }

    // Update display
    this.updateDisplay();

    // Show notification
    this.showAlertNotification(alert);
  }

  /**
   * Show a browser notification for new alert
   */
  private showAlertNotification(alert: DetectionAlert): void {
    // Dispatch custom event for notification center
    window.dispatchEvent(
      new CustomEvent('notification:add', {
        detail: {
          type: alert.severity === 'critical' ? 'error' : 'warning',
          title: alert.title,
          message: alert.message,
          persistent: alert.severity === 'critical',
        },
      })
    );
  }

  /**
   * Acknowledge an alert
   */
  async acknowledgeAlert(alertId: number): Promise<void> {
    try {
      const username = this.currentUsername || 'admin';
      await this.api.acknowledgeDetectionAlert(alertId, username);

      // Update local state
      const alert = this.alerts.find((a) => a.id === alertId);
      if (alert) {
        alert.acknowledged = true;
        alert.acknowledged_by = username;
        alert.acknowledged_at = new Date().toISOString();
      }

      // Refresh display
      if (!this.config.showAcknowledged) {
        this.alerts = this.alerts.filter((a) => a.id !== alertId);
      }
      this.updateDisplay();

      // Show success notification
      window.dispatchEvent(
        new CustomEvent('notification:add', {
          detail: {
            type: 'success',
            title: 'Alert Acknowledged',
            message: 'The security alert has been acknowledged.',
          },
        })
      );
    } catch (error) {
      logger.error('Failed to acknowledge alert:', error);
      window.dispatchEvent(
        new CustomEvent('notification:add', {
          detail: {
            type: 'error',
            title: 'Error',
            message: 'Failed to acknowledge alert. Please try again.',
          },
        })
      );
    }
  }

  /**
   * Show loading state
   */
  private showLoading(): void {
    const container = document.getElementById('security-alerts-content');
    if (container) {
      container.innerHTML = `
        <div class="security-alerts-loading">
          <div class="security-alerts-spinner"></div>
          <span>Loading security alerts...</span>
        </div>
      `;
    }
  }

  /**
   * Show error state
   */
  private showError(): void {
    const container = document.getElementById('security-alerts-content');
    if (container) {
      container.innerHTML = `
        <div class="security-alerts-empty">
          <span class="security-alerts-icon" aria-hidden="true">&#x26A0;</span>
          <span>Unable to load security alerts</span>
        </div>
      `;
    }
  }

  /**
   * Update the display with current alerts
   */
  private updateDisplay(): void {
    this.updateAlertsPanel();
    this.updateStatsPanel();
    this.updateBadge();
  }

  /**
   * Update the alerts panel
   */
  private updateAlertsPanel(): void {
    const container = document.getElementById('security-alerts-content');
    if (!container) return;

    if (this.alerts.length === 0) {
      container.innerHTML = `
        <div class="security-alerts-empty">
          <span class="security-alerts-icon" aria-hidden="true">&#x2714;</span>
          <span>No security alerts</span>
        </div>
      `;
      return;
    }

    const html = this.alerts.map((alert) => this.renderAlertCard(alert)).join('');
    container.innerHTML = html;

    // Use event delegation with the bound handler (set up once in init)
    // The handler is bound in constructor and added once here
    if (this.boundAcknowledgeHandler) {
      // Remove and re-add to ensure single listener
      container.removeEventListener('click', this.boundAcknowledgeHandler);
      container.addEventListener('click', this.boundAcknowledgeHandler);
    }
  }

  /**
   * Render a single alert card
   */
  private renderAlertCard(alert: DetectionAlert): string {
    const icon = RULE_TYPE_ICONS[alert.rule_type] || '\u26A0';
    const ruleName = RULE_TYPE_NAMES[alert.rule_type] || alert.rule_type;
    const severityColor = SEVERITY_COLORS[alert.severity];
    const timeAgo = this.formatTimeAgo(alert.created_at);

    const acknowledgeButton = alert.acknowledged
      ? `<span class="security-alert-acked">Acknowledged by ${alert.acknowledged_by}</span>`
      : `<button class="security-alert-acknowledge" data-alert-id="${alert.id}"
           title="Acknowledge this alert">Acknowledge</button>`;

    return `
      <div class="security-alert-card security-alert-${alert.severity}"
           role="article" aria-label="${alert.title}">
        <div class="security-alert-header">
          <span class="security-alert-icon" style="color: ${severityColor}" aria-hidden="true">${icon}</span>
          <div class="security-alert-title-row">
            <span class="security-alert-title">${this.escapeHtml(alert.title)}</span>
            <span class="security-alert-severity security-alert-severity-${alert.severity}">${alert.severity}</span>
          </div>
          <span class="security-alert-time" title="${alert.created_at}">${timeAgo}</span>
        </div>
        <div class="security-alert-body">
          <p class="security-alert-message">${this.escapeHtml(alert.message)}</p>
          <div class="security-alert-meta">
            <span class="security-alert-rule">${ruleName}</span>
            <span class="security-alert-user">User: ${this.escapeHtml(alert.username)}</span>
            ${alert.ip_address ? `<span class="security-alert-ip">IP: ${this.escapeHtml(alert.ip_address)}</span>` : ''}
          </div>
        </div>
        <div class="security-alert-footer">
          ${acknowledgeButton}
        </div>
      </div>
    `;
  }

  /**
   * Update the stats panel
   */
  private updateStatsPanel(): void {
    const container = document.getElementById('security-alerts-stats');
    if (!container || !this.stats) return;

    container.innerHTML = `
      <div class="security-stats-grid">
        <div class="security-stat security-stat-critical">
          <span class="security-stat-value">${this.stats.by_severity.critical}</span>
          <span class="security-stat-label">Critical</span>
        </div>
        <div class="security-stat security-stat-warning">
          <span class="security-stat-value">${this.stats.by_severity.warning}</span>
          <span class="security-stat-label">Warning</span>
        </div>
        <div class="security-stat security-stat-info">
          <span class="security-stat-value">${this.stats.by_severity.info}</span>
          <span class="security-stat-label">Info</span>
        </div>
        <div class="security-stat security-stat-unacked">
          <span class="security-stat-value">${this.stats.unacknowledged}</span>
          <span class="security-stat-label">Unacknowledged</span>
        </div>
      </div>
    `;
  }

  /**
   * Update the badge count
   */
  private updateBadge(): void {
    const badge = document.getElementById('security-alerts-badge');
    if (badge) {
      const count = this.stats?.unacknowledged || this.alerts.filter((a) => !a.acknowledged).length;
      badge.textContent = count > 0 ? count.toString() : '';
      badge.classList.toggle('security-alerts-badge-visible', count > 0);
    }
  }

  /**
   * Format time ago string
   */
  private formatTimeAgo(dateStr: string): string {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;

    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
  }

  /**
   * Escape HTML to prevent XSS
   */
  private escapeHtml(str: string): string {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  /**
   * Get current alerts
   */
  getAlerts(): DetectionAlert[] {
    return this.alerts;
  }

  /**
   * Get alert statistics
   */
  getStats(): DetectionAlertStats | null {
    return this.stats;
  }

  /**
   * Check if there are critical alerts
   */
  hasCriticalAlerts(): boolean {
    return this.alerts.some((a) => a.severity === 'critical' && !a.acknowledged);
  }
}

export default SecurityAlertsManager;
