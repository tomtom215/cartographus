// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * WrappedDashboardManager - Annual Wrapped Reports Dashboard
 *
 * Manages the "Spotify Wrapped" style annual analytics feature.
 * Provides animated card-by-card reveal of statistics, timeline visualization,
 * share card generation, and year-over-year comparison.
 */

import type { API } from './api';
import type { ToastManager } from './toast';
import type {
    WrappedReport,
    WrappedServerStats,
    WrappedContentRank,
    WrappedGenreRank,
    WrappedAchievement,
    WrappedMonthly,
} from './types/wrapped';
import { createLogger } from './logger';

const logger = createLogger('WrappedDashboard');

/**
 * Day names for display
 */
const DAY_NAMES = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

/**
 * Achievement icons mapping
 */
const ACHIEVEMENT_ICONS: Record<string, string> = {
    'bingemaster': 'play-circle',
    'night_owl': 'moon',
    'early_bird': 'sunrise',
    'weekend_warrior': 'calendar',
    'movie_buff': 'film',
    'series_addict': 'tv',
    'quality_enthusiast': 'award',
    'explorer': 'compass',
    'marathoner': 'clock',
    'consistent': 'trending-up',
};

/**
 * WrappedDashboardManager handles the Annual Wrapped Reports UI
 */
export class WrappedDashboardManager {
    private api: API;
    private toastManager: ToastManager | null = null;
    private currentYear: number;
    private currentUserID: number | null = null;
    private currentReport: WrappedReport | null = null;

    constructor(api: API) {
        this.api = api;
        this.currentYear = new Date().getFullYear();
    }

    /**
     * Set toast manager reference for notifications
     */
    setToastManager(toast: ToastManager): void {
        this.toastManager = toast;
    }

    /**
     * Set current user ID for personal reports
     */
    setCurrentUserID(userID: number): void {
        this.currentUserID = userID;
    }

    /**
     * Initialize the wrapped dashboard
     */
    async init(): Promise<void> {
        logger.info('Initializing wrapped dashboard');

        this.setupEventListeners();
        this.populateYearSelector();

        // Load initial data
        await this.loadWrappedData();
    }

    /**
     * Clean up resources
     */
    destroy(): void {
        this.currentReport = null;
        logger.debug('Wrapped dashboard destroyed');
    }

    /**
     * Set up event listeners for UI interactions
     */
    private setupEventListeners(): void {
        // Year selector
        const yearSelect = document.getElementById('wrapped-year-select') as HTMLSelectElement;
        if (yearSelect) {
            yearSelect.addEventListener('change', () => {
                this.currentYear = parseInt(yearSelect.value, 10);
                this.loadWrappedData();
            });
        }

        // Generate button (admin)
        const generateBtn = document.getElementById('wrapped-generate-btn');
        if (generateBtn) {
            generateBtn.addEventListener('click', () => this.handleGenerateReports());
        }

        // Share button
        const shareBtn = document.getElementById('wrapped-share-btn');
        if (shareBtn) {
            shareBtn.addEventListener('click', () => this.handleShare());
        }

        // Export button
        const exportBtn = document.getElementById('wrapped-export-btn');
        if (exportBtn) {
            exportBtn.addEventListener('click', () => this.handleExport());
        }
    }

    /**
     * Populate the year selector with available years
     */
    private populateYearSelector(): void {
        const yearSelect = document.getElementById('wrapped-year-select') as HTMLSelectElement;
        if (!yearSelect) return;

        // Clear existing options
        yearSelect.innerHTML = '';

        // Add years from current year back to 2020
        const currentYear = new Date().getFullYear();
        for (let year = currentYear; year >= 2020; year--) {
            const option = document.createElement('option');
            option.value = year.toString();
            option.textContent = year.toString();
            if (year === this.currentYear) {
                option.selected = true;
            }
            yearSelect.appendChild(option);
        }
    }

