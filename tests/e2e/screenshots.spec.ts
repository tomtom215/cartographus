// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import { type Page } from '@playwright/test';
import {
  test,
  expect,
  TIMEOUTS,
  SELECTORS,
  waitForMapReady,
  waitForGlobeReady,
  waitForChartsReady,
  waitForAppReady,
  waitForNavReady,
} from './fixtures';

/**
 * UI Screenshot Capture Tests
 *
 * Automatically captures screenshots of all major UI screens during CI/CD builds
 * to track the evolution of the UI/UX over time.
 *
 * Prerequisites:
 * - Requires auth.setup.ts to run first (establishes authentication)
 * - Uses saved authentication state from playwright/.auth/user.json
 * - Uses fixtures for automatic tile and API mocking
 *
 * Screenshots are saved to web/screenshots/ directory.
 *
 * IMPORTANT: This file uses ./fixtures which automatically mocks:
 * - Map tiles (for consistent rendering)
 * - API endpoints (for populated data displays)
 */

// Define screenshot configurations
interface ScreenshotConfig {
  name: string;
  url: string;
  selector: string;
  description: string;
  action: string | null;
  requiresAuth: boolean;
  waitForReady?: 'map' | 'globe' | 'charts' | 'app';
  // Custom timeout in ms (default: 90000)
  timeout?: number;
}

// Screenshots ordered to minimize WebGL resource accumulation:
// 1. Non-WebGL tests first (login, activity, recently-added, server-info)
// 2. Light map tests (filters-panel needs map visible but doesn't stress WebGL)
// 3. Heavy WebGL tests last (map-view, analytics-dashboard, globe-view, globe-controls)
// This ensures that if browser crashes on heavy tests, all lightweight screenshots are captured.
const screenshots: ScreenshotConfig[] = [
  // === Non-WebGL Screenshots (run first) ===
  {
    name: 'login-page',
    url: '/',
    selector: '#login-container',
    description: 'Login page with authentication form',
    action: null,
    requiresAuth: false,
  },
  {
    name: 'live-activity',
    url: '/',
    selector: '#activity-container',
    description: 'Live activity view from Tautulli',
    action: 'clickLiveActivityTab',
    requiresAuth: true,
    waitForReady: 'app',
  },
  {
    name: 'recently-added',
    url: '/',
    selector: '#recently-added-container',
    description: 'Recently added media from Plex',
    action: 'clickRecentlyAddedTab',
    requiresAuth: true,
    waitForReady: 'app',
  },
  {
    name: 'server-info',
    url: '/',
    selector: '#server-container',
    description: 'Server information and health dashboard',
    action: 'clickServerTab',
    requiresAuth: true,
    waitForReady: 'app',
  },
  {
    name: 'filters-panel',
    url: '/',
    selector: '#filters',
    description: 'Filter panel with 14+ filter dimensions',
    action: null,
    requiresAuth: true,
    waitForReady: 'app',
  },
  // === Light WebGL Screenshots (map-based but lightweight) ===
  {
    name: 'timeline-controls',
    url: '/',
    selector: '#timeline-container',
    description: 'Timeline controls for temporal playback visualization',
    action: null,
    requiresAuth: true,
    waitForReady: 'map',
  },
  // === Heavy WebGL Screenshots (run last to avoid resource accumulation) ===
  {
    name: 'map-view',
    url: '/',
    selector: '#map',
    description: 'Main map view with playback locations',
    action: null,
    requiresAuth: true,
    waitForReady: 'map',
  },
  {
    name: 'analytics-overview',
    url: '/',
    selector: '#analytics-overview',
    description: 'Analytics Overview - Quick insights and key metrics',
    action: 'clickAnalyticsTab',
    requiresAuth: true,
    waitForReady: 'charts',
  },
  {
    name: 'analytics-content',
    url: '/',
    selector: '#analytics-content',
    description: 'Analytics Content - Library usage and popular content',
    action: 'clickAnalyticsContentTab',
    requiresAuth: true,
    waitForReady: 'charts',
  },
  {
    name: 'analytics-users',
    url: '/',
    selector: '#analytics-users',
    description: 'Analytics Users - User engagement and viewing habits',
    action: 'clickAnalyticsUsersTab',
    requiresAuth: true,
    waitForReady: 'charts',
  },
  {
    name: 'analytics-performance',
    url: '/',
    selector: '#analytics-performance',
    description: 'Analytics Performance - Transcode and streaming stats',
    action: 'clickAnalyticsPerformanceTab',
    requiresAuth: true,
    waitForReady: 'charts',
  },
  {
    name: 'analytics-geographic',
    url: '/',
    selector: '#analytics-geographic',
    description: 'Analytics Geographic - Countries and cities breakdown',
    action: 'clickAnalyticsGeographicTab',
    requiresAuth: true,
    waitForReady: 'charts',
  },
  {
    name: 'analytics-advanced',
    url: '/',
    selector: '#analytics-advanced',
    description: 'Analytics Advanced - Comparative and temporal analysis',
    action: 'clickAnalyticsAdvancedTab',
    requiresAuth: true,
    waitForReady: 'charts',
  },
  {
    name: 'analytics-library',
    url: '/',
    selector: '#analytics-library',
    description: 'Analytics Library - Per-library deep dive',
    action: 'clickAnalyticsLibraryTab',
    requiresAuth: true,
    waitForReady: 'charts',
  },
  {
    name: 'analytics-users-profile',
    url: '/',
    selector: '#analytics-users-profile',
    description: 'Analytics User Profile - Individual user dashboard',
    action: 'clickAnalyticsUsersProfileTab',
    requiresAuth: true,
    waitForReady: 'charts',
  },
  {
    name: 'settings-panel',
    url: '/',
    selector: '#settings-panel',
    description: 'Settings panel with theme, map, and data options',
    action: 'openSettingsPanel',
    requiresAuth: true,
    waitForReady: 'app',
  },
  {
    name: 'globe-view',
    url: '/',
    selector: '#globe',
    description: '3D globe view powered by deck.gl',
    action: 'clickGlobeToggle',
    requiresAuth: true,
    waitForReady: 'globe',
  },
  {
    name: 'globe-controls',
    url: '/',
    selector: '#globe-controls',
    description: 'Globe control panel with auto-rotate and reset view buttons',
    action: 'openGlobeControls',
    requiresAuth: true,
    waitForReady: 'globe',
  },
];

