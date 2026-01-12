// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

//nolint:revive // package name with underscore is intentional for clarity
package tautulli_import

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/tomtom215/cartographus/internal/eventprocessor"
)

// TestDeduplication_CrossSourceScenarios tests cross-source deduplication
// which is critical for handling:
// - Plex webhook events + Tautulli API sync events
// - Import events + existing API sync events
// - Multiple imports of the same/overlapping data
func TestDeduplication_CrossSourceScenarios(t *testing.T) {
	t.Run("same playback from different sources generates matching cross-source keys", func(t *testing.T) {
		// Create import event
		ratingKey := "12345"
		machineID := "device123"
		startTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

		record := &TautulliRecord{
			ID:         1,
			SessionKey: "tautulli-session-123",
			StartedAt:  startTime,
			UserID:     42,
			Username:   "testuser",
			IPAddress:  "192.168.1.100",
			MediaType:  "movie",
			Title:      "Test Movie",
			RatingKey:  &ratingKey,
			MachineID:  &machineID,
		}

		mapper := NewMapper()
		importEvent := mapper.ToPlaybackEvent(record)

		// Create equivalent event from a different source (simulating Plex webhook)
		plexEvent := &eventprocessor.MediaEvent{
			EventID:    "plex-event-abc",   // Different EventID
			SessionKey: "plex-session-xyz", // Different SessionKey
			Source:     "plex",
			UserID:     42,
			Username:   "testuser",
			MediaType:  "movie",
			Title:      "Test Movie",
			RatingKey:  "12345",
			MachineID:  "device123",
			StartedAt:  startTime, // Same start time
		}

		// Generate correlation key for Plex event
		plexEvent.SetCorrelationKey()

		// v2.3: Full correlation keys are intentionally DIFFERENT (include source + session_key)
		// Cross-source deduplication works via GetCrossSourceKey() which extracts the
		// content-based portion (server_id:user_id:rating_key:machine_id:time_bucket)
		if *importEvent.CorrelationKey == plexEvent.CorrelationKey {
			t.Errorf("Full CorrelationKeys should be DIFFERENT in v2.3 format:\nImport: %s\nPlex:   %s",
				*importEvent.CorrelationKey, plexEvent.CorrelationKey)
		}

		// The cross-source keys should match for deduplication
		importCrossKey := eventprocessor.GetCrossSourceKey(*importEvent.CorrelationKey)
		plexCrossKey := eventprocessor.GetCrossSourceKey(plexEvent.CorrelationKey)

		if importCrossKey != plexCrossKey {
			t.Errorf("CrossSourceKey mismatch:\nImport: %s\nPlex:   %s",
				importCrossKey, plexCrossKey)
		}
	})

	t.Run("same playback at same timestamp produces same cross-source key", func(t *testing.T) {
		mapper := NewMapper()

		// Same exact timestamp for the same playback
		startTime := time.Date(2024, 1, 15, 10, 2, 0, 0, time.UTC)
		ratingKey := "12345"
		machineID := "device123"

		// Same playback, different session keys (simulating different sources)
		record1 := &TautulliRecord{
			ID:         1,
			SessionKey: "session1",
			StartedAt:  startTime,
			UserID:     42,
			Username:   "testuser",
			IPAddress:  "192.168.1.100",
			MediaType:  "movie",
			Title:      "Test Movie",
			RatingKey:  &ratingKey,
			MachineID:  &machineID,
		}

		record2 := &TautulliRecord{
			ID:         2,
			SessionKey: "session2",
			StartedAt:  startTime, // Same timestamp
			UserID:     42,
			Username:   "testuser",
			IPAddress:  "192.168.1.100",
			MediaType:  "movie",
			Title:      "Test Movie",
			RatingKey:  &ratingKey,
			MachineID:  &machineID,
		}

		event1 := mapper.ToPlaybackEvent(record1)
		event2 := mapper.ToPlaybackEvent(record2)

		// Full keys should be different (include session_key for uniqueness)
		if *event1.CorrelationKey == *event2.CorrelationKey {
			t.Errorf("Full CorrelationKeys should be different (include session_key):\nEvent1: %s\nEvent2: %s",
				*event1.CorrelationKey, *event2.CorrelationKey)
		}

		// Cross-source keys should match (same content at same timestamp)
		crossKey1 := eventprocessor.GetCrossSourceKey(*event1.CorrelationKey)
		crossKey2 := eventprocessor.GetCrossSourceKey(*event2.CorrelationKey)
		if crossKey1 != crossKey2 {
			t.Errorf("Same playback should have same CrossSourceKey:\nEvent1: %s\nEvent2: %s",
				crossKey1, crossKey2)
		}
	})

	t.Run("different timestamps produce different cross-source keys", func(t *testing.T) {
		mapper := NewMapper()

		baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		ratingKey := "12345"
		machineID := "device123"

		// Event at 10:00:00
		record1 := &TautulliRecord{
			ID:         1,
			SessionKey: "session1",
			StartedAt:  baseTime,
			UserID:     42,
			Username:   "testuser",
			IPAddress:  "192.168.1.100",
			MediaType:  "movie",
			Title:      "Test Movie",
			RatingKey:  &ratingKey,
			MachineID:  &machineID,
		}

		// Event at 10:06:00 (different timestamp = different playback session)
		record2 := &TautulliRecord{
			ID:         2,
			SessionKey: "session2",
			StartedAt:  baseTime.Add(6 * time.Minute),
			UserID:     42,
			Username:   "testuser",
			IPAddress:  "192.168.1.100",
			MediaType:  "movie",
			Title:      "Test Movie",
			RatingKey:  &ratingKey,
			MachineID:  &machineID,
		}

		event1 := mapper.ToPlaybackEvent(record1)
		event2 := mapper.ToPlaybackEvent(record2)

		// Different timestamps should produce different cross-source keys
		// (these are separate playback sessions, not the same playback)
		crossKey1 := eventprocessor.GetCrossSourceKey(*event1.CorrelationKey)
		crossKey2 := eventprocessor.GetCrossSourceKey(*event2.CorrelationKey)
		if crossKey1 == crossKey2 {
			t.Errorf("Different timestamps should produce different CrossSourceKeys:\nEvent1: %s\nEvent2: %s",
				crossKey1, crossKey2)
		}
	})

	t.Run("different users watching same content are distinct", func(t *testing.T) {
		mapper := NewMapper()

		startTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		ratingKey := "12345"
		machineID := "device123"

		// User 1
		record1 := &TautulliRecord{
			ID:         1,
			SessionKey: "session1",
			StartedAt:  startTime,
			UserID:     42,
			Username:   "user1",
			IPAddress:  "192.168.1.100",
			MediaType:  "movie",
			Title:      "Test Movie",
			RatingKey:  &ratingKey,
			MachineID:  &machineID,
		}

		// User 2 (different user, same content, same time)
		record2 := &TautulliRecord{
			ID:         2,
			SessionKey: "session2",
			StartedAt:  startTime,
			UserID:     99,
			Username:   "user2",
			IPAddress:  "192.168.1.101",
			MediaType:  "movie",
			Title:      "Test Movie",
			RatingKey:  &ratingKey,
			MachineID:  &machineID,
		}

		event1 := mapper.ToPlaybackEvent(record1)
		event2 := mapper.ToPlaybackEvent(record2)

		// Different users should have different cross-source keys
		crossKey1 := eventprocessor.GetCrossSourceKey(*event1.CorrelationKey)
		crossKey2 := eventprocessor.GetCrossSourceKey(*event2.CorrelationKey)
		if crossKey1 == crossKey2 {
			t.Errorf("Different users should have different CrossSourceKeys:\nUser1: %s\nUser2: %s",
				crossKey1, crossKey2)
		}
	})

	t.Run("same user different devices are distinct", func(t *testing.T) {
		mapper := NewMapper()

		startTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		ratingKey := "12345"
		machineID1 := "device-living-room"
		machineID2 := "device-bedroom"

		// Device 1
		record1 := &TautulliRecord{
			ID:         1,
			SessionKey: "session1",
			StartedAt:  startTime,
			UserID:     42,
			Username:   "testuser",
			IPAddress:  "192.168.1.100",
			MediaType:  "movie",
			Title:      "Test Movie",
			RatingKey:  &ratingKey,
			MachineID:  &machineID1,
		}

		// Device 2 (same user, same content, same time, different device)
		record2 := &TautulliRecord{
			ID:         2,
			SessionKey: "session2",
			StartedAt:  startTime,
			UserID:     42,
			Username:   "testuser",
			IPAddress:  "192.168.1.101",
			MediaType:  "movie",
			Title:      "Test Movie",
			RatingKey:  &ratingKey,
			MachineID:  &machineID2,
		}

		event1 := mapper.ToPlaybackEvent(record1)
		event2 := mapper.ToPlaybackEvent(record2)

		// Different devices should have different cross-source keys
		crossKey1 := eventprocessor.GetCrossSourceKey(*event1.CorrelationKey)
		crossKey2 := eventprocessor.GetCrossSourceKey(*event2.CorrelationKey)
		if crossKey1 == crossKey2 {
			t.Errorf("Same user on different devices should have different CrossSourceKeys:\nDevice1: %s\nDevice2: %s",
				crossKey1, crossKey2)
		}
	})
}

