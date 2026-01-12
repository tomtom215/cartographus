// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
database_schema.go - Database Schema Management

This file manages the DuckDB database schema including table creation
and index management for optimal query performance.

Tables:
  - playback_events: Core table storing all Plex/Jellyfin/Emby/Tautulli playback activity
    (203 columns covering media, user, stream, transcode, and metadata)
  - geolocations: IP geolocation data with optional GEOMETRY column for spatial queries
  - user_mappings: Cross-platform user ID mapping for multi-server support
  - failed_events: Dead letter queue for events that failed processing
  - dedupe_audit_log: Audit trail for deduplication decisions

Schema Strategy (Pre-Release):
All columns are defined in the initial CREATE TABLE statement. This provides:
  - Single source of truth for the complete schema
  - Faster startup (no migrations to run)
  - Cleaner codebase

Post-Release Migration Strategy:
After the first public release with real users, use versioned migrations in
migrations.go to add new columns without losing existing data. See CLAUDE.md
for details on when and how to re-enable migrations.

Index Strategy:
Indexes are created for:
  - Frequently filtered columns (user_id, started_at, media_type, etc.)
  - Composite indexes for common query patterns
  - Binge detection queries (user + show + episode ordering)
  - Bitrate analytics and hardware transcode tracking
  - Deduplication (rating_key + user_id + started_at, correlation_key)
*/

//nolint:staticcheck // File documentation, not package doc
package database

import (
	"context"
	"fmt"
	"time"
)

// schemaContext returns a context with timeout for schema operations
func schemaContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 60*time.Second)
}

// createTables creates the core database tables
func (db *DB) createTables() error {
	ctx, cancel := schemaContext()
	defer cancel()

	queries := db.getTableCreationQueries()

	for _, query := range queries {
		if _, err := db.conn.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query: %s: %w", query, err)
		}
	}

	return nil
}

