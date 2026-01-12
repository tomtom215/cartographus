// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * User Network Graph Chart Renderers
 *
 * Renders user relationship network graphs using:
 * - Force-directed graph layout for network visualization
 * - Cluster visualization
 * - Connection statistics
 *
 * Enables social viewing pattern analysis and community detection.
 */

import type { UserNetworkGraph, UserNode, UserCluster, UserEdge } from '../../types/enhanced-analytics';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS } from '../config/colors';
import type { EChartsCallbackDataParams, EChartsLabelFormatterParams } from '../types';

/**
 * User Network Force Graph Renderer
 * Shows force-directed network visualization
 */
export class UserNetworkGraphRenderer extends BaseChartRenderer {
	render(data: UserNetworkGraph): void {
		if (!this.hasData(data?.nodes)) {
			this.showEmptyState('No network data available');
			return;
		}

		const nodes = data.nodes;
		const edges = data.edges;
		const clusters = data.clusters;

		const option = {
			...this.getBaseOption(),
			tooltip: {
				trigger: 'item' as const,
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
			formatter: (params: any) => {
					if (params.dataType === 'node') {
						return this.createNodeTooltip(params.data?.nodeData || params.data as unknown as UserNode, clusters);
					}
					if (params.dataType === 'edge') {
						return this.createEdgeTooltip(params.data?.edgeData || params.data as unknown as UserEdge, nodes);
					}
					return '';
				},
			},
			legend: [
				{
					data: clusters.slice(0, 10).map(c => c.name),
					left: 'left',
					top: 10,
					orient: 'vertical' as const,
					textStyle: { color: '#a0a0a0', fontSize: 11 },
				},
			],
			series: [
				{
					name: 'User Network',
					type: 'graph' as const,
					layout: 'force' as const,
					data: nodes.map(node => ({
						id: node.id,
						name: node.username,
						value: node.playback_count,
						symbolSize: Math.min(50, Math.max(15, node.node_size)),
						category: node.cluster_id,
						itemStyle: {
							color: node.node_color,
							borderColor: node.is_central ? CHART_COLORS.primary : 'transparent',
							borderWidth: node.is_central ? 3 : 0,
						},
						label: {
							show: node.connection_count >= 3,
							formatter: '{b}',
							position: 'right' as const,
							fontSize: 10,
							color: '#a0a0a0',
						},
						// Store full node data for tooltip
						nodeData: node,
					})),
					links: edges.map(edge => ({
						source: edge.source,
						target: edge.target,
						value: edge.weight,
						lineStyle: {
							width: Math.min(5, Math.max(1, edge.edge_width)),
							color: this.getEdgeColor(edge.connection_type),
							curveness: 0.1,
							opacity: 0.6,
						},
						// Store full edge data for tooltip
						edgeData: edge,
					})),
					categories: clusters.slice(0, 10).map(c => ({
						name: c.name,
						itemStyle: { color: c.cluster_color },
					})),
					roam: true,
					draggable: true,
					force: {
						repulsion: 200,
						gravity: 0.1,
						edgeLength: [50, 200],
						layoutAnimation: true,
					},
					emphasis: {
						focus: 'adjacency' as const,
						lineStyle: {
							width: 4,
						},
					},
					edgeSymbol: ['none', 'arrow'],
					edgeSymbolSize: [0, 8],
				},
			],
			animationDuration: 1500,
			animationEasingUpdate: 'quinticInOut' as const,
		};

		this.setOption(option);
	}

