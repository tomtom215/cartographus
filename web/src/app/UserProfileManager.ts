// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * User Profile Analytics Manager
 * Manages the user profile dashboard including:
 * - User selector population
 * - User profile data loading
 * - Stats display updates
 * - Chart rendering coordination
 */

import * as echarts from 'echarts';
import type { API, UserProfileAnalytics, LocationFilter } from '../lib/api';
import { createLogger } from '../lib/logger';
import { UserProfileChartRenderer } from '../lib/charts/renderers/UserProfileChartRenderer';

const logger = createLogger('UserProfile');

export class UserProfileManager {
  private api: API;
  private filterManager: { buildFilter: () => LocationFilter } | null = null;
  private charts: Map<string, echarts.ECharts> = new Map();
  private currentUsername: string | null = null;
  private usersLoaded: boolean = false;

  constructor(api: API) {
    this.api = api;
    this.setupEventListeners();
  }

  /**
   * Initialize the user profile analytics tab (called when tab is first shown)
   */
  async init(): Promise<void> {
    if (!this.usersLoaded) {
      await this.loadUserList();
      this.usersLoaded = true;
    }
    this.initializeCharts();
  }

  /**
   * Set filter manager reference for building filters
   */
  setFilterManager(manager: { buildFilter: () => LocationFilter }): void {
    this.filterManager = manager;
  }

  /**
   * Set up event listeners for user selector
   */
  private setupEventListeners(): void {
    const userSelector = document.getElementById('user-profile-selector') as HTMLSelectElement;
    if (userSelector) {
      userSelector.addEventListener('change', async () => {
        const username = userSelector.value;
        if (username) {
          await this.loadUserProfile(username);
        } else {
          this.showEmptyState();
        }
      });
    }
  }

  /**
   * Initialize charts when user profile tab is first shown
   */
  initializeCharts(): void {
    const chartIds = ['chart-user-activity-trend', 'chart-user-top-content', 'chart-user-platforms'];

    chartIds.forEach(chartId => {
      const container = document.getElementById(chartId);
      if (container && !this.charts.has(chartId)) {
        const chart = echarts.init(container, 'dark');
        this.charts.set(chartId, chart);
      }
    });
  }

  /**
   * Load list of users for selector
   */
  private async loadUserList(): Promise<void> {
    try {
      const users = await this.api.getUsers();
      this.populateUserSelector(users);
    } catch (error) {
      logger.error('Failed to load user list:', error);
      const selector = document.getElementById('user-profile-selector') as HTMLSelectElement;
      if (selector) {
        selector.innerHTML = '<option value="">-- Error loading users --</option>';
      }
    }
  }

  /**
   * Populate the user selector with available users
   */
  private populateUserSelector(users: string[]): void {
    const selector = document.getElementById('user-profile-selector') as HTMLSelectElement;
    if (!selector) return;

    // Clear existing options (keep the placeholder)
    selector.innerHTML = '<option value="">-- Select a user --</option>';

    // Add user options
    users.forEach(username => {
      const option = document.createElement('option');
      option.value = username;
      option.textContent = username;
      selector.appendChild(option);
    });
  }

  /**
   * Load profile analytics for a specific user
   */
  async loadUserProfile(username: string): Promise<void> {
    this.currentUsername = username;
    this.showLoading(true);

    try {
      const filter = this.filterManager?.buildFilter() || {};
      const data = await this.api.getAnalyticsUserProfile(username, filter);

      this.hideEmptyState();
      this.updateProfileInfo(data);
      this.updateStats(data);
      this.renderCharts(data);
    } catch (error) {
      logger.error('Failed to load user profile:', error);
      this.showError('Failed to load user profile. Please try again.');
    } finally {
      this.showLoading(false);
    }
  }

  /**
   * Update the user profile info display
   */
  private updateProfileInfo(data: UserProfileAnalytics): void {
    const profile = data.profile;

    // Update profile header
    this.updateElement('user-profile-name', profile.friendly_name || profile.username);
    this.updateElement('user-profile-username', `@${profile.username}`);

    // Update status badge
    const statusBadge = document.getElementById('user-profile-status');
    if (statusBadge) {
      statusBadge.textContent = profile.is_active ? 'Active' : 'Inactive';
      statusBadge.className = `status-badge ${profile.is_active ? 'active' : 'inactive'}`;
    }

    // Show profile info section
    const profileInfo = document.getElementById('user-profile-info');
    if (profileInfo) {
      profileInfo.style.display = 'flex';
    }
  }

