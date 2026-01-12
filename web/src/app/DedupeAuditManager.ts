// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * DedupeAuditManager - Displays and manages deduplication audit entries (ADR-0022)
 *
 * Features:
 * - Displays deduplication audit log entries
 * - Shows statistics dashboard with dedupe metrics
 * - Allows confirming correct deduplication decisions
 * - Allows restoring incorrectly deduplicated events
 * - Exports audit log to CSV
 *
 * Reference: ADR-0022 Deduplication Audit and Management
 */

import type { API } from '../lib/api';
import type {
  DedupeAuditEntry,
  DedupeAuditStats,
  DedupeReason,
  DedupeLayer,
  DedupeStatus,
  DedupeAuditFilter,
} from '../lib/types';
import { createLogger } from '../lib/logger';

const logger = createLogger('DedupeAuditManager');

/** Display configuration */
interface DedupeAuditConfig {
  maxEntries: number;
  autoRefreshMs: number;
  showRestored: boolean;
  showConfirmed: boolean;
}

const DEFAULT_CONFIG: DedupeAuditConfig = {
  maxEntries: 50,
  autoRefreshMs: 60000, // 1 minute
  showRestored: false,
  showConfirmed: false,
};

/** Reason display names */
const REASON_NAMES: Record<DedupeReason, string> = {
  event_id: 'Event ID Match',
  session_key: 'Session Key Match',
  correlation_key: 'Correlation Key Match',
  cross_source_key: 'Cross-Source Match',
  db_constraint: 'Database Constraint',
};

/** Layer display names */
const LAYER_NAMES: Record<DedupeLayer, string> = {
  bloom_cache: 'BloomLRU Cache',
  nats_dedup: 'NATS JetStream',
  db_unique: 'DuckDB Index',
};

/** Status display names and colors */
const STATUS_CONFIG: Record<DedupeStatus, { name: string; color: string; icon: string }> = {
  auto_dedupe: { name: 'Pending Review', color: '#f39c12', icon: '\u26A0' },
  user_confirmed: { name: 'Confirmed', color: '#27ae60', icon: '\u2714' },
  user_restored: { name: 'Restored', color: '#3498db', icon: '\u21A9' },
};

export class DedupeAuditManager {
  private api: API;
  private config: DedupeAuditConfig;
  private entries: DedupeAuditEntry[] = [];
  private stats: DedupeAuditStats | null = null;
  private isLoading: boolean = false;
  private refreshInterval: ReturnType<typeof setInterval> | null = null;
  private currentFilter: DedupeAuditFilter = {};
  private totalCount: number = 0;

  constructor(api: API, config: Partial<DedupeAuditConfig> = {}) {
    this.api = api;
    this.config = { ...DEFAULT_CONFIG, ...config };
  }

  /**
   * Initialize the dedupe audit manager
   */
  init(): void {
    logger.debug('DedupeAuditManager initialized');
    this.setupEventListeners();
    this.loadStats();
    this.loadEntries();
    this.startAutoRefresh();
  }

  /**
   * Set up DOM event listeners
   */
  private setupEventListeners(): void {
    // Filter controls
    const statusFilter = document.getElementById('dedupe-filter-status') as HTMLSelectElement | null;
    statusFilter?.addEventListener('change', () => {
      this.currentFilter.status = statusFilter.value as DedupeStatus || undefined;
      this.loadEntries();
    });

    const reasonFilter = document.getElementById('dedupe-filter-reason') as HTMLSelectElement | null;
    reasonFilter?.addEventListener('change', () => {
      this.currentFilter.reason = reasonFilter.value as DedupeReason || undefined;
      this.loadEntries();
    });

    const sourceFilter = document.getElementById('dedupe-filter-source') as HTMLSelectElement | null;
    sourceFilter?.addEventListener('change', () => {
      this.currentFilter.source = sourceFilter.value || undefined;
      this.loadEntries();
    });

    // Refresh button
    const refreshBtn = document.getElementById('dedupe-refresh-btn');
    refreshBtn?.addEventListener('click', () => {
      this.loadStats();
      this.loadEntries();
    });

    // Export button
    const exportBtn = document.getElementById('dedupe-export-btn');
    exportBtn?.addEventListener('click', () => {
      this.exportToCSV();
    });
  }

  /**
   * Start auto-refresh interval
   */
  private startAutoRefresh(): void {
    this.refreshInterval = setInterval(() => {
      this.loadStats();
      this.loadEntries();
    }, this.config.autoRefreshMs);
  }

