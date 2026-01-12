// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * RoleGuard - Role-based UI access control
 *
 * Provides methods for:
 * - Checking if current user can access resources
 * - Hiding/showing UI elements based on role
 * - Disabling elements for unauthorized users
 *
 * RBAC Phase 4: Frontend Role Integration
 * ADR-0015: Zero Trust Authentication & Authorization
 */

import { AuthContext } from './AuthContext';
import type { AuthContextInstance } from './AuthContext';
import type { UserRole } from '../types/auth';
import { createLogger } from '../logger';

const logger = createLogger('RoleGuard');

/**
 * Resource types for access control.
 */
export type ResourceType =
    | 'playbacks'
    | 'analytics'
    | 'users'
    | 'backup'
    | 'server'
    | 'sync'
    | 'detection'
    | 'audit'
    | 'settings'
    | 'wrapped';

/**
 * Action types for access control.
 */
export type ActionType = 'read' | 'write' | 'delete' | 'admin';

/**
 * Resource access matrix defining which roles can perform which actions.
 * Based on the backend Casbin policies in internal/authz/policy.csv.
 */
const RESOURCE_ACCESS_MATRIX: Record<ResourceType, Record<ActionType, UserRole>> = {
    playbacks: { read: 'viewer', write: 'editor', delete: 'admin', admin: 'admin' },
    analytics: { read: 'viewer', write: 'editor', delete: 'admin', admin: 'admin' },
    users: { read: 'viewer', write: 'admin', delete: 'admin', admin: 'admin' },
    backup: { read: 'admin', write: 'admin', delete: 'admin', admin: 'admin' },
    server: { read: 'viewer', write: 'admin', delete: 'admin', admin: 'admin' },
    sync: { read: 'viewer', write: 'admin', delete: 'admin', admin: 'admin' },
    detection: { read: 'viewer', write: 'admin', delete: 'admin', admin: 'admin' },
    audit: { read: 'admin', write: 'admin', delete: 'admin', admin: 'admin' },
    settings: { read: 'viewer', write: 'editor', delete: 'admin', admin: 'admin' },
    wrapped: { read: 'viewer', write: 'editor', delete: 'admin', admin: 'admin' },
};

/**
 * RoleGuard provides role-based access control for UI elements.
 *
 * Usage:
 * ```typescript
 * const guard = new RoleGuard();
 *
 * // Check access
 * if (guard.canAccess('backup', 'read')) {
 *     // Show backup UI
 * }
 *
 * // Hide elements based on role
 * guard.hideIfNotAdmin(backupButton);
 *
 * // Show elements for specific roles
 * guard.showIfRole(adminPanel, 'admin');
 * ```
 */
export class RoleGuard {
    private authContext: AuthContextInstance;

    constructor(authContext?: AuthContextInstance) {
        this.authContext = authContext || AuthContext.getInstance();
    }

    // ========================================================================
    // Access Checks
    // ========================================================================

    /**
     * Check if the current user can access a resource with a specific action.
     *
     * @param resource - The resource type to access
     * @param action - The action to perform
     * @returns true if access is allowed
     */
    canAccess(resource: ResourceType, action: ActionType): boolean {
        if (!this.authContext.isAuthenticated()) {
            return false;
        }

        const requiredRole = RESOURCE_ACCESS_MATRIX[resource]?.[action];
        if (!requiredRole) {
            logger.warn(`Unknown resource/action combination: ${resource}/${action}`);
            return false;
        }

        return this.authContext.hasRole(requiredRole);
    }

    /**
     * Check if the current user can view a specific user's data.
     * Users can view their own data; admins can view all users' data.
     *
     * @param targetUserId - The ID of the user whose data is being accessed
     * @returns true if access is allowed
     */
    canViewUser(targetUserId: string): boolean {
        return this.authContext.canAccessUser(targetUserId);
    }

    /**
     * Check if the current user is an admin.
     */
    isAdmin(): boolean {
        return this.authContext.isAdmin();
    }

    /**
     * Check if the current user is an editor or higher.
     */
    isEditor(): boolean {
        return this.authContext.isEditor();
    }

    /**
     * Check if the current user has at least the required role.
     */
    hasRole(role: UserRole): boolean {
        return this.authContext.hasRole(role);
    }

    // ========================================================================
    // UI Element Control
    // ========================================================================

    /**
     * Hide an element if the current user is not an admin.
     *
     * @param element - The HTML element to hide
     */
    hideIfNotAdmin(element: HTMLElement | null): void {
        if (!element) return;
        if (!this.isAdmin()) {
            element.style.display = 'none';
            element.setAttribute('aria-hidden', 'true');
        }
    }

    /**
     * Show an element only if the current user is an admin.
     * Sets display to '' (default) when admin, 'none' otherwise.
     *
     * @param element - The HTML element to show/hide
     */
    showIfAdmin(element: HTMLElement | null): void {
        if (!element) return;
        if (this.isAdmin()) {
            element.style.display = '';
            element.removeAttribute('aria-hidden');
        } else {
            element.style.display = 'none';
            element.setAttribute('aria-hidden', 'true');
        }
    }

