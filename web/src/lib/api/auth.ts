// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Authentication API Module
 *
 * Handles login, logout, and authentication state management.
 * Uses SafeStorage for robust token persistence with private browsing fallback.
 * Integrates with AuthContext for role-based access control.
 *
 * RBAC Phase 4: Frontend Role Integration
 */

import type { LoginRequest, LoginResponse, UserRole, SessionsResponse, LogoutResponse } from '../types/auth';
import { BaseAPIClient } from './client';
import { SafeStorage } from '../utils/SafeStorage';
import { AuthContext } from '../auth/AuthContext';

// Storage keys for auth data
const AUTH_KEYS = {
    TOKEN: 'auth_token',
    USERNAME: 'auth_username',
    USER_ID: 'auth_user_id',
    ROLE: 'auth_role',
    EXPIRES_AT: 'auth_expires_at',
} as const;

/**
 * Authentication API methods
 */
export class AuthAPI extends BaseAPIClient {
    /**
     * Login with credentials and update auth context.
     */
    async login(credentials: LoginRequest): Promise<LoginResponse> {
        const response = await this.fetch<LoginResponse>('/auth/login', {
            method: 'POST',
            body: JSON.stringify(credentials),
        });

        const { token, username, role, user_id, expires_at } = response.data;

        // Store in SafeStorage
        this.setToken(token);
        SafeStorage.setItem(AUTH_KEYS.TOKEN, token);
        SafeStorage.setItem(AUTH_KEYS.USERNAME, username);
        SafeStorage.setItem(AUTH_KEYS.USER_ID, user_id);
        SafeStorage.setItem(AUTH_KEYS.ROLE, role);
        SafeStorage.setItem(AUTH_KEYS.EXPIRES_AT, expires_at);

        // Update AuthContext
        AuthContext.getInstance().setAuth(user_id, username, role, expires_at);

        return response.data;
    }

    /**
     * Logout and clear auth context.
     */
    logout(): void {
        this.setToken(null);
        SafeStorage.removeItem(AUTH_KEYS.TOKEN);
        SafeStorage.removeItem(AUTH_KEYS.USERNAME);
        SafeStorage.removeItem(AUTH_KEYS.USER_ID);
        SafeStorage.removeItem(AUTH_KEYS.ROLE);
        SafeStorage.removeItem(AUTH_KEYS.EXPIRES_AT);

        // Clear AuthContext
        AuthContext.getInstance().clearAuth();
    }

    /**
     * Check if user is authenticated.
     */
    isAuthenticated(): boolean {
        return AuthContext.getInstance().isAuthenticated();
    }

    /**
     * Get the current user's username.
     */
    getUsername(): string | null {
        return AuthContext.getInstance().getUsername();
    }

    /**
     * Get the current user's role.
     */
    getRole(): UserRole | null {
        return AuthContext.getInstance().getRole();
    }

    /**
     * Get the current user's ID.
     */
    getUserId(): string | null {
        return AuthContext.getInstance().getUserId();
    }

    /**
     * Check if the current user is an admin.
     */
    isAdmin(): boolean {
        return AuthContext.getInstance().isAdmin();
    }

    /**
     * Check if the current user is an editor or higher.
     */
    isEditor(): boolean {
        return AuthContext.getInstance().isEditor();
    }

    // ============================================
    // Session Management
    // ADR-0015: Zero Trust Authentication
    // ============================================

    /**
     * Get all active sessions for the current user.
     * Each session includes provider, creation time, last access, and current flag.
     */
    async getSessions(): Promise<SessionsResponse> {
        const response = await this.fetch<SessionsResponse>('/oidc/sessions', {
            method: 'GET',
        });
        return response.data;
    }

    /**
     * Revoke a specific session by ID.
     * Users can revoke their own sessions; admins can revoke any session.
     * @param sessionId - The ID of the session to revoke
     */
    async revokeSession(sessionId: string): Promise<LogoutResponse> {
        const response = await this.fetch<LogoutResponse>(`/oidc/sessions/${sessionId}`, {
            method: 'DELETE',
        });
        return response.data;
    }

    /**
     * Logout from all sessions ("Sign out everywhere").
     * Destroys all sessions for the current user and clears local auth state.
     */
    async logoutAll(): Promise<LogoutResponse> {
        const response = await this.fetch<LogoutResponse>('/oidc/logout/all', {
            method: 'POST',
        });

        // Clear local auth state
        this.logout();

        return response.data;
    }

    /**
     * Get user info for the current session.
     */
    async getUserInfo(): Promise<Record<string, unknown>> {
        const response = await this.fetch<Record<string, unknown>>('/oidc/userinfo', {
            method: 'GET',
        });
        return response.data;
    }
}
