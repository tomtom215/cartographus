// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * API Module Index
 *
 * Provides a unified API class that composes all domain-specific modules.
 * Maintains backward compatibility with the original monolithic API class.
 *
 * Architecture:
 * - BaseAPIClient: Core fetch, auth, caching functionality
 * - Domain modules: Specialized API methods grouped by domain
 * - API class: Facade that delegates to domain modules
 *
 * Usage:
 * ```typescript
 * import { API } from './api';
 * const api = new API('http://localhost:3857/api');
 * await api.login({ username, password });
 * const stats = await api.getStats();
 * ```
 */

// Export base client for extension
export { BaseAPIClient, type APIResponse, type CacheStatusCallback, queuedFetch } from './client';

// Export domain-specific API classes for direct use if needed
export { AuthAPI } from './auth';
export { CoreAPI, LocationsAPI } from './core';
export { AnalyticsAPI } from './analytics';
export { SpatialAPI } from './spatial';
export { TautulliAPI } from './tautulli';
export { BackupAPI } from './backup';
export { PlexAPI } from './plex';
export { DetectionAPI } from './detection';
export { DedupeAPI } from './dedupe';
export { AuditAPI } from './audit';
export { DLQAPI } from './dlq';
export { WALAPI } from './wal';
export { CrossPlatformAPI } from './cross-platform';
export { RecommendAPI } from './recommend';
export { WrappedAPI } from './wrapped';
export { NewsletterAPI } from './newsletter';
export { ServerAPI } from './server';
export { SyncAPI } from './sync';

// Import all domain modules for composition
import type { CacheStatusCallback } from './client';
import { AuthAPI } from './auth';
import { CoreAPI, LocationsAPI } from './core';
import { AnalyticsAPI } from './analytics';
import { SpatialAPI } from './spatial';
import { TautulliAPI } from './tautulli';
import { BackupAPI } from './backup';
import { PlexAPI } from './plex';
import { DetectionAPI } from './detection';
import { DedupeAPI } from './dedupe';
import { AuditAPI } from './audit';
import { DLQAPI } from './dlq';
import { WALAPI } from './wal';
import { CrossPlatformAPI } from './cross-platform';
import { RecommendAPI } from './recommend';
import { WrappedAPI } from './wrapped';
import { NewsletterAPI } from './newsletter';
import { ServerAPI } from './server';
import { SyncAPI } from './sync';

// Import only the types that are used directly in the API facade class
import type { LocationFilter } from '../types/core';
import type { LoginRequest, LoginResponse, SessionsResponse, LogoutResponse } from '../types/auth';
import type { ApproximateStatsFilter } from '../types/analytics';
import type { CreateBackupRequest, RestoreBackupRequest, RetentionPolicy, SetScheduleConfigRequest } from '../types/backup';
import type { DetectionRuleType, DetectionAlertFilter, UpdateRuleRequest } from '../types/detection';
import type { DedupeAuditActionRequest, DedupeAuditFilter } from '../types/dedupe';
import type { AuditEventFilter } from '../types/audit';
import type { DLQEntryFilter } from '../types/dlq';
import type { ContentMappingRequest, ContentLookupParams, UserLinkRequest } from '../types/cross-platform';
import type { RecommendationRequest, UserPreference, WhatsNextRequest } from '../types/recommend';
import type { WrappedGenerateRequest } from '../types/wrapped';
import { getAPICacheManager, type APICacheManager } from '../api-cache';

/**
 * Unified API class that composes all domain-specific modules.
 * Provides full backward compatibility with the original monolithic API class.
 */
export class API {
    private readonly auth: AuthAPI;
    private readonly core: CoreAPI;
    private readonly locations: LocationsAPI;
    private readonly analytics: AnalyticsAPI;
    private readonly spatial: SpatialAPI;
    private readonly tautulli: TautulliAPI;
    private readonly backup: BackupAPI;
    private readonly plex: PlexAPI;
    private readonly detection: DetectionAPI;
    private readonly dedupe: DedupeAPI;
    private readonly audit: AuditAPI;
    private readonly dlq: DLQAPI;
    private readonly wal: WALAPI;
    private readonly crossPlatform: CrossPlatformAPI;
    private readonly recommend: RecommendAPI;
    private readonly wrapped: WrappedAPI;
    private readonly newsletter: NewsletterAPI;
    private readonly server: ServerAPI;
    private readonly sync: SyncAPI;

    private cacheManager: APICacheManager;

