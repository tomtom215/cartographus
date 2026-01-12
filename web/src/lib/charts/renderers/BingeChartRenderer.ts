// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Renderer for binge-watching analytics charts
 */

import type { BingeAnalyticsResponse } from '../../api';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS, COLOR_PALETTES, STYLE_CONSTANTS } from '../config/colors';
import type { EChartsCallbackDataParams } from '../types';

export class BingeChartRenderer extends BaseChartRenderer {
  render(_data: BingeAnalyticsResponse): void {
    // Base render method - unused in this renderer
  }

  renderSummary(data: BingeAnalyticsResponse): void {
    if (!this.hasData(data?.binges_by_day)) {
      this.showEmptyState('No binge data available');
      return;
    }
    const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
    const bingesByDay = data.binges_by_day.map(b => ({
      day: days[b.day_of_week],
      count: b.binge_count,
      avgEpisodes: b.avg_episodes
    }));

    const option = {
      ...this.getBaseOption(),
      title: {
        text: `${this.formatters.formatNumber(data.total_binge_sessions)} Binge Sessions | ${this.formatters.formatNumber(data.total_episodes_binged)} Episodes`,
        left: 'center',
        top: 10,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 16, fontWeight: 600 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const dayData = bingesByDay[params[0].dataIndex];
          return `<div style="font-weight: 600; margin-bottom: 8px;">${dayData.day}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Binge Sessions:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(dayData.count)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Avg Episodes:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${dayData.avgEpisodes.toFixed(1)}</span>
            </div>`;
        }
      }),
      grid: { left: '10%', right: '10%', bottom: '10%', top: '20%' },
      xAxis: this.getCategoryAxis(bingesByDay.map(b => b.day), { rotate: 30 }),
      yAxis: this.getValueAxis({ name: 'Sessions' }),
      series: [{
        type: 'bar' as const,
        data: bingesByDay.map(b => b.count),
        itemStyle: {
          color: this.helpers.createLinearGradient('vertical', COLOR_PALETTES.gradient.redToPurple),
          borderRadius: this.helpers.getBarBorderRadius('vertical')
        },
        label: { show: true, position: 'top' as const, color: STYLE_CONSTANTS.textColor }
      }]
    };

    this.setOption(option);
  }

  renderShows(data: BingeAnalyticsResponse): void {
    if (!this.hasData(data?.top_binge_shows)) {
      this.showEmptyState('No binge show data available');
      return;
    }
    const shows = data.top_binge_shows.map(s => s.show_name);
    const counts = data.top_binge_shows.map(s => s.binge_count);

    const [reversedShows, reversedCounts] = this.helpers.reverseArrays(shows, counts);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const show = data.top_binge_shows[params[0].dataIndex];
          return `<div style="font-weight: 600; margin-bottom: 8px;">${show.show_name}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Binge Sessions:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(show.binge_count)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Total Episodes:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(show.total_episodes)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Unique Watchers:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(show.unique_watchers)}</span>
            </div>`;
        }
      }),
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: this.getValueAxis(),
      yAxis: this.getCategoryAxis(reversedShows),
      series: [{
        type: 'bar' as const,
        data: reversedCounts,
        itemStyle: {
          color: this.helpers.createLinearGradient('horizontal', COLOR_PALETTES.gradient.redToBlue),
          borderRadius: this.helpers.getBarBorderRadius('horizontal')
        },
        label: { show: true, position: 'right' as const, color: STYLE_CONSTANTS.textColor }
      }]
    };

    this.setOption(option);
  }

  renderUsers(data: BingeAnalyticsResponse): void {
    if (!this.hasData(data?.top_binge_watchers)) {
      this.showEmptyState('No binge watcher data available');
      return;
    }
    const users = data.top_binge_watchers.map(u => u.username);
    const counts = data.top_binge_watchers.map(u => u.total_episodes);

    const [reversedUsers, reversedCounts] = this.helpers.reverseArrays(users, counts);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const user = data.top_binge_watchers[params[0].dataIndex];
          return `<div style="font-weight: 600; margin-bottom: 8px;">${user.username}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Binge Sessions:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(user.binge_count)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Total Episodes:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(user.total_episodes)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Favorite Show:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${user.favorite_show}</span>
            </div>`;
        }
      }),
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: this.getValueAxis({ name: 'Episodes Binged' }),
      yAxis: this.getCategoryAxis(reversedUsers),
      series: [{
        type: 'bar' as const,
        data: reversedCounts,
        itemStyle: {
          color: this.helpers.createLinearGradient('horizontal', COLOR_PALETTES.gradient.redToPurple),
          borderRadius: this.helpers.getBarBorderRadius('horizontal')
        },
        label: { show: true, position: 'right' as const, color: STYLE_CONSTANTS.textColor }
      }]
    };

    this.setOption(option);
  }
}
