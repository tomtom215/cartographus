// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
const CACHE_NAME = 'cartographus-v1';
const CACHE_MAX_SIZE = 50; // Maximum number of cached items
const CACHE_MAX_AGE = 24 * 60 * 60 * 1000; // 24 hours in milliseconds

// Only cache local resources during install, not external CDN resources
const urlsToCache = [
  '/',
  '/index.html',
  '/bundle.js',
  '/styles.css',
  '/manifest.json',
  '/icon.svg',
  '/icon-192.png',
  '/icon-512.png',
  '/robots.txt'
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => {
        // Use addAll with better error handling
        return cache.addAll(urlsToCache).catch((error) => {
          console.error('Cache installation failed for some resources:', error);
          // Try to cache individually to identify which resources failed
          return Promise.allSettled(
            urlsToCache.map(url =>
              cache.add(url).catch(err =>
                console.error(`Failed to cache ${url}:`, err)
              )
            )
          );
        });
      })
  );
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          if (cacheName !== CACHE_NAME) {
            console.log('Deleting old cache:', cacheName);
            return caches.delete(cacheName);
          }
        })
      );
    })
  );
  self.clients.claim();
});

self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);

  if (request.method !== 'GET') {
    return;
  }

  // Network-first strategy for API requests
  if (url.pathname.startsWith('/api/')) {
    event.respondWith(
      fetch(request)
        .then((response) => {
          if (response.ok) {
            const responseClone = response.clone();
            caches.open(CACHE_NAME).then((cache) => {
              cache.put(request, responseClone).catch((err) => {
                console.warn('Failed to cache API response:', err);
              });
            });
          }
          return response;
        })
        .catch((error) => {
          console.warn('Network request failed, trying cache:', error);
          return caches.match(request).then((cachedResponse) => {
            if (cachedResponse) {
              return cachedResponse;
            }
            // Return a custom offline response for API failures
            return new Response(
              JSON.stringify({ error: 'Network unavailable', offline: true }),
              {
                status: 503,
                statusText: 'Service Unavailable',
                headers: { 'Content-Type': 'application/json' }
              }
            );
          });
        })
    );
    return;
  }

  // Cache-first strategy for static resources
  event.respondWith(
    caches.match(request)
      .then((response) => {
        if (response) {
          return response;
        }

        return fetch(request).then((response) => {
          // Don't cache error responses or opaque responses
          if (!response || response.status !== 200 || response.type === 'error') {
            return response;
          }

          // Don't cache external resources (except Mapbox/Carto tiles with network-first)
          // Use strict origin/hostname comparison to prevent URL validation bypass
          const hostname = url.hostname.toLowerCase();
          const isSameOrigin = url.origin === self.location.origin;
          const isCartoTile = hostname === 'basemaps.cartocdn.com' || hostname.endsWith('.basemaps.cartocdn.com');
          const isMapboxTile = hostname === 'api.mapbox.com' || hostname.endsWith('.api.mapbox.com');
          if (!isSameOrigin && !isCartoTile && !isMapboxTile) {
            return response;
          }

          const responseToCache = response.clone();
          caches.open(CACHE_NAME).then((cache) => {
            cache.put(request, responseToCache).catch((err) => {
              console.warn('Failed to cache resource:', err);
            });
          });

          return response;
        }).catch((error) => {
          console.error('Fetch failed:', error);
          // For HTML pages, return cached index.html as fallback for SPA routing
          if (request.headers.get('accept').includes('text/html')) {
            return caches.match('/index.html');
          }
          throw error;
        });
      })
  );
});

// Periodic cache cleanup
// Validates message origin and source for security
self.addEventListener('message', (event) => {
  // Security: Validate message origin matches our origin
  // This prevents cross-origin messages from being processed
  const swOrigin = self.location.origin;
  if (event.origin && event.origin !== swOrigin) {
    console.warn('[Service Worker] Ignoring message from untrusted origin:', event.origin);
    return;
  }

  // Security: Validate message source is a controlled client
  if (!event.source) {
    console.warn('[Service Worker] Ignoring message from invalid source');
    return;
  }

  if (event.data && event.data.action === 'cleanupCache') {
    event.waitUntil(cleanupCache());
  }
});

async function cleanupCache() {
  const cache = await caches.open(CACHE_NAME);
  const keys = await cache.keys();

  if (keys.length > CACHE_MAX_SIZE) {
    // Remove oldest entries
    const toDelete = keys.length - CACHE_MAX_SIZE;
    for (let i = 0; i < toDelete; i++) {
      await cache.delete(keys[i]);
    }
    console.log(`Cleaned up ${toDelete} old cache entries`);
  }
}
