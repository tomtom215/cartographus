// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Tautulli-specific types for real-time activity, metadata, and library information
 */

// Tautulli Real-Time Activity interfaces
export interface TautulliActivitySession {
    session_key: string;
    user_id: number;
    username: string;
    friendly_name: string;
    title: string;
    parent_title?: string;
    grandparent_title?: string;
    media_type: string;
    year?: number;
    platform: string;
    player: string;
    ip_address: string;
    stream_container: string;
    stream_video_codec: string;
    stream_video_resolution: string;
    stream_audio_codec: string;
    transcode_decision: string;
    bandwidth: number;
    progress_percent: number;
    state: string;
}

export interface TautulliActivityData {
    stream_count: number;
    stream_count_direct_play: number;
    stream_count_direct_stream: number;
    stream_count_transcode: number;
    total_bandwidth: number;
    lan_bandwidth: number;
    wan_bandwidth: number;
    sessions: TautulliActivitySession[];
}

// Tautulli Metadata interfaces
export interface TautulliMetadataData {
    rating_key: string;
    parent_rating_key?: string;
    grandparent_rating_key?: string;
    title: string;
    parent_title?: string;
    grandparent_title?: string;
    media_type: string;
    year?: number;
    summary: string;
    rating?: number;
    audience_rating?: number;
    content_rating?: string;
    duration: number;
    thumb?: string;
    art?: string;
    banner?: string;
    genres: string[];
    directors: string[];
    writers: string[];
    actors: string[];
    studio?: string;
    originally_available_at?: string;
    added_at: number;
    updated_at: number;
}

// Tautulli User interfaces
export interface TautulliUserData {
    user_id: number;
    username: string;
    friendly_name: string;
    email?: string;
    thumb?: string;
    is_admin: number;
    is_home_user: number;
    is_allow_sync: number;
    is_restricted: number;
    is_active: number;
    do_notify: number;
    keep_history: number;
    allow_guest: number;
}

// Tautulli Library User Stats interfaces
export interface TautulliLibraryUserStat {
    user_id: number;
    username: string;
    friendly_name: string;
    total_plays: number;
    total_time: number;
    last_played: number;
}

// Tautulli Recently Added interfaces
export interface TautulliRecentlyAddedItem {
    rating_key: string;
    parent_rating_key?: string;
    grandparent_rating_key?: string;
    title: string;
    parent_title?: string;
    grandparent_title?: string;
    media_type: string;
    year?: number;
    thumb?: string;
    art?: string;
    added_at: number;
    section_id: number;
    library_name: string;
    summary?: string;
}

export interface TautulliRecentlyAddedData {
    records_total: number;
    recently_added: TautulliRecentlyAddedItem[];
}

// Tautulli Libraries interfaces
export interface TautulliLibraryDetail {
    section_id: number;
    section_name: string;
    section_type: string;
    count: number;
    parent_count?: number;
    child_count?: number;
    is_active: number;
    thumb?: string;
    art?: string;
}

export interface TautulliLibraryData {
    section_id: number;
    section_name: string;
    section_type: string;
    count: number;
    parent_count?: number;
    child_count?: number;
    is_active: number;
    thumb?: string;
    art?: string;
    agent?: string;
    scanner?: string;
    library_art?: string;
}

// Tautulli Server Info interfaces
export interface TautulliServerInfoData {
    plex_server_name: string;
    plex_server_version: string;
    plex_server_platform: string;
    platform: string;
    platform_version: string;
    plex_server_up_to_date: number;
    update_available: number;
    update_version?: string;
    update_release_date?: string;
    machine_identifier: string;
}

// Tautulli Synced Items interfaces
export interface TautulliSyncedItem {
    id: number;
    device_name: string;
    platform: string;
    user_id: number;
    username: string;
    friendly_name: string;
    sync_id: string;
    rating_key: string;
    title: string;
    parent_title?: string;
    grandparent_title?: string;
    media_type: string;
    content_type: string;
    state: string;
    item_count: number;
    item_complete_count: number;
    item_downloaded_count: number;
    item_downloaded_percent_complete: number;
}

