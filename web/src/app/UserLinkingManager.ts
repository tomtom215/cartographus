// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * UserLinkingManager - User Identity Linking Across Platforms
 *
 * Manages user identity linking between Plex, Jellyfin, and Emby platforms.
 * Features:
 * - View all users with their linked identities
 * - Create/delete manual user links
 * - View suggested links based on email matching
 * - Display link confidence scores
 * - Bulk linking interface
 */

import type { API } from '../lib/api';
import type {
  UserLinkRequest,
  LinkedUserInfo,
  SuggestedLinksResponse,
  CrossPlatformSummary
} from '../lib/types/cross-platform';
import { escapeHtml } from '../lib/sanitize';
import {
  type Platform,
  isValidPlatform,
  renderConfidenceIndicatorHTML,
  UserSelector,
  linkedUserToSelectorItem,
  type UserSelectorItem
} from '../lib/components';
import { createLogger } from '../lib/logger';

const logger = createLogger('UserLinkingManager');

/**
 * UserLinkingManager class
 */
export class UserLinkingManager {
  private api: API;
  private container: HTMLElement | null = null;
  private allUsers: UserSelectorItem[] = [];
  private suggestions: SuggestedLinksResponse['suggestions'] = {};
  private selectedUserId: number | null = null;
  private linkedUsers: LinkedUserInfo[] = [];
  private isLoading = false;

  // UI components
  private userSelector: UserSelector | null = null;

  // Event handler references for cleanup
  private refreshBtnHandler: (() => void) | null = null;
  private createLinkBtnHandler: (() => void) | null = null;
  private viewSuggestionsBtnHandler: (() => void) | null = null;

  /** Get currently selected user ID (for external access) */
  getSelectedUserId(): number | null { return this.selectedUserId; }
  /** Check if create link handler is set (for testing) */
  hasCreateLinkHandler(): boolean { return this.createLinkBtnHandler !== null; }
  private modalCloseHandler: ((e: MouseEvent) => void) | null = null;
  private keydownHandler: ((e: KeyboardEvent) => void) | null = null;

  constructor(api: API) {
    this.api = api;
  }

  /**
   * Initialize the user linking panel
   */
  async init(containerId: string): Promise<void> {
    this.container = document.getElementById(containerId);
    if (!this.container) {
      logger.warn('Container not found', { containerId });
      return;
    }

    logger.debug('Initializing');
    this.render();
    await this.loadData();
  }

  /**
   * Render the main user linking UI
   */
  private render(): void {
    if (!this.container) return;

    this.container.innerHTML = this.buildMainPanelHTML();
    this.setupEventHandlers();
  }

  /**
   * Build main panel HTML
   */
  private buildMainPanelHTML(): string {
    return `
      <div class="user-linking-panel">
        <!-- Toolbar -->
        <div class="content-mapping-toolbar">
          <div class="content-mapping-search" id="user-selector-container">
            <!-- User selector will be inserted here -->
          </div>
          <div class="content-mapping-actions">
            <button type="button" id="user-linking-refresh-btn" class="btn btn-secondary">
              Refresh
            </button>
            <button type="button" id="user-linking-suggestions-btn" class="btn btn-primary">
              View Suggestions
            </button>
          </div>
        </div>

        <!-- Stats bar -->
        <div class="content-mapping-stats" id="user-linking-stats"></div>

        <!-- Content area -->
        <div class="content-mapping-content" id="user-linking-content">
          <div class="cross-platform-loading">
            <div class="cross-platform-loading-spinner"></div>
          </div>
        </div>

        <!-- Selected user detail (hidden by default) -->
        <div class="content-mapping-detail-panel" id="user-linking-detail" style="display: none;"></div>
      </div>
    `;
  }

