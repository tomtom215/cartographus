// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Renderer for popular content charts
 */

import type { PopularAnalyticsResponse } from '../../api';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS, COLOR_PALETTES, STYLE_CONSTANTS } from '../config/colors';
import * as echarts from 'echarts';
import type { EChartsCallbackDataParams } from '../types';

export class PopularChartRenderer extends BaseChartRenderer {
  render(_data: PopularAnalyticsResponse): void {
    // Base render method - unused in this renderer
  }

  renderMovies(data: PopularAnalyticsResponse): void {
    if (!this.hasData(data?.top_movies)) {
      this.showEmptyState('No movie data available');
      return;
    }
    const movies = data.top_movies.map(m => m.title);
    const counts = data.top_movies.map(m => m.playback_count);

    const [reversedMovies, reversedCounts] = this.helpers.reverseArrays(movies, counts);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const movie = data.top_movies[params[0].dataIndex];
          const watchTimeHours = Math.round(movie.total_watch_time_minutes / 60);
          return `<div style="font-weight: 600; margin-bottom: 8px;">${movie.title}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Playbacks:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(movie.playback_count)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Unique Users:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(movie.unique_users)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Watch Time:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${watchTimeHours}h</span>
            </div>`;
        }
      }),
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: this.getValueAxis(),
      yAxis: this.getCategoryAxis(reversedMovies),
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

  renderShows(data: PopularAnalyticsResponse): void {
    if (!this.hasData(data?.top_shows)) {
      this.showEmptyState('No TV show data available');
      return;
    }
    const shows = data.top_shows.map(s => s.title);
    const counts = data.top_shows.map(s => s.playback_count);

    const [reversedShows, reversedCounts] = this.helpers.reverseArrays(shows, counts);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const show = data.top_shows[params[0].dataIndex];
          const watchTimeHours = Math.round(show.total_watch_time_minutes / 60);
          return `<div style="font-weight: 600; margin-bottom: 8px;">${show.title}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Playbacks:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(show.playback_count)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Unique Users:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(show.unique_users)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Watch Time:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${watchTimeHours}h</span>
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
          color: new echarts.graphic.LinearGradient(1, 0, 0, 0, [
            { offset: 0, color: CHART_COLORS.orange },
            { offset: 1, color: CHART_COLORS.secondary }
          ]),
          borderRadius: this.helpers.getBarBorderRadius('horizontal')
        },
        label: { show: true, position: 'right' as const, color: STYLE_CONSTANTS.textColor }
      }]
    };

    this.setOption(option);
  }

  renderEpisodes(data: PopularAnalyticsResponse): void {
    if (!this.hasData(data?.top_episodes)) {
      this.showEmptyState('No episode data available');
      return;
    }
    const episodes = data.top_episodes.map(e => `${e.grandparent_title} - ${e.title}`);
    const counts = data.top_episodes.map(e => e.playback_count);

    const [reversedEpisodes, reversedCounts] = this.helpers.reverseArrays(episodes, counts);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const episode = data.top_episodes[params[0].dataIndex];
          return `<div style="font-weight: 600; margin-bottom: 8px;">${episode.grandparent_title}</div>
            <div style="margin: 4px 0; color: #a0a0a0;">${episode.title}</div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Playbacks:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(episode.playback_count)}</span>
            </div>
            <div style="margin: 4px 0;">
              <span style="font-weight: 500;">Unique Users:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${this.formatters.formatNumber(episode.unique_users)}</span>
            </div>`;
        }
      }),
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: this.getValueAxis(),
      yAxis: this.getCategoryAxis(reversedEpisodes, { fontSize: 10 }),
      series: [{
        type: 'bar' as const,
        data: reversedCounts,
        itemStyle: {
          color: new echarts.graphic.LinearGradient(1, 0, 0, 0, [
            { offset: 0, color: CHART_COLORS.warning },
            { offset: 1, color: CHART_COLORS.accent }
          ]),
          borderRadius: this.helpers.getBarBorderRadius('horizontal')
        },
        label: { show: true, position: 'right' as const, color: STYLE_CONSTANTS.textColor }
      }]
    };

    this.setOption(option);
  }
}
