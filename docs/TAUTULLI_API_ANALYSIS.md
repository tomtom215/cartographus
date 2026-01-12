# Tautulli API Surface Analysis

**Generated**: 2025-11-21
**Last Updated**: 2025-11-22 (Post-v1.19 Enhanced Metadata & Export Implementation)
**Purpose**: Comprehensive analysis of Tautulli's complete API surface to identify opportunities for Cartographus

---

## Executive Summary

**Total Tautulli API Endpoints**: 93
**Currently Used by Cartographus**: 41 (44.1%) ⬆️ +5 endpoints (v1.20 - 2025-11-22)
**Available for Integration**: 52 (55.9%)

This analysis reveals significant opportunities to enhance Cartographus by leveraging additional Tautulli API capabilities, particularly in areas of:
- Advanced user analytics
- Library management and statistics
- Notification and newsletter systems
- Collection and playlist data
- Server management and diagnostics
- Data export capabilities

---

## Currently Used Endpoints (36)

### Core Data Sync
1. **get_history** - Primary playback history retrieval
2. **get_geoip_lookup** - Geographic location resolution for IP addresses
3. **arnold** - Connection health check (Ping)

### Statistics & Analytics (Core)
4. **get_home_stats** - Homepage statistics (top movies, users, platforms)
5. **get_plays_by_date** - Daily playback trends
6. **get_plays_by_dayofweek** - Weekly pattern analysis
7. **get_plays_by_hourofday** - Hourly activity distribution
8. **get_plays_by_stream_type** - Streaming method breakdown over time
9. **get_concurrent_streams_by_stream_type** - Concurrent streams by type
10. **get_item_watch_time_stats** - Watch time statistics per media item

### Statistics & Analytics (Priority 1 - Implemented v1.16)
11. **get_plays_by_source_resolution** - Original content quality distribution
12. **get_plays_by_stream_resolution** - Delivered streaming resolutions
13. **get_plays_by_top_10_platforms** - Platform leaderboard
14. **get_plays_by_top_10_users** - User ranking by activity
15. **get_plays_per_month** - Long-term monthly trends
16. **get_user_player_stats** - Per-user platform preferences
17. **get_user_watch_time_stats** - User engagement by media type
18. **get_item_user_stats** - Content demographics

### Library Management (Core + Priority 2 - Implemented v1.16)
19. **get_libraries** - All library sections
20. **get_library** - Specific library details
21. **get_library_user_stats** - User activity per library
22. **get_libraries_table** - Paginated library management
23. **get_library_media_info** - Library content with technical specs
24. **get_library_watch_time_stats** - Library-specific analytics
25. **get_children_metadata** - Episode/season/track metadata

### Activity Monitoring
26. **get_activity** - Current server activity and active sessions

### Metadata & Content
27. **get_metadata** - Rich media metadata
28. **get_recently_added** - Recently added content
29. **get_user** - User profile information

### Server Information
30. **get_server_info** - Plex Media Server information

### Device Management
31. **get_synced_items** - Synced media on devices
32. **terminate_session** - Terminate active sessions

### User Geography & Management (Priority 1 - Implemented v1.17)
33. **get_user_ips** - IP address history per user for geographic mobility patterns
34. **get_users_table** - Paginated user data with sorting/filtering
35. **get_user_logins** - Login history and patterns for security analytics

### Enhanced Metadata & Export (Priority 2 - Implemented v1.19)
36. **get_stream_data** - Enhanced per-session quality details (transcoding decisions, codecs, bitrates)
37. **get_library_names** - Simple ID→Name mapping for efficient library lookups
38. **export_metadata** - Initiate metadata export with configurable fields for data portability
39. **get_export_fields** - Available export fields per media type for export customization

