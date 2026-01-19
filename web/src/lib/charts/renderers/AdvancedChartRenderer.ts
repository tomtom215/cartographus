// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Renderer for advanced analytics charts
 * Handles temporal heatmap, resolution mismatch, HDR, audio, subtitle,
 * connection security, pause patterns, and concurrent streams
 */

import type {
  TemporalHeatmapResponse,
  ResolutionMismatchAnalytics,
  HDRAnalytics,
  AudioAnalytics,
  SubtitleAnalytics,
  ConnectionSecurityAnalytics,
  PausePatternAnalytics,
  ConcurrentStreamsAnalytics,
  HardwareTranscodeStats,
  AbandonmentStats,
  FrameRateAnalytics,
  ContainerAnalytics,
  HWTranscodeTrend
} from '../../api';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS, STYLE_CONSTANTS } from '../config/colors';
import type { EChartsCallbackDataParams } from '../types';

export class AdvancedChartRenderer extends BaseChartRenderer {
  /**
   * FIX: Store temporal data as instance property instead of polluting window global.
   * Used for animation control in temporal heatmap.
   */
  private temporalData: TemporalHeatmapResponse | null = null;

  render(_data: any): void {
    // Base render method - unused in this renderer
  }

  /**
   * Get stored temporal heatmap data for animation control.
   * Returns null if no data has been loaded.
   */
  getTemporalData(): TemporalHeatmapResponse | null {
    return this.temporalData;
  }

  renderTemporalHeatmap(data: TemporalHeatmapResponse): void {
    if (!this.hasData(data?.buckets)) {
      this.showEmptyState('No temporal heatmap data available');
      return;
    }
    // For the temporal heatmap, we need a more complex implementation with animation
    // This is a placeholder that shows summary information
    // The actual map animation should be implemented separately
    const container = document.getElementById('temporal-label');
    if (container && data.buckets.length > 0) {
      container.textContent = `${data.buckets.length} time periods (${data.interval})`;
    }

    // Store temporal data for animation control (uses instance property, not window global)
    this.temporalData = data;
  }

