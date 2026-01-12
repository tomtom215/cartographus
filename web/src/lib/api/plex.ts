// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Plex Direct API Module
 *
 * Direct Plex server API integration.
 * Includes Friends and Library Sharing management.
 */

import type {
    PlexServerIdentity,
    PlexLibrarySection,
    PlexMediaMetadata,
    PlexDirectSession,
    PlexDevice,
    PlexAccount,
    PlexActivity,
    PlexPlaylist,
    PlexCapabilities,
    PlexBandwidthStatistics,
    PlexMediaContainer,
    PlexFriendsListResponse,
    PlexInviteFriendRequest,
    PlexSharedServersListResponse,
    PlexShareLibrariesRequest,
    PlexUpdateSharingRequest,
    PlexManagedUsersListResponse,
    PlexManagedUser,
    PlexCreateManagedUserRequest,
    PlexUpdateManagedUserRequest,
    PlexLibrarySectionsListResponse,
} from '../types/plex';
import { BaseAPIClient } from './client';

/**
 * Plex Direct API methods
 */
export class PlexAPI extends BaseAPIClient {
    async getPlexIdentity(): Promise<PlexServerIdentity> {
        const response = await this.fetch<PlexMediaContainer<PlexServerIdentity>>('/plex/identity');
        return response.data.MediaContainer as unknown as PlexServerIdentity;
    }

    async getPlexLibrarySections(): Promise<PlexLibrarySection[]> {
        const response = await this.fetch<PlexMediaContainer<PlexLibrarySection>>('/plex/library/sections');
        return response.data.MediaContainer.Directory || [];
    }

    async getPlexLibraryAll(sectionKey: string): Promise<PlexMediaMetadata[]> {
        const response = await this.fetch<PlexMediaContainer<PlexMediaMetadata>>(
            `/plex/library/sections/${sectionKey}/all`
        );
        return response.data.MediaContainer.Metadata || [];
    }

    async getPlexLibraryRecentlyAdded(sectionKey: string): Promise<PlexMediaMetadata[]> {
        const response = await this.fetch<PlexMediaContainer<PlexMediaMetadata>>(
            `/plex/library/sections/${sectionKey}/recentlyAdded`
        );
        return response.data.MediaContainer.Metadata || [];
    }

    async getPlexLibrarySearch(sectionKey: string, query: string): Promise<PlexMediaMetadata[]> {
        const encodedQuery = encodeURIComponent(query);
        const response = await this.fetch<PlexMediaContainer<PlexMediaMetadata>>(
            `/plex/library/sections/${sectionKey}/search?query=${encodedQuery}`
        );
        return response.data.MediaContainer.Metadata || [];
    }

    async getPlexMetadata(ratingKey: string): Promise<PlexMediaMetadata | null> {
        const response = await this.fetch<PlexMediaContainer<PlexMediaMetadata>>(
            `/plex/library/metadata/${ratingKey}`
        );
        const metadata = response.data.MediaContainer.Metadata;
        return metadata && metadata.length > 0 ? metadata[0] : null;
    }

    async getPlexOnDeck(): Promise<PlexMediaMetadata[]> {
        const response = await this.fetch<PlexMediaContainer<PlexMediaMetadata>>('/plex/library/onDeck');
        return response.data.MediaContainer.Metadata || [];
    }

    async getPlexSessions(): Promise<PlexDirectSession[]> {
        const response = await this.fetch<PlexMediaContainer<PlexDirectSession>>('/plex/sessions');
        return response.data.MediaContainer.Metadata || [];
    }

    async getPlexActivities(): Promise<PlexActivity[]> {
        const response = await this.fetch<PlexMediaContainer<PlexActivity>>('/plex/activities');
        return response.data.MediaContainer.Activity || [];
    }

    async getPlexDevices(): Promise<PlexDevice[]> {
        const response = await this.fetch<PlexMediaContainer<PlexDevice>>('/plex/devices');
        return response.data.MediaContainer.Device || [];
    }

    async getPlexAccounts(): Promise<PlexAccount[]> {
        const response = await this.fetch<PlexMediaContainer<PlexAccount>>('/plex/accounts');
        return response.data.MediaContainer.Account || [];
    }

    async getPlexPlaylists(): Promise<PlexPlaylist[]> {
        const response = await this.fetch<PlexMediaContainer<PlexPlaylist>>('/plex/playlists');
        return response.data.MediaContainer.Metadata || [];
    }

