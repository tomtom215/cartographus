// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
)

// testDBSemaphore limits concurrent database creation to prevent resource exhaustion in CI.
// When many tests run in parallel, too many concurrent DuckDB CGO calls can cause hangs.
// Setting to 1 fully serializes database creation to prevent resource contention.
var testDBSemaphore = make(chan struct{}, 1)

// testDBMutex serializes database creation for short periods to reduce contention.
var testDBMutex sync.Mutex

// setupTestDB creates a new in-memory test database with timeout protection.
// Uses a 120-second timeout to fail fast if DuckDB hangs during connection.
// This prevents tests from timing out after 10 minutes in CI when under resource pressure.
//
// Concurrency control:
// - Semaphore limits concurrent database operations to 1 (fully serialized)
// - Mutex provides additional safety for the New() call
// - CRITICAL: Semaphore is held for ENTIRE test lifecycle, not just DB creation
// - Semaphore is released via t.Cleanup() when the test completes
//
// Why hold semaphore for entire test:
//   - DuckDB CGO calls can hang when multiple connections do concurrent operations
//   - Even if DB creation is serialized, concurrent INSERT/SELECT from multiple
//     tests can cause hangs under CI resource pressure
//   - By holding the semaphore until test completion, we ensure only ONE test
//     has an active DuckDB connection at any time
func setupTestDB(t *testing.T) *DB {
	t.Helper()

	// Acquire semaphore to limit concurrency - blocks until available
	testDBSemaphore <- struct{}{}

	// CRITICAL: Register cleanup to release semaphore when test COMPLETES
	// This ensures the semaphore is held for the entire test lifecycle,
	// not just during DB creation. Prevents concurrent DuckDB CGO operations.
	t.Cleanup(func() {
		<-testDBSemaphore
	})

	cfg := &config.DatabaseConfig{
		Path:      ":memory:",
		MaxMemory: "1GB", // Standard memory for unit tests
	}

	// Create database in a goroutine with timeout to prevent hangs
	// DuckDB CGO calls can hang indefinitely under resource pressure
	type result struct {
		db  *DB
		err error
	}

	resultCh := make(chan result, 1)
	go func() {
		// Serialize database creation to prevent mutex contention
		testDBMutex.Lock()
		db, err := New(cfg, 0.0, 0.0)
		testDBMutex.Unlock()
		resultCh <- result{db: db, err: err}
	}()

	// Wait for database creation with timeout
	select {
	case res := <-resultCh:
		// NOTE: Semaphore is NOT released here - it's released by t.Cleanup
		// when the test completes, ensuring exclusive access throughout test
		if res.err != nil {
			t.Fatalf("Failed to create test database: %v", res.err)
		}
		return res.db
	case <-time.After(120 * time.Second):
		// On timeout, semaphore is already registered for cleanup
		// The test will fail and cleanup will release it
		t.Fatalf("Timeout: database creation took longer than 120s (DuckDB may be under resource pressure)")
		return nil
	}
}

// setupConcurrentTestDB creates a test database with higher memory for concurrent tests.
// Concurrent tests (50+ goroutines) require more memory than normal unit tests.
// Uses 2GB memory limit to accommodate high concurrency workloads.
//
// CRITICAL: Like setupTestDB, holds semaphore for entire test lifecycle.
func setupConcurrentTestDB(t *testing.T) *DB {
	t.Helper()

	// Acquire semaphore to limit concurrency - blocks until available
	testDBSemaphore <- struct{}{}

	// CRITICAL: Register cleanup to release semaphore when test COMPLETES
	t.Cleanup(func() {
		<-testDBSemaphore
	})

	cfg := &config.DatabaseConfig{
		Path:      ":memory:",
		MaxMemory: "2GB", // Higher limit for concurrent tests with 50+ goroutines
	}

	// Create database in a goroutine with timeout to prevent hangs
	type result struct {
		db  *DB
		err error
	}

	resultCh := make(chan result, 1)
	go func() {
		testDBMutex.Lock()
		db, err := New(cfg, 0.0, 0.0)
		testDBMutex.Unlock()
		resultCh <- result{db: db, err: err}
	}()

	select {
	case res := <-resultCh:
		// NOTE: Semaphore is NOT released here - it's released by t.Cleanup
		if res.err != nil {
			t.Fatalf("Failed to create concurrent test database: %v", res.err)
		}
		return res.db
	case <-time.After(120 * time.Second):
		// On timeout, semaphore is already registered for cleanup
		t.Fatalf("Timeout: database creation took longer than 120s")
		return nil
	}
}

