// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Plex PIN-Based Authentication Flow (Production Grade)
 *
 * Implements the Plex PIN authentication method (same approach as Overseerr).
 * This flow does NOT require HTTPS or a public domain, making it ideal for:
 * - Local network deployments (http://192.168.x.x)
 * - Homelab setups without public DNS
 * - Development environments
 *
 * Architecture:
 * - Class-based with dependency injection for testability
 * - Structured audit logging for security compliance
 * - Exponential backoff for resilient polling
 * - Correlation IDs for distributed tracing
 * - CSRF protection via server-side session binding
 * - Durable state storage for recovery after page refresh
 *
 * How it works:
 * 1. Request a PIN from the backend (which gets it from Plex)
 * 2. Open Plex auth page in a popup/new tab
 * 3. Poll the backend with exponential backoff to check authorization
 * 4. Once authorized, complete authentication and create session
 *
 * Security:
 * - PIN is short-lived (expires in ~5 minutes)
 * - Auth token is only returned after user explicitly approves on plex.tv
 * - All sensitive operations happen on Plex's servers
 * - CSRF token bound to server-side session
 * - Correlation IDs enable audit trail across distributed systems
 * - Session storage (not local storage) ensures auth state doesn't persist across browser sessions
 *
 * @module plex-pin
 */

import { SafeSessionStorage } from '../utils/SafeSessionStorage';

// ============================================================================
// Interfaces for Dependency Injection
// ============================================================================

/**
 * HTTP client interface for making API requests.
 * Abstraction allows for easy mocking in unit tests.
 */
export interface FetchClient {
    fetch(url: string, options?: RequestInit): Promise<Response>;
}

/**
 * Structured audit event for security logging.
 * Follows OWASP logging guidelines for authentication events.
 */
export interface AuditEvent {
    /** Unique identifier for tracing across services */
    correlationId: string;
    /** ISO 8601 timestamp */
    timestamp: string;
    /** Authentication event type */
    eventType: PlexAuthAuditEventType;
    /** Event severity level */
    level: 'info' | 'warn' | 'error';
    /** Human-readable description */
    message: string;
    /** Additional structured data */
    metadata?: Record<string, unknown>;
}

/**
 * Authentication audit event types.
 * Covers the complete authentication lifecycle.
 */
export type PlexAuthAuditEventType =
    | 'PLEX_AUTH_INITIATED'
    | 'PLEX_PIN_REQUESTED'
    | 'PLEX_PIN_RECEIVED'
    | 'PLEX_POPUP_OPENED'
    | 'PLEX_POPUP_BLOCKED'
    | 'PLEX_POPUP_CLOSED'
    | 'PLEX_POLL_STARTED'
    | 'PLEX_POLL_CHECK'
    | 'PLEX_POLL_AUTHORIZED'
    | 'PLEX_POLL_PENDING'
    | 'PLEX_POLL_ERROR'
    | 'PLEX_POLL_RETRY'
    | 'PLEX_AUTH_COMPLETING'
    | 'PLEX_AUTH_SUCCESS'
    | 'PLEX_AUTH_FAILED'
    | 'PLEX_AUTH_TIMEOUT'
    | 'PLEX_AUTH_CANCELLED'
    | 'PLEX_CSRF_VALIDATION';

/**
 * Audit logger interface for security event logging.
 * Implementations can send to console, remote logging service, or SIEM.
 */
export interface AuditLogger {
    log(event: AuditEvent): void;
}

/**
 * Correlation ID generator interface.
 * Allows for custom ID generation strategies (UUID, ULID, etc.).
 */
export interface CorrelationIdGenerator {
    generate(): string;
}

/**
 * Window abstraction for popup management.
 * Enables testing without actual browser window.
 */
export interface WindowManager {
    open(url: string, target: string, features: string): Window | null;
    setTimeout(callback: () => void, ms: number): number;
    clearTimeout(id: number): void;
    getLocation(): Location;
    setLocationHref(url: string): void;
}

/**
 * Persistent state storage interface for durability.
 * Allows auth flow to survive page refreshes.
 */
export interface StateStorage {
    get(key: string): string | null;
    set(key: string, value: string): void;
    remove(key: string): void;
}

/**
 * Serializable state for persistence.
 * Contains only the data needed to resume an auth flow.
 */
export interface PersistedPlexAuthState {
    correlationId: string;
    pinId: number;
    pinCode: string;
    authUrl: string;
    csrfToken: string | null;
    status: PlexAuthStatus;
    currentPollInterval: number;
    pollAttempts: number;
    createdAt: number;
    expiresAt: number;
}

// ============================================================================
// API Response Types
// ============================================================================

/**
 * Response from GET /api/auth/plex/login
 */
export interface PlexPINResponse {
    pin_id: number;
    pin_code: string;
    auth_url: string;
    expires: string;
    /** CSRF token bound to server-side session */
    csrf_token?: string;
}

/**
 * Response from GET /api/auth/plex/poll
 */
export interface PlexPollResponse {
    authorized: boolean;
    expires: string;
}

/**
 * Response from POST /api/auth/plex/callback
 */
export interface PlexCallbackResponse {
    success: boolean;
    redirect_url: string;
    user: PlexUser;
}

/**
 * Authenticated Plex user information
 */
export interface PlexUser {
    id: string;
    username: string;
    email: string;
    roles: string[];
}

// ============================================================================
// Configuration Types
// ============================================================================

/**
 * Configuration for the PIN authentication flow
 */
export interface PlexPINConfig {
    /** Initial poll interval in ms (will increase with backoff) */
    initialPollInterval?: number;
    /** Maximum poll interval in ms (backoff ceiling) */
    maxPollInterval?: number;
    /** Backoff multiplier (e.g., 1.5 = 50% increase each retry) */
    backoffMultiplier?: number;
    /** Maximum time to wait for authorization (ms) */
    timeout?: number;
    /** Whether to use a popup (true) or redirect (false) */
    usePopup?: boolean;
    /** Callback when authentication succeeds */
    onSuccess?: (user: PlexUser) => void;
    /** Callback when authentication fails */
    onError?: (error: Error) => void;
    /** Callback for status updates */
    onStatusChange?: (status: PlexAuthStatus) => void;
}

/**
 * Status of the PIN authentication flow
 */
export type PlexAuthStatus =
    | 'idle'
    | 'requesting_pin'
    | 'waiting_for_user'
    | 'checking_authorization'
    | 'completing_auth'
    | 'success'
    | 'error'
    | 'timeout'
    | 'cancelled';

/**
 * Internal state of an ongoing authentication flow
 */
interface PlexAuthState {
    correlationId: string;
    pinId: number | null;
    pinCode: string | null;
    authUrl: string | null;
    csrfToken: string | null;
    status: PlexAuthStatus;
    popup: Window | null;
    pollTimer: number | null;
    timeoutTimer: number | null;
    abortController: AbortController | null;
    currentPollInterval: number;
    pollAttempts: number;
}

// ============================================================================
// Default Implementations
// ============================================================================

/**
 * Default fetch client using browser's fetch API
 */
export class BrowserFetchClient implements FetchClient {
    constructor(private baseUrl: string = '') {}

    async fetch(url: string, options?: RequestInit): Promise<Response> {
        const fullUrl = this.baseUrl ? `${this.baseUrl}${url}` : url;
        return fetch(fullUrl, {
            ...options,
            credentials: 'include',
        });
    }
}

/**
 * Default audit logger that logs to console with structured format
 */
export class ConsoleAuditLogger implements AuditLogger {
    constructor(private prefix: string = '[PlexPIN]') {}

    log(event: AuditEvent): void {
        const logFn = event.level === 'error' ? console.error :
                      event.level === 'warn' ? console.warn : console.info;

        logFn(`${this.prefix} [${event.correlationId}] ${event.eventType}: ${event.message}`,
              event.metadata ? JSON.stringify(event.metadata) : '');
    }
}

/**
 * Default correlation ID generator using crypto API
 */
export class CryptoCorrelationIdGenerator implements CorrelationIdGenerator {
    generate(): string {
        // Generate a UUID v4 style ID using crypto API
        if (typeof crypto !== 'undefined' && crypto.randomUUID) {
            return crypto.randomUUID();
        }
        // Fallback for older browsers
        return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
            const r = Math.random() * 16 | 0;
            const v = c === 'x' ? r : (r & 0x3 | 0x8);
            return v.toString(16);
        });
    }
}

