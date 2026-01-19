// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Chart Manager - Orchestrates all chart rendering using specialized renderers
 */

import * as echarts from 'echarts';
import type { API, LocationFilter } from '../api';
import { ALL_CHART_IDS, CHART_DESCRIPTIONS, getChartConfig } from './ChartRegistry';
import { createLogger } from '../logger';
import type { ChartRenderer, NavigatorWithDeviceMemory, EChartsChartOption, EChartsSeriesOption } from './types';

const logger = createLogger('ChartManager');

export class ChartManager {
  private charts: Map<string, echarts.ECharts> = new Map();
  private renderers: Map<string, ChartRenderer> = new Map();
  private pendingData: Map<string, unknown> = new Map();
  private observer: IntersectionObserver | null = null;
  private isMobileDevice: boolean;
  private isLowMemory: boolean;
  /** Debounce timer for resize events - prevents excessive redraws */
  private resizeDebounceTimer: number | null = null;
  /** Debounce delay in milliseconds (200ms is optimal for resize) */
  private static readonly RESIZE_DEBOUNCE_DELAY = 200;
  /** Bound resize handler reference for proper cleanup */
  private boundResizeHandler: () => void;
  /** Keyboard event listeners by container ID for cleanup on theme change */
  private keyboardListeners: Map<string, { keydown: (e: KeyboardEvent) => void; blur: () => void }> = new Map();

  constructor(private api: API) {
    // Detect device capabilities for optimal renderer selection
    this.isMobileDevice = /Android|iPhone|iPad|iPod/i.test(navigator.userAgent);
    this.isLowMemory = this.detectLowMemory();

    // Store bound resize handler reference for proper cleanup in destroy()
    this.boundResizeHandler = () => this.debouncedResizeCharts();

    this.setupLazyLoading();
    this.setupExportButtons();
    // Use debounced resize handler to prevent excessive redraws during window resize
    window.addEventListener('resize', this.boundResizeHandler);
  }

  // Detect low memory devices
  private detectLowMemory(): boolean {
    const nav = navigator as NavigatorWithDeviceMemory;
    if (nav.deviceMemory !== undefined) {
      return nav.deviceMemory < 4; // Less than 4GB RAM
    }
    // Fallback: assume low memory on mobile
    return /Android|iPhone|iPad|iPod/i.test(navigator.userAgent);
  }

  // Select optimal renderer based on device capabilities and data size
  // SVG: Better for mobile/low-memory, smaller datasets (<1000 points)
  // Canvas: Better for desktop with larger datasets (>1000 points)
  private getOptimalRenderer(dataSize: number = 0): 'canvas' | 'svg' {
    // Use SVG for mobile or low memory devices
    if (this.isMobileDevice || this.isLowMemory) {
      return 'svg';
    }

    // Use Canvas for desktop with larger datasets
    return dataSize > 1000 ? 'canvas' : 'svg';
  }

  /**
   * Setup accessibility attributes for chart container
   * Implements WCAG 2.1 SC 1.1.1 (Non-text Content) and SC 2.1.1 (Keyboard)
   */
  private setupChartAccessibility(chartId: string): void {
    const container = document.getElementById(chartId);
    if (!container) return;

    // Add role="img" for screen readers to treat chart as image
    container.setAttribute('role', 'img');

    // Add aria-label with chart description from registry
    const description = CHART_DESCRIPTIONS.get(chartId);
    if (description) {
      container.setAttribute('aria-label', description);
    }

    // Make chart keyboard focusable
    container.setAttribute('tabindex', '0');

    // Add data attribute to indicate keyboard support
    container.setAttribute('data-keyboard-enabled', 'true');

    // Set initial aria-busy state
    container.setAttribute('aria-busy', 'true');

    // Setup keyboard navigation
    this.setupKeyboardNavigation(chartId);
  }

