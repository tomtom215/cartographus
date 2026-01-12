// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Unit Tests for Form Validation Utilities
 *
 * Tests the validators module which provides WCAG 2.1 compliant
 * form validation rules for the FormValidator class.
 *
 * These tests verify that the validator factory functions return
 * correctly structured ValidationRule objects.
 *
 * Run with: npx tsx --test src/lib/form-validation.test.ts
 */

import { describe, it } from 'node:test';
import assert from 'node:assert';
import { validators } from './form-validation';

describe('validators.required', () => {
    it('returns a ValidationRule with required: true', () => {
        const rule = validators.required();
        assert.strictEqual(rule.required, true);
    });

    it('includes custom message when provided', () => {
        const rule = validators.required('Field is required');
        assert.strictEqual(rule.required, true);
        assert.strictEqual(rule.messages?.required, 'Field is required');
    });

    it('returns undefined messages when no custom message', () => {
        const rule = validators.required();
        assert.strictEqual(rule.messages, undefined);
    });
});

describe('validators.email', () => {
    it('returns a ValidationRule with email: true', () => {
        const rule = validators.email();
        assert.strictEqual(rule.email, true);
    });

    it('includes custom message when provided', () => {
        const rule = validators.email('Invalid email address');
        assert.strictEqual(rule.email, true);
        assert.strictEqual(rule.messages?.email, 'Invalid email address');
    });
});

describe('validators.minLength', () => {
    it('returns a ValidationRule with minLength set', () => {
        const rule = validators.minLength(5);
        assert.strictEqual(rule.minLength, 5);
    });

    it('includes custom message when provided', () => {
        const rule = validators.minLength(5, 'Must be at least 5 characters');
        assert.strictEqual(rule.minLength, 5);
        assert.strictEqual(rule.messages?.minLength, 'Must be at least 5 characters');
    });
});

describe('validators.maxLength', () => {
    it('returns a ValidationRule with maxLength set', () => {
        const rule = validators.maxLength(100);
        assert.strictEqual(rule.maxLength, 100);
    });

    it('includes custom message when provided', () => {
        const rule = validators.maxLength(100, 'Too long');
        assert.strictEqual(rule.maxLength, 100);
        assert.strictEqual(rule.messages?.maxLength, 'Too long');
    });
});

describe('validators.pattern', () => {
    it('returns a ValidationRule with pattern set', () => {
        const regex = /^\d{3}-\d{4}$/;
        const rule = validators.pattern(regex);
        assert.strictEqual(rule.pattern, regex);
    });

    it('includes custom message when provided', () => {
        const regex = /^\d+$/;
        const rule = validators.pattern(regex, 'Must be numeric');
        assert.strictEqual(rule.pattern, regex);
        assert.strictEqual(rule.messages?.pattern, 'Must be numeric');
    });
});

describe('validators.password', () => {
    it('returns a ValidationRule with password: true and required: true', () => {
        const rule = validators.password();
        assert.strictEqual(rule.password, true);
        assert.strictEqual(rule.required, true);
    });
});

describe('validators.passwordConfirm', () => {
    it('returns a ValidationRule with match field name', () => {
        const rule = validators.passwordConfirm('password');
        assert.strictEqual(rule.match, 'password');
        assert.strictEqual(rule.required, true);
    });

    it('includes default match message', () => {
        const rule = validators.passwordConfirm('password');
        assert.strictEqual(rule.messages?.match, 'Passwords do not match');
    });
});

describe('validators.username', () => {
    it('returns a complete username validation rule', () => {
        const rule = validators.username();
        assert.strictEqual(rule.required, true);
        assert.strictEqual(rule.minLength, 3);
        assert.strictEqual(rule.maxLength, 50);
        assert.ok(rule.pattern instanceof RegExp);
    });

    it('includes all username messages', () => {
        const rule = validators.username();
        assert.ok(rule.messages?.required);
        assert.ok(rule.messages?.minLength);
        assert.ok(rule.messages?.maxLength);
        assert.ok(rule.messages?.pattern);
    });

    it('pattern matches valid usernames', () => {
        const rule = validators.username();
        const pattern = rule.pattern as RegExp;
        assert.strictEqual(pattern.test('valid_user'), true);
        assert.strictEqual(pattern.test('user123'), true);
        assert.strictEqual(pattern.test('User-Name'), true);
    });

    it('pattern rejects invalid usernames', () => {
        const rule = validators.username();
        const pattern = rule.pattern as RegExp;
        assert.strictEqual(pattern.test('user@name'), false);
        assert.strictEqual(pattern.test('user name'), false);
        assert.strictEqual(pattern.test('user.name'), false);
    });
});

describe('validators.combine', () => {
    it('merges multiple rules into one', () => {
        const rule = validators.combine(
            validators.required('Required'),
            validators.minLength(5, 'Min 5'),
            validators.maxLength(20, 'Max 20')
        );

        assert.strictEqual(rule.required, true);
        assert.strictEqual(rule.minLength, 5);
        assert.strictEqual(rule.maxLength, 20);
    });

    it('merges messages from all rules', () => {
        const rule = validators.combine(
            validators.required('Required'),
            validators.minLength(5, 'Min 5'),
            validators.email('Invalid email')
        );

        assert.strictEqual(rule.messages?.required, 'Required');
        assert.strictEqual(rule.messages?.minLength, 'Min 5');
        assert.strictEqual(rule.messages?.email, 'Invalid email');
    });

    it('later rules override earlier rules for same property', () => {
        const rule = validators.combine(
            validators.minLength(5, 'At least 5'),
            validators.minLength(10, 'At least 10')
        );

        assert.strictEqual(rule.minLength, 10);
        assert.strictEqual(rule.messages?.minLength, 'At least 10');
    });

    it('returns empty object for no rules', () => {
        const rule = validators.combine();
        // Empty object with no rules
        assert.deepStrictEqual(Object.keys(rule), []);
    });
});

describe('ValidationRule structure', () => {
    it('all validators return objects with expected shape', () => {
        const rules = [
            validators.required(),
            validators.email(),
            validators.minLength(1),
            validators.maxLength(100),
            validators.pattern(/./),
            validators.password(),
            validators.passwordConfirm('pass'),
            validators.username(),
        ];

        for (const rule of rules) {
            assert.strictEqual(typeof rule, 'object');
            assert.notStrictEqual(rule, null);
        }
    });
});

describe('edge cases', () => {
    it('minLength with 0 is valid', () => {
        const rule = validators.minLength(0);
        assert.strictEqual(rule.minLength, 0);
    });

    it('maxLength with large number is valid', () => {
        const rule = validators.maxLength(1000000);
        assert.strictEqual(rule.maxLength, 1000000);
    });

    it('pattern with complex regex is preserved', () => {
        const complexRegex = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)[a-zA-Z\d]{8,}$/;
        const rule = validators.pattern(complexRegex);
        assert.strictEqual(rule.pattern, complexRegex);
    });

    it('combine handles single rule', () => {
        const rule = validators.combine(validators.required('Required'));
        assert.strictEqual(rule.required, true);
        assert.strictEqual(rule.messages?.required, 'Required');
    });

    it('validators can be reused', () => {
        const requiredRule = validators.required('Field required');
        const rule1 = validators.combine(requiredRule, validators.minLength(5));
        const rule2 = validators.combine(requiredRule, validators.maxLength(50));

        assert.strictEqual(rule1.required, true);
        assert.strictEqual(rule1.minLength, 5);
        assert.strictEqual(rule2.required, true);
        assert.strictEqual(rule2.maxLength, 50);
    });
});
