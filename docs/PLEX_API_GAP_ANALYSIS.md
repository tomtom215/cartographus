# Plex API Gap Analysis for Complete Data Science Coverage

**Document Version**: 1.0
**Date**: 2025-11-30
**Purpose**: Comprehensive comparison of Cartographus implementation against the complete Plex Media Server API to achieve 99%+ coverage for data science and analytics applications.

---

## Executive Summary

This document provides a thorough gap analysis comparing our current implementation against the complete Plex Media Server API reference (150+ endpoints, 500+ data fields). The goal is to ensure **zero data blind spots** for any possible correlation analysis, cohort study, or machine learning application.

### Current State Assessment

| Category | Current Coverage | Target | Gap |
|----------|-----------------|--------|-----|
| **Direct Plex API Endpoints** | 11 of 150+ | 100% critical | ~139 endpoints |
| **Tautulli API Endpoints** | 54 of ~60 | 100% | ~6 endpoints |
| **Playback Data Fields** | 137 of ~200 | 100% | ~63 fields |
| **Library Metadata Fields** | ~40 of ~100 | 100% | ~60 fields |
| **Server/System Data** | ~20 of ~50 | 100% | ~30 fields |

### Priority Classification

- **P0 (Critical)**: Data essential for core analytics - must have for any meaningful analysis
- **P1 (High)**: Data important for advanced analytics and correlations
- **P2 (Medium)**: Data useful for specialized analysis and edge cases
- **P3 (Low)**: Nice-to-have data for completeness

---

## Part 1: Library and Media API Gaps

### 1.1 Core Library Endpoints

#### Currently Implemented
| Endpoint | Method | Status |
|----------|--------|--------|
| `/library/sections` | GET | Implemented |
| `/library/sections/{id}/all` | GET | Implemented |
| `/library/sections/{id}/recentlyAdded` | GET | Implemented |

#### Missing Endpoints (P0 - Critical for Data Science)

| Endpoint | Method | Priority | Data Science Use Case |
|----------|--------|----------|----------------------|
| `/library/sections/{id}/refresh` | GET | P2 | Track library scan frequency, content discovery patterns |
| `/library/sections/{id}/analyze` | GET | P2 | Media analysis completion rates, quality assessment |
| `/library/sections/{id}/emptyTrash` | POST | P3 | Library maintenance patterns |
| `/library/metadata/{ratingKey}` | GET | **P0** | **Complete metadata for any item - essential for content analysis** |
| `/library/metadata/{ratingKey}/children` | GET | **P0** | **Hierarchical content navigation (seasons/episodes)** |
| `/library/metadata/{ratingKey}/similar` | GET | P1 | Recommendation engine training data |
| `/library/metadata/{ratingKey}/related` | GET | P1 | Content relationship analysis |
| `/library/metadata/{ratingKey}/posters` | GET | P2 | Artwork availability analysis |
| `/library/metadata/{ratingKey}/matches` | GET | P2 | Metadata matching accuracy analysis |
| `/library/metadata/{ratingKey}/match` | PUT | P3 | Manual match corrections tracking |
| `/library/metadata/{ratingKey}/unmatch` | DELETE | P3 | Unmatch frequency analysis |
| `/library/metadata/{ratingKey}/split` | POST | P3 | Content merging/splitting patterns |
| `/library/metadata/{ratingKey}/merge` | PUT | P3 | Content organization analysis |

### 1.2 Missing Metadata Fields (Critical for ML/Analytics)

#### Movie Metadata - Missing Fields

| Field | Type | Priority | Data Science Use Case |
|-------|------|----------|----------------------|
| `slug` | string | P2 | URL-safe identifiers for external integrations |
| `viewCount` | int | **P0** | **Popularity metrics, watch frequency analysis** |
| `skipCount` | int | **P0** | **Content quality signal - high skip = potential issues** |
| `lastViewedAt` | int | P1 | Recency analysis, re-watch patterns |
| `viewOffset` | int | P1 | Resume position analytics, completion prediction |
| `theme` | string | P2 | Theme music availability |
| `primaryExtraKey` | string | P2 | Trailer engagement analysis |
| `hasPremiumExtras` | int | P2 | Premium content availability |
| `hasPremiumPrimaryExtra` | int | P2 | Premium extras engagement |
| `chapterSource` | string | P1 | Chapter navigation usage |
| `librarySectionUUID` | string | P2 | Cross-library content tracking |

#### Nested Tag Arrays - Currently Stored as Comma-Separated Strings

**Current Implementation**: We store `genres`, `directors`, `writers`, `actors` as comma-separated strings.

**Missing Structure** (each tag object should include):

```json
{
  "id": 123,
  "filter": "genre=123",
  "tag": "Action",
  "tagKey": "action",
  "thumb": "/tags/genre/action.png",
  "role": "Director"  // For Role[] only
}
```