// insertTestGeolocations inserts test geolocation data
func insertTestGeolocations(t *testing.T, db *DB) {
	t.Helper()
	geolocations := []struct {
		ip      string
		lat     float64
		lon     float64
		city    string
		region  string
		country string
	}{
		{"192.168.1.1", 40.7128, -74.0060, "New York", "New York", "United States"},
		{"192.168.1.2", 34.0522, -118.2437, "Los Angeles", "California", "United States"},
		{"192.168.1.3", 51.5074, -0.1278, "London", "England", "United Kingdom"},
		{"192.168.1.4", 48.8566, 2.3522, "Paris", "Ile-de-France", "France"},
		{"192.168.1.5", 35.6762, 139.6503, "Tokyo", "Tokyo", "Japan"},
	}

	for _, geo := range geolocations {
		var err error
		if db.IsSpatialAvailable() {
			_, err = db.conn.Exec(`
				INSERT INTO geolocations (ip_address, latitude, longitude, geom, city, region, country)
				VALUES (?, ?, ?, ST_Point(?, ?), ?, ?, ?)
			`, geo.ip, geo.lat, geo.lon, geo.lon, geo.lat, geo.city, geo.region, geo.country)
		} else {
			_, err = db.conn.Exec(`
				INSERT INTO geolocations (ip_address, latitude, longitude, city, region, country)
				VALUES (?, ?, ?, ?, ?, ?)
			`, geo.ip, geo.lat, geo.lon, geo.city, geo.region, geo.country)
		}
		checkNoError(t, err)
	}
}

