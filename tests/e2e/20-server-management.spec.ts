// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Test: Server Management UI (ADR-0026 Phase 5)
 *
 * Tests the multi-server management interface:
 * - Server list display
 * - Add server flow
 * - Edit server flow
 * - Delete server flow
 * - Toggle server enable/disable
 * - RBAC integration
 * - Connection testing
 *
 * NOTE: All API requests are automatically mocked via fixtures.ts
 * The mock server provides deterministic data for all server management endpoints.
 */

import {
  test,
  expect,
  TIMEOUTS,
  gotoAppAndWaitReady,
} from './fixtures';

/**
 * Helper to navigate to server management section.
 * The MultiServerManager is initialized on the server tab.
 */
async function navigateToServerTab(page: import('@playwright/test').Page): Promise<void> {
  // Navigate using JavaScript click for CI reliability
  await page.evaluate(() => {
    const btn = document.querySelector('button[data-view="server"]') as HTMLElement;
    if (btn) btn.click();
  });
  await expect(page.locator('#server-container')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
}

/**
 * Helper to wait for the server management panel to be ready
 */
async function waitForServerManagementReady(page: import('@playwright/test').Page): Promise<void> {
  // Wait for the multi-server panel to render
  await page.waitForFunction(
    () => {
      const panel = document.querySelector('.multi-server-panel');
      const list = document.querySelector('[data-testid="server-list"], [data-testid="server-list-loading"], [data-testid="server-list-empty"]');
      return panel && list;
    },
    { timeout: TIMEOUTS.DEFAULT }
  );
}

test.describe('Server Management UI', () => {
  test.beforeEach(async ({ page }) => {
    await gotoAppAndWaitReady(page);
  });

  test.describe('Server List Display', () => {
    test('should display server management panel on server tab', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // The multi-server panel should be visible
      await expect(page.locator('.multi-server-panel')).toBeVisible();
      await expect(page.locator('.multi-server-header')).toBeVisible();
    });

    test('should display correct server count in header', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Check for server count badge
      const countBadge = page.locator('[data-testid="server-count"]');
      await expect(countBadge).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
      const countText = await countBadge.textContent();
      expect(countText).toContain('servers');
    });

    test('should display status summary cards', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Status summary should show connected, syncing, and error counts
      const statusSummary = page.locator('[data-testid="status-summary"]');
      await expect(statusSummary).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      await expect(page.locator('[data-testid="connected-count"]')).toBeVisible();
      await expect(page.locator('[data-testid="syncing-count"]')).toBeVisible();
      await expect(page.locator('[data-testid="error-count"]')).toBeVisible();
    });

    test('should display server list with server cards', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Server list should be visible
      const serverList = page.locator('[data-testid="server-list"]');
      await expect(serverList).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Should have at least one server card
      const serverCards = page.locator('.server-card');
      const cardCount = await serverCards.count();
      expect(cardCount).toBeGreaterThan(0);
    });

    test('should display server name and URL on each card', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Get first server card
      const firstCard = page.locator('.server-card').first();
      await expect(firstCard).toBeVisible();

      // Should have server name
      const serverName = firstCard.locator('[data-testid="server-name"]');
      await expect(serverName).toBeVisible();
      const nameText = await serverName.textContent();
      expect(nameText).toBeTruthy();

      // Should have server URL
      const serverUrl = firstCard.locator('[data-testid="server-url"]');
      await expect(serverUrl).toBeVisible();
      const urlText = await serverUrl.textContent();
      expect(urlText).toBeTruthy();
    });

    test('should display status indicator on server cards', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // First server card should have status indicator
      const firstCard = page.locator('.server-card').first();
      const statusIndicator = firstCard.locator('[data-testid="status-indicator"]');
      await expect(statusIndicator).toBeVisible();

      const statusText = firstCard.locator('[data-testid="status-text"]');
      await expect(statusText).toBeVisible();
    });

    test('should display source badge (ENV or UI) on server cards', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Should have at least one server with env source badge
      const envBadge = page.locator('[data-testid="source-env"]');
      const envCount = await envBadge.count();

      // Should have at least one server with UI source badge
      const uiBadge = page.locator('[data-testid="source-ui"]');
      const uiCount = await uiBadge.count();

      // At least one source badge should be visible
      expect(envCount + uiCount).toBeGreaterThan(0);
    });

    test('should not show edit/delete buttons for immutable (env) servers', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Find a server with env source
      const envServer = page.locator('.server-card:has([data-testid="source-env"])').first();
      const hasEnvServer = await envServer.count() > 0;

      if (hasEnvServer) {
        // Env servers should not have edit button
        const editBtn = envServer.locator('[data-testid="btn-edit"]');
        await expect(editBtn).not.toBeVisible();

        // Env servers should not have delete button
        const deleteBtn = envServer.locator('[data-testid="btn-delete"]');
        await expect(deleteBtn).not.toBeVisible();
      }
    });

    test('should show edit/delete buttons for UI-added servers', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Find a server with UI source
      const uiServer = page.locator('.server-card:has([data-testid="source-ui"])').first();
      const hasUiServer = await uiServer.count() > 0;

      if (hasUiServer) {
        // UI servers should have edit button
        const editBtn = uiServer.locator('[data-testid="btn-edit"]');
        await expect(editBtn).toBeVisible();

        // UI servers should have delete button
        const deleteBtn = uiServer.locator('[data-testid="btn-delete"]');
        await expect(deleteBtn).toBeVisible();
      }
    });

    test('should display last sync time for servers', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // At least one server should show last sync time
      const lastSyncElements = page.locator('[data-testid="last-sync"]');
      const count = await lastSyncElements.count();

      // If any servers have been synced, they should show the time
      if (count > 0) {
        const firstSync = lastSyncElements.first();
        await expect(firstSync).toBeVisible();
      }
    });

    test('should display error message for servers with errors', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Find server with error status (mock data has one)
      const errorServer = page.locator('.server-card:has([data-testid="server-error"])').first();
      const hasErrorServer = await errorServer.count() > 0;

      if (hasErrorServer) {
        const errorElement = errorServer.locator('[data-testid="server-error"]');
        await expect(errorElement).toBeVisible();
        const errorText = await errorElement.textContent();
        expect(errorText).toBeTruthy();
      }
    });
  });

  test.describe('Add Server Flow', () => {
    test('should display Add Server button', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      const addBtn = page.locator('[data-testid="add-server"]');
      await expect(addBtn).toBeVisible();
    });

    test('should open modal when Add Server is clicked', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Click add server button
      await page.evaluate(() => {
        const btn = document.querySelector('[data-testid="add-server"]') as HTMLElement;
        if (btn) btn.click();
      });

      // Modal should be visible
      const modal = page.locator('[data-testid="server-modal"]');
      await expect(modal).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test('should display form fields in add modal', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Open add modal
      await page.evaluate(() => {
        const btn = document.querySelector('[data-testid="add-server"]') as HTMLElement;
        if (btn) btn.click();
      });

      await expect(page.locator('[data-testid="server-modal"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Check for form fields
      await expect(page.locator('[data-testid="select-platform"]')).toBeVisible();
      await expect(page.locator('[data-testid="input-name"]')).toBeVisible();
      await expect(page.locator('[data-testid="input-url"]')).toBeVisible();
      await expect(page.locator('[data-testid="input-token"]')).toBeVisible();
    });

    test('should have platform selection options', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Open add modal
      await page.evaluate(() => {
        const btn = document.querySelector('[data-testid="add-server"]') as HTMLElement;
        if (btn) btn.click();
      });

      await expect(page.locator('[data-testid="server-modal"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Check platform options
      const platformSelect = page.locator('[data-testid="select-platform"]');
      const options = platformSelect.locator('option');
      const optionCount = await options.count();

      // Should have at least Plex, Jellyfin, Emby, Tautulli + placeholder
      expect(optionCount).toBeGreaterThanOrEqual(5);
    });

    test('should display Test Connection button', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Open add modal
      await page.evaluate(() => {
        const btn = document.querySelector('[data-testid="add-server"]') as HTMLElement;
        if (btn) btn.click();
      });

      await expect(page.locator('[data-testid="server-modal"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      const testBtn = page.locator('[data-testid="btn-test-connection"]');
      await expect(testBtn).toBeVisible();
    });

    test('should close modal when Cancel is clicked', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Open add modal
      await page.evaluate(() => {
        const btn = document.querySelector('[data-testid="add-server"]') as HTMLElement;
        if (btn) btn.click();
      });

      await expect(page.locator('[data-testid="server-modal"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Click cancel
      await page.evaluate(() => {
        const btn = document.querySelector('[data-testid="btn-cancel"]') as HTMLElement;
        if (btn) btn.click();
      });

      // Modal should be hidden
      await expect(page.locator('[data-testid="server-modal"]')).not.toBeVisible();
    });

    test('should validate required fields before submission', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Open add modal
      await page.evaluate(() => {
        const btn = document.querySelector('[data-testid="add-server"]') as HTMLElement;
        if (btn) btn.click();
      });

      await expect(page.locator('[data-testid="server-modal"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Try to submit without filling fields
      const nameInput = page.locator('[data-testid="input-name"]');
      await expect(nameInput).toHaveAttribute('required');

      const urlInput = page.locator('[data-testid="input-url"]');
      await expect(urlInput).toHaveAttribute('required');

      const platformSelect = page.locator('[data-testid="select-platform"]');
      await expect(platformSelect).toHaveAttribute('required');
    });

    test('should display checkbox options for server features', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Open add modal
      await page.evaluate(() => {
        const btn = document.querySelector('[data-testid="add-server"]') as HTMLElement;
        if (btn) btn.click();
      });

      await expect(page.locator('[data-testid="server-modal"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Check for feature checkboxes
      await expect(page.locator('[data-testid="checkbox-realtime"]')).toBeVisible();
      await expect(page.locator('[data-testid="checkbox-webhooks"]')).toBeVisible();
      await expect(page.locator('[data-testid="checkbox-polling"]')).toBeVisible();
    });

    test('should display polling interval selection', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Open add modal
      await page.evaluate(() => {
        const btn = document.querySelector('[data-testid="add-server"]') as HTMLElement;
        if (btn) btn.click();
      });

      await expect(page.locator('[data-testid="server-modal"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      const pollingSelect = page.locator('[data-testid="select-polling-interval"]');
      await expect(pollingSelect).toBeVisible();
    });
  });

  test.describe('Connection Testing', () => {
    test('should show test result after connection test', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Open add modal
      await page.evaluate(() => {
        const btn = document.querySelector('[data-testid="add-server"]') as HTMLElement;
        if (btn) btn.click();
      });

      await expect(page.locator('[data-testid="server-modal"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Fill required fields
      await page.selectOption('[data-testid="select-platform"]', 'plex');
      await page.fill('[data-testid="input-name"]', 'Test Server');
      await page.fill('[data-testid="input-url"]', 'http://localhost:32400');
      await page.fill('[data-testid="input-token"]', 'test-token-12345678');

      // Click test connection
      await page.evaluate(() => {
        const btn = document.querySelector('[data-testid="btn-test-connection"]') as HTMLElement;
        if (btn) btn.click();
      });

      // Wait for test result to appear
      const testResult = page.locator('[data-testid="test-result"]');
      await expect(testResult).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });
  });

  test.describe('Edit Server Flow', () => {
    test('should open edit modal with pre-filled data', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Find a UI server and click edit
      const uiServer = page.locator('.server-card:has([data-testid="source-ui"])').first();
      const hasUiServer = await uiServer.count() > 0;

      if (hasUiServer) {
        const editBtn = uiServer.locator('[data-testid="btn-edit"]');
        await expect(editBtn).toBeVisible();

        await editBtn.click();

        // Modal should open
        const modal = page.locator('[data-testid="server-modal"]');
        await expect(modal).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

        // Name field should have a value
        const nameInput = page.locator('[data-testid="input-name"]');
        const nameValue = await nameInput.inputValue();
        expect(nameValue).toBeTruthy();
      }
    });

    test('should have platform select disabled in edit mode', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Find a UI server and click edit
      const uiServer = page.locator('.server-card:has([data-testid="source-ui"])').first();
      const hasUiServer = await uiServer.count() > 0;

      if (hasUiServer) {
        const editBtn = uiServer.locator('[data-testid="btn-edit"]');
        await editBtn.click();

        await expect(page.locator('[data-testid="server-modal"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

        // Platform should be disabled in edit mode
        const platformSelect = page.locator('[data-testid="select-platform"]');
        await expect(platformSelect).toBeDisabled();
      }
    });
  });

  test.describe('Delete Server Flow', () => {
    test('should open delete confirmation modal', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Find a UI server and click delete
      const uiServer = page.locator('.server-card:has([data-testid="source-ui"])').first();
      const hasUiServer = await uiServer.count() > 0;

      if (hasUiServer) {
        const deleteBtn = uiServer.locator('[data-testid="btn-delete"]');
        await expect(deleteBtn).toBeVisible();

        await deleteBtn.click();

        // Delete confirmation modal should open
        const deleteModal = page.locator('[data-testid="delete-modal"]');
        await expect(deleteModal).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
      }
    });

    test('should display server name in delete confirmation', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Find a UI server and click delete
      const uiServer = page.locator('.server-card:has([data-testid="source-ui"])').first();
      const hasUiServer = await uiServer.count() > 0;

      if (hasUiServer) {
        const deleteBtn = uiServer.locator('[data-testid="btn-delete"]');
        await deleteBtn.click();

        const deleteModal = page.locator('[data-testid="delete-modal"]');
        await expect(deleteModal).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

        // Server name should be displayed
        const serverNameEl = page.locator('[data-testid="delete-server-name"]');
        await expect(serverNameEl).toBeVisible();
        const serverName = await serverNameEl.textContent();
        expect(serverName).toBeTruthy();
      }
    });

    test('should close delete modal when Cancel is clicked', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Find a UI server and click delete
      const uiServer = page.locator('.server-card:has([data-testid="source-ui"])').first();
      const hasUiServer = await uiServer.count() > 0;

      if (hasUiServer) {
        const deleteBtn = uiServer.locator('[data-testid="btn-delete"]');
        await deleteBtn.click();

        await expect(page.locator('[data-testid="delete-modal"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

        // Click cancel
        await page.evaluate(() => {
          const btn = document.querySelector('[data-testid="btn-cancel-delete"]') as HTMLElement;
          if (btn) btn.click();
        });

        // Modal should be hidden
        await expect(page.locator('[data-testid="delete-modal"]')).not.toBeVisible();
      }
    });
  });

  test.describe('Toggle Server Enable/Disable', () => {
    test('should display toggle switch for UI servers', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Find a UI server
      const uiServer = page.locator('.server-card:has([data-testid="source-ui"])').first();
      const hasUiServer = await uiServer.count() > 0;

      if (hasUiServer) {
        const toggle = uiServer.locator('[data-testid="toggle-enabled"]');
        await expect(toggle).toBeVisible();
      }
    });

    test('should not display toggle switch for env servers', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Find an env server
      const envServer = page.locator('.server-card:has([data-testid="source-env"])').first();
      const hasEnvServer = await envServer.count() > 0;

      if (hasEnvServer) {
        // Env servers should not have toggle
        const toggle = envServer.locator('[data-testid="toggle-enabled"]');
        await expect(toggle).not.toBeVisible();
      }
    });
  });

  test.describe('Refresh Functionality', () => {
    test('should display refresh button', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      const refreshBtn = page.locator('[data-testid="refresh-servers"]');
      await expect(refreshBtn).toBeVisible();
    });

    test('should reload servers when refresh is clicked', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Get initial server count
      const initialCards = await page.locator('.server-card').count();

      // Click refresh
      await page.evaluate(() => {
        const btn = document.querySelector('[data-testid="refresh-servers"]') as HTMLElement;
        if (btn) btn.click();
      });

      // DETERMINISTIC: Wait for server list to be updated (network + DOM update)
      await page.waitForLoadState('networkidle', { timeout: TIMEOUTS.MEDIUM }).catch(() => {});

      // Should still have servers
      const finalCards = await page.locator('.server-card').count();
      expect(finalCards).toBeGreaterThan(0);
    });
  });

  test.describe('Sync Trigger', () => {
    test('should display sync button on server cards', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // First server card should have sync button
      const firstCard = page.locator('.server-card').first();
      const syncBtn = firstCard.locator('[data-testid="btn-sync"]');
      await expect(syncBtn).toBeVisible();
    });
  });

  test.describe('Responsive Layout', () => {
    test('should display server cards stacked on mobile', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Switch to mobile viewport
      await page.setViewportSize({ width: 375, height: 667 });
      // DETERMINISTIC: Wait for server list to be visible (layout stabilized)
      await expect(page.locator('[data-testid="server-list"]')).toBeVisible({ timeout: TIMEOUTS.ANIMATION });

      // Cards should stack (check that container is narrower)
      const serverList = page.locator('[data-testid="server-list"]');
      const box = await serverList.boundingBox();
      expect(box).not.toBeNull();
      expect(box!.width).toBeLessThan(400);
    });

    test('should maintain header on mobile', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Switch to mobile viewport
      await page.setViewportSize({ width: 375, height: 667 });
      // DETERMINISTIC: Wait for header to be visible (layout stabilized)
      await expect(page.locator('.multi-server-header')).toBeVisible({ timeout: TIMEOUTS.ANIMATION });
    });
  });

  test.describe('Accessibility', () => {
    test('should have proper aria labels on buttons', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Add server button should have aria-label
      const addBtn = page.locator('[data-testid="add-server"]');
      await expect(addBtn).toHaveAttribute('aria-label');

      // Refresh button should have aria-label
      const refreshBtn = page.locator('[data-testid="refresh-servers"]');
      await expect(refreshBtn).toHaveAttribute('aria-label');
    });

    test('should have role attributes on server list', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Server list should have role="list"
      const serverList = page.locator('[data-testid="server-list"]');
      const isVisible = await serverList.isVisible().catch(() => false);

      if (isVisible) {
        await expect(serverList).toHaveAttribute('role', 'list');
      }
    });

    test('should have role listitem on server cards', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // First server card should have role="listitem"
      const firstCard = page.locator('.server-card').first();
      const isVisible = await firstCard.isVisible().catch(() => false);

      if (isVisible) {
        await expect(firstCard).toHaveAttribute('role', 'listitem');
      }
    });

    test('should have modal with aria-modal', async ({ page }) => {
      await navigateToServerTab(page);
      await waitForServerManagementReady(page);

      // Open add modal
      await page.evaluate(() => {
        const btn = document.querySelector('[data-testid="add-server"]') as HTMLElement;
        if (btn) btn.click();
      });

      const modal = page.locator('[data-testid="server-modal"]');
      await expect(modal).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
      await expect(modal).toHaveAttribute('aria-modal', 'true');
    });
  });
});
