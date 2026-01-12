// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package cache provides high-performance data structures for caching and deduplication.
package cache

import (
	"sort"
	"strings"
	"sync"
)

// TrieNode represents a node in the Trie.
type TrieNode struct {
	children map[rune]*TrieNode
	isEnd    bool   // Marks end of a complete word
	value    string // The complete string stored at this node (if isEnd is true)
	data     any    // Optional associated data (e.g., ID, metadata)
	count    int    // Number of times this prefix has been inserted (for ranking)
}

// Trie implements a thread-safe prefix tree (trie) for efficient autocomplete.
// It provides O(m) operations where m is the length of the query string.
//
// Key features:
//   - O(m) insert, search, and prefix lookup where m = string length
//   - Case-insensitive matching (configurable)
//   - Returns results sorted by frequency (most inserted first)
//   - Thread-safe operations
//   - Optional associated data per entry
//
// Use cases:
//   - Autocomplete for media titles
//   - Username/email prefix suggestions
//   - Fast prefix-based filtering before expensive fuzzy search
type Trie struct {
	mu             sync.RWMutex
	root           *TrieNode
	size           int  // Number of complete words
	caseSensitive  bool // If false, all keys are lowercased
	maxSuggestions int  // Maximum suggestions to return (default 10)
}

// TrieResult represents a match from the Trie with associated data.
type TrieResult struct {
	Value string // The matched string
	Data  any    // Associated data (may be nil)
	Count int    // Number of times this value was inserted
}

// NewTrie creates a new Trie with default settings (case-insensitive, max 10 suggestions).
func NewTrie() *Trie {
	return &Trie{
		root:           newTrieNode(),
		caseSensitive:  false,
		maxSuggestions: 10,
	}
}

// NewTrieWithOptions creates a new Trie with custom settings.
func NewTrieWithOptions(caseSensitive bool, maxSuggestions int) *Trie {
	if maxSuggestions <= 0 {
		maxSuggestions = 10
	}
	return &Trie{
		root:           newTrieNode(),
		caseSensitive:  caseSensitive,
		maxSuggestions: maxSuggestions,
	}
}

// newTrieNode creates a new TrieNode.
func newTrieNode() *TrieNode {
	return &TrieNode{
		children: make(map[rune]*TrieNode),
	}
}

// normalizeKey normalizes the key based on case sensitivity setting.
func (t *Trie) normalizeKey(key string) string {
	if t.caseSensitive {
		return key
	}
	return strings.ToLower(key)
}

// Insert adds a string to the Trie.
// If the string already exists, increments its count.
// Returns true if this is a new insertion, false if it already existed.
func (t *Trie) Insert(value string) bool {
	return t.InsertWithData(value, nil)
}