// Tautulli library names
export interface TautulliLibraryNameItem {
    section_id: number;
    section_name: string;
    section_type?: string;
}

// Tautulli User Login interfaces
export interface TautulliUserLoginRow {
    timestamp: number;
    time: string;
    user_id: number;
    user: string;
    friendly_name: string;
    ip_address: string;
    host?: string;
    user_agent?: string;
    os?: string;
    browser?: string;
    success: number;
}

export interface TautulliUserLoginsData {
    recordsTotal: number;
    recordsFiltered: number;
    draw: number;
    data: TautulliUserLoginRow[];
}

// Tautulli User IP interfaces
export interface TautulliUserIPData {
    friendly_name: string;
    ip_address: string;
    last_seen: number;
    last_played: string;
    play_count: number;
    platform_name?: string;
    player_name?: string;
    user_id: number;
}

// Tautulli Stream Type by Platform interfaces
export interface TautulliStreamTypeByPlatform {
    platform: string;
    direct_play: number;
    direct_stream: number;
    transcode: number;
    total_plays: number;
}

// Tautulli Activity History interfaces
export interface TautulliHistoryRow {
    reference_id: number;
    row_id: number;
    id: number;
    date: number;
    started: number;
    stopped: number;
    duration: number;
    paused_counter: number;
    user_id: number;
    user: string;
    friendly_name: string;
    platform: string;
    player: string;
    ip_address: string;
    media_type: string;
    rating_key: string;
    parent_rating_key?: string;
    grandparent_rating_key?: string;
    full_title: string;
    title: string;
    parent_title?: string;
    grandparent_title?: string;
    year?: number;
    transcode_decision: string;
    percent_complete: number;
    watched_status: number;
    group_count: number;
    group_ids: string;
    state?: string;
    session_key?: string;
}

// Tautulli Plays Per Month interfaces
export interface TautulliPlaysByDateSeries {
    name: string;  // "Movies", "TV", "Music", "Live TV"
    data: number[];  // Array of play counts
}

export interface TautulliPlaysPerMonthData {
    categories: string[];  // Array of month strings (YYYY-MM)
    series: TautulliPlaysByDateSeries[];
}

// Tautulli Concurrent Streams by Type interfaces
export interface TautulliConcurrentStreamsByType {
    transcode_decision: string;  // "direct play", "direct stream", "transcode"
    avg_concurrent: number;
    max_concurrent: number;
    percentage: number;
}

// ========================================================================
// Server Management Types
// ========================================================================

// Tautulli application info
export interface TautulliInfo {
    tautulli_version: string;
    tautulli_platform: string;
    tautulli_branch: string;
    tautulli_install_type: string;
    tautulli_remote_access: number;
    tautulli_update_available: boolean;
}

// Server list item
export interface ServerListItem {
    name: string;
    machine_identifier: string;
    host: string;
    port: number;
    ssl: number;
    is_cloud: number;
    platform: string;
    version: string;
}

// Server identity
export interface ServerIdentity {
    machine_identifier: string;
    version: string;
}

// PMS Update status
export interface PMSUpdate {
    update_available: boolean;
    release_date?: string;
    version?: string;
    download_url?: string;
}

// NATS Health status
export interface NATSHealth {
    status: string;
    connected: boolean;
    jetstream_enabled?: boolean;
    streams?: number;
    consumers?: number;
    server_id?: string;
    version?: string;
    error?: string;
}

// ========================================================================
// Collections & Playlists Types
// ========================================================================

// Collection item
export interface TautulliCollectionItem {
    rating_key: string;
    section_id: number;
    title: string;
    sort_title?: string;
    summary?: string;
    thumb?: string;
    art?: string;
    child_count: number;
    min_year?: number;
    max_year?: number;
    added_at?: number;
    updated_at?: number;
    content_rating?: string;
    labels?: string[];
}

