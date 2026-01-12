// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package eventprocessor

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

// =============================================================================
// Table-Driven Tests for Stub Implementations (non-NATS builds)
// =============================================================================
// These stubs return ErrNATSNotEnabled or similar errors.
// Refactored to use consolidated table-driven tests for reduced complexity.

// =============================================================================
// Test Helpers
// =============================================================================

// constructorTest defines a test case for constructor functions.
type constructorTest struct {
	name      string
	construct func() (interface{}, error)
	wantErr   bool // true = expect error, false = expect nil error
}

// runConstructorTests runs a slice of constructor tests.
func runConstructorTests(t *testing.T, tests []constructorTest) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := tt.construct()
			if tt.wantErr {
				if err == nil {
					t.Errorf("%s() should return error in non-NATS build", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("%s() unexpected error = %v", tt.name, err)
				}
			}
			// Note: Result nil check removed - some stubs return empty structs
		})
	}
}

// methodTest defines a test case for stub methods.
type methodTest struct {
	name    string
	method  func() error
	wantErr error
}

// runMethodTests runs a slice of method tests checking expected errors.
func runMethodTests(t *testing.T, tests []methodTest) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.method()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("%s() error = %v, want nil", tt.name, err)
				}
			} else {
				if err == nil {
					t.Errorf("%s() should return error, want %v", tt.name, tt.wantErr)
				}
			}
		})
	}
}

// configField defines a test case for configuration field validation.
type configField struct {
	name string
	got  interface{}
	want interface{}
}

// verifyConfigFields runs a slice of config field tests.
func verifyConfigFields(t *testing.T, fields []configField) {
	t.Helper()
	for _, f := range fields {
		t.Run(f.name, func(t *testing.T) {
			if f.got != f.want {
				t.Errorf("%s = %v, want %v", f.name, f.got, f.want)
			}
		})
	}
}

