// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Renderer for bitrate & bandwidth analytics charts (v1.42 - Phase 2.2)
 * Tracks bitrate at 3 levels (source, transcode, network) for network bottleneck identification
 */

import type { BitrateAnalyticsResponse } from '../../api';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS, STYLE_CONSTANTS } from '../config/colors';
import type { EChartsCallbackDataParams } from '../types';

export class BitrateChartRenderer extends BaseChartRenderer {
  render(_data: BitrateAnalyticsResponse): void {
    // Base render method - unused in this renderer
  }

  /**
   * Renders bitrate distribution histogram
   * Shows distribution of source/transcode bitrates across all sessions
   */
  renderDistribution(data: BitrateAnalyticsResponse): void {
    if (!this.hasData(data?.bitrate_timeseries)) {
      this.showEmptyState('No bitrate distribution data available');
      return;
    }
    // Create histogram buckets from time series data
    const bitrates = data.bitrate_timeseries.map(t => t.avg_bitrate);
    const buckets = this.createHistogramBuckets(bitrates, 10);

    const option = {
      ...this.getBaseOption(),
      title: {
        text: `Bitrate Distribution | Median: ${data.median_bitrate.toLocaleString()} Kbps | Peak: ${data.peak_bitrate.toLocaleString()} Kbps`,
        left: 'center',
        top: 10,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 14 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const bucket = params[0];
          return `<div style="font-weight: 600; margin-bottom: 8px;">Bitrate Range</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Range:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${bucket.name}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Sessions:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${bucket.value}</span>
            </div>`;
        }
      }),
      grid: { left: '10%', right: '10%', bottom: '15%', top: '20%' },
      xAxis: this.getCategoryAxis(buckets.map(b => b.label), {
        axisLabel: { rotate: 45, interval: 0, fontSize: 10 }
      }),
      yAxis: this.getValueAxis({ name: 'Session Count' }),
      series: [{
        type: 'bar' as const,
        data: buckets.map(b => b.count),
        itemStyle: {
          color: CHART_COLORS.primary,
          borderRadius: this.helpers.getBarBorderRadius('vertical')
        },
        emphasis: {
          itemStyle: { color: CHART_COLORS.secondary }
        }
      }]
    };

    this.setOption(option);
  }

  /**
   * Renders bandwidth utilization area chart
   * Shows bandwidth utilization percentage over 30-day rolling window
   */
  renderUtilization(data: BitrateAnalyticsResponse): void {
    if (!this.hasData(data?.bitrate_timeseries)) {
      this.showEmptyState('No bandwidth utilization data available');
      return;
    }
    const timestamps = data.bitrate_timeseries.map(t => {
      // Format timestamp for display (remove time portion)
      return t.timestamp.split('T')[0];
    });
    const avgBitrates = data.bitrate_timeseries.map(t => t.avg_bitrate);
    const peakBitrates = data.bitrate_timeseries.map(t => t.peak_bitrate);

    const option = {
      ...this.getBaseOption(),
      title: {
        text: `Bandwidth Utilization | Avg: ${data.bandwidth_utilization.toFixed(1)}% | Constrained Sessions: ${data.constrained_sessions}`,
        left: 'center',
        top: 10,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 14 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params)) return '';
          const date = params[0].axisValue;
          let html = `<div style="font-weight: 600; margin-bottom: 8px;">${date}</div>`;
          params.forEach((param: EChartsCallbackDataParams) => {
            const value = typeof param.value === 'number' ? param.value : 0;
            html += `<div style="margin: 4px 0;">
              <span style="display: inline-block; width: 10px; height: 10px; border-radius: 50%; background: ${param.color}; margin-right: 8px;"></span>
              <span style="font-weight: 500;">${param.seriesName}:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${value.toLocaleString()} Kbps</span>
            </div>`;
          });
          return html;
        }
      }),
      legend: this.getLegend({ data: ['Avg Bitrate', 'Peak Bitrate'], top: '10%' }),
      grid: { left: '10%', right: '10%', bottom: '15%', top: '25%' },
      xAxis: this.getCategoryAxis(timestamps, {
        axisLabel: { rotate: 45, interval: Math.floor(timestamps.length / 10) }
      }),
      yAxis: this.getValueAxis({ name: 'Bitrate (Kbps)' }),
      series: [
        {
          name: 'Avg Bitrate',
          type: 'line' as const,
          data: avgBitrates,
          smooth: true,
          areaStyle: {
            color: {
              type: 'linear' as const,
              x: 0, y: 0, x2: 0, y2: 1,
              colorStops: [
                { offset: 0, color: `${CHART_COLORS.primary}80` },
                { offset: 1, color: `${CHART_COLORS.primary}10` }
              ]
            }
          },
          lineStyle: { width: 2, color: CHART_COLORS.primary }
        },
        {
          name: 'Peak Bitrate',
          type: 'line' as const,
          data: peakBitrates,
          smooth: true,
          lineStyle: { width: 2, color: CHART_COLORS.warning, type: 'dashed' as const }
        }
      ]
    };

    this.setOption(option);
  }

  /**
   * Renders bitrate by resolution bar chart
   * Compares average bitrate across resolution tiers (4K, 1080p, 720p, SD)
   */
  renderByResolution(data: BitrateAnalyticsResponse): void {
    if (!this.hasData(data?.bitrate_by_resolution)) {
      this.showEmptyState('No bitrate by resolution data available');
      return;
    }
    const resolutions = data.bitrate_by_resolution.map(r => r.resolution);
    const avgBitrates = data.bitrate_by_resolution.map(r => r.avg_bitrate);
    const transcodeRates = data.bitrate_by_resolution.map(r => r.transcode_rate);

    // Color code bars by resolution quality
    const colors = resolutions.map(res => {
      if (res === '4K') return CHART_COLORS.danger;
      if (res === '1080p') return CHART_COLORS.primary;
      if (res === '720p') return CHART_COLORS.warning;
      if (res === 'SD') return CHART_COLORS.success;
      return CHART_COLORS.dark;
    });

    const option = {
      ...this.getBaseOption(),
      title: {
        text: `Bitrate by Resolution | Source Avg: ${data.avg_source_bitrate.toLocaleString()} Kbps | Transcode Avg: ${data.avg_transcode_bitrate.toLocaleString()} Kbps`,
        left: 'center',
        top: 10,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 14 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const index = params[0].dataIndex;
          const item = data.bitrate_by_resolution[index];
          return `<div style="font-weight: 600; margin-bottom: 8px;">${item.resolution}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Avg Bitrate:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${item.avg_bitrate.toLocaleString()} Kbps</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Sessions:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${item.session_count.toLocaleString()}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Transcode Rate:</span>
              <span style="color: ${CHART_COLORS.warning}; font-weight: 600;">${item.transcode_rate.toFixed(1)}%</span>
            </div>`;
        }
      }),
      grid: { left: '10%', right: '10%', bottom: '10%', top: '20%' },
      xAxis: this.getCategoryAxis(resolutions),
      yAxis: [
        this.getValueAxis({ name: 'Avg Bitrate (Kbps)' }),
        this.getValueAxis({ name: 'Transcode Rate (%)', max: 100, splitLine: { show: false } })
      ],
      series: [
        {
          name: 'Avg Bitrate',
          type: 'bar' as const,
          data: avgBitrates.map((val, idx) => ({
            value: val,
            itemStyle: { color: colors[idx], borderRadius: this.helpers.getBarBorderRadius('vertical') }
          })),
          emphasis: {
            itemStyle: { shadowBlur: 10, shadowColor: 'rgba(0,0,0,0.5)' }
          }
        },
        {
          name: 'Transcode Rate',
          type: 'line' as const,
          yAxisIndex: 1,
          data: transcodeRates,
          smooth: true,
          lineStyle: { width: 2, color: CHART_COLORS.warning, type: 'dashed' as const },
          symbol: 'circle',
          symbolSize: 8
        }
      ]
    };

    this.setOption(option);
  }

  /**
   * Creates histogram buckets from bitrate data
   * @param values Array of bitrate values
   * @param bucketCount Number of buckets to create
   * @returns Array of histogram buckets with labels and counts
   */
  private createHistogramBuckets(values: number[], bucketCount: number): { label: string; count: number }[] {
    if (values.length === 0) return [];

    const min = Math.min(...values);
    const max = Math.max(...values);
    const bucketSize = (max - min) / bucketCount;

    const buckets: { label: string; count: number }[] = [];
    for (let i = 0; i < bucketCount; i++) {
      const bucketMin = min + i * bucketSize;
      const bucketMax = min + (i + 1) * bucketSize;
      const count = values.filter(v => v >= bucketMin && v < bucketMax).length;

      // Handle last bucket edge case (include max value)
      if (i === bucketCount - 1) {
        const lastCount = values.filter(v => v >= bucketMin && v <= bucketMax).length;
        buckets.push({
          label: `${Math.round(bucketMin / 1000)}-${Math.round(bucketMax / 1000)}K`,
          count: lastCount
        });
      } else {
        buckets.push({
          label: `${Math.round(bucketMin / 1000)}-${Math.round(bucketMax / 1000)}K`,
          count
        });
      }
    }

    return buckets;
  }
}
