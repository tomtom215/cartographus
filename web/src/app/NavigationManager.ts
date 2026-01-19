// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * NavigationManager - Handles dashboard and analytics page navigation
 *
 * Manages:
 * - Dashboard view switching (Maps, Analytics, Activity, Server, Recently Added)
 * - Analytics page navigation (Overview, Content, Users, Performance, Geographic, Advanced, Library, User Profile, Tautulli)
 * - URL hash management and deep linking
 * - Browser back/forward navigation
 * - Keyboard shortcuts (Arrow keys, Number keys 1-9, Help key)
 * - Active tab state management
 *
 * Navigation Modes:
 * - Dashboard views: maps, activity, analytics, recently-added, server
 * - Analytics pages: overview, content, users, performance, geographic, advanced, library, users-profile, tautulli
 */

import type { MapManager } from '../lib/map';
import type { GlobeManagerDeckGL } from '../lib/globe-deckgl';
import type { ChartManager } from '../lib/charts';
import type { ToastManager } from '../lib/toast';
import type { ActivityDashboardManager } from '../lib/activity-dashboard';
import type { ServerDashboardManager } from '../lib/server-dashboard';
import type { RecentlyAddedDashboardManager } from '../lib/recently-added-dashboard';
import type { ViewportManager } from './ViewportManager';
import type { BreadcrumbNavigationManager } from './BreadcrumbNavigationManager';
import { createLogger } from '../lib/logger';
import { getRoleGuard } from '../lib/auth/RoleGuard';

const logger = createLogger('NavigationManager');

export type DashboardView = 'maps' | 'activity' | 'analytics' | 'recently-added' | 'server' | 'cross-platform' | 'data-governance' | 'newsletter';
export type AnalyticsPage = 'overview' | 'content' | 'users' | 'performance' | 'geographic' | 'advanced' | 'library' | 'users-profile' | 'tautulli' | 'wrapped';

// Dashboard container element IDs - single source of truth
const DASHBOARD_CONTAINER_IDS: Record<DashboardView, string> = {
    'maps': 'map-container',
    'activity': 'activity-container',
    'analytics': 'analytics-container',
    'recently-added': 'recently-added-container',
    'server': 'server-container',
    'cross-platform': 'cross-platform-container',
    'data-governance': 'data-governance-container',
    'newsletter': 'newsletter-container'
};

// Analytics page element IDs - single source of truth
const ANALYTICS_PAGE_IDS: Record<AnalyticsPage, string> = {
    'overview': 'analytics-overview',
    'content': 'analytics-content',
    'users': 'analytics-users',
    'performance': 'analytics-performance',
    'geographic': 'analytics-geographic',
    'advanced': 'analytics-advanced',
    'library': 'analytics-library',
    'users-profile': 'analytics-users-profile',
    'tautulli': 'analytics-tautulli',
    'wrapped': 'analytics-wrapped'
};

// Valid dashboard views and analytics pages - derived from the mappings above
const VALID_DASHBOARD_VIEWS = Object.keys(DASHBOARD_CONTAINER_IDS) as DashboardView[];
const VALID_ANALYTICS_PAGES = Object.keys(ANALYTICS_PAGE_IDS) as AnalyticsPage[];

/**
 * Get dashboard container elements from the constant mapping.
 * Returns a record of view -> HTML element (or null if not found).
 */
function getDashboardContainerElements(): Record<DashboardView, HTMLElement | null> {
    return Object.fromEntries(
        Object.entries(DASHBOARD_CONTAINER_IDS).map(([view, id]) => [view, document.getElementById(id)])
    ) as Record<DashboardView, HTMLElement | null>;
}

/**
 * Get analytics page elements from the constant mapping.
 * Returns a record of page -> HTML element (or null if not found).
 */
function getAnalyticsPageElements(): Record<AnalyticsPage, HTMLElement | null> {
    return Object.fromEntries(
        Object.entries(ANALYTICS_PAGE_IDS).map(([page, id]) => [page, document.getElementById(id)])
    ) as Record<AnalyticsPage, HTMLElement | null>;
}

export class NavigationManager {
    private currentDashboardView: DashboardView = 'maps';
    private currentAnalyticsPage: AnalyticsPage = 'overview';
    private onLibraryPageShow?: () => void;
    private onUserProfilePageShow?: () => void;
    private onCrossPlatformShow?: () => void;
    private onDataGovernanceShow?: () => void;
    private onNewsletterShow?: () => void;
    private onWrappedPageShow?: () => void;

