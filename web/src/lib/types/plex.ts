// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Plex Direct API Types
 *
 * Types for Plex Direct API integration including:
 * - Library sections and metadata
 * - Sessions and activities
 * - Devices and accounts
 * - Bandwidth and capabilities
 */

// Plex Server Identity
export interface PlexServerIdentity {
    machineIdentifier: string;
    version: string;
    size?: number;
    claimed?: boolean;
    machineIdentifier_platform?: string;
}

// Plex Library Section
export interface PlexLibrarySection {
    key: string;
    type: string;
    title: string;
    agent: string;
    scanner: string;
    language: string;
    uuid: string;
    updatedAt?: number;
    createdAt?: number;
    scannedAt?: number;
    content?: boolean;
    directory?: boolean;
    contentChangedAt?: number;
    hidden?: number;
    Location?: PlexLocation[];
}

export interface PlexLocation {
    id: number;
    path: string;
}

// Plex Media Metadata
export interface PlexMediaMetadata {
    ratingKey: string;
    key: string;
    guid?: string;
    type: string;
    title: string;
    titleSort?: string;
    originalTitle?: string;
    summary?: string;
    rating?: number;
    audienceRating?: number;
    year?: number;
    thumb?: string;
    art?: string;
    banner?: string;
    duration?: number;
    originallyAvailableAt?: string;
    addedAt?: number;
    updatedAt?: number;
    contentRating?: string;
    studio?: string;
    tagline?: string;
    parentRatingKey?: string;
    parentTitle?: string;
    grandparentRatingKey?: string;
    grandparentTitle?: string;
    grandparentThumb?: string;
    grandparentArt?: string;
    index?: number;
    parentIndex?: number;
    viewCount?: number;
    lastViewedAt?: number;
    Genre?: PlexTag[];
    Director?: PlexTag[];
    Writer?: PlexTag[];
    Role?: PlexTag[];
    Country?: PlexTag[];
    Media?: PlexMedia[];
}

export interface PlexTag {
    id?: number;
    filter?: string;
    tag: string;
    role?: string;
    thumb?: string;
}

export interface PlexMedia {
    id: number;
    duration?: number;
    bitrate?: number;
    width?: number;
    height?: number;
    aspectRatio?: number;
    audioChannels?: number;
    audioCodec?: string;
    videoCodec?: string;
    videoResolution?: string;
    container?: string;
    videoFrameRate?: string;
    videoProfile?: string;
    Part?: PlexMediaPart[];
}

export interface PlexMediaPart {
    id: number;
    key?: string;
    duration?: number;
    file?: string;
    size?: number;
    container?: string;
    videoProfile?: string;
    Stream?: PlexStream[];
}

export interface PlexStream {
    id: number;
    streamType: number;
    default?: boolean;
    codec?: string;
    index?: number;
    bitrate?: number;
    language?: string;
    languageCode?: string;
    title?: string;
    displayTitle?: string;
}

// Plex Session
export interface PlexDirectSession {
    sessionKey: string;
    ratingKey: string;
    key: string;
    type: string;
    title: string;
    parentTitle?: string;
    grandparentTitle?: string;
    thumb?: string;
    art?: string;
    User?: PlexUser;
    Player?: PlexPlayer;
    Session?: PlexSessionInfo;
    TranscodeSession?: PlexTranscodeSession;
    viewOffset?: number;
    duration?: number;
}

export interface PlexUser {
    id: string;
    title: string;
    thumb?: string;
}

export interface PlexPlayer {
    machineIdentifier: string;
    title: string;
    state: string;
    local: boolean;
    relayed?: boolean;
    secure?: boolean;
    address?: string;
    port?: number;
    device?: string;
    model?: string;
    platform?: string;
    platformVersion?: string;
    product?: string;
    profile?: string;
    version?: string;
}

export interface PlexSessionInfo {
    id: string;
    bandwidth?: number;
    location?: string;
}

export interface PlexTranscodeSession {
    key: string;
    throttled: boolean;
    complete: boolean;
    progress: number;
    size?: number;
    speed?: number;
    error?: boolean;
    duration?: number;
    remaining?: number;
    context?: string;
    sourceVideoCodec?: string;
    sourceAudioCodec?: string;
    videoCodec?: string;
    audioCodec?: string;
    videoDecision?: string;
    audioDecision?: string;
    subtitleDecision?: string;
    protocol?: string;
    container?: string;
    width?: number;
    height?: number;
    audioChannels?: number;
    transcodeHwRequested?: boolean;
    transcodeHwDecoding?: string;
    transcodeHwEncoding?: string;
    transcodeHwDecodingTitle?: string;
    transcodeHwFullPipeline?: boolean;
}

// Plex Device
export interface PlexDevice {
    id: number;
    name: string;
    platform: string;
    clientIdentifier: string;
    createdAt?: number;
    lastSeenAt?: number;
    provides?: string;
    owned?: boolean;
    accessToken?: string;
    httpsRequired?: boolean;
    publicAddress?: string;
    publicAddressMatches?: boolean;
    presence?: boolean;
    Connection?: PlexConnection[];
}

export interface PlexConnection {
    protocol: string;
    address: string;
    port: number;
    uri: string;
    local: boolean;
    relay?: boolean;
}

// Plex Account
export interface PlexAccount {
    id: number;
    key: string;
    name: string;
    defaultAudioLanguage?: string;
    autoSelectAudio?: boolean;
    defaultSubtitleLanguage?: string;
    subtitleMode?: number;
    thumb?: string;
}

