// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * TrendPredictionManager - Trend Prediction/Forecasting
 *
 * Adds predictive trend lines to time-series charts using linear regression
 * and moving averages to forecast future values.
 *
 * Features:
 * - Linear regression for trend projection
 * - Configurable prediction period (7, 14, 30 days)
 * - Visual differentiation of predicted vs actual data
 * - Toggle to enable/disable predictions
 * - Confidence interval display
 */

import { CHART_COLORS } from '../lib/charts/config/colors';
import { SafeStorage } from '../lib/utils/SafeStorage';

/**
 * Prediction configuration
 */
export interface PredictionConfig {
  /** Number of days to predict into the future */
  predictionDays: number;
  /** Whether to show confidence intervals */
  showConfidence: boolean;
  /** Confidence interval percentage (0.0 - 1.0) */
  confidenceLevel: number;
}

/**
 * Result of trend prediction calculation
 */
export interface PredictionResult {
  /** Predicted values for future dates */
  predictions: number[];
  /** Future dates for predictions */
  dates: string[];
  /** Upper confidence bound */
  upperBound: number[];
  /** Lower confidence bound */
  lowerBound: number[];
  /** Slope of the trend line (positive = increasing) */
  slope: number;
  /** R-squared value indicating fit quality */
  rSquared: number;
  /** Trend direction: 'up', 'down', or 'stable' */
  trend: 'up' | 'down' | 'stable';
  /** Percentage change per day */
  dailyChange: number;
}

/**
 * Linear regression result
 */
interface RegressionResult {
  slope: number;
  intercept: number;
  rSquared: number;
  standardError: number;
}

/**
 * Default prediction configuration
 */
const DEFAULT_CONFIG: PredictionConfig = {
  predictionDays: 7,
  showConfidence: true,
  confidenceLevel: 0.95,
};

export class TrendPredictionManager {
  private config: PredictionConfig;
  private enabled: boolean = false;

  constructor(config: Partial<PredictionConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
    this.loadEnabledState();
  }

  /**
   * Load enabled state from localStorage
   */
  private loadEnabledState(): void {
    const stored = SafeStorage.getItem('trendPrediction.enabled');
    this.enabled = stored === 'true';
  }

  /**
   * Save enabled state to localStorage
   */
  private saveEnabledState(): void {
    SafeStorage.setItem('trendPrediction.enabled', String(this.enabled));
  }

  /**
   * Toggle prediction display
   */
  toggle(): boolean {
    this.enabled = !this.enabled;
    this.saveEnabledState();
    return this.enabled;
  }

  /**
   * Check if predictions are enabled
   */
  isEnabled(): boolean {
    return this.enabled;
  }

  /**
   * Set prediction days
   */
  setPredictionDays(days: number): void {
    this.config.predictionDays = Math.max(1, Math.min(30, days));
  }

  /**
   * Get prediction days
   */
  getPredictionDays(): number {
    return this.config.predictionDays;
  }

  /**
   * Calculate trend prediction using linear regression
   *
   * @param values - Historical data values
   * @param dates - Historical dates (YYYY-MM-DD format)
   * @returns Prediction result with forecasted values
   */
  predict(values: number[], dates: string[]): PredictionResult | null {
    // Need at least 7 data points for meaningful prediction
    if (values.length < 7) {
      return null;
    }

    // Perform linear regression
    const regression = this.linearRegression(values);

    // Generate future dates
    const futureDates = this.generateFutureDates(
      dates[dates.length - 1],
      this.config.predictionDays
    );

    // Calculate predictions
    const startIndex = values.length;
    const predictions: number[] = [];
    const upperBound: number[] = [];
    const lowerBound: number[] = [];

    // t-value for confidence interval (approximate for large samples)
    const tValue = this.config.confidenceLevel === 0.95 ? 1.96 : 1.645;

    for (let i = 0; i < this.config.predictionDays; i++) {
      const x = startIndex + i;
      const predicted = regression.slope * x + regression.intercept;
      const marginOfError = tValue * regression.standardError * Math.sqrt(1 + 1 / values.length + Math.pow(x - (values.length / 2), 2) / this.sumSquaredDeviations(values.length));

      predictions.push(Math.max(0, predicted)); // Ensure non-negative
      upperBound.push(Math.max(0, predicted + marginOfError));
      lowerBound.push(Math.max(0, predicted - marginOfError));
    }

    // Determine trend direction
    const dailyChange = regression.slope / (this.average(values) || 1) * 100;
    let trend: 'up' | 'down' | 'stable';
    if (dailyChange > 1) {
      trend = 'up';
    } else if (dailyChange < -1) {
      trend = 'down';
    } else {
      trend = 'stable';
    }

    return {
      predictions,
      dates: futureDates,
      upperBound,
      lowerBound,
      slope: regression.slope,
      rSquared: regression.rSquared,
      trend,
      dailyChange,
    };
  }

  /**
   * Perform linear regression on data
   */
  private linearRegression(values: number[]): RegressionResult {
    const n = values.length;
    let sumX = 0;
    let sumY = 0;
    let sumXY = 0;
    let sumX2 = 0;
    let sumY2 = 0;

    for (let i = 0; i < n; i++) {
      sumX += i;
      sumY += values[i];
      sumXY += i * values[i];
      sumX2 += i * i;
      sumY2 += values[i] * values[i];
    }

    const slope = (n * sumXY - sumX * sumY) / (n * sumX2 - sumX * sumX);
    const intercept = (sumY - slope * sumX) / n;

    // Calculate R-squared
    const meanY = sumY / n;
    let ssTotal = 0;
    let ssResidual = 0;

    for (let i = 0; i < n; i++) {
      const predicted = slope * i + intercept;
      ssTotal += Math.pow(values[i] - meanY, 2);
      ssResidual += Math.pow(values[i] - predicted, 2);
    }

    const rSquared = ssTotal > 0 ? 1 - ssResidual / ssTotal : 0;

    // Calculate standard error
    const standardError = Math.sqrt(ssResidual / (n - 2));

    return { slope, intercept, rSquared, standardError };
  }