    // Bound event handlers for proper cleanup
    private readonly boundHandleHashChange: () => void;
    private readonly boundHandleKeyDown: (e: KeyboardEvent) => void;
    private readonly boundHandleResize: () => void;

    // MutationObserver reference for cleanup
    private scrollIndicatorObserver: MutationObserver | null = null;
    // Reference to scroll update function for cleanup
    private scrollUpdateHandler: (() => void) | null = null;
    // Reference to the nav element for cleanup
    private analyticsNavElement: HTMLElement | null = null;

    // Maps to store click handlers for nav tabs and analytics tabs
    // Enables proper cleanup without DOM cloning anti-pattern
    private navTabHandlers: Map<HTMLElement, (e: MouseEvent) => void> = new Map();
    private analyticsTabHandlers: Map<HTMLElement, (e: MouseEvent) => void> = new Map();

    // BreadcrumbNavigationManager reference for notifying of view changes
    private breadcrumbManager: BreadcrumbNavigationManager | null = null;

    constructor(
        private mapManager: MapManager | null,
        private globeManager: GlobeManagerDeckGL | null,
        private chartManager: ChartManager | null,
        private viewportManager: ViewportManager | null,
        private activityDashboard: ActivityDashboardManager | null,
        private serverDashboard: ServerDashboardManager | null,
        private recentlyAddedDashboard: RecentlyAddedDashboardManager | null,
        private toastManager: ToastManager | null
    ) {
        // Bind event handlers once to ensure consistent reference for add/remove
        this.boundHandleHashChange = this.handleHashChange.bind(this);
        this.boundHandleKeyDown = this.handleKeyDown.bind(this);
        this.boundHandleResize = () => {
            if (this.scrollUpdateHandler) {
                this.scrollUpdateHandler();
            }
        };

        this.setupNavigationListeners();
        this.setupURLHashListener();
        this.setupKeyboardNavigation();
        this.setupAnalyticsNavScrollIndicators();
        // Load view from URL on initialization
        this.loadViewFromURL();
    }

    /**
     * Set globe manager reference (for lazy-loaded module)
     */
    setGlobeManager(globeManager: GlobeManagerDeckGL | null): void {
        this.globeManager = globeManager;
    }

    /**
     * Set chart manager reference (for lazy-loaded module)
     */
    setChartManager(chartManager: ChartManager | null): void {
        this.chartManager = chartManager;
    }

    /**
     * Set callback for when library page is shown
     */
    setLibraryPageShowCallback(callback: () => void): void {
        this.onLibraryPageShow = callback;
    }

    /**
     * Set callback for when user profile page is shown
     */
    setUserProfilePageShowCallback(callback: () => void): void {
        this.onUserProfilePageShow = callback;
    }

    /**
     * Set callback for when cross-platform view is shown
     */
    setCrossPlatformShowCallback(callback: () => void): void {
        this.onCrossPlatformShow = callback;
    }

    /**
     * Set callback for when data governance tab is shown
     */
    setDataGovernanceShowCallback(callback: () => void): void {
        this.onDataGovernanceShow = callback;
    }

    /**
     * Set callback for when newsletter tab is shown
     */
    setNewsletterShowCallback(callback: () => void): void {
        this.onNewsletterShow = callback;
    }

    /**
     * Set callback for when wrapped page is shown
     */
    setWrappedPageShowCallback(callback: () => void): void {
        this.onWrappedPageShow = callback;
    }

    /**
     * Set breadcrumb navigation manager reference
     * Used to notify breadcrumbs of view changes (prevents duplicate listeners)
     */
    setBreadcrumbManager(manager: BreadcrumbNavigationManager | null): void {
        this.breadcrumbManager = manager;
    }

    /**
     * Get current dashboard view
     */
    getCurrentDashboardView(): DashboardView {
        return this.currentDashboardView;
    }

    /**
     * Get current analytics page
     */
    getCurrentAnalyticsPage(): AnalyticsPage {
        return this.currentAnalyticsPage;
    }

    /**
     * Load view from URL hash on page load
     * Supports URLs like: /#maps, /#activity, /#analytics-content, /#server
     */
    loadViewFromURL(): void {
        const hash = window.location.hash.substring(1); // Remove the '#'

        if (!hash) {
            // No hash, default to maps view
            return;
        }

        // Check if it's an analytics page hash
        if (hash.startsWith('analytics-')) {
            // Switch to analytics view first, then the specific page will be loaded
            this.switchDashboardView('analytics');
            return;
        }

        // Check if it's a valid main view hash
        if (VALID_DASHBOARD_VIEWS.includes(hash as DashboardView)) {
            logger.debug(`[URL] Loading dashboard view from URL: ${hash}`);
            this.switchDashboardView(hash as DashboardView);
        }
    }