### Advanced Analytics & Metadata (Priority 3 - Implemented v1.20)
40. **get_stream_type_by_top_10_users** - User quality preferences (direct play vs transcode patterns)
41. **get_stream_type_by_top_10_platforms** - Platform transcoding patterns (which devices require transcoding)
42. **search** - Global search across Tautulli data (media titles, users, libraries)
43. **get_new_rating_keys** - Updated content identifiers after Plex database changes
44. **get_old_rating_keys** - Historical content identifier mappings for legacy tracking

**Total**: 41 endpoints (3 core sync + 38 proxied Tautulli endpoints)

---

## Available But Unused Endpoints (57)

### Category: User Analytics (4 endpoints remaining) - HIGH PRIORITY

**Why Important**: Enhanced user engagement metrics and behavior analysis

| Endpoint | Purpose | Cartographus Use Case |
|----------|---------|------------------------|
| `edit_user` | Update user profile settings | User management features |
| `delete_user` | Remove user and history | Administrative functions |
| `get_pms_token` | Retrieve Plex authentication token | Advanced Plex integration |
| `refresh_users_list` | Force user list refresh | Data sync management |

**Note**: ✓ `get_user_ips`, `get_users_table`, and `get_user_logins` implemented in v1.17

**Impact**: Would enable comprehensive user engagement analytics (already partially implemented with UserEngagement models)

---

### Category: Statistics & Analytics (Advanced - Still Unused) - MEDIUM PRIORITY

**Why Important**: Additional advanced analytics for power users

| Endpoint | Purpose | Cartographus Use Case |
|----------|---------|------------------------|
| `get_stream_type_by_top_10_users` | User streaming preferences | User quality preferences analysis |
| `get_stream_type_by_top_10_platforms` | Platform streaming methods | Platform-specific transcoding patterns |

**Status**: ✅ Priority 1 endpoints implemented (v1.16), above remain as future enhancements

**Impact**: Would add user-specific transcoding preference insights

---

### Category: Library Management (Still Unused) - LOW PRIORITY

| Endpoint | Purpose | Cartographus Use Case |
|----------|---------|------------------------|
| `edit_library` | Update library settings | Library configuration |
| `delete_library` | Remove library and history | Administrative functions |
| `delete_all_library_history` | Purge library viewing records | Data management |
| `delete_media_info_cache` | Clear media cache | Performance optimization |
| `refresh_libraries_list` | Force library refresh | Data sync management |

**Status**: ✅ Priority 2 core library endpoints implemented (v1.16): `get_libraries_table`, `get_library_media_info`, `get_library_watch_time_stats`, `get_children_metadata`

**Impact**: Remaining endpoints are administrative/management functions with lower priority for analytics

---

### Category: Collections & Playlists (2 endpoints) - LOW PRIORITY

| Endpoint | Purpose | Cartographus Use Case |
|----------|---------|------------------------|
| `get_collections_table` | Collection data | Content grouping analytics |
| `get_playlists_table` | Playlist information | Playlist usage patterns |

**Impact**: New content organization insights

---

### Category: Notification System (11 endpoints) - LOW PRIORITY

| Endpoint | Purpose | Potential Use Case |
|----------|---------|-------------------|
| `get_notifiers` | List notification agents | Integration with external systems |
| `get_notifier_config` | Agent configuration | Notification management |
| `get_notifier_parameters` | Available variables | Custom notification formatting |
| `get_notification_log` | Notification history | Audit trail |
| `add_notifier_config` | Register agent | Alert configuration |
| `delete_notifier` | Remove agent | Agent management |
| `delete_notification_log` | Clear records | Log management |
| `notify` | Send notification | Custom alerts for map events |
| `register_device` | Register mobile app | Mobile integration |
| `delete_mobile_device` | Unregister device | Device management |
| `pms_image_proxy` | Proxy Plex images | Image optimization |

**Impact**: Would enable real-time notifications for geographic events (e.g., "New viewer from Antarctica!")

---

### Category: Newsletter System (6 endpoints) - LOW PRIORITY