test.describe('UI Screenshots @screenshot', () => {
  // Default timeout for each test (screenshots can take time)
  // Increased to 90s since some views require loading multiple API calls
  test.setTimeout(90000);

  // Set up dark mode for all screenshot tests
  test.beforeEach(async ({ page }) => {
    // Emulate dark color scheme to ensure consistent dark theme screenshots
    await page.emulateMedia({ colorScheme: 'dark' });
  });

  // Clean up WebGL resources after each test to prevent memory exhaustion
  test.afterEach(async ({ page }) => {
    try {
      if (page.isClosed()) return;
      const cleanedUp = await page.evaluate(() => {
        // Clean up WebGL contexts to prevent memory leaks
        const canvases = document.querySelectorAll('canvas');
        canvases.forEach(canvas => {
          const gl = canvas.getContext('webgl') || canvas.getContext('webgl2');
          if (gl && typeof (gl as any).getExtension === 'function') {
            const ext = (gl as any).getExtension('WEBGL_lose_context');
            if (ext) ext.loseContext();
          }
        });
      }).then(() => true).catch(() => false);
      if (!cleanedUp) {
        console.warn('[E2E] screenshots afterEach: WebGL cleanup failed');
      }
    } catch {
      // Ignore cleanup errors - browser context may be closing
    }
  });

  for (const screenshot of screenshots) {
    test(`capture ${screenshot.name}`, async ({ page }) => {
      // Use custom timeout if specified
      if (screenshot.timeout) {
        test.setTimeout(screenshot.timeout);
      }
      // Set viewport size for consistent screenshots
      await page.setViewportSize({ width: 1920, height: 1080 });

      console.log(`Capturing ${screenshot.name}: ${screenshot.description}`);

      // For login page only, clear authentication state
      // IMPORTANT: Must navigate to a valid page first before accessing localStorage
      if (!screenshot.requiresAuth) {
        await page.context().clearCookies();
        // Navigate first to have a valid origin for localStorage
        await page.goto(screenshot.url, { waitUntil: 'domcontentloaded', timeout: 30000 });
        // Now we can access localStorage
        await page.evaluate(() => {
          localStorage.removeItem('auth_token');
          localStorage.removeItem('auth_username');
          localStorage.removeItem('auth_expires_at');
        });
        // Reload to show login page
        await page.reload({ waitUntil: 'domcontentloaded' });
      } else {
        // Skip onboarding for authenticated tests (runs before page scripts)
        await page.addInitScript(() => {
          localStorage.setItem('onboarding_completed', 'true');
          localStorage.setItem('onboarding_skipped', 'true');
        });

        // Navigate to the page (authenticated)
        await page.goto(screenshot.url, { waitUntil: 'domcontentloaded', timeout: 30000 });

        // Verify WebGL is available (critical for MapLibre GL and deck.gl)
        const webglInfo = await page.evaluate(() => {
          try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
            if (!gl) return { supported: false, error: 'No WebGL context' };

            const webglContext = gl as WebGLRenderingContext;
            return {
              supported: true,
              vendor: webglContext.getParameter(webglContext.VENDOR),
              renderer: webglContext.getParameter(webglContext.RENDERER),
              version: webglContext.getParameter(webglContext.VERSION),
            };
          } catch (e) {
            return { supported: false, error: String(e) };
          }
        });

        if (!webglInfo.supported) {
          console.warn('WebGL NOT AVAILABLE:', webglInfo.error);
        } else {
          console.log('WebGL Available:', webglInfo.renderer);
        }

        // Verify login container is NOT visible
        await expect(page.locator('#login-container')).not.toBeVisible({ timeout: TIMEOUTS.MEDIUM });

        // Wait for main app container to be visible
        await expect(page.locator(SELECTORS.APP_VISIBLE)).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
        await waitForNavReady(page); // Wait for NavigationManager to initialize before clicking nav buttons
      }

      // Perform action if needed - DO NOT swallow errors
      if (screenshot.action) {
        await performAction(page, screenshot.action);
      }

      // Wait for specific ready state
      if (screenshot.waitForReady) {
        try {
          switch (screenshot.waitForReady) {
            case 'map':
              await waitForMapReady(page);
              break;
            case 'globe':
              await waitForGlobeReady(page);
              break;
            case 'charts':
              await waitForChartsReady(page);
              break;
            case 'app':
              await waitForAppReady(page);
              break;
          }
        } catch (error) {
          console.warn(`Wait for ${screenshot.waitForReady} timed out, continuing with screenshot`);
        }
      }

      // Additional wait for rendering
      await page.waitForTimeout(TIMEOUTS.WEBGL_INIT);

      // Take full-page screenshot
      await page.screenshot({
        path: `web/screenshots/${screenshot.name}.png`,
        fullPage: true,
      });

      // Verify the target element exists - fail test if not visible
      // This ensures the screenshot captured the intended content
      const element = page.locator(screenshot.selector).first();
      await expect(element).toBeVisible({
        timeout: TIMEOUTS.SHORT
      });
      console.log(`Captured screenshot: ${screenshot.name}`);
    });
  }
});