| Missing Field | Priority | Data Science Use Case |
|---------------|----------|----------------------|
| Tag `id` | **P0** | **Unique identifier for filtering and joining** |
| Tag `filter` | P1 | Programmatic filtering capability |
| Tag `tagKey` | P1 | URL-safe tag references |
| Tag `thumb` | P2 | Tag artwork availability |
| Role `role` | **P0** | **Actor character names for cast analysis** |

#### External Identifiers - Currently Single GUID String

**Current Implementation**: Single `guid` string (e.g., "plex://movie/5d776...")

**Missing**: Separate `Guid[]` array with individual providers:

```json
{
  "Guid": [
    {"id": "imdb://tt0816692"},
    {"id": "tmdb://157336"},
    {"id": "tvdb://121361"}
  ]
}
```

| Missing | Priority | Data Science Use Case |
|---------|----------|----------------------|
| Individual provider GUIDs | **P0** | **Cross-platform content matching, external data enrichment** |
| Provider-specific metadata | P1 | Rating comparison across IMDb/TMDB/TVDB |

#### Rating Sources - Missing Separate Rating[] Array

**Current Implementation**: Single `rating`, `audienceRating`, `userRating` fields

**Missing Structure**:

```json
{
  "Rating": [
    {"type": "critic", "value": 8.5, "image": "rottentomatoes://image.rating.ripe"},
    {"type": "audience", "value": 9.1, "image": "rottentomatoes://image.rating.upright"}
  ]
}
```

| Missing | Priority | Data Science Use Case |
|---------|----------|----------------------|
| Rating source type | P1 | Rating source comparison analysis |
| Rating image | P2 | Rating badge tracking |
| Multiple rating sources | P1 | Cross-source rating correlation |

### 1.3 TV Show Hierarchy - Missing Fields

#### Show-Level Missing Fields

| Field | Type | Priority | Data Science Use Case |
|-------|------|----------|----------------------|
| `showOrdering` | string | P1 | aired/dvd/absolute ordering preference analysis |
| `autoDeletionItemPolicyUnwatchedLibrary` | int | P2 | Auto-deletion policy impact on viewing |
| `autoDeletionItemPolicyWatchedLibrary` | int | P2 | Watched content retention patterns |

#### Episode-Level Missing Fields

| Field | Type | Priority | Data Science Use Case |
|-------|------|----------|----------------------|
| `hasIntroMarker` | bool | **P0** | **Intro skip usage analysis** |
| `hasCreditsMarker` | bool | **P0** | **Credits skip analysis, binge detection** |
| `hasCommercialMarker` | bool | P1 | Commercial detection accuracy |

### 1.4 Music Library - Missing Fields

#### Artist Fields
| Field | Priority | Data Science Use Case |
|-------|----------|----------------------|
| `albumSort` | P2 | Sorting preference analysis |
| `countries[]` | P1 | Geographic music preference analysis |
| `moods[]` | **P0** | **Mood-based listening pattern analysis** |
| `styles[]` | P1 | Genre sub-style analysis |
| `similar[]` | P1 | Artist similarity mapping |

#### Album Fields
| Field | Priority | Data Science Use Case |
|-------|----------|----------------------|
| `formats[]` | P2 | Audio format preference analysis |
| `subformats[]` | P2 | Sub-format (vinyl, CD, digital) analysis |
| `loudnessAnalysisVersion` | P2 | Audio normalization tracking |
| `musicAnalysisVersion` | P2 | Analysis algorithm versioning |

#### Track Fields
| Field | Priority | Data Science Use Case |
|-------|----------|----------------------|
| `ratingCount` | P1 | Track popularity by rating count |
| `skipCount` | **P0** | **Track skip patterns - quality signal** |

### 1.5 Media Analysis Data - Missing Nested Structures

#### Multiple Media[] Versions

**Current Implementation**: Single media object

**Missing**: Full `Media[]` array for multiple versions:

```json
{
  "Media": [
    {"id": 1, "videoResolution": "4k", "videoCodec": "hevc"},
    {"id": 2, "videoResolution": "1080", "videoCodec": "h264"}
  ]
}
```

| Missing | Priority | Data Science Use Case |
|---------|----------|----------------------|
| Multiple media versions | **P0** | **Version selection analysis, quality preference** |
| Version comparison | P1 | Optimal encoding analysis |

#### Part[] Details

| Field | Priority | Data Science Use Case |
|-------|----------|----------------------|
| `indexes` | P1 | Index file availability |
| `deepAnalysisVersion` | P2 | Analysis algorithm tracking |

#### Video Stream Fields - Missing Dolby Vision Specifics

