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
 * E2E Test: Cross-Platform Feature
 *
 * Tests cross-platform content mapping and user linking:
 * - Navigation to cross-platform view
 * - Tab switching between Analytics, Content Mapping, User Linking
 * - Cross-platform analytics dashboard display
 * - Content mapping interface
 * - User linking interface
 *
 * Reference: docs/working/AUDIT_REPORT.md
 */

test.describe('Cross-Platform Feature', () => {
  test.beforeEach(async ({ page }) => {
    await gotoAppAndWaitReady(page);
  });

  test.describe('Navigation', () => {
    test('should render cross-platform tab in main navigation', async ({ page }) => {
      // Cross-platform tab should be visible in nav
      const crossPlatformTab = page.locator('button[data-view="cross-platform"]');
      await expect(crossPlatformTab).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test('should navigate to cross-platform view', async ({ page }) => {
      // Click on cross-platform tab
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const btn = document.querySelector('button[data-view="cross-platform"]') as HTMLElement;
        if (btn) btn.click();
      });

      // Container should be visible
      await expect(page.locator('#cross-platform-container')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test('should show cross-platform header when navigating to view', async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const btn = document.querySelector('button[data-view="cross-platform"]') as HTMLElement;
        if (btn) btn.click();
      });
      await page.waitForSelector('#cross-platform-container', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

      // Should have a header with title
      const title = page.locator('.cross-platform-container .cp-title, .cp-header h2');
      await expect(title).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test('should update active tab state when navigating', async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const btn = document.querySelector('button[data-view="cross-platform"]') as HTMLElement;
        if (btn) btn.click();
      });
      await page.waitForSelector('#cross-platform-container', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

      // Cross-platform tab should have active class
      const crossPlatformTab = page.locator('button[data-view="cross-platform"]');
      await expect(crossPlatformTab).toHaveClass(/active/);
    });
  });

  test.describe('Tab Navigation', () => {
    test.beforeEach(async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const btn = document.querySelector('button[data-view="cross-platform"]') as HTMLElement;
        if (btn) btn.click();
      });
      await page.waitForSelector('#cross-platform-container', { state: 'visible', timeout: TIMEOUTS.MEDIUM });
    });

    test('should show tabs for Analytics, Content Mapping, and User Linking', async ({ page }) => {
      // Wait for tabs to be rendered
      await page.waitForSelector('.cp-tabs', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

      // Should have 3 tabs
      const tabs = page.locator('.cp-tab');
      await expect(tabs).toHaveCount(3);
    });

    test('should show Analytics tab as active by default', async ({ page }) => {
      // Wait for tabs to be rendered
      await page.waitForSelector('.cp-tabs', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

      const analyticsTab = page.locator('.cp-tab[data-tab="analytics"]');
      await expect(analyticsTab).toHaveClass(/active/);
    });

    test('should switch to Content Mapping tab', async ({ page }) => {
      await page.waitForSelector('.cp-tabs', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const el = document.querySelector('.cp-tab[data-tab="content"]') as HTMLElement;
        if (el) el.click();
      });

      // Content tab should be active
      const contentTab = page.locator('.cp-tab[data-tab="content"]');
      await expect(contentTab).toHaveClass(/active/);

      // Content panel should be visible
      const contentPanel = page.locator('#cp-content-panel');
      await expect(contentPanel).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test('should switch to User Linking tab', async ({ page }) => {
      await page.waitForSelector('.cp-tabs', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const el = document.querySelector('.cp-tab[data-tab="users"]') as HTMLElement;
        if (el) el.click();
      });

      // Users tab should be active
      const usersTab = page.locator('.cp-tab[data-tab="users"]');
      await expect(usersTab).toHaveClass(/active/);

      // Users panel should be visible
      const usersPanel = page.locator('#cp-users-panel');
      await expect(usersPanel).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test('should have keyboard accessible tabs', async ({ page }) => {
      await page.waitForSelector('.cp-tabs', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

      // Focus on tabs
      await page.focus('.cp-tab[data-tab="analytics"]');

      // Press ArrowRight to move to next tab
      await page.keyboard.press('ArrowRight');

      // Content tab should now be focused and active
      const contentTab = page.locator('.cp-tab[data-tab="content"]');
      await expect(contentTab).toBeFocused();
    });
  });

  test.describe('Analytics Dashboard', () => {
    test.beforeEach(async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const btn = document.querySelector('button[data-view="cross-platform"]') as HTMLElement;
        if (btn) btn.click();
      });
      await page.waitForSelector('#cross-platform-container', { state: 'visible', timeout: TIMEOUTS.MEDIUM });
    });

    test('should show analytics panel by default', async ({ page }) => {
      const analyticsPanel = page.locator('#cp-analytics-panel');
      await expect(analyticsPanel).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test('should show view toggle buttons (By Platform / Combined)', async ({ page }) => {
      await page.waitForSelector('.cp-analytics', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

      const viewToggle = page.locator('.cp-view-toggle');
      await expect(viewToggle).toBeVisible();

      // Should have 2 buttons
      const buttons = viewToggle.locator('.cp-view-btn');
      await expect(buttons).toHaveCount(2);
    });

    test('should show refresh button', async ({ page }) => {
      await page.waitForSelector('.cp-analytics', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

      const refreshBtn = page.locator('#cp-analytics-refresh');
      await expect(refreshBtn).toBeVisible();
    });

    test('should show loading or content state', async ({ page }) => {
      await page.waitForSelector('.cp-analytics', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

      // Either loading, content, or empty state should be visible
      const hasLoading = await page.locator('#cp-analytics-loading').isVisible().catch(() => false);
      const hasContent = await page.locator('#cp-charts-container').isVisible().catch(() => false);
      const hasEmpty = await page.locator('#cp-analytics-empty').isVisible().catch(() => false);

      expect(hasLoading || hasContent || hasEmpty).toBe(true);
    });
  });

  test.describe('Content Mapping Panel', () => {
    test.beforeEach(async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const btn = document.querySelector('button[data-view="cross-platform"]') as HTMLElement;
        if (btn) btn.click();
      });
      await page.waitForSelector('#cross-platform-container', { state: 'visible', timeout: TIMEOUTS.MEDIUM });
      await page.waitForSelector('.cp-tabs', { state: 'visible', timeout: TIMEOUTS.MEDIUM });
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const el = document.querySelector('.cp-tab[data-tab="content"]') as HTMLElement;
        if (el) el.click();
      });
      await page.waitForSelector('#cp-content-panel', { state: 'visible', timeout: TIMEOUTS.MEDIUM });
    });

    test('should show content mapping search input', async ({ page }) => {
      const searchInput = page.locator('#content-mapping-search');
      await expect(searchInput).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test('should show create mapping button', async ({ page }) => {
      const createBtn = page.locator('#content-mapping-create-btn, .content-mapping-create');
      await expect(createBtn).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test('should show import/export buttons', async ({ page }) => {
      const importBtn = page.locator('#content-mapping-import-btn, .content-mapping-import');
      const exportBtn = page.locator('#content-mapping-export-btn, .content-mapping-export');
      await expect(importBtn).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
      await expect(exportBtn).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test('should show mappings list or empty state', async ({ page }) => {
      // Either the list or empty state should be visible
      const hasList = await page.locator('.content-mapping-list, .content-mapping-table').isVisible().catch(() => false);
      const hasEmpty = await page.locator('.content-mapping-empty').isVisible().catch(() => false);
      const hasLoading = await page.locator('.content-mapping-loading').isVisible().catch(() => false);

      expect(hasList || hasEmpty || hasLoading).toBe(true);
    });
  });

  test.describe('User Linking Panel', () => {
    test.beforeEach(async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const btn = document.querySelector('button[data-view="cross-platform"]') as HTMLElement;
        if (btn) btn.click();
      });
      await page.waitForSelector('#cross-platform-container', { state: 'visible', timeout: TIMEOUTS.MEDIUM });
      await page.waitForSelector('.cp-tabs', { state: 'visible', timeout: TIMEOUTS.MEDIUM });
      // Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const el = document.querySelector('.cp-tab[data-tab="users"]') as HTMLElement;
        if (el) el.click();
      });
      await page.waitForSelector('#cp-users-panel', { state: 'visible', timeout: TIMEOUTS.MEDIUM });
    });

    test('should show user linking panel', async ({ page }) => {
      const userLinkingPanel = page.locator('.user-linking-panel');
      await expect(userLinkingPanel).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test('should show create link button', async ({ page }) => {
      const createBtn = page.locator('#user-linking-create-btn, .user-linking-create');
      await expect(createBtn).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test('should show linked users list or empty state', async ({ page }) => {
      // Either the list or empty state should be visible
      const hasList = await page.locator('.user-linking-list').isVisible().catch(() => false);
      const hasEmpty = await page.locator('.user-linking-empty').isVisible().catch(() => false);
      const hasLoading = await page.locator('.user-linking-loading').isVisible().catch(() => false);

      expect(hasList || hasEmpty || hasLoading).toBe(true);
    });
  });

  test.describe('Accessibility', () => {
    test.beforeEach(async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const btn = document.querySelector('button[data-view="cross-platform"]') as HTMLElement;
        if (btn) btn.click();
      });
      await page.waitForSelector('#cross-platform-container', { state: 'visible', timeout: TIMEOUTS.MEDIUM });
    });

    test('should have proper ARIA roles on tabs', async ({ page }) => {
      await page.waitForSelector('.cp-tabs', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

      const tabList = page.locator('.cp-tabs');
      await expect(tabList).toHaveAttribute('role', 'tablist');

      const tabs = page.locator('.cp-tab');
      const count = await tabs.count();
      for (let i = 0; i < count; i++) {
        await expect(tabs.nth(i)).toHaveAttribute('role', 'tab');
      }
    });

    test('should have aria-selected on active tab', async ({ page }) => {
      await page.waitForSelector('.cp-tabs', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

      const activeTab = page.locator('.cp-tab.active');
      await expect(activeTab).toHaveAttribute('aria-selected', 'true');
    });

    test('should have tabpanel role on panels', async ({ page }) => {
      await page.waitForSelector('.cp-tabs', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

      const panels = page.locator('.cp-tab-panel');
      const count = await panels.count();
      for (let i = 0; i < count; i++) {
        await expect(panels.nth(i)).toHaveAttribute('role', 'tabpanel');
      }
    });
  });

  test.describe('URL State', () => {
    test('should update URL hash when navigating to cross-platform view', async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const btn = document.querySelector('button[data-view="cross-platform"]') as HTMLElement;
        if (btn) btn.click();
      });
      await page.waitForSelector('#cross-platform-container', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

      // URL should have cross-platform hash
      const url = page.url();
      expect(url).toContain('#cross-platform');
    });

    test('should restore cross-platform view from URL hash', async ({ page }) => {
      // Navigate directly to cross-platform via URL
      await page.goto(page.url().split('#')[0] + '#cross-platform');
      await page.waitForSelector('#cross-platform-container', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

      // Cross-platform tab should be active
      const crossPlatformTab = page.locator('button[data-view="cross-platform"]');
      await expect(crossPlatformTab).toHaveClass(/active/);
    });
  });
});
