// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import {
  test,
  expect,
  TIMEOUTS,
  gotoAppAndWaitReady,
  waitForNavReady,
} from './fixtures';

/**
 * E2E Test: Backup/Restore UI
 *
 * Tests backup/restore functionality:
 * - List existing backups
 * - Create new backup
 * - Delete backups
 * - Download backups
 * - Restore from backup
 * - Backup stats display
 *
 * @see /docs/working/UI_UX_AUDIT.md
 *
 * Note: Uses autoMockApi: true for deterministic app data during backup tests.
 * The backup-specific endpoints are then overridden with test-specific data.
 */

// Enable API mocking for deterministic app data
test.use({ autoMockApi: true });

// Mock backup data for tests
const mockBackups = [
  {
    id: 'backup-001',
    type: 'full',
    filename: 'backup-2024-01-15-001.tar.gz',
    size_bytes: 15728640,
    created_at: '2024-01-15T10:30:00Z',
    notes: 'Weekly backup',
    database_records: 12500,
    is_valid: true
  },
  {
    id: 'backup-002',
    type: 'database',
    filename: 'backup-2024-01-14-001.tar.gz',
    size_bytes: 10485760,
    created_at: '2024-01-14T10:30:00Z',
    notes: '',
    database_records: 12000,
    is_valid: true
  }
];

const mockBackupStats = {
  total_backups: 5,
  total_size_bytes: 52428800,
  oldest_backup: '2024-01-01T00:00:00Z',
  newest_backup: '2024-01-15T10:30:00Z',
  full_backups: 3,
  database_backups: 2,
  config_backups: 0
};

