// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * DataCachingIndicatorManager - Shows cache vs live data status
 *
 * Features:
 * - Displays whether current data is from cache or live query
 * - Shows query performance metrics (response time)
 * - Visual indicator with cache/live status
 * - Tooltip with detailed cache statistics
 */

import { createLogger } from '../lib/logger';

const logger = createLogger('DataCachingIndicatorManager');

/**
 * Cache status entry for tracking history
 */
interface CacheEntry {
  timestamp: number;
  cached: boolean;
  queryTimeMs: number;
}

/**
 * Configuration for cache indicator
 */
interface CacheIndicatorConfig {
  maxHistory: number;      // Maximum entries to track
  fadeTimeout: number;     // Time before indicator fades (ms)
}

const DEFAULT_CONFIG: CacheIndicatorConfig = {
  maxHistory: 20,
  fadeTimeout: 5000
};

export class DataCachingIndicatorManager {
  private config: CacheIndicatorConfig;
  private history: CacheEntry[] = [];
  private indicator: HTMLElement | null = null;
  private fadeTimer: ReturnType<typeof setTimeout> | null = null;
  private totalCacheHits: number = 0;
  private totalRequests: number = 0;
  // AbortController for clean event listener removal
  private abortController: AbortController | null = null;
  // Track modal escape handler for cleanup
  private modalEscapeHandler: ((e: KeyboardEvent) => void) | null = null;

  constructor(config: Partial<CacheIndicatorConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
  }

  /**
   * Initialize the cache indicator
   */
  init(): void {
    this.createIndicator();
    logger.log('DataCachingIndicatorManager initialized');
  }

  /**
   * Create the cache status indicator element
   */
  private createIndicator(): void {
    // Create AbortController for cleanup
    this.abortController = new AbortController();
    const signal = this.abortController.signal;

    // Create indicator container
    this.indicator = document.createElement('div');
    this.indicator.id = 'cache-status-indicator';
    this.indicator.className = 'cache-status-indicator';
    this.indicator.setAttribute('role', 'status');
    this.indicator.setAttribute('aria-live', 'polite');
    this.indicator.setAttribute('aria-label', 'Data source status');
    this.indicator.title = 'Data source status';

    // Create indicator content
    this.indicator.innerHTML = `
      <div class="cache-indicator-icon" aria-hidden="true">
        <span class="cache-dot"></span>
      </div>
      <div class="cache-indicator-content">
        <span class="cache-indicator-label">Waiting for data...</span>
        <span class="cache-indicator-time"></span>
      </div>
      <button class="cache-indicator-details-btn" aria-label="Show cache statistics" title="Cache Statistics">
        <span aria-hidden="true">&#x2139;</span>
      </button>
    `;

    // Add click handler for details
    const detailsBtn = this.indicator.querySelector('.cache-indicator-details-btn');
    if (detailsBtn) {
      detailsBtn.addEventListener('click', () => this.showStatistics(), { signal });
    }

    // Add hover tooltip
    this.indicator.addEventListener('mouseenter', () => this.showTooltip(), { signal });
    this.indicator.addEventListener('mouseleave', () => this.hideTooltip(), { signal });

    document.body.appendChild(this.indicator);
  }

  /**
   * Update cache status from API response
   * This is called by the API class when a response is received
   */
  updateStatus(cached: boolean, queryTimeMs: number): void {
    const entry: CacheEntry = {
      timestamp: Date.now(),
      cached,
      queryTimeMs
    };

    // Track history
    this.history.unshift(entry);
    if (this.history.length > this.config.maxHistory) {
      this.history.pop();
    }

    // Update statistics
    this.totalRequests++;
    if (cached) {
      this.totalCacheHits++;
    }

    // Update indicator UI
    this.updateIndicatorUI(cached, queryTimeMs);

    // Reset fade timer
    this.resetFadeTimer();
  }

  /**
   * Update the indicator UI based on cache status
   */
  private updateIndicatorUI(cached: boolean, queryTimeMs: number): void {
    if (!this.indicator) return;

    // Update class for styling
    this.indicator.classList.remove('cache-hit', 'cache-miss', 'faded');
    this.indicator.classList.add(cached ? 'cache-hit' : 'cache-miss');

    // Update label
    const label = this.indicator.querySelector('.cache-indicator-label');
    if (label) {
      label.textContent = cached ? 'Cached data' : 'Live data';
    }

    // Update time
    const time = this.indicator.querySelector('.cache-indicator-time');
    if (time) {
      if (queryTimeMs > 0) {
        time.textContent = `${queryTimeMs}ms`;
      } else if (cached) {
        time.textContent = '<1ms';
      } else {
        time.textContent = '';
      }
    }

    // Update ARIA label
    this.indicator.setAttribute(
      'aria-label',
      `Data source: ${cached ? 'from cache' : 'live query'}${queryTimeMs > 0 ? `, ${queryTimeMs}ms` : ''}`
    );
  }

