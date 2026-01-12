// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * BreadcrumbNavigationManager - Provides breadcrumb navigation for analytics pages
 *
 * Features:
 * - Shows current location path in analytics section
 * - Allows quick navigation back to parent views
 * - Updates dynamically when page changes
 * - Keyboard accessible with proper ARIA attributes
 * - Screen reader friendly with landmark roles
 */

import { createLogger } from '../lib/logger';

const logger = createLogger('BreadcrumbNavigationManager');

/**
 * Breadcrumb item configuration
 */
interface BreadcrumbItem {
  label: string;
  ariaLabel?: string;
  action?: () => void;
  isCurrent: boolean;
}

/**
 * Page configuration for breadcrumb generation
 */
interface PageConfig {
  id: string;
  label: string;
  parent?: string;
}

/**
 * Callbacks for breadcrumb navigation
 */
export interface BreadcrumbCallbacks {
  navigateToDashboard: (view: string) => void;
  navigateToAnalyticsPage: (page: string) => void;
}

export class BreadcrumbNavigationManager {
  private callbacks: BreadcrumbCallbacks | null = null;
  private container: HTMLElement | null = null;
  private currentDashboardView: string = 'maps';
  private currentAnalyticsPage: string = 'overview';

  // Page configuration for all analytics pages
  private pageConfig: Map<string, PageConfig> = new Map([
    ['overview', { id: 'overview', label: 'Overview' }],
    ['content', { id: 'content', label: 'Content Analytics' }],
    ['users', { id: 'users', label: 'User Analytics' }],
    ['performance', { id: 'performance', label: 'Performance' }],
    ['geographic', { id: 'geographic', label: 'Geographic' }],
    ['advanced', { id: 'advanced', label: 'Advanced Analytics' }],
    ['library', { id: 'library', label: 'Library Analytics' }],
    ['users-profile', { id: 'users-profile', label: 'User Profile', parent: 'users' }],
    ['tautulli', { id: 'tautulli', label: 'Tautulli Data' }]
  ]);

  constructor() {}

  /**
   * Initialize the breadcrumb navigation manager
   * Note: Click listeners are NOT added here - NavigationManager handles navigation
   * and calls updateDashboardView/updateAnalyticsPage directly.
   * This prevents duplicate event handlers on navigation elements.
   */
  init(callbacks: BreadcrumbCallbacks): void {
    this.callbacks = callbacks;
    this.createBreadcrumbContainer();
    // Note: setupNavigationListeners() was removed because NavigationManager
    // uses cloneNode() which removes any listeners we add. Instead, NavigationManager
    // should call updateDashboardView/updateAnalyticsPage directly.
    logger.info('BreadcrumbNavigationManager initialized');
  }

  /**
   * Create the breadcrumb container element
   */
  private createBreadcrumbContainer(): void {
    // Create container
    this.container = document.createElement('nav');
    this.container.id = 'breadcrumb-nav';
    this.container.className = 'breadcrumb-nav';
    this.container.setAttribute('aria-label', 'Breadcrumb navigation');
    this.container.setAttribute('role', 'navigation');

    // Create list container
    const list = document.createElement('ol');
    list.className = 'breadcrumb-list';
    list.setAttribute('role', 'list');
    this.container.appendChild(list);

    // Insert into the analytics container, before the tabs
    const analyticsContainer = document.getElementById('analytics-container');
    const analyticsTabs = document.getElementById('analytics-tabs');

    if (analyticsContainer && analyticsTabs) {
      analyticsContainer.insertBefore(this.container, analyticsTabs);
    }

    // Initially hidden (only shown in analytics view)
    this.container.style.display = 'none';
  }

  /**
   * Update the current dashboard view
   * Called by NavigationManager when dashboard view changes
   */
  updateDashboardView(view: string): void {
    this.currentDashboardView = view;

    // Show/hide breadcrumbs based on view
    if (this.container) {
      this.container.style.display = view === 'analytics' ? 'block' : 'none';
    }

    if (view === 'analytics') {
      this.render();
    }
  }

  /**
   * Update the current analytics page
   */
  updateAnalyticsPage(page: string): void {
    this.currentAnalyticsPage = page;
    if (this.currentDashboardView === 'analytics') {
      this.render();
    }
  }

