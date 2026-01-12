// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Grid and layout configurations for charts
 */

import type { GridComponentOption } from 'echarts';

export const GRID_PRESETS = {
  default: {
    left: '3%',
    right: '4%',
    bottom: '3%',
    containLabel: true,
  },
  /**
   * Grid preset for charts with dataZoom slider
   * Provides extra bottom space for the zoom slider control
   */
  withDataZoom: {
    left: '3%',
    right: '4%',
    bottom: '15%',
    containLabel: true,
  },
  withTitle: {
    left: '10%',
    right: '10%',
    bottom: '10%',
    top: '20%',
  },
  compact: {
    left: '10%',
    right: '10%',
    bottom: '10%',
    top: '10%',
  },
  heatmap: {
    left: '10%',
    right: '10%',
    top: '10%',
    bottom: '10%',
    containLabel: true,
  },
} as const;

export function getGridConfig(preset: keyof typeof GRID_PRESETS): GridComponentOption {
  return { ...GRID_PRESETS[preset] };
}
