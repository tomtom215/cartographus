// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Renderer for content discovery analytics charts
 * Visualizes time-to-first-watch metrics, discovery rates, early adopters, and stale content
 */

import type { ContentDiscoveryAnalytics } from '../../types';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS, STYLE_CONSTANTS } from '../config/colors';
import type {
  EChartsCallbackDataParams,
  EChartsItemStyleColorParams,
  EChartsLabelFormatterParams,
} from '../types';

export class ContentDiscoveryChartRenderer extends BaseChartRenderer {
  render(_data: ContentDiscoveryAnalytics): void {
    // Base render method - unused, use specific render methods instead
  }

  /**
   * Render summary metrics as gauge/stat cards
   */
  renderSummary(data: ContentDiscoveryAnalytics): void {
    if (!data?.summary) {
      this.showEmptyState('No content discovery data available');
      return;
    }

    const summary = data.summary;
    const metrics = [
      { name: 'Total Content', value: summary.total_content_with_added_at },
      { name: 'Discovered', value: summary.total_discovered },
      { name: 'Never Watched', value: summary.total_never_watched },
      { name: 'Discovery Rate', value: Math.round(summary.overall_discovery_rate), suffix: '%' },
      { name: 'Avg Time (hrs)', value: Math.round(summary.avg_time_to_discovery_hours) },
      { name: 'Early Discovery', value: Math.round(summary.early_discovery_rate), suffix: '%' },
    ];

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const metric = metrics[params[0].dataIndex];
          const suffix = metric.suffix || '';
          return `<div style="font-weight: 600; margin-bottom: 8px;">${metric.name}</div>
            <div style="margin: 4px 0;">
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600; font-size: 16px;">${this.formatters.formatNumber(metric.value)}${suffix}</span>
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
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          color: ((params: EChartsItemStyleColorParams) => {
            const colors = [CHART_COLORS.primary, CHART_COLORS.success, CHART_COLORS.warning, CHART_COLORS.accent, CHART_COLORS.orange, CHART_COLORS.secondary];
            return colors[params.dataIndex % colors.length];
          }) as any,
          borderRadius: this.helpers.getBarBorderRadius('vertical')
        },
        label: {
          show: true,
          position: 'top' as const,
          color: STYLE_CONSTANTS.textColor,
          fontSize: 11,
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          formatter: ((params: EChartsLabelFormatterParams) => {
            const metric = metrics[params.dataIndex];
            return `${params.value}${metric.suffix || ''}`;
          }) as any
        }
      }]
    };

    this.setOption(option);
  }

  /**
   * Render discovery time buckets as a funnel/bar chart
   */
  renderTimeBuckets(data: ContentDiscoveryAnalytics): void {
    if (!this.hasData(data?.time_buckets)) {
      this.showEmptyState('No time bucket data available');
      return;
    }

    const buckets = data.time_buckets;

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'item',
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        formatter: ((params: EChartsCallbackDataParams) => {
          const bucket = buckets[params.dataIndex];
          if (!bucket) return '';
          return `<div style="font-weight: 600; margin-bottom: 8px;">${bucket.bucket}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Content Items:</span>
              <span style="color: ${params.color}; font-weight: 600;"> ${bucket.content_count}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Percentage:</span>
              <span style="font-weight: 600;"> ${bucket.percentage.toFixed(1)}%</span>
            </div>`;
        }) as any
      }),
      legend: this.getLegend({ show: false }),
      grid: { left: '15%', right: '10%', bottom: '10%', top: '10%' },
      xAxis: this.getValueAxis({ name: 'Content Count' }),
      yAxis: this.getCategoryAxis(buckets.map(b => b.bucket), { inverse: true }),
      series: [{
        type: 'bar' as const,
        data: buckets.map((b, idx) => ({
          value: b.content_count,
          itemStyle: {
            color: idx === 0 ? CHART_COLORS.success :
                   idx === 1 ? CHART_COLORS.primary :
                   idx === 2 ? CHART_COLORS.warning :
                   idx === 3 ? CHART_COLORS.orange :
                   CHART_COLORS.danger,
            borderRadius: this.helpers.getBarBorderRadius('horizontal')
          }
        })),
        label: {
          show: true,
          position: 'right' as const,
          color: STYLE_CONSTANTS.textColor,
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          formatter: ((params: EChartsLabelFormatterParams) => {
            const bucket = buckets[params.dataIndex];
            return `${params.value} (${bucket.percentage.toFixed(1)}%)`;
          }) as any
        }
      }]
    };

    this.setOption(option);
  }

  /**
   * Render early adopters as a leaderboard
   */
  renderEarlyAdopters(data: ContentDiscoveryAnalytics): void {
    if (!this.hasData(data?.early_adopters)) {
      this.showEmptyState('No early adopter data available');
      return;
    }

    const adopters = data.early_adopters.slice(0, 10);
    const usernames = adopters.map(a => a.username);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const adopter = adopters[params[0].dataIndex];
          if (!adopter) return '';

          return `<div style="font-weight: 600; margin-bottom: 8px;">${adopter.username}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Early Discoveries:</span>
              <span style="color: ${CHART_COLORS.success}; font-weight: 600;"> ${adopter.early_discovery_count}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Total Discoveries:</span>
              <span style="font-weight: 600;"> ${adopter.total_discoveries}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Early Rate:</span>
              <span style="font-weight: 600;"> ${adopter.early_discovery_rate.toFixed(1)}%</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Avg Time to Discover:</span>
              <span style="font-weight: 600;"> ${adopter.avg_time_to_discovery_hours.toFixed(1)}h</span>
            </div>
            ${adopter.favorite_library ? `<div style="margin: 4px 0;">
              <span style="font-weight: 500;">Favorite Library:</span>
              <span style="font-weight: 600;"> ${adopter.favorite_library}</span>
            </div>` : ''}`;
        }
      }),
      legend: this.getLegend({ data: ['Early Discoveries', 'Total Discoveries'] }),
      grid: { left: '15%', right: '10%', bottom: '10%', top: '15%' },
      xAxis: this.getValueAxis(),
      yAxis: this.getCategoryAxis(usernames, { inverse: true }),
      series: [
        {
          name: 'Early Discoveries',
          type: 'bar' as const,
          data: adopters.map(a => a.early_discovery_count),
          itemStyle: { color: CHART_COLORS.success, borderRadius: this.helpers.getBarBorderRadius('horizontal') },
          label: { show: true, position: 'right' as const, color: STYLE_CONSTANTS.textColor }
        },
        {
          name: 'Total Discoveries',
          type: 'bar' as const,
          data: adopters.map(a => a.total_discoveries),
          itemStyle: { color: CHART_COLORS.primary, borderRadius: this.helpers.getBarBorderRadius('horizontal') }
        }
      ]
    };

    this.setOption(option);
  }

  /**
   * Render library discovery stats as a comparison chart
   */
  renderLibraryStats(data: ContentDiscoveryAnalytics): void {
    if (!this.hasData(data?.library_stats)) {
      this.showEmptyState('No library statistics available');
      return;
    }

    const libraries = data.library_stats;
    const libraryNames = libraries.map(l => l.library_name);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const library = libraries[params[0].dataIndex];
          if (!library) return '';

          return `<div style="font-weight: 600; margin-bottom: 8px;">${library.library_name}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Total Items:</span>
              <span style="font-weight: 600;"> ${library.total_items}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Watched:</span>
              <span style="color: ${CHART_COLORS.success}; font-weight: 600;"> ${library.watched_items}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Unwatched:</span>
              <span style="color: ${CHART_COLORS.warning}; font-weight: 600;"> ${library.unwatched_items}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Discovery Rate:</span>
              <span style="font-weight: 600;"> ${library.discovery_rate.toFixed(1)}%</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Avg Discovery Time:</span>
              <span style="font-weight: 600;"> ${library.avg_time_to_discovery_hours.toFixed(1)}h</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Early Discovery Rate:</span>
              <span style="font-weight: 600;"> ${library.early_discovery_rate.toFixed(1)}%</span>
            </div>`;
        }
      }),
      legend: this.getLegend({ data: ['Watched', 'Unwatched'] }),
      grid: { left: '15%', right: '10%', bottom: '15%', top: '15%' },
      xAxis: this.getCategoryAxis(libraryNames, {
        axisLabel: { rotate: 30, fontSize: 10 }
      }),
      yAxis: this.getValueAxis({ name: 'Items' }),
      series: [
        {
          name: 'Watched',
          type: 'bar' as const,
          stack: 'total',
          data: libraries.map(l => l.watched_items),
          itemStyle: { color: CHART_COLORS.success }
        },
        {
          name: 'Unwatched',
          type: 'bar' as const,
          stack: 'total',
          data: libraries.map(l => l.unwatched_items),
          itemStyle: { color: CHART_COLORS.warning }
        }
      ]
    };

    this.setOption(option);
  }

  /**
   * Render discovery trends over time
   */
  renderTrends(data: ContentDiscoveryAnalytics): void {
    if (!this.hasData(data?.trends)) {
      this.showEmptyState('No trend data available');
      return;
    }

    const trends = data.trends.slice().reverse(); // Oldest first
    const dates = trends.map(t => {
      const date = new Date(t.date);
      return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
    });

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'cross' as const },
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const trend = trends[params[0].dataIndex];
          if (!trend) return '';

          return `<div style="font-weight: 600; margin-bottom: 8px;">${params[0].axisValue}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Content Added:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;"> ${trend.content_added}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Discovered:</span>
              <span style="color: ${CHART_COLORS.success}; font-weight: 600;"> ${trend.content_discovered}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Discovery Rate:</span>
              <span style="font-weight: 600;"> ${trend.discovery_rate.toFixed(1)}%</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Avg Time to Discovery:</span>
              <span style="font-weight: 600;"> ${trend.avg_time_to_discovery_hours.toFixed(1)}h</span>
            </div>`;
        }
      }),
      legend: this.getAccessibleLegend(['Content Added', 'Discovered', 'Discovery Rate'], {
        bottom: 0
      }),
      grid: { left: '10%', right: '10%', bottom: '20%', top: '10%' },
      xAxis: this.getCategoryAxis(dates),
      yAxis: [
        this.getValueAxis({ name: 'Count', position: 'left' as const }),
        this.getValueAxis({ name: 'Rate (%)', position: 'right' as const, max: 100, splitLine: { show: false } })
      ],
      dataZoom: this.getDataZoom('both'),
      series: [
        {
          name: 'Content Added',
          type: 'bar' as const,
          yAxisIndex: 0,
          data: trends.map(t => t.content_added),
          itemStyle: { color: CHART_COLORS.primary, borderRadius: this.helpers.getBarBorderRadius('vertical') },
          symbol: this.getSeriesIcon(0),
        },
        {
          name: 'Discovered',
          type: 'bar' as const,
          yAxisIndex: 0,
          data: trends.map(t => t.content_discovered),
          itemStyle: { color: CHART_COLORS.success, borderRadius: this.helpers.getBarBorderRadius('vertical') },
          symbol: this.getSeriesIcon(1),
        },
        {
          name: 'Discovery Rate',
          type: 'line' as const,
          yAxisIndex: 1,
          data: trends.map(t => t.discovery_rate),
          itemStyle: { color: CHART_COLORS.accent },
          symbol: this.getSeriesIcon(2),
          symbolSize: 8,
          smooth: true
        }
      ]
    };

    this.setOption(option);
  }

  /**
   * Render stale content list as a horizontal bar chart
   */
  renderStaleContent(data: ContentDiscoveryAnalytics): void {
    if (!this.hasData(data?.stale_content)) {
      this.showEmptyState('No stale content - everything has been watched!');
      return;
    }

    const stale = data.stale_content.slice(0, 15);
    const titles = stale.map(s => s.title.length > 30 ? s.title.substring(0, 27) + '...' : s.title);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const content = stale[params[0].dataIndex];
          if (!content) return '';

          const addedDate = new Date(content.added_at);
          return `<div style="font-weight: 600; margin-bottom: 8px;">${content.title}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Type:</span>
              <span style="font-weight: 600;"> ${content.media_type}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Library:</span>
              <span style="font-weight: 600;"> ${content.library_name}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Added:</span>
              <span style="font-weight: 600;"> ${addedDate.toLocaleDateString()}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Days Since Added:</span>
              <span style="color: ${CHART_COLORS.danger}; font-weight: 600;"> ${content.days_since_added}</span>
            </div>
            ${content.year ? `<div style="margin: 4px 0;">
              <span style="font-weight: 500;">Year:</span>
              <span style="font-weight: 600;"> ${content.year}</span>
            </div>` : ''}
            ${content.genres ? `<div style="margin: 4px 0;">
              <span style="font-weight: 500;">Genres:</span>
              <span style="font-weight: 600;"> ${content.genres}</span>
            </div>` : ''}`;
        }
      }),
      grid: { left: '25%', right: '10%', bottom: '10%', top: '10%' },
      xAxis: this.getValueAxis({ name: 'Days Since Added' }),
      yAxis: this.getCategoryAxis(titles, { inverse: true }),
      series: [{
        type: 'bar' as const,
        data: stale.map(s => ({
          value: s.days_since_added,
          itemStyle: {
            color: s.days_since_added > 180 ? CHART_COLORS.danger :
                   s.days_since_added > 120 ? CHART_COLORS.warning :
                   CHART_COLORS.orange,
            borderRadius: this.helpers.getBarBorderRadius('horizontal')
          }
        })),
        label: {
          show: true,
          position: 'right' as const,
          color: STYLE_CONSTANTS.textColor,
          formatter: '{c} days'
        }
      }]
    };

    this.setOption(option);
  }

  /**
   * Render recently discovered content
   */
  renderRecentlyDiscovered(data: ContentDiscoveryAnalytics): void {
    if (!this.hasData(data?.recently_discovered)) {
      this.showEmptyState('No recently discovered content');
      return;
    }

    const discovered = data.recently_discovered.slice(0, 15);
    const titles = discovered.map(d => d.title.length > 25 ? d.title.substring(0, 22) + '...' : d.title);

    // Color by discovery velocity
    const velocityColors: Record<string, string> = {
      'fast': CHART_COLORS.success,
      'medium': CHART_COLORS.warning,
      'slow': CHART_COLORS.orange,
      'not_discovered': CHART_COLORS.danger
    };

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const content = discovered[params[0].dataIndex];
          if (!content) return '';

          const addedDate = new Date(content.added_at);
          const watchedDate = content.first_watched_at ? new Date(content.first_watched_at) : null;

          let html = `<div style="font-weight: 600; margin-bottom: 8px;">${content.title}</div>`;
          html += `<div style="margin: 4px 0;">
            <span style="font-weight: 500;">Type:</span>
            <span style="font-weight: 600;"> ${content.media_type}</span>
          </div>`;
          html += `<div style="margin: 4px 0;">
            <span style="font-weight: 500;">Library:</span>
            <span style="font-weight: 600;"> ${content.library_name}</span>
          </div>`;
          html += `<div style="margin: 4px 0;">
            <span style="font-weight: 500;">Added:</span>
            <span style="font-weight: 600;"> ${addedDate.toLocaleDateString()}</span>
          </div>`;
          if (watchedDate) {
            html += `<div style="margin: 4px 0;">
              <span style="font-weight: 500;">First Watched:</span>
              <span style="font-weight: 600;"> ${watchedDate.toLocaleDateString()}</span>
            </div>`;
          }
          if (content.time_to_first_watch_hours !== undefined) {
            html += `<div style="margin: 4px 0;">
              <span style="font-weight: 500;">Time to Discovery:</span>
              <span style="color: ${velocityColors[content.discovery_velocity]}; font-weight: 600;"> ${content.time_to_first_watch_hours.toFixed(1)}h</span>
            </div>`;
          }
          html += `<div style="margin: 4px 0;">
            <span style="font-weight: 500;">Velocity:</span>
            <span style="color: ${velocityColors[content.discovery_velocity]}; font-weight: 600;"> ${content.discovery_velocity.toUpperCase()}</span>
          </div>`;
          html += `<div style="margin: 4px 0;">
            <span style="font-weight: 500;">Playbacks:</span>
            <span style="font-weight: 600;"> ${content.total_playbacks}</span>
          </div>`;
          html += `<div style="margin: 4px 0;">
            <span style="font-weight: 500;">Viewers:</span>
            <span style="font-weight: 600;"> ${content.unique_viewers}</span>
          </div>`;

          return html;
        }
      }),
      grid: { left: '20%', right: '10%', bottom: '10%', top: '10%' },
      xAxis: this.getValueAxis({ name: 'Hours to Discovery' }),
      yAxis: this.getCategoryAxis(titles, { inverse: true }),
      series: [{
        type: 'bar' as const,
        data: discovered.map(d => ({
          value: d.time_to_first_watch_hours || 0,
          itemStyle: {
            color: velocityColors[d.discovery_velocity],
            borderRadius: this.helpers.getBarBorderRadius('horizontal')
          }
        })),
        label: {
          show: true,
          position: 'right' as const,
          color: STYLE_CONSTANTS.textColor,
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          formatter: ((params: EChartsLabelFormatterParams) => {
            const content = discovered[params.dataIndex];
            const value = params.value as number;
            return `${value.toFixed(0)}h (${content.discovery_velocity})`;
          }) as any
        }
      }]
    };

    this.setOption(option);
  }
}
