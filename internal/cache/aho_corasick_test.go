// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package cache

import (
	"strings"
	"sync"
	"testing"
)

func TestAhoCorasick_BasicOperations(t *testing.T) {
	t.Parallel()

	ac := NewAhoCorasick()
	ac.AddPattern("he", nil)
	ac.AddPattern("she", nil)
	ac.AddPattern("his", nil)
	ac.AddPattern("hers", nil)
	ac.Build()

	text := "ushers"
	matches := ac.Search(text)

	// Should find: "she" at 1, "he" at 2, "hers" at 2
	if len(matches) < 3 {
		t.Errorf("Expected at least 3 matches, got %d", len(matches))
	}

	// Verify specific matches
	foundShe := false
	foundHe := false
	foundHers := false

	for _, m := range matches {
		switch m.Pattern {
		case "she":
			foundShe = true
		case "he":
			foundHe = true
		case "hers":
			foundHers = true
		}
	}

	if !foundShe {
		t.Error("Expected to find 'she'")
	}
	if !foundHe {
		t.Error("Expected to find 'he'")
	}
	if !foundHers {
		t.Error("Expected to find 'hers'")
	}
}

func TestAhoCorasick_CaseInsensitive(t *testing.T) {
	t.Parallel()

	ac := NewAhoCorasick() // Default is case-insensitive
	ac.AddPattern("hello", nil)
	ac.AddPattern("world", nil)
	ac.Build()

	// All variations should match
	tests := []string{
		"hello world",
		"HELLO WORLD",
		"Hello World",
		"hElLo WoRlD",
	}

	for _, text := range tests {
		matches := ac.Search(text)
		if len(matches) != 2 {
			t.Errorf("Search(%q) = %d matches, want 2", text, len(matches))
		}
	}
}

func TestAhoCorasick_CaseSensitive(t *testing.T) {
	t.Parallel()

	ac := NewAhoCorasickCaseSensitive()
	ac.AddPattern("Hello", nil)
	ac.Build()

	if !ac.Contains("Hello World") {
		t.Error("Should find 'Hello' in 'Hello World'")
	}

	if ac.Contains("hello world") {
		t.Error("Should NOT find 'Hello' in 'hello world' (case-sensitive)")
	}
}

func TestAhoCorasick_SearchFirst(t *testing.T) {
	t.Parallel()

	ac := NewAhoCorasick()
	ac.AddPattern("first", "1")
	ac.AddPattern("second", "2")
	ac.AddPattern("third", "3")
	ac.Build()

	text := "The first thing, then second and third"

	match, found := ac.SearchFirst(text)
	if !found {
		t.Error("SearchFirst should find a match")
	}

	if match.Pattern != "first" {
		t.Errorf("SearchFirst pattern = %q, want 'first'", match.Pattern)
	}

	if match.Data != "1" {
		t.Errorf("SearchFirst data = %v, want '1'", match.Data)
	}
}

func TestAhoCorasick_Contains(t *testing.T) {
	t.Parallel()

	ac := NewAhoCorasick()
	ac.AddPattern("pattern1", nil)
	ac.AddPattern("pattern2", nil)
	ac.Build()

	if !ac.Contains("text with pattern1 inside") {
		t.Error("Contains should return true")
	}

	if ac.Contains("text without any patterns") {
		t.Error("Contains should return false")
	}
}

func TestAhoCorasick_MatchCount(t *testing.T) {
	t.Parallel()

	ac := NewAhoCorasick()
	ac.AddPattern("a", nil)
	ac.Build()

	text := "abracadabra"
	count := ac.MatchCount(text)

	if count != 5 {
		t.Errorf("MatchCount = %d, want 5", count)
	}
}

func TestAhoCorasick_EmptyPattern(t *testing.T) {
	t.Parallel()

	ac := NewAhoCorasick()
	ac.AddPattern("", nil) // Should be ignored
	ac.AddPattern("valid", nil)
	ac.Build()

	if ac.PatternCount() != 1 {
		t.Errorf("PatternCount = %d, want 1", ac.PatternCount())
	}
}

func TestAhoCorasick_NoPatterns(t *testing.T) {
	t.Parallel()

	ac := NewAhoCorasick()
	ac.Build()

	matches := ac.Search("any text")
	if len(matches) != 0 {
		t.Errorf("Search with no patterns should return empty, got %d", len(matches))
	}

	if ac.Contains("any text") {
		t.Error("Contains with no patterns should return false")
	}
}

