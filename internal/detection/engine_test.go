// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package detection

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// mockAlertStore implements AlertStore for testing
type mockAlertStore struct {
	alerts []Alert
	mu     sync.Mutex
}

func (m *mockAlertStore) SaveAlert(ctx context.Context, alert *Alert) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	alert.ID = int64(len(m.alerts) + 1)
	m.alerts = append(m.alerts, *alert)
	return nil
}

func (m *mockAlertStore) GetAlert(ctx context.Context, id int64) (*Alert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range m.alerts {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, nil
}

func (m *mockAlertStore) ListAlerts(ctx context.Context, filter AlertFilter) ([]Alert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.alerts, nil
}

func (m *mockAlertStore) AcknowledgeAlert(ctx context.Context, id int64, acknowledgedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.alerts {
		if m.alerts[i].ID == id {
			m.alerts[i].Acknowledged = true
			m.alerts[i].AcknowledgedBy = acknowledgedBy
			now := time.Now()
			m.alerts[i].AcknowledgedAt = &now
			return nil
		}
	}
	return nil
}

func (m *mockAlertStore) GetAlertCount(ctx context.Context, filter AlertFilter) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.alerts), nil
}

// mockTrustStore implements TrustStore for testing
type mockTrustStore struct {
	scores map[int]*TrustScore
	mu     sync.Mutex
}

func newMockTrustStore() *mockTrustStore {
	return &mockTrustStore{scores: make(map[int]*TrustScore)}
}

func (m *mockTrustStore) GetTrustScore(ctx context.Context, userID int) (*TrustScore, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if score, ok := m.scores[userID]; ok {
		return score, nil
	}
	return &TrustScore{UserID: userID, Score: 100}, nil
}

func (m *mockTrustStore) UpdateTrustScore(ctx context.Context, score *TrustScore) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scores[score.UserID] = score
	return nil
}

func (m *mockTrustStore) DecrementTrustScore(ctx context.Context, userID int, amount int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if score, ok := m.scores[userID]; ok {
		score.Score -= amount
		if score.Score < 0 {
			score.Score = 0
		}
		score.ViolationsCount++
		now := time.Now()
		score.LastViolationAt = &now
	} else {
		now := time.Now()
		m.scores[userID] = &TrustScore{
			UserID:          userID,
			Score:           100 - amount,
			ViolationsCount: 1,
			LastViolationAt: &now,
		}
	}
	return nil
}

func (m *mockTrustStore) RecoverTrustScores(ctx context.Context, amount int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, score := range m.scores {
		score.Score += amount
		if score.Score > 100 {
			score.Score = 100
		}
	}
	return nil
}

func (m *mockTrustStore) ListLowTrustUsers(ctx context.Context, threshold int) ([]TrustScore, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []TrustScore
	for _, score := range m.scores {
		if score.Score < threshold {
			result = append(result, *score)
		}
	}
	return result, nil
}

// mockBroadcaster implements AlertBroadcaster for testing
type mockBroadcaster struct {
	messages []interface{}
	mu       sync.Mutex
}

func (m *mockBroadcaster) BroadcastJSON(messageType string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, data)
}

func TestEngine_RegisterDetector(t *testing.T) {
	alertStore := &mockAlertStore{}
	trustStore := newMockTrustStore()
	eventHistory := &mockEventHistory{}
	broadcaster := &mockBroadcaster{}

	engine := NewEngine(alertStore, trustStore, eventHistory, broadcaster)
	defer engine.Close()

	detector := NewImpossibleTravelDetector(eventHistory)
	engine.RegisterDetector(detector)

	d, ok := engine.GetDetector(RuleTypeImpossibleTravel)
	if !ok {
		t.Error("detector not found after registration")
	}
	if d.Type() != RuleTypeImpossibleTravel {
		t.Errorf("wrong detector type: got %s, want %s", d.Type(), RuleTypeImpossibleTravel)
	}
}

func TestEngine_Process(t *testing.T) {
	eventHistory := &mockEventHistory{
		lastEvent: &DetectionEvent{
			UserID:    1,
			Latitude:  40.7128,
			Longitude: -74.0060, // NYC
			Timestamp: time.Now().Add(-30 * time.Minute),
		},
	}
	alertStore := &mockAlertStore{}
	trustStore := newMockTrustStore()
	broadcaster := &mockBroadcaster{}

	engine := NewEngine(alertStore, trustStore, eventHistory, broadcaster)
	defer engine.Close()

	// Register impossible travel detector
	detector := NewImpossibleTravelDetector(eventHistory)
	engine.RegisterDetector(detector)

	// Process event that should trigger alert
	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		Latitude:  51.5074,
		Longitude: -0.1278, // London
		Timestamp: time.Now(),
	}

	alerts, err := engine.Process(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(alerts) != 1 {
		t.Errorf("expected 1 alert, got %d", len(alerts))
	}

	// Verify alert was saved
	if len(alertStore.alerts) != 1 {
		t.Errorf("expected 1 saved alert, got %d", len(alertStore.alerts))
	}

	// Verify broadcast
	if len(broadcaster.messages) != 1 {
		t.Errorf("expected 1 broadcast, got %d", len(broadcaster.messages))
	}
}

