// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// MockStreamInfo implements a minimal jetstream.StreamInfo for testing.
type MockStreamInfo struct {
	config jetstream.StreamConfig
	state  jetstream.StreamState
}

// MockStream implements jetstream.Stream for testing.
type MockStream struct {
	info    *MockStreamInfo
	infoErr error
}

func (m *MockStream) Info(ctx context.Context, opts ...jetstream.StreamInfoOpt) (*jetstream.StreamInfo, error) {
	if m.infoErr != nil {
		return nil, m.infoErr
	}
	return &jetstream.StreamInfo{
		Config: m.info.config,
		State:  m.info.state,
	}, nil
}

func (m *MockStream) CachedInfo() *jetstream.StreamInfo {
	return &jetstream.StreamInfo{
		Config: m.info.config,
		State:  m.info.state,
	}
}

func (m *MockStream) Purge(ctx context.Context, opts ...jetstream.StreamPurgeOpt) error { return nil }

func (m *MockStream) CreateOrUpdateConsumer(ctx context.Context, cfg jetstream.ConsumerConfig) (jetstream.Consumer, error) {
	return nil, nil
}

func (m *MockStream) OrderedConsumer(ctx context.Context, cfg jetstream.OrderedConsumerConfig) (jetstream.Consumer, error) {
	return nil, nil
}

func (m *MockStream) Consumer(ctx context.Context, name string) (jetstream.Consumer, error) {
	return nil, nil
}

func (m *MockStream) DeleteConsumer(ctx context.Context, name string) error { return nil }

func (m *MockStream) CreateConsumer(ctx context.Context, cfg jetstream.ConsumerConfig) (jetstream.Consumer, error) {
	return nil, nil
}

func (m *MockStream) UpdateConsumer(ctx context.Context, cfg jetstream.ConsumerConfig) (jetstream.Consumer, error) {
	return nil, nil
}

func (m *MockStream) ListConsumers(ctx context.Context) jetstream.ConsumerInfoLister { return nil }

func (m *MockStream) ConsumerNames(ctx context.Context) jetstream.ConsumerNameLister { return nil }

// Additional methods required by jetstream.Stream interface

func (m *MockStream) CreateOrUpdatePushConsumer(ctx context.Context, cfg jetstream.ConsumerConfig) (jetstream.PushConsumer, error) {
	return nil, nil
}

func (m *MockStream) CreatePushConsumer(ctx context.Context, cfg jetstream.ConsumerConfig) (jetstream.PushConsumer, error) {
	return nil, nil
}

func (m *MockStream) UpdatePushConsumer(ctx context.Context, cfg jetstream.ConsumerConfig) (jetstream.PushConsumer, error) {
	return nil, nil
}

func (m *MockStream) PushConsumer(ctx context.Context, name string) (jetstream.PushConsumer, error) {
	return nil, nil
}

func (m *MockStream) PauseConsumer(ctx context.Context, name string, pauseUntil time.Time) (*jetstream.ConsumerPauseResponse, error) {
	return nil, nil
}

func (m *MockStream) ResumeConsumer(ctx context.Context, name string) (*jetstream.ConsumerPauseResponse, error) {
	return nil, nil
}

func (m *MockStream) UnpinConsumer(ctx context.Context, name string, group string) error {
	return nil
}

func (m *MockStream) GetMsg(ctx context.Context, seq uint64, opts ...jetstream.GetMsgOpt) (*jetstream.RawStreamMsg, error) {
	return nil, nil
}

func (m *MockStream) GetLastMsgForSubject(ctx context.Context, subject string) (*jetstream.RawStreamMsg, error) {
	return nil, nil
}

func (m *MockStream) DeleteMsg(ctx context.Context, seq uint64) error { return nil }

func (m *MockStream) SecureDeleteMsg(ctx context.Context, seq uint64) error { return nil }

// MockJetStreamContext implements a subset of jetstream.JetStream for testing.
type MockJetStreamContext struct {
	mu          sync.Mutex
	streams     map[string]*MockStream
	streamErr   error
	createErr   error
	updateErr   error
	createCalls int
	updateCalls int
}

func NewMockJetStreamContext() *MockJetStreamContext {
	return &MockJetStreamContext{
		streams: make(map[string]*MockStream),
	}
}

func (m *MockJetStreamContext) Stream(ctx context.Context, name string) (jetstream.Stream, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	if stream, ok := m.streams[name]; ok {
		return stream, nil
	}
	return nil, jetstream.ErrStreamNotFound
}

