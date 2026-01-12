-- =============================================================================
-- Seed Data for Documentation Screenshots
-- =============================================================================
--
-- This script generates realistic fake data for documentation screenshots.
-- It uses the DuckDB fakeit community extension to generate random but
-- realistic-looking data.
--
-- Usage:
--   duckdb /path/to/cartographus.duckdb < seed-screenshot-data.sql
--
-- The script generates:
--   - 500 playback events across 30 days
--   - 50 unique users with realistic names
--   - 100 unique IP addresses with global geographic distribution
--   - Various media types (movies, episodes, tracks)
--   - Multiple devices and platforms
--
-- =============================================================================

-- Install and load the fakeit extension
INSTALL fakeit FROM community;
LOAD fakeit;

-- Also ensure we have the required extensions loaded
LOAD spatial;
LOAD icu;

-- =============================================================================
-- Configuration
-- =============================================================================

-- Set random seed for reproducibility (same data each run)
SELECT setseed(0.42);

-- =============================================================================
-- Helper: Generate base data sets
-- =============================================================================

-- Create temporary tables for consistent reference data

-- Users (50 unique users)
CREATE TEMP TABLE temp_users AS
SELECT
    row_number() OVER () as user_id,
    fakeit_name_first() || ' ' || fakeit_name_last() as username,
    fakeit_name_first() || ' ' || fakeit_name_last() as friendly_name,
    lower(fakeit_username()) || '@' ||
        (CASE (row_number() OVER () % 5)
            WHEN 0 THEN 'gmail.com'
            WHEN 1 THEN 'yahoo.com'
            WHEN 2 THEN 'outlook.com'
            WHEN 3 THEN 'icloud.com'
            ELSE 'protonmail.com'
        END) as email,
    CASE WHEN row_number() OVER () <= 2 THEN 1 ELSE 0 END as is_admin
FROM generate_series(1, 50);