  /**
   * Update the overview stats display
   */
  private updateStats(data: UserProfileAnalytics): void {
    const statsContainer = document.getElementById('user-profile-stats');
    if (statsContainer) {
      statsContainer.style.display = 'grid';
    }

    const profile = data.profile;

    this.updateElement('user-total-plays', profile.total_plays.toLocaleString());

    // Format watch time as hours
    const watchTimeHours = Math.round(profile.total_watch_time_minutes / 60);
    this.updateElement('user-watch-time', `${watchTimeHours.toLocaleString()}h`);

    this.updateElement('user-unique-content', profile.unique_content_count.toLocaleString());
    this.updateElement('user-avg-completion', `${profile.avg_completion_rate.toFixed(1)}%`);
    this.updateElement('user-favorite-library', profile.favorite_library || '-');
    this.updateElement('user-favorite-platform', profile.favorite_platform || '-');

    // Format dates
    if (profile.first_played_date) {
      const firstDate = new Date(profile.first_played_date);
      this.updateElement('user-first-played', firstDate.toLocaleDateString());
    } else {
      this.updateElement('user-first-played', '-');
    }

    if (profile.last_played_date) {
      const lastDate = new Date(profile.last_played_date);
      this.updateElement('user-last-played', lastDate.toLocaleDateString());
    } else {
      this.updateElement('user-last-played', '-');
    }
  }

  /**
   * Render all user profile charts
   */
  private renderCharts(data: UserProfileAnalytics): void {
    this.initializeCharts();

    // Render activity trend chart
    const trendChart = this.charts.get('chart-user-activity-trend');
    if (trendChart) {
      const renderer = new UserProfileChartRenderer({ chartId: 'chart-user-activity-trend', chart: trendChart });
      renderer.renderUserActivityTrend(data);
    }

    // Render top content chart
    const contentChart = this.charts.get('chart-user-top-content');
    if (contentChart) {
      const renderer = new UserProfileChartRenderer({ chartId: 'chart-user-top-content', chart: contentChart });
      renderer.renderUserTopContent(data);
    }

    // Render platforms chart
    const platformsChart = this.charts.get('chart-user-platforms');
    if (platformsChart) {
      const renderer = new UserProfileChartRenderer({ chartId: 'chart-user-platforms', chart: platformsChart });
      renderer.renderUserPlatforms(data);
    }

    // Show chart grid
    const chartGrid = document.querySelector('#analytics-users-profile .chart-grid') as HTMLElement;
    if (chartGrid) {
      chartGrid.style.display = 'grid';
    }
  }

  /**
   * Show empty state when no user is selected
   */
  private showEmptyState(): void {
    const emptyState = document.getElementById('user-profile-empty-state');
    const profileInfo = document.getElementById('user-profile-info');
    const statsContainer = document.getElementById('user-profile-stats');
    const chartGrid = document.querySelector('#analytics-users-profile .chart-grid') as HTMLElement;

    if (emptyState) emptyState.style.display = 'flex';
    if (profileInfo) profileInfo.style.display = 'none';
    if (statsContainer) statsContainer.style.display = 'none';
    if (chartGrid) chartGrid.style.display = 'none';
  }

  /**
   * Hide empty state
   */
  private hideEmptyState(): void {
    const emptyState = document.getElementById('user-profile-empty-state');
    if (emptyState) emptyState.style.display = 'none';
  }

  /**
   * Show loading indicator
   */
  private showLoading(show: boolean): void {
    const loadingEl = document.getElementById('user-profile-loading');
    if (loadingEl) {
      loadingEl.style.display = show ? 'inline' : 'none';
    }
  }

  /**
   * Show error message
   */
  private showError(message: string): void {
    logger.error(message);
    // Could integrate with ToastManager if available
  }

  /**
   * Helper to update element text content
   */
  private updateElement(id: string, value: string): void {
    const el = document.getElementById(id);
    if (el) {
      el.textContent = value;
    }
  }

  /**
   * Resize charts on window resize
   */
  resizeCharts(): void {
    this.charts.forEach(chart => {
      chart.resize();
    });
  }

  /**
   * Dispose charts and clean up
   */
  dispose(): void {
    this.charts.forEach(chart => {
      chart.dispose();
    });
    this.charts.clear();
  }

  /**
   * Refresh data for current user (called when filters change)
   */
  async refresh(): Promise<void> {
    if (this.currentUsername) {
      await this.loadUserProfile(this.currentUsername);
    }
  }

  /**
   * Cleanup resources to prevent memory leaks
   * Disposes charts and clears references
   */
  destroy(): void {
    this.dispose();
    this.filterManager = null;
    this.currentUsername = null;
  }
}