// Collections table data
export interface TautulliCollectionsTableData {
    draw: number;
    recordsTotal: number;
    recordsFiltered: number;
    data: TautulliCollectionItem[];
}

// Playlist item
export interface TautulliPlaylistItem {
    rating_key: string;
    section_id?: number;
    title: string;
    sort_title?: string;
    summary?: string;
    thumb?: string;
    composite?: string;
    duration?: number;
    leaf_count: number;
    smart: boolean;
    playlist_type: string;  // "video", "audio", "photo"
    user?: string;
    username?: string;
    added_at?: number;
    updated_at?: number;
}

// Playlists table data
export interface TautulliPlaylistsTableData {
    draw: number;
    recordsTotal: number;
    recordsFiltered: number;
    data: TautulliPlaylistItem[];
}

// User IP data
export interface TautulliUserIP {
    ip_address: string;
    host?: string;
    platform?: string;
    player?: string;
    last_seen?: number;
    play_count?: number;
    city?: string;
    region?: string;
    country?: string;
    code?: string;
    latitude?: number;
    longitude?: number;
}

// ========================================================================
// Library Details Enhancement Types
// ========================================================================

// Library user stats item
export interface TautulliLibraryUserStatsItem {
    user_id: number;
    username: string;
    friendly_name: string;
    total_plays: number;
    total_duration: number;
    last_watch: number;
    last_played?: string;
    user_thumb?: string;
}

// Library media info item
export interface TautulliLibraryMediaInfoItem {
    rating_key: string;
    parent_rating_key?: string;
    grandparent_rating_key?: string;
    title: string;
    sort_title?: string;
    year?: number;
    media_type: string;
    thumb?: string;
    added_at?: number;
    last_played?: number;
    play_count: number;
    file_size?: number;
    bitrate?: number;
    video_resolution?: string;
    video_codec?: string;
    video_full_resolution?: string;
    audio_codec?: string;
    audio_channels?: number;
    container?: string;
    duration?: number;
}

// Library media info response
export interface TautulliLibraryMediaInfoData {
    draw: number;
    recordsTotal: number;
    recordsFiltered: number;
    data: TautulliLibraryMediaInfoItem[];
    filtered_file_size?: number;
    total_file_size?: number;
}

// Library watch time stats item
export interface TautulliLibraryWatchTimeStatsItem {
    query_days: number;
    total_plays: number;
    total_duration: number;
}

// Library table item
export interface TautulliLibraryTableItem {
    section_id: number;
    section_name: string;
    section_type: string;
    agent: string;
    thumb?: string;
    art?: string;
    count: number;
    parent_count?: number;
    child_count?: number;
    is_active: number;
    do_notify: number;
    do_notify_created: number;
    keep_history: number;
    deleted_section: number;
    last_accessed?: number;
    plays: number;
    duration: number;
}

// Library table response
export interface TautulliLibraryTableData {
    draw: number;
    recordsTotal: number;
    recordsFiltered: number;
    data: TautulliLibraryTableItem[];
}

// ========================================================================
// Export Types
// ========================================================================

// Export metadata response
export interface TautulliExportMetadata {
    export_id?: string;
    status: string;
    file_format?: string;
    record_count?: number;
    download_url?: string;
    error?: string;
}

// Export field item
export interface TautulliExportFieldItem {
    field_name: string;
    display_name: string;
    description?: string;
    field_type?: string;
    default?: string;
}

// Exports table row
export interface TautulliExportsTableRow {
    id: number;
    timestamp: number;
    section_id: number;
    user_id?: number;
    rating_key?: number;
    file_format: string;
    export_type: string;
    media_type?: string;
    title?: string;
    custom_fields?: string;
    individual_files?: number;
    file_size: number;
    complete: number;
}

// Exports table data
export interface TautulliExportsTableData {
    draw: number;
    recordsTotal: number;
    recordsFiltered: number;
    data: TautulliExportsTableRow[];
    filter_duration?: string;
}

// ========================================================================
// Rating Keys Types
// ========================================================================

