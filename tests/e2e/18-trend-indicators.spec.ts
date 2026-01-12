// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import { test, expect, TIMEOUTS, gotoAppAndWaitReady } from "./fixtures";

/**
 * E2E Test: Trend Indicators for Stats
 *
 * Tests the trend indicators on stat cards:
 * - Visual trend indicators (up/down arrows)
 * - Percentage change display
 * - Color coding (green for up, red for down)
 * - Accessible trend announcements
 *
 * Reference: UI/UX Audit
 * @see /docs/working/UI_UX_AUDIT.md
 */

test.describe("Trend Indicators", () => {
  test.beforeEach(async ({ page }) => {
    await gotoAppAndWaitReady(page);
    // Wait for stats to fully load (API call + render)
    await page.waitForSelector(".stat-value", {
      state: "visible",
      timeout: TIMEOUTS.MEDIUM,
    });
  });

  test("stat cards should have trend indicator elements", async ({
    page,
  }) => {
    // Wait for stats to load with longer timeout
    await page.waitForSelector(".stat-value", { timeout: TIMEOUTS.LONG });

    // Wait for trend indicators to render
    await page
      .waitForSelector('.stat-trend, [data-testid="stat-trend"]', {
        timeout: TIMEOUTS.MEDIUM,
      })
      .catch(() => {
        // Trend indicators may not be present if no comparison data exists
      });

    // Check for trend indicators on stat cards
    const trendIndicators = page.locator(
      '.stat-trend, [data-testid="stat-trend"]',
    );

    // Should have at least one trend indicator
    const count = await trendIndicators.count();
    expect(count).toBeGreaterThan(0);
  });

  test("trend indicator should show direction (up/down/neutral)", async ({
    page,
  }) => {
    await page.waitForSelector(".stat-value", { timeout: TIMEOUTS.LONG });

    const trendIndicator = page
      .locator('.stat-trend, [data-testid="stat-trend"]')
      .first();

    // Should have a direction class or data attribute
    const hasDirection = await trendIndicator.evaluate((el) => {
      return (
        el.classList.contains("trend-up") ||
        el.classList.contains("trend-down") ||
        el.classList.contains("trend-neutral") ||
        el.hasAttribute("data-direction")
      );
    });

    expect(hasDirection).toBe(true);
  });

  test("trend indicator should show percentage change", async ({
    page,
  }) => {
    await page.waitForSelector(".stat-value", { timeout: TIMEOUTS.MEDIUM });

    const trendText = page.locator(".stat-trend-value, .trend-percentage");

    // Should have percentage text or be empty for neutral
    const count = await trendText.count();
    if (count > 0) {
      const text = await trendText.first().textContent();
      // Text should contain a number or percentage symbol
      expect(text).toMatch(/[\d%+-]/);
    }
  });

  test("positive trend should use success color (green)", async ({
    page,
  }) => {
    await page.waitForSelector(".stat-value", { timeout: TIMEOUTS.MEDIUM });

    const upTrend = page
      .locator('.stat-trend.trend-up, .stat-trend[data-direction="up"]')
      .first();

    if ((await upTrend.count()) > 0) {
      const color = await upTrend.evaluate((el) => {
        return getComputedStyle(el).color;
      });

      // Should be green-ish (success color)
      // #10b981 = rgb(16, 185, 129)
      expect(color).toMatch(/rgb\(16,?\s*185,?\s*129\)|#10b981|green/i);
    }
  });

  test("negative trend should use error color (red)", async ({
    page,
  }) => {
    await page.waitForSelector(".stat-value", { timeout: TIMEOUTS.MEDIUM });

    const downTrend = page
      .locator('.stat-trend.trend-down, .stat-trend[data-direction="down"]')
      .first();

    if ((await downTrend.count()) > 0) {
      const color = await downTrend.evaluate((el) => {
        return getComputedStyle(el).color;
      });

      // Should be red-ish (error color)
      // #ef4444 = rgb(239, 68, 68)
      expect(color).toMatch(/rgb\(239,?\s*68,?\s*68\)|#ef4444|red/i);
    }
  });

  test("trend indicator should have accessible label", async ({
    page,
  }) => {
    await page.waitForSelector(".stat-value", { timeout: TIMEOUTS.MEDIUM });

    const trendIndicator = page
      .locator('.stat-trend, [data-testid="stat-trend"]')
      .first();

    if ((await trendIndicator.count()) > 0) {
      // Should have aria-label or title
      const ariaLabel = await trendIndicator.getAttribute("aria-label");
      const title = await trendIndicator.getAttribute("title");

      expect(ariaLabel || title).toBeTruthy();
    }
  });

  test("trend indicator should have arrow icon or symbol", async ({
    page,
  }) => {
    await page.waitForSelector(".stat-value", { timeout: TIMEOUTS.MEDIUM });

    const trendIndicator = page
      .locator('.stat-trend, [data-testid="stat-trend"]')
      .first();

    if ((await trendIndicator.count()) > 0) {
      const hasArrow = await trendIndicator.evaluate((el) => {
        const text = el.textContent || "";
        const hasArrowSymbol = /[↑↓▲▼⬆⬇+\-]/.test(text);
        const hasArrowIcon =
          el.querySelector(".trend-arrow, .trend-icon, svg") !== null;
        return hasArrowSymbol || hasArrowIcon;
      });

      expect(hasArrow).toBe(true);
    }
  });
});

test.describe("Trend Calculation", () => {
  test("trends should compare current period to previous", async ({
    page,
  }) => {
    await gotoAppAndWaitReady(page);
    await page.waitForSelector(".stat-value", { timeout: TIMEOUTS.MEDIUM });

    // The trend should have a data attribute or title indicating the comparison period
    const trendIndicator = page
      .locator('.stat-trend, [data-testid="stat-trend"]')
      .first();

    if ((await trendIndicator.count()) > 0) {
      const title = await trendIndicator.getAttribute("title");
      const dataCompare = await trendIndicator.getAttribute(
        "data-compare-period",
      );

      // Should indicate what period is being compared
      expect(title || dataCompare).toBeTruthy();
    }
  });
});
