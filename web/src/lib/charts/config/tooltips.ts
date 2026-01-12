// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Tooltip configurations for charts
 */

import type { TooltipComponentOption } from 'echarts';
import { CHART_COLORS, STYLE_CONSTANTS } from './colors';

export function getEnhancedTooltip(
  config?: Partial<TooltipComponentOption>
): TooltipComponentOption {
  return {
    trigger: 'axis',
    backgroundColor: 'rgba(22, 33, 62, 0.98)',
    borderColor: CHART_COLORS.primary,
    borderWidth: 1,
    padding: [12, 16],
    textStyle: {
      color: STYLE_CONSTANTS.textColor,
      fontSize: 13,
    },
    extraCssText: 'box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4); border-radius: 6px;',
    ...config,
  };
}