/**
 * Default window manager using browser's window object
 */
export class BrowserWindowManager implements WindowManager {
    open(url: string, target: string, features: string): Window | null {
        return window.open(url, target, features);
    }

    setTimeout(callback: () => void, ms: number): number {
        return window.setTimeout(callback, ms);
    }

    clearTimeout(id: number): void {
        window.clearTimeout(id);
    }

    getLocation(): Location {
        return window.location;
    }

    setLocationHref(url: string): void {
        window.location.href = url;
    }
}

/**
 * Default state storage using SafeSessionStorage for durability across page refreshes.
 * Uses session storage (not local storage) for security - auth state should not persist
 * across browser sessions.
 */
export class SafeSessionStorageStateStorage implements StateStorage {
    private static readonly KEY_PREFIX = 'plex_pin_auth_';

    get(key: string): string | null {
        return SafeSessionStorage.getItem(SafeSessionStorageStateStorage.KEY_PREFIX + key);
    }

    set(key: string, value: string): void {
        SafeSessionStorage.setItem(SafeSessionStorageStateStorage.KEY_PREFIX + key, value);
    }

    remove(key: string): void {
        SafeSessionStorage.removeItem(SafeSessionStorageStateStorage.KEY_PREFIX + key);
    }
}

// ============================================================================
// Default Configuration
// ============================================================================

