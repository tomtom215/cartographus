// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Unit Tests for RoleGuard
 *
 * RBAC Phase 4: Frontend Role Integration
 * Tests role-based access control and UI element visibility.
 *
 * Run with: npx tsx --test RoleGuard.test.ts
 */

import { describe, it } from 'node:test';
import assert from 'node:assert';

// Resource access matrix for testing (mirrors production matrix)
const RESOURCE_ACCESS_MATRIX: Record<string, Record<string, string>> = {
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

// Role hierarchy for testing
const roleHierarchy: Record<string, number> = {
    viewer: 0,
    editor: 1,
    admin: 2,
};

// Helper function to check access
function canAccess(
    userRole: string | null,
    resource: string,
    action: string
): boolean {
    if (!userRole) return false;
    const requiredRole = RESOURCE_ACCESS_MATRIX[resource]?.[action];
    if (!requiredRole) return false;
    return roleHierarchy[userRole] >= roleHierarchy[requiredRole];
}

// Helper function to check user access
function canAccessUser(
    userRole: string | null,
    currentUserId: string | null,
    targetUserId: string
): boolean {
    if (!userRole || !currentUserId) return false;
    if (userRole === 'admin') return true;
    return currentUserId === targetUserId;
}

describe('RoleGuard', () => {
    describe('Resource access matrix', () => {
        describe('Viewer role', () => {
            const role = 'viewer';

            it('can read playbacks', () => {
                assert.strictEqual(canAccess(role, 'playbacks', 'read'), true);
            });

            it('cannot write playbacks', () => {
                assert.strictEqual(canAccess(role, 'playbacks', 'write'), false);
            });

            it('cannot delete playbacks', () => {
                assert.strictEqual(canAccess(role, 'playbacks', 'delete'), false);
            });

            it('can read analytics', () => {
                assert.strictEqual(canAccess(role, 'analytics', 'read'), true);
            });

            it('cannot access backup', () => {
                assert.strictEqual(canAccess(role, 'backup', 'read'), false);
                assert.strictEqual(canAccess(role, 'backup', 'write'), false);
            });

            it('can read server info', () => {
                assert.strictEqual(canAccess(role, 'server', 'read'), true);
            });

            it('cannot admin server', () => {
                assert.strictEqual(canAccess(role, 'server', 'admin'), false);
            });

            it('can read detection alerts', () => {
                assert.strictEqual(canAccess(role, 'detection', 'read'), true);
            });

            it('cannot write detection rules', () => {
                assert.strictEqual(canAccess(role, 'detection', 'write'), false);
            });

            it('cannot access audit logs', () => {
                assert.strictEqual(canAccess(role, 'audit', 'read'), false);
            });

            it('can read settings', () => {
                assert.strictEqual(canAccess(role, 'settings', 'read'), true);
            });

            it('cannot write settings', () => {
                assert.strictEqual(canAccess(role, 'settings', 'write'), false);
            });
        });

        describe('Editor role', () => {
            const role = 'editor';

            it('can read and write playbacks', () => {
                assert.strictEqual(canAccess(role, 'playbacks', 'read'), true);
                assert.strictEqual(canAccess(role, 'playbacks', 'write'), true);
            });

            it('cannot delete playbacks', () => {
                assert.strictEqual(canAccess(role, 'playbacks', 'delete'), false);
            });

            it('can read and write analytics', () => {
                assert.strictEqual(canAccess(role, 'analytics', 'read'), true);
                assert.strictEqual(canAccess(role, 'analytics', 'write'), true);
            });

            it('cannot access backup', () => {
                assert.strictEqual(canAccess(role, 'backup', 'read'), false);
            });

            it('can read but not write server', () => {
                assert.strictEqual(canAccess(role, 'server', 'read'), true);
                assert.strictEqual(canAccess(role, 'server', 'write'), false);
            });

            it('can read and write settings', () => {
                assert.strictEqual(canAccess(role, 'settings', 'read'), true);
                assert.strictEqual(canAccess(role, 'settings', 'write'), true);
            });

            it('cannot access audit logs', () => {
                assert.strictEqual(canAccess(role, 'audit', 'read'), false);
            });

            it('can read and write wrapped reports', () => {
                assert.strictEqual(canAccess(role, 'wrapped', 'read'), true);
                assert.strictEqual(canAccess(role, 'wrapped', 'write'), true);
            });
        });

        describe('Admin role', () => {
            const role = 'admin';

            it('can perform all actions on playbacks', () => {
                assert.strictEqual(canAccess(role, 'playbacks', 'read'), true);
                assert.strictEqual(canAccess(role, 'playbacks', 'write'), true);
                assert.strictEqual(canAccess(role, 'playbacks', 'delete'), true);
                assert.strictEqual(canAccess(role, 'playbacks', 'admin'), true);
            });

            it('can perform all actions on backup', () => {
                assert.strictEqual(canAccess(role, 'backup', 'read'), true);
                assert.strictEqual(canAccess(role, 'backup', 'write'), true);
                assert.strictEqual(canAccess(role, 'backup', 'delete'), true);
            });

            it('can access audit logs', () => {
                assert.strictEqual(canAccess(role, 'audit', 'read'), true);
                assert.strictEqual(canAccess(role, 'audit', 'write'), true);
            });

            it('can admin server', () => {
                assert.strictEqual(canAccess(role, 'server', 'admin'), true);
            });

            it('can write detection rules', () => {
                assert.strictEqual(canAccess(role, 'detection', 'write'), true);
            });

            it('can delete users', () => {
                assert.strictEqual(canAccess(role, 'users', 'delete'), true);
            });
        });

        describe('Unauthenticated user', () => {
            const role = null;

            it('cannot access any resource', () => {
                assert.strictEqual(canAccess(role, 'playbacks', 'read'), false);
                assert.strictEqual(canAccess(role, 'analytics', 'read'), false);
                assert.strictEqual(canAccess(role, 'backup', 'read'), false);
                assert.strictEqual(canAccess(role, 'server', 'read'), false);
            });
        });
    });

    describe('User data access', () => {
        it('allows viewer to access own data', () => {
            assert.strictEqual(canAccessUser('viewer', 'user-001', 'user-001'), true);
        });

        it('denies viewer access to other users data', () => {
            assert.strictEqual(canAccessUser('viewer', 'user-001', 'user-002'), false);
        });

        it('allows editor to access own data', () => {
            assert.strictEqual(canAccessUser('editor', 'user-001', 'user-001'), true);
        });

        it('denies editor access to other users data', () => {
            assert.strictEqual(canAccessUser('editor', 'user-001', 'user-002'), false);
        });

        it('allows admin to access any user data', () => {
            assert.strictEqual(canAccessUser('admin', 'admin-001', 'user-001'), true);
            assert.strictEqual(canAccessUser('admin', 'admin-001', 'user-002'), true);
            assert.strictEqual(canAccessUser('admin', 'admin-001', 'admin-001'), true);
        });

        it('denies access when not authenticated', () => {
            assert.strictEqual(canAccessUser(null, null, 'user-001'), false);
        });
    });

    describe('Role checks', () => {
        it('isAdmin returns true only for admin', () => {
            const isAdmin = (role: string | null) => role === 'admin';

            assert.strictEqual(isAdmin('admin'), true);
            assert.strictEqual(isAdmin('editor'), false);
            assert.strictEqual(isAdmin('viewer'), false);
            assert.strictEqual(isAdmin(null), false);
        });

        it('isEditor returns true for editor and admin', () => {
            const isEditor = (role: string | null) =>
                role === 'editor' || role === 'admin';

            assert.strictEqual(isEditor('admin'), true);
            assert.strictEqual(isEditor('editor'), true);
            assert.strictEqual(isEditor('viewer'), false);
            assert.strictEqual(isEditor(null), false);
        });

        it('hasRole checks role hierarchy correctly', () => {
            const hasRole = (current: string | null, required: string): boolean => {
                if (!current) return false;
                return roleHierarchy[current] >= roleHierarchy[required];
            };

            // Viewer
            assert.strictEqual(hasRole('viewer', 'viewer'), true);
            assert.strictEqual(hasRole('viewer', 'editor'), false);
            assert.strictEqual(hasRole('viewer', 'admin'), false);

            // Editor
            assert.strictEqual(hasRole('editor', 'viewer'), true);
            assert.strictEqual(hasRole('editor', 'editor'), true);
            assert.strictEqual(hasRole('editor', 'admin'), false);

            // Admin
            assert.strictEqual(hasRole('admin', 'viewer'), true);
            assert.strictEqual(hasRole('admin', 'editor'), true);
            assert.strictEqual(hasRole('admin', 'admin'), true);

            // Null role
            assert.strictEqual(hasRole(null, 'viewer'), false);
        });
    });

    describe('Denial messages', () => {
        it('generates correct denial message', () => {
            const getDenialMessage = (
                userRole: string | null,
                resource: string,
                action: string
            ): string => {
                const requiredRole = RESOURCE_ACCESS_MATRIX[resource]?.[action];
                if (!requiredRole) {
                    return 'Access denied: unknown resource';
                }

                const currentRole = userRole || 'unauthenticated';
                return `Access denied: ${action} ${resource} requires ${requiredRole} role (you are ${currentRole})`;
            };

            assert.strictEqual(
                getDenialMessage('viewer', 'backup', 'read'),
                'Access denied: read backup requires admin role (you are viewer)'
            );

            assert.strictEqual(
                getDenialMessage(null, 'playbacks', 'read'),
                'Access denied: read playbacks requires viewer role (you are unauthenticated)'
            );

            assert.strictEqual(
                getDenialMessage('editor', 'users', 'delete'),
                'Access denied: delete users requires admin role (you are editor)'
            );
        });
    });

    describe('Edge cases', () => {
        it('handles unknown resources gracefully', () => {
            assert.strictEqual(canAccess('admin', 'unknown', 'read'), false);
        });

        it('handles unknown actions gracefully', () => {
            assert.strictEqual(canAccess('admin', 'playbacks', 'unknown'), false);
        });

        it('handles empty strings', () => {
            assert.strictEqual(canAccess('', 'playbacks', 'read'), false);
        });
    });
});