| Endpoint | Purpose | Potential Use Case |
|----------|---------|-------------------|
| `get_newsletters` | List newsletters | Automated reports |
| `get_newsletter_config` | Configuration | Newsletter management |
| `get_newsletter_log` | Newsletter history | Report tracking |
| `add_newsletter_config` | Register newsletter | Report scheduling |
| `delete_newsletter` | Remove newsletter | Newsletter management |
| `delete_newsletter_log` | Clear records | Log management |
| `notify_newsletter` | Send newsletter | Weekly geographic summary emails |

**Impact**: Would enable automated geographic analytics reports

---

### Category: Metadata & Search (6 endpoints) - MEDIUM PRIORITY

| Endpoint | Purpose | Cartographus Use Case |
|----------|---------|------------------------|
| `get_children_metadata` | Child item details | Season/episode navigation |
| `get_new_rating_keys` | Updated item identifiers | Content tracking |
| `get_old_rating_keys` | Previous identifiers | Historical tracking |
| `search` | Search Tautulli data | Global search functionality |
| `delete_lookup_info` | Clear third-party cache | Performance optimization |

**Impact**: Enhanced metadata navigation and search

---

### Category: Data Export (5 endpoints) - MEDIUM PRIORITY

| Endpoint | Purpose | Cartographus Use Case |
|----------|---------|------------------------|
| `get_exports_table` | Export history | Export management |
| `download_export` | Retrieve exported file | Data download |
| `delete_export` | Remove export file | Export cleanup |

**Impact**: Would enable GeoJSON and CSV exports with richer metadata (already partially implemented)

---

### Category: Server Management (10 endpoints) - LOW PRIORITY

| Endpoint | Purpose | Potential Use Case |
|----------|---------|-------------------|
| `get_server_friendly_name` | Display name | Server identification |
| `get_server_id` | Server identifier | Multi-server support |
| `get_server_identity` | Machine info | Server tracking |
| `get_server_list` | Published servers | Multi-server selection |
| `get_server_pref` | Preference value | Server settings |
| `get_servers_info` | Infrastructure details | Multi-server analytics |
| `get_pms_update` | Check for updates | Update notifications |
| `update_metadata` | Refresh metadata | Metadata management |
| `delete_temp_sessions` | Clear temporary cache | Performance optimization |
| `get_tautulli_info` | Version and platform | System information |

**Impact**: Multi-server support (future enhancement)

---

### Category: Logs & Diagnostics (7 endpoints) - LOW PRIORITY

| Endpoint | Purpose | Potential Use Case |
|----------|---------|-------------------|
| `get_logs` | Application logs | Debugging and monitoring |
| `get_plex_log` | PMS logs | PMS troubleshooting |
| `download_log` | Download app logs | Log export |
| `download_plex_log` | Download PMS logs | PMS log export |
| `get_date_formats` | Temporal formats | Date formatting |
| `delete_login_log` | Clear login records | Log management |
| `sql` | Raw SQL queries | Advanced queries (requires API_SQL flag) |

**Impact**: Enhanced debugging and monitoring capabilities

---

### Category: Configuration & Backup (6 endpoints) - LOW PRIORITY

| Endpoint | Purpose | Potential Use Case |
|----------|---------|-------------------|
| `get_settings` | Configuration values | Settings management |
| `get_apikey` | Generate/retrieve API key | API key management |
| `backup_config` | Backup config.ini | Configuration backup |
| `backup_db` | Backup plexpy.db | Database backup |
| `download_config` | Download config file | Config export |
| `download_database` | Download database | Database export |

**Impact**: Administrative and backup capabilities

---

### Category: System Control (3 endpoints) - LOW PRIORITY

| Endpoint | Purpose | Potential Use Case |
|----------|---------|-------------------|
| `restart` | Restart Tautulli | Remote management |
| `update` | Update Tautulli | Version management |
| `shutdown` | Shutdown Tautulli | Remote control |

**Impact**: Remote Tautulli management

---