// getTableCreationQueries returns the table creation SQL statements
func (db *DB) getTableCreationQueries() []string {
	queries := []string{
		// Playback events table - Complete schema with all 203 columns
		// Organized into logical groups for maintainability
		`CREATE TABLE IF NOT EXISTS playback_events (
			-- ============================================
			-- Core Fields (16 columns)
			-- ============================================
			id UUID PRIMARY KEY,
			session_key TEXT NOT NULL,
			started_at TIMESTAMP NOT NULL,
			stopped_at TIMESTAMP,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			ip_address TEXT NOT NULL,
			media_type TEXT NOT NULL,
			title TEXT NOT NULL,
			parent_title TEXT,
			grandparent_title TEXT,
			platform TEXT,
			player TEXT,
			location_type TEXT,
			percent_complete INTEGER,
			paused_counter INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

			-- ============================================
			-- Media Metadata (25 columns)
			-- ============================================
			section_id INTEGER,
			library_name TEXT,
			content_rating TEXT,
			year INTEGER,
			rating_key TEXT,
			parent_rating_key TEXT,
			grandparent_rating_key TEXT,
			media_index INTEGER,
			parent_media_index INTEGER,
			guid TEXT,
			parent_guid TEXT,
			grandparent_guid TEXT,
			original_title TEXT,
			full_title TEXT,
			sort_title TEXT,
			originally_available_at TEXT,
			watched_status INTEGER,
			thumb TEXT,
			parent_thumb TEXT,
			grandparent_thumb TEXT,
			art TEXT,
			grandparent_art TEXT,
			banner TEXT,
			file TEXT,
			file_size BIGINT,

			-- ============================================
			-- Content Details (15 columns)
			-- ============================================
			directors TEXT,
			writers TEXT,
			actors TEXT,
			genres TEXT,
			studio TEXT,
			summary TEXT,
			tagline TEXT,
			rating TEXT,
			audience_rating TEXT,
			user_rating TEXT,
			labels TEXT,
			collections TEXT,
			added_at TEXT,
			updated_at TEXT,
			last_viewed_at TEXT,

			-- ============================================
			-- User Information (15 columns)
			-- ============================================
			friendly_name TEXT,
			user_thumb TEXT,
			email TEXT,
			is_admin INTEGER,
			is_home_user INTEGER,
			is_allow_sync INTEGER,
			is_restricted INTEGER,
			keep_history INTEGER,
			deleted_user INTEGER,
			do_notify INTEGER,
			allow_guest INTEGER,
			shared_libraries TEXT,
			ip_address_public TEXT,
			location TEXT,

			-- ============================================
			-- Device/Platform Information (10 columns)
			-- ============================================
			platform_name TEXT,
			platform_version TEXT,
			product TEXT,
			product_version TEXT,
			device TEXT,
			machine_id TEXT,
			quality_profile TEXT,
			optimized_version INTEGER,
			synced_version INTEGER,
			plex_key TEXT,

			-- ============================================
			-- Video Source Properties (20 columns)
			-- ============================================
			video_resolution TEXT,
			video_full_resolution TEXT,
			video_codec TEXT,
			video_codec_level TEXT,
			video_profile TEXT,
			video_bitrate INTEGER,
			video_bit_depth INTEGER,
			video_width INTEGER,
			video_height INTEGER,
			video_framerate TEXT,
			video_dynamic_range TEXT,
			video_scan_type TEXT,
			video_language TEXT,
			video_language_code TEXT,
			video_ref_frames INTEGER,
			video_color_primaries TEXT,
			video_color_range TEXT,
			video_color_space TEXT,
			video_color_trc TEXT,
			video_chroma_subsampling TEXT,
			aspect_ratio TEXT,
			container TEXT,
			bitrate INTEGER,

			-- ============================================
			-- Audio Source Properties (12 columns)
			-- ============================================
			audio_codec TEXT,
			audio_profile TEXT,
			audio_channels TEXT,
			audio_channel_layout TEXT,
			audio_bitrate INTEGER,
			audio_bitrate_mode TEXT,
			audio_sample_rate INTEGER,
			audio_language TEXT,
			audio_language_code TEXT,

			-- ============================================
			-- Subtitle Source Properties (8 columns)
			-- ============================================
			subtitle_codec TEXT,
			subtitle_language TEXT,
			subtitle_language_code TEXT,
			subtitle_container TEXT,
			subtitle_container_fmt TEXT,
			subtitle_format TEXT,
			subtitle_forced INTEGER,
			subtitle_location TEXT,
			subtitles INTEGER,

			-- ============================================
			-- Stream Video Properties (15 columns)
			-- ============================================
			stream_video_resolution TEXT,
			stream_video_full_resolution TEXT,
			stream_video_codec TEXT,
			stream_video_codec_level TEXT,
			stream_video_profile TEXT,
			stream_video_bitrate INTEGER,
			stream_video_width INTEGER,
			stream_video_height INTEGER,
			stream_video_bit_depth INTEGER,
			stream_video_framerate TEXT,
			stream_video_dynamic_range TEXT,
			stream_video_scan_type TEXT,
			stream_video_language TEXT,
			stream_video_language_code TEXT,
			stream_video_decision TEXT,
			stream_container TEXT,
			stream_bitrate INTEGER,
			stream_aspect_ratio TEXT,

			-- ============================================
			-- Stream Audio Properties (12 columns)
			-- ============================================
			stream_audio_codec TEXT,
			stream_audio_profile TEXT,
			stream_audio_channels TEXT,
			stream_audio_channel_layout TEXT,
			stream_audio_bitrate INTEGER,
			stream_audio_bitrate_mode TEXT,
			stream_audio_sample_rate INTEGER,
			stream_audio_language TEXT,
			stream_audio_language_code TEXT,
			stream_audio_decision TEXT,

			-- ============================================
			-- Stream Subtitle Properties (8 columns)
			-- ============================================
			stream_subtitle_codec TEXT,
			stream_subtitle_language TEXT,
			stream_subtitle_language_code TEXT,
			stream_subtitle_container TEXT,
			stream_subtitle_format TEXT,
			stream_subtitle_forced INTEGER,
			stream_subtitle_location TEXT,
			stream_subtitle_decision TEXT,

			-- ============================================
			-- Transcode Properties (22 columns)
			-- ============================================
			transcode_decision TEXT,
			video_decision TEXT,
			audio_decision TEXT,
			subtitle_decision TEXT,
			transcode_key TEXT,
			transcode_throttled INTEGER,
			throttled INTEGER,
			transcode_progress INTEGER,
			transcode_speed TEXT,
			transcode_container TEXT,
			transcode_video_codec TEXT,
			transcode_video_width INTEGER,
			transcode_video_height INTEGER,
			transcode_audio_codec TEXT,
			transcode_audio_channels INTEGER,
			transcode_bitrate INTEGER,
			transcode_hw_requested INTEGER,
			transcode_hw_decoding INTEGER,
			transcode_hw_encoding INTEGER,
			transcode_hw_full_pipeline INTEGER,
			transcode_hw_decode TEXT,
			transcode_hw_decode_title TEXT,
			transcode_hw_encode TEXT,
			transcode_hw_encode_title TEXT,

			-- ============================================
			-- Network/Bandwidth Properties (10 columns)
			-- ============================================
			secure INTEGER,
			relayed INTEGER,
			relay INTEGER,
			local INTEGER,
			bandwidth INTEGER,
			bandwidth_lan INTEGER,
			bandwidth_wan INTEGER,
			source_bitrate INTEGER,
			network_bandwidth INTEGER,

			-- ============================================
			-- Live TV Properties (6 columns)
			-- ============================================
			live INTEGER,
			live_uuid TEXT,
			channel_stream INTEGER,
			channel_call_sign TEXT,
			channel_identifier TEXT,

			-- ============================================
			-- Grouping Properties (3 columns)
			-- ============================================
			group_count INTEGER,
			group_ids TEXT,
			state TEXT,

			-- ============================================
			-- Multi-Source/Server Properties (5 columns)
			-- ============================================
			source TEXT DEFAULT 'tautulli',
			server_id TEXT,
			correlation_key TEXT,
			transaction_id TEXT,
			play_duration INTEGER
		);`,
	}

	// Geolocations table (varies based on spatial availability)
	if db.spatialAvailable {
		queries = append(queries, `CREATE TABLE IF NOT EXISTS geolocations (
			ip_address TEXT PRIMARY KEY,
			latitude DOUBLE NOT NULL,
			longitude DOUBLE NOT NULL,
			geom GEOMETRY NOT NULL,
			city TEXT,
			region TEXT,
			country TEXT NOT NULL,
			postal_code TEXT,
			timezone TEXT,
			accuracy_radius INTEGER,
			last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`)
	} else {
		queries = append(queries, `CREATE TABLE IF NOT EXISTS geolocations (
			ip_address TEXT PRIMARY KEY,
			latitude DOUBLE NOT NULL,
			longitude DOUBLE NOT NULL,
			city TEXT,
			region TEXT,
			country TEXT NOT NULL,
			postal_code TEXT,
			timezone TEXT,
			accuracy_radius INTEGER,
			last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`)
	}

	// User mappings table (v2.0 - Phase 0.7: Multi-source user ID mapping)
	// Maps external user IDs (Jellyfin/Emby UUIDs, Plex user IDs) to internal integer IDs.
	// This enables consistent user tracking across all media server sources and
	// allows correlating the same person across different platforms.
	// Note: ID is managed manually (MAX(id)+1) since DuckDB doesn't support IDENTITY with PRIMARY KEY
	queries = append(queries, `CREATE TABLE IF NOT EXISTS user_mappings (
		id INTEGER PRIMARY KEY,
		source TEXT NOT NULL,
		server_id TEXT NOT NULL,
		external_user_id TEXT NOT NULL,
		internal_user_id INTEGER NOT NULL,
		username TEXT,
		friendly_name TEXT,
		email TEXT,
		user_thumb TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(source, server_id, external_user_id)
	);`)

	// Failed events table (v2.1 - ADR-0023: Consumer-Side WAL for Exactly-Once Delivery)
	// Persistent DLQ for events that failed to be inserted into DuckDB after max retries.
	// Stores full event payload for manual investigation and recovery.
	queries = append(queries, `CREATE TABLE IF NOT EXISTS failed_events (
		id UUID PRIMARY KEY,

		-- Original event data
		transaction_id TEXT NOT NULL,
		event_id TEXT NOT NULL,
		session_key TEXT,
		correlation_key TEXT,
		source TEXT NOT NULL,
		event_payload JSON NOT NULL,

		-- Failure details
		failed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		failure_reason TEXT NOT NULL,
		failure_layer TEXT NOT NULL,
		last_error TEXT,

		-- Retry tracking
		retry_count INTEGER NOT NULL DEFAULT 0,
		last_retry_at TIMESTAMPTZ,
		max_retries_exceeded BOOLEAN DEFAULT FALSE,

		-- Resolution
		status TEXT NOT NULL DEFAULT 'pending',
		resolved_at TIMESTAMPTZ,
		resolved_by TEXT,
		resolution_notes TEXT,

		-- Audit
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`)

	// Wrapped reports table (v2.3 - Annual Wrapped Reports)
	// Caches generated annual wrapped reports for users.
	// Reports are generated on-demand and cached with share tokens for social sharing.
	// JSON columns store complex nested data (arrays, objects) that would be inefficient as normalized tables.
	queries = append(queries, `CREATE TABLE IF NOT EXISTS wrapped_reports (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		username TEXT NOT NULL,
		year INTEGER NOT NULL,

		-- Core statistics
		total_watch_time_hours DOUBLE NOT NULL,
		total_playbacks INTEGER NOT NULL,
		unique_content_count INTEGER NOT NULL,
		completion_rate DOUBLE NOT NULL,
		days_active INTEGER NOT NULL,
		longest_streak_days INTEGER NOT NULL,
		avg_daily_watch_minutes DOUBLE NOT NULL,

		-- Binge statistics
		binge_sessions INTEGER NOT NULL,
		total_binge_hours DOUBLE NOT NULL,
		favorite_binge_show TEXT,
		avg_binge_episodes DOUBLE NOT NULL,
		longest_binge_json JSON,

		-- Quality metrics
		avg_bitrate_mbps DOUBLE NOT NULL,
		direct_play_rate DOUBLE NOT NULL,
		hdr_viewing_percent DOUBLE NOT NULL,
		four_k_viewing_percent DOUBLE NOT NULL,
		preferred_platform TEXT,
		preferred_player TEXT,

		-- Discovery metrics
		new_content_count INTEGER NOT NULL,
		discovery_rate DOUBLE NOT NULL,
		first_watch_of_year TEXT,
		last_watch_of_year TEXT,

		-- Viewing patterns (stored as JSON arrays)
		peak_hour INTEGER NOT NULL,
		peak_day TEXT NOT NULL,
		peak_month TEXT NOT NULL,
		viewing_by_hour JSON NOT NULL,
		viewing_by_day JSON NOT NULL,
		viewing_by_month JSON NOT NULL,
		monthly_trends JSON NOT NULL,

		-- Ranked content (stored as JSON arrays)
		top_movies JSON NOT NULL,
		top_shows JSON NOT NULL,
		top_episodes JSON,
		top_genres JSON NOT NULL,
		top_actors JSON,
		top_directors JSON,

		-- Achievements and percentiles
		achievements JSON NOT NULL,
		percentiles JSON NOT NULL,

		-- Sharing
		share_token TEXT UNIQUE,
		shareable_text TEXT,

		-- Timestamps
		generated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

		-- Unique constraint: one report per user per year
		UNIQUE(user_id, year)
	);`)

	// Dedupe audit log table (v2.2 - ADR-0022: Deduplication Audit and Management)
	// Records all deduplication decisions for visibility, troubleshooting, and recovery.
	// Stores full event payload to enable restoration of incorrectly deduplicated events.
	queries = append(queries, `CREATE TABLE IF NOT EXISTS dedupe_audit_log (
		id UUID PRIMARY KEY,
		timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

		-- The event that was deduplicated (discarded)
		discarded_event_id TEXT NOT NULL,
		discarded_session_key TEXT,
		discarded_correlation_key TEXT,
		discarded_source TEXT NOT NULL,
		discarded_started_at TIMESTAMPTZ,
		discarded_raw_payload JSON,

		-- The event that it was matched against (kept)
		matched_event_id TEXT,
		matched_session_key TEXT,
		matched_correlation_key TEXT,
		matched_source TEXT,

		-- Deduplication details
		dedupe_reason TEXT NOT NULL,
		dedupe_layer TEXT NOT NULL,
		similarity_score DOUBLE,

		-- User information
		user_id INTEGER NOT NULL,
		username TEXT,

		-- Media information
		media_type TEXT,
		title TEXT,
		rating_key TEXT,

		-- Resolution status
		status TEXT NOT NULL DEFAULT 'auto_dedupe',
		resolved_by TEXT,
		resolved_at TIMESTAMPTZ,
		resolution_notes TEXT,

		-- Audit timestamps
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`)

	// User roles table (v2.4 - RBAC Implementation)
	// Stores persistent role assignments for users.
	// Roles determine authorization levels: viewer (default), editor, admin.
	// Note: ID is managed manually (MAX(id)+1) since DuckDB doesn't support IDENTITY with PRIMARY KEY
	queries = append(queries, `CREATE TABLE IF NOT EXISTS user_roles (
		id INTEGER PRIMARY KEY,
		user_id TEXT NOT NULL,
		username TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'viewer',
		assigned_by TEXT,
		assigned_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMPTZ,
		is_active BOOLEAN DEFAULT TRUE,
		metadata JSON,
		UNIQUE(user_id, role)
	);`)

	// Role audit log table (v2.4 - RBAC Implementation)
	// Records all role changes for security auditing and compliance.
	// Immutable append-only log of role assignments, revocations, and updates.
	queries = append(queries, `CREATE TABLE IF NOT EXISTS role_audit_log (
		id UUID PRIMARY KEY,
		timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		actor_id TEXT NOT NULL,
		actor_username TEXT,
		action TEXT NOT NULL,
		target_user_id TEXT NOT NULL,
		target_username TEXT,
		old_role TEXT,
		new_role TEXT,
		reason TEXT,
		ip_address TEXT,
		user_agent TEXT,
		session_id TEXT
	);`)

	// Personal access tokens table (v2.5 - PAT System)
	// Stores personal access tokens for programmatic API access (similar to GitHub PATs).
	// Tokens are hashed with bcrypt, never stored in plaintext.
	// Supports scoped permissions, expiration, and IP allowlisting.
	queries = append(queries, `CREATE TABLE IF NOT EXISTS personal_access_tokens (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		username TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		token_prefix TEXT NOT NULL,
		token_hash TEXT NOT NULL,
		scopes JSON NOT NULL,
		expires_at TIMESTAMPTZ,
		last_used_at TIMESTAMPTZ,
		last_used_ip TEXT,
		use_count INTEGER NOT NULL DEFAULT 0,
		ip_allowlist JSON,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		revoked_at TIMESTAMPTZ,
		revoked_by TEXT,
		revoke_reason TEXT
	);`)

	// PAT usage audit log table (v2.5 - PAT System)
	// Records PAT usage for security monitoring and troubleshooting.
	// Immutable append-only log of token usage events.
	queries = append(queries, `CREATE TABLE IF NOT EXISTS pat_usage_log (
		id UUID PRIMARY KEY,
		timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		token_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		action TEXT NOT NULL,
		endpoint TEXT,
		method TEXT,
		ip_address TEXT,
		user_agent TEXT,
		success BOOLEAN NOT NULL,
		error_code TEXT,
		response_time_ms INTEGER
	);`)

	// Newsletter templates table (v2.6 - Newsletter Generator)
	// Stores reusable templates for newsletter content with HTML/text bodies.
	// Templates support variable substitution and versioning for audit tracking.
	queries = append(queries, `CREATE TABLE IF NOT EXISTS newsletter_templates (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		type TEXT NOT NULL,
		subject TEXT NOT NULL,
		body_html TEXT NOT NULL,
		body_text TEXT,
		variables JSON,
		default_config JSON,
		version INTEGER NOT NULL DEFAULT 1,
		is_built_in BOOLEAN NOT NULL DEFAULT FALSE,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_by TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_by TEXT,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`)

	// Newsletter schedules table (v2.6 - Newsletter Generator)
	// Stores scheduled newsletter delivery configurations with cron expressions.
	// Links templates to recipients with channel-specific delivery settings.
	queries = append(queries, `CREATE TABLE IF NOT EXISTS newsletter_schedules (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		template_id TEXT NOT NULL,
		recipients JSON NOT NULL,
		cron_expression TEXT NOT NULL,
		timezone TEXT NOT NULL DEFAULT 'UTC',
		config JSON,
		channels JSON NOT NULL,
		channel_configs JSON,
		is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
		last_run_at TIMESTAMPTZ,
		next_run_at TIMESTAMPTZ,
		last_run_status TEXT,
		run_count INTEGER NOT NULL DEFAULT 0,
		success_count INTEGER NOT NULL DEFAULT 0,
		failure_count INTEGER NOT NULL DEFAULT 0,
		created_by TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_by TEXT,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`)

	// Newsletter deliveries table (v2.6 - Newsletter Generator)
	// Records all newsletter delivery attempts for audit and analytics.
	// Stores per-recipient delivery results and content statistics.
	queries = append(queries, `CREATE TABLE IF NOT EXISTS newsletter_deliveries (
		id TEXT PRIMARY KEY,
		schedule_id TEXT,
		template_id TEXT NOT NULL,
		template_version INTEGER NOT NULL,
		channel TEXT NOT NULL,
		status TEXT NOT NULL,
		recipients_total INTEGER NOT NULL DEFAULT 0,
		recipients_delivered INTEGER NOT NULL DEFAULT 0,
		recipients_failed INTEGER NOT NULL DEFAULT 0,
		recipient_details JSON,
		content_summary TEXT,
		content_stats JSON,
		rendered_subject TEXT,
		rendered_body_size INTEGER,
		started_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMPTZ,
		duration_ms BIGINT,
		error_message TEXT,
		error_details JSON,
		triggered_by TEXT NOT NULL DEFAULT 'schedule',
		triggered_by_user_id TEXT
	);`)

	// Newsletter user preferences table (v2.6 - Newsletter Generator)
	// Stores per-user newsletter preferences including opt-out status.
	// Allows users to customize their newsletter experience.
	queries = append(queries, `CREATE TABLE IF NOT EXISTS newsletter_user_preferences (
		user_id TEXT PRIMARY KEY,
		username TEXT NOT NULL,
		global_opt_out BOOLEAN NOT NULL DEFAULT FALSE,
		global_opt_out_at TIMESTAMPTZ,
		schedule_preferences JSON,
		preferred_channel TEXT,
		preferred_email TEXT,
		language TEXT,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`)

	// Newsletter audit log table (v2.6 - Newsletter Generator)
	// Records all newsletter-related actions for security auditing.
	// Immutable append-only log of template, schedule, and delivery operations.
	queries = append(queries, `CREATE TABLE IF NOT EXISTS newsletter_audit_log (
		id TEXT PRIMARY KEY,
		timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		actor_id TEXT NOT NULL,
		actor_username TEXT,
		action TEXT NOT NULL,
		resource_type TEXT NOT NULL,
		resource_id TEXT NOT NULL,
		resource_name TEXT,
		details JSON,
		ip_address TEXT,
		user_agent TEXT
	);`)

	// Media servers table (v2.7 - ADR-0026: Multi-Server Management UI)
	// Stores additional media server configurations added via UI (beyond env vars).
	// Credentials (URL, token) are encrypted with AES-256-GCM using key derived from JWT_SECRET.
	// Server configurations from env vars are NOT stored here (managed by config.go).
	queries = append(queries, `CREATE TABLE IF NOT EXISTS media_servers (
		id TEXT PRIMARY KEY,
		platform TEXT NOT NULL CHECK (platform IN ('plex', 'jellyfin', 'emby', 'tautulli')),
		name TEXT NOT NULL,
		url_encrypted TEXT NOT NULL,
		token_encrypted TEXT NOT NULL,
		server_id TEXT UNIQUE,

		-- Operational state
		enabled BOOLEAN DEFAULT TRUE,

		-- Platform-specific settings (JSON blob for flexibility)
		settings JSON DEFAULT '{}',

		-- Sync configuration
		realtime_enabled BOOLEAN DEFAULT FALSE,
		webhooks_enabled BOOLEAN DEFAULT FALSE,
		session_polling_enabled BOOLEAN DEFAULT FALSE,
		session_polling_interval TEXT DEFAULT '30s',

		-- Source tracking
		source TEXT DEFAULT 'ui' CHECK (source IN ('env', 'ui', 'import')),

		-- Ownership and timestamps
		created_by TEXT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

		-- Sync status (updated by sync service)
		last_sync_at TIMESTAMPTZ,
		last_sync_status TEXT,
		last_error TEXT,
		last_error_at TIMESTAMPTZ
	);`)

	// Media server audit log table (v2.7 - ADR-0026: Multi-Server Management UI)
	// Records all server configuration changes for security auditing and compliance.
	// Immutable append-only log of server create, update, delete, enable, disable, test operations.
	// Note: Credentials are NEVER logged - only metadata changes are recorded.
	queries = append(queries, `CREATE TABLE IF NOT EXISTS media_server_audit (
		id TEXT PRIMARY KEY,
		server_id TEXT NOT NULL,
		action TEXT NOT NULL CHECK (action IN ('create', 'update', 'delete', 'enable', 'disable', 'test', 'sync')),
		user_id TEXT NOT NULL,
		username TEXT,
		changes JSON,
		ip_address TEXT,
		user_agent TEXT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`)

	// Standard indexes
	queries = append(queries,
		`CREATE INDEX IF NOT EXISTS idx_playback_started_at ON playback_events(started_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_user_id ON playback_events(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_ip ON playback_events(ip_address);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_session_key ON playback_events(session_key);`,
		// User mappings indexes (v2.0 - Phase 0.7)
		`CREATE INDEX IF NOT EXISTS idx_user_mappings_internal ON user_mappings(internal_user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_user_mappings_external ON user_mappings(source, server_id, external_user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_user_mappings_username ON user_mappings(username);`,
		// Failed events indexes (v2.1 - ADR-0023)
		`CREATE INDEX IF NOT EXISTS idx_failed_events_status ON failed_events(status);`,
		`CREATE INDEX IF NOT EXISTS idx_failed_events_source ON failed_events(source);`,
		`CREATE INDEX IF NOT EXISTS idx_failed_events_failed_at ON failed_events(failed_at);`,
		`CREATE INDEX IF NOT EXISTS idx_failed_events_transaction_id ON failed_events(transaction_id);`,
		// Dedupe audit log indexes (v2.2 - ADR-0022)
		`CREATE INDEX IF NOT EXISTS idx_dedupe_audit_timestamp ON dedupe_audit_log(timestamp DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_dedupe_audit_user_id ON dedupe_audit_log(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_dedupe_audit_status ON dedupe_audit_log(status);`,
		`CREATE INDEX IF NOT EXISTS idx_dedupe_audit_discarded ON dedupe_audit_log(discarded_event_id);`,
		`CREATE INDEX IF NOT EXISTS idx_dedupe_audit_source ON dedupe_audit_log(discarded_source);`,
		`CREATE INDEX IF NOT EXISTS idx_dedupe_audit_reason ON dedupe_audit_log(dedupe_reason);`,
		// Wrapped reports indexes (v2.3 - Annual Wrapped Reports)
		`CREATE INDEX IF NOT EXISTS idx_wrapped_reports_year ON wrapped_reports(year);`,
		`CREATE INDEX IF NOT EXISTS idx_wrapped_reports_user_year ON wrapped_reports(user_id, year);`,
		`CREATE INDEX IF NOT EXISTS idx_wrapped_reports_share_token ON wrapped_reports(share_token);`,
		`CREATE INDEX IF NOT EXISTS idx_wrapped_reports_generated ON wrapped_reports(generated_at DESC);`,
		// User roles indexes (v2.4 - RBAC Implementation)
		`CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles(role);`,
		`CREATE INDEX IF NOT EXISTS idx_user_roles_active ON user_roles(is_active);`,
		`CREATE INDEX IF NOT EXISTS idx_user_roles_expires ON user_roles(expires_at);`,
		// Role audit log indexes (v2.4 - RBAC Implementation)
		`CREATE INDEX IF NOT EXISTS idx_role_audit_timestamp ON role_audit_log(timestamp DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_role_audit_actor ON role_audit_log(actor_id);`,
		`CREATE INDEX IF NOT EXISTS idx_role_audit_target ON role_audit_log(target_user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_role_audit_action ON role_audit_log(action);`,
		// Personal access tokens indexes (v2.5 - PAT System)
		`CREATE INDEX IF NOT EXISTS idx_pat_user_id ON personal_access_tokens(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_pat_token_prefix ON personal_access_tokens(token_prefix);`,
		`CREATE INDEX IF NOT EXISTS idx_pat_created_at ON personal_access_tokens(created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_pat_expires_at ON personal_access_tokens(expires_at);`,
		`CREATE INDEX IF NOT EXISTS idx_pat_revoked_at ON personal_access_tokens(revoked_at);`,
		`CREATE INDEX IF NOT EXISTS idx_pat_last_used ON personal_access_tokens(last_used_at DESC);`,
		// PAT usage log indexes (v2.5 - PAT System)
		`CREATE INDEX IF NOT EXISTS idx_pat_usage_timestamp ON pat_usage_log(timestamp DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_pat_usage_token_id ON pat_usage_log(token_id);`,
		`CREATE INDEX IF NOT EXISTS idx_pat_usage_user_id ON pat_usage_log(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_pat_usage_success ON pat_usage_log(success);`,
		// Newsletter templates indexes (v2.6 - Newsletter Generator)
		`CREATE INDEX IF NOT EXISTS idx_newsletter_templates_type ON newsletter_templates(type);`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_templates_active ON newsletter_templates(is_active);`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_templates_created ON newsletter_templates(created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_templates_created_by ON newsletter_templates(created_by);`,
		// Newsletter schedules indexes (v2.6 - Newsletter Generator)
		`CREATE INDEX IF NOT EXISTS idx_newsletter_schedules_template ON newsletter_schedules(template_id);`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_schedules_enabled ON newsletter_schedules(is_enabled);`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_schedules_next_run ON newsletter_schedules(next_run_at);`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_schedules_created_by ON newsletter_schedules(created_by);`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_schedules_last_run ON newsletter_schedules(last_run_at DESC);`,
		// Newsletter deliveries indexes (v2.6 - Newsletter Generator)
		`CREATE INDEX IF NOT EXISTS idx_newsletter_deliveries_schedule ON newsletter_deliveries(schedule_id);`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_deliveries_template ON newsletter_deliveries(template_id);`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_deliveries_channel ON newsletter_deliveries(channel);`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_deliveries_status ON newsletter_deliveries(status);`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_deliveries_started ON newsletter_deliveries(started_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_deliveries_triggered_by ON newsletter_deliveries(triggered_by);`,
		// Newsletter user preferences indexes (v2.6 - Newsletter Generator)
		`CREATE INDEX IF NOT EXISTS idx_newsletter_prefs_opt_out ON newsletter_user_preferences(global_opt_out);`,
		// Newsletter audit log indexes (v2.6 - Newsletter Generator)
		`CREATE INDEX IF NOT EXISTS idx_newsletter_audit_timestamp ON newsletter_audit_log(timestamp DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_audit_actor ON newsletter_audit_log(actor_id);`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_audit_resource ON newsletter_audit_log(resource_type, resource_id);`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_audit_action ON newsletter_audit_log(action);`,
		// Media servers indexes (v2.7 - ADR-0026: Multi-Server Management UI)
		`CREATE INDEX IF NOT EXISTS idx_media_servers_platform ON media_servers(platform);`,
		`CREATE INDEX IF NOT EXISTS idx_media_servers_enabled ON media_servers(enabled);`,
		`CREATE INDEX IF NOT EXISTS idx_media_servers_source ON media_servers(source);`,
		`CREATE INDEX IF NOT EXISTS idx_media_servers_created_at ON media_servers(created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_media_servers_last_sync ON media_servers(last_sync_at DESC);`,
		// Media server audit log indexes (v2.7 - ADR-0026: Multi-Server Management UI)
		`CREATE INDEX IF NOT EXISTS idx_media_server_audit_server ON media_server_audit(server_id);`,
		`CREATE INDEX IF NOT EXISTS idx_media_server_audit_timestamp ON media_server_audit(created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_media_server_audit_user ON media_server_audit(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_media_server_audit_action ON media_server_audit(action);`,
	)

	return queries
}