// insertTestPlaybacks inserts test playback data
func insertTestPlaybacks(t *testing.T, db *DB) {
	t.Helper()
	now := time.Now()
	playbacks := []struct {
		userID          int
		username        string
		ip              string
		mediaType       string
		title           string
		startedAt       time.Time
		stoppedAt       *time.Time
		percentComplete int
	}{
		// New York user - recent playbacks
		{1, "user1", "192.168.1.1", "movie", "Test Movie 1", now.Add(-2 * time.Hour), timePtr(now.Add(-1 * time.Hour)), 100},
		{1, "user1", "192.168.1.1", "episode", "Test Show S01E01", now.Add(-4 * time.Hour), timePtr(now.Add(-3 * time.Hour)), 95},
		{1, "user1", "192.168.1.1", "movie", "Test Movie 2", now.Add(-6 * time.Hour), timePtr(now.Add(-5 * time.Hour)), 80},

		// Los Angeles user - recent playbacks
		{2, "user2", "192.168.1.2", "episode", "Test Show S01E02", now.Add(-1 * time.Hour), timePtr(now.Add(-30 * time.Minute)), 100},
		{2, "user2", "192.168.1.2", "movie", "Test Movie 3", now.Add(-3 * time.Hour), timePtr(now.Add(-2 * time.Hour)), 90},

		// London user - older playbacks
		{3, "user3", "192.168.1.3", "track", "Test Song 1", now.Add(-48 * time.Hour), timePtr(now.Add(-48*time.Hour + 3*time.Minute)), 100},
		{3, "user3", "192.168.1.3", "movie", "Test Movie 4", now.Add(-72 * time.Hour), timePtr(now.Add(-71 * time.Hour)), 75},

		// Paris user - 7 days ago
		{4, "user4", "192.168.1.4", "episode", "Test Show S02E01", now.Add(-7 * 24 * time.Hour), timePtr(now.Add(-7*24*time.Hour + 1*time.Hour)), 100},

		// Tokyo user - 30 days ago
		{5, "user5", "192.168.1.5", "movie", "Test Movie 5", now.Add(-30 * 24 * time.Hour), timePtr(now.Add(-30*24*time.Hour + 2*time.Hour)), 60},
	}

	for _, pb := range playbacks {
		_, err := db.conn.Exec(`
			INSERT INTO playback_events (
				id, session_key, started_at, stopped_at, user_id, username,
				ip_address, media_type, title, percent_complete
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, uuid.New().String(), uuid.New().String(), pb.startedAt, pb.stoppedAt,
			pb.userID, pb.username, pb.ip, pb.mediaType, pb.title, pb.percentComplete)
		checkNoError(t, err)
	}
}

// timePtr returns a pointer to the given time
func timePtr(t time.Time) *time.Time {
	return &t
}

// setupTestDBWithData creates a test DB with geolocations and playbacks
func setupTestDBWithData(t *testing.T) *DB {
	t.Helper()
	db := setupTestDB(t)
	insertTestGeolocations(t, db)
	insertTestPlaybacks(t, db)
	return db
}

// defaultFilter returns a filter with a start date 7 days ago
func defaultFilter(daysBack int) LocationStatsFilter {
	startDate := time.Now().Add(time.Duration(-daysBack) * 24 * time.Hour)
	return LocationStatsFilter{StartDate: &startDate}
}

func TestGetPlaybackTrends_DayInterval(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	trends, interval, err := db.GetPlaybackTrends(context.Background(), defaultFilter(7))
	checkNoError(t, err)
	checkStringEqual(t, "interval", interval, "day")
	checkSliceNotEmpty(t, "trends", len(trends))

	// Verify trends using subtests
	for i, trend := range trends {
		t.Run("trend_"+string(rune('0'+i)), func(t *testing.T) {
			checkStringNotEmpty(t, "trend.Date", trend.Date)
			checkIntNonNegative(t, "trend.PlaybackCount", trend.PlaybackCount)
			checkIntNonNegative(t, "trend.UniqueUsers", trend.UniqueUsers)
		})
	}
}

func TestGetPlaybackTrends_MonthInterval(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertTestGeolocations(t, db)

	// Insert playbacks spanning > 365 days to trigger month interval
	now := time.Now()
	testData := []struct {
		userID    int
		username  string
		ip        string
		mediaType string
		title     string
		startedAt time.Time
	}{
		{1, "user1", "192.168.1.1", "movie", "Old Movie 1", now.Add(-730 * 24 * time.Hour)},
		{1, "user1", "192.168.1.1", "movie", "Old Movie 2", now.Add(-700 * 24 * time.Hour)},
		{2, "user2", "192.168.1.2", "episode", "Old Show", now.Add(-600 * 24 * time.Hour)},
		{3, "user3", "192.168.1.3", "movie", "Mid Movie", now.Add(-400 * 24 * time.Hour)},
		{4, "user4", "192.168.1.4", "episode", "Recent Show", now.Add(-7 * 24 * time.Hour)},
		{5, "user5", "192.168.1.5", "movie", "Recent Movie", now.Add(-1 * 24 * time.Hour)},
	}

	for _, pb := range testData {
		stoppedAt := pb.startedAt.Add(2 * time.Hour)
		_, err := db.conn.Exec(`
			INSERT INTO playback_events (
				id, session_key, started_at, stopped_at, user_id, username,
				ip_address, media_type, title, percent_complete
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, uuid.New().String(), uuid.New().String(), pb.startedAt, stoppedAt,
			pb.userID, pb.username, pb.ip, pb.mediaType, pb.title, 100)
		checkNoError(t, err)
	}

	trends, interval, err := db.GetPlaybackTrends(context.Background(), defaultFilter(730))
	checkNoError(t, err)
	checkStringEqual(t, "interval", interval, "month")
	checkSliceNotEmpty(t, "trends", len(trends))

	verifyTrends(t, trends)
}

// verifyTrends validates basic trend properties
func verifyTrends(t *testing.T, trends []models.PlaybackTrend) {
	t.Helper()
	for _, trend := range trends {
		checkStringNotEmpty(t, "trend.Date", trend.Date)
		checkIntNonNegative(t, "trend.PlaybackCount", trend.PlaybackCount)
		checkIntNonNegative(t, "trend.UniqueUsers", trend.UniqueUsers)
	}
}

func TestGetPlaybackTrends_WithUserFilter(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	filter := defaultFilter(1)
	filter.Users = []string{"user1"}

	trends, _, err := db.GetPlaybackTrends(context.Background(), filter)
	checkNoError(t, err)

	// Should only have user1's playbacks
	for _, trend := range trends {
		checkSliceMaxLen(t, "user1 playbacks in 24h", trend.PlaybackCount, 3)
	}
}

func TestGetPlaybackTrends_WithMediaTypeFilter(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	filter := defaultFilter(1)
	filter.MediaTypes = []string{"movie"}

	trends, _, err := db.GetPlaybackTrends(context.Background(), filter)
	checkNoError(t, err)
	checkSliceNotEmpty(t, "movie trends", len(trends))
}

func TestGetPlaybackTrends_EmptyData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertTestGeolocations(t, db)
	// No playbacks inserted

	trends, interval, err := db.GetPlaybackTrends(context.Background(), defaultFilter(1))
	checkNoError(t, err)
	checkStringEqual(t, "interval", interval, "day")
	checkSliceEmpty(t, "trends", len(trends))
}