// Rating key mapping - tracks old to new rating key changes
export interface TautulliRatingKeyMapping {
    old_rating_key: string;
    new_rating_key: string;
    title: string;
    media_type: string;
    updated_at?: number;
}

// New rating keys data
export interface TautulliNewRatingKeysData {
    rating_keys: TautulliRatingKeyMapping[];
}

// Old rating keys data
export interface TautulliOldRatingKeysData {
    rating_keys: TautulliRatingKeyMapping[];
}

// ========================================================================
// Metadata Types
// ========================================================================

// Media info for a media item
export interface TautulliMediaInfo {
    id: number;
    container: string;
    bitrate: number;
    height: number;
    width: number;
    aspect_ratio: number;
    video_codec: string;
    video_resolution: string;
    video_framerate: string;
    video_bit_depth: number;
    video_profile: string;
    audio_codec: string;
    audio_channels: number;
    audio_channel_layout: string;
    audio_bitrate: number;
    optimized_version: number;
}

// Full metadata for a media item
export interface TautulliFullMetadataData {
    rating_key: string;
    parent_rating_key: string;
    grandparent_rating_key: string;
    title: string;
    parent_title: string;
    grandparent_title: string;
    original_title: string;
    sort_title: string;
    media_index: number;
    parent_media_index: number;
    studio: string;
    content_rating: string;
    summary: string;
    tagline: string;
    rating: number;
    rating_image: string;
    audience_rating: number;
    audience_rating_image: string;
    user_rating: number;
    duration: number;
    year: number;
    thumb: string;
    parent_thumb: string;
    grandparent_thumb: string;
    art: string;
    banner: string;
    originally_available_at: string;
    added_at: number;
    updated_at: number;
    last_viewed_at: number;
    guid: string;
    guids: string[];
    directors: string[];
    writers: string[];
    actors: string[];
    genres: string[];
    labels: string[];
    collections: string[];
    media_info: TautulliMediaInfo[];
}

// ========================================================================
// Children Metadata Types
// ========================================================================

// Child metadata item (episode, track, etc.)
export interface TautulliChildMetadataItem {
    media_type: string;
    section_id: number;
    library_name?: string;
    rating_key: string;
    parent_rating_key?: string;
    grandparent_rating_key?: string;
    title: string;
    parent_title?: string;
    grandparent_title?: string;
    original_title?: string;
    sort_title?: string;
    media_index?: number;
    parent_media_index?: number;
    year?: number;
    thumb?: string;
    parent_thumb?: string;
    grandparent_thumb?: string;
    added_at?: number;
    updated_at?: number;
    last_viewed_at?: number;
    guid?: string;
    parent_guid?: string;
    grandparent_guid?: string;
}

// Children metadata response
export interface TautulliChildrenMetadataData {
    children_count: number;
    children_list: TautulliChildMetadataItem[];
}

// ========================================================================
// Search Types
// ========================================================================

// Search result item
export interface TautulliSearchResult {
    type: string;
    rating_key: string;
    title: string;
    year: number;
    thumb: string;
    score: number;
    library: string;
    library_id: number;
    media_type: string;
    summary: string;
    grandparent_title?: string;
    parent_title?: string;
}

// Search data response
export interface TautulliSearchData {
    results_count: number;
    results: TautulliSearchResult[];
}

// ========================================================================
// Fuzzy Search Types (RapidFuzz Extension)
// ========================================================================

// Fuzzy search result from local DuckDB
export interface FuzzySearchResult {
    id: string;                      // rating_key
    title: string;
    parent_title?: string;           // Season/album title
    grandparent_title?: string;      // Show/artist title
    media_type: string;              // movie, episode, track
    year?: number;
    score: number;                   // Fuzzy match score (0-100)
    thumb?: string;
}

// Fuzzy search response
export interface FuzzySearchResponse {
    results: FuzzySearchResult[];
    count: number;
    fuzzy_search: boolean;           // true if RapidFuzz was used
}

// Fuzzy user search result
export interface FuzzyUserSearchResult {
    user_id: number;
    username: string;
    friendly_name?: string;
    score: number;                   // Fuzzy match score (0-100)
    user_thumb?: string;
}

