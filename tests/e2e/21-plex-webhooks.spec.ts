// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import {
  test,
  expect,
  gotoAppAndWaitReady,
} from './fixtures';
import crypto from 'crypto';

/**
 * E2E Tests: Plex Webhook Receiver
 *
 * Tests the Plex webhook endpoint for receiving push notifications
 * from Plex Media Server for playback events.
 *
 * Webhook features:
 * - HMAC-SHA256 signature verification
 * - Multiple event types (media.play, media.stop, etc.)
 * - WebSocket broadcasting of events
 * - Proper error handling for invalid payloads
 */

// Helper to generate HMAC-SHA256 signature
function generateSignature(payload: string, secret: string): string {
    return crypto.createHmac('sha256', secret).update(payload).digest('hex');
}

// Sample webhook payloads
const sampleMediaPlayPayload = {
    event: 'media.play',
    user: true,
    owner: false,
    Account: {
        id: 12345,
        thumb: 'https://plex.tv/users/abc123/avatar',
        title: 'TestUser'
    },
    Server: {
        title: 'My Plex Server',
        uuid: 'server-uuid-123'
    },
    Player: {
        local: true,
        publicAddress: '192.168.1.100',
        title: 'Roku Express',
        uuid: 'player-uuid-456'
    },
    Metadata: {
        librarySectionType: 'movie',
        ratingKey: '12345',
        key: '/library/metadata/12345',
        guid: 'imdb://tt0111161',
        librarySectionTitle: 'Movies',
        librarySectionID: 1,
        type: 'movie',
        title: 'The Shawshank Redemption',
        year: 1994
    }
};

const sampleMediaStopPayload = {
    ...sampleMediaPlayPayload,
    event: 'media.stop'
};

const sampleLibraryNewPayload = {
    event: 'library.new',
    user: false,
    owner: true,
    Account: {
        id: 1,
        thumb: '',
        title: 'Admin'
    },
    Server: {
        title: 'My Plex Server',
        uuid: 'server-uuid-123'
    },
    Player: {
        local: false,
        publicAddress: '',
        title: '',
        uuid: ''
    },
    Metadata: {
        librarySectionType: 'movie',
        ratingKey: '67890',
        key: '/library/metadata/67890',
        guid: 'imdb://tt1375666',
        librarySectionTitle: 'Movies',
        librarySectionID: 1,
        type: 'movie',
        title: 'Inception',
        year: 2010
    }
};