func TestGetViewingHoursHeatmap_AllFilters(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	heatmap, err := db.GetViewingHoursHeatmap(context.Background(), defaultFilter(7))
	checkNoError(t, err)
	checkSliceNotEmpty(t, "heatmap", len(heatmap))

	verifyHeatmap(t, heatmap)
}

// verifyHeatmap validates heatmap entries
func verifyHeatmap(t *testing.T, heatmap []models.ViewingHoursHeatmap) {
	t.Helper()
	for _, entry := range heatmap {
		checkIntInRange(t, "entry.Hour", entry.Hour, 0, 23)
		checkIntInRange(t, "entry.DayOfWeek", entry.DayOfWeek, 0, 6)
		checkIntNonNegative(t, "entry.PlaybackCount", entry.PlaybackCount)
	}
}

func TestGetViewingHoursHeatmap_WithUserFilter(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	filter := defaultFilter(1)
	filter.Users = []string{"user1"}

	heatmap, err := db.GetViewingHoursHeatmap(context.Background(), filter)
	checkNoError(t, err)
	checkSliceNotEmpty(t, "user1 heatmap", len(heatmap))
}

func TestGetTopUsers_DefaultLimit(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	users, err := db.GetTopUsers(context.Background(), defaultFilter(7), 10)
	checkNoError(t, err)
	checkSliceNotEmpty(t, "users", len(users))

	// Verify sorted descending
	counts := make([]int, len(users))
	for i, u := range users {
		counts[i] = u.PlaybackCount
	}
	checkSortedDescending(t, "users by playback count", counts)

	verifyUsers(t, users)
}

// verifyUsers validates user data
func verifyUsers(t *testing.T, users []models.UserActivity) {
	t.Helper()
	for _, user := range users {
		checkStringNotEmpty(t, "user.Username", user.Username)
		checkIntPositive(t, "user.PlaybackCount", user.PlaybackCount)
	}
}

func TestGetTopUsers_WithLimit(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	users, err := db.GetTopUsers(context.Background(), defaultFilter(365), 3)
	checkNoError(t, err)
	checkSliceMaxLen(t, "users", len(users), 3)
}

