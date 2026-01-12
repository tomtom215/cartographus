// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package recommend

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// mockDataProvider implements DataProvider for testing.
type mockDataProvider struct {
	interactions       []Interaction
	items              []Item
	userHistory        map[int][]int
	candidates         map[int][]int
	interactionsErr    error
	itemsErr           error
	userHistoryErr     error
	candidatesErr      error
	getCandidatesCalls int32
}

func (m *mockDataProvider) GetInteractions(ctx context.Context, since time.Time) ([]Interaction, error) {
	if m.interactionsErr != nil {
		return nil, m.interactionsErr
	}
	return m.interactions, nil
}

func (m *mockDataProvider) GetItems(ctx context.Context) ([]Item, error) {
	if m.itemsErr != nil {
		return nil, m.itemsErr
	}
	return m.items, nil
}

func (m *mockDataProvider) GetUserHistory(ctx context.Context, userID int) ([]int, error) {
	if m.userHistoryErr != nil {
		return nil, m.userHistoryErr
	}
	if m.userHistory == nil {
		return []int{}, nil
	}
	return m.userHistory[userID], nil
}

func (m *mockDataProvider) GetCandidates(ctx context.Context, userID int, limit int) ([]int, error) {
	atomic.AddInt32(&m.getCandidatesCalls, 1)
	if m.candidatesErr != nil {
		return nil, m.candidatesErr
	}
	if m.candidates == nil {
		return []int{}, nil
	}
	candidates := m.candidates[userID]
	if len(candidates) > limit {
		return candidates[:limit], nil
	}
	return candidates, nil
}

// mockAlgorithm implements Algorithm for testing.
type mockAlgorithm struct {
	name          string
	trained       bool
	version       int
	lastTrainedAt time.Time
	trainErr      error
	predictScores map[int]float64
	predictErr    error
	similarScores map[int]float64
	similarErr    error
	trainDelay    time.Duration
	predictDelay  time.Duration
	mu            sync.RWMutex
}

func newMockAlgorithm(name string) *mockAlgorithm {
	return &mockAlgorithm{
		name:          name,
		predictScores: make(map[int]float64),
		similarScores: make(map[int]float64),
	}
}

func (m *mockAlgorithm) Name() string {
	return m.name
}