-- Media library (realistic movie/TV show titles)
CREATE TEMP TABLE temp_media AS
SELECT * FROM (VALUES
    -- Movies
    ('movie', 'The Matrix', NULL, NULL, 'Movies', 1999, 'Sci-Fi', 'Warner Bros'),
    ('movie', 'Inception', NULL, NULL, 'Movies', 2010, 'Sci-Fi', 'Warner Bros'),
    ('movie', 'The Dark Knight', NULL, NULL, 'Movies', 2008, 'Action', 'Warner Bros'),
    ('movie', 'Interstellar', NULL, NULL, 'Movies', 2014, 'Sci-Fi', 'Paramount'),
    ('movie', 'Pulp Fiction', NULL, NULL, 'Movies', 1994, 'Crime', 'Miramax'),
    ('movie', 'The Shawshank Redemption', NULL, NULL, 'Movies', 1994, 'Drama', 'Columbia'),
    ('movie', 'Fight Club', NULL, NULL, 'Movies', 1999, 'Drama', '20th Century Fox'),
    ('movie', 'Forrest Gump', NULL, NULL, 'Movies', 1994, 'Drama', 'Paramount'),
    ('movie', 'The Godfather', NULL, NULL, 'Movies', 1972, 'Crime', 'Paramount'),
    ('movie', 'Goodfellas', NULL, NULL, 'Movies', 1990, 'Crime', 'Warner Bros'),
    ('movie', 'Blade Runner 2049', NULL, NULL, 'Movies', 2017, 'Sci-Fi', 'Warner Bros'),
    ('movie', 'Dune', NULL, NULL, 'Movies', 2021, 'Sci-Fi', 'Warner Bros'),
    ('movie', 'Avatar', NULL, NULL, 'Movies', 2009, 'Sci-Fi', '20th Century Fox'),
    ('movie', 'Oppenheimer', NULL, NULL, 'Movies', 2023, 'Drama', 'Universal'),
    ('movie', 'Barbie', NULL, NULL, 'Movies', 2023, 'Comedy', 'Warner Bros'),
    -- TV Episodes
    ('episode', 'Chapter One: The Vanishing of Will Byers', 'Season 1', 'Stranger Things', 'TV Shows', 2016, 'Sci-Fi', 'Netflix'),
    ('episode', 'Chapter Two: The Weirdo on Maple Street', 'Season 1', 'Stranger Things', 'TV Shows', 2016, 'Sci-Fi', 'Netflix'),
    ('episode', 'Winter Is Coming', 'Season 1', 'Game of Thrones', 'TV Shows', 2011, 'Fantasy', 'HBO'),
    ('episode', 'The Rains of Castamere', 'Season 3', 'Game of Thrones', 'TV Shows', 2013, 'Fantasy', 'HBO'),
    ('episode', 'Battle of the Bastards', 'Season 6', 'Game of Thrones', 'TV Shows', 2016, 'Fantasy', 'HBO'),
    ('episode', 'Pilot', 'Season 1', 'Breaking Bad', 'TV Shows', 2008, 'Drama', 'AMC'),
    ('episode', 'Ozymandias', 'Season 5', 'Breaking Bad', 'TV Shows', 2013, 'Drama', 'AMC'),
    ('episode', 'The Constant', 'Season 4', 'Lost', 'TV Shows', 2008, 'Drama', 'ABC'),
    ('episode', 'Chapter 1', 'Season 1', 'The Last of Us', 'TV Shows', 2023, 'Drama', 'HBO'),
    ('episode', 'Long, Long Time', 'Season 1', 'The Last of Us', 'TV Shows', 2023, 'Drama', 'HBO'),
    ('episode', 'The Bells', 'Season 1', 'House of the Dragon', 'TV Shows', 2022, 'Fantasy', 'HBO'),
    ('episode', 'Pilot', 'Season 1', 'The Office', 'TV Shows', 2005, 'Comedy', 'NBC'),
    ('episode', 'Dinner Party', 'Season 4', 'The Office', 'TV Shows', 2008, 'Comedy', 'NBC'),
    -- Music tracks
    ('track', 'Bohemian Rhapsody', 'A Night at the Opera', 'Queen', 'Music', 1975, 'Rock', 'EMI'),
    ('track', 'Stairway to Heaven', 'Led Zeppelin IV', 'Led Zeppelin', 'Music', 1971, 'Rock', 'Atlantic'),
    ('track', 'Hotel California', 'Hotel California', 'Eagles', 'Music', 1977, 'Rock', 'Asylum'),
    ('track', 'Blinding Lights', 'After Hours', 'The Weeknd', 'Music', 2020, 'Pop', 'Republic'),
    ('track', 'As It Was', 'Harrys House', 'Harry Styles', 'Music', 2022, 'Pop', 'Columbia')
) AS t(media_type, title, parent_title, grandparent_title, library_name, year, genres, studio);

-- Platforms and devices
CREATE TEMP TABLE temp_platforms AS
SELECT * FROM (VALUES
    ('Plex Web', 'Chrome', 'Web', '4.0', 'Chrome'),
    ('Plex Web', 'Firefox', 'Web', '4.0', 'Firefox'),
    ('Plex Web', 'Safari', 'Web', '4.0', 'Safari'),
    ('Plex for iOS', 'iPhone', 'iOS', '8.0', 'iPhone 14 Pro'),
    ('Plex for iOS', 'iPad', 'iOS', '8.0', 'iPad Pro'),
    ('Plex for Android', 'Android', 'Android', '9.0', 'Samsung Galaxy S23'),
    ('Plex for Android', 'Android TV', 'Android TV', '9.0', 'NVIDIA Shield'),
    ('Plex for Roku', 'Roku', 'Roku', '6.0', 'Roku Ultra'),
    ('Plex for Apple TV', 'Apple TV', 'tvOS', '8.0', 'Apple TV 4K'),
    ('Plex for Smart TV', 'Samsung TV', 'Tizen', '5.0', 'Samsung QN90A'),
    ('Plex for Smart TV', 'LG TV', 'webOS', '5.0', 'LG C2 OLED'),
    ('Plex HTPC', 'Windows', 'Windows', '1.50', 'Desktop PC'),
    ('Plex for macOS', 'macOS', 'macOS', '1.50', 'MacBook Pro')
) AS t(product, platform, platform_name, product_version, device);

