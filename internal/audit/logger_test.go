// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package audit

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"
)

func TestLogger_Log(t *testing.T) {
	store := NewMemoryStore(100)
	config := &Config{
		Enabled:     true,
		LogLevel:    SeverityInfo,
		LogToStdout: false,
		BufferSize:  10,
	}
	logger := NewLogger(store, config)
	defer logger.Close()

	event := &Event{
		Type:        EventTypeAuthSuccess,
		Severity:    SeverityInfo,
		Outcome:     OutcomeSuccess,
		Actor:       Actor{ID: "user1", Type: "user", Name: "testuser"},
		Source:      Source{IPAddress: "192.168.1.1"},
		Action:      "login",
		Description: "User logged in successfully",
	}

	logger.Log(event)

	// Wait for async write
	time.Sleep(100 * time.Millisecond)

	if store.Len() != 1 {
		t.Errorf("expected 1 event in store, got %d", store.Len())
	}

	// Query the event
	ctx := context.Background()
	events, err := store.Query(ctx, QueryFilter{Limit: 10})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type != EventTypeAuthSuccess {
		t.Errorf("expected type %s, got %s", EventTypeAuthSuccess, events[0].Type)
	}
	if events[0].Actor.ID != "user1" {
		t.Errorf("expected actor ID user1, got %s", events[0].Actor.ID)
	}
}

func TestLogger_Disabled(t *testing.T) {
	store := NewMemoryStore(100)
	config := &Config{
		Enabled:    false, // Disabled
		BufferSize: 10,
	}
	logger := NewLogger(store, config)
	defer logger.Close()

	event := &Event{
		Type:     EventTypeAuthSuccess,
		Severity: SeverityInfo,
	}

	logger.Log(event)
	time.Sleep(100 * time.Millisecond)

	if store.Len() != 0 {
		t.Error("disabled logger should not log events")
	}
}

func TestLogger_SeverityFiltering(t *testing.T) {
	store := NewMemoryStore(100)
	config := &Config{
		Enabled:      true,
		LogLevel:     SeverityWarning, // Only warning and above
		IncludeDebug: false,
		BufferSize:   10,
	}
	logger := NewLogger(store, config)
	defer logger.Close()

	// Info event (should be filtered)
	logger.Log(&Event{Type: EventTypeAuthSuccess, Severity: SeverityInfo})
	// Warning event (should be logged)
	logger.Log(&Event{Type: EventTypeAuthFailure, Severity: SeverityWarning})
	// Critical event (should be logged)
	logger.Log(&Event{Type: EventTypeAuthLockout, Severity: SeverityCritical})

	time.Sleep(100 * time.Millisecond)

	if store.Len() != 2 {
		t.Errorf("expected 2 events (warning + critical), got %d", store.Len())
	}
}

func TestLogger_DebugFiltering(t *testing.T) {
	store := NewMemoryStore(100)
	config := &Config{
		Enabled:      true,
		LogLevel:     SeverityDebug,
		IncludeDebug: false, // Debug excluded
		BufferSize:   10,
	}
	logger := NewLogger(store, config)
	defer logger.Close()

	logger.Log(&Event{Type: EventTypeAdminAction, Severity: SeverityDebug})
	time.Sleep(100 * time.Millisecond)

	if store.Len() != 0 {
		t.Error("debug events should be filtered when IncludeDebug is false")
	}

	// Enable debug
	logger.mu.Lock()
	logger.config.IncludeDebug = true
	logger.mu.Unlock()

	logger.Log(&Event{Type: EventTypeAdminAction, Severity: SeverityDebug})
	time.Sleep(100 * time.Millisecond)

	if store.Len() != 1 {
		t.Error("debug events should be logged when IncludeDebug is true")
	}
}