// TestNATSDisabledError tests the error message format.
func TestNATSDisabledError(t *testing.T) {
	t.Parallel()

	err := ErrNATSNotEnabled
	expected := "NATS event processing not enabled (build with -tags nats)"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

// =============================================================================
// Constructor Tests - All should return error in non-NATS builds
// =============================================================================

func TestStub_Constructors(t *testing.T) {
	t.Parallel()

	tests := []constructorTest{
		{"NewAppender", func() (interface{}, error) { return NewAppender(nil, AppenderConfig{}) }, true},
		{"NewDuckDBStore", func() (interface{}, error) { return NewDuckDBStore(nil) }, true},
		{"NewPublisher", func() (interface{}, error) { return NewPublisher(PublisherConfig{}, nil) }, true},
		{"NewFallbackReader", func() (interface{}, error) { return NewFallbackReader("nats://localhost:4222") }, true},
		{"NewResilientReader", func() (interface{}, error) {
			cfg := DefaultResilientReaderConfig("nats://localhost:4222")
			return NewResilientReader(&cfg)
		}, true},
		{"NewEmbeddedServer", func() (interface{}, error) { return NewEmbeddedServer(nil) }, true},
		{"NewStreamManager", func() (interface{}, error) { return NewStreamManager(nil, nil) }, true},
		{"NewSubscriber", func() (interface{}, error) { return NewSubscriber(nil, nil) }, true},
		{"NewSyncEventPublisher", func() (interface{}, error) { return NewSyncEventPublisher(nil) }, true},
		{"NewDLQHandler", func() (interface{}, error) { return NewDLQHandler(DLQConfig{}) }, true},
		{"NewDuckDBConsumer", func() (interface{}, error) {
			cfg := DefaultConsumerConfig()
			return NewDuckDBConsumer(nil, nil, &cfg)
		}, true},
		{"NewRouter", func() (interface{}, error) {
			cfg := DefaultRouterConfig()
			return NewRouter(&cfg, nil, nil)
		}, true},
		{"NewStreamInitializer", func() (interface{}, error) { return NewStreamInitializer(nil, &StreamConfig{}) }, true},
		{"NewDetectionHandler", func() (interface{}, error) { return NewDetectionHandler(nil, nil) }, false}, // returns (nil, nil)
	}

	runConstructorTests(t, tests)
}

// =============================================================================
// Appender Stub Tests
// =============================================================================

func TestAppenderStub_Methods(t *testing.T) {
	t.Parallel()

	appender := &Appender{}
	ctx := context.Background()

	runMethodTests(t, []methodTest{
		{"Start", func() error { return appender.Start(ctx) }, ErrNATSNotEnabled},
		{"Append", func() error { return appender.Append(ctx, NewMediaEvent(SourcePlex)) }, ErrNATSNotEnabled},
		{"Flush", func() error { return appender.Flush(ctx) }, ErrNATSNotEnabled},
		{"Close", func() error { return appender.Close() }, nil},
	})

	// Stats should return empty values
	t.Run("Stats", func(t *testing.T) {
		stats := appender.Stats()
		if stats.EventsReceived != 0 || stats.EventsFlushed != 0 ||
			stats.FlushCount != 0 || stats.ErrorCount != 0 || stats.BufferSize != 0 {
			t.Error("Stats() should return all zero values")
		}
	})
}

// =============================================================================
// DuckDBStore Stub Tests
// =============================================================================

func TestDuckDBStoreStub_Methods(t *testing.T) {
	t.Parallel()
	store := &DuckDBStore{}
	runMethodTests(t, []methodTest{
		{"InsertMediaEvents", func() error { return store.InsertMediaEvents(context.Background(), nil) }, ErrNATSNotEnabled},
	})
}

// =============================================================================
// Publisher Stub Tests
// =============================================================================

func TestPublisherStub_Methods(t *testing.T) {
	t.Parallel()

	pub := &Publisher{}
	ctx := context.Background()
	pub.SetCircuitBreaker(nil) // Should not panic

	runMethodTests(t, []methodTest{
		{"Publish", func() error { return pub.Publish(ctx, "topic", nil) }, ErrNATSNotEnabled},
		{"PublishEvent", func() error { return pub.PublishEvent(ctx, NewMediaEvent(SourcePlex)) }, ErrNATSNotEnabled},
		{"PublishBatch", func() error { return pub.PublishBatch(ctx, "topic", "msg1", "msg2") }, ErrNATSNotEnabled},
		{"Close", func() error { return pub.Close() }, nil},
	})

	if wp := pub.WatermillPublisher(); wp != nil {
		t.Error("Publisher.WatermillPublisher() should return nil")
	}
}

// =============================================================================
// Reader Stub Test Helper
// =============================================================================

// readerStubTest is a helper for testing reader-like stub implementations.
type readerStubTest struct {
	name       string
	query      func(context.Context, string) (interface{}, error)
	getMessage func(context.Context, string, uint64) (interface{}, error)
	getLastSeq func(context.Context, string) (uint64, error)
	health     func(context.Context) error
	close      func() error
}

func (r readerStubTest) run(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	t.Run(r.name+"/Query", func(t *testing.T) {
		_, err := r.query(ctx, "stream")
		if err == nil {
			t.Error("Query() should return error")
		}
	})

	t.Run(r.name+"/GetMessage", func(t *testing.T) {
		_, err := r.getMessage(ctx, "stream", 1)
		if err == nil {
			t.Error("GetMessage() should return error")
		}
	})

	t.Run(r.name+"/GetLastSequence", func(t *testing.T) {
		seq, err := r.getLastSeq(ctx, "stream")
		if err == nil {
			t.Error("GetLastSequence() should return error")
		}
		if seq != 0 {
			t.Errorf("GetLastSequence() = %d, want 0", seq)
		}
	})

	t.Run(r.name+"/Health", func(t *testing.T) {
		if err := r.health(ctx); err == nil {
			t.Error("Health() should return error")
		}
	})

	t.Run(r.name+"/Close", func(t *testing.T) {
		if err := r.close(); err != nil {
			t.Errorf("Close() error = %v, want nil", err)
		}
	})
}

// =============================================================================
// FallbackReader Stub Tests
// =============================================================================

func TestFallbackReaderStub_Methods(t *testing.T) {
	t.Parallel()
	reader := &FallbackReader{}
	readerStubTest{
		name:  "FallbackReader",
		query: func(ctx context.Context, s string) (interface{}, error) { return reader.Query(ctx, s, nil) },
		getMessage: func(ctx context.Context, s string, seq uint64) (interface{}, error) {
			return reader.GetMessage(ctx, s, seq)
		},
		getLastSeq: reader.GetLastSequence,
		health:     reader.Health,
		close:      reader.Close,
	}.run(t)
}

// =============================================================================
// ResilientReader Stub Tests
// =============================================================================

func TestResilientReaderStub_DefaultConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultResilientReaderConfig("nats://localhost:4222")
	verifyConfigFields(t, []configField{
		{"NATSURL", cfg.NATSURL, "nats://localhost:4222"},
		{"CircuitBreakerName", cfg.CircuitBreakerName, "stream-reader"},
		{"MaxRequests", cfg.MaxRequests, uint32(3)},
		{"FailureThreshold", cfg.FailureThreshold, uint32(5)},
	})
}

