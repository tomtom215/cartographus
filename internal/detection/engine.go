// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package detection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// Engine coordinates detection rule evaluation and alert generation.
type Engine struct {
	detectors    map[RuleType]Detector
	alertStore   AlertStore
	trustStore   TrustStore
	eventHistory EventHistory
	notifiers    []Notifier
	broadcaster  AlertBroadcaster

	mu            sync.RWMutex
	enabled       bool
	metricsStore  *EngineMetrics
	violationChan chan *Alert // Internal channel for trust score updates
}

// AlertBroadcaster broadcasts alerts via WebSocket.
type AlertBroadcaster interface {
	BroadcastJSON(messageType string, data interface{})
}

// EngineMetrics tracks detection engine performance.
type EngineMetrics struct {
	EventsProcessed  int64
	AlertsGenerated  int64
	DetectionErrors  int64
	ProcessingTimeMs int64
	LastProcessedAt  time.Time
	DetectorMetrics  map[RuleType]*DetectorMetrics
	mu               sync.RWMutex
}

// DetectorMetrics tracks individual detector performance.
type DetectorMetrics struct {
	EventsChecked   int64
	AlertsGenerated int64
	Errors          int64
	AvgProcessingMs float64
	LastTriggeredAt *time.Time
}

// EngineConfig configures the detection engine.
type EngineConfig struct {
	// Enabled controls whether the engine processes events.
	Enabled bool `json:"enabled"`

	// TrustScoreDecrement is the amount to decrease trust score per violation.
	TrustScoreDecrement int `json:"trust_score_decrement"`

	// TrustScoreRecovery is the daily recovery amount for trust scores.
	TrustScoreRecovery int `json:"trust_score_recovery"`

	// TrustScoreThreshold is the score below which users are auto-restricted.
	TrustScoreThreshold int `json:"trust_score_threshold"`
}

// DefaultEngineConfig returns sensible defaults.
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		Enabled:             true,
		TrustScoreDecrement: 10,
		TrustScoreRecovery:  1,
		TrustScoreThreshold: 50,
	}
}

// NewEngine creates a new detection engine.
func NewEngine(
	alertStore AlertStore,
	trustStore TrustStore,
	eventHistory EventHistory,
	broadcaster AlertBroadcaster,
) *Engine {
	e := &Engine{
		detectors:     make(map[RuleType]Detector),
		alertStore:    alertStore,
		trustStore:    trustStore,
		eventHistory:  eventHistory,
		broadcaster:   broadcaster,
		notifiers:     make([]Notifier, 0),
		enabled:       true,
		violationChan: make(chan *Alert, 100),
		metricsStore: &EngineMetrics{
			DetectorMetrics: make(map[RuleType]*DetectorMetrics),
		},
	}

	// Start background violation processor for trust score updates
	go e.processViolations()

	return e
}

// RegisterDetector adds a detector to the engine.
func (e *Engine) RegisterDetector(detector Detector) {
	e.mu.Lock()
	defer e.mu.Unlock()

	ruleType := detector.Type()
	e.detectors[ruleType] = detector
	e.metricsStore.DetectorMetrics[ruleType] = &DetectorMetrics{}

	logging.Info().Str("detector", string(ruleType)).Msg("registered detector")
}

// RegisterNotifier adds a notifier to the engine.
func (e *Engine) RegisterNotifier(notifier Notifier) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.notifiers = append(e.notifiers, notifier)
	logging.Info().Str("notifier", notifier.Name()).Msg("registered notifier")
}

// Process evaluates an event against all enabled detection rules.
func (e *Engine) Process(ctx context.Context, event *DetectionEvent) ([]*Alert, error) {
	detectors := e.getEnabledDetectors()
	if detectors == nil {
		return nil, nil
	}

	start := time.Now()

	// Enrich event with geolocation if not present
	e.enrichWithGeolocation(ctx, event)

	// Run detectors and collect alerts
	alerts, errs := e.runDetectors(ctx, detectors, event)

	// Update processing metrics
	e.updateProcessingMetrics(start)

	// Persist and notify
	e.persistAlerts(ctx, alerts)
	e.notify(ctx, alerts)
	e.broadcast(alerts)

	if len(errs) > 0 {
		return alerts, fmt.Errorf("detection errors: %v", errs)
	}

	return alerts, nil
}

// getEnabledDetectors returns all enabled detectors, or nil if engine is disabled or no detectors.
func (e *Engine) getEnabledDetectors() []Detector {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.enabled {
		return nil
	}

	detectors := make([]Detector, 0, len(e.detectors))
	for _, d := range e.detectors {
		if d.Enabled() {
			detectors = append(detectors, d)
		}
	}

	if len(detectors) == 0 {
		return nil
	}
	return detectors
}

