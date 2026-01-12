// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * UserProfileSettingsManager - User Profile Management UI
 *
 * Provides a dedicated user profile section in the settings panel with:
 * - User info display (username, avatar placeholder)
 * - Account settings (future password change)
 * - Activity history summary
 * - Data management options
 * - Export user data
 *
 * Note: Password change requires backend API support (not yet implemented)
 */

import type { API, UserProfileAnalytics } from '../lib/api';
import type { ToastManager } from '../lib/toast';
import { createLogger } from '../lib/logger';
import { SessionManagementManager } from './SessionManagementManager';
import { SafeStorage } from '../lib/utils/SafeStorage';

const logger = createLogger('UserProfileSettingsManager');

/**
 * User profile section configuration
 */
export interface ProfileSection {
  /** Section ID */
  id: string;
  /** Section title */
  title: string;
  /** Whether section is expanded by default */
  expanded: boolean;
}

/**
 * User data export format
 */
export type ExportFormat = 'json' | 'csv';

export class UserProfileSettingsManager {
  private api: API;
  private toastManager: ToastManager | null = null;
  private container: HTMLElement | null = null;
  private currentUsername: string | null = null;
  private profileData: UserProfileAnalytics | null = null;
  // AbortController for clean event listener removal
  private abortController: AbortController | null = null;
  // Track active timeouts for cleanup
  private activeTimeouts: Set<ReturnType<typeof setTimeout>> = new Set();
  // Session management manager (ADR-0015)
  private sessionManager: SessionManagementManager | null = null;

  constructor(api: API) {
    this.api = api;
    this.currentUsername = SafeStorage.getItem('auth_username');
    this.sessionManager = new SessionManagementManager(api);
  }

  /**
   * Set toast manager for notifications
   */
  setToastManager(toastManager: ToastManager): void {
    this.toastManager = toastManager;
    this.sessionManager?.setToastManager(toastManager);
  }

  /**
   * Initialize the profile settings section
   */
  async initialize(containerId: string): Promise<void> {
    this.container = document.getElementById(containerId);
    if (!this.container) {
      logger.warn(`Container #${containerId} not found`);
      return;
    }

    this.render();
    await this.loadProfileData();
    this.setupEventListeners();

    // Initialize session management (ADR-0015)
    await this.sessionManager?.initialize('security-sessions-container');

    logger.debug('UserProfileSettingsManager initialized');
  }

  /**
   * Load user profile data
   */
  private async loadProfileData(): Promise<void> {
    if (!this.currentUsername) {
      this.showNoUserState();
      return;
    }

    this.setLoading(true);

    try {
      this.profileData = await this.api.getAnalyticsUserProfile(this.currentUsername);
      this.updateProfileDisplay();
    } catch (error) {
      logger.error('Failed to load profile:', error);
      this.showErrorState();
    } finally {
      this.setLoading(false);
    }
  }

