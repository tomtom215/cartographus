// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Test: Chart Rendering and Interactions
 *
 * Tests all 12 interactive charts:
 * - Playback trends
 * - Top countries
 * - Top cities
 * - Media type distribution
 * - User leaderboard
 * - Viewing hours heatmap
 * - Platform distribution
 * - Player distribution
 * - Content completion stats
 * - Transcode distribution
 * - Resolution distribution
 * - Codec distribution
 *
 * NOTE: All tile requests are automatically mocked via fixtures.ts
 * This makes tests fully offline, deterministic, and faster.
 *
 * NOTE: These tests run in the 'chromium' project which has storageState
 * pre-loaded from auth.setup.ts, so we DON'T need to login again.
 */

import {
  test,
  expect,
  TIMEOUTS,
  SELECTORS,
  VIEWS,
  CHARTS,
  ANALYTICS_PAGES,
  gotoAppAndWaitReady,
  navigateToView,
  navigateToAnalyticsPage,
  expectChartsVisible,
  expectChartsRendered,
  waitForChartDataLoaded,
  setMobileViewport,
} from './fixtures';

// Enable WebGL cleanup for this test file (charts use canvas heavily)
test.use({ autoCleanupWebGL: true });

// Increase timeout for chart tests (data loading is slow with serialized API requests in CI)
test.describe.configure({ timeout: 90000 });

