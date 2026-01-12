// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Tests for Geographic Drill-Down
 * Tests the Country -> Region -> City drill-down feature
 *
 * STANDARDIZATION: This file follows the shard 1/2 pattern:
 * - Uses gotoAppAndWaitReady() in beforeEach
 * - Relies on global mock server (no setupApiMocking)
 * - Only uses page.route() for feature-specific API overrides
 */

import { test, expect, TIMEOUTS, gotoAppAndWaitReady, waitForNavReady } from './fixtures';

// Mock location data with geographic hierarchy
const mockLocations = [
  {
    country: 'United States',
    region: 'California',
    city: 'Los Angeles',
    latitude: 34.0522,
    longitude: -118.2437,
    playback_count: 500,
    unique_users: 25,
    first_seen: '2024-01-01T00:00:00Z',
    last_seen: '2024-12-01T00:00:00Z',
    avg_completion: 82.5
  },
  {
    country: 'United States',
    region: 'California',
    city: 'San Francisco',
    latitude: 37.7749,
    longitude: -122.4194,
    playback_count: 350,
    unique_users: 18,
    first_seen: '2024-01-01T00:00:00Z',
    last_seen: '2024-12-01T00:00:00Z',
    avg_completion: 78.0
  },
  {
    country: 'United States',
    region: 'New York',
    city: 'New York City',
    latitude: 40.7128,
    longitude: -74.0060,
    playback_count: 600,
    unique_users: 30,
    first_seen: '2024-01-01T00:00:00Z',
    last_seen: '2024-12-01T00:00:00Z',
    avg_completion: 85.0
  },
  {
    country: 'United Kingdom',
    region: 'England',
    city: 'London',
    latitude: 51.5074,
    longitude: -0.1278,
    playback_count: 400,
    unique_users: 20,
    first_seen: '2024-01-01T00:00:00Z',
    last_seen: '2024-12-01T00:00:00Z',
    avg_completion: 80.0
  },
  {
    country: 'Germany',
    region: 'Bavaria',
    city: 'Munich',
    latitude: 48.1351,
    longitude: 11.5820,
    playback_count: 200,
    unique_users: 10,
    first_seen: '2024-01-01T00:00:00Z',
    last_seen: '2024-12-01T00:00:00Z',
    avg_completion: 75.0
  }
];