    constructor(baseURL: string) {
        // Initialize all domain modules with the same baseURL
        this.auth = new AuthAPI(baseURL);
        this.core = new CoreAPI(baseURL);
        this.locations = new LocationsAPI(baseURL);
        this.analytics = new AnalyticsAPI(baseURL);
        this.spatial = new SpatialAPI(baseURL);
        this.tautulli = new TautulliAPI(baseURL);
        this.backup = new BackupAPI(baseURL);
        this.plex = new PlexAPI(baseURL);
        this.detection = new DetectionAPI(baseURL);
        this.dedupe = new DedupeAPI(baseURL);
        this.audit = new AuditAPI(baseURL);
        this.dlq = new DLQAPI(baseURL);
        this.wal = new WALAPI(baseURL);
        this.crossPlatform = new CrossPlatformAPI(baseURL);
        this.recommend = new RecommendAPI(baseURL);
        this.wrapped = new WrappedAPI(baseURL);
        this.newsletter = new NewsletterAPI(baseURL);
        this.server = new ServerAPI(baseURL);
        this.sync = new SyncAPI(baseURL);
        this.cacheManager = getAPICacheManager();
    }

    // ========================================================================
    // Cache Management
    // ========================================================================

    setCachingEnabled(enabled: boolean): void {
        const modules = [this.auth, this.core, this.locations, this.analytics, this.spatial,
            this.tautulli, this.backup, this.plex, this.detection, this.dedupe,
            this.audit, this.dlq, this.wal, this.crossPlatform, this.recommend, this.wrapped,
            this.newsletter, this.server, this.sync];
        modules.forEach(m => m.setCachingEnabled(enabled));
    }

    getCacheStats(): ReturnType<APICacheManager['getStats']> {
        return this.cacheManager.getStats();
    }

    invalidateCache(pattern: string): void {
        this.cacheManager.invalidate(pattern);
    }

    clearCache(): void {
        this.cacheManager.clear();
    }

    setCacheStatusCallback(callback: CacheStatusCallback): void {
        const modules = [this.auth, this.core, this.locations, this.analytics, this.spatial,
            this.tautulli, this.backup, this.plex, this.detection, this.dedupe,
            this.audit, this.dlq, this.wal, this.crossPlatform, this.wrapped, this.newsletter,
            this.server, this.sync];
        modules.forEach(m => m.setCacheStatusCallback(callback));
    }

    // ========================================================================
    // Authentication (from AuthAPI)
    // ========================================================================

    async login(credentials: LoginRequest): Promise<LoginResponse> {
        const result = await this.auth.login(credentials);
        // Sync token to all modules
        this.syncTokenToModules(this.auth.getToken());
        return result;
    }

    logout(): void {
        this.auth.logout();
        this.syncTokenToModules(null);
    }

    isAuthenticated(): boolean {
        return this.auth.isAuthenticated();
    }

    getUsername(): string | null {
        return this.auth.getUsername();
    }

    /**
     * Get the current user's role.
     * RBAC Phase 4: Frontend Role Integration
     */
    getRole = () => this.auth.getRole();

    /**
     * Get the current user's ID.
     * RBAC Phase 4: Frontend Role Integration
     */
    getUserId = () => this.auth.getUserId();

    /**
     * Check if the current user is an admin.
     * RBAC Phase 4: Frontend Role Integration
     */
    isAdmin = () => this.auth.isAdmin();

    /**
     * Check if the current user is an editor or higher.
     * RBAC Phase 4: Frontend Role Integration
     */
    isEditor = () => this.auth.isEditor();

    // ========================================================================
    // Session Management (ADR-0015: Zero Trust Authentication)
    // ========================================================================

    /**
     * Get all active sessions for the current user.
     */
    async getSessions(): Promise<SessionsResponse> {
        return this.auth.getSessions();
    }

    /**
     * Revoke a specific session by ID.
     */
    async revokeSession(sessionId: string): Promise<LogoutResponse> {
        return this.auth.revokeSession(sessionId);
    }

    /**
     * Logout from all sessions ("Sign out everywhere").
     */
    async logoutAll(): Promise<LogoutResponse> {
        const result = await this.auth.logoutAll();
        this.syncTokenToModules(null);
        return result;
    }

    /**
     * Get user info for the current session.
     */
    async getUserInfo(): Promise<Record<string, unknown>> {
        return this.auth.getUserInfo();
    }

    private syncTokenToModules(token: string | null): void {
        const modules = [this.core, this.locations, this.analytics, this.spatial,
            this.tautulli, this.backup, this.plex, this.detection, this.dedupe,
            this.audit, this.dlq, this.wal, this.crossPlatform, this.wrapped, this.newsletter,
            this.server, this.sync];
        modules.forEach(m => m.setToken(token));
    }

    // ========================================================================
    // Core (from CoreAPI)
    // ========================================================================

    getStats = () => this.core.getStats();
    getHealthStatus = () => this.core.getHealthStatus();
    getUsers = () => this.core.getUsers();
    getMediaTypes = () => this.core.getMediaTypes();
    triggerSync = () => this.core.triggerSync();
    getServerInfo = () => this.core.getServerInfo();