  /**
   * Render the breadcrumb trail
   */
  private render(): void {
    const list = this.container?.querySelector('.breadcrumb-list');
    if (!list) return;

    // Build breadcrumb items
    const items = this.buildBreadcrumbItems();

    // Clear and rebuild
    list.innerHTML = '';

    items.forEach((item, index) => {
      const li = document.createElement('li');
      li.className = 'breadcrumb-item';

      if (item.isCurrent) {
        li.classList.add('current');
        li.setAttribute('aria-current', 'page');

        const span = document.createElement('span');
        span.textContent = item.label;
        li.appendChild(span);
      } else {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'breadcrumb-link';
        button.textContent = item.label;
        button.setAttribute('aria-label', item.ariaLabel || `Go to ${item.label}`);

        if (item.action) {
          button.addEventListener('click', item.action);
        }

        li.appendChild(button);
      }

      // Add separator (except for last item)
      if (index < items.length - 1) {
        const separator = document.createElement('span');
        separator.className = 'breadcrumb-separator';
        separator.setAttribute('aria-hidden', 'true');
        separator.innerHTML = '&#x203A;'; // Single right-pointing angle quotation mark
        li.appendChild(separator);
      }

      list.appendChild(li);
    });
  }

  /**
   * Build the breadcrumb items based on current state
   */
  private buildBreadcrumbItems(): BreadcrumbItem[] {
    const items: BreadcrumbItem[] = [];

    // Home/Maps item
    items.push({
      label: 'Home',
      ariaLabel: 'Go to home (Maps view)',
      action: () => this.callbacks?.navigateToDashboard('maps'),
      isCurrent: false
    });

    // Analytics item
    if (this.currentDashboardView === 'analytics') {
      const pageConfig = this.pageConfig.get(this.currentAnalyticsPage);

      // If we're on a sub-page with a parent, add the parent first
      if (pageConfig?.parent) {
        const parentConfig = this.pageConfig.get(pageConfig.parent);
        if (parentConfig) {
          // Analytics overview
          items.push({
            label: 'Analytics',
            ariaLabel: 'Go to Analytics Overview',
            action: () => this.callbacks?.navigateToAnalyticsPage('overview'),
            isCurrent: false
          });

          // Parent page
          items.push({
            label: parentConfig.label,
            ariaLabel: `Go to ${parentConfig.label}`,
            action: () => this.callbacks?.navigateToAnalyticsPage(parentConfig.id),
            isCurrent: false
          });

          // Current sub-page
          items.push({
            label: pageConfig.label,
            isCurrent: true
          });
        }
      } else if (this.currentAnalyticsPage !== 'overview') {
        // Regular analytics page (not overview)
        items.push({
          label: 'Analytics',
          ariaLabel: 'Go to Analytics Overview',
          action: () => this.callbacks?.navigateToAnalyticsPage('overview'),
          isCurrent: false
        });

        items.push({
          label: pageConfig?.label || this.currentAnalyticsPage,
          isCurrent: true
        });
      } else {
        // We're on the overview
        items.push({
          label: 'Analytics',
          isCurrent: true
        });
      }
    }

    return items;
  }

  /**
   * Force refresh the breadcrumb display
   */
  refresh(): void {
    if (this.currentDashboardView === 'analytics') {
      this.render();
    }
  }

  /**
   * Get current navigation state
   */
  getState(): { dashboardView: string; analyticsPage: string } {
    return {
      dashboardView: this.currentDashboardView,
      analyticsPage: this.currentAnalyticsPage
    };
  }

  /**
   * Set navigation state programmatically
   */
  setState(dashboardView: string, analyticsPage?: string): void {
    this.currentDashboardView = dashboardView;
    if (analyticsPage) {
      this.currentAnalyticsPage = analyticsPage;
    }

    if (this.container) {
      this.container.style.display = dashboardView === 'analytics' ? 'block' : 'none';
    }

    if (dashboardView === 'analytics') {
      this.render();
    }
  }

  /**
   * Cleanup resources to prevent memory leaks
   * Removes container from DOM and clears callbacks
   */
  destroy(): void {
    if (this.container) {
      this.container.remove();
      this.container = null;
    }
    this.callbacks = null;
  }
}

export default BreadcrumbNavigationManager;
