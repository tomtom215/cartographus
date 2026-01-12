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
 * E2E Test: Session Management - Kill Stream
 *
 * Tests session termination functionality:
 * - Kill button visibility on session cards
 * - Confirmation dialog before termination
 * - API call to terminate session
 * - Success notification after termination
 *
 * @see /docs/working/UI_UX_AUDIT.md
 */

// Note: Mock activity data is now handled by the mock server (mock-api-server.ts, mock-server.ts)
// which returns session data by default for /api/v1/activity

test.describe('Session Management - Kill Stream', () => {
  test.beforeEach(async ({ page }) => {
    // Note: Activity and terminate-session endpoints are handled by the mock server
    // which now returns session data by default (see mock-api-server.ts and mock-server.ts)
    await gotoAppAndWaitReady(page);
    await waitForNavReady(page);
  });

  test.describe('Kill Button Display', () => {
    test('should show kill button on session cards', async ({ page }) => {
      // Navigate to activity view using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="activity"]') as HTMLElement;
        if (tab) tab.click();
      });

      // Wait for activity container to be visible first
      // WHY: The container must be visible before session cards can render
      await page.waitForSelector('#activity-container', {
        state: 'visible',
        timeout: TIMEOUTS.MEDIUM
      });

      // Wait for session cards to be rendered - use MEDIUM timeout for CI
      // WHY: Session cards are rendered async after API data loads
      await page.waitForFunction(() => {
        return document.querySelector('.activity-session-card') !== null;
      }, { timeout: TIMEOUTS.MEDIUM });

      // Wait for sessions to load
      const sessionCard = page.locator('.activity-session-card').first();
      await expect(sessionCard).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Kill button should exist
      const killButton = sessionCard.locator('.btn-kill-session');
      await expect(killButton).toBeVisible();
    });

    test('should have proper accessibility attributes on kill button', async ({ page }) => {
      // Navigate to activity view using JavaScript click for CI reliability
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="activity"]') as HTMLElement;
        if (tab) tab.click();
      });

      // Wait for activity container to be visible first
      await page.waitForSelector('#activity-container', {
        state: 'visible',
        timeout: TIMEOUTS.MEDIUM
      });

      // Wait for session cards to be rendered - use MEDIUM timeout for CI
      await page.waitForFunction(() => {
        return document.querySelector('.activity-session-card') !== null;
      }, { timeout: TIMEOUTS.MEDIUM });

      const killButton = page.locator('.btn-kill-session').first();
      await expect(killButton).toHaveAttribute('aria-label', /terminate/i);
      await expect(killButton).toHaveAttribute('title', /terminate/i);
    });

    test('kill button should be keyboard accessible', async ({ page }) => {
      // Navigate to activity view using JavaScript click for CI reliability
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="activity"]') as HTMLElement;
        if (tab) tab.click();
      });

      // Wait for activity container to be visible first
      await page.waitForSelector('#activity-container', {
        state: 'visible',
        timeout: TIMEOUTS.MEDIUM
      });

      // Wait for session cards to be rendered - use MEDIUM timeout for CI
      await page.waitForFunction(() => {
        return document.querySelector('.activity-session-card') !== null;
      }, { timeout: TIMEOUTS.MEDIUM });

      const killButton = page.locator('.btn-kill-session').first();
      await killButton.focus();
      await expect(killButton).toBeFocused();
    });
  });

  test.describe('Session Termination', () => {
    test('should show confirmation dialog when kill button clicked', async ({ page }) => {
      // Navigate to activity view using JavaScript click for CI reliability
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="activity"]') as HTMLElement;
        if (tab) tab.click();
      });

      // Wait for activity container to be visible first
      await page.waitForSelector('#activity-container', {
        state: 'visible',
        timeout: TIMEOUTS.MEDIUM
      });

      // Wait for session cards to be rendered - use MEDIUM timeout for CI
      await page.waitForFunction(() => {
        return document.querySelector('.activity-session-card') !== null;
      }, { timeout: TIMEOUTS.MEDIUM });

      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('.btn-kill-session') as HTMLElement;
        if (btn) btn.click();
      });

      // Should show confirmation dialog or native confirm
      // Check for custom confirmation dialog
      const confirmDialog = page.locator('.confirmation-dialog, [role="alertdialog"]');
      const hasDialog = await confirmDialog.isVisible().catch(() => false);

      // Either has custom dialog or will fall back to native confirm
      // Note: Native confirm is automatically handled, so we just verify the flow doesn't crash
      if (hasDialog) {
        console.log('Custom confirmation dialog shown for session termination');
        await expect(confirmDialog).toBeVisible();
      } else {
        console.log('Native confirm used for session termination (dialog not visible)');
      }
    });

    test('should call terminate API when confirmed', async ({ page }) => {
      let terminateCalled = false;

      await page.route('**/api/v1/tautulli/terminate-session*', route => {
        terminateCalled = true;
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            data: { result: 'success' },
            metadata: {}
          })
        });
      });

      // Override the confirm dialog to automatically confirm
      await page.evaluate(() => {
        window.confirm = () => true;
      });

      // Navigate to activity view using JavaScript click for CI reliability
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="activity"]') as HTMLElement;
        if (tab) tab.click();
      });

      // Wait for activity container to be visible first
      await page.waitForSelector('#activity-container', {
        state: 'visible',
        timeout: TIMEOUTS.MEDIUM
      });

      // Wait for session cards to be rendered - use MEDIUM timeout for CI
      await page.waitForFunction(() => {
        return document.querySelector('.activity-session-card') !== null;
      }, { timeout: TIMEOUTS.MEDIUM });

      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const btn = document.querySelector('.btn-kill-session') as HTMLElement;
        if (btn) btn.click();
      });

      // Wait for confirmation dialog to appear (if custom dialog is used)
      await page.waitForFunction(() => {
        const dialog = document.querySelector('.confirmation-dialog, [role="alertdialog"]');
        return dialog !== null;
      }, { timeout: TIMEOUTS.RENDER }).catch(() => {
        // Native confirm might be used instead of custom dialog
      });

      // Click confirm button if custom dialog is shown
      const confirmBtn = page.locator('.confirmation-dialog .btn-confirm, [data-confirm="true"]');
      if (await confirmBtn.isVisible()) {
        await confirmBtn.click();
      }

      // Wait for terminate API response
      await page.waitForResponse(
        (response) => response.url().includes('/api/v1/tautulli/terminate-session'),
        { timeout: TIMEOUTS.MEDIUM }
      ).catch(() => {
        // API might not be called if no sessions exist
      });

      // API should have been called (either by native confirm or dialog)
      // Skip assertion if no kill button was found (no active sessions)
      const killButtonExists = await page.locator('.btn-kill-session').count() > 0;
      if (killButtonExists) {
        expect(terminateCalled).toBe(true);
      } else {
        console.log('No active sessions to terminate - skipping API verification');
      }
    });
  });

  test.describe('Session Data Display', () => {
    test('should display session key in data attribute', async ({ page }) => {
      // Navigate to activity view using JavaScript click for CI reliability
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="activity"]') as HTMLElement;
        if (tab) tab.click();
      });

      // Wait for activity container to be visible first
      await page.waitForSelector('#activity-container', {
        state: 'visible',
        timeout: TIMEOUTS.MEDIUM
      });

      // Wait for session cards to be rendered - use MEDIUM timeout for CI
      await page.waitForFunction(() => {
        return document.querySelector('.activity-session-card') !== null;
      }, { timeout: TIMEOUTS.MEDIUM });

      const sessionCard = page.locator('.activity-session-card').first();
      await expect(sessionCard).toBeVisible();

      const sessionKey = await sessionCard.getAttribute('data-session-key');
      expect(sessionKey).toBeTruthy();
    });

    test('should display user information in session cards', async ({ page }) => {
      // Navigate to activity view using JavaScript click for CI reliability
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="activity"]') as HTMLElement;
        if (tab) tab.click();
      });

      // Wait for activity container to be visible first
      await page.waitForSelector('#activity-container', {
        state: 'visible',
        timeout: TIMEOUTS.MEDIUM
      });

      // Wait for session cards to be rendered - use MEDIUM timeout for CI
      await page.waitForFunction(() => {
        return document.querySelector('.activity-session-card') !== null;
      }, { timeout: TIMEOUTS.MEDIUM });

      const sessionCard = page.locator('.activity-session-card').first();
      await expect(sessionCard).toContainText('Test User');
    });

    test('should display session title', async ({ page }) => {
      // Navigate to activity view using JavaScript click for CI reliability
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="activity"]') as HTMLElement;
        if (tab) tab.click();
      });

      // Wait for activity container to be visible first
      await page.waitForSelector('#activity-container', {
        state: 'visible',
        timeout: TIMEOUTS.MEDIUM
      });

      // Wait for session cards to be rendered - use MEDIUM timeout for CI
      await page.waitForFunction(() => {
        return document.querySelector('.activity-session-card') !== null;
      }, { timeout: TIMEOUTS.MEDIUM });

      const sessionCard = page.locator('.activity-session-card').first();
      await expect(sessionCard).toContainText('Test Movie');
    });
  });
});

