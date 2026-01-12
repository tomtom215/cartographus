// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package cache

import (
	"sync"
	"testing"
)

func TestTrie_BasicOperations(t *testing.T) {
	t.Parallel()

	trie := NewTrie()

	// Test Insert
	if !trie.Insert("hello") {
		t.Error("Insert should return true for new value")
	}
	if trie.Insert("hello") {
		t.Error("Insert should return false for existing value")
	}

	// Test Size
	if trie.Size() != 1 {
		t.Errorf("Size() = %d, want 1", trie.Size())
	}

	// Test Search
	if _, found := trie.Search("hello"); !found {
		t.Error("Search should find 'hello'")
	}
	if _, found := trie.Search("hell"); found {
		t.Error("Search should not find partial match 'hell'")
	}
	if _, found := trie.Search("hello world"); found {
		t.Error("Search should not find 'hello world'")
	}
}

func TestTrie_CaseInsensitive(t *testing.T) {
	t.Parallel()

	trie := NewTrie() // Default is case-insensitive

	trie.Insert("Hello")
	trie.Insert("HELLO")
	trie.Insert("hello")

	// All should refer to the same entry
	if trie.Size() != 1 {
		t.Errorf("Size() = %d, want 1 (case-insensitive)", trie.Size())
	}

	// Search should be case-insensitive
	if _, found := trie.Search("HELLO"); !found {
		t.Error("Search should find 'HELLO' case-insensitively")
	}
	if _, found := trie.Search("HeLLo"); !found {
		t.Error("Search should find 'HeLLo' case-insensitively")
	}
}

func TestTrie_CaseSensitive(t *testing.T) {
	t.Parallel()

	trie := NewTrieWithOptions(true, 10)

	trie.Insert("Hello")
	trie.Insert("hello")
	trie.Insert("HELLO")

	// Each should be a separate entry
	if trie.Size() != 3 {
		t.Errorf("Size() = %d, want 3 (case-sensitive)", trie.Size())
	}

	// Search should be case-sensitive
	if _, found := trie.Search("Hello"); !found {
		t.Error("Search should find 'Hello'")
	}
	if _, found := trie.Search("HELLO"); !found {
		t.Error("Search should find 'HELLO'")
	}
}

func TestTrie_InsertWithData(t *testing.T) {
	t.Parallel()

	trie := NewTrie()

	type MediaInfo struct {
		ID   string
		Year int
	}

	info := MediaInfo{ID: "movie-123", Year: 2024}
	trie.InsertWithData("The Matrix", info)

	data, found := trie.Search("the matrix")
	if !found {
		t.Error("Search should find 'the matrix'")
	}

	retrieved, ok := data.(MediaInfo)
	if !ok {
		t.Error("Data should be MediaInfo type")
	}
	if retrieved.ID != "movie-123" {
		t.Errorf("ID = %s, want 'movie-123'", retrieved.ID)
	}
	if retrieved.Year != 2024 {
		t.Errorf("Year = %d, want 2024", retrieved.Year)
	}
}

func TestTrie_HasPrefix(t *testing.T) {
	t.Parallel()

	trie := NewTrie()

	trie.Insert("hello")
	trie.Insert("help")
	trie.Insert("world")

	tests := []struct {
		prefix string
		want   bool
	}{
		{"hel", true},
		{"hello", true},
		{"he", true},
		{"wor", true},
		{"world", true},
		{"xyz", false},
		{"helloo", false},
		{"", true}, // Empty prefix should return true if trie has any entries
	}

	for _, tt := range tests {
		if got := trie.HasPrefix(tt.prefix); got != tt.want {
			t.Errorf("HasPrefix(%q) = %v, want %v", tt.prefix, got, tt.want)
		}
	}
}

func TestTrie_Autocomplete(t *testing.T) {
	t.Parallel()

	trie := NewTrie()

	// Insert some media titles
	trie.Insert("The Matrix")
	trie.Insert("The Matrix Reloaded")
	trie.Insert("The Matrix Revolutions")
	trie.Insert("The Godfather")
	trie.Insert("The Dark Knight")
	trie.Insert("Inception")

	// Test autocomplete with "the m"
	results := trie.Autocomplete("the m")
	if len(results) != 3 {
		t.Errorf("Autocomplete('the m') returned %d results, want 3", len(results))
	}

	// Verify all results start with the prefix (case-insensitive)
	for _, r := range results {
		lower := toLower(r.Value)
		if !hasPrefix(lower, "the m") {
			t.Errorf("Result %q does not start with 'the m'", r.Value)
		}
	}

	// Test autocomplete with "the"
	results = trie.Autocomplete("the")
	if len(results) != 5 {
		t.Errorf("Autocomplete('the') returned %d results, want 5", len(results))
	}

	// Test autocomplete with non-matching prefix
	results = trie.Autocomplete("xyz")
	if len(results) != 0 {
		t.Errorf("Autocomplete('xyz') returned %d results, want 0", len(results))
	}
}