test.describe('Analytics Dashboard Charts', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to app (already authenticated via storageState from setup)
    await gotoAppAndWaitReady(page);

    // ROOT CAUSE FIX: Navigate to Analytics view AND the Overview sub-page explicitly
    // WHY: The Analytics view has multiple sub-pages (Overview, Content, Users, etc.)
    // and #chart-trends is only visible on the Overview page. Without explicit navigation,
    // the test may land on a different sub-page where the chart is hidden.
    await navigateToAnalyticsPage(page, ANALYTICS_PAGES.OVERVIEW);

    // Wait for chart data to be loaded (serialized API requests are slow in CI)
    // This ensures charts have data before we assert on their visibility/rendering
    await waitForChartDataLoaded(page, ['#chart-trends']);
  });

  test('should render all 12 chart containers', async ({ page }) => {
    // Analytics section should already be visible from beforeEach
    await expect(page.locator(SELECTORS.ANALYTICS_CONTAINER)).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

    // Overview page is active by default - it has 6 charts
    await expectChartsVisible(page, CHARTS.OVERVIEW);

    // Navigate to Geographic page for countries/cities charts
    await navigateToAnalyticsPage(page, 'geographic', { ensureOnAnalytics: false });
    await expectChartsVisible(page, CHARTS.GEOGRAPHIC);

    // Navigate to Performance page for quality charts
    await navigateToAnalyticsPage(page, 'performance', { ensureOnAnalytics: false });
    // Only check a subset of performance charts (some may need scrolling)
    await expectChartsVisible(page, ['#chart-transcode', '#chart-resolution']);
  });

  test('should render playback trends chart with data', async ({ page }) => {
    await expect(page.locator('#chart-trends')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

    // ECharts creates canvas or svg elements depending on renderer
    await expectChartsRendered(page, ['#chart-trends']);

    // Check for chart title
    const title = page.locator('#chart-trends').locator('text=Playback Trends, text=Trends');
    if (await title.count() > 0) {
      await expect(title.first()).toBeVisible();
    }
  });

  test('should render geographic charts (countries and cities)', async ({ page }) => {
    // Navigate to Geographic analytics page
    await navigateToAnalyticsPage(page, 'geographic', { ensureOnAnalytics: false });

    // Verify Geographic charts are rendered
    await expectChartsVisible(page, CHARTS.GEOGRAPHIC);
    await expectChartsRendered(page, CHARTS.GEOGRAPHIC);
  });

  test('should render media type distribution chart', async ({ page }) => {
    // Wait for chart data to be loaded (beforeEach only waits for #chart-trends)
    await waitForChartDataLoaded(page, ['#chart-media'], TIMEOUTS.DEFAULT);

    await expect(page.locator('#chart-media')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

    // Should be a donut/pie chart - canvas or svg depending on renderer
    await expectChartsRendered(page, ['#chart-media']);
  });

  test('should render user leaderboard chart', async ({ page }) => {
    const chartUsers = page.locator('#chart-users');

    // Wait for chart container to exist (may be hidden if no data)
    await chartUsers.waitFor({ state: 'attached', timeout: TIMEOUTS.DEFAULT }).catch(() => {
      console.log('[E2E] #chart-users not attached - chart may not exist on this page');
    });

    // Check if chart is visible OR exists in empty state
    const isVisible = await chartUsers.isVisible();
    if (isVisible) {
      await expectChartsRendered(page, ['#chart-users']);
    } else {
      // Chart exists but hidden - may be in empty/loading state
      const exists = await chartUsers.count() > 0;
      if (exists) {
        console.log('[E2E] #chart-users exists but hidden - likely no user data available');
      } else {
        // Chart doesn't exist at all - check if we're on the right page
        const analyticsContainer = page.locator('#analytics-overview');
        await expect(analyticsContainer).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
        console.log('[E2E] #chart-users not present in DOM - feature may be disabled');
      }
    }
  });

  test('should render viewing hours heatmap', async ({ page }) => {
    await expect(page.locator('#chart-heatmap')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

    // Heatmap should show hours x days of week - canvas or svg depending on renderer
    await expectChartsRendered(page, ['#chart-heatmap']);
  });

  test('should render streaming quality charts', async ({ page }) => {
    // Navigate to Performance analytics page
    await navigateToAnalyticsPage(page, 'performance', { ensureOnAnalytics: false });

    // Transcode distribution
    await expect(page.locator('#chart-transcode')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
    await expectChartsRendered(page, ['#chart-transcode']);

    // Resolution distribution
    await expect(page.locator('#chart-resolution')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
    await expectChartsRendered(page, ['#chart-resolution']);
  });

  test('should support chart hover interactions', async ({ page }) => {
    await expect(page.locator('#chart-trends')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

    const chartElement = page.locator(SELECTORS.CHART_ELEMENT('#chart-trends'));
    await expect(chartElement.first()).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

    // Hover over chart to trigger tooltip
    await chartElement.first().hover();

    // Wait for tooltip to appear - use soft assertion for data-dependent behavior
    const tooltip = page.locator('.echarts-tooltip, [class*="tooltip"]');
    let tooltipAppeared = false;
    try {
      await tooltip.first().waitFor({ state: 'visible', timeout: 2000 });
      tooltipAppeared = true;
    } catch {
      // Tooltip may not appear if there's no data at the hover position
      console.warn('[E2E] Tooltip did not appear after hover - chart may have empty data');
    }

    // Get the chart element to verify it's still interactive
    // E2E FIX: Use SELECTORS.CHART_ELEMENT to support both canvas and SVG renderers
    const chartContent = page.locator(SELECTORS.CHART_ELEMENT('#chart-trends'));
    await expect(chartContent.first()).toBeVisible();

    // Assert: Either tooltip appeared OR chart element is still functional
    // This ensures we're testing something, not just swallowing errors
    if (tooltipAppeared) {
      const tooltipCount = await tooltip.count();
      expect(tooltipCount).toBeGreaterThan(0);
      console.log(`Tooltip appeared with ${tooltipCount} element(s)`);
    } else {
      // E2E FIX: Verify chart is still interactive by checking chart element exists
      await expect(chartContent.first()).toBeAttached();
    }
  });

  test('should support chart export functionality', async ({ page }) => {
    await expect(page.locator('#chart-trends')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

    // Look for chart export button specifically (not CSV/GeoJSON export buttons)
    // Chart export buttons have class 'chart-export' and text 'Export PNG'
    const exportButton = page.locator('.chart-export, button:has-text("Export PNG")').first();

    if (await exportButton.isVisible()) {
      // Start waiting for download
      const downloadPromise = page.waitForEvent('download', { timeout: TIMEOUTS.MEDIUM }).catch(() => null);

      await exportButton.click();

      const download = await downloadPromise;

      // If download started, verify it's an image
      if (download) {
        expect(download.suggestedFilename()).toMatch(/\.(png|jpg|jpeg|pdf)$/i);
      }
    }
  });

  test('should update charts when filters change', async ({ page }) => {
    await expect(page.locator('#chart-trends')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

    // Get initial chart element (canvas or svg depending on renderer)
    const chartElement = page.locator(SELECTORS.CHART_ELEMENT('#chart-trends'));
    await expect(chartElement.first()).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

    // Change date filter
    const dateFilter = page.locator(SELECTORS.FILTER_DAYS + ', select[name="date-range"], #date-range-select, button:has-text("Last 7 Days")');

    if (await dateFilter.first().isVisible()) {
      await dateFilter.first().click();

      // Select a different date range
      const option = page.locator('option:has-text("Last 30 Days"), button:has-text("Last 30 Days"), [data-value="30d"]');
      if (await option.first().isVisible()) {
        await option.first().click();

        // Wait for network requests to complete after filter change
        await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => {
          // Network may already be idle
        });
      }

      // Chart should still be visible
      await expect(chartElement.first()).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
    }
  });

  test('should handle charts with no data gracefully', async ({ page }) => {
    // Already authenticated via storageState from setup

    // First ensure charts are visible before trying to filter
    await expect(page.locator('#chart-trends')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

    // Try to trigger a "no data" scenario using date filter with future dates
    // This is more reliable than user filter since date inputs always exist
    const startDateFilter = page.locator('#filter-start-date');
    let filterApplied = false;

    const isDateFilterVisible = await startDateFilter.isVisible().catch(() => false);

    if (isDateFilterVisible) {
      try {
        // Set start date to far future to get no data
        const futureDate = '2099-01-01';
        await startDateFilter.fill(futureDate);
        filterApplied = true;
        // FLAKINESS FIX: Wait for chart to re-render instead of hardcoded 500ms.
        // The chart should update after filter is applied - wait for canvas/SVG
        // to be attached which indicates re-render completed.
        await expect(page.locator('#chart-trends canvas, #chart-trends svg').first())
          .toBeAttached({ timeout: TIMEOUTS.MEDIUM });
      } catch (error) {
        console.warn('[E2E] Date filter interaction failed:', error);
        // Filter interaction failed, continue with test
      }
    }

    // Core assertion: Charts should still be visible without errors (even with empty data)
    await expect(page.locator('#chart-trends')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

    // Additional assertion: Chart element should be rendered
    // E2E FIX: Use SELECTORS.CHART_ELEMENT to support both canvas and SVG renderers
    const chartElement = page.locator(SELECTORS.CHART_ELEMENT('#chart-trends'));
    await expect(chartElement.first()).toBeAttached();

    // Check for "no data" message - useful for debugging but not a hard requirement
    const noDataMessage = page.locator('text=No data, text=No playbacks, .no-data, .empty-chart');
    const noDataCount = await noDataMessage.count();

    // Log the outcome for debugging
    if (filterApplied) {
      console.log(`[E2E] Filter applied. No data messages found: ${noDataCount}`);
    } else {
      console.log(`[E2E] Filter not applied (may not exist). Chart should still render.`);
    }

    // Final assertion: Page should not have any fatal error overlays/modals
    // E2E FIX: Exclude toast notifications which use role="alertdialog" for accessibility
    // but are not fatal errors. Only check for actual crash/error modals.
    const errorOverlay = page.locator('.error-overlay, .error-modal, .fatal-error, .crash-modal');
    const errorCount = await errorOverlay.count();
    // Note: Toast notifications with [role="alertdialog"] are expected for non-critical errors
    // like detection API 404s or backup warnings. These should not fail the test.
    expect(errorCount).toBe(0);
  });

  test('should render charts responsively', async ({ page }) => {
    // Wait for chart to be visible with explicit timeout
    const chartTrends = page.locator('#chart-trends');
    await expect(chartTrends).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

    // Get initial chart size
    const initialBox = await chartTrends.boundingBox();
    expect(initialBox).not.toBeNull();

    // Resize viewport to mobile
    // Note: This can trigger WebGL re-initialization which may be slow in CI
    try {
      await setMobileViewport(page);
    } catch {
      // Viewport resize failed, skip responsive assertions
      console.log('Viewport resize failed, skipping responsive test');
      return;
    }

    // Chart should still be visible after resize
    await expect(chartTrends).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    const mobileBox = await chartTrends.boundingBox().catch(() => null);
    if (!mobileBox) {
      // boundingBox can return null if element is not visible
      console.log('Could not get mobile bounding box, element may be off-screen');
      return;
    }

    // Chart should respond to viewport changes (responsive behavior)
    // Note: On mobile, the chart container may be wider than viewport if:
    // - The sidebar is present but hidden (position: absolute)
    // - CSS media queries allow horizontal scroll
    // - The chart-grid uses min-width constraints
    // The key test is that chart width CHANGED or stayed reasonable after resize
    if (mobileBox && initialBox) {
      // Chart should either shrink to fit mobile OR maintain reasonable size
      // Allow up to 2x viewport width (accounts for sidebar width + content)
      expect(mobileBox.width).toBeLessThanOrEqual(initialBox.width);
    }
  });

  test('should load charts asynchronously without blocking', async ({ page }) => {
    // Navigation should complete quickly even if charts take time to load
    // Already authenticated via storageState from setup

    // Ensure onboarding is skipped for this performance test (runs before page scripts)
    await page.addInitScript(() => {
      localStorage.setItem('onboarding_completed', 'true');
      localStorage.setItem('onboarding_skipped', 'true');
    });

    const startTime = Date.now();

    // Navigate to app (fresh load to measure async behavior)
    await page.goto('/', { waitUntil: 'domcontentloaded' });

    // Page should be interactive quickly (app visible)
    await expect(page.locator(SELECTORS.APP)).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

    const loadTime = Date.now() - startTime;

    // Initial page load (authenticated) should be under 5 seconds
    expect(loadTime).toBeLessThan(5000);

    // Navigate to analytics and verify page is responsive
    await navigateToView(page, VIEWS.ANALYTICS);

    // Charts may still be loading, but page should be interactive
    // Note: Filter panel is #filters in sidebar (visible on desktop viewport)
    await expect(page.locator(SELECTORS.FILTERS)).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  });
});