    /**
     * Update URL hash based on current view
     * For analytics, uses format #analytics-{page}, for others uses #{view}
     */
    private updateURLFromView(view: DashboardView): void {
        // Don't update hash for analytics view (switchAnalyticsPage handles that)
        if (view === 'analytics') {
            return;
        }

        // For maps view, we can optionally clear the hash or keep it
        // Using replaceState to avoid adding to browser history on every switch
        const newHash = view === 'maps' ? '' : `#${view}`;
        const newURL = `${window.location.pathname}${window.location.search}${newHash}`;
        window.history.replaceState({}, '', newURL);
    }

    /**
     * Set up navigation tab listeners.
     * This is called immediately after showing the app, BEFORE any async operations,
     * to ensure navigation tabs are clickable right away.
     * This is critical for E2E tests that wait for #app visibility and then click tabs.
     *
     * Uses stored handler references for proper cleanup instead of DOM cloning.
     */
    private setupNavigationListeners(): void {
        // Clean up any existing handlers first (prevents duplicates on re-initialization)
        this.cleanupNavigationListeners();

        // Navigation tabs (Maps, Analytics, Activity, Server Info, Recently Added)
        const navTabs = document.querySelectorAll<HTMLElement>('.nav-tab');
        navTabs.forEach(tab => {
            const handler = (e: MouseEvent): void => {
                const button = e.currentTarget as HTMLButtonElement;
                const view = button.getAttribute('data-view') as DashboardView;
                if (view) {
                    this.switchDashboardView(view);
                }
            };
            // Store handler reference for cleanup
            this.navTabHandlers.set(tab, handler);
            tab.addEventListener('click', handler);
        });

        // Analytics sub-navigation tabs (Overview, Content, Users, Performance, Geographic, Advanced)
        const analyticsTabs = document.querySelectorAll<HTMLElement>('.analytics-tab');
        analyticsTabs.forEach(tab => {
            const handler = (e: MouseEvent): void => {
                const button = e.currentTarget as HTMLButtonElement;
                const page = button.getAttribute('data-analytics-page') as AnalyticsPage;
                if (page) {
                    this.switchAnalyticsPage(page);
                }
            };
            // Store handler reference for cleanup
            this.analyticsTabHandlers.set(tab, handler);
            tab.addEventListener('click', handler);
        });
    }

    /**
     * Clean up navigation tab listeners.
     * Removes all stored click handlers from nav and analytics tabs.
     */
    private cleanupNavigationListeners(): void {
        // Remove nav tab handlers
        this.navTabHandlers.forEach((handler, tab) => {
            tab.removeEventListener('click', handler);
        });
        this.navTabHandlers.clear();

        // Remove analytics tab handlers
        this.analyticsTabHandlers.forEach((handler, tab) => {
            tab.removeEventListener('click', handler);
        });
        this.analyticsTabHandlers.clear();
    }

    /**
     * Switch between dashboard views
     */
    async switchDashboardView(view: DashboardView): Promise<void> {
        // Destroy current dashboard if switching away
        this.destroyCurrentDashboard();

        this.currentDashboardView = view;

        // Notify breadcrumb manager of view change
        this.breadcrumbManager?.updateDashboardView(view);

        // Update URL hash
        this.updateURLFromView(view);

        // Update UI state
        this.updateDashboardTabStates(view);
        this.updateDashboardContainerVisibility(view);
        this.focusDashboardContainer(view);

        // Initialize and show the selected dashboard
        await this.initializeDashboardView(view);
    }

    /**
     * Destroy current dashboard if switching away
     */
    private destroyCurrentDashboard(): void {
        if (this.currentDashboardView === 'activity' && this.activityDashboard) {
            this.activityDashboard.destroy();
        }
    }

    /**
     * Update tab active states with proper ARIA semantics
     */
    private updateDashboardTabStates(view: DashboardView): void {
        const navTabs = document.querySelectorAll('.nav-tab');
        navTabs.forEach(tab => {
            const tabView = tab.getAttribute('data-view');
            const isActive = tabView === view;
            tab.classList.toggle('active', isActive);
            tab.setAttribute('aria-selected', String(isActive));
            // Only active tab should be in tab order (roving tabindex pattern)
            tab.setAttribute('tabindex', isActive ? '0' : '-1');
        });
    }

