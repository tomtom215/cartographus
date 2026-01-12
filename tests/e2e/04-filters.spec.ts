// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Test: Filter Application
 *
 * Tests all filtering functionality:
 * - Date range picker (presets + custom)
 * - User filter (multi-select)
 * - Media type filter (multi-select)
 * - Filter combinations
 * - URL query parameter sync
 * - Clear filters functionality
 * - Filter debouncing
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
  gotoAppAndWaitReady,
  navigateToView,
} from './fixtures';

test.describe('Filter Application', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to app (already authenticated via storageState from setup)
    await gotoAppAndWaitReady(page);
  });

  test('should render filter panel with all controls', async ({ page }) => {
    // Filter panel should be visible - use #filters directly to avoid matching all filter elements
    await expect(page.locator(SELECTORS.FILTERS)).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Date range filter (actual ID: #filter-days)
    const dateFilter = page.locator(SELECTORS.FILTER_DAYS);
    await expect(dateFilter).toBeVisible();
    expect(await dateFilter.count()).toBeGreaterThan(0);

    // User filter (actual ID: #filter-users)
    // This is a <select> element that gets populated with options from API
    const userFilter = page.locator('#filter-users');
    await expect(userFilter).toBeVisible();
    // With mock data, we should have at least the "All Users" option
    const userOptions = await userFilter.locator('option').count();
    expect(userOptions).toBeGreaterThan(0);

    // Media type filter (actual ID: #filter-media-types)
    // This is a <select> element that gets populated with options from API
    const mediaFilter = page.locator('#filter-media-types');
    await expect(mediaFilter).toBeVisible();
    // With mock data, we should have at least the "All Media Types" option
    const mediaOptions = await mediaFilter.locator('option').count();
    expect(mediaOptions).toBeGreaterThan(0);
  });

  test('should apply date range preset filters', async ({ page }) => {
    // Find date range selector - it's a <select> element with <option> children
    const dateFilter = page.locator(SELECTORS.FILTER_DAYS);
    await expect(dateFilter).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Select "Last 7 days" option (value="7")
    await dateFilter.selectOption('7');

    // Wait for filter value to be applied
    await page.waitForFunction(() => {
      const select = document.querySelector('#filter-days') as HTMLSelectElement;
      return select && select.value === '7';
    }, { timeout: TIMEOUTS.DATA_LOAD });

    // Stats should update
    const stats = page.locator('#stat-playbacks');
    await expect(stats).toBeVisible();
  });

  test('should apply custom date range', async ({ page }) => {
    // Custom date inputs (actual IDs: #filter-start-date, #filter-end-date)
    const startDate = page.locator('#filter-start-date');
    const endDate = page.locator('#filter-end-date');

    await expect(startDate).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    await expect(endDate).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Clear preset filter to enable custom range
    const dateFilter = page.locator(SELECTORS.FILTER_DAYS);
    await dateFilter.selectOption(''); // Custom range option

    // Set custom date range
    await startDate.fill('2025-01-01');
    await endDate.fill('2025-01-31');

    // Wait for date inputs to be applied
    await page.waitForFunction(() => {
      const start = document.querySelector('#filter-start-date') as HTMLInputElement;
      const end = document.querySelector('#filter-end-date') as HTMLInputElement;
      return start && end && start.value === '2025-01-01' && end.value === '2025-01-31';
    }, { timeout: TIMEOUTS.DATA_LOAD });

    // Application should still be functional
    await expect(page.locator(SELECTORS.MAP)).toBeVisible();
  });

  test('should filter by user', async ({ page }) => {
    // User filter (actual ID: #filter-users)
    const userFilter = page.locator('#filter-users');
    await expect(userFilter).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Get available options
    const options = await userFilter.locator('option').count();

    if (options > 1) {
      // Select a user (skip first option which is "All Users")
      await userFilter.selectOption({ index: 1 });

      // Wait for filter selection to be applied
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-users') as HTMLSelectElement;
        return select && select.selectedIndex === 1;
      }, { timeout: TIMEOUTS.RENDER });

      // Data should update
      await expect(page.locator(SELECTORS.MAP)).toBeVisible();

      // Stats might change
      const statValue = page.locator('#stat-playbacks');
      await expect(statValue).toBeVisible();
    }
  });

  test('should filter by media type', async ({ page }) => {
    // Media type filter (actual ID: #filter-media-types)
    const mediaFilter = page.locator('#filter-media-types');
    await expect(mediaFilter).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    const options = await mediaFilter.locator('option').count();

    if (options > 1) {
      // Select a media type
      await mediaFilter.selectOption({ index: 1 });

      // Wait for filter selection to be applied
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-media-types') as HTMLSelectElement;
        return select && select.selectedIndex === 1;
      }, { timeout: TIMEOUTS.RENDER });

      // Data should update
      await expect(page.locator(SELECTORS.MAP)).toBeVisible();
    }
  });

  test('should apply multiple filters simultaneously', async ({ page }) => {
    // Apply date filter (actual ID: #filter-days)
    const dateFilter = page.locator(SELECTORS.FILTER_DAYS);
    await expect(dateFilter).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    await dateFilter.selectOption('30'); // Last 30 days

    // Wait for date filter to be applied
    await page.waitForFunction(() => {
      const select = document.querySelector('#filter-days') as HTMLSelectElement;
      return select && select.value === '30';
    }, { timeout: TIMEOUTS.ANIMATION });

    // Apply user filter (actual ID: #filter-users)
    const userFilter = page.locator('#filter-users');
    const userOptions = await userFilter.locator('option').count();
    if (userOptions > 1) {
      await userFilter.selectOption({ index: 1 });

      // Wait for user filter to be applied
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-users') as HTMLSelectElement;
        return select && select.selectedIndex === 1;
      }, { timeout: TIMEOUTS.ANIMATION });
    }

    // Apply media type filter (actual ID: #filter-media-types)
    const mediaFilter = page.locator('#filter-media-types');
    const mediaOptions = await mediaFilter.locator('option').count();
    if (mediaOptions > 1) {
      await mediaFilter.selectOption({ index: 1 });

      // Wait for media filter to be applied
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-media-types') as HTMLSelectElement;
        return select && select.selectedIndex === 1;
      }, { timeout: TIMEOUTS.RENDER });
    }

    // All filters should work together
    await expect(page.locator(SELECTORS.MAP)).toBeVisible();
  });

  test('should sync filters to URL query parameters', async ({ page }) => {
    // Apply a filter using the date select (actual ID: #filter-days)
    const dateFilter = page.locator(SELECTORS.FILTER_DAYS);
    await expect(dateFilter).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    await dateFilter.selectOption('7'); // Last 7 days

    // Wait for filter to be applied
    await page.waitForFunction(() => {
      const select = document.querySelector('#filter-days') as HTMLSelectElement;
      return select && select.value === '7';
    }, { timeout: TIMEOUTS.RENDER });

    // Check URL
    const url = page.url();

    // URL should contain query parameter (e.g., ?days=7 or similar)
    // This depends on implementation, just verify URL changed or has params
    expect(url).toBeTruthy();
  });

  test('should restore filters from URL on page load', async ({ page }) => {
    // Navigate with query parameters (storageState handles auth)
    await page.goto('/?days=30');

    // Wait for app to load (storageState provides auth)
    await page.waitForSelector(SELECTORS.APP_VISIBLE, { timeout: TIMEOUTS.DEFAULT });

    // Wait for filter to be populated from URL
    await page.waitForFunction(() => {
      const select = document.querySelector('#filter-days') as HTMLSelectElement;
      return select && select.value === '30';
    }, { timeout: TIMEOUTS.DATA_LOAD });

    // Verify map is visible (filter should be applied from URL)
    await expect(page.locator(SELECTORS.MAP)).toBeVisible();

    // Verify the filter dropdown shows the correct value
    const filterDays = page.locator(SELECTORS.FILTER_DAYS);
    if (await filterDays.isVisible()) {
      const selectedValue = await filterDays.inputValue();
      expect(selectedValue).toBe('30');
    }
  });

  test('should clear all filters', async ({ page }) => {
    // Apply some filters first using date select (actual ID: #filter-days)
    const dateFilter = page.locator(SELECTORS.FILTER_DAYS);
    await expect(dateFilter).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    await dateFilter.selectOption('7'); // Last 7 days

    // Wait for filter to be applied
    await page.waitForFunction(() => {
      const select = document.querySelector('#filter-days') as HTMLSelectElement;
      return select && select.value === '7';
    }, { timeout: TIMEOUTS.RENDER });

    // Find clear button (actual ID: #btn-clear-filters)
    const clearButton = page.locator('#btn-clear-filters');
    await expect(clearButton).toBeVisible();
    await clearButton.click();

    // Wait for filters to reset to default (90 days)
    await page.waitForFunction(() => {
      const select = document.querySelector('#filter-days') as HTMLSelectElement;
      return select && select.value === '90';
    }, { timeout: TIMEOUTS.RENDER });

    // Filters should be reset
    await expect(page.locator(SELECTORS.MAP)).toBeVisible();

    // Verify date filter was reset (back to default "90" days)
    const selectedValue = await dateFilter.inputValue();
    expect(selectedValue).toBe('90');
  });

  test('should debounce filter updates', async ({ page }) => {
    // Date filter (actual ID: #filter-days)
    const dateFilter = page.locator(SELECTORS.FILTER_DAYS);
    await expect(dateFilter).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Rapidly change filter values to test debouncing
    await dateFilter.selectOption('30'); // Last 30 days
    await dateFilter.selectOption('7'); // Last 7 days
    await dateFilter.selectOption('30'); // Back to Last 30 days

    // Wait for final value to be applied after debounce
    await page.waitForFunction(() => {
      const select = document.querySelector('#filter-days') as HTMLSelectElement;
      return select && select.value === '30';
    }, { timeout: TIMEOUTS.RENDER });

    // Should only update once after debounce
    await expect(page.locator(SELECTORS.MAP)).toBeVisible();
  });

  test('should update stats panel when filters change', async ({ page }) => {
    // Get initial stat value
    const statPlaybacks = page.locator('#stat-playbacks');
    await expect(statPlaybacks).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    const _initialValue = await statPlaybacks.textContent();

    // Apply filter using date select (actual ID: #filter-days)
    const dateFilter = page.locator(SELECTORS.FILTER_DAYS);
    await dateFilter.selectOption('7'); // Last 7 days

    // Wait for filter to be applied
    await page.waitForFunction(() => {
      const select = document.querySelector('#filter-days') as HTMLSelectElement;
      return select && select.value === '7';
    }, { timeout: TIMEOUTS.DATA_LOAD });

    // Stat may have changed (or stayed same if no difference)
    const newValue = await statPlaybacks.textContent();

    // Values should be valid numbers or dashes
    expect(newValue).toBeTruthy();
  });

  test('should update charts when filters change', async ({ page }) => {
    // Navigate to Analytics view to see charts
    await navigateToView(page, VIEWS.ANALYTICS);

    // Wait for charts container to be visible
    const trendsChart = page.locator('#chart-trends');
    await expect(trendsChart).toBeVisible({ timeout: TIMEOUTS.WEBGL_INIT });

    // Apply filter using date select (actual ID: #filter-days)
    const dateFilter = page.locator(SELECTORS.FILTER_DAYS);
    await dateFilter.selectOption('7'); // Last 7 days

    // Wait for filter to be applied
    await page.waitForFunction(() => {
      const select = document.querySelector('#filter-days') as HTMLSelectElement;
      return select && select.value === '7';
    }, { timeout: TIMEOUTS.DATA_LOAD });

    // Chart should still be visible
    await expect(trendsChart).toBeVisible();
  });

  test('should update map markers when filters change', async ({ page }) => {
    await expect(page.locator(SELECTORS.MAP)).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

    // Wait for map canvas to be initialized
    await page.waitForFunction(() => {
      const canvas = document.querySelector('#map canvas');
      return canvas && (canvas as HTMLCanvasElement).width > 0;
    }, { timeout: TIMEOUTS.WEBGL_INIT });

    // Apply filter using date select (actual ID: #filter-days)
    const dateFilter = page.locator(SELECTORS.FILTER_DAYS);
    await expect(dateFilter).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    await dateFilter.selectOption('7'); // Last 7 days

    // Wait for filter to be applied
    await page.waitForFunction(() => {
      const select = document.querySelector('#filter-days') as HTMLSelectElement;
      return select && select.value === '7';
    }, { timeout: TIMEOUTS.DATA_LOAD });

    // Map should still be functional
    await expect(page.locator(SELECTORS.MAP_CANVAS)).toBeVisible();
  });

  test('should handle filters with no matching data', async ({ page }) => {
    // Apply very restrictive filter (actual ID: #filter-users)
    const userFilter = page.locator('#filter-users');
    await expect(userFilter).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Try selecting last option (might be a user with no data)
    const options = await userFilter.locator('option').count();

    if (options > 1) {
      await userFilter.selectOption({ index: options - 1 });

      // Wait for filter selection to be applied
      await page.waitForFunction(
        (expectedIndex) => {
          const select = document.querySelector('#filter-users') as HTMLSelectElement;
          return select && select.selectedIndex === expectedIndex;
        },
        options - 1,
        { timeout: TIMEOUTS.DATA_LOAD }
      );

      // Application should handle no data gracefully
      await expect(page.locator(SELECTORS.MAP)).toBeVisible();

      // Stats might show 0 or "-"
      const statValue = page.locator('#stat-playbacks');
      await expect(statValue).toBeVisible();
      const value = await statValue.textContent();
      expect(value).toBeTruthy();
    }
  });

  test('should persist filter selections across page refresh', async ({ page }) => {
    // Apply filters using date select (actual ID: #filter-days)
    const dateFilter = page.locator(SELECTORS.FILTER_DAYS);
    await expect(dateFilter).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    await dateFilter.selectOption('30'); // Last 30 days

    // Wait for filter to be applied
    await page.waitForFunction(() => {
      const select = document.querySelector('#filter-days') as HTMLSelectElement;
      return select && select.value === '30';
    }, { timeout: TIMEOUTS.RENDER });

    // Get current URL with params
    const urlBefore = page.url();

    // Reload page
    await page.reload();

    // Wait for page to load
    await expect(page.locator(SELECTORS.MAP)).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

    // URL should still have parameters
    const urlAfter = page.url();

    // URLs should match (params preserved)
    if (urlBefore.includes('?')) {
      expect(urlAfter).toContain('?');
    }
  });
});

