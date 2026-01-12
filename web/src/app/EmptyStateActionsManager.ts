// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * EmptyStateActionsManager - Adds actionable CTAs to empty states
 *
 * Features:
 * - Enhances existing empty states with action buttons
 * - Context-aware CTAs based on empty state type
 * - Supports different empty state categories (no data, no selection, error)
 * - Keyboard accessible with proper ARIA attributes
 */

import { createLogger } from '../lib/logger';

const logger = createLogger('EmptyStateActionsManager');

/**
 * Empty state configuration
 */
interface EmptyStateConfig {
  elementId: string;
  type: 'no-selection' | 'no-data' | 'no-activity' | 'no-presets';
  icon?: string;
  title: string;
  message: string;
  actions: EmptyStateAction[];
}

/**
 * Action button configuration
 */
interface EmptyStateAction {
  label: string;
  ariaLabel: string;
  type: 'primary' | 'secondary';
  handler: () => void;
}

/**
 * Callbacks for empty state actions
 */
export interface EmptyStateCallbacks {
  navigateTo: (view: string) => void;
  refreshData: () => void;
  openHelp: () => void;
  selectLibrary: (libraryId: string) => void;
  selectUser: (userId: string) => void;
}

export class EmptyStateActionsManager {
  private callbacks: EmptyStateCallbacks | null = null;
  private enhancedStates: Set<string> = new Set();
  // Track MutationObservers for cleanup
  private observers: MutationObserver[] = [];

  constructor() {}

  /**
   * Initialize the empty state actions manager
   */
  init(callbacks: EmptyStateCallbacks): void {
    this.callbacks = callbacks;
    this.enhanceEmptyStates();
    logger.info('EmptyStateActionsManager initialized');
  }

  /**
   * Enhance all known empty states with actions
   */
  private enhanceEmptyStates(): void {
    // Define all empty state configurations
    const emptyStates: EmptyStateConfig[] = [
      {
        elementId: 'preset-list-empty',
        type: 'no-presets',
        icon: '📁',
        title: 'No Saved Presets',
        message: 'Save your current filter settings as a preset for quick access later.',
        actions: [
          {
            label: 'Learn More',
            ariaLabel: 'Open help documentation',
            type: 'secondary',
            handler: () => this.callbacks?.openHelp()
          }
        ]
      },
      {
        elementId: 'library-empty-state',
        type: 'no-selection',
        icon: '📚',
        title: 'Select a Library',
        message: 'Choose a library from the dropdown above to view detailed analytics.',
        actions: [
          {
            label: 'View Overview',
            ariaLabel: 'Go to analytics overview page',
            type: 'primary',
            handler: () => this.callbacks?.navigateTo('analytics')
          }
        ]
      },
      {
        elementId: 'user-profile-empty-state',
        type: 'no-selection',
        icon: '👤',
        title: 'Select a User',
        message: 'Choose a user from the dropdown above to view their profile and activity analytics.',
        actions: [
          {
            label: 'View All Users',
            ariaLabel: 'Go to users analytics page',
            type: 'primary',
            handler: () => this.callbacks?.navigateTo('analytics-users')
          }
        ]
      },
      {
        elementId: 'activity-empty-state',
        type: 'no-activity',
        icon: '📺',
        title: 'No Active Streams',
        message: 'There are no active streams at this time. Start playing something on Plex to see activity here.',
        actions: [
          {
            label: 'Refresh',
            ariaLabel: 'Refresh activity data',
            type: 'primary',
            handler: () => this.callbacks?.refreshData()
          },
          {
            label: 'View History',
            ariaLabel: 'View playback history on map',
            type: 'secondary',
            handler: () => this.callbacks?.navigateTo('map')
          }
        ]
      }
    ];

    // Enhance each empty state
    emptyStates.forEach(config => this.enhanceEmptyState(config));

    // Also enhance dynamically generated empty states
    this.setupDynamicEnhancements();
  }

  /**
   * Enhance a single empty state element
   */
  private enhanceEmptyState(config: EmptyStateConfig): void {
    const element = document.getElementById(config.elementId);
    if (!element || this.enhancedStates.has(config.elementId)) return;

    // Mark as enhanced
    this.enhancedStates.add(config.elementId);

    // Rebuild the empty state content
    element.innerHTML = '';
    element.className = 'empty-state';
    element.setAttribute('role', 'status');
    element.setAttribute('aria-label', config.title);

    // Icon
    if (config.icon) {
      const iconEl = document.createElement('div');
      iconEl.className = 'empty-state-icon';
      iconEl.setAttribute('aria-hidden', 'true');
      iconEl.textContent = config.icon;
      element.appendChild(iconEl);
    }

    // Title
    const titleEl = document.createElement('h3');
    titleEl.className = 'empty-state-title';
    titleEl.textContent = config.title;
    element.appendChild(titleEl);

    // Message
    const messageEl = document.createElement('p');
    messageEl.className = 'empty-state-message';
    messageEl.textContent = config.message;
    element.appendChild(messageEl);

    // Actions
    if (config.actions.length > 0) {
      const actionsEl = document.createElement('div');
      actionsEl.className = 'empty-state-actions';

      config.actions.forEach((action, index) => {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = `empty-state-btn empty-state-btn-${action.type}`;
        button.textContent = action.label;
        button.setAttribute('aria-label', action.ariaLabel);
        button.addEventListener('click', action.handler);

        // Focus first primary action
        if (index === 0 && action.type === 'primary') {
          button.setAttribute('data-autofocus', 'true');
        }

        actionsEl.appendChild(button);
      });

      element.appendChild(actionsEl);
    }
  }