// Fuzzy user search response
export interface FuzzyUserSearchResponse {
    results: FuzzyUserSearchResult[];
    count: number;
    fuzzy_search: boolean;           // true if RapidFuzz was used
}

// ========================================================================
// Stream Data Types
// ========================================================================

// Stream data info - detailed stream information
export interface TautulliStreamDataInfo {
    session_key: string;
    transcode_decision: string;
    video_decision: string;
    audio_decision: string;
    subtitle_decision?: string;
    container: string;
    bitrate: number;
    video_codec: string;
    video_resolution: string;
    video_width: number;
    video_height: number;
    video_framerate: string;
    video_bitrate: number;
    audio_codec: string;
    audio_channels: number;
    audio_channel_layout?: string;
    audio_bitrate: number;
    audio_sample_rate: number;
    stream_container?: string;
    stream_container_decision?: string;
    stream_bitrate?: number;
    stream_video_codec?: string;
    stream_video_resolution?: string;
    stream_video_bitrate?: number;
    stream_video_width?: number;
    stream_video_height?: number;
    stream_video_framerate?: string;
    stream_audio_codec?: string;
    stream_audio_channels?: number;
    stream_audio_bitrate?: number;
    stream_audio_sample_rate?: number;
    subtitle_codec?: string;
    optimized?: number;
    throttled?: number;
}

// ============================================
// Phase 3: Additional Tautulli Types
// ============================================

// Tautulli Home Stats interfaces
export interface TautulliHomeStat {
    stat_id: string;
    stat_type: string;
    stat_title?: string;
    rows: TautulliHomeStatRow[];
}

export interface TautulliHomeStatRow {
    title?: string;
    thumb?: string;
    users_watched?: number;
    rating_key?: string;
    grandparent_rating_key?: string;
    user?: string;
    user_id?: number;
    friendly_name?: string;
    user_thumb?: string;
    platform?: string;
    total_plays?: number;
    total_duration?: number;
    last_play?: number;
    last_watch?: number;
    content_rating?: string;
    section_id?: number;
    section_name?: string;
    media_type?: string;
}

export interface TautulliHomeStatsData {
    response: {
        result: string;
        message: string | null;
        data: TautulliHomeStat[];
    };
}

// Tautulli Plays by Date interfaces
export interface TautulliPlaysByDateCategory {
    name: string;
    data: number[];
}

export interface TautulliPlaysByDateData {
    categories: string[];
    series: TautulliPlaysByDateCategory[];
}

// Tautulli Plays by Day of Week interfaces
export interface TautulliPlaysByDayOfWeekData {
    categories: string[];
    series: TautulliPlaysByDateCategory[];
}

// Tautulli Plays by Hour of Day interfaces
export interface TautulliPlaysByHourOfDayData {
    categories: string[];
    series: TautulliPlaysByDateCategory[];
}

// Tautulli Plays by Stream Type interfaces
export interface TautulliPlaysByStreamTypeData {
    categories: string[];
    series: TautulliPlaysByDateCategory[];
}

// Tautulli Plays by Resolution interfaces
export interface TautulliPlaysByResolutionData {
    categories: string[];
    series: TautulliPlaysByDateCategory[];
}

// Tautulli Plays by Top 10 Users/Platforms interfaces
export interface TautulliPlaysByTop10Data {
    categories: string[];
    series: TautulliPlaysByDateCategory[];
}

// Tautulli User interfaces
export interface TautulliUserDetail {
    user_id: number;
    username: string;
    friendly_name?: string;
    email?: string;
    is_active: number;
    is_admin: number;
    is_home_user: number;
    is_allow_sync: number;
    is_restricted: number;
    do_notify: number;
    keep_history: number;
    allow_guest: number;
    user_token?: string;
    server_token?: string;
    shared_libraries?: string;
    filter_all?: string;
    filter_movies?: string;
    filter_tv?: string;
    filter_music?: string;
    filter_photos?: string;
}
