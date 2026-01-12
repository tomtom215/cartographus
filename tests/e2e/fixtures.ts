// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Extended Playwright Test Fixtures
 *
 * ARCHITECTURE (ADR-0024: Deterministic E2E Test Mocking)
 * =========================================================
 *
 * This module provides deterministic API mocking using BROWSER CONTEXT routing.
 * The previous implementation used page.route() which had race conditions.
 *
 * KEY CHANGES:
 * 1. Routes are registered at CONTEXT level, not page level
 * 2. Single unified mock handler (mock-server.ts) instead of multiple files
 * 3. Routes are registered BEFORE page creation (deterministic)
 * 4. No more conflicting registrations between auth-mocks and api-mocks
 *
 * WHY CONTEXT ROUTING IS DETERMINISTIC:
 * - context.route() is called before any page is created
 * - All pages in the context inherit the routes
 * - No race between route registration and navigation
 * - Routes persist across page navigations
 *
 * Usage:
 * ```typescript
 * import { test, expect } from './fixtures';
 *
 * test('my test', async ({ page }) => {
 *   // API is automatically mocked
 *   await page.goto('/');
 * });
 * ```
 */

import { test as base, expect, devices, Page, BrowserContext } from '@playwright/test';
import { setupMockServer, resetMockData } from './fixtures/mock-server';
import { cleanupWebGLResources } from './fixtures/helpers';

// =============================================================================
// EXTENDED TEST FIXTURES
// =============================================================================

/**
 * Extended test fixture options
 */
interface TestFixtureOptions {
  /** Auto-mock all API endpoints (enabled by default) */
  autoMockApi: boolean;
  /** Auto-mock map tiles (enabled by default) */
  autoMockTiles: boolean;
  /** Auto-cleanup WebGL resources after each test */
  autoCleanupWebGL: boolean;
  /** Log API requests for debugging (auto-enabled in CI) */
  logRequests: boolean;
  /**
   * Auto-load auth state from playwright/.auth/user.json (enabled by default).
   * Set to false for login tests that need to start unauthenticated.
   */
  autoLoadAuthState: boolean;
}

/**
 * Extended test with deterministic mocking via context-level routing.
 *
 * CRITICAL: API mocking is enabled by default for deterministic tests.
 * The mock server is configured at the CONTEXT level before page creation.
 */