    /**
     * Load wrapped data for the current year
     */
    private async loadWrappedData(): Promise<void> {
        this.showLoading(true);

        try {
            // Load server aggregate stats (no individual user data for privacy)
            const serverStats = await this.api.getWrappedServerStats(this.currentYear);
            this.renderServerStats(serverStats);

            // Note: Leaderboard is admin-only and not shown to regular users
            // to protect user privacy. Individual users only see their own data.

            // Load user report if user ID is set
            if (this.currentUserID) {
                try {
                    const report = await this.api.getWrappedUserReport(
                        this.currentYear,
                        this.currentUserID,
                        true // generate if not exists
                    );
                    this.currentReport = report;
                    this.renderUserReport(report);
                } catch (error) {
                    logger.warn('No wrapped report for current user:', error);
                    this.showNoReportMessage();
                }
            }

            this.showLoading(false);
        } catch (error) {
            logger.error('Failed to load wrapped data:', error);
            this.showError('Failed to load wrapped data');
            this.showLoading(false);
        }
    }

    /**
     * Render server-wide statistics
     */
    private renderServerStats(stats: WrappedServerStats): void {
        // Total users
        this.setElementText('wrapped-total-users', stats.total_users.toLocaleString());

        // Total watch time
        this.setElementText('wrapped-total-watch-time', this.formatWatchTime(stats.total_watch_time_hours));

        // Total playbacks
        this.setElementText('wrapped-total-playbacks', stats.total_playbacks.toLocaleString());

        // Unique content
        this.setElementText('wrapped-unique-content', stats.unique_content_watched.toLocaleString());

        // Top movie
        this.setElementText('wrapped-top-movie', stats.top_movie || 'N/A');

        // Top show
        this.setElementText('wrapped-top-show', stats.top_show || 'N/A');

        // Top genre
        this.setElementText('wrapped-top-genre', stats.top_genre || 'N/A');

        // Peak month
        this.setElementText('wrapped-peak-month', stats.peak_month || 'N/A');

        // Average completion rate
        this.setElementText('wrapped-avg-completion', `${stats.avg_completion_rate.toFixed(1)}%`);
    }

    // Note: Leaderboard rendering removed for user privacy.
    // Individual users should only see their own statistics.
    // Admin-only leaderboard would require role-based access control.

    /**
     * Render user-specific wrapped report
     */
    private renderUserReport(report: WrappedReport): void {
        // Core stats
        this.renderCoreStats(report);

        // Content rankings
        this.renderTopContent(report);

        // Viewing patterns
        this.renderViewingPatterns(report);

        // Timeline
        this.renderTimeline(report.monthly_trends);

        // Achievements
        this.renderAchievements(report.achievements);

        // Share card
        this.renderShareCard(report);

        // Start reveal animation
        this.startRevealAnimation();
    }

    /**
     * Render core statistics
     */
    private renderCoreStats(report: WrappedReport): void {
        // Total watch time
        const watchTimeEl = document.getElementById('wrapped-user-watch-time');
        if (watchTimeEl) {
            watchTimeEl.textContent = this.formatWatchTime(report.total_watch_time_hours);
            watchTimeEl.setAttribute('data-animate', 'true');
        }

        // Total playbacks
        this.setElementText('wrapped-user-playbacks', report.total_playbacks.toLocaleString());

        // Unique content
        this.setElementText('wrapped-user-unique-content', report.unique_content_count.toLocaleString());

        // Completion rate
        this.setElementText('wrapped-user-completion', `${report.completion_rate.toFixed(1)}%`);

        // Days active
        this.setElementText('wrapped-user-days-active', report.days_active.toString());

        // Longest streak
        this.setElementText('wrapped-user-streak', `${report.longest_streak_days} days`);

        // Binge sessions
        this.setElementText('wrapped-user-binges', report.binge_sessions.toString());

        // Discovery rate
        this.setElementText('wrapped-user-discovery', `${report.discovery_rate.toFixed(1)}%`);

        // Preferred platform
        this.setElementText('wrapped-user-platform', report.preferred_platform || 'N/A');

        // Quality stats
        this.setElementText('wrapped-user-direct-play', `${report.direct_play_rate.toFixed(0)}%`);
        this.setElementText('wrapped-user-4k', `${report['4k_viewing_percent'].toFixed(0)}%`);
        this.setElementText('wrapped-user-hdr', `${report.hdr_viewing_percent.toFixed(0)}%`);
    }