	private createNodeTooltip(node: UserNode, clusters: UserCluster[]): string {
		const cluster = clusters.find(c => c.id === node.cluster_id);

		return `
			<div style="font-weight: 600; margin-bottom: 8px; font-size: 14px;">
				${node.username}
				${node.is_central ? '<span style="color: ' + CHART_COLORS.primary + ';"> (Hub)</span>' : ''}
			</div>
			<div style="margin: 4px 0;">
				<span style="color: #888;">Playbacks:</span>
				<span style="font-weight: 500;">${node.playback_count.toLocaleString()}</span>
			</div>
			<div style="margin: 4px 0;">
				<span style="color: #888;">Watch Hours:</span>
				<span style="font-weight: 500;">${node.total_watch_hours.toFixed(1)}h</span>
			</div>
			<div style="margin: 4px 0;">
				<span style="color: #888;">Connections:</span>
				<span style="font-weight: 500;">${node.connection_count}</span>
			</div>
			${node.top_genre ? `
			<div style="margin: 4px 0;">
				<span style="color: #888;">Top Genre:</span>
				<span style="font-weight: 500;">${node.top_genre}</span>
			</div>
			` : ''}
			<div style="margin: 4px 0;">
				<span style="color: #888;">Last Active:</span>
				<span style="font-weight: 500;">${new Date(node.last_active).toLocaleDateString()}</span>
			</div>
			${cluster ? `
			<div style="margin: 8px 0; padding: 6px 8px; background: ${cluster.cluster_color}22; border-radius: 4px; border-left: 3px solid ${cluster.cluster_color};">
				<span style="color: ${cluster.cluster_color}; font-weight: 500;">${cluster.name}</span>
				<span style="color: #888; font-size: 11px;"> (${cluster.characteristic_type})</span>
			</div>
			` : ''}
		`;
	}

	private createEdgeTooltip(edge: UserEdge, nodes: UserNode[]): string {
		const source = nodes.find(n => n.id === edge.source);
		const target = nodes.find(n => n.id === edge.target);

		return `
			<div style="font-weight: 600; margin-bottom: 8px;">
				${source?.username || edge.source} <span style="color: #888;">&harr;</span> ${target?.username || edge.target}
			</div>
			<div style="margin: 4px 0;">
				<span style="color: #888;">Type:</span>
				<span style="color: ${this.getEdgeColor(edge.connection_type)}; font-weight: 500;">
					${this.formatConnectionType(edge.connection_type)}
				</span>
			</div>
			${edge.shared_sessions ? `
			<div style="margin: 4px 0;">
				<span style="color: #888;">Shared Sessions:</span>
				<span style="font-weight: 500;">${edge.shared_sessions}</span>
			</div>
			` : ''}
			${edge.content_overlap ? `
			<div style="margin: 4px 0;">
				<span style="color: #888;">Content Overlap:</span>
				<span style="font-weight: 500;">${(edge.content_overlap * 100).toFixed(0)}%</span>
			</div>
			` : ''}
			${edge.top_shared_content?.length ? `
			<div style="margin: 8px 0;">
				<span style="color: #888;">Shared Content:</span>
				<ul style="margin: 4px 0 0 16px; padding: 0; font-size: 11px;">
					${edge.top_shared_content.slice(0, 3).map((c: string) => `<li>${c}</li>`).join('')}
				</ul>
			</div>
			` : ''}
			${edge.first_interaction ? `
			<div style="margin: 4px 0; font-size: 11px; color: #666;">
				First: ${new Date(edge.first_interaction as string).toLocaleDateString()} &mdash;
				Last: ${new Date((edge.last_interaction || edge.first_interaction) as string).toLocaleDateString()}
			</div>
			` : ''}
		`;
	}

	private getEdgeColor(connectionType: string): string {
		switch (connectionType) {
			case 'shared_session': return CHART_COLORS.success;
			case 'watch_party': return CHART_COLORS.primary;
			case 'content_overlap': return CHART_COLORS.secondary;
			case 'sequential': return CHART_COLORS.info;
			default: return CHART_COLORS.primary;
		}
	}

	private formatConnectionType(type: string): string {
		return type.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
	}
}

/**
 * Network Summary Stats Renderer
 * Shows network statistics as gauges and numbers
 */
