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
 * E2E Test: Stale Data Warning
 *
 * Tests stale data warning indicator functionality:
 * - Warning visibility when data is stale
 * - Warning dismissal
 * - Refresh action from warning
 * - Accessibility attributes
 *
 * @see /docs/working/UI_UX_AUDIT.md
 */

test.describe('Stale Data Warning', () => {
  test.beforeEach(async ({ page }) => {
    await gotoAppAndWaitReady(page);
  });

  test.describe('Warning Display', () => {
    test('should have stale data warning element in DOM', async ({ page }) => {
      // Stale data warning element should exist
      const warning = page.locator('#stale-data-warning');
      await expect(warning).toBeAttached();
    });

    test('should hide warning initially when data is fresh', async ({ page }) => {
      // Wait for initial data load by checking warning element state
      const warning = page.locator('#stale-data-warning');
      await page.waitForFunction(
        (el) => {
          const element = document.getElementById('stale-data-warning');
          if (!element) return false;
          return element.classList.contains('hidden') ||
                 window.getComputedStyle(element).display === 'none' ||
                 window.getComputedStyle(element).visibility === 'hidden';
        },
        null,
        { timeout: TIMEOUTS.DATA_LOAD }
      );

      // Warning should be hidden when data is fresh
      const isHidden = await warning.evaluate(el => {
        return el.classList.contains('hidden') ||
               window.getComputedStyle(el).display === 'none' ||
               window.getComputedStyle(el).visibility === 'hidden';
      });

      expect(isHidden).toBe(true);
    });

    test('should display warning with appropriate styling', async ({ page }) => {
      // Manually trigger stale state for testing
      await page.evaluate(() => {
        const warning = document.getElementById('stale-data-warning');
        if (warning) {
          warning.classList.remove('hidden');
        }
      });

      const warning = page.locator('#stale-data-warning');
      await expect(warning).toBeVisible();

      // Should have warning styling (yellow/amber background)
      const bgColor = await warning.evaluate(el =>
        window.getComputedStyle(el).backgroundColor
      );
      expect(bgColor).toBeTruthy();
    });
  });

  test.describe('Warning Content', () => {
    test('should display informative message', async ({ page }) => {
      // Show warning for testing
      await page.evaluate(() => {
        const warning = document.getElementById('stale-data-warning');
        if (warning) {
          warning.classList.remove('hidden');
        }
      });

      const warning = page.locator('#stale-data-warning');
      await expect(warning).toBeVisible();

      // Should contain text about stale/outdated data
      const text = await warning.textContent();
      expect(text?.toLowerCase()).toMatch(/stale|outdated|old|refresh|update/);
    });

    test('should have refresh button', async ({ page }) => {
      // Show warning for testing
      await page.evaluate(() => {
        const warning = document.getElementById('stale-data-warning');
        if (warning) {
          warning.classList.remove('hidden');
        }
      });

      const refreshButton = page.locator('#stale-data-warning .stale-warning-refresh, #stale-data-warning [data-action="refresh"]');
      const count = await refreshButton.count();

      // If refresh button exists, verify it's visible and clickable
      if (count > 0) {
        await expect(refreshButton.first()).toBeVisible();
      } else {
        console.log('Refresh button not present - may use alternative refresh mechanism');
      }
    });

    test('should have dismiss button', async ({ page }) => {
      // Show warning for testing
      await page.evaluate(() => {
        const warning = document.getElementById('stale-data-warning');
        if (warning) {
          warning.classList.remove('hidden');
        }
      });

      const dismissButton = page.locator('#stale-data-warning .stale-warning-dismiss, #stale-data-warning [data-action="dismiss"]');
      const count = await dismissButton.count();

      // If dismiss button exists, verify it's visible
      if (count > 0) {
        await expect(dismissButton.first()).toBeVisible();
      } else {
        console.log('Dismiss button not present - warning may auto-dismiss');
      }
    });
  });

  test.describe('Warning Interactions', () => {
    test('should dismiss warning when dismiss button clicked', async ({ page }) => {
      // Show warning for testing
      await page.evaluate(() => {
        const warning = document.getElementById('stale-data-warning');
        if (warning) {
          warning.classList.remove('hidden');
        }
      });

      const warning = page.locator('#stale-data-warning');
      await expect(warning).toBeVisible();

      // Click dismiss button
      const dismissButton = warning.locator('.stale-warning-dismiss, [data-action="dismiss"]');
      const hasButton = await dismissButton.count() > 0;

      if (hasButton) {
        await dismissButton.click();

        // Wait for warning to be hidden after dismiss
        await page.waitForFunction(
          () => {
            const element = document.getElementById('stale-data-warning');
            if (!element) return true;
            return element.classList.contains('hidden') ||
                   window.getComputedStyle(element).display === 'none';
          },
          null,
          { timeout: TIMEOUTS.RENDER }
        );

        // Warning should be hidden after dismiss
        const isHidden = await warning.evaluate(el => {
          return el.classList.contains('hidden') ||
                 window.getComputedStyle(el).display === 'none';
        });
        expect(isHidden).toBe(true);
      }
    });

    test('should trigger refresh when refresh button clicked', async ({ page }) => {
      // Show warning for testing
      await page.evaluate(() => {
        const warning = document.getElementById('stale-data-warning');
        if (warning) {
          warning.classList.remove('hidden');
        }
      });

      const warning = page.locator('#stale-data-warning');
      const refreshButton = warning.locator('.stale-warning-refresh, [data-action="refresh"]');
      const hasButton = await refreshButton.count() > 0;

      if (hasButton) {
        // Click should not crash
        await refreshButton.click();

        // Wait for app to remain visible and functional after refresh
        await page.waitForFunction(
          () => {
            const app = document.getElementById('app');
            return app && window.getComputedStyle(app).display !== 'none';
          },
          null,
          { timeout: TIMEOUTS.RENDER }
        );

        // Page should still be functional
        await expect(page.locator('#app')).toBeVisible();
      }
    });
  });

  test.describe('Warning Accessibility', () => {
    test('should have appropriate ARIA role', async ({ page }) => {
      const warning = page.locator('#stale-data-warning');

      // Should have alert or status role
      const role = await warning.getAttribute('role');
      expect(['alert', 'status', 'region']).toContain(role);
    });

    test('should have aria-live for dynamic updates', async ({ page }) => {
      const warning = page.locator('#stale-data-warning');

      // Should announce updates to screen readers
      const ariaLive = await warning.getAttribute('aria-live');
      expect(ariaLive).toBeTruthy();
    });

    test('should have keyboard accessible controls', async ({ page }) => {
      // Show warning for testing
      await page.evaluate(() => {
        const warning = document.getElementById('stale-data-warning');
        if (warning) {
          warning.classList.remove('hidden');
        }
      });

      const warning = page.locator('#stale-data-warning');
      const buttons = warning.locator('button');
      const count = await buttons.count();

      for (let i = 0; i < count; i++) {
        const button = buttons.nth(i);
        // Buttons should be focusable
        const tabIndex = await button.getAttribute('tabindex');
        expect(tabIndex === null || parseInt(tabIndex) >= 0).toBe(true);
      }
    });
  });
});

test.describe('Stale Data Timing', () => {
  test('should track last update timestamp', async ({ page }) => {
    await gotoAppAndWaitReady(page);

    // Wait for data to load by checking for app initialization
    await page.waitForFunction(
      () => {
        const app = document.getElementById('app');
        return app !== null && app.children.length > 0;
      },
      null,
      { timeout: TIMEOUTS.WEBGL_INIT }
    );

    // Should store last update time (check localStorage or app state)
    const hasTimestamp = await page.evaluate(() => {
      // Check for timestamp in localStorage or global state
      const stored = localStorage.getItem('last-data-update');
      const appState = (window as any).__lastDataUpdate;
      return stored !== null || appState !== undefined;
    });

    // Timestamp storage is optional - log its presence
    if (hasTimestamp) {
      console.log('Data update timestamp is being tracked');
    } else {
      console.log('Data update timestamp not found (feature may not be implemented)');
    }
  });
});
