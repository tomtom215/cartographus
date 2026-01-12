// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * GeographicDrillDownManager - Country -> Region -> City drill-down
 *
 * Features:
 * - Three-level geographic hierarchy: Country -> Region -> City
 * - Click to drill down to next level
 * - Breadcrumb navigation to go back up
 * - Aggregated statistics at each level
 * - ECharts horizontal bar chart visualization
 * - Click on map to drill down by location
 */

import type { API, LocationFilter } from '../lib/api';
import type { LocationStats } from '../lib/types/core';
import { createLogger } from '../lib/logger';

const logger = createLogger('GeographicDrillDown');

type DrillLevel = 'country' | 'region' | 'city';

interface AggregatedGeoData {
  name: string;
  playbackCount: number;
  uniqueUsers: number;
  avgCompletion: number;
  childCount: number;
}

interface DrillDownState {
  level: DrillLevel;
  country?: string;
  region?: string;
}

export class GeographicDrillDownManager {
  private api: API;
  private containerId: string;
  private locations: LocationStats[] = [];
  private currentState: DrillDownState = { level: 'country' };
  private isLoading: boolean = false;
  private chartInstance: unknown | null = null;

  // Store bound resize handler for cleanup
  private boundHandleResize: () => void;
  private resizeListenerAttached: boolean = false;

  constructor(api: API, containerId: string = 'geographic-drilldown-container') {
    this.api = api;
    this.containerId = containerId;

    // Bind resize handler for cleanup
    this.boundHandleResize = this.handleChartResize.bind(this);
  }

  /**
   * Handle window resize for chart
   */
  private handleChartResize(): void {
    if (this.chartInstance) {
      (this.chartInstance as { resize: () => void }).resize();
    }
  }

  /**
   * Initialize the geographic drill-down manager
   */
  async init(): Promise<void> {
    this.setupContainer();
    this.setupEventListeners();
    logger.debug('GeographicDrillDownManager initialized');
  }

  /**
   * Set up the container HTML
   */
  private setupContainer(): void {
    const container = document.getElementById(this.containerId);
    if (!container) {
      logger.warn('Geographic drill-down container not found');
      return;
    }

    container.innerHTML = `
      <div class="geo-drilldown-header">
        <h4 class="geo-drilldown-title">Geographic Distribution</h4>
        <nav class="geo-drilldown-breadcrumb" id="geo-breadcrumb" role="navigation" aria-label="Geographic navigation">
          <button class="geo-breadcrumb-item active" data-level="country" aria-current="page">
            Countries
          </button>
        </nav>
      </div>
      <div class="geo-drilldown-chart" id="geo-drilldown-chart" role="img" aria-label="Geographic distribution chart">
        <div class="geo-drilldown-loading">Loading geographic data...</div>
      </div>
      <div class="geo-drilldown-stats" id="geo-drilldown-stats" aria-live="polite">
        <!-- Stats will be populated dynamically -->
      </div>
    `;
  }

  /**
   * Set up event listeners
   */
  private setupEventListeners(): void {
    const breadcrumb = document.getElementById('geo-breadcrumb');
    if (breadcrumb) {
      breadcrumb.addEventListener('click', (e) => {
        const target = e.target as HTMLElement;
        const levelButton = target.closest('[data-level]') as HTMLElement;
        if (levelButton) {
          const level = levelButton.dataset.level as DrillLevel;
          this.navigateToLevel(level);
        }
      });
    }
  }