export const test = base.extend<TestFixtureOptions & { mockContext: BrowserContext }>({
  // Default options
  autoMockApi: [true, { option: true }],
  autoMockTiles: [true, { option: true }],
  autoCleanupWebGL: [false, { option: true }],
  // Only log requests if E2E_VERBOSE is set (reduces CI noise significantly)
  logRequests: [process.env.E2E_VERBOSE === 'true', { option: true }],
  // Auto-load auth state for authenticated tests (login tests should disable this)
  autoLoadAuthState: [true, { option: true }],

  /**
   * Custom context fixture that sets up mocking BEFORE page creation.
   *
   * This is the KEY to deterministic mocking:
   * - Routes are registered on the context, not the page
   * - Context routes are inherited by all pages
   * - No race between route registration and navigation
   */
  mockContext: async ({ browser, autoMockApi, autoMockTiles, logRequests, autoLoadAuthState }, use) => {
    // Create a new context with auth state loaded from the default location
    // This ensures tests start authenticated (matching playwright.config.ts storageState setting)
    // The storageState is created by auth.setup.ts and contains auth tokens in localStorage
    let contextOptions: Parameters<typeof browser.newContext>[0] = {};
    const authPath = 'playwright/.auth/user.json';

    // DETERMINISTIC FIX: Only load auth state if autoLoadAuthState is true.
    // Login tests set autoLoadAuthState: false to start unauthenticated.
    // This prevents the fixture from overriding the project-level storageState setting.
    if (autoLoadAuthState) {
      try {
        const fs = await import('fs');

        // In sharded execution, auth.setup.ts runs first due to project dependencies.
        // However, there can be a race condition if the file write is not atomic.
        // Wait for the file to exist and have valid content.
        //
        // FLAKINESS FIX: Use exponential backoff and scale timeout with CI multiplier.
        // Previous implementation had fixed 100ms polling which missed fast writes
        // and 30s fixed timeout which wasn't enough for slow CI environments.
        const ciMultiplier = process.env.CI ? 2 : 1;
        const maxWaitMs = 30000 * ciMultiplier; // 30s locally, 60s in CI
        const startTime = Date.now();
        let fileReady = false;
        let pollDelay = 100; // Start with 100ms, exponentially backoff
        const maxPollDelay = 2000; // Cap at 2s between polls
        let lastFileSize = 0;
        let fileSizeStableCount = 0;
        const fileSizeStableRequired = 2; // File must be same size for 2 consecutive polls

        while (!fileReady && (Date.now() - startTime) < maxWaitMs) {
          if (fs.existsSync(authPath)) {
            try {
              // FLAKINESS FIX: Check file size stability before parsing.
              // This prevents reading partial JSON while file is being written.
              const stats = fs.statSync(authPath);
              const currentSize = stats.size;

              if (currentSize === lastFileSize && currentSize > 0) {
                fileSizeStableCount++;
              } else {
                fileSizeStableCount = 0;
                lastFileSize = currentSize;
              }

              // Only try to parse if file size has been stable
              if (fileSizeStableCount < fileSizeStableRequired) {
                await new Promise(r => setTimeout(r, pollDelay));
                pollDelay = Math.min(pollDelay * 1.5, maxPollDelay);
                continue;
              }

              // Verify the file is valid JSON with required localStorage data
              const content = fs.readFileSync(authPath, 'utf-8');
              const state = JSON.parse(content);

              // Check for auth_token in localStorage origins
              const hasAuthToken = state.origins?.some((origin: { localStorage?: Array<{ name: string }> }) =>
                origin.localStorage?.some((item: { name: string }) => item.name === 'auth_token')
              );

              if (hasAuthToken) {
                contextOptions = { storageState: authPath };
                fileReady = true;
                if (logRequests || process.env.CI) {
                  console.log(`[E2E-FIXTURE] Auth state loaded from ${authPath} (waited ${Date.now() - startTime}ms)`);
                }
              } else {
                // File exists but doesn't have auth_token - might still be writing
                if (logRequests || process.env.CI) {
                  console.log(`[E2E-FIXTURE] Auth file exists but missing auth_token, waiting...`);
                }
                await new Promise(r => setTimeout(r, pollDelay));
                pollDelay = Math.min(pollDelay * 1.5, maxPollDelay);
              }
            } catch (parseError) {
              // File exists but isn't valid JSON yet - might still be writing
              // Reset file size stability counter on parse error
              fileSizeStableCount = 0;
              if (logRequests || process.env.CI) {
                console.log(`[E2E-FIXTURE] Auth file exists but not valid JSON, waiting...`);
              }
              await new Promise(r => setTimeout(r, pollDelay));
              pollDelay = Math.min(pollDelay * 1.5, maxPollDelay);
            }
          } else {
            // File doesn't exist yet
            await new Promise(r => setTimeout(r, pollDelay));
            pollDelay = Math.min(pollDelay * 1.5, maxPollDelay);
          }
        }

        if (!fileReady) {
          // Log warning but continue - tests may still work if login tests
          console.warn(`[E2E-FIXTURE] WARNING: Auth state file not ready after ${maxWaitMs}ms`);
          console.warn(`[E2E-FIXTURE] Tests requiring authentication may fail`);
          console.warn(`[E2E-FIXTURE] Ensure auth.setup.ts ran successfully (check setup project logs)`);
        }
      } catch (error) {
        // Unexpected error during auth state loading
        console.error(`[E2E-FIXTURE] Error loading auth state:`, error);
      }
    } else {
      // autoLoadAuthState is false - start unauthenticated (for login tests)
      if (logRequests || process.env.CI) {
        console.log(`[E2E-FIXTURE] Auth state loading disabled (autoLoadAuthState=false)`);
      }
    }

    const context = await browser.newContext(contextOptions);

    // Setup mock server on context BEFORE any page is created
    await setupMockServer(context, {
      enableApiMocking: autoMockApi,
      enableTileMocking: autoMockTiles,
      logRequests,
    });

    if (logRequests || process.env.CI) {
      console.log(`[E2E-FIXTURE] Context created with mocking: api=${autoMockApi}, tiles=${autoMockTiles}, storageState=${!!contextOptions.storageState}`);
    }

    await use(context);

    // Cleanup
    await context.close();
  },

  /**
   * Override context fixture to use our mocked context.
   * This ensures tests can access cookies and other context-level state correctly.
   * Without this, test({ page, context }) would receive different contexts.
   */
  context: async ({ mockContext }, use) => {
    // Use the mockContext as the context fixture
    // This ensures context.cookies() works correctly in tests like login tests
    await use(mockContext);
  },

  /**
   * Override page fixture to use our mocked context.
   */
  page: async ({ mockContext, autoCleanupWebGL }, use) => {
    // Create page from the mocked context
    const page = await mockContext.newPage();

    // Set localStorage flags BEFORE any navigation using addInitScript
    // This runs before any page JavaScript executes
    await page.addInitScript(() => {
      localStorage.setItem('onboarding_completed', 'true');
      localStorage.setItem('onboarding_skipped', 'true');
      localStorage.setItem('setup_wizard_completed', 'true');
      localStorage.setItem('setup_wizard_skipped', 'true');
      const cookiePreferences = {
        necessary: true,
        analytics: false,
        preferences: false,
        consentGiven: true,
        consentDate: new Date().toISOString(),
      };
      localStorage.setItem('cartographus-cookie-consent', JSON.stringify(cookiePreferences));
    });

    // Use the page
    await use(page);

    // Cleanup
    if (autoCleanupWebGL) {
      await cleanupWebGLResources(page);
    }

    // Reset mock data between tests for isolation
    resetMockData();
  },
});

