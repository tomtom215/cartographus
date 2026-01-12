// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Shared E2E Test Helpers
 *
 * Provides reusable helper functions for common test operations:
 * - Navigation between views and tabs
 * - Waiting for specific states
 * - Chart and map assertions
 * - WebGL cleanup
 * - Login operations
 */

import { Page, expect } from '@playwright/test';
import {
  TIMEOUTS,
  SELECTORS,
  VIEWS,
  CHARTS,
  ANALYTICS_PAGES,
  getTestCredentials,
  type View,
  type AnalyticsPage,
  type TautulliSubTab,
} from './constants';

// =============================================================================
// E2E DEBUG LOGGING
// =============================================================================

/**
 * Enable E2E debug network logging on a page.
 * This logs all requests, responses, and failures to help diagnose CI issues.
 *
 * Call this at the beginning of a test or in beforeEach:
 * ```typescript
 * test.beforeEach(async ({ page }) => {
 *   setupE2EDebugLogging(page);
 * });
 * ```
 *
 * Enable via environment variable: E2E_DEBUG=true
 */
export function setupE2EDebugLogging(page: Page): void {
  const e2eDebug = process.env.E2E_DEBUG === 'true';
  if (!e2eDebug) return;

  console.log('[E2E-DEBUG] Network logging enabled for page');

  // Track request counts for summary
  let requestCount = 0;
  let responseCount = 0;
  let failedCount = 0;

  // Log all requests
  page.on('request', request => {
    requestCount++;
    // Only log API requests to reduce noise (skip static assets)
    const url = request.url();
    if (url.includes('/api/') || url.includes('/auth/')) {
      console.log(`[E2E-DEBUG] >> ${request.method()} ${url}`);
    }
  });

  // Log all responses
  page.on('response', response => {
    responseCount++;
    const url = response.url();
    // Only log API responses and errors to reduce noise
    if (url.includes('/api/') || url.includes('/auth/') || response.status() >= 400) {
      const contentType = response.headers()['content-type'] || 'unknown';
      console.log(`[E2E-DEBUG] << ${response.status()} ${url} (${contentType})`);
    }
  });

  // Log all failed requests with detailed error info
  page.on('requestfailed', request => {
    failedCount++;
    const failure = request.failure();
    console.error(`[E2E-DEBUG] XX FAILED: ${request.method()} ${request.url()}`);
    console.error(`[E2E-DEBUG]    Error: ${failure?.errorText || 'unknown'}`);
    console.error(`[E2E-DEBUG]    Resource type: ${request.resourceType()}`);
  });

  // Log network summary at end of test
  page.on('close', () => {
    console.log('[E2E-DEBUG] === Network Summary ===');
    console.log(`[E2E-DEBUG] Total requests: ${requestCount}`);
    console.log(`[E2E-DEBUG] Total responses: ${responseCount}`);
    console.log(`[E2E-DEBUG] Failed requests: ${failedCount}`);
  });
}

// =============================================================================
// APP INITIALIZATION HELPERS
// =============================================================================

/**
 * Wait for all async-loaded stylesheets to be fully loaded.
 *
 * The app uses async stylesheet loading (media="print" with class="async-css" and
 * JavaScript event listeners) for better performance. However, tests must wait for
 * these to be applied before asserting on CSS-dependent properties like visibility,
 * transform, and dimensions.
 *
 * ROOT CAUSE FIX: This addresses chart visibility, touch target sizes,
 * and mobile sidebar transform which all failed because tests ran before
 * styles.css finished loading asynchronously.
 *
 * NOTE: This function is designed to be fast and resilient in CI environments.
 * It uses short timeouts and gracefully handles page closure during teardown.
 *
 * @param page - Playwright page
 * @param timeout - Maximum time to wait for stylesheets (default: 3 seconds in CI, 5 seconds locally)
 */
export async function waitForStylesheetsLoaded(page: Page, timeout: number = process.env.CI ? 3000 : 5000): Promise<void> {
  // Check if page is still open before proceeding
  const isPageOpen = await page.evaluate(() => true).catch(() => false);
  if (!isPageOpen) {
    return; // Page already closed, skip
  }

  // First, force a reflow to ensure CSS media queries are evaluated for current viewport
  try {
    await page.evaluate(() => {
      // Force layout recalculation
      document.body.offsetHeight;
      // Trigger resize event to ensure media queries are re-evaluated
      window.dispatchEvent(new Event('resize'));
    });
  } catch {
    return; // Page closed during evaluation
  }

  // Wait for stylesheets to load with a short timeout
  await page.waitForFunction(
    () => {
      // Check that all stylesheets have finished loading
      // Async-loaded stylesheets start with media="print" and change to media="all" on load
      const stylesheets = document.querySelectorAll('link[rel="stylesheet"]');
      for (const link of stylesheets) {
        const stylesheet = link as HTMLLinkElement;
        // If any stylesheet still has media="print", it hasn't loaded yet
        if (stylesheet.media === 'print') {
          return false;
        }
        // Check if stylesheet is actually loaded by checking if sheet is accessible
        // Note: Cross-origin stylesheets may throw on cssRules access, which is OK
        if (stylesheet.sheet === null && stylesheet.href) {
          // Sheet not loaded yet
          return false;
        }
      }
      return true;
    },
    { timeout }
  ).catch(() => {
    // If timeout, log warning but continue - some stylesheets may be blocked in CI
    console.warn('Warning: Stylesheet load timeout - some styles may not be applied');
  });

  // Force another reflow after stylesheets load to ensure computed styles are updated
  // Wrap in try-catch in case page was closed during the wait above
  try {
    await page.evaluate(() => {
      document.body.offsetHeight;
    });
  } catch {
    return; // Page closed, skip remaining steps
  }

  // In CI, skip the CSS verification step to save time - the stylesheet wait above is sufficient
  if (process.env.CI) {
    return;
  }

  // Additional verification (local only): check that a known CSS property is applied
  // This ensures styles.css specifically has been applied
  const cssVerified = await page.waitForFunction(
    () => {
      // Force reflow before checking computed styles
      document.body.offsetHeight;

      // Check for a specific CSS rule from styles.css to verify it's loaded
      // The theme-toggle should have min-height: 44px from controls.css
      const themeToggle = document.getElementById('theme-toggle');
      if (themeToggle) {
        const styles = getComputedStyle(themeToggle);
        const minHeight = parseInt(styles.minHeight, 10);
        // If min-height is at least 40px, styles.css is likely loaded
        if (minHeight >= 40) {
          return true;
        }
      }
      // Also check sidebar transform on mobile as another indicator
      const sidebar = document.getElementById('sidebar');
      if (sidebar && window.innerWidth <= 768) {
        const styles = getComputedStyle(sidebar);
        // Force recalculation of styles
        sidebar.offsetHeight;
        // If transform is set (not 'none'), mobile styles are applied
        // Mobile sidebar should have translateX(-100%) when hidden
        if (styles.transform !== 'none' && styles.transform !== 'matrix(1, 0, 0, 1, 0, 0)') {
          return true;
        }
      }
      // Fallback: check if any non-inline stylesheet has loaded
      const sheets = document.styleSheets;
      for (const sheet of sheets) {
        try {
          // Try to access rules - this will throw for cross-origin but work for local
          if (sheet.href && sheet.href.includes('styles.css') && sheet.cssRules) {
            return true;
          }
        } catch {
          // Cross-origin sheet, ignore
        }
      }
      return false;
    },
    { timeout: 2000 }
  ).then(() => true).catch(() => false);

  if (!cssVerified) {
    console.warn('[E2E] waitForStylesheetsLoaded: CSS property verification timed out - styles may not be fully applied');
  }

  // Final reflow to ensure all styles are computed
  try {
    await page.evaluate(() => {
      document.body.offsetHeight;
    });
  } catch {
    // Page closed, ignore
  }
}

/**
 * Wait for CSS custom properties (variables) to be properly resolved.
 *
 * ROOT CAUSE FIX: This addresses color theme assertion failures where tests
 * failed because CSS variables weren't resolved yet when color assertions ran.
 *
 * FLAKINESS FIX: Changed from soft-fail (catch and continue) to hard-fail (throw).
 * Previously, timeout was silently swallowed causing subsequent color assertions
 * to fail with confusing error messages. Now throws with clear diagnostic info.
 *
 * @param page - Playwright page
 * @param expectedHighlight - Expected value for --highlight (e.g., '#7c3aed')
 * @param timeout - Maximum time to wait (default: 5 seconds, scaled for CI)
 * @throws Error if CSS variables are not resolved within timeout
 */
export async function waitForCSSVariablesResolved(
  page: Page,
  expectedHighlight: string = '#7c3aed',
  timeout: number = process.env.CI ? 7500 : 5000 // Scale for CI
): Promise<void> {
  try {
    await page.waitForFunction(
      (expected) => {
        const highlight = getComputedStyle(document.documentElement)
          .getPropertyValue('--highlight')
          .trim()
          .toLowerCase();
        // Accept either hex or rgb format
        // #7c3aed = rgb(124, 58, 237)
        const expectedLower = expected.toLowerCase();
        return highlight === expectedLower ||
               highlight === 'rgb(124, 58, 237)' ||
               highlight.includes(expectedLower.replace('#', ''));
      },
      expectedHighlight,
      { timeout }
    );
  } catch (error) {
    // FLAKINESS FIX: Get current value for diagnostic error message
    const actualValue = await page.evaluate(() => {
      return getComputedStyle(document.documentElement)
        .getPropertyValue('--highlight')
        .trim();
    }).catch(() => '<unable to read>');

    throw new Error(
      `CSS variable --highlight not resolved within ${timeout}ms. ` +
      `Expected: ${expectedHighlight}, Actual: ${actualValue}. ` +
      `This may indicate stylesheets haven't loaded or theme wasn't applied.`
    );
  }
}

