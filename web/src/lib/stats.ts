// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import type { API, Stats } from './api';
import { createLogger } from './logger';
import { SafeStorage } from './utils/SafeStorage';

const logger = createLogger('Stats');

/**
 * Statistics for trend calculation
 */
interface TrendStats {
    current: number;
    previous: number;
}

/**
 * Previous stats stored for trend comparison
 */
interface PreviousStats {
    total_playbacks: number;
    unique_locations: number;
    unique_users: number;
    recent_activity: number;
}

/**
 * Historical data points for sparkline visualization
 */
interface SparklineHistory {
    playbacks: number[];
    locations: number[];
    users: number[];
    recent: number[];
    timestamps: number[];
}

/** Maximum number of data points to show in sparkline */
const SPARKLINE_MAX_POINTS = 7;

/**
 * StatsManager - Handles loading and displaying statistics with trend indicators
 *
 * Features:
 * - Trend indicators showing up/down/neutral
 * - Sparkline mini trend visualizations
 * - Percentage change calculation
 * - Color-coded trend display (green up, red down, gray neutral)
 *
 * Reference: UI/UX Audit Tasks
 * @see /docs/working/UI_UX_AUDIT.md
 */
export class StatsManager {
    private previousStats: PreviousStats | null = null;
    /** Historical data for sparklines */
    private sparklineHistory: SparklineHistory | null = null;

    constructor(private api: API) {
        // Load previous stats from localStorage if available
        this.loadPreviousStats();
        // Load sparkline history
        this.loadSparklineHistory();
    }

    async loadStats(): Promise<void> {
        try {
            const stats = await this.api.getStats();
            this.updateStatsUI(stats);
            // Save current stats for next comparison
            this.savePreviousStats(stats);
            // Update sparkline history and render
            this.updateSparklineHistory(stats);
            this.renderSparklines();
        } catch (error) {
            logger.error('Failed to load stats:', error);
        }
    }

    /**
     * Load previous stats from localStorage
     */
    private loadPreviousStats(): void {
        const saved = SafeStorage.getItem('stats-previous');
        if (saved) {
            try {
                const parsed = JSON.parse(saved);
                // Only use if not older than 24 hours
                if (parsed.timestamp && Date.now() - parsed.timestamp < 24 * 60 * 60 * 1000) {
                    this.previousStats = parsed.stats;
                }
            } catch {
                // Ignore parse errors
            }
        }
    }

    /**
     * Save current stats for next comparison
     */
    private savePreviousStats(stats: Stats): void {
        SafeStorage.setJSON('stats-previous', {
            timestamp: Date.now(),
            stats: {
                total_playbacks: stats.total_playbacks,
                unique_locations: stats.unique_locations,
                unique_users: stats.unique_users,
                recent_activity: stats.recent_activity
            }
        });
    }

    private updateStatsUI(stats: Stats): void {
        this.updateStatWithTrend('stat-playbacks', stats.total_playbacks, this.previousStats?.total_playbacks);
        this.updateStatWithTrend('stat-locations', stats.unique_locations, this.previousStats?.unique_locations);
        this.updateStatWithTrend('stat-users', stats.unique_users, this.previousStats?.unique_users);
        this.updateStatWithTrend('stat-recent', stats.recent_activity, this.previousStats?.recent_activity);
    }

    /**
     * Update a stat value and its trend indicator
     */
    private updateStatWithTrend(id: string, currentValue: number, previousValue: number | undefined): void {
        // Update the main value
        const valueElement = document.getElementById(id);
        if (valueElement) {
            valueElement.textContent = currentValue.toLocaleString();
        }

        // Update the trend indicator
        const trendElement = document.getElementById(`${id}-trend`);
        if (trendElement) {
            this.updateTrendIndicator(trendElement, {
                current: currentValue,
                previous: previousValue ?? currentValue
            });
        }
    }

