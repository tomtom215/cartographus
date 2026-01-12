// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package cache provides high-performance data structures for caching and deduplication.
package cache

import (
	"strings"
	"sync"
)

// AhoCorasick implements the Aho-Corasick string matching algorithm.
// It efficiently finds all occurrences of multiple patterns in a text
// in O(n + m + z) time, where:
//   - n = length of text
//   - m = total length of all patterns
//   - z = number of matches
//
// This is much faster than checking each pattern individually (O(n * numPatterns)).
//
// Use cases:
//   - User agent detection: Match against known VPN/bot/crawler signatures
//   - Content filtering: Find prohibited words/phrases
//   - Log analysis: Extract known error patterns
//   - Security: Detect malicious patterns in requests
//
// Example:
//
//	ac := NewAhoCorasick()
//	ac.AddPattern("vpn", "vpn_service")
//	ac.AddPattern("proxy", "proxy_service")
//	ac.AddPattern("tor", "tor_network")
//	ac.Build()
//
//	matches := ac.Search("Mozilla/5.0 VPN Client")
//	// matches contains Match{Pattern: "vpn", Data: "vpn_service", Position: 12}
type AhoCorasick struct {
	mu            sync.RWMutex
	root          *acNode
	patterns      []Pattern
	built         bool
	caseSensitive bool
}

// acNode represents a node in the Aho-Corasick automaton.
type acNode struct {
	children map[rune]*acNode
	failure  *acNode // Failure link for when match fails
	output   []int   // Indices of patterns that end at this node
	depth    int     // Depth from root
}

// Pattern represents a search pattern with associated data.
type Pattern struct {
	Text string // The pattern text
	Data any    // Optional associated data (e.g., category, severity)
}

// Match represents a pattern match in the text.
type Match struct {
	Pattern  string // The matched pattern
	Data     any    // Associated data from the pattern
	Position int    // Start position in the text
}

// NewAhoCorasick creates a new Aho-Corasick automaton.
// By default, matching is case-insensitive.
func NewAhoCorasick() *AhoCorasick {
	return &AhoCorasick{
		root:          newACNode(0),
		caseSensitive: false,
	}
}

// NewAhoCorasickCaseSensitive creates a case-sensitive automaton.
func NewAhoCorasickCaseSensitive() *AhoCorasick {
	return &AhoCorasick{
		root:          newACNode(0),
		caseSensitive: true,
	}
}

// newACNode creates a new automaton node.
func newACNode(depth int) *acNode {
	return &acNode{
		children: make(map[rune]*acNode),
		output:   make([]int, 0),
		depth:    depth,
	}
}

// AddPattern adds a pattern to the automaton.
// Must be called before Build().
func (ac *AhoCorasick) AddPattern(pattern string, data any) {
	if pattern == "" {
		return
	}

	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.built {
		ac.built = false // Need to rebuild
	}

	ac.patterns = append(ac.patterns, Pattern{Text: pattern, Data: data})
}

// AddPatterns adds multiple patterns at once.
func (ac *AhoCorasick) AddPatterns(patterns []string, data any) {
	for _, p := range patterns {
		ac.AddPattern(p, data)
	}
}

// Build constructs the automaton. Must be called after adding patterns
// and before searching.
func (ac *AhoCorasick) Build() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.built {
		return
	}

	// Reset the trie
	ac.root = newACNode(0)

	// Build the trie from patterns
	for i, p := range ac.patterns {
		ac.insertPattern(i, p.Text)
	}

	// Build failure links using BFS
	ac.buildFailureLinks()

	ac.built = true
}

// insertPattern inserts a pattern into the trie.
func (ac *AhoCorasick) insertPattern(index int, pattern string) {
	node := ac.root

	text := pattern
	if !ac.caseSensitive {
		text = strings.ToLower(pattern)
	}

	for _, ch := range text {
		if node.children[ch] == nil {
			node.children[ch] = newACNode(node.depth + 1)
		}
		node = node.children[ch]
	}

	node.output = append(node.output, index)
}

