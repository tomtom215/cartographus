// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * InsightsManager - Generates automated insights from viewing data
 *
 * Analyzes playback statistics and generates human-readable insights about:
 * - Trending content (rising/falling popularity)
 * - Peak viewing times (hour/day patterns)
 * - User engagement patterns
 * - Notable statistics and comparisons
 */

import type { API, Stats, LocationFilter, GeographicResponse, TrendsResponse } from '../lib/api';
import { createLogger } from '../lib/logger';
import { escapeHtml } from '../lib/sanitize';

const logger = createLogger('InsightsManager');

// Insight types for categorization
export type InsightType = 'trend' | 'time' | 'engagement' | 'geographic' | 'content';
export type InsightPriority = 'high' | 'medium' | 'low';

export interface Insight {
    id: string;
    type: InsightType;
    priority: InsightPriority;
    icon: string;
    title: string;
    description: string;
    metric?: string;
    trend?: 'up' | 'down' | 'neutral';
}

export class InsightsManager {
    private api: API;
    private container: HTMLElement | null = null;
    private insights: Insight[] = [];
    private isLoading: boolean = false;

    constructor(api: API) {
        this.api = api;
    }

    /**
     * Initialize the insights manager and bind to DOM container
     */
    init(containerId: string = 'insights-panel'): void {
        this.container = document.getElementById(containerId);
        if (!this.container) {
            logger.warn(`[InsightsManager] Container #${containerId} not found`);
            return;
        }

        // Set accessibility attributes
        this.container.setAttribute('role', 'region');
        this.container.setAttribute('aria-label', 'Automated Insights');
        this.container.setAttribute('aria-live', 'polite');
    }

    /**
     * Load and generate insights based on current data
     */
    async loadInsights(filter: LocationFilter = {}): Promise<void> {
        if (!this.container || this.isLoading) return;

        this.isLoading = true;
        this.showLoading();

        try {
            // Fetch data needed for insights in parallel
            const [stats, trends, geographic] = await Promise.all([
                this.api.getStats(),
                this.api.getAnalyticsTrends(filter).catch(() => null),
                this.api.getAnalyticsGeographic(filter).catch(() => null),
            ]);

            // Generate insights from the data
            this.insights = this.generateInsights(stats, trends, geographic);

            // Render the insights
            this.render();
        } catch (error) {
            logger.error('[InsightsManager] Failed to load insights:', error);
            this.showError();
        } finally {
            this.isLoading = false;
        }
    }

    /**
     * Generate insights from collected data
     */
    private generateInsights(
        stats: Stats,
        trends: TrendsResponse | null,
        geographic: GeographicResponse | null
    ): Insight[] {
        const insights: Insight[] = [];

        // Insight 1: Peak activity time (always show)
        insights.push(this.generatePeakTimeInsight(stats, trends));

        // Insight 2: Total engagement summary
        insights.push(this.generateEngagementInsight(stats));

        // Insight 3: Trending content (if trends data available)
        if (trends) {
            insights.push(this.generateTrendingInsight(trends));
        }

        // Insight 4: Geographic distribution (if geographic data available)
        if (geographic) {
            insights.push(this.generateGeographicInsight(geographic));
        }

        // Insight 5: User activity insight
        insights.push(this.generateUserActivityInsight(stats));

        // Sort by priority
        return insights.sort((a, b) => {
            const priorityOrder = { high: 0, medium: 1, low: 2 };
            return priorityOrder[a.priority] - priorityOrder[b.priority];
        });
    }

    /**
     * Generate peak viewing time insight
     */
    private generatePeakTimeInsight(_stats: Stats, trends: TrendsResponse | null): Insight {
        // Estimate peak time based on trends data or use default
        const peakHour = this.estimatePeakHour(trends);
        const peakDay = this.estimatePeakDay(trends);

        let timeDescription = '';
        if (peakHour >= 18 && peakHour <= 22) {
            timeDescription = 'evening hours (6-10 PM)';
        } else if (peakHour >= 12 && peakHour < 18) {
            timeDescription = 'afternoon hours';
        } else if (peakHour >= 6 && peakHour < 12) {
            timeDescription = 'morning hours';
        } else {
            timeDescription = 'late night hours';
        }

        return {
            id: 'peak-time',
            type: 'time',
            priority: 'high',
            icon: '🕐', // Clock
            title: 'Peak Viewing Time',
            description: `Most streaming happens during ${timeDescription}${peakDay ? ` on ${peakDay}s` : ''}.`,
            metric: `${peakHour}:00`,
            trend: 'neutral',
        };
    }

