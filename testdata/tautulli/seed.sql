-- Tautulli Database Schema for Testing
-- Auto-generated schema reference for creating test databases
--
-- This file documents the Tautulli SQLite database schema used by Cartographus
-- for import functionality. The schema matches Tautulli v2.x database structure.

-- =============================================================================
-- session_history - Core playback session information
-- =============================================================================
CREATE TABLE IF NOT EXISTS session_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    started INTEGER NOT NULL,                    -- Unix timestamp of playback start
    stopped INTEGER,                             -- Unix timestamp of playback end
    rating_key INTEGER,                          -- Plex media rating key
    parent_rating_key INTEGER,                   -- Season/album rating key
    grandparent_rating_key INTEGER,              -- Show/artist rating key
    media_type TEXT,                             -- movie, episode, track
    user_id INTEGER NOT NULL,                    -- Plex user ID
    user TEXT NOT NULL,                          -- Username
    friendly_name TEXT,                          -- Display name
    ip_address TEXT,                             -- Client IP address
    platform TEXT,                               -- Client platform (Chrome, iOS, etc)
    player TEXT,                                 -- Player application
    product TEXT,                                -- Product name
    machine_id TEXT,                             -- Device machine ID
    bandwidth INTEGER,                           -- Connection bandwidth (kbps)
    location TEXT,                               -- lan or wan
    quality_profile TEXT,                        -- Quality profile used
    session_key TEXT NOT NULL,                   -- Unique session identifier
    view_offset INTEGER DEFAULT 0,               -- Current playback position (ms)
    percent_complete INTEGER DEFAULT 0,          -- Playback completion percentage
    paused_counter INTEGER DEFAULT 0,            -- Number of times paused
    transcode_decision TEXT,                     -- direct play, transcode, copy
    geo_city TEXT,                               -- GeoIP city
    geo_region TEXT,                             -- GeoIP region
    geo_country TEXT,                            -- GeoIP country code
    geo_code TEXT,                               -- GeoIP postal code
    geo_latitude REAL,                           -- GeoIP latitude
    geo_longitude REAL,                          -- GeoIP longitude
    relayed INTEGER DEFAULT 0,                   -- Was connection relayed
    secure INTEGER DEFAULT 1,                    -- Was connection secure
    live INTEGER DEFAULT 0                       -- Is live content
);

-- =============================================================================
-- session_history_metadata - Media metadata for each session
-- =============================================================================
CREATE TABLE IF NOT EXISTS session_history_metadata (
    id INTEGER PRIMARY KEY,                      -- Links to session_history.id
    rating_key INTEGER,                          -- Plex media rating key
    parent_rating_key INTEGER,                   -- Season/album rating key
    grandparent_rating_key INTEGER,              -- Show/artist rating key
    title TEXT,                                  -- Media title
    parent_title TEXT,                           -- Season name or album title
    grandparent_title TEXT,                      -- Show name or artist name
    original_title TEXT,                         -- Original language title
    full_title TEXT,                             -- Full display title
    media_index INTEGER,                         -- Episode/track number
    parent_media_index INTEGER,                  -- Season number
    section_id INTEGER,                          -- Library section ID
    library_name TEXT,                           -- Library name
    thumb TEXT,                                  -- Thumbnail URL
    parent_thumb TEXT,                           -- Season/album thumbnail
    grandparent_thumb TEXT,                      -- Show/artist thumbnail
    art TEXT,                                    -- Background art URL
    media_type TEXT,                             -- movie, episode, track
    year INTEGER,                                -- Release year
    originally_available_at TEXT,                -- Original air/release date
    added_at INTEGER,                            -- When added to library
    updated_at INTEGER,                          -- Last update time
    last_viewed_at INTEGER,                      -- Last viewed time
    content_rating TEXT,                         -- Content rating (PG, R, etc)
    summary TEXT,                                -- Plot summary
    tagline TEXT,                                -- Media tagline
    rating REAL,                                 -- Rating score
    duration INTEGER,                            -- Duration in milliseconds
    guid TEXT,                                   -- Plex GUID
    directors TEXT,                              -- Pipe-separated directors
    writers TEXT,                                -- Pipe-separated writers
    actors TEXT,                                 -- Pipe-separated actors
    genres TEXT,                                 -- Pipe-separated genres
    studio TEXT,                                 -- Studio name
    labels TEXT                                  -- Pipe-separated labels
);

