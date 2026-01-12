// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Unit Tests for Sanitization Utilities
 *
 * These tests verify XSS prevention functions.
 * Run with: npx tsx --test sanitize.test.ts
 */

import { describe, it } from 'node:test';
import assert from 'node:assert';
import { escapeHtml, escapeAttribute, escapeJs, sanitizeUrl, safeHtml } from './sanitize';

describe('escapeHtml', () => {
    it('escapes HTML special characters', () => {
        assert.strictEqual(
            escapeHtml('<script>alert("xss")</script>'),
            '&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;'
        );
    });

    it('escapes ampersands', () => {
        assert.strictEqual(escapeHtml('Tom & Jerry'), 'Tom &amp; Jerry');
    });

    it('escapes single quotes', () => {
        assert.strictEqual(escapeHtml("User's input"), 'User&#039;s input');
    });

    it('escapes all special characters together', () => {
        assert.strictEqual(
            escapeHtml('<div class="test">O\'Brien & Co.</div>'),
            '&lt;div class=&quot;test&quot;&gt;O&#039;Brien &amp; Co.&lt;/div&gt;'
        );
    });

    it('handles null and undefined', () => {
        assert.strictEqual(escapeHtml(null), '');
        assert.strictEqual(escapeHtml(undefined), '');
    });

    it('handles empty string', () => {
        assert.strictEqual(escapeHtml(''), '');
    });

    it('preserves safe strings', () => {
        assert.strictEqual(escapeHtml('Hello World'), 'Hello World');
        assert.strictEqual(escapeHtml('12345'), '12345');
    });

    it('handles numbers converted to string', () => {
        assert.strictEqual(escapeHtml(123 as unknown as string), '123');
    });
});

describe('escapeAttribute', () => {
    it('escapes attribute values', () => {
        assert.strictEqual(
            escapeAttribute('value with "quotes"'),
            'value with &quot;quotes&quot;'
        );
    });

    it('handles null and undefined', () => {
        assert.strictEqual(escapeAttribute(null), '');
        assert.strictEqual(escapeAttribute(undefined), '');
    });
});

describe('escapeJs', () => {
    it('escapes JavaScript string special characters', () => {
        assert.strictEqual(escapeJs("test'value"), "test\\'value");
        assert.strictEqual(escapeJs('test"value'), 'test\\"value');
    });

    it('escapes backslashes', () => {
        assert.strictEqual(escapeJs('path\\to\\file'), 'path\\\\to\\\\file');
    });

    it('escapes newlines', () => {
        assert.strictEqual(escapeJs('line1\nline2'), 'line1\\nline2');
        assert.strictEqual(escapeJs('line1\rline2'), 'line1\\rline2');
    });

    it('escapes HTML brackets to prevent script injection', () => {
        assert.strictEqual(escapeJs('<script>'), '\\x3cscript\\x3e');
    });

    it('handles null and undefined', () => {
        assert.strictEqual(escapeJs(null), '');
        assert.strictEqual(escapeJs(undefined), '');
    });
});

describe('sanitizeUrl', () => {
    it('allows safe URLs', () => {
        assert.strictEqual(sanitizeUrl('https://example.com'), 'https://example.com');
        assert.strictEqual(
            sanitizeUrl('http://example.com/path?query=1'),
            'http://example.com/path?query=1'
        );
        assert.strictEqual(sanitizeUrl('/relative/path'), '/relative/path');
        assert.strictEqual(sanitizeUrl('#anchor'), '#anchor');
    });

    it('blocks javascript: URLs', () => {
        assert.strictEqual(sanitizeUrl('javascript:alert(1)'), '');
        assert.strictEqual(sanitizeUrl('JAVASCRIPT:alert(1)'), '');
        assert.strictEqual(sanitizeUrl('  javascript:alert(1)'), '');
    });

    it('blocks data: URLs', () => {
        assert.strictEqual(sanitizeUrl('data:text/html,<script>alert(1)</script>'), '');
        assert.strictEqual(sanitizeUrl('DATA:text/html,test'), '');
    });

    it('blocks vbscript: URLs', () => {
        assert.strictEqual(sanitizeUrl('vbscript:msgbox(1)'), '');
    });

    it('handles null and undefined', () => {
        assert.strictEqual(sanitizeUrl(null), '');
        assert.strictEqual(sanitizeUrl(undefined), '');
    });

    it('handles empty string', () => {
        assert.strictEqual(sanitizeUrl(''), '');
    });
});

describe('safeHtml', () => {
    it('automatically escapes interpolated values', () => {
        const userInput = '<script>alert("xss")</script>';
        const result = safeHtml`<div>${userInput}</div>`;
        assert.strictEqual(
            result,
            '<div>&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;</div>'
        );
    });

    it('handles multiple interpolations', () => {
        const name = "O'Brien";
        const company = 'Tom & Jerry Inc.';
        const result = safeHtml`<span>${name}</span> works at <span>${company}</span>`;
        assert.strictEqual(
            result,
            '<span>O&#039;Brien</span> works at <span>Tom &amp; Jerry Inc.</span>'
        );
    });

    it('handles numbers', () => {
        const count = 42;
        const result = safeHtml`<span>Count: ${count}</span>`;
        assert.strictEqual(result, '<span>Count: 42</span>');
    });
});