func TestLogger_AutoGenerateID(t *testing.T) {
	store := NewMemoryStore(100)
	config := &Config{
		Enabled:    true,
		LogLevel:   SeverityInfo,
		BufferSize: 10,
	}
	logger := NewLogger(store, config)
	defer logger.Close()

	event := &Event{
		Type:     EventTypeAuthSuccess,
		Severity: SeverityInfo,
		// ID not set
	}

	logger.Log(event)
	time.Sleep(100 * time.Millisecond)

	ctx := context.Background()
	events, _ := store.Query(ctx, QueryFilter{Limit: 1})
	if len(events) != 1 {
		t.Fatal("expected 1 event")
	}

	if events[0].ID == "" {
		t.Error("event ID should be auto-generated")
	}
}

func TestLogger_AutoSetTimestamp(t *testing.T) {
	store := NewMemoryStore(100)
	config := &Config{
		Enabled:    true,
		LogLevel:   SeverityInfo,
		BufferSize: 10,
	}
	logger := NewLogger(store, config)
	defer logger.Close()

	event := &Event{
		Type:     EventTypeAuthSuccess,
		Severity: SeverityInfo,
		// Timestamp not set
	}

	before := time.Now()
	logger.Log(event)
	time.Sleep(100 * time.Millisecond)

	ctx := context.Background()
	events, _ := store.Query(ctx, QueryFilter{Limit: 1})
	if len(events) != 1 {
		t.Fatal("expected 1 event")
	}

	if events[0].Timestamp.IsZero() {
		t.Error("timestamp should be auto-set")
	}
	if events[0].Timestamp.Before(before) {
		t.Error("timestamp should be recent")
	}
}

func TestLogger_HelperMethods(t *testing.T) {
	store := NewMemoryStore(100)
	config := &Config{
		Enabled:    true,
		LogLevel:   SeverityInfo,
		BufferSize: 10,
	}
	logger := NewLogger(store, config)
	defer logger.Close()

	ctx := context.Background()
	actor := Actor{ID: "user1", Type: "user", Name: "testuser"}
	source := Source{IPAddress: "192.168.1.1"}

	// Test LogAuthSuccess
	logger.LogAuthSuccess(ctx, actor, source, "jwt")
	time.Sleep(50 * time.Millisecond)

	// Test LogAuthFailure
	logger.LogAuthFailure(ctx, "user2", "baduser", source, "invalid password")
	time.Sleep(50 * time.Millisecond)

	// Test LogAuthLockout
	logger.LogAuthLockout(ctx, "user2", "baduser", source, 15*time.Minute, 5)
	time.Sleep(50 * time.Millisecond)

	// Test LogLogout
	logger.LogLogout(ctx, actor, source, "session123")
	time.Sleep(50 * time.Millisecond)

	// Test LogAuthzDenied
	logger.LogAuthzDenied(ctx, actor, source, "/api/admin", "delete")
	time.Sleep(50 * time.Millisecond)

	// Test LogDetectionAlert
	logger.LogDetectionAlert(ctx, "impossible_travel", "Impossible Travel Alert", 1, "testuser", SeverityCritical)
	time.Sleep(50 * time.Millisecond)

	// Test LogConfigChange
	logger.LogConfigChange(ctx, actor, source, "max_streams", "3", "5")
	time.Sleep(50 * time.Millisecond)

	// Verify all events were logged
	if store.Len() < 7 {
		t.Errorf("expected at least 7 events, got %d", store.Len())
	}
}

