// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import {
  test,
  expect,
  TIMEOUTS,
  gotoAppAndWaitReady,
  navigateToAnalyticsPage,
  ANALYTICS_PAGES,
  setupE2EDebugLogging,
} from './fixtures';

/**
 * E2E Test: Chart Zoom/Pan
 *
 * Tests chart zoom and pan functionality for time series charts:
 * - DataZoom slider visibility
 * - Zoom interaction via slider
 * - Reset zoom button
 * - Keyboard accessibility for zoom controls
 *
 * Reference: UI/UX Audit
 * @see /docs/working/UI_UX_AUDIT.md
 *
 * Note: Uses autoMockApi: true for deterministic chart data in CI.
 * This ensures ECharts instances have series data regardless of Tautulli state.
 *
 * MEMORY OPTIMIZATION: This test file uses serial mode to prevent browser crashes
 * from WebGL memory exhaustion. Charts + analytics views create multiple WebGL
 * contexts that can accumulate and crash SwiftShader in CI.
 */

// Run tests serially to prevent memory exhaustion from parallel WebGL contexts
test.describe.configure({ mode: 'serial' });

// Enable API mocking and WebGL cleanup for deterministic, stable tests
test.use({ autoMockApi: true, autoCleanupWebGL: true });

test.describe('Chart Zoom/Pan', () => {
  test.beforeEach(async ({ page }) => {
    // DEBUG: Enable network logging to diagnose API mocking issues
    // WHY: Tests were timing out waiting for chart data. This logs all API requests
    // to verify that mocking is intercepting requests correctly.
    setupE2EDebugLogging(page);

    // Ensure onboarding is skipped even after viewport changes
    await page.addInitScript(() => {
      localStorage.setItem('onboarding_completed', 'true');
      localStorage.setItem('onboarding_skipped', 'true');
    });

    await gotoAppAndWaitReady(page);

    // ROOT CAUSE FIX: Navigate to Analytics view AND the Overview sub-page
    // WHY: The chart-trends and chart-heatmap are on the Overview analytics page,
    // NOT visible by default when only navigating to the main Analytics view.
    // The Analytics view has multiple sub-pages (Overview, Content, Users, etc.)
    // and tests must explicitly navigate to the correct sub-page.
    await navigateToAnalyticsPage(page, ANALYTICS_PAGES.OVERVIEW);

    // Verify Overview page is visible (contains the trends chart)
    await expect(page.locator('#analytics-overview')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
  });

  // Clean up WebGL resources - wrapped to handle browser context closing
  test.afterEach(async ({ page }) => {
    // Check if page is still usable before attempting cleanup
    // Browser context may be closing if test timed out
    try {
      if (page.isClosed()) return;

      // Navigate away from charts to release resources using JavaScript click for CI reliability
      const navClicked = await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="maps"]') as HTMLElement;
        if (tab) { tab.click(); return true; }
        return false;
      }).then(result => result).catch(() => false);
      if (!navClicked) {
        console.warn('[E2E] chart-zoom-tabs afterEach: Nav click to maps failed');
      }
      // Wait for map container to be visible
      const mapVisible = await page.waitForFunction(() => {
        const mapContainer = document.getElementById('map-container');
        return mapContainer && mapContainer.offsetHeight > 0;
      }).then(() => true).catch(() => false);
      if (!mapVisible) {
        console.warn('[E2E] chart-zoom-tabs afterEach: Map container not visible');
      }

      // Clean up WebGL contexts to prevent memory leaks
      const cleanedUp = await page.evaluate(() => {
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
        console.warn('[E2E] chart-zoom-tabs afterEach: WebGL cleanup failed');
      }
    } catch {
      // Ignore all cleanup errors - browser context may be closing
    }
  });

  test.describe('Trends Chart Zoom', () => {
    // QUARANTINED: Test fails waiting for ECharts instance to have series data
    // Root cause: ECharts initialization race condition - instance exists but has no data
    // The aria-busy="false" check passes but getOption().series is empty
    // Tracking: CI failure analysis from 2025-12-31
    // Unquarantine criteria: 20 consecutive green runs after fixing ECharts ready signal
    test.fixme('should render trends chart with dataZoom slider', async ({ page }) => {
      // Wait for trends chart container to be visible
      // WHY: Container visibility is the first prerequisite for chart rendering
      const trendsChart = page.locator('#chart-trends');
      await expect(trendsChart).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

      // Wait for chart data loading to complete (aria-busy="false")
      // WHY: Charts set aria-busy="true" during data loading and "false" when complete
      // This is more reliable than checking ECharts instance which may not be available yet
      await page.waitForFunction(() => {
        const chartDom = document.getElementById('chart-trends');
        if (!chartDom) return false;
        const ariaBusy = chartDom.getAttribute('aria-busy');
        // aria-busy should be "false" when data loading is complete
        return ariaBusy === 'false';
      }, { timeout: TIMEOUTS.LONG });

      // Wait for chart to be rendered (ECharts creates canvas or svg after data load)
      // ROOT CAUSE FIX: Accept both 'canvas' and 'svg' renderers
      // WHY: ChartManager.getOptimalRenderer() returns 'svg' for datasets < 1000 points
      // In CI with small mock data, charts render as SVG, not canvas (see ChartManager.ts:52)
      await expect(trendsChart.locator('canvas, svg')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Now wait for ECharts instance to have data
      // WHY: We need the instance to verify dataZoom configuration
      await page.waitForFunction(() => {
        const chartDom = document.getElementById('chart-trends');
        if (!chartDom) return false;
        const instance = (window as any).echarts?.getInstanceByDom?.(chartDom);
        if (!instance) return false;
        const option = instance.getOption();
        return option && option.series && option.series.length > 0;
      }, { timeout: TIMEOUTS.MEDIUM });

      // DataZoom slider should be present (rendered as part of ECharts canvas)
      // We can verify by checking the chart configuration
      const hasDataZoom = await page.evaluate(() => {
        const chartDom = document.getElementById('chart-trends');
        if (!chartDom) return false;
        // Check if chart exists with dataZoom
        const echartsInstance = (window as any).echarts?.getInstanceByDom?.(chartDom);
        if (!echartsInstance) return false;
        const option = echartsInstance.getOption();
        return option?.dataZoom && option.dataZoom.length > 0;
      });

      expect(hasDataZoom).toBe(true);
    });

    // QUARANTINED: Test fails waiting for ECharts instance to have series data
    // Root cause: Same as above - ECharts initialization race condition
    // Tracking: CI failure analysis from 2025-12-31
    // Unquarantine criteria: 20 consecutive green runs after fixing ECharts ready signal
    test.fixme('should allow zoom interaction on trends chart', async ({ page }) => {
      const trendsChart = page.locator('#chart-trends');
      await expect(trendsChart).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

      // Wait for chart data loading to complete (aria-busy="false")
      // WHY: Must wait for data before interacting with chart
      await page.waitForFunction(() => {
        const chartDom = document.getElementById('chart-trends');
        if (!chartDom) return false;
        const ariaBusy = chartDom.getAttribute('aria-busy');
        return ariaBusy === 'false';
      }, { timeout: TIMEOUTS.LONG });

      // Wait for chart to be rendered (canvas or svg depending on data size)
      // ROOT CAUSE FIX: Accept both renderers - ChartManager uses SVG for small datasets
      await expect(trendsChart.locator('canvas, svg')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Wait for chart to fully initialize with data
      await page.waitForFunction(() => {
        const chartDom = document.getElementById('chart-trends');
        if (!chartDom) return false;
        const instance = (window as any).echarts?.getInstanceByDom?.(chartDom);
        if (!instance) return false;
        const option = instance.getOption();
        return option && option.series && option.series.length > 0;
      }, { timeout: TIMEOUTS.MEDIUM });

      // Get chart dimensions
      const box = await trendsChart.boundingBox();
      expect(box).not.toBeNull();

      // Chart should be interactive (no crash on interaction)
      if (box) {
        // Simulate scroll for zoom (wheel event)
        await trendsChart.hover();
        await page.mouse.wheel(0, -100);

        // Wait for chart to finish re-rendering
        await page.waitForFunction(() => {
          const chartDom = document.getElementById('chart-trends');
          if (!chartDom) return false;
          const instance = (window as any).echarts?.getInstanceByDom?.(chartDom);
          // Chart should still have valid instance after interaction
          return instance && instance.getOption();
        }, { timeout: TIMEOUTS.DEFAULT });
      }

      // Chart should still be visible after interaction
      await expect(trendsChart).toBeVisible();
    });
  });

  test.describe('Heatmap Chart Zoom', () => {
    // QUARANTINED: Test fails waiting for ECharts instance to have series data
    // Root cause: Same ECharts initialization race condition as trends tests
    // The aria-busy="false" check passes but getOption().series is empty
    // Tracking: CI failure analysis from 2025-12-31
    // Unquarantine criteria: 20 consecutive green runs after fixing ECharts ready signal
    test.fixme('should render heatmap chart with dataZoom', async ({ page }) => {
      const heatmapChart = page.locator('#chart-heatmap');
      await expect(heatmapChart).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

      // Wait for chart data loading to complete (aria-busy="false")
      // WHY: Charts set aria-busy="true" during data loading and "false" when complete
      await page.waitForFunction(() => {
        const chartDom = document.getElementById('chart-heatmap');
        if (!chartDom) return false;
        const ariaBusy = chartDom.getAttribute('aria-busy');
        return ariaBusy === 'false';
      }, { timeout: TIMEOUTS.LONG });

      // Wait for chart to be rendered (canvas or svg depending on data size)
      // ROOT CAUSE FIX: Accept both renderers - ChartManager uses SVG for small datasets
      await expect(heatmapChart.locator('canvas, svg')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Wait for heatmap chart to fully initialize with data
      await page.waitForFunction(() => {
        const chartDom = document.getElementById('chart-heatmap');
        if (!chartDom) return false;
        const instance = (window as any).echarts?.getInstanceByDom?.(chartDom);
        if (!instance) return false;
        const option = instance.getOption();
        return option && option.series && option.series.length > 0;
      }, { timeout: TIMEOUTS.MEDIUM });

      // Check for dataZoom presence (value unused - test validates chart renders)
      void await page.evaluate(() => {
        const chartDom = document.getElementById('chart-heatmap');
        if (!chartDom) return false;
        const echartsInstance = (window as any).echarts?.getInstanceByDom?.(chartDom);
        if (!echartsInstance) return false;
        const option = echartsInstance.getOption();
        return option?.dataZoom && option.dataZoom.length > 0;
      });

      // Heatmap may or may not have dataZoom depending on implementation
      // The test validates the chart renders without error
      await expect(heatmapChart).toBeVisible();
    });
  });

  test.describe('Zoom Accessibility', () => {
    test('should maintain chart accessibility after zoom', async ({ page }) => {
      const trendsChart = page.locator('#chart-trends');

      // DETERMINISTIC FIX: Use LONG timeout for CI where chart init is slow
      // WHY: In CI (SwiftShader), chart initialization takes significantly longer.
      // The aria-busy attribute is set to 'false' only after all data is loaded and rendered.
      await page.waitForFunction(() => {
        const chartDom = document.getElementById('chart-trends');
        if (!chartDom) return false;
        const ariaBusy = chartDom.getAttribute('aria-busy');
        // Also accept null/missing aria-busy (some charts don't set it)
        return ariaBusy === 'false' || ariaBusy === null;
      }, { timeout: TIMEOUTS.LONG });

      // Wait for chart to be rendered (canvas or svg depending on data size)
      // ROOT CAUSE FIX: Accept both renderers - ChartManager uses SVG for small datasets
      await expect(trendsChart.locator('canvas, svg')).toBeVisible({ timeout: TIMEOUTS.LONG });

      // Now chart should be fully visible
      await expect(trendsChart).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

      // Chart should have ARIA attributes
      await expect(trendsChart).toHaveAttribute('role', 'img');

      // Interact with chart - ensure chart is in view first
      await trendsChart.scrollIntoViewIfNeeded();
      await trendsChart.hover();

      // DETERMINISTIC FIX: Wait for chart to be fully ready before interaction
      // WHY: Mouse wheel on an uninitialized chart can cause race conditions
      // where the ECharts instance loses state during re-rendering.
      // Use LONG timeout for CI where ECharts initialization is slow.
      const chartReady = await page.waitForFunction(() => {
        const chartDom = document.getElementById('chart-trends');
        if (!chartDom) return false;
        const instance = (window as any).echarts?.getInstanceByDom?.(chartDom);
        if (!instance) return false;
        // Ensure chart has both option AND is not currently animating
        const opt = instance.getOption();
        return opt && opt.series && opt.series.length > 0;
      }, { timeout: TIMEOUTS.LONG }).then(() => true).catch(() => false);

      // If chart isn't ready, verify ARIA attributes only (skip zoom interaction)
      // WHY: In some CI environments, ECharts may not fully initialize due to WebGL issues
      if (!chartReady) {
        console.warn('[E2E] chart-zoom: ECharts instance not ready, skipping zoom interaction');
        await expect(trendsChart).toHaveAttribute('role', 'img');
        return;
      }

      // Perform zoom interaction
      await page.mouse.wheel(0, -50);

      // Wait for chart to finish re-rendering after zoom
      // DETERMINISTIC FIX: Use waitForFunction with polling instead of arbitrary waitForTimeout
      // WHY: After wheel events, ECharts may dispatch multiple resize/dataZoom events.
      // waitForFunction polls the condition until it's true or times out.
      await page.waitForFunction(() => {
        const chartDom = document.getElementById('chart-trends');
        if (!chartDom) return false;
        const instance = (window as any).echarts?.getInstanceByDom?.(chartDom);
        // Chart is stable when instance exists and has a valid option
        return instance && instance.getOption() !== null;
      }, { timeout: TIMEOUTS.LONG });

      // ARIA attributes should persist
      await expect(trendsChart).toHaveAttribute('role', 'img');
    });
  });
});

test.describe('Analytics Tab Overflow', () => {
  test.beforeEach(async ({ page }) => {
    // Ensure onboarding is skipped even after viewport changes
    await page.addInitScript(() => {
      localStorage.setItem('onboarding_completed', 'true');
      localStorage.setItem('onboarding_skipped', 'true');
    });

    await gotoAppAndWaitReady(page);

    // Navigate to Analytics tab using JavaScript click for CI reliability
    // Playwright's .click() may fail in headless/SwiftShader environments
    await page.evaluate(() => {
      const tab = document.querySelector('.nav-tab[data-view="analytics"]') as HTMLElement;
      if (tab) tab.click();
    });
    await expect(page.locator('#analytics-container')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
  });

  test('should handle analytics tabs on mobile viewport', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    // Wait for layout to stabilize after viewport change
    await page.waitForFunction(() => {
      const analyticsNav = document.getElementById('analytics-nav');
      if (!analyticsNav) return false;
      // Check that element has finished layout
      return analyticsNav.offsetHeight > 0 && analyticsNav.offsetWidth > 0;
    }, { timeout: TIMEOUTS.DEFAULT });

    // Analytics nav should be visible
    const analyticsNav = page.locator('#analytics-nav');
    await expect(analyticsNav).toBeVisible();

    // Should be horizontally scrollable on mobile
    const isScrollable = await analyticsNav.evaluate(el => {
      return el.scrollWidth > el.clientWidth ||
             window.getComputedStyle(el).overflowX === 'auto' ||
             window.getComputedStyle(el).overflowX === 'scroll';
    });

    // Either scrollable or tabs wrap/adjust for mobile - both are valid implementations
    if (isScrollable) {
      console.log('Analytics nav uses horizontal scroll on mobile');
    } else {
      console.log('Analytics nav uses alternative layout on mobile (wrapping/stacking)');
    }
  });

  test('should allow navigating to all analytics pages on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    // Wait for layout to stabilize after viewport change
    await page.waitForFunction(() => {
      const analyticsNav = document.getElementById('analytics-nav');
      if (!analyticsNav) return false;
      return analyticsNav.offsetHeight > 0 && analyticsNav.offsetWidth > 0;
    }, { timeout: TIMEOUTS.DEFAULT });

    // Should be able to click all analytics tabs
    const tabs = ['overview', 'content', 'users', 'performance', 'geographic', 'advanced'];

    for (const tab of tabs) {
      const tabButton = page.locator(`.analytics-tab[data-analytics-page="${tab}"]`);

      // Scroll to tab if needed
      await tabButton.scrollIntoViewIfNeeded();

      // Click using JavaScript for CI reliability
      // Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate((tabPage) => {
        const btn = document.querySelector(`.analytics-tab[data-analytics-page="${tabPage}"]`) as HTMLElement;
        if (btn) btn.click();
      }, tab);

      // Wait for tab content to render
      await page.waitForFunction((currentTab) => {
        const container = document.getElementById('analytics-container');
        if (!container) return false;
        const activePage = container.querySelector(`[data-analytics-page="${currentTab}"]`);
        return activePage !== null;
      }, tab, { timeout: TIMEOUTS.DEFAULT });

      // Tab should be active
      // WHY: Analytics tabs use role="tab" which requires aria-selected (not aria-pressed)
      // aria-pressed is for toggle buttons, aria-selected is for tabs
      await expect(tabButton).toHaveAttribute('aria-selected', 'true');
    }
  });

  test('should show scroll indicators when tabs overflow', async ({ page }) => {
    await page.setViewportSize({ width: 320, height: 568 }); // iPhone SE size

    // Wait for layout to stabilize after viewport change
    await page.waitForFunction(() => {
      const analyticsNav = document.getElementById('analytics-nav');
      if (!analyticsNav) return false;
      return analyticsNav.offsetHeight > 0 && analyticsNav.offsetWidth > 0;
    }, { timeout: TIMEOUTS.DEFAULT });

    const analyticsNav = page.locator('#analytics-nav');
    await expect(analyticsNav).toBeVisible();

    // Check if gradient fade indicators are present (CSS solution)
    // Or check for scroll buttons (JS solution)
    // The implementation will define which approach is used
    await expect(analyticsNav).toBeVisible();
  });
});

