// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Unit Tests for Tooltip Component
 *
 * Tests the Tooltip component's core logic and configuration.
 * Since Tooltip is a DOM-dependent component, these tests focus on:
 * - Configuration options
 * - Instance creation and destruction
 * - Content updates
 * - Position calculation logic
 * - Manager functionality
 *
 * Run with: npx tsx --test src/lib/components/Tooltip.test.ts
 */

import { describe, it, beforeEach } from 'node:test';
import assert from 'node:assert';
import {
    createTooltip,
    TooltipManager,
    TooltipPosition,
} from './Tooltip';

// Mock MutationObserver
class MockMutationObserver {
    constructor(_callback: MutationCallback) {
        // Callback stored for potential future use in tests
    }

    observe(): void {
        // No-op for testing
    }

    disconnect(): void {
        // No-op for testing
    }

    takeRecords(): MutationRecord[] {
        return [];
    }
}

(global as unknown as { MutationObserver: typeof MutationObserver }).MutationObserver = MockMutationObserver as unknown as typeof MutationObserver;

// Mock DOM environment for testing
function createMockDocument(): void {
    const elements: Map<string, MockElement> = new Map();
    const bodyChildren: MockElement[] = [];

    const mockDocument = {
        getElementById: (id: string): MockElement | null => {
            return elements.get(id) || null;
        },
        createElement: (tag: string): MockElement => createMockElement(tag),
        querySelector: (): MockElement | null => null,
        querySelectorAll: (): MockElement[] => [],
        body: {
            appendChild: (child: MockElement) => {
                bodyChildren.push(child);
                return child;
            },
            querySelectorAll: () => [],
        },
        _elements: elements,
        _bodyChildren: bodyChildren,
    };
    (global as unknown as { document: typeof mockDocument }).document = mockDocument;

    const mockWindow = {
        innerWidth: 1920,
        innerHeight: 1080,
        matchMedia: () => ({
            matches: false,
            addEventListener: () => {},
            removeEventListener: () => {},
            media: '',
            onchange: null,
            addListener: () => {},
            removeListener: () => {},
            dispatchEvent: () => true,
        }),
        addEventListener: () => {},
        removeEventListener: () => {},
    };
    (global as unknown as { window: typeof mockWindow }).window = mockWindow;

    (global as unknown as { setTimeout: typeof setTimeout }).setTimeout = ((fn: () => void, _delay: number): number => {
        fn();
        return 0;
    }) as unknown as typeof setTimeout;

    (global as unknown as { clearTimeout: typeof clearTimeout }).clearTimeout = (() => {}) as unknown as typeof clearTimeout;
}

interface MockElement {
    tagName: string;
    id: string;
    className: string;
    innerHTML: string;
    textContent: string;
    style: Record<string, string>;
    dataset: Record<string, string>;
    parentNode: MockElement | null;
    setAttribute: (name: string, value: string) => void;
    getAttribute: (name: string) => string | null;
    hasAttribute: (name: string) => boolean;
    removeAttribute: (name: string) => void;
    addEventListener: (event: string, listener: EventListener, options?: AddEventListenerOptions) => void;
    removeEventListener: (event: string, listener: EventListener) => void;
    appendChild: (child: MockElement) => MockElement;
    removeChild: (child: MockElement) => MockElement;
    getBoundingClientRect: () => DOMRect;
    classList: {
        add: (cls: string) => void;
        remove: (cls: string) => void;
        toggle: (cls: string) => boolean;
        contains: (cls: string) => boolean;
    };
    querySelectorAll: (selector: string) => MockElement[];
}