func TestEngine_Process_Disabled(t *testing.T) {
	eventHistory := &mockEventHistory{
		lastEvent: &DetectionEvent{
			UserID:    1,
			Latitude:  40.7128,
			Longitude: -74.0060,
			Timestamp: time.Now().Add(-30 * time.Minute),
		},
	}
	alertStore := &mockAlertStore{}
	trustStore := newMockTrustStore()
	broadcaster := &mockBroadcaster{}

	engine := NewEngine(alertStore, trustStore, eventHistory, broadcaster)
	defer engine.Close()
	engine.SetEnabled(false)

	detector := NewImpossibleTravelDetector(eventHistory)
	engine.RegisterDetector(detector)

	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		Latitude:  51.5074,
		Longitude: -0.1278,
		Timestamp: time.Now(),
	}

	alerts, err := engine.Process(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if alerts != nil {
		t.Error("expected nil alerts when engine is disabled")
	}
}

func TestEngine_Process_MultipleDetectors(t *testing.T) {
	eventHistory := &mockEventHistory{
		lastEvent: &DetectionEvent{
			UserID:    1,
			Latitude:  40.7128,
			Longitude: -74.0060,
			Timestamp: time.Now().Add(-30 * time.Minute),
		},
		activeStreams: []DetectionEvent{
			{SessionKey: "session1", UserID: 1},
			{SessionKey: "session2", UserID: 1},
			{SessionKey: "session3", UserID: 1},
		},
	}
	alertStore := &mockAlertStore{}
	trustStore := newMockTrustStore()
	broadcaster := &mockBroadcaster{}

	engine := NewEngine(alertStore, trustStore, eventHistory, broadcaster)
	defer engine.Close()

	// Register multiple detectors
	engine.RegisterDetector(NewImpossibleTravelDetector(eventHistory))
	engine.RegisterDetector(NewConcurrentStreamsDetector(eventHistory))

	// Process event that should trigger multiple alerts
	event := &DetectionEvent{
		UserID:     1,
		Username:   "testuser",
		SessionKey: "session4",
		EventType:  "start",
		Latitude:   51.5074,
		Longitude:  -0.1278, // London (impossible travel)
		Timestamp:  time.Now(),
	}

	alerts, err := engine.Process(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have alerts from both detectors
	if len(alerts) < 1 {
		t.Errorf("expected at least 1 alert, got %d", len(alerts))
	}
}

func TestEngine_Metrics(t *testing.T) {
	alertStore := &mockAlertStore{}
	trustStore := newMockTrustStore()
	eventHistory := &mockEventHistory{}
	broadcaster := &mockBroadcaster{}

	engine := NewEngine(alertStore, trustStore, eventHistory, broadcaster)
	defer engine.Close()

	detector := NewImpossibleTravelDetector(eventHistory)
	engine.RegisterDetector(detector)

	// Process a few events
	for i := 0; i < 5; i++ {
		event := &DetectionEvent{
			UserID:    1,
			Username:  "testuser",
			Timestamp: time.Now(),
		}
		if _, err := engine.Process(context.Background(), event); err != nil {
			t.Fatalf("Process() error = %v", err)
		}
	}

	metrics := engine.Metrics()
	if metrics.EventsProcessed != 5 {
		t.Errorf("events processed = %d, want 5", metrics.EventsProcessed)
	}
}

func TestEngine_ConfigureDetector(t *testing.T) {
	alertStore := &mockAlertStore{}
	trustStore := newMockTrustStore()
	eventHistory := &mockEventHistory{}
	broadcaster := &mockBroadcaster{}

	engine := NewEngine(alertStore, trustStore, eventHistory, broadcaster)
	defer engine.Close()

	detector := NewImpossibleTravelDetector(eventHistory)
	engine.RegisterDetector(detector)

	// Configure via engine
	config := []byte(`{"max_speed_kmh": 1200}`)
	err := engine.ConfigureDetector(RuleTypeImpossibleTravel, config)
	if err != nil {
		t.Fatalf("failed to configure detector: %v", err)
	}

	// Verify configuration applied
	d, _ := engine.GetDetector(RuleTypeImpossibleTravel)
	itd := d.(*ImpossibleTravelDetector)
	if itd.Config().MaxSpeedKmH != 1200 {
		t.Errorf("config not applied: max_speed_kmh = %v, want 1200", itd.Config().MaxSpeedKmH)
	}
}

func TestEngine_SetDetectorEnabled(t *testing.T) {
	alertStore := &mockAlertStore{}
	trustStore := newMockTrustStore()
	eventHistory := &mockEventHistory{}
	broadcaster := &mockBroadcaster{}

	engine := NewEngine(alertStore, trustStore, eventHistory, broadcaster)
	defer engine.Close()

	detector := NewImpossibleTravelDetector(eventHistory)
	engine.RegisterDetector(detector)

	// Disable detector
	err := engine.SetDetectorEnabled(RuleTypeImpossibleTravel, false)
	if err != nil {
		t.Fatalf("failed to disable detector: %v", err)
	}

	d, _ := engine.GetDetector(RuleTypeImpossibleTravel)
	if d.Enabled() {
		t.Error("detector should be disabled")
	}

	// Re-enable
	if err := engine.SetDetectorEnabled(RuleTypeImpossibleTravel, true); err != nil {
		t.Fatalf("SetDetectorEnabled() error = %v", err)
	}
	if !d.Enabled() {
		t.Error("detector should be enabled")
	}
}

func TestDefaultEngineConfig(t *testing.T) {
	config := DefaultEngineConfig()

	if !config.Enabled {
		t.Error("engine should be enabled by default")
	}
	if config.TrustScoreDecrement != 10 {
		t.Errorf("TrustScoreDecrement = %d, want 10", config.TrustScoreDecrement)
	}
	if config.TrustScoreRecovery != 1 {
		t.Errorf("TrustScoreRecovery = %d, want 1", config.TrustScoreRecovery)
	}
	if config.TrustScoreThreshold != 50 {
		t.Errorf("TrustScoreThreshold = %d, want 50", config.TrustScoreThreshold)
	}
}

func TestEngine_Enabled(t *testing.T) {
	alertStore := &mockAlertStore{}
	trustStore := newMockTrustStore()
	eventHistory := &mockEventHistory{}
	broadcaster := &mockBroadcaster{}

	engine := NewEngine(alertStore, trustStore, eventHistory, broadcaster)
	defer engine.Close()

	// Initially enabled
	if !engine.Enabled() {
		t.Error("engine should be enabled by default")
	}

	// Disable
	engine.SetEnabled(false)
	if engine.Enabled() {
		t.Error("engine should be disabled after SetEnabled(false)")
	}

	// Re-enable
	engine.SetEnabled(true)
	if !engine.Enabled() {
		t.Error("engine should be enabled after SetEnabled(true)")
	}
}

func TestEngine_ListDetectors(t *testing.T) {
	alertStore := &mockAlertStore{}
	trustStore := newMockTrustStore()
	eventHistory := &mockEventHistory{}
	broadcaster := &mockBroadcaster{}

	engine := NewEngine(alertStore, trustStore, eventHistory, broadcaster)
	defer engine.Close()

	// Initially empty
	detectors := engine.ListDetectors()
	if len(detectors) != 0 {
		t.Errorf("expected 0 detectors, got %d", len(detectors))
	}

	// Register detectors
	engine.RegisterDetector(NewImpossibleTravelDetector(eventHistory))
	engine.RegisterDetector(NewConcurrentStreamsDetector(eventHistory))

	// List should return both
	detectors = engine.ListDetectors()
	if len(detectors) != 2 {
		t.Errorf("expected 2 detectors, got %d", len(detectors))
	}
}

// mockNotifier implements Notifier for testing
type mockNotifier struct {
	name       string
	enabled    bool
	sentAlerts []*Alert
	sendError  error
	mu         sync.Mutex
}

func (m *mockNotifier) Name() string {
	return m.name
}

func (m *mockNotifier) Enabled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.enabled
}