    // ========================================================================
    // Locations & Playbacks (from LocationsAPI)
    // ========================================================================

    getLocations = (filter?: LocationFilter) => this.locations.getLocations(filter);
    getPlaybacks = (filter?: LocationFilter, limit?: number, offset?: number) => this.locations.getPlaybacks(filter, limit, offset);
    getPlaybacksWithCursor = (filter?: LocationFilter, limit?: number, cursor?: string) => this.locations.getPlaybacksWithCursor(filter, limit, cursor);
    getAllPlaybacks = (filter?: LocationFilter, pageSize?: number, onProgress?: (loaded: number, hasMore: boolean) => void) => this.locations.getAllPlaybacks(filter, pageSize, onProgress);

    // ========================================================================
    // Analytics (from AnalyticsAPI)
    // ========================================================================

    getAnalyticsTrends = (filter?: LocationFilter) => this.analytics.getAnalyticsTrends(filter);
    getAnalyticsGeographic = (filter?: LocationFilter) => this.analytics.getAnalyticsGeographic(filter);
    getAnalyticsUsers = (filter?: LocationFilter, limit?: number) => this.analytics.getAnalyticsUsers(filter, limit);
    getAnalyticsBinge = (filter?: LocationFilter) => this.analytics.getAnalyticsBinge(filter);
    getAnalyticsBandwidth = (filter?: LocationFilter) => this.analytics.getAnalyticsBandwidth(filter);
    getAnalyticsBitrate = (filter?: LocationFilter) => this.analytics.getAnalyticsBitrate(filter);
    getAnalyticsPopular = (filter?: LocationFilter, limit?: number) => this.analytics.getAnalyticsPopular(filter, limit);
    getAnalyticsWatchParties = (filter?: LocationFilter) => this.analytics.getAnalyticsWatchParties(filter);
    getAnalyticsUserEngagement = (filter?: LocationFilter, limit?: number) => this.analytics.getAnalyticsUserEngagement(filter, limit);
    getAnalyticsComparative = (filter?: LocationFilter, comparisonType?: 'week' | 'month' | 'quarter' | 'year' | 'custom') => this.analytics.getAnalyticsComparative(filter, comparisonType);
    getTemporalHeatmap = (filter?: LocationFilter, interval?: 'hour' | 'day' | 'week' | 'month') => this.analytics.getTemporalHeatmap(filter, interval);
    getAnalyticsResolutionMismatch = (filter?: LocationFilter) => this.analytics.getAnalyticsResolutionMismatch(filter);
    getAnalyticsHDR = (filter?: LocationFilter) => this.analytics.getAnalyticsHDR(filter);
    getAnalyticsAudio = (filter?: LocationFilter) => this.analytics.getAnalyticsAudio(filter);
    getAnalyticsSubtitles = (filter?: LocationFilter) => this.analytics.getAnalyticsSubtitles(filter);
    getAnalyticsConnectionSecurity = (filter?: LocationFilter) => this.analytics.getAnalyticsConnectionSecurity(filter);
    getAnalyticsPausePatterns = (filter?: LocationFilter) => this.analytics.getAnalyticsPausePatterns(filter);
    getAnalyticsConcurrentStreams = (filter?: LocationFilter) => this.analytics.getAnalyticsConcurrentStreams(filter);
    getAnalyticsHardwareTranscode = (filter?: LocationFilter) => this.analytics.getAnalyticsHardwareTranscode(filter);
    getAnalyticsAbandonment = (filter?: LocationFilter) => this.analytics.getAnalyticsAbandonment(filter);
    getAnalyticsLibrary = (sectionId: number, filter?: LocationFilter) => this.analytics.getAnalyticsLibrary(sectionId, filter);
    getAnalyticsUserProfile = (username: string, filter?: LocationFilter) => this.analytics.getAnalyticsUserProfile(username, filter);
    getAnalyticsFrameRate = (filter?: LocationFilter) => this.analytics.getAnalyticsFrameRate(filter);
    getAnalyticsContainer = (filter?: LocationFilter) => this.analytics.getAnalyticsContainer(filter);
    getAnalyticsHardwareTranscodeTrends = (filter?: LocationFilter) => this.analytics.getAnalyticsHardwareTranscodeTrends(filter);
    getAnalyticsApproximate = (filter?: ApproximateStatsFilter) => this.analytics.getAnalyticsApproximate(filter);
    getAnalyticsApproximateDistinct = (column: string, filter?: ApproximateStatsFilter) => this.analytics.getAnalyticsApproximateDistinct(column, filter);
    getAnalyticsApproximatePercentile = (column: string, percentile: number, filter?: ApproximateStatsFilter) => this.analytics.getAnalyticsApproximatePercentile(column, percentile, filter);

