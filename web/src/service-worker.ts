// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Service Worker for Cartographus PWA
 * Provides offline functionality and faster repeat visits
 *
 * Enhanced Service Worker Caching Strategy
 * - Stale-while-revalidate for analytics API calls
 * - Cache expiration with configurable TTL
 * - Tiered caching (static, API, map tiles, images)
 * - Cache size management
 * - Automatic cache versioning and cleanup
 *
 * Reference: UI/UX Audit
 * @see /docs/working/UI_UX_AUDIT.md
 */

// Service worker type declarations
declare const self: ServiceWorkerGlobalScope;

const CACHE_VERSION = 'cartographus-v2';
const STATIC_CACHE = `${CACHE_VERSION}-static`;
const API_CACHE = `${CACHE_VERSION}-api`;
const IMAGE_CACHE = `${CACHE_VERSION}-images`;
const TILE_CACHE = `${CACHE_VERSION}-tiles`;

// Cache configuration
const CACHE_CONFIG = {
    // Maximum entries per cache
    maxEntries: {
        api: 50,
        images: 100,
        tiles: 200
    },
    // TTL in milliseconds
    ttl: {
        api: 5 * 60 * 1000,       // 5 minutes for API responses
        apiStatic: 30 * 60 * 1000, // 30 minutes for static API data (users, media types)
        images: 24 * 60 * 60 * 1000, // 24 hours for images
        tiles: 7 * 24 * 60 * 60 * 1000  // 7 days for map tiles
    }
};

// Static assets to cache on install
const STATIC_ASSETS = [
    '/',
    '/index.html',
    '/index.js',
    '/styles.css',
    '/manifest.json',
    '/icon.svg',
    '/icon-192.png',
    '/icon-512.png',
];

// API endpoints that can be cached with different TTLs
const CACHEABLE_API_PATTERNS = {
    // Static data - longer TTL
    static: [
        /\/api\/v1\/users$/,
        /\/api\/v1\/media-types$/,
        /\/api\/v1\/health$/,
    ],
    // Dynamic data - shorter TTL, stale-while-revalidate
    dynamic: [
        /\/api\/v1\/stats$/,
        /\/api\/v1\/locations/,
        /\/api\/v1\/analytics/,
        /\/api\/v1\/tautulli\//,
    ]
};

// Map tile providers
const TILE_PROVIDERS = [
    'basemaps.cartocdn.com',
    'a.basemaps.cartocdn.com',
    'b.basemaps.cartocdn.com',
    'c.basemaps.cartocdn.com',
    'tile.openstreetmap.org',
];

/**
 * Install event - cache static assets
 */
self.addEventListener('install', (event: ExtendableEvent) => {
    console.log('[Service Worker] Installing...');

    event.waitUntil(
        caches.open(STATIC_CACHE)
            .then((cache) => {
                console.log('[Service Worker] Caching static assets');
                return cache.addAll(STATIC_ASSETS.map(url => new Request(url, { cache: 'reload' })));
            })
            .catch((error) => {
                console.error('[Service Worker] Failed to cache static assets:', error);
                // Don't fail installation if caching fails
                return Promise.resolve();
            })
            .then(() => {
                console.log('[Service Worker] Installed successfully');
                // Skip waiting to activate immediately
                return self.skipWaiting();
            })
    );
});

/**
 * Activate event - clean up old caches
 */
self.addEventListener('activate', (event: ExtendableEvent) => {
    console.log('[Service Worker] Activating...');

    event.waitUntil(
        caches.keys()
            .then((cacheNames) => {
                // Delete old cache versions
                return Promise.all(
                    cacheNames
                        .filter((cacheName) => {
                            return cacheName.startsWith('cartographus-') &&
                                   cacheName !== STATIC_CACHE &&
                                   cacheName !== API_CACHE &&
                                   cacheName !== IMAGE_CACHE &&
                                   cacheName !== TILE_CACHE;
                        })
                        .map((cacheName) => {
                            console.log('[Service Worker] Deleting old cache:', cacheName);
                            return caches.delete(cacheName);
                        })
                );
            })
            .then(() => {
                console.log('[Service Worker] Activated successfully');
                // Take control of all clients immediately
                return self.clients.claim();
            })
    );
});

