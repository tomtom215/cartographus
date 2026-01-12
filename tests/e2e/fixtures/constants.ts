// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Shared E2E Test Constants
 *
 * Centralizes timeout values, selectors, and configuration constants
 * to ensure consistency across all test files.
 */

// =============================================================================
// TIMEOUT CONSTANTS
// =============================================================================

/**
 * Timeout values for different operations.
 * Use these instead of hardcoded numbers to maintain consistency.
 *
 * IMPORTANT: The test timeout in playwright.config.ts is 90s in CI.
 * Individual operation timeouts should leave room within this limit.
 *
 * CI NOTE: SwiftShader (software WebGL) is significantly slower than hardware.
 * - Standard operations: 1.5x multiplier
 * - Animation/render operations: 3x multiplier (these are often used with waitForFunction
 *   which needs extra headroom for polling in slow CI environments)
 */
const CI_MULTIPLIER = process.env.CI ? 1.5 : 1;
const CI_ANIMATION_MULTIPLIER = process.env.CI ? 3 : 1;

export const TIMEOUTS = {
  /** Short timeout for quick UI interactions (button clicks, simple waits) - 2s (3s in CI) */
  SHORT: Math.round(2000 * CI_MULTIPLIER),

  /** Navigation timeout for page/view transitions - 5s (7.5s in CI) */
  NAVIGATION: Math.round(5000 * CI_MULTIPLIER),

  /** Medium timeout for navigation and element visibility - 10s (15s in CI) */
  MEDIUM: Math.round(10000 * CI_MULTIPLIER),

  /** Element visibility timeout (alias for MEDIUM) - 10s (15s in CI) */
  ELEMENT_VISIBLE: Math.round(10000 * CI_MULTIPLIER),

  /** Default timeout for most assertions - 20s (30s in CI) */
  DEFAULT: Math.round(20000 * CI_MULTIPLIER),

  /** Long timeout for slow operations (lazy loading, WebGL init) - 25s (37.5s in CI) */
  LONG: Math.round(25000 * CI_MULTIPLIER),

  /** Extended timeout for page navigation - 20s (30s in CI) */
  EXTENDED: Math.round(20000 * CI_MULTIPLIER),

  /** Maximum timeout for setup operations - 60s (90s in CI) */
  MAX: Math.round(60000 * CI_MULTIPLIER),

  /** Small delay for animations and transitions - 300ms (900ms in CI) */
  ANIMATION: Math.round(300 * CI_ANIMATION_MULTIPLIER),

  /** Delay for chart/map rendering - 500ms (1500ms in CI) */
  RENDER: Math.round(500 * CI_ANIMATION_MULTIPLIER),

  /** Delay for data loading - 1s (1.5s in CI) */
  DATA_LOAD: Math.round(1000 * CI_MULTIPLIER),

  /** Delay for WebGL initialization - 2s (6s in CI) */
  WEBGL_INIT: Math.round(2000 * CI_ANIMATION_MULTIPLIER),
} as const;

// =============================================================================
// SELECTOR CONSTANTS
// =============================================================================

/**
 * Common selectors used across tests.
 * Centralized to make refactoring easier if HTML structure changes.
 */
export const SELECTORS = {
  // App structure
  APP: '#app',
  APP_VISIBLE: '#app:not(.hidden)',
  LOGIN_CONTAINER: '#login-container',

  // Navigation
  NAV_TAB: '.nav-tab',
  NAV_TAB_MAPS: '.nav-tab[data-view="maps"]',
  NAV_TAB_ANALYTICS: '.nav-tab[data-view="analytics"]',
  NAV_TAB_ACTIVITY: '.nav-tab[data-view="activity"]',
  NAV_TAB_RECENTLY_ADDED: '.nav-tab[data-view="recently-added"]',
  NAV_TAB_GLOBE: '.nav-tab[data-view="globe"]',
  NAV_TAB_SERVER: '.nav-tab[data-view="server"]',
  NAV_TAB_BY_VIEW: (view: string) => `.nav-tab[data-view="${view}"]`,

  // Dashboard containers
  ACTIVITY_CONTAINER: '#activity-container',
  RECENTLY_ADDED_CONTAINER: '#recently-added-container',

  // Analytics tabs
  ANALYTICS_CONTAINER: '#analytics-container',
  ANALYTICS_TAB: '.analytics-tab',
  ANALYTICS_TAB_BY_PAGE: (page: string) => `.analytics-tab[data-analytics-page="${page}"]`,
  ANALYTICS_PAGE_BY_ID: (page: string) => `#analytics-${page}`,

  // Map
  MAP: '#map',
  MAP_CONTAINER: '#map-container',
  MAP_CANVAS: '#map canvas.maplibregl-canvas',
  MAP_ZOOM_IN: '#map .maplibregl-ctrl-zoom-in',
  MAP_ZOOM_OUT: '#map .maplibregl-ctrl-zoom-out',
  MAP_MODE_POINTS: '#map-mode-points',
  MAP_MODE_CLUSTERS: '#map-mode-clusters',
  MAP_MODE_HEATMAP: '#map-mode-heatmap',
  MAP_MODE_HEXAGONS: '#map-mode-hexagons',

  // View mode toggle (2D/3D)
  VIEW_TOGGLE_CONTROL: '#view-toggle-control',
  VIEW_MODE_2D: '#view-mode-2d',
  VIEW_MODE_3D: '#view-mode-3d',

  // Globe (3D view)
  GLOBE: '#globe',
  GLOBE_CANVAS: '#globe canvas',
  GLOBE_CONTROLS: '#globe-controls',
  GLOBE_AUTO_ROTATE: '#globe-auto-rotate',
  GLOBE_RESET_VIEW: '#globe-reset-view',
  GLOBE_LOADING: '#globe-loading',
  GLOBE_NOT_SUPPORTED: '#globe-not-supported',

  // Charts (ECharts can render to canvas or SVG)
  CHART_ELEMENT: (chartId: string) => `${chartId} canvas, ${chartId} svg`,
  CHART_CONTAINER: '.chart-content',

  // Server
  SERVER_CONTAINER: '#server-container',
  SERVER_HEALTH: '#server-health',
  BACKUP_SECTION: '#backup-section',

  // Stats (main dashboard stats panel)
  STATS: '#stats, #stats-panel, .stats-panel, #stats-container, .stats-container',

  // Authentication
  LOGIN_USERNAME: 'input[name="username"]',
  LOGIN_PASSWORD: 'input[name="password"]',
  LOGIN_SUBMIT: 'button[type="submit"]',
  LOGIN_BUTTON: '#btn-login',
  LOGOUT_BUTTON: '#btn-logout',
  LOGIN_ERROR: '.error-message.show, #login-error.show',

  // Filters
  FILTERS: '#filters',
  FILTER_DAYS: '#filter-days',

  // Action buttons
  BTN_REFRESH: '#btn-refresh',
  BTN_EXPORT_CSV: '#btn-export-csv',
  BTN_EXPORT_GEOJSON: '#btn-export-geojson',

  // Common UI elements
  TOAST: '.toast',
  DIALOG: '.dialog',
  LOADING: '.loading',

  // Theme toggle
  THEME_TOGGLE: '#theme-toggle',
} as const;

