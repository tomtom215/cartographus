// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Cohort Retention Chart Renderer
 *
 * Renders cohort retention analysis using:
 * - Heatmap for retention matrix
 * - Line chart for retention curve
 * - Summary statistics
 *
 * Based on industry best practices from Mixpanel, Amplitude, and Tableau.
 */

import type { CohortRetentionAnalytics, CohortData, RetentionPoint, WeekRetention } from '../../types/enhanced-analytics';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS, getHeatmapColors } from '../config/colors';
import { getGridConfig } from '../config/grid';
import type {
  EChartsCallbackDataParams,
  EChartsLabelFormatterParams,
} from '../types';

/**
 * Cohort Retention Heatmap Renderer
 */
export class CohortRetentionHeatmapRenderer extends BaseChartRenderer {
	render(data: CohortRetentionAnalytics): void {
		if (!this.hasData(data?.cohorts)) {
			this.showEmptyState('No cohort data available');
			return;
		}

		const cohorts = data.cohorts;
		const maxWeeks = this.getMaxWeeks(cohorts);
		const heatmapData = this.buildHeatmapData(cohorts, maxWeeks);
		const heatmapColors = getHeatmapColors();

		const option = {
			...this.getBaseOption(),
			tooltip: this.getTooltip({
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				formatter: ((params: EChartsCallbackDataParams) => this.createHeatmapTooltip(params, cohorts)) as any,
			}),
			grid: getGridConfig('heatmap'),
			xAxis: {
				type: 'category' as const,
				data: this.getWeekLabels(maxWeeks),
				position: 'top' as const,
				axisLabel: {
					color: '#a0a0a0',
					fontSize: 11,
				},
				axisTick: { show: false },
				axisLine: { show: false },
				splitLine: { show: false },
			},
			yAxis: {
				type: 'category' as const,
				data: cohorts.map(c => c.cohort_week),
				axisLabel: {
					color: '#a0a0a0',
					fontSize: 11,
				},
				axisTick: { show: false },
				axisLine: { show: false },
				splitLine: { show: false },
			},
			visualMap: {
				min: 0,
				max: 100,
				calculable: true,
				orient: 'horizontal' as const,
				left: 'center',
				bottom: 10,
				inRange: {
					color: heatmapColors,
				},
				textStyle: {
					color: '#a0a0a0',
				},
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				formatter: ((value: number) => `${value.toFixed(0)}%`) as any,
			},
			series: [
				{
					name: 'Retention',
					type: 'heatmap' as const,
					data: heatmapData,
					label: {
						show: true,
						// eslint-disable-next-line @typescript-eslint/no-explicit-any
						formatter: ((params: EChartsLabelFormatterParams) => {
							const valueArr = params.value as (number | null)[];
							const value = valueArr[2];
							if (value === null || value === undefined) return '';
							return `${value.toFixed(0)}%`;
						}) as any,
						fontSize: 10,
						color: '#fff',
					},
					emphasis: {
						itemStyle: {
							shadowBlur: 10,
							shadowColor: 'rgba(0, 0, 0, 0.5)',
						},
					},
				},
			],
		};

		this.setOption(option);
	}

	private getMaxWeeks(cohorts: CohortData[]): number {
		return Math.max(...cohorts.map(c => c.retention.length));
	}

	private getWeekLabels(maxWeeks: number): string[] {
		return Array.from({ length: maxWeeks }, (_, i) =>
			i === 0 ? 'Week 0' : `Week ${i}`
		);
	}

	private buildHeatmapData(cohorts: CohortData[], maxWeeks: number): [number, number, number | null][] {
		const data: [number, number, number | null][] = [];

		cohorts.forEach((cohort, yIndex) => {
			for (let weekOffset = 0; weekOffset < maxWeeks; weekOffset++) {
				const retention = cohort.retention.find(r => r.week_offset === weekOffset);
				const value = retention ? retention.retention_rate : null;
				data.push([weekOffset, yIndex, value]);
			}
		});

		return data;
	}