func TestResilientReaderStub_Methods(t *testing.T) {
	t.Parallel()
	reader := &ResilientReader{}
	reader.SetPrimaryReader(nil) // Should not panic

	readerStubTest{
		name:  "ResilientReader",
		query: func(ctx context.Context, s string) (interface{}, error) { return reader.Query(ctx, s, nil) },
		getMessage: func(ctx context.Context, s string, seq uint64) (interface{}, error) {
			return reader.GetMessage(ctx, s, seq)
		},
		getLastSeq: reader.GetLastSequence,
		health:     reader.Health,
		close:      reader.Close,
	}.run(t)

	_ = reader.Stats() // Should not panic
	if count := reader.FallbackCount(); count != 0 {
		t.Errorf("FallbackCount() = %d, want 0", count)
	}
}

// =============================================================================
// EmbeddedServer Stub Tests
// =============================================================================

func TestEmbeddedServerStub_Methods(t *testing.T) {
	t.Parallel()
	server := &EmbeddedServer{}
	verifyConfigFields(t, []configField{
		{"ClientURL", server.ClientURL(), ""},
		{"IsRunning", server.IsRunning(), false},
		{"JetStreamEnabled", server.JetStreamEnabled(), false},
	})
	if err := server.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() error = %v, want nil", err)
	}
}

// =============================================================================
// StreamManager Stub Tests
// =============================================================================

func TestStreamManagerStub_Methods(t *testing.T) {
	t.Parallel()

	mgr := &StreamManager{}
	ctx := context.Background()

	tests := []struct {
		name   string
		method func() (interface{}, error)
	}{
		{"EnsureStream", func() (interface{}, error) { return mgr.EnsureStream(ctx) }},
		{"GetStreamInfo", func() (interface{}, error) { return mgr.GetStreamInfo(ctx) }},
		{"PurgeStream", func() (interface{}, error) { return nil, mgr.PurgeStream(ctx) }},
		{"DeleteStream", func() (interface{}, error) { return nil, mgr.DeleteStream(ctx) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.method()
			if err == nil {
				t.Errorf("StreamManager.%s() should return error", tt.name)
			}
			if result != nil {
				t.Errorf("StreamManager.%s() result should be nil", tt.name)
			}
		})
	}
}

// =============================================================================
// Subscriber Stub Tests
// =============================================================================

func TestSubscriberStub_Methods(t *testing.T) {
	t.Parallel()

	sub := &Subscriber{}
	ctx := context.Background()

	// Subscribe returns error
	ch, err := sub.Subscribe(ctx, "topic")
	if err == nil {
		t.Error("Subscribe() should return error")
	}
	if ch != nil {
		t.Error("Subscribe() should return nil channel")
	}

	// Close returns nil
	if err := sub.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	// NewMessageHandler returns non-nil handler
	handler := sub.NewMessageHandler("topic")
	if handler == nil {
		t.Error("NewMessageHandler() should return non-nil handler")
	}

	// Handle returns self for chaining
	result := handler.Handle(func(ctx context.Context, msg interface{}) error { return nil })
	if result != handler {
		t.Error("Handle() should return self for chaining")
	}

	// Run returns error
	if err := handler.Run(ctx); err == nil {
		t.Error("Run() should return error")
	}

	// NewEventHandler returns non-nil handler
	eventHandler := sub.NewEventHandler("topic")
	if eventHandler == nil {
		t.Error("NewEventHandler() should return non-nil handler")
	}

	// EventHandler.Handle returns self for chaining
	eventResult := eventHandler.Handle(func(ctx context.Context, event *MediaEvent) error { return nil })
	if eventResult != eventHandler {
		t.Error("EventHandler.Handle() should return self for chaining")
	}

	// EventHandler.Run returns error
	if err := eventHandler.Run(ctx); err == nil {
		t.Error("EventHandler.Run() should return error")
	}
}

// =============================================================================
// SyncEventPublisher Stub Tests
// =============================================================================

