// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Color palettes and style constants for charts
 *
 * v2.0 - Updated with new accessible color palette
 * Reference: UI/UX Audit
 * @see /docs/working/UI_UX_AUDIT.md
 *
 * Colors now use the redesigned palette:
 * - Primary: Violet (#7c3aed) - softer, more modern accent
 * - Secondary: Blue (#3b82f6) - harmonious complement
 * - Full 10-color harmonious palette for charts
 * - Colorblind-safe alternatives available via CSS
 */

/**
 * Get computed CSS color variable value
 * Falls back to provided default if CSS variable not available
 */
function getCSSColor(varName: string, fallback: string): string {
  if (typeof window === 'undefined' || typeof getComputedStyle === 'undefined') {
    return fallback;
  }
  const value = getComputedStyle(document.documentElement).getPropertyValue(varName).trim();
  return value || fallback;
}

/**
 * Chart colors - new softer violet-based palette
 * These are the default values; actual values come from CSS variables
 */
export const CHART_COLORS = {
  // Primary accent colors
  primary: '#7c3aed',       // Violet - main accent (was #e94560)
  secondary: '#3b82f6',     // Blue - secondary accent
  accent: '#8b5cf6',        // Purple
  // Semantic colors
  success: '#10b981',       // Emerald green
  warning: '#f59e0b',       // Amber
  danger: '#ef4444',        // Red (for errors only)
  info: '#06b6d4',          // Cyan
  // Additional palette colors
  purple: '#8b5cf6',
  orange: '#f97316',
  teal: '#14b8a6',
  pink: '#ec4899',
  indigo: '#6366f1',
  // Background colors
  dark: '#16213e',
  darker: '#1a1a2e',
  // Backwards compatibility aliases
  crimson: '#ef4444',       // Alias for danger (backwards compatibility)
} as const;

/**
 * Dynamic chart colors interface for theme support
 */
interface DynamicChartColors {
  primary: string;
  secondary: string;
  accent: string;
  success: string;
  warning: string;
  danger: string;
  info: string;
  purple: string;
  orange: string;
  teal: string;
  pink: string;
  indigo: string;
  dark: string;
  darker: string;
  crimson: string;  // Alias for danger (backwards compatibility)
}

/**
 * Get chart colors from CSS variables (for theme/colorblind mode support)
 * This function reads current CSS custom properties for dynamic theming
 */
export function getThemeColors(): DynamicChartColors {
  const danger = getCSSColor('--color-error', CHART_COLORS.danger);
  return {
    primary: getCSSColor('--chart-primary', CHART_COLORS.primary),
    secondary: getCSSColor('--chart-color-2', CHART_COLORS.secondary),
    accent: getCSSColor('--chart-color-7', CHART_COLORS.accent),
    success: getCSSColor('--color-success', CHART_COLORS.success),
    warning: getCSSColor('--color-warning', CHART_COLORS.warning),
    danger: danger,
    info: getCSSColor('--color-info', CHART_COLORS.info),
    purple: getCSSColor('--chart-color-7', CHART_COLORS.purple),
    orange: getCSSColor('--chart-color-9', CHART_COLORS.orange),
    teal: getCSSColor('--chart-color-8', CHART_COLORS.teal),
    pink: getCSSColor('--chart-color-5', CHART_COLORS.pink),
    indigo: getCSSColor('--chart-color-10', CHART_COLORS.indigo),
    dark: getCSSColor('--secondary-bg', CHART_COLORS.dark),
    darker: getCSSColor('--primary-bg', CHART_COLORS.darker),
    crimson: danger,  // Alias for backwards compatibility
  };
}

/**
 * Get the 10-color chart palette from CSS variables
 * Supports theme switching and colorblind mode
 */
export function getChartPalette(): string[] {
  return [
    getCSSColor('--chart-color-1', '#7c3aed'),
    getCSSColor('--chart-color-2', '#3b82f6'),
    getCSSColor('--chart-color-3', '#10b981'),
    getCSSColor('--chart-color-4', '#f59e0b'),
    getCSSColor('--chart-color-5', '#ec4899'),
    getCSSColor('--chart-color-6', '#06b6d4'),
    getCSSColor('--chart-color-7', '#8b5cf6'),
    getCSSColor('--chart-color-8', '#14b8a6'),
    getCSSColor('--chart-color-9', '#f97316'),
    getCSSColor('--chart-color-10', '#6366f1'),
  ];
}

/**
 * Get heatmap colors from CSS variables
 */
export function getHeatmapColors(): string[] {
  return [
    getCSSColor('--heatmap-1', '#1e3a5f'),
    getCSSColor('--heatmap-2', '#2d4a6f'),
    getCSSColor('--heatmap-3', '#3b5998'),
    getCSSColor('--heatmap-4', '#7c3aed'),
    getCSSColor('--heatmap-5', '#a855f7'),
    getCSSColor('--heatmap-6', '#ec4899'),
    getCSSColor('--heatmap-7', '#f43f5e'),
  ];
}

export const COLOR_PALETTES = {
  // Default palette - now uses the new 10-color harmony
  default: [
    CHART_COLORS.primary,
    CHART_COLORS.secondary,
    CHART_COLORS.success,
    CHART_COLORS.warning,
    CHART_COLORS.pink,
    CHART_COLORS.info,
    CHART_COLORS.purple,
    CHART_COLORS.teal,
    CHART_COLORS.orange,
    CHART_COLORS.indigo,
  ],
  gradient: {
    // Updated gradients using new palette
    violetToBlue: [
      { offset: 0, color: CHART_COLORS.primary },
      { offset: 1, color: CHART_COLORS.secondary },
    ],
    blueToViolet: [
      { offset: 0, color: CHART_COLORS.secondary },
      { offset: 1, color: CHART_COLORS.primary },
    ],
    violetToPurple: [
      { offset: 0, color: CHART_COLORS.primary },
      { offset: 1, color: CHART_COLORS.accent },
    ],
    greenToBlue: [
      { offset: 0, color: CHART_COLORS.success },
      { offset: 1, color: CHART_COLORS.secondary },
    ],
    // Legacy aliases for backwards compatibility
    redToBlue: [
      { offset: 0, color: CHART_COLORS.primary },
      { offset: 1, color: CHART_COLORS.secondary },
    ],
    blueToRed: [
      { offset: 0, color: CHART_COLORS.secondary },
      { offset: 1, color: CHART_COLORS.primary },
    ],
    redToPurple: [
      { offset: 0, color: CHART_COLORS.primary },
      { offset: 1, color: CHART_COLORS.accent },
    ],
  },
  // Heatmap now uses 7 colors for better granularity
  heatmap: [
    '#1e3a5f',
    '#2d4a6f',
    '#3b5998',
    '#7c3aed',
    '#a855f7',
    '#ec4899',
    '#f43f5e',
  ],
  // Completion uses semantic colors
  completion: [
    CHART_COLORS.danger,     // Not started (red)
    CHART_COLORS.orange,     // < 25%
    CHART_COLORS.warning,    // 25-50%
    CHART_COLORS.info,       // 50-75%
    CHART_COLORS.success,    // > 75% (green)
  ],
} as const;

export const STYLE_CONSTANTS = {
  backgroundColor: 'transparent',
  borderColor: '#1a1a2e',
  axisLineColor: '#2a2a3e',
  axisLabelColor: '#a0a0a0',
  splitLineColor: '#2a2a3e',
  textColor: '#eaeaea',
  inactiveColor: '#3a3a4e',
} as const;

/**
 * Dynamic style constants interface for theme support
 */
interface DynamicStyleConstants {
  backgroundColor: string;
  borderColor: string;
  axisLineColor: string;
  axisLabelColor: string;
  splitLineColor: string;
  textColor: string;
  inactiveColor: string;
}

/**
 * Get style constants based on current theme
 */
export function getThemedStyleConstants(): DynamicStyleConstants {
  return {
    backgroundColor: 'transparent',
    borderColor: getCSSColor('--primary-bg', STYLE_CONSTANTS.borderColor),
    axisLineColor: getCSSColor('--border', STYLE_CONSTANTS.axisLineColor),
    axisLabelColor: getCSSColor('--text-secondary', STYLE_CONSTANTS.axisLabelColor),
    splitLineColor: getCSSColor('--border', STYLE_CONSTANTS.splitLineColor),
    textColor: getCSSColor('--text-primary', STYLE_CONSTANTS.textColor),
    inactiveColor: getCSSColor('--border', STYLE_CONSTANTS.inactiveColor),
  };
}
