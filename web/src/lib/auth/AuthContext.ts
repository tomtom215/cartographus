// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * AuthContext - Central authentication state management
 *
 * Manages:
 * - Authentication state (isAuthenticated, userId, username, role)
 * - State persistence (localStorage with memory fallback)
 * - State change notifications via listeners
 * - Role hierarchy validation
 *
 * RBAC Phase 4: Frontend Role Integration
 * ADR-0015: Zero Trust Authentication & Authorization
 */

import { SafeStorage } from '../utils/SafeStorage';
import { createLogger } from '../logger';
import type { UserRole } from '../types/auth';
import { isValidRole } from '../types/auth';

const logger = createLogger('AuthContext');

// Storage keys for auth data
const AUTH_KEYS = {
    TOKEN: 'auth_token',
    USERNAME: 'auth_username',
    USER_ID: 'auth_user_id',
    ROLE: 'auth_role',
    EXPIRES_AT: 'auth_expires_at',
} as const;

/**
 * AuthState represents the current authentication state.
 */
export interface AuthState {
    /** Whether the user is authenticated */
    isAuthenticated: boolean;
    /** Unique user identifier */
    userId: string | null;
    /** Display username */
    username: string | null;
    /** User's role (viewer, editor, admin) */
    role: UserRole | null;
    /** Token expiration timestamp (Unix ms) */
    expiresAt: number | null;
}

/**
 * Listener callback type for auth state changes.
 */
export type AuthStateListener = (state: AuthState) => void;

/**
 * AuthContext provides centralized authentication state management.
 *
 * Features:
 * - Singleton pattern for global access
 * - Automatic persistence to storage
 * - State change notifications
 * - Role-based permission checks
 *
 * Usage:
 * ```typescript
 * const authContext = AuthContext.getInstance();
 *
 * // Check if admin
 * if (authContext.isAdmin()) {
 *     // Show admin UI
 * }
 *
 * // Listen for auth changes
 * authContext.addListener((state) => {
 *     console.log('Auth state changed:', state);
 * });
 * ```
 */
class AuthContextImpl {
    private state: AuthState;
    private listeners: Set<AuthStateListener> = new Set();
    private static instance: AuthContextImpl | null = null;

    private constructor() {
        this.state = this.loadState();
        logger.debug('AuthContext initialized', { isAuthenticated: this.state.isAuthenticated, role: this.state.role });
    }

    /**
     * Get the singleton instance of AuthContext.
     */
    static getInstance(): AuthContextImpl {
        if (!AuthContextImpl.instance) {
            AuthContextImpl.instance = new AuthContextImpl();
        }
        return AuthContextImpl.instance;
    }

    /**
     * Reset the singleton instance (for testing).
     */
    static resetInstance(): void {
        AuthContextImpl.instance = null;
    }

    /**
     * Load authentication state from storage.
     */
    private loadState(): AuthState {
        const token = SafeStorage.getItem(AUTH_KEYS.TOKEN);
        const username = SafeStorage.getItem(AUTH_KEYS.USERNAME);
        const userId = SafeStorage.getItem(AUTH_KEYS.USER_ID);
        const roleStr = SafeStorage.getItem(AUTH_KEYS.ROLE);
        const expiresAtStr = SafeStorage.getItem(AUTH_KEYS.EXPIRES_AT);

        // Parse role with validation
        const role: UserRole | null = roleStr && isValidRole(roleStr) ? roleStr : null;

        // Parse expiration timestamp
        let expiresAt: number | null = null;
        if (expiresAtStr) {
            const parsed = new Date(expiresAtStr).getTime();
            if (!isNaN(parsed)) {
                expiresAt = parsed;
            }
        }

        // Check if token is valid (exists and not expired)
        const isAuthenticated = Boolean(token) && (expiresAt === null || expiresAt > Date.now());

        return {
            isAuthenticated,
            userId: isAuthenticated ? userId : null,
            username: isAuthenticated ? username : null,
            role: isAuthenticated ? role : null,
            expiresAt: isAuthenticated ? expiresAt : null,
        };
    }

    /**
     * Save authentication state to storage.
     */
    private saveState(): void {
        if (this.state.isAuthenticated) {
            if (this.state.userId) SafeStorage.setItem(AUTH_KEYS.USER_ID, this.state.userId);
            if (this.state.role) SafeStorage.setItem(AUTH_KEYS.ROLE, this.state.role);
        } else {
            SafeStorage.removeItem(AUTH_KEYS.USER_ID);
            SafeStorage.removeItem(AUTH_KEYS.ROLE);
        }
    }

