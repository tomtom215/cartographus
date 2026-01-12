# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> **Note**: For detailed change descriptions of versions prior to 1.40, see [docs/CHANGELOG-ARCHIVE.md](./docs/CHANGELOG-ARCHIVE.md).

---

## [Unreleased]

### Added
- **Data Sync UI**: Web interface for initiating and monitoring data synchronization
  - `DataSyncSettingsSection.ts` - Settings panel integration for sync operations (~750 lines)
  - `SyncManager.ts` - Production-grade state management with DI for testability (~800 lines)
  - `SyncProgressComponent.ts` - Reusable progress display with WCAG 2.1 AA accessibility (~450 lines)
  - Backend API endpoints: `GET /api/v1/sync/status`, `POST /api/v1/sync/plex/historical`
  - `internal/api/handlers_sync.go` - Sync API handlers with correlation ID tracing
  - WebSocket `sync_progress` message type for real-time progress updates
  - Tautulli Import UI: database path input, resume/dry-run options, progress tracking
  - Plex Historical Sync UI: days-back selector, library filtering, progress tracking
  - Polling fallback when WebSocket disconnected (3-second interval)
  - Durable state persistence via SafeSessionStorage (survives page refresh)
  - Operation mutex (Tautulli import and Plex historical sync cannot run concurrently)
  - Progress display: progress bar, records/sec, ETA, imported/skipped/error counts
  - Expandable error log with recoverable/non-recoverable indicators
  - CSS: `data-sync.css` with responsive design and reduced motion support
  - TypeScript types: `SyncProgress`, `SyncStatusResponse`, `TautulliImportRequest`, etc.
  - E2E tests: `tests/e2e/68-data-sync-ui.spec.ts` (20+ test cases)
  - Unit tests: `handlers_sync_test.go` (19 tests), `SyncManager.test.ts` (48 tests)
  - Design document: `docs/design/DATA_SYNC_UI_DESIGN.md`
- **Backup Scheduling System**: Automatic backup scheduling with configurable intervals
  - Backend API endpoints: GET/PUT `/api/v1/backup/schedule`, POST `/api/v1/backup/schedule/trigger`
  - `internal/backup/manager_schedule.go` - Schedule configuration and validation
  - Frontend `ScheduleConfig` and `SetScheduleConfigRequest` types
  - BackupsRenderer enhancement with schedule configuration UI
  - Schedule toggle, interval selection (hourly to weekly), preferred hour, backup type
  - Pre-sync backup option for automatic backups before sync operations
  - Manual backup trigger button ("Run Backup Now")
  - Next backup time estimation display
  - CSS: Toggle switch, schedule configuration grid, action buttons
  - E2E tests: `tests/e2e/24-backup-scheduling.spec.ts` (30+ test cases)
  - Mock server endpoints for schedule configuration testing
  - Integrated within Data Governance > Backups page
- **Health Dashboard Enhancement (ADR-0005, ADR-0006, ADR-0026)**: Comprehensive system health monitoring
  - Enhanced `HealthRenderer.ts` (~400 lines) - Real-time health data fetching
  - Overall system status with healthy/degraded indicators
  - Component health grid: Database (DuckDB), Tautulli, NATS JetStream, WAL, WebSocket, Detection Engine
  - Connected media servers display with platform icons (Plex, Jellyfin, Emby, Tautulli)
  - Sync lag indicators with color-coded status (good < 5m, warning 5-30m, critical > 30m)
  - Server error display for troubleshooting
  - Version and uptime information
  - Last sync times section (data sync, backup, detection)
  - CSS: Media server cards, sync lag styling, status badges
  - E2E tests: `tests/e2e/23-health-dashboard.spec.ts` (40+ test cases)
  - Mock server endpoints for WAL stats, WAL health, and server status
  - Integrated within Data Governance > Health page
