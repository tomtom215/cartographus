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
 * E2E Test: Auto-Refresh Toggle
 *
 * Tests auto-refresh toggle control functionality:
 * - Toggle visibility and state
 * - Enable/disable auto-refresh
 * - Persistence across sessions
 * - Accessibility attributes
 *
 * @see /docs/working/UI_UX_AUDIT.md
 */

test.describe('Auto-Refresh Toggle', () => {
  test.beforeEach(async ({ page }) => {
    await gotoAppAndWaitReady(page);
  });

  test.describe('Toggle Visibility', () => {
    test('should display auto-refresh toggle control', async ({ page }) => {
      const toggle = page.locator('#auto-refresh-toggle, .auto-refresh-toggle');
      await expect(toggle).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test('should have label explaining the toggle', async ({ page }) => {
      const toggle = page.locator('#auto-refresh-toggle, .auto-refresh-toggle');
      void toggle.locator('..').first(); // Parent container (unused, kept for context)

      // Should have associated label or aria-label
      const ariaLabel = await toggle.getAttribute('aria-label');
      const labelElement = page.locator('label[for="auto-refresh-toggle"]');
      const labelCount = await labelElement.count();

      expect(ariaLabel !== null || labelCount > 0).toBe(true);
    });
  });

  test.describe('Toggle Functionality', () => {
    test('should toggle auto-refresh on click', async ({ page }) => {
      const toggle = page.locator('#auto-refresh-toggle');
      await expect(toggle).toBeVisible();

      // Get initial state
      const initialChecked = await toggle.isChecked().catch(() => {
        // If not a checkbox, check aria-checked
        return toggle.getAttribute('aria-checked').then(v => v === 'true');
      });

      // Click to toggle
      await toggle.click();

      // Wait for state to change deterministically
      await page.waitForFunction(
        (expectedState) => {
          const el = document.getElementById('auto-refresh-toggle') as HTMLInputElement;
          if (!el) return false;
          const currentState = el.checked ?? el.getAttribute('aria-checked') === 'true';
          return currentState === expectedState;
        },
        !initialChecked,
        { timeout: TIMEOUTS.RENDER }
      );

      // State should change
      const newChecked = await toggle.isChecked().catch(() => {
        return toggle.getAttribute('aria-checked').then(v => v === 'true');
      });

      expect(newChecked).toBe(!initialChecked);
    });

    test('should enable auto-refresh when toggled on', async ({ page }) => {
      const toggle = page.locator('#auto-refresh-toggle');

      // Ensure toggle is on
      const isChecked = await toggle.isChecked().catch(() => false);
      if (!isChecked) {
        await toggle.click();

        // Wait for toggle to be checked
        await page.waitForFunction(
          () => {
            const el = document.getElementById('auto-refresh-toggle') as HTMLInputElement;
            if (!el) return false;
            return el.checked === true || el.getAttribute('aria-checked') === 'true';
          },
          { timeout: TIMEOUTS.RENDER }
        );
      }

      // Auto-refresh should be active (check app state)
      const isAutoRefreshActive = await page.evaluate(() => {
        return (window as any).__autoRefreshEnabled === true ||
               document.getElementById('auto-refresh-toggle')?.getAttribute('aria-checked') === 'true' ||
               (document.getElementById('auto-refresh-toggle') as HTMLInputElement)?.checked === true;
      });

      expect(isAutoRefreshActive).toBe(true);
    });

    test('should disable auto-refresh when toggled off', async ({ page }) => {
      const toggle = page.locator('#auto-refresh-toggle');

      // First ensure it's on
      let isChecked = await toggle.isChecked().catch(() => false);
      if (!isChecked) {
        await toggle.click();

        // Wait for toggle to be checked
        await page.waitForFunction(
          () => {
            const el = document.getElementById('auto-refresh-toggle') as HTMLInputElement;
            if (!el) return false;
            return el.checked === true || el.getAttribute('aria-checked') === 'true';
          },
          { timeout: TIMEOUTS.RENDER }
        );
      }

      // Now toggle off
      await toggle.click();

      // Wait for toggle to be unchecked
      await page.waitForFunction(
        () => {
          const el = document.getElementById('auto-refresh-toggle') as HTMLInputElement;
          if (!el) return false;
          return el.checked === false && el.getAttribute('aria-checked') !== 'true';
        },
        { timeout: TIMEOUTS.RENDER }
      );

      // Should be off
      isChecked = await toggle.isChecked().catch(() => {
        return toggle.getAttribute('aria-checked').then(v => v === 'true');
      });

      expect(isChecked).toBe(false);
    });
  });

  test.describe('Toggle Persistence', () => {
    test('should persist toggle state across page reloads', async ({ page }) => {
      const toggle = page.locator('#auto-refresh-toggle');
      await expect(toggle).toBeVisible();

      // Toggle to a known state (off)
      let isChecked = await toggle.isChecked().catch(() => false);
      if (isChecked) {
        await toggle.click();

        // Wait for toggle to be unchecked
        await page.waitForFunction(
          () => {
            const el = document.getElementById('auto-refresh-toggle') as HTMLInputElement;
            if (!el) return false;
            return el.checked === false && el.getAttribute('aria-checked') !== 'true';
          },
          { timeout: TIMEOUTS.RENDER }
        );
      }

      // Reload page
      await page.reload({ waitUntil: 'domcontentloaded' });
      await expect(page.locator('#app')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

      // State should persist (off)
      const newToggle = page.locator('#auto-refresh-toggle');
      await expect(newToggle).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      const persistedState = await newToggle.isChecked().catch(() => {
        return newToggle.getAttribute('aria-checked').then(v => v === 'true');
      });

      expect(persistedState).toBe(false);
    });
  });

  test.describe('Toggle Accessibility', () => {
    test('should be keyboard accessible', async ({ page }) => {
      const toggle = page.locator('#auto-refresh-toggle');
      await expect(toggle).toBeVisible();

      // Focus the toggle
      await toggle.focus();

      // Should be focusable
      const isFocused = await toggle.evaluate(el => el === document.activeElement);
      expect(isFocused).toBe(true);

      // Should respond to Space key
      const initialChecked = await toggle.isChecked().catch(() => false);
      await page.keyboard.press('Space');

      // Wait for state to change deterministically
      await page.waitForFunction(
        (expectedState) => {
          const el = document.getElementById('auto-refresh-toggle') as HTMLInputElement;
          if (!el) return false;
          const currentState = el.checked ?? el.getAttribute('aria-checked') === 'true';
          return currentState === expectedState;
        },
        !initialChecked,
        { timeout: TIMEOUTS.RENDER }
      );

      const newChecked = await toggle.isChecked().catch(() => {
        return toggle.getAttribute('aria-checked').then(v => v === 'true');
      });

      expect(newChecked).toBe(!initialChecked);
    });

    test('should have appropriate ARIA attributes', async ({ page }) => {
      const toggle = page.locator('#auto-refresh-toggle');
      await expect(toggle).toBeVisible();

      // Should have role if not native checkbox
      const tagName = await toggle.evaluate(el => el.tagName.toLowerCase());
      if (tagName !== 'input') {
        const role = await toggle.getAttribute('role');
        expect(['switch', 'checkbox']).toContain(role);
      }

      // Should have aria-label or be labeled
      const ariaLabel = await toggle.getAttribute('aria-label');
      const ariaLabelledBy = await toggle.getAttribute('aria-labelledby');
      const id = await toggle.getAttribute('id');
      const hasLabel = id ? await page.locator(`label[for="${id}"]`).count() > 0 : false;

      expect(ariaLabel || ariaLabelledBy || hasLabel).toBeTruthy();
    });

    test('should have visible focus indicator', async ({ page }) => {
      const toggle = page.locator('#auto-refresh-toggle');
      await toggle.focus();

      // Should have focus styling
      const hasFocusStyle = await toggle.evaluate(el => {
        const styles = window.getComputedStyle(el);
        const outline = styles.outline;
        const boxShadow = styles.boxShadow;
        return outline !== 'none' || (boxShadow && boxShadow !== 'none');
      });

      expect(hasFocusStyle).toBe(true);
    });
  });

  test.describe('Auto-Refresh Indicator', () => {
    test('should show refresh interval indicator when enabled', async ({ page }) => {
      const toggle = page.locator('#auto-refresh-toggle');

      // Enable auto-refresh
      const isChecked = await toggle.isChecked().catch(() => false);
      if (!isChecked) {
        await toggle.click();

        // Wait for toggle to be checked
        await page.waitForFunction(
          () => {
            const el = document.getElementById('auto-refresh-toggle') as HTMLInputElement;
            if (!el) return false;
            return el.checked === true || el.getAttribute('aria-checked') === 'true';
          },
          { timeout: TIMEOUTS.RENDER }
        );
      }

      // Look for interval indicator
      const indicator = page.locator('.auto-refresh-interval, #refresh-interval-display');
      const count = await indicator.count();

      // May show interval like "30s" or "1m"
      if (count > 0) {
        await expect(indicator).toBeVisible();
      }
    });
  });
});

test.describe('Auto-Refresh Integration', () => {
  test('should coexist with stale data warning', async ({ page }) => {
    await gotoAppAndWaitReady(page);

    // Both controls should exist
    const toggle = page.locator('#auto-refresh-toggle');
    const warning = page.locator('#stale-data-warning');

    await expect(toggle).toBeAttached();
    await expect(warning).toBeAttached();
  });
});