/**
 * Navigate to the app and wait for it to be ready.
 * This is the standard way to start a test.
 *
 * IMPORTANT: This function handles both authenticated and unauthenticated states.
 * - For authenticated tests (using storageState), it waits for #app to become visible
 * - For unauthenticated tests, it waits for the login form to be visible
 *
 * DETERMINISM FIX (2025-12-31):
 * - Added retry logic for empty page content scenario
 * - Added explicit network idle wait before DOM checks
 * - Improved error diagnostics with full page state capture
 *
 * @param page - Playwright page
 * @param options - Configuration options
 */
export async function gotoAppAndWaitReady(
  page: Page,
  options: {
    waitForNav?: boolean;
    waitForLazyLoad?: boolean;
    timeout?: number;
    expectAuth?: boolean; // If true, expect authenticated state; if false, expect login
  } = {}
): Promise<void> {
  // DISABLED: waitForLazyLoad is disabled because __lazyLoadComplete is never set in CI
  // The flag should be set by the frontend after lazy loading completes, but it's broken
  // Tests will proceed without waiting and rely on Playwright's auto-waiting for elements
  const { waitForNav = true, waitForLazyLoad = false, timeout = TIMEOUTS.DEFAULT, expectAuth = true } = options;

  // IMPORTANT: Set onboarding, setup wizard, and cookie consent flags BEFORE navigating to prevent modals from appearing
  // This uses addInitScript which runs before any page JavaScript executes
  await page.addInitScript(() => {
    localStorage.setItem('onboarding_completed', 'true');
    localStorage.setItem('onboarding_skipped', 'true');
    // Skip setup wizard to prevent it from blocking UI interactions (root cause of E2E failures)
    // The setup wizard modal intercepts all pointer events when visible
    localStorage.setItem('setup_wizard_completed', 'true');
    localStorage.setItem('setup_wizard_skipped', 'true');
    // Skip cookie consent banner to prevent it from blocking UI interactions
    const cookiePreferences = {
      necessary: true,
      analytics: false,
      preferences: false,
      consentGiven: true,
      consentDate: new Date().toISOString(),
    };
    localStorage.setItem('cartographus-cookie-consent', JSON.stringify(cookiePreferences));
  });

  // Navigate with a shorter timeout first - if server isn't responding, fail fast
  try {
    await page.goto('/', { waitUntil: 'commit', timeout: TIMEOUTS.MEDIUM });
  } catch (error) {
    throw new Error(`Failed to navigate to app: server may not be running. Error: ${error}`);
  }

  // Wait for DOM to be interactive (script execution complete)
  await page.waitForLoadState('domcontentloaded', { timeout: TIMEOUTS.DEFAULT });

  // DETERMINISM FIX: Wait for network to settle before checking DOM
  // This ensures all initial API calls and assets have completed
  try {
    await page.waitForLoadState('networkidle', { timeout: TIMEOUTS.MEDIUM });
  } catch {
    // Network may not fully idle if WebSocket keeps reconnecting - continue anyway
    console.log('[E2E] gotoAppAndWaitReady: Network did not reach idle state, continuing...');
  }

  // CRITICAL: Wait for async-loaded stylesheets to be applied
  // This fixes chart visibility, touch target sizes, and mobile sidebar transform
  // which all failed due to CSS timing issues. The app loads styles.css asynchronously
  // for performance, but tests must wait
  await waitForStylesheetsLoaded(page);

  // Check what state the app is in
  const appLocator = page.locator(SELECTORS.APP_VISIBLE);
  const loginLocator = page.locator(SELECTORS.LOGIN_CONTAINER);

  // DETERMINISM FIX: Add retry logic for app visibility check
  // In CI with SwiftShader, the initial render can be slow and the check may fail
  // We retry the visibility check up to 3 times with short delays
  let appVisible = false;
  let loginVisible = false;
  const maxRetries = 3;
  const retryDelay = 500;

  for (let attempt = 0; attempt < maxRetries; attempt++) {
    try {
      // Wait for either the app or login form to be visible
      await Promise.race([
        appLocator.waitFor({ state: 'visible', timeout: timeout / maxRetries }),
        loginLocator.waitFor({ state: 'visible', timeout: timeout / maxRetries }),
      ]);

      // Check which one became visible
      appVisible = await appLocator.isVisible().catch(() => false);
      loginVisible = await loginLocator.isVisible().catch(() => false);

      if (appVisible || loginVisible) {
        break; // Success - exit retry loop
      }
    } catch (error) {
      if (attempt < maxRetries - 1) {
        // Not the last attempt - wait and retry
        console.log(`[E2E] gotoAppAndWaitReady: Attempt ${attempt + 1} failed, retrying in ${retryDelay}ms...`);
        await new Promise(resolve => setTimeout(resolve, retryDelay));
      }
    }
  }

  // Final check - if still not visible, capture diagnostics and fail
  if (!appVisible && !loginVisible) {
    // Capture comprehensive diagnostics for debugging
    const diagnostics = await page.evaluate(() => {
      const body = document.body;
      return {
        url: window.location.href,
        title: document.title,
        bodyHTML: body?.innerHTML?.substring(0, 1000) || 'no body',
        bodyChildCount: body?.children?.length || 0,
        appElement: document.getElementById('app')?.outerHTML?.substring(0, 200) || 'not found',
        loginElement: document.getElementById('login-container')?.outerHTML?.substring(0, 200) || 'not found',
        documentReadyState: document.readyState,
        hasLocalStorage: !!localStorage.getItem('auth_token'),
      };
    }).catch(() => ({ error: 'Failed to capture diagnostics' }));

    console.error('[E2E] gotoAppAndWaitReady DIAGNOSTIC:', JSON.stringify(diagnostics, null, 2));

    throw new Error(
      `Neither app nor login form visible after ${timeout}ms with ${maxRetries} retries. ` +
      `URL: ${(diagnostics as any).url || 'unknown'}, ` +
      `Ready state: ${(diagnostics as any).documentReadyState || 'unknown'}, ` +
      `Body children: ${(diagnostics as any).bodyChildCount || 0}`
    );
  }

  // If we expected auth but got login, the authentication state is invalid
  if (expectAuth && !appVisible && loginVisible) {
    throw new Error(
      'Expected authenticated state but got login form. ' +
      'StorageState may be missing or token may be expired. ' +
      'Ensure auth.setup.ts ran successfully and created valid auth state.'
    );
  }

  // If app is visible (authenticated), continue with initialization
  if (appVisible) {
    // Double-check onboarding and setup wizard flags are set (in case addInitScript didn't run)
    await page.evaluate(() => {
      localStorage.setItem('onboarding_completed', 'true');
      localStorage.setItem('onboarding_skipped', 'true');
      localStorage.setItem('setup_wizard_completed', 'true');
      localStorage.setItem('setup_wizard_skipped', 'true');
    });

    // Dismiss setup wizard modal if it's visible (CRITICAL: blocks all UI interactions)
    // The setup wizard intercepts all pointer events when visible
    const setupWizardModal = page.locator('#setup-wizard-modal');
    const setupWizardVisible = await setupWizardModal.isVisible().catch(() => false);
    if (setupWizardVisible) {
      console.log('[E2E] gotoAppAndWaitReady: Setup wizard modal detected, dismissing...');
      // Try clicking the skip button first
      const skipWizardBtn = page.locator('#wizard-skip-btn');
      if (await skipWizardBtn.isVisible().catch(() => false)) {
        await skipWizardBtn.click().catch(() => {});
      } else {
        // Try pressing Escape as fallback
        await page.keyboard.press('Escape');
      }
      // Wait for modal to be hidden
      const wizardHidden = await setupWizardModal.waitFor({ state: 'hidden', timeout: TIMEOUTS.SHORT }).then(() => true).catch(() => false);
      if (!wizardHidden) {
        console.warn('[E2E] gotoAppAndWaitReady: Setup wizard modal not hidden after dismissal attempts');
      }
    }

    // Dismiss onboarding modal if it's visible (may appear due to timing race)
    // Try multiple methods to ensure it's dismissed
    const onboardingModal = page.locator('#onboarding-modal');
    const modalVisible = await onboardingModal.isVisible().catch(() => false);
    if (modalVisible) {
      // Try clicking the skip button first
      const skipBtn = page.locator('#onboarding-skip-btn, [data-action="skip-onboarding"], .onboarding-skip');
      const skipVisible = await skipBtn.isVisible().catch(() => false);
      if (skipVisible) {
        const skipClicked = await skipBtn.click().then(() => true).catch(() => false);
        if (!skipClicked) {
          console.warn('[E2E] gotoAppAndWaitReady: Skip button click failed');
        }
      }

      // Try pressing Escape as fallback
      await page.keyboard.press('Escape');

      // Wait for modal to be hidden
      const modalHidden = await onboardingModal.waitFor({ state: 'hidden', timeout: TIMEOUTS.SHORT }).then(() => true).catch(() => false);
      if (!modalHidden) {
        // If modal still visible, try clicking outside it
        console.warn('[E2E] gotoAppAndWaitReady: Modal not hidden after Escape, trying click outside');
        const clickedOutside = await page.mouse.click(10, 10).then(() => true).catch(() => false);
        if (!clickedOutside) {
          console.warn('[E2E] gotoAppAndWaitReady: Click outside modal failed');
        }
      }
    }

    if (waitForNav) {
      await waitForNavReady(page);
    }

    // Wait for lazy-loaded modules (ChartManager, GlobeManager) to be ready
    // DISABLED BY DEFAULT: The __lazyLoadComplete flag is never set in CI (broken)
    // When enabled, this waits for the flag which causes 60s+ delays before timeout
    if (waitForLazyLoad) {
      await page.waitForFunction(
        () => (window as any).__lazyLoadComplete === true,
        { timeout: TIMEOUTS.LONG }
      ).catch(() => {
        // Lazy loading may not be flagged in older builds, continue anyway
        // Log for debugging
        console.log('Note: __lazyLoadComplete flag not set within timeout');
      });
    } else {
      // Wait for app managers to be initialized after becoming visible
      //
      // FLAKINESS FIX: Changed from soft-fail to hard-fail with better check.
      // Previous check looked for aria-selected which is in the HTML from the start,
      // so it always passed immediately even if JS hadn't initialized.
      // Now we check for window.__app which is only set after JS fully initializes.
      //
      // CRITICAL: window.__app is set at the END of the app initialization chain,
      // after all managers (Navigation, Chart, Globe, etc.) are created.
      // This ensures tests don't click elements before event handlers are attached.
      try {
        await page.waitForFunction(
          () => {
            // Check if app JavaScript has fully initialized
            // window.__app is set AFTER all managers are created (line ~2206 in index.ts)
            const appInitialized = typeof (window as any).__app !== 'undefined';

            // Check if app container is visible
            const appContainer = document.getElementById('app');
            const hasApp = appContainer && appContainer.style.display !== 'none';

            return appInitialized && hasApp;
          },
          { timeout: TIMEOUTS.MEDIUM } // 10s locally, 15s in CI - enough for JS init
        );
      } catch (error) {
        // FLAKINESS FIX: Hard fail with diagnostic info instead of silent continue
        const diagnostics = await page.evaluate(() => {
          return {
            hasApp: !!(window as any).__app,
            appContainerVisible: document.getElementById('app')?.style.display !== 'none',
            navTabCount: document.querySelectorAll('.nav-tab').length,
            bodyLoaded: document.readyState,
          };
        }).catch(() => ({ error: 'Failed to gather diagnostics' }));

        throw new Error(
          `App JavaScript did not initialize within timeout. ` +
          `Diagnostics: ${JSON.stringify(diagnostics)}. ` +
          `This may indicate JS bundle failed to load or an uncaught error during init.`
        );
      }
    }
  }
}

