// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * SyncManager - Data Sync State Management (Production Grade)
 *
 * Manages state for data synchronization operations including:
 * - Tautulli database import
 * - Plex historical sync
 * - Per-server sync status
 *
 * Features:
 * - Class-based with dependency injection for testability
 * - Durable state that survives page refreshes (SafeSessionStorage)
 * - WebSocket integration for real-time progress updates
 * - Polling fallback when WebSocket is disconnected
 * - Correlation IDs for distributed tracing
 * - Structured audit logging
 *
 * Architecture follows PlexPINAuthenticator pattern (plex-pin.ts)
 */

import { SafeSessionStorage } from '../lib/utils/SafeSessionStorage';
import { createLogger } from '../lib/logger';
import type {
    SyncProgress,
    SyncOperationType,
    SyncManagerConfig,
    PersistedSyncState,
    SyncStatusResponse,
    TautulliImportRequest,
    TautulliImportResponse,
    TautulliValidateResponse,
    PlexHistoricalRequest,
    PlexHistoricalResponse,
    SyncProgressMessage,
} from '../lib/types/sync';

const logger = createLogger('SyncManager');

// ============================================================================
// Interfaces for Dependency Injection
// ============================================================================

/**
 * HTTP client interface for making API requests.
 * Abstraction allows for easy mocking in unit tests.
 */
export interface SyncFetchClient {
    fetch<T>(url: string, options?: RequestInit): Promise<{ data: T }>;
}

/**
 * Correlation ID generator interface.
 */
export interface SyncCorrelationIdGenerator {
    generate(): string;
}

/**
 * State storage interface for durability.
 */
export interface SyncStateStorage {
    get(key: string): string | null;
    set(key: string, value: string): void;
    remove(key: string): void;
}

/**
 * Timer interface for polling.
 */
export interface SyncTimerManager {
    setTimeout(callback: () => void, ms: number): number;
    clearTimeout(id: number): void;
    setInterval(callback: () => void, ms: number): number;
    clearInterval(id: number): void;
}

// ============================================================================
// Default Implementations
// ============================================================================

/**
 * Default state storage using SafeSessionStorage
 */
export class SafeSessionStorageSyncStorage implements SyncStateStorage {
    private static readonly KEY_PREFIX = 'sync_manager_';

    get(key: string): string | null {
        return SafeSessionStorage.getItem(SafeSessionStorageSyncStorage.KEY_PREFIX + key);
    }

    set(key: string, value: string): void {
        SafeSessionStorage.setItem(SafeSessionStorageSyncStorage.KEY_PREFIX + key, value);
    }

    remove(key: string): void {
        SafeSessionStorage.removeItem(SafeSessionStorageSyncStorage.KEY_PREFIX + key);
    }
}

/**
 * Default correlation ID generator using crypto API
 */
export class CryptoSyncCorrelationIdGenerator implements SyncCorrelationIdGenerator {
    generate(): string {
        if (typeof crypto !== 'undefined' && crypto.randomUUID) {
            return crypto.randomUUID();
        }
        return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
            const r = (Math.random() * 16) | 0;
            const v = c === 'x' ? r : (r & 0x3) | 0x8;
            return v.toString(16);
        });
    }
}

/**
 * Default timer manager using window timers
 */
export class BrowserSyncTimerManager implements SyncTimerManager {
    setTimeout(callback: () => void, ms: number): number {
        return window.setTimeout(callback, ms);
    }

    clearTimeout(id: number): void {
        window.clearTimeout(id);
    }

    setInterval(callback: () => void, ms: number): number {
        return window.setInterval(callback, ms);
    }

    clearInterval(id: number): void {
        window.clearInterval(id);
    }
}

// ============================================================================
// Default Configuration
// ============================================================================

const DEFAULT_CONFIG: Required<Omit<SyncManagerConfig, 'onStatusChange' | 'onError' | 'onComplete'>> = {
    pollingInterval: 3000, // Poll every 3 seconds when WebSocket is down
    maxStateAge: 5 * 60 * 1000, // 5 minutes max state age
};

const STORAGE_KEY = 'state';

// ============================================================================
// SyncManager Class
// ============================================================================

/**
 * Internal state for tracking sync operations
 */
