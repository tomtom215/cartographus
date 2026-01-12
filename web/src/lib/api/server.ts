// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Server Status API Module
 *
 * API client for media server management (ADR-0026 Phases 1-4).
 * Supports both read-only env-var servers and CRUD for UI-managed servers.
 */

import type {
    MediaServerListResponse,
    MediaServerTestRequest,
    MediaServerTestResponse,
    CreateMediaServerRequest,
    UpdateMediaServerRequest,
    MediaServerResponse,
} from '../types/server';
import { BaseAPIClient } from './client';

/**
 * Server Status API methods
 */
export class ServerAPI extends BaseAPIClient {
    // ========================================================================
    // Server Status (Read-Only)
    // ========================================================================

    /**
     * Get status of all configured media servers.
     * Returns servers from environment variables with connection status.
     * Requires admin role.
     */
    async listServers(): Promise<MediaServerListResponse> {
        return this.fetchSimple<MediaServerListResponse>('/admin/servers');
    }

    /**
     * Test connectivity to a media server.
     * Can be used to verify server configuration before saving.
     * Requires admin role.
     */
    async testConnection(request: MediaServerTestRequest): Promise<MediaServerTestResponse> {
        const response = await this.fetch<MediaServerTestResponse>('/admin/servers/test', {
            method: 'POST',
            body: JSON.stringify(request)
        });
        return response.data;
    }

    /**
     * Trigger manual sync for a specific server.
     * Requires admin role.
     */
    async triggerSync(serverId: string, fullSync: boolean = false): Promise<{ success: boolean; message: string }> {
        const response = await this.fetch<{ success: boolean; message: string }>(
            `/admin/servers/${encodeURIComponent(serverId)}/sync`,
            {
                method: 'POST',
                body: JSON.stringify({ full_sync: fullSync })
            }
        );
        return response.data;
    }

    // ========================================================================
    // Utility Methods
    // ========================================================================

    /**
     * Get count of servers by status.
     */
    async getServerCounts(): Promise<{
        total: number;
        connected: number;
        syncing: number;
        error: number;
        disabled: number;
    }> {
        const response = await this.listServers();
        const disabled = response.servers.filter(s => !s.enabled).length;

        return {
            total: response.total_count,
            connected: response.connected_count,
            syncing: response.syncing_count,
            error: response.error_count,
            disabled
        };
    }

    /**
     * Check if any server has errors.
     */
    async hasServerErrors(): Promise<boolean> {
        const response = await this.listServers();
        return response.error_count > 0;
    }

    /**
     * Get servers that need attention (errors or disconnected).
     */
    async getServersNeedingAttention(): Promise<MediaServerListResponse['servers']> {
        const response = await this.listServers();
        return response.servers.filter(s =>
            s.status === 'error' || s.status === 'disconnected'
        );
    }

    // ========================================================================
    // Server CRUD Operations (ADR-0026 Phase 4)
    // ========================================================================

    /**
     * Create a new media server configuration.
     * The server will be stored in the database (source: 'ui').
     * Requires admin role.
     *
     * @param request - Server configuration details
     * @returns The created server with masked credentials
     */
    async createServer(request: CreateMediaServerRequest): Promise<MediaServerResponse> {
        const response = await this.fetch<MediaServerResponse>('/admin/servers', {
            method: 'POST',
            body: JSON.stringify(request)
        });
        return response.data;
    }

    /**
     * Get a specific server by ID.
     * Requires admin role.
     *
     * @param serverId - The server's unique identifier
     * @returns Server details with masked credentials
     */
    async getServer(serverId: string): Promise<MediaServerResponse> {
        const response = await this.fetch<MediaServerResponse>(
            `/admin/servers/${encodeURIComponent(serverId)}`
        );
        return response.data;
    }

    /**
     * Update an existing server configuration.
     * Only UI-managed servers (source: 'ui') can be updated.
     * Env-var servers are immutable.
     * Requires admin role.
     *
     * @param serverId - The server's unique identifier
     * @param request - Fields to update (partial update supported)
     * @returns The updated server with masked credentials
     */
    async updateServer(serverId: string, request: UpdateMediaServerRequest): Promise<MediaServerResponse> {
        const response = await this.fetch<MediaServerResponse>(
            `/admin/servers/${encodeURIComponent(serverId)}`,
            {
                method: 'PUT',
                body: JSON.stringify(request)
            }
        );
        return response.data;
    }

    /**
     * Delete a server configuration.
     * Only UI-managed servers (source: 'ui') can be deleted.
     * Env-var servers cannot be deleted.
     * Requires admin role.
     *
     * @param serverId - The server's unique identifier
     */
    async deleteServer(serverId: string): Promise<void> {
        await this.fetch<{ message: string }>(
            `/admin/servers/${encodeURIComponent(serverId)}`,
            { method: 'DELETE' }
        );
    }

    /**
     * List only database-stored servers (excludes env-var servers).
     * Useful for admin UI that manages only UI-added servers.
     * Requires admin role.
     *
     * @param platform - Optional: filter by platform type
     * @param enabledOnly - Optional: filter to only enabled servers
     * @returns List of database-stored servers
     */
    async listDBServers(platform?: string, enabledOnly?: boolean): Promise<MediaServerResponse[]> {
        const params = new URLSearchParams();
        if (platform) params.append('platform', platform);
        if (enabledOnly) params.append('enabled', 'true');

        const queryString = params.toString();
        const url = queryString ? `/admin/servers/db?${queryString}` : '/admin/servers/db';

        const response = await this.fetch<MediaServerResponse[]>(url);
        return response.data;
    }

    /**
     * Toggle server enabled/disabled state.
     * Convenience method that calls updateServer with only the enabled field.
     *
     * @param serverId - The server's unique identifier
     * @param enabled - New enabled state
     * @returns The updated server
     */
    async setServerEnabled(serverId: string, enabled: boolean): Promise<MediaServerResponse> {
        return this.updateServer(serverId, { enabled });
    }
}
