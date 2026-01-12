// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Renderer for user activity charts
 */

import type { UsersResponse } from '../../api';
import { BaseChartRenderer } from './BaseChartRenderer';
import { CHART_COLORS, STYLE_CONSTANTS } from '../config/colors';
import { getGridConfig } from '../config/grid';
import type { EChartsCallbackDataParams } from '../types';

export class UserChartRenderer extends BaseChartRenderer {
  render(data: UsersResponse): void {
    if (!this.hasData(data?.top_users)) {
      this.showEmptyState('No user data available');
      return;
    }
    const users = data.top_users.map(u => u.username);
    const counts = data.top_users.map(u => u.playback_count);
    const total = counts.reduce((sum, count) => sum + count, 0);

    const [reversedUsers, reversedCounts] = this.helpers.reverseArrays(users, counts);

    const option = {
      ...this.getBaseOption(),
      tooltip: this.getTooltip({
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: this.createTooltipFormatter(total),
      }),
      grid: getGridConfig('default'),
      xAxis: this.getValueAxis(),
      yAxis: this.getCategoryAxis(reversedUsers),
      series: [
        {
          type: 'bar' as const,
          data: reversedCounts,
          itemStyle: {
            color: this.helpers.createLinearGradient('horizontal', [
              { offset: 0, color: CHART_COLORS.primary },
              { offset: 1, color: CHART_COLORS.accent },
            ]),
            borderRadius: this.helpers.getBarBorderRadius('horizontal'),
          },
          label: {
            show: true,
            position: 'right' as const,
            color: STYLE_CONSTANTS.textColor,
          },
        },
      ],
    };

    this.setOption(option);
  }

  private createTooltipFormatter(total: number) {
    return (params: EChartsCallbackDataParams | EChartsCallbackDataParams[]) => {
      if (!Array.isArray(params) || params.length === 0) return '';
      const param = params[0];
      const paramValue = typeof param.value === 'number' ? param.value : 0;
      const value = this.formatters.formatNumber(paramValue);
      const percentage = this.formatters.formatPercent(
        this.helpers.calculatePercentage(paramValue, total)
      );
      return `<div style="font-weight: 600; margin-bottom: 8px;">${param.name}</div>
        <div style="margin: 4px 0;">
          <span style="font-weight: 500;">Playbacks:</span>
          <span style="color: ${CHART_COLORS.primary}; font-weight: 600;">${value}</span>
          <span style="color: #a0a0a0; margin-left: 8px;">(${percentage})</span>
        </div>`;
    };
  }
}