interface SyncManagerState {
    correlationId: string;
    tautulliImport: SyncProgress | null;
    plexHistorical: SyncProgress | null;
    serverSyncs: Record<string, SyncProgress>;
    isWebSocketConnected: boolean;
    pollingTimerId: number | null;
}

/**
 * Creates an initial empty progress object
 */
function createEmptyProgress(): SyncProgress {
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
    };
}

/**
 * Production-grade sync manager for data synchronization operations.
 *
 * @example
 * ```typescript
 * // Production usage with defaults
 * const syncManager = new SyncManager(api);
 * await syncManager.init();
 *
 * // Start Tautulli import
 * await syncManager.startTautulliImport({ db_path: '/path/to/db' });
 *
 * // Check status
 * const status = syncManager.getTautulliImportStatus();
 *
 * // Test usage with mocks
 * const mockFetch = { fetch: jest.fn() };
 * const syncManager = new SyncManager(mockFetch, config);
 * ```
 */
export class SyncManager {
    private state: SyncManagerState;
    private config: Required<Omit<SyncManagerConfig, 'onStatusChange' | 'onError' | 'onComplete'>>;
    private callbacks: Pick<SyncManagerConfig, 'onStatusChange' | 'onError' | 'onComplete'>;
    private initialized = false;

    constructor(
        private readonly fetchClient: SyncFetchClient,
        config: SyncManagerConfig = {},
        private readonly stateStorage: SyncStateStorage = new SafeSessionStorageSyncStorage(),
        private readonly correlationIdGenerator: SyncCorrelationIdGenerator = new CryptoSyncCorrelationIdGenerator(),
        private readonly timerManager: SyncTimerManager = new BrowserSyncTimerManager()
    ) {
        this.config = { ...DEFAULT_CONFIG, ...config };
        this.callbacks = {
            onStatusChange: config.onStatusChange,
            onError: config.onError,
            onComplete: config.onComplete,
        };
        this.state = this.createInitialState();
    }

    /**
     * Creates a fresh state object
     */
    private createInitialState(): SyncManagerState {
        return {
            correlationId: this.correlationIdGenerator.generate(),
            tautulliImport: null,
            plexHistorical: null,
            serverSyncs: {},
            isWebSocketConnected: false,
            pollingTimerId: null,
        };
    }

    // ========================================================================
    // Initialization & Lifecycle
    // ========================================================================

    /**
     * Initialize the sync manager.
     * Loads any persisted state and checks current sync status.
     */
    async init(): Promise<void> {
        if (this.initialized) {
            logger.warn('SyncManager already initialized');
            return;
        }

        logger.debug('Initializing SyncManager', { correlationId: this.state.correlationId });

        // Try to restore persisted state
        this.loadPersistedState();

        // Fetch current status from backend
        await this.refreshStatus();

        this.initialized = true;
        logger.info('SyncManager initialized');
    }

    /**
     * Clean up resources.
     * Call this when the component is unmounted.
     */
    destroy(): void {
        logger.debug('Destroying SyncManager');

        // Stop polling
        if (this.state.pollingTimerId !== null) {
            this.timerManager.clearInterval(this.state.pollingTimerId);
            this.state.pollingTimerId = null;
        }

        this.initialized = false;
    }

    // ========================================================================
    // State Persistence
    // ========================================================================

    /**
     * Persist current state to storage for durability
     */
    private persistState(): void {
        const activeOp = this.getActiveOperation();
        if (!activeOp) {
            return; // Nothing to persist
        }

        const [operation, progress] = activeOp;
        const persistedState: PersistedSyncState = {
            correlationId: this.state.correlationId,
            operation,
            status: progress.status,
            progress,
            persistedAt: Date.now(),
            expiresAt: Date.now() + this.config.maxStateAge,
        };

        try {
            this.stateStorage.set(STORAGE_KEY, JSON.stringify(persistedState));
            logger.debug('State persisted', { operation, status: progress.status });
        } catch (error) {
            logger.warn('Failed to persist state', { error });
        }
    }