// =============================================================================
// SPECIALIZED TEST VARIANTS
// =============================================================================

/**
 * Test with real API (no mocking).
 * Use for integration tests against real backend.
 *
 * WARNING: Tests using real API may be slow and flaky.
 */
export const testWithRealApi = test.extend({
  autoMockApi: false,
});

/**
 * Test with real network (no mocking at all).
 * Use for full integration tests.
 */
export const testWithRealNetwork = test.extend({
  autoMockApi: false,
  autoMockTiles: false,
});

/**
 * Test with WebGL cleanup enabled.
 * Use for WebGL-heavy tests (maps, globe, charts).
 */
export const testWithWebGLCleanup = test.extend({
  autoCleanupWebGL: true,
});

/**
 * Test with verbose request logging.
 * Use for debugging API mocking issues.
 */
export const testWithLogging = test.extend({
  logRequests: true,
});

/**
 * @deprecated Use `test` directly. API mocking is now the default.
 */
export const testWithMockApi = test;

/**
 * @deprecated Use `testWithRealNetwork` instead.
 */
export const testWithRealTiles = test.extend({
  autoMockTiles: false,
});

// =============================================================================
// RE-EXPORTS
// =============================================================================

// Playwright utilities
export { expect, devices, Page };

// Mock server utilities (for advanced use cases)
export { setupMockServer, resetMockData, getCurrentMockData } from './fixtures/mock-server';

// Constants
export {
  TIMEOUTS,
  SELECTORS,
  VIEWS,
  ANALYTICS_PAGES,
  TAUTULLI_SUB_TABS,
  CHARTS,
  getTestCredentials,
} from './fixtures/constants';
export type { View, AnalyticsPage, TautulliSubTab } from './fixtures/constants';
export type { MemoryDiagnostics } from './fixtures/helpers';

