// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package audit

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
)

// MemoryStore implements Store using in-memory storage.
// Suitable for development and testing. Data is lost on restart.
type MemoryStore struct {
	events []Event
	mu     sync.RWMutex
	maxLen int
}

// NewMemoryStore creates a new in-memory audit store.
func NewMemoryStore(maxLen int) *MemoryStore {
	if maxLen <= 0 {
		maxLen = 10000
	}
	return &MemoryStore{
		events: make([]Event, 0, maxLen),
		maxLen: maxLen,
	}
}

// Save persists an audit event.
func (s *MemoryStore) Save(ctx context.Context, event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Enforce max length by removing oldest events
	if len(s.events) >= s.maxLen {
		// Remove oldest 10%
		removeCount := s.maxLen / 10
		s.events = s.events[removeCount:]
	}

	s.events = append(s.events, *event)
	return nil
}

// Get retrieves an event by ID.
func (s *MemoryStore) Get(ctx context.Context, id string) (*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.events {
		if s.events[i].ID == id {
			event := s.events[i]
			return &event, nil
		}
	}

	return nil, fmt.Errorf("event not found: %s", id)
}

// Query retrieves events matching the filter.
func (s *MemoryStore) Query(ctx context.Context, filter QueryFilter) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []Event

	for i := len(s.events) - 1; i >= 0; i-- { // Iterate in reverse for recent-first
		event := s.events[i]

		if !s.matchesFilter(&event, &filter) {
			continue
		}

		results = append(results, event)

		if filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}

	return results, nil
}

