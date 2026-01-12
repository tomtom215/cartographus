// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * SafeSessionStorage - Robust sessionStorage wrapper with fallbacks
 *
 * Similar to SafeStorage but uses sessionStorage instead of localStorage.
 * Ideal for temporary auth state that should not persist across browser sessions.
 *
 * Handles edge cases:
 * - Private browsing mode (sessionStorage disabled)
 * - Quota exceeded errors
 * - JSON parse errors
 * - Browser compatibility issues
 *
 * Falls back to in-memory storage when sessionStorage is unavailable.
 */

import { createLogger } from '../logger';

const logger = createLogger('SafeSessionStorage');

class SafeSessionStorageImpl {
    private memoryFallback: Map<string, string> = new Map();
    private isSessionStorageAvailable: boolean | null = null;

    /**
     * Check if sessionStorage is available and functional
     */
    private checkSessionStorage(): boolean {
        if (this.isSessionStorageAvailable !== null) {
            return this.isSessionStorageAvailable;
        }

        try {
            const testKey = '__session_storage_test__';
            window.sessionStorage.setItem(testKey, 'test');
            window.sessionStorage.removeItem(testKey);
            this.isSessionStorageAvailable = true;
            return true;
        } catch {
            logger.warn('sessionStorage not available, using memory fallback');
            this.isSessionStorageAvailable = false;
            return false;
        }
    }

    /**
     * Get an item from storage
     */
    getItem(key: string): string | null {
        try {
            if (this.checkSessionStorage()) {
                return window.sessionStorage.getItem(key);
            }
            return this.memoryFallback.get(key) ?? null;
        } catch (error) {
            logger.warn('Error reading from session storage:', error);
            return this.memoryFallback.get(key) ?? null;
        }
    }

    /**
     * Set an item in storage
     */
    setItem(key: string, value: string): boolean {
        try {
            if (this.checkSessionStorage()) {
                window.sessionStorage.setItem(key, value);
            }
            // Always update memory fallback as backup
            this.memoryFallback.set(key, value);
            return true;
        } catch (error) {
            // Handle QuotaExceededError
            if (error instanceof DOMException && error.name === 'QuotaExceededError') {
                logger.warn('Quota exceeded, using memory fallback');
            } else {
                logger.warn('Error writing to session storage:', error);
            }
            // Fall back to memory
            this.memoryFallback.set(key, value);
            return false;
        }
    }

    /**
     * Remove an item from storage
     */
    removeItem(key: string): void {
        try {
            if (this.checkSessionStorage()) {
                window.sessionStorage.removeItem(key);
            }
            this.memoryFallback.delete(key);
        } catch (error) {
            logger.warn('Error removing from session storage:', error);
            this.memoryFallback.delete(key);
        }
    }

    /**
     * Get a JSON-parsed value with type safety
     */
    getJSON<T>(key: string, defaultValue: T): T {
        try {
            const value = this.getItem(key);
            if (value === null) {
                return defaultValue;
            }
            return JSON.parse(value) as T;
        } catch (error) {
            logger.warn('Error parsing JSON from session storage:', error);
            return defaultValue;
        }
    }

    /**
     * Set a value as JSON
     */
    setJSON<T>(key: string, value: T): boolean {
        try {
            return this.setItem(key, JSON.stringify(value));
        } catch (error) {
            logger.warn('Error stringifying value for session storage:', error);
            return false;
        }
    }

    /**
     * Check if sessionStorage is being used (vs memory fallback)
     */
    isUsingSessionStorage(): boolean {
        return this.checkSessionStorage();
    }

    /**
     * Clear all data (both sessionStorage and memory)
     */
    clear(): void {
        try {
            if (this.checkSessionStorage()) {
                window.sessionStorage.clear();
            }
        } catch (error) {
            logger.warn('Error clearing session storage:', error);
        }
        this.memoryFallback.clear();
    }
}

// Export singleton instance
export const SafeSessionStorage = new SafeSessionStorageImpl();
