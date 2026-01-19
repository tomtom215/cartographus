#!/usr/bin/env node
/**
 * Cartographus - Media Server Analytics and Geographic Visualization
 * Copyright 2026 Tom F. (tomtom215)
 * SPDX-License-Identifier: AGPL-3.0-or-later
 * https://github.com/tomtom215/cartographus
 * =============================================================================
 * HTML Composition Script
 * =============================================================================
 * Combines partial HTML files into a single index.html using {{include "path"}}
 * markers. This allows the 4,109-line index.html to be split into maintainable
 * partial files that fit within AI context windows.
 *
 * Usage:
 *   node build/compose-html.js [--watch] [--verify]
 *
 * Options:
 *   --watch   Watch for changes and recompose automatically
 *   --verify  Verify composed output matches expected line count
 *
 * Include Syntax:
 *   {{include "path/to/file.html"}}
 *
 * Notes:
 *   - Paths are relative to the partials/ directory
 *   - Recursive includes supported (max depth 10)
 *   - Syntax chosen to avoid conflicts with Go templates ({{.Nonce}})
 * =============================================================================
 */

'use strict';

const fs = require('fs');
const path = require('path');

// Configuration
const PARTIALS_DIR = path.join(__dirname, '../partials');
const OUTPUT_FILE = path.join(__dirname, '../public/index.html');
const MAX_INCLUDE_DEPTH = 10;
const EXPECTED_LINE_COUNT = 4109;

// Include pattern: {{include "path/to/file.html"}}
const INCLUDE_PATTERN = /\{\{include\s+"([^"]+)"\}\}/g;

/**
 * Recursively resolve includes in HTML content
 * @param {string} content - HTML content with include markers
 * @param {string} basePath - Base path for resolving relative includes
 * @param {number} depth - Current recursion depth (for circular reference protection)
 * @param {Set<string>} visited - Set of visited file paths (for circular reference detection)
 * @returns {string} Content with all includes resolved
 */
function resolveIncludes(content, basePath = PARTIALS_DIR, depth = 0, visited = new Set()) {
    if (depth > MAX_INCLUDE_DEPTH) {
        console.error(`ERROR: Maximum include depth (${MAX_INCLUDE_DEPTH}) exceeded`);
        console.error('This may indicate a circular reference in your partials');
        process.exit(1);
    }

    return content.replace(INCLUDE_PATTERN, (match, includePath) => {
        const fullPath = path.resolve(basePath, includePath);
        const normalizedPath = path.normalize(fullPath);

        // Verify the path is within the partials directory (security check)
        if (!normalizedPath.startsWith(path.normalize(PARTIALS_DIR))) {
            console.error(`ERROR: Include path escapes partials directory: ${includePath}`);
            console.error(`  Resolved to: ${fullPath}`);
            console.error(`  Partials dir: ${PARTIALS_DIR}`);
            process.exit(1);
        }

        // Check for circular references
        if (visited.has(normalizedPath)) {
            console.error(`ERROR: Circular reference detected: ${includePath}`);
            console.error(`  Path: ${normalizedPath}`);
            console.error(`  Already visited: ${Array.from(visited).join(' -> ')}`);
            process.exit(1);
        }

        // Check if file exists
        if (!fs.existsSync(fullPath)) {
            console.error(`ERROR: Include file not found: ${includePath}`);
            console.error(`  Expected at: ${fullPath}`);
            console.error(`  Referenced from: ${basePath}`);
            process.exit(1);
        }

        // Read the include file
        const includeContent = fs.readFileSync(fullPath, 'utf8');
        const includeDir = path.dirname(fullPath);

        // Track visited paths for this branch
        const newVisited = new Set(visited);
        newVisited.add(normalizedPath);

        // Recursively resolve nested includes
        return resolveIncludes(includeContent, includeDir, depth + 1, newVisited);
    });
}

/**
 * Count partial files in a directory recursively
 * @param {string} dir - Directory to count
 * @returns {number} Number of HTML files
 */