  /**
   * Setup keyboard navigation for chart
   * Implements WCAG 2.1 SC 2.1.1 (Keyboard) with arrow key navigation
   *
   * Keyboard controls:
   * - Arrow Left/Right: Navigate between data points
   * - Home/End: Jump to first/last data point
   * - Enter: Announce current data point details
   *
   * FIX: Stores listener references to prevent accumulation on theme changes.
   * Old listeners are removed before adding new ones.
   */
  private setupKeyboardNavigation(chartId: string): void {
    const container = document.getElementById(chartId);
    if (!container) return;

    // Remove existing listeners for this container to prevent accumulation
    this.removeKeyboardListeners(chartId);

    let currentDataIndex = -1; // -1 = no selection

    const keydownHandler = (e: KeyboardEvent) => {
      const chart = this.charts.get(chartId);
      if (!chart) return;

      const option = chart.getOption() as EChartsChartOption;
      if (!option || !option.series || option.series.length === 0) return;

      // Get first series data for navigation
      const series = option.series[0];
      const dataLength = series.data?.length || 0;
      if (dataLength === 0) return;

      let needsUpdate = false;
      let announcement = '';

      switch (e.key) {
        case 'ArrowLeft':
        case 'ArrowUp':
          e.preventDefault();
          currentDataIndex = currentDataIndex <= 0 ? dataLength - 1 : currentDataIndex - 1;
          needsUpdate = true;
          announcement = this.formatDataPointAnnouncement(chartId, series, currentDataIndex);
          break;

        case 'ArrowRight':
        case 'ArrowDown':
          e.preventDefault();
          currentDataIndex = currentDataIndex >= dataLength - 1 ? 0 : currentDataIndex + 1;
          needsUpdate = true;
          announcement = this.formatDataPointAnnouncement(chartId, series, currentDataIndex);
          break;

        case 'Home':
          e.preventDefault();
          currentDataIndex = 0;
          needsUpdate = true;
          announcement = `First data point: ${this.formatDataPointAnnouncement(chartId, series, currentDataIndex)}`;
          break;

        case 'End':
          e.preventDefault();
          currentDataIndex = dataLength - 1;
          needsUpdate = true;
          announcement = `Last data point: ${this.formatDataPointAnnouncement(chartId, series, currentDataIndex)}`;
          break;

        case 'Enter':
          e.preventDefault();
          if (currentDataIndex >= 0) {
            announcement = `Current: ${this.formatDataPointAnnouncement(chartId, series, currentDataIndex)}`;
            this.announceToScreenReader(announcement);
          }
          return;
      }

      if (needsUpdate && currentDataIndex >= 0) {
        // Highlight selected data point
        chart.dispatchAction({
          type: 'highlight',
          seriesIndex: 0,
          dataIndex: currentDataIndex,
        });

        // Announce to screen reader
        this.announceToScreenReader(announcement);
      }
    };

    // Clear highlight when focus leaves chart
    const blurHandler = () => {
      const chart = this.charts.get(chartId);
      if (chart) {
        chart.dispatchAction({
          type: 'downplay',
          seriesIndex: 0,
        });
      }
      currentDataIndex = -1;
    };

    // Store references for cleanup
    this.keyboardListeners.set(chartId, { keydown: keydownHandler, blur: blurHandler });

    // Add the event listeners
    container.addEventListener('keydown', keydownHandler);
    container.addEventListener('blur', blurHandler);
  }

  /**
   * Remove keyboard listeners for a specific chart container.
   * Called before adding new listeners to prevent accumulation.
   */
  private removeKeyboardListeners(chartId: string): void {
    const listeners = this.keyboardListeners.get(chartId);
    if (!listeners) return;

    const container = document.getElementById(chartId);
    if (container) {
      container.removeEventListener('keydown', listeners.keydown);
      container.removeEventListener('blur', listeners.blur);
    }
    this.keyboardListeners.delete(chartId);
  }

  /**
   * Remove all keyboard listeners for all chart containers.
   * Called during destroy() for complete cleanup.
   */
  private removeAllKeyboardListeners(): void {
    this.keyboardListeners.forEach((listeners, chartId) => {
      const container = document.getElementById(chartId);
      if (container) {
        container.removeEventListener('keydown', listeners.keydown);
        container.removeEventListener('blur', listeners.blur);
      }
    });
    this.keyboardListeners.clear();
  }

  /**
   * Format data point announcement for screen readers
   */
  private formatDataPointAnnouncement(chartId: string, series: EChartsSeriesOption, dataIndex: number): string {
    const data = series.data?.[dataIndex];
    const xAxisData = this.getXAxisData(chartId);

    let label = '';
    let value = '';

    if (typeof data === 'object' && data !== null) {
      // Handle object data format (e.g., {name: 'Label', value: 123})
      label = data.name || xAxisData?.[dataIndex] || `Point ${dataIndex + 1}`;
      value = data.value !== undefined ? this.formatValue(data.value) : 'No data';
    } else {
      // Handle simple number format
      label = xAxisData?.[dataIndex] || `Point ${dataIndex + 1}`;
      value = this.formatValue(data);
    }

    return `${label}: ${value}`;
  }

