// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Data Quality Chart Renderers
 *
 * Renders data quality monitoring dashboards using:
 * - Gauge charts for overall quality scores
 * - Bar charts for field-level quality
 * - Line charts for quality trends
 * - Treemap for source breakdown
 *
 * Provides production-grade observability for data pipelines.
 */

import type { DataQualityReport, DataQualitySummary, SourceQuality } from '../../types/enhanced-analytics';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS } from '../config/colors';
import { getGridConfig } from '../config/grid';
import type { EChartsCallbackDataParams, EChartsLabelFormatterParams } from '../types';

/**
 * Data Quality Score Gauge Renderer
 * Shows overall data quality score with component breakdown
 */
export class DataQualityScoreRenderer extends BaseChartRenderer {
	render(data: DataQualityReport): void {
		if (!data?.summary) {
			this.showEmptyState('No quality data available');
			return;
		}

		const summary = data.summary;
		const gradeColor = this.getGradeColor(summary.grade);

		const option = {
			...this.getBaseOption(),
			series: [
				// Main gauge
				{
					type: 'gauge' as const,
					startAngle: 180,
					endAngle: 0,
					center: ['50%', '70%'],
					radius: '90%',
					min: 0,
					max: 100,
					splitNumber: 10,
					axisLine: {
						lineStyle: {
							width: 25,
							color: [
								[0.4, CHART_COLORS.danger],
								[0.6, CHART_COLORS.warning],
								[0.8, CHART_COLORS.info],
								[1, CHART_COLORS.success],
							],
						},
					},
					pointer: {
						icon: 'path://M12.8,0.7l12,40.1H0.7L12.8,0.7z',
						length: '55%',
						width: 12,
						offsetCenter: [0, '-25%'],
						itemStyle: { color: gradeColor },
					},
					axisTick: {
						length: 10,
						lineStyle: { color: 'auto', width: 1 },
					},
					splitLine: {
						length: 15,
						lineStyle: { color: 'auto', width: 2 },
					},
					axisLabel: {
						distance: 30,
						color: '#888',
						fontSize: 11,
					},
					title: {
						show: true,
						offsetCenter: [0, '15%'],
						fontSize: 14,
						color: '#888',
					},
					detail: {
						fontSize: 42,
						offsetCenter: [0, '-10%'],
						valueAnimation: true,
						formatter: (value: number) => `${value.toFixed(0)}`,
						color: gradeColor,
					},
					data: [
						{
							value: summary.overall_score,
							name: `Grade: ${summary.grade}`,
						},
					],
				},
				// Sub-scores as mini gauges
				...this.createSubGauges(summary),
			],
		};

		this.setOption(option);
	}

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	private createSubGauges(summary: DataQualitySummary): any[] {
		const subScores = [
			{ name: 'Completeness', value: summary.completeness_score, x: '20%' },
			{ name: 'Validity', value: summary.validity_score, x: '50%' },
			{ name: 'Consistency', value: summary.consistency_score, x: '80%' },
		];

		return subScores.map(score => ({
			type: 'gauge' as const,
			center: [score.x, '92%'],
			radius: '20%',
			startAngle: 180,
			endAngle: 0,
			min: 0,
			max: 100,
			axisLine: {
				lineStyle: {
					width: 8,
					color: [
						[0.5, CHART_COLORS.danger] as [number, string],
						[0.7, CHART_COLORS.warning] as [number, string],
						[1, CHART_COLORS.success],
					],
				},
			},
			pointer: { show: false },
			axisTick: { show: false },
			splitLine: { show: false },
			axisLabel: { show: false },
			title: {
				show: true,
				offsetCenter: [0, '85%'],
				fontSize: 10,
				color: '#888',
			},
			detail: {
				fontSize: 12,
				offsetCenter: [0, '30%'],
				formatter: (value: number) => `${value.toFixed(0)}%`,
				color: '#a0a0a0',
			},
			data: [{ value: score.value, name: score.name }],
		}));
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
 * Field Quality Renderer
 * Shows quality metrics per field as horizontal bars
 */
export class FieldQualityRenderer extends BaseChartRenderer {
	render(data: DataQualityReport): void {
		if (!this.hasData(data?.field_quality)) {
			this.showEmptyState('No field quality data available');
			return;
		}

		// Sort by quality score and take top 15
		const fields = [...data.field_quality]
			.sort((a, b) => a.quality_score - b.quality_score)
			.slice(0, 15);

		const option = {
			...this.getBaseOption(),
			tooltip: this.getTooltip({
				trigger: 'axis',
				axisPointer: { type: 'shadow' },
				formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
					const p = Array.isArray(params) ? params[0] : params;
					const field = fields[p.dataIndex];
					return `
						<div style="font-weight: 600; margin-bottom: 8px;">${field.field_name}</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Category:</span>
							<span style="font-weight: 500;">${field.category}</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Quality Score:</span>
							<span style="color: ${this.getStatusColor(field.status)}; font-weight: 600;">${field.quality_score.toFixed(1)}%</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Null Rate:</span>
							<span style="font-weight: 500;">${field.null_rate.toFixed(2)}%</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Invalid Rate:</span>
							<span style="font-weight: 500;">${field.invalid_rate.toFixed(2)}%</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Records:</span>
							<span style="font-weight: 500;">${field.total_records.toLocaleString()}</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Required:</span>
							<span style="font-weight: 500;">${field.is_required ? 'Yes' : 'No'}</span>
						</div>
					`;
				},
			}),
			grid: {
				left: '25%',
				right: '15%',
				top: '5%',
				bottom: '5%',
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
				data: fields.map(f => f.field_name),
				axisLabel: {
					color: '#a0a0a0',
					fontSize: 11,
					width: 100,
					overflow: 'truncate' as const,
				},
				axisTick: { show: false },
				axisLine: { show: false },
			},
			series: [
				{
					type: 'bar' as const,
					data: fields.map(f => ({
						value: f.quality_score,
						itemStyle: {
							color: this.getStatusColor(f.status),
							borderRadius: [0, 4, 4, 0],
						},
					})),
					barWidth: '70%',
					label: {
						show: true,
						position: 'right' as const,
						formatter: (params: EChartsLabelFormatterParams) => {
							const value = typeof params.value === 'number' ? params.value : 0;
							return `${value.toFixed(0)}%`;
						},
						fontSize: 10,
						color: '#a0a0a0',
					},
				},
			],
		};

		this.setOption(option);
	}

	private getStatusColor(status: string): string {
		switch (status) {
			case 'healthy': return CHART_COLORS.success;
			case 'warning': return CHART_COLORS.warning;
			case 'critical': return CHART_COLORS.danger;
			default: return CHART_COLORS.primary;
		}
	}
}

/**
 * Quality Trends Renderer
 * Shows data quality trends over time
 */
export class QualityTrendsRenderer extends BaseChartRenderer {
	render(data: DataQualityReport): void {
		if (!this.hasData(data?.daily_trends)) {
			this.showEmptyState('No trend data available');
			return;
		}

		const trends = data.daily_trends;
		const dates = trends.map(t =>
			new Date(t.date).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
		);

		const option = {
			...this.getBaseOption(),
			tooltip: this.getTooltip({
				trigger: 'axis',
			}),
			legend: this.getLegend({
				data: ['Quality Score', 'Null Rate', 'Invalid Rate', 'New Issues'],
				top: 0,
			}),
			grid: getGridConfig('withDataZoom'),
			xAxis: this.getCategoryAxis(dates, { boundaryGap: false }),
			yAxis: [
				this.getValueAxis({
					name: 'Score',
					max: 100,
					position: 'left',
				}),
				this.getValueAxis({
					name: 'Rate / Issues',
					position: 'right' as const,
					splitLine: { show: false },
				}),
			],
			dataZoom: this.getDataZoom('both'),
			series: [
				{
					name: 'Quality Score',
					type: 'line' as const,
					smooth: true,
					data: trends.map(t => t.overall_score),
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
					name: 'Null Rate',
					type: 'line' as const,
					yAxisIndex: 1,
					data: trends.map(t => t.null_rate),
					lineStyle: { color: CHART_COLORS.warning, width: 2 },
					itemStyle: { color: CHART_COLORS.warning },
				},
				{
					name: 'Invalid Rate',
					type: 'line' as const,
					yAxisIndex: 1,
					data: trends.map(t => t.invalid_rate),
					lineStyle: { color: CHART_COLORS.danger, width: 2 },
					itemStyle: { color: CHART_COLORS.danger },
				},
				{
					name: 'New Issues',
					type: 'bar' as const,
					yAxisIndex: 1,
					data: trends.map(t => t.new_issues),
					itemStyle: {
						color: CHART_COLORS.info,
						opacity: 0.6,
					},
					barWidth: '30%',
				},
			],
		};

		this.setOption(option);
	}
}

/**
 * Source Quality Renderer
 * Shows quality breakdown by data source
 */
export class SourceQualityRenderer extends BaseChartRenderer {
	render(data: DataQualityReport): void {
		if (!this.hasData(data?.source_breakdown)) {
			this.showEmptyState('No source data available');
			return;
		}

		const sources = data.source_breakdown;

		const option = {
			...this.getBaseOption(),
			tooltip: this.getTooltip({
				trigger: 'item',
				formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
					const p = Array.isArray(params) ? params[0] : params;
					const source = sources.find(s => s.source === p.name);
					if (!source) return '';
					return `
						<div style="font-weight: 600; margin-bottom: 8px;">${source.source}</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Events:</span>
							<span style="font-weight: 500;">${source.event_count.toLocaleString()} (${source.event_percentage.toFixed(1)}%)</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Quality Score:</span>
							<span style="color: ${this.getStatusColor(source.status)}; font-weight: 600;">${source.quality_score.toFixed(1)}%</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Null Rate:</span>
							<span style="font-weight: 500;">${source.null_rate.toFixed(2)}%</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Invalid Rate:</span>
							<span style="font-weight: 500;">${source.invalid_rate.toFixed(2)}%</span>
						</div>
						${source.top_issue ? `
						<div style="margin: 8px 0; padding: 8px; background: rgba(239, 68, 68, 0.1); border-radius: 4px;">
							<span style="color: ${CHART_COLORS.danger};">Top Issue:</span>
							<div style="color: #a0a0a0; margin-top: 4px;">${source.top_issue}</div>
						</div>
						` : ''}
					`;
				},
			}),
			series: [
				{
					type: 'treemap' as const,
					width: '90%',
					height: '85%',
					top: '5%',
					left: '5%',
					roam: false,
					nodeClick: false as const,
					breadcrumb: { show: false },
					label: {
						show: true,
						formatter: (params: EChartsLabelFormatterParams) => {
							const source = sources.find(s => s.source === params.name);
							if (!source) return params.name || '';
							return `{name|${params.name}}\n{score|${source.quality_score.toFixed(0)}%}`;
						},
						rich: {
							name: {
								fontSize: 14,
								fontWeight: 'bold' as const,
								color: '#fff',
							},
							score: {
								fontSize: 12,
								color: '#eee',
							},
						},
					},
					itemStyle: {
						borderColor: '#1a1a2e',
						borderWidth: 2,
						gapWidth: 2,
					},
					data: sources.map((s) => ({
						name: s.source,
						value: s.event_count,
						itemStyle: {
							color: this.getSourceColor(s),
						},
					})),
				},
			],
		};

