// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Library Details Sub-Tab Tests
 *
 * Tests the Library page sub-tab navigation between Charts and Details views.
 * LibraryDetailsManager is integrated into the Library Analytics page as a sub-tab.
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

// Library Details Sub-Tab - Tests enabled
// UI elements exist in HTML templates, mock endpoints added to mock-api-server.ts
test.describe('Library Details Sub-Tab', () => {
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

        // Navigate to Library page using JavaScript click
        await page.evaluate(() => {
            const libraryTab = document.querySelector('[data-analytics-page="library"]') as HTMLElement;
            if (libraryTab) libraryTab.click();
        });

        // Wait for Library page to be visible
        await page.waitForSelector('#analytics-library', { state: 'visible', timeout: TIMEOUTS.MEDIUM });
    });

    test('should display sub-tabs for Charts and Details', async ({ page }) => {
        // Verify sub-tabs navigation exists
        // WHY: This is the primary UI element for the Library Details feature
        const subTabs = page.locator('[data-testid="library-sub-tabs"]');
        await expect(subTabs).toBeVisible({ timeout: TIMEOUTS.ELEMENT_VISIBLE });

        // Verify both tabs are present
        const chartsTab = page.locator('#tab-library-charts');
        const detailsTab = page.locator('#tab-library-details');

        await expect(chartsTab).toBeVisible();
        await expect(detailsTab).toBeVisible();

        // Verify Charts tab is active by default
        // WHY: The existing Library Analytics (Charts) should be the default view
        await expect(chartsTab).toHaveClass(/active/);
        await expect(chartsTab).toHaveAttribute('aria-selected', 'true');
    });

    test('should show Charts panel by default', async ({ page }) => {
        // Verify Charts panel is visible
        const chartsPanel = page.locator('[data-testid="library-charts-panel"]');
        await expect(chartsPanel).toBeVisible({ timeout: TIMEOUTS.ELEMENT_VISIBLE });

        // Verify Details panel is hidden
        const detailsPanel = page.locator('[data-testid="library-details-panel"]');
        await expect(detailsPanel).not.toBeVisible();
    });

    test('should switch to Details panel when Details tab is clicked', async ({ page }) => {
        // Click Details tab using JavaScript
        // WHY: JavaScript clicks are deterministic in CI environments
        await page.evaluate(() => {
            const detailsTab = document.getElementById('tab-library-details');
            if (detailsTab) detailsTab.click();
        });

        // Wait for tab state change
        // WHY: We wait for the aria-selected attribute to change, not arbitrary timeout
        await page.waitForFunction(() => {
            const detailsTab = document.getElementById('tab-library-details');
            return detailsTab?.getAttribute('aria-selected') === 'true';
        }, { timeout: TIMEOUTS.ELEMENT_VISIBLE });

        // Verify Details panel is now visible
        const detailsPanel = page.locator('[data-testid="library-details-panel"]');
        await expect(detailsPanel).toBeVisible({ timeout: TIMEOUTS.ELEMENT_VISIBLE });

        // Verify Charts panel is hidden
        const chartsPanel = page.locator('[data-testid="library-charts-panel"]');
        await expect(chartsPanel).not.toBeVisible();

        // Verify Details tab is active
        const detailsTab = page.locator('#tab-library-details');
        await expect(detailsTab).toHaveClass(/active/);
    });

    test('should switch back to Charts panel from Details', async ({ page }) => {
        // First switch to Details
        await page.evaluate(() => {
            const detailsTab = document.getElementById('tab-library-details');
            if (detailsTab) detailsTab.click();
        });

        await page.waitForSelector('[data-testid="library-details-panel"]', {
            state: 'visible',
            timeout: TIMEOUTS.ELEMENT_VISIBLE
        });

        // Now switch back to Charts
        await page.evaluate(() => {
            const chartsTab = document.getElementById('tab-library-charts');
            if (chartsTab) chartsTab.click();
        });

        // Verify Charts panel is visible again
        const chartsPanel = page.locator('[data-testid="library-charts-panel"]');
        await expect(chartsPanel).toBeVisible({ timeout: TIMEOUTS.ELEMENT_VISIBLE });

        // Verify Details panel is hidden
        const detailsPanel = page.locator('[data-testid="library-details-panel"]');
        await expect(detailsPanel).not.toBeVisible();
    });

    test('should render library details container in Details tab', async ({ page }) => {
        // Switch to Details tab
        await page.evaluate(() => {
            const detailsTab = document.getElementById('tab-library-details');
            if (detailsTab) detailsTab.click();
        });

        // Wait for container to be visible
        // WHY: The container must exist for LibraryDetailsManager to render into
        const container = page.locator('[data-testid="library-details-container"]');
        await expect(container).toBeVisible({ timeout: TIMEOUTS.ELEMENT_VISIBLE });
    });

    test('should have proper ARIA attributes for accessibility', async ({ page }) => {
        // Verify tablist role on sub-tabs navigation
        const subTabs = page.locator('[data-testid="library-sub-tabs"]');
        await expect(subTabs).toHaveAttribute('role', 'tablist');

        // Verify tab role on individual tabs
        const chartsTab = page.locator('#tab-library-charts');
        const detailsTab = page.locator('#tab-library-details');

        await expect(chartsTab).toHaveAttribute('role', 'tab');
        await expect(detailsTab).toHaveAttribute('role', 'tab');

        // Verify aria-controls points to correct panels
        await expect(chartsTab).toHaveAttribute('aria-controls', 'library-charts-panel');
        await expect(detailsTab).toHaveAttribute('aria-controls', 'library-details-panel');

        // Verify tabpanel roles on panels
        const chartsPanel = page.locator('#library-charts-panel');
        const detailsPanel = page.locator('#library-details-panel');

        await expect(chartsPanel).toHaveAttribute('role', 'tabpanel');
        await expect(detailsPanel).toHaveAttribute('role', 'tabpanel');
    });
});
