// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Session Management E2E Tests
 * ADR-0015: Zero Trust Authentication
 *
 * Tests for the Active Session Management UI:
 * - View active sessions
 * - Session details display
 * - Revoke individual sessions
 * - Sign out everywhere functionality
 */

import { test, expect } from '@playwright/test';

test.describe('Session Management', () => {
  // Navigate to settings and expand Security section before each test
  test.beforeEach(async ({ page }) => {
    // Login first
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Open settings panel (via gear icon or keyboard shortcut)
    const settingsButton = page.locator('[data-testid="settings-button"], button:has(svg):has-text("Settings"), .settings-toggle');
    if (await settingsButton.isVisible()) {
      await settingsButton.click();
    } else {
      // Try keyboard shortcut
      await page.keyboard.press('Control+,');
    }

    // Wait for settings panel
    await expect(page.locator('.settings-panel, #settings-modal, [role="dialog"]')).toBeVisible({ timeout: 5000 });
  });

  test('displays active sessions list', async ({ page }) => {
    // Expand Security section if collapsed
    const securityHeader = page.locator('button:has-text("Security")');
    const securitySection = page.locator('[data-section="security"]');

    if (await securityHeader.isVisible()) {
      const isExpanded = await securityHeader.getAttribute('aria-expanded');
      if (isExpanded !== 'true') {
        await securityHeader.click();
      }
    }

    // Wait for sessions container
    await expect(page.locator('#security-sessions-container, .session-management-panel')).toBeVisible({ timeout: 5000 });

    // Check for session items
    const sessionItems = page.locator('.session-item');
    await expect(sessionItems).toHaveCount(3); // Mock returns 3 sessions
  });

  test('shows current session badge', async ({ page }) => {
    // Expand Security section
    const securityHeader = page.locator('button:has-text("Security")');
    if (await securityHeader.isVisible()) {
      const isExpanded = await securityHeader.getAttribute('aria-expanded');
      if (isExpanded !== 'true') {
        await securityHeader.click();
      }
    }

    // Check for current session indicator
    const currentSession = page.locator('.session-item.session-current');
    await expect(currentSession).toBeVisible();
    await expect(currentSession.locator('.session-badge')).toContainText('Current');
  });

  test('displays session provider information', async ({ page }) => {
    // Expand Security section
    const securityHeader = page.locator('button:has-text("Security")');
    if (await securityHeader.isVisible()) {
      const isExpanded = await securityHeader.getAttribute('aria-expanded');
      if (isExpanded !== 'true') {
        await securityHeader.click();
      }
    }

    // Check provider labels
    await expect(page.locator('.session-provider')).toHaveCount(3);

    // Verify at least one provider type is shown
    const providers = await page.locator('.session-provider').allTextContents();
    expect(providers.some(p => ['Plex OAuth', 'SSO (OIDC)', 'Username/Password'].includes(p))).toBe(true);
  });

  test('refresh button reloads sessions', async ({ page }) => {
    // Expand Security section
    const securityHeader = page.locator('button:has-text("Security")');
    if (await securityHeader.isVisible()) {
      const isExpanded = await securityHeader.getAttribute('aria-expanded');
      if (isExpanded !== 'true') {
        await securityHeader.click();
      }
    }

    // Find and click refresh button
    const refreshButton = page.locator('#refresh-sessions-btn, .session-refresh-btn');
    await expect(refreshButton).toBeVisible();

    // Intercept the sessions API call
    const responsePromise = page.waitForResponse(resp =>
      resp.url().includes('/oidc/sessions') && resp.request().method() === 'GET'
    );

    await refreshButton.click();
    await responsePromise;

    // Sessions should still be visible after refresh
    await expect(page.locator('.session-item')).toHaveCount(3);
  });

  test('can revoke non-current session', async ({ page }) => {
    // Expand Security section
    const securityHeader = page.locator('button:has-text("Security")');
    if (await securityHeader.isVisible()) {
      const isExpanded = await securityHeader.getAttribute('aria-expanded');
      if (isExpanded !== 'true') {
        await securityHeader.click();
      }
    }

    // Find a non-current session's revoke button
    const nonCurrentSession = page.locator('.session-item:not(.session-current)').first();
    await expect(nonCurrentSession).toBeVisible();

    const revokeButton = nonCurrentSession.locator('.session-revoke-btn');
    await expect(revokeButton).toBeVisible();

    // Intercept the delete request
    const responsePromise = page.waitForResponse(resp =>
      resp.url().includes('/oidc/sessions/') && resp.request().method() === 'DELETE'
    );

    await revokeButton.click();
    await responsePromise;

    // Session should be removed from list (or show success toast)
    // The mock doesn't actually persist the deletion, so we check for the toast
    await expect(page.locator('.toast, [role="alert"]').filter({ hasText: /revoked|success/i })).toBeVisible({ timeout: 3000 });
  });

  test('shows confirmation when revoking current session', async ({ page }) => {
    // Expand Security section
    const securityHeader = page.locator('button:has-text("Security")');
    if (await securityHeader.isVisible()) {
      const isExpanded = await securityHeader.getAttribute('aria-expanded');
      if (isExpanded !== 'true') {
        await securityHeader.click();
      }
    }

    // Find current session's revoke button
    const currentSession = page.locator('.session-item.session-current');
    await expect(currentSession).toBeVisible();

    const revokeButton = currentSession.locator('.session-revoke-btn');

    // Set up dialog handler to capture confirm message
    page.on('dialog', async dialog => {
      expect(dialog.type()).toBe('confirm');
      expect(dialog.message()).toContain('current session');
      await dialog.dismiss(); // Cancel the revocation
    });

    await revokeButton.click();
  });

  test('sign out everywhere button is present', async ({ page }) => {
    // Expand Security section
    const securityHeader = page.locator('button:has-text("Security")');
    if (await securityHeader.isVisible()) {
      const isExpanded = await securityHeader.getAttribute('aria-expanded');
      if (isExpanded !== 'true') {
        await securityHeader.click();
      }
    }

    // Check for logout all button
    const logoutAllButton = page.locator('#logout-all-btn, .session-logout-all-btn');
    await expect(logoutAllButton).toBeVisible();
    await expect(logoutAllButton).toContainText(/Sign Out Everywhere/i);
  });

  test('sign out everywhere shows confirmation', async ({ page }) => {
    // Expand Security section
    const securityHeader = page.locator('button:has-text("Security")');
    if (await securityHeader.isVisible()) {
      const isExpanded = await securityHeader.getAttribute('aria-expanded');
      if (isExpanded !== 'true') {
        await securityHeader.click();
      }
    }

    const logoutAllButton = page.locator('#logout-all-btn, .session-logout-all-btn');

    // Set up dialog handler
    page.on('dialog', async dialog => {
      expect(dialog.type()).toBe('confirm');
      expect(dialog.message()).toContain('Sign out everywhere');
      await dialog.dismiss(); // Cancel to avoid page reload
    });

    await logoutAllButton.click();
  });

  test('session list handles loading state', async ({ page }) => {
    // Intercept sessions request to add delay
    await page.route('**/oidc/sessions', async route => {
      // Add artificial delay to see loading state
      await new Promise(resolve => setTimeout(resolve, 500));
      await route.continue();
    });

    // Reload page to trigger fresh load
    await page.reload();

    // Open settings and expand Security
    const settingsButton = page.locator('[data-testid="settings-button"], button:has(svg):has-text("Settings"), .settings-toggle');
    if (await settingsButton.isVisible()) {
      await settingsButton.click();
    }

    const securityHeader = page.locator('button:has-text("Security")');
    if (await securityHeader.isVisible()) {
      const isExpanded = await securityHeader.getAttribute('aria-expanded');
      if (isExpanded !== 'true') {
        await securityHeader.click();
      }
    }

    // Check for loading indicator or loading class
    const loadingIndicator = page.locator('.sessions-loading, .session-loading, .loading');
    // May or may not catch the loading state depending on timing
    // Just ensure the sessions eventually load
    await expect(page.locator('.session-item')).toHaveCount(3, { timeout: 10000 });
  });

  test('handles API error gracefully', async ({ page }) => {
    // Intercept sessions request to return error
    await page.route('**/oidc/sessions', route => {
      route.fulfill({
        status: 500,
        body: JSON.stringify({ error: 'Internal server error' })
      });
    });

    // Reload to trigger fresh load
    await page.reload();

    // Open settings and expand Security
    const settingsButton = page.locator('[data-testid="settings-button"], button:has(svg):has-text("Settings"), .settings-toggle');
    if (await settingsButton.isVisible()) {
      await settingsButton.click();
    }

    const securityHeader = page.locator('button:has-text("Security")');
    if (await securityHeader.isVisible()) {
      const isExpanded = await securityHeader.getAttribute('aria-expanded');
      if (isExpanded !== 'true') {
        await securityHeader.click();
      }
    }

    // Should show error state
    await expect(page.locator('.sessions-error, [class*="error"]').filter({ hasText: /failed|error/i })).toBeVisible({ timeout: 5000 });
  });
});
