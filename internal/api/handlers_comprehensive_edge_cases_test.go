// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/goccy/go-json"
	ws "github.com/tomtom215/cartographus/internal/websocket"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/models"
)

// TestNewHandler_EdgeCases tests handler constructor edge cases
func TestNewHandler_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("all nil dependencies", func(t *testing.T) {
		cfg := &config.Config{API: config.APIConfig{DefaultPageSize: 100, MaxPageSize: 1000}}
		handler := NewHandler(nil, nil, nil, cfg, nil, nil)

		if handler == nil {
			t.Fatal("NewHandler should not return nil")
		}
		if handler.cache == nil || handler.perfMon == nil {
			t.Error("Cache and perfMon should always be initialized")
		}
	})

	t.Run("partial plex oauth config", func(t *testing.T) {
		tests := []struct {
			name           string
			clientID       string
			redirectURI    string
			expectOAuthNil bool
		}{
			{"empty client ID", "", "http://localhost/callback", true},
			{"empty redirect URI", "client-123", "", true},
			{"both empty", "", "", true},
			{"both present", "client-123", "http://localhost/callback", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := &config.Config{Plex: config.PlexConfig{OAuthClientID: tt.clientID, OAuthRedirectURI: tt.redirectURI}}
				handler := NewHandler(nil, nil, nil, cfg, nil, nil)

				if tt.expectOAuthNil && handler.plexOAuthClient != nil {
					t.Error("Expected OAuth client to be nil")
				}
				if !tt.expectOAuthNil && handler.plexOAuthClient == nil {
					t.Error("Expected OAuth client to be initialized")
				}
			})
		}
	})
}

// TestSetBackupManager_EdgeCases tests backup manager setter edge cases
func TestSetBackupManager_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("nil manager", func(t *testing.T) {
		handler := &Handler{backupManager: &mockBackupManager{}}
		handler.SetBackupManager(nil)
		if handler.backupManager != nil {
			t.Error("BackupManager should be nil")
		}
	})

	t.Run("set valid manager", func(t *testing.T) {
		handler := &Handler{}
		mock := &mockBackupManager{}
		handler.SetBackupManager(mock)
		if handler.backupManager == nil {
			t.Error("BackupManager should be set")
		}
	})

	t.Run("replace existing manager", func(t *testing.T) {
		handler := &Handler{backupManager: &mockBackupManager{}}
		newMock := &mockBackupManager{}
		handler.SetBackupManager(newMock)
		if handler.backupManager != newMock {
			t.Error("BackupManager should be replaced")
		}
	})
}

// TestClearCache_EdgeCases tests cache clearing edge cases
func TestClearCache_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("nil cache", func(t *testing.T) {
		handler := &Handler{cache: nil}
		handler.ClearCache() // Should not panic
	})

	t.Run("concurrent access", func(t *testing.T) {
		c := cache.New(5 * time.Minute)
		handler := &Handler{cache: c}
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(2)
			go func(n int) {
				defer wg.Done()
				c.Set("key"+string(rune('0'+n%10)), "value")
			}(i)
			go func() {
				defer wg.Done()
				handler.ClearCache()
			}()
		}
		wg.Wait() // Should not panic or deadlock
	})
}

// TestOnSyncCompleted_EdgeCases tests sync completion edge cases
func TestOnSyncCompleted_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("nil db", func(t *testing.T) {
		c := cache.New(5 * time.Minute)
		c.Set("test", "value")
		wsHub := ws.NewHub()
		go wsHub.RunWithContext(context.Background())
		handler := &Handler{cache: c, wsHub: wsHub, db: nil}

		handler.OnSyncCompleted(10, 100)
		if _, found := c.Get("test"); found {
			t.Error("Cache should be cleared")
		}
	})

	t.Run("concurrent calls", func(t *testing.T) {
		c := cache.New(5 * time.Minute)
		wsHub := ws.NewHub()
		go wsHub.RunWithContext(context.Background())
		handler := &Handler{cache: c, wsHub: wsHub}

		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				handler.OnSyncCompleted(n, int64(n*100))
			}(i)
		}
		wg.Wait() // Should not panic or deadlock
	})
}