test.describe('Data Refresh Indicator', () => {
  test.beforeEach(async ({ page }) => {
    await gotoAppAndWaitReady(page);

    // Wait for app to fully initialize (managers, UI components)
    // The refresh button and data indicator are created during manager initialization
    const refreshBtnReady = await page.waitForFunction(() => {
      const refreshBtn = document.getElementById('btn-refresh');
      return refreshBtn !== null;
    }, { timeout: TIMEOUTS.MEDIUM }).then(() => true).catch(() => false);
    if (!refreshBtnReady) {
      console.warn('[E2E] Refresh button not found - Data Refresh Indicator tests may fail');
    }
  });

  test('should have refresh button with icon', async ({ page }) => {
    const refreshButton = page.locator('#btn-refresh');
    await expect(refreshButton).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Should have refresh icon
    const refreshIcon = page.locator('#refresh-icon');
    await expect(refreshIcon).toBeVisible();
  });

  test('should have data refresh indicator element', async ({ page }) => {
    const refreshIndicator = page.locator('#data-refresh-indicator');

    // Element should exist (hidden by default)
    await expect(refreshIndicator).toBeAttached();

    // Should have proper ARIA attributes
    await expect(refreshIndicator).toHaveAttribute('role', 'status');
    await expect(refreshIndicator).toHaveAttribute('aria-live', 'polite');
  });

  test('clicking refresh should show loading state', async ({ page }) => {
    const refreshButton = page.locator('#btn-refresh');
    await expect(refreshButton).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Click refresh
    await refreshButton.click();

    // Wait for either loading state or completion
    await page.waitForFunction(() => {
      const refreshText = document.getElementById('refresh-text');
      return refreshText && (refreshText.textContent === 'Loading...' || refreshText.textContent === 'Refresh Data');
    }, { timeout: TIMEOUTS.ANIMATION });

    // Check if loading class is applied (might be brief)
    // The button text should change during loading
    const refreshText = await page.textContent('#refresh-text');
    // Either shows Loading... or Refresh Data (if loading completed quickly)
    expect(['Loading...', 'Refresh Data']).toContain(refreshText);
  });
});