  /**
   * Get x-axis data from chart option
   */
  private getXAxisData(chartId: string): string[] | undefined {
    const chart = this.charts.get(chartId);
    if (!chart) return undefined;

    const option = chart.getOption() as EChartsChartOption;
    return option?.xAxis?.[0]?.data;
  }

  /**
   * Format value for screen reader announcement
   */
  private formatValue(value: unknown): string {
    if (value === null || value === undefined) return 'No data';
    if (typeof value === 'number') {
      return value.toLocaleString();
    }
    return String(value);
  }

  /**
   * Announce message to screen reader via aria-live region
   */
  private announceToScreenReader(message: string): void {
    const announcer = document.getElementById('chart-announcer');
    if (!announcer) return;

    // Clear and update announcement
    announcer.textContent = '';
    setTimeout(() => {
      announcer.textContent = message;
    }, 100);
  }

  /**
   * Update chart aria-label with current data summary
   * Called after chart renders with data
   *
   * ENHANCEMENT (P2): This method enables dynamic ARIA descriptions like
   * "Playback trends showing increase of 20% this week". Currently, static
   * aria-labels from CHART_DESCRIPTIONS provide adequate accessibility.
   * To implement: call this method from chart render functions with computed
   * data summaries. See: docs/working/AUDIT_REMAINING.md issue 3.6 for
   * accessibility roadmap.
   *
   * @param chartId - Chart container ID
   * @param dataSummary - Optional data summary to append to base description
   */
  // @ts-ignore - Method reserved for future enhancement (see docstring)
  private updateChartAriaLabel(chartId: string, dataSummary?: string): void {
    const container = document.getElementById(chartId);
    if (!container) return;

    const baseDescription = CHART_DESCRIPTIONS.get(chartId) || 'Chart';
    const fullDescription = dataSummary
      ? `${baseDescription}. ${dataSummary}`
      : baseDescription;

    container.setAttribute('aria-label', fullDescription);
    container.setAttribute('aria-busy', 'false');
  }

  private setupLazyLoading(): void {
    if (typeof IntersectionObserver === 'undefined') {
      this.initializeCharts();
      return;
    }

    this.observer = new IntersectionObserver(
      (entries) => {
        // Batch chart initialization and rendering to prevent blocking main thread
        // when many charts become visible at once (e.g., Performance page with 16 charts)
        const chartsToRender: Array<{ id: string, data: unknown }> = [];

        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            const container = entry.target as HTMLElement;
            const chartId = container.id;

            if (!this.charts.has(chartId)) {
              // Setup accessibility attributes before initializing chart
              this.setupChartAccessibility(chartId);

              // Use optimal renderer based on device capabilities
              const renderer = this.getOptimalRenderer();
              const chart = echarts.init(container, 'dark', { renderer });
              this.charts.set(chartId, chart);

              if (this.pendingData.has(chartId)) {
                const data = this.pendingData.get(chartId);
                if (data) {
                  chartsToRender.push({ id: chartId, data });
                  this.pendingData.delete(chartId);
                }
              }
            }

            this.observer?.unobserve(container);
          }
        });