func TestSyncEventPublisherStub_Methods(t *testing.T) {
	t.Parallel()

	pub := &SyncEventPublisher{}
	ctx := context.Background()

	if err := pub.PublishPlaybackEvent(ctx, nil); !errors.Is(err, ErrNATSNotEnabled) {
		t.Errorf("PublishPlaybackEvent() error = %v, want ErrNATSNotEnabled", err)
	}
}

// =============================================================================
// CQRS Stub Tests
// =============================================================================

func TestCQRSStub_Methods(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// DefaultEventBusConfig returns empty config
	cfg := DefaultEventBusConfig()
	if cfg.GeneratePublishTopic != nil {
		t.Error("GeneratePublishTopic should be nil")
	}

	// EventBus methods
	bus := &EventBus{}
	if err := bus.Publish(ctx, nil); !errors.Is(err, ErrNATSNotEnabled) {
		t.Errorf("Publish() error = %v, want ErrNATSNotEnabled", err)
	}
	if err := bus.PublishMediaEvent(ctx, nil); !errors.Is(err, ErrNATSNotEnabled) {
		t.Errorf("PublishMediaEvent() error = %v, want ErrNATSNotEnabled", err)
	}

	// MediaEventHandlerFunc
	called := false
	handler := MediaEventHandlerFunc(func(ctx context.Context, event *MediaEvent) error {
		called = true
		return nil
	})
	if err := handler.Handle(ctx, nil); err != nil {
		t.Errorf("Handle() error = %v, want nil", err)
	}
	if !called {
		t.Error("Handler was not called")
	}

	// EventHandlerGroup
	group := &EventHandlerGroup{}
	result := group.AddMediaEventFunc("test", nil)
	if result != group {
		t.Error("AddMediaEventFunc() should return self for chaining")
	}
	if name := group.Name(); name != "" {
		t.Errorf("Name() = %q, want empty string", name)
	}
}

func TestCQRSStub_EventRegistry(t *testing.T) {
	t.Parallel()

	registry := NewEventRegistry()
	if registry == nil {
		t.Fatal("NewEventRegistry() returned nil")
	}

	// Register and retrieve
	registry.Register("TestEvent", struct{}{})
	typ, ok := registry.Get("TestEvent")
	if !ok || typ == nil {
		t.Error("Get() should find registered type")
	}

	// Unregistered type
	_, ok = registry.Get("Unknown")
	if ok {
		t.Error("Get() should not find unregistered type")
	}

	// RegisterMediaEvent
	registry.RegisterMediaEvent()
	typ, ok = registry.Get("MediaEvent")
	if !ok || typ == nil {
		t.Error("MediaEvent should be registered")
	}

	// Names count
	names := registry.Names()
	if len(names) != 2 {
		t.Errorf("Names() length = %d, want 2", len(names))
	}
}

// =============================================================================
// DLQ Stub Tests
// =============================================================================

func TestDLQStub_Errors(t *testing.T) {
	t.Parallel()

	// ErrorCategoryString
	cat := ErrorCategoryUnknown
	if cat.String() != "unknown" {
		t.Errorf("String() = %q, want unknown", cat.String())
	}

	// RetryableError
	cause := errors.New("underlying error")
	retryErr := NewRetryableError("retry message", cause)
	if retryErr.Error() != "retry message" {
		t.Errorf("Error() = %q, want retry message", retryErr.Error())
	}
	if !errors.Is(retryErr.Unwrap(), cause) {
		t.Error("Unwrap() should return cause")
	}

	// PermanentError
	permErr := NewPermanentError("permanent message", cause)
	if permErr.Error() != "permanent message" {
		t.Errorf("Error() = %q, want permanent message", permErr.Error())
	}
	if !errors.Is(permErr.Unwrap(), cause) {
		t.Error("Unwrap() should return cause")
	}

	// IsRetryableError - stub returns false
	if IsRetryableError(errors.New("test")) {
		t.Error("IsRetryableError() should return false")
	}

	// IsPermanentError - stub returns false
	if IsPermanentError(errors.New("test")) {
		t.Error("IsPermanentError() should return false")
	}
}

