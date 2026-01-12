// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Collections & Playlists Sub-Tab Tests
 *
 * Tests the Content page sub-tab navigation between Analytics, Collections, and Playlists views.
 * LibraryContentManager is integrated as Collections and Playlists sub-tabs.
 *
 * Test Strategy:
 * - Uses gotoAppAndWaitReady for reliable app initialization in CI
 * - Uses JavaScript clicks for CI reliability (Playwright's .click() can be flaky in headless/SwiftShader)
 * - Uses proper waitFor conditions instead of arbitrary timeouts
 * - All selectors use data-testid for stability
 */

// ROOT CAUSE FIX: Import from fixtures to enable autoMockApi for fast, deterministic tests
// Previously imported from @playwright/test which bypassed API mocking, causing 30+ second test times
import { test, expect, TIMEOUTS, gotoAppAndWaitReady, waitForNavReady } from './fixtures';

// Collections & Playlists Sub-Tabs - Tests enabled
// UI elements exist in HTML templates, mock endpoints added to mock-api-server.ts
test.describe('Collections & Playlists Sub-Tabs', () => {
    test.beforeEach(async ({ page }) => {
        // Use standard gotoAppAndWaitReady for reliable app initialization in CI
        // WHY: This helper has retry logic and proper timeout handling for SwiftShader environments
        await gotoAppAndWaitReady(page);
        await waitForNavReady(page);

        // Navigate to Analytics view using JavaScript click
        // WHY: JavaScript clicks are more reliable in CI/headless environments than Playwright's .click()
        await page.evaluate(() => {
            const analyticsTab = document.querySelector('.nav-tab[data-view="analytics"]') as HTMLElement;
            if (analyticsTab) analyticsTab.click();
        });

        // Wait for analytics navigation to render
        await page.waitForSelector('#analytics-nav', { state: 'visible', timeout: TIMEOUTS.MEDIUM });

        // Navigate to Content page using JavaScript click
        await page.evaluate(() => {
            const contentTab = document.querySelector('[data-analytics-page="content"]') as HTMLElement;
            if (contentTab) contentTab.click();
        });

        // Wait for Content page to be visible
        await page.waitForSelector('#analytics-content', { state: 'visible', timeout: TIMEOUTS.MEDIUM });
    });

    test('should display sub-tabs for Analytics, Collections, and Playlists', async ({ page }) => {
        // Verify sub-tabs navigation exists
        // WHY: This is the primary UI element we added
        const subTabs = page.locator('[data-testid="content-sub-tabs"]');
        await expect(subTabs).toBeVisible({ timeout: TIMEOUTS.ELEMENT_VISIBLE });

        // Verify all three tabs are present
        const analyticsTab = page.locator('#tab-content-analytics');
        const collectionsTab = page.locator('#tab-content-collections');
        const playlistsTab = page.locator('#tab-content-playlists');

        await expect(analyticsTab).toBeVisible();
        await expect(collectionsTab).toBeVisible();
        await expect(playlistsTab).toBeVisible();

        // Verify Analytics tab is active by default
        // WHY: The existing Content Analytics charts should be the default view
        await expect(analyticsTab).toHaveClass(/active/);
        await expect(analyticsTab).toHaveAttribute('aria-selected', 'true');
    });

    test('should show Analytics panel by default', async ({ page }) => {
        // Verify Analytics panel is visible
        const analyticsPanel = page.locator('[data-testid="content-analytics-panel"]');
        await expect(analyticsPanel).toBeVisible({ timeout: TIMEOUTS.ELEMENT_VISIBLE });

        // Verify Collections and Playlists panels are hidden
        const collectionsPanel = page.locator('[data-testid="content-collections-panel"]');
        const playlistsPanel = page.locator('[data-testid="content-playlists-panel"]');

        await expect(collectionsPanel).not.toBeVisible();
        await expect(playlistsPanel).not.toBeVisible();
    });

    test('should switch to Collections panel when Collections tab is clicked', async ({ page }) => {
        // Click Collections tab using JavaScript
        // WHY: JavaScript clicks are deterministic in CI environments
        await page.evaluate(() => {
            const collectionsTab = document.getElementById('tab-content-collections');
            if (collectionsTab) collectionsTab.click();
        });

        // Wait for tab state change
        // WHY: We wait for the aria-selected attribute to change, not arbitrary timeout
        await page.waitForFunction(() => {
            const collectionsTab = document.getElementById('tab-content-collections');
            return collectionsTab?.getAttribute('aria-selected') === 'true';
        }, { timeout: TIMEOUTS.ELEMENT_VISIBLE });

        // Verify Collections panel is now visible
        const collectionsPanel = page.locator('[data-testid="content-collections-panel"]');
        await expect(collectionsPanel).toBeVisible({ timeout: TIMEOUTS.ELEMENT_VISIBLE });

        // Verify other panels are hidden
        const analyticsPanel = page.locator('[data-testid="content-analytics-panel"]');
        const playlistsPanel = page.locator('[data-testid="content-playlists-panel"]');

        await expect(analyticsPanel).not.toBeVisible();
        await expect(playlistsPanel).not.toBeVisible();

        // Verify Collections tab is active
        const collectionsTab = page.locator('#tab-content-collections');
        await expect(collectionsTab).toHaveClass(/active/);
    });

    test('should switch to Playlists panel when Playlists tab is clicked', async ({ page }) => {
        // Click Playlists tab using JavaScript
        await page.evaluate(() => {
            const playlistsTab = document.getElementById('tab-content-playlists');
            if (playlistsTab) playlistsTab.click();
        });

        // Wait for tab state change
        await page.waitForFunction(() => {
            const playlistsTab = document.getElementById('tab-content-playlists');
            return playlistsTab?.getAttribute('aria-selected') === 'true';
        }, { timeout: TIMEOUTS.ELEMENT_VISIBLE });

        // Verify Playlists panel is now visible
        const playlistsPanel = page.locator('[data-testid="content-playlists-panel"]');
        await expect(playlistsPanel).toBeVisible({ timeout: TIMEOUTS.ELEMENT_VISIBLE });

        // Verify other panels are hidden
        const analyticsPanel = page.locator('[data-testid="content-analytics-panel"]');
        const collectionsPanel = page.locator('[data-testid="content-collections-panel"]');

        await expect(analyticsPanel).not.toBeVisible();
        await expect(collectionsPanel).not.toBeVisible();

        // Verify Playlists tab is active
        const playlistsTab = page.locator('#tab-content-playlists');
        await expect(playlistsTab).toHaveClass(/active/);
    });

    test('should switch back to Analytics from Collections', async ({ page }) => {
        // First switch to Collections
        await page.evaluate(() => {
            const collectionsTab = document.getElementById('tab-content-collections');
            if (collectionsTab) collectionsTab.click();
        });

        await page.waitForSelector('[data-testid="content-collections-panel"]', {
            state: 'visible',
            timeout: TIMEOUTS.ELEMENT_VISIBLE
        });

        // Now switch back to Analytics
        await page.evaluate(() => {
            const analyticsTab = document.getElementById('tab-content-analytics');
            if (analyticsTab) analyticsTab.click();
        });

        // Verify Analytics panel is visible again
        const analyticsPanel = page.locator('[data-testid="content-analytics-panel"]');
        await expect(analyticsPanel).toBeVisible({ timeout: TIMEOUTS.ELEMENT_VISIBLE });

        // Verify Collections panel is hidden
        const collectionsPanel = page.locator('[data-testid="content-collections-panel"]');
        await expect(collectionsPanel).not.toBeVisible();
    });

    test('should render collections container in Collections tab', async ({ page }) => {
        // Switch to Collections tab
        await page.evaluate(() => {
            const collectionsTab = document.getElementById('tab-content-collections');
            if (collectionsTab) collectionsTab.click();
        });

        // Wait for container to be visible
        // WHY: The container must exist for LibraryContentManager to render into
        const container = page.locator('[data-testid="collections-container"]');
        await expect(container).toBeVisible({ timeout: TIMEOUTS.ELEMENT_VISIBLE });
    });

    test('should render playlists container in Playlists tab', async ({ page }) => {
        // Switch to Playlists tab
        await page.evaluate(() => {
            const playlistsTab = document.getElementById('tab-content-playlists');
            if (playlistsTab) playlistsTab.click();
        });

        // Wait for container to be visible
        const container = page.locator('[data-testid="playlists-container"]');
        await expect(container).toBeVisible({ timeout: TIMEOUTS.ELEMENT_VISIBLE });
    });

    test('should have proper ARIA attributes for accessibility', async ({ page }) => {
        // Verify tablist role on sub-tabs navigation
        const subTabs = page.locator('[data-testid="content-sub-tabs"]');
        await expect(subTabs).toHaveAttribute('role', 'tablist');

        // Verify tab role on individual tabs
        const analyticsTab = page.locator('#tab-content-analytics');
        const collectionsTab = page.locator('#tab-content-collections');
        const playlistsTab = page.locator('#tab-content-playlists');

        await expect(analyticsTab).toHaveAttribute('role', 'tab');
        await expect(collectionsTab).toHaveAttribute('role', 'tab');
        await expect(playlistsTab).toHaveAttribute('role', 'tab');

        // Verify aria-controls points to correct panels
        await expect(analyticsTab).toHaveAttribute('aria-controls', 'content-analytics-panel');
        await expect(collectionsTab).toHaveAttribute('aria-controls', 'content-collections-panel');
        await expect(playlistsTab).toHaveAttribute('aria-controls', 'content-playlists-panel');
    });

    test('should have library filter in Collections panel', async ({ page }) => {
        // Switch to Collections tab
        await page.evaluate(() => {
            const collectionsTab = document.getElementById('tab-content-collections');
            if (collectionsTab) collectionsTab.click();
        });

        await page.waitForSelector('[data-testid="content-collections-panel"]', {
            state: 'visible',
            timeout: TIMEOUTS.ELEMENT_VISIBLE
        });

        // Verify library filter exists
        // WHY: Collections can be filtered by library (section_id)
        const libraryFilter = page.locator('#collections-library-filter');
        await expect(libraryFilter).toBeVisible();
    });
});