    /**
     * Show an element only if the current user has at least the required role.
     *
     * @param element - The HTML element to show/hide
     * @param requiredRole - The minimum required role
     */
    showIfRole(element: HTMLElement | null, requiredRole: UserRole): void {
        if (!element) return;
        if (this.hasRole(requiredRole)) {
            element.style.display = '';
            element.removeAttribute('aria-hidden');
        } else {
            element.style.display = 'none';
            element.setAttribute('aria-hidden', 'true');
        }
    }

    /**
     * Disable an element if the current user is not an admin.
     * Preserves visibility but prevents interaction.
     *
     * @param element - The HTML element to disable
     */
    disableIfNotAdmin(element: HTMLElement | null): void {
        if (!element) return;
        if (!this.isAdmin()) {
            if (element instanceof HTMLButtonElement || element instanceof HTMLInputElement) {
                element.disabled = true;
            }
            element.setAttribute('aria-disabled', 'true');
            element.classList.add('disabled');
        }
    }

    /**
     * Disable an element if the current user doesn't have the required role.
     *
     * @param element - The HTML element to disable
     * @param requiredRole - The minimum required role
     */
    disableIfNotRole(element: HTMLElement | null, requiredRole: UserRole): void {
        if (!element) return;
        if (!this.hasRole(requiredRole)) {
            if (element instanceof HTMLButtonElement || element instanceof HTMLInputElement) {
                element.disabled = true;
            }
            element.setAttribute('aria-disabled', 'true');
            element.classList.add('disabled');
        }
    }

    /**
     * Enable an element if the current user has the required role.
     *
     * @param element - The HTML element to enable
     * @param requiredRole - The minimum required role
     */
    enableIfRole(element: HTMLElement | null, requiredRole: UserRole): void {
        if (!element) return;
        if (this.hasRole(requiredRole)) {
            if (element instanceof HTMLButtonElement || element instanceof HTMLInputElement) {
                element.disabled = false;
            }
            element.removeAttribute('aria-disabled');
            element.classList.remove('disabled');
        } else {
            if (element instanceof HTMLButtonElement || element instanceof HTMLInputElement) {
                element.disabled = true;
            }
            element.setAttribute('aria-disabled', 'true');
            element.classList.add('disabled');
        }
    }

    // ========================================================================
    // Bulk Operations
    // ========================================================================

    /**
     * Apply admin-only visibility to multiple elements.
     *
     * @param elements - Array of HTML elements or IDs
     */
    hideAllIfNotAdmin(elements: (HTMLElement | string | null)[]): void {
        elements.forEach(el => {
            const element = typeof el === 'string' ? document.getElementById(el) : el;
            this.hideIfNotAdmin(element);
        });
    }

    /**
     * Apply role-based visibility to multiple elements.
     *
     * @param elements - Array of HTML elements or IDs
     * @param requiredRole - The minimum required role
     */
    hideAllIfNotRole(elements: (HTMLElement | string | null)[], requiredRole: UserRole): void {
        elements.forEach(el => {
            const element = typeof el === 'string' ? document.getElementById(el) : el;
            this.showIfRole(element, requiredRole);
        });
    }

    /**
     * Apply role-based interactivity to multiple elements.
     *
     * @param elements - Array of HTML elements or IDs
     * @param requiredRole - The minimum required role
     */
    disableAllIfNotRole(elements: (HTMLElement | string | null)[], requiredRole: UserRole): void {
        elements.forEach(el => {
            const element = typeof el === 'string' ? document.getElementById(el) : el;
            this.disableIfNotRole(element, requiredRole);
        });
    }

    // ========================================================================
    // Access Control Messages
    // ========================================================================

    /**
     * Get a user-friendly message explaining why access was denied.
     *
     * @param resource - The resource that was denied
     * @param action - The action that was denied
     * @returns A user-friendly denial message
     */
    getDenialMessage(resource: ResourceType, action: ActionType): string {
        const requiredRole = RESOURCE_ACCESS_MATRIX[resource]?.[action];
        if (!requiredRole) {
            return 'Access denied: unknown resource';
        }

        const currentRole = this.authContext.getRole() || 'unauthenticated';
        return `Access denied: ${action} ${resource} requires ${requiredRole} role (you are ${currentRole})`;
    }
}

// Export singleton instance for convenience
let roleGuardInstance: RoleGuard | null = null;

/**
 * Get the singleton RoleGuard instance.
 */
export function getRoleGuard(): RoleGuard {
    if (!roleGuardInstance) {
        roleGuardInstance = new RoleGuard();
    }
    return roleGuardInstance;
}

/**
 * Reset the singleton RoleGuard instance (for testing).
 */
export function resetRoleGuard(): void {
    roleGuardInstance = null;
}
