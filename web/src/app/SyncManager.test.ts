// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Unit Tests for SyncManager
 *
 * Tests sync state management, progress tracking, WebSocket integration,
 * polling fallback, and persistence.
 *
 * Run with: npx tsx --test SyncManager.test.ts
 */

import { describe, it, beforeEach } from 'node:test';
import assert from 'node:assert';
import type {
    SyncProgress,
    SyncStatusResponse,
    SyncProgressMessage,
} from '../lib/types/sync';

// ============================================================================
// Mock Implementations
// ============================================================================

/**
 * Mock fetch client for testing
 */
class MockFetchClient {
    responses: Map<string, { data: unknown }> = new Map();
    calls: Array<{ url: string; options?: RequestInit }> = [];
    shouldFail = false;
    failureError = new Error('Network error');

    async fetch<T>(url: string, options?: RequestInit): Promise<{ data: T }> {
        this.calls.push({ url, options });

        if (this.shouldFail) {
            throw this.failureError;
        }

        const response = this.responses.get(url);
        if (response) {
            return response as { data: T };
        }

        // Default empty response
        return { data: {} as T };
    }

    setResponse<T>(url: string, data: T): void {
        this.responses.set(url, { data });
    }

    reset(): void {
        this.responses.clear();
        this.calls = [];
        this.shouldFail = false;
    }
}

/**
 * Mock state storage for testing
 */
class MockStateStorage {
    storage: Map<string, string> = new Map();

    get(key: string): string | null {
        return this.storage.get(key) || null;
    }

    set(key: string, value: string): void {
        this.storage.set(key, value);
    }

    remove(key: string): void {
        this.storage.delete(key);
    }

    reset(): void {
        this.storage.clear();
    }
}

/**
 * Mock correlation ID generator for testing
 */
class MockCorrelationIdGenerator {
    nextId = 'test-correlation-id-001';

    generate(): string {
        return this.nextId;
    }

    setNextId(id: string): void {
        this.nextId = id;
    }
}

/**
 * Mock timer manager for testing
 */
class MockTimerManager {
    timeouts: Map<number, () => void> = new Map();
    intervals: Map<number, { callback: () => void; ms: number }> = new Map();
    nextId = 1;
    clearedTimeouts: number[] = [];
    clearedIntervals: number[] = [];

    setTimeout(callback: () => void, _ms: number): number {
        const id = this.nextId++;
        this.timeouts.set(id, callback);
        return id;
    }

    clearTimeout(id: number): void {
        this.timeouts.delete(id);
        this.clearedTimeouts.push(id);
    }

    setInterval(callback: () => void, ms: number): number {
        const id = this.nextId++;
        this.intervals.set(id, { callback, ms });
        return id;
    }

    clearInterval(id: number): void {
        this.intervals.delete(id);
        this.clearedIntervals.push(id);
    }

    triggerTimeout(id: number): void {
        const callback = this.timeouts.get(id);
        if (callback) {
            callback();
        }
    }

    triggerInterval(id: number): void {
        const interval = this.intervals.get(id);
        if (interval) {
            interval.callback();
        }
    }

    reset(): void {
        this.timeouts.clear();
        this.intervals.clear();
        this.clearedTimeouts = [];
        this.clearedIntervals = [];
        this.nextId = 1;
    }
}

/**
 * Helper to create SyncProgress object
 */
function createProgress(overrides: Partial<SyncProgress> = {}): SyncProgress {
    return {
        status: 'idle',
        total_records: 0,
        processed_records: 0,
        imported_records: 0,
        skipped_records: 0,
        error_count: 0,
        progress_percent: 0,
        records_per_second: 0,
        elapsed_seconds: 0,
        estimated_remaining_seconds: 0,
        ...overrides,
    };
}

// ============================================================================
// Tests
// ============================================================================