    /**
     * Load persisted state from storage
     */
    private loadPersistedState(): void {
        try {
            const stored = this.stateStorage.get(STORAGE_KEY);
            if (!stored) {
                return;
            }

            const state: PersistedSyncState = JSON.parse(stored);

            // Check if state has expired
            if (Date.now() > state.expiresAt) {
                logger.debug('Persisted state expired, clearing');
                this.stateStorage.remove(STORAGE_KEY);
                return;
            }

            // Only restore state for running operations
            if (state.status !== 'running') {
                this.stateStorage.remove(STORAGE_KEY);
                return;
            }

            // Restore state based on operation type
            switch (state.operation) {
                case 'tautulli_import':
                    this.state.tautulliImport = state.progress;
                    break;
                case 'plex_historical':
                    this.state.plexHistorical = state.progress;
                    break;
                case 'server_sync':
                    // Server syncs need server_id which we don't have here
                    break;
            }

            this.state.correlationId = state.correlationId;
            logger.info('Restored persisted state', {
                operation: state.operation,
                status: state.status,
            });
        } catch (error) {
            logger.warn('Failed to load persisted state', { error });
            this.stateStorage.remove(STORAGE_KEY);
        }
    }

    /**
     * Clear persisted state
     */
    private clearPersistedState(): void {
        try {
            this.stateStorage.remove(STORAGE_KEY);
        } catch {
            // Ignore errors during cleanup
        }
    }

    /**
     * Get the currently active operation, if any
     */
    private getActiveOperation(): [SyncOperationType, SyncProgress] | null {
        if (this.state.tautulliImport?.status === 'running') {
            return ['tautulli_import', this.state.tautulliImport];
        }
        if (this.state.plexHistorical?.status === 'running') {
            return ['plex_historical', this.state.plexHistorical];
        }
        for (const [_serverId, progress] of Object.entries(this.state.serverSyncs)) {
            if (progress.status === 'running') {
                // For server syncs, we'd need to encode the serverId somehow
                return ['server_sync', progress];
            }
        }
        return null;
    }

    // ========================================================================
    // Status Refresh
    // ========================================================================

    /**
     * Refresh status from backend
     */
    async refreshStatus(): Promise<void> {
        try {
            const response = await this.fetchClient.fetch<SyncStatusResponse>('/api/v1/sync/status');
            this.updateFromStatusResponse(response.data);
        } catch (error) {
            logger.warn('Failed to refresh sync status', { error });
            // Don't throw - we might be in a network-down scenario
        }
    }

    /**
     * Update internal state from status response
     */
    private updateFromStatusResponse(response: SyncStatusResponse): void {
        const prevTautulli = this.state.tautulliImport;
        const prevPlex = this.state.plexHistorical;

        if (response.tautulli_import) {
            this.state.tautulliImport = response.tautulli_import;
            this.checkForCompletion('tautulli_import', prevTautulli, response.tautulli_import);
        }

        if (response.plex_historical) {
            this.state.plexHistorical = response.plex_historical;
            this.checkForCompletion('plex_historical', prevPlex, response.plex_historical);
        }

        if (response.server_syncs) {
            for (const [serverId, progress] of Object.entries(response.server_syncs)) {
                const prevProgress = this.state.serverSyncs[serverId];
                this.state.serverSyncs[serverId] = progress;
                this.checkForCompletion('server_sync', prevProgress, progress);
            }
        }

        // Persist if there's an active operation
        if (this.getActiveOperation()) {
            this.persistState();
        } else {
            this.clearPersistedState();
        }
    }

    /**
     * Check if an operation just completed and fire callback
     */
    private checkForCompletion(
        operation: SyncOperationType,
        prev: SyncProgress | null | undefined,
        current: SyncProgress
    ): void {
        // Notify status change
        this.callbacks.onStatusChange?.(operation, current);

        // Check for completion
        if (prev?.status === 'running' && (current.status === 'completed' || current.status === 'error')) {
            logger.info('Sync operation completed', {
                operation,
                status: current.status,
                processedRecords: current.processed_records,
            });
            this.callbacks.onComplete?.(operation, current);
        }

        // Check for error
        if (current.status === 'error' && prev?.status !== 'error') {
            const error = new Error(`Sync operation failed: ${operation}`);
            this.callbacks.onError?.(operation, error);
        }
    }

    // ========================================================================
    // WebSocket Integration
    // ========================================================================