    // ========================================================================
    // Spatial (from SpatialAPI)
    // ========================================================================

    getH3HexagonData = (filter?: LocationFilter, resolution?: number) => this.spatial.getH3HexagonData(filter, resolution);
    getArcData = (filter?: LocationFilter) => this.spatial.getArcData(filter);
    getSpatialViewport = (bbox: { west: number; south: number; east: number; north: number }, filter?: LocationFilter) => this.spatial.getSpatialViewport(bbox, filter);
    getSpatialTemporalDensity = (interval?: 'hour' | 'day' | 'week' | 'month', resolution?: number, filter?: LocationFilter) => this.spatial.getSpatialTemporalDensity(interval, resolution, filter);
    getSpatialNearby = (lat: number, lon: number, radius?: number, filter?: LocationFilter) => this.spatial.getSpatialNearby(lat, lon, radius, filter);
    getStreamingGeoJSONUrl = (filter?: LocationFilter) => this.spatial.getStreamingGeoJSONUrl(filter);
    streamLocationsGeoJSON = (filter?: LocationFilter, onProgress?: (loaded: number, total: number) => void, signal?: AbortSignal) => this.spatial.streamLocationsGeoJSON(filter, onProgress, signal);

    // ========================================================================
    // Tautulli (from TautulliAPI)
    // ========================================================================

    getTautulliActivity = () => this.tautulli.getTautulliActivity();
    getTautulliServerInfo = () => this.tautulli.getTautulliServerInfo();
    getTautulliRecentlyAdded = (count?: number, start?: number, mediaType?: string, sectionId?: number) => this.tautulli.getTautulliRecentlyAdded(count, start, mediaType, sectionId);
    getTautulliLibraries = () => this.tautulli.getTautulliLibraries();
    getTautulliLibraryNames = () => this.tautulli.getTautulliLibraryNames();
    getTautulliUserLogins = (userId?: number) => this.tautulli.getTautulliUserLogins(userId);
    getTautulliUserIPs = (userId: number) => this.tautulli.getTautulliUserIPs(userId);
    getTautulliUser = (userId: number) => this.tautulli.getTautulliUser(userId);
    getTautulliUsers = () => this.tautulli.getTautulliUsers();
    getUserIPDetails = (userId: number) => this.tautulli.getUserIPDetails(userId);
    getTautulliSyncedItems = () => this.tautulli.getTautulliSyncedItems();
    getTautulliStreamTypeByPlatform = () => this.tautulli.getTautulliStreamTypeByPlatform();
    getTautulliHistory = (start?: number, length?: number) => this.tautulli.getTautulliHistory(start, length);
    getTautulliPlaysPerMonth = (timeRange?: number, yAxis?: string) => this.tautulli.getTautulliPlaysPerMonth(timeRange, yAxis);
    getTautulliConcurrentStreamsByType = () => this.tautulli.getTautulliConcurrentStreamsByType();
    getTautulliHomeStats = (timeRange?: number, statId?: string) => this.tautulli.getTautulliHomeStats(timeRange, statId);
    getTautulliPlaysByDate = (timeRange?: number, yAxis?: string) => this.tautulli.getTautulliPlaysByDate(timeRange, yAxis);
    getTautulliPlaysByDayOfWeek = (timeRange?: number, yAxis?: string) => this.tautulli.getTautulliPlaysByDayOfWeek(timeRange, yAxis);
    getTautulliPlaysByHourOfDay = (timeRange?: number, yAxis?: string) => this.tautulli.getTautulliPlaysByHourOfDay(timeRange, yAxis);
    getTautulliPlaysByStreamType = (timeRange?: number, yAxis?: string) => this.tautulli.getTautulliPlaysByStreamType(timeRange, yAxis);
    getTautulliPlaysBySourceResolution = (timeRange?: number, yAxis?: string) => this.tautulli.getTautulliPlaysBySourceResolution(timeRange, yAxis);
    getTautulliPlaysByStreamResolution = (timeRange?: number, yAxis?: string) => this.tautulli.getTautulliPlaysByStreamResolution(timeRange, yAxis);
    getTautulliPlaysByTop10Platforms = (timeRange?: number, yAxis?: string) => this.tautulli.getTautulliPlaysByTop10Platforms(timeRange, yAxis);
    getTautulliPlaysByTop10Users = (timeRange?: number, yAxis?: string) => this.tautulli.getTautulliPlaysByTop10Users(timeRange, yAxis);
    getTautulliInfo = () => this.tautulli.getTautulliInfo();
    getTautulliServerList = () => this.tautulli.getTautulliServerList();
    getTautulliServerIdentity = () => this.tautulli.getTautulliServerIdentity();
    getTautulliPMSUpdate = () => this.tautulli.getTautulliPMSUpdate();
    getNATSHealth = () => this.tautulli.getNATSHealth();
    terminateSession = (sessionId: string, message?: string) => this.tautulli.terminateSession(sessionId, message);
    getCollectionsTable = (sectionId?: number, start?: number, length?: number) => this.tautulli.getCollectionsTable(sectionId, start, length);
    getPlaylistsTable = (start?: number, length?: number) => this.tautulli.getPlaylistsTable(start, length);
    getLibraryUserStats = (sectionId: number) => this.tautulli.getLibraryUserStats(sectionId);
    getLibraryMediaInfo = (sectionId: number, start?: number, length?: number, orderColumn?: string, orderDir?: 'asc' | 'desc') => this.tautulli.getLibraryMediaInfo(sectionId, start, length, orderColumn, orderDir);
    getLibraryWatchTimeStats = (sectionId: number, queryDays?: number[]) => this.tautulli.getLibraryWatchTimeStats(sectionId, queryDays);
    getLibrariesTable = () => this.tautulli.getLibrariesTable();
    getExportFields = (mediaType: string) => this.tautulli.getExportFields(mediaType);
    getExportsTable = (start?: number, length?: number, orderColumn?: string, orderDir?: 'asc' | 'desc') => this.tautulli.getExportsTable(start, length, orderColumn, orderDir);
    createExport = (sectionId: number, exportType: string, fileFormat?: string, userId?: number, ratingKey?: number) => this.tautulli.createExport(sectionId, exportType, fileFormat, userId, ratingKey);
    getExportDownloadUrl = (exportId: number) => this.tautulli.getExportDownloadUrl(exportId);
    deleteExport = (exportId: number) => this.tautulli.deleteExport(exportId);
    getNewRatingKeys = (ratingKey: string) => this.tautulli.getNewRatingKeys(ratingKey);
    getOldRatingKeys = (ratingKey: string) => this.tautulli.getOldRatingKeys(ratingKey);
    getMetadata = (ratingKey: string) => this.tautulli.getMetadata(ratingKey);
    getChildrenMetadata = (ratingKey: string) => this.tautulli.getChildrenMetadata(ratingKey);
    search = (query: string, limit?: number) => this.tautulli.search(query, limit);
    fuzzySearch = (query: string, options?: { minScore?: number; limit?: number }) => this.tautulli.fuzzySearch(query, options);
    fuzzySearchUsers = (query: string, options?: { minScore?: number; limit?: number }) => this.tautulli.fuzzySearchUsers(query, options);
    getStreamData = (sessionKey?: string, rowId?: number) => this.tautulli.getStreamData(sessionKey, rowId);

