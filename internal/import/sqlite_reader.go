// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//nolint:revive // package name with underscore is intentional for clarity
package tautulli_import

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// DuckDB driver - used with SQLite extension for reading Tautulli databases
	_ "github.com/duckdb/duckdb-go/v2"
)

// TautulliRecord represents a raw record from the Tautulli SQLite database.
// This struct combines data from session_history, session_history_metadata,
// and session_history_media_info tables.
type TautulliRecord struct {
	// Core session fields (session_history)
	ID              int64     // session_history.id
	SessionKey      string    // session_history.session_key
	StartedAt       time.Time // session_history.started (unix timestamp)
	StoppedAt       time.Time // session_history.stopped (unix timestamp)
	UserID          int       // session_history.user_id
	Username        string    // session_history.user
	IPAddress       string    // session_history.ip_address
	Platform        string    // session_history.platform
	Player          string    // session_history.player
	PercentComplete int       // session_history.percent_complete
	PausedCounter   int       // session_history.paused_counter

	// Media metadata fields (session_history_metadata)
	MediaType        string  // session_history_metadata.media_type
	Title            string  // session_history_metadata.title
	ParentTitle      *string // session_history_metadata.parent_title
	GrandparentTitle *string // session_history_metadata.grandparent_title
	RatingKey        *string // session_history_metadata.rating_key
	ParentRatingKey  *string // session_history_metadata.parent_rating_key
	GrandparentRKey  *string // session_history_metadata.grandparent_rating_key
	Year             *int    // session_history_metadata.year
	MediaIndex       *int    // session_history_metadata.media_index
	ParentMediaIndex *int    // session_history_metadata.parent_media_index
	Thumb            *string // session_history_metadata.thumb
	ParentThumb      *string // session_history_metadata.parent_thumb
	GrandparentThumb *string // session_history_metadata.grandparent_thumb
	SectionID        *int    // session_history_metadata.section_id
	LibraryName      *string // session_history_metadata.library_name
	ContentRating    *string // session_history_metadata.content_rating
	Guid             *string // session_history_metadata.guid
	Directors        *string // session_history_metadata.directors
	Writers          *string // session_history_metadata.writers
	Actors           *string // session_history_metadata.actors
	Genres           *string // session_history_metadata.genres
	Studio           *string // session_history_metadata.studio
	FullTitle        *string // session_history_metadata.full_title
	OriginalTitle    *string // session_history_metadata.original_title
	OriginallyAvail  *string // session_history_metadata.originally_available_at

	// Stream quality fields (session_history_media_info)
	VideoResolution     *string // session_history_media_info.video_resolution
	VideoCodec          *string // session_history_media_info.video_codec
	VideoFullResolution *string // session_history_media_info.video_full_resolution
	AudioCodec          *string // session_history_media_info.audio_codec
	AudioChannels       *string // session_history_media_info.audio_channels
	Container           *string // session_history_media_info.container
	Bitrate             *int    // session_history_media_info.bitrate
	TranscodeDecision   *string // session_history_media_info.transcode_decision
	VideoDecision       *string // session_history_media_info.video_decision
	AudioDecision       *string // session_history_media_info.audio_decision
	SubtitleDecision    *string // session_history_media_info.subtitle_decision
	StreamBitrate       *int    // session_history_media_info.stream_bitrate
	StreamVideoCodec    *string // session_history_media_info.stream_video_codec
	StreamVideoRes      *string // session_history_media_info.stream_video_resolution
	StreamAudioCodec    *string // session_history_media_info.stream_audio_codec
	StreamAudioChannels *string // session_history_media_info.stream_audio_channels

	// Additional fields
	FriendlyName *string // session_history.friendly_name
	MachineID    *string // session_history.machine_id
	Product      *string // session_history.product
	LocationType string  // derived from session_history.location
}

// SQLiteReader reads records from a Tautulli SQLite database using DuckDB's SQLite extension.
// This approach allows direct reading of SQLite databases without a separate SQLite driver.
type SQLiteReader struct {
	db     *sql.DB
	dbPath string
}

// NewSQLiteReader creates a new reader for the specified Tautulli database file.
// It uses DuckDB's SQLite extension to attach and read the SQLite database.
func NewSQLiteReader(dbPath string) (*SQLiteReader, error) {
	// Create an in-memory DuckDB connection for reading the SQLite database
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}

	// Install and load the SQLite extension
	if err := loadSQLiteExtension(db); err != nil {
		db.Close() //nolint:errcheck // best-effort cleanup on error path
		return nil, fmt.Errorf("load sqlite extension: %w", err)
	}

	// Attach the Tautulli SQLite database
	if err := attachSQLiteDatabase(db, dbPath); err != nil {
		db.Close() //nolint:errcheck // best-effort cleanup on error path
		return nil, fmt.Errorf("attach database: %w", err)
	}

	// Verify required tables exist
	if err := verifyTables(db); err != nil {
		// Detach before closing
		detachSQLiteDatabase(db)
		db.Close() //nolint:errcheck // best-effort cleanup on error path
		return nil, fmt.Errorf("verify tables: %w", err)
	}

	return &SQLiteReader{
		db:     db,
		dbPath: dbPath,
	}, nil
}