func TestMemoryStore_Query(t *testing.T) {
	store := NewMemoryStore(100)
	ctx := context.Background()

	// Add test events
	events := []Event{
		{ID: "1", Type: EventTypeAuthSuccess, Severity: SeverityInfo, Outcome: OutcomeSuccess,
			Actor: Actor{ID: "user1"}, Source: Source{IPAddress: "192.168.1.1"}, Timestamp: time.Now().Add(-3 * time.Hour)},
		{ID: "2", Type: EventTypeAuthFailure, Severity: SeverityWarning, Outcome: OutcomeFailure,
			Actor: Actor{ID: "user2"}, Source: Source{IPAddress: "192.168.1.2"}, Timestamp: time.Now().Add(-2 * time.Hour)},
		{ID: "3", Type: EventTypeAuthLockout, Severity: SeverityCritical, Outcome: OutcomeSuccess,
			Actor: Actor{ID: "user2"}, Source: Source{IPAddress: "192.168.1.2"}, Timestamp: time.Now().Add(-1 * time.Hour)},
	}

	for _, e := range events {
		store.Save(ctx, &e)
	}

	// Query by type
	results, err := store.Query(ctx, QueryFilter{Types: []EventType{EventTypeAuthSuccess}})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 event of type auth.success, got %d", len(results))
	}

	// Query by severity
	results, _ = store.Query(ctx, QueryFilter{Severities: []Severity{SeverityWarning, SeverityCritical}})
	if len(results) != 2 {
		t.Errorf("expected 2 events (warning + critical), got %d", len(results))
	}

	// Query by actor
	results, _ = store.Query(ctx, QueryFilter{ActorID: "user2"})
	if len(results) != 2 {
		t.Errorf("expected 2 events for user2, got %d", len(results))
	}

	// Query by source IP
	results, _ = store.Query(ctx, QueryFilter{SourceIP: "192.168.1.1"})
	if len(results) != 1 {
		t.Errorf("expected 1 event from 192.168.1.1, got %d", len(results))
	}

	// Query with limit
	results, _ = store.Query(ctx, QueryFilter{Limit: 2})
	if len(results) != 2 {
		t.Errorf("expected 2 events with limit, got %d", len(results))
	}
}

func TestMemoryStore_TimeRangeQuery(t *testing.T) {
	store := NewMemoryStore(100)
	ctx := context.Background()

	now := time.Now()
	events := []Event{
		{ID: "1", Type: EventTypeAuthSuccess, Timestamp: now.Add(-3 * time.Hour)},
		{ID: "2", Type: EventTypeAuthSuccess, Timestamp: now.Add(-2 * time.Hour)},
		{ID: "3", Type: EventTypeAuthSuccess, Timestamp: now.Add(-1 * time.Hour)},
	}

	for _, e := range events {
		store.Save(ctx, &e)
	}

	// Query last 90 minutes
	startTime := now.Add(-90 * time.Minute)
	results, _ := store.Query(ctx, QueryFilter{StartTime: &startTime})
	if len(results) != 1 {
		t.Errorf("expected 1 event in last 90 minutes, got %d", len(results))
	}

	// Query between 2.5 and 1.5 hours ago
	endTime := now.Add(-90 * time.Minute)
	startTime = now.Add(-150 * time.Minute)
	results, _ = store.Query(ctx, QueryFilter{StartTime: &startTime, EndTime: &endTime})
	if len(results) != 1 {
		t.Errorf("expected 1 event between 2.5h and 1.5h ago, got %d", len(results))
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore(100)
	ctx := context.Background()

	now := time.Now()
	events := []Event{
		{ID: "1", Timestamp: now.Add(-48 * time.Hour)},
		{ID: "2", Timestamp: now.Add(-24 * time.Hour)},
		{ID: "3", Timestamp: now.Add(-1 * time.Hour)},
	}

	for _, e := range events {
		store.Save(ctx, &e)
	}

	// Delete events older than 36 hours
	cutoff := now.Add(-36 * time.Hour)
	deleted, err := store.Delete(ctx, cutoff)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}
	if store.Len() != 2 {
		t.Errorf("expected 2 remaining events, got %d", store.Len())
	}
}

func TestMemoryStore_Count(t *testing.T) {
	store := NewMemoryStore(100)
	ctx := context.Background()

	events := []Event{
		{ID: "1", Type: EventTypeAuthSuccess},
		{ID: "2", Type: EventTypeAuthSuccess},
		{ID: "3", Type: EventTypeAuthFailure},
	}

	for _, e := range events {
		store.Save(ctx, &e)
	}

	// Count all
	count, _ := store.Count(ctx, QueryFilter{})
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}

	// Count by type
	count, _ = store.Count(ctx, QueryFilter{Types: []EventType{EventTypeAuthSuccess}})
	if count != 2 {
		t.Errorf("expected count 2 for auth.success, got %d", count)
	}
}