test.describe('UI Component Screenshots @screenshot', () => {
  test.setTimeout(60000);

  // Set up dark mode for all component screenshot tests
  test.beforeEach(async ({ page }) => {
    await page.emulateMedia({ colorScheme: 'dark' });
  });

  test('capture individual chart screenshots', async ({ page }) => {
    console.log('Capturing individual chart screenshots from all analytics pages');
    test.setTimeout(180000); // Extended timeout for comprehensive chart capture

    await page.setViewportSize({ width: 1920, height: 1080 });

    // Skip onboarding (runs before page scripts)
    await page.addInitScript(() => {
      localStorage.setItem('onboarding_completed', 'true');
      localStorage.setItem('onboarding_skipped', 'true');
    });

    await page.goto('/', { waitUntil: 'domcontentloaded', timeout: 30000 });

    // Verify authentication
    await expect(page.locator('#login-container')).not.toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    await expect(page.locator(SELECTORS.APP_VISIBLE)).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
    await waitForNavReady(page);

    // Navigate to analytics section
    // Navigate using JavaScript click for CI reliability
    // WHY: Playwright's .click() may fail in headless/SwiftShader environments
    await page.evaluate(() => {
      const btn = document.querySelector('button[data-view="analytics"], .nav-tab[data-view="analytics"]') as HTMLElement;
      if (btn) btn.click();
    });
    await page.waitForTimeout(TIMEOUTS.DATA_LOAD);

    // Define charts by analytics page for comprehensive capture
    const chartsByPage: Record<string, string[]> = {
      'overview': [
        'chart-trends', 'chart-media', 'chart-users', 'chart-heatmap',
        'chart-platforms', 'chart-players'
      ],
      'content': [
        'chart-libraries', 'chart-ratings', 'chart-duration', 'chart-years', 'chart-codec',
        'chart-popular-movies', 'chart-popular-shows', 'chart-popular-episodes'
      ],
      'users': [
        'chart-engagement-summary', 'chart-engagement-hours', 'chart-engagement-days',
        'chart-completion', 'chart-binge-summary', 'chart-binge-shows', 'chart-binge-users',
        'chart-watch-parties-summary', 'chart-watch-parties-content', 'chart-watch-parties-users'
      ],
      'performance': [
        'chart-bandwidth-trends', 'chart-bandwidth-transcode', 'chart-bandwidth-resolution', 'chart-bandwidth-users',
        'chart-bitrate-distribution', 'chart-bitrate-utilization', 'chart-bitrate-resolution',
        'chart-transcode', 'chart-resolution', 'chart-resolution-mismatch',
        'chart-hdr-analytics', 'chart-audio-analytics', 'chart-subtitle-analytics',
        'chart-connection-security', 'chart-concurrent-streams', 'chart-pause-patterns'
      ],
      'geographic': [
        'chart-countries', 'chart-cities'
      ],
      'advanced': [
        'chart-comparative-metrics', 'chart-comparative-content', 'chart-comparative-users',
        'chart-temporal-heatmap', 'chart-hardware-transcode', 'chart-abandonment'
      ],
      'library': [
        'chart-library-users', 'chart-library-trend', 'chart-library-quality'
      ],
      'users-profile': [
        'chart-user-activity-trend', 'chart-user-top-content', 'chart-user-platforms'
      ]
    };

    let totalCaptured = 0;
    let totalCharts = 0;

    for (const [pageName, chartIds] of Object.entries(chartsByPage)) {
      console.log(`\nNavigating to analytics/${pageName}...`);
      totalCharts += chartIds.length;

      // Navigate to the specific analytics sub-page
      const pageTab = page.locator(`.analytics-tab[data-analytics-page="${pageName}"]`);
      if (await pageTab.isVisible({ timeout: TIMEOUTS.WEBGL_INIT }).catch(() => false)) {
        await pageTab.click();
        await page.waitForTimeout(TIMEOUTS.DATA_LOAD + 500);

        // Wait for charts to render
        try {
          await waitForChartsReady(page);
        } catch {
          console.warn(`Charts on ${pageName} may not have fully rendered`);
        }

        // Capture each chart on this page
        for (const chartId of chartIds) {
          const chart = page.locator(`#${chartId}`);
          const isVisible = await chart.isVisible().catch(() => false);

          if (isVisible) {
            // Wait for canvas to render
            // ROOT CAUSE FIX: Accept both 'canvas' and 'svg' renderers - ChartManager uses SVG for small datasets
            const canvas = chart.locator('canvas, svg');
            if (await canvas.count() > 0) {
              await page.waitForTimeout(TIMEOUTS.ANIMATION);
            }

            await chart.screenshot({
              path: `web/screenshots/${chartId}.png`,
            });
            console.log(`  Captured: ${chartId}`);
            totalCaptured++;
          } else {
            console.warn(`  Not visible: ${chartId}`);
          }
        }
      } else {
        console.warn(`Tab not found: ${pageName}`);
      }
    }

    console.log(`\nTotal charts captured: ${totalCaptured}/${totalCharts}`);
    // We expect at least 20 charts to be captured (some pages may have conditional rendering)
    expect(totalCaptured).toBeGreaterThan(20);
  });

  test('capture mobile viewport screenshots', async ({ page }) => {
    console.log('Capturing mobile viewport');

    // Mobile viewport (iPhone 12 Pro)
    await page.setViewportSize({ width: 390, height: 844 });

    // Skip onboarding (runs before page scripts)
    await page.addInitScript(() => {
      localStorage.setItem('onboarding_completed', 'true');
      localStorage.setItem('onboarding_skipped', 'true');
    });

    await page.goto('/', { waitUntil: 'domcontentloaded', timeout: 30000 });

    // Verify authentication
    await expect(page.locator('#login-container')).not.toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    await expect(page.locator(SELECTORS.APP_VISIBLE)).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
    await waitForNavReady(page); // Wait for NavigationManager to initialize before clicking nav buttons

    // Wait for map to render
    try {
      await waitForMapReady(page);
    } catch {
      console.warn('Map may not have fully rendered');
    }

    await page.screenshot({
      path: 'web/screenshots/mobile-map-view.png',
      fullPage: true,
    });

    console.log('Captured mobile screenshot');
  });

  test('capture tablet viewport screenshots', async ({ page }) => {
    console.log('Capturing tablet viewport');

    // Tablet viewport (iPad Pro)
    await page.setViewportSize({ width: 1024, height: 1366 });

    // Skip onboarding (runs before page scripts)
    await page.addInitScript(() => {
      localStorage.setItem('onboarding_completed', 'true');
      localStorage.setItem('onboarding_skipped', 'true');
    });

    await page.goto('/', { waitUntil: 'domcontentloaded', timeout: 30000 });

    // Verify authentication
    await expect(page.locator('#login-container')).not.toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    await expect(page.locator(SELECTORS.APP_VISIBLE)).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
    await waitForNavReady(page); // Wait for NavigationManager to initialize before clicking nav buttons

    // Wait for map to render
    try {
      await waitForMapReady(page);
    } catch {
      console.warn('Map may not have fully rendered');
    }

    await page.screenshot({
      path: 'web/screenshots/tablet-map-view.png',
      fullPage: true,
    });

    console.log('Captured tablet screenshot');
  });
});

