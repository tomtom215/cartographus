// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/models"
)

// setupTestHandlerSpatial creates a test handler with a nil database (for validation-only tests)
func setupTestHandlerSpatial(t *testing.T) *Handler {
	t.Helper()
	c := cache.New(5 * time.Minute)
	return &Handler{db: nil, cache: c}
}

// handlerMethodTest defines a test case for HTTP method validation
type handlerMethodTest struct {
	name           string
	handlerFunc    func(*Handler, http.ResponseWriter, *http.Request)
	path           string
	invalidMethods []string
}

// runMethodNotAllowedTests runs method validation tests for multiple handlers
func runMethodNotAllowedTests(t *testing.T, tests []handlerMethodTest) {
	t.Helper()
	handler := setupTestHandlerSpatial(t)
	for _, tt := range tests {
		for _, method := range tt.invalidMethods {
			t.Run(tt.name+"_"+method, func(t *testing.T) {
				req := httptest.NewRequest(method, tt.path, nil)
				rr := httptest.NewRecorder()
				tt.handlerFunc(handler, rr, req)
				if rr.Code != http.StatusMethodNotAllowed {
					t.Errorf("%s with method %s: status = %d, want %d", tt.name, method, rr.Code, http.StatusMethodNotAllowed)
				}
			})
		}
	}
}

// paramValidationTest defines a test case for parameter validation
type paramValidationTest struct {
	name   string
	params string
}

// runBadRequestTests runs parameter validation tests expecting BadRequest
func runBadRequestTests(t *testing.T, handler *Handler, handlerFunc func(*Handler, http.ResponseWriter, *http.Request), basePath string, tests []paramValidationTest) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := basePath
			if tt.params != "" {
				path += "?" + tt.params
			}
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rr := httptest.NewRecorder()
			handlerFunc(handler, rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d for params: %s", rr.Code, http.StatusBadRequest, tt.params)
			}
		})
	}
}