  /**
   * Render the profile settings panel
   */
  private render(): void {
    if (!this.container) return;

    this.container.innerHTML = `
      <div class="profile-settings-panel">
        <!-- Profile Header -->
        <div class="profile-settings-header">
          <div class="profile-avatar">
            <span class="profile-avatar-initial" id="profile-avatar-initial">?</span>
          </div>
          <div class="profile-info">
            <h3 class="profile-name" id="profile-name">Loading...</h3>
            <span class="profile-username" id="profile-username">@...</span>
            <span class="profile-status" id="profile-status">
              <span class="status-dot"></span>
              <span class="status-text">Active</span>
            </span>
          </div>
        </div>

        <!-- Account Section -->
        <div class="profile-section" data-section="account">
          <button class="profile-section-header" aria-expanded="true" aria-controls="account-content">
            <span class="section-icon">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                <circle cx="12" cy="7" r="4"/>
              </svg>
            </span>
            <span class="section-title">Account</span>
            <span class="section-arrow">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="6 9 12 15 18 9"/>
              </svg>
            </span>
          </button>
          <div class="profile-section-content" id="account-content">
            <div class="profile-field">
              <label>Username</label>
              <span class="profile-field-value" id="account-username">-</span>
            </div>
            <div class="profile-field">
              <label>Member Since</label>
              <span class="profile-field-value" id="account-member-since">-</span>
            </div>
            <div class="profile-field">
              <label>Last Active</label>
              <span class="profile-field-value" id="account-last-active">-</span>
            </div>
            <div class="profile-action-note">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <line x1="12" y1="8" x2="12" y2="12"/>
                <line x1="12" y1="16" x2="12.01" y2="16"/>
              </svg>
              <span>Password changes are managed through your Plex account.</span>
            </div>
          </div>
        </div>

        <!-- Activity Summary Section -->
        <div class="profile-section" data-section="activity">
          <button class="profile-section-header" aria-expanded="true" aria-controls="activity-content">
            <span class="section-icon">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
              </svg>
            </span>
            <span class="section-title">Activity Summary</span>
            <span class="section-arrow">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="6 9 12 15 18 9"/>
              </svg>
            </span>
          </button>
          <div class="profile-section-content" id="activity-content">
            <div class="profile-stats-grid">
              <div class="profile-stat">
                <span class="profile-stat-value" id="activity-total-plays">-</span>
                <span class="profile-stat-label">Total Plays</span>
              </div>
              <div class="profile-stat">
                <span class="profile-stat-value" id="activity-watch-time">-</span>
                <span class="profile-stat-label">Watch Time</span>
              </div>
              <div class="profile-stat">
                <span class="profile-stat-value" id="activity-unique-content">-</span>
                <span class="profile-stat-label">Unique Items</span>
              </div>
              <div class="profile-stat">
                <span class="profile-stat-value" id="activity-completion">-</span>
                <span class="profile-stat-label">Avg Completion</span>
              </div>
            </div>
            <div class="profile-activity-details">
              <div class="profile-field">
                <label>Favorite Library</label>
                <span class="profile-field-value" id="activity-favorite-library">-</span>
              </div>
              <div class="profile-field">
                <label>Favorite Platform</label>
                <span class="profile-field-value" id="activity-favorite-platform">-</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Data Management Section -->
        <div class="profile-section" data-section="data">
          <button class="profile-section-header" aria-expanded="false" aria-controls="data-content">
            <span class="section-icon">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
                <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
                <line x1="12" y1="22.08" x2="12" y2="12"/>
              </svg>
            </span>
            <span class="section-title">Data Management</span>
            <span class="section-arrow">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="6 9 12 15 18 9"/>
              </svg>
            </span>
          </button>
          <div class="profile-section-content collapsed" id="data-content">
            <div class="profile-data-actions">
              <button class="profile-action-btn" id="export-history-btn">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                  <polyline points="7 10 12 15 17 10"/>
                  <line x1="12" y1="15" x2="12" y2="3"/>
                </svg>
                <span>Export My Data</span>
              </button>
              <button class="profile-action-btn" id="clear-search-history-btn">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="11" cy="11" r="8"/>
                  <line x1="21" y1="21" x2="16.65" y2="16.65"/>
                  <line x1="8" y1="11" x2="14" y2="11"/>
                </svg>
                <span>Clear Search History</span>
              </button>
              <button class="profile-action-btn profile-action-danger" id="clear-all-data-btn">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3 6 5 6 21 6"/>
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                </svg>
                <span>Clear All Local Data</span>
              </button>
            </div>
            <p class="profile-data-note">
              Clearing local data removes saved preferences, search history, and filter presets.
              Your viewing history on the server is not affected.
            </p>
          </div>
        </div>

        <!-- Security Section (ADR-0015) -->
        <div class="profile-section" data-section="security">
          <button class="profile-section-header" aria-expanded="true" aria-controls="security-content">
            <span class="section-icon">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
              </svg>
            </span>
            <span class="section-title">Security</span>
            <span class="section-arrow">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="6 9 12 15 18 9"/>
              </svg>
            </span>
          </button>
          <div class="profile-section-content" id="security-content">
            <!-- Session Management will be rendered here -->
            <div id="security-sessions-container"></div>
          </div>
        </div>

        <!-- Profile Loading Overlay -->
        <div class="profile-loading-overlay hidden" id="profile-loading">
          <div class="profile-loading-spinner"></div>
        </div>
      </div>
    `;
  }

