// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * DateRangeBrushManager - Date Range Brush Selection
 *
 * Enables drag-select date range filtering on timeline charts.
 * Users can click and drag on time series charts to select a date range,
 * which automatically applies as a filter.
 *
 * Features:
 * - Brush selection on trend charts
 * - Visual feedback during selection
 * - Automatic filter application
 * - Clear selection button
 * - Keyboard accessibility
 */

import * as echarts from 'echarts';

/**
 * Callback when a date range is selected via brush
 */
export type DateRangeSelectedCallback = (startDate: string, endDate: string) => void;

/**
 * Chart IDs that support brush selection (time series charts)
 */
const BRUSH_ENABLED_CHARTS = [
  'chart-trends',
  'chart-bandwidth-trends',
] as const;

type BrushEnabledChartId = typeof BRUSH_ENABLED_CHARTS[number];

/**
 * ECharts brush end event params
 */
interface BrushEndParams {
  areas?: Array<{
    coordRange?: [number, number];
    brushType?: string;
  }>;
}

/**
 * ECharts brush selected event params
 */
interface BrushSelectedParams {
  batch?: Array<{
    selected?: Array<{
      dataIndex?: number[];
    }>;
  }>;
}

/**
 * ECharts option fragment for brush configuration
 */
interface BrushConfigOption {
  toolbox: Record<string, unknown>;
  brush: Record<string, unknown>;
}

export class DateRangeBrushManager {
  private charts: Map<string, echarts.ECharts> = new Map();
  private onDateRangeSelected: DateRangeSelectedCallback | null = null;
  private currentSelection: { start: string; end: string } | null = null;
  private clearButton: HTMLElement | null = null;

  /**
   * Initialize the DateRangeBrushManager
   * @param onDateRangeSelected - Callback when date range is selected
   */
  constructor(onDateRangeSelected: DateRangeSelectedCallback) {
    this.onDateRangeSelected = onDateRangeSelected;
    this.createClearButton();
    this.setupKeyboardShortcuts();
  }

  /**
   * Register a chart for brush selection
   * @param chartId - The chart container ID
   * @param chart - The ECharts instance
   */
  registerChart(chartId: string, chart: echarts.ECharts): void {
    if (!BRUSH_ENABLED_CHARTS.includes(chartId as BrushEnabledChartId)) {
      return;
    }

    this.charts.set(chartId, chart);
    this.setupBrushListener(chartId, chart);
  }

  /**
   * Unregister a chart
   * @param chartId - The chart container ID
   */
  unregisterChart(chartId: string): void {
    const chart = this.charts.get(chartId);
    if (chart) {
      chart.off('brushEnd');
      chart.off('brushSelected');
      this.charts.delete(chartId);
    }
  }

  /**
   * Get brush configuration for ECharts
   * Call this from chart renderers to add brush capability
   */
  static getBrushConfig(): BrushConfigOption {
    return {
      toolbox: {
        feature: {
          brush: {
            type: ['lineX', 'clear'],
            title: {
              lineX: 'Select Date Range',
              clear: 'Clear Selection',
            },
          },
        },
        right: 60,
        top: 5,
        itemSize: 18,
        iconStyle: {
          borderColor: 'var(--text-secondary, #a0a0a0)',
        },
        emphasis: {
          iconStyle: {
            borderColor: 'var(--highlight, #7c3aed)',
          },
        },
      },
      brush: {
        xAxisIndex: 0,
        brushLink: 'all',
        brushType: 'lineX',
        brushStyle: {
          borderWidth: 1,
          color: 'var(--highlight-transparent, rgba(124, 58, 237, 0.2))',
          borderColor: 'var(--highlight, #7c3aed)',
        },
        outOfBrush: {
          colorAlpha: 0.3,
        },
        throttleType: 'debounce',
        throttleDelay: 300,
      },
    };
  }