func TestMemoryStore_GetStats(t *testing.T) {
	store := NewMemoryStore(100)
	ctx := context.Background()

	now := time.Now()
	events := []Event{
		{ID: "1", Type: EventTypeAuthSuccess, Severity: SeverityInfo, Outcome: OutcomeSuccess, Timestamp: now.Add(-2 * time.Hour)},
		{ID: "2", Type: EventTypeAuthFailure, Severity: SeverityWarning, Outcome: OutcomeFailure, Timestamp: now.Add(-1 * time.Hour)},
		{ID: "3", Type: EventTypeAuthSuccess, Severity: SeverityInfo, Outcome: OutcomeSuccess, Timestamp: now},
	}

	for _, e := range events {
		store.Save(ctx, &e)
	}

	stats, err := store.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalEvents != 3 {
		t.Errorf("expected total 3, got %d", stats.TotalEvents)
	}
	if stats.EventsByType[string(EventTypeAuthSuccess)] != 2 {
		t.Errorf("expected 2 auth.success events")
	}
	if stats.EventsBySeverity[string(SeverityInfo)] != 2 {
		t.Errorf("expected 2 info events")
	}
	if stats.EventsByOutcome[string(OutcomeSuccess)] != 2 {
		t.Errorf("expected 2 success outcomes")
	}
}

func TestCEFExporter(t *testing.T) {
	exporter := NewCEFExporter()

	events := []Event{
		{
			ID:          "test1",
			Type:        EventTypeAuthFailure,
			Severity:    SeverityWarning,
			Outcome:     OutcomeFailure,
			Actor:       Actor{ID: "user1", Name: "testuser"},
			Source:      Source{IPAddress: "192.168.1.1"},
			Action:      "authenticate",
			Description: "Authentication failed",
			Timestamp:   time.Now(),
			RequestID:   "req123",
		},
	}

	data, err := exporter.Export(events)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	cefLine := string(data)

	// Check CEF format
	if !startsWith(cefLine, "CEF:0|") {
		t.Error("CEF line should start with 'CEF:0|'")
	}
	if !contains(cefLine, "Cartographus") {
		t.Error("CEF line should contain vendor name")
	}
	if !contains(cefLine, "auth.failure") {
		t.Error("CEF line should contain event type")
	}
	if !contains(cefLine, "suser=testuser") {
		t.Error("CEF line should contain source user")
	}
	if !contains(cefLine, "src=192.168.1.1") {
		t.Error("CEF line should contain source IP")
	}
}