	private createHeatmapTooltip(params: EChartsCallbackDataParams, cohorts: CohortData[]): string {
		const valueArr = params.value as (number | null)[];
		const [weekOffset, cohortIndex, value] = valueArr;

		if (cohortIndex === null || weekOffset === null || value === null) {
			return 'No data';
		}

		const cohort = cohorts[cohortIndex];
		if (!cohort) {
			return 'No data';
		}

		const retention = cohort.retention.find((r: WeekRetention) => r.week_offset === weekOffset);

		return `
			<div style="font-weight: 600; margin-bottom: 8px;">
				${cohort.cohort_week}
			</div>
			<div style="margin: 4px 0;">
				<span style="color: #888;">Week ${weekOffset}:</span>
				<span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${value.toFixed(1)}%</span>
			</div>
			<div style="margin: 4px 0;">
				<span style="color: #888;">Active Users:</span>
				<span style="font-weight: 500;">${retention?.active_users || 0}</span>
			</div>
			<div style="margin: 4px 0;">
				<span style="color: #888;">Initial Cohort:</span>
				<span style="font-weight: 500;">${cohort.initial_users}</span>
			</div>
		`;
	}
}

/**
 * Cohort Retention Curve Renderer
 * Shows average retention across all cohorts over time
 */
export class CohortRetentionCurveRenderer extends BaseChartRenderer {
	render(data: CohortRetentionAnalytics): void {
		if (!this.hasData(data?.retention_curve)) {
			this.showEmptyState('No retention curve data available');
			return;
		}

		const curve = data.retention_curve;
		const weeks = curve.map(p => `Week ${p.week_offset}`);

		const option = {
			...this.getBaseOption(),
			tooltip: this.getTooltip({
				trigger: 'axis',
				formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => this.createCurveTooltip(params, curve),
			}),
			legend: this.getLegend({
				data: ['Average', 'Median', 'Min/Max Range'],
				top: 0,
			}),
			grid: getGridConfig('default'),
			xAxis: this.getCategoryAxis(weeks),
			yAxis: this.getValueAxis({
				name: 'Retention Rate',
				max: 100,
				axisLabel: {
					formatter: '{value}%',
				},
			}),
			series: [
				// Min/Max area
				{
					name: 'Min/Max Range',
					type: 'line' as const,
					data: curve.map(p => [p.min_retention, p.max_retention]),
					areaStyle: {
						color: 'rgba(124, 58, 237, 0.1)',
					},
					lineStyle: { opacity: 0 },
					stack: 'range',
					symbol: 'none',
				},
				// Average line
				{
					name: 'Average',
					type: 'line' as const,
					data: curve.map(p => p.average_retention),
					smooth: true,
					lineStyle: { color: CHART_COLORS.primary, width: 3 },
					itemStyle: { color: CHART_COLORS.primary },
					areaStyle: {
						color: {
							type: 'linear' as const,
							x: 0, y: 0, x2: 0, y2: 1,
							colorStops: [
								{ offset: 0, color: 'rgba(124, 58, 237, 0.4)' },
								{ offset: 1, color: 'rgba(124, 58, 237, 0.05)' },
							],
						},
					},
				},
				// Median line
				{
					name: 'Median',
					type: 'line' as const,
					data: curve.map(p => p.median_retention),
					smooth: true,
					lineStyle: { color: CHART_COLORS.secondary, width: 2, type: 'dashed' as const },
					itemStyle: { color: CHART_COLORS.secondary },
				},
			],
		};

		this.setOption(option);
	}

	private createCurveTooltip(params: EChartsCallbackDataParams | EChartsCallbackDataParams[], curve: RetentionPoint[]): string {
		if (!Array.isArray(params) || params.length === 0) return '';

		const weekIndex = params[0].dataIndex;
		const point = curve[weekIndex];

		if (!point) return '';

		return `
			<div style="font-weight: 600; margin-bottom: 8px;">
				Week ${point.week_offset}
			</div>
			<div style="margin: 4px 0;">
				<span style="display: inline-block; width: 10px; height: 10px; border-radius: 50%; background: ${CHART_COLORS.primary}; margin-right: 8px;"></span>
				<span style="color: #888;">Average:</span>
				<span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${point.average_retention.toFixed(1)}%</span>
			</div>
			<div style="margin: 4px 0;">
				<span style="display: inline-block; width: 10px; height: 10px; border-radius: 50%; background: ${CHART_COLORS.secondary}; margin-right: 8px;"></span>
				<span style="color: #888;">Median:</span>
				<span style="font-weight: 500;">${point.median_retention.toFixed(1)}%</span>
			</div>
			<div style="margin: 4px 0;">
				<span style="color: #888;">Range:</span>
				<span style="font-weight: 500;">${point.min_retention.toFixed(1)}% - ${point.max_retention.toFixed(1)}%</span>
			</div>
			<div style="margin: 4px 0;">
				<span style="color: #888;">Cohorts:</span>
				<span style="font-weight: 500;">${point.cohorts_with_data}</span>
			</div>
		`;
	}
}