// enrichWithGeolocation adds geolocation data to an event if missing.
// DETERMINISM: Uses epsilon-based coordinate check instead of direct float equality.
func (e *Engine) enrichWithGeolocation(ctx context.Context, event *DetectionEvent) {
	if IsUnknownLocation(event.Latitude, event.Longitude) && event.IPAddress != "" {
		if geo, err := e.eventHistory.GetGeolocation(ctx, event.IPAddress); err == nil && geo != nil {
			event.Latitude = geo.Latitude
			event.Longitude = geo.Longitude
			event.City = geo.City
			event.Region = geo.Region
			event.Country = geo.Country
		}
	}
}

// runDetectors executes all detectors against the event and returns alerts and errors.
func (e *Engine) runDetectors(ctx context.Context, detectors []Detector, event *DetectionEvent) ([]*Alert, []error) {
	var alerts []*Alert
	var errs []error

	for _, detector := range detectors {
		alert, err := e.runSingleDetector(ctx, detector, event)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if alert != nil {
			alerts = append(alerts, alert)
		}
	}

	return alerts, errs
}

// runSingleDetector executes one detector and updates its metrics.
func (e *Engine) runSingleDetector(ctx context.Context, detector Detector, event *DetectionEvent) (*Alert, error) {
	ruleType := detector.Type()

	// Increment events checked
	e.metricsStore.mu.Lock()
	if metrics, ok := e.metricsStore.DetectorMetrics[ruleType]; ok {
		metrics.EventsChecked++
	}
	e.metricsStore.mu.Unlock()

	// Check the event
	alert, err := detector.Check(ctx, event)
	if err != nil {
		e.metricsStore.mu.Lock()
		if metrics, ok := e.metricsStore.DetectorMetrics[ruleType]; ok {
			metrics.Errors++
		}
		e.metricsStore.DetectionErrors++
		e.metricsStore.mu.Unlock()
		return nil, fmt.Errorf("%s: %w", ruleType, err)
	}

	if alert != nil {
		// Update metrics for generated alert
		e.metricsStore.mu.Lock()
		if metrics, ok := e.metricsStore.DetectorMetrics[ruleType]; ok {
			metrics.AlertsGenerated++
			now := time.Now()
			metrics.LastTriggeredAt = &now
		}
		e.metricsStore.AlertsGenerated++
		e.metricsStore.mu.Unlock()

		// Queue for trust score update
		select {
		case e.violationChan <- alert:
		default:
			logging.Warn().Msg("violation channel full, dropping trust score update")
		}
	}

	return alert, nil
}

// updateProcessingMetrics records processing time and event count.
func (e *Engine) updateProcessingMetrics(start time.Time) {
	processingTime := time.Since(start)
	e.metricsStore.mu.Lock()
	e.metricsStore.EventsProcessed++
	e.metricsStore.ProcessingTimeMs = processingTime.Milliseconds()
	e.metricsStore.LastProcessedAt = time.Now()
	e.metricsStore.mu.Unlock()
}

// persistAlerts saves alerts to the alert store.
func (e *Engine) persistAlerts(ctx context.Context, alerts []*Alert) {
	for _, alert := range alerts {
		if err := e.alertStore.SaveAlert(ctx, alert); err != nil {
			logging.Error().Err(err).Msg("failed to save alert")
		}
	}
}

// processViolations handles trust score updates in the background.
func (e *Engine) processViolations() {
	for alert := range e.violationChan {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		// Decrement trust score
		if e.trustStore != nil {
			if err := e.trustStore.DecrementTrustScore(ctx, alert.UserID, 10); err != nil {
				logging.Error().Err(err).Int("user_id", alert.UserID).Msg("failed to update trust score")
			}
		}

		cancel()
	}
}

// StartTrustScoreRecovery starts the daily trust score recovery scheduler.
// This should be called from main.go after engine creation.
//
// The recovery job runs once per day (at the specified interval) and increments
// all users' trust scores by the specified amount, up to a maximum of 100.
//
// Parameters:
//   - ctx: Context for cancellation (stop when context is done)
//   - recoveryAmount: Points to recover per day (e.g., 1)
//   - interval: How often to run recovery (typically 24h)
func (e *Engine) StartTrustScoreRecovery(ctx context.Context, recoveryAmount int, interval time.Duration) {
	if e.trustStore == nil {
		logging.Info().Msg("trust store not configured, skipping recovery scheduler")
		return
	}

	logging.Info().Int("amount", recoveryAmount).Str("interval", interval.String()).Msg("starting trust score recovery scheduler")

	// Run immediately on startup, then on interval
	go func() {
		// Run once at startup
		e.runRecovery(recoveryAmount)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logging.Info().Msg("trust score recovery scheduler stopped")
				return
			case <-ticker.C:
				e.runRecovery(recoveryAmount)
			}
		}
	}()
}

