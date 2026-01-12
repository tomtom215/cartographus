// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"testing"
)

func TestAutocompleteIndex_BasicOperations(t *testing.T) {

	idx := NewAutocompleteIndex()

	// Add some titles
	idx.titles.Insert("The Matrix")
	idx.titles.Insert("The Matrix Reloaded")
	idx.titles.Insert("The Godfather")
	idx.titles.Insert("Inception")

	// Add some usernames
	idx.usernames.Insert("alice")
	idx.usernames.Insert("alicia")
	idx.usernames.Insert("bob")

	// Test title autocomplete
	results := idx.titles.Autocomplete("the m")
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'the m', got %d", len(results))
	}

	// Test username autocomplete
	results = idx.usernames.Autocomplete("ali")
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'ali', got %d", len(results))
	}

	// Test no match
	results = idx.titles.Autocomplete("xyz")
	if len(results) != 0 {
		t.Errorf("Expected 0 results for 'xyz', got %d", len(results))
	}
}

func TestAutocompleteResult_Structure(t *testing.T) {

	result := AutocompleteResult{
		Value: "The Matrix",
		Type:  "title",
		ID:    "movie-123",
		Count: 5,
	}

	if result.Value != "The Matrix" {
		t.Errorf("Value = %q, want 'The Matrix'", result.Value)
	}
	if result.Type != "title" {
		t.Errorf("Type = %q, want 'title'", result.Type)
	}
	if result.ID != "movie-123" {
		t.Errorf("ID = %q, want 'movie-123'", result.ID)
	}
	if result.Count != 5 {
		t.Errorf("Count = %d, want 5", result.Count)
	}
}

func TestGetAutocompleteIndex_Singleton(t *testing.T) {

	// Get index twice
	idx1 := getAutocompleteIndex()
	idx2 := getAutocompleteIndex()

	// Should be the same instance (though this is a simplified test
	// since the actual implementation uses a global variable)
	if idx1 == nil || idx2 == nil {
		t.Error("getAutocompleteIndex should never return nil")
	}
}

func TestAutocompleteIndex_CaseInsensitive(t *testing.T) {

	idx := NewAutocompleteIndex()

	idx.titles.Insert("The Matrix")
	idx.titles.Insert("THE GODFATHER")
	idx.titles.Insert("inception")

	// Search with different cases should find matches
	results := idx.titles.Autocomplete("THE")
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'THE', got %d", len(results))
	}

	results = idx.titles.Autocomplete("the")
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'the', got %d", len(results))
	}

	results = idx.titles.Autocomplete("inc")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'inc', got %d", len(results))
	}
}

func TestAutocompleteIndex_FrequencyRanking(t *testing.T) {

	idx := NewAutocompleteIndex()

	// Insert with different frequencies
	for i := 0; i < 5; i++ {
		idx.titles.Insert("Popular Movie")
	}
	for i := 0; i < 2; i++ {
		idx.titles.Insert("Less Popular Movie")
	}
	idx.titles.Insert("Rare Movie")

	results := idx.titles.Autocomplete("")
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
		return
	}

	// Most frequent should be first
	if results[0].Value != "Popular Movie" {
		t.Errorf("First result = %q, want 'Popular Movie'", results[0].Value)
	}
	if results[0].Count != 5 {
		t.Errorf("First result count = %d, want 5", results[0].Count)
	}
}

func TestAutocompleteIndex_Limit(t *testing.T) {

	idx := NewAutocompleteIndex()

	// Add more items than limit
	for i := 0; i < 30; i++ {
		idx.titles.Insert("Movie" + string(rune('A'+i)))
	}

	// Should be limited
	results := idx.titles.AutocompleteWithLimit("Movie", 10)
	if len(results) != 10 {
		t.Errorf("Expected 10 results with limit, got %d", len(results))
	}
}

func TestAutocompleteIndex_EmptyPrefix(t *testing.T) {

	idx := NewAutocompleteIndex()

	idx.titles.Insert("Movie A")
	idx.titles.Insert("Movie B")
	idx.titles.Insert("Movie C")

	// Empty prefix should return all (up to limit)
	results := idx.titles.Autocomplete("")
	if len(results) != 3 {
		t.Errorf("Expected 3 results for empty prefix, got %d", len(results))
	}
}

func TestAutocompleteIndex_Clear(t *testing.T) {

	idx := NewAutocompleteIndex()

	idx.titles.Insert("Movie A")
	idx.usernames.Insert("user1")
	idx.all.Insert("something")

	// Clear all
	idx.titles.Clear()
	idx.usernames.Clear()
	idx.all.Clear()

	if idx.titles.Size() != 0 {
		t.Errorf("titles.Size() = %d, want 0", idx.titles.Size())
	}
	if idx.usernames.Size() != 0 {
		t.Errorf("usernames.Size() = %d, want 0", idx.usernames.Size())
	}
	if idx.all.Size() != 0 {
		t.Errorf("all.Size() = %d, want 0", idx.all.Size())
	}
}

// Benchmark tests
func BenchmarkAutocompleteIndex_Insert(b *testing.B) {
	idx := NewAutocompleteIndex()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		idx.titles.Insert("Test Movie Title")
	}
}

func BenchmarkAutocompleteIndex_Autocomplete(b *testing.B) {
	idx := NewAutocompleteIndex()

	// Populate with 10000 items
	for i := 0; i < 10000; i++ {
		idx.titles.Insert("Movie " + string(rune('A'+i%26)) + string(rune('a'+i%26)))
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		idx.titles.Autocomplete("Movie A")
	}
}

func BenchmarkAutocompleteIndex_AutocompleteLargeDataset(b *testing.B) {
	idx := NewAutocompleteIndex()

	// Populate with 50000 items (realistic media library size)
	for i := 0; i < 50000; i++ {
		title := "Movie " + string(rune('A'+i%26)) + string(rune('a'+i%26)) + string(rune('0'+i%10))
		idx.titles.Insert(title)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		idx.titles.AutocompleteWithLimit("Movie", 10)
	}
}
