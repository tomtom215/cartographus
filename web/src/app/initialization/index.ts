// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Application Initialization Module
 *
 * Exports grouped manager initialization functions to reduce index.ts complexity.
 * Each initializer handles a category of related managers.
 */

export { initializeCoreManagers, type CoreManagers } from './CoreInitializer';
export { initializeUIManagers, type UIManagers } from './UIInitializer';
export { initializeDashboardManagers, type DashboardManagers } from './DashboardInitializer';
export { initializeChartManagers, type ChartManagers } from './ChartInitializer';
export { initializeAnalyticsManagers, type AnalyticsManagers } from './AnalyticsInitializer';
export { initializeSecurityManagers, type SecurityManagers } from './SecurityInitializer';