    async getPlexCapabilities(): Promise<PlexCapabilities> {
        const response = await this.fetch<PlexMediaContainer<PlexCapabilities>>('/plex/capabilities');
        return response.data.MediaContainer as unknown as PlexCapabilities;
    }

    async getPlexBandwidthStatistics(): Promise<PlexBandwidthStatistics[]> {
        const response = await this.fetch<PlexMediaContainer<PlexBandwidthStatistics>>(
            '/plex/statistics/bandwidth'
        );
        return response.data.MediaContainer.StatisticsBandwidth || [];
    }

    async getPlexTranscodeSessions(): Promise<PlexDirectSession[]> {
        const response = await this.fetch<PlexMediaContainer<PlexDirectSession>>('/plex/transcode/sessions');
        return response.data.MediaContainer.Metadata || [];
    }

    async killPlexTranscodeSession(sessionKey: string): Promise<void> {
        await this.fetch<void>(`/plex/transcode/sessions/${sessionKey}`, {
            method: 'DELETE'
        });
    }

    // ========================================================================
    // Plex Friends and Sharing (Library Access Management)
    // ========================================================================

    /**
     * Get list of Plex friends
     */
    async getPlexFriends(): Promise<PlexFriendsListResponse> {
        const response = await this.fetch<PlexFriendsListResponse>('/plex/friends');
        return response.data;
    }

    /**
     * Invite a friend via email
     */
    async invitePlexFriend(request: PlexInviteFriendRequest): Promise<void> {
        await this.fetch<void>('/plex/friends/invite', {
            method: 'POST',
            body: JSON.stringify(request),
        });
    }

    /**
     * Remove a friend by ID
     */
    async removePlexFriend(friendId: number): Promise<void> {
        await this.fetch<void>(`/plex/friends/${friendId}`, {
            method: 'DELETE',
        });
    }

    /**
     * Get list of shared servers
     */
    async getPlexSharedServers(): Promise<PlexSharedServersListResponse> {
        const response = await this.fetch<PlexSharedServersListResponse>('/plex/sharing');
        return response.data;
    }

    /**
     * Share libraries with a user
     */
    async sharePlexLibraries(request: PlexShareLibrariesRequest): Promise<void> {
        await this.fetch<void>('/plex/sharing', {
            method: 'POST',
            body: JSON.stringify(request),
        });
    }

    /**
     * Update sharing settings for a user
     */
    async updatePlexSharing(sharedServerId: number, request: PlexUpdateSharingRequest): Promise<void> {
        await this.fetch<void>(`/plex/sharing/${sharedServerId}`, {
            method: 'PUT',
            body: JSON.stringify(request),
        });
    }

    /**
     * Revoke sharing for a user
     */
    async revokePlexSharing(sharedServerId: number): Promise<void> {
        await this.fetch<void>(`/plex/sharing/${sharedServerId}`, {
            method: 'DELETE',
        });
    }

    /**
     * Get list of managed users (Plex Home)
     */
    async getPlexManagedUsers(): Promise<PlexManagedUsersListResponse> {
        const response = await this.fetch<PlexManagedUsersListResponse>('/plex/home/users');
        return response.data;
    }

    /**
     * Create a managed user
     */
    async createPlexManagedUser(request: PlexCreateManagedUserRequest): Promise<PlexManagedUser> {
        const response = await this.fetch<PlexManagedUser>('/plex/home/users', {
            method: 'POST',
            body: JSON.stringify(request),
        });
        return response.data;
    }

    /**
     * Delete a managed user
     */
    async deletePlexManagedUser(userId: number): Promise<void> {
        await this.fetch<void>(`/plex/home/users/${userId}`, {
            method: 'DELETE',
        });
    }

    /**
     * Update managed user restrictions
     */
    async updatePlexManagedUser(userId: number, request: PlexUpdateManagedUserRequest): Promise<void> {
        await this.fetch<void>(`/plex/home/users/${userId}`, {
            method: 'PUT',
            body: JSON.stringify(request),
        });
    }

    /**
     * Get library sections for sharing UI
     */
    async getPlexLibrariesForSharing(): Promise<PlexLibrarySectionsListResponse> {
        const response = await this.fetch<PlexLibrarySectionsListResponse>('/plex/libraries');
        return response.data;
    }
}
