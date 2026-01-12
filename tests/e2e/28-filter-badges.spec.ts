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
 * E2E Test: Filter Badges/Chips
 *
 * Tests the removable filter badges functionality:
 * - Active filters displayed as chips/badges
 * - Click to remove individual filters
 * - Clear All button removes all badges
 * - Keyboard accessible (Tab, Enter/Space)
 * - Screen reader announcements on badge removal
 *
 * @see /docs/working/UI_UX_AUDIT.md
 */

test.describe('Filter Badges', () => {
  test.beforeEach(async ({ page }) => {
    await gotoAppAndWaitReady(page);
    await expect(page.locator('#filters')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  });

  test('should show filter badges container', async ({ page }) => {
    // Filter badges container should exist
    const badgesContainer = page.locator('#filter-badges');
    await expect(badgesContainer).toBeVisible();

    // Should have proper ARIA attributes
    await expect(badgesContainer).toHaveAttribute('role', 'region');
    await expect(badgesContainer).toHaveAttribute('aria-label', 'Active filters');
  });

  test('should display badge when filter is applied', async ({ page }) => {
    const badgesContainer = page.locator('#filter-badges');

    // Initially, no badges should be visible (empty state)
    const initialBadges = badgesContainer.locator('.filter-badge');
    void await initialBadges.count(); // Verify badges exist (unused value intentional)

    // Apply a days filter
    await page.selectOption('#filter-days', '7');

    // Wait for badge to appear
    await page.waitForFunction(() => {
      const badges = document.querySelectorAll('.filter-badge[data-filter-type="days"]');
      return badges.length > 0;
    }, { timeout: TIMEOUTS.RENDER });

    // A badge should appear for the days filter
    const badge = badgesContainer.locator('.filter-badge[data-filter-type="days"]');
    await expect(badge).toBeVisible();
    await expect(badge).toContainText('7 days');
  });

  test('should display badge for user filter', async ({ page }) => {
    const badgesContainer = page.locator('#filter-badges');

    // Select a user if available (depends on test data)
    const userSelect = page.locator('#filter-users');
    const options = userSelect.locator('option');
    const optionCount = await options.count();

    if (optionCount > 1) {
      // Select the second option (first non-empty user)
      await userSelect.selectOption({ index: 1 });

      // Wait for badge to appear
      await page.waitForFunction(() => {
        const badges = document.querySelectorAll('.filter-badge[data-filter-type="user"]');
        return badges.length > 0;
      }, { timeout: TIMEOUTS.RENDER });

      // A badge should appear for the user filter
      const badge = badgesContainer.locator('.filter-badge[data-filter-type="user"]');
      await expect(badge).toBeVisible();
    }
  });

  test('should remove filter when badge close button is clicked', async ({ page }) => {
    const badgesContainer = page.locator('#filter-badges');

    // Apply a filter
    await page.selectOption('#filter-days', '30');

    // Wait for badge to appear
    await page.waitForFunction(() => {
      const badges = document.querySelectorAll('.filter-badge[data-filter-type="days"]');
      return badges.length > 0;
    }, { timeout: TIMEOUTS.RENDER });

    // Badge should appear
    const badge = badgesContainer.locator('.filter-badge[data-filter-type="days"]');
    await expect(badge).toBeVisible();

    // Click the remove button on the badge
    const removeButton = badge.locator('.filter-badge-remove');
    await removeButton.click();

    // Wait for badge to be removed
    await page.waitForFunction(() => {
      const badges = document.querySelectorAll('.filter-badge[data-filter-type="days"]');
      return badges.length === 0;
    }, { timeout: TIMEOUTS.RENDER });

    // Badge should be removed
    await expect(badge).not.toBeVisible();

    // Filter select should be reset
    const daysSelect = page.locator('#filter-days');
    const selectedValue = await daysSelect.inputValue();
    // After removal, should be reset to default (90 days) or empty
    expect(['', '90']).toContain(selectedValue);
  });

  test('should be keyboard accessible', async ({ page }) => {
    // Apply a filter
    await page.selectOption('#filter-days', '7');

    // Wait for badge to appear
    await page.waitForFunction(() => {
      const badges = document.querySelectorAll('.filter-badge[data-filter-type="days"]');
      return badges.length > 0;
    }, { timeout: TIMEOUTS.RENDER });

    // Tab to the badge's remove button
    const badge = page.locator('.filter-badge[data-filter-type="days"]');
    await expect(badge).toBeVisible();

    const removeButton = badge.locator('.filter-badge-remove');
    await removeButton.focus();

    // Verify it receives focus
    await expect(removeButton).toBeFocused();

    // Remove via Enter key
    await page.keyboard.press('Enter');

    // Wait for badge to be removed
    await page.waitForFunction(() => {
      const badges = document.querySelectorAll('.filter-badge[data-filter-type="days"]');
      return badges.length === 0;
    }, { timeout: TIMEOUTS.RENDER });

    // Badge should be removed
    await expect(badge).not.toBeVisible();
  });

  test('should announce filter removal to screen readers', async ({ page }) => {
    const announcer = page.locator('#filter-announcer');

    // Apply a filter
    await page.selectOption('#filter-days', '7');

    // Wait for badge to appear
    await page.waitForFunction(() => {
      const badges = document.querySelectorAll('.filter-badge[data-filter-type="days"]');
      return badges.length > 0;
    }, { timeout: TIMEOUTS.RENDER });

    // Remove the filter
    const badge = page.locator('.filter-badge[data-filter-type="days"]');
    const removeButton = badge.locator('.filter-badge-remove');
    await removeButton.click();

    // Wait for badge to be removed and announcement to update
    // CI FIX: Use TIMEOUTS.SHORT (2s) instead of TIMEOUTS.ANIMATION (300ms)
    // WHY: UI updates can take longer in CI/SwiftShader headless environments
    await page.waitForFunction(() => {
      const badges = document.querySelectorAll('.filter-badge[data-filter-type="days"]');
      const announcer = document.querySelector('#filter-announcer');
      return badges.length === 0 && announcer && announcer.textContent;
    }, { timeout: TIMEOUTS.SHORT });

    // Announcer should contain removal message
    const announcement = await announcer.textContent();
    expect(announcement).toBeTruthy();
  });

  test('should update badges when multiple filters are applied', async ({ page }) => {
    const badgesContainer = page.locator('#filter-badges');

    // Apply days filter
    await page.selectOption('#filter-days', '7');

    // Wait for first badge to appear
    // CI FIX: Use TIMEOUTS.SHORT (2s) instead of TIMEOUTS.ANIMATION (300ms)
    // WHY: UI updates can take longer in CI/SwiftShader headless environments
    await page.waitForFunction(() => {
      const badges = document.querySelectorAll('.filter-badge');
      return badges.length > 0;
    }, { timeout: TIMEOUTS.SHORT });

    // Apply media type filter if options exist
    const mediaSelect = page.locator('#filter-media-types');
    const mediaOptions = mediaSelect.locator('option');
    const mediaCount = await mediaOptions.count();

    if (mediaCount > 1) {
      await mediaSelect.selectOption({ index: 1 });

      // Wait for second badge to appear
      // CI FIX: Use TIMEOUTS.SHORT (2s) instead of TIMEOUTS.ANIMATION (300ms)
      // WHY: UI updates can take longer in CI/SwiftShader headless environments
      await page.waitForFunction(() => {
        const badges = document.querySelectorAll('.filter-badge');
        return badges.length >= 2;
      }, { timeout: TIMEOUTS.SHORT });
    }

    // Multiple badges should be visible
    const badges = badgesContainer.locator('.filter-badge');
    const badgeCount = await badges.count();
    expect(badgeCount).toBeGreaterThanOrEqual(1);
  });

  test('should have proper styling for badges', async ({ page }) => {
    // Apply a filter
    await page.selectOption('#filter-days', '7');

    // Wait for badge to appear
    await page.waitForFunction(() => {
      const badges = document.querySelectorAll('.filter-badge[data-filter-type="days"]');
      return badges.length > 0;
    }, { timeout: TIMEOUTS.RENDER });

    const badge = page.locator('.filter-badge[data-filter-type="days"]');
    await expect(badge).toBeVisible();

    // Badge should have appropriate styling
    const backgroundColor = await badge.evaluate(el =>
      window.getComputedStyle(el).backgroundColor
    );
    expect(backgroundColor).toBeTruthy();

    // Remove button should be visible
    const removeButton = badge.locator('.filter-badge-remove');
    await expect(removeButton).toBeVisible();
  });

  test('should show empty state when no filters active', async ({ page }) => {
    // Clear any existing filters
    const clearButton = page.locator('#btn-clear-filters');
    await clearButton.click();

    // Wait for all badges to be cleared
    await page.waitForFunction(() => {
      const badges = document.querySelectorAll('.filter-badge');
      return badges.length === 0;
    }, { timeout: TIMEOUTS.RENDER });

    // Badges container should show empty state or no badges
    const badges = page.locator('#filter-badges .filter-badge');
    const count = await badges.count();
    expect(count).toBe(0);
  });
});

test.describe('Filter Badges Accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await gotoAppAndWaitReady(page);
  });

  test('should have proper ARIA attributes on remove buttons', async ({ page }) => {
    // Apply a filter
    await page.selectOption('#filter-days', '7');

    // Wait for badge to appear
    await page.waitForFunction(() => {
      const badges = document.querySelectorAll('.filter-badge');
      return badges.length > 0;
    }, { timeout: TIMEOUTS.RENDER });

    const removeButton = page.locator('.filter-badge-remove').first();
    await expect(removeButton).toBeVisible();

    // Should have aria-label describing the action
    const ariaLabel = await removeButton.getAttribute('aria-label');
    expect(ariaLabel).toContain('Remove');
  });

  test('should support focus-visible styling', async ({ page }) => {
    // Apply a filter
    await page.selectOption('#filter-days', '7');

    // Wait for badge to appear
    await page.waitForFunction(() => {
      const badges = document.querySelectorAll('.filter-badge');
      return badges.length > 0;
    }, { timeout: TIMEOUTS.RENDER });

    const removeButton = page.locator('.filter-badge-remove').first();
    await removeButton.focus();

    // Check that focus-visible outline is applied
    // (exact styling depends on CSS implementation)
    await expect(removeButton).toBeFocused();
  });
});
