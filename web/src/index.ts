// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import { API, LocationFilter, PlaybackEvent } from './lib/api';
import { createLogger } from './lib/logger';
import { MapManager } from './lib/map';
import { getRoleGuard } from './lib/auth/RoleGuard';
import { AuthContext } from './lib/auth/AuthContext';
// Lazy-loaded modules for code splitting (reduces initial bundle by ~60%)
// GlobeManagerDeckGL and ChartManager are loaded on-demand
import { StatsManager } from './lib/stats';
import { TimelineManager } from './lib/timeline';
import { WebSocketManager, SyncCompletedData, StatsUpdateData, PlexRealtimePlaybackData, PlexTranscodeSessionsData, BufferHealthUpdateData } from './lib/websocket';
import { ToastManager } from './lib/toast';
import { ActivityDashboardManager } from './lib/activity-dashboard';
import { ServerDashboardManager } from './lib/server-dashboard';
import { RecentlyAddedDashboardManager } from './lib/recently-added-dashboard';
import { WrappedDashboardManager } from './lib/wrapped-dashboard';
import { initializeOAuth } from './lib/auth/plex-oauth';
import { PlexMonitoringManager } from './lib/plex-monitoring';
import { FilterManager } from './lib/filters';
import { ThemeManager } from './app/ThemeManager';
import { ViewportManager } from './app/ViewportManager';
import { AuthenticationManager } from './app/AuthenticationManager';
import { NavigationManager } from './app/NavigationManager';
import { TimelineController } from './app/TimelineController';
import { DataExportManager } from './app/DataExportManager';
import { SidebarManager } from './app/SidebarManager';
import { InsightsManager } from './app/InsightsManager';
import { LibraryAnalyticsManager } from './app/LibraryAnalyticsManager';
import { UserProfileManager } from './app/UserProfileManager';
import { OnboardingManager } from './app/OnboardingManager';
import { SetupWizardManager } from './app/SetupWizardManager';
import { ProgressiveOnboardingManager } from './app/ProgressiveOnboardingManager';
import { KeyboardShortcutsManager } from './app/KeyboardShortcutsManager';
import { ConfirmationDialogManager } from './app/ConfirmationDialogManager';
import { DataFreshnessManager } from './app/DataFreshnessManager';
import { ErrorBoundaryManager } from './app/ErrorBoundaryManager';
import { FilterPresetManager } from './app/FilterPresetManager';
import { BackupRestoreManager } from './app/BackupRestoreManager';
import { ServerManagementManager } from './app/ServerManagementManager';
import { SettingsManager } from './app/SettingsManager';
import { HelpDocumentationManager } from './app/HelpDocumentationManager';
import { ChartTooltipManager } from './app/ChartTooltipManager';
import { QuickInsightsManager } from './app/QuickInsightsManager';
import { ComparativeInsightsManager } from './app/ComparativeInsightsManager';
import { CustomDateComparisonManager } from './app/CustomDateComparisonManager';
import { GeographicDrillDownManager } from './app/GeographicDrillDownManager';
import { StreamingGeoJSONManager } from './app/StreamingGeoJSONManager';
import { ChartTimelineAnimationManager } from './app/ChartTimelineAnimationManager';
import { AnomalyDetectionManager } from './app/AnomalyDetectionManager';
import { ChartDrillDownManager } from './app/ChartDrillDownManager';
import { ChartExportManager } from './app/ChartExportManager';
import { ChartMaximizeManager } from './app/ChartMaximizeManager';
import { MapKeyboardManager } from './app/MapKeyboardManager';
import { NotificationCenterManager } from './app/NotificationCenterManager';
import { DataCachingIndicatorManager } from './app/DataCachingIndicatorManager';
import { LoadingProgressManager } from './app/LoadingProgressManager';
import { EmptyStateActionsManager } from './app/EmptyStateActionsManager';
import { BreadcrumbNavigationManager } from './app/BreadcrumbNavigationManager';
// Cookie consent banner disabled - self-hosted app with no tracking analytics
// import { CookieConsentManager } from './app/CookieConsentManager';
import { DateRangeBrushManager } from './app/DateRangeBrushManager';
import { TableViewManager } from './app/TableViewManager';
import { RecentSearchesManager } from './app/RecentSearchesManager';
import { FilterHistoryManager } from './app/FilterHistoryManager';
import { SecurityAlertsManager } from './app/SecurityAlertsManager';
import { DetectionRulesManager } from './app/DetectionRulesManager';
import { DedupeAuditManager } from './app/DedupeAuditManager';
import { CrossPlatformManager } from './app/CrossPlatformManager';
import { DataGovernanceManager } from './app/DataGovernanceManager';
import { NewsletterManager } from './app/NewsletterManager';
import { LibraryDetailsManager } from './app/LibraryDetailsManager';
import { LibraryContentManager } from './app/LibraryContentManager';
import { UserIPDetailsManager } from './app/UserIPDetailsManager';
import { MetadataDeepDiveManager } from './app/MetadataDeepDiveManager';
import { RatingKeysManager } from './app/RatingKeysManager';
import { SearchManager } from './app/SearchManager';
import { StreamDataManager } from './app/StreamDataManager';
import {
    initializeAccessibilityAnnouncers,
    announceLoadingStart,
    announceLoadingComplete,
    announceLoadingError
} from './lib/accessibility';
import { SafeStorage } from './lib/utils/SafeStorage';
// Type-only imports for lazy-loaded modules
import type { GlobeManagerDeckGL } from './lib/globe-deckgl';
import type { ChartManager } from './lib/charts';
import type { WebSocketMessageData } from './lib/websocket';

/**
 * Window and Navigator interface extensions for E2E testing
 */
declare global {
    interface Window {
        __E2E_ERROR_BOUNDARY_THRESHOLD__?: number;
        __E2E_SHOW_CONFIRMATION_DIALOGS__?: boolean;
        __lazyLoadComplete?: boolean;
        __app?: App;
        mapManager?: MapManager | null;
        viewportManager?: ViewportManager | null;
    }
    interface Navigator {
        webdriver?: boolean;
    }
}

/**
 * Interface for managers that have a destroy method
 */
interface Destroyable {
    destroy(): void;
}

/**
 * Type guard to check if an object has a destroy method
 */
function hasDestroy(obj: unknown): obj is Destroyable {
    return obj !== null && typeof obj === 'object' && 'destroy' in obj && typeof (obj as Destroyable).destroy === 'function';
}

const logger = createLogger('App');

class App {
    private api: API;
    private loginContainer: HTMLElement | null = null;
    private appContainer: HTMLElement | null = null;

    // Core managers (critical for app functionality)
    private mapManager: MapManager | null = null;
    private globeManager: GlobeManagerDeckGL | null = null;
    private statsManager: StatsManager | null = null;
    private chartManager: ChartManager | null = null;
    private timelineManager: TimelineManager | null = null;
    private wsManager: WebSocketManager | null = null;
    private toastManager: ToastManager | null = null;
    private filterManager: FilterManager | null = null;

    // UI managers
    private themeManager: ThemeManager | null = null;
    private viewportManager: ViewportManager | null = null;
    private authManager: AuthenticationManager | null = null;
    private navigationManager: NavigationManager | null = null;
    private sidebarManager: SidebarManager | null = null;
    private timelineController: TimelineController | null = null;

    // Dashboard managers
    private activityDashboard: ActivityDashboardManager | null = null;
    private serverDashboard: ServerDashboardManager | null = null;
    private recentlyAddedDashboard: RecentlyAddedDashboardManager | null = null;
    private wrappedDashboard: WrappedDashboardManager | null = null;

    // Feature managers
    private plexMonitoringManager: PlexMonitoringManager | null = null;
    private insightsManager: InsightsManager | null = null;
    private libraryAnalyticsManager: LibraryAnalyticsManager | null = null;
    private userProfileManager: UserProfileManager | null = null;
    private setupWizardManager: SetupWizardManager | null = null;
    private onboardingManager: OnboardingManager | null = null;
    private progressiveOnboardingManager: ProgressiveOnboardingManager | null = null;
    private keyboardShortcutsManager: KeyboardShortcutsManager | null = null;
    private confirmationDialogManager: ConfirmationDialogManager | null = null;
    private dataFreshnessManager: DataFreshnessManager | null = null;
    private errorBoundaryManager: ErrorBoundaryManager | null = null;
    private filterPresetManager: FilterPresetManager | null = null;
    private backupRestoreManager: BackupRestoreManager | null = null;
    private serverManagementManager: ServerManagementManager | null = null;
    private settingsManager: SettingsManager | null = null;
    private helpDocumentationManager: HelpDocumentationManager | null = null;

    // Chart-related managers
    private chartTooltipManager: ChartTooltipManager | null = null;
    private chartDrillDownManager: ChartDrillDownManager | null = null;
    private chartExportManager: ChartExportManager | null = null;
    private chartMaximizeManager: ChartMaximizeManager | null = null;
    private chartTimelineAnimationManager: ChartTimelineAnimationManager | null = null;

    // Analytics managers
    private quickInsightsManager: QuickInsightsManager | null = null;
    private comparativeInsightsManager: ComparativeInsightsManager | null = null;
    private customDateComparisonManager: CustomDateComparisonManager | null = null;
    private geographicDrillDownManager: GeographicDrillDownManager | null = null;
    private anomalyDetectionManager: AnomalyDetectionManager | null = null;
    private streamingGeoJSONManager: StreamingGeoJSONManager | null = null;

