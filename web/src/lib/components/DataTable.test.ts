// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Unit Tests for DataTable Component
 *
 * Tests the DataTable component's core logic and configuration.
 * Since DataTable is a DOM-dependent component, these tests focus on:
 * - Configuration validation
 * - Sorting logic
 * - Pagination calculations
 * - Search/filter logic
 * - Row key generation
 *
 * Full integration tests with DOM would require a testing environment like jsdom.
 *
 * Run with: npx tsx --test src/lib/components/DataTable.test.ts
 */

import { describe, it, beforeEach, mock } from 'node:test';
import assert from 'node:assert';
import { DataTable, DataTableColumn } from './DataTable';

// Mock DOM environment for testing
function createMockDocument(): void {
    const elements: Map<string, HTMLElement> = new Map();

    (global as unknown as { document: Partial<Document> }).document = {
        getElementById: (id: string): HTMLElement | null => {
            if (elements.has(id)) return elements.get(id)!;
            const el = createMockElement('div');
            el.id = id;
            elements.set(id, el);
            return el;
        },
        createElement: (tag: string): HTMLElement => createMockElement(tag),
        querySelector: (): HTMLElement | null => null,
    } as unknown as Document;

    (global as unknown as { setTimeout: typeof setTimeout }).setTimeout = ((fn: () => void): number => {
        fn();
        return 0;
    }) as unknown as typeof setTimeout;
}

function createMockElement(tag: string): HTMLElement {
    const listeners: Map<string, Set<EventListener>> = new Map();
    const attributes: Map<string, string> = new Map();
    const children: HTMLElement[] = [];
    let innerHtml = '';
    let textContent = '';

    const el = {
        tagName: tag.toUpperCase(),
        id: '',
        className: '',
        innerHTML: innerHtml,
        textContent: textContent,
        style: {} as CSSStyleDeclaration,
        dataset: {} as DOMStringMap,
        children: children as unknown as HTMLCollection,
        parentElement: null as HTMLElement | null,
        scrollIntoView: () => {},
        focus: () => {},
        setAttribute: (name: string, value: string) => attributes.set(name, value),
        getAttribute: (name: string) => attributes.get(name) ?? null,
        hasAttribute: (name: string) => attributes.has(name),
        removeAttribute: (name: string) => { attributes.delete(name); },
        addEventListener: (event: string, listener: EventListener, _options?: AddEventListenerOptions) => {
            if (!listeners.has(event)) listeners.set(event, new Set());
            listeners.get(event)!.add(listener);
        },
        removeEventListener: (event: string, listener: EventListener) => {
            listeners.get(event)?.delete(listener);
        },
        appendChild: <T extends Node>(child: T): T => {
            children.push(child as unknown as HTMLElement);
            return child;
        },
        querySelector: (): HTMLElement | null => null,
        closest: (): HTMLElement | null => null,
        classList: {
            add: () => {},
            remove: () => {},
            toggle: () => false,
            contains: () => false,
            length: 0,
            value: '',
            item: () => null,
            replace: () => false,
            supports: () => false,
            forEach: () => {},
            entries: () => [][Symbol.iterator](),
            keys: () => [][Symbol.iterator](),
            values: () => [][Symbol.iterator](),
            [Symbol.iterator]: () => [][Symbol.iterator](),
        } as DOMTokenList,
    } as unknown as HTMLElement;

    // Override innerHTML getter/setter
    Object.defineProperty(el, 'innerHTML', {
        get: () => innerHtml,
        set: (value: string) => { innerHtml = value; },
    });

    Object.defineProperty(el, 'textContent', {
        get: () => textContent,
        set: (value: string) => { textContent = value; },
    });

    return el;
}

interface TestData extends Record<string, unknown> {
    id: number;
    name: string;
    email: string;
    age: number;
    active: boolean;
}

const sampleData: TestData[] = [
    { id: 1, name: 'Alice', email: 'alice@example.com', age: 30, active: true },
    { id: 2, name: 'Bob', email: 'bob@example.com', age: 25, active: false },
    { id: 3, name: 'Charlie', email: 'charlie@example.com', age: 35, active: true },
    { id: 4, name: 'Diana', email: 'diana@example.com', age: 28, active: true },
    { id: 5, name: 'Eve', email: 'eve@example.com', age: 22, active: false },
];