/**
 * Wait for NavigationManager to be fully initialized.
 *
 * CRITICAL: Call this after the app is visible and before clicking any nav tabs.
 * The NavigationManager is created AFTER the app container becomes visible,
 * so tests that click nav tabs immediately may fail due to race conditions.
 *
 * NOTE: The NavigationManager sets aria-selected ONLY when switchDashboardView() is
 * called (during navigation). On initial load with no URL hash, aria-selected may not
 * be set. We check for:
 * 1. .nav-tab elements exist
 * 2. They have role="tab" (from HTML)
 * 3. They have tabindex (indicating keyboard navigation is set up)
 */
export async function waitForNavReady(page: Page): Promise<void> {
  // PERFORMANCE FIX: Reduced timeouts from TIMEOUTS.SHORT (2s) to TIMEOUTS.RENDER (500ms)
  // WHY: The HTML already has role="tab" and tabindex attributes, so these checks should
  // pass immediately if the DOM is ready. Using 2s timeouts added 4s to every test when
  // checks failed in CI. With 500ms timeouts, we fail fast and continue (soft checks).
  //
  // NOTE: In CI environments with SwiftShader, these checks may occasionally fail due to
  // timing differences in how quickly the DOM is ready for querying. This is expected
  // behavior - the warnings are soft and tests continue regardless. Navigation should
  // still work because the HTML has all required attributes from the start.

  // Check that nav tabs exist and have basic accessibility attributes
  // WHY: We check for role="tab" which should be in HTML initially,
  // rather than aria-selected which is only set after navigation
  const navTabsReady = await page.waitForFunction(
    () => {
      const tabs = document.querySelectorAll('.nav-tab');
      if (tabs.length === 0) return false;
      // Check at least one tab has role="tab" (from HTML)
      for (const tab of tabs) {
        if (tab.getAttribute('role') === 'tab') {
          return true;
        }
      }
      return false;
    },
    { timeout: TIMEOUTS.RENDER }
  ).then(() => true).catch(() => false);

  if (!navTabsReady && process.env.E2E_VERBOSE === 'true') {
    // Soft warning - navigation may still work without the role attribute
    console.warn('[E2E] waitForNavReady: Nav tabs not ready');
  }

  // Wait for event handlers to be fully attached
  // NavigationManager clones tabs and adds click listeners during setup
  // We verify tabs are interactive by checking tabindex attribute
  const handlersReady = await page.waitForFunction(
    () => {
      const tab = document.querySelector('.nav-tab') as HTMLElement;
      if (!tab) return false;
      // After NavigationManager.setupNavigationListeners(), tabs are interactive
      // Check for tabindex which is set during initialization
      // Note: tabindex might be '-1' for inactive tabs (roving tabindex pattern)
      return tab.hasAttribute('tabindex');
    },
    { timeout: TIMEOUTS.RENDER }
  ).then(() => true).catch(() => false);

  if (!handlersReady && process.env.E2E_VERBOSE === 'true') {
    console.warn('[E2E] waitForNavReady: Nav handlers not ready');
  }
}

/**
 * Wait for the app to be fully authenticated and initialized.
 */
export async function waitForAppReady(page: Page): Promise<void> {
  await expect(page.locator(SELECTORS.APP_VISIBLE)).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
  // Wait for stats to load (indicates API data is available)
  const statsLoaded = await page.waitForFunction(
    () => {
      const statsEl = document.getElementById('stat-playbacks');
      return statsEl && statsEl.textContent !== '-';
    },
    { timeout: TIMEOUTS.DEFAULT }
  ).then(() => true).catch(() => false);

  if (!statsLoaded && process.env.E2E_VERBOSE === 'true') {
    console.warn('[E2E] waitForAppReady: Stats not loaded');
  }
}

// =============================================================================
// NAVIGATION HELPERS
// =============================================================================

/**
 * Navigate to a main view (maps, analytics, globe, server).
 *
 * @param page - Playwright page
 * @param view - The view to navigate to
 * @param options - Configuration options
 */
export async function navigateToView(
  page: Page,
  view: View,
  options: {
    waitForContainer?: boolean;
    timeout?: number;
  } = {}
): Promise<void> {
  const { waitForContainer = true, timeout = TIMEOUTS.MEDIUM } = options;

  await page.click(SELECTORS.NAV_TAB_BY_VIEW(view));

  if (waitForContainer) {
    const containerSelector = getViewContainerSelector(view);
    await expect(page.locator(containerSelector)).toBeVisible({ timeout });

    // Wait for view container to be fully visible and ready (opacity: 1, not transitioning)
    const transitionComplete = await page.waitForFunction(
      (selector) => {
        const container = document.querySelector(selector);
        if (!container) return false;
        const styles = getComputedStyle(container);
        // Check if container is fully visible (opacity 1 and display not none)
        const isVisible = styles.opacity === '1' && styles.display !== 'none';
        // Check if no transition is running
        const isNotTransitioning = !styles.transition || styles.transition === 'none' ||
                                   !container.classList.contains('transitioning');
        return isVisible && isNotTransitioning;
      },
      containerSelector,
      { timeout: TIMEOUTS.RENDER }
    ).then(() => true).catch(() => false);

    if (!transitionComplete) {
      console.warn(`[E2E] navigateToView: View transition to ${view} may not be complete`);
    }
  }
}

/**
 * Navigate to an analytics sub-page.
 *
 * @param page - Playwright page
 * @param analyticsPage - The analytics page to navigate to
 * @param options - Configuration options
 */
export async function navigateToAnalyticsPage(
  page: Page,
  analyticsPage: AnalyticsPage,
  options: {
    ensureOnAnalytics?: boolean;
    waitForPage?: boolean;
    timeout?: number;
  } = {}
): Promise<void> {
  const { ensureOnAnalytics = true, waitForPage = true, timeout = TIMEOUTS.MEDIUM } = options;

  // First ensure we're on the Analytics view
  if (ensureOnAnalytics) {
    const analyticsContainer = page.locator(SELECTORS.ANALYTICS_CONTAINER);
    if (!(await analyticsContainer.isVisible().catch(() => false))) {
      await navigateToView(page, VIEWS.ANALYTICS);
    }
  }

  // Click the analytics tab
  await page.click(SELECTORS.ANALYTICS_TAB_BY_PAGE(analyticsPage));

  if (waitForPage) {
    await expect(page.locator(SELECTORS.ANALYTICS_PAGE_BY_ID(analyticsPage))).toBeVisible({ timeout });

    // Wait for analytics page to be fully visible and ready
    const pageReady = await page.waitForFunction(
      (pageId) => {
        const pageElement = document.getElementById(`analytics-${pageId}`);
        if (!pageElement) return false;
        const styles = getComputedStyle(pageElement);
        // Check if page is fully visible (opacity 1, display block)
        const isVisible = styles.opacity === '1' && styles.display !== 'none';
        // Check if page has content (at least one chart or content element)
        // ROOT CAUSE FIX: Use correct CSS class names from actual HTML
        // WHY: HTML uses 'chart-grid' and 'chart-content' (not 'chart-container' or 'analytics-content')
        const hasContent = pageElement.querySelector('.chart-grid, .chart-content, .chart-card') !== null;
        return isVisible && hasContent;
      },
      analyticsPage,
      { timeout: TIMEOUTS.RENDER }
    ).then(() => true).catch(() => false);

    if (!pageReady) {
      console.warn(`[E2E] navigateToAnalyticsPage: Analytics page ${analyticsPage} may not be fully ready`);
    }
  }
}

/**
 * Navigate to a specific Tautulli sub-tab
 * First navigates to the Tautulli analytics page, then clicks the sub-tab
 */