func TestTrie_AutocompleteWithLimit(t *testing.T) {
	t.Parallel()

	trie := NewTrie()

	// Insert 20 items with same prefix
	for i := 0; i < 20; i++ {
		trie.Insert("test" + string(rune('A'+i)))
	}

	// Default limit should apply
	results := trie.Autocomplete("test")
	if len(results) != 10 {
		t.Errorf("Autocomplete with default limit returned %d, want 10", len(results))
	}

	// Custom limit
	results = trie.AutocompleteWithLimit("test", 5)
	if len(results) != 5 {
		t.Errorf("AutocompleteWithLimit(5) returned %d, want 5", len(results))
	}

	// Large limit
	results = trie.AutocompleteWithLimit("test", 100)
	if len(results) != 20 {
		t.Errorf("AutocompleteWithLimit(100) returned %d, want 20", len(results))
	}
}

func TestTrie_AutocompleteRanking(t *testing.T) {
	t.Parallel()

	trie := NewTrie()

	// Insert same value multiple times
	trie.Insert("popular")
	trie.Insert("popular")
	trie.Insert("popular")
	trie.Insert("popular")

	trie.Insert("less popular")
	trie.Insert("less popular")

	trie.Insert("rare")

	results := trie.Autocomplete("")
	if len(results) != 3 {
		t.Errorf("Autocomplete('') returned %d results, want 3", len(results))
	}

	// Most frequent should be first
	if results[0].Value != "popular" {
		t.Errorf("First result = %q, want 'popular'", results[0].Value)
	}
	if results[0].Count != 4 {
		t.Errorf("First result count = %d, want 4", results[0].Count)
	}

	if results[1].Value != "less popular" {
		t.Errorf("Second result = %q, want 'less popular'", results[1].Value)
	}
	if results[1].Count != 2 {
		t.Errorf("Second result count = %d, want 2", results[1].Count)
	}

	if results[2].Value != "rare" {
		t.Errorf("Third result = %q, want 'rare'", results[2].Value)
	}
	if results[2].Count != 1 {
		t.Errorf("Third result count = %d, want 1", results[2].Count)
	}
}

func TestTrie_Delete(t *testing.T) {
	t.Parallel()

	trie := NewTrie()

	trie.Insert("hello")
	trie.Insert("help")
	trie.Insert("world")

	if trie.Size() != 3 {
		t.Errorf("Size before delete = %d, want 3", trie.Size())
	}

	// Delete existing
	if !trie.Delete("hello") {
		t.Error("Delete('hello') should return true")
	}

	if trie.Size() != 2 {
		t.Errorf("Size after delete = %d, want 2", trie.Size())
	}

	if _, found := trie.Search("hello"); found {
		t.Error("Search should not find deleted 'hello'")
	}

	// "help" should still exist
	if _, found := trie.Search("help"); !found {
		t.Error("Search should still find 'help'")
	}

	// Delete non-existing
	if trie.Delete("notexists") {
		t.Error("Delete('notexists') should return false")
	}
}

func TestTrie_Clear(t *testing.T) {
	t.Parallel()

	trie := NewTrie()

	trie.Insert("hello")
	trie.Insert("world")

	trie.Clear()

	if trie.Size() != 0 {
		t.Errorf("Size after clear = %d, want 0", trie.Size())
	}

	if _, found := trie.Search("hello"); found {
		t.Error("Search should not find anything after clear")
	}
}

func TestTrie_EmptyString(t *testing.T) {
	t.Parallel()

	trie := NewTrie()

	// Empty string should not be inserted
	if trie.Insert("") {
		t.Error("Insert('') should return false")
	}

	if trie.Size() != 0 {
		t.Errorf("Size = %d, want 0", trie.Size())
	}

	// Search for empty string
	if _, found := trie.Search(""); found {
		t.Error("Search('') should return false")
	}

	// Delete empty string
	if trie.Delete("") {
		t.Error("Delete('') should return false")
	}
}

