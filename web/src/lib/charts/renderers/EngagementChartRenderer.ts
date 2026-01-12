// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Renderer for user engagement analytics charts
 */

import type { UserEngagementAnalyticsResponse } from '../../api';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS, STYLE_CONSTANTS } from '../config/colors';
import type { EChartsCallbackDataParams, EChartsItemStyleColorParams } from '../types';

export class EngagementChartRenderer extends BaseChartRenderer {
  render(_data: UserEngagementAnalyticsResponse): void {
    // Base render method - unused in this renderer
  }

  renderSummary(data: UserEngagementAnalyticsResponse): void {
    if (!data?.summary) {
      this.showEmptyState('No engagement data available');
      return;
    }
    const metrics = [
      { name: 'Total Users', value: data.summary.total_users },
      { name: 'Active Users', value: data.summary.active_users },
      { name: 'Total Sessions', value: data.summary.total_sessions },
      { name: 'Avg Session (min)', value: Math.round(data.summary.avg_session_minutes) },
      { name: 'Watch Time (hrs)', value: Math.round(data.summary.total_watch_time_minutes / 60) },
      { name: 'Completion Rate (%)', value: Math.round(data.summary.avg_completion_rate) }
    ];

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const metric = metrics[params[0].dataIndex];
          return `<div style="font-weight: 600; margin-bottom: 8px;">${metric.name}</div>
            <div style="margin: 4px 0;">
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600; font-size: 16px;">${this.formatters.formatNumber(metric.value)}</span>
            </div>`;
        }
      }),
      grid: { left: '10%', right: '10%', bottom: '15%', top: '10%' },
      xAxis: this.getCategoryAxis(metrics.map(m => m.name), { rotate: 30, fontSize: 11 }),
      yAxis: this.getValueAxis(),
      series: [{
        type: 'bar' as const,
        data: metrics.map(m => m.value),
        itemStyle: {
          color: (params: EChartsItemStyleColorParams) => {
            const colors = [CHART_COLORS.primary, CHART_COLORS.success, CHART_COLORS.warning, CHART_COLORS.accent, CHART_COLORS.orange, CHART_COLORS.secondary];
            return colors[params.dataIndex % colors.length];
          },
          borderRadius: this.helpers.getBarBorderRadius('vertical')
        },
        label: { show: true, position: 'top' as const, color: STYLE_CONSTANTS.textColor }
      }]
    };

    this.setOption(option);
  }

  renderHours(data: UserEngagementAnalyticsResponse): void {
    if (!this.hasData(data?.viewing_patterns_by_hour)) {
      this.showEmptyState('No hourly viewing data available');
      return;
    }
    const hours = data.viewing_patterns_by_hour.map(h => h.hour_of_day.toString().padStart(2, '0'));
    const sessions = data.viewing_patterns_by_hour.map(h => h.session_count);
    const watchTime = data.viewing_patterns_by_hour.map(h => Math.round(h.watch_time_minutes / 60));

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params)) return '';
          const hourData = data.viewing_patterns_by_hour[params[0].dataIndex];
          return `<div style="font-weight: 600; margin-bottom: 8px;">Hour: ${hourData.hour_of_day}:00</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Sessions:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(hourData.session_count)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Watch Time:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${Math.round(hourData.watch_time_minutes / 60)}h</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Unique Users:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${hourData.unique_users}</span>
            </div>`;
        }
      }),
      legend: this.getLegend({ data: ['Sessions', 'Watch Time (hrs)'] }),
      grid: { left: '10%', right: '10%', bottom: '10%', top: '15%' },
      xAxis: this.getCategoryAxis(hours),
      yAxis: [
        this.getValueAxis({ name: 'Sessions' }),
        this.getValueAxis({ name: 'Hours', splitLine: { show: false } })
      ],
      series: [
        {
          name: 'Sessions',
          type: 'bar' as const,
          data: sessions,
          itemStyle: { color: CHART_COLORS.primary, borderRadius: this.helpers.getBarBorderRadius('vertical') }
        },
        {
          name: 'Watch Time (hrs)',
          type: 'line' as const,
          yAxisIndex: 1,
          data: watchTime,
          smooth: true,
          itemStyle: { color: CHART_COLORS.success },
          lineStyle: { width: 3 }
        }
      ]
    };

    this.setOption(option);
  }

  renderDays(data: UserEngagementAnalyticsResponse): void {
    if (!this.hasData(data?.viewing_patterns_by_day)) {
      this.showEmptyState('No daily viewing data available');
      return;
    }
    const days = data.viewing_patterns_by_day.map(d => d.day_name);
    const sessions = data.viewing_patterns_by_day.map(d => d.session_count);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const dayData = data.viewing_patterns_by_day[params[0].dataIndex];
          return `<div style="font-weight: 600; margin-bottom: 8px;">${dayData.day_name}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Sessions:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(dayData.session_count)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Watch Time:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${Math.round(dayData.watch_time_minutes / 60)}h</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Unique Users:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${dayData.unique_users}</span>
            </div>`;
        }
      }),
      grid: { left: '10%', right: '10%', bottom: '10%', top: '10%' },
      xAxis: this.getCategoryAxis(days),
      yAxis: this.getValueAxis({ name: 'Sessions' }),
      series: [{
        type: 'bar' as const,
        data: sessions,
        itemStyle: {
          color: this.helpers.createLinearGradient('vertical', [
            { offset: 0, color: CHART_COLORS.success },
            { offset: 1, color: CHART_COLORS.secondary }
          ]),
          borderRadius: this.helpers.getBarBorderRadius('vertical')
        },
        label: { show: true, position: 'top' as const, color: STYLE_CONSTANTS.textColor }
      }]
    };

    this.setOption(option);
  }
}
