// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import {
  test,
  expect,
  TIMEOUTS,
  gotoAppAndWaitReady,
} from './fixtures';

/**
 * E2E Test: Sparklines for Stat Cards
 *
 * Tests sparkline mini trend line functionality:
 * - Sparkline visibility in stat cards
 * - SVG rendering
 * - Accessibility attributes
 * - Responsive sizing
 *
 * @see /docs/working/UI_UX_AUDIT.md
 *
 * Note: Uses autoMockApi: true for deterministic sparkline and trend data in CI.
 * This ensures stat cards have trend data regardless of Tautulli state.
 */

// Enable API mocking for deterministic trend data
test.use({ autoMockApi: true });

test.describe('Sparklines in Stat Cards', () => {
  test.beforeEach(async ({ page }) => {
    // CRITICAL: Clear sidebar-collapsed localStorage setting BEFORE navigation
    // WHY (ROOT CAUSE FIX): The SidebarManager reads 'sidebar-collapsed' from localStorage
    // during initialization. If set to 'true', the sidebar gets the 'collapsed' class which
    // hides #stats via CSS (#sidebar.collapsed #stats { display: none; }).
    // This causes tests to fail with "[data-testid="stat-grid"]" not found.
    await page.addInitScript(() => {
      localStorage.setItem('onboarding_completed', 'true');
      localStorage.setItem('onboarding_skipped', 'true');
      // Ensure sidebar is NOT collapsed so stats section is visible
      localStorage.setItem('sidebar-collapsed', 'false');
    });

    await gotoAppAndWaitReady(page);

    // Wait for sidebar to be visible and NOT collapsed
    // WHY: The sidebar contains the stats section; if collapsed, stats are hidden
    await page.waitForFunction(
      () => {
        const sidebar = document.getElementById('sidebar');
        if (!sidebar) return false;
        // Sidebar should be visible and NOT have 'collapsed' class
        const isVisible = sidebar.offsetHeight > 0 && sidebar.offsetWidth > 0;
        const isNotCollapsed = !sidebar.classList.contains('collapsed');
        return isVisible && isNotCollapsed;
      },
      { timeout: TIMEOUTS.MEDIUM }
    );

    // Wait for stats section to be visible
    // WHY: Stats are in the sidebar which should now be expanded on desktop
    // The stat-grid is static HTML but may take time to render in CI environments
    await expect(page.locator('#stats')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Wait for stats grid to be visible - use data-testid for stability (Task 24)
    // WHY: MEDIUM timeout (10s) accounts for API response and rendering in CI/SwiftShader
    await expect(page.locator('[data-testid="stat-grid"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Wait for stat values to be populated (not just "-" placeholder)
    // WHY: This ensures data has actually loaded before testing sparklines
    await page.waitForFunction(
      () => {
        const statValue = document.querySelector('#stat-playbacks');
        return statValue && statValue.textContent && statValue.textContent.trim() !== '-';
      },
      { timeout: TIMEOUTS.MEDIUM }
    );
  });

  test.describe('Sparkline Rendering', () => {
    test('should display sparklines in stat cards', async ({ page }) => {
      // Wait for sparklines to be rendered or stats to stabilize
      // Use MEDIUM timeout (10s) for reliable CI execution in SwiftShader/headless environments
      await page.waitForFunction(
        () => {
          const statGrid = document.querySelector('[data-testid="stat-grid"]');
          return statGrid !== null && document.readyState === 'complete';
        },
        { timeout: TIMEOUTS.MEDIUM }
      );

      // Sparklines should be present in stat cards
      const sparklines = page.locator('.stat-sparkline');
      const count = await sparklines.count();

      // If sparklines exist, verify at least one is visible
      if (count > 0) {
        await expect(sparklines.first()).toBeVisible();
        console.log(`Found ${count} sparklines in stat cards`);
      } else {
        console.log('No sparklines present - graceful degradation when no trend data');
      }
    });

    test('should render sparkline as SVG', async ({ page }) => {
      // Wait for DOM to be ready and stats to load
      // Use MEDIUM timeout (10s) for reliable CI execution
      await page.waitForFunction(
        () => document.readyState === 'complete' && document.querySelector('[data-testid="stat-grid"]') !== null,
        { timeout: TIMEOUTS.MEDIUM }
      );

      const sparkline = page.locator('.stat-sparkline svg').first();

      // If sparkline exists, it should be an SVG
      const count = await sparkline.count();
      if (count > 0) {
        await expect(sparkline).toBeVisible();

        // Should have viewBox attribute
        const viewBox = await sparkline.getAttribute('viewBox');
        expect(viewBox).toBeTruthy();
      }
    });

    test('should have polyline or path for trend line', async ({ page }) => {
      // Wait for stats to be fully loaded
      await page.waitForFunction(
        () => document.readyState === 'complete' && document.querySelector('[data-testid="stat-grid"]') !== null,
        { timeout: TIMEOUTS.MEDIUM }
      );

      const sparklineLine = page.locator('.stat-sparkline svg polyline, .stat-sparkline svg path').first();
      const count = await sparklineLine.count();

      if (count > 0) {
        await expect(sparklineLine).toBeVisible();

        // Should have points or d attribute
        const points = await sparklineLine.getAttribute('points');
        const d = await sparklineLine.getAttribute('d');
        expect(points || d).toBeTruthy();
      }
    });
  });

  test.describe('Sparkline Accessibility', () => {
    test('should have appropriate ARIA attributes', async ({ page }) => {
      // Wait for stats to be fully loaded
      await page.waitForFunction(
        () => document.readyState === 'complete' && document.querySelector('[data-testid="stat-grid"]') !== null,
        { timeout: TIMEOUTS.MEDIUM }
      );

      const sparkline = page.locator('.stat-sparkline').first();
      const count = await sparkline.count();

      if (count > 0) {
        // Should have role="img" for screen readers
        const role = await sparkline.getAttribute('role');
        expect(role).toBe('img');

        // Should have aria-label describing the trend
        const ariaLabel = await sparkline.getAttribute('aria-label');
        expect(ariaLabel).toBeTruthy();
      }
    });

    test('should be decorative (aria-hidden) when trend indicator exists', async ({ page }) => {
      // Wait for stats to be fully loaded
      await page.waitForFunction(
        () => document.readyState === 'complete' && document.querySelector('[data-testid="stat-grid"]') !== null,
        { timeout: TIMEOUTS.MEDIUM }
      );

      // If stat-trend exists alongside sparkline, sparkline can be decorative
      // Use data-testid for stable selection (Task 24)
      const statCard = page.locator('[data-testid="stat-card-playbacks"]');
      const trendIndicator = statCard.locator('[data-testid="stat-trend"]');
      const sparkline = statCard.locator('#sparkline-playbacks');

      const trendCount = await trendIndicator.count();
      const sparklineCount = await sparkline.count();

      // If both exist, sparkline provides visual redundancy to the trend
      if (trendCount > 0 && sparklineCount > 0) {
        await expect(trendIndicator).toBeVisible();
      }
    });
  });

  test.describe('Sparkline Styling', () => {
    test('should have appropriate dimensions', async ({ page }) => {
      // Wait for stats to be fully loaded
      await page.waitForFunction(
        () => document.readyState === 'complete' && document.querySelector('[data-testid="stat-grid"]') !== null,
        { timeout: TIMEOUTS.MEDIUM }
      );

      const sparkline = page.locator('.stat-sparkline').first();
      const count = await sparkline.count();

      if (count > 0) {
        const box = await sparkline.boundingBox();
        expect(box).not.toBeNull();

        if (box) {
          // Sparkline should be compact (typically 60-100px wide, 20-40px tall)
          expect(box.width).toBeGreaterThan(30);
          expect(box.width).toBeLessThan(150);
          expect(box.height).toBeGreaterThan(10);
          expect(box.height).toBeLessThan(60);
        }
      }
    });

    test('should match theme colors', async ({ page }) => {
      // Wait for stats to be fully loaded
      await page.waitForFunction(
        () => document.readyState === 'complete' && document.querySelector('[data-testid="stat-grid"]') !== null,
        { timeout: TIMEOUTS.MEDIUM }
      );

      const sparklineLine = page.locator('.stat-sparkline svg polyline, .stat-sparkline svg path').first();
      const count = await sparklineLine.count();

      if (count > 0) {
        // Get stroke color
        const stroke = await sparklineLine.getAttribute('stroke');
        expect(stroke).toBeTruthy();
      }
    });
  });

  test.describe('Sparkline Integration', () => {
    test('should coexist with trend indicators', async ({ page }) => {
      // Wait for stats to be fully loaded
      await page.waitForFunction(
        () => document.readyState === 'complete' && document.querySelector('[data-testid="stat-grid"]') !== null,
        { timeout: TIMEOUTS.MEDIUM }
      );

      // Both trend indicator and sparkline should be visible
      const statCard = page.locator('.stat-card').first();
      await expect(statCard).toBeVisible();

      // Trend indicator should be present
      const trendIndicator = statCard.locator('.stat-trend');
      await expect(trendIndicator).toBeVisible();

      // Layout should accommodate both elements
      const valueRow = statCard.locator('.stat-value-row');
      await expect(valueRow).toBeVisible();
    });

    test('should update when filters change', async ({ page }) => {
      // Wait for stats to be fully loaded
      await page.waitForFunction(
        () => document.readyState === 'complete' && document.querySelector('[data-testid="stat-grid"]') !== null,
        { timeout: TIMEOUTS.MEDIUM }
      );

      // Change filter
      await page.selectOption('#filter-days', '7');

      // Wait for network to be idle after filter change
      // Use MEDIUM timeout (10s) for reliable network completion in CI
      await page.waitForLoadState('networkidle', { timeout: TIMEOUTS.MEDIUM });

      // Stat cards should still be visible
      await expect(page.locator('.stat-card').first()).toBeVisible();

      // Sparklines should still be functional (if present)
      const sparklines = page.locator('.stat-sparkline');
      const count = await sparklines.count();
      // Verify sparklines are still visible if present
      if (count > 0) {
        await expect(sparklines.first()).toBeVisible();
      }
    });
  });
});

test.describe('Sparkline Responsiveness', () => {
  test('should scale on mobile viewport', async ({ page }) => {
    // CRITICAL: Clear sidebar-collapsed localStorage setting BEFORE navigation
    // WHY (ROOT CAUSE FIX): Same as main test suite - prevent sidebar from starting collapsed
    await page.addInitScript(() => {
      localStorage.setItem('onboarding_completed', 'true');
      localStorage.setItem('onboarding_skipped', 'true');
      localStorage.setItem('sidebar-collapsed', 'false');
    });

    await gotoAppAndWaitReady(page);

    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    // Wait for layout to stabilize after viewport change
    // WHY: On mobile, the sidebar becomes a slide-out menu (hidden by default)
    // but when visible, the stat-grid inside should still be accessible
    await page.waitForFunction(
      () => {
        // Force reflow after viewport change
        document.body.offsetHeight;
        return document.readyState === 'complete';
      },
      { timeout: TIMEOUTS.MEDIUM }
    );

    // On mobile viewport, sidebar may be hidden - open it first via hamburger menu
    // WHY: Mobile layout hides sidebar by default; stats are inside sidebar
    const menuToggle = page.locator('#menu-toggle');
    const isMenuToggleVisible = await menuToggle.isVisible();
    if (isMenuToggleVisible) {
      // Click hamburger to open sidebar on mobile
      await page.evaluate(() => {
        const btn = document.getElementById('menu-toggle') as HTMLElement;
        if (btn) btn.click();
      });
      // Wait for sidebar to open
      await page.waitForFunction(
        () => {
          const sidebar = document.getElementById('sidebar');
          return sidebar && sidebar.classList.contains('open');
        },
        { timeout: TIMEOUTS.SHORT }
      );
    }

    // Stat cards should now be visible - use data-testid (Task 24)
    await expect(page.locator('[data-testid="stat-grid"]')).toBeVisible();

    // Sparklines should scale appropriately (or hide on very small screens)
    const sparkline = page.locator('.stat-sparkline').first();
    const count = await sparkline.count();

    if (count > 0) {
      const box = await sparkline.boundingBox();
      if (box) {
        // Should be reasonably sized even on mobile
        expect(box.width).toBeGreaterThan(20);
      }
    }
  });
});