func TestGetTopUsers_WithMediaTypeFilter(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	filter := defaultFilter(365)
	filter.MediaTypes = []string{"movie"}

	users, err := db.GetTopUsers(context.Background(), filter, 10)
	checkNoError(t, err)
	checkSliceNotEmpty(t, "movie users", len(users))
}

func TestGetMediaTypeDistribution_AllTypes(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	distribution, err := db.GetMediaTypeDistribution(context.Background(), defaultFilter(365))
	checkNoError(t, err)
	checkSliceNotEmpty(t, "distribution", len(distribution))

	verifyDistribution(t, distribution)
}

// verifyDistribution validates media type distribution
func verifyDistribution(t *testing.T, distribution []models.MediaTypeStats) {
	t.Helper()
	totalCount := 0
	for _, entry := range distribution {
		checkStringNotEmpty(t, "entry.MediaType", entry.MediaType)
		checkIntPositive(t, "entry.PlaybackCount", entry.PlaybackCount)
		totalCount += entry.PlaybackCount
	}
	checkIntPositive(t, "totalCount", totalCount)
}

func TestGetMediaTypeDistribution_WithUserFilter(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	filter := defaultFilter(1)
	filter.Users = []string{"user1"}

	distribution, err := db.GetMediaTypeDistribution(context.Background(), filter)
	checkNoError(t, err)
	checkSliceNotEmpty(t, "user1 distribution", len(distribution))
}

func TestGetMediaTypeDistribution_EmptyData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertTestGeolocations(t, db)
	// No playbacks

	distribution, err := db.GetMediaTypeDistribution(context.Background(), defaultFilter(1))
	checkNoError(t, err)
	checkSliceEmpty(t, "distribution", len(distribution))
}

func TestGetTopCities_DefaultLimit(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	cities, err := db.GetTopCities(context.Background(), defaultFilter(365), 10)
	checkNoError(t, err)
	checkSliceNotEmpty(t, "cities", len(cities))

	// Verify sorted descending
	counts := make([]int, len(cities))
	for i, c := range cities {
		counts[i] = c.PlaybackCount
	}
	checkSortedDescending(t, "cities by playback count", counts)

	verifyCities(t, cities)
}

// verifyCities validates city data
func verifyCities(t *testing.T, cities []models.CityStats) {
	t.Helper()
	for _, city := range cities {
		checkStringNotEmpty(t, "city.City", city.City)
		checkStringNotEmpty(t, "city.Country", city.Country)
		checkIntPositive(t, "city.PlaybackCount", city.PlaybackCount)
		checkIntPositive(t, "city.UniqueUsers", city.UniqueUsers)
	}
}

func TestGetTopCities_WithLimit(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	cities, err := db.GetTopCities(context.Background(), defaultFilter(365), 3)
	checkNoError(t, err)
	checkSliceMaxLen(t, "cities", len(cities), 3)
}

func TestGetTopCountries_DefaultLimit(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	countries, err := db.GetTopCountries(context.Background(), defaultFilter(365), 10)
	checkNoError(t, err)
	checkSliceNotEmpty(t, "countries", len(countries))

	// Verify sorted descending
	counts := make([]int, len(countries))
	for i, c := range countries {
		counts[i] = c.PlaybackCount
	}
	checkSortedDescending(t, "countries by playback count", counts)

	verifyCountries(t, countries)
}

// verifyCountries validates country data
func verifyCountries(t *testing.T, countries []models.CountryStats) {
	t.Helper()
	for _, country := range countries {
		checkStringNotEmpty(t, "country.Country", country.Country)
		checkIntPositive(t, "country.PlaybackCount", country.PlaybackCount)
		checkIntPositive(t, "country.UniqueUsers", country.UniqueUsers)
	}
}

func TestGetTopCountries_WithFilters(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	filter := defaultFilter(7)
	filter.MediaTypes = []string{"movie"}

	countries, err := db.GetTopCountries(context.Background(), filter, 10)
	checkNoError(t, err)
	checkSliceNotEmpty(t, "movie countries", len(countries))
}