export class NetworkSummaryRenderer extends BaseChartRenderer {
	render(data: UserNetworkGraph): void {
		if (!data?.summary) {
			this.showEmptyState('No summary data available');
			return;
		}

		const summary = data.summary;

		const option = {
			...this.getBaseOption(),
			title: {
				text: `Network Type: ${this.formatNetworkType(summary.network_type)}`,
				left: 'center',
				top: 10,
				textStyle: {
					color: CHART_COLORS.primary,
					fontSize: 16,
					fontWeight: 'bold' as const,
				},
			},
			series: [
				// Network Density Gauge
				{
					type: 'gauge' as const,
					center: ['25%', '55%'],
					radius: '50%',
					startAngle: 200,
					endAngle: -20,
					min: 0,
					max: 1,
					splitNumber: 5,
					axisLine: {
						lineStyle: {
							width: 12,
							color: [
								[0.2, CHART_COLORS.info] as [number, string],
								[0.5, CHART_COLORS.success] as [number, string],
								[1, CHART_COLORS.primary] as [number, string],
							],
						},
					},
					pointer: { itemStyle: { color: 'auto' } },
					axisTick: { show: false },
					splitLine: { show: false },
					axisLabel: { show: false },
					title: {
						offsetCenter: [0, '80%'],
						fontSize: 11,
						color: '#888',
					},
					detail: {
						fontSize: 18,
						offsetCenter: [0, '30%'],
						formatter: (value: number) => `${(value * 100).toFixed(1)}%`,
						color: 'inherit',
					},
					data: [{ value: summary.network_density, name: 'Density' }],
				},
				// Avg Connections Gauge
				{
					type: 'gauge' as const,
					center: ['50%', '55%'],
					radius: '50%',
					startAngle: 200,
					endAngle: -20,
					min: 0,
					max: Math.max(10, summary.max_connections_count),
					axisLine: {
						lineStyle: {
							width: 12,
							color: [
								[0.3, CHART_COLORS.info] as [number, string],
								[0.6, CHART_COLORS.success] as [number, string],
								[1, CHART_COLORS.primary] as [number, string],
							],
						},
					},
					pointer: { itemStyle: { color: 'auto' } },
					axisTick: { show: false },
					splitLine: { show: false },
					axisLabel: { show: false },
					title: {
						offsetCenter: [0, '80%'],
						fontSize: 11,
						color: '#888',
					},
					detail: {
						fontSize: 18,
						offsetCenter: [0, '30%'],
						formatter: (value: number) => value.toFixed(1),
						color: 'inherit',
					},
					data: [{ value: summary.avg_connections_per_user, name: 'Avg Connections' }],
				},
				// Clusters Gauge
				{
					type: 'gauge' as const,
					center: ['75%', '55%'],
					radius: '50%',
					startAngle: 200,
					endAngle: -20,
					min: 0,
					max: Math.max(20, summary.total_clusters * 2),
					axisLine: {
						lineStyle: {
							width: 12,
							color: [
								[0.3, CHART_COLORS.info] as [number, string],
								[0.6, CHART_COLORS.success] as [number, string],
								[1, CHART_COLORS.primary] as [number, string],
							],
						},
					},
					pointer: { itemStyle: { color: 'auto' } },
					axisTick: { show: false },
					splitLine: { show: false },
					axisLabel: { show: false },
					title: {
						offsetCenter: [0, '80%'],
						fontSize: 11,
						color: '#888',
					},
					detail: {
						fontSize: 18,
						offsetCenter: [0, '30%'],
						formatter: (value: number) => value.toString(),
						color: 'inherit',
					},
					data: [{ value: summary.total_clusters, name: 'Clusters' }],
				},
			],
		};

		this.setOption(option);
	}

	private formatNetworkType(type: string): string {
		return type.charAt(0).toUpperCase() + type.slice(1);
	}
}

/**
 * Cluster Distribution Renderer
 * Shows cluster sizes and characteristics
 */