    /**
     * Render top content sections
     */
    private renderTopContent(report: WrappedReport): void {
        // Top movies
        this.renderContentList('wrapped-top-movies', report.top_movies);

        // Top shows
        this.renderContentList('wrapped-top-shows', report.top_shows);

        // Top genres
        this.renderGenreList('wrapped-top-genres', report.top_genres);
    }

    /**
     * Render a content list (movies/shows)
     */
    private renderContentList(containerId: string, items: WrappedContentRank[]): void {
        const container = document.getElementById(containerId);
        if (!container) return;

        if (!items || items.length === 0) {
            container.innerHTML = '<p class="empty-state">No data available</p>';
            return;
        }

        const html = items.slice(0, 5).map(item => `
            <div class="content-rank-item">
                <div class="content-rank-number">#${item.rank}</div>
                <div class="content-rank-info">
                    <div class="content-rank-title">${this.escapeHtml(item.title)}</div>
                    <div class="content-rank-stats">
                        <span>${item.watch_count} plays</span>
                        <span>${this.formatWatchTime(item.watch_time_hours)}</span>
                    </div>
                </div>
            </div>
        `).join('');

        container.innerHTML = html;
    }

    /**
     * Render genre list
     */
    private renderGenreList(containerId: string, genres: WrappedGenreRank[]): void {
        const container = document.getElementById(containerId);
        if (!container) return;

        if (!genres || genres.length === 0) {
            container.innerHTML = '<p class="empty-state">No genre data available</p>';
            return;
        }

        const html = genres.slice(0, 5).map(genre => `
            <div class="genre-rank-item">
                <div class="genre-rank-info">
                    <div class="genre-rank-name">${this.escapeHtml(genre.genre)}</div>
                    <div class="genre-rank-bar" style="width: ${genre.percentage}%"></div>
                </div>
                <div class="genre-rank-stats">
                    <span>${genre.percentage.toFixed(1)}%</span>
                </div>
            </div>
        `).join('');

        container.innerHTML = html;
    }

    /**
     * Render viewing patterns
     */
    private renderViewingPatterns(report: WrappedReport): void {
        // Peak hour
        this.setElementText('wrapped-peak-hour', this.formatHour(report.peak_hour));

        // Peak day
        this.setElementText('wrapped-peak-day', report.peak_day);

        // Peak month
        this.setElementText('wrapped-user-peak-month', report.peak_month);

        // First watch
        this.setElementText('wrapped-first-watch', report.first_watch_of_year || 'N/A');

        // Last watch
        this.setElementText('wrapped-last-watch', report.last_watch_of_year || 'N/A');

        // Render hour distribution
        this.renderHourChart(report.viewing_by_hour);

        // Render day distribution
        this.renderDayChart(report.viewing_by_day);
    }

    /**
     * Render hour distribution chart
     */
    private renderHourChart(viewingByHour: number[]): void {
        const container = document.getElementById('wrapped-hour-chart');
        if (!container) return;

        const maxValue = Math.max(...viewingByHour, 1);

        const bars = viewingByHour.map((value, hour) => {
            const height = (value / maxValue) * 100;
            return `
                <div class="hour-bar" style="height: ${height}%" title="${hour}:00 - ${value} plays">
                    <span class="hour-label">${hour}</span>
                </div>
            `;
        }).join('');

        container.innerHTML = `<div class="hour-chart-bars">${bars}</div>`;
    }

    /**
     * Render day distribution chart
     */
    private renderDayChart(viewingByDay: number[]): void {
        const container = document.getElementById('wrapped-day-chart');
        if (!container) return;

        const maxValue = Math.max(...viewingByDay, 1);

        const bars = viewingByDay.map((value, day) => {
            const width = (value / maxValue) * 100;
            return `
                <div class="day-bar-row">
                    <span class="day-label">${DAY_NAMES[day].slice(0, 3)}</span>
                    <div class="day-bar" style="width: ${width}%"></div>
                    <span class="day-value">${value}</span>
                </div>
            `;
        }).join('');

        container.innerHTML = bars;
    }