export async function navigateToTautulliSubTab(
  page: Page,
  subTab: TautulliSubTab,
  options: {
    waitForContainer?: boolean;
    timeout?: number;
  } = {}
): Promise<void> {
  const { waitForContainer = true, timeout = TIMEOUTS.MEDIUM } = options;

  // First navigate to Tautulli analytics page
  await navigateToAnalyticsPage(page, ANALYTICS_PAGES.TAUTULLI);

  // Wait for the sub-tabs to be visible
  const subTabsNav = page.locator('[data-testid="tautulli-sub-tabs"]');
  await expect(subTabsNav).toBeVisible({ timeout: TIMEOUTS.SHORT });

  // Click the sub-tab button
  const subTabButton = page.locator(`[data-sub-tab="${subTab}"]`);
  await subTabButton.click();

  // Wait for the panel to be visible
  if (waitForContainer) {
    const panelId = `${subTab}-panel`;

    await page.waitForFunction(
      (id) => {
        const el = document.getElementById(id);
        return el && window.getComputedStyle(el).display !== 'none';
      },
      panelId,
      { timeout }
    ).catch(() => {
      console.warn(`[E2E] navigateToTautulliSubTab: Panel ${panelId} may not be visible`);
    });
  }
}

/**
 * Get the container selector for a given view.
 */
function getViewContainerSelector(view: View): string {
  switch (view) {
    case VIEWS.MAPS:
      return SELECTORS.MAP;
    case VIEWS.ACTIVITY:
      return SELECTORS.ACTIVITY_CONTAINER;
    case VIEWS.ANALYTICS:
      return SELECTORS.ANALYTICS_CONTAINER;
    case VIEWS.RECENTLY_ADDED:
      return SELECTORS.RECENTLY_ADDED_CONTAINER;
    case VIEWS.GLOBE:
      return SELECTORS.GLOBE;
    case VIEWS.SERVER:
      return SELECTORS.SERVER_CONTAINER;
    default:
      return SELECTORS.APP;
  }
}

// =============================================================================
// MAP HELPERS
// =============================================================================

/**
 * Wait for the map to be fully loaded.
 *
 * This function waits for:
 * 1. The map container element to be visible
 * 2. The MapLibre canvas to be rendered
 * 3. The MapLibre map 'load' event to have fired (required for restoreVisualizationMode)
 *
 * ROOT CAUSE FIX: This addresses state persistence failures where tests failed
 * because they checked button state before the map 'load' event fired and
 * restoreVisualizationMode() was called.
 */
export async function waitForMapReady(page: Page): Promise<void> {
  await expect(page.locator(SELECTORS.MAP)).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
  await expect(page.locator(SELECTORS.MAP_CANVAS)).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

  // Wait for the actual MapLibre map 'load' event to fire
  // The map exposes loaded() method which returns true after 'load' event
  await page.waitForFunction(
    () => {
      const win = window as any;
      // Check if mapManager exists and map is loaded
      if (win.mapManager && typeof win.mapManager.getMap === 'function') {
        const map = win.mapManager.getMap();
        // MapLibre map has a loaded() method that returns true after 'load' event
        if (map && typeof map.loaded === 'function') {
          return map.loaded();
        }
      }
      return false;
    },
    { timeout: TIMEOUTS.DEFAULT }
  ).catch(() => {
    // If mapManager isn't exposed or loaded() check fails, fall back to delay
    // This handles cases where the app structure is different
    console.warn('Warning: Could not check map.loaded() state, using fallback delay');
  });

  // Wait for post-load initialization (like restoreVisualizationMode) to complete
  const postLoadReady = await page.waitForFunction(
    () => {
      const win = window as any;
      // Check if map is initialized and has styles applied
      if (win.mapManager && typeof win.mapManager.getMap === 'function') {
        const map = win.mapManager.getMap();
        if (map && map.loaded && map.loaded()) {
          // Check if map has a style loaded
          const style = map.getStyle && map.getStyle();
          return style !== null && style !== undefined;
        }
      }
      return false;
    },
    { timeout: TIMEOUTS.ANIMATION }
  ).then(() => true).catch(() => false);

  if (!postLoadReady) {
    console.warn('[E2E] waitForMapReady: Post-load initialization may not be complete');
  }

  // Set flag for tests that wait for it
  await page.evaluate(() => {
    (window as any).mapLoaded = true;
  });
}

/**
 * Wait for the globe (3D view) to be fully loaded.
 */
export async function waitForGlobeReady(page: Page): Promise<void> {
  await expect(page.locator(SELECTORS.GLOBE)).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
  await expect(page.locator(SELECTORS.GLOBE_CANVAS)).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

  // Wait for WebGL initialization by checking canvas dimensions
  await page.waitForFunction(
    () => {
      const canvas = document.querySelector('#globe canvas') as HTMLCanvasElement;
      if (!canvas) return false;
      // Check if canvas has non-zero dimensions (indicates WebGL is initialized)
      const hasSize = canvas.width > 0 && canvas.height > 0;
      // Check if canvas has a WebGL context
      const gl = canvas.getContext('webgl') || canvas.getContext('webgl2');
      const hasContext = gl !== null;
      return hasSize && hasContext;
    },
    { timeout: TIMEOUTS.RENDER }
  ).catch(() => {
    // WebGL initialization check may fail in software rendering, continue anyway
    console.warn('Warning: Globe WebGL initialization check timed out');
  });

  // Set flag for tests that wait for it
  await page.evaluate(() => {
    (window as any).globeLoaded = true;
  });
}

// =============================================================================
// CHART HELPERS
// =============================================================================

/**
 * Wait for chart data to be fully loaded.
 *
 * This waits for all specified chart containers to have aria-busy="false",
 * which indicates that the chart has finished loading data and rendering.
 *
 * Use this in CI where serialized API requests make data loading slow.
 * The default timeout is 60s to accommodate 30+ serialized API calls.
 *
 * @param page - Playwright page
 * @param chartIds - Optional array of chart IDs to wait for (defaults to first visible chart)
 * @param timeout - Maximum time to wait (default: 60s for CI compatibility)
 */
export async function waitForChartDataLoaded(
  page: Page,
  chartIds?: readonly string[],
  timeout: number = 60000
): Promise<void> {
  // If specific charts specified, wait for each one
  if (chartIds && chartIds.length > 0) {
    for (const chartId of chartIds) {
      await page.waitForFunction(
        (id) => {
          const chart = document.querySelector(id);
          if (!chart) return false;
          // Check aria-busy is false (data loaded)
          const ariaBusy = chart.getAttribute('aria-busy');
          if (ariaBusy !== 'false') return false;
          // Also verify that canvas or svg element exists (ECharts rendered)
          // This prevents race conditions where aria-busy is set before canvas creation
          const hasCanvas = chart.querySelector('canvas') !== null;
          const hasSvg = chart.querySelector('svg') !== null;
          return hasCanvas || hasSvg;
        },
        chartId,
        { timeout }
      );
    }
  } else {
    // Wait for at least one chart to finish loading
    await page.waitForFunction(
      () => {
        const charts = document.querySelectorAll('[id^="chart-"]');
        for (const chart of charts) {
          const ariaBusy = chart.getAttribute('aria-busy');
          if (ariaBusy === 'false') {
            // Also verify canvas/svg exists
            const hasCanvas = chart.querySelector('canvas') !== null;
            const hasSvg = chart.querySelector('svg') !== null;
            if (hasCanvas || hasSvg) {
              return true;
            }
          }
        }
        return false;
      },
      { timeout }
    );
  }
}

/**
 * Wait for charts to be loaded on the current page.
 */
export async function waitForChartsReady(page: Page): Promise<void> {
  // Wait for at least one chart canvas to be visible
  const chartCanvases = page.locator(`${SELECTORS.CHART_CONTAINER} canvas`);
  await expect(chartCanvases.first()).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

  // Wait for charts to have rendered content (canvas dimensions > 0)
  await page.waitForFunction(
    () => {
      const canvases = document.querySelectorAll('.chart-container canvas');
      if (canvases.length === 0) return false;
      // Check if at least one canvas has non-zero dimensions (rendered)
      for (const canvas of canvases) {
        const cvs = canvas as HTMLCanvasElement;
        if (cvs.width > 0 && cvs.height > 0) {
          return true;
        }
      }
      return false;
    },
    { timeout: TIMEOUTS.RENDER }
  ).catch(() => {
    // Chart rendering check may fail, continue anyway
    console.warn('Warning: Chart rendering check timed out');
  });

  // Set flag for tests that wait for it
  await page.evaluate(() => {
    (window as any).chartsLoaded = true;
  });
}

/**
 * Assert that all specified charts are visible.
 *
 * @param page - Playwright page
 * @param chartIds - Array of chart container IDs (e.g., ['#chart-trends', '#chart-media'])
 * @param options - Configuration options
 */
export async function expectChartsVisible(
  page: Page,
  chartIds: readonly string[],
  options: {
    timeout?: number;
    scrollIntoView?: boolean;
  } = {}
): Promise<void> {
  const { timeout = TIMEOUTS.DEFAULT, scrollIntoView = false } = options;

  for (const chartId of chartIds) {
    const chart = page.locator(chartId);

    if (scrollIntoView) {
      await chart.scrollIntoViewIfNeeded();
    }

    await expect(chart).toBeVisible({ timeout });
  }
}

/**
 * Assert that all specified charts have rendered content (canvas or SVG).
 *
 * @param page - Playwright page
 * @param chartIds - Array of chart container IDs
 * @param options - Configuration options
 */
export async function expectChartsRendered(
  page: Page,
  chartIds: readonly string[],
  options: {
    timeout?: number;
    scrollIntoView?: boolean;
  } = {}
): Promise<void> {
  const { timeout = TIMEOUTS.DEFAULT, scrollIntoView = false } = options;

  for (const chartId of chartIds) {
    const chart = page.locator(chartId);

    if (scrollIntoView) {
      await chart.scrollIntoViewIfNeeded();
    }

    // ECharts can render to canvas or SVG depending on configuration
    const chartElement = page.locator(SELECTORS.CHART_ELEMENT(chartId));
    await expect(chartElement.first()).toBeVisible({ timeout });
  }
}

/**
 * Assert that charts for a specific analytics page are visible.
 */
export async function expectAnalyticsPageChartsVisible(
  page: Page,
  analyticsPage: keyof typeof CHARTS,
  options: {
    timeout?: number;
    scrollIntoView?: boolean;
  } = {}
): Promise<void> {
  const chartIds = CHARTS[analyticsPage];
  if (!chartIds) {
    throw new Error(`Unknown analytics page: ${analyticsPage}`);
  }
  await expectChartsVisible(page, chartIds, options);
}

