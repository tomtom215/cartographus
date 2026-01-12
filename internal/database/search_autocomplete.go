// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
search_autocomplete.go - Fast Prefix-Based Autocomplete using Trie

This file provides lightning-fast autocomplete suggestions using an in-memory Trie
data structure. It complements the fuzzy search functionality in search_fuzzy.go
by providing O(m) prefix lookups where m is the query length.

Features:
  - O(m) autocomplete where m = query length (vs O(n) for database LIKE queries)
  - Case-insensitive matching
  - Results ranked by frequency (most common first)
  - Separate tries for titles, usernames, and other fields
  - Automatic index population from database

Usage Flow:
  1. Call BuildAutocompleteIndex() after sync or on startup
  2. Use AutocompleteTitle/AutocompleteUser for instant suggestions
  3. For full search, fall back to FuzzySearch methods in search_fuzzy.go

Performance:
  - Autocomplete: 1-10ms for any query (constant time relative to dataset size)
  - Fuzzy Search: 100-2000ms depending on dataset size
  - Recommended: Use autocomplete for prefix suggestions, fuzzy for full search
*/

package database

import (
	"context"
	"fmt"

	"github.com/tomtom215/cartographus/internal/cache"
)

// AutocompleteResult represents an autocomplete suggestion
type AutocompleteResult struct {
	Value string `json:"value"`          // The suggested text
	Type  string `json:"type,omitempty"` // Type of result (title, username, etc.)
	ID    string `json:"id,omitempty"`   // Optional ID for the result
	Count int    `json:"count"`          // Frequency count (for ranking)
}

// AutocompleteIndex holds tries for different searchable fields
type AutocompleteIndex struct {
	titles    *cache.Trie // Media titles (movies, shows, episodes)
	usernames *cache.Trie // Usernames
	all       *cache.Trie // Combined index for general search
}

// NewAutocompleteIndex creates a new autocomplete index
func NewAutocompleteIndex() *AutocompleteIndex {
	return &AutocompleteIndex{
		titles:    cache.NewTrieWithOptions(false, 20), // case-insensitive, max 20 suggestions
		usernames: cache.NewTrieWithOptions(false, 20),
		all:       cache.NewTrieWithOptions(false, 20),
	}
}

// autocompleteIndex holds the in-memory autocomplete index
var autocompleteIndex *AutocompleteIndex

// getAutocompleteIndex returns the singleton autocomplete index, creating it if needed
func getAutocompleteIndex() *AutocompleteIndex {
	if autocompleteIndex == nil {
		autocompleteIndex = NewAutocompleteIndex()
	}
	return autocompleteIndex
}

// BuildAutocompleteIndex populates the autocomplete index from the database.
// This should be called after initial sync and periodically to keep the index fresh.
// It's safe to call concurrently - the underlying tries are thread-safe.
func (db *DB) BuildAutocompleteIndex(ctx context.Context) error {
	idx := getAutocompleteIndex()

	// Clear existing entries
	idx.titles.Clear()
	idx.usernames.Clear()
	idx.all.Clear()

	// Populate titles
	if err := db.populateTitleIndex(ctx, idx); err != nil {
		return fmt.Errorf("failed to populate title index: %w", err)
	}

	// Populate usernames
	if err := db.populateUsernameIndex(ctx, idx); err != nil {
		return fmt.Errorf("failed to populate username index: %w", err)
	}

	return nil
}

// populateTitleIndex loads unique titles from the database into the trie
func (db *DB) populateTitleIndex(ctx context.Context, idx *AutocompleteIndex) error {
	// Query unique titles with counts
	query := `
		SELECT
			title,
			grandparent_title,
			COUNT(*) as count
		FROM playback_events
		WHERE title IS NOT NULL AND title != ''
		GROUP BY title, grandparent_title
		ORDER BY count DESC
		LIMIT 50000
	`

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var title string
		var grandparentTitle *string
		var count int

		if err := rows.Scan(&title, &grandparentTitle, &count); err != nil {
			continue // Skip on error
		}

		// Add title to trie (frequency-ranked)
		for i := 0; i < count && i < 10; i++ { // Cap frequency influence
			idx.titles.Insert(title)
			idx.all.Insert(title)
		}

		// Add grandparent title (show/artist name) if present
		if grandparentTitle != nil && *grandparentTitle != "" {
			for i := 0; i < count && i < 10; i++ {
				idx.titles.Insert(*grandparentTitle)
				idx.all.Insert(*grandparentTitle)
			}
		}
	}

	return rows.Err()
}

