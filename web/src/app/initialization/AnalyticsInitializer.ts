// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Analytics Managers Initializer
 *
 * Initializes analytics and insights managers: Insights, QuickInsights,
 * ComparativeInsights, CustomDateComparison, GeographicDrillDown,
 * StreamingGeoJSON, AnomalyDetection.
 */

import type { API } from '../../lib/api';
import type { FilterManager } from '../../lib/filters';
import type { NavigationManager } from '../NavigationManager';
import { InsightsManager } from '../InsightsManager';
import { QuickInsightsManager } from '../QuickInsightsManager';
import { ComparativeInsightsManager } from '../ComparativeInsightsManager';
import { CustomDateComparisonManager } from '../CustomDateComparisonManager';
import { GeographicDrillDownManager } from '../GeographicDrillDownManager';
import { StreamingGeoJSONManager } from '../StreamingGeoJSONManager';
import { AnomalyDetectionManager } from '../AnomalyDetectionManager';
import { LibraryAnalyticsManager } from '../LibraryAnalyticsManager';
import { UserProfileManager } from '../UserProfileManager';

export interface AnalyticsManagers {
    insightsManager: InsightsManager;
    quickInsightsManager: QuickInsightsManager;
    comparativeInsightsManager: ComparativeInsightsManager;
    customDateComparisonManager: CustomDateComparisonManager;
    geographicDrillDownManager: GeographicDrillDownManager;
    streamingGeoJSONManager: StreamingGeoJSONManager;
    anomalyDetectionManager: AnomalyDetectionManager;
    libraryAnalyticsManager: LibraryAnalyticsManager;
    userProfileManager: UserProfileManager;
}

export interface AnalyticsInitConfig {
    api: API;
    filterManager: FilterManager;
    navigationManager: NavigationManager;
}

/**
 * Initialize analytics and insights managers
 */
export async function initializeAnalyticsManagers(config: AnalyticsInitConfig): Promise<AnalyticsManagers> {
    // Insights manager
    const insightsManager = new InsightsManager(config.api);
    insightsManager.init('insights-panel');

    // Quick insights manager
    const quickInsightsManager = new QuickInsightsManager(config.api);
    quickInsightsManager.init();

    // Comparative insights manager
    const comparativeInsightsManager = new ComparativeInsightsManager(config.api);
    comparativeInsightsManager.init();

    // Custom date comparison manager
    const customDateComparisonManager = new CustomDateComparisonManager(config.api);
    customDateComparisonManager.init();

    // Wire up callback to update comparative insights
    customDateComparisonManager.setOnComparisonLoad((data) => {
        comparativeInsightsManager.updateWithData(data);
    });

    // Geographic drill-down manager
    const geographicDrillDownManager = new GeographicDrillDownManager(config.api);
    await geographicDrillDownManager.init();

    // Streaming GeoJSON manager
    const streamingGeoJSONManager = new StreamingGeoJSONManager(config.api);
    streamingGeoJSONManager.init();

    // Anomaly detection manager
    const anomalyDetectionManager = new AnomalyDetectionManager(config.api);
    anomalyDetectionManager.init();

    // Library analytics manager
    const libraryAnalyticsManager = new LibraryAnalyticsManager(config.api);
    libraryAnalyticsManager.setFilterManager(config.filterManager);

    // Set callback for when library page is shown
    config.navigationManager.setLibraryPageShowCallback(() => {
        libraryAnalyticsManager.init();
    });

    // User profile manager
    const userProfileManager = new UserProfileManager(config.api);
    userProfileManager.setFilterManager(config.filterManager);

    // Set callback for when user profile page is shown
    config.navigationManager.setUserProfilePageShowCallback(() => {
        userProfileManager.init();
    });

    return {
        insightsManager,
        quickInsightsManager,
        comparativeInsightsManager,
        customDateComparisonManager,
        geographicDrillDownManager,
        streamingGeoJSONManager,
        anomalyDetectionManager,
        libraryAnalyticsManager,
        userProfileManager
    };
}
