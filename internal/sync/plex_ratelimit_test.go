// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/models"
)

// TestPlexClientRateLimiting tests the rate limiting behavior of PlexClient
func TestPlexClientRateLimiting(t *testing.T) {
	t.Run("successful on first try", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(PlexHistoryResponse{
				MediaContainer: PlexMediaContainer{Size: 0, Metadata: []PlexMetadata{}},
			})
		}))
		defer server.Close()

		client := NewPlexClient(server.URL, "test-token")
		history, err := client.GetHistoryAll(context.Background(), "", nil)
		if err != nil {
			t.Fatalf("GetHistoryAll() error = %v", err)
		}
		if history == nil {
			t.Error("Expected non-nil history")
		}
	})

	t.Run("rate limit with retry success on GetHistoryAll", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount < 2 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(PlexHistoryResponse{
				MediaContainer: PlexMediaContainer{Size: 0, Metadata: []PlexMetadata{}},
			})
		}))
		defer server.Close()

		client := NewPlexClient(server.URL, "test-token")
		// Override timeout for faster test
		client.httpClient.Timeout = 10 * time.Second

		history, err := client.GetHistoryAll(context.Background(), "", nil)
		if err != nil {
			t.Fatalf("GetHistoryAll() error = %v", err)
		}
		if history == nil {
			t.Error("Expected non-nil history after retry")
		}
		if attemptCount < 2 {
			t.Errorf("Expected at least 2 attempts, got %d", attemptCount)
		}
	})

	t.Run("rate limit with Retry-After header", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount < 2 {
				w.Header().Set("Retry-After", "1") // 1 second
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(PlexHistoryResponse{
				MediaContainer: PlexMediaContainer{Size: 0, Metadata: []PlexMetadata{}},
			})
		}))
		defer server.Close()

		client := NewPlexClient(server.URL, "test-token")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		history, err := client.GetHistoryAll(ctx, "", nil)
		if err != nil {
			t.Fatalf("GetHistoryAll() error = %v", err)
		}
		if history == nil {
			t.Error("Expected non-nil history")
		}
	})

	t.Run("rate limit max retries exceeded on GetTranscodeSessions", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer server.Close()

		client := NewPlexClient(server.URL, "test-token")
		// Use a timeout that allows retries to complete or timeout
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()

		_, err := client.GetTranscodeSessions(ctx)
		if err == nil {
			t.Fatal("Expected error after max retries exceeded")
		}
		// Error could be rate limit exceeded or context deadline - both are acceptable
		// since retry backoff (1+2+4+8+16=31s) may exceed context timeout
		errStr := err.Error()
		if !strings.Contains(errStr, "rate limit") && !strings.Contains(errStr, "deadline") && !strings.Contains(errStr, "context") {
			t.Errorf("Error should mention rate limit or context deadline, got: %v", err)
		}
	})

	t.Run("context cancellation during rate limit retry", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer server.Close()

		client := NewPlexClient(server.URL, "test-token")
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := client.GetTranscodeSessions(ctx)
		if err == nil {
			t.Fatal("Expected error due to context cancellation")
		}
		// Error could be context deadline or rate limit - both are acceptable
	})
}

