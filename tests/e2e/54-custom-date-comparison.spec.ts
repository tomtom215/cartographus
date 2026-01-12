// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Tests for Custom Date Comparison
 * Tests the custom date range comparison feature
 *
 * STANDARDIZATION: This file follows the shard 1/2 pattern:
 * - Uses gotoAppAndWaitReady() in beforeEach
 * - Relies on global mock server (no setupApiMocking)
 * - Only uses page.route() for feature-specific API overrides
 */

import { test, expect, TIMEOUTS, gotoAppAndWaitReady, waitForNavReady } from './fixtures';

// Mock comparative analytics response
const mockComparativeData = {
  current_period: {
    playback_count: 1250,
    unique_users: 45,
    watch_time_minutes: 18000,
    avg_completion: 78.5,
    unique_content: 120,
    unique_locations: 35,
    avg_session_mins: 45.2
  },
  previous_period: {
    playback_count: 1000,
    unique_users: 38,
    watch_time_minutes: 15000,
    avg_completion: 72.0,
    unique_content: 95,
    unique_locations: 28,
    avg_session_mins: 42.5
  },
  comparison_type: 'custom',
  metrics_comparison: [
    {
      metric: 'playback_count',
      current_value: 1250,
      previous_value: 1000,
      absolute_change: 250,
      percentage_change: 25.0,
      growth_direction: 'up',
      is_improvement: true
    },
    {
      metric: 'unique_users',
      current_value: 45,
      previous_value: 38,
      absolute_change: 7,
      percentage_change: 18.4,
      growth_direction: 'up',
      is_improvement: true
    },
    {
      metric: 'watch_time_minutes',
      current_value: 18000,
      previous_value: 15000,
      absolute_change: 3000,
      percentage_change: 20.0,
      growth_direction: 'up',
      is_improvement: true
    },
    {
      metric: 'avg_completion',
      current_value: 78.5,
      previous_value: 72.0,
      absolute_change: 6.5,
      percentage_change: 9.0,
      growth_direction: 'up',
      is_improvement: true
    }
  ],
  top_content_comparison: [],
  top_user_comparison: [],
  overall_trend: 'growing',
  key_insights: ['Playback activity increased by 25%', 'User engagement improved']
};