/**
 * Cohort Summary Stats Renderer
 * Shows key retention metrics as gauges/numbers
 */
export class CohortSummaryRenderer extends BaseChartRenderer {
	render(data: CohortRetentionAnalytics): void {
		if (!data?.summary) {
			this.showEmptyState('No summary data available');
			return;
		}

		const summary = data.summary;

		const option = {
			...this.getBaseOption(),
			series: [
				// Week 1 Retention Gauge
				{
					type: 'gauge' as const,
					center: ['25%', '50%'],
					radius: '70%',
					startAngle: 200,
					endAngle: -20,
					min: 0,
					max: 100,
					splitNumber: 5,
					axisLine: {
						lineStyle: {
							width: 15,
							color: [
								[0.3, CHART_COLORS.danger] as [number, string],
								[0.6, CHART_COLORS.warning] as [number, string],
								[1, CHART_COLORS.success] as [number, string],
							],
						},
					},
					pointer: {
						itemStyle: {
							color: 'auto',
						},
					},
					axisTick: { show: false },
					splitLine: { show: false },
					axisLabel: { show: false },
					title: {
						offsetCenter: [0, '80%'],
						fontSize: 12,
						color: '#888',
					},
					detail: {
						fontSize: 24,
						offsetCenter: [0, '30%'],
						valueAnimation: true,
						formatter: (value: number) => `${value.toFixed(0)}%`,
						color: 'inherit',
					},
					data: [{ value: summary.week1_retention, name: 'Week 1 Retention' }],
				},
				// Week 4 Retention Gauge
				{
					type: 'gauge' as const,
					center: ['50%', '50%'],
					radius: '70%',
					startAngle: 200,
					endAngle: -20,
					min: 0,
					max: 100,
					splitNumber: 5,
					axisLine: {
						lineStyle: {
							width: 15,
							color: [
								[0.2, CHART_COLORS.danger] as [number, string],
								[0.4, CHART_COLORS.warning] as [number, string],
								[1, CHART_COLORS.success] as [number, string],
							],
						},
					},
					pointer: {
						itemStyle: {
							color: 'auto',
						},
					},
					axisTick: { show: false },
					splitLine: { show: false },
					axisLabel: { show: false },
					title: {
						offsetCenter: [0, '80%'],
						fontSize: 12,
						color: '#888',
					},
					detail: {
						fontSize: 24,
						offsetCenter: [0, '30%'],
						valueAnimation: true,
						formatter: (value: number) => `${value.toFixed(0)}%`,
						color: 'inherit',
					},
					data: [{ value: summary.week4_retention, name: 'Week 4 Retention' }],
				},
				// Week 12 Retention Gauge
				{
					type: 'gauge' as const,
					center: ['75%', '50%'],
					radius: '70%',
					startAngle: 200,
					endAngle: -20,
					min: 0,
					max: 100,
					splitNumber: 5,
					axisLine: {
						lineStyle: {
							width: 15,
							color: [
								[0.15, CHART_COLORS.danger] as [number, string],
								[0.3, CHART_COLORS.warning] as [number, string],
								[1, CHART_COLORS.success] as [number, string],
							],
						},
					},
					pointer: {
						itemStyle: {
							color: 'auto',
						},
					},
					axisTick: { show: false },
					splitLine: { show: false },
					axisLabel: { show: false },
					title: {
						offsetCenter: [0, '80%'],
						fontSize: 12,
						color: '#888',
					},
					detail: {
						fontSize: 24,
						offsetCenter: [0, '30%'],
						valueAnimation: true,
						formatter: (value: number) => `${value.toFixed(0)}%`,
						color: 'inherit',
					},
					data: [{ value: summary.week12_retention, name: 'Week 12 Retention' }],
				},
			],
		};

		this.setOption(option);
	}
}