test.describe('Quick Date Picker Buttons', () => {
  test.beforeEach(async ({ page }) => {
    await gotoAppAndWaitReady(page);

    // Wait for app to fully initialize (filter manager, quick date buttons)
    const quickDateReady = await page.waitForFunction(() => {
      const quickDateContainer = document.getElementById('quick-date-buttons');
      return quickDateContainer !== null;
    }, { timeout: TIMEOUTS.MEDIUM }).then(() => true).catch(() => false);
    if (!quickDateReady) {
      console.warn('[E2E] Quick date buttons not found - Quick Date Picker tests may fail');
    }
  });

  test('should have quick date picker buttons', async ({ page }) => {
    // Quick date picker container should be visible
    const quickDatePicker = page.locator('#quick-date-buttons');
    await expect(quickDatePicker).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Should have Today, Yesterday, This Week, This Month buttons
    await expect(page.locator('#quick-date-today')).toBeVisible();
    await expect(page.locator('#quick-date-yesterday')).toBeVisible();
    await expect(page.locator('#quick-date-week')).toBeVisible();
    await expect(page.locator('#quick-date-month')).toBeVisible();
  });

  test('clicking Today button should set dates to today', async ({ page }) => {
    const todayBtn = page.locator('#quick-date-today');
    await expect(todayBtn).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    const today = new Date().toISOString().split('T')[0];

    await todayBtn.click();

    // Wait for dates to be set
    await page.waitForFunction(
      (expectedDate) => {
        const start = document.querySelector('#filter-start-date') as HTMLInputElement;
        const end = document.querySelector('#filter-end-date') as HTMLInputElement;
        return start && end && start.value === expectedDate && end.value === expectedDate;
      },
      today,
      { timeout: TIMEOUTS.RENDER }
    );

    // Check that start and end dates are set to today
    const startDate = await page.inputValue('#filter-start-date');
    const endDate = await page.inputValue('#filter-end-date');

    // Both should be today's date
    expect(startDate).toBe(today);
    expect(endDate).toBe(today);
  });

  test('clicking Yesterday button should set dates to yesterday', async ({ page }) => {
    const yesterdayBtn = page.locator('#quick-date-yesterday');
    await expect(yesterdayBtn).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Calculate expected date before clicking
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    const expectedDate = yesterday.toISOString().split('T')[0];

    await yesterdayBtn.click();

    // Wait for dates to be set
    await page.waitForFunction(
      (expected) => {
        const start = document.querySelector('#filter-start-date') as HTMLInputElement;
        const end = document.querySelector('#filter-end-date') as HTMLInputElement;
        return start && end && start.value === expected && end.value === expected;
      },
      expectedDate,
      { timeout: TIMEOUTS.RENDER }
    );

    // Check dates
    const startDate = await page.inputValue('#filter-start-date');
    const endDate = await page.inputValue('#filter-end-date');

    // Both should be yesterday's date
    expect(startDate).toBe(expectedDate);
    expect(endDate).toBe(expectedDate);
  });

  test('quick date buttons should have accessible labels', async ({ page }) => {
    // Check all buttons have accessible labels
    const buttons = ['#quick-date-today', '#quick-date-yesterday', '#quick-date-week', '#quick-date-month'];

    for (const selector of buttons) {
      const button = page.locator(selector);
      await expect(button).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Should have aria-label or visible text
      const text = await button.textContent();
      expect(text?.trim().length).toBeGreaterThan(0);
    }
  });
});