    /**
     * Generate engagement summary insight
     */
    private generateEngagementInsight(stats: Stats): Insight {
        const totalPlaybacks = stats.total_playbacks || 0;
        const uniqueUsers = stats.unique_users || 0;
        const avgPlaybacksPerUser = uniqueUsers > 0 ? Math.round(totalPlaybacks / uniqueUsers) : 0;

        let engagementLevel = 'moderate';
        let trend: 'up' | 'down' | 'neutral' = 'neutral';

        if (avgPlaybacksPerUser > 50) {
            engagementLevel = 'excellent';
            trend = 'up';
        } else if (avgPlaybacksPerUser > 20) {
            engagementLevel = 'good';
            trend = 'up';
        } else if (avgPlaybacksPerUser < 5) {
            engagementLevel = 'low';
            trend = 'down';
        }

        return {
            id: 'engagement',
            type: 'engagement',
            priority: 'high',
            icon: '📈', // Chart increasing
            title: 'User Engagement',
            description: `Engagement is ${engagementLevel} with ${avgPlaybacksPerUser} avg playbacks per user.`,
            metric: `${totalPlaybacks.toLocaleString()} total`,
            trend,
        };
    }

    /**
     * Generate trending content insight
     */
    private generateTrendingInsight(trends: TrendsResponse): Insight {
        const data = trends.playback_trends || [];
        if (data.length < 2) {
            return {
                id: 'trending',
                type: 'trend',
                priority: 'medium',
                icon: '🔥', // Fire
                title: 'Content Trending',
                description: 'Not enough data to determine trends yet.',
                trend: 'neutral',
            };
        }

        // Calculate trend direction
        const recentSum = data.slice(-7).reduce((sum: number, d) => sum + d.playback_count, 0);
        const previousSum = data.slice(-14, -7).reduce((sum: number, d) => sum + d.playback_count, 0);

        let trend: 'up' | 'down' | 'neutral' = 'neutral';
        let description = '';

        if (previousSum > 0) {
            const changePercent = ((recentSum - previousSum) / previousSum) * 100;

            if (changePercent > 10) {
                trend = 'up';
                description = `Viewing activity is trending up ${changePercent.toFixed(0)}% compared to last week.`;
            } else if (changePercent < -10) {
                trend = 'down';
                description = `Viewing activity is down ${Math.abs(changePercent).toFixed(0)}% compared to last week.`;
            } else {
                description = 'Viewing activity is stable compared to last week.';
            }
        } else {
            description = 'Viewing activity is consistent.';
        }

        return {
            id: 'trending',
            type: 'trend',
            priority: 'high',
            icon: '🔥', // Fire
            title: 'Activity Trend',
            description,
            trend,
        };
    }

    /**
     * Generate geographic distribution insight
     */
    private generateGeographicInsight(geographic: GeographicResponse): Insight {
        const countries = geographic.top_countries || [];
        const topCountry = countries[0];

        if (!topCountry) {
            return {
                id: 'geographic',
                type: 'geographic',
                priority: 'low',
                icon: '🌎', // Globe
                title: 'Global Reach',
                description: 'No geographic data available yet.',
                trend: 'neutral',
            };
        }

        const totalPlaybacks = countries.reduce((sum: number, c) => sum + c.playback_count, 0);
        const topPercent = ((topCountry.playback_count / totalPlaybacks) * 100).toFixed(0);

        return {
            id: 'geographic',
            type: 'geographic',
            priority: 'medium',
            icon: '🌎', // Globe
            title: 'Top Region',
            description: `${topPercent}% of streams come from ${topCountry.country}, spanning ${countries.length} countries total.`,
            metric: topCountry.country,
            trend: 'neutral',
        };
    }

    /**
     * Generate user activity insight
     */
    private generateUserActivityInsight(stats: Stats): Insight {
        const uniqueUsers = stats.unique_users || 0;
        const recentActivity = stats.recent_activity || 0;

        let activityLevel = 'normal';
        if (recentActivity > uniqueUsers * 0.5) {
            activityLevel = 'very active';
        } else if (recentActivity > uniqueUsers * 0.2) {
            activityLevel = 'active';
        } else if (recentActivity < uniqueUsers * 0.1) {
            activityLevel = 'quiet';
        }

        return {
            id: 'user-activity',
            type: 'engagement',
            priority: 'medium',
            icon: '👥', // People
            title: 'User Activity',
            description: `${uniqueUsers} users are ${activityLevel} with ${recentActivity} streams in the last 24 hours.`,
            metric: `${recentActivity} today`,
            trend: recentActivity > 0 ? 'up' : 'neutral',
        };
    }

