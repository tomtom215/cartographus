// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Unit Tests for Auth Types
 *
 * RBAC Phase 4: Frontend Role Integration
 * Tests role validation and permission helper functions.
 *
 * Run with: npx tsx --test auth.test.ts
 */

import { describe, it } from 'node:test';
import assert from 'node:assert';
import { hasRolePermission, isValidRole } from './auth';

describe('Auth Type Helpers', () => {
    describe('hasRolePermission', () => {
        it('viewer has viewer permission', () => {
            assert.strictEqual(hasRolePermission('viewer', 'viewer'), true);
        });

        it('viewer does not have editor permission', () => {
            assert.strictEqual(hasRolePermission('viewer', 'editor'), false);
        });

        it('viewer does not have admin permission', () => {
            assert.strictEqual(hasRolePermission('viewer', 'admin'), false);
        });

        it('editor has viewer permission', () => {
            assert.strictEqual(hasRolePermission('editor', 'viewer'), true);
        });

        it('editor has editor permission', () => {
            assert.strictEqual(hasRolePermission('editor', 'editor'), true);
        });

        it('editor does not have admin permission', () => {
            assert.strictEqual(hasRolePermission('editor', 'admin'), false);
        });

        it('admin has viewer permission', () => {
            assert.strictEqual(hasRolePermission('admin', 'viewer'), true);
        });

        it('admin has editor permission', () => {
            assert.strictEqual(hasRolePermission('admin', 'editor'), true);
        });

        it('admin has admin permission', () => {
            assert.strictEqual(hasRolePermission('admin', 'admin'), true);
        });
    });

    describe('isValidRole', () => {
        it('validates viewer as valid', () => {
            assert.strictEqual(isValidRole('viewer'), true);
        });

        it('validates editor as valid', () => {
            assert.strictEqual(isValidRole('editor'), true);
        });

        it('validates admin as valid', () => {
            assert.strictEqual(isValidRole('admin'), true);
        });

        it('rejects invalid roles', () => {
            assert.strictEqual(isValidRole('superuser'), false);
            assert.strictEqual(isValidRole('root'), false);
            assert.strictEqual(isValidRole(''), false);
            assert.strictEqual(isValidRole('ADMIN'), false);
            assert.strictEqual(isValidRole('Viewer'), false);
        });
    });
});