const DEFAULT_CONFIG: Required<Omit<PlexPINConfig, 'onSuccess' | 'onError' | 'onStatusChange'>> = {
    initialPollInterval: 2000,   // Start polling every 2 seconds
    maxPollInterval: 10000,      // Cap at 10 seconds between polls
    backoffMultiplier: 1.5,      // Increase interval by 50% each retry
    timeout: 300000,             // 5 minute timeout
    usePopup: true,              // Use popup by default
};

// ============================================================================
// PlexPINAuthenticator Class
// ============================================================================

/**
 * Storage key for persisted auth state
 */
const PERSISTED_STATE_KEY = 'state';

/**
 * Production-grade Plex PIN-based authenticator.
 *
 * Features:
 * - Dependency injection for all external services
 * - Structured audit logging for security compliance
 * - Exponential backoff for resilient polling
 * - Correlation IDs for distributed tracing
 * - CSRF protection via server-side session binding
 * - Durable state that survives page refreshes
 * - Resume capability for interrupted auth flows
 * - Fully mockable for unit tests
 *
 * @example
 * ```typescript
 * // Production usage with defaults
 * const authenticator = new PlexPINAuthenticator();
 * const result = await authenticator.authenticate({
 *     onStatusChange: (status) => updateUI(status),
 *     onSuccess: (user) => console.log('Logged in as', user.username),
 * });
 *
 * // Resume an interrupted flow after page refresh
 * if (authenticator.hasPendingAuth()) {
 *     const result = await authenticator.resume(config);
 * }
 *
 * // Test usage with mocks
 * const mockFetch = { fetch: jest.fn() };
 * const mockLogger = { log: jest.fn() };
 * const mockStorage = { get: jest.fn(), set: jest.fn(), remove: jest.fn() };
 * const authenticator = new PlexPINAuthenticator(mockFetch, mockLogger, undefined, undefined, mockStorage);
 * ```
 */
export class PlexPINAuthenticator {
    private state: PlexAuthState;

    constructor(
        private readonly fetchClient: FetchClient = new BrowserFetchClient(),
        private readonly auditLogger: AuditLogger = new ConsoleAuditLogger(),
        private readonly correlationIdGenerator: CorrelationIdGenerator = new CryptoCorrelationIdGenerator(),
        private readonly windowManager: WindowManager = new BrowserWindowManager(),
        private readonly stateStorage: StateStorage = new SafeSessionStorageStateStorage()
    ) {
        this.state = this.createInitialState();
    }

    /**
     * Creates a fresh authentication state
     */
    private createInitialState(): PlexAuthState {
        return {
            correlationId: '',
            pinId: null,
            pinCode: null,
            authUrl: null,
            csrfToken: null,
            status: 'idle',
            popup: null,
            pollTimer: null,
            timeoutTimer: null,
            abortController: null,
            currentPollInterval: DEFAULT_CONFIG.initialPollInterval,
            pollAttempts: 0,
        };
    }

    /**
     * Persists the current auth state to storage.
     * Called after significant state changes for durability.
     */
    private persistState(): void {
        if (!this.state.pinId || !this.state.pinCode || !this.state.authUrl) {
            return; // Nothing meaningful to persist yet
        }

        const persistedState: PersistedPlexAuthState = {
            correlationId: this.state.correlationId,
            pinId: this.state.pinId,
            pinCode: this.state.pinCode,
            authUrl: this.state.authUrl,
            csrfToken: this.state.csrfToken,
            status: this.state.status,
            currentPollInterval: this.state.currentPollInterval,
            pollAttempts: this.state.pollAttempts,
            createdAt: Date.now(),
            // PIN expires in ~5 minutes, use 4 minutes to be safe
            expiresAt: Date.now() + (4 * 60 * 1000),
        };

        try {
            this.stateStorage.set(PERSISTED_STATE_KEY, JSON.stringify(persistedState));
        } catch (error) {
            this.audit('PLEX_AUTH_FAILED', 'warn', 'Failed to persist auth state', {
                error: error instanceof Error ? error.message : 'Unknown error',
            });
        }
    }

