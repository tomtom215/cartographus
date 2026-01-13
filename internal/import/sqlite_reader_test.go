// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulliimport

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupDuckDBWithSQLiteScanner creates an in-memory DuckDB connection with the sqlite_scanner extension loaded.
// Returns the database connection and context. Caller must close the database.
func setupDuckDBWithSQLiteScanner(t *testing.T) (*sql.DB, context.Context) {
	t.Helper()

	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("Failed to open duckdb: %v", err)
	}

	ctx := context.Background()

	// Install and load sqlite_scanner extension (install may fail if already installed, that's ok)
	_, _ = db.ExecContext(ctx, "INSTALL sqlite_scanner;")
	if _, err := db.ExecContext(ctx, "LOAD sqlite_scanner;"); err != nil {
		db.Close()
		t.Fatalf("Failed to load sqlite_scanner extension: %v", err)
	}

	return db, ctx
}

// attachSQLiteDB attaches a SQLite database to the DuckDB connection with the given alias.
func attachSQLiteDB(t *testing.T, db *sql.DB, ctx context.Context, dbPath, alias string) {
	t.Helper()
	attachSQL := fmt.Sprintf("ATTACH '%s' AS %s (TYPE SQLITE)", dbPath, alias)
	if _, err := db.ExecContext(ctx, attachSQL); err != nil {
		t.Fatalf("Failed to attach SQLite database: %v", err)
	}
}

// detachSQLiteDB detaches a SQLite database from the DuckDB connection.
func detachSQLiteDB(t *testing.T, db *sql.DB, ctx context.Context, alias string) {
	t.Helper()
	if _, err := db.ExecContext(ctx, fmt.Sprintf("DETACH %s", alias)); err != nil {
		t.Fatalf("Failed to detach database: %v", err)
	}
}

// createTestDatabase creates a temporary Tautulli-like SQLite database for testing.
// Uses DuckDB's SQLite extension to create and populate the database.
func createTestDatabase(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "tautulli-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "tautulli.db")

	db, ctx := setupDuckDBWithSQLiteScanner(t)
	defer db.Close()

	attachSQLiteDB(t, db, ctx, dbPath, "tautulli")

	// Create Tautulli tables in the SQLite database
	schema := `
		CREATE TABLE tautulli.session_history (
			id INTEGER PRIMARY KEY,
			session_key TEXT,
			started INTEGER,
			stopped INTEGER,
			user_id INTEGER,
			user TEXT,
			ip_address TEXT,
			platform TEXT,
			player TEXT,
			percent_complete INTEGER,
			paused_counter INTEGER,
			friendly_name TEXT,
			machine_id TEXT,
			product TEXT,
			location TEXT
		);

		CREATE TABLE tautulli.session_history_metadata (
			id INTEGER PRIMARY KEY,
			media_type TEXT,
			title TEXT,
			parent_title TEXT,
			grandparent_title TEXT,
			rating_key TEXT,
			parent_rating_key TEXT,
			grandparent_rating_key TEXT,
			year INTEGER,
			media_index INTEGER,
			parent_media_index INTEGER,
			thumb TEXT,
			parent_thumb TEXT,
			grandparent_thumb TEXT,
			section_id INTEGER,
			library_name TEXT,
			content_rating TEXT,
			guid TEXT,
			directors TEXT,
			writers TEXT,
			actors TEXT,
			genres TEXT,
			studio TEXT,
			full_title TEXT,
			original_title TEXT,
			originally_available_at TEXT
		);

		CREATE TABLE tautulli.session_history_media_info (
			id INTEGER PRIMARY KEY,
			video_resolution TEXT,
			video_codec TEXT,
			video_full_resolution TEXT,
			audio_codec TEXT,
			audio_channels TEXT,
			container TEXT,
			bitrate INTEGER,
			transcode_decision TEXT,
			video_decision TEXT,
			audio_decision TEXT,
			subtitle_decision TEXT,
			stream_bitrate INTEGER,
			stream_video_codec TEXT,
			stream_video_resolution TEXT,
			stream_audio_codec TEXT,
			stream_audio_channels TEXT
		);
	`

	if _, err := db.ExecContext(ctx, schema); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create tables: %v", err)
	}

	detachSQLiteDB(t, db, ctx, "tautulli")

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return dbPath, cleanup
}

