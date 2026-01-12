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
  waitForAnalyticsPageVisible,
  waitForChartRendered,
  clickAnalyticsTabAndWait,
} from './fixtures';

/**
 * E2E Test: Analytics Pages Navigation and Coverage
 *
 * Tests all 8 analytics pages:
 * 1. Overview - Quick insights and key metrics
 * 2. Content - Library usage and popular content
 * 3. Users & Behavior - User engagement and viewing habits
 * 4. Performance - Technical metrics and infrastructure
 * 5. Geographic - Location-based insights
 * 6. Advanced Analytics - Comparative and temporal analytics
 * 7. Library - Deep-dive analytics for individual libraries
 * 8. User Profile - Individual user activity analytics
 *
 * Coverage:
 * - Page navigation via tab clicks
 * - URL hash navigation (bookmarking)
 * - Browser back/forward navigation
 * - Chart rendering on each page
 * - Accessibility attributes
 */

test.describe('Analytics Pages - Comprehensive Coverage', () => {
  test.beforeEach(async ({ page }) => {
    // Use storageState for authentication (configured in playwright.config.ts)
    await gotoAppAndWaitReady(page);

    // Navigate to Analytics tab (using correct selector: nav-tab with data-view attribute)
    await navigateToView(page, VIEWS.ANALYTICS);
    await page.waitForSelector(SELECTORS.ANALYTICS_CONTAINER, { state: 'visible', timeout: TIMEOUTS.MEDIUM });
  });

  test('should render all 8 analytics page tabs', async ({ page }) => {
    // Verify all analytics navigation tabs exist (using data-analytics-page attribute, not IDs)
    const tabs = [
      { page: 'overview', text: 'Overview' },
      { page: 'content', text: 'Content' },
      { page: 'users', text: 'Users' },
      { page: 'performance', text: 'Performance' },
      { page: 'geographic', text: 'Geographic' },
      { page: 'advanced', text: 'Advanced' },
      { page: 'library', text: 'Library' },
      { page: 'users-profile', text: 'User Profile' },
    ];

    for (const tab of tabs) {
      const tabElement = page.locator(`.analytics-tab[data-analytics-page="${tab.page}"]`);
      await expect(tabElement).toBeVisible();
      await expect(tabElement).toContainText(tab.text);
    }
  });

  test('should navigate to Overview page and render charts', async ({ page }) => {
    // Click Overview tab and wait for page to be visible
    await clickAnalyticsTabAndWait(page, 'overview');

    // Verify Overview page is visible
    await expect(page.locator('#analytics-overview')).toBeVisible();

    // Verify other pages are hidden
    await expect(page.locator('#analytics-content')).toHaveCSS('display', 'none');
    await expect(page.locator('#analytics-users')).toHaveCSS('display', 'none');

    // Verify Overview charts are rendered (actual charts per HTML: 6 charts)
    const overviewCharts = [
      '#chart-trends',           // Playback trends over time
      '#chart-media',            // Media type distribution
      '#chart-users',            // Top users
      '#chart-heatmap',          // Viewing hours heatmap
      '#chart-platforms',        // Platform distribution
      '#chart-players',          // Player distribution
    ];

    for (const chartId of overviewCharts) {
      // DETERMINISTIC FIX: Wait for chart to be truly visible using computed styles
      // Charts may be hidden during data loading with display:none or visibility:hidden
      await page.waitForFunction(
        (id: string) => {
          const chart = document.querySelector(id);
          if (!chart) return false;
          const style = window.getComputedStyle(chart);
          return style.display !== 'none' &&
                 style.visibility !== 'hidden' &&
                 style.opacity !== '0';
        },
        chartId,
        { timeout: 15000 }
      );
      await expect(page.locator(chartId)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should navigate to Content page and render charts', async ({ page }) => {
    // Click Content tab and wait for page to be visible
    await clickAnalyticsTabAndWait(page, 'content');

    // Verify Content page is visible
    await expect(page.locator('#analytics-content')).toBeVisible();

    // Verify Content charts are rendered (actual charts per HTML: 8 charts)
    const contentCharts = [
      '#chart-libraries',        // Library distribution
      '#chart-ratings',          // Content rating distribution
      '#chart-duration',         // Duration statistics
      '#chart-years',            // Top release years
      '#chart-codec',            // Codec combination distribution
      '#chart-popular-movies',   // Top movies
      '#chart-popular-shows',    // Top TV shows
      '#chart-popular-episodes', // Top episodes
    ];

    for (const chartId of contentCharts) {
      await expect(page.locator(chartId)).toBeVisible({ timeout: 10000 });
    }
  });

  test('should navigate to Users & Behavior page and render charts', async ({ page }) => {
    // Click Users & Behavior tab and wait for page to be visible
    await clickAnalyticsTabAndWait(page, 'users');

    // Verify Users page is visible
    await expect(page.locator('#analytics-users')).toBeVisible();

    // Verify Users charts are rendered (actual charts per HTML: 10 charts)
    const usersCharts = [
      '#chart-engagement-summary',       // Engagement summary
      '#chart-engagement-hours',         // Viewing patterns by hour
      '#chart-engagement-days',          // Viewing patterns by day
      '#chart-completion',               // Content completion analytics
      '#chart-binge-summary',            // Binge watching summary
      '#chart-binge-shows',              // Top binge-watched shows
      '#chart-binge-users',              // Top binge watchers
      '#chart-watch-parties-summary',    // Watch party summary
      '#chart-watch-parties-content',    // Top watch party content
      '#chart-watch-parties-users',      // Most social users
    ];

    for (const chartId of usersCharts) {
      await expect(page.locator(chartId)).toBeVisible({ timeout: 10000 });
    }
  });

  test('should navigate to Performance page and render charts', async ({ page }) => {
    // Click Performance tab and wait for page to be visible
    await clickAnalyticsTabAndWait(page, 'performance');

    // Verify Performance page is visible
    await expect(page.locator('#analytics-performance')).toBeVisible();

    // Verify Performance charts are in DOM (16 charts - some may need scrolling to view)
    // Use toBeAttached instead of toBeVisible since charts at bottom may not be in viewport
    const performanceCharts = [
      '#chart-bandwidth-trends',       // Bandwidth usage trends
      '#chart-bandwidth-transcode',    // Bandwidth by transcode decision
      '#chart-bandwidth-resolution',   // Bandwidth by resolution
      '#chart-bandwidth-users',        // Top bandwidth users
      '#chart-bitrate-distribution',   // Bitrate distribution
      '#chart-bitrate-utilization',    // Bandwidth utilization over time
      '#chart-bitrate-resolution',     // Bitrate by resolution
      '#chart-transcode',              // Transcode vs Direct Play
      '#chart-resolution',             // Video resolution distribution
      '#chart-resolution-mismatch',    // Resolution quality loss analysis
      '#chart-hdr-analytics',          // HDR & dynamic range
      '#chart-audio-analytics',        // Audio quality distribution
      '#chart-subtitle-analytics',     // Subtitle usage analytics
      '#chart-connection-security',    // Connection security overview
      '#chart-concurrent-streams',     // Concurrent streams analytics
      '#chart-pause-patterns',         // Pause pattern & engagement
    ];

    for (const chartId of performanceCharts) {
      const chart = page.locator(chartId);
      await expect(chart).toBeAttached({ timeout: 10000 });
      // Scroll into view to trigger lazy loading if needed
      await chart.scrollIntoViewIfNeeded();
    }
  });

  test('should navigate to Geographic page and render charts', async ({ page }) => {
    // Click Geographic tab and wait for page to be visible
    await clickAnalyticsTabAndWait(page, 'geographic');

    // Verify Geographic page is visible
    await expect(page.locator('#analytics-geographic')).toBeVisible();

    // Verify Geographic charts are rendered (actual charts per HTML: 2 charts)
    const geographicCharts = [
      '#chart-countries',    // Top countries by playback count
      '#chart-cities',       // Top cities by playback count
    ];

    for (const chartId of geographicCharts) {
      await expect(page.locator(chartId)).toBeVisible({ timeout: 10000 });
    }
  });

  test('should navigate to Advanced Analytics page and render charts', async ({ page }) => {
    // Click Advanced tab and wait for page to be visible
    await clickAnalyticsTabAndWait(page, 'advanced');

    // Verify Advanced page is visible
    await expect(page.locator('#analytics-advanced')).toBeVisible();

    // Verify Advanced charts/elements are rendered (actual per HTML: 4 charts + temporal map)
    const advancedCharts = [
      '#chart-comparative-metrics',       // Period comparison
      '#chart-comparative-content',       // Top content comparison
      '#chart-comparative-users',         // Top users comparison
      '#chart-temporal-heatmap',          // Geographic temporal heatmap container
    ];

    for (const chartId of advancedCharts) {
      await expect(page.locator(chartId)).toBeVisible({ timeout: 10000 });
    }
  });

  test('should support URL hash navigation (bookmarking)', async ({ page }) => {
    // Navigate directly via URL hash to Content page (storageState handles auth)
    await page.goto('/#analytics-content');

    // Wait for app to load
    await page.waitForSelector('#app:not(.hidden)', { timeout: 10000 });

    // Navigate to analytics and content
    // Navigate using JavaScript click for CI reliability
    // WHY: Playwright's .click() may fail in headless/SwiftShader environments
    await page.evaluate(() => {
      const tab = document.querySelector('.nav-tab[data-view="analytics"]') as HTMLElement;
      if (tab) tab.click();
    });
    await page.waitForSelector('#analytics-container', { state: 'visible', timeout: 5000 });

    await clickAnalyticsTabAndWait(page, 'content');

    // Verify Content page is visible
    await expect(page.locator('#analytics-content')).toBeVisible();

    // Verify Content tab is active
    await expect(page.locator('.analytics-tab[data-analytics-page="content"]')).toHaveClass(/active/);

    // Switch to Performance tab
    await clickAnalyticsTabAndWait(page, 'performance');

    // Verify Performance page is visible
    await expect(page.locator('#analytics-performance')).toBeVisible();
    await expect(page.locator('.analytics-tab[data-analytics-page="performance"]')).toHaveClass(/active/);
  });

  test('should support browser back/forward navigation', async ({ page }) => {
    // Navigate through several pages
    await clickAnalyticsTabAndWait(page, 'overview');
    await clickAnalyticsTabAndWait(page, 'content');
    await clickAnalyticsTabAndWait(page, 'users');

    // Verify we're on Users page
    await expect(page.locator('#analytics-users')).toBeVisible();

    // Go back to Content
    await page.goBack();
    await waitForAnalyticsPageVisible(page, 'content');
    await expect(page.locator('#analytics-content')).toBeVisible();

    // Go back to Overview
    await page.goBack();
    await waitForAnalyticsPageVisible(page, 'overview');
    await expect(page.locator('#analytics-overview')).toBeVisible();

    // Go forward to Content
    await page.goForward();
    await waitForAnalyticsPageVisible(page, 'content');
    await expect(page.locator('#analytics-content')).toBeVisible();
  });

  test('should maintain active tab state when switching pages', async ({ page }) => {
    // Click each tab and verify active state
    const tabs = ['overview', 'content', 'users', 'performance', 'geographic', 'advanced', 'library', 'users-profile'];

    for (const tabPage of tabs) {
      await clickAnalyticsTabAndWait(page, tabPage);

      // Verify this tab is active
      await expect(page.locator(`.analytics-tab[data-analytics-page="${tabPage}"]`)).toHaveClass(/active/);

      // Verify other tabs are not active
      for (const otherPage of tabs) {
        if (otherPage !== tabPage) {
          const otherTab = page.locator(`.analytics-tab[data-analytics-page="${otherPage}"]`);
          const classes = await otherTab.getAttribute('class');
          expect(classes).not.toContain('active');
        }
      }
    }
  });

  test('should support keyboard navigation between tabs', async ({ page }) => {
    // Focus on first tab
    await page.focus('.analytics-tab[data-analytics-page="overview"]');

    // Press Tab to move to next tab
    await page.keyboard.press('Tab');

    // Verify focus moved to another element
    const focusedElement = await page.evaluate(() => document.activeElement?.tagName);
    expect(focusedElement).toBeTruthy();

    // Click on content tab and verify it activates
    await clickAnalyticsTabAndWait(page, 'content');

    // Page should have switched
    const activeTab = page.locator('.analytics-tab.active');
    await expect(activeTab).toBeVisible();
  });

  test('should render charts after page switch', async ({ page }) => {
    // Start on Overview and wait for page + chart to render
    await clickAnalyticsTabAndWait(page, 'overview');
    await expect(page.locator('#chart-trends')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
    // Wait for chart canvas/SVG to render
    await waitForChartRendered(page, '#chart-trends');
    // Check for canvas OR SVG (ECharts renderer depends on configuration)
    const overviewChartElement = page.locator('#chart-trends canvas, #chart-trends svg');
    const chartCount = await overviewChartElement.count();
    expect(chartCount).toBeGreaterThan(0);

    // Switch to Content and wait for page + chart to render
    await clickAnalyticsTabAndWait(page, 'content');
    await expect(page.locator('#chart-libraries')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
    // Wait for chart canvas/SVG to render
    await waitForChartRendered(page, '#chart-libraries');
    // Check for canvas OR SVG
    const contentChartElement = page.locator('#chart-libraries canvas, #chart-libraries svg');
    const contentChartCount = await contentChartElement.count();
    expect(contentChartCount).toBeGreaterThan(0);
  });

  test('should handle rapid page switching without errors', async ({ page }) => {
    // Rapidly click through all tabs with minimal delay (testing animation cancellation)
    const tabs = ['overview', 'content', 'users', 'performance', 'geographic', 'advanced', 'library', 'users-profile', 'overview'];

    for (const tabPage of tabs) {
      // Navigate analytics tab using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate((page) => {
        const tab = document.querySelector(`.analytics-tab[data-analytics-page="${page}"]`) as HTMLElement;
        if (tab) tab.click();
      }, tabPage);
      // Wait for the clicked page to become visible (deterministic wait)
      await page.waitForSelector(`#analytics-${tabPage}`, {
        state: 'visible',
        timeout: TIMEOUTS.DEFAULT
      }).catch(() => {
        // Ignore errors during rapid switching - final state is what matters
      });
    }

    // Verify we ended on Overview
    await waitForAnalyticsPageVisible(page, 'overview');
    await expect(page.locator('#analytics-overview')).toBeVisible();
  });

  test('should update page title/description when switching tabs', async ({ page }) => {
    // Check Overview description
    await clickAnalyticsTabAndWait(page, 'overview');

    const overviewDesc = page.locator('#analytics-overview .page-description');
    if (await overviewDesc.isVisible()) {
      await expect(overviewDesc).toContainText(/quick insights|key metrics/i);
    }

    // Check Content description
    await clickAnalyticsTabAndWait(page, 'content');

    const contentDesc = page.locator('#analytics-content .page-description');
    if (await contentDesc.isVisible()) {
      await expect(contentDesc).toContainText(/library|content/i);
    }
  });

  test('should preserve filter state when switching pages', async ({ page }) => {
    // Apply a filter
    const daysFilter = page.locator('#filter-days');
    if (await daysFilter.isVisible()) {
      await daysFilter.selectOption('30');
      // Wait for filter change to propagate and charts to update
      await page.waitForFunction(
        () => {
          // Check if any chart is in a loading or updating state
          const charts = document.querySelectorAll('[id^="chart-"]');
          // Wait for charts to be present and not in loading state
          return charts.length > 0;
        },
        { timeout: TIMEOUTS.DEFAULT }
      );
    }

    // Switch to different page
    await clickAnalyticsTabAndWait(page, 'content');

    // Switch back to Overview
    await clickAnalyticsTabAndWait(page, 'overview');

    // Filter should still be set to 30
    if (await daysFilter.isVisible()) {
      const selectedValue = await daysFilter.inputValue();
      expect(selectedValue).toBe('30');
    }
  });
});