// loadSQLiteExtension installs and loads the sqlite_scanner extension in DuckDB.
func loadSQLiteExtension(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Try to install, then load (extension may already be installed)
	if _, err := db.ExecContext(ctx, "INSTALL sqlite_scanner;"); err != nil {
		// Installation might fail if already installed, try loading
		if _, loadErr := db.ExecContext(ctx, "LOAD sqlite_scanner;"); loadErr != nil {
			// Try force install as last resort
			if _, forceErr := db.ExecContext(ctx, "FORCE INSTALL sqlite_scanner;"); forceErr != nil {
				return fmt.Errorf("install error: %w, load error: %w, force install error: %w", err, loadErr, forceErr)
			}
		}
		return nil
	}

	_, err := db.ExecContext(ctx, "LOAD sqlite_scanner;")
	return err
}

// attachSQLiteDatabase attaches a SQLite database file to DuckDB.
func attachSQLiteDatabase(db *sql.DB, dbPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use sqlite_attach to attach the database with a schema name
	_, err := db.ExecContext(ctx, "CALL sqlite_attach(?)", dbPath)
	if err != nil {
		return fmt.Errorf("sqlite_attach: %w", err)
	}

	return nil
}

// detachSQLiteDatabase detaches the SQLite database from DuckDB.
func detachSQLiteDatabase(db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Use DETACH DATABASE with the database name derived from the file
	// The default name is the file name without extension
	db.ExecContext(ctx, "DETACH DATABASE IF EXISTS tautulli") //nolint:errcheck // best-effort detach, errors not actionable
}

// verifyTables checks that all required Tautulli tables exist.
func verifyTables(db *sql.DB) error {
	requiredTables := []string{
		"session_history",
		"session_history_metadata",
		"session_history_media_info",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Query the information schema to find tables
	// DuckDB exposes attached SQLite tables through information_schema
	for _, table := range requiredTables {
		var count int
		err := db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?",
			table,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("check table %s: %w", table, err)
		}
		if count == 0 {
			return fmt.Errorf("table %s not found in attached database", table)
		}
	}

	return nil
}

// Close closes the database connection.
func (r *SQLiteReader) Close() error {
	detachSQLiteDatabase(r.db)
	return r.db.Close()
}

// CountRecords returns the total number of session history records.
func (r *SQLiteReader) CountRecords(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM session_history").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count records: %w", err)
	}
	return count, nil
}

// CountRecordsSince returns the number of records with ID greater than the given ID.
func (r *SQLiteReader) CountRecordsSince(ctx context.Context, sinceID int64) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM session_history WHERE id > ?",
		sinceID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count records since %d: %w", sinceID, err)
	}
	return count, nil
}