    // ========================================================================
    // Backup (from BackupAPI)
    // ========================================================================

    listBackups = () => this.backup.listBackups();
    getBackupStats = () => this.backup.getBackupStats();
    createBackup = (request?: CreateBackupRequest) => this.backup.createBackup(request);
    deleteBackup = (backupId: string) => this.backup.deleteBackup(backupId);
    getBackupDownloadUrl = (backupId: string) => this.backup.getBackupDownloadUrl(backupId);
    validateBackup = (backupId: string) => this.backup.validateBackup(backupId);
    restoreBackup = (backupId: string, options?: RestoreBackupRequest) => this.backup.restoreBackup(backupId, options);
    getRetentionPolicy = () => this.backup.getRetentionPolicy();
    setRetentionPolicy = (policy: RetentionPolicy) => this.backup.setRetentionPolicy(policy);
    getRetentionPreview = () => this.backup.getRetentionPreview();
    applyRetention = () => this.backup.applyRetention();
    cleanupCorruptedBackups = () => this.backup.cleanupCorruptedBackups();
    getScheduleConfig = () => this.backup.getScheduleConfig();
    setScheduleConfig = (config: SetScheduleConfigRequest) => this.backup.setScheduleConfig(config);
    triggerScheduledBackup = () => this.backup.triggerScheduledBackup();

    // ========================================================================
    // Plex Direct (from PlexAPI)
    // ========================================================================