// TestOptionalString tests the optionalString helper function
func TestOptionalString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{"nil pointer returns empty string", nil, ""},
		{"empty string returns empty string", stringPtr(""), ""},
		{"simple string", stringPtr("hello"), "hello"},
		{"string with comma gets escaped", stringPtr("hello, world"), `"hello, world"`},
		{"string with quotes gets escaped", stringPtr(`say "hello"`), `"say ""hello"""`},
		{"string with newline gets escaped", stringPtr("line1\nline2"), "\"line1\nline2\""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := optionalString(tt.input); result != tt.expected {
				t.Errorf("optionalString(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestOptionalInt tests the optionalInt helper function
func TestOptionalInt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    *int
		expected string
	}{
		{"nil pointer returns empty string", nil, ""},
		{"zero value", intPtr(0), "0"},
		{"positive number", intPtr(42), "42"},
		{"negative number", intPtr(-123), "-123"},
		{"large number", intPtr(1000000), "1000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := optionalInt(tt.input); result != tt.expected {
				t.Errorf("optionalInt(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestOptionalTime tests the optionalTime helper function
func TestOptionalTime(t *testing.T) {
	t.Parallel()
	fixedTime := time.Date(2025, 11, 25, 14, 30, 0, 0, time.UTC)
	tests := []struct {
		name     string
		input    *time.Time
		expected string
	}{
		{"nil pointer returns empty string", nil, ""},
		{"valid time formats as RFC3339", &fixedTime, "2025-11-25T14:30:00Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := optionalTime(tt.input); result != tt.expected {
				t.Errorf("optionalTime(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestValidateBoundingBox tests the ValidateBoundingBox helper function
func TestValidateBoundingBox(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		queryParams map[string]string
		wantBox     *BoundingBoxParams
		wantErr     string
	}{
		{"missing all parameters", map[string]string{}, nil, "missing required parameter: west"},
		{"missing west parameter", map[string]string{"south": "40.0", "east": "-73.0", "north": "41.0"}, nil, "missing required parameter: west"},
		{"invalid west value", map[string]string{"west": "not-a-number", "south": "40.0", "east": "-73.0", "north": "41.0"}, nil, "invalid west parameter"},
		{"west out of range (too low)", map[string]string{"west": "-181.0", "south": "40.0", "east": "-73.0", "north": "41.0"}, nil, "invalid west parameter"},
		{"west out of range (too high)", map[string]string{"west": "181.0", "south": "40.0", "east": "-73.0", "north": "41.0"}, nil, "invalid west parameter"},
		{"invalid south value", map[string]string{"west": "-74.0", "south": "invalid", "east": "-73.0", "north": "41.0"}, nil, "invalid south parameter"},
		{"south out of range", map[string]string{"west": "-74.0", "south": "-91.0", "east": "-73.0", "north": "41.0"}, nil, "invalid south parameter"},
		{"invalid east value", map[string]string{"west": "-74.0", "south": "40.0", "east": "abc", "north": "41.0"}, nil, "invalid east parameter"},
		{"east out of range", map[string]string{"west": "-74.0", "south": "40.0", "east": "200.0", "north": "41.0"}, nil, "invalid east parameter"},
		{"invalid north value", map[string]string{"west": "-74.0", "south": "40.0", "east": "-73.0", "north": "xyz"}, nil, "invalid north parameter"},
		{"north out of range", map[string]string{"west": "-74.0", "south": "40.0", "east": "-73.0", "north": "95.0"}, nil, "invalid north parameter"},
		{"valid bounding box", map[string]string{"west": "-74.0", "south": "40.0", "east": "-73.0", "north": "41.0"}, &BoundingBoxParams{West: -74.0, South: 40.0, East: -73.0, North: 41.0}, ""},
		{"edge case: exact boundary values", map[string]string{"west": "-180", "south": "-90", "east": "180", "north": "90"}, &BoundingBoxParams{West: -180, South: -90, East: 180, North: 90}, ""},
		{"edge case: zero coordinates", map[string]string{"west": "0", "south": "0", "east": "0", "north": "0"}, &BoundingBoxParams{West: 0, South: 0, East: 0, North: 0}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			for k, v := range tt.queryParams {
				params.Set(k, v)
			}
			req := httptest.NewRequest(http.MethodGet, "/?"+params.Encode(), nil)
			box, err := ValidateBoundingBox(req)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ValidateBoundingBox() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("ValidateBoundingBox() unexpected error = %q", err.Error())
				return
			}
			if box == nil || box.West != tt.wantBox.West || box.South != tt.wantBox.South ||
				box.East != tt.wantBox.East || box.North != tt.wantBox.North {
				t.Errorf("ValidateBoundingBox() = %+v, want %+v", box, tt.wantBox)
			}
		})
	}
}

// TestParseExportFilter tests the parseExportFilter helper function
func TestParseExportFilter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		queryParams map[string]string
		checkFilter func(*testing.T, database.LocationStatsFilter)
		wantErr     string
	}{
		{
			name: "empty parameters returns empty filter",
			checkFilter: func(t *testing.T, f database.LocationStatsFilter) {
				if f.StartDate != nil || f.EndDate != nil {
					t.Error("expected nil StartDate and EndDate")
				}
			},
		},
		{"invalid start_date format", map[string]string{"start_date": "2025-01-01"}, nil, "Invalid start_date format"},
		{
			name:        "valid start_date",
			queryParams: map[string]string{"start_date": "2025-01-01T00:00:00Z"},
			checkFilter: func(t *testing.T, f database.LocationStatsFilter) {
				if f.StartDate == nil || !f.StartDate.Equal(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)) {
					t.Errorf("StartDate = %v, want 2025-01-01", f.StartDate)
				}
			},
		},
		{"invalid end_date format", map[string]string{"end_date": "not-a-date"}, nil, "Invalid end_date format"},
		{
			name:        "valid end_date",
			queryParams: map[string]string{"end_date": "2025-12-31T23:59:59Z"},
			checkFilter: func(t *testing.T, f database.LocationStatsFilter) {
				if f.EndDate == nil || !f.EndDate.Equal(time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)) {
					t.Errorf("EndDate = %v, want 2025-12-31", f.EndDate)
				}
			},
		},
		{
			name:        "days parameter - valid",
			queryParams: map[string]string{"days": "30"},
			checkFilter: func(t *testing.T, f database.LocationStatsFilter) {
				if f.StartDate == nil {
					t.Error("expected non-nil StartDate from days param")
					return
				}
				expectedApprox := time.Now().AddDate(0, 0, -30)
				if diff := f.StartDate.Sub(expectedApprox); diff < -time.Minute || diff > time.Minute {
					t.Errorf("StartDate from days = %v, want approximately %v", f.StartDate, expectedApprox)
				}
			},
		},
		{"days parameter - too small", map[string]string{"days": "0"}, nil, "Days must be between 1 and 3650"},
		{"days parameter - too large", map[string]string{"days": "3651"}, nil, "Days must be between 1 and 3650"},
		{
			name:        "users parameter",
			queryParams: map[string]string{"users": "alice,bob,charlie"},
			checkFilter: func(t *testing.T, f database.LocationStatsFilter) {
				if len(f.Users) != 3 || f.Users[0] != "alice" || f.Users[1] != "bob" || f.Users[2] != "charlie" {
					t.Errorf("Users = %v, want [alice bob charlie]", f.Users)
				}
			},
		},
		{
			name:        "media_types parameter",
			queryParams: map[string]string{"media_types": "movie,episode"},
			checkFilter: func(t *testing.T, f database.LocationStatsFilter) {
				if len(f.MediaTypes) != 2 || f.MediaTypes[0] != "movie" || f.MediaTypes[1] != "episode" {
					t.Errorf("MediaTypes = %v, want [movie episode]", f.MediaTypes)
				}
			},
		},
		{
			name:        "all parameters combined",
			queryParams: map[string]string{"start_date": "2025-01-01T00:00:00Z", "end_date": "2025-12-31T23:59:59Z", "users": "user1,user2", "media_types": "movie"},
			checkFilter: func(t *testing.T, f database.LocationStatsFilter) {
				if f.StartDate == nil || f.EndDate == nil || len(f.Users) != 2 || len(f.MediaTypes) != 1 {
					t.Error("expected all parameters to be parsed")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			for k, v := range tt.queryParams {
				params.Set(k, v)
			}
			req := httptest.NewRequest(http.MethodGet, "/?"+params.Encode(), nil)
			filter, errMsg := parseExportFilter(req)

			if tt.wantErr != "" {
				if errMsg == "" || !strings.Contains(errMsg, tt.wantErr) {
					t.Errorf("parseExportFilter() error = %q, want error containing %q", errMsg, tt.wantErr)
				}
				return
			}
			if errMsg != "" {
				t.Errorf("parseExportFilter() unexpected error = %q", errMsg)
				return
			}
			if tt.checkFilter != nil {
				tt.checkFilter(t, filter)
			}
		})
	}
}

// TestBuildCSVRow tests the buildCSVRow helper function
func TestBuildCSVRow(t *testing.T) {
	t.Parallel()
	fixedTime := time.Date(2025, 11, 25, 14, 30, 0, 0, time.UTC)
	stoppedTime := time.Date(2025, 11, 25, 16, 0, 0, 0, time.UTC)
	testUUID := uuid.MustParse("12345678-1234-5678-1234-567812345678")

	tests := []struct {
		name     string
		event    *models.PlaybackEvent
		contains []string
	}{
		{
			name: "basic event with required fields",
			event: &models.PlaybackEvent{
				ID: testUUID, SessionKey: "session123", StartedAt: fixedTime, UserID: 1,
				Username: "testuser", IPAddress: "192.168.1.1", MediaType: "movie",
				Title: "Test Movie", Platform: "Roku", Player: "Roku Ultra",
				LocationType: "lan", PercentComplete: 100, PausedCounter: 2, CreatedAt: fixedTime,
			},
			contains: []string{"12345678-1234-5678-1234-567812345678", "session123", "2025-11-25T14:30:00Z", "testuser", "192.168.1.1", "movie", "Test Movie", "Roku", "100", "2"},
		},
		{
			name: "event with optional fields populated",
			event: &models.PlaybackEvent{
				ID: testUUID, SessionKey: "session456", StartedAt: fixedTime, StoppedAt: &stoppedTime,
				UserID: 2, Username: "anotheruser", IPAddress: "10.0.0.1", MediaType: "episode",
				Title: "Episode Title", ParentTitle: stringPtr("Season 1"), GrandparentTitle: stringPtr("TV Show Name"),
				Platform: "Web", Player: "Chrome", LocationType: "wan", PercentComplete: 75, PausedCounter: 0,
				TranscodeDecision: stringPtr("transcode"), VideoResolution: stringPtr("1080"),
				VideoCodec: stringPtr("h264"), AudioCodec: stringPtr("aac"),
				SectionID: intPtr(1), LibraryName: stringPtr("TV Shows"),
				ContentRating: stringPtr("TV-MA"), PlayDuration: intPtr(3600), Year: intPtr(2025), CreatedAt: fixedTime,
			},
			contains: []string{"session456", "Season 1", "TV Show Name", "transcode", "1080", "h264", "aac", "TV Shows", "TV-MA", "3600", "2025", "2025-11-25T16:00:00Z"},
		},
		{
			name: "event with special characters",
			event: &models.PlaybackEvent{
				ID: testUUID, SessionKey: "session,with,commas", StartedAt: fixedTime, UserID: 3,
				Username: "user\"quotes\"", IPAddress: "127.0.0.1", MediaType: "movie",
				Title: "Movie, With \"Special\" Characters", Platform: "Test", Player: "Test Player",
				LocationType: "lan", PercentComplete: 50, PausedCounter: 1, CreatedAt: fixedTime,
			},
			contains: []string{`"session,with,commas"`, `"user""quotes"""`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildCSVRow(tt.event)
			if !strings.HasSuffix(result, "\n") {
				t.Error("buildCSVRow() result should end with newline")
			}
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("buildCSVRow() result missing expected substring %q\nGot: %s", substr, result)
				}
			}
			fields := strings.Split(strings.TrimSuffix(result, "\n"), ",")
			if len(fields) < 20 {
				t.Errorf("buildCSVRow() expected at least 20 fields, got %d", len(fields))
			}
		})
	}
}

// Helper functions for creating pointers
func stringPtr(s string) *string { return &s }
func intPtr(i int) *int          { return &i }

// TestAllHandlers_MethodNotAllowed consolidates method validation tests
func TestAllHandlers_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	tests := []handlerMethodTest{
		{"ExportPlaybacksCSV", (*Handler).ExportPlaybacksCSV, "/api/v1/export/playbacks/csv", []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}},
		{"SpatialViewport", (*Handler).SpatialViewport, "/api/v1/spatial/viewport", []string{http.MethodPost, http.MethodPut, http.MethodDelete}},
		{"SpatialHexagons", (*Handler).SpatialHexagons, "/api/v1/spatial/hexagons", []string{http.MethodPost}},
		{"SpatialArcs", (*Handler).SpatialArcs, "/api/v1/spatial/arcs", []string{http.MethodPost}},
		{"SpatialTemporalDensity", (*Handler).SpatialTemporalDensity, "/api/v1/spatial/temporal-density", []string{http.MethodPost}},
		{"SpatialNearby", (*Handler).SpatialNearby, "/api/v1/spatial/nearby", []string{http.MethodPost}},
		{"ExportLocationsGeoJSON", (*Handler).ExportLocationsGeoJSON, "/export/locations/geojson", []string{http.MethodPost}},
	}
	runMethodNotAllowedTests(t, tests)
}