// TestCheckWebSocketOrigin_EdgeCases tests WebSocket origin validation edge cases
func TestCheckWebSocketOrigin_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      *config.Config
		origin      string
		shouldAllow bool
	}{
		{"nil config", nil, "http://example.com", true},
		{"empty origin header - SECURITY: must reject", &config.Config{Security: config.SecurityConfig{CORSOrigins: []string{}}}, "", false},
		{"unicode origin", &config.Config{Security: config.SecurityConfig{CORSOrigins: []string{"http://例え.jp"}}}, "http://例え.jp", true},
		{"punycode origin", &config.Config{Security: config.SecurityConfig{CORSOrigins: []string{"http://xn--r8jz45g.jp"}}}, "http://xn--r8jz45g.jp", true},
		{"origin with path", &config.Config{Security: config.SecurityConfig{CORSOrigins: []string{"http://localhost:3857/path"}}}, "http://localhost:3857/path", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{config: tt.config}
			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if result := handler.checkWebSocketOrigin(req); result != tt.shouldAllow {
				t.Errorf("Expected %v for origin %q", tt.shouldAllow, tt.origin)
			}
		})
	}
}

// TestRespondJSON_EdgeCases tests JSON response edge cases
func TestRespondJSON_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("large response", func(t *testing.T) {
		largeData := make([]map[string]interface{}, 10000)
		for i := 0; i < 10000; i++ {
			largeData[i] = map[string]interface{}{"id": i, "name": strings.Repeat("test", 100)}
		}
		w := httptest.NewRecorder()
		respondJSON(w, http.StatusOK, &models.APIResponse{Status: "success", Data: largeData})

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
		if w.Header().Get("Content-Type") != "application/json" || w.Header().Get("ETag") == "" {
			t.Error("Missing expected headers")
		}
	})

	t.Run("special characters", func(t *testing.T) {
		tests := []struct {
			name string
			data interface{}
		}{
			{"unicode", map[string]string{"name": "日本語テスト"}},
			{"emoji", map[string]string{"status": "🎬"}},
			{"html entities", map[string]string{"html": "<script>alert('xss')</script>"}},
			{"null bytes", map[string]string{"data": "test\x00data"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				respondJSON(w, http.StatusOK, &models.APIResponse{Status: "success", Data: tt.data})

				var decoded models.APIResponse
				if err := json.NewDecoder(w.Body).Decode(&decoded); err != nil {
					t.Errorf("Failed to decode: %v", err)
				}
			})
		}
	})
}

// TestRespondError_EdgeCases tests error response edge cases
func TestRespondError_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("with error", func(t *testing.T) {
		w := httptest.NewRecorder()
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "Database error", errors.New("test error"))

		var resp models.APIResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if resp.Error == nil || resp.Error.Code != "DB_ERROR" {
			t.Error("Expected DB_ERROR")
		}
	})

	t.Run("all status codes", func(t *testing.T) {
		codes := []int{400, 401, 403, 404, 405, 409, 410, 429, 500, 501, 502, 503}
		for _, code := range codes {
			w := httptest.NewRecorder()
			respondError(w, code, "TEST", "msg", nil)
			if w.Code != code {
				t.Errorf("Expected %d, got %d", code, w.Code)
			}
		}
	})
}

// TestGenerateETag_EdgeCases tests ETag generation edge cases
func TestGenerateETag_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("consistency", func(t *testing.T) {
		data := []byte(`{"key": "value"}`)
		etags := make(map[string]int)
		for i := 0; i < 1000; i++ {
			etags[generateETag(data)]++
		}
		if len(etags) != 1 {
			t.Errorf("Expected 1 unique ETag, got %d", len(etags))
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		etags := make(map[string]bool)
		for i := 0; i < 1000; i++ {
			etag := generateETag([]byte(strings.Repeat("x", i)))
			if etags[etag] {
				t.Errorf("Collision at %d", i)
			}
			etags[etag] = true
		}
	})
}

// TestParseCommaSeparated_ExtendedCases tests comma-separated parsing
func TestParseCommaSeparated_ExtendedCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"single space", "a b", []string{"a b"}},
		{"tabs", "a\tb", []string{"a\tb"}},
		{"only commas", ",,,", []string{}},
		{"numbers", "1,2,3", []string{"1", "2", "3"}},
		{"quoted", `"a","b"`, []string{`"a"`, `"b"`}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparated(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d items, got %d: %v", len(tt.expected), len(result), result)
			}
		})
	}
}

