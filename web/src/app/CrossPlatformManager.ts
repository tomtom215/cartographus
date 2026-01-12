// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * CrossPlatformManager - Cross-Platform Feature Orchestrator
 *
 * Main orchestrator for the cross-platform linking feature.
 * Coordinates ContentMappingManager, UserLinkingManager, and CrossPlatformAnalyticsManager
 * with tabbed navigation.
 *
 * Features:
 * - Tabbed interface for Analytics, Content Mapping, and User Linking
 * - Lazy initialization of sub-managers
 * - Proper cleanup and resource management
 * - Keyboard accessibility support
 */

import type { API } from '../lib/api';
import { ContentMappingManager } from './ContentMappingManager';
import { UserLinkingManager } from './UserLinkingManager';
import { CrossPlatformAnalyticsManager } from './CrossPlatformAnalyticsManager';
import { createLogger } from '../lib/logger';

const logger = createLogger('CrossPlatformManager');

/**
 * Available tabs in the cross-platform view
 */
type CrossPlatformTab = 'analytics' | 'content' | 'users';

/**
 * Tab configuration
 */
interface TabConfig {
  id: CrossPlatformTab;
  label: string;
  icon: string;
  containerId: string;
}

/**
 * CrossPlatformManager class
 */
export class CrossPlatformManager {
  private api: API;
  private container: HTMLElement | null = null;
  private activeTab: CrossPlatformTab = 'analytics';
  private initialized = false;

  // Sub-managers (lazily initialized)
  private contentMappingManager: ContentMappingManager | null = null;
  private userLinkingManager: UserLinkingManager | null = null;
  private analyticsManager: CrossPlatformAnalyticsManager | null = null;

  // Track which managers have been initialized
  private managersInitialized: Set<CrossPlatformTab> = new Set();

  // Event handler references for cleanup
  private tabClickHandler: ((e: Event) => void) | null = null;
  private keydownHandler: ((e: Event) => void) | null = null;

  // Tab configuration
  private readonly tabs: TabConfig[] = [
    {
      id: 'analytics',
      label: 'Analytics',
      icon: '<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor"><path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zM9 17H7v-7h2v7zm4 0h-2V7h2v10zm4 0h-2v-4h2v4z"/></svg>',
      containerId: 'cp-analytics-panel'
    },
    {
      id: 'content',
      label: 'Content Mapping',
      icon: '<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor"><path d="M4 6h16v2H4zm0 5h16v2H4zm0 5h16v2H4z"/></svg>',
      containerId: 'cp-content-panel'
    },
    {
      id: 'users',
      label: 'User Linking',
      icon: '<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor"><path d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z"/></svg>',
      containerId: 'cp-users-panel'
    }
  ];

  constructor(api: API) {
    this.api = api;
  }

  /**
   * Initialize the cross-platform manager
   */
  async init(containerId: string): Promise<void> {
    this.container = document.getElementById(containerId);
    if (!this.container) {
      logger.warn('Container not found', { containerId });
      return;
    }

    logger.debug('Initializing');
    this.render();
    this.setupEventListeners();

    // Initialize the default tab
    await this.initializeTab(this.activeTab);
    this.initialized = true;

    logger.debug('Initialization complete');
  }

  /**
   * Render the main cross-platform UI
   */
  private render(): void {
    if (!this.container) return;

    const tabsHtml = this.tabs
      .map(tab => `
        <button
          class="cp-tab ${tab.id === this.activeTab ? 'active' : ''}"
          data-tab="${tab.id}"
          role="tab"
          aria-selected="${tab.id === this.activeTab}"
          aria-controls="${tab.containerId}"
          tabindex="${tab.id === this.activeTab ? '0' : '-1'}"
        >
          <span class="cp-tab-icon" aria-hidden="true">${tab.icon}</span>
          <span class="cp-tab-label">${tab.label}</span>
        </button>
      `)
      .join('');

    const panelsHtml = this.tabs
      .map(tab => `
        <div
          id="${tab.containerId}"
          class="cp-tab-panel ${tab.id === this.activeTab ? 'active' : ''}"
          role="tabpanel"
          aria-labelledby="cp-tab-${tab.id}"
          ${tab.id === this.activeTab ? '' : 'hidden'}
        >
          <div class="cp-panel-loading" id="${tab.containerId}-loading">
            <div class="loading-spinner"></div>
            <span>Loading ${tab.label}...</span>
          </div>
        </div>
      `)
      .join('');

    this.container.innerHTML = `
      <div class="cross-platform-container">
        <!-- Header -->
        <div class="cp-header">
          <h2 class="cp-title">Cross-Platform</h2>
          <p class="cp-subtitle">Manage content mappings and user links across Plex, Jellyfin, and Emby</p>
        </div>

        <!-- Tab Navigation -->
        <div class="cp-tabs" role="tablist" aria-label="Cross-platform sections">
          ${tabsHtml}
        </div>

        <!-- Tab Panels -->
        <div class="cp-panels">
          ${panelsHtml}
        </div>
      </div>
    `;
  }