  /**
   * Stop auto-refresh
   */
  destroy(): void {
    if (this.refreshInterval) {
      clearInterval(this.refreshInterval);
      this.refreshInterval = null;
    }
  }

  /**
   * Load dedupe statistics
   */
  async loadStats(): Promise<void> {
    try {
      this.stats = await this.api.getDedupeAuditStats();
      this.renderStats();
    } catch (error) {
      logger.error('Failed to load stats:', error);
    }
  }

  /**
   * Load dedupe audit entries
   */
  async loadEntries(): Promise<void> {
    if (this.isLoading) return;

    this.isLoading = true;
    this.showLoading(true);

    try {
      const filter: DedupeAuditFilter = {
        limit: this.config.maxEntries,
      };

      if (this.currentFilter.status) {
        filter.status = this.currentFilter.status;
      }
      if (this.currentFilter.reason) {
        filter.reason = this.currentFilter.reason;
      }
      if (this.currentFilter.source) {
        filter.source = this.currentFilter.source;
      }

      const data = await this.api.getDedupeAuditEntries(filter);
      this.entries = data.entries || [];
      this.totalCount = data.total_count || 0;

      this.renderEntries();
    } catch (error) {
      logger.error('Failed to load entries:', error);
      this.showError('Failed to load dedupe audit entries');
    } finally {
      this.isLoading = false;
      this.showLoading(false);
    }
  }

  /**
   * Confirm a dedupe decision as correct
   */
  async confirmEntry(id: string): Promise<void> {
    try {
      await this.api.confirmDedupeEntry(id, {
        resolved_by: this.api.getUsername() || 'user',
      });

      this.showSuccess('Deduplication confirmed as correct');
      this.loadStats();
      this.loadEntries();
    } catch (error) {
      logger.error('Failed to confirm entry:', error);
      this.showError('Failed to confirm deduplication');
    }
  }

  /**
   * Restore a deduplicated event
   */
  async restoreEntry(id: string): Promise<void> {
    const userConfirmed = window.confirm(
      'This will insert the deduplicated event back into the database. Are you sure?'
    );
    if (!userConfirmed) return;

    try {
      await this.api.restoreDedupeEntry(id, {
        resolved_by: this.api.getUsername() || 'user',
      });

      this.showSuccess('Event restored successfully');
      this.loadStats();
      this.loadEntries();
    } catch (error) {
      logger.error('Failed to restore entry:', error);
      this.showError(`Failed to restore event: ${error}`);
    }
  }

  /**
   * Export audit log to CSV
   */
  exportToCSV(): void {
    const filter: DedupeAuditFilter = {};
    if (this.currentFilter.status) {
      filter.status = this.currentFilter.status;
    }
    if (this.currentFilter.reason) {
      filter.reason = this.currentFilter.reason;
    }
    if (this.currentFilter.source) {
      filter.source = this.currentFilter.source;
    }

    // Open in new tab to trigger download
    window.open(this.api.getDedupeAuditExportUrl(filter), '_blank');
  }

  /**
   * Render statistics dashboard
   */
  private renderStats(): void {
    if (!this.stats) return;

    // Total deduped
    const totalEl = document.getElementById('dedupe-stat-total');
    if (totalEl) totalEl.textContent = String(this.stats.total_deduped);

    // Pending review
    const pendingEl = document.getElementById('dedupe-stat-pending');
    if (pendingEl) pendingEl.textContent = String(this.stats.pending_review);

    // Restored
    const restoredEl = document.getElementById('dedupe-stat-restored');
    if (restoredEl) restoredEl.textContent = String(this.stats.user_restored);

    // Accuracy rate
    const accuracyEl = document.getElementById('dedupe-stat-accuracy');
    if (accuracyEl) {
      const rate = this.stats.accuracy_rate || 0;
      accuracyEl.textContent = `${rate.toFixed(1)}%`;
    }

    // Last 24h/7d/30d
    const last24hEl = document.getElementById('dedupe-stat-24h');
    if (last24hEl) last24hEl.textContent = String(this.stats.last_24_hours);

    const last7dEl = document.getElementById('dedupe-stat-7d');
    if (last7dEl) last7dEl.textContent = String(this.stats.last_7_days);

    const last30dEl = document.getElementById('dedupe-stat-30d');
    if (last30dEl) last30dEl.textContent = String(this.stats.last_30_days);

    // Breakdown charts (if containers exist)
    this.renderBreakdownChart('dedupe-chart-by-reason', this.stats.dedupe_by_reason, REASON_NAMES);
    this.renderBreakdownChart('dedupe-chart-by-layer', this.stats.dedupe_by_layer, LAYER_NAMES);
    this.renderBreakdownChart('dedupe-chart-by-source', this.stats.dedupe_by_source);
  }

