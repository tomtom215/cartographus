// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Chart-specific type definitions
 */

import type { EChartsOption } from 'echarts';

/**
 * Configuration for chart renderer initialization
 */
export interface ChartRendererConfig {
    chartId: string;
    chart: import('echarts').ECharts;
}

/**
 * Base option type for all charts
 */
export type BaseChartOption = EChartsOption;

/**
 * Chart export format
 */
export type ChartExportFormat = 'png' | 'svg';

/**
 * Chart resize mode
 */
export type ChartResizeMode = 'responsive' | 'fixed';

/**
 * ECharts callback params for tooltip formatters and itemStyle color functions
 * This is a simplified type that renderers use internally after casting from ECharts params
 */
export interface EChartsCallbackDataParams {
    /** Component type: 'series' */
    componentType?: string;
    /** Series type: 'bar', 'line', 'pie', etc. */
    seriesType?: string;
    /** Series index in option.series */
    seriesIndex?: number;
    /** Series name */
    seriesName?: string;
    /** Data name (category axis value for bar/line charts) */
    name?: string;
    /** Data index in current series */
    dataIndex: number;
    /** Data value - permissive type to match ECharts internal types */
    value?: unknown;
    /** Color of the data item - can be string or gradient object */
    color?: unknown;
    /** Percent value for pie/funnel charts */
    percent?: number;
    /** Marker HTML for tooltip - can be string or RichTextTooltipMarker */
    marker?: unknown;
    /** Axis value in time/value axis */
    axisValue?: string | number;
    /** Axis value as string label */
    axisValueLabel?: string;
}

/**
 * Tooltip formatter params - can be single item or array depending on trigger type
 * trigger: 'item' returns single EChartsCallbackDataParams
 * trigger: 'axis' returns EChartsCallbackDataParams[]
 */
export type EChartsTooltipFormatterParams = EChartsCallbackDataParams | EChartsCallbackDataParams[];

/**
 * ItemStyle color callback params
 * Note: seriesIndex is optional to match ECharts CallbackDataParams
 */
export interface EChartsItemStyleColorParams {
    /** Data index in current series */
    dataIndex: number;
    /** Data value - permissive type to match ECharts internal types */
    value: unknown;
    /** Series index - optional to match ECharts internal types */
    seriesIndex?: number;
}

/**
 * Label formatter callback params
 */
export interface EChartsLabelFormatterParams {
    /** Data name */
    name?: string;
    /** Data index */
    dataIndex: number;
    /** Data value - permissive type to match ECharts internal types */
    value: unknown;
    /** Percent for pie charts */
    percent?: number;
    /** Color of the data item - can be string or gradient object */
    color?: unknown;
}

/**
 * Axis configuration extension type
 * Allows extending base axis styles with additional properties
 */
export interface AxisConfigExtension {
    name?: string;
    nameLocation?: 'start' | 'middle' | 'center' | 'end';
    nameGap?: number;
    nameTextStyle?: Record<string, unknown>;
    min?: number | string;
    max?: number | string;
    rotate?: number;
    interval?: number;
    /** Whether to reverse the axis direction */
    inverse?: boolean;
    /** Axis position: 'top', 'bottom', 'left', 'right' */
    position?: 'top' | 'bottom' | 'left' | 'right';
    /** Whether to leave a gap between axis and data - use boolean for category, [n,n] for value axis */
    boundaryGap?: boolean;
    /** Font size for axis labels at root level */
    fontSize?: number;
    axisLabel?: {
        formatter?: string | ((value: string | number) => string);
        color?: string;
        fontSize?: number;
        rotate?: number;
        interval?: number;
    };
    axisLine?: { show?: boolean; lineStyle?: Record<string, unknown> };
    axisTick?: { show?: boolean };
    splitLine?: { show?: boolean; lineStyle?: Record<string, unknown> };
}

/**
 * DataZoom slider configuration
 */
export interface DataZoomSliderConfig {
    type: 'slider';
    xAxisIndex: number;
    start: number;
    end: number;
    height: number;
    bottom: number;
    borderColor: string;
    backgroundColor: string;
    fillerColor: string;
    handleStyle: { color: string; borderColor: string };
    textStyle: { color: string; fontSize: number };
    dataBackground: {
        lineStyle: { color: string; opacity: number };
        areaStyle: { color: string; opacity: number };
    };
    selectedDataBackground: {
        lineStyle: { color: string; opacity: number };
        areaStyle: { color: string; opacity: number };
    };
}

/**
 * DataZoom inside configuration
 */
export interface DataZoomInsideConfig {
    type: 'inside';
    xAxisIndex: number;
    start: number;
    end: number;
    zoomOnMouseWheel: boolean;
    moveOnMouseMove: boolean;
    preventDefaultMouseMove: boolean;
}

/**
 * DataZoom configuration union type
 */
export type DataZoomConfig = DataZoomSliderConfig | DataZoomInsideConfig;

/**
 * Navigator interface extension for device memory detection
 * deviceMemory is part of the Device Memory API (Chrome/Edge)
 * @see https://developer.mozilla.org/en-US/docs/Web/API/Navigator/deviceMemory
 */
export interface NavigatorWithDeviceMemory extends Navigator {
    deviceMemory?: number;
}

/**
 * ECharts option type for getOption() return value
 * This is a partial type that includes common chart option properties
 */
export interface EChartsChartOption {
    title?: Array<{
        text?: string;
        subtext?: string;
        left?: string | number;
        top?: string | number;
        textStyle?: Record<string, unknown>;
    }> | {
        text?: string;
        subtext?: string;
        left?: string | number;
        top?: string | number;
        textStyle?: Record<string, unknown>;
    };
    xAxis?: Array<{
        type?: string;
        data?: string[];
        name?: string;
    }>;
    yAxis?: Array<{
        type?: string;
        data?: string[];
        name?: string;
    }>;
    series?: EChartsSeriesOption[];
    legend?: unknown;
    tooltip?: unknown;
    grid?: unknown;
    dataZoom?: unknown[];
}

/**
 * ECharts series option type for chart data extraction
 */
export interface EChartsSeriesOption {
    type?: 'bar' | 'line' | 'pie' | 'scatter' | 'heatmap' | 'boxplot' | 'gauge' | 'radar' | 'tree' | 'treemap' | 'sunburst' | 'funnel' | 'graph' | 'sankey' | 'parallel' | 'map' | 'candlestick' | 'custom';
    name?: string;
    data?: Array<number | string | { name?: string; value?: number | number[] | null } | null>;
}

/**
 * Base interface for chart renderers
 * Used by ChartManager to store renderer instances
 * Note: Uses 'any' to allow dynamic method access on renderer classes
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type ChartRenderer = any;

/**
 * Type for chart renderer constructor
 * Note: Uses 'any' to allow any renderer class to be used
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type ChartRendererConstructor = new (config: ChartRendererConfig) => any;