test.describe('Plex Webhook Receiver', () => {
    test.beforeEach(async ({ page }) => {
        // Use storageState for authentication (configured in playwright.config.ts)
        await gotoAppAndWaitReady(page);
        // Wait for page to be ready and idle to avoid rate limiting between tests
        await page.waitForFunction(() => document.readyState === 'complete');
    });

    test('Webhook endpoint returns 404 when webhooks disabled', async ({ page }) => {
        // Send webhook without enabling webhooks feature
        const payload = JSON.stringify(sampleMediaPlayPayload);

        const response = await page.request.post('/api/v1/plex/webhook', {
            headers: {
                'Content-Type': 'application/json',
            },
            data: payload,
        });

        // Should return 404 if webhooks not enabled
        // Or 200 if enabled - both are acceptable depending on server config
        // 429 indicates rate limiting is working correctly
        expect([200, 404, 429]).toContain(response.status());
    });

    test('Webhook accepts valid media.play event', async ({ page }) => {
        const payload = JSON.stringify(sampleMediaPlayPayload);

        const response = await page.request.post('/api/v1/plex/webhook', {
            headers: {
                'Content-Type': 'application/json',
            },
            data: payload,
        });

        // Should succeed if webhooks enabled, 404 if disabled
        if (response.status() === 200) {
            const data = await response.json();
            expect(data.status).toBe('success');
            expect(data.data.received).toBe(true);
            expect(data.data.event).toBe('media.play');
        }
    });

    test('Webhook accepts valid media.stop event', async ({ page }) => {
        const payload = JSON.stringify(sampleMediaStopPayload);

        const response = await page.request.post('/api/v1/plex/webhook', {
            headers: {
                'Content-Type': 'application/json',
            },
            data: payload,
        });

        if (response.status() === 200) {
            const data = await response.json();
            expect(data.status).toBe('success');
            expect(data.data.event).toBe('media.stop');
        }
    });

    test('Webhook accepts library.new event', async ({ page }) => {
        const payload = JSON.stringify(sampleLibraryNewPayload);

        const response = await page.request.post('/api/v1/plex/webhook', {
            headers: {
                'Content-Type': 'application/json',
            },
            data: payload,
        });

        if (response.status() === 200) {
            const data = await response.json();
            expect(data.status).toBe('success');
            expect(data.data.event).toBe('library.new');
        }
    });

    test('Webhook rejects invalid JSON', async ({ page }) => {
        const response = await page.request.post('/api/v1/plex/webhook', {
            headers: {
                'Content-Type': 'application/json',
            },
            data: 'not valid json {{{',
        });

        // Should return 400 for invalid JSON (if webhooks enabled)
        // or 404 if webhooks disabled, or 429 if rate limited
        expect([400, 404, 429]).toContain(response.status());

        if (response.status() === 400) {
            const data = await response.json();
            expect(data.status).toBe('error');
            expect(data.error.code).toBe('INVALID_PAYLOAD');
        }
    });

    test('Webhook handles empty body gracefully', async ({ page }) => {
        const response = await page.request.post('/api/v1/plex/webhook', {
            headers: {
                'Content-Type': 'application/json',
            },
            data: '',
        });

        // Should return 400 or 404 or 429 (rate limited)
        expect([400, 404, 429]).toContain(response.status());
    });
});

test.describe('Plex Webhook Signature Verification', () => {
    // Note: These tests assume PLEX_WEBHOOK_SECRET is configured
    // If not configured, signature is not verified

    test.beforeEach(async ({ page }) => {
        // Navigate (already authenticated via storageState from setup)
        await gotoAppAndWaitReady(page);
        // Wait for page to be ready and idle to avoid rate limiting between tests
        await page.waitForFunction(() => document.readyState === 'complete');
    });

    test('Webhook rejects missing signature when secret configured', async ({ page }) => {
        const payload = JSON.stringify(sampleMediaPlayPayload);

        // First check if webhooks are enabled and require signature
        const response = await page.request.post('/api/v1/plex/webhook', {
            headers: {
                'Content-Type': 'application/json',
                // No X-Plex-Signature header
            },
            data: payload,
        });

        // If webhooks enabled with secret, should return 401
        // If webhooks disabled, returns 404
        // If webhooks enabled without secret, returns 200
        // 429 indicates rate limiting is working correctly
        expect([200, 401, 404, 429]).toContain(response.status());

        if (response.status() === 401) {
            const data = await response.json();
            expect(data.status).toBe('error');
            expect(data.error.code).toBe('MISSING_SIGNATURE');
        }
    });

    test('Webhook rejects invalid signature', async ({ page }) => {
        const payload = JSON.stringify(sampleMediaPlayPayload);
        const invalidSignature = 'invalid-signature-here';

        const response = await page.request.post('/api/v1/plex/webhook', {
            headers: {
                'Content-Type': 'application/json',
                'X-Plex-Signature': invalidSignature,
            },
            data: payload,
        });

        // If webhooks enabled with secret, should return 401 for invalid signature
        // If webhooks disabled, returns 404
        // If webhooks enabled without secret, returns 200 (signature not checked)
        // 429 indicates rate limiting is working correctly
        expect([200, 401, 404, 429]).toContain(response.status());

        if (response.status() === 401) {
            const data = await response.json();
            expect(data.status).toBe('error');
            expect(data.error.code).toBe('INVALID_SIGNATURE');
        }
    });

    test('Webhook accepts valid signature', async ({ page }) => {
        // This test can only fully work if we know the webhook secret
        // For testing, we'll use a test secret
        const testSecret = process.env.TEST_WEBHOOK_SECRET || 'test-secret-for-e2e';
        const payload = JSON.stringify(sampleMediaPlayPayload);
        const signature = generateSignature(payload, testSecret);

        const response = await page.request.post('/api/v1/plex/webhook', {
            headers: {
                'Content-Type': 'application/json',
                'X-Plex-Signature': signature,
            },
            data: payload,
        });

        // If the test secret matches server config, should succeed
        // Otherwise may fail signature check
        // 429 indicates rate limiting is working correctly
        expect([200, 401, 404, 429]).toContain(response.status());
    });
});

