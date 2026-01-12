// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Renderer for geographic and content analytics charts
 * Handles countries, cities, media types, heatmap, platforms, players, completion,
 * transcode, resolution, codecs, libraries, ratings, duration, and years
 */

import type { GeographicResponse } from '../../api';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS, COLOR_PALETTES, STYLE_CONSTANTS } from '../config/colors';
import { getGridConfig } from '../config/grid';
import type {
  EChartsCallbackDataParams,
  EChartsItemStyleColorParams,
  EChartsLabelFormatterParams,
} from '../types';

export class GeographicChartRenderer extends BaseChartRenderer {
  render(_data: GeographicResponse): void {
    // Base render method - unused in this renderer
  }

  renderCountries(data: GeographicResponse): void {
    if (!this.hasData(data?.top_countries)) {
      this.showEmptyState('No country data available');
      return;
    }
    const countries = data.top_countries.map(c => c.country);
    const counts = data.top_countries.map(c => c.playback_count);
    const total = counts.reduce((sum, count) => sum + count, 0);

    const [reversedCountries, reversedCounts] = this.helpers.reverseArrays(countries, counts);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: this.createBarTooltipFormatter(total),
      }),
      grid: getGridConfig('default'),
      xAxis: this.getValueAxis(),
      yAxis: this.getCategoryAxis(reversedCountries),
      series: [
        {
          type: 'bar' as const,
          data: reversedCounts,
          itemStyle: {
            color: this.helpers.createLinearGradient('horizontal', COLOR_PALETTES.gradient.redToBlue),
            borderRadius: this.helpers.getBarBorderRadius('horizontal'),
          },
          label: {
            show: true,
            position: 'right' as const,
            color: STYLE_CONSTANTS.textColor,
          },
        },
      ],
    };

    this.setOption(option);
  }

  renderCities(data: GeographicResponse): void {
    if (!this.hasData(data?.top_cities)) {
      this.showEmptyState('No city data available');
      return;
    }
    const cities = data.top_cities.map(c => c.city);
    const counts = data.top_cities.map(c => c.playback_count);
    const total = counts.reduce((sum, count) => sum + count, 0);

    const [reversedCities, reversedCounts] = this.helpers.reverseArrays(cities, counts);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: this.createBarTooltipFormatter(total),
      }),
      grid: getGridConfig('default'),
      xAxis: this.getValueAxis(),
      yAxis: this.getCategoryAxis(reversedCities),
      series: [
        {
          type: 'bar' as const,
          data: reversedCounts,
          itemStyle: {
            color: this.helpers.createLinearGradient('horizontal', COLOR_PALETTES.gradient.blueToRed),
            borderRadius: this.helpers.getBarBorderRadius('horizontal'),
          },
          label: {
            show: true,
            position: 'right' as const,
            color: STYLE_CONSTANTS.textColor,
          },
        },
      ],
    };

    this.setOption(option);
  }

  renderMediaDistribution(data: GeographicResponse): void {
    if (!this.hasData(data?.media_type_distribution)) {
      this.showEmptyState('No media distribution data available');
      return;
    }
    const chartData = data.media_type_distribution.map(m => ({
      name: m.media_type,
      value: m.playback_count,
    }));

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'item',
        formatter: this.createPieTooltipFormatter(),
      }),
      legend: this.getLegend({
        orient: 'vertical' as const,
        left: 'left',
      }),
      series: [
        {
          type: 'pie' as const,
          radius: ['40%', '70%'],
          avoidLabelOverlap: false,
          itemStyle: {
            borderRadius: 8,
            borderColor: STYLE_CONSTANTS.borderColor,
            borderWidth: 2,
          },
          label: {
            show: true,
            color: STYLE_CONSTANTS.textColor,
          },
          emphasis: {
            label: {
              show: true,
              fontSize: 16,
              fontWeight: 'bold' as const,
            },
          },
          data: chartData,
          color: [...COLOR_PALETTES.default],
        },
      ],
    };

    this.setOption(option);
  }

  renderHeatmap(data: GeographicResponse): void {
    if (!this.hasData(data?.viewing_hours_heatmap)) {
      this.showEmptyState('No viewing hours data available');
      return;
    }
    const hours = Array.from({ length: 24 }, (_, i) => i.toString().padStart(2, '0'));
    const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

    const heatmapData = data.viewing_hours_heatmap.map(h => [h.hour, h.day_of_week, h.playback_count]);
    const maxValue = Math.max(...data.viewing_hours_heatmap.map(h => h.playback_count));

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        position: 'top' as const,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        formatter: ((params: EChartsCallbackDataParams) => {
          const value = params.value as number[];
          const hour = value[0];
          const day = days[value[1]];
          const count = this.formatters.formatNumber(value[2]);
          const percentage = this.formatters.formatPercent(
            this.helpers.calculatePercentage(value[2], maxValue)
          );
          return `<div style="font-weight: 600; margin-bottom: 8px;">${day} ${hour}:00</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Playbacks:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${count}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Activity Level:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${percentage} of peak</span>
            </div>`;
        }) as any,
      }),
      grid: getGridConfig('heatmap'),
      xAxis: {
        type: 'category' as const,
        data: hours,
        splitArea: { show: true },
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
      },
      yAxis: {
        type: 'category' as const,
        data: days,
        splitArea: { show: true },
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
      },
      visualMap: {
        min: 0,
        max: maxValue,
        calculable: true,
        orient: 'horizontal' as const,
        left: 'center',
        bottom: '0%',
        inRange: {
          color: [...COLOR_PALETTES.heatmap],
        },
        textStyle: {
          color: STYLE_CONSTANTS.axisLabelColor,
        },
      },
      series: [
        {
          type: 'heatmap' as const,
          data: heatmapData,
          label: { show: false },
          emphasis: {
            itemStyle: {
              shadowBlur: 10,
              shadowColor: 'rgba(233, 69, 96, 0.5)',
            },
          },
        },
      ],
    };

    this.setOption(option);
  }

  renderPlatforms(data: GeographicResponse): void {
    if (!this.hasData(data?.platform_distribution)) {
      this.showEmptyState('No platform data available');
      return;
    }
    const platforms = data.platform_distribution.map(p => p.platform);
    const counts = data.platform_distribution.map(p => p.playback_count);
    const total = counts.reduce((sum, count) => sum + count, 0);

    const [reversedPlatforms, reversedCounts] = this.helpers.reverseArrays(platforms, counts);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: this.createBarTooltipFormatter(total),
      }),
      grid: getGridConfig('default'),
      xAxis: this.getValueAxis(),
      yAxis: this.getCategoryAxis(reversedPlatforms),
      series: [
        {
          type: 'bar' as const,
          data: reversedCounts,
          itemStyle: {
            color: this.helpers.createLinearGradient('horizontal', COLOR_PALETTES.gradient.redToPurple),
            borderRadius: this.helpers.getBarBorderRadius('horizontal'),
          },
          label: {
            show: true,
            position: 'right' as const,
            color: STYLE_CONSTANTS.textColor,
          },
        },
      ],
    };

    this.setOption(option);
  }

  renderPlayers(data: GeographicResponse): void {
    if (!this.hasData(data?.player_distribution)) {
      this.showEmptyState('No player data available');
      return;
    }
    const chartData = data.player_distribution.map(p => ({
      name: p.player,
      value: p.playback_count,
    }));

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'item',
        formatter: this.createPieTooltipFormatter(),
      }),
      legend: this.getLegend({
        orient: 'vertical' as const,
        left: 'left',
      }),
      series: [
        {
          type: 'pie' as const,
          radius: ['40%', '70%'],
          avoidLabelOverlap: false,
          itemStyle: {
            borderRadius: 8,
            borderColor: STYLE_CONSTANTS.borderColor,
            borderWidth: 2,
          },
          label: {
            show: true,
            color: STYLE_CONSTANTS.textColor,
          },
          emphasis: {
            label: {
              show: true,
              fontSize: 16,
              fontWeight: 'bold' as const,
            },
          },
          data: chartData,
          color: [...COLOR_PALETTES.default, CHART_COLORS.warning],
        },
      ],
    };

    this.setOption(option);
  }

  renderCompletion(data: GeographicResponse): void {
    if (!this.hasData(data?.content_completion_stats?.buckets)) {
      this.showEmptyState('No completion data available');
      return;
    }
    const completionStats = data.content_completion_stats;
    const buckets = completionStats.buckets.map(b => b.bucket);
    const counts = completionStats.buckets.map(b => b.playback_count);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const param = params[0];
          const bucketData = completionStats.buckets[param.dataIndex];
          const value = this.formatters.formatNumber(param.value as number);
          const avgCompletion = this.formatters.formatPercent(bucketData.avg_completion);
          return `<div style="font-weight: 600; margin-bottom: 8px;">${param.name}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Playbacks:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${value}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Avg Completion:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${avgCompletion}</span>
            </div>`;
        },
      }),
      grid: getGridConfig('default'),
      xAxis: this.getCategoryAxis(buckets),
      yAxis: this.getValueAxis(),
      series: [
        {
          type: 'bar' as const,
          data: counts,
          itemStyle: {
            color: (params: EChartsItemStyleColorParams) => {
              return COLOR_PALETTES.completion[params.dataIndex] || CHART_COLORS.primary;
            },
            borderRadius: this.helpers.getBarBorderRadius('vertical'),
          },
          label: {
            show: true,
            position: 'top' as const,
            color: STYLE_CONSTANTS.textColor,
            formatter: (params: EChartsLabelFormatterParams) => {
              const pct = this.helpers.calculatePercentage(params.value as number, completionStats.total_playbacks);
              return `${params.value}\n(${pct.toFixed(1)}%)`;
            },
          },
        },
      ],
    };

    this.setOption(option);
  }

  renderTranscode(data: GeographicResponse): void {
    if (!this.hasData(data?.transcode_distribution)) {
      this.showEmptyState('No transcode data available');
      return;
    }
    const chartData = data.transcode_distribution.map(t => ({
      name: t.transcode_decision,
      value: t.playback_count,
    }));

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'item',
        formatter: this.createPieTooltipFormatter(),
      }),
      legend: this.getLegend({
        orient: 'horizontal' as const,
        bottom: '0%',
      }),
      series: [
        {
          type: 'pie' as const,
          radius: ['40%', '70%'],
          center: ['50%', '45%'],
          avoidLabelOverlap: false,
          itemStyle: {
            borderRadius: 8,
            borderColor: STYLE_CONSTANTS.borderColor,
            borderWidth: 2,
          },
          label: {
            show: true,
            color: STYLE_CONSTANTS.textColor,
            formatter: '{b}\n{d}%',
          },
          emphasis: {
            label: {
              show: true,
              fontSize: 16,
              fontWeight: 'bold' as const,
            },
          },
          data: chartData,
          color: [CHART_COLORS.success, CHART_COLORS.primary, CHART_COLORS.warning],
        },
      ],
    };

    this.setOption(option);
  }

  renderResolution(data: GeographicResponse): void {
    if (!this.hasData(data?.resolution_distribution)) {
      this.showEmptyState('No resolution data available');
      return;
    }
    const resolutions = data.resolution_distribution.map(r => r.video_resolution);
    const counts = data.resolution_distribution.map(r => r.playback_count);
    const total = counts.reduce((sum, count) => sum + count, 0);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: this.createBarTooltipFormatter(total),
      }),
      grid: getGridConfig('default'),
      xAxis: this.getCategoryAxis(resolutions),
      yAxis: this.getValueAxis(),
      series: [
        {
          type: 'bar' as const,
          data: counts,
          itemStyle: {
            color: this.helpers.createLinearGradient('vertical', COLOR_PALETTES.gradient.redToPurple),
            borderRadius: this.helpers.getBarBorderRadius('vertical'),
          },
          label: {
            show: true,
            position: 'top' as const,
            color: STYLE_CONSTANTS.textColor,
          },
        },
      ],
    };

    this.setOption(option);
  }

  renderCodec(data: GeographicResponse): void {
    if (!this.hasData(data?.codec_distribution)) {
      this.showEmptyState('No codec data available');
      return;
    }
    const labels = data.codec_distribution.map(c => `${c.video_codec}\n+\n${c.audio_codec}`);
    const counts = data.codec_distribution.map(c => c.playback_count);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const param = params[0];
          const codec = data.codec_distribution[param.dataIndex];
          const value = this.formatters.formatNumber(param.value as number);
          const percentage = this.formatters.formatPercent(codec.percentage);
          return `<div style="font-weight: 600; margin-bottom: 8px;">${codec.video_codec} + ${codec.audio_codec}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Playbacks:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${value}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Percentage:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${percentage}</span>
            </div>`;
        },
      }),
      grid: {
        left: '3%',
        right: '4%',
        bottom: '10%',
        containLabel: true,
      },
      xAxis: this.getCategoryAxis(labels, {
        interval: 0,
        rotate: 0,
        fontSize: 10,
      }),
      yAxis: this.getValueAxis(),
      series: [
        {
          type: 'bar' as const,
          data: counts,
          itemStyle: {
            color: (params: EChartsItemStyleColorParams) => {
              const colors = [...COLOR_PALETTES.default, CHART_COLORS.darker, CHART_COLORS.crimson, CHART_COLORS.dark];
              return colors[params.dataIndex % colors.length];
            },
            borderRadius: this.helpers.getBarBorderRadius('vertical'),
          },
          label: {
            show: true,
            position: 'top' as const,
            color: STYLE_CONSTANTS.textColor,
          },
        },
      ],
    };

    this.setOption(option);
  }

  renderLibraries(data: GeographicResponse): void {
    if (!this.hasData(data?.library_distribution)) {
      this.showEmptyState('No library data available');
      return;
    }
    const libraries = data.library_distribution;
    const libraryNames = libraries.map(l => l.library_name);
    const playbackCounts = libraries.map(l => l.playback_count);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const p = params[0];
          const lib = libraries[p.dataIndex];
          const watchTimeHours = Math.round(lib.total_duration_minutes / 60);
          return `<div style="font-weight: 600; margin-bottom: 8px;">${lib.library_name}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Playbacks:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(lib.playback_count)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Unique Users:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(lib.unique_users)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Watch Time:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(watchTimeHours)}h</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Avg Completion:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatPercent(lib.avg_completion)}</span>
            </div>`;
        },
      }),
      grid: {
        left: '15%',
        right: '10%',
        bottom: '10%',
        top: '10%',
      },
      xAxis: this.getValueAxis(),
      yAxis: this.getCategoryAxis(libraryNames),
      series: [
        {
          type: 'bar' as const,
          data: playbackCounts,
          itemStyle: {
            color: this.helpers.createLinearGradient('horizontal', COLOR_PALETTES.gradient.redToPurple),
            borderRadius: this.helpers.getBarBorderRadius('horizontal'),
          },
          label: {
            show: true,
            position: 'right' as const,
            color: STYLE_CONSTANTS.textColor,
          },
        },
      ],
    };

    this.setOption(option);
  }

  renderRatings(data: GeographicResponse): void {
    if (!this.hasData(data?.rating_distribution)) {
      this.showEmptyState('No rating data available');
      return;
    }
    const ratings = data.rating_distribution;
    const chartData = ratings.map(r => ({
      name: r.content_rating,
      value: r.playback_count,
    }));

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'item',
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        formatter: ((params: EChartsCallbackDataParams) => {
          const value = this.formatters.formatNumber(params.value as number);
          const rating = ratings.find(r => r.content_rating === params.name);
          const percentage = rating ? this.formatters.formatPercent(rating.percentage) : '0%';
          return `<div style="font-weight: 600; margin-bottom: 8px;">${params.name}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Playbacks:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${value}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Percentage:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${percentage}</span>
            </div>`;
        }) as any,
      }),
      legend: this.getLegend({
        orient: 'vertical' as const,
        left: 'left',
      }),
      series: [
        {
          type: 'pie' as const,
          radius: ['40%', '70%'],
          avoidLabelOverlap: false,
          itemStyle: {
            borderRadius: 10,
            borderColor: STYLE_CONSTANTS.borderColor,
            borderWidth: 2,
          },
          label: {
            show: true,
            formatter: '{b}: {d}%',
            color: STYLE_CONSTANTS.textColor,
          },
          emphasis: {
            label: {
              show: true,
              fontSize: 16,
              fontWeight: 'bold' as const,
            },
          },
          data: chartData,
        },
      ],
    };

    this.setOption(option);
  }

  renderDuration(data: GeographicResponse): void {
    const stats = data.duration_stats;
    if (!stats) return;

    const durationByType = stats.duration_by_media_type || [];
    const mediaTypes = durationByType.map(d => d.media_type);
    const avgDurations = durationByType.map(d => d.avg_duration);
    const totalDurations = durationByType.map(d => d.total_duration);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const idx = params[0].dataIndex;
          const type = durationByType[idx];
          const totalHours = Math.round(type.total_duration / 60);
          return `<div style="font-weight: 600; margin-bottom: 8px;">${type.media_type}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Avg Duration:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(type.avg_duration)} min</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Total Duration:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(totalHours)}h</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Playbacks:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(type.playback_count)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Avg Completion:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatPercent(type.avg_completion)}</span>
            </div>`;
        },
      }),
      legend: this.getLegend({
        data: ['Average Duration (min)', 'Total Watch Time (hours)'],
      }),
      grid: {
        left: '10%',
        right: '10%',
        bottom: '10%',
        top: '15%',
      },
      xAxis: this.getCategoryAxis(mediaTypes),
      yAxis: [
        {
          ...this.getValueAxis(),
          name: 'Avg Duration (min)',
          position: 'left' as const,
        },
        {
          ...this.getValueAxis(),
          name: 'Total (hours)',
          position: 'right' as const,
          axisLabel: {
            color: STYLE_CONSTANTS.axisLabelColor,
            formatter: (value: number) => Math.round(value).toString(),
          },
        },
      ],
      series: [
        {
          name: 'Average Duration (min)',
          type: 'bar' as const,
          data: avgDurations,
          itemStyle: {
            color: CHART_COLORS.success,
            borderRadius: this.helpers.getBarBorderRadius('vertical'),
          },
        },
        {
          name: 'Total Watch Time (hours)',
          type: 'bar' as const,
          yAxisIndex: 1,
          data: totalDurations.map(d => d / 60),
          itemStyle: {
            color: CHART_COLORS.info,
            borderRadius: this.helpers.getBarBorderRadius('vertical'),
          },
        },
      ],
    };

    this.setOption(option);
  }

  renderYears(data: GeographicResponse): void {
    if (!this.hasData(data?.year_distribution)) {
      this.showEmptyState('No year data available');
      return;
    }
    const years = data.year_distribution;
    const yearLabels = years.map(y => y.year.toString());
    const playbackCounts = years.map(y => y.playback_count);
    const total = playbackCounts.reduce((sum, count) => sum + count, 0);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const p = params[0];
          const value = this.formatters.formatNumber(p.value as number);
          const percentage = this.formatters.formatPercent(
            this.helpers.calculatePercentage(p.value as number, total)
          );
          return `<div style="font-weight: 600; margin-bottom: 8px;">Year: ${p.name}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Playbacks:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${value}</span>
              <span style="color: #a0a0a0; margin-left: 8px;">(${percentage})</span>
            </div>`;
        },
      }),
      grid: {
        left: '10%',
        right: '10%',
        bottom: '10%',
        top: '10%',
      },
      xAxis: this.getCategoryAxis(yearLabels, { rotate: 45 }),
      yAxis: this.getValueAxis(),
      series: [
        {
          type: 'bar' as const,
          data: playbackCounts,
          itemStyle: {
            color: (params: EChartsItemStyleColorParams) => {
              const colors = [...COLOR_PALETTES.default, CHART_COLORS.darker, CHART_COLORS.crimson, CHART_COLORS.dark];
              return colors[params.dataIndex % colors.length];
            },
            borderRadius: this.helpers.getBarBorderRadius('vertical'),
          },
          label: {
            show: true,
            position: 'top' as const,
            color: STYLE_CONSTANTS.textColor,
          },
        },
      ],
    };

    this.setOption(option);
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private createBarTooltipFormatter(total: number): any {
    return (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
      if (!Array.isArray(params) || params.length === 0) return '';
      const param = params[0];
      const value = this.formatters.formatNumber(param.value as number);
      const percentage = this.formatters.formatPercent(
        this.helpers.calculatePercentage(param.value as number, total)
      );
      return `<div style="font-weight: 600; margin-bottom: 8px;">${param.name}</div>
        <div style="margin: 4px 0;">
          <span style="font-weight: 500;">Playbacks:</span>
          <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${value}</span>
          <span style="color: #a0a0a0; margin-left: 8px;">(${percentage})</span>
        </div>`;
    };
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private createPieTooltipFormatter(): any {
    return (params: EChartsCallbackDataParams) => {
      const value = this.formatters.formatNumber(params.value as number);
      const percentage = this.formatters.formatPercent(params.percent ?? 0);
      return `<div style="font-weight: 600; margin-bottom: 8px;">${params.name}</div>
        <div style="margin: 4px 0;">
          <span style="font-weight: 500;">Playbacks:</span>
          <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${value}</span>
        </div>
        <div style="margin: 4px 0;">
          <span style="font-weight: 500;">Percentage:</span>
          <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${percentage}</span>
        </div>`;
    };
  }
}