### Category: Cache Management (6 endpoints) - LOW PRIORITY

| Endpoint | Purpose | Potential Use Case |
|----------|---------|-------------------|
| `delete_cache` | Clear all cache | Performance optimization |
| `delete_image_cache` | Clear cached images | Storage management |
| `delete_hosted_images` | Remove uploaded images | Storage cleanup |
| `delete_recently_added` | Flush recently added | Cache management |
| `delete_all_user_history` | Purge user records | Privacy compliance |
| `delete_history` | Remove specific entries | Data management |

**Impact**: Data and cache management features

---

### Category: Documentation (2 endpoints) - INFORMATIONAL

| Endpoint | Purpose | Use Case |
|----------|---------|----------|
| `docs` | API documentation dictionary | Runtime documentation |
| `docs_md` | API documentation in Markdown | Documentation generation |

---

## Data Fields Available from get_history

Tautulli's `get_history` endpoint provides **60+ fields** per playback record. Cartographus currently stores **55+ fields** in the `PlaybackEvent` model.

### Already Captured Fields (55)

**Core Playback Data** (14 fields):
- session_key, started, stopped, user_id, user, ip_address
- media_type, title, parent_title, grandparent_title
- percent_complete, paused_counter, play_duration, duration

**Platform & Player** (3 fields):
- platform, player, location

**Quality & Streaming** (15 fields):
- transcode_decision, video_resolution, video_codec, audio_codec
- stream_video_resolution, stream_audio_codec, stream_audio_channels
- stream_video_decision, stream_audio_decision, stream_container
- stream_bitrate, video_dynamic_range, video_framerate
- video_bitrate, video_bit_depth

**Metadata Enrichment** (15 fields) - **ADDED 2025-11-21**:
- rating_key, parent_rating_key, grandparent_rating_key
- media_index, parent_media_index (CRITICAL for binge detection)
- guid, original_title, full_title, originally_available_at
- watched_status, thumb
- directors, writers, actors, genres (cast/crew analytics)

**Audio Details** (5 fields):
- audio_channels, audio_channel_layout, audio_bitrate
- audio_sample_rate, audio_language

**Video Details** (3 fields):
- video_width, video_height

**Container & Subtitles** (4 fields):
- container, subtitle_codec, subtitle_language, subtitles

**Connection Security** (3 fields):
- secure, relayed, local

**Library & Content** (4 fields):
- section_id, library_name, content_rating, year

**File Metadata** (2 fields):
- file_size, bitrate

### Missing But Available from Tautulli (5+ fields)

**Identifiers**:
- `row_id` - Tautulli database row ID (useful for updates/deletes)
- `reference_id` - Reference to parent record (session continuations)
- `machine_id` - Device identifier (for device tracking)

**Session State**:
- `state` - Current playback state (playing, paused, stopped) - only for active sessions
- `session_key` (for active sessions vs completed history)

**Group Data** (for aggregated views):
- `group_count` - Number of grouped playbacks
- `group_ids` - Comma-separated IDs of grouped playbacks

**Date** (for filtering):
- `date` - Unix timestamp (same as `started`, but sometimes different in Tautulli)

**Note**: Most of these are internal Tautulli fields. The core playback data is already comprehensive.

---

## Recommendations for Cartographus Integration

### Priority 1: HIGH - Immediate Value (Complete Analytics Dashboard)

**Goal**: Complete the 43-chart analytics dashboard with missing data sources

1. **Fix Concurrent Streams Bug**:
   - Change `GetConcurrentStreamsByStreamType()` to call correct endpoint: `get_concurrent_streams_by_stream_type`
   - Currently calls: `get_stream_type_by_top_10_platforms` (wrong endpoint)
   - Impact: Fixes concurrent streams analytics

