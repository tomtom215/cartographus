// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Server Status Types
 *
 * Types for media server status display (ADR-0026 Phase 1).
 * Read-only view of servers configured via environment variables.
 */

/**
 * Platform types for media servers.
 */
export type MediaServerPlatform = 'plex' | 'jellyfin' | 'emby' | 'tautulli';

/**
 * Server connection status.
 */
export type MediaServerStatusType =
    | 'connected'      // Successfully connected to server
    | 'disconnected'   // Not currently connected
    | 'syncing'        // Currently syncing data
    | 'error'          // Connection error
    | 'disabled'       // Server is disabled
    | 'configured'     // Configured but not tested
    | 'unknown';       // Status cannot be determined

/**
 * Configuration source for a server.
 */
export type ServerSource = 'env' | 'ui' | 'import';

/**
 * Status information for a single media server.
 */
export interface MediaServerStatus {
    /** Unique identifier for the server */
    id: string;
    /** Server platform type */
    platform: MediaServerPlatform;
    /** Display name for the server */
    name: string;
    /** Server URL (masked for security) */
    url: string;
    /** Whether the server is enabled */
    enabled: boolean;
    /** Configuration source */
    source: ServerSource;
    /** Current connection status */
    status: MediaServerStatusType;
    /** Whether real-time updates are enabled */
    realtime_enabled: boolean;
    /** Whether webhooks are enabled */
    webhooks_enabled: boolean;
    /** Whether session polling is enabled */
    session_polling_enabled: boolean;
    /** Timestamp of last successful sync */
    last_sync_at?: string;
    /** Status of last sync operation */
    last_sync_status?: string;
    /** Last error message if status is 'error' */
    last_error?: string;
    /** Timestamp of last error */
    last_error_at?: string;
    /** Server software version if available */
    server_version?: string;
    /** Whether this server can be edited in UI (false for env-var servers) */
    immutable: boolean;
}

/**
 * Response from the server status list endpoint.
 */
export interface MediaServerListResponse {
    /** List of all configured servers */
    servers: MediaServerStatus[];
    /** Total number of servers */
    total_count: number;
    /** Number of connected servers */
    connected_count: number;
    /** Number of currently syncing servers */
    syncing_count: number;
    /** Number of servers with errors */
    error_count: number;
    /** Timestamp when status was checked */
    last_checked: string;
}

/**
 * Request to test server connectivity.
 */
export interface MediaServerTestRequest {
    platform: MediaServerPlatform;
    url: string;
    token: string;
}

/**
 * Response from server connectivity test.
 */
export interface MediaServerTestResponse {
    success: boolean;
    latency_ms: number;
    server_name?: string;
    version?: string;
    error?: string;
    error_code?: string;
}

/**
 * Platform display information.
 */
export interface PlatformInfo {
    name: string;
    icon: string;
    color: string;
    description: string;
}

/**
 * Platform display configuration.
 */
export const PLATFORM_INFO: Record<MediaServerPlatform, PlatformInfo> = {
    plex: {
        name: 'Plex',
        icon: 'plex',
        color: '#E5A00D',
        description: 'Plex Media Server'
    },
    jellyfin: {
        name: 'Jellyfin',
        icon: 'jellyfin',
        color: '#00A4DC',
        description: 'Jellyfin Media Server'
    },
    emby: {
        name: 'Emby',
        icon: 'emby',
        color: '#52B54B',
        description: 'Emby Media Server'
    },
    tautulli: {
        name: 'Tautulli',
        icon: 'tautulli',
        color: '#FF6B35',
        description: 'Tautulli Analytics for Plex'
    }
};

/**
 * Status display information.
 */
export interface StatusInfo {
    label: string;
    color: string;
    icon: string;
}

/**
 * Status display configuration.
 */
