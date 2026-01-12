// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * APICacheManager - Client-side caching for API responses
 *
 * Features:
 * - In-memory caching with configurable TTL per endpoint type
 * - Automatic cache invalidation on expiry
 * - Cache size limits to prevent memory issues
 * - Support for cache bypass and manual invalidation
 * - Statistics tracking for cache hit/miss rates
 *
 * Reference: UI/UX Audit Task
 * @see /docs/working/UI_UX_AUDIT.md
 */

interface CacheEntry<T> {
  data: T;
  timestamp: number;
  ttl: number;
}

interface CacheStats {
  hits: number;
  misses: number;
  size: number;
  lastCleanup: number;
}

/** TTL configuration by endpoint pattern (in milliseconds) */
const DEFAULT_TTL_CONFIG: Record<string, number> = {
  // Analytics endpoints - cache for 5 minutes
  '/api/v1/analytics/': 5 * 60 * 1000,
  // Tautulli proxy endpoints - cache for 2 minutes
  '/api/v1/tautulli/': 2 * 60 * 1000,
  // Spatial endpoints - cache for 3 minutes
  '/api/v1/spatial/': 3 * 60 * 1000,
  // Stats endpoint - cache for 1 minute
  '/api/v1/stats': 60 * 1000,
  // Playbacks - cache for 1 minute
  '/api/v1/playbacks': 60 * 1000,
  // Health - short cache (30 seconds)
  '/api/v1/health': 30 * 1000,
  // Users list - cache for 5 minutes
  '/api/v1/users': 5 * 60 * 1000,
  // Server info - cache for 10 minutes
  '/api/v1/server': 10 * 60 * 1000,
  // Default TTL for other endpoints
  'default': 60 * 1000,
};

/** Endpoints that should never be cached */
const NO_CACHE_PATTERNS = [
  '/api/v1/auth/',
  '/api/v1/backup/',
  '/api/v1/export/',
  '/api/v1/stream/',
];

/** Maximum number of cache entries */
const MAX_CACHE_ENTRIES = 200;

/** Cleanup interval (5 minutes) */
const CLEANUP_INTERVAL = 5 * 60 * 1000;

export class APICacheManager {
  private cache: Map<string, CacheEntry<unknown>> = new Map();
  private stats: CacheStats = {
    hits: 0,
    misses: 0,
    size: 0,
    lastCleanup: Date.now(),
  };
  private ttlConfig: Record<string, number>;
  private cleanupTimer: number | null = null;

  constructor(ttlConfig?: Record<string, number>) {
    this.ttlConfig = { ...DEFAULT_TTL_CONFIG, ...ttlConfig };
    this.startCleanupTimer();
  }

  /**
   * Generate a cache key from URL and optional params
   */
  private getCacheKey(url: string, params?: Record<string, unknown>): string {
    let key = url;
    if (params && Object.keys(params).length > 0) {
      const sortedParams = Object.keys(params)
        .sort()
        .map(k => `${k}=${JSON.stringify(params[k])}`)
        .join('&');
      key = `${url}?${sortedParams}`;
    }
    return key;
  }

  /**
   * Get TTL for a given endpoint
   */
  private getTTL(url: string): number {
    for (const [pattern, ttl] of Object.entries(this.ttlConfig)) {
      if (pattern !== 'default' && url.includes(pattern)) {
        return ttl;
      }
    }
    return this.ttlConfig['default'] || 60 * 1000;
  }

  /**
   * Check if an endpoint should be cached
   */
  private shouldCache(url: string, method: string): boolean {
    // Only cache GET requests
    if (method.toUpperCase() !== 'GET') {
      return false;
    }

    // Check against no-cache patterns
    for (const pattern of NO_CACHE_PATTERNS) {
      if (url.includes(pattern)) {
        return false;
      }
    }

    return true;
  }

  /**
   * Get a cached response if available and not expired
   */
  get<T>(url: string, params?: Record<string, unknown>): T | null {
    const key = this.getCacheKey(url, params);
    const entry = this.cache.get(key) as CacheEntry<T> | undefined;

    if (!entry) {
      this.stats.misses++;
      return null;
    }

    // Check if expired
    if (Date.now() - entry.timestamp > entry.ttl) {
      this.cache.delete(key);
      this.stats.misses++;
      this.stats.size = this.cache.size;
      return null;
    }

    this.stats.hits++;
    return entry.data;
  }

  /**
   * Store a response in the cache
   */
  set<T>(url: string, data: T, params?: Record<string, unknown>): void {
    if (!this.shouldCache(url, 'GET')) {
      return;
    }

    // Enforce size limit
    if (this.cache.size >= MAX_CACHE_ENTRIES) {
      this.evictOldest();
    }

    const key = this.getCacheKey(url, params);
    const ttl = this.getTTL(url);

    this.cache.set(key, {
      data,
      timestamp: Date.now(),
      ttl,
    });

    this.stats.size = this.cache.size;
  }

  /**
   * Check if a URL should be cached (for external use)
   */
  isCacheable(url: string, method: string = 'GET'): boolean {
    return this.shouldCache(url, method);
  }

  /**
   * Invalidate cache entries matching a pattern
   */
  invalidate(pattern: string): void {
    const keysToDelete: string[] = [];

    this.cache.forEach((_, key) => {
      if (key.includes(pattern)) {
        keysToDelete.push(key);
      }
    });

    keysToDelete.forEach(key => this.cache.delete(key));
    this.stats.size = this.cache.size;
  }

  /**
   * Clear the entire cache
   */
  clear(): void {
    this.cache.clear();
    this.stats.size = 0;
  }

  /**
   * Get cache statistics
   */
  getStats(): CacheStats & { hitRate: number } {
    const total = this.stats.hits + this.stats.misses;
    const hitRate = total > 0 ? (this.stats.hits / total) * 100 : 0;

    return {
      ...this.stats,
      hitRate,
    };
  }

  /**
   * Evict the oldest cache entry
   */
  private evictOldest(): void {
    let oldestKey: string | null = null;
    let oldestTime = Infinity;

    this.cache.forEach((entry, key) => {
      if (entry.timestamp < oldestTime) {
        oldestTime = entry.timestamp;
        oldestKey = key;
      }
    });

    if (oldestKey) {
      this.cache.delete(oldestKey);
    }
  }

  /**
   * Start the periodic cleanup timer
   */
  private startCleanupTimer(): void {
    if (typeof window !== 'undefined') {
      this.cleanupTimer = window.setInterval(() => {
        this.cleanup();
      }, CLEANUP_INTERVAL);
    }
  }

  /**
   * Clean up expired entries
   */
  private cleanup(): void {
    const now = Date.now();
    const keysToDelete: string[] = [];

    this.cache.forEach((entry, key) => {
      if (now - entry.timestamp > entry.ttl) {
        keysToDelete.push(key);
      }
    });

    keysToDelete.forEach(key => this.cache.delete(key));
    this.stats.size = this.cache.size;
    this.stats.lastCleanup = now;
  }

  /**
   * Destroy the cache manager
   */
  destroy(): void {
    if (this.cleanupTimer !== null) {
      clearInterval(this.cleanupTimer);
      this.cleanupTimer = null;
    }
    this.clear();
  }
}

// Singleton instance for global use
let globalCacheInstance: APICacheManager | null = null;

export function getAPICacheManager(): APICacheManager {
  if (!globalCacheInstance) {
    globalCacheInstance = new APICacheManager();
  }
  return globalCacheInstance;
}

export default APICacheManager;