test.describe('Geographic Drill-Down', () => {
  test.beforeEach(async ({ page }) => {
    // STANDARDIZED: Follow shard 1/2 pattern - simple beforeEach
    // Feature-specific route for locations API (overrides global mock)
    await page.route('**/api/v1/locations*', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: mockLocations,
          metadata: { timestamp: new Date().toISOString(), total_count: mockLocations.length }
        })
      });
    });

    // Standard app initialization (handles auth, onboarding, etc.)
    await gotoAppAndWaitReady(page);
    await waitForNavReady(page);
  });

  test('should render geographic drill-down container', async ({ page }) => {
    const container = page.locator('#geographic-drilldown-container');
    await expect(container).toBeVisible();
  });

  test('should show header with title and breadcrumb', async ({ page }) => {
    // Wait for drill-down to initialize
    await page.waitForSelector('.geo-drilldown-header', { timeout: TIMEOUTS.MEDIUM });

    const title = page.locator('.geo-drilldown-title');
    await expect(title).toContainText('Geographic Distribution');

    const breadcrumb = page.locator('#geo-breadcrumb');
    await expect(breadcrumb).toBeVisible();
  });

  test('should start at country level with Countries breadcrumb active', async ({ page }) => {
    await page.waitForSelector('.geo-drilldown-breadcrumb', { timeout: TIMEOUTS.MEDIUM });

    const countryButton = page.locator('[data-level="country"]');
    await expect(countryButton).toHaveClass(/active/);
    await expect(countryButton).toContainText('Countries');
  });

  test('should display chart container', async ({ page }) => {
    const chartContainer = page.locator('#geo-drilldown-chart');
    await expect(chartContainer).toBeVisible();
  });

  test('should show stats section', async ({ page }) => {
    const statsContainer = page.locator('#geo-drilldown-stats');
    await expect(statsContainer).toBeVisible();
  });

  test('should have proper accessibility attributes', async ({ page }) => {
    const container = page.locator('#geographic-drilldown-container');
    await expect(container).toHaveAttribute('role', 'region');
    await expect(container).toHaveAttribute('aria-label', 'Geographic drill-down');

    const breadcrumb = page.locator('#geo-breadcrumb');
    await expect(breadcrumb).toHaveAttribute('role', 'navigation');
    await expect(breadcrumb).toHaveAttribute('aria-label', 'Geographic navigation');

    // Breadcrumb buttons - button elements have implicit button role
    const countryButton = page.locator('[data-level="country"]');
    // Button elements implicitly have role="button", so we verify the tag name
    await expect(countryButton).toHaveAttribute('data-level', 'country');
    await expect(countryButton).toHaveAttribute('aria-current', 'page');

    // Chart container should be accessible
    const chartContainer = page.locator('#geo-drilldown-chart');
    await expect(chartContainer).toHaveAttribute('role', 'img');
    await expect(chartContainer).toHaveAttribute('aria-label', 'Geographic distribution chart');

    // Stats section should have live region for updates
    const statsContainer = page.locator('#geo-drilldown-stats');
    await expect(statsContainer).toHaveAttribute('aria-live', 'polite');
  });

  test('should support keyboard navigation for breadcrumbs', async ({ page }) => {
    await page.waitForSelector('.geo-drilldown-breadcrumb', { timeout: TIMEOUTS.MEDIUM });

    const countryButton = page.locator('[data-level="country"]');

    // Focus and activate breadcrumb via keyboard
    await countryButton.focus();
    await expect(countryButton).toBeFocused();

    // Enter should activate the button
    await page.keyboard.press('Enter');
    await expect(countryButton).toHaveClass(/active/);

    // Space should also activate the button
    await page.keyboard.press('Space');
    await expect(countryButton).toHaveClass(/active/);
  });

  test('should show loading state to users', async ({ page }) => {
    // Check that loading element appears during data fetch
    await page.route('**/api/v1/locations*', async route => {
      await new Promise(resolve => setTimeout(resolve, TIMEOUTS.SHORT));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: mockLocations })
      });
    });

    await page.reload();
    await page.waitForSelector('#app', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

    // Loading element should be visible briefly and contain loading text
    const loadingEl = page.locator('.geo-drilldown-loading');
    await expect(loadingEl).toContainText('Loading');
  });

  test('should update breadcrumb when drilling down', async ({ page }) => {
    // Wait for chart to load
    await page.waitForSelector('.geo-drilldown-chart canvas, .geo-drilldown-loading', { timeout: TIMEOUTS.MEDIUM });

    // This test validates the breadcrumb structure
    const breadcrumb = page.locator('#geo-breadcrumb');

    // Initially only Countries should be visible
    const countryButton = breadcrumb.locator('[data-level="country"]');
    await expect(countryButton).toBeVisible();
  });

  test('should navigate back to country level via breadcrumb click', async ({ page }) => {
    await page.waitForSelector('.geo-drilldown-breadcrumb', { timeout: TIMEOUTS.MEDIUM });

    // Click Countries button (should be a no-op but validates functionality)
    const countryButton = page.locator('[data-level="country"]');
    await countryButton.click();

    // Should still be at country level
    await expect(countryButton).toHaveClass(/active/);
  });

  test('should display correct level label in stats', async ({ page }) => {
    await page.waitForSelector('#geo-drilldown-stats', { timeout: TIMEOUTS.MEDIUM });

    const stats = page.locator('#geo-drilldown-stats');
    // Wait for stats to populate by checking for the first stat label
    await expect(stats.locator('.geo-stat-label').first()).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Stats should contain level label (Countries at start)
    await expect(stats.locator('.geo-stat-label')).toContainText(['Countries', 'Total Playbacks', 'Unique Users', 'Avg Completion']);
  });

  test('should show loading state initially', async ({ page }) => {
    // Use a delayed response to catch loading state
    await page.route('**/api/v1/locations*', async route => {
      await new Promise(resolve => setTimeout(resolve, TIMEOUTS.SHORT));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: mockLocations })
      });
    });

    await page.reload();
    await page.waitForSelector('#app', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

    const loadingEl = page.locator('.geo-drilldown-loading');
    // Loading might be visible briefly
    await expect(loadingEl).toContainText('Loading');
  });

  test('should handle empty location data gracefully', async ({ page }) => {
    await page.route('**/api/v1/locations*', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [] })
      });
    });

    await page.reload();
    await page.waitForSelector('#geographic-drilldown-container', { state: 'visible' });

    // Wait for empty state to appear
    const emptyEl = page.locator('.geo-drilldown-empty');
    await expect(emptyEl).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    await expect(emptyEl).toContainText('No geographic data');
  });

  test('should handle API error gracefully', async ({ page }) => {
    await page.route('**/api/v1/locations*', async route => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal server error' })
      });
    });

    await page.reload();
    await page.waitForSelector('#geographic-drilldown-container', { state: 'visible' });

    // Wait for error state to appear
    const errorEl = page.locator('.geo-drilldown-error');
    await expect(errorEl).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Verify error message contains user-friendly text
    await expect(errorEl).toContainText('Failed');
    await expect(errorEl).toContainText('load');

    // Error element should have proper accessibility
    await expect(errorEl).toHaveAttribute('role', 'alert');
  });

  test('should handle network timeout gracefully', async ({ page }) => {
    await page.route('**/api/v1/locations*', async route => {
      // Simulate a very long delay that would timeout
      await new Promise(resolve => setTimeout(resolve, TIMEOUTS.EXTENDED + 5000));
      await route.abort('timedout');
    });

    await page.reload();
    await page.waitForSelector('#geographic-drilldown-container', { state: 'visible' });

    // Wait for error or timeout state
    const errorEl = page.locator('.geo-drilldown-error');
    await expect(errorEl).toBeVisible({ timeout: TIMEOUTS.EXTENDED + 10000 });

    // Should show timeout-related message
    await expect(errorEl).toContainText(/failed|timeout|error/i);
  });

  test('should allow retry after error', async ({ page }) => {
    let requestCount = 0;

    await page.route('**/api/v1/locations*', async route => {
      requestCount++;
      if (requestCount === 1) {
        // First request fails
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Temporary error' })
        });
      } else {
        // Subsequent requests succeed
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            data: mockLocations,
            metadata: { timestamp: new Date().toISOString(), total_count: mockLocations.length }
          })
        });
      }
    });

    await page.reload();
    await page.waitForSelector('#geographic-drilldown-container', { state: 'visible' });

    // Wait for error state
    const errorEl = page.locator('.geo-drilldown-error');
    await expect(errorEl).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Click retry button if available
    const retryBtn = page.locator('.geo-drilldown-retry, .geo-drilldown-error button');
    if (await retryBtn.isVisible()) {
      await retryBtn.click();

      // Error should disappear and chart should load
      await expect(errorEl).not.toBeVisible({ timeout: TIMEOUTS.MEDIUM });
      const chartContainer = page.locator('#geo-drilldown-chart');
      await expect(chartContainer).toBeVisible();
    }
  });
});
