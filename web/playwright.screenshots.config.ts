// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Playwright Configuration for Documentation Screenshots
 *
 * This is a standalone configuration for capturing deterministic screenshots
 * for README and documentation. It is intentionally separate from the E2E
 * test configuration to ensure:
 *
 * 1. Determinism: Fixed viewport, no animations, consistent state
 * 2. Idempotency: Running multiple times produces identical results
 * 3. Observability: Detailed logging and error reporting
 * 4. Durability: Handles failures gracefully, continues with remaining screenshots
 *
 * Usage:
 *   npx playwright test --config=playwright.screenshots.config.ts
 *
 * @see scripts/capture-screenshots.ts for the screenshot capture logic
 */

import { defineConfig, devices } from '@playwright/test';
import * as path from 'path';

// Screenshot output directory (relative to web/)
const SCREENSHOTS_DIR = path.join(__dirname, 'screenshots', 'captured');

// Base URL for the running application
const BASE_URL = process.env.BASE_URL || 'http://localhost:3857';

// Theme configuration from environment (dark, light, dark-and-light, or all-themes)
// When 'dark-and-light', the capture script handles dual-theme capture
const THEME = (process.env.SCREENSHOT_THEME || 'dark') as 'dark' | 'light' | 'dark-and-light';
const DEFAULT_COLOR_SCHEME = THEME === 'light' ? 'light' : 'dark';

// Fixed viewport for consistent screenshots
const VIEWPORT = {
  width: 1920,
  height: 1080,
};

// Timeouts optimized for screenshot capture (not speed)
const TIMEOUTS = {
  // Overall test timeout per screenshot
  test: 120_000, // 2 minutes
  // Expect assertion timeout
  expect: 30_000, // 30 seconds
  // Action timeout (clicks, fills)
  action: 15_000, // 15 seconds
  // Navigation timeout
  navigation: 60_000, // 1 minute (allow for slow Docker startup)
};

export default defineConfig({
  // Test directory - contains the screenshot capture script
  testDir: './scripts',

  // Only run screenshot tests
  testMatch: 'capture-screenshots.ts',

  // Fail fast on first error in CI, continue locally for debugging
  maxFailures: process.env.CI ? 1 : 0,

  // No parallelism - screenshots must be captured sequentially for WebGL stability
  fullyParallel: false,
  workers: 1,

  // No retries - screenshots should be deterministic
  retries: 0,

  // Timeouts
  timeout: TIMEOUTS.test,
  expect: {
    timeout: TIMEOUTS.expect,
  },

  // Reporter configuration
  reporter: process.env.CI
    ? [
        ['list'],
        ['html', { outputFolder: 'screenshot-report', open: 'never' }],
        ['json', { outputFile: 'screenshot-results.json' }],
      ]
    : [['list']],

  // Global setup/teardown not needed - app should be running externally
  use: {
    // Base URL for navigation
    baseURL: BASE_URL,

    // Fixed viewport for consistent screenshots
    viewport: VIEWPORT,

    // Timeouts
    actionTimeout: TIMEOUTS.action,
    navigationTimeout: TIMEOUTS.navigation,

    // Capture artifacts on failure for debugging
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
    video: 'retain-on-failure',

    // Browser context settings
    colorScheme: DEFAULT_COLOR_SCHEME, // Configurable via SCREENSHOT_THEME env var
    locale: 'en-US',
    timezoneId: 'America/New_York',

    // Reduce flakiness by disabling animations
    // Note: CSS animations are handled in the screenshot script
  },

  // Output directories
  outputDir: path.join(__dirname, 'screenshot-artifacts'),

  // Single browser project - Chromium with WebGL support
  projects: [
    {
      name: 'screenshots',
      use: {
        ...devices['Desktop Chrome'],
        viewport: VIEWPORT,
        // Chrome-specific options for WebGL in headless mode
        launchOptions: {
          args: [
            // WebGL support via SwiftShader (software renderer)
            '--use-gl=swiftshader',
            '--enable-webgl',
            '--enable-webgl2',

            // Disable GPU blocklist to allow WebGL
            '--ignore-gpu-blocklist',
            '--disable-gpu-sandbox',

            // Stability options for CI
            '--disable-dev-shm-usage',
            '--no-sandbox',
            '--disable-setuid-sandbox',

            // Disable features that cause flakiness
            '--disable-background-timer-throttling',
            '--disable-backgrounding-occluded-windows',
            '--disable-renderer-backgrounding',

            // Consistent rendering
            '--force-device-scale-factor=1',
            '--high-dpi-support=1',
          ],
        },
      },
    },
  ],

  // Metadata for reporting
  metadata: {
    purpose: 'Documentation Screenshots',
    viewport: `${VIEWPORT.width}x${VIEWPORT.height}`,
    outputDir: SCREENSHOTS_DIR,
  },
});