// TestExportPlaybacksCSV_InvalidLimit tests validation of limit parameter
func TestExportPlaybacksCSV_InvalidLimit(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerSpatial(t)
	tests := []paramValidationTest{
		{"zero limit", "limit=0"},
		{"negative limit", "limit=-1"},
		{"too large limit", "limit=100001"},
	}
	runBadRequestTests(t, handler, (*Handler).ExportPlaybacksCSV, "/api/v1/export/playbacks/csv", tests)
}

// TestSpatialViewport_Validation tests parameter validation
func TestSpatialViewport_Validation(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerSpatial(t)

	// Test missing parameters
	t.Run("missing parameters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/viewport", nil)
		rr := httptest.NewRecorder()
		handler.SpatialViewport(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
		var response models.APIResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		if response.Status != "error" {
			t.Errorf("Response status = %s, want error", response.Status)
		}
	})

	// Test invalid coordinates
	tests := []paramValidationTest{
		{"invalid west", "west=abc&south=40&east=-73&north=41"},
		{"west too low", "west=-181&south=40&east=-73&north=41"},
		{"west too high", "west=181&south=40&east=-73&north=41"},
		{"invalid south", "west=-74&south=xyz&east=-73&north=41"},
		{"south too low", "west=-74&south=-91&east=-73&north=41"},
		{"south too high", "west=-74&south=91&east=-73&north=41"},
		{"invalid east", "west=-74&south=40&east=bad&north=41"},
		{"east too low", "west=-74&south=40&east=-181&north=41"},
		{"east too high", "west=-74&south=40&east=181&north=41"},
		{"invalid north", "west=-74&south=40&east=-73&north=invalid"},
		{"north too low", "west=-74&south=40&east=-73&north=-91"},
		{"north too high", "west=-74&south=40&east=-73&north=91"},
	}
	runBadRequestTests(t, handler, (*Handler).SpatialViewport, "/api/v1/spatial/viewport", tests)
}