// NOTE: The schema is now fully consolidated in getTableCreationQueries().
// For post-release schema changes, use versioned migrations in migrations.go.
// See CLAUDE.md "Schema Consolidation" section for details on the migration strategy.

// createIndexes creates database indexes for query optimization
// Skips index creation if cfg.SkipIndexes is true (for fast test setup).
// This is critical for CI performance: 97 indexes × 326 tests = 31,622 index operations
// which causes CGO resource exhaustion and test timeouts.
func (db *DB) createIndexes() error {
	// Skip index creation for tests to avoid CGO resource exhaustion
	// Tests that specifically need indexes can call CreateIndexes() explicitly
	if db.cfg != nil && db.cfg.SkipIndexes {
		return nil
	}

	return db.doCreateIndexes()
}

// CreateIndexes creates all database indexes.
// This is exposed for tests that specifically need indexes (e.g., TestSpatialIndexCreation).
// Most tests should use SkipIndexes: true for fast setup.
func (db *DB) CreateIndexes() error {
	return db.doCreateIndexes()
}

// doCreateIndexes is the internal implementation that creates all indexes.
func (db *DB) doCreateIndexes() error {
	ctx, cancel := schemaContext()
	defer cancel()

	indexes := db.getIndexQueries()

	for _, query := range indexes {
		if _, err := db.conn.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute index query: %s: %w", query, err)
		}
	}

	return nil
}