func TestSourceFromRequest(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		headers       map[string]string
		host          string
		userAgent     string
		expectedIP    string
		expectedHost  string
		expectedAgent string
	}{
		{
			name:          "basic request with RemoteAddr",
			remoteAddr:    "192.168.1.100:54321",
			headers:       nil,
			host:          "api.example.com",
			userAgent:     "Mozilla/5.0",
			expectedIP:    "192.168.1.100:54321",
			expectedHost:  "api.example.com",
			expectedAgent: "Mozilla/5.0",
		},
		{
			name:       "X-Forwarded-For takes precedence",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.50",
			},
			host:          "api.example.com",
			userAgent:     "curl/7.68.0",
			expectedIP:    "203.0.113.50",
			expectedHost:  "api.example.com",
			expectedAgent: "curl/7.68.0",
		},
		{
			name:       "X-Real-IP when no X-Forwarded-For",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Real-IP": "198.51.100.25",
			},
			host:          "localhost:3857",
			userAgent:     "Go-http-client/1.1",
			expectedIP:    "198.51.100.25",
			expectedHost:  "localhost:3857",
			expectedAgent: "Go-http-client/1.1",
		},
		{
			name:       "X-Forwarded-For takes precedence over X-Real-IP",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.50",
				"X-Real-IP":       "198.51.100.25",
			},
			host:          "api.example.com",
			userAgent:     "TestClient/1.0",
			expectedIP:    "203.0.113.50",
			expectedHost:  "api.example.com",
			expectedAgent: "TestClient/1.0",
		},
		{
			name:          "empty user agent",
			remoteAddr:    "127.0.0.1:8080",
			headers:       nil,
			host:          "localhost",
			userAgent:     "",
			expectedIP:    "127.0.0.1:8080",
			expectedHost:  "localhost",
			expectedAgent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://"+tt.host+"/api/v1/test", nil)
			req.RemoteAddr = tt.remoteAddr
			req.Host = tt.host
			if tt.userAgent != "" {
				req.Header.Set("User-Agent", tt.userAgent)
			}
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			source := SourceFromRequest(req)

			if source.IPAddress != tt.expectedIP {
				t.Errorf("IPAddress = %q, want %q", source.IPAddress, tt.expectedIP)
			}
			if source.Hostname != tt.expectedHost {
				t.Errorf("Hostname = %q, want %q", source.Hostname, tt.expectedHost)
			}
			if source.UserAgent != tt.expectedAgent {
				t.Errorf("UserAgent = %q, want %q", source.UserAgent, tt.expectedAgent)
			}
		})
	}
}

func TestActorFromUser(t *testing.T) {
	actor := ActorFromUser("user123", "testuser", []string{"admin", "editor"}, "jwt", "sess456")

	if actor.ID != "user123" {
		t.Errorf("expected ID user123, got %s", actor.ID)
	}
	if actor.Name != "testuser" {
		t.Errorf("expected name testuser, got %s", actor.Name)
	}
	if actor.Type != "user" {
		t.Errorf("expected type user, got %s", actor.Type)
	}
	if len(actor.Roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(actor.Roles))
	}
	if actor.AuthMethod != "jwt" {
		t.Errorf("expected auth method jwt, got %s", actor.AuthMethod)
	}
	if actor.SessionID != "sess456" {
		t.Errorf("expected session ID sess456, got %s", actor.SessionID)
	}
}

func TestSystemActor(t *testing.T) {
	actor := SystemActor()

	if actor.ID != "system" {
		t.Errorf("expected ID system, got %s", actor.ID)
	}
	if actor.Type != "system" {
		t.Errorf("expected type system, got %s", actor.Type)
	}
}

func TestMustJSON(t *testing.T) {
	result := mustJSON(map[string]string{"key": "value"})

	var parsed map[string]string
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if parsed["key"] != "value" {
		t.Errorf("expected value 'value', got %s", parsed["key"])
	}
}

// TestCEFExporter_EmptyEvents tests export with empty event list.
func TestCEFExporter_EmptyEvents(t *testing.T) {
	exporter := NewCEFExporter()

	data, err := exporter.Export([]Event{})
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	if string(data) != "" {
		t.Errorf("expected empty output for empty events, got %q", string(data))
	}
}

