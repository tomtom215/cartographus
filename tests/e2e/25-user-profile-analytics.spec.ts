// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Tests for User Profile Analytics Dashboard
 * TDD: These tests are written first, implementation follows
 */

import {
  TIMEOUTS,
  test,
  expect,
  gotoAppAndWaitReady,
  waitForNavReady,
} from "./fixtures";

test.describe("User Profile Analytics Dashboard", () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to app
    await gotoAppAndWaitReady(page);
  });

  test.describe("Analytics User Tab Navigation", () => {
    test("should have User tab in analytics navigation", async ({
      page,
    }) => {
      // Navigate to analytics section
      const analyticsTab = page.locator('.nav-tab[data-view="analytics"]');
      await expect(analyticsTab).toBeVisible();
      await analyticsTab.click();

      // Wait for analytics section
      await page.waitForSelector("#analytics-container", {
        state: "visible",
        timeout: TIMEOUTS.MEDIUM,
      });

      // Check for User tab in analytics sub-navigation
      const userTab = page.locator(
        '.analytics-tab[data-analytics-page="users-profile"]',
      );
      await expect(userTab).toBeVisible();
      await expect(userTab).toHaveText(/User Profile/i);
    });

    test("should switch to User Profile page when tab clicked", async ({
      page,
    }) => {
      // Navigate to analytics section
      await page.locator('.nav-tab[data-view="analytics"]').click();
      await page.waitForSelector("#analytics-container", {
        state: "visible",
        timeout: TIMEOUTS.MEDIUM,
      });

      // Click User Profile tab
      const userTab = page.locator(
        '.analytics-tab[data-analytics-page="users-profile"]',
      );
      await userTab.click();

      // Verify User Profile page is visible
      const userProfilePage = page.locator("#analytics-users-profile");
      await expect(userProfilePage).toBeVisible();
    });
  });

  test.describe("User Selector", () => {
    test("should display user selector dropdown", async ({ page }) => {
      // Navigate to User Profile page
      await page.locator('.nav-tab[data-view="analytics"]').click();
      await page.waitForSelector("#analytics-container", { state: "visible" });
      await page
        .locator('.analytics-tab[data-analytics-page="users-profile"]')
        .click();
      await page.waitForSelector("#analytics-users-profile", {
        state: "visible",
      });

      // Check for user selector
      const userSelector = page.locator("#user-profile-selector");
      await expect(userSelector).toBeVisible();
      await expect(userSelector).toHaveAttribute("aria-label");
    });

    test("should show empty state when no user selected", async ({
      page,
    }) => {
      // Navigate to User Profile page
      await page.locator('.nav-tab[data-view="analytics"]').click();
      await page.waitForSelector("#analytics-container", { state: "visible" });
      await page
        .locator('.analytics-tab[data-analytics-page="users-profile"]')
        .click();
      await page.waitForSelector("#analytics-users-profile", {
        state: "visible",
      });

      // Check for empty state message
      const emptyState = page.locator("#user-profile-empty-state");
      await expect(emptyState).toBeVisible();
      await expect(emptyState).toContainText(/select a user/i);
    });
  });

  test.describe("User Profile Display", () => {
    test("should have user overview stats section", async ({
      page,
    }) => {
      // Navigate to User Profile page
      await page.locator('.nav-tab[data-view="analytics"]').click();
      await page.waitForSelector("#analytics-container", { state: "visible" });
      await page
        .locator('.analytics-tab[data-analytics-page="users-profile"]')
        .click();
      await page.waitForSelector("#analytics-users-profile", {
        state: "visible",
      });

      // Check for user stats section (hidden initially)
      const statsSection = page.locator("#user-profile-stats");
      await expect(statsSection).toBeAttached();
    });

    test("should have user activity charts section", async ({
      page,
    }) => {
      // Navigate to User Profile page
      await page.locator('.nav-tab[data-view="analytics"]').click();
      await page.waitForSelector("#analytics-container", { state: "visible" });
      await page
        .locator('.analytics-tab[data-analytics-page="users-profile"]')
        .click();
      await page.waitForSelector("#analytics-users-profile", {
        state: "visible",
      });

      // Check for chart containers (hidden initially)
      const trendChart = page.locator("#chart-user-activity-trend");
      const contentChart = page.locator("#chart-user-top-content");
      const platformChart = page.locator("#chart-user-platforms");

      await expect(trendChart).toBeAttached();
      await expect(contentChart).toBeAttached();
      await expect(platformChart).toBeAttached();
    });
  });

  test.describe("Accessibility", () => {
    test("should have proper ARIA labels", async ({ page }) => {
      // Navigate to User Profile page
      await page.locator('.nav-tab[data-view="analytics"]').click();
      await page.waitForSelector("#analytics-container", { state: "visible" });
      await page
        .locator('.analytics-tab[data-analytics-page="users-profile"]')
        .click();
      await page.waitForSelector("#analytics-users-profile", {
        state: "visible",
      });

      // Check selector accessibility
      const selector = page.locator("#user-profile-selector");
      await expect(selector).toHaveAttribute("aria-label");

      // Check page has proper landmark
      // Note: role="tabpanel" is correct for tab content (not region)
      const profilePage = page.locator("#analytics-users-profile");
      await expect(profilePage).toHaveAttribute("role", "tabpanel");
    });

    test("should support keyboard navigation", async ({ page }) => {
      // Navigate to User Profile page
      await page.locator('.nav-tab[data-view="analytics"]').click();
      await page.waitForSelector("#analytics-container", { state: "visible" });
      await page
        .locator('.analytics-tab[data-analytics-page="users-profile"]')
        .click();
      await page.waitForSelector("#analytics-users-profile", {
        state: "visible",
      });

      // Tab to selector
      const selector = page.locator("#user-profile-selector");
      await selector.focus();
      await expect(selector).toBeFocused();
    });
  });

  test.describe("Chart Integration", () => {
    test("should have chart containers with proper IDs", async ({
      page,
    }) => {
      // Navigate to User Profile page
      await page.locator('.nav-tab[data-view="analytics"]').click();
      await page.waitForSelector("#analytics-container", { state: "visible" });
      await page
        .locator('.analytics-tab[data-analytics-page="users-profile"]')
        .click();
      await page.waitForSelector("#analytics-users-profile", {
        state: "visible",
      });

      // Verify chart container IDs exist
      const chartIds = [
        "chart-user-activity-trend",
        "chart-user-top-content",
        "chart-user-platforms",
      ];

      for (const chartId of chartIds) {
        const chart = page.locator(`#${chartId}`);
        await expect(chart).toBeAttached();
        // Charts should have chart-container class
        await expect(chart).toHaveClass(/chart-container/);
      }
    });
  });

  test.describe("URL Hash Navigation", () => {
    // QUARANTINED: Test fails with "Target page, context or browser has been closed"
    // Root cause: Browser context closed during assertion - likely suite timeout killed the browser
    // The test tries to assert visibility after page.goto() but context may be closed
    // Tracking: CI failure analysis from 2025-12-31
    // Unquarantine criteria: 20 consecutive green runs after fixing suite timeout
    test.fixme("should support deep linking to users-profile page", async ({
      page,
    }) => {
      // Navigate directly via URL hash
      await page.goto("/#analytics-users-profile");
      await page.waitForSelector("#app:not(.hidden)", {
        state: "visible",
        timeout: TIMEOUTS.DEFAULT,
      });
      // Wait for NavigationManager to initialize
      await waitForNavReady(page);

      // Should navigate to analytics then to users-profile
      const analyticsContainer = page.locator("#analytics-container");
      await expect(analyticsContainer).toBeVisible({
        timeout: TIMEOUTS.MEDIUM,
      });

      const userProfilePage = page.locator("#analytics-users-profile");
      await expect(userProfilePage).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });
  });
});