    /**
     * Update trend indicator element with calculated trend
     */
    private updateTrendIndicator(element: HTMLElement, stats: TrendStats): void {
        const { current, previous } = stats;

        // Calculate percentage change
        let percentChange = 0;
        if (previous > 0) {
            percentChange = ((current - previous) / previous) * 100;
        } else if (current > 0) {
            percentChange = 100; // From zero to something is 100% increase
        }

        // Determine direction
        let direction: 'up' | 'down' | 'neutral';
        let arrow: string;

        if (percentChange > 0.5) {
            direction = 'up';
            arrow = '\u2191'; // Up arrow
        } else if (percentChange < -0.5) {
            direction = 'down';
            arrow = '\u2193'; // Down arrow
        } else {
            direction = 'neutral';
            arrow = '-';
        }

        // Update element
        element.setAttribute('data-direction', direction);
        element.classList.remove('trend-up', 'trend-down', 'trend-neutral');
        element.classList.add(`trend-${direction}`);

        // Update arrow
        const arrowElement = element.querySelector('.trend-arrow');
        if (arrowElement) {
            arrowElement.textContent = arrow;
        }

        // Update percentage value
        const valueElement = element.querySelector('.stat-trend-value');
        if (valueElement) {
            if (direction === 'neutral') {
                valueElement.textContent = '0%';
            } else {
                const sign = percentChange > 0 ? '+' : '';
                valueElement.textContent = `${sign}${Math.round(percentChange)}%`;
            }
        }

        // Update aria-label for accessibility
        const ariaLabel = this.getAriaLabel(direction, percentChange);
        element.setAttribute('aria-label', ariaLabel);
        element.setAttribute('title', ariaLabel);
    }

    /**
     * Generate accessible aria-label for trend
     */
    private getAriaLabel(direction: 'up' | 'down' | 'neutral', percentChange: number): string {
        switch (direction) {
            case 'up':
                return `Increased ${Math.abs(Math.round(percentChange))}% vs previous period`;
            case 'down':
                return `Decreased ${Math.abs(Math.round(percentChange))}% vs previous period`;
            default:
                return 'No significant change vs previous period';
        }
    }

    /**
     * Update a single stat value (for WebSocket updates)
     */
    updateStat(id: string, value: number): void {
        const element = document.getElementById(id);
        if (element) {
            element.textContent = value.toLocaleString();
        }
    }

    // =========================================================================
    // Sparkline Methods
    // Mini trend line visualization for stat cards
    // =========================================================================

    /**
     * Load sparkline history from localStorage
     */
    private loadSparklineHistory(): void {
        const saved = SafeStorage.getItem('stats-sparkline-history');
        if (saved) {
            try {
                const parsed = JSON.parse(saved);
                // Only use if recent (within last week)
                const oneWeekAgo = Date.now() - (7 * 24 * 60 * 60 * 1000);
                if (parsed.timestamps?.length > 0 && parsed.timestamps[parsed.timestamps.length - 1] > oneWeekAgo) {
                    this.sparklineHistory = parsed;
                }
            } catch {
                // Ignore parse errors
            }
        }

        // Initialize if no valid history
        if (!this.sparklineHistory) {
            this.sparklineHistory = {
                playbacks: [],
                locations: [],
                users: [],
                recent: [],
                timestamps: []
            };
        }
    }

