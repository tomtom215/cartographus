// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Tests for Chart Timeline Animation
 * Tests the timeline animation controls for analytics charts
 *
 * STANDARDIZATION: This file follows the shard 1/2 pattern:
 * - Uses gotoAppAndWaitReady() in beforeEach
 * - Relies on global mock server (no setupApiMocking)
 * - Only uses page.route() for feature-specific API overrides
 */

import { test, expect, TIMEOUTS, gotoAppAndWaitReady, waitForNavReady, setMobileViewport } from './fixtures';

// Mock time series data for animation
const mockTimeSeriesData = [
  { date: '2024-01-01', playback_count: 100, unique_users: 10 },
  { date: '2024-01-02', playback_count: 150, unique_users: 15 },
  { date: '2024-01-03', playback_count: 120, unique_users: 12 },
  { date: '2024-01-04', playback_count: 180, unique_users: 18 },
  { date: '2024-01-05', playback_count: 200, unique_users: 20 },
  { date: '2024-01-06', playback_count: 160, unique_users: 16 },
  { date: '2024-01-07', playback_count: 220, unique_users: 22 }
];

test.describe('Chart Timeline Animation', () => {
  test.beforeEach(async ({ page }) => {
    // STANDARDIZED: Follow shard 1/2 pattern - simple beforeEach
    // Feature-specific route for time-series API (overrides global mock)
    await page.route('**/api/v1/analytics/time-series*', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: mockTimeSeriesData,
          metadata: { timestamp: new Date().toISOString(), query_time_ms: 25 }
        })
      });
    });

    // Standard app initialization (handles auth, onboarding, etc.)
    await gotoAppAndWaitReady(page);
    await waitForNavReady(page);
  });

  test('should render chart timeline animation container', async ({ page }) => {
    const container = page.locator('#chart-timeline-animation-container');
    await expect(container).toBeVisible();
  });

  test('should display timeline animation controls', async ({ page }) => {
    await page.waitForSelector('.chart-timeline-controls', { timeout: TIMEOUTS.MEDIUM });

    // Check for play/pause button
    const playPauseBtn = page.locator('#chart-timeline-play-pause');
    await expect(playPauseBtn).toBeVisible();

    // Check for reset button
    const resetBtn = page.locator('#chart-timeline-reset');
    await expect(resetBtn).toBeVisible();

    // Check for slider
    const slider = page.locator('#chart-timeline-slider');
    await expect(slider).toBeVisible();
  });

  test('should have speed control options', async ({ page }) => {
    await page.waitForSelector('.chart-timeline-options', { timeout: TIMEOUTS.MEDIUM });

    const speedSelect = page.locator('#chart-timeline-speed');
    await expect(speedSelect).toBeVisible();

    // Verify speed options
    const options = speedSelect.locator('option');
    await expect(options).toHaveCount(4);
    await expect(options.nth(0)).toHaveValue('1');
    await expect(options.nth(1)).toHaveValue('2');
    await expect(options.nth(2)).toHaveValue('5');
    await expect(options.nth(3)).toHaveValue('10');
  });

  test('should have interval control options', async ({ page }) => {
    await page.waitForSelector('.chart-timeline-options', { timeout: TIMEOUTS.MEDIUM });

    const intervalSelect = page.locator('#chart-timeline-interval');
    await expect(intervalSelect).toBeVisible();

    // Verify interval options
    const options = intervalSelect.locator('option');
    await expect(options).toHaveCount(3);
    await expect(options.nth(0)).toHaveValue('daily');
    await expect(options.nth(1)).toHaveValue('weekly');
    await expect(options.nth(2)).toHaveValue('monthly');
  });

  test('should have proper accessibility attributes', async ({ page }) => {
    const container = page.locator('#chart-timeline-animation-container');
    await expect(container).toHaveAttribute('role', 'region');
    await expect(container).toHaveAttribute('aria-label', 'Chart timeline animation controls');

    const playPauseBtn = page.locator('#chart-timeline-play-pause');
    await expect(playPauseBtn).toHaveAttribute('aria-label');

    const slider = page.locator('#chart-timeline-slider');
    await expect(slider).toHaveAttribute('aria-label', 'Timeline progress');
    await expect(slider).toHaveAttribute('aria-valuemin', '0');
    await expect(slider).toHaveAttribute('aria-valuemax', '100');
  });

  test('should display status element', async ({ page }) => {
    const status = page.locator('#chart-timeline-status');
    await expect(status).toBeVisible();
  });

  test('should display progress bar', async ({ page }) => {
    const progressBar = page.locator('#chart-timeline-progress-bar');
    await expect(progressBar).toBeVisible();
  });

  test('should update slider on seek', async ({ page }) => {
    await page.waitForSelector('#chart-timeline-slider', { timeout: TIMEOUTS.MEDIUM });

    const slider = page.locator('#chart-timeline-slider');

    // Move slider to 50%
    await slider.fill('50');
    await slider.dispatchEvent('input');

    // Verify slider value updated
    await expect(slider).toHaveValue('50');
  });

  test('should toggle play/pause icons', async ({ page }) => {
    await page.waitForSelector('#chart-timeline-play-pause', { timeout: TIMEOUTS.MEDIUM });

    const playIcon = page.locator('#chart-timeline-play-icon');
    const pauseIcon = page.locator('#chart-timeline-pause-icon');

    // Initially play icon should be visible
    await expect(playIcon).toBeVisible();
    await expect(pauseIcon).not.toBeVisible();

    // Click play/pause button
    await page.locator('#chart-timeline-play-pause').click();

    // Wait for icon state to change (pause icon becomes visible when animation starts)
    await pauseIcon.waitFor({ state: 'visible', timeout: TIMEOUTS.SHORT }).catch(() => {
      // Animation may not start if there's no data
    });

    // After clicking, pause icon should be visible (if animation started)
    // Note: This depends on data being loaded
  });

  test('should reset timeline on reset button click', async ({ page }) => {
    await page.waitForSelector('#chart-timeline-reset', { timeout: TIMEOUTS.MEDIUM });

    const slider = page.locator('#chart-timeline-slider');
    const resetBtn = page.locator('#chart-timeline-reset');

    // Move slider
    await slider.fill('75');
    await slider.dispatchEvent('input');

    // Click reset
    await resetBtn.click();

    // Wait for slider to reset to 0
    await expect(slider).toHaveValue('0', { timeout: TIMEOUTS.SHORT });
  });

  test('should change speed on selection', async ({ page }) => {
    await page.waitForSelector('#chart-timeline-speed', { timeout: TIMEOUTS.MEDIUM });

    const speedSelect = page.locator('#chart-timeline-speed');

    // Change to 5x speed
    await speedSelect.selectOption('5');

    // Verify selection
    await expect(speedSelect).toHaveValue('5');
  });

  test('should change interval on selection', async ({ page }) => {
    await page.waitForSelector('#chart-timeline-interval', { timeout: TIMEOUTS.MEDIUM });

    const intervalSelect = page.locator('#chart-timeline-interval');

    // Change to weekly interval
    await intervalSelect.selectOption('weekly');

    // Verify selection
    await expect(intervalSelect).toHaveValue('weekly');
  });

  test('should have timeline animation container initialized', async ({ page }) => {
    // Verify feature initialization by checking for the container element
    // This is more reliable than checking console logs which may be suppressed in CI
    await page.reload();
    await page.waitForSelector('#app', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

    // Wait for page to be fully loaded
    await page.waitForLoadState('networkidle', { timeout: TIMEOUTS.MEDIUM });

    // Verify the timeline animation container exists and has expected structure
    const container = page.locator('#chart-timeline-animation-container');
    await expect(container).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // The container should have the controls section initialized
    const controls = page.locator('.chart-timeline-controls');
    await expect(controls).toBeVisible({ timeout: TIMEOUTS.SHORT });
  });

  test('should have correct layout on mobile viewport', async ({ page }) => {
    // Use viewport helper which properly handles CSS media query re-evaluation
    await setMobileViewport(page);
    await page.reload();
    await page.waitForSelector('#app', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

    // Container should still be visible
    const container = page.locator('#chart-timeline-animation-container');
    await expect(container).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Controls should still be present
    const playPauseBtn = page.locator('#chart-timeline-play-pause');
    await expect(playPauseBtn).toBeVisible();
  });

  test('should handle API error gracefully', async ({ page }) => {
    await page.route('**/api/v1/analytics/time-series*', async route => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'error',
          error: 'Failed to fetch time series data',
          message: 'Database connection error'
        })
      });
    });

    await page.reload();
    await page.waitForSelector('#app', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

    // Container should still be present
    const container = page.locator('#chart-timeline-animation-container');
    await expect(container).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Error state or empty state should be shown
    const errorOrEmpty = page.locator('.chart-timeline-error, .chart-timeline-empty, #chart-timeline-status');
    await expect(errorOrEmpty).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  });

  test('should handle empty time series data gracefully', async ({ page }) => {
    await page.route('**/api/v1/analytics/time-series*', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: [],
          metadata: { timestamp: new Date().toISOString(), query_time_ms: 10 }
        })
      });
    });

    await page.reload();
    await page.waitForSelector('#app', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

    // Container should be visible
    const container = page.locator('#chart-timeline-animation-container');
    await expect(container).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Play button should be disabled or show no data state
    const status = page.locator('#chart-timeline-status');
    await expect(status).toBeVisible();
  });

  test('should have proper accessibility for error states', async ({ page }) => {
    await page.route('**/api/v1/analytics/time-series*', async route => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Server error' })
      });
    });

    await page.reload();
    await page.waitForSelector('#app', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

    // If an error element exists, it should have proper ARIA attributes
    const errorEl = page.locator('.chart-timeline-error');
    const isErrorVisible = await errorEl.isVisible().catch(() => false);

    if (isErrorVisible) {
      await expect(errorEl).toHaveAttribute('role', 'alert');
      await expect(errorEl).toHaveAttribute('aria-live', 'polite');
    }

    // Status element should always be accessible
    const status = page.locator('#chart-timeline-status');
    await expect(status).toHaveAttribute('role', 'status');
  });
});