// =============================================================================
// WEBGL CLEANUP HELPERS
// =============================================================================

/**
 * Clean up WebGL resources to prevent memory exhaustion.
 * This is critical for CI environments using SwiftShader (software WebGL).
 *
 * Call this in afterEach for tests that use WebGL-heavy features
 * (maps, globe, charts with canvas renderer).
 *
 * NOTE: This function is designed to be safe during test teardown.
 * It avoids navigation and uses timeouts to prevent hanging.
 *
 * @param page - Playwright page
 */
export async function cleanupWebGLResources(page: Page): Promise<void> {
  try {
    // Check if page is still open and responsive
    const isPageOpen = await Promise.race([
      page.evaluate(() => true).catch(() => false),
      new Promise<boolean>((resolve) => setTimeout(() => resolve(false), 1000)),
    ]);

    if (!isPageOpen) {
      // Page is already closed or unresponsive, skip cleanup
      return;
    }

    // Force WebGL context loss to release GPU memory
    // Use a timeout to prevent hanging during teardown
    await Promise.race([
      page.evaluate(() => {
        const canvases = document.querySelectorAll('canvas');
        canvases.forEach(canvas => {
          try {
            // Try both webgl and webgl2 contexts
            const gl = canvas.getContext('webgl') || canvas.getContext('webgl2');
            if (gl && typeof (gl as any).getExtension === 'function') {
              const ext = (gl as any).getExtension('WEBGL_lose_context');
              if (ext) ext.loseContext();
            }
          } catch {
            // Individual canvas cleanup failed, continue with others
          }
        });
      }),
      new Promise<void>((resolve) => setTimeout(resolve, 2000)), // 2s timeout
    ]);
  } catch {
    // Ignore cleanup errors - page may already be closed or in bad state
  }
}

// =============================================================================
// MEMORY DIAGNOSTICS HELPERS
// =============================================================================

/**
 * Memory diagnostics information returned by logMemoryUsage.
 */
export interface MemoryDiagnostics {
  /** JS heap size currently in use (bytes) */
  usedJSHeapSize: number | null;
  /** Total JS heap size (bytes) */
  totalJSHeapSize: number | null;
  /** JS heap size limit (bytes) */
  jsHeapSizeLimit: number | null;
  /** Number of active WebGL contexts */
  webglContextCount: number;
  /** Number of canvas elements in DOM */
  canvasCount: number;
  /** Human-readable memory summary */
  summary: string;
}

/**
 * Log memory usage diagnostics for debugging browser crashes and memory exhaustion.
 *
 * This function is useful for:
 * - Diagnosing browser crashes in CI (SwiftShader memory limits)
 * - Tracking memory growth across tests
 * - Identifying WebGL context accumulation
 *
 * NOTE: performance.memory is only available in Chromium with --enable-precise-memory-info flag.
 * In other browsers or without the flag, heap sizes will be null.
 *
 * @param page - Playwright page
 * @param label - Optional label for the log message (e.g., "beforeEach", "afterEach", test name)
 * @returns Memory diagnostics object
 *
 * @example
 * ```typescript
 * test.beforeEach(async ({ page }) => {
 *   await logMemoryUsage(page, 'beforeEach');
 * });
 *
 * test.afterEach(async ({ page }) => {
 *   await logMemoryUsage(page, 'afterEach');
 * });
 * ```
 */
export async function logMemoryUsage(page: Page, label?: string): Promise<MemoryDiagnostics> {
  try {
    // Check if page is still open
    const isPageOpen = await page.evaluate(() => true).catch(() => false);
    if (!isPageOpen) {
      return {
        usedJSHeapSize: null,
        totalJSHeapSize: null,
        jsHeapSizeLimit: null,
        webglContextCount: 0,
        canvasCount: 0,
        summary: '[MEMORY] Page closed',
      };
    }

    const diagnostics = await page.evaluate(() => {
      // Get JS heap memory (Chromium only)
      let usedJSHeapSize: number | null = null;
      let totalJSHeapSize: number | null = null;
      let jsHeapSizeLimit: number | null = null;

      if ('memory' in performance) {
        const memory = (performance as any).memory;
        usedJSHeapSize = memory.usedJSHeapSize;
        totalJSHeapSize = memory.totalJSHeapSize;
        jsHeapSizeLimit = memory.jsHeapSizeLimit;
      }

      // Count canvas elements and WebGL contexts
      const canvases = document.querySelectorAll('canvas');
      let webglContextCount = 0;

      canvases.forEach(canvas => {
        try {
          // Check if canvas has an active WebGL context
          const gl = canvas.getContext('webgl') || canvas.getContext('webgl2');
          if (gl && !gl.isContextLost()) {
            webglContextCount++;
          }
        } catch {
          // Context access failed, skip
        }
      });

      return {
        usedJSHeapSize,
        totalJSHeapSize,
        jsHeapSizeLimit,
        webglContextCount,
        canvasCount: canvases.length,
      };
    });

    // Format human-readable summary
    const formatMB = (bytes: number | null): string =>
      bytes !== null ? `${Math.round(bytes / 1024 / 1024)}MB` : 'N/A';

    const heapInfo = diagnostics.usedJSHeapSize !== null
      ? `Heap: ${formatMB(diagnostics.usedJSHeapSize)}/${formatMB(diagnostics.totalJSHeapSize)} (limit: ${formatMB(diagnostics.jsHeapSizeLimit)})`
      : 'Heap: N/A (not Chromium or flag not set)';

    const prefix = label ? `[MEMORY:${label}]` : '[MEMORY]';
    const summary = `${prefix} ${heapInfo}, WebGL contexts: ${diagnostics.webglContextCount}, Canvases: ${diagnostics.canvasCount}`;

    // Only log in CI or when E2E_DEBUG is set
    if (process.env.CI || process.env.E2E_DEBUG === 'true') {
      console.log(summary);
    }

    return {
      ...diagnostics,
      summary,
    };
  } catch {
    // Return empty diagnostics if evaluation failed
    return {
      usedJSHeapSize: null,
      totalJSHeapSize: null,
      jsHeapSizeLimit: null,
      webglContextCount: 0,
      canvasCount: 0,
      summary: '[MEMORY] Diagnostics failed (page may be closed)',
    };
  }
}

/**
 * Check if memory usage exceeds a threshold.
 * Useful for conditional test skipping or early cleanup.
 *
 * @param page - Playwright page
 * @param thresholdMB - Memory threshold in megabytes (default: 1024MB)
 * @returns true if memory exceeds threshold, false otherwise (or if unavailable)
 */
export async function isMemoryPressureHigh(page: Page, thresholdMB: number = 1024): Promise<boolean> {
  const diagnostics = await logMemoryUsage(page);
  if (diagnostics.usedJSHeapSize === null) {
    return false; // Can't determine, assume OK
  }
  const usedMB = diagnostics.usedJSHeapSize / 1024 / 1024;
  return usedMB > thresholdMB;
}

// =============================================================================
// LOGIN HELPERS
// =============================================================================

/**
 * Skip cookie consent banner by setting localStorage flag.
 * Call this before navigating or after page.goto() but before interacting with the app.
 * This prevents the cookie consent banner from appearing and blocking interactions.
 */
export async function skipCookieConsent(page: Page): Promise<void> {
  await page.evaluate(() => {
    const cookiePreferences = {
      necessary: true,
      analytics: false,
      preferences: false,
      consentGiven: true,
      consentDate: new Date().toISOString(),
    };
    localStorage.setItem('cartographus-cookie-consent', JSON.stringify(cookiePreferences));
  });
}

/**
 * Dismiss cookie consent banner if it's visible.
 * Use this when the banner might already be showing.
 */
export async function dismissCookieConsentBanner(page: Page): Promise<void> {
  const banner = page.locator('.cookie-consent-banner');
  const isVisible = await banner.isVisible().catch(() => false);

  if (isVisible) {
    // Click "Essential Only" button to dismiss with minimal cookies
    const declineBtn = page.locator('#cookie-decline');
    if (await declineBtn.isVisible().catch(() => false)) {
      await declineBtn.click();
      const bannerHidden = await banner.waitFor({ state: 'hidden', timeout: TIMEOUTS.SHORT }).then(() => true).catch(() => false);
      if (!bannerHidden) {
        console.warn('[E2E] dismissCookieConsentBanner: Banner not hidden after clicking decline button');
      }
    } else {
      // Try pressing Escape as fallback
      await page.keyboard.press('Escape');
      const bannerHidden = await banner.waitFor({ state: 'hidden', timeout: TIMEOUTS.SHORT }).then(() => true).catch(() => false);
      if (!bannerHidden) {
        console.warn('[E2E] dismissCookieConsentBanner: Banner not hidden after pressing Escape');
      }
    }
  }
}

/**
 * Skip onboarding modal by setting localStorage flag.
 * Call this after page.goto() but before interacting with the app.
 */
export async function skipOnboarding(page: Page): Promise<void> {
  await page.evaluate(() => {
    localStorage.setItem('onboarding_completed', 'true');
    localStorage.setItem('onboarding_skipped', 'true');
  });
}

/**
 * Dismiss onboarding modal if it's visible.
 * Use this when the modal might already be showing.
 */
export async function dismissOnboardingModal(page: Page): Promise<void> {
  const modal = page.locator('#onboarding-modal');
  const isVisible = await modal.isVisible().catch(() => false);

  if (isVisible) {
    // Try to click skip button first
    const skipBtn = page.locator('#onboarding-skip-btn');
    if (await skipBtn.isVisible().catch(() => false)) {
      await skipBtn.click();
      const modalHidden = await modal.waitFor({ state: 'hidden', timeout: TIMEOUTS.SHORT }).then(() => true).catch(() => false);
      if (!modalHidden) {
        console.warn('[E2E] dismissOnboardingModal: Modal not hidden after clicking skip button');
      }
    } else {
      // If skip button not visible, try pressing Escape
      await page.keyboard.press('Escape');
      const modalHidden = await modal.waitFor({ state: 'hidden', timeout: TIMEOUTS.SHORT }).then(() => true).catch(() => false);
      if (!modalHidden) {
        console.warn('[E2E] dismissOnboardingModal: Modal not hidden after pressing Escape');
      }
    }
  }
}