// ReadBatch reads a batch of records starting from the given ID.
// Records are ordered by ID ascending to ensure consistent resumability.
func (r *SQLiteReader) ReadBatch(ctx context.Context, sinceID int64, limit int) ([]TautulliRecord, error) {
	// Join all three tables to get complete record data
	// Tables are accessed through the attached SQLite database
	query := `
		SELECT
			-- Core session fields
			sh.id,
			sh.session_key,
			sh.started,
			sh.stopped,
			sh.user_id,
			sh.user,
			sh.ip_address,
			sh.platform,
			sh.player,
			sh.percent_complete,
			sh.paused_counter,
			sh.friendly_name,
			sh.machine_id,
			sh.product,
			sh.location,

			-- Media metadata fields
			shm.media_type,
			shm.title,
			shm.parent_title,
			shm.grandparent_title,
			shm.rating_key,
			shm.parent_rating_key,
			shm.grandparent_rating_key,
			shm.year,
			shm.media_index,
			shm.parent_media_index,
			shm.thumb,
			shm.parent_thumb,
			shm.grandparent_thumb,
			shm.section_id,
			shm.library_name,
			shm.content_rating,
			shm.guid,
			shm.directors,
			shm.writers,
			shm.actors,
			shm.genres,
			shm.studio,
			shm.full_title,
			shm.original_title,
			shm.originally_available_at,

			-- Stream quality fields
			shmi.video_resolution,
			shmi.video_codec,
			shmi.video_full_resolution,
			shmi.audio_codec,
			shmi.audio_channels,
			shmi.container,
			shmi.bitrate,
			shmi.transcode_decision,
			shmi.video_decision,
			shmi.audio_decision,
			shmi.subtitle_decision,
			shmi.stream_bitrate,
			shmi.stream_video_codec,
			shmi.stream_video_resolution,
			shmi.stream_audio_codec,
			shmi.stream_audio_channels
		FROM session_history sh
		LEFT JOIN session_history_metadata shm ON sh.id = shm.id
		LEFT JOIN session_history_media_info shmi ON sh.id = shmi.id
		WHERE sh.id > ?
		ORDER BY sh.id ASC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, sinceID, limit)
	if err != nil {
		return nil, fmt.Errorf("query records: %w", err)
	}
	defer rows.Close()

	var records []TautulliRecord
	for rows.Next() {
		var rec TautulliRecord
		var startedUnix, stoppedUnix int64
		var location sql.NullString

		err := rows.Scan(
			// Core session fields
			&rec.ID,
			&rec.SessionKey,
			&startedUnix,
			&stoppedUnix,
			&rec.UserID,
			&rec.Username,
			&rec.IPAddress,
			&rec.Platform,
			&rec.Player,
			&rec.PercentComplete,
			&rec.PausedCounter,
			&rec.FriendlyName,
			&rec.MachineID,
			&rec.Product,
			&location,

			// Media metadata fields
			&rec.MediaType,
			&rec.Title,
			&rec.ParentTitle,
			&rec.GrandparentTitle,
			&rec.RatingKey,
			&rec.ParentRatingKey,
			&rec.GrandparentRKey,
			&rec.Year,
			&rec.MediaIndex,
			&rec.ParentMediaIndex,
			&rec.Thumb,
			&rec.ParentThumb,
			&rec.GrandparentThumb,
			&rec.SectionID,
			&rec.LibraryName,
			&rec.ContentRating,
			&rec.Guid,
			&rec.Directors,
			&rec.Writers,
			&rec.Actors,
			&rec.Genres,
			&rec.Studio,
			&rec.FullTitle,
			&rec.OriginalTitle,
			&rec.OriginallyAvail,

			// Stream quality fields
			&rec.VideoResolution,
			&rec.VideoCodec,
			&rec.VideoFullResolution,
			&rec.AudioCodec,
			&rec.AudioChannels,
			&rec.Container,
			&rec.Bitrate,
			&rec.TranscodeDecision,
			&rec.VideoDecision,
			&rec.AudioDecision,
			&rec.SubtitleDecision,
			&rec.StreamBitrate,
			&rec.StreamVideoCodec,
			&rec.StreamVideoRes,
			&rec.StreamAudioCodec,
			&rec.StreamAudioChannels,
		)
		if err != nil {
			return nil, fmt.Errorf("scan record: %w", err)
		}

		// Convert Unix timestamps to time.Time
		rec.StartedAt = time.Unix(startedUnix, 0)
		if stoppedUnix > 0 {
			rec.StoppedAt = time.Unix(stoppedUnix, 0)
		}

		// Derive location type from location field
		if location.Valid {
			switch location.String {
			case "lan":
				rec.LocationType = "lan"
			case "wan":
				rec.LocationType = "wan"
			default:
				rec.LocationType = "wan" // Default to WAN for unknown
			}
		} else {
			rec.LocationType = "wan"
		}

		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate records: %w", err)
	}

	return records, nil
}

// GetDateRange returns the earliest and latest session timestamps.
func (r *SQLiteReader) GetDateRange(ctx context.Context) (earliest, latest time.Time, err error) {
	var minTS, maxTS int64
	err = r.db.QueryRowContext(ctx,
		"SELECT MIN(started), MAX(started) FROM session_history",
	).Scan(&minTS, &maxTS)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("get date range: %w", err)
	}

	return time.Unix(minTS, 0), time.Unix(maxTS, 0), nil
}

// GetUserStats returns statistics about users in the database.
func (r *SQLiteReader) GetUserStats(ctx context.Context) (uniqueUsers int, err error) {
	err = r.db.QueryRowContext(ctx,
		"SELECT COUNT(DISTINCT user_id) FROM session_history",
	).Scan(&uniqueUsers)
	if err != nil {
		return 0, fmt.Errorf("get user stats: %w", err)
	}
	return uniqueUsers, nil
}

// GetMediaTypeStats returns counts by media type.
func (r *SQLiteReader) GetMediaTypeStats(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT shm.media_type, COUNT(*)
		FROM session_history sh
		JOIN session_history_metadata shm ON sh.id = shm.id
		GROUP BY shm.media_type
	`)
	if err != nil {
		return nil, fmt.Errorf("get media type stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var mediaType string
		var count int
		if err := rows.Scan(&mediaType, &count); err != nil {
			return nil, fmt.Errorf("scan media type: %w", err)
		}
		stats[mediaType] = count
	}

	return stats, rows.Err()
}
