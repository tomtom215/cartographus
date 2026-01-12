// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Cross-Platform Types for Multi-Server Analytics
 *
 * Types for content mapping and user linking across Plex, Jellyfin, and Emby.
 * These enable unified analytics across different media server platforms.
 *
 * Reference: Phase 3 PRR - Cross-Platform Content Reconciliation
 */

// ========================================
// Content Mapping Types
// ========================================

/**
 * ContentMapping represents a mapping of content IDs across platforms
 */
export interface ContentMapping {
    id: number;
    imdb_id?: string;              // IMDb ID (e.g., tt1234567)
    tmdb_id?: number;              // TMDB movie/show ID
    tvdb_id?: number;              // TVDB series ID
    plex_rating_key?: string;      // Plex rating_key
    jellyfin_item_id?: string;     // Jellyfin Item.Id (UUID)
    emby_item_id?: string;         // Emby Item.Id
    title: string;
    media_type: 'movie' | 'show' | 'episode';
    year?: number;
    created_at: string;
    updated_at: string;
}

/**
 * Request to create or update a content mapping
 */
export interface ContentMappingRequest {
    imdb_id?: string;              // At least one external ID required
    tmdb_id?: number;
    tvdb_id?: number;
    plex_rating_key?: string;      // Optional platform-specific IDs
    jellyfin_item_id?: string;
    emby_item_id?: string;
    title: string;                 // Required
    media_type: 'movie' | 'show' | 'episode';  // Required
    year?: number;
}

/**
 * Response from content mapping API
 */
export interface ContentMappingResponse {
    success: boolean;
    message?: string;
    data?: ContentMapping;
    created?: boolean;
}

/**
 * Content lookup query parameters
 */
export interface ContentLookupParams {
    imdb_id?: string;
    tmdb_id?: number;
    tvdb_id?: number;
    plex_rating_key?: string;
    jellyfin_item_id?: string;
    emby_item_id?: string;
}

// ========================================
// User Linking Types
// ========================================

/**
 * UserLink represents a link between users across platforms
 */
export interface UserLink {
    id: number;
    primary_user_id: number;       // Primary user internal ID
    linked_user_id: number;        // Linked user internal ID
    link_type: 'manual' | 'email' | 'plex_home';
    confidence: number;            // 0.0-1.0, 1.0 = certain
    created_by?: string;
    created_at: string;
}

/**
 * Request to create a user link
 */
export interface UserLinkRequest {
    primary_user_id: number;
    linked_user_id: number;
    link_type: 'manual' | 'email' | 'plex_home';
}

/**
 * Response from user link API
 */
export interface UserLinkResponse {
    success: boolean;
    message?: string;
    data?: UserLink;
}

/**
 * Information about a linked user
 */
export interface LinkedUserInfo {
    internal_user_id: number;
    username?: string;
    friendly_name?: string;
    source: string;                // 'plex', 'jellyfin', 'emby', 'tautulli'
    server_id: string;
    link_type?: string;
}

/**
 * Response containing linked users
 */
export interface LinkedUsersResponse {
    success: boolean;
    user_ids?: number[];
    users?: LinkedUserInfo[];
}

/**
 * Response containing suggested user links
 */
export interface SuggestedLinksResponse {
    success: boolean;
    suggestions: Record<string, LinkedUserInfo[]>;  // email -> users with that email
}

// ========================================
// Cross-Platform Analytics Types
// ========================================

/**
 * Cross-platform user statistics
 */
export interface CrossPlatformUserStats {
    internal_user_id: number;
    linked_users: LinkedUserInfo[];
    total_plays: number;
    total_duration: number;
    platforms_used: string[];
    last_played: string;
    by_platform: {
        platform: string;
        plays: number;
        duration: number;
    }[];
}

/**
 * Cross-platform content statistics
 */
export interface CrossPlatformContentStats {
    content_id: number;
    title: string;
    media_type: string;
    external_ids: {
        imdb_id?: string;
        tmdb_id?: number;
        tvdb_id?: number;
    };
    total_plays: number;
    total_users: number;
    by_platform: {
        platform: string;
        plays: number;
        unique_users: number;
    }[];
}

/**
 * Cross-platform summary statistics
 */
export interface CrossPlatformSummary {
    total_content_mappings: number;
    total_user_links: number;
    platforms: {
        name: string;
        users: number;
        content_items: number;
    }[];
    link_coverage: {
        users_with_links: number;
        total_users: number;
        percentage: number;
    };
    content_coverage: {
        mapped_content: number;
        total_content: number;
        percentage: number;
    };
}

/**
 * Cross-platform user stats response
 */
export interface CrossPlatformUserStatsResponse {
    success: boolean;
    data?: CrossPlatformUserStats;
    error?: string;
}

/**
 * Cross-platform content stats response
 */
export interface CrossPlatformContentStatsResponse {
    success: boolean;
    data?: CrossPlatformContentStats;
    error?: string;
}

/**
 * Cross-platform summary response
 */
export interface CrossPlatformSummaryResponse {
    success: boolean;
    data?: CrossPlatformSummary;
    error?: string;
}
