// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Renderer for comparative analytics charts
 */

import type { ComparativeAnalyticsResponse } from '../../api';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS, STYLE_CONSTANTS } from '../config/colors';
import { escapeHtml } from '../../sanitize';
import type {
  EChartsCallbackDataParams,
  EChartsItemStyleColorParams,
  EChartsLabelFormatterParams,
} from '../types';

export class ComparativeChartRenderer extends BaseChartRenderer {
  render(_data: ComparativeAnalyticsResponse): void {
    // Base render method - unused in this renderer
  }

  renderMetrics(data: ComparativeAnalyticsResponse): void {
    if (!this.hasData(data?.metrics_comparison)) {
      this.showEmptyState('No metrics comparison data available');
      return;
    }
    const metrics = data.metrics_comparison;
    const metricNames = metrics.map(m => m.metric);
    const currentValues = metrics.map(m => m.current_value);
    const previousValues = metrics.map(m => m.previous_value);
    const percentChanges = metrics.map(m => m.percentage_change);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const metric = metrics[params[0].dataIndex];
          return `
            <strong>${metric.metric}</strong><br/>
            Current: ${this.formatters.formatNumber(metric.current_value)}<br/>
            Previous: ${this.formatters.formatNumber(metric.previous_value)}<br/>
            Change: ${metric.percentage_change >= 0 ? '+' : ''}${this.formatters.formatPercent(metric.percentage_change)}
            <span style="color: ${metric.is_improvement ? CHART_COLORS.success : CHART_COLORS.crimson}">${metric.is_improvement ? '↑' : '↓'}</span>
          `;
        }
      }),
      legend: this.getLegend({ data: ['Current Period', 'Previous Period'] }),
      grid: { left: '3%', right: '4%', bottom: '10%', top: '15%', containLabel: true },
      xAxis: this.getCategoryAxis(metricNames, { rotate: 30, interval: 0 }),
      yAxis: this.getValueAxis(),
      series: [
        {
          name: 'Current Period',
          type: 'bar' as const,
          data: currentValues,
          itemStyle: { color: CHART_COLORS.success },
          label: {
            show: true,
            position: 'top' as const,
            color: STYLE_CONSTANTS.textColor,
            formatter: (params: EChartsLabelFormatterParams) => {
              const pct = percentChanges[params.dataIndex];
              return pct >= 0 ? `+${this.formatters.formatPercent(pct)}` : this.formatters.formatPercent(pct);
            }
          }
        },
        {
          name: 'Previous Period',
          type: 'bar' as const,
          data: previousValues,
          itemStyle: { color: CHART_COLORS.info }
        }
      ]
    };

    this.setOption(option);

    // Update insights list - escape HTML to prevent XSS
    const insightsList = document.getElementById('insights-list');
    if (insightsList && data.key_insights) {
      insightsList.innerHTML = data.key_insights
        .map(insight => `<li style="padding: 10px 0; border-bottom: 1px solid #2a2a3e;">${escapeHtml(insight)}</li>`)
        .join('');
    }
  }

  renderContent(data: ComparativeAnalyticsResponse): void {
    if (!this.hasData(data?.top_content_comparison)) {
      this.showEmptyState('No content comparison data available');
      return;
    }
    const content = data.top_content_comparison.slice(0, 10);
    const titles = content.map(c => c.title.length > 30 ? c.title.substring(0, 27) + '...' : c.title);
    const currentCounts = content.map(c => c.current_count);
    const trendingIcons = content.map(c => {
      if (c.trending === 'new') return '🆕';
      if (c.trending === 'rising') return '📈';
      if (c.trending === 'falling') return '📉';
      return '➡️';
    });

    const [reversedTitles, reversedCounts, reversedIcons] = this.helpers.reverseArrays(titles, currentCounts, trendingIcons);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const item = content[params[0].dataIndex];
          return `
            <strong>${item.title}</strong><br/>
            Current Rank: #${item.current_rank}<br/>
            Previous Rank: ${item.previous_rank > 0 ? '#' + item.previous_rank : 'New'}<br/>
            Playbacks: ${item.current_count}<br/>
            Change: ${item.count_change >= 0 ? '+' : ''}${item.count_change} (${item.count_change_pct >= 0 ? '+' : ''}${this.formatters.formatPercent(item.count_change_pct)})
          `;
        }
      }),
      grid: { left: '3%', right: '4%', bottom: '3%', top: '10%', containLabel: true },
      xAxis: this.getValueAxis(),
      yAxis: this.getCategoryAxis(reversedTitles),
      series: [{
        type: 'bar' as const,
        data: reversedCounts,
        itemStyle: {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          color: ((params: EChartsItemStyleColorParams) => {
            const trending = content[content.length - 1 - params.dataIndex].trending;
            if (trending === 'new') return CHART_COLORS.purple;
            if (trending === 'rising') return CHART_COLORS.success;
            if (trending === 'falling') return CHART_COLORS.crimson;
            return CHART_COLORS.info;
          }) as any,
          borderRadius: this.helpers.getBarBorderRadius('horizontal')
        },
        label: {
          show: true,
          position: 'right' as const,
          color: STYLE_CONSTANTS.textColor,
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          formatter: ((params: EChartsLabelFormatterParams) => reversedIcons[params.dataIndex]) as any
        }
      }]
    };

    this.setOption(option);
  }

  renderUsers(data: ComparativeAnalyticsResponse): void {
    if (!this.hasData(data?.top_user_comparison)) {
      this.showEmptyState('No user comparison data available');
      return;
    }
    const users = data.top_user_comparison.slice(0, 10);
    const usernames = users.map(u => u.username);
    const currentWatchTime = users.map(u => u.current_watch_time);
    const trendingIcons = users.map(u => {
      if (u.trending === 'new') return '🆕';
      if (u.trending === 'rising') return '📈';
      if (u.trending === 'falling') return '📉';
      return '➡️';
    });

    const [reversedUsernames, reversedWatchTime, reversedIcons] = this.helpers.reverseArrays(usernames, currentWatchTime, trendingIcons);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const user = users[params[0].dataIndex];
          return `
            <strong>${user.username}</strong><br/>
            Current Rank: #${user.current_rank}<br/>
            Previous Rank: ${user.previous_rank > 0 ? '#' + user.previous_rank : 'New'}<br/>
            Watch Time: ${this.formatters.formatNumber(user.current_watch_time)} min<br/>
            Change: ${user.watch_time_change >= 0 ? '+' : ''}${this.formatters.formatNumber(user.watch_time_change)} min (${user.watch_time_change_pct >= 0 ? '+' : ''}${this.formatters.formatPercent(user.watch_time_change_pct)})
          `;
        }
      }),
      grid: { left: '3%', right: '4%', bottom: '3%', top: '10%', containLabel: true },
      xAxis: this.getValueAxis({ name: 'Minutes' }),
      yAxis: this.getCategoryAxis(reversedUsernames),
      series: [{
        type: 'bar' as const,
        data: reversedWatchTime,
        itemStyle: {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          color: ((params: EChartsItemStyleColorParams) => {
            const trending = users[users.length - 1 - params.dataIndex].trending;
            if (trending === 'new') return CHART_COLORS.purple;
            if (trending === 'rising') return CHART_COLORS.success;
            if (trending === 'falling') return CHART_COLORS.crimson;
            return CHART_COLORS.info;
          }) as any,
          borderRadius: this.helpers.getBarBorderRadius('horizontal')
        },
        label: {
          show: true,
          position: 'right' as const,
          color: STYLE_CONSTANTS.textColor,
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          formatter: ((params: EChartsLabelFormatterParams) => reversedIcons[params.dataIndex]) as any
        }
      }]
    };

    this.setOption(option);
  }
}
