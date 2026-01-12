// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Renderer for playback trends charts
 */

import type { TrendsResponse } from '../../api';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS } from '../config/colors';
import { getGridConfig } from '../config/grid';
import { DateRangeBrushManager } from '../../../app/DateRangeBrushManager';
import { trendPredictionManager } from '../../../app/TrendPredictionManager';
import type { EChartsCallbackDataParams } from '../types';

export class TrendsChartRenderer extends BaseChartRenderer {
  render(data: TrendsResponse): void {
    // Defensive null check to prevent "Cannot read properties of null" error
    if (!this.hasData(data?.playback_trends)) {
      this.showEmptyState('No trend data available');
      return;
    }

    const dates = data.playback_trends.map(t => t.date);
    const playbacks = data.playback_trends.map(t => t.playback_count);
    const users = data.playback_trends.map(t => t.unique_users);

    const enableSampling = this.helpers.shouldEnableSampling(data.playback_trends.length);

    // Enable dataZoom for large datasets (more than 7 days of data)
    const enableDataZoom = dates.length > 7;

    // Enable brush for date range selection
    const brushConfig = DateRangeBrushManager.getBrushConfig();

    // Get extended dates including forecast period
    const extendedDates = trendPredictionManager.getExtendedDates(dates);

    // Get prediction series for playbacks and users
    const predictionSeries = [
      ...trendPredictionManager.getPredictionSeries(playbacks, dates, 'Playbacks', CHART_COLORS.primary),
      ...trendPredictionManager.getPredictionSeries(users, dates, 'Users', CHART_COLORS.secondary),
    ];

    // Build legend data including forecast series if enabled
    const legendData = ['Playbacks', 'Unique Users'];
    if (trendPredictionManager.isEnabled()) {
      legendData.push('Playbacks Forecast', 'Users Forecast');
    }

    const option = {
      ...this.getBaseOption(),
      ...brushConfig,
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: this.createTooltipFormatter(),
      }),
      legend: this.getLegend({
        data: legendData,
        top: 0,
      }),
      // Use withDataZoom grid when zoom is enabled for extra bottom space
      grid: getGridConfig(enableDataZoom ? 'withDataZoom' : 'default'),
      xAxis: this.getCategoryAxis(extendedDates, { boundaryGap: false }),
      yAxis: [
        this.getValueAxis({ name: 'Playbacks' }),
        this.getValueAxis({ name: 'Users', splitLine: { show: false } }),
      ],
      // Add dataZoom for pan/zoom capability on time series
      dataZoom: enableDataZoom ? this.getDataZoom('both') : undefined,
      series: [
        this.createPlaybacksSeries(playbacks, enableSampling),
        this.createUsersSeries(users, enableSampling),
        ...predictionSeries,
      ],
    };

    this.setOption(option);
  }

  private createTooltipFormatter() {
    return (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
      if (!Array.isArray(params)) return '';
      const date = params[0].axisValue;
      let html = `<div style="font-weight: 600; margin-bottom: 8px;">${date}</div>`;
      params.forEach((param: EChartsCallbackDataParams) => {
        const value = this.formatters.formatNumber(typeof param.value === 'number' ? param.value : 0);
        html += `<div style="margin: 4px 0;">
          <span style="display: inline-block; width: 10px; height: 10px; border-radius: 50%; background: ${param.color}; margin-right: 8px;"></span>
          <span style="font-weight: 500;">${param.seriesName}:</span>
          <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${value}</span>
        </div>`;
      });
      return html;
    };
  }

  private createPlaybacksSeries(data: number[], enableSampling: boolean) {
    const dataSize = data.length;

    // Enable progressive rendering for large datasets (M4)
    // This addresses Medium Priority Issue M4 from the production audit
    // Renders 5000 points per frame for datasets > 100k points
    const progressive = dataSize > 100000 ? 5000 : 0;
    const progressiveThreshold = dataSize > 100000 ? 10000 : 0;

    return {
      name: 'Playbacks',
      type: 'line' as const,
      smooth: true,
      data,
      sampling: enableSampling ? 'lttb' as const : undefined,
      showSymbol: !enableSampling,
      areaStyle: this.helpers.getAreaStyleGradient(CHART_COLORS.primary),
      lineStyle: { color: CHART_COLORS.primary, width: 2 },
      itemStyle: { color: CHART_COLORS.primary },
      large: dataSize > 10000, // Enable large mode for >10k points
      progressive: progressive,
      progressiveThreshold: progressiveThreshold,
    };
  }

  private createUsersSeries(data: number[], enableSampling: boolean) {
    const dataSize = data.length;

    // Enable progressive rendering for large datasets (M4)
    const progressive = dataSize > 100000 ? 5000 : 0;
    const progressiveThreshold = dataSize > 100000 ? 10000 : 0;

    return {
      name: 'Unique Users',
      type: 'line' as const,
      smooth: true,
      yAxisIndex: 1,
      data,
      sampling: enableSampling ? 'lttb' as const : undefined,
      showSymbol: !enableSampling,
      lineStyle: { color: CHART_COLORS.secondary, width: 2 },
      itemStyle: { color: CHART_COLORS.secondary },
      large: dataSize > 10000, // Enable large mode for >10k points
      progressive: progressive,
      progressiveThreshold: progressiveThreshold,
    };
  }
}