func TestAhoCorasick_NotBuilt(t *testing.T) {
	t.Parallel()

	ac := NewAhoCorasick()
	ac.AddPattern("test", nil)
	// Don't call Build()

	matches := ac.Search("test string")
	if len(matches) != 0 {
		t.Errorf("Search without Build should return empty, got %d", len(matches))
	}
}

func TestAhoCorasick_Rebuild(t *testing.T) {
	t.Parallel()

	ac := NewAhoCorasick()
	ac.AddPattern("first", nil)
	ac.Build()

	// Add more patterns after build
	ac.AddPattern("second", nil)
	ac.Build() // Rebuild

	if ac.PatternCount() != 2 {
		t.Errorf("PatternCount after rebuild = %d, want 2", ac.PatternCount())
	}

	if !ac.Contains("first and second") {
		t.Error("Should find both patterns after rebuild")
	}
}

func TestAhoCorasick_Clear(t *testing.T) {
	t.Parallel()

	ac := NewAhoCorasick()
	ac.AddPattern("test", nil)
	ac.Build()

	ac.Clear()

	if ac.PatternCount() != 0 {
		t.Errorf("PatternCount after Clear = %d, want 0", ac.PatternCount())
	}

	if ac.Contains("test") {
		t.Error("Contains should return false after Clear")
	}
}

func TestAhoCorasick_OverlappingPatterns(t *testing.T) {
	t.Parallel()

	ac := NewAhoCorasick()
	ac.AddPattern("ab", nil)
	ac.AddPattern("abc", nil)
	ac.AddPattern("bc", nil)
	ac.Build()

	matches := ac.Search("abc")

	// Should find all overlapping patterns
	if len(matches) != 3 {
		t.Errorf("Expected 3 matches for overlapping patterns, got %d", len(matches))
	}
}

func TestAhoCorasick_WithData(t *testing.T) {
	t.Parallel()

	type Category struct {
		Name     string
		Severity int
	}

	ac := NewAhoCorasick()
	ac.AddPattern("error", Category{Name: "Error", Severity: 3})
	ac.AddPattern("warning", Category{Name: "Warning", Severity: 2})
	ac.AddPattern("info", Category{Name: "Info", Severity: 1})
	ac.Build()

	matches := ac.Search("error: something went wrong, warning: check this")

	for _, m := range matches {
		cat, ok := m.Data.(Category)
		if !ok {
			t.Error("Data should be Category type")
			continue
		}

		if m.Pattern == "error" && cat.Severity != 3 {
			t.Errorf("error severity = %d, want 3", cat.Severity)
		}
		if m.Pattern == "warning" && cat.Severity != 2 {
			t.Errorf("warning severity = %d, want 2", cat.Severity)
		}
	}
}

func TestAhoCorasick_Concurrent(t *testing.T) {
	t.Parallel()

	ac := NewAhoCorasick()
	ac.AddPattern("pattern1", nil)
	ac.AddPattern("pattern2", nil)
	ac.AddPattern("pattern3", nil)
	ac.Build()

	var wg sync.WaitGroup
	numGoroutines := 50
	numOps := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				text := "text with pattern1 and pattern2 inside"
				ac.Search(text)
				ac.Contains(text)
				ac.SearchFirst(text)
			}
		}(i)
	}

	wg.Wait()
}

func TestPatternMatcher_BasicOperations(t *testing.T) {
	t.Parallel()

	patterns := map[string]any{
		"vpn":   "vpn_detected",
		"proxy": "proxy_detected",
	}

	pm := NewPatternMatcher(patterns)

	matches := pm.Match("Using VPN client with proxy")
	if len(matches) != 2 {
		t.Errorf("Match returned %d results, want 2", len(matches))
	}

	if !pm.Contains("vpn client") {
		t.Error("Contains should return true for 'vpn client'")
	}

	if pm.Contains("normal browser") {
		t.Error("Contains should return false for 'normal browser'")
	}
}

func TestPatternMatcherFromSlice(t *testing.T) {
	t.Parallel()

	patterns := []string{"bot", "crawler", "spider"}
	pm := NewPatternMatcherFromSlice(patterns, "automated")

	matches := pm.Match("googlebot crawler")
	if len(matches) != 2 {
		t.Errorf("Match returned %d results, want 2", len(matches))
	}

	for _, m := range matches {
		if m.Data != "automated" {
			t.Errorf("Match data = %v, want 'automated'", m.Data)
		}
	}
}