  /**
   * Setup event listeners
   */
  private setupEventListeners(): void {
    // Tab click handler
    this.tabClickHandler = (e: Event) => {
      const target = e.target as HTMLElement;
      const tabBtn = target.closest('.cp-tab') as HTMLButtonElement;
      if (!tabBtn) return;

      const tabId = tabBtn.dataset.tab as CrossPlatformTab;
      if (tabId && tabId !== this.activeTab) {
        this.switchTab(tabId);
      }
    };

    const tabList = this.container?.querySelector('.cp-tabs');
    tabList?.addEventListener('click', this.tabClickHandler);

    // Keyboard navigation
    this.keydownHandler = (e: Event) => {
      if (!this.container) return;
      const keyboardEvent = e as KeyboardEvent;

      const tabList = this.container.querySelector('.cp-tabs');
      if (!tabList?.contains(document.activeElement)) return;

      const tabs = Array.from(tabList.querySelectorAll('.cp-tab')) as HTMLButtonElement[];
      const currentIndex = tabs.findIndex(tab => tab === document.activeElement);

      if (currentIndex === -1) return;

      let newIndex: number;

      switch (keyboardEvent.key) {
        case 'ArrowLeft':
        case 'ArrowUp':
          keyboardEvent.preventDefault();
          newIndex = currentIndex === 0 ? tabs.length - 1 : currentIndex - 1;
          break;
        case 'ArrowRight':
        case 'ArrowDown':
          keyboardEvent.preventDefault();
          newIndex = currentIndex === tabs.length - 1 ? 0 : currentIndex + 1;
          break;
        case 'Home':
          keyboardEvent.preventDefault();
          newIndex = 0;
          break;
        case 'End':
          keyboardEvent.preventDefault();
          newIndex = tabs.length - 1;
          break;
        default:
          return;
      }

      if (newIndex !== currentIndex) {
        tabs[newIndex].focus();
        const tabId = tabs[newIndex].dataset.tab as CrossPlatformTab;
        if (tabId) {
          this.switchTab(tabId);
        }
      }
    };

    this.container?.addEventListener('keydown', this.keydownHandler);
  }

  /**
   * Switch to a different tab
   */
  private async switchTab(tabId: CrossPlatformTab): Promise<void> {
    if (tabId === this.activeTab) return;

    // Update active tab
    const previousTab = this.activeTab;
    this.activeTab = tabId;

    // Update tab buttons
    const tabs = this.container?.querySelectorAll('.cp-tab');
    tabs?.forEach(tab => {
      const btn = tab as HTMLButtonElement;
      const isActive = btn.dataset.tab === tabId;
      btn.classList.toggle('active', isActive);
      btn.setAttribute('aria-selected', String(isActive));
      btn.setAttribute('tabindex', isActive ? '0' : '-1');
    });

    // Update panels
    const panels = this.container?.querySelectorAll('.cp-tab-panel');
    panels?.forEach(panel => {
      const isActive = panel.id === this.tabs.find(t => t.id === tabId)?.containerId;
      panel.classList.toggle('active', isActive);
      if (isActive) {
        panel.removeAttribute('hidden');
      } else {
        panel.setAttribute('hidden', '');
      }
    });

    // Initialize the tab if not already done
    await this.initializeTab(tabId);

    logger.debug(`[CrossPlatformManager] Switched from ${previousTab} to ${tabId}`);
  }