export class ClusterDistributionRenderer extends BaseChartRenderer {
	render(data: UserNetworkGraph): void {
		if (!this.hasData(data?.clusters)) {
			this.showEmptyState('No cluster data available');
			return;
		}

		const clusters = data.clusters;

		const option = {
			...this.getBaseOption(),
			tooltip: this.getTooltip({
				trigger: 'item',
				formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
					const p = Array.isArray(params) ? params[0] : params;
					const cluster = clusters.find(c => c.name === p.name);
					if (!cluster) return '';
					return `
						<div style="font-weight: 600; margin-bottom: 8px;">${cluster.name}</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Type:</span>
							<span style="color: ${cluster.cluster_color}; font-weight: 500;">${cluster.characteristic_type}</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Users:</span>
							<span style="font-weight: 500;">${cluster.user_count}</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Density:</span>
							<span style="font-weight: 500;">${(cluster.density * 100).toFixed(0)}%</span>
						</div>
						<div style="margin: 4px 0;">
							<span style="color: #888;">Avg Watch Hours:</span>
							<span style="font-weight: 500;">${cluster.avg_watch_hours.toFixed(1)}h</span>
						</div>
						${cluster.top_genres?.length ? `
						<div style="margin: 8px 0;">
							<span style="color: #888;">Top Genres:</span>
							<div style="margin-top: 4px; font-size: 11px;">
								${cluster.top_genres.slice(0, 3).join(', ')}
							</div>
						</div>
						` : ''}
					`;
				},
			}),
			series: [
				{
					type: 'pie' as const,
					radius: ['35%', '70%'],
					center: ['50%', '50%'],
					avoidLabelOverlap: true,
					itemStyle: {
						borderRadius: 4,
						borderColor: '#1a1a2e',
						borderWidth: 2,
					},
					label: {
						show: true,
						formatter: (params: EChartsLabelFormatterParams) => {
							const cluster = clusters.find(c => c.name === params.name);
							if (!cluster) return params.name || '';
							return `{name|${cluster.name}}\n{type|${cluster.characteristic_type}}`;
						},
						rich: {
							name: {
								fontSize: 12,
								fontWeight: 'bold' as const,
								color: '#fff',
							},
							type: {
								fontSize: 10,
								color: '#888',
							},
						},
					},
					emphasis: {
						label: {
							show: true,
							fontSize: 14,
							fontWeight: 'bold' as const,
						},
						itemStyle: {
							shadowBlur: 10,
							shadowColor: 'rgba(0, 0, 0, 0.5)',
						},
					},
					data: clusters.map(c => ({
						value: c.user_count,
						name: c.name,
						itemStyle: { color: c.cluster_color },
					})),
				},
			],
		};

		this.setOption(option);
	}
}

/**
 * Connection Type Distribution Renderer
 * Shows breakdown of connection types
 */
export class ConnectionTypeRenderer extends BaseChartRenderer {
	render(data: UserNetworkGraph): void {
		if (!this.hasData(data?.edges)) {
			this.showEmptyState('No connection data available');
			return;
		}

		// Aggregate by connection type
		const typeCounts: Record<string, number> = {};
		data.edges.forEach(edge => {
			typeCounts[edge.connection_type] = (typeCounts[edge.connection_type] || 0) + 1;
		});

		const types = Object.entries(typeCounts)
			.map(([type, count]) => ({ type, count }))
			.sort((a, b) => b.count - a.count);

		const option = {
			...this.getBaseOption(),
			tooltip: this.getTooltip({
				trigger: 'item',
				formatter: (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
					const p = Array.isArray(params) ? params[0] : params;
					const value = typeof p.value === 'number' ? p.value : 0;
					const percent = p.percent ?? 0;
					return `
						<div style="font-weight: 600; margin-bottom: 4px;">
							${this.formatConnectionType(p.name || '')}
						</div>
						<div>
							<span style="color: #888;">Connections:</span>
							<span style="font-weight: 500;">${value.toLocaleString()} (${percent.toFixed(1)}%)</span>
						</div>
					`;
				},
			}),
			legend: this.getLegend({
				data: types.map(t => this.formatConnectionType(t.type)),
				bottom: 10,
				orient: 'horizontal' as const,
			}),
			series: [
				{
					type: 'pie' as const,
					radius: ['0%', '65%'],
					center: ['50%', '45%'],
					roseType: 'area' as const,
					itemStyle: {
						borderRadius: 5,
						borderColor: '#1a1a2e',
						borderWidth: 2,
					},
					label: {
						show: true,
						formatter: '{b}: {c}',
						fontSize: 11,
					},
					data: types.map(t => ({
						value: t.count,
						name: this.formatConnectionType(t.type),
						itemStyle: { color: this.getEdgeColor(t.type) },
					})),
				},
			],
		};

		this.setOption(option);
	}

	private formatConnectionType(type: string): string {
		return type.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
	}

	private getEdgeColor(connectionType: string): string {
		switch (connectionType) {
			case 'shared_session': return CHART_COLORS.success;
			case 'watch_party': return CHART_COLORS.primary;
			case 'content_overlap': return CHART_COLORS.secondary;
			case 'sequential': return CHART_COLORS.info;
			default: return CHART_COLORS.primary;
		}
	}
}