export const STATUS_INFO: Record<MediaServerStatusType, StatusInfo> = {
    connected: {
        label: 'Connected',
        color: '#10B981',
        icon: 'check-circle'
    },
    disconnected: {
        label: 'Disconnected',
        color: '#6B7280',
        icon: 'x-circle'
    },
    syncing: {
        label: 'Syncing',
        color: '#3B82F6',
        icon: 'refresh'
    },
    error: {
        label: 'Error',
        color: '#EF4444',
        icon: 'exclamation-circle'
    },
    disabled: {
        label: 'Disabled',
        color: '#9CA3AF',
        icon: 'pause-circle'
    },
    configured: {
        label: 'Configured',
        color: '#F59E0B',
        icon: 'cog'
    },
    unknown: {
        label: 'Unknown',
        color: '#6B7280',
        icon: 'question-mark-circle'
    }
};

// ============================================================================
// ADR-0026 Phase 4: CRUD Types for UI-managed servers
// ============================================================================

/**
 * Request to create a new media server (added via UI).
 */
export interface CreateMediaServerRequest {
    /** Server platform type */
    platform: MediaServerPlatform;
    /** Display name for the server */
    name: string;
    /** Server URL (e.g., http://localhost:32400) */
    url: string;
    /** API token for authentication */
    token: string;
    /** Whether real-time updates are enabled */
    realtime_enabled?: boolean;
    /** Whether webhooks are enabled */
    webhooks_enabled?: boolean;
    /** Whether session polling is enabled */
    session_polling_enabled?: boolean;
    /** Session polling interval (e.g., "30s", "1m") */
    session_polling_interval?: string;
    /** Platform-specific settings */
    settings?: Record<string, unknown>;
}

/**
 * Request to update an existing media server.
 * All fields are optional - only provided fields will be updated.
 */
export interface UpdateMediaServerRequest {
    /** Display name for the server */
    name?: string;
    /** Server URL */
    url?: string;
    /** API token for authentication */
    token?: string;
    /** Whether the server is enabled */
    enabled?: boolean;
    /** Whether real-time updates are enabled */
    realtime_enabled?: boolean;
    /** Whether webhooks are enabled */
    webhooks_enabled?: boolean;
    /** Whether session polling is enabled */
    session_polling_enabled?: boolean;
    /** Session polling interval (e.g., "30s", "1m") */
    session_polling_interval?: string;
    /** Platform-specific settings */
    settings?: Record<string, unknown>;
}

/**
 * Response from server CRUD operations.
 * Matches the backend MediaServerResponse model.
 */
export interface MediaServerResponse {
    /** Unique identifier */
    id: string;
    /** Server platform type */
    platform: MediaServerPlatform;
    /** Display name */
    name: string;
    /** Server URL (decrypted) */
    url: string;
    /** Masked token (****...last4) */
    token_masked: string;
    /** Server ID for deduplication */
    server_id: string;
    /** Whether enabled */
    enabled: boolean;
    /** Configuration source */
    source: ServerSource;
    /** Whether real-time updates are enabled */
    realtime_enabled: boolean;
    /** Whether webhooks are enabled */
    webhooks_enabled: boolean;
    /** Whether session polling is enabled */
    session_polling_enabled: boolean;
    /** Session polling interval */
    session_polling_interval: string;
    /** Current status */
    status: MediaServerStatusType;
    /** Timestamp of last sync */
    last_sync_at?: string;
    /** Last sync status */
    last_sync_status?: string;
    /** Last error message */
    last_error?: string;
    /** Timestamp of last error */
    last_error_at?: string;
    /** Creation timestamp */
    created_at: string;
    /** Last update timestamp */
    updated_at: string;
    /** Whether this server is immutable (env-var servers) */
    immutable: boolean;
}

/**
 * Request to trigger sync for a server.
 */
export interface SyncTriggerRequest {
    /** Server ID to sync */
    server_id: string;
    /** Whether to do a full historical sync */
    full_sync?: boolean;
}

/**
 * Response from sync trigger.
 */
export interface SyncTriggerResponse {
    success: boolean;
    message: string;
    sync_id?: string;
}