2. **Add Missing Analytics Endpoints**:
   - `get_plays_by_source_resolution` → Source quality charts
   - `get_plays_by_stream_resolution` → Stream quality charts
   - `get_plays_by_top_10_platforms` → Platform leaderboard
   - `get_plays_by_top_10_users` → User leaderboard
   - `get_plays_per_month` → Long-term trend charts

3. **Enhanced User Analytics**:
   - `get_user_player_stats` → Per-user platform preferences
   - `get_user_watch_time_stats` → User engagement trends
   - `get_item_user_stats` → Content popularity demographics

**Estimated Impact**: Completes analytics dashboard, fixes existing bug, adds 8-10 new charts

---

### Priority 2: MEDIUM - Enhanced Features

**Goal**: Add library-specific analytics and improved metadata navigation

1. **Library Analytics**:
   - `get_libraries_table` → Library management dashboard
   - `get_library_media_info` → Media quality per library
   - `get_library_watch_time_stats` → Library usage trends
   - Impact: Enables library-specific analytics (models already exist)

2. **Metadata Navigation**:
   - `get_children_metadata` → Season/episode navigation
   - Impact: Richer content exploration

3. **Data Export Enhancement**:
   - ✓ `export_metadata` → Implemented in v1.19
   - ✓ `get_export_fields` → Implemented in v1.19
   - Impact: Better data portability (COMPLETED)

**Estimated Impact**: 5-7 new features, enhanced metadata capabilities

---

### Priority 3: LOW - Future Enhancements

**Goal**: Administrative features and advanced integrations

1. **Multi-Server Support**:
   - `get_servers_info`, `get_server_list`
   - Impact: Multi-server geographic visualization

2. **Notification System**:
   - `notify`, `get_notifiers`
   - Impact: Real-time geographic event alerts

3. **Newsletter Integration**:
   - `notify_newsletter`, `get_newsletters`
   - Impact: Automated weekly geographic reports

4. **Administrative Tools**:
   - User management (edit_user, delete_user)
   - Library management (edit_library, delete_library)
   - Cache management (delete_cache, etc.)

**Estimated Impact**: 10-15 new administrative features

---

## Technical Implementation Notes

### API Client Updates Required

**File**: `/home/user/map/internal/sync/tautulli.go`

**Add Methods**:
```go
// Priority 1: HIGH
GetPlaysBySourceResolution(timeRange int, yAxis string, userID int, grouping int)
GetPlaysByStreamResolution(timeRange int, yAxis string, userID int, grouping int)
GetPlaysByTop10Platforms(timeRange int, yAxis string, userID int, grouping int)
GetPlaysByTop10Users(timeRange int, yAxis string, userID int, grouping int)
GetPlaysPerMonth(timeRange int, yAxis string, userID int, grouping int)
GetUserPlayerStats(userID int)
GetUserWatchTimeStats(userID int, queryDays string)
GetItemUserStats(ratingKey string, grouping int)

// Priority 2: MEDIUM
GetLibrariesTable(grouping int, orderColumn string, orderDir string, start int, length int, search string)
GetLibraryMediaInfo(sectionID int, orderColumn string, orderDir string, start int, length int)
GetLibraryWatchTimeStats(sectionID int, grouping int, queryDays string)
GetChildrenMetadata(ratingKey string, mediaType string)
GetStreamData(rowID int) // or sessionKey
ExportMetadata(params ExportParams)
GetExportFields()
```

**Bug Fix**:
```go
// WRONG (current implementation):
func (c *TautulliClient) GetConcurrentStreamsByStreamType(timeRange int, userID int) {
    params.Set("cmd", "get_stream_type_by_top_10_platforms") // WRONG!
}

// CORRECT (should be):
func (c *TautulliClient) GetConcurrentStreamsByStreamType(timeRange int, userID int) {
    params.Set("cmd", "get_concurrent_streams_by_stream_type") // CORRECT!
}
```

---

### Model Updates Required

**File**: `/home/user/map/internal/models/models.go`