  /**
   * Reset the fade timer
   */
  private resetFadeTimer(): void {
    if (this.fadeTimer) {
      clearTimeout(this.fadeTimer);
    }

    this.fadeTimer = setTimeout(() => {
      this.indicator?.classList.add('faded');
    }, this.config.fadeTimeout);
  }

  /**
   * Show tooltip with cache statistics
   */
  private showTooltip(): void {
    if (!this.indicator) return;

    // Remove any existing tooltip
    this.hideTooltip();

    const tooltip = document.createElement('div');
    tooltip.className = 'cache-indicator-tooltip';
    tooltip.setAttribute('role', 'tooltip');

    const hitRate = this.totalRequests > 0
      ? Math.round((this.totalCacheHits / this.totalRequests) * 100)
      : 0;

    const avgTime = this.history.length > 0
      ? Math.round(this.history.reduce((sum, e) => sum + e.queryTimeMs, 0) / this.history.length)
      : 0;

    const recentHits = this.history.filter(e => e.cached).length;
    const recentTotal = this.history.length;

    tooltip.innerHTML = `
      <div class="cache-tooltip-header">Cache Statistics</div>
      <div class="cache-tooltip-stats">
        <div class="cache-stat-row">
          <span class="cache-stat-label">Cache Hit Rate:</span>
          <span class="cache-stat-value">${hitRate}%</span>
        </div>
        <div class="cache-stat-row">
          <span class="cache-stat-label">Avg Response:</span>
          <span class="cache-stat-value">${avgTime}ms</span>
        </div>
        <div class="cache-stat-row">
          <span class="cache-stat-label">Recent (${recentTotal}):</span>
          <span class="cache-stat-value">${recentHits} cached</span>
        </div>
        <div class="cache-stat-row">
          <span class="cache-stat-label">Total Requests:</span>
          <span class="cache-stat-value">${this.totalRequests}</span>
        </div>
      </div>
    `;

    this.indicator.appendChild(tooltip);
  }

  /**
   * Hide the tooltip
   */
  private hideTooltip(): void {
    const tooltip = this.indicator?.querySelector('.cache-indicator-tooltip');
    if (tooltip) {
      tooltip.remove();
    }
  }

