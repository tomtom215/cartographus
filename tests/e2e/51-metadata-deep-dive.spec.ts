// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Tests for Metadata Deep-Dive
 *
 * Tests for:
 * - Metadata search
 * - Basic info display
 * - Ratings display
 * - Credits display
 * - Media info display
 *
 * WIRED: MetadataDeepDiveManager is connected to the main application
 * Navigation: Analytics -> Tautulli -> Metadata sub-tab
 */

import {
    test,
    expect,
    TIMEOUTS,
    TAUTULLI_SUB_TABS,
    gotoAppAndWaitReady,
    navigateToTautulliSubTab,
} from './fixtures';

test.describe('Metadata Deep-Dive', () => {
    const mockMetadata = {
        status: 'success',
        data: {
            rating_key: '12345',
            parent_rating_key: '',
            grandparent_rating_key: '',
            title: 'Test Movie',
            parent_title: '',
            grandparent_title: '',
            original_title: 'Test Movie Original',
            sort_title: 'Test Movie',
            media_index: 0,
            parent_media_index: 0,
            studio: 'Test Studios',
            content_rating: 'PG-13',
            summary: 'This is a test movie with a great story.',
            tagline: 'An adventure like no other',
            rating: 8.5,
            rating_image: '',
            audience_rating: 9.2,
            audience_rating_image: '',
            user_rating: 10.0,
            duration: 7200000,
            year: 2023,
            thumb: '/path/to/thumb.jpg',
            parent_thumb: '',
            grandparent_thumb: '',
            art: '/path/to/art.jpg',
            banner: '',
            originally_available_at: '2023-06-15',
            added_at: 1702400000,
            updated_at: 1702500000,
            last_viewed_at: 1702600000,
            guid: 'plex://movie/12345',
            guids: ['imdb://tt1234567', 'tmdb://12345'],
            directors: ['John Director', 'Jane Director'],
            writers: ['Bob Writer', 'Alice Writer'],
            actors: ['Tom Actor', 'Sarah Actress', 'Mike Star', 'Lisa Lead'],
            genres: ['Action', 'Adventure', 'Sci-Fi'],
            labels: ['4K', 'HDR'],
            collections: ['Test Collection'],
            media_info: [
                {
                    id: 1,
                    container: 'mkv',
                    bitrate: 15000,
                    height: 2160,
                    width: 3840,
                    aspect_ratio: 1.78,
                    video_codec: 'hevc',
                    video_resolution: '4K',
                    video_framerate: '24p',
                    video_bit_depth: 10,
                    video_profile: 'main 10',
                    audio_codec: 'truehd',
                    audio_channels: 8,
                    audio_channel_layout: '7.1',
                    audio_bitrate: 2500,
                    optimized_version: 0
                }
            ]
        },
        metadata: { timestamp: new Date().toISOString() }
    };

    test.beforeEach(async ({ page }) => {
        // Feature-specific route for metadata API
        // IMPORTANT: Must include /v1/ in the path to match actual API endpoints
        await page.route('**/api/v1/tautulli/metadata*', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify(mockMetadata)
            });
        });

        // Navigate to app and then to the Metadata sub-tab
        await gotoAppAndWaitReady(page);
        await navigateToTautulliSubTab(page, TAUTULLI_SUB_TABS.METADATA);

        // Verify the manager container is visible
        await expect(page.locator('[data-testid="metadata-deep-dive"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test.describe('Search Form', () => {
        test('should display search input', async ({ page }) => {
            const input = page.locator('#metadata-rating-key');
            await expect(input).toBeVisible();
        });

        test('should have search button', async ({ page }) => {
            const button = page.locator('#metadata-search-btn');
            await expect(button).toBeVisible();
            await expect(button).toContainText('Load Metadata');
        });
    });

    test.describe('Basic Information', () => {
        test('should display title', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const basicInfo = page.locator('[data-testid="basic-info"]');
            await expect(basicInfo).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(basicInfo).toContainText('Test Movie');
        });

        test('should display year', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const basicInfo = page.locator('[data-testid="basic-info"]');
            await expect(basicInfo).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(basicInfo).toContainText('2023');
        });

        test('should display studio', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const basicInfo = page.locator('[data-testid="basic-info"]');
            await expect(basicInfo).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(basicInfo).toContainText('Test Studios');
        });

        test('should display content rating', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const basicInfo = page.locator('[data-testid="basic-info"]');
            await expect(basicInfo).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(basicInfo).toContainText('PG-13');
        });
    });

    test.describe('Ratings', () => {
        test('should display critic rating', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const ratings = page.locator('[data-testid="ratings"]');
            await expect(ratings).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(ratings).toContainText('8.5');
        });

        test('should display audience rating', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const ratings = page.locator('[data-testid="ratings"]');
            await expect(ratings).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(ratings).toContainText('9.2');
        });

        test('should display user rating', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const ratings = page.locator('[data-testid="ratings"]');
            await expect(ratings).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(ratings).toContainText('10');
        });
    });

    test.describe('Summary', () => {
        test('should display tagline', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const summary = page.locator('[data-testid="summary"]');
            await expect(summary).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(summary).toContainText('An adventure like no other');
        });

        test('should display description', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const summary = page.locator('[data-testid="summary"]');
            await expect(summary).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(summary).toContainText('great story');
        });
    });

    test.describe('Credits', () => {
        test('should display directors', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const credits = page.locator('[data-testid="credits"]');
            await expect(credits).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(credits).toContainText('John Director');
        });

        test('should display writers', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const credits = page.locator('[data-testid="credits"]');
            await expect(credits).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(credits).toContainText('Bob Writer');
        });

        test('should display actors', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const credits = page.locator('[data-testid="credits"]');
            await expect(credits).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(credits).toContainText('Tom Actor');
        });
    });

    test.describe('Categories', () => {
        test('should display genres', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const categories = page.locator('[data-testid="categories"]');
            await expect(categories).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(categories).toContainText('Action');
        });

        test('should display collections', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const categories = page.locator('[data-testid="categories"]');
            await expect(categories).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(categories).toContainText('Test Collection');
        });

        test('should display labels', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const categories = page.locator('[data-testid="categories"]');
            await expect(categories).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(categories).toContainText('4K');
        });
    });

    test.describe('Media Info', () => {
        test('should display resolution', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const mediaInfo = page.locator('[data-testid="media-info"]');
            await expect(mediaInfo).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(mediaInfo).toContainText('4K');
        });

        test('should display video codec', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const mediaInfo = page.locator('[data-testid="media-info"]');
            await expect(mediaInfo).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(mediaInfo).toContainText('hevc');
        });

        test('should display audio codec', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const mediaInfo = page.locator('[data-testid="media-info"]');
            await expect(mediaInfo).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(mediaInfo).toContainText('truehd');
        });

        test('should display container format', async ({ page }) => {
            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const mediaInfo = page.locator('[data-testid="media-info"]');
            await expect(mediaInfo).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(mediaInfo).toContainText('mkv');
        });
    });

    test.describe('API Integration', () => {
        test('should call metadata API with rating key', async ({ page }) => {
            let apiCalled = false;
            let apiRatingKey = '';

            await page.route('**/api/v1/tautulli/metadata*', async route => {
                apiCalled = true;
                const url = new URL(route.request().url());
                apiRatingKey = url.searchParams.get('rating_key') || '';
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify(mockMetadata)
                });
            });

            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            await expect(page.locator('[data-testid="basic-info"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

            expect(apiCalled).toBe(true);
            expect(apiRatingKey).toBe('12345');
        });
    });

    test.describe('Error Handling', () => {
        test('should show error state on API failure', async ({ page }) => {
            await page.route('**/api/v1/tautulli/metadata*', async route => {
                await route.fulfill({
                    status: 500,
                    contentType: 'application/json',
                    body: JSON.stringify({ status: 'error', message: 'Server error' })
                });
            });

            await page.fill('#metadata-rating-key', '12345');
            await page.click('#metadata-search-btn');

            const errorState = page.locator('.error-message');
            await expect(errorState).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
        });
    });
});
