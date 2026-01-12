// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * AnomalyDetectionManager - Detects and visualizes unusual patterns
 *
 * Features:
 * - Statistical anomaly detection using Z-score analysis
 * - Identifies unusual spikes and dips in playback activity
 * - Visualizes anomalies on the Overview page
 * - Provides actionable insights for detected anomalies
 */

import type { API, LocationFilter } from '../lib/api';
import { createLogger } from '../lib/logger';
import type { PlaybackTrend } from '../lib/types/core';
import type { ComparativeAnalyticsResponse } from '../lib/types/analytics';
import { escapeHtml } from '../lib/sanitize';

const logger = createLogger('AnomalyDetection');

/**
 * Detected anomaly structure
 */
interface DetectedAnomaly {
  date: string;
  value: number;
  type: 'spike' | 'dip';
  severity: 'high' | 'medium' | 'low';
  zScore: number;
  description: string;
  metric: string;
}

/**
 * Anomaly detection configuration
 */
interface AnomalyConfig {
  highThreshold: number;      // Z-score threshold for high severity
  mediumThreshold: number;    // Z-score threshold for medium severity
  lowThreshold: number;       // Z-score threshold for low severity
  minDataPoints: number;      // Minimum data points for analysis
}

const DEFAULT_CONFIG: AnomalyConfig = {
  highThreshold: 3.0,
  mediumThreshold: 2.5,
  lowThreshold: 2.0,
  minDataPoints: 7
};

export class AnomalyDetectionManager {
  private api: API;
  private config: AnomalyConfig;
  private isLoading: boolean = false;
  private detectedAnomalies: DetectedAnomaly[] = [];

  constructor(api: API, config: Partial<AnomalyConfig> = {}) {
    this.api = api;
    this.config = { ...DEFAULT_CONFIG, ...config };
  }

  /**
   * Initialize the anomaly detection manager
   */
  init(): void {
    logger.debug('AnomalyDetectionManager initialized');
  }

  /**
   * Analyze data for anomalies
   */
  async analyzeAnomalies(filter: LocationFilter = {}): Promise<DetectedAnomaly[]> {
    if (this.isLoading) return this.detectedAnomalies;

    this.isLoading = true;
    this.showLoading();
    this.detectedAnomalies = [];

    try {
      // Fetch trends data for analysis
      const [trendsData, comparativeData] = await Promise.all([
        this.api.getAnalyticsTrends(filter).catch(() => null),
        this.api.getAnalyticsComparative(filter, 'week').catch(() => null)
      ]);

      // Analyze playback trends
      if (trendsData?.playback_trends && trendsData.playback_trends.length >= this.config.minDataPoints) {
        const playbackAnomalies = this.detectPlaybackAnomalies(trendsData.playback_trends);
        this.detectedAnomalies.push(...playbackAnomalies);

        const userAnomalies = this.detectUserActivityAnomalies(trendsData.playback_trends);
        this.detectedAnomalies.push(...userAnomalies);
      }

      // Analyze comparative data for period-over-period anomalies
      if (comparativeData) {
        const comparativeAnomalies = this.detectComparativeAnomalies(comparativeData);
        this.detectedAnomalies.push(...comparativeAnomalies);
      }

      // Sort by severity and date
      this.detectedAnomalies.sort((a, b) => {
        const severityOrder = { high: 0, medium: 1, low: 2 };
        if (severityOrder[a.severity] !== severityOrder[b.severity]) {
          return severityOrder[a.severity] - severityOrder[b.severity];
        }
        return new Date(b.date).getTime() - new Date(a.date).getTime();
      });

      // Update the UI
      this.updateAnomaliesDisplay();
    } catch (error) {
      logger.error('Failed to analyze anomalies:', error);
      this.showError();
    } finally {
      this.isLoading = false;
    }

    return this.detectedAnomalies;
  }

  /**
   * Detect anomalies in playback counts
   */
  private detectPlaybackAnomalies(trends: PlaybackTrend[]): DetectedAnomaly[] {
    const anomalies: DetectedAnomaly[] = [];
    const values = trends.map(t => t.playback_count);
    const stats = this.calculateStats(values);

    if (stats.stdDev === 0) return anomalies; // No variance

    trends.forEach((trend) => {
      const zScore = (trend.playback_count - stats.mean) / stats.stdDev;
      const absZScore = Math.abs(zScore);

      if (absZScore >= this.config.lowThreshold) {
        const severity = this.getSeverity(absZScore);
        const type = zScore > 0 ? 'spike' : 'dip';

        anomalies.push({
          date: trend.date,
          value: trend.playback_count,
          type,
          severity,
          zScore,
          metric: 'playback_count',
          description: this.getPlaybackDescription(type, severity, trend.playback_count, stats.mean, trend.date)
        });
      }
    });

    return anomalies;
  }