-- Geographic locations (global distribution with weighted cities)
CREATE TEMP TABLE temp_locations AS
SELECT * FROM (VALUES
    -- North America (40%)
    ('New York', 'New York', 'United States', 40.7128, -74.0060, 'America/New_York'),
    ('Los Angeles', 'California', 'United States', 34.0522, -118.2437, 'America/Los_Angeles'),
    ('Chicago', 'Illinois', 'United States', 41.8781, -87.6298, 'America/Chicago'),
    ('Houston', 'Texas', 'United States', 29.7604, -95.3698, 'America/Chicago'),
    ('Phoenix', 'Arizona', 'United States', 33.4484, -112.0740, 'America/Phoenix'),
    ('San Francisco', 'California', 'United States', 37.7749, -122.4194, 'America/Los_Angeles'),
    ('Seattle', 'Washington', 'United States', 47.6062, -122.3321, 'America/Los_Angeles'),
    ('Denver', 'Colorado', 'United States', 39.7392, -104.9903, 'America/Denver'),
    ('Toronto', 'Ontario', 'Canada', 43.6532, -79.3832, 'America/Toronto'),
    ('Vancouver', 'British Columbia', 'Canada', 49.2827, -123.1207, 'America/Vancouver'),
    -- Europe (30%)
    ('London', 'England', 'United Kingdom', 51.5074, -0.1278, 'Europe/London'),
    ('Paris', 'Ile-de-France', 'France', 48.8566, 2.3522, 'Europe/Paris'),
    ('Berlin', 'Berlin', 'Germany', 52.5200, 13.4050, 'Europe/Berlin'),
    ('Amsterdam', 'North Holland', 'Netherlands', 52.3676, 4.9041, 'Europe/Amsterdam'),
    ('Madrid', 'Madrid', 'Spain', 40.4168, -3.7038, 'Europe/Madrid'),
    ('Rome', 'Lazio', 'Italy', 41.9028, 12.4964, 'Europe/Rome'),
    ('Stockholm', 'Stockholm', 'Sweden', 59.3293, 18.0686, 'Europe/Stockholm'),
    -- Asia Pacific (20%)
    ('Tokyo', 'Tokyo', 'Japan', 35.6762, 139.6503, 'Asia/Tokyo'),
    ('Sydney', 'New South Wales', 'Australia', -33.8688, 151.2093, 'Australia/Sydney'),
    ('Singapore', 'Singapore', 'Singapore', 1.3521, 103.8198, 'Asia/Singapore'),
    ('Hong Kong', 'Hong Kong', 'Hong Kong', 22.3193, 114.1694, 'Asia/Hong_Kong'),
    ('Seoul', 'Seoul', 'South Korea', 37.5665, 126.9780, 'Asia/Seoul'),
    ('Melbourne', 'Victoria', 'Australia', -37.8136, 144.9631, 'Australia/Melbourne'),
    -- South America (5%)
    ('Sao Paulo', 'Sao Paulo', 'Brazil', -23.5505, -46.6333, 'America/Sao_Paulo'),
    ('Buenos Aires', 'Buenos Aires', 'Argentina', -34.6037, -58.3816, 'America/Argentina/Buenos_Aires'),
    -- Other (5%)
    ('Dubai', 'Dubai', 'United Arab Emirates', 25.2048, 55.2708, 'Asia/Dubai'),
    ('Mumbai', 'Maharashtra', 'India', 19.0760, 72.8777, 'Asia/Kolkata'),
    ('Cape Town', 'Western Cape', 'South Africa', -33.9249, 18.4241, 'Africa/Johannesburg')
) AS t(city, region, country, latitude, longitude, timezone);

-- =============================================================================
-- Generate IP addresses with locations
-- =============================================================================

