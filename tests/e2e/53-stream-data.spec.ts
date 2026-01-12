// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Tests for Stream Data Details
 *
 * Tests for:
 * - Stream data form
 * - Transcode decisions display
 * - Video/Audio details
 * - Stream output section
 * - Status indicators
 *
 * WIRED: StreamDataManager is connected to the main application
 * Navigation: Analytics -> Tautulli -> Stream Data sub-tab
 */

import {
    test,
    expect,
    TIMEOUTS,
    TAUTULLI_SUB_TABS,
    gotoAppAndWaitReady,
    navigateToTautulliSubTab,
} from './fixtures';

test.describe('Stream Data Details', () => {
    const mockStreamData = {
        status: 'success',
        data: {
            session_key: 'test-session-123',
            transcode_decision: 'transcode',
            video_decision: 'transcode',
            audio_decision: 'copy',
            subtitle_decision: 'burn',
            container: 'mkv',
            bitrate: 15000,
            video_codec: 'hevc',
            video_resolution: '4k',
            video_width: 3840,
            video_height: 2160,
            video_framerate: '24p',
            video_bitrate: 12000,
            audio_codec: 'truehd',
            audio_channels: 8,
            audio_channel_layout: '7.1',
            audio_bitrate: 3000,
            audio_sample_rate: 48000,
            stream_container: 'mpegts',
            stream_container_decision: 'transcode',
            stream_bitrate: 8000,
            stream_video_codec: 'h264',
            stream_video_resolution: '1080p',
            stream_video_bitrate: 6000,
            stream_video_width: 1920,
            stream_video_height: 1080,
            stream_video_framerate: '24p',
            stream_audio_codec: 'aac',
            stream_audio_channels: 2,
            stream_audio_bitrate: 256,
            stream_audio_sample_rate: 48000,
            subtitle_codec: 'srt',
            optimized: 0,
            throttled: 0
        },
        metadata: { timestamp: new Date().toISOString() }
    };

    const mockEmptyStreamData = {
        status: 'success',
        data: {
            session_key: '',
            transcode_decision: '',
            video_decision: '',
            audio_decision: '',
            container: '',
            bitrate: 0,
            video_codec: '',
            video_resolution: '',
            video_width: 0,
            video_height: 0,
            video_framerate: '',
            video_bitrate: 0,
            audio_codec: '',
            audio_channels: 0,
            audio_bitrate: 0,
            audio_sample_rate: 0
        },
        metadata: { timestamp: new Date().toISOString() }
    };

    test.beforeEach(async ({ page }) => {
        // Feature-specific routes for stream data functionality
        await page.route('**/api/v1/tautulli/stream-data*', async route => {
            const url = new URL(route.request().url());
            const sessionKey = url.searchParams.get('session_key');

            if (sessionKey === 'test-session-123') {
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify(mockStreamData)
                });
            } else if (sessionKey === 'empty') {
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify(mockEmptyStreamData)
                });
            } else {
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify(mockStreamData)
                });
            }
        });

        // Navigate to app and then to the Stream Data sub-tab
        await gotoAppAndWaitReady(page);
        await navigateToTautulliSubTab(page, TAUTULLI_SUB_TABS.STREAM_DATA);

        // Verify the manager container is visible
        await expect(page.locator('[data-testid="stream-data-manager"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test.describe('Form', () => {
        test('should display stream data form', async ({ page }) => {
            const sessionInput = page.locator('#session-key-input');
            const rowIdInput = page.locator('#row-id-input');
            const loadBtn = page.locator('#load-stream-btn');

            await expect(sessionInput).toBeVisible();
            await expect(rowIdInput).toBeVisible();
            await expect(loadBtn).toBeVisible();
        });

        test('should have placeholder text', async ({ page }) => {
            const sessionInput = page.locator('#session-key-input');
            await expect(sessionInput).toHaveAttribute('placeholder', /session key/i);
        });
    });

    test.describe('Loading Stream Data', () => {
        test('should load stream data by session key', async ({ page }) => {
            await page.fill('#session-key-input', 'test-session-123');
            await page.click('#load-stream-btn');

            const results = page.locator('[data-testid="stream-data-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
        });

        test('should load stream data by row ID', async ({ page }) => {
            await page.fill('#row-id-input', '12345');
            await page.click('#load-stream-btn');

            const results = page.locator('[data-testid="stream-data-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
        });
    });

    test.describe('Transcode Decisions', () => {
        test('should display transcode decision', async ({ page }) => {
            await page.fill('#session-key-input', 'test-session-123');
            await page.click('#load-stream-btn');

            const results = page.locator('[data-testid="stream-data-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(results).toContainText('transcode');
        });

        test('should display video decision', async ({ page }) => {
            await page.fill('#session-key-input', 'test-session-123');
            await page.click('#load-stream-btn');

            const results = page.locator('[data-testid="stream-data-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(results).toContainText('Video');
        });

        test('should display audio decision', async ({ page }) => {
            await page.fill('#session-key-input', 'test-session-123');
            await page.click('#load-stream-btn');

            const results = page.locator('[data-testid="stream-data-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(results).toContainText('Audio');
        });
    });

    test.describe('Video Details', () => {
        test('should display video codec', async ({ page }) => {
            await page.fill('#session-key-input', 'test-session-123');
            await page.click('#load-stream-btn');

            const results = page.locator('[data-testid="stream-data-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(results).toContainText('hevc');
        });

        test('should display video resolution', async ({ page }) => {
            await page.fill('#session-key-input', 'test-session-123');
            await page.click('#load-stream-btn');

            const results = page.locator('[data-testid="stream-data-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(results).toContainText('4k');
        });

        test('should display video dimensions', async ({ page }) => {
            await page.fill('#session-key-input', 'test-session-123');
            await page.click('#load-stream-btn');

            const results = page.locator('[data-testid="stream-data-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(results).toContainText('3840');
        });

        test('should display video bitrate', async ({ page }) => {
            await page.fill('#session-key-input', 'test-session-123');
            await page.click('#load-stream-btn');

            const results = page.locator('[data-testid="stream-data-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(results).toContainText('12000');
        });
    });

    test.describe('Audio Details', () => {
        test('should display audio codec', async ({ page }) => {
            await page.fill('#session-key-input', 'test-session-123');
            await page.click('#load-stream-btn');

            const results = page.locator('[data-testid="stream-data-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(results).toContainText('truehd');
        });

        test('should display audio channels', async ({ page }) => {
            await page.fill('#session-key-input', 'test-session-123');
            await page.click('#load-stream-btn');

            const results = page.locator('[data-testid="stream-data-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(results).toContainText('7.1');
        });
    });

    test.describe('Stream Output', () => {
        test('should display stream container', async ({ page }) => {
            await page.fill('#session-key-input', 'test-session-123');
            await page.click('#load-stream-btn');

            const results = page.locator('[data-testid="stream-data-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(results).toContainText('mpegts');
        });

        test('should display stream video codec', async ({ page }) => {
            await page.fill('#session-key-input', 'test-session-123');
            await page.click('#load-stream-btn');

            const results = page.locator('[data-testid="stream-data-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(results).toContainText('h264');
        });

        test('should display stream resolution', async ({ page }) => {
            await page.fill('#session-key-input', 'test-session-123');
            await page.click('#load-stream-btn');

            const results = page.locator('[data-testid="stream-data-results"]');
            await expect(results).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            await expect(results).toContainText('1080p');
        });
    });

    test.describe('API Integration', () => {
        test('should call stream-data API with session key', async ({ page }) => {
            let apiCalled = false;
            let sessionKey = '';

            await page.route('**/api/v1/tautulli/stream-data*', async route => {
                apiCalled = true;
                const url = new URL(route.request().url());
                sessionKey = url.searchParams.get('session_key') || '';
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify(mockStreamData)
                });
            });

            await page.fill('#session-key-input', 'test-session-123');
            await page.click('#load-stream-btn');

            await expect(page.locator('[data-testid="stream-data-results"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

            expect(apiCalled).toBe(true);
            expect(sessionKey).toBe('test-session-123');
        });

        test('should call stream-data API with row ID', async ({ page }) => {
            let apiCalled = false;
            let rowId = '';

            await page.route('**/api/v1/tautulli/stream-data*', async route => {
                apiCalled = true;
                const url = new URL(route.request().url());
                rowId = url.searchParams.get('row_id') || '';
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify(mockStreamData)
                });
            });

            await page.fill('#row-id-input', '12345');
            await page.click('#load-stream-btn');

            await expect(page.locator('[data-testid="stream-data-results"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

            expect(apiCalled).toBe(true);
            expect(rowId).toBe('12345');
        });
    });

    test.describe('Error Handling', () => {
        test('should show error state on API failure', async ({ page }) => {
            await page.route('**/api/v1/tautulli/stream-data*', async route => {
                await route.fulfill({
                    status: 500,
                    contentType: 'application/json',
                    body: JSON.stringify({ status: 'error', message: 'Server error' })
                });
            });

            await page.fill('#session-key-input', 'test-session-123');
            await page.click('#load-stream-btn');

            const errorState = page.locator('.error-message');
            await expect(errorState).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
        });
    });

    test.describe('Keyboard Support', () => {
        test('should load on Enter key in session input', async ({ page }) => {
            let apiCalled = false;

            await page.route('**/api/v1/tautulli/stream-data*', async route => {
                apiCalled = true;
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify(mockStreamData)
                });
            });

            const input = page.locator('#session-key-input');
            await input.fill('test-session-123');
            await input.press('Enter');

            await expect(page.locator('[data-testid="stream-data-results"]')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
            expect(apiCalled).toBe(true);
        });
    });
});