    /**
     * Handle incoming WebSocket message for sync progress
     */
    handleWebSocketMessage(message: SyncProgressMessage): void {
        if (message.type !== 'sync_progress') {
            return;
        }

        const { operation, progress, server_id } = message.data;

        logger.debug('Received sync progress via WebSocket', {
            operation,
            status: progress.status,
            progressPercent: progress.progress_percent,
        });

        switch (operation) {
            case 'tautulli_import':
                this.updateProgress('tautulli_import', progress);
                break;
            case 'plex_historical':
                this.updateProgress('plex_historical', progress);
                break;
            case 'server_sync':
                if (server_id) {
                    this.updateProgress('server_sync', progress, server_id);
                }
                break;
        }
    }

    /**
     * Update progress for a specific operation
     */
    private updateProgress(operation: SyncOperationType, progress: SyncProgress, serverId?: string): void {
        let prev: SyncProgress | null | undefined;

        switch (operation) {
            case 'tautulli_import':
                prev = this.state.tautulliImport;
                this.state.tautulliImport = progress;
                break;
            case 'plex_historical':
                prev = this.state.plexHistorical;
                this.state.plexHistorical = progress;
                break;
            case 'server_sync':
                if (serverId) {
                    prev = this.state.serverSyncs[serverId];
                    this.state.serverSyncs[serverId] = progress;
                }
                break;
        }

        this.checkForCompletion(operation, prev, progress);

        // Persist if running
        if (progress.status === 'running') {
            this.persistState();
        } else if (progress.status === 'completed' || progress.status === 'error') {
            this.clearPersistedState();
        }
    }

    /**
     * Set WebSocket connection status.
     * When disconnected, starts polling. When reconnected, stops polling.
     */
    setWebSocketConnected(connected: boolean): void {
        const wasConnected = this.state.isWebSocketConnected;
        this.state.isWebSocketConnected = connected;

        if (connected && !wasConnected) {
            // WebSocket reconnected - stop polling
            this.stopPolling();
            logger.debug('WebSocket connected, stopped polling');
        } else if (!connected && wasConnected && this.getActiveOperation()) {
            // WebSocket disconnected during active operation - start polling
            this.startPolling();
            logger.debug('WebSocket disconnected, started polling fallback');
        }
    }

    /**
     * Start polling for status updates
     */
    private startPolling(): void {
        if (this.state.pollingTimerId !== null) {
            return; // Already polling
        }

        this.state.pollingTimerId = this.timerManager.setInterval(() => {
            this.refreshStatus();
        }, this.config.pollingInterval);
    }

    /**
     * Stop polling for status updates
     */
    private stopPolling(): void {
        if (this.state.pollingTimerId !== null) {
            this.timerManager.clearInterval(this.state.pollingTimerId);
            this.state.pollingTimerId = null;
        }
    }

    // ========================================================================
    // Tautulli Import Operations
    // ========================================================================

    /**
     * Start a Tautulli database import.
     *
     * @param request - Import options
     * @returns Import start response
     */
    async startTautulliImport(request: TautulliImportRequest = {}): Promise<TautulliImportResponse> {
        logger.info('Starting Tautulli import', {
            correlationId: this.state.correlationId,
            dbPath: request.db_path,
            resume: request.resume,
            dryRun: request.dry_run,
        });

        // Initialize progress state
        this.state.tautulliImport = {
            ...createEmptyProgress(),
            status: 'running',
            dry_run: request.dry_run,
        };

        try {
            const response = await this.fetchClient.fetch<TautulliImportResponse>('/api/v1/import/tautulli', {
                method: 'POST',
                body: JSON.stringify(request),
            });

            if (response.data.stats) {
                this.state.tautulliImport = response.data.stats;
            }

            this.persistState();

            // Start polling if WebSocket is not connected
            if (!this.state.isWebSocketConnected) {
                this.startPolling();
            }

            return response.data;
        } catch (error) {
            this.state.tautulliImport = {
                ...createEmptyProgress(),
                status: 'error',
            };
            throw error;
        }
    }