- **Audit Log Viewer Enhancement (ADR-0015)**: Enhanced security audit log viewing capabilities
  - `AuditLogViewer.ts` - Standalone audit log viewing component (~700 lines)
  - `audit-log.css` - Dedicated styling for audit log viewer (~500 lines)
  - Stats summary cards (total events, success/failure counts, warnings)
  - Comprehensive filtering (type, severity, outcome, date range)
  - Full-text search with debounced input
  - Pagination controls with page size options
  - Event detail modal with complete event information
  - Export to JSON and CEF formats (SIEM integration)
  - Responsive design for mobile and tablet
  - E2E tests: `tests/e2e/22-audit-log-viewer.spec.ts` (20+ test cases)
  - Mock server endpoints for audit log testing
  - Integrated within Data Governance > Audit Log page
- **Active Session Management UI (ADR-0015)**: Security-focused session management interface
  - `SessionManagementManager.ts` - View and manage active login sessions
  - Session list showing all active sessions across devices
  - Device name parsing from User-Agent (Mac, Windows, iPhone, etc.)
  - Provider identification (Plex OAuth, SSO/OIDC, Username/Password)
  - Current session badge to prevent accidental self-logout
  - Individual session revocation with confirmation for current session
  - "Sign Out Everywhere" functionality with session count
  - Refresh button for real-time session updates
  - Integrated into User Profile Settings under new Security section
  - E2E tests: `tests/e2e/21-session-management.spec.ts` (10 test cases)
  - Mock server endpoints for session management testing
- **Session Management API Methods**: Frontend API integration
  - `getSessions()` - List all active sessions for current user
  - `revokeSession(id)` - Revoke a specific session
  - `logoutAll()` - Sign out from all sessions
  - `getUserInfo()` - Get current user info
- **Multi-Server Management UI (ADR-0026)**: Complete server management interface
  - `MultiServerManager.ts` (~800 lines) - Full CRUD manager with status indicators
  - Server list with platform icons (Plex, Jellyfin, Emby, Tautulli)
  - Source badges distinguish env var servers (immutable) from UI-added servers
  - Real-time status indicators (connected, disconnected, error, syncing)
  - Add/Edit Server modal with connection testing
  - Delete confirmation dialog
  - Enable/disable toggles for UI-added servers
  - RBAC integration via RoleGuard (admin-only)
  - `data-testid` attributes for E2E testing
- **Server Management API Handlers**: Backend CRUD operations
  - `POST /api/v1/admin/servers` - Create new server
  - `GET /api/v1/admin/servers` - List all servers (env + DB)
  - `GET /api/v1/admin/servers/db` - List DB-stored servers only
  - `GET /api/v1/admin/servers/{id}` - Get server by ID
  - `PUT /api/v1/admin/servers/{id}` - Update server
  - `DELETE /api/v1/admin/servers/{id}` - Delete server
  - `POST /api/v1/admin/servers/test` - Test connection
- **Server Management Tests**: Comprehensive test coverage
  - E2E tests: `tests/e2e/20-server-management.spec.ts` (40+ test cases)
  - Integration tests: `handlers_server_management_test.go` (20+ test cases)
  - Mock handlers in `tests/e2e/fixtures/mock-server.ts`
- **OpenID Connect with Zitadel OIDC v3.45.1**: Production-grade OIDC implementation using OpenID Foundation certified library
  - PKCE (RFC 7636) support with automatic code verifier generation
  - Nonce validation with replay attack protection
  - Back-channel logout (OIDC Back-Channel Logout 1.0 spec)
  - Token refresh with automatic session extension
  - Compatible with Authelia, Authentik, Keycloak, Okta, and other OIDC providers
- **Durable OIDC State Storage**: BadgerDB-backed state store for ACID-compliant OIDC state persistence
  - Survives server restarts without losing pending authentication flows
  - Automatic cleanup of expired states with configurable TTL
  - Thread-safe concurrent access
- **OIDC Prometheus Metrics**: 15+ metrics for authentication observability
  - `oidc_login_attempts_total`, `oidc_login_duration_seconds` - Login monitoring
  - `oidc_logout_total`, `oidc_token_refresh_total` - Session lifecycle
  - `oidc_state_store_operations_total`, `oidc_state_store_size` - State store health
  - `oidc_back_channel_logout_total`, `oidc_validation_errors_total` - Security events
- **OIDC Audit Logging**: Security event logging for all authentication events
  - Login success/failure with IP, user agent, duration
  - Logout events (RP-initiated and back-channel)
  - Token refresh tracking with expiry updates
  - Session creation and termination events