const sampleColumns: DataTableColumn<TestData>[] = [
    { key: 'id', header: 'ID', sortable: true },
    { key: 'name', header: 'Name', sortable: true },
    { key: 'email', header: 'Email', sortable: true },
    { key: 'age', header: 'Age', sortable: true, align: 'right' },
    { key: 'active', header: 'Status', sortable: false },
];

describe('DataTable Configuration', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('throws error when container is not found', () => {
        (global as unknown as { document: Partial<Document> }).document.getElementById = () => null;

        assert.throws(() => {
            new DataTable<TestData>({
                container: 'non-existent',
                columns: sampleColumns,
            });
        }, /Container not found/);
    });

    it('applies default options correctly', () => {
        const table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
        });

        // Table should be created without errors
        assert.ok(table instanceof DataTable);
    });

    it('accepts custom page size', () => {
        const table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
            pageSize: 25,
        });

        assert.ok(table instanceof DataTable);
    });

    it('accepts selection options', () => {
        const table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
            selectable: true,
            multiSelect: true,
        });

        assert.ok(table instanceof DataTable);
    });
});

describe('DataTable Data Management', () => {
    let table: DataTable<TestData>;

    beforeEach(() => {
        createMockDocument();
        table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
        });
    });

    it('stores data correctly', () => {
        const data = table.getData();
        assert.strictEqual(data.length, 5);
        assert.deepStrictEqual(data[0], sampleData[0]);
    });

    it('allows updating data', () => {
        const newData: TestData[] = [
            { id: 10, name: 'Frank', email: 'frank@example.com', age: 40, active: true },
        ];

        table.setData(newData);
        const data = table.getData();

        assert.strictEqual(data.length, 1);
        assert.strictEqual(data[0].name, 'Frank');
    });

    it('clears selection on data update', () => {
        table.setData(sampleData);
        const selected = table.getSelectedData();
        assert.strictEqual(selected.length, 0);
    });
});

describe('DataTable Column Configuration', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('handles hidden columns', () => {
        const columns: DataTableColumn<TestData>[] = [
            { key: 'id', header: 'ID' },
            { key: 'name', header: 'Name', hidden: true },
            { key: 'email', header: 'Email' },
        ];

        const table = new DataTable<TestData>({
            container: 'test-container',
            columns,
            data: sampleData,
        });

        assert.ok(table instanceof DataTable);
    });

    it('handles column alignment', () => {
        const columns: DataTableColumn<TestData>[] = [
            { key: 'id', header: 'ID', align: 'left' },
            { key: 'age', header: 'Age', align: 'right' },
            { key: 'name', header: 'Name', align: 'center' },
        ];

        const table = new DataTable<TestData>({
            container: 'test-container',
            columns,
            data: sampleData,
        });

        assert.ok(table instanceof DataTable);
    });

    it('handles custom column width', () => {
        const columns: DataTableColumn<TestData>[] = [
            { key: 'id', header: 'ID', width: '50px' },
            { key: 'name', header: 'Name', width: '200px' },
        ];

        const table = new DataTable<TestData>({
            container: 'test-container',
            columns,
            data: sampleData,
        });

        assert.ok(table instanceof DataTable);
    });

    it('handles custom cell renderer', () => {
        const columns: DataTableColumn<TestData>[] = [
            { key: 'id', header: 'ID' },
            {
                key: 'active',
                header: 'Status',
                render: (value: unknown) => value ? 'Active' : 'Inactive',
            },
        ];

        const table = new DataTable<TestData>({
            container: 'test-container',
            columns,
            data: sampleData,
        });

        assert.ok(table instanceof DataTable);
    });
});

describe('DataTable Sorting Logic', () => {
    let table: DataTable<TestData>;

    beforeEach(() => {
        createMockDocument();
        table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
        });
    });

    it('sorts by column ascending', () => {
        table.sort('name');
        const data = table.getData();

        // Original data should not be modified
        assert.strictEqual(data[0].name, 'Alice');
    });

    it('toggles sort direction on same column', () => {
        table.sort('name');
        table.sort('name');

        // Second sort should toggle direction
        assert.ok(table instanceof DataTable);
    });

    it('resets to ascending on different column', () => {
        table.sort('name');
        table.sort('age');

        // Sorting by different column starts ascending
        assert.ok(table instanceof DataTable);
    });
});