  /**
   * Show detailed statistics in a modal
   */
  private showStatistics(): void {
    // Create modal for detailed statistics
    const existingModal = document.getElementById('cache-stats-modal');
    if (existingModal) {
      existingModal.remove();
    }

    const modal = document.createElement('div');
    modal.id = 'cache-stats-modal';
    modal.className = 'cache-stats-modal';
    modal.setAttribute('role', 'dialog');
    modal.setAttribute('aria-labelledby', 'cache-stats-title');
    modal.setAttribute('aria-modal', 'true');

    const hitRate = this.totalRequests > 0
      ? ((this.totalCacheHits / this.totalRequests) * 100).toFixed(1)
      : '0.0';

    const avgCachedTime = this.history.filter(e => e.cached).length > 0
      ? Math.round(this.history.filter(e => e.cached).reduce((sum, e) => sum + e.queryTimeMs, 0) / this.history.filter(e => e.cached).length)
      : 0;

    const avgLiveTime = this.history.filter(e => !e.cached).length > 0
      ? Math.round(this.history.filter(e => !e.cached).reduce((sum, e) => sum + e.queryTimeMs, 0) / this.history.filter(e => !e.cached).length)
      : 0;

    modal.innerHTML = `
      <div class="cache-stats-modal-content">
        <div class="cache-stats-modal-header">
          <h2 id="cache-stats-title">Data Cache Statistics</h2>
          <button class="cache-stats-close-btn" aria-label="Close statistics">&times;</button>
        </div>
        <div class="cache-stats-modal-body">
          <div class="cache-stats-grid">
            <div class="cache-stat-card">
              <div class="cache-stat-number">${hitRate}%</div>
              <div class="cache-stat-desc">Cache Hit Rate</div>
            </div>
            <div class="cache-stat-card">
              <div class="cache-stat-number">${this.totalRequests}</div>
              <div class="cache-stat-desc">Total Requests</div>
            </div>
            <div class="cache-stat-card">
              <div class="cache-stat-number">${this.totalCacheHits}</div>
              <div class="cache-stat-desc">Cache Hits</div>
            </div>
            <div class="cache-stat-card">
              <div class="cache-stat-number">${this.totalRequests - this.totalCacheHits}</div>
              <div class="cache-stat-desc">Live Queries</div>
            </div>
          </div>
          <div class="cache-stats-performance">
            <h3>Response Times</h3>
            <div class="cache-perf-row">
              <span>Avg Cached Response:</span>
              <span class="cache-perf-value cache-hit">${avgCachedTime}ms</span>
            </div>
            <div class="cache-perf-row">
              <span>Avg Live Response:</span>
              <span class="cache-perf-value cache-miss">${avgLiveTime}ms</span>
            </div>
          </div>
          <div class="cache-stats-history">
            <h3>Recent Requests</h3>
            <div class="cache-history-list">
              ${this.history.slice(0, 10).map(e => `
                <div class="cache-history-item ${e.cached ? 'cache-hit' : 'cache-miss'}">
                  <span class="cache-history-status">${e.cached ? 'Cached' : 'Live'}</span>
                  <span class="cache-history-time">${e.queryTimeMs}ms</span>
                  <span class="cache-history-ago">${this.formatTimeAgo(e.timestamp)}</span>
                </div>
              `).join('')}
            </div>
          </div>
        </div>
      </div>
    `;

    // Add close handler
    const closeBtn = modal.querySelector('.cache-stats-close-btn');
    if (closeBtn) {
      closeBtn.addEventListener('click', () => modal.remove());
    }

    // Close on backdrop click
    modal.addEventListener('click', (e) => {
      if (e.target === modal) {
        modal.remove();
      }
    });

    // Close on Escape - track handler for cleanup
    this.modalEscapeHandler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        modal.remove();
        if (this.modalEscapeHandler) {
          document.removeEventListener('keydown', this.modalEscapeHandler);
          this.modalEscapeHandler = null;
        }
      }
    };
    document.addEventListener('keydown', this.modalEscapeHandler);

    document.body.appendChild(modal);

    // Focus close button for accessibility
    (closeBtn as HTMLElement)?.focus();
  }

  /**
   * Format timestamp as time ago
   */
  private formatTimeAgo(timestamp: number): string {
    const seconds = Math.floor((Date.now() - timestamp) / 1000);
    if (seconds < 60) return 'just now';
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    return `${Math.floor(seconds / 3600)}h ago`;
  }

  /**
   * Get cache statistics
   */
  getStatistics(): { hitRate: number; totalRequests: number; totalHits: number; avgTime: number } {
    const hitRate = this.totalRequests > 0
      ? (this.totalCacheHits / this.totalRequests) * 100
      : 0;
    const avgTime = this.history.length > 0
      ? this.history.reduce((sum, e) => sum + e.queryTimeMs, 0) / this.history.length
      : 0;

    return {
      hitRate,
      totalRequests: this.totalRequests,
      totalHits: this.totalCacheHits,
      avgTime
    };
  }

  /**
   * Reset statistics
   */
  resetStatistics(): void {
    this.history = [];
    this.totalRequests = 0;
    this.totalCacheHits = 0;
  }

  /**
   * Clean up event listeners and timers
   */
  destroy(): void {
    // Abort all event listeners
    if (this.abortController) {
      this.abortController.abort();
      this.abortController = null;
    }

    // Clear fade timer
    if (this.fadeTimer) {
      clearTimeout(this.fadeTimer);
      this.fadeTimer = null;
    }

    // Remove modal escape handler if active
    if (this.modalEscapeHandler) {
      document.removeEventListener('keydown', this.modalEscapeHandler);
      this.modalEscapeHandler = null;
    }

    // Remove modal if open
    const modal = document.getElementById('cache-stats-modal');
    if (modal) {
      modal.remove();
    }

    // Remove indicator from DOM
    if (this.indicator) {
      this.indicator.remove();
      this.indicator = null;
    }

    // Clear history
    this.history = [];
    this.totalRequests = 0;
    this.totalCacheHits = 0;

    logger.log('DataCachingIndicatorManager destroyed');
  }
}

export default DataCachingIndicatorManager;