func (m *mockNotifier) Send(ctx context.Context, alert *Alert) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendError != nil {
		return m.sendError
	}
	m.sentAlerts = append(m.sentAlerts, alert)
	return nil
}

func TestEngine_RegisterNotifier(t *testing.T) {
	alertStore := &mockAlertStore{}
	trustStore := newMockTrustStore()
	eventHistory := &mockEventHistory{
		lastEvent: &DetectionEvent{
			UserID:    1,
			Latitude:  40.7128,
			Longitude: -74.0060,
			Timestamp: time.Now().Add(-30 * time.Minute),
		},
	}
	broadcaster := &mockBroadcaster{}

	engine := NewEngine(alertStore, trustStore, eventHistory, broadcaster)
	defer engine.Close()

	notifier := &mockNotifier{
		name:    "test",
		enabled: true,
	}
	engine.RegisterNotifier(notifier)
	engine.RegisterDetector(NewImpossibleTravelDetector(eventHistory))

	// Process an event that triggers an alert
	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		Latitude:  51.5074,
		Longitude: -0.1278, // London
		Timestamp: time.Now(),
	}

	alerts, err := engine.Process(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts) == 0 {
		t.Fatal("expected at least 1 alert")
	}

	// Give the async notifier time to send
	time.Sleep(50 * time.Millisecond)

	notifier.mu.Lock()
	sentCount := len(notifier.sentAlerts)
	notifier.mu.Unlock()

	if sentCount == 0 {
		t.Error("expected notifier to receive alerts")
	}
}