-- =============================================================================
-- session_history_media_info - Stream quality and transcoding information
-- =============================================================================
CREATE TABLE IF NOT EXISTS session_history_media_info (
    id INTEGER PRIMARY KEY,                      -- Links to session_history.id
    video_codec TEXT,                            -- Source video codec
    video_codec_level TEXT,                      -- Codec level
    video_bitrate INTEGER,                       -- Source video bitrate
    video_bit_depth INTEGER,                     -- Bit depth (8, 10, 12)
    video_chroma_subsampling TEXT,               -- Chroma subsampling
    video_color_primaries TEXT,                  -- Color primaries
    video_color_range TEXT,                      -- Color range
    video_color_space TEXT,                      -- Color space
    video_color_trc TEXT,                        -- Transfer characteristics
    video_dynamic_range TEXT,                    -- SDR, HDR, etc
    video_frame_rate TEXT,                       -- Frame rate
    video_ref_frames INTEGER,                    -- Reference frames
    video_height INTEGER,                        -- Video height
    video_width INTEGER,                         -- Video width
    video_language TEXT,                         -- Video language
    video_language_code TEXT,                    -- Language code
    video_profile TEXT,                          -- Codec profile
    video_resolution TEXT,                       -- Resolution (480, 720, 1080, 2160)
    video_scan_type TEXT,                        -- progressive, interlaced
    video_full_resolution TEXT,                  -- Full resolution string (1080p)
    audio_codec TEXT,                            -- Source audio codec
    audio_bitrate INTEGER,                       -- Audio bitrate
    audio_bitrate_mode TEXT,                     -- CBR, VBR
    audio_channels INTEGER,                      -- Channel count
    audio_channel_layout TEXT,                   -- Channel layout (5.1, 7.1)
    audio_language TEXT,                         -- Audio language
    audio_language_code TEXT,                    -- Language code
    audio_profile TEXT,                          -- Audio profile
    audio_sample_rate INTEGER,                   -- Sample rate
    subtitle_codec TEXT,                         -- Subtitle codec
    subtitle_container TEXT,                     -- Subtitle container
    subtitle_format TEXT,                        -- Subtitle format
    subtitle_forced INTEGER,                     -- Is forced subtitle
    subtitle_language TEXT,                      -- Subtitle language
    subtitle_language_code TEXT,                 -- Language code
    subtitle_location TEXT,                      -- internal, external
    container TEXT,                              -- Container format (mkv, mp4)
    bitrate INTEGER,                             -- Total bitrate
    aspect_ratio TEXT,                           -- Aspect ratio
    transcode_decision TEXT,                     -- direct play, transcode, copy
    transcode_hw_requested INTEGER,              -- HW transcode requested
    transcode_hw_decode TEXT,                    -- HW decode codec
    transcode_hw_decode_title TEXT,              -- HW decode title
    transcode_hw_encode TEXT,                    -- HW encode codec
    transcode_hw_encode_title TEXT,              -- HW encode title
    transcode_hw_full_pipeline INTEGER,          -- Full HW pipeline
    video_decision TEXT,                         -- Video transcode decision
    audio_decision TEXT,                         -- Audio transcode decision
    subtitle_decision TEXT,                      -- Subtitle handling decision
    stream_container TEXT,                       -- Stream container
    stream_bitrate INTEGER,                      -- Stream bitrate
    stream_aspect_ratio TEXT,                    -- Stream aspect ratio
    stream_video_codec TEXT,                     -- Stream video codec
    stream_video_codec_level TEXT,               -- Stream codec level
    stream_video_bitrate INTEGER,                -- Stream video bitrate
    stream_video_bit_depth INTEGER,              -- Stream bit depth
    stream_video_chroma_subsampling TEXT,        -- Stream chroma
    stream_video_color_primaries TEXT,           -- Stream color primaries
    stream_video_color_range TEXT,               -- Stream color range
    stream_video_color_space TEXT,               -- Stream color space
    stream_video_color_trc TEXT,                 -- Stream transfer chars
    stream_video_dynamic_range TEXT,             -- Stream dynamic range
    stream_video_frame_rate TEXT,                -- Stream frame rate
    stream_video_ref_frames INTEGER,             -- Stream ref frames
    stream_video_height INTEGER,                 -- Stream height
    stream_video_width INTEGER,                  -- Stream width
    stream_video_language TEXT,                  -- Stream video language
    stream_video_language_code TEXT,             -- Stream language code
    stream_video_resolution TEXT,                -- Stream resolution
    stream_video_full_resolution TEXT,           -- Stream full resolution
    stream_audio_codec TEXT,                     -- Stream audio codec
    stream_audio_bitrate INTEGER,                -- Stream audio bitrate
    stream_audio_bitrate_mode TEXT,              -- Stream bitrate mode
    stream_audio_channels INTEGER,               -- Stream channels
    stream_audio_channel_layout TEXT,            -- Stream channel layout
    stream_audio_language TEXT,                  -- Stream audio language
    stream_audio_language_code TEXT,             -- Stream language code
    stream_audio_sample_rate INTEGER,            -- Stream sample rate
    stream_subtitle_codec TEXT,                  -- Stream subtitle codec
    stream_subtitle_container TEXT,              -- Stream subtitle container
    stream_subtitle_format TEXT,                 -- Stream subtitle format
    stream_subtitle_forced INTEGER,              -- Stream forced subtitle
    stream_subtitle_language TEXT,               -- Stream subtitle language
    stream_subtitle_language_code TEXT,          -- Stream language code
    stream_subtitle_location TEXT,               -- Stream subtitle location
    optimized_version INTEGER DEFAULT 0,         -- Is optimized version
    optimized_version_profile TEXT,              -- Optimization profile
    optimized_version_title TEXT,                -- Optimization title
    synced_version INTEGER DEFAULT 0,            -- Is synced version
    synced_version_profile TEXT                  -- Sync profile
);

-- =============================================================================
-- Indexes for common queries
-- =============================================================================
CREATE INDEX IF NOT EXISTS idx_session_history_started ON session_history(started);
CREATE INDEX IF NOT EXISTS idx_session_history_user_id ON session_history(user_id);
CREATE INDEX IF NOT EXISTS idx_session_history_media_type ON session_history(media_type);
CREATE INDEX IF NOT EXISTS idx_session_history_rating_key ON session_history(rating_key);

-- =============================================================================
-- Seed Data Statistics (generated database will contain):
-- - 10 users with different viewing habits
-- - 550 playback sessions across 6 months
-- - Movies, TV episodes, and music tracks
-- - Multiple platforms: Chrome, Roku, Android, iOS, Samsung, Apple TV,
--   Firefox, PlayStation, Xbox
-- - Geographic diversity: US (New York, Los Angeles, Chicago),
--   UK (London), DE (Berlin), JP (Tokyo), AU (Sydney), CA (Toronto),
--   FR (Paris), SG (Singapore)
-- =============================================================================