| Field | Priority | Data Science Use Case |
|-------|----------|----------------------|
| `DOVIPresent` | **P0** | **Dolby Vision content tracking** |
| `DOVIProfile` | **P0** | **DV profile analysis (5, 7, 8)** |
| `DOVIVersion` | P1 | DV version compatibility |
| `DOVIBLPresent` | P1 | Base layer presence |
| `DOVIELPresent` | P1 | Enhancement layer presence |
| `DOVILevel` | P1 | DV level tracking |
| `DOVIRPUPresent` | P1 | RPU data presence |

### 1.6 Collections and Playlists - Missing Fields

| Field | Priority | Data Science Use Case |
|-------|----------|----------------------|
| `collectionMode` | P1 | Collection display mode analysis |
| `collectionSort` | P1 | Sorting preference analysis |
| `content` (smart filter) | **P0** | **Smart collection criteria analysis** |
| `minYear`, `maxYear` | P1 | Year range filtering patterns |
| `playlistType` | P1 | Playlist type distribution |
| `radio` | P2 | Radio playlist usage |

---

## Part 2: Server and System API Gaps

### 2.1 Server Root Endpoint (GET /)

#### Currently Captured (via Tautulli)
- `machineIdentifier`, `version`, `platform`, `myPlexSubscription`

#### Missing Fields (P1 - Important for System Analytics)

| Field | Priority | Data Science Use Case |
|-------|----------|----------------------|
| `platformVersion` | P1 | OS version correlation with issues |
| `myPlex` | P1 | Plex.tv linkage status |
| `myPlexUsername` | P1 | Account identification |
| `allowSync` | P1 | Sync permission analysis |
| `allowSharing` | P1 | Sharing feature usage |
| `allowMediaDeletion` | P2 | Deletion permission patterns |
| `transcoderActiveVideoSessions` | **P0** | **Real-time transcode load** |
| `transcoderVideo` | P1 | Transcoder capability |
| `transcoderAudio` | P1 | Audio transcoding support |
| `transcoderSubtitles` | P1 | Subtitle transcoding support |
| `transcoderVideoBitrates` | P1 | Supported bitrate range |
| `transcoderVideoQualities` | P1 | Quality preset analysis |
| `transcoderVideoResolutions` | P1 | Resolution capability |
| `streamingBrainVersion` | P1 | Streaming algorithm version |
| `streamingBrainABRVersion` | P1 | ABR algorithm tracking |
| `ownerFeatures[]` | **P0** | **Plex Pass feature availability** |
| `updatedAt` | P1 | Server update tracking |

### 2.2 Server Preferences (GET /:/prefs) - NOT IMPLEMENTED

**Priority**: P1 - 168+ configurable settings for system behavior analysis

**Missing Endpoint**: `GET /:/prefs`

| Setting Category | Example Keys | Data Science Use Case |
|------------------|--------------|----------------------|
| Transcoder | `TranscoderQuality`, `HardwareAcceleratedCodecs`, `TranscoderTempDirectory`, `TranscoderThrottleBuffer` | **Transcode performance optimization** |
| Network | `customConnections`, `EnableIPv6`, `secureConnections`, `WanPerStreamMaxUploadRate`, `RelayEnabled` | **Network configuration impact analysis** |
| Library | `FSEventLibraryUpdatesEnabled`, `autoEmptyTrash`, `OnDeckWindow` | Library management patterns |
| DLNA | `DlnaEnabled`, `DlnaPlatinumLoggingLevel` | DLNA usage analysis |
| General | `FriendlyName`, `sendCrashReports`, `logDebug` | Server configuration baseline |

### 2.3 Butler Scheduled Tasks (GET /butler) - NOT IMPLEMENTED

**Priority**: P1 - Server maintenance patterns

**Missing Endpoint**: `GET /butler`

| Task | Data Science Use Case |
|------|----------------------|
| `BackupDatabase` | Backup frequency and timing |
| `OptimizeDatabase` | DB optimization patterns |
| `CleanOldBundles` | Storage management |
| `CleanOldCacheFiles` | Cache cleanup patterns |
| `RefreshLibraries` | Scan scheduling analysis |
| `RefreshLocalMedia` | Local agent refresh patterns |
| `RefreshPeriodicMetadata` | Metadata update frequency |
| `DeepMediaAnalysis` | Analysis completion rates |
| `GenerateIntroMarkers` | Intro detection coverage |
| `GenerateCreditsMarkers` | Credits detection coverage |
| `LoudnessAnalysis` | Audio normalization coverage |

**Missing Endpoints**:
- `POST /butler/{taskName}` - Trigger task
- `DELETE /butler/{taskName}` - Cancel task

### 2.4 Resource Monitoring (GET /statistics/resources) - NOT IMPLEMENTED

**Priority**: P0 - Critical for system health analytics

**Missing Endpoint**: `GET /statistics/resources`