    /**
     * Update dashboard container visibility
     */
    private updateDashboardContainerVisibility(view: DashboardView): void {
        const containers = getDashboardContainerElements();
        Object.entries(containers).forEach(([containerView, element]) => {
            if (element) {
                element.style.display = containerView === view ? 'block' : 'none';
            }
        });
    }

    /**
     * Move focus to the new container for screen reader accessibility
     */
    private focusDashboardContainer(view: DashboardView): void {
        const containers = getDashboardContainerElements();
        const targetContainer = containers[view];
        if (targetContainer) {
            // Use requestAnimationFrame to ensure the container is visible before focusing
            requestAnimationFrame(() => {
                targetContainer.focus();
            });
        }
    }

    /**
     * Initialize and show the selected dashboard view
     */
    private async initializeDashboardView(view: DashboardView): Promise<void> {
        try {
            switch (view) {
                case 'activity':
                    await this.initializeActivityView();
                    break;
                case 'server':
                    await this.initializeServerView();
                    break;
                case 'recently-added':
                    await this.initializeRecentlyAddedView();
                    break;
                case 'maps':
                    await this.initializeMapsView();
                    break;
                case 'analytics':
                    await this.initializeAnalyticsView();
                    break;
                case 'cross-platform':
                    this.initializeCrossPlatformView();
                    break;
                case 'data-governance':
                    this.initializeDataGovernanceView();
                    break;
                case 'newsletter':
                    this.initializeNewsletterView();
                    break;
            }
        } catch (error) {
            logger.error(`Failed to initialize ${view} dashboard:`, error);
            if (this.toastManager) {
                this.toastManager.error(`Failed to load ${view} dashboard`, 'Error', 5000);
            }
        }
    }

    /**
     * Initialize activity dashboard view
     */
    private async initializeActivityView(): Promise<void> {
        if (this.activityDashboard) {
            await this.activityDashboard.init();
        }
    }

    /**
     * Initialize server dashboard view
     */
    private async initializeServerView(): Promise<void> {
        if (this.serverDashboard) {
            await this.serverDashboard.init();
        }
    }

    /**
     * Initialize recently added dashboard view
     */
    private async initializeRecentlyAddedView(): Promise<void> {
        if (this.recentlyAddedDashboard) {
            await this.recentlyAddedDashboard.init();
        }
    }

    /**
     * Initialize maps view with WebGL check and viewport handling
     */
    private async initializeMapsView(): Promise<void> {
        // Show WebGL warning if needed
        if (this.viewportManager) {
            this.viewportManager.checkWebGLForMaps();
        }

        // Trigger map resize if needed
        const currentViewMode = this.viewportManager?.getCurrentViewMode() || '2d';

        // Handle 2D map view
        if (currentViewMode === '2d' && this.mapManager) {
            // Map should already be visible, no action needed
            // Keep reference to mapManager to indicate 2D view is map-based
        } else if (currentViewMode === '3d' && this.globeManager) {
            // Handle 3D globe view
            setTimeout(() => {
                if (this.globeManager) {
                    this.globeManager.show();
                }
            }, 50);
        }
    }

    /**
     * Initialize analytics view with chart initialization
     */
    private async initializeAnalyticsView(): Promise<void> {
        // Load analytics page from URL if available, otherwise show current page
        this.loadAnalyticsPageFromURL();

        // Force initialize charts that are now visible
        // This ensures charts render immediately without waiting for
        // IntersectionObserver callbacks (which may be delayed in headless browsers)
        if (this.chartManager) {
            // Delay to ensure container visibility and layout have fully propagated
            // 200ms is needed for CI/headless environments where layout is slower
            setTimeout(() => {
                this.chartManager?.ensureChartsInitialized();
            }, 200);
        }
    }

    /**
     * Initialize cross-platform view
     */
    private initializeCrossPlatformView(): void {
        if (this.onCrossPlatformShow) {
            this.onCrossPlatformShow();
        }
    }

    /**
     * Initialize data governance view
     */
    private initializeDataGovernanceView(): void {
        if (this.onDataGovernanceShow) {
            this.onDataGovernanceShow();
        }
    }

    /**
     * Initialize newsletter view
     */
    private initializeNewsletterView(): void {
        if (this.onNewsletterShow) {
            this.onNewsletterShow();
        }
    }

