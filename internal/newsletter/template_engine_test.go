// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package newsletter

import (
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

func TestNewTemplateEngine(t *testing.T) {
	engine := NewTemplateEngine()
	if engine == nil {
		t.Fatal("NewTemplateEngine returned nil")
	}
}

func TestTemplateEngine_RenderSubject(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		data     *models.NewsletterContentData
		want     string
		wantErr  bool
	}{
		{
			name:     "simple subject",
			template: "New content on {{.ServerName}}",
			data: &models.NewsletterContentData{
				ServerName: "Media Server",
			},
			want:    "New content on Media Server",
			wantErr: false,
		},
		{
			name:     "subject with date",
			template: "Weekly Digest - {{formatDateDefault .GeneratedAt}}",
			data: &models.NewsletterContentData{
				GeneratedAt: time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC),
			},
			want:    "Weekly Digest - January 5, 2026",
			wantErr: false,
		},
		{
			name:     "subject with movie count",
			template: "{{len .NewMovies}} new movies this week",
			data: &models.NewsletterContentData{
				NewMovies: []models.NewsletterMediaItem{
					{Title: "Movie 1"},
					{Title: "Movie 2"},
				},
			},
			want:    "2 new movies this week",
			wantErr: false,
		},
		{
			name:     "invalid template syntax",
			template: "{{.Invalid",
			data:     &models.NewsletterContentData{},
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.RenderSubject(tt.template, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderSubject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("RenderSubject() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTemplateEngine_RenderHTML(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		data     *models.NewsletterContentData
		contains string
		wantErr  bool
	}{
		{
			name:     "basic html",
			template: "<h1>{{.ServerName}}</h1>",
			data: &models.NewsletterContentData{
				ServerName: "Media Server",
			},
			contains: "<h1>Media Server</h1>",
			wantErr:  false,
		},
		{
			name:     "html with movie list",
			template: `<ul>{{range .NewMovies}}<li>{{.Title}}</li>{{end}}</ul>`,
			data: &models.NewsletterContentData{
				NewMovies: []models.NewsletterMediaItem{
					{Title: "The Matrix"},
					{Title: "Inception"},
				},
			},
			contains: "<li>The Matrix</li>",
			wantErr:  false,
		},
		{
			name:     "html with safe html function",
			template: `{{safeHTML "<b>Bold</b>"}}`,
			data:     &models.NewsletterContentData{},
			contains: "<b>Bold</b>",
			wantErr:  false,
		},
		{
			name:     "html with truncate function",
			template: `{{truncate "This is a very long string" 10}}`,
			data:     &models.NewsletterContentData{},
			contains: "This is...",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.RenderHTML(tt.template, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderHTML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !containsString(got, tt.contains) {
				t.Errorf("RenderHTML() = %q, want to contain %q", got, tt.contains)
			}
		})
	}
}

func TestTemplateEngine_RenderText(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		data     *models.NewsletterContentData
		contains string
		wantErr  bool
	}{
		{
			name:     "basic text",
			template: "Server: {{.ServerName}}",
			data: &models.NewsletterContentData{
				ServerName: "Media Server",
			},
			contains: "Server: Media Server",
			wantErr:  false,
		},
		{
			name:     "text with stats",
			template: `Total Playbacks: {{formatNumber .Stats.TotalPlaybacks}}`,
			data: &models.NewsletterContentData{
				Stats: &models.NewsletterStats{
					TotalPlaybacks: 1234,
				},
			},
			contains: "1,234",
			wantErr:  false,
		},
		{
			name:     "text with hours formatting",
			template: `Watch Time: {{formatHours .Stats.TotalWatchTimeHours}}`,
			data: &models.NewsletterContentData{
				Stats: &models.NewsletterStats{
					TotalWatchTimeHours: 125.5,
				},
			},
			contains: "125 hr 30 min",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.RenderText(tt.template, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !containsString(got, tt.contains) {
				t.Errorf("RenderText() = %q, want to contain %q", got, tt.contains)
			}
		})
	}
}

func TestTemplateEngine_ValidateTemplate(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{
			name:     "valid simple template",
			template: "Hello {{.ServerName}}",
			wantErr:  false,
		},
		{
			name:     "valid template with range",
			template: "{{range .NewMovies}}{{.Title}}{{end}}",
			wantErr:  false,
		},
		{
			name:     "valid template with if",
			template: "{{if .Stats}}Stats available{{end}}",
			wantErr:  false,
		},
		{
			name:     "valid template with function",
			template: "{{formatNumber 1234}}",
			wantErr:  false,
		},
		{
			name:     "invalid unclosed action",
			template: "{{.ServerName",
			wantErr:  true,
		},
		{
			name:     "invalid unclosed range",
			template: "{{range .NewMovies}}{{.Title}}",
			wantErr:  true,
		},
		{
			name:     "invalid unknown function",
			template: "{{unknownFunc .ServerName}}",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateTemplate(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTemplateFunctions(t *testing.T) {
	engine := NewTemplateEngine()

	t.Run("formatDate", func(t *testing.T) {
		data := &models.NewsletterContentData{
			GeneratedAt: time.Date(2026, 1, 5, 15, 30, 0, 0, time.UTC),
		}
		got, err := engine.RenderSubject(`{{formatDate .GeneratedAt "2006-01-02"}}`, data)
		if err != nil {
			t.Fatalf("formatDate failed: %v", err)
		}
		if got != "2026-01-05" {
			t.Errorf("formatDate = %q, want %q", got, "2026-01-05")
		}
	})

	t.Run("formatDuration", func(t *testing.T) {
		got, err := engine.RenderSubject(`{{formatDuration 135}}`, &models.NewsletterContentData{})
		if err != nil {
			t.Fatalf("formatDuration failed: %v", err)
		}
		if got != "2h 15m" {
			t.Errorf("formatDuration = %q, want %q", got, "2h 15m")
		}
	})

	t.Run("formatPercent", func(t *testing.T) {
		got, err := engine.RenderSubject(`{{formatPercent 75.5}}`, &models.NewsletterContentData{})
		if err != nil {
			t.Fatalf("formatPercent failed: %v", err)
		}
		if got != "75.5%" {
			t.Errorf("formatPercent = %q, want %q", got, "75.5%")
		}
	})

	t.Run("truncateWords", func(t *testing.T) {
		got, err := engine.RenderSubject(`{{truncateWords "one two three four five" 3}}`, &models.NewsletterContentData{})
		if err != nil {
			t.Fatalf("truncateWords failed: %v", err)
		}
		if got != "one two three..." {
			t.Errorf("truncateWords = %q, want %q", got, "one two three...")
		}
	})

	t.Run("toLowerCase", func(t *testing.T) {
		got, err := engine.RenderSubject(`{{toLowerCase "HELLO"}}`, &models.NewsletterContentData{})
		if err != nil {
			t.Fatalf("toLowerCase failed: %v", err)
		}
		if got != "hello" {
			t.Errorf("toLowerCase = %q, want %q", got, "hello")
		}
	})

	t.Run("toUpperCase", func(t *testing.T) {
		got, err := engine.RenderSubject(`{{toUpperCase "hello"}}`, &models.NewsletterContentData{})
		if err != nil {
			t.Fatalf("toUpperCase failed: %v", err)
		}
		if got != "HELLO" {
			t.Errorf("toUpperCase = %q, want %q", got, "HELLO")
		}
	})

	t.Run("ratingStars", func(t *testing.T) {
		got, err := engine.RenderSubject(`{{ratingStars 4 5}}`, &models.NewsletterContentData{})
		if err != nil {
			t.Fatalf("ratingStars failed: %v", err)
		}
		expected := "★★★★☆"
		if got != expected {
			t.Errorf("ratingStars = %q, want %q", got, expected)
		}
	})

	t.Run("mediaTypeIcon", func(t *testing.T) {
		got, err := engine.RenderSubject(`{{mediaTypeIcon "movie"}}`, &models.NewsletterContentData{})
		if err != nil {
			t.Fatalf("mediaTypeIcon failed: %v", err)
		}
		if got == "" {
			t.Error("mediaTypeIcon returned empty string")
		}
	})

	t.Run("add", func(t *testing.T) {
		got, err := engine.RenderSubject(`{{add 5 3}}`, &models.NewsletterContentData{})
		if err != nil {
			t.Fatalf("add failed: %v", err)
		}
		if got != "8" {
			t.Errorf("add = %q, want %q", got, "8")
		}
	})

	t.Run("mul", func(t *testing.T) {
		got, err := engine.RenderSubject(`{{mul 5 3}}`, &models.NewsletterContentData{})
		if err != nil {
			t.Fatalf("mul failed: %v", err)
		}
		if got != "15" {
			t.Errorf("mul = %q, want %q", got, "15")
		}
	})
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