func TestEngine_ConfigureDetector_NotFound(t *testing.T) {
	alertStore := &mockAlertStore{}
	trustStore := newMockTrustStore()
	eventHistory := &mockEventHistory{}
	broadcaster := &mockBroadcaster{}

	engine := NewEngine(alertStore, trustStore, eventHistory, broadcaster)
	defer engine.Close()

	err := engine.ConfigureDetector("nonexistent", []byte(`{}`))
	if err == nil {
		t.Error("expected error for nonexistent detector")
	}
}

func TestEngine_SetDetectorEnabled_NotFound(t *testing.T) {
	alertStore := &mockAlertStore{}
	trustStore := newMockTrustStore()
	eventHistory := &mockEventHistory{}
	broadcaster := &mockBroadcaster{}

	engine := NewEngine(alertStore, trustStore, eventHistory, broadcaster)
	defer engine.Close()

	err := engine.SetDetectorEnabled("nonexistent", true)
	if err == nil {
		t.Error("expected error for nonexistent detector")
	}
}

func TestEngine_GetDetector_NotFound(t *testing.T) {
	alertStore := &mockAlertStore{}
	trustStore := newMockTrustStore()
	eventHistory := &mockEventHistory{}
	broadcaster := &mockBroadcaster{}

	engine := NewEngine(alertStore, trustStore, eventHistory, broadcaster)
	defer engine.Close()

	_, ok := engine.GetDetector("nonexistent")
	if ok {
		t.Error("expected false for nonexistent detector")
	}
}

func TestEngine_RunWithContext(t *testing.T) {
	alertStore := &mockAlertStore{}
	trustStore := newMockTrustStore()
	eventHistory := &mockEventHistory{}
	broadcaster := &mockBroadcaster{}

	engine := NewEngine(alertStore, trustStore, eventHistory, broadcaster)
	// Note: Not calling defer engine.Close() because RunWithContext closes the channel

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := engine.RunWithContext(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

func TestEngine_Process_NilBroadcaster(t *testing.T) {
	eventHistory := &mockEventHistory{
		lastEvent: &DetectionEvent{
			UserID:    1,
			Latitude:  40.7128,
			Longitude: -74.0060,
			Timestamp: time.Now().Add(-30 * time.Minute),
		},
	}
	alertStore := &mockAlertStore{}
	trustStore := newMockTrustStore()

	// Create engine with nil broadcaster
	engine := NewEngine(alertStore, trustStore, eventHistory, nil)
	defer engine.Close()

	engine.RegisterDetector(NewImpossibleTravelDetector(eventHistory))

	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		Latitude:  51.5074,
		Longitude: -0.1278,
		Timestamp: time.Now(),
	}

	// Should not panic with nil broadcaster
	alerts, err := engine.Process(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts) == 0 {
		t.Error("expected at least 1 alert")
	}
}

func TestEngine_Metrics_DetectorMetrics(t *testing.T) {
	alertStore := &mockAlertStore{}
	trustStore := newMockTrustStore()
	eventHistory := &mockEventHistory{
		lastEvent: &DetectionEvent{
			UserID:    1,
			Latitude:  40.7128,
			Longitude: -74.0060,
			Timestamp: time.Now().Add(-30 * time.Minute),
		},
	}
	broadcaster := &mockBroadcaster{}

	engine := NewEngine(alertStore, trustStore, eventHistory, broadcaster)
	defer engine.Close()

	engine.RegisterDetector(NewImpossibleTravelDetector(eventHistory))

	// Process an event that triggers an alert
	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		Latitude:  51.5074,
		Longitude: -0.1278,
		Timestamp: time.Now(),
	}
	if _, err := engine.Process(context.Background(), event); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	metrics := engine.Metrics()

	if metrics.EventsProcessed != 1 {
		t.Errorf("EventsProcessed = %d, want 1", metrics.EventsProcessed)
	}
	if metrics.AlertsGenerated < 1 {
		t.Errorf("AlertsGenerated = %d, want >= 1", metrics.AlertsGenerated)
	}

	detectorMetrics, ok := metrics.DetectorMetrics[RuleTypeImpossibleTravel]
	if !ok {
		t.Error("expected detector metrics for impossible_travel")
	}
	if detectorMetrics.EventsChecked < 1 {
		t.Errorf("EventsChecked = %d, want >= 1", detectorMetrics.EventsChecked)
	}
}
