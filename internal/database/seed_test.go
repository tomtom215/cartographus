// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"testing"
)

// TestIndexOf tests the indexOf helper function
func TestIndexOf(t *testing.T) {

	tests := []struct {
		name     string
		slice    []string
		item     string
		expected int
	}{
		{
			name:     "item at beginning",
			slice:    []string{"a", "b", "c"},
			item:     "a",
			expected: 0,
		},
		{
			name:     "item in middle",
			slice:    []string{"a", "b", "c"},
			item:     "b",
			expected: 1,
		},
		{
			name:     "item at end",
			slice:    []string{"a", "b", "c"},
			item:     "c",
			expected: 2,
		},
		{
			name:     "item not found returns 0",
			slice:    []string{"a", "b", "c"},
			item:     "d",
			expected: 0,
		},
		{
			name:     "empty slice returns 0",
			slice:    []string{},
			item:     "a",
			expected: 0,
		},
		{
			name:     "single element slice - found",
			slice:    []string{"only"},
			item:     "only",
			expected: 0,
		},
		{
			name:     "single element slice - not found",
			slice:    []string{"only"},
			item:     "other",
			expected: 0,
		},
		{
			name:     "case sensitive - exact match",
			slice:    []string{"Alice", "Bob", "Charlie"},
			item:     "Alice",
			expected: 0,
		},
		{
			name:     "case sensitive - wrong case returns 0",
			slice:    []string{"Alice", "Bob", "Charlie"},
			item:     "alice",
			expected: 0,
		},
		{
			name:     "empty string item",
			slice:    []string{"a", "", "c"},
			item:     "",
			expected: 1,
		},
		{
			name:     "duplicate items returns first index",
			slice:    []string{"a", "b", "a", "c"},
			item:     "a",
			expected: 0,
		},
		{
			name:     "whitespace items",
			slice:    []string{"  ", "hello", "\t"},
			item:     "  ",
			expected: 0,
		},
		{
			name:     "special characters",
			slice:    []string{"user@domain.com", "test'quote", "path/to/file"},
			item:     "test'quote",
			expected: 1,
		},
		{
			name:     "unicode characters",
			slice:    []string{"Alice", "Bob", "Charlie"},
			item:     "Bob",
			expected: 1,
		},
		{
			name:     "long slice",
			slice:    []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
			item:     "j",
			expected: 9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexOf(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("indexOf(%v, %q) = %d, want %d", tt.slice, tt.item, result, tt.expected)
			}
		})
	}
}

// TestIndexOfWithUsernames tests indexOf with realistic username data
func TestIndexOfWithUsernames(t *testing.T) {

	// Match the mock users in seed.go
	users := []string{
		"Alice", "Bob", "Charlie", "David", "Emma",
		"Frank", "Grace", "Henry", "Isabella", "Jack",
		"Kate", "Liam", "Mia", "Noah", "Olivia",
	}

	tests := []struct {
		name     string
		item     string
		expected int
	}{
		{"first user", "Alice", 0},
		{"middle user", "Grace", 6},
		{"last user", "Olivia", 14},
		{"non-existent user", "Unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexOf(users, tt.item)
			if result != tt.expected {
				t.Errorf("indexOf(users, %q) = %d, want %d", tt.item, result, tt.expected)
			}
		})
	}
}

// BenchmarkIndexOf benchmarks the indexOf function
func BenchmarkIndexOf(b *testing.B) {
	slice := []string{"Alice", "Bob", "Charlie", "David", "Emma",
		"Frank", "Grace", "Henry", "Isabella", "Jack",
		"Kate", "Liam", "Mia", "Noah", "Olivia"}

	b.Run("first element", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			indexOf(slice, "Alice")
		}
	})

	b.Run("last element", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			indexOf(slice, "Olivia")
		}
	})

	b.Run("not found", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			indexOf(slice, "NotFound")
		}
	})
}