  /**
   * Initialize a specific tab's manager
   */
  private async initializeTab(tabId: CrossPlatformTab): Promise<void> {
    if (this.managersInitialized.has(tabId)) {
      // Already initialized, just refresh if needed
      return;
    }

    const tabConfig = this.tabs.find(t => t.id === tabId);
    if (!tabConfig) return;

    // Show loading state
    const loadingEl = document.getElementById(`${tabConfig.containerId}-loading`);

    try {
      switch (tabId) {
        case 'analytics':
          if (!this.analyticsManager) {
            this.analyticsManager = new CrossPlatformAnalyticsManager(this.api);
          }
          await this.analyticsManager.init(tabConfig.containerId);
          break;

        case 'content':
          if (!this.contentMappingManager) {
            this.contentMappingManager = new ContentMappingManager(this.api);
          }
          await this.contentMappingManager.init(tabConfig.containerId);
          break;

        case 'users':
          if (!this.userLinkingManager) {
            this.userLinkingManager = new UserLinkingManager(this.api);
          }
          await this.userLinkingManager.init(tabConfig.containerId);
          break;
      }

      this.managersInitialized.add(tabId);

      // Hide loading state
      if (loadingEl) {
        loadingEl.style.display = 'none';
      }
    } catch (error) {
      logger.error('Failed to initialize tab', { tabId, error });

      // Show error state
      if (loadingEl) {
        loadingEl.innerHTML = `
          <div class="cp-error">
            <span class="cp-error-icon">!</span>
            <span>Failed to load ${tabConfig.label}</span>
            <button class="btn btn-secondary btn-sm cp-retry-btn">
              Retry
            </button>
          </div>
        `;

        // CSP-compliant click handler (replaces inline onclick)
        const retryBtn = loadingEl.querySelector('.cp-retry-btn');
        if (retryBtn) {
          retryBtn.addEventListener('click', () => {
            loadingEl.style.display = 'flex';
            // Re-attempt initialization
            this.initializeTab(tabId);
          });
        }
      }
    }
  }

  /**
   * Get the content mapping manager
   */
  getContentMappingManager(): ContentMappingManager | null {
    return this.contentMappingManager;
  }

  /**
   * Get the user linking manager
   */
  getUserLinkingManager(): UserLinkingManager | null {
    return this.userLinkingManager;
  }

  /**
   * Get the analytics manager
   */
  getAnalyticsManager(): CrossPlatformAnalyticsManager | null {
    return this.analyticsManager;
  }

  /**
   * Refresh the current tab's data
   */
  async refresh(): Promise<void> {
    switch (this.activeTab) {
      case 'analytics':
        await this.analyticsManager?.refresh();
        break;
      case 'content':
        // ContentMappingManager doesn't have a public refresh, reload mappings
        if (this.contentMappingManager) {
          await this.contentMappingManager.init(
            this.tabs.find(t => t.id === 'content')!.containerId
          );
        }
        break;
      case 'users':
        // UserLinkingManager doesn't have a public refresh, reload data
        if (this.userLinkingManager) {
          await this.userLinkingManager.init(
            this.tabs.find(t => t.id === 'users')!.containerId
          );
        }
        break;
    }
  }

  /**
   * Navigate to a specific tab programmatically
   */
  navigateToTab(tabId: CrossPlatformTab): void {
    if (this.tabs.find(t => t.id === tabId)) {
      this.switchTab(tabId);
    }
  }

  /**
   * Check if the manager is initialized
   */
  isInitialized(): boolean {
    return this.initialized;
  }

  /**
   * Get the active tab
   */
  getActiveTab(): CrossPlatformTab {
    return this.activeTab;
  }

  /**
   * Clean up resources
   */
  destroy(): void {
    // Remove event listeners
    const tabList = this.container?.querySelector('.cp-tabs');
    if (tabList && this.tabClickHandler) {
      tabList.removeEventListener('click', this.tabClickHandler);
    }

    if (this.container && this.keydownHandler) {
      this.container.removeEventListener('keydown', this.keydownHandler);
    }

    // Destroy sub-managers
    this.contentMappingManager?.destroy();
    this.userLinkingManager?.destroy();
    this.analyticsManager?.destroy();

    // Clear references
    this.tabClickHandler = null;
    this.keydownHandler = null;
    this.contentMappingManager = null;
    this.userLinkingManager = null;
    this.analyticsManager = null;
    this.managersInitialized.clear();
    this.container = null;
    this.initialized = false;

    logger.debug('Destroyed');
  }
}
