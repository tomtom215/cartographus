// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * SafeStorage - Robust localStorage wrapper with fallbacks
 *
 * Handles edge cases:
 * - Private browsing mode (localStorage disabled)
 * - Quota exceeded errors
 * - JSON parse errors
 * - Browser compatibility issues
 *
 * Falls back to in-memory storage when localStorage is unavailable.
 */

import { createLogger } from '../logger';

const logger = createLogger('SafeStorage');

class SafeStorageImpl {
  private memoryFallback: Map<string, string> = new Map();
  private isLocalStorageAvailable: boolean | null = null;

  /**
   * Check if localStorage is available and functional
   */
  private checkLocalStorage(): boolean {
    if (this.isLocalStorageAvailable !== null) {
      return this.isLocalStorageAvailable;
    }

    try {
      const testKey = '__storage_test__';
      window.localStorage.setItem(testKey, 'test');
      window.localStorage.removeItem(testKey);
      this.isLocalStorageAvailable = true;
      return true;
    } catch {
      logger.warn('localStorage not available, using memory fallback');
      this.isLocalStorageAvailable = false;
      return false;
    }
  }

  /**
   * Get an item from storage
   */
  getItem(key: string): string | null {
    try {
      if (this.checkLocalStorage()) {
        return window.localStorage.getItem(key);
      }
      return this.memoryFallback.get(key) ?? null;
    } catch (error) {
      logger.warn('Error reading from storage:', error);
      return this.memoryFallback.get(key) ?? null;
    }
  }

  /**
   * Set an item in storage
   */
  setItem(key: string, value: string): boolean {
    try {
      if (this.checkLocalStorage()) {
        window.localStorage.setItem(key, value);
      }
      // Always update memory fallback as backup
      this.memoryFallback.set(key, value);
      return true;
    } catch (error) {
      // Handle QuotaExceededError
      if (error instanceof DOMException && error.name === 'QuotaExceededError') {
        logger.warn('Quota exceeded, using memory fallback');
      } else {
        logger.warn('Error writing to storage:', error);
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
      if (this.checkLocalStorage()) {
        window.localStorage.removeItem(key);
      }
      this.memoryFallback.delete(key);
    } catch (error) {
      logger.warn('Error removing from storage:', error);
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
      logger.warn('Error parsing JSON from storage:', error);
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
      logger.warn('Error stringifying value for storage:', error);
      return false;
    }
  }

  /**
   * Check if localStorage is being used (vs memory fallback)
   */
  isUsingLocalStorage(): boolean {
    return this.checkLocalStorage();
  }

  /**
   * Clear all data (both localStorage and memory)
   */
  clear(): void {
    try {
      if (this.checkLocalStorage()) {
        window.localStorage.clear();
      }
    } catch (error) {
      logger.warn('Error clearing storage:', error);
    }
    this.memoryFallback.clear();
  }
}

// Export singleton instance
export const SafeStorage = new SafeStorageImpl();