/**
 * Fetch event - implement caching strategies
 */
self.addEventListener('fetch', (event: FetchEvent) => {
    const { request } = event;
    const url = new URL(request.url);

    // Ignore non-GET requests
    if (request.method !== 'GET') {
        return;
    }

    // Ignore WebSocket connections
    if (url.protocol === 'ws:' || url.protocol === 'wss:') {
        return;
    }

    // Ignore Chrome extension requests
    if (url.protocol === 'chrome-extension:') {
        return;
    }

    // Strategy 1: Cache-first for static assets
    if (isStaticAsset(url)) {
        event.respondWith(cacheFirst(request, STATIC_CACHE));
        return;
    }

    // Strategy 2: Cache-first for map tiles with long TTL
    if (isMapTile(url)) {
        event.respondWith(cacheFirstWithTTL(request, TILE_CACHE, CACHE_CONFIG.ttl.tiles));
        return;
    }

    // Strategy 3: Cache-first for images with TTL
    if (isImageRequest(url)) {
        event.respondWith(cacheFirstWithTTL(request, IMAGE_CACHE, CACHE_CONFIG.ttl.images));
        return;
    }

    // Strategy 4: Stale-while-revalidate for static API data
    if (isStaticApiRequest(url.href)) {
        event.respondWith(staleWhileRevalidate(request, API_CACHE, CACHE_CONFIG.ttl.apiStatic));
        return;
    }

    // Strategy 5: Stale-while-revalidate for dynamic API data
    if (isDynamicApiRequest(url.href)) {
        event.respondWith(staleWhileRevalidate(request, API_CACHE, CACHE_CONFIG.ttl.api));
        return;
    }

    // Strategy 6: Network-first for other API calls
    if (isApiRequest(url)) {
        event.respondWith(networkFirst(request, API_CACHE));
        return;
    }

    // Strategy 7: Network-first for external resources
    if (isExternalResource(url)) {
        event.respondWith(networkFirst(request, IMAGE_CACHE));
        return;
    }

    // Default: Network-first
    event.respondWith(networkFirst(request, STATIC_CACHE));
});

/**
 * Cache-first strategy: Check cache first, fallback to network
 */
async function cacheFirst(request: Request, cacheName: string): Promise<Response> {
    try {
        const cache = await caches.open(cacheName);
        const cachedResponse = await cache.match(request);

        if (cachedResponse) {
            console.log('[Service Worker] Cache hit:', request.url);
            return cachedResponse;
        }

        console.log('[Service Worker] Cache miss, fetching:', request.url);
        const networkResponse = await fetch(request);

        // Cache successful responses
        if (networkResponse.ok) {
            cache.put(request, networkResponse.clone());
        }

        return networkResponse;
    } catch (error) {
        console.error('[Service Worker] Cache-first error:', error);

        // Try to return cached version as ultimate fallback
        const cache = await caches.open(cacheName);
        const cachedResponse = await cache.match(request);
        if (cachedResponse) {
            return cachedResponse;
        }

        // Return offline page or error
        return new Response('Offline - Unable to fetch resource', {
            status: 503,
            statusText: 'Service Unavailable',
            headers: { 'Content-Type': 'text/plain' },
        });
    }
}

/**
 * Cache-first with TTL: Check cache, validate TTL, fallback to network
 */
