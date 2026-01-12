// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * QuickInsightsManager - Displays binge watching and watch party summaries
 *
 * Features:
 * - Fetches binge watching analytics data
 * - Fetches watch party analytics data
 * - Displays summary metrics on Overview page
 * - Provides navigation to detailed views
 */

import type { API, LocationFilter } from '../lib/api';
import type { BingeAnalyticsResponse, WatchPartyAnalyticsResponse, WatchPartyContentStats } from '../lib/types/analytics';
import { createLogger } from '../lib/logger';

const logger = createLogger('QuickInsightsManager');

export class QuickInsightsManager {
  private api: API;
  private isLoading: boolean = false;

  constructor(api: API) {
    this.api = api;
  }

  /**
   * Initialize the quick insights manager
   */
  init(): void {
    this.setupNavigationLinks();
    logger.debug('QuickInsightsManager initialized');
  }

  /**
   * Set up navigation links to detailed views
   */
  private setupNavigationLinks(): void {
    const links = document.querySelectorAll('[data-navigate-analytics]');
    links.forEach(link => {
      link.addEventListener('click', (e) => {
        e.preventDefault();
        const targetPage = (e.currentTarget as HTMLElement).getAttribute('data-navigate-analytics');
        if (targetPage) {
          // Dispatch custom event to navigate to analytics page
          const event = new CustomEvent('navigate-analytics', { detail: { page: targetPage } });
          document.dispatchEvent(event);
        }
      });
    });
  }

  /**
   * Load quick insights data
   */
  async loadInsights(filter: LocationFilter = {}): Promise<void> {
    if (this.isLoading) return;

    this.isLoading = true;

    try {
      // Fetch binge and watch party data in parallel
      const [bingeData, watchPartyData] = await Promise.all([
        this.fetchBingeAnalytics(filter),
        this.fetchWatchPartyAnalytics(filter)
      ]);

      // Update UI with the data
      this.updateBingeCard(bingeData);
      this.updateWatchPartyCard(watchPartyData);
    } catch (error) {
      logger.error('Failed to load quick insights:', error);
      this.showError();
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * Fetch binge analytics data
   */
  private async fetchBingeAnalytics(filter: LocationFilter): Promise<BingeAnalyticsResponse | null> {
    try {
      const response = await this.api.getAnalyticsBinge(filter);
      return response;
    } catch (error) {
      logger.warn('Binge analytics fetch failed:', error);
      return null;
    }
  }

  /**
   * Fetch watch party analytics data
   */
  private async fetchWatchPartyAnalytics(filter: LocationFilter): Promise<WatchPartyAnalyticsResponse | null> {
    try {
      const response = await this.api.getAnalyticsWatchParties(filter);
      return response;
    } catch (error) {
      logger.warn('Watch party analytics fetch failed:', error);
      return null;
    }
  }

  /**
   * Update binge watching card with data
   */
  private updateBingeCard(data: BingeAnalyticsResponse | null): void {
    const sessionsEl = document.getElementById('binge-sessions-count');
    const avgEpisodesEl = document.getElementById('binge-avg-episodes');
    const topShowEl = document.getElementById('binge-top-show');

    if (!data) {
      if (sessionsEl) sessionsEl.textContent = '0';
      if (avgEpisodesEl) avgEpisodesEl.textContent = '0';
      if (topShowEl) topShowEl.textContent = 'N/A';
      return;
    }

    // Update metrics - BingeAnalyticsResponse has flat structure
    if (sessionsEl) {
      sessionsEl.textContent = this.formatNumber(data.total_binge_sessions);
    }

    if (avgEpisodesEl) {
      avgEpisodesEl.textContent = data.avg_episodes_per_binge?.toFixed(1) || '0';
    }

    if (topShowEl) {
      if (data.top_binge_shows && data.top_binge_shows.length > 0) {
        const topShow = data.top_binge_shows[0].show_name;
        // Truncate long titles
        topShowEl.textContent = topShow.length > 15 ? topShow.substring(0, 12) + '...' : topShow;
        topShowEl.title = topShow; // Full title on hover
      } else {
        topShowEl.textContent = 'N/A';
      }
    }
  }

  /**
   * Update watch party card with data
   */
  private updateWatchPartyCard(data: WatchPartyAnalyticsResponse | null): void {
    const countEl = document.getElementById('watch-party-count');
    const avgUsersEl = document.getElementById('watch-party-avg-users');
    const topContentEl = document.getElementById('watch-party-top-content');

    if (!data) {
      if (countEl) countEl.textContent = '0';
      if (avgUsersEl) avgUsersEl.textContent = '0';
      if (topContentEl) topContentEl.textContent = 'N/A';
      return;
    }

    // Update metrics - WatchPartyAnalyticsResponse has flat structure
    if (countEl) {
      countEl.textContent = this.formatNumber(data.total_watch_parties);
    }

    if (avgUsersEl) {
      avgUsersEl.textContent = data.avg_participants_per_party?.toFixed(1) || '0';
    }

    if (topContentEl) {
      if (data.top_content && data.top_content.length > 0) {
        const topContent = this.getTopContentTitle(data.top_content[0]);
        // Truncate long titles
        topContentEl.textContent = topContent.length > 15 ? topContent.substring(0, 12) + '...' : topContent;
        topContentEl.title = topContent; // Full title on hover
      } else {
        topContentEl.textContent = 'N/A';
      }
    }
  }

  /**
   * Get title from WatchPartyContentStats
   */
  private getTopContentTitle(content: WatchPartyContentStats): string {
    // WatchPartyContentStats may have different title field
    return content.title || content.grandparent_title || 'Unknown';
  }

  /**
   * Show error state in cards
   */
  private showError(): void {
    const elements = [
      'binge-sessions-count',
      'binge-avg-episodes',
      'binge-top-show',
      'watch-party-count',
      'watch-party-avg-users',
      'watch-party-top-content'
    ];

    elements.forEach(id => {
      const el = document.getElementById(id);
      if (el) el.textContent = '--';
    });
  }

  /**
   * Format large numbers with K/M suffixes
   */
  private formatNumber(num: number): string {
    if (!num || isNaN(num)) return '0';
    if (num >= 1000000) {
      return (num / 1000000).toFixed(1) + 'M';
    }
    if (num >= 1000) {
      return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
  }

  /**
   * Cleanup resources to prevent memory leaks
   */
  destroy(): void {
    // No persistent state to clean
  }
}

export default QuickInsightsManager;