- **Enhanced Analytics Visualizations**: Production-grade analytics dashboards with comprehensive frontend and backend support
  - **Cohort Retention Analysis**: Weekly cohort tracking with retention matrix heatmap, retention curve visualization, week 1/4/12 summary gauges
  - **Quality of Experience (QoE) Dashboard**: Netflix-style metrics including EBVS rate, quality degradation, completion rates, platform/transcode breakdown
  - **Data Quality Monitoring**: Field-level quality scores, daily trend analysis, issue detection with severity and recommendations
  - **User Network Graph**: Force-directed social network visualization showing viewing relationships, cluster detection, connection types
  - **Cross-Filter Event Bus**: Pub/sub event system for linked chart interactions (Tableau-style cross-highlighting)
  - **Data Provenance Tracking**: Query hash, execution time, cache status for data auditability and reproducibility
- New API endpoints: `GET /api/v1/analytics/cohort-retention`, `GET /api/v1/analytics/qoe`, `GET /api/v1/analytics/data-quality`, `GET /api/v1/analytics/user-network`
- TypeScript types for all new analytics (CohortRetentionAnalytics, QoEDashboard, DataQualityReport, UserNetworkGraph)
- 18 new chart renderers across 4 categories for enhanced analytics visualization
- Unit tests for all new analytics backend functions
- **Cross-Platform Content Reconciliation (Phase 3)**: Link content across Plex/Jellyfin/Emby using external IDs
  - `content_mappings` table for cross-platform content linking via IMDb, TMDB, TVDB IDs
  - API endpoints: `POST /api/v1/content/link`, `GET /api/v1/content/lookup`
  - Platform-specific linking: `POST /api/v1/content/{id}/link/plex|jellyfin|emby`
- **Cross-Platform User Identity Linking (Phase 3)**: Unified user analytics across platforms
  - `user_links` table for manual and email-based user linking
  - API endpoints: `POST /api/v1/users/link`, `GET /api/v1/users/{id}/linked`, `DELETE /api/v1/users/link`
  - Email-based link suggestions: `GET /api/v1/users/suggest-links`
- **Cross-Platform Analytics (Phase 3)**: Aggregated statistics across linked identities
  - `GET /api/v1/analytics/cross-platform/user/{id}` - User stats across all linked accounts
  - `GET /api/v1/analytics/cross-platform/content/{id}` - Content stats across platforms
  - `GET /api/v1/analytics/cross-platform/summary` - Overall cross-platform summary

### Changed
- **OIDC Implementation Migration**: Replaced custom OIDC implementation with OpenID Foundation certified Zitadel library
  - See [ADR-0015](./docs/adr/0015-zero-trust-authentication-authorization.md) for migration rationale
  - No breaking changes to environment variables or API endpoints
  - Improved security with certified token validation
- **Documentation Overhaul**: Major updates to all core documentation files
  - CLAUDE.md: Updated ADR count (20), multi-server architecture diagram, detection features, improved environment setup instructions with DuckDB limitations
  - README.md: Complete rewrite - Docker quick-start first, multi-server support (Plex, Jellyfin, Emby), security detection features, K8s moved to advanced deployment
  - ARCHITECTURE.md: Streamlined to focus on current production architecture, removed changelog-style content, all history/decisions now in ADRs
  - ADR README.md: Updated last modified date
  - API-REFERENCE.md: Added Cross-Platform Endpoints section with content mapping and user linking APIs
  - PRODUCTION_READINESS_AUDIT.md: Updated to v4.0 with Phase 3 completion status

### Fixed
- **DuckDB Compatibility**: Removed partial index WHERE clauses (unsupported by DuckDB)
- **Handler Compilation**: Added `writeJSONResponse` helper for cross-platform handlers

### Added
- **Post-Restore Verification**: Database restore operations now verify integrity
  - Verifies database file exists and is not empty
  - Opens read-only DuckDB connection for verification
  - Verifies core tables (playbacks, geolocations) exist
  - Compares record counts against backup metadata (5% variance allowed)