  /**
   * Load data and render chart
   */
  async loadData(filter: LocationFilter = {}): Promise<void> {
    if (this.isLoading) return;

    this.isLoading = true;
    this.showLoading();

    try {
      const data = await this.api.getLocations(filter);
      this.locations = data;
      this.renderCurrentLevel();
    } catch (error) {
      logger.error('Failed to load geographic data:', error);
      this.showError();
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * Show loading state
   */
  private showLoading(): void {
    const chartEl = document.getElementById('geo-drilldown-chart');
    if (chartEl) {
      chartEl.innerHTML = '<div class="geo-drilldown-loading">Loading geographic data...</div>';
    }
  }

  /**
   * Show error state
   */
  private showError(): void {
    const chartEl = document.getElementById('geo-drilldown-chart');
    if (chartEl) {
      chartEl.innerHTML = '<div class="geo-drilldown-error">Failed to load geographic data</div>';
    }
  }

  /**
   * Render the current drill-down level
   */
  private renderCurrentLevel(): void {
    const aggregatedData = this.aggregateDataForLevel();
    this.updateBreadcrumb();
    this.renderChart(aggregatedData);
    this.updateStats(aggregatedData);
  }

  /**
   * Aggregate data based on current drill level
   */
  private aggregateDataForLevel(): AggregatedGeoData[] {
    const { level, country, region } = this.currentState;
    const aggregated = new Map<string, AggregatedGeoData>();

    let filteredLocations = this.locations;

    // Filter by current context
    if (level === 'region' && country) {
      filteredLocations = this.locations.filter(loc => loc.country === country);
    } else if (level === 'city' && country) {
      filteredLocations = this.locations.filter(loc => {
        return loc.country === country && (!region || loc.region === region);
      });
    }

    // Aggregate based on level
    for (const loc of filteredLocations) {
      let key: string;
      if (level === 'country') {
        key = loc.country || 'Unknown';
      } else if (level === 'region') {
        key = loc.region || loc.city || 'Unknown Region';
      } else {
        key = loc.city || 'Unknown City';
      }

      const existing = aggregated.get(key);
      if (existing) {
        existing.playbackCount += loc.playback_count;
        existing.uniqueUsers += loc.unique_users;
        existing.avgCompletion = (existing.avgCompletion + loc.avg_completion) / 2;
        existing.childCount++;
      } else {
        aggregated.set(key, {
          name: key,
          playbackCount: loc.playback_count,
          uniqueUsers: loc.unique_users,
          avgCompletion: loc.avg_completion,
          childCount: 1
        });
      }
    }

    // Sort by playback count and take top 15
    return Array.from(aggregated.values())
      .sort((a, b) => b.playbackCount - a.playbackCount)
      .slice(0, 15);
  }

  /**
   * Update breadcrumb navigation
   */
  private updateBreadcrumb(): void {
    const breadcrumb = document.getElementById('geo-breadcrumb');
    if (!breadcrumb) return;

    const { level, country, region } = this.currentState;
    let html = '';

    // Always show Countries
    html += `
      <button class="geo-breadcrumb-item ${level === 'country' ? 'active' : ''}"
              data-level="country"
              ${level === 'country' ? 'aria-current="page"' : ''}>
        Countries
      </button>
    `;

    // Show country if drilled down
    if ((level === 'region' || level === 'city') && country) {
      html += `
        <span class="geo-breadcrumb-separator" aria-hidden="true">&gt;</span>
        <button class="geo-breadcrumb-item ${level === 'region' ? 'active' : ''}"
                data-level="region"
                ${level === 'region' ? 'aria-current="page"' : ''}>
          ${this.escapeHtml(country)}
        </button>
      `;
    }

    // Show region if drilled to city
    if (level === 'city' && region) {
      html += `
        <span class="geo-breadcrumb-separator" aria-hidden="true">&gt;</span>
        <button class="geo-breadcrumb-item active"
                data-level="city"
                aria-current="page">
          ${this.escapeHtml(region)}
        </button>
      `;
    }

    breadcrumb.innerHTML = html;
  }

  /**
   * Render the chart using ECharts
   */
  private async renderChart(data: AggregatedGeoData[]): Promise<void> {
    const chartEl = document.getElementById('geo-drilldown-chart');
    if (!chartEl) return;

    // Lazy load ECharts
    const echarts = await import('echarts');

    // Dispose existing instance
    if (this.chartInstance) {
      (this.chartInstance as { dispose: () => void }).dispose();
    }

    if (data.length === 0) {
      chartEl.innerHTML = '<div class="geo-drilldown-empty">No geographic data available at this level</div>';
      return;
    }

    // Create chart instance
    this.chartInstance = echarts.init(chartEl);

    const canDrillDown = this.currentState.level !== 'city';
    const levelLabel = this.getLevelLabel();

    const option = {
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: (params: { name: string; value: number; dataIndex: number }[]) => {
          const item = data[params[0].dataIndex];
          return `
            <strong>${this.escapeHtml(item.name)}</strong><br/>
            Playbacks: ${item.playbackCount.toLocaleString()}<br/>
            Users: ${item.uniqueUsers.toLocaleString()}<br/>
            Avg Completion: ${item.avgCompletion.toFixed(1)}%
            ${canDrillDown ? '<br/><em>Click to drill down</em>' : ''}
          `;
        }
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        containLabel: true
      },
      xAxis: {
        type: 'value',
        name: 'Playbacks',
        nameLocation: 'middle',
        nameGap: 30,
        axisLabel: {
          formatter: (value: number) => value >= 1000 ? `${(value / 1000).toFixed(0)}K` : value
        }
      },
      yAxis: {
        type: 'category',
        data: data.map(d => d.name).reverse(),
        axisLabel: {
          width: 100,
          overflow: 'truncate',
          ellipsis: '...'
        }
      },
      series: [{
        name: levelLabel,
        type: 'bar',
        data: data.map(d => d.playbackCount).reverse(),
        itemStyle: {
          color: 'var(--highlight)',
          borderRadius: [0, 4, 4, 0]
        },
        emphasis: {
          itemStyle: {
            color: 'var(--highlight-hover)'
          }
        },
        cursor: canDrillDown ? 'pointer' : 'default'
      }]
    };

    (this.chartInstance as { setOption: (opt: unknown) => void }).setOption(option);

    // Handle click for drill-down
    if (canDrillDown) {
      (this.chartInstance as { on: (event: string, handler: (params: { dataIndex: number }) => void) => void }).on('click', (params: { dataIndex: number }) => {
        const clickedItem = data[data.length - 1 - params.dataIndex]; // Reverse index
        this.drillDown(clickedItem.name);
      });
    }

    // Handle resize - only attach once to avoid memory leak
    if (!this.resizeListenerAttached) {
      window.addEventListener('resize', this.boundHandleResize);
      this.resizeListenerAttached = true;
    }
  }

  /**
   * Get label for current level
   */
  private getLevelLabel(): string {
    switch (this.currentState.level) {
      case 'country':
        return 'Countries';
      case 'region':
        return 'Regions';
      case 'city':
        return 'Cities';
    }
  }

  /**
   * Update stats summary
   */
  private updateStats(data: AggregatedGeoData[]): void {
    const statsEl = document.getElementById('geo-drilldown-stats');
    if (!statsEl) return;

    const totalPlaybacks = data.reduce((sum, d) => sum + d.playbackCount, 0);
    const totalUsers = data.reduce((sum, d) => sum + d.uniqueUsers, 0);
    const avgCompletion = data.length > 0
      ? data.reduce((sum, d) => sum + d.avgCompletion, 0) / data.length
      : 0;

    statsEl.innerHTML = `
      <div class="geo-stat-item">
        <span class="geo-stat-value">${data.length}</span>
        <span class="geo-stat-label">${this.getLevelLabel()}</span>
      </div>
      <div class="geo-stat-item">
        <span class="geo-stat-value">${totalPlaybacks.toLocaleString()}</span>
        <span class="geo-stat-label">Total Playbacks</span>
      </div>
      <div class="geo-stat-item">
        <span class="geo-stat-value">${totalUsers.toLocaleString()}</span>
        <span class="geo-stat-label">Unique Users</span>
      </div>
      <div class="geo-stat-item">
        <span class="geo-stat-value">${avgCompletion.toFixed(1)}%</span>
        <span class="geo-stat-label">Avg Completion</span>
      </div>
    `;
  }

  /**
   * Drill down to next level
   */
  private drillDown(name: string): void {
    const { level } = this.currentState;

    if (level === 'country') {
      this.currentState = {
        level: 'region',
        country: name
      };
    } else if (level === 'region') {
      this.currentState = {
        level: 'city',
        country: this.currentState.country,
        region: name
      };
    }

    this.renderCurrentLevel();
  }

  /**
   * Navigate to a specific level
   */
  private navigateToLevel(level: DrillLevel): void {
    if (level === 'country') {
      this.currentState = { level: 'country' };
    } else if (level === 'region') {
      this.currentState = {
        level: 'region',
        country: this.currentState.country
      };
    }
    // City level stays as is

    this.renderCurrentLevel();
  }

  /**
   * Drill down by country name (can be called externally, e.g., from map click)
   */
  drillDownToCountry(country: string): void {
    this.currentState = {
      level: 'region',
      country: country
    };
    this.renderCurrentLevel();
  }

  /**
   * Reset to country level
   */
  reset(): void {
    this.currentState = { level: 'country' };
    this.renderCurrentLevel();
  }

  /**
   * Get current drill state
   */
  getCurrentState(): DrillDownState {
    return { ...this.currentState };
  }

  /**
   * Dispose chart resources and cleanup event listeners
   */
  dispose(): void {
    // Remove resize listener
    if (this.resizeListenerAttached) {
      window.removeEventListener('resize', this.boundHandleResize);
      this.resizeListenerAttached = false;
    }

    // Dispose chart instance
    if (this.chartInstance) {
      (this.chartInstance as { dispose: () => void }).dispose();
      this.chartInstance = null;
    }

    logger.debug('GeographicDrillDownManager disposed');
  }

  /**
   * Escape HTML to prevent XSS
   */
  private escapeHtml(unsafe: string): string {
    if (!unsafe) return '';
    return unsafe
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  /**
   * Cleanup resources to prevent memory leaks
   * Disposes chart and clears data
   */
  destroy(): void {
    this.dispose();
    this.locations = [];
    this.currentState = { level: 'country' };
  }
}

export default GeographicDrillDownManager;
