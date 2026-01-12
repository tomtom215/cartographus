// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * ComparativeInsightsManager - Displays period comparison on Overview page
 *
 * Features:
 * - Fetches comparative analytics data
 * - Displays current vs previous period metrics
 * - Shows percentage change with visual indicators
 * - Period selector (week, month, quarter, year)
 * - Key insights summary
 */

import type { API, LocationFilter } from '../lib/api';
import type { ComparativeAnalyticsResponse, ComparativeMetrics } from '../lib/types/analytics';
import { createLogger } from '../lib/logger';

const logger = createLogger('ComparativeInsightsManager');

type ComparisonPeriod = 'week' | 'month' | 'quarter' | 'year';

export class ComparativeInsightsManager {
  private api: API;
  private isLoading: boolean = false;
  private currentPeriod: ComparisonPeriod = 'week';
  private periodSelect: HTMLSelectElement | null = null;

  constructor(api: API) {
    this.api = api;
  }

  /**
   * Initialize the comparative insights manager
   */
  init(): void {
    this.setupPeriodSelector();
    logger.log('ComparativeInsightsManager initialized');
  }

  /**
   * Set up period selector event handler
   */
  private setupPeriodSelector(): void {
    this.periodSelect = document.getElementById('comparison-period-select') as HTMLSelectElement;
    if (this.periodSelect) {
      this.periodSelect.addEventListener('change', (e) => {
        const target = e.target as HTMLSelectElement;
        this.currentPeriod = target.value as ComparisonPeriod;
        this.loadComparison();
      });
    }
  }