// TestReimport_SameDatabase tests that reimporting the same database file
// produces identical EventIDs and CorrelationKeys.
func TestReimport_SameDatabase(t *testing.T) {
	dbPath, cleanup := createTestDatabaseWithKnownData(t)
	defer cleanup()

	cfg := createImportConfig(dbPath)
	publisher1 := newMockEventPublisher()
	importer1 := NewImporter(cfg, publisher1, nil)

	// First import
	stats1, err := importer1.Import(context.Background())
	if err != nil {
		t.Fatalf("First import error: %v", err)
	}

	events1 := publisher1.getEvents()
	eventIDs1 := make(map[string]string)
	correlationKeys1 := make(map[string]string)
	for _, e := range events1 {
		eventIDs1[e.SessionKey] = e.EventID
		correlationKeys1[e.SessionKey] = e.CorrelationKey
	}

	// Second import with fresh importer
	publisher2 := newMockEventPublisher()
	importer2 := NewImporter(cfg, publisher2, nil)

	stats2, err := importer2.Import(context.Background())
	if err != nil {
		t.Fatalf("Second import error: %v", err)
	}

	events2 := publisher2.getEvents()

	// Verify same number of events
	if len(events1) != len(events2) {
		t.Fatalf("Different event counts: %d vs %d", len(events1), len(events2))
	}

	// Verify same stats
	if stats1.Imported != stats2.Imported {
		t.Errorf("Different imported counts: %d vs %d", stats1.Imported, stats2.Imported)
	}

	// Verify deterministic IDs
	for _, e2 := range events2 {
		id1, ok := eventIDs1[e2.SessionKey]
		if !ok {
			t.Errorf("Session %s not found in first import", e2.SessionKey)
			continue
		}
		if id1 != e2.EventID {
			t.Errorf("Session %s: EventID mismatch: %s vs %s", e2.SessionKey, id1, e2.EventID)
		}

		corr1 := correlationKeys1[e2.SessionKey]
		if corr1 != e2.CorrelationKey {
			t.Errorf("Session %s: CorrelationKey mismatch: %s vs %s", e2.SessionKey, corr1, e2.CorrelationKey)
		}
	}
}