func TestDLQStub_Handler(t *testing.T) {
	t.Parallel()

	// NewDLQEntry returns nil
	if entry := NewDLQEntry(nil, nil, "msg-1"); entry != nil {
		t.Error("NewDLQEntry() should return nil in stub")
	}

	// DefaultDLQConfig returns empty config
	cfg := DefaultDLQConfig()
	if cfg.MaxRetries != 0 {
		t.Errorf("MaxRetries = %d, want 0", cfg.MaxRetries)
	}

	// DLQHandler methods
	handler := &DLQHandler{}

	// AddEntry returns nil
	if entry := handler.AddEntry(nil, nil, "msg-1"); entry != nil {
		t.Error("AddEntry() should return nil")
	}

	// GetEntry returns nil
	if entry := handler.GetEntry("event-1"); entry != nil {
		t.Error("GetEntry() should return nil")
	}

	// IncrementRetry returns false
	if handler.IncrementRetry("event-1", nil) {
		t.Error("IncrementRetry() should return false")
	}

	// RemoveEntry returns false
	if handler.RemoveEntry("event-1") {
		t.Error("RemoveEntry() should return false")
	}

	// GetPendingRetries returns nil
	if entries := handler.GetPendingRetries(); entries != nil {
		t.Error("GetPendingRetries() should return nil")
	}

	// ListEntries returns nil
	if entries := handler.ListEntries(); entries != nil {
		t.Error("ListEntries() should return nil")
	}

	// Cleanup returns 0
	if count := handler.Cleanup(); count != 0 {
		t.Errorf("Cleanup() = %d, want 0", count)
	}

	// Stats
	stats := handler.Stats()
	if stats.TotalEntries != 0 {
		t.Errorf("Stats().TotalEntries = %d, want 0", stats.TotalEntries)
	}
}

func TestDLQStub_RetryPolicy(t *testing.T) {
	t.Parallel()

	policy := DefaultRetryPolicy()
	if policy == nil {
		t.Fatal("DefaultRetryPolicy() returned nil")
	}

	if backoff := policy.CalculateBackoff(1); backoff != 0 {
		t.Errorf("CalculateBackoff() = %v, want 0", backoff)
	}

	if policy.ShouldRetry(nil, 1) {
		t.Error("ShouldRetry() should return false")
	}
}

// =============================================================================
// Detection Handler Stub Tests
// =============================================================================

func TestDetectionHandlerStub_Methods(t *testing.T) {
	t.Parallel()

	cfg := DefaultDetectionHandlerConfig()
	if !cfg.ContinueOnError {
		t.Error("ContinueOnError should be true by default")
	}

	handler := &DetectionHandler{}
	stats := handler.Stats()
	if stats.MessagesProcessed != 0 {
		t.Errorf("Stats().MessagesProcessed = %d, want 0", stats.MessagesProcessed)
	}
}

// =============================================================================
// DuckDB Consumer Stub Tests
// =============================================================================

func TestDuckDBConsumerStub_Config(t *testing.T) {
	t.Parallel()
	cfg := DefaultConsumerConfig()
	verifyConfigFields(t, []configField{
		{"Topic", cfg.Topic, "playback.>"},
		{"EnableDeduplication", cfg.EnableDeduplication, true},
		{"WorkerCount", cfg.WorkerCount, 1},
	})
}

func TestDuckDBConsumerStub_Methods(t *testing.T) {
	t.Parallel()

	consumer := &DuckDBConsumer{}
	ctx := context.Background()

	if err := consumer.Start(ctx); !errors.Is(err, ErrNATSNotEnabled) {
		t.Errorf("Start() error = %v, want ErrNATSNotEnabled", err)
	}

	consumer.Stop() // Should not panic

	if consumer.IsRunning() {
		t.Error("IsRunning() should return false")
	}

	stats := consumer.Stats()
	if stats.MessagesProcessed != 0 {
		t.Errorf("Stats().MessagesProcessed = %d, want 0", stats.MessagesProcessed)
	}
}

// =============================================================================
// Forwarder Stub Tests
// =============================================================================

func TestForwarderStub_Config(t *testing.T) {
	t.Parallel()
	cfg := DefaultForwarderConfig()
	verifyConfigFields(t, []configField{
		{"BatchSize", cfg.BatchSize, 100},
		{"MaxRetries", cfg.MaxRetries, 5},
		{"PollInterval", cfg.PollInterval, 100 * time.Millisecond},
	})
}