    // Map/Globe keyboard and UI managers
    private mapKeyboardManager: MapKeyboardManager | null = null;
    private globeKeyboardManager: MapKeyboardManager | null = null;
    private dateRangeBrushManager: DateRangeBrushManager | null = null;

    // UI enhancement managers
    private notificationCenterManager: NotificationCenterManager | null = null;
    private dataCachingIndicatorManager: DataCachingIndicatorManager | null = null;
    private loadingProgressManager: LoadingProgressManager | null = null;
    private emptyStateActionsManager: EmptyStateActionsManager | null = null;
    private breadcrumbNavigationManager: BreadcrumbNavigationManager | null = null;
    private tableViewManager: TableViewManager | null = null;
    private recentSearchesManager: RecentSearchesManager | null = null;
    private filterHistoryManager: FilterHistoryManager | null = null;

    // Security & Detection managers (ADR-0020)
    private securityAlertsManager: SecurityAlertsManager | null = null;
    private detectionRulesManager: DetectionRulesManager | null = null;

    // Deduplication Audit manager (ADR-0022)
    private dedupeAuditManager: DedupeAuditManager | null = null;

    // Cross-Platform manager
    private crossPlatformManager: CrossPlatformManager | null = null;

    // Data Governance manager
    private dataGovernanceManager: DataGovernanceManager | null = null;
    private newsletterManager: NewsletterManager | null = null;

    // Library & Content managers (library details, collections, and playlists)
    private libraryDetailsManager: LibraryDetailsManager | null = null;
    private collectionsManager: LibraryContentManager | null = null;
    private playlistsManager: LibraryContentManager | null = null;
    private userIPDetailsManager: UserIPDetailsManager | null = null;

    // Metadata Deep-Dive manager
    private metadataDeepDiveManager: MetadataDeepDiveManager | null = null;

    // Rating Keys manager
    private ratingKeysManager: RatingKeysManager | null = null;

    // Search manager
    private searchManager: SearchManager | null = null;

    // Stream Data manager
    private streamDataManager: StreamDataManager | null = null;

    // @ts-expect-error - dataExportManager is used only for side effects (event listener setup in constructor)
    private dataExportManager: DataExportManager | null = null;

    // Interval/Timer IDs for cleanup on logout (prevents memory leaks)
    private serviceWorkerUpdateIntervalId: ReturnType<typeof setInterval> | null = null;
    private statsAutoRefreshIntervalId: ReturnType<typeof setInterval> | null = null;

    constructor() {
        this.api = new API('/api/v1');

        // Validate required DOM elements exist before proceeding
        if (!this.validateRequiredElements()) {
            return; // Error already displayed by showInitializationError()
        }

        // Initialize theme manager
        this.themeManager = new ThemeManager();

        // Cookie consent banner disabled - self-hosted app with no tracking analytics
        // To re-enable: uncomment import and initialization of CookieConsentManager

        // Initialize authentication manager
        this.authManager = new AuthenticationManager(
            this.api,
            () => this.showApp(),
            () => this.handleLogoutComplete()
        );

        this.init();
    }

    /**
     * Validate that all required DOM elements exist
     * Returns false and displays error if any critical elements are missing
     */
    private validateRequiredElements(): boolean {
        const loginContainer = document.getElementById('login-container');
        const appContainer = document.getElementById('app');

        const missingElements: string[] = [];

        if (!loginContainer) missingElements.push('login-container');
        if (!appContainer) missingElements.push('app');

        if (missingElements.length > 0) {
            logger.error('[App] Critical DOM elements missing:', missingElements);
            this.showInitializationError(missingElements);
            return false;
        }

        // Type-safe assignments after validation
        this.loginContainer = loginContainer;
        this.appContainer = appContainer;

        logger.debug('[App] All required DOM elements validated successfully');
        return true;
    }