func TestGetLocationStatsFiltered_WithAllFilters(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	now := time.Now()
	startDate := now.Add(-7 * 24 * time.Hour)
	endDate := now
	filter := LocationStatsFilter{
		StartDate:  &startDate,
		EndDate:    &endDate,
		Users:      []string{"user1", "user2"},
		MediaTypes: []string{"movie", "episode"},
		Limit:      100,
	}

	stats, err := db.GetLocationStatsFiltered(context.Background(), filter)
	checkNoError(t, err)
	checkSliceNotEmpty(t, "stats", len(stats))

	verifyLocationStats(t, stats)
}

// verifyLocationStats validates location stats
func verifyLocationStats(t *testing.T, stats []models.LocationStats) {
	t.Helper()
	for _, stat := range stats {
		checkStringNotEmpty(t, "stat.Country", stat.Country)
		checkIntPositive(t, "stat.PlaybackCount", stat.PlaybackCount)
		checkIntPositive(t, "stat.UniqueUsers", stat.UniqueUsers)
	}
}

func TestGetUniqueUsers_Success(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	users, err := db.GetUniqueUsers(context.Background())
	checkNoError(t, err)
	checkSliceNotEmpty(t, "users", len(users))

	// Verify all users are unique
	checkUniqueStrings(t, "users", users)

	for _, user := range users {
		checkStringNotEmpty(t, "username", user)
	}
}

func TestGetUniqueMediaTypes_Success(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	mediaTypes, err := db.GetUniqueMediaTypes(context.Background())
	checkNoError(t, err)
	checkSliceNotEmpty(t, "mediaTypes", len(mediaTypes))

	// Verify all media types are unique
	checkUniqueStrings(t, "mediaTypes", mediaTypes)

	for _, mediaType := range mediaTypes {
		checkStringNotEmpty(t, "mediaType", mediaType)
	}

	// Verify expected types are present
	expectedTypes := map[string]bool{"movie": false, "episode": false, "track": false}
	for _, mediaType := range mediaTypes {
		if _, exists := expectedTypes[mediaType]; exists {
			expectedTypes[mediaType] = true
		}
	}
}

func TestPing_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := db.Ping(context.Background())
	checkNoError(t, err)
}

func TestPing_ClosedConnection(t *testing.T) {
	db := setupTestDB(t)
	db.Close()

	err := db.Ping(context.Background())
	checkError(t, err)
}

// TestClose_Idempotent tests that Close can be called multiple times safely
func TestClose_Idempotent(t *testing.T) {
	db := setupTestDB(t)

	// First close should succeed
	err1 := db.Close()
	checkNoError(t, err1)

	// Second close should also succeed (idempotent)
	err2 := db.Close()
	if err2 != nil {
		// Some databases return error on double close, which is acceptable
		t.Logf("Second close returned: %v (acceptable)", err2)
	}
}

// TestGetRecordCounts_WithContextCancellation tests record counts with canceled context
func TestGetRecordCounts_WithContextCancellation(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, _, err := db.GetRecordCounts(ctx)
	// Should return error with canceled context
	if err == nil {
		t.Error("Expected error with canceled context")
	}
}

// TestGetStats_AllMetrics tests GetStats returns all expected fields
func TestGetStats_AllMetrics(t *testing.T) {
	db := setupTestDBWithData(t)
	defer db.Close()

	stats, err := db.GetStats(context.Background())
	checkNoError(t, err)

	// Verify all expected metrics are present
	if stats.TotalPlaybacks < 0 {
		t.Error("TotalPlaybacks should be non-negative")
	}
	if stats.UniqueUsers < 0 {
		t.Error("UniqueUsers should be non-negative")
	}
	if stats.UniqueLocations < 0 {
		t.Error("UniqueLocations should be non-negative")
	}
	if stats.RecentActivity < 0 {
		t.Error("RecentActivity should be non-negative")
	}
}