// TestSpatialHexagons_InvalidResolution tests validation of resolution parameter
func TestSpatialHexagons_InvalidResolution(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerSpatial(t)
	tests := []paramValidationTest{
		{"too low", "resolution=5"},
		{"too high", "resolution=9"},
	}
	runBadRequestTests(t, handler, (*Handler).SpatialHexagons, "/api/v1/spatial/hexagons", tests)
}

// TestSpatialTemporalDensity_Validation tests parameter validation
func TestSpatialTemporalDensity_Validation(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerSpatial(t)
	tests := []paramValidationTest{
		{"invalid interval", "interval=invalid"},
		{"resolution too low", "resolution=5"},
		{"resolution too high", "resolution=9"},
	}
	runBadRequestTests(t, handler, (*Handler).SpatialTemporalDensity, "/api/v1/spatial/temporal-density", tests)
}

// TestSpatialNearby_Validation tests parameter validation
func TestSpatialNearby_Validation(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerSpatial(t)
	tests := []paramValidationTest{
		// Missing parameters
		{"missing both", ""},
		{"missing lat", "lon=-74.0"},
		{"missing lon", "lat=40.0"},
		// Invalid coordinates
		{"invalid lat", "lat=abc&lon=-74.0"},
		{"lat too low", "lat=-91&lon=-74.0"},
		{"lat too high", "lat=91&lon=-74.0"},
		{"invalid lon", "lat=40.0&lon=xyz"},
		{"lon too low", "lat=40.0&lon=-181"},
		{"lon too high", "lat=40.0&lon=181"},
		// Invalid radius
		{"invalid radius format", "lat=40.0&lon=-74.0&radius=abc"},
		{"radius too small", "lat=40.0&lon=-74.0&radius=0"},
		{"radius too large", "lat=40.0&lon=-74.0&radius=20001"},
	}
	runBadRequestTests(t, handler, (*Handler).SpatialNearby, "/api/v1/spatial/nearby", tests)
}