| Field | Data Science Use Case |
|-------|----------------------|
| `hostCpuUtilization` | **System load correlation with playback issues** |
| `processCpuUtilization` | **Plex-specific CPU usage** |
| `hostMemoryUtilization` | **Memory pressure analysis** |
| `processMemoryUtilization` | **Plex memory footprint** |

---

## Part 3: Session and Activity API Gaps

### 3.1 Active Sessions - Additional Missing Fields

#### Currently Implemented
- Most session fields via Tautulli's `get_activity`

#### Missing from Native Plex API (GET /status/sessions)

| Field | Priority | Data Science Use Case |
|-------|----------|----------------------|
| `session.id` | P1 | Native session identification |
| `session.bandwidth` | **P0** | **Real-time bandwidth per session** |
| `session.location` | P1 | lan/wan classification |
| `player.remotePublicAddress` | P1 | Public IP for remote sessions |

### 3.2 Timeline Reporting - NOT IMPLEMENTED

**Priority**: P0 - Critical for accurate playback position tracking

**Missing Endpoint**: `PUT /:/timeline`

| Parameter | Data Science Use Case |
|-----------|----------------------|
| `ratingKey` | Item being played |
| `key` | API path |
| `state` | playing/paused/stopped transitions |
| `time` | Exact playback position in ms |
| `duration` | Content duration |

**Impact**: Without timeline reporting, we rely on Tautulli's polling which may miss short sessions or precise position data.

### 3.3 Scrobble/Rating Endpoints - NOT IMPLEMENTED

**Priority**: P0 - User preference data

| Endpoint | Method | Data Science Use Case |
|----------|--------|----------------------|
| `/:/scrobble` | GET | **Track watch completion events** |
| `/:/unscrobble` | GET | **Track unwatched marking** |
| `/:/rate?key={key}&rating={0-10}` | PUT | **User rating collection** |

### 3.4 Play Queues - NOT IMPLEMENTED

**Priority**: P1 - Queue behavior analysis

| Endpoint | Method | Data Science Use Case |
|----------|--------|----------------------|
| `POST /playQueues` | POST | Queue creation patterns |
| `GET /playQueues/{id}` | GET | Queue composition analysis |
| `DELETE /playQueues/{id}` | DELETE | Queue abandonment patterns |

**Missing Fields**:
- `playQueueID`
- `playQueueSelectedItemID`
- `playQueueTotalCount`
- `playQueueVersion`
- `playQueueShuffled`

### 3.5 Transcode Control - NOT IMPLEMENTED

**Priority**: P1 - Transcode session management

| Endpoint | Data Science Use Case |
|----------|----------------------|
| `/video/:/transcode/universal/decision` | Transcode decision analysis |
| `/video/:/transcode/universal/start.m3u8` | Stream initiation tracking |
| `/video/:/transcode/universal/ping` | Session keep-alive patterns |
| `/video/:/transcode/universal/stop` | Transcode termination tracking |

---

## Part 4: Search and Discovery API Gaps

### 4.1 Hub Search - NOT IMPLEMENTED

**Priority**: P0 - Critical for content discovery analytics

**Missing Endpoint**: `GET /hubs/search?query={term}&limit={n}`

| Field | Data Science Use Case |
|-------|----------------------|
| `hubIdentifier` | Search result type distribution |
| `hubKey` | Direct result access |
| `size`, `more` | Pagination analysis |
| `score` | Relevance scoring analysis |
| `reason` | Search relevance factors |

### 4.2 Recommendation Hubs - NOT IMPLEMENTED

**Priority**: P1 - Recommendation engine analysis

| Endpoint | Data Science Use Case |
|----------|----------------------|
| `GET /hubs` | All recommendation types |
| `GET /hubs/sections/{sectionID}` | Section-specific recommendations |

**Hub Identifiers**:
- `home.playlists`, `home.onDeck`, `home.continue`
- `home.movies.recentlyAdded`, `home.television.recentlyAdded`
- `hub.tv.ondeck`

### 4.3 Browse/Filter Endpoints - NOT IMPLEMENTED

**Priority**: P1 - Content navigation patterns

| Endpoint | Data Science Use Case |
|----------|----------------------|
| `/library/sections/{id}/firstCharacter` | A-Z navigation usage |
| `/library/sections/{id}/genre` | Genre browsing patterns |
| `/library/sections/{id}/year` | Year filtering usage |
| `/library/sections/{id}/decade` | Decade preference analysis |
| `/library/sections/{id}/director` | Director browsing patterns |
| `/library/sections/{id}/actor` | Actor browsing patterns |
| `/library/sections/{id}/contentRating` | Rating filter usage |
| `/library/sections/{id}/resolution` | Quality filtering patterns |
| `/library/sections/{id}/studio` | Studio preference analysis |

**Filter Metadata Endpoints**:
- `GET /library/sections/{id}/filters` - Available filter fields
- `GET /library/sections/{id}/sorts` - Available sort options

---