// =============================================================================
// VIEW CONSTANTS
// =============================================================================

/**
 * Main navigation views
 * These match the data-view attributes on nav-tab buttons in index.html
 */
export const VIEWS = {
  MAPS: 'maps',
  ACTIVITY: 'activity',
  ANALYTICS: 'analytics',
  RECENTLY_ADDED: 'recently-added',
  SERVER: 'server',
  // Note: GLOBE is not a nav tab - it's a view mode toggle within the maps view
  GLOBE: 'globe',
} as const;

export type View = typeof VIEWS[keyof typeof VIEWS];

/**
 * Analytics page tabs
 */
export const ANALYTICS_PAGES = {
  OVERVIEW: 'overview',
  CONTENT: 'content',
  USERS: 'users',
  PERFORMANCE: 'performance',
  GEOGRAPHIC: 'geographic',
  ADVANCED: 'advanced',
  LIBRARY: 'library',
  USER_PROFILE: 'users-profile',
  TAUTULLI: 'tautulli',
} as const;

export type AnalyticsPage = typeof ANALYTICS_PAGES[keyof typeof ANALYTICS_PAGES];

/**
 * * Tautulli sub-tab identifiers
 */
export const TAUTULLI_SUB_TABS = {
  CHARTS: 'tautulli-charts',
  METADATA: 'metadata-deep-dive',
  RATING_KEYS: 'rating-keys',
  SEARCH: 'search',
  STREAM_DATA: 'stream-data',
} as const;

export type TautulliSubTab = typeof TAUTULLI_SUB_TABS[keyof typeof TAUTULLI_SUB_TABS];

// =============================================================================
// CHART CONSTANTS
// =============================================================================

/**
 * Chart IDs organized by analytics page
 */
export const CHARTS = {
  OVERVIEW: [
    '#chart-trends',
    '#chart-media',
    '#chart-users',
    '#chart-heatmap',
    '#chart-platforms',
    '#chart-players',
  ],
  CONTENT: [
    '#chart-libraries',
    '#chart-ratings',
    '#chart-duration',
    '#chart-years',
    '#chart-codec',
    '#chart-popular-movies',
    '#chart-popular-shows',
    '#chart-popular-episodes',
  ],
  USERS: [
    '#chart-engagement-summary',
    '#chart-engagement-hours',
    '#chart-engagement-days',
    '#chart-completion',
    '#chart-binge-summary',
    '#chart-binge-shows',
    '#chart-binge-users',
    '#chart-watch-parties-summary',
    '#chart-watch-parties-content',
    '#chart-watch-parties-users',
  ],
  PERFORMANCE: [
    '#chart-bandwidth-trends',
    '#chart-bandwidth-transcode',
    '#chart-bandwidth-resolution',
    '#chart-bandwidth-users',
    '#chart-bitrate-distribution',
    '#chart-bitrate-utilization',
    '#chart-bitrate-resolution',
    '#chart-transcode',
    '#chart-resolution',
    '#chart-resolution-mismatch',
    '#chart-hdr-analytics',
    '#chart-audio-analytics',
    '#chart-subtitle-analytics',
    '#chart-connection-security',
    '#chart-concurrent-streams',
    '#chart-pause-patterns',
  ],
  GEOGRAPHIC: [
    '#chart-countries',
    '#chart-cities',
  ],
  ADVANCED: [
    '#chart-comparative-metrics',
    '#chart-comparative-content',
    '#chart-comparative-users',
    '#chart-temporal-heatmap',
  ],
} as const;

// =============================================================================
// TEST CREDENTIALS
// =============================================================================

/**
 * Get test credentials from environment or use defaults
 */
export function getTestCredentials(): { username: string; password: string } {
  return {
    username: process.env.ADMIN_USERNAME || 'admin',
    password: process.env.ADMIN_PASSWORD || 'E2eTestPass123!',
  };
}
