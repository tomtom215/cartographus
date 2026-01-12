// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Base abstract class for all chart renderers
 * Provides common functionality and enforces consistent patterns
 */

import * as echarts from 'echarts';
import type { EChartsOption } from 'echarts';
import { getEnhancedTooltip } from '../config/tooltips';
import { getEnhancedLegend, getAccessibleLegend, getAccessibleIcon } from '../config/legends';
import { STYLE_CONSTANTS } from '../config/colors';
import { ChartFormatters } from '../utils/formatters';
import { ChartHelpers } from '../utils/chartHelpers';
import type { ChartRendererConfig, AxisConfigExtension, DataZoomConfig } from '../types';

export abstract class BaseChartRenderer {
  protected chartId: string;
  protected chart: echarts.ECharts;
  protected formatters = ChartFormatters;
  protected helpers = ChartHelpers;

  constructor(config: ChartRendererConfig) {
    this.chartId = config.chartId;
    this.chart = config.chart;
  }

  /**
   * Abstract render method - must be implemented by subclasses
   * Each subclass specifies its own data type
   */
  abstract render(data: unknown): void;

  /**
   * Set chart option with optional merge control
   */
  protected setOption(option: EChartsOption, opts?: { notMerge?: boolean }): void {
    this.chart.setOption(option, opts);
  }

  /**
   * Get base chart option with common settings
   */
  protected getBaseOption(): EChartsOption {
    return {
      backgroundColor: STYLE_CONSTANTS.backgroundColor,
    };
  }

  /**
   * Get enhanced tooltip configuration
   */
  protected getTooltip(config?: Parameters<typeof getEnhancedTooltip>[0]) {
    return getEnhancedTooltip(config);
  }

  /**
   * Get enhanced legend configuration
   */
  protected getLegend(config?: Parameters<typeof getEnhancedLegend>[0]) {
    return getEnhancedLegend(config);
  }

  /**
   * Get accessible legend with different shapes per series
   * WCAG SC 1.4.1 - Use of Color: Information conveyed by shape, not just color
   *
   * @param seriesNames - Array of series names for the legend
   * @param config - Additional legend configuration
   */
  protected getAccessibleLegend(seriesNames: string[], config?: Parameters<typeof getEnhancedLegend>[0]) {
    return getAccessibleLegend(seriesNames.length, {
      ...config,
      data: seriesNames,
    });
  }

  /**
   * Get accessible icon for a specific series index
   * Returns a unique shape for each series to distinguish beyond color alone
   */
  protected getSeriesIcon(seriesIndex: number): string {
    return getAccessibleIcon(seriesIndex);
  }

  /**
   * Get common axis styling
   */
  protected getAxisStyle() {
    return {
      axisLine: { lineStyle: { color: STYLE_CONSTANTS.axisLineColor } },
      axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
      splitLine: { lineStyle: { color: STYLE_CONSTANTS.splitLineColor } },
    };
  }

  /**
   * Get category axis configuration
   * Note: Returns 'any' to avoid ECharts strict type conflicts with boundaryGap
   * (category axis uses boolean, value axis uses tuple)
   */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  protected getCategoryAxis(data: string[], config?: AxisConfigExtension): any {
    return {
      type: 'category' as const,
      data,
      ...this.getAxisStyle(),
      ...config,
    };
  }

  /**
   * Get value axis configuration
   * Note: Returns 'any' to avoid ECharts strict type conflicts with boundaryGap
   * (category axis uses boolean, value axis uses tuple)
   */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  protected getValueAxis(config?: AxisConfigExtension): any {
    return {
      type: 'value' as const,
      ...this.getAxisStyle(),
      ...config,
    };
  }

  /**
   * Get dataZoom configuration for time series charts
   * Enables zoom and pan functionality for exploring data
   *
   * @param type - 'slider' shows a slider bar, 'inside' enables scroll/drag zoom
   * @param xAxisIndex - Index of the x-axis to control (defaults to 0)
   * @returns Array of dataZoom configurations
   */
  protected getDataZoom(type: 'slider' | 'inside' | 'both' = 'both', xAxisIndex: number = 0): DataZoomConfig[] {
    const sliderConfig = {
      type: 'slider' as const,
      xAxisIndex,
      start: 0,
      end: 100,
      height: 20,
      bottom: 5,
      borderColor: 'var(--border, #2a2a3e)',
      backgroundColor: 'var(--accent, rgba(15, 52, 96, 0.3))',
      fillerColor: 'var(--highlight, rgba(124, 58, 237, 0.3))',
      handleStyle: {
        color: 'var(--highlight, #7c3aed)',
        borderColor: 'var(--highlight, #7c3aed)',
      },
      textStyle: {
        color: 'var(--text-secondary, #a0a0a0)',
        fontSize: 11,
      },
      dataBackground: {
        lineStyle: { color: 'var(--text-secondary, #a0a0a0)', opacity: 0.3 },
        areaStyle: { color: 'var(--accent, #0f3460)', opacity: 0.2 },
      },
      selectedDataBackground: {
        lineStyle: { color: 'var(--highlight, #7c3aed)', opacity: 0.6 },
        areaStyle: { color: 'var(--highlight, #7c3aed)', opacity: 0.2 },
      },
    };

    const insideConfig = {
      type: 'inside' as const,
      xAxisIndex,
      start: 0,
      end: 100,
      zoomOnMouseWheel: true,
      moveOnMouseMove: true,
      preventDefaultMouseMove: false,
    };

    switch (type) {
      case 'slider':
        return [sliderConfig];
      case 'inside':
        return [insideConfig];
      case 'both':
      default:
        return [sliderConfig, insideConfig];
    }
  }

  /**
   * Show empty state when no data is available
   * @param message - Optional custom message
   */
  protected showEmptyState(message: string = 'No data available'): void {
    this.setOption({
      ...this.getBaseOption(),
      title: {
        text: message,
        left: 'center',
        top: 'center',
        textStyle: { color: '#888', fontSize: 14 }
      }
    });
  }

  /**
   * Check if an array property is valid (not null/undefined and has items)
   * Use this to prevent "Cannot read properties of null (reading 'map')" errors
   */
  protected hasData<T>(array: T[] | null | undefined): array is T[] {
    return Array.isArray(array) && array.length > 0;
  }
}
