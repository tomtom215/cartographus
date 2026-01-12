// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Utility functions for formatting numbers, percentages, dates, etc.
 */

export class ChartFormatters {
  static formatNumber(num: number): string {
    return num.toLocaleString();
  }

  static formatPercent(num: number): string {
    return `${num.toFixed(1)}%`;
  }

  static formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  }

  static formatMinutesToHours(minutes: number): string {
    const hours = Math.round(minutes / 60);
    return `${this.formatNumber(hours)}h`;
  }

  static formatDuration(minutes: number): string {
    if (minutes < 60) return `${Math.round(minutes)}m`;
    const hours = Math.floor(minutes / 60);
    const mins = Math.round(minutes % 60);
    return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`;
  }

  static formatDate(date: string | Date): string {
    return new Date(date).toLocaleDateString();
  }

  static formatDateTime(date: string | Date): string {
    return new Date(date).toLocaleString();
  }
}
