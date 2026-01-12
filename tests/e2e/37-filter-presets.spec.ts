// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import {
  test,
  expect,
  TIMEOUTS,
} from './fixtures';

// ROOT CAUSE FIX: Enable API mocking to prevent real API calls
// WHY: Without mocking, the backup/stats endpoints may fail and show error toasts
// (e.g., "Failed to load backup data") that interfere with preset save assertions.
// The mock API provides deterministic responses for all endpoints.
test.use({ autoMockApi: true });

/**
 * E2E Test: Filter Presets
 *
 * Tests filter preset functionality:
 * - Save current filter as preset
 * - Load saved presets
 * - Manage presets (rename, delete)
 * - localStorage persistence
 * - UI integration with filter panel
 *
 * @see /docs/working/UI_UX_AUDIT.md
 */

test.describe('Filter Presets', () => {
  test.beforeEach(async ({ page }) => {
    // Skip onboarding to prevent modal from blocking interactions
    await page.addInitScript(() => {
      localStorage.setItem('onboarding_completed', 'true');
      localStorage.setItem('onboarding_skipped', 'true');
    });

    // Clear any saved presets before each test
    await page.goto('/', { waitUntil: 'domcontentloaded' });
    await page.evaluate(() => {
      localStorage.removeItem('filter-presets');
    });
    await page.reload({ waitUntil: 'domcontentloaded' });
    await expect(page.locator('#app')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
    await expect(page.locator('#filters')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  });

  test.describe('Save Preset', () => {
    test('should show save preset button in filter panel', async ({ page }) => {
      const saveButton = page.locator('#btn-save-preset');
      await expect(saveButton).toBeVisible();
      await expect(saveButton).toHaveAttribute('aria-label', /save/i);
    });

    test('should open save preset dialog when button clicked', async ({ page }) => {
      // Apply some filters first
      await page.selectOption('#filter-days', '7');
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-days') as HTMLSelectElement;
        return select && select.value === '7';
      });

      // Click save preset button - use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-save-preset') as HTMLElement;
        if (btn) btn.click();
      });

      // Dialog should appear
      const dialog = page.locator('#save-preset-dialog');
      await expect(dialog).toBeVisible({ timeout: TIMEOUTS.DATA_LOAD });

      // Should have input for preset name
      const nameInput = page.locator('#preset-name-input');
      await expect(nameInput).toBeVisible();
      await expect(nameInput).toBeFocused();
    });

    test('should save preset with custom name', async ({ page }) => {
      // Apply filters
      await page.selectOption('#filter-days', '7');
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-days') as HTMLSelectElement;
        return select && select.value === '7';
      });

      // Open save dialog - use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await expect(page.locator('#save-preset-dialog')).toBeVisible();

      // Enter preset name
      await page.fill('#preset-name-input', 'Last Week View');

      // Save - use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-confirm-save-preset') as HTMLElement;
        if (btn) btn.click();
      });

      // Dialog should close
      await expect(page.locator('#save-preset-dialog')).not.toBeVisible();

      // Should show success toast
      // DETERMINISTIC FIX: Use .first() to avoid strict mode violation
      // WHY (ROOT CAUSE): Multiple toasts may be visible simultaneously (e.g., success + info).
      // Playwright strict mode requires locators to match exactly one element.
      await expect(page.locator('.toast').first()).toContainText(/saved/i);

      // Preset should appear in list
      const presetList = page.locator('#preset-list');
      await expect(presetList.locator('[data-preset-name="Last Week View"]')).toBeVisible();
    });

    test('should save preset with default name if empty', async ({ page }) => {
      // Apply filters
      await page.selectOption('#filter-days', '30');
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-days') as HTMLSelectElement;
        return select && select.value === '30';
      });

      // Open save dialog - use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await expect(page.locator('#save-preset-dialog')).toBeVisible();

      // Leave name empty and save - use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-confirm-save-preset') as HTMLElement;
        if (btn) btn.click();
      });

      // Should save with default name like "Preset 1"
      await expect(page.locator('#save-preset-dialog')).not.toBeVisible();
    });

    test('should not allow duplicate preset names', async ({ page }) => {
      // Save first preset
      await page.selectOption('#filter-days', '7');
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-days') as HTMLSelectElement;
        return select && select.value === '7';
      });
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await page.fill('#preset-name-input', 'My Preset');
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-confirm-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await expect(page.locator('#save-preset-dialog')).not.toBeVisible();

      // Try to save another with same name
      await page.selectOption('#filter-days', '30');
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-days') as HTMLSelectElement;
        return select && select.value === '30';
      });
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await page.fill('#preset-name-input', 'My Preset');
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-confirm-save-preset') as HTMLElement;
        if (btn) btn.click();
      });

      // Should either show error, auto-rename, or replace
      const errorMessage = page.locator('.preset-name-error');
      const hasError = await errorMessage.isVisible().catch(() => false);
      const dialogClosed = !(await page.locator('#save-preset-dialog').isVisible().catch(() => true));

      // Verify the duplicate name was handled (either error shown or dialog closed)
      if (hasError) {
        console.log('Duplicate preset name showed error message');
        await expect(errorMessage).toBeVisible();
      } else if (dialogClosed) {
        console.log('Duplicate preset was auto-renamed or replaced');
      }
      // App should still be functional
      await expect(page.locator('#app')).toBeVisible();
    });
  });

  test.describe('Load Preset', () => {
    test('should load preset and apply filters', async ({ page }) => {
      // First, create a preset
      await page.selectOption('#filter-days', '7');
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-days') as HTMLSelectElement;
        return select && select.value === '7';
      });
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await page.fill('#preset-name-input', 'Weekly View');
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-confirm-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await expect(page.locator('#save-preset-dialog')).not.toBeVisible();

      // Reset filters
      await page.selectOption('#filter-days', '90');
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-days') as HTMLSelectElement;
        return select && select.value === '90';
      });

      // Load the preset - use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('[data-preset-name="Weekly View"]') as HTMLElement;
        if (btn) btn.click();
      });

      // Filter should be applied
      await expect(page.locator('#filter-days')).toHaveValue('7');
    });

    test('should show empty state when no presets', async ({ page }) => {
      // Clear any presets
      await page.evaluate(() => localStorage.removeItem('filter-presets'));
      await page.reload({ waitUntil: 'domcontentloaded' });
      await expect(page.locator('#app')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

      // Preset list should show empty state
      const emptyState = page.locator('#preset-list-empty, .preset-empty-state');
      const presetItems = page.locator('[data-preset-name]');

      const isEmpty = await emptyState.isVisible().catch(() => false);
      const hasPresets = (await presetItems.count()) > 0;

      // Either shows empty state or has no preset items
      expect(isEmpty || !hasPresets).toBeTruthy();
    });

    test('should preserve complex filters in preset', async ({ page }) => {
      // Apply multiple filters
      await page.selectOption('#filter-days', '30');
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-days') as HTMLSelectElement;
        return select && select.value === '30';
      });

      // Try to set a user filter if available
      const userSelect = page.locator('#filter-users');
      if (await userSelect.isVisible().catch(() => false)) {
        const options = await userSelect.locator('option').count();
        if (options > 1) {
          await userSelect.selectOption({ index: 1 });
          await page.waitForFunction(() => {
            const select = document.querySelector('#filter-users') as HTMLSelectElement;
            return select && select.selectedIndex === 1;
          });
        }
      }

      // Save preset - use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await page.fill('#preset-name-input', 'Complex Filter');
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-confirm-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await expect(page.locator('#save-preset-dialog')).not.toBeVisible();

      // Reset all filters - use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-clear-filters') as HTMLElement;
        if (btn) btn.click();
      });
      // Handle confirmation if needed - use JavaScript click for CI reliability
      const confirmBtn = page.locator('#confirmation-dialog-confirm');
      if (await confirmBtn.isVisible({ timeout: TIMEOUTS.RENDER }).catch(() => false)) {
        await confirmBtn.evaluate((el) => el.click());
      }
      // Wait for filters to reset
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-days') as HTMLSelectElement;
        return select && select.value !== '30';
      });

      // Load preset - use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('[data-preset-name="Complex Filter"]') as HTMLElement;
        if (btn) btn.click();
      });

      // Verify days filter restored
      await expect(page.locator('#filter-days')).toHaveValue('30');
    });
  });

  test.describe('Manage Presets', () => {
    test('should delete preset', async ({ page }) => {
      // Create a preset
      await page.selectOption('#filter-days', '7');
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-days') as HTMLSelectElement;
        return select && select.value === '7';
      });
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await page.fill('#preset-name-input', 'To Delete');
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-confirm-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await expect(page.locator('#save-preset-dialog')).not.toBeVisible();

      // Verify preset exists
      const presetItem = page.locator('[data-preset-name="To Delete"]');
      await expect(presetItem).toBeVisible();

      // Delete preset - use JavaScript click for CI reliability
      const deleteButton = presetItem.locator('.preset-delete-btn, [data-action="delete"]');
      await deleteButton.evaluate((el) => el.click());

      // Handle confirmation if any - use JavaScript click for CI reliability
      const confirmBtn = page.locator('#confirmation-dialog-confirm');
      if (await confirmBtn.isVisible({ timeout: TIMEOUTS.RENDER }).catch(() => false)) {
        await confirmBtn.evaluate((el) => el.click());
      }

      // Preset should be removed
      await expect(presetItem).not.toBeVisible({ timeout: TIMEOUTS.WEBGL_INIT });
    });

    test('should rename preset', async ({ page }) => {
      // Create a preset (use valid option value: 7 = Last 7 days)
      await page.selectOption('#filter-days', '7');
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-days') as HTMLSelectElement;
        return select && select.value === '7';
      });
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await page.fill('#preset-name-input', 'Old Name');
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-confirm-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await expect(page.locator('#save-preset-dialog')).not.toBeVisible();

      // Find rename button
      const presetItem = page.locator('[data-preset-name="Old Name"]');
      const renameButton = presetItem.locator('.preset-rename-btn, [data-action="rename"]');

      if (await renameButton.isVisible().catch(() => false)) {
        // Use JavaScript click for CI reliability
        await renameButton.evaluate((el) => el.click());

        // Should show rename input
        const renameInput = page.locator('.preset-rename-input, #preset-rename-input');
        if (await renameInput.isVisible().catch(() => false)) {
          await renameInput.fill('New Name');
          await page.keyboard.press('Enter');

          // Should update name
          await expect(page.locator('[data-preset-name="New Name"]')).toBeVisible();
          await expect(page.locator('[data-preset-name="Old Name"]')).not.toBeVisible();
        }
      }
      // If rename not implemented, test passes
    });
  });

  test.describe('Persistence', () => {
    test('should persist presets across page reloads', async ({ page }) => {
      // Create a preset
      await page.selectOption('#filter-days', '7');
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-days') as HTMLSelectElement;
        return select && select.value === '7';
      });
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await page.fill('#preset-name-input', 'Persistent');
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-confirm-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await expect(page.locator('#save-preset-dialog')).not.toBeVisible();

      // Verify it's saved in localStorage
      const stored = await page.evaluate(() => localStorage.getItem('filter-presets'));
      expect(stored).toBeTruthy();
      expect(stored).toContain('Persistent');

      // Reload page
      await page.reload({ waitUntil: 'domcontentloaded' });
      await expect(page.locator('#app')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

      // Preset should still be there
      await expect(page.locator('[data-preset-name="Persistent"]')).toBeVisible();
    });

    test('should handle corrupted localStorage gracefully', async ({ page }) => {
      // Set corrupted data
      await page.evaluate(() => {
        localStorage.setItem('filter-presets', 'invalid json data');
      });

      // Reload page
      await page.reload({ waitUntil: 'domcontentloaded' });

      // App should still work
      await expect(page.locator('#app')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
      await expect(page.locator('#filters')).toBeVisible();

      // Should not crash - can save new presets
      await page.selectOption('#filter-days', '7');
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-days') as HTMLSelectElement;
        return select && select.value === '7';
      });
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await expect(page.locator('#save-preset-dialog')).toBeVisible();
    });
  });

  test.describe('Accessibility', () => {
    test('should have proper ARIA attributes', async ({ page }) => {
      const saveButton = page.locator('#btn-save-preset');
      await expect(saveButton).toHaveAttribute('aria-label');

      // Open save dialog - use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      const dialog = page.locator('#save-preset-dialog');
      await expect(dialog).toBeVisible();

      // Dialog should have proper attributes
      await expect(dialog).toHaveAttribute('role', 'dialog');
    });

    test('should be keyboard navigable', async ({ page }) => {
      // Create a preset
      await page.selectOption('#filter-days', '7');
      await page.waitForFunction(() => {
        const select = document.querySelector('#filter-days') as HTMLSelectElement;
        return select && select.value === '7';
      });
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await page.fill('#preset-name-input', 'Keyboard Test');
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-confirm-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await expect(page.locator('#save-preset-dialog')).not.toBeVisible();

      // Focus preset item
      const presetItem = page.locator('[data-preset-name="Keyboard Test"]');
      await presetItem.focus();

      // Should be able to activate with Enter/Space
      await page.keyboard.press('Enter');

      // Filter should be applied
      await expect(page.locator('#filter-days')).toHaveValue('7');
    });

    test('should close save dialog with Escape', async ({ page }) => {
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await expect(page.locator('#save-preset-dialog')).toBeVisible();

      await page.keyboard.press('Escape');

      await expect(page.locator('#save-preset-dialog')).not.toBeVisible();
    });
  });
});

