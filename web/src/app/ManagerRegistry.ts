// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * ManagerRegistry - Centralized manager lifecycle management
 *
 * Reduces complexity in index.ts by:
 * 1. Providing a single place to register/retrieve managers
 * 2. Handling cleanup of all managers on logout
 * 3. Grouping related manager initialization
 */

import { createLogger } from '../lib/logger';

const logger = createLogger('ManagerRegistry');

export interface Destroyable {
    destroy?(): void;
    disconnect?(): void;
    dismissAll?(): void;
}

type ManagerEntry = {
    instance: Destroyable;
    name: string;
};

/**
 * Registry for managing application managers lifecycle
 */
export class ManagerRegistry {
    private managers: Map<string, ManagerEntry> = new Map();

    /**
     * Register a manager instance
     */
    register<T extends Destroyable>(name: string, instance: T): T {
        this.managers.set(name, { instance, name });
        return instance;
    }

    /**
     * Get a registered manager by name
     */
    get<T extends Destroyable>(name: string): T | null {
        const entry = this.managers.get(name);
        return entry ? (entry.instance as T) : null;
    }

    /**
     * Check if a manager is registered
     */
    has(name: string): boolean {
        return this.managers.has(name);
    }

    /**
     * Destroy all registered managers
     * Calls destroy(), disconnect(), or dismissAll() as appropriate
     */
    destroyAll(): void {
        for (const [name, entry] of this.managers) {
            try {
                const manager = entry.instance;
                if (manager.destroy) {
                    manager.destroy();
                } else if (manager.disconnect) {
                    manager.disconnect();
                } else if (manager.dismissAll) {
                    manager.dismissAll();
                }
            } catch (error) {
                logger.warn(`Failed to destroy ${name}:`, error);
            }
        }
        this.managers.clear();
    }

    /**
     * Get count of registered managers
     */
    get size(): number {
        return this.managers.size;
    }

    /**
     * Get all registered manager names
     */
    getNames(): string[] {
        return Array.from(this.managers.keys());
    }
}

// Singleton instance for the application
let registryInstance: ManagerRegistry | null = null;

export function getManagerRegistry(): ManagerRegistry {
    if (!registryInstance) {
        registryInstance = new ManagerRegistry();
    }
    return registryInstance;
}

export function resetManagerRegistry(): void {
    if (registryInstance) {
        registryInstance.destroyAll();
    }
    registryInstance = null;
}