    /**
     * Render monthly timeline
     */
    private renderTimeline(trends: WrappedMonthly[]): void {
        const container = document.getElementById('wrapped-timeline-content');
        if (!container) return;

        if (!trends || trends.length === 0) {
            container.innerHTML = '<p class="empty-state">No monthly data available</p>';
            return;
        }

        const maxWatchTime = Math.max(...trends.map(t => t.watch_time_hours), 1);

        const html = trends.map(month => {
            const barWidth = (month.watch_time_hours / maxWatchTime) * 100;
            return `
                <div class="timeline-month">
                    <div class="timeline-month-header">
                        <span class="timeline-month-name">${month.month_name}</span>
                        <span class="timeline-month-stats">${this.formatWatchTime(month.watch_time_hours)}</span>
                    </div>
                    <div class="timeline-bar-container">
                        <div class="timeline-bar" style="width: ${barWidth}%"></div>
                    </div>
                    <div class="timeline-month-details">
                        <span>${month.playback_count} plays</span>
                        <span>${month.unique_content} unique</span>
                        ${month.top_content ? `<span>Top: ${this.escapeHtml(month.top_content)}</span>` : ''}
                    </div>
                </div>
            `;
        }).join('');

        container.innerHTML = html;
    }

    /**
     * Render achievements
     */
    private renderAchievements(achievements: WrappedAchievement[]): void {
        const container = document.getElementById('wrapped-achievements-content');
        if (!container) return;

        if (!achievements || achievements.length === 0) {
            container.innerHTML = '<p class="empty-state">No achievements yet. Keep watching!</p>';
            return;
        }

        const html = achievements.map(achievement => `
            <div class="achievement-card ${achievement.tier || 'bronze'}">
                <div class="achievement-icon">
                    <i data-feather="${ACHIEVEMENT_ICONS[achievement.id] || 'award'}"></i>
                </div>
                <div class="achievement-info">
                    <div class="achievement-name">${this.escapeHtml(achievement.name)}</div>
                    <div class="achievement-description">${this.escapeHtml(achievement.description)}</div>
                </div>
                ${achievement.tier ? `<div class="achievement-tier ${achievement.tier}">${achievement.tier}</div>` : ''}
            </div>
        `).join('');

        container.innerHTML = html;

        // Reinitialize feather icons if available
        if (typeof window !== 'undefined' && (window as unknown as { feather?: { replace: () => void } }).feather) {
            (window as unknown as { feather: { replace: () => void } }).feather.replace();
        }
    }

    /**
     * Render share card preview
     */
    private renderShareCard(report: WrappedReport): void {
        const container = document.getElementById('wrapped-share-card');
        if (!container) return;

        const html = `
            <div class="share-card-preview">
                <div class="share-card-header">
                    <h3>${this.escapeHtml(report.username)}'s ${report.year} Wrapped</h3>
                </div>
                <div class="share-card-stats">
                    <div class="share-stat">
                        <div class="share-stat-value">${this.formatWatchTime(report.total_watch_time_hours)}</div>
                        <div class="share-stat-label">watched</div>
                    </div>
                    <div class="share-stat">
                        <div class="share-stat-value">${report.unique_content_count}</div>
                        <div class="share-stat-label">titles</div>
                    </div>
                    <div class="share-stat">
                        <div class="share-stat-value">${report.total_playbacks}</div>
                        <div class="share-stat-label">plays</div>
                    </div>
                </div>
                <div class="share-card-top-content">
                    ${report.top_shows.length > 0 ? `<p>Top Show: ${this.escapeHtml(report.top_shows[0].title)}</p>` : ''}
                    ${report.top_movies.length > 0 ? `<p>Top Movie: ${this.escapeHtml(report.top_movies[0].title)}</p>` : ''}
                </div>
                <div class="share-card-footer">
                    <p>${report.shareable_text || ''}</p>
                </div>
            </div>
        `;

        container.innerHTML = html;
    }

