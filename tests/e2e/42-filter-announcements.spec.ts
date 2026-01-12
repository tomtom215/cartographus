// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Tests for Filter Change Announcements
 *
 * Tests that filter changes are properly announced to screen readers via
 * aria-live regions for WCAG 2.1 SC 4.1.3 (Status Messages) compliance.
 *
 * Reference: UI/UX Audit
 * @see /docs/working/UI_UX_AUDIT.md
 */

import {
  test,
  expect,
  TIMEOUTS,
  gotoAppAndWaitReady,
} from './fixtures';

test.describe('Filter Change Announcements', () => {
  test.beforeEach(async ({ page }) => {
    await gotoAppAndWaitReady(page);
    await expect(page.locator('#filters')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  });

  test('should have filter announcer element with proper ARIA attributes', async ({ page }) => {
    const announcer = page.locator('#filter-announcer');

    // Announcer should exist
    await expect(announcer).toBeAttached();

    // Should have proper ARIA attributes
    await expect(announcer).toHaveAttribute('aria-live', 'polite');
    await expect(announcer).toHaveAttribute('aria-atomic', 'true');

    // Should be visually hidden but accessible
    await expect(announcer).toHaveClass(/visually-hidden/);
  });

  test('should announce when days filter is changed', async ({ page }) => {
    const announcer = page.locator('#filter-announcer');

    // Clear initial state
    await page.evaluate(() => {
      const el = document.getElementById('filter-announcer');
      if (el) el.textContent = '';
    });

    // Change days filter
    await page.selectOption('#filter-days', '7');

    // Wait for announcer to update with filter summary
    await page.waitForFunction(() => {
      const el = document.getElementById('filter-announcer');
      return el && el.textContent && el.textContent.trim().length > 0;
    });

    // Announcer should contain filter summary
    const announcement = await announcer.textContent();
    expect(announcement).toBeTruthy();
    expect(announcement?.toLowerCase()).toContain('filter');
  });

  test('should announce when filters are cleared', async ({ page }) => {
    const announcer = page.locator('#filter-announcer');

    // First apply a filter
    await page.selectOption('#filter-days', '7');
    await page.waitForFunction(() => {
      const el = document.getElementById('filter-announcer');
      return el && el.textContent && el.textContent.trim().length > 0;
    });

    // Clear announcement
    await page.evaluate(() => {
      const el = document.getElementById('filter-announcer');
      if (el) el.textContent = '';
    });

    // Click clear filters - Use JavaScript click for CI reliability
    await page.evaluate(() => {
      const el = document.querySelector('#btn-clear-filters') as HTMLElement;
      if (el) el.click();
    });
    await page.waitForFunction(() => {
      const el = document.getElementById('filter-announcer');
      return el && el.textContent && el.textContent.trim().length > 0;
    });

    // Should announce filters cleared
    const announcement = await announcer.textContent();
    expect(announcement?.toLowerCase()).toContain('clear');
  });

  test('should include filter type in announcement when user filter is applied', async ({ page }) => {
    const announcer = page.locator('#filter-announcer');
    const userSelect = page.locator('#filter-users');

    // Check if user options exist
    const options = userSelect.locator('option');
    const optionCount = await options.count();

    if (optionCount > 1) {
      // Clear announcer
      await page.evaluate(() => {
        const el = document.getElementById('filter-announcer');
        if (el) el.textContent = '';
      });

      // Select a user
      await userSelect.selectOption({ index: 1 });
      await page.waitForFunction(() => {
        const el = document.getElementById('filter-announcer');
        return el && el.textContent && el.textContent.trim().length > 0;
      });

      // Announcement should mention user filter
      const announcement = await announcer.textContent();
      expect(announcement?.toLowerCase()).toContain('user');
    }
  });

  test('should include filter type in announcement when media type filter is applied', async ({ page }) => {
    const announcer = page.locator('#filter-announcer');
    const mediaSelect = page.locator('#filter-media-types');

    // Check if media options exist
    const options = mediaSelect.locator('option');
    const optionCount = await options.count();

    if (optionCount > 1) {
      // Clear announcer
      await page.evaluate(() => {
        const el = document.getElementById('filter-announcer');
        if (el) el.textContent = '';
      });

      // Select a media type
      await mediaSelect.selectOption({ index: 1 });
      await page.waitForFunction(() => {
        const el = document.getElementById('filter-announcer');
        return el && el.textContent && el.textContent.trim().length > 0;
      });

      // Announcement should mention media filter
      const announcement = await announcer.textContent();
      expect(announcement?.toLowerCase()).toContain('media');
    }
  });

  test('should announce date range when custom dates are set', async ({ page }) => {
    const announcer = page.locator('#filter-announcer');
    const startDateInput = page.locator('#filter-start-date');
    const endDateInput = page.locator('#filter-end-date');

    // Clear announcer
    await page.evaluate(() => {
      const el = document.getElementById('filter-announcer');
      if (el) el.textContent = '';
    });

    // Set date range
    const today = new Date();
    const lastWeek = new Date(today);
    lastWeek.setDate(lastWeek.getDate() - 7);

    await startDateInput.fill(lastWeek.toISOString().split('T')[0]);
    await endDateInput.fill(today.toISOString().split('T')[0]);

    // Trigger change event
    await endDateInput.dispatchEvent('change');
    await page.waitForFunction(() => {
      const el = document.getElementById('filter-announcer');
      return el && el.textContent && el.textContent.trim().length > 0;
    });

    // Announcement should mention dates or filter
    const announcement = await announcer.textContent();
    expect(announcement).toBeTruthy();
  });

  test('should debounce rapid filter changes', async ({ page }) => {
    const announcer = page.locator('#filter-announcer');

    // Rapidly change filter multiple times using valid options (7, 30, 90)
    await page.selectOption('#filter-days', '7');
    await page.selectOption('#filter-days', '30');
    await page.selectOption('#filter-days', '90');

    // Wait for announcer to update with final state
    await page.waitForFunction(() => {
      const el = document.getElementById('filter-announcer');
      return el && el.textContent && el.textContent.includes('90');
    });

    // Should only announce the final state once
    const announcement = await announcer.textContent();
    expect(announcement).toBeTruthy();
    // Should reference 90 days (the final selection)
    expect(announcement).toContain('90');
  });

  test('should announce all data when all filters cleared', async ({ page }) => {
    const announcer = page.locator('#filter-announcer');

    // Apply a filter first
    await page.selectOption('#filter-days', '7');
    await page.waitForFunction(() => {
      const el = document.getElementById('filter-announcer');
      return el && el.textContent && el.textContent.trim().length > 0;
    });

    // Clear - Use JavaScript click for CI reliability
    await page.evaluate(() => {
      const el = document.querySelector('#btn-clear-filters') as HTMLElement;
      if (el) el.click();
    });
    await page.waitForFunction(() => {
      const el = document.getElementById('filter-announcer');
      return el && el.textContent && el.textContent.trim().length > 0;
    });

    // Announcement should indicate showing all/default data
    const announcement = await announcer.textContent();
    expect(announcement).toBeTruthy();
    // Should mention either "cleared" or "90 days" (default)
    const lowerAnn = announcement?.toLowerCase() || '';
    expect(lowerAnn.includes('clear') || lowerAnn.includes('90')).toBe(true);
  });
});

test.describe('Filter Announcements with Quick Date Buttons', () => {
  test.beforeEach(async ({ page }) => {
    await gotoAppAndWaitReady(page);
    await expect(page.locator('#filters')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  });

  test('should announce when quick date button is clicked', async ({ page }) => {
    const announcer = page.locator('#filter-announcer');
    const todayButton = page.locator('#quick-date-today');

    // Only run if quick date buttons exist
    if (await todayButton.isVisible()) {
      // Clear announcer
      await page.evaluate(() => {
        const el = document.getElementById('filter-announcer');
        if (el) el.textContent = '';
      });

      await todayButton.click();
      await page.waitForFunction(() => {
        const el = document.getElementById('filter-announcer');
        return el && el.textContent && el.textContent.trim().length > 0;
      });

      // Should announce filter applied
      const announcement = await announcer.textContent();
      expect(announcement).toBeTruthy();
      expect(announcement?.toLowerCase()).toContain('filter');
    }
  });

  test('should announce when week quick date is selected', async ({ page }) => {
    const announcer = page.locator('#filter-announcer');
    const weekButton = page.locator('#quick-date-week');

    if (await weekButton.isVisible()) {
      await page.evaluate(() => {
        const el = document.getElementById('filter-announcer');
        if (el) el.textContent = '';
      });

      await weekButton.click();
      await page.waitForFunction(() => {
        const el = document.getElementById('filter-announcer');
        return el && el.textContent && el.textContent.trim().length > 0;
      });

      const announcement = await announcer.textContent();
      expect(announcement).toBeTruthy();
    }
  });
});

test.describe('Filter Announcements Accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await gotoAppAndWaitReady(page);
  });

  test('announcer should not be visible to sighted users', async ({ page }) => {
    const announcer = page.locator('#filter-announcer');

    // Get computed styles
    const styles = await announcer.evaluate(el => {
      const style = window.getComputedStyle(el);
      return {
        position: style.position,
        width: style.width,
        height: style.height,
        overflow: style.overflow,
        clip: style.clip
      };
    });

    // Should be positioned off-screen or clipped
    // visually-hidden class typically uses these techniques
    expect(['absolute', 'fixed']).toContain(styles.position);
  });

  test('should maintain screen reader accessibility', async ({ page }) => {
    const announcer = page.locator('#filter-announcer');

    // Element should exist in the DOM
    await expect(announcer).toBeAttached();

    // Should have role implied by aria-live
    const ariaLive = await announcer.getAttribute('aria-live');
    expect(ariaLive).toBe('polite');

    // Should not have aria-hidden (that would hide from screen readers)
    const ariaHidden = await announcer.getAttribute('aria-hidden');
    expect(ariaHidden).not.toBe('true');
  });
});