        // Render charts asynchronously to avoid blocking main thread
        if (chartsToRender.length > 0) {
          this.batchRenderCharts(chartsToRender);
        }
      },
      {
        root: null,
        rootMargin: '50px',
        threshold: 0.1,
      }
    );

    ALL_CHART_IDS.forEach((id) => {
      const container = document.getElementById(id);
      if (container && this.observer) {
        this.observer.observe(container);
      }
    });
  }

  /**
   * Force initialize charts that are currently visible.
   * Called when analytics view is shown to ensure charts render without waiting
   * for IntersectionObserver callbacks (which may be delayed in some browsers).
   * This is critical for E2E tests and immediate chart rendering.
   *
   * IMPORTANT: This method also renders any pending data for charts that were
   * initialized before their data was loaded. Without this, charts would remain
   * empty because the IntersectionObserver skips already-initialized charts.
   */
  public ensureChartsInitialized(): void {
    // Initialize all charts in visible containers
    this.initializeCharts();

    // Collect charts that need to render pending data
    const chartsToRender: Array<{ id: string, data: unknown }> = [];

    // Resize existing charts that might have been initialized while hidden
    // (charts initialized in hidden containers have 0 dimensions and no canvas)
    this.charts.forEach((chart, id) => {
      const container = document.getElementById(id);
      if (container) {
        // Ensure container is explicitly visible with 'important' priority
        // This fixes E2E visibility detection by ensuring inline styles override any CSS
        container.style.setProperty('visibility', 'visible', 'important');
        container.style.setProperty('opacity', '1', 'important');
        container.style.setProperty('display', 'block', 'important');

        // Check if container is visible and has dimensions
        const rect = container.getBoundingClientRect();
        if (rect.width > 0 && rect.height > 0) {
          // Force resize to create canvas if needed
          chart.resize();
          // Mark chart as ready (aria-busy=false) for accessibility and E2E tests
          container.setAttribute('aria-busy', 'false');

          // CRITICAL FIX: Render any pending data for this chart
          // This fixes a race condition where ensureChartsInitialized() runs before
          // the IntersectionObserver callback. Since we add charts to this.charts here,
          // the observer will skip them (it checks !this.charts.has(chartId)).
          // Without rendering pending data here, charts would remain empty.
          if (this.pendingData.has(id)) {
            const data = this.pendingData.get(id);
            if (data) {
              chartsToRender.push({ id, data });
              this.pendingData.delete(id);
            }
          }
        }
      }
    });

    // Render charts with pending data using batch rendering for performance
    if (chartsToRender.length > 0) {
      this.batchRenderCharts(chartsToRender);
    }
  }

  private initializeCharts(): void {
    ALL_CHART_IDS.forEach(id => {
      // Skip if already initialized
      if (this.charts.has(id)) {
        return;
      }

      const container = document.getElementById(id);
      if (container) {
        // Setup accessibility attributes before initializing chart
        this.setupChartAccessibility(id);

        // Use optimal renderer based on device capabilities
        const renderer = this.getOptimalRenderer();
        const chart = echarts.init(container, 'dark', { renderer });
        this.charts.set(id, chart);
      }
    });
  }

  private setupExportButtons(): void {
    document.querySelectorAll('.chart-export').forEach(button => {
      button.addEventListener('click', (e) => {
        const target = e.target as HTMLElement;
        const chartName = target.getAttribute('data-chart');
        if (chartName) {
          this.exportChart(chartName);
        }
      });
    });
  }

  async loadAllCharts(filter: LocationFilter = {}): Promise<void> {
    // Show loading skeletons for all charts
    ALL_CHART_IDS.forEach(id => {
      if (this.charts.has(id)) {
        this.showLoadingSkeleton(id);
      }
    });

    try {
      // Get comparison type from selector (default to 'week')
      const comparisonTypeSelector = document.getElementById('comparison-type-selector') as HTMLSelectElement;
      const comparisonType = (comparisonTypeSelector?.value as 'week' | 'month' | 'quarter' | 'year') || 'week';

      // Get temporal interval from selector (default to 'day')
      const temporalIntervalSelector = document.getElementById('temporal-interval-selector') as HTMLSelectElement;
      const temporalInterval = (temporalIntervalSelector?.value as 'hour' | 'day' | 'week' | 'month') || 'day';

      const [trendsData, geographicData, usersData, bingeData, watchPartiesData, popularData, bandwidthData, bitrateData, engagementData, comparativeData, temporalHeatmapData, resolutionMismatchData, hdrData, audioData, subtitleData, connectionSecurityData, pausePatternData, concurrentStreamsData, hardwareTranscodeData, abandonmentData] = await Promise.all([
        this.api.getAnalyticsTrends(filter),
        this.api.getAnalyticsGeographic(filter),
        this.api.getAnalyticsUsers(filter, 10),
        this.api.getAnalyticsBinge(filter),
        this.api.getAnalyticsWatchParties(filter),
        this.api.getAnalyticsPopular(filter, 10),
        this.api.getAnalyticsBandwidth(filter),
        this.api.getAnalyticsBitrate(filter),
        this.api.getAnalyticsUserEngagement(filter, 10),
        this.api.getAnalyticsComparative(filter, comparisonType),
        this.api.getTemporalHeatmap(filter, temporalInterval),
        this.api.getAnalyticsResolutionMismatch(filter),
        this.api.getAnalyticsHDR(filter),
        this.api.getAnalyticsAudio(filter),
        this.api.getAnalyticsSubtitles(filter),
        this.api.getAnalyticsConnectionSecurity(filter),
        this.api.getAnalyticsPausePatterns(filter),
        this.api.getAnalyticsConcurrentStreams(filter),
        this.api.getAnalyticsHardwareTranscode(filter),
        this.api.getAnalyticsAbandonment(filter),
      ]);

      const chartDataMap: Record<string, unknown> = {
        'chart-trends': trendsData,
        'chart-countries': geographicData,
        'chart-cities': geographicData,
        'chart-media': geographicData,
        'chart-users': usersData,
        'chart-heatmap': geographicData,
        'chart-platforms': geographicData,
        'chart-players': geographicData,
        'chart-completion': geographicData,
        'chart-transcode': geographicData,
        'chart-resolution': geographicData,
        'chart-codec': geographicData,
        'chart-libraries': geographicData,
        'chart-ratings': geographicData,
        'chart-duration': geographicData,
        'chart-years': geographicData,
        'chart-binge-summary': bingeData,
        'chart-binge-shows': bingeData,
        'chart-binge-users': bingeData,
        'chart-watch-parties-summary': watchPartiesData,
        'chart-watch-parties-content': watchPartiesData,
        'chart-watch-parties-users': watchPartiesData,
        'chart-popular-movies': popularData,
        'chart-popular-shows': popularData,
        'chart-popular-episodes': popularData,
        'chart-bandwidth-trends': bandwidthData,
        'chart-bandwidth-transcode': bandwidthData,
        'chart-bandwidth-resolution': bandwidthData,
        'chart-bandwidth-users': bandwidthData,
        'chart-bitrate-distribution': bitrateData,
        'chart-bitrate-utilization': bitrateData,
        'chart-bitrate-resolution': bitrateData,
        'chart-engagement-summary': engagementData,
        'chart-engagement-hours': engagementData,
        'chart-engagement-days': engagementData,
        'chart-comparative-metrics': comparativeData,
        'chart-comparative-content': comparativeData,
        'chart-comparative-users': comparativeData,
        'chart-temporal-heatmap': temporalHeatmapData,
        'chart-resolution-mismatch': resolutionMismatchData,
        'chart-hdr-analytics': hdrData,
        'chart-audio-analytics': audioData,
        'chart-subtitle-analytics': subtitleData,
        'chart-connection-security': connectionSecurityData,
        'chart-pause-patterns': pausePatternData,
        'chart-concurrent-streams': concurrentStreamsData,
        'chart-hardware-transcode': hardwareTranscodeData,
        'chart-abandonment': abandonmentData,
      };

      Object.entries(chartDataMap).forEach(([chartId, data]) => {
        if (this.charts.has(chartId)) {
          this.hideLoadingSkeleton(chartId);
          this.renderChartById(chartId, data);
        } else {
          this.pendingData.set(chartId, data);
        }
      });
    } catch (error) {
      logger.error('Failed to load charts:', error);
      // Hide skeletons and show error message on failure
      ALL_CHART_IDS.forEach(id => {
        if (this.charts.has(id)) {
          this.hideLoadingSkeleton(id);
          this.showEmptyState(id, 'Failed to load chart data. Please try again.');
        }
      });
    }
  }

  private initializeRenderer(chartId: string): void {
    const chart = this.charts.get(chartId);
    if (!chart) return;

    const chartConfig = getChartConfig(chartId);
    if (!chartConfig) return;

    const config = { chartId, chart };
    this.renderers.set(chartId, new chartConfig.renderer(config));
  }

  private renderChartById(chartId: string, data: unknown): void {
    if (!this.renderers.has(chartId)) {
      this.initializeRenderer(chartId);
    }

    const renderer = this.renderers.get(chartId);
    const chartConfig = getChartConfig(chartId);

    if (renderer && chartConfig) {
      renderer[chartConfig.renderMethod](data);
    }
  }

  /**
   * Show loading skeleton with meaningful text
   * Updates aria-busy for screen readers
   */
  private showLoadingSkeleton(containerId: string): void {
    const container = document.getElementById(containerId);
    if (!container) return;

    container.style.opacity = '0.5';
    container.style.pointerEvents = 'none';

    // Update aria-busy for accessibility
    container.setAttribute('aria-busy', 'true');

    // Get chart description for loading message
    const description = CHART_DESCRIPTIONS.get(containerId) || 'Chart';
    const chartType = description.split(' ')[0]; // Get first word (Line, Bar, Pie, etc.)

    // Show loading message on chart using ECharts
    const chart = this.charts.get(containerId);
    if (chart) {
      chart.setOption({
        title: {
          text: `Loading ${chartType} data...`,
          subtext: 'Please wait',
          left: 'center',
          top: 'center',
          textStyle: {
            color: '#a0a0a0',
            fontSize: 14,
            fontWeight: 'normal'
          },
          subtextStyle: {
            color: '#666',
            fontSize: 12
          }
        }
      }, { replaceMerge: ['title'] });
    }
  }

  /**
   * Hide loading skeleton
   * Updates aria-busy for screen readers
   */
  private hideLoadingSkeleton(containerId: string): void {
    const container = document.getElementById(containerId);
    if (!container) return;

    container.style.opacity = '1';
    container.style.pointerEvents = 'auto';

    // Update aria-busy for accessibility
    container.setAttribute('aria-busy', 'false');
  }

  private showEmptyState(containerId: string, message: string = 'No data available'): void {
    const chart = this.charts.get(containerId);
    if (!chart) return;

    chart.setOption({
      title: {
        text: message,
        left: 'center',
        top: 'center',
        textStyle: {
          color: '#888',
          fontSize: 14
        }
      }
    });
  }

  private exportChart(chartName: string, quality: 'low' | 'high' = 'high'): void {
    const chart = this.charts.get(chartName);
    if (!chart) return;

    // Export quality configuration
    // Low quality: 1x pixel ratio for smaller files (~200KB)
    // High quality: 3x pixel ratio for better quality (~800KB)
    const pixelRatio = quality === 'high' ? 3 : 1;

    const url = chart.getDataURL({
      type: 'png',
      pixelRatio: pixelRatio,
      backgroundColor: '#1a1a2e',
      excludeComponents: ['toolbox'] // Don't export toolbox controls
    });

    // Append link to DOM for reliable download triggering in all browsers (including headless)
    const link = document.createElement('a');
    link.href = url;
    link.download = `${chartName}-${quality}.png`;
    link.style.display = 'none';
    document.body.appendChild(link);
    link.click();

    // Clean up after a short delay to ensure download starts
    setTimeout(() => {
      if (link.parentNode) {
        document.body.removeChild(link);
      }
    }, 100);
  }

  // Public method to export chart with quality selection
  exportChartWithQuality(chartName: string, quality: 'low' | 'high' = 'high'): void {
    this.exportChart(chartName, quality);
  }

  /**
   * Batch render multiple charts asynchronously to prevent blocking main thread.
   * Uses requestAnimationFrame to yield control between chart renders,
   * allowing the browser to remain responsive during bulk chart rendering.
   * Critical for pages with many charts (e.g., Performance page with 16 charts).
   */
  private batchRenderCharts(chartsToRender: Array<{ id: string, data: unknown }>): void {
    let index = 0;

    const renderNext = () => {
      if (index >= chartsToRender.length) {
        return; // All charts rendered
      }

      const { id, data } = chartsToRender[index];
      this.renderChartById(id, data);
      index++;

      // Yield control to browser every 2 charts to keep UI responsive
      if (index % 2 === 0) {
        requestAnimationFrame(renderNext);
      } else {
        renderNext(); // Render immediately for better perceived performance
      }
    };

    // Start rendering
    requestAnimationFrame(renderNext);
  }

  private resizeCharts(): void {
    this.charts.forEach(chart => chart.resize());
  }

  /**
   * Debounced resize handler to prevent excessive chart redraws during window resize
   *
   * Problem: Without debouncing, rapidly resizing the window (e.g., dragging to resize)
   * triggers many resize events, each causing all charts to redraw. This can:
   * - Cause jank and poor performance
   * - Overwhelm the GPU/CPU especially on low-end devices
   * - Make the UI feel sluggish during resize
   *
   * Solution: Debounce resize events so that charts only redraw after the user
   * stops resizing for 200ms. This provides smooth resize UX while preventing
   * excessive redraws.
   *
   * @see /docs/working/UI_UX_AUDIT.md
   */
  private debouncedResizeCharts(): void {
    // Clear any pending resize timer
    if (this.resizeDebounceTimer !== null) {
      window.clearTimeout(this.resizeDebounceTimer);
    }

    // Set a new timer to execute resize after the debounce delay
    this.resizeDebounceTimer = window.setTimeout(() => {
      this.resizeCharts();
      this.resizeDebounceTimer = null;
    }, ChartManager.RESIZE_DEBOUNCE_DELAY);
  }

  updateFilter(filter: LocationFilter): void {
    this.loadAllCharts(filter);
  }

  /**
   * Update chart theme when user changes theme
   * Reinitializes charts with new theme and reloads data
   *
   * @param theme - 'dark' | 'light' | 'high-contrast'
   */
  updateTheme(theme: 'dark' | 'light' | 'high-contrast'): void {
    // Dispose existing charts
    this.charts.forEach(chart => chart.dispose());
    this.charts.clear();
    this.renderers.clear();

    // Reinitialize with new theme
    // ECharts theme is determined by the 'dark' parameter
    // We use 'dark' for dark and high-contrast modes, null for light
    const echartsTheme = theme === 'light' ? undefined : 'dark';

    ALL_CHART_IDS.forEach(id => {
      const container = document.getElementById(id);
      if (container) {
        this.setupChartAccessibility(id);
        const renderer = this.getOptimalRenderer();
        const chart = echarts.init(container, echartsTheme, { renderer });
        this.charts.set(id, chart);
      }
    });

    // Re-render with pending data if available
    this.pendingData.forEach((data, chartId) => {
      if (this.charts.has(chartId)) {
        this.renderChartById(chartId, data);
      }
    });
  }

  /**
   * Update charts when colorblind mode changes
   * Colors are read from CSS variables, so just need to re-render
   *
   * @param _enabled - Whether colorblind mode is enabled (CSS handles the colors)
   */
  updateColorblindMode(_enabled: boolean): void {
    // Colors come from CSS variables which are already updated
    // Just need to re-render charts with new colors
    this.pendingData.forEach((data, chartId) => {
      if (this.charts.has(chartId)) {
        this.renderChartById(chartId, data);
      }
    });
  }

  /**
   * Get a chart instance by ID
   * Used by DateRangeBrushManager to register brush event listeners
   *
   * @param chartId - The chart container ID
   * @returns The ECharts instance or undefined if not found
   */
  getChart(chartId: string): echarts.ECharts | undefined {
    return this.charts.get(chartId);
  }

  /**
   * Get all chart IDs
   * @returns Array of chart IDs
   */
  getChartIds(): string[] {
    return Array.from(this.charts.keys());
  }

  /**
   * Register a callback to be called when a chart is initialized
   * Useful for external managers that need to hook into chart creation
   *
   * @param chartId - The chart ID to watch
   * @param callback - Callback to call when chart is ready
   */
  onChartReady(chartId: string, callback: (chart: echarts.ECharts) => void): void {
    const existing = this.charts.get(chartId);
    if (existing) {
      callback(existing);
    } else {
      // Use MutationObserver to watch for chart initialization
      const container = document.getElementById(chartId);
      if (!container) return;

      const observer = new MutationObserver((_mutations, obs) => {
        const chart = this.charts.get(chartId);
        if (chart) {
          obs.disconnect();
          callback(chart);
        }
      });

      observer.observe(container, { childList: true, subtree: true });

      // Timeout cleanup
      setTimeout(() => observer.disconnect(), 30000);
    }
  }

  destroy(): void {
    // Remove window resize listener to prevent memory leak
    window.removeEventListener('resize', this.boundResizeHandler);

    // Clear any pending resize debounce timer
    if (this.resizeDebounceTimer !== null) {
      window.clearTimeout(this.resizeDebounceTimer);
      this.resizeDebounceTimer = null;
    }

    // Remove all keyboard listeners to prevent memory leak
    this.removeAllKeyboardListeners();

    if (this.observer) {
      this.observer.disconnect();
      this.observer = null;
    }
    this.charts.forEach(chart => chart.dispose());
    this.charts.clear();
    this.renderers.clear();
    this.pendingData.clear();
  }
}
