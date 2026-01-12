// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Charts library public API
 * Re-exports ChartManager for use in the main application
 */

export { ChartManager } from './ChartManager';

// Export utilities for external use if needed
export { ChartFormatters } from './utils/formatters';
export { ChartHelpers } from './utils/chartHelpers';

// Export config functions if needed
export { getEnhancedTooltip } from './config/tooltips';
export { getEnhancedLegend, getAccessibleLegend, getAccessibleIcon, getAccessibleIcons } from './config/legends';
export { CHART_COLORS, COLOR_PALETTES, STYLE_CONSTANTS } from './config/colors';

// Enhanced Analytics Chart Renderers
export {
	CohortRetentionHeatmapRenderer,
	CohortRetentionCurveRenderer,
	CohortSummaryRenderer,
} from './renderers/CohortRetentionChartRenderer';

export {
	QoEScoreGaugeRenderer,
	QoESummaryMetricsRenderer,
	QoETrendsRenderer,
	QoEByPlatformRenderer,
	QoEByTranscodeRenderer,
	QoEIssuesRenderer,
} from './renderers/QoEChartRenderer';

export {
	DataQualityScoreRenderer,
	FieldQualityRenderer,
	QualityTrendsRenderer,
	SourceQualityRenderer,
	DataQualityIssuesRenderer,
} from './renderers/DataQualityChartRenderer';

export {
	UserNetworkGraphRenderer,
	NetworkSummaryRenderer,
	ClusterDistributionRenderer,
	ConnectionTypeRenderer,
} from './renderers/UserNetworkChartRenderer';
