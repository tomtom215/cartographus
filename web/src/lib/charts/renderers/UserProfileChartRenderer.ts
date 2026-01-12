// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Renderer for user profile analytics charts
 * Handles activity trends, top content, platform usage
 */

import type { UserProfileAnalytics } from '../../api';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS, STYLE_CONSTANTS } from '../config/colors';
import type { EChartsCallbackDataParams } from '../types';

export class UserProfileChartRenderer extends BaseChartRenderer {
  render(_data: UserProfileAnalytics): void {
    // Base render method - unused in this renderer
  }

  /**
   * Render user activity trend chart (bar + line combo)
   */
  renderUserActivityTrend(data: UserProfileAnalytics): void {
    if (!this.hasData(data?.activity_trend)) {
      this.showEmptyState('No activity data available for this user');
      return;
    }

    const trends = data.activity_trend;
    const dates = trends.map(t => t.date);
    const plays = trends.map(t => t.plays);
    const watchTimeHours = trends.map(t => Math.round(t.watch_time_minutes / 60));

    const option = {
      ...this.getBaseOption(),
      title: {
        text: `Activity Trend - ${data.profile.friendly_name || data.profile.username}`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 16, fontWeight: 600 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'cross' }
      }),
      legend: this.getLegend({
        data: ['Plays', 'Watch Time (hours)'],
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
          name: 'Plays',
          axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
          axisLine: { lineStyle: { color: '#3a3a4e' } },
          splitLine: { lineStyle: { color: '#2a2a3e', type: 'dashed' as const } },
          nameTextStyle: { color: STYLE_CONSTANTS.axisLabelColor }
        },
        {
          type: 'value' as const,
          name: 'Hours',
          axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
          axisLine: { lineStyle: { color: '#3a3a4e' } },
          nameTextStyle: { color: STYLE_CONSTANTS.axisLabelColor }
        }
      ],
      series: [
        {
          name: 'Plays',
          type: 'bar' as const,
          data: plays,
          itemStyle: { color: CHART_COLORS.primary }
        },
        {
          name: 'Watch Time (hours)',
          type: 'line' as const,
          yAxisIndex: 1,
          data: watchTimeHours,
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
   * Render user top content chart (horizontal bar)
   */
  renderUserTopContent(data: UserProfileAnalytics): void {
    if (!this.hasData(data?.top_content)) {
      this.showEmptyState('No content data available for this user');
      return;
    }

    const content = data.top_content.slice(0, 10);
    const titles = content.map(c => c.title);
    const plays = content.map(c => c.plays);
    const watchTimeHours = content.map(c => Math.round(c.watch_time_minutes / 60));

    // Color by media type
    const colors = content.map(c => {
      switch (c.media_type) {
        case 'movie': return '#3b82f6';
        case 'episode': return '#10b981';
        case 'track': return '#8b5cf6';
        default: return CHART_COLORS.primary;
      }
    });

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Top Content',
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 16, fontWeight: 600 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const idx = params[0].dataIndex;
          const item = content[idx];
          if (!item) return '';
          return `<strong>${item.title}</strong><br/>
                  Type: ${item.media_type}<br/>
                  Plays: ${item.plays}<br/>
                  Watch Time: ${Math.round(item.watch_time_minutes / 60)}h<br/>
                  Last Played: ${item.last_played || 'N/A'}`;
        }
      }),
      legend: this.getLegend({
        data: ['Plays', 'Watch Time (hours)'],
        bottom: 0
      }),
      grid: {
        left: 120,
        right: 40,
        top: 50,
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
        data: titles,
        inverse: true,
        axisLabel: {
          color: STYLE_CONSTANTS.axisLabelColor,
          width: 100,
          overflow: 'truncate' as const
        },
        axisLine: { lineStyle: { color: '#3a3a4e' } }
      },
      series: [
        {
          name: 'Plays',
          type: 'bar' as const,
          data: plays.map((value, idx) => ({
            value,
            itemStyle: { color: colors[idx] }
          })),
          barWidth: '40%'
        },
        {
          name: 'Watch Time (hours)',
          type: 'bar' as const,
          data: watchTimeHours,
          itemStyle: { color: CHART_COLORS.secondary },
          barWidth: '40%'
        }
      ]
    };

    this.setOption(option, { notMerge: true });
  }

  /**
   * Render user platform usage chart (pie)
   */
  renderUserPlatforms(data: UserProfileAnalytics): void {
    if (!this.hasData(data?.platform_usage)) {
      this.showEmptyState('No platform data available for this user');
      return;
    }

    const platforms = data.platform_usage;
    const pieData = platforms.map((p, idx) => ({
      name: p.platform,
      value: p.plays,
      itemStyle: { color: this.getPlatformColor(p.platform, idx) }
    }));

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Platform Usage',
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 16, fontWeight: 600 }
      },
      tooltip: this.getTooltip({
        trigger: 'item',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          const p = Array.isArray(params) ? params[0] : params;
          const platform = platforms.find(pl => pl.platform === p.name);
          if (!platform) return p.name ?? '';
          return `<strong>${platform.platform}</strong><br/>
                  Plays: ${platform.plays}<br/>
                  Watch Time: ${Math.round(platform.watch_time_minutes / 60)}h<br/>
                  Share: ${platform.percentage.toFixed(1)}%`;
        }
      }),
      legend: this.getLegend({
        data: platforms.map(p => p.platform),
        bottom: 0,
        type: 'scroll' as const
      }),
      series: [{
        type: 'pie' as const,
        radius: ['35%', '60%'],
        center: ['50%', '50%'],
        data: pieData,
        label: {
          show: true,
          formatter: '{b}: {d}%',
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

  /**
   * Get color for platform
   */
  private getPlatformColor(platform: string, index: number): string {
    const platformColors: Record<string, string> = {
      'Android': '#3DDC84',
      'iOS': '#007AFF',
      'Windows': '#0078D4',
      'macOS': '#555555',
      'Linux': '#FCC624',
      'Chrome': '#4285F4',
      'Firefox': '#FF7139',
      'Safari': '#006CFF',
      'Roku': '#6F2DA8',
      'Apple TV': '#000000',
      'Fire TV': '#FF9900',
      'Chromecast': '#4285F4',
      'Xbox': '#107C10',
      'PlayStation': '#003087',
      'Smart TV': '#00A5E0',
      'Plex Web': '#E5A00D',
      'Plex HTPC': '#E5A00D'
    };

    if (platformColors[platform]) {
      return platformColors[platform];
    }

    // Fallback to chart colors
    const fallbackColors = [
      CHART_COLORS.primary,
      CHART_COLORS.secondary,
      CHART_COLORS.accent,
      '#f59e0b',
      '#ef4444',
      '#8b5cf6',
      '#06b6d4',
      '#ec4899'
    ];

    return fallbackColors[index % fallbackColors.length];
  }
}
