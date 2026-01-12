// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * ChartDrillDownManager - Enables click-to-filter on charts
 *
 * Features:
 * - Click on chart data to filter by that value
 * - Supports user, media type, country, city, platform, player, library filters
 * - Visual feedback with cursor and tooltip hints
 * - Toast notification when filter is applied
 * - Undo button to clear drill-down filter
 */

import type * as echarts from 'echarts';
import { createLogger } from '../lib/logger';
import type { EChartsCallbackDataParams } from '../lib/charts/types';

const logger = createLogger('ChartDrillDownManager');

/**
 * Minimal interface for ToastManager to avoid circular imports
 */
interface ToastManagerInterface {
  info(message: string, title?: string, duration?: number): void;
  success(message: string, title?: string, duration?: number): void;
}

/**
 * Minimal interface for FilterManager to avoid circular imports
 */
interface FilterManagerInterface {
  onFilterChange(): void;
}

/**
 * ZRender mouse event params
 */
interface ZRenderMouseEventParams {
  offsetX: number;
  offsetY: number;
}

/**
 * Filter types that can be set via drill-down
 */
export type DrillDownFilterType = 'user' | 'media_type' | 'country' | 'city' | 'platform' | 'player' | 'library' | 'rating' | 'year';

/**
 * Mapping of chart IDs to their drill-down filter types
 */
const CHART_DRILLDOWN_MAP: Record<string, DrillDownFilterType> = {
  // User charts
  'chart-users': 'user',
  'chart-binge-users': 'user',
  'chart-watch-parties-users': 'user',
  'chart-bandwidth-users': 'user',
  'chart-engagement-summary': 'user',
  'chart-comparative-users': 'user',

  // Media type charts
  'chart-media': 'media_type',

  // Geographic charts
  'chart-countries': 'country',
  'chart-cities': 'city',

  // Platform/Player charts
  'chart-platforms': 'platform',
  'chart-players': 'player',

  // Library charts
  'chart-libraries': 'library',

  // Content charts
  'chart-ratings': 'rating',
  'chart-years': 'year',
};

/**
 * Callback type for when a drill-down filter is applied
 */
export type DrillDownCallback = (filterType: DrillDownFilterType, value: string) => void;

export class ChartDrillDownManager {
  private charts: Map<string, echarts.ECharts> = new Map();
  private onDrillDown: DrillDownCallback | null = null;
  private toastManager: ToastManagerInterface | null = null;
  private filterManager: FilterManagerInterface | null = null;
  private activeDrillDown: { filterType: DrillDownFilterType; value: string } | null = null;
  private undoButton: HTMLElement | null = null;

  constructor() {
    this.createUndoButton();
  }

  /**
   * Set the callback for drill-down events
   */
  setDrillDownCallback(callback: DrillDownCallback): void {
    this.onDrillDown = callback;
  }

  /**
   * Set toast manager for notifications
   */
  setToastManager(toastManager: ToastManagerInterface): void {
    this.toastManager = toastManager;
  }

  /**
   * Set filter manager for applying filters
   */
  setFilterManager(filterManager: FilterManagerInterface): void {
    this.filterManager = filterManager;
  }

  /**
   * Register a chart for drill-down support
   */
  registerChart(chartId: string, chart: echarts.ECharts): void {
    if (!CHART_DRILLDOWN_MAP[chartId]) {
      return; // Chart doesn't support drill-down
    }

    this.charts.set(chartId, chart);
    this.setupChartClickHandler(chartId, chart);
    this.addDrillDownIndicator(chartId);
  }

  /**
   * Set up click handler for chart
   */
  private setupChartClickHandler(chartId: string, chart: echarts.ECharts): void {
    const filterType = CHART_DRILLDOWN_MAP[chartId];
    if (!filterType) return;

    // Remove any existing click handler
    chart.off('click');

    // Add new click handler
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    chart.on('click', ((params: EChartsCallbackDataParams) => {
      const value = this.extractValueFromParams(params, filterType);
      if (value) {
        this.applyDrillDownFilter(filterType, value);
      }
    }) as any);

    // Add cursor pointer on hover over data
    chart.getZr().on('mousemove', (params: ZRenderMouseEventParams) => {
      const container = document.getElementById(chartId);
      if (!container) return;

      // Check if mouse is over a data element
      const pointInPixel = [params.offsetX, params.offsetY];
      const isOverData = chart.containPixel('grid', pointInPixel) ||
                         chart.containPixel('series', pointInPixel);

      container.style.cursor = isOverData ? 'pointer' : 'default';
    });

    logger.debug(`Drill-down enabled for chart: ${chartId} (filter: ${filterType})`);
  }