## Part 5: User and Account API Gaps (plex.tv)

### 5.1 Authentication - Partially Implemented

#### Currently Implemented
- OAuth 2.0 PKCE flow

#### Missing
| Endpoint | Priority | Data Science Use Case |
|----------|----------|----------------------|
| PIN authentication flow | P2 | Device authentication patterns |
| `POST /api/v2/users/signin` | P2 | Direct sign-in tracking |

### 5.2 Account Information (plex.tv) - NOT IMPLEMENTED

**Priority**: P0 - Critical for user segmentation

**Missing Endpoint**: `GET https://plex.tv/api/v2/user`

| Field | Data Science Use Case |
|-------|----------------------|
| `id`, `uuid` | User identification |
| `username`, `email` | Account correlation |
| `subscriptionActive` | **Plex Pass correlation with behavior** |
| `subscriptionPlan` | **Plan tier analysis** |
| `subscriptionFeatures[]` | **Feature availability correlation** |
| `home`, `homeAdmin` | Home membership analysis |
| `homeSize`, `maxHomeSize` | Home size limits |
| `restricted` | Content restriction analysis |
| `profile` (audio/subtitle prefs) | Default preference analysis |

### 5.3 Managed Users (Plex Home) - NOT IMPLEMENTED

**Priority**: P1 - Multi-user household analysis

| Endpoint | Data Science Use Case |
|----------|----------------------|
| `GET /api/v2/home/users` | Home member enumeration |
| `POST /api/v2/home/users/restricted` | Managed user creation patterns |

**Missing Data**:
- `restrictionProfile` (little_kid, older_kid, teen)
- `friendlyName`
- Sharing settings

### 5.4 Friends and Sharing - NOT IMPLEMENTED

**Priority**: P1 - Social sharing analysis

| Endpoint | Data Science Use Case |
|----------|----------------------|
| `GET /api/v2/friends` | Friend network analysis |
| `POST /api/servers/{id}/shared_servers` | Sharing invitation patterns |

**Sharing Parameters**:
- `allowSync`, `allowCameraUpload`, `allowChannels`
- `filterMovies`, `filterTelevision` (library restrictions)

### 5.5 Sync/Download APIs - NOT IMPLEMENTED

**Priority**: P1 - Offline viewing patterns

| Endpoint | Data Science Use Case |
|----------|----------------------|
| `GET /sync/items` | Sync queue analysis |
| `POST /sync/items` | Sync initiation patterns |

**Sync Policy Data**:
- `scope` (user, device, library)
- `unwatched` (sync unwatched only)
- `value` (sync limit)
- `maxVideoBitrate`, `videoResolution` (quality settings)

---

## Part 6: Tautulli API Gaps

### 6.1 Currently Implemented Tautulli Endpoints (54)

See `internal/sync/tautulli_client.go` - full implementation

### 6.2 Missing Tautulli Endpoints

| Endpoint | Priority | Data Science Use Case |
|----------|----------|----------------------|
| `get_notification_log` | P2 | Notification delivery analysis |
| `get_notifier_config` | P3 | Notification setup patterns |
| `set_notifier_config` | P3 | Configuration change tracking |
| `get_plex_log` | P1 | **Error and warning analysis** |
| `backup_config` | P3 | Backup scheduling |
| `backup_db` | P3 | Database backup patterns |

### 6.3 Missing Tautulli Data Fields

**From `get_activity` - Additional Fields**:

| Field | Priority | Data Science Use Case |
|-------|----------|----------------------|
| `selected` | P2 | Selected item tracking |
| `indexes` | P1 | Index file usage |
| `bif_thumb` | P2 | BIF thumbnail availability |
| `synced_version_profile` | P1 | Sync version selection |
| `optimized_version_profile` | P1 | Optimized version usage |

---

## Part 7: Undocumented/Community API Gaps

### 7.1 Marker Endpoints - PARTIALLY IMPLEMENTED

**Currently**: We capture marker presence via Tautulli

**Missing Direct Access**:

| Data | Priority | Data Science Use Case |
|------|----------|----------------------|
| `markers[]` array with `startTimeOffset`, `endTimeOffset` | **P0** | **Intro/credits skip behavior analysis** |
| Marker types: `intro`, `credits`, `commercial`, `chapter` | **P0** | **Skip pattern by marker type** |

**Recommended**: Add `?includeMarkers=1` to metadata requests

### 7.2 DVR/Live TV - PARTIALLY IMPLEMENTED

**Currently**: Basic Live TV fields via Tautulli

**Missing Endpoints**:

| Endpoint | Priority | Data Science Use Case |
|----------|----------|----------------------|
| `GET /livetv/sessions` | P1 | Live TV session tracking |
| `GET /livetv/dvrs` | P1 | DVR device inventory |
| `GET /livetv/dvrs/{id}/channels` | P1 | Channel lineup analysis |
| `GET /media/subscriptions` | P1 | Recording subscription patterns |