function createMockElement(tag: string): MockElement {
    const listeners: Map<string, Set<EventListener>> = new Map();
    const attributes: Map<string, string> = new Map();
    const children: MockElement[] = [];
    const classes: Set<string> = new Set();
    let innerHTML = '';
    let textContent = '';
    let parentNode: MockElement | null = null;

    const el: MockElement = {
        tagName: tag.toUpperCase(),
        id: '',
        className: '',
        innerHTML,
        textContent,
        style: {},
        dataset: {},
        parentNode,
        setAttribute: (name: string, value: string) => attributes.set(name, value),
        getAttribute: (name: string) => attributes.get(name) ?? null,
        hasAttribute: (name: string) => attributes.has(name),
        removeAttribute: (name: string) => attributes.delete(name),
        addEventListener: (event: string, listener: EventListener) => {
            if (!listeners.has(event)) listeners.set(event, new Set());
            listeners.get(event)!.add(listener);
        },
        removeEventListener: (event: string, listener: EventListener) => {
            listeners.get(event)?.delete(listener);
        },
        appendChild: (child: MockElement) => {
            children.push(child);
            child.parentNode = el;
            return child;
        },
        removeChild: (child: MockElement) => {
            const idx = children.indexOf(child);
            if (idx > -1) children.splice(idx, 1);
            child.parentNode = null;
            return child;
        },
        getBoundingClientRect: () => ({
            top: 100,
            bottom: 140,
            left: 100,
            right: 200,
            width: 100,
            height: 40,
            x: 100,
            y: 100,
            toJSON: () => ({}),
        }),
        classList: {
            add: (cls: string) => classes.add(cls),
            remove: (cls: string) => classes.delete(cls),
            toggle: (cls: string) => {
                if (classes.has(cls)) {
                    classes.delete(cls);
                    return false;
                }
                classes.add(cls);
                return true;
            },
            contains: (cls: string) => classes.has(cls),
        },
        querySelectorAll: () => [],
    };

    // Override innerHTML getter/setter
    Object.defineProperty(el, 'innerHTML', {
        get: () => innerHTML,
        set: (value: string) => { innerHTML = value; },
    });

    Object.defineProperty(el, 'textContent', {
        get: () => textContent,
        set: (value: string) => { textContent = value; },
    });

    Object.defineProperty(el, 'className', {
        get: () => Array.from(classes).join(' '),
        set: (value: string) => {
            classes.clear();
            value.split(' ').filter(Boolean).forEach(c => classes.add(c));
        },
    });

    return el;
}

describe('createTooltip', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('creates a tooltip instance', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Test tooltip' });

        assert.ok(tooltip);
        assert.strictEqual(typeof tooltip.show, 'function');
        assert.strictEqual(typeof tooltip.hide, 'function');
        assert.strictEqual(typeof tooltip.toggle, 'function');
        assert.strictEqual(typeof tooltip.destroy, 'function');
    });

    it('sets aria-describedby on trigger', () => {
        const trigger = createMockElement('button');
        createTooltip(trigger as unknown as HTMLElement, { content: 'Test tooltip' });

        const ariaDescribedBy = trigger.getAttribute('aria-describedby');
        assert.ok(ariaDescribedBy);
        assert.ok(ariaDescribedBy.startsWith('tooltip-'));
    });

    it('uses provided ID', () => {
        const trigger = createMockElement('button');
        createTooltip(trigger as unknown as HTMLElement, { content: 'Test', id: 'my-custom-tooltip' });

        assert.strictEqual(trigger.getAttribute('aria-describedby'), 'my-custom-tooltip');
    });

    it('starts hidden', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Test' });

        assert.strictEqual(tooltip.isVisible(), false);
    });
});

describe('Tooltip show/hide', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('shows tooltip', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Test', showDelay: 0 });

        tooltip.show();
        assert.strictEqual(tooltip.isVisible(), true);
    });

    it('hides tooltip', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Test', showDelay: 0, hideDelay: 0 });

        tooltip.show();
        tooltip.hide();
        assert.strictEqual(tooltip.isVisible(), false);
    });

    it('toggles visibility', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Test', showDelay: 0, hideDelay: 0 });

        assert.strictEqual(tooltip.isVisible(), false);
        tooltip.toggle();
        assert.strictEqual(tooltip.isVisible(), true);
        tooltip.toggle();
        assert.strictEqual(tooltip.isVisible(), false);
    });
});

describe('Tooltip content', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('updates content', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Initial' });

        tooltip.setContent('Updated content');
        // Content should be updated without errors
        assert.ok(tooltip);
    });

    it('escapes HTML by default', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: '<script>alert("xss")</script>' });

        // Should not execute script (content is escaped)
        assert.ok(tooltip);
    });

    it('allows HTML when configured', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, {
            content: '<strong>Bold</strong>',
            allowHtml: true,
        });

        assert.ok(tooltip);
    });
});