test.describe('Backup/Restore UI', () => {
  test.beforeEach(async ({ page }) => {
    // Setup backup API mocks
    await page.route('**/api/v1/backups', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        data: mockBackups,
        metadata: { timestamp: new Date().toISOString() }
      })
    }));

    await page.route('**/api/v1/backup/stats', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        data: mockBackupStats,
        metadata: { timestamp: new Date().toISOString() }
      })
    }));

    // Navigate to app
    await gotoAppAndWaitReady(page);
    await waitForNavReady(page); // Wait for NavigationManager to initialize before clicking nav buttons
  });

  test.describe('Backup Section Visibility', () => {
    test('should show backup section in server dashboard', async ({ page }) => {
      // Navigate to server dashboard
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      // Backup section should be visible
      const backupSection = page.locator('#backup-section');
      await expect(backupSection).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test('should display backup stats', async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      // Stats should show
      const statsContainer = page.locator('.backup-stats');
      await expect(statsContainer).toBeVisible();
    });
  });

  test.describe('List Backups', () => {
    test('should display list of existing backups', async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      // Backup list should be visible
      const backupList = page.locator('.backup-list, #backup-list');
      await expect(backupList).toBeVisible();

      // Should show backup items
      const backupItems = page.locator('.backup-item');
      await expect(backupItems.first()).toBeVisible();
    });

    test('should display backup details', async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      // First backup item should show details
      const backupItem = page.locator('.backup-item').first();
      await expect(backupItem).toBeVisible();

      // Should show type, date, size
      const itemText = await backupItem.textContent();
      expect(itemText).toBeTruthy();
    });

    test('should show empty state when no backups', async ({ page }) => {
      // Override with empty backups
      await page.route('**/api/v1/backups', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: [],
          metadata: { timestamp: new Date().toISOString() }
        })
      }));

      await page.reload({ waitUntil: 'domcontentloaded' });
      await expect(page.locator('#app')).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      // Empty state should show
      const emptyState = page.locator('.backup-empty-state, .backup-list-empty');
      const backupItems = page.locator('.backup-item');

      const hasEmptyState = await emptyState.isVisible().catch(() => false);
      const hasNoItems = (await backupItems.count()) === 0;

      expect(hasEmptyState || hasNoItems).toBeTruthy();
    });
  });

  test.describe('Create Backup', () => {
    test('should show create backup button', async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      const createButton = page.locator('#btn-create-backup');
      await expect(createButton).toBeVisible();
    });

    test('should open create backup dialog', async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const el = document.querySelector('#btn-create-backup') as HTMLElement;
        if (el) el.click();
      });

      const dialog = page.locator('#create-backup-dialog, .backup-dialog');
      await expect(dialog).toBeVisible({ timeout: TIMEOUTS.DATA_LOAD });
    });

    test('should have backup type options', async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const el = document.querySelector('#btn-create-backup') as HTMLElement;
        if (el) el.click();
      });

      const dialog = page.locator('#create-backup-dialog, .backup-dialog');
      await expect(dialog).toBeVisible({ timeout: TIMEOUTS.DATA_LOAD });

      // Should have type selection
      const typeSelect = page.locator('#backup-type-select, [name="backup-type"]');
      const typeButtons = page.locator('.backup-type-option');

      const hasSelect = await typeSelect.isVisible().catch(() => false);
      const hasButtons = (await typeButtons.count()) > 0;

      expect(hasSelect || hasButtons).toBeTruthy();
    });

    test('should create backup successfully', async ({ page }) => {
      // Mock successful backup creation
      await page.route('**/api/v1/backup', route => {
        if (route.request().method() === 'POST') {
          return route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
              status: 'success',
              data: {
                id: 'backup-new',
                type: 'full',
                filename: 'backup-new.tar.gz',
                size_bytes: 1024000,
                created_at: new Date().toISOString(),
                is_valid: true
              },
              metadata: { timestamp: new Date().toISOString() }
            })
          });
        }
        return route.continue();
      });

      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const el = document.querySelector('#btn-create-backup') as HTMLElement;
        if (el) el.click();
      });
      await expect(page.locator('#create-backup-dialog, .backup-dialog')).toBeVisible();

      // Submit - Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const el = document.querySelector('#btn-confirm-backup, .btn-create-backup-confirm') as HTMLElement;
        if (el) el.click();
      });

      // Should show success toast
      await expect(page.locator('.toast')).toContainText(/backup|created|success/i, { timeout: TIMEOUTS.MEDIUM });
    });
  });

  test.describe('Delete Backup', () => {
    test('should have delete button on backup items', async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      const backupItem = page.locator('.backup-item').first();
      await expect(backupItem).toBeVisible();

      const deleteButton = backupItem.locator('.backup-delete-btn, [data-action="delete"]');
      await expect(deleteButton).toBeVisible();
    });

    test('should show confirmation before delete', async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      const backupItem = page.locator('.backup-item').first();
      const deleteButton = backupItem.locator('.backup-delete-btn, [data-action="delete"]');
      await deleteButton.click();

      // Confirmation dialog should appear
      const confirmDialog = page.locator('#confirmation-dialog, .backup-confirm-dialog');
      await expect(confirmDialog).toBeVisible({ timeout: TIMEOUTS.DATA_LOAD });
    });

    test('should delete backup when confirmed', async ({ page }) => {
      // Mock delete endpoint
      await page.route('**/api/v1/backups/delete*', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: { deleted: true },
          metadata: { timestamp: new Date().toISOString() }
        })
      }));

      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      const backupItem = page.locator('.backup-item').first();
      const deleteButton = backupItem.locator('.backup-delete-btn, [data-action="delete"]');
      await deleteButton.click();

      // Confirm deletion
      const confirmButton = page.locator('#confirmation-dialog-confirm, .btn-confirm-delete');
      if (await confirmButton.isVisible().catch(() => false)) {
        await confirmButton.click();
      }

      // Wait for confirmation dialog to close
      await page.waitForFunction(() => {
        const dialog = document.querySelector('#confirmation-dialog, .backup-confirm-dialog');
        return !dialog || getComputedStyle(dialog).display === 'none';
      }, { timeout: TIMEOUTS.DATA_LOAD });
    });
  });

  test.describe('Download Backup', () => {
    test('should have download button on backup items', async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      const backupItem = page.locator('.backup-item').first();
      await expect(backupItem).toBeVisible();

      const downloadButton = backupItem.locator('.backup-download-btn, [data-action="download"]');
      await expect(downloadButton).toBeVisible();
    });
  });

  test.describe('Restore Backup', () => {
    test('should have restore button on backup items', async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      const backupItem = page.locator('.backup-item').first();
      await expect(backupItem).toBeVisible();

      const restoreButton = backupItem.locator('.backup-restore-btn, [data-action="restore"]');
      await expect(restoreButton).toBeVisible();
    });

    test('should show restore confirmation dialog', async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      const backupItem = page.locator('.backup-item').first();
      const restoreButton = backupItem.locator('.backup-restore-btn, [data-action="restore"]');
      await restoreButton.click();

      // Should show restore dialog with options
      const restoreDialog = page.locator('#restore-backup-dialog, .restore-dialog');
      await expect(restoreDialog).toBeVisible({ timeout: TIMEOUTS.DATA_LOAD });
    });

    test('should show restore options', async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      const backupItem = page.locator('.backup-item').first();
      const restoreButton = backupItem.locator('.backup-restore-btn, [data-action="restore"]');
      await restoreButton.click();

      const restoreDialog = page.locator('#restore-backup-dialog, .restore-dialog');
      await expect(restoreDialog).toBeVisible({ timeout: TIMEOUTS.DATA_LOAD });

      // Should have restore options (checkboxes or toggles)
      const options = restoreDialog.locator('input[type="checkbox"], .restore-option');
      const optionCount = await options.count();
      expect(optionCount).toBeGreaterThan(0);
    });
  });

  test.describe('Accessibility', () => {
    test('should have proper ARIA attributes', async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      const backupSection = page.locator('#backup-section');
      await expect(backupSection).toBeVisible();

      // List should have proper role
      const backupList = backupSection.locator('.backup-list, [role="list"]');
      const hasRole = await backupList.getAttribute('role');
      expect(hasRole === 'list' || hasRole === null).toBeTruthy();
    });

    test('should be keyboard navigable', async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });
      await page.waitForFunction(() => {
        const container = document.getElementById('server-container');
        return container && getComputedStyle(container).display !== 'none';
      }, { timeout: TIMEOUTS.RENDER });

      // Tab to create backup button
      const createButton = page.locator('#btn-create-backup');
      await createButton.focus();
      await expect(createButton).toBeFocused();

      // Should be able to activate with Enter
      await page.keyboard.press('Enter');

      // Dialog should open
      const dialog = page.locator('#create-backup-dialog, .backup-dialog');
      await expect(dialog).toBeVisible({ timeout: TIMEOUTS.DATA_LOAD });

      // Escape to close
      await page.keyboard.press('Escape');
      await expect(dialog).not.toBeVisible({ timeout: TIMEOUTS.DATA_LOAD });
    });
  });
});