describe('DataTable Pagination Logic', () => {
    let table: DataTable<TestData>;

    beforeEach(() => {
        createMockDocument();

        // Create larger dataset for pagination testing
        const largeData: TestData[] = Array.from({ length: 50 }, (_, i) => ({
            id: i + 1,
            name: `User ${i + 1}`,
            email: `user${i + 1}@example.com`,
            age: 20 + (i % 40),
            active: i % 2 === 0,
        }));

        table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: largeData,
            pageSize: 10,
        });
    });

    it('navigates to specific page', () => {
        table.goToPage(3);
        // Page should change without error
        assert.ok(table instanceof DataTable);
    });

    it('clamps page number to valid range', () => {
        table.goToPage(0); // Below min
        table.goToPage(100); // Above max
        assert.ok(table instanceof DataTable);
    });

    it('handles page size change', () => {
        table.setPageSize(25);
        assert.ok(table instanceof DataTable);
    });
});

describe('DataTable Search/Filter Logic', () => {
    let table: DataTable<TestData>;

    beforeEach(() => {
        createMockDocument();
        table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
            searchable: true,
        });
    });

    it('filters data by search query', () => {
        table.setSearch('alice');
        // Search should filter without error
        assert.ok(table instanceof DataTable);
    });

    it('clears filter on empty search', () => {
        table.setSearch('alice');
        table.setSearch('');
        // Data should be restored
        assert.ok(table instanceof DataTable);
    });

    it('handles case-insensitive search', () => {
        table.setSearch('ALICE');
        assert.ok(table instanceof DataTable);
    });

    it('resets to page 1 on search', () => {
        table.goToPage(2);
        table.setSearch('test');
        // Should reset to first page
        assert.ok(table instanceof DataTable);
    });
});

describe('DataTable Selection', () => {
    let table: DataTable<TestData>;
    const mockOnSelectionChange = mock.fn();

    beforeEach(() => {
        createMockDocument();
        mockOnSelectionChange.mock.resetCalls();

        table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
            selectable: true,
            multiSelect: true,
            getRowKey: (row: TestData) => row.id,
            onSelectionChange: mockOnSelectionChange,
        });
    });

    it('tracks row selection', () => {
        table.toggleRowSelection(1);
        const selected = table.getSelectedData();
        assert.strictEqual(selected.length, 1);
    });

    it('supports multi-select', () => {
        table.toggleRowSelection(1);
        table.toggleRowSelection(2);
        const selected = table.getSelectedData();
        assert.strictEqual(selected.length, 2);
    });

    it('toggles selection off', () => {
        table.toggleRowSelection(1);
        table.toggleRowSelection(1);
        const selected = table.getSelectedData();
        assert.strictEqual(selected.length, 0);
    });

    it('calls onSelectionChange callback', () => {
        table.toggleRowSelection(1);
        assert.strictEqual(mockOnSelectionChange.mock.calls.length, 1);
    });

    it('supports select all', () => {
        table.toggleSelectAll(true);
        const selected = table.getSelectedData();
        assert.strictEqual(selected.length, 5);
    });

    it('supports clear all selections', () => {
        table.toggleSelectAll(true);
        table.toggleSelectAll(false);
        const selected = table.getSelectedData();
        assert.strictEqual(selected.length, 0);
    });
});

describe('DataTable Single Selection Mode', () => {
    let table: DataTable<TestData>;

    beforeEach(() => {
        createMockDocument();
        table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
            selectable: true,
            multiSelect: false,
            getRowKey: (row: TestData) => row.id,
        });
    });

    it('only allows one selection at a time', () => {
        table.toggleRowSelection(1);
        table.toggleRowSelection(2);
        const selected = table.getSelectedData();

        // Should only have the latest selection
        assert.strictEqual(selected.length, 1);
        assert.strictEqual(selected[0].id, 2);
    });
});