func TestForwarderStub_Methods(t *testing.T) {
	t.Parallel()

	fwd := &Forwarder{}
	ctx := context.Background()

	if err := fwd.Start(ctx); !errors.Is(err, ErrNATSNotEnabled) {
		t.Errorf("Start() error = %v, want ErrNATSNotEnabled", err)
	}

	if err := fwd.Stop(); err != nil {
		t.Errorf("Stop() error = %v, want nil", err)
	}

	if fwd.IsRunning() {
		t.Error("IsRunning() should return false")
	}
}

func TestForwarderStub_TransactionalPublisher(t *testing.T) {
	t.Parallel()

	pub := &TransactionalPublisher{}

	if err := pub.Publish("test.topic", nil); !errors.Is(err, ErrNATSNotEnabled) {
		t.Errorf("Publish() error = %v, want ErrNATSNotEnabled", err)
	}

	if err := pub.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestForwarderStub_OutboxStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewInMemoryOutboxStore()
	if store == nil {
		t.Fatal("NewInMemoryOutboxStore() returned nil")
	}

	// Initially empty
	if store.Size() != 0 {
		t.Errorf("Size() = %d, want 0", store.Size())
	}

	// Store a message
	msg := &OutboxMessage{
		ID:      "test-1",
		Topic:   "test.topic",
		Payload: []byte("test data"),
	}
	if err := store.Store(ctx, msg); err != nil {
		t.Errorf("Store() error = %v, want nil", err)
	}

	// Size should now be 1
	if store.Size() != 1 {
		t.Errorf("Size() = %d, want 1", store.Size())
	}

	// GetPending
	entries, err := store.GetPending(ctx, 10)
	if err != nil {
		t.Errorf("GetPending() error = %v, want nil", err)
	}
	if len(entries) != 1 {
		t.Errorf("GetPending() returned %d entries, want 1", len(entries))
	}

	// GetByID
	entry, err := store.GetByID(ctx, "test-1")
	if err != nil || entry == nil || entry.ID != "test-1" {
		t.Error("GetByID() should return the stored message")
	}

	// GetByID for non-existent
	if _, err := store.GetByID(ctx, "non-existent"); err == nil {
		t.Error("GetByID() should error for non-existent ID")
	}

	// MarkFailed
	if err := store.MarkFailed(ctx, "test-1", errors.New("test error")); err != nil {
		t.Errorf("MarkFailed() error = %v, want nil", err)
	}
	entry, _ = store.GetByID(ctx, "test-1")
	if entry.RetryCount != 1 {
		t.Errorf("RetryCount = %d, want 1", entry.RetryCount)
	}

	// MarkDelivered
	if err := store.MarkDelivered(ctx, "test-1"); err != nil {
		t.Errorf("MarkDelivered() error = %v, want nil", err)
	}

	// Size should be 0 after delivery
	if store.Size() != 0 {
		t.Errorf("Size() = %d, want 0 after MarkDelivered", store.Size())
	}
}

// =============================================================================
// Health Stub Tests
// =============================================================================

func TestHealthStub_Config(t *testing.T) {
	t.Parallel()
	cfg := DefaultHealthConfig()
	verifyConfigFields(t, []configField{
		{"Timeout", cfg.Timeout, 5 * time.Second},
		{"Interval", cfg.Interval, 30 * time.Second},
	})
}

func TestHealthStub_HealthChecker(t *testing.T) {
	t.Parallel()

	checker := NewHealthChecker(HealthConfig{})
	if checker == nil {
		t.Fatal("NewHealthChecker() returned nil")
	}

	checker.RegisterComponent("test", nil)
	checker.UnregisterComponent("test")

	health := checker.CheckAll(context.Background())
	if health.Healthy {
		t.Error("CheckAll() should return unhealthy")
	}

	componentHealth := checker.CheckComponent(context.Background(), "test")
	if componentHealth.Healthy {
		t.Error("CheckComponent() should return unhealthy")
	}
}

// =============================================================================
// Logging Stub Tests
// =============================================================================

func TestLoggingStub_CorrelationID(t *testing.T) {
	t.Parallel()

	// GenerateCorrelationID - stub returns empty string
	if genID := GenerateCorrelationID(); genID != "" {
		t.Errorf("GenerateCorrelationID() = %q, stub should return empty", genID)
	}

	// Context functions - stub just returns the context unchanged
	ctx := ContextWithCorrelationID(context.Background(), "test-id")
	if ctx == nil {
		t.Error("ContextWithCorrelationID() returned nil")
	}

	ctx2 := ContextWithNewCorrelationID(context.Background())
	if ctx2 == nil {
		t.Error("ContextWithNewCorrelationID() returned nil")
	}

	// Stub returns empty string from context
	if retrieved := CorrelationIDFromContext(ctx); retrieved != "" {
		t.Errorf("CorrelationIDFromContext() = %q, stub should return empty", retrieved)
	}
}