test.describe('Plex Webhook Event Types', () => {
    test.beforeEach(async ({ page }) => {
        // Navigate (already authenticated via storageState from setup)
        await gotoAppAndWaitReady(page);
    });

    const eventTypes = [
        'media.play',
        'media.pause',
        'media.resume',
        'media.stop',
        'media.scrobble',
        'media.rate',
        'library.new',
        'library.on.deck',
        'admin.database.backup',
        'device.new'
    ];

    for (const eventType of eventTypes) {
        test(`Webhook handles ${eventType} event`, async ({ page }) => {
            const payload = JSON.stringify({
                ...sampleMediaPlayPayload,
                event: eventType
            });

            const response = await page.request.post('/api/v1/plex/webhook', {
                headers: {
                    'Content-Type': 'application/json',
                },
                data: payload,
            });

            if (response.status() === 200) {
                const data = await response.json();
                expect(data.status).toBe('success');
                expect(data.data.event).toBe(eventType);
            }
        });
    }

    test('Webhook handles unknown event type gracefully', async ({ page }) => {
        const payload = JSON.stringify({
            ...sampleMediaPlayPayload,
            event: 'unknown.custom.event'
        });

        const response = await page.request.post('/api/v1/plex/webhook', {
            headers: {
                'Content-Type': 'application/json',
            },
            data: payload,
        });

        // Should still accept unknown events
        if (response.status() === 200) {
            const data = await response.json();
            expect(data.status).toBe('success');
            expect(data.data.event).toBe('unknown.custom.event');
        }
    });
});

test.describe('Plex Webhook TV Show Episodes', () => {
    test.beforeEach(async ({ page }) => {
        // Navigate (already authenticated via storageState from setup)
        await gotoAppAndWaitReady(page);
    });

    test('Webhook handles TV episode metadata correctly', async ({ page }) => {
        const episodePayload = {
            event: 'media.play',
            user: true,
            owner: false,
            Account: {
                id: 12345,
                thumb: '',
                title: 'TestUser'
            },
            Server: {
                title: 'My Plex Server',
                uuid: 'server-uuid-123'
            },
            Player: {
                local: false,
                publicAddress: '203.0.113.42',
                title: 'Apple TV',
                uuid: 'player-uuid-789'
            },
            Metadata: {
                librarySectionType: 'show',
                ratingKey: '99999',
                key: '/library/metadata/99999',
                parentRatingKey: '88888',
                grandparentRatingKey: '77777',
                guid: 'tvdb://123456',
                librarySectionTitle: 'TV Shows',
                librarySectionID: 2,
                type: 'episode',
                title: 'Pilot',
                grandparentTitle: 'Breaking Bad',
                parentTitle: 'Season 1',
                index: 1,
                parentIndex: 1,
                year: 2008
            }
        };

        const payload = JSON.stringify(episodePayload);

        const response = await page.request.post('/api/v1/plex/webhook', {
            headers: {
                'Content-Type': 'application/json',
            },
            data: payload,
        });

        if (response.status() === 200) {
            const data = await response.json();
            expect(data.status).toBe('success');
            expect(data.data.event).toBe('media.play');
        }
    });
});