// TestEscapeCSV_RealWorldCases tests CSV escaping
func TestEscapeCSV_RealWorldCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"comma", "a,b", `"a,b"`},
		{"quotes", `"a"`, `"""a"""`},
		{"newline", "a\nb", "\"a\nb\""},
		{"unicode", "日本映画", "日本映画"},
		{"emoji", "🎬", "🎬"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := escapeCSV(tt.input); result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestHandler_FullLifecycle tests complete handler lifecycle
func TestHandler_FullLifecycle(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		API:      config.APIConfig{DefaultPageSize: 100, MaxPageSize: 1000},
		Security: config.SecurityConfig{CORSOrigins: []string{"*"}},
	}
	wsHub := ws.NewHub()
	go wsHub.RunWithContext(context.Background())

	handler := NewHandler(nil, nil, nil, cfg, nil, wsHub)
	handler.SetBackupManager(&mockBackupManager{})
	handler.cache.Set("test-key", "test-value")
	handler.GetCacheStats()
	handler.ClearCache()

	if _, found := handler.cache.Get("test-key"); found {
		t.Error("Cache should be cleared")
	}
	handler.OnSyncCompleted(100, 500)
}

// TestConcurrentRequests_MixedOperations tests concurrent operations
func TestConcurrentRequests_MixedOperations(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		API:      config.APIConfig{DefaultPageSize: 100},
		Security: config.SecurityConfig{CORSOrigins: []string{"*"}},
	}
	wsHub := ws.NewHub()
	go wsHub.RunWithContext(context.Background())

	handler := NewHandler(nil, nil, nil, cfg, nil, wsHub)

	var wg sync.WaitGroup
	ops := []func(){
		func() { handler.ClearCache() },
		func() { handler.OnSyncCompleted(1, 100) },
		func() { handler.GetCacheStats() },
		func() { handler.GetPerformanceStats() },
		func() { handler.cache.Set("key", "value") },
		func() { handler.cache.Get("key") },
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ops[n%len(ops)]()
		}(i)
	}
	wg.Wait()
}

// TestAnalyticsExecutor_FilterPreservation tests filter preservation
func TestAnalyticsExecutor_FilterPreservation(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{API: config.APIConfig{DefaultPageSize: 100, MaxPageSize: 1000}}
	handler := &Handler{db: &database.DB{}, cache: cache.New(5 * time.Minute), config: cfg}
	executor := NewAnalyticsQueryExecutor(handler)

	var capturedFilter database.LocationStatsFilter
	queryFunc := func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		capturedFilter = filter
		return map[string]interface{}{"captured": true}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test?users=alice,bob&media_types=movie&days=30", nil)
	w := httptest.NewRecorder()
	executor.ExecuteSimple(w, req, "FilterTest", queryFunc)

	if len(capturedFilter.Users) == 0 {
		t.Error("Expected users filter")
	}
}

// TestSpatialExecutor_BoundingBoxEdgeCases tests bounding box edge cases
func TestSpatialExecutor_BoundingBoxEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params map[string]string
	}{
		{"point box", map[string]string{"west": "0", "south": "0", "east": "0", "north": "0"}},
		{"line box", map[string]string{"west": "-74", "south": "40", "east": "-74", "north": "41"}},
		{"global box", map[string]string{"west": "-180", "south": "-90", "east": "180", "north": "90"}},
		{"anti-meridian", map[string]string{"west": "170", "south": "-10", "east": "-170", "north": "10"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test?"
			for k, v := range tt.params {
				url += k + "=" + v + "&"
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			if box, err := ValidateBoundingBox(req); err != nil || box == nil {
				t.Errorf("Error: %v, box: %v", err, box)
			}
		})
	}
}

// Benchmarks

func BenchmarkRespondJSON_LargePayload(b *testing.B) {
	largeData := make([]map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		largeData[i] = map[string]interface{}{"id": i, "name": "test"}
	}
	resp := &models.APIResponse{Status: "success", Data: largeData}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		respondJSON(w, http.StatusOK, resp)
	}
}

func BenchmarkParseCommaSeparated_LargeInput(b *testing.B) {
	parts := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		parts[i] = "item" + string(rune('0'+i%10))
	}
	input := strings.Join(parts, ",")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseCommaSeparated(input)
	}
}

func BenchmarkConcurrentCacheOperations(b *testing.B) {
	c := cache.New(5 * time.Minute)
	handler := &Handler{cache: c}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "key" + string(rune('0'+i%10))
			if i%2 == 0 {
				c.Set(key, "value")
			} else {
				c.Get(key)
			}
			if i%100 == 0 {
				handler.ClearCache()
			}
			i++
		}
	})
}