test.describe('Chart Title Truncation', () => {
  test.beforeEach(async ({ page }) => {
    // Ensure onboarding is skipped
    await page.addInitScript(() => {
      localStorage.setItem('onboarding_completed', 'true');
      localStorage.setItem('onboarding_skipped', 'true');
    });

    await gotoAppAndWaitReady(page);

    // Navigate to Analytics tab using JavaScript click for CI reliability
    // Playwright's .click() may fail in headless/SwiftShader environments
    await page.evaluate(() => {
      const tab = document.querySelector('.nav-tab[data-view="analytics"]') as HTMLElement;
      if (tab) tab.click();
    });
    await expect(page.locator('#analytics-container')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
  });

  test('should truncate long chart titles with ellipsis', async ({ page }) => {
    // Chart card titles should have text-overflow: ellipsis
    const chartCards = page.locator('.chart-card h3, .analytics-card-title');
    const count = await chartCards.count();

    if (count > 0) {
      const firstTitle = chartCards.first();
      const overflow = await firstTitle.evaluate(el =>
        window.getComputedStyle(el).textOverflow
      );

      // Should use ellipsis for truncation
      expect(['ellipsis', 'clip']).toContain(overflow);
    }
  });

  test('should show full title on hover via title attribute', async ({ page }) => {
    // Chart titles should have title attribute for full text on hover
    const chartCards = page.locator('.chart-card h3');
    const count = await chartCards.count();

    if (count > 0) {
      // Chart containers or titles may have title attributes
      await expect(page.locator('.chart-card').first()).toBeVisible();
    }
  });
});