- **Plex OAuth Implementation**: Complete OAuth flow with user info fetch, token storage, JWT session generation
- **Casbin RBAC Enforcer**: Zero Trust authorization initialized in Chi router
- **Suture v4 Process Supervision**: Erlang/OTP-style supervisor tree for automatic service restart
  - 3-layer hierarchy: Root, Data, Messaging, API supervisors
  - WebSocket Hub, Sync Manager, NATS Components, HTTP Server as supervised services
  - Exponential backoff on failures (5 failures before considered dead)
  - Graceful shutdown with timeout handling and unstopped service reporting
  - 41 new unit tests with race detection coverage
- NATS-First Event Sourcing Architecture for cross-source deduplication (Plex + future Jellyfin support)
- CorrelationKey mechanism for event deduplication across sources
- **Request Validation Package**: go-playground/validator v10.28.0 integration
  - Singleton thread-safe validator with custom error translation
  - Request validation structs with struct tags for all API endpoints
  - `validateRequest()` helper function for consistent validation error responses
  - Custom validators for base64url cursors, RFC3339 dates, coordinates
  - 90%+ test coverage with comprehensive table-driven tests

### Changed
- **Cookie Consent Banner**: Disabled for self-hosted deployment (no tracking analytics)
  - To re-enable for SaaS: uncomment `CookieConsentManager` import in `web/src/index.ts`

### Fixed
- **Memory Leak Prevention**: Added proper event listener cleanup in 8 TypeScript managers
  - ExportManager, SettingsManager, BackupRestoreManager, FilterPresetManager
  - SidebarManager, NotificationCenterManager, ErrorBoundaryManager, ChartTooltipManager
  - Pattern: stored handler refs, event delegation, `destroy()` methods
- E2E Test Suite fixes (40+ tests): ECharts SVG/Canvas rendering, keyboard navigation, container visibility
- SBOM Generation for multi-arch Docker images
- H3 DuckDB extension installation with correct `FROM community` syntax
- DuckDB spatial column binding in GROUP BY clauses

### Refactored
- Test suite complexity reduction: 3 files refactored with 41% line reduction
- API handler complexity reduction (~50%) with executor pattern

### Security
- Debian 13 (Trixie) migration fixing CVE-2023-2953 and CVE-2025-6020

---

## [1.44] - 2025-11-25

### Refactored
- **Frontend TypeScript Refactoring**: Extracted PlexMonitoringManager (395 lines) and FilterManager (218 lines) from index.ts, reducing main file by 27.6% (1,999 to 1,447 lines)

---

## [1.43] - 2025-11-25

### Added
- **Plex OAuth 2.0 PKCE**: Complete OAuth flow with PKCE, CSRF protection, token refresh
- **Plex Webhook Endpoint**: `/api/v1/plex/webhook` for real-time push notifications
- Backend tests (14 test cases), E2E tests (5 test suites)

---

## [1.42] - 2025-11-24

### Added
- **Bitrate Analytics**: 3-level bitrate tracking (source, transcode, network) via `/api/v1/analytics/bitrate`
- Distribution histogram, 30-day trends, resolution comparison, daily time series

---

## [1.41] - 2025-11-24

### Added
- **Buffer Health Monitoring**: Predictive buffering detection with 10-15 second warning capability
- Buffer fill percentage tracking, drain rate calculation, three-tier health status
- Per-session analysis with toast alerts for critical/risky sessions
- Backend tests (12 cases), E2E tests (15 cases)

---

## [1.40] - 2025-11-24

### Added
- **Plex Transcode Monitoring**: Real-time transcode session tracking
- Quality transitions (4K to 1080p), hardware acceleration detection (Quick Sync, NVENC, VAAPI)
- Live UI panel with progress bars, throttling warnings

---

## [1.39] - 2025-11-24

### Added
- **Plex WebSocket Real-time**: Sub-second playback notifications via Plex WebSocket API
- Session start/stop/pause detection, 3-attempt reconnection with backoff

---

## [1.38] - 2025-11-24

### Added
- **Cursor-Based Pagination**: O(1) pagination for `/api/v1/playbacks` endpoint
- Base64-encoded cursors, backwards compatible with offset pagination

---

## [1.37] - 2025-11-24

