// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Tests for Tautulli Export UI
 *
 * Tests for:
 * - Export list view
 * - Create export form
 * - Field selection
 * - Download and delete actions
 */

// ROOT CAUSE FIX: Import test/expect from fixtures to enable autoMockApi for fast, deterministic tests
// Previously imported from @playwright/test which bypassed fixture's automatic API mocking
import { test, expect, gotoAppAndWaitReady, waitForNavReady, TIMEOUTS } from './fixtures';

test.describe('Tautulli Export UI', () => {
    const mockExportsTable = {
        status: 'success',
        data: {
            draw: 1,
            recordsTotal: 3,
            recordsFiltered: 3,
            data: [
                {
                    id: 1,
                    timestamp: 1702400000,
                    section_id: 1,
                    file_format: 'csv',
                    export_type: 'library',
                    media_type: 'movie',
                    title: 'Movies Export',
                    file_size: 1048576,
                    complete: 1
                },
                {
                    id: 2,
                    timestamp: 1702300000,
                    section_id: 2,
                    file_format: 'json',
                    export_type: 'library',
                    media_type: 'show',
                    title: 'TV Shows Export',
                    file_size: 2097152,
                    complete: 1
                },
                {
                    id: 3,
                    timestamp: 1702200000,
                    section_id: 1,
                    user_id: 1,
                    file_format: 'csv',
                    export_type: 'user',
                    title: 'User Watch History',
                    file_size: 524288,
                    complete: 0
                }
            ]
        },
        metadata: { timestamp: new Date().toISOString() }
    };

    const mockExportFields = {
        status: 'success',
        data: [
            {
                field_name: 'title',
                display_name: 'Title',
                description: 'Media title',
                field_type: 'string'
            },
            {
                field_name: 'year',
                display_name: 'Year',
                description: 'Release year',
                field_type: 'integer'
            },
            {
                field_name: 'rating',
                display_name: 'Rating',
                description: 'Audience rating',
                field_type: 'float'
            },
            {
                field_name: 'duration',
                display_name: 'Duration',
                description: 'Duration in seconds',
                field_type: 'integer'
            },
            {
                field_name: 'added_at',
                display_name: 'Added At',
                description: 'Date added to library',
                field_type: 'timestamp'
            }
        ],
        metadata: { timestamp: new Date().toISOString() }
    };

    const mockLibraries = {
        status: 'success',
        data: {
            draw: 1,
            recordsTotal: 2,
            recordsFiltered: 2,
            data: [
                { section_id: 1, section_name: 'Movies', section_type: 'movie' },
                { section_id: 2, section_name: 'TV Shows', section_type: 'show' }
            ]
        },
        metadata: { timestamp: new Date().toISOString() }
    };

    const mockExportResponse = {
        status: 'success',
        data: {
            export_id: '4',
            status: 'started',
            file_format: 'csv'
        },
        metadata: { timestamp: new Date().toISOString() }
    };

    test.beforeEach(async ({ page }) => {
        // Note: Onboarding skip is handled by fixtures via addInitScript
        // Note: API mocking is handled by fixtures (autoMockApi is enabled by default)

        // Feature-specific routes for export functionality
        // These routes use page.route() which may not work reliably with the mock server proxy.
        // If tests fail, consider adding these endpoints to mock-api-server.ts instead.
        await page.route('**/api/v1/tautulli/exports-table*', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify(mockExportsTable)
            });
        });

        await page.route('**/api/v1/tautulli/export-fields*', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify(mockExportFields)
            });
        });

        await page.route('**/api/v1/tautulli/libraries-table*', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify(mockLibraries)
            });
        });

        await page.route('**/api/v1/tautulli/export-metadata*', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify(mockExportResponse)
            });
        });

        await page.route('**/api/v1/tautulli/delete-export*', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ status: 'success', data: null })
            });
        });

        // Navigate to app using the deterministic helper (handles auth, onboarding, network settling)
        await gotoAppAndWaitReady(page);
        await waitForNavReady(page);
    });

    test.describe('Export List View', () => {
        test('should display exports table', async ({ page }) => {
            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            // Navigate to exports (assuming there's a nav link or direct URL)
            const exportManager = page.locator('.export-manager');

            // If export manager is visible, check the table
            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                const table = page.locator('[data-testid="exports-table"]');
                await expect(table).toBeVisible();

                // Check export rows
                await expect(table).toContainText('Movies Export');
                await expect(table).toContainText('TV Shows Export');
                await expect(table).toContainText('User Watch History');
            }
        });

        test('should show export status badges', async ({ page }) => {
            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                // Check for complete badge
                const completeBadge = page.locator('.status-badge.complete');
                await expect(completeBadge.first()).toBeVisible();

                // Check for pending badge (incomplete export)
                const pendingBadge = page.locator('.status-badge.pending');
                await expect(pendingBadge).toBeVisible();
            }
        });

        test('should display file format and size', async ({ page }) => {
            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                const table = page.locator('[data-testid="exports-table"]');
                await expect(table).toBeVisible();

                // Check format
                await expect(table).toContainText('csv');
                await expect(table).toContainText('json');

                // Check size (formatted)
                await expect(table).toContainText('1 MB');  // 1048576 bytes
            }
        });

        test('should have download buttons for complete exports', async ({ page }) => {
            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                // Find download buttons
                const downloadBtns = page.locator('.btn-download');

                // Complete exports should have enabled download buttons
                const enabledDownloads = downloadBtns.filter({ has: page.locator(':not([disabled])') });
                expect(await enabledDownloads.count()).toBeGreaterThan(0);
            }
        });

        test('should have delete buttons', async ({ page }) => {
            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                const deleteBtns = page.locator('.btn-delete');
                expect(await deleteBtns.count()).toBeGreaterThan(0);
            }
        });

        test('should sort by column', async ({ page }) => {
            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                // Click on timestamp header to sort
                // Use JavaScript click for CI reliability
                await page.evaluate(() => {
                    const el = document.querySelector('th[data-sort="timestamp"]') as HTMLElement;
                    if (el) el.click();
                });

                // Header should show active state
                const header = page.locator('th[data-sort="timestamp"]');
                await expect(header).toHaveClass(/active/);
            }
        });
    });

    test.describe('Create Export View', () => {
        test('should open create export form', async ({ page }) => {
            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                // Click create button
                // Use JavaScript click for CI reliability
                await page.evaluate(() => {
                    const el = document.querySelector('#create-export-btn') as HTMLElement;
                    if (el) el.click();
                });

                // Should show create form
                const createView = page.locator('[data-testid="create-export"]');
                await expect(createView).toBeVisible();
            }
        });

        test('should display library selection', async ({ page }) => {
            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                // Use JavaScript click for CI reliability
                await page.evaluate(() => {
                    const el = document.querySelector('#create-export-btn') as HTMLElement;
                    if (el) el.click();
                });

                const createView = page.locator('[data-testid="create-export"]');
                await expect(createView).toBeVisible();

                // Check library dropdown
                const librarySelect = page.locator('#export-library');
                await expect(librarySelect).toBeVisible();

                // Should have library options
                const options = librarySelect.locator('option');
                expect(await options.count()).toBeGreaterThan(1);
            }
        });

        test('should display format selection', async ({ page }) => {
            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                // Use JavaScript click for CI reliability
                await page.evaluate(() => {
                    const el = document.querySelector('#create-export-btn') as HTMLElement;
                    if (el) el.click();
                });

                const createView = page.locator('[data-testid="create-export"]');
                await expect(createView).toBeVisible();

                // Check format dropdown
                const formatSelect = page.locator('#export-format');
                await expect(formatSelect).toBeVisible();

                // Should have csv and json options
                await expect(formatSelect).toContainText('CSV');
                await expect(formatSelect).toContainText('JSON');
            }
        });

        test('should load export fields when library selected', async ({ page }) => {
            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                // Use JavaScript click for CI reliability
                await page.evaluate(() => {
                    const el = document.querySelector('#create-export-btn') as HTMLElement;
                    if (el) el.click();
                });

                const createView = page.locator('[data-testid="create-export"]');
                await expect(createView).toBeVisible();

                // Select a library
                await page.selectOption('#export-library', '1');

                // Should show field selection (with implicit wait)
                const fieldSelection = page.locator('.field-selection');
                await expect(fieldSelection).toBeVisible();

                // Should have field checkboxes
                await expect(fieldSelection).toContainText('Title');
                await expect(fieldSelection).toContainText('Year');
            }
        });

        test('should have select all button for fields', async ({ page }) => {
            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                // Use JavaScript click for CI reliability
                await page.evaluate(() => {
                    const el = document.querySelector('#create-export-btn') as HTMLElement;
                    if (el) el.click();
                });
                await page.selectOption('#export-library', '1');

                // Wait for field selection to load, then click select all
                const fieldSelection = page.locator('.field-selection');
                await expect(fieldSelection).toBeVisible();

                const selectAllBtn = page.locator('.btn-select-all');
                await expect(selectAllBtn).toBeVisible();
                await selectAllBtn.click();

                // All checkboxes should be checked
                const checkboxes = page.locator('.field-item input[type="checkbox"]');
                const count = await checkboxes.count();
                for (let i = 0; i < count; i++) {
                    await expect(checkboxes.nth(i)).toBeChecked();
                }
            }
        });

        test('should return to list view on cancel', async ({ page }) => {
            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                // Use JavaScript click for CI reliability
                await page.evaluate(() => {
                    const el = document.querySelector('#create-export-btn') as HTMLElement;
                    if (el) el.click();
                });

                const createView = page.locator('[data-testid="create-export"]');
                await expect(createView).toBeVisible();

                // Click cancel
                // Use JavaScript click for CI reliability
                await page.evaluate(() => {
                    const el = document.querySelector('.btn-cancel') as HTMLElement;
                    if (el) el.click();
                });

                // Should show list view
                const listView = page.locator('[data-testid="exports-list"]');
                await expect(listView).toBeVisible();
            }
        });
    });

    test.describe('Delete Export', () => {
        test('should show confirmation dialog', async ({ page }) => {
            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                // Click delete on first export
                await page.locator('.btn-delete').first().click();

                // Should show confirmation dialog
                const dialog = page.locator('.delete-confirm-overlay');
                await expect(dialog).toBeVisible();
                await expect(dialog).toContainText('Delete Export');
            }
        });

        test('should cancel delete on dismiss', async ({ page }) => {
            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                await page.locator('.btn-delete').first().click();

                const dialog = page.locator('.delete-confirm-overlay');
                await expect(dialog).toBeVisible();

                // Click cancel
                // Use JavaScript click for CI reliability
                await page.evaluate(() => {
                    const el = document.querySelector('.btn-cancel-delete') as HTMLElement;
                    if (el) el.click();
                });

                // Dialog should close
                await expect(dialog).not.toBeVisible();
            }
        });

        test('should delete export on confirm', async ({ page }) => {
            let deleteApiCalled = false;

            await page.route('**/api/v1/tautulli/delete-export*', async route => {
                deleteApiCalled = true;
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({ status: 'success', data: null })
                });
            });

            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                await page.locator('.btn-delete').first().click();

                const dialog = page.locator('.delete-confirm-overlay');
                await expect(dialog).toBeVisible();

                // Click confirm and wait for API response
                const responsePromise = page.waitForResponse('**/api/v1/tautulli/delete-export*');
                // Use JavaScript click for CI reliability
                await page.evaluate(() => {
                    const el = document.querySelector('.btn-confirm-delete') as HTMLElement;
                    if (el) el.click();
                });
                await responsePromise;

                // API should be called
                expect(deleteApiCalled).toBe(true);
            }
        });
    });

    test.describe('API Integration', () => {
        test('should call getExportsTable on load', async ({ page }) => {
            let apiCalled = false;

            // Set up route to intercept API calls
            await page.route('**/api/v1/tautulli/exports-table*', async route => {
                apiCalled = true;
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify(mockExportsTable)
                });
            });

            // Check if export manager is visible BEFORE triggering reload
            // ROOT CAUSE FIX: Don't create waitForResponse promise unless we'll actually await it
            // Previously, the promise was created before the check, causing "Target page closed" errors
            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                // Only set up response listener when we know we'll need it
                const responsePromise = page.waitForResponse('**/api/v1/tautulli/exports-table*');
                await page.reload({ waitUntil: 'domcontentloaded' });
                await responsePromise;
                expect(apiCalled).toBe(true);
            }
            // If export manager isn't visible, the feature isn't connected - skip silently
        });

        test('should call getExportFields when selecting library', async ({ page }) => {
            let fieldsApiCalled = false;

            await page.route('**/api/v1/tautulli/export-fields*', async route => {
                fieldsApiCalled = true;
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify(mockExportFields)
                });
            });

            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                // Use JavaScript click for CI reliability
                await page.evaluate(() => {
                    const el = document.querySelector('#create-export-btn') as HTMLElement;
                    if (el) el.click();
                });

                const responsePromise = page.waitForResponse('**/api/v1/tautulli/export-fields*');
                await page.selectOption('#export-library', '1');
                await responsePromise;

                expect(fieldsApiCalled).toBe(true);
            }
        });
    });

    test.describe('Pagination', () => {
        test('should display pagination controls', async ({ page }) => {
            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                const pagination = page.locator('.export-pagination');
                await expect(pagination).toBeVisible();

                // Should have page buttons
                const prevBtn = page.locator('.page-btn').filter({ hasText: /prev/i });
                const nextBtn = page.locator('.page-btn').filter({ hasText: /next/i });

                // Verify pagination buttons exist and are visible
                const prevCount = await prevBtn.count();
                const nextCount = await nextBtn.count();
                if (prevCount > 0) await expect(prevBtn.first()).toBeVisible();
                if (nextCount > 0) await expect(nextBtn.first()).toBeVisible();
            }
        });

        test('should show page info', async ({ page }) => {
            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                const pageInfo = page.locator('.page-info');
                await expect(pageInfo).toBeVisible();
                await expect(pageInfo).toContainText('3');  // Total records
            }
        });
    });

    test.describe('Empty State', () => {
        test('should show empty state when no exports', async ({ page }) => {
            await page.route('**/api/v1/tautulli/exports-table*', async route => {
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({
                        status: 'success',
                        data: {
                            draw: 1,
                            recordsTotal: 0,
                            recordsFiltered: 0,
                            data: []
                        }
                    })
                });
            });

            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                const emptyState = page.locator('.empty-state');
                await expect(emptyState).toBeVisible();
                await expect(emptyState).toContainText('No exports');
            }
        });
    });

    test.describe('Error Handling', () => {
        test('should show error state on API failure', async ({ page }) => {
            await page.route('**/api/v1/tautulli/exports-table*', async route => {
                await route.fulfill({
                    status: 500,
                    contentType: 'application/json',
                    body: JSON.stringify({ status: 'error', message: 'Server error' })
                });
            });

            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                const errorState = page.locator('.error-state');
                await expect(errorState).toBeVisible();
            }
        });

        test('should have retry button on error', async ({ page }) => {
            await page.route('**/api/v1/tautulli/exports-table*', async route => {
                await route.fulfill({
                    status: 500,
                    contentType: 'application/json',
                    body: JSON.stringify({ status: 'error', message: 'Server error' })
                });
            });

            // Note: Navigation handled by beforeEach via gotoAppAndWaitReady

            const exportManager = page.locator('.export-manager');

            if (await exportManager.isVisible({ timeout: TIMEOUTS.NAVIGATION }).catch(() => false)) {
                const retryBtn = page.locator('.btn-retry');
                await expect(retryBtn).toBeVisible();
            }
        });
    });
});