  /**
   * Load comparative insights data
   */
  async loadComparison(filter: LocationFilter = {}): Promise<void> {
    if (this.isLoading) return;

    this.isLoading = true;
    this.showLoading();

    try {
      const data = await this.api.getAnalyticsComparative(filter, this.currentPeriod);
      this.updateMetrics(data);
      this.updateInsights(data);
    } catch (error) {
      logger.error('Failed to load comparative insights:', error);
      this.showError();
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * Show loading state
   */
  private showLoading(): void {
    const elements = [
      'comp-playbacks-value', 'comp-users-value',
      'comp-watchtime-value', 'comp-completion-value'
    ];
    elements.forEach(id => {
      const el = document.getElementById(id);
      if (el) el.textContent = '...';
    });

    // Clear change indicators
    const changeElements = [
      'comp-playbacks-change', 'comp-users-change',
      'comp-watchtime-change', 'comp-completion-change'
    ];
    changeElements.forEach(id => {
      const el = document.getElementById(id);
      if (el) {
        el.className = 'comparison-metric-change';
        const arrow = el.querySelector('.change-arrow');
        const value = el.querySelector('.change-value');
        if (arrow) arrow.textContent = '';
        if (value) value.textContent = '';
      }
    });
  }

  /**
   * Show error state
   */
  private showError(): void {
    const elements = [
      'comp-playbacks-value', 'comp-users-value',
      'comp-watchtime-value', 'comp-completion-value'
    ];
    elements.forEach(id => {
      const el = document.getElementById(id);
      if (el) el.textContent = '--';
    });
  }

  /**
   * Update metric cards with comparison data
   */
  private updateMetrics(data: ComparativeAnalyticsResponse): void {
    if (!data || !data.metrics_comparison) {
      this.showError();
      return;
    }

    // Map API metric names to our UI IDs
    const metricMap: Record<string, { valueId: string; changeId: string }> = {
      'playback_count': { valueId: 'comp-playbacks-value', changeId: 'comp-playbacks-change' },
      'unique_users': { valueId: 'comp-users-value', changeId: 'comp-users-change' },
      'watch_time_minutes': { valueId: 'comp-watchtime-value', changeId: 'comp-watchtime-change' },
      'avg_completion': { valueId: 'comp-completion-value', changeId: 'comp-completion-change' }
    };

    // Update each metric from the comparison data
    data.metrics_comparison.forEach((metric: ComparativeMetrics) => {
      const mapping = metricMap[metric.metric];
      if (!mapping) return;

      const valueEl = document.getElementById(mapping.valueId);
      const changeEl = document.getElementById(mapping.changeId);

      if (valueEl) {
        valueEl.textContent = this.formatValue(metric.current_value, metric.metric);
      }

      if (changeEl) {
        this.updateChangeIndicator(changeEl, metric);
      }
    });

    // Also try to update from period data directly if metrics_comparison is empty
    if (data.metrics_comparison.length === 0 && data.current_period) {
      this.updateFromPeriodData(data);
    }
  }

  /**
   * Update from period data when metrics_comparison is empty
   */
  private updateFromPeriodData(data: ComparativeAnalyticsResponse): void {
    const current = data.current_period;
    const previous = data.previous_period;

    if (!current || !previous) return;

    // Playbacks
    this.updateSingleMetric(
      'comp-playbacks-value',
      'comp-playbacks-change',
      current.playback_count,
      previous.playback_count,
      'playback_count'
    );

    // Users
    this.updateSingleMetric(
      'comp-users-value',
      'comp-users-change',
      current.unique_users,
      previous.unique_users,
      'unique_users'
    );

    // Watch time
    this.updateSingleMetric(
      'comp-watchtime-value',
      'comp-watchtime-change',
      current.watch_time_minutes,
      previous.watch_time_minutes,
      'watch_time_minutes'
    );

    // Completion
    this.updateSingleMetric(
      'comp-completion-value',
      'comp-completion-change',
      current.avg_completion,
      previous.avg_completion,
      'avg_completion'
    );
  }

  /**
   * Update a single metric from period data
   */
  private updateSingleMetric(
    valueId: string,
    changeId: string,
    currentValue: number,
    previousValue: number,
    metricType: string
  ): void {
    const valueEl = document.getElementById(valueId);
    const changeEl = document.getElementById(changeId);

    if (valueEl) {
      valueEl.textContent = this.formatValue(currentValue, metricType);
    }

    if (changeEl) {
      const percentChange = previousValue > 0
        ? ((currentValue - previousValue) / previousValue) * 100
        : (currentValue > 0 ? 100 : 0);

      const metric: ComparativeMetrics = {
        metric: metricType,
        current_value: currentValue,
        previous_value: previousValue,
        absolute_change: currentValue - previousValue,
        percentage_change: percentChange,
        growth_direction: percentChange > 0 ? 'up' : (percentChange < 0 ? 'down' : 'stable'),
        is_improvement: percentChange > 0
      };

      this.updateChangeIndicator(changeEl, metric);
    }
  }

  /**
   * Update the change indicator element
   */
  private updateChangeIndicator(changeEl: HTMLElement, metric: ComparativeMetrics): void {
    const arrow = changeEl.querySelector('.change-arrow');
    const value = changeEl.querySelector('.change-value');

    const percentChange = metric.percentage_change;
    const isPositive = percentChange > 0;
    const isNegative = percentChange < 0;

    // Update CSS class for color
    changeEl.classList.remove('positive', 'negative', 'neutral');
    if (isPositive) {
      changeEl.classList.add('positive');
    } else if (isNegative) {
      changeEl.classList.add('negative');
    } else {
      changeEl.classList.add('neutral');
    }

    // Update arrow
    if (arrow) {
      if (isPositive) {
        arrow.textContent = '\u25B2'; // Up arrow
      } else if (isNegative) {
        arrow.textContent = '\u25BC'; // Down arrow
      } else {
        arrow.textContent = '\u25AC'; // Horizontal bar
      }
    }

    // Update value
    if (value) {
      const absPercent = Math.abs(percentChange);
      if (absPercent >= 1000) {
        value.textContent = `${(absPercent / 1000).toFixed(1)}K%`;
      } else {
        value.textContent = `${absPercent.toFixed(1)}%`;
      }
    }
  }

  /**
   * Format value based on metric type
   */
  private formatValue(value: number, metricType: string): string {
    if (value === null || value === undefined || isNaN(value)) {
      return '--';
    }

    switch (metricType) {
      case 'watch_time_minutes':
        // Convert to hours
        const hours = value / 60;
        if (hours >= 1000) {
          return `${(hours / 1000).toFixed(1)}K hrs`;
        } else if (hours >= 1) {
          return `${Math.round(hours)} hrs`;
        } else {
          return `${Math.round(value)} min`;
        }

      case 'avg_completion':
        return `${Math.round(value)}%`;

      case 'playback_count':
      case 'unique_users':
      default:
        if (value >= 1000000) {
          return `${(value / 1000000).toFixed(1)}M`;
        } else if (value >= 1000) {
          return `${(value / 1000).toFixed(1)}K`;
        }
        return value.toString();
    }
  }

  /**
   * Update the key insights section
   */
  private updateInsights(data: ComparativeAnalyticsResponse): void {
    const insightsEl = document.getElementById('period-comparison-insights');
    if (!insightsEl) return;

    // Clear existing insights
    insightsEl.innerHTML = '';

    // Add overall trend
    if (data.overall_trend) {
      const trendEl = document.createElement('div');
      trendEl.className = 'comparison-trend';
      trendEl.innerHTML = `
        <span class="trend-icon" aria-hidden="true">${this.getTrendIcon(data.overall_trend)}</span>
        <span class="trend-text">${this.formatTrendText(data.overall_trend)}</span>
      `;
      insightsEl.appendChild(trendEl);
    }

    // Add key insights
    if (data.key_insights && data.key_insights.length > 0) {
      const insightsList = document.createElement('ul');
      insightsList.className = 'comparison-insights-list';
      insightsList.setAttribute('role', 'list');

      data.key_insights.slice(0, 3).forEach(insight => {
        const li = document.createElement('li');
        li.className = 'comparison-insight-item';
        li.textContent = insight;
        insightsList.appendChild(li);
      });

      insightsEl.appendChild(insightsList);
    }
  }

  /**
   * Get icon for overall trend
   */
  private getTrendIcon(trend: string): string {
    const trendLower = trend.toLowerCase();
    if (trendLower.includes('up') || trendLower.includes('growth') || trendLower.includes('increase')) {
      return '\u2191'; // Up arrow
    } else if (trendLower.includes('down') || trendLower.includes('decline') || trendLower.includes('decrease')) {
      return '\u2193'; // Down arrow
    }
    return '\u2192'; // Right arrow (stable)
  }

  /**
   * Format trend text for display
   */
  private formatTrendText(trend: string): string {
    // Capitalize first letter
    return trend.charAt(0).toUpperCase() + trend.slice(1);
  }

  /**
   * Get the current comparison period
   */
  getCurrentPeriod(): ComparisonPeriod {
    return this.currentPeriod;
  }

  /**
   * Set the comparison period and reload
   */
  setPeriod(period: ComparisonPeriod): void {
    this.currentPeriod = period;
    if (this.periodSelect) {
      this.periodSelect.value = period;
    }
    this.loadComparison();
  }

  /**
   * Update display with provided comparison data
   * Used by CustomDateComparisonManager to update UI with custom date results
   */
  updateWithData(data: ComparativeAnalyticsResponse): void {
    this.updateMetrics(data);
    this.updateInsights(data);
  }

  /**
   * Cleanup resources to prevent memory leaks
   */
  destroy(): void {
    this.periodSelect = null;
  }
}

export default ComparativeInsightsManager;