func TestTrie_UnicodeSupport(t *testing.T) {
	t.Parallel()

	trie := NewTrie()

	// Test with various Unicode strings
	trie.Insert("Hello")
	trie.Insert("Cafe")
	trie.Insert("Tokyo")
	trie.Insert("Beijing")

	if trie.Size() != 4 {
		t.Errorf("Size = %d, want 4", trie.Size())
	}

	// Search
	if _, found := trie.Search("cafe"); !found {
		t.Error("Search should find 'cafe'")
	}
	if _, found := trie.Search("tokyo"); !found {
		t.Error("Search should find 'tokyo'")
	}
}

func TestTrie_GetAll(t *testing.T) {
	t.Parallel()

	trie := NewTrie()

	trie.Insert("apple")
	trie.Insert("apple")
	trie.Insert("banana")
	trie.Insert("cherry")

	results := trie.GetAll()
	if len(results) != 3 {
		t.Errorf("GetAll() returned %d results, want 3", len(results))
	}

	// First should be apple (count 2)
	if results[0].Value != "apple" {
		t.Errorf("First result = %q, want 'apple'", results[0].Value)
	}
	if results[0].Count != 2 {
		t.Errorf("First result count = %d, want 2", results[0].Count)
	}
}

func TestTrie_Concurrent(t *testing.T) {
	t.Parallel()

	trie := NewTrie()

	var wg sync.WaitGroup
	numGoroutines := 100
	numOps := 100

	// Concurrent inserts
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := string(rune('a' + (id % 26)))
				trie.Insert(key)
			}
		}(i)
	}

	wg.Wait()

	// Verify size is correct (26 unique letters)
	if trie.Size() != 26 {
		t.Errorf("Size = %d, want 26", trie.Size())
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := string(rune('a' + (id % 26)))
				trie.Search(key)
				trie.HasPrefix(key)
				trie.Autocomplete(key)
			}
		}(i)
	}

	wg.Wait()
}

func TestTrieIndex_BasicOperations(t *testing.T) {
	t.Parallel()

	idx := NewTrieIndex()

	// GetOrCreate should create new trie
	titleTrie := idx.GetOrCreate("title")
	if titleTrie == nil {
		t.Error("GetOrCreate should return a non-nil trie")
	}

	// GetOrCreate again should return same trie
	sameTrie := idx.GetOrCreate("title")
	if sameTrie != titleTrie {
		t.Error("GetOrCreate should return the same trie instance")
	}

	// Create another trie
	userTrie := idx.GetOrCreate("username")
	if userTrie == titleTrie {
		t.Error("Different fields should have different tries")
	}

	// Get should return existing trie
	if idx.Get("title") != titleTrie {
		t.Error("Get should return the existing trie")
	}

	// Get should return nil for non-existing
	if idx.Get("nonexistent") != nil {
		t.Error("Get should return nil for non-existing field")
	}

	// Stats
	titleTrie.Insert("test1")
	titleTrie.Insert("test2")
	userTrie.Insert("user1")

	stats := idx.Stats()
	if stats["title"] != 2 {
		t.Errorf("Stats['title'] = %d, want 2", stats["title"])
	}
	if stats["username"] != 1 {
		t.Errorf("Stats['username'] = %d, want 1", stats["username"])
	}
}

func TestTrieIndex_Clear(t *testing.T) {
	t.Parallel()

	idx := NewTrieIndex()

	idx.GetOrCreate("field1").Insert("value1")
	idx.GetOrCreate("field2").Insert("value2")

	idx.Clear()

	stats := idx.Stats()
	if len(stats) != 0 {
		t.Errorf("Stats after clear has %d entries, want 0", len(stats))
	}

	if idx.Get("field1") != nil {
		t.Error("Get should return nil after clear")
	}
}

// Helper functions for tests
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && toLower(s[:len(prefix)]) == toLower(prefix)
}

// Benchmark tests
func BenchmarkTrie_Insert(b *testing.B) {
	trie := NewTrie()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		trie.Insert("test string for benchmark")
	}
}

func BenchmarkTrie_Search(b *testing.B) {
	trie := NewTrie()
	trie.Insert("test string for benchmark")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		trie.Search("test string for benchmark")
	}
}

func BenchmarkTrie_Autocomplete(b *testing.B) {
	trie := NewTrie()

	// Insert 1000 items
	for i := 0; i < 1000; i++ {
		trie.Insert("prefix" + string(rune('a'+i%26)) + string(rune('0'+i%10)))
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		trie.Autocomplete("prefix")
	}
}

func BenchmarkTrie_HasPrefix(b *testing.B) {
	trie := NewTrie()

	for i := 0; i < 1000; i++ {
		trie.Insert("prefix" + string(rune('a'+i%26)))
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		trie.HasPrefix("pre")
	}
}