    /**
     * Start the reveal animation sequence
     */
    private startRevealAnimation(): void {
        const cards = document.querySelectorAll('.wrapped-card[data-animate="true"]');
        cards.forEach((card, index) => {
            setTimeout(() => {
                card.classList.add('revealed');
            }, index * 200);
        });
    }

    /**
     * Handle generate reports button click
     */
    private async handleGenerateReports(): Promise<void> {
        try {
            this.toastManager?.info('Generating wrapped reports...', 'Wrapped');

            const result = await this.api.generateWrappedReports({
                year: this.currentYear,
                force: false,
            });

            this.toastManager?.success(
                `Generated ${result.reports_generated} reports in ${result.duration_ms}ms`,
                'Wrapped'
            );

            // Reload data
            await this.loadWrappedData();
        } catch (error) {
            logger.error('Failed to generate reports:', error);
            this.toastManager?.error('Failed to generate reports', 'Error');
        }
    }

    /**
     * Handle share button click
     */
    private handleShare(): void {
        if (!this.currentReport?.share_token) {
            this.toastManager?.warning('Share token not available', 'Share');
            return;
        }

        const shareUrl = `${window.location.origin}/wrapped/${this.currentReport.share_token}`;

        if (navigator.clipboard) {
            navigator.clipboard.writeText(shareUrl).then(() => {
                this.toastManager?.success('Share link copied to clipboard!', 'Share');
            }).catch(() => {
                this.toastManager?.error('Failed to copy link', 'Error');
            });
        } else {
            this.toastManager?.info(`Share URL: ${shareUrl}`, 'Share');
        }
    }

    /**
     * Handle export button click
     */
    private handleExport(): void {
        const shareCard = document.getElementById('wrapped-share-card');
        if (!shareCard) {
            this.toastManager?.error('Share card not found', 'Export');
            return;
        }

        // Note: Full image export would require html2canvas or similar library
        // For now, just notify the user
        this.toastManager?.info('Image export requires the html2canvas library', 'Export');
    }

    /**
     * Show loading state
     */
    private showLoading(show: boolean): void {
        const loading = document.getElementById('wrapped-loading');
        const content = document.getElementById('wrapped-content');

        if (loading) {
            loading.style.display = show ? 'flex' : 'none';
        }
        if (content) {
            content.style.display = show ? 'none' : 'block';
        }
    }

    /**
     * Show error message
     */
    private showError(message: string): void {
        this.toastManager?.error(message, 'Wrapped Error');
    }

    /**
     * Show no report message
     */
    private showNoReportMessage(): void {
        const container = document.getElementById('wrapped-user-content');
        if (container) {
            container.innerHTML = `
                <div class="empty-state">
                    <h3>No Wrapped Report Available</h3>
                    <p>We don't have enough data to generate your ${this.currentYear} wrapped report yet.</p>
                    <p>Keep watching and check back later!</p>
                </div>
            `;
        }
    }

    // ========================================================================
    // Utility Methods
    // ========================================================================

    /**
     * Set element text content safely
     */
    private setElementText(id: string, text: string): void {
        const el = document.getElementById(id);
        if (el) {
            el.textContent = text;
        }
    }

    /**
     * Format watch time for display
     */
    private formatWatchTime(hours: number): string {
        if (hours < 1) {
            return `${Math.round(hours * 60)}m`;
        }
        if (hours < 24) {
            return `${hours.toFixed(1)}h`;
        }
        const days = Math.floor(hours / 24);
        const remainingHours = Math.round(hours % 24);
        if (remainingHours === 0) {
            return `${days}d`;
        }
        return `${days}d ${remainingHours}h`;
    }

    /**
     * Format hour for display
     */
    private formatHour(hour: number): string {
        if (hour === 0) return '12 AM';
        if (hour === 12) return '12 PM';
        if (hour < 12) return `${hour} AM`;
        return `${hour - 12} PM`;
    }

    /**
     * Escape HTML to prevent XSS
     */
    private escapeHtml(text: string): string {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}