    /**
     * Update sparkline history with new stats
     */
    private updateSparklineHistory(stats: Stats): void {
        if (!this.sparklineHistory) return;

        const now = Date.now();
        const lastTimestamp = this.sparklineHistory.timestamps[this.sparklineHistory.timestamps.length - 1] || 0;

        // Only add new data point if at least 1 hour has passed
        const oneHour = 60 * 60 * 1000;
        if (now - lastTimestamp < oneHour && this.sparklineHistory.timestamps.length > 0) {
            // Update the last value instead
            const lastIdx = this.sparklineHistory.playbacks.length - 1;
            if (lastIdx >= 0) {
                this.sparklineHistory.playbacks[lastIdx] = stats.total_playbacks;
                this.sparklineHistory.locations[lastIdx] = stats.unique_locations;
                this.sparklineHistory.users[lastIdx] = stats.unique_users;
                this.sparklineHistory.recent[lastIdx] = stats.recent_activity;
            }
        } else {
            // Add new data point
            this.sparklineHistory.playbacks.push(stats.total_playbacks);
            this.sparklineHistory.locations.push(stats.unique_locations);
            this.sparklineHistory.users.push(stats.unique_users);
            this.sparklineHistory.recent.push(stats.recent_activity);
            this.sparklineHistory.timestamps.push(now);

            // Keep only last N points
            if (this.sparklineHistory.playbacks.length > SPARKLINE_MAX_POINTS) {
                this.sparklineHistory.playbacks.shift();
                this.sparklineHistory.locations.shift();
                this.sparklineHistory.users.shift();
                this.sparklineHistory.recent.shift();
                this.sparklineHistory.timestamps.shift();
            }
        }

        // Save to localStorage
        this.saveSparklineHistory();
    }

    /**
     * Save sparkline history to localStorage
     */
    private saveSparklineHistory(): void {
        SafeStorage.setJSON('stats-sparkline-history', this.sparklineHistory);
    }

    /**
     * Render all sparklines
     */
    private renderSparklines(): void {
        if (!this.sparklineHistory) return;

        this.renderSparkline('sparkline-playbacks', this.sparklineHistory.playbacks);
        this.renderSparkline('sparkline-locations', this.sparklineHistory.locations);
        this.renderSparkline('sparkline-users', this.sparklineHistory.users);
        this.renderSparkline('sparkline-recent', this.sparklineHistory.recent);
    }

    /**
     * Render a single sparkline SVG
     */
    private renderSparkline(containerId: string, data: number[]): void {
        const container = document.getElementById(containerId);
        if (!container) return;

        // Need at least 2 points to draw a line
        if (data.length < 2) {
            container.innerHTML = '';
            return;
        }

        // Calculate trend direction
        const first = data[0];
        const last = data[data.length - 1];
        let direction: 'up' | 'down' | 'neutral';

        if (first === 0 && last === 0) {
            direction = 'neutral';
        } else if (first === 0) {
            direction = 'up';
        } else {
            const change = ((last - first) / first) * 100;
            if (change > 0.5) {
                direction = 'up';
            } else if (change < -0.5) {
                direction = 'down';
            } else {
                direction = 'neutral';
            }
        }

        // Update container class for styling
        container.classList.remove('trend-up', 'trend-down', 'trend-neutral');
        container.classList.add(`trend-${direction}`);

        // Generate SVG points
        const width = 80;
        const height = 24;
        const padding = 2;

        const min = Math.min(...data);
        const max = Math.max(...data);
        const range = max - min || 1; // Avoid division by zero

        const points = data.map((value, index) => {
            const x = padding + (index / (data.length - 1)) * (width - 2 * padding);
            const y = height - padding - ((value - min) / range) * (height - 2 * padding);
            return `${x.toFixed(1)},${y.toFixed(1)}`;
        }).join(' ');

        // Create SVG
        const svg = `
            <svg viewBox="0 0 ${width} ${height}" preserveAspectRatio="none">
                <polyline points="${points}" />
            </svg>
        `;

        container.innerHTML = svg;

        // Update aria-label with trend info
        const trendText = direction === 'up' ? 'increasing' : direction === 'down' ? 'decreasing' : 'stable';
        container.setAttribute('aria-label', `Trend line showing ${trendText} values over recent period`);
    }

    /**
     * Cleanup resources to prevent memory leaks
     * Clears internal state
     */
    destroy(): void {
        this.previousStats = null;
        this.sparklineHistory = null;
    }
}