  /**
   * Set up event handlers
   */
  private setupEventHandlers(): void {
    // Refresh button
    const refreshBtn = document.getElementById('user-linking-refresh-btn');
    if (refreshBtn) {
      this.refreshBtnHandler = () => this.loadData();
      refreshBtn.addEventListener('click', this.refreshBtnHandler);
    }

    // Suggestions button
    const suggestionsBtn = document.getElementById('user-linking-suggestions-btn');
    if (suggestionsBtn) {
      this.viewSuggestionsBtnHandler = () => this.showSuggestionsPanel();
      suggestionsBtn.addEventListener('click', this.viewSuggestionsBtnHandler);
    }

    // Keyboard handler for modal escape
    this.keydownHandler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        this.closeModal();
      }
    };
    document.addEventListener('keydown', this.keydownHandler);
  }

  /**
   * Load data from API
   */
  private async loadData(): Promise<void> {
    if (this.isLoading) return;

    this.isLoading = true;
    this.showLoading();

    try {
      // Load summary and suggestions in parallel
      const [summaryResponse, suggestionsResponse] = await Promise.all([
        this.api.getCrossPlatformSummary(),
        this.api.getSuggestedUserLinks()
      ]);

      if (summaryResponse.success && summaryResponse.data) {
        this.renderStats(summaryResponse.data);
      }

      if (suggestionsResponse.success && suggestionsResponse.suggestions) {
        this.suggestions = suggestionsResponse.suggestions;
        this.extractUsersFromSuggestions();
      }

      this.renderMainContent();
    } catch (error) {
      logger.error('Failed to load data', { error });
      this.renderError('Failed to load user linking data');
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * Extract users from suggestions for the selector
   */
  private extractUsersFromSuggestions(): void {
    const usersMap = new Map<number, UserSelectorItem>();

    Object.values(this.suggestions).forEach(users => {
      users.forEach(user => {
        if (!usersMap.has(user.internal_user_id)) {
          usersMap.set(user.internal_user_id, linkedUserToSelectorItem(user));
        }
      });
    });

    this.allUsers = Array.from(usersMap.values());

    // Initialize user selector
    const selectorContainer = document.getElementById('user-selector-container');
    if (selectorContainer && this.allUsers.length > 0) {
      this.userSelector = new UserSelector({
        users: this.allUsers,
        placeholder: 'Search for a user...',
        onSelect: (selected) => {
          if (selected.length > 0) {
            this.selectUser(selected[0].id);
          }
        }
      });
      selectorContainer.innerHTML = '';
      selectorContainer.appendChild(this.userSelector.render());
    }
  }

  /**
   * Render stats bar
   */
  private renderStats(summary: CrossPlatformSummary): void {
    const statsContainer = document.getElementById('user-linking-stats');
    if (!statsContainer) return;

    const coveragePercent = this.calculateCoveragePercent(summary.link_coverage);
    const suggestionCount = Object.keys(this.suggestions).length;

    statsContainer.innerHTML = this.buildStatsHTML(summary, coveragePercent, suggestionCount);
  }

  /**
   * Calculate coverage percentage
   */
  private calculateCoveragePercent(coverage: CrossPlatformSummary['link_coverage']): number {
    return coverage ? Math.round(coverage.percentage) : 0;
  }

  /**
   * Build stats HTML
   */
  private buildStatsHTML(summary: CrossPlatformSummary, coveragePercent: number, suggestionCount: number): string {
    const coreStatsHTML = this.buildCoreStatsHTML(summary.total_user_links, coveragePercent, suggestionCount);
    const platformStatsHTML = this.buildPlatformStatsHTML(summary.platforms);

    return `${coreStatsHTML}${platformStatsHTML}`;
  }

  /**
   * Build core stats HTML (links, coverage, suggestions)
   */
  private buildCoreStatsHTML(totalLinks: number, coveragePercent: number, suggestionCount: number): string {
    return `
      <div class="content-mapping-stat">
        <span class="content-mapping-stat-value">${totalLinks}</span>
        <span class="content-mapping-stat-label">User Links</span>
      </div>
      <div class="content-mapping-stat">
        <span class="content-mapping-stat-value">${coveragePercent}%</span>
        <span class="content-mapping-stat-label">Link Coverage</span>
      </div>
      <div class="content-mapping-stat">
        <span class="content-mapping-stat-value">${suggestionCount}</span>
        <span class="content-mapping-stat-label">Pending Suggestions</span>
      </div>
    `;
  }

  /**
   * Build platform stats HTML
   */
  private buildPlatformStatsHTML(platforms: CrossPlatformSummary['platforms']): string {
    if (!platforms) return '';

    return platforms.map(p => `
      <div class="content-mapping-stat">
        <span class="content-mapping-stat-value">${p.users}</span>
        <span class="content-mapping-stat-label">${escapeHtml(p.name)} Users</span>
      </div>
    `).join('');
  }

  /**
   * Render main content
   */
  private renderMainContent(): void {
    const content = document.getElementById('user-linking-content');
    if (!content) return;

    const suggestionCount = Object.keys(this.suggestions).length;

    if (suggestionCount === 0 && this.allUsers.length === 0) {
      this.renderEmpty('No users found. Users will appear here once they have activity on your media servers.');
      return;
    }

    content.innerHTML = this.buildMainContentHTML(suggestionCount);
    this.attachMainContentHandlers();
  }

  /**
   * Build main content HTML
   */
  private buildMainContentHTML(suggestionCount: number): string {
    const suggestionsSection = this.buildSuggestionsSection(suggestionCount);
    const instructionsSection = this.buildInstructionsSection();

    return `
      <div class="user-linking-overview">
        ${suggestionsSection}
        ${instructionsSection}
      </div>
    `;
  }

  /**
   * Build suggestions section HTML
   */
  private buildSuggestionsSection(suggestionCount: number): string {
    if (suggestionCount > 0) {
      const pluralSuffix = suggestionCount !== 1 ? 's' : '';
      return `
        <div class="link-suggestions-preview">
          <h3>Link Suggestions</h3>
          <p>${suggestionCount} email group${pluralSuffix} with potential user links detected.</p>
          <button type="button" class="btn btn-primary" id="view-suggestions-inline">
            Review Suggestions
          </button>
        </div>
      `;
    }

    return `
      <div class="cross-platform-empty">
        <span class="cross-platform-empty-icon" aria-hidden="true">\u{1F465}</span>
        <h3 class="cross-platform-empty-title">No Pending Suggestions</h3>
        <p class="cross-platform-empty-description">
          All suggested user links have been processed. Use the search above to find and link specific users.
        </p>
      </div>
    `;
  }

  /**
   * Build instructions section HTML
   */
  private buildInstructionsSection(): string {
    if (this.allUsers.length === 0) {
      return '';
    }

    return `
      <div class="user-linking-instructions">
        <h4>How User Linking Works</h4>
        <ol>
          <li>Search for a user above to view their linked identities</li>
          <li>Click "Create Link" to manually link two users across platforms</li>
          <li>Review suggestions to quickly link users with matching emails</li>
        </ol>
      </div>
    `;
  }

  /**
   * Attach main content event handlers
   */
  private attachMainContentHandlers(): void {
    const viewSuggestionsInline = document.getElementById('view-suggestions-inline');
    if (viewSuggestionsInline) {
      viewSuggestionsInline.addEventListener('click', () => this.showSuggestionsPanel());
    }
  }

  /**
   * Select a user and show their linked identities
   */
  private async selectUser(userId: number): Promise<void> {
    this.selectedUserId = userId;

    const detailPanel = document.getElementById('user-linking-detail');
    if (!detailPanel) return;

    this.showUserDetailLoading(detailPanel);

    try {
      const linkedUsers = await this.fetchLinkedUsers(userId);
      this.linkedUsers = linkedUsers;
      this.renderUserDetail(userId, linkedUsers);
    } catch (error) {
      logger.error('Failed to load linked users', { error, userId });
      this.showUserDetailError(detailPanel);
    }
  }

  /**
   * Show loading state in user detail panel
   */
  private showUserDetailLoading(detailPanel: HTMLElement): void {
    detailPanel.innerHTML = `
      <div class="cross-platform-loading">
        <div class="cross-platform-loading-spinner"></div>
      </div>
    `;
    detailPanel.style.display = 'block';
  }

  /**
   * Fetch linked users from API
   */
  private async fetchLinkedUsers(userId: number): Promise<LinkedUserInfo[]> {
    const response = await this.api.getLinkedUsers(userId);
    return (response.success && response.users) ? response.users : [];
  }

  /**
   * Show error state in user detail panel
   */
  private showUserDetailError(detailPanel: HTMLElement): void {
    detailPanel.innerHTML = `
      <div class="cross-platform-empty">
        <span class="cross-platform-empty-icon" aria-hidden="true">\u26A0</span>
        <p>Failed to load user details</p>
      </div>
    `;
  }

  /**
   * Render user detail panel
   */
  private renderUserDetail(userId: number, linkedUsers: LinkedUserInfo[]): void {
    const detailPanel = document.getElementById('user-linking-detail');
    if (!detailPanel) return;

    const primaryUser = this.allUsers.find(u => u.id === userId);
    const displayName = this.getUserDisplayName(primaryUser, userId);

    detailPanel.innerHTML = this.buildUserDetailHTML(userId, displayName, primaryUser, linkedUsers);
    this.attachUserDetailHandlers(userId, detailPanel);
  }

  /**
   * Get user display name with fallback
   */
  private getUserDisplayName(user: UserSelectorItem | undefined, userId: number): string {
    return user?.friendly_name || user?.username || `User ${userId}`;
  }

  /**
   * Build user detail HTML
   */
  private buildUserDetailHTML(
    userId: number,
    displayName: string,
    primaryUser: UserSelectorItem | undefined,
    linkedUsers: LinkedUserInfo[]
  ): string {
    const headerHTML = this.buildUserDetailHeaderHTML(userId, displayName, primaryUser);
    const linkedUsersHTML = this.buildLinkedUsersHTML(userId, linkedUsers);

    return `
      <div class="content-mapping-detail">
        ${headerHTML}
        <div class="content-mapping-section">
          <h3 class="content-mapping-section-title">Linked Identities (${linkedUsers.length})</h3>
          <div class="user-linking-list" id="linked-users-list">
            ${linkedUsersHTML}
          </div>
        </div>
      </div>
    `;
  }

  /**
   * Build user detail header HTML
   */
  private buildUserDetailHeaderHTML(userId: number, displayName: string, primaryUser: UserSelectorItem | undefined): string {
    const sourceHTML = primaryUser?.source ? `<span>Source: ${escapeHtml(primaryUser.source)}</span>` : '';

    return `
      <div class="content-mapping-detail-header">
        <div>
          <h2 class="content-mapping-detail-title">${escapeHtml(displayName)}</h2>
          <div class="content-mapping-detail-meta">
            <span>Internal ID: ${userId}</span>
            ${sourceHTML}
          </div>
        </div>
        <div style="display: flex; gap: 0.5rem;">
          <button type="button" class="btn btn-primary" id="create-link-btn">
            + Create Link
          </button>
          <button type="button" class="btn btn-secondary" id="user-detail-close">
            Close
          </button>
        </div>
      </div>
    `;
  }

  /**
   * Build linked users HTML
   */
  private buildLinkedUsersHTML(userId: number, linkedUsers: LinkedUserInfo[]): string {
    if (linkedUsers.length === 0) {
      return `
        <div class="cross-platform-empty" style="padding: 1rem;">
          <p>No linked identities. Click "Create Link" to link this user to another identity.</p>
        </div>
      `;
    }

    return linkedUsers.map(user => this.renderLinkedUserCard(userId, user)).join('');
  }

  /**
   * Attach user detail event handlers
   */
  private attachUserDetailHandlers(userId: number, detailPanel: HTMLElement): void {
    this.attachCloseHandler();
    this.attachCreateLinkHandler(userId);
    this.attachUnlinkHandlers(userId, detailPanel);
  }

  /**
   * Attach close button handler
   */
  private attachCloseHandler(): void {
    const closeBtn = document.getElementById('user-detail-close');
    if (closeBtn) {
      closeBtn.addEventListener('click', () => this.closeUserDetail());
    }
  }

  /**
   * Attach create link button handler
   */
  private attachCreateLinkHandler(userId: number): void {
    const createLinkBtn = document.getElementById('create-link-btn');
    if (createLinkBtn) {
      createLinkBtn.addEventListener('click', () => this.showCreateLinkDialog(userId));
    }
  }

  /**
   * Attach unlink button handlers
   */
  private attachUnlinkHandlers(userId: number, detailPanel: HTMLElement): void {
    const unlinkBtns = detailPanel.querySelectorAll('[data-action="unlink"]');
    unlinkBtns.forEach(btn => {
      btn.addEventListener('click', async () => {
        const linkedId = this.getLinkedIdFromButton(btn);
        if (linkedId) {
          await this.unlinkUsers(userId, linkedId);
        }
      });
    });
  }

  /**
   * Get linked user ID from unlink button
   */
  private getLinkedIdFromButton(btn: Element): number | null {
    const linkedIdStr = btn.getAttribute('data-linked-id') || '0';
    const linkedId = parseInt(linkedIdStr, 10);
    return linkedId || null;
  }

  /**
   * Render a linked user card
   */
  private renderLinkedUserCard(_primaryId: number, user: LinkedUserInfo): string {
    const displayName = this.getLinkedUserDisplayName(user);
    const platformBadge = this.buildLinkedUserPlatformBadge(user.source);
    const linkTypeHTML = this.buildLinkTypeHTML(user.link_type);

    return `
      <div class="user-link-card ${user.link_type || 'manual'}">
        <div class="user-link-info">
          <div class="user-link-avatar">${escapeHtml(displayName.charAt(0).toUpperCase())}</div>
          <div class="user-link-details">
            <span class="user-link-name">${escapeHtml(displayName)}</span>
            <div class="user-link-meta">
              ${platformBadge}
              <span>ID: ${user.internal_user_id}</span>
              ${linkTypeHTML}
            </div>
          </div>
        </div>
        <div class="user-link-actions">
          <button type="button" class="btn btn-sm btn-danger" data-action="unlink" data-linked-id="${user.internal_user_id}">
            Unlink
          </button>
        </div>
      </div>
    `;
  }

  /**
   * Get display name for a linked user
   */
  private getLinkedUserDisplayName(user: LinkedUserInfo): string {
    return user.friendly_name || user.username || `User ${user.internal_user_id}`;
  }

  /**
   * Build platform badge for linked user
   */
  private buildLinkedUserPlatformBadge(source: string): string {
    const platform = source as Platform;
    if (isValidPlatform(platform)) {
      return `<span class="platform-badge platform-badge--${platform}">${escapeHtml(platform)}</span>`;
    }
    return `<span class="platform-badge">${escapeHtml(source)}</span>`;
  }

  /**
   * Build link type HTML
   */
  private buildLinkTypeHTML(linkType: string | undefined): string {
    return linkType ? `<span>Link type: ${escapeHtml(linkType)}</span>` : '';
  }

  /**
   * Close user detail panel
   */
  private closeUserDetail(): void {
    const detailPanel = document.getElementById('user-linking-detail');
    if (detailPanel) {
      detailPanel.style.display = 'none';
      detailPanel.innerHTML = '';
    }
    this.selectedUserId = null;
    this.linkedUsers = [];

    // Clear user selector
    if (this.userSelector) {
      this.userSelector.clear();
    }
  }

  /**
   * Show create link dialog
   */
  private showCreateLinkDialog(primaryUserId: number): void {
    const primaryUser = this.allUsers.find(u => u.id === primaryUserId);
    const primaryName = this.getUserDisplayName(primaryUser, primaryUserId);
    const availableUsers = this.getAvailableUsersForLinking(primaryUserId);

    const modalContent = this.buildCreateLinkFormHTML(primaryUserId, primaryName);
    const modal = this.createModal('Create User Link', modalContent);

    this.setupCreateLinkDialog(modal, primaryUserId, availableUsers);
  }

  /**
   * Get available users for linking (exclude already linked)
   */
  private getAvailableUsersForLinking(primaryUserId: number): UserSelectorItem[] {
    const linkedIds = new Set(this.linkedUsers.map(u => u.internal_user_id));
    linkedIds.add(primaryUserId);
    return this.allUsers.filter(u => !linkedIds.has(u.id));
  }

  /**
   * Build create link form HTML
   */
  private buildCreateLinkFormHTML(primaryUserId: number, primaryName: string): string {
    return `
      <form id="create-link-form" class="cross-platform-modal-form">
        <div class="cross-platform-modal-form-group">
          <label class="cross-platform-modal-form-label">Primary User</label>
          <div class="form-input" style="background: var(--surface-secondary);">
            ${escapeHtml(primaryName)} (ID: ${primaryUserId})
          </div>
        </div>

        <div class="cross-platform-modal-form-group">
          <label class="cross-platform-modal-form-label">Link To User *</label>
          <div id="link-user-selector"></div>
          <span class="cross-platform-modal-form-hint">
            Select the user identity to link to ${escapeHtml(primaryName)}
          </span>
        </div>

        <div class="cross-platform-modal-form-group">
          <label class="cross-platform-modal-form-label">Link Type</label>
          <select id="link-type" class="form-select">
            <option value="manual">Manual</option>
            <option value="email">Email Match</option>
            <option value="plex_home">Plex Home</option>
          </select>
          <span class="cross-platform-modal-form-hint">
            Manual: Manually confirmed by admin<br/>
            Email: Users share the same email address<br/>
            Plex Home: Users are in the same Plex Home
          </span>
        </div>

        <div class="modal-actions">
          <button type="button" class="btn btn-secondary" data-action="cancel">Cancel</button>
          <button type="submit" class="btn btn-primary" id="create-link-submit" disabled>Create Link</button>
        </div>
      </form>
    `;
  }

  /**
   * Setup create link dialog interactions
   */
  private setupCreateLinkDialog(
    modal: HTMLElement,
    primaryUserId: number,
    availableUsers: UserSelectorItem[]
  ): void {
    const selectorContainer = modal.querySelector('#link-user-selector');
    const submitBtn = modal.querySelector('#create-link-submit') as HTMLButtonElement;

    let selectedLinkedUser: UserSelectorItem | null = null;

    if (selectorContainer) {
      this.renderUserSelectorOrEmpty(
        selectorContainer,
        availableUsers,
        primaryUserId,
        (selected) => {
          selectedLinkedUser = selected;
          if (submitBtn) {
            submitBtn.disabled = !selectedLinkedUser;
          }
        }
      );
    }

    this.attachFormSubmitHandler(modal, primaryUserId, () => selectedLinkedUser);
    this.attachCancelHandler(modal);
  }

  /**
   * Render user selector or empty state
   */
  private renderUserSelectorOrEmpty(
    container: Element,
    availableUsers: UserSelectorItem[],
    primaryUserId: number,
    onSelect: (selected: UserSelectorItem | null) => void
  ): void {
    if (availableUsers.length === 0) {
      container.innerHTML = `
        <div class="cross-platform-empty" style="padding: 1rem;">
          <p>No available users to link. All users are already linked.</p>
        </div>
      `;
      return;
    }

    const linkedIds = new Set(this.linkedUsers.map(u => u.internal_user_id));
    linkedIds.add(primaryUserId);

    const selector = new UserSelector({
      users: availableUsers,
      placeholder: 'Search for user to link...',
      excludeIds: Array.from(linkedIds),
      onSelect: (selected) => {
        onSelect(selected.length > 0 ? selected[0] : null);
      }
    });
    container.appendChild(selector.render());
  }

  /**
   * Attach form submit handler
   */
  private attachFormSubmitHandler(
    modal: HTMLElement,
    primaryUserId: number,
    getSelectedUser: () => UserSelectorItem | null
  ): void {
    const form = modal.querySelector('#create-link-form') as HTMLFormElement;
    if (!form) return;

    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      const selectedLinkedUser = getSelectedUser();
      if (!selectedLinkedUser) return;

      const linkTypeSelect = form.querySelector('#link-type') as HTMLSelectElement;
      const linkType = linkTypeSelect.value as UserLinkRequest['link_type'];

      await this.createUserLink(primaryUserId, selectedLinkedUser.id, linkType);
    });
  }

  /**
   * Attach cancel button handler
   */
  private attachCancelHandler(modal: HTMLElement): void {
    const cancelBtn = modal.querySelector('[data-action="cancel"]');
    if (cancelBtn) {
      cancelBtn.addEventListener('click', () => this.closeModal());
    }
  }

  /**
   * Create a user link
   */
  private async createUserLink(
    primaryUserId: number,
    linkedUserId: number,
    linkType: UserLinkRequest['link_type']
  ): Promise<void> {
    try {
      const response = await this.api.createUserLink({
        primary_user_id: primaryUserId,
        linked_user_id: linkedUserId,
        link_type: linkType
      });

      if (response.success) {
        this.closeModal();
        this.showToast('User link created successfully', 'success');
        // Refresh the detail view
        await this.selectUser(primaryUserId);
        await this.loadData();
      } else {
        this.showToast(response.message || 'Failed to create link', 'error');
      }
    } catch (error) {
      logger.error('Create link failed', { error, primaryUserId, linkedUserId, linkType });
      this.showToast('Failed to create user link', 'error');
    }
  }

  /**
   * Unlink users
   */
  private async unlinkUsers(primaryUserId: number, linkedUserId: number): Promise<void> {
    if (!confirm('Are you sure you want to unlink these users?')) {
      return;
    }

    try {
      await this.api.deleteUserLink(primaryUserId, linkedUserId);
      this.showToast('Users unlinked successfully', 'success');
      await this.selectUser(primaryUserId);
      await this.loadData();
    } catch (error) {
      logger.error('Unlink failed', { error, primaryUserId, linkedUserId });
      this.showToast('Failed to unlink users', 'error');
    }
  }

  /**
   * Show suggestions panel
   */
  private showSuggestionsPanel(): void {
    const suggestionCount = Object.keys(this.suggestions).length;

    if (suggestionCount === 0) {
      this.showToast('No link suggestions available', 'info');
      return;
    }

    const modalContent = this.buildSuggestionsModalHTML();
    const modal = this.createModal('Link Suggestions', modalContent);

    this.populateSuggestionsList(modal);
    this.attachSuggestionsModalHandlers(modal);
  }

  /**
   * Build suggestions modal HTML
   */
  private buildSuggestionsModalHTML(): string {
    return `
      <div class="link-suggestions">
        <p style="margin-bottom: 1rem;">
          The following user groups share the same email address and may be the same person
          across different platforms. Review and approve links to combine their analytics.
        </p>
        <div id="suggestions-list"></div>
        <div class="modal-actions">
          <button type="button" class="btn btn-secondary" data-action="cancel">Close</button>
        </div>
      </div>
    `;
  }

  /**
   * Populate suggestions list
   */
  private populateSuggestionsList(modal: HTMLElement): void {
    const suggestionsList = modal.querySelector('#suggestions-list');
    if (!suggestionsList) return;

    Object.entries(this.suggestions).forEach(([email, users]) => {
      if (users.length > 1) {
        const group = this.renderSuggestionGroup(email, users);
        suggestionsList.appendChild(group);
      }
    });
  }

  /**
   * Attach suggestions modal handlers
   */
  private attachSuggestionsModalHandlers(modal: HTMLElement): void {
    const cancelBtn = modal.querySelector('[data-action="cancel"]');
    if (cancelBtn) {
      cancelBtn.addEventListener('click', () => this.closeModal());
    }
  }

  /**
   * Render a suggestion group
   */
  private renderSuggestionGroup(email: string, users: LinkedUserInfo[]): HTMLElement {
    const group = document.createElement('div');
    group.className = 'link-suggestion-group';
    group.innerHTML = this.buildSuggestionGroupHTML(email, users);
    this.attachSuggestionGroupHandlers(group, email, users);
    return group;
  }

  /**
   * Build suggestion group HTML
   */
  private buildSuggestionGroupHTML(email: string, users: LinkedUserInfo[]): string {
    const headerHTML = this.buildSuggestionHeaderHTML(email);
    const usersHTML = this.buildSuggestionUsersHTML(users);
    const explanationHTML = this.buildSuggestionExplanationHTML(email, users.length);

    return `
      ${headerHTML}
      <div class="link-suggestion-users">${usersHTML}</div>
      ${explanationHTML}
    `;
  }

  /**
   * Build suggestion header HTML
   */
  private buildSuggestionHeaderHTML(email: string): string {
    const confidence = 0.95; // High confidence for email match

    return `
      <div class="link-suggestion-header">
        <div class="link-suggestion-email">
          <span class="link-suggestion-email-icon" aria-hidden="true">\u2709</span>
          <span>${escapeHtml(email)}</span>
        </div>
        <div class="link-suggestion-actions">
          ${renderConfidenceIndicatorHTML(confidence, { showLabel: true, label: 'High' })}
          <button type="button" class="btn btn-sm btn-primary" data-action="accept-all">
            Link All
          </button>
          <button type="button" class="btn btn-sm btn-secondary" data-action="dismiss">
            Dismiss
          </button>
        </div>
      </div>
    `;
  }

  /**
   * Build suggestion users HTML
   */
  private buildSuggestionUsersHTML(users: LinkedUserInfo[]): string {
    return users.map(user => this.buildSuggestionUserCardHTML(user)).join('');
  }

  /**
   * Build single suggestion user card HTML
   */
  private buildSuggestionUserCardHTML(user: LinkedUserInfo): string {
    const displayName = user.friendly_name || user.username || `User ${user.internal_user_id}`;
    const platformBadge = this.buildPlatformBadge(user.source);

    return `
      <div class="link-suggestion-user">
        <div class="link-suggestion-user-info">
          ${platformBadge}
          <span class="link-suggestion-user-name">${escapeHtml(displayName)}</span>
          <span class="link-suggestion-user-server">${escapeHtml(user.server_id)}</span>
        </div>
      </div>
    `;
  }

  /**
   * Build platform badge HTML
   */
  private buildPlatformBadge(source: string): string {
    const platform = source as Platform;
    if (isValidPlatform(platform)) {
      return `<span class="platform-badge platform-badge--${platform}">${escapeHtml(platform)}</span>`;
    }
    return '';
  }

  /**
   * Build suggestion explanation HTML
   */
  private buildSuggestionExplanationHTML(email: string, userCount: number): string {
    return `
      <div class="link-suggestion-explanation">
        <strong>Why this suggestion?</strong> These ${userCount} user accounts share the email address
        "${escapeHtml(email)}". Linking them will combine their watch history and analytics into a single unified view.
      </div>
    `;
  }

  /**
   * Attach suggestion group event handlers
   */
  private attachSuggestionGroupHandlers(group: HTMLElement, email: string, users: LinkedUserInfo[]): void {
    this.attachAcceptAllHandler(group, email, users);
    this.attachDismissHandler(group, email);
  }

  /**
   * Attach accept all handler
   */
  private attachAcceptAllHandler(group: HTMLElement, email: string, users: LinkedUserInfo[]): void {
    const acceptBtn = group.querySelector('[data-action="accept-all"]');
    if (acceptBtn) {
      acceptBtn.addEventListener('click', async () => {
        await this.acceptSuggestion(email, users);
        group.remove();
      });
    }
  }

  /**
   * Attach dismiss handler
   */
  private attachDismissHandler(group: HTMLElement, email: string): void {
    const dismissBtn = group.querySelector('[data-action="dismiss"]');
    if (dismissBtn) {
      dismissBtn.addEventListener('click', () => {
        group.remove();
        delete this.suggestions[email];
        this.showToast('Suggestion dismissed', 'info');
      });
    }
  }

  /**
   * Accept a suggestion and link all users
   */
  private async acceptSuggestion(email: string, users: LinkedUserInfo[]): Promise<void> {
    if (users.length < 2) return;

    const primaryUser = users[0];
    const secondaryUsers = users.slice(1);

    const successCount = await this.linkUsersToPrimary(primaryUser, secondaryUsers);

    delete this.suggestions[email];
    this.showToast(`Linked ${successCount} users successfully`, 'success');
    await this.loadData();
  }

  /**
   * Link multiple users to a primary user
   */
  private async linkUsersToPrimary(primaryUser: LinkedUserInfo, secondaryUsers: LinkedUserInfo[]): Promise<number> {
    let successCount = 0;

    for (const user of secondaryUsers) {
      const success = await this.tryLinkUser(primaryUser.internal_user_id, user.internal_user_id);
      if (success) {
        successCount++;
      }
    }

    return successCount;
  }

  /**
   * Try to link a single user (returns true on success)
   */
  private async tryLinkUser(primaryUserId: number, linkedUserId: number): Promise<boolean> {
    try {
      const response = await this.api.createUserLink({
        primary_user_id: primaryUserId,
        linked_user_id: linkedUserId,
        link_type: 'email'
      });

      return response.success;
    } catch (error) {
      logger.error('Failed to link user', { error, userId: linkedUserId });
      return false;
    }
  }

  /**
   * Create a modal dialog
   */
  private createModal(title: string, content: string): HTMLElement {
    this.closeModal();

    const modal = this.buildModalElement(title, content);
    this.attachModalHandlers(modal);
    document.body.appendChild(modal);

    return modal;
  }

  /**
   * Build modal element
   */
  private buildModalElement(title: string, content: string): HTMLElement {
    const modal = document.createElement('div');
    modal.className = 'modal-overlay';
    modal.id = 'user-linking-modal';
    modal.innerHTML = `
      <div class="modal cross-platform-modal-content" style="max-width: 600px;">
        <div class="modal-header">
          <h3 class="modal-title">${escapeHtml(title)}</h3>
          <button type="button" class="modal-close" aria-label="Close">&times;</button>
        </div>
        <div class="modal-body" style="max-height: 70vh; overflow-y: auto;">
          ${content}
        </div>
      </div>
    `;
    return modal;
  }

  /**
   * Attach modal event handlers
   */
  private attachModalHandlers(modal: HTMLElement): void {
    this.attachModalCloseButtonHandler(modal);
    this.attachModalOverlayHandler(modal);
  }

  /**
   * Attach modal close button handler
   */
  private attachModalCloseButtonHandler(modal: HTMLElement): void {
    const closeBtn = modal.querySelector('.modal-close');
    if (closeBtn) {
      closeBtn.addEventListener('click', () => this.closeModal());
    }
  }

  /**
   * Attach modal overlay click handler (click outside to close)
   */
  private attachModalOverlayHandler(modal: HTMLElement): void {
    this.modalCloseHandler = (e: MouseEvent) => {
      if (e.target === modal) {
        this.closeModal();
      }
    };
    modal.addEventListener('click', this.modalCloseHandler);
  }

  /**
   * Close modal dialog
   */
  private closeModal(): void {
    const modal = document.getElementById('user-linking-modal');
    if (modal) {
      if (this.modalCloseHandler) {
        modal.removeEventListener('click', this.modalCloseHandler);
        this.modalCloseHandler = null;
      }
      modal.remove();
    }
  }

  /**
   * Show loading state
   */
  private showLoading(): void {
    const content = document.getElementById('user-linking-content');
    if (content) {
      content.innerHTML = `
        <div class="cross-platform-loading">
          <div class="cross-platform-loading-spinner"></div>
        </div>
      `;
    }
  }

  /**
   * Render empty state
   */
  private renderEmpty(message: string): void {
    const content = document.getElementById('user-linking-content');
    if (content) {
      content.innerHTML = `
        <div class="cross-platform-empty">
          <span class="cross-platform-empty-icon" aria-hidden="true">\u{1F465}</span>
          <h3 class="cross-platform-empty-title">No Users</h3>
          <p class="cross-platform-empty-description">${escapeHtml(message)}</p>
        </div>
      `;
    }
  }

  /**
   * Render error state
   */
  private renderError(message: string): void {
    const content = document.getElementById('user-linking-content');
    if (content) {
      content.innerHTML = `
        <div class="cross-platform-empty">
          <span class="cross-platform-empty-icon" aria-hidden="true">\u26A0</span>
          <h3 class="cross-platform-empty-title">Error</h3>
          <p class="cross-platform-empty-description">${escapeHtml(message)}</p>
          <button type="button" class="btn btn-primary" id="error-retry-btn">
            Retry
          </button>
        </div>
      `;

      const retryBtn = document.getElementById('error-retry-btn');
      if (retryBtn) {
        retryBtn.addEventListener('click', () => this.loadData());
      }
    }
  }

  /**
   * Show toast notification
   */
  private showToast(message: string, type: 'success' | 'error' | 'info' = 'info'): void {
    const event = new CustomEvent('show-toast', {
      detail: { message, type }
    });
    window.dispatchEvent(event);
    logger.debug('Toast shown', { message, type });
  }

  /**
   * Destroy the manager and cleanup resources
   */
  destroy(): void {
    // Remove event handlers
    const refreshBtn = document.getElementById('user-linking-refresh-btn');
    if (refreshBtn && this.refreshBtnHandler) {
      refreshBtn.removeEventListener('click', this.refreshBtnHandler);
    }

    const suggestionsBtn = document.getElementById('user-linking-suggestions-btn');
    if (suggestionsBtn && this.viewSuggestionsBtnHandler) {
      suggestionsBtn.removeEventListener('click', this.viewSuggestionsBtnHandler);
    }

    if (this.keydownHandler) {
      document.removeEventListener('keydown', this.keydownHandler);
    }

    // Cleanup user selector
    if (this.userSelector) {
      this.userSelector.destroy();
    }

    // Close any open modal
    this.closeModal();

    // Clear references
    this.refreshBtnHandler = null;
    this.createLinkBtnHandler = null;
    this.viewSuggestionsBtnHandler = null;
    this.keydownHandler = null;
    this.container = null;
    this.userSelector = null;

    logger.debug('Destroyed');
  }
}

export default UserLinkingManager;