// TestGetVectorTile_Validation tests tile path and coordinate validation
func TestGetVectorTile_Validation(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerSpatial(t)
	invalidPaths := []struct {
		name string
		path string
	}{
		{"too few path segments", "/api/v1/tiles/1/2"},
		{"invalid zoom", "/api/v1/tiles/abc/0/0.pbf"},
		{"invalid x", "/api/v1/tiles/1/xyz/0.pbf"},
		{"invalid y", "/api/v1/tiles/1/0/abc.pbf"},
		{"negative zoom", "/api/v1/tiles/-1/0/0.pbf"},
		{"zoom too high", "/api/v1/tiles/23/0/0.pbf"},
		{"x out of range", "/api/v1/tiles/1/10/0.pbf"},
		{"y out of range", "/api/v1/tiles/1/0/10.pbf"},
		{"negative x", "/api/v1/tiles/1/-1/0.pbf"},
		{"negative y", "/api/v1/tiles/1/0/-1.pbf"},
	}

	for _, tt := range invalidPaths {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()
			handler.GetVectorTile(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d for path %s", rr.Code, http.StatusBadRequest, tt.path)
			}
		})
	}
}

// TestExportGeoParquet_InvalidDateFormat tests validation of date parameters
func TestExportGeoParquet_InvalidDateFormat(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerSpatial(t)
	tests := []paramValidationTest{
		{"invalid start_date", "start_date=2025-01-01"},
		{"invalid end_date", "end_date=not-a-date"},
	}
	runBadRequestTests(t, handler, (*Handler).ExportGeoParquet, "/api/v1/export/geoparquet", tests)
}

