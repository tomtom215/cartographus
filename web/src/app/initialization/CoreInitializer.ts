// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Core Managers Initializer
 *
 * Initializes essential managers: Map, Stats, Toast, WebSocket, Timeline, Filter.
 */

import type { API } from '../../lib/api';
import { MapManager } from '../../lib/map';
import { StatsManager } from '../../lib/stats';
import { TimelineManager } from '../../lib/timeline';
import { WebSocketManager } from '../../lib/websocket';
import { ToastManager } from '../../lib/toast';
import { FilterManager } from '../../lib/filters';
import { TimelineController } from '../TimelineController';
import { MapKeyboardManager } from '../MapKeyboardManager';

export interface CoreManagers {
    mapManager: MapManager;
    statsManager: StatsManager;
    toastManager: ToastManager;
    wsManager: WebSocketManager;
    timelineManager: TimelineManager;
    timelineController: TimelineController;
    filterManager: FilterManager;
    mapKeyboardManager: MapKeyboardManager | null;
}

export interface CoreInitConfig {
    api: API;
    onMapModeChange: () => void;
    onHexagonDataRequest: (resolution: number) => void;
    onArcDataRequest: () => void;
    onFilterChange: () => Promise<void>;
}

/**
 * Initialize core managers required for basic app functionality
 */
export function initializeCoreManagers(config: CoreInitConfig): CoreManagers {
    // Create map manager
    const mapManager = new MapManager('map', config.onMapModeChange);

    // Set up hexagon and arc data request callbacks
    mapManager.setHexagonDataRequestCallback(config.onHexagonDataRequest);
    mapManager.setArcDataRequestCallback(config.onArcDataRequest);

    // Create stats and toast managers
    const statsManager = new StatsManager(config.api);
    const toastManager = new ToastManager();
    const wsManager = new WebSocketManager();

    // Create filter manager
    const filterManager = new FilterManager(config.api);
    filterManager.setFilterChangeCallback(config.onFilterChange);

    // Create timeline controller first (for callbacks)
    const timelineController = new TimelineController(null);

    // Create timeline manager with callbacks
    const timelineManager = new TimelineManager(
        config.api,
        (currentTime, playbacks, totalCount) => timelineController.handleTimelineUpdate(currentTime, playbacks, totalCount),
        (isPlaying) => timelineController.handlePlayStateChange(isPlaying)
    );

    // Set timeline manager reference in controller
    timelineController.setTimelineManager(timelineManager);

    // Initialize map keyboard navigation with delay
    let mapKeyboardManager: MapKeyboardManager | null = null;
    setTimeout(() => {
        const map = mapManager.getMap();
        if (map) {
            mapKeyboardManager = new MapKeyboardManager({
                containerId: 'map',
                map: map,
                type: 'map'
            });
            mapKeyboardManager.init();
        }
    }, 500);

    return {
        mapManager,
        statsManager,
        toastManager,
        wsManager,
        timelineManager,
        timelineController,
        filterManager,
        mapKeyboardManager
    };
}