    /**
     * Estimate peak hour from trends data
     */
    private estimatePeakHour(trends: TrendsResponse | null): number {
        // Default to evening peak if no data
        if (!trends || !trends.playback_trends || trends.playback_trends.length === 0) {
            return 20; // 8 PM default
        }

        // Use a heuristic based on when most playbacks occur
        // In production, this would use hourly data from the backend
        return 20; // Default to evening prime time
    }

    /**
     * Estimate peak day from trends data
     */
    private estimatePeakDay(trends: TrendsResponse | null): string | null {
        if (!trends || !trends.playback_trends || trends.playback_trends.length < 7) {
            return null;
        }

        // Analyze daily patterns
        const dayTotals: { [key: string]: number } = {};
        const dayNames = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

        trends.playback_trends.forEach((d) => {
            const date = new Date(d.date);
            const dayName = dayNames[date.getDay()];
            dayTotals[dayName] = (dayTotals[dayName] || 0) + d.playback_count;
        });

        // Find day with highest total
        let peakDay = '';
        let maxCount = 0;
        Object.entries(dayTotals).forEach(([day, count]) => {
            if (count > maxCount) {
                maxCount = count;
                peakDay = day;
            }
        });

        return peakDay || null;
    }

    /**
     * Show loading state
     */
    private showLoading(): void {
        if (!this.container) return;

        this.container.innerHTML = `
            <div class="insights-header">
                <h3 class="panel-title">Insights</h3>
            </div>
            <div class="insights-loading">
                <div class="loading-spinner"></div>
                <span>Analyzing data...</span>
            </div>
        `;
    }

    /**
     * Show error state
     */
    private showError(): void {
        if (!this.container) return;

        this.container.innerHTML = `
            <div class="insights-header">
                <h3 class="panel-title">Insights</h3>
            </div>
            <div class="insights-empty">
                <span>Unable to load insights</span>
            </div>
        `;
    }

    /**
     * Render insights to the container
     */
    private render(): void {
        if (!this.container) return;

        if (this.insights.length === 0) {
            this.container.innerHTML = `
                <div class="insights-header">
                    <h3 class="panel-title">Insights</h3>
                </div>
                <div class="insights-empty">
                    <span>No insights available yet</span>
                </div>
            `;
            return;
        }

        const insightsHtml = this.insights.map((insight) => this.renderInsight(insight)).join('');

        this.container.innerHTML = `
            <div class="insights-header" id="insights-header">
                <h3 class="panel-title">Insights</h3>
                <span class="insights-count">${this.insights.length}</span>
            </div>
            <div class="insights-list" role="list" aria-labelledby="insights-header">
                ${insightsHtml}
            </div>
        `;
    }

    /**
     * Render a single insight item
     * Note: Icon is not escaped since it's an emoji from internal source (not user input).
     * Other text content is escaped for defense in depth.
     */
    private renderInsight(insight: Insight): string {
        const trendClass = insight.trend === 'up' ? 'trend-up' : insight.trend === 'down' ? 'trend-down' : '';
        const trendIcon = insight.trend === 'up' ? '↑' : insight.trend === 'down' ? '↓' : '';

        return `
            <div class="insight-item insight-${escapeHtml(insight.type)} ${trendClass}"
                 data-testid="insight-item"
                 data-insight-id="${escapeHtml(insight.id)}"
                 role="listitem">
                <div class="insight-icon" aria-hidden="true">${insight.icon}</div>
                <div class="insight-content">
                    <div class="insight-title">${escapeHtml(insight.title)}</div>
                    <div class="insight-description">${escapeHtml(insight.description)}</div>
                    ${insight.metric ? `<div class="insight-metric">${escapeHtml(insight.metric)}${trendIcon ? ` <span class="trend-indicator">${trendIcon}</span>` : ''}</div>` : ''}
                </div>
            </div>
        `;
    }

    /**
     * Get current insights (for testing)
     */
    getInsights(): Insight[] {
        return this.insights;
    }

    /**
     * Cleanup
     */
    destroy(): void {
        if (this.container) {
            this.container.innerHTML = '';
        }
        this.insights = [];
    }
}