  /**
   * Render a simple bar chart for breakdown data
   */
  private renderBreakdownChart(
    containerId: string,
    data: Record<string, number>,
    labels?: Record<string, string>
  ): void {
    const container = document.getElementById(containerId);
    if (!container) return;

    const entries = Object.entries(data);
    if (entries.length === 0) {
      container.innerHTML = '<div class="text-muted">No data</div>';
      return;
    }

    const maxValue = Math.max(...entries.map(([, v]) => v));

    const html = entries.map(([key, value]) => {
      const label = labels?.[key] || key;
      const percentage = maxValue > 0 ? (value / maxValue) * 100 : 0;
      return `
        <div class="breakdown-row">
          <span class="breakdown-label">${label}</span>
          <div class="breakdown-bar-container">
            <div class="breakdown-bar" style="width: ${percentage}%"></div>
          </div>
          <span class="breakdown-value">${value}</span>
        </div>
      `;
    }).join('');

    container.innerHTML = html;
  }

  /**
   * Render audit log entries
   */
  private renderEntries(): void {
    const container = document.getElementById('dedupe-audit-log');
    if (!container) return;

    if (this.entries.length === 0) {
      container.innerHTML = `
        <div class="empty-state">
          <p>No deduplication entries found</p>
          <p class="text-muted">Deduplicated events will appear here when they occur</p>
        </div>
      `;
      return;
    }

    const html = `
      <div class="dedupe-count">Showing ${this.entries.length} of ${this.totalCount} entries</div>
      <table class="dedupe-table">
        <thead>
          <tr>
            <th>Time</th>
            <th>User</th>
            <th>Title</th>
            <th>Source</th>
            <th>Reason</th>
            <th>Layer</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          ${this.entries.map(entry => this.renderEntryRow(entry)).join('')}
        </tbody>
      </table>
    `;

    container.innerHTML = html;

    // Attach event handlers
    this.entries.forEach(entry => {
      const confirmBtn = document.getElementById(`dedupe-confirm-${entry.id}`);
      confirmBtn?.addEventListener('click', () => this.confirmEntry(entry.id));

      const restoreBtn = document.getElementById(`dedupe-restore-${entry.id}`);
      restoreBtn?.addEventListener('click', () => this.restoreEntry(entry.id));

      const viewBtn = document.getElementById(`dedupe-view-${entry.id}`);
      viewBtn?.addEventListener('click', () => this.showEntryDetails(entry));
    });
  }

  /**
   * Render a single entry row
   */
  private renderEntryRow(entry: DedupeAuditEntry): string {
    const timestamp = new Date(entry.timestamp);
    const timeAgo = this.formatTimeAgo(timestamp);
    const statusConfig = STATUS_CONFIG[entry.status];
    const reasonName = REASON_NAMES[entry.dedupe_reason] || entry.dedupe_reason;
    const layerName = LAYER_NAMES[entry.dedupe_layer] || entry.dedupe_layer;

    const canAct = entry.status === 'auto_dedupe';

    return `
      <tr class="dedupe-entry status-${entry.status}">
        <td title="${timestamp.toLocaleString()}">${timeAgo}</td>
        <td>${entry.username || `User ${entry.user_id}`}</td>
        <td class="title-cell" title="${entry.title || 'Unknown'}">${entry.title || 'Unknown'}</td>
        <td>${entry.discarded_source}</td>
        <td>${reasonName}</td>
        <td>${layerName}</td>
        <td>
          <span class="status-badge" style="background-color: ${statusConfig.color}">
            ${statusConfig.icon} ${statusConfig.name}
          </span>
        </td>
        <td class="actions-cell">
          <button id="dedupe-view-${entry.id}" class="btn-icon" title="View details">\u{1F50D}</button>
          ${canAct ? `
            <button id="dedupe-confirm-${entry.id}" class="btn-icon btn-success" title="Confirm correct">\u2714</button>
            <button id="dedupe-restore-${entry.id}" class="btn-icon btn-warning" title="Restore event">\u21A9</button>
          ` : ''}
        </td>
      </tr>
    `;
  }

