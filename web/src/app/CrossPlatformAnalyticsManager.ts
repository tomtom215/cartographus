// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * CrossPlatformAnalyticsManager - Cross-Platform Analytics Dashboard
 *
 * Displays unified analytics across Plex, Jellyfin, and Emby platforms.
 * Features:
 * - Summary dashboard with key metrics
 * - Platform usage comparison (pie chart)
 * - User and content link coverage (gauge charts)
 * - Activity trends by platform (line chart)
 * - Platform comparison matrix (bar chart)
 */

import * as echarts from 'echarts';
import type { API } from '../lib/api';
import type {
  CrossPlatformSummary,
  CrossPlatformUserStats,
  CrossPlatformContentStats
} from '../lib/types/cross-platform';
import { escapeHtml } from '../lib/sanitize';
import { getPlatformConfig, type Platform } from '../lib/components';
import { createLogger } from '../lib/logger';

const logger = createLogger('CrossPlatformAnalytics');

/**
 * Chart instance configuration
 */
interface ChartInstance {
  element: HTMLElement;
  chart: echarts.ECharts;
  id: string;
}

/**
 * View mode for analytics display
 */
type ViewMode = 'summary' | 'combined';

/**
 * CrossPlatformAnalyticsManager class
 */
export class CrossPlatformAnalyticsManager {
  private api: API;
  private container: HTMLElement | null = null;
  private charts: Map<string, ChartInstance> = new Map();
  private summary: CrossPlatformSummary | null = null;
  private viewMode: ViewMode = 'summary';
  private isLoading = false;

  // Event handler references for cleanup
  private viewModeHandler: ((e: Event) => void) | null = null;
  private refreshHandler: (() => void) | null = null;
  private resizeHandler: (() => void) | null = null;
  private resizeDebounceTimer: number | null = null;

  // Platform colors from config
  private platformColors: Record<string, string> = {};

  constructor(api: API) {
    this.api = api;
    this.initPlatformColors();
  }

  /**
   * Initialize platform colors from config
   */
  private initPlatformColors(): void {
    const platforms: Platform[] = ['plex', 'jellyfin', 'emby', 'tautulli'];
    platforms.forEach(platform => {
      const config = getPlatformConfig(platform);
      this.platformColors[platform] = config.color;
    });
  }

  /**
   * Initialize the analytics panel
   */
  async init(containerId: string): Promise<void> {
    this.container = document.getElementById(containerId);
    if (!this.container) {
      logger.warn('Container not found', { containerId });
      return;
    }

    logger.debug('Initializing');
    this.render();
    this.setupEventListeners();
    await this.loadData();
  }