// TestExportGeoJSON_InvalidDateFormat tests validation of date parameters
func TestExportGeoJSON_InvalidDateFormat(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerSpatial(t)
	tests := []paramValidationTest{
		{"invalid start_date", "start_date=2025-01-01"},
		{"invalid end_date", "end_date=not-a-date"},
	}
	runBadRequestTests(t, handler, (*Handler).ExportGeoJSON, "/api/v1/export/geojson", tests)
}

// ========================================
// DB-Backed Export Tests
// ========================================

// TestExportGeoJSON_WithDB tests the ExportGeoJSON handler with database
func TestExportGeoJSON_WithDB(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/export/geojson", nil)
	w := httptest.NewRecorder()

	handler.ExportGeoJSON(w, req)

	// Handler executes without panicking - status can vary based on DB state
	// This tests the code path through the handler, not the file serving
	validStatuses := map[int]bool{
		http.StatusOK:                  true,
		http.StatusInternalServerError: true,
		http.StatusNotFound:            true, // http.ServeFile returns 404 if file not created
	}
	if !validStatuses[w.Code] {
		t.Errorf("Unexpected status %d. Body: %s", w.Code, w.Body.String())
	}
}

// TestExportGeoJSON_WithDB_ValidFilters tests ExportGeoJSON with valid filter parameters
func TestExportGeoJSON_WithDB_ValidFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{"with_date_range", "start_date=2025-01-01T00:00:00Z&end_date=2025-12-31T23:59:59Z"},
		{"with_users", "users=user1,user2"},
		{"with_media_types", "media_types=movie,episode"},
		{"combined_filters", "start_date=2025-01-01T00:00:00Z&users=user1&media_types=movie"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/export/geojson?"+tt.query, nil)
			w := httptest.NewRecorder()

			handler.ExportGeoJSON(w, req)

			// Handler executes without panicking
			// Valid filters should not cause validation error
			if w.Code == http.StatusBadRequest {
				var response models.APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err == nil {
					if response.Error != nil && response.Error.Code == "VALIDATION_ERROR" {
						t.Errorf("Got VALIDATION_ERROR for valid params: %s", tt.query)
					}
				}
			}
			// Success: handler didn't panic and processed the request
		})
	}
}

// TestExportGeoParquet_WithDB tests the ExportGeoParquet handler with database
func TestExportGeoParquet_WithDB(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/export/geoparquet", nil)
	w := httptest.NewRecorder()

	handler.ExportGeoParquet(w, req)

	// May return 503 if spatial extension not available, or 200 for success
	// Should not return validation error for valid request
	if w.Code == http.StatusBadRequest {
		var response models.APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err == nil {
			if response.Error != nil && response.Error.Code == "VALIDATION_ERROR" {
				t.Error("Got VALIDATION_ERROR for valid request")
			}
		}
	}
}

// TestExportGeoParquet_WithDB_ValidFilters tests ExportGeoParquet with valid filter parameters
func TestExportGeoParquet_WithDB_ValidFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{"with_date_range", "start_date=2025-01-01T00:00:00Z&end_date=2025-12-31T23:59:59Z"},
		{"with_users", "users=user1,user2,user3"},
		{"with_media_types", "media_types=movie,episode,track"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/export/geoparquet?"+tt.query, nil)
			w := httptest.NewRecorder()

			handler.ExportGeoParquet(w, req)

			// Valid filters should not cause validation error
			if w.Code == http.StatusBadRequest {
				var response models.APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err == nil {
					if response.Error != nil && response.Error.Code == "VALIDATION_ERROR" {
						t.Errorf("Got VALIDATION_ERROR for valid params: %s", tt.query)
					}
				}
			}
		})
	}
}

