// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Sanitization utilities for preventing XSS and other injection attacks.
 *
 * This module provides centralized HTML escaping functions that should be used
 * throughout the application when inserting user-provided or external data
 * into HTML templates.
 *
 * @module sanitize
 * @example
 * import { escapeHtml, escapeAttribute } from './lib/sanitize';
 *
 * // In template literals:
 * const html = `<div class="title">${escapeHtml(userInput)}</div>`;
 * const attr = `<input value="${escapeAttribute(userInput)}">`;
 */

/**
 * Escapes HTML special characters to prevent XSS attacks.
 *
 * Converts the following characters to their HTML entity equivalents:
 * - & → &amp;
 * - < → &lt;
 * - > → &gt;
 * - " → &quot;
 * - ' → &#039;
 *
 * @param unsafe - The potentially unsafe string to escape
 * @returns The escaped string safe for HTML content insertion
 *
 * @example
 * escapeHtml('<script>alert("xss")</script>')
 * // Returns: '&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;'
 *
 * escapeHtml("User's input & more")
 * // Returns: 'User&#039;s input &amp; more'
 */
export function escapeHtml(unsafe: string | null | undefined): string {
    if (unsafe == null) {
        return '';
    }
    return String(unsafe)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

/**
 * Escapes a string for safe use in HTML attributes.
 * This is an alias for escapeHtml as the same escaping rules apply.
 *
 * @param unsafe - The potentially unsafe string to escape
 * @returns The escaped string safe for HTML attribute insertion
 *
 * @example
 * const html = `<img src="${escapeAttribute(url)}" alt="${escapeAttribute(title)}">`;
 */
export function escapeAttribute(unsafe: string | null | undefined): string {
    return escapeHtml(unsafe);
}

/**
 * Escapes a string for safe use in JavaScript string literals within HTML.
 * Use this when embedding data in inline scripts or event handlers.
 *
 * @param unsafe - The potentially unsafe string to escape
 * @returns The escaped string safe for JavaScript string context
 *
 * @example
 * const html = `<button onclick="handleClick('${escapeJs(userInput)}')">`;
 */
export function escapeJs(unsafe: string | null | undefined): string {
    if (unsafe == null) {
        return '';
    }
    return String(unsafe)
        .replace(/\\/g, '\\\\')
        .replace(/'/g, "\\'")
        .replace(/"/g, '\\"')
        .replace(/\n/g, '\\n')
        .replace(/\r/g, '\\r')
        .replace(/</g, '\\x3c')
        .replace(/>/g, '\\x3e');
}

/**
 * Sanitizes a URL to prevent javascript: and data: URL injection.
 * Returns an empty string if the URL scheme is potentially dangerous.
 *
 * @param url - The URL to sanitize
 * @returns The original URL if safe, or empty string if dangerous
 *
 * @example
 * sanitizeUrl('https://example.com') // Returns: 'https://example.com'
 * sanitizeUrl('javascript:alert(1)') // Returns: ''
 * sanitizeUrl('data:text/html,...')  // Returns: ''
 */
export function sanitizeUrl(url: string | null | undefined): string {
    if (url == null) {
        return '';
    }
    const trimmed = String(url).trim().toLowerCase();

    // Block dangerous URL schemes
    if (trimmed.indexOf('javascript:') === 0 ||
        trimmed.indexOf('data:') === 0 ||
        trimmed.indexOf('vbscript:') === 0) {
        return '';
    }

    return url;
}

/**
 * Creates a safe HTML string from a template and values.
 * All values are automatically escaped.
 *
 * @param strings - Template literal strings
 * @param values - Values to interpolate (will be escaped)
 * @returns The safe HTML string
 *
 * @example
 * const html = safeHtml`<div class="user">${userName}</div>`;
 */
export function safeHtml(strings: TemplateStringsArray, ...values: unknown[]): string {
    let result = strings[0];
    for (let i = 0; i < values.length; i++) {
        result += escapeHtml(String(values[i])) + strings[i + 1];
    }
    return result;
}