// TestReimport_NewerBackup tests importing a newer backup that contains
// some overlapping records and some new records.
func TestReimport_NewerBackup(t *testing.T) {
	// Create first database with records 1-5
	dbPath1, cleanup1 := createTestDatabaseWithRecordRange(t, 1, 5)
	defer cleanup1()

	cfg := createImportConfig(dbPath1)
	publisher1 := newMockEventPublisher()
	importer1 := NewImporter(cfg, publisher1, nil)

	// Import first backup
	_, err := importer1.Import(context.Background())
	if err != nil {
		t.Fatalf("First import error: %v", err)
	}

	events1 := publisher1.getEvents()
	originalEventIDs := make(map[int64]string)
	for _, e := range events1 {
		// Extract the record ID from SessionKey (format: "session-N")
		var id int64
		fmt.Sscanf(e.SessionKey, "session-%d", &id)
		originalEventIDs[id] = e.EventID
	}

	// Create second database with records 3-10 (overlaps 3-5, new 6-10)
	dbPath2, cleanup2 := createTestDatabaseWithRecordRange(t, 3, 10)
	defer cleanup2()

	cfg.DBPath = dbPath2
	publisher2 := newMockEventPublisher()
	importer2 := NewImporter(cfg, publisher2, nil)

	// Import second backup
	_, err = importer2.Import(context.Background())
	if err != nil {
		t.Fatalf("Second import error: %v", err)
	}

	events2 := publisher2.getEvents()

	// Verify overlapping records have same EventIDs
	for _, e2 := range events2 {
		var id int64
		fmt.Sscanf(e2.SessionKey, "session-%d", &id)

		if id <= 5 { // Overlapping record
			if origID, ok := originalEventIDs[id]; ok {
				if origID != e2.EventID {
					t.Errorf("Overlapping record %d: EventID changed from %s to %s",
						id, origID, e2.EventID)
				}
			}
		}
	}
}