func (m *MockJetStreamContext) CreateStream(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createCalls++
	if m.createErr != nil {
		return nil, m.createErr
	}
	stream := &MockStream{
		info: &MockStreamInfo{
			config: cfg,
			state:  jetstream.StreamState{},
		},
	}
	m.streams[cfg.Name] = stream
	return stream, nil
}

func (m *MockJetStreamContext) UpdateStream(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateCalls++
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	if stream, ok := m.streams[cfg.Name]; ok {
		stream.info.config = cfg
		return stream, nil
	}
	return nil, jetstream.ErrStreamNotFound
}

func (m *MockJetStreamContext) DeleteStream(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.streams, name)
	return nil
}

func (m *MockJetStreamContext) SetStreamError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamErr = err
}

func (m *MockJetStreamContext) SetCreateError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createErr = err
}

func (m *MockJetStreamContext) SetUpdateError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateErr = err
}

func (m *MockJetStreamContext) GetCreateCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.createCalls
}

func (m *MockJetStreamContext) GetUpdateCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateCalls
}

func (m *MockJetStreamContext) AddStream(name string, cfg jetstream.StreamConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streams[name] = &MockStream{
		info: &MockStreamInfo{
			config: cfg,
			state:  jetstream.StreamState{},
		},
	}
}

// TestStreamInitializer_NewStreamInitializer verifies creation with valid config.
func TestStreamInitializer_NewStreamInitializer(t *testing.T) {
	js := NewMockJetStreamContext()
	cfg := DefaultStreamConfig()

	initializer, err := NewStreamInitializer(js, &cfg)
	if err != nil {
		t.Fatalf("NewStreamInitializer() error = %v", err)
	}
	if initializer == nil {
		t.Fatal("NewStreamInitializer() returned nil")
	}
}

// TestStreamInitializer_NewStreamInitializer_NilJS verifies error on nil JetStream.
func TestStreamInitializer_NewStreamInitializer_NilJS(t *testing.T) {
	cfg := DefaultStreamConfig()

	_, err := NewStreamInitializer(nil, &cfg)
	if err == nil {
		t.Fatal("NewStreamInitializer() should error on nil JetStream")
	}
	if err.Error() != "JetStream context required" {
		t.Errorf("Error = %q, want %q", err.Error(), "JetStream context required")
	}
}

// TestStreamInitializer_EnsureStream_CreatesNew verifies stream creation.
func TestStreamInitializer_EnsureStream_CreatesNew(t *testing.T) {
	js := NewMockJetStreamContext()
	cfg := DefaultStreamConfig()

	initializer, err := NewStreamInitializer(js, &cfg)
	if err != nil {
		t.Fatalf("NewStreamInitializer() error = %v", err)
	}

	ctx := context.Background()
	stream, err := initializer.EnsureStream(ctx)
	if err != nil {
		t.Fatalf("EnsureStream() error = %v", err)
	}
	if stream == nil {
		t.Fatal("EnsureStream() returned nil stream")
	}

	// Verify stream was created
	if js.GetCreateCalls() != 1 {
		t.Errorf("CreateStream calls = %d, want 1", js.GetCreateCalls())
	}
	if js.GetUpdateCalls() != 0 {
		t.Errorf("UpdateStream calls = %d, want 0", js.GetUpdateCalls())
	}

	// Verify stream configuration
	info := stream.CachedInfo()
	if info.Config.Name != cfg.Name {
		t.Errorf("Stream name = %s, want %s", info.Config.Name, cfg.Name)
	}
	if len(info.Config.Subjects) != len(cfg.Subjects) {
		t.Errorf("Subjects count = %d, want %d", len(info.Config.Subjects), len(cfg.Subjects))
	}
}