  /**
   * Render the main analytics UI
   */
  private render(): void {
    if (!this.container) return;

    this.container.innerHTML = `
      <div class="cross-platform-analytics">
        <!-- Header -->
        <div class="cp-analytics-header">
          <h3 class="cp-analytics-title">Cross-Platform Analytics</h3>
          <div class="cp-analytics-controls">
            <div class="cp-view-toggle" role="group" aria-label="View mode">
              <button
                class="cp-view-btn active"
                data-view="summary"
                aria-pressed="true"
                title="Show platform breakdown"
              >
                By Platform
              </button>
              <button
                class="cp-view-btn"
                data-view="combined"
                aria-pressed="false"
                title="Show combined statistics"
              >
                Combined
              </button>
            </div>
            <button
              id="cp-analytics-refresh"
              class="btn btn-secondary btn-sm"
              title="Refresh analytics"
              aria-label="Refresh cross-platform analytics"
            >
              Refresh
            </button>
          </div>
        </div>

        <!-- Loading state -->
        <div id="cp-analytics-loading" class="cp-analytics-loading" aria-live="polite">
          <div class="loading-spinner"></div>
          <span>Loading analytics...</span>
        </div>

        <!-- Summary Cards -->
        <div id="cp-summary-cards" class="cp-summary-cards" style="display: none;">
          <div class="cp-summary-card">
            <div class="cp-summary-icon" aria-hidden="true">
              <svg viewBox="0 0 24 24" width="24" height="24" fill="currentColor">
                <path d="M4 6h16v2H4zm0 5h16v2H4zm0 5h16v2H4z"/>
              </svg>
            </div>
            <div class="cp-summary-content">
              <span class="cp-summary-value" id="cp-total-mappings">0</span>
              <span class="cp-summary-label">Content Mappings</span>
            </div>
          </div>
          <div class="cp-summary-card">
            <div class="cp-summary-icon" aria-hidden="true">
              <svg viewBox="0 0 24 24" width="24" height="24" fill="currentColor">
                <path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/>
              </svg>
            </div>
            <div class="cp-summary-content">
              <span class="cp-summary-value" id="cp-total-links">0</span>
              <span class="cp-summary-label">User Links</span>
            </div>
          </div>
          <div class="cp-summary-card">
            <div class="cp-summary-icon" aria-hidden="true">
              <svg viewBox="0 0 24 24" width="24" height="24" fill="currentColor">
                <path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-5 14H7v-2h7v2zm3-4H7v-2h10v2zm0-4H7V7h10v2z"/>
              </svg>
            </div>
            <div class="cp-summary-content">
              <span class="cp-summary-value" id="cp-platforms-count">0</span>
              <span class="cp-summary-label">Active Platforms</span>
            </div>
          </div>
        </div>

        <!-- Charts Grid -->
        <div id="cp-charts-container" class="cp-charts-container" style="display: none;">
          <!-- Platform Usage Chart -->
          <div class="cp-chart-card">
            <h4 class="cp-chart-title">Platform Usage</h4>
            <div
              id="cp-chart-platform-usage"
              class="cp-chart"
              role="img"
              aria-label="Platform usage distribution pie chart"
            ></div>
          </div>

          <!-- Coverage Gauges -->
          <div class="cp-chart-card cp-gauges-container">
            <h4 class="cp-chart-title">Link Coverage</h4>
            <div class="cp-gauges-row">
              <div class="cp-gauge-wrapper">
                <div
                  id="cp-chart-user-coverage"
                  class="cp-gauge"
                  role="img"
                  aria-label="User link coverage gauge"
                ></div>
                <span class="cp-gauge-label">Users</span>
              </div>
              <div class="cp-gauge-wrapper">
                <div
                  id="cp-chart-content-coverage"
                  class="cp-gauge"
                  role="img"
                  aria-label="Content mapping coverage gauge"
                ></div>
                <span class="cp-gauge-label">Content</span>
              </div>
            </div>
          </div>

          <!-- Platform Comparison Chart -->
          <div class="cp-chart-card cp-chart-wide">
            <h4 class="cp-chart-title">Platform Comparison</h4>
            <div
              id="cp-chart-platform-comparison"
              class="cp-chart"
              role="img"
              aria-label="Platform comparison bar chart showing users and content per platform"
            ></div>
          </div>
        </div>

        <!-- Empty State -->
        <div id="cp-analytics-empty" class="cp-analytics-empty" style="display: none;">
          <div class="cp-empty-icon" aria-hidden="true">
            <svg viewBox="0 0 24 24" width="48" height="48" fill="currentColor">
              <path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-5 14H7v-2h7v2zm3-4H7v-2h10v2zm0-4H7V7h10v2z"/>
            </svg>
          </div>
          <h4 class="cp-empty-title">No Cross-Platform Data</h4>
          <p class="cp-empty-message">
            Create content mappings and user links to see cross-platform analytics.
          </p>
        </div>
      </div>
    `;

    // Initialize chart containers
    this.initializeCharts();
  }

  /**
   * Initialize ECharts instances
   */
  private initializeCharts(): void {
    const chartConfigs = [
      { id: 'cp-chart-platform-usage', type: 'pie' },
      { id: 'cp-chart-user-coverage', type: 'gauge' },
      { id: 'cp-chart-content-coverage', type: 'gauge' },
      { id: 'cp-chart-platform-comparison', type: 'bar' }
    ];

    chartConfigs.forEach(config => {
      const element = document.getElementById(config.id);
      if (element) {
        const chart = echarts.init(element, 'dark', { renderer: 'canvas' });
        this.charts.set(config.id, { element, chart, id: config.id });
      }
    });
  }