		this.setOption(option);
	}

	private getStatusColor(status: string): string {
		switch (status) {
			case 'healthy': return CHART_COLORS.success;
			case 'warning': return CHART_COLORS.warning;
			case 'critical': return CHART_COLORS.danger;
			default: return CHART_COLORS.primary;
		}
	}

	private getSourceColor(source: SourceQuality): string {
		// Color based on quality status with transparency for smaller sources
		const baseColor = this.getStatusColor(source.status);
		const opacity = Math.max(0.5, source.event_percentage / 50);
		return this.adjustColorOpacity(baseColor, opacity);
	}

	private adjustColorOpacity(hex: string, opacity: number): string {
		// Convert hex to rgba
		const r = parseInt(hex.slice(1, 3), 16);
		const g = parseInt(hex.slice(3, 5), 16);
		const b = parseInt(hex.slice(5, 7), 16);
		return `rgba(${r}, ${g}, ${b}, ${opacity})`;
	}
}

/**
 * Data Quality Issues Renderer
 * Shows issues with severity and impact
 */
export class DataQualityIssuesRenderer extends BaseChartRenderer {
	render(data: DataQualityReport): void {
		if (!this.hasData(data?.issues)) {
			this.showEmptyState('No issues detected');
			return;
		}

		// Sort by impact and take top 10
		const issues = [...data.issues]
			.sort((a, b) => b.impact_percentage - a.impact_percentage)
			.slice(0, 10);

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
							<span style="color: #888;">Type:</span>
							<span style="font-weight: 500;">${issue.type.replace(/_/g, ' ')}</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Severity:</span>
							<span style="color: ${this.getSeverityColor(issue.severity)}; font-weight: 600;">${issue.severity.toUpperCase()}</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Affected:</span>
							<span style="font-weight: 500;">${issue.affected_records.toLocaleString()} records (${issue.impact_percentage.toFixed(1)}%)</span>
						</div>
						${issue.field ? `
						<div style="margin: 4px 0;">
							<span style="color: #888;">Field:</span>
							<span style="font-weight: 500;">${issue.field}</span>
						</div>
						` : ''}
						<div style="margin: 8px 0; padding: 8px; background: rgba(124, 58, 237, 0.1); border-radius: 4px;">
							<span style="color: ${CHART_COLORS.primary};">Recommendation:</span>
							<div style="color: #a0a0a0; margin-top: 4px;">${issue.recommendation}</div>
						</div>
						${issue.auto_resolvable ? `
						<div style="margin: 4px 0; color: ${CHART_COLORS.success};">
							Auto-resolvable
						</div>
						` : ''}
					`;
				},
			}),
			grid: {
				left: '35%',
				right: '15%',
				top: '5%',
				bottom: '5%',
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
				data: issues.map(i => this.truncateText(i.title, 30)),
				axisLabel: {
					color: '#a0a0a0',
					fontSize: 11,
				},
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
						fontSize: 10,
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
