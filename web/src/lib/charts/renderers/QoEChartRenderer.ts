// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Quality of Experience (QoE) Chart Renderers
 *
 * Renders QoE dashboard metrics using:
 * - Gauge charts for overall QoE score
 * - Bar charts for platform/transcode comparisons
 * - Line charts for trends
 * - Issue severity indicators
 *
 * Based on Netflix QoE metrics and Conviva SPI methodology.
 */

import type { QoEDashboard } from '../../types/enhanced-analytics';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS, getChartPalette } from '../config/colors';
import { getGridConfig } from '../config/grid';
import type { EChartsCallbackDataParams, EChartsLabelFormatterParams } from '../types';

/**
 * QoE Score Gauge Renderer
 * Shows overall QoE score with grade indicator
 */
export class QoEScoreGaugeRenderer extends BaseChartRenderer {
	render(data: QoEDashboard): void {
		if (!data?.summary) {
			this.showEmptyState('No QoE data available');
			return;
		}

		const summary = data.summary;
		const gradeColor = this.getGradeColor(summary.qoe_grade);

		const option = {
			...this.getBaseOption(),
			series: [
				{
					type: 'gauge' as const,
					startAngle: 180,
					endAngle: 0,
					center: ['50%', '75%'],
					radius: '100%',
					min: 0,
					max: 100,
					splitNumber: 10,
					axisLine: {
						lineStyle: {
							width: 20,
							color: [
								[0.4, CHART_COLORS.danger] as [number, string],
								[0.6, CHART_COLORS.warning] as [number, string],
								[0.8, CHART_COLORS.info] as [number, string],
								[1, CHART_COLORS.success] as [number, string],
							],
						},
					},
					pointer: {
						icon: 'path://M2.9,0.7L2.9,0.7c1.4,0,2.6,1.2,2.6,2.6v115c0,1.4-1.2,2.6-2.6,2.6l0,0c-1.4,0-2.6-1.2-2.6-2.6V3.3C0.3,1.9,1.4,0.7,2.9,0.7z',
						length: '75%',
						width: 8,
						offsetCenter: [0, '-10%'],
						itemStyle: {
							color: gradeColor,
						},
					},
					axisTick: {
						length: 8,
						lineStyle: { color: 'auto', width: 1 },
					},
					splitLine: {
						length: 12,
						lineStyle: { color: 'auto', width: 2 },
					},
					axisLabel: {
						distance: 25,
						color: '#888',
						fontSize: 11,
					},
					title: {
						show: true,
						offsetCenter: [0, '-35%'],
						fontSize: 14,
						color: '#888',
					},
					detail: {
						fontSize: 36,
						offsetCenter: [0, '-10%'],
						valueAnimation: true,
						formatter: (value: number) => `${value.toFixed(0)}`,
						color: gradeColor,
					},
					data: [
						{
							value: summary.qoe_score,
							name: `Grade: ${summary.qoe_grade}`,
						},
					],
				},
			],
		};

		this.setOption(option);
	}

	private getGradeColor(grade: string): string {
		switch (grade) {
			case 'A': return CHART_COLORS.success;
			case 'B': return CHART_COLORS.info;
			case 'C': return CHART_COLORS.warning;
			case 'D': return CHART_COLORS.orange;
			case 'F': return CHART_COLORS.danger;
			default: return CHART_COLORS.primary;
		}
	}
}

/**
 * QoE Summary Metrics Renderer
 * Shows key metrics as horizontal bar chart
 */