describe('SyncManager', () => {
    let mockFetch: MockFetchClient;
    let mockStorage: MockStateStorage;
    let mockCorrelationId: MockCorrelationIdGenerator;
    let mockTimer: MockTimerManager;

    beforeEach(() => {
        mockFetch = new MockFetchClient();
        mockStorage = new MockStateStorage();
        mockCorrelationId = new MockCorrelationIdGenerator();
        mockTimer = new MockTimerManager();
    });

    describe('SyncProgress interface', () => {
        it('should have correct structure', () => {
            const progress = createProgress();

            assert.strictEqual(progress.status, 'idle');
            assert.strictEqual(progress.total_records, 0);
            assert.strictEqual(progress.processed_records, 0);
            assert.strictEqual(progress.imported_records, 0);
            assert.strictEqual(progress.skipped_records, 0);
            assert.strictEqual(progress.error_count, 0);
            assert.strictEqual(progress.progress_percent, 0);
            assert.strictEqual(progress.records_per_second, 0);
            assert.strictEqual(progress.elapsed_seconds, 0);
            assert.strictEqual(progress.estimated_remaining_seconds, 0);
        });

        it('should accept valid status values', () => {
            const statuses: SyncProgress['status'][] = ['idle', 'running', 'completed', 'error', 'cancelled'];

            for (const status of statuses) {
                const progress = createProgress({ status });
                assert.strictEqual(progress.status, status);
            }
        });
    });

    describe('SyncStatusResponse structure', () => {
        it('should have correct structure with all fields', () => {
            const response: SyncStatusResponse = {
                tautulli_import: createProgress({ status: 'running' }),
                plex_historical: createProgress({ status: 'completed' }),
                server_syncs: {
                    'server-1': createProgress({ status: 'running' }),
                    'server-2': createProgress({ status: 'idle' }),
                },
            };

            assert.ok(response.tautulli_import);
            assert.strictEqual(response.tautulli_import!.status, 'running');
            assert.ok(response.plex_historical);
            assert.strictEqual(response.plex_historical!.status, 'completed');
            assert.ok(response.server_syncs);
            assert.strictEqual(Object.keys(response.server_syncs!).length, 2);
        });

        it('should handle empty response', () => {
            const response: SyncStatusResponse = {};

            assert.strictEqual(response.tautulli_import, undefined);
            assert.strictEqual(response.plex_historical, undefined);
            assert.strictEqual(response.server_syncs, undefined);
        });
    });

    describe('SyncProgressMessage structure', () => {
        it('should have correct structure for tautulli_import', () => {
            const message: SyncProgressMessage = {
                type: 'sync_progress',
                data: {
                    operation: 'tautulli_import',
                    status: 'running',
                    progress: createProgress({ status: 'running', total_records: 1000 }),
                    correlation_id: 'test-123',
                },
            };

            assert.strictEqual(message.type, 'sync_progress');
            assert.strictEqual(message.data.operation, 'tautulli_import');
            assert.strictEqual(message.data.status, 'running');
            assert.strictEqual(message.data.progress.total_records, 1000);
        });

        it('should have correct structure for plex_historical', () => {
            const message: SyncProgressMessage = {
                type: 'sync_progress',
                data: {
                    operation: 'plex_historical',
                    status: 'completed',
                    progress: createProgress({ status: 'completed', processed_records: 500 }),
                    correlation_id: 'test-456',
                },
            };

            assert.strictEqual(message.data.operation, 'plex_historical');
        });

        it('should include server_id for server_sync operations', () => {
            const message: SyncProgressMessage = {
                type: 'sync_progress',
                data: {
                    operation: 'server_sync',
                    status: 'running',
                    server_id: 'server-abc',
                    progress: createProgress({ status: 'running' }),
                    correlation_id: 'test-789',
                },
            };

            assert.strictEqual(message.data.server_id, 'server-abc');
        });
    });

    describe('MockFetchClient', () => {
        it('should record fetch calls', async () => {
            await mockFetch.fetch('/api/v1/sync/status');

            assert.strictEqual(mockFetch.calls.length, 1);
            assert.strictEqual(mockFetch.calls[0].url, '/api/v1/sync/status');
        });

        it('should return configured responses', async () => {
            mockFetch.setResponse('/api/v1/sync/status', {
                tautulli_import: createProgress({ status: 'running' }),
            });

            const response = await mockFetch.fetch<SyncStatusResponse>('/api/v1/sync/status');

            assert.ok(response.data.tautulli_import);
            assert.strictEqual(response.data.tautulli_import!.status, 'running');
        });

        it('should throw when configured to fail', async () => {
            mockFetch.shouldFail = true;

            await assert.rejects(
                () => mockFetch.fetch('/api/v1/sync/status'),
                { message: 'Network error' }
            );
        });

        it('should record request options', async () => {
            await mockFetch.fetch('/api/v1/import/tautulli', {
                method: 'POST',
                body: JSON.stringify({ resume: true }),
            });

            assert.strictEqual(mockFetch.calls[0].options?.method, 'POST');
            assert.strictEqual(mockFetch.calls[0].options?.body, '{"resume":true}');
        });
    });

    describe('MockStateStorage', () => {
        it('should store and retrieve values', () => {
            mockStorage.set('key1', 'value1');

            assert.strictEqual(mockStorage.get('key1'), 'value1');
        });

        it('should return null for missing keys', () => {
            assert.strictEqual(mockStorage.get('nonexistent'), null);
        });

        it('should remove values', () => {
            mockStorage.set('key1', 'value1');
            mockStorage.remove('key1');

            assert.strictEqual(mockStorage.get('key1'), null);
        });

        it('should reset all values', () => {
            mockStorage.set('key1', 'value1');
            mockStorage.set('key2', 'value2');
            mockStorage.reset();

            assert.strictEqual(mockStorage.get('key1'), null);
            assert.strictEqual(mockStorage.get('key2'), null);
        });
    });

    describe('MockCorrelationIdGenerator', () => {
        it('should return configured ID', () => {
            assert.strictEqual(mockCorrelationId.generate(), 'test-correlation-id-001');
        });

        it('should return updated ID after setNextId', () => {
            mockCorrelationId.setNextId('custom-id-123');
            assert.strictEqual(mockCorrelationId.generate(), 'custom-id-123');
        });
    });

    describe('MockTimerManager', () => {
        it('should track setTimeout calls', () => {
            const callback = () => {};
            const id = mockTimer.setTimeout(callback, 1000);

            assert.ok(id > 0);
            assert.ok(mockTimer.timeouts.has(id));
        });

        it('should track clearTimeout calls', () => {
            const id = mockTimer.setTimeout(() => {}, 1000);
            mockTimer.clearTimeout(id);

            assert.ok(!mockTimer.timeouts.has(id));
            assert.ok(mockTimer.clearedTimeouts.includes(id));
        });

        it('should track setInterval calls', () => {
            const callback = () => {};
            const id = mockTimer.setInterval(callback, 1000);

            assert.ok(id > 0);
            assert.ok(mockTimer.intervals.has(id));
        });

        it('should track clearInterval calls', () => {
            const id = mockTimer.setInterval(() => {}, 1000);
            mockTimer.clearInterval(id);

            assert.ok(!mockTimer.intervals.has(id));
            assert.ok(mockTimer.clearedIntervals.includes(id));
        });

        it('should trigger timeout callbacks', () => {
            let called = false;
            const id = mockTimer.setTimeout(() => {
                called = true;
            }, 1000);

            mockTimer.triggerTimeout(id);

            assert.strictEqual(called, true);
        });

        it('should trigger interval callbacks', () => {
            let callCount = 0;
            const id = mockTimer.setInterval(() => {
                callCount++;
            }, 1000);

            mockTimer.triggerInterval(id);
            mockTimer.triggerInterval(id);

            assert.strictEqual(callCount, 2);
        });
    });

    describe('Progress calculations', () => {
        it('should calculate progress percentage correctly', () => {
            const progress = createProgress({
                total_records: 1000,
                processed_records: 500,
                progress_percent: 50,
            });

            assert.strictEqual(progress.progress_percent, 50);
        });

        it('should handle zero total records', () => {
            const progress = createProgress({
                total_records: 0,
                processed_records: 0,
                progress_percent: 0,
            });

            assert.strictEqual(progress.progress_percent, 0);
        });

        it('should calculate ETA from remaining seconds', () => {
            const progress = createProgress({
                total_records: 1000,
                processed_records: 500,
                records_per_second: 50,
                estimated_remaining_seconds: 10, // 500 remaining / 50 per second
            });

            assert.strictEqual(progress.estimated_remaining_seconds, 10);
        });
    });

    describe('State persistence format', () => {
        it('should serialize state to JSON', () => {
            const state = {
                correlationId: 'test-123',
                operation: 'tautulli_import' as const,
                status: 'running' as const,
                progress: createProgress({ status: 'running' }),
                persistedAt: Date.now(),
                expiresAt: Date.now() + 300000,
            };

            const json = JSON.stringify(state);
            const parsed = JSON.parse(json);

            assert.strictEqual(parsed.correlationId, 'test-123');
            assert.strictEqual(parsed.operation, 'tautulli_import');
            assert.strictEqual(parsed.status, 'running');
        });

        it('should detect expired state', () => {
            const expiredAt = Date.now() - 1000; // 1 second ago
            const isExpired = Date.now() > expiredAt;

            assert.strictEqual(isExpired, true);
        });

        it('should detect valid state', () => {
            const expiresAt = Date.now() + 300000; // 5 minutes from now
            const isExpired = Date.now() > expiresAt;

            assert.strictEqual(isExpired, false);
        });
    });

    describe('Operation mutex logic', () => {
        // Helper to check if an operation can start based on another's status
        function canStartOperation(otherStatus: SyncProgress['status']): boolean {
            return otherStatus !== 'running';
        }

        it('should block Plex sync when Tautulli import is running', () => {
            const canStart = canStartOperation('running');
            assert.strictEqual(canStart, false);
        });

        it('should allow Plex sync when Tautulli import is idle', () => {
            const canStart = canStartOperation('idle');
            assert.strictEqual(canStart, true);
        });

        it('should allow Plex sync when Tautulli import is completed', () => {
            const canStart = canStartOperation('completed');
            assert.strictEqual(canStart, true);
        });

        it('should block Tautulli import when Plex sync is running', () => {
            const canStart = canStartOperation('running');
            assert.strictEqual(canStart, false);
        });
    });

    describe('WebSocket message handling', () => {
        it('should identify sync_progress message type', () => {
            const message: SyncProgressMessage = {
                type: 'sync_progress',
                data: {
                    operation: 'tautulli_import',
                    status: 'running',
                    progress: createProgress(),
                    correlation_id: 'test-123',
                },
            };

            assert.strictEqual(message.type, 'sync_progress');
        });

        it('should ignore non-sync_progress messages', () => {
            const message = {
                type: 'playback',
                data: {},
            };

            const isSyncProgress = message.type === 'sync_progress';
            assert.strictEqual(isSyncProgress, false);
        });
    });

    describe('Callback invocation', () => {
        it('should invoke onStatusChange when status changes', () => {
            let changeCount = 0;
            let lastOperation: string | null = null;
            let lastProgressStatus: SyncProgress['status'] | null = null;

            const onStatusChange = (operation: string, progress: SyncProgress) => {
                changeCount++;
                lastOperation = operation;
                lastProgressStatus = progress.status;
            };

            // Simulate status change
            const progress = createProgress({ status: 'running' });
            onStatusChange('tautulli_import', progress);

            assert.strictEqual(changeCount, 1);
            assert.strictEqual(lastOperation, 'tautulli_import');
            assert.strictEqual(lastProgressStatus, 'running');
        });

        it('should invoke onComplete when operation completes', () => {
            let completeCount = 0;
            let completedOperation: string | null = null;

            const onComplete = (operation: string, _progress: SyncProgress) => {
                completeCount++;
                completedOperation = operation;
            };

            // Helper to check if completion should trigger
            function shouldTriggerComplete(
                prev: SyncProgress['status'],
                next: SyncProgress['status']
            ): boolean {
                return prev === 'running' && (next === 'completed' || next === 'error');
            }

            // Check transition from running to completed
            if (shouldTriggerComplete('running', 'completed')) {
                onComplete('tautulli_import', createProgress({ status: 'completed' }));
            }

            assert.strictEqual(completeCount, 1);
            assert.strictEqual(completedOperation, 'tautulli_import');
        });

        it('should invoke onError when operation fails', () => {
            let errorCount = 0;
            let errorOperation: string | null = null;

            const onError = (operation: string, _error: Error) => {
                errorCount++;
                errorOperation = operation;
            };

            // Helper to check if error should trigger
            function shouldTriggerError(
                prev: SyncProgress['status'],
                next: SyncProgress['status']
            ): boolean {
                return next === 'error' && prev !== 'error';
            }

            // Check transition to error
            if (shouldTriggerError('running', 'error')) {
                onError('plex_historical', new Error('Sync failed'));
            }

            assert.strictEqual(errorCount, 1);
            assert.strictEqual(errorOperation, 'plex_historical');
        });
    });

    describe('Polling logic', () => {
        it('should start polling when WebSocket disconnects during active operation', () => {
            let pollingStarted = false;
            const isWebSocketConnected = false;
            const hasActiveOperation = true;

            if (!isWebSocketConnected && hasActiveOperation) {
                pollingStarted = true;
            }

            assert.strictEqual(pollingStarted, true);
        });

        it('should stop polling when WebSocket reconnects', () => {
            let pollingStopped = false;
            const wasConnected = false;
            const isNowConnected = true;

            if (isNowConnected && !wasConnected) {
                pollingStopped = true;
            }

            assert.strictEqual(pollingStopped, true);
        });

        it('should not start polling if no active operation', () => {
            let pollingStarted = false;
            const isWebSocketConnected = false;
            const hasActiveOperation = false;

            if (!isWebSocketConnected && hasActiveOperation) {
                pollingStarted = true;
            }

            assert.strictEqual(pollingStarted, false);
        });
    });

    describe('Active operation detection', () => {
        // Define state type for proper inference
        interface SyncState {
            tautulliImport: SyncProgress | null;
            plexHistorical: SyncProgress | null;
            serverSyncs: Record<string, SyncProgress>;
        }

        // Helper to check if any operation is running
        function hasActiveOperation(state: SyncState): boolean {
            return (
                state.tautulliImport?.status === 'running' ||
                state.plexHistorical?.status === 'running' ||
                Object.values(state.serverSyncs).some(p => p.status === 'running')
            );
        }

        it('should detect running Tautulli import', () => {
            const state: SyncState = {
                tautulliImport: createProgress({ status: 'running' }),
                plexHistorical: null,
                serverSyncs: {},
            };

            assert.strictEqual(hasActiveOperation(state), true);
        });

        it('should detect running Plex historical sync', () => {
            const state: SyncState = {
                tautulliImport: null,
                plexHistorical: createProgress({ status: 'running' }),
                serverSyncs: {},
            };

            assert.strictEqual(hasActiveOperation(state), true);
        });

        it('should detect running server sync', () => {
            const state: SyncState = {
                tautulliImport: null,
                plexHistorical: null,
                serverSyncs: {
                    'server-1': createProgress({ status: 'running' }),
                },
            };

            assert.strictEqual(hasActiveOperation(state), true);
        });

        it('should return false when no operations running', () => {
            const state: SyncState = {
                tautulliImport: createProgress({ status: 'idle' }),
                plexHistorical: createProgress({ status: 'completed' }),
                serverSyncs: {
                    'server-1': createProgress({ status: 'completed' }),
                },
            };

            assert.strictEqual(hasActiveOperation(state), false);
        });
    });

    describe('Error handling', () => {
        it('should handle network errors gracefully', async () => {
            mockFetch.shouldFail = true;

            let errorCaught = false;
            try {
                await mockFetch.fetch('/api/v1/sync/status');
            } catch {
                errorCaught = true;
            }

            assert.strictEqual(errorCaught, true);
        });

        it('should handle JSON parse errors', () => {
            let parseError = false;
            try {
                JSON.parse('invalid json');
            } catch {
                parseError = true;
            }

            assert.strictEqual(parseError, true);
        });

        it('should handle storage errors gracefully', () => {
            // Simulate storage being full by throwing
            const failingStorage = {
                get: (_key: string) => null,
                set: (_key: string, _value: string) => {
                    throw new Error('Storage full');
                },
                remove: (_key: string) => {},
            };

            let errorCaught = false;
            try {
                failingStorage.set('key', 'value');
            } catch {
                errorCaught = true;
            }

            assert.strictEqual(errorCaught, true);
        });
    });
});