// insertTestRecords inserts test records into the database.
// Uses DuckDB's SQLite extension to write to the database.
func insertTestRecords(t *testing.T, dbPath string, count int) {
	t.Helper()

	db, ctx := setupDuckDBWithSQLiteScanner(t)
	defer db.Close()

	attachSQLiteDB(t, db, ctx, dbPath, "tautulli")

	baseTime := time.Now().Add(-24 * time.Hour).Unix()

	for i := 1; i <= count; i++ {
		started := baseTime + int64(i*3600) // 1 hour apart
		stopped := started + 7200           // 2 hour session

		// Insert into session_history
		_, err := db.ExecContext(ctx, `
			INSERT INTO tautulli.session_history (id, session_key, started, stopped, user_id, user, ip_address, platform, player, percent_complete, paused_counter, location)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, i, "session-"+string(rune('0'+i%10)), started, stopped, (i%5)+1, "user"+string(rune('0'+i%5)), "192.168.1."+string(rune('0'+i%254+1)), "Chrome", "Plex Web", 100, 0, "lan")
		if err != nil {
			t.Fatalf("Failed to insert session_history: %v", err)
		}

		// Insert into session_history_metadata
		mediaType := "movie"
		if i%3 == 0 {
			mediaType = "episode"
		}
		_, err = db.ExecContext(ctx, `
			INSERT INTO tautulli.session_history_metadata (id, media_type, title, year)
			VALUES (?, ?, ?, ?)
		`, i, mediaType, "Test Media "+string(rune('0'+i%10)), 2024)
		if err != nil {
			t.Fatalf("Failed to insert session_history_metadata: %v", err)
		}

		// Insert into session_history_media_info
		_, err = db.ExecContext(ctx, `
			INSERT INTO tautulli.session_history_media_info (id, video_resolution, video_codec, transcode_decision)
			VALUES (?, ?, ?, ?)
		`, i, "1080", "h264", "direct play")
		if err != nil {
			t.Fatalf("Failed to insert session_history_media_info: %v", err)
		}
	}

	detachSQLiteDB(t, db, ctx, "tautulli")
}

func TestNewSQLiteReader(t *testing.T) {
	t.Run("opens valid database", func(t *testing.T) {
		dbPath, cleanup := createTestDatabase(t)
		defer cleanup()

		reader, err := NewSQLiteReader(dbPath)
		if err != nil {
			t.Fatalf("NewSQLiteReader() error = %v", err)
		}
		defer reader.Close()
	})

	t.Run("fails on non-existent file", func(t *testing.T) {
		_, err := NewSQLiteReader("/nonexistent/path/to/database.db")
		if err == nil {
			t.Error("NewSQLiteReader() expected error for non-existent file")
		}
	})

	t.Run("fails on database without required tables", func(t *testing.T) {
		// Create temp directory
		tmpDir, err := os.MkdirTemp("", "empty-db-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		emptyDBPath := filepath.Join(tmpDir, "empty.db")

		db, ctx := setupDuckDBWithSQLiteScanner(t)
		attachSQLiteDB(t, db, ctx, emptyDBPath, "emptydb")
		detachSQLiteDB(t, db, ctx, "emptydb")
		db.Close()

		_, err = NewSQLiteReader(emptyDBPath)
		if err == nil {
			t.Error("NewSQLiteReader() expected error for database without required tables")
		}
	})
}

func TestSQLiteReader_CountRecords(t *testing.T) {
	dbPath, cleanup := createTestDatabase(t)
	defer cleanup()

	insertTestRecords(t, dbPath, 10)

	reader, err := NewSQLiteReader(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteReader() error = %v", err)
	}
	defer reader.Close()

	count, err := reader.CountRecords(context.Background())
	if err != nil {
		t.Fatalf("CountRecords() error = %v", err)
	}

	if count != 10 {
		t.Errorf("CountRecords() = %d, want 10", count)
	}
}

func TestSQLiteReader_CountRecordsSince(t *testing.T) {
	dbPath, cleanup := createTestDatabase(t)
	defer cleanup()

	insertTestRecords(t, dbPath, 10)

	reader, err := NewSQLiteReader(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteReader() error = %v", err)
	}
	defer reader.Close()

	// Count records with ID > 5 (should be 5 records: 6, 7, 8, 9, 10)
	count, err := reader.CountRecordsSince(context.Background(), 5)
	if err != nil {
		t.Fatalf("CountRecordsSince() error = %v", err)
	}

	if count != 5 {
		t.Errorf("CountRecordsSince(5) = %d, want 5", count)
	}
}

func TestSQLiteReader_ReadBatch(t *testing.T) {
	dbPath, cleanup := createTestDatabase(t)
	defer cleanup()

	insertTestRecords(t, dbPath, 10)

	reader, err := NewSQLiteReader(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteReader() error = %v", err)
	}
	defer reader.Close()

	t.Run("reads first batch", func(t *testing.T) {
		records, err := reader.ReadBatch(context.Background(), 0, 5)
		if err != nil {
			t.Fatalf("ReadBatch() error = %v", err)
		}

		if len(records) != 5 {
			t.Errorf("ReadBatch() returned %d records, want 5", len(records))
		}

		// Verify first record
		if records[0].ID != 1 {
			t.Errorf("First record ID = %d, want 1", records[0].ID)
		}

		// Verify last record in batch
		if records[4].ID != 5 {
			t.Errorf("Last record ID = %d, want 5", records[4].ID)
		}
	})

	t.Run("reads second batch", func(t *testing.T) {
		records, err := reader.ReadBatch(context.Background(), 5, 5)
		if err != nil {
			t.Fatalf("ReadBatch() error = %v", err)
		}

		if len(records) != 5 {
			t.Errorf("ReadBatch() returned %d records, want 5", len(records))
		}

		// Verify first record
		if records[0].ID != 6 {
			t.Errorf("First record ID = %d, want 6", records[0].ID)
		}
	})

	t.Run("returns empty slice when no more records", func(t *testing.T) {
		records, err := reader.ReadBatch(context.Background(), 10, 5)
		if err != nil {
			t.Fatalf("ReadBatch() error = %v", err)
		}

		if len(records) != 0 {
			t.Errorf("ReadBatch() returned %d records, want 0", len(records))
		}
	})

	t.Run("handles partial last batch", func(t *testing.T) {
		records, err := reader.ReadBatch(context.Background(), 8, 5)
		if err != nil {
			t.Fatalf("ReadBatch() error = %v", err)
		}

		if len(records) != 2 {
			t.Errorf("ReadBatch() returned %d records, want 2", len(records))
		}
	})
}

func TestSQLiteReader_RecordFields(t *testing.T) {
	dbPath, cleanup := createTestDatabase(t)
	defer cleanup()

	// Insert a record with known values using DuckDB
	db, ctx := setupDuckDBWithSQLiteScanner(t)
	attachSQLiteDB(t, db, ctx, dbPath, "tautulli")

	started := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC).Unix()
	stopped := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix()

	_, err := db.ExecContext(ctx, `
		INSERT INTO tautulli.session_history (id, session_key, started, stopped, user_id, user, ip_address, platform, player, percent_complete, paused_counter, location, friendly_name, machine_id)
		VALUES (1, 'test-session', ?, ?, 42, 'testuser', '192.168.1.100', 'Chrome', 'Plex Web', 75, 120, 'lan', 'Test User', 'machine-123')
	`, started, stopped)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to insert session_history: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO tautulli.session_history_metadata (id, media_type, title, parent_title, grandparent_title, year, section_id, library_name)
		VALUES (1, 'episode', 'Pilot', 'Season 1', 'Test Show', 2024, 1, 'TV Shows')
	`)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to insert session_history_metadata: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO tautulli.session_history_media_info (id, video_resolution, video_codec, transcode_decision, stream_bitrate)
		VALUES (1, '1080', 'hevc', 'transcode', 10000)
	`)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to insert session_history_media_info: %v", err)
	}

	detachSQLiteDB(t, db, ctx, "tautulli")
	db.Close()

	// Read the record
	reader, err := NewSQLiteReader(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteReader() error = %v", err)
	}
	defer reader.Close()

	records, err := reader.ReadBatch(context.Background(), 0, 1)
	if err != nil {
		t.Fatalf("ReadBatch() error = %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("ReadBatch() returned %d records, want 1", len(records))
	}

	rec := records[0]

	// Verify core fields
	if rec.ID != 1 {
		t.Errorf("ID = %d, want 1", rec.ID)
	}
	if rec.SessionKey != "test-session" {
		t.Errorf("SessionKey = %s, want test-session", rec.SessionKey)
	}
	if rec.UserID != 42 {
		t.Errorf("UserID = %d, want 42", rec.UserID)
	}
	if rec.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", rec.Username)
	}
	if rec.PercentComplete != 75 {
		t.Errorf("PercentComplete = %d, want 75", rec.PercentComplete)
	}
	if rec.PausedCounter != 120 {
		t.Errorf("PausedCounter = %d, want 120", rec.PausedCounter)
	}
	if rec.LocationType != "lan" {
		t.Errorf("LocationType = %s, want lan", rec.LocationType)
	}

	// Verify timestamps
	expectedStart := time.Unix(started, 0)
	if !rec.StartedAt.Equal(expectedStart) {
		t.Errorf("StartedAt = %v, want %v", rec.StartedAt, expectedStart)
	}
	expectedStop := time.Unix(stopped, 0)
	if !rec.StoppedAt.Equal(expectedStop) {
		t.Errorf("StoppedAt = %v, want %v", rec.StoppedAt, expectedStop)
	}

	// Verify metadata fields
	if rec.MediaType != "episode" {
		t.Errorf("MediaType = %s, want episode", rec.MediaType)
	}
	if rec.Title != "Pilot" {
		t.Errorf("Title = %s, want Pilot", rec.Title)
	}
	if rec.ParentTitle == nil || *rec.ParentTitle != "Season 1" {
		t.Errorf("ParentTitle = %v, want Season 1", rec.ParentTitle)
	}
	if rec.GrandparentTitle == nil || *rec.GrandparentTitle != "Test Show" {
		t.Errorf("GrandparentTitle = %v, want Test Show", rec.GrandparentTitle)
	}

	// Verify stream quality fields
	if rec.VideoResolution == nil || *rec.VideoResolution != "1080" {
		t.Errorf("VideoResolution = %v, want 1080", rec.VideoResolution)
	}
	if rec.TranscodeDecision == nil || *rec.TranscodeDecision != "transcode" {
		t.Errorf("TranscodeDecision = %v, want transcode", rec.TranscodeDecision)
	}
	if rec.StreamBitrate == nil || *rec.StreamBitrate != 10000 {
		t.Errorf("StreamBitrate = %v, want 10000", rec.StreamBitrate)
	}
}

func TestSQLiteReader_GetDateRange(t *testing.T) {
	dbPath, cleanup := createTestDatabase(t)
	defer cleanup()

	insertTestRecords(t, dbPath, 10)

	reader, err := NewSQLiteReader(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteReader() error = %v", err)
	}
	defer reader.Close()

	earliest, latest, err := reader.GetDateRange(context.Background())
	if err != nil {
		t.Fatalf("GetDateRange() error = %v", err)
	}

	// Earliest should be before latest
	if !earliest.Before(latest) {
		t.Errorf("earliest (%v) should be before latest (%v)", earliest, latest)
	}
}

func TestSQLiteReader_GetUserStats(t *testing.T) {
	dbPath, cleanup := createTestDatabase(t)
	defer cleanup()

	insertTestRecords(t, dbPath, 10) // Creates users 1-5 (10 % 5 + 1)

	reader, err := NewSQLiteReader(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteReader() error = %v", err)
	}
	defer reader.Close()

	userCount, err := reader.GetUserStats(context.Background())
	if err != nil {
		t.Fatalf("GetUserStats() error = %v", err)
	}

	if userCount != 5 {
		t.Errorf("GetUserStats() = %d, want 5", userCount)
	}
}

func TestSQLiteReader_GetMediaTypeStats(t *testing.T) {
	dbPath, cleanup := createTestDatabase(t)
	defer cleanup()

	insertTestRecords(t, dbPath, 10) // Creates 3 episodes (i % 3 == 0), 7 movies

	reader, err := NewSQLiteReader(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteReader() error = %v", err)
	}
	defer reader.Close()

	stats, err := reader.GetMediaTypeStats(context.Background())
	if err != nil {
		t.Fatalf("GetMediaTypeStats() error = %v", err)
	}

	// i = 3, 6, 9 are episodes (3 total)
	if stats["episode"] != 3 {
		t.Errorf("episode count = %d, want 3", stats["episode"])
	}

	// Remaining 7 are movies
	if stats["movie"] != 7 {
		t.Errorf("movie count = %d, want 7", stats["movie"])
	}
}