// getIndexQueries returns index creation SQL statements
func (db *DB) getIndexQueries() []string {
	return []string{
		// Basic indexes
		`CREATE INDEX IF NOT EXISTS idx_playback_section_id ON playback_events(section_id);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_library_name ON playback_events(library_name);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_content_rating ON playback_events(content_rating);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_year ON playback_events(year);`,

		// Analytics indexes
		`CREATE INDEX IF NOT EXISTS idx_playback_video_dynamic_range ON playback_events(video_dynamic_range);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_audio_channels ON playback_events(audio_channels);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_container ON playback_events(container);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_subtitles ON playback_events(subtitles);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_relayed ON playback_events(relayed);`,

		// Composite indexes for query performance
		`CREATE INDEX IF NOT EXISTS idx_playback_started_user ON playback_events(started_at DESC, user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_started_media ON playback_events(started_at DESC, media_type);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_user_started ON playback_events(user_id, started_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_library_started ON playback_events(section_id, started_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_transcode ON playback_events(transcode_decision, started_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_duration_started ON playback_events(play_duration, started_at DESC);`,

		// Binge detection indexes
		`CREATE INDEX IF NOT EXISTS idx_playback_media_index ON playback_events(media_index);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_parent_media_index ON playback_events(parent_media_index);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_rating_key ON playback_events(rating_key);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_parent_rating_key ON playback_events(parent_rating_key);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_grandparent_rating_key ON playback_events(grandparent_rating_key);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_binge ON playback_events(user_id, grandparent_rating_key, parent_media_index, media_index, started_at);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_genres ON playback_events(genres);`,

		// Plex API indexes
		`CREATE INDEX IF NOT EXISTS idx_playback_source ON playback_events(source);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_plex_key ON playback_events(plex_key);`,

		// Legacy deduplication index (kept for backwards compatibility with pre-v1.47 events)
		// This catches exact timestamp duplicates for events without correlation_key
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_playback_dedup ON playback_events(rating_key, user_id, started_at);`,

		// v1.47 - Cross-source deduplication index
		// Primary deduplication mechanism for NATS event sourcing mode
		// NULL correlation_keys are allowed (legacy events) - DuckDB allows multiple NULLs in unique indexes
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_playback_correlation_key ON playback_events(correlation_key);`,

		// Bitrate analytics index
		`CREATE INDEX IF NOT EXISTS idx_playback_bitrate_analytics ON playback_events(source_bitrate, transcode_bitrate);`,

		// Hardware transcode indexes
		`CREATE INDEX IF NOT EXISTS idx_playback_transcode_hw_encoding ON playback_events(transcode_hw_encoding);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_transcode_hw_decoding ON playback_events(transcode_hw_decoding);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_transcode_hw_full_pipeline ON playback_events(transcode_hw_full_pipeline);`,
		`CREATE INDEX IF NOT EXISTS idx_playback_hw_transcode ON playback_events(transcode_hw_encoding, transcode_hw_decoding, started_at DESC);`,

		// HDR/Color metadata index
		`CREATE INDEX IF NOT EXISTS idx_playback_video_color_space ON playback_events(video_color_space);`,

		// Machine ID index
		`CREATE INDEX IF NOT EXISTS idx_playback_machine_id ON playback_events(machine_id);`,

		// Live TV index
		`CREATE INDEX IF NOT EXISTS idx_playback_live ON playback_events(live);`,

		// v2.0 Multi-server indexes (Phase 0.6)
		// Server ID index for per-server analytics and filtering
		`CREATE INDEX IF NOT EXISTS idx_playback_server_id ON playback_events(server_id);`,
		// Composite index for source + server_id filtering
		`CREATE INDEX IF NOT EXISTS idx_playback_source_server ON playback_events(source, server_id);`,

		// v2.1 Exactly-once delivery index (ADR-0023)
		// Transaction ID index for idempotent Consumer WAL commits
		// Allows efficient duplicate detection on crash recovery
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_playback_transaction_id ON playback_events(transaction_id);`,

		// Concurrent streams analytics index
		// Composite index for interval overlap queries: started_at <= X AND stopped_at >= Y
		// Covers the concurrent streams time series query pattern for efficient session counting
		`CREATE INDEX IF NOT EXISTS idx_playback_concurrent_streams ON playback_events(stopped_at, started_at, session_key, transcode_decision);`,
	}
}