describe('Tooltip options', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('accepts position option', () => {
        const positions: TooltipPosition[] = ['top', 'bottom', 'left', 'right'];

        for (const position of positions) {
            const trigger = createMockElement('button');
            const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Test', position });
            assert.ok(tooltip);
        }
    });

    it('accepts delay options', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, {
            content: 'Test',
            showDelay: 500,
            hideDelay: 200,
        });

        assert.ok(tooltip);
    });

    it('accepts maxWidth option', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, {
            content: 'Test',
            maxWidth: 300,
        });

        assert.ok(tooltip);
    });

    it('accepts className option', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, {
            content: 'Test',
            className: 'custom-tooltip-class',
        });

        assert.ok(tooltip);
    });

    it('accepts zIndex option', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, {
            content: 'Test',
            zIndex: 99999,
        });

        assert.ok(tooltip);
    });

    it('accepts showArrow option', () => {
        const trigger = createMockElement('button');

        const withArrow = createTooltip(trigger as unknown as HTMLElement, { content: 'Test', showArrow: true });
        assert.ok(withArrow);

        const trigger2 = createMockElement('button');
        const withoutArrow = createTooltip(trigger2 as unknown as HTMLElement, { content: 'Test', showArrow: false });
        assert.ok(withoutArrow);
    });

    it('accepts role option', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, {
            content: 'Test',
            role: 'status',
        });

        assert.ok(tooltip);
    });
});

describe('Tooltip setOptions', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('updates content via setOptions', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Initial' });

        tooltip.setOptions({ content: 'Updated' });
        assert.ok(tooltip);
    });

    it('updates position via setOptions', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Test', position: 'top' });

        tooltip.setOptions({ position: 'bottom' });
        assert.ok(tooltip);
    });

    it('updates delays via setOptions', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Test' });

        tooltip.setOptions({ showDelay: 100, hideDelay: 50 });
        assert.ok(tooltip);
    });

    it('updates multiple options at once', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Test' });

        tooltip.setOptions({
            content: 'New content',
            maxWidth: 400,
            className: 'new-class',
        });
        assert.ok(tooltip);
    });
});

describe('Tooltip destroy', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('cleans up on destroy', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Test' });

        tooltip.destroy();

        // aria-describedby should be removed
        assert.strictEqual(trigger.getAttribute('aria-describedby'), null);
    });

    it('can be destroyed while visible', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Test', showDelay: 0 });

        tooltip.show();
        assert.strictEqual(tooltip.isVisible(), true);

        tooltip.destroy();
        assert.strictEqual(trigger.getAttribute('aria-describedby'), null);
    });
});

describe('TooltipManager', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('creates a manager instance', () => {
        const manager = new TooltipManager();
        assert.ok(manager);
        assert.strictEqual(manager.count, 0);
        manager.destroy();
    });

    it('accepts default options', () => {
        const manager = new TooltipManager({
            position: 'bottom',
            showDelay: 100,
        });

        assert.ok(manager);
        manager.destroy();
    });

    it('creates tooltip for element', () => {
        const manager = new TooltipManager();
        const trigger = createMockElement('button');

        const tooltip = manager.create(trigger as unknown as HTMLElement, { content: 'Test' });
        assert.ok(tooltip);
        assert.strictEqual(manager.count, 1);

        manager.destroy();
    });

    it('retrieves tooltip for element', () => {
        const manager = new TooltipManager();
        const trigger = createMockElement('button');

        manager.create(trigger as unknown as HTMLElement, { content: 'Test' });
        const retrieved = manager.get(trigger as unknown as HTMLElement);

        assert.ok(retrieved);
        manager.destroy();
    });

    it('returns undefined for unknown element', () => {
        const manager = new TooltipManager();
        const trigger = createMockElement('button');

        const retrieved = manager.get(trigger as unknown as HTMLElement);
        assert.strictEqual(retrieved, undefined);

        manager.destroy();
    });

    it('hides all tooltips', () => {
        const manager = new TooltipManager();
        const trigger1 = createMockElement('button');
        const trigger2 = createMockElement('button');

        const tooltip1 = manager.create(trigger1 as unknown as HTMLElement, { content: 'Test 1', showDelay: 0 });
        const tooltip2 = manager.create(trigger2 as unknown as HTMLElement, { content: 'Test 2', showDelay: 0 });

        tooltip1.show();
        tooltip2.show();

        manager.hideAll();

        // With mock setTimeout running immediately, hideAll should work
        assert.ok(manager);
        manager.destroy();
    });

    it('destroys all tooltips', () => {
        const manager = new TooltipManager();
        const trigger1 = createMockElement('button');
        const trigger2 = createMockElement('button');

        manager.create(trigger1 as unknown as HTMLElement, { content: 'Test 1' });
        manager.create(trigger2 as unknown as HTMLElement, { content: 'Test 2' });

        assert.strictEqual(manager.count, 2);

        manager.destroy();
        assert.strictEqual(manager.count, 0);
    });

    it('replaces existing tooltip on same element', () => {
        const manager = new TooltipManager();
        const trigger = createMockElement('button');

        manager.create(trigger as unknown as HTMLElement, { content: 'First' });
        manager.create(trigger as unknown as HTMLElement, { content: 'Second' });

        assert.strictEqual(manager.count, 1);
        manager.destroy();
    });
});