async function cacheFirstWithTTL(request: Request, cacheName: string, ttl: number): Promise<Response> {
    try {
        const cache = await caches.open(cacheName);
        const cachedResponse = await cache.match(request);

        if (cachedResponse) {
            // Check if cache entry is still valid
            const cachedTime = cachedResponse.headers.get('sw-cache-time');
            if (cachedTime) {
                const age = Date.now() - parseInt(cachedTime, 10);
                if (age < ttl) {
                    return cachedResponse;
                }
            } else {
                // No cache time header, return cached response anyway
                return cachedResponse;
            }
        }

        // Fetch fresh copy
        const networkResponse = await fetch(request);

        if (networkResponse.ok) {
            // Add cache time header to track expiry
            const responseWithTime = new Response(networkResponse.body, {
                status: networkResponse.status,
                statusText: networkResponse.statusText,
                headers: addCacheTimeHeader(networkResponse.headers)
            });
            cache.put(request, responseWithTime.clone());
            return responseWithTime;
        }

        return networkResponse;
    } catch (error) {
        // Return cached response even if expired
        const cache = await caches.open(cacheName);
        const cachedResponse = await cache.match(request);
        if (cachedResponse) {
            return cachedResponse;
        }

        return new Response('Offline - Unable to fetch resource', {
            status: 503,
            statusText: 'Service Unavailable',
            headers: { 'Content-Type': 'text/plain' },
        });
    }
}

/**
 * Stale-while-revalidate: Return cached response immediately, update in background
 */
async function staleWhileRevalidate(request: Request, cacheName: string, ttl: number): Promise<Response> {
    const cache = await caches.open(cacheName);
    const cachedResponse = await cache.match(request);

    // Start network fetch in background
    const fetchPromise = fetch(request).then(async (networkResponse) => {
        if (networkResponse.ok) {
            // Add cache time header
            const responseWithTime = new Response(networkResponse.body, {
                status: networkResponse.status,
                statusText: networkResponse.statusText,
                headers: addCacheTimeHeader(networkResponse.headers)
            });
            await cache.put(request, responseWithTime);

            // Enforce cache size limits
            await enforceStorageLimit(cacheName, CACHE_CONFIG.maxEntries.api);
        }
        return networkResponse;
    }).catch(() => {
        // Network failed, ignore for background revalidation
        return null;
    });

    // Return cached response immediately if available and not too old
    if (cachedResponse) {
        const cachedTime = cachedResponse.headers.get('sw-cache-time');
        if (cachedTime) {
            const age = Date.now() - parseInt(cachedTime, 10);
            // If cache is very stale, wait for network
            if (age > ttl * 2) {
                const networkResponse = await fetchPromise;
                if (networkResponse) {
                    return networkResponse;
                }
            }
        }
        return cachedResponse;
    }

    // No cache, wait for network
    const networkResponse = await fetchPromise;
    if (networkResponse) {
        return networkResponse;
    }

    return new Response('Offline - Unable to fetch resource', {
        status: 503,
        statusText: 'Service Unavailable',
        headers: { 'Content-Type': 'text/plain' },
    });
}

/**
 * Network-first strategy: Try network first, fallback to cache
 */
async function networkFirst(request: Request, cacheName: string): Promise<Response> {
    try {
        const networkResponse = await fetch(request);

        // Cache successful responses for cacheable API endpoints
        if (networkResponse.ok && isCacheableApiRequest(request.url)) {
            const cache = await caches.open(cacheName);
            const responseWithTime = new Response(networkResponse.body, {
                status: networkResponse.status,
                statusText: networkResponse.statusText,
                headers: addCacheTimeHeader(networkResponse.headers)
            });
            cache.put(request, responseWithTime);
        }

        return networkResponse;
    } catch (error) {
        console.warn('[Service Worker] Network failed, trying cache:', request.url);

        // Fallback to cache
        const cache = await caches.open(cacheName);
        const cachedResponse = await cache.match(request);

        if (cachedResponse) {
            return cachedResponse;
        }

        // Return error response
        console.error('[Service Worker] Network-first error:', error);
        return new Response('Offline - Unable to fetch resource', {
            status: 503,
            statusText: 'Service Unavailable',
            headers: { 'Content-Type': 'text/plain' },
        });
    }
}

/**
 * Add cache time header to response headers
 */
function addCacheTimeHeader(originalHeaders: Headers): Headers {
    const headers = new Headers(originalHeaders);
    headers.set('sw-cache-time', Date.now().toString());
    return headers;
}

/**
 * Enforce cache storage limits by removing oldest entries
 */
async function enforceStorageLimit(cacheName: string, maxEntries: number): Promise<void> {
    const cache = await caches.open(cacheName);
    const keys = await cache.keys();

    if (keys.length > maxEntries) {
        // Remove oldest entries (first in the list)
        const keysToDelete = keys.slice(0, keys.length - maxEntries);
        await Promise.all(keysToDelete.map(key => cache.delete(key)));
    }
}