    /**
     * Loads persisted auth state from storage.
     * Returns null if no valid state exists or state has expired.
     */
    private loadPersistedState(): PersistedPlexAuthState | null {
        try {
            const stored = this.stateStorage.get(PERSISTED_STATE_KEY);
            if (!stored) {
                return null;
            }

            const state: PersistedPlexAuthState = JSON.parse(stored);

            // Check if state has expired
            if (Date.now() > state.expiresAt) {
                this.stateStorage.remove(PERSISTED_STATE_KEY);
                return null;
            }

            // Only return state that can be resumed (waiting_for_user)
            if (!['waiting_for_user', 'checking_authorization'].includes(state.status)) {
                this.stateStorage.remove(PERSISTED_STATE_KEY);
                return null;
            }

            return state;
        } catch {
            this.stateStorage.remove(PERSISTED_STATE_KEY);
            return null;
        }
    }

    /**
     * Clears persisted auth state from storage.
     */
    private clearPersistedState(): void {
        try {
            this.stateStorage.remove(PERSISTED_STATE_KEY);
        } catch {
            // Ignore errors during cleanup
        }
    }

    /**
     * Checks if there is a pending auth flow that can be resumed.
     * Call this on page load to check for interrupted auth flows.
     *
     * @returns True if a resumable auth flow exists
     *
     * @example
     * ```typescript
     * const authenticator = new PlexPINAuthenticator();
     * if (authenticator.hasPendingAuth()) {
     *     // Show "Resume authentication?" UI
     *     const result = await authenticator.resume(config);
     * }
     * ```
     */
    hasPendingAuth(): boolean {
        return this.loadPersistedState() !== null;
    }

    /**
     * Gets information about the pending auth flow, if any.
     * Useful for displaying status to the user.
     *
     * @returns Pending auth info or null if none exists
     */
    getPendingAuthInfo(): { correlationId: string; createdAt: number; expiresAt: number } | null {
        const state = this.loadPersistedState();
        if (!state) {
            return null;
        }
        return {
            correlationId: state.correlationId,
            createdAt: state.createdAt,
            expiresAt: state.expiresAt,
        };
    }

