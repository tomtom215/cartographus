// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Cross-Platform API Module
 *
 * Content mapping, user linking, and cross-platform analytics (Phase 3).
 */

import type {
    ContentMappingRequest,
    ContentMappingResponse,
    ContentLookupParams,
    UserLinkRequest,
    UserLinkResponse,
    LinkedUsersResponse,
    SuggestedLinksResponse,
    CrossPlatformUserStatsResponse,
    CrossPlatformContentStatsResponse,
    CrossPlatformSummaryResponse,
} from '../types/cross-platform';
import { BaseAPIClient } from './client';

/**
 * Cross-Platform API methods
 */
export class CrossPlatformAPI extends BaseAPIClient {
    // ========================================================================
    // Content Mapping
    // ========================================================================

    async createContentMapping(request: ContentMappingRequest): Promise<ContentMappingResponse> {
        const response = await this.fetch<ContentMappingResponse>('/content/link', {
            method: 'POST',
            body: JSON.stringify(request),
        });
        return response.data;
    }

    async lookupContentMapping(params: ContentLookupParams): Promise<ContentMappingResponse> {
        const urlParams = new URLSearchParams();
        if (params.imdb_id) urlParams.append('imdb_id', params.imdb_id);
        if (params.tmdb_id !== undefined) urlParams.append('tmdb_id', params.tmdb_id.toString());
        if (params.tvdb_id !== undefined) urlParams.append('tvdb_id', params.tvdb_id.toString());
        if (params.plex_rating_key) urlParams.append('plex_rating_key', params.plex_rating_key);
        if (params.jellyfin_item_id) urlParams.append('jellyfin_item_id', params.jellyfin_item_id);
        if (params.emby_item_id) urlParams.append('emby_item_id', params.emby_item_id);

        const response = await this.fetch<ContentMappingResponse>(
            `/content/lookup?${urlParams.toString()}`
        );
        return response.data;
    }

    async linkContentToPlex(contentId: number, plexRatingKey: string): Promise<ContentMappingResponse> {
        const response = await this.fetch<ContentMappingResponse>(
            `/content/${contentId}/link/plex`,
            {
                method: 'POST',
                body: JSON.stringify({ plex_rating_key: plexRatingKey }),
            }
        );
        return response.data;
    }

    async linkContentToJellyfin(contentId: number, jellyfinItemId: string): Promise<ContentMappingResponse> {
        const response = await this.fetch<ContentMappingResponse>(
            `/content/${contentId}/link/jellyfin`,
            {
                method: 'POST',
                body: JSON.stringify({ jellyfin_item_id: jellyfinItemId }),
            }
        );
        return response.data;
    }

    async linkContentToEmby(contentId: number, embyItemId: string): Promise<ContentMappingResponse> {
        const response = await this.fetch<ContentMappingResponse>(
            `/content/${contentId}/link/emby`,
            {
                method: 'POST',
                body: JSON.stringify({ emby_item_id: embyItemId }),
            }
        );
        return response.data;
    }

    // ========================================================================
    // User Linking
    // ========================================================================

    async createUserLink(request: UserLinkRequest): Promise<UserLinkResponse> {
        const response = await this.fetch<UserLinkResponse>('/users/link', {
            method: 'POST',
            body: JSON.stringify(request),
        });
        return response.data;
    }

    async deleteUserLink(primaryUserId: number, linkedUserId: number): Promise<void> {
        await this.fetch('/users/link', {
            method: 'DELETE',
            body: JSON.stringify({
                primary_user_id: primaryUserId,
                linked_user_id: linkedUserId,
            }),
        });
    }

    async getSuggestedUserLinks(): Promise<SuggestedLinksResponse> {
        const response = await this.fetch<SuggestedLinksResponse>('/users/suggest-links');
        return response.data;
    }

    async getLinkedUsers(userId: number): Promise<LinkedUsersResponse> {
        const response = await this.fetch<LinkedUsersResponse>(`/users/${userId}/linked`);
        return response.data;
    }

    // ========================================================================
    // Cross-Platform Analytics
    // ========================================================================

    async getCrossPlatformUserStats(userId: number): Promise<CrossPlatformUserStatsResponse> {
        const response = await this.fetch<CrossPlatformUserStatsResponse>(
            `/analytics/cross-platform/user/${userId}`
        );
        return response.data;
    }

    async getCrossPlatformContentStats(contentId: number): Promise<CrossPlatformContentStatsResponse> {
        const response = await this.fetch<CrossPlatformContentStatsResponse>(
            `/analytics/cross-platform/content/${contentId}`
        );
        return response.data;
    }

    async getCrossPlatformSummary(): Promise<CrossPlatformSummaryResponse> {
        const response = await this.fetch<CrossPlatformSummaryResponse>(
            '/analytics/cross-platform/summary'
        );
        return response.data;
    }
}
