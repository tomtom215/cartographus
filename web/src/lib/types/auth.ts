// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Authentication types
 */

export interface LoginRequest {
    username: string;
    password: string;
    remember_me: boolean;
}

export interface LoginResponse {
    token: string;
    expires_at: string;
    username: string;
    role: UserRole;
    user_id: string;
}

/**
 * User roles for authorization.
 * - viewer: Read-only access to own data (default)
 * - editor: Can write/modify data, inherits viewer permissions
 * - admin: Full access including user management, inherits editor permissions
 */
export type UserRole = 'viewer' | 'editor' | 'admin';

/**
 * Checks if a role has at least the required permission level.
 * Role hierarchy: viewer < editor < admin
 */
export function hasRolePermission(currentRole: UserRole, requiredRole: UserRole): boolean {
    const roleHierarchy: Record<UserRole, number> = {
        viewer: 0,
        editor: 1,
        admin: 2,
    };
    return roleHierarchy[currentRole] >= roleHierarchy[requiredRole];
}

/**
 * Checks if a role is valid.
 */
export function isValidRole(role: string): role is UserRole {
    return role === 'viewer' || role === 'editor' || role === 'admin';
}

// ============================================
// Session Management Types
// ADR-0015: Zero Trust Authentication
// ============================================

/**
 * Represents an active user session.
 */
export interface SessionInfo {
    /** Unique session identifier */
    id: string;
    /** Authentication provider (oidc, plex, jwt, basic) */
    provider: string;
    /** When the session was created */
    created_at: string;
    /** When the session was last accessed */
    last_accessed_at?: string;
    /** Whether this is the current session */
    current: boolean;
    /** IP address from which session was created (if available) */
    ip_address?: string;
    /** User agent string (if available) */
    user_agent?: string;
}

/**
 * Response from the sessions list endpoint.
 */
export interface SessionsResponse {
    sessions: SessionInfo[];
}

/**
 * Response from logout/revoke operations.
 */
export interface LogoutResponse {
    message: string;
    sessions_count?: number;
}