### Added
- **Abandonment Analytics**: Content drop-off analysis via `/api/v1/analytics/abandonment`
- Top abandoned content, completion by media type, genre patterns, first-episode analysis

---

## [1.36] - 2025-11-24

### Added
- **Audio Analytics**: Audio channel distribution, bitrate analysis, format breakdown
- **Subtitle Analytics**: Usage patterns, language distribution, codec analysis

---

## [1.35] - 2025-11-23

### Added
- **Concurrent Streams Analytics**: Peak usage tracking, capacity recommendations
- **Connection Security Analytics**: Secure/insecure connection monitoring, relay detection

---

## [1.34] - 2025-11-23

### Refactored
- P1 Part 3 Complexity Reduction: 7 functions refactored (76% average reduction)
- All main functions now complexity 6 or less

---

## [1.33] - 2025-11-23

### Refactored
- P1 Complexity Sprint: 8 analytics functions refactored (75% average reduction)
- Functions >15 complexity reduced from 32 to 15 (-53%)

---

## [1.32] - 2025-11-23

### Refactored
- Split analytics_advanced.go into 4 domain files (binge, bandwidth, engagement, comparative)
- Created query builder package for reusable SQL templates
- Added package-level documentation (doc.go, 165 lines)

---

## [1.31] - 2025-11-23

### Refactored
- Split tautulli.go (2,566 lines) into 6 domain-specific files
- Refactored 4 complex analytics functions (78% reduction)

---

## [1.30] - 2025-11-23

### Added
- Codebase Metrics Analysis document (800+ lines)
- Grade B+ (87/100), identified 36 functions with complexity >15

---

## [1.29] - 2025-11-23

### Changed
- Go 1.25.4 upgrade with toolchain alignment
- golangci-lint v2.x migration with working exclusion rules

---

## [1.28] - 2025-11-23

### Added
- 12 security-focused fuzz tests (JWT, API parameters, SQL injection prevention)
- CI/CD integration with 30-second per-test timeout

---

## [1.27] - 2025-11-23

### Added
- Code Metrics (scc) in CI/CD: LOC, complexity, comment ratio tracking

---

## [1.26] - 2025-11-22

### Added
- Coverage Diff Analysis: Per-package coverage with regression detection

---

## [1.25] - 2025-11-22

### Added
- Block/Mutex Profiling for concurrency analysis

---

## [1.24] - 2025-11-22

### Added
- golangci-lint with 40+ linters
- Benchmark regression detection with benchstat

---

## [1.23] - 2025-11-22

### Added
- Performance Profiling (pprof) in CI/CD

---

## [1.22] - 2025-11-22

### Added
- **100 Routes Milestone**: 8 new Tautulli endpoints for export management and server info

---

## Version Summary (Recent)

| Version | Date | Highlights |
|---------|------|------------|
| Unreleased | - | Zitadel OIDC (certified), detection engine (ADR-0020), documentation overhaul |
| 1.44 | 2025-11-25 | Frontend refactoring (-27.6% main file) |
| 1.43 | 2025-11-25 | Plex OAuth PKCE, webhooks |
| 1.42 | 2025-11-24 | Bitrate analytics |
| 1.41 | 2025-11-24 | Buffer health monitoring |
| 1.40 | 2025-11-24 | Transcode monitoring |
| 1.39 | 2025-11-24 | Plex WebSocket real-time |
| 1.38 | 2025-11-24 | Cursor pagination |
| 1.37 | 2025-11-24 | Abandonment analytics |
| 1.36 | 2025-11-24 | Audio/subtitle analytics |
| 1.35 | 2025-11-23 | Concurrent streams, connection security |

For versions 1.0-1.21 and earlier releases (0.x), see [docs/CHANGELOG-ARCHIVE.md](./docs/CHANGELOG-ARCHIVE.md).

---

## Breaking Changes

### 0.1.0
- **JWT_SECRET required**: When `AUTH_MODE=jwt`, set `JWT_SECRET` with at least 32 characters.
  ```bash
  export JWT_SECRET=$(openssl rand -base64 48)
  ```

---

## Support

- GitHub Issues: https://github.com/tomtom215/cartographus/issues
- GitHub Discussions: https://github.com/tomtom215/cartographus/discussions