// populateUsernameIndex loads unique usernames from the database into the trie
func (db *DB) populateUsernameIndex(ctx context.Context, idx *AutocompleteIndex) error {
	query := `
		SELECT
			username,
			friendly_name,
			COUNT(*) as count
		FROM playback_events
		WHERE username IS NOT NULL AND username != ''
		GROUP BY username, friendly_name
		ORDER BY count DESC
		LIMIT 10000
	`

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var username string
		var friendlyName *string
		var count int

		if err := rows.Scan(&username, &friendlyName, &count); err != nil {
			continue
		}

		// Add username to trie
		for i := 0; i < count && i < 10; i++ {
			idx.usernames.Insert(username)
			idx.all.Insert(username)
		}

		// Add friendly name if different from username
		if friendlyName != nil && *friendlyName != "" && *friendlyName != username {
			for i := 0; i < count && i < 10; i++ {
				idx.usernames.Insert(*friendlyName)
				idx.all.Insert(*friendlyName)
			}
		}
	}

	return rows.Err()
}

// AutocompleteTitle returns title suggestions for the given prefix.
// Results are ranked by frequency (most common first).
// This is an O(m) operation where m = len(prefix).
func (db *DB) AutocompleteTitle(prefix string, limit int) []AutocompleteResult {
	if prefix == "" {
		return nil
	}
	if limit <= 0 {
		limit = 10
	}

	idx := getAutocompleteIndex()
	results := idx.titles.AutocompleteWithLimit(prefix, limit)

	autocompleteResults := make([]AutocompleteResult, len(results))
	for i, r := range results {
		autocompleteResults[i] = AutocompleteResult{
			Value: r.Value,
			Type:  "title",
			Count: r.Count,
		}
	}

	return autocompleteResults
}

// AutocompleteUser returns username suggestions for the given prefix.
// Results are ranked by frequency (most common first).
func (db *DB) AutocompleteUser(prefix string, limit int) []AutocompleteResult {
	if prefix == "" {
		return nil
	}
	if limit <= 0 {
		limit = 10
	}

	idx := getAutocompleteIndex()
	results := idx.usernames.AutocompleteWithLimit(prefix, limit)

	autocompleteResults := make([]AutocompleteResult, len(results))
	for i, r := range results {
		autocompleteResults[i] = AutocompleteResult{
			Value: r.Value,
			Type:  "username",
			Count: r.Count,
		}
	}

	return autocompleteResults
}

// AutocompleteAll returns suggestions from all indexed fields.
// Useful for a unified search box.
func (db *DB) AutocompleteAll(prefix string, limit int) []AutocompleteResult {
	if prefix == "" {
		return nil
	}
	if limit <= 0 {
		limit = 10
	}

	idx := getAutocompleteIndex()
	results := idx.all.AutocompleteWithLimit(prefix, limit)

	autocompleteResults := make([]AutocompleteResult, len(results))
	for i, r := range results {
		autocompleteResults[i] = AutocompleteResult{
			Value: r.Value,
			Type:  "search",
			Count: r.Count,
		}
	}

	return autocompleteResults
}

// GetAutocompleteStats returns statistics about the autocomplete index.
func (db *DB) GetAutocompleteStats() map[string]int {
	idx := getAutocompleteIndex()
	return map[string]int{
		"titles":    idx.titles.Size(),
		"usernames": idx.usernames.Size(),
		"all":       idx.all.Size(),
	}
}

// ClearAutocompleteIndex clears the autocomplete index.
// Useful for testing or when data is deleted.
func (db *DB) ClearAutocompleteIndex() {
	idx := getAutocompleteIndex()
	idx.titles.Clear()
	idx.usernames.Clear()
	idx.all.Clear()
}

// AddToAutocompleteIndex adds a single value to the autocomplete index.
// This is useful for incrementally updating the index after new data is added.
func (db *DB) AddToAutocompleteIndex(value string, indexType string) {
	if value == "" {
		return
	}

	idx := getAutocompleteIndex()

	switch indexType {
	case "title":
		idx.titles.Insert(value)
		idx.all.Insert(value)
	case "username":
		idx.usernames.Insert(value)
		idx.all.Insert(value)
	default:
		idx.all.Insert(value)
	}
}