    getPlexIdentity = () => this.plex.getPlexIdentity();
    getPlexLibrarySections = () => this.plex.getPlexLibrarySections();
    getPlexLibraryAll = (sectionKey: string) => this.plex.getPlexLibraryAll(sectionKey);
    getPlexLibraryRecentlyAdded = (sectionKey: string) => this.plex.getPlexLibraryRecentlyAdded(sectionKey);
    getPlexLibrarySearch = (sectionKey: string, query: string) => this.plex.getPlexLibrarySearch(sectionKey, query);
    getPlexMetadata = (ratingKey: string) => this.plex.getPlexMetadata(ratingKey);
    getPlexOnDeck = () => this.plex.getPlexOnDeck();
    getPlexSessions = () => this.plex.getPlexSessions();
    getPlexActivities = () => this.plex.getPlexActivities();
    getPlexDevices = () => this.plex.getPlexDevices();
    getPlexAccounts = () => this.plex.getPlexAccounts();
    getPlexPlaylists = () => this.plex.getPlexPlaylists();
    getPlexCapabilities = () => this.plex.getPlexCapabilities();
    getPlexBandwidthStatistics = () => this.plex.getPlexBandwidthStatistics();
    getPlexTranscodeSessions = () => this.plex.getPlexTranscodeSessions();
    killPlexTranscodeSession = (sessionKey: string) => this.plex.killPlexTranscodeSession(sessionKey);

    // Plex Friends and Sharing (Library Access Management)
    getPlexFriends = () => this.plex.getPlexFriends();
    invitePlexFriend = (request: import('../types/plex').PlexInviteFriendRequest) => this.plex.invitePlexFriend(request);
    removePlexFriend = (friendId: number) => this.plex.removePlexFriend(friendId);
    getPlexSharedServers = () => this.plex.getPlexSharedServers();
    sharePlexLibraries = (request: import('../types/plex').PlexShareLibrariesRequest) => this.plex.sharePlexLibraries(request);
    updatePlexSharing = (sharedServerId: number, request: import('../types/plex').PlexUpdateSharingRequest) => this.plex.updatePlexSharing(sharedServerId, request);
    revokePlexSharing = (sharedServerId: number) => this.plex.revokePlexSharing(sharedServerId);
    getPlexManagedUsers = () => this.plex.getPlexManagedUsers();
    createPlexManagedUser = (request: import('../types/plex').PlexCreateManagedUserRequest) => this.plex.createPlexManagedUser(request);
    deletePlexManagedUser = (userId: number) => this.plex.deletePlexManagedUser(userId);
    updatePlexManagedUser = (userId: number, request: import('../types/plex').PlexUpdateManagedUserRequest) => this.plex.updatePlexManagedUser(userId, request);
    getPlexLibrariesForSharing = () => this.plex.getPlexLibrariesForSharing();

    // ========================================================================
    // Detection (from DetectionAPI)
    // ========================================================================

    getDetectionAlerts = (filter?: DetectionAlertFilter) => this.detection.getDetectionAlerts(filter);
    getDetectionAlert = (id: number) => this.detection.getDetectionAlert(id);
    acknowledgeDetectionAlert = (id: number, acknowledgedBy: string) => this.detection.acknowledgeDetectionAlert(id, acknowledgedBy);
    getDetectionRules = () => this.detection.getDetectionRules();
    getDetectionRule = (ruleType: DetectionRuleType) => this.detection.getDetectionRule(ruleType);
    updateDetectionRule = (ruleType: DetectionRuleType, update: UpdateRuleRequest) => this.detection.updateDetectionRule(ruleType, update);
    setDetectionRuleEnabled = (ruleType: DetectionRuleType, enabled: boolean) => this.detection.setDetectionRuleEnabled(ruleType, enabled);
    getUserTrustScore = (userId: number) => this.detection.getUserTrustScore(userId);
    getLowTrustUsers = (threshold?: number) => this.detection.getLowTrustUsers(threshold);
    getDetectionMetrics = () => this.detection.getDetectionMetrics();
    getDetectionAlertStats = () => this.detection.getDetectionAlertStats();

    // ========================================================================
    // Dedupe Audit (from DedupeAPI)
    // ========================================================================

    getDedupeAuditStats = () => this.dedupe.getDedupeAuditStats();
    getDedupeAuditEntries = (filter?: DedupeAuditFilter) => this.dedupe.getDedupeAuditEntries(filter);
    confirmDedupeEntry = (id: string, request?: DedupeAuditActionRequest) => this.dedupe.confirmDedupeEntry(id, request);
    restoreDedupeEntry = (id: string, request?: DedupeAuditActionRequest) => this.dedupe.restoreDedupeEntry(id, request);
    getDedupeAuditExportUrl = (filter?: DedupeAuditFilter) => this.dedupe.getDedupeAuditExportUrl(filter);

    // ========================================================================
    // Audit Log (from AuditAPI)
    // ========================================================================

    getAuditEvents = (filter?: AuditEventFilter) => this.audit.getAuditEvents(filter);
    getAuditEvent = (id: string) => this.audit.getAuditEvent(id);
    getAuditStats = () => this.audit.getAuditStats();
    getAuditTypes = () => this.audit.getAuditTypes();
    getAuditSeverities = () => this.audit.getAuditSeverities();
    getAuditExportUrl = (format?: 'json' | 'cef', filter?: AuditEventFilter) => this.audit.getAuditExportUrl(format, filter);