export class QoESummaryMetricsRenderer extends BaseChartRenderer {
	render(data: QoEDashboard): void {
		if (!data?.summary) {
			this.showEmptyState('No QoE summary available');
			return;
		}

		const summary = data.summary;
		const metrics = [
			{ name: 'EBVS Rate', value: summary.ebvs_rate, target: 2, inverse: true },
			{ name: 'Quality Degrade', value: summary.quality_degrade_rate, target: 5, inverse: true },
			{ name: 'Direct Play Rate', value: summary.direct_play_rate, target: 60, inverse: false },
			{ name: 'High Completion', value: summary.high_completion_rate, target: 70, inverse: false },
			{ name: 'Secure Connection', value: summary.secure_connection_rate, target: 80, inverse: false },
		];

		const option = {
			...this.getBaseOption(),
			tooltip: this.getTooltip({
				trigger: 'axis',
				axisPointer: { type: 'shadow' },
				formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
					const p = Array.isArray(params) ? params[0] : params;
					const metric = metrics[p.dataIndex];
					const status = this.getMetricStatus(metric);
					return `
						<div style="font-weight: 600; margin-bottom: 8px;">${metric.name}</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Current:</span>
							<span style="color: ${status.color}; font-weight: 600;">${metric.value.toFixed(1)}%</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Target:</span>
							<span style="font-weight: 500;">${metric.inverse ? '<' : '>'} ${metric.target}%</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Status:</span>
							<span style="color: ${status.color}; font-weight: 500;">${status.label}</span>
						</div>
					`;
				},
			}),
			grid: {
				left: '25%',
				right: '10%',
				top: '10%',
				bottom: '10%',
			},
			xAxis: {
				type: 'value' as const,
				max: 100,
				...this.getAxisStyle(),
				axisLabel: {
					...this.getAxisStyle().axisLabel,
					formatter: '{value}%',
				},
			},
			yAxis: {
				type: 'category' as const,
				data: metrics.map(m => m.name),
				axisLabel: { color: '#a0a0a0' },
				axisTick: { show: false },
				axisLine: { show: false },
			},
			series: [
				{
					type: 'bar' as const,
					data: metrics.map(m => ({
						value: m.value,
						itemStyle: {
							color: this.getMetricStatus(m).color,
							borderRadius: [0, 4, 4, 0],
						},
					})),
					barWidth: '60%',
					label: {
						show: true,
						position: 'right' as const,
						formatter: (params: EChartsLabelFormatterParams) => {
							const value = typeof params.value === 'number' ? params.value : 0;
							return `${value.toFixed(1)}%`;
						},
						fontSize: 11,
						color: '#a0a0a0',
					},
				},
			],
		};

		this.setOption(option);
	}

	private getMetricStatus(metric: { value: number; target: number; inverse: boolean }): { color: string; label: string } {
		const good = metric.inverse ? metric.value < metric.target : metric.value > metric.target;
		const warning = metric.inverse
			? metric.value < metric.target * 2
			: metric.value > metric.target * 0.7;

		if (good) return { color: CHART_COLORS.success, label: 'Good' };
		if (warning) return { color: CHART_COLORS.warning, label: 'Warning' };
		return { color: CHART_COLORS.danger, label: 'Critical' };
	}
}

/**
 * QoE Trends Renderer
 * Shows QoE metrics over time
 */
export class QoETrendsRenderer extends BaseChartRenderer {
	render(data: QoEDashboard): void {
		if (!this.hasData(data?.trends)) {
			this.showEmptyState('No trend data available');
			return;
		}

		const trends = data.trends;
		const timestamps = trends.map(t =>
			new Date(t.timestamp).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
		);

		const option = {
			...this.getBaseOption(),
			tooltip: this.getTooltip({
				trigger: 'axis',
			}),
			legend: this.getLegend({
				data: ['QoE Score', 'EBVS Rate', 'Completion'],
				top: 0,
			}),
			grid: getGridConfig('withDataZoom'),
			xAxis: this.getCategoryAxis(timestamps, { boundaryGap: false }),
			yAxis: [
				this.getValueAxis({
					name: 'Score',
					max: 100,
					position: 'left',
				}),
				this.getValueAxis({
					name: 'Rate (%)',
					max: 20,
					position: 'right' as const,
					splitLine: { show: false },
				}),
			],
			dataZoom: this.getDataZoom('both'),
			series: [
				{
					name: 'QoE Score',
					type: 'line' as const,
					smooth: true,
					data: trends.map(t => t.qoe_score),
					lineStyle: { color: CHART_COLORS.primary, width: 3 },
					itemStyle: { color: CHART_COLORS.primary },
					areaStyle: {
						color: {
							type: 'linear' as const,
							x: 0, y: 0, x2: 0, y2: 1,
							colorStops: [
								{ offset: 0, color: 'rgba(124, 58, 237, 0.3)' },
								{ offset: 1, color: 'rgba(124, 58, 237, 0.05)' },
							],
						},
					},
				},
				{
					name: 'EBVS Rate',
					type: 'line' as const,
					yAxisIndex: 1,
					data: trends.map(t => t.ebvs_rate),
					lineStyle: { color: CHART_COLORS.danger, width: 2 },
					itemStyle: { color: CHART_COLORS.danger },
				},
				{
					name: 'Completion',
					type: 'line' as const,
					data: trends.map(t => t.avg_completion),
					lineStyle: { color: CHART_COLORS.success, width: 2 },
					itemStyle: { color: CHART_COLORS.success },
				},
			],
		};

		this.setOption(option);
	}
}