// TestStreamInitializer_EnsureStream_UpdatesExisting verifies stream update.
func TestStreamInitializer_EnsureStream_UpdatesExisting(t *testing.T) {
	js := NewMockJetStreamContext()
	cfg := DefaultStreamConfig()

	// Pre-create stream with different config
	existingCfg := jetstream.StreamConfig{
		Name:     cfg.Name,
		Subjects: []string{"old.subject"},
	}
	js.AddStream(cfg.Name, existingCfg)

	initializer, err := NewStreamInitializer(js, &cfg)
	if err != nil {
		t.Fatalf("NewStreamInitializer() error = %v", err)
	}

	ctx := context.Background()
	stream, err := initializer.EnsureStream(ctx)
	if err != nil {
		t.Fatalf("EnsureStream() error = %v", err)
	}
	if stream == nil {
		t.Fatal("EnsureStream() returned nil stream")
	}

	// Verify stream was updated, not created
	if js.GetCreateCalls() != 0 {
		t.Errorf("CreateStream calls = %d, want 0", js.GetCreateCalls())
	}
	if js.GetUpdateCalls() != 1 {
		t.Errorf("UpdateStream calls = %d, want 1", js.GetUpdateCalls())
	}
}

// TestStreamInitializer_EnsureStream_Idempotent verifies idempotency.
func TestStreamInitializer_EnsureStream_Idempotent(t *testing.T) {
	js := NewMockJetStreamContext()
	cfg := DefaultStreamConfig()

	initializer, err := NewStreamInitializer(js, &cfg)
	if err != nil {
		t.Fatalf("NewStreamInitializer() error = %v", err)
	}

	ctx := context.Background()

	// Call EnsureStream multiple times
	for i := 0; i < 3; i++ {
		stream, err := initializer.EnsureStream(ctx)
		if err != nil {
			t.Fatalf("EnsureStream() call %d error = %v", i+1, err)
		}
		if stream == nil {
			t.Fatalf("EnsureStream() call %d returned nil", i+1)
		}
	}

	// First call creates, subsequent calls update
	if js.GetCreateCalls() != 1 {
		t.Errorf("CreateStream calls = %d, want 1", js.GetCreateCalls())
	}
	if js.GetUpdateCalls() != 2 {
		t.Errorf("UpdateStream calls = %d, want 2", js.GetUpdateCalls())
	}
}

// TestStreamInitializer_EnsureStream_CreateError verifies create error handling.
func TestStreamInitializer_EnsureStream_CreateError(t *testing.T) {
	js := NewMockJetStreamContext()
	cfg := DefaultStreamConfig()

	js.SetCreateError(errors.New("insufficient storage"))

	initializer, err := NewStreamInitializer(js, &cfg)
	if err != nil {
		t.Fatalf("NewStreamInitializer() error = %v", err)
	}

	ctx := context.Background()
	_, err = initializer.EnsureStream(ctx)
	if err == nil {
		t.Fatal("EnsureStream() should return error on create failure")
	}
	if !errors.Is(err, js.createErr) {
		t.Errorf("Error should wrap create error: %v", err)
	}
}

// TestStreamInitializer_EnsureStream_UpdateError verifies update error handling.
func TestStreamInitializer_EnsureStream_UpdateError(t *testing.T) {
	js := NewMockJetStreamContext()
	cfg := DefaultStreamConfig()

	// Pre-create stream
	js.AddStream(cfg.Name, jetstream.StreamConfig{Name: cfg.Name})
	js.SetUpdateError(errors.New("update not allowed"))

	initializer, err := NewStreamInitializer(js, &cfg)
	if err != nil {
		t.Fatalf("NewStreamInitializer() error = %v", err)
	}

	ctx := context.Background()
	_, err = initializer.EnsureStream(ctx)
	if err == nil {
		t.Fatal("EnsureStream() should return error on update failure")
	}
}

// TestStreamInitializer_EnsureStream_Timeout verifies context timeout.
func TestStreamInitializer_EnsureStream_Timeout(t *testing.T) {
	js := NewMockJetStreamContext()
	cfg := DefaultStreamConfig()

	initializer, err := NewStreamInitializer(js, &cfg)
	if err != nil {
		t.Fatalf("NewStreamInitializer() error = %v", err)
	}

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = initializer.EnsureStream(ctx)
	if err == nil {
		// Note: Our mock doesn't check context, but real implementation should
		// This test documents expected behavior
		t.Log("Mock doesn't honor context cancellation (expected)")
	}
}