  /**
   * Setup brush event listener for a chart
   */
  private setupBrushListener(chartId: string, chart: echarts.ECharts): void {
    // Listen for brush selection end
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    chart.on('brushEnd', ((params: BrushEndParams) => {
      this.handleBrushEnd(chartId, params);
    }) as any);

    // Listen for brush selected event (for visual feedback)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    chart.on('brushSelected', ((params: BrushSelectedParams) => {
      this.handleBrushSelected(chartId, params);
    }) as any);
  }

  /**
   * Handle brush selection completion
   */
  private handleBrushEnd(chartId: string, params: BrushEndParams): void {
    const chart = this.charts.get(chartId);
    if (!chart || !params.areas || params.areas.length === 0) {
      return;
    }

    const area = params.areas[0];
    if (!area || !area.coordRange || area.coordRange.length !== 2) {
      return;
    }

    // Get x-axis data (dates) from chart
    const option = chart.getOption();
    const xAxis = option?.xAxis as Array<{ data?: string[] }> | undefined;
    const xAxisData = xAxis?.[0]?.data;

    if (!xAxisData || xAxisData.length === 0) {
      return;
    }

    // coordRange contains [startIndex, endIndex]
    const [startIndex, endIndex] = area.coordRange;
    const clampedStart = Math.max(0, Math.floor(startIndex));
    const clampedEnd = Math.min(xAxisData.length - 1, Math.ceil(endIndex));

    const startDate = xAxisData[clampedStart];
    const endDate = xAxisData[clampedEnd];

    if (startDate && endDate) {
      this.currentSelection = { start: startDate, end: endDate };
      this.showClearButton();

      // Notify callback
      if (this.onDateRangeSelected) {
        // Convert to ISO format if needed
        const start = this.normalizeDate(startDate);
        const end = this.normalizeDate(endDate);
        this.onDateRangeSelected(start, end);
      }

      // Announce to screen readers
      this.announceSelection(startDate, endDate);
    }
  }

  /**
   * Handle brush selection (for visual feedback during drag)
   */
  private handleBrushSelected(_chartId: string, params: BrushSelectedParams): void {
    // Visual feedback is handled by ECharts brush component
    // We can add additional feedback here if needed
    if (params.batch && params.batch.length > 0) {
      const selected = params.batch[0].selected;
      if (selected && selected.length > 0) {
        const dataIndices = selected[0].dataIndex;
        if (dataIndices && dataIndices.length > 0) {
          // Show selection count as tooltip or status
          const count = dataIndices.length;
          this.updateSelectionStatus(`${count} days selected`);
        }
      }
    }
  }

  /**
   * Normalize date string to YYYY-MM-DD format
   */
  private normalizeDate(dateStr: string): string {
    // Handle various date formats (e.g., "Dec 13, 2025", "2025-12-13")
    const date = new Date(dateStr);
    if (isNaN(date.getTime())) {
      return dateStr;
    }
    return date.toISOString().split('T')[0];
  }

  /**
   * Create the clear selection button
   */
  private createClearButton(): void {
    // Check if button already exists
    let button = document.getElementById('brush-clear-button');
    if (!button) {
      button = document.createElement('button');
      button.id = 'brush-clear-button';
      button.className = 'brush-clear-button';
      button.setAttribute('type', 'button');
      button.setAttribute('aria-label', 'Clear date range selection');
      button.innerHTML = '<span class="brush-clear-icon" aria-hidden="true">&times;</span> Clear Selection';
      button.style.display = 'none';

      button.addEventListener('click', () => this.clearSelection());

      // Add to filter section
      const filterBadges = document.getElementById('filter-badges');
      if (filterBadges && filterBadges.parentElement) {
        filterBadges.parentElement.insertBefore(button, filterBadges.nextSibling);
      } else {
        // Fallback: add to filter controls
        const filterControls = document.querySelector('.filter-controls');
        if (filterControls) {
          filterControls.appendChild(button);
        }
      }
    }

    this.clearButton = button;
  }

  /**
   * Show the clear button
   */
  private showClearButton(): void {
    if (this.clearButton) {
      this.clearButton.style.display = 'inline-flex';
    }
  }

  /**
   * Hide the clear button
   */
  private hideClearButton(): void {
    if (this.clearButton) {
      this.clearButton.style.display = 'none';
    }
  }

  /**
   * Clear the current brush selection
   */
  clearSelection(): void {
    // Clear brush on all registered charts
    this.charts.forEach((chart) => {
      chart.dispatchAction({
        type: 'brush',
        command: 'clear',
        areas: [],
      });
    });

    this.currentSelection = null;
    this.hideClearButton();
    this.updateSelectionStatus('');
    this.announceSelection(null, null);
  }

  /**
   * Update selection status display
   */
  private updateSelectionStatus(message: string): void {
    let status = document.getElementById('brush-selection-status');
    if (!status && message) {
      status = document.createElement('span');
      status.id = 'brush-selection-status';
      status.className = 'brush-selection-status';
      status.setAttribute('aria-live', 'polite');

      // Add after clear button
      if (this.clearButton && this.clearButton.parentElement) {
        this.clearButton.parentElement.insertBefore(status, this.clearButton.nextSibling);
      }
    }

    if (status) {
      status.textContent = message;
      status.style.display = message ? 'inline' : 'none';
    }
  }

  /**
   * Announce selection to screen readers
   */
  private announceSelection(start: string | null, end: string | null): void {
    const announcer = document.getElementById('filter-announcer');
    if (!announcer) return;

    let message: string;
    if (start && end) {
      message = `Date range selected: ${start} to ${end}. Filters applied.`;
    } else {
      message = 'Date range selection cleared.';
    }

    announcer.textContent = '';
    setTimeout(() => {
      if (announcer) {
        announcer.textContent = message;
      }
    }, 100);
  }

  /**
   * Setup keyboard shortcuts
   */
  private setupKeyboardShortcuts(): void {
    document.addEventListener('keydown', (e: KeyboardEvent) => {
      // Escape clears selection when a brush-enabled chart is focused
      if (e.key === 'Escape' && this.currentSelection) {
        const activeElement = document.activeElement;
        if (activeElement && BRUSH_ENABLED_CHARTS.some(id => activeElement.id === id)) {
          e.preventDefault();
          this.clearSelection();
        }
      }
    });
  }

  /**
   * Get the current selection
   */
  getCurrentSelection(): { start: string; end: string } | null {
    return this.currentSelection;
  }

  /**
   * Check if brush is supported for a chart
   */
  static isBrushSupported(chartId: string): boolean {
    return BRUSH_ENABLED_CHARTS.includes(chartId as BrushEnabledChartId);
  }

  /**
   * Destroy the manager
   */
  destroy(): void {
    this.charts.forEach((_chart, chartId) => {
      this.unregisterChart(chartId);
    });
    this.charts.clear();

    if (this.clearButton) {
      this.clearButton.remove();
      this.clearButton = null;
    }

    const status = document.getElementById('brush-selection-status');
    if (status) {
      status.remove();
    }
  }
}
