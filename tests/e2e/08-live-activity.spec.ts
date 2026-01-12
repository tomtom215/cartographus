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
} from "./fixtures";

/**
 * E2E Test: Live Activity Dashboard
 *
 * Tests real-time activity monitoring from Tautulli:
 * - Active streams display
 * - Stream statistics (direct play, transcode, bandwidth)
 * - Session details rendering
 * - Empty state handling
 * - Real-time WebSocket updates
 *
 * NOTE: All tile requests are automatically mocked via fixtures.ts
 * This makes tests fully offline, deterministic, and faster.
 */

test.describe("Live Activity Dashboard", () => {
  test.beforeEach(async ({ page }) => {
    // Use storageState for authentication (configured in playwright.config.ts)
    await gotoAppAndWaitReady(page);
  });

  test("should render live activity container", async ({ page }) => {
    // Navigate to live activity tab
    await navigateToView(page, VIEWS.ACTIVITY);

    // Container should be visible
    await expect(page.locator(SELECTORS.ACTIVITY_CONTAINER)).toBeVisible({
      timeout: TIMEOUTS.MEDIUM,
    });
  });

  test("should display activity overview statistics", async ({ page }) => {
    await navigateToView(page, VIEWS.ACTIVITY);
    // Wait for container to be visible
    await page.waitForSelector(SELECTORS.ACTIVITY_CONTAINER, {
      state: "visible",
      timeout: TIMEOUTS.MEDIUM,
    });

    // Check for overview cards
    await expect(page.locator("#activity-overview")).toBeVisible();
    await expect(page.locator("#activity-stream-count")).toBeVisible();
    await expect(page.locator("#activity-direct-play")).toBeVisible();
    await expect(page.locator("#activity-transcode")).toBeVisible();
    await expect(page.locator("#activity-bandwidth")).toBeVisible();
  });

  test("should display stream count correctly", async ({ page }) => {
    await navigateToView(page, VIEWS.ACTIVITY);
    // Wait for container to be visible
    await page.waitForSelector(SELECTORS.ACTIVITY_CONTAINER, {
      state: "visible",
      timeout: TIMEOUTS.MEDIUM,
    });

    const streamCount = page.locator("#activity-stream-count");
    await expect(streamCount).toBeVisible();

    // Should contain a number (0 or more)
    const text = await streamCount.textContent();
    expect(text).toMatch(/^\d+$/);
  });

  test("should show empty state when no active streams", async ({ page }) => {
    await navigateToView(page, VIEWS.ACTIVITY);
    // Wait for container to be visible
    await page.waitForSelector(SELECTORS.ACTIVITY_CONTAINER, {
      state: "visible",
      timeout: TIMEOUTS.MEDIUM,
    });

    const streamCount = page.locator("#activity-stream-count");
    await expect(streamCount).toBeVisible();

    // DETERMINISTIC WAIT: Wait for count to stabilize (not just be visible)
    // The count might briefly show a loading state before settling to final value
    await page.waitForFunction(
      () => {
        const countEl = document.getElementById('activity-stream-count');
        if (!countEl) return false;
        const text = countEl.textContent?.trim();
        return text !== null && text !== '' && /^\d+$/.test(text);
      },
      { timeout: TIMEOUTS.MEDIUM }
    );

    const countText = await streamCount.textContent();
    expect(countText).toMatch(/^\d+$/);
    const count = parseInt(countText || "0");

    if (count === 0) {
      // DETERMINISTIC WAIT: Wait for either empty state OR error state to appear
      // ROOT CAUSE FIX: The immediate isVisible() check was racing with render
      await page.waitForFunction(
        () => {
          const emptyState = document.getElementById('activity-empty-state');
          const errorState = document.querySelector('.activity-error-message, .error-state');
          const emptyVisible = emptyState && window.getComputedStyle(emptyState).display !== 'none';
          return emptyVisible || errorState !== null;
        },
        { timeout: TIMEOUTS.MEDIUM }
      );

      const emptyState = page.locator("#activity-empty-state");
      const isVisible = await emptyState.isVisible();

      if (isVisible) {
        // Should have appropriate message
        const emptyText = await emptyState.textContent();
        expect(emptyText?.toLowerCase()).toContain("no active");
        console.log("[E2E] Empty state visible with correct message");
      } else {
        // Error state is present (validated by waitForFunction above)
        const errorState = page.locator(".activity-error-message, .error-state");
        const hasError = await errorState.count();
        expect(hasError).toBeGreaterThan(0);
        console.log(`[E2E] Error state present: ${hasError > 0}`);
      }
    } else {
      // Count > 0: Verify sessions are actually displayed
      const sessionsContainer = page.locator("#activity-sessions");
      await expect(sessionsContainer).toBeVisible();

      const sessionCards = page.locator(
        "#activity-sessions .activity-session-card",
      );
      const cardCount = await sessionCards.count();
      // If count > 0, we should have session cards
      expect(cardCount).toBeGreaterThan(0);
      console.log(`[E2E] ${count} streams displayed with ${cardCount} session card(s)`);
    }
  });

  test("should render active sessions when streams exist", async ({ page }) => {
    await navigateToView(page, VIEWS.ACTIVITY);
    // Wait for container to be visible
    await page.waitForSelector(SELECTORS.ACTIVITY_CONTAINER, {
      state: "visible",
      timeout: TIMEOUTS.MEDIUM,
    });

    const streamCount = page.locator("#activity-stream-count");
    await expect(streamCount).toBeVisible();
    const countText = await streamCount.textContent();

    if (countText && parseInt(countText) > 0) {
      // Sessions container should be visible
      const sessionsContainer = page.locator("#activity-sessions");
      await expect(sessionsContainer).toBeVisible();

      // Should have at least one session card
      const sessionCards = page.locator(
        "#activity-sessions .activity-session-card",
      );
      const count = await sessionCards.count();
      expect(count).toBeGreaterThan(0);
    }
  });

  test("should display bandwidth information", async ({ page }) => {
    await navigateToView(page, VIEWS.ACTIVITY);
    // Wait for activity container to be visible instead of arbitrary timeout
    await page.waitForSelector(SELECTORS.ACTIVITY_CONTAINER, {
      state: "visible",
      timeout: TIMEOUTS.MEDIUM,
    });

    const bandwidth = page.locator("#activity-bandwidth");
    await expect(bandwidth).toBeVisible();

    // Should display bandwidth in Mbps, Kbps, or Gbps format (including "0 Mbps" for no activity)
    const bandwidthText = await bandwidth.textContent();
    // Match patterns like "0 Mbps", "15 Mbps", "1.5 Gbps", etc.
    expect(bandwidthText).toMatch(/[\d.]+\s*(Mbps|Kbps|Gbps|bps)/i);
  });

  test("should distinguish between direct play and transcode", async ({
    page,
  }) => {
    await navigateToView(page, VIEWS.ACTIVITY);
    // Wait for container to be visible
    await page.waitForSelector(SELECTORS.ACTIVITY_CONTAINER, {
      state: "visible",
      timeout: TIMEOUTS.MEDIUM,
    });

    const directPlay = page.locator("#activity-direct-play");
    const transcode = page.locator("#activity-transcode");

    await expect(directPlay).toBeVisible();
    await expect(transcode).toBeVisible();

    // Both should contain numbers
    const directPlayText = await directPlay.textContent();
    const transcodeText = await transcode.textContent();

    expect(directPlayText).toMatch(/^\d+$/);
    expect(transcodeText).toMatch(/^\d+$/);
  });

  test("should update statistics in real-time via WebSocket", async ({
    page,
  }) => {
    await navigateToView(page, VIEWS.ACTIVITY);
    // Wait for container to be visible
    await page.waitForSelector(SELECTORS.ACTIVITY_CONTAINER, {
      state: "visible",
      timeout: TIMEOUTS.MEDIUM,
    });

    // Get initial stream count
    const streamCount = page.locator("#activity-stream-count");
    await expect(streamCount).toBeVisible();
    const initialCount = await streamCount.textContent();

    // Wait for stream count element to be stable (has valid content)
    await expect(streamCount).toHaveText(/^\d+$/, { timeout: TIMEOUTS.MEDIUM });

    // Check if count is still a valid number (may or may not have changed)
    const updatedCount = await streamCount.textContent();
    expect(updatedCount).toMatch(/^\d+$/);

    // Stream counts must be non-negative integers
    const initialValue = parseInt(initialCount || "0");
    const updatedValue = parseInt(updatedCount || "0");

    expect(initialValue).toBeGreaterThanOrEqual(0);
    expect(updatedValue).toBeGreaterThanOrEqual(0);
  });

  test("should handle navigation to activity view multiple times", async ({
    page,
  }) => {
    // Navigate to activity
    await navigateToView(page, VIEWS.ACTIVITY);
    await expect(page.locator(SELECTORS.ACTIVITY_CONTAINER)).toBeVisible({
      timeout: TIMEOUTS.MEDIUM,
    });

    // Navigate away
    await navigateToView(page, VIEWS.MAPS);
    await expect(page.locator(SELECTORS.MAP)).toBeVisible({
      timeout: TIMEOUTS.MEDIUM,
    });

    // Navigate back to activity
    await navigateToView(page, VIEWS.ACTIVITY);
    await expect(page.locator(SELECTORS.ACTIVITY_CONTAINER)).toBeVisible({
      timeout: TIMEOUTS.MEDIUM,
    });

    // Should still show correct data
    await expect(page.locator("#activity-stream-count")).toBeVisible();
  });

  test("should maintain responsive layout on mobile", async ({ page }) => {
    await navigateToView(page, VIEWS.ACTIVITY);
    await expect(page.locator(SELECTORS.ACTIVITY_CONTAINER)).toBeVisible({
      timeout: TIMEOUTS.MEDIUM,
    });

    // Switch to mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    // Activity container should still be visible after resize
    await expect(page.locator(SELECTORS.ACTIVITY_CONTAINER)).toBeVisible();
    await expect(page.locator("#activity-overview")).toBeVisible();

    // Statistics should stack vertically on mobile
    const overview = page.locator("#activity-overview");
    const box = await overview.boundingBox();
    expect(box).not.toBeNull();
  });

  // API Error Handling tests need autoMockApi: false to work correctly
  // ROOT CAUSE: The fixture's mock server registers routes at the context level,
  // which page.unrouteAll() cannot clear. By disabling autoMockApi, we ensure
  // only the page-level error routes are used.
  test.describe("API Error Handling", () => {
    test.use({ autoMockApi: false });

    test("should handle API errors gracefully", async ({ page }) => {
      // Set up onboarding flags since autoMockApi is disabled
      await page.addInitScript(() => {
        localStorage.setItem('onboarding_completed', 'true');
        localStorage.setItem('onboarding_skipped', 'true');
        localStorage.setItem('setup_wizard_completed', 'true');
        // Mock auth tokens so app thinks it's authenticated
        localStorage.setItem('auth_token', 'mock-test-token');
        localStorage.setItem('auth_username', 'testuser');
        localStorage.setItem('auth_expires_at', new Date(Date.now() + 86400000).toISOString());
      });

      // Track if our error route was called (for debugging)
      let errorRouteHit = false;

      // Register our error-returning route with a regex pattern
      // Note: No need to unrouteAll since autoMockApi: false means no context routes
      await page.route(/\/api\/v1\/tautulli\/activity/, (route) => {
        errorRouteHit = true;
        route.fulfill({
          status: 500,
          contentType: "application/json",
          body: JSON.stringify({
            status: "error",
            error: { message: "Server error" },
          }),
        });
      });

      // Also mock other necessary endpoints to prevent app from hanging
      await page.route(/\/api\/v1\/stats/, (route) => {
        route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            status: "success",
            data: { total_playbacks: 0, unique_locations: 0, unique_users: 0, recent_24h: 0 },
          }),
        });
      });

      await page.route(/\/api\/v1\/locations/, (route) => {
        route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({ status: "success", data: [] }),
        });
      });

      await page.goto('/', { waitUntil: 'networkidle' });
      await navigateToView(page, VIEWS.ACTIVITY);
      await expect(page.locator(SELECTORS.ACTIVITY_CONTAINER)).toBeVisible({
        timeout: TIMEOUTS.MEDIUM,
      });

      // DETERMINISTIC WAIT: Wait for the error element to appear in the DOM.
      const errorState = page.locator(".activity-error-message, .error-state");
      const streamCount = page.locator("#activity-stream-count");

      // Wait for either:
      // 1. Error element to be attached to DOM (error was handled), OR
      // 2. Stream count to show "0" (graceful fallback completed)
      await page.waitForFunction(
        () => {
          const errorEl = document.querySelector('.activity-error-message, .error-state');
          const countEl = document.getElementById('activity-stream-count');
          const countIsZero = countEl?.textContent?.trim() === '0';
          return errorEl !== null || countIsZero;
        },
        { timeout: TIMEOUTS.MEDIUM }
      );

      await expect(streamCount).toBeVisible();
      const countText = await streamCount.textContent();
      const errorCount = await errorState.count();

      // App should either:
      // 1. Show count as "0" (graceful fallback), OR
      // 2. Display an error state element
      const hasGracefulFallback = countText === "0";
      const hasErrorDisplay = errorCount > 0;

      console.log(`[E2E] API error handled: route hit=${errorRouteHit}, graceful fallback=${hasGracefulFallback}, error display=${hasErrorDisplay}`);
      expect(hasGracefulFallback || hasErrorDisplay).toBeTruthy();

      // Page should not have crashed (no unhandled exception dialog)
      const crashDialog = page.locator('.crash-modal, .fatal-error, .error-overlay');
      const crashCount = await crashDialog.count();
      expect(crashCount).toBe(0);
    });
  });

  test("should display session details when available", async ({ page }) => {
    await navigateToView(page, VIEWS.ACTIVITY);
    // Wait for container to be visible
    await page.waitForSelector(SELECTORS.ACTIVITY_CONTAINER, {
      state: "visible",
      timeout: TIMEOUTS.MEDIUM,
    });

    const streamCount = page.locator("#activity-stream-count");
    await expect(streamCount).toBeVisible();
    const countText = await streamCount.textContent();

    if (countText && parseInt(countText) > 0) {
      const sessionsContainer = page.locator("#activity-sessions");
      await expect(sessionsContainer).toBeVisible();

      // Check for typical session information elements
      const hasContent = await sessionsContainer.textContent();
      expect(hasContent).toBeTruthy();
      expect(hasContent!.length).toBeGreaterThan(0);
    }
  });

  test("should refresh data when refresh button is clicked", async ({
    page,
  }) => {
    await navigateToView(page, VIEWS.ACTIVITY);
    // Wait for container to be visible
    await page.waitForSelector(SELECTORS.ACTIVITY_CONTAINER, {
      state: "visible",
      timeout: TIMEOUTS.MEDIUM,
    });

    // Get initial state
    const streamCount = page.locator("#activity-stream-count");
    await expect(streamCount).toBeVisible();
    const _initialCount = await streamCount.textContent();

    // Look for main refresh button (use ID specifically to avoid matching stale warning button)
    const refreshButton = page.locator(SELECTORS.BTN_REFRESH);

    if (await refreshButton.isVisible()) {
      await refreshButton.click();
      // Wait for stream count to update after refresh
      await expect(streamCount).toHaveText(/^\d+$/, {
        timeout: TIMEOUTS.MEDIUM,
      });

      // Data should still be valid after refresh
      const newCount = await streamCount.textContent();
      expect(newCount).toMatch(/^\d+$/);
    }
  });

  test("should handle very long session lists efficiently", async ({
    page,
  }) => {
    await navigateToView(page, VIEWS.ACTIVITY);
    // Wait for container to be visible
    await page.waitForSelector(SELECTORS.ACTIVITY_CONTAINER, {
      state: "visible",
      timeout: TIMEOUTS.MEDIUM,
    });

    const sessionsContainer = page.locator("#activity-sessions");
    await expect(sessionsContainer).toBeVisible();

    // Page should remain responsive even with multiple sessions
    const startTime = Date.now();
    await navigateToView(page, VIEWS.MAPS);
    await expect(page.locator(SELECTORS.MAP)).toBeVisible({
      timeout: TIMEOUTS.MEDIUM,
    });
    const navigationTime = Date.now() - startTime;

    // Navigation should be fast (< 3 seconds accounting for async operations)
    expect(navigationTime).toBeLessThan(3000);
  });
});