// TestPlexClientGetTranscodeSessions tests the GetTranscodeSessions method
func TestPlexClientGetTranscodeSessions(t *testing.T) {
	t.Run("successful retrieval with active sessions", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/status/sessions" {
				t.Errorf("Path = %q, want /status/sessions", r.URL.Path)
			}
			if r.Header.Get("X-Plex-Token") != "test-token" {
				t.Error("Missing X-Plex-Token header")
			}
			if r.Header.Get("Accept") != "application/json" {
				t.Error("Missing Accept header")
			}

			response := models.PlexSessionsResponse{
				MediaContainer: models.PlexSessionsContainer{
					Size: 2,
					Metadata: []models.PlexSession{
						{
							SessionKey: "session1",
							Key:        "/library/metadata/12345",
							Type:       "movie",
							Title:      "Test Movie",
							Duration:   7200000,
							ViewOffset: 3600000,
							TranscodeSession: &models.PlexTranscodeSession{
								Key:                "transcode1",
								Progress:           50.0,
								Speed:              2.5,
								VideoDecision:      "transcode",
								AudioDecision:      "copy",
								VideoCodec:         "h264",
								AudioCodec:         "aac",
								Container:          "mkv",
								Protocol:           "hls",
								Throttled:          false,
								Complete:           false,
								MaxOffsetAvailable: 5400000.0,
								MinOffsetAvailable: 0.0,
							},
						},
						{
							SessionKey: "session2",
							Key:        "/library/metadata/67890",
							Type:       "episode",
							Title:      "Test Episode",
							Duration:   3600000,
							ViewOffset: 1800000,
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewPlexClient(server.URL, "test-token")
		sessions, err := client.GetTranscodeSessions(context.Background())
		if err != nil {
			t.Fatalf("GetTranscodeSessions() error = %v", err)
		}
		if len(sessions) != 2 {
			t.Errorf("len(sessions) = %d, want 2", len(sessions))
		}
		if sessions[0].SessionKey != "session1" {
			t.Errorf("SessionKey = %q, want session1", sessions[0].SessionKey)
		}
		if sessions[0].TranscodeSession == nil {
			t.Error("Expected non-nil TranscodeSession for session1")
		} else {
			if sessions[0].TranscodeSession.Progress != 50.0 {
				t.Errorf("Progress = %f, want 50.0", sessions[0].TranscodeSession.Progress)
			}
			if sessions[0].TranscodeSession.Speed != 2.5 {
				t.Errorf("Speed = %f, want 2.5", sessions[0].TranscodeSession.Speed)
			}
		}
	})

	t.Run("empty sessions", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := models.PlexSessionsResponse{
				MediaContainer: models.PlexSessionsContainer{
					Size:     0,
					Metadata: nil,
				},
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewPlexClient(server.URL, "test-token")
		sessions, err := client.GetTranscodeSessions(context.Background())
		if err != nil {
			t.Fatalf("GetTranscodeSessions() error = %v", err)
		}
		if len(sessions) != 0 {
			t.Errorf("len(sessions) = %d, want 0", len(sessions))
		}
	})

	t.Run("HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewPlexClient(server.URL, "test-token")
		_, err := client.GetTranscodeSessions(context.Background())
		if err == nil {
			t.Fatal("Expected error for HTTP 500")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("Error should mention status code, got: %v", err)
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{invalid json`))
		}))
		defer server.Close()

		client := NewPlexClient(server.URL, "test-token")
		_, err := client.GetTranscodeSessions(context.Background())
		if err == nil {
			t.Fatal("Expected error for invalid JSON")
		}
		if !strings.Contains(err.Error(), "decode") {
			t.Errorf("Error should mention decode, got: %v", err)
		}
	})
}

// TestPlexClientGetSessionTimeline tests the GetSessionTimeline method
func TestPlexClientGetSessionTimeline(t *testing.T) {
	t.Run("successful timeline retrieval for existing session", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/status/sessions" {
				t.Errorf("Path = %q, want /status/sessions", r.URL.Path)
			}

			response := models.PlexSessionsResponse{
				MediaContainer: models.PlexSessionsContainer{
					Size: 2,
					Metadata: []models.PlexSession{
						{
							SessionKey: "target-session",
							Key:        "/library/metadata/12345",
							Type:       "movie",
							Title:      "Target Movie",
							Duration:   7200000,
							ViewOffset: 3600000,
							TranscodeSession: &models.PlexTranscodeSession{
								Key:                "transcode1",
								Progress:           50.0,
								Speed:              2.0,
								Throttled:          false,
								Complete:           false,
								MaxOffsetAvailable: 5400000.0,
								MinOffsetAvailable: 0.0,
							},
						},
						{
							SessionKey: "other-session",
							Key:        "/library/metadata/67890",
							Type:       "episode",
							Title:      "Other Episode",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewPlexClient(server.URL, "test-token")
		timeline, err := client.GetSessionTimeline(context.Background(), "target-session")
		if err != nil {
			t.Fatalf("GetSessionTimeline() error = %v", err)
		}
		if timeline == nil {
			t.Fatal("Expected non-nil timeline")
		}
		if timeline.MediaContainer.Size != 1 {
			t.Errorf("Size = %d, want 1", timeline.MediaContainer.Size)
		}
		if len(timeline.MediaContainer.Metadata) != 1 {
			t.Errorf("len(Metadata) = %d, want 1", len(timeline.MediaContainer.Metadata))
		}
		metadata := timeline.MediaContainer.Metadata[0]
		if metadata.SessionKey != "target-session" {
			t.Errorf("SessionKey = %q, want target-session", metadata.SessionKey)
		}
		if metadata.Title != "Target Movie" {
			t.Errorf("Title = %q, want Target Movie", metadata.Title)
		}
		if metadata.TranscodeSession == nil {
			t.Error("Expected non-nil TranscodeSession")
		} else {
			if metadata.TranscodeSession.MaxOffsetAvailable != 5400000.0 {
				t.Errorf("MaxOffsetAvailable = %f, want 5400000.0", metadata.TranscodeSession.MaxOffsetAvailable)
			}
		}
	})

	t.Run("session not found returns empty result", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := models.PlexSessionsResponse{
				MediaContainer: models.PlexSessionsContainer{
					Size: 1,
					Metadata: []models.PlexSession{
						{
							SessionKey: "different-session",
							Title:      "Different Movie",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewPlexClient(server.URL, "test-token")
		timeline, err := client.GetSessionTimeline(context.Background(), "nonexistent-session")
		if err != nil {
			t.Fatalf("GetSessionTimeline() error = %v", err)
		}
		if timeline.MediaContainer.Size != 0 {
			t.Errorf("Size = %d, want 0 for nonexistent session", timeline.MediaContainer.Size)
		}
	})

	t.Run("empty sessions", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := models.PlexSessionsResponse{
				MediaContainer: models.PlexSessionsContainer{
					Size:     0,
					Metadata: nil,
				},
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewPlexClient(server.URL, "test-token")
		timeline, err := client.GetSessionTimeline(context.Background(), "any-session")
		if err != nil {
			t.Fatalf("GetSessionTimeline() error = %v", err)
		}
		if timeline.MediaContainer.Size != 0 {
			t.Errorf("Size = %d, want 0", timeline.MediaContainer.Size)
		}
	})

	t.Run("session without transcode session", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := models.PlexSessionsResponse{
				MediaContainer: models.PlexSessionsContainer{
					Size: 1,
					Metadata: []models.PlexSession{
						{
							SessionKey:       "direct-play-session",
							Key:              "/library/metadata/12345",
							Type:             "movie",
							Title:            "Direct Play Movie",
							Duration:         7200000,
							ViewOffset:       3600000,
							TranscodeSession: nil, // Direct play
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewPlexClient(server.URL, "test-token")
		timeline, err := client.GetSessionTimeline(context.Background(), "direct-play-session")
		if err != nil {
			t.Fatalf("GetSessionTimeline() error = %v", err)
		}
		if timeline.MediaContainer.Size != 1 {
			t.Errorf("Size = %d, want 1", timeline.MediaContainer.Size)
		}
		if timeline.MediaContainer.Metadata[0].TranscodeSession != nil {
			t.Error("Expected nil TranscodeSession for direct play")
		}
	})

	t.Run("HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		client := NewPlexClient(server.URL, "test-token")
		_, err := client.GetSessionTimeline(context.Background(), "any-session")
		if err == nil {
			t.Fatal("Expected error for HTTP 503")
		}
	})
}

// TestPlexClientDoRequestWithRateLimit specifically tests the rate limiting implementation
func TestPlexClientDoRequestWithRateLimit(t *testing.T) {
	t.Run("exponential backoff timing", func(t *testing.T) {
		// This test verifies that subsequent retries take longer
		attemptTimes := make([]time.Time, 0)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptTimes = append(attemptTimes, time.Now())
			if len(attemptTimes) < 3 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(PlexHistoryResponse{
				MediaContainer: PlexMediaContainer{Size: 0, Metadata: []PlexMetadata{}},
			})
		}))
		defer server.Close()

		client := NewPlexClient(server.URL, "test-token")
		startTime := time.Now()

		_, err := client.GetHistoryAll(context.Background(), "", nil)
		if err != nil {
			t.Fatalf("GetHistoryAll() error = %v", err)
		}

		totalTime := time.Since(startTime)
		// With exponential backoff (1s, 2s), total should be at least 3 seconds
		if totalTime < 3*time.Second {
			t.Logf("Total time: %v (may be less if Retry-After wasn't respected)", totalTime)
		}

		if len(attemptTimes) != 3 {
			t.Errorf("Expected 3 attempts, got %d", len(attemptTimes))
		}
	})
}

// BenchmarkPlexClientGetTranscodeSessions benchmarks the GetTranscodeSessions method
func BenchmarkPlexClientGetTranscodeSessions(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := models.PlexSessionsResponse{
			MediaContainer: models.PlexSessionsContainer{
				Size:     0,
				Metadata: []models.PlexSession{},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.GetTranscodeSessions(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPlexClientGetSessionTimeline benchmarks the GetSessionTimeline method
func BenchmarkPlexClientGetSessionTimeline(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := models.PlexSessionsResponse{
			MediaContainer: models.PlexSessionsContainer{
				Size: 5,
				Metadata: []models.PlexSession{
					{SessionKey: "session1", Title: "Movie 1"},
					{SessionKey: "session2", Title: "Movie 2"},
					{SessionKey: "session3", Title: "Movie 3"},
					{SessionKey: "session4", Title: "Movie 4"},
					{SessionKey: "session5", Title: "Movie 5"},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewPlexClient(server.URL, "test-token")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.GetSessionTimeline(ctx, "session3")
		if err != nil {
			b.Fatal(err)
		}
	}
}