  /**
   * Setup event listeners
   */
  private setupEventListeners(): void {
    // View mode toggle
    this.viewModeHandler = (e: Event) => {
      const target = e.target as HTMLElement;
      const btn = target.closest('.cp-view-btn') as HTMLButtonElement;
      if (!btn) return;

      const view = btn.dataset.view as ViewMode;
      if (view && view !== this.viewMode) {
        this.viewMode = view;
        this.updateViewModeButtons();
        this.renderCharts();
      }
    };

    const viewToggle = this.container?.querySelector('.cp-view-toggle');
    viewToggle?.addEventListener('click', this.viewModeHandler);

    // Refresh button
    this.refreshHandler = () => {
      this.loadData();
    };

    const refreshBtn = document.getElementById('cp-analytics-refresh');
    refreshBtn?.addEventListener('click', this.refreshHandler);

    // Resize handler with debounce
    this.resizeHandler = () => {
      if (this.resizeDebounceTimer !== null) {
        window.clearTimeout(this.resizeDebounceTimer);
      }
      this.resizeDebounceTimer = window.setTimeout(() => {
        this.resizeCharts();
        this.resizeDebounceTimer = null;
      }, 200);
    };

    window.addEventListener('resize', this.resizeHandler);
  }

  /**
   * Update view mode button states
   */
  private updateViewModeButtons(): void {
    const buttons = this.container?.querySelectorAll('.cp-view-btn');
    buttons?.forEach(btn => {
      const isActive = (btn as HTMLButtonElement).dataset.view === this.viewMode;
      btn.classList.toggle('active', isActive);
      btn.setAttribute('aria-pressed', String(isActive));
    });
  }