// matchesFilter returns true if the event matches all filter criteria.
//
//nolint:gocyclo // complexity inherent to multi-criteria filter matching
func (s *MemoryStore) matchesFilter(event *Event, filter *QueryFilter) bool {
	// Type filter
	if len(filter.Types) > 0 {
		found := false
		for _, t := range filter.Types {
			if event.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Severity filter
	if len(filter.Severities) > 0 {
		found := false
		for _, sev := range filter.Severities {
			if event.Severity == sev {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Outcome filter
	if len(filter.Outcomes) > 0 {
		found := false
		for _, o := range filter.Outcomes {
			if event.Outcome == o {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Actor filters
	if filter.ActorID != "" && event.Actor.ID != filter.ActorID {
		return false
	}
	if filter.ActorType != "" && event.Actor.Type != filter.ActorType {
		return false
	}

	// Target filters
	if filter.TargetID != "" {
		if event.Target == nil || event.Target.ID != filter.TargetID {
			return false
		}
	}
	if filter.TargetType != "" {
		if event.Target == nil || event.Target.Type != filter.TargetType {
			return false
		}
	}

	// Source IP filter
	if filter.SourceIP != "" && event.Source.IPAddress != filter.SourceIP {
		return false
	}

	// Time range filter
	if filter.StartTime != nil && event.Timestamp.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && event.Timestamp.After(*filter.EndTime) {
		return false
	}

	// Correlation ID filter
	if filter.CorrelationID != "" && event.CorrelationID != filter.CorrelationID {
		return false
	}

	// Request ID filter
	if filter.RequestID != "" && event.RequestID != filter.RequestID {
		return false
	}

	// Text search
	if filter.SearchText != "" {
		searchLower := strings.ToLower(filter.SearchText)
		if !strings.Contains(strings.ToLower(event.Description), searchLower) &&
			!strings.Contains(strings.ToLower(event.Action), searchLower) {
			return false
		}
	}

	return true
}

// Count returns the number of events matching the filter.
func (s *MemoryStore) Count(ctx context.Context, filter QueryFilter) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int64
	for i := range s.events {
		if s.matchesFilter(&s.events[i], &filter) {
			count++
		}
	}

	return count, nil
}

// Delete removes events older than the given time.
func (s *MemoryStore) Delete(ctx context.Context, olderThan time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var kept []Event
	var deleted int64

	for idx := range s.events {
		if s.events[idx].Timestamp.Before(olderThan) {
			deleted++
		} else {
			kept = append(kept, s.events[idx])
		}
	}

	s.events = kept
	return deleted, nil
}

// Clear removes all events (for testing).
func (s *MemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = s.events[:0]
}

// Len returns the number of events in the store.
func (s *MemoryStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events)
}

// JSONExporter exports events in JSON format.
type JSONExporter struct{}

// Export exports events to JSON format.
func (e *JSONExporter) Export(events []Event) ([]byte, error) {
	return json.MarshalIndent(events, "", "  ")
}

// CEFExporter exports events in Common Event Format (for SIEM integration).
type CEFExporter struct {
	DeviceVendor  string
	DeviceProduct string
	DeviceVersion string
}

// NewCEFExporter creates a new CEF exporter with defaults.
func NewCEFExporter() *CEFExporter {
	return &CEFExporter{
		DeviceVendor:  "Cartographus",
		DeviceProduct: "MediaServerAnalytics",
		DeviceVersion: "1.0",
	}
}

// Export exports events to CEF format.
// CEF Format: CEF:Version|Device Vendor|Device Product|Device Version|Signature ID|Name|Severity|Extension
func (e *CEFExporter) Export(events []Event) ([]byte, error) {
	var lines []string

	for idx := range events {
		event := &events[idx]
		severity := e.cefSeverity(event.Severity)
		extension := e.buildExtension(event)

		line := fmt.Sprintf("CEF:0|%s|%s|%s|%s|%s|%d|%s",
			e.escape(e.DeviceVendor),
			e.escape(e.DeviceProduct),
			e.escape(e.DeviceVersion),
			e.escape(string(event.Type)),
			e.escape(event.Description),
			severity,
			extension,
		)

		lines = append(lines, line)
	}

	return []byte(strings.Join(lines, "\n")), nil
}

// cefSeverity maps our severity to CEF severity (0-10).
func (e *CEFExporter) cefSeverity(severity Severity) int {
	switch severity {
	case SeverityDebug:
		return 0
	case SeverityInfo:
		return 3
	case SeverityWarning:
		return 5
	case SeverityError:
		return 7
	case SeverityCritical:
		return 10
	default:
		return 0
	}
}

// buildExtension builds the CEF extension string.
func (e *CEFExporter) buildExtension(event *Event) string {
	var parts []string

	// Standard CEF extension fields
	parts = append(parts, fmt.Sprintf("rt=%d", event.Timestamp.UnixMilli()))

	if event.Actor.ID != "" {
		parts = append(parts, fmt.Sprintf("suser=%s", e.escape(event.Actor.Name)))
		parts = append(parts, fmt.Sprintf("suid=%s", e.escape(event.Actor.ID)))
	}

	if event.Source.IPAddress != "" {
		parts = append(parts, fmt.Sprintf("src=%s", e.escape(event.Source.IPAddress)))
	}

	if event.Target != nil {
		parts = append(parts, fmt.Sprintf("duser=%s", e.escape(event.Target.Name)))
		parts = append(parts, fmt.Sprintf("duid=%s", e.escape(event.Target.ID)))
	}

	parts = append(parts, fmt.Sprintf("act=%s", e.escape(event.Action)))
	parts = append(parts, fmt.Sprintf("outcome=%s", e.escape(string(event.Outcome))))

	if event.RequestID != "" {
		parts = append(parts, fmt.Sprintf("externalId=%s", e.escape(event.RequestID)))
	}

	return strings.Join(parts, " ")
}

// escape escapes special characters for CEF format.
func (e *CEFExporter) escape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "=", "\\=")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// Stats returns statistics about the audit store.
type Stats struct {
	TotalEvents      int64            `json:"total_events"`
	EventsByType     map[string]int64 `json:"events_by_type"`
	EventsBySeverity map[string]int64 `json:"events_by_severity"`
	EventsByOutcome  map[string]int64 `json:"events_by_outcome"`
	OldestEvent      *time.Time       `json:"oldest_event,omitempty"`
	NewestEvent      *time.Time       `json:"newest_event,omitempty"`
}

// GetStats returns statistics for the memory store.
func (s *MemoryStore) GetStats(ctx context.Context) (*Stats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &Stats{
		TotalEvents:      int64(len(s.events)),
		EventsByType:     make(map[string]int64),
		EventsBySeverity: make(map[string]int64),
		EventsByOutcome:  make(map[string]int64),
	}

	for idx := range s.events {
		event := &s.events[idx]
		stats.EventsByType[string(event.Type)]++
		stats.EventsBySeverity[string(event.Severity)]++
		stats.EventsByOutcome[string(event.Outcome)]++

		if stats.OldestEvent == nil || event.Timestamp.Before(*stats.OldestEvent) {
			t := event.Timestamp
			stats.OldestEvent = &t
		}
		if stats.NewestEvent == nil || event.Timestamp.After(*stats.NewestEvent) {
			t := event.Timestamp
			stats.NewestEvent = &t
		}
	}

	return stats, nil
}
