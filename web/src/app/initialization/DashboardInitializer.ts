// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Dashboard Managers Initializer
 *
 * Initializes dashboard managers: Activity, Server, Recently Added.
 */

import type { API } from '../../lib/api';
import type { ToastManager } from '../../lib/toast';
import { ActivityDashboardManager } from '../../lib/activity-dashboard';
import { ServerDashboardManager } from '../../lib/server-dashboard';
import { RecentlyAddedDashboardManager } from '../../lib/recently-added-dashboard';
import { PlexMonitoringManager } from '../../lib/plex-monitoring';
import type { StatsManager } from '../../lib/stats';

export interface DashboardManagers {
    activityDashboard: ActivityDashboardManager;
    serverDashboard: ServerDashboardManager;
    recentlyAddedDashboard: RecentlyAddedDashboardManager;
    plexMonitoringManager: PlexMonitoringManager;
}

export interface DashboardInitConfig {
    api: API;
    toastManager: ToastManager;
    statsManager: StatsManager;
}

/**
 * Initialize dashboard managers
 */
export function initializeDashboardManagers(config: DashboardInitConfig): DashboardManagers {
    // Activity dashboard
    const activityDashboard = new ActivityDashboardManager(config.api);
    activityDashboard.setToastManager(config.toastManager);

    // Server dashboard
    const serverDashboard = new ServerDashboardManager(config.api);

    // Recently added dashboard
    const recentlyAddedDashboard = new RecentlyAddedDashboardManager(config.api);

    // Plex monitoring manager
    const plexMonitoringManager = new PlexMonitoringManager(
        config.toastManager,
        config.statsManager
    );

    return {
        activityDashboard,
        serverDashboard,
        recentlyAddedDashboard,
        plexMonitoringManager
    };
}