    /**
     * Resumes an interrupted authentication flow.
     * Call this after page refresh if hasPendingAuth() returns true.
     *
     * The resumed flow will:
     * 1. Restore the correlation ID for tracing continuity
     * 2. Re-open the auth popup (or resume polling if already authorized)
     * 3. Continue polling with exponential backoff
     *
     * @param config - Configuration (same as authenticate())
     * @returns Promise that resolves when authentication completes
     * @throws Error if no pending auth flow exists
     *
     * @example
     * ```typescript
     * const authenticator = new PlexPINAuthenticator();
     *
     * // On page load
     * if (authenticator.hasPendingAuth()) {
     *     try {
     *         const result = await authenticator.resume({
     *             onStatusChange: (status) => updateUI(status),
     *             onSuccess: (user) => handleLogin(user),
     *         });
     *     } catch (error) {
     *         console.error('Resume failed:', error);
     *     }
     * }
     * ```
     */
    async resume(config: PlexPINConfig = {}): Promise<PlexCallbackResponse> {
        const persistedState = this.loadPersistedState();
        if (!persistedState) {
            throw new Error('No pending authentication to resume');
        }

        const mergedConfig = { ...DEFAULT_CONFIG, ...config };

        // Cancel any existing auth flow (but don't clear persisted state)
        this.cleanup();

        // Restore state from persisted data
        this.state = this.createInitialState();
        this.state.correlationId = persistedState.correlationId;
        this.state.pinId = persistedState.pinId;
        this.state.pinCode = persistedState.pinCode;
        this.state.authUrl = persistedState.authUrl;
        this.state.csrfToken = persistedState.csrfToken;
        this.state.status = 'waiting_for_user';
        this.state.currentPollInterval = persistedState.currentPollInterval;
        this.state.pollAttempts = persistedState.pollAttempts;
        this.state.abortController = new AbortController();

        this.audit('PLEX_AUTH_INITIATED', 'info', 'Resuming interrupted authentication flow', {
            pinId: persistedState.pinId,
            originalCorrelationId: persistedState.correlationId,
            pollAttempts: persistedState.pollAttempts,
            ageMs: Date.now() - persistedState.createdAt,
        });

        config.onStatusChange?.('waiting_for_user');

        try {
            // Re-open the popup for the user
            if (mergedConfig.usePopup) {
                this.openAuthPopup(this.state.authUrl!);
            }

            // Calculate remaining time based on original expiry
            const remainingMs = persistedState.expiresAt - Date.now();
            const effectiveTimeout = Math.min(remainingMs, mergedConfig.timeout);

            // Set up timeout
            const timeoutPromise = new Promise<never>((_, reject) => {
                this.state.timeoutTimer = this.windowManager.setTimeout(() => {
                    this.state.status = 'timeout';
                    config.onStatusChange?.('timeout');
                    this.audit('PLEX_AUTH_TIMEOUT', 'warn', 'Resumed authentication timed out', {
                        timeout: effectiveTimeout,
                    });
                    this.clearPersistedState();
                    reject(new Error('Plex authentication timed out. Please try again.'));
                }, effectiveTimeout);
            });

            // Resume polling
            const authPromise = this.pollForAuthorization(
                this.state.pinId!,
                mergedConfig,
                config.onStatusChange
            );

            await Promise.race([authPromise, timeoutPromise]);

            // Complete authentication
            this.state.status = 'completing_auth';
            config.onStatusChange?.('completing_auth');

            this.audit('PLEX_AUTH_COMPLETING', 'info', 'Completing resumed authentication');

            const callbackResult = await this.completeAuthentication(this.state.pinId!);

            // Success!
            this.state.status = 'success';
            config.onStatusChange?.('success');
            config.onSuccess?.(callbackResult.user);

            this.audit('PLEX_AUTH_SUCCESS', 'info', 'Resumed authentication completed successfully', {
                userId: callbackResult.user.id,
                username: callbackResult.user.username,
                roles: callbackResult.user.roles,
                totalPollAttempts: this.state.pollAttempts,
            });

            this.cleanup();
            this.clearPersistedState();

            return callbackResult;

        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : 'Resumed authentication failed';

            this.audit('PLEX_AUTH_FAILED', 'error', 'Resumed authentication failed', {
                error: errorMessage,
                pinId: this.state.pinId,
                pollAttempts: this.state.pollAttempts,
            });

            this.state.status = 'error';
            config.onStatusChange?.('error');
            config.onError?.(error instanceof Error ? error : new Error(errorMessage));

            this.cleanup();
            this.clearPersistedState();
            throw error;
        }
    }

    /**
     * Logs an audit event with the current correlation ID
     */
    private audit(
        eventType: PlexAuthAuditEventType,
        level: AuditEvent['level'],
        message: string,
        metadata?: Record<string, unknown>
    ): void {
        this.auditLogger.log({
            correlationId: this.state.correlationId,
            timestamp: new Date().toISOString(),
            eventType,
            level,
            message,
            metadata,
        });
    }

    /**
     * Calculates the next poll interval using exponential backoff
     */
    private calculateNextPollInterval(config: Required<Omit<PlexPINConfig, 'onSuccess' | 'onError' | 'onStatusChange'>>): number {
        const nextInterval = this.state.currentPollInterval * config.backoffMultiplier;
        return Math.min(nextInterval, config.maxPollInterval);
    }