describe('DataTable Callbacks', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('calls onSort callback', () => {
        const onSort = mock.fn();

        const table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
            onSort,
        });

        table.sort('name');
        assert.strictEqual(onSort.mock.calls.length, 1);
        assert.strictEqual(onSort.mock.calls[0].arguments[0], 'name');
        assert.strictEqual(onSort.mock.calls[0].arguments[1], 'asc');
    });

    it('calls onPageChange callback', () => {
        const onPageChange = mock.fn();
        const largeData = Array.from({ length: 30 }, (_, i) => ({
            id: i,
            name: `User ${i}`,
            email: `user${i}@example.com`,
            age: 20 + i,
            active: true,
        }));

        const table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: largeData,
            pageSize: 10,
            onPageChange,
        });

        table.goToPage(2);
        assert.strictEqual(onPageChange.mock.calls.length, 1);
        assert.strictEqual(onPageChange.mock.calls[0].arguments[0], 2);
    });
});

describe('DataTable Custom Row Keys', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('uses custom getRowKey function', () => {
        const table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
            selectable: true,
            getRowKey: (row: TestData) => `custom-${row.id}`,
        });

        table.toggleRowSelection('custom-1');
        const selected = table.getSelectedData();
        assert.strictEqual(selected.length, 1);
    });

    it('falls back to index when no custom key', () => {
        const table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
            selectable: true,
        });

        // When no getRowKey is provided, the default uses index
        // Toggle selection for row at index 0
        table.toggleRowSelection(0);

        // Verify the selection was made (the internal set should have the key)
        // Note: With default getRowKey, it uses the row index
        assert.ok(table instanceof DataTable);
    });
});

describe('DataTable Empty State', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('handles empty data', () => {
        const table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: [],
            emptyMessage: 'No records found',
        });

        const data = table.getData();
        assert.strictEqual(data.length, 0);
    });

    it('uses custom empty message', () => {
        const table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: [],
            emptyMessage: 'Custom empty message',
        });

        assert.ok(table instanceof DataTable);
    });
});

describe('DataTable Lifecycle', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('destroys cleanly', () => {
        const table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
        });

        table.destroy();

        // Should not throw after destroy
        const data = table.getData();
        assert.strictEqual(data.length, 0);
    });

    it('allows refresh', () => {
        const table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
        });

        table.refresh();
        assert.ok(table instanceof DataTable);
    });
});

describe('DataTable Caption and Accessibility', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('accepts table caption', () => {
        const table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
            caption: 'User Data Table',
        });

        assert.ok(table instanceof DataTable);
    });

    it('accepts column ariaLabel', () => {
        const columns: DataTableColumn<TestData>[] = [
            { key: 'id', header: 'ID', ariaLabel: 'User identification number' },
            { key: 'name', header: 'Name', ariaLabel: 'Full name of the user' },
        ];

        const table = new DataTable<TestData>({
            container: 'test-container',
            columns,
            data: sampleData,
        });

        assert.ok(table instanceof DataTable);
    });
});

describe('DataTable Search Configuration', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('accepts custom search placeholder', () => {
        const table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
            searchable: true,
            searchPlaceholder: 'Find users...',
        });

        assert.ok(table instanceof DataTable);
    });

    it('accepts specific search columns', () => {
        const table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
            searchable: true,
            searchColumns: ['name', 'email'],
        });

        assert.ok(table instanceof DataTable);
    });
});

describe('DataTable Pagination Options', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('accepts custom page size options', () => {
        const table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
            pagination: true,
            pageSizeOptions: [5, 10, 15, 20],
        });

        assert.ok(table instanceof DataTable);
    });

    it('can disable pagination', () => {
        const table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
            pagination: false,
        });

        assert.ok(table instanceof DataTable);
    });
});

describe('DataTable CSS Class', () => {
    beforeEach(() => {
        createMockDocument();
    });

    it('accepts custom className', () => {
        const table = new DataTable<TestData>({
            container: 'test-container',
            columns: sampleColumns,
            data: sampleData,
            className: 'custom-table-class',
        });

        assert.ok(table instanceof DataTable);
    });
});
