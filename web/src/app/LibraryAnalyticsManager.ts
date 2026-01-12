// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Library Analytics Manager
 * Manages the library deep-dive dashboard including:
 * - Library selector population
 * - Library data loading
 * - Stats display updates
 * - Chart rendering coordination
 */

import * as echarts from 'echarts';
import type { API, LibraryAnalytics, TautulliLibraryNameItem, LocationFilter } from '../lib/api';
import { createLogger } from '../lib/logger';
import { LibraryChartRenderer } from '../lib/charts/renderers/LibraryChartRenderer';

const logger = createLogger('LibraryAnalytics');

export class LibraryAnalyticsManager {
  private api: API;
  private filterManager: { buildFilter: () => LocationFilter } | null = null;
  private charts: Map<string, echarts.ECharts> = new Map();
  private currentLibraryId: number | null = null;
  private librariesLoaded: boolean = false;

  constructor(api: API) {
    this.api = api;
    this.setupEventListeners();
  }

  /**
   * Initialize the library analytics tab (called when tab is first shown)
   */
  async init(): Promise<void> {
    if (!this.librariesLoaded) {
      await this.loadLibraryNames();
      this.librariesLoaded = true;
    }
    this.initializeCharts();
  }

  /**
   * Set filter manager reference for building filters
   */
  setFilterManager(manager: { buildFilter: () => LocationFilter }): void {
    this.filterManager = manager;
  }

  /**
   * Set up event listeners for library selector and page navigation
   */
  private setupEventListeners(): void {
    const librarySelector = document.getElementById('library-selector') as HTMLSelectElement;
    if (librarySelector) {
      librarySelector.addEventListener('change', async () => {
        const sectionId = parseInt(librarySelector.value, 10);
        if (sectionId > 0) {
          await this.loadLibraryAnalytics(sectionId);
        } else {
          this.showEmptyState();
        }
      });
    }
  }

  /**
   * Initialize charts when library tab is first shown
   */
  initializeCharts(): void {
    const chartIds = ['chart-library-users', 'chart-library-trend', 'chart-library-quality'];

    chartIds.forEach(chartId => {
      const container = document.getElementById(chartId);
      if (container && !this.charts.has(chartId)) {
        const chart = echarts.init(container, 'dark');
        this.charts.set(chartId, chart);
      }
    });
  }

  /**
   * Load library names from Tautulli API
   */
  private async loadLibraryNames(): Promise<void> {
    try {
      const libraries = await this.api.getTautulliLibraryNames();
      this.populateLibrarySelector(libraries);
    } catch (error) {
      logger.error('Failed to load library names:', error);
      // Show error in selector
      const selector = document.getElementById('library-selector') as HTMLSelectElement;
      if (selector) {
        selector.innerHTML = '<option value="">-- Error loading libraries --</option>';
      }
    }
  }

  /**
   * Populate the library selector with available libraries
   */
  private populateLibrarySelector(libraries: TautulliLibraryNameItem[]): void {
    const selector = document.getElementById('library-selector') as HTMLSelectElement;
    if (!selector) return;

    // Clear existing options (keep the placeholder)
    selector.innerHTML = '<option value="">-- Select a library --</option>';

    // Add library options
    libraries.forEach(lib => {
      const option = document.createElement('option');
      option.value = lib.section_id.toString();
      option.textContent = `${lib.section_name}${lib.section_type ? ` (${lib.section_type})` : ''}`;
      selector.appendChild(option);
    });
  }

  /**
   * Load analytics for a specific library
   */
  async loadLibraryAnalytics(sectionId: number): Promise<void> {
    this.currentLibraryId = sectionId;
    this.showLoading(true);

    try {
      const filter = this.filterManager?.buildFilter() || {};
      const data = await this.api.getAnalyticsLibrary(sectionId, filter);

      this.hideEmptyState();
      this.updateStats(data);
      this.updateHealthMetrics(data);
      this.renderCharts(data);
    } catch (error) {
      logger.error('Failed to load library analytics:', error);
      this.showError('Failed to load library analytics. Please try again.');
    } finally {
      this.showLoading(false);
    }
  }