    /**
     * Notify all listeners of state change.
     */
    private notifyListeners(): void {
        const stateCopy = { ...this.state };
        this.listeners.forEach(listener => {
            try {
                listener(stateCopy);
            } catch (error) {
                logger.error('Error in auth state listener:', error);
            }
        });
    }

    // ========================================================================
    // State Getters
    // ========================================================================

    /**
     * Get the current authentication state.
     */
    getState(): Readonly<AuthState> {
        return { ...this.state };
    }

    /**
     * Check if user is authenticated.
     */
    isAuthenticated(): boolean {
        // Re-check expiration on each call
        if (this.state.expiresAt && this.state.expiresAt <= Date.now()) {
            this.clearAuth();
            return false;
        }
        return this.state.isAuthenticated;
    }

    /**
     * Get the current user's ID.
     */
    getUserId(): string | null {
        return this.state.userId;
    }

    /**
     * Get the current user's username.
     */
    getUsername(): string | null {
        return this.state.username;
    }

    /**
     * Get the current user's role.
     */
    getRole(): UserRole | null {
        return this.state.role;
    }

    // ========================================================================
    // Role Checks
    // ========================================================================

    /**
     * Check if the current user is an admin.
     */
    isAdmin(): boolean {
        return this.state.role === 'admin';
    }

    /**
     * Check if the current user is an editor or higher.
     */
    isEditor(): boolean {
        return this.state.role === 'editor' || this.state.role === 'admin';
    }

    /**
     * Check if the current user is a viewer or higher (always true if authenticated).
     */
    isViewer(): boolean {
        return this.state.isAuthenticated && this.state.role !== null;
    }

    /**
     * Check if the current user has at least the required role.
     */
    hasRole(requiredRole: UserRole): boolean {
        if (!this.state.role) return false;

        const roleHierarchy: Record<UserRole, number> = {
            viewer: 0,
            editor: 1,
            admin: 2,
        };

        return roleHierarchy[this.state.role] >= roleHierarchy[requiredRole];
    }

    /**
     * Check if the current user can access another user's data.
     * Users can access their own data; admins can access all users' data.
     */
    canAccessUser(targetUserId: string): boolean {
        if (!this.isAuthenticated()) return false;
        if (this.isAdmin()) return true;
        return this.state.userId === targetUserId;
    }

    // ========================================================================
    // State Management
    // ========================================================================

    /**
     * Set authentication state after successful login.
     *
     * @param userId - User's unique identifier
     * @param username - User's display name
     * @param role - User's role
     * @param expiresAt - Token expiration timestamp (ISO string)
     */
    setAuth(userId: string, username: string, role: UserRole, expiresAt: string): void {
        const expiresAtMs = new Date(expiresAt).getTime();

        this.state = {
            isAuthenticated: true,
            userId,
            username,
            role,
            expiresAt: expiresAtMs,
        };

        this.saveState();
        this.notifyListeners();
        logger.debug('Auth state set', { userId, username, role });
    }

    /**
     * Clear authentication state on logout.
     */
    clearAuth(): void {
        this.state = {
            isAuthenticated: false,
            userId: null,
            username: null,
            role: null,
            expiresAt: null,
        };

        this.saveState();
        this.notifyListeners();
        logger.debug('Auth state cleared');
    }

    /**
     * Refresh authentication state from storage.
     * Useful after storage has been modified externally.
     */
    refresh(): void {
        this.state = this.loadState();
        this.notifyListeners();
    }

    // ========================================================================
    // Listeners
    // ========================================================================

    /**
     * Add a listener for auth state changes.
     * Returns a cleanup function to remove the listener.
     */
    addListener(listener: AuthStateListener): () => void {
        this.listeners.add(listener);
        return () => {
            this.listeners.delete(listener);
        };
    }

    /**
     * Remove a specific listener.
     */
    removeListener(listener: AuthStateListener): void {
        this.listeners.delete(listener);
    }

    /**
     * Remove all listeners.
     */
    clearListeners(): void {
        this.listeners.clear();
    }
}

// Export singleton accessor
export const AuthContext = {
    getInstance: () => AuthContextImpl.getInstance(),
    resetInstance: () => AuthContextImpl.resetInstance(),
};

// Export type for the instance
export type AuthContextInstance = AuthContextImpl;
