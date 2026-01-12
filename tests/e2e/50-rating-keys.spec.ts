// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Tests for Rating Keys Tracking
 *
 * Tests for:
 * - Rating key lookup form
 * - New rating keys lookup
 * - Old rating keys lookup
 * - Results display
 * - Copy functionality
 *
 * WIRED: RatingKeysManager is connected to the main application
 * Navigation: Analytics -> Tautulli -> Rating Keys sub-tab
 */

import {
    test,
    expect,
    TIMEOUTS,
    TAUTULLI_SUB_TABS,
    gotoAppAndWaitReady,
    navigateToTautulliSubTab,
} from './fixtures';

test.describe('Rating Keys Tracking', () => {
    // NOTE: Mock data is now provided by mock-api-server.ts (context-level mocking)
    // Tests use X-Mock-* headers to control mock behavior for specific scenarios
    // See ADR-0025: Deterministic E2E Test Mocking

    test.beforeEach(async ({ page }) => {
        // NOTE: API mocking is handled by the fixture's Express mock server (mock-api-server.ts)
        // Do NOT use page.route() here as it conflicts with context-level routing
        // See ADR-0025: Deterministic E2E Test Mocking

        // Navigate to app and then to the Rating Keys sub-tab
        await gotoAppAndWaitReady(page);
        await navigateToTautulliSubTab(page, TAUTULLI_SUB_TABS.RATING_KEYS);

        // Verify the manager container is visible
        await expect(page.locator('[data-testid="rating-keys-manager"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test.describe('Lookup Form', () => {
        test('should display rating key input', async ({ page }) => {
            const input = page.locator('#rating-key-input');
            await expect(input).toBeVisible({ timeout: TIMEOUTS.SHORT });
        });

        test('should display direction select', async ({ page }) => {
            const select = page.locator('#lookup-direction');
            await expect(select).toBeVisible({ timeout: TIMEOUTS.SHORT });

            // Should have both options
            await expect(select).toContainText('Find New Keys');
            await expect(select).toContainText('Find Old Keys');
        });

        test('should have lookup button', async ({ page }) => {
            const button = page.locator('#lookup-btn');
            await expect(button).toBeVisible({ timeout: TIMEOUTS.SHORT });
            await expect(button).toContainText('Lookup');
        });
    });

    test.describe('New Rating Keys Lookup', () => {
        test('should lookup new rating keys', async ({ page }) => {
            // Enter rating key
            await page.fill('#rating-key-input', '12345');

            // Ensure "Find New Keys" is selected
            await page.selectOption('#lookup-direction', 'new');

            // Click lookup
            await page.click('#lookup-btn');

            // Wait for results
            const results = page.locator('[data-testid="rating-keys-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

            // Should show the mappings
            const table = page.locator('[data-testid="rating-keys-table"]');
            await expect(table).toBeVisible({ timeout: TIMEOUTS.SHORT });
            await expect(table).toContainText('Test Movie');
            await expect(table).toContainText('67890');
        });

        test('should display mapping details', async ({ page }) => {
            await page.fill('#rating-key-input', '12345');
            await page.click('#lookup-btn');

            const table = page.locator('[data-testid="rating-keys-table"]');
            await expect(table).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

            // Check columns
            await expect(table).toContainText('Title');
            await expect(table).toContainText('Media Type');
            await expect(table).toContainText('Old Key');
            await expect(table).toContainText('New Key');
            await expect(table).toContainText('Updated');
        });
    });

    test.describe('Old Rating Keys Lookup', () => {
        test('should lookup old rating keys', async ({ page }) => {
            // Enter rating key
            await page.fill('#rating-key-input', '67890');

            // Select "Find Old Keys"
            await page.selectOption('#lookup-direction', 'old');

            // Click lookup
            await page.click('#lookup-btn');

            // Wait for results
            const results = page.locator('[data-testid="rating-keys-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

            const table = page.locator('[data-testid="rating-keys-table"]');
            await expect(table).toContainText('11111');
        });
    });

    test.describe('Empty Results', () => {
        test('should show empty state when no mappings found', async ({ page }) => {
            // Use X-Mock-* header to trigger empty results from mock server
            // See mock-api-server.ts - X-Mock-Rating-Keys-Empty triggers empty response
            await page.setExtraHTTPHeaders({ 'X-Mock-Rating-Keys-Empty': 'true' });

            await page.fill('#rating-key-input', '99999');
            await page.click('#lookup-btn');

            const noMappings = page.locator('[data-testid="no-mappings"]');
            await expect(noMappings).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(noMappings).toContainText('No rating key mappings found');

            // Clear header for subsequent tests
            await page.setExtraHTTPHeaders({});
        });
    });

    test.describe('API Integration', () => {
        test('should call new-rating-keys API with correct rating key', async ({ page }) => {
            // API mocking handled by mock-server.ts at context level
            // The mock server returns data based on the rating_key query param
            // so if results display correctly, the API was called correctly
            await page.fill('#rating-key-input', '12345');
            await page.selectOption('#lookup-direction', 'new');
            await page.click('#lookup-btn');

            // Wait for results to appear (indicates API call completed)
            const results = page.locator('[data-testid="rating-keys-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

            // Verify the table shows mappings with the new_rating_key matching our input
            // Mock server returns data with new_rating_key set to the queried rating_key
            const table = page.locator('[data-testid="rating-keys-table"]');
            await expect(table).toBeVisible();
            await expect(table).toContainText('12345');
        });

        test('should call old-rating-keys API with correct rating key', async ({ page }) => {
            // API mocking handled by mock-server.ts at context level
            await page.fill('#rating-key-input', '67890');
            await page.selectOption('#lookup-direction', 'old');
            await page.click('#lookup-btn');

            // Wait for results to appear (indicates API call completed)
            const results = page.locator('[data-testid="rating-keys-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

            // Verify the table shows mappings with the old_rating_key matching our input
            // Mock server returns data with old_rating_key set to the queried rating_key
            const table = page.locator('[data-testid="rating-keys-table"]');
            await expect(table).toBeVisible();
            await expect(table).toContainText('67890');
        });
    });

    test.describe('Error Handling', () => {
        test('should show error state on API failure', async ({ page }) => {
            // Use X-Mock-* header to trigger error response from mock server
            // See mock-api-server.ts - X-Mock-Rating-Keys-Error triggers 500 error
            await page.setExtraHTTPHeaders({ 'X-Mock-Rating-Keys-Error': 'true' });

            await page.fill('#rating-key-input', '12345');
            await page.click('#lookup-btn');

            // Use specific selector within rating-keys container to avoid matching other .error-message elements
            const errorState = page.locator('.rating-keys-results .error-message');
            await expect(errorState).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

            // Clear header for subsequent tests
            await page.setExtraHTTPHeaders({});
        });

        test('should have retry button on error', async ({ page }) => {
            // Use X-Mock-* header to trigger error response from mock server
            await page.setExtraHTTPHeaders({ 'X-Mock-Rating-Keys-Error': 'true' });

            await page.fill('#rating-key-input', '12345');
            await page.click('#lookup-btn');

            const retryBtn = page.locator('#retry-btn');
            await expect(retryBtn).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

            // Clear header for subsequent tests
            await page.setExtraHTTPHeaders({});
        });
    });

    test.describe('Keyboard Support', () => {
        test('should submit on Enter key', async ({ page }) => {
            // API mocking handled by mock-server.ts at context level
            const input = page.locator('#rating-key-input');
            await input.fill('12345');
            await input.press('Enter');

            // Wait for results to appear (indicates API call was triggered by Enter key)
            const results = page.locator('[data-testid="rating-keys-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

            // Verify the table displays data (confirms API call completed)
            const table = page.locator('[data-testid="rating-keys-table"]');
            await expect(table).toBeVisible();
        });
    });
});