test.describe('Custom Date Comparison', () => {
  test.beforeEach(async ({ page }) => {
    // STANDARDIZED: Follow shard 1/2 pattern - simple beforeEach
    // Feature-specific route for comparative analytics API (overrides global mock)
    await page.route('**/api/v1/analytics/comparative*', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: mockComparativeData,
          metadata: { timestamp: new Date().toISOString(), query_time_ms: 30 }
        })
      });
    });

    // Standard app initialization (handles auth, onboarding, etc.)
    await gotoAppAndWaitReady(page);
    await waitForNavReady(page);
  });

  test('should have Custom Range option in period selector', async ({ page }) => {
    const periodSelect = page.locator('#comparison-period-select');
    await expect(periodSelect).toBeVisible();

    // Check custom option exists
    const customOption = periodSelect.locator('option[value="custom"]');
    await expect(customOption).toHaveText('Custom Range');
  });

  test('should show date pickers when Custom Range is selected', async ({ page }) => {
    const periodSelect = page.locator('#comparison-period-select');
    const customDateContainer = page.locator('#custom-date-container');

    // Initially hidden
    await expect(customDateContainer).toHaveClass(/hidden/);

    // Select custom
    await periodSelect.selectOption('custom');

    // Should now be visible
    await expect(customDateContainer).not.toHaveClass(/hidden/);
    await expect(page.locator('#comparison-start-date')).toBeVisible();
    await expect(page.locator('#comparison-end-date')).toBeVisible();
    await expect(page.locator('#apply-custom-comparison')).toBeVisible();
  });

  test('should hide date pickers when switching away from Custom Range', async ({ page }) => {
    const periodSelect = page.locator('#comparison-period-select');
    const customDateContainer = page.locator('#custom-date-container');

    // Select custom first
    await periodSelect.selectOption('custom');
    await expect(customDateContainer).not.toHaveClass(/hidden/);

    // Switch to week
    await periodSelect.selectOption('week');
    await expect(customDateContainer).toHaveClass(/hidden/);
  });

  test('should have default dates set (last 30 days)', async ({ page }) => {
    const periodSelect = page.locator('#comparison-period-select');
    await periodSelect.selectOption('custom');

    const startInput = page.locator('#comparison-start-date');
    const endInput = page.locator('#comparison-end-date');

    // Check that dates are set
    const startValue = await startInput.inputValue();
    const endValue = await endInput.inputValue();

    expect(startValue).toBeTruthy();
    expect(endValue).toBeTruthy();

    // End date should be recent (today or yesterday, accounting for timezone differences)
    const today = new Date();
    const yesterday = new Date(today);
    yesterday.setDate(yesterday.getDate() - 1);

    const todayStr = today.toISOString().split('T')[0];
    const yesterdayStr = yesterday.toISOString().split('T')[0];

    // End date should be either today or yesterday
    expect([todayStr, yesterdayStr]).toContain(endValue);

    // Start date should be approximately 30 days before end date
    const startDate = new Date(startValue);
    const endDate = new Date(endValue);
    const daysDiff = Math.round((endDate.getTime() - startDate.getTime()) / (1000 * 60 * 60 * 24));

    // Allow for 29-31 days range to account for different month lengths
    expect(daysDiff).toBeGreaterThanOrEqual(28);
    expect(daysDiff).toBeLessThanOrEqual(31);
  });

  test('should validate that start date is before end date', async ({ page }) => {
    const periodSelect = page.locator('#comparison-period-select');
    await periodSelect.selectOption('custom');

    const startInput = page.locator('#comparison-start-date');
    const endInput = page.locator('#comparison-end-date');
    const applyButton = page.locator('#apply-custom-comparison');
    const errorEl = page.locator('#custom-date-error');

    // Set start date after end date
    await startInput.fill('2024-12-15');
    await endInput.fill('2024-12-10');

    // Trigger validation
    await startInput.blur();

    // Apply button should be disabled
    await expect(applyButton).toBeDisabled();

    // Error should be visible
    await expect(errorEl).not.toHaveClass(/hidden/);
  });

  test('should enable apply button with valid date range', async ({ page }) => {
    const periodSelect = page.locator('#comparison-period-select');
    await periodSelect.selectOption('custom');

    const startInput = page.locator('#comparison-start-date');
    const endInput = page.locator('#comparison-end-date');
    const applyButton = page.locator('#apply-custom-comparison');

    // Set valid date range
    await startInput.fill('2024-11-01');
    await endInput.fill('2024-11-30');

    // Apply button should be enabled
    await expect(applyButton).toBeEnabled();
  });

  test('should load custom comparison data when apply is clicked', async ({ page }) => {
    const periodSelect = page.locator('#comparison-period-select');
    await periodSelect.selectOption('custom');

    const startInput = page.locator('#comparison-start-date');
    const endInput = page.locator('#comparison-end-date');
    const applyButton = page.locator('#apply-custom-comparison');

    // Set valid date range
    await startInput.fill('2024-11-01');
    await endInput.fill('2024-11-30');

    // Click apply
    await applyButton.click();

    // Wait for data to load and update by checking the value changes
    const playbacksValue = page.locator('#comp-playbacks-value');
    await expect(playbacksValue).not.toHaveText('--', { timeout: TIMEOUTS.MEDIUM });
  });

  test('should show success feedback after applying', async ({ page }) => {
    const periodSelect = page.locator('#comparison-period-select');
    await periodSelect.selectOption('custom');

    const startInput = page.locator('#comparison-start-date');
    const endInput = page.locator('#comparison-end-date');
    const applyButton = page.locator('#apply-custom-comparison');

    await startInput.fill('2024-11-01');
    await endInput.fill('2024-11-30');
    await applyButton.click();

    // Button text should change to "Applied"
    await expect(applyButton).toHaveText('Applied');

    // Should revert back to "Apply"
    await expect(applyButton).toHaveText('Apply', { timeout: TIMEOUTS.MEDIUM });
  });

  test('should apply comparison on Enter key in date input', async ({ page }) => {
    const periodSelect = page.locator('#comparison-period-select');
    await periodSelect.selectOption('custom');

    const startInput = page.locator('#comparison-start-date');
    const endInput = page.locator('#comparison-end-date');

    await startInput.fill('2024-11-01');
    await endInput.fill('2024-11-30');

    // Press Enter on end date input
    await endInput.press('Enter');

    // Should trigger apply and update values
    const playbacksValue = page.locator('#comp-playbacks-value');
    await expect(playbacksValue).not.toHaveText('--', { timeout: TIMEOUTS.MEDIUM });
  });

  test('should focus start date input when switching to custom', async ({ page }) => {
    const periodSelect = page.locator('#comparison-period-select');
    await periodSelect.selectOption('custom');

    // Start date should be focused
    const startInput = page.locator('#comparison-start-date');
    await expect(startInput).toBeFocused({ timeout: TIMEOUTS.SHORT });
  });

  test('should show hint text about comparison period', async ({ page }) => {
    const periodSelect = page.locator('#comparison-period-select');
    await periodSelect.selectOption('custom');

    const hint = page.locator('.custom-date-hint');
    await expect(hint).toBeVisible();
    await expect(hint).toContainText('equivalent previous period');
  });

  test('should reject date ranges exceeding 365 days', async ({ page }) => {
    const periodSelect = page.locator('#comparison-period-select');
    await periodSelect.selectOption('custom');

    const startInput = page.locator('#comparison-start-date');
    const endInput = page.locator('#comparison-end-date');
    const applyButton = page.locator('#apply-custom-comparison');
    const errorEl = page.locator('#custom-date-error');

    // Set date range > 365 days
    await startInput.fill('2023-01-01');
    await endInput.fill('2024-12-31');
    await startInput.blur();

    // Apply button should be disabled
    await expect(applyButton).toBeDisabled();

    // Error should mention 365 days
    await expect(errorEl).not.toHaveClass(/hidden/);
    await expect(errorEl).toContainText('365');
  });

  test('should have proper aria attributes for accessibility', async ({ page }) => {
    const periodSelect = page.locator('#comparison-period-select');
    await periodSelect.selectOption('custom');

    const container = page.locator('#custom-date-container');
    const errorEl = page.locator('#custom-date-error');
    const applyButton = page.locator('#apply-custom-comparison');
    const startInput = page.locator('#comparison-start-date');
    const endInput = page.locator('#comparison-end-date');

    // Container should have aria-hidden when visible
    await expect(container).toHaveAttribute('aria-hidden', 'false');

    // Date inputs should have accessible labels
    await expect(startInput).toHaveAttribute('aria-label');
    await expect(endInput).toHaveAttribute('aria-label');

    // Date inputs should be linked to error via aria-describedby when invalid
    await startInput.fill('2024-12-15');
    await endInput.fill('2024-12-10');
    await startInput.blur();

    // After triggering validation error, inputs should reference error element
    await expect(startInput).toHaveAttribute('aria-invalid', 'true');
    await expect(endInput).toHaveAttribute('aria-invalid', 'true');

    // Error should have proper live region attributes
    await expect(errorEl).toHaveAttribute('role', 'alert');
    await expect(errorEl).toHaveAttribute('aria-live', 'polite');

    // Apply button should have descriptive aria-label
    await expect(applyButton).toHaveAttribute('aria-label');

    // Period select should have accessible label
    await expect(periodSelect).toHaveAttribute('aria-label');
  });

  test('should maintain focus management for keyboard users', async ({ page }) => {
    const periodSelect = page.locator('#comparison-period-select');

    // Focus the period select
    await periodSelect.focus();
    await expect(periodSelect).toBeFocused();

    // Select custom option
    await periodSelect.selectOption('custom');

    // Start date should receive focus after switching to custom
    const startInput = page.locator('#comparison-start-date');
    await expect(startInput).toBeFocused({ timeout: TIMEOUTS.SHORT });

    // Tab should move focus to end date
    await page.keyboard.press('Tab');
    const endInput = page.locator('#comparison-end-date');
    await expect(endInput).toBeFocused();

    // Tab should move focus to apply button
    await page.keyboard.press('Tab');
    const applyButton = page.locator('#apply-custom-comparison');
    await expect(applyButton).toBeFocused();
  });

  test('should disable inputs during loading', async ({ page }) => {
    // Create a delayed response to test loading state
    await page.route('**/api/v1/analytics/comparative*', async route => {
      await new Promise(resolve => setTimeout(resolve, TIMEOUTS.ANIMATION));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: mockComparativeData })
      });
    });

    const periodSelect = page.locator('#comparison-period-select');
    await periodSelect.selectOption('custom');

    const startInput = page.locator('#comparison-start-date');
    const endInput = page.locator('#comparison-end-date');
    const applyButton = page.locator('#apply-custom-comparison');

    await startInput.fill('2024-11-01');
    await endInput.fill('2024-11-30');

    // Click apply
    const applyPromise = applyButton.click();

    // During loading, inputs should be disabled
    await expect(startInput).toBeDisabled();
    await expect(endInput).toBeDisabled();
    await expect(applyButton).toHaveText('Loading...');

    await applyPromise;
  });
});