**Add Models** (Priority 1):
```go
// Tautulli API Response Models (add to existing)
type TautulliPlaysBySourceResolution struct { ... }
type TautulliPlaysByStreamResolution struct { ... }
type TautulliPlaysByTop10Platforms struct { ... }
type TautulliPlaysByTop10Users struct { ... }
type TautulliPlaysPerMonth struct { ... }
type TautulliUserPlayerStats struct { ... }
type TautulliUserWatchTimeStatsResponse struct { ... }
type TautulliItemUserStats struct { ... }
```

**Add Models** (Priority 2):
```go
type TautulliLibrariesTable struct { ... }
type TautulliLibraryMediaInfo struct { ... }
type TautulliChildrenMetadata struct { ... }
type TautulliStreamDataResponse struct { ... }
type TautulliExportMetadata struct { ... }
```

---

### Database Schema Updates

**No schema changes required** - Most new endpoints return aggregated statistics that don't need to be stored. They can be cached and served directly to the frontend.

**Optional Enhancement**: Add caching for aggregated statistics:
- Add cache entries for new analytics endpoints
- 5-minute TTL (matches current cache strategy)
- Clear on sync completion

---

### Frontend Integration

**Add New Charts** (Priority 1):

**File**: `/home/user/map/web/src/lib/charts/renderers/TrendsChartRenderer.ts`
- Monthly playback trends (get_plays_per_month)

**File**: `/home/user/map/web/src/lib/charts/renderers/GeographicChartRenderer.ts`
- Source vs stream resolution comparison
- Resolution distribution charts

**File**: `/home/user/map/web/src/lib/charts/renderers/UserChartRenderer.ts`
- User leaderboard (get_plays_by_top_10_users)
- Per-user platform statistics
- User watch time trends

**File**: `/home/user/map/web/src/lib/charts/renderers/PerformanceChartRenderer.ts`
- Platform leaderboard (get_plays_by_top_10_platforms)
- Source quality distribution
- Stream quality distribution

**Add New Pages** (Priority 2):
- Library analytics page with library-specific charts
- User management page
- Export configuration page

---

## API Version & Compatibility

**Tautulli API Version**: v2 (stable)
**Endpoint Base**: `/api/v2`
**Authentication**: Query parameter `apikey`
**Format**: JSON responses

**Key Compatibility Notes**:
1. All endpoints return consistent response format:
   ```json
   {
     "response": {
       "result": "success",
       "message": null,
       "data": { ... }
     }
   }
   ```

2. Error handling pattern:
   - Check `response.result` for "success"
   - Read `response.message` for error details
   - Handle null data gracefully

3. Optional parameters:
   - All filtering parameters are optional
   - Use default values when not specified
   - Empty strings are treated as "not set"

4. Pagination:
   - Standard DataTables format: `start`, `length`, `order_column`, `order_dir`
   - Default length: 25 items
   - Returns `recordsTotal` and `recordsFiltered`

---

## Conclusion

Cartographus currently leverages **21.5%** of Tautulli's API capabilities. The remaining **78.5%** represents significant opportunities for:

1. **Completing the analytics dashboard** (Priority 1) - 8-10 missing charts
2. **Fixing concurrent streams bug** (Priority 1) - Incorrect endpoint
3. **Adding library-specific analytics** (Priority 2) - 5-7 new features
4. **Enhancing metadata navigation** (Priority 2) - Richer content exploration
5. **Administrative features** (Priority 3) - User/library management
6. **Multi-server support** (Priority 3) - Geographic visualization across servers
7. **Notification integration** (Priority 3) - Real-time geographic alerts

**Immediate Next Steps**:
1. Fix `GetConcurrentStreamsByStreamType()` endpoint call (BUG)
2. Add Priority 1 analytics endpoints (8 methods)
3. Create corresponding models in models.go
4. Add 8-10 new charts to complete analytics dashboard
5. Update API handler tests

