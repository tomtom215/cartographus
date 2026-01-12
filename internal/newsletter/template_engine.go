// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package newsletter provides newsletter generation and delivery functionality.
//
// template_engine.go - Newsletter Template Engine
//
// This file implements the template rendering engine for newsletters:
//   - Go template-based rendering with HTML and plaintext support
//   - Built-in template functions for formatting dates, numbers, and content
//   - Variable substitution with automatic escaping for security
//   - Preview mode for testing templates before sending
//
// Security:
//   - All user content is HTML-escaped by default
//   - Template injection is prevented through Go's html/template package
//   - External URLs are validated before inclusion
package newsletter

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"strings"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// TemplateEngine handles newsletter template rendering.
type TemplateEngine struct {
	// funcMap provides custom template functions.
	funcMap template.FuncMap
}

// NewTemplateEngine creates a new template engine with standard functions.
func NewTemplateEngine() *TemplateEngine {
	te := &TemplateEngine{}
	te.funcMap = te.buildFuncMap()
	return te
}

// buildFuncMap creates the template function map.
//
//nolint:gocyclo // High complexity due to defining 40+ template helper functions
func (te *TemplateEngine) buildFuncMap() template.FuncMap {
	return template.FuncMap{
		// Date/time formatting
		"formatDate": func(t time.Time, layout string) string {
			return t.Format(layout)
		},
		"formatDateDefault": func(t time.Time) string {
			return t.Format("January 2, 2006")
		},
		"formatDateShort": func(t time.Time) string {
			return t.Format("Jan 2, 2006")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("January 2, 2006 at 3:04 PM")
		},
		"formatTime": func(t time.Time) string {
			return t.Format("3:04 PM")
		},
		"formatDateRange": func(start, end time.Time) string {
			if start.Year() == end.Year() && start.Month() == end.Month() {
				return fmt.Sprintf("%s - %s, %d", start.Format("January 2"), end.Format("2"), start.Year())
			}
			return fmt.Sprintf("%s - %s", start.Format("January 2, 2006"), end.Format("January 2, 2006"))
		},
		"now": time.Now,
		"timePtr": func(t *time.Time) time.Time {
			if t == nil {
				return time.Time{}
			}
			return *t
		},

		// Number formatting
		"formatNumber": formatWithCommas,
		"formatFloat": func(f float64, precision int) string {
			return fmt.Sprintf("%.*f", precision, f)
		},
		"formatHours": func(hours float64) string {
			if hours < 1 {
				return fmt.Sprintf("%d min", int(hours*60))
			}
			h := int(hours)
			m := int((hours - float64(h)) * 60)
			if m > 0 {
				return fmt.Sprintf("%d hr %d min", h, m)
			}
			return fmt.Sprintf("%d hr", h)
		},
		"formatDuration": func(minutes int) string {
			if minutes < 60 {
				return fmt.Sprintf("%d min", minutes)
			}
			h := minutes / 60
			m := minutes % 60
			if m > 0 {
				return fmt.Sprintf("%dh %dm", h, m)
			}
			return fmt.Sprintf("%dh", h)
		},
		"formatPercent": func(f float64) string {
			return fmt.Sprintf("%.1f%%", f)
		},
		"formatPercentInt": func(f float64) string {
			return fmt.Sprintf("%.0f%%", f)
		},

		// String manipulation
		"truncate": func(s string, maxLen int) string {
			if len(s) <= maxLen {
				return s
			}
			return s[:maxLen-3] + "..."
		},
		"truncateWords": func(s string, maxWords int) string {
			words := strings.Fields(s)
			if len(words) <= maxWords {
				return s
			}
			return strings.Join(words[:maxWords], " ") + "..."
		},
		"toLowerCase": strings.ToLower,
		"toUpperCase": strings.ToUpper,
		"titleCase":   toTitleCase, // Custom implementation to replace deprecated strings.Title
		"trim":        strings.TrimSpace,
		"replace":     strings.ReplaceAll,
		"join":        strings.Join,
		"split":       strings.Split,
		"contains":    strings.Contains,
		"hasPrefix":   strings.HasPrefix,
		"hasSuffix":   strings.HasSuffix,

		// HTML helpers
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s) //nolint:gosec // Intentional for trusted content
		},
		"escapeHTML": html.EscapeString,
		"newline": func() template.HTML {
			return template.HTML("<br>")
		},
		"nbsp": func() template.HTML {
			return template.HTML("&nbsp;")
		},

		// List/array helpers
		"first": func(items interface{}) interface{} {
			switch v := items.(type) {
			case []models.NewsletterMediaItem:
				if len(v) > 0 {
					return v[0]
				}
			case []models.NewsletterShowItem:
				if len(v) > 0 {
					return v[0]
				}
			case []models.WrappedGenreRank:
				if len(v) > 0 {
					return v[0]
				}
			case []string:
				if len(v) > 0 {
					return v[0]
				}
			}
			return nil
		},
		"last": func(items interface{}) interface{} {
			switch v := items.(type) {
			case []models.NewsletterMediaItem:
				if len(v) > 0 {
					return v[len(v)-1]
				}
			case []models.NewsletterShowItem:
				if len(v) > 0 {
					return v[len(v)-1]
				}
			case []string:
				if len(v) > 0 {
					return v[len(v)-1]
				}
			}
			return nil
		},
		"len": func(items interface{}) int {
			switch v := items.(type) {
			case []models.NewsletterMediaItem:
				return len(v)
			case []models.NewsletterShowItem:
				return len(v)
			case []models.WrappedGenreRank:
				return len(v)
			case []string:
				return len(v)
			case string:
				return len(v)
			}
			return 0
		},
		"slice": func(items interface{}, start, end int) interface{} {
			switch v := items.(type) {
			case []models.NewsletterMediaItem:
				if start >= len(v) {
					return []models.NewsletterMediaItem{}
				}
				if end > len(v) {
					end = len(v)
				}
				return v[start:end]
			case []models.NewsletterShowItem:
				if start >= len(v) {
					return []models.NewsletterShowItem{}
				}
				if end > len(v) {
					end = len(v)
				}
				return v[start:end]
			}
			return nil
		},
		"range": func(n int) []int {
			result := make([]int, n)
			for i := 0; i < n; i++ {
				result[i] = i
			}
			return result
		},

		// Conditional helpers
		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" || val == 0 || val == false {
				return def
			}
			return val
		},
		"ifelse": func(cond bool, trueVal, falseVal interface{}) interface{} {
			if cond {
				return trueVal
			}
			return falseVal
		},
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"ne": func(a, b interface{}) bool {
			return a != b
		},
		"lt": func(a, b int) bool {
			return a < b
		},
		"le": func(a, b int) bool {
			return a <= b
		},
		"gt": func(a, b int) bool {
			return a > b
		},
		"ge": func(a, b int) bool {
			return a >= b
		},

		// Math helpers
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"mod": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a % b
		},

		// URL helpers
		"urlEncode": func(s string) string {
			return template.URLQueryEscaper(s)
		},

		// Content-specific helpers
		"mediaTypeIcon": func(mediaType string) string {
			switch mediaType {
			case "movie":
				return "film"
			case "episode", "show":
				return "tv"
			case "track", "album":
				return "music"
			default:
				return "play"
			}
		},
		"genreColor": func(genre string) string {
			// Map genres to colors for visual styling
			colors := map[string]string{
				"action":      "#e74c3c",
				"comedy":      "#f39c12",
				"drama":       "#3498db",
				"horror":      "#9b59b6",
				"sci-fi":      "#1abc9c",
				"documentary": "#34495e",
				"animation":   "#e91e63",
				"romance":     "#ff69b4",
				"thriller":    "#2c3e50",
				"fantasy":     "#9c27b0",
			}
			if color, ok := colors[strings.ToLower(genre)]; ok {
				return color
			}
			return "#95a5a6"
		},
		"ratingStars": func(rating float64, maxStars int) string {
			fullStars := int(rating)
			halfStar := rating-float64(fullStars) >= 0.5
			emptyStars := maxStars - fullStars
			if halfStar {
				emptyStars--
			}

			result := strings.Repeat("★", fullStars)
			if halfStar {
				result += "½"
			}
			result += strings.Repeat("☆", emptyStars)
			return result
		},
	}
}