// TestCEFExporter_SpecialCharacterEscaping tests CEF escaping of special characters.
func TestCEFExporter_SpecialCharacterEscaping(t *testing.T) {
	exporter := NewCEFExporter()

	tests := []struct {
		name        string
		input       string
		shouldFind  string
		description string
	}{
		{
			name:        "pipe character",
			input:       "test|pipe",
			shouldFind:  "test\\|pipe",
			description: "Pipes must be escaped with backslash",
		},
		{
			name:        "equals character",
			input:       "key=value",
			shouldFind:  "key\\=value",
			description: "Equals signs must be escaped in extensions",
		},
		{
			name:        "backslash character",
			input:       "path\\file",
			shouldFind:  "path\\\\file",
			description: "Backslashes must be escaped",
		},
		{
			name:        "newline character",
			input:       "line1\nline2",
			shouldFind:  "line1 line2",
			description: "Newlines should be replaced with spaces",
		},
		{
			name:        "carriage return",
			input:       "text\rwith\rCR",
			shouldFind:  "textwithCR",
			description: "Carriage returns should be removed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := []Event{
				{
					ID:          "test-escape",
					Type:        EventTypeAuthSuccess,
					Severity:    SeverityInfo,
					Outcome:     OutcomeSuccess,
					Description: tt.input,
					Actor:       Actor{ID: "user1", Name: tt.input},
					Source:      Source{IPAddress: "192.168.1.1"},
					Action:      "test",
					Timestamp:   time.Now(),
				},
			}

			data, err := exporter.Export(events)
			if err != nil {
				t.Fatalf("export failed: %v", err)
			}

			cefLine := string(data)
			if !contains(cefLine, tt.shouldFind) {
				t.Errorf("%s: expected to find %q in CEF output, got: %s",
					tt.description, tt.shouldFind, cefLine)
			}
		})
	}
}

// TestCEFExporter_NilTarget tests export with nil target.
func TestCEFExporter_NilTarget(t *testing.T) {
	exporter := NewCEFExporter()

	events := []Event{
		{
			ID:          "no-target",
			Type:        EventTypeAuthSuccess,
			Severity:    SeverityInfo,
			Outcome:     OutcomeSuccess,
			Actor:       Actor{ID: "user1", Name: "testuser"},
			Source:      Source{IPAddress: "192.168.1.1"},
			Target:      nil, // Explicitly nil
			Action:      "login",
			Description: "User logged in",
			Timestamp:   time.Now(),
		},
	}

	data, err := exporter.Export(events)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	cefLine := string(data)
	// Should NOT contain duser or duid for nil target
	if contains(cefLine, "duser=") || contains(cefLine, "duid=") {
		t.Error("CEF line should not contain target fields when target is nil")
	}
}

// TestCEFExporter_EmptyFields tests export with empty/zero fields.
func TestCEFExporter_EmptyFields(t *testing.T) {
	exporter := NewCEFExporter()

	events := []Event{
		{
			ID:          "minimal",
			Type:        EventTypeAuthSuccess,
			Severity:    SeverityInfo,
			Outcome:     OutcomeSuccess,
			Actor:       Actor{},  // Empty actor
			Source:      Source{}, // Empty source
			Action:      "",
			Description: "",
			Timestamp:   time.Now(),
		},
	}

	data, err := exporter.Export(events)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	cefLine := string(data)

	// Should still have basic CEF structure
	if !startsWith(cefLine, "CEF:0|") {
		t.Error("CEF line should start with 'CEF:0|' even with empty fields")
	}

	// Should have timestamp
	if !contains(cefLine, "rt=") {
		t.Error("CEF line should contain timestamp (rt=)")
	}
}

// TestCEFExporter_MultipleEvents tests export of multiple events.
func TestCEFExporter_MultipleEvents(t *testing.T) {
	exporter := NewCEFExporter()

	events := []Event{
		{
			ID:          "event1",
			Type:        EventTypeAuthSuccess,
			Severity:    SeverityInfo,
			Description: "First event",
			Actor:       Actor{ID: "user1", Name: "first"},
			Source:      Source{IPAddress: "10.0.0.1"},
			Action:      "login",
			Timestamp:   time.Now(),
		},
		{
			ID:          "event2",
			Type:        EventTypeAuthFailure,
			Severity:    SeverityWarning,
			Description: "Second event",
			Actor:       Actor{ID: "user2", Name: "second"},
			Source:      Source{IPAddress: "10.0.0.2"},
			Action:      "login",
			Timestamp:   time.Now(),
		},
		{
			ID:          "event3",
			Type:        EventTypeAuthLockout,
			Severity:    SeverityCritical,
			Description: "Third event",
			Actor:       Actor{ID: "user3", Name: "third"},
			Source:      Source{IPAddress: "10.0.0.3"},
			Action:      "lockout",
			Timestamp:   time.Now(),
		},
	}

	data, err := exporter.Export(events)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	lines := splitLines(string(data))
	if len(lines) != 3 {
		t.Errorf("expected 3 CEF lines for 3 events, got %d", len(lines))
	}

	// Verify each line is valid CEF
	for i, line := range lines {
		if !startsWith(line, "CEF:0|") {
			t.Errorf("line %d should start with 'CEF:0|', got: %s", i+1, line)
		}
	}

	// Verify each event's unique data appears in the output
	if !contains(string(data), "src=10.0.0.1") {
		t.Error("missing first event's source IP")
	}
	if !contains(string(data), "src=10.0.0.2") {
		t.Error("missing second event's source IP")
	}
	if !contains(string(data), "src=10.0.0.3") {
		t.Error("missing third event's source IP")
	}
}

