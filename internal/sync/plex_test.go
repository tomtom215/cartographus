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

// Test helper functions to reduce cyclomatic complexity

// assertStringField checks a string field value
func assertStringField(t *testing.T, got, want, field string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", field, got, want)
	}
}

// assertIntField checks an integer field value
func assertIntField(t *testing.T, got, want int, field string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %d, want %d", field, got, want)
	}
}

// assertInt64Field checks an int64 field value
func assertInt64Field(t *testing.T, got, want int64, field string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %d, want %d", field, got, want)
	}
}

// assertFloatField checks a float64 field value
func assertFloatField(t *testing.T, got, want float64, field string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", field, got, want)
	}
}

// assertSliceLen checks slice length
func assertSliceLen(t *testing.T, got, want int, field string) {
	t.Helper()
	if got != want {
		t.Errorf("len(%s) = %d, want %d", field, got, want)
	}
}

// assertNotNil checks that a value is not nil
func assertNotNil(t *testing.T, val interface{}, field string) {
	t.Helper()
	if val == nil {
		t.Fatalf("%s should not be nil", field)
	}
}

// assertNoError checks that error is nil
func assertNoError(t *testing.T, err error, context string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: unexpected error = %v", context, err)
	}
}

// assertError checks that error occurred
func assertError(t *testing.T, err error, context string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: expected error, got nil", context)
	}
}

// assertErrorContains checks that error contains expected string
func assertErrorContains(t *testing.T, err error, expected, context string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: expected error containing %q, got nil", context, expected)
		return
	}
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("%s: error = %v, want error containing %q", context, err, expected)
	}
}

// assertPlexClient validates PlexClient initialization
func assertPlexClient(t *testing.T, client *PlexClient, expectedURL, expectedToken string) {
	t.Helper()
	assertNotNil(t, client, "PlexClient")
	assertStringField(t, client.baseURL, expectedURL, "baseURL")
	assertStringField(t, client.token, expectedToken, "token")
	assertNotNil(t, client.httpClient, "httpClient")
	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("httpClient.Timeout = %v, want %v", client.httpClient.Timeout, 30*time.Second)
	}
}

// assertPlexIdentity validates PlexIdentityContainer fields
func assertPlexIdentity(t *testing.T, identity *PlexIdentityContainer, machineID, version, platform string) {
	t.Helper()
	assertStringField(t, identity.MachineIdentifier, machineID, "MachineIdentifier")
	assertStringField(t, identity.Version, version, "Version")
	assertStringField(t, identity.Platform, platform, "Platform")
}

// assertPlexMetadata validates PlexMetadata fields
func assertPlexMetadata(t *testing.T, meta PlexMetadata, ratingKey, mediaType, title string) {
	t.Helper()
	assertStringField(t, meta.RatingKey, ratingKey, "RatingKey")
	assertStringField(t, meta.Type, mediaType, "Type")
	assertStringField(t, meta.Title, title, "Title")
}

// assertPlexSession validates PlexSession fields
func assertPlexSession(t *testing.T, session models.PlexSession, sessionKey, title string) {
	t.Helper()
	assertStringField(t, session.SessionKey, sessionKey, "SessionKey")
	assertStringField(t, session.Title, title, "Title")
}

// newMockPlexServer creates a test server with custom handler
func newMockPlexServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// newMockJSONServer creates a test server that returns JSON response
func newMockJSONServer(t *testing.T, response interface{}) *httptest.Server {
	t.Helper()
	return newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(response)
	})
}

// verifyPlexTokenHeader checks X-Plex-Token header
func verifyPlexTokenHeader(t *testing.T, r *http.Request, expectedToken string) {
	t.Helper()
	got := r.Header.Get("X-Plex-Token")
	if got != expectedToken {
		t.Errorf("X-Plex-Token = %q, want %q", got, expectedToken)
	}
}

// verifyRequestPath checks request path
func verifyRequestPath(t *testing.T, r *http.Request, expectedPath string) {
	t.Helper()
	if r.URL.Path != expectedPath {
		t.Errorf("Path = %q, want %q", r.URL.Path, expectedPath)
	}
}