    /**
     * Stop the current Tautulli import.
     *
     * @returns Stop response with final statistics
     */
    async stopTautulliImport(): Promise<TautulliImportResponse> {
        logger.info('Stopping Tautulli import', { correlationId: this.state.correlationId });

        const response = await this.fetchClient.fetch<TautulliImportResponse>('/api/v1/import', {
            method: 'DELETE',
        });

        if (response.data.stats) {
            this.state.tautulliImport = response.data.stats;
        } else if (this.state.tautulliImport) {
            this.state.tautulliImport.status = 'cancelled';
        }

        this.stopPolling();
        this.clearPersistedState();

        return response.data;
    }

    /**
     * Validate a Tautulli database file.
     *
     * @param dbPath - Path to the database file
     * @returns Validation result
     */
    async validateTautulliDatabase(dbPath: string): Promise<TautulliValidateResponse> {
        logger.debug('Validating Tautulli database', { dbPath });

        const response = await this.fetchClient.fetch<TautulliValidateResponse>('/api/v1/import/validate', {
            method: 'POST',
            body: JSON.stringify({ db_path: dbPath }),
        });

        return response.data;
    }

    /**
     * Get current Tautulli import status.
     *
     * @returns Current progress or null if no import
     */
    getTautulliImportStatus(): SyncProgress | null {
        return this.state.tautulliImport;
    }

    // ========================================================================
    // Plex Historical Sync Operations
    // ========================================================================

    /**
     * Start a Plex historical sync.
     *
     * @param request - Sync options
     * @returns Sync start response
     */
    async startPlexHistoricalSync(request: PlexHistoricalRequest = {}): Promise<PlexHistoricalResponse> {
        logger.info('Starting Plex historical sync', {
            correlationId: this.state.correlationId,
            daysBack: request.days_back,
            libraryIds: request.library_ids,
        });

        // Check if Tautulli import is running (mutex)
        if (this.state.tautulliImport?.status === 'running') {
            throw new Error('Cannot start Plex historical sync while Tautulli import is running');
        }

        // Initialize progress state
        this.state.plexHistorical = {
            ...createEmptyProgress(),
            status: 'running',
        };

        try {
            const response = await this.fetchClient.fetch<PlexHistoricalResponse>('/api/v1/sync/plex/historical', {
                method: 'POST',
                body: JSON.stringify(request),
            });

            this.persistState();

            // Start polling if WebSocket is not connected
            if (!this.state.isWebSocketConnected) {
                this.startPolling();
            }

            return response.data;
        } catch (error) {
            this.state.plexHistorical = {
                ...createEmptyProgress(),
                status: 'error',
            };
            throw error;
        }
    }

    /**
     * Get current Plex historical sync status.
     *
     * @returns Current progress or null if no sync
     */
    getPlexHistoricalStatus(): SyncProgress | null {
        return this.state.plexHistorical;
    }

    // ========================================================================
    // Server Sync Status
    // ========================================================================

    /**
     * Get sync status for a specific server.
     *
     * @param serverId - The server ID
     * @returns Server sync progress or null
     */
    getServerSyncStatus(serverId: string): SyncProgress | null {
        return this.state.serverSyncs[serverId] || null;
    }

    /**
     * Get all server sync statuses.
     *
     * @returns Map of server IDs to progress
     */
    getAllServerSyncStatuses(): Record<string, SyncProgress> {
        return { ...this.state.serverSyncs };
    }

    // ========================================================================
    // Utility Methods
    // ========================================================================

    /**
     * Check if any sync operation is currently running.
     */
    isAnySyncRunning(): boolean {
        return this.getActiveOperation() !== null;
    }

    /**
     * Check if Tautulli import is available (not blocked by mutex).
     */
    isTautulliImportAvailable(): boolean {
        return this.state.plexHistorical?.status !== 'running';
    }

    /**
     * Check if Plex historical sync is available (not blocked by mutex).
     */
    isPlexHistoricalAvailable(): boolean {
        return this.state.tautulliImport?.status !== 'running';
    }

    /**
     * Get the current correlation ID for tracing.
     */
    getCorrelationId(): string {
        return this.state.correlationId;
    }

    /**
     * Update callbacks after initialization.
     */
    setCallbacks(callbacks: Pick<SyncManagerConfig, 'onStatusChange' | 'onError' | 'onComplete'>): void {
        this.callbacks = { ...this.callbacks, ...callbacks };
    }
}

export default SyncManager;