  /**
   * Calculate sum of squared deviations from mean for x values
   */
  private sumSquaredDeviations(n: number): number {
    const meanX = (n - 1) / 2;
    let sum = 0;
    for (let i = 0; i < n; i++) {
      sum += Math.pow(i - meanX, 2);
    }
    return sum;
  }

  /**
   * Calculate average of values
   */
  private average(values: number[]): number {
    if (values.length === 0) return 0;
    return values.reduce((sum, v) => sum + v, 0) / values.length;
  }

  /**
   * Generate future dates starting from lastDate
   */
  private generateFutureDates(lastDate: string, days: number): string[] {
    const dates: string[] = [];
    const date = new Date(lastDate);

    for (let i = 1; i <= days; i++) {
      const futureDate = new Date(date);
      futureDate.setDate(date.getDate() + i);
      dates.push(futureDate.toISOString().split('T')[0]);
    }

    return dates;
  }

  /**
   * Get ECharts series configuration for prediction line
   */
  getPredictionSeries(
    values: number[],
    dates: string[],
    seriesName: string,
    color: string
  ): any[] {
    if (!this.enabled) {
      return [];
    }

    const prediction = this.predict(values, dates);
    if (!prediction) {
      return [];
    }

    const series: any[] = [];

    // Main prediction line (dashed)
    series.push({
      name: `${seriesName} Forecast`,
      type: 'line' as const,
      smooth: true,
      data: [
        // Start from last actual point for continuity
        ...Array(values.length - 1).fill(null),
        values[values.length - 1],
        ...prediction.predictions,
      ],
      lineStyle: {
        color,
        width: 2,
        type: 'dashed' as const,
      },
      itemStyle: { color },
      symbol: 'diamond',
      symbolSize: 6,
      showSymbol: true,
      emphasis: {
        focus: 'series' as const,
      },
    });

    // Confidence interval (area between upper and lower bounds)
    if (this.config.showConfidence) {
      // Upper bound
      series.push({
        name: `${seriesName} Upper`,
        type: 'line' as const,
        data: [
          ...Array(values.length - 1).fill(null),
          values[values.length - 1],
          ...prediction.upperBound,
        ],
        lineStyle: {
          opacity: 0,
        },
        stack: `confidence-${seriesName}`,
        areaStyle: {
          color: 'transparent',
        },
        symbol: 'none',
        silent: true,
      });

      // Lower bound with fill to upper
      series.push({
        name: `${seriesName} Confidence`,
        type: 'line' as const,
        data: [
          ...Array(values.length - 1).fill(null),
          values[values.length - 1],
          ...prediction.lowerBound,
        ],
        lineStyle: {
          opacity: 0,
        },
        stack: `confidence-${seriesName}`,
        areaStyle: {
          color,
          opacity: 0.1,
        },
        symbol: 'none',
        silent: true,
      });
    }

    return series;
  }

  /**
   * Get extended x-axis data including future dates
   */
  getExtendedDates(dates: string[]): string[] {
    if (!this.enabled || dates.length < 7) {
      return dates;
    }

    const futureDates = this.generateFutureDates(
      dates[dates.length - 1],
      this.config.predictionDays
    );

    return [...dates, ...futureDates];
  }

  /**
   * Get trend indicator for display
   */
  getTrendIndicator(values: number[], dates: string[]): { icon: string; label: string; color: string } | null {
    const prediction = this.predict(values, dates);
    if (!prediction) {
      return null;
    }

    switch (prediction.trend) {
      case 'up':
        return {
          icon: '\u2191', // Up arrow
          label: `+${prediction.dailyChange.toFixed(1)}%/day`,
          color: CHART_COLORS.success || '#10b981',
        };
      case 'down':
        return {
          icon: '\u2193', // Down arrow
          label: `${prediction.dailyChange.toFixed(1)}%/day`,
          color: CHART_COLORS.danger || '#ef4444',
        };
      case 'stable':
      default:
        return {
          icon: '\u2192', // Right arrow
          label: 'Stable',
          color: CHART_COLORS.warning || '#f59e0b',
        };
    }
  }

  /**
   * Create toggle button HTML
   */
  static createToggleButton(): HTMLElement {
    const button = document.createElement('button');
    button.id = 'prediction-toggle';
    button.className = 'prediction-toggle';
    button.setAttribute('aria-pressed', 'false');
    button.setAttribute('aria-label', 'Toggle trend forecast');
    button.title = 'Show/hide trend forecast';
    button.innerHTML = `
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline>
      </svg>
      <span class="prediction-toggle-label">Forecast</span>
    `;
    return button;
  }

  /**
   * Create prediction days selector HTML
   */
  static createDaysSelector(): HTMLElement {
    const container = document.createElement('div');
    container.className = 'prediction-days-selector';
    container.innerHTML = `
      <label for="prediction-days" class="sr-only">Forecast period</label>
      <select id="prediction-days" class="prediction-days-select" aria-label="Forecast period">
        <option value="7">7 days</option>
        <option value="14">14 days</option>
        <option value="30">30 days</option>
      </select>
    `;
    return container;
  }

  /**
   * Cleanup resources to prevent memory leaks
   * Resets configuration to defaults
   */
  destroy(): void {
    this.config = { ...DEFAULT_CONFIG };
    this.enabled = false;
  }
}

// Export singleton instance
export const trendPredictionManager = new TrendPredictionManager();