-- Insert geolocations with fake IP addresses
INSERT INTO geolocations (ip_address, latitude, longitude, city, region, country, timezone, accuracy_radius)
SELECT
    -- Generate realistic-looking IP addresses
    CASE
        WHEN country = 'United States' THEN
            (50 + (row_number() OVER () % 50))::TEXT || '.' ||
            ((row_number() OVER () * 7) % 256)::TEXT || '.' ||
            ((row_number() OVER () * 13) % 256)::TEXT || '.' ||
            ((row_number() OVER () * 17) % 256)::TEXT
        WHEN country = 'United Kingdom' THEN
            '82.' || ((row_number() OVER () * 11) % 256)::TEXT || '.' ||
            ((row_number() OVER () * 23) % 256)::TEXT || '.' ||
            ((row_number() OVER () * 29) % 256)::TEXT
        WHEN country = 'Germany' THEN
            '85.' || ((row_number() OVER () * 31) % 256)::TEXT || '.' ||
            ((row_number() OVER () * 37) % 256)::TEXT || '.' ||
            ((row_number() OVER () * 41) % 256)::TEXT
        ELSE
            ((row_number() OVER () * 3 + 100) % 200 + 50)::TEXT || '.' ||
            ((row_number() OVER () * 7) % 256)::TEXT || '.' ||
            ((row_number() OVER () * 11) % 256)::TEXT || '.' ||
            ((row_number() OVER () * 13) % 256)::TEXT
    END as ip_address,
    -- Add small random offset to coordinates for variety
    latitude + (random() - 0.5) * 0.5 as latitude,
    longitude + (random() - 0.5) * 0.5 as longitude,
    city,
    region,
    country,
    timezone,
    CASE
        WHEN country IN ('United States', 'United Kingdom', 'Germany', 'Japan') THEN 10
        ELSE 50
    END as accuracy_radius
FROM temp_locations
CROSS JOIN generate_series(1, 4) as multiplier  -- 4 IPs per location = ~100 total
ON CONFLICT (ip_address) DO NOTHING;

-- =============================================================================
-- Generate playback events
-- =============================================================================

INSERT INTO playback_events (
    id,
    session_key,
    started_at,
    stopped_at,
    user_id,
    username,
    ip_address,
    media_type,
    title,
    parent_title,
    grandparent_title,
    platform,
    player,
    location_type,
    percent_complete,
    paused_counter,
    library_name,
    year,
    genres,
    studio,
    friendly_name,
    email,
    is_admin,
    product,
    platform_name,
    product_version,
    device,
    video_resolution,
    video_codec,
    video_bitrate,
    transcode_decision,
    bandwidth,
    local,
    source,
    rating_key
)
SELECT
    -- Generate UUID
    uuid() as id,
    -- Session key
    'session_' || (row_number() OVER ())::TEXT as session_key,
    -- Started at: Random time in last 30 days, weighted towards evenings
    (CURRENT_TIMESTAMP - INTERVAL ((random() * 30)::INTEGER) DAY
     - INTERVAL ((random() * 24)::INTEGER) HOUR
     + INTERVAL (CASE
         WHEN random() < 0.6 THEN (18 + (random() * 6)::INTEGER)  -- 60% evening (6pm-midnight)
         WHEN random() < 0.8 THEN (12 + (random() * 6)::INTEGER)  -- 20% afternoon
         ELSE (random() * 12)::INTEGER  -- 20% morning
       END) HOUR
    ) as started_at,
    -- Stopped at: 30-180 minutes after start
    (CURRENT_TIMESTAMP - INTERVAL ((random() * 30)::INTEGER) DAY
     - INTERVAL ((random() * 24)::INTEGER) HOUR
     + INTERVAL (CASE
         WHEN random() < 0.6 THEN (18 + (random() * 6)::INTEGER)
         WHEN random() < 0.8 THEN (12 + (random() * 6)::INTEGER)
         ELSE (random() * 12)::INTEGER
       END) HOUR
     + INTERVAL ((30 + (random() * 150)::INTEGER)) MINUTE
    ) as stopped_at,
    -- User (weighted: some users watch more)
    u.user_id,
    u.username,
    -- IP address from geolocations
    g.ip_address,
    -- Media
    m.media_type,
    m.title,
    m.parent_title,
    m.grandparent_title,
    -- Platform
    p.platform,
    p.platform,
    -- Location type
    CASE WHEN random() < 0.7 THEN 'wan' ELSE 'lan' END as location_type,
    -- Percent complete (most finish, some abandon)
    CASE
        WHEN random() < 0.7 THEN 90 + (random() * 10)::INTEGER  -- 70% complete
        WHEN random() < 0.9 THEN 50 + (random() * 40)::INTEGER  -- 20% half-watched
        ELSE (random() * 50)::INTEGER  -- 10% abandoned early
    END as percent_complete,
    -- Paused counter
    (random() * 5)::INTEGER as paused_counter,
    -- Library
    m.library_name,
    m.year,
    m.genres,
    m.studio,
    -- User info
    u.friendly_name,
    u.email,
    u.is_admin,
    -- Platform details
    p.product,
    p.platform_name,
    p.product_version,
    p.device,
    -- Video quality (weighted towards HD)
    CASE
        WHEN random() < 0.4 THEN '4k'
        WHEN random() < 0.8 THEN '1080'
        WHEN random() < 0.95 THEN '720'
        ELSE '480'
    END as video_resolution,
    CASE WHEN random() < 0.7 THEN 'h264' ELSE 'hevc' END as video_codec,
    -- Bitrate (varies by resolution)
    CASE
        WHEN random() < 0.4 THEN 15000 + (random() * 10000)::INTEGER  -- 4k
        WHEN random() < 0.8 THEN 8000 + (random() * 4000)::INTEGER   -- 1080p
        ELSE 3000 + (random() * 3000)::INTEGER  -- 720p/480p
    END as video_bitrate,
    -- Transcode decision
    CASE
        WHEN random() < 0.6 THEN 'direct play'
        WHEN random() < 0.9 THEN 'transcode'
        ELSE 'copy'
    END as transcode_decision,
    -- Bandwidth
    (5000 + (random() * 20000)::INTEGER) as bandwidth,
    -- Local (30% local, 70% remote)
    CASE WHEN random() < 0.3 THEN 1 ELSE 0 END as local,
    -- Source
    'tautulli' as source,
    -- Rating key
    'rk_' || ((random() * 10000)::INTEGER)::TEXT as rating_key