// TestCorrelationKey_MissingFields tests correlation key generation
// when optional fields are missing.
func TestCorrelationKey_MissingFields(t *testing.T) {
	mapper := NewMapper()

	t.Run("missing rating key uses title hash", func(t *testing.T) {
		record := &TautulliRecord{
			ID:         1,
			SessionKey: "session1",
			StartedAt:  time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			UserID:     42,
			Username:   "testuser",
			IPAddress:  "192.168.1.100",
			MediaType:  "movie",
			Title:      "Test Movie",
			RatingKey:  nil, // Missing rating key
		}

		event := mapper.ToPlaybackEvent(record)

		if event.CorrelationKey == nil || *event.CorrelationKey == "" {
			t.Error("CorrelationKey should be generated even without rating key")
		}

		// v2.3 format: Should contain a hex hash instead of rating key
		// The hash is the first 16 chars of SHA256(title)
		key := *event.CorrelationKey
		if !strings.HasPrefix(key, "tautulli-import:default:42:") {
			t.Errorf("CorrelationKey should have correct prefix: %s", key)
		}
	})

	t.Run("missing machine ID uses unknown", func(t *testing.T) {
		ratingKey := "12345"
		record := &TautulliRecord{
			ID:         1,
			SessionKey: "session1",
			StartedAt:  time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			UserID:     42,
			Username:   "testuser",
			IPAddress:  "192.168.1.100",
			MediaType:  "movie",
			Title:      "Test Movie",
			RatingKey:  &ratingKey,
			MachineID:  nil, // Missing machine ID
		}

		event := mapper.ToPlaybackEvent(record)

		if event.CorrelationKey == nil {
			t.Fatal("CorrelationKey should be generated")
		}

		// v2.3 format: {source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}:{session_key}
		// Should contain "unknown" for machine ID
		expected := "tautulli-import:default:42:12345:unknown:2024-01-15T10:00:00:session1"
		if *event.CorrelationKey != expected {
			t.Errorf("CorrelationKey = %s, want %s", *event.CorrelationKey, expected)
		}
	})

	t.Run("empty rating key string uses title hash", func(t *testing.T) {
		emptyRatingKey := ""
		record := &TautulliRecord{
			ID:         1,
			SessionKey: "session1",
			StartedAt:  time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			UserID:     42,
			Username:   "testuser",
			IPAddress:  "192.168.1.100",
			MediaType:  "movie",
			Title:      "Test Movie",
			RatingKey:  &emptyRatingKey, // Empty string rating key
		}

		event := mapper.ToPlaybackEvent(record)

		if event.CorrelationKey == nil || *event.CorrelationKey == "" {
			t.Error("CorrelationKey should be generated with title hash")
		}
	})
}

