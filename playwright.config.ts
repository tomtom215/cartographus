// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright E2E Test Configuration
 *
 * Test coverage:
 * - Login flow with JWT authentication
 * - Chart rendering and interactions
 * - Map visualization and clustering
 * - Filter application (date, user, media type)
 * - WebSocket real-time updates
 * - Data export (CSV and GeoJSON)
 *
 * Test Suite Split:
 * - FAST tests: Non-WebGL tests (settings, auth, filters, dialogs)
 * - SLOW tests: WebGL-heavy tests (maps, globes, charts)
 *
 * Sharding: 4 shards for parallel execution in CI
 */

/**
 * Shared Chrome launch options for headless CI environments.
 * These flags ensure Chrome can run on systems without GPU support.
 */
const chromeLaunchOptions = {
  args: [
    '--use-gl=swiftshader',           // Software WebGL renderer (required for CI)
    '--enable-webgl',                 // Enable WebGL
    '--enable-accelerated-2d-canvas', // Enable canvas acceleration
    '--disable-gpu-sandbox',          // Required for SwiftShader in CI
    '--ignore-gpu-blocklist',         // Bypass GPU blocklist
    '--disable-dev-shm-usage',        // Overcome limited shared memory in CI
    '--no-sandbox',                   // Required for some CI environments
    '--disable-setuid-sandbox',       // Required for some CI environments
  ],
};

/**
 * WebGL-heavy test files (SLOW suite)
 * These tests require full WebGL initialization and take longer to run.
 * Pattern matches: maps, globes, charts, analytics pages with visualizations
 */
const SLOW_TEST_PATTERN = /^(02-charts|03-map|05-websocket|06-globe-deckgl|07-globe-enhanced|12-analytics-pages|13-mobile-responsive|14-theme-accessibility|15-color-scheme|17-bitrate-bandwidth-analytics|19-hexagon-map-mode|21-arc-visualization|22-hardware-transcode|23-abandonment-analytics|27-performance-optimizations|30-fullscreen-mode|31-chart-zoom-tabs|40-globe-theming|41-high-contrast-map|47-user-ip-details|screenshots)\.spec\.ts$/;

/**
 * Non-WebGL test files (FAST suite)
 * These tests don't require heavy WebGL rendering and complete quickly.
 * Excludes: login tests (separate project), slow tests (WebGL)
 */
const FAST_TEST_PATTERN = /^(?!01-login)(04-filters|08-live-activity|09-recently-added|10-server-info|11-data-export|15-plex-realtime|16-mobile-navigation|16-plex-transcode-monitoring|17-websocket-status|18-buffer-health-monitoring|18-trend-indicators|19-oauth-flow|20-automated-insights|20-cursor-pagination|21-plex-webhooks|24-library-deep-dive|25-user-profile-analytics|26-onboarding-welcome|28-filter-badges|29-confirmation-dialogs|32-sparklines|33-stale-data-warning|34-auto-refresh-toggle|35-loading-progress|36-error-boundary|37-filter-presets|38-backup-restore|39-settings-preferences|42-filter-announcements|42-server-management|43-plex-direct-api|44-session-management|45-library-details|46-collections-playlists|48-exports|49-rating-keys|50-metadata-deep-dive|51-search|52-stream-data|53-custom-date-comparison|54-geographic-drilldown|55-streaming-geojson|56-chart-timeline-animation|57-url-state-persistence|58-cross-platform|66-setup-wizard)\.spec\.ts$/;

