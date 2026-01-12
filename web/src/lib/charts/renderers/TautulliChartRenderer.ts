// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Renderer for Tautulli-specific charts
 * Handles user logins, user IPs, synced items, stream type by platform, activity history,
 * plays per month, and concurrent streams by type
 */

import type {
  TautulliUserLoginsData,
  TautulliUserIPData,
  TautulliSyncedItem,
  TautulliStreamTypeByPlatform,
  TautulliHistoryRow,
  TautulliPlaysPerMonthData,
  TautulliConcurrentStreamsByType
} from '../../api';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS, STYLE_CONSTANTS } from '../config/colors';

export class TautulliChartRenderer extends BaseChartRenderer {
  render(_data: unknown): void {
    // Base render method - unused in this renderer
  }

  /**
   * Render user login history chart
   * Shows login attempts with success/failure breakdown
   */
  renderUserLogins(data: TautulliUserLoginsData): void {
    if (!this.hasData(data?.data)) {
      this.showEmptyState('No user login data available');
      return;
    }

    // Group logins by date
    const loginsByDate = new Map<string, { success: number; failed: number }>();
    data.data.forEach(login => {
      const date = login.time.split(' ')[0]; // Extract date part
      const current = loginsByDate.get(date) || { success: 0, failed: 0 };
      if (login.success === 1) {
        current.success++;
      } else {
        current.failed++;
      }
      loginsByDate.set(date, current);
    });

    const dates = Array.from(loginsByDate.keys()).sort();
    const successCounts = dates.map(d => loginsByDate.get(d)?.success || 0);
    const failedCounts = dates.map(d => loginsByDate.get(d)?.failed || 0);

    // Calculate totals
    const totalLogins = data.recordsTotal;
    const successfulLogins = data.data.filter(l => l.success === 1).length;
    const successRate = totalLogins > 0 ? ((successfulLogins / totalLogins) * 100).toFixed(1) : '0';

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'User Login History',
        subtext: `${successRate}% success rate | ${this.formatters.formatNumber(totalLogins)} total logins`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 },
        subtextStyle: { color: STYLE_CONSTANTS.axisLabelColor, fontSize: 12 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' }
      }),
      legend: this.getLegend({
        data: ['Successful', 'Failed'],
        bottom: 10
      }),
      grid: {
        left: 60,
        right: 40,
        top: 100,
        bottom: 60
      },
      xAxis: {
        type: 'category' as const,
        data: dates,
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor, rotate: 45 },
        axisLine: { lineStyle: { color: '#3a3a4e' } }
      },
      yAxis: {
        type: 'value' as const,
        name: 'Logins',
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
        axisLine: { lineStyle: { color: '#3a3a4e' } },
        splitLine: { lineStyle: { color: '#2a2a3e', type: 'dashed' as const } }
      },
      series: [
        {
          name: 'Successful',
          type: 'bar' as const,
          stack: 'logins',
          data: successCounts,
          itemStyle: { color: CHART_COLORS.primary }
        },
        {
          name: 'Failed',
          type: 'bar' as const,
          stack: 'logins',
          data: failedCounts,
          itemStyle: { color: '#ff5252' }
        }
      ]
    };

    this.setOption(option, { notMerge: true });
  }

  /**
   * Render user IP geolocation chart
   * Shows IP addresses and their usage patterns for a user
   */
  renderUserIPs(data: TautulliUserIPData[]): void {
    if (!this.hasData(data)) {
      this.showEmptyState('No user IP data available');
      return;
    }

    // Sort by play count descending
    const sortedData = [...data].sort((a, b) => b.play_count - a.play_count).slice(0, 10);

    const ips = sortedData.map(d => d.ip_address.substring(0, 20));
    const playCounts = sortedData.map(d => d.play_count);

    const totalPlays = data.reduce((sum, d) => sum + d.play_count, 0);
    const uniqueIPs = data.length;

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'User IP Addresses',
        subtext: `${uniqueIPs} unique IPs | ${this.formatters.formatNumber(totalPlays)} total plays`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 },
        subtextStyle: { color: STYLE_CONSTANTS.axisLabelColor, fontSize: 12 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: unknown) => {
          const p = params as { dataIndex: number; value: number }[];
          if (!Array.isArray(p) || p.length === 0) return '';
          const idx = p[0].dataIndex;
          const ipData = sortedData[idx];
          return `<strong>${ipData.ip_address}</strong><br/>
                  Platform: ${ipData.platform_name || 'Unknown'}<br/>
                  Player: ${ipData.player_name || 'Unknown'}<br/>
                  Plays: ${this.formatters.formatNumber(ipData.play_count)}<br/>
                  Last Played: ${ipData.last_played}`;
        }
      }),
      grid: {
        left: 120,
        right: 40,
        top: 100,
        bottom: 40
      },
      xAxis: {
        type: 'value' as const,
        name: 'Plays',
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
        axisLine: { lineStyle: { color: '#3a3a4e' } },
        splitLine: { lineStyle: { color: '#2a2a3e', type: 'dashed' as const } }
      },
      yAxis: {
        type: 'category' as const,
        data: ips,
        axisLabel: {
          color: STYLE_CONSTANTS.axisLabelColor,
          fontSize: 10
        },
        axisLine: { lineStyle: { color: '#3a3a4e' } }
      },
      series: [
        {
          name: 'Plays',
          type: 'bar' as const,
          data: playCounts.map((count, i) => ({
            value: count,
            itemStyle: {
              color: this.getColorByIndex(i)
            }
          })),
          label: {
            show: true,
            position: 'right' as const,
            formatter: '{c}',
            color: STYLE_CONSTANTS.textColor,
            fontSize: 10
          }
        }
      ]
    };

    this.setOption(option, { notMerge: true });
  }

  /**
   * Render synced items dashboard
   * Shows offline sync status for all users
   */
  renderSyncedItems(data: TautulliSyncedItem[]): void {
    if (!this.hasData(data)) {
      this.showEmptyState('No synced items data available');
      return;
    }

    // Group by user
    const byUser = new Map<string, { items: number; complete: number; downloading: number }>();
    data.forEach(item => {
      const current = byUser.get(item.friendly_name) || { items: 0, complete: 0, downloading: 0 };
      current.items += item.item_count;
      current.complete += item.item_complete_count;
      current.downloading += item.item_count - item.item_complete_count;
      byUser.set(item.friendly_name, current);
    });

    const users = Array.from(byUser.keys());
    const completeCounts = users.map(u => byUser.get(u)?.complete || 0);
    const downloadingCounts = users.map(u => byUser.get(u)?.downloading || 0);

    const totalItems = data.reduce((sum, d) => sum + d.item_count, 0);
    const totalComplete = data.reduce((sum, d) => sum + d.item_complete_count, 0);

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Synced Items by User',
        subtext: `${this.formatters.formatNumber(totalComplete)}/${this.formatters.formatNumber(totalItems)} items synced`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 },
        subtextStyle: { color: STYLE_CONSTANTS.axisLabelColor, fontSize: 12 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' }
      }),
      legend: this.getLegend({
        data: ['Complete', 'Downloading'],
        bottom: 10
      }),
      grid: {
        left: 100,
        right: 40,
        top: 100,
        bottom: 60
      },
      xAxis: {
        type: 'value' as const,
        name: 'Items',
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
        axisLine: { lineStyle: { color: '#3a3a4e' } },
        splitLine: { lineStyle: { color: '#2a2a3e', type: 'dashed' as const } }
      },
      yAxis: {
        type: 'category' as const,
        data: users,
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
        axisLine: { lineStyle: { color: '#3a3a4e' } }
      },
      series: [
        {
          name: 'Complete',
          type: 'bar' as const,
          stack: 'sync',
          data: completeCounts,
          itemStyle: { color: CHART_COLORS.primary }
        },
        {
          name: 'Downloading',
          type: 'bar' as const,
          stack: 'sync',
          data: downloadingCounts,
          itemStyle: { color: CHART_COLORS.accent }
        }
      ]
    };

    this.setOption(option, { notMerge: true });
  }

  /**
   * Render stream type by platform chart
   * Shows direct play vs transcode breakdown by platform
   */
  renderStreamTypeByPlatform(data: TautulliStreamTypeByPlatform[]): void {
    if (!this.hasData(data)) {
      this.showEmptyState('No stream type by platform data available');
      return;
    }

    const platforms = data.map(d => d.platform);
    const directPlay = data.map(d => d.direct_play);
    const directStream = data.map(d => d.direct_stream);
    const transcode = data.map(d => d.transcode);

    const totalPlays = data.reduce((sum, d) => sum + d.total_plays, 0);
    const totalDirectPlay = data.reduce((sum, d) => sum + d.direct_play, 0);
    const directPlayPercent = totalPlays > 0 ? ((totalDirectPlay / totalPlays) * 100).toFixed(1) : '0';

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Stream Type by Platform',
        subtext: `${directPlayPercent}% Direct Play | ${this.formatters.formatNumber(totalPlays)} total plays`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 },
        subtextStyle: { color: STYLE_CONSTANTS.axisLabelColor, fontSize: 12 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' }
      }),
      legend: this.getLegend({
        data: ['Direct Play', 'Direct Stream', 'Transcode'],
        bottom: 10
      }),
      grid: {
        left: 100,
        right: 40,
        top: 100,
        bottom: 60
      },
      xAxis: {
        type: 'value' as const,
        name: 'Plays',
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
        axisLine: { lineStyle: { color: '#3a3a4e' } },
        splitLine: { lineStyle: { color: '#2a2a3e', type: 'dashed' as const } }
      },
      yAxis: {
        type: 'category' as const,
        data: platforms,
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
        axisLine: { lineStyle: { color: '#3a3a4e' } }
      },
      series: [
        {
          name: 'Direct Play',
          type: 'bar' as const,
          stack: 'stream',
          data: directPlay,
          itemStyle: { color: CHART_COLORS.primary }
        },
        {
          name: 'Direct Stream',
          type: 'bar' as const,
          stack: 'stream',
          data: directStream,
          itemStyle: { color: CHART_COLORS.secondary }
        },
        {
          name: 'Transcode',
          type: 'bar' as const,
          stack: 'stream',
          data: transcode,
          itemStyle: { color: '#ff5252' }
        }
      ]
    };

    this.setOption(option, { notMerge: true });
  }

  /**
   * Render activity history timeline
   * Shows recent playback activity in a timeline format
   */
  renderActivityHistory(data: TautulliHistoryRow[]): void {
    if (!this.hasData(data)) {
      this.showEmptyState('No activity history available');
      return;
    }

    // Group by date and hour
    const byHour = new Map<string, number>();
    data.forEach(item => {
      const date = new Date(item.date * 1000);
      const hourKey = `${date.toLocaleDateString()} ${date.getHours()}:00`;
      byHour.set(hourKey, (byHour.get(hourKey) || 0) + 1);
    });

    const sortedKeys = Array.from(byHour.keys()).sort((a, b) => {
      const dateA = new Date(a.replace(' ', 'T'));
      const dateB = new Date(b.replace(' ', 'T'));
      return dateA.getTime() - dateB.getTime();
    });

    const timeLabels = sortedKeys.slice(-48); // Last 48 hours
    const playCounts = timeLabels.map(k => byHour.get(k) || 0);

    // Media type breakdown
    const mediaTypes = new Map<string, number>();
    data.forEach(item => {
      mediaTypes.set(item.media_type, (mediaTypes.get(item.media_type) || 0) + 1);
    });

    const totalPlays = data.length;
    const uniqueUsers = new Set(data.map(d => d.user_id)).size;

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Activity History',
        subtext: `${this.formatters.formatNumber(totalPlays)} plays | ${uniqueUsers} users`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 },
        subtextStyle: { color: STYLE_CONSTANTS.axisLabelColor, fontSize: 12 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: unknown) => {
          const p = params as { axisValue: string; value: number }[];
          if (!Array.isArray(p) || p.length === 0) return '';
          return `<strong>${p[0].axisValue}</strong><br/>
                  Plays: ${this.formatters.formatNumber(p[0].value)}`;
        }
      }),
      grid: {
        left: 60,
        right: 40,
        top: 100,
        bottom: 60
      },
      xAxis: {
        type: 'category' as const,
        data: timeLabels,
        axisLabel: {
          color: STYLE_CONSTANTS.axisLabelColor,
          rotate: 45,
          fontSize: 9,
          interval: Math.floor(timeLabels.length / 10)
        },
        axisLine: { lineStyle: { color: '#3a3a4e' } }
      },
      yAxis: {
        type: 'value' as const,
        name: 'Plays',
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
        axisLine: { lineStyle: { color: '#3a3a4e' } },
        splitLine: { lineStyle: { color: '#2a2a3e', type: 'dashed' as const } }
      },
      series: [
        {
          name: 'Activity',
          type: 'line' as const,
          data: playCounts,
          smooth: true,
          areaStyle: this.helpers.getAreaStyleGradient(CHART_COLORS.primary),
          lineStyle: { color: CHART_COLORS.primary, width: 2 },
          itemStyle: { color: CHART_COLORS.primary },
          showSymbol: false
        }
      ]
    };

    this.setOption(option, { notMerge: true });
  }

  /**
   * Render plays per month chart
   * Shows monthly playback statistics by media type
   */
  renderPlaysPerMonth(data: TautulliPlaysPerMonthData): void {
    if (!this.hasData(data?.categories) || !this.hasData(data?.series)) {
      this.showEmptyState('No plays per month data available');
      return;
    }

    // Calculate total plays
    const totalPlays = data.series.reduce((sum, series) => {
      return sum + series.data.reduce((s, v) => s + (v || 0), 0);
    }, 0);

    // Create series with stacked bars
    const seriesData = data.series.map((s, index) => ({
      name: s.name,
      type: 'bar' as const,
      stack: 'plays',
      data: s.data,
      itemStyle: { color: this.getColorByIndex(index) },
      emphasis: {
        focus: 'series' as const
      }
    }));

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Plays Per Month',
        subtext: `${this.formatters.formatNumber(totalPlays)} total plays`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 },
        subtextStyle: { color: STYLE_CONSTANTS.axisLabelColor, fontSize: 12 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' }
      }),
      legend: this.getLegend({
        data: data.series.map(s => s.name),
        bottom: 10
      }),
      grid: {
        left: 60,
        right: 40,
        top: 100,
        bottom: 60
      },
      xAxis: {
        type: 'category' as const,
        data: data.categories,
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor, rotate: 45 },
        axisLine: { lineStyle: { color: '#3a3a4e' } }
      },
      yAxis: {
        type: 'value' as const,
        name: 'Plays',
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
        axisLine: { lineStyle: { color: '#3a3a4e' } },
        splitLine: { lineStyle: { color: '#2a2a3e', type: 'dashed' as const } }
      },
      series: seriesData
    };

    this.setOption(option, { notMerge: true });
  }

  /**
   * Render concurrent streams by type chart
   * Shows distribution of concurrent streams by transcode decision
   */
  renderConcurrentStreamsByType(data: TautulliConcurrentStreamsByType[]): void {
    if (!this.hasData(data)) {
      this.showEmptyState('No concurrent streams data available');
      return;
    }

    // Calculate totals
    const totalMax = data.reduce((sum, d) => sum + d.max_concurrent, 0);
    const avgConcurrent = data.reduce((sum, d) => sum + d.avg_concurrent, 0) / data.length;

    // Prepare pie chart data
    const pieData = data.map((d, index) => ({
      name: d.transcode_decision,
      value: d.percentage,
      itemStyle: { color: this.getColorByIndex(index) }
    }));

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Concurrent Streams by Type',
        subtext: `Peak: ${totalMax} | Avg: ${avgConcurrent.toFixed(1)}`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 },
        subtextStyle: { color: STYLE_CONSTANTS.axisLabelColor, fontSize: 12 }
      },
      tooltip: this.getTooltip({
        trigger: 'item',
        formatter: (params: unknown) => {
          const p = params as { name: string; value: number; dataIndex: number };
          const item = data[p.dataIndex];
          if (!item) return '';
          return `<strong>${item.transcode_decision}</strong><br/>
                  Percentage: ${item.percentage.toFixed(1)}%<br/>
                  Max Concurrent: ${item.max_concurrent}<br/>
                  Avg Concurrent: ${item.avg_concurrent.toFixed(2)}`;
        }
      }),
      legend: this.getLegend({
        data: data.map(d => d.transcode_decision),
        bottom: 10
      }),
      series: [
        {
          name: 'Stream Type',
          type: 'pie' as const,
          radius: ['40%', '70%'],
          center: ['50%', '55%'],
          avoidLabelOverlap: true,
          label: {
            show: true,
            formatter: '{b}: {d}%',
            color: STYLE_CONSTANTS.textColor
          },
          labelLine: {
            show: true,
            lineStyle: { color: '#3a3a4e' }
          },
          data: pieData,
          emphasis: {
            itemStyle: {
              shadowBlur: 10,
              shadowOffsetX: 0,
              shadowColor: 'rgba(0, 0, 0, 0.5)'
            }
          }
        }
      ]
    };

    this.setOption(option, { notMerge: true });
  }

  /**
   * Get color by index for consistent coloring
   */
  private getColorByIndex(index: number): string {
    const colors = [
      CHART_COLORS.primary,
      CHART_COLORS.secondary,
      CHART_COLORS.accent,
      '#ff7c43',
      '#00bfa5',
      '#7c3aed',
      '#f59e0b',
      '#10b981',
      '#6366f1',
      '#ec4899'
    ];
    return colors[index % colors.length];
  }
}