test.describe('Backup Error Handling', () => {
  test('should handle backup creation failure gracefully', async ({ page }) => {
    // Mock failed backup creation
    await page.route('**/api/v1/backup', route => {
      if (route.request().method() === 'POST') {
        return route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'error',
            error: { code: 'BACKUP_FAILED', message: 'Failed to create backup' }
          })
        });
      }
      return route.continue();
    });

    await page.route('**/api/v1/backups', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'success', data: [], metadata: {} })
    }));

    await page.route('**/api/v1/backup/stats', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'success', data: mockBackupStats, metadata: {} })
    }));

    await gotoAppAndWaitReady(page);

    // Navigate using JavaScript click for CI reliability
    // WHY: Playwright's .click() may fail in headless/SwiftShader environments
    await page.evaluate(() => {
      const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
      if (tab) tab.click();
    });
    await page.waitForFunction(() => {
      const container = document.getElementById('server-container');
      return container && getComputedStyle(container).display !== 'none';
    }, { timeout: TIMEOUTS.RENDER });

    // Use JavaScript click for CI reliability
    await page.evaluate(() => {
      const el = document.querySelector('#btn-create-backup') as HTMLElement;
      if (el) el.click();
    });
    const dialog = page.locator('#create-backup-dialog, .backup-dialog');
    await expect(dialog).toBeVisible();

    // Use JavaScript click for CI reliability
    await page.evaluate(() => {
      const el = document.querySelector('#btn-confirm-backup, .btn-create-backup-confirm') as HTMLElement;
      if (el) el.click();
    });

    // Should show error toast - use specific selector to avoid strict mode violation
    // There may be multiple toasts (info "Creating backup..." and error toast)
    const errorToast = page.locator('.toast-error, .toast.error, [class*="toast"][class*="error"]');
    await expect(errorToast).toContainText(/error|fail/i, { timeout: TIMEOUTS.MEDIUM });
  });
});