    /**
     * Initiates Plex PIN-based authentication.
     *
     * This is the main entry point for Plex login. It:
     * 1. Requests a PIN from the backend
     * 2. Opens Plex auth in a popup (or redirects if popup is blocked)
     * 3. Polls for authorization with exponential backoff
     * 4. Completes authentication when user approves
     *
     * @param config - Optional configuration
     * @returns Promise that resolves when authentication completes
     */
    async authenticate(config: PlexPINConfig = {}): Promise<PlexCallbackResponse> {
        const mergedConfig = { ...DEFAULT_CONFIG, ...config };

        // Cancel any existing auth flow
        this.cancel();

        // Initialize new auth state with fresh correlation ID
        this.state = this.createInitialState();
        this.state.correlationId = this.correlationIdGenerator.generate();
        this.state.status = 'requesting_pin';
        this.state.abortController = new AbortController();
        this.state.currentPollInterval = mergedConfig.initialPollInterval;

        this.audit('PLEX_AUTH_INITIATED', 'info', 'Plex PIN authentication flow initiated', {
            usePopup: mergedConfig.usePopup,
            timeout: mergedConfig.timeout,
        });

        config.onStatusChange?.('requesting_pin');

        try {
            // Step 1: Request a PIN from the backend
            const pinResponse = await this.requestPIN();

            this.state.pinId = pinResponse.pin_id;
            this.state.pinCode = pinResponse.pin_code;
            this.state.authUrl = pinResponse.auth_url;
            this.state.csrfToken = pinResponse.csrf_token || null;

            this.audit('PLEX_PIN_RECEIVED', 'info', 'PIN received from backend', {
                pinId: pinResponse.pin_id,
                hasCSRFToken: !!pinResponse.csrf_token,
            });

            // Step 2: Open Plex auth page
            this.state.status = 'waiting_for_user';
            config.onStatusChange?.('waiting_for_user');

            // Persist state for durability (survive page refresh)
            this.persistState();

            if (mergedConfig.usePopup) {
                this.openAuthPopup(pinResponse.auth_url);
            } else {
                // For mobile or when popups are blocked, redirect
                this.audit('PLEX_AUTH_INITIATED', 'info', 'Redirecting to Plex auth page');
                this.windowManager.setLocationHref(pinResponse.auth_url);
                // This won't return - user will come back via forward URL
                return new Promise(() => {});
            }

            // Step 3: Set up timeout
            const timeoutPromise = new Promise<never>((_, reject) => {
                this.state.timeoutTimer = this.windowManager.setTimeout(() => {
                    this.state.status = 'timeout';
                    config.onStatusChange?.('timeout');
                    this.audit('PLEX_AUTH_TIMEOUT', 'warn', 'Authentication timed out', {
                        timeout: mergedConfig.timeout,
                    });
                    reject(new Error('Plex authentication timed out. Please try again.'));
                }, mergedConfig.timeout);
            });

            // Step 4: Poll for authorization with exponential backoff
            const authPromise = this.pollForAuthorization(
                pinResponse.pin_id,
                mergedConfig,
                config.onStatusChange
            );

            // Wait for either authorization or timeout
            await Promise.race([authPromise, timeoutPromise]);

            // Step 5: Complete authentication
            this.state.status = 'completing_auth';
            config.onStatusChange?.('completing_auth');

            this.audit('PLEX_AUTH_COMPLETING', 'info', 'Completing authentication');

            const callbackResult = await this.completeAuthentication(pinResponse.pin_id);

            // Success!
            this.state.status = 'success';
            config.onStatusChange?.('success');
            config.onSuccess?.(callbackResult.user);

            this.audit('PLEX_AUTH_SUCCESS', 'info', 'Authentication completed successfully', {
                userId: callbackResult.user.id,
                username: callbackResult.user.username,
                roles: callbackResult.user.roles,
            });

            // Clean up
            this.cleanup();
            this.clearPersistedState();

            return callbackResult;

        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : 'Plex authentication failed';

            this.audit('PLEX_AUTH_FAILED', 'error', 'Authentication failed', {
                error: errorMessage,
                pinId: this.state.pinId,
                pollAttempts: this.state.pollAttempts,
            });

            this.state.status = 'error';
            config.onStatusChange?.('error');
            config.onError?.(error instanceof Error ? error : new Error(errorMessage));

            this.cleanup();
            this.clearPersistedState();
            throw error;
        }
    }