// runRecovery performs a single trust score recovery operation.
func (e *Engine) runRecovery(amount int) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := e.trustStore.RecoverTrustScores(ctx, amount); err != nil {
		logging.Error().Err(err).Msg("trust score recovery failed")
	} else {
		logging.Info().Int("amount", amount).Msg("trust score recovery completed")
	}
}

// notify sends alerts to all enabled notifiers.
func (e *Engine) notify(ctx context.Context, alerts []*Alert) {
	if len(alerts) == 0 {
		return
	}

	e.mu.RLock()
	notifiers := make([]Notifier, 0, len(e.notifiers))
	for _, n := range e.notifiers {
		if n.Enabled() {
			notifiers = append(notifiers, n)
		}
	}
	e.mu.RUnlock()

	for _, alert := range alerts {
		for _, notifier := range notifiers {
			go func(n Notifier, a *Alert) {
				if err := n.Send(ctx, a); err != nil {
					logging.Error().Err(err).Str("notifier", n.Name()).Msg("failed to send alert")
				}
			}(notifier, alert)
		}
	}
}

// broadcast sends alerts via WebSocket.
func (e *Engine) broadcast(alerts []*Alert) {
	if e.broadcaster == nil || len(alerts) == 0 {
		return
	}

	for _, alert := range alerts {
		e.broadcaster.BroadcastJSON("detection_alert", alert)
	}
}

// SetEnabled enables or disables the detection engine.
func (e *Engine) SetEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = enabled
}

// Enabled returns whether the engine is enabled.
func (e *Engine) Enabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// GetDetector returns a detector by rule type.
func (e *Engine) GetDetector(ruleType RuleType) (Detector, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	d, ok := e.detectors[ruleType]
	return d, ok
}

// ListDetectors returns all registered detectors.
func (e *Engine) ListDetectors() []Detector {
	e.mu.RLock()
	defer e.mu.RUnlock()

	detectors := make([]Detector, 0, len(e.detectors))
	for _, d := range e.detectors {
		detectors = append(detectors, d)
	}
	return detectors
}

// Metrics returns a copy of the engine metrics.
func (e *Engine) Metrics() EngineMetrics {
	e.metricsStore.mu.RLock()
	defer e.metricsStore.mu.RUnlock()

	// Deep copy detector metrics
	detectorMetrics := make(map[RuleType]*DetectorMetrics)
	for k, v := range e.metricsStore.DetectorMetrics {
		dm := *v
		detectorMetrics[k] = &dm
	}

	return EngineMetrics{
		EventsProcessed:  e.metricsStore.EventsProcessed,
		AlertsGenerated:  e.metricsStore.AlertsGenerated,
		DetectionErrors:  e.metricsStore.DetectionErrors,
		ProcessingTimeMs: e.metricsStore.ProcessingTimeMs,
		LastProcessedAt:  e.metricsStore.LastProcessedAt,
		DetectorMetrics:  detectorMetrics,
	}
}

// Close gracefully shuts down the engine.
func (e *Engine) Close() error {
	close(e.violationChan)
	return nil
}

// RunWithContext starts the detection engine and blocks until the context is canceled.
// This method is designed to work with suture supervision.
//
// It runs the violation processor for trust score updates and returns when
// the context is done. The method returns ctx.Err() on normal shutdown.
func (e *Engine) RunWithContext(ctx context.Context) error {
	logging.Info().Msg("detection engine started")

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	logging.Info().Msg("detection engine shutting down")
	if err := e.Close(); err != nil {
		logging.Error().Err(err).Msg("error during shutdown")
	}

	return ctx.Err()
}

// ConfigureDetector updates a detector's configuration.
func (e *Engine) ConfigureDetector(ruleType RuleType, config json.RawMessage) error {
	e.mu.RLock()
	detector, ok := e.detectors[ruleType]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("detector not found: %s", ruleType)
	}

	return detector.Configure(config)
}

// SetDetectorEnabled enables or disables a specific detector.
func (e *Engine) SetDetectorEnabled(ruleType RuleType, enabled bool) error {
	e.mu.RLock()
	detector, ok := e.detectors[ruleType]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("detector not found: %s", ruleType)
	}

	detector.SetEnabled(enabled)
	return nil
}
