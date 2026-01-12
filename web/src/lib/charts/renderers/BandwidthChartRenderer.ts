// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Renderer for bandwidth analytics charts
 */

import type { BandwidthAnalyticsResponse } from '../../api';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS, STYLE_CONSTANTS } from '../config/colors';
import { DateRangeBrushManager } from '../../../app/DateRangeBrushManager';
import type { EChartsCallbackDataParams, EChartsItemStyleColorParams } from '../types';

export class BandwidthChartRenderer extends BaseChartRenderer {
  render(_data: BandwidthAnalyticsResponse): void {
    // Base render method - unused in this renderer
  }

  renderTrends(data: BandwidthAnalyticsResponse): void {
    if (!this.hasData(data?.trends)) {
      this.showEmptyState('No bandwidth trend data available');
      return;
    }
    const dates = data.trends.map(t => t.date);
    const bandwidth = data.trends.map(t => t.bandwidth_gb);
    const mbps = data.trends.map(t => t.avg_mbps);

    // Enable brush for date range selection
    const brushConfig = DateRangeBrushManager.getBrushConfig();

    const option = {
      ...this.getBaseOption(),
      ...brushConfig,
      title: {
        text: `Total: ${data.total_bandwidth_gb.toFixed(1)} GB | Peak: ${data.peak_bandwidth_mbps.toFixed(1)} Mbps`,
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
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${param.seriesName?.includes('GB') ? value.toFixed(2) + ' GB' : value.toFixed(1) + ' Mbps'}</span>
            </div>`;
          });
          return html;
        }
      }),
      legend: this.getLegend({ data: ['Bandwidth (GB)', 'Avg Speed (Mbps)'], top: '10%' }),
      grid: { left: '10%', right: '10%', bottom: '10%', top: '25%' },
      xAxis: this.getCategoryAxis(dates),
      yAxis: [
        this.getValueAxis({ name: 'Bandwidth (GB)' }),
        this.getValueAxis({ name: 'Speed (Mbps)', splitLine: { show: false } })
      ],
      series: [
        {
          name: 'Bandwidth (GB)',
          type: 'bar' as const,
          data: bandwidth,
          itemStyle: { color: CHART_COLORS.primary, borderRadius: this.helpers.getBarBorderRadius('vertical') }
        },
        {
          name: 'Avg Speed (Mbps)',
          type: 'line' as const,
          yAxisIndex: 1,
          data: mbps,
          smooth: true,
          itemStyle: { color: CHART_COLORS.success },
          lineStyle: { width: 3 }
        }
      ]
    };

    this.setOption(option);
  }

  renderTranscode(data: BandwidthAnalyticsResponse): void {
    if (!this.hasData(data?.by_transcode)) {
      this.showEmptyState('No transcode bandwidth data available');
      return;
    }
    const chartData = data.by_transcode.map(t => ({
      name: t.transcode_decision,
      value: t.bandwidth_gb
    }));

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'item',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          const p = Array.isArray(params) ? params[0] : params;
          const item = data.by_transcode.find(t => t.transcode_decision === p.name);
          if (!item) return '';
          return `<div style="font-weight: 600; margin-bottom: 8px;">${p.name}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Bandwidth:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${item.bandwidth_gb.toFixed(2)} GB</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Percentage:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatPercent(item.percentage)}</span>
            </div>`;
        }
      }),
      legend: this.getLegend({ orient: 'vertical' as const, left: 'left' }),
      series: [{
        type: 'pie' as const,
        radius: ['40%', '70%'],
        itemStyle: { borderRadius: 8, borderColor: STYLE_CONSTANTS.borderColor, borderWidth: 2 },
        label: { show: true, formatter: '{b}: {d}%', color: STYLE_CONSTANTS.textColor },
        data: chartData,
        color: [CHART_COLORS.success, CHART_COLORS.primary, CHART_COLORS.warning]
      }]
    };

    this.setOption(option);
  }

  renderResolution(data: BandwidthAnalyticsResponse): void {
    if (!this.hasData(data?.by_resolution)) {
      this.showEmptyState('No resolution bandwidth data available');
      return;
    }
    const resolutions = data.by_resolution.map(r => r.resolution);
    const bandwidth = data.by_resolution.map(r => r.bandwidth_gb);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const res = data.by_resolution[params[0].dataIndex];
          return `<div style="font-weight: 600; margin-bottom: 8px;">${res.resolution}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Bandwidth:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${res.bandwidth_gb.toFixed(2)} GB</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Avg Speed:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${res.avg_bandwidth_mbps.toFixed(1)} Mbps</span>
            </div>`;
        }
      }),
      grid: { left: '10%', right: '10%', bottom: '10%', top: '10%' },
      xAxis: this.getCategoryAxis(resolutions),
      yAxis: this.getValueAxis({ name: 'Bandwidth (GB)' }),
      series: [{
        type: 'bar' as const,
        data: bandwidth,
        itemStyle: {
          color: (params: EChartsItemStyleColorParams) => {
            const colors = [CHART_COLORS.primary, CHART_COLORS.orange, CHART_COLORS.warning, CHART_COLORS.success, CHART_COLORS.accent];
            return colors[params.dataIndex % colors.length];
          },
          borderRadius: this.helpers.getBarBorderRadius('vertical')
        },
        label: { show: true, position: 'top' as const, color: STYLE_CONSTANTS.textColor, formatter: '{c} GB' }
      }]
    };

    this.setOption(option);
  }

  renderUsers(data: BandwidthAnalyticsResponse): void {
    if (!this.hasData(data?.top_users)) {
      this.showEmptyState('No top user bandwidth data available');
      return;
    }
    const users = data.top_users.map(u => u.username);
    const bandwidth = data.top_users.map(u => u.bandwidth_gb);

    const [reversedUsers, reversedBandwidth] = this.helpers.reverseArrays(users, bandwidth);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const user = data.top_users[params[0].dataIndex];
          return `<div style="font-weight: 600; margin-bottom: 8px;">${user.username}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Bandwidth:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${user.bandwidth_gb.toFixed(2)} GB</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Direct Play:</span>
              <span style="color: ${CHART_COLORS.success}; font-weight: 600;">${user.direct_play_count}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Transcode:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${user.transcode_count}</span>
            </div>`;
        }
      }),
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: this.getValueAxis({ name: 'Bandwidth (GB)' }),
      yAxis: this.getCategoryAxis(reversedUsers),
      series: [{
        type: 'bar' as const,
        data: reversedBandwidth,
        itemStyle: {
          color: this.helpers.createLinearGradient('horizontal', [
            { offset: 0, color: CHART_COLORS.primary },
            { offset: 1, color: CHART_COLORS.secondary }
          ]),
          borderRadius: this.helpers.getBarBorderRadius('horizontal')
        },
        label: { show: true, position: 'right' as const, color: STYLE_CONSTANTS.textColor, formatter: '{c} GB' }
      }]
    };

    this.setOption(option);
  }
}