    /**
     * Request a PIN from the backend.
     */
    private async requestPIN(): Promise<PlexPINResponse> {
        this.audit('PLEX_PIN_REQUESTED', 'info', 'Requesting PIN from backend');

        const response = await this.fetchClient.fetch('/api/auth/plex/login', {
            method: 'GET',
            signal: this.state.abortController!.signal,
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Failed to request PIN: ${response.status} ${errorText}`);
        }

        return response.json();
    }

    /**
     * Open Plex auth page in a popup window.
     */
    private openAuthPopup(authUrl: string): void {
        // Calculate popup position (center of screen)
        const width = 600;
        const height = 700;
        const screenWidth = typeof screen !== 'undefined' ? screen.width : 1920;
        const screenHeight = typeof screen !== 'undefined' ? screen.height : 1080;
        const left = (screenWidth - width) / 2;
        const top = (screenHeight - height) / 2;

        const features = [
            `width=${width}`,
            `height=${height}`,
            `left=${left}`,
            `top=${top}`,
            'toolbar=no',
            'menubar=no',
            'location=yes',
            'status=no',
            'scrollbars=yes',
            'resizable=yes',
        ].join(',');

        this.state.popup = this.windowManager.open(authUrl, 'plex-auth', features);

        if (!this.state.popup) {
            this.audit('PLEX_POPUP_BLOCKED', 'warn', 'Popup was blocked, falling back to redirect');
            // Popup was blocked, fall back to redirect
            this.windowManager.setLocationHref(authUrl);
        } else {
            this.audit('PLEX_POPUP_OPENED', 'info', 'Auth popup opened', {
                width,
                height,
            });
        }
    }

    /**
     * Poll the backend to check if the PIN has been authorized.
     * Uses exponential backoff to reduce server load.
     */
    private async pollForAuthorization(
        pinId: number,
        config: Required<Omit<PlexPINConfig, 'onSuccess' | 'onError' | 'onStatusChange'>>,
        onStatusChange?: (status: PlexAuthStatus) => void
    ): Promise<void> {
        this.audit('PLEX_POLL_STARTED', 'info', 'Starting authorization polling', {
            pinId,
            initialInterval: config.initialPollInterval,
            maxInterval: config.maxPollInterval,
            backoffMultiplier: config.backoffMultiplier,
        });

        return new Promise((resolve, reject) => {
            const poll = async () => {
                if (this.state.abortController?.signal.aborted) {
                    reject(new Error('Authentication cancelled'));
                    return;
                }

                // Check if popup was closed
                if (this.state.popup && this.state.popup.closed) {
                    this.audit('PLEX_POPUP_CLOSED', 'warn', 'User closed the auth popup');
                    reject(new Error('Authentication window was closed'));
                    return;
                }

                this.state.pollAttempts++;

                try {
                    onStatusChange?.('checking_authorization');

                    this.audit('PLEX_POLL_CHECK', 'info', 'Checking authorization status', {
                        pinId,
                        attempt: this.state.pollAttempts,
                        interval: this.state.currentPollInterval,
                    });

                    const response = await this.fetchClient.fetch(`/api/auth/plex/poll?pin_id=${pinId}`, {
                        method: 'GET',
                        signal: this.state.abortController!.signal,
                    });

                    if (!response.ok) {
                        if (response.status === 404) {
                            this.audit('PLEX_POLL_ERROR', 'error', 'PIN expired or not found');
                            reject(new Error('PIN expired. Please try again.'));
                            return;
                        }
                        throw new Error(`Poll failed: ${response.status}`);
                    }

                    const data: PlexPollResponse = await response.json();

                    if (data.authorized) {
                        this.audit('PLEX_POLL_AUTHORIZED', 'info', 'PIN has been authorized', {
                            pinId,
                            totalAttempts: this.state.pollAttempts,
                        });

                        // Close popup if still open
                        if (this.state.popup && !this.state.popup.closed) {
                            this.state.popup.close();
                        }
                        resolve();
                        return;
                    }

                    // Not yet authorized, schedule next poll with backoff
                    this.audit('PLEX_POLL_PENDING', 'info', 'PIN not yet authorized, scheduling retry', {
                        nextInterval: this.state.currentPollInterval,
                    });

                    onStatusChange?.('waiting_for_user');

                    // Calculate next interval with exponential backoff
                    const nextInterval = this.calculateNextPollInterval(config);
                    this.state.currentPollInterval = nextInterval;

                    this.state.pollTimer = this.windowManager.setTimeout(poll, this.state.currentPollInterval);

                } catch (error) {
                    if (this.state.abortController?.signal.aborted) {
                        reject(new Error('Authentication cancelled'));
                        return;
                    }

                    // Network errors - retry with backoff
                    this.audit('PLEX_POLL_RETRY', 'warn', 'Poll error, retrying with backoff', {
                        error: error instanceof Error ? error.message : 'Unknown error',
                        nextInterval: this.state.currentPollInterval,
                    });

                    const nextInterval = this.calculateNextPollInterval(config);
                    this.state.currentPollInterval = nextInterval;

                    this.state.pollTimer = this.windowManager.setTimeout(poll, this.state.currentPollInterval);
                }
            };

            // Start polling
            poll();
        });
    }

    /**
     * Complete authentication after PIN is authorized.
     * Includes CSRF token if available for server-side validation.
     */
    private async completeAuthentication(pinId: number): Promise<PlexCallbackResponse> {
        // Build request body with CSRF token if available
        const requestBody: Record<string, unknown> = { pin_id: pinId };

        if (this.state.csrfToken) {
            requestBody.csrf_token = this.state.csrfToken;
            this.audit('PLEX_CSRF_VALIDATION', 'info', 'Including CSRF token in callback');
        }

        const response = await this.fetchClient.fetch('/api/auth/plex/callback', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(requestBody),
            signal: this.state.abortController!.signal,
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Failed to complete authentication: ${response.status} ${errorText}`);
        }

        return response.json();
    }