  /**
   * Load analytics data
   */
  async loadData(): Promise<void> {
    if (this.isLoading) return;

    this.isLoading = true;
    this.showLoading(true);

    try {
      const response = await this.api.getCrossPlatformSummary();

      if (response.success && response.data) {
        this.summary = response.data;
        this.showLoading(false);
        this.updateSummaryCards();
        this.renderCharts();
        this.showContent(true);
      } else {
        this.showLoading(false);
        this.showEmpty(true);
      }
    } catch (error) {
      logger.error('Failed to load data', { error });
      this.showLoading(false);
      this.showEmpty(true);
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * Show/hide loading state
   */
  private showLoading(show: boolean): void {
    const loading = document.getElementById('cp-analytics-loading');
    if (loading) {
      loading.style.display = show ? 'flex' : 'none';
    }
  }

  /**
   * Show/hide content
   */
  private showContent(show: boolean): void {
    const cards = document.getElementById('cp-summary-cards');
    const charts = document.getElementById('cp-charts-container');
    const empty = document.getElementById('cp-analytics-empty');

    if (cards) cards.style.display = show ? 'grid' : 'none';
    if (charts) charts.style.display = show ? 'grid' : 'none';
    if (empty) empty.style.display = 'none';
  }

  /**
   * Show/hide empty state
   */
  private showEmpty(show: boolean): void {
    const cards = document.getElementById('cp-summary-cards');
    const charts = document.getElementById('cp-charts-container');
    const empty = document.getElementById('cp-analytics-empty');

    if (cards) cards.style.display = 'none';
    if (charts) charts.style.display = 'none';
    if (empty) empty.style.display = show ? 'flex' : 'none';
  }

  /**
   * Update summary cards with data
   */
  private updateSummaryCards(): void {
    if (!this.summary) return;

    const mappingsEl = document.getElementById('cp-total-mappings');
    const linksEl = document.getElementById('cp-total-links');
    const platformsEl = document.getElementById('cp-platforms-count');

    if (mappingsEl) {
      mappingsEl.textContent = this.formatNumber(this.summary.total_content_mappings);
    }
    if (linksEl) {
      linksEl.textContent = this.formatNumber(this.summary.total_user_links);
    }
    if (platformsEl) {
      platformsEl.textContent = String(this.summary.platforms.length);
    }
  }

  /**
   * Format number with locale
   */
  private formatNumber(num: number): string {
    return num.toLocaleString();
  }

  /**
   * Render all charts
   */
  private renderCharts(): void {
    if (!this.summary) return;

    this.renderPlatformUsageChart();
    this.renderCoverageGauges();
    this.renderPlatformComparisonChart();
  }

  /**
   * Render platform usage pie chart
   */
  private renderPlatformUsageChart(): void {
    const chartInstance = this.charts.get('cp-chart-platform-usage');
    if (!chartInstance || !this.summary) return;

    const data = this.summary.platforms.map(platform => ({
      name: this.formatPlatformName(platform.name),
      value: platform.users + platform.content_items,
      itemStyle: {
        color: this.platformColors[platform.name.toLowerCase()] || '#666'
      }
    }));

    const option: echarts.EChartsOption = {
      tooltip: {
        trigger: 'item',
        formatter: (params: any) => {
          const platform = this.summary?.platforms.find(
            p => this.formatPlatformName(p.name) === params.name
          );
          if (!platform) return params.name;
          return `
            <div style="padding: 8px;">
              <strong>${escapeHtml(params.name)}</strong><br/>
              Users: ${platform.users.toLocaleString()}<br/>
              Content: ${platform.content_items.toLocaleString()}
            </div>
          `;
        }
      },
      legend: {
        orient: 'horizontal',
        bottom: 0,
        textStyle: { color: '#a0a0a0' }
      },
      series: [
        {
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['50%', '45%'],
          avoidLabelOverlap: true,
          itemStyle: {
            borderRadius: 4,
            borderColor: '#1a1a2e',
            borderWidth: 2
          },
          label: {
            show: false
          },
          emphasis: {
            label: {
              show: true,
              fontSize: 14,
              fontWeight: 'bold'
            },
            itemStyle: {
              shadowBlur: 10,
              shadowOffsetX: 0,
              shadowColor: 'rgba(0, 0, 0, 0.5)'
            }
          },
          data
        }
      ]
    };

    chartInstance.chart.setOption(option, true);
  }

  /**
   * Render coverage gauge charts
   */
  private renderCoverageGauges(): void {
    if (!this.summary) return;

    // User coverage gauge
    this.renderGauge(
      'cp-chart-user-coverage',
      this.summary.link_coverage.percentage,
      this.getGaugeColor(this.summary.link_coverage.percentage)
    );

    // Content coverage gauge
    this.renderGauge(
      'cp-chart-content-coverage',
      this.summary.content_coverage.percentage,
      this.getGaugeColor(this.summary.content_coverage.percentage)
    );
  }

  /**
   * Render a single gauge chart
   */
  private renderGauge(chartId: string, value: number, color: string): void {
    const chartInstance = this.charts.get(chartId);
    if (!chartInstance) return;

    const option: echarts.EChartsOption = {
      series: [
        {
          type: 'gauge',
          startAngle: 180,
          endAngle: 0,
          min: 0,
          max: 100,
          radius: '100%',
          center: ['50%', '75%'],
          splitNumber: 5,
          axisLine: {
            lineStyle: {
              width: 12,
              color: [
                [value / 100, color],
                [1, '#2a2a4a']
              ]
            }
          },
          axisTick: { show: false },
          splitLine: { show: false },
          axisLabel: { show: false },
          pointer: { show: false },
          title: { show: false },
          detail: {
            offsetCenter: [0, '-10%'],
            fontSize: 18,
            fontWeight: 'bold',
            color: color,
            formatter: (value: number) => `${value.toFixed(1)}%`
          },
          data: [{ value }]
        }
      ]
    };

    chartInstance.chart.setOption(option, true);
  }

  /**
   * Get color for gauge based on value
   */
  private getGaugeColor(value: number): string {
    if (value >= 80) return '#52c41a'; // Green - high coverage
    if (value >= 50) return '#faad14'; // Yellow - medium coverage
    return '#ff4d4f'; // Red - low coverage
  }

  /**
   * Render platform comparison bar chart
   */
  private renderPlatformComparisonChart(): void {
    const chartInstance = this.charts.get('cp-chart-platform-comparison');
    if (!chartInstance || !this.summary) return;

    const platforms = this.summary.platforms.map(p => this.formatPlatformName(p.name));
    const users = this.summary.platforms.map(p => p.users);
    const content = this.summary.platforms.map(p => p.content_items);
    const colors = this.summary.platforms.map(
      p => this.platformColors[p.name.toLowerCase()] || '#666'
    );

    const option: echarts.EChartsOption = {
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'shadow' }
      },
      legend: {
        data: ['Users', 'Content'],
        bottom: 0,
        textStyle: { color: '#a0a0a0' }
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '15%',
        top: '3%',
        containLabel: true
      },
      xAxis: {
        type: 'category',
        data: platforms,
        axisLine: { lineStyle: { color: '#444' } },
        axisLabel: { color: '#a0a0a0' }
      },
      yAxis: {
        type: 'value',
        axisLine: { lineStyle: { color: '#444' } },
        axisLabel: { color: '#a0a0a0' },
        splitLine: { lineStyle: { color: '#333' } }
      },
      series: [
        {
          name: 'Users',
          type: 'bar',
          data: users.map((value, index) => ({
            value,
            itemStyle: { color: colors[index] }
          })),
          barGap: '10%'
        },
        {
          name: 'Content',
          type: 'bar',
          data: content.map((value, index) => ({
            value,
            itemStyle: { color: this.adjustColorOpacity(colors[index], 0.6) }
          }))
        }
      ]
    };

    chartInstance.chart.setOption(option, true);
  }

