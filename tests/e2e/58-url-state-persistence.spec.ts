// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import {
  test,
  expect,
  TIMEOUTS,
  SELECTORS,
  VIEWS,
  gotoAppAndWaitReady,
  navigateToView,
} from './fixtures';

/**
 * E2E Test: URL State Persistence (Task 26)
 *
 * Tests URL hash-based navigation for main dashboard views:
 * - Maps: / (default, no hash)
 * - Activity: #activity
 * - Analytics: #analytics-{page}
 * - Recently Added: #recently-added
 * - Server: #server
 *
 * Coverage:
 * - Direct URL navigation with hash
 * - View switching updates URL hash
 * - Browser back/forward navigation
 * - Page refresh preserves view state
 */

test.describe('URL State Persistence', () => {
  test.beforeEach(async ({ page }) => {
    await gotoAppAndWaitReady(page);
  });

  test('should load default maps view with no hash', async ({ page }) => {
    // Verify maps view is active by default
    const mapsTab = page.locator(`[data-testid="nav-tab-maps"]`);
    await expect(mapsTab).toHaveAttribute('aria-selected', 'true');

    // URL should have no hash (or empty hash)
    const url = page.url();
    expect(url.includes('#activity')).toBeFalsy();
    expect(url.includes('#server')).toBeFalsy();
    expect(url.includes('#recently-added')).toBeFalsy();
  });

  test('should navigate to activity view and update URL hash', async ({ page }) => {
    // Navigate to activity view
    await navigateToView(page, VIEWS.ACTIVITY);
    await page.waitForSelector(SELECTORS.ACTIVITY_CONTAINER, {
      state: 'visible',
      timeout: TIMEOUTS.MEDIUM
    });

    // Verify activity tab is active
    const activityTab = page.locator(`[data-testid="nav-tab-activity"]`);
    await expect(activityTab).toHaveAttribute('aria-selected', 'true');

    // URL should have #activity hash
    await page.waitForFunction(() => window.location.hash === '#activity', {
      timeout: TIMEOUTS.SHORT
    });
    expect(page.url()).toContain('#activity');
  });

  test('should navigate to server view and update URL hash', async ({ page }) => {
    // Navigate to server view
    await navigateToView(page, VIEWS.SERVER);
    await page.waitForSelector(SELECTORS.SERVER_CONTAINER, {
      state: 'visible',
      timeout: TIMEOUTS.MEDIUM
    });

    // Verify server tab is active
    const serverTab = page.locator(`[data-testid="nav-tab-server"]`);
    await expect(serverTab).toHaveAttribute('aria-selected', 'true');

    // URL should have #server hash
    await page.waitForFunction(() => window.location.hash === '#server', {
      timeout: TIMEOUTS.SHORT
    });
    expect(page.url()).toContain('#server');
  });

  test('should navigate to recently-added view and update URL hash', async ({ page }) => {
    // Navigate to recently-added view
    await navigateToView(page, VIEWS.RECENTLY_ADDED);
    await page.waitForSelector(SELECTORS.RECENTLY_ADDED_CONTAINER, {
      state: 'visible',
      timeout: TIMEOUTS.MEDIUM
    });

    // Verify recently-added tab is active
    const recentlyAddedTab = page.locator(`[data-testid="nav-tab-recently-added"]`);
    await expect(recentlyAddedTab).toHaveAttribute('aria-selected', 'true');

    // URL should have #recently-added hash
    await page.waitForFunction(() => window.location.hash === '#recently-added', {
      timeout: TIMEOUTS.SHORT
    });
    expect(page.url()).toContain('#recently-added');
  });

  test('should load activity view directly from URL hash', async ({ page }) => {
    // Navigate directly with hash
    await page.goto('/#activity');
    await page.waitForSelector('#app', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

    // Wait for the view to switch
    await page.waitForSelector(SELECTORS.ACTIVITY_CONTAINER, {
      state: 'visible',
      timeout: TIMEOUTS.MEDIUM
    });

    // Verify activity tab is active
    const activityTab = page.locator(`[data-testid="nav-tab-activity"]`);
    await expect(activityTab).toHaveAttribute('aria-selected', 'true');
  });

  test('should load server view directly from URL hash', async ({ page }) => {
    // Navigate directly with hash
    await page.goto('/#server');
    await page.waitForSelector('#app', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

    // Wait for the view to switch
    await page.waitForSelector(SELECTORS.SERVER_CONTAINER, {
      state: 'visible',
      timeout: TIMEOUTS.MEDIUM
    });

    // Verify server tab is active
    const serverTab = page.locator(`[data-testid="nav-tab-server"]`);
    await expect(serverTab).toHaveAttribute('aria-selected', 'true');
  });

  test('should load analytics page directly from URL hash', async ({ page }) => {
    // Navigate directly to analytics content page
    await page.goto('/#analytics-content');
    await page.waitForSelector('#app', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

    // Wait for analytics container and content page
    await page.waitForSelector(SELECTORS.ANALYTICS_CONTAINER, {
      state: 'visible',
      timeout: TIMEOUTS.MEDIUM
    });

    // Verify analytics tab is active
    const analyticsTab = page.locator(`[data-testid="nav-tab-analytics"]`);
    await expect(analyticsTab).toHaveAttribute('aria-selected', 'true');

    // Verify content analytics sub-tab is active
    const contentTab = page.locator(`.analytics-tab[data-analytics-page="content"]`);
    await expect(contentTab).toHaveAttribute('aria-selected', 'true');
  });

  test('should support browser back/forward navigation between views', async ({ page }) => {
    // Start on maps (default)
    const mapsTab = page.locator(`[data-testid="nav-tab-maps"]`);
    await expect(mapsTab).toHaveAttribute('aria-selected', 'true');

    // Navigate to activity
    await navigateToView(page, VIEWS.ACTIVITY);
    await page.waitForSelector(SELECTORS.ACTIVITY_CONTAINER, {
      state: 'visible',
      timeout: TIMEOUTS.MEDIUM
    });

    // Navigate to server
    await navigateToView(page, VIEWS.SERVER);
    await page.waitForSelector(SELECTORS.SERVER_CONTAINER, {
      state: 'visible',
      timeout: TIMEOUTS.MEDIUM
    });

    // Go back (should be activity)
    await page.goBack();
    await page.waitForSelector(SELECTORS.ACTIVITY_CONTAINER, {
      state: 'visible',
      timeout: TIMEOUTS.MEDIUM
    });
    const activityTab = page.locator(`[data-testid="nav-tab-activity"]`);
    await expect(activityTab).toHaveAttribute('aria-selected', 'true');

    // Go forward (should be server)
    await page.goForward();
    await page.waitForSelector(SELECTORS.SERVER_CONTAINER, {
      state: 'visible',
      timeout: TIMEOUTS.MEDIUM
    });
    const serverTab = page.locator(`[data-testid="nav-tab-server"]`);
    await expect(serverTab).toHaveAttribute('aria-selected', 'true');
  });

  test('should clear hash when returning to maps view', async ({ page }) => {
    // Navigate to activity first
    await navigateToView(page, VIEWS.ACTIVITY);
    await page.waitForFunction(() => window.location.hash === '#activity', {
      timeout: TIMEOUTS.SHORT
    });

    // Navigate back to maps
    await navigateToView(page, VIEWS.MAPS);
    await page.waitForSelector(SELECTORS.MAP_CONTAINER, {
      state: 'visible',
      timeout: TIMEOUTS.MEDIUM
    });

    // Hash should be empty (maps is default)
    await page.waitForFunction(() => window.location.hash === '' || window.location.hash === '#', {
      timeout: TIMEOUTS.SHORT
    });
  });

  test('should preserve view after page refresh', async ({ page }) => {
    // Navigate to server view
    await navigateToView(page, VIEWS.SERVER);
    await page.waitForSelector(SELECTORS.SERVER_CONTAINER, {
      state: 'visible',
      timeout: TIMEOUTS.MEDIUM
    });

    // Verify URL has hash
    expect(page.url()).toContain('#server');

    // Refresh the page
    await page.reload();
    await page.waitForSelector('#app', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

    // Wait for server container to be visible again
    await page.waitForSelector(SELECTORS.SERVER_CONTAINER, {
      state: 'visible',
      timeout: TIMEOUTS.MEDIUM
    });

    // Verify server tab is still active
    const serverTab = page.locator(`[data-testid="nav-tab-server"]`);
    await expect(serverTab).toHaveAttribute('aria-selected', 'true');
  });
});
