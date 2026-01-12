#!/usr/bin/env python3
"""
Categorize models.go structs into domain-specific files.
This script analyzes the models.go file and categorizes all 301 structs.
"""
import re
from collections import defaultdict

def categorize_struct(struct_name):
    """Categorize a struct based on its name."""

    # Internal domain models (not Tautulli-related)
    if struct_name in ['PlaybackEvent', 'Geolocation', 'LocationStats']:
        return 'playback'

    # API Response structures
    if struct_name in ['APIResponse', 'APIError', 'Metadata', 'LoginResponse']:
        return 'api_responses'

    # Analytics structures (non-Tautulli)
    if any(x in struct_name for x in ['TrendsResponse', 'GeographicResponse', 'UsersResponse',
                                       'BingeAnalytics', 'BandwidthAnalytics', 'PopularAnalytics',
                                       'WatchPartyAnalytics', 'UserEngagementAnalytics',
                                       'ContentAbandonmentAnalytics', 'ComparativeAnalytics',
                                       'TemporalHeatmapResponse', 'ResolutionMismatchAnalytics',
                                       'HDRAnalytics', 'AudioAnalytics', 'SubtitleAnalytics',
                                       'FrameRateAnalytics', 'ContainerAnalytics',
                                       'ConnectionSecurityAnalytics', 'PausePatternAnalytics',
                                       'LibraryAnalytics', 'ConcurrentStreamsAnalytics']):
        return 'analytics'

    # Filter structures
    if 'Filter' in struct_name:
        return 'filters'

    # Tautulli models - need subcategorization
    if struct_name.startswith('Tautulli'):
        # Activity
        if any(x in struct_name for x in ['Activity', 'Session']):
            return 'tautulli/activity'

        # History
        if 'History' in struct_name:
            return 'tautulli/history'

        # Library
        if any(x in struct_name for x in ['Library', 'Libraries', 'RecentlyAdded']):
            return 'tautulli/library'

        # User
        if any(x in struct_name for x in ['User', 'UserIPs', 'UserLogins', 'UserPlayer',
                                           'UserWatchTime', 'ItemUser']):
            return 'tautulli/user'

        # Server
        if any(x in struct_name for x in ['Server', 'ServerInfo', 'ServerPref', 'ServerList',
                                           'ServerFriendlyName', 'ServerID', 'ServerIdentity',
                                           'PMSUpdate']):
            return 'tautulli/server'

        # Analytics
        if any(x in struct_name for x in ['PlaysByDate', 'PlaysByDayOfWeek', 'PlaysByHourOfDay',
                                           'PlaysByStreamType', 'PlaysBySourceResolution',
                                           'PlaysByStreamResolution', 'PlaysByTop10',
                                           'PlaysPerMonth', 'ConcurrentStreams', 'HomeStats',
                                           'ItemWatchTime', 'StreamType']):
            return 'tautulli/analytics'

        # Export
        if any(x in struct_name for x in ['Export', 'Download']):
            return 'tautulli/export'

        # Metadata
        if any(x in struct_name for x in ['Metadata', 'MediaInfo', 'ChildrenMetadata']):
            return 'tautulli/metadata'

        # Playlists & Collections
        if any(x in struct_name for x in ['Playlist', 'Collection']):
            return 'tautulli/playlists'

        # Search and rating keys
        if any(x in struct_name for x in ['Search', 'RatingKeys']):
            return 'tautulli/search'

        # GeoIP
        if 'GeoIP' in struct_name:
            return 'tautulli/geoip'

        # Synced items, Terminate
        if any(x in struct_name for x in ['Synced', 'Terminate']):
            return 'tautulli/common'

        # Stream data
        if 'StreamData' in struct_name:
            return 'tautulli/stream'

        # Fallback for other Tautulli structs
        return 'tautulli/common'

    # Unknown - should be investigated
    return 'unknown'

def analyze_models_file(filepath):
    """Analyze models.go and categorize all structs."""
    with open(filepath, 'r') as f:
        content = f.read()

    # Find all struct definitions
    struct_pattern = r'^type\s+(\w+)\s+struct'
    structs = re.findall(struct_pattern, content, re.MULTILINE)

    # Categorize
    categories = defaultdict(list)
    for struct in structs:
        category = categorize_struct(struct)
        categories[category].append(struct)

    return categories, len(structs)

if __name__ == '__main__':
    filepath = '/home/user/map/internal/models/models.go'
    categories, total = analyze_models_file(filepath)

    print(f"Total structs found: {total}")
    print(f"\nCategorization:")
    print("=" * 80)

    for category in sorted(categories.keys()):
        structs = categories[category]
        print(f"\n{category} ({len(structs)} structs):")
        print("-" * 80)
        for struct in sorted(structs):
            print(f"  - {struct}")

    print("\n" + "=" * 80)
    print("\nFile organization plan:")
    print("=" * 80)

    file_map = {
        'playback': 'internal/models/playback.go',
        'api_responses': 'internal/models/api_responses.go',
        'analytics': 'internal/models/analytics.go',
        'filters': 'internal/models/filters.go',
        'tautulli/activity': 'internal/models/tautulli/activity.go',
        'tautulli/history': 'internal/models/tautulli/history.go',
        'tautulli/library': 'internal/models/tautulli/library.go',
        'tautulli/user': 'internal/models/tautulli/user.go',
        'tautulli/server': 'internal/models/tautulli/server.go',
        'tautulli/analytics': 'internal/models/tautulli/analytics.go',
        'tautulli/export': 'internal/models/tautulli/export.go',
        'tautulli/metadata': 'internal/models/tautulli/metadata.go',
        'tautulli/playlists': 'internal/models/tautulli/playlists.go',
        'tautulli/search': 'internal/models/tautulli/search.go',
        'tautulli/geoip': 'internal/models/tautulli/geoip.go',
        'tautulli/stream': 'internal/models/tautulli/stream.go',
        'tautulli/common': 'internal/models/tautulli/common.go',
        'unknown': 'internal/models/REVIEW_NEEDED.go',
    }

    for category in sorted(categories.keys()):
        target_file = file_map.get(category, 'UNKNOWN')
        count = len(categories[category])
        print(f"{category:30s} → {target_file:50s} ({count:3d} structs)")