  /**
   * Setup event listeners with AbortController for clean removal
   */
  private setupEventListeners(): void {
    if (!this.container) return;

    // Create AbortController for cleanup
    this.abortController = new AbortController();
    const signal = this.abortController.signal;

    // Section toggles
    const sectionHeaders = this.container.querySelectorAll('.profile-section-header');
    sectionHeaders.forEach(header => {
      header.addEventListener('click', () => this.toggleSection(header as HTMLElement), { signal });
    });

    // Export button
    const exportBtn = this.container.querySelector('#export-history-btn');
    exportBtn?.addEventListener('click', () => this.exportUserData(), { signal });

    // Clear search history button
    const clearSearchBtn = this.container.querySelector('#clear-search-history-btn');
    clearSearchBtn?.addEventListener('click', () => this.clearSearchHistory(), { signal });

    // Clear all data button
    const clearAllBtn = this.container.querySelector('#clear-all-data-btn');
    clearAllBtn?.addEventListener('click', () => this.confirmClearAllData(), { signal });
  }

  /**
   * Toggle section expansion
   */
  private toggleSection(header: HTMLElement): void {
    const expanded = header.getAttribute('aria-expanded') === 'true';
    const contentId = header.getAttribute('aria-controls');
    const content = document.getElementById(contentId || '');

    if (content) {
      header.setAttribute('aria-expanded', String(!expanded));
      content.classList.toggle('collapsed', expanded);
    }
  }

  /**
   * Update profile display with data
   */
  private updateProfileDisplay(): void {
    if (!this.profileData?.profile) return;

    const profile = this.profileData.profile;

    // Update header
    this.updateElement('profile-name', profile.friendly_name || profile.username);
    this.updateElement('profile-username', `@${profile.username}`);

    const initial = (profile.friendly_name || profile.username || '?').charAt(0).toUpperCase();
    this.updateElement('profile-avatar-initial', initial);

    // Update account section
    this.updateElement('account-username', profile.username);

    if (profile.first_played_date) {
      const memberSince = new Date(profile.first_played_date);
      this.updateElement('account-member-since', memberSince.toLocaleDateString());
    }

    if (profile.last_played_date) {
      const lastActive = new Date(profile.last_played_date);
      this.updateElement('account-last-active', this.formatRelativeTime(lastActive));
    }

    // Update activity section
    this.updateElement('activity-total-plays', profile.total_plays.toLocaleString());

    const watchTimeHours = Math.round(profile.total_watch_time_minutes / 60);
    this.updateElement('activity-watch-time', `${watchTimeHours.toLocaleString()}h`);

    this.updateElement('activity-unique-content', profile.unique_content_count.toLocaleString());
    this.updateElement('activity-completion', `${profile.avg_completion_rate.toFixed(1)}%`);
    this.updateElement('activity-favorite-library', profile.favorite_library || '-');
    this.updateElement('activity-favorite-platform', profile.favorite_platform || '-');
  }

  /**
   * Update element text content
   */
  private updateElement(id: string, value: string): void {
    const element = document.getElementById(id);
    if (element) {
      element.textContent = value;
    }
  }

  /**
   * Format relative time
   */
  private formatRelativeTime(date: Date): string {
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
  }

  /**
   * Set loading state
   */
  private setLoading(loading: boolean): void {
    const overlay = document.getElementById('profile-loading');
    if (overlay) {
      overlay.classList.toggle('hidden', !loading);
    }
  }

  /**
   * Show no user state
   */
  private showNoUserState(): void {
    this.updateElement('profile-name', 'Not logged in');
    this.updateElement('profile-username', '');
    this.updateElement('profile-avatar-initial', '?');
  }