FROM
    generate_series(1, 500) as event_num
    -- Join to get random user (some users more active)
    CROSS JOIN LATERAL (
        SELECT * FROM temp_users
        ORDER BY random() * (1.0 / (user_id * 0.1 + 1))  -- Weight towards lower user IDs
        LIMIT 1
    ) u
    -- Join to get random media
    CROSS JOIN LATERAL (
        SELECT * FROM temp_media ORDER BY random() LIMIT 1
    ) m
    -- Join to get random platform
    CROSS JOIN LATERAL (
        SELECT * FROM temp_platforms ORDER BY random() LIMIT 1
    ) p
    -- Join to get random geolocation
    CROSS JOIN LATERAL (
        SELECT * FROM geolocations ORDER BY random() LIMIT 1
    ) g;

-- =============================================================================
-- Cleanup temporary tables
-- =============================================================================

DROP TABLE temp_users;
DROP TABLE temp_media;
DROP TABLE temp_platforms;
DROP TABLE temp_locations;

-- =============================================================================
-- Verify data generation
-- =============================================================================

SELECT 'Playback Events' as table_name, COUNT(*) as row_count FROM playback_events
UNION ALL
SELECT 'Geolocations', COUNT(*) FROM geolocations
UNION ALL
SELECT 'Unique Users', COUNT(DISTINCT user_id) FROM playback_events
UNION ALL
SELECT 'Unique IPs', COUNT(DISTINCT ip_address) FROM playback_events
UNION ALL
SELECT 'Media Types', COUNT(DISTINCT media_type) FROM playback_events;

-- Show sample data
SELECT 'Sample playback events:' as info;
SELECT
    username,
    title,
    media_type,
    platform,
    video_resolution,
    started_at::DATE as date
FROM playback_events
ORDER BY started_at DESC
LIMIT 10;

SELECT 'Sample geographic distribution:' as info;
SELECT
    g.country,
    g.city,
    COUNT(*) as playbacks
FROM playback_events p
JOIN geolocations g ON p.ip_address = g.ip_address
GROUP BY g.country, g.city
ORDER BY playbacks DESC
LIMIT 10;