    /**
     * Switch to a specific analytics page and update URL
     * Supports deep linking: /analytics#page=content
     */
    switchAnalyticsPage(page: AnalyticsPage, updateURL: boolean = true): void {
        this.currentAnalyticsPage = page;

        // Notify breadcrumb manager of page change
        this.breadcrumbManager?.updateAnalyticsPage(page);

        // Show loading overlay for smooth transition
        const loadingOverlay = document.getElementById('analytics-loading-overlay');
        this.showLoadingOverlay(loadingOverlay);

        // Update URL hash for bookmarking/sharing (only if not loading from URL)
        if (updateURL) {
            window.location.hash = `analytics-${page}`;
        }

        // Update UI state
        this.updateAnalyticsTabStates(page);
        this.updateAnalyticsPageVisibility(page);
        this.focusAnalyticsPage(page);

        // Trigger chart initialization and page-specific callbacks
        setTimeout(() => {
            this.finalizeAnalyticsPageSwitch(page, loadingOverlay);
        }, 200); // 200ms delay for smooth transition feel

        logger.debug(`[Analytics] Switched to ${page} page`, { url: window.location.href });
    }

    /**
     * Show loading overlay
     */
    private showLoadingOverlay(loadingOverlay: HTMLElement | null): void {
        if (loadingOverlay) {
            loadingOverlay.classList.add('active');
        }
    }

    /**
     * Update analytics tab active states with proper ARIA semantics
     */
    private updateAnalyticsTabStates(page: AnalyticsPage): void {
        const analyticsTabs = document.querySelectorAll('.analytics-tab');
        analyticsTabs.forEach(tab => {
            const tabPage = tab.getAttribute('data-analytics-page');
            const isActive = tabPage === page;
            tab.classList.toggle('active', isActive);
            tab.setAttribute('aria-selected', String(isActive));
            // Only active tab should be in tab order (roving tabindex pattern)
            tab.setAttribute('tabindex', isActive ? '0' : '-1');
        });
    }

    /**
     * Update analytics page container visibility
     */
    private updateAnalyticsPageVisibility(page: AnalyticsPage): void {
        const pages = getAnalyticsPageElements();
        Object.entries(pages).forEach(([pageName, element]) => {
            if (element) {
                element.style.display = pageName === page ? 'block' : 'none';
            }
        });
    }

    /**
     * Move focus to the new analytics page for screen reader accessibility
     */
    private focusAnalyticsPage(page: AnalyticsPage): void {
        const pages = getAnalyticsPageElements();
        const targetPage = pages[page];
        if (targetPage) {
            // Add tabindex if not present, then focus
            if (!targetPage.hasAttribute('tabindex')) {
                targetPage.setAttribute('tabindex', '-1');
            }
            requestAnimationFrame(() => {
                targetPage.focus();
            });
        }
    }

    /**
     * Finalize analytics page switch with chart initialization and callbacks
     */
    private finalizeAnalyticsPageSwitch(page: AnalyticsPage, loadingOverlay: HTMLElement | null): void {
        // Initialize charts and trigger resize
        this.initializeChartsForPage();

        // Trigger page-specific callbacks
        this.triggerAnalyticsPageCallback(page);

        // Hide loading overlay after page transition completes
        if (loadingOverlay) {
            loadingOverlay.classList.remove('active');
        }
    }

    /**
     * Initialize charts for the current page
     */
    private initializeChartsForPage(): void {
        if (this.chartManager) {
            // Initialize charts that might not have been initialized yet
            // (charts in hidden pages may not have been initialized on first load)
            this.chartManager.ensureChartsInitialized();

            // Also trigger resize for proper dimensions
            window.dispatchEvent(new Event('resize'));
        }
    }

    /**
     * Trigger page-specific initialization callbacks
     */
    private triggerAnalyticsPageCallback(page: AnalyticsPage): void {
        // Map of page-specific callbacks
        const pageCallbacks: Partial<Record<AnalyticsPage, () => void>> = {
            'library': this.onLibraryPageShow,
            'users-profile': this.onUserProfilePageShow,
            'wrapped': this.onWrappedPageShow
        };

        const callback = pageCallbacks[page];
        if (callback) {
            callback();
        }
    }

    /**
     * Load analytics page from URL hash
     * Supports URLs like: /#analytics-content or /#analytics-users
     */
    loadAnalyticsPageFromURL(): void {
        const hash = window.location.hash.substring(1); // Remove the '#'

        // Check if hash starts with 'analytics-'
        if (hash.startsWith('analytics-')) {
            this.loadAnalyticsPageFromHash(hash);
        } else {
            this.loadDefaultAnalyticsPage();
        }
    }

