// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * UI Managers Initializer
 *
 * Initializes UI enhancement managers: Viewport, Navigation, Sidebar,
 * Notifications, Loading Progress, Breadcrumbs, etc.
 */

import type { MapManager } from '../../lib/map';
import type { ToastManager } from '../../lib/toast';
import type { ActivityDashboardManager } from '../../lib/activity-dashboard';
import type { ServerDashboardManager } from '../../lib/server-dashboard';
import type { RecentlyAddedDashboardManager } from '../../lib/recently-added-dashboard';
import { ViewportManager } from '../ViewportManager';
import { NavigationManager } from '../NavigationManager';
import { SidebarManager } from '../SidebarManager';
import { KeyboardShortcutsManager } from '../KeyboardShortcutsManager';
import { HelpDocumentationManager } from '../HelpDocumentationManager';
import { NotificationCenterManager } from '../NotificationCenterManager';
import { DataCachingIndicatorManager } from '../DataCachingIndicatorManager';
import { LoadingProgressManager } from '../LoadingProgressManager';
import { EmptyStateActionsManager } from '../EmptyStateActionsManager';
import { BreadcrumbNavigationManager } from '../BreadcrumbNavigationManager';
import { RecentSearchesManager } from '../RecentSearchesManager';
import { FilterHistoryManager } from '../FilterHistoryManager';
import { TableViewManager } from '../TableViewManager';
import type { FilterManager } from '../../lib/filters';
import type { API } from '../../lib/api';

export interface UIManagers {
    viewportManager: ViewportManager;
    navigationManager: NavigationManager;
    sidebarManager: SidebarManager;
    keyboardShortcutsManager: KeyboardShortcutsManager;
    helpDocumentationManager: HelpDocumentationManager;
    notificationCenterManager: NotificationCenterManager;
    dataCachingIndicatorManager: DataCachingIndicatorManager;
    loadingProgressManager: LoadingProgressManager;
    emptyStateActionsManager: EmptyStateActionsManager;
    breadcrumbNavigationManager: BreadcrumbNavigationManager;
    recentSearchesManager: RecentSearchesManager;
    filterHistoryManager: FilterHistoryManager;
    tableViewManager: TableViewManager | null;
}

export interface UIInitConfig {
    api: API;
    mapManager: MapManager;
    toastManager: ToastManager;
    filterManager: FilterManager;
    activityDashboard: ActivityDashboardManager;
    serverDashboard: ServerDashboardManager;
    recentlyAddedDashboard: RecentlyAddedDashboardManager;
    navigateTo: (view: string) => void;
    refreshData: () => Promise<void>;
    openHelp: () => void;
}

/**
 * Initialize UI enhancement managers
 */
export function initializeUIManagers(config: UIInitConfig): UIManagers {
    // Viewport manager
    const viewportManager = new ViewportManager(config.mapManager);
    viewportManager.setToastManager(config.toastManager);

    // Navigation manager (chartManager and globeManager set after lazy loading)
    const navigationManager = new NavigationManager(
        config.mapManager,
        null, // globeManager - set after lazy load
        null, // chartManager - set after lazy load
        viewportManager,
        config.activityDashboard,
        config.serverDashboard,
        config.recentlyAddedDashboard,
        config.toastManager
    );

    // Sidebar manager
    const sidebarManager = new SidebarManager();
    sidebarManager.init();

    // Keyboard shortcuts manager
    const keyboardShortcutsManager = new KeyboardShortcutsManager();
    keyboardShortcutsManager.init();

    // Help documentation manager
    const helpDocumentationManager = new HelpDocumentationManager();
    helpDocumentationManager.init();

    // Notification center manager
    const notificationCenterManager = new NotificationCenterManager();
    notificationCenterManager.init();

    // Connect toast manager to notification center
    config.toastManager.setNotificationCallback((type, message, title) => {
        notificationCenterManager.addNotification(type, message, title);
    });

    // Data caching indicator manager
    const dataCachingIndicatorManager = new DataCachingIndicatorManager();
    dataCachingIndicatorManager.init();

    // Connect API to cache indicator
    config.api.setCacheStatusCallback((cached, queryTimeMs) => {
        dataCachingIndicatorManager.updateStatus(cached, queryTimeMs);
    });

    // Loading progress manager
    const loadingProgressManager = new LoadingProgressManager();
    loadingProgressManager.init();

    // Empty state actions manager
    const emptyStateActionsManager = new EmptyStateActionsManager();
    emptyStateActionsManager.init({
        navigateTo: config.navigateTo,
        refreshData: config.refreshData,
        openHelp: config.openHelp,
        selectLibrary: () => {},
        selectUser: () => {}
    });

    // Breadcrumb navigation manager
    const breadcrumbNavigationManager = new BreadcrumbNavigationManager();
    breadcrumbNavigationManager.init({
        navigateToDashboard: (view: string) => {
            if (view === 'maps' || view === 'activity' || view === 'analytics' ||
                view === 'recently-added' || view === 'server') {
                navigationManager.switchDashboardView(view as 'maps' | 'activity' | 'analytics' | 'recently-added' | 'server');
            }
        },
        navigateToAnalyticsPage: (page: string) => {
            navigationManager.switchAnalyticsPage(page as 'overview' | 'content' | 'users' | 'performance' | 'geographic' | 'advanced' | 'library' | 'users-profile' | 'tautulli');
        }
    });

    // Connect navigation manager to breadcrumb manager
    navigationManager.setBreadcrumbManager(breadcrumbNavigationManager);

    // Filter history manager for undo/redo
    const filterHistoryManager = new FilterHistoryManager();
    filterHistoryManager.setRestoreCallback((filter) => {
        config.filterManager.restoreFilterState(filter);
    });

    // Recent searches manager
    const recentSearchesManager = new RecentSearchesManager((type, value) => {
        if (type === 'user') {
            const userFilter = document.getElementById('filter-users') as HTMLSelectElement;
            if (userFilter) {
                userFilter.value = value;
                config.filterManager.onFilterChange();
            }
        } else if (type === 'mediaType') {
            const mediaTypeFilter = document.getElementById('filter-media-types') as HTMLSelectElement;
            if (mediaTypeFilter) {
                mediaTypeFilter.value = value;
                config.filterManager.onFilterChange();
            }
        }
    });

    // Table view manager (delayed initialization)
    let tableViewManager: TableViewManager | null = null;
    setTimeout(() => {
        tableViewManager = new TableViewManager();
    }, 100);

    return {
        viewportManager,
        navigationManager,
        sidebarManager,
        keyboardShortcutsManager,
        helpDocumentationManager,
        notificationCenterManager,
        dataCachingIndicatorManager,
        loadingProgressManager,
        emptyStateActionsManager,
        breadcrumbNavigationManager,
        recentSearchesManager,
        filterHistoryManager,
        tableViewManager
    };
}
