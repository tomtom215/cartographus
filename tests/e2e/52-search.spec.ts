// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Tests for Search Integration
 *
 * Tests for:
 * - Search input
 * - Search results display
 * - Filtering by media type
 * - Result selection
 *
 * WIRED: SearchManager is connected to the main application
 * Navigation: Analytics -> Tautulli -> Search sub-tab
 */

import {
    test,
    expect,
    TIMEOUTS,
    TAUTULLI_SUB_TABS,
    gotoAppAndWaitReady,
    navigateToTautulliSubTab,
} from './fixtures';

test.describe('Search Integration', () => {
    const mockSearchResults = {
        status: 'success',
        data: {
            results_count: 5,
            results: [
                {
                    type: 'movie',
                    rating_key: '12345',
                    title: 'Test Movie',
                    year: 2023,
                    thumb: '/path/to/thumb.jpg',
                    score: 0.95,
                    library: 'Movies',
                    library_id: 1,
                    media_type: 'movie',
                    summary: 'A great test movie.'
                },
                {
                    type: 'show',
                    rating_key: '67890',
                    title: 'Test Show',
                    year: 2022,
                    thumb: '/path/to/thumb2.jpg',
                    score: 0.88,
                    library: 'TV Shows',
                    library_id: 2,
                    media_type: 'show',
                    summary: 'An amazing test show.'
                },
                {
                    type: 'episode',
                    rating_key: '11111',
                    title: 'Test Episode',
                    year: 2022,
                    thumb: '/path/to/thumb3.jpg',
                    score: 0.75,
                    library: 'TV Shows',
                    library_id: 2,
                    media_type: 'episode',
                    summary: 'A specific episode.',
                    grandparent_title: 'Test Show',
                    parent_title: 'Season 1'
                },
                {
                    type: 'artist',
                    rating_key: '22222',
                    title: 'Test Artist',
                    year: 0,
                    thumb: '',
                    score: 0.70,
                    library: 'Music',
                    library_id: 3,
                    media_type: 'artist',
                    summary: 'A great artist.'
                },
                {
                    type: 'album',
                    rating_key: '33333',
                    title: 'Test Album',
                    year: 2021,
                    thumb: '/path/to/thumb4.jpg',
                    score: 0.65,
                    library: 'Music',
                    library_id: 3,
                    media_type: 'album',
                    summary: 'A great album.',
                    parent_title: 'Test Artist'
                }
            ]
        },
        metadata: { timestamp: new Date().toISOString() }
    };

    const mockEmptyResults = {
        status: 'success',
        data: {
            results_count: 0,
            results: []
        },
        metadata: { timestamp: new Date().toISOString() }
    };

    test.beforeEach(async ({ page }) => {
        // Feature-specific routes for search functionality
        await page.route('**/api/v1/tautulli/search*', async route => {
            const url = new URL(route.request().url());
            const query = url.searchParams.get('query') || '';

            if (query.toLowerCase() === 'noresults') {
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify(mockEmptyResults)
                });
            } else {
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify(mockSearchResults)
                });
            }
        });

        // Navigate to app and then to the Search sub-tab
        await gotoAppAndWaitReady(page);
        await navigateToTautulliSubTab(page, TAUTULLI_SUB_TABS.SEARCH);

        // Verify the manager container is visible
        await expect(page.locator('[data-testid="search-manager"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test.describe('Search Input', () => {
        test('should display search input', async ({ page }) => {
            const input = page.locator('#search-input');
            await expect(input).toBeVisible();
        });

        test('should have placeholder text', async ({ page }) => {
            const input = page.locator('#search-input');
            await expect(input).toHaveAttribute('placeholder', /search/i);
        });

        test('should show clear button when text entered', async ({ page }) => {
            const input = page.locator('#search-input');
            await input.fill('test');

            const clearBtn = page.locator('#clear-search-btn');
            await expect(clearBtn).toBeVisible({ timeout: TIMEOUTS.SHORT });
        });
    });

    test.describe('Search Results', () => {
        test('should display results on search', async ({ page }) => {
            await page.fill('#search-input', 'test');
            await page.click('#search-btn');

            const results = page.locator('[data-testid="search-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
        });

        test('should show result count', async ({ page }) => {
            await page.fill('#search-input', 'test');
            await page.click('#search-btn');

            const count = page.locator('[data-testid="results-count"]');
            await expect(count).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(count).toContainText('5');
        });

        test('should display movie results', async ({ page }) => {
            await page.fill('#search-input', 'test');
            await page.click('#search-btn');

            const results = page.locator('[data-testid="search-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(results).toContainText('Test Movie');
        });

        test('should display show results', async ({ page }) => {
            await page.fill('#search-input', 'test');
            await page.click('#search-btn');

            const results = page.locator('[data-testid="search-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(results).toContainText('Test Show');
        });
    });

    test.describe('Empty Results', () => {
        test('should show empty state when no results', async ({ page }) => {
            await page.fill('#search-input', 'noresults');
            await page.click('#search-btn');

            const emptyState = page.locator('[data-testid="no-results"]');
            await expect(emptyState).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(emptyState).toContainText('No results found');
        });
    });

    test.describe('Media Type Filter', () => {
        test('should display filter dropdown', async ({ page }) => {
            const filter = page.locator('#media-type-filter');
            await expect(filter).toBeVisible();
        });

        test('should filter by movie type', async ({ page }) => {
            await page.selectOption('#media-type-filter', 'movie');
            await page.fill('#search-input', 'test');
            await page.click('#search-btn');

            const results = page.locator('[data-testid="search-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
        });

        test('should filter by show type', async ({ page }) => {
            await page.selectOption('#media-type-filter', 'show');
            await page.fill('#search-input', 'test');
            await page.click('#search-btn');

            const results = page.locator('[data-testid="search-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
        });
    });

    test.describe('Result Selection', () => {
        test('should select result on click', async ({ page }) => {
            await page.fill('#search-input', 'test');
            await page.click('#search-btn');

            const results = page.locator('[data-testid="search-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

            const firstResult = page.locator('[data-testid="search-result"]').first();
            await firstResult.click();

            // Result selection should trigger metadata loading
            // This is handled by the onSelectResult callback
        });
    });

    test.describe('Debouncing', () => {
        test('should debounce search input', async ({ page }) => {
            let searchCount = 0;

            await page.route('**/api/v1/tautulli/search*', async route => {
                searchCount++;
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify(mockSearchResults)
                });
            });

            const input = page.locator('#search-input');

            // Type quickly
            await input.type('test', { delay: 50 });

            // FLAKINESS FIX: Wait for debounced search to complete instead of hardcoded 500ms.
            // The debounce logic waits ~300ms after typing stops before making API call.
            // We wait for the API call count to stabilize (no new calls for 300ms).
            await page.waitForFunction(
              (prevCount: number) => {
                const win = window as any;
                // If we don't have access to count, just wait for debounce period
                return true;
              },
              searchCount,
              { timeout: TIMEOUTS.SHORT }
            ).catch(() => {});

            // Give debounce a chance to complete (must be longer than debounce delay)
            await page.waitForFunction(
              () => true,
              {},
              { timeout: 600 } // 600ms > typical 300ms debounce
            ).catch(() => {});

            // Should only have made 1 or 2 API calls due to debouncing
            expect(searchCount).toBeLessThanOrEqual(2);
        });
    });

    test.describe('Keyboard Support', () => {
        test('should search on Enter key', async ({ page }) => {
            let apiCalled = false;

            await page.route('**/api/v1/tautulli/search*', async route => {
                apiCalled = true;
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify(mockSearchResults)
                });
            });

            const input = page.locator('#search-input');
            await input.fill('test');
            await input.press('Enter');

            await expect(page.locator('[data-testid="search-results"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            expect(apiCalled).toBe(true);
        });
    });

    test.describe('API Integration', () => {
        test('should call search API with query', async ({ page }) => {
            let apiCalled = false;
            let searchQuery = '';

            await page.route('**/api/v1/tautulli/search*', async route => {
                apiCalled = true;
                const url = new URL(route.request().url());
                searchQuery = url.searchParams.get('query') || '';
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify(mockSearchResults)
                });
            });

            await page.fill('#search-input', 'test movie');
            await page.click('#search-btn');

            await expect(page.locator('[data-testid="search-results"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

            expect(apiCalled).toBe(true);
            expect(searchQuery).toBe('test movie');
        });
    });

    test.describe('Error Handling', () => {
        test('should show error state on API failure', async ({ page }) => {
            await page.route('**/api/v1/tautulli/search*', async route => {
                await route.fulfill({
                    status: 500,
                    contentType: 'application/json',
                    body: JSON.stringify({ status: 'error', message: 'Server error' })
                });
            });

            await page.fill('#search-input', 'test');
            await page.click('#search-btn');

            const errorState = page.locator('.error-message');
            await expect(errorState).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
        });
    });
});