**Estimated Development Time**:
- Priority 1 (Bug fix + 8 endpoints + 10 charts): 2-3 days
- Priority 2 (Library analytics + metadata): 1-2 days
- Priority 3 (Admin features + notifications): 3-4 days
- **Total**: 6-9 days for complete Tautulli API integration

---

## Appendix: Complete Tautulli API Endpoint List

### Currently Used (20)
1. ✓ arnold
2. ✓ get_activity
3. ✓ get_geoip_lookup
4. ✓ get_history
5. ✓ get_home_stats
6. ✓ get_item_watch_time_stats
7. ✓ get_libraries
8. ✓ get_library
9. ✓ get_library_user_stats
10. ✓ get_metadata
11. ✓ get_plays_by_date
12. ✓ get_plays_by_dayofweek
13. ✓ get_plays_by_hourofday
14. ✓ get_plays_by_stream_type
15. ✓ get_recently_added
16. ✓ get_server_info
17. ✓ get_stream_type_by_top_10_platforms (used incorrectly as concurrent streams)
18. ✓ get_synced_items
19. ✓ get_user
20. ✓ terminate_session

### Available But Unused (73)
21. add_newsletter_config
22. add_notifier_config
23. backup_config
24. backup_db
25. delete_all_library_history
26. delete_all_user_history
27. delete_cache
28. delete_export
29. delete_history
30. delete_hosted_images
31. delete_image_cache
32. delete_library
33. delete_login_log
34. delete_lookup_info
35. delete_media_info_cache
36. delete_mobile_device
37. delete_newsletter
38. delete_newsletter_log
39. delete_notification_log
40. delete_notifier
41. delete_recently_added
42. delete_synced_item
43. delete_temp_sessions
44. delete_user
45. docs
46. docs_md
47. download_config
48. download_database
49. download_export
50. download_log
51. download_plex_log
52. edit_library
53. edit_user
54. get_apikey
56. get_children_metadata
57. get_collections_table
58. get_concurrent_streams_by_stream_type (NOT USED - bug in client)
59. get_date_formats
60. get_exports_table
62. get_item_user_stats
63. get_libraries_table
64. get_library_media_info
65. get_library_watch_time_stats
67. get_logs
68. get_new_rating_keys
69. get_newsletter_config
70. get_newsletter_log
71. get_newsletters
72. get_notification_log
73. get_notifier_config
74. get_notifier_parameters
75. get_notifiers
76. get_old_rating_keys
77. get_plex_log
78. get_playlists_table
79. get_plays_by_source_resolution
80. get_plays_by_stream_resolution
81. get_plays_by_top_10_platforms
82. get_plays_by_top_10_users
83. get_plays_per_month
84. get_pms_update
85. get_server_friendly_name
86. get_server_id
87. get_server_identity
88. get_server_list
89. get_server_pref
90. get_servers_info
91. get_settings
92. get_stream_type_by_top_10_users
94. get_tautulli_info
95. get_user_ips (not documented but exists)
96. get_user_logins (not documented but exists)
97. get_user_player_stats
98. get_user_watch_time_stats
99. get_users_table
100. notify
101. notify_newsletter
102. pms_image_proxy
103. refresh_libraries_list
104. refresh_users_list
105. register_device
106. restart
107. search (not documented but may exist)
108. shutdown (not documented but may exist)
109. sql
110. update
111. update_metadata (not documented but may exist)

**Total**: 93+ documented endpoints (some undocumented endpoints may exist)

---

## References

- **Tautulli GitHub**: https://github.com/Tautulli/Tautulli
- **Tautulli API Wiki**: https://github.com/Tautulli/Tautulli/wiki/Tautulli-API-Reference
- **Cartographus Repository**: https://github.com/tomtom215/cartographus
- **DuckDB Spatial Extension**: https://duckdb.org/docs/extensions/spatial

---

**Document Version**: 1.0
**Last Updated**: 2025-11-21
**Author**: Claude Code AI Assistant
**Review Status**: Ready for implementation
