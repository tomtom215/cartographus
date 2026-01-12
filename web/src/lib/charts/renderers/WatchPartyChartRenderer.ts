// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Renderer for watch party analytics charts
 */

import type { WatchPartyAnalyticsResponse } from '../../api';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS, COLOR_PALETTES, STYLE_CONSTANTS } from '../config/colors';
import * as echarts from 'echarts';
import type { EChartsCallbackDataParams } from '../types';

export class WatchPartyChartRenderer extends BaseChartRenderer {
  render(_data: WatchPartyAnalyticsResponse): void {
    // Base render method - unused in this renderer
  }

  renderSummary(data: WatchPartyAnalyticsResponse): void {
    if (!this.hasData(data?.parties_by_day)) {
      this.showEmptyState('No watch party data available');
      return;
    }
    const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
    const partiesByDay = data.parties_by_day.map(p => ({
      day: days[p.day_of_week],
      count: p.party_count,
      avgParticipants: p.avg_participants
    }));

    const option = {
      ...this.getBaseOption(),
      title: {
        text: `${this.formatters.formatNumber(data.total_watch_parties)} Watch Parties | ${this.formatters.formatNumber(data.total_participants)} Participants`,
        left: 'center',
        top: 10,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 16, fontWeight: 600 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const dayData = partiesByDay[params[0].dataIndex];
          return `<div style="font-weight: 600; margin-bottom: 8px;">${dayData.day}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Watch Parties:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(dayData.count)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Avg Participants:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${dayData.avgParticipants.toFixed(1)}</span>
            </div>`;
        }
      }),
      grid: { left: '10%', right: '10%', bottom: '10%', top: '20%' },
      xAxis: this.getCategoryAxis(partiesByDay.map(p => p.day), { rotate: 30 }),
      yAxis: this.getValueAxis({ name: 'Watch Parties' }),
      series: [{
        type: 'bar' as const,
        data: partiesByDay.map(p => p.count),
        itemStyle: {
          color: this.helpers.createLinearGradient('vertical', COLOR_PALETTES.gradient.greenToBlue),
          borderRadius: this.helpers.getBarBorderRadius('vertical')
        },
        label: { show: true, position: 'top' as const, color: STYLE_CONSTANTS.textColor }
      }]
    };

    this.setOption(option);
  }

  renderContent(data: WatchPartyAnalyticsResponse): void {
    if (!this.hasData(data?.top_content)) {
      this.showEmptyState('No watch party content data available');
      return;
    }
    const content = data.top_content.map(c => c.title);
    const counts = data.top_content.map(c => c.party_count);

    const [reversedContent, reversedCounts] = this.helpers.reverseArrays(content, counts);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const item = data.top_content[params[0].dataIndex];
          return `<div style="font-weight: 600; margin-bottom: 8px;">${item.title}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Watch Parties:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(item.party_count)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Total Participants:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(item.total_participants)}</span>
            </div>`;
        }
      }),
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: this.getValueAxis(),
      yAxis: this.getCategoryAxis(reversedContent),
      series: [{
        type: 'bar' as const,
        data: reversedCounts,
        itemStyle: {
          color: new echarts.graphic.LinearGradient(1, 0, 0, 0, [
            { offset: 0, color: CHART_COLORS.success },
            { offset: 1, color: CHART_COLORS.orange }
          ]),
          borderRadius: this.helpers.getBarBorderRadius('horizontal')
        },
        label: { show: true, position: 'right' as const, color: STYLE_CONSTANTS.textColor }
      }]
    };

    this.setOption(option);
  }

  renderUsers(data: WatchPartyAnalyticsResponse): void {
    if (!this.hasData(data?.top_social_users)) {
      this.showEmptyState('No social user data available');
      return;
    }
    const users = data.top_social_users.map(u => u.username);
    const counts = data.top_social_users.map(u => u.party_count);

    const [reversedUsers, reversedCounts] = this.helpers.reverseArrays(users, counts);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const user = data.top_social_users[params[0].dataIndex];
          return `<div style="font-weight: 600; margin-bottom: 8px;">${user.username}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Watch Parties:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(user.party_count)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Avg Party Size:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${user.avg_party_size.toFixed(1)}</span>
            </div>`;
        }
      }),
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: this.getValueAxis(),
      yAxis: this.getCategoryAxis(reversedUsers),
      series: [{
        type: 'bar' as const,
        data: reversedCounts,
        itemStyle: {
          color: new echarts.graphic.LinearGradient(1, 0, 0, 0, [
            { offset: 0, color: CHART_COLORS.warning },
            { offset: 1, color: CHART_COLORS.success }
          ]),
          borderRadius: this.helpers.getBarBorderRadius('horizontal')
        },
        label: { show: true, position: 'right' as const, color: STYLE_CONSTANTS.textColor }
      }]
    };

    this.setOption(option);
  }
}