// TestCEFExporter_AllSeverityLevels tests CEF severity mapping for all levels.
func TestCEFExporter_AllSeverityLevels(t *testing.T) {
	exporter := NewCEFExporter()

	tests := []struct {
		severity Severity
		expected int
	}{
		{SeverityDebug, 0},
		{SeverityInfo, 3},
		{SeverityWarning, 5},
		{SeverityError, 7},
		{SeverityCritical, 10},
		{Severity("unknown"), 0}, // Unknown should default to 0
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			got := exporter.cefSeverity(tt.severity)
			if got != tt.expected {
				t.Errorf("cefSeverity(%s) = %d, want %d", tt.severity, got, tt.expected)
			}
		})
	}
}

// TestCEFExporter_WithTarget tests CEF export with target information.
func TestCEFExporter_WithTarget(t *testing.T) {
	exporter := NewCEFExporter()

	events := []Event{
		{
			ID:       "with-target",
			Type:     EventTypeAuthzDenied,
			Severity: SeverityWarning,
			Outcome:  OutcomeFailure,
			Actor:    Actor{ID: "attacker", Name: "baduser"},
			Source:   Source{IPAddress: "10.20.30.40"},
			Target: &Target{
				ID:   "resource123",
				Type: "file",
				Name: "secret.txt",
			},
			Action:      "access",
			Description: "Unauthorized access attempt",
			Timestamp:   time.Now(),
		},
	}

	data, err := exporter.Export(events)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	cefLine := string(data)

	if !contains(cefLine, "duser=secret.txt") {
		t.Error("CEF line should contain target name as duser")
	}
	if !contains(cefLine, "duid=resource123") {
		t.Error("CEF line should contain target ID as duid")
	}
}

// TestCEFExporter_RequestID tests CEF export with request ID.
func TestCEFExporter_RequestID(t *testing.T) {
	exporter := NewCEFExporter()

	events := []Event{
		{
			ID:          "with-reqid",
			Type:        EventTypeAuthSuccess,
			Severity:    SeverityInfo,
			Outcome:     OutcomeSuccess,
			Actor:       Actor{ID: "user1"},
			Source:      Source{IPAddress: "192.168.1.1"},
			Action:      "login",
			Description: "Login with request ID",
			RequestID:   "req-abc-123-xyz",
			Timestamp:   time.Now(),
		},
	}

	data, err := exporter.Export(events)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	cefLine := string(data)
	if !contains(cefLine, "externalId=req-abc-123-xyz") {
		t.Error("CEF line should contain request ID as externalId")
	}
}

