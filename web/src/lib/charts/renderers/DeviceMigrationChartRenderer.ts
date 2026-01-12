// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Renderer for device migration tracking analytics charts
 * Visualizes platform adoption, migration patterns, and user device profiles
 */

import type { DeviceMigrationAnalytics } from '../../types';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS, STYLE_CONSTANTS } from '../config/colors';
import type { EChartsCallbackDataParams, EChartsItemStyleColorParams } from '../types';

export class DeviceMigrationChartRenderer extends BaseChartRenderer {
  render(_data: DeviceMigrationAnalytics): void {
    // Base render method - unused, use specific render methods instead
  }

  /**
   * Render summary metrics as a bar chart
   */
  renderSummary(data: DeviceMigrationAnalytics): void {
    if (!data?.summary) {
      this.showEmptyState('No device migration data available');
      return;
    }

    const summary = data.summary;
    const metrics = [
      { name: 'Total Users', value: summary.total_users },
      { name: 'Multi-Device Users', value: summary.multi_device_users },
      { name: 'Multi-Device %', value: Math.round(summary.multi_device_percentage) },
      { name: 'Total Migrations', value: summary.total_migrations },
      { name: 'Avg Platforms/User', value: Math.round(summary.avg_platforms_per_user * 10) / 10 },
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
      grid: { left: '10%', right: '10%', bottom: '20%', top: '10%' },
      xAxis: this.getCategoryAxis(metrics.map(m => m.name), {
        axisLabel: { rotate: 30, fontSize: 10 }
      }),
      yAxis: this.getValueAxis(),
      series: [{
        type: 'bar' as const,
        data: metrics.map(m => m.value),
        itemStyle: {
          color: (params: EChartsItemStyleColorParams) => {
            const colors = [CHART_COLORS.primary, CHART_COLORS.success, CHART_COLORS.warning, CHART_COLORS.accent, CHART_COLORS.orange];
            return colors[params.dataIndex % colors.length];
          },
          borderRadius: this.helpers.getBarBorderRadius('vertical')
        },
        label: { show: true, position: 'top' as const, color: STYLE_CONSTANTS.textColor, fontSize: 11 }
      }]
    };

    this.setOption(option);
  }