func TestLoggingStub_StructuredLogger(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewStructuredLogger(&buf)
	if logger == nil {
		t.Fatal("NewStructuredLogger() returned nil")
	}

	// Test methods - should not panic
	logger.SetLevel(LogLevelDebug)
	if logger.GetLevel() != LogLevelInfo {
		t.Errorf("GetLevel() = %d, want %d", logger.GetLevel(), LogLevelInfo)
	}

	logger.WithFields("key", "value")
	logger.Debug("test")
	logger.Info("test")
	logger.Warn("test")
	logger.Error("test")

	ctx := context.Background()
	logger.DebugContext(ctx, "test")
	logger.InfoContext(ctx, "test")
	logger.WarnContext(ctx, "test")
	logger.ErrorContext(ctx, "test")
}

func TestLoggingStub_StructuredLoggerNilWriter(t *testing.T) {
	t.Parallel()

	logger := NewStructuredLogger(nil)
	if logger == nil {
		t.Fatal("NewStructuredLogger(nil) returned nil")
	}
}

func TestLoggingStub_SetDefaultLogLevel(t *testing.T) {
	t.Parallel()
	SetDefaultLogLevel(LogLevelInfo) // Should not panic
}

func TestLoggingStub_EventLogger(t *testing.T) {
	t.Parallel()

	logger := NewEventLogger(nil)
	if logger == nil {
		t.Fatal("NewEventLogger() returned nil")
	}

	ctx := context.Background()

	// Methods should not panic
	logger.LogEventReceived(ctx, "event-1", "plex", "movie")
	logger.LogEventProcessed(ctx, "event-1", 100)
	logger.LogEventFailed(ctx, "event-1", errors.New("test error"))
	logger.LogDuplicate(ctx, "event-1", "already processed")
	logger.LogDLQEntry(ctx, "event-1", errors.New("max retries"), 5)
	logger.LogBatchFlush(ctx, 10, 50)
}

// =============================================================================
// Router Stub Tests
// =============================================================================

func TestRouterStub_Config(t *testing.T) {
	t.Parallel()
	cfg := DefaultRouterConfig()
	verifyConfigFields(t, []configField{
		{"CloseTimeout", cfg.CloseTimeout, 30 * time.Second},
		{"RetryMaxRetries", cfg.RetryMaxRetries, 5},
		{"PoisonQueueTopic", cfg.PoisonQueueTopic, "dlq.playback"},
	})
}

func TestRouterStub_Methods(t *testing.T) {
	t.Parallel()

	router := &Router{}
	ctx := context.Background()

	// AddConsumerHandler returns nil
	if result := router.AddConsumerHandler("handler", "topic", nil, nil); result != nil {
		t.Error("AddConsumerHandler() should return nil")
	}

	// AddHandler returns nil
	if result := router.AddHandler("name", "subTopic", nil, "pubTopic", nil, nil); result != nil {
		t.Error("AddHandler() should return nil")
	}

	// AddHandlerMiddleware returns error
	if err := router.AddHandlerMiddleware("handler"); !errors.Is(err, ErrNATSNotEnabled) {
		t.Errorf("AddHandlerMiddleware() error = %v, want ErrNATSNotEnabled", err)
	}

	// Run returns error
	if err := router.Run(ctx); !errors.Is(err, ErrNATSNotEnabled) {
		t.Errorf("Run() error = %v, want ErrNATSNotEnabled", err)
	}

	// RunAsync returns closed channel
	if ch := router.RunAsync(ctx); ch == nil {
		t.Error("RunAsync() should return channel")
	}

	// Running returns closed channel
	if running := router.Running(); running == nil {
		t.Error("Running() should return channel")
	}

	// Close returns nil
	if err := router.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	// IsRunning returns false
	if router.IsRunning() {
		t.Error("IsRunning() should return false")
	}

	// Metrics returns empty
	metrics := router.Metrics()
	if metrics.MessagesReceived != 0 {
		t.Errorf("Metrics().MessagesReceived = %d, want 0", metrics.MessagesReceived)
	}

	// HealthCheck returns unhealthy
	health := router.HealthCheck(ctx)
	if health.Healthy {
		t.Error("HealthCheck() should return unhealthy")
	}
}