  /**
   * Update the overview stats display
   */
  private updateStats(data: LibraryAnalytics): void {
    const statsContainer = document.getElementById('library-overview-stats');
    if (statsContainer) {
      statsContainer.style.display = 'block';
    }

    this.updateElement('library-total-items', data.total_items.toLocaleString());
    this.updateElement('library-watched-items', data.watched_items.toLocaleString());
    this.updateElement('library-watched-percent', `${data.watched_percentage.toFixed(1)}%`);
    this.updateElement('library-total-playbacks', data.total_playbacks.toLocaleString());
    this.updateElement('library-unique-users', data.unique_users.toLocaleString());

    // Format watch time as hours
    const watchTimeHours = Math.round(data.total_watch_time_minutes / 60);
    this.updateElement('library-watch-time', `${watchTimeHours.toLocaleString()}h`);

    this.updateElement('library-avg-completion', `${data.avg_completion.toFixed(1)}%`);
    this.updateElement('library-most-watched', data.most_watched_item || '-');
  }

  /**
   * Update health metrics display
   */
  private updateHealthMetrics(data: LibraryAnalytics): void {
    if (!data.content_health) return;

    const health = data.content_health;

    this.updateElement('health-stale-content', health.stale_content_count.toString());
    this.updateElement('health-popularity', health.popularity_score.toFixed(1));
    this.updateElement('health-engagement', `${health.engagement_score.toFixed(0)}%`);

    // Format growth rate with color indication
    const growthEl = document.getElementById('health-growth');
    if (growthEl) {
      const growth = health.growth_rate_percent;
      const sign = growth >= 0 ? '+' : '';
      growthEl.textContent = `${sign}${growth.toFixed(1)}%`;
      growthEl.style.color = growth >= 0 ? '#00d4aa' : '#f44336';
    }
  }

  /**
   * Render all library charts
   */
  private renderCharts(data: LibraryAnalytics): void {
    this.initializeCharts();

    // Render users chart
    const usersChart = this.charts.get('chart-library-users');
    if (usersChart) {
      const renderer = new LibraryChartRenderer({ chartId: 'chart-library-users', chart: usersChart });
      renderer.renderLibraryUsers(data);
    }

    // Render trend chart
    const trendChart = this.charts.get('chart-library-trend');
    if (trendChart) {
      const renderer = new LibraryChartRenderer({ chartId: 'chart-library-trend', chart: trendChart });
      renderer.renderLibraryTrend(data);
    }

    // Render quality chart
    const qualityChart = this.charts.get('chart-library-quality');
    if (qualityChart) {
      const renderer = new LibraryChartRenderer({ chartId: 'chart-library-quality', chart: qualityChart });
      renderer.renderLibraryQuality(data);
    }
  }

  /**
   * Show empty state when no library is selected
   */
  private showEmptyState(): void {
    const emptyState = document.getElementById('library-empty-state');
    const statsContainer = document.getElementById('library-overview-stats');
    const chartGrid = document.querySelector('#analytics-library .chart-grid') as HTMLElement;

    if (emptyState) emptyState.style.display = 'block';
    if (statsContainer) statsContainer.style.display = 'none';
    if (chartGrid) chartGrid.style.display = 'none';
  }

  /**
   * Hide empty state
   */
  private hideEmptyState(): void {
    const emptyState = document.getElementById('library-empty-state');
    const chartGrid = document.querySelector('#analytics-library .chart-grid') as HTMLElement;

    if (emptyState) emptyState.style.display = 'none';
    if (chartGrid) chartGrid.style.display = 'grid';
  }

  /**
   * Show loading indicator
   */
  private showLoading(show: boolean): void {
    const loadingEl = document.getElementById('library-loading');
    if (loadingEl) {
      loadingEl.style.display = show ? 'inline' : 'none';
    }
  }

  /**
   * Show error message
   */
  private showError(message: string): void {
    logger.error(message);
    // Could integrate with ToastManager if available
  }

  /**
   * Helper to update element text content
   */
  private updateElement(id: string, value: string): void {
    const el = document.getElementById(id);
    if (el) {
      el.textContent = value;
    }
  }

  /**
   * Resize charts on window resize
   */
  resizeCharts(): void {
    this.charts.forEach(chart => {
      chart.resize();
    });
  }

  /**
   * Dispose charts and clean up
   */
  dispose(): void {
    this.charts.forEach(chart => {
      chart.dispose();
    });
    this.charts.clear();
  }

  /**
   * Refresh data for current library (called when filters change)
   */
  async refresh(): Promise<void> {
    if (this.currentLibraryId) {
      await this.loadLibraryAnalytics(this.currentLibraryId);
    }
  }

  /**
   * Cleanup resources to prevent memory leaks
   * Disposes charts and clears references
   */
  destroy(): void {
    this.dispose();
    this.filterManager = null;
    this.currentLibraryId = null;
  }
}