  /**
   * Format platform name for display
   */
  private formatPlatformName(name: string): string {
    return name.charAt(0).toUpperCase() + name.slice(1).toLowerCase();
  }

  /**
   * Adjust color opacity
   */
  private adjustColorOpacity(hexColor: string, opacity: number): string {
    // Convert hex to rgba
    const hex = hexColor.replace('#', '');
    const r = parseInt(hex.substring(0, 2), 16);
    const g = parseInt(hex.substring(2, 4), 16);
    const b = parseInt(hex.substring(4, 6), 16);
    return `rgba(${r}, ${g}, ${b}, ${opacity})`;
  }

  /**
   * Resize all charts
   */
  private resizeCharts(): void {
    this.charts.forEach(instance => {
      instance.chart.resize();
    });
  }

  /**
   * Refresh analytics data
   */
  async refresh(): Promise<void> {
    await this.loadData();
  }

  /**
   * Get user statistics for a specific user
   */
  async getUserStats(userId: number): Promise<CrossPlatformUserStats | null> {
    try {
      const response = await this.api.getCrossPlatformUserStats(userId);
      if (response.success && response.data) {
        return response.data;
      }
    } catch (error) {
      logger.error('Failed to get user stats', { error, userId });
    }
    return null;
  }

  /**
   * Get content statistics for specific content
   */
  async getContentStats(contentId: number): Promise<CrossPlatformContentStats | null> {
    try {
      const response = await this.api.getCrossPlatformContentStats(contentId);
      if (response.success && response.data) {
        return response.data;
      }
    } catch (error) {
      logger.error('Failed to get content stats', { error, contentId });
    }
    return null;
  }

  /**
   * Render user stats widget (for embedding in user profile)
   */
  renderUserStatsWidget(container: HTMLElement, stats: CrossPlatformUserStats): void {
    const platformsHtml = stats.by_platform
      .map(p => {
        const config = getPlatformConfig(p.platform.toLowerCase() as Platform);
        return `
          <div class="cp-user-stat-platform">
            <span class="cp-platform-dot" style="background: ${config.color}"></span>
            <span class="cp-platform-name">${escapeHtml(this.formatPlatformName(p.platform))}</span>
            <span class="cp-platform-plays">${p.plays.toLocaleString()} plays</span>
            <span class="cp-platform-duration">${this.formatDuration(p.duration)}</span>
          </div>
        `;
      })
      .join('');

    container.innerHTML = `
      <div class="cp-user-stats-widget">
        <div class="cp-user-stats-header">
          <h4>Cross-Platform Activity</h4>
          <span class="cp-user-stats-total">
            ${stats.total_plays.toLocaleString()} total plays
          </span>
        </div>
        <div class="cp-user-stats-platforms">
          ${platformsHtml || '<div class="cp-user-stats-empty">No cross-platform activity</div>'}
        </div>
        <div class="cp-user-stats-footer">
          <span class="cp-user-stats-duration">
            Total: ${this.formatDuration(stats.total_duration)}
          </span>
          <span class="cp-user-stats-linked">
            ${stats.linked_users.length} linked identities
          </span>
        </div>
      </div>
    `;
  }