    /**
     * Cancel an ongoing Plex authentication flow.
     *
     * Call this if the user clicks "Cancel" or navigates away.
     * This also clears any persisted state.
     *
     * @param clearPersisted - Whether to clear persisted state (default: true)
     */
    cancel(clearPersisted: boolean = true): void {
        if (this.state.status === 'idle') {
            return;
        }

        this.audit('PLEX_AUTH_CANCELLED', 'info', 'Authentication cancelled by user', {
            status: this.state.status,
            pinId: this.state.pinId,
            pollAttempts: this.state.pollAttempts,
            clearPersisted,
        });

        this.state.status = 'cancelled';

        // Abort any in-flight requests
        if (this.state.abortController) {
            this.state.abortController.abort();
        }

        this.cleanup();

        if (clearPersisted) {
            this.clearPersistedState();
        }
    }

    /**
     * Clean up auth state and timers.
     */
    private cleanup(): void {
        // Clear timers
        if (this.state.pollTimer) {
            this.windowManager.clearTimeout(this.state.pollTimer);
            this.state.pollTimer = null;
        }
        if (this.state.timeoutTimer) {
            this.windowManager.clearTimeout(this.state.timeoutTimer);
            this.state.timeoutTimer = null;
        }

        // Close popup if open
        if (this.state.popup && !this.state.popup.closed) {
            this.state.popup.close();
        }
        this.state.popup = null;

        // Clear abort controller
        this.state.abortController = null;
    }

    /**
     * Get the current status of the authentication flow.
     */
    getStatus(): PlexAuthStatus {
        return this.state.status;
    }

    /**
     * Get the current correlation ID for the auth flow.
     */
    getCorrelationId(): string {
        return this.state.correlationId;
    }

    /**
     * Check if authentication is currently in progress.
     */
    isInProgress(): boolean {
        return !['idle', 'success', 'error', 'timeout', 'cancelled'].includes(this.state.status);
    }
}

// ============================================================================
// Convenience Functions (for backwards compatibility)
// ============================================================================

// Default singleton instance for simple usage
let defaultAuthenticator: PlexPINAuthenticator | null = null;

/**
 * Gets or creates the default authenticator instance.
 * Use PlexPINAuthenticator directly for better testability.
 */
function getDefaultAuthenticator(): PlexPINAuthenticator {
    if (!defaultAuthenticator) {
        defaultAuthenticator = new PlexPINAuthenticator();
    }
    return defaultAuthenticator;
}

/**
 * Initiates Plex PIN-based authentication using the default authenticator.
 *
 * @param config - Optional configuration
 * @returns Promise that resolves when authentication completes
 *
 * @deprecated Use PlexPINAuthenticator directly for better testability
 *
 * @example
 * ```typescript
 * // Simple usage (convenience function)
 * await initiatePlexPINAuth({
 *     onStatusChange: (status) => updateUI(status),
 *     onSuccess: (user) => console.log('Logged in as', user.username),
 * });
 *
 * // Recommended: Use class directly
 * const authenticator = new PlexPINAuthenticator();
 * await authenticator.authenticate(config);
 * ```
 */
export async function initiatePlexPINAuth(config: PlexPINConfig = {}): Promise<PlexCallbackResponse> {
    return getDefaultAuthenticator().authenticate(config);
}

/**
 * Cancel an ongoing Plex authentication flow using the default authenticator.
 *
 * @deprecated Use PlexPINAuthenticator.cancel() directly for better testability
 */
export function cancelPlexAuth(): void {
    if (defaultAuthenticator) {
        defaultAuthenticator.cancel();
    }
}

/**
 * Get the current status of the authentication flow.
 *
 * @deprecated Use PlexPINAuthenticator.getStatus() directly for better testability
 */
export function getPlexAuthStatus(): PlexAuthStatus {
    return defaultAuthenticator?.getStatus() ?? 'idle';
}

/**
 * Check if authentication is currently in progress.
 *
 * @deprecated Use PlexPINAuthenticator.isInProgress() directly for better testability
 */
export function isPlexAuthInProgress(): boolean {
    return defaultAuthenticator?.isInProgress() ?? false;
}

/**
 * Reset the default authenticator instance.
 * Useful for testing or when you need a fresh state.
 */
export function resetDefaultAuthenticator(): void {
    if (defaultAuthenticator) {
        defaultAuthenticator.cancel();
        defaultAuthenticator = null;
    }
}
