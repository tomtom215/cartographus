// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"testing"
)

// TestJsonExtensionAvailable verifies the JSON extension is loaded
func TestJsonExtensionAvailable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Check if JSON extension is available
	if !db.IsJSONAvailable() {
		t.Skip("JSON extension not available")
		return
	}

	// Verify we can use JSON functions
	var result string
	err := db.conn.QueryRow("SELECT json_extract('{\"name\":\"test\"}', '$.name')::VARCHAR").Scan(&result)
	if err != nil {
		t.Fatalf("Failed to execute json_extract function: %v", err)
	}

	if result != `"test"` {
		t.Errorf("Expected json_extract to return '\"test\"', got '%s'", result)
	}
}

// TestJsonParsing tests JSON parsing and validation
func TestJsonParsing(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsJSONAvailable() {
		t.Skip("JSON extension not available")
		return
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid object", `{"name":"test","value":123}`, false},
		{"valid array", `[1,2,3,"four"]`, false},
		{"nested object", `{"user":{"name":"Alice","age":30}}`, false},
		{"nested array", `{"items":[{"id":1},{"id":2}]}`, false},
		{"empty object", `{}`, false},
		{"empty array", `[]`, false},
		{"string value", `"hello"`, false},
		{"number value", `42`, false},
		{"boolean true", `true`, false},
		{"boolean false", `false`, false},
		{"null value", `null`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			err := db.conn.QueryRow("SELECT json($1)::VARCHAR", tt.input).Scan(&result)
			if (err != nil) != tt.wantErr {
				t.Errorf("json(%s): error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// TestJsonExtractString tests json_extract_string for scalar extraction
func TestJsonExtractString(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsJSONAvailable() {
		t.Skip("JSON extension not available")
		return
	}

	tests := []struct {
		name     string
		json     string
		path     string
		expected string
	}{
		{
			name:     "extract string field",
			json:     `{"name":"Alice","age":30}`,
			path:     "$.name",
			expected: "Alice",
		},
		{
			name:     "extract nested string",
			json:     `{"user":{"name":"Bob"}}`,
			path:     "$.user.name",
			expected: "Bob",
		},
		{
			name:     "extract array element",
			json:     `{"items":["first","second","third"]}`,
			path:     "$.items[1]",
			expected: "second",
		},
		{
			name:     "extract number as string",
			json:     `{"count":42}`,
			path:     "$.count",
			expected: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			err := db.conn.QueryRow("SELECT json_extract_string($1, $2)", tt.json, tt.path).Scan(&result)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestJsonExtract tests json_extract for typed extraction
func TestJsonExtract(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsJSONAvailable() {
		t.Skip("JSON extension not available")
		return
	}

	// Test extracting different types
	t.Run("extract integer", func(t *testing.T) {
		var result int
		err := db.conn.QueryRow("SELECT json_extract('{\"value\":42}', '$.value')::INTEGER").Scan(&result)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if result != 42 {
			t.Errorf("Expected 42, got %d", result)
		}
	})

	t.Run("extract double", func(t *testing.T) {
		var result float64
		err := db.conn.QueryRow("SELECT json_extract('{\"value\":3.14}', '$.value')::DOUBLE").Scan(&result)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if result != 3.14 {
			t.Errorf("Expected 3.14, got %f", result)
		}
	})

	t.Run("extract boolean", func(t *testing.T) {
		var result bool
		err := db.conn.QueryRow("SELECT json_extract('{\"active\":true}', '$.active')::BOOLEAN").Scan(&result)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if !result {
			t.Error("Expected true, got false")
		}
	})
}

// TestJsonArrayLength tests json_array_length function
func TestJsonArrayLength(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsJSONAvailable() {
		t.Skip("JSON extension not available")
		return
	}

	tests := []struct {
		name     string
		json     string
		path     string
		expected int
	}{
		{
			name:     "top-level array",
			json:     `[1,2,3,4,5]`,
			path:     "$",
			expected: 5,
		},
		{
			name:     "nested array",
			json:     `{"items":[1,2,3]}`,
			path:     "$.items",
			expected: 3,
		},
		{
			name:     "empty array",
			json:     `{"items":[]}`,
			path:     "$.items",
			expected: 0,
		},
		{
			name:     "array of objects",
			json:     `{"users":[{"id":1},{"id":2}]}`,
			path:     "$.users",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result int
			err := db.conn.QueryRow("SELECT json_array_length(json_extract($1, $2))", tt.json, tt.path).Scan(&result)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestJsonKeys tests json_keys function for schema inspection
func TestJsonKeys(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsJSONAvailable() {
		t.Skip("JSON extension not available")
		return
	}

	// Test getting keys from an object
	var result string
	err := db.conn.QueryRow(`SELECT json_keys('{"name":"Alice","age":30,"city":"NYC"}')::VARCHAR`).Scan(&result)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	// Result should contain all three keys as a JSON array
	t.Logf("json_keys result: %s", result)

	// Verify it's a valid JSON array containing the expected keys
	var length int
	err = db.conn.QueryRow(`SELECT json_array_length(json_keys('{"name":"Alice","age":30,"city":"NYC"}'))`).Scan(&length)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if length != 3 {
		t.Errorf("Expected 3 keys, got %d", length)
	}
}

// TestJsonStructure tests json_structure for type inspection
func TestJsonStructure(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsJSONAvailable() {
		t.Skip("JSON extension not available")
		return
	}

	// Test getting structure of a nested JSON object
	var result string
	err := db.conn.QueryRow(`SELECT json_structure('{"name":"Alice","tags":["a","b"],"meta":{"id":1}}')::VARCHAR`).Scan(&result)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	t.Logf("json_structure result: %s", result)

	// Just verify we get a non-empty result describing the structure
	if len(result) == 0 {
		t.Error("Expected non-empty structure description")
	}
}

// TestJsonOperators tests -> and ->> shorthand operators
func TestJsonOperators(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsJSONAvailable() {
		t.Skip("JSON extension not available")
		return
	}

	// Test -> operator (returns JSON)
	t.Run("arrow operator returns JSON", func(t *testing.T) {
		var result string
		err := db.conn.QueryRow(`SELECT ('{"name":"test"}'::JSON -> 'name')::VARCHAR`).Scan(&result)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		// -> returns the JSON value (with quotes for strings)
		if result != `"test"` {
			t.Errorf("Expected '\"test\"', got '%s'", result)
		}
	})

	// Test ->> operator (returns string)
	t.Run("double arrow operator returns string", func(t *testing.T) {
		var result string
		err := db.conn.QueryRow(`SELECT '{"name":"test"}'::JSON ->> 'name'`).Scan(&result)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		// ->> returns the actual string value (no quotes)
		if result != "test" {
			t.Errorf("Expected 'test', got '%s'", result)
		}
	})

	// Test chained operators for nested access
	t.Run("chained operators for nested access", func(t *testing.T) {
		var result string
		err := db.conn.QueryRow(`SELECT '{"user":{"name":"Alice"}}'::JSON -> 'user' ->> 'name'`).Scan(&result)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if result != "Alice" {
			t.Errorf("Expected 'Alice', got '%s'", result)
		}
	})
}

// TestJsonNestedAccess tests accessing deeply nested JSON values
func TestJsonNestedAccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsJSONAvailable() {
		t.Skip("JSON extension not available")
		return
	}

	complexJSON := `{
		"media": {
			"type": "movie",
			"details": {
				"title": "Inception",
				"year": 2010,
				"directors": ["Christopher Nolan"],
				"ratings": {
					"imdb": 8.8,
					"rottenTomatoes": 87
				}
			}
		}
	}`

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"media type", "$.media.type", "movie"},
		{"movie title", "$.media.details.title", "Inception"},
		{"release year", "$.media.details.year", "2010"},
		{"imdb rating", "$.media.details.ratings.imdb", "8.8"},
		{"first director", "$.media.details.directors[0]", "Christopher Nolan"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			err := db.conn.QueryRow("SELECT json_extract_string($1, $2)", complexJSON, tt.path).Scan(&result)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestJsonCreation tests JSON creation functions
func TestJsonCreation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsJSONAvailable() {
		t.Skip("JSON extension not available")
		return
	}

	// Test json_object creation
	t.Run("json_object", func(t *testing.T) {
		var result string
		err := db.conn.QueryRow(`SELECT json_object('name', 'Alice', 'age', 30)::VARCHAR`).Scan(&result)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		t.Logf("json_object result: %s", result)
		// Verify it contains expected fields
		if len(result) == 0 {
			t.Error("Expected non-empty JSON object")
		}
	})

	// Test json_array creation
	t.Run("json_array", func(t *testing.T) {
		var result string
		err := db.conn.QueryRow(`SELECT json_array(1, 2, 'three', true)::VARCHAR`).Scan(&result)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		t.Logf("json_array result: %s", result)
		// Should be [1,2,"three",true]
		if result != `[1,2,"three",true]` {
			t.Errorf("Unexpected json_array result: %s", result)
		}
	})
}

// TestJsonAggregation tests JSON aggregation functions
func TestJsonAggregation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsJSONAvailable() {
		t.Skip("JSON extension not available")
		return
	}

	// Create test data
	_, err := db.conn.Exec(`
		CREATE TEMPORARY TABLE test_users (id INTEGER, name TEXT, score INTEGER);
		INSERT INTO test_users VALUES (1, 'Alice', 95), (2, 'Bob', 87), (3, 'Charlie', 92);
	`)
	if err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// Test json_group_array - aggregate into JSON array
	t.Run("json_group_array", func(t *testing.T) {
		var result string
		// Use a subquery to ensure ordering before aggregation
		err := db.conn.QueryRow(`SELECT json_group_array(name)::VARCHAR FROM (SELECT name FROM test_users ORDER BY id)`).Scan(&result)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		t.Logf("json_group_array result: %s", result)

		// Verify it's a valid array with 3 elements
		var length int
		err = db.conn.QueryRow("SELECT json_array_length($1::JSON)", result).Scan(&length)
		if err != nil {
			t.Fatalf("Failed to get array length: %v", err)
		}
		if length != 3 {
			t.Errorf("Expected 3 elements, got %d", length)
		}
	})

	// Test json_group_object - aggregate into JSON object
	t.Run("json_group_object", func(t *testing.T) {
		var result string
		err := db.conn.QueryRow(`SELECT json_group_object(name, score)::VARCHAR FROM test_users`).Scan(&result)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		t.Logf("json_group_object result: %s", result)

		// Verify it has 3 keys
		var length int
		err = db.conn.QueryRow("SELECT json_array_length(json_keys($1::JSON))", result).Scan(&length)
		if err != nil {
			t.Fatalf("Failed to get keys length: %v", err)
		}
		if length != 3 {
			t.Errorf("Expected 3 keys, got %d", length)
		}
	})
}

// TestJsonTableStorage tests storing and querying JSON in tables
func TestJsonTableStorage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsJSONAvailable() {
		t.Skip("JSON extension not available")
		return
	}

	// Create a table with JSON column
	_, err := db.conn.Exec(`
		CREATE TEMPORARY TABLE test_metadata (
			id INTEGER PRIMARY KEY,
			name TEXT,
			metadata JSON
		);
		INSERT INTO test_metadata VALUES
			(1, 'Movie A', '{"genre":"action","rating":8.5,"tags":["thriller","sci-fi"]}'),
			(2, 'Movie B', '{"genre":"comedy","rating":7.2,"tags":["romance"]}'),
			(3, 'Movie C', '{"genre":"action","rating":9.0,"tags":["adventure","action"]}');
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test querying JSON field
	t.Run("filter by JSON field", func(t *testing.T) {
		var count int
		err := db.conn.QueryRow(`
			SELECT COUNT(*) FROM test_metadata
			WHERE metadata ->> 'genre' = 'action'
		`).Scan(&count)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if count != 2 {
			t.Errorf("Expected 2 action movies, got %d", count)
		}
	})

	// Test extracting and aggregating JSON data
	t.Run("aggregate JSON values", func(t *testing.T) {
		var avgRating float64
		err := db.conn.QueryRow(`
			SELECT AVG((metadata ->> 'rating')::DOUBLE) FROM test_metadata
		`).Scan(&avgRating)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		// Expected: (8.5 + 7.2 + 9.0) / 3 = 8.233...
		if avgRating < 8.2 || avgRating > 8.3 {
			t.Errorf("Expected avg rating ~8.23, got %f", avgRating)
		}
	})
}

// TestJsonTautulliMetadata tests a practical use case: extracting metadata from Tautulli API responses
// This simulates storing and querying flexible metadata fields
func TestJsonTautulliMetadata(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsJSONAvailable() {
		t.Skip("JSON extension not available")
		return
	}

	// Simulate Tautulli-like metadata storage
	_, err := db.conn.Exec(`
		CREATE TEMPORARY TABLE test_playback_metadata (
			session_key TEXT PRIMARY KEY,
			media_type TEXT,
			metadata JSON
		);
		INSERT INTO test_playback_metadata VALUES
			('session1', 'movie', '{
				"title": "Inception",
				"year": 2010,
				"video": {"resolution": "1080p", "codec": "h264", "bitrate": 8000},
				"audio": {"codec": "aac", "channels": 6, "language": "eng"},
				"stream": {"bandwidth": 15000, "transcoding": false}
			}'),
			('session2', 'episode', '{
				"title": "The Office",
				"season": 3,
				"episode": 5,
				"video": {"resolution": "720p", "codec": "hevc", "bitrate": 4000},
				"audio": {"codec": "ac3", "channels": 2, "language": "eng"},
				"stream": {"bandwidth": 8000, "transcoding": true}
			}'),
			('session3', 'movie', '{
				"title": "Matrix",
				"year": 1999,
				"video": {"resolution": "4K", "codec": "hevc", "bitrate": 20000},
				"audio": {"codec": "truehd", "channels": 8, "language": "eng"},
				"stream": {"bandwidth": 25000, "transcoding": false}
			}');
	`)
	if err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// Test extracting video resolution distribution
	t.Run("video resolution distribution", func(t *testing.T) {
		rows, err := db.conn.Query(`
			SELECT
				metadata -> 'video' ->> 'resolution' as resolution,
				COUNT(*) as count
			FROM test_playback_metadata
			GROUP BY resolution
			ORDER BY count DESC
		`)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		defer rows.Close()

		var count int
		for rows.Next() {
			var resolution string
			var resCount int
			if err := rows.Scan(&resolution, &resCount); err != nil {
				t.Fatalf("Scan failed: %v", err)
			}
			t.Logf("Resolution: %s, Count: %d", resolution, resCount)
			count++
		}

		if err := rows.Err(); err != nil {
			t.Fatalf("Row iteration error: %v", err)
		}

		if count != 3 {
			t.Errorf("Expected 3 different resolutions, got %d", count)
		}
	})

	// Test finding transcoding sessions
	t.Run("find transcoding sessions", func(t *testing.T) {
		var count int
		err := db.conn.QueryRow(`
			SELECT COUNT(*) FROM test_playback_metadata
			WHERE (metadata -> 'stream' ->> 'transcoding')::BOOLEAN = true
		`).Scan(&count)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 transcoding session, got %d", count)
		}
	})

	// Test total bandwidth calculation
	t.Run("total bandwidth", func(t *testing.T) {
		var totalBandwidth int
		err := db.conn.QueryRow(`
			SELECT SUM((metadata -> 'stream' ->> 'bandwidth')::INTEGER)
			FROM test_playback_metadata
		`).Scan(&totalBandwidth)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		// Expected: 15000 + 8000 + 25000 = 48000
		if totalBandwidth != 48000 {
			t.Errorf("Expected total bandwidth 48000, got %d", totalBandwidth)
		}
	})

	// Test audio codec breakdown
	t.Run("audio codec breakdown", func(t *testing.T) {
		rows, err := db.conn.Query(`
			SELECT
				metadata -> 'audio' ->> 'codec' as audio_codec,
				json_group_array(metadata ->> 'title')::VARCHAR as titles
			FROM test_playback_metadata
			GROUP BY audio_codec
		`)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var codec, titles string
			if err := rows.Scan(&codec, &titles); err != nil {
				t.Fatalf("Scan failed: %v", err)
			}
			t.Logf("Codec: %s, Titles: %s", codec, titles)
		}

		if err := rows.Err(); err != nil {
			t.Fatalf("Row iteration error: %v", err)
		}
	})
}

// TestJsonTransform tests json_transform for modifying JSON values
func TestJsonTransform(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsJSONAvailable() {
		t.Skip("JSON extension not available")
		return
	}

	// Test merging JSON objects with json_merge_patch
	t.Run("json_merge_patch", func(t *testing.T) {
		var result string
		err := db.conn.QueryRow(`
			SELECT json_merge_patch(
				'{"name":"Alice","age":30}',
				'{"age":31,"city":"NYC"}'
			)::VARCHAR
		`).Scan(&result)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		t.Logf("json_merge_patch result: %s", result)

		// Verify merged result contains updated age and new city
		var age int
		err = db.conn.QueryRow("SELECT json_extract($1::JSON, '$.age')::INTEGER", result).Scan(&age)
		if err != nil {
			t.Fatalf("Failed to extract age: %v", err)
		}
		if age != 31 {
			t.Errorf("Expected age 31, got %d", age)
		}

		var city string
		err = db.conn.QueryRow("SELECT json_extract_string($1, '$.city')", result).Scan(&city)
		if err != nil {
			t.Fatalf("Failed to extract city: %v", err)
		}
		if city != "NYC" {
			t.Errorf("Expected city NYC, got %s", city)
		}
	})
}

// TestJsonValid tests json_valid for JSON validation
func TestJsonValid(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsJSONAvailable() {
		t.Skip("JSON extension not available")
		return
	}

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid object", `{"name":"test"}`, true},
		{"valid array", `[1,2,3]`, true},
		{"invalid - missing quotes", `{name:"test"}`, false},
		// Note: DuckDB's JSON parser is lenient and accepts trailing commas
		{"trailing comma - DuckDB lenient", `{"a":1,}`, true},
		{"invalid - single quotes", `{'name':'test'}`, false},
		{"empty string", ``, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result bool
			err := db.conn.QueryRow("SELECT json_valid($1)", tt.input).Scan(&result)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("json_valid(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}