export default defineConfig({
  testDir: './tests/e2e',

  // =========================================================================
  // TESTS DISABLED: All Playwright E2E tests are temporarily disabled.
  // To re-enable tests, remove or comment out the testIgnore line below.
  // =========================================================================
  testIgnore: '**/*.spec.ts',

  // Global setup/teardown for mock API server
  // The Express mock server starts ONCE before all tests for deterministic,
  // concurrent-safe API mocking. See ADR-0025 for rationale.
  globalSetup: require.resolve('./tests/e2e/global-setup'),
  globalTeardown: require.resolve('./tests/e2e/global-teardown'),

  // Maximum time one test can run
  // Local: 45s, CI: 90s (SwiftShader is slower, need room for retry logic)
  // Note: TIMEOUTS constants are also increased by 1.5x in CI (see constants.ts)
  timeout: process.env.CI ? 90 * 1000 : 45 * 1000,

  // Test configuration
  expect: {
    // Maximum time to wait for assertions (15 seconds in CI for slower rendering)
    timeout: process.env.CI ? 15000 : 10000,
  },

  // Global timeout for the entire test run
  // With 4 shards: ~240 tests per shard, ~30 min per shard with 2 workers
  // Increased to 40 min to accommodate longer individual test timeouts (90s in CI)
  globalTimeout: process.env.CI ? 40 * 60 * 1000 : undefined,

  // Run tests in parallel
  fullyParallel: true,

  // Fail the build on CI if you accidentally left test.only in the source code
  forbidOnly: !!process.env.CI,

  // Retry on CI only (2 retries = 3 total attempts per test for flaky WebGL tests)
  retries: process.env.CI ? 2 : 0,

  // Fail fast when too many tests fail to avoid wasting CI time
  maxFailures: process.env.CI ? 15 : undefined,

  // Number of parallel workers
  // CI: 2 workers for stability (WebGL/SwiftShader is memory-intensive)
  // Local: undefined = use all available cores
  workers: process.env.CI ? 2 : undefined,

  // Reporter to use
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['list'],
    ...(process.env.CI ? [['github'] as const] : []),
  ],

  // Shared settings for all projects
  use: {
    // Base URL for tests
    baseURL: process.env.BASE_URL || 'http://localhost:3857',

    // Action timeout (click, fill, etc.) - 15 seconds in CI
    actionTimeout: process.env.CI ? 15000 : 10000,

    // Navigation timeout - 30 seconds in CI for container startup delays
    navigationTimeout: process.env.CI ? 30000 : 15000,

    // Collect trace when retrying the failed test
    trace: 'on-first-retry',

    // Screenshot on failure
    screenshot: 'only-on-failure',

    // Video on failure
    video: 'retain-on-failure',
  },

  // Configure projects for major browsers
  projects: [
    // Setup project - runs ONCE before all tests to authenticate
    // CRITICAL: Must use same browser config as other projects for CI compatibility
    {
      name: 'setup',
      testMatch: /.*\.setup\.ts/,
      use: {
        ...devices['Desktop Chrome'],
        launchOptions: chromeLaunchOptions,
      },
    },

    // Login tests project - runs WITHOUT authentication (tests the login flow itself)
    // This project has NO storageState and NO dependency on setup
    {
      name: 'login-tests',
      testMatch: /01-login\.spec\.ts/,
      use: {
        ...devices['Desktop Chrome'],
        // NO storageState - tests start unauthenticated
        launchOptions: chromeLaunchOptions,
      },
      // NO dependencies - runs independently
    },

    // =========================================================================
    // FAST TEST SUITE - Non-WebGL tests (settings, auth, filters, dialogs)
    // These tests complete quickly without heavy WebGL initialization
    // Run on: All PRs and main branch
    // =========================================================================
    {
      name: 'chromium-fast',
      testMatch: FAST_TEST_PATTERN,
      use: {
        ...devices['Desktop Chrome'],
        storageState: 'playwright/.auth/user.json',
        launchOptions: chromeLaunchOptions,
      },
      dependencies: ['setup'],
    },

    // =========================================================================
    // SLOW TEST SUITE - WebGL-heavy tests (maps, globes, charts)
    // These tests require full WebGL initialization via SwiftShader
    // Run on: Main branch only (skipped on PRs for faster feedback)
    //
    // Memory optimization settings:
    // - fullyParallel: false - Run tests sequentially within files to limit
    //   memory accumulation from multiple WebGL contexts
    // - autoCleanupWebGL: true - Clean up WebGL resources after each test
    //   to prevent GPU memory exhaustion in SwiftShader
    // =========================================================================
    {
      name: 'chromium-slow',
      testMatch: SLOW_TEST_PATTERN,
      fullyParallel: false, // Run tests sequentially to prevent memory exhaustion
      use: {
        ...devices['Desktop Chrome'],
        storageState: 'playwright/.auth/user.json',
        launchOptions: chromeLaunchOptions,
        // Enable WebGL cleanup for all slow tests (fixture option)
        autoCleanupWebGL: true,
      } as any, // Type assertion needed for custom fixture options
      dependencies: ['setup'],
    },

    // Main chromium project - runs ALL tests (used when not splitting fast/slow)
    {
      name: 'chromium',
      testMatch: /^(?!.*01-login\.spec\.ts).*\.spec\.ts$/,  // Exclude login tests
      use: {
        ...devices['Desktop Chrome'],
        // Use saved authentication state from setup
        storageState: 'playwright/.auth/user.json',
        // Enable WebGL for Mapbox GL JS, deck.gl, and ECharts
        launchOptions: chromeLaunchOptions,
      },
      dependencies: ['setup'], // Run setup project first
    },

    // Firefox browser testing
    {
      name: 'firefox',
      testMatch: /^(?!.*01-login\.spec\.ts).*\.spec\.ts$/,  // Exclude login tests
      use: {
        ...devices['Desktop Firefox'],
        // Use saved authentication state from setup
        storageState: 'playwright/.auth/user.json',
        // Enable WebGL for Mapbox GL JS, deck.gl, and ECharts
        launchOptions: {
          firefoxUserPrefs: {
            'webgl.force-enabled': true,
            'webgl.disabled': false,
          },
        },
      },
      dependencies: ['setup'], // Run setup project first
    },

    // WebKit (Safari) browser testing
    {
      name: 'webkit',
      testMatch: /^(?!.*01-login\.spec\.ts).*\.spec\.ts$/,  // Exclude login tests
      use: {
        ...devices['Desktop Safari'],
        // Use saved authentication state from setup
        storageState: 'playwright/.auth/user.json',
        // Note: WebKit WebGL support is limited in headless mode
        // Some chart tests may be skipped on WebKit
      },
      dependencies: ['setup'], // Run setup project first
    },
  ],

  // Run local dev server before starting tests
  // In CI, containers are started manually by the workflow, so we skip this
  webServer: process.env.CI ? undefined : {
    // Use E2E-specific docker-compose with SEED_MOCK_DATA=true and test credentials
    command: 'docker compose -f docker-compose.e2e.yml up',
    url: 'http://localhost:3857/api/v1/health',
    reuseExistingServer: true,
    timeout: 120 * 1000, // 2 minutes for container startup
  },
});