func (m *mockAlgorithm) Train(ctx context.Context, interactions []Interaction, items []Item) error {
	if m.trainDelay > 0 {
		select {
		case <-time.After(m.trainDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if m.trainErr != nil {
		return m.trainErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trained = true
	m.version++
	m.lastTrainedAt = time.Now()
	return nil
}

func (m *mockAlgorithm) Predict(ctx context.Context, userID int, candidates []int) (map[int]float64, error) {
	if m.predictDelay > 0 {
		select {
		case <-time.After(m.predictDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if m.predictErr != nil {
		return nil, m.predictErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.predictScores, nil
}

func (m *mockAlgorithm) PredictSimilar(ctx context.Context, itemID int, candidates []int) (map[int]float64, error) {
	if m.similarErr != nil {
		return nil, m.similarErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.similarScores, nil
}

func (m *mockAlgorithm) IsTrained() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.trained
}

func (m *mockAlgorithm) Version() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.version
}

func (m *mockAlgorithm) LastTrainedAt() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastTrainedAt
}

// mockReranker implements Reranker for testing.
type mockReranker struct {
	name  string
	calls int32
}

func (m *mockReranker) Name() string {
	return m.name
}

func (m *mockReranker) Rerank(ctx context.Context, items []ScoredItem, k int) []ScoredItem {
	atomic.AddInt32(&m.calls, 1)
	return items
}

// testLogger returns a zerolog logger for testing.
func testLogger() zerolog.Logger {
	return zerolog.Nop()
}

// --- Test: NewEngine ---

func TestNewEngine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "nil config uses defaults",
			cfg:     nil,
			wantErr: false,
		},
		{
			name:    "valid default config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid config returns error",
			cfg: &Config{
				EASE: EASEConfig{Lambda: -1}, // Invalid
				Limits: LimitsConfig{
					DefaultK:          10,
					MaxK:              100,
					MaxCandidates:     1000,
					PredictionTimeout: 5 * time.Second,
				},
				Training: TrainingConfig{
					Timeout: 10 * time.Minute,
				},
			},
			wantErr: true,
		},
		{
			name: "zero seed uses default",
			cfg: func() *Config {
				c := DefaultConfig()
				c.Seed = 0
				return c
			}(),
			wantErr: false,
		},
		{
			name: "custom seed is used",
			cfg: func() *Config {
				c := DefaultConfig()
				c.Seed = 12345
				return c
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			engine, err := NewEngine(tt.cfg, testLogger())

			if tt.wantErr {
				if err == nil {
					t.Error("NewEngine() = nil error, want error")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewEngine() error = %v, want nil", err)
			}

			if engine == nil {
				t.Fatal("NewEngine() = nil, want non-nil")
			}

			// Verify engine is properly initialized
			if engine.config == nil {
				t.Error("engine.config = nil, want non-nil")
			}
			if engine.cache == nil {
				t.Error("engine.cache = nil, want non-nil")
			}
			if engine.algorithms == nil {
				t.Error("engine.algorithms = nil, want non-nil")
			}
			if engine.rerankers == nil {
				t.Error("engine.rerankers = nil, want non-nil")
			}
		})
	}
}

// --- Test: SetDataProvider ---

func TestEngine_SetDataProvider(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine(nil, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if engine.dataProvider != nil {
		t.Error("initial dataProvider should be nil")
	}

	dp := &mockDataProvider{}
	engine.SetDataProvider(dp)

	if engine.dataProvider != dp {
		t.Error("SetDataProvider() did not set the provider")
	}
}

// --- Test: RegisterAlgorithm ---

func TestEngine_RegisterAlgorithm(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine(nil, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if len(engine.algorithms) != 0 {
		t.Errorf("initial algorithms count = %d, want 0", len(engine.algorithms))
	}

	alg1 := newMockAlgorithm("test-alg-1")
	alg2 := newMockAlgorithm("test-alg-2")

	engine.RegisterAlgorithm(alg1)
	if len(engine.algorithms) != 1 {
		t.Errorf("after first register, algorithms count = %d, want 1", len(engine.algorithms))
	}

	engine.RegisterAlgorithm(alg2)
	if len(engine.algorithms) != 2 {
		t.Errorf("after second register, algorithms count = %d, want 2", len(engine.algorithms))
	}
}

// --- Test: RegisterReranker ---

func TestEngine_RegisterReranker(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine(nil, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if len(engine.rerankers) != 0 {
		t.Errorf("initial rerankers count = %d, want 0", len(engine.rerankers))
	}

	rr1 := &mockReranker{name: "test-rr-1"}
	rr2 := &mockReranker{name: "test-rr-2"}

	engine.RegisterReranker(rr1)
	if len(engine.rerankers) != 1 {
		t.Errorf("after first register, rerankers count = %d, want 1", len(engine.rerankers))
	}

	engine.RegisterReranker(rr2)
	if len(engine.rerankers) != 2 {
		t.Errorf("after second register, rerankers count = %d, want 2", len(engine.rerankers))
	}
}

// --- Test: Recommend ---

func TestEngine_Recommend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setupEngine  func() *Engine
		request      Request
		wantErr      bool
		wantItems    int
		wantCacheHit bool
	}{
		{
			name: "successful recommendation",
			setupEngine: func() *Engine {
				engine, _ := NewEngine(nil, testLogger())
				alg := newMockAlgorithm("ease")
				alg.trained = true
				alg.predictScores = map[int]float64{1: 0.9, 2: 0.8, 3: 0.7}
				engine.RegisterAlgorithm(alg)
				engine.SetDataProvider(&mockDataProvider{
					candidates: map[int][]int{1: {1, 2, 3, 4, 5}},
				})
				return engine
			},
			request:      Request{UserID: 1, K: 3},
			wantErr:      false,
			wantItems:    3,
			wantCacheHit: false,
		},
		{
			name: "no data provider returns error",
			setupEngine: func() *Engine {
				engine, _ := NewEngine(nil, testLogger())
				alg := newMockAlgorithm("ease")
				alg.trained = true
				engine.RegisterAlgorithm(alg)
				// No data provider set
				return engine
			},
			request: Request{UserID: 1, K: 5},
			wantErr: true,
		},
		{
			name: "no algorithms returns error",
			setupEngine: func() *Engine {
				engine, _ := NewEngine(nil, testLogger())
				engine.SetDataProvider(&mockDataProvider{
					candidates: map[int][]int{1: {1, 2, 3}},
				})
				// No algorithms registered
				return engine
			},
			request: Request{UserID: 1, K: 5},
			wantErr: true,
		},
		{
			name: "no candidates returns empty response",
			setupEngine: func() *Engine {
				engine, _ := NewEngine(nil, testLogger())
				alg := newMockAlgorithm("ease")
				alg.trained = true
				engine.RegisterAlgorithm(alg)
				engine.SetDataProvider(&mockDataProvider{
					candidates: map[int][]int{1: {}}, // Empty candidates
				})
				return engine
			},
			request:   Request{UserID: 1, K: 5},
			wantErr:   false,
			wantItems: 0,
		},
		{
			name: "K clamped to MaxK",
			setupEngine: func() *Engine {
				cfg := DefaultConfig()
				cfg.Limits.DefaultK = 3 // Must be <= MaxK for validation to pass
				cfg.Limits.MaxK = 5
				engine, err := NewEngine(cfg, testLogger())
				if err != nil {
					panic("failed to create engine: " + err.Error())
				}
				alg := newMockAlgorithm("ease")
				alg.trained = true
				alg.predictScores = map[int]float64{1: 0.9, 2: 0.8, 3: 0.7, 4: 0.6, 5: 0.5, 6: 0.4}
				engine.RegisterAlgorithm(alg)
				engine.SetDataProvider(&mockDataProvider{
					candidates: map[int][]int{1: {1, 2, 3, 4, 5, 6, 7, 8}},
				})
				return engine
			},
			request:   Request{UserID: 1, K: 100}, // Request more than MaxK
			wantErr:   false,
			wantItems: 5, // Should be clamped to MaxK
		},
		{
			name: "K defaults to DefaultK when zero",
			setupEngine: func() *Engine {
				cfg := DefaultConfig()
				cfg.Limits.DefaultK = 3
				engine, _ := NewEngine(cfg, testLogger())
				alg := newMockAlgorithm("ease")
				alg.trained = true
				alg.predictScores = map[int]float64{1: 0.9, 2: 0.8, 3: 0.7, 4: 0.6}
				engine.RegisterAlgorithm(alg)
				engine.SetDataProvider(&mockDataProvider{
					candidates: map[int][]int{1: {1, 2, 3, 4}},
				})
				return engine
			},
			request:   Request{UserID: 1, K: 0}, // Zero K
			wantErr:   false,
			wantItems: 3, // Should use DefaultK
		},
		{
			name: "candidate error propagates",
			setupEngine: func() *Engine {
				cfg := DefaultConfig()
				engine, _ := NewEngine(cfg, testLogger())
				alg := newMockAlgorithm("ease")
				alg.trained = true
				engine.RegisterAlgorithm(alg)
				engine.SetDataProvider(&mockDataProvider{
					candidatesErr: errors.New("database error"),
				})
				return engine
			},
			request: Request{UserID: 1, K: 5},
			wantErr: true,
		},
		{
			name: "exclude IDs are respected",
			setupEngine: func() *Engine {
				engine, _ := NewEngine(nil, testLogger())
				alg := newMockAlgorithm("ease")
				alg.trained = true
				alg.predictScores = map[int]float64{1: 0.9, 2: 0.8}
				engine.RegisterAlgorithm(alg)
				engine.SetDataProvider(&mockDataProvider{
					candidates:  map[int][]int{1: {1, 2, 3, 4, 5}},
					userHistory: map[int][]int{1: {3, 4}}, // 3, 4 should be excluded
				})
				return engine
			},
			request:   Request{UserID: 1, K: 5, ExcludeIDs: []int{5}}, // Also exclude 5
			wantErr:   false,
			wantItems: 2, // Only 1 and 2 should remain
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			engine := tt.setupEngine()

			resp, err := engine.Recommend(context.Background(), tt.request)

			if tt.wantErr {
				if err == nil {
					t.Error("Recommend() = nil error, want error")
				}
				return
			}

			if err != nil {
				t.Fatalf("Recommend() error = %v, want nil", err)
			}

			if resp == nil {
				t.Fatal("Recommend() = nil response, want non-nil")
			}

			if len(resp.Items) != tt.wantItems {
				t.Errorf("Recommend() returned %d items, want %d", len(resp.Items), tt.wantItems)
			}

			if resp.Metadata.CacheHit != tt.wantCacheHit {
				t.Errorf("Recommend() cache hit = %v, want %v", resp.Metadata.CacheHit, tt.wantCacheHit)
			}

			// Verify request ID is set
			if resp.Metadata.RequestID == "" {
				t.Error("Recommend() response has empty RequestID")
			}
		})
	}
}

func TestEngine_Recommend_CacheHit(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.Cache.Enabled = true
	cfg.Cache.TTL = 5 * time.Minute
	engine, err := NewEngine(cfg, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	alg := newMockAlgorithm("ease")
	alg.trained = true
	alg.predictScores = map[int]float64{1: 0.9, 2: 0.8, 3: 0.7}
	engine.RegisterAlgorithm(alg)

	dp := &mockDataProvider{
		candidates: map[int][]int{1: {1, 2, 3, 4, 5}},
	}
	engine.SetDataProvider(dp)

	req := Request{UserID: 1, K: 3}

	// First request - cache miss
	resp1, err := engine.Recommend(context.Background(), req)
	if err != nil {
		t.Fatalf("First Recommend() error = %v", err)
	}
	if resp1.Metadata.CacheHit {
		t.Error("First request should be cache miss")
	}
	firstCalls := atomic.LoadInt32(&dp.getCandidatesCalls)

	// Second request - cache hit
	resp2, err := engine.Recommend(context.Background(), req)
	if err != nil {
		t.Fatalf("Second Recommend() error = %v", err)
	}
	if !resp2.Metadata.CacheHit {
		t.Error("Second request should be cache hit")
	}
	secondCalls := atomic.LoadInt32(&dp.getCandidatesCalls)

	// Should not have called GetCandidates again
	if secondCalls != firstCalls {
		t.Errorf("Cache hit should not call GetCandidates, calls = %d, want %d", secondCalls, firstCalls)
	}
}

func TestEngine_Recommend_CacheDisabled(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.Cache.Enabled = false
	engine, err := NewEngine(cfg, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	alg := newMockAlgorithm("ease")
	alg.trained = true
	alg.predictScores = map[int]float64{1: 0.9}
	engine.RegisterAlgorithm(alg)

	dp := &mockDataProvider{
		candidates: map[int][]int{1: {1, 2, 3}},
	}
	engine.SetDataProvider(dp)

	req := Request{UserID: 1, K: 3}

	// First request
	_, _ = engine.Recommend(context.Background(), req)
	firstCalls := atomic.LoadInt32(&dp.getCandidatesCalls)

	// Second request - should not use cache
	_, _ = engine.Recommend(context.Background(), req)
	secondCalls := atomic.LoadInt32(&dp.getCandidatesCalls)

	if secondCalls != firstCalls+1 {
		t.Errorf("Cache disabled should call GetCandidates again, calls = %d, want %d", secondCalls, firstCalls+1)
	}
}

func TestEngine_Recommend_SimilarMode(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine(nil, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	alg := newMockAlgorithm("ease")
	alg.trained = true
	alg.similarScores = map[int]float64{2: 0.9, 3: 0.8}
	engine.RegisterAlgorithm(alg)

	engine.SetDataProvider(&mockDataProvider{
		candidates: map[int][]int{1: {2, 3, 4, 5}},
	})

	req := Request{
		UserID:        1,
		K:             3,
		Mode:          ModeSimilar,
		CurrentItemID: 1,
	}

	resp, err := engine.Recommend(context.Background(), req)
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}

	if resp.Metadata.Mode != "similar" {
		t.Errorf("Response mode = %q, want 'similar'", resp.Metadata.Mode)
	}
}

func TestEngine_Recommend_Rerankers(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine(nil, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	alg := newMockAlgorithm("ease")
	alg.trained = true
	alg.predictScores = map[int]float64{1: 0.9, 2: 0.8}
	engine.RegisterAlgorithm(alg)

	rr := &mockReranker{name: "test-rr"}
	engine.RegisterReranker(rr)

	engine.SetDataProvider(&mockDataProvider{
		candidates: map[int][]int{1: {1, 2, 3}},
	})

	req := Request{UserID: 1, K: 3}

	_, err = engine.Recommend(context.Background(), req)
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}

	if atomic.LoadInt32(&rr.calls) != 1 {
		t.Errorf("Reranker was called %d times, want 1", rr.calls)
	}
}

func TestEngine_Recommend_AlgorithmError(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine(nil, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	// Add a failing algorithm (using "als" which has a weight in config)
	failAlg := newMockAlgorithm("als")
	failAlg.trained = true
	failAlg.predictErr = errors.New("prediction error")
	engine.RegisterAlgorithm(failAlg)

	// Add a working algorithm (using "ease" which has a weight in config)
	workAlg := newMockAlgorithm("ease")
	workAlg.trained = true
	workAlg.predictScores = map[int]float64{1: 0.9, 2: 0.8}
	engine.RegisterAlgorithm(workAlg)

	engine.SetDataProvider(&mockDataProvider{
		candidates: map[int][]int{1: {1, 2, 3}},
	})

	// Should still work with partial algorithm success
	resp, err := engine.Recommend(context.Background(), Request{UserID: 1, K: 3})
	if err != nil {
		t.Fatalf("Recommend() error = %v, want nil (partial success)", err)
	}

	if len(resp.Items) == 0 {
		t.Error("Recommend() should return items from working algorithm")
	}
}

func TestEngine_Recommend_UntrainedAlgorithm(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine(nil, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	// Add an untrained algorithm (using "als" which has a weight in config)
	untrainedAlg := newMockAlgorithm("als")
	untrainedAlg.trained = false
	engine.RegisterAlgorithm(untrainedAlg)

	// Add a trained algorithm (using "ease" which has a weight in config)
	trainedAlg := newMockAlgorithm("ease")
	trainedAlg.trained = true
	trainedAlg.predictScores = map[int]float64{1: 0.9}
	engine.RegisterAlgorithm(trainedAlg)

	engine.SetDataProvider(&mockDataProvider{
		candidates: map[int][]int{1: {1, 2, 3}},
	})

	resp, err := engine.Recommend(context.Background(), Request{UserID: 1, K: 3})
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}

	// Should still work with only trained algorithms
	if len(resp.Metadata.AlgorithmsUsed) == 0 {
		t.Error("Should use at least one trained algorithm")
	}
}

// --- Test: Train ---

func TestEngine_Train(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupEngine func() *Engine
		wantErr     bool
		errContains string
	}{
		{
			name: "successful training",
			setupEngine: func() *Engine {
				cfg := DefaultConfig()
				cfg.Training.MinInteractions = 2
				engine, _ := NewEngine(cfg, testLogger())
				alg := newMockAlgorithm("ease")
				engine.RegisterAlgorithm(alg)
				engine.SetDataProvider(&mockDataProvider{
					interactions: []Interaction{
						{UserID: 1, ItemID: 1, Confidence: 1.0},
						{UserID: 2, ItemID: 2, Confidence: 1.0},
					},
					items: []Item{{ID: 1}, {ID: 2}},
				})
				return engine
			},
			wantErr: false,
		},
		{
			name: "no data provider returns error",
			setupEngine: func() *Engine {
				engine, _ := NewEngine(nil, testLogger())
				alg := newMockAlgorithm("ease")
				engine.RegisterAlgorithm(alg)
				// No data provider set
				return engine
			},
			wantErr:     true,
			errContains: "data provider not set",
		},
		{
			name: "insufficient interactions returns error",
			setupEngine: func() *Engine {
				cfg := DefaultConfig()
				cfg.Training.MinInteractions = 100
				engine, _ := NewEngine(cfg, testLogger())
				alg := newMockAlgorithm("ease")
				engine.RegisterAlgorithm(alg)
				engine.SetDataProvider(&mockDataProvider{
					interactions: []Interaction{{UserID: 1, ItemID: 1}},
					items:        []Item{{ID: 1}},
				})
				return engine
			},
			wantErr:     true,
			errContains: "insufficient interactions",
		},
		{
			name: "interactions error propagates",
			setupEngine: func() *Engine {
				engine, _ := NewEngine(nil, testLogger())
				alg := newMockAlgorithm("ease")
				engine.RegisterAlgorithm(alg)
				engine.SetDataProvider(&mockDataProvider{
					interactionsErr: errors.New("database error"),
				})
				return engine
			},
			wantErr:     true,
			errContains: "get interactions",
		},
		{
			name: "items error propagates",
			setupEngine: func() *Engine {
				cfg := DefaultConfig()
				cfg.Training.MinInteractions = 1
				engine, _ := NewEngine(cfg, testLogger())
				alg := newMockAlgorithm("ease")
				engine.RegisterAlgorithm(alg)
				engine.SetDataProvider(&mockDataProvider{
					interactions: []Interaction{{UserID: 1, ItemID: 1}},
					itemsErr:     errors.New("database error"),
				})
				return engine
			},
			wantErr:     true,
			errContains: "get items",
		},
		{
			name: "algorithm error is logged but continues",
			setupEngine: func() *Engine {
				cfg := DefaultConfig()
				cfg.Training.MinInteractions = 1
				engine, _ := NewEngine(cfg, testLogger())
				failAlg := newMockAlgorithm("fail")
				failAlg.trainErr = errors.New("train error")
				engine.RegisterAlgorithm(failAlg)
				engine.SetDataProvider(&mockDataProvider{
					interactions: []Interaction{{UserID: 1, ItemID: 1}},
					items:        []Item{{ID: 1}},
				})
				return engine
			},
			wantErr: false, // Training continues despite algorithm error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			engine := tt.setupEngine()

			err := engine.Train(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Error("Train() = nil error, want error")
				} else if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("Train() error = %q, want containing %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("Train() error = %v, want nil", err)
			}

			// Verify training status updated
			status := engine.GetStatus()
			if status.ModelVersion == 0 {
				t.Error("Train() should increment model version")
			}
			if status.LastTrainedAt.IsZero() {
				t.Error("Train() should set LastTrainedAt")
			}
		})
	}
}

func TestEngine_Train_AlreadyInProgress(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.Training.MinInteractions = 1
	cfg.Training.Timeout = 10 * time.Second
	engine, err := NewEngine(cfg, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	slowAlg := newMockAlgorithm("ease") // Use a name that has weight in config
	slowAlg.trainDelay = 500 * time.Millisecond
	engine.RegisterAlgorithm(slowAlg)

	engine.SetDataProvider(&mockDataProvider{
		interactions: []Interaction{{UserID: 1, ItemID: 1}},
		items:        []Item{{ID: 1}},
	})

	var wg sync.WaitGroup
	var firstErr, secondErr error

	// Start first training
	wg.Add(1)
	go func() {
		defer wg.Done()
		firstErr = engine.Train(context.Background())
	}()

	// Give time for first training to acquire lock and start
	time.Sleep(100 * time.Millisecond)

	// Try second training (should fail)
	wg.Add(1)
	go func() {
		defer wg.Done()
		secondErr = engine.Train(context.Background())
	}()

	wg.Wait()

	if firstErr != nil {
		t.Errorf("First Train() error = %v, want nil", firstErr)
	}
	if secondErr == nil {
		t.Error("Second Train() = nil error, want 'already in progress' error")
	}
}

func TestEngine_Train_ClearsCache(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.Cache.Enabled = true
	cfg.Cache.InvalidateOnTrain = true
	cfg.Training.MinInteractions = 1
	engine, err := NewEngine(cfg, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	alg := newMockAlgorithm("ease")
	alg.trained = true
	alg.predictScores = map[int]float64{1: 0.9}
	engine.RegisterAlgorithm(alg)

	engine.SetDataProvider(&mockDataProvider{
		candidates:   map[int][]int{1: {1, 2, 3}},
		interactions: []Interaction{{UserID: 1, ItemID: 1}},
		items:        []Item{{ID: 1}},
	})

	// Populate cache
	_, _ = engine.Recommend(context.Background(), Request{UserID: 1, K: 3})
	if len(engine.cache) == 0 {
		t.Fatal("Cache should have entries after recommendation")
	}

	// Train should clear cache
	if err := engine.Train(context.Background()); err != nil {
		t.Fatalf("Train() error = %v", err)
	}

	if len(engine.cache) != 0 {
		t.Errorf("Cache should be empty after training, got %d entries", len(engine.cache))
	}
}

func TestEngine_Train_ContextCancellation(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.Training.MinInteractions = 1
	cfg.Training.Timeout = 10 * time.Second
	engine, err := NewEngine(cfg, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	slowAlg := newMockAlgorithm("slow")
	slowAlg.trainDelay = 5 * time.Second
	engine.RegisterAlgorithm(slowAlg)

	engine.SetDataProvider(&mockDataProvider{
		interactions: []Interaction{{UserID: 1, ItemID: 1}},
		items:        []Item{{ID: 1}},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = engine.Train(ctx)
	// Training should complete but algorithm may have been canceled
	// The training function catches algorithm errors and continues
	// so this may or may not error depending on timing
	_ = err
}

// --- Test: GetStatus ---

func TestEngine_GetStatus(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine(nil, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	status := engine.GetStatus()

	if status.IsTraining {
		t.Error("Initial status should not be training")
	}
	if status.ModelVersion != 0 {
		t.Errorf("Initial ModelVersion = %d, want 0", status.ModelVersion)
	}
}

// --- Test: GetMetrics ---

func TestEngine_GetMetrics(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine(nil, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	alg := newMockAlgorithm("ease")
	alg.trained = true
	alg.predictScores = map[int]float64{1: 0.9}
	engine.RegisterAlgorithm(alg)
	engine.SetDataProvider(&mockDataProvider{
		candidates: map[int][]int{1: {1, 2, 3}},
	})

	// Make some requests
	_, _ = engine.Recommend(context.Background(), Request{UserID: 1, K: 3})
	_, _ = engine.Recommend(context.Background(), Request{UserID: 1, K: 3}) // Cache hit

	metrics := engine.GetMetrics()

	if metrics.RequestCount != 2 {
		t.Errorf("RequestCount = %d, want 2", metrics.RequestCount)
	}
	if metrics.CacheHits != 1 {
		t.Errorf("CacheHits = %d, want 1", metrics.CacheHits)
	}
	if metrics.CacheMisses != 1 {
		t.Errorf("CacheMisses = %d, want 1", metrics.CacheMisses)
	}
}

// --- Test: GetConfig ---

func TestEngine_GetConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.EASE.Lambda = 999
	engine, err := NewEngine(cfg, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	gotCfg := engine.GetConfig()

	if gotCfg.EASE.Lambda != 999 {
		t.Errorf("GetConfig().EASE.Lambda = %f, want 999", gotCfg.EASE.Lambda)
	}

	// Verify it's a copy
	gotCfg.EASE.Lambda = 123
	if engine.config.EASE.Lambda == 123 {
		t.Error("GetConfig() should return a copy, not the original")
	}
}

// --- Test: UpdateConfig ---

func TestEngine_UpdateConfig(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine(nil, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "valid config update",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid config returns error",
			cfg: &Config{
				EASE: EASEConfig{Lambda: -1}, // Invalid
				Limits: LimitsConfig{
					DefaultK:          10,
					MaxK:              100,
					MaxCandidates:     1000,
					PredictionTimeout: 5 * time.Second,
				},
				Training: TrainingConfig{
					Timeout: 10 * time.Minute,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.UpdateConfig(tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("UpdateConfig() = nil error, want error")
				}
				return
			}

			if err != nil {
				t.Fatalf("UpdateConfig() error = %v, want nil", err)
			}
		})
	}
}

// --- Test: Cache Operations ---

func TestEngine_CacheOperations(t *testing.T) {
	t.Parallel()

	t.Run("checkCache returns nil for missing key", func(t *testing.T) {
		t.Parallel()
		engine, _ := NewEngine(nil, testLogger())
		if resp := engine.checkCache("nonexistent"); resp != nil {
			t.Error("checkCache() should return nil for missing key")
		}
	})

	t.Run("checkCache returns nil for expired entry", func(t *testing.T) {
		t.Parallel()
		cfg := DefaultConfig()
		cfg.Cache.TTL = 1 * time.Millisecond
		engine, _ := NewEngine(cfg, testLogger())

		resp := &Response{Items: []ScoredItem{{Item: Item{ID: 1}}}}
		engine.storeCache("test-key", resp)

		time.Sleep(10 * time.Millisecond)

		if got := engine.checkCache("test-key"); got != nil {
			t.Error("checkCache() should return nil for expired entry")
		}
	})

	t.Run("storeCache and checkCache roundtrip", func(t *testing.T) {
		t.Parallel()
		engine, _ := NewEngine(nil, testLogger())

		resp := &Response{Items: []ScoredItem{{Item: Item{ID: 1}, Score: 0.9}}}
		engine.storeCache("test-key", resp)

		got := engine.checkCache("test-key")
		if got == nil {
			t.Fatal("checkCache() = nil, want cached response")
		}
		if len(got.Items) != 1 || got.Items[0].Item.ID != 1 {
			t.Error("checkCache() returned wrong data")
		}
	})

	t.Run("clearCache removes all entries", func(t *testing.T) {
		t.Parallel()
		engine, _ := NewEngine(nil, testLogger())

		engine.storeCache("key1", &Response{})
		engine.storeCache("key2", &Response{})

		engine.clearCache()

		if len(engine.cache) != 0 {
			t.Errorf("clearCache() left %d entries, want 0", len(engine.cache))
		}
	})

	t.Run("storeCache evicts expired on full cache", func(t *testing.T) {
		t.Parallel()
		cfg := DefaultConfig()
		cfg.Cache.MaxEntries = 2
		cfg.Cache.TTL = 1 * time.Millisecond
		engine, _ := NewEngine(cfg, testLogger())

		// Fill cache with soon-to-expire entries
		engine.storeCache("key1", &Response{})
		engine.storeCache("key2", &Response{})

		// Wait for entries to expire
		time.Sleep(10 * time.Millisecond)

		// This should trigger eviction and still store successfully
		cfg.Cache.TTL = 5 * time.Minute
		engine.config = cfg
		engine.storeCache("key3", &Response{})

		if engine.checkCache("key3") == nil {
			t.Error("storeCache() should store new entry after eviction")
		}
	})
}

// --- Test: Concurrent Access ---

func TestEngine_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine(nil, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	alg := newMockAlgorithm("ease")
	alg.trained = true
	alg.predictScores = map[int]float64{1: 0.9, 2: 0.8, 3: 0.7}
	engine.RegisterAlgorithm(alg)
	engine.SetDataProvider(&mockDataProvider{
		candidates: map[int][]int{1: {1, 2, 3, 4, 5}},
	})

	const goroutines = 10
	const requestsPerGoroutine = 20
	var wg sync.WaitGroup
	errChan := make(chan error, goroutines*requestsPerGoroutine)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				_, err := engine.Recommend(context.Background(), Request{UserID: 1, K: 3})
				if err != nil {
					errChan <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("Concurrent Recommend() error: %v", err)
	}

	metrics := engine.GetMetrics()
	expectedRequests := int64(goroutines * requestsPerGoroutine)
	if metrics.RequestCount != expectedRequests {
		t.Errorf("RequestCount = %d, want %d", metrics.RequestCount, expectedRequests)
	}
}

func TestEngine_ConcurrentTrainAndRecommend(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.Training.MinInteractions = 1
	cfg.Cache.Enabled = false // Disable cache to test actual scoring
	engine, err := NewEngine(cfg, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	alg := newMockAlgorithm("ease")
	alg.trained = true
	alg.predictScores = map[int]float64{1: 0.9, 2: 0.8}
	engine.RegisterAlgorithm(alg)

	engine.SetDataProvider(&mockDataProvider{
		candidates:   map[int][]int{1: {1, 2, 3}},
		interactions: []Interaction{{UserID: 1, ItemID: 1}},
		items:        []Item{{ID: 1}},
	})

	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Concurrent recommendations
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, _ = engine.Recommend(context.Background(), Request{UserID: 1, K: 3})
			}
		}
	}()

	// Concurrent training attempts
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Training may fail with "already in progress" which is expected
				_ = engine.Train(context.Background())
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	wg.Wait()
	// If we get here without panic/deadlock, the test passes
}

// --- Test: generateRequestID ---

func TestEngine_generateRequestID(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine(nil, testLogger())
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	ids := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		id := engine.generateRequestID()
		if id == "" {
			t.Error("generateRequestID() returned empty string")
		}
		if _, exists := ids[id]; exists {
			t.Errorf("generateRequestID() produced duplicate: %s", id)
		}
		ids[id] = struct{}{}
	}
}

// --- Test: countUniqueUsers ---

func TestCountUniqueUsers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		interactions []Interaction
		want         int
	}{
		{
			name:         "empty interactions",
			interactions: []Interaction{},
			want:         0,
		},
		{
			name: "single user",
			interactions: []Interaction{
				{UserID: 1, ItemID: 1},
				{UserID: 1, ItemID: 2},
			},
			want: 1,
		},
		{
			name: "multiple unique users",
			interactions: []Interaction{
				{UserID: 1, ItemID: 1},
				{UserID: 2, ItemID: 2},
				{UserID: 3, ItemID: 3},
			},
			want: 3,
		},
		{
			name: "duplicate users",
			interactions: []Interaction{
				{UserID: 1, ItemID: 1},
				{UserID: 2, ItemID: 2},
				{UserID: 1, ItemID: 3},
				{UserID: 2, ItemID: 4},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := countUniqueUsers(tt.interactions)
			if got != tt.want {
				t.Errorf("countUniqueUsers() = %d, want %d", got, tt.want)
			}
		})
	}
}

// --- Test: cacheKey ---

func TestEngine_cacheKey(t *testing.T) {
	t.Parallel()

	engine, _ := NewEngine(nil, testLogger())

	tests := []struct {
		name string
		req  Request
	}{
		{
			name: "basic request",
			req:  Request{UserID: 1, K: 10, Mode: ModePersonalized},
		},
		{
			name: "different mode",
			req:  Request{UserID: 1, K: 10, Mode: ModeSimilar},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			key := engine.cacheKey(tt.req)
			if key == "" {
				t.Error("cacheKey() returned empty string")
			}
		})
	}

	// Verify different requests produce different keys
	key1 := engine.cacheKey(Request{UserID: 1, K: 10, Mode: ModePersonalized})
	key2 := engine.cacheKey(Request{UserID: 2, K: 10, Mode: ModePersonalized})
	key3 := engine.cacheKey(Request{UserID: 1, K: 20, Mode: ModePersonalized})
	key4 := engine.cacheKey(Request{UserID: 1, K: 10, Mode: ModeSimilar})

	if key1 == key2 || key1 == key3 || key1 == key4 {
		t.Error("Different requests should produce different cache keys")
	}
}

// --- Helper functions ---

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsString(s[1:], substr) || s[:len(substr)] == substr)
}