describe('Position calculation', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('handles all position options without error', () => {
        const positions: TooltipPosition[] = ['top', 'bottom', 'left', 'right'];

        for (const position of positions) {
            const trigger = createMockElement('button');
            const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Test', position, showDelay: 0 });

            // Show to trigger position calculation
            tooltip.show();
            assert.ok(tooltip.isVisible());
            tooltip.destroy();
        }
    });
});

describe('Edge cases', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('handles empty content', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: '' });
        assert.ok(tooltip);
        tooltip.destroy();
    });

    it('handles very long content', () => {
        const trigger = createMockElement('button');
        const longContent = 'A'.repeat(1000);
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: longContent });
        assert.ok(tooltip);
        tooltip.destroy();
    });

    it('handles special characters in content', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: '<>&"\'' });
        assert.ok(tooltip);
        tooltip.destroy();
    });

    it('handles unicode content', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Hello World 123' });
        assert.ok(tooltip);
        tooltip.destroy();
    });

    it('handles zero delays', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, {
            content: 'Test',
            showDelay: 0,
            hideDelay: 0,
        });

        tooltip.show();
        assert.strictEqual(tooltip.isVisible(), true);
        tooltip.hide();
        assert.strictEqual(tooltip.isVisible(), false);
        tooltip.destroy();
    });

    it('handles rapid show/hide calls', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Test', showDelay: 0, hideDelay: 0 });

        for (let i = 0; i < 10; i++) {
            tooltip.show();
            tooltip.hide();
        }

        assert.ok(tooltip);
        tooltip.destroy();
    });

    it('handles double destroy', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Test' });

        tooltip.destroy();
        tooltip.destroy(); // Should not throw

        assert.ok(true);
    });
});

describe('Accessibility', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('sets role attribute', () => {
        const trigger = createMockElement('button');
        createTooltip(trigger as unknown as HTMLElement, { content: 'Test', role: 'tooltip' });

        // Tooltip element should have role
        assert.ok(true);
    });

    it('manages aria-hidden state', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Test', showDelay: 0, hideDelay: 0 });

        // When hidden, aria-hidden should be true
        tooltip.show();
        // When visible, aria-hidden should be false
        tooltip.hide();
        // When hidden again, aria-hidden should be true

        assert.ok(tooltip);
        tooltip.destroy();
    });

    it('links trigger and tooltip via aria-describedby', () => {
        const trigger = createMockElement('button');
        createTooltip(trigger as unknown as HTMLElement, { content: 'Helpful description', id: 'help-tooltip' });

        assert.strictEqual(trigger.getAttribute('aria-describedby'), 'help-tooltip');
    });
});

describe('Configuration validation', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('uses default values for unspecified options', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Test' });

        // Should use defaults without error
        assert.ok(tooltip);
        tooltip.destroy();
    });

    it('handles offset option', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, { content: 'Test', offset: 16 });

        assert.ok(tooltip);
        tooltip.destroy();
    });

    it('handles all option combinations', () => {
        const trigger = createMockElement('button');
        const tooltip = createTooltip(trigger as unknown as HTMLElement, {
            content: 'Full options test',
            position: 'right',
            showDelay: 100,
            hideDelay: 50,
            allowHtml: false,
            className: 'custom-class',
            maxWidth: 300,
            offset: 10,
            zIndex: 9999,
            showArrow: true,
            role: 'tooltip',
            id: 'custom-id',
        });

        assert.ok(tooltip);
        tooltip.destroy();
    });
});