// TestEventID_Determinism tests that EventID generation is deterministic
// and does not change between runs.
func TestEventID_Determinism(t *testing.T) {
	mapper := NewMapper()

	// Fixed values for deterministic testing
	record := &TautulliRecord{
		ID:         12345,
		SessionKey: "fixed-session-key",
		StartedAt:  time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
		UserID:     42,
		Username:   "testuser",
		IPAddress:  "192.168.1.100",
		MediaType:  "movie",
		Title:      "Test Movie",
	}

	// Generate event multiple times
	events := make([]*eventprocessor.MediaEvent, 10)
	for i := 0; i < 10; i++ {
		pe := mapper.ToPlaybackEvent(record)
		events[i] = playbackEventToMediaEvent(pe)
	}

	// All should have same EventID
	firstID := events[0].EventID
	for i, e := range events {
		if e.EventID != firstID {
			t.Errorf("Run %d: EventID %s != first EventID %s", i, e.EventID, firstID)
		}
	}
}

// Helper functions

func createTestDatabaseWithKnownData(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "import-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "tautulli.db")

	// Create DuckDB connection to create SQLite database
	db, err := sql.Open("duckdb", "")
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to open duckdb: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Install and load sqlite_scanner
	db.ExecContext(ctx, "INSTALL sqlite_scanner;")
	db.ExecContext(ctx, "LOAD sqlite_scanner;")

	// Attach SQLite database
	if _, err := db.ExecContext(ctx, fmt.Sprintf("ATTACH '%s' AS tautulli (TYPE SQLITE)", dbPath)); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to attach database: %v", err)
	}

	// Create tables
	createTautulliTables(t, db, ctx)

	// Insert known data
	started := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC).Unix()
	stopped := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix()

	for i := 1; i <= 5; i++ {
		_, err := db.ExecContext(ctx, `
			INSERT INTO tautulli.session_history (id, session_key, started, stopped, user_id, user, ip_address, platform, player, percent_complete, paused_counter, location)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, i, fmt.Sprintf("known-session-%d", i), started+int64(i*3600), stopped+int64(i*3600),
			42, "testuser", "192.168.1.100", "Chrome", "Plex Web", 100, 0, "lan")
		if err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("Failed to insert session_history: %v", err)
		}

		_, err = db.ExecContext(ctx, `
			INSERT INTO tautulli.session_history_metadata (id, media_type, title, year)
			VALUES (?, ?, ?, ?)
		`, i, "movie", fmt.Sprintf("Test Movie %d", i), 2024)
		if err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("Failed to insert metadata: %v", err)
		}

		_, err = db.ExecContext(ctx, `
			INSERT INTO tautulli.session_history_media_info (id, video_resolution, video_codec, transcode_decision)
			VALUES (?, ?, ?, ?)
		`, i, "1080", "h264", "direct play")
		if err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("Failed to insert media_info: %v", err)
		}
	}

	db.ExecContext(ctx, "DETACH tautulli")

	return dbPath, func() { os.RemoveAll(tmpDir) }
}

func createTestDatabaseWithRecordRange(t *testing.T, start, end int) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "import-range-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "tautulli.db")

	db, err := sql.Open("duckdb", "")
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to open duckdb: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	db.ExecContext(ctx, "INSTALL sqlite_scanner;")
	db.ExecContext(ctx, "LOAD sqlite_scanner;")

	if _, err := db.ExecContext(ctx, fmt.Sprintf("ATTACH '%s' AS tautulli (TYPE SQLITE)", dbPath)); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to attach database: %v", err)
	}

	createTautulliTables(t, db, ctx)

	// Use deterministic timestamps based on record ID
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	for i := start; i <= end; i++ {
		started := baseTime.Add(time.Duration(i) * time.Hour).Unix()
		stopped := started + 7200

		_, err := db.ExecContext(ctx, `
			INSERT INTO tautulli.session_history (id, session_key, started, stopped, user_id, user, ip_address, platform, player, percent_complete, paused_counter, location)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, i, fmt.Sprintf("session-%d", i), started, stopped,
			42, "testuser", "192.168.1.100", "Chrome", "Plex Web", 100, 0, "lan")
		if err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("Failed to insert session_history: %v", err)
		}

		_, err = db.ExecContext(ctx, `
			INSERT INTO tautulli.session_history_metadata (id, media_type, title, year)
			VALUES (?, ?, ?, ?)
		`, i, "movie", fmt.Sprintf("Movie %d", i), 2024)
		if err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("Failed to insert metadata: %v", err)
		}

		_, err = db.ExecContext(ctx, `
			INSERT INTO tautulli.session_history_media_info (id, video_resolution, video_codec, transcode_decision)
			VALUES (?, ?, ?, ?)
		`, i, "1080", "h264", "direct play")
		if err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("Failed to insert media_info: %v", err)
		}
	}

	db.ExecContext(ctx, "DETACH tautulli")

	return dbPath, func() { os.RemoveAll(tmpDir) }
}