  /**
   * Extract value from click event params based on filter type
   */
  private extractValueFromParams(params: EChartsCallbackDataParams, filterType: DrillDownFilterType): string | null {
    // Handle different data formats
    if (params.name) {
      return params.name;
    }

    if (params.value !== undefined && params.value !== null) {
      // Object format with name property
      const data = params.value as Record<string, unknown>;
      if (typeof data === 'object' && data !== null && 'name' in data && typeof data.name === 'string') {
        return data.name;
      }
      // Direct value
      if (typeof params.value === 'string') {
        return params.value;
      }
    }

    // Try to get from seriesName for pie charts
    if (params.seriesName && filterType === 'media_type') {
      return params.seriesName;
    }

    return null;
  }

  /**
   * Apply drill-down filter and reload data
   */
  private applyDrillDownFilter(filterType: DrillDownFilterType, value: string): void {
    // Store active drill-down
    this.activeDrillDown = { filterType, value };

    // Apply filter based on type
    this.setFilterValue(filterType, value);

    // Show undo button
    this.showUndoButton(filterType, value);

    // Trigger callback
    if (this.onDrillDown) {
      this.onDrillDown(filterType, value);
    }

    // Show toast notification
    if (this.toastManager) {
      const filterLabel = this.getFilterLabel(filterType);
      this.toastManager.info(
        `Filtered by ${filterLabel}: ${value}. Click "Clear" to reset.`,
        'Drill-Down Applied',
        5000
      );
    }

    // Trigger filter change to reload data
    if (this.filterManager) {
      this.filterManager.onFilterChange();
    }

    logger.debug(`Drill-down applied: ${filterType} = ${value}`);
  }

  /**
   * Set filter value in UI
   */
  private setFilterValue(filterType: DrillDownFilterType, value: string): void {
    switch (filterType) {
      case 'user': {
        const filterUsers = document.getElementById('filter-users') as HTMLSelectElement;
        if (filterUsers) {
          // Check if value exists in options
          const optionExists = Array.from(filterUsers.options).some(opt => opt.value === value);
          if (optionExists) {
            filterUsers.value = value;
          } else {
            // Add option if it doesn't exist
            const option = document.createElement('option');
            option.value = value;
            option.textContent = value;
            filterUsers.appendChild(option);
            filterUsers.value = value;
          }
        }
        break;
      }
      case 'media_type': {
        const filterMediaTypes = document.getElementById('filter-media-types') as HTMLSelectElement;
        if (filterMediaTypes) {
          const optionExists = Array.from(filterMediaTypes.options).some(opt => opt.value === value);
          if (optionExists) {
            filterMediaTypes.value = value;
          }
        }
        break;
      }
      // For other filter types that may not have dedicated dropdowns,
      // we could use URL parameters or a custom filter system
      default:
        // Store in URL or custom state for filters without dedicated UI
        const params = new URLSearchParams(window.location.search);
        params.set(`filter_${filterType}`, value);
        const newURL = `${window.location.pathname}?${params}`;
        window.history.replaceState({}, '', newURL);
        break;
    }
  }

  /**
   * Get human-readable label for filter type
   */
  private getFilterLabel(filterType: DrillDownFilterType): string {
    const labels: Record<DrillDownFilterType, string> = {
      user: 'User',
      media_type: 'Media Type',
      country: 'Country',
      city: 'City',
      platform: 'Platform',
      player: 'Player',
      library: 'Library',
      rating: 'Rating',
      year: 'Year'
    };
    return labels[filterType] || filterType;
  }

