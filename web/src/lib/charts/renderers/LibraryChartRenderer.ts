// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Renderer for library-specific analytics charts
 * Handles top users, daily trends, quality distribution
 */

import type { LibraryAnalytics } from '../../api';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS, STYLE_CONSTANTS } from '../config/colors';
import type { EChartsCallbackDataParams } from '../types';

export class LibraryChartRenderer extends BaseChartRenderer {
  render(_data: LibraryAnalytics): void {
    // Base render method - unused in this renderer
  }

  /**
   * Render top users in library chart
   */
  renderLibraryUsers(data: LibraryAnalytics): void {
    if (!this.hasData(data?.top_users)) {
      this.showEmptyState('No user data available for this library');
      return;
    }

    const users = data.top_users.slice(0, 10);
    const usernames = users.map(u => u.username);
    const plays = users.map(u => u.plays);
    const watchTime = users.map(u => Math.round(u.watch_time_minutes / 60)); // Convert to hours

    const option = {
      ...this.getBaseOption(),
      title: {
        text: `Top Users - ${data.library_name}`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 16, fontWeight: 600 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const username = params[0].name;
          const user = users.find(u => u.username === username);
          if (!user) return '';
          return `<strong>${username}</strong><br/>
                  Plays: ${user.plays}<br/>
                  Watch Time: ${Math.round(user.watch_time_minutes / 60)}h<br/>
                  Avg Completion: ${user.avg_completion.toFixed(1)}%`;
        }
      }),
      legend: this.getLegend({
        data: ['Plays', 'Watch Time (hours)'],
        bottom: 0
      }),
      grid: {
        left: 100,
        right: 40,
        top: 60,
        bottom: 50,
        containLabel: true
      },
      xAxis: {
        type: 'value' as const,
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
        axisLine: { lineStyle: { color: '#3a3a4e' } },
        splitLine: { lineStyle: { color: '#2a2a3e', type: 'dashed' as const } }
      },
      yAxis: {
        type: 'category' as const,
        data: usernames,
        inverse: true,
        axisLabel: {
          color: STYLE_CONSTANTS.axisLabelColor,
          width: 80,
          overflow: 'truncate' as const
        },
        axisLine: { lineStyle: { color: '#3a3a4e' } }
      },
      series: [
        {
          name: 'Plays',
          type: 'bar' as const,
          data: plays,
          itemStyle: { color: CHART_COLORS.primary },
          barWidth: '40%'
        },
        {
          name: 'Watch Time (hours)',
          type: 'bar' as const,
          data: watchTime,
          itemStyle: { color: CHART_COLORS.secondary },
          barWidth: '40%'
        }
      ]
    };

    this.setOption(option, { notMerge: true });
  }

  /**
   * Render daily playback trend for library
   */
  renderLibraryTrend(data: LibraryAnalytics): void {
    if (!this.hasData(data?.plays_by_day)) {
      this.showEmptyState('No trend data available for this library');
      return;
    }

    const dates = data.plays_by_day.map(d => d.date);
    const playbacks = data.plays_by_day.map(d => d.playback_count);
    const uniqueUsers = data.plays_by_day.map(d => d.unique_users);

    const option = {
      ...this.getBaseOption(),
      title: {
        text: `Daily Trend - ${data.library_name}`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 16, fontWeight: 600 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'cross' }
      }),
      legend: this.getLegend({
        data: ['Playbacks', 'Unique Users'],
        bottom: 0
      }),
      grid: {
        left: 60,
        right: 60,
        top: 60,
        bottom: 50,
        containLabel: true
      },
      xAxis: {
        type: 'category' as const,
        data: dates,
        axisLabel: {
          color: STYLE_CONSTANTS.axisLabelColor,
          rotate: 45,
          formatter: (value: string) => {
            const date = new Date(value);
            return `${date.getMonth() + 1}/${date.getDate()}`;
          }
        },
        axisLine: { lineStyle: { color: '#3a3a4e' } }
      },
      yAxis: [
        {
          type: 'value' as const,
          name: 'Playbacks',
          axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
          axisLine: { lineStyle: { color: '#3a3a4e' } },
          splitLine: { lineStyle: { color: '#2a2a3e', type: 'dashed' as const } },
          nameTextStyle: { color: STYLE_CONSTANTS.axisLabelColor }
        },
        {
          type: 'value' as const,
          name: 'Users',
          axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
          axisLine: { lineStyle: { color: '#3a3a4e' } },
          nameTextStyle: { color: STYLE_CONSTANTS.axisLabelColor }
        }
      ],
      series: [
        {
          name: 'Playbacks',
          type: 'bar' as const,
          data: playbacks,
          itemStyle: { color: CHART_COLORS.primary }
        },
        {
          name: 'Unique Users',
          type: 'line' as const,
          yAxisIndex: 1,
          data: uniqueUsers,
          smooth: true,
          lineStyle: { color: CHART_COLORS.accent, width: 2 },
          itemStyle: { color: CHART_COLORS.accent },
          symbol: 'circle',
          symbolSize: 6
        }
      ]
    };

    this.setOption(option, { notMerge: true });
  }

  /**
   * Render quality distribution chart for library
   */
  renderLibraryQuality(data: LibraryAnalytics): void {
    if (!data?.quality_distribution) {
      this.showEmptyState('No quality data available for this library');
      return;
    }

    const quality = data.quality_distribution;
    const qualityData = [
      { name: 'HDR Content', value: quality.hdr_content_count, color: '#f59e0b' },
      { name: '4K Content', value: quality['4k_content_count'], color: '#3b82f6' },
      { name: 'Surround Sound', value: quality.surround_sound_count, color: '#8b5cf6' }
    ].filter(item => item.value > 0);

    if (qualityData.length === 0) {
      this.showEmptyState('No premium quality content in this library');
      return;
    }

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Quality Distribution',
        subtext: `Avg Bitrate: ${(quality.avg_bitrate_kbps / 1000).toFixed(1)} Mbps`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 16, fontWeight: 600 },
        subtextStyle: { color: STYLE_CONSTANTS.axisLabelColor, fontSize: 12 }
      },
      tooltip: this.getTooltip({
        trigger: 'item',
        formatter: '{b}: {c} items ({d}%)'
      }),
      legend: this.getLegend({
        data: qualityData.map(d => d.name),
        bottom: 0
      }),
      series: [{
        type: 'pie' as const,
        radius: ['40%', '65%'],
        center: ['50%', '55%'],
        data: qualityData.map(item => ({
          name: item.name,
          value: item.value,
          itemStyle: { color: item.color }
        })),
        label: {
          show: true,
          formatter: '{b}: {c}',
          color: STYLE_CONSTANTS.textColor
        },
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        }
      }]
    };

    this.setOption(option, { notMerge: true });
  }
}