func TestNewPlexClient(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		token   string
	}{
		{
			name:    "standard initialization",
			baseURL: "http://localhost:32400",
			token:   "plex-token-123",
		},
		{
			name:    "HTTPS URL",
			baseURL: "https://plex.example.com:32400",
			token:   "secure-token-456",
		},
		{
			name:    "custom port",
			baseURL: "http://192.168.1.100:8080",
			token:   "custom-token",
		},
		{
			name:    "empty token (invalid but should not panic)",
			baseURL: "http://localhost:32400",
			token:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewPlexClient(tt.baseURL, tt.token)
			assertPlexClient(t, client, tt.baseURL, tt.token)
		})
	}
}

func TestPlexClient_Ping(t *testing.T) {
	t.Run("successful ping", func(t *testing.T) {
		server := newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
			verifyPlexTokenHeader(t, r, "test-token")
			verifyRequestPath(t, r, "/")
			w.WriteHeader(http.StatusOK)
		})

		client := NewPlexClient(server.URL, "test-token")
		err := client.Ping(context.Background())
		assertNoError(t, err, "Ping()")
	})

	t.Run("authentication failure", func(t *testing.T) {
		server := newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})

		client := NewPlexClient(server.URL, "invalid-token")
		err := client.Ping(context.Background())
		assertErrorContains(t, err, "401", "Ping()")
	})

	t.Run("server error", func(t *testing.T) {
		server := newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		client := NewPlexClient(server.URL, "test-token")
		err := client.Ping(context.Background())
		assertError(t, err, "Ping() with 500 response")
	})

	t.Run("context cancellation", func(t *testing.T) {
		server := newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second) // Simulate slow server
		})

		client := NewPlexClient(server.URL, "test-token")
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := client.Ping(ctx)
		assertError(t, err, "Ping() with context timeout")
	})
}