test.describe('UI State Screenshots @screenshot', () => {
  test.setTimeout(60000);

  // Set up dark mode for all state screenshot tests
  test.beforeEach(async ({ page }) => {
    await page.emulateMedia({ colorScheme: 'dark' });
  });

  test('capture loading states', async ({ page }) => {
    console.log('Capturing loading state');

    await page.setViewportSize({ width: 1920, height: 1080 });

    // Clear authentication to show login page
    await page.context().clearCookies();
    // Navigate first to have a valid origin for localStorage
    await page.goto('/');
    // Now we can safely clear localStorage
    await page.evaluate(() => {
      localStorage.clear();
    });
    // Reload to apply the cleared state
    await page.reload({ waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(TIMEOUTS.RENDER);

    // Capture login page state
    await page.screenshot({
      path: 'web/screenshots/loading-state.png',
      fullPage: false,
    });

    console.log('Captured loading state screenshot');
  });

  // Disable auto API mocking for empty state test
  test.describe(() => {
    test.use({ autoMockApi: false });

    test('capture empty state (no data)', async ({ page }) => {
      console.log('Capturing empty state');

      await page.setViewportSize({ width: 1920, height: 1080 });

      // Import and use the empty API mocking
      const { setupEmptyApiMocking } = await import('./fixtures/api-mocks');
      await setupEmptyApiMocking(page);

      // Skip onboarding (runs before page scripts)
      await page.addInitScript(() => {
        localStorage.setItem('onboarding_completed', 'true');
        localStorage.setItem('onboarding_skipped', 'true');
      });

      await page.goto('/', { waitUntil: 'domcontentloaded', timeout: 30000 });

      // Verify authentication
      await expect(page.locator('#login-container')).not.toBeVisible({ timeout: TIMEOUTS.MEDIUM });
      await expect(page.locator(SELECTORS.APP_VISIBLE)).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
      await waitForNavReady(page); // Wait for NavigationManager to initialize before clicking nav buttons

      await page.waitForTimeout(TIMEOUTS.WEBGL_INIT);

      await page.screenshot({
        path: 'web/screenshots/empty-state.png',
        fullPage: true,
      });

      console.log('Captured empty state screenshot');
    });
  });

  // Disable auto API mocking for error state test
  test.describe(() => {
    test.use({ autoMockApi: false });

    test('capture error state', async ({ page }) => {
      console.log('Capturing error state');

      await page.setViewportSize({ width: 1920, height: 1080 });

      // Import and use the error API mocking
      const { setupErrorApiMocking } = await import('./fixtures/api-mocks');
      await setupErrorApiMocking(page);

      // Skip onboarding (runs before page scripts)
      await page.addInitScript(() => {
        localStorage.setItem('onboarding_completed', 'true');
        localStorage.setItem('onboarding_skipped', 'true');
      });

      await page.goto('/', { waitUntil: 'domcontentloaded', timeout: 30000 });

      // Verify authentication
      await expect(page.locator('#login-container')).not.toBeVisible({ timeout: TIMEOUTS.MEDIUM });
      await expect(page.locator(SELECTORS.APP_VISIBLE)).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
      await waitForNavReady(page); // Wait for NavigationManager to initialize before clicking nav buttons

      await page.waitForTimeout(TIMEOUTS.WEBGL_INIT);

      await page.screenshot({
        path: 'web/screenshots/error-state.png',
        fullPage: false,
      });

      console.log('Captured error state screenshot');
    });
  });
});

/**
 * Helper function to perform UI actions
 *
 * Uses button[data-view="..."] selectors which are consistent with
 * other working tests (08-live-activity.spec.ts, 09-recently-added.spec.ts, etc.)
 *
 * IMPORTANT: No try-catch error swallowing - if an action fails, the test should fail
 * so we know the screenshot won't show the intended content.
 */
async function performAction(page: Page, action: string): Promise<void> {
  const shortTimeout = TIMEOUTS.DEFAULT;  // 10s - Increased for CI stability
  const mediumTimeout = TIMEOUTS.LONG;    // 15s

  switch (action) {
    case 'clickGlobeToggle': {
      // Clean up any existing WebGL resources before initializing new globe
      await page.evaluate(() => {
        const canvases = document.querySelectorAll('canvas');
        canvases.forEach(canvas => {
          const gl = canvas.getContext('webgl') || canvas.getContext('webgl2');
          if (gl && typeof (gl as any).getExtension === 'function') {
            const ext = (gl as any).getExtension('WEBGL_lose_context');
            if (ext) ext.loseContext();
          }
        });
      });
      await page.waitForTimeout(TIMEOUTS.ANIMATION / 3);

      const globeToggle = page.locator(SELECTORS.VIEW_MODE_3D);
      await expect(globeToggle).toBeVisible({ timeout: shortTimeout });
      await globeToggle.click();
      // Wait for WebGL globe to initialize
      await page.waitForTimeout(TIMEOUTS.WEBGL_INIT + 1000);
      await expect(page.locator(SELECTORS.GLOBE)).toBeVisible({ timeout: mediumTimeout });
      break;
    }

    case 'clickAnalyticsTab': {
      // Use button[data-view="..."] selector - consistent with working tests
      const analyticsTab = page.locator('button[data-view="analytics"], .nav-tab[data-view="analytics"]').first();
      await expect(analyticsTab).toBeVisible({ timeout: shortTimeout });
      await analyticsTab.click();
      await page.waitForTimeout(TIMEOUTS.DATA_LOAD);
      await expect(page.locator('#analytics-container')).toBeVisible({ timeout: mediumTimeout });
      break;
    }

    case 'clickLiveActivityTab': {
      // Clean up WebGL resources before switching to activity view to prevent memory exhaustion
      await page.evaluate(() => {
        const canvases = document.querySelectorAll('canvas');
        canvases.forEach(canvas => {
          const gl = canvas.getContext('webgl') || canvas.getContext('webgl2');
          if (gl && typeof (gl as any).getExtension === 'function') {
            const ext = (gl as any).getExtension('WEBGL_lose_context');
            if (ext) ext.loseContext();
          }
        });
      });
      await page.waitForTimeout(TIMEOUTS.ANIMATION / 3); // Brief pause for cleanup

      // Use button[data-view="..."] selector - consistent with 08-live-activity.spec.ts
      const activityTab = page.locator('button[data-view="activity"], .nav-tab[data-view="activity"]').first();
      await expect(activityTab).toBeVisible({ timeout: shortTimeout });
      await activityTab.click();
      await page.waitForTimeout(TIMEOUTS.DATA_LOAD);
      await expect(page.locator('#activity-container')).toBeVisible({ timeout: mediumTimeout });
      break;
    }

    case 'clickRecentlyAddedTab': {
      // Use button[data-view="..."] selector - consistent with 09-recently-added.spec.ts
      const recentTab = page.locator('button[data-view="recently-added"], .nav-tab[data-view="recently-added"]').first();
      await expect(recentTab).toBeVisible({ timeout: shortTimeout });
      await recentTab.click();
      await page.waitForTimeout(TIMEOUTS.DATA_LOAD);
      await expect(page.locator('#recently-added-container')).toBeVisible({ timeout: mediumTimeout });
      break;
    }

    case 'clickServerTab': {
      // Use button[data-view="..."] selector - consistent with 10-server-info.spec.ts
      const serverTab = page.locator('button[data-view="server"], .nav-tab[data-view="server"]').first();
      await expect(serverTab).toBeVisible({ timeout: shortTimeout });
      await serverTab.click();
      await page.waitForTimeout(TIMEOUTS.DATA_LOAD);
      await expect(page.locator('#server-container')).toBeVisible({ timeout: mediumTimeout });
      break;
    }

    case 'openGlobeControls': {
      // Clean up any previous WebGL resources before initializing globe
      await page.evaluate(() => {
        const canvases = document.querySelectorAll('canvas');
        canvases.forEach(canvas => {
          const gl = canvas.getContext('webgl') || canvas.getContext('webgl2');
          if (gl && typeof (gl as any).getExtension === 'function') {
            const ext = (gl as any).getExtension('WEBGL_lose_context');
            if (ext) ext.loseContext();
          }
        });
      });
      await page.waitForTimeout(TIMEOUTS.ANIMATION / 3);

      const globeBtn = page.locator(SELECTORS.VIEW_MODE_3D);
      await expect(globeBtn).toBeVisible({ timeout: shortTimeout });
      await globeBtn.click();
      // Wait for globe to fully initialize with controls
      await page.waitForTimeout(TIMEOUTS.WEBGL_INIT + 1000);
      await expect(page.locator('#globe-controls')).toBeVisible({ timeout: mediumTimeout });
      break;
    }

    // === Analytics Sub-Tab Actions ===
    case 'clickAnalyticsContentTab': {
      // First navigate to analytics section
      const analyticsTab = page.locator('button[data-view="analytics"], .nav-tab[data-view="analytics"]').first();
      await expect(analyticsTab).toBeVisible({ timeout: shortTimeout });
      await analyticsTab.click();
      await page.waitForTimeout(TIMEOUTS.RENDER);
      // Then click Content sub-tab
      const contentTab = page.locator('.analytics-tab[data-analytics-page="content"]');
      await expect(contentTab).toBeVisible({ timeout: shortTimeout });
      await contentTab.click();
      await page.waitForTimeout(TIMEOUTS.DATA_LOAD);
      await expect(page.locator('#analytics-content')).toBeVisible({ timeout: mediumTimeout });
      break;
    }

    case 'clickAnalyticsUsersTab': {
      const analyticsTab = page.locator('button[data-view="analytics"], .nav-tab[data-view="analytics"]').first();
      await expect(analyticsTab).toBeVisible({ timeout: shortTimeout });
      await analyticsTab.click();
      await page.waitForTimeout(TIMEOUTS.RENDER);
      const usersTab = page.locator('.analytics-tab[data-analytics-page="users"]');
      await expect(usersTab).toBeVisible({ timeout: shortTimeout });
      await usersTab.click();
      await page.waitForTimeout(TIMEOUTS.DATA_LOAD);
      await expect(page.locator('#analytics-users')).toBeVisible({ timeout: mediumTimeout });
      break;
    }

    case 'clickAnalyticsPerformanceTab': {
      const analyticsTab = page.locator('button[data-view="analytics"], .nav-tab[data-view="analytics"]').first();
      await expect(analyticsTab).toBeVisible({ timeout: shortTimeout });
      await analyticsTab.click();
      await page.waitForTimeout(TIMEOUTS.RENDER);
      const perfTab = page.locator('.analytics-tab[data-analytics-page="performance"]');
      await expect(perfTab).toBeVisible({ timeout: shortTimeout });
      await perfTab.click();
      await page.waitForTimeout(TIMEOUTS.DATA_LOAD);
      await expect(page.locator('#analytics-performance')).toBeVisible({ timeout: mediumTimeout });
      break;
    }

    case 'clickAnalyticsGeographicTab': {
      const analyticsTab = page.locator('button[data-view="analytics"], .nav-tab[data-view="analytics"]').first();
      await expect(analyticsTab).toBeVisible({ timeout: shortTimeout });
      await analyticsTab.click();
      await page.waitForTimeout(TIMEOUTS.RENDER);
      const geoTab = page.locator('.analytics-tab[data-analytics-page="geographic"]');
      await expect(geoTab).toBeVisible({ timeout: shortTimeout });
      await geoTab.click();
      await page.waitForTimeout(TIMEOUTS.DATA_LOAD);
      await expect(page.locator('#analytics-geographic')).toBeVisible({ timeout: mediumTimeout });
      break;
    }

    case 'clickAnalyticsAdvancedTab': {
      const analyticsTab = page.locator('button[data-view="analytics"], .nav-tab[data-view="analytics"]').first();
      await expect(analyticsTab).toBeVisible({ timeout: shortTimeout });
      await analyticsTab.click();
      await page.waitForTimeout(TIMEOUTS.RENDER);
      const advancedTab = page.locator('.analytics-tab[data-analytics-page="advanced"]');
      await expect(advancedTab).toBeVisible({ timeout: shortTimeout });
      await advancedTab.click();
      await page.waitForTimeout(TIMEOUTS.DATA_LOAD);
      await expect(page.locator('#analytics-advanced')).toBeVisible({ timeout: mediumTimeout });
      break;
    }

    case 'clickAnalyticsLibraryTab': {
      const analyticsTab = page.locator('button[data-view="analytics"], .nav-tab[data-view="analytics"]').first();
      await expect(analyticsTab).toBeVisible({ timeout: shortTimeout });
      await analyticsTab.click();
      await page.waitForTimeout(TIMEOUTS.RENDER);
      const libraryTab = page.locator('.analytics-tab[data-analytics-page="library"]');
      await expect(libraryTab).toBeVisible({ timeout: shortTimeout });
      await libraryTab.click();
      await page.waitForTimeout(TIMEOUTS.DATA_LOAD);
      await expect(page.locator('#analytics-library')).toBeVisible({ timeout: mediumTimeout });
      break;
    }

    case 'clickAnalyticsUsersProfileTab': {
      const analyticsTab = page.locator('button[data-view="analytics"], .nav-tab[data-view="analytics"]').first();
      await expect(analyticsTab).toBeVisible({ timeout: shortTimeout });
      await analyticsTab.click();
      await page.waitForTimeout(TIMEOUTS.RENDER);
      const usersProfileTab = page.locator('.analytics-tab[data-analytics-page="users-profile"]');
      await expect(usersProfileTab).toBeVisible({ timeout: shortTimeout });
      await usersProfileTab.click();
      await page.waitForTimeout(TIMEOUTS.DATA_LOAD);
      await expect(page.locator('#analytics-users-profile')).toBeVisible({ timeout: mediumTimeout });
      break;
    }

    case 'openSettingsPanel': {
      const settingsBtn = page.locator('#settings-button');
      await expect(settingsBtn).toBeVisible({ timeout: shortTimeout });
      await settingsBtn.click();
      await page.waitForTimeout(TIMEOUTS.RENDER);
      await expect(page.locator('#settings-panel')).toBeVisible({ timeout: mediumTimeout });
      break;
    }

    default:
      throw new Error(`Unknown action: ${action}`);
  }
}