  /**
   * Setup observers for dynamically generated empty states
   */
  private setupDynamicEnhancements(): void {
    // Observer for chart empty states
    const chartContainers = document.querySelectorAll('.chart-container, .chart-content');

    chartContainers.forEach(container => {
      const observer = new MutationObserver(mutations => {
        mutations.forEach(mutation => {
          if (mutation.type === 'childList') {
            this.checkAndEnhanceChartEmptyState(container as HTMLElement);
          }
        });
      });

      observer.observe(container, { childList: true, subtree: true });
      // Track observer for cleanup
      this.observers.push(observer);
    });

    // Initial check for map empty state
    this.setupMapEmptyState();
  }

  /**
   * Check and enhance chart container empty states
   */
  private checkAndEnhanceChartEmptyState(container: HTMLElement): void {
    // Look for existing "no data" indicators in chart containers
    const noDataEl = container.querySelector('.echarts-no-data, .no-data-message');
    if (noDataEl && !noDataEl.classList.contains('enhanced-empty-state')) {
      noDataEl.classList.add('enhanced-empty-state');

      // Add refresh button if not present
      if (!noDataEl.querySelector('.empty-state-btn')) {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'empty-state-btn empty-state-btn-secondary empty-state-btn-small';
        button.textContent = 'Refresh';
        button.setAttribute('aria-label', 'Refresh chart data');
        button.addEventListener('click', () => this.callbacks?.refreshData());
        noDataEl.appendChild(button);
      }
    }
  }

  /**
   * Setup empty state for map view when no locations
   */
  private setupMapEmptyState(): void {
    // This will be called by the map component when needed
    // The map has its own empty state handling, so we provide a public method
  }

  /**
   * Show map empty state (called by MapManager when no locations)
   */
  showMapEmptyState(container: HTMLElement): void {
    // Check if already showing empty state
    if (container.querySelector('.map-empty-state')) return;

    const emptyState = document.createElement('div');
    emptyState.className = 'map-empty-state empty-state';
    emptyState.setAttribute('role', 'status');
    emptyState.innerHTML = `
      <div class="empty-state-icon" aria-hidden="true">🗺️</div>
      <h3 class="empty-state-title">No Locations Found</h3>
      <p class="empty-state-message">No playback locations match your current filters. Try adjusting the date range or clearing filters.</p>
      <div class="empty-state-actions">
        <button type="button" class="empty-state-btn empty-state-btn-primary" aria-label="Clear all filters">Clear Filters</button>
        <button type="button" class="empty-state-btn empty-state-btn-secondary" aria-label="Refresh map data">Refresh</button>
      </div>
    `;

    // Add click handlers
    const buttons = emptyState.querySelectorAll('button');
    if (buttons[0]) {
      buttons[0].addEventListener('click', () => {
        // Trigger filter clear
        document.getElementById('clear-filters')?.click();
      });
    }
    if (buttons[1]) {
      buttons[1].addEventListener('click', () => this.callbacks?.refreshData());
    }

    container.appendChild(emptyState);
  }

  /**
   * Hide map empty state
   */
  hideMapEmptyState(container: HTMLElement): void {
    const emptyState = container.querySelector('.map-empty-state');
    if (emptyState) {
      emptyState.remove();
    }
  }

  /**
   * Show generic empty state in a container
   */
  showEmptyState(
    container: HTMLElement,
    config: {
      icon?: string;
      title: string;
      message: string;
      actions?: EmptyStateAction[];
    }
  ): HTMLElement {
    const emptyState = document.createElement('div');
    emptyState.className = 'empty-state dynamic-empty-state';
    emptyState.setAttribute('role', 'status');

    let html = '';

    if (config.icon) {
      html += `<div class="empty-state-icon" aria-hidden="true">${config.icon}</div>`;
    }

    html += `
      <h3 class="empty-state-title">${this.escapeHtml(config.title)}</h3>
      <p class="empty-state-message">${this.escapeHtml(config.message)}</p>
    `;

    emptyState.innerHTML = html;

    if (config.actions && config.actions.length > 0) {
      const actionsEl = document.createElement('div');
      actionsEl.className = 'empty-state-actions';

      config.actions.forEach(action => {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = `empty-state-btn empty-state-btn-${action.type}`;
        button.textContent = action.label;
        button.setAttribute('aria-label', action.ariaLabel);
        button.addEventListener('click', action.handler);
        actionsEl.appendChild(button);
      });

      emptyState.appendChild(actionsEl);
    }

    container.appendChild(emptyState);
    return emptyState;
  }

  /**
   * Remove dynamic empty state from container
   */
  removeEmptyState(container: HTMLElement): void {
    const emptyState = container.querySelector('.dynamic-empty-state');
    if (emptyState) {
      emptyState.remove();
    }
  }

  /**
   * Escape HTML to prevent XSS
   */
  private escapeHtml(text: string): string {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  /**
   * Re-enhance empty states (call after dynamic content changes)
   */
  refresh(): void {
    // Clear enhanced states set and re-enhance
    this.enhancedStates.clear();
    this.enhanceEmptyStates();
  }

  /**
   * Clean up observers and resources
   */
  destroy(): void {
    // Disconnect all MutationObservers
    for (const observer of this.observers) {
      observer.disconnect();
    }
    this.observers = [];

    // Clear tracking state
    this.enhancedStates.clear();
    this.callbacks = null;
  }
}

export default EmptyStateActionsManager;