func createTautulliTables(t *testing.T, db *sql.DB, ctx context.Context) {
	t.Helper()

	// Create session_history table - matches Tautulli's actual schema
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS tautulli.session_history (
			id INTEGER PRIMARY KEY,
			session_key TEXT NOT NULL,
			started INTEGER NOT NULL,
			stopped INTEGER,
			user_id INTEGER NOT NULL,
			user TEXT NOT NULL,
			friendly_name TEXT,
			ip_address TEXT,
			platform TEXT,
			player TEXT,
			product TEXT,
			percent_complete INTEGER DEFAULT 0,
			paused_counter INTEGER DEFAULT 0,
			location TEXT,
			machine_id TEXT,
			rating_key TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create session_history: %v", err)
	}

	// Create session_history_metadata table
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS tautulli.session_history_metadata (
			id INTEGER PRIMARY KEY,
			media_type TEXT,
			title TEXT,
			parent_title TEXT,
			grandparent_title TEXT,
			year INTEGER,
			rating_key TEXT,
			parent_rating_key TEXT,
			grandparent_rating_key TEXT,
			guid TEXT,
			section_id INTEGER,
			library_name TEXT,
			content_rating TEXT,
			thumb TEXT,
			parent_thumb TEXT,
			grandparent_thumb TEXT,
			directors TEXT,
			writers TEXT,
			actors TEXT,
			genres TEXT,
			studio TEXT,
			full_title TEXT,
			original_title TEXT,
			originally_available_at TEXT,
			media_index INTEGER,
			parent_media_index INTEGER
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create session_history_metadata: %v", err)
	}

	// Create session_history_media_info table
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS tautulli.session_history_media_info (
			id INTEGER PRIMARY KEY,
			video_resolution TEXT,
			video_full_resolution TEXT,
			video_codec TEXT,
			audio_codec TEXT,
			audio_channels TEXT,
			container TEXT,
			bitrate INTEGER,
			stream_bitrate INTEGER,
			transcode_decision TEXT,
			video_decision TEXT,
			audio_decision TEXT,
			subtitle_decision TEXT,
			stream_video_codec TEXT,
			stream_video_resolution TEXT,
			stream_audio_codec TEXT,
			stream_audio_channels TEXT,
			stream_video_dynamic_range TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create session_history_media_info: %v", err)
	}
}