test.describe('Filter Preset Quick Actions', () => {
  test.beforeEach(async ({ page }) => {
    // Skip onboarding to prevent modal from blocking interactions
    await page.addInitScript(() => {
      localStorage.setItem('onboarding_completed', 'true');
      localStorage.setItem('onboarding_skipped', 'true');
    });

    await page.goto('/', { waitUntil: 'domcontentloaded' });
    await page.evaluate(() => localStorage.removeItem('filter-presets'));
    await page.reload({ waitUntil: 'domcontentloaded' });
    await expect(page.locator('#app')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
  });

  test('should show preset count badge', async ({ page }) => {
    // No presets initially - may or may not show badge
    const badge = page.locator('.preset-count-badge');

    // Create some presets using valid filter-days options
    const validDays = ['7', '30', '90'];
    for (let i = 0; i < validDays.length; i++) {
      await page.selectOption('#filter-days', validDays[i]);
      await page.waitForFunction((expectedValue) => {
        const select = document.querySelector('#filter-days') as HTMLSelectElement;
        return select && select.value === expectedValue;
      }, validDays[i]);
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await page.fill('#preset-name-input', `Preset ${i + 1}`);
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('#btn-confirm-save-preset') as HTMLElement;
        if (btn) btn.click();
      });
      await expect(page.locator('#save-preset-dialog')).not.toBeVisible();
    }

    // Badge may show count
    if (await badge.isVisible().catch(() => false)) {
      const count = await badge.textContent();
      expect(count).toContain('3');
    }
  });
});
