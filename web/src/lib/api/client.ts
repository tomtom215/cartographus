// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Base API Client
 *
 * Provides core functionality for all API modules:
 * - Fetch with authentication and error handling
 * - Client-side caching
 * - Cache status notifications
 * - Filter parameter building
 * - Request queuing to prevent connection overload (E2E-FIX)
 */

import type { LocationFilter } from '../types/core';
import { getAPICacheManager, APICacheManager } from '../api-cache';
import { SafeStorage } from '../utils/SafeStorage';

// Internal API response wrapper type
export interface APIResponse<T> {
    status: string;
    data: T;
    metadata: {
        timestamp: string;
        query_time_ms?: number;
        cached?: boolean;
    };
    error?: {
        code: string;
        message: string;
        details?: Record<string, unknown>;
    };
}

// Callback type for cache status notifications
export type CacheStatusCallback = (cached: boolean, queryTimeMs: number) => void;

/**
 * E2E-FIX: Request queue for deterministic API behavior
 *
 * Problem: Browsers limit concurrent connections per host (typically 6 for HTTP/1.1).
 * When the app fires 30+ requests after login, some fail with net::ERR_FAILED
 * because connections can't be established or reused properly. JS chunks loaded
 * via native import() bypass this queue and compete for the same connection pool.
 *
 * Solution: Queue API requests with conservative concurrency (maxConcurrent=2) and
 * moderate delay (minDelayMs=25) to leave room for chunk loading while preventing
 * connection pool exhaustion.
 *
 * Configuration rationale:
 * - maxConcurrent=2: Leave 4 connections for chunks, CSS, WebSocket, etc.
 * - minDelayMs=25: Prevent burst issues during rapid page navigation
 *
 * Combined with retry logic in fetch(), this ensures reliable API communication
 * even in constrained CI/container networking environments.
 */
class RequestQueue {
    private queue: Array<{
        execute: () => Promise<unknown>;
        resolve: (value: unknown) => void;
        reject: (error: unknown) => void;
    }> = [];
    private activeCount = 0;
    private lastRequestTime = 0;

    constructor(
        private maxConcurrent: number = 2,
        private minDelayMs: number = 25
    ) {}

    async enqueue<T>(fn: () => Promise<T>): Promise<T> {
        return new Promise<T>((resolve, reject) => {
            this.queue.push({
                execute: fn as () => Promise<unknown>,
                resolve: resolve as (value: unknown) => void,
                reject,
            });
            this.processQueue();
        });
    }

    private async processQueue(): Promise<void> {
        if (this.activeCount >= this.maxConcurrent || this.queue.length === 0) {
            return;
        }

        const item = this.queue.shift();
        if (!item) return;

        // Add minimum delay between requests to prevent connection issues
        const now = Date.now();
        const elapsed = now - this.lastRequestTime;
        if (elapsed < this.minDelayMs && this.lastRequestTime > 0) {
            await new Promise(r => setTimeout(r, this.minDelayMs - elapsed));
        }
        this.lastRequestTime = Date.now();

        this.activeCount++;
        try {
            const result = await item.execute();
            item.resolve(result);
        } catch (error) {
            item.reject(error);
        } finally {
            this.activeCount--;
            this.processQueue();
        }
    }
}

// Singleton request queue shared across all API clients
// Configuration: 2 concurrent requests, 25ms minimum delay
// - Conservative settings to prevent net::ERR_FAILED during page load
// - JS chunks are loaded via native import() which bypasses this queue
// - 2 concurrent leaves 4 connections for chunks, CSS, WebSockets, etc.
// - 25ms delay prevents burst issues during rapid page navigation
// - Retry logic (below) handles any transient network failures
const globalRequestQueue = new RequestQueue(2, 25);