// Plex Activity
export interface PlexActivity {
    uuid: string;
    type: string;
    cancellable: boolean;
    userID: number;
    title: string;
    subtitle?: string;
    progress: number;
    Context?: PlexActivityContext;
}

export interface PlexActivityContext {
    key: string;
}

// Plex Playlist
export interface PlexPlaylist {
    ratingKey: string;
    key: string;
    guid?: string;
    type: string;
    title: string;
    summary?: string;
    smart: boolean;
    playlistType: string;
    composite?: string;
    icon?: string;
    viewCount?: number;
    lastViewedAt?: number;
    duration?: number;
    leafCount?: number;
    addedAt?: number;
    updatedAt?: number;
}

// Plex Capabilities
export interface PlexCapabilities {
    machineIdentifier: string;
    version: string;
    transcoderActiveVideoSessions?: number;
    transcoderAudio?: boolean;
    transcoderLyrics?: boolean;
    transcoderPhoto?: boolean;
    transcoderSubtitles?: boolean;
    transcoderVideo?: boolean;
    transcoderVideoBitrates?: string;
    transcoderVideoQualities?: string;
    transcoderVideoResolutions?: string;
}

// Plex Bandwidth Statistics
export interface PlexBandwidthStatistics {
    accountID: number;
    deviceID: number;
    timespan: number;
    at?: number;
    lan?: boolean;
    bytes?: number;
}

// ========================================================================
// Plex Friends and Sharing Types (Library Access Management)
// ========================================================================

// Plex Friend
export interface PlexFriend {
    id: number;
    uuid: string;
    username: string;
    email: string;
    thumb: string;
    title: string;
    server: boolean;
    home: boolean;
    allowSync: boolean;
    allowCameraUpload: boolean;
    allowChannels: boolean;
    sharedSections: number[];
    filterMovies?: string;
    filterTelevision?: string;
    filterMusic?: string;
    status: 'accepted' | 'pending' | 'pending_received';
}

// Plex Friends List Response
export interface PlexFriendsListResponse {
    friends: PlexFriend[];
    total: number;
}

// Plex Invite Friend Request
export interface PlexInviteFriendRequest {
    email: string;
    allowSync?: boolean;
    allowCameraUpload?: boolean;
    allowChannels?: boolean;
}

// Plex Shared Server
export interface PlexSharedServer {
    id: number;
    userId: number;
    username: string;
    email: string;
    thumb: string;
    invitedEmail?: string;
    acceptedAt?: string;
    allowSync: boolean;
    allowCameraUpload: boolean;
    allowChannels: boolean;
    filterMovies?: string;
    filterTelevision?: string;
    filterMusic?: string;
    sharedSections: number[];
}

// Plex Shared Servers List Response
export interface PlexSharedServersListResponse {
    sharedServers: PlexSharedServer[];
    total: number;
}

// Plex Share Libraries Request
export interface PlexShareLibrariesRequest {
    email: string;
    librarySectionIds: number[];
    allowSync?: boolean;
    allowCameraUpload?: boolean;
    allowChannels?: boolean;
    filterMovies?: string;
    filterTelevision?: string;
    filterMusic?: string;
}

// Plex Update Sharing Request
export interface PlexUpdateSharingRequest {
    librarySectionIds: number[];
    allowSync?: boolean;
    allowCameraUpload?: boolean;
    allowChannels?: boolean;
    filterMovies?: string;
    filterTelevision?: string;
    filterMusic?: string;
}

// Plex Managed User
export interface PlexManagedUser {
    id: number;
    uuid: string;
    username: string;
    title: string;
    thumb?: string;
    restricted: boolean;
    restrictionProfile: 'little_kid' | 'older_kid' | 'teen' | '';
    home: boolean;
    homeAdmin: boolean;
    guest: boolean;
    protected: boolean;
}

// Plex Managed Users List Response
export interface PlexManagedUsersListResponse {
    users: PlexManagedUser[];
    total: number;
}

// Plex Create Managed User Request
export interface PlexCreateManagedUserRequest {
    name: string;
    restrictionProfile?: 'little_kid' | 'older_kid' | 'teen';
}

// Plex Update Managed User Request
export interface PlexUpdateManagedUserRequest {
    restrictionProfile: 'little_kid' | 'older_kid' | 'teen' | '';
}

// Plex Library Section for Sharing UI
export interface PlexLibrarySectionForSharing {
    id: number;
    key: string;
    type: string;
    title: string;
    thumb?: string;
    itemCount: number;
}

// Plex Library Sections List Response (for sharing UI)
export interface PlexLibrarySectionsListResponse {
    sections: PlexLibrarySectionForSharing[];
    total: number;
}

// Response wrapper types
export interface PlexMediaContainer<T> {
    MediaContainer: {
        size: number;
        allowSync?: boolean;
        identifier?: string;
        librarySectionID?: number;
        librarySectionTitle?: string;
        librarySectionUUID?: string;
        Metadata?: T[];
        Directory?: T[];
        Device?: T[];
        Account?: T[];
        Activity?: T[];
        Video?: T[];
        Track?: T[];
        Photo?: T[];
        Playlist?: T[];
        Server?: T[];
        StatisticsBandwidth?: T[];
    };
}

/**
 * FIX: Specific response type for /plex/identity endpoint.
 * The identity data is directly on MediaContainer, not in an array.
 */
export interface PlexIdentityResponse {
    MediaContainer: PlexServerIdentity & {
        size?: number;
    };
}

/**
 * FIX: Specific response type for /plex/capabilities endpoint.
 * The capabilities data is directly on MediaContainer, not in an array.
 */
export interface PlexCapabilitiesResponse {
    MediaContainer: PlexCapabilities & {
        size?: number;
    };
}