  /**
   * Detect anomalies in user activity
   */
  private detectUserActivityAnomalies(trends: PlaybackTrend[]): DetectedAnomaly[] {
    const anomalies: DetectedAnomaly[] = [];
    const values = trends.map(t => t.unique_users);
    const stats = this.calculateStats(values);

    if (stats.stdDev === 0) return anomalies;

    trends.forEach((trend) => {
      const zScore = (trend.unique_users - stats.mean) / stats.stdDev;
      const absZScore = Math.abs(zScore);

      if (absZScore >= this.config.lowThreshold) {
        const severity = this.getSeverity(absZScore);
        const type = zScore > 0 ? 'spike' : 'dip';

        anomalies.push({
          date: trend.date,
          value: trend.unique_users,
          type,
          severity,
          zScore,
          metric: 'unique_users',
          description: this.getUserDescription(type, severity, trend.unique_users, stats.mean, trend.date)
        });
      }
    });

    return anomalies;
  }

  /**
   * Detect anomalies from comparative period data
   */
  private detectComparativeAnomalies(data: ComparativeAnalyticsResponse): DetectedAnomaly[] {
    const anomalies: DetectedAnomaly[] = [];

    if (!data.metrics_comparison) return anomalies;

    data.metrics_comparison.forEach(metric => {
      const percentChange = Math.abs(metric.percentage_change);

      // Consider >100% change as potential anomaly
      if (percentChange >= 100) {
        const severity = percentChange >= 300 ? 'high' : percentChange >= 200 ? 'medium' : 'low';
        const type = metric.percentage_change > 0 ? 'spike' : 'dip';

        anomalies.push({
          date: data.current_period?.end_date || new Date().toISOString().split('T')[0],
          value: metric.current_value,
          type,
          severity,
          zScore: percentChange / 100, // Use percentage as proxy for z-score
          metric: metric.metric,
          description: this.getComparativeDescription(metric.metric, type, percentChange)
        });
      }
    });

    return anomalies;
  }

  /**
   * Calculate mean and standard deviation
   */
  private calculateStats(values: number[]): { mean: number; stdDev: number } {
    if (values.length === 0) return { mean: 0, stdDev: 0 };

    const mean = values.reduce((sum, val) => sum + val, 0) / values.length;
    const squaredDiffs = values.map(val => Math.pow(val - mean, 2));
    const avgSquaredDiff = squaredDiffs.reduce((sum, val) => sum + val, 0) / values.length;
    const stdDev = Math.sqrt(avgSquaredDiff);

    return { mean, stdDev };
  }

  /**
   * Get severity level from z-score
   */
  private getSeverity(absZScore: number): 'high' | 'medium' | 'low' {
    if (absZScore >= this.config.highThreshold) return 'high';
    if (absZScore >= this.config.mediumThreshold) return 'medium';
    return 'low';
  }

  /**
   * Generate description for playback anomaly
   */
  private getPlaybackDescription(
    type: 'spike' | 'dip',
    severity: string,
    value: number,
    mean: number,
    date: string
  ): string {
    const percentDiff = Math.round(((value - mean) / mean) * 100);
    const dateStr = this.formatDate(date);

    if (type === 'spike') {
      return `${severity === 'high' ? 'Significant' : 'Notable'} playback spike on ${dateStr}: ${value} plays (${Math.abs(percentDiff)}% above average)`;
    }
    return `${severity === 'high' ? 'Significant' : 'Notable'} playback drop on ${dateStr}: ${value} plays (${Math.abs(percentDiff)}% below average)`;
  }