// buildFailureLinks builds failure links using BFS.
func (ac *AhoCorasick) buildFailureLinks() {
	// Root's children fail to root
	queue := make([]*acNode, 0)
	for _, child := range ac.root.children {
		child.failure = ac.root
		queue = append(queue, child)
	}

	// BFS to build failure links
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for ch, child := range current.children {
			queue = append(queue, child)

			// Follow failure links to find longest proper suffix
			fail := current.failure
			for fail != nil && fail.children[ch] == nil {
				fail = fail.failure
			}

			if fail == nil {
				child.failure = ac.root
			} else {
				child.failure = fail.children[ch]
				// Merge output from failure link
				child.output = append(child.output, child.failure.output...)
			}
		}
	}
}

// Search finds all pattern matches in the text.
// Returns all matches with their positions.
func (ac *AhoCorasick) Search(text string) []Match {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	if !ac.built || len(ac.patterns) == 0 {
		return nil
	}

	searchText := text
	if !ac.caseSensitive {
		searchText = strings.ToLower(text)
	}

	var matches []Match
	node := ac.root

	for i, ch := range searchText {
		// Follow failure links until we find a match or reach root
		for node != nil && node.children[ch] == nil {
			node = node.failure
		}

		if node == nil {
			node = ac.root
			continue
		}

		node = node.children[ch]

		// Collect all patterns that match at this position
		for _, patternIdx := range node.output {
			pattern := ac.patterns[patternIdx]
			matches = append(matches, Match{
				Pattern:  pattern.Text,
				Data:     pattern.Data,
				Position: i - len(pattern.Text) + 1,
			})
		}
	}

	return matches
}

// SearchFirst finds the first pattern match in the text.
// More efficient than Search when you only need one match.
func (ac *AhoCorasick) SearchFirst(text string) (Match, bool) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	if !ac.built || len(ac.patterns) == 0 {
		return Match{}, false
	}

	searchText := text
	if !ac.caseSensitive {
		searchText = strings.ToLower(text)
	}

	node := ac.root

	for i, ch := range searchText {
		for node != nil && node.children[ch] == nil {
			node = node.failure
		}

		if node == nil {
			node = ac.root
			continue
		}

		node = node.children[ch]

		if len(node.output) > 0 {
			patternIdx := node.output[0]
			pattern := ac.patterns[patternIdx]
			return Match{
				Pattern:  pattern.Text,
				Data:     pattern.Data,
				Position: i - len(pattern.Text) + 1,
			}, true
		}
	}

	return Match{}, false
}

// Contains checks if any pattern matches in the text.
// Most efficient when you only need a boolean result.
func (ac *AhoCorasick) Contains(text string) bool {
	_, found := ac.SearchFirst(text)
	return found
}

// MatchCount returns the number of pattern matches in the text.
func (ac *AhoCorasick) MatchCount(text string) int {
	matches := ac.Search(text)
	return len(matches)
}

// PatternCount returns the number of patterns in the automaton.
func (ac *AhoCorasick) PatternCount() int {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return len(ac.patterns)
}

// Clear removes all patterns and resets the automaton.
func (ac *AhoCorasick) Clear() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.root = newACNode(0)
	ac.patterns = nil
	ac.built = false
}

// PatternMatcher provides a convenient way to create and use an Aho-Corasick
// automaton for common detection use cases.
type PatternMatcher struct {
	ac *AhoCorasick
}

// NewPatternMatcher creates a new pattern matcher with the given patterns.
// The automaton is built automatically.
func NewPatternMatcher(patterns map[string]any) *PatternMatcher {
	ac := NewAhoCorasick()
	for pattern, data := range patterns {
		ac.AddPattern(pattern, data)
	}
	ac.Build()

	return &PatternMatcher{ac: ac}
}