    /**
     * Load analytics page from hash string
     */
    private loadAnalyticsPageFromHash(hash: string): void {
        const pageFromURL = hash.replace('analytics-', '') as AnalyticsPage;

        // Validate that it's a valid analytics page
        if (VALID_ANALYTICS_PAGES.includes(pageFromURL)) {
            logger.debug(`[URL] Loading analytics page from URL: ${pageFromURL}`);
            this.switchAnalyticsPage(pageFromURL, false); // Don't update URL again
        } else {
            logger.warn(`[URL] Invalid analytics page in URL: ${pageFromURL}, defaulting to overview`);
            this.switchAnalyticsPage('overview', true);
        }
    }

    /**
     * Load default analytics page
     */
    private loadDefaultAnalyticsPage(): void {
        const defaultPage = this.currentAnalyticsPage || 'overview';
        this.switchAnalyticsPage(defaultPage, true);
    }

    /**
     * Handle hash change event (browser back/forward buttons)
     * Extracted for proper cleanup in destroy()
     */
    private handleHashChange(): void {
        const hash = window.location.hash.substring(1);

        if (!hash) {
            this.handleEmptyHash();
            return;
        }

        if (hash.startsWith('analytics-')) {
            this.handleAnalyticsHash();
            return;
        }

        this.handleDashboardViewHash(hash);
    }

    /**
     * Handle empty hash (default to maps view)
     */
    private handleEmptyHash(): void {
        if (this.currentDashboardView !== 'maps') {
            this.switchDashboardView('maps');
        }
    }

    /**
     * Handle analytics page hash (e.g., analytics-content)
     */
    private handleAnalyticsHash(): void {
        // Switch to analytics view if not already there
        if (this.currentDashboardView !== 'analytics') {
            // Use a flag to prevent double URL update
            this.currentDashboardView = 'analytics';
            this.updateTabActiveStates('analytics');
            this.showDashboardContainer('analytics');
        }
        this.loadAnalyticsPageFromURL();
    }

    /**
     * Handle main dashboard view hash (maps, activity, server, etc.)
     */
    private handleDashboardViewHash(hash: string): void {
        if (VALID_DASHBOARD_VIEWS.includes(hash as DashboardView)) {
            const view = hash as DashboardView;
            if (this.currentDashboardView !== view) {
                this.switchDashboardView(view);
            }
        }
    }

    /**
     * Set up URL hash change listener for browser back/forward buttons
     * Handles both main dashboard views and analytics sub-pages
     */
    private setupURLHashListener(): void {
        window.addEventListener('hashchange', this.boundHandleHashChange);
    }

    /**
     * Update tab active states without triggering view switch (for hash change handling)
     */
    private updateTabActiveStates(view: DashboardView): void {
        const navTabs = document.querySelectorAll('.nav-tab');
        navTabs.forEach(tab => {
            const tabView = tab.getAttribute('data-view');
            const isActive = tabView === view;
            tab.classList.toggle('active', isActive);
            tab.setAttribute('aria-selected', String(isActive));
            tab.setAttribute('tabindex', isActive ? '0' : '-1');
        });
    }

    /**
     * Show specific dashboard container (for hash change handling)
     */
    private showDashboardContainer(view: DashboardView): void {
        const containers = getDashboardContainerElements();

        Object.entries(containers).forEach(([containerView, element]) => {
            if (element) {
                element.style.display = containerView === view ? 'block' : 'none';
            }
        });
    }

    /**
     * Handle keyboard navigation for analytics pages
     * Extracted for proper cleanup in destroy()
     */
    private handleKeyDown(e: KeyboardEvent): void {
        // Only handle keyboard shortcuts when conditions are met
        if (!this.shouldHandleKeyboardShortcut(e)) {
            return;
        }

        // Handle different keyboard shortcuts
        if (this.isArrowKey(e.key)) {
            this.handleArrowKeyNavigation(e);
        } else if (this.isNumberKey(e.key)) {
            this.handleNumberKeyNavigation(e);
        } else if (this.isHelpKey(e.key)) {
            this.handleHelpKeyPress();
        }
    }

    /**
     * Check if keyboard shortcuts should be handled
     */
    private shouldHandleKeyboardShortcut(e: KeyboardEvent): boolean {
        // Only handle when on analytics view
        if (this.currentDashboardView !== 'analytics') {
            return false;
        }

        // Don't handle when typing in input fields
        if (e.target instanceof HTMLInputElement ||
            e.target instanceof HTMLTextAreaElement ||
            e.target instanceof HTMLSelectElement) {
            return false;
        }

        // Don't handle with modifier keys (Ctrl, Alt, Cmd)
        if (e.ctrlKey || e.metaKey || e.altKey) {
            return false;
        }

        return true;
    }