/**
 * Queue a fetch request through the global request queue.
 *
 * Use this for any direct fetch() calls outside of BaseAPIClient to ensure
 * all requests go through the same connection management. This prevents
 * net::ERR_FAILED errors from connection pool exhaustion.
 *
 * @param url - The URL to fetch
 * @param options - Standard fetch options
 * @returns Promise<Response> - The fetch response
 *
 * @example
 * // Instead of:
 * const response = await fetch('/api/v1/health');
 *
 * // Use:
 * const response = await queuedFetch('/api/v1/health');
 */
export async function queuedFetch(url: string, options?: RequestInit): Promise<Response> {
    return globalRequestQueue.enqueue(() => fetch(url, options));
}

/**
 * Base API client with authentication, caching, and error handling
 */
export class BaseAPIClient {
    protected token: string | null = null;
    protected cacheStatusCallback: CacheStatusCallback | null = null;
    protected cacheManager: APICacheManager;
    protected cachingEnabled: boolean = true;

    constructor(protected baseURL: string) {
        this.token = SafeStorage.getItem('auth_token');
        this.cacheManager = getAPICacheManager();
    }

    /**
     * Enable or disable client-side caching
     */
    setCachingEnabled(enabled: boolean): void {
        this.cachingEnabled = enabled;
    }

    /**
     * Get cache statistics
     */
    getCacheStats(): ReturnType<APICacheManager['getStats']> {
        return this.cacheManager.getStats();
    }

    /**
     * Invalidate cache entries matching a pattern
     */
    invalidateCache(pattern: string): void {
        this.cacheManager.invalidate(pattern);
    }

    /**
     * Clear entire client-side cache
     */
    clearCache(): void {
        this.cacheManager.clear();
    }

    /**
     * Set callback to receive cache status updates
     */
    setCacheStatusCallback(callback: CacheStatusCallback): void {
        this.cacheStatusCallback = callback;
    }

    /**
     * Set authentication token (used by AuthAPI after login)
     */
    setToken(token: string | null): void {
        this.token = token;
    }

    /**
     * Get current authentication token
     */
    getToken(): string | null {
        return this.token;
    }

    /**
     * Notify of cache status from response metadata
     */
    protected notifyCacheStatus(metadata: { cached?: boolean; query_time_ms?: number }): void {
        if (this.cacheStatusCallback) {
            this.cacheStatusCallback(
                metadata.cached ?? false,
                metadata.query_time_ms ?? 0
            );
        }
    }

    /**
     * Build URL search params from a LocationFilter
     */
    protected buildFilterParams(filter: LocationFilter): URLSearchParams {
        const params = new URLSearchParams();

        // Simple scalar filters
        if (filter.days !== undefined) {
            params.append('days', filter.days.toString());
        }
        if (filter.start_date) {
            params.append('start_date', filter.start_date);
        }
        if (filter.end_date) {
            params.append('end_date', filter.end_date);
        }

        // String array filters
        const stringArrayFilters: [string, string[] | undefined][] = [
            ['users', filter.users],
            ['media_types', filter.media_types],
            ['platforms', filter.platforms],
            ['players', filter.players],
            ['transcode_decisions', filter.transcode_decisions],
            ['video_resolutions', filter.video_resolutions],
            ['video_codecs', filter.video_codecs],
            ['audio_codecs', filter.audio_codecs],
            ['libraries', filter.libraries],
            ['content_ratings', filter.content_ratings],
            ['location_types', filter.location_types],
        ];

        for (const [key, values] of stringArrayFilters) {
            if (values && values.length > 0) {
                params.append(key, values.join(','));
            }
        }

        // Number array filter (years)
        if (filter.years && filter.years.length > 0) {
            params.append('years', filter.years.join(','));
        }

        return params;
    }

    /**
     * Generic helper for analytics endpoints that take only a filter parameter
     */
    protected async fetchWithFilter<T>(endpoint: string, filter: LocationFilter = {}): Promise<T> {
        const params = this.buildFilterParams(filter);
        const queryString = params.toString();
        const url = queryString ? `${endpoint}?${queryString}` : endpoint;
        const response = await this.fetch<T>(url);
        return response.data;
    }