/**
 * Check if request is for a static asset
 */
function isStaticAsset(url: URL): boolean {
    const pathname = url.pathname;
    return (
        pathname === '/' ||
        pathname === '/index.html' ||
        pathname.endsWith('.js') ||
        pathname.endsWith('.css') ||
        pathname.endsWith('.json') ||
        pathname.endsWith('.svg') ||
        pathname.endsWith('.png') ||
        pathname.endsWith('.jpg') ||
        pathname.endsWith('.ico')
    );
}

/**
 * Check if request is for an image
 */
function isImageRequest(url: URL): boolean {
    const pathname = url.pathname;
    return (
        pathname.endsWith('.png') ||
        pathname.endsWith('.jpg') ||
        pathname.endsWith('.jpeg') ||
        pathname.endsWith('.gif') ||
        pathname.endsWith('.webp') ||
        pathname.endsWith('.svg')
    );
}

/**
 * Check if request is for an API endpoint
 */
function isApiRequest(url: URL): boolean {
    return url.pathname.startsWith('/api/');
}

/**
 * Check if API request matches static (longer TTL) patterns
 */
function isStaticApiRequest(urlString: string): boolean {
    return CACHEABLE_API_PATTERNS.static.some(pattern => pattern.test(urlString));
}

/**
 * Check if API request matches dynamic (shorter TTL) patterns
 */
function isDynamicApiRequest(urlString: string): boolean {
    return CACHEABLE_API_PATTERNS.dynamic.some(pattern => pattern.test(urlString));
}

/**
 * Check if API request can be cached (any pattern)
 */
function isCacheableApiRequest(urlString: string): boolean {
    return isStaticApiRequest(urlString) || isDynamicApiRequest(urlString);
}

/**
 * Check if request is for a map tile
 */
function isMapTile(url: URL): boolean {
    const hostname = url.hostname.toLowerCase();
    // Check against known tile providers
    if (TILE_PROVIDERS.some(provider => hostname.endsWith(provider))) {
        return true;
    }
    // Check URL path for common tile patterns (z/x/y)
    return /\/\d+\/\d+\/\d+\.(?:png|pbf|jpg|mvt)/.test(url.pathname);
}

/**
 * Check if request is for an external resource (Mapbox, CARTO tiles)
 * Uses exact hostname matching or subdomain validation to prevent URL validation bypass
 */
function isExternalResource(url: URL): boolean {
    const hostname = url.hostname.toLowerCase();
    return (
        hostname === 'mapbox.com' || hostname.endsWith('.mapbox.com') ||
        hostname === 'cartocdn.com' || hostname.endsWith('.cartocdn.com') ||
        hostname === 'openstreetmap.org' || hostname.endsWith('.openstreetmap.org')
    );
}

/**
 * Message handler for communication with main thread
 * Validates that messages come from controlled clients for security
 */
self.addEventListener('message', (event: ExtendableMessageEvent) => {
    // Security: Validate message origin matches our origin
    // This prevents cross-origin messages from being processed
    const swOrigin = self.location.origin;
    if (event.origin && event.origin !== swOrigin) {
        console.warn('[Service Worker] Ignoring message from untrusted origin:', event.origin);
        return;
    }

    // Additional security: Validate message source is a controlled client
    // Service workers should only accept messages from their own clients
    if (!event.source || !(event.source instanceof Client)) {
        console.warn('[Service Worker] Ignoring message from invalid source');
        return;
    }

    if (event.data && event.data.type === 'SKIP_WAITING') {
        self.skipWaiting();
    }

    if (event.data && event.data.type === 'CLEAR_CACHE') {
        event.waitUntil(
            caches.keys().then((cacheNames) => {
                return Promise.all(
                    cacheNames.map((cacheName) => caches.delete(cacheName))
                );
            }).then(() => {
                console.log('[Service Worker] All caches cleared');
            })
        );
    }
});

// Export for testing
export {};