  /**
   * Add visual indicator that chart supports drill-down
   */
  private addDrillDownIndicator(chartId: string): void {
    const container = document.getElementById(chartId);
    if (!container) return;

    // Add data attribute for CSS styling
    container.setAttribute('data-drilldown', 'true');

    // Find or create chart wrapper
    const wrapper = container.closest('.chart-container') || container.parentElement;
    if (!wrapper) return;

    // Check if indicator already exists
    if (wrapper.querySelector('.drill-down-hint')) return;

    // Add hint element
    const hint = document.createElement('div');
    hint.className = 'drill-down-hint';
    hint.innerHTML = '<span class="drill-down-icon">&#x1F50D;</span> Click to filter';
    hint.setAttribute('aria-hidden', 'true');
    wrapper.appendChild(hint);
  }

  /**
   * Create undo button for clearing drill-down
   */
  private createUndoButton(): void {
    // Check if button already exists
    if (document.getElementById('drill-down-undo')) return;

    const button = document.createElement('button');
    button.id = 'drill-down-undo';
    button.className = 'drill-down-undo';
    button.innerHTML = `
      <span class="drill-down-undo-icon">&times;</span>
      <span class="drill-down-undo-text"></span>
    `;
    button.setAttribute('aria-label', 'Clear drill-down filter');
    button.style.display = 'none';

    button.addEventListener('click', () => this.clearDrillDown());

    // Add to document
    document.body.appendChild(button);
    this.undoButton = button;
  }

  /**
   * Show undo button with current filter info
   */
  private showUndoButton(_filterType: DrillDownFilterType, value: string): void {
    if (!this.undoButton) return;

    const textEl = this.undoButton.querySelector('.drill-down-undo-text');
    if (textEl) {
      textEl.textContent = `Clear: ${value}`;
    }

    this.undoButton.style.display = 'flex';
    this.undoButton.setAttribute('aria-label', `Clear drill-down filter: ${value}`);
  }

  /**
   * Clear drill-down filter and reset
   */
  clearDrillDown(): void {
    if (!this.activeDrillDown) return;

    const { filterType, value } = this.activeDrillDown;

    // Reset filter based on type
    switch (filterType) {
      case 'user': {
        const filterUsers = document.getElementById('filter-users') as HTMLSelectElement;
        if (filterUsers) filterUsers.value = '';
        break;
      }
      case 'media_type': {
        const filterMediaTypes = document.getElementById('filter-media-types') as HTMLSelectElement;
        if (filterMediaTypes) filterMediaTypes.value = '';
        break;
      }
      default: {
        // Remove from URL
        const params = new URLSearchParams(window.location.search);
        params.delete(`filter_${filterType}`);
        const newURL = params.toString()
          ? `${window.location.pathname}?${params}`
          : window.location.pathname;
        window.history.replaceState({}, '', newURL);
        break;
      }
    }

    // Hide undo button
    if (this.undoButton) {
      this.undoButton.style.display = 'none';
    }

    // Clear active drill-down
    this.activeDrillDown = null;

    // Show toast
    if (this.toastManager) {
      this.toastManager.success(`Cleared filter: ${value}`, 'Filter Cleared', 3000);
    }

    // Trigger filter change
    if (this.filterManager) {
      this.filterManager.onFilterChange();
    }

    logger.debug(`Drill-down cleared: ${filterType} = ${value}`);
  }

  /**
   * Initialize drill-down for all registered charts
   */
  init(): void {
    logger.debug('ChartDrillDownManager initialized');
  }

  /**
   * Check if a chart supports drill-down
   */
  supportsDrillDown(chartId: string): boolean {
    return chartId in CHART_DRILLDOWN_MAP;
  }

  /**
   * Get the filter type for a chart
   */
  getFilterType(chartId: string): DrillDownFilterType | null {
    return CHART_DRILLDOWN_MAP[chartId] || null;
  }

  /**
   * Destroy and cleanup
   */
  destroy(): void {
    this.charts.forEach((chart) => {
      chart.off('click');
    });
    this.charts.clear();

    if (this.undoButton && this.undoButton.parentNode) {
      this.undoButton.parentNode.removeChild(this.undoButton);
    }
  }
}

export default ChartDrillDownManager;