/**
 * QoE by Platform Renderer
 * Shows QoE comparison across platforms
 */
export class QoEByPlatformRenderer extends BaseChartRenderer {
	render(data: QoEDashboard): void {
		if (!this.hasData(data?.by_platform)) {
			this.showEmptyState('No platform data available');
			return;
		}

		const platforms = data.by_platform;
		const palette = getChartPalette();

		const option = {
			...this.getBaseOption(),
			tooltip: this.getTooltip({
				trigger: 'axis',
				axisPointer: { type: 'shadow' },
				formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
					const p = Array.isArray(params) ? params[0] : params;
					const platform = platforms[p.dataIndex];
					return `
						<div style="font-weight: 600; margin-bottom: 8px;">${platform.platform}</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Sessions:</span>
							<span style="font-weight: 500;">${platform.session_count.toLocaleString()} (${platform.session_percentage.toFixed(1)}%)</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">QoE Score:</span>
							<span style="color: ${this.getGradeColor(platform.qoe_grade)}; font-weight: 600;">${platform.qoe_score.toFixed(0)} (${platform.qoe_grade})</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Direct Play:</span>
							<span style="font-weight: 500;">${platform.direct_play_rate.toFixed(1)}%</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">EBVS:</span>
							<span style="font-weight: 500;">${platform.ebvs_rate.toFixed(1)}%</span>
						</div>
					`;
				},
			}),
			legend: this.getLegend({
				data: ['QoE Score', 'Direct Play %', 'EBVS %'],
				top: 0,
			}),
			grid: getGridConfig('default'),
			xAxis: this.getCategoryAxis(platforms.map(p => p.platform)),
			yAxis: [
				this.getValueAxis({ name: 'Score', max: 100 }),
				this.getValueAxis({ name: 'Rate (%)', max: 100, splitLine: { show: false } }),
			],
			series: [
				{
					name: 'QoE Score',
					type: 'bar' as const,
					data: platforms.map((p, i) => ({
						value: p.qoe_score,
						itemStyle: { color: palette[i % palette.length] },
					})),
					barWidth: '30%',
				},
				{
					name: 'Direct Play %',
					type: 'line' as const,
					yAxisIndex: 1,
					data: platforms.map(p => p.direct_play_rate),
					lineStyle: { color: CHART_COLORS.success, width: 2 },
					itemStyle: { color: CHART_COLORS.success },
					symbol: 'circle',
					symbolSize: 8,
				},
				{
					name: 'EBVS %',
					type: 'line' as const,
					yAxisIndex: 1,
					data: platforms.map(p => p.ebvs_rate),
					lineStyle: { color: CHART_COLORS.danger, width: 2 },
					itemStyle: { color: CHART_COLORS.danger },
					symbol: 'circle',
					symbolSize: 8,
				},
			],
		};

		this.setOption(option);
	}

	private getGradeColor(grade: string): string {
		switch (grade) {
			case 'A': return CHART_COLORS.success;
			case 'B': return CHART_COLORS.info;
			case 'C': return CHART_COLORS.warning;
			case 'D': return CHART_COLORS.orange;
			case 'F': return CHART_COLORS.danger;
			default: return CHART_COLORS.primary;
		}
	}
}

/**
 * QoE by Transcode Decision Renderer
 * Compares QoE across transcode decisions
 */
export class QoEByTranscodeRenderer extends BaseChartRenderer {
	render(data: QoEDashboard): void {
		if (!this.hasData(data?.by_transcode_decision)) {
			this.showEmptyState('No transcode data available');
			return;
		}

		const decisions = data.by_transcode_decision;

		const option = {
			...this.getBaseOption(),
			tooltip: this.getTooltip({
				trigger: 'item',
			}),
			legend: this.getLegend({
				data: decisions.map(d => d.transcode_decision),
				top: 0,
			}),
			series: [
				{
					name: 'Sessions by Transcode',
					type: 'pie' as const,
					radius: ['40%', '70%'],
					avoidLabelOverlap: true,
					itemStyle: {
						borderRadius: 4,
						borderColor: '#1a1a2e',
						borderWidth: 2,
					},
					label: {
						show: true,
						formatter: (params: EChartsLabelFormatterParams) => {
							const decision = decisions[params.dataIndex];
							return `{name|${params.name}}\n{score|QoE: ${decision.qoe_score.toFixed(0)}}`;
						},
						rich: {
							name: {
								fontSize: 12,
								color: '#a0a0a0',
							},
							score: {
								fontSize: 14,
								fontWeight: 'bold' as const,
								color: CHART_COLORS.primary,
							},
						},
					},
					emphasis: {
						label: {
							show: true,
							fontSize: 14,
							fontWeight: 'bold' as const,
						},
					},
					data: decisions.map((d, i) => ({
						value: d.session_count,
						name: d.transcode_decision,
						itemStyle: {
							color: this.getTranscodeColor(d.transcode_decision, i),
						},
					})),
				},
			],
		};

		this.setOption(option);
	}