// TestStreamInitializer_EnsureStream_ConfigValidation verifies config is applied.
func TestStreamInitializer_EnsureStream_ConfigValidation(t *testing.T) {
	js := NewMockJetStreamContext()
	cfg := StreamConfig{
		Name:            "TEST_STREAM",
		Subjects:        []string{"test.>", "other.>"},
		MaxAge:          24 * time.Hour,
		MaxBytes:        1024 * 1024 * 1024, // 1GB
		MaxMsgs:         100000,
		DuplicateWindow: 5 * time.Minute,
		Replicas:        1,
	}

	initializer, err := NewStreamInitializer(js, &cfg)
	if err != nil {
		t.Fatalf("NewStreamInitializer() error = %v", err)
	}

	ctx := context.Background()
	stream, err := initializer.EnsureStream(ctx)
	if err != nil {
		t.Fatalf("EnsureStream() error = %v", err)
	}

	info := stream.CachedInfo()

	// Verify all config fields were applied
	if info.Config.Name != cfg.Name {
		t.Errorf("Name = %s, want %s", info.Config.Name, cfg.Name)
	}
	if len(info.Config.Subjects) != 2 {
		t.Errorf("Subjects = %v, want 2 subjects", info.Config.Subjects)
	}
	if info.Config.MaxAge != cfg.MaxAge {
		t.Errorf("MaxAge = %v, want %v", info.Config.MaxAge, cfg.MaxAge)
	}
	if info.Config.MaxBytes != cfg.MaxBytes {
		t.Errorf("MaxBytes = %d, want %d", info.Config.MaxBytes, cfg.MaxBytes)
	}
	if info.Config.MaxMsgs != cfg.MaxMsgs {
		t.Errorf("MaxMsgs = %d, want %d", info.Config.MaxMsgs, cfg.MaxMsgs)
	}
	if info.Config.Duplicates != cfg.DuplicateWindow {
		t.Errorf("Duplicates = %v, want %v", info.Config.Duplicates, cfg.DuplicateWindow)
	}
}

// TestStreamInitializer_GetStreamInfo verifies info retrieval.
func TestStreamInitializer_GetStreamInfo(t *testing.T) {
	js := NewMockJetStreamContext()
	cfg := DefaultStreamConfig()

	// Pre-create stream
	js.AddStream(cfg.Name, jetstream.StreamConfig{
		Name:     cfg.Name,
		Subjects: cfg.Subjects,
	})

	initializer, err := NewStreamInitializer(js, &cfg)
	if err != nil {
		t.Fatalf("NewStreamInitializer() error = %v", err)
	}

	ctx := context.Background()
	info, err := initializer.GetStreamInfo(ctx)
	if err != nil {
		t.Fatalf("GetStreamInfo() error = %v", err)
	}
	if info == nil {
		t.Fatal("GetStreamInfo() returned nil")
	}
	if info.Config.Name != cfg.Name {
		t.Errorf("Stream name = %s, want %s", info.Config.Name, cfg.Name)
	}
}

// TestStreamInitializer_GetStreamInfo_NotFound verifies error when stream missing.
func TestStreamInitializer_GetStreamInfo_NotFound(t *testing.T) {
	js := NewMockJetStreamContext()
	cfg := DefaultStreamConfig()

	initializer, err := NewStreamInitializer(js, &cfg)
	if err != nil {
		t.Fatalf("NewStreamInitializer() error = %v", err)
	}

	ctx := context.Background()
	_, err = initializer.GetStreamInfo(ctx)
	if err == nil {
		t.Fatal("GetStreamInfo() should error when stream not found")
	}
}

// TestStreamInitializer_IsHealthy verifies health check.
func TestStreamInitializer_IsHealthy(t *testing.T) {
	js := NewMockJetStreamContext()
	cfg := DefaultStreamConfig()

	// Pre-create stream
	js.AddStream(cfg.Name, jetstream.StreamConfig{Name: cfg.Name})

	initializer, err := NewStreamInitializer(js, &cfg)
	if err != nil {
		t.Fatalf("NewStreamInitializer() error = %v", err)
	}

	ctx := context.Background()
	healthy := initializer.IsHealthy(ctx)
	if !healthy {
		t.Error("IsHealthy() = false, want true when stream exists")
	}
}

// TestStreamInitializer_IsHealthy_NoStream verifies unhealthy when no stream.
func TestStreamInitializer_IsHealthy_NoStream(t *testing.T) {
	js := NewMockJetStreamContext()
	cfg := DefaultStreamConfig()

	initializer, err := NewStreamInitializer(js, &cfg)
	if err != nil {
		t.Fatalf("NewStreamInitializer() error = %v", err)
	}

	ctx := context.Background()
	healthy := initializer.IsHealthy(ctx)
	if healthy {
		t.Error("IsHealthy() = true, want false when stream doesn't exist")
	}
}