### 7.3 Photo Transcode - NOT IMPLEMENTED

**Priority**: P3

| Endpoint | Data Science Use Case |
|----------|----------------------|
| `GET /photo/:/transcode` | Photo resize patterns |

---

## Part 8: Data Field Coverage Summary

### 8.1 Fields We Capture (137+)

See `internal/models/playback.go` - comprehensive list

### 8.2 Critical Missing Fields for Data Science

| Category | Missing Fields | Priority | Impact |
|----------|---------------|----------|--------|
| **Content Quality Signals** | `skipCount`, `viewCount`, `hasIntroMarker`, `hasCreditsMarker` | **P0** | Cannot detect content quality issues or skip patterns |
| **User Preferences** | Full rating sources, subscription features, profile preferences | **P0** | Limited user segmentation capability |
| **Content Relationships** | Similar content, related hubs, smart collection criteria | **P0** | Cannot build recommendation analysis |
| **External Identifiers** | Individual IMDb/TMDB/TVDB GUIDs | **P0** | Cannot enrich with external data |
| **System Health** | CPU/memory utilization, preferences | **P1** | Cannot correlate system load with playback issues |
| **Dolby Vision** | DOVIProfile, DOVIVersion, DOVIBLPresent | **P0** | Incomplete HDR analysis |
| **Tag Structure** | Tag IDs, filter keys, role names | **P0** | Cannot do proper cast/crew analysis |
| **Multiple Versions** | Full Media[] array with all versions | **P0** | Cannot analyze version selection patterns |

---

## Part 9: Implementation Roadmap

### Phase 1: Critical Data Gaps (P0) - Estimated 40 endpoints/fields

**Goal**: Capture all data essential for basic analytics

1. **Direct Metadata Lookup** (`/library/metadata/{id}`)
   - Full metadata with all nested structures
   - Children endpoint for hierarchy
   - Include markers

2. **External ID Parsing**
   - Parse GUID string into individual providers
   - Store IMDb, TMDB, TVDB separately

3. **Skip/View Counts**
   - Add `skipCount`, `viewCount` to models
   - Capture via metadata endpoint

4. **System Resources**
   - Implement `/statistics/resources`
   - Track CPU/memory utilization

5. **Tag Structure Enhancement**
   - Store tag arrays as proper JSON
   - Include tag IDs, filter keys, role names

6. **Dolby Vision Fields**
   - Add all DOVI fields to stream models
   - Capture via enhanced session data

### Phase 2: Advanced Analytics Data (P1) - Estimated 60 endpoints/fields

**Goal**: Enable sophisticated correlation analysis

1. **Search and Discovery**
   - Hub search endpoint
   - Recommendation hubs
   - Browse/filter endpoints

2. **User Account Data**
   - plex.tv user info
   - Subscription status
   - Profile preferences

3. **Timeline Reporting**
   - Implement timeline endpoint
   - Track precise playback positions

4. **Server Preferences**
   - Full preferences endpoint
   - Butler task status

5. **Sync/Download APIs**
   - Sync queue analysis
   - Offline viewing patterns

### Phase 3: Specialized Data (P2) - Estimated 30 endpoints/fields

**Goal**: Complete coverage for edge cases

1. **Play Queues**
   - Queue creation/deletion
   - Shuffle patterns

2. **DVR/Live TV**
   - Full DVR support
   - Recording analysis

3. **Friends/Sharing**
   - Social network analysis
   - Sharing patterns

4. **Managed Users**
   - Plex Home analysis
   - Restriction profiles

### Phase 4: Completeness (P3) - Estimated 20 endpoints/fields

**Goal**: 100% coverage

1. **Remaining undocumented endpoints**
2. **Administrative endpoints**
3. **Photo endpoints**

---

## Part 10: Database Schema Changes Required

### New Tables

