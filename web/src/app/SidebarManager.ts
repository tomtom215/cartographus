// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * SidebarManager - Handles mobile hamburger menu and desktop collapse functionality
 *
 * Features:
 * - Mobile hamburger menu toggle
 * - Desktop sidebar collapse toggle
 * - Filter panel collapse toggle
 * - Overlay click to close sidebar
 * - Escape key to close sidebar
 * - Navigation tab click closes sidebar on mobile
 * - Persists collapse state in localStorage
 */

import { createLogger } from '../lib/logger';
import { SafeStorage } from '../lib/utils/SafeStorage';

const logger = createLogger('SidebarManager');

export class SidebarManager {
  private sidebar: HTMLElement | null;
  private menuToggle: HTMLElement | null;
  private sidebarOverlay: HTMLElement | null;
  private collapseToggle: HTMLElement | null;
  private collapseIcon: HTMLElement | null;
  private filtersPanel: HTMLElement | null;
  private filtersCollapseBtn: HTMLElement | null;
  private isMobile: boolean;
  private isOpen: boolean;
  private isCollapsed: boolean;
  private isFiltersCollapsed: boolean;

  // Event handler references for cleanup
  private menuToggleClickHandler: (() => void) | null = null;
  private overlayClickHandler: (() => void) | null = null;
  private documentKeydownHandler: ((e: KeyboardEvent) => void) | null = null;
  private collapseToggleClickHandler: (() => void) | null = null;
  private filtersCollapseBtnClickHandler: (() => void) | null = null;
  private windowResizeHandler: (() => void) | null = null;
  private tabTrapKeydownHandler: ((e: KeyboardEvent) => void) | null = null;
  private navTabClickHandlers: Map<Element, () => void> = new Map();

  constructor() {
    this.sidebar = document.getElementById('sidebar');
    this.menuToggle = document.getElementById('menu-toggle');
    this.sidebarOverlay = document.getElementById('sidebar-overlay');
    this.collapseToggle = document.getElementById('sidebar-collapse');
    this.collapseIcon = document.getElementById('collapse-icon');
    this.filtersPanel = document.getElementById('filters');
    this.filtersCollapseBtn = document.getElementById('filters-collapse-btn');
    this.isMobile = window.innerWidth <= 768;
    this.isOpen = false;
    this.isCollapsed = false;
    this.isFiltersCollapsed = false;
  }

  /**
   * Initialize the sidebar manager
   */
  init(): void {
    this.loadCollapseState();
    this.loadFiltersCollapseState();
    this.setupEventListeners();
    this.updateMobileState();
  }

  /**
   * Load collapse state from localStorage
   */
  private loadCollapseState(): void {
    const saved = SafeStorage.getItem('sidebar-collapsed');
    if (saved === 'true' && !this.isMobile) {
      this.isCollapsed = true;
      this.applyCollapseState();
    }
  }

  /**
   * Load filters panel collapse state from localStorage
   */
  private loadFiltersCollapseState(): void {
    const saved = SafeStorage.getItem('filters-panel-collapsed');
    if (saved === 'true') {
      this.isFiltersCollapsed = true;
      this.applyFiltersCollapseState();
    }
  }