function countPartials(dir) {
    let count = 0;
    const entries = fs.readdirSync(dir, { withFileTypes: true });

    for (const entry of entries) {
        const fullPath = path.join(dir, entry.name);
        if (entry.isDirectory()) {
            count += countPartials(fullPath);
        } else if (entry.isFile() && entry.name.endsWith('.html')) {
            count++;
        }
    }

    return count;
}

/**
 * Main composition function
 */
function compose() {
    const startTime = Date.now();

    console.log('Cartographus HTML Composition');
    console.log('=' .repeat(40));

    // Check for base template
    const baseTemplate = path.join(PARTIALS_DIR, '_base.html');
    if (!fs.existsSync(baseTemplate)) {
        console.error(`ERROR: Base template not found: ${baseTemplate}`);
        console.error('');
        console.error('The partials/_base.html file is the entry point for composition.');
        console.error('Make sure you have extracted the HTML structure into partials.');
        process.exit(1);
    }

    // Read and process base template
    console.log(`Reading: ${baseTemplate}`);
    const baseContent = fs.readFileSync(baseTemplate, 'utf8');
    const composedContent = resolveIncludes(baseContent);

    // Ensure output directory exists
    const outputDir = path.dirname(OUTPUT_FILE);
    if (!fs.existsSync(outputDir)) {
        fs.mkdirSync(outputDir, { recursive: true });
    }

    // Write composed output
    fs.writeFileSync(OUTPUT_FILE, composedContent);

    // Calculate metrics
    const duration = Date.now() - startTime;
    // Count lines like wc -l (count newlines, not segments)
    const lineCount = composedContent.endsWith('\n')
        ? composedContent.split('\n').length - 1
        : composedContent.split('\n').length;
    const partialCount = countPartials(PARTIALS_DIR);
    const sizeKB = (Buffer.byteLength(composedContent, 'utf8') / 1024).toFixed(1);

    console.log('');
    console.log('Composition complete:');
    console.log(`  Output: ${OUTPUT_FILE}`);
    console.log(`  Lines: ${lineCount}`);
    console.log(`  Size: ${sizeKB} KB`);
    console.log(`  Partials: ${partialCount} files`);
    console.log(`  Time: ${duration}ms`);

    return { lineCount, composedContent };
}

/**
 * Verify composed output against expected metrics
 */
function verify(lineCount) {
    console.log('');
    console.log('Verification:');

    if (lineCount === EXPECTED_LINE_COUNT) {
        console.log(`  ✓ Line count matches expected (${EXPECTED_LINE_COUNT})`);
        return true;
    } else {
        console.error(`  ✗ Line count mismatch!`);
        console.error(`    Expected: ${EXPECTED_LINE_COUNT}`);
        console.error(`    Actual: ${lineCount}`);
        console.error(`    Difference: ${lineCount - EXPECTED_LINE_COUNT}`);
        return false;
    }
}

/**
 * Watch mode - recompose on file changes
 */
function watchMode() {
    console.log('');
    console.log('Watch mode enabled. Press Ctrl+C to stop.');
    console.log(`Watching: ${PARTIALS_DIR}`);
    console.log('');

    // Initial compose
    compose();

    // Set up file watcher
    const debounceMs = 100;
    let debounceTimer = null;

    fs.watch(PARTIALS_DIR, { recursive: true }, (eventType, filename) => {
        if (filename && filename.endsWith('.html')) {
            // Debounce rapid changes
            clearTimeout(debounceTimer);
            debounceTimer = setTimeout(() => {
                console.log(`\nChange detected: ${filename}`);
                try {
                    compose();
                } catch (error) {
                    console.error(`Composition failed: ${error.message}`);
                }
            }, debounceMs);
        }
    });
}

// Parse command line arguments
const args = process.argv.slice(2);
const shouldWatch = args.includes('--watch');
const shouldVerify = args.includes('--verify');

// Run
if (shouldWatch) {
    watchMode();
} else {
    const { lineCount } = compose();

    if (shouldVerify) {
        const verified = verify(lineCount);
        if (!verified) {
            process.exit(1);
        }
    }
}