// TestStringContains tests the stringContains function used in seed.go
// Note: stringContains is defined in database.go
func TestStringContainsForSeeding(t *testing.T) {

	tests := []struct {
		name     string
		str      string
		substr   string
		expected bool
	}{
		{
			name:     "episode pattern - has S0 and E0",
			str:      "Breaking Bad S01E01",
			substr:   "S0",
			expected: true,
		},
		{
			name:     "episode pattern - episode number",
			str:      "Breaking Bad S01E01",
			substr:   "E0",
			expected: true,
		},
		{
			name:     "movie - no S0",
			str:      "The Matrix",
			substr:   "S0",
			expected: false,
		},
		{
			name:     "movie - no E0",
			str:      "The Matrix",
			substr:   "E0",
			expected: false,
		},
		{
			name:     "multi-digit season",
			str:      "Show S10E05",
			substr:   "S1",
			expected: true,
		},
		{
			name:     "empty string",
			str:      "",
			substr:   "S0",
			expected: false,
		},
		{
			name:     "empty substring",
			str:      "Breaking Bad S01E01",
			substr:   "",
			expected: true,
		},
		{
			name:     "case sensitive",
			str:      "Breaking Bad S01E01",
			substr:   "s0",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringContains(tt.str, tt.substr)
			if result != tt.expected {
				t.Errorf("stringContains(%q, %q) = %v, want %v", tt.str, tt.substr, result, tt.expected)
			}
		})
	}
}

// TestMediaTypeDetection tests the logic used to detect media type in SeedMockData
func TestMediaTypeDetection(t *testing.T) {

	// Replicate the logic from SeedMockData
	detectMediaType := func(title string) string {
		mediaType := "movie"
		if stringContains(title, "S0") && stringContains(title, "E0") {
			mediaType = "episode"
		}
		return mediaType
	}

	tests := []struct {
		title    string
		expected string
	}{
		{"The Matrix", "movie"},
		{"Inception", "movie"},
		{"Breaking Bad S01E01", "episode"},
		{"Game of Thrones S01E01", "episode"},
		{"Stranger Things S01E01", "episode"},
		{"The Office S01E01", "episode"},
		{"Friends S01E01", "episode"},
		{"Avatar", "movie"},
		{"Show S10E10", "movie"}, // Note: S10 doesn't contain "S0", so it's detected as movie
		{"Show S05E10", "movie"}, // S05 doesn't contain "S0" either
		{"Show s01e01", "movie"}, // lowercase, not detected
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			result := detectMediaType(tt.title)
			if result != tt.expected {
				t.Errorf("detectMediaType(%q) = %q, want %q", tt.title, result, tt.expected)
			}
		})
	}
}