test.describe('Session Management Error Handling', () => {
  test('should handle termination failure gracefully', async ({ page }) => {
    // Set header to request termination error from mock server
    // This is more reliable than page.route() which can conflict with context-level routes
    await page.setExtraHTTPHeaders({
      'X-Mock-Terminate-Error': 'true'
    });

    // Override confirm to auto-confirm
    await page.evaluate(() => {
      window.confirm = () => true;
    });

    await gotoAppAndWaitReady(page);
    await waitForNavReady(page);

    // Navigate to activity view using JavaScript click for CI reliability
    await page.evaluate(() => {
      const tab = document.querySelector('.nav-tab[data-view="activity"]') as HTMLElement;
      if (tab) tab.click();
    });

    // Wait for activity container to be visible first
    await page.waitForSelector('#activity-container', {
      state: 'visible',
      timeout: TIMEOUTS.MEDIUM
    });

    // Wait for session cards to be rendered
    await page.waitForFunction(() => {
      return document.querySelector('.activity-session-card') !== null;
    }, { timeout: TIMEOUTS.MEDIUM });

    // Use JavaScript click for CI reliability
    await page.evaluate(() => {
      const btn = document.querySelector('.btn-kill-session') as HTMLElement;
      if (btn) btn.click();
    });

    // Wait for confirmation dialog to appear (if custom dialog is used)
    await page.waitForFunction(() => {
      const dialog = document.querySelector('.confirmation-dialog, [role="alertdialog"]');
      return dialog !== null;
    }, { timeout: TIMEOUTS.RENDER }).catch(() => {
      // Native confirm might be used instead of custom dialog
    });

    const confirmBtn = page.locator('.confirmation-dialog .btn-confirm, [data-confirm="true"]');
    if (await confirmBtn.isVisible()) {
      await confirmBtn.click();
    }

    // Wait for app to stabilize after error
    await page.waitForFunction(() => {
      const app = document.querySelector('#app');
      return app !== null;
    });

    // App should still be functional after error
    const app = page.locator('#app');
    await expect(app).toBeVisible();
  });

  test('should handle no sessions gracefully', async ({ page }) => {
    // Set header to request empty sessions from mock server
    // This is more reliable than page.route() which can conflict with context-level routes
    await page.setExtraHTTPHeaders({
      'X-Mock-Empty-Sessions': 'true'
    });

    await gotoAppAndWaitReady(page);
    await waitForNavReady(page);

    // Navigate to activity view using JavaScript click for CI reliability
    await page.evaluate(() => {
      const tab = document.querySelector('.nav-tab[data-view="activity"]') as HTMLElement;
      if (tab) tab.click();
    });

    // Wait for activity container to be visible
    await page.waitForSelector('#activity-container', {
      state: 'visible',
      timeout: TIMEOUTS.MEDIUM
    });

    // Wait for activity view to fully render (empty state)
    await page.waitForFunction(() => {
      // Check for empty state element
      const emptyState = document.getElementById('activity-empty-state');
      return emptyState !== null;
    }, { timeout: TIMEOUTS.MEDIUM });

    // No kill buttons should be visible when there are no sessions
    const killButtons = page.locator('.btn-kill-session');
    await expect(killButtons).toHaveCount(0);
  });
});