    /**
     * Check if key is an arrow key
     */
    private isArrowKey(key: string): boolean {
        return key === 'ArrowLeft' || key === 'ArrowRight';
    }

    /**
     * Check if key is a number key (0-9)
     */
    private isNumberKey(key: string): boolean {
        return (key >= '1' && key <= '9') || key === '0';
    }

    /**
     * Check if key is a help key
     */
    private isHelpKey(key: string): boolean {
        return key === 'h' || key === 'H' || key === '?';
    }

    /**
     * Handle arrow key navigation (Left/Right)
     */
    private handleArrowKeyNavigation(e: KeyboardEvent): void {
        e.preventDefault();

        const currentIndex = VALID_ANALYTICS_PAGES.indexOf(this.currentAnalyticsPage);
        const newIndex = this.calculateArrowKeyIndex(e.key, currentIndex);
        const targetPage = VALID_ANALYTICS_PAGES[newIndex];

        this.switchAnalyticsPage(targetPage);
        logger.debug(`[Keyboard Navigation] Switched to ${targetPage} page`);
    }

    /**
     * Calculate new index for arrow key navigation
     */
    private calculateArrowKeyIndex(key: string, currentIndex: number): number {
        if (key === 'ArrowLeft') {
            // Navigate to previous page (wrap around)
            return currentIndex > 0 ? currentIndex - 1 : VALID_ANALYTICS_PAGES.length - 1;
        } else {
            // Navigate to next page (wrap around)
            return currentIndex < VALID_ANALYTICS_PAGES.length - 1 ? currentIndex + 1 : 0;
        }
    }

    /**
     * Handle number key navigation (1-9, 0)
     */
    private handleNumberKeyNavigation(e: KeyboardEvent): void {
        e.preventDefault();

        // 0 key maps to index 9 (wrapped), 1-9 map to indices 0-8
        const pageIndex = e.key === '0' ? 9 : parseInt(e.key) - 1;
        const targetPage = VALID_ANALYTICS_PAGES[pageIndex];

        if (targetPage) {
            this.switchAnalyticsPage(targetPage);
            logger.debug(`[Keyboard Navigation] Jumped to ${targetPage} page (key: ${e.key})`);
        }
    }

    /**
     * Handle help key press (H or ?)
     */
    private handleHelpKeyPress(): void {
        if (this.toastManager) {
            this.toastManager.info(
                'Use ← → arrow keys to navigate pages, or press 1-0 to jump directly:\n1: Overview  2: Content  3: Users  4: Performance  5: Geographic  6: Advanced  7: Library  8: User Profile  9: Tautulli  0: Wrapped',
                'Analytics Keyboard Shortcuts',
                8000
            );
        }
    }

    /**
     * Set up keyboard navigation for analytics pages
     * Arrow keys: Navigate between pages
     * Number keys 1-6: Jump directly to specific pages
     */
    private setupKeyboardNavigation(): void {
        window.addEventListener('keydown', this.boundHandleKeyDown);
        logger.debug('[Keyboard Navigation] Enabled for analytics pages (← → or 1-9, press H for help)');
    }

    /**
     * Set up scroll indicators for analytics navigation
     * Shows gradient indicators when content can be scrolled left/right
     */
    private setupAnalyticsNavScrollIndicators(): void {
        const wrapper = document.getElementById('analytics-nav-wrapper');
        const nav = document.getElementById('analytics-nav');

        if (!wrapper || !nav) {
            logger.warn('Analytics nav wrapper or nav not found');
            return;
        }

        // Store nav element reference for cleanup
        this.analyticsNavElement = nav;

        // Update scroll indicator classes based on scroll position
        const updateScrollIndicators = (): void => {
            const scrollLeft = nav.scrollLeft;
            const maxScrollLeft = nav.scrollWidth - nav.clientWidth;
            const threshold = 5; // Small threshold to avoid floating point issues

            // Can scroll left if not at the beginning
            wrapper.classList.toggle('can-scroll-left', scrollLeft > threshold);

            // Can scroll right if not at the end
            wrapper.classList.toggle('can-scroll-right', scrollLeft < maxScrollLeft - threshold);
        };

        // Store scroll update handler for cleanup
        this.scrollUpdateHandler = updateScrollIndicators;

        // Listen for scroll events on the nav
        nav.addEventListener('scroll', updateScrollIndicators, { passive: true });

        // Also update on resize (viewport changes may affect overflow)
        window.addEventListener('resize', this.boundHandleResize, { passive: true });

        // Initial update
        updateScrollIndicators();

        // Also update when switching to analytics view (nav might not be visible initially)
        this.scrollIndicatorObserver = new MutationObserver(() => {
            setTimeout(updateScrollIndicators, 100);
        });

        const analyticsContainer = document.getElementById('analytics-container');
        if (analyticsContainer) {
            this.scrollIndicatorObserver.observe(analyticsContainer, { attributes: true, attributeFilter: ['class', 'style'] });
        }

        logger.debug('Analytics nav scroll indicators initialized');
    }