	private getTranscodeColor(decision: string, index: number): string {
		const lowerDecision = decision.toLowerCase();
		if (lowerDecision.includes('direct') && lowerDecision.includes('play')) {
			return CHART_COLORS.success;
		}
		if (lowerDecision.includes('transcode')) {
			return CHART_COLORS.warning;
		}
		if (lowerDecision.includes('copy')) {
			return CHART_COLORS.info;
		}
		const palette = getChartPalette();
		return palette[index % palette.length];
	}
}

/**
 * QoE Issues Renderer
 * Shows top quality issues with severity indicators
 */
export class QoEIssuesRenderer extends BaseChartRenderer {
	render(data: QoEDashboard): void {
		if (!this.hasData(data?.top_issues)) {
			this.showEmptyState('No issues detected');
			return;
		}

		const issues = data.top_issues;

		const option = {
			...this.getBaseOption(),
			tooltip: this.getTooltip({
				trigger: 'axis',
				axisPointer: { type: 'shadow' },
				formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
					const p = Array.isArray(params) ? params[0] : params;
					const issue = issues[p.dataIndex];
					return `
						<div style="font-weight: 600; margin-bottom: 8px;">${issue.title}</div>
						<div style="margin: 4px 0; color: #888; max-width: 300px;">
							${issue.description}
						</div>
						<div style="margin: 8px 0;">
							<span style="color: #888;">Affected:</span>
							<span style="font-weight: 500;">${issue.affected_sessions.toLocaleString()} sessions (${issue.impact_percentage.toFixed(1)}%)</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Severity:</span>
							<span style="color: ${this.getSeverityColor(issue.severity)}; font-weight: 600;">${issue.severity.toUpperCase()}</span>
						</div>
						<div style="margin: 8px 0; padding: 8px; background: rgba(124, 58, 237, 0.1); border-radius: 4px;">
							<span style="color: ${CHART_COLORS.primary};">Recommendation:</span>
							<div style="color: #a0a0a0; margin-top: 4px;">${issue.recommendation}</div>
						</div>
					`;
				},
			}),
			grid: {
				left: '30%',
				right: '15%',
				top: '10%',
				bottom: '10%',
			},
			xAxis: {
				type: 'value' as const,
				...this.getAxisStyle(),
				axisLabel: {
					...this.getAxisStyle().axisLabel,
					formatter: '{value}%',
				},
			},
			yAxis: {
				type: 'category' as const,
				data: issues.map(i => this.truncateText(i.title, 25)),
				axisLabel: { color: '#a0a0a0', fontSize: 11 },
				axisTick: { show: false },
				axisLine: { show: false },
			},
			series: [
				{
					type: 'bar' as const,
					data: issues.map(i => ({
						value: i.impact_percentage,
						itemStyle: {
							color: this.getSeverityColor(i.severity),
							borderRadius: [0, 4, 4, 0],
						},
					})),
					barWidth: '60%',
					label: {
						show: true,
						position: 'right' as const,
						formatter: (params: EChartsLabelFormatterParams) => {
							const value = typeof params.value === 'number' ? params.value : 0;
							return `${value.toFixed(1)}%`;
						},
						fontSize: 11,
						color: '#a0a0a0',
					},
				},
			],
		};

		this.setOption(option);
	}

	private getSeverityColor(severity: string): string {
		switch (severity) {
			case 'critical': return CHART_COLORS.danger;
			case 'warning': return CHART_COLORS.warning;
			case 'info': return CHART_COLORS.info;
			default: return CHART_COLORS.primary;
		}
	}

	private truncateText(text: string, maxLength: number): string {
		if (text.length <= maxLength) return text;
		return text.substring(0, maxLength - 3) + '...';
	}
}