/**
 * Dismiss setup wizard modal if it's visible.
 * The setup wizard intercepts all pointer events when visible, blocking all UI interactions.
 * Use this as a fallback when the modal might appear despite localStorage flags.
 */
export async function dismissSetupWizardModal(page: Page): Promise<void> {
  const modal = page.locator('#setup-wizard-modal');
  const isVisible = await modal.isVisible().catch(() => false);

  if (isVisible) {
    console.log('[E2E] dismissSetupWizardModal: Setup wizard modal detected, dismissing...');

    // Set localStorage flags to prevent future appearances
    await page.evaluate(() => {
      localStorage.setItem('setup_wizard_completed', 'true');
      localStorage.setItem('setup_wizard_skipped', 'true');
    });

    // Try to click skip button first (uses ID from SetupWizardManager.ts)
    const skipBtn = page.locator('#wizard-skip-btn');
    if (await skipBtn.isVisible().catch(() => false)) {
      await skipBtn.click();
      const modalHidden = await modal.waitFor({ state: 'hidden', timeout: TIMEOUTS.SHORT }).then(() => true).catch(() => false);
      if (modalHidden) {
        console.log('[E2E] dismissSetupWizardModal: Modal dismissed via skip button');
        return;
      }
    }

    // Try pressing Escape as fallback
    await page.keyboard.press('Escape');
    const modalHiddenEscape = await modal.waitFor({ state: 'hidden', timeout: TIMEOUTS.SHORT }).then(() => true).catch(() => false);
    if (modalHiddenEscape) {
      console.log('[E2E] dismissSetupWizardModal: Modal dismissed via Escape key');
      return;
    }

    // Try clicking the close button
    const closeBtn = modal.locator('.setup-wizard-close');
    if (await closeBtn.isVisible().catch(() => false)) {
      await closeBtn.click();
      const modalHiddenClose = await modal.waitFor({ state: 'hidden', timeout: TIMEOUTS.SHORT }).then(() => true).catch(() => false);
      if (modalHiddenClose) {
        console.log('[E2E] dismissSetupWizardModal: Modal dismissed via close button');
        return;
      }
    }

    console.warn('[E2E] dismissSetupWizardModal: Failed to dismiss modal using all methods');
  }
}

/**
 * Perform login with the provided or default credentials.
 *
 * @param page - Playwright page
 * @param options - Configuration options
 */
export async function performLogin(
  page: Page,
  options: {
    username?: string;
    password?: string;
    waitForApp?: boolean;
    timeout?: number;
  } = {}
): Promise<void> {
  const credentials = getTestCredentials();
  const username = options.username ?? credentials.username;
  const password = options.password ?? credentials.password;
  const waitForApp = options.waitForApp ?? true;
  const timeout = options.timeout ?? TIMEOUTS.DEFAULT;

  // Skip onboarding and cookie consent to prevent modals from blocking interactions
  await skipOnboarding(page);
  await skipCookieConsent(page);

  // Dismiss cookie consent banner if it's already visible
  await dismissCookieConsentBanner(page);

  // Ensure login button is ready before interacting
  await expect(page.locator(SELECTORS.LOGIN_BUTTON)).toBeEnabled({ timeout: TIMEOUTS.MEDIUM });

  // Fill in credentials
  await page.fill(SELECTORS.LOGIN_USERNAME, username);
  await page.fill(SELECTORS.LOGIN_PASSWORD, password);

  // Click login button using JavaScript for CI reliability
  // WHY: Playwright's .click() may fail in headless/SwiftShader environments
  await page.evaluate(() => {
    const btn = document.querySelector('button[type="submit"]') as HTMLElement;
    if (btn) btn.click();
  });

  if (waitForApp) {
    // DETERMINISTIC FIX: Wait for the login UI state change using condition-based wait.
    // We use waitForFunction to check for DOM state change rather than arbitrary delays.
    // This is 100% deterministic because it waits for the actual condition to be true.
    await page.waitForFunction(
      (selectors) => {
        const loginContainer = document.querySelector(selectors.login);
        const appContainer = document.querySelector(selectors.app);
        // Login is complete when: login container is hidden AND app is visible
        const loginHidden = !loginContainer ||
          (loginContainer as HTMLElement).style.display === 'none' ||
          !(loginContainer as HTMLElement).offsetParent;
        const appVisible = appContainer &&
          (appContainer as HTMLElement).offsetParent !== null;
        return loginHidden && appVisible;
      },
      { login: SELECTORS.LOGIN_CONTAINER, app: SELECTORS.APP },
      { timeout }
    );

    // Verify final state with Playwright assertions (belt and suspenders)
    await expect(page.locator(SELECTORS.LOGIN_CONTAINER)).not.toBeVisible({ timeout: 5000 });
    await expect(page.locator(SELECTORS.APP)).toBeVisible({ timeout: 5000 });

    // Dismiss any onboarding modal that might have appeared
    await dismissOnboardingModal(page);
  }
}

/**
 * Clear authentication state and reload the page.
 */
export async function clearAuthAndReload(page: Page): Promise<void> {
  await page.context().clearCookies();
  await page.evaluate(() => {
    localStorage.removeItem('auth_token');
    localStorage.removeItem('auth_username');
    localStorage.removeItem('auth_expires_at');
  });
  await page.reload();
}

// =============================================================================
// WEBSOCKET HELPERS
// =============================================================================

/**
 * Wait for the test helpers to be exposed on window.__app.
 */
export async function waitForTestHelpers(page: Page): Promise<boolean> {
  try {
    await page.waitForFunction(
      () => {
        const app = (window as any).__app;
        return (
          app &&
          typeof app.__testSimulateWebSocketMessage === 'function' &&
          typeof app.__testIsWebSocketConnected === 'function' &&
          typeof app.__testGetPlexMonitoringManager === 'function'
        );
      },
      { timeout: TIMEOUTS.LONG }
    );
    return true;
  } catch {
    return false;
  }
}

/**
 * Simulate a WebSocket message using the test helpers.
 */
export async function simulateWebSocketMessage(
  page: Page,
  type: string,
  data: any
): Promise<boolean> {
  return page.evaluate(
    ({ type, data }) => {
      const app = (window as any).__app;
      if (app && app.__testSimulateWebSocketMessage) {
        app.__testSimulateWebSocketMessage(type, data);
        return true;
      }
      return false;
    },
    { type, data }
  );
}

/**
 * Check if WebSocket is connected.
 */
export async function isWebSocketConnected(page: Page): Promise<boolean> {
  return page.evaluate(() => {
    const app = (window as any).__app;
    return app ? app.__testIsWebSocketConnected() : false;
  });
}

// =============================================================================
// WAIT PATTERN HELPERS
// =============================================================================

/**
 * Wait for an analytics page to be fully visible after tab click.
 * Use this instead of waitForTimeout after clicking an analytics tab.
 *
 * @param page - Playwright page
 * @param analyticsPageId - The analytics page ID (e.g., 'overview', 'content', 'users')
 * @param timeout - Maximum time to wait (default: TIMEOUTS.MEDIUM)
 */
export async function waitForAnalyticsPageVisible(
  page: Page,
  analyticsPageId: string,
  timeout: number = TIMEOUTS.MEDIUM
): Promise<void> {
  await expect(page.locator(`#analytics-${analyticsPageId}`)).toBeVisible({ timeout });
}

/**
 * Wait for a chart to be rendered (has canvas or SVG content).
 * Use this instead of waitForTimeout after loading chart pages.
 *
 * @param page - Playwright page
 * @param chartId - The chart container ID (e.g., '#chart-trends')
 * @param timeout - Maximum time to wait (default: TIMEOUTS.DEFAULT)
 */
export async function waitForChartRendered(
  page: Page,
  chartId: string,
  timeout: number = TIMEOUTS.DEFAULT
): Promise<void> {
  const chartElement = page.locator(`${chartId} canvas, ${chartId} svg`);
  await expect(chartElement.first()).toBeVisible({ timeout });
}

/**
 * Wait for multiple charts to be rendered.
 * Use this for pages with multiple charts.
 *
 * @param page - Playwright page
 * @param chartIds - Array of chart container IDs
 * @param timeout - Maximum time to wait per chart (default: TIMEOUTS.DEFAULT)
 */
export async function waitForChartsRenderedMultiple(
  page: Page,
  chartIds: string[],
  timeout: number = TIMEOUTS.DEFAULT
): Promise<void> {
  for (const chartId of chartIds) {
    const chartElement = page.locator(`${chartId} canvas, ${chartId} svg`);
    await expect(chartElement.first()).toBeVisible({ timeout });
  }
}

/**
 * Wait for the globe view to be ready after switching from 2D.
 * Use this instead of waitForTimeout after clicking the 3D button.
 *
 * @param page - Playwright page
 * @param timeout - Maximum time to wait (default: TIMEOUTS.LONG)
 */
export async function waitForGlobeViewReady(
  page: Page,
  timeout: number = TIMEOUTS.LONG
): Promise<void> {
  // Wait for globe container to be visible
  await expect(page.locator(SELECTORS.GLOBE)).toBeVisible({ timeout });
  // Wait for canvas to be rendered
  await expect(page.locator(SELECTORS.GLOBE_CANVAS)).toBeVisible({ timeout });
  // Wait for globe to be fully initialized (WebGL context ready)
  const webglReady = await page.waitForFunction(
    () => {
      const canvas = document.querySelector('#globe canvas');
      if (!canvas) return false;
      const gl = (canvas as HTMLCanvasElement).getContext('webgl') ||
                 (canvas as HTMLCanvasElement).getContext('webgl2');
      return gl !== null;
    },
    { timeout }
  ).then(() => true).catch(() => false);

  if (!webglReady) {
    console.warn('[E2E] waitForGlobeViewReady: WebGL context not available - may be using SwiftShader');
  }
}