// NewPatternMatcherFromSlice creates a matcher from a slice of patterns.
// All patterns are associated with the same data value.
func NewPatternMatcherFromSlice(patterns []string, data any) *PatternMatcher {
	ac := NewAhoCorasick()
	ac.AddPatterns(patterns, data)
	ac.Build()

	return &PatternMatcher{ac: ac}
}

// Match returns all matches in the text.
func (pm *PatternMatcher) Match(text string) []Match {
	return pm.ac.Search(text)
}

// MatchFirst returns the first match in the text.
func (pm *PatternMatcher) MatchFirst(text string) (Match, bool) {
	return pm.ac.SearchFirst(text)
}

// Contains returns true if any pattern matches.
func (pm *PatternMatcher) Contains(text string) bool {
	return pm.ac.Contains(text)
}

// UserAgentDetector provides specialized pattern matching for user agent strings.
type UserAgentDetector struct {
	vpnMatcher     *PatternMatcher
	botMatcher     *PatternMatcher
	crawlerMatcher *PatternMatcher
}

// NewUserAgentDetector creates a detector with common patterns.
func NewUserAgentDetector() *UserAgentDetector {
	// Common VPN patterns
	vpnPatterns := []string{
		"vpn", "proxy", "tunnel", "anonymizer", "hidemyass",
		"expressvpn", "nordvpn", "surfshark", "cyberghost",
		"privateinternetaccess", "windscribe", "protonvpn",
		"openvpn", "wireguard", "shadowsocks",
	}

	// Common bot patterns
	botPatterns := []string{
		"bot", "spider", "crawler", "scraper", "headless",
		"phantom", "selenium", "puppeteer", "playwright",
		"httpclient", "python-requests", "curl", "wget",
		"java/", "go-http-client", "axios", "fetch",
	}

	// Common crawler patterns
	crawlerPatterns := []string{
		"googlebot", "bingbot", "slurp", "duckduckbot",
		"baiduspider", "yandexbot", "sogou", "exabot",
		"facebookexternalhit", "twitterbot", "linkedinbot",
		"applebot", "ahrefsbot", "semrushbot",
	}

	return &UserAgentDetector{
		vpnMatcher:     NewPatternMatcherFromSlice(vpnPatterns, "vpn"),
		botMatcher:     NewPatternMatcherFromSlice(botPatterns, "bot"),
		crawlerMatcher: NewPatternMatcherFromSlice(crawlerPatterns, "crawler"),
	}
}

// DetectionResult contains user agent detection results.
type DetectionResult struct {
	IsVPN     bool
	IsBot     bool
	IsCrawler bool
	Matches   []Match
}

// Detect analyzes a user agent string.
func (d *UserAgentDetector) Detect(userAgent string) DetectionResult {
	result := DetectionResult{}

	if matches := d.vpnMatcher.Match(userAgent); len(matches) > 0 {
		result.IsVPN = true
		result.Matches = append(result.Matches, matches...)
	}

	if matches := d.botMatcher.Match(userAgent); len(matches) > 0 {
		result.IsBot = true
		result.Matches = append(result.Matches, matches...)
	}

	if matches := d.crawlerMatcher.Match(userAgent); len(matches) > 0 {
		result.IsCrawler = true
		result.Matches = append(result.Matches, matches...)
	}

	return result
}

// IsVPN checks if the user agent indicates VPN usage.
func (d *UserAgentDetector) IsVPN(userAgent string) bool {
	return d.vpnMatcher.Contains(userAgent)
}

// IsBot checks if the user agent indicates a bot.
func (d *UserAgentDetector) IsBot(userAgent string) bool {
	return d.botMatcher.Contains(userAgent)
}

// IsCrawler checks if the user agent indicates a crawler.
func (d *UserAgentDetector) IsCrawler(userAgent string) bool {
	return d.crawlerMatcher.Contains(userAgent)
}