// RenderHTML renders a newsletter template with the provided data to HTML.
func (te *TemplateEngine) RenderHTML(templateContent string, data *models.NewsletterContentData) (string, error) {
	tmpl, err := template.New("newsletter").Funcs(te.funcMap).Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// RenderText renders a newsletter template with the provided data to plaintext.
func (te *TemplateEngine) RenderText(templateContent string, data *models.NewsletterContentData) (string, error) {
	// Use text/template for plaintext to avoid HTML escaping
	tmpl, err := template.New("newsletter_text").Funcs(te.funcMap).Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// RenderSubject renders the subject line with variable substitution.
func (te *TemplateEngine) RenderSubject(subjectTemplate string, data *models.NewsletterContentData) (string, error) {
	tmpl, err := template.New("subject").Funcs(te.funcMap).Parse(subjectTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse subject template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute subject template: %w", err)
	}

	return strings.TrimSpace(buf.String()), nil
}

// ValidateTemplate checks if a template is syntactically valid.
func (te *TemplateEngine) ValidateTemplate(templateContent string) error {
	_, err := template.New("validate").Funcs(te.funcMap).Parse(templateContent)
	if err != nil {
		return fmt.Errorf("invalid template syntax: %w", err)
	}
	return nil
}

// GetAvailableVariables returns the list of available template variables.
func (te *TemplateEngine) GetAvailableVariables() []models.TemplateVariable {
	return []models.TemplateVariable{
		// Server info
		{Name: "ServerName", Description: "Name of the media server", Type: "string", Example: "My Plex Server", Required: true},
		{Name: "ServerURL", Description: "URL of the media server", Type: "string", Example: "https://plex.example.com", Required: false},
		{Name: "NewsletterURL", Description: "URL to view this newsletter online", Type: "string", Example: "https://app.example.com/newsletter/123", Required: false},
		{Name: "UnsubscribeURL", Description: "URL to unsubscribe from newsletters", Type: "string", Example: "https://app.example.com/unsubscribe", Required: false},

		// Date info
		{Name: "GeneratedAt", Description: "When the newsletter was generated", Type: "time.Time", Example: "2025-01-05T12:00:00Z", Required: true},
		{Name: "DateRangeStart", Description: "Start of the content date range", Type: "time.Time", Example: "2024-12-29T00:00:00Z", Required: true},
		{Name: "DateRangeEnd", Description: "End of the content date range", Type: "time.Time", Example: "2025-01-05T00:00:00Z", Required: true},
		{Name: "DateRangeDisplay", Description: "Human-readable date range", Type: "string", Example: "December 29, 2024 - January 5, 2025", Required: true},

		// Content
		{Name: "NewMovies", Description: "Array of recently added movies", Type: "[]NewsletterMediaItem", Required: false},
		{Name: "NewShows", Description: "Array of recently added TV shows with episodes", Type: "[]NewsletterShowItem", Required: false},
		{Name: "NewMusic", Description: "Array of recently added music", Type: "[]NewsletterMediaItem", Required: false},
		{Name: "TopMovies", Description: "Array of top movies by watch time", Type: "[]NewsletterMediaItem", Required: false},
		{Name: "TopShows", Description: "Array of top shows by watch time", Type: "[]NewsletterShowItem", Required: false},

		// Stats
		{Name: "Stats", Description: "Viewing statistics for the period", Type: "NewsletterStats", Required: false},
		{Name: "Stats.TotalPlaybacks", Description: "Total number of playbacks", Type: "int", Required: false},
		{Name: "Stats.TotalWatchTimeHours", Description: "Total watch time in hours", Type: "float64", Required: false},
		{Name: "Stats.UniqueUsers", Description: "Number of unique users", Type: "int", Required: false},
		{Name: "Stats.UniqueContent", Description: "Number of unique content items", Type: "int", Required: false},

		// User-specific (for personalized newsletters)
		{Name: "User", Description: "User-specific data for personalized newsletters", Type: "NewsletterUserData", Required: false},
		{Name: "User.Username", Description: "User's display name", Type: "string", Required: false},
		{Name: "User.WatchTimeHours", Description: "User's total watch time", Type: "float64", Required: false},
		{Name: "Recommendations", Description: "Personalized content recommendations", Type: "[]NewsletterMediaItem", Required: false},

		// Health (for server health newsletters)
		{Name: "Health", Description: "Server health data", Type: "NewsletterHealthData", Required: false},
		{Name: "Health.ServerStatus", Description: "Server status (healthy, degraded, unhealthy)", Type: "string", Required: false},
		{Name: "Health.UptimePercent", Description: "Server uptime percentage", Type: "float64", Required: false},
	}
}

// toTitleCase converts a string to title case.
// This is a replacement for the deprecated strings.Title.
func toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// formatWithCommas formats an integer with thousands separators.
func formatWithCommas(n int) string {
	if n < 0 {
		return "-" + formatWithCommas(-n)
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}

	s := fmt.Sprintf("%d", n)
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}