```sql
-- External identifiers (separate from single GUID)
CREATE TABLE media_external_ids (
    rating_key VARCHAR PRIMARY KEY,
    imdb_id VARCHAR,
    tmdb_id VARCHAR,
    tvdb_id VARCHAR,
    cartographus_id UUID REFERENCES playback_events(id)
);

-- Tag details (replacing comma-separated strings)
CREATE TABLE media_tags (
    id SERIAL PRIMARY KEY,
    rating_key VARCHAR NOT NULL,
    tag_type VARCHAR NOT NULL,  -- genre, director, writer, actor
    tag_id INT,
    tag_name VARCHAR NOT NULL,
    tag_key VARCHAR,
    tag_filter VARCHAR,
    tag_thumb VARCHAR,
    role_name VARCHAR  -- For actors only
);

-- Media versions (multiple per content item)
CREATE TABLE media_versions (
    id SERIAL PRIMARY KEY,
    rating_key VARCHAR NOT NULL,
    media_id INT NOT NULL,
    video_codec VARCHAR,
    video_resolution VARCHAR,
    video_bitrate INT,
    audio_codec VARCHAR,
    audio_channels INT,
    container VARCHAR,
    file_size BIGINT,
    is_optimized BOOLEAN,
    is_synced BOOLEAN
);

-- System resource snapshots
CREATE TABLE system_resources (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    host_cpu_utilization FLOAT,
    process_cpu_utilization FLOAT,
    host_memory_utilization FLOAT,
    process_memory_utilization FLOAT,
    active_transcode_sessions INT
);

-- Server preferences (for change tracking)
CREATE TABLE server_preferences (
    id SERIAL PRIMARY KEY,
    captured_at TIMESTAMPTZ NOT NULL,
    pref_key VARCHAR NOT NULL,
    pref_value VARCHAR,
    pref_type VARCHAR
);

-- Intro/credits markers
CREATE TABLE content_markers (
    id SERIAL PRIMARY KEY,
    rating_key VARCHAR NOT NULL,
    marker_type VARCHAR NOT NULL,  -- intro, credits, commercial, chapter
    start_offset_ms INT NOT NULL,
    end_offset_ms INT NOT NULL,
    chapter_title VARCHAR
);

-- User subscription data
CREATE TABLE user_subscriptions (
    user_id INT PRIMARY KEY,
    subscription_active BOOLEAN,
    subscription_plan VARCHAR,
    subscription_features TEXT[],  -- Array of feature flags
    home_member BOOLEAN,
    home_admin BOOLEAN,
    captured_at TIMESTAMPTZ
);
```

### PlaybackEvent Model Additions

```go
// Add to PlaybackEvent struct
type PlaybackEvent struct {
    // ... existing fields ...

    // P0: Quality signals
    ViewCount  *int `json:"view_count,omitempty"`
    SkipCount  *int `json:"skip_count,omitempty"`

    // P0: Marker flags
    HasIntroMarker    *bool `json:"has_intro_marker,omitempty"`
    HasCreditsMarker  *bool `json:"has_credits_marker,omitempty"`
    HasChapterMarkers *bool `json:"has_chapter_markers,omitempty"`

    // P0: Dolby Vision
    DOVIPresent  *bool   `json:"dovi_present,omitempty"`
    DOVIProfile  *int    `json:"dovi_profile,omitempty"`
    DOVIVersion  *string `json:"dovi_version,omitempty"`
    DOVIBLPresent *bool  `json:"dovi_bl_present,omitempty"`
    DOVIELPresent *bool  `json:"dovi_el_present,omitempty"`

    // P1: Subscription correlation
    UserSubscriptionActive *bool   `json:"user_subscription_active,omitempty"`
    UserSubscriptionPlan   *string `json:"user_subscription_plan,omitempty"`
}
```

---

## Part 11: API Implementation Checklist

### Direct Plex API Endpoints to Add

- [ ] `GET /library/metadata/{id}` - Complete metadata
- [ ] `GET /library/metadata/{id}/children` - Hierarchy
- [ ] `GET /library/metadata/{id}/similar` - Recommendations
- [ ] `GET /statistics/resources` - System health
- [ ] `GET /:/prefs` - Server preferences
- [ ] `GET /butler` - Scheduled tasks
- [ ] `PUT /:/timeline` - Playback reporting
- [ ] `GET /:/scrobble` - Watch completion
- [ ] `PUT /:/rate` - User ratings
- [ ] `GET /hubs/search` - Content search
- [ ] `GET /hubs` - Recommendations
- [ ] `GET /library/sections/{id}/genre` (and other browse endpoints)
- [ ] `GET https://plex.tv/api/v2/user` - Account info
- [ ] `GET /sync/items` - Sync queue
- [ ] `GET /livetv/sessions` - Live TV

### Tautulli API Endpoints to Add

- [ ] `get_plex_log` - Error analysis
- [ ] `get_notification_log` - Notification tracking

### Data Model Enhancements

- [ ] External ID parsing (IMDb, TMDB, TVDB)
- [ ] Tag structure enhancement (arrays with IDs)
- [ ] Media version arrays
- [ ] Dolby Vision fields
- [ ] Marker data
- [ ] System resource tracking
- [ ] User subscription data

---

## Appendix A: Complete Plex API Endpoint Reference

Based on the provided reference document, here is the complete list of 150+ Plex API endpoints:

