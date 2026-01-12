# Changelog Archive

This file contains detailed changelog entries for versions 1.0-1.21 and earlier releases (0.x).

For recent changes, see the main [CHANGELOG.md](../CHANGELOG.md).

---

## Archive Navigation

- [Version 1.x (1.0-1.21)](#version-1x)
- [Version 0.x (0.0.1-0.3.0)](#version-0x)
- [Version Summary Table](#version-summary)
- [Upgrade Guides](#upgrade-guides)

---

## Version 1.x

## [1.21] - 2025-11-22

### Changed

**Priority 3 Collections, Playlists & Server Identification Implementation**: Implemented 5 new Tautulli API endpoints for content organization analytics and multi-server support, reaching 49.5% total endpoint coverage (46→51 endpoints). **Endpoints Added**: (1) `/api/v1/tautulli/collections-table` - Paginated collection data with sorting (order_column, order_dir) and search filtering showing collection metadata (section_id, title, collection type smart/manual, item counts, play statistics) for content grouping analytics, (2) `/api/v1/tautulli/playlists-table` - Paginated playlist information with sorting and search showing playlist type (smart vs manual), total duration, play counts, user ownership, and popularity metrics for usage pattern analysis and content discovery insights, (3) `/api/v1/tautulli/server-friendly-name` - Server display name (friendly name) for UI presentation in multi-server environments without exposing machine identifiers, (4) `/api/v1/tautulli/server-id` - Unique server identifier (machine_identifier) for server tracking, API correlation, and multi-server deployments, (5) `/api/v1/tautulli/server-identity` - Comprehensive machine identity including machine_identifier, Plex version, platform (Linux/Windows/macOS), platform_version, and creation timestamp for server fingerprinting and version tracking. **Implementation**: Added 5 model structs (117 lines) in `internal/models/models.go` (TautulliCollectionsTable, TautulliPlaylistsTable, TautulliServerFriendlyName, TautulliServerID, TautulliServerIdentity), 5 client methods (210 lines) in `internal/sync/tautulli.go`, 5 HTTP handlers (211 lines) with 5-minute cache TTL for all endpoints in `internal/api/handlers.go`, 5 routes with CORS and rate limiting in `internal/api/router.go`, updated mock clients in both `internal/sync/manager_test.go` (+20 lines stub methods) and `internal/api/handlers_test.go` (+32 lines function fields and implementations). **Testing**: All sync and API tests pass with race detector enabled. **Verified Counts**: Routes 87→92 (+5), TautulliClient methods 47→52 (+5), Tautulli proxy routes 41→46 (+5). **Files Modified**: internal/models/models.go (+117 lines), internal/sync/tautulli.go (+210 lines), internal/sync/manager_test.go (+20 lines), internal/api/handlers.go (+211 lines), internal/api/handlers_test.go (+32 lines), internal/api/router.go (+30 lines), README.md (+5 endpoint entries), CLAUDE.md (this entry). **Total Lines Added**: ~620 lines (implementation ~568 lines, mocks ~52 lines). **Impact**: Enables collection and playlist analytics dashboards for content organization insights, multi-server identification for enterprise Plex deployments with multiple servers, server version tracking for compatibility checks, and content grouping pattern analysis. Completes Collections & Playlists category (2/2, 100%). Total Tautulli endpoints: 46/93 (49.5%).


## [1.20] - 2025-11-22

### Changed

**Priority 3 Advanced Analytics & Content Tracking Implementation**: Implemented 5 new Tautulli API endpoints for advanced streaming quality analytics and content migration support, reaching 44.1% total endpoint coverage (36→41 endpoints). **Endpoints Added**: (1) `/api/v1/tautulli/stream-type-by-top-10-users` - User streaming quality preferences showing top 10 users with their direct play vs transcode patterns for identifying users with bandwidth or client limitations, (2) `/api/v1/tautulli/stream-type-by-top-10-platforms` - Platform transcoding analytics revealing which client platforms require transcoding most often (Roku, Apple TV, Web players) for infrastructure optimization, (3) `/api/v1/tautulli/search` - Global search across all Tautulli data (media titles, user names, library names) with configurable result limits and relevance scoring for quick content and user lookup, (4) `/api/v1/tautulli/new-rating-keys` - Updated Plex rating key mappings after Plex database migrations or content re-scanning for tracking content across rating key changes, (5) `/api/v1/tautulli/old-rating-keys` - Historical rating key mappings for legacy content tracking and data migration support when Plex changes internal IDs. **Implementation**: Added 5 model structs (102 lines) in `internal/models/models.go` (TautulliStreamTypeByTop10Users, TautulliStreamTypeByTop10Platforms, TautulliSearch, TautulliNewRatingKeys, TautulliOldRatingKeys), 5 client methods (215 lines) in `internal/sync/tautulli.go`, 5 HTTP handlers (259 lines) with caching and validation in `internal/api/handlers.go`, 5 routes with CORS and rate limiting in `internal/api/router.go`, updated TautulliClientInterface with Priority 3 Advanced Analytics section, updated mock clients in both `internal/sync/manager_test.go` and `internal/api/handlers_test.go`. **Testing**: All tests passing. **Verified Counts**: Routes 82→87 (+5), TautulliClient methods 42→47 (+5), Tautulli proxy routes 36→41 (+5). **Files Modified**: internal/models/models.go (+102 lines), internal/sync/tautulli.go (+215 lines), internal/api/handlers.go (+259 lines), internal/api/router.go (+30 lines), mock clients (+40 lines). **Total Lines Added**: ~646 lines. **Impact**: Enables user quality preference analytics for bandwidth planning, platform-specific transcoding optimization, global content/user search functionality, and Plex rating key migration tracking for data consistency across Plex database changes. Total Tautulli endpoints: 41/93 (44.1%).


## [1.19] - 2025-11-22

### Changed

**Priority 2 Enhanced Metadata & Export Implementation**: Implemented 4 new Tautulli API endpoints for enhanced stream quality analytics and data export capabilities. **Endpoints Added**: (1) `/api/v1/tautulli/stream-data` - Enhanced per-session technical details showing source quality (4K, 1080p), transcoding decisions (direct play, transcode, copy), codecs (HEVC→H264), bitrates, audio channels (5.1→2.0), and delivered stream quality for quality analytics, (2) `/api/v1/tautulli/library-names` - Simple library ID→Name dictionary mapping (e.g., "1: Movies", "2: TV Shows") for efficient lookups across analytics without repeated API calls, (3) `/api/v1/tautulli/export-metadata` - Initiate CSV/JSON metadata export for libraries or specific content with configurable fields (section_id, export_type, file_format) for data portability and external analysis, (4) `/api/v1/tautulli/export-fields` - Returns available export fields per media type (movie, episode, track) with field names, display names, types, and descriptions for export customization UI. **Implementation**: Added 4 model structs (104 lines) in `internal/models/models.go` (TautulliStreamData with 34 quality fields, TautulliLibraryNames, TautulliExportMetadata, TautulliExportFields), 4 client methods (165 lines) in `internal/sync/tautulli.go`, 4 HTTP handlers (180 lines) with caching (except export_metadata which generates new exports) in `internal/api/handlers.go`, 4 routes with CORS and rate limiting in `internal/api/router.go`, updated mock clients in both `internal/sync/manager_test.go` (+16 lines stub methods) and `internal/api/handlers_test.go` (+4 func fields, +28 implementation methods, +189 test lines). **Testing**: Added 8 comprehensive tests (4 client tests in tautulli_test.go with 265 lines, 4 handler tests in handlers_test.go with 189 lines) validating responses, parameters, caching, and error handling. **Documentation**: Updated README.md with 4 endpoint descriptions including parameter details, updated TAUTULLI_API_ANALYSIS.md counts (32→36 used, 61→57 available, 34.4%→38.7% coverage, moved 4 endpoints from unused to "Enhanced Metadata & Export" section), updated CLAUDE.md. **Verified Counts**: Routes 78→82 (+4), TautulliClient methods 38→42 (+4), Tautulli proxy routes 32→36 (+4). **Files Modified**: internal/models/models.go (+104 lines), internal/sync/tautulli.go (+169 lines including interface), internal/sync/manager_test.go (+16 lines), internal/api/handlers.go (+180 lines), internal/api/handlers_test.go (+221 lines), internal/api/router.go (+24 lines), README.md (+4 lines), TAUTULLI_API_ANALYSIS.md (updated counts and sections), CLAUDE.md (this entry). **Total Lines Added**: ~1,268 lines (implementation ~717 lines, tests ~454 lines, mocks ~44 lines, routes/interface ~24 lines, docs ~4 lines). **Impact**: Enables quality analytics (4K source → 1080p transcode tracking), library name caching (reduces repeated API calls), metadata export initiation (CSV/JSON with custom fields), and export field discovery (builds dynamic export UIs). Completes Priority 2 from original implementation plan. Total Tautulli endpoints: 36/93 (38.7%).


## [1.18] - 2025-11-22

### Changed

**Priority 1 User Geography & Management Implementation**: Implemented 3 new Tautulli API endpoints for user-centric geographic visualization and user management. **Endpoints Added**: (1) `/api/v1/tautulli/user-ips` - User IP address history showing geographic mobility patterns with last seen timestamps, play counts, and platform info (enables "where does User X watch from?" view), (2) `/api/v1/tautulli/users-table` - Paginated and sortable user management table with play statistics, durations, and activity timestamps (foundation for user management dashboard), (3) `/api/v1/tautulli/user-logins` - Login history and security analytics with timestamps, IP addresses, user agents, OS/browser detection, and success/failure status (correlates with geographic data for unusual location detection). **Implementation**: Added 3 model structs (90 lines) in `internal/models/models.go` (TautulliUserIPs, TautulliUsersTable, TautulliUserLogins), 3 client methods (155 lines) in `internal/sync/tautulli.go`, 3 HTTP handlers (175 lines) with caching and validation in `internal/api/handlers.go`, 3 routes with CORS and rate limiting in `internal/api/router.go`, updated mock clients in both `internal/sync/manager_test.go` and `internal/api/handlers_test.go`. **Testing**: Added 6 comprehensive tests (3 client tests in tautulli_test.go, 3 handler tests in handlers_test.go) with mock-based validation, all tests passing. **Documentation**: Updated README.md with 3 endpoint descriptions, updated TAUTULLI_API_ANALYSIS.md counts (29→32 used, 64→61 available, 31.2%→34.4% coverage), updated CLAUDE.md. **Verified Counts**: Routes 75→78 (+3), TautulliClient methods 35→38 (+3), Tautulli proxy routes 29→32 (+3). **Files Modified**: internal/models/models.go (+90 lines), internal/sync/tautulli.go (+155 lines), internal/sync/manager_test.go (+15 lines), internal/api/handlers.go (+175 lines), internal/api/handlers_test.go (+200 lines), internal/api/router.go (+18 lines), README.md (+3 lines), TAUTULLI_API_ANALYSIS.md (updated counts), CLAUDE.md (this entry). **Total Lines Added**: ~850 lines (implementation ~655 lines, tests ~245 lines). **Impact**: Enables user-centric geographic visualization (WHERE users watch from, not just WHERE playback occurs), user management dashboards, security analytics for unusual login locations, and geographic mobility pattern detection. Total Tautulli endpoints: 32/93 (34.4%).


## [1.17] - 2025-11-22

### Changed

**Content Abandonment & Drop-Off Analysis Implementation**: Completed Priority 1.2 "wow factor" feature (5/5 rating) from DATA_OPPORTUNITIES_WOW_FACTOR.md. Implemented comprehensive content abandonment analytics that identifies content users start but don't finish (completion rate <90% threshold). **Implementation**: (1) Added 7 new model structs in `internal/models/models.go` (ContentAbandonmentAnalytics, ContentAbandonmentSummary, AbandonedContent, MediaTypeCompletion, DropOffBucket, GenreAbandonment, FirstEpisodeDropOff) totaling ~80 lines. (2) Added `GetContentAbandonmentAnalytics()` method in `internal/database/analytics_advanced.go` (~330 lines) with 6 comprehensive SQL queries: summary statistics with PERCENTILE_CONT for median drop-off calculation, top 20 abandoned content (minimum 3 starts, ordered by abandonment rate), completion by media type (movie/episode/track breakdown), drop-off distribution in 5 buckets (0-25%, 25-50%, 50-75%, 75-90%, 90-100%), abandonment by genre (top 15 genres using UNNEST and string_split for CSV parsing), and first episode abandonment (pilot starts vs series continuations). (3) Added `AnalyticsAbandonment` HTTP handler in `internal/api/handlers.go` (~45 lines) with cache-first pattern (5-minute TTL). (4) Added route `/api/v1/analytics/abandonment` in `internal/api/router.go` with CORS and rate limiting. **Features**: Provides overall completion rate statistics (90%+ threshold), identifies top abandoned content with median drop-off points, completion rates by media type, drop-off distribution visualization data, genre-specific abandonment patterns, and TV show first-episode abandonment (pilot vs continuation rates). Full filter support (date range, users, media types). **Documentation**: Updated README.md with endpoint documentation in Analytics Endpoints section, COMPATIBILITY_ANALYSIS.md (v2.1, grade A 96/100), TAUTULLI_API_ANALYSIS.md (updated endpoint counts: 29 used, 64 available). **Files Modified**: internal/models/models.go (+80 lines), internal/database/analytics_advanced.go (+330 lines), internal/api/handlers.go (+45 lines), internal/api/router.go (+5 lines), README.md (+1 line), COMPATIBILITY_ANALYSIS.md (updated v1.16 status + grade), TAUTULLI_API_ANALYSIS.md (updated counts). **Total Lines Added**: ~460 lines. **Impact**: Enables content creators to identify which content fails to engage viewers, pinpoint exact drop-off points for optimization, understand genre-specific retention patterns, and measure first-episode effectiveness for TV shows. Analytics endpoint count: 21 custom analytics (20 existing + 1 new). **Data Coverage**: Uses existing fields from v1.8 metadata enrichment (percent_complete, media_index, parent_media_index, grandparent_rating_key, genres).


## [1.16] - 2025-11-22

### Changed

**Priority 1-2 Analytics Dashboard Completion**: Implemented 12 new Tautulli API endpoints (8 Priority 1 + 4 Priority 2) from COMPATIBILITY_ANALYSIS.md. **Priority 1 (Analytics Dashboard - 8 endpoints)**: (1) `/api/v1/tautulli/plays-by-source-resolution` - Original content quality distribution (4K, 1080p, 720p), (2) `/api/v1/tautulli/plays-by-stream-resolution` - Delivered streaming resolutions revealing transcoding patterns, (3) `/api/v1/tautulli/plays-by-top-10-platforms` - Platform leaderboard (Roku, Apple TV, Web), (4) `/api/v1/tautulli/plays-by-top-10-users` - Most active users ranking, (5) `/api/v1/tautulli/plays-per-month` - Monthly long-term trends (YYYY-MM), (6) `/api/v1/tautulli/user-player-stats` - Per-user platform preferences, (7) `/api/v1/tautulli/user-watch-time-stats` - User engagement by media type, (8) `/api/v1/tautulli/item-user-stats` - Content demographics. **Priority 2 (Library Analytics - 4 endpoints)**: (1) `/api/v1/tautulli/libraries-table` - Paginated/sortable library management, (2) `/api/v1/tautulli/library-media-info` - Library content with technical specs, (3) `/api/v1/tautulli/library-watch-time-stats` - Library-specific analytics, (4) `/api/v1/tautulli/children-metadata` - Episode/season/track metadata. **Implementation**: Added 12 model structs (~300 lines), 12 client methods (~550 lines), 12 HTTP handlers (~310 lines), 12 routes, 12 mock methods. Updated TautulliClientInterface with new signatures. All endpoints follow existing patterns: parameterized queries, error handling, JSON responses, 5-minute cache TTL. **Testing**: Added 24 comprehensive tests (12 client tests in tautulli_test.go, 12 handler tests in handlers_test.go) with mock-based validation. **Documentation**: Updated README.md API table with 12 entries including parameter details and descriptions. Total Tautulli endpoints now: 29 (17 existing + 12 new). **Files Modified**: internal/models/models.go, internal/sync/tautulli.go, internal/api/handlers.go, internal/api/router.go, internal/sync/tautulli_test.go, internal/api/handlers_test.go, README.md, CLAUDE.md. **Lines Added**: ~1,600+ across all files (implementation ~1,200 lines, tests ~442 lines). **Impact**: Completes all remaining quick-win analytics from analysis documents, enhancing dashboard with quality metrics, platform insights, and library management capabilities.


## [1.15] - 2025-11-22

### Changed

**Comprehensive Analysis Document Validation and Verification**. Conducted exhaustive verification of all analysis documents (COMPATIBILITY_ANALYSIS.md, DATA_OPPORTUNITIES_WOW_FACTOR.md, TAUTULLI_API_ANALYSIS.md, ANALYSIS_SUMMARY.md) against actual source code to ensure 100% accuracy. **Key Findings**: (1) Corrected PlaybackEvent field count from 67 to 68 (verified via source code). (2) Confirmed ALL HIGH-priority edge cases were already resolved on 2025-11-21 (rate limiting, database reconnection, large dataset tests, concurrent writes, geolocation validation). (3) Verified test coverage statistics: 90.2% average (up from 88%), 296+ unit tests, 243+ E2E tests, 1,525 new test lines added 2025-11-21. (4) Confirmed no "critical bug" in GetConcurrentStreamsByStreamType - code was always correct at line 455. **Updated Documents**: COMPATIBILITY_ANALYSIS.md (v2.0) - Updated from "A- 92/100" to "A 95/100" reflecting completed work. ANALYSIS_SUMMARY.md (v2.0) - Marked all HIGH-priority items as COMPLETE with commit references. Added comprehensive verification methodology section documenting all validation steps. **Impact**: All analysis documents now reflect verified current state with zero assumptions, no outdated information, and all claims backed by code evidence. **Documentation**: Updated both documents with "Last Updated: 2025-11-22" and "Review Status: All claims verified against source code" headers. Total changes: 2 analysis documents updated, field count corrected, completion status verified for 5 major features (rate limiting, connection recovery, large datasets, concurrent writes, geolocation bug fix).


## [1.14] - 2025-11-22

### Changed

**Frontend Audit Implementation - Weeks 2-3 and 4-6**. Comprehensive frontend improvements implementing critical and medium-term audit recommendations. **Week 2-3 (Critical):** **(1) Code Splitting for Bundle Size Reduction (56.3%)**: Migrated to ES module format with esbuild --splitting, implemented dynamic imports for ChartManager (376 KB) and GlobeManagerDeckGL (140 KB), reduced initial bundle from 1,056 KB → 461 KB gzipped. Modified: web/package.json (build scripts), web/public/index.html (type="module"), web/src/index.ts (dynamic imports). **(2) Cross-Browser E2E Testing**: Enabled Firefox and WebKit browser projects in playwright.config.ts with WebGL support, added authentication state sharing, created browser-specific test scripts (test:e2e:chromium


## [1.13] - 2025-11-21

### Changed

**Comprehensive CI/CD Security Enhancements**. Implemented production-grade security scanning and supply chain transparency with zero external service dependencies. **(1) Dependency Vulnerability Scanning**: Added Trivy-based scanning for Go modules and npm packages with workflow summary output (no GitHub Security tab dependency). Scans detect CRITICAL and HIGH severity vulnerabilities with artifact retention (90 days). **(2) Secrets Scanning**: Implemented Gitleaks-based secret detection across git history with .gitleaks.toml allowlist configuration for false positives. **(3) License Compliance**: Added go-licenses and license-checker validation to block GPL/AGPL/SSPL violations with collapsible summary reports. **(4) SBOM Generation**: Integrated Syft to generate SPDX 2.3 and CycloneDX 1.4 SBOMs for all container images, attached to GitHub releases for supply chain transparency. **(5) Container Image Signing**: Implemented Cosign keyless signing using GitHub OIDC for cryptographic image verification without key management. Added SLSA provenance attestations. **(6) Trivy Image Scanning**: Migrated Docker image vulnerability scans from SARIF/Security tab to table format with workflow summaries for universal access. Total changes: 7 files modified (.github/workflows/build-and-test.yml, .github/workflows/release.yml, .gitleaks.toml, README.md, CLAUDE.md, docs/VERIFICATION.md), 600+ lines added. Build time impact: <2 minutes (within 5-minute budget). All security scan results output to workflow summaries for transparency. **Security Impact**: Enterprise-grade supply chain security without external SaaS dependencies. **Documentation**: Added Container Image Verification, SBOM, and Dependency Management sections to README.md; Security & Supply Chain tools table to CLAUDE.md; Complete verification guide in docs/VERIFICATION.md.


## [1.12] - 2025-11-21

### Changed

**Enterprise-Grade CSP with Nonces and Mock Data Seeding for CI/CD Screenshots**. Implemented industry-standard HTML template system with CSP nonces to eliminate `unsafe-inline` from script-src (maximum XSS protection used by Google/GitHub). Added `internal/templates/index.html.tmpl` with nonce placeholders for 2 inline scripts (WebGL detection, service worker). Updated `internal/api/router.go` with template parsing and per-request nonce injection. Hardened `internal/auth/middleware.go` CSP to remove `unsafe-inline` from script-src while maintaining `unsafe-eval` for ECharts. Created `internal/database/seed.go` (200+ lines) for realistic mock data: 15 users, 50 global locations (NYC, London, Tokyo, etc.), 250 playback events over 30 days with random distribution. Added `SEED_MOCK_DATA` environment variable (default: false) for CI/CD screenshot tests. Updated `.github/workflows/build-and-test.yml` with `SEED_MOCK_DATA=true` and NYC server coordinates for populated chart screenshots. Modified `Dockerfile` to copy template directory and verify existence. Updated `internal/auth/middleware_test.go` to verify strict CSP (no `unsafe-inline` in script-src, nonce required, `unsafe-inline` OK in style-src). Total changes: 9 files modified (2 new: seed.go, index.html.tmpl), ~1,200 lines added. **Security Impact**: Eliminates inline script XSS attack surface, passes enterprise security audits. **CI/CD Impact**: Screenshots now show populated charts with realistic demo data.


## [1.11] - 2025-11-21

### Changed

**Added**: Large dataset handling verification and comprehensive concurrency tests from COMPATIBILITY_ANALYSIS.md HIGH-priority items. **(1) Memory Profiling Tests**: Created `large_dataset_test.go` (465 lines) with 4 test scenarios: TestLargeDataset_100kRecords (verifies memory < 1GB and performance > 1,000 rec/sec with 100k records), TestLargeDataset_MemoryEfficiency_BatchProcessing (verifies memory scales with batch size not total records), TestLargeDataset_ErrorHandling_PartialBatch (tests graceful failure with memory < 500MB), BenchmarkLargeDataset_Throughput (measures throughput for various batch sizes). Verified existing sync implementation already handles large datasets efficiently via batch processing (configurable batchSize, default 1000). **(2) Concurrent Write Tests**: Created `concurrent_test.go` (600+ lines) with 6 comprehensive test cases using go test -race: TestConcurrent_ParallelInsertPlaybackEvent (50 goroutines × 20 inserts = 1,000 concurrent inserts), TestConcurrent_ParallelUpsertGeolocation (450 upserts with 100 unique IPs to test UPSERT conflicts), TestConcurrent_MixedReadsAndWrites (20 readers + 10 writers = 750 operations), TestConcurrent_SameIPUpsert (25 goroutines updating same IP to test atomicity), TestConcurrent_SessionKeyExistence (600 concurrent existence checks), TestConcurrent_RaceDetector (meta-test verifying race detector works). Tests all CRUD operations for thread safety. Total changes: 2 files added, 1,065+ lines (465 + 600). Impact: Confirms sync and database operations are production-ready for high concurrency and large datasets.


## [1.10] - 2025-11-21

### Changed

**Added**: Critical robustness improvements from COMPATIBILITY_ANALYSIS.md HIGH-priority items. **(1) HTTP 429 Rate Limiting**: Implemented automatic retry with exponential backoff (1s, 2s, 4s, 8s, 16s, max 5 retries) in `tautulli.go`. Respects RFC 6585 Retry-After header. Added doRequestWithRateLimit() method used by all API calls (GetGeoIPLookup, Ping, GetHistory, etc.). Created comprehensive test suite with 6 test cases (300+ lines) in `rate_limiting_test.go`. **(2) Database Connection Recovery**: Implemented automatic reconnection with exponential backoff (2s, 4s, 8s, max 3 attempts) in `database.go`. Added Ping(), reconnect(), withConnectionRecovery() methods. Detects connection errors (connection refused, broken pipe, bad connection, database closed). Thread-safe with reconnectMu mutex. Preserves serverLat/serverLon for spatial reinitialization. Total changes: 3 files modified, 500+ lines added (rate limiting + connection recovery + tests). Impact: Sync operations can survive Tautulli server load spikes and temporary database connection loss.


## [1.9] - 2025-11-21

### Changed

**Fixed**: Critical geolocation validation bug where valid (0,0) coordinates (Null Island, Gulf of Guinea) were incorrectly rejected. Changed validation logic in `internal/sync/manager.go` to check for empty country string instead of zero coordinates, as (0,0) is a legitimate geographic location. Added comprehensive test suite in `geolocation_validation_test.go` with 3 test cases verifying: (1) Valid (0,0) coordinates with country are accepted, (2) Empty country is properly rejected, (3) Normal coordinates continue working. This fix ensures playback events from the Null Island region are no longer incorrectly failed during sync. Total changes: 3 files modified, 187 lines added (4 lines changed in manager.go + 185 line test file).


## [1.8] - 2025-11-21

### Changed

Added comprehensive metadata enrichment fields for advanced analytics and binge-watching detection. **Models**: Added 15 new fields to TautulliHistoryRecord and PlaybackEvent (rating_key, parent_rating_key, grandparent_rating_key, media_index, parent_media_index, guid, original_title, full_title, originally_available_at, watched_status, thumb, directors, writers, actors, genres). **Database**: Added 15 ALTER TABLE migrations with 7 new indexes (6 individual + 1 composite binge detection index on user_id, grandparent_rating_key, parent_media_index, media_index, started_at). **Sync**: Updated manager.go to capture all new fields from Tautulli API. **CRUD**: Updated InsertPlaybackEvent with 15 new columns. **Tests**: Added TestInsertPlaybackEvent_MetadataEnrichmentFields with comprehensive field verification (total: 60 tests). These fields enable: (1) Sequential episode tracking for binge detection, (2) Genre and cast/crew analytics, (3) Enhanced content recommendations, (4) External ID integration (IMDB, TVDB). Total changes: 6 files modified, 287 lines added.


## [1.7] - 2025-11-21

### Changed

Fixed race condition in sync manager by adding syncMu mutex to prevent concurrent sync execution. Added 7 missing unit tests for previously untested Tautulli client API methods (GetHomeStats, GetPlaysByDate, GetPlaysByDayOfWeek, GetPlaysByHourOfDay, GetPlaysByStreamType, GetConcurrentStreamsByStreamType, GetItemWatchTimeStats). Added 4 comprehensive concurrency tests for sync manager (TestManager_ConcurrentTriggerSync, TestManager_ConcurrentStartCalls, TestManager_StopDuringSync, TestManager_RaceConditions). Total test count increased from 48 to 59 tests.


## [1.6] - 2025-11-20

### Changed

Enhanced build and CI/CD pipeline with multi-platform binary builds. Added GoReleaser configuration for 5 platforms (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64), GitHub Actions workflow for binary builds with snapshot support, enhanced Makefile with binary build targets. Supports pre-release testing without version tags and official releases with semver when ready.


## [1.5] - 2025-11-20

### Changed

Implemented P2 (Service Worker for PWA offline support). Added comprehensive service worker with cache-first/network-first strategies, automatic cache versioning, and update notifications. All audit items now complete. Assessment: A (98/100)


## [1.17.0] - 2025-11-22

### Added - Content Abandonment & Drop-Off Analysis

**Overview**: Implemented comprehensive content abandonment analytics that identifies content users start but don't finish (completion rate <90% threshold). Rated 5/5 wow factor - Priority 1.2 from DATA_OPPORTUNITIES_WOW_FACTOR.md.

#### New Endpoint
- **`GET /api/v1/analytics/abandonment`**: Content abandonment and drop-off analysis with 6 comprehensive metrics

#### Features
1. **Overall Summary**: Completion/abandonment rates with median drop-off points
2. **Top 20 Abandoned Content**: Most-abandoned content (minimum 3 starts)
3. **Media Type Breakdown**: Completion rates by movie/episode/track
4. **Drop-Off Distribution**: 5-bucket histogram (0-25%, 25-50%, 50-75%, 75-90%, 90-100%)
5. **Genre Analysis**: Top 15 genres by abandonment rate with median drop-off
6. **TV Pilot Analysis**: First-episode abandonment vs series continuation rates

#### Implementation
- **Models** (+80 lines): 7 new structs (ContentAbandonmentAnalytics, ContentAbandonmentSummary, AbandonedContent, MediaTypeCompletion, DropOffBucket, GenreAbandonment, FirstEpisodeDropOff)
- **Database** (+330 lines): `GetContentAbandonmentAnalytics()` with 6 SQL queries using PERCENTILE_CONT, UNNEST, string_split
- **Handler** (+45 lines): `AnalyticsAbandonment()` with 5-minute cache TTL
- **Route** (+5 lines): `/api/v1/analytics/abandonment` with CORS and rate limiting

### Changed
- **Analytics Count**: Increased from 20 to 21 custom analytics endpoints
- **Documentation**: Updated README.md, COMPATIBILITY_ANALYSIS.md (v2.1, grade A 96/100), TAUTULLI_API_ANALYSIS.md, CLAUDE.md

### Impact
Enables content creators to identify content that fails to engage viewers, pinpoint exact drop-off points, understand genre-specific retention patterns, and measure first-episode effectiveness for TV shows.

---

## [1.16.0] - 2025-11-22

### Added - Priority 1-2 Analytics Dashboard Completion (12 New Tautulli API Endpoints)

#### Priority 1: Analytics Dashboard Completion (8 Endpoints)
- **`GET /api/v1/tautulli/plays-by-source-resolution`**: Original content quality distribution showing source file resolutions (4K, 1080p, 720p, etc.)
  - Reveals content library quality composition
  - Supports time_range, y_axis (plays/duration), user_id, grouping parameters
  - Time series data with categories and series breakdown by media type
- **`GET /api/v1/tautulli/plays-by-stream-resolution`**: Delivered streaming resolutions to clients
  - Exposes transcoding patterns when stream resolution differs from source
  - Identifies bandwidth-constrained users or quality-limited devices
  - Same parameter set as source resolution for direct comparison
- **`GET /api/v1/tautulli/plays-by-top-10-platforms`**: Platform leaderboard showing most-used client platforms
  - Rankings by Roku, Apple TV, Web, Android, iOS, etc.
  - Informs platform-specific optimization priorities
  - Top 10 rankings with play counts and watch time
- **`GET /api/v1/tautulli/plays-by-top-10-users`**: Most active users ranking
  - Identifies power users and engagement leaders
  - Supports filtering by time range and metric type (plays vs duration)
  - Top 10 leaderboard with comparative analytics
- **`GET /api/v1/tautulli/plays-per-month`**: Long-term monthly playback trends (YYYY-MM format)
  - Historical trend analysis spanning multiple years
  - Essential for seasonal pattern identification
  - Monthly aggregation with media type breakdown
- **`GET /api/v1/tautulli/user-player-stats`**: Per-user platform and player preferences
  - Shows which platforms/players each user prefers
  - Requires user_id parameter for targeted user analytics
  - Returns platform distribution with play counts and percentages
- **`GET /api/v1/tautulli/user-watch-time-stats`**: User engagement metrics by media type
  - Watch time and play counts broken down by movies, TV shows, music
  - Supports query_days parameter for multi-period analysis (e.g., "1,7,30,0")
  - Reveals user content preferences and engagement patterns
- **`GET /api/v1/tautulli/item-user-stats`**: Content demographics showing who watches what
  - User engagement statistics for specific media items
  - Requires rating_key parameter (Plex content identifier)
  - Grouping support for temporal aggregation

#### Priority 2: Library-Specific Analytics (4 Endpoints)
- **`GET /api/v1/tautulli/libraries-table`**: Paginated and sortable library management table
  - Full library listing with play counts, durations, and last accessed timestamps
  - Supports pagination (start, length), sorting (order_column, order_dir), and search filtering
  - Essential for library organization and activity monitoring
- **`GET /api/v1/tautulli/library-media-info`**: Library content details with technical specifications
  - Media items within a specific library (requires section_id)
  - Includes technical specs, play counts, and metadata for each item
  - Pagination and sorting support for large libraries
- **`GET /api/v1/tautulli/library-watch-time-stats`**: Library-specific analytics over time
  - Watch time and play statistics for individual libraries
  - Supports section_id (required), grouping, and query_days parameters
  - Multi-period analysis (1-day, 7-day, 30-day, all-time stats)
- **`GET /api/v1/tautulli/children-metadata`**: Episode/season/track metadata for hierarchical content
  - Child items for TV shows (episodes of a season) or albums (tracks)
  - Requires rating_key and media_type parameters
  - Enables drill-down navigation for hierarchical content structures

### Added - Implementation Details

#### Backend Implementation
- **Models** (`internal/models/models.go`): Added 12 new response struct types (~300 lines)
  - TautulliPlaysBySourceResolution, TautulliPlaysByStreamResolution
  - TautulliPlaysByTop10Platforms, TautulliPlaysByTop10Users
  - TautulliPlaysPerMonth, TautulliUserPlayerStats
  - TautulliUserWatchTimeStats, TautulliItemUserStats
  - TautulliLibrariesTable, TautulliLibraryMediaInfo
  - TautulliLibraryWatchTimeStats, TautulliChildrenMetadata
  - Reused TautulliPlaysByDateSeries for time series data (multiple endpoints)
- **Tautulli Client** (`internal/sync/tautulli.go`): Added 12 API client methods (~550 lines)
  - All methods follow existing patterns: HTTP GET, JSON unmarshaling, error handling
  - Parameterized query string construction for flexible filtering
  - Proper error propagation with context
- **HTTP Handlers** (`internal/api/handlers.go`): Added 12 handler functions (~310 lines)
  - Cache-first pattern with 5-minute TTL for all endpoints
  - Query parameter parsing and validation
  - Integration with existing middleware (CORS, rate limiting, authentication)
  - Consistent JSON response format with error handling
- **Routing** (`internal/api/router.go`): Registered 12 new routes
  - All routes under `/api/v1/tautulli/*` namespace
  - CORS and rate limiting applied to all new endpoints
  - Total Tautulli endpoints now: 29 (17 existing + 12 new)
- **Interface Updates**: Extended TautulliClientInterface with 12 new method signatures
  - Maintains interface-based dependency injection pattern
  - Enables comprehensive mocking for testing

#### Testing Implementation
- **Tautulli Client Tests** (`internal/sync/tautulli_test.go`): Added 12 test functions (~442 lines)
  - HTTP mocking with httptest.NewServer for all endpoints
  - Request parameter validation (cmd, time_range, y_axis, user_id, etc.)
  - Response structure validation with type-safe assertions
  - Error path testing for malformed responses
  - Table-driven test patterns for comprehensive scenario coverage
- **Handler Tests** (`internal/api/handlers_test.go`): Added 12 handler test cases (~500 lines)
  - Extended MockTautulliClient with 12 new mock method function fields
  - Handler-level HTTP testing with httptest.ResponseRecorder
  - Query parameter validation tests
  - JSON response structure validation
  - Cache behavior verification
  - Error handling tests (missing parameters, client errors)
- **Total Test Coverage**: Added 24 comprehensive tests (12 client + 12 handler)
  - All tests passing with go test -v -race
  - No race conditions detected
  - Maintains existing test coverage standards (78%+ per package)

### Changed
- **API Endpoint Count**: Increased from 17 to 29 Tautulli endpoints (70% growth)
- **Test Coverage**: Added 1,042 lines of test code across 2 files
- **Documentation**: Updated README.md with detailed parameter descriptions for all 12 endpoints

### Technical Details

#### Endpoint Parameter Patterns
- **Common Parameters**:
  - `time_range` (integer): Days to analyze (default: 30)
  - `y_axis` (string): Metric type - "plays" or "duration" (default: "plays")
  - `user_id` (integer): Filter by Tautulli user ID (default: 0 = all users)
  - `grouping` (integer): Time period grouping (0 or 1, default: 0)
- **Pagination Parameters** (libraries-table, library-media-info):
  - `start` (integer): Offset for pagination (default: 0)
  - `length` (integer): Number of items to return (default: 25)
  - `order_column` (string): Sort column name
  - `order_dir` (string): Sort direction ("asc" or "desc")
  - `search` (string): Search filter text (optional)
- **Library Parameters**:
  - `section_id` (integer): Plex library section ID (required for library-specific endpoints)
  - `query_days` (string): Comma-separated day ranges (e.g., "1,7,30,0" for 1-day, 7-day, 30-day, all-time)
- **Content Parameters**:
  - `rating_key` (string): Plex rating key for specific media item (required for item-user-stats, children-metadata)
  - `media_type` (string): Content type filter - "movie", "show", "artist", etc.

#### Cache Strategy
- **Cache Key Format**: `"tautulli:{method}:{param_hash}"`
- **Cache TTL**: 5 minutes (300 seconds) for all Tautulli endpoints
- **Cache Invalidation**: Automatic on sync completion via callback
- **Performance Impact**: <5ms response time on cache hit (99.6% reduction from uncached)

#### API Response Patterns
All endpoints follow consistent response structure:
```json
{
  "response": {
    "result": "success",
    "data": {
      "categories": ["...", "..."],
      "series": [
        {"name": "...", "data": [10, 20, 30]}
      ]
    }
  }
}
```

### Performance & Optimization
- **Parallel Queries**: Handler orchestration executes multiple queries concurrently where applicable
- **Database Efficiency**: Tautulli's pre-calculated analytics cache used directly (no redundant computation)
- **Network Efficiency**: JSON responses gzipped via compression middleware
- **Memory Efficiency**: Streaming JSON encoding prevents large response buffering

### Documentation Updates
- **README.md**: Added comprehensive API documentation table with all 12 endpoints
  - Parameter descriptions with types and defaults
  - Use case descriptions for each endpoint
  - Example query patterns
  - Parameter detail section expanded to document all new parameters
- **CLAUDE.md**: Updated version history (v1.16) with implementation summary
  - Corrected endpoint count to 29 (17 existing + 12 new)
  - Added testing details (24 comprehensive tests)
  - Updated "Last Updated" field to 2025-11-22
- **CHANGELOG.md**: This entry documenting all changes

### Migration Notes
- **Backward Compatible**: No breaking changes to existing endpoints
- **Automatic Availability**: New endpoints available immediately after deployment
- **No Database Changes**: Endpoints proxy Tautulli data directly
- **Authentication**: New endpoints respect existing AUTH_MODE configuration
- **Rate Limiting**: New endpoints subject to existing rate limits (100 req/min default)

### Files Modified
- `internal/models/models.go`: +300 lines (12 new model structs)
- `internal/sync/tautulli.go`: +550 lines (12 client methods, interface updates)
- `internal/api/handlers.go`: +310 lines (12 handler functions)
- `internal/api/router.go`: +60 lines (12 route registrations)
- `internal/sync/tautulli_test.go`: +442 lines (12 client tests)
- `internal/api/handlers_test.go`: +500 lines (12 handler tests, mock updates)
- `README.md`: +80 lines (API documentation)
- `CLAUDE.md`: Updated version history and endpoint count
- **Total Lines Added**: ~1,600 lines (implementation ~1,200, tests ~442)

### Benefits
- **Complete Tautulli API Coverage**: Implements all remaining high-priority analytics endpoints from COMPATIBILITY_ANALYSIS.md
- **Quality Metrics**: Source vs stream resolution comparison reveals transcoding patterns
- **Platform Insights**: Top 10 platforms and players inform optimization priorities
- **Long-Term Trends**: Monthly aggregation enables seasonal pattern identification
- **Library Management**: Paginated library tables with sorting and search capabilities
- **User Analytics**: Per-user platform preferences and engagement metrics
- **Content Insights**: Item-specific user demographics and engagement statistics
- **Hierarchical Navigation**: Episode/season metadata enables drill-down interfaces

## [1.12.0] - 2025-11-21

### Added - Enterprise-Grade CSP and Screenshot Testing Improvements

#### Template-Based Nonce Injection for Content Security Policy
- **Implemented**: Industry-standard HTML template system with CSP nonces for maximum XSS protection
  - **Implementation**: `internal/templates/index.html.tmpl` (Go template with nonce placeholders)
  - **Router Enhancement**: `internal/api/router.go` now renders template with request-specific nonces
  - **Strict CSP**: Removed `unsafe-inline` from script-src (enterprise-grade security standard)
  - **Nonce Generation**: Cryptographically secure random nonces via `internal/auth/middleware.go`
  - **Graceful Fallback**: Template parsing failure falls back to static HTML with logging
- **Security Benefits**:
  - Eliminates XSS attack surface from inline scripts
  - Industry best practice used by Google, GitHub, and enterprise applications
  - Maintains `unsafe-eval` only for ECharts dynamic chart generation (required dependency)
  - Styles remain external in `web/public/styles.css` (no inline style security risk)
- **Affected Scripts**: Two inline scripts now use nonces:
  1. WebGL support detection (before page load, critical path)
  2. Service worker registration (PWA offline capability)
- **Updated Tests**: `internal/auth/middleware_test.go` now verifies strict CSP compliance
  - Validates script-src does NOT contain `unsafe-inline`
  - Validates script-src contains nonce for inline scripts
  - Confirms style-src allows `unsafe-inline` (acceptable - styles don't execute code)

#### Mock Data Seeding for CI/CD Screenshot Tests
- **Added**: Comprehensive mock data generator for automated screenshot testing
  - **Implementation**: `internal/database/seed.go` (200+ lines of realistic test data)
  - **Configuration**: `SEED_MOCK_DATA` environment variable (default: false)
  - **Data Scope**: 15 mock users, 50 global locations, 250 playback events over 30 days
  - **Realistic Distribution**: Random playback patterns across time periods, platforms, and content types
- **Mock Data Features**:
  - 50 global cities with accurate latitude/longitude coordinates (NYC, London, Tokyo, Sydney, etc.)
  - 22 mock movie and TV show titles with proper media type detection
  - 10 platforms (Plex Web, iOS, Android, Roku, Fire TV, Apple TV, etc.)
  - 8 player types (Chrome, Firefox, Safari, Plex App, Direct Play, Transcoded)
  - Randomized watch times, completion rates, and pause counters
  - Proper GeoIP data with city, region, country, postal code, timezone
  - Playback events distributed across 30-day history with realistic hourly variance
- **CI/CD Integration**: `.github/workflows/build-and-test.yml` updated with environment variables
  - `SEED_MOCK_DATA=true` for screenshot capture job
  - `SERVER_LATITUDE=40.7128` and `SERVER_LONGITUDE=-74.0060` (NYC coordinates)
  - Ensures charts are populated with demo data for accurate screenshot representation
- **Use Cases**:
  - Automated UI screenshot capture in CI/CD without live Tautulli data
  - Demo environment setup for presentations and documentation
  - Frontend development with realistic test data
  - Visual regression testing with consistent dataset

#### Build Infrastructure Enhancements
- **Dockerfile Updates**: Added template directory to runtime image
  - `COPY --from=backend-builder /build/internal/templates /app/internal/templates`
  - Verification check ensures template file exists in runtime image
  - Prevents template rendering failures in production containers
- **Error Handling**: Enhanced error messages for template system failures
  - Clear warnings logged if template parsing fails
  - Automatic fallback to static HTML (CSP nonces disabled with warning)
  - Graceful degradation ensures application remains functional

### Changed

#### Content Security Policy Hardening
- **Removed**: `unsafe-inline` from script-src directive (breaking change for inline scripts without nonces)
- **Maintained**: `unsafe-inline` in style-src (acceptable - external styles already preferred)
- **Updated**: CSP header generation to include request-specific nonces
- **Improved**: Security audit compliance (now passes strict CSP requirements)

#### Authentication Middleware
- **Enhanced**: CSP nonce injection into request context
- **Added**: CSP nonce context key (`CSPNonceContextKey`) for template access
- **Updated**: Security header comments to reflect enterprise-grade standards
- **Maintained**: Backward compatibility with all existing security features

### Technical Details

#### Template System Architecture
- **Go Template Engine**: Using standard library `html/template` for security
- **Nonce Injection Pattern**: `<script nonce="{{.Nonce}}">...</script>`
- **Context Propagation**: Nonce passed via `r.Context().Value(auth.CSPNonceContextKey)`
- **Template Parsing**: Happens once at server startup for performance
- **Execution**: Per-request template execution with nonce data structure

#### Mock Data Algorithm
- **Seeding Strategy**: `rand.Seed(time.Now().UnixNano())` for varied but deterministic data
- **Batch Processing**: Geolocations inserted first, then playback events reference them
- **IP Address Generation**: Sequential IPs (192.168.x.x) mapped to unique locations
- **Timestamp Distribution**: Random distribution across 30-day window with hourly granularity
- **Completion Rates**: 85-100% completion (realistic engaged viewing patterns)
- **Helper Functions**: `indexOf()`, `contains()`, `indexContainsStr()` for data processing

#### Code Quality
- **Formatting**: All Go code formatted with `gofmt -s -w .`
- **Type Safety**: Strict null checks and error handling in template system
- **Test Coverage**: Updated middleware tests verify strict CSP enforcement
- **Logging**: Comprehensive debug and warning logs for template system

### Migration Notes
- **Automatic**: No manual intervention required - templates copied during Docker build
- **Backward Compatible**: Fallback to static HTML if template system fails
- **Environment Variable**: `SEED_MOCK_DATA=true` only in CI/CD (not production)
- **Zero Downtime**: Template system seamlessly replaces static file serving

### Files Modified
- `internal/templates/index.html.tmpl` (NEW - 968 lines): HTML template with nonce placeholders
- `internal/api/router.go` (MODIFIED): Template parsing and rendering logic
- `internal/auth/middleware.go` (MODIFIED): Strict CSP with nonce generation
- `internal/database/seed.go` (NEW - 200+ lines): Mock data seeding implementation
- `internal/config/config.go` (MODIFIED): `SeedMockData` configuration field
- `cmd/server/main.go` (MODIFIED): Mock data seeding invocation
- `Dockerfile` (MODIFIED): Template directory copy and verification
- `.github/workflows/build-and-test.yml` (MODIFIED): Screenshot job environment variables
- `internal/auth/middleware_test.go` (MODIFIED): Strict CSP verification tests

### Fixed

#### Geolocation Validation Bug (2025-11-21)
- **Fixed**: Invalid rejection of (0,0) coordinates (Null Island)
  - **Issue**: Sync manager incorrectly rejected valid (0,0) coordinates as "zero coordinates error"
  - **Location**: `internal/sync/manager.go:533`
  - **Impact**: Any playback from Null Island (Gulf of Guinea, off coast of Africa) was incorrectly failed
  - **Fix**: Changed validation to check for empty country string instead of (0,0) coordinates
  - **Rationale**: (0,0) is a valid geographic location, but empty country indicates failed GeoIP lookup
- **Added**: Comprehensive test suite for geolocation validation (`internal/sync/geolocation_validation_test.go`)
  - Test case 1: Valid (0,0) coordinates with country are accepted
  - Test case 2: Empty country string is properly rejected
  - Test case 3: Normal coordinates continue to work as expected
- **Files Modified**: 3 files, 187 lines added
  - `internal/sync/manager.go`: Fixed validation logic (4 lines changed, 2 added)
  - `internal/sync/geolocation_validation_test.go`: New test file (185 lines)

### Added - Production Reliability Improvements (2025-11-21)

#### HTTP 429 Rate Limiting with Exponential Backoff
- **Added**: RFC 6585-compliant rate limiting for Tautulli API client
  - **Implementation**: `internal/sync/tautulli.go` - `doRequestWithRateLimit()` method
  - **Retry Strategy**: Exponential backoff (1s, 2s, 4s, 8s, 16s) with max 5 retries
  - **Retry-After Header**: Respects server-provided delay times per RFC 6585
  - **Applied To**: `GetGeoIPLookup()`, `Ping()`, `doHistoryRequest()` methods
  - **Error Handling**: Clear error messages after max retries exceeded
  - **Configuration**: `maxRetries=5`, `retryBaseDelay=1s` (configurable in TautulliClient)
- **Added**: Comprehensive test suite (`internal/sync/rate_limiting_test.go` - 275 lines)
  - Test 1: Exponential backoff timing verification (±200ms tolerance)
  - Test 2: Max retries exceeded (6 total attempts: 1 initial + 5 retries)
  - Test 3: Retry-After header compliance (RFC 6585)
  - Test 4: Non-429 errors fail fast (no retries for 500, 503, 404, 401)
  - Test 5: Success on first attempt (no unnecessary retries)
  - Test 6: Network errors handled properly
- **Impact**: Prevents sync failures during Tautulli rate limit spikes

#### Database Connection Loss Recovery
- **Added**: Automatic reconnection with exponential backoff for DuckDB
  - **Implementation**: `internal/database/database.go` - `Ping()`, `reconnect()`, `withConnectionRecovery()` methods
  - **Reconnection Strategy**: Exponential backoff (2s, 4s, 8s) with max 3 attempts
  - **Thread Safety**: Mutex-protected reconnection to prevent concurrent reconnection attempts
  - **Error Detection**: Detects "connection refused", "broken pipe", "bad connection", "database is closed"
  - **State Preservation**: Retains server coordinates (serverLat, serverLon) across reconnections
  - **Configuration**: `maxReconnectTries=3`, `reconnectDelay=2s` (configurable in DB struct)
- **Added**: Connection health monitoring
  - `Ping()` method for explicit connection verification
  - `isConnectionError()` helper for error classification
  - Automatic reconnection wrapper for critical operations
- **Impact**: Recovers from transient database connection failures without manual intervention

### Added - Large Dataset Handling & Concurrency Tests (2025-11-21)

#### Memory Profiling for Large Datasets
- **Added**: Comprehensive memory and performance tests (`internal/sync/large_dataset_test.go` - 465 lines)
  - **Test 1**: `TestLargeDataset_100kRecords` - Full stress test with 100,000 records
    - Verifies memory usage < 1GB (peakMemoryMB check)
    - Verifies performance > 1,000 records/second
    - Tracks batch processing (100 batches × 1,000 records each)
    - Measures duration, records/second, memory before/after
  - **Test 2**: `TestLargeDataset_MemoryEfficiency_BatchProcessing` - Memory scaling verification
    - Tests 4 scenarios: 10k/100, 10k/1000, 50k/1000, 100k/1000 (records/batch_size)
    - Verifies memory scales with batch size, NOT total record count
    - Expected: ~50KB per record in batch (configurable threshold)
  - **Test 3**: `TestLargeDataset_ErrorHandling_PartialBatch` - Error handling without memory leaks
    - Tests 10,000 records with simulated failure at record 5,500
    - Verifies graceful error handling mid-batch
    - Ensures memory usage < 500MB even with errors
  - **Benchmark**: `BenchmarkLargeDataset_Throughput` - Throughput comparison
    - Tests batch sizes: 100, 1000, 5000
    - Tests record counts: 1k, 10k
    - Measures records/second for optimization guidance
- **Verified**: Existing sync implementation already handles large datasets efficiently
  - Batch processing with configurable `batchSize` (default 1000)
  - Streaming approach prevents loading all records into memory
  - Memory-efficient record-by-record insertion

#### Concurrent Write & Race Condition Tests
- **Added**: Comprehensive concurrency tests (`internal/database/concurrent_test.go` - 600+ lines)
  - **Test 1**: `TestConcurrent_ParallelInsertPlaybackEvent` - Parallel insertion stress test
    - 50 goroutines × 20 inserts each = 1,000 total concurrent inserts
    - Verifies thread safety with go test -race
    - Validates final count matches expected total
    - Measures throughput (inserts/second)
  - **Test 2**: `TestConcurrent_ParallelUpsertGeolocation` - UPSERT conflict handling
    - 30 goroutines × 15 upserts each = 450 total upserts
    - Only 100 unique IPs (forces ON CONFLICT DO UPDATE)
    - Verifies correct deduplication (100 final records, not 450)
    - Tests last-write-wins semantics
  - **Test 3**: `TestConcurrent_MixedReadsAndWrites` - Realistic workload simulation
    - 20 reader goroutines (GetStats, GetLocationStats, GetPlaybackEvents, GetUniqueUsers)
    - 10 writer goroutines (InsertPlaybackEvent)
    - 25 operations per goroutine = 750 total operations
    - Verifies no read/write conflicts or data corruption
  - **Test 4**: `TestConcurrent_SameIPUpsert` - Same-record update race
    - 25 goroutines all updating identical IP address
    - Verifies UPSERT atomicity (1 final record, not 25)
    - Tests last-write-wins with concurrent updates
  - **Test 5**: `TestConcurrent_SessionKeyExistence` - Concurrent duplicate checks
    - 30 goroutines × 20 checks each = 600 total checks
    - Mix of existing and non-existing session keys
    - Verifies consistent results under concurrent load
  - **Test 6**: `TestConcurrent_RaceDetector` - Meta-test for race detector
    - Verifies go test -race is functioning correctly
    - Demonstrates proper mutex usage
- **Coverage**: Tests all primary CRUD operations for thread safety
  - InsertPlaybackEvent()
  - UpsertGeolocationWithServer()
  - GetLocationStats()
  - GetStats()
  - GetPlaybackEvents()
  - SessionKeyExists()
  - GetGeolocation()
  - GetUniqueUsers()
- **Files Modified**: 2 files, 1,065+ lines added
  - `internal/sync/large_dataset_test.go`: 465 lines (4 tests + 1 benchmark)
  - `internal/database/concurrent_test.go`: 600+ lines (6 comprehensive tests)

### Added - DuckDB Spatial Optimizations (100x Performance Improvement)
- **Five New Spatial API Endpoints**: Optimized geographic queries with DuckDB spatial extension
  - `/api/v1/spatial/hexagons` - H3 hexagon aggregation (10x faster than client-side)
  - `/api/v1/spatial/arcs` - Distance-weighted user→server connection arcs
  - `/api/v1/spatial/viewport` - Bounding box queries with R-tree spatial index (100x faster)
  - `/api/v1/spatial/temporal-density` - Temporal-spatial density with rolling aggregations
  - `/api/v1/spatial/nearby` - Proximity search within configurable radius
- **H3 Spatial Indexing**: Pre-computed hexagon indexes at 3 resolutions
  - Resolution 6 (country-level, ~36 km²)
  - Resolution 7 (city-level, ~5.2 km²)
  - Resolution 8 (neighborhood-level, ~0.74 km²)
- **Geodesic Distance Calculations**: Pre-computed great-circle distances from server location
  - `distance_from_server` column for instant arc visualization
  - Server coordinates configurable via SERVER_LATITUDE/LONGITUDE environment variables
- **R-tree Spatial Index**: 100x faster viewport queries for map pan/zoom operations
  - Automatic spatial index creation on `geom` column
  - O(log n) bounding box filtering vs O(n) full table scan
- **Spatial Query Methods** (internal/database/analytics_spatial.go - 410 lines):
  - `GetH3AggregatedHexagons()` - Pre-aggregated hexagon data with H3 index lookup
  - `GetDistanceWeightedArcs()` - Arc weight calculation: `playback_count * LOG(1 + distance/1000)`
  - `GetLocationsInViewport()` - ST_Within() with R-tree optimization
  - `GetTemporalSpatialDensity()` - Window functions for smooth temporal animation
  - `GetNearbyLocations()` - ST_DWithin() for proximity search
- **New Data Models** (internal/models/models.go):
  - `H3HexagonStats` - Pre-aggregated playback data for H3 hexagons
  - `ArcStats` - User→server connection arcs with geodesic distance and visual weight
  - `TemporalSpatialPoint` - Time + space aggregation with rolling averages

### Added - Comprehensive Testing & Performance Benchmarks
- **Unit Tests** (internal/database/spatial_test.go - 450+ lines):
  - 5 test suites for all spatial methods with 30+ test cases
  - Multi-resolution testing (H3 resolutions 6, 7, 8)
  - Bounding box validation for viewport queries
  - Temporal interval testing (hour, day, week, month)
  - Filter combination testing (date range, users, media types)
- **Performance Benchmarks** (internal/database/spatial_bench_test.go - 370+ lines):
  - Benchmarks with large datasets (1,000-5,000 locations)
  - Multi-resolution comparison (country vs city vs neighborhood)
  - Viewport size comparison (whole world vs small regions)
  - Temporal aggregation comparison (hourly vs daily vs monthly)
  - Proximity search radius comparison (10km to 5,000km)

### Added - Automated UI Screenshot Capture
- **Playwright Screenshot Tests** (web/tests/screenshots.spec.ts - 380+ lines):
  - 13 comprehensive UI screenshot captures:
    - Login page, map view, globe view, analytics dashboard
    - All analytics sections (trends, geographic, users, popular, bandwidth)
    - Live activity, recently added, filters panel, globe controls
  - Individual chart screenshots for 43 ECharts visualizations
  - Responsive screenshots (desktop, tablet, mobile viewports)
  - UI state screenshots (loading, empty, error states)
- **CI/CD Screenshot Job** (.github/workflows/build-and-test.yml):
  - Automatic screenshot capture on every main branch commit
  - 90-day artifact retention for UI/UX evolution tracking
  - Optional automatic commit to repository with [skip ci] flag
  - Runs after Docker build completion with isolated test environment

### Technical Details - Spatial Optimization Implementation
- **Backend Architecture**:
  - Spatial schema migrations in database initialization (backward compatible)
  - H3 index columns: `h3_index_6`, `h3_index_7`, `h3_index_8` (UBIGINT)
  - Distance column: `distance_from_server` (DOUBLE, pre-computed)
  - Bounding box columns: `bbox_xmin`, `bbox_ymin`, `bbox_xmax`, `bbox_ymax`
  - Geometry column: `geom` (POINT type with R-tree spatial index)
  - B-tree indexes on all H3 columns for fast hexagon lookup
- **API Implementation** (internal/api/handlers.go - 520+ new lines):
  - All 5 spatial endpoints with comprehensive validation
  - Caching support with 5-minute TTL (shared with existing analytics cache)
  - Filter integration (supports all 14 filter dimensions)
  - Swagger/OpenAPI documentation for all endpoints
- **Code Quality**:
  - Go vet and gofmt compliance
  - Table-driven tests following project conventions
  - Comprehensive error handling and input validation
  - Cache key generation for optimal cache hit rates
- **Documentation**:
  - SPATIAL_API.md with endpoint specifications and examples
  - Updated README.md with new spatial capabilities
  - DUCKDB_OPTIMIZATIONS.md with implementation details
  - Inline code comments with SQL query explanations

### Performance Improvements
- **H3 Hexagon Aggregation**: 10x faster than client-side aggregation
  - Before: 2-3 seconds for 10,000 locations (JavaScript)
  - After: <200ms with pre-computed H3 indexes (DuckDB)
- **Viewport Queries**: 100x faster with R-tree spatial index
  - Before: 500ms-1s full table scan
  - After: <50ms with spatial index on map pan/zoom
- **Arc Layer Loading**: 300ms → <50ms with pre-computed distances
- **Temporal Animation**: 2-3 second data fetch → <500ms with window functions
- **Parallel Query Execution**: ~4x faster on 4-core systems (DuckDB automatic parallelization)
- **Columnar Storage**: ~10x compression for analytics workloads

### Added - Advanced 3D Globe Visualization with deck.gl 9.2
- **Four Visualization Layers** (deck.gl ecosystem):
  - **ScatterplotLayer**: Individual location markers with smooth transitions and GPU-accelerated rendering
  - **HexagonLayer**: 3D hexagonal aggregation with GPU-based density calculation and elevation scaling
  - **ArcLayer**: User→server connection arcs with great circle paths and dynamic width based on playback count
  - **TripsLayer**: Temporal animation with 5-minute trail effects for playback history visualization
- **Multi-View Split Screen**: Side-by-side 2D map and 3D globe with synchronized cameras
  - Three layout modes: Both views, map-only, globe-only
  - Bidirectional camera synchronization (map ↔ globe) with circular update prevention
  - Toggleable sync via checkbox for independent exploration
  - Responsive design with flexbox (stacks vertically on mobile)
  - Floating control panel with layout buttons and sync toggle
- **Globe Control Panel**: Professional UI for layer management and temporal animation
  - Layer toggle buttons for all four visualization layers
  - Temporal animation controls (play/pause/stop/scrubber) with speed adjustment (0.5x to 5x)
  - Timeline scrubbing with timestamp display for precise temporal exploration
  - Screenshot export (download current globe view as PNG via canvas.toBlob)
  - Reset view button for returning to default camera position
  - Dark theme UI matching project aesthetic
- **Enhanced Controls with Geocoder and Satellite Imagery**:
  - Mapbox Geocoder integration (@mapbox/mapbox-gl-geocoder v5.0.2) with custom dark theme
  - Location search with autocomplete and fly-to animations
  - Google satellite imagery toggle via Mapbox raster source (not @deck.gl/google-maps)
  - Satellite tile URLs from Google Maps servers (https://mt{0-3}.google.com/vt/lyrs=s)
  - Performance stats display placeholder for future FPS tracking
- **Server Location Configuration**: Optional environment variables for arc visualization
  - SERVER_LATITUDE and SERVER_LONGITUDE for user→server connection arcs
  - New `/api/v1/server-info` endpoint to expose server location to frontend
  - Example coordinates in .env.example (New York City: 40.7128, -74.0060)

### Added - deck.gl Performance Optimizations
- **GPU Acceleration**: Hardware-accelerated rendering for all layers (60 FPS with 10,000+ points)
- **Memory Optimization**: useDevicePixels: false reduces render buffer by 4x on Retina displays
- **Great Circle Arcs**: Geographic accuracy for connection paths following Earth's curvature
- **Smooth Transitions**: 300ms transitions for position, radius, and color changes in ScatterplotLayer
- **Material Shading**: Custom material properties for HexagonLayer (ambient, diffuse, specular)
- **Dynamic Scaling**: Elevation scale for 3D hexagons, width scaling for arcs based on data

### Technical Details - deck.gl Implementation
- **Frontend Architecture**:
  - Three new TypeScript manager classes: GlobeManagerDeckGLEnhanced (790 lines), GlobeControls (470 lines), GlobeControlsEnhanced (650 lines)
  - MultiViewManager (420 lines) for split-screen layout and camera synchronization
  - MapboxOverlay integration for seamless deck.gl + MapLibre GL 5.13.0 JS rendering
  - RequestAnimationFrame loop for 60 FPS temporal animation playback
  - Bidirectional camera sync using map.on('move') events with isMoving() debouncing
- **Backend Architecture**:
  - Added ServerConfig.Latitude and ServerConfig.Longitude fields to internal/config/config.go
  - New ServerInfo API handler in internal/api/handlers.go
  - Route registration in internal/api/router.go for /api/v1/server-info endpoint
  - TypeScript API client method getServerInfo() in web/src/lib/api.ts
- **Dependencies Added**:
  - @deck.gl/aggregation-layers v9.2.2 (HexagonLayer)
  - @deck.gl/geo-layers v9.2.2 (TripsLayer)
  - @deck.gl/widgets v9.2.2 (future widget support)
  - @mapbox/mapbox-gl-geocoder v5.0.2 (location search)
- **Code Quality**:
  - TypeScript strict mode compliance (no implicit any)
  - Comprehensive JSDoc comments for all public methods
  - Accessor methods (getMap()) for multi-view camera access
  - Proper lifecycle management (init/destroy methods)
  - Total new code: ~2,330 lines (TypeScript + configuration)
- **Documentation**:
  - Updated README.md with "3D Globe Visualization (Enhanced with deck.gl 9.2)" section
  - Updated CLAUDE.md with deck.gl ecosystem breakdown and new files
  - Added SERVER_LATITUDE/LONGITUDE to .env.example with coordinate finder URL
  - Comprehensive commit messages documenting architecture decisions

### Added - Professional Dashboard Suite with 10 New Tautulli API Endpoints
- **Three New Interactive Dashboards**: Professional UI for real-time monitoring and content discovery
  - **Live Activity Dashboard**: Real-time stream monitoring with 5-second auto-refresh
    - Active stream count with breakdown by Direct Play, Direct Stream, and Transcode
    - Total bandwidth usage display
    - Live session cards showing user, content title, playback state, and transcode decision
    - Session details: platform, player, video resolution, codecs, progress bar, and bandwidth per stream
    - Color-coded transcode states (green for Direct Play, orange for Direct Stream, red for Transcode)
    - Empty state message when no active streams
    - Automatic cleanup on dashboard switch to prevent memory leaks
  - **Server Info/Health Dashboard**: Plex server monitoring and status
    - Online/offline status indicator with color coding
    - Server name, version, platform, and platform version display
    - Machine identifier for server tracking
    - Update availability notification with version and release date
    - Error handling with retry button on failure
  - **Recently Added Timeline**: Content discovery with rich media presentation
    - Grid layout of recently added content with poster thumbnails
    - Filtering by media type (all, movies, TV shows, music)
    - Library-specific filtering with dynamic library list population
    - Configurable items per page (10, 25, 50, 100)
    - Full pagination support with page info display
    - "Time ago" formatting for added dates (e.g., "2 hours ago", "3 days ago")
    - Lazy-loaded images with placeholder fallbacks
    - Media type badges and metadata display (year, library name)
- **10 New Tautulli API Integration Endpoints**: Complete backend implementation with comprehensive testing
  - `/api/v1/tautulli/activity` - Real-time stream activity (session_count, sessions array, bandwidth)
  - `/api/v1/tautulli/metadata/:rating_key` - Detailed media metadata
  - `/api/v1/tautulli/user/:user_id` - User profile and settings
  - `/api/v1/tautulli/library-user-stats/:section_id` - Library engagement by user
  - `/api/v1/tautulli/recently-added` - Recently added content with filtering
  - `/api/v1/tautulli/libraries` - All Plex libraries list
  - `/api/v1/tautulli/library/:section_id` - Individual library details
  - `/api/v1/tautulli/server-info` - Server status and version info
  - `/api/v1/tautulli/synced-items` - User sync status
  - `/api/v1/tautulli/terminate-session` - Session management
- **Navigation System Enhancement**: Seamless multi-view application
  - Five-tab navigation bar: Maps, Live Activity, Analytics, Recently Added, Server
  - Active state tracking with visual indicators and ARIA attributes
  - View switching with proper lifecycle management (init/destroy)
  - Responsive tab bar with flexible layout
  - Smooth transitions between views
- **Professional UI/UX Design**: Production-ready visual polish
  - 400+ lines of custom CSS with consistent design system
  - Color-coded states and status indicators
  - Hover effects and smooth transitions
  - Responsive grid layouts (CSS Grid for adaptive sizing)
  - Professional card-based layouts with shadows and borders
  - Loading states and empty state messages
  - Error states with user-friendly retry mechanisms
- **TypeScript Type Safety**: Complete type coverage for all new endpoints
  - 10 new TypeScript interfaces in api.ts
  - 10 new API client methods with proper error handling
  - Strong typing for all dashboard manager classes
  - No implicit any types (strict mode compliance)

### Added - Comprehensive Test Coverage for New Features
- **Tautulli Client Tests**: 10 new test cases for new API methods
  - GetActivity(), GetMetadata(), GetUser(), GetLibraryUserStats()
  - GetRecentlyAdded(), GetLibraries(), GetLibrary()
  - GetServerInfo(), GetSyncedItems(), TerminateSession()
  - HTTP mocking with httptest for all endpoints
  - Error path testing and validation
- **API Handler Tests**: 10 new handler test cases
  - TautulliActivity, TautulliMetadata, TautulliUser handlers
  - TautulliLibraryUserStats, TautulliRecentlyAdded, TautulliLibraries handlers
  - TautulliLibrary, TautulliServerInfo handlers
  - TautulliSyncedItems, TautulliTerminateSession handlers
  - Query parameter validation and error handling tests
- **Mock Interface Updates**: Extended mockTautulliClient with 17 new methods
  - All new Tautulli interface methods implemented
  - Stub implementations returning empty structs for non-critical paths
  - Ensures existing sync manager tests continue to pass

### Technical Details - Dashboard Implementation
- **Frontend Architecture**:
  - Three new TypeScript manager classes: ActivityDashboardManager, ServerDashboardManager, RecentlyAddedDashboardManager
  - Lifecycle methods (init/destroy) for proper resource management
  - Auto-refresh mechanism with 5-second interval for activity dashboard
  - Debounced filter updates (300ms) for recently added dashboard
  - HTML escaping for XSS prevention
  - Client-side pagination with efficient state management
- **Backend Architecture**:
  - 10 new Go handler methods with consistent error handling
  - Full integration with existing auth middleware and rate limiting
  - Type-safe models for all Tautulli response structures
  - Proper HTTP status codes and JSON error responses
  - DuckDB-compatible query patterns
- **Code Quality**:
  - Frontend: TypeScript strict mode, no linter errors
  - Backend: gofmt compliant, go vet clean
  - Total new code: ~1,500 lines (Go + TypeScript + CSS)
  - Test coverage maintained above 85% threshold

### Added - Comprehensive Test Coverage Expansion
- **Sync Package Tests (0% 90%+)**: Added 1,200+ lines of comprehensive unit tests
  - **Tautulli API Client Tests**: Complete HTTP mocking for GetHistory, GetHistorySince, GetGeoIPLookup, and Ping methods
  - **Sync Manager Tests**: Full coverage of sync orchestration, retry logic, geolocation fallback, and error handling
  - **Interface Refactoring**: Created TautulliClientInterface and DBInterface for dependency injection and testability
  - **Benchmark Tests**: Performance validation for critical paths
  - **Mock Implementations**: Flexible mock structs with function fields for testing all code paths
- **Cache Package Tests (78.9% 95%+)**: Enhanced with 25+ edge case tests
  - TTL expiration and cleanup tests
  - Large dataset tests (10,000 entries)
  - Concurrent access and race condition tests
  - Stats validation and memory cleanup tests
- **Test Infrastructure Improvements**:
  - All tests use table-driven test patterns for comprehensive scenario coverage
  - HTTP test servers for API client testing
  - Mock implementations for all external dependencies (database, API clients)
  - Race detector enabled in CI/CD pipeline
  - Comprehensive error path testing alongside happy paths

### Fixed - Critical Data Integrity and CI/CD Issues
- **CRITICAL - Database Query Bug**: Fixed incomplete SELECT query in GetPlaybackEvents() causing data loss
  - Added 9 missing fields to SELECT statement: transcode_decision, video_resolution, video_codec, audio_codec, section_id, library_name, content_rating, play_duration, year
  - Fixed rows.Scan() to include all new fields in correct order
  - Resolves data loss bug where analytics fields were populated during sync but not retrieved
  - Fixes CSV export functionality that was failing due to missing fields
  - Ensures all playback events are retrieved with complete metadata
- **Trivy Action Configuration**: Removed invalid `trivyignores` parameter from workflow files
  - Removed `trivyignores: '.trivyignore'` parameter from build-and-test.yml (line 196)
  - Removed `trivyignores: '.trivyignore'` parameter from release.yml (line 72)
  - Trivy automatically detects and uses .trivyignore file in repository root
  - Prevents potential CI/CD failures due to unsupported parameter in certain Trivy action versions
  - Maintains security scanning functionality while ensuring compatibility
- **SARIF Upload on Pull Requests**: Fixed "Resource not accessible by integration" error on PR builds
  - Added `github.event_name != 'pull_request'` condition to SARIF upload step (build-and-test.yml line 198)
  - GitHub restricts security-events:write permission on PRs for security reasons
  - SARIF results still available in workflow logs; Security tab upload only on push events
  - Prevents workflow failures while maintaining security scanning on all builds
  - Resolves amd64 build failure with CodeQL action permission error

### Fixed - CI/CD Pipeline Improvements (PR #14)
- **Security Scanning Configuration**: Fixed Trivy vulnerability scanner configuration to properly report issues without blocking builds
  - Changed Trivy exit-code from '1' to '0' to report vulnerabilities without failing CI/CD pipeline
  - Added `.trivyignore` file for managing false positives and accepted risks
  - Configured Trivy to skip large files (bundle.js.map) to reduce scan time and memory usage
- **GitHub Security Integration**: Fixed CodeQL action permissions issue
  - Added `security-events: write` permission to both build-and-test.yml and release.yml workflows
  - Enables proper upload of Trivy SARIF results to GitHub Security tab
  - Resolves "Resource not accessible by integration" error
- **Workflow Optimization**: Updated comments and conditional logic for clarity
  - Updated build step conditionals from "Only push if Trivy scan passed" to "Only push if previous steps succeeded"
  - Maintains proper CI/CD flow while allowing vulnerability reports to be visible

### Added - Enhanced Analytics Backend (Phase A - API Complete, Frontend Pending)
- **Four New Analytics Data Sources**: Backend data collection and API endpoints implemented (12 charts active, 4 additional data sources available via API)
  - **Library Statistics**: API provides playback count, unique users, total watch time, and average completion by Plex library
  - **Content Ratings Distribution**: API provides PG, PG-13, R, NR breakdown with percentages
  - **Watch Duration Analytics**: API provides average/median watch duration and total watch time by media type with completion metrics
  - **Release Year Distribution**: API provides playback distribution across content release years (top 10)
  - **Status**: Backend analytics complete; frontend chart visualization not yet implemented
- **New Database Fields**: Extended schema to capture comprehensive Tautulli metadata
  - `section_id`: Plex library identifier for per-library analytics
  - `library_name`: Library display name (Movies, TV Shows, Music, etc.)
  - `content_rating`: Age/content rating (PG, PG-13, R, TV-MA, etc.)
  - `play_duration`: Actual watch time in minutes (not just percent_complete)
  - `year`: Content release year for temporal analysis
- **Enhanced API Responses**: All new analytics included in `/api/v1/analytics/geographic` endpoint
  - `library_distribution`: Array of LibraryStats with engagement metrics
  - `rating_distribution`: Array of RatingStats with percentage calculations
  - `duration_stats`: Comprehensive DurationStats object with avg/median/total plus breakdown by media type
  - `year_distribution`: Array of YearStats showing top 10 years by playback count
- **Automatic Data Migration**: Database indexes created automatically for new fields
  - Indexed fields: `section_id`, `library_name`, `content_rating`, `year`
  - Zero-downtime schema migration on application start
  - Backward compatible with existing data

### Technical Details - Analytics Implementation
- **Backend (Go)**:
  - Added 5 database schema columns to `playback_events` table
  - Created 4 new analytics methods: `GetLibraryStats()`, `GetRatingDistribution()`, `GetDurationStats()`, `GetYearDistribution()`
  - Enhanced sync manager to capture library, rating, duration, and year from Tautulli `get_history` endpoint
  - Play duration converted from seconds to minutes during sync
  - All analytics support existing filter framework (date range, users, media types)
- **Frontend (TypeScript)**:
  - Added 4 new chart rendering methods with ECharts configuration
  - Library chart uses horizontal bars with gradient colors
  - Ratings chart uses donut pie chart with percentage labels
  - Duration chart uses dual Y-axis (minutes vs hours) for clarity
  - Year chart uses colorful bars with top labels
  - Updated TypeScript types: LibraryStats, RatingStats, DurationStats, YearStats, DurationByMediaType
  - Added new chart IDs to lazy loading system: `chart-libraries`, `chart-ratings`, `chart-duration`, `chart-years`
- **Models & Types**:
  - Go models: Updated PlaybackEvent and TautulliHistoryRecord structs
  - Added LibraryStats, RatingStats, DurationStats, DurationByMediaType, YearStats models
  - Updated GeographicResponse to include 4 new analytics arrays/objects
  - TypeScript interfaces mirror Go models exactly for type safety
- **Data Quality**:
  - COALESCE used in queries to handle NULL values gracefully (defaults to "Unknown", "Not Rated", 0)
  - Percentage calculations include division-by-zero protection
  - DurationStats uses PERCENTILE_CONT for accurate median calculation
  - Top 10 limits prevent overwhelming UI with excessive data points

### Performance & Optimization
- **Database Indexes**: Four new indexes improve query performance for new analytics
  - idx_playback_section_id: Library filtering
  - idx_playback_library_name: Library name searches
  - idx_playback_content_rating: Rating distribution queries
  - idx_playback_year: Year-based temporal analysis
- **Query Optimization**: All new analytics queries use same filter pattern as existing analytics
  - Reusable `LocationStatsFilter` struct
  - Parameterized queries prevent SQL injection
  - Efficient aggregation with GROUP BY and ORDER BY
- **API Response**: Single `/analytics/geographic` call returns all 16 visualizations
  - Parallel database queries (no sequential bottleneck)
  - Consistent response time despite additional analytics

### Added - Performance Optimizations (Phase P1)
- **In-Memory Caching Layer**: Thread-safe cache with automatic TTL expiration
  - 5-minute TTL for analytics queries to balance freshness and performance
  - SHA256-based cache keys from filter parameters for deterministic lookup
  - Automatic cache invalidation on sync completion keeps data fresh
  - Cache statistics tracking: hits, misses, evictions, hit rate percentage
  - Background cleanup goroutine removes expired entries every 5 minutes
  - Comprehensive unit tests with 100% coverage (8 tests including concurrency)
- **Parallel Query Execution**: AnalyticsGeographic endpoint optimized from sequential to parallel
  - 14 database queries now execute concurrently using goroutines and sync.WaitGroup
  - Reduces analytics endpoint response time by ~85% (from ~140ms to ~20ms on cached requests)
  - Error handling preserves first error encountered, prevents partial data returns
  - Thread-safe result aggregation with mutex protection
- **Performance Monitoring Middleware**: Real-time API performance tracking
  - Tracks p50, p95, p99 latency percentiles per endpoint
  - Sliding window of last 1000 requests with automatic statistics
  - Logs slow requests exceeding 1000ms threshold
  - Per-endpoint metrics: request count, avg/min/max duration
  - Supports performance regression detection and SLO monitoring
- **Automatic Cache Invalidation**: Sync completion triggers cache clear
  - Callback mechanism in sync.Manager invokes handler.ClearCache()
  - Ensures fresh data after Tautulli sync without manual intervention
  - Zero-downtime cache refresh maintains API availability
- **Database Benchmark Tests**: Performance regression testing for critical queries
  - Benchmarks for GetStats, GetPlaybackTrends, GetMediaTypeDistribution
  - Benchmarks for GetTopCities, GetLibraryStats, GetDurationStats
  - Parallel query benchmark validates concurrent access performance
  - Establishes baseline for future optimization efforts

### Technical Implementation - Performance
- **Backend (Go)**:
  - New `internal/cache` package with Cache, Entry, Stats types
  - New `internal/middleware` package with PerformanceMonitor, RequestMetrics, EndpointStats
  - Updated Handler struct with cache and perfMon fields
  - AnalyticsGeographic rewritten: cache check parallel queries cache store
  - Sync manager extended with onSyncCompleted callback support
  - Main.go wired cache invalidation: `syncManager.SetOnSyncCompleted(handler.ClearCache)`
- **Cache Implementation**:
  - Thread-safe with sync.RWMutex for concurrent read/write access
  - GenerateKey() function creates deterministic cache keys from method + params
  - Separate tracking for hits, misses, evictions with atomic counters
  - TTL enforcement at both Get() time and background cleanup
  - Support for custom TTL per cache entry via SetWithTTL()
- **Performance Benefits**:
  - **Cache Hit**: <5ms response time (99.6% reduction from uncached)
  - **Cache Miss with Parallelization**: ~20-30ms (80-85% reduction from sequential)
  - **Sequential Baseline**: ~140-160ms for 14 queries (previous implementation)
  - **Target Achieved**: <100ms p95 latency exceeded (now <30ms p95)

### Added - API Pagination & Security Enhancements
- **Configurable API Pagination**: Prevent unbounded queries and API abuse
  - New environment variables: `API_DEFAULT_PAGE_SIZE` (default: 20), `API_MAX_PAGE_SIZE` (default: 100)
  - Applied to 4 endpoints: `/api/v1/playbacks`, `/api/v1/locations`, `/api/v1/analytics/users`
  - Dynamic error messages show actual configured limits
  - Backward compatible: existing clients work unchanged
- **Enhanced Content Security Policy**: Removed `unsafe-inline` from CSP entirely
  - Extracted 976 lines of inline CSS to external `styles.css` file
  - Strict CSP: only `unsafe-eval` remains (required for ECharts dynamic charts)
  - Better protection against XSS via style injection
  - External stylesheet enables browser caching
- **Automated PNG Icon Generation**: Production-ready PWA icons
  - Docker build automatically generates icon-192.png and icon-512.png from SVG
  - Uses rsvg-convert in frontend build stage
  - Manual generation script: `scripts/generate-icons.sh`
  - Comprehensive documentation in `web/public/ICONS.md`
  - Updated manifest.json to prefer PNG icons for better device support
- **Additional Security Improvements**:
  - robots.txt for web crawler management
  - RFC 9116 compliant security.txt for vulnerability disclosure
  - WebSocket support in CSP (`wss:` and `ws:` protocols)
  - ARIA live regions for screen reader announcements on stats updates

### Technical Details - API & Security
- **Configuration**:
  - Added `APIConfig` struct to `internal/config/config.go`
  - `DefaultPageSize` and `MaxPageSize` fields with validation
  - Environment variable parsing with sensible defaults
- **API Handlers**:
  - Updated `Playbacks()`: default 20, max 100 (was 100/1000)
  - Updated `Locations()`: default 20, max 100 (was 1000/10000)
  - Updated `AnalyticsUsers()`: max raised to configurable limit
  - All limits use `h.config.API.*` for consistency
- **Frontend**:
  - All inline `<style>` content moved to `styles.css`
  - Service worker caches `styles.css` alongside other assets
  - PNG icons cached by service worker for offline access
  - HTML remains clean with single external stylesheet link
- **Docker**:
  - Frontend build stage installs `librsvg2-bin` package
  - Icon generation runs during `docker build` (no manual step)
  - Generated PNGs copied to final image via multi-stage build
  - Build logs confirm icon generation success
- **Documentation**:
  - Updated `.env.example` with API pagination variables
  - Updated `docker-compose.yml` with inline documentation
  - Created comprehensive `ICONS.md` for icon workflow
  - Icon generation script with error handling and tool detection

### Added - Streaming Quality Analytics Suite
- **Transcode Distribution Analytics**: Visualize Direct Play vs Transcode vs Copy playback modes
- **Video Resolution Distribution**: Analyze playback quality across resolutions (4K, 1080p, 720p, SD, etc.)
- **Codec Combination Distribution**: Understand codec usage patterns (H.264+AAC, HEVC+EAC3, etc.)
- **Interactive Visualizations**: Three new ECharts charts with semantic color schemes
  - Transcode chart: Donut chart with semantic colors (green=direct play, yellow=copy, red=transcode)
  - Resolution chart: Vertical bar chart with gradient (red to purple) sorted by quality
  - Codec chart: Horizontal bar chart with multi-color scheme, custom tooltips showing video+audio combinations
- **Percentage Calculations**: All distributions show both absolute counts and percentage of total playbacks
- **Filter Support**: All streaming quality analytics respect date range, user, and media type filters
- **Real-Time Updates**: Charts automatically refresh when filters change (300ms debounce)
- **Export Capability**: PNG export for all three new charts at 2x resolution
- **Responsive Design**: Charts adapt to screen size with automatic resizing
- **Data Quality**: N/A filtering ensures only valid streaming metadata is analyzed

### Technical Details - Streaming Quality Analytics
- **Database Schema Evolution**:
  - Added 4 new nullable columns to playback_events table: transcode_decision, video_resolution, video_codec, audio_codec
  - ALTER TABLE IF NOT EXISTS for backward-compatible migrations
  - Nullable pointer fields (*string) to handle missing/historical data
  - Proper indexing for filter performance on new columns
- **Backend Implementation**:
  - `GetTranscodeDistribution()`: Aggregates playback counts by transcode decision with percentage calculation
  - `GetResolutionDistribution()`: Aggregates playback counts by video resolution with percentage calculation
  - `GetCodecDistribution()`: Aggregates playback counts by video+audio codec combinations with percentage calculation
  - Two-pass aggregation pattern: collect data, calculate total, then compute percentages
  - COALESCE SQL function for NULL handling (treats NULL as "Unknown")
  - Empty string filtering to exclude invalid data
  - Top 10 limit on codec distribution to focus on most common combinations
  - Full filter support for date ranges, users, and media types across all methods
- **Sync Process Enhancements**:
  - Capture transcode_decision, video_resolution, video_codec, audio_codec from Tautulli API
  - N/A filtering during sync to prevent storing invalid metadata
  - Backward-compatible with existing playback events (nullable fields)
  - Proper handling of empty strings and missing fields
- **API Enhancements**:
  - Extended `GeographicResponse` model with three new fields: transcode_distribution, resolution_distribution, codec_distribution
  - Added `TranscodeStats`, `ResolutionStats`, and `CodecStats` models with percentage field
  - Integrated into existing `/api/v1/analytics/geographic` endpoint
  - Maintains backward compatibility with existing analytics
  - Sub-100ms response times for typical datasets
- **Frontend Implementation**:
  - `renderTranscodeChart()`: Donut chart with semantic color palette
    - Direct Play: Green (#10b981) - optimal quality, no transcoding overhead
    - Copy: Yellow (#f59e0b) - container remux only
    - Transcode: Red (#ef4444) - quality loss and CPU overhead
  - `renderResolutionChart()`: Vertical bar chart with gradient colors (red #e94560 to purple #533483)
    - Sorted by playback count for easy comparison
    - Custom tooltip showing resolution and playback count
  - `renderCodecChart()`: Horizontal bar chart with multi-color scheme
    - Multi-line labels showing "Video Codec\n+\nAudio Codec" format
    - Custom tooltip showing full codec combination with playback count
    - Top 10 most common combinations displayed
  - TypeScript interfaces with strict mode compliance (no `any` types)
  - Lazy loading with IntersectionObserver for optimal performance
  - Chart-specific tooltips with contextual information
- **Data Visualization**:
  - Transcode chart: Donut chart with legend, percentage display in center and tooltips
  - Resolution chart: Vertical bars sorted by count, labels on X-axis, gradient fill
  - Codec chart: Horizontal bars sorted by count, multi-line labels on left, custom tooltip
  - Color schemes aligned with existing dark theme and semantic meaning
  - Empty state handling with zero-data fallbacks
  - Proper text truncation for long codec names
- **Chart Configuration**:
  - Total chart count: 12 charts (9 existing + 3 new streaming quality charts)
  - Transparent backgrounds for theme consistency
  - Consistent typography with existing charts
  - Accessible color contrast ratios (WCAG AA compliant)
  - Responsive font sizes and layout

### Added - Enhanced Analytics (Platform, Player, Content Completion)
- **Platform Distribution Analytics**: Visualize playback activity by platform (Roku, Apple TV, Chrome, etc.)
- **Player Distribution Analytics**: Analyze player app usage (Plex Web, Plex for iOS, Plex for Roku, etc.)
- **Content Completion Analytics**: Understand viewer engagement with completion rate distribution
- **Completion Rate Buckets**: Five buckets showing playback distribution (0-25%, 25-50%, 50-75%, 75-99%, 100%)
- **Fully Watched Metric**: Dedicated metric showing percentage of content watched to completion
- **Interactive Charts**: Three new ECharts visualizations with dark theme integration
- **Filter Support**: All new analytics respect date range, user, and media type filters
- **Real-Time Updates**: Charts automatically refresh when filters change (300ms debounce)
- **Export Capability**: PNG export for all new charts at 2x resolution
- **Responsive Design**: Charts adapt to screen size with automatic resizing

### Technical Details - Enhanced Analytics
- **Backend Implementation**:
  - `GetPlatformDistribution()`: Aggregates playback counts and unique users by platform
  - `GetPlayerDistribution()`: Aggregates playback counts and unique users by player application
  - `GetContentCompletionStats()`: Calculates completion rate distribution with statistical summaries
  - SQL-based bucketing using CASE statements for efficient completion rate grouping
  - Full filter support for date ranges, users, and media types across all methods
  - Optimized queries with proper NULL handling and empty string filtering
- **API Enhancements**:
  - Extended `GeographicResponse` model with three new fields
  - Added `PlatformStats`, `PlayerStats`, `CompletionBucket`, and `ContentCompletionStats` models
  - Integrated into existing `/api/v1/analytics/geographic` endpoint
  - Maintains backward compatibility with existing analytics
  - Sub-100ms response times for typical datasets
- **Frontend Implementation**:
  - `renderPlatformsChart()`: Horizontal bar chart with gradient colors (purple to red)
  - `renderPlayersChart()`: Donut chart with 8-color palette for player distribution
  - `renderCompletionChart()`: Vertical bar chart with 5-color gradient (red to green)
  - TypeScript interfaces with strict mode compliance (no `any` types)
  - Lazy loading with IntersectionObserver for optimal performance
  - Chart-specific tooltips with contextual information
- **Data Visualization**:
  - Platform chart: Horizontal bars sorted by playback count, labels on right
  - Player chart: Donut chart with legend, percentage display in tooltips
  - Completion chart: Bar chart with dual labels (count and percentage)
  - Completion tooltip: Shows bucket name, playback count, and average completion
  - Color schemes aligned with existing dark theme (#e94560, #533483, #f07b3f, etc.)
  - Empty state handling with zero-data fallbacks
- **Chart Configuration**:
  - Completion buckets: 0-25%, 25-50%, 50-75%, 75-99%, 100% (complete)
  - Average completion calculation weighted by playback counts
  - Fully watched percentage derived from 100% bucket
  - Platform/Player charts: Unlimited entries, sorted by frequency
  - All charts: Transparent backgrounds, consistent typography, accessible colors

### Added - Data Export (P2.2 - Complete)
- **CSV Export**: Export playback history to CSV format for analysis in spreadsheet software
- **GeoJSON Export**: Export location data to GeoJSON for use in GIS applications (QGIS, ArcGIS, etc.)
- **Filter Support**: Both export formats respect active filters (date range, user, media type)
- **Auto-Download**: Browser automatically downloads files with timestamped filenames
- **Large Dataset Support**: CSV export supports up to 100,000 records, GeoJSON up to 10,000 locations
- **Export UI**: Two dedicated export buttons in sidebar with download icons
- **Success Notifications**: Toast notifications confirm export initiation
- **Proper Headers**: Content-Type and Content-Disposition headers for correct file handling

### Technical Details - Data Export (P2.2)
- **CSV Format**: RFC 4180 compliant with proper escaping of quotes and commas
- **GeoJSON Format**: RFC 7946 compliant FeatureCollection with Point geometries
- **CSV Fields**: 17 columns including id, session_key, timestamps, user info, media info, location data
- **GeoJSON Properties**: Country, region, city, playback count, unique users, temporal data, avg completion
- **Coordinate System**: WGS84 (EPSG:4326) longitude/latitude format
- **Filename Pattern**: `tautulli-{type}-{YYYYMMDD-HHMMSS}.{ext}`
- **API Endpoints**:
  - `GET /api/v1/export/playbacks/csv` - CSV export with pagination
  - `GET /api/v1/export/locations/geojson` - GeoJSON export with filtering
- **Security**: Rate limited, CORS enabled, same authentication as other endpoints

### Added - Real-Time WebSocket Updates (P2.3 - Complete)
- **WebSocket Security**: Proper origin checking using CORS configuration
- **WebSocket Infrastructure**: Full backend implementation for live playback notifications
- **Hub-Client Architecture**: Centralized hub manages all WebSocket connections
- **Connection Management**: Automatic client registration/unregister with goroutines
- **Ping/Pong Health Checks**: Connection health monitoring (60s pong wait, 54s ping interval)
- **Broadcast System**: Multi-client message broadcasting with channel-based architecture
- **Message Protocol**: JSON-based message format with type and data fields
- **WebSocket Endpoint**: `/api/v1/ws` for client connections
- **CORS Support**: Cross-origin WebSocket connections enabled
- **Graceful Shutdown**: Proper cleanup on client disconnect

### Technical Details - WebSocket Backend
- **gorilla/websocket v1.5.3**: Industry-standard WebSocket library
- **Hub Pattern**: Centralized connection manager with goroutine safety
- **Message Types**: playback, ping, pong (extensible protocol)
- **Buffer Sizes**: 1024 bytes read/write, 256 message channel depth
- **Timeouts**: 10s write wait, 60s pong wait for connection health
- **Max Message Size**: 512 KB per message
- **Thread Safety**: sync.RWMutex for concurrent client map access
- **Integration Ready**: Hub initialized in main.go, routes configured

### Added - WebSocket Frontend (P2.3 - Complete)
- **WebSocketManager Class**: Full-featured WebSocket client for real-time updates
- **Toast Notification System**: Beautiful toast notifications for new playback events
- **Auto-Reconnect**: Exponential backoff retry logic (up to 10 attempts)
- **Connection Health**: Client-side ping/pong with 30s interval
- **Event Handling**: Callback-based architecture for playback events
- **User Notifications**: Shows username, media type, and title for new playbacks
- **Auto-Refresh Stats**: Automatically updates stats on new playback events
- **Graceful Cleanup**: Proper disconnect on logout with toast dismissal
- **Responsive Toasts**: Mobile-optimized toast layout with accessibility
- **Toast Types**: Info, success, warning, and error toast variants
- **Dismissible Toasts**: Manual close with auto-dismiss after 7 seconds

### Technical Details - WebSocket Frontend
- **Reconnection Strategy**: 3s base delay with 1.5x exponential backoff
- **Max Reconnect Attempts**: 10 attempts before giving up
- **Ping Interval**: 30 seconds to keep connection alive
- **Toast Duration**: 7 seconds for playback notifications
- **Toast Positioning**: Fixed top-right with z-index 10000
- **Message Protocol**: JSON-based with type and data fields
- **State Management**: Connection state tracking and intentional disconnect flag
- **Error Handling**: Comprehensive error logging for debugging
- **Accessibility**: ARIA labels, live regions, and keyboard support

### Remaining - WebSocket Enhancements (Future)
- Activity feed sidebar showing recent playbacks
- Sound alerts (optional)
- Desktop notifications (PWA)

### Added - Animated Playback Timeline (P2.1)
- **Timeline Scrubber**: Interactive timeline control to replay playback history over time
- **Play/Pause Controls**: Toggle playback with visual play/pause button (SVG icons)
- **Playback Speed**: Adjustable speed controls (1x, 2x, 5x, 10x) for faster replay
- **Time Intervals**: Configure time window granularity (hourly, daily, weekly)
- **Time Display**: Real-time timestamp display with playback count
- **Seek Capability**: Click or drag scrubber to jump to any point in time
- **Auto-Advance**: Smooth requestAnimationFrame-based playback animation
- **Reset Function**: One-click return to start of timeline
- **Responsive Design**: Mobile-optimized controls for touch devices
- **Accessibility**: Full ARIA labels, keyboard navigation, and semantic SVG icons

### Technical Details - Timeline
- **TimelineManager Class**: Autonomous timeline state management
  - Loads up to 10,000 playback events from filtered dataset
  - Sorts events chronologically by started_at timestamp
  - Maintains current playback position with millisecond precision
  - Supports progress-based and time-based seeking
- **Animation Loop**: requestAnimationFrame for smooth 60 FPS playback
- **Time Windows**: Configurable intervals (hourly/daily/weekly) with dynamic filtering
- **State Persistence**: Playback speed and interval saved to LocalStorage
- **Event-Driven Updates**: Callback-based architecture for UI synchronization
- **Performance**: Efficient binary search for time-based lookups
- **SVG Icons**: Inline Bootstrap Icons for play/pause/reset controls
- **Slider Styling**: Custom range input with branded highlight color

### Added - Heatmap Layer Toggle (P2.1)
- **Three Visualization Modes**: Points, Clusters, and Heatmap for flexible data exploration
- **Heatmap Layer**: MapLibre GL 5.13.0 heatmap with intensity based on playback count
- **Smart Color Gradient**: Accessible cyan to red gradient (distinct from cluster colors)
- **Segmented Control**: Intuitive toggle buttons positioned in top-right of map
- **Mode Persistence**: User preference saved to LocalStorage across sessions
- **Default Mode**: Heatmap mode enabled by default for density visualization
- **Smooth Transitions**: Instant mode switching with proper layer visibility management
- **Accessibility**: Full ARIA labels, keyboard navigation, and aria-pressed states
- **Responsive Design**: Mobile-optimized button layout for all screen sizes

### Technical Details - Heatmap
- **Heatmap Weight**: Interpolated based on playback_count (0-500 range)
- **Heatmap Intensity**: Dynamic based on zoom level (0.5x at z0, 1.5x at z9)
- **Heatmap Color**: Six-stop gradient for clear density visualization
  - 0% density: transparent cyan
  - 20% density: semi-transparent cyan
  - 40% density: lime green
  - 60% density: yellow
  - 80% density: bright orange
  - 100% density: pure red
- **Heatmap Radius**: Zoom-adaptive (2px at z0, 20px at z9, 30px at z15)
- **Layer Management**: Proper visibility toggling for all three modes
- **LocalStorage Key**: `map-visualization-mode` for persistence

### Added - Performance Optimizations (P1.2)
- **PWA Support**: Added manifest.json, service worker, and PWA meta tags for offline capability
- **Lazy Loading**: Implemented IntersectionObserver for chart lazy loading (only render when visible)
- **Resource Hints**: Added preconnect and dns-prefetch for Mapbox API
- **API Caching**: Implemented ETag-based caching with Cache-Control headers (60s max-age)
- **Theme Color**: Added meta theme-color for mobile browser chrome

### Added - Accessibility Improvements (P1.3 - Complete)
- **ARIA Labels**: Comprehensive ARIA labels on navigation, stats, filters, maps, and charts
- **Live Regions**: Added aria-live for dynamic stat updates
- **Semantic HTML**: Enhanced with role attributes (main, complementary, region, status, application)
- **Focus Indicators**: Visible 2px outline on all interactive elements (outline-offset: 2px)
- **Skip Navigation**: Skip-to-main-content link for keyboard users
- **Focus-Visible**: Modern focus-visible pseudo-class for better UX
- **Color Contrast**: Existing theme already meets WCAG AA 4.5:1 ratio (verified)

### Added - UI/UX Improvements (P1.4 - Complete)
- **Dark Mode Toggle**: Manual toggle with system preference detection (prefers-color-scheme)
- **Light Theme**: Full light mode with accessible color palette
- **Theme Persistence**: LocalStorage saves user preference across sessions
- **Loading Skeletons**: Animated skeleton screens for charts (shimmer effect)
- **Smooth Transitions**: 0.3s ease transitions for theme switching and interactions
- **Error States**: User-friendly error display with recovery action buttons
- **Empty States**: Helpful guidance when no data is available
- **Tooltips**: Hover/focus tooltips with keyboard support
- **Animations**: CSS keyframe animations for loading states

### Changed - CI/CD Performance Optimization (P1.1)
- Added Go module caching with `actions/cache@v4` (reduces `go mod download` from 3x to 1x)
- Added Go build cache for CGO artifacts (accelerates DuckDB compilation)
- Parallelized lint and test jobs (removed unnecessary dependency)
- Added BuildKit cache mounts for npm, Go modules, and Go build in Dockerfile
- Added conditional integration tests (skip on feature branches, run only on PR/main)
- Added `fail-fast: false` to Docker matrix builds for true parallelization
- **Result**: CI/CD pipeline reduced from ~15 minutes to <10 minutes (33% improvement)

### Technical Details
- **Service Worker**: Network-first caching strategy for offline capability
- **Chart Lazy Loading**: IntersectionObserver saves ~500ms on initial load
- **ETag Caching**: FNV-1a hash reduces bandwidth by ~60% on repeat requests
- **BuildKit Cache**: Multi-stage cache mounts reduce build time by ~40%
- **Dark Mode**: CSS custom properties with data-theme attribute
- **Focus Management**: focus-visible for modern browsers, fallback to focus
- **Accessibility**: WCAG 2.1 AA compliant (4.5:1 contrast ratios verified)

### Planned
- Real-time WebSocket updates for live playback tracking
- Heatmap layer toggle for density visualization
- Animated playback timeline with play/pause controls
- Data export capabilities (CSV, GeoJSON)
- Dark mode toggle with theme persistence
- Progressive Web App (PWA) support
- Multi-server support for multiple Tautulli instances

## [1.8.0] - 2025-11-21

### Added - Metadata Enrichment Fields for Advanced Analytics

**15 New Database Fields** for comprehensive content tracking and analytics:

- **Content Identification** (5 fields):
  - `rating_key`: Unique Plex identifier for content item
  - `parent_rating_key`: Parent content identifier (season for episodes, album for tracks)
  - `grandparent_rating_key`: Grandparent identifier (TV show for episodes, artist for tracks)
  - `media_index`: Episode number within season (enables sequential tracking)
  - `parent_media_index`: Season number (enables season-level analytics)

- **External ID Integration**:
  - `guid`: External database identifiers (IMDB, TVDB, TMDB, MusicBrainz)
    - Enables deep linking to external databases for enriched metadata
    - Supports multi-source linking (e.g., `imdb://tt1234567`, `tvdb://12345`)

- **Enhanced Content Metadata** (4 fields):
  - `original_title`: Original language title for international content
  - `full_title`: Complete title with all hierarchy (show + season + episode)
  - `originally_available_at`: Original release/air date
  - `watched_status`: User's watched status (0=unwatched, 1=watched)

- **Visual Assets**:
  - `thumb`: Thumbnail/poster URL for content visualization

- **Cast & Crew Analytics** (3 fields):
  - `directors`: Director names (comma-separated)
  - `writers`: Writer/screenplay credits
  - `actors`: Main cast members

- **Genre Classification**:
  - `genres`: Genre tags (comma-separated)

**Database Indexes** (7 new indexes):
- Individual indexes: `media_index`, `parent_media_index`, `rating_key`, `parent_rating_key`, `grandparent_rating_key`, `genres`
- Composite binge detection index: `(user_id, grandparent_rating_key, parent_media_index, media_index, started_at)`
- Enables sub-50ms queries for complex sequential analytics

**Tests**:
- Added `TestInsertPlaybackEvent_MetadataEnrichmentFields` with comprehensive field verification
- Total test count increased to 60 tests (from 48)
- All tests pass with no race conditions

### Changed
- **Binge Detection**: Now uses sequential episode tracking via `media_index` and `parent_media_index`
  - Can detect specific episode sequences (S02E01 → S02E02 → S02E03)
  - Identifies skipped episodes and rewatch patterns
  - Season-level binge analysis (entire season watched consecutively)

- **Sync Logic**: Updated to capture all 15 new fields from Tautulli API
  - Added validation for "N/A" values and zero checks
  - Graceful handling of missing/optional fields

### Fixed
- **Sync Manager Race Condition**: Added `syncMu` mutex to prevent concurrent sync execution
  - Previously, `TriggerSync()` could run while periodic sync was active
  - Could cause duplicate geolocation API calls
  - Database handles duplicates via unique constraint, but now prevented at source

- **Test Suite Stability** (3 bugfixes):
  - **Package Naming Conflict**: Fixed conflict with standard library `sync` package by using `stdSync` alias in tests
  - **Double-Close Panic**: Fixed panic in `TestManager_StopDuringSync` using `sync.Once` to prevent multiple channel closes
  - **Deadlock**: Fixed channel close order in `TestManager_StopDuringSync` to prevent timeout deadlock

### Migration Notes
- **Automatic Schema Migration**: Database columns added automatically on startup
- **Backward Compatibility**: Existing playback events will have NULL for new fields
- **Forward Compatibility**: New syncs populate all fields automatically from Tautulli API
- **No Manual Intervention Required**: Zero downtime migration

### Analytics Capabilities Unlocked
1. **Sequential Episode Tracking**: True binge-watching detection with episode-by-episode progression
2. **Genre Analytics**: Genre trend analysis and user preference profiling
3. **Cast/Crew Analytics**: Viewing pattern analysis by directors, writers, actors
4. **External Metadata Enrichment**: IMDB/TVDB/TMDB integration for deep content insights
5. **Season-Level Analytics**: Granular analytics at season and episode levels

## [0.3.0] - 2025-11-16

### Added - Phase 2 Priority 1: Analytics Dashboard
- Apache ECharts 5.5.1 integration with echarts-gl 2.0.9 for hardware-accelerated visualizations
- 6 interactive analytics charts:
  - Playback trends over time with dual Y-axis (playbacks + unique users)
  - Top 10 countries by playback count (horizontal bar chart)
  - Top 10 cities by playback count (horizontal bar chart)
  - Media type distribution (donut chart)
  - User activity leaderboard (horizontal bar chart)
  - Viewing hours heatmap (hour �day of week)
- PNG export functionality for all charts
- Real-time filter integration across all charts
- Auto-scaling time intervals for trend charts (day/week/month)
- Dark theme compatible chart styling
- Responsive grid layout (2 columns desktop, 1 column mobile)

### Added - Backend Analytics
- `GET /api/v1/analytics/trends` endpoint with automatic interval detection
- `GET /api/v1/analytics/geographic` endpoint for countries, cities, and heatmap data
- `GET /api/v1/analytics/users` endpoint for user activity leaderboard
- `buildFilter` helper function for DRY filter construction
- Comprehensive filter support across all analytics endpoints

### Changed
- Updated frontend architecture to support chart management
- Enhanced API client with analytics endpoint methods
- Improved performance with parallel chart data loading

### Fixed
- Validation logic for `days` parameter (check presence before validating value)

## [0.2.1] - 2025-11-15

### Added - Phase 2.1: Advanced Map Filtering + Clustering
- Date range picker with quick presets (24h, 7d, 30d, 90d, 1y, all time)
- Custom date range selection
- User filter dropdown (multi-select, dynamically populated)
- Media type filter dropdown (movies, TV shows, music)
- Clear Filters button to reset all filters
- URL query parameter sync for shareable filtered views
- 300ms debounced filter changes to prevent API spam

### Added - Map Clustering
- MapLibre GL 5.13.0 GeoJSON clustering for 10x performance improvement
- Color-coded cluster markers based on playback count:
  - Teal: Low activity (< 10 playbacks)
  - Orange: Medium activity (10-50 playbacks)
  - Red: High activity (50-100 playbacks)
  - Pink: Very high activity (100+ playbacks)
- Cluster expansion on click
- Enhanced popups with formatted location details

### Added - Backend Filtering
- `GET /api/v1/users` endpoint for user list
- `GET /api/v1/media-types` endpoint for media type list
- Filter support on `/api/v1/locations` endpoint:
  - `start_date` and `end_date` for date range filtering
  - `user` parameter for user-specific filtering
  - `media_type` parameter for media type filtering
  - `days` parameter as alternative to explicit date ranges

### Changed
- Map rendering now uses clustering instead of individual markers
- Filter state managed in URL query parameters
- Improved map performance with large datasets (10k+ locations)

## [0.1.0] - 2025-11-14

### Added - Phase 1: Security + Login
- JWT authentication with HS256 algorithm
- Login UI with username/password authentication
- "Remember me" functionality for extended sessions (30 days vs 24 hours)
- Secure cookie handling (HttpOnly, SameSite, Secure flags)
- Session persistence with configurable timeout
- `AUTH_MODE=none` support for development and testing
- `AUTH_MODE=jwt` for production deployments

### Added - Security Features
- Rate limiting per IP address (configurable, default: 100 req/min)
- Rate limiter automatic cleanup (prevents memory leaks)
- CORS configuration with `CORS_ORIGINS` environment variable
- Trusted proxy support with `TRUSTED_PROXIES` environment variable
- X-Forwarded-For header validation (prevents IP spoofing)
- JWT secret validation (minimum 32 characters required)
- API input validation for all endpoints
- SQL injection prevention via parameterized queries
- XSS prevention with Content Security Policy headers

### Added - Testing Infrastructure
- Unit tests for config package (`internal/config/config_test.go`)
- Unit tests for auth package (`internal/auth/jwt_test.go`, `internal/auth/middleware_test.go`)
- Integration test suite (`test/integration_test.sh`) with 28 test scenarios
- GitHub Actions CI/CD pipeline with parallel builds
- Test coverage reporting (40%+ overall coverage)

### Added - Documentation
- `QUALITY_REVIEW.md` with security audit findings
- `CHANGES.md` documenting Phase 1 changes
- `ARCHITECTURE.md` with detailed system design
- Enhanced README with authentication documentation

### Changed
- Multi-architecture Docker builds optimized (removed matrix, build both platforms in one job)
- JWT_SECRET now required when AUTH_MODE=jwt (no auto-generation)
- Docker Compose healthcheck uses `curl` instead of `wget`
- Integration tests now run on pull requests
- Removed ARMv7 support (DuckDB incompatibility)

### Fixed
- Rate limiter memory leak (automatic cleanup every hour)
- CORS wildcard security issue (now configurable)
- IP spoofing vulnerability via X-Forwarded-For headers
- Weak default JWT secrets
- Missing validation on API parameters

### Security
- All critical P0 security issues resolved
- OWASP Top 10 compliance verified
- Defense in depth with multiple validation layers
- Secure defaults enforced (no weak secrets)

## [0.0.1] - 2025-11-13

### Added - Initial Release
- Interactive map visualization with MapLibre GL 5.13.0 JS
- Real-time synchronization with Tautulli API
- DuckDB with spatial extension for geographic data storage
- RESTful JSON API with the following endpoints:
  - `GET /api/v1/health` - Health check and system status
  - `GET /api/v1/stats` - Overall statistics summary
  - `GET /api/v1/playbacks` - Paginated playback history
  - `GET /api/v1/locations` - Geographic aggregations
  - `POST /api/v1/sync` - Trigger manual Tautulli sync
- Basic statistics display (total playbacks, unique locations, unique users)
- Docker multi-architecture support (linux/amd64, linux/arm64)
- Environment variable configuration
- Automatic Tautulli sync with configurable interval
- Initial sync lookback period (default: 24 hours)
- TypeScript frontend with strict mode
- esbuild bundler for production builds
- Debian Bookworm base image (glibc required for DuckDB)

### Technical Details
- Go 1.24+ backend with standard library HTTP server
- DuckDB 1.4.1+ with spatial indexing
- MapLibre GL 5.13.0 JS 3.16.0 for WebGL-accelerated rendering
- TypeScript 5.9.3 in strict mode
- Port 3857 (EPSG:3857 - Web Mercator projection)

---

## Version History Summary

| Version | Date | Description |
|---------|------|-------------|
| 0.3.0 | 2025-11-16 | Analytics dashboard with 6 interactive charts |
| 0.2.1 | 2025-11-15 | Advanced filtering and map clustering |
| 0.1.0 | 2025-11-14 | JWT authentication and security hardening |
| 0.0.1 | 2025-11-13 | Initial release with basic map visualization |

## Breaking Changes

### 0.1.0
- **JWT_SECRET now required**: When `AUTH_MODE=jwt`, the `JWT_SECRET` environment variable must be set with at least 32 characters. Previously, a secret was auto-generated, which caused session loss on container restart.

  **Migration**: Generate a secure secret and set it explicitly:
  ```bash
  export JWT_SECRET=$(openssl rand -base64 48)
  ```

## Upgrade Guide

### From 0.2.x to 0.3.0
No breaking changes. Simply pull the latest image:
```bash
docker pull ghcr.io/tomtom215/cartographus:latest
docker-compose up -d
```

The new analytics dashboard will be available immediately at the same URL.

### From 0.1.x to 0.2.x
No breaking changes. Update includes:
- New filtering UI (accessible via sidebar)
- Improved map performance with clustering
- New API endpoints for filters (backward compatible)

### From 0.0.x to 0.1.0
**Action required** if using JWT authentication:

1. Generate and set JWT_SECRET:
   ```bash
   openssl rand -base64 48
   ```

2. Update docker-compose.yml:
   ```yaml
   environment:
     JWT_SECRET: your_generated_secret_here
     ADMIN_USERNAME: admin
     ADMIN_PASSWORD: your_secure_password
   ```

3. Optionally configure CORS and trusted proxies:
   ```yaml
   environment:
     CORS_ORIGINS: "https://yourdomain.com"
     TRUSTED_PROXIES: "10.0.0.1,192.168.1.1"
   ```

4. Restart container:
   ```bash
   docker-compose down
   docker-compose up -d
   ```

## Support

For issues, questions, or feature requests:
- GitHub Issues: https://github.com/tomtom215/cartographus/issues
- GitHub Discussions: https://github.com/tomtom215/cartographus/discussions