  /**
   * Render platform distribution as a pie chart
   */
  renderPlatformDistribution(data: DeviceMigrationAnalytics): void {
    if (!this.hasData(data?.platform_distribution)) {
      this.showEmptyState('No platform distribution data available');
      return;
    }

    const platformData = data.platform_distribution.map(p => ({
      name: p.platform,
      value: p.playback_count
    }));

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'item',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          const p = Array.isArray(params) ? params[0] : params;
          const platform = data.platform_distribution.find(pl => pl.platform === p.name);
          if (!platform) return '';
          const percent = p.percent ?? 0;
          return `<div style="font-weight: 600; margin-bottom: 8px;">${p.name}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Playbacks:</span>
              <span style="color: ${p.color}; font-weight: 600;"> ${this.formatters.formatNumber(platform.playback_count)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Users:</span>
              <span style="font-weight: 600;"> ${platform.unique_users}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Share:</span>
              <span style="font-weight: 600;"> ${percent.toFixed(1)}%</span>
            </div>`;
        }
      }),
      legend: this.getLegend({
        type: 'scroll',
        orient: 'vertical',
        right: 10,
        top: 20,
        bottom: 20
      }),
      series: [{
        type: 'pie' as const,
        radius: ['40%', '70%'],
        center: ['40%', '50%'],
        data: platformData,
        label: {
          show: true,
          position: 'outside' as const,
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

    this.setOption(option);
  }

  /**
   * Render common platform transitions as a Sankey diagram
   */
  renderTransitions(data: DeviceMigrationAnalytics): void {
    if (!this.hasData(data?.common_transitions)) {
      this.showEmptyState('No transition data available');
      return;
    }

    // Create unique nodes from transitions
    const nodeSet = new Set<string>();
    data.common_transitions.forEach(t => {
      nodeSet.add(t.from_platform);
      nodeSet.add(t.to_platform);
    });
    const nodes = Array.from(nodeSet).map(name => ({ name }));

    // Create links
    const links = data.common_transitions.map(t => ({
      source: t.from_platform,
      target: t.to_platform,
      value: t.transition_count
    }));

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'item',
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        formatter: (params: any) => {
          if (params.dataType === 'edge') {
            const transition = data.common_transitions.find(
              t => t.from_platform === params.data?.source && t.to_platform === params.data?.target
            );
            if (!transition) return '';
            return `<div style="font-weight: 600; margin-bottom: 8px;">${params.data?.source} to ${params.data?.target}</div>
              <div style="margin: 4px 0;">
                <span style="font-weight: 500;">Transitions:</span>
                <span style="font-weight: 600;"> ${transition.transition_count}</span>
              </div>
              <div style="margin: 4px 0;">
                <span style="font-weight: 500;">Unique Users:</span>
                <span style="font-weight: 600;"> ${transition.unique_users}</span>
              </div>
              <div style="margin: 4px 0;">
                <span style="font-weight: 500;">Avg Days Before Switch:</span>
                <span style="font-weight: 600;"> ${transition.avg_days_before_switch.toFixed(1)}</span>
              </div>
              <div style="margin: 4px 0;">
                <span style="font-weight: 500;">Return Rate:</span>
                <span style="font-weight: 600;"> ${transition.return_rate.toFixed(1)}%</span>
              </div>`;
          }
          return `${params.name}`;
        }
      }),
      series: [{
        type: 'sankey' as const,
        emphasis: { focus: 'adjacency' as const },
        data: nodes,
        links: links,
        lineStyle: {
          color: 'gradient',
          curveness: 0.5
        },
        label: {
          color: STYLE_CONSTANTS.textColor
        }
      }]
    };

    this.setOption(option);
  }

  /**
   * Render adoption trends as a stacked area chart
   */
  renderAdoptionTrends(data: DeviceMigrationAnalytics): void {
    if (!this.hasData(data?.adoption_trends)) {
      this.showEmptyState('No adoption trend data available');
      return;
    }

    // Group by platform and date
    const platformMap = new Map<string, Map<string, number>>();
    const dates = new Set<string>();

    data.adoption_trends.forEach(t => {
      dates.add(t.date);
      if (!platformMap.has(t.platform)) {
        platformMap.set(t.platform, new Map());
      }
      platformMap.get(t.platform)!.set(t.date, t.session_count);
    });

    const sortedDates = Array.from(dates).sort();
    const platforms = Array.from(platformMap.keys());

    const series = platforms.map((platform, idx) => {
      const platformData = platformMap.get(platform)!;
      return {
        name: platform,
        type: 'line' as const,
        stack: 'Total',
        areaStyle: { opacity: 0.4 },
        emphasis: { focus: 'series' as const },
        data: sortedDates.map(d => platformData.get(d) || 0),
        smooth: true,
        symbol: this.getSeriesIcon(idx),
        symbolSize: 6
      };
    });

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'cross' as const }
      }),
      legend: this.getAccessibleLegend(platforms, {
        type: 'scroll',
        bottom: 0
      }),
      grid: { left: '10%', right: '10%', bottom: '15%', top: '10%' },
      xAxis: this.getCategoryAxis(sortedDates.map(d => {
        const date = new Date(d);
        return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
      })),
      yAxis: this.getValueAxis({ name: 'Sessions' }),
      dataZoom: this.getDataZoom('both'),
      series
    };

    this.setOption(option);
  }

  /**
   * Render user device profiles as a table-like bar chart
   */
  renderUserProfiles(data: DeviceMigrationAnalytics): void {
    if (!this.hasData(data?.top_user_profiles)) {
      this.showEmptyState('No user profile data available');
      return;
    }

    const profiles = data.top_user_profiles.slice(0, 10);
    const usernames = profiles.map(p => p.username);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const profile = profiles[params[0].dataIndex];
          if (!profile) return '';

          let html = `<div style="font-weight: 600; margin-bottom: 8px;">${profile.username}</div>`;
          html += `<div style="margin: 4px 0;">
            <span style="font-weight: 500;">Platforms:</span>
            <span style="font-weight: 600;"> ${profile.total_platforms_used}</span>
          </div>`;
          html += `<div style="margin: 4px 0;">
            <span style="font-weight: 500;">Primary:</span>
            <span style="font-weight: 600;"> ${profile.primary_platform} (${profile.primary_platform_percentage.toFixed(1)}%)</span>
          </div>`;
          html += `<div style="margin: 4px 0;">
            <span style="font-weight: 500;">Sessions:</span>
            <span style="font-weight: 600;"> ${profile.total_sessions}</span>
          </div>`;
          html += `<div style="margin: 4px 0;">
            <span style="font-weight: 500;">Migrations:</span>
            <span style="font-weight: 600;"> ${profile.total_migrations}</span>
          </div>`;

          return html;
        }
      }),
      legend: this.getLegend({ data: ['Sessions', 'Platforms', 'Migrations'] }),
      grid: { left: '15%', right: '10%', bottom: '10%', top: '15%' },
      xAxis: this.getValueAxis(),
      yAxis: this.getCategoryAxis(usernames, { inverse: true }),
      series: [
        {
          name: 'Sessions',
          type: 'bar' as const,
          data: profiles.map(p => p.total_sessions),
          itemStyle: { color: CHART_COLORS.primary, borderRadius: this.helpers.getBarBorderRadius('horizontal') }
        },
        {
          name: 'Platforms',
          type: 'bar' as const,
          data: profiles.map(p => p.total_platforms_used * 50), // Scale for visibility
          itemStyle: { color: CHART_COLORS.success, borderRadius: this.helpers.getBarBorderRadius('horizontal') }
        },
        {
          name: 'Migrations',
          type: 'bar' as const,
          data: profiles.map(p => p.total_migrations * 10), // Scale for visibility
          itemStyle: { color: CHART_COLORS.warning, borderRadius: this.helpers.getBarBorderRadius('horizontal') }
        }
      ]
    };

    this.setOption(option);
  }

  /**
   * Render recent migrations as a timeline
   */
  renderRecentMigrations(data: DeviceMigrationAnalytics): void {
    if (!this.hasData(data?.recent_migrations)) {
      this.showEmptyState('No recent migration data available');
      return;
    }

    const migrations = data.recent_migrations.slice(0, 15);
    const labels = migrations.map((_, i) => `#${i + 1}`);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const migration = migrations[params[0].dataIndex];
          if (!migration) return '';

          const date = new Date(migration.migration_date);
          return `<div style="font-weight: 600; margin-bottom: 8px;">${migration.username}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">From:</span>
              <span style="font-weight: 600;"> ${migration.from_platform}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">To:</span>
              <span style="font-weight: 600;"> ${migration.to_platform}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Date:</span>
              <span style="font-weight: 600;"> ${date.toLocaleDateString()}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Sessions Before:</span>
              <span style="font-weight: 600;"> ${migration.sessions_before_migration}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Sessions After:</span>
              <span style="font-weight: 600;"> ${migration.sessions_after_migration}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Permanent:</span>
              <span style="font-weight: 600;"> ${migration.is_permanent_switch ? 'Yes' : 'No'}</span>
            </div>`;
        }
      }),
      grid: { left: '10%', right: '10%', bottom: '10%', top: '10%' },
      xAxis: this.getCategoryAxis(labels),
      yAxis: [
        this.getValueAxis({ name: 'Sessions Before', position: 'left' as const }),
        this.getValueAxis({ name: 'Sessions After', position: 'right' as const, splitLine: { show: false } })
      ],
      series: [
        {
          name: 'Sessions Before',
          type: 'bar' as const,
          yAxisIndex: 0,
          data: migrations.map(m => m.sessions_before_migration),
          itemStyle: { color: CHART_COLORS.warning, borderRadius: this.helpers.getBarBorderRadius('vertical') }
        },
        {
          name: 'Sessions After',
          type: 'bar' as const,
          yAxisIndex: 1,
          data: migrations.map(m => m.sessions_after_migration),
          itemStyle: { color: CHART_COLORS.success, borderRadius: this.helpers.getBarBorderRadius('vertical') }
        }
      ]
    };

    this.setOption(option);
  }
}