/**
 * Wait for a view transition to complete.
 * Use this after clicking nav tabs or view toggles.
 *
 * @param page - Playwright page
 * @param containerSelector - The container that should become visible
 * @param timeout - Maximum time to wait (default: TIMEOUTS.MEDIUM)
 */
export async function waitForViewTransition(
  page: Page,
  containerSelector: string,
  timeout: number = TIMEOUTS.MEDIUM
): Promise<void> {
  await expect(page.locator(containerSelector)).toBeVisible({ timeout });

  // Wait for CSS transition to complete by checking computed styles
  const transitionComplete = await page.waitForFunction(
    (selector) => {
      const container = document.querySelector(selector);
      if (!container) return false;
      const styles = getComputedStyle(container);
      // Check if container is fully visible (opacity 1)
      const isFullyVisible = styles.opacity === '1';
      // Check if no transition is currently running
      // This checks both the transition property and if element is in transitioning state
      const transitionDuration = parseFloat(styles.transitionDuration || '0');
      const isNotTransitioning = transitionDuration === 0 ||
                                !container.classList.contains('transitioning');
      return isFullyVisible && isNotTransitioning;
    },
    containerSelector,
    { timeout: TIMEOUTS.ANIMATION }
  ).then(() => true).catch(() => false);

  if (!transitionComplete) {
    console.warn(`[E2E] waitForViewTransition: CSS transition for ${containerSelector} may not be complete`);
  }
}

/**
 * Wait for data to load after filter change or refresh.
 * Waits for network idle and loading indicators to disappear.
 *
 * @param page - Playwright page
 * @param timeout - Maximum time to wait (default: TIMEOUTS.DEFAULT)
 */
export async function waitForDataLoad(
  page: Page,
  timeout: number = TIMEOUTS.DEFAULT
): Promise<void> {
  // Wait for any loading indicators to disappear
  const loading = page.locator('.loading:visible, .spinner:visible, [data-loading="true"]');
  const loadingHidden = await loading.waitFor({ state: 'hidden', timeout }).then(() => true).catch(() => false);
  if (!loadingHidden) {
    console.warn('[E2E] waitForDataLoad: Loading indicators may still be visible');
  }
  // Wait for network to settle
  const networkSettled = await page.waitForLoadState('networkidle', { timeout }).then(() => true).catch(() => false);
  if (!networkSettled) {
    console.warn('[E2E] waitForDataLoad: Network did not reach idle state');
  }
}

/**
 * Click an analytics tab and wait for the page to be visible.
 * Replaces the pattern: click() + waitForTimeout()
 *
 * @param page - Playwright page
 * @param analyticsPageId - The analytics page ID (e.g., 'overview', 'content')
 * @param timeout - Maximum time to wait (default: TIMEOUTS.MEDIUM)
 */
export async function clickAnalyticsTabAndWait(
  page: Page,
  analyticsPageId: string,
  timeout: number = TIMEOUTS.MEDIUM
): Promise<void> {
  await page.click(`.analytics-tab[data-analytics-page="${analyticsPageId}"]`);
  await waitForAnalyticsPageVisible(page, analyticsPageId, timeout);
}

/**
 * Wait for an element to have a specific CSS class.
 * Use this instead of waitForTimeout when checking for state changes.
 *
 * @param page - Playwright page
 * @param selector - Element selector
 * @param className - Expected CSS class
 * @param timeout - Maximum time to wait (default: TIMEOUTS.MEDIUM)
 */
export async function waitForElementClass(
  page: Page,
  selector: string,
  className: string,
  timeout: number = TIMEOUTS.MEDIUM
): Promise<void> {
  await page.waitForFunction(
    ({ sel, cls }) => {
      const el = document.querySelector(sel);
      return el && el.classList.contains(cls);
    },
    { sel: selector, cls: className },
    { timeout }
  );
}

/**
 * Wait for tooltip to appear at current mouse position.
 * Use this instead of waitForTimeout after mouse hover.
 *
 * @param page - Playwright page
 * @param tooltipSelector - Tooltip element selector (default: common tooltip selectors)
 * @param timeout - Maximum time to wait (default: TIMEOUTS.SHORT)
 */
export async function waitForTooltipVisible(
  page: Page,
  tooltipSelector: string = '#deckgl-tooltip:visible, #deckgl-tooltip-enhanced:visible, .tooltip:visible',
  timeout: number = TIMEOUTS.SHORT
): Promise<boolean> {
  try {
    await page.locator(tooltipSelector).waitFor({ state: 'visible', timeout });
    return true;
  } catch {
    return false;
  }
}

/**
 * Wait for theme to change to a specific value.
 * Use this instead of waitForTimeout after clicking theme toggle.
 *
 * @param page - Playwright page
 * @param expectedTheme - The expected theme value ('dark', 'light', 'high-contrast', or null for dark default)
 * @param timeout - Maximum time to wait (default: TIMEOUTS.SHORT)
 */
export async function waitForThemeChange(
  page: Page,
  expectedTheme: 'dark' | 'light' | 'high-contrast' | null,
  timeout: number = TIMEOUTS.SHORT
): Promise<void> {
  await page.waitForFunction(
    (expected) => {
      const dataTheme = document.documentElement.getAttribute('data-theme');
      if (expected === 'dark' || expected === null) {
        return dataTheme === null || dataTheme === 'dark';
      }
      return dataTheme === expected;
    },
    expectedTheme,
    { timeout }
  );
}

/**
 * Click theme toggle and wait for theme to change.
 * Replaces the pattern: click() + waitForTimeout()
 *
 * @param page - Playwright page
 * @param expectedTheme - The expected theme after clicking
 * @param timeout - Maximum time to wait (default: TIMEOUTS.SHORT)
 */
export async function clickThemeToggleAndWait(
  page: Page,
  expectedTheme: 'dark' | 'light' | 'high-contrast',
  timeout: number = TIMEOUTS.SHORT
): Promise<void> {
  await page.click('#theme-toggle');
  await waitForThemeChange(page, expectedTheme, timeout);
}

// =============================================================================
// VIEWPORT HELPERS
// =============================================================================

/**
 * Helper to force CSS media query re-evaluation after viewport change.
 * This is critical for mobile/responsive tests to work correctly.
 */
async function forceMediaQueryReEvaluation(page: Page): Promise<void> {
  await page.evaluate(() => {
    // Force layout recalculation
    document.body.offsetHeight;
    // Trigger resize event to ensure CSS media queries are re-evaluated
    window.dispatchEvent(new Event('resize'));
    // Force another reflow
    document.body.offsetHeight;
  });

  // Wait for CSS media queries to be applied by checking if viewport-specific styles are set
  await page.waitForFunction(
    () => {
      const width = window.innerWidth;
      // Force reflow
      document.body.offsetHeight;

      // Check if media query specific styles are applied
      // For mobile (<= 768px), sidebar should have transform
      if (width <= 768) {
        const sidebar = document.getElementById('sidebar');
        if (sidebar) {
          const styles = getComputedStyle(sidebar);
          // Mobile sidebar should have translateX transform when hidden
          const hasTransform = styles.transform !== 'none' && styles.transform !== 'matrix(1, 0, 0, 1, 0, 0)';
          if (hasTransform) return true;
        }
      }

      // For desktop (> 768px), check if desktop-specific styles are applied
      if (width > 768) {
        const sidebar = document.getElementById('sidebar');
        if (sidebar) {
          const styles = getComputedStyle(sidebar);
          // Desktop sidebar position should be static or relative (not absolute)
          const hasDesktopPosition = styles.position === 'static' || styles.position === 'relative';
          if (hasDesktopPosition) return true;
        }
      }

      // Fallback: check if any CSS custom property is set (indicates styles are loaded)
      const highlight = getComputedStyle(document.documentElement).getPropertyValue('--highlight').trim();
      return highlight !== '';
    },
    { timeout: TIMEOUTS.RENDER }
  ).catch(() => {
    // Media query check may fail, continue anyway
    console.warn('Warning: Media query re-evaluation check timed out');
  });

  // Force one more reflow after styles are applied
  await page.evaluate(() => {
    document.body.offsetHeight;
  });
}

/**
 * Set viewport to mobile size and wait for CSS media queries to apply.
 * This ensures mobile-specific styles (like sidebar transform) are applied.
 */
export async function setMobileViewport(page: Page): Promise<void> {
  await page.setViewportSize({ width: 375, height: 667 });
  await forceMediaQueryReEvaluation(page);
}

/**
 * Set viewport to tablet size and wait for CSS media queries to apply.
 */
export async function setTabletViewport(page: Page): Promise<void> {
  await page.setViewportSize({ width: 768, height: 1024 });
  await forceMediaQueryReEvaluation(page);
}

/**
 * Set viewport to desktop size and wait for CSS media queries to apply.
 */
export async function setDesktopViewport(page: Page): Promise<void> {
  await page.setViewportSize({ width: 1280, height: 720 });
  await forceMediaQueryReEvaluation(page);
}

// =============================================================================
// SAFE NAVIGATION HELPERS
// =============================================================================

/**
 * Wait for pending API requests to complete before navigation.
 *
 * ROOT CAUSE FIX: This addresses context canceled errors in CI logs where
 * database queries are interrupted because tests navigate away before API
 * requests complete. When the HTTP request context is canceled, DuckDB
 * returns "context canceled" errors which appear as 500 responses.
 *
 * Use this helper before navigating to a new view or closing the page to
 * ensure all pending API requests have completed.
 *
 * @param page - Playwright page
 * @param timeout - Maximum time to wait for network idle (default: 5 seconds)
 */