    /**
     * Display a user-friendly error message when critical elements are missing
     */
    private showInitializationError(missingElements: string[]): void {
        // CSP-compliant error page (no inline event handlers)
        const errorHtml = `
            <style>
                .error-refresh-btn {
                    background: #e94560;
                    color: white;
                    border: none;
                    padding: 12px 24px;
                    border-radius: 8px;
                    font-size: 16px;
                    cursor: pointer;
                    transition: background 0.2s;
                }
                .error-refresh-btn:hover {
                    background: #ff6b6b;
                }
            </style>
            <div style="
                position: fixed;
                top: 0;
                left: 0;
                right: 0;
                bottom: 0;
                background: #1a1a2e;
                color: #eee;
                display: flex;
                align-items: center;
                justify-content: center;
                font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                padding: 20px;
            ">
                <div style="
                    max-width: 500px;
                    text-align: center;
                    background: #16213e;
                    padding: 40px;
                    border-radius: 12px;
                    box-shadow: 0 4px 20px rgba(0,0,0,0.3);
                ">
                    <h1 style="color: #e94560; margin: 0 0 20px 0; font-size: 24px;">
                        Application Error
                    </h1>
                    <p style="margin: 0 0 20px 0; line-height: 1.6;">
                        The application could not start because required page elements are missing.
                        This may be caused by a corrupted page load or browser extension interference.
                    </p>
                    <p style="
                        background: #0f3460;
                        padding: 15px;
                        border-radius: 8px;
                        font-family: monospace;
                        font-size: 12px;
                        margin: 0 0 20px 0;
                        text-align: left;
                        word-break: break-all;
                    ">
                        Missing elements: ${missingElements.join(', ')}
                    </p>
                    <button class="error-refresh-btn" id="error-refresh-btn">
                        Refresh Page
                    </button>
                </div>
            </div>
        `;
        document.body.innerHTML = errorHtml;

        // CSP-compliant click handler (replaces inline onclick)
        const refreshBtn = document.getElementById('error-refresh-btn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                location.reload();
            });
        }
    }



    private async init(): Promise<void> {
        // Initialize accessibility announcers for screen reader support
        initializeAccessibilityAnnouncers();

        // Register service worker for offline support and faster repeat visits
        this.registerServiceWorker();

        // Initialize OAuth system (handles OAuth callback if present)
        // Wrapped in try-catch to prevent app from failing if OAuth init fails
        try {
            await initializeOAuth();
        } catch (error) {
            logger.error('[App] OAuth initialization failed:', error);
            // Continue without OAuth - user can still log in with other methods
        }

        // Setup login listeners via authentication manager
        this.authManager?.setupLoginListeners();

        if (this.authManager?.isAuthenticated()) {
            this.showApp();
        } else {
            this.showLogin();
        }
    }

    /**
     * Register service worker for PWA functionality
     */
    private registerServiceWorker(): void {
        if ('serviceWorker' in navigator) {
            window.addEventListener('load', () => {
                navigator.serviceWorker
                    .register('/service-worker.js')
                    .then((registration) => {
                        logger.debug('[App] Service Worker registered successfully:', registration.scope);

                        // Check for updates periodically (every 1 hour)
                        // Store interval ID for cleanup on logout
                        this.serviceWorkerUpdateIntervalId = setInterval(() => {
                            registration.update();
                        }, 60 * 60 * 1000);

                        // Handle service worker updates
                        registration.addEventListener('updatefound', () => {
                            const newWorker = registration.installing;
                            if (newWorker) {
                                newWorker.addEventListener('statechange', () => {
                                    if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
                                        // New service worker available, show update notification
                                        if (this.toastManager) {
                                            this.toastManager.show({
                                                message: 'A new version is available! Refresh to update.',
                                                type: 'info',
                                                duration: 10000
                                            });
                                        }
                                    }
                                });
                            }
                        });
                    })
                    .catch((error) => {
                        logger.warn('[App] Service Worker registration failed:', error);
                        // Don't block app functionality if service worker fails
                    });
            });
        } else {
            logger.debug('[App] Service Worker not supported in this browser');
        }
    }


    private showLogin(): void {
        if (!this.loginContainer || !this.appContainer) return;

        this.loginContainer.classList.remove('hidden');
        this.appContainer.classList.add('hidden');
    }

    private async showApp(): Promise<void> {
        if (!this.loginContainer || !this.appContainer) return;

        this.loginContainer.classList.add('hidden');
        this.appContainer.classList.remove('hidden');

        if (!this.mapManager || !this.statsManager || !this.chartManager || !this.timelineManager) {
            // Wrap initialization in try/catch to ensure __lazyLoadComplete is always set
            // This prevents E2E tests from hanging indefinitely if initialization fails
            try {
                // Initialize managers in logical groups for better maintainability
                this.initializeCoreManagers();
                this.initializeUIEnhancementManagers();
                this.initializeTimelineAndViewport();
                this.initializeFilterAndSearchManagers();
                this.initializeDashboardManagers();
                this.initializeNavigationManagers();
                this.initializeSecurityManagers();
                this.initializeGovernanceManagers();
                this.initializeChartManagers();
                this.initializeAnalyticsManagers();

                // CRITICAL: Setup listeners BEFORE any await statements to ensure button handlers
                // are registered immediately. This fixes E2E test race conditions where tests
                // clicked buttons before handlers were set up.
                this.setupWebSocket();
                this.setupAppListeners();

                // Initialize async managers
                await this.initializeAsyncManagers();

                // Initialize UX enhancement managers
                this.initializeUXManagers();
                this.initializeProfileManagers();
                this.initializeNavigationCallbacks();
                this.initializeContentManagers();

                // Setup sub-tabs and modals
                this.setupLibrarySubTabs();
                this.setupContentSubTabs();
                this.setupTautulliSubTabs();
                this.setupIPHistoryModal();

                // Initialize onboarding
                this.initializeOnboarding();

                // Load heavy dependencies in parallel
                await this.loadLazyModules();

                // CRITICAL: Set __lazyLoadComplete IMMEDIATELY after lazy loading completes
                window.__lazyLoadComplete = true;

                // RBAC Phase 4: Apply role-based UI visibility
                this.applyRoleBasedVisibility();
                this.navigationManager?.applyRoleBasedTabVisibility();

                // Initialize TableViewManager after charts are available
                setTimeout(() => {
                    this.tableViewManager = new TableViewManager();
                }, 100);

                await this.filterManager!.loadFiltersData();
                this.filterManager!.loadDataFromURL();

                // Record initial filter state for undo/redo
                if (this.filterHistoryManager) {
                    this.filterHistoryManager.recordState(this.filterManager!.buildFilter());
                }

                this.loadData();
                this.startAutoRefresh();
            } catch (error) {
                // Log the error but don't crash the app
                logger.error('[App] Initialization error:', error);

                // Show user-friendly error message if toast manager is available
                if (this.toastManager) {
                    this.toastManager.error(
                        'Application initialization failed. Some features may not work correctly.',
                        'Initialization Error',
                        10000
                    );
                }

                // Still set __lazyLoadComplete on error to prevent E2E tests from hanging
                window.__lazyLoadComplete = true;
            }
        }
    }

    /**
     * Initialize core managers (map, stats, toast, websocket)
     */
    private initializeCoreManagers(): void {
        // Create critical managers FIRST (synchronously) so they're immediately available
        this.mapManager = new MapManager('map', () => this.viewportManager?.onMapModeChange());

        // Initialize map keyboard navigation
        setTimeout(() => {
            if (this.mapManager) {
                const map = this.mapManager.getMap();
                if (map) {
                    this.mapKeyboardManager = new MapKeyboardManager({
                        containerId: 'map',
                        map: map,
                        type: 'map'
                    });
                    this.mapKeyboardManager.init();
                }
            }
        }, 500);

        // Set up hexagon data request callback
        this.mapManager.setHexagonDataRequestCallback((resolution: number) => {
            this.loadHexagonData(resolution);
        });

        // Set up arc data request callback
        this.mapManager.setArcDataRequestCallback(() => {
            this.loadArcData();
        });

        this.statsManager = new StatsManager(this.api);
        this.toastManager = new ToastManager();
        this.wsManager = new WebSocketManager();
    }

    /**
     * Initialize UI enhancement managers (notifications, caching, loading, empty states)
     */
    private initializeUIEnhancementManagers(): void {
        // Initialize notification center manager (toast notification history)
        this.notificationCenterManager = new NotificationCenterManager();
        this.notificationCenterManager.init();
        if (this.toastManager) {
            this.toastManager.setNotificationCallback((type, message, title) => {
                this.notificationCenterManager?.addNotification(type, message, title);
            });
        }

        // Initialize data caching indicator manager (displays cache status)
        this.dataCachingIndicatorManager = new DataCachingIndicatorManager();
        this.dataCachingIndicatorManager.init();
        this.api.setCacheStatusCallback((cached, queryTimeMs) => {
            this.dataCachingIndicatorManager?.updateStatus(cached, queryTimeMs);
        });

        // Initialize loading progress manager (displays loading progress bars)
        this.loadingProgressManager = new LoadingProgressManager();
        this.loadingProgressManager.init();

        // Initialize empty state actions manager (provides actions for empty states)
        this.emptyStateActionsManager = new EmptyStateActionsManager();
        this.emptyStateActionsManager.init({
            navigateTo: (view: string) => this.handleEmptyStateNavigation(view),
            refreshData: () => this.loadData(),
            openHelp: () => this.helpDocumentationManager?.open(),
            selectLibrary: () => {},
            selectUser: () => {}
        });

        // Initialize breadcrumb navigation manager (breadcrumb trail navigation)
        this.breadcrumbNavigationManager = new BreadcrumbNavigationManager();
        this.breadcrumbNavigationManager.init({
            navigateToDashboard: (view: string) => {
                const validViews = ['maps', 'activity', 'analytics', 'recently-added', 'server'];
                if (validViews.includes(view)) {
                    this.navigationManager?.switchDashboardView(view as 'maps' | 'activity' | 'analytics' | 'recently-added' | 'server');
                }
            },
            navigateToAnalyticsPage: (page: string) => {
                this.navigationManager?.switchAnalyticsPage(page as 'overview' | 'content' | 'users' | 'performance' | 'geographic' | 'advanced' | 'library' | 'users-profile' | 'tautulli');
            }
        });
    }

    /**
     * Handle navigation from empty state actions
     */
    private handleEmptyStateNavigation(view: string): void {
        const viewMap: Record<string, () => void> = {
            'map': () => this.navigationManager?.switchDashboardView('maps'),
            'maps': () => this.navigationManager?.switchDashboardView('maps'),
            'analytics': () => this.navigationManager?.switchDashboardView('analytics'),
            'analytics-overview': () => this.navigationManager?.switchDashboardView('analytics'),
            'analytics-users': () => {
                this.navigationManager?.switchDashboardView('analytics');
                this.navigationManager?.switchAnalyticsPage('users');
            },
            'activity': () => this.navigationManager?.switchDashboardView('activity')
        };
        viewMap[view]?.();
    }

    /**
     * Initialize timeline and viewport managers
     */
    private initializeTimelineAndViewport(): void {
        // Initialize timeline controller BEFORE timeline manager (for callbacks)
        this.timelineController = new TimelineController(null);

        this.timelineManager = new TimelineManager(
            this.api,
            (currentTime, playbacks, totalCount) => this.timelineController?.handleTimelineUpdate(currentTime, playbacks, totalCount),
            (isPlaying) => this.timelineController?.handlePlayStateChange(isPlaying)
        );

        this.timelineController.setTimelineManager(this.timelineManager);

        // Initialize viewport manager (handles WebGL detection, 2D/3D switching)
        this.viewportManager = new ViewportManager(this.mapManager!);
        this.viewportManager.setToastManager(this.toastManager!);

        // Initialize Plex monitoring manager
        this.plexMonitoringManager = new PlexMonitoringManager(this.toastManager!, this.statsManager!);
    }

    /**
     * Initialize filter and search managers
     */
    private initializeFilterAndSearchManagers(): void {
        this.filterManager = new FilterManager(this.api);

        // Initialize FilterHistoryManager for undo/redo functionality
        this.filterHistoryManager = new FilterHistoryManager();
        this.filterHistoryManager.setRestoreCallback((filter) => {
            this.filterManager?.restoreFilterState(filter);
        });

        // Set filter change callback - records to history then loads data
        this.filterManager.setFilterChangeCallback(() => {
            if (this.filterHistoryManager && this.filterManager) {
                this.filterHistoryManager.recordState(this.filterManager.buildFilter());
            }
            return this.loadData();
        });

        // Initialize RecentSearchesManager for quick access to recent filter selections
        this.recentSearchesManager = new RecentSearchesManager((type, value) => {
            const filterElementMap: Record<string, string> = {
                'user': 'filter-users',
                'mediaType': 'filter-media-types'
            };
            const elementId = filterElementMap[type];
            if (elementId) {
                const element = document.getElementById(elementId) as HTMLSelectElement;
                if (element) {
                    element.value = value;
                    this.filterManager?.onFilterChange();
                }
            }
        });

        // Initialize data export manager (handles CSV and GeoJSON exports)
        this.dataExportManager = new DataExportManager(this.filterManager, this.toastManager!);
    }

    /**
     * Initialize dashboard managers
     */
    private initializeDashboardManagers(): void {
        this.activityDashboard = new ActivityDashboardManager(this.api);
        if (this.toastManager) {
            this.activityDashboard.setToastManager(this.toastManager);
        }
        this.serverDashboard = new ServerDashboardManager(this.api);
        this.recentlyAddedDashboard = new RecentlyAddedDashboardManager(this.api);
        this.wrappedDashboard = new WrappedDashboardManager(this.api);
        if (this.toastManager) {
            this.wrappedDashboard.setToastManager(this.toastManager);
        }
    }

    /**
     * Initialize navigation and sidebar managers
     */
    private initializeNavigationManagers(): void {
        // Initialize navigation manager IMMEDIATELY (before async operations)
        this.navigationManager = new NavigationManager(
            this.mapManager!,
            null, // globeManager - set after lazy load
            null, // chartManager - set after lazy load
            this.viewportManager!,
            this.activityDashboard!,
            this.serverDashboard!,
            this.recentlyAddedDashboard!,
            this.toastManager!
        );

        if (this.breadcrumbNavigationManager) {
            this.navigationManager.setBreadcrumbManager(this.breadcrumbNavigationManager);
        }

        // Initialize sidebar manager (hamburger menu and collapse toggle)
        this.sidebarManager = new SidebarManager();
        this.sidebarManager.init();

        // Initialize keyboard shortcuts manager
        this.keyboardShortcutsManager = new KeyboardShortcutsManager();
        this.keyboardShortcutsManager.init();

        // Initialize help documentation manager
        this.helpDocumentationManager = new HelpDocumentationManager();
        this.helpDocumentationManager.init();
    }

    /**
     * Initialize security and detection managers
     */
    private initializeSecurityManagers(): void {
        // Initialize security alerts manager (ADR-0020: Detection Engine)
        this.securityAlertsManager = new SecurityAlertsManager(this.api);
        this.securityAlertsManager.init();

        // Initialize detection rules manager (ADR-0020: Detection Engine)
        this.detectionRulesManager = new DetectionRulesManager(this.api);
        this.detectionRulesManager.init();

        // Initialize dedupe audit manager (ADR-0022: Deduplication Audit)
        this.dedupeAuditManager = new DedupeAuditManager(this.api);
        this.dedupeAuditManager.init();
    }

    /**
     * Initialize governance managers (cross-platform, data governance, newsletter)
     */
    private initializeGovernanceManagers(): void {
        // Initialize cross-platform manager (cross-platform linking)
        this.crossPlatformManager = new CrossPlatformManager(this.api);

        // Initialize data governance manager
        this.dataGovernanceManager = new DataGovernanceManager(this.api);

        // Initialize newsletter manager
        this.newsletterManager = new NewsletterManager(this.api);
    }

    /**
     * Initialize chart-related managers
     */
    private initializeChartManagers(): void {
        // Initialize chart tooltip manager
        this.chartTooltipManager = new ChartTooltipManager();
    }

    /**
     * Initialize analytics managers
     */
    private initializeAnalyticsManagers(): void {
        // Initialize quick insights manager (automated insights)
        this.quickInsightsManager = new QuickInsightsManager(this.api);
        this.quickInsightsManager.init();

        // Initialize comparative insights manager
        this.comparativeInsightsManager = new ComparativeInsightsManager(this.api);
        this.comparativeInsightsManager.init();

        // Initialize custom date comparison manager
        this.customDateComparisonManager = new CustomDateComparisonManager(this.api);
        this.customDateComparisonManager.init();
        if (this.comparativeInsightsManager) {
            this.customDateComparisonManager.setOnComparisonLoad((data) => {
                this.comparativeInsightsManager?.updateWithData(data);
            });
        }
    }

    /**
     * Initialize async managers that require await
     */
    private async initializeAsyncManagers(): Promise<void> {
        // Initialize geographic drill-down manager
        this.geographicDrillDownManager = new GeographicDrillDownManager(this.api);
        await this.geographicDrillDownManager.init();

        // Initialize streaming GeoJSON manager
        this.streamingGeoJSONManager = new StreamingGeoJSONManager(this.api);
        this.streamingGeoJSONManager.init();

        // Initialize chart timeline animation manager
        this.chartTimelineAnimationManager = new ChartTimelineAnimationManager(this.api);
        await this.chartTimelineAnimationManager.init();

        // Initialize anomaly detection manager
        this.anomalyDetectionManager = new AnomalyDetectionManager(this.api);
        this.anomalyDetectionManager.init();
    }

    /**
     * Initialize UX enhancement managers
     */
    private initializeUXManagers(): void {
        // Initialize chart drill-down manager
        this.chartDrillDownManager = new ChartDrillDownManager();
        if (this.toastManager) {
            this.chartDrillDownManager.setToastManager(this.toastManager);
        }
        if (this.filterManager) {
            this.chartDrillDownManager.setFilterManager(this.filterManager);
        }
        this.chartDrillDownManager.init();

        // Initialize chart export manager
        this.chartExportManager = new ChartExportManager();
        if (this.toastManager) {
            this.chartExportManager.setToastManager(this.toastManager);
        }
        this.chartExportManager.init();

        // Initialize chart maximize manager
        this.chartMaximizeManager = new ChartMaximizeManager();
        this.chartMaximizeManager.init();

        // Initialize confirmation dialog manager
        this.confirmationDialogManager = new ConfirmationDialogManager();
        this.confirmationDialogManager.init();

        if (this.activityDashboard && this.confirmationDialogManager) {
            this.activityDashboard.setConfirmationManager(this.confirmationDialogManager);
        }

        // Initialize data freshness manager (displays data age and refresh controls)
        this.dataFreshnessManager = new DataFreshnessManager();
        this.dataFreshnessManager.setRefreshCallback(() => this.loadData());
        this.dataFreshnessManager.init();

        // Initialize error boundary manager (error recovery UI)
        const e2eThreshold = window.__E2E_ERROR_BOUNDARY_THRESHOLD__;
        const errorBoundaryConfig = e2eThreshold !== undefined ? { overlayThreshold: e2eThreshold } : {};
        this.errorBoundaryManager = new ErrorBoundaryManager(errorBoundaryConfig);
        this.errorBoundaryManager.setRetryCallback(async () => {
            await this.loadData();
        });
        this.errorBoundaryManager.init();

        // Initialize filter preset manager (save and load filter presets)
        this.filterPresetManager = new FilterPresetManager();
        if (this.filterManager) {
            this.filterPresetManager.setFilterManager(this.filterManager);
        }
        if (this.toastManager) {
            this.filterPresetManager.setToastManager(this.toastManager);
        }
        this.filterPresetManager.init();

        // Initialize backup/restore manager (backup and restore UI)
        this.backupRestoreManager = new BackupRestoreManager(this.api);
        if (this.toastManager) {
            this.backupRestoreManager.setToastManager(this.toastManager);
        }
        if (this.confirmationDialogManager) {
            this.backupRestoreManager.setConfirmationManager(this.confirmationDialogManager);
        }
        this.backupRestoreManager.init();

        // Initialize server management manager (server management UI)
        this.serverManagementManager = new ServerManagementManager(this.api);
        if (this.toastManager) {
            this.serverManagementManager.setToastManager(this.toastManager);
        }
        this.serverManagementManager.init();

        // Initialize settings manager (settings and preferences page)
        this.settingsManager = new SettingsManager();
        if (this.toastManager) {
            this.settingsManager.setToastManager(this.toastManager);
        }
        if (this.themeManager) {
            this.settingsManager.setThemeManager(this.themeManager);
        }
        this.settingsManager.init();
    }

    /**
     * Initialize profile managers (insights, library analytics, user profile)
     */
    private initializeProfileManagers(): void {
        // Initialize insights manager (automated insights)
        this.insightsManager = new InsightsManager(this.api);
        this.insightsManager.init('insights-panel');

        // Initialize library analytics manager (library deep-dive analytics)
        this.libraryAnalyticsManager = new LibraryAnalyticsManager(this.api);
        if (this.filterManager) {
            this.libraryAnalyticsManager.setFilterManager(this.filterManager);
        }

        // Initialize user profile manager (user profile analytics)
        this.userProfileManager = new UserProfileManager(this.api);
        if (this.filterManager) {
            this.userProfileManager.setFilterManager(this.filterManager);
        }
    }

    /**
     * Set up navigation callbacks for lazy initialization
     */
    private initializeNavigationCallbacks(): void {
        if (!this.navigationManager) return;

        this.navigationManager.setLibraryPageShowCallback(() => {
            this.libraryAnalyticsManager?.init();
        });

        this.navigationManager.setUserProfilePageShowCallback(() => {
            this.userProfileManager?.init();
        });

        this.navigationManager.setCrossPlatformShowCallback(() => {
            if (this.crossPlatformManager && !this.crossPlatformManager.isInitialized()) {
                this.crossPlatformManager.init('cross-platform-container');
            }
        });

        this.navigationManager.setDataGovernanceShowCallback(() => {
            this.dataGovernanceManager?.init();
        });

        this.navigationManager.setNewsletterShowCallback(() => {
            this.newsletterManager?.init();
        });

        this.navigationManager.setWrappedPageShowCallback(() => {
            this.wrappedDashboard?.init();
        });
    }

    /**
     * Initialize content managers (library details, collections, playlists, etc.)
     */
    private initializeContentManagers(): void {
        const toastManagerOrUndefined = this.toastManager || undefined;

        // Initialize Library Details Manager (library details table)
        this.libraryDetailsManager = new LibraryDetailsManager({
            api: this.api,
            containerId: 'library-details-container',
            toastManager: toastManagerOrUndefined
        });

        // Initialize Library Content Managers (collections and playlists)
        this.collectionsManager = new LibraryContentManager({
            api: this.api,
            containerId: 'collections-container',
            toastManager: toastManagerOrUndefined
        });

        this.playlistsManager = new LibraryContentManager({
            api: this.api,
            containerId: 'playlists-container',
            toastManager: toastManagerOrUndefined
        });

        // Initialize User IP Details Manager (user IP history)
        this.userIPDetailsManager = new UserIPDetailsManager({
            api: this.api,
            containerId: 'user-ip-details-container',
            toastManager: toastManagerOrUndefined
        });

        // Initialize Metadata Deep-Dive Manager (metadata deep-dive)
        this.metadataDeepDiveManager = new MetadataDeepDiveManager({
            api: this.api,
            containerId: 'metadata-deep-dive-container',
            toastManager: toastManagerOrUndefined
        });

        // Initialize Rating Keys Manager (rating keys lookup)
        this.ratingKeysManager = new RatingKeysManager({
            api: this.api,
            containerId: 'rating-keys-container',
            toastManager: toastManagerOrUndefined
        });

        // Initialize Search Manager (media search)
        this.searchManager = new SearchManager({
            api: this.api,
            containerId: 'search-container',
            toastManager: toastManagerOrUndefined,
            onSelectResult: (ratingKey: string) => {
                this.metadataDeepDiveManager?.loadMetadata(ratingKey);
            }
        });

        // Initialize Stream Data Manager (stream data details)
        this.streamDataManager = new StreamDataManager({
            api: this.api,
            containerId: 'stream-data-container',
            toastManager: toastManagerOrUndefined
        });
    }

    /**
     * Initialize onboarding experience
     */
    private initializeOnboarding(): void {
        // Initialize setup wizard manager (first-time setup experience)
        this.setupWizardManager = new SetupWizardManager();
        this.onboardingManager = new OnboardingManager();

        // Initialize setup wizard and onboarding after a small delay to ensure UI is ready
        setTimeout(async () => {
            if (this.setupWizardManager) {
                this.setupWizardManager.init({
                    onComplete: () => {
                        this.onboardingManager?.init();
                    },
                    onStartTour: () => {
                        this.onboardingManager?.init();
                        this.onboardingManager?.startTour();
                    },
                    onTriggerSync: async () => {
                        try {
                            await this.api.triggerSync();
                            this.toastManager?.success('Data sync started successfully', 'Sync Initiated');
                        } catch (error) {
                            logger.error('[SetupWizard] Failed to trigger sync:', error);
                            this.toastManager?.error('Failed to start data sync. Please try again.', 'Sync Error');
                        }
                    }
                });

                const wasShown = await this.setupWizardManager.show();
                if (!wasShown) {
                    this.onboardingManager?.init();
                }
            } else {
                this.onboardingManager?.init();
            }
        }, 500);

        // Setup help tour button
        document.getElementById('help-tour-btn')?.addEventListener('click', () => {
            this.onboardingManager?.restartTour();
        });

        // Initialize progressive onboarding manager (Phase 3: Contextual tips)
        this.progressiveOnboardingManager = new ProgressiveOnboardingManager();
        setTimeout(() => {
            this.progressiveOnboardingManager?.init();
        }, 1500);
    }

    /**
     * Load heavy dependencies lazily
     */
    private async loadLazyModules(): Promise<void> {
        const LAZY_LOAD_TIMEOUT_MS = 20000;

        const lazyLoadPromise = Promise.all([
            this.loadGlobeModule(),
            this.loadChartModule()
        ]);

        let timedOut = false;
        const timeoutPromise = new Promise<void>((resolve) => {
            setTimeout(() => {
                timedOut = true;
                logger.warn(`[App] Lazy loading timed out after ${LAZY_LOAD_TIMEOUT_MS}ms. App may have reduced functionality.`);
                resolve();
            }, LAZY_LOAD_TIMEOUT_MS);
        });

        await Promise.race([lazyLoadPromise, timeoutPromise]).catch(error => {
            logger.error('[App] Failed to load heavy dependencies:', error);
            if (this.toastManager) {
                this.toastManager.error('Some features failed to load. Please refresh.', 'Error', 5000);
            }
        });

        if (timedOut) {
            this.reportMissingModules();
        }
    }

    /**
     * Load globe module lazily
     */
    private async loadGlobeModule(): Promise<void> {
        const { GlobeManagerDeckGL } = await import('./lib/globe-deckgl');
        this.globeManager = new GlobeManagerDeckGL('globe');

        if (this.themeManager) {
            this.themeManager.setGlobeManager(this.globeManager);
        }
        if (this.viewportManager) {
            this.viewportManager.setGlobeManager(this.globeManager);
        }
        if (this.navigationManager) {
            this.navigationManager.setGlobeManager(this.globeManager);
        }

        setTimeout(() => {
            if (this.globeManager) {
                this.globeManager.initialize();
                setTimeout(() => {
                    if (this.globeManager) {
                        const globeMap = this.globeManager.getMap();
                        if (globeMap) {
                            this.globeKeyboardManager = new MapKeyboardManager({
                                containerId: 'globe',
                                map: globeMap,
                                type: 'globe'
                            });
                            this.globeKeyboardManager.init();
                        }
                    }
                }, 300);
            }
        }, 100);
    }

    /**
     * Load chart module lazily
     */
    private async loadChartModule(): Promise<void> {
        const { ChartManager } = await import('./lib/charts');
        this.chartManager = new ChartManager(this.api);

        if (this.navigationManager) {
            this.navigationManager.setChartManager(this.chartManager);
        }
        if (this.themeManager) {
            this.themeManager.setChartManager(this.chartManager);
        }

        // Initialize DateRangeBrushManager for brush date selection
        if (this.filterManager) {
            this.dateRangeBrushManager = new DateRangeBrushManager(
                (start: string, end: string) => {
                    this.filterManager?.setDateRange(start, end);
                }
            );
            this.chartManager.onChartReady('chart-trends', (chart) => {
                this.dateRangeBrushManager?.registerChart('chart-trends', chart);
            });
            this.chartManager.onChartReady('chart-bandwidth-trends', (chart) => {
                this.dateRangeBrushManager?.registerChart('chart-bandwidth-trends', chart);
            });
        }
    }

    /**
     * Report missing modules after timeout
     */
    private reportMissingModules(): void {
        const missingModules: string[] = [];
        if (!this.chartManager) missingModules.push('ChartManager');
        if (!this.globeManager) missingModules.push('GlobeManager');

        if (missingModules.length > 0) {
            logger.error(`[App] Critical modules failed to load: ${missingModules.join(', ')}`);
            if (this.toastManager) {
                this.toastManager.error(
                    `Failed to load: ${missingModules.join(', ')}. Some features will be unavailable.`,
                    'Module Load Error',
                    10000
                );
            }
        }
    }


    private setupWebSocket(): void {
        if (!this.wsManager || !this.toastManager || !this.plexMonitoringManager) return;

        this.wsManager.onPlayback((event: PlaybackEvent) => {
            this.handleNewPlayback(event);
        });

        this.wsManager.onSyncCompleted((data: SyncCompletedData) => {
            this.handleSyncCompleted(data);
        });

        this.wsManager.onStatsUpdate((data: StatsUpdateData) => {
            this.handleStatsUpdate(data);
        });

        // Delegate Plex monitoring to PlexMonitoringManager (refactored v1.44)
        this.wsManager.onPlexRealtimePlayback((data: PlexRealtimePlaybackData) => {
            this.plexMonitoringManager!.handlePlexRealtimePlayback(data);
        });

        this.wsManager.onPlexTranscodeSessions((data: PlexTranscodeSessionsData) => {
            this.plexMonitoringManager!.handlePlexTranscodeSessions(data);
        });

        this.wsManager.onBufferHealthUpdate((data: BufferHealthUpdateData) => {
            this.plexMonitoringManager!.handleBufferHealthUpdate(data);
        });

        this.wsManager.connect();
    }

    private handleNewPlayback(event: PlaybackEvent): void {
        if (!this.toastManager) return;

        const username = event.username || 'Unknown user';
        const title = event.title || 'Unknown title';
        const mediaType = event.media_type || 'media';

        this.toastManager.info(
            `${username} is watching ${mediaType}: "${title}"`,
            'New Playback',
            7000
        );

        this.statsManager?.loadStats();
    }

    private handleSyncCompleted(data: SyncCompletedData): void {
        if (!this.toastManager) return;

        // Show toast notification if new playbacks were added
        if (data.new_playbacks > 0) {
            const message = data.new_playbacks === 1
                ? `1 new playback synced in ${data.sync_duration_ms}ms`
                : `${data.new_playbacks} new playbacks synced in ${data.sync_duration_ms}ms`;

            this.toastManager.success(message, 'Data Updated', 5000);

            // Refresh all data views
            this.refreshAllData();
        }
    }

    private handleStatsUpdate(_data: StatsUpdateData): void {
        // Stats update is less intrusive - we can silently refresh if needed
        // For now, we rely on sync_completed to trigger refreshes
    }


    private async refreshAllData(): Promise<void> {
        const filter = this.filterManager!.buildFilter();

        // Refresh stats
        this.statsManager?.loadStats();

        // Refresh charts
        this.chartManager?.loadAllCharts(filter);

        // Refresh map/globe (reload locations)
        await this.loadLocationData(filter);

        // Note: Timeline will be refreshed when user interacts with it
    }

    private setupAppListeners(): void {
        // Split listener setup into smaller grouped methods for maintainability
        this.setupFilterListeners();
        this.setupMapModeListeners();
        this.setupViewModeListeners();
        this.setupGlobalActionListeners();
    }

    /**
     * Setup filter-related event listeners
     */
    private setupFilterListeners(): void {
        // Filter change listeners - use array to reduce repetition
        const filterIds = ['filter-days', 'filter-start-date', 'filter-end-date', 'filter-users', 'filter-media-types'];
        for (const id of filterIds) {
            const element = document.getElementById(id);
            if (element) {
                element.addEventListener('change', () => this.filterManager!.onFilterChange());
            }
        }

        const btnClearFilters = document.getElementById('btn-clear-filters') as HTMLButtonElement;
        if (btnClearFilters) {
            btnClearFilters.addEventListener('click', async () => {
                // Skip confirmation dialog in E2E test environment (navigator.webdriver is set by Playwright)
                // UNLESS the test explicitly opts-in by setting window.__E2E_SHOW_CONFIRMATION_DIALOGS__ = true
                // This allows confirmation dialog tests to test actual dialog behavior
                const isAutomatedTest = navigator.webdriver === true;
                const forceShowDialog = window.__E2E_SHOW_CONFIRMATION_DIALOGS__ === true;
                const skipDialog = isAutomatedTest && !forceShowDialog;

                // Show confirmation dialog for Clear Filters action (skip in E2E tests unless opted-in)
                if (this.confirmationDialogManager && !skipDialog) {
                    const confirmed = await this.confirmationDialogManager.show({
                        title: 'Clear Filters',
                        message: 'Are you sure you want to clear all filters? This will reset to the default 90-day view.',
                        confirmText: 'Clear Filters',
                        cancelText: 'Cancel'
                    });
                    if (confirmed) {
                        this.filterManager!.clearFilters();
                    }
                } else {
                    // Fallback if confirmation manager not available or skipping dialog in E2E test mode
                    this.filterManager!.clearFilters();
                }
            });
        }

        // Quick date picker buttons
        this.filterManager!.setupQuickDateButtons();

        // Comparative analytics selectors
        const comparisonTypeSelector = document.getElementById('comparison-type-selector') as HTMLSelectElement;
        if (comparisonTypeSelector) {
            comparisonTypeSelector.addEventListener('change', () => this.loadData());
        }

        const temporalIntervalSelector = document.getElementById('temporal-interval-selector') as HTMLSelectElement;
        if (temporalIntervalSelector) {
            temporalIntervalSelector.addEventListener('change', () => this.loadData());
        }
    }

    /**
     * Setup map mode related event listeners
     */
    private setupMapModeListeners(): void {
        // Map mode buttons - use object map to reduce repetition
        const mapModeButtons: Record<string, 'points' | 'clusters' | 'heatmap' | 'hexagons'> = {
            'map-mode-points': 'points',
            'map-mode-clusters': 'clusters',
            'map-mode-heatmap': 'heatmap',
            'map-mode-hexagons': 'hexagons'
        };

        for (const [id, mode] of Object.entries(mapModeButtons)) {
            const btn = document.getElementById(id) as HTMLButtonElement;
            if (btn) {
                btn.addEventListener('click', () => this.viewportManager?.setMapMode(mode));
            }
        }

        // H3 Resolution selector
        const hexagonResolutionSelector = document.getElementById('hexagon-resolution') as HTMLSelectElement;
        if (hexagonResolutionSelector) {
            hexagonResolutionSelector.addEventListener('change', () => {
                const resolution = parseInt(hexagonResolutionSelector.value, 10);
                this.mapManager?.setHexagonResolution(resolution);
            });
        }

        // Arc overlay toggle
        const arcToggle = document.getElementById('arc-toggle') as HTMLButtonElement;
        if (arcToggle) {
            arcToggle.addEventListener('click', () => {
                if (!this.mapManager) return;
                const enabled = !this.mapManager.isArcOverlayEnabled();
                this.mapManager.setArcOverlayEnabled(enabled);
                arcToggle.classList.toggle('active', enabled);
                arcToggle.classList.toggle('enabled', enabled);
                arcToggle.setAttribute('aria-pressed', String(enabled));
            });
        }

        // Fullscreen toggle
        const fullscreenToggle = document.getElementById('fullscreen-toggle') as HTMLButtonElement;
        if (fullscreenToggle) {
            fullscreenToggle.addEventListener('click', () => this.toggleFullscreen(fullscreenToggle));

            // Listen for fullscreen change events to update button state
            document.addEventListener('fullscreenchange', () => {
                const isFullscreen = !!document.fullscreenElement;
                fullscreenToggle.classList.toggle('active', isFullscreen);
                fullscreenToggle.setAttribute('aria-label', isFullscreen ? 'Exit fullscreen mode' : 'Enter fullscreen mode');
                fullscreenToggle.setAttribute('title', isFullscreen ? 'Exit fullscreen' : 'Toggle fullscreen mode');
            });
        }
    }

    /**
     * Setup 2D/3D view mode event listeners
     */
    private setupViewModeListeners(): void {
        const btnViewMode2D = document.getElementById('view-mode-2d') as HTMLButtonElement;
        if (btnViewMode2D) {
            btnViewMode2D.addEventListener('click', () => this.viewportManager?.setViewMode('2d'));
        }

        const btnViewMode3D = document.getElementById('view-mode-3d') as HTMLButtonElement;
        if (btnViewMode3D) {
            btnViewMode3D.addEventListener('click', () => this.viewportManager?.setViewMode('3d'));
        }

        const btnGlobeAutoRotate = document.getElementById('globe-auto-rotate') as HTMLButtonElement;
        if (btnGlobeAutoRotate) {
            btnGlobeAutoRotate.addEventListener('click', () => this.viewportManager?.toggleGlobeAutoRotate());
        }

        const btnGlobeResetView = document.getElementById('globe-reset-view') as HTMLButtonElement;
        if (btnGlobeResetView) {
            btnGlobeResetView.addEventListener('click', () => this.viewportManager?.resetGlobeView());
        }
    }

    /**
     * Setup global action buttons (refresh, logout, theme)
     */
    private setupGlobalActionListeners(): void {
        const btnRefresh = document.getElementById('btn-refresh') as HTMLButtonElement;
        if (btnRefresh) {
            btnRefresh.addEventListener('click', () => this.loadData());
        }

        const btnLogout = document.getElementById('btn-logout') as HTMLButtonElement;
        if (btnLogout) {
            btnLogout.addEventListener('click', () => this.handleLogout());
        }

        const btnThemeToggle = document.getElementById('theme-toggle') as HTMLButtonElement;
        if (btnThemeToggle) {
            btnThemeToggle.addEventListener('click', () => this.themeManager?.toggleTheme());
        }
    }

    /**
     * Generic sub-tab setup helper to reduce code duplication.
     * @param containerSelector - CSS selector for the container (e.g., '#analytics-library')
     * @param panels - Map of tab name to panel config { id: string, onShow?: () => void }
     */
    private setupSubTabs(
        containerSelector: string,
        panels: Array<{ tabName: string; panelId: string; onShow?: () => void }>
    ): void {
        const subTabs = document.querySelectorAll(`${containerSelector} .sub-tab`);
        const panelElements = panels.map(p => ({
            tabName: p.tabName,
            element: document.getElementById(p.panelId),
            onShow: p.onShow
        }));

        subTabs.forEach(tab => {
            tab.addEventListener('click', () => {
                const targetTab = (tab as HTMLElement).dataset.subTab;

                // Update tab states
                subTabs.forEach(t => {
                    t.classList.remove('active');
                    t.setAttribute('aria-selected', 'false');
                });
                tab.classList.add('active');
                tab.setAttribute('aria-selected', 'true');

                // Hide all panels, show selected
                panelElements.forEach(({ tabName, element, onShow }) => {
                    if (!element) return;
                    const isActive = tabName === targetTab;
                    element.classList.toggle('active', isActive);
                    element.style.display = isActive ? 'block' : 'none';
                    if (isActive && onShow) {
                        onShow();
                    }
                });
            });
        });
    }

    /**
     * Setup sub-tab switching for Library page
     */
    private setupLibrarySubTabs(): void {
        this.setupSubTabs('#analytics-library', [
            { tabName: 'library-charts', panelId: 'library-charts-panel' },
            { tabName: 'library-details', panelId: 'library-details-panel', onShow: () => this.libraryDetailsManager?.init() }
        ]);
    }

    /**
     * Setup sub-tab switching for Content page (collections and playlists)
     */
    private setupContentSubTabs(): void {
        this.setupSubTabs('#analytics-content', [
            { tabName: 'content-analytics', panelId: 'content-analytics-panel' },
            { tabName: 'content-collections', panelId: 'content-collections-panel', onShow: () => this.collectionsManager?.init() },
            { tabName: 'content-playlists', panelId: 'content-playlists-panel', onShow: () => this.playlistsManager?.init() }
        ]);
    }

    /**
     * Setup sub-tab switching for Tautulli page
     */
    private setupTautulliSubTabs(): void {
        this.setupSubTabs('#analytics-tautulli', [
            { tabName: 'tautulli-charts', panelId: 'tautulli-charts-panel' },
            { tabName: 'metadata-deep-dive', panelId: 'metadata-deep-dive-panel', onShow: () => this.metadataDeepDiveManager?.init() },
            { tabName: 'rating-keys', panelId: 'rating-keys-panel', onShow: () => this.ratingKeysManager?.init() },
            { tabName: 'search', panelId: 'search-panel', onShow: () => this.searchManager?.init() },
            { tabName: 'stream-data', panelId: 'stream-data-panel', onShow: () => this.streamDataManager?.init() }
        ]);
    }

    /**
     * Setup IP History modal handlers (user IP history)
     * Opens modal when "View IP History" button is clicked on User Profile
     */
    private setupIPHistoryModal(): void {
        const btnViewIPHistory = document.getElementById('btn-view-ip-history');
        const modal = document.getElementById('user-ip-history-modal');
        const btnClose = document.getElementById('btn-close-ip-history');

        // Open modal on button click
        btnViewIPHistory?.addEventListener('click', async () => {
            if (!modal || !this.userProfileManager) return;

            // Get selected user from user profile selector
            const userSelector = document.getElementById('user-profile-selector') as HTMLSelectElement;
            const userId = parseInt(userSelector?.value || '0', 10);

            if (userId <= 0) {
                this.toastManager?.warning('Please select a user first');
                return;
            }

            // Show modal
            modal.style.display = 'flex';
            modal.setAttribute('aria-hidden', 'false');

            // Load IP details for user
            try {
                await this.userIPDetailsManager?.loadUserIPDetails(userId, userSelector?.options[userSelector.selectedIndex]?.text || 'User');
            } catch (error) {
                logger.error('[IPHistory] Failed to load IP details:', error);
                this.toastManager?.error('Failed to load IP history');
            }
        });

        // Close modal handlers
        const closeModal = () => {
            if (modal) {
                modal.style.display = 'none';
                modal.setAttribute('aria-hidden', 'true');
            }
        };

        btnClose?.addEventListener('click', closeModal);

        // Close on overlay click
        modal?.addEventListener('click', (e) => {
            if (e.target === modal) {
                closeModal();
            }
        });

        // Close on Escape key
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && modal?.style.display === 'flex') {
                closeModal();
            }
        });

        // Show button when user is selected
        const userSelector = document.getElementById('user-profile-selector') as HTMLSelectElement;
        userSelector?.addEventListener('change', () => {
            if (btnViewIPHistory) {
                const hasUser = parseInt(userSelector.value || '0', 10) > 0;
                btnViewIPHistory.style.display = hasUser ? 'inline-block' : 'none';
            }
        });
    }

    /**
     * Toggle fullscreen mode for the map/globe container
     * Uses the Fullscreen API for immersive viewing experience
     */
    private async toggleFullscreen(button: HTMLButtonElement): Promise<void> {
        const mapContainer = document.getElementById('map-container');
        if (!mapContainer) return;

        try {
            if (!document.fullscreenElement) {
                // Enter fullscreen
                await mapContainer.requestFullscreen();
                button.classList.add('active');
                button.setAttribute('aria-label', 'Exit fullscreen mode');
            } else {
                // Exit fullscreen
                await document.exitFullscreen();
                button.classList.remove('active');
                button.setAttribute('aria-label', 'Enter fullscreen mode');
            }
        } catch (error) {
            logger.error('[Fullscreen] Fullscreen toggle error:', error);
            // Show toast notification on error
            this.toastManager?.warning('Fullscreen mode is not available in this browser');
        }
    }

    /**
     * Handle logout button click
     * Delegates to AuthenticationManager which will call handleLogoutComplete callback
     */
    private handleLogout(): void {
        this.authManager?.handleLogout();
    }

    /**
     * Apply role-based visibility to UI elements.
     * RBAC Phase 4: Frontend Role Integration
     *
     * Hides admin-only elements for non-admin users:
     * - Data Governance tab
     * - Backup/Restore settings
     * - Server management controls
     * - Security alerts configuration
     * - Detection rules configuration
     *
     * Called after app initialization to ensure all DOM elements exist.
     */
    private applyRoleBasedVisibility(): void {
        const roleGuard = getRoleGuard();

        // Admin-only navigation tabs
        // These tabs should only be visible to admin users
        const adminOnlyTabs = [
            'nav-tab-data-governance',  // Data Governance tab
            'nav-tab-newsletter',       // Newsletter Generator tab
        ];

        // Hide admin-only tabs
        adminOnlyTabs.forEach(tabId => {
            const tab = document.querySelector(`[data-view="${tabId.replace('nav-tab-', '')}"]`);
            roleGuard.hideIfNotAdmin(tab as HTMLElement);
        });

        // Hide admin-only elements in the server tab
        const adminOnlyServerElements = [
            'backup-section',           // Backup/Restore section
            'server-sync-controls',     // Sync controls
        ];

        adminOnlyServerElements.forEach(elementId => {
            const element = document.getElementById(elementId);
            roleGuard.hideIfNotAdmin(element);
        });

        // Editor-required elements (editors and admins can see)
        const editorOnlyElements = [
            'detection-rules-config',   // Detection rules configuration
        ];

        editorOnlyElements.forEach(elementId => {
            const element = document.getElementById(elementId);
            roleGuard.showIfRole(element, 'editor');
        });

        logger.debug('[RBAC] Applied role-based visibility', {
            role: AuthContext.getInstance().getRole(),
            isAdmin: roleGuard.isAdmin(),
            isEditor: roleGuard.isEditor()
        });
    }

    /**
     * Cleanup after logout (called as callback from AuthenticationManager)
     * Destroys all managers and returns to login screen
     */
    private handleLogoutComplete(): void {
        this.destroyAllManagers();
        this.clearManagerReferences();

        // Reset lazy loading flag for E2E tests (ensures clean state on re-login)
        window.__lazyLoadComplete = false;

        this.showLogin();
    }

    /**
     * Destroy all active managers by calling their cleanup methods
     */
    private destroyAllManagers(): void {
        // Managers with destroy() method
        const destroyableManagers = [
            this.chartManager,
            this.timelineManager,
            this.globeManager,
            this.activityDashboard,
            this.serverDashboard,
            this.recentlyAddedDashboard,
            this.dataFreshnessManager,
            this.chartDrillDownManager,
            this.chartExportManager,
            this.chartMaximizeManager,
            this.mapKeyboardManager,
            this.globeKeyboardManager,
            this.dateRangeBrushManager,
            this.tableViewManager,
            this.recentSearchesManager,
            this.securityAlertsManager,
            this.detectionRulesManager,
            this.crossPlatformManager,
            this.dataGovernanceManager,
            this.newsletterManager,
            this.libraryDetailsManager,
            this.collectionsManager,
            this.playlistsManager,
            this.userIPDetailsManager,
            this.metadataDeepDiveManager,
            this.ratingKeysManager,
            this.searchManager,
            this.streamDataManager,
            this.navigationManager,
            this.setupWizardManager,
            this.progressiveOnboardingManager,
        ];

        for (const manager of destroyableManagers) {
            try {
                if (hasDestroy(manager)) {
                    manager.destroy();
                }
            } catch (error) {
                logger.warn('[App] Error destroying manager:', error);
            }
        }

        // Special cleanup methods
        this.wsManager?.disconnect();
        this.toastManager?.dismissAll();

        // Stop all intervals/timers to prevent memory leaks
        this.stopAutoRefresh();
    }

    /**
     * Clear all manager references for garbage collection.
     * Groups managers by category for systematic cleanup.
     */
    private clearManagerReferences(): void {
        // Core managers
        this.mapManager = null;
        this.globeManager = null;
        this.statsManager = null;
        this.chartManager = null;
        this.timelineManager = null;
        this.wsManager = null;
        this.toastManager = null;
        this.filterManager = null;

        // UI managers
        this.viewportManager = null;
        this.navigationManager = null;
        this.sidebarManager = null;
        this.timelineController = null;

        // Dashboard managers
        this.activityDashboard = null;
        this.serverDashboard = null;
        this.recentlyAddedDashboard = null;
        this.wrappedDashboard = null;

        // Feature managers
        this.plexMonitoringManager = null;
        this.insightsManager = null;
        this.libraryAnalyticsManager = null;
        this.userProfileManager = null;
        this.setupWizardManager = null;
        this.onboardingManager = null;
        this.progressiveOnboardingManager = null;
        this.keyboardShortcutsManager = null;
        this.confirmationDialogManager = null;
        this.dataFreshnessManager = null;
        this.errorBoundaryManager = null;
        this.filterPresetManager = null;
        this.backupRestoreManager = null;
        this.serverManagementManager = null;
        this.settingsManager = null;
        this.helpDocumentationManager = null;

        // Chart-related managers
        this.chartTooltipManager = null;
        this.chartDrillDownManager = null;
        this.chartExportManager = null;
        this.chartMaximizeManager = null;
        this.chartTimelineAnimationManager = null;

        // Analytics managers
        this.quickInsightsManager = null;
        this.comparativeInsightsManager = null;
        this.customDateComparisonManager = null;
        this.geographicDrillDownManager = null;
        this.anomalyDetectionManager = null;
        this.streamingGeoJSONManager = null;

        // Map/Globe managers
        this.mapKeyboardManager = null;
        this.globeKeyboardManager = null;
        this.dateRangeBrushManager = null;

        // UI enhancement managers
        this.notificationCenterManager = null;
        this.dataCachingIndicatorManager = null;
        this.loadingProgressManager = null;
        this.emptyStateActionsManager = null;
        this.breadcrumbNavigationManager = null;
        this.tableViewManager = null;
        this.recentSearchesManager = null;
        this.filterHistoryManager = null;
        this.dataExportManager = null;

        // Security & Detection managers
        this.securityAlertsManager = null;
        this.detectionRulesManager = null;
        this.dedupeAuditManager = null;

        // Cross-Platform & Governance managers
        this.crossPlatformManager = null;
        this.dataGovernanceManager = null;
        this.newsletterManager = null;

        // Library & Content managers
        this.libraryDetailsManager = null;
        this.collectionsManager = null;
        this.playlistsManager = null;
        this.userIPDetailsManager = null;
        this.metadataDeepDiveManager = null;
        this.ratingKeysManager = null;
        this.searchManager = null;
        this.streamDataManager = null;
    }


    private async loadData(): Promise<void> {
        this.showLoading(true);

        // Define loading operations for progress tracking
        const loadingOperations = [
            { id: 'stats', label: 'Loading statistics', weight: 1 },
            { id: 'locations', label: 'Loading locations', weight: 2 },
            { id: 'charts', label: 'Loading charts', weight: 3 },
            { id: 'timeline', label: 'Loading timeline', weight: 1 },
            { id: 'insights', label: 'Loading insights', weight: 1 },
            { id: 'quickInsights', label: 'Loading quick insights', weight: 1 },
            { id: 'comparative', label: 'Loading comparisons', weight: 1 },
            { id: 'anomalies', label: 'Analyzing patterns', weight: 1 }
        ];

        // Start tracked loading progress
        this.loadingProgressManager?.startLoading(loadingOperations);

        try {
            const filter = this.filterManager!.buildFilter();

            // Track each operation's completion for accurate progress
            await Promise.all([
                this.statsManager!.loadStats().then(() => {
                    this.loadingProgressManager?.completeOperation('stats');
                }),
                this.loadLocationData(filter).then(() => {
                    this.loadingProgressManager?.completeOperation('locations');
                }),
                this.chartManager!.loadAllCharts(filter).then(() => {
                    this.loadingProgressManager?.completeOperation('charts');
                }),
                this.timelineManager!.loadData(filter).then(() => {
                    this.loadingProgressManager?.completeOperation('timeline');
                }),
                (async () => {
                    if (this.insightsManager) {
                        await this.insightsManager.loadInsights(filter);
                    }
                    this.loadingProgressManager?.completeOperation('insights');
                })(),
                (async () => {
                    if (this.quickInsightsManager) {
                        await this.quickInsightsManager.loadInsights(filter);
                    }
                    this.loadingProgressManager?.completeOperation('quickInsights');
                })(),
                (async () => {
                    if (this.comparativeInsightsManager) {
                        await this.comparativeInsightsManager.loadComparison(filter);
                    }
                    this.loadingProgressManager?.completeOperation('comparative');
                })(),
                (async () => {
                    if (this.anomalyDetectionManager) {
                        await this.anomalyDetectionManager.analyzeAnomalies(filter);
                    }
                    this.loadingProgressManager?.completeOperation('anomalies');
                })()
            ]);

            // Restore arc overlay state after map is ready
            this.restoreArcToggleState();

            // Mark data as updated for freshness tracking
            this.dataFreshnessManager?.markDataUpdated();

            // Report success to error boundary
            this.errorBoundaryManager?.reportSuccess();

            // Announce loading complete to screen readers
            announceLoadingComplete('map data');

            // Initialize chart info tooltips after charts are rendered
            this.chartTooltipManager?.init();
        } catch (error) {
            logger.error('Failed to load data:', error);

            // Report error to error boundary for global error handling
            const errorContext = ErrorBoundaryManager.createContext(error, 'data-load');
            this.errorBoundaryManager?.reportError(errorContext);

            // Announce loading error to screen readers
            announceLoadingError('map data');

            // Still show toast for immediate feedback
            this.showError('Failed to load data. Please check your connection.');
        } finally {
            this.showLoading(false);
            // Finish loading progress (ensures smooth animation to 100%)
            this.loadingProgressManager?.finishLoading();
        }
    }

    /**
     * Restore arc toggle button state from localStorage
     */
    private restoreArcToggleState(): void {
        const saved = SafeStorage.getItem('arc-overlay-enabled');
        if (saved === 'true' && this.mapManager) {
            const arcToggle = document.getElementById('arc-toggle') as HTMLButtonElement;
            if (arcToggle) {
                arcToggle.classList.add('active', 'enabled');
                arcToggle.setAttribute('aria-pressed', 'true');
            }
            // Restore the map arc overlay state (will also load arc data)
            this.mapManager.restoreArcOverlayState();
        }
    }

    private async loadLocationData(filter: LocationFilter): Promise<void> {
        const locations = await this.api.getLocations(filter);
        this.mapManager!.updateLocations(locations);
        if (this.globeManager) {
            this.globeManager.updateLocations(locations);
        }
        // Also load geographic drill-down data
        if (this.geographicDrillDownManager) {
            this.geographicDrillDownManager.loadData(filter);
        }
    }

    /**
     * Load H3 hexagon aggregated data for hexagon visualization mode
     */
    private async loadHexagonData(resolution: number = 7): Promise<void> {
        if (!this.mapManager || !this.filterManager) return;

        try {
            const filter = this.filterManager.buildFilter();
            const hexagons = await this.api.getH3HexagonData(filter, resolution);
            this.mapManager.updateHexagons(hexagons);
        } catch (error) {
            logger.error('Failed to load hexagon data:', error);
            if (this.toastManager) {
                this.toastManager.error('Failed to load hexagon data. Please try again.');
            }
        }
    }

    /**
     * Load arc data for server-to-user connection visualization
     */
    private async loadArcData(): Promise<void> {
        if (!this.mapManager || !this.filterManager) return;

        try {
            const filter = this.filterManager.buildFilter();
            const arcs = await this.api.getArcData(filter);
            this.mapManager.updateArcs(arcs);
        } catch (error) {
            logger.error('Failed to load arc data:', error);
            if (this.toastManager) {
                this.toastManager.error('Failed to load arc data. Server location may not be configured.');
            }
        }
    }

    private showLoading(show: boolean): void {
        const loading = document.getElementById('loading');
        if (loading) {
            loading.className = show ? 'show' : '';
        }

        // Data refresh indicator
        const refreshIndicator = document.getElementById('data-refresh-indicator');
        const refreshButton = document.getElementById('btn-refresh');
        const refreshText = document.getElementById('refresh-text');
        const refreshStatusText = document.getElementById('refresh-status-text');

        if (refreshIndicator) {
            refreshIndicator.style.display = show ? 'flex' : 'none';
            if (refreshStatusText) {
                refreshStatusText.textContent = show ? 'Loading data...' : 'Data loaded';
            }
        }

        if (refreshButton) {
            refreshButton.classList.toggle('loading', show);
            if (refreshText) {
                refreshText.textContent = show ? 'Loading...' : 'Refresh Data';
            }
        }

        // Screen reader announcements for loading states
        if (show) {
            announceLoadingStart('map data');
        }
    }

    private showError(message: string): void {
        logger.error(message);
    }

    /**
     * Start periodic stats-only refresh (background sync for real-time feel)
     * Note: Full data auto-refresh is controlled by DataFreshnessManager
     */
    private startAutoRefresh(): void {
        // Clear any existing interval before starting a new one
        if (this.statsAutoRefreshIntervalId !== null) {
            clearInterval(this.statsAutoRefreshIntervalId);
        }

        // Stats refresh runs independently of the full data refresh toggle
        // This provides a lightweight way to keep stats updated
        // Store interval ID for cleanup on logout
        this.statsAutoRefreshIntervalId = setInterval(() => {
            if (this.statsManager) {
                this.statsManager.loadStats();
            }
        }, 60000);
    }

    /**
     * Stop periodic stats refresh (called on logout)
     */
    private stopAutoRefresh(): void {
        if (this.statsAutoRefreshIntervalId !== null) {
            clearInterval(this.statsAutoRefreshIntervalId);
            this.statsAutoRefreshIntervalId = null;
        }
        if (this.serviceWorkerUpdateIntervalId !== null) {
            clearInterval(this.serviceWorkerUpdateIntervalId);
            this.serviceWorkerUpdateIntervalId = null;
        }
    }

    // ============================================
    // Test Helpers (for E2E testing only)
    // ============================================

    /**
     * Simulate a WebSocket message for testing
     * @param type - The message type (e.g., 'plex_realtime_playback', 'plex_transcode_sessions', 'buffer_health_update')
     * @param data - The message data
     */
    public __testSimulateWebSocketMessage(type: string, data: WebSocketMessageData): void {
        if (!this.plexMonitoringManager) return;

        switch (type) {
            case 'plex_realtime_playback':
                this.plexMonitoringManager.handlePlexRealtimePlayback(data as PlexRealtimePlaybackData);
                break;
            case 'plex_transcode_sessions':
                this.plexMonitoringManager.handlePlexTranscodeSessions(data as PlexTranscodeSessionsData);
                break;
            case 'buffer_health_update':
                this.plexMonitoringManager.handleBufferHealthUpdate(data as BufferHealthUpdateData);
                break;
            default:
                logger.warn(`Unknown test message type: ${type}`);
        }
    }

    /**
     * Check if WebSocket is connected (for testing)
     */
    public __testIsWebSocketConnected(): boolean {
        return this.wsManager?.isConnected() ?? false;
    }

    /**
     * Get the PlexMonitoringManager for testing
     */
    public __testGetPlexMonitoringManager(): PlexMonitoringManager | null {
        return this.plexMonitoringManager;
    }

    /**
     * Get the MapManager for testing (E2E tests check map layers)
     */
    public __testGetMapManager(): MapManager | null {
        return this.mapManager;
    }

    /**
     * Get the ViewportManager for testing
     */
    public __testGetViewportManager(): ViewportManager | null {
        return this.viewportManager;
    }

    /**
     * Get the InsightsManager for testing
     */
    public __testGetInsightsManager(): InsightsManager | null {
        return this.insightsManager;
    }

    /**
     * Get the FilterManager for testing
     */
    public __testGetFilterManager(): FilterManager | null {
        return this.filterManager;
    }

    /**
     * Get the SidebarManager for testing
     */
    public __testGetSidebarManager(): SidebarManager | null {
        return this.sidebarManager;
    }

    /**
     * Get the ChartManager for testing
     */
    public __testGetChartManager(): ChartManager | null {
        return this.chartManager;
    }
}

document.addEventListener('DOMContentLoaded', () => {
    const app = new App();
    // Expose app instance for E2E testing
    window.__app = app;

    // Expose mapManager globally for E2E tests that check map layers
    // Tests use window.mapManager.getMap() to verify layer visibility
    Object.defineProperty(window, 'mapManager', {
        get: () => app.__testGetMapManager(),
        configurable: true
    });

    // Expose viewportManager globally for E2E tests that set map modes
    // Tests use window.viewportManager.setMapMode() to switch visualization modes
    Object.defineProperty(window, 'viewportManager', {
        get: () => app.__testGetViewportManager(),
        configurable: true
    });
});

// Service worker registration is handled by App.registerServiceWorker() method
// to ensure proper initialization order and error handling