// TestCEFExporter_CustomVendorProduct tests custom vendor/product configuration.
func TestCEFExporter_CustomVendorProduct(t *testing.T) {
	exporter := &CEFExporter{
		DeviceVendor:  "CustomVendor",
		DeviceProduct: "CustomProduct",
		DeviceVersion: "2.0.0",
	}

	events := []Event{
		{
			ID:          "custom-vendor",
			Type:        EventTypeAuthSuccess,
			Severity:    SeverityInfo,
			Outcome:     OutcomeSuccess,
			Actor:       Actor{ID: "user1"},
			Source:      Source{IPAddress: "192.168.1.1"},
			Action:      "test",
			Description: "Test with custom vendor",
			Timestamp:   time.Now(),
		},
	}

	data, err := exporter.Export(events)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	cefLine := string(data)

	if !contains(cefLine, "CustomVendor") {
		t.Error("CEF line should contain custom vendor name")
	}
	if !contains(cefLine, "CustomProduct") {
		t.Error("CEF line should contain custom product name")
	}
	if !contains(cefLine, "2.0.0") {
		t.Error("CEF line should contain custom version")
	}
}

// TestCEFExporter_VendorWithSpecialChars tests vendor name with special characters.
func TestCEFExporter_VendorWithSpecialChars(t *testing.T) {
	exporter := &CEFExporter{
		DeviceVendor:  "Vendor|With|Pipes",
		DeviceProduct: "Product=With=Equals",
		DeviceVersion: "1.0\\beta",
	}

	events := []Event{
		{
			ID:          "special-vendor",
			Type:        EventTypeAuthSuccess,
			Severity:    SeverityInfo,
			Description: "Test",
			Timestamp:   time.Now(),
		},
	}

	data, err := exporter.Export(events)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	cefLine := string(data)

	// Verify special chars are escaped
	if !contains(cefLine, "Vendor\\|With\\|Pipes") {
		t.Error("Vendor name pipes should be escaped")
	}
	if !contains(cefLine, "Product\\=With\\=Equals") {
		t.Error("Product name equals should be escaped")
	}
}

// TestCEFExporter_Timestamp tests that timestamp is exported correctly.
func TestCEFExporter_Timestamp(t *testing.T) {
	exporter := NewCEFExporter()

	// Use a specific timestamp
	ts := time.Date(2024, 6, 15, 10, 30, 45, 123000000, time.UTC)
	expectedMillis := ts.UnixMilli()

	events := []Event{
		{
			ID:          "timestamp-test",
			Type:        EventTypeAuthSuccess,
			Severity:    SeverityInfo,
			Description: "Timestamp test",
			Timestamp:   ts,
		},
	}

	data, err := exporter.Export(events)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	cefLine := string(data)
	expectedRT := "rt=" + itoa(expectedMillis)
	if !contains(cefLine, expectedRT) {
		t.Errorf("CEF line should contain rt=%d, got: %s", expectedMillis, cefLine)
	}
}

// TestJSONExporter tests the JSON exporter.
func TestJSONExporter(t *testing.T) {
	exporter := &JSONExporter{}

	events := []Event{
		{
			ID:          "json-test",
			Type:        EventTypeAuthSuccess,
			Severity:    SeverityInfo,
			Outcome:     OutcomeSuccess,
			Actor:       Actor{ID: "user1", Name: "testuser"},
			Description: "Test event",
			Timestamp:   time.Now(),
		},
	}

	data, err := exporter.Export(events)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	// Should be valid JSON
	var parsed []Event
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("exported JSON is invalid: %v", err)
	}

	if len(parsed) != 1 {
		t.Fatalf("expected 1 event in JSON, got %d", len(parsed))
	}

	if parsed[0].ID != "json-test" {
		t.Errorf("expected ID 'json-test', got %s", parsed[0].ID)
	}
}

// TestJSONExporter_EmptyEvents tests JSON export with empty event list.
func TestJSONExporter_EmptyEvents(t *testing.T) {
	exporter := &JSONExporter{}

	data, err := exporter.Export([]Event{})
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	// Should be empty array
	if string(data) != "[]" {
		t.Errorf("expected '[]' for empty events, got %s", string(data))
	}
}

// Helper functions
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func itoa(i int64) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		pos--
		buf[pos] = byte(i%10) + '0'
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