    /**
     * Generic helper for simple GET endpoints with no parameters
     */
    protected async fetchSimple<T>(endpoint: string): Promise<T> {
        const response = await this.fetch<T>(endpoint);
        return response.data;
    }

    /**
     * Core fetch method with authentication, caching, and error handling
     */
    protected async fetch<T>(endpoint: string, options: RequestInit = {}): Promise<APIResponse<T>> {
        const url = `${this.baseURL}${endpoint}`;
        const method = options.method?.toUpperCase() || 'GET';

        // Check client-side cache for GET requests (before queueing)
        if (this.cachingEnabled && method === 'GET' && this.cacheManager.isCacheable(url)) {
            const cachedData = this.cacheManager.get<APIResponse<T>>(url);
            if (cachedData) {
                this.notifyCacheStatus({ cached: true, query_time_ms: 0 });
                return cachedData;
            }
        }

        const headers: Record<string, string> = {
            'Content-Type': 'application/json',
        };

        if (options.headers) {
            Object.entries(options.headers).forEach(([key, value]) => {
                if (typeof value === 'string') {
                    headers[key] = value;
                }
            });
        }

        if (this.token && !endpoint.includes('/auth/login')) {
            headers['Authorization'] = `Bearer ${this.token}`;
        }

        // E2E-FIX: Queue the fetch with retry logic for transient failures
        const startTime = Date.now();
        const maxRetries = 3;
        let lastError: Error | null = null;

        for (let attempt = 0; attempt < maxRetries; attempt++) {
            try {
                const response = await globalRequestQueue.enqueue(() =>
                    fetch(url, {
                        ...options,
                        headers,
                        credentials: 'include',
                    })
                );

                // Only auto-logout for 401s on authenticated endpoints, not login attempts
                if (response.status === 401 && !endpoint.includes('/auth/login')) {
                    this.token = null;
                    // FIX: Clear ALL auth storage keys (was missing auth_user_id and auth_role)
                    SafeStorage.removeItem('auth_token');
                    SafeStorage.removeItem('auth_username');
                    SafeStorage.removeItem('auth_user_id');
                    SafeStorage.removeItem('auth_role');
                    SafeStorage.removeItem('auth_expires_at');
                    window.location.reload();
                    throw new Error('Session expired. Please login again.');
                }

                if (!response.ok) {
                    try {
                        const data = await response.json();
                        throw new Error(data.error?.message || `API request failed: ${response.statusText}`);
                    } catch (parseError) {
                        if (parseError instanceof Error && !parseError.message.includes('API request failed')) {
                            throw parseError;
                        }
                        throw new Error(`API request failed: ${response.statusText}`);
                    }
                }

                const data = await response.json();
                const requestTime = Date.now() - startTime;

                if (data.status === 'error') {
                    throw new Error(data.error?.message || 'Unknown API error');
                }

                // Notify cache status if metadata is present
                if (data.metadata) {
                    this.notifyCacheStatus(data.metadata);
                } else {
                    this.notifyCacheStatus({ cached: false, query_time_ms: requestTime });
                }

                // Store successful GET responses in client-side cache
                if (this.cachingEnabled && method === 'GET' && this.cacheManager.isCacheable(url)) {
                    this.cacheManager.set(url, data);
                }

                return data;
            } catch (error) {
                lastError = error instanceof Error ? error : new Error(String(error));

                // Only retry on network errors (Failed to fetch), not on HTTP errors
                const isNetworkError = lastError.message.includes('Failed to fetch') ||
                                       lastError.message.includes('NetworkError') ||
                                       lastError.message.includes('net::ERR_');

                if (!isNetworkError || attempt >= maxRetries - 1) {
                    throw lastError;
                }

                // Exponential backoff: 50ms, 100ms, 200ms
                const delay = 50 * Math.pow(2, attempt);
                await new Promise(r => setTimeout(r, delay));
            }
        }

        // Should never reach here, but TypeScript needs it
        throw lastError || new Error('Request failed after retries');
    }
}