  /**
   * Render content stats widget (for embedding in content details)
   */
  renderContentStatsWidget(container: HTMLElement, stats: CrossPlatformContentStats): void {
    const platformsHtml = stats.by_platform
      .map(p => {
        const config = getPlatformConfig(p.platform.toLowerCase() as Platform);
        return `
          <div class="cp-content-stat-platform">
            <span class="cp-platform-dot" style="background: ${config.color}"></span>
            <span class="cp-platform-name">${escapeHtml(this.formatPlatformName(p.platform))}</span>
            <span class="cp-platform-plays">${p.plays.toLocaleString()} plays</span>
            <span class="cp-platform-users">${p.unique_users.toLocaleString()} users</span>
          </div>
        `;
      })
      .join('');

    const externalIds = [];
    if (stats.external_ids.imdb_id) {
      externalIds.push(`IMDb: ${escapeHtml(stats.external_ids.imdb_id)}`);
    }
    if (stats.external_ids.tmdb_id) {
      externalIds.push(`TMDb: ${stats.external_ids.tmdb_id}`);
    }
    if (stats.external_ids.tvdb_id) {
      externalIds.push(`TVDb: ${stats.external_ids.tvdb_id}`);
    }

    container.innerHTML = `
      <div class="cp-content-stats-widget">
        <div class="cp-content-stats-header">
          <h4>Cross-Platform Statistics</h4>
          <span class="cp-content-stats-type">${escapeHtml(stats.media_type)}</span>
        </div>
        <div class="cp-content-stats-summary">
          <div class="cp-content-stat">
            <span class="cp-content-stat-value">${stats.total_plays.toLocaleString()}</span>
            <span class="cp-content-stat-label">Total Plays</span>
          </div>
          <div class="cp-content-stat">
            <span class="cp-content-stat-value">${stats.total_users.toLocaleString()}</span>
            <span class="cp-content-stat-label">Unique Users</span>
          </div>
        </div>
        <div class="cp-content-stats-platforms">
          ${platformsHtml || '<div class="cp-content-stats-empty">No cross-platform data</div>'}
        </div>
        ${externalIds.length > 0 ? `
          <div class="cp-content-stats-external">
            <span class="cp-external-label">External IDs:</span>
            <span class="cp-external-ids">${externalIds.join(' | ')}</span>
          </div>
        ` : ''}
      </div>
    `;
  }

  /**
   * Format duration in seconds to human readable
   */
  private formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);

    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  }

  /**
   * Get current summary data
   */
  getSummary(): CrossPlatformSummary | null {
    return this.summary;
  }

  /**
   * Clean up resources
   */
  destroy(): void {
    // Remove event listeners
    const viewToggle = this.container?.querySelector('.cp-view-toggle');
    if (viewToggle && this.viewModeHandler) {
      viewToggle.removeEventListener('click', this.viewModeHandler);
    }

    const refreshBtn = document.getElementById('cp-analytics-refresh');
    if (refreshBtn && this.refreshHandler) {
      refreshBtn.removeEventListener('click', this.refreshHandler);
    }

    if (this.resizeHandler) {
      window.removeEventListener('resize', this.resizeHandler);
    }

    // Clear debounce timer
    if (this.resizeDebounceTimer !== null) {
      window.clearTimeout(this.resizeDebounceTimer);
    }

    // Dispose charts
    this.charts.forEach(instance => {
      instance.chart.dispose();
    });
    this.charts.clear();

    // Clear references
    this.viewModeHandler = null;
    this.refreshHandler = null;
    this.resizeHandler = null;
    this.container = null;
    this.summary = null;

    logger.debug('Destroyed');
  }
}