  /**
   * Show entry details in a modal
   */
  private showEntryDetails(entry: DedupeAuditEntry): void {
    const modal = document.getElementById('dedupe-detail-modal');
    if (!modal) return;

    const reasonName = REASON_NAMES[entry.dedupe_reason] || entry.dedupe_reason;
    const layerName = LAYER_NAMES[entry.dedupe_layer] || entry.dedupe_layer;
    const statusConfig = STATUS_CONFIG[entry.status];

    const content = `
      <div class="modal-header">
        <h3>Deduplication Details</h3>
        <button class="modal-close">\u2715</button>
      </div>
      <div class="modal-body">
        <div class="detail-section">
          <h4>Discarded Event</h4>
          <dl>
            <dt>Event ID</dt><dd>${entry.discarded_event_id}</dd>
            <dt>Source</dt><dd>${entry.discarded_source}</dd>
            <dt>Session Key</dt><dd>${entry.discarded_session_key || '-'}</dd>
            <dt>Correlation Key</dt><dd class="monospace">${entry.discarded_correlation_key || '-'}</dd>
            <dt>Started At</dt><dd>${entry.discarded_started_at ? new Date(entry.discarded_started_at).toLocaleString() : '-'}</dd>
          </dl>
        </div>

        <div class="detail-section">
          <h4>Media Information</h4>
          <dl>
            <dt>Title</dt><dd>${entry.title || '-'}</dd>
            <dt>Type</dt><dd>${entry.media_type || '-'}</dd>
            <dt>User</dt><dd>${entry.username || `User ${entry.user_id}`}</dd>
            <dt>Rating Key</dt><dd>${entry.rating_key || '-'}</dd>
          </dl>
        </div>

        <div class="detail-section">
          <h4>Deduplication</h4>
          <dl>
            <dt>Reason</dt><dd>${reasonName}</dd>
            <dt>Layer</dt><dd>${layerName}</dd>
            <dt>Timestamp</dt><dd>${new Date(entry.timestamp).toLocaleString()}</dd>
          </dl>
        </div>

        <div class="detail-section">
          <h4>Status</h4>
          <dl>
            <dt>Current Status</dt>
            <dd>
              <span class="status-badge" style="background-color: ${statusConfig.color}">
                ${statusConfig.icon} ${statusConfig.name}
              </span>
            </dd>
            ${entry.resolved_by ? `<dt>Resolved By</dt><dd>${entry.resolved_by}</dd>` : ''}
            ${entry.resolved_at ? `<dt>Resolved At</dt><dd>${new Date(entry.resolved_at).toLocaleString()}</dd>` : ''}
            ${entry.resolution_notes ? `<dt>Notes</dt><dd>${entry.resolution_notes}</dd>` : ''}
          </dl>
        </div>

        ${entry.discarded_raw_payload ? `
          <div class="detail-section">
            <h4>Raw Payload</h4>
            <details>
              <summary>Click to expand</summary>
              <pre class="raw-payload">${this.formatJSON(entry.discarded_raw_payload)}</pre>
            </details>
          </div>
        ` : ''}
      </div>
    `;

    const modalContent = modal.querySelector('.modal-content');
    if (modalContent) {
      modalContent.innerHTML = content;

      // CSP-compliant close button handler (replaces inline onclick)
      const closeBtn = modalContent.querySelector('.modal-close');
      if (closeBtn) {
        closeBtn.addEventListener('click', () => {
          modal.style.display = 'none';
        });
      }
    }
    modal.style.display = 'flex';
  }

  /**
   * Format JSON for display
   */
  private formatJSON(jsonString: string): string {
    try {
      const parsed = JSON.parse(jsonString);
      return JSON.stringify(parsed, null, 2);
    } catch {
      return jsonString;
    }
  }

  /**
   * Format time ago
   */
  private formatTimeAgo(date: Date): string {
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffSec = Math.floor(diffMs / 1000);
    const diffMin = Math.floor(diffSec / 60);
    const diffHour = Math.floor(diffMin / 60);
    const diffDay = Math.floor(diffHour / 24);

    if (diffSec < 60) return 'just now';
    if (diffMin < 60) return `${diffMin}m ago`;
    if (diffHour < 24) return `${diffHour}h ago`;
    if (diffDay < 7) return `${diffDay}d ago`;
    return date.toLocaleDateString();
  }

  /**
   * Show loading state
   */
  private showLoading(loading: boolean): void {
    const container = document.getElementById('dedupe-audit-log');
    if (!container) return;

    if (loading) {
      container.classList.add('loading');
    } else {
      container.classList.remove('loading');
    }
  }

  /**
   * Show error message
   */
  private showError(message: string): void {
    // Use toast notification if available
    const event = new CustomEvent('toast', {
      detail: { message, type: 'error' },
    });
    window.dispatchEvent(event);
  }

  /**
   * Show success message
   */
  private showSuccess(message: string): void {
    const event = new CustomEvent('toast', {
      detail: { message, type: 'success' },
    });
    window.dispatchEvent(event);
  }
}