// TestGetVectorTile_WithDB tests the GetVectorTile handler with database
func TestGetVectorTile_WithDB(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerWithDB(t, db)

	// Valid tile coordinates - path must have 7 parts: /api/v1/tiles/{z}/{x}/{y}.pbf
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tiles/10/512/512.pbf", nil)
	w := httptest.NewRecorder()

	handler.GetVectorTile(w, req)

	// Should not return a validation error for valid tile coordinates
	// May return 500 if spatial extension not available
	if w.Code == http.StatusBadRequest {
		var response models.APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err == nil {
			if response.Error != nil && response.Error.Code == "VALIDATION_ERROR" {
				t.Error("Got VALIDATION_ERROR for valid tile coordinates")
			}
		}
	}
}

// TestGetVectorTile_WithDB_InvalidCoordinates tests invalid tile coordinates
func TestGetVectorTile_WithDB_InvalidCoordinates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		z    string
		x    string
		y    string
	}{
		{"negative_z", "-1", "0", "0"},
		{"z_too_high", "25", "0", "0"},
		{"x_out_of_range", "1", "10", "0"},
		{"y_out_of_range", "1", "0", "10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerWithDB(t, db)

			// Path must have 7 parts for parsing: /api/v1/tiles/{z}/{x}/{y}.pbf
			path := "/api/v1/tiles/" + tt.z + "/" + tt.x + "/" + tt.y + ".pbf"
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			handler.GetVectorTile(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for invalid coordinates %s/%s/%s, got %d",
					tt.z, tt.x, tt.y, w.Code)
			}
		})
	}
}

// BenchmarkBuildCSVRow benchmarks the CSV row building function
func BenchmarkBuildCSVRow(b *testing.B) {
	fixedTime := time.Date(2025, 11, 25, 14, 30, 0, 0, time.UTC)
	testUUID := uuid.MustParse("12345678-1234-5678-1234-567812345678")
	event := &models.PlaybackEvent{
		ID: testUUID, SessionKey: "session123", StartedAt: fixedTime, StoppedAt: &fixedTime,
		UserID: 1, Username: "testuser", IPAddress: "192.168.1.1", MediaType: "movie",
		Title: "Test Movie", ParentTitle: stringPtr("Parent"), GrandparentTitle: stringPtr("Grandparent"),
		Platform: "Roku", Player: "Roku Ultra", LocationType: "lan", PercentComplete: 100, PausedCounter: 2,
		TranscodeDecision: stringPtr("direct play"), VideoResolution: stringPtr("1080"),
		VideoCodec: stringPtr("h264"), AudioCodec: stringPtr("aac"), SectionID: intPtr(1),
		LibraryName: stringPtr("Movies"), ContentRating: stringPtr("PG-13"),
		PlayDuration: intPtr(7200), Year: intPtr(2025), CreatedAt: fixedTime,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buildCSVRow(event)
	}
}

// BenchmarkValidateBoundingBox benchmarks the bounding box validation function
func BenchmarkValidateBoundingBox(b *testing.B) {
	params := url.Values{}
	params.Set("west", "-74.0")
	params.Set("south", "40.0")
	params.Set("east", "-73.0")
	params.Set("north", "41.0")
	req := httptest.NewRequest(http.MethodGet, "/?"+params.Encode(), nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ValidateBoundingBox(req)
	}
}

// BenchmarkParseExportFilter benchmarks the export filter parsing function
func BenchmarkParseExportFilter(b *testing.B) {
	params := url.Values{}
	params.Set("start_date", "2025-01-01T00:00:00Z")
	params.Set("end_date", "2025-12-31T23:59:59Z")
	params.Set("users", "user1,user2,user3")
	params.Set("media_types", "movie,episode")
	req := httptest.NewRequest(http.MethodGet, "/?"+params.Encode(), nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseExportFilter(req)
	}
}