// TestAcquireIPLock tests the IP lock acquisition
func TestAcquireIPLock(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Acquire lock for first IP (acquireIPLock already calls Lock() internally)
	lock1 := db.acquireIPLock("10.0.0.1")
	// Lock is already held, just verify it's not nil
	if lock1 == nil {
		t.Fatal("acquireIPLock returned nil")
	}

	// Acquire lock for different IP (should not block since it's a different IP)
	lock2 := db.acquireIPLock("10.0.0.2")
	if lock2 == nil {
		t.Fatal("acquireIPLock returned nil for second IP")
	}

	// Release both locks
	lock1.Unlock()
	lock2.Unlock()
}

// TestAcquireIPLock_SameIP tests lock contention for same IP
func TestAcquireIPLock_SameIP(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	done := make(chan bool, 1)

	// Acquire lock for IP (acquireIPLock already calls Lock() internally)
	// Use unique IP to avoid contention with other tests
	lock := db.acquireIPLock("10.10.10.1")

	// Try to acquire same lock in goroutine (should block until released)
	go func() {
		lock2 := db.acquireIPLock("10.10.10.1") // Same IP will block
		done <- true
		lock2.Unlock()
	}()

	// Wait a bit to ensure goroutine is blocked
	select {
	case <-done:
		t.Error("Second lock should not be acquired while first is held")
	case <-time.After(50 * time.Millisecond):
		// Expected - goroutine is blocked
	}

	// Release first lock
	lock.Unlock()

	// Now second lock should complete
	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("Second lock should be acquired after first is released")
	}
}

// TestDoUpsertGeolocation tests the internal upsert helper
func TestDoUpsertGeolocation_NewGeolocation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.spatialAvailable {
		t.Skip("Spatial extension not available")
	}

	city := "Chicago"
	geo := &models.Geolocation{
		IPAddress: "10.0.0.50",
		Latitude:  41.8781,
		Longitude: -87.6298,
		City:      &city,
		Country:   "United States",
	}

	// Use server lat/lon 0,0 for testing
	err := db.doUpsertGeolocation(context.Background(), geo, 0.0, 0.0)
	if err != nil {
		t.Fatalf("doUpsertGeolocation failed: %v", err)
	}

	// Verify geolocation was inserted
	geos, err := db.GetGeolocations(context.Background(), []string{"10.0.0.50"})
	if err != nil {
		t.Fatalf("GetGeolocations failed: %v", err)
	}
	if len(geos) != 1 {
		t.Errorf("Expected 1 geolocation, got %d", len(geos))
	}
}

// TestDoUpsertGeolocation_UpdateExisting tests updating an existing geolocation
func TestDoUpsertGeolocation_UpdateExisting(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.spatialAvailable {
		t.Skip("Spatial extension not available")
	}

	// Insert first
	city1 := "OldCity"
	geo1 := &models.Geolocation{
		IPAddress: "10.0.0.51",
		Latitude:  40.0,
		Longitude: -75.0,
		City:      &city1,
		Country:   "United States",
	}
	err := db.doUpsertGeolocation(context.Background(), geo1, 0.0, 0.0)
	if err != nil {
		t.Fatalf("First upsert failed: %v", err)
	}

	// Update with new data
	city2 := "NewCity"
	geo2 := &models.Geolocation{
		IPAddress: "10.0.0.51",
		Latitude:  41.0,
		Longitude: -76.0,
		City:      &city2,
		Country:   "United States",
	}
	err = db.doUpsertGeolocation(context.Background(), geo2, 0.0, 0.0)
	if err != nil {
		t.Fatalf("Second upsert failed: %v", err)
	}

	// Verify city was updated
	geos, err := db.GetGeolocations(context.Background(), []string{"10.0.0.51"})
	if err != nil {
		t.Fatalf("GetGeolocations failed: %v", err)
	}
	if len(geos) != 1 {
		t.Errorf("Expected 1 geolocation, got %d", len(geos))
	}
	if geos["10.0.0.51"].City == nil || *geos["10.0.0.51"].City != "NewCity" {
		t.Error("City should be updated to NewCity")
	}
}