func TestPlexClient_GetServerIdentity(t *testing.T) {
	t.Run("successful identity retrieval", func(t *testing.T) {
		server := newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
			verifyRequestPath(t, r, "/identity")
			verifyPlexTokenHeader(t, r, "test-token")
			assertStringField(t, r.Header.Get("Accept"), "application/json", "Accept header")

			json.NewEncoder(w).Encode(PlexIdentityResponse{
				MediaContainer: PlexIdentityContainer{
					MachineIdentifier: "abc123def456",
					Version:           "1.40.0.8395",
					Platform:          "Linux",
				},
			})
		})

		client := NewPlexClient(server.URL, "test-token")
		identity, err := client.GetServerIdentity(context.Background())
		assertNoError(t, err, "GetServerIdentity()")
		assertNotNil(t, identity, "identity")
		assertPlexIdentity(t, identity, "abc123def456", "1.40.0.8395", "Linux")
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{invalid json`))
		})

		client := NewPlexClient(server.URL, "test-token")
		identity, err := client.GetServerIdentity(context.Background())
		assertError(t, err, "GetServerIdentity() with invalid JSON")
		if identity != nil {
			t.Errorf("identity should be nil, got %v", identity)
		}
	})

	t.Run("HTTP error response", func(t *testing.T) {
		server := newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		client := NewPlexClient(server.URL, "test-token")
		identity, err := client.GetServerIdentity(context.Background())
		assertError(t, err, "GetServerIdentity() with 404 response")
		if identity != nil {
			t.Errorf("identity should be nil, got %v", identity)
		}
	})
}

func TestPlexClient_GetHistoryAll(t *testing.T) {
	t.Run("successful history retrieval", func(t *testing.T) {
		server := newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
			verifyRequestPath(t, r, "/status/sessions/history/all")
			verifyPlexTokenHeader(t, r, "test-token")

			json.NewEncoder(w).Encode(PlexHistoryResponse{
				MediaContainer: PlexMediaContainer{
					Size: 2,
					Metadata: []PlexMetadata{
						{
							RatingKey:  "12345",
							Type:       "episode",
							Title:      "Episode Title",
							ViewedAt:   1732483200,
							Duration:   3600000, // 1 hour in ms
							ViewOffset: 1800000, // 30 minutes
							AccountID:  1,
						},
						{
							RatingKey:        "67890",
							Type:             "movie",
							Title:            "Movie Title",
							GrandparentTitle: "",
							ViewedAt:         1732396800,
							Duration:         7200000, // 2 hours
							AccountID:        2,
						},
					},
				},
			})
		})

		client := NewPlexClient(server.URL, "test-token")
		history, err := client.GetHistoryAll(context.Background(), "viewedAt", nil)
		assertNoError(t, err, "GetHistoryAll()")
		assertSliceLen(t, len(history), 2, "history")
		assertPlexMetadata(t, history[0], "12345", "episode", "Episode Title")
	})

	t.Run("with sort parameter", func(t *testing.T) {
		server := newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
			assertStringField(t, r.URL.Query().Get("sort"), "-viewedAt", "sort parameter")
			json.NewEncoder(w).Encode(PlexHistoryResponse{MediaContainer: PlexMediaContainer{}})
		})

		client := NewPlexClient(server.URL, "test-token")
		_, err := client.GetHistoryAll(context.Background(), "-viewedAt", nil)
		assertNoError(t, err, "GetHistoryAll() with sort")
	})

	t.Run("with accountID filter", func(t *testing.T) {
		server := newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
			assertStringField(t, r.URL.Query().Get("accountID"), "42", "accountID parameter")
			json.NewEncoder(w).Encode(PlexHistoryResponse{MediaContainer: PlexMediaContainer{}})
		})

		client := NewPlexClient(server.URL, "test-token")
		accountID := 42
		_, err := client.GetHistoryAll(context.Background(), "", &accountID)
		assertNoError(t, err, "GetHistoryAll() with accountID")
	})

	t.Run("empty history", func(t *testing.T) {
		server := newMockJSONServer(t, PlexHistoryResponse{
			MediaContainer: PlexMediaContainer{Size: 0, Metadata: []PlexMetadata{}},
		})

		client := NewPlexClient(server.URL, "test-token")
		history, err := client.GetHistoryAll(context.Background(), "", nil)
		assertNoError(t, err, "GetHistoryAll() with empty history")
		assertSliceLen(t, len(history), 0, "history")
	})

	t.Run("HTTP error response", func(t *testing.T) {
		server := newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})

		client := NewPlexClient(server.URL, "test-token")
		history, err := client.GetHistoryAll(context.Background(), "", nil)
		assertError(t, err, "GetHistoryAll() with 403 response")
		if len(history) != 0 {
			t.Errorf("history should be empty on error, got %d items", len(history))
		}
	})
}

func TestPlexClient_GetTranscodeSessions(t *testing.T) {
	t.Run("active sessions", func(t *testing.T) {
		server := newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
			verifyRequestPath(t, r, "/status/sessions")

			json.NewEncoder(w).Encode(models.PlexSessionsResponse{
				MediaContainer: models.PlexSessionsContainer{
					Size: 2,
					Metadata: []models.PlexSession{
						{
							SessionKey: "session-1",
							Title:      "Movie 1",
							Type:       "movie",
							TranscodeSession: &models.PlexTranscodeSession{
								VideoDecision: "transcode",
								Progress:      45.5,
								Speed:         2.0,
							},
						},
						{
							SessionKey: "session-2",
							Title:      "Episode 1",
							Type:       "episode",
							TranscodeSession: &models.PlexTranscodeSession{
								VideoDecision: "copy",
								Progress:      100.0,
								Speed:         0.0,
							},
						},
					},
				},
			})
		})

		client := NewPlexClient(server.URL, "test-token")
		sessions, err := client.GetTranscodeSessions(context.Background())
		assertNoError(t, err, "GetTranscodeSessions()")
		assertSliceLen(t, len(sessions), 2, "sessions")
		assertPlexSession(t, sessions[0], "session-1", "Movie 1")
		assertFloatField(t, sessions[0].TranscodeSession.Progress, 45.5, "sessions[0].TranscodeSession.Progress")
	})

	t.Run("no active sessions", func(t *testing.T) {
		server := newMockJSONServer(t, models.PlexSessionsResponse{
			MediaContainer: models.PlexSessionsContainer{Size: 0, Metadata: nil},
		})

		client := NewPlexClient(server.URL, "test-token")
		sessions, err := client.GetTranscodeSessions(context.Background())
		assertNoError(t, err, "GetTranscodeSessions() with empty sessions")
		assertNotNil(t, sessions, "sessions")
		assertSliceLen(t, len(sessions), 0, "sessions")
	})

	t.Run("HTTP error", func(t *testing.T) {
		server := newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		client := NewPlexClient(server.URL, "test-token")
		sessions, err := client.GetTranscodeSessions(context.Background())
		assertError(t, err, "GetTranscodeSessions() with 500 response")
		if len(sessions) != 0 {
			t.Errorf("sessions should be empty on error, got %d items", len(sessions))
		}
	})
}

func TestPlexClient_GetSessionTimeline(t *testing.T) {
	t.Run("finds matching session", func(t *testing.T) {
		server := newMockJSONServer(t, models.PlexSessionsResponse{
			MediaContainer: models.PlexSessionsContainer{
				Size: 2,
				Metadata: []models.PlexSession{
					{
						SessionKey: "target-session",
						Title:      "Target Movie",
						Type:       "movie",
						Key:        "/library/metadata/12345",
						ViewOffset: 1800000, // 30 min
						Duration:   7200000, // 2 hours
						TranscodeSession: &models.PlexTranscodeSession{
							MaxOffsetAvailable: 2400000, // 40 min buffered
							MinOffsetAvailable: 0,
							Progress:           50.0,
							Speed:              1.5,
							Throttled:          false,
							Complete:           false,
							Key:                "/transcode/sessions/abc",
						},
					},
					{
						SessionKey: "other-session",
						Title:      "Other Movie",
						Type:       "movie",
					},
				},
			},
		})

		client := NewPlexClient(server.URL, "test-token")
		timeline, err := client.GetSessionTimeline(context.Background(), "target-session")
		assertNoError(t, err, "GetSessionTimeline()")
		assertIntField(t, timeline.MediaContainer.Size, 1, "Size")
		assertSliceLen(t, len(timeline.MediaContainer.Metadata), 1, "Metadata")

		meta := timeline.MediaContainer.Metadata[0]
		assertStringField(t, meta.SessionKey, "target-session", "SessionKey")
		assertStringField(t, meta.Title, "Target Movie", "Title")
		assertInt64Field(t, meta.ViewOffset, 1800000, "ViewOffset")
		assertNotNil(t, meta.TranscodeSession, "TranscodeSession")
		assertFloatField(t, meta.TranscodeSession.MaxOffsetAvailable, 2400000, "MaxOffsetAvailable")
	})

	t.Run("session not found", func(t *testing.T) {
		server := newMockJSONServer(t, models.PlexSessionsResponse{
			MediaContainer: models.PlexSessionsContainer{
				Size: 1,
				Metadata: []models.PlexSession{
					{SessionKey: "other-session", Title: "Other"},
				},
			},
		})

		client := NewPlexClient(server.URL, "test-token")
		timeline, err := client.GetSessionTimeline(context.Background(), "nonexistent")
		assertNoError(t, err, "GetSessionTimeline() with nonexistent session")
		assertIntField(t, timeline.MediaContainer.Size, 0, "Size")
	})

	t.Run("session without transcode", func(t *testing.T) {
		server := newMockJSONServer(t, models.PlexSessionsResponse{
			MediaContainer: models.PlexSessionsContainer{
				Size: 1,
				Metadata: []models.PlexSession{
					{
						SessionKey:       "direct-play",
						Title:            "Direct Play Movie",
						Type:             "movie",
						TranscodeSession: nil, // No transcode - direct play
					},
				},
			},
		})

		client := NewPlexClient(server.URL, "test-token")
		timeline, err := client.GetSessionTimeline(context.Background(), "direct-play")
		assertNoError(t, err, "GetSessionTimeline() with direct play")
		assertIntField(t, timeline.MediaContainer.Size, 1, "Size")
		meta := timeline.MediaContainer.Metadata[0]
		if meta.TranscodeSession != nil {
			t.Errorf("TranscodeSession should be nil for direct play, got %v", meta.TranscodeSession)
		}
	})
}

func TestPlexClient_DoRequestWithRateLimit(t *testing.T) {
	t.Run("success without rate limiting", func(t *testing.T) {
		server := newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		})

		client := NewPlexClient(server.URL, "test-token")
		req, _ := http.NewRequest("GET", server.URL+"/test", nil)
		resp, err := client.doRequestWithRateLimit(req)
		assertNoError(t, err, "doRequestWithRateLimit()")
		defer resp.Body.Close()
		assertIntField(t, resp.StatusCode, http.StatusOK, "StatusCode")
	})

	t.Run("retries on rate limiting", func(t *testing.T) {
		attempts := 0
		server := newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		client := NewPlexClient(server.URL, "test-token")
		req, _ := http.NewRequest("GET", server.URL+"/test", nil)
		resp, err := client.doRequestWithRateLimit(req)
		assertNoError(t, err, "doRequestWithRateLimit() with retries")
		defer resp.Body.Close()
		assertIntField(t, attempts, 3, "attempts")
	})

	t.Run("exceeds max retries", func(t *testing.T) {
		server := newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		})

		client := NewPlexClient(server.URL, "test-token")
		req, _ := http.NewRequest("GET", server.URL+"/test", nil)
		resp, err := client.doRequestWithRateLimit(req)
		assertError(t, err, "doRequestWithRateLimit() exceeding max retries")
		assertErrorContains(t, err, "rate limit exceeded", "doRequestWithRateLimit() error message")
		if resp != nil {
			resp.Body.Close()
		}
	})

	t.Run("respects context cancellation during retry", func(t *testing.T) {
		server := newMockPlexServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		})

		client := NewPlexClient(server.URL, "test-token")
		ctx, cancel := context.WithCancel(context.Background())

		req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/test", nil)

		// Cancel after a short delay
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		resp, err := client.doRequestWithRateLimit(req)
		assertError(t, err, "doRequestWithRateLimit() with context cancellation")
		if resp != nil {
			resp.Body.Close()
		}
	})
}

// Test PlexMetadata struct
func TestPlexMetadata_Fields(t *testing.T) {
	t.Run("full metadata parsing", func(t *testing.T) {
		jsonData := `{
			"ratingKey": "12345",
			"key": "/library/metadata/12345",
			"parentRatingKey": "123",
			"grandparentRatingKey": "1",
			"type": "episode",
			"title": "Episode Title",
			"grandparentTitle": "Show Name",
			"parentTitle": "Season 1",
			"viewedAt": 1732483200,
			"duration": 3600000,
			"viewOffset": 1800000,
			"accountID": 42,
			"index": 5,
			"parentIndex": 1,
			"year": 2024,
			"guid": "plex://episode/12345"
		}`

		var meta PlexMetadata
		err := json.Unmarshal([]byte(jsonData), &meta)
		assertNoError(t, err, "JSON unmarshal")

		assertPlexMetadata(t, meta, "12345", "episode", "Episode Title")
		assertStringField(t, meta.GrandparentTitle, "Show Name", "GrandparentTitle")
		assertIntField(t, meta.Index, 5, "Index")
		assertIntField(t, meta.ParentIndex, 1, "ParentIndex")
	})

	t.Run("movie metadata (no parent fields)", func(t *testing.T) {
		jsonData := `{
			"ratingKey": "98765",
			"type": "movie",
			"title": "Movie Title",
			"viewedAt": 1732483200,
			"duration": 7200000,
			"year": 2024
		}`

		var meta PlexMetadata
		err := json.Unmarshal([]byte(jsonData), &meta)
		assertNoError(t, err, "JSON unmarshal")

		assertPlexMetadata(t, meta, "98765", "movie", "Movie Title")
		assertStringField(t, meta.GrandparentRatingKey, "", "GrandparentRatingKey")
	})
}

// Benchmark tests
func BenchmarkPlexClient_GetHistoryAll_Parse(b *testing.B) {
	// Generate sample response with 1000 items
	metadata := make([]PlexMetadata, 1000)
	for i := range metadata {
		metadata[i] = PlexMetadata{
			RatingKey:  string(rune(i)),
			Title:      "Test Movie",
			Type:       "movie",
			ViewedAt:   1732483200,
			Duration:   7200000,
			ViewOffset: 3600000,
		}
	}

	jsonData, _ := json.Marshal(PlexHistoryResponse{
		MediaContainer: PlexMediaContainer{Size: 1000, Metadata: metadata},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var resp PlexHistoryResponse
		if err := json.Unmarshal(jsonData, &resp); err != nil {
			b.Fatal(err)
		}
	}
}