  /**
   * Show error state
   */
  private showErrorState(): void {
    this.updateElement('profile-name', 'Error loading profile');
  }

  /**
   * Export user data
   */
  private async exportUserData(): Promise<void> {
    try {
      const data = {
        exportedAt: new Date().toISOString(),
        username: this.currentUsername,
        profile: this.profileData?.profile || null,
        preferences: this.getStoredPreferences(),
        filterPresets: SafeStorage.getJSON<unknown[]>('filter-presets', []),
      };

      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `cartographus-user-data-${new Date().toISOString().split('T')[0]}.json`;
      a.click();
      URL.revokeObjectURL(url);

      this.toastManager?.success('Data exported successfully');
    } catch (error) {
      logger.error('Export failed:', error);
      this.toastManager?.error('Failed to export data');
    }
  }

  /**
   * Get stored user preferences
   */
  private getStoredPreferences(): Record<string, string | null> {
    return {
      theme: SafeStorage.getItem('theme'),
      colorblindMode: SafeStorage.getItem('colorblind-mode'),
      visualizationMode: SafeStorage.getItem('map-visualization-mode'),
      viewMode: SafeStorage.getItem('view-mode'),
      autoRefresh: SafeStorage.getItem('auto-refresh-enabled'),
    };
  }

  /**
   * Clear search history
   */
  private clearSearchHistory(): void {
    SafeStorage.removeItem('recent-searches');
    SafeStorage.removeItem('recent-filters');
    this.toastManager?.success('Search history cleared');
  }

  /**
   * Confirm and clear all local data
   */
  private confirmClearAllData(): void {
    // Use confirmation dialog if available, otherwise use native confirm
    const confirmed = confirm(
      'Are you sure you want to clear all local data?\n\n' +
      'This will remove:\n' +
      '- Saved preferences\n' +
      '- Filter presets\n' +
      '- Search history\n\n' +
      'Your viewing history on the server will not be affected.'
    );

    if (confirmed) {
      this.clearAllLocalData();
    }
  }

  /**
   * Clear all local data
   */
  private clearAllLocalData(): void {
    const keysToRemove = [
      'theme',
      'colorblind-mode',
      'map-visualization-mode',
      'view-mode',
      'auto-refresh-enabled',
      'auto-refresh-interval',
      'sidebar-collapsed',
      'filter-presets',
      'recent-searches',
      'recent-filters',
      'stats-previous',
      'stats-sparkline-history',
      'onboarding-completed',
      'trendPrediction.enabled',
    ];

    keysToRemove.forEach(key => {
      SafeStorage.removeItem(key);
    });

    this.toastManager?.success('All local data cleared. Reloading...');

    // Reload page to apply defaults
    const timeoutId = setTimeout(() => {
      this.activeTimeouts.delete(timeoutId);
      window.location.reload();
    }, 1500);
    this.activeTimeouts.add(timeoutId);
  }

  /**
   * Refresh profile data
   */
  async refresh(): Promise<void> {
    await this.loadProfileData();
  }

  /**
   * Clean up event listeners and resources
   */
  destroy(): void {
    // Abort all event listeners
    if (this.abortController) {
      this.abortController.abort();
      this.abortController = null;
    }

    // Clear all active timeouts
    for (const timeoutId of this.activeTimeouts) {
      clearTimeout(timeoutId);
    }
    this.activeTimeouts.clear();

    // Clean up session manager
    this.sessionManager?.destroy();
    this.sessionManager = null;

    // Clear references
    this.container = null;
    this.profileData = null;
    this.toastManager = null;
  }
}

// Export singleton factory
let userProfileSettingsManagerInstance: UserProfileSettingsManager | null = null;

export function getUserProfileSettingsManager(api: API): UserProfileSettingsManager {
  if (!userProfileSettingsManagerInstance) {
    userProfileSettingsManagerInstance = new UserProfileSettingsManager(api);
  }
  return userProfileSettingsManagerInstance;
}