    // ========================================================================
    // DLQ (from DLQAPI)
    // ========================================================================

    getDLQEntries = (filter?: DLQEntryFilter) => this.dlq.getDLQEntries(filter);
    getDLQEntry = (eventId: string) => this.dlq.getDLQEntry(eventId);
    deleteDLQEntry = (eventId: string) => this.dlq.deleteDLQEntry(eventId);
    retryDLQEntry = (eventId: string) => this.dlq.retryDLQEntry(eventId);
    retryAllDLQEntries = () => this.dlq.retryAllDLQEntries();
    getDLQStats = () => this.dlq.getDLQStats();
    getDLQCategories = () => this.dlq.getDLQCategories();
    cleanupDLQ = () => this.dlq.cleanupDLQ();

    // ========================================================================
    // WAL (from WALAPI)
    // ========================================================================

    getWALStats = () => this.wal.getWALStats();
    getWALHealth = () => this.wal.getWALHealth();
    triggerWALCompaction = () => this.wal.triggerWALCompaction();

    // ========================================================================
    // Cross-Platform (from CrossPlatformAPI)
    // ========================================================================

    createContentMapping = (request: ContentMappingRequest) => this.crossPlatform.createContentMapping(request);
    lookupContentMapping = (params: ContentLookupParams) => this.crossPlatform.lookupContentMapping(params);
    linkContentToPlex = (contentId: number, plexRatingKey: string) => this.crossPlatform.linkContentToPlex(contentId, plexRatingKey);
    linkContentToJellyfin = (contentId: number, jellyfinItemId: string) => this.crossPlatform.linkContentToJellyfin(contentId, jellyfinItemId);
    linkContentToEmby = (contentId: number, embyItemId: string) => this.crossPlatform.linkContentToEmby(contentId, embyItemId);
    createUserLink = (request: UserLinkRequest) => this.crossPlatform.createUserLink(request);
    deleteUserLink = (primaryUserId: number, linkedUserId: number) => this.crossPlatform.deleteUserLink(primaryUserId, linkedUserId);
    getSuggestedUserLinks = () => this.crossPlatform.getSuggestedUserLinks();
    getLinkedUsers = (userId: number) => this.crossPlatform.getLinkedUsers(userId);
    getCrossPlatformUserStats = (userId: number) => this.crossPlatform.getCrossPlatformUserStats(userId);
    getCrossPlatformContentStats = (contentId: number) => this.crossPlatform.getCrossPlatformContentStats(contentId);
    getCrossPlatformSummary = () => this.crossPlatform.getCrossPlatformSummary();

    // ========================================================================
    // Recommendations (from RecommendAPI)
    // ========================================================================

    getRecommendations = (request: RecommendationRequest) => this.recommend.getRecommendations(request);
    getSimilarItems = (itemId: number, k?: number) => this.recommend.getSimilarItems(itemId, k);
    getTrending = (k?: number) => this.recommend.getTrending(k);
    getRecommendTrainingStatus = () => this.recommend.getTrainingStatus();
    triggerRecommendTraining = () => this.recommend.triggerTraining();
    getRecommendConfig = () => this.recommend.getConfig();
    getRecommendMetrics = () => this.recommend.getMetrics();
    getRecommendUserPreferences = (userId: number) => this.recommend.getUserPreferences(userId);
    updateRecommendUserPreferences = (userId: number, prefs: Partial<UserPreference>) => this.recommend.updateUserPreferences(userId, prefs);
    recordRecommendFeedback = (userId: number, itemId: number, feedback: 'click' | 'watch' | 'skip' | 'dismiss') => this.recommend.recordFeedback(userId, itemId, feedback);
    getWhatsNext = (request: WhatsNextRequest) => this.recommend.getWhatsNext(request);
    getRecommendAlgorithms = () => this.recommend.getAlgorithms();
    getRecommendAlgorithmMetrics = () => this.recommend.getAlgorithmMetrics();

    // ========================================================================
    // Wrapped Reports (from WrappedAPI)
    // ========================================================================

    getWrappedServerStats = (year: number) => this.wrapped.getWrappedServerStats(year);
    getWrappedUserReport = (year: number, userID: number, generate?: boolean) => this.wrapped.getWrappedUserReport(year, userID, generate);
    getWrappedLeaderboard = (year: number, limit?: number) => this.wrapped.getWrappedLeaderboard(year, limit);
    getWrappedByShareToken = (token: string) => this.wrapped.getWrappedByShareToken(token);
    generateWrappedReports = (request: WrappedGenerateRequest) => this.wrapped.generateWrappedReports(request);