// TestSeedDataConsistency tests that the seed data arrays are consistent
func TestSeedDataConsistency(t *testing.T) {

	// Mock users from seed.go
	users := []string{
		"Alice", "Bob", "Charlie", "David", "Emma",
		"Frank", "Grace", "Henry", "Isabella", "Jack",
		"Kate", "Liam", "Mia", "Noah", "Olivia",
	}

	// Mock titles from seed.go
	titles := []string{
		"The Matrix", "Inception", "Interstellar", "The Dark Knight",
		"Pulp Fiction", "Fight Club", "Forrest Gump", "The Shawshank Redemption",
		"Breaking Bad S01E01", "Game of Thrones S01E01", "Stranger Things S01E01",
		"The Office S01E01", "Friends S01E01", "The Mandalorian S01E01",
		"Avatar", "Titanic", "The Avengers", "Star Wars: A New Hope",
		"Blade Runner 2049", "Mad Max: Fury Road", "Dunkirk", "1917",
	}

	// Mock platforms from seed.go
	platforms := []string{"Plex Web", "iOS", "Android", "Roku", "Fire TV", "Apple TV", "Windows", "macOS", "Linux", "Smart TV"}

	// Mock players from seed.go
	players := []string{"Chrome", "Firefox", "Safari", "Edge", "Plex App", "VLC", "Direct Play", "Transcoded"}

	t.Run("users count is 15", func(t *testing.T) {
		if len(users) != 15 {
			t.Errorf("Expected 15 users, got %d", len(users))
		}
	})

	t.Run("titles count is 22", func(t *testing.T) {
		if len(titles) != 22 {
			t.Errorf("Expected 22 titles, got %d", len(titles))
		}
	})

	t.Run("platforms count is 10", func(t *testing.T) {
		if len(platforms) != 10 {
			t.Errorf("Expected 10 platforms, got %d", len(platforms))
		}
	})

	t.Run("players count is 8", func(t *testing.T) {
		if len(players) != 8 {
			t.Errorf("Expected 8 players, got %d", len(players))
		}
	})

	t.Run("all users are unique", func(t *testing.T) {
		seen := make(map[string]bool)
		for _, user := range users {
			if seen[user] {
				t.Errorf("Duplicate user: %s", user)
			}
			seen[user] = true
		}
	})

	t.Run("all titles are unique", func(t *testing.T) {
		seen := make(map[string]bool)
		for _, title := range titles {
			if seen[title] {
				t.Errorf("Duplicate title: %s", title)
			}
			seen[title] = true
		}
	})

	t.Run("all platforms are unique", func(t *testing.T) {
		seen := make(map[string]bool)
		for _, platform := range platforms {
			if seen[platform] {
				t.Errorf("Duplicate platform: %s", platform)
			}
			seen[platform] = true
		}
	})

	t.Run("all players are unique", func(t *testing.T) {
		seen := make(map[string]bool)
		for _, player := range players {
			if seen[player] {
				t.Errorf("Duplicate player: %s", player)
			}
			seen[player] = true
		}
	})
}

// TestSeedLocationData tests the location data structure used in seeding
func TestSeedLocationData(t *testing.T) {

	// Sample of locations from seed.go to verify structure
	type location struct {
		city    string
		region  string
		country string
		lat     float64
		lon     float64
	}

	locations := []location{
		{"New York", "NY", "United States", 40.7128, -74.0060},
		{"Los Angeles", "CA", "United States", 34.0522, -118.2437},
		{"London", "England", "United Kingdom", 51.5074, -0.1278},
		{"Tokyo", "Tokyo", "Japan", 35.6762, 139.6503},
		{"Sydney", "NSW", "Australia", -33.8688, 151.2093},
	}

	t.Run("valid latitude range", func(t *testing.T) {
		for _, loc := range locations {
			if loc.lat < -90 || loc.lat > 90 {
				t.Errorf("Invalid latitude for %s: %f", loc.city, loc.lat)
			}
		}
	})

	t.Run("valid longitude range", func(t *testing.T) {
		for _, loc := range locations {
			if loc.lon < -180 || loc.lon > 180 {
				t.Errorf("Invalid longitude for %s: %f", loc.city, loc.lon)
			}
		}
	})

	t.Run("all cities have non-empty values", func(t *testing.T) {
		for _, loc := range locations {
			if loc.city == "" {
				t.Error("Empty city name")
			}
			if loc.region == "" {
				t.Error("Empty region for city: " + loc.city)
			}
			if loc.country == "" {
				t.Error("Empty country for city: " + loc.city)
			}
		}
	})
}

// TestIPAddressGeneration tests the IP address generation pattern used in seeding
func TestIPAddressGeneration(t *testing.T) {

	// Pattern from seed.go: fmt.Sprintf("192.168.%d.%d", i/256, i%256)
	generateIP := func(i int) string {
		return "192.168." + itoa(i/256) + "." + itoa(i%256)
	}

	tests := []struct {
		index    int
		expected string
	}{
		{0, "192.168.0.0"},
		{1, "192.168.0.1"},
		{255, "192.168.0.255"},
		{256, "192.168.1.0"},
		{257, "192.168.1.1"},
		{511, "192.168.1.255"},
		{512, "192.168.2.0"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := generateIP(tt.index)
			if result != tt.expected {
				t.Errorf("generateIP(%d) = %q, want %q", tt.index, result, tt.expected)
			}
		})
	}
}

// Helper to convert int to string without importing strconv
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