  /**
   * Generate description for user activity anomaly
   */
  private getUserDescription(
    type: 'spike' | 'dip',
    severity: string,
    value: number,
    mean: number,
    date: string
  ): string {
    const percentDiff = Math.round(((value - mean) / mean) * 100);
    const dateStr = this.formatDate(date);

    if (type === 'spike') {
      return `${severity === 'high' ? 'Unusual' : 'Notable'} user surge on ${dateStr}: ${value} users (${Math.abs(percentDiff)}% above average)`;
    }
    return `${severity === 'high' ? 'Unusual' : 'Notable'} user decline on ${dateStr}: ${value} users (${Math.abs(percentDiff)}% below average)`;
  }

  /**
   * Generate description for comparative anomaly
   */
  private getComparativeDescription(
    metric: string,
    type: 'spike' | 'dip',
    percentChange: number
  ): string {
    const metricName = this.formatMetricName(metric);
    const direction = type === 'spike' ? 'increased' : 'decreased';

    return `${metricName} has ${direction} by ${Math.round(percentChange)}% compared to the previous period`;
  }

  /**
   * Format date for display
   */
  private formatDate(date: string): string {
    try {
      return new Date(date).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric'
      });
    } catch {
      return date;
    }
  }

  /**
   * Format metric name for display
   */
  private formatMetricName(metric: string): string {
    const names: Record<string, string> = {
      'playback_count': 'Playback count',
      'unique_users': 'Active users',
      'watch_time_minutes': 'Watch time',
      'avg_completion': 'Completion rate',
      'unique_content': 'Content variety'
    };
    return names[metric] || metric.replace(/_/g, ' ');
  }

  /**
   * Show loading state
   */
  private showLoading(): void {
    const container = document.getElementById('anomaly-detection-content');
    if (container) {
      container.innerHTML = `
        <div class="anomaly-loading">
          <div class="anomaly-loading-spinner"></div>
          <span>Analyzing patterns...</span>
        </div>
      `;
    }
  }

  /**
   * Show error state
   */
  private showError(): void {
    const container = document.getElementById('anomaly-detection-content');
    if (container) {
      container.innerHTML = `
        <div class="anomaly-empty">
          <span class="anomaly-empty-icon" aria-hidden="true">&#x26A0;</span>
          <span>Unable to analyze patterns</span>
        </div>
      `;
    }
  }

  /**
   * Update the anomalies display
   */
  private updateAnomaliesDisplay(): void {
    const container = document.getElementById('anomaly-detection-content');
    if (!container) return;

    if (this.detectedAnomalies.length === 0) {
      container.innerHTML = `
        <div class="anomaly-empty">
          <span class="anomaly-empty-icon" aria-hidden="true">&#x2714;</span>
          <span>No unusual patterns detected</span>
        </div>
      `;
      return;
    }

    // Show top 5 anomalies
    const topAnomalies = this.detectedAnomalies.slice(0, 5);
    const html = topAnomalies.map(anomaly => this.renderAnomalyCard(anomaly)).join('');

    container.innerHTML = html;
  }

  /**
   * Render a single anomaly card
   * Defense in depth: escape description even though it's internally generated
   */
  private renderAnomalyCard(anomaly: DetectedAnomaly): string {
    const icon = anomaly.type === 'spike' ? '\u2191' : '\u2193';
    const severityClass = `anomaly-${anomaly.severity}`;
    const typeClass = `anomaly-${anomaly.type}`;
    const escapedDescription = escapeHtml(anomaly.description);

    return `
      <div class="anomaly-card ${severityClass} ${typeClass}" role="article" aria-label="${escapedDescription}">
        <div class="anomaly-indicator">
          <span class="anomaly-icon" aria-hidden="true">${icon}</span>
          <span class="anomaly-severity">${escapeHtml(anomaly.severity)}</span>
        </div>
        <div class="anomaly-content">
          <p class="anomaly-description">${escapedDescription}</p>
        </div>
      </div>
    `;
  }

  /**
   * Get detected anomalies
   */
  getAnomalies(): DetectedAnomaly[] {
    return this.detectedAnomalies;
  }

  /**
   * Check if there are high severity anomalies
   */
  hasHighSeverityAnomalies(): boolean {
    return this.detectedAnomalies.some(a => a.severity === 'high');
  }

  /**
   * Destroy the manager and cleanup resources
   */
  destroy(): void {
    this.detectedAnomalies = [];
    this.isLoading = false;

    // Clear the container
    const container = document.getElementById('anomaly-detection-content');
    if (container) {
      container.innerHTML = '';
    }
  }
}

export default AnomalyDetectionManager;
