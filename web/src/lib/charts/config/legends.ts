// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Legend configurations for charts
 *
 * Accessibility - WCAG SC 1.4.1 Use of Color
 * Legends use different shapes to convey information beyond color alone.
 */

import type { LegendComponentOption } from 'echarts';
import { STYLE_CONSTANTS } from './colors';

/**
 * Accessible legend icon shapes
 * Each series gets a distinct shape to distinguish it beyond color alone
 * Order is designed for visual distinctiveness
 */
const ACCESSIBLE_ICONS = [
  'circle',       // Round - most common
  'rect',         // Square - clearly different from circle
  'triangle',     // Three-sided - very distinct
  'diamond',      // Rotated square - different orientation
  'roundRect',    // Rounded rectangle - subtle distinction
  'pin',          // Map pin shape - very distinct
  'arrow',        // Arrow shape - unique direction
];

/**
 * Get an accessible icon shape based on series index
 * Returns different shapes to distinguish series beyond color alone
 */
export function getAccessibleIcon(index: number): string {
  return ACCESSIBLE_ICONS[index % ACCESSIBLE_ICONS.length];
}

/**
 * Get the full array of accessible icons for formatter customization
 */
export function getAccessibleIcons(): readonly string[] {
  return ACCESSIBLE_ICONS;
}

export function getEnhancedLegend(
  config?: Partial<LegendComponentOption> & { useAccessibleIcons?: boolean }
): LegendComponentOption {
  const { useAccessibleIcons, ...restConfig } = config || {};

  return {
    textStyle: {
      color: STYLE_CONSTANTS.axisLabelColor,
      fontSize: 12,
    },
    itemGap: 16,
    itemWidth: 20,
    itemHeight: 12,
    // Use roundRect as default, but charts can set useAccessibleIcons
    // for multi-series to get different shapes per series
    icon: 'roundRect',
    inactiveColor: STYLE_CONSTANTS.inactiveColor,
    ...restConfig,
  };
}

/**
 * Get legend with accessible icons for multi-series charts
 * Uses different shapes for each series to distinguish beyond color alone
 *
 * @param _seriesCount - Number of series (unused, kept for API compatibility)
 * @param config - Additional legend configuration
 */
export function getAccessibleLegend(
  _seriesCount: number,
  config?: Partial<LegendComponentOption>
): LegendComponentOption {
  const base = getEnhancedLegend(config);

  // If we have the legend data with names, add icons
  if (config?.data && Array.isArray(config.data)) {
    return {
      ...base,
      data: config.data.map((item, index) => {
        // Handle both string and object legend data items
        const name = typeof item === 'string' ? item : (item as { name: string }).name;
        return {
          name,
          icon: getAccessibleIcon(index),
        };
      }),
    };
  }

  return base;
}