// InsertWithData adds a string to the Trie with associated data.
// If the string already exists, increments its count and updates the data.
// Returns true if this is a new insertion, false if it already existed.
func (t *Trie) InsertWithData(value string, data any) bool {
	if value == "" {
		return false
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	key := t.normalizeKey(value)
	node := t.root

	// Traverse/create path through trie
	for _, ch := range key {
		if node.children[ch] == nil {
			node.children[ch] = newTrieNode()
		}
		node = node.children[ch]
	}

	isNew := !node.isEnd

	node.isEnd = true
	node.value = value // Store original value (preserves case)
	node.data = data
	node.count++

	if isNew {
		t.size++
	}

	return isNew
}

// Search checks if an exact string exists in the Trie.
// Returns the associated data and true if found, nil and false otherwise.
func (t *Trie) Search(value string) (any, bool) {
	if value == "" {
		return nil, false
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	key := t.normalizeKey(value)
	node := t.root

	for _, ch := range key {
		if node.children[ch] == nil {
			return nil, false
		}
		node = node.children[ch]
	}

	if node.isEnd {
		return node.data, true
	}
	return nil, false
}

// HasPrefix checks if any string in the Trie starts with the given prefix.
func (t *Trie) HasPrefix(prefix string) bool {
	if prefix == "" {
		return t.size > 0
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	key := t.normalizeKey(prefix)
	node := t.root

	for _, ch := range key {
		if node.children[ch] == nil {
			return false
		}
		node = node.children[ch]
	}

	return true
}

// Autocomplete returns all strings that start with the given prefix.
// Results are sorted by count (most frequently inserted first), then alphabetically.
// Limited to maxSuggestions results (default 10).
func (t *Trie) Autocomplete(prefix string) []TrieResult {
	return t.AutocompleteWithLimit(prefix, t.maxSuggestions)
}

// AutocompleteWithLimit returns strings starting with prefix, limited to n results.
func (t *Trie) AutocompleteWithLimit(prefix string, limit int) []TrieResult {
	if limit <= 0 {
		limit = t.maxSuggestions
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	// Find the node at the end of the prefix
	key := t.normalizeKey(prefix)
	node := t.root

	for _, ch := range key {
		if node.children[ch] == nil {
			return nil // No matches
		}
		node = node.children[ch]
	}

	// Collect all words from this point
	var results []TrieResult
	t.collectWords(node, &results)

	// Sort by count (descending), then alphabetically
	sort.Slice(results, func(i, j int) bool {
		if results[i].Count != results[j].Count {
			return results[i].Count > results[j].Count
		}
		return results[i].Value < results[j].Value
	})

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results
}

// collectWords recursively collects all complete words from a node.
func (t *Trie) collectWords(node *TrieNode, results *[]TrieResult) {
	if node == nil {
		return
	}

	if node.isEnd {
		*results = append(*results, TrieResult{
			Value: node.value,
			Data:  node.data,
			Count: node.count,
		})
	}

	for _, child := range node.children {
		t.collectWords(child, results)
	}
}

// Delete removes a string from the Trie.
// Returns true if the string was found and removed.
func (t *Trie) Delete(value string) bool {
	if value == "" {
		return false
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	key := t.normalizeKey(value)
	return t.deleteRecursive(t.root, key, 0)
}

// deleteRecursive removes a string and cleans up empty nodes.
func (t *Trie) deleteRecursive(node *TrieNode, key string, depth int) bool {
	if node == nil {
		return false
	}

	if depth == len(key) {
		if !node.isEnd {
			return false
		}

		node.isEnd = false
		node.value = ""
		node.data = nil
		node.count = 0
		t.size--
		return true
	}

	ch := rune(key[depth])
	child := node.children[ch]
	if child == nil {
		return false
	}

	deleted := t.deleteRecursive(child, key, depth+1)

	// Clean up empty child nodes
	if deleted && !child.isEnd && len(child.children) == 0 {
		delete(node.children, ch)
	}

	return deleted
}

// Size returns the number of complete strings in the Trie.
func (t *Trie) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.size
}

// Clear removes all entries from the Trie.
func (t *Trie) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.root = newTrieNode()
	t.size = 0
}

// GetAll returns all strings in the Trie.
// Results are sorted by count (descending), then alphabetically.
func (t *Trie) GetAll() []TrieResult {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var results []TrieResult
	t.collectWords(t.root, &results)

	// Sort by count (descending), then alphabetically
	sort.Slice(results, func(i, j int) bool {
		if results[i].Count != results[j].Count {
			return results[i].Count > results[j].Count
		}
		return results[i].Value < results[j].Value
	})

	return results
}

// TrieIndex provides a concurrent-safe index for fast prefix lookups.
// It wraps a Trie and provides methods for bulk operations and rebuilding.
type TrieIndex struct {
	mu    sync.RWMutex
	tries map[string]*Trie // Multiple tries for different fields (e.g., "title", "username")
}

// NewTrieIndex creates a new TrieIndex.
func NewTrieIndex() *TrieIndex {
	return &TrieIndex{
		tries: make(map[string]*Trie),
	}
}

// GetOrCreate gets or creates a Trie for the given field name.
func (idx *TrieIndex) GetOrCreate(field string) *Trie {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if t, exists := idx.tries[field]; exists {
		return t
	}

	t := NewTrie()
	idx.tries[field] = t
	return t
}

// Get returns the Trie for the given field, or nil if not found.
func (idx *TrieIndex) Get(field string) *Trie {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.tries[field]
}

// Clear removes all tries from the index.
func (idx *TrieIndex) Clear() {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.tries = make(map[string]*Trie)
}

// Stats returns statistics about the index.
func (idx *TrieIndex) Stats() map[string]int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	stats := make(map[string]int)
	for field, trie := range idx.tries {
		stats[field] = trie.Size()
	}
	return stats
}