export async function waitForPendingRequests(
  page: Page,
  timeout: number = 5000
): Promise<void> {
  // Check if page is still open before proceeding
  const isPageOpen = await page.evaluate(() => true).catch(() => false);
  if (!isPageOpen) {
    return; // Page already closed, skip
  }

  try {
    // Wait for network to become idle (no pending requests for 500ms)
    await page.waitForLoadState('networkidle', { timeout });
  } catch {
    // Timeout is acceptable - some long-polling or WebSocket connections
    // may keep the network "busy". Log for debugging.
    console.log('[E2E] waitForPendingRequests: Network did not reach idle state within timeout');
  }
}

/**
 * Navigate to a view with safe cleanup of pending requests.
 *
 * This is a safer version of navigateToView that waits for pending
 * requests before navigation to prevent context canceled errors.
 *
 * @param page - Playwright page
 * @param view - The view to navigate to
 * @param options - Configuration options
 */
export async function safeNavigateToView(
  page: Page,
  view: View,
  options: {
    waitForContainer?: boolean;
    waitForPendingRequests?: boolean;
    timeout?: number;
  } = {}
): Promise<void> {
  const {
    waitForContainer = true,
    waitForPendingRequests: waitPending = true,
    timeout = TIMEOUTS.MEDIUM
  } = options;

  // Wait for any pending requests to complete before navigating
  if (waitPending) {
    await waitForPendingRequests(page, TIMEOUTS.SHORT);
  }

  // Now navigate to the new view
  await navigateToView(page, view, { waitForContainer, timeout });
}

/**
 * Wait for the map managers to be fully initialized.
 * This is critical for tests that interact with map mode buttons.
 *
 * The issue is that event listeners are attached synchronously during app init,
 * but the managers they reference (viewportManager, mapManager) need to be verified
 * as ready before interactions will work correctly.
 *
 * @param page - Playwright page
 * @param timeout - Maximum time to wait for initialization
 */
export async function waitForMapManagersReady(
  page: Page,
  timeout: number = TIMEOUTS.MEDIUM
): Promise<boolean> {
  try {
    await page.waitForFunction(
      () => {
        const win = window as any;
        // Check that both managers exist and are connected
        return !!(
          win.mapManager &&
          win.viewportManager &&
          win.viewportManager.mapManager &&
          typeof win.mapManager.getVisualizationMode === 'function' &&
          typeof win.viewportManager.setMapMode === 'function'
        );
      },
      { timeout }
    );
    return true;
  } catch {
    console.warn('[E2E] waitForMapManagersReady: Managers not fully initialized within timeout');
    return false;
  }
}

/**
 * Set the map visualization mode directly via JavaScript.
 * This is more reliable than clicking buttons in CI environments where
 * event timing can be unpredictable.
 *
 * @param page - Playwright page
 * @param mode - Visualization mode to set ('points', 'clusters', 'heatmap', 'hexagons')
 * @param options - Configuration options
 * @returns true if mode was set successfully
 */
export async function setMapVisualizationMode(
  page: Page,
  mode: 'points' | 'clusters' | 'heatmap' | 'hexagons',
  options: {
    waitForManagers?: boolean;
    timeout?: number;
  } = {}
): Promise<boolean> {
  const { waitForManagers = true, timeout = TIMEOUTS.MEDIUM } = options;

  // Ensure managers are ready before attempting to set mode
  if (waitForManagers) {
    const managersReady = await waitForMapManagersReady(page, timeout);
    if (!managersReady) {
      console.warn('[E2E] setMapVisualizationMode: Managers not ready, attempting anyway');
    }
  }

  // Set the mode directly via JavaScript
  const result = await page.evaluate((targetMode) => {
    const win = window as any;
    try {
      if (win.viewportManager && typeof win.viewportManager.setMapMode === 'function') {
        win.viewportManager.setMapMode(targetMode);
        return { success: true, mode: targetMode };
      }
      return { success: false, error: 'viewportManager.setMapMode not available' };
    } catch (error) {
      return { success: false, error: String(error) };
    }
  }, mode);

  if (!result.success) {
    console.error(`[E2E] setMapVisualizationMode: Failed - ${result.error}`);
    return false;
  }

  // Wait for button to reflect the mode change
  try {
    await page.waitForFunction(
      (expectedMode) => {
        const btn = document.querySelector(`#map-mode-${expectedMode}`);
        return btn?.classList.contains('active');
      },
      mode,
      { timeout: TIMEOUTS.SHORT }
    );
    return true;
  } catch {
    console.warn(`[E2E] setMapVisualizationMode: Button did not get 'active' class`);
    return false;
  }
}

/**
 * Click a map mode button with proper initialization and retry logic.
 * This is more robust than a simple click, handling CI timing issues.
 *
 * @param page - Playwright page
 * @param mode - Visualization mode to activate
 * @param options - Configuration options
 */
export async function clickMapModeButton(
  page: Page,
  mode: 'points' | 'clusters' | 'heatmap' | 'hexagons',
  options: {
    waitForManagers?: boolean;
    waitForActive?: boolean;
    timeout?: number;
    useDirectJs?: boolean;
  } = {}
): Promise<void> {
  const {
    waitForManagers = true,
    waitForActive = true,
    timeout = TIMEOUTS.MEDIUM,
    useDirectJs = false
  } = options;

  // If direct JS is preferred (more reliable in CI), use that
  if (useDirectJs) {
    const success = await setMapVisualizationMode(page, mode, { waitForManagers, timeout });
    if (!success) {
      throw new Error(`Failed to set map mode to ${mode} via JavaScript`);
    }
    return;
  }

  // Ensure managers are ready
  if (waitForManagers) {
    await waitForMapManagersReady(page, timeout);
  }

  // Find and click the button
  const buttonSelector = `#map-mode-${mode}`;
  const button = page.locator(buttonSelector);

  // Wait for button to be visible
  await expect(button).toBeVisible({ timeout: TIMEOUTS.SHORT });

  // Click without force - let Playwright handle actionability
  await button.click();

  // Wait for active class if requested
  if (waitForActive) {
    await page.waitForFunction(
      (selector) => {
        const btn = document.querySelector(selector);
        return btn?.classList.contains('active');
      },
      buttonSelector,
      { timeout }
    );
  }
}

/**
 * Enable hexagon mode with full initialization waiting.
 * This is the recommended way to enable hexagon mode in tests.
 *
 * Strategy:
 * 1. Wait for managers to be ready (viewportManager exposed to window)
 * 2. Try direct JavaScript call via viewportManager.setMapMode()
 * 3. Fall back to click-based approach if JS fails
 *
 * @param page - Playwright page
 * @param options - Configuration options
 */
export async function enableHexagonMode(
  page: Page,
  options: {
    timeout?: number;
  } = {}
): Promise<void> {
  const { timeout = TIMEOUTS.MEDIUM } = options;

  // Dismiss any toasts that may block the button
  await page.evaluate(() => {
    const toasts = document.querySelectorAll('#toast-container .toast');
    toasts.forEach(toast => toast.remove());
  });

  // Wait for managers to be ready (exposed to window via Object.defineProperty)
  const managersReady = await waitForMapManagersReady(page, timeout);

  if (managersReady) {
    // Try direct JavaScript call - more reliable than click in CI
    const result = await page.evaluate(() => {
      const win = window as any;
      try {
        if (win.viewportManager && typeof win.viewportManager.setMapMode === 'function') {
          win.viewportManager.setMapMode('hexagons');
          return { success: true };
        }
        return { success: false, error: 'viewportManager.setMapMode not available' };
      } catch (error) {
        return { success: false, error: String(error) };
      }
    });

    if (result.success) {
      // Wait for button to reflect the mode change
      await page.waitForFunction(() => {
        const btn = document.querySelector('#map-mode-hexagons');
        return btn?.classList.contains('active');
      }, { timeout });
      return;
    }
    console.warn(`[E2E] enableHexagonMode: JS call failed (${result.error}), trying click fallback`);
  } else {
    console.warn('[E2E] enableHexagonMode: Managers not ready, trying click fallback');
  }

  // Fallback: Click-based approach
  const hexagonButton = page.locator('#map-mode-hexagons');
  await expect(hexagonButton).toBeVisible({ timeout: TIMEOUTS.SHORT });
  await hexagonButton.click({ force: true });

  // Wait for button to become active after click
  await page.waitForFunction(() => {
    const btn = document.querySelector('#map-mode-hexagons');
    return btn?.classList.contains('active');
  }, { timeout });
}

/**
 * Toggle arc overlay with proper initialization waiting.
 * Uses force:true click to bypass overlapping elements.
 *
 * @param page - Playwright page
 * @param enable - Whether to enable (true) or disable (false) the arc overlay
 * @param options - Configuration options
 */
export async function toggleArcOverlay(
  page: Page,
  enable: boolean = true,
  options: {
    timeout?: number;
  } = {}
): Promise<void> {
  const { timeout = TIMEOUTS.MEDIUM } = options;

  // Dismiss any toasts that may block the button
  await page.evaluate(() => {
    const toasts = document.querySelectorAll('#toast-container .toast');
    toasts.forEach(toast => toast.remove());
  });

  const arcToggle = page.locator('#arc-toggle');

  // Check current state
  const isCurrentlyActive = await arcToggle.evaluate((el) => el.classList.contains('active'));

  // Only click if we need to change state
  if (enable !== isCurrentlyActive) {
    // DETERMINISTIC FIX: Use JavaScript click to bypass MapLibre controls/SVG icons
    // that may intercept pointer events (more reliable than force:true in CI)
    await page.evaluate(() => {
      const btn = document.querySelector('#arc-toggle') as HTMLButtonElement;
      if (btn) btn.click();
    });
  }

  // Wait for button to reflect the expected state
  if (enable) {
    await page.waitForFunction(
      () => {
        const btn = document.querySelector('#arc-toggle');
        return btn?.classList.contains('active');
      },
      { timeout }
    );
  }
}