    // ========================================================================
    // Newsletter (from NewsletterAPI)
    // ========================================================================

    getNewsletterTemplates = (filter?: import('../types/newsletter').NewsletterFilter) => this.newsletter.getTemplates(filter);
    getNewsletterTemplate = (id: string) => this.newsletter.getTemplate(id);
    createNewsletterTemplate = (req: import('../types/newsletter').CreateTemplateRequest) => this.newsletter.createTemplate(req);
    updateNewsletterTemplate = (id: string, req: import('../types/newsletter').UpdateTemplateRequest) => this.newsletter.updateTemplate(id, req);
    deleteNewsletterTemplate = (id: string) => this.newsletter.deleteTemplate(id);
    previewNewsletterTemplate = (req: import('../types/newsletter').PreviewNewsletterRequest) => this.newsletter.previewTemplate(req);
    getNewsletterSchedules = (filter?: import('../types/newsletter').NewsletterFilter) => this.newsletter.getSchedules(filter);
    getNewsletterSchedule = (id: string) => this.newsletter.getSchedule(id);
    createNewsletterSchedule = (req: import('../types/newsletter').CreateScheduleRequest) => this.newsletter.createSchedule(req);
    updateNewsletterSchedule = (id: string, req: import('../types/newsletter').UpdateScheduleRequest) => this.newsletter.updateSchedule(id, req);
    deleteNewsletterSchedule = (id: string) => this.newsletter.deleteSchedule(id);
    triggerNewsletterSchedule = (id: string) => this.newsletter.triggerSchedule(id);
    enableNewsletterSchedule = (id: string) => this.newsletter.enableSchedule(id);
    disableNewsletterSchedule = (id: string) => this.newsletter.disableSchedule(id);
    getNewsletterDeliveries = (filter?: import('../types/newsletter').NewsletterFilter) => this.newsletter.getDeliveries(filter);
    getNewsletterDelivery = (id: string) => this.newsletter.getDelivery(id);
    getNewsletterStats = () => this.newsletter.getStats();
    getNewsletterAuditLog = (filter?: import('../types/newsletter').NewsletterAuditFilter) => this.newsletter.getAuditLog(filter);
    getNewsletterUserPreferences = () => this.newsletter.getUserPreferences();
    updateNewsletterUserPreferences = (prefs: Partial<import('../types/newsletter').NewsletterUserPreferences>) => this.newsletter.updateUserPreferences(prefs);
    unsubscribeFromNewsletter = () => this.newsletter.unsubscribe();

    // ========================================================================
    // Server Status (from ServerAPI) - ADR-0026 Phase 1
    // ========================================================================

    getServerStatus = () => this.server.listServers();
    testServerConnection = (request: import('../types/server').MediaServerTestRequest) => this.server.testConnection(request);
    triggerServerSync = (serverId: string, fullSync?: boolean) => this.server.triggerSync(serverId, fullSync);
    getServerCounts = () => this.server.getServerCounts();
    hasServerErrors = () => this.server.hasServerErrors();
    getServersNeedingAttention = () => this.server.getServersNeedingAttention();

    // ========================================================================
    // Server CRUD (from ServerAPI) - ADR-0026 Phase 4
    // ========================================================================

    createServer = (request: import('../types/server').CreateMediaServerRequest) => this.server.createServer(request);
    getServer = (serverId: string) => this.server.getServer(serverId);
    updateServer = (serverId: string, request: import('../types/server').UpdateMediaServerRequest) => this.server.updateServer(serverId, request);
    deleteServer = (serverId: string) => this.server.deleteServer(serverId);
    listDBServers = (platform?: string, enabledOnly?: boolean) => this.server.listDBServers(platform, enabledOnly);
    setServerEnabled = (serverId: string, enabled: boolean) => this.server.setServerEnabled(serverId, enabled);

    // ========================================================================
    // Data Sync (from SyncAPI)
    // ========================================================================

    startTautulliImport = (request?: import('../types/sync').TautulliImportRequest) => this.sync.startTautulliImport(request);
    getImportStatus = () => this.sync.getImportStatus();
    stopImport = () => this.sync.stopImport();
    clearImportProgress = () => this.sync.clearImportProgress();
    validateTautulliDatabase = (dbPath: string) => this.sync.validateDatabase(dbPath);
    startPlexHistoricalSync = (request?: import('../types/sync').PlexHistoricalRequest) => this.sync.startPlexHistoricalSync(request);
    getPlexHistoricalStatus = () => this.sync.getPlexHistoricalStatus();
    getSyncStatus = () => this.sync.getSyncStatus();
    isAnySyncRunning = () => this.sync.isAnySyncRunning();
    checkSyncAvailability = () => this.sync.checkSyncAvailability();
}