  renderResolutionMismatch(data: ResolutionMismatchAnalytics): void {
    if (!this.hasData(data?.mismatches)) {
      this.showEmptyState('No resolution mismatch data available');
      return;
    }
    const nodes: Array<{name: string}> = [];
    const links: Array<{source: string; target: string; value: number}> = [];
    const nodeNames = new Set<string>();

    data.mismatches.forEach(mismatch => {
      nodeNames.add(mismatch.source_resolution);
      nodeNames.add(mismatch.stream_resolution);
    });

    nodeNames.forEach(name => nodes.push({name}));
    data.mismatches.forEach(mismatch => {
      links.push({
        source: mismatch.source_resolution,
        target: mismatch.stream_resolution,
        value: mismatch.playback_count
      });
    });

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Resolution Quality Loss Analysis',
        subtext: `${this.formatters.formatPercent(data.mismatch_rate)} downgraded`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 },
        subtextStyle: { color: STYLE_CONSTANTS.axisLabelColor, fontSize: 12 }
      },
      tooltip: this.getTooltip({ trigger: 'item' }),
      series: [{
        type: 'sankey' as const,
        data: nodes,
        links: links,
        emphasis: { focus: 'adjacency' as const },
        lineStyle: { color: 'gradient' as const, curveness: 0.5 }
      }]
    };

    this.setOption(option, { notMerge: true });
  }

  renderHDRAnalytics(data: HDRAnalytics): void {
    if (!this.hasData(data?.format_distribution)) {
      this.showEmptyState('No HDR analytics data available');
      return;
    }
    const formatData = data.format_distribution.map(f => ({
      name: f.dynamic_range,
      value: f.playback_count
    }));

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'HDR & Dynamic Range Distribution',
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 }
      },
      tooltip: this.getTooltip({ trigger: 'item' }),
      legend: this.getLegend(),
      series: [{
        type: 'pie' as const,
        radius: ['40%', '70%'],
        data: formatData
      }]
    };

    this.setOption(option, { notMerge: true });
  }

  renderAudioAnalytics(data: AudioAnalytics): void {
    if (!this.hasData(data?.channel_distribution)) {
      this.showEmptyState('No audio analytics data available');
      return;
    }
    const channelData = data.channel_distribution.map(c => ({
      name: c.layout || c.channels,
      value: c.playback_count
    }));

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Audio Quality Distribution',
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 }
      },
      tooltip: this.getTooltip({ trigger: 'axis' }),
      xAxis: this.getCategoryAxis(channelData.map(c => c.name), { rotate: 45 }),
      yAxis: this.getValueAxis(),
      series: [{
        type: 'bar' as const,
        data: channelData.map(c => c.value)
      }]
    };

    this.setOption(option, { notMerge: true });
  }

  renderSubtitleAnalytics(data: SubtitleAnalytics): void {
    if (!this.hasData(data?.language_distribution)) {
      this.showEmptyState('No subtitle analytics data available');
      return;
    }
    const languageData = data.language_distribution.map(lang => ({
      name: lang.language,
      value: lang.playback_count
    }));

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Subtitle Usage by Language',
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 }
      },
      tooltip: this.getTooltip({ trigger: 'item' }),
      series: [{
        type: 'pie' as const,
        radius: '60%',
        data: languageData
      }]
    };

    this.setOption(option, { notMerge: true });
  }

  renderConnectionSecurity(data: ConnectionSecurityAnalytics): void {
    if (!data) {
      this.showEmptyState('No connection security data available');
      return;
    }
    const securityData = [
      { name: 'Secure', value: data.secure_connections },
      { name: 'Insecure', value: data.insecure_connections },
      { name: 'Relayed', value: data.relayed_connections.count }
    ];

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Connection Security Overview',
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 }
      },
      tooltip: this.getTooltip({ trigger: 'item' }),
      series: [{
        type: 'pie' as const,
        radius: ['40%', '70%'],
        data: securityData.map((item, idx) => ({
          ...item,
          itemStyle: {
            color: idx === 0 ? '#4caf50' : idx === 1 ? '#f44336' : '#ff9800'
          }
        }))
      }]
    };

    this.setOption(option, { notMerge: true });
  }

  renderPausePatterns(data: PausePatternAnalytics): void {
    if (!this.hasData(data?.pause_distribution)) {
      this.showEmptyState('No pause pattern data available');
      return;
    }
    const distributionData = data.pause_distribution.map(d => ({
      name: d.pause_bucket,
      value: d.playback_count
    }));

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Pause Pattern Distribution',
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 }
      },
      tooltip: this.getTooltip({ trigger: 'axis' }),
      xAxis: this.getCategoryAxis(distributionData.map(d => d.name)),
      yAxis: this.getValueAxis(),
      series: [{
        type: 'bar' as const,
        data: distributionData.map(d => d.value)
      }]
    };

    this.setOption(option, { notMerge: true });
  }

  renderConcurrentStreams(data: ConcurrentStreamsAnalytics): void {
    if (!this.hasData(data?.time_series_data)) {
      this.showEmptyState('No concurrent streams data available');
      return;
    }
    const timestamps = data.time_series_data.map(d => d.timestamp);
    const directPlayData = data.time_series_data.map(d => d.direct_play);
    const directStreamData = data.time_series_data.map(d => d.direct_stream);
    const transcodeData = data.time_series_data.map(d => d.transcode);
    const totalConcurrent = data.time_series_data.map(d => d.concurrent_count);

    const peakTimeFormatted = new Date(data.peak_time).toLocaleString();

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Concurrent Streams Over Time',
        subtext: `Peak: ${data.peak_concurrent} streams at ${peakTimeFormatted} | Avg: ${data.avg_concurrent.toFixed(1)} | ${data.capacity_recommendation}`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 },
        subtextStyle: { color: STYLE_CONSTANTS.axisLabelColor, fontSize: 11 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'cross' },
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const time = params[0].axisValue;
          let html = `<div style="font-weight: 600; margin-bottom: 8px;">${time}</div>`;
          let total = 0;
          params.forEach((p: EChartsCallbackDataParams) => {
            const value = (p.value as number) || 0;
            total += value;
            html += `<div style="margin: 4px 0;">
              <span style="display: inline-block; width: 10px; height: 10px; border-radius: 50%; background: ${p.color}; margin-right: 8px;"></span>
              <span style="font-weight: 500;">${p.seriesName}:</span>
              <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${value}</span>
            </div>`;
          });
          html += `<div style="margin-top: 8px; padding-top: 8px; border-top: 1px solid #3a3a4e;">
            <strong>Total Concurrent: ${total}</strong>
          </div>`;
          return html;
        }
      }),
      legend: this.getLegend({
        data: ['Direct Play', 'Direct Stream', 'Transcode', 'Total'],
        bottom: 20
      }),
      grid: {
        left: 80,
        right: 40,
        top: 120,
        bottom: 80,
        containLabel: true
      },
      xAxis: {
        type: 'category' as const,
        data: timestamps,
        axisLabel: {
          color: STYLE_CONSTANTS.axisLabelColor,
          rotate: 45,
          formatter: (value: string) => {
            const date = new Date(value);
            return `${date.getMonth() + 1}/${date.getDate()} ${date.getHours()}:00`;
          }
        },
        axisLine: { lineStyle: { color: '#3a3a4e' } }
      },
      yAxis: {
        type: 'value' as const,
        name: 'Concurrent Streams',
        min: 0,
        axisLabel: {
          color: STYLE_CONSTANTS.axisLabelColor,
          formatter: (value: number) => this.formatters.formatNumber(value)
        },
        axisLine: { lineStyle: { color: '#3a3a4e' } },
        splitLine: { lineStyle: { color: '#2a2a3e', type: 'dashed' as const } },
        nameTextStyle: { color: STYLE_CONSTANTS.axisLabelColor }
      },
      series: [
        {
          name: 'Direct Play',
          type: 'line' as const,
          stack: 'total',
          areaStyle: { opacity: 0.7 },
          emphasis: { focus: 'series' as const },
          data: directPlayData,
          smooth: true,
          itemStyle: { color: '#4caf50' }
        },
        {
          name: 'Direct Stream',
          type: 'line' as const,
          stack: 'total',
          areaStyle: { opacity: 0.7 },
          emphasis: { focus: 'series' as const },
          data: directStreamData,
          smooth: true,
          itemStyle: { color: '#2196f3' }
        },
        {
          name: 'Transcode',
          type: 'line' as const,
          stack: 'total',
          areaStyle: { opacity: 0.7 },
          emphasis: { focus: 'series' as const },
          data: transcodeData,
          smooth: true,
          itemStyle: { color: '#ff9800' }
        },
        {
          name: 'Total',
          type: 'line' as const,
          data: totalConcurrent,
          lineStyle: { type: 'dashed' as const, color: CHART_COLORS.primary, width: 2 },
          symbol: 'none',
          z: 10,
          markLine: {
            silent: true,
            lineStyle: { color: '#ff0000', type: 'solid' as const, width: 1 },
            label: {
              formatter: `Peak: ${data.peak_concurrent}`,
              color: '#ff0000',
              position: 'end' as const
            },
            data: [{ yAxis: data.peak_concurrent }]
          }
        }
      ]
    };

    this.setOption(option, { notMerge: true });
  }

  /**
   * Render hardware transcode analytics chart
   * Shows GPU utilization breakdown (NVIDIA NVENC, Intel Quick Sync, AMD VCE)
   */
  renderHardwareTranscode(data: HardwareTranscodeStats): void {
    if (!data || data.total_sessions === 0) {
      this.showEmptyState('No hardware transcode data available');
      return;
    }

    // Create pie data for HW vs SW vs Direct Play breakdown
    const overviewData = [
      {
        name: 'Hardware Transcode',
        value: data.hw_transcode_sessions,
        itemStyle: { color: '#4caf50' } // Green for HW
      },
      {
        name: 'Software Transcode',
        value: data.sw_transcode_sessions,
        itemStyle: { color: '#ff9800' } // Orange for SW
      },
      {
        name: 'Direct Play',
        value: data.direct_play_sessions,
        itemStyle: { color: '#2196f3' } // Blue for Direct
      }
    ].filter(item => item.value > 0);

    // Create decoder breakdown data for secondary pie
    const decoderData = data.decoder_stats?.map(d => ({
      name: d.codec.toUpperCase(),
      value: d.session_count
    })) || [];

    const hwPercentFormatted = data.hw_percentage.toFixed(1);

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Hardware Transcode Analytics',
        subtext: `${hwPercentFormatted}% GPU Accelerated | Total: ${this.formatters.formatNumber(data.total_sessions)} sessions`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 },
        subtextStyle: { color: STYLE_CONSTANTS.axisLabelColor, fontSize: 12 }
      },
      tooltip: this.getTooltip({
        trigger: 'item',
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        formatter: ((params: EChartsCallbackDataParams) => {
          const value = params.value as number;
          const percent = ((value / data.total_sessions) * 100).toFixed(1);
          return `<strong>${params.seriesName}</strong><br/>
                  ${params.name}: ${this.formatters.formatNumber(value)} sessions (${percent}%)`;
        }) as any
      }),
      legend: this.getLegend({
        data: overviewData.map(d => d.name).concat(decoderData.map(d => d.name)),
        bottom: 10,
        type: 'scroll' as const
      }),
      grid: {
        left: 60,
        right: 60,
        top: 100,
        bottom: 80
      },
      series: [
        {
          name: 'Session Type',
          type: 'pie' as const,
          radius: ['30%', '50%'],
          center: ['30%', '50%'],
          data: overviewData,
          label: {
            show: true,
            formatter: '{b}: {d}%',
            color: STYLE_CONSTANTS.textColor
          },
          emphasis: {
            itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0, 0, 0, 0.5)' }
          }
        },
        {
          name: 'GPU Decoder',
          type: 'pie' as const,
          radius: ['25%', '40%'],
          center: ['70%', '50%'],
          data: decoderData,
          label: {
            show: true,
            formatter: '{b}',
            color: STYLE_CONSTANTS.textColor,
            fontSize: 10
          },
          emphasis: {
            itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0, 0, 0, 0.5)' }
          }
        }
      ]
    };

    this.setOption(option, { notMerge: true });
  }

  /**
   * Render abandonment analytics chart
   * Shows content abandonment patterns and drop-off analysis
   */
  renderAbandonment(data: AbandonmentStats): void {
    if (!data || data.total_sessions === 0) {
      this.showEmptyState('No abandonment data available');
      return;
    }

    // Null safety: ensure numeric properties exist before calling toFixed
    const abandonRateFormatted = (data.abandonment_rate ?? 0).toFixed(1);
    const avgAbandonPointFormatted = (data.avg_abandonment_point ?? 0).toFixed(0);

    // Hourly abandonment data
    const hourLabels = data.by_hour?.map(h => `${h.hour}:00`) || [];
    const hourCounts = data.by_hour?.map(h => h.count) || [];
    const hourRates = data.by_hour?.map(h => h.rate) || [];

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Content Abandonment Analysis',
        subtext: `${abandonRateFormatted}% abandonment rate | Avg drop-off at ${avgAbandonPointFormatted}%`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 },
        subtextStyle: { color: STYLE_CONSTANTS.axisLabelColor, fontSize: 12 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'cross' }
      }),
      legend: this.getLegend({
        data: ['Abandonments', 'Rate (%)'],
        bottom: 10
      }),
      grid: {
        left: 80,
        right: 80,
        top: 100,
        bottom: 80,
        containLabel: true
      },
      xAxis: {
        type: 'category' as const,
        data: hourLabels,
        name: 'Hour of Day',
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor, rotate: 45 },
        axisLine: { lineStyle: { color: '#3a3a4e' } },
        nameTextStyle: { color: STYLE_CONSTANTS.axisLabelColor }
      },
      yAxis: [
        {
          type: 'value' as const,
          name: 'Abandonments',
          axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
          axisLine: { lineStyle: { color: '#3a3a4e' } },
          splitLine: { lineStyle: { color: '#2a2a3e', type: 'dashed' as const } },
          nameTextStyle: { color: STYLE_CONSTANTS.axisLabelColor }
        },
        {
          type: 'value' as const,
          name: 'Rate (%)',
          max: 100,
          axisLabel: {
            color: STYLE_CONSTANTS.axisLabelColor,
            formatter: '{value}%'
          },
          axisLine: { lineStyle: { color: '#3a3a4e' } },
          nameTextStyle: { color: STYLE_CONSTANTS.axisLabelColor }
        }
      ],
      series: [
        {
          name: 'Abandonments',
          type: 'bar' as const,
          data: hourCounts,
          itemStyle: { color: '#ff5252' }
        },
        {
          name: 'Rate (%)',
          type: 'line' as const,
          yAxisIndex: 1,
          data: hourRates,
          smooth: true,
          lineStyle: { color: CHART_COLORS.primary, width: 2 },
          itemStyle: { color: CHART_COLORS.primary }
        }
      ]
    };

    this.setOption(option, { notMerge: true });
  }

  /**
   * Render frame rate analytics chart
   * Shows distribution of playback frame rates (24/30/60fps etc.)
   */
  renderFrameRate(data: FrameRateAnalytics): void {
    if (!this.hasData(data?.frame_rate_distribution)) {
      this.showEmptyState('No frame rate data available');
      return;
    }

    const highFpsFormatted = data.high_frame_rate_adoption_percent.toFixed(1);

    // Prepare data for pie chart
    const pieData = data.frame_rate_distribution.map(d => ({
      name: d.frame_rate,
      value: d.playback_count,
      itemStyle: {
        color: d.frame_rate.includes('60') ? CHART_COLORS.primary :
               d.frame_rate.includes('24') ? CHART_COLORS.secondary :
               CHART_COLORS.accent
      }
    }));

    // Prepare bar chart data for completion rates
    const frameRates = data.frame_rate_distribution.map(d => d.frame_rate);
    const completionRates = data.frame_rate_distribution.map(d => d.avg_completion);

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Frame Rate Distribution',
        subtext: `${highFpsFormatted}% High Frame Rate (60fps+) | Total: ${this.formatters.formatNumber(data.total_playbacks)} playbacks`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 },
        subtextStyle: { color: STYLE_CONSTANTS.axisLabelColor, fontSize: 12 }
      },
      tooltip: this.getTooltip({
        trigger: 'item',
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        formatter: ((params: EChartsCallbackDataParams) => {
          const value = params.value as number;
          if (params.seriesType === 'pie') {
            const item = data.frame_rate_distribution.find(d => d.frame_rate === params.name);
            return `<strong>${params.name}</strong><br/>
                    Playbacks: ${this.formatters.formatNumber(value)} (${params.percent}%)<br/>
                    Avg Completion: ${item?.avg_completion.toFixed(1)}%`;
          }
          return `${params.name}: ${value.toFixed(1)}% completion`;
        }) as any
      }),
      legend: this.getLegend({
        data: frameRates,
        bottom: 10,
        type: 'scroll' as const
      }),
      grid: {
        left: '55%',
        right: 40,
        top: 100,
        bottom: 80
      },
      xAxis: {
        type: 'category' as const,
        data: frameRates,
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor, rotate: 45 },
        axisLine: { lineStyle: { color: '#3a3a4e' } }
      },
      yAxis: {
        type: 'value' as const,
        name: 'Completion %',
        max: 100,
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor, formatter: '{value}%' },
        axisLine: { lineStyle: { color: '#3a3a4e' } },
        splitLine: { lineStyle: { color: '#2a2a3e', type: 'dashed' as const } }
      },
      series: [
        {
          name: 'Distribution',
          type: 'pie' as const,
          radius: ['30%', '55%'],
          center: ['25%', '55%'],
          data: pieData,
          label: {
            show: true,
            formatter: '{b}: {d}%',
            color: STYLE_CONSTANTS.textColor,
            fontSize: 11
          },
          emphasis: {
            itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0, 0, 0, 0.5)' }
          }
        },
        {
          name: 'Completion Rate',
          type: 'bar' as const,
          data: completionRates,
          itemStyle: { color: CHART_COLORS.secondary },
          barWidth: '60%'
        }
      ]
    };

    this.setOption(option, { notMerge: true });
  }

  /**
   * Render container format analytics chart
   * Shows distribution of container formats (MKV, MP4, AVI etc.)
   */
  renderContainer(data: ContainerAnalytics): void {
    if (!this.hasData(data?.format_distribution)) {
      this.showEmptyState('No container format data available');
      return;
    }

    // Prepare data for pie chart
    const pieData = data.format_distribution.map((d, i) => ({
      name: d.container,
      value: d.playback_count,
      itemStyle: { color: [CHART_COLORS.primary, CHART_COLORS.secondary, CHART_COLORS.accent, '#ff7c43', '#00bfa5', '#7c3aed'][i % 6] }
    }));

    // Prepare data for direct play rate bar chart
    const containers = data.format_distribution.map(d => d.container);
    const directPlayRates = data.format_distribution.map(d => d.direct_play_rate_percent);

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Container Format Distribution',
        subtext: `Total: ${this.formatters.formatNumber(data.total_playbacks)} playbacks`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 },
        subtextStyle: { color: STYLE_CONSTANTS.axisLabelColor, fontSize: 12 }
      },
      tooltip: this.getTooltip({
        trigger: 'item',
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        formatter: ((params: EChartsCallbackDataParams) => {
          const value = params.value as number;
          if (params.seriesType === 'pie') {
            const item = data.format_distribution.find(d => d.container === params.name);
            return `<strong>${params.name}</strong><br/>
                    Playbacks: ${this.formatters.formatNumber(value)} (${params.percent}%)<br/>
                    Direct Play: ${item?.direct_play_rate_percent.toFixed(1)}%`;
          }
          return `${params.name}: ${value.toFixed(1)}% direct play`;
        }) as any
      }),
      legend: this.getLegend({
        data: containers,
        bottom: 10,
        type: 'scroll' as const
      }),
      grid: {
        left: '55%',
        right: 40,
        top: 100,
        bottom: 80
      },
      xAxis: {
        type: 'category' as const,
        data: containers,
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor, rotate: 45 },
        axisLine: { lineStyle: { color: '#3a3a4e' } }
      },
      yAxis: {
        type: 'value' as const,
        name: 'Direct Play %',
        max: 100,
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor, formatter: '{value}%' },
        axisLine: { lineStyle: { color: '#3a3a4e' } },
        splitLine: { lineStyle: { color: '#2a2a3e', type: 'dashed' as const } }
      },
      series: [
        {
          name: 'Format Distribution',
          type: 'pie' as const,
          radius: ['30%', '55%'],
          center: ['25%', '55%'],
          data: pieData,
          label: {
            show: true,
            formatter: '{b}: {d}%',
            color: STYLE_CONSTANTS.textColor,
            fontSize: 11
          },
          emphasis: {
            itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0, 0, 0, 0.5)' }
          }
        },
        {
          name: 'Direct Play Rate',
          type: 'bar' as const,
          data: directPlayRates,
          itemStyle: { color: CHART_COLORS.primary },
          barWidth: '60%'
        }
      ]
    };

    this.setOption(option, { notMerge: true });
  }

  /**
   * Render hardware transcode trends chart
   * Shows historical trends of HW vs SW transcoding over time
   */
  renderHardwareTranscodeTrends(data: HWTranscodeTrend[]): void {
    if (!this.hasData(data)) {
      this.showEmptyState('No hardware transcode trend data available');
      return;
    }

    const dates = data.map(d => d.day);
    const hwSessions = data.map(d => d.hw_sessions);
    const swSessions = data.map(d => d.total_sessions - d.hw_sessions);
    const hwPercentages = data.map(d => d.hw_percentage);
    const fullPipelineSessions = data.map(d => d.full_pipeline_sessions);

    // Calculate average HW percentage
    const avgHwPercentage = data.length > 0
      ? (data.reduce((sum, d) => sum + d.hw_percentage, 0) / data.length).toFixed(1)
      : '0';

    // Enable dataZoom for datasets with more than 14 days
    const enableDataZoom = dates.length > 14;

    const option = {
      ...this.getBaseOption(),
      title: {
        text: 'Hardware Transcode Trends',
        subtext: `Average HW Acceleration: ${avgHwPercentage}%`,
        textStyle: { color: STYLE_CONSTANTS.textColor, fontSize: 18, fontWeight: 600 },
        subtextStyle: { color: STYLE_CONSTANTS.axisLabelColor, fontSize: 12 }
      },
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'cross' },
        formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
          if (!Array.isArray(params) || params.length === 0) return '';
          const date = params[0].axisValue;
          let html = `<strong>${date}</strong><br/>`;
          params.forEach((p: EChartsCallbackDataParams) => {
            const numValue = p.value as number;
            const value = p.seriesName === 'HW %' ? `${numValue.toFixed(1)}%` : this.formatters.formatNumber(numValue);
            html += `<span style="color:${p.color}">\u25CF</span> ${p.seriesName}: ${value}<br/>`;
          });
          return html;
        }
      }),
      legend: this.getLegend({
        data: ['Hardware', 'Software', 'Full Pipeline', 'HW %'],
        bottom: enableDataZoom ? 50 : 10
      }),
      grid: {
        left: 60,
        right: 60,
        top: 100,
        bottom: enableDataZoom ? 100 : 60
      },
      dataZoom: enableDataZoom ? this.getDataZoom('both') : undefined,
      xAxis: {
        type: 'category' as const,
        data: dates,
        boundaryGap: false,
        axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
        axisLine: { lineStyle: { color: '#3a3a4e' } }
      },
      yAxis: [
        {
          type: 'value' as const,
          name: 'Sessions',
          axisLabel: { color: STYLE_CONSTANTS.axisLabelColor },
          axisLine: { lineStyle: { color: '#3a3a4e' } },
          splitLine: { lineStyle: { color: '#2a2a3e', type: 'dashed' as const } }
        },
        {
          type: 'value' as const,
          name: 'HW %',
          max: 100,
          axisLabel: { color: STYLE_CONSTANTS.axisLabelColor, formatter: '{value}%' },
          axisLine: { lineStyle: { color: '#3a3a4e' } }
        }
      ],
      series: [
        {
          name: 'Hardware',
          type: 'bar' as const,
          stack: 'sessions',
          data: hwSessions,
          itemStyle: { color: CHART_COLORS.primary }
        },
        {
          name: 'Software',
          type: 'bar' as const,
          stack: 'sessions',
          data: swSessions,
          itemStyle: { color: '#ff5252' }
        },
        {
          name: 'Full Pipeline',
          type: 'line' as const,
          data: fullPipelineSessions,
          smooth: true,
          lineStyle: { color: CHART_COLORS.accent, width: 2 },
          itemStyle: { color: CHART_COLORS.accent },
          showSymbol: false
        },
        {
          name: 'HW %',
          type: 'line' as const,
          yAxisIndex: 1,
          data: hwPercentages,
          smooth: true,
          lineStyle: { color: CHART_COLORS.secondary, width: 2, type: 'dashed' as const },
          itemStyle: { color: CHART_COLORS.secondary },
          showSymbol: false
        }
      ]
    };

    this.setOption(option, { notMerge: true });
  }
}