### Library APIs (~40 endpoints)
1. `/library/sections` - GET
2. `/library/sections/{id}` - GET
3. `/library/sections/{id}/all` - GET
4. `/library/sections/{id}/refresh` - GET
5. `/library/sections/{id}/analyze` - GET
6. `/library/sections/{id}/emptyTrash` - POST
7. `/library/metadata/{ratingKey}` - GET
8. `/library/metadata/{ratingKey}/children` - GET
9. `/library/metadata/{ratingKey}/similar` - GET
10. `/library/metadata/{ratingKey}/related` - GET
11. `/library/metadata/{ratingKey}/posters` - GET/POST
12. `/library/metadata/{ratingKey}/matches` - GET
13. `/library/metadata/{ratingKey}/match` - PUT
14. `/library/metadata/{ratingKey}/unmatch` - DELETE
15. `/library/metadata/{ratingKey}/split` - POST
16. `/library/metadata/{ratingKey}/merge` - PUT
17. `/library/sections/{id}/firstCharacter` - GET
18. `/library/sections/{id}/genre` - GET
19. `/library/sections/{id}/year` - GET
20. `/library/sections/{id}/decade` - GET
21. `/library/sections/{id}/director` - GET
22. `/library/sections/{id}/actor` - GET
23. `/library/sections/{id}/contentRating` - GET
24. `/library/sections/{id}/resolution` - GET
25. `/library/sections/{id}/studio` - GET
26. `/library/sections/{id}/filters` - GET
27. `/library/sections/{id}/sorts` - GET
28. `/library/sections/{id}/collections` - GET
29. `/library/collections/{ratingKey}` - GET
30. `/playlists` - GET
31. `/playlists/{ratingKey}` - GET
32. `/playlists/{ratingKey}/items` - GET

### Server APIs (~25 endpoints)
33. `/` - GET (root with capabilities)
34. `/identity` - GET
35. `/:/prefs` - GET
36. `/butler` - GET
37. `/butler/{taskName}` - POST/DELETE
38. `/transcode/sessions` - GET
39. `/activities` - GET
40. `/statistics/resources` - GET
41. `/statistics/bandwidth` - GET

### Session APIs (~15 endpoints)
42. `/status/sessions` - GET
43. `/status/sessions/history/all` - GET
44. `/:/timeline` - PUT
45. `/:/scrobble` - GET
46. `/:/unscrobble` - GET
47. `/:/rate` - PUT
48. `/playQueues` - POST
49. `/playQueues/{id}` - GET/DELETE

### Search APIs (~10 endpoints)
50. `/hubs/search` - GET
51. `/hubs` - GET
52. `/hubs/sections/{sectionID}` - GET

### User APIs (plex.tv) (~20 endpoints)
53. `/api/v2/users/signin` - POST
54. `/api/v2/pins` - POST
55. `/api/v2/pins/{pinId}` - GET
56. `/api/v2/user` - GET
57. `/api/v2/home/users` - GET
58. `/api/v2/home/users/restricted` - POST
59. `/api/v2/friends` - GET
60. `/api/servers/{machineId}/shared_servers` - POST

### Sync APIs (~5 endpoints)
61. `/sync/items` - GET/POST

### Transcode Control (~5 endpoints)
62. `/video/:/transcode/universal/decision` - GET
63. `/video/:/transcode/universal/start.m3u8` - GET
64. `/video/:/transcode/universal/ping` - GET
65. `/video/:/transcode/universal/stop` - GET
66. `/photo/:/transcode` - GET

### DVR/Live TV (~10 endpoints)
67. `/livetv/sessions` - GET
68. `/livetv/dvrs` - GET
69. `/livetv/dvrs/{id}/channels` - GET
70. `/media/subscriptions` - GET

### Internal/Undocumented (~15 endpoints)
71. `/library/parts/{id}` - GET
72. `/library/parts/{id}/file` - GET
73. `/library/streams/{id}` - GET

---

## Appendix B: Data Science Use Cases by Priority

### P0 Use Cases (Blocked Without This Data)

1. **Content Quality Detection**: Skip counts reveal problematic content
2. **Binge Pattern Analysis**: Intro/credits markers show skip behavior
3. **Cross-Platform Enrichment**: IMDb/TMDB IDs enable external data joins
4. **HDR Compatibility Analysis**: Full Dolby Vision data required
5. **Cast/Crew Analysis**: Proper tag structure with IDs needed
6. **Version Selection Analysis**: Multiple media versions required

### P1 Use Cases (Limited Without This Data)

1. **Recommendation Quality**: Hub data shows recommendation effectiveness
2. **Search Behavior**: Search query analysis
3. **User Segmentation**: Subscription and profile data
4. **System Optimization**: CPU/memory correlation with issues
5. **Offline Viewing Patterns**: Sync data analysis

### P2 Use Cases (Enhanced With This Data)

1. **Queue Behavior**: Play queue patterns
2. **Social Analysis**: Friend network and sharing
3. **DVR Patterns**: Recording analysis
4. **Maintenance Correlation**: Butler task impact

---

**Document End**

*This gap analysis should be reviewed and updated as implementation progresses. Target completion: Full P0 coverage within 4-6 weeks, P1 within 8-10 weeks.*