func TestRouterStub_Deduplicator(t *testing.T) {
	t.Parallel()

	// NewInMemoryDeduplicator returns nil in stub
	if dedup := NewInMemoryDeduplicator(time.Minute); dedup != nil {
		t.Error("NewInMemoryDeduplicator() should return nil in stub")
	}

	// Test on a zero value dedup
	d := &InMemoryDeduplicator{}
	isDup, err := d.IsDuplicate(context.Background(), "msg-1")
	if !errors.Is(err, ErrNATSNotEnabled) {
		t.Errorf("IsDuplicate() error = %v, want ErrNATSNotEnabled", err)
	}
	if isDup {
		t.Error("IsDuplicate() should return false")
	}
}

// =============================================================================
// Router Init Stub Tests
// =============================================================================

func TestRouterInitStub_Config(t *testing.T) {
	t.Parallel()

	cfg := DefaultRouterComponentsConfig()

	if cfg.RouterConfig == nil {
		t.Error("RouterConfig should not be nil")
	}
	if !cfg.DuckDBHandlerConfig.EnableCrossSourceDedup {
		t.Error("DuckDBHandlerConfig.EnableCrossSourceDedup should be true")
	}
}

func TestRouterInitStub_RouterComponents(t *testing.T) {
	t.Parallel()

	components := &RouterComponents{}
	ctx := context.Background()

	if err := components.Start(ctx); !errors.Is(err, ErrNATSNotEnabled) {
		t.Errorf("Start() error = %v, want ErrNATSNotEnabled", err)
	}

	if err := components.Stop(); err != nil {
		t.Errorf("Stop() error = %v, want nil", err)
	}

	if components.IsRunning() {
		t.Error("IsRunning() should return false")
	}

	stats := components.Stats()
	if stats.DuckDB.MessagesReceived != 0 {
		t.Errorf("Stats().DuckDB.MessagesReceived = %d, want 0", stats.DuckDB.MessagesReceived)
	}
}

// =============================================================================
// Stream Init Stub Tests
// =============================================================================

func TestStreamInitStub_Methods(t *testing.T) {
	t.Parallel()

	si := &StreamInitializer{}
	ctx := context.Background()

	// EnsureStream returns error
	result, err := si.EnsureStream(ctx)
	if !errors.Is(err, ErrNATSNotEnabled) {
		t.Errorf("EnsureStream() error = %v, want ErrNATSNotEnabled", err)
	}
	if result != nil {
		t.Error("EnsureStream() should return nil")
	}

	// GetStreamInfo returns error
	info, err := si.GetStreamInfo(ctx)
	if !errors.Is(err, ErrNATSNotEnabled) {
		t.Errorf("GetStreamInfo() error = %v, want ErrNATSNotEnabled", err)
	}
	if info != nil {
		t.Error("GetStreamInfo() should return nil")
	}

	// IsHealthy returns false
	if si.IsHealthy(ctx) {
		t.Error("IsHealthy() should return false")
	}

	// Config returns empty config
	cfg := si.Config()
	if cfg.Name != "" {
		t.Errorf("Config().Name = %q, want empty", cfg.Name)
	}
}

// =============================================================================
// Handlers Stub Tests
// =============================================================================

func TestHandlersStub_Config(t *testing.T) {
	t.Parallel()

	cfg := DefaultDuckDBHandlerConfig()

	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"EnableCrossSourceDedup", cfg.EnableCrossSourceDedup, true},
		{"DeduplicationWindow", cfg.DeduplicationWindow, 5 * time.Minute},
		{"MaxDeduplicationEntries", cfg.MaxDeduplicationEntries, 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestHandlersStub_DuckDBHandler(t *testing.T) {
	t.Parallel()

	handler := &DuckDBHandler{}

	stats := handler.Stats()
	if stats.MessagesReceived != 0 {
		t.Errorf("Stats().MessagesReceived = %d, want 0", stats.MessagesReceived)
	}

	handler.StartCleanup(context.Background()) // Should not panic
}

func TestHandlersStub_WebSocketHandler(t *testing.T) {
	t.Parallel()

	handler := &WebSocketHandler{}

	stats := handler.Stats()
	if stats.MessagesReceived != 0 {
		t.Errorf("Stats().MessagesReceived = %d, want 0", stats.MessagesReceived)
	}
}