  /**
   * Set up all event listeners
   */
  private setupEventListeners(): void {
    // Hamburger menu toggle
    if (this.menuToggle) {
      this.menuToggleClickHandler = () => this.toggleMobileMenu();
      this.menuToggle.addEventListener('click', this.menuToggleClickHandler);
    }

    // Overlay click to close
    if (this.sidebarOverlay) {
      this.overlayClickHandler = () => this.closeMobileMenu();
      this.sidebarOverlay.addEventListener('click', this.overlayClickHandler);
    }

    // Escape key to close
    this.documentKeydownHandler = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && this.isOpen) {
        this.closeMobileMenu();
      }
    };
    document.addEventListener('keydown', this.documentKeydownHandler);

    // Nav tab clicks close sidebar on mobile
    // Use a small delay to allow navigation to complete before closing
    // This prevents the sidebar closing from interfering with view transitions
    const navTabs = document.querySelectorAll('.nav-tab');
    navTabs.forEach((tab) => {
      const handler = () => {
        if (this.isMobile && this.isOpen) {
          // Delay close to allow navigation handler to process first
          requestAnimationFrame(() => {
            // Double-check we're still open (another handler might have closed it)
            if (this.isOpen) {
              this.closeMobileMenu();
            }
          });
        }
      };
      this.navTabClickHandlers.set(tab, handler);
      tab.addEventListener('click', handler);
    });

    // Sidebar collapse toggle
    if (this.collapseToggle) {
      this.collapseToggleClickHandler = () => this.toggleCollapse();
      this.collapseToggle.addEventListener('click', this.collapseToggleClickHandler);
    }

    // Filters panel collapse toggle
    if (this.filtersCollapseBtn) {
      this.filtersCollapseBtnClickHandler = () => this.toggleFiltersCollapse();
      this.filtersCollapseBtn.addEventListener('click', this.filtersCollapseBtnClickHandler);
    }

    // Window resize handler
    this.windowResizeHandler = () => this.handleResize();
    window.addEventListener('resize', this.windowResizeHandler);
  }

  /**
   * Handle window resize - update mobile state
   */
  private handleResize(): void {
    const wasMobile = this.isMobile;
    this.isMobile = window.innerWidth <= 768;

    // Switching from desktop to mobile
    if (!wasMobile && this.isMobile) {
      // Reset collapse state when going mobile
      if (this.sidebar) {
        this.sidebar.classList.remove('collapsed');
      }
      this.closeMobileMenu();
    }

    // Switching from mobile to desktop
    if (wasMobile && !this.isMobile) {
      this.closeMobileMenu();
      // Restore collapse state on desktop
      if (this.isCollapsed) {
        this.applyCollapseState();
      }
    }

    this.updateMobileState();
  }

  /**
   * Update mobile-specific state
   */
  private updateMobileState(): void {
    if (this.menuToggle) {
      this.menuToggle.setAttribute('aria-hidden', this.isMobile ? 'false' : 'true');
    }
    if (this.collapseToggle) {
      this.collapseToggle.setAttribute('aria-hidden', this.isMobile ? 'true' : 'false');
    }
  }

  /**
   * Toggle mobile menu open/close
   */
  toggleMobileMenu(): void {
    if (this.isOpen) {
      this.closeMobileMenu();
    } else {
      this.openMobileMenu();
    }
  }

  /**
   * Open mobile menu
   */
  openMobileMenu(): void {
    this.isOpen = true;

    if (this.sidebar) {
      this.sidebar.classList.add('open');
    }
    if (this.menuToggle) {
      this.menuToggle.classList.add('open');
      this.menuToggle.setAttribute('aria-expanded', 'true');
    }
    if (this.sidebarOverlay) {
      this.sidebarOverlay.classList.add('open');
      this.sidebarOverlay.setAttribute('aria-hidden', 'false');
    }

    // Trap focus in sidebar
    this.trapFocus();
  }

  /**
   * Close mobile menu
   */
  closeMobileMenu(): void {
    this.isOpen = false;

    if (this.sidebar) {
      this.sidebar.classList.remove('open');
    }
    if (this.menuToggle) {
      this.menuToggle.classList.remove('open');
      this.menuToggle.setAttribute('aria-expanded', 'false');
    }
    if (this.sidebarOverlay) {
      this.sidebarOverlay.classList.remove('open');
      this.sidebarOverlay.setAttribute('aria-hidden', 'true');
    }

    // Return focus to hamburger button
    if (this.menuToggle) {
      this.menuToggle.focus();
    }
  }

  /**
   * Toggle sidebar collapse
   */
  toggleCollapse(): void {
    if (this.isMobile) return; // Don't collapse on mobile

    this.isCollapsed = !this.isCollapsed;
    this.applyCollapseState();

    // Save to localStorage
    SafeStorage.setItem('sidebar-collapsed', this.isCollapsed.toString());

    // Temporarily disable hover expansion after collapse
    // This prevents the sidebar from immediately expanding when the mouse
    // is still over the sidebar after clicking the collapse button
    if (this.isCollapsed && this.sidebar) {
      this.sidebar.classList.add('no-hover-expand');
      setTimeout(() => {
        this.sidebar?.classList.remove('no-hover-expand');
      }, 400); // Remove after CSS transition completes
    }
  }

  /**
   * Apply the current collapse state to the UI
   */
  private applyCollapseState(): void {
    if (!this.sidebar || !this.collapseToggle) return;

    if (this.isCollapsed) {
      this.sidebar.classList.add('collapsed');
      this.collapseToggle.setAttribute('aria-label', 'Expand sidebar');
      this.collapseToggle.setAttribute('title', 'Expand sidebar');
      this.collapseToggle.setAttribute('aria-expanded', 'false');
      if (this.collapseIcon) {
        this.collapseIcon.textContent = '\u25B6'; // Right arrow
      }
    } else {
      this.sidebar.classList.remove('collapsed');
      this.collapseToggle.setAttribute('aria-label', 'Collapse sidebar');
      this.collapseToggle.setAttribute('title', 'Collapse sidebar');
      this.collapseToggle.setAttribute('aria-expanded', 'true');
      if (this.collapseIcon) {
        this.collapseIcon.textContent = '\u25C0'; // Left arrow
      }
    }
  }

  /**
   * Trap focus within sidebar when open (accessibility)
   */
  private trapFocus(): void {
    if (!this.sidebar) return;

    const focusableElements = this.sidebar.querySelectorAll(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );

    if (focusableElements.length === 0) return;

    const firstElement = focusableElements[0] as HTMLElement;
    const lastElement = focusableElements[focusableElements.length - 1] as HTMLElement;

    // Focus first element
    firstElement.focus();

    // Remove previous tab trap handler if exists
    if (this.tabTrapKeydownHandler) {
      document.removeEventListener('keydown', this.tabTrapKeydownHandler);
    }

    // Handle tab navigation
    this.tabTrapKeydownHandler = (e: KeyboardEvent) => {
      if (!this.isOpen) {
        if (this.tabTrapKeydownHandler) {
          document.removeEventListener('keydown', this.tabTrapKeydownHandler);
          this.tabTrapKeydownHandler = null;
        }
        return;
      }

      if (e.key !== 'Tab') return;

      if (e.shiftKey) {
        // Shift + Tab
        if (document.activeElement === firstElement) {
          e.preventDefault();
          lastElement.focus();
        }
      } else {
        // Tab
        if (document.activeElement === lastElement) {
          e.preventDefault();
          firstElement.focus();
        }
      }
    };

    document.addEventListener('keydown', this.tabTrapKeydownHandler);
  }

  /**
   * Check if sidebar is currently open (mobile) or expanded (desktop)
   */
  isExpanded(): boolean {
    if (this.isMobile) {
      return this.isOpen;
    }
    return !this.isCollapsed;
  }

  /**
   * Toggle filter panel collapse
   */
  toggleFiltersCollapse(): void {
    this.isFiltersCollapsed = !this.isFiltersCollapsed;
    this.applyFiltersCollapseState();

    // Save to localStorage
    SafeStorage.setItem('filters-panel-collapsed', this.isFiltersCollapsed.toString());

    logger.info('Filters panel', this.isFiltersCollapsed ? 'collapsed' : 'expanded');
  }

  /**
   * Apply the current filter collapse state to the UI
   */
  private applyFiltersCollapseState(): void {
    if (!this.filtersPanel || !this.filtersCollapseBtn) return;

    if (this.isFiltersCollapsed) {
      this.filtersPanel.classList.add('collapsed');
      this.filtersCollapseBtn.setAttribute('aria-expanded', 'false');
      this.filtersCollapseBtn.setAttribute('aria-label', 'Expand filters panel');
      this.filtersCollapseBtn.setAttribute('title', 'Expand filters');
    } else {
      this.filtersPanel.classList.remove('collapsed');
      this.filtersCollapseBtn.setAttribute('aria-expanded', 'true');
      this.filtersCollapseBtn.setAttribute('aria-label', 'Collapse filters panel');
      this.filtersCollapseBtn.setAttribute('title', 'Collapse filters');
    }
  }

  /**
   * Check if filters panel is collapsed
   */
  isFiltersPanelCollapsed(): boolean {
    return this.isFiltersCollapsed;
  }

  /**
   * Remove all event listeners for cleanup
   */
  private removeEventListeners(): void {
    // Remove hamburger menu toggle handler
    if (this.menuToggle && this.menuToggleClickHandler) {
      this.menuToggle.removeEventListener('click', this.menuToggleClickHandler);
      this.menuToggleClickHandler = null;
    }

    // Remove overlay click handler
    if (this.sidebarOverlay && this.overlayClickHandler) {
      this.sidebarOverlay.removeEventListener('click', this.overlayClickHandler);
      this.overlayClickHandler = null;
    }

    // Remove document escape key handler
    if (this.documentKeydownHandler) {
      document.removeEventListener('keydown', this.documentKeydownHandler);
      this.documentKeydownHandler = null;
    }

    // Remove nav tab click handlers
    this.navTabClickHandlers.forEach((handler, tab) => {
      tab.removeEventListener('click', handler);
    });
    this.navTabClickHandlers.clear();

    // Remove collapse toggle handler
    if (this.collapseToggle && this.collapseToggleClickHandler) {
      this.collapseToggle.removeEventListener('click', this.collapseToggleClickHandler);
      this.collapseToggleClickHandler = null;
    }

    // Remove filters collapse button handler
    if (this.filtersCollapseBtn && this.filtersCollapseBtnClickHandler) {
      this.filtersCollapseBtn.removeEventListener('click', this.filtersCollapseBtnClickHandler);
      this.filtersCollapseBtnClickHandler = null;
    }

    // Remove window resize handler
    if (this.windowResizeHandler) {
      window.removeEventListener('resize', this.windowResizeHandler);
      this.windowResizeHandler = null;
    }

    // Remove tab trap keydown handler
    if (this.tabTrapKeydownHandler) {
      document.removeEventListener('keydown', this.tabTrapKeydownHandler);
      this.tabTrapKeydownHandler = null;
    }
  }

  /**
   * Destroy the sidebar manager and clean up resources
   */
  destroy(): void {
    this.removeEventListeners();
    this.closeMobileMenu();
  }
}

export default SidebarManager;