    // ========================================================================
    // Role-Based Visibility (RBAC Phase 4)
    // ========================================================================

    /**
     * Apply role-based visibility to navigation tabs.
     * RBAC Phase 4: Frontend Role Integration
     *
     * Hides tabs that require specific roles:
     * - data-governance: Admin only
     *
     * @param adminOnlyViews - Array of view names that require admin role
     * @param editorOnlyViews - Array of view names that require editor role
     */
    applyRoleBasedTabVisibility(
        adminOnlyViews: DashboardView[] = ['data-governance'],
        editorOnlyViews: DashboardView[] = []
    ): void {
        const roleGuard = getRoleGuard();

        // Hide admin-only tabs
        adminOnlyViews.forEach(view => {
            const tab = document.querySelector(`.nav-tab[data-view="${view}"]`) as HTMLElement;
            if (tab) {
                roleGuard.hideIfNotAdmin(tab);
            }
        });

        // Hide editor-only tabs (editors and admins can see)
        editorOnlyViews.forEach(view => {
            const tab = document.querySelector(`.nav-tab[data-view="${view}"]`) as HTMLElement;
            if (tab) {
                roleGuard.showIfRole(tab, 'editor');
            }
        });

        logger.debug('[RBAC] Applied role-based tab visibility', {
            isAdmin: roleGuard.isAdmin(),
            isEditor: roleGuard.isEditor()
        });
    }

    /**
     * Admin-only views lookup
     */
    private readonly adminOnlyViews = new Set<DashboardView>(['data-governance', 'newsletter']);

    /**
     * Check if the current user can access a specific dashboard view.
     * Used to prevent navigation to restricted views.
     *
     * @param view - The dashboard view to check
     * @returns true if the user can access the view
     */
    canAccessView(view: DashboardView): boolean {
        const roleGuard = getRoleGuard();

        // Admin-only views
        if (this.adminOnlyViews.has(view)) {
            return roleGuard.isAdmin();
        }

        // All authenticated users can access other views
        return true;
    }

    /**
     * Check if the current user can access a specific analytics page.
     * Used to prevent navigation to restricted pages.
     *
     * @param _page - The analytics page to check (currently unused - all pages accessible)
     * @returns true if the user can access the page
     */
    canAccessAnalyticsPage(_page: AnalyticsPage): boolean {
        // All analytics pages are accessible to authenticated users
        return true;
    }

    /**
     * Clean up all event listeners and observers
     * Prevents memory leaks when the manager is destroyed
     */
    destroy(): void {
        // Remove window event listeners
        window.removeEventListener('hashchange', this.boundHandleHashChange);
        window.removeEventListener('keydown', this.boundHandleKeyDown);
        window.removeEventListener('resize', this.boundHandleResize);

        // Remove navigation tab listeners
        this.cleanupNavigationListeners();

        // Remove scroll listener from nav element
        if (this.analyticsNavElement && this.scrollUpdateHandler) {
            this.analyticsNavElement.removeEventListener('scroll', this.scrollUpdateHandler);
        }

        // Disconnect mutation observer
        if (this.scrollIndicatorObserver) {
            this.scrollIndicatorObserver.disconnect();
            this.scrollIndicatorObserver = null;
        }

        // Destroy dashboard managers if they exist
        if (this.activityDashboard) {
            this.activityDashboard.destroy();
        }
        if (this.serverDashboard) {
            this.serverDashboard.destroy();
        }
        if (this.recentlyAddedDashboard) {
            this.recentlyAddedDashboard.destroy();
        }

        // Clear references
        this.analyticsNavElement = null;
        this.scrollUpdateHandler = null;

        logger.debug('[NavigationManager] Destroyed and cleaned up event listeners');
    }
}
