// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Common chart utility functions
 */

import * as echarts from 'echarts';

export class ChartHelpers {
  static createLinearGradient(
    direction: 'horizontal' | 'vertical',
    colors: readonly { offset: number; color: string }[]
  ): echarts.graphic.LinearGradient {
    if (direction === 'horizontal') {
      return new echarts.graphic.LinearGradient(1, 0, 0, 0, [...colors]);
    }
    return new echarts.graphic.LinearGradient(0, 0, 0, 1, [...colors]);
  }

  static getAreaStyleGradient(
    color: string,
    opacity: { start: number; end: number } = { start: 0.5, end: 0.1 }
  ): { color: echarts.graphic.LinearGradient } {
    return {
      color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
        { offset: 0, color: color.replace(')', `, ${opacity.start})`).replace('rgb', 'rgba') },
        { offset: 1, color: color.replace(')', `, ${opacity.end})`).replace('rgb', 'rgba') },
      ]),
    };
  }

  static shouldEnableSampling(dataPoints: number): boolean {
    return dataPoints > 1000;
  }

  static getBarBorderRadius(position: 'horizontal' | 'vertical'): number[] {
    return position === 'horizontal' ? [0, 4, 4, 0] : [4, 4, 0, 0];
  }

  static truncateText(text: string, maxLength: number): string {
    return text.length > maxLength ? text.substring(0, maxLength - 3) + '...' : text;
  }

  static calculatePercentage(value: number, total: number): number {
    return total > 0 ? (value / total) * 100 : 0;
  }

  static reverseArrays(...arrays: readonly (readonly any[])[]): any[][] {
    return arrays.map(arr => [...arr].reverse());
  }
}