// Helpers
export {
  // Debug logging
  setupE2EDebugLogging,
  // App initialization
  gotoAppAndWaitReady,
  waitForStylesheetsLoaded,
  waitForNavReady,
  waitForAppReady,
  // CSS variables
  waitForCSSVariablesResolved,
  // Navigation
  navigateToView,
  navigateToAnalyticsPage,
  navigateToTautulliSubTab,
  // Map/Globe
  waitForMapReady,
  waitForGlobeReady,
  // Charts
  waitForChartDataLoaded,
  waitForChartsReady,
  expectChartsVisible,
  expectChartsRendered,
  expectAnalyticsPageChartsVisible,
  // WebGL cleanup
  cleanupWebGLResources,
  // Memory diagnostics
  logMemoryUsage,
  isMemoryPressureHigh,
  // Login & Onboarding
  performLogin,
  clearAuthAndReload,
  skipOnboarding,
  dismissOnboardingModal,
  // Setup Wizard
  dismissSetupWizardModal,
  // Cookie Consent
  skipCookieConsent,
  dismissCookieConsentBanner,
  // WebSocket
  waitForTestHelpers,
  simulateWebSocketMessage,
  isWebSocketConnected,
  // Viewport
  setMobileViewport,
  setTabletViewport,
  setDesktopViewport,
  // Wait pattern helpers
  waitForAnalyticsPageVisible,
  waitForChartRendered,
  waitForChartsRenderedMultiple,
  waitForGlobeViewReady,
  waitForViewTransition,
  waitForDataLoad,
  clickAnalyticsTabAndWait,
  waitForElementClass,
  waitForTooltipVisible,
  // Theme helpers
  waitForThemeChange,
  clickThemeToggleAndWait,
  // Safe navigation helpers
  waitForPendingRequests,
  safeNavigateToView,
  // Map mode helpers
  waitForMapManagersReady,
  setMapVisualizationMode,
  clickMapModeButton,
  enableHexagonMode,
  toggleArcOverlay,
} from './fixtures/helpers';

// Auth mocks (for error filtering only - API mocking is now in mock-server.ts)
export {
  setupConsoleErrorFilter,
  setupPageErrorFilter,
  setupAllErrorFilters,
  KNOWN_BENIGN_ERRORS,
  isBenignError,
  isDeckGLInitError,
} from './fixtures/auth-mocks';

// Legacy map mocks export (for backwards compatibility)
export { setupMapMocking, setupTileMocking, setupGeocoderMocking } from './fixtures/map-mocks';

// Legacy API mocks export (deprecated - use mock-server.ts instead)
export { setupApiMocking, setupEmptyApiMocking, setupErrorApiMocking } from './fixtures/api-mocks';

// =============================================================================
// LEGACY LOGIN HELPER (for backward compatibility)
// =============================================================================

/**
 * @deprecated Use clearAuthAndReload + performLogin from helpers instead
 */
export async function login(page: Page): Promise<void> {
  await page.context().clearCookies();
  await page.goto('/');
  await page.evaluate(() => {
    localStorage.removeItem('auth_token');
    localStorage.removeItem('auth_username');
    localStorage.removeItem('auth_expires_at');
    localStorage.setItem('onboarding_completed', 'true');
    localStorage.setItem('onboarding_skipped', 'true');
    localStorage.setItem('setup_wizard_completed', 'true');
    localStorage.setItem('setup_wizard_skipped', 'true');
    const cookiePreferences = {
      necessary: true,
      analytics: false,
      preferences: false,
      consentGiven: true,
      consentDate: new Date().toISOString(),
    };
    localStorage.setItem('cartographus-cookie-consent', JSON.stringify(cookiePreferences));
  });
  await page.reload();

  const username = process.env.ADMIN_USERNAME || 'admin';
  const password = process.env.ADMIN_PASSWORD || 'E2eTestPass123!';

  await expect(page.locator('#btn-login')).toBeEnabled({ timeout: 5000 });
  await page.fill('input[name="username"]', username);
  await page.fill('input[name="password"]', password);
  await page.evaluate(() => {
    const btn = document.querySelector('button[type="submit"]') as HTMLElement;
    if (btn) btn.click();
  });
  await expect(page.locator('#map')).toBeVisible({ timeout: 10000 });
}