func TestUserAgentDetector_VPN(t *testing.T) {
	t.Parallel()

	detector := NewUserAgentDetector()

	vpnUserAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) ExpressVPN",
		"OpenVPN Client 2.5.0",
		"WireGuard/1.0",
		"Mozilla/5.0 NordVPN Proxy",
	}

	for _, ua := range vpnUserAgents {
		if !detector.IsVPN(ua) {
			t.Errorf("IsVPN(%q) = false, want true", ua)
		}
	}

	normalUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	if detector.IsVPN(normalUA) {
		t.Errorf("IsVPN(%q) = true, want false", normalUA)
	}
}

func TestUserAgentDetector_Bot(t *testing.T) {
	t.Parallel()

	detector := NewUserAgentDetector()

	botUserAgents := []string{
		"python-requests/2.25.1",
		"curl/7.68.0",
		"Go-http-client/1.1",
		"axios/0.21.1",
		"HeadlessChrome/91.0.4472.124",
	}

	for _, ua := range botUserAgents {
		if !detector.IsBot(ua) {
			t.Errorf("IsBot(%q) = false, want true", ua)
		}
	}
}

func TestUserAgentDetector_Crawler(t *testing.T) {
	t.Parallel()

	detector := NewUserAgentDetector()

	crawlerUserAgents := []string{
		"Googlebot/2.1 (+http://www.google.com/bot.html)",
		"Bingbot/2.0",
		"Mozilla/5.0 (compatible; AhrefsBot/7.0)",
		"facebookexternalhit/1.1",
	}

	for _, ua := range crawlerUserAgents {
		if !detector.IsCrawler(ua) {
			t.Errorf("IsCrawler(%q) = false, want true", ua)
		}
	}
}

func TestUserAgentDetector_Detect(t *testing.T) {
	t.Parallel()

	detector := NewUserAgentDetector()

	// Test combined detection
	ua := "Googlebot with VPN proxy"
	result := detector.Detect(ua)

	if !result.IsVPN {
		t.Error("Should detect VPN")
	}
	if !result.IsCrawler {
		t.Error("Should detect Crawler")
	}
	if len(result.Matches) < 2 {
		t.Errorf("Expected at least 2 matches, got %d", len(result.Matches))
	}
}

func TestUserAgentDetector_Normal(t *testing.T) {
	t.Parallel()

	detector := NewUserAgentDetector()

	normalUA := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
	result := detector.Detect(normalUA)

	if result.IsVPN || result.IsBot || result.IsCrawler {
		t.Error("Normal browser should not be detected as VPN, bot, or crawler")
	}
}

// Benchmark tests

func BenchmarkAhoCorasick_Build(b *testing.B) {
	patterns := make([]string, 100)
	for i := 0; i < 100; i++ {
		patterns[i] = "pattern" + string(rune(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ac := NewAhoCorasick()
		for _, p := range patterns {
			ac.AddPattern(p, nil)
		}
		ac.Build()
	}
}

func BenchmarkAhoCorasick_Search(b *testing.B) {
	ac := NewAhoCorasick()
	for i := 0; i < 100; i++ {
		ac.AddPattern("pattern"+string(rune(i%26+'a')), nil)
	}
	ac.Build()

	text := strings.Repeat("This is a test text with patterna and patternb inside. ", 10)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ac.Search(text)
	}
}

func BenchmarkAhoCorasick_Contains(b *testing.B) {
	ac := NewAhoCorasick()
	for i := 0; i < 100; i++ {
		ac.AddPattern("pattern"+string(rune(i%26+'a')), nil)
	}
	ac.Build()

	text := "This is a test text with patternz at the end"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ac.Contains(text)
	}
}

func BenchmarkUserAgentDetector_Detect(b *testing.B) {
	detector := NewUserAgentDetector()
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"Googlebot/2.1",
		"python-requests/2.25.1",
		"Mozilla/5.0 NordVPN",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(userAgents[i%len(userAgents)])
	}
}

// Compare with naive approach
func BenchmarkNaivePatternMatch(b *testing.B) {
	patterns := make([]string, 100)
	for i := 0; i < 100; i++ {
		patterns[i] = "pattern" + string(rune(i%26+'a'))
	}

	text := strings.Repeat("This is a test text with patterna and patternb inside. ", 10)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, p := range patterns {
			strings.Contains(strings.ToLower(text), strings.ToLower(p))
		}
	}
}
